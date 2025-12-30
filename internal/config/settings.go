package config

import (
	"regexp"
	"strings"
)

// ParseSettings parses the settings list from properties into a map of option names to values.
// Supports both simple key=value and bracket notation key[sub][sub2]=value.
// Bracket notation entries are grouped by base option name into nested maps.
func ParseSettings(props Properties) map[string]interface{} {
	settings := make(map[string]interface{})

	// Get settings as a list
	settingsList := props.GetList("settings")
	if len(settingsList) == 0 {
		return settings
	}

	for _, item := range settingsList {
		// Parse key=value
		eqIdx := strings.Index(item, "=")
		if eqIdx == -1 {
			continue
		}

		key := strings.TrimSpace(item[:eqIdx])
		value := strings.TrimSpace(item[eqIdx+1:])

		// Check if key has bracket notation
		if strings.Contains(key, "[") {
			parseAndSetBracketNotation(settings, key, value)
		} else {
			// Simple key=value
			settings[key] = value
		}
	}

	return settings
}

// parseAndSetBracketNotation parses a bracket notation key like "option[key1][key2]"
// and sets the value in the nested map structure.
func parseAndSetBracketNotation(settings map[string]interface{}, key, value string) {
	// Extract base option name and path
	bracketIdx := strings.Index(key, "[")
	baseName := key[:bracketIdx]
	pathPart := key[bracketIdx:]

	// Parse bracket path into slice of keys
	path := parseBracketPath(pathPart)
	if len(path) == 0 {
		return
	}

	// Get or create the base option map
	var baseMap map[string]interface{}
	if existing, ok := settings[baseName]; ok {
		if m, ok := existing.(map[string]interface{}); ok {
			baseMap = m
		} else {
			// Base option exists but is not a map - skip
			return
		}
	} else {
		baseMap = make(map[string]interface{})
		settings[baseName] = baseMap
	}

	// Navigate/create the nested path and set the value
	setNestedValue(baseMap, path, value)
}

// parseBracketPath extracts keys from bracket notation like "[key1][key2]"
// Returns slice of keys: ["key1", "key2"]
func parseBracketPath(pathPart string) []string {
	re := regexp.MustCompile(`\[([^\]]+)\]`)
	matches := re.FindAllStringSubmatch(pathPart, -1)

	path := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			path = append(path, match[1])
		}
	}
	return path
}

// setNestedValue sets a value at a nested path in a map.
// Creates intermediate maps as needed.
func setNestedValue(m map[string]interface{}, path []string, value string) {
	if len(path) == 0 {
		return
	}

	// Navigate to the parent of the final key
	current := m
	for i := 0; i < len(path)-1; i++ {
		key := path[i]
		if existing, ok := current[key]; ok {
			if nextMap, ok := existing.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// Path exists but is not a map - create new map
				newMap := make(map[string]interface{})
				current[key] = newMap
				current = newMap
			}
		} else {
			// Key doesn't exist - create new map
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		}
	}

	// Set the final value
	finalKey := path[len(path)-1]
	current[finalKey] = value
}
