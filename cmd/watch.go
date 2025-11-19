package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
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

		if !config.Exists(dir) {
			ui.PrintError("No plugin.properties found in current directory")
			os.Exit(1)
		}

		cfg, err := config.LoadProperties(dir)
		if err != nil {
			ui.PrintError("Failed to load plugin.properties: %v", err)
			os.Exit(1)
		}

		ui.PrintInfo("Watching for changes (mode: %s)...", mode)
		ui.PrintInfo("Press Ctrl+C to stop")
		fmt.Println()

		lastMod := time.Now()
		debounce := 500 * time.Millisecond

		for {
			time.Sleep(500 * time.Millisecond)

			changed, newMod := hasChanges(dir, cfg, lastMod)
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

func hasChanges(dir string, cfg *config.PluginConfig, since time.Time) (bool, time.Time) {
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

	mainFile := filepath.Join(dir, cfg.Main)
	checkFile(mainFile)

	for _, include := range cfg.Include {
		path := filepath.Join(dir, include)
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
			checkFile(path)
		}
	}

	propsFile := filepath.Join(dir, "plugin.properties")
	checkFile(propsFile)

	return changed, latestMod
}
