package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"wordsmith/internal/builder"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [file]",
	Short: "Build and deploy plugin or theme to WordPress",
	Args:  cobra.MaximumNArgs(1),
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

		// Determine which properties file to use for WordPress instance
		var propsFile string
		if len(args) > 0 {
			// User provided a specific file
			propsFile = args[0]
			if !filepath.IsAbs(propsFile) {
				propsFile = filepath.Join(dir, propsFile)
			}
			if !config.FileExists(propsFile) {
				ui.PrintError("Properties file not found: %s", propsFile)
				os.Exit(1)
			}
		} else {
			// Check for wordpress.properties first, then use plugin/theme name
			wpProps := filepath.Join(dir, "wordpress.properties")
			if config.FileExists(wpProps) {
				propsFile = wpProps
			}
		}

		// Determine WordPress instance name
		var instanceName string
		if propsFile != "" {
			filename := filepath.Base(propsFile)
			if filename == "wordpress.properties" {
				wpConfig, err := config.LoadWordPressProperties(filepath.Dir(propsFile))
				if err != nil {
					ui.PrintError("Failed to load %s: %v", filename, err)
					os.Exit(1)
				}
				instanceName = wpConfig.Name
			}
		}

		// Fall back to plugin/theme name
		if instanceName == "" {
			if isTheme {
				cfg, err := config.LoadThemeProperties(dir)
				if err == nil {
					instanceName = cfg.Name
				}
			} else if isPlugin {
				cfg, err := config.LoadPluginProperties(dir)
				if err == nil {
					instanceName = cfg.Name
				}
			}
		}

		instanceSlug := sanitizeForDocker(instanceName)

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

			// Check if WordPress container is running
			containerName := instanceSlug + "-wordpress"
			if !isContainerRunning(containerName) {
				if !quiet {
					ui.PrintInfo("WordPress is not running, starting it...")
					fmt.Println()
				}
				var startArgs []string
				startArgs = append(startArgs, "wordpress", "start")
				if propsFile != "" {
					startArgs = append(startArgs, propsFile)
				}
				startArgs = append(startArgs, "--quiet")
				startCmd := exec.Command(os.Args[0], startArgs...)
				startCmd.Stdout = os.Stdout
				startCmd.Stderr = os.Stderr
				startCmd.Dir = dir
				if err := startCmd.Run(); err != nil {
					ui.PrintError("Failed to start WordPress: %v", err)
					os.Exit(1)
				}
				fmt.Println()
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

			// Deploy all parent themes first (grandparent, then parent, etc.)
			parentThemes := b.GetAllParentThemes()
			for _, parent := range parentThemes {
				parentSlug := sanitizeForDocker(parent.Name)

				if !quiet {
					ui.PrintInfo("Deploying parent theme '%s'...", parent.Name)
				}

				parentContainerPath := fmt.Sprintf("/var/www/html/wp-content/themes/%s", parentSlug)

				dockerCmd := exec.Command("docker", "exec", containerName, "rm", "-rf", parentContainerPath)
				dockerCmd.Run()

				dockerCmd = exec.Command("docker", "cp", parent.Path+"/.", containerName+":"+parentContainerPath)
				if err := dockerCmd.Run(); err != nil {
					ui.PrintError("Failed to deploy parent theme '%s': %v", parent.Name, err)
					os.Exit(1)
				}
			}

			// Deploy child theme
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
			networkName := instanceSlug + "-network"
			activateCmd := exec.Command("docker", "run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", instanceSlug+"-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST="+instanceSlug+"-mysql",
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

			// Check if WordPress container is running
			containerName := instanceSlug + "-wordpress"
			if !isContainerRunning(containerName) {
				if !quiet {
					ui.PrintInfo("WordPress is not running, starting it...")
					fmt.Println()
				}
				var startArgs []string
				startArgs = append(startArgs, "wordpress", "start")
				if propsFile != "" {
					startArgs = append(startArgs, propsFile)
				}
				startArgs = append(startArgs, "--quiet")
				startCmd := exec.Command(os.Args[0], startArgs...)
				startCmd.Stdout = os.Stdout
				startCmd.Stderr = os.Stderr
				startCmd.Dir = dir
				if err := startCmd.Run(); err != nil {
					ui.PrintError("Failed to start WordPress: %v", err)
					os.Exit(1)
				}
				fmt.Println()
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
			networkName := instanceSlug + "-network"
			activateCmd := exec.Command("docker", "run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", instanceSlug+"-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST="+instanceSlug+"-mysql",
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
