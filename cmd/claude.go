package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"wordsmith/internal/ui"
)

const skillVersionPrefix = "<!-- wordsmith:"
const skillVersionSuffix = " -->"

// addClaudeSupport generates .claude/skills/wordsmith/SKILL.md
// Returns list of created file paths (relative to dir)
func addClaudeSupport(dir string) []string {
	var created []string

	skillDir := filepath.Join(dir, ".claude", "skills", "wordsmith")
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			ui.PrintWarning("Failed to create .claude/skills/wordsmith directory: %v", err)
			return created
		}
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		content := generateSkillMD()
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			ui.PrintWarning("Failed to create SKILL.md: %v", err)
		} else {
			created = append(created, ".claude/skills/wordsmith/SKILL.md")
		}
	}

	return created
}

// upgradeClaudeSkill checks if the skill file exists and is outdated, and upgrades it.
// Returns true if an upgrade was performed.
func upgradeClaudeSkill(dir string) bool {
	skillPath := filepath.Join(dir, ".claude", "skills", "wordsmith", "SKILL.md")

	f, err := os.Open(skillPath)
	if err != nil {
		return false
	}
	defer f.Close()

	// Read first line to check version
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return false
	}
	firstLine := scanner.Text()

	// Parse embedded version
	if !strings.HasPrefix(firstLine, skillVersionPrefix) {
		// Old file without version marker — upgrade it
		content := generateSkillMD()
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			return false
		}
		return true
	}

	embeddedVersion := strings.TrimPrefix(firstLine, skillVersionPrefix)
	embeddedVersion = strings.TrimSuffix(embeddedVersion, skillVersionSuffix)

	if embeddedVersion == Version {
		return false
	}

	// Version mismatch — regenerate
	content := generateSkillMD()
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return false
	}
	return true
}

func generateSkillMD() string {
	return fmt.Sprintf(`%s%s%s
---
name: wordsmith
description: "Use this skill when working on a WordPress plugin, theme, or library project built with Wordsmith, including CLI commands, configuration, build process, deployment, and project structure."
---

# Wordsmith Development Skill

## CLI Commands

### wordsmith init [plugin|theme|library]
Initialize a new WordPress plugin, theme, or library project.

Flags:
- `+"`--name`"+` — Plugin/theme/library name (default: directory name)
- `+"`--description`"+` — Plugin/theme description
- `+"`--author`"+` — Author name
- `+"`--author-uri`"+` — Author website URL
- `+"`--type`"+` — Theme type: block, classic, hybrid, or child
- `+"`--template`"+` — Parent theme name (for child themes)
- `+"`--template-uri`"+` — Parent theme URL or path (for child themes)
- `+"`--git, -g`"+` — Generate GitHub Actions build workflow and .gitignore
- `+"`--claude, -c`"+` — Generate Claude Code support files

Theme types:
- **block** — Modern, uses Site Editor & block templates
- **classic** — Traditional PHP templates
- **hybrid** — Classic templates with theme.json (recommended)
- **child** — Inherits from a parent theme

### wordsmith build
Build the plugin, theme, or library into a distributable ZIP file.

Flags:
- `+"`--quiet`"+` — Suppress output

Detects project type from properties file (plugin.properties, theme.properties, or library.properties).
Version is read from git tags using `+"`git describe --tags --match \"v*.*.*\"`"+`.

### wordsmith deploy [file]
Build and deploy the plugin or theme to a local WordPress Docker environment.

Flags:
- `+"`--quiet`"+` — Suppress output

Automatically starts WordPress if not running. Handles plugin dependencies and theme parent chains.

### wordsmith watch [build|deploy]
Watch for file changes and automatically rebuild or redeploy.

Modes:
- `+"`build`"+` — Rebuild on changes
- `+"`deploy`"+` — Rebuild and redeploy on changes

Uses 500ms debounce to avoid rapid rebuilds.

### wordsmith wordpress [command]
Manage WordPress Docker development environments.

Subcommands:
- `+"`start [file]`"+` — Start WordPress in Docker (auto-assigns ports 8080-8099)
- `+"`stop [name]`"+` — Stop WordPress containers
- `+"`ps`"+` — List all WordPress environments with status
- `+"`delete [name]`"+` — Delete WordPress environment and data
- `+"`browse [name]`"+` — Open WordPress in browser

### wordsmith site [command]
Manage WordPress site projects with multiple plugins and themes.

Subcommands:
- `+"`init`"+` — Create site.properties and directory structure
- `+"`start`"+` — Start WordPress for the site
- `+"`stop`"+` — Stop WordPress for the site
- `+"`delete`"+` — Delete WordPress environment
- `+"`build`"+` — Build all local plugins/themes in the site
- `+"`build docker`"+` — Create Docker image with site pre-installed

### wordsmith add [feature]
Add features to an existing project.

Available features:
- `+"`git`"+` — GitHub Actions build workflow and .gitignore
- `+"`claude`"+` — Claude Code support files

### wordsmith completion [shell]
Generate shell completion scripts (bash, zsh, fish, powershell).

## Configuration Files

### plugin.properties
`+"```properties"+`
# Plugin Configuration
name=My Plugin
description=A WordPress plugin
author=Author Name
author-uri=https://example.com
license=GPL-2.0+
license-uri=https://www.gnu.org/licenses/gpl-2.0.html

# Main plugin file
main=my-plugin.php

# WordPress requirements
requires=5.0
requires-php=7.4

# Files to include (supports wildcards)
include=includes,assets,languages

# Files to exclude
exclude=node_modules,tests,.*

# Text domain for internationalization
text-domain=my-plugin
domain-path=/languages

# Dependencies
libraries=my-library
plugins=dependency-plugin

# Minification and obfuscation
minify=true
obfuscate=false
`+"```"+`

### theme.properties
`+"```properties"+`
# Theme Configuration
name=My Theme
description=A WordPress theme
author=Author Name
author-uri=https://example.com
license=GPL-2.0+
license-uri=https://www.gnu.org/licenses/gpl-2.0.html

# Main stylesheet file
main=style.css

# WordPress requirements
requires=6.0
requires-php=7.4

# Files to include (supports wildcards)
include=*.php,theme.json,assets,languages,screenshot.png

# Files to exclude
exclude=node_modules,build,.*

# Text domain
text-domain=my-theme
domain-path=/languages

# Theme tags
tags=custom-logo,custom-menu,editor-style

# Parent theme (for child themes)
# template=parent-theme
# template-uri=https://github.com/user/parent-theme
`+"```"+`

### library.properties
`+"```properties"+`
# Library Configuration
name=My Library

# Files to include
include=src

# Files to exclude
exclude=node_modules,tests,.*

# Dependencies
libraries=other-library
`+"```"+`

### wordpress.properties
`+"```properties"+`
# WordPress environment configuration
name=my-site
description=Development site

# Plugins and themes to install
plugins=plugin-slug,https://example.com/plugin.zip
themes=theme-slug
`+"```"+`

### site.properties
`+"```properties"+`
# Site project configuration
name=my-site
description=A WordPress site

# Docker image
image=wordpress:latest

# Plugins and themes (local paths or remote)
plugins=../my-plugin,other-plugin
themes=../my-theme
`+"```"+`

## Project Structure

### Plugin
`+"```"+`
my-plugin/
├── plugin.properties      # Build configuration
├── my-plugin.php          # Main plugin file (WordPress header)
├── readme.txt             # WordPress readme
├── includes/              # PHP includes
├── assets/
│   ├── css/my-plugin.css
│   └── js/my-plugin.js
├── languages/             # Translation files
└── .gitignore
`+"```"+`

### Theme (Hybrid)
`+"```"+`
my-theme/
├── theme.properties       # Build configuration
├── style.css              # Main stylesheet (WordPress header)
├── theme.json             # Block editor settings
├── functions.php          # Theme setup and hooks
├── index.php              # Main template
├── header.php             # Header template
├── footer.php             # Footer template
├── sidebar.php            # Sidebar template
├── assets/
│   ├── css/main.css
│   └── js/main.js
├── languages/             # Translation files
└── .gitignore
`+"```"+`

### Theme (Block)
`+"```"+`
my-theme/
├── theme.properties       # Build configuration
├── style.css              # Main stylesheet
├── theme.json             # Block editor settings & styles
├── functions.php          # Theme setup
├── templates/
│   └── index.html         # Block template
├── parts/
│   ├── header.html        # Header block part
│   └── footer.html        # Footer block part
├── assets/
├── languages/
└── .gitignore
`+"```"+`

## Version Detection

Wordsmith reads version from git tags using `+"`git describe --tags --match \"v*.*.*\"`"+`. The version format is:
- Clean tag: `+"`v1.2.3`"+` → `+"`1.2.3`"+`
- Commits ahead: `+"`v1.2.3-5-gabcdef`"+` → `+"`1.2.3-5`"+`
- Dirty working tree: appends timestamp

A `+"`version.properties`"+` file is generated during build with `+"`major`"+`, `+"`minor`"+`, and `+"`maintenance`"+` fields, which PHP code reads at runtime.

## Build Process

1. Reads configuration from properties file
2. Detects version from git tags
3. Resolves file includes/excludes using glob patterns
4. Generates `+"`version.properties`"+` with version info
5. Packages into ZIP file in `+"`build/`"+` directory
6. Output: `+"`build/{slug}-{version}.zip`"+`

## Deployment

- Uses Docker to run WordPress locally
- `+"`wordsmith deploy`"+` builds the plugin/theme and installs it into the Docker WordPress instance
- `+"`wordsmith wordpress start`"+` launches WordPress with MySQL in Docker
- Supports plugin settings deployment via WordPress CLI
- Auto-detects and deploys plugin dependencies and theme parent chains
`, skillVersionPrefix, Version, skillVersionSuffix)
}
