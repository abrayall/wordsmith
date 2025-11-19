package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ThemeConfig represents the theme.properties configuration
type ThemeConfig struct {
	Name        string
	Version     string
	Description string
	Author      string
	AuthorURI   string
	ThemeURI    string
	License     string
	LicenseURI  string
	Main        string // Main stylesheet (style.css)
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
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open theme.properties: %w", err)
	}
	defer file.Close()

	config := &ThemeConfig{
		Include: []string{},
		Exclude: []string{},
		Minify:  false,
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "name":
			config.Name = value
		case "version":
			config.Version = value
		case "description":
			config.Description = value
		case "author":
			config.Author = value
		case "author-uri":
			config.AuthorURI = value
		case "theme-uri":
			config.ThemeURI = value
		case "license":
			config.License = value
		case "license-uri":
			config.LicenseURI = value
		case "main":
			config.Main = value
		case "text-domain":
			config.TextDomain = value
		case "domain-path":
			config.DomainPath = value
		case "requires":
			config.Requires = value
		case "requires-php":
			config.RequiresPHP = value
		case "tags":
			config.Tags = value
		case "include":
			// Parse comma-separated list
			items := strings.Split(value, ",")
			for _, item := range items {
				item = strings.TrimSpace(item)
				if item != "" {
					config.Include = append(config.Include, item)
				}
			}
		case "exclude":
			// Parse comma-separated list
			items := strings.Split(value, ",")
			for _, item := range items {
				item = strings.TrimSpace(item)
				if item != "" {
					config.Exclude = append(config.Exclude, item)
				}
			}
		case "minify":
			config.Minify = !(value == "false" || value == "no" || value == "0")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading theme.properties: %w", err)
	}

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("missing required field: name")
	}
	if config.Main == "" {
		config.Main = "style.css" // Default for themes
	}

	return config, nil
}

// ThemeExists checks if theme.properties exists in the directory
func ThemeExists(dir string) bool {
	path := filepath.Join(dir, "theme.properties")
	_, err := os.Stat(path)
	return err == nil
}
