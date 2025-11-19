# Wordsmith

WordPress plugin and theme build tool written in Go.

## Version Detection

Reads from git tags using `git describe --tags --match "v*.*.*"`. Appends commit count if ahead of tag, timestamp if dirty.

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
