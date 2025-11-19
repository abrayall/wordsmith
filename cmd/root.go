package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"wordsmith/internal/ui"
)

// Version is set by ldflags during build
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "wordsmith",
	Short: "WordPress plugin build tool",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Long = ui.Banner() + "\n" + ui.VersionLine(Version) + "\n\n  A beautiful CLI tool for building WordPress plugins"
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wordsmith %s\n", Version)
	},
}
