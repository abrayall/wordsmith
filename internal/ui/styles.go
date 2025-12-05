package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#3B82F6") // Blue
	Success   = lipgloss.Color("#27C93F") // Green
	Warning   = lipgloss.Color("#F59E0B") // Amber
	Error     = lipgloss.Color("#EF4444") // Red
	Muted     = lipgloss.Color("#888888") // Gray

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Secondary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Italic(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Secondary)

	KeyStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	HighlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
)

// Highlight returns the string formatted as a highlighted value
func Highlight(s string) string {
	return HighlightStyle.Render(s)
}

// Banner prints the wordsmith banner
func Banner() string {
	banner := `
 █   █ █▀▀█ █▀▀█ █▀▀▄ █▀▀ █▀▄▀█ ▀█▀ ▀▀█▀▀ █  █
 █▄█▄█ █  █ █▄▄▀ █  █ ▀▀█ █ ▀ █  █    █   █▀▀█
  ▀ ▀  ▀▀▀▀ ▀ ▀▀ ▀▀▀  ▀▀▀ ▀   ▀ ▀▀▀   ▀   ▀  ▀`
	return TitleStyle.Render(banner)
}

// Header prints a section header
func Header(text string) string {
	return TitleStyle.Render("▸ " + text)
}

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	fmt.Println(SuccessStyle.Render("✓ " + fmt.Sprintf(format, args...)))
}

// PrintInfo prints an info message
func PrintInfo(format string, args ...interface{}) {
	fmt.Println(InfoStyle.Render("• " + fmt.Sprintf(format, args...)))
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	fmt.Println(ErrorStyle.Render("✗ " + fmt.Sprintf(format, args...)))
}

// PrintWarning prints a warning message
func PrintWarning(format string, args ...interface{}) {
	fmt.Println(WarningStyle.Render("⚠ " + fmt.Sprintf(format, args...)))
}

// PrintKeyValue prints a key-value pair
func PrintKeyValue(key, value string) {
	fmt.Printf("  %s %s\n", KeyStyle.Render(key+":"), ValueStyle.Render(value))
}

// Divider prints a divider line
func Divider() string {
	return MutedStyle.Render("──────────────────────────────────────────────")
}

// VersionLine returns the formatted version string
func VersionLine(version string) string {
	return ValueStyle.Render(" v" + version)
}

// PrintVersion prints the version
func PrintVersion(version string) {
	fmt.Println(VersionLine(version))
}

// PrintHeader prints the standard header
func PrintHeader(version string) {
	fmt.Println()
	fmt.Println(Divider())
	fmt.Println(Banner())
	PrintVersion(version)
	fmt.Println()
	fmt.Println(Divider())
	fmt.Println()
}
