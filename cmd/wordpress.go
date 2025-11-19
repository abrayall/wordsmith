package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var wordpressCmd = &cobra.Command{
	Use:   "wordpress",
	Short: "Manage WordPress development environment",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start WordPress in Docker",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)

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

		pluginSlug := sanitizePluginName(cfg.Name)

		if !isCommandAvailable("docker") {
			ui.PrintError("Docker is not installed or not in PATH")
			ui.PrintInfo("Please install Docker: https://docs.docker.com/get-docker/")
			os.Exit(1)
		}

		if isContainerRunning(pluginSlug + "-wordpress") {
			ui.PrintWarning("WordPress is already running")
			wpPort := getContainerPort(pluginSlug + "-wordpress")
			if wpPort != "" {
				ui.PrintInfo("WordPress: http://localhost:%s", wpPort)
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
				if err := installWordPress(pluginSlug, port, cfg.Name); err != nil {
					ui.PrintWarning("Auto-install failed: %v", err)
				}
			}

			fmt.Println()
			ui.PrintSuccess("WordPress is running!")
			fmt.Println()
			ui.PrintInfo("WordPress: %s", wpURL)
			ui.PrintInfo("Admin:     %s/wp-admin", wpURL)
			ui.PrintInfo("Username:  admin")
			ui.PrintInfo("Password:  admin")
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

		ui.PrintInfo("Using ports - WordPress: %d, MySQL: %d", wpPort, mysqlPort)

		if err := startContainers(pluginSlug, dir, wpPort, mysqlPort); err != nil {
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
			if err := installWordPress(pluginSlug, wpPort, cfg.Name); err != nil {
				ui.PrintWarning("Auto-install failed: %v", err)
				ui.PrintInfo("You may need to complete setup manually")
			}

			ui.PrintInfo("Deploying plugin...")
			deployCmd := exec.Command(os.Args[0], "deploy", "--quiet")
			deployCmd.Dir = dir
			deployCmd.Run()
		}

		fmt.Println()
		ui.PrintSuccess("WordPress is running!")
		fmt.Println()
		ui.PrintInfo("WordPress: %s", wpURL)
		ui.PrintInfo("Admin:     %s/wp-admin", wpURL)
		ui.PrintInfo("Username:  admin")
		ui.PrintInfo("Password:  admin")
		fmt.Println()

		openBrowser(wpURL)
		openBrowser(wpURL + "/wp-admin")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop WordPress Docker environment",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)

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

		pluginSlug := sanitizePluginName(cfg.Name)

		ui.PrintInfo("Stopping WordPress environment [%s]...", pluginSlug)

		stopContainer(pluginSlug + "-wordpress")
		stopContainer(pluginSlug + "-mysql")

		removeContainer(pluginSlug + "-wordpress")
		removeContainer(pluginSlug + "-mysql")

		ui.PrintSuccess("WordPress stopped")
		fmt.Println()
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete WordPress environment and all data",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)

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

		pluginSlug := sanitizePluginName(cfg.Name)

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
	wordpressCmd.AddCommand(startCmd)
	wordpressCmd.AddCommand(stopCmd)
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

func startContainers(pluginSlug, projectDir string, wpPort, mysqlPort int) error {
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
		"wordpress:latest",
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
