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
```

Available flags:
- `--name` - Plugin/theme name
- `--description` - Description
- `--author` - Author name
- `--author-uri` - Author website URL
- `--type` - Theme type: `block`, `classic`, or `hybrid` (themes only)

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

### Wildcard Support

Use glob patterns in includes and excludes:
- `*.php` - All PHP files in root
- `**/*.php` - All PHP files recursively
- `assets` - Entire directory (automatically includes all contents)

## License

GPL-2.0+
