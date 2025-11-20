package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Properties represents a parsed properties/YAML file as a map
type Properties map[string]string

// ParseProperties parses a properties file supporting both = and : delimiters
func ParseProperties(path string) (Properties, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	props := make(Properties)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value or key: value (YAML-style)
		var parts []string
		if strings.Contains(line, "=") {
			parts = strings.SplitN(line, "=", 2)
		} else if strings.Contains(line, ":") {
			parts = strings.SplitN(line, ":", 2)
		}
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		props[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return props, nil
}

// Get returns the value for a key, or empty string if not found
func (p Properties) Get(key string) string {
	return p[key]
}

// GetWithDefault returns the value for a key, or the default if not found
func (p Properties) GetWithDefault(key, defaultValue string) string {
	if val, ok := p[key]; ok && val != "" {
		return val
	}
	return defaultValue
}

// GetBool returns true unless the value is "false", "no", or "0"
func (p Properties) GetBool(key string) bool {
	val := p[key]
	if val == "" {
		return false
	}
	return !(val == "false" || val == "no" || val == "0")
}

// GetList parses a comma-separated value into a slice
func (p Properties) GetList(key string) []string {
	val := p[key]
	if val == "" {
		return []string{}
	}

	var result []string
	items := strings.Split(val, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// FileExists checks if a file exists at the given path
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// PropertiesFileExists checks if a properties file exists in the directory
func PropertiesFileExists(dir, filename string) bool {
	path := filepath.Join(dir, filename)
	return FileExists(path)
}
