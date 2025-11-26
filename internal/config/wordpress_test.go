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

func TestResolvePluginURI(t *testing.T) {
	// Create a temp directory structure for testing
	tmpDir, err := os.MkdirTemp("", "resolve_plugin_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directory structure:
	// tmpDir/
	//   plugins/
	//     my-plugin/
	//       plugin.zip
	//       plugin.properties
	//     other-plugin/
	//       plugin.properties  (no zip - needs build)
	//     prebuilt.zip
	//   relative-plugin/
	//     plugin.zip

	// Create plugins directory
	pluginsDir := filepath.Join(tmpDir, "plugins")
	os.MkdirAll(filepath.Join(pluginsDir, "my-plugin"), 0755)
	os.MkdirAll(filepath.Join(pluginsDir, "other-plugin"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "relative-plugin"), 0755)

	// Create plugin.zip files
	os.WriteFile(filepath.Join(pluginsDir, "my-plugin", "plugin.zip"), []byte("fake zip"), 0644)
	os.WriteFile(filepath.Join(pluginsDir, "prebuilt.zip"), []byte("fake zip"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "relative-plugin", "plugin.zip"), []byte("fake zip"), 0644)

	// Create plugin.properties files
	pluginProps := "name=My Plugin\nmain=plugin.php"
	os.WriteFile(filepath.Join(pluginsDir, "my-plugin", "plugin.properties"), []byte(pluginProps), 0644)
	os.WriteFile(filepath.Join(pluginsDir, "other-plugin", "plugin.properties"), []byte(pluginProps), 0644)

	tests := []struct {
		name           string
		plugin         WordPressPlugin
		wantIsLocal    bool
		wantNeedsBuild bool
		wantZipPath    string
		wantBuildDir   string
	}{
		{
			name:           "WordPress.org plugin (no local match)",
			plugin:         WordPressPlugin{Slug: "akismet"},
			wantIsLocal:    false,
			wantNeedsBuild: false,
		},
		{
			name:           "Plugin with explicit HTTP URL",
			plugin:         WordPressPlugin{Slug: "custom", URI: "https://example.com/plugin.zip"},
			wantIsLocal:    false,
			wantNeedsBuild: false,
			wantZipPath:    "https://example.com/plugin.zip",
		},
		{
			name:           "Plugin with explicit local path",
			plugin:         WordPressPlugin{Slug: "custom", URI: "path/to/plugin.zip"},
			wantIsLocal:    true,
			wantNeedsBuild: false,
			wantZipPath:    filepath.Join(tmpDir, "path/to/plugin.zip"),
		},
		{
			name:           "Plugin found in plugins/<slug>/plugin.zip",
			plugin:         WordPressPlugin{Slug: "my-plugin"},
			wantIsLocal:    true,
			wantNeedsBuild: false,
			wantZipPath:    filepath.Join(pluginsDir, "my-plugin", "plugin.zip"),
		},
		{
			name:           "Plugin found in plugins/<slug>.zip",
			plugin:         WordPressPlugin{Slug: "prebuilt"},
			wantIsLocal:    true,
			wantNeedsBuild: false,
			wantZipPath:    filepath.Join(pluginsDir, "prebuilt.zip"),
		},
		{
			name:           "Plugin found in <slug>/plugin.zip (relative path)",
			plugin:         WordPressPlugin{Slug: "relative-plugin"},
			wantIsLocal:    true,
			wantNeedsBuild: false,
			wantZipPath:    filepath.Join(tmpDir, "relative-plugin", "plugin.zip"),
		},
		{
			name:           "Plugin needs build (has plugin.properties but no zip)",
			plugin:         WordPressPlugin{Slug: "other-plugin"},
			wantIsLocal:    true,
			wantNeedsBuild: true,
			wantBuildDir:   filepath.Join(pluginsDir, "other-plugin"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolvePluginURI(tmpDir, tt.plugin)

			if result.IsLocal != tt.wantIsLocal {
				t.Errorf("IsLocal = %v, want %v", result.IsLocal, tt.wantIsLocal)
			}

			if result.NeedsBuild != tt.wantNeedsBuild {
				t.Errorf("NeedsBuild = %v, want %v", result.NeedsBuild, tt.wantNeedsBuild)
			}

			if tt.wantZipPath != "" && result.ZipPath != tt.wantZipPath {
				t.Errorf("ZipPath = %q, want %q", result.ZipPath, tt.wantZipPath)
			}

			if tt.wantBuildDir != "" && result.BuildDir != tt.wantBuildDir {
				t.Errorf("BuildDir = %q, want %q", result.BuildDir, tt.wantBuildDir)
			}
		})
	}
}

func TestResolvePluginURI_RelativePath(t *testing.T) {
	// Test with relative paths like ../../some-plugin
	tmpDir, err := os.MkdirTemp("", "resolve_relative_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create structure:
	// tmpDir/
	//   project/
	//     subdir/
	//       (baseDir - where wordpress.properties would be)
	//   external-plugin/
	//     plugin.properties

	baseDir := filepath.Join(tmpDir, "project", "subdir")
	externalPluginDir := filepath.Join(tmpDir, "external-plugin")
	os.MkdirAll(baseDir, 0755)
	os.MkdirAll(externalPluginDir, 0755)

	// Create plugin.properties in external-plugin
	pluginProps := "name=External Plugin\nmain=plugin.php"
	os.WriteFile(filepath.Join(externalPluginDir, "plugin.properties"), []byte(pluginProps), 0644)

	// Test relative path resolution
	plugin := WordPressPlugin{Slug: "../../external-plugin"}
	result := ResolvePluginURI(baseDir, plugin)

	if !result.IsLocal {
		t.Error("IsLocal should be true for relative path to local plugin")
	}

	if !result.NeedsBuild {
		t.Error("NeedsBuild should be true (plugin.properties exists, no zip)")
	}

	// The BuildDir should resolve to the external-plugin directory
	expectedBuildDir := filepath.Join(baseDir, "../../external-plugin")
	cleanExpected := filepath.Clean(expectedBuildDir)
	cleanActual := filepath.Clean(result.BuildDir)

	if cleanActual != cleanExpected {
		t.Errorf("BuildDir = %q, want %q", result.BuildDir, expectedBuildDir)
	}
}

func TestResolveThemeURI(t *testing.T) {
	// Create a temp directory structure for testing
	tmpDir, err := os.MkdirTemp("", "resolve_theme_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directory structure:
	// tmpDir/
	//   themes/
	//     my-theme/
	//       theme.zip
	//       theme.properties
	//     other-theme/
	//       theme.properties  (no zip - needs build)
	//     prebuilt.zip
	//   relative-theme/
	//     theme.zip

	// Create themes directory
	themesDir := filepath.Join(tmpDir, "themes")
	os.MkdirAll(filepath.Join(themesDir, "my-theme"), 0755)
	os.MkdirAll(filepath.Join(themesDir, "other-theme"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "relative-theme"), 0755)

	// Create theme.zip files
	os.WriteFile(filepath.Join(themesDir, "my-theme", "theme.zip"), []byte("fake zip"), 0644)
	os.WriteFile(filepath.Join(themesDir, "prebuilt.zip"), []byte("fake zip"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "relative-theme", "theme.zip"), []byte("fake zip"), 0644)

	// Create theme.properties files
	themeProps := "name=My Theme"
	os.WriteFile(filepath.Join(themesDir, "my-theme", "theme.properties"), []byte(themeProps), 0644)
	os.WriteFile(filepath.Join(themesDir, "other-theme", "theme.properties"), []byte(themeProps), 0644)

	tests := []struct {
		name           string
		theme          WordPressTheme
		wantIsLocal    bool
		wantNeedsBuild bool
		wantZipPath    string
		wantBuildDir   string
	}{
		{
			name:           "WordPress.org theme (no local match)",
			theme:          WordPressTheme{Slug: "twentytwentyfour"},
			wantIsLocal:    false,
			wantNeedsBuild: false,
		},
		{
			name:           "Theme with explicit HTTP URL",
			theme:          WordPressTheme{Slug: "custom", URI: "https://example.com/theme.zip"},
			wantIsLocal:    false,
			wantNeedsBuild: false,
			wantZipPath:    "https://example.com/theme.zip",
		},
		{
			name:           "Theme with explicit local path",
			theme:          WordPressTheme{Slug: "custom", URI: "path/to/theme.zip"},
			wantIsLocal:    true,
			wantNeedsBuild: false,
			wantZipPath:    filepath.Join(tmpDir, "path/to/theme.zip"),
		},
		{
			name:           "Theme found in themes/<slug>/theme.zip",
			theme:          WordPressTheme{Slug: "my-theme"},
			wantIsLocal:    true,
			wantNeedsBuild: false,
			wantZipPath:    filepath.Join(themesDir, "my-theme", "theme.zip"),
		},
		{
			name:           "Theme found in themes/<slug>.zip",
			theme:          WordPressTheme{Slug: "prebuilt"},
			wantIsLocal:    true,
			wantNeedsBuild: false,
			wantZipPath:    filepath.Join(themesDir, "prebuilt.zip"),
		},
		{
			name:           "Theme found in <slug>/theme.zip (relative path)",
			theme:          WordPressTheme{Slug: "relative-theme"},
			wantIsLocal:    true,
			wantNeedsBuild: false,
			wantZipPath:    filepath.Join(tmpDir, "relative-theme", "theme.zip"),
		},
		{
			name:           "Theme needs build (has theme.properties but no zip)",
			theme:          WordPressTheme{Slug: "other-theme"},
			wantIsLocal:    true,
			wantNeedsBuild: true,
			wantBuildDir:   filepath.Join(themesDir, "other-theme"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveThemeURI(tmpDir, tt.theme)

			if result.IsLocal != tt.wantIsLocal {
				t.Errorf("IsLocal = %v, want %v", result.IsLocal, tt.wantIsLocal)
			}

			if result.NeedsBuild != tt.wantNeedsBuild {
				t.Errorf("NeedsBuild = %v, want %v", result.NeedsBuild, tt.wantNeedsBuild)
			}

			if tt.wantZipPath != "" && result.ZipPath != tt.wantZipPath {
				t.Errorf("ZipPath = %q, want %q", result.ZipPath, tt.wantZipPath)
			}

			if tt.wantBuildDir != "" && result.BuildDir != tt.wantBuildDir {
				t.Errorf("BuildDir = %q, want %q", result.BuildDir, tt.wantBuildDir)
			}
		})
	}
}

func TestResolveThemeURI_RelativePath(t *testing.T) {
	// Test with relative paths like ../../some-theme
	tmpDir, err := os.MkdirTemp("", "resolve_theme_relative_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create structure:
	// tmpDir/
	//   project/
	//     subdir/
	//       (baseDir - where wordpress.properties would be)
	//   external-theme/
	//     theme.properties

	baseDir := filepath.Join(tmpDir, "project", "subdir")
	externalThemeDir := filepath.Join(tmpDir, "external-theme")
	os.MkdirAll(baseDir, 0755)
	os.MkdirAll(externalThemeDir, 0755)

	// Create theme.properties in external-theme
	themeProps := "name=External Theme"
	os.WriteFile(filepath.Join(externalThemeDir, "theme.properties"), []byte(themeProps), 0644)

	// Test relative path resolution
	theme := WordPressTheme{Slug: "../../external-theme"}
	result := ResolveThemeURI(baseDir, theme)

	if !result.IsLocal {
		t.Error("IsLocal should be true for relative path to local theme")
	}

	if !result.NeedsBuild {
		t.Error("NeedsBuild should be true (theme.properties exists, no zip)")
	}

	// The BuildDir should resolve to the external-theme directory
	expectedBuildDir := filepath.Join(baseDir, "../../external-theme")
	cleanExpected := filepath.Clean(expectedBuildDir)
	cleanActual := filepath.Clean(result.BuildDir)

	if cleanActual != cleanExpected {
		t.Errorf("BuildDir = %q, want %q", result.BuildDir, expectedBuildDir)
	}
}

func TestResolvePluginURI_URLs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resolve_url_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		plugin      WordPressPlugin
		wantIsLocal bool
		wantZipPath string
	}{
		{
			name:        "HTTPS URL",
			plugin:      WordPressPlugin{Slug: "my-plugin", URI: "https://example.com/plugins/my-plugin.zip"},
			wantIsLocal: false,
			wantZipPath: "https://example.com/plugins/my-plugin.zip",
		},
		{
			name:        "HTTP URL",
			plugin:      WordPressPlugin{Slug: "my-plugin", URI: "http://example.com/plugins/my-plugin.zip"},
			wantIsLocal: false,
			wantZipPath: "http://example.com/plugins/my-plugin.zip",
		},
		{
			name:        "GitHub releases URL",
			plugin:      WordPressPlugin{Slug: "woocommerce", URI: "https://github.com/woocommerce/woocommerce/releases/download/8.0.0/woocommerce.zip"},
			wantIsLocal: false,
			wantZipPath: "https://github.com/woocommerce/woocommerce/releases/download/8.0.0/woocommerce.zip",
		},
		{
			name:        "URL with query params",
			plugin:      WordPressPlugin{Slug: "premium-plugin", URI: "https://example.com/download.php?id=123&token=abc"},
			wantIsLocal: false,
			wantZipPath: "https://example.com/download.php?id=123&token=abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolvePluginURI(tmpDir, tt.plugin)

			if result.IsLocal != tt.wantIsLocal {
				t.Errorf("IsLocal = %v, want %v", result.IsLocal, tt.wantIsLocal)
			}

			if result.ZipPath != tt.wantZipPath {
				t.Errorf("ZipPath = %q, want %q", result.ZipPath, tt.wantZipPath)
			}

			if result.NeedsBuild {
				t.Error("NeedsBuild should be false for URL")
			}
		})
	}
}

func TestResolveThemeURI_URLs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resolve_theme_url_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		theme       WordPressTheme
		wantIsLocal bool
		wantZipPath string
	}{
		{
			name:        "HTTPS URL",
			theme:       WordPressTheme{Slug: "my-theme", URI: "https://example.com/themes/my-theme.zip"},
			wantIsLocal: false,
			wantZipPath: "https://example.com/themes/my-theme.zip",
		},
		{
			name:        "HTTP URL",
			theme:       WordPressTheme{Slug: "my-theme", URI: "http://example.com/themes/my-theme.zip"},
			wantIsLocal: false,
			wantZipPath: "http://example.com/themes/my-theme.zip",
		},
		{
			name:        "GitHub releases URL",
			theme:       WordPressTheme{Slug: "storefront", URI: "https://github.com/woocommerce/storefront/releases/download/4.0.0/storefront.zip"},
			wantIsLocal: false,
			wantZipPath: "https://github.com/woocommerce/storefront/releases/download/4.0.0/storefront.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveThemeURI(tmpDir, tt.theme)

			if result.IsLocal != tt.wantIsLocal {
				t.Errorf("IsLocal = %v, want %v", result.IsLocal, tt.wantIsLocal)
			}

			if result.ZipPath != tt.wantZipPath {
				t.Errorf("ZipPath = %q, want %q", result.ZipPath, tt.wantZipPath)
			}

			if result.NeedsBuild {
				t.Error("NeedsBuild should be false for URL")
			}
		})
	}
}

func TestFileExistsAndIsFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_exists_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with non-existent file
	if fileExistsAndIsFile(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("fileExistsAndIsFile should return false for non-existent file")
	}

	// Test with existing file
	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("test"), 0644)
	if !fileExistsAndIsFile(filePath) {
		t.Error("fileExistsAndIsFile should return true for existing file")
	}

	// Test with directory (should return false)
	dirPath := filepath.Join(tmpDir, "testdir")
	os.MkdirAll(dirPath, 0755)
	if fileExistsAndIsFile(dirPath) {
		t.Error("fileExistsAndIsFile should return false for directory")
	}
}

func TestExtractSlugFromURL(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "GitHub repo URL",
			uri:      "https://github.com/owner/my-plugin",
			expected: "my-plugin",
		},
		{
			name:     "GitHub repo URL with trailing slash",
			uri:      "https://github.com/owner/my-plugin/",
			expected: "my-plugin",
		},
		{
			name:     "GitHub repo URL with /releases",
			uri:      "https://github.com/owner/my-plugin/releases",
			expected: "my-plugin",
		},
		{
			name:     "GitHub repo URL with /releases/",
			uri:      "https://github.com/owner/my-plugin/releases/",
			expected: "my-plugin",
		},
		{
			name:     "Direct zip URL",
			uri:      "https://example.com/downloads/my-plugin.zip",
			expected: "my-plugin",
		},
		{
			name:     "Simple URL",
			uri:      "https://example.com/my-plugin",
			expected: "my-plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSlugFromURL(tt.uri)
			if result != tt.expected {
				t.Errorf("extractSlugFromURL(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestParsePluginItemWithURL(t *testing.T) {
	// Test that a URL string is parsed correctly
	plugin := parsePluginItem("https://github.com/owner/my-plugin")
	if plugin.Slug != "my-plugin" {
		t.Errorf("Expected slug 'my-plugin', got %q", plugin.Slug)
	}
	if plugin.URI != "https://github.com/owner/my-plugin" {
		t.Errorf("Expected URI to be set, got %q", plugin.URI)
	}
	if !plugin.Active {
		t.Error("Expected plugin to be active by default")
	}
}

func TestParseThemeItemWithURL(t *testing.T) {
	// Test that a URL string is parsed correctly
	theme := parseThemeItem("https://github.com/owner/my-theme/releases", true)
	if theme.Slug != "my-theme" {
		t.Errorf("Expected slug 'my-theme', got %q", theme.Slug)
	}
	if theme.URI != "https://github.com/owner/my-theme/releases" {
		t.Errorf("Expected URI to be set, got %q", theme.URI)
	}
	if !theme.Active {
		t.Error("Expected theme to be active (first theme)")
	}
}
