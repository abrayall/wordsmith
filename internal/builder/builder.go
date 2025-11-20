package builder

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wordsmith/internal/config"
	"wordsmith/internal/obfuscator"
	"wordsmith/internal/ui"
	"wordsmith/internal/version"
)

type Builder struct {
	SourceDir string
	BuildDir  string
	WorkDir   string
	Config    *config.PluginConfig
	Version   *version.Version
	Quiet     bool
}

func New(sourceDir string) *Builder {
	buildDir := filepath.Join(sourceDir, "build")
	return &Builder{
		SourceDir: sourceDir,
		BuildDir:  buildDir,
		WorkDir:   filepath.Join(buildDir, "work"),
	}
}

func (b *Builder) Build() error {
	if !b.Quiet {
		ui.PrintInfo("Loading plugin.properties...")
	}
	cfg, err := config.LoadPluginProperties(b.SourceDir)
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

	sourceWorkDir := filepath.Join(b.WorkDir, "source")
	stageDir := filepath.Join(b.WorkDir, "stage")
	pluginName := b.sanitizeName(b.Config.Name)

	if err := os.MkdirAll(sourceWorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create source directory: %w", err)
	}
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return fmt.Errorf("failed to create stage directory: %w", err)
	}

	if !b.Quiet {
		ui.PrintInfo("Copying plugin files...")
	}

	mainFile := filepath.Base(b.Config.Main)
	mainSrc := filepath.Join(b.SourceDir, b.Config.Main)

	if err := b.copyFile(mainSrc, filepath.Join(sourceWorkDir, mainFile)); err != nil {
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
				if err := b.copyFile(src, filepath.Join(sourceWorkDir, include)); err != nil {
					return fmt.Errorf("failed to copy file %s: %w", include, err)
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
	}

	readmeSrc := filepath.Join(b.SourceDir, "readme.txt")
	readmeDst := filepath.Join(stageDir, "readme.txt")
	if _, err := os.Stat(readmeSrc); err == nil {
		if err := b.copyFile(readmeSrc, readmeDst); err != nil {
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
	if err := b.writeVersionProperties(versionFile); err != nil {
		return fmt.Errorf("failed to write version.properties: %w", err)
	}

	pluginPropsFile := filepath.Join(stageDir, "plugin.properties")
	if err := b.writePluginProperties(pluginPropsFile); err != nil {
		return fmt.Errorf("failed to write plugin.properties: %w", err)
	}

	b.cleanDevFiles(stageDir)

	// Set permissions on all files before zipping
	if err := chmodAll(stageDir, 0777); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if !b.Quiet {
		ui.PrintInfo("Creating ZIP archive...")
	}
	zipPath := filepath.Join(b.BuildDir, fmt.Sprintf("%s-%s.zip", pluginName, b.Version.String()))
	if err := b.createZip(stageDir, zipPath, pluginName); err != nil {
		return fmt.Errorf("failed to create ZIP: %w", err)
	}

	if !b.Quiet {
		fmt.Println()
		ui.PrintSuccess("Created: %s", filepath.Base(zipPath))
	}

	return nil
}

func (b *Builder) sanitizeName(name string) string {
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	result = re.ReplaceAllString(result, "")
	return result
}

func (b *Builder) copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (b *Builder) copyDirSplit(src, relBase, phpDst, otherDst string) error {
	return b.copyDirSplitWithExcludes(src, relBase, phpDst, otherDst, nil)
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
			return b.copyFile(path, filepath.Join(phpDst, fullRel))
		}

		dst := filepath.Join(otherDst, fullRel)
		if b.Config.Minify && (strings.HasSuffix(info.Name(), ".css") || strings.HasSuffix(info.Name(), ".js")) {
			return b.copyAndMinify(path, dst)
		}
		return b.copyFile(path, dst)
	})
}

func (b *Builder) copyAndMinify(src, dst string) error {
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

func (b *Builder) replaceVersionConstants(content string) string {
	pluginName := strings.ToUpper(b.sanitizeName(b.Config.Name))
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

func (b *Builder) writeVersionProperties(path string) error {
	content := fmt.Sprintf(`# %s Version Information
# Generated by wordsmith

major=%d
minor=%d
maintenance=%s
`, b.Config.Name, b.Version.Major, b.Version.Minor, b.Version.Maintenance)

	return os.WriteFile(path, []byte(content), 0644)
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

func (b *Builder) cleanDevFiles(dir string) {
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

func (b *Builder) createZip(sourceDir, zipPath, baseName string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		archivePath := filepath.Join(baseName, relPath)

		if info.IsDir() {
			if relPath != "." {
				_, err = archive.Create(archivePath + "/")
			}
			return err
		}

		writer, err := archive.Create(archivePath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// chmodAll recursively sets permissions on all files and directories
func chmodAll(dir string, mode os.FileMode) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(path, mode)
	})
}
