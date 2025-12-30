package builder

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

// ThemeBuilder builds WordPress themes
type ThemeBuilder struct {
	BaseBuilder
	Config *config.ThemeConfig
}

// NewThemeBuilder creates a new theme Builder
func NewThemeBuilder(sourceDir string) *ThemeBuilder {
	return &ThemeBuilder{
		BaseBuilder: NewBaseBuilder(sourceDir),
	}
}

// Build builds the theme
func (b *ThemeBuilder) Build() error {
	if !b.Quiet {
		ui.PrintInfo("Loading theme.properties...")
	}
	cfg, err := config.LoadThemeProperties(b.SourceDir)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	b.Config = cfg

	// Parse version
	if cfg.Version != "" {
		b.Version = ParseVersion(cfg.Version)
	} else {
		ver, err := b.GetVersionFromGit()
		if err != nil {
			return err
		}
		b.Version = ver
	}

	b.PrintBuildInfo(b.Config.Name)

	if err := b.CleanBuildDir(); err != nil {
		return err
	}

	stageDir, err := b.CreateStageDir()
	if err != nil {
		return err
	}

	themeName := b.GetThemeSlug()

	if !b.Quiet {
		ui.PrintInfo("Copying theme files...")
	}

	// Copy main stylesheet
	mainFile := filepath.Base(b.Config.Main)
	mainSrc := filepath.Join(b.SourceDir, b.Config.Main)
	mainDst := filepath.Join(stageDir, mainFile)

	if err := CopyFile(mainSrc, mainDst); err != nil {
		return fmt.Errorf("failed to copy main stylesheet: %w", err)
	}

	// Expand glob patterns in includes
	expandedIncludes, err := ExpandIncludes(b.SourceDir, b.Config.Include, b.Config.Exclude)
	if err != nil {
		return fmt.Errorf("failed to expand include patterns: %w", err)
	}

	// Copy included files/directories
	for _, include := range expandedIncludes {
		src := filepath.Join(b.SourceDir, include)
		info, err := os.Stat(src)
		if err != nil {
			ui.PrintWarning("Skipping %s: %v", include, err)
			continue
		}

		if info.IsDir() {
			if err := b.copyDirWithExcludes(src, filepath.Join(stageDir, include), b.Config.Exclude); err != nil {
				return fmt.Errorf("failed to copy directory %s: %w", include, err)
			}
		} else {
			dst := filepath.Join(stageDir, include)
			if b.Config.Minify && (strings.HasSuffix(include, ".css") || strings.HasSuffix(include, ".js")) {
				if err := CopyAndMinify(src, dst, true); err != nil {
					return fmt.Errorf("failed to minify file %s: %w", include, err)
				}
			} else {
				if err := CopyFile(src, dst); err != nil {
					return fmt.Errorf("failed to copy file %s: %w", include, err)
				}
			}
		}
	}

	// Generate theme header in style.css
	if !b.Quiet {
		ui.PrintInfo("Generating theme header...")
	}
	if err := b.generateThemeHeader(mainDst); err != nil {
		return fmt.Errorf("failed to generate theme header: %w", err)
	}

	// Write version.properties
	versionFile := filepath.Join(stageDir, "version.properties")
	if err := WriteVersionProperties(versionFile, b.Config.Name, b.Version); err != nil {
		return fmt.Errorf("failed to write version.properties: %w", err)
	}

	// Write theme.properties
	themePropsFile := filepath.Join(stageDir, "theme.properties")
	if err := b.writeThemeProperties(themePropsFile); err != nil {
		return fmt.Errorf("failed to write theme.properties: %w", err)
	}

	// Copy libraries to stage directory
	if len(b.Config.Libraries) > 0 {
		if !b.Quiet {
			ui.PrintInfo("Copying libraries...")
		}
		if err := CopyLibraries(b.Config.Libraries, stageDir, b.Quiet); err != nil {
			return fmt.Errorf("failed to copy libraries: %w", err)
		}
	}

	// Fetch parent theme if template-uri is specified
	if b.Config.TemplateURI != "" {
		if !b.Quiet {
			ui.PrintInfo("Fetching parent theme...")
		}
		if err := b.fetchParentTheme(); err != nil {
			return fmt.Errorf("failed to fetch parent theme: %w", err)
		}

		// Update child theme's functions.php with parent style dependencies
		if err := b.updateChildStyleDependencies(stageDir); err != nil {
			ui.PrintWarning("Could not update style dependencies: %v", err)
		}
	}

	// Clean dev files
	CleanDevFiles(stageDir)

	// Set permissions on all files before zipping
	if err := ChmodAll(stageDir, 0777); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Create ZIP
	if !b.Quiet {
		ui.PrintInfo("Creating ZIP archive...")
	}
	zipPath := filepath.Join(b.BuildDir, fmt.Sprintf("%s-%s.zip", themeName, b.Version.String()))
	if err := CreateZip(stageDir, zipPath, themeName); err != nil {
		return fmt.Errorf("failed to create ZIP: %w", err)
	}

	if !b.Quiet {
		fmt.Println()
		ui.PrintSuccess("Created: %s", filepath.Base(zipPath))
	}

	return nil
}

// GetThemeSlug returns the WordPress theme slug (directory name) for this theme.
func (b *ThemeBuilder) GetThemeSlug() string {
	if b.Config == nil {
		return ""
	}
	if b.Config.Slug != "" {
		return b.Config.Slug
	}
	return SanitizeName(b.Config.Name)
}

func (b *ThemeBuilder) copyDirWithExcludes(src, dst string, excludes []string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Check if excluded
		if IsExcluded(relPath, excludes) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		if b.Config.Minify && (strings.HasSuffix(info.Name(), ".css") || strings.HasSuffix(info.Name(), ".js")) {
			return CopyAndMinify(path, targetPath, true)
		}
		return CopyFile(path, targetPath)
	})
}

func (b *ThemeBuilder) generateThemeHeader(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Build theme header
	header := "/*\n"
	header += fmt.Sprintf("Theme Name: %s\n", b.Config.Name)

	if b.Config.ThemeURI != "" {
		header += fmt.Sprintf("Theme URI: %s\n", b.Config.ThemeURI)
	}
	if b.Config.Description != "" {
		header += fmt.Sprintf("Description: %s\n", b.Config.Description)
	}
	header += fmt.Sprintf("Version: %s\n", b.Version.String())
	if b.Config.Author != "" {
		header += fmt.Sprintf("Author: %s\n", b.Config.Author)
	}
	if b.Config.AuthorURI != "" {
		header += fmt.Sprintf("Author URI: %s\n", b.Config.AuthorURI)
	}
	if b.Config.Template != "" {
		header += fmt.Sprintf("Template: %s\n", b.Config.Template)
	}
	if b.Config.License != "" {
		header += fmt.Sprintf("License: %s\n", b.Config.License)
	}
	if b.Config.LicenseURI != "" {
		header += fmt.Sprintf("License URI: %s\n", b.Config.LicenseURI)
	}
	if b.Config.TextDomain != "" {
		header += fmt.Sprintf("Text Domain: %s\n", b.Config.TextDomain)
	}
	if b.Config.DomainPath != "" {
		header += fmt.Sprintf("Domain Path: %s\n", b.Config.DomainPath)
	}
	if b.Config.Tags != "" {
		header += fmt.Sprintf("Tags: %s\n", b.Config.Tags)
	}
	if b.Config.Requires != "" {
		header += fmt.Sprintf("Requires at least: %s\n", b.Config.Requires)
	}
	if b.Config.RequiresPHP != "" {
		header += fmt.Sprintf("Requires PHP: %s\n", b.Config.RequiresPHP)
	}
	header += "*/\n"

	contentStr := string(content)
	// Replace existing header if present
	re := regexp.MustCompile(`(?s)^/\*.*?\*/\s*`)
	updated := re.ReplaceAllString(contentStr, header)

	if updated == contentStr {
		// No existing header, prepend
		updated = header + "\n" + contentStr
	}

	return os.WriteFile(path, []byte(updated), 0644)
}

func (b *ThemeBuilder) writeThemeProperties(path string) error {
	var lines []string
	lines = append(lines, "# Theme metadata")
	lines = append(lines, "# Generated by wordsmith")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("name=%s", b.Config.Name))
	lines = append(lines, fmt.Sprintf("version=%s", b.Version.String()))

	if b.Config.Description != "" {
		lines = append(lines, fmt.Sprintf("description=%s", b.Config.Description))
	}
	if b.Config.Author != "" {
		lines = append(lines, fmt.Sprintf("author=%s", b.Config.Author))
	}
	if b.Config.AuthorURI != "" {
		lines = append(lines, fmt.Sprintf("author-uri=%s", b.Config.AuthorURI))
	}
	if b.Config.ThemeURI != "" {
		lines = append(lines, fmt.Sprintf("theme-uri=%s", b.Config.ThemeURI))
	}
	if b.Config.Template != "" {
		lines = append(lines, fmt.Sprintf("template=%s", b.Config.Template))
	}
	if b.Config.License != "" {
		lines = append(lines, fmt.Sprintf("license=%s", b.Config.License))
	}
	if b.Config.LicenseURI != "" {
		lines = append(lines, fmt.Sprintf("license-uri=%s", b.Config.LicenseURI))
	}
	if b.Config.TextDomain != "" {
		lines = append(lines, fmt.Sprintf("text-domain=%s", b.Config.TextDomain))
	}
	if b.Config.Tags != "" {
		lines = append(lines, fmt.Sprintf("tags=%s", b.Config.Tags))
	}
	if b.Config.Requires != "" {
		lines = append(lines, fmt.Sprintf("requires=%s", b.Config.Requires))
	}
	if b.Config.RequiresPHP != "" {
		lines = append(lines, fmt.Sprintf("requires-php=%s", b.Config.RequiresPHP))
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0644)
}

// GetStagePath returns the path to the staged theme directory
func (b *ThemeBuilder) GetStagePath() string {
	return filepath.Join(b.WorkDir, "stage")
}

// GetThemeName returns the sanitized theme name
func (b *ThemeBuilder) GetThemeName() string {
	return SanitizeName(b.Config.Name)
}

// GetParentThemePath returns the path to the parent theme directory, if it exists
func (b *ThemeBuilder) GetParentThemePath() string {
	parentDir := filepath.Join(b.WorkDir, "parent")
	if _, err := os.Stat(parentDir); err == nil {
		return parentDir
	}
	return ""
}

// GetAllParentThemes returns all parent themes in order (grandparent first, then parent, etc.)
func (b *ThemeBuilder) GetAllParentThemes() []struct {
	Name string
	Path string
} {
	var themes []struct {
		Name string
		Path string
	}

	// Walk the parent chain
	currentDir := b.WorkDir
	for {
		parentDir := filepath.Join(currentDir, "parent")
		if _, err := os.Stat(parentDir); err != nil {
			break
		}

		// Try to get the theme name from theme.properties
		themeName := ""
		propsPath := filepath.Join(parentDir, "theme.properties")
		if cfg, err := config.LoadThemeProperties(filepath.Dir(propsPath)); err == nil {
			themeName = cfg.Name
		} else {
			// Try to infer from style.css
			themeName = b.getThemeNameFromStyleCSS(parentDir)
		}

		if themeName == "" {
			break
		}

		themes = append([]struct {
			Name string
			Path string
		}{{Name: themeName, Path: parentDir}}, themes...)

		currentDir = parentDir
	}

	return themes
}

func (b *ThemeBuilder) getThemeNameFromStyleCSS(dir string) string {
	stylePath := filepath.Join(dir, "style.css")
	content, err := os.ReadFile(stylePath)
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`Theme Name:\s*(.+)`)
	matches := re.FindStringSubmatch(string(content))
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// GetParentThemeName returns the template (parent theme slug)
func (b *ThemeBuilder) GetParentThemeName() string {
	return b.Config.Template
}

// GetParentStyleHandles returns the stylesheet handles from the parent theme
func (b *ThemeBuilder) GetParentStyleHandles() []string {
	parentPath := b.GetParentThemePath()
	if parentPath == "" {
		return nil
	}

	functionsPath := filepath.Join(parentPath, "functions.php")
	content, err := os.ReadFile(functionsPath)
	if err != nil {
		return nil
	}

	// Find all wp_enqueue_style calls and extract handles
	re := regexp.MustCompile(`wp_enqueue_style\s*\(\s*['"]([^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	var handles []string
	for _, match := range matches {
		if len(match) > 1 {
			handles = append(handles, match[1])
		}
	}

	return handles
}

// updateChildStyleDependencies updates the child theme's functions.php with parent style dependencies
func (b *ThemeBuilder) updateChildStyleDependencies(stageDir string) error {
	handles := b.GetParentStyleHandles()
	if len(handles) == 0 {
		return nil
	}

	functionsPath := filepath.Join(stageDir, "functions.php")
	content, err := os.ReadFile(functionsPath)
	if err != nil {
		return err
	}

	// Build the new dependency array with all parent handles
	var deps []string
	for _, h := range handles {
		deps = append(deps, fmt.Sprintf("'%s'", h))
	}
	newDeps := "array(" + strings.Join(deps, ", ") + ")"

	// Find and replace the child CSS dependency array
	// Look for pattern like: array('theme-style') in the child CSS enqueue
	slug := SanitizeName(b.Config.Name)

	// Match the specific enqueue for child.css with its dependency array
	re := regexp.MustCompile(`(wp_enqueue_style\s*\(\s*'` + regexp.QuoteMeta(slug) + `-child'\s*,\s*[^,]+,\s*)array\s*\([^)]*\)`)

	updated := re.ReplaceAllString(string(content), "${1}"+newDeps)

	if updated == string(content) {
		// Try alternate pattern - maybe the array reference is different
		re = regexp.MustCompile(`(get_stylesheet_directory_uri\s*\(\s*\)\s*\.\s*'/assets/css/child\.css'\s*,\s*)array\s*\([^)]*\)`)
		updated = re.ReplaceAllString(string(content), "${1}"+newDeps)
	}

	return os.WriteFile(functionsPath, []byte(updated), 0644)
}

func (b *ThemeBuilder) fetchParentTheme() error {
	uri := b.Config.TemplateURI
	parentDir := filepath.Join(b.WorkDir, "parent")

	// Create parent directory
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Check if it's a URL
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return b.downloadAndExtractTheme(uri, parentDir)
	}

	// It's a file path
	srcPath := uri
	if !filepath.IsAbs(srcPath) {
		srcPath = filepath.Join(b.SourceDir, uri)
	}

	// Check if it's a zip file
	if strings.HasSuffix(strings.ToLower(srcPath), ".zip") {
		return ExtractZip(srcPath, parentDir)
	}

	// It's a directory - check if it has theme.properties (needs to be built)
	themePropsPath := filepath.Join(srcPath, "theme.properties")
	if _, err := os.Stat(themePropsPath); err == nil {
		// This is a theme source directory - check if it's already built
		builtPath := filepath.Join(srcPath, "build", "work", "stage")
		parentParentPath := filepath.Join(srcPath, "build", "work", "parent")

		if _, err := os.Stat(builtPath); err == nil {
			// Use existing build
			if err := CopyDir(builtPath, parentDir); err != nil {
				return err
			}
			// Also copy grandparent if exists
			if _, err := os.Stat(parentParentPath); err == nil {
				grandparentDir := filepath.Join(parentDir, "parent")
				return CopyDir(parentParentPath, grandparentDir)
			}
			return nil
		}

		// Need to build the parent theme
		if !b.Quiet {
			ui.PrintInfo("Building parent theme...")
		}
		parentBuilder := NewThemeBuilder(srcPath)
		parentBuilder.Quiet = true
		if err := parentBuilder.Build(); err != nil {
			return fmt.Errorf("failed to build parent theme: %w", err)
		}

		// Copy the built theme
		if err := CopyDir(parentBuilder.GetStagePath(), parentDir); err != nil {
			return err
		}

		// Also copy grandparent chain if exists
		grandparentPath := parentBuilder.GetParentThemePath()
		if grandparentPath != "" {
			grandparentDir := filepath.Join(parentDir, "parent")
			return CopyDir(grandparentPath, grandparentDir)
		}
		return nil
	}

	// It's a regular directory - copy it directly
	return CopyDir(srcPath, parentDir)
}

func (b *ThemeBuilder) downloadAndExtractTheme(url, destDir string) error {
	// Download to temp file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create temp file for zip
	tmpFile, err := os.CreateTemp("", "theme-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy response to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}
	tmpFile.Close()

	// Extract the zip
	return ExtractZip(tmpFile.Name(), destDir)
}
