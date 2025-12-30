package cmd

import (
	"encoding/json"
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

			// Deploy plugin dependencies first
			networkName := instanceSlug + "-network"
			dependencies := b.GetPluginDependencies()
			if len(dependencies) > 0 {
				if err := deployPluginDependencies(dependencies, containerName, networkName, instanceSlug, quiet); err != nil {
					ui.PrintError("Failed to deploy plugin dependencies: %v", err)
					os.Exit(1)
				}
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

			// Deploy plugin settings
			if len(cfg.Settings) > 0 {
				if !quiet {
					ui.PrintInfo("Deploying settings...")
				}
				if err := deployPluginSettings(cfg.Settings, networkName, instanceSlug, quiet); err != nil {
					ui.PrintError("Failed to deploy settings: %v", err)
					os.Exit(1)
				}
			}
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

// deployPluginDependencies deploys all plugin dependencies before the main plugin
func deployPluginDependencies(deps []builder.PluginDependency, containerName, networkName, instanceSlug string, quiet bool) error {
	mysqlContainer := instanceSlug + "-mysql"

	for _, dep := range deps {
		if dep.IsWPOrg {
			// Install from WordPress.org
			if !quiet {
				ui.PrintInfo("  Installing dependency '%s' from WordPress.org...", dep.Slug)
			}

			installArgs := []string{"run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", instanceSlug + "-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST=" + mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "plugin", "install", dep.Slug, "--activate",
			}
			if dep.Version != "" {
				// Insert version before --activate
				installArgs = append(installArgs[:len(installArgs)-1], "--version="+dep.Version, "--activate")
			}

			installCmd := exec.Command("docker", installArgs...)
			if err := installCmd.Run(); err != nil {
				return fmt.Errorf("failed to install plugin '%s': %w", dep.Slug, err)
			}
		} else if dep.Path != "" {
			// Deploy built/resolved plugin via docker cp
			if !quiet {
				ui.PrintInfo("  Deploying dependency '%s'...", dep.Slug)
			}

			containerPath := fmt.Sprintf("/var/www/html/wp-content/plugins/%s", dep.Slug)

			// Remove old version
			dockerCmd := exec.Command("docker", "exec", containerName, "rm", "-rf", containerPath)
			dockerCmd.Run()

			// Copy new version
			dockerCmd = exec.Command("docker", "cp", dep.Path+"/.", containerName+":"+containerPath)
			if err := dockerCmd.Run(); err != nil {
				return fmt.Errorf("failed to deploy plugin '%s': %w", dep.Slug, err)
			}

			// Activate
			activateCmd := exec.Command("docker", "run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", instanceSlug+"-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "plugin", "activate", dep.Slug,
			)
			activateCmd.Run()
		}
	}

	return nil
}

// deployPluginSettings deploys plugin settings to the WordPress database
func deployPluginSettings(settings map[string]interface{}, networkName, instanceSlug string, quiet bool) error {
	if len(settings) == 0 {
		return nil
	}

	mysqlContainer := instanceSlug + "-mysql"

	for optionName, value := range settings {
		if !quiet {
			ui.PrintInfo("  Setting option '%s'...", optionName)
		}

		var updateArgs []string

		switch v := value.(type) {
		case string:
			// Simple string value
			updateArgs = []string{"run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", instanceSlug + "-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST=" + mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "option", "update", optionName, v,
			}
		case map[string]interface{}:
			// Complex nested value - use JSON format
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal setting '%s': %w", optionName, err)
			}

			updateArgs = []string{"run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", instanceSlug + "-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST=" + mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "option", "update", optionName, string(jsonBytes), "--format=json",
			}
		default:
			// Convert other types to string
			updateArgs = []string{"run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", instanceSlug + "-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST=" + mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "option", "update", optionName, fmt.Sprintf("%v", value),
			}
		}

		updateCmd := exec.Command("docker", updateArgs...)
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to set option '%s': %w", optionName, err)
		}
	}

	return nil
}
