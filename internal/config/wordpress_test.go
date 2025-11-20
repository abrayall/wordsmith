package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWordPressProperties(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantName    string
		wantImage   string
		wantPlugins int
		wantThemes  int
		wantErr     bool
	}{
		{
			name: "basic with defaults",
			content: `name: My WordPress
`,
			wantName:    "My WordPress",
			wantImage:   "wordpress:latest",
			wantPlugins: 0,
			wantThemes:  0,
		},
		{
			name: "custom image",
			content: `name: My WordPress
image: wordpress:6.4-php8.2
`,
			wantName:  "My WordPress",
			wantImage: "wordpress:6.4-php8.2",
		},
		{
			name: "simple plugin list",
			content: `name: Test Site
plugins:
  - akismet
  - jetpack
  - woocommerce
`,
			wantName:    "Test Site",
			wantImage:   "wordpress:latest",
			wantPlugins: 3,
		},
		{
			name: "detailed plugin with uri",
			content: `name: Test Site
plugins:
  - slug: my-plugin
    uri: https://example.com/my-plugin.zip
    active: true
  - slug: other-plugin
    active: false
`,
			wantName:    "Test Site",
			wantPlugins: 2,
		},
		{
			name: "themes list",
			content: `name: Test Site
themes:
  - twentytwentyfour
  - astra
`,
			wantName:   "Test Site",
			wantThemes: 2,
		},
		{
			name: "theme with active",
			content: `name: Test Site
themes:
  - slug: astra
    active: true
  - slug: oceanwp
`,
			wantName:   "Test Site",
			wantThemes: 2,
		},
		{
			name: "mixed plugins and themes",
			content: `name: Full Site
image: wordpress:6.4
plugins:
  - akismet
  - slug: woocommerce
    active: true
themes:
  - slug: storefront
    active: true
`,
			wantName:    "Full Site",
			wantImage:   "wordpress:6.4",
			wantPlugins: 2,
			wantThemes:  1,
		},
		{
			name: "properties syntax",
			content: `name=My Site
image=wordpress:php8.1
`,
			wantName:  "My Site",
			wantImage: "wordpress:php8.1",
		},
		{
			name: "no name defaults empty",
			content: `image: wordpress:latest
plugins:
  - akismet
`,
			wantName:    "",
			wantImage:   "wordpress:latest",
			wantPlugins: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "wp_test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			propsPath := filepath.Join(tmpDir, "wordpress.properties")
			err = os.WriteFile(propsPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadWordPressProperties(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadWordPressProperties() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if cfg.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cfg.Name, tt.wantName)
			}

			if tt.wantImage != "" && cfg.Image != tt.wantImage {
				t.Errorf("Image = %q, want %q", cfg.Image, tt.wantImage)
			}

			if tt.wantPlugins > 0 && len(cfg.Plugins) != tt.wantPlugins {
				t.Errorf("len(Plugins) = %d, want %d", len(cfg.Plugins), tt.wantPlugins)
			}

			if tt.wantThemes > 0 && len(cfg.Themes) != tt.wantThemes {
				t.Errorf("len(Themes) = %d, want %d", len(cfg.Themes), tt.wantThemes)
			}
		})
	}
}

func TestWordPressPluginParsing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wp_plugin_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `name: Test
plugins:
  - slug: plugin1
    uri: https://example.com/plugin1.zip
    active: true
  - slug: plugin2
    uri: /path/to/plugin2.zip
    active: false
  - simple-plugin
`
	propsPath := filepath.Join(tmpDir, "wordpress.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWordPressProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadWordPressProperties error: %v", err)
	}

	if len(cfg.Plugins) != 3 {
		t.Logf("Plugins: %+v", cfg.Plugins)
		t.Fatalf("Expected 3 plugins, got %d", len(cfg.Plugins))
	}

	// Check first plugin
	if cfg.Plugins[0].Slug != "plugin1" {
		t.Errorf("Plugin[0].Slug = %q, want %q", cfg.Plugins[0].Slug, "plugin1")
	}
	if cfg.Plugins[0].URI != "https://example.com/plugin1.zip" {
		t.Errorf("Plugin[0].URI = %q, want %q", cfg.Plugins[0].URI, "https://example.com/plugin1.zip")
	}
	if !cfg.Plugins[0].Active {
		t.Error("Plugin[0].Active should be true")
	}

	// Check second plugin
	if cfg.Plugins[1].Slug != "plugin2" {
		t.Errorf("Plugin[1].Slug = %q, want %q", cfg.Plugins[1].Slug, "plugin2")
	}
	if cfg.Plugins[1].URI != "/path/to/plugin2.zip" {
		t.Errorf("Plugin[1].URI = %q, want %q", cfg.Plugins[1].URI, "/path/to/plugin2.zip")
	}
	if cfg.Plugins[1].Active {
		t.Error("Plugin[1].Active should be false")
	}

	// Check third plugin (simple string)
	if cfg.Plugins[2].Slug != "simple-plugin" {
		t.Errorf("Plugin[2].Slug = %q, want %q", cfg.Plugins[2].Slug, "simple-plugin")
	}
	if cfg.Plugins[2].URI != "" {
		t.Errorf("Plugin[2].URI = %q, want empty", cfg.Plugins[2].URI)
	}
	if !cfg.Plugins[2].Active {
		t.Error("Plugin[2].Active should default to true")
	}
}

func TestWordPressThemeParsing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wp_theme_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `name: Test
themes:
  - slug: theme1
    uri: https://example.com/theme1.zip
    active: true
  - slug: theme2
  - simple-theme
`
	propsPath := filepath.Join(tmpDir, "wordpress.properties")
	err = os.WriteFile(propsPath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWordPressProperties(tmpDir)
	if err != nil {
		t.Fatalf("LoadWordPressProperties error: %v", err)
	}

	if len(cfg.Themes) != 3 {
		t.Fatalf("Expected 3 themes, got %d", len(cfg.Themes))
	}

	// Check first theme
	if cfg.Themes[0].Slug != "theme1" {
		t.Errorf("Theme[0].Slug = %q, want %q", cfg.Themes[0].Slug, "theme1")
	}
	if cfg.Themes[0].URI != "https://example.com/theme1.zip" {
		t.Errorf("Theme[0].URI = %q, want %q", cfg.Themes[0].URI, "https://example.com/theme1.zip")
	}
	if !cfg.Themes[0].Active {
		t.Error("Theme[0].Active should be true")
	}

	// Check second theme
	if cfg.Themes[1].Slug != "theme2" {
		t.Errorf("Theme[1].Slug = %q, want %q", cfg.Themes[1].Slug, "theme2")
	}
	if cfg.Themes[1].Active {
		t.Error("Theme[1].Active should default to false (not first theme)")
	}

	// Check third theme (simple string)
	if cfg.Themes[2].Slug != "simple-theme" {
		t.Errorf("Theme[2].Slug = %q, want %q", cfg.Themes[2].Slug, "simple-theme")
	}
	if cfg.Themes[2].Active {
		t.Error("Theme[2].Active should default to false (not first theme)")
	}
}

func TestWordPressExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wp_exists_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Should not exist initially
	if WordPressExists(tmpDir) {
		t.Error("WordPressExists should return false when file doesn't exist")
	}

	// Create the file
	propsPath := filepath.Join(tmpDir, "wordpress.properties")
	err = os.WriteFile(propsPath, []byte("name: Test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Should exist now
	if !WordPressExists(tmpDir) {
		t.Error("WordPressExists should return true when file exists")
	}
}
