package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"wordsmith/internal/config"
	"wordsmith/internal/ui"
)

var (
	initName        string
	initDescription string
	initAuthor      string
	initAuthorURI   string
	initThemeType   string
	initTemplate    string
	initTemplateURI string
)

var initCmd = &cobra.Command{
	Use:   "init [plugin|theme]",
	Short: "Initialize a new WordPress plugin or theme",
	Long:  "Create a new plugin or theme with all necessary files and directories",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader(Version)

		dir, err := os.Getwd()
		if err != nil {
			ui.PrintError("Failed to get current directory: %v", err)
			os.Exit(1)
		}

		// Determine type from args (default to plugin)
		buildType := "plugin"
		if len(args) > 0 {
			if args[0] == "theme" {
				buildType = "theme"
			} else if args[0] != "plugin" {
				ui.PrintError("Invalid type: %s (use 'plugin' or 'theme')", args[0])
				os.Exit(1)
			}
		}

		// Check if any flags were provided (non-interactive mode)
		interactive := initName == "" && initDescription == "" && initAuthor == "" && initAuthorURI == "" && initThemeType == ""

		if buildType == "theme" {
			initTheme(dir, interactive)
		} else {
			initPlugin(dir, interactive)
		}
	},
}

func init() {
	initCmd.Flags().StringVar(&initName, "name", "", "Plugin/theme name")
	initCmd.Flags().StringVar(&initDescription, "description", "", "Plugin/theme description")
	initCmd.Flags().StringVar(&initAuthor, "author", "", "Author name")
	initCmd.Flags().StringVar(&initAuthorURI, "author-uri", "", "Author website URL")
	initCmd.Flags().StringVar(&initThemeType, "type", "", "Theme type: block, classic, hybrid, or child")
	initCmd.Flags().StringVar(&initTemplate, "template", "", "Parent theme name (for child themes)")
	initCmd.Flags().StringVar(&initTemplateURI, "template-uri", "", "Parent theme URL or path (for child themes)")
}

func initPlugin(dir string, interactive bool) {
	// Get default name from directory
	defaultName := formatName(filepath.Base(dir))

	var name, description, author, authorURI string

	if interactive {
		reader := bufio.NewReader(os.Stdin)

		// Ask questions
		ui.PrintInfo("Let's set up your WordPress plugin!")
		fmt.Println()

		name = prompt(reader, "Plugin name", defaultName)
		description = prompt(reader, "Description", "A WordPress plugin")
		author = prompt(reader, "Author", "")
		if author != "" {
			authorURI = prompt(reader, "Author website", "")
		}

		fmt.Println()
	} else {
		// Non-interactive mode - use flags or defaults
		name = initName
		if name == "" {
			name = defaultName
		}
		description = initDescription
		if description == "" {
			description = "A WordPress plugin"
		}
		author = initAuthor
		authorURI = initAuthorURI
	}

	// If current directory is not empty, create subdirectory
	if !isEmptyDir(dir) {
		slug := sanitizeName(name)
		newDir := filepath.Join(dir, slug)
		if err := os.MkdirAll(newDir, 0755); err != nil {
			ui.PrintError("Failed to create directory %s: %v", slug, err)
			os.Exit(1)
		}
		dir = newDir
	}

	// Check if plugin.properties already exists
	if config.PluginExists(dir) {
		ui.PrintWarning("plugin.properties already exists")
		os.Exit(1)
	}

	// Generate slug from name
	slug := sanitizeName(name)
	mainFile := slug + ".php"
	textDomain := slug

	// Create plugin.properties
	var props []string
	props = append(props, "# Plugin Configuration")
	props = append(props, "")
	props = append(props, fmt.Sprintf("name=%s", name))
	props = append(props, fmt.Sprintf("description=%s", description))
	if author != "" {
		props = append(props, fmt.Sprintf("author=%s", author))
	}
	if authorURI != "" {
		props = append(props, fmt.Sprintf("author-uri=%s", authorURI))
	}
	props = append(props, "license=GPL-2.0+")
	props = append(props, "license-uri=https://www.gnu.org/licenses/gpl-2.0.html")
	props = append(props, "")
	props = append(props, "# Main plugin file")
	props = append(props, fmt.Sprintf("main=%s", mainFile))
	props = append(props, "")
	props = append(props, "# WordPress requirements")
	props = append(props, "requires=5.0")
	props = append(props, "requires-php=7.4")
	props = append(props, "")
	props = append(props, "# Files to include (supports wildcards)")
	props = append(props, "include=includes,assets,languages")
	props = append(props, "")
	props = append(props, "# Files to exclude")
	props = append(props, "exclude=node_modules,tests,.*")
	props = append(props, "")
	props = append(props, "# Text domain for internationalization")
	props = append(props, fmt.Sprintf("text-domain=%s", textDomain))
	props = append(props, "domain-path=/languages")
	props = append(props, "")

	propsContent := strings.Join(props, "\n")
	propsPath := filepath.Join(dir, "plugin.properties")
	if err := os.WriteFile(propsPath, []byte(propsContent), 0644); err != nil {
		ui.PrintError("Failed to create plugin.properties: %v", err)
		os.Exit(1)
	}

	// Create main plugin file
	mainContent := generateMainPluginFile(name, description, author, authorURI, slug)
	mainPath := filepath.Join(dir, mainFile)
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		ui.PrintError("Failed to create %s: %v", mainFile, err)
		os.Exit(1)
	}

	// Create directories
	dirs := []string{"includes", "assets", "assets/css", "assets/js", "languages"}
	for _, d := range dirs {
		path := filepath.Join(dir, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			ui.PrintWarning("Failed to create directory %s: %v", d, err)
		}
	}

	// Create a basic CSS file
	cssContent := fmt.Sprintf("/**\n * %s Styles\n */\n", name)
	cssPath := filepath.Join(dir, "assets", "css", slug+".css")
	os.WriteFile(cssPath, []byte(cssContent), 0644)

	// Create a basic JS file
	jsContent := fmt.Sprintf("/**\n * %s Scripts\n */\n\n(function($) {\n    'use strict';\n\n    $(document).ready(function() {\n        // Your code here\n    });\n\n})(jQuery);\n", name)
	jsPath := filepath.Join(dir, "assets", "js", slug+".js")
	os.WriteFile(jsPath, []byte(jsContent), 0644)

	// Create readme.txt
	readmeContent := generateReadme(name, description, author, slug)
	readmePath := filepath.Join(dir, "readme.txt")
	os.WriteFile(readmePath, []byte(readmeContent), 0644)

	// Create .gitignore
	gitignoreContent := "build/\n"
	gitignorePath := filepath.Join(dir, ".gitignore")
	os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)

	// Print success
	ui.PrintSuccess("Created plugin: %s", name)
	fmt.Println()
	ui.PrintInfo("Files created:")
	fmt.Printf("  • plugin.properties\n")
	fmt.Printf("  • %s\n", mainFile)
	fmt.Printf("  • readme.txt\n")
	fmt.Printf("  • includes/\n")
	fmt.Printf("  • assets/css/%s.css\n", slug)
	fmt.Printf("  • assets/js/%s.js\n", slug)
	fmt.Printf("  • languages/\n")
	fmt.Println()
	ui.PrintInfo("Run 'wordsmith build' to build your plugin")
	fmt.Println()
}

func initTheme(dir string, interactive bool) {
	// Get default name from directory
	defaultName := formatName(filepath.Base(dir))

	var name, description, author, authorURI, themeType, template, templateURI string

	if interactive {
		reader := bufio.NewReader(os.Stdin)

		// Ask questions
		ui.PrintInfo("Let's set up your WordPress theme!")
		fmt.Println()

		name = prompt(reader, "Theme name", defaultName)
		description = prompt(reader, "Description", "A WordPress theme")
		author = prompt(reader, "Author", "")
		if author != "" {
			authorURI = prompt(reader, "Author website", "")
		}

		fmt.Println()
		fmt.Println("  Theme type:")
		fmt.Println("    1. Block    - Modern, uses Site Editor & block templates")
		fmt.Println("    2. Classic  - Traditional PHP templates")
		fmt.Println("    3. Hybrid   - Classic templates with theme.json (recommended)")
		fmt.Println("    4. Child    - Inherits from a parent theme")
		fmt.Println()
		themeTypeInput := prompt(reader, "Choose type (1/2/3/4)", "3")

		themeType = "hybrid"
		switch themeTypeInput {
		case "1", "block":
			themeType = "block"
		case "2", "classic":
			themeType = "classic"
		case "4", "child":
			themeType = "child"
		default:
			themeType = "hybrid"
		}

		// Ask for parent theme details if child theme
		if themeType == "child" {
			fmt.Println()
			template = prompt(reader, "Parent theme name", "")
			if template == "" {
				ui.PrintError("Parent theme name is required for child themes")
				os.Exit(1)
			}
			templateURI = prompt(reader, "Parent theme URL or path", "")
			if templateURI == "" {
				ui.PrintError("Parent theme URL or path is required for child themes")
				os.Exit(1)
			}
		}

		fmt.Println()
	} else {
		// Non-interactive mode - use flags or defaults
		name = initName
		if name == "" {
			name = defaultName
		}
		description = initDescription
		if description == "" {
			description = "A WordPress theme"
		}
		author = initAuthor
		authorURI = initAuthorURI
		template = initTemplate
		templateURI = initTemplateURI

		themeType = initThemeType
		switch themeType {
		case "block", "classic", "hybrid", "child":
			// Valid type
		default:
			themeType = "hybrid"
		}

		// Validate child theme requirements
		if themeType == "child" {
			if template == "" {
				ui.PrintError("--template is required for child themes")
				os.Exit(1)
			}
			if templateURI == "" {
				ui.PrintError("--template-uri is required for child themes")
				os.Exit(1)
			}
		}
	}

	// If current directory is not empty, create subdirectory
	if !isEmptyDir(dir) {
		slug := sanitizeName(name)
		newDir := filepath.Join(dir, slug)
		if err := os.MkdirAll(newDir, 0755); err != nil {
			ui.PrintError("Failed to create directory %s: %v", slug, err)
			os.Exit(1)
		}
		dir = newDir
	}

	// Check if theme.properties already exists
	if config.ThemeExists(dir) {
		ui.PrintWarning("theme.properties already exists")
		os.Exit(1)
	}

	// Generate slug from name
	slug := sanitizeName(name)
	textDomain := slug

	// Create theme.properties
	var props []string
	props = append(props, "# Theme Configuration")
	props = append(props, "")
	props = append(props, fmt.Sprintf("name=%s", name))
	props = append(props, fmt.Sprintf("description=%s", description))
	if author != "" {
		props = append(props, fmt.Sprintf("author=%s", author))
	}
	if authorURI != "" {
		props = append(props, fmt.Sprintf("author-uri=%s", authorURI))
	}
	props = append(props, "license=GPL-2.0+")
	props = append(props, "license-uri=https://www.gnu.org/licenses/gpl-2.0.html")
	props = append(props, "")
	props = append(props, "# Main stylesheet file")
	props = append(props, "main=style.css")
	props = append(props, "")
	props = append(props, "# WordPress requirements")
	props = append(props, "requires=6.0")
	props = append(props, "requires-php=7.4")
	props = append(props, "")
	props = append(props, "# Files to include (supports wildcards)")

	switch themeType {
	case "block":
		props = append(props, "include=*.php,theme.json,templates,parts,assets,languages")
	case "classic":
		props = append(props, "include=*.php,assets,languages,screenshot.png")
	case "child":
		props = append(props, "include=*.php,assets,languages,screenshot.png")
		props = append(props, "")
		props = append(props, "# Parent theme")
		props = append(props, fmt.Sprintf("template=%s", template))
		props = append(props, fmt.Sprintf("template-uri=%s", templateURI))
	default: // hybrid
		props = append(props, "include=*.php,theme.json,assets,languages,screenshot.png")
	}

	props = append(props, "")
	props = append(props, "# Files to exclude")
	props = append(props, "exclude=node_modules,build,.*")
	props = append(props, "")
	props = append(props, "# Text domain for internationalization")
	props = append(props, fmt.Sprintf("text-domain=%s", textDomain))
	props = append(props, "domain-path=/languages")
	props = append(props, "")
	props = append(props, "# Theme tags")
	props = append(props, "tags=custom-logo,custom-menu,editor-style")
	props = append(props, "")

	propsContent := strings.Join(props, "\n")
	propsPath := filepath.Join(dir, "theme.properties")
	if err := os.WriteFile(propsPath, []byte(propsContent), 0644); err != nil {
		ui.PrintError("Failed to create theme.properties: %v", err)
		os.Exit(1)
	}

	// Generate theme files based on type
	switch themeType {
	case "block":
		generateBlockTheme(dir, name, description, author, authorURI, slug)
	case "classic":
		generateClassicTheme(dir, name, description, author, authorURI, slug)
	case "child":
		generateChildTheme(dir, name, description, author, authorURI, slug, template)
	default:
		generateHybridTheme(dir, name, description, author, authorURI, slug)
	}

	// Print success
	ui.PrintSuccess("Created %s theme: %s", themeType, name)
	fmt.Println()
	ui.PrintInfo("Run 'wordsmith build' to build your theme")
	fmt.Println()
}

func generateBlockTheme(dir, name, description, author, authorURI, slug string) {
	// Create directories
	dirs := []string{"templates", "parts", "assets", "assets/css", "assets/js", "languages"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}

	// Create .gitignore
	gitignoreContent := "build/\n"
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignoreContent), 0644)

	// style.css
	styleContent := generateStyleCSS(name, description, author, authorURI, slug)
	os.WriteFile(filepath.Join(dir, "style.css"), []byte(styleContent), 0644)

	// theme.json
	themeJSON := generateThemeJSON(name, slug)
	os.WriteFile(filepath.Join(dir, "theme.json"), []byte(themeJSON), 0644)

	// functions.php
	functionsContent := generateBlockFunctionsPHP(name, slug)
	os.WriteFile(filepath.Join(dir, "functions.php"), []byte(functionsContent), 0644)

	// templates/index.html
	indexHTML := `<!-- wp:template-part {"slug":"header","tagName":"header"} /-->

<!-- wp:group {"tagName":"main","layout":{"type":"constrained"}} -->
<main class="wp-block-group">
    <!-- wp:query {"queryId":1,"query":{"perPage":10,"pages":0,"offset":0,"postType":"post","order":"desc","orderBy":"date","author":"","search":"","exclude":[],"sticky":"","inherit":true}} -->
    <div class="wp-block-query">
        <!-- wp:post-template -->
            <!-- wp:post-title {"isLink":true} /-->
            <!-- wp:post-excerpt /-->
        <!-- /wp:post-template -->

        <!-- wp:query-pagination -->
            <!-- wp:query-pagination-previous /-->
            <!-- wp:query-pagination-numbers /-->
            <!-- wp:query-pagination-next /-->
        <!-- /wp:query-pagination -->
    </div>
    <!-- /wp:query -->
</main>
<!-- /wp:group -->

<!-- wp:template-part {"slug":"footer","tagName":"footer"} /-->
`
	os.WriteFile(filepath.Join(dir, "templates", "index.html"), []byte(indexHTML), 0644)

	// parts/header.html
	headerHTML := fmt.Sprintf(`<!-- wp:group {"tagName":"header","className":"site-header"} -->
<header class="wp-block-group site-header">
    <!-- wp:group {"layout":{"type":"flex","justifyContent":"space-between"}} -->
    <div class="wp-block-group">
        <!-- wp:site-title /-->
        <!-- wp:navigation /-->
    </div>
    <!-- /wp:group -->
</header>
<!-- /wp:group -->
`)
	os.WriteFile(filepath.Join(dir, "parts", "header.html"), []byte(headerHTML), 0644)

	// parts/footer.html
	footerHTML := fmt.Sprintf(`<!-- wp:group {"tagName":"footer","className":"site-footer"} -->
<footer class="wp-block-group site-footer">
    <!-- wp:paragraph {"align":"center"} -->
    <p class="has-text-align-center">© {year} %s</p>
    <!-- /wp:paragraph -->
</footer>
<!-- /wp:group -->
`, name)
	os.WriteFile(filepath.Join(dir, "parts", "footer.html"), []byte(footerHTML), 0644)

	ui.PrintInfo("Files created:")
	fmt.Printf("  • theme.properties\n")
	fmt.Printf("  • style.css\n")
	fmt.Printf("  • theme.json\n")
	fmt.Printf("  • functions.php\n")
	fmt.Printf("  • templates/index.html\n")
	fmt.Printf("  • parts/header.html\n")
	fmt.Printf("  • parts/footer.html\n")
	fmt.Printf("  • assets/\n")
	fmt.Printf("  • languages/\n")
}

func generateClassicTheme(dir, name, description, author, authorURI, slug string) {
	// Create directories
	dirs := []string{"assets", "assets/css", "assets/js", "languages"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}

	// Create .gitignore
	gitignoreContent := "build/\n"
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignoreContent), 0644)

	// style.css
	styleContent := generateStyleCSS(name, description, author, authorURI, slug)
	os.WriteFile(filepath.Join(dir, "style.css"), []byte(styleContent), 0644)

	// functions.php
	functionsContent := generateClassicFunctionsPHP(name, slug)
	os.WriteFile(filepath.Join(dir, "functions.php"), []byte(functionsContent), 0644)

	// index.php
	indexContent := generateIndexPHP(slug)
	os.WriteFile(filepath.Join(dir, "index.php"), []byte(indexContent), 0644)

	// header.php
	headerContent := generateHeaderPHP(name, slug)
	os.WriteFile(filepath.Join(dir, "header.php"), []byte(headerContent), 0644)

	// footer.php
	footerContent := generateFooterPHP(name, slug)
	os.WriteFile(filepath.Join(dir, "footer.php"), []byte(footerContent), 0644)

	// sidebar.php
	sidebarContent := generateSidebarPHP(slug)
	os.WriteFile(filepath.Join(dir, "sidebar.php"), []byte(sidebarContent), 0644)

	// assets/css/main.css
	cssContent := fmt.Sprintf("/**\n * %s Main Styles\n */\n\n/* Add your custom styles here */\n", name)
	os.WriteFile(filepath.Join(dir, "assets", "css", "main.css"), []byte(cssContent), 0644)

	// assets/js/main.js
	jsContent := fmt.Sprintf("/**\n * %s Scripts\n */\n\n(function($) {\n    'use strict';\n\n    $(document).ready(function() {\n        // Your code here\n    });\n\n})(jQuery);\n", name)
	os.WriteFile(filepath.Join(dir, "assets", "js", "main.js"), []byte(jsContent), 0644)

	ui.PrintInfo("Files created:")
	fmt.Printf("  • theme.properties\n")
	fmt.Printf("  • style.css\n")
	fmt.Printf("  • functions.php\n")
	fmt.Printf("  • index.php\n")
	fmt.Printf("  • header.php\n")
	fmt.Printf("  • footer.php\n")
	fmt.Printf("  • sidebar.php\n")
	fmt.Printf("  • assets/css/main.css\n")
	fmt.Printf("  • assets/js/main.js\n")
	fmt.Printf("  • languages/\n")
}

func generateHybridTheme(dir, name, description, author, authorURI, slug string) {
	// Create directories
	dirs := []string{"assets", "assets/css", "assets/js", "languages"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}

	// Create .gitignore
	gitignoreContent := "build/\n"
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignoreContent), 0644)

	// style.css
	styleContent := generateStyleCSS(name, description, author, authorURI, slug)
	os.WriteFile(filepath.Join(dir, "style.css"), []byte(styleContent), 0644)

	// theme.json
	themeJSON := generateThemeJSON(name, slug)
	os.WriteFile(filepath.Join(dir, "theme.json"), []byte(themeJSON), 0644)

	// functions.php
	functionsContent := generateHybridFunctionsPHP(name, slug)
	os.WriteFile(filepath.Join(dir, "functions.php"), []byte(functionsContent), 0644)

	// index.php
	indexContent := generateIndexPHP(slug)
	os.WriteFile(filepath.Join(dir, "index.php"), []byte(indexContent), 0644)

	// header.php
	headerContent := generateHeaderPHP(name, slug)
	os.WriteFile(filepath.Join(dir, "header.php"), []byte(headerContent), 0644)

	// footer.php
	footerContent := generateFooterPHP(name, slug)
	os.WriteFile(filepath.Join(dir, "footer.php"), []byte(footerContent), 0644)

	// sidebar.php
	sidebarContent := generateSidebarPHP(slug)
	os.WriteFile(filepath.Join(dir, "sidebar.php"), []byte(sidebarContent), 0644)

	// assets/css/main.css
	cssContent := fmt.Sprintf("/**\n * %s Main Styles\n */\n\n/* Add your custom styles here */\n", name)
	os.WriteFile(filepath.Join(dir, "assets", "css", "main.css"), []byte(cssContent), 0644)

	// assets/js/main.js
	jsContent := fmt.Sprintf("/**\n * %s Scripts\n */\n\n(function($) {\n    'use strict';\n\n    $(document).ready(function() {\n        // Your code here\n    });\n\n})(jQuery);\n", name)
	os.WriteFile(filepath.Join(dir, "assets", "js", "main.js"), []byte(jsContent), 0644)

	ui.PrintInfo("Files created:")
	fmt.Printf("  • theme.properties\n")
	fmt.Printf("  • style.css\n")
	fmt.Printf("  • theme.json\n")
	fmt.Printf("  • functions.php\n")
	fmt.Printf("  • index.php\n")
	fmt.Printf("  • header.php\n")
	fmt.Printf("  • footer.php\n")
	fmt.Printf("  • sidebar.php\n")
	fmt.Printf("  • assets/css/main.css\n")
	fmt.Printf("  • assets/js/main.js\n")
	fmt.Printf("  • languages/\n")
}

func generateChildTheme(dir, name, description, author, authorURI, slug, template string) {
	// Create directories
	dirs := []string{"assets", "assets/css", "assets/js", "languages"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}

	// Create .gitignore
	gitignoreContent := "build/\n"
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignoreContent), 0644)

	// style.css (minimal - header generated by build)
	styleContent := generateStyleCSS(name, description, author, authorURI, slug)
	os.WriteFile(filepath.Join(dir, "style.css"), []byte(styleContent), 0644)

	// functions.php
	functionsContent := generateChildFunctionsPHP(name, slug, template)
	os.WriteFile(filepath.Join(dir, "functions.php"), []byte(functionsContent), 0644)

	// assets/css/child.css
	cssContent := fmt.Sprintf("/**\n * %s Child Theme Styles\n */\n\n/* Add your custom styles here */\n", name)
	os.WriteFile(filepath.Join(dir, "assets", "css", "child.css"), []byte(cssContent), 0644)

	// assets/js/child.js
	jsContent := fmt.Sprintf("/**\n * %s Child Theme Scripts\n */\n\n(function($) {\n    'use strict';\n\n    $(document).ready(function() {\n        // Your code here\n    });\n\n})(jQuery);\n", name)
	os.WriteFile(filepath.Join(dir, "assets", "js", "child.js"), []byte(jsContent), 0644)

	ui.PrintInfo("Files created:")
	fmt.Printf("  • theme.properties\n")
	fmt.Printf("  • style.css\n")
	fmt.Printf("  • functions.php\n")
	fmt.Printf("  • assets/css/child.css\n")
	fmt.Printf("  • assets/js/child.js\n")
	fmt.Printf("  • languages/\n")
}

func generateChildFunctionsPHP(name, slug, template string) string {
	funcPrefix := strings.ReplaceAll(slug, "-", "_")
	constName := strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
	parentSlug := sanitizeName(template)
	return fmt.Sprintf(`<?php
/**
 * %s functions and definitions
 *
 * Child theme of %s
 */

if (!defined('ABSPATH')) {
    exit;
}

// Load version from version.properties
$version_file = get_stylesheet_directory() . '/version.properties';
$version = '1.0.0';
if (file_exists($version_file)) {
    $props = parse_ini_file($version_file);
    if ($props && isset($props['major'], $props['minor'], $props['maintenance'])) {
        $version = $props['major'] . '.' . $props['minor'] . '.' . $props['maintenance'];
    }
}
define('%s_VERSION', $version);

/**
 * Enqueue parent and child theme styles
 */
function %s_enqueue_styles() {
    // Parent theme style
    wp_enqueue_style(
        '%s-parent-style',
        get_template_directory_uri() . '/style.css',
        array(),
        wp_get_theme(get_template())->get('Version')
    );

    // Child theme style
    wp_enqueue_style(
        '%s-style',
        get_stylesheet_uri(),
        array('%s-parent-style'),
        %s_VERSION
    );

    // Additional child CSS
    wp_enqueue_style(
        '%s-child',
        get_stylesheet_directory_uri() . '/assets/css/child.css',
        array('%s-style'),
        %s_VERSION
    );

    // Child JavaScript
    wp_enqueue_script(
        '%s-child',
        get_stylesheet_directory_uri() . '/assets/js/child.js',
        array('jquery'),
        %s_VERSION,
        true
    );
}
// Priority 20 ensures child styles load after all parent styles
add_action('wp_enqueue_scripts', '%s_enqueue_styles', 20);
`, name, template, constName, funcPrefix, parentSlug, slug, parentSlug, constName, slug, slug, constName, slug, constName, funcPrefix)
}

func generateStyleCSS(name, description, author, authorURI, slug string) string {
	return fmt.Sprintf(`/**
 * %s Styles
 *
 * @package %s
 */

/* Add your custom styles here */
`, name, slug)
}

func generateThemeJSON(name, slug string) string {
	return fmt.Sprintf(`{
	"$schema": "https://schemas.wp.org/trunk/theme.json",
	"version": 2,
	"settings": {
		"color": {
			"palette": [
				{
					"slug": "primary",
					"color": "#0073aa",
					"name": "Primary"
				},
				{
					"slug": "secondary",
					"color": "#23282d",
					"name": "Secondary"
				},
				{
					"slug": "background",
					"color": "#ffffff",
					"name": "Background"
				},
				{
					"slug": "foreground",
					"color": "#333333",
					"name": "Foreground"
				}
			]
		},
		"typography": {
			"fontFamilies": [
				{
					"slug": "system",
					"name": "System Font",
					"fontFamily": "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen-Sans, Ubuntu, Cantarell, 'Helvetica Neue', sans-serif"
				}
			],
			"fontSizes": [
				{
					"slug": "small",
					"size": "14px",
					"name": "Small"
				},
				{
					"slug": "medium",
					"size": "18px",
					"name": "Medium"
				},
				{
					"slug": "large",
					"size": "24px",
					"name": "Large"
				},
				{
					"slug": "x-large",
					"size": "36px",
					"name": "Extra Large"
				}
			]
		},
		"spacing": {
			"units": ["px", "em", "rem", "vh", "vw", "%%"]
		},
		"layout": {
			"contentSize": "800px",
			"wideSize": "1200px"
		}
	},
	"styles": {
		"color": {
			"background": "var(--wp--preset--color--background)",
			"text": "var(--wp--preset--color--foreground)"
		},
		"elements": {
			"link": {
				"color": {
					"text": "var(--wp--preset--color--primary)"
				}
			}
		}
	}
}
`)
}

func generateBlockFunctionsPHP(name, slug string) string {
	funcPrefix := strings.ReplaceAll(slug, "-", "_")
	constName := strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
	return fmt.Sprintf(`<?php
/**
 * %s functions and definitions
 */

if (!defined('ABSPATH')) {
    exit;
}

// Load version from version.properties
$version_file = get_template_directory() . '/version.properties';
$version = '1.0.0';
if (file_exists($version_file)) {
    $props = parse_ini_file($version_file);
    if ($props && isset($props['major'], $props['minor'], $props['maintenance'])) {
        $version = $props['major'] . '.' . $props['minor'] . '.' . $props['maintenance'];
    }
}
define('%s_VERSION', $version);

/**
 * Theme setup
 */
function %s_setup() {
    // Add support for block styles
    add_theme_support('wp-block-styles');

    // Add support for editor styles
    add_theme_support('editor-styles');

    // Enqueue editor styles
    add_editor_style('style.css');
}
add_action('after_setup_theme', '%s_setup');

/**
 * Enqueue theme styles
 */
function %s_enqueue_styles() {
    wp_enqueue_style(
        '%s-style',
        get_stylesheet_uri(),
        array(),
        %s_VERSION
    );
}
add_action('wp_enqueue_scripts', '%s_enqueue_styles');
`, name, constName, funcPrefix, funcPrefix, funcPrefix, slug, constName, funcPrefix)
}

func generateClassicFunctionsPHP(name, slug string) string {
	funcPrefix := strings.ReplaceAll(slug, "-", "_")
	constName := strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
	return fmt.Sprintf(`<?php
/**
 * %s functions and definitions
 */

if (!defined('ABSPATH')) {
    exit;
}

// Load version from version.properties
$version_file = get_template_directory() . '/version.properties';
$version = '1.0.0';
if (file_exists($version_file)) {
    $props = parse_ini_file($version_file);
    if ($props && isset($props['major'], $props['minor'], $props['maintenance'])) {
        $version = $props['major'] . '.' . $props['minor'] . '.' . $props['maintenance'];
    }
}
define('%s_VERSION', $version);

/**
 * Theme setup
 */
function %s_setup() {
    // Let WordPress handle the title tag
    add_theme_support('title-tag');

    // Enable featured images
    add_theme_support('post-thumbnails');

    // Custom logo support
    add_theme_support('custom-logo', array(
        'height'      => 100,
        'width'       => 400,
        'flex-height' => true,
        'flex-width'  => true,
    ));

    // HTML5 markup support
    add_theme_support('html5', array(
        'search-form',
        'comment-form',
        'comment-list',
        'gallery',
        'caption',
        'style',
        'script',
    ));

    // Register navigation menus
    register_nav_menus(array(
        'primary' => __('Primary Menu', '%s'),
        'footer'  => __('Footer Menu', '%s'),
    ));

    // Load textdomain for translations
    load_theme_textdomain('%s', get_template_directory() . '/languages');
}
add_action('after_setup_theme', '%s_setup');

/**
 * Register widget areas
 */
function %s_widgets_init() {
    register_sidebar(array(
        'name'          => __('Sidebar', '%s'),
        'id'            => 'sidebar-1',
        'description'   => __('Add widgets here.', '%s'),
        'before_widget' => '<section id="%%1$s" class="widget %%2$s">',
        'after_widget'  => '</section>',
        'before_title'  => '<h2 class="widget-title">',
        'after_title'   => '</h2>',
    ));
}
add_action('widgets_init', '%s_widgets_init');

/**
 * Enqueue scripts and styles
 */
function %s_enqueue_scripts() {
    // Main stylesheet
    wp_enqueue_style(
        '%s-style',
        get_stylesheet_uri(),
        array(),
        %s_VERSION
    );

    // Additional CSS
    wp_enqueue_style(
        '%s-main',
        get_template_directory_uri() . '/assets/css/main.css',
        array('%s-style'),
        %s_VERSION
    );

    // JavaScript
    wp_enqueue_script(
        '%s-main',
        get_template_directory_uri() . '/assets/js/main.js',
        array('jquery'),
        %s_VERSION,
        true
    );
}
add_action('wp_enqueue_scripts', '%s_enqueue_scripts');
`, name, constName, funcPrefix, slug, slug, slug, funcPrefix, funcPrefix, slug, slug, funcPrefix, funcPrefix, slug, constName, slug, constName, slug, constName, funcPrefix)
}

func generateHybridFunctionsPHP(name, slug string) string {
	funcPrefix := strings.ReplaceAll(slug, "-", "_")
	constName := strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
	return fmt.Sprintf(`<?php
/**
 * %s functions and definitions
 */

if (!defined('ABSPATH')) {
    exit;
}

// Load version from version.properties
$version_file = get_template_directory() . '/version.properties';
$version = '1.0.0';
if (file_exists($version_file)) {
    $props = parse_ini_file($version_file);
    if ($props && isset($props['major'], $props['minor'], $props['maintenance'])) {
        $version = $props['major'] . '.' . $props['minor'] . '.' . $props['maintenance'];
    }
}
define('%s_VERSION', $version);

/**
 * Theme setup
 */
function %s_setup() {
    // Let WordPress handle the title tag
    add_theme_support('title-tag');

    // Enable featured images
    add_theme_support('post-thumbnails');

    // Custom logo support
    add_theme_support('custom-logo', array(
        'height'      => 100,
        'width'       => 400,
        'flex-height' => true,
        'flex-width'  => true,
    ));

    // HTML5 markup support
    add_theme_support('html5', array(
        'search-form',
        'comment-form',
        'comment-list',
        'gallery',
        'caption',
        'style',
        'script',
    ));

    // Block editor support
    add_theme_support('wp-block-styles');
    add_theme_support('align-wide');
    add_theme_support('editor-styles');
    add_editor_style('style.css');

    // Register navigation menus
    register_nav_menus(array(
        'primary' => __('Primary Menu', '%s'),
        'footer'  => __('Footer Menu', '%s'),
    ));

    // Load textdomain for translations
    load_theme_textdomain('%s', get_template_directory() . '/languages');
}
add_action('after_setup_theme', '%s_setup');

/**
 * Register widget areas
 */
function %s_widgets_init() {
    register_sidebar(array(
        'name'          => __('Sidebar', '%s'),
        'id'            => 'sidebar-1',
        'description'   => __('Add widgets here.', '%s'),
        'before_widget' => '<section id="%%1$s" class="widget %%2$s">',
        'after_widget'  => '</section>',
        'before_title'  => '<h2 class="widget-title">',
        'after_title'   => '</h2>',
    ));
}
add_action('widgets_init', '%s_widgets_init');

/**
 * Enqueue scripts and styles
 */
function %s_enqueue_scripts() {
    // Main stylesheet
    wp_enqueue_style(
        '%s-style',
        get_stylesheet_uri(),
        array(),
        %s_VERSION
    );

    // Additional CSS
    wp_enqueue_style(
        '%s-main',
        get_template_directory_uri() . '/assets/css/main.css',
        array('%s-style'),
        %s_VERSION
    );

    // JavaScript
    wp_enqueue_script(
        '%s-main',
        get_template_directory_uri() . '/assets/js/main.js',
        array('jquery'),
        %s_VERSION,
        true
    );
}
add_action('wp_enqueue_scripts', '%s_enqueue_scripts');
`, name, constName, funcPrefix, slug, slug, slug, funcPrefix, funcPrefix, slug, slug, funcPrefix, funcPrefix, slug, constName, slug, constName, slug, constName, funcPrefix)
}

func generateIndexPHP(slug string) string {
	return fmt.Sprintf(`<?php
/**
 * The main template file
 */

get_header();
?>

<main id="primary" class="site-main">
    <?php
    if (have_posts()) :
        while (have_posts()) :
            the_post();
            ?>
            <article id="post-<?php the_ID(); ?>" <?php post_class(); ?>>
                <header class="entry-header">
                    <?php
                    if (is_singular()) :
                        the_title('<h1 class="entry-title">', '</h1>');
                    else :
                        the_title('<h2 class="entry-title"><a href="' . esc_url(get_permalink()) . '">', '</a></h2>');
                    endif;
                    ?>
                </header>

                <div class="entry-content">
                    <?php
                    if (is_singular()) :
                        the_content();
                    else :
                        the_excerpt();
                    endif;
                    ?>
                </div>
            </article>
            <?php
        endwhile;

        the_posts_navigation();
    else :
        ?>
        <p><?php esc_html_e('No posts found.', '%s'); ?></p>
        <?php
    endif;
    ?>
</main>

<?php
get_sidebar();
get_footer();
`, slug)
}

func generateHeaderPHP(name, slug string) string {
	return fmt.Sprintf(`<?php
/**
 * The header template
 */
?>
<!DOCTYPE html>
<html <?php language_attributes(); ?>>
<head>
    <meta charset="<?php bloginfo('charset'); ?>">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <?php wp_head(); ?>
</head>

<body <?php body_class(); ?>>
<?php wp_body_open(); ?>

<div id="page" class="site">
    <header id="masthead" class="site-header">
        <div class="site-branding">
            <?php
            if (has_custom_logo()) :
                the_custom_logo();
            else :
                ?>
                <h1 class="site-title">
                    <a href="<?php echo esc_url(home_url('/')); ?>">
                        <?php bloginfo('name'); ?>
                    </a>
                </h1>
                <?php
                $description = get_bloginfo('description', 'display');
                if ($description) :
                    ?>
                    <p class="site-description"><?php echo $description; ?></p>
                    <?php
                endif;
            endif;
            ?>
        </div>

        <nav id="site-navigation" class="main-navigation">
            <?php
            wp_nav_menu(array(
                'theme_location' => 'primary',
                'menu_id'        => 'primary-menu',
                'fallback_cb'    => false,
            ));
            ?>
        </nav>
    </header>

    <div id="content" class="site-content">
`)
}

func generateFooterPHP(name, slug string) string {
	return fmt.Sprintf(`<?php
/**
 * The footer template
 */
?>
    </div><!-- #content -->

    <footer id="colophon" class="site-footer">
        <div class="site-info">
            <p>&copy; <?php echo date('Y'); ?> <?php bloginfo('name'); ?></p>
        </div>

        <?php
        if (has_nav_menu('footer')) :
            wp_nav_menu(array(
                'theme_location' => 'footer',
                'menu_id'        => 'footer-menu',
                'depth'          => 1,
            ));
        endif;
        ?>
    </footer>
</div><!-- #page -->

<?php wp_footer(); ?>
</body>
</html>
`)
}

func generateSidebarPHP(slug string) string {
	return fmt.Sprintf(`<?php
/**
 * The sidebar template
 */

if (!is_active_sidebar('sidebar-1')) {
    return;
}
?>

<aside id="secondary" class="widget-area">
    <?php dynamic_sidebar('sidebar-1'); ?>
</aside>
`)
}

func prompt(reader *bufio.Reader, label, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("  %s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("  %s: ", label)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

func formatName(s string) string {
	// Convert kebab-case or snake_case to Title Case
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

func sanitizeName(name string) string {
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	result = re.ReplaceAllString(result, "")
	return result
}

func isEmptyDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true // Treat errors as empty
	}
	return len(entries) == 0
}

func generateMainPluginFile(name, description, author, authorURI, slug string) string {
	constName := strings.ToUpper(strings.ReplaceAll(slug, "-", "_"))
	funcPrefix := strings.ReplaceAll(slug, "-", "_")

	content := fmt.Sprintf(`<?php
/**
 * %s
 *
 * @package %s
 */

// If this file is called directly, abort.
if (!defined('WPINC')) {
    die;
}

// Plugin path
define('%s_PATH', plugin_dir_path(__FILE__));

// Plugin URL
define('%s_URL', plugin_dir_url(__FILE__));

// Load version from version.properties
$version_file = %s_PATH . 'version.properties';
$version = '1.0.0';
if (file_exists($version_file)) {
    $props = parse_ini_file($version_file);
    if ($props && isset($props['major'], $props['minor'], $props['maintenance'])) {
        $version = $props['major'] . '.' . $props['minor'] . '.' . $props['maintenance'];
    }
}
define('%s_VERSION', $version);

/**
 * Plugin activation hook
 */
function %s_activate() {
    // Activation code here
}
register_activation_hook(__FILE__, '%s_activate');

/**
 * Plugin deactivation hook
 */
function %s_deactivate() {
    // Deactivation code here
}
register_deactivation_hook(__FILE__, '%s_deactivate');

/**
 * Enqueue scripts and styles
 */
function %s_enqueue_scripts() {
    wp_enqueue_style(
        '%s-style',
        %s_URL . 'assets/css/%s.css',
        array(),
        %s_VERSION
    );

    wp_enqueue_script(
        '%s-script',
        %s_URL . 'assets/js/%s.js',
        array('jquery'),
        %s_VERSION,
        true
    );
}
add_action('wp_enqueue_scripts', '%s_enqueue_scripts');
`, name, slug,
		constName, constName, constName, constName,
		funcPrefix, funcPrefix, funcPrefix, funcPrefix,
		funcPrefix, slug, constName, slug, constName,
		slug, constName, slug, constName,
		funcPrefix)

	return content
}

func generateReadme(name, description, author, slug string) string {
	return fmt.Sprintf(`=== %s ===
Contributors: %s
Tags: wordpress
Requires at least: 5.0
Tested up to: 6.4
Stable tag: 1.0.0
Requires PHP: 7.4
License: GPLv2 or later
License URI: https://www.gnu.org/licenses/gpl-2.0.html

%s

== Description ==

%s

== Installation ==

1. Upload the plugin files to the /wp-content/plugins/%s directory
2. Activate the plugin through the 'Plugins' screen in WordPress

== Changelog ==

= 1.0.0 =
* Initial release
`, name, author, description, description, slug)
}
