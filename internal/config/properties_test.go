package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseProperties(t *testing.T) {
	// Create temp file
	tmpDir, err := os.MkdirTemp("", "props_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `# Comment
name=Test Name
version=1.0.0
description=Test description
include=src, *.php, assets
minify=true
empty=
`
	propsPath := filepath.Join(tmpDir, "test.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	props, err := ParseProperties(propsPath)
	if err != nil {
		t.Fatalf("ParseProperties error: %v", err)
	}

	// Test Get
	if props.Get("name") != "Test Name" {
		t.Errorf("Get(name) = %q, want %q", props.Get("name"), "Test Name")
	}

	// Test missing key
	if props.Get("missing") != "" {
		t.Errorf("Get(missing) = %q, want empty string", props.Get("missing"))
	}

	// Test empty value
	if props.Get("empty") != "" {
		t.Errorf("Get(empty) = %q, want empty string", props.Get("empty"))
	}
}

func TestParsePropertiesYAMLStyle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yaml_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `name: Test Name
version: 1.0.0
description: Test description
`
	propsPath := filepath.Join(tmpDir, "test.yaml")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	props, err := ParseProperties(propsPath)
	if err != nil {
		t.Fatalf("ParseProperties error: %v", err)
	}

	if props.Get("name") != "Test Name" {
		t.Errorf("Get(name) = %q, want %q", props.Get("name"), "Test Name")
	}
	if props.Get("version") != "1.0.0" {
		t.Errorf("Get(version) = %q, want %q", props.Get("version"), "1.0.0")
	}
}

func TestLoadPluginPropertiesYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_yaml_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `# Plugin configuration in YAML style
name: My YAML Plugin
version: 2.0.0
description: A plugin using YAML syntax
author: YAML Author
main: yaml-plugin.php
include: src, *.php, assets
exclude: tests, node_modules
minify: true
obfuscate: false
`
	propsPath := filepath.Join(tmpDir, "plugin.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPluginProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadPluginProperties error: %v", err)
	}

	if cfg.Name != "My YAML Plugin" {
		t.Errorf("Name = %q, want %q", cfg.Name, "My YAML Plugin")
	}
	if cfg.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "2.0.0")
	}
	if cfg.Main != "yaml-plugin.php" {
		t.Errorf("Main = %q, want %q", cfg.Main, "yaml-plugin.php")
	}
	if len(cfg.Include) != 3 {
		t.Errorf("Include count = %d, want 3", len(cfg.Include))
	}
	if len(cfg.Exclude) != 2 {
		t.Errorf("Exclude count = %d, want 2", len(cfg.Exclude))
	}
	if !cfg.Minify {
		t.Error("Minify should be true")
	}
	if cfg.Obfuscate {
		t.Error("Obfuscate should be false")
	}
}

func TestLoadThemePropertiesYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "theme_yaml_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `# Theme configuration in YAML style
name: My YAML Theme
version: 1.5.0
description: A theme using YAML syntax
author: YAML Theme Author
template: parent-theme
template-uri: ../parent
include: *.php, assets, templates
minify: yes
tags: modern, responsive
`
	propsPath := filepath.Join(tmpDir, "theme.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadThemeProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadThemeProperties error: %v", err)
	}

	if cfg.Name != "My YAML Theme" {
		t.Errorf("Name = %q, want %q", cfg.Name, "My YAML Theme")
	}
	if cfg.Version != "1.5.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.5.0")
	}
	if cfg.Template != "parent-theme" {
		t.Errorf("Template = %q, want %q", cfg.Template, "parent-theme")
	}
	if cfg.TemplateURI != "../parent" {
		t.Errorf("TemplateURI = %q, want %q", cfg.TemplateURI, "../parent")
	}
	if cfg.Main != "style.css" {
		t.Errorf("Main = %q, want %q (default)", cfg.Main, "style.css")
	}
	if len(cfg.Include) != 3 {
		t.Errorf("Include count = %d, want 3", len(cfg.Include))
	}
	if !cfg.Minify {
		t.Error("Minify should be true")
	}
	if cfg.Tags != "modern, responsive" {
		t.Errorf("Tags = %q, want %q", cfg.Tags, "modern, responsive")
	}
}

func TestMixedDelimiters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mixed_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test that = takes precedence when both are present
	content := `name=Plugin With Equals
version: 1.0.0
main=main.php
author: Test
`
	propsPath := filepath.Join(tmpDir, "plugin.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPluginProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadPluginProperties error: %v", err)
	}

	if cfg.Name != "Plugin With Equals" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Plugin With Equals")
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
	}
	if cfg.Main != "main.php" {
		t.Errorf("Main = %q, want %q", cfg.Main, "main.php")
	}
	if cfg.Author != "Test" {
		t.Errorf("Author = %q, want %q", cfg.Author, "Test")
	}
}

func TestValueWithColon(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "colon_value_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test URL values that contain colons
	content := `name=My Plugin
main=plugin.php
author-uri=https://example.com
plugin-uri=http://plugin.example.com:8080/path
`
	propsPath := filepath.Join(tmpDir, "plugin.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPluginProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadPluginProperties error: %v", err)
	}

	if cfg.AuthorURI != "https://example.com" {
		t.Errorf("AuthorURI = %q, want %q", cfg.AuthorURI, "https://example.com")
	}
	if cfg.PluginURI != "http://plugin.example.com:8080/path" {
		t.Errorf("PluginURI = %q, want %q", cfg.PluginURI, "http://plugin.example.com:8080/path")
	}
}

func TestYAMLListSyntax(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yaml_list_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test YAML list syntax with - items
	content := `name: My Plugin
main: plugin.php
include:
  - src
  - "*.php"
  - assets/js
  - assets/css
exclude:
  - tests
  - node_modules
`
	propsPath := filepath.Join(tmpDir, "plugin.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPluginProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadPluginProperties error: %v", err)
	}

	if cfg.Name != "My Plugin" {
		t.Errorf("Name = %q, want %q", cfg.Name, "My Plugin")
	}
	if len(cfg.Include) != 4 {
		t.Errorf("Include count = %d, want 4. Got: %v", len(cfg.Include), cfg.Include)
	}
	if len(cfg.Exclude) != 2 {
		t.Errorf("Exclude count = %d, want 2. Got: %v", len(cfg.Exclude), cfg.Exclude)
	}

	// Check specific items
	expectedIncludes := []string{"src", "*.php", "assets/js", "assets/css"}
	for i, expected := range expectedIncludes {
		if i < len(cfg.Include) && cfg.Include[i] != expected {
			t.Errorf("Include[%d] = %q, want %q", i, cfg.Include[i], expected)
		}
	}
}

func TestYAMLNestedStructure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yaml_nested_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test that basic YAML with quoted strings works
	content := `name: My Plugin
main: plugin.php
description: "This is a description with special chars: @#$%"
version: "1.0.0"
`
	propsPath := filepath.Join(tmpDir, "plugin.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPluginProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadPluginProperties error: %v", err)
	}

	if cfg.Name != "My Plugin" {
		t.Errorf("Name = %q, want %q", cfg.Name, "My Plugin")
	}
	if !strings.Contains(cfg.Description, "special chars") {
		t.Errorf("Description should contain 'special chars', got: %q", cfg.Description)
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
	}
}

func TestMixedPropertiesAndYAMLLists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mixed_list_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Mix properties syntax with YAML lists
	content := `name=My Theme
version=1.0.0
main=style.css
include:
  - "*.php"
  - templates
  - assets
exclude=build, node_modules
minify=true
`
	propsPath := filepath.Join(tmpDir, "theme.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadThemeProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadThemeProperties error: %v", err)
	}

	if cfg.Name != "My Theme" {
		t.Errorf("Name = %q, want %q", cfg.Name, "My Theme")
	}
	if len(cfg.Include) != 3 {
		t.Errorf("Include count = %d, want 3. Got: %v", len(cfg.Include), cfg.Include)
	}
	if len(cfg.Exclude) != 2 {
		t.Errorf("Exclude count = %d, want 2. Got: %v", len(cfg.Exclude), cfg.Exclude)
	}
	if !cfg.Minify {
		t.Error("Minify should be true")
	}
}

func TestPropertiesGetWithDefault(t *testing.T) {
	props := Properties{
		"key1": "value1",
		"key2": "",
	}

	if props.GetWithDefault("key1", "default") != "value1" {
		t.Error("GetWithDefault should return existing value")
	}
	if props.GetWithDefault("key2", "default") != "default" {
		t.Error("GetWithDefault should return default for empty value")
	}
	if props.GetWithDefault("missing", "default") != "default" {
		t.Error("GetWithDefault should return default for missing key")
	}
}

func TestPropertiesGetBool(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"yes", true},
		{"1", true},
		{"anything", true},
		{"false", false},
		{"no", false},
		{"0", false},
		{"", false},
	}

	for _, tt := range tests {
		props := Properties{"key": tt.value}
		if props.GetBool("key") != tt.expected {
			t.Errorf("GetBool(%q) = %v, want %v", tt.value, props.GetBool("key"), tt.expected)
		}
	}

	// Test missing key
	props := Properties{}
	if props.GetBool("missing") != false {
		t.Error("GetBool should return false for missing key")
	}
}

func TestPropertiesGetList(t *testing.T) {
	tests := []struct {
		value    string
		expected int
	}{
		{"a, b, c", 3},
		{"single", 1},
		{"a,b,c", 3},
		{"", 0},
		{"  a  ,  b  ", 2},
	}

	for _, tt := range tests {
		props := Properties{"key": tt.value}
		result := props.GetList("key")
		if len(result) != tt.expected {
			t.Errorf("GetList(%q) = %d items, want %d", tt.value, len(result), tt.expected)
		}
	}

	// Test missing key
	props := Properties{}
	if len(props.GetList("missing")) != 0 {
		t.Error("GetList should return empty slice for missing key")
	}
}

func TestLoadPluginProperties(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		validate    func(*testing.T, *PluginConfig)
	}{
		{
			name: "basic plugin",
			content: `name=My Plugin
main=my-plugin.php
version=1.0.0
description=A test plugin
author=Test Author`,
			expectError: false,
			validate: func(t *testing.T, cfg *PluginConfig) {
				if cfg.Name != "My Plugin" {
					t.Errorf("Name = %q, want %q", cfg.Name, "My Plugin")
				}
				if cfg.Main != "my-plugin.php" {
					t.Errorf("Main = %q, want %q", cfg.Main, "my-plugin.php")
				}
				if cfg.Version != "1.0.0" {
					t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
				}
				if cfg.Description != "A test plugin" {
					t.Errorf("Description = %q, want %q", cfg.Description, "A test plugin")
				}
				if cfg.Author != "Test Author" {
					t.Errorf("Author = %q, want %q", cfg.Author, "Test Author")
				}
			},
		},
		{
			name: "with includes and excludes",
			content: `name=My Plugin
main=my-plugin.php
include=src, *.php, assets
exclude=tests, *.md`,
			expectError: false,
			validate: func(t *testing.T, cfg *PluginConfig) {
				if len(cfg.Include) != 3 {
					t.Errorf("Include count = %d, want 3", len(cfg.Include))
				}
				if len(cfg.Exclude) != 2 {
					t.Errorf("Exclude count = %d, want 2", len(cfg.Exclude))
				}
			},
		},
		{
			name: "with boolean options",
			content: `name=My Plugin
main=my-plugin.php
obfuscate=true
minify=yes`,
			expectError: false,
			validate: func(t *testing.T, cfg *PluginConfig) {
				if !cfg.Obfuscate {
					t.Error("Obfuscate should be true")
				}
				if !cfg.Minify {
					t.Error("Minify should be true")
				}
			},
		},
		{
			name: "boolean false values",
			content: `name=My Plugin
main=my-plugin.php
obfuscate=false
minify=no`,
			expectError: false,
			validate: func(t *testing.T, cfg *PluginConfig) {
				if cfg.Obfuscate {
					t.Error("Obfuscate should be false")
				}
				if cfg.Minify {
					t.Error("Minify should be false")
				}
			},
		},
		{
			name: "with comments",
			content: `# This is a comment
name=My Plugin
# Another comment
main=my-plugin.php`,
			expectError: false,
			validate: func(t *testing.T, cfg *PluginConfig) {
				if cfg.Name != "My Plugin" {
					t.Errorf("Name = %q, want %q", cfg.Name, "My Plugin")
				}
			},
		},
		{
			name: "missing name",
			content: `main=my-plugin.php
version=1.0.0`,
			expectError: true,
			validate:    nil,
		},
		{
			name: "missing main",
			content: `name=My Plugin
version=1.0.0`,
			expectError: true,
			validate:    nil,
		},
		{
			name: "all fields",
			content: `name=Full Plugin
version=2.0.0
description=Full description
author=John Doe
author-uri=https://example.com
plugin-uri=https://plugin.example.com
license=GPL-2.0
license-uri=https://www.gnu.org/licenses/gpl-2.0.html
main=full-plugin.php
text-domain=full-plugin
domain-path=/languages
requires=5.0
requires-php=7.4`,
			expectError: false,
			validate: func(t *testing.T, cfg *PluginConfig) {
				if cfg.AuthorURI != "https://example.com" {
					t.Errorf("AuthorURI = %q, want %q", cfg.AuthorURI, "https://example.com")
				}
				if cfg.PluginURI != "https://plugin.example.com" {
					t.Errorf("PluginURI = %q, want %q", cfg.PluginURI, "https://plugin.example.com")
				}
				if cfg.License != "GPL-2.0" {
					t.Errorf("License = %q, want %q", cfg.License, "GPL-2.0")
				}
				if cfg.TextDomain != "full-plugin" {
					t.Errorf("TextDomain = %q, want %q", cfg.TextDomain, "full-plugin")
				}
				if cfg.Requires != "5.0" {
					t.Errorf("Requires = %q, want %q", cfg.Requires, "5.0")
				}
				if cfg.RequiresPHP != "7.4" {
					t.Errorf("RequiresPHP = %q, want %q", cfg.RequiresPHP, "7.4")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "plugin_test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Write plugin.properties
			propsPath := filepath.Join(tmpDir, "plugin.properties")
			err = os.WriteFile(propsPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Load properties
			cfg, err := LoadPluginProperties(tmpDir)

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

func TestPluginExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "plugin_exists_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test when file doesn't exist
	if PluginExists(tmpDir) {
		t.Error("PluginExists should return false when file doesn't exist")
	}

	// Create plugin.properties
	propsPath := filepath.Join(tmpDir, "plugin.properties")
	err = os.WriteFile(propsPath, []byte("name=Test\nmain=test.php"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test when file exists
	if !PluginExists(tmpDir) {
		t.Error("PluginExists should return true when file exists")
	}
}

func TestLoadPluginPropertiesFileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_notfound_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = LoadPluginProperties(tmpDir)
	if err == nil {
		t.Error("Expected error when plugin.properties doesn't exist")
	}
}
