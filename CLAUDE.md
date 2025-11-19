# Wordsmith

WordPress plugin build tool written in Go.

## Build Types

- `dev` - No processing
- `release` - Minify CSS/JS only
- `prod` - Obfuscate PHP + minify CSS/JS

## Version Detection

Reads from git tags using `git describe --tags --match "v*.*.*"`. Appends commit count if ahead of tag, timestamp if dirty.

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
