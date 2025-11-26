package config

import (
	"fmt"
	"path/filepath"
)

// ThemeConfig represents the theme.properties configuration
type ThemeConfig struct {
	Name        string
	Slug        string
	Version     string
	Description string
	Author      string
	AuthorURI   string
	ThemeURI    string
	License     string
	LicenseURI  string
	Main        string // Main stylesheet (style.css)
	Template    string // Parent theme for child themes
	TemplateURI string // URL or path to parent theme
	TextDomain  string
	DomainPath  string
	Requires    string
	RequiresPHP string
	Tags        string

	// Additional files/directories to include (supports wildcards: *.php, **/*.php)
	Include []string

	// Files/directories to exclude (supports wildcards)
	Exclude []string

	// Minify CSS/JS files
	Minify bool
}

// LoadThemeProperties loads theme configuration from theme.properties file
func LoadThemeProperties(dir string) (*ThemeConfig, error) {
	path := filepath.Join(dir, "theme.properties")
	props, err := ParseProperties(path)
	if err != nil {
		return nil, err
	}

	config := &ThemeConfig{
		Name:        props.Get("name"),
		Slug:        props.Get("slug"),
		Version:     props.Get("version"),
		Description: props.Get("description"),
		Author:      props.Get("author"),
		AuthorURI:   props.Get("author-uri"),
		ThemeURI:    props.Get("theme-uri"),
		License:     props.Get("license"),
		LicenseURI:  props.Get("license-uri"),
		Main:        props.GetWithDefault("main", "style.css"),
		Template:    props.Get("template"),
		TemplateURI: props.Get("template-uri"),
		TextDomain:  props.Get("text-domain"),
		DomainPath:  props.Get("domain-path"),
		Requires:    props.Get("requires"),
		RequiresPHP: props.Get("requires-php"),
		Tags:        props.Get("tags"),
		Include:     props.GetList("include"),
		Exclude:     props.GetList("exclude"),
		Minify:      props.GetBool("minify"),
	}

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("missing required field: name")
	}

	return config, nil
}

// ThemeExists checks if theme.properties exists in the directory
func ThemeExists(dir string) bool {
	return PropertiesFileExists(dir, "theme.properties")
}
