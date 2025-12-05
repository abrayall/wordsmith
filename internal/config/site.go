package config

import (
	"os"
	"path/filepath"
	"strings"
)

// SiteConfig represents the site.properties configuration
type SiteConfig struct {
	Name        string // Site name
	Description string // Site description
	URL         string // Site URL

	// WordPress configuration (same as WordPressConfig)
	Image   string            // Docker image (defaults to "wordpress:latest")
	Plugins []WordPressPlugin // Plugins from site.properties
	Themes  []WordPressTheme  // Themes from site.properties

	// Discovered plugins and themes from directories
	LocalPlugins []LocalPlugin // Plugins discovered in plugins/ directory
	LocalThemes  []LocalTheme  // Themes discovered in themes/ directory
}

// LocalPlugin represents a plugin found in the plugins/ directory
type LocalPlugin struct {
	Slug       string // Plugin slug (directory name or zip filename without extension)
	Path       string // Full path to the plugin (directory or zip file)
	IsZip      bool   // True if this is a zip file
	NeedsBuild bool   // True if this is a source directory with plugin.properties
	Active     bool   // Whether to activate (defaults to true)
}

// LocalTheme represents a theme found in the themes/ directory
type LocalTheme struct {
	Slug       string // Theme slug (directory name or zip filename without extension)
	Path       string // Full path to the theme (directory or zip file)
	IsZip      bool   // True if this is a zip file
	NeedsBuild bool   // True if this is a source directory with theme.properties
	Active     bool   // Whether to activate (defaults to true for first theme)
}

// LoadSiteProperties loads site configuration from site.properties file
func LoadSiteProperties(dir string) (*SiteConfig, error) {
	path := filepath.Join(dir, "site.properties")
	props, err := ParseProperties(path)
	if err != nil {
		return nil, err
	}

	config := &SiteConfig{
		Name:        props.Get("name"),
		Description: props.Get("description"),
		URL:         props.Get("url"),
		Image:       props.GetWithDefault("image", "wordpress:latest"),
	}

	// Parse plugins from site.properties
	pluginsVal, ok := props["plugins"]
	if ok {
		config.Plugins = parsePluginsList(pluginsVal)
	}

	// Parse themes from site.properties
	themesVal, ok := props["themes"]
	if ok {
		config.Themes = parseThemesList(themesVal)
	}

	// Discover local plugins in plugins/ directory
	config.LocalPlugins = discoverLocalPlugins(dir)

	// Discover local themes in themes/ directory
	config.LocalThemes = discoverLocalThemes(dir)

	return config, nil
}

// discoverLocalPlugins scans the plugins/ directory for plugins
func discoverLocalPlugins(dir string) []LocalPlugin {
	var plugins []LocalPlugin
	pluginsDir := filepath.Join(dir, "plugins")

	// Check if plugins directory exists
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		return plugins
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return plugins
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(pluginsDir, name)

		if entry.IsDir() {
			// Check if directory contains plugin.properties
			if PluginExists(fullPath) {
				plugins = append(plugins, LocalPlugin{
					Slug:       name,
					Path:       fullPath,
					IsZip:      false,
					NeedsBuild: true,
					Active:     true,
				})
			} else {
				// Check for zip files inside the directory
				subEntries, err := os.ReadDir(fullPath)
				if err == nil {
					for _, subEntry := range subEntries {
						if strings.HasSuffix(subEntry.Name(), ".zip") {
							zipPath := filepath.Join(fullPath, subEntry.Name())
							slug := strings.TrimSuffix(subEntry.Name(), ".zip")
							plugins = append(plugins, LocalPlugin{
								Slug:       slug,
								Path:       zipPath,
								IsZip:      true,
								NeedsBuild: false,
								Active:     true,
							})
						}
					}
				}
			}
		} else if strings.HasSuffix(name, ".zip") {
			// Zip file directly in plugins/
			slug := strings.TrimSuffix(name, ".zip")
			plugins = append(plugins, LocalPlugin{
				Slug:       slug,
				Path:       fullPath,
				IsZip:      true,
				NeedsBuild: false,
				Active:     true,
			})
		}
	}

	return plugins
}

// discoverLocalThemes scans the themes/ directory for themes
func discoverLocalThemes(dir string) []LocalTheme {
	var themes []LocalTheme
	themesDir := filepath.Join(dir, "themes")

	// Check if themes directory exists
	if _, err := os.Stat(themesDir); os.IsNotExist(err) {
		return themes
	}

	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return themes
	}

	isFirst := true
	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(themesDir, name)

		if entry.IsDir() {
			// Check if directory contains theme.properties
			if ThemeExists(fullPath) {
				themes = append(themes, LocalTheme{
					Slug:       name,
					Path:       fullPath,
					IsZip:      false,
					NeedsBuild: true,
					Active:     isFirst,
				})
				isFirst = false
			} else {
				// Check for zip files inside the directory
				subEntries, err := os.ReadDir(fullPath)
				if err == nil {
					for _, subEntry := range subEntries {
						if strings.HasSuffix(subEntry.Name(), ".zip") {
							zipPath := filepath.Join(fullPath, subEntry.Name())
							slug := strings.TrimSuffix(subEntry.Name(), ".zip")
							themes = append(themes, LocalTheme{
								Slug:       slug,
								Path:       zipPath,
								IsZip:      true,
								NeedsBuild: false,
								Active:     isFirst,
							})
							isFirst = false
						}
					}
				}
			}
		} else if strings.HasSuffix(name, ".zip") {
			// Zip file directly in themes/
			slug := strings.TrimSuffix(name, ".zip")
			themes = append(themes, LocalTheme{
				Slug:       slug,
				Path:       fullPath,
				IsZip:      true,
				NeedsBuild: false,
				Active:     isFirst,
			})
			isFirst = false
		}
	}

	return themes
}

// SiteExists checks if site.properties exists in the directory
func SiteExists(dir string) bool {
	return PropertiesFileExists(dir, "site.properties")
}

// ToWordPressConfig converts a SiteConfig to a WordPressConfig
// This merges local plugins/themes with those from site.properties
func (s *SiteConfig) ToWordPressConfig() *WordPressConfig {
	wpConfig := &WordPressConfig{
		Name:    s.Name,
		Image:   s.Image,
		Plugins: make([]WordPressPlugin, 0),
		Themes:  make([]WordPressTheme, 0),
	}

	// Add local plugins first (they take precedence)
	for _, lp := range s.LocalPlugins {
		wpConfig.Plugins = append(wpConfig.Plugins, WordPressPlugin{
			Slug:   lp.Slug,
			URI:    lp.Path,
			Active: lp.Active,
		})
	}

	// Add plugins from site.properties
	wpConfig.Plugins = append(wpConfig.Plugins, s.Plugins...)

	// Add local themes first (they take precedence)
	for _, lt := range s.LocalThemes {
		wpConfig.Themes = append(wpConfig.Themes, WordPressTheme{
			Slug:   lt.Slug,
			URI:    lt.Path,
			Active: lt.Active,
		})
	}

	// Add themes from site.properties
	wpConfig.Themes = append(wpConfig.Themes, s.Themes...)

	return wpConfig
}

// GetAllPlugins returns all plugins (local + from properties) as a unified list
func (s *SiteConfig) GetAllPlugins() []WordPressPlugin {
	return s.ToWordPressConfig().Plugins
}

// GetAllThemes returns all themes (local + from properties) as a unified list
func (s *SiteConfig) GetAllThemes() []WordPressTheme {
	return s.ToWordPressConfig().Themes
}
