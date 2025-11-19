package builder

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandGlob expands a glob pattern relative to baseDir, supporting ** for recursive matching
func ExpandGlob(baseDir, pattern string) ([]string, error) {
	var results []string

	// Check if pattern contains **
	if strings.Contains(pattern, "**") {
		// Handle ** recursive glob
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], string(filepath.Separator))
			suffix := strings.TrimPrefix(parts[1], string(filepath.Separator))

			startDir := baseDir
			if prefix != "" {
				startDir = filepath.Join(baseDir, prefix)
			}

			err := filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip errors
				}

				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					return nil
				}

				// If there's a suffix pattern, match against it
				if suffix != "" {
					// Match the filename or path suffix
					matched, _ := filepath.Match(suffix, info.Name())
					if !matched {
						// Try matching against the relative path from the ** point
						relFromStart, _ := filepath.Rel(startDir, path)
						matched, _ = filepath.Match(suffix, relFromStart)
					}
					if !matched {
						return nil
					}
				}

				if !info.IsDir() || suffix == "" {
					results = append(results, relPath)
				}

				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Standard glob without **
		fullPattern := filepath.Join(baseDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}

			relPath, err := filepath.Rel(baseDir, match)
			if err != nil {
				continue
			}

			if info.IsDir() {
				// If it's a directory, recursively include all contents
				filepath.Walk(match, func(path string, fi os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					rel, err := filepath.Rel(baseDir, path)
					if err != nil {
						return nil
					}
					results = append(results, rel)
					return nil
				})
			} else {
				results = append(results, relPath)
			}
		}

		// If no matches and no glob chars, try as direct path
		if len(results) == 0 && !containsGlobChars(pattern) {
			fullPath := filepath.Join(baseDir, pattern)
			info, err := os.Stat(fullPath)
			if err == nil {
				if info.IsDir() {
					// If it's a directory, recursively include all contents
					filepath.Walk(fullPath, func(path string, fi os.FileInfo, err error) error {
						if err != nil {
							return nil
						}
						relPath, err := filepath.Rel(baseDir, path)
						if err != nil {
							return nil
						}
						results = append(results, relPath)
						return nil
					})
				} else {
					results = append(results, pattern)
				}
			}
		}
	}

	return results, nil
}

// containsGlobChars checks if a pattern contains glob special characters
func containsGlobChars(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// IsExcluded checks if a path matches any of the exclude patterns
func IsExcluded(path string, excludes []string) bool {
	for _, pattern := range excludes {
		if matchPattern(path, pattern) {
			return true
		}
	}
	return false
}

// matchPattern checks if a path matches a pattern (supports * and **)
func matchPattern(path, pattern string) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)
	pattern = filepath.ToSlash(pattern)

	// Handle ** recursive matching
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")

			// Check prefix
			if prefix != "" && !strings.HasPrefix(path, prefix) {
				// Try matching prefix as glob
				matched, _ := filepath.Match(prefix+"*", path)
				if !matched {
					return false
				}
			}

			// Check suffix
			if suffix != "" {
				// Match against filename
				matched, _ := filepath.Match(suffix, filepath.Base(path))
				if matched {
					return true
				}
				// Match against path suffix
				if strings.HasSuffix(path, suffix) {
					return true
				}
				// Try glob match on the suffix
				matched, _ = filepath.Match("*"+suffix, path)
				if matched {
					return true
				}
			} else {
				return true
			}
		}
		return false
	}

	// Standard glob matching
	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}

	// Also try matching against just the filename
	matched, _ = filepath.Match(pattern, filepath.Base(path))
	return matched
}

// ExpandIncludes expands all include patterns and returns unique file paths
func ExpandIncludes(baseDir string, includes []string, excludes []string) ([]string, error) {
	seen := make(map[string]bool)
	var results []string

	for _, pattern := range includes {
		expanded, err := ExpandGlob(baseDir, pattern)
		if err != nil {
			continue
		}

		for _, path := range expanded {
			// Skip if excluded
			if IsExcluded(path, excludes) {
				continue
			}

			// Skip if already seen
			if seen[path] {
				continue
			}

			seen[path] = true
			results = append(results, path)
		}
	}

	return results, nil
}
