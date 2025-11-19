package obfuscator

import (
	"regexp"
	"strings"
)

func MinifyCSS(source string) string {
	result := source

	// Remove comments
	re := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	result = re.ReplaceAllString(result, "")

	// Remove whitespace
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	// Remove spaces around punctuation
	patterns := []struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`\s*([{};:,>+~])\s*`), "$1"},
		{regexp.MustCompile(`;\s*}`), "}"},
	}

	for _, p := range patterns {
		result = p.re.ReplaceAllString(result, p.repl)
	}

	return strings.TrimSpace(result)
}
