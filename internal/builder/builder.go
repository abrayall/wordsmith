package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wordsmith/internal/config"
	"wordsmith/internal/obfuscator"
	"wordsmith/internal/ui"
)

// PluginDependency represents a resolved plugin dependency
type PluginDependency struct {
	Slug    string // WordPress slug for installation/activation
	Path    string // Local path to built/resolved plugin (empty for WP.org plugins)
	IsWPOrg bool   // True if WordPress.org plugin
	Version string // Version if specified
}

// Builder builds WordPress plugins
type Builder struct {
	BaseBuilder
	Config       *config.PluginConfig
	Dependencies []PluginDependency // Resolved plugin dependencies
}

// New creates a new plugin Builder
func New(sourceDir string) *Builder {
	return &Builder{
		BaseBuilder: NewBaseBuilder(sourceDir),
	}
}

// Build builds the plugin
func (b *Builder) Build() error {
	if !b.Quiet {
		ui.PrintInfo("Loading plugin.properties...")
	}
	cfg, err := config.LoadPluginProperties(b.SourceDir)
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

	sourceWorkDir := filepath.Join(b.WorkDir, "source")
	stageDir, err := b.CreateStageDir()
	if err != nil {
		return err
	}

	pluginName := b.GetPluginSlug()

	if err := os.MkdirAll(sourceWorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create source directory: %w", err)
	}

	if !b.Quiet {
		ui.PrintInfo("Copying plugin files...")
	}

	mainFile := filepath.Base(b.Config.Main)
	mainSrc := filepath.Join(b.SourceDir, b.Config.Main)

	if err := CopyFile(mainSrc, filepath.Join(sourceWorkDir, mainFile)); err != nil {
		return fmt.Errorf("failed to copy main plugin file: %w", err)
	}

	// Expand glob patterns in includes
	expandedIncludes, err := ExpandIncludes(b.SourceDir, b.Config.Include, b.Config.Exclude)
	if err != nil {
		return fmt.Errorf("failed to expand include patterns: %w", err)
	}

	for _, include := range expandedIncludes {
		src := filepath.Join(b.SourceDir, include)
		info, err := os.Stat(src)
		if err != nil {
			ui.PrintWarning("Skipping %s: %v", include, err)
			continue
		}

		if info.IsDir() {
			if err := b.copyDirSplitWithExcludes(src, include, sourceWorkDir, stageDir, b.Config.Exclude); err != nil {
				return fmt.Errorf("failed to copy directory %s: %w", include, err)
			}
		} else {
			if strings.HasSuffix(include, ".php") {
				if err := CopyFile(src, filepath.Join(sourceWorkDir, include)); err != nil {
					return fmt.Errorf("failed to copy file %s: %w", include, err)
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
	}

	readmeSrc := filepath.Join(b.SourceDir, "readme.txt")
	readmeDst := filepath.Join(stageDir, "readme.txt")
	if _, err := os.Stat(readmeSrc); err == nil {
		if err := CopyFile(readmeSrc, readmeDst); err != nil {
			ui.PrintWarning("Failed to copy readme.txt: %v", err)
		}
	} else {
		if err := b.generateReadme(readmeDst); err != nil {
			ui.PrintWarning("Failed to generate readme.txt: %v", err)
		}
	}

	if !b.Quiet {
		ui.PrintInfo("Processing PHP files...")
	}

	err = filepath.Walk(sourceWorkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(sourceWorkDir, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(stageDir, relPath)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		output := string(content)

		if strings.HasSuffix(info.Name(), ".php") {
			output = b.replaceVersionConstants(output)

			if b.Config.Obfuscate {
				output, err = obfuscator.Obfuscate(output)
				if err != nil {
					return fmt.Errorf("failed to obfuscate %s: %w", relPath, err)
				}
			}
		}

		return os.WriteFile(dstPath, []byte(output), info.Mode())
	})
	if err != nil {
		return fmt.Errorf("failed to process PHP files: %w", err)
	}

	if err := b.generatePluginHeader(filepath.Join(stageDir, mainFile)); err != nil {
		return fmt.Errorf("failed to generate plugin header: %w", err)
	}

	versionFile := filepath.Join(stageDir, "version.properties")
	if err := WriteVersionProperties(versionFile, b.Config.Name, b.Version); err != nil {
		return fmt.Errorf("failed to write version.properties: %w", err)
	}

	pluginPropsFile := filepath.Join(stageDir, "plugin.properties")
	if err := b.writePluginProperties(pluginPropsFile); err != nil {
		return fmt.Errorf("failed to write plugin.properties: %w", err)
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

	// Build/resolve plugin dependencies
	if len(b.Config.Plugins) > 0 {
		if !b.Quiet {
			ui.PrintInfo("Resolving plugin dependencies...")
		}
		if err := b.buildPluginDependencies(); err != nil {
			return fmt.Errorf("failed to resolve plugin dependencies: %w", err)
		}
	}

	CleanDevFiles(stageDir)

	// Set permissions on all files before zipping
	if err := ChmodAll(stageDir, 0777); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if !b.Quiet {
		ui.PrintInfo("Creating ZIP archive...")
	}
	zipPath := filepath.Join(b.BuildDir, fmt.Sprintf("%s-%s.zip", pluginName, b.Version.String()))
	if err := CreateZip(stageDir, zipPath, pluginName); err != nil {
		return fmt.Errorf("failed to create ZIP: %w", err)
	}

	if !b.Quiet {
		fmt.Println()
		ui.PrintSuccess("Created: %s", filepath.Base(zipPath))
	}

	return nil
}

// GetPluginSlug returns the WordPress plugin slug (directory name) for this plugin.
func (b *Builder) GetPluginSlug() string {
	if b.Config == nil {
		return ""
	}
	if b.Config.Slug != "" {
		return b.Config.Slug
	}
	return SanitizeName(b.Config.Name)
}

func (b *Builder) copyDirSplitWithExcludes(src, relBase, phpDst, otherDst string, excludes []string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		fullRel := relBase
		if relPath != "." {
			fullRel = filepath.Join(relBase, relPath)
		}

		// Check if excluded
		if IsExcluded(fullRel, excludes) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			os.MkdirAll(filepath.Join(phpDst, fullRel), info.Mode())
			os.MkdirAll(filepath.Join(otherDst, fullRel), info.Mode())
			return nil
		}

		if strings.HasSuffix(info.Name(), ".php") {
			return CopyFile(path, filepath.Join(phpDst, fullRel))
		}

		dst := filepath.Join(otherDst, fullRel)
		if b.Config.Minify && (strings.HasSuffix(info.Name(), ".css") || strings.HasSuffix(info.Name(), ".js")) {
			return CopyAndMinify(path, dst, true)
		}
		return CopyFile(path, dst)
	})
}

func (b *Builder) replaceVersionConstants(content string) string {
	pluginName := strings.ToUpper(SanitizeName(b.Config.Name))
	pluginName = strings.ReplaceAll(pluginName, "-", "_")

	re := regexp.MustCompile(`define\s*\(\s*['"]` + pluginName + `_VERSION['"]\s*,\s*['"][^'"]*['"]\s*\)`)
	replacement := fmt.Sprintf("define('%s_VERSION', '%s')", pluginName, b.Version.String())

	return re.ReplaceAllString(content, replacement)
}

func (b *Builder) generatePluginHeader(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	header := "<?php\n/**\n"
	header += fmt.Sprintf(" * Plugin Name: %s\n", b.Config.Name)

	if b.Config.PluginURI != "" {
		header += fmt.Sprintf(" * Plugin URI: %s\n", b.Config.PluginURI)
	}
	if b.Config.Description != "" {
		header += fmt.Sprintf(" * Description: %s\n", b.Config.Description)
	}
	header += fmt.Sprintf(" * Version: %s\n", b.Version.String())
	if b.Config.Author != "" {
		header += fmt.Sprintf(" * Author: %s\n", b.Config.Author)
	}
	if b.Config.AuthorURI != "" {
		header += fmt.Sprintf(" * Author URI: %s\n", b.Config.AuthorURI)
	}
	if b.Config.License != "" {
		header += fmt.Sprintf(" * License: %s\n", b.Config.License)
	}
	if b.Config.LicenseURI != "" {
		header += fmt.Sprintf(" * License URI: %s\n", b.Config.LicenseURI)
	}
	if b.Config.TextDomain != "" {
		header += fmt.Sprintf(" * Text Domain: %s\n", b.Config.TextDomain)
	}
	if b.Config.DomainPath != "" {
		header += fmt.Sprintf(" * Domain Path: %s\n", b.Config.DomainPath)
	}
	if b.Config.Requires != "" {
		header += fmt.Sprintf(" * Requires at least: %s\n", b.Config.Requires)
	}
	if b.Config.RequiresPHP != "" {
		header += fmt.Sprintf(" * Requires PHP: %s\n", b.Config.RequiresPHP)
	}
	// Add Requires Plugins header for WordPress.org plugin dependencies
	if requiresPlugins := b.getRequiresPluginsFromConfig(); requiresPlugins != "" {
		header += fmt.Sprintf(" * Requires Plugins: %s\n", requiresPlugins)
	}
	header += " */\n"

	contentStr := string(content)
	re := regexp.MustCompile(`(?s)^<\?php\s*/\*\*.*?\*/\s*`)
	updated := re.ReplaceAllString(contentStr, header)

	if updated == contentStr {
		re = regexp.MustCompile(`^<\?php\s*`)
		updated = re.ReplaceAllString(contentStr, header)
	}

	return os.WriteFile(path, []byte(updated), 0644)
}

func (b *Builder) writePluginProperties(path string) error {
	var lines []string
	lines = append(lines, "# Plugin metadata")
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
	if b.Config.PluginURI != "" {
		lines = append(lines, fmt.Sprintf("plugin-uri=%s", b.Config.PluginURI))
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
	if b.Config.Requires != "" {
		lines = append(lines, fmt.Sprintf("requires=%s", b.Config.Requires))
	}
	if b.Config.RequiresPHP != "" {
		lines = append(lines, fmt.Sprintf("requires-php=%s", b.Config.RequiresPHP))
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0644)
}

func (b *Builder) generateReadme(path string) error {
	requires := b.Config.Requires
	if requires == "" {
		requires = "5.0"
	}
	requiresPHP := b.Config.RequiresPHP
	if requiresPHP == "" {
		requiresPHP = "7.4"
	}
	license := b.Config.License
	if license == "" {
		license = "GPLv2 or later"
	}

	content := fmt.Sprintf(`=== %s ===
Contributors: %s
Tags: wordpress
Requires at least: %s
Tested up to: 6.4
Stable tag: %s
Requires PHP: %s
License: %s
License URI: %s

== Description ==

%s

== Installation ==

1. Upload the plugin files to the /wp-content/plugins/ directory
2. Activate the plugin through the 'Plugins' screen in WordPress
`, b.Config.Name, b.Config.Author, requires, b.Version.String(), requiresPHP, license, b.Config.LicenseURI, b.Config.Description)

	return os.WriteFile(path, []byte(content), 0644)
}

// buildPluginDependencies resolves and builds all plugin dependencies.
// For WordPress.org slugs, just records them for the header.
// For local directories with plugin.properties, builds them recursively.
// For URLs/zips, downloads and extracts them.
func (b *Builder) buildPluginDependencies() error {
	pluginsDir := filepath.Join(b.WorkDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	for _, spec := range b.Config.Plugins {
		dep, err := b.resolvePluginDependency(spec, pluginsDir)
		if err != nil {
			return fmt.Errorf("failed to resolve plugin '%s': %w", spec.Name, err)
		}
		b.Dependencies = append(b.Dependencies, dep)
	}

	return nil
}

// resolvePluginDependency resolves a single plugin dependency
func (b *Builder) resolvePluginDependency(spec config.LibrarySpec, pluginsDir string) (PluginDependency, error) {
	// Check if this is a WordPress.org slug
	if config.IsWordPressOrgSlug(spec) {
		return PluginDependency{
			Slug:    spec.Name,
			IsWPOrg: true,
			Version: spec.Version,
		}, nil
	}

	// Resolve the path relative to source directory for local paths
	url := spec.URL
	if config.IsLocalPath(url) && !filepath.IsAbs(url) {
		url = filepath.Join(b.SourceDir, url)
	}

	// Check if it's a directory with plugin.properties (needs building)
	if info, err := os.Stat(url); err == nil && info.IsDir() {
		pluginPropsPath := filepath.Join(url, "plugin.properties")
		if _, err := os.Stat(pluginPropsPath); err == nil {
			return b.buildPluginFromSource(url, pluginsDir)
		}
	}

	// Otherwise resolve like a library (download/extract)
	resolvedSpec := config.LibrarySpec{
		Name:    spec.Name,
		URL:     url,
		Version: spec.Version,
	}

	libPath, err := config.ResolveLibrary(resolvedSpec)
	if err != nil {
		return PluginDependency{}, err
	}

	// Copy to plugins directory
	targetDir := filepath.Join(pluginsDir, spec.Name)
	if err := copyDir(libPath, targetDir); err != nil {
		return PluginDependency{}, fmt.Errorf("failed to copy plugin: %w", err)
	}

	return PluginDependency{
		Slug:    spec.Name,
		Path:    targetDir,
		IsWPOrg: false,
		Version: spec.Version,
	}, nil
}

// buildPluginFromSource builds a plugin from a source directory with plugin.properties
func (b *Builder) buildPluginFromSource(srcDir string, pluginsDir string) (PluginDependency, error) {
	// Check if already built
	stageDir := filepath.Join(srcDir, "build", "work", "stage")
	if info, err := os.Stat(stageDir); err == nil && info.IsDir() {
		// Use the pre-built version
		cfg, err := config.LoadPluginProperties(srcDir)
		if err != nil {
			return PluginDependency{}, fmt.Errorf("failed to load plugin.properties: %w", err)
		}

		slug := SanitizeName(cfg.Name)
		if cfg.Slug != "" {
			slug = cfg.Slug
		}

		targetDir := filepath.Join(pluginsDir, slug)
		if err := copyDir(stageDir, targetDir); err != nil {
			return PluginDependency{}, fmt.Errorf("failed to copy built plugin: %w", err)
		}

		return PluginDependency{
			Slug:    slug,
			Path:    targetDir,
			IsWPOrg: false,
		}, nil
	}

	// Build the plugin
	depBuilder := New(srcDir)
	depBuilder.Quiet = true
	if err := depBuilder.Build(); err != nil {
		return PluginDependency{}, fmt.Errorf("failed to build plugin: %w", err)
	}

	slug := depBuilder.GetPluginSlug()
	builtStageDir := filepath.Join(srcDir, "build", "work", "stage")
	targetDir := filepath.Join(pluginsDir, slug)

	if err := copyDir(builtStageDir, targetDir); err != nil {
		return PluginDependency{}, fmt.Errorf("failed to copy built plugin: %w", err)
	}

	return PluginDependency{
		Slug:    slug,
		Path:    targetDir,
		IsWPOrg: false,
	}, nil
}

// GetPluginDependencies returns all resolved plugin dependencies
func (b *Builder) GetPluginDependencies() []PluginDependency {
	return b.Dependencies
}

// GetRequiresPlugins returns a comma-separated list of WordPress.org plugin slugs
// for use in the Requires Plugins header
func (b *Builder) GetRequiresPlugins() string {
	var slugs []string
	for _, dep := range b.Dependencies {
		if dep.IsWPOrg {
			slugs = append(slugs, dep.Slug)
		}
	}
	return strings.Join(slugs, ", ")
}

// getRequiresPluginsFromConfig returns WordPress.org plugin slugs from config
// Used during header generation before dependencies are fully resolved
func (b *Builder) getRequiresPluginsFromConfig() string {
	var slugs []string
	for _, spec := range b.Config.Plugins {
		if config.IsWordPressOrgSlug(spec) {
			slugs = append(slugs, spec.Name)
		}
	}
	return strings.Join(slugs, ", ")
}

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(targetPath, content, info.Mode())
	})
}
