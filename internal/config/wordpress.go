package config

import (
	"os"
	"path/filepath"
	"strings"
)

// WordPressPlugin represents a plugin to install
type WordPressPlugin struct {
	Slug    string
	Version string // Specific version to install
	URI     string // HTTP URL or file path
	Active  bool
}

// WordPressTheme represents a theme to install
type WordPressTheme struct {
	Slug    string
	Version string // Specific version to install
	URI     string // HTTP URL or file path
	Active  bool
}

// WordPressConfig represents the wordpress.properties configuration
type WordPressConfig struct {
	Name    string             // Instance name (optional, defaults to plugin/theme name or directory)
	Image   string             // Docker image (defaults to "wordpress:latest")
	Plugins []WordPressPlugin
	Themes  []WordPressTheme
}

// LoadWordPressProperties loads WordPress configuration from wordpress.properties file
func LoadWordPressProperties(dir string) (*WordPressConfig, error) {
	path := filepath.Join(dir, "wordpress.properties")
	props, err := ParseProperties(path)
	if err != nil {
		return nil, err
	}

	config := &WordPressConfig{
		Name:  props.Get("name"),
		Image: props.GetWithDefault("image", "wordpress:latest"),
	}

	// Parse plugins
	// Format can be:
	// plugins:
	//   - slug: akismet
	//     uri: https://example.com/plugin.zip
	//     active: true
	// Or simple list:
	// plugins:
	//   - akismet
	//   - jetpack
	pluginsVal, ok := props["plugins"]
	if ok {
		config.Plugins = parsePluginsList(pluginsVal)
	}

	// Parse themes
	themesVal, ok := props["themes"]
	if ok {
		config.Themes = parseThemesList(themesVal)
	}

	return config, nil
}

// parsePluginsList parses the plugins list from various formats
func parsePluginsList(val interface{}) []WordPressPlugin {
	var plugins []WordPressPlugin

	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			plugin := parsePluginItem(item)
			if plugin.Slug != "" {
				plugins = append(plugins, plugin)
			}
		}
	case []map[string]interface{}:
		for _, item := range v {
			plugin := parsePluginItem(item)
			if plugin.Slug != "" {
				plugins = append(plugins, plugin)
			}
		}
	case string:
		// Single plugin as string
		if v != "" {
			plugins = append(plugins, WordPressPlugin{Slug: v, Active: true})
		}
	default:
		// For debugging - should not happen
		_ = v
	}

	return plugins
}

// parsePluginItem parses a single plugin item
func parsePluginItem(item interface{}) WordPressPlugin {
	switch v := item.(type) {
	case string:
		// Simple slug
		return WordPressPlugin{Slug: v, Active: true}
	case Properties:
		// Map with slug, version, uri, active (from our YAML parser)
		plugin := WordPressPlugin{Active: true} // Default to active

		if slug, ok := v["slug"].(string); ok {
			plugin.Slug = slug
		}
		if version, ok := v["version"].(string); ok {
			plugin.Version = version
		}
		if uri, ok := v["uri"].(string); ok {
			plugin.URI = uri
		}
		if active, ok := v["active"].(bool); ok {
			plugin.Active = active
		} else if activeStr, ok := v["active"].(string); ok {
			plugin.Active = !(activeStr == "false" || activeStr == "no" || activeStr == "0")
		}

		return plugin
	case map[string]interface{}:
		// Map with slug, version, uri, active
		plugin := WordPressPlugin{Active: true} // Default to active

		if slug, ok := v["slug"].(string); ok {
			plugin.Slug = slug
		}
		if version, ok := v["version"].(string); ok {
			plugin.Version = version
		}
		if uri, ok := v["uri"].(string); ok {
			plugin.URI = uri
		}
		if active, ok := v["active"].(bool); ok {
			plugin.Active = active
		} else if activeStr, ok := v["active"].(string); ok {
			plugin.Active = !(activeStr == "false" || activeStr == "no" || activeStr == "0")
		}

		return plugin
	}

	return WordPressPlugin{}
}

// parseThemesList parses the themes list from various formats
func parseThemesList(val interface{}) []WordPressTheme {
	var themes []WordPressTheme

	switch v := val.(type) {
	case []interface{}:
		for i, item := range v {
			theme := parseThemeItem(item, i == 0)
			if theme.Slug != "" {
				themes = append(themes, theme)
			}
		}
	case string:
		// Single theme as string - first theme defaults to active
		if v != "" {
			themes = append(themes, WordPressTheme{Slug: v, Active: true})
		}
	}

	return themes
}

// parseThemeItem parses a single theme item
// isFirst indicates if this is the first theme in the list (defaults to active)
func parseThemeItem(item interface{}, isFirst bool) WordPressTheme {
	switch v := item.(type) {
	case string:
		// Simple slug - first theme defaults to active
		return WordPressTheme{Slug: v, Active: isFirst}
	case Properties:
		// Map with slug, version, uri, active (from our YAML parser)
		theme := WordPressTheme{Active: isFirst} // First theme defaults to active

		if slug, ok := v["slug"].(string); ok {
			theme.Slug = slug
		}
		if version, ok := v["version"].(string); ok {
			theme.Version = version
		}
		if uri, ok := v["uri"].(string); ok {
			theme.URI = uri
		}
		// Explicit active setting overrides default
		if active, ok := v["active"].(bool); ok {
			theme.Active = active
		} else if activeStr, ok := v["active"].(string); ok {
			theme.Active = !(activeStr == "false" || activeStr == "no" || activeStr == "0")
		}

		return theme
	case map[string]interface{}:
		// Map with slug, version, uri, active
		theme := WordPressTheme{Active: isFirst} // First theme defaults to active

		if slug, ok := v["slug"].(string); ok {
			theme.Slug = slug
		}
		if version, ok := v["version"].(string); ok {
			theme.Version = version
		}
		if uri, ok := v["uri"].(string); ok {
			theme.URI = uri
		}
		// Explicit active setting overrides default
		if active, ok := v["active"].(bool); ok {
			theme.Active = active
		} else if activeStr, ok := v["active"].(string); ok {
			theme.Active = !(activeStr == "false" || activeStr == "no" || activeStr == "0")
		}

		return theme
	}

	return WordPressTheme{}
}

// WordPressExists checks if wordpress.properties exists in the directory
func WordPressExists(dir string) bool {
	return PropertiesFileExists(dir, "wordpress.properties")
}

// PluginResolution represents the result of resolving a plugin slug
type PluginResolution struct {
	Slug       string // Original slug
	ZipPath    string // Path to zip file (if exists or will be built)
	BuildDir   string // Directory containing plugin.properties (if needs build)
	NeedsBuild bool   // True if plugin.properties exists but no zip
	IsLocal    bool   // True if this is a local plugin (not from wordpress.org)
}

// ThemeResolution represents the result of resolving a theme slug
type ThemeResolution struct {
	Slug       string // Original slug
	ZipPath    string // Path to zip file (if exists or will be built)
	BuildDir   string // Directory containing theme.properties (if needs build)
	NeedsBuild bool   // True if theme.properties exists but no zip
	IsLocal    bool   // True if this is a local theme (not from wordpress.org)
}

// ResolvePluginURI resolves a plugin slug or URI to determine how to install it.
// It checks in this order:
// 1. If URI is an HTTP URL or absolute file path, use as-is
// 2. Check plugins/<slug>/plugin.zip
// 3. Check plugins/<slug>.zip
// 4. Check <slug>/plugin.zip (relative path)
// 5. Check <slug>.zip (relative path)
// 6. Check plugins/<slug>/plugin.properties (needs build)
// 7. Check <slug>/plugin.properties (needs build)
// 8. Otherwise, treat as WordPress.org slug
func ResolvePluginURI(baseDir string, plugin WordPressPlugin) PluginResolution {
	result := PluginResolution{Slug: plugin.Slug}

	// If URI is already set, check if it's a URL or absolute path
	if plugin.URI != "" {
		if strings.HasPrefix(plugin.URI, "http://") || strings.HasPrefix(plugin.URI, "https://") {
			result.ZipPath = plugin.URI
			result.IsLocal = false
			return result
		}
		// Treat as file path - could be absolute or relative
		if filepath.IsAbs(plugin.URI) {
			result.ZipPath = plugin.URI
			result.IsLocal = true
			return result
		}
		// Relative path - resolve from baseDir
		result.ZipPath = filepath.Join(baseDir, plugin.URI)
		result.IsLocal = true
		return result
	}

	slug := plugin.Slug

	// Check plugins/<slug>/plugin.zip
	zipPath := filepath.Join(baseDir, "plugins", slug, "plugin.zip")
	if fileExistsAndIsFile(zipPath) {
		result.ZipPath = zipPath
		result.IsLocal = true
		return result
	}

	// Check plugins/<slug>.zip
	zipPath = filepath.Join(baseDir, "plugins", slug+".zip")
	if fileExistsAndIsFile(zipPath) {
		result.ZipPath = zipPath
		result.IsLocal = true
		return result
	}

	// Check <slug>/plugin.zip (for relative directory reference)
	zipPath = filepath.Join(baseDir, slug, "plugin.zip")
	if fileExistsAndIsFile(zipPath) {
		result.ZipPath = zipPath
		result.IsLocal = true
		return result
	}

	// Check <slug>.zip (for relative file reference)
	zipPath = filepath.Join(baseDir, slug+".zip")
	if fileExistsAndIsFile(zipPath) {
		result.ZipPath = zipPath
		result.IsLocal = true
		return result
	}

	// Check plugins/<slug>/plugin.properties (needs build)
	propsDir := filepath.Join(baseDir, "plugins", slug)
	if PluginExists(propsDir) {
		result.BuildDir = propsDir
		result.NeedsBuild = true
		result.IsLocal = true
		// The zip will be in build/<plugin-name>-<version>.zip after build
		return result
	}

	// Check <slug>/plugin.properties (for relative directory reference)
	propsDir = filepath.Join(baseDir, slug)
	if PluginExists(propsDir) {
		result.BuildDir = propsDir
		result.NeedsBuild = true
		result.IsLocal = true
		return result
	}

	// No local resolution found - treat as WordPress.org slug
	result.IsLocal = false
	return result
}

// ResolveThemeURI resolves a theme slug or URI to determine how to install it.
// It checks in this order:
// 1. If URI is an HTTP URL or absolute file path, use as-is
// 2. Check themes/<slug>/theme.zip
// 3. Check themes/<slug>.zip
// 4. Check <slug>/theme.zip (relative path)
// 5. Check <slug>.zip (relative path)
// 6. Check themes/<slug>/theme.properties (needs build)
// 7. Check <slug>/theme.properties (needs build)
// 8. Otherwise, treat as WordPress.org slug
func ResolveThemeURI(baseDir string, theme WordPressTheme) ThemeResolution {
	result := ThemeResolution{Slug: theme.Slug}

	// If URI is already set, check if it's a URL or absolute path
	if theme.URI != "" {
		if strings.HasPrefix(theme.URI, "http://") || strings.HasPrefix(theme.URI, "https://") {
			result.ZipPath = theme.URI
			result.IsLocal = false
			return result
		}
		// Treat as file path - could be absolute or relative
		if filepath.IsAbs(theme.URI) {
			result.ZipPath = theme.URI
			result.IsLocal = true
			return result
		}
		// Relative path - resolve from baseDir
		result.ZipPath = filepath.Join(baseDir, theme.URI)
		result.IsLocal = true
		return result
	}

	slug := theme.Slug

	// Check themes/<slug>/theme.zip
	zipPath := filepath.Join(baseDir, "themes", slug, "theme.zip")
	if fileExistsAndIsFile(zipPath) {
		result.ZipPath = zipPath
		result.IsLocal = true
		return result
	}

	// Check themes/<slug>.zip
	zipPath = filepath.Join(baseDir, "themes", slug+".zip")
	if fileExistsAndIsFile(zipPath) {
		result.ZipPath = zipPath
		result.IsLocal = true
		return result
	}

	// Check <slug>/theme.zip (for relative directory reference)
	zipPath = filepath.Join(baseDir, slug, "theme.zip")
	if fileExistsAndIsFile(zipPath) {
		result.ZipPath = zipPath
		result.IsLocal = true
		return result
	}

	// Check <slug>.zip (for relative file reference)
	zipPath = filepath.Join(baseDir, slug+".zip")
	if fileExistsAndIsFile(zipPath) {
		result.ZipPath = zipPath
		result.IsLocal = true
		return result
	}

	// Check themes/<slug>/theme.properties (needs build)
	propsDir := filepath.Join(baseDir, "themes", slug)
	if ThemeExists(propsDir) {
		result.BuildDir = propsDir
		result.NeedsBuild = true
		result.IsLocal = true
		return result
	}

	// Check <slug>/theme.properties (for relative directory reference)
	propsDir = filepath.Join(baseDir, slug)
	if ThemeExists(propsDir) {
		result.BuildDir = propsDir
		result.NeedsBuild = true
		result.IsLocal = true
		return result
	}

	// No local resolution found - treat as WordPress.org slug
	result.IsLocal = false
	return result
}

// fileExistsAndIsFile checks if a file exists at the given path and is not a directory
func fileExistsAndIsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
