package config

import (
	"path/filepath"
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
