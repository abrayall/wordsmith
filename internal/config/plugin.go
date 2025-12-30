package config

import (
	"fmt"
	"path/filepath"
)

// PluginConfig represents the plugin.properties configuration
type PluginConfig struct {
	Name        string
	Slug        string
	Version     string
	Description string
	Author      string
	AuthorURI   string
	PluginURI   string
	License     string
	LicenseURI  string
	Main        string
	TextDomain  string
	DomainPath  string
	Requires    string
	RequiresPHP string

	// Additional files/directories to include (supports wildcards: *.php, **/*.php)
	Include []string

	// Files/directories to exclude (supports wildcards)
	Exclude []string

	// Libraries to include in the build
	Libraries []LibrarySpec

	// Plugin dependencies (other plugins this plugin requires)
	Plugins []LibrarySpec

	// Obfuscate PHP files
	Obfuscate bool

	// Minify CSS/JS files
	Minify bool

	// Settings to deploy to WordPress database
	Settings map[string]interface{}
}

// LoadPluginProperties loads plugin configuration from plugin.properties file
func LoadPluginProperties(dir string) (*PluginConfig, error) {
	path := filepath.Join(dir, "plugin.properties")
	props, err := ParseProperties(path)
	if err != nil {
		return nil, err
	}

	config := &PluginConfig{
		Name:        props.Get("name"),
		Slug:        props.Get("slug"),
		Version:     props.Get("version"),
		Description: props.Get("description"),
		Author:      props.Get("author"),
		AuthorURI:   props.Get("author-uri"),
		PluginURI:   props.Get("plugin-uri"),
		License:     props.Get("license"),
		LicenseURI:  props.Get("license-uri"),
		Main:        props.Get("main"),
		TextDomain:  props.Get("text-domain"),
		DomainPath:  props.Get("domain-path"),
		Requires:    props.Get("requires"),
		RequiresPHP: props.Get("requires-php"),
		Include:     props.GetList("include"),
		Exclude:     props.GetList("exclude"),
		Libraries:   ParseLibraries(props),
		Plugins:     ParsePlugins(props),
		Obfuscate:   props.GetBool("obfuscate"),
		Minify:      props.GetBool("minify"),
		Settings:    ParseSettings(props),
	}

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("missing required field: name")
	}
	if config.Main == "" {
		return nil, fmt.Errorf("missing required field: main")
	}

	return config, nil
}

// PluginExists checks if plugin.properties exists in the directory
func PluginExists(dir string) bool {
	return PropertiesFileExists(dir, "plugin.properties")
}
