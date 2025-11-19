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

Builds the plugin and creates a ZIP file ready for upload to WordPress.

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

### Deploy to WordPress

Build and deploy your plugin to the running WordPress container:

```bash
wordsmith deploy
```

This builds the plugin, copies it to the WordPress container, and activates it.

### Watch for Changes

Automatically rebuild or deploy when files change:

```bash
wordsmith watch build
wordsmith watch deploy
```

## plugin.properties

```properties
name=My Plugin
description=A WordPress plugin

author=Your Name
author-uri=https://example.com
plugin-uri=https://github.com/user/plugin
license=GPL v2 or later
license-uri=https://www.gnu.org/licenses/gpl-2.0.html

main=my-plugin.php
include=assets,templates,includes
text-domain=my-plugin
domain-path=/languages
```

### Options

- `name` - Plugin name (required)
- `main` - Main plugin PHP file (required)
- `version` - Version string (optional, defaults to git tag)
- `description` - Plugin description
- `author` - Author name
- `author-uri` - Author URL
- `plugin-uri` - Plugin URL
- `license` - License name
- `license-uri` - License URL
- `include` - Comma-separated list of files/directories to include
- `text-domain` - Text domain for i18n
- `domain-path` - Path to language files
- `requires` - Minimum WordPress version
- `requires-php` - Minimum PHP version

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
