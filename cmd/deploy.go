package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"wordsmith/internal/builder"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Build and deploy plugin or theme to WordPress",
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

		isTheme := config.ThemeExists(dir)
		isPlugin := config.PluginExists(dir)

		if !isTheme && !isPlugin {
			ui.PrintError("No plugin.properties or theme.properties found in current directory")
			os.Exit(1)
		}

		var slug string
		var containerPath string
		var stageDir string

		if isTheme {
			cfg, err := config.LoadThemeProperties(dir)
			if err != nil {
				ui.PrintError("Failed to load theme.properties: %v", err)
				os.Exit(1)
			}

			slug = sanitizeForDocker(cfg.Name)
			containerName := slug + "-wordpress"

			if !isContainerRunning(containerName) {
				ui.PrintError("WordPress is not running. Run 'wordsmith wordpress start' first")
				os.Exit(1)
			}

			b := builder.NewThemeBuilder(dir)
			b.Quiet = quiet
			if err := b.Build(); err != nil {
				ui.PrintError("Build failed: %v", err)
				os.Exit(1)
			}

			if !quiet {
				fmt.Println()
				ui.PrintInfo("Deploying theme to WordPress...")
			}

			stageDir = fmt.Sprintf("%s/build/work/stage", dir)
			containerPath = fmt.Sprintf("/var/www/html/wp-content/themes/%s", slug)

			dockerCmd := exec.Command("docker", "exec", containerName, "rm", "-rf", containerPath)
			dockerCmd.Run()

			dockerCmd = exec.Command("docker", "cp", stageDir+"/.", containerName+":"+containerPath)
			if err := dockerCmd.Run(); err != nil {
				ui.PrintError("Failed to deploy: %v", err)
				os.Exit(1)
			}

			// Activate theme
			networkName := slug + "-network"
			activateCmd := exec.Command("docker", "run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", slug+"-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST="+slug+"-mysql",
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "theme", "activate", slug,
			)
			activateCmd.Run()
		} else {
			cfg, err := config.LoadPluginProperties(dir)
			if err != nil {
				ui.PrintError("Failed to load plugin.properties: %v", err)
				os.Exit(1)
			}

			slug = sanitizeForDocker(cfg.Name)
			containerName := slug + "-wordpress"

			if !isContainerRunning(containerName) {
				ui.PrintError("WordPress is not running. Run 'wordsmith wordpress start' first")
				os.Exit(1)
			}

			b := builder.New(dir)
			b.Quiet = quiet
			if err := b.Build(); err != nil {
				ui.PrintError("Build failed: %v", err)
				os.Exit(1)
			}

			if !quiet {
				fmt.Println()
				ui.PrintInfo("Deploying to WordPress...")
			}

			stageDir = fmt.Sprintf("%s/build/work/stage", dir)
			containerPath = fmt.Sprintf("/var/www/html/wp-content/plugins/%s", slug)

			dockerCmd := exec.Command("docker", "exec", containerName, "rm", "-rf", containerPath)
			dockerCmd.Run()

			dockerCmd = exec.Command("docker", "cp", stageDir+"/.", containerName+":"+containerPath)
			if err := dockerCmd.Run(); err != nil {
				ui.PrintError("Failed to deploy: %v", err)
				os.Exit(1)
			}

			// Activate plugin
			networkName := slug + "-network"
			activateCmd := exec.Command("docker", "run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", slug+"-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST="+slug+"-mysql",
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "plugin", "activate", slug,
			)
			activateCmd.Run()
		}

		if quiet {
			ui.PrintSuccess("Deployed to WordPress!")
		} else {
			fmt.Println()
			fmt.Println(ui.Divider())
			fmt.Println()
			ui.PrintSuccess("Deployed to WordPress!")
			fmt.Println()
		}
	},
}

func init() {
	deployCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	rootCmd.AddCommand(deployCmd)
}

func sanitizeForDocker(name string) string {
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	var clean string
	for _, ch := range result {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			clean += string(ch)
		}
	}
	return clean
}
