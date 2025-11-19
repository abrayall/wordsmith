# Wordsmith

A CLI tool for building WordPress plugins.

## Installation

```bash
./install.sh
```

Or build manually:

```bash
go build -o wordsmith .
```

## Usage

### Initialize a plugin

```bash
wordsmith init
```

Creates a `plugin.properties` file in the current directory.

### Build a plugin

```bash
wordsmith build
```

Build types:

- `--type=dev` - No obfuscation or minification
- `--type=release` - CSS/JS minification only (for WordPress.org)
- `--type=prod` - Full obfuscation and minification (default)

If no type is specified, it auto-detects based on version:
- Clean version (0.1.0) → production
- Dev version (0.1.0-5) → dev

## plugin.properties

```properties
name=My Plugin
description=A WordPress plugin

author=Your Name
author_uri=https://example.com
plugin_uri=https://github.com/user/plugin
license=GPL v2 or later
license_uri=https://www.gnu.org/licenses/gpl-2.0.html

main=my-plugin.php
include=assets,templates,includes
text_domain=my-plugin
domain_path=/languages
```

### Options

- `name` - Plugin name (required)
- `main` - Main plugin PHP file (required)
- `version` - Version string (optional, defaults to git tag)
- `description` - Plugin description
- `author` - Author name
- `author_uri` - Author URL
- `plugin_uri` - Plugin URL
- `license` - License name
- `license_uri` - License URL
- `include` - Comma-separated list of files/directories to include
- `text_domain` - Text domain for i18n
- `domain_path` - Path to language files
- `requires` - Minimum WordPress version
- `requires_php` - Minimum PHP version
- `obfuscate` - Enable/disable PHP obfuscation (default: true)

## Build Output

```
build/
  work/
    source/   # PHP files before obfuscation
    stage/    # Final assembled plugin
  plugin-name-version.zip
```

## Building Wordsmith

Build for all platforms:

```bash
./build.sh
```

Build and install locally:

```bash
./install.sh
```
