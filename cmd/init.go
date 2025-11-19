package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new plugin.properties file",
	Long:  "Create a new plugin.properties file in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(ui.Banner())
		fmt.Println()

		dir, err := os.Getwd()
		if err != nil {
			ui.PrintError("Failed to get current directory: %v", err)
			os.Exit(1)
		}

		// Check if plugin.properties already exists
		if config.Exists(dir) {
			ui.PrintWarning("plugin.properties already exists")
			os.Exit(1)
		}

		// Create template
		template := `# Plugin Configuration
# Edit these values for your plugin

name=My Plugin
description=A WordPress plugin
author=Your Name
author_uri=https://example.com
license=GPL-2.0+
license_uri=https://www.gnu.org/licenses/gpl-2.0.html

# Main plugin file
main=my-plugin.php

# WordPress requirements
requires=5.0
requires_php=7.4

# Additional files/directories to include (comma-separated)
include=assets,templates,includes

# Text domain for internationalization
text_domain=my-plugin
domain_path=/languages
`

		path := filepath.Join(dir, "plugin.properties")
		if err := os.WriteFile(path, []byte(template), 0644); err != nil {
			ui.PrintError("Failed to create plugin.properties: %v", err)
			os.Exit(1)
		}

		ui.PrintSuccess("Created plugin.properties")
		ui.PrintInfo("Edit the file to configure your plugin, then run 'wordsmith build'")
		fmt.Println()
	},
}
