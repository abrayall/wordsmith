package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"wordsmith/internal/builder"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Manage WordPress site projects",
	Long:  "Manage WordPress site projects with plugins and themes",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)
		cmd.Help()
	},
}

var siteStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start WordPress site in Docker",
	Long:  "Start a WordPress development environment for the site",
	Run: func(cmd *cobra.Command, args []string) {
		// Delegate to wordpress start command
		startCmd.Run(cmd, args)
	},
}

var siteStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop WordPress site",
	Long:  "Stop the WordPress development environment for the site",
	Run: func(cmd *cobra.Command, args []string) {
		// Delegate to wordpress stop command
		stopCmd.Run(cmd, args)
	},
}

var siteDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete WordPress site environment",
	Long:  "Delete the WordPress development environment and all data for the site",
	Run: func(cmd *cobra.Command, args []string) {
		// Delegate to wordpress delete command
		deleteCmd.Run(cmd, args)
	},
}

var siteBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build site plugins and themes",
	Long:  "Build all plugins and themes in the site",
	Run: func(cmd *cobra.Command, args []string) {
		quiet, _ := cmd.Flags().GetBool("quiet")
		if !quiet {
			ui.PrintHeader(Version)
		}

		dir, err := os.Getwd()
		if err != nil {
			ui.PrintError("Failed to get current directory: %v", err)
			os.Exit(1)
		}

		if !config.SiteExists(dir) {
			ui.PrintError("No site.properties found in current directory")
			os.Exit(1)
		}

		siteConfig, err := config.LoadSiteProperties(dir)
		if err != nil {
			ui.PrintError("Failed to load site.properties: %v", err)
			os.Exit(1)
		}

		if !quiet {
			ui.PrintInfo("Building site: %s", siteConfig.Name)
			fmt.Println()
		}

		// Build local plugins
		for _, plugin := range siteConfig.LocalPlugins {
			if plugin.NeedsBuild {
				if !quiet {
					ui.PrintInfo("Building plugin: %s", plugin.Slug)
				}
				b := builder.New(plugin.Path)
				b.Quiet = quiet
				if err := b.Build(); err != nil {
					ui.PrintWarning("Failed to build plugin %s: %v", plugin.Slug, err)
				}
			}
		}

		// Build local themes
		for _, theme := range siteConfig.LocalThemes {
			if theme.NeedsBuild {
				if !quiet {
					ui.PrintInfo("Building theme: %s", theme.Slug)
				}
				b := builder.NewThemeBuilder(theme.Path)
				b.Quiet = quiet
				if err := b.Build(); err != nil {
					ui.PrintWarning("Failed to build theme %s: %v", theme.Slug, err)
				}
			}
		}

		if !quiet {
			fmt.Println()
			ui.PrintSuccess("Site build complete!")
		}
	},
}

var siteBuildDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Build a Docker image for the site",
	Long:  "Build a Docker image containing WordPress with all site plugins and themes pre-installed",
	Run: func(cmd *cobra.Command, args []string) {
		quiet, _ := cmd.Flags().GetBool("quiet")
		if !quiet {
			ui.PrintHeader(Version)
		}

		dir, err := os.Getwd()
		if err != nil {
			ui.PrintError("Failed to get current directory: %v", err)
			os.Exit(1)
		}

		if !config.SiteExists(dir) {
			ui.PrintError("No site.properties found in current directory")
			os.Exit(1)
		}

		siteConfig, err := config.LoadSiteProperties(dir)
		if err != nil {
			ui.PrintError("Failed to load site.properties: %v", err)
			os.Exit(1)
		}

		d := builder.NewSiteDockerBuilder(dir, siteConfig)
		d.Quiet = quiet
		d.WordsmithVersion = Version
		if err := d.Build(); err != nil {
			ui.PrintError("Docker build failed: %v", err)
			os.Exit(1)
		}
	},
}

var siteInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new site",
	Long:  "Create a new site with site.properties and directory structure",
	Run: func(cmd *cobra.Command, args []string) {
		quiet, _ := cmd.Flags().GetBool("quiet")
		if !quiet {
			ui.PrintHeader(Version)
		}

		dir, err := os.Getwd()
		if err != nil {
			ui.PrintError("Failed to get current directory: %v", err)
			os.Exit(1)
		}

		// Check if site.properties already exists
		if config.SiteExists(dir) {
			ui.PrintError("site.properties already exists in current directory")
			os.Exit(1)
		}

		// Get site name from directory name or flag
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = filepath.Base(dir)
		}

		// Create site.properties
		siteProps := fmt.Sprintf(`name: %s
description: A WordPress site
# url: https://example.com

# Docker image (defaults to wordpress:latest)
# image: wordpress:6.4-php8.2

# Plugins from WordPress.org, GitHub, or URLs
plugins:
  # - akismet
  # - https://github.com/owner/repo

# Themes from WordPress.org, GitHub, or URLs
themes:
  # - flavor
`, name)

		if err := os.WriteFile(filepath.Join(dir, "site.properties"), []byte(siteProps), 0644); err != nil {
			ui.PrintError("Failed to create site.properties: %v", err)
			os.Exit(1)
		}

		// Create plugins and themes directories
		if err := os.MkdirAll(filepath.Join(dir, "plugins"), 0755); err != nil {
			ui.PrintError("Failed to create plugins directory: %v", err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Join(dir, "themes"), 0755); err != nil {
			ui.PrintError("Failed to create themes directory: %v", err)
			os.Exit(1)
		}

		if !quiet {
			ui.PrintSuccess("Site initialized: %s", name)
			fmt.Println()
			ui.PrintInfo("Created:")
			ui.PrintInfo("  site.properties")
			ui.PrintInfo("  plugins/")
			ui.PrintInfo("  themes/")
			fmt.Println()
			ui.PrintInfo("Add plugins to plugins/ and themes to themes/")
			ui.PrintInfo("Run 'wordsmith site start' to start WordPress")
		}
	},
}

func init() {
	// Site command flags
	siteStartCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	siteStopCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	siteDeleteCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	siteBuildCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	siteBuildDockerCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	siteInitCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	siteInitCmd.Flags().StringP("name", "n", "", "Site name")

	// Add subcommands
	siteBuildCmd.AddCommand(siteBuildDockerCmd)
	siteCmd.AddCommand(siteStartCmd)
	siteCmd.AddCommand(siteStopCmd)
	siteCmd.AddCommand(siteDeleteCmd)
	siteCmd.AddCommand(siteBuildCmd)
	siteCmd.AddCommand(siteInitCmd)

	rootCmd.AddCommand(siteCmd)
}
