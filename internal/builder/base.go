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

// BaseBuilder contains shared functionality for plugin and theme builders
type BaseBuilder struct {
	SourceDir string
	BuildDir  string
	WorkDir   string
	Version   *version.Version
	Quiet     bool
}

// NewBaseBuilder creates a new BaseBuilder
func NewBaseBuilder(sourceDir string) BaseBuilder {
	buildDir := filepath.Join(sourceDir, "build")
	return BaseBuilder{
		SourceDir: sourceDir,
		BuildDir:  buildDir,
		WorkDir:   filepath.Join(buildDir, "work"),
	}
}

// ParseVersion parses a version string into a Version struct
func ParseVersion(versionStr string) *version.Version {
	ver := &version.Version{
		Major:       0,
		Minor:       0,
		Maintenance: versionStr,
	}
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(.+)$`)
	if matches := re.FindStringSubmatch(versionStr); matches != nil {
		fmt.Sscanf(matches[1], "%d", &ver.Major)
		fmt.Sscanf(matches[2], "%d", &ver.Minor)
		ver.Maintenance = matches[3]
	}
	return ver
}

// GetVersionFromGit gets version from git tags
func (b *BaseBuilder) GetVersionFromGit() (*version.Version, error) {
	if !b.Quiet {
		ui.PrintInfo("Reading version from git tags...")
	}
	ver, err := version.GetFromGit(b.SourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get version from git: %w", err)
	}
	if ver.IsDirty && !b.Quiet {
		ui.PrintWarning("Detected uncommitted changes, appending timestamp")
	}
	return ver, nil
}

// PrintBuildInfo prints the build information
func (b *BaseBuilder) PrintBuildInfo(name string) {
	if b.Quiet {
		ui.PrintInfo("Building %s v%s", name, b.Version.String())
	} else {
		fmt.Println()
		ui.PrintKeyValue("Name", "    "+name)
		ui.PrintKeyValue("Version", " "+b.Version.String())
		fmt.Println()
	}
}

// CleanBuildDir removes and recreates the build directory
func (b *BaseBuilder) CleanBuildDir() error {
	if !b.Quiet {
		ui.PrintInfo("Cleaning build directory...")
	}
	if err := os.RemoveAll(b.BuildDir); err != nil {
		return fmt.Errorf("failed to clean build directory: %w", err)
	}
	return nil
}

// CreateStageDir creates the stage directory
func (b *BaseBuilder) CreateStageDir() (string, error) {
	stageDir := filepath.Join(b.WorkDir, "stage")
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create stage directory: %w", err)
	}
	return stageDir, nil
}

// SanitizeName converts a name to a slug
func SanitizeName(name string) string {
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	result = re.ReplaceAllString(result, "")
	return result
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
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

// CopyDir copies a directory recursively
func CopyDir(src, dst string) error {
	return CopyDirWithExcludes(src, dst, nil)
}

// CopyDirWithExcludes copies a directory recursively, excluding specified patterns
func CopyDirWithExcludes(src, dst string, excludes []string) error {
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

		return CopyFile(path, targetPath)
	})
}

// CopyAndMinify copies a file and minifies it if it's CSS or JS
func CopyAndMinify(src, dst string, minify bool) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if minify {
		var minified string
		if strings.HasSuffix(src, ".css") {
			minified = obfuscator.MinifyCSS(string(content))
		} else if strings.HasSuffix(src, ".js") {
			minified = obfuscator.MinifyJS(string(content))
		} else {
			return os.WriteFile(dst, content, 0644)
		}
		return os.WriteFile(dst, []byte(minified), 0644)
	}

	return os.WriteFile(dst, content, 0644)
}

// CleanDevFiles removes development files from a directory
func CleanDevFiles(dir string) {
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

// CreateZip creates a zip archive from a directory
func CreateZip(sourceDir, zipPath, baseName string) error {
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

// ChmodAll recursively sets permissions on all files and directories
func ChmodAll(dir string, mode os.FileMode) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(path, mode)
	})
}

// CopyLibraries resolves and copies all libraries to the stage directory
func CopyLibraries(libraries []config.LibrarySpec, stageDir string, quiet bool) error {
	for _, lib := range libraries {
		if !quiet {
			ui.PrintInfo("  Resolving library: %s", lib.Name)
		}

		// Resolve the library to a local path
		libPath, err := config.ResolveLibrary(lib)
		if err != nil {
			return fmt.Errorf("failed to resolve library %s: %w", lib.Name, err)
		}

		// Copy to stage directory
		if err := config.CopyLibraryToDir(libPath, stageDir, lib.Name); err != nil {
			return fmt.Errorf("failed to copy library %s: %w", lib.Name, err)
		}
	}
	return nil
}

// WriteVersionProperties writes version.properties file
func WriteVersionProperties(path, name string, ver *version.Version) error {
	content := fmt.Sprintf(`# %s Version Information
# Generated by wordsmith

major=%d
minor=%d
maintenance=%s
`, name, ver.Major, ver.Minor, ver.Maintenance)

	return os.WriteFile(path, []byte(content), 0644)
}

// ExtractZip extracts a zip file to a destination directory
func ExtractZip(zipPath, destDir string) error {
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
