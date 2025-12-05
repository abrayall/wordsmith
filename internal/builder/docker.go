package builder

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"wordsmith/internal/config"
	"wordsmith/internal/ui"
	"wordsmith/internal/version"
)

// DockerBuilder builds a Docker image with WordPress, plugins, and themes
type DockerBuilder struct {
	SourceDir string
	BuildDir  string
	WorkDir   string
	Config    *config.PluginConfig
	ThemeConfig *config.ThemeConfig
	WPConfig  *config.WordPressConfig
	Version   string
	Quiet     bool
	IsTheme   bool
}

// NewDockerBuilder creates a new DockerBuilder
func NewDockerBuilder(sourceDir string) *DockerBuilder {
	buildDir := filepath.Join(sourceDir, "build")
	return &DockerBuilder{
		SourceDir: sourceDir,
		BuildDir:  buildDir,
		WorkDir:   filepath.Join(buildDir, "docker"),
	}
}

// Build builds the Docker image
func (d *DockerBuilder) Build() error {
	// Determine if this is a plugin or theme
	d.IsTheme = config.ThemeExists(d.SourceDir)
	isPlugin := config.PluginExists(d.SourceDir)

	if !d.IsTheme && !isPlugin {
		return fmt.Errorf("no plugin.properties or theme.properties found")
	}

	// Build the plugin/theme first
	if !d.Quiet {
		ui.PrintInfo("Building plugin/theme...")
	}

	var slug string
	if d.IsTheme {
		b := NewThemeBuilder(d.SourceDir)
		b.Quiet = d.Quiet
		if err := b.Build(); err != nil {
			return fmt.Errorf("failed to build theme: %w", err)
		}
		d.ThemeConfig = b.Config
		d.Version = b.Version.String()
		slug = b.GetThemeSlug()
	} else {
		b := New(d.SourceDir)
		b.Quiet = d.Quiet
		if err := b.Build(); err != nil {
			return fmt.Errorf("failed to build plugin: %w", err)
		}
		d.Config = b.Config
		d.Version = b.Version.String()
		slug = b.GetPluginSlug()
	}

	// Load wordpress.properties if it exists
	if config.WordPressExists(d.SourceDir) {
		wpConfig, err := config.LoadWordPressProperties(d.SourceDir)
		if err != nil {
			return fmt.Errorf("failed to load wordpress.properties: %w", err)
		}
		d.WPConfig = wpConfig
	}

	// Create docker work directory
	if err := os.RemoveAll(d.WorkDir); err != nil {
		return fmt.Errorf("failed to clean docker work directory: %w", err)
	}
	if err := os.MkdirAll(d.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create docker work directory: %w", err)
	}

	// Create plugins and themes directories
	pluginsDir := filepath.Join(d.WorkDir, "plugins")
	themesDir := filepath.Join(d.WorkDir, "themes")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return fmt.Errorf("failed to create themes directory: %w", err)
	}

	// Copy the built plugin/theme zip to the work directory
	if !d.Quiet {
		ui.PrintInfo("Preparing Docker build context...")
	}

	builtZip, err := d.findBuiltZip()
	if err != nil {
		return err
	}

	if d.IsTheme {
		if err := copyFile(builtZip, filepath.Join(themesDir, filepath.Base(builtZip))); err != nil {
			return fmt.Errorf("failed to copy theme zip: %w", err)
		}
	} else {
		if err := copyFile(builtZip, filepath.Join(pluginsDir, filepath.Base(builtZip))); err != nil {
			return fmt.Errorf("failed to copy plugin zip: %w", err)
		}
	}

	// Collect plugins and themes to activate
	var pluginsToActivate []string
	var themesToActivate []string

	// Add the main plugin/theme
	if d.IsTheme {
		themesToActivate = append(themesToActivate, slug)
	} else {
		pluginsToActivate = append(pluginsToActivate, slug)
	}

	// Process additional plugins/themes from wordpress.properties
	if d.WPConfig != nil {
		for _, plugin := range d.WPConfig.Plugins {
			if plugin.Active {
				pluginsToActivate = append(pluginsToActivate, plugin.Slug)
			}
		}
		for _, theme := range d.WPConfig.Themes {
			if theme.Active {
				themesToActivate = append(themesToActivate, theme.Slug)
			}
		}
	}

	// Generate Dockerfile
	if !d.Quiet {
		ui.PrintInfo("Generating Dockerfile...")
	}
	if err := d.generateDockerfile(); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Generate entrypoint script
	if err := d.generateEntrypoint(pluginsToActivate, themesToActivate); err != nil {
		return fmt.Errorf("failed to generate entrypoint script: %w", err)
	}

	// Build Docker image
	imageTag := fmt.Sprintf("%s:v%s", slug, d.Version)
	if !d.Quiet {
		ui.PrintInfo("Building Docker image: %s", imageTag)
	}

	buildCmd := exec.Command("docker", "build", "-t", imageTag, d.WorkDir)
	if !d.Quiet {
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
	}

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	if !d.Quiet {
		fmt.Println()
		ui.PrintSuccess("Docker image built: %s", imageTag)
		fmt.Println()
		ui.PrintInfo("Run with: docker run -p 8080:80 %s", imageTag)
	}

	return nil
}

func (d *DockerBuilder) findBuiltZip() (string, error) {
	entries, err := os.ReadDir(d.BuildDir)
	if err != nil {
		return "", fmt.Errorf("failed to read build directory: %w", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".zip") {
			return filepath.Join(d.BuildDir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no zip file found in build directory")
}

func (d *DockerBuilder) generateDockerfile() error {
	baseImage := "wordpress:latest"
	if d.WPConfig != nil && d.WPConfig.Image != "" {
		baseImage = d.WPConfig.Image
	}

	var dockerfileContent strings.Builder

	dockerfileContent.WriteString(fmt.Sprintf("FROM %s\n\n", baseImage))

	// Install unzip and wp-cli
	dockerfileContent.WriteString("# Install dependencies\n")
	dockerfileContent.WriteString("RUN apt-get update && apt-get install -y unzip less mariadb-client && rm -rf /var/lib/apt/lists/*\n\n")

	// Install WP-CLI
	dockerfileContent.WriteString("# Install WP-CLI\n")
	dockerfileContent.WriteString("RUN curl -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar \\\n")
	dockerfileContent.WriteString("    && chmod +x wp-cli.phar \\\n")
	dockerfileContent.WriteString("    && mv wp-cli.phar /usr/local/bin/wp\n\n")

	// Copy plugins
	dockerfileContent.WriteString("# Copy plugins\n")
	dockerfileContent.WriteString("COPY plugins/ /tmp/plugins/\n\n")

	// Copy themes
	dockerfileContent.WriteString("# Copy themes\n")
	dockerfileContent.WriteString("COPY themes/ /tmp/themes/\n\n")

	// Copy and set entrypoint
	dockerfileContent.WriteString("# Copy entrypoint script\n")
	dockerfileContent.WriteString("COPY entrypoint.sh /usr/local/bin/wordsmith-entrypoint.sh\n")
	dockerfileContent.WriteString("RUN chmod +x /usr/local/bin/wordsmith-entrypoint.sh\n\n")

	// Set entrypoint
	dockerfileContent.WriteString("ENTRYPOINT [\"/usr/local/bin/wordsmith-entrypoint.sh\"]\n")
	dockerfileContent.WriteString("CMD [\"apache2-foreground\"]\n")

	return os.WriteFile(filepath.Join(d.WorkDir, "Dockerfile"), []byte(dockerfileContent.String()), 0644)
}

func (d *DockerBuilder) generateEntrypoint(pluginsToActivate, themesToActivate []string) error {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")

	script.WriteString("# Run the original WordPress entrypoint first\n")
	script.WriteString("docker-entrypoint.sh apache2-foreground &\n")
	script.WriteString("APACHE_PID=$!\n\n")

	script.WriteString("# Wait for WordPress to be ready\n")
	script.WriteString("echo 'Waiting for WordPress to be ready...'\n")
	script.WriteString("sleep 10\n\n")

	script.WriteString("# Check if WordPress is installed, if not wait more\n")
	script.WriteString("until wp core is-installed --allow-root 2>/dev/null; do\n")
	script.WriteString("    echo 'Waiting for WordPress installation...'\n")
	script.WriteString("    sleep 5\n")
	script.WriteString("done\n\n")

	// Install and activate plugins from zip files
	script.WriteString("# Install plugins from zip files\n")
	script.WriteString("for zip in /tmp/plugins/*.zip; do\n")
	script.WriteString("    if [ -f \"$zip\" ]; then\n")
	script.WriteString("        echo \"Installing plugin: $zip\"\n")
	script.WriteString("        wp plugin install \"$zip\" --allow-root || true\n")
	script.WriteString("    fi\n")
	script.WriteString("done\n\n")

	// Install and activate themes from zip files
	script.WriteString("# Install themes from zip files\n")
	script.WriteString("for zip in /tmp/themes/*.zip; do\n")
	script.WriteString("    if [ -f \"$zip\" ]; then\n")
	script.WriteString("        echo \"Installing theme: $zip\"\n")
	script.WriteString("        wp theme install \"$zip\" --allow-root || true\n")
	script.WriteString("    fi\n")
	script.WriteString("done\n\n")

	// Activate plugins
	if len(pluginsToActivate) > 0 {
		script.WriteString("# Activate plugins\n")
		for _, plugin := range pluginsToActivate {
			script.WriteString(fmt.Sprintf("echo 'Activating plugin: %s'\n", plugin))
			script.WriteString(fmt.Sprintf("wp plugin activate %s --allow-root || true\n", plugin))
		}
		script.WriteString("\n")
	}

	// Activate theme (only one can be active)
	if len(themesToActivate) > 0 {
		script.WriteString("# Activate theme\n")
		// Activate the last one in the list (typically the main theme or explicitly active one)
		theme := themesToActivate[len(themesToActivate)-1]
		script.WriteString(fmt.Sprintf("echo 'Activating theme: %s'\n", theme))
		script.WriteString(fmt.Sprintf("wp theme activate %s --allow-root || true\n", theme))
		script.WriteString("\n")
	}

	script.WriteString("echo 'WordPress setup complete!'\n\n")

	script.WriteString("# Wait for Apache to exit\n")
	script.WriteString("wait $APACHE_PID\n")

	return os.WriteFile(filepath.Join(d.WorkDir, "entrypoint.sh"), []byte(script.String()), 0755)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}

// SiteDockerBuilder builds a Docker image for a site with all plugins and themes
type SiteDockerBuilder struct {
	SourceDir        string
	BuildDir         string
	WorkDir          string
	SiteConfig       *config.SiteConfig
	Quiet            bool
	WordsmithVersion string
}

// NewSiteDockerBuilder creates a new SiteDockerBuilder
func NewSiteDockerBuilder(sourceDir string, siteConfig *config.SiteConfig) *SiteDockerBuilder {
	buildDir := filepath.Join(sourceDir, "build")
	return &SiteDockerBuilder{
		SourceDir:  sourceDir,
		BuildDir:   buildDir,
		WorkDir:    filepath.Join(buildDir, "docker"),
		SiteConfig: siteConfig,
	}
}

// Build builds the Docker image for the site
func (s *SiteDockerBuilder) Build() error {
	if !s.Quiet {
		ui.PrintInfo("Building site: %s", s.SiteConfig.Name)
	}

	// Create docker work directory
	if err := os.RemoveAll(s.WorkDir); err != nil {
		return fmt.Errorf("failed to clean docker work directory: %w", err)
	}
	if err := os.MkdirAll(s.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create docker work directory: %w", err)
	}

	// Create plugins and themes directories
	pluginsDir := filepath.Join(s.WorkDir, "plugins")
	themesDir := filepath.Join(s.WorkDir, "themes")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return fmt.Errorf("failed to create themes directory: %w", err)
	}

	var pluginsToActivate []string
	var themesToActivate []string

	// Build and copy local plugins
	for _, plugin := range s.SiteConfig.LocalPlugins {
		if plugin.NeedsBuild {
			if !s.Quiet {
				ui.PrintInfo("  Building plugin: %s", plugin.Slug)
			}
			b := New(plugin.Path)
			b.Quiet = true
			if err := b.Build(); err != nil {
				ui.PrintWarning("  Failed to build plugin %s: %v", plugin.Slug, err)
				continue
			}

			// Find the built zip
			zipFile, err := findBuiltZipInDir(filepath.Join(plugin.Path, "build"))
			if err != nil {
				ui.PrintWarning("  No zip found for plugin %s: %v", plugin.Slug, err)
				continue
			}

			// Copy to docker work dir
			if err := copyFile(zipFile, filepath.Join(pluginsDir, filepath.Base(zipFile))); err != nil {
				return fmt.Errorf("failed to copy plugin zip: %w", err)
			}

			// Get actual slug from builder
			pluginsToActivate = append(pluginsToActivate, b.GetPluginSlug())
		} else if plugin.IsZip {
			// Copy zip directly
			if !s.Quiet {
				ui.PrintInfo("  Copying plugin: %s", plugin.Slug)
			}
			if err := copyFile(plugin.Path, filepath.Join(pluginsDir, filepath.Base(plugin.Path))); err != nil {
				return fmt.Errorf("failed to copy plugin zip: %w", err)
			}
			if plugin.Active {
				pluginsToActivate = append(pluginsToActivate, plugin.Slug)
			}
		}
	}

	// Build and copy local themes
	for _, theme := range s.SiteConfig.LocalThemes {
		if theme.NeedsBuild {
			if !s.Quiet {
				ui.PrintInfo("  Building theme: %s", theme.Slug)
			}
			b := NewThemeBuilder(theme.Path)
			b.Quiet = true
			if err := b.Build(); err != nil {
				ui.PrintWarning("  Failed to build theme %s: %v", theme.Slug, err)
				continue
			}

			// Find the built zip
			zipFile, err := findBuiltZipInDir(filepath.Join(theme.Path, "build"))
			if err != nil {
				ui.PrintWarning("  No zip found for theme %s: %v", theme.Slug, err)
				continue
			}

			// Copy to docker work dir
			if err := copyFile(zipFile, filepath.Join(themesDir, filepath.Base(zipFile))); err != nil {
				return fmt.Errorf("failed to copy theme zip: %w", err)
			}

			if theme.Active {
				themesToActivate = append(themesToActivate, b.GetThemeSlug())
			}
		} else if theme.IsZip {
			// Copy zip directly
			if !s.Quiet {
				ui.PrintInfo("  Copying theme: %s", theme.Slug)
			}
			if err := copyFile(theme.Path, filepath.Join(themesDir, filepath.Base(theme.Path))); err != nil {
				return fmt.Errorf("failed to copy theme zip: %w", err)
			}
			if theme.Active {
				themesToActivate = append(themesToActivate, theme.Slug)
			}
		}
	}

	// Download and copy plugins from site.properties (URLs, GitHub, WordPress.org)
	for _, plugin := range s.SiteConfig.Plugins {
		if plugin.URI != "" {
			// Resolve GitHub URLs to release asset URLs
			uri, err := config.ResolveGitHubURL(plugin.URI, plugin.Slug, plugin.Version)
			if err != nil {
				ui.PrintWarning("  Failed to resolve plugin URL %s: %v", plugin.Slug, err)
				continue
			}
			if !s.Quiet {
				ui.PrintInfo("  Downloading plugin: %s", plugin.Slug)
			}
			zipPath := filepath.Join(pluginsDir, plugin.Slug+".zip")
			if err := downloadFile(uri, zipPath); err != nil {
				ui.PrintWarning("  Failed to download plugin %s: %v", plugin.Slug, err)
				continue
			}
		} else if plugin.Slug != "" {
			// WordPress.org plugin - download from API
			if !s.Quiet {
				ui.PrintInfo("  Downloading plugin: %s", plugin.Slug)
			}
			uri := fmt.Sprintf("https://downloads.wordpress.org/plugin/%s.zip", plugin.Slug)
			if plugin.Version != "" {
				uri = fmt.Sprintf("https://downloads.wordpress.org/plugin/%s.%s.zip", plugin.Slug, plugin.Version)
			}
			zipPath := filepath.Join(pluginsDir, plugin.Slug+".zip")
			if err := downloadFile(uri, zipPath); err != nil {
				ui.PrintWarning("  Failed to download plugin %s: %v", plugin.Slug, err)
				continue
			}
		}
		if plugin.Active {
			pluginsToActivate = append(pluginsToActivate, plugin.Slug)
		}
	}

	// Download and copy themes from site.properties (URLs, GitHub, WordPress.org)
	for _, theme := range s.SiteConfig.Themes {
		if theme.URI != "" {
			// Resolve GitHub URLs to release asset URLs
			uri, err := config.ResolveGitHubURL(theme.URI, theme.Slug, theme.Version)
			if err != nil {
				ui.PrintWarning("  Failed to resolve theme URL %s: %v", theme.Slug, err)
				continue
			}
			if !s.Quiet {
				ui.PrintInfo("  Downloading theme: %s", theme.Slug)
			}
			zipPath := filepath.Join(themesDir, theme.Slug+".zip")
			if err := downloadFile(uri, zipPath); err != nil {
				ui.PrintWarning("  Failed to download theme %s: %v", theme.Slug, err)
				continue
			}
		} else if theme.Slug != "" {
			// WordPress.org theme - download from API
			if !s.Quiet {
				ui.PrintInfo("  Downloading theme: %s", theme.Slug)
			}
			uri := fmt.Sprintf("https://downloads.wordpress.org/theme/%s.zip", theme.Slug)
			if theme.Version != "" {
				uri = fmt.Sprintf("https://downloads.wordpress.org/theme/%s.%s.zip", theme.Slug, theme.Version)
			}
			zipPath := filepath.Join(themesDir, theme.Slug+".zip")
			if err := downloadFile(uri, zipPath); err != nil {
				ui.PrintWarning("  Failed to download theme %s: %v", theme.Slug, err)
				continue
			}
		}
		if theme.Active {
			themesToActivate = append(themesToActivate, theme.Slug)
		}
	}

	// Get version from git
	slug := sanitizeName(s.SiteConfig.Name)
	ver, err := version.GetFromGit(s.SourceDir)
	if err != nil {
		ver = &version.Version{Major: 0, Minor: 1, Maintenance: "0"}
	}
	siteVersion := ver.String()
	imageTag := fmt.Sprintf("%s:v%s", slug, siteVersion)

	// Generate Dockerfile
	if !s.Quiet {
		ui.PrintInfo("  Generating Dockerfile...")
	}
	if err := s.generateDockerfile(); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Generate entrypoint script
	if err := s.generateEntrypoint(pluginsToActivate, themesToActivate, siteVersion); err != nil {
		return fmt.Errorf("failed to generate entrypoint script: %w", err)
	}

	if !s.Quiet {
		ui.PrintInfo("  Building Docker image: %s", imageTag)
	}

	latestTag := fmt.Sprintf("%s:latest", slug)
	buildCmd := exec.Command("docker", "build", "--platform", "linux/amd64", "-t", latestTag, s.WorkDir)
	if !s.Quiet {
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
	}

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Tag with version
	if !s.Quiet {
		ui.PrintInfo("  Tagging image: %s", imageTag)
	}
	tagCmd := exec.Command("docker", "tag", latestTag, imageTag)
	if err := tagCmd.Run(); err != nil {
		return fmt.Errorf("failed to tag Docker image: %w", err)
	}

	if !s.Quiet {
		fmt.Println()
		ui.PrintSuccess("Docker image built: %s", imageTag)
		fmt.Println()
		ui.PrintInfo("Run with: docker run -p 8080:80 %s", imageTag)
	}

	return nil
}

func (s *SiteDockerBuilder) generateDockerfile() error {
	baseImage := "wordpress:latest"
	if s.SiteConfig.Image != "" {
		baseImage = s.SiteConfig.Image
	}

	var dockerfileContent strings.Builder

	dockerfileContent.WriteString(fmt.Sprintf("FROM %s\n\n", baseImage))

	// Install unzip and wp-cli
	dockerfileContent.WriteString("# Install dependencies\n")
	dockerfileContent.WriteString("RUN apt-get update && apt-get install -y unzip less mariadb-client && rm -rf /var/lib/apt/lists/*\n\n")

	// Install WP-CLI
	dockerfileContent.WriteString("# Install WP-CLI\n")
	dockerfileContent.WriteString("RUN curl -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar \\\n")
	dockerfileContent.WriteString("    && chmod +x wp-cli.phar \\\n")
	dockerfileContent.WriteString("    && mv wp-cli.phar /usr/local/bin/wp\n\n")

	// Copy plugins
	dockerfileContent.WriteString("# Copy plugins\n")
	dockerfileContent.WriteString("COPY plugins/ /tmp/plugins/\n\n")

	// Copy themes
	dockerfileContent.WriteString("# Copy themes\n")
	dockerfileContent.WriteString("COPY themes/ /tmp/themes/\n\n")

	// Copy and set entrypoint
	dockerfileContent.WriteString("# Copy entrypoint script\n")
	dockerfileContent.WriteString("COPY entrypoint.sh /usr/local/bin/wordsmith-entrypoint.sh\n")
	dockerfileContent.WriteString("RUN sed -i 's/\\r$//' /usr/local/bin/wordsmith-entrypoint.sh && chmod +x /usr/local/bin/wordsmith-entrypoint.sh\n\n")

	// Set entrypoint - use /bin/bash explicitly to avoid exec format errors
	dockerfileContent.WriteString("ENTRYPOINT [\"/bin/bash\", \"/usr/local/bin/wordsmith-entrypoint.sh\"]\n")
	dockerfileContent.WriteString("CMD [\"apache2-foreground\"]\n")

	return os.WriteFile(filepath.Join(s.WorkDir, "Dockerfile"), []byte(dockerfileContent.String()), 0644)
}

func (s *SiteDockerBuilder) generateEntrypoint(pluginsToActivate, themesToActivate []string, siteVersion string) error {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")

	// Print wordsmith logo
	script.WriteString("echo ''\n")
	script.WriteString("echo ' █   █ █▀▀█ █▀▀█ █▀▀▄ █▀▀ █▀▄▀█ ▀█▀ ▀▀█▀▀ █  █'\n")
	script.WriteString("echo ' █▄█▄█ █  █ █▄▄▀ █  █ ▀▀█ █ ▀ █  █    █   █▀▀█'\n")
	script.WriteString("echo '  ▀ ▀  ▀▀▀▀ ▀ ▀▀ ▀▀▀  ▀▀▀ ▀   ▀ ▀▀▀   ▀   ▀  ▀'\n")
	script.WriteString("echo ''\n")

	// Print launch banner
	script.WriteString(fmt.Sprintf("echo 'Launching site %s v%s [powered by wordsmith v%s]...'\n\n",
		s.SiteConfig.Name, siteVersion, s.WordsmithVersion))

	script.WriteString("# Run the original WordPress entrypoint first\n")
	script.WriteString("docker-entrypoint.sh apache2-foreground &\n")
	script.WriteString("APACHE_PID=$!\n\n")

	script.WriteString("# Wait for WordPress files to be ready\n")
	script.WriteString("echo 'Waiting for WordPress to be ready...'\n")
	script.WriteString("until [ -f /var/www/html/wp-config.php ]; do\n")
	script.WriteString("    sleep 2\n")
	script.WriteString("done\n\n")

	script.WriteString("# Wait for WordPress to be accessible\n")
	script.WriteString("echo 'Waiting for WordPress to be accessible...'\n")
	script.WriteString("until curl -s -o /dev/null -w '%{http_code}' http://localhost/wp-admin/install.php | grep -q '200\\|302'; do\n")
	script.WriteString("    sleep 2\n")
	script.WriteString("done\n\n")

	// Use WORDPRESS_SITEURL, then WORDPRESS_HOME, then site.properties URL as fallback
	siteURL := s.SiteConfig.URL
	if siteURL == "" {
		siteURL = "http://localhost"
	}
	urlExpr := fmt.Sprintf("${WORDPRESS_SITEURL:-${WORDPRESS_HOME:-%s}}", siteURL)

	script.WriteString("# Install WordPress if not already installed\n")
	script.WriteString("if ! wp core is-installed --allow-root 2>/dev/null; then\n")
	script.WriteString("    echo 'Installing WordPress...'\n")
	script.WriteString(fmt.Sprintf("    wp core install --url=\"%s\" --title=\"%s\" --admin_user=\"${WORDPRESS_ADMIN_USER:-admin}\" --admin_password=\"${WORDPRESS_ADMIN_PASSWORD:-admin}\" --admin_email=\"${WORDPRESS_ADMIN_EMAIL:-admin@example.com}\" --skip-email --allow-root\n", urlExpr, s.SiteConfig.Name))
	script.WriteString("fi\n\n")

	script.WriteString("# Always update site URL and title to match config\n")
	script.WriteString("echo 'Updating site configuration...'\n")
	script.WriteString(fmt.Sprintf("wp option update siteurl \"%s\" --allow-root\n", urlExpr))
	script.WriteString(fmt.Sprintf("wp option update home \"%s\" --allow-root\n", urlExpr))
	script.WriteString(fmt.Sprintf("wp option update blogname \"%s\" --allow-root\n\n", s.SiteConfig.Name))

	// Install plugins from zip files
	script.WriteString("# Install plugins from zip files\n")
	script.WriteString("for zip in /tmp/plugins/*.zip; do\n")
	script.WriteString("    if [ -f \"$zip\" ]; then\n")
	script.WriteString("        echo \"Installing plugin: $zip\"\n")
	script.WriteString("        wp plugin install \"$zip\" --allow-root || true\n")
	script.WriteString("    fi\n")
	script.WriteString("done\n\n")

	// Install themes from zip files
	script.WriteString("# Install themes from zip files\n")
	script.WriteString("for zip in /tmp/themes/*.zip; do\n")
	script.WriteString("    if [ -f \"$zip\" ]; then\n")
	script.WriteString("        echo \"Installing theme: $zip\"\n")
	script.WriteString("        wp theme install \"$zip\" --allow-root || true\n")
	script.WriteString("    fi\n")
	script.WriteString("done\n\n")

	// Activate plugins
	if len(pluginsToActivate) > 0 {
		script.WriteString("# Activate plugins\n")
		for _, plugin := range pluginsToActivate {
			script.WriteString(fmt.Sprintf("echo 'Activating plugin: %s'\n", plugin))
			script.WriteString(fmt.Sprintf("wp plugin activate %s --allow-root || true\n", plugin))
		}
		script.WriteString("\n")
	}

	// Activate theme (only one can be active)
	if len(themesToActivate) > 0 {
		script.WriteString("# Activate theme\n")
		theme := themesToActivate[len(themesToActivate)-1]
		script.WriteString(fmt.Sprintf("echo 'Activating theme: %s'\n", theme))
		script.WriteString(fmt.Sprintf("wp theme activate %s --allow-root || true\n", theme))
		script.WriteString("\n")
	}

	script.WriteString(fmt.Sprintf("echo 'Launched site %s!'\n\n", s.SiteConfig.Name))

	script.WriteString("# Wait for Apache to exit\n")
	script.WriteString("wait $APACHE_PID\n")

	return os.WriteFile(filepath.Join(s.WorkDir, "entrypoint.sh"), []byte(script.String()), 0755)
}

// findBuiltZipInDir finds the first zip file in a directory
func findBuiltZipInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".zip") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no zip file found in %s", dir)
}

// sanitizeName sanitizes a name for use as a Docker image tag
func sanitizeName(name string) string {
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	// Remove any characters that aren't alphanumeric or dashes
	var clean strings.Builder
	for _, r := range result {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			clean.WriteRune(r)
		}
	}
	return clean.String()
}

// downloadFile downloads a file from a URL to a local path, using cache if available
func downloadFile(url string, destPath string) error {
	// Try to use cache
	cacheDir := "/tmp/wordsmith/cache"
	cacheFile := ""

	// Create cache directory if possible
	if err := os.MkdirAll(cacheDir, 0755); err == nil {
		// Use URL hash as cache filename
		cacheFile = filepath.Join(cacheDir, sanitizeFilename(url)+".zip")

		// Check if file exists in cache
		if _, err := os.Stat(cacheFile); err == nil {
			// Copy from cache
			return copyFile(cacheFile, destPath)
		}
	}

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Save to cache if cache dir is available
	if cacheFile != "" {
		copyFile(destPath, cacheFile) // ignore errors, caching is optional
	}

	return nil
}

// sanitizeFilename creates a safe filename from a URL
func sanitizeFilename(url string) string {
	// Remove protocol
	name := strings.TrimPrefix(url, "https://")
	name = strings.TrimPrefix(name, "http://")
	// Replace unsafe characters
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "&", "_")
	name = strings.ReplaceAll(name, "=", "_")
	// Trim to reasonable length
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}
