package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var wordpressCmd = &cobra.Command{
	Use:   "wordpress",
	Short: "Manage WordPress development environment",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)
		cmd.Help()
	},
}

var startCmd = &cobra.Command{
	Use:   "start [file]",
	Short: "Start WordPress in Docker",
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

		// Determine which properties file to use
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
			// Check for wordpress.properties, then plugin/theme
			wpProps := filepath.Join(dir, "wordpress.properties")
			pluginProps := filepath.Join(dir, "plugin.properties")
			themeProps := filepath.Join(dir, "theme.properties")

			if config.FileExists(wpProps) {
				propsFile = wpProps
			} else if config.FileExists(pluginProps) {
				propsFile = pluginProps
			} else if config.FileExists(themeProps) {
				propsFile = themeProps
			} else {
				ui.PrintError("No properties file found")
				ui.PrintInfo("Create wordpress.properties, plugin.properties, or theme.properties")
				os.Exit(1)
			}
		}

		// Load configuration based on file type
		var wpConfig *config.WordPressConfig
		var dockerImage string = "wordpress:latest"
		var envName string

		filename := filepath.Base(propsFile)
		switch filename {
		case "wordpress.properties":
			wpConfig, err = config.LoadWordPressProperties(filepath.Dir(propsFile))
			if err != nil {
				ui.PrintError("Failed to load %s: %v", filename, err)
				os.Exit(1)
			}
			dockerImage = wpConfig.Image
			envName = wpConfig.Name
		case "plugin.properties":
			cfg, err := config.LoadPluginProperties(filepath.Dir(propsFile))
			if err != nil {
				ui.PrintError("Failed to load %s: %v", filename, err)
				os.Exit(1)
			}
			envName = cfg.Name
		case "theme.properties":
			cfg, err := config.LoadThemeProperties(filepath.Dir(propsFile))
			if err != nil {
				ui.PrintError("Failed to load %s: %v", filename, err)
				os.Exit(1)
			}
			envName = cfg.Name
		default:
			// Try to parse as wordpress.properties format
			wpConfig, err = config.LoadWordPressProperties(filepath.Dir(propsFile))
			if err != nil {
				ui.PrintError("Failed to load %s: %v", propsFile, err)
				os.Exit(1)
			}
			dockerImage = wpConfig.Image
			envName = wpConfig.Name
		}

		// If no name found, fall back to plugin/theme name, then directory name
		if envName == "" {
			if config.PluginExists(dir) {
				cfg, err := config.LoadPluginProperties(dir)
				if err == nil {
					envName = cfg.Name
				}
			} else if config.ThemeExists(dir) {
				cfg, err := config.LoadThemeProperties(dir)
				if err == nil {
					envName = cfg.Name
				}
			}
		}

		// Final fall back to directory name
		if envName == "" {
			envName = filepath.Base(dir)
		}

		pluginSlug := sanitizePluginName(envName)

		if !isCommandAvailable("docker") {
			ui.PrintError("Docker is not installed or not in PATH")
			ui.PrintInfo("Please install Docker: https://docs.docker.com/get-docker/")
			os.Exit(1)
		}

		if isContainerRunning(pluginSlug + "-wordpress") {
			ui.PrintWarning("WordPress is already running")
			wpPort := getContainerPort(pluginSlug + "-wordpress")
			if wpPort != "" {
				wpURL := "http://localhost:" + wpPort
				ui.PrintInfo("WordPress: %s", ui.Highlight(wpURL))
				ui.PrintInfo("Admin:     %s", ui.Highlight(wpURL+"/wp-admin"))
				fmt.Println()
				openBrowser(wpURL)
				openBrowser(wpURL + "/wp-admin")
			}
			os.Exit(0)
		}

		if containerExists(pluginSlug + "-wordpress") {
			ui.PrintInfo("Starting existing WordPress environment [%s]...", pluginSlug)
			exec.Command("docker", "start", pluginSlug+"-mysql").Run()
			exec.Command("docker", "start", pluginSlug+"-wordpress").Run()

			wpPort := getContainerPort(pluginSlug + "-wordpress")
			wpURL := fmt.Sprintf("http://localhost:%s", wpPort)

			fmt.Println()
			ui.PrintInfo("Waiting for WordPress to be ready...")
			waitForWordPress(wpURL, 60)

			if needsInstall(wpURL) {
				ui.PrintInfo("Installing WordPress...")
				port := 0
				fmt.Sscanf(wpPort, "%d", &port)
				if err := installWordPress(pluginSlug, port, envName); err != nil {
					ui.PrintWarning("Auto-install failed: %v", err)
				}
			}

			fmt.Println()
			ui.PrintSuccess("WordPress is running!")
			fmt.Println()
			ui.PrintInfo("WordPress: %s", ui.Highlight(wpURL))
			ui.PrintInfo("Admin:     %s", ui.Highlight(wpURL+"/wp-admin"))
			ui.PrintInfo("Username:  %s", ui.Highlight("admin"))
			ui.PrintInfo("Password:  %s", ui.Highlight("admin"))
			fmt.Println()
			openBrowser(wpURL)
			openBrowser(wpURL + "/wp-admin")
			os.Exit(0)
		}

		ui.PrintInfo("Starting WordPress environment [%s]...", pluginSlug)

		wpPort := findAvailablePort(8080, 8099)
		if wpPort == 0 {
			ui.PrintError("No available ports in range 8080-8099")
			os.Exit(1)
		}

		mysqlPort := findAvailablePort(3306, 3399)
		if mysqlPort == 0 {
			ui.PrintError("No available ports in range 3306-3399")
			os.Exit(1)
		}

		fmt.Printf("\033[38;2;59;130;246m• Using ports - WordPress: \033[0m%s\033[38;2;59;130;246m, MySQL: \033[0m%s\n", ui.Highlight(fmt.Sprintf("%d", wpPort)), ui.Highlight(fmt.Sprintf("%d", mysqlPort)))

		if err := startContainers(pluginSlug, dir, wpPort, mysqlPort, dockerImage); err != nil {
			ui.PrintError("Failed to start containers: %v", err)
			os.Exit(1)
		}

		fmt.Println()
		ui.PrintInfo("Waiting for WordPress to be ready...")

		wpURL := fmt.Sprintf("http://localhost:%d", wpPort)
		if !waitForWordPress(wpURL, 60) {
			ui.PrintWarning("WordPress took too long to start, but containers are running")
		}

		if needsInstall(wpURL) {
			ui.PrintInfo("Installing WordPress...")
			if err := installWordPress(pluginSlug, wpPort, envName); err != nil {
				ui.PrintWarning("Auto-install failed: %v", err)
				ui.PrintInfo("You may need to complete setup manually")
			}
		}

		// Install plugins and themes from wordpress.properties
		if wpConfig != nil {
			if len(wpConfig.Plugins) > 0 {
				fmt.Println()
				ui.PrintInfo("Installing plugins...")
				installPluginsAndThemes(pluginSlug, wpConfig)
			} else if len(wpConfig.Themes) > 0 {
				fmt.Println()
				ui.PrintInfo("Installing themes...")
				installPluginsAndThemes(pluginSlug, wpConfig)
			}
		}

		fmt.Println()
		ui.PrintSuccess("WordPress is running!")
		fmt.Println()
		ui.PrintInfo("WordPress: %s", ui.Highlight(wpURL))
		ui.PrintInfo("Admin:     %s", ui.Highlight(wpURL+"/wp-admin"))
		ui.PrintInfo("Username:  %s", ui.Highlight("admin"))
		ui.PrintInfo("Password:  %s", ui.Highlight("admin"))
		fmt.Println()

		openBrowser(wpURL)
		openBrowser(wpURL + "/wp-admin")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop WordPress Docker environment",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)

		var pluginSlug string

		if len(args) > 0 {
			// User provided instance name
			instance := args[0]
			if isContainerRunning(instance) || containerExists(instance) {
				pluginSlug = instance
			} else if isContainerRunning(instance+"-wordpress") || containerExists(instance+"-wordpress") {
				pluginSlug = instance
			} else {
				ui.PrintError("WordPress container '%s' not found", instance)
				os.Exit(1)
			}
		} else {
			// Get from properties files
			dir, err := os.Getwd()
			if err != nil {
				ui.PrintError("Failed to get current directory: %v", err)
				os.Exit(1)
			}

			isTheme := config.ThemeExists(dir)
			isPlugin := config.PluginExists(dir)

			if !isTheme && !isPlugin {
				ui.PrintError("No plugin.properties or theme.properties found in current directory")
				ui.PrintInfo("Specify instance name: wordsmith wordpress stop <name>")
				os.Exit(1)
			}

			var name string
			if isTheme {
				cfg, err := config.LoadThemeProperties(dir)
				if err != nil {
					ui.PrintError("Failed to load theme.properties: %v", err)
					os.Exit(1)
				}
				name = cfg.Name
			} else {
				cfg, err := config.LoadPluginProperties(dir)
				if err != nil {
					ui.PrintError("Failed to load plugin.properties: %v", err)
					os.Exit(1)
				}
				name = cfg.Name
			}

			pluginSlug = sanitizePluginName(name)
		}

		ui.PrintInfo("Stopping WordPress environment [%s]...", pluginSlug)

		stopContainer(pluginSlug + "-wordpress")
		stopContainer(pluginSlug + "-mysql")

		removeContainer(pluginSlug + "-wordpress")
		removeContainer(pluginSlug + "-mysql")

		ui.PrintSuccess("WordPress stopped")
		fmt.Println()
	},
}

var browseCmd = &cobra.Command{
	Use:   "browse [admin]",
	Short: "Open WordPress in browser",
	Run: func(cmd *cobra.Command, args []string) {
		pluginSlug := getProjectSlug()

		if !isContainerRunning(pluginSlug + "-wordpress") {
			ui.PrintError("WordPress is not running. Run 'wordsmith wordpress start' first")
			os.Exit(1)
		}

		wpPort := getContainerPort(pluginSlug + "-wordpress")
		if wpPort == "" {
			ui.PrintError("Could not determine WordPress port")
			os.Exit(1)
		}

		wpURL := "http://localhost:" + wpPort

		if len(args) > 0 && args[0] == "admin" {
			openBrowser(wpURL + "/wp-admin")
		} else {
			openBrowser(wpURL)
		}
	},
}

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List WordPress environments",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)

		// Get all wordsmith containers (filter by wordsmith.project label existence)
		dockerCmd := exec.Command("docker", "ps", "-a",
			"--filter", "label=wordsmith.project",
			"--format", "{{.Label \"wordsmith.project\"}}|{{.Label \"wordsmith.type\"}}|{{.Status}}|{{.Ports}}",
		)
		output, err := dockerCmd.Output()
		if err != nil {
			ui.PrintError("Failed to list containers: %v", err)
			os.Exit(1)
		}

		// Parse output and group by project
		projects := make(map[string]map[string]struct {
			status string
			port   string
		})

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) < 4 {
				continue
			}
			project := parts[0]
			containerType := parts[1]
			status := parts[2]
			ports := parts[3]

			if projects[project] == nil {
				projects[project] = make(map[string]struct {
					status string
					port   string
				})
			}

			// Extract port number
			port := ""
			if ports != "" {
				// Parse port like "0.0.0.0:8080->80/tcp"
				if idx := strings.Index(ports, ":"); idx != -1 {
					portPart := ports[idx+1:]
					if dashIdx := strings.Index(portPart, "-"); dashIdx != -1 {
						port = portPart[:dashIdx]
					}
				}
			}

			projects[project][containerType] = struct {
				status string
				port   string
			}{status: status, port: port}
		}

		if len(projects) == 0 {
			ui.PrintInfo("No WordPress environments found")
			return
		}

		// Column widths
		nameWidth := 20
		wpWidth := 20

		// Print header
		fmt.Println()
		fmt.Printf(" %s%s%s%s%s\n",
			ui.Highlight("NAME"), strings.Repeat(" ", nameWidth-4),
			ui.Highlight("WORDPRESS"), strings.Repeat(" ", wpWidth-9),
			ui.Highlight("MYSQL"))
		fmt.Printf(" \033[38;2;107;114;128m%s\033[0m\n", strings.Repeat("─", nameWidth+wpWidth+15))

		// Print each project
		for name, containers := range projects {
			wp := containers["wordpress"]
			mysql := containers["mysql"]

			var wpStatus string
			var mysqlStatus string
			var wpLen int
			var mysqlLen int

			if wp.status != "" && strings.Contains(wp.status, "Up") {
				if wp.port != "" {
					wpStatus = fmt.Sprintf("\033[32mrunning\033[0m \033[97m[%s]\033[0m", wp.port)
					wpLen = 7 + 3 + len(wp.port) // "running" + " []" + port
				} else {
					wpStatus = "\033[32mrunning\033[0m"
					wpLen = 7
				}
			} else {
				wpStatus = "\033[33mstopped\033[0m"
				wpLen = 7
			}

			if mysql.status != "" && strings.Contains(mysql.status, "Up") {
				if mysql.port != "" {
					mysqlStatus = fmt.Sprintf("\033[32mrunning\033[0m \033[97m[%s]\033[0m", mysql.port)
					mysqlLen = 7 + 3 + len(mysql.port)
				} else {
					mysqlStatus = "\033[32mrunning\033[0m"
					mysqlLen = 7
				}
			} else {
				mysqlStatus = "\033[33mstopped\033[0m"
				mysqlLen = 7
			}

			// Pad name to fit column
			namePadding := nameWidth - len(name)
			if namePadding < 1 {
				namePadding = 1
			}

			// Pad wp status to fit column
			wpPadding := wpWidth - wpLen
			if wpPadding < 1 {
				wpPadding = 1
			}

			_ = mysqlLen

			// Blue for name (same as UI Secondary color #3B82F6)
			nameColored := fmt.Sprintf("\033[38;2;59;130;246m%s\033[0m", name)

			fmt.Printf(" %s%s%s%s%s\n", nameColored, strings.Repeat(" ", namePadding), wpStatus, strings.Repeat(" ", wpPadding), mysqlStatus)
		}
		fmt.Println()
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete WordPress environment and all data",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)

		var pluginSlug string

		if len(args) > 0 {
			// User provided instance name
			pluginSlug = args[0]
		} else {
			// Get from properties files
			dir, err := os.Getwd()
			if err != nil {
				ui.PrintError("Failed to get current directory: %v", err)
				os.Exit(1)
			}

			isTheme := config.ThemeExists(dir)
			isPlugin := config.PluginExists(dir)

			if !isTheme && !isPlugin {
				ui.PrintError("No plugin.properties or theme.properties found in current directory")
				ui.PrintInfo("Specify instance name: wordsmith wordpress delete <name>")
				os.Exit(1)
			}

			var name string
			if isTheme {
				cfg, err := config.LoadThemeProperties(dir)
				if err != nil {
					ui.PrintError("Failed to load theme.properties: %v", err)
					os.Exit(1)
				}
				name = cfg.Name
			} else {
				cfg, err := config.LoadPluginProperties(dir)
				if err != nil {
					ui.PrintError("Failed to load plugin.properties: %v", err)
					os.Exit(1)
				}
				name = cfg.Name
			}

			pluginSlug = sanitizePluginName(name)
		}

		ui.PrintInfo("Deleting WordPress environment [%s]...", pluginSlug)

		stopContainer(pluginSlug + "-wordpress")
		stopContainer(pluginSlug + "-mysql")

		removeContainer(pluginSlug + "-wordpress")
		removeContainer(pluginSlug + "-mysql")

		exec.Command("docker", "volume", "rm", pluginSlug+"-wp").Run()
		exec.Command("docker", "volume", "rm", pluginSlug+"-db").Run()
		exec.Command("docker", "network", "rm", pluginSlug+"-network").Run()

		ui.PrintSuccess("WordPress environment deleted")
		fmt.Println()
	},
}

func init() {
	startCmd.Flags().BoolP("quiet", "q", false, "Suppress header output")
	wordpressCmd.AddCommand(startCmd)
	wordpressCmd.AddCommand(stopCmd)
	wordpressCmd.AddCommand(psCmd)
	wordpressCmd.AddCommand(browseCmd)
	wordpressCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(wordpressCmd)
}

func sanitizePluginName(name string) string {
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

// getProjectSlug returns the sanitized project slug from plugin.properties or theme.properties
func getProjectSlug() string {
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

	var name string
	if isTheme {
		cfg, err := config.LoadThemeProperties(dir)
		if err != nil {
			ui.PrintError("Failed to load theme.properties: %v", err)
			os.Exit(1)
		}
		name = cfg.Name
	} else {
		cfg, err := config.LoadPluginProperties(dir)
		if err != nil {
			ui.PrintError("Failed to load plugin.properties: %v", err)
			os.Exit(1)
		}
		name = cfg.Name
	}

	return sanitizePluginName(name)
}

func findAvailablePort(start, end int) int {
	for port := start; port <= end; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

func isContainerRunning(name string) bool {
	cmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=%s", name))
	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

func containerExists(name string) bool {
	cmd := exec.Command("docker", "ps", "-aq", "-f", fmt.Sprintf("name=%s", name))
	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

func getContainerPort(name string) string {
	cmd := exec.Command("docker", "port", name, "80")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func startContainers(pluginSlug, projectDir string, wpPort, mysqlPort int, dockerImage string) error {
	networkName := pluginSlug + "-network"
	exec.Command("docker", "network", "create", networkName).Run()

	mysqlCmd := exec.Command("docker", "run", "-d",
		"--name", pluginSlug+"-mysql",
		"--network", networkName,
		"-p", fmt.Sprintf("%d:3306", mysqlPort),
		"-e", "MYSQL_DATABASE=wordpress",
		"-e", "MYSQL_USER=wordpress",
		"-e", "MYSQL_PASSWORD=wordpress",
		"-e", "MYSQL_ROOT_PASSWORD=rootpassword",
		"-v", pluginSlug+"-db:/var/lib/mysql",
		"--label", "wordsmith.type=mysql",
		"--label", "wordsmith.project="+pluginSlug,
		"mysql:8.0",
	)
	if err := mysqlCmd.Run(); err != nil {
		return fmt.Errorf("failed to start MySQL: %w", err)
	}

	wpCmd := exec.Command("docker", "run", "-d",
		"--name", pluginSlug+"-wordpress",
		"--network", networkName,
		"-p", fmt.Sprintf("%d:80", wpPort),
		"-e", "WORDPRESS_DB_HOST="+pluginSlug+"-mysql",
		"-e", "WORDPRESS_DB_USER=wordpress",
		"-e", "WORDPRESS_DB_PASSWORD=wordpress",
		"-e", "WORDPRESS_DB_NAME=wordpress",
		"-v", pluginSlug+"-wp:/var/www/html",
		"--label", "wordsmith.type=wordpress",
		"--label", "wordsmith.project="+pluginSlug,
		dockerImage,
	)
	_ = projectDir
	if err := wpCmd.Run(); err != nil {
		return fmt.Errorf("failed to start WordPress: %w", err)
	}

	return nil
}

func stopContainer(name string) {
	exec.Command("docker", "stop", name).Run()
}

func removeContainer(name string) {
	exec.Command("docker", "rm", name).Run()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch {
	case isCommandAvailable("open"):
		cmd = exec.Command("open", url)
	case isCommandAvailable("xdg-open"):
		cmd = exec.Command("xdg-open", url)
	case isCommandAvailable("start"):
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	cmd.Run()
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func waitForWordPress(url string, timeoutSeconds int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < timeoutSeconds; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func needsInstall(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return true
	}
	defer resp.Body.Close()

	if resp.Request.URL.Path == "/wp-admin/install.php" ||
	   strings.Contains(resp.Request.URL.String(), "install.php") {
		return true
	}
	return false
}

func installWordPress(pluginSlug string, port int, pluginName string) error {
	containerName := pluginSlug + "-wordpress"
	networkName := pluginSlug + "-network"

	mysqlContainer := pluginSlug + "-mysql"
	for i := 0; i < 30; i++ {
		checkCmd := exec.Command("docker", "exec", mysqlContainer, "mysqladmin", "ping", "-h", "localhost", "-uroot", "-prootpassword", "--silent")
		if err := checkCmd.Run(); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	installCmd := exec.Command("docker", "run", "--rm",
		"--network", networkName,
		"--user", "33:33",
		"-v", pluginSlug+"-wp:/var/www/html",
		"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
		"-e", "WORDPRESS_DB_USER=wordpress",
		"-e", "WORDPRESS_DB_PASSWORD=wordpress",
		"-e", "WORDPRESS_DB_NAME=wordpress",
		"wordpress:cli",
		"wp", "core", "install",
		"--url=http://localhost:"+fmt.Sprintf("%d", port),
		"--title=WordPress "+pluginName,
		"--admin_user=admin",
		"--admin_password=admin",
		"--admin_email=admin@localhost.com",
		"--skip-email",
	)
	output, err := installCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}

	activateCmd := exec.Command("docker", "run", "--rm",
		"--network", networkName,
		"--user", "33:33",
		"-v", pluginSlug+"-wp:/var/www/html",
		"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
		"-e", "WORDPRESS_DB_USER=wordpress",
		"-e", "WORDPRESS_DB_PASSWORD=wordpress",
		"-e", "WORDPRESS_DB_NAME=wordpress",
		"wordpress:cli",
		"wp", "plugin", "activate", pluginSlug,
	)
	activateCmd.Run()

	_ = containerName

	return nil
}

// installPluginsAndThemes installs plugins and themes from wordpress.properties
func installPluginsAndThemes(pluginSlug string, wpConfig *config.WordPressConfig) {
	networkName := pluginSlug + "-network"
	mysqlContainer := pluginSlug + "-mysql"

	// Install plugins
	for _, plugin := range wpConfig.Plugins {
		var installCmd *exec.Cmd

		if plugin.URI != "" {
			// Install from URI (URL or file path)
			if strings.HasPrefix(plugin.URI, "http://") || strings.HasPrefix(plugin.URI, "https://") {
				// Install from URL
				ui.PrintInfo("  Installing plugin '%s' from URL...", plugin.Slug)
				installCmd = exec.Command("docker", "run", "--rm",
					"--network", networkName,
					"--user", "33:33",
					"-v", pluginSlug+"-wp:/var/www/html",
					"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
					"-e", "WORDPRESS_DB_USER=wordpress",
					"-e", "WORDPRESS_DB_PASSWORD=wordpress",
					"-e", "WORDPRESS_DB_NAME=wordpress",
					"wordpress:cli",
					"wp", "plugin", "install", plugin.URI,
				)
			} else {
				// Install from local file path - copy to container first
				ui.PrintInfo("  Installing plugin '%s' from file...", plugin.Slug)
				containerPath := fmt.Sprintf("/tmp/%s.zip", plugin.Slug)

				// Copy file to container
				copyCmd := exec.Command("docker", "cp", plugin.URI, pluginSlug+"-wordpress:"+containerPath)
				if err := copyCmd.Run(); err != nil {
					ui.PrintWarning("  Failed to copy plugin file: %v", err)
					continue
				}

				installCmd = exec.Command("docker", "run", "--rm",
					"--network", networkName,
					"--user", "33:33",
					"-v", pluginSlug+"-wp:/var/www/html",
					"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
					"-e", "WORDPRESS_DB_USER=wordpress",
					"-e", "WORDPRESS_DB_PASSWORD=wordpress",
					"-e", "WORDPRESS_DB_NAME=wordpress",
					"wordpress:cli",
					"wp", "plugin", "install", containerPath,
				)
			}
		} else {
			// Install from WordPress.org
			ui.PrintInfo("  Installing plugin '%s'...", plugin.Slug)
			installArgs := []string{"run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", pluginSlug + "-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST=" + mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "plugin", "install", plugin.Slug,
			}
			if plugin.Version != "" {
				installArgs = append(installArgs, "--version="+plugin.Version)
			}
			installCmd = exec.Command("docker", installArgs...)
		}

		if err := installCmd.Run(); err != nil {
			ui.PrintWarning("  Failed to install plugin '%s': %v", plugin.Slug, err)
			continue
		}

		// Activate if requested
		if plugin.Active {
			activateCmd := exec.Command("docker", "run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", pluginSlug+"-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "plugin", "activate", plugin.Slug,
			)
			activateCmd.Run()
		}
	}

	// Install themes
	for _, theme := range wpConfig.Themes {
		var installCmd *exec.Cmd

		if theme.URI != "" {
			// Install from URI (URL or file path)
			if strings.HasPrefix(theme.URI, "http://") || strings.HasPrefix(theme.URI, "https://") {
				// Install from URL
				ui.PrintInfo("  Installing theme '%s' from URL...", theme.Slug)
				installCmd = exec.Command("docker", "run", "--rm",
					"--network", networkName,
					"--user", "33:33",
					"-v", pluginSlug+"-wp:/var/www/html",
					"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
					"-e", "WORDPRESS_DB_USER=wordpress",
					"-e", "WORDPRESS_DB_PASSWORD=wordpress",
					"-e", "WORDPRESS_DB_NAME=wordpress",
					"wordpress:cli",
					"wp", "theme", "install", theme.URI,
				)
			} else {
				// Install from local file path
				ui.PrintInfo("  Installing theme '%s' from file...", theme.Slug)
				containerPath := fmt.Sprintf("/tmp/%s.zip", theme.Slug)

				// Copy file to container
				copyCmd := exec.Command("docker", "cp", theme.URI, pluginSlug+"-wordpress:"+containerPath)
				if err := copyCmd.Run(); err != nil {
					ui.PrintWarning("  Failed to copy theme file: %v", err)
					continue
				}

				installCmd = exec.Command("docker", "run", "--rm",
					"--network", networkName,
					"--user", "33:33",
					"-v", pluginSlug+"-wp:/var/www/html",
					"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
					"-e", "WORDPRESS_DB_USER=wordpress",
					"-e", "WORDPRESS_DB_PASSWORD=wordpress",
					"-e", "WORDPRESS_DB_NAME=wordpress",
					"wordpress:cli",
					"wp", "theme", "install", containerPath,
				)
			}
		} else {
			// Install from WordPress.org
			ui.PrintInfo("  Installing theme '%s'...", theme.Slug)
			installArgs := []string{"run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", pluginSlug + "-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST=" + mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "theme", "install", theme.Slug,
			}
			if theme.Version != "" {
				installArgs = append(installArgs, "--version="+theme.Version)
			}
			installCmd = exec.Command("docker", installArgs...)
		}

		if err := installCmd.Run(); err != nil {
			ui.PrintWarning("  Failed to install theme '%s': %v", theme.Slug, err)
			continue
		}

		// Activate if requested
		if theme.Active {
			activateCmd := exec.Command("docker", "run", "--rm",
				"--network", networkName,
				"--user", "33:33",
				"-v", pluginSlug+"-wp:/var/www/html",
				"-e", "WORDPRESS_DB_HOST="+mysqlContainer,
				"-e", "WORDPRESS_DB_USER=wordpress",
				"-e", "WORDPRESS_DB_PASSWORD=wordpress",
				"-e", "WORDPRESS_DB_NAME=wordpress",
				"wordpress:cli",
				"wp", "theme", "activate", theme.Slug,
			)
			activateCmd.Run()
		}
	}
}
