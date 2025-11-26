package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// ResolveGitHubURL checks if a URL is a GitHub repo URL and resolves it to a release asset URL.
// Returns the resolved URL and any error encountered.
// If the URL is not a GitHub repo URL, returns the original URL unchanged.
func ResolveGitHubURL(uri string, slug string, version string) (string, error) {
	// Check if this is a GitHub repo URL (not already a release/raw URL)
	if !isGitHubRepoURL(uri) {
		return uri, nil
	}

	// Extract owner/repo from URL
	owner, repo, err := parseGitHubRepoURL(uri)
	if err != nil {
		return "", err
	}

	// If version specified, get that specific release
	if version != "" {
		return getGitHubReleaseAsset(owner, repo, slug, version)
	}

	// Otherwise get latest release
	return getGitHubLatestReleaseAsset(owner, repo, slug)
}

// isGitHubRepoURL checks if URL is a GitHub repository URL (not a raw/release download URL)
func isGitHubRepoURL(uri string) bool {
	if !strings.Contains(uri, "github.com") {
		return false
	}

	// Skip if already a release download URL
	if strings.Contains(uri, "/releases/download/") {
		return false
	}

	// Skip if raw content URL
	if strings.Contains(uri, "raw.githubusercontent.com") {
		return false
	}

	// Skip if already ends with .zip
	if strings.HasSuffix(uri, ".zip") {
		return false
	}

	// Match pattern: github.com/owner/repo or github.com/owner/repo/releases
	pattern := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)(/releases)?/?$`)
	return pattern.MatchString(uri)
}

// parseGitHubRepoURL extracts owner and repo from a GitHub URL
func parseGitHubRepoURL(uri string) (string, string, error) {
	// Match github.com/owner/repo or github.com/owner/repo/releases
	pattern := regexp.MustCompile(`github\.com/([^/]+)/([^/]+?)(/releases)?/?$`)
	matches := pattern.FindStringSubmatch(uri)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("invalid GitHub repository URL: %s", uri)
	}
	return matches[1], matches[2], nil
}

// getGitHubReleaseAsset gets the download URL for a specific release version
func getGitHubReleaseAsset(owner, repo, slug, version string) (string, error) {
	// Try with 'v' prefix first, then without
	tags := []string{"v" + version, version}

	for _, tag := range tags {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, tag)

		release, err := fetchGitHubRelease(url)
		if err != nil {
			continue
		}

		// Look for matching asset
		assetURL := findReleaseAsset(release, slug, version)
		if assetURL != "" {
			return assetURL, nil
		}
	}

	return "", fmt.Errorf("no release found for version %s in %s/%s", version, owner, repo)
}

// getGitHubLatestReleaseAsset gets the download URL for the latest release
func getGitHubLatestReleaseAsset(owner, repo, slug string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	release, err := fetchGitHubRelease(url)
	if err != nil {
		return "", fmt.Errorf("no releases found for %s/%s: %w", owner, repo, err)
	}

	// Extract version from tag (remove 'v' prefix if present)
	version := strings.TrimPrefix(release.TagName, "v")

	// Look for matching asset
	assetURL := findReleaseAsset(release, slug, version)
	if assetURL != "" {
		return assetURL, nil
	}

	return "", fmt.Errorf("no matching asset found in latest release of %s/%s", owner, repo)
}

// fetchGitHubRelease fetches release info from GitHub API
func fetchGitHubRelease(url string) (*GitHubRelease, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "wordsmith")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// findReleaseAsset finds a matching zip asset in a release
func findReleaseAsset(release *GitHubRelease, slug, version string) string {
	// Try common naming patterns
	patterns := []string{
		fmt.Sprintf("%s-%s.zip", slug, version),      // slug-version.zip
		fmt.Sprintf("%s.zip", slug),                  // slug.zip
		fmt.Sprintf("%s-v%s.zip", slug, version),     // slug-vversion.zip
		"plugin.zip",                                 // plugin.zip
		"theme.zip",                                  // theme.zip
	}

	for _, asset := range release.Assets {
		// Check exact matches first
		for _, pattern := range patterns {
			if asset.Name == pattern {
				return asset.BrowserDownloadURL
			}
		}
	}

	// Fallback: find any .zip file
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".zip") {
			return asset.BrowserDownloadURL
		}
	}

	return ""
}
