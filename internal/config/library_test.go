package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLibraryProperties(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		validate    func(*testing.T, *LibraryConfig)
	}{
		{
			name: "basic library",
			content: `name=My Library
version=1.0.0
include=src
exclude=node_modules,tests`,
			expectError: false,
			validate: func(t *testing.T, cfg *LibraryConfig) {
				if cfg.Name != "My Library" {
					t.Errorf("Name = %q, want %q", cfg.Name, "My Library")
				}
				if cfg.Version != "1.0.0" {
					t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
				}
				if len(cfg.Include) != 1 {
					t.Errorf("Include count = %d, want 1", len(cfg.Include))
				}
				if len(cfg.Exclude) != 2 {
					t.Errorf("Exclude count = %d, want 2", len(cfg.Exclude))
				}
			},
		},
		{
			name: "with slug",
			content: `name=My Library
slug=my-lib`,
			expectError: false,
			validate: func(t *testing.T, cfg *LibraryConfig) {
				if cfg.Slug != "my-lib" {
					t.Errorf("Slug = %q, want %q", cfg.Slug, "my-lib")
				}
			},
		},
		{
			name: "with multiple includes",
			content: `name=My Library
include=src,lib,vendor`,
			expectError: false,
			validate: func(t *testing.T, cfg *LibraryConfig) {
				if len(cfg.Include) != 3 {
					t.Errorf("Include count = %d, want 3", len(cfg.Include))
				}
			},
		},
		{
			name:        "missing name",
			content:     `version=1.0.0`,
			expectError: true,
			validate:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "library_test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			propsPath := filepath.Join(tmpDir, "library.properties")
			err = os.WriteFile(propsPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadLibraryProperties(tmpDir)

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

func TestLibraryExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "library_exists_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if LibraryExists(tmpDir) {
		t.Error("LibraryExists should return false when file doesn't exist")
	}

	propsPath := filepath.Join(tmpDir, "library.properties")
	err = os.WriteFile(propsPath, []byte("name=Test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	if !LibraryExists(tmpDir) {
		t.Error("LibraryExists should return true when file exists")
	}
}

func TestLoadLibraryPropertiesFileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "library_notfound_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = LoadLibraryProperties(tmpDir)
	if err == nil {
		t.Error("Expected error when library.properties doesn't exist")
	}
}
