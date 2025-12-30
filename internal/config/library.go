package config

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	libraryBaseDir = ".wordsmith/libraries"
)

// LibrarySpec represents a library specification from properties file
type LibrarySpec struct {
	Name    string // Directory name to use in the build
	URL     string // URL to download from (can be zip URL or GitHub repo URL)
	Version string // Version to download (for GitHub repos)
}

// ParseLibraries parses the libraries property from a properties file.
// Supports multiple formats:
//   - Simple list: libraries: [url1, url2]
//   - YAML list with properties:
//     libraries:
//   - name: mylib
//     url: https://github.com/owner/repo
//     version: 1.0.0
//   - Shortcut format: url:version (e.g., https://github.com/owner/repo:1.0.0)
func ParseLibraries(props Properties) []LibrarySpec {
	val, ok := props["libraries"]
	if !ok {
		return nil
	}

	var specs []LibrarySpec

	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			spec := parseLibraryItem(item)
			if spec != nil {
				specs = append(specs, *spec)
			}
		}
	case string:
		// Comma-separated list
		if v == "" {
			return nil
		}
		items := strings.Split(v, ",")
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			spec := parseLibraryString(item)
			if spec != nil {
				specs = append(specs, *spec)
			}
		}
	}

	return specs
}

// parseLibraryItem parses a single library item which can be a string or a map
func parseLibraryItem(item interface{}) *LibrarySpec {
	switch v := item.(type) {
	case string:
		return parseLibraryString(v)
	case Properties:
		return parseLibraryProperties(v)
	case map[string]interface{}:
		return parseLibraryMap(v)
	}
	return nil
}

// parseLibraryString parses a library string in format: url or url:version
func parseLibraryString(s string) *LibrarySpec {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	spec := &LibrarySpec{}

	// Check for version suffix (url:version)
	// Be careful not to split on :// in URLs
	if idx := findVersionSeparator(s); idx != -1 {
		spec.URL = s[:idx]
		spec.Version = s[idx+1:]
	} else {
		spec.URL = s
	}

	// Derive name from URL
	spec.Name = deriveLibraryName(spec.URL)

	return spec
}

// findVersionSeparator finds the index of the version separator (:) in a library string.
// Returns -1 if no version separator is found.
// Skips :// in URLs.
func findVersionSeparator(s string) int {
	// Skip past the protocol if present
	start := 0
	if idx := strings.Index(s, "://"); idx != -1 {
		start = idx + 3
	}

	// Find the last colon after the protocol
	lastColon := strings.LastIndex(s[start:], ":")
	if lastColon == -1 {
		return -1
	}

	return start + lastColon
}

// parseLibraryProperties parses a library from Properties type (from YAML parser)
func parseLibraryProperties(p Properties) *LibrarySpec {
	spec := &LibrarySpec{}

	if name, ok := p["name"].(string); ok {
		spec.Name = name
	}
	if url, ok := p["url"].(string); ok {
		spec.URL = url
	}
	if version, ok := p["version"].(string); ok {
		spec.Version = version
	}

	// If no name specified, derive from URL
	if spec.Name == "" && spec.URL != "" {
		spec.Name = deriveLibraryName(spec.URL)
	}

	// If URL is empty, spec is invalid
	if spec.URL == "" {
		return nil
	}

	return spec
}

// parseLibraryMap parses a library map with name, url, version properties
func parseLibraryMap(m map[string]interface{}) *LibrarySpec {
	spec := &LibrarySpec{}

	if name, ok := m["name"].(string); ok {
		spec.Name = name
	}
	if url, ok := m["url"].(string); ok {
		spec.URL = url
	}
	if version, ok := m["version"].(string); ok {
		spec.Version = version
	}

	// If no name specified, derive from URL
	if spec.Name == "" && spec.URL != "" {
		spec.Name = deriveLibraryName(spec.URL)
	}

	// If URL is empty, spec is invalid
	if spec.URL == "" {
		return nil
	}

	return spec
}

// deriveLibraryName derives a library directory name from a URL.
// For GitHub URLs: uses the repo name
// For zip URLs: uses the filename without extension
// For file paths: uses the filename without extension
func deriveLibraryName(url string) string {
	// Handle GitHub URLs
	if strings.Contains(url, "github.com") {
		// Extract repo name from GitHub URL
		// Pattern: github.com/owner/repo or github.com/owner/repo/...
		re := regexp.MustCompile(`github\.com/[^/]+/([^/]+)`)
		if matches := re.FindStringSubmatch(url); len(matches) > 1 {
			name := matches[1]
			// Remove .git suffix if present
			name = strings.TrimSuffix(name, ".git")
			return name
		}
	}

	// For other URLs or file paths, use the filename
	name := filepath.Base(url)

	// Remove query string if present
	if idx := strings.Index(name, "?"); idx != -1 {
		name = name[:idx]
	}

	// Remove .zip extension
	name = strings.TrimSuffix(name, ".zip")

	// If name is empty or just an extension, generate a fallback
	if name == "" || name == "." {
		return "library"
	}

	return name
}

// ResolveLibrary resolves a library spec to a local path.
// Downloads the library if necessary and caches it.
// Returns the path to the library directory.
func ResolveLibrary(spec LibrarySpec) (string, error) {
	// Determine if this is a local file path
	if IsLocalPath(spec.URL) {
		return resolveLocalLibrary(spec)
	}

	// It's a URL - need to download
	return resolveRemoteLibrary(spec)
}

// IsLocalPath checks if a URL is actually a local file path
func IsLocalPath(url string) bool {
	// Check for URL protocols
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return false
	}
	// Check for absolute or relative paths
	return strings.HasPrefix(url, "/") || strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../") || !strings.Contains(url, "://")
}

// resolveLocalLibrary resolves a local library path
func resolveLocalLibrary(spec LibrarySpec) (string, error) {
	path := spec.URL

	// Check if it exists
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("library not found: %s", path)
	}

	// If it's a zip file, extract to temp directory
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		return extractLocalZip(path)
	}

	// If it's a directory, find the latest zip file
	if info.IsDir() {
		zipPath, err := findLatestZipInDir(path)
		if err != nil {
			return "", fmt.Errorf("no zip file found in directory %s: %w", path, err)
		}
		return extractLocalZip(zipPath)
	}

	return "", fmt.Errorf("library path is neither a zip file nor a directory: %s", path)
}

// extractLocalZip extracts a local zip to a temp directory (no caching)
func extractLocalZip(zipPath string) (string, error) {
	tempDir, err := os.MkdirTemp("", "wordsmith-lib-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	if err := extractZipToDir(zipPath, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to extract library: %w", err)
	}

	return tempDir, nil
}

// findLatestZipInDir finds the most recently modified zip file in a directory
func findLatestZipInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var latestPath string
	var latestTime int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".zip") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime().Unix()
		if latestPath == "" || modTime > latestTime {
			latestPath = filepath.Join(dir, entry.Name())
			latestTime = modTime
		}
	}

	if latestPath == "" {
		return "", fmt.Errorf("no zip files found")
	}

	return latestPath, nil
}

// resolveRemoteLibrary resolves a remote library (URL or GitHub)
func resolveRemoteLibrary(spec LibrarySpec) (string, error) {
	// If version is specified, check cache first
	if spec.Version != "" {
		cacheDir := getLibraryCacheDir(spec.Name, spec.Version)
		if isLibraryCached(cacheDir) {
			return cacheDir, nil
		}
	}

	// For GitHub URLs without a version, resolve the latest version
	if spec.Version == "" && strings.Contains(spec.URL, "github.com") && isGitHubRepoURL(spec.URL) {
		resolvedVersion, downloadURL, err := resolveGitHubLatestVersion(spec.URL, spec.Name)
		if err != nil {
			// GitHub API failed, try to use locally cached version
			cachedVersion := findLatestCachedVersion(spec.Name)
			if cachedVersion != "" {
				cacheDir := getLibraryCacheDir(spec.Name, cachedVersion)
				if isLibraryCached(cacheDir) {
					return cacheDir, nil
				}
			}
			return "", fmt.Errorf("failed to resolve library %s: %w", spec.Name, err)
		}

		// Check if this version is already cached
		cacheDir := getLibraryCacheDir(spec.Name, resolvedVersion)
		if isLibraryCached(cacheDir) {
			return cacheDir, nil
		}

		// Download and extract with the resolved version
		return downloadAndExtractLibrary(downloadURL, spec.Name, resolvedVersion)
	}

	// For non-GitHub URLs without version, we still need a version for caching
	// Use URL hash as a pseudo-version
	if spec.Version == "" {
		spec.Version = hashURL(spec.URL)
	}

	// Check cache
	cacheDir := getLibraryCacheDir(spec.Name, spec.Version)
	if isLibraryCached(cacheDir) {
		return cacheDir, nil
	}

	// Resolve the download URL
	downloadURL, err := resolveDownloadURL(spec)
	if err != nil {
		return "", err
	}

	// Download and extract
	return downloadAndExtractLibrary(downloadURL, spec.Name, spec.Version)
}

// resolveDownloadURL resolves a library spec to a download URL
func resolveDownloadURL(spec LibrarySpec) (string, error) {
	// If it's already a direct zip URL, use it
	if strings.HasSuffix(strings.ToLower(spec.URL), ".zip") {
		return spec.URL, nil
	}

	// If it's a GitHub releases download URL, use it
	if strings.Contains(spec.URL, "/releases/download/") {
		return spec.URL, nil
	}

	// If it's a GitHub repo URL, resolve to release asset
	if strings.Contains(spec.URL, "github.com") {
		return ResolveGitHubURL(spec.URL, spec.Name, spec.Version)
	}

	// Otherwise, assume it's a direct download URL
	return spec.URL, nil
}

// getLibraryCacheDir returns the cache directory for a library
func getLibraryCacheDir(name, version string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	if version == "" {
		return ""
	}

	return filepath.Join(homeDir, libraryBaseDir, name, "v"+strings.TrimPrefix(version, "v"))
}

// findLatestCachedVersion finds the latest cached version of a library
func findLatestCachedVersion(name string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	libDir := filepath.Join(homeDir, libraryBaseDir, name)
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return ""
	}

	var latestVersion string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "v") {
			version := strings.TrimPrefix(entry.Name(), "v")
			if latestVersion == "" || compareVersions(version, latestVersion) > 0 {
				latestVersion = version
			}
		}
	}

	return latestVersion
}

// compareVersions compares two version strings (simple comparison)
func compareVersions(v1, v2 string) int {
	// Split versions into parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &n2)
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	return 0
}

// hashURL creates a short hash of a URL for use as a cache key
func hashURL(url string) string {
	// Simple hash - first 8 chars of a basic checksum
	var sum uint32
	for _, c := range url {
		sum = sum*31 + uint32(c)
	}
	return fmt.Sprintf("%08x", sum)
}

// resolveGitHubLatestVersion resolves the latest version and download URL for a GitHub repo
func resolveGitHubLatestVersion(url, name string) (version string, downloadURL string, err error) {
	owner, repo, err := parseGitHubRepoURL(url)
	if err != nil {
		return "", "", err
	}

	// Fetch latest release
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	release, err := fetchGitHubRelease(apiURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// Extract version from tag
	version = strings.TrimPrefix(release.TagName, "v")

	// Find matching asset
	assetURL := findReleaseAsset(release, name, version)
	if assetURL == "" {
		return "", "", fmt.Errorf("no matching asset found in release %s", release.TagName)
	}

	return version, assetURL, nil
}

// isLibraryCached checks if a library is already cached
func isLibraryCached(cacheDir string) bool {
	if cacheDir == "" {
		return false
	}
	info, err := os.Stat(cacheDir)
	if err != nil || !info.IsDir() {
		return false
	}
	// Check for at least one file
	files, err := os.ReadDir(cacheDir)
	return err == nil && len(files) > 0
}

// downloadAndExtractLibrary downloads a library zip and extracts it to the cache
func downloadAndExtractLibrary(url, name, version string) (string, error) {
	cacheDir := getLibraryCacheDir(name, version)
	if cacheDir == "" {
		return "", fmt.Errorf("could not determine cache directory")
	}

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "wordsmith-library-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Download
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download library: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download library: HTTP %d", resp.StatusCode)
	}

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return "", fmt.Errorf("failed to save library: %w", err)
	}

	// Extract
	if err := extractZipToDir(tmpPath, cacheDir); err != nil {
		os.RemoveAll(cacheDir)
		return "", fmt.Errorf("failed to extract library: %w", err)
	}

	return cacheDir, nil
}

// extractZipToDir extracts a zip file to a directory
func extractZipToDir(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Find the root directory in the zip (if any)
	// Many GitHub releases have a single root directory
	var rootDir string
	for _, f := range r.File {
		parts := strings.Split(f.Name, "/")
		if len(parts) > 1 && rootDir == "" {
			rootDir = parts[0]
		}
		break
	}

	// Check if all files are under the same root
	allUnderRoot := rootDir != ""
	if allUnderRoot {
		for _, f := range r.File {
			if !strings.HasPrefix(f.Name, rootDir+"/") && f.Name != rootDir+"/" {
				allUnderRoot = false
				break
			}
		}
	}

	for _, f := range r.File {
		name := f.Name

		// Strip root directory if all files are under it
		if allUnderRoot && rootDir != "" {
			if name == rootDir+"/" {
				continue
			}
			name = strings.TrimPrefix(name, rootDir+"/")
		}

		if name == "" {
			continue
		}

		fpath := filepath.Join(destDir, name)

		// Security check - prevent path traversal
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// ParsePlugins parses the plugins property from a properties file.
// Supports the same formats as ParseLibraries, plus slug/version format:
//   - Simple string: "woocommerce" or "woocommerce:8.0"
//   - Object with slug: { slug: woocommerce, version: 8.0 }
//   - Object with name/url: { name: my-plugin, url: ../path }
func ParsePlugins(props Properties) []LibrarySpec {
	val, ok := props["plugins"]
	if !ok {
		return nil
	}

	var specs []LibrarySpec

	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			spec := parsePluginDependencyItem(item)
			if spec != nil {
				specs = append(specs, *spec)
			}
		}
	case string:
		// Comma-separated list
		if v == "" {
			return nil
		}
		items := strings.Split(v, ",")
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			spec := parseLibraryString(item)
			if spec != nil {
				specs = append(specs, *spec)
			}
		}
	}

	return specs
}

// parsePluginDependencyItem parses a single plugin dependency item which can be a string or a map
// Supports both library format (name/url) and plugin format (slug/version)
func parsePluginDependencyItem(item interface{}) *LibrarySpec {
	switch v := item.(type) {
	case string:
		return parseLibraryString(v)
	case Properties:
		return parsePluginProperties(v)
	case map[string]interface{}:
		return parsePluginMap(v)
	}
	return nil
}

// parsePluginProperties parses a plugin from Properties type
// Supports both slug/version and name/url formats
func parsePluginProperties(p Properties) *LibrarySpec {
	spec := &LibrarySpec{}

	// Check for slug first (plugin-specific), then fall back to name
	if slug, ok := p["slug"].(string); ok && slug != "" {
		spec.Name = slug
		// For slug-only entries, URL is the same as name (WordPress.org slug)
		spec.URL = slug
	} else if name, ok := p["name"].(string); ok {
		spec.Name = name
	}

	// URL overrides the slug-based URL if provided
	if url, ok := p["url"].(string); ok && url != "" {
		spec.URL = url
	} else if uri, ok := p["uri"].(string); ok && uri != "" {
		spec.URL = uri
	}

	if version, ok := p["version"].(string); ok {
		spec.Version = version
	}

	// If no name specified, derive from URL
	if spec.Name == "" && spec.URL != "" {
		spec.Name = deriveLibraryName(spec.URL)
	}

	// If both name and URL are empty, spec is invalid
	if spec.Name == "" && spec.URL == "" {
		return nil
	}

	return spec
}

// parsePluginMap parses a plugin map with slug/version or name/url properties
func parsePluginMap(m map[string]interface{}) *LibrarySpec {
	spec := &LibrarySpec{}

	// Check for slug first (plugin-specific), then fall back to name
	if slug, ok := m["slug"].(string); ok && slug != "" {
		spec.Name = slug
		// For slug-only entries, URL is the same as name (WordPress.org slug)
		spec.URL = slug
	} else if name, ok := m["name"].(string); ok {
		spec.Name = name
	}

	// URL overrides the slug-based URL if provided
	if url, ok := m["url"].(string); ok && url != "" {
		spec.URL = url
	} else if uri, ok := m["uri"].(string); ok && uri != "" {
		spec.URL = uri
	}

	if version, ok := m["version"].(string); ok {
		spec.Version = version
	}

	// If no name specified, derive from URL
	if spec.Name == "" && spec.URL != "" {
		spec.Name = deriveLibraryName(spec.URL)
	}

	// If both name and URL are empty, spec is invalid
	if spec.Name == "" && spec.URL == "" {
		return nil
	}

	return spec
}

// IsWordPressOrgSlug checks if a library spec refers to a WordPress.org plugin slug
// (as opposed to a local path, URL, or GitHub repo)
func IsWordPressOrgSlug(spec LibrarySpec) bool {
	url := spec.URL
	if url == "" {
		url = spec.Name
	}

	// WordPress.org slug: no protocol, no path separators, no .zip extension
	if strings.Contains(url, "://") {
		return false
	}
	if strings.Contains(url, "/") {
		return false
	}
	if strings.HasSuffix(strings.ToLower(url), ".zip") {
		return false
	}
	// Could be a relative path starting with .
	if strings.HasPrefix(url, ".") {
		return false
	}

	return true
}

// CopyLibraryToDir copies a resolved library to a destination directory
func CopyLibraryToDir(libPath, destDir, libName string) error {
	targetDir := filepath.Join(destDir, libName)

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create library directory: %w", err)
	}

	// Copy all files from library path to target
	return filepath.Walk(libPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(libPath, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(targetPath, content, info.Mode())
	})
}
