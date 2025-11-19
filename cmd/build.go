package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"wordsmith/internal/builder"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the WordPress plugin",
	Long:  "Build the WordPress plugin from the current directory",
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

		// Check for plugin.properties
		if !config.Exists(dir) {
			ui.PrintError("No plugin.properties found in current directory")
			ui.PrintInfo("Run 'wordsmith init' to create one")
			os.Exit(1)
		}

		b := builder.New(dir)
		b.Quiet = quiet
		if err := b.Build(); err != nil {
			ui.PrintError("Build failed: %v", err)
			os.Exit(1)
		}

		if quiet {
			ui.PrintSuccess("Build complete!")
		} else {
			fmt.Println()
			fmt.Println(ui.Divider())
			fmt.Println()
			ui.PrintSuccess("Build complete!")
			fmt.Println()
			ui.PrintInfo("Upload the ZIP file to WordPress via:")
			ui.PrintInfo("Plugins → Add New → Upload Plugin")
			fmt.Println()
		}
	},
}

func init() {
	buildCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	rootCmd.AddCommand(buildCmd)
}
