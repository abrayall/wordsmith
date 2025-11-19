package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PluginConfig represents the plugin.properties configuration
type PluginConfig struct {
	Name        string
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

	// Additional files/directories to include
	Include []string

	// Obfuscate PHP files
	Obfuscate bool

	// Minify CSS/JS files
	Minify bool
}

// LoadProperties loads plugin configuration from plugin.properties file
func LoadProperties(dir string) (*PluginConfig, error) {
	path := filepath.Join(dir, "plugin.properties")
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin.properties: %w", err)
	}
	defer file.Close()

	config := &PluginConfig{
		Include:   []string{},
		Obfuscate: false,
		Minify:    false,
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
		case "plugin-uri":
			config.PluginURI = value
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
		case "include":
			// Parse comma-separated list
			items := strings.Split(value, ",")
			for _, item := range items {
				item = strings.TrimSpace(item)
				if item != "" {
					config.Include = append(config.Include, item)
				}
			}
		case "obfuscate":
			config.Obfuscate = !(value == "false" || value == "no" || value == "0")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading plugin.properties: %w", err)
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

// Exists checks if plugin.properties exists in the directory
func Exists(dir string) bool {
	path := filepath.Join(dir, "plugin.properties")
	_, err := os.Stat(path)
	return err == nil
}
