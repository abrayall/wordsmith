package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

// LibraryBuilder builds reusable code libraries
type LibraryBuilder struct {
	BaseBuilder
	Config *config.LibraryConfig
}

// NewLibraryBuilder creates a new library Builder
func NewLibraryBuilder(sourceDir string) *LibraryBuilder {
	return &LibraryBuilder{
		BaseBuilder: NewBaseBuilder(sourceDir),
	}
}

// Build builds the library
func (b *LibraryBuilder) Build() error {
	if !b.Quiet {
		ui.PrintInfo("Loading library.properties...")
	}
	cfg, err := config.LoadLibraryProperties(b.SourceDir)
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

	slug := b.GetLibrarySlug()

	if !b.Quiet {
		ui.PrintInfo("Copying library files...")
	}

	// Expand glob patterns in includes
	expandedIncludes, err := ExpandIncludes(b.SourceDir, b.Config.Include, b.Config.Exclude)
	if err != nil {
		return fmt.Errorf("failed to expand include patterns: %w", err)
	}

	// Copy included files/directories (no processing, no minification)
	for _, include := range expandedIncludes {
		src := filepath.Join(b.SourceDir, include)
		info, err := os.Stat(src)
		if err != nil {
			ui.PrintWarning("Skipping %s: %v", include, err)
			continue
		}

		if info.IsDir() {
			if err := CopyDirWithExcludes(src, filepath.Join(stageDir, include), b.Config.Exclude); err != nil {
				return fmt.Errorf("failed to copy directory %s: %w", include, err)
			}
		} else {
			if err := CopyFile(src, filepath.Join(stageDir, include)); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", include, err)
			}
		}
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

	// Clean dev files
	CleanDevFiles(stageDir)

	// Set permissions
	if err := ChmodAll(stageDir, 0777); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Create ZIP
	if !b.Quiet {
		ui.PrintInfo("Creating ZIP archive...")
	}
	zipPath := filepath.Join(b.BuildDir, fmt.Sprintf("%s-%s.zip", slug, b.Version.String()))
	if err := CreateZip(stageDir, zipPath, slug); err != nil {
		return fmt.Errorf("failed to create ZIP: %w", err)
	}

	if !b.Quiet {
		fmt.Println()
		ui.PrintSuccess("Created: %s", filepath.Base(zipPath))
	}

	return nil
}

// GetLibrarySlug returns the slug for this library.
func (b *LibraryBuilder) GetLibrarySlug() string {
	if b.Config == nil {
		return ""
	}
	if b.Config.Slug != "" {
		return b.Config.Slug
	}
	return SanitizeName(b.Config.Name)
}
