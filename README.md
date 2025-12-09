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
wordsmith wordpress start [file]           # use specific properties file
wordsmith wordpress start --quiet          # suppress header output
```

This will:
- Start MySQL and WordPress containers
- Auto-install WordPress with admin/admin credentials
- Install plugins/themes from wordpress.properties (if present)
- Open the browser to your local WordPress site

Stop the environment:
```bash
wordsmith wordpress stop
wordsmith wordpress stop [name]            # stop specific instance
```

Delete the environment and all data:
```bash
wordsmith wordpress delete
wordsmith wordpress delete [name]          # delete specific instance
```

Open WordPress in browser:
```bash
wordsmith wordpress browse        # opens frontend
wordsmith wordpress browse admin  # opens admin panel
```

List all WordPress environments:
```bash
wordsmith wordpress ps
```

### Site Management

Sites are complete WordPress projects containing multiple plugins and themes. A site directory has:
- `site.properties` - Configuration file
- `plugins/` - Local plugins (zip files or source directories)
- `themes/` - Local themes (zip files or source directories)

#### Initialize a Site

```bash
mkdir my-site && cd my-site
wordsmith site init
wordsmith site init --name="My Site"
```

This creates:
```
my-site/
├── site.properties
├── plugins/
└── themes/
```

#### site.properties

```yaml
name: my-site
description: A WordPress site
# url: https://example.com

# Docker image (defaults to wordpress:latest)
# image: wordpress:6.4-php8.2

# Plugins from WordPress.org, GitHub, or URLs
plugins:
  - akismet
  - https://github.com/owner/repo
  - slug: woocommerce
    version: 8.0.0

# Themes from WordPress.org, GitHub, or URLs
themes:
  - flavor
```

Site properties support all the same options as `wordpress.properties`.

#### Adding Local Plugins and Themes

Place plugins in the `plugins/` directory:
```
plugins/
├── my-plugin/              # Source directory with plugin.properties
│   ├── plugin.properties
│   └── my-plugin.php
├── another-plugin.zip      # Pre-built zip file
└── premium-plugin/
    └── premium-plugin.zip  # Zip inside a directory
```

Place themes in the `themes/` directory:
```
themes/
├── my-theme/               # Source directory with theme.properties
│   ├── theme.properties
│   └── style.css
└── purchased-theme.zip     # Pre-built zip file
```

Source directories (with `plugin.properties` or `theme.properties`) are automatically built before deployment.

#### Start a Site

```bash
wordsmith site start
```

This is an alias for `wordsmith wordpress start` - both commands work with `site.properties`.

#### Build Site Plugins and Themes

```bash
wordsmith site build
```

Builds all source plugins and themes in the site.

#### Build Docker Image

```bash
wordsmith site build docker
```

Creates a Docker image with WordPress and all plugins/themes pre-installed. The image:
- Uses the base image from `site.properties` (or `wordpress:latest`)
- Includes all local plugins and themes from the `plugins/` and `themes/` directories
- Installs plugins and themes from WordPress.org, GitHub, or URLs specified in `site.properties`
- Activates plugins and themes automatically on container startup

Run the built image:
```bash
docker run -p 8080:80 my-site:latest
```

#### Stop and Delete

```bash
wordsmith site stop           # Stop the site
wordsmith site delete         # Delete all site data
```

### Deploy

Build and deploy to the running WordPress container:

```bash
wordsmith deploy
wordsmith deploy [file]            # use specific properties file for WordPress instance
```

If WordPress is not running, deploy will automatically start it using the properties file.

### Watch for Changes

Automatically rebuild and deploy when files change:

```bash
wordsmith watch
```

## Configuration

Configuration files support both properties syntax (`key=value`) and YAML syntax (`key: value`). You can mix both in the same file.

### wordpress.properties

Define WordPress environment settings, plugins, and themes to install:

```yaml
# Instance name (optional, defaults to plugin/theme name or directory)
name: my-site

# Docker image (defaults to wordpress:latest)
image: wordpress:6.4-php8.2

# Plugins to install (active: true is default)
plugins:
  - akismet                           # simple slug from WordPress.org (latest)
  - slug: woocommerce
    version: 8.0.0                    # specific version from WordPress.org
  - slug: custom-plugin
    uri: https://example.com/plugin.zip   # install from URL
  - slug: local-plugin
    uri: /path/to/plugin.zip          # install from local file
  - https://github.com/owner/repo     # GitHub repo URL (latest release)
  - https://github.com/owner/repo/releases  # also works with /releases
  - slug: github-plugin
    uri: https://github.com/owner/repo
    version: 1.2.0                    # specific version from GitHub releases
  - my-local-plugin                   # auto-resolves from plugins/my-local-plugin/
  - ../../sibling-project             # relative path to another project
  - slug: inactive-plugin
    active: false                     # install but don't activate

# Themes to install (first theme defaults to active)
themes:
  - twentytwentyfour                  # simple slug from WordPress.org (latest)
  - slug: astra
    version: 4.0.0                    # specific version
    active: true
  - slug: custom-theme
    uri: https://example.com/theme.zip
  - https://github.com/owner/theme-repo   # GitHub repo URL (latest release)
  - my-local-theme                    # auto-resolves from themes/my-local-theme/
  - ../../sibling-theme               # relative path to another project
```

#### Plugin/Theme Resolution

When you specify a plugin or theme by slug (e.g., `my-plugin`), Wordsmith checks for local sources before falling back to WordPress.org:

**For plugins:**
1. `plugins/my-plugin/plugin.zip` - Pre-built zip file
2. `plugins/my-plugin.zip` - Zip file in plugins directory
3. `my-plugin/plugin.zip` - Zip file in slug directory
4. `my-plugin.zip` - Zip file at root level
5. `plugins/my-plugin/plugin.properties` - Build from source
6. `my-plugin/plugin.properties` - Build from source
7. WordPress.org repository (fallback)

**For themes:**
1. `themes/my-theme/theme.zip` - Pre-built zip file
2. `themes/my-theme.zip` - Zip file in themes directory
3. `my-theme/theme.zip` - Zip file in slug directory
4. `my-theme.zip` - Zip file at root level
5. `themes/my-theme/theme.properties` - Build from source
6. `my-theme/theme.properties` - Build from source
7. WordPress.org repository (fallback)

**Relative paths** (e.g., `../../other-project`) are resolved from the directory containing `wordpress.properties` and follow the same resolution logic.

#### GitHub Releases

When you specify a GitHub repository URL (e.g., `https://github.com/owner/repo`), Wordsmith automatically resolves it to the appropriate release asset:

- **With version**: Downloads from `https://github.com/owner/repo/releases/tag/v{version}/{slug}-{version}.zip`
- **Without version**: Downloads the latest release

The release asset is matched by looking for:
1. `{slug}-{version}.zip` (exact match)
2. `{slug}.zip`
3. `plugin.zip` or `theme.zip`
4. Any `.zip` file in the release

If no releases exist, an error is displayed and installation is skipped.

### plugin.properties

```properties
name=My Plugin
slug=my-plugin
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

The `slug` field is optional. If not specified, it's derived from the `name` field (lowercased, spaces replaced with dashes, special characters removed).

#### Libraries

Include external PHP libraries in your plugin or theme build using the `libraries` property:

```yaml
libraries:
  - https://github.com/owner/php-library           # GitHub repo (latest release)
  - https://github.com/owner/another-lib:v1.0.0   # GitHub repo with specific version
  - https://example.com/library.zip                # Direct zip URL
  - ./vendor/local-lib.zip                         # Local zip file
  - name: custom-name
    url: https://github.com/owner/repo
    version: 2.0.0
```

Libraries are downloaded, extracted, and copied into the built plugin/theme at the top level using the library name as the directory. For example, a library named `php-utils` would be available at `my-plugin/php-utils/` in the built zip.

**Library name resolution:**
- Explicit `name` property takes precedence
- GitHub URLs: uses the repository name (e.g., `https://github.com/owner/my-lib` → `my-lib`)
- Zip URLs: uses the filename without extension (e.g., `library.zip` → `library`)

**Caching:** Libraries are cached in `~/.wordsmith/library/` to avoid re-downloading on subsequent builds.

### theme.properties

```properties
name=My Theme
slug=my-theme
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

The `slug` field is optional. If not specified, it's derived from the `name` field.

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

### YAML Syntax

You can also use YAML syntax in plugin.properties and theme.properties:

```yaml
name: My Plugin
description: A WordPress plugin
author: Your Name

# YAML lists for includes
include:
  - includes
  - assets
  - "*.php"

exclude:
  - node_modules
  - tests
```

## License

GPL-2.0+
