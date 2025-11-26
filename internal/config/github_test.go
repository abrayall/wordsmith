package config

import (
	"testing"
)

func TestIsGitHubRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		{
			name:     "GitHub repo URL",
			uri:      "https://github.com/abrayall/wordsmith",
			expected: true,
		},
		{
			name:     "GitHub repo URL with trailing slash",
			uri:      "https://github.com/abrayall/wordsmith/",
			expected: true,
		},
		{
			name:     "GitHub repo URL with /releases",
			uri:      "https://github.com/abrayall/wordsmith/releases",
			expected: true,
		},
		{
			name:     "GitHub repo URL with /releases/",
			uri:      "https://github.com/abrayall/wordsmith/releases/",
			expected: true,
		},
		{
			name:     "GitHub release download URL",
			uri:      "https://github.com/abrayall/wordsmith/releases/download/v1.0.0/plugin.zip",
			expected: false,
		},
		{
			name:     "GitHub raw content URL",
			uri:      "https://raw.githubusercontent.com/abrayall/wordsmith/main/file.txt",
			expected: false,
		},
		{
			name:     "Direct zip URL",
			uri:      "https://example.com/plugin.zip",
			expected: false,
		},
		{
			name:     "Non-GitHub URL",
			uri:      "https://example.com/repo",
			expected: false,
		},
		{
			name:     "GitHub repo with subpath",
			uri:      "https://github.com/abrayall/wordsmith/tree/main",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGitHubRepoURL(tt.uri)
			if result != tt.expected {
				t.Errorf("isGitHubRepoURL(%q) = %v, want %v", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestParseGitHubRepoURL(t *testing.T) {
	tests := []struct {
		name          string
		uri           string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "Valid GitHub URL",
			uri:           "https://github.com/abrayall/wordsmith",
			expectedOwner: "abrayall",
			expectedRepo:  "wordsmith",
			expectError:   false,
		},
		{
			name:          "Valid GitHub URL with trailing slash",
			uri:           "https://github.com/owner/repo/",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectError:   false,
		},
		{
			name:          "Valid GitHub URL with /releases",
			uri:           "https://github.com/owner/repo/releases",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectError:   false,
		},
		{
			name:          "Valid GitHub URL with /releases/",
			uri:           "https://github.com/owner/repo/releases/",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectError:   false,
		},
		{
			name:        "Invalid URL",
			uri:         "https://example.com/something",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubRepoURL(tt.uri)
			if tt.expectError {
				if err == nil {
					t.Errorf("parseGitHubRepoURL(%q) expected error, got nil", tt.uri)
				}
				return
			}
			if err != nil {
				t.Errorf("parseGitHubRepoURL(%q) unexpected error: %v", tt.uri, err)
				return
			}
			if owner != tt.expectedOwner {
				t.Errorf("parseGitHubRepoURL(%q) owner = %q, want %q", tt.uri, owner, tt.expectedOwner)
			}
			if repo != tt.expectedRepo {
				t.Errorf("parseGitHubRepoURL(%q) repo = %q, want %q", tt.uri, repo, tt.expectedRepo)
			}
		})
	}
}

func TestFindReleaseAsset(t *testing.T) {
	release := &GitHubRelease{
		TagName: "v1.0.0",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "my-plugin-1.0.0.zip", BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1.0.0/my-plugin-1.0.0.zip"},
			{Name: "source.tar.gz", BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1.0.0/source.tar.gz"},
		},
	}

	tests := []struct {
		name        string
		slug        string
		version     string
		expectedURL string
	}{
		{
			name:        "Exact match slug-version.zip",
			slug:        "my-plugin",
			version:     "1.0.0",
			expectedURL: "https://github.com/owner/repo/releases/download/v1.0.0/my-plugin-1.0.0.zip",
		},
		{
			name:        "No match falls back to any zip",
			slug:        "other-plugin",
			version:     "2.0.0",
			expectedURL: "https://github.com/owner/repo/releases/download/v1.0.0/my-plugin-1.0.0.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findReleaseAsset(release, tt.slug, tt.version)
			if result != tt.expectedURL {
				t.Errorf("findReleaseAsset() = %q, want %q", result, tt.expectedURL)
			}
		})
	}
}

func TestResolveGitHubURL_NonGitHubURL(t *testing.T) {
	// Non-GitHub URLs should be returned unchanged
	uri := "https://example.com/plugin.zip"
	result, err := ResolveGitHubURL(uri, "plugin", "1.0.0")
	if err != nil {
		t.Errorf("ResolveGitHubURL() unexpected error: %v", err)
	}
	if result != uri {
		t.Errorf("ResolveGitHubURL() = %q, want %q", result, uri)
	}
}

func TestResolveGitHubURL_AlreadyReleaseURL(t *testing.T) {
	// Already a release download URL should be returned unchanged
	uri := "https://github.com/owner/repo/releases/download/v1.0.0/plugin.zip"
	result, err := ResolveGitHubURL(uri, "plugin", "1.0.0")
	if err != nil {
		t.Errorf("ResolveGitHubURL() unexpected error: %v", err)
	}
	if result != uri {
		t.Errorf("ResolveGitHubURL() = %q, want %q", result, uri)
	}
}
