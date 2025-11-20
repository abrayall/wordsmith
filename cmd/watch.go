package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"wordsmith/internal/builder"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var watchCmd = &cobra.Command{
	Use:   "watch [build|deploy]",
	Short: "Watch for changes and build or deploy",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)

		mode := args[0]
		if mode != "build" && mode != "deploy" {
			ui.PrintError("Invalid mode. Use 'build' or 'deploy'")
			os.Exit(1)
		}

		dir, err := os.Getwd()
		if err != nil {
			ui.PrintError("Failed to get current directory: %v", err)
			os.Exit(1)
		}

		isTheme := config.ThemeExists(dir)
		isPlugin := config.PluginExists(dir)

		if !isTheme && !isPlugin {
			ui.PrintError("No plugin.properties or theme.properties found in current directory")
			os.Exit(1)
		}

		var mainFile string
		var includes []string
		var propsFile string

		if isTheme {
			cfg, err := config.LoadThemeProperties(dir)
			if err != nil {
				ui.PrintError("Failed to load theme.properties: %v", err)
				os.Exit(1)
			}
			mainFile = cfg.Main
			includes = cfg.Include
			propsFile = "theme.properties"
		} else {
			cfg, err := config.LoadPluginProperties(dir)
			if err != nil {
				ui.PrintError("Failed to load plugin.properties: %v", err)
				os.Exit(1)
			}
			mainFile = cfg.Main
			includes = cfg.Include
			propsFile = "plugin.properties"
		}

		// Run initial build/deploy
		ui.PrintInfo("Running initial %s...", mode)
		fmt.Println()

		var initialCmd *exec.Cmd
		if mode == "deploy" {
			initialCmd = exec.Command(os.Args[0], "deploy", "--quiet")
		} else {
			initialCmd = exec.Command(os.Args[0], "build", "--quiet")
		}
		initialCmd.Stdout = os.Stdout
		initialCmd.Stderr = os.Stderr
		initialCmd.Dir = dir
		initialCmd.Run()

		fmt.Println()
		ui.PrintInfo("Watching for changes (mode: %s)...", mode)
		ui.PrintInfo("Press Ctrl+C to stop")
		fmt.Println()

		lastMod := time.Now()
		debounce := 500 * time.Millisecond

		for {
			time.Sleep(500 * time.Millisecond)

			changed, newMod := hasChangesGeneric(dir, mainFile, includes, propsFile, lastMod)
			if !changed {
				continue
			}

			if time.Since(newMod) < debounce {
				continue
			}

			lastMod = time.Now()

			fmt.Println()
			ui.PrintInfo("Changes detected, running %s...", mode)
			fmt.Println()

			var buildCmd *exec.Cmd
			if mode == "deploy" {
				buildCmd = exec.Command(os.Args[0], "deploy", "--quiet")
			} else {
				buildCmd = exec.Command(os.Args[0], "build", "--quiet")
			}
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			buildCmd.Dir = dir
			buildCmd.Run()

			fmt.Println()
			ui.PrintInfo("Watching for changes...")
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

func hasChangesGeneric(dir string, mainFile string, includes []string, propsFile string, since time.Time) (bool, time.Time) {
	var latestMod time.Time
	changed := false

	checkFile := func(path string) {
		info, err := os.Stat(path)
		if err != nil {
			return
		}
		if info.ModTime().After(since) {
			changed = true
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
		}
	}

	mainPath := filepath.Join(dir, mainFile)
	checkFile(mainPath)

	for _, include := range includes {
		// Expand glob patterns
		expanded, err := builder.ExpandGlob(dir, include)
		if err != nil {
			continue
		}

		for _, relPath := range expanded {
			path := filepath.Join(dir, relPath)
			info, err := os.Stat(path)
			if err != nil {
				continue
			}

			if info.IsDir() {
				filepath.Walk(path, func(p string, i os.FileInfo, e error) error {
					if e != nil || i.IsDir() {
						return nil
					}
					if strings.HasPrefix(i.Name(), ".") {
						return nil
					}
					checkFile(p)
					return nil
				})
			} else {
				if !strings.HasPrefix(info.Name(), ".") {
					checkFile(path)
				}
			}
		}
	}

	propsPath := filepath.Join(dir, propsFile)
	checkFile(propsPath)

	return changed, latestMod
}
