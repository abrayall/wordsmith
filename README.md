# Wordsmith

A CLI tool for building WordPress plugins and themes.

## Installation

### Quick Install

**macOS/Linux:**
```bash
curl -sfL https://raw.githubusercontent.com/abrayall/wordsmith/refs/heads/main/install.sh | sh -
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/abrayall/wordsmith/refs/heads/main/install.ps1 | iex
```

### Build from Source

```bash
git clone https://github.com/abrayall/wordsmith.git
cd wordsmith
./install.sh
```

## Quick Start

### Create a new plugin
```bash
mkdir my-plugin && cd my-plugin
wordsmith init plugin
wordsmith build
```

### Create a new theme
```bash
mkdir my-theme && cd my-theme
wordsmith init theme
wordsmith build
```

## Usage

### Initialize a plugin or theme

Interactive mode:
```bash
wordsmith init plugin
wordsmith init theme
```

Non-interactive mode with flags:
```bash
# Plugin
wordsmith init plugin --name="My Plugin" --author="John Doe" --author-uri="https://example.com"

# Theme with type selection
wordsmith init theme --name="My Theme" --type=block
wordsmith init theme --name="My Theme" --type=classic
wordsmith init theme --name="My Theme" --type=hybrid

# Child theme
wordsmith init theme --name="My Child Theme" --type=child --template="Parent Theme" --template-uri="../parent-theme"
```

Available flags:
- `--name` - Plugin/theme name
- `--description` - Description
- `--author` - Author name
- `--author-uri` - Author website URL
- `--type` - Theme type: `block`, `classic`, `hybrid`, or `child` (themes only)
- `--template` - Parent theme name (required for child themes)
- `--template-uri` - Parent theme URL or path (required for child themes)

### Build

```bash
wordsmith build
```

Builds the plugin/theme and creates a ZIP file ready for upload to WordPress.

### WordPress Development Environment

Start a local WordPress instance in Docker:

```bash
wordsmith wordpress start
```

This will:
- Start MySQL and WordPress containers
- Auto-install WordPress with admin/admin credentials
- Open the browser to your local WordPress site

Stop the environment:
```bash
wordsmith wordpress stop
```

Delete the environment and all data:
```bash
wordsmith wordpress delete
```

Open WordPress in browser:
```bash
wordsmith wordpress browse        # opens frontend
wordsmith wordpress browse admin  # opens admin panel
```

### Deploy

Build and deploy to the running WordPress container:

```bash
wordsmith deploy
```

### Watch for Changes

Automatically rebuild and deploy when files change:

```bash
wordsmith watch
```

## Configuration

### plugin.properties

```properties
name=My Plugin
description=A WordPress plugin
author=Your Name
author-uri=https://example.com
license=GPL-2.0+
license-uri=https://www.gnu.org/licenses/gpl-2.0.html

main=my-plugin.php
requires=5.0
requires-php=7.4

include=includes,assets,languages
exclude=node_modules,tests,.*

text-domain=my-plugin
domain-path=/languages
```

### theme.properties

```properties
name=My Theme
description=A WordPress theme
author=Your Name
author-uri=https://example.com
license=GPL-2.0+
license-uri=https://www.gnu.org/licenses/gpl-2.0.html

main=style.css
requires=6.0
requires-php=7.4

include=*.php,theme.json,assets,languages
exclude=node_modules,build,.*

text-domain=my-theme
tags=custom-logo,custom-menu,editor-style
```

### Child Theme Configuration

For child themes, add parent theme settings:

```properties
name=My Child Theme
description=A child theme
template=parent-theme-slug
template-uri=https://example.com/parent-theme.zip

# Or use a local path
template-uri=../parent-theme
```

The `template-uri` can be:
- A URL to a zip file (downloaded automatically)
- A local path to a theme directory (built if it has theme.properties)
- A local path to a zip file

Child themes support recursive parent chains (child → parent → grandparent).

### Wildcard Support

Use glob patterns in includes and excludes:
- `*.php` - All PHP files in root
- `**/*.php` - All PHP files recursively
- `assets` - Entire directory (automatically includes all contents)

## License

GPL-2.0+
