package obfuscator

import (
	"regexp"
	"strings"
)

func MinifyJS(source string) string {
	result := source

	// Remove multi-line comments
	re := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	result = re.ReplaceAllString(result, "")

	// Remove single-line comments (but not in strings or regex)
	lines := strings.Split(result, "\n")
	var newLines []string
	for _, line := range lines {
		if idx := findJSCommentStart(line); idx != -1 {
			line = line[:idx]
		}
		newLines = append(newLines, line)
	}
	result = strings.Join(newLines, "\n")

	// Remove whitespace
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	// Remove spaces around operators and punctuation
	patterns := []struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`\s*([{}\[\]();,:<>+\-*/%=!&|?])\s*`), "$1"},
	}

	for _, p := range patterns {
		result = p.re.ReplaceAllString(result, p.repl)
	}

	// Ensure space after keywords
	keywords := []string{"return", "throw", "new", "delete", "typeof", "void", "in", "instanceof", "var", "let", "const", "if", "else", "for", "while", "do", "switch", "case", "break", "continue", "function", "class", "extends", "import", "export", "default", "try", "catch", "finally", "async", "await", "yield"}
	for _, kw := range keywords {
		re := regexp.MustCompile(`\b` + kw + `\b([^\s;{(])`)
		result = re.ReplaceAllString(result, kw+" $1")
	}

	return strings.TrimSpace(result)
}

func findJSCommentStart(line string) int {
	inSingle := false
	inDouble := false
	inTemplate := false
	escaped := false

	for i := 0; i < len(line); i++ {
		if escaped {
			escaped = false
			continue
		}

		ch := line[i]

		if ch == '\\' {
			escaped = true
			continue
		}

		if ch == '\'' && !inDouble && !inTemplate {
			inSingle = !inSingle
		} else if ch == '"' && !inSingle && !inTemplate {
			inDouble = !inDouble
		} else if ch == '`' && !inSingle && !inDouble {
			inTemplate = !inTemplate
		}

		if !inSingle && !inDouble && !inTemplate && i+1 < len(line) {
			if line[i:i+2] == "//" {
				return i
			}
		}
	}

	return -1
}
