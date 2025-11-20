package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadThemeProperties(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		validate    func(*testing.T, *ThemeConfig)
	}{
		{
			name: "basic theme",
			content: `name=My Theme
version=1.0.0
description=A test theme
author=Test Author`,
			expectError: false,
			validate: func(t *testing.T, cfg *ThemeConfig) {
				if cfg.Name != "My Theme" {
					t.Errorf("Name = %q, want %q", cfg.Name, "My Theme")
				}
				if cfg.Main != "style.css" {
					t.Errorf("Main = %q, want %q (default)", cfg.Main, "style.css")
				}
				if cfg.Version != "1.0.0" {
					t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
				}
			},
		},
		{
			name: "child theme with template",
			content: `name=Child Theme
template=parent-theme
template-uri=../parent-theme`,
			expectError: false,
			validate: func(t *testing.T, cfg *ThemeConfig) {
				if cfg.Template != "parent-theme" {
					t.Errorf("Template = %q, want %q", cfg.Template, "parent-theme")
				}
				if cfg.TemplateURI != "../parent-theme" {
					t.Errorf("TemplateURI = %q, want %q", cfg.TemplateURI, "../parent-theme")
				}
			},
		},
		{
			name: "with includes and excludes",
			content: `name=My Theme
include=*.php, assets, templates
exclude=node_modules, *.md`,
			expectError: false,
			validate: func(t *testing.T, cfg *ThemeConfig) {
				if len(cfg.Include) != 3 {
					t.Errorf("Include count = %d, want 3", len(cfg.Include))
				}
				if len(cfg.Exclude) != 2 {
					t.Errorf("Exclude count = %d, want 2", len(cfg.Exclude))
				}
			},
		},
		{
			name: "with minify option",
			content: `name=My Theme
minify=true`,
			expectError: false,
			validate: func(t *testing.T, cfg *ThemeConfig) {
				if !cfg.Minify {
					t.Error("Minify should be true")
				}
			},
		},
		{
			name: "minify false",
			content: `name=My Theme
minify=false`,
			expectError: false,
			validate: func(t *testing.T, cfg *ThemeConfig) {
				if cfg.Minify {
					t.Error("Minify should be false")
				}
			},
		},
		{
			name: "with comments and empty lines",
			content: `# Theme configuration
name=My Theme

# This is the version
version=1.0.0`,
			expectError: false,
			validate: func(t *testing.T, cfg *ThemeConfig) {
				if cfg.Name != "My Theme" {
					t.Errorf("Name = %q, want %q", cfg.Name, "My Theme")
				}
				if cfg.Version != "1.0.0" {
					t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
				}
			},
		},
		{
			name:        "missing name",
			content:     `version=1.0.0`,
			expectError: true,
			validate:    nil,
		},
		{
			name: "all fields",
			content: `name=Full Theme
version=2.0.0
description=Full description
author=Jane Doe
author-uri=https://example.com
theme-uri=https://theme.example.com
license=GPL-2.0
license-uri=https://www.gnu.org/licenses/gpl-2.0.html
main=custom-style.css
template=parent
template-uri=https://parent.example.com
text-domain=full-theme
domain-path=/languages
requires=5.0
requires-php=7.4
tags=responsive, modern`,
			expectError: false,
			validate: func(t *testing.T, cfg *ThemeConfig) {
				if cfg.AuthorURI != "https://example.com" {
					t.Errorf("AuthorURI = %q, want %q", cfg.AuthorURI, "https://example.com")
				}
				if cfg.ThemeURI != "https://theme.example.com" {
					t.Errorf("ThemeURI = %q, want %q", cfg.ThemeURI, "https://theme.example.com")
				}
				if cfg.Main != "custom-style.css" {
					t.Errorf("Main = %q, want %q", cfg.Main, "custom-style.css")
				}
				if cfg.Template != "parent" {
					t.Errorf("Template = %q, want %q", cfg.Template, "parent")
				}
				if cfg.Tags != "responsive, modern" {
					t.Errorf("Tags = %q, want %q", cfg.Tags, "responsive, modern")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "theme_test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Write theme.properties
			propsPath := filepath.Join(tmpDir, "theme.properties")
			err = os.WriteFile(propsPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Load properties
			cfg, err := LoadThemeProperties(tmpDir)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestThemeExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "theme_exists_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test when file doesn't exist
	if ThemeExists(tmpDir) {
		t.Error("ThemeExists should return false when file doesn't exist")
	}

	// Create theme.properties
	propsPath := filepath.Join(tmpDir, "theme.properties")
	err = os.WriteFile(propsPath, []byte("name=Test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test when file exists
	if !ThemeExists(tmpDir) {
		t.Error("ThemeExists should return true when file exists")
	}
}

func TestLoadThemePropertiesFileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "theme_notfound_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = LoadThemeProperties(tmpDir)
	if err == nil {
		t.Error("Expected error when theme.properties doesn't exist")
	}
}
