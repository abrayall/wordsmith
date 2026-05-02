package config

import (
	"fmt"
	"path/filepath"
)

// LibraryConfig represents the library.properties configuration
type LibraryConfig struct {
	Name    string
	Slug    string
	Version string

	// Additional files/directories to include (supports wildcards)
	Include []string

	// Files/directories to exclude (supports wildcards)
	Exclude []string

	// Libraries to include in the build
	Libraries []LibrarySpec
}

// LoadLibraryProperties loads library configuration from library.properties file
func LoadLibraryProperties(dir string) (*LibraryConfig, error) {
	path := filepath.Join(dir, "library.properties")
	props, err := ParseProperties(path)
	if err != nil {
		return nil, err
	}

	config := &LibraryConfig{
		Name:      props.Get("name"),
		Slug:      props.Get("slug"),
		Version:   props.Get("version"),
		Include:   props.GetList("include"),
		Exclude:   props.GetList("exclude"),
		Libraries: ParseLibraries(props),
	}

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("missing required field: name")
	}

	return config, nil
}

// LibraryExists checks if library.properties exists in the directory
func LibraryExists(dir string) bool {
	return PropertiesFileExists(dir, "library.properties")
}
