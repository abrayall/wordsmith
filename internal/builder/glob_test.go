package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandGlob(t *testing.T) {
	// Create temp directory structure for testing
	tmpDir, err := os.MkdirTemp("", "glob_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := []string{
		"file1.php",
		"file2.php",
		"file.js",
		"style.css",
		"src/app.php",
		"src/util.php",
		"src/lib/helper.php",
		"assets/script.js",
		"assets/style.css",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("test"), 0644)
	}

	tests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{"single wildcard php", "*.php", 2},
		{"single wildcard js", "*.js", 1},
		{"all css files", "*.css", 1},
		{"directory", "src", 5},           // includes dir entries
		{"recursive php", "**/*.php", 5},
		{"recursive js", "**/*.js", 2},
		{"specific file", "file1.php", 1},
		{"subdirectory wildcard", "src/*.php", 2},
		{"assets directory", "assets", 3}, // includes dir entry
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExpandGlob(tmpDir, tt.pattern)
			if err != nil {
				t.Errorf("ExpandGlob(%q) error = %v", tt.pattern, err)
				return
			}
			if len(results) != tt.expected {
				t.Errorf("ExpandGlob(%q) = %d files, want %d. Got: %v", tt.pattern, len(results), tt.expected, results)
			}
		})
	}
}

func TestContainsGlobChars(t *testing.T) {
	tests := []struct {
		pattern  string
		expected bool
	}{
		{"*.php", true},
		{"file?.txt", true},
		{"[abc].txt", true},
		{"file.txt", false},
		{"src/file.php", false},
		{"**/*.php", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := containsGlobChars(tt.pattern)
			if result != tt.expected {
				t.Errorf("containsGlobChars(%q) = %v, want %v", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		excludes []string
		expected bool
	}{
		{"no excludes", "file.php", []string{}, false},
		{"exact match", "file.php", []string{"file.php"}, true},
		{"wildcard match", "file.php", []string{"*.php"}, true},
		{"no match", "file.php", []string{"*.js"}, false},
		{"directory exclude", "build/file.php", []string{"build/*"}, true},
		{"recursive exclude", "src/lib/file.php", []string{"**/*.php"}, true},
		{"multiple excludes match", "file.php", []string{"*.js", "*.php"}, true},
		{"multiple excludes no match", "file.txt", []string{"*.js", "*.php"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExcluded(tt.path, tt.excludes)
			if result != tt.expected {
				t.Errorf("IsExcluded(%q, %v) = %v, want %v", tt.path, tt.excludes, result, tt.expected)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{"exact match", "file.php", "file.php", true},
		{"wildcard extension", "file.php", "*.php", true},
		{"wildcard name", "file.php", "file.*", true},
		{"no match", "file.php", "*.js", false},
		{"recursive pattern", "src/lib/file.php", "**/*.php", true},
		{"path with directory", "src/file.php", "src/*.php", true},
		{"directory prefix", "build/output.zip", "build/*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestExpandIncludes(t *testing.T) {
	// Create temp directory structure for testing
	tmpDir, err := os.MkdirTemp("", "expand_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := []string{
		"file1.php",
		"file2.php",
		"file.js",
		"vendor/lib.php",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("test"), 0644)
	}

	tests := []struct {
		name     string
		includes []string
		excludes []string
		expected int
	}{
		{"all php", []string{"*.php"}, []string{}, 2},
		{"all php exclude vendor", []string{"**/*.php"}, []string{"vendor/*"}, 2},
		{"multiple patterns", []string{"*.php", "*.js"}, []string{}, 3},
		{"with exclusion", []string{"*.php", "*.js"}, []string{"*.js"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExpandIncludes(tmpDir, tt.includes, tt.excludes)
			if err != nil {
				t.Errorf("ExpandIncludes() error = %v", err)
				return
			}
			if len(results) != tt.expected {
				t.Errorf("ExpandIncludes() = %d files, want %d. Got: %v", len(results), tt.expected, results)
			}
		})
	}
}
