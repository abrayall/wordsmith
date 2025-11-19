package builder

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wordsmith/internal/config"
	"wordsmith/internal/obfuscator"
	"wordsmith/internal/ui"
	"wordsmith/internal/version"
)

type ThemeBuilder struct {
	SourceDir string
	BuildDir  string
	WorkDir   string
	Config    *config.ThemeConfig
	Version   *version.Version
	Quiet     bool
}

func NewThemeBuilder(sourceDir string) *ThemeBuilder {
	buildDir := filepath.Join(sourceDir, "build")
	return &ThemeBuilder{
		SourceDir: sourceDir,
		BuildDir:  buildDir,
		WorkDir:   filepath.Join(buildDir, "work"),
	}
}

func (b *ThemeBuilder) Build() error {
	if !b.Quiet {
		ui.PrintInfo("Loading theme.properties...")
	}
	cfg, err := config.LoadThemeProperties(b.SourceDir)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	b.Config = cfg

	if cfg.Version != "" {
		b.Version = &version.Version{
			Major:       0,
			Minor:       0,
			Maintenance: cfg.Version,
		}
		re := regexp.MustCompile(`^(\d+)\.(\d+)\.(.+)$`)
		if matches := re.FindStringSubmatch(cfg.Version); matches != nil {
			fmt.Sscanf(matches[1], "%d", &b.Version.Major)
			fmt.Sscanf(matches[2], "%d", &b.Version.Minor)
			b.Version.Maintenance = matches[3]
		}
	} else {
		if !b.Quiet {
			ui.PrintInfo("Reading version from git tags...")
		}
		ver, err := version.GetFromGit(b.SourceDir)
		if err != nil {
			return fmt.Errorf("failed to get version from git: %w", err)
		}
		b.Version = ver
		if ver.IsDirty && !b.Quiet {
			ui.PrintWarning("Detected uncommitted changes, appending timestamp")
		}
	}

	if b.Quiet {
		ui.PrintInfo("Building %s v%s", b.Config.Name, b.Version.String())
	} else {
		fmt.Println()
		ui.PrintKeyValue("Name", "    "+b.Config.Name)
		ui.PrintKeyValue("Version", " "+b.Version.String())
		fmt.Println()
	}

	if !b.Quiet {
		ui.PrintInfo("Cleaning build directory...")
	}
	if err := os.RemoveAll(b.BuildDir); err != nil {
		return fmt.Errorf("failed to clean build directory: %w", err)
	}

	stageDir := filepath.Join(b.WorkDir, "stage")
	themeName := b.sanitizeName(b.Config.Name)

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return fmt.Errorf("failed to create stage directory: %w", err)
	}

	if !b.Quiet {
		ui.PrintInfo("Copying theme files...")
	}

	// Copy main stylesheet
	mainFile := filepath.Base(b.Config.Main)
	mainSrc := filepath.Join(b.SourceDir, b.Config.Main)
	mainDst := filepath.Join(stageDir, mainFile)

	if err := b.copyFile(mainSrc, mainDst); err != nil {
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
				if err := b.copyAndMinify(src, dst); err != nil {
					return fmt.Errorf("failed to minify file %s: %w", include, err)
				}
			} else {
				if err := b.copyFile(src, dst); err != nil {
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
	if err := b.writeVersionProperties(versionFile); err != nil {
		return fmt.Errorf("failed to write version.properties: %w", err)
	}

	// Write theme.properties
	themePropsFile := filepath.Join(stageDir, "theme.properties")
	if err := b.writeThemeProperties(themePropsFile); err != nil {
		return fmt.Errorf("failed to write theme.properties: %w", err)
	}

	// Fetch parent theme if template-uri is specified
	if b.Config.TemplateURI != "" {
		if !b.Quiet {
			ui.PrintInfo("Fetching parent theme...")
		}
		if err := b.fetchParentTheme(); err != nil {
			return fmt.Errorf("failed to fetch parent theme: %w", err)
		}
	}

	// Clean dev files
	b.cleanDevFiles(stageDir)

	// Create ZIP
	if !b.Quiet {
		ui.PrintInfo("Creating ZIP archive...")
	}
	zipPath := filepath.Join(b.BuildDir, fmt.Sprintf("%s-%s.zip", themeName, b.Version.String()))
	if err := b.createZip(stageDir, zipPath, themeName); err != nil {
		return fmt.Errorf("failed to create ZIP: %w", err)
	}

	if !b.Quiet {
		fmt.Println()
		ui.PrintSuccess("Created: %s", filepath.Base(zipPath))
	}

	return nil
}

func (b *ThemeBuilder) sanitizeName(name string) string {
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	result = re.ReplaceAllString(result, "")
	return result
}

func (b *ThemeBuilder) copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, content, 0644)
}

func (b *ThemeBuilder) copyDir(src, dst string) error {
	return b.copyDirWithExcludes(src, dst, nil)
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
			return b.copyAndMinify(path, targetPath)
		}
		return b.copyFile(path, targetPath)
	})
}

func (b *ThemeBuilder) copyAndMinify(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	var minified string
	if strings.HasSuffix(src, ".css") {
		minified = obfuscator.MinifyCSS(string(content))
	} else {
		minified = obfuscator.MinifyJS(string(content))
	}

	return os.WriteFile(dst, []byte(minified), 0644)
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

func (b *ThemeBuilder) writeVersionProperties(path string) error {
	content := fmt.Sprintf(`# %s Version Information
# Generated by wordsmith

major=%d
minor=%d
maintenance=%s
`, b.Config.Name, b.Version.Major, b.Version.Minor, b.Version.Maintenance)

	return os.WriteFile(path, []byte(content), 0644)
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

func (b *ThemeBuilder) cleanDevFiles(dir string) {
	patterns := []string{".DS_Store", "*.swp", "*.swo", "*~", ".git", ".gitignore"}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		name := info.Name()
		for _, pattern := range patterns {
			if matched, _ := filepath.Match(pattern, name); matched {
				os.RemoveAll(path)
				break
			}
		}
		return nil
	})
}

func (b *ThemeBuilder) createZip(sourceDir, zipPath, baseName string) error {
	// Reuse the Builder's createZip by creating a temporary Builder
	tempBuilder := &Builder{
		SourceDir: b.SourceDir,
		BuildDir:  b.BuildDir,
		WorkDir:   b.WorkDir,
	}
	return tempBuilder.createZip(sourceDir, zipPath, baseName)
}

// GetStagePath returns the path to the staged theme directory
func (b *ThemeBuilder) GetStagePath() string {
	return filepath.Join(b.WorkDir, "stage")
}

// GetThemeName returns the sanitized theme name
func (b *ThemeBuilder) GetThemeName() string {
	return b.sanitizeName(b.Config.Name)
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
		return b.extractZip(srcPath, parentDir)
	}

	// It's a directory - check if it has theme.properties (needs to be built)
	themePropsPath := filepath.Join(srcPath, "theme.properties")
	if _, err := os.Stat(themePropsPath); err == nil {
		// This is a theme source directory - check if it's already built
		builtPath := filepath.Join(srcPath, "build", "work", "stage")
		parentParentPath := filepath.Join(srcPath, "build", "work", "parent")

		if _, err := os.Stat(builtPath); err == nil {
			// Use existing build
			if err := b.copyDir(builtPath, parentDir); err != nil {
				return err
			}
			// Also copy grandparent if exists
			if _, err := os.Stat(parentParentPath); err == nil {
				grandparentDir := filepath.Join(parentDir, "parent")
				return b.copyDir(parentParentPath, grandparentDir)
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
		if err := b.copyDir(parentBuilder.GetStagePath(), parentDir); err != nil {
			return err
		}

		// Also copy grandparent chain if exists
		grandparentPath := parentBuilder.GetParentThemePath()
		if grandparentPath != "" {
			grandparentDir := filepath.Join(parentDir, "parent")
			return b.copyDir(grandparentPath, grandparentDir)
		}
		return nil
	}

	// It's a regular directory - copy it directly
	return b.copyDir(srcPath, parentDir)
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
	return b.extractZip(tmpFile.Name(), destDir)
}

func (b *ThemeBuilder) extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	// Find the root directory in the zip (if any)
	var rootDir string
	for _, f := range r.File {
		parts := strings.Split(f.Name, "/")
		if len(parts) > 1 && rootDir == "" {
			rootDir = parts[0]
		}
		break
	}

	for _, f := range r.File {
		// Calculate destination path
		name := f.Name

		// Strip the root directory if all files are in one
		if rootDir != "" && strings.HasPrefix(name, rootDir+"/") {
			name = strings.TrimPrefix(name, rootDir+"/")
		}

		if name == "" {
			continue
		}

		fpath := filepath.Join(destDir, name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// Extract file
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
