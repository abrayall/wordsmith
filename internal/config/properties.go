package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Properties represents a parsed properties/YAML file as a map
type Properties map[string]interface{}

// ParseProperties parses a properties file, converting = to : and using YAML parser
func ParseProperties(path string) (Properties, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	// Read and convert = to : for YAML compatibility
	var yamlContent strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			yamlContent.WriteString(line)
			yamlContent.WriteString("\n")
			continue
		}

		// Convert = to : if line contains = (and it's not in a value)
		if strings.Contains(line, "=") && !strings.HasPrefix(trimmed, "-") {
			// Find the first = and replace with :
			// But only if it's before any : (to handle URLs like http://...)
			eqIdx := strings.Index(line, "=")
			colonIdx := strings.Index(line, ":")

			if colonIdx == -1 || eqIdx < colonIdx {
				// Replace = with : and ensure there's a space after
				key := line[:eqIdx]
				value := strings.TrimSpace(line[eqIdx+1:])

				// Quote the value if it contains YAML special characters
				if value != "" && needsQuoting(value) {
					value = "\"" + strings.ReplaceAll(value, "\"", "\\\"") + "\""
				}

				line = key + ": " + value
			}
		} else if strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "-") {
			// Also check YAML-style lines for special characters
			colonIdx := strings.Index(line, ":")
			if colonIdx > 0 {
				key := line[:colonIdx]
				value := strings.TrimSpace(line[colonIdx+1:])

				// Quote the value if it contains YAML special characters and isn't already quoted
				if value != "" && !strings.HasPrefix(value, "\"") && !strings.HasPrefix(value, "'") && needsQuoting(value) {
					value = "\"" + strings.ReplaceAll(value, "\"", "\\\"") + "\""
					line = key + ": " + value
				}
			}
		}

		yamlContent.WriteString(line)
		yamlContent.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	// Parse as YAML
	props := make(Properties)
	if err := yaml.Unmarshal([]byte(yamlContent.String()), &props); err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", path, err)
	}

	return props, nil
}

// Get returns the string value for a key, or empty string if not found
func (p Properties) Get(key string) string {
	val, ok := p[key]
	if !ok || val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		// Check if it's a whole number
		if v == float64(int(v)) {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetWithDefault returns the value for a key, or the default if not found
func (p Properties) GetWithDefault(key, defaultValue string) string {
	val := p.Get(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetBool returns the boolean value for a key
func (p Properties) GetBool(key string) bool {
	val, ok := p[key]
	if !ok {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case string:
		return !(v == "" || v == "false" || v == "no" || v == "0")
	default:
		return true
	}
}

// GetList returns a slice of strings for a key
// Supports both comma-separated strings and YAML lists
func (p Properties) GetList(key string) []string {
	val, ok := p[key]
	if !ok {
		return []string{}
	}

	switch v := val.(type) {
	case []interface{}:
		// YAML list
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			} else {
				result = append(result, fmt.Sprintf("%v", item))
			}
		}
		return result
	case string:
		// Comma-separated string
		if v == "" {
			return []string{}
		}
		var result []string
		items := strings.Split(v, ",")
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				result = append(result, item)
			}
		}
		return result
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}

// needsQuoting checks if a value contains YAML special characters that need quoting
func needsQuoting(value string) bool {
	return strings.ContainsAny(value, "*[]{}|>&!%@`#") ||
		strings.HasPrefix(value, "-") ||
		strings.HasPrefix(value, "?")
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
