package config

import (
	"testing"
)

func TestParseLibraryString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantName    string
		wantURL     string
		wantVersion string
	}{
		{
			name:        "simple URL",
			input:       "https://example.com/mylib.zip",
			wantName:    "mylib",
			wantURL:     "https://example.com/mylib.zip",
			wantVersion: "",
		},
		{
			name:        "URL with version",
			input:       "https://example.com/mylib.zip:1.0.0",
			wantName:    "mylib",
			wantURL:     "https://example.com/mylib.zip",
			wantVersion: "1.0.0",
		},
		{
			name:        "GitHub URL",
			input:       "https://github.com/owner/myrepo",
			wantName:    "myrepo",
			wantURL:     "https://github.com/owner/myrepo",
			wantVersion: "",
		},
		{
			name:        "GitHub URL with version",
			input:       "https://github.com/owner/myrepo:v2.0.0",
			wantName:    "myrepo",
			wantURL:     "https://github.com/owner/myrepo",
			wantVersion: "v2.0.0",
		},
		{
			name:        "local path",
			input:       "./vendor/mylib.zip",
			wantName:    "mylib",
			wantURL:     "./vendor/mylib.zip",
			wantVersion: "",
		},
		{
			name:        "absolute path",
			input:       "/path/to/lib.zip",
			wantName:    "lib",
			wantURL:     "/path/to/lib.zip",
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := parseLibraryString(tt.input)
			if spec == nil {
				t.Fatalf("parseLibraryString(%q) returned nil", tt.input)
			}
			if spec.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", spec.Name, tt.wantName)
			}
			if spec.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", spec.URL, tt.wantURL)
			}
			if spec.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", spec.Version, tt.wantVersion)
			}
		})
	}
}

func TestParseLibraryMap(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]interface{}
		wantName    string
		wantURL     string
		wantVersion string
		wantNil     bool
	}{
		{
			name: "full spec",
			input: map[string]interface{}{
				"name":    "mylib",
				"url":     "https://example.com/lib.zip",
				"version": "1.0.0",
			},
			wantName:    "mylib",
			wantURL:     "https://example.com/lib.zip",
			wantVersion: "1.0.0",
		},
		{
			name: "url only - name derived",
			input: map[string]interface{}{
				"url": "https://github.com/owner/myrepo",
			},
			wantName:    "myrepo",
			wantURL:     "https://github.com/owner/myrepo",
			wantVersion: "",
		},
		{
			name: "name and url",
			input: map[string]interface{}{
				"name": "custom-name",
				"url":  "https://example.com/lib.zip",
			},
			wantName:    "custom-name",
			wantURL:     "https://example.com/lib.zip",
			wantVersion: "",
		},
		{
			name:    "missing url",
			input:   map[string]interface{}{"name": "mylib"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := parseLibraryMap(tt.input)
			if tt.wantNil {
				if spec != nil {
					t.Errorf("expected nil, got %+v", spec)
				}
				return
			}
			if spec == nil {
				t.Fatalf("parseLibraryMap returned nil")
			}
			if spec.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", spec.Name, tt.wantName)
			}
			if spec.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", spec.URL, tt.wantURL)
			}
			if spec.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", spec.Version, tt.wantVersion)
			}
		})
	}
}

func TestDeriveLibraryName(t *testing.T) {
	tests := []struct {
		url      string
		wantName string
	}{
		{"https://github.com/owner/myrepo", "myrepo"},
		{"https://github.com/owner/myrepo.git", "myrepo"},
		{"https://github.com/owner/myrepo/releases", "myrepo"},
		{"https://example.com/path/to/library.zip", "library"},
		{"https://example.com/download?file=lib.zip", "download"}, // query params stripped from filename
		{"./vendor/mylib.zip", "mylib"},
		{"/absolute/path/to/lib.zip", "lib"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := deriveLibraryName(tt.url)
			if got != tt.wantName {
				t.Errorf("deriveLibraryName(%q) = %q, want %q", tt.url, got, tt.wantName)
			}
		})
	}
}

func TestFindVersionSeparator(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"https://example.com/lib.zip:1.0.0", 27},
		{"https://github.com/owner/repo:v2.0", 29},
		{"https://example.com/lib.zip", -1},
		{"./local/path.zip", -1},
		{"./local/path.zip:1.0", 16},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := findVersionSeparator(tt.input)
			if got != tt.want {
				t.Errorf("findVersionSeparator(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLibraries(t *testing.T) {
	// Test with YAML list
	props := Properties{
		"libraries": []interface{}{
			"https://github.com/owner/repo1",
			map[string]interface{}{
				"name":    "mylib",
				"url":     "https://example.com/lib.zip",
				"version": "1.0.0",
			},
			"https://github.com/owner/repo2:v2.0.0",
		},
	}

	specs := ParseLibraries(props)
	if len(specs) != 3 {
		t.Fatalf("expected 3 specs, got %d", len(specs))
	}

	// First: simple GitHub URL
	if specs[0].Name != "repo1" {
		t.Errorf("specs[0].Name = %q, want %q", specs[0].Name, "repo1")
	}
	if specs[0].URL != "https://github.com/owner/repo1" {
		t.Errorf("specs[0].URL = %q, want %q", specs[0].URL, "https://github.com/owner/repo1")
	}

	// Second: full map
	if specs[1].Name != "mylib" {
		t.Errorf("specs[1].Name = %q, want %q", specs[1].Name, "mylib")
	}
	if specs[1].Version != "1.0.0" {
		t.Errorf("specs[1].Version = %q, want %q", specs[1].Version, "1.0.0")
	}

	// Third: URL with version
	if specs[2].Name != "repo2" {
		t.Errorf("specs[2].Name = %q, want %q", specs[2].Name, "repo2")
	}
	if specs[2].Version != "v2.0.0" {
		t.Errorf("specs[2].Version = %q, want %q", specs[2].Version, "v2.0.0")
	}
}

func TestParseLibrariesCommaSeparated(t *testing.T) {
	props := Properties{
		"libraries": "https://github.com/owner/repo1, https://example.com/lib.zip:1.0.0",
	}

	specs := ParseLibraries(props)
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}

	if specs[0].Name != "repo1" {
		t.Errorf("specs[0].Name = %q, want %q", specs[0].Name, "repo1")
	}

	if specs[1].Name != "lib" {
		t.Errorf("specs[1].Name = %q, want %q", specs[1].Name, "lib")
	}
	if specs[1].Version != "1.0.0" {
		t.Errorf("specs[1].Version = %q, want %q", specs[1].Version, "1.0.0")
	}
}
