package obfuscator

import (
	"encoding/base64"
	"regexp"
	"strings"
)

// Obfuscate takes PHP source code and returns obfuscated version
func Obfuscate(source string) (string, error) {
	result := source

	// Step 1: Strip comments
	result = stripComments(result)

	// Step 2: Encode string literals
	result = encodeStrings(result)

	// Step 3: Rename local variables
	result = renameLocalVariables(result)

	// Step 4: Minify whitespace
	result = minifyWhitespace(result)

	return result, nil
}

// stripComments removes all PHP comments
func stripComments(source string) string {
	// Remove multi-line comments /* */
	re := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	result := re.ReplaceAllString(source, "")

	// Remove single-line comments // but not inside strings
	// This is simplified - we handle it line by line
	lines := strings.Split(result, "\n")
	var newLines []string

	for _, line := range lines {
		// Remove // comments (simplified - doesn't handle // inside strings perfectly)
		if idx := findCommentStart(line, "//"); idx != -1 {
			line = line[:idx]
		}
		// Remove # comments
		if idx := findCommentStart(line, "#"); idx != -1 {
			line = line[:idx]
		}
		newLines = append(newLines, line)
	}

	return strings.Join(newLines, "\n")
}

// findCommentStart finds comment marker outside of strings
func findCommentStart(line, marker string) int {
	inSingle := false
	inDouble := false
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

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
		} else if ch == '"' && !inSingle {
			inDouble = !inDouble
		}

		if !inSingle && !inDouble && i+len(marker) <= len(line) {
			if line[i:i+len(marker)] == marker {
				return i
			}
		}
	}

	return -1
}

// encodeStrings encodes string literals to base64
func encodeStrings(source string) string {
	var result strings.Builder
	i := 0

	for i < len(source) {
		// Check for single-quoted string
		if source[i] == '\'' {
			str, end := extractString(source, i, '\'')
			if len(str) > 2 { // Only encode non-empty strings
				inner := str[1 : len(str)-1]
				encoded := base64.StdEncoding.EncodeToString([]byte(inner))
				result.WriteString("base64_decode('" + encoded + "')")
			} else {
				result.WriteString(str)
			}
			i = end
			continue
		}

		// Check for double-quoted string (skip these - they may have variables)
		if source[i] == '"' {
			str, end := extractString(source, i, '"')
			// Only encode if no variables inside
			if !strings.Contains(str, "$") && len(str) > 2 {
				inner := str[1 : len(str)-1]
				encoded := base64.StdEncoding.EncodeToString([]byte(inner))
				result.WriteString("base64_decode('" + encoded + "')")
			} else {
				result.WriteString(str)
			}
			i = end
			continue
		}

		result.WriteByte(source[i])
		i++
	}

	return result.String()
}

// extractString extracts a quoted string starting at position i
func extractString(source string, i int, quote byte) (string, int) {
	var str strings.Builder
	str.WriteByte(source[i])
	i++
	escaped := false

	for i < len(source) {
		ch := source[i]
		str.WriteByte(ch)

		if escaped {
			escaped = false
			i++
			continue
		}

		if ch == '\\' {
			escaped = true
			i++
			continue
		}

		if ch == quote {
			i++
			break
		}
		i++
	}

	return str.String(), i
}

// renameLocalVariables renames local variables in functions
func renameLocalVariables(source string) string {
	// Find function bodies and rename variables within them
	// This is a simplified approach - find function blocks and process them

	// Superglobals and special variables to skip
	skipVars := map[string]bool{
		"$_GET": true, "$_POST": true, "$_REQUEST": true, "$_SERVER": true,
		"$_SESSION": true, "$_COOKIE": true, "$_FILES": true, "$_ENV": true,
		"$GLOBALS": true, "$this": true, "$argc": true, "$argv": true,
	}

	// Find all function bodies
	funcRe := regexp.MustCompile(`function\s+\w+\s*\([^)]*\)\s*\{`)
	matches := funcRe.FindAllStringIndex(source, -1)

	if len(matches) == 0 {
		return source
	}

	// Process each function
	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		// Write everything before this function
		result.WriteString(source[lastEnd:match[0]])

		// Find the end of this function (matching braces)
		funcStart := match[0]
		bodyStart := match[1] - 1 // Position of opening brace
		bodyEnd := findMatchingBrace(source, bodyStart)

		if bodyEnd == -1 {
			// Couldn't find matching brace, skip this function
			result.WriteString(source[funcStart:match[1]])
			lastEnd = match[1]
			continue
		}

		// Extract function signature and body
		signature := source[funcStart:match[1]]
		body := source[match[1]:bodyEnd]

		// Find all variables in this function body
		varRe := regexp.MustCompile(`\$[a-zA-Z_][a-zA-Z0-9_]*`)
		vars := varRe.FindAllString(body, -1)

		// Create unique list and mapping
		varMap := make(map[string]string)
		counter := 0

		for _, v := range vars {
			if skipVars[v] {
				continue
			}
			if _, exists := varMap[v]; !exists {
				varMap[v] = "$" + genVarName(counter)
				counter++
			}
		}

		// Replace variables in body (longest first to avoid partial replacements)
		newBody := body
		// Sort by length descending
		sortedVars := sortByLengthDesc(varMap)
		for _, oldVar := range sortedVars {
			newVar := varMap[oldVar]
			// Use word boundary to avoid partial matches
			re := regexp.MustCompile(regexp.QuoteMeta(oldVar) + `\b`)
			newBody = re.ReplaceAllString(newBody, newVar)
		}

		result.WriteString(signature)
		result.WriteString(newBody)
		result.WriteString("}")

		lastEnd = bodyEnd + 1
	}

	// Write remaining content
	result.WriteString(source[lastEnd:])

	return result.String()
}

// findMatchingBrace finds the position of the closing brace
func findMatchingBrace(source string, start int) int {
	if start >= len(source) || source[start] != '{' {
		return -1
	}

	depth := 0
	inSingle := false
	inDouble := false
	escaped := false

	for i := start; i < len(source); i++ {
		if escaped {
			escaped = false
			continue
		}

		ch := source[i]

		if ch == '\\' && (inSingle || inDouble) {
			escaped = true
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
		} else if ch == '"' && !inSingle {
			inDouble = !inDouble
		}

		if !inSingle && !inDouble {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					return i
				}
			}
		}
	}

	return -1
}

// genVarName generates a short variable name from an index
func genVarName(index int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz"
	if index < 26 {
		return string(chars[index])
	}
	return string(chars[index/26-1]) + string(chars[index%26])
}

// sortByLengthDesc returns map keys sorted by length descending
func sortByLengthDesc(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	// Simple bubble sort by length (descending)
	for i := 0; i < len(keys)-1; i++ {
		for j := 0; j < len(keys)-i-1; j++ {
			if len(keys[j]) < len(keys[j+1]) {
				keys[j], keys[j+1] = keys[j+1], keys[j]
			}
		}
	}

	return keys
}

// minifyWhitespace reduces whitespace to minimum
func minifyWhitespace(source string) string {
	// Replace multiple whitespace with single space
	re := regexp.MustCompile(`\s+`)
	result := re.ReplaceAllString(source, " ")

	// Remove spaces around operators and punctuation
	patterns := []struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`\s*([;,\{\}\(\)\[\]])\s*`), "$1"},
		{regexp.MustCompile(`\s*([=+\-*/<>!&|])\s*`), "$1"},
		{regexp.MustCompile(`\s*:\s*`), ":"},
		{regexp.MustCompile(`\s*\?\s*`), "?"},
	}

	for _, p := range patterns {
		result = p.re.ReplaceAllString(result, p.repl)
	}

	// Ensure space after keywords
	keywords := []string{"if", "else", "elseif", "while", "for", "foreach", "switch", "case", "return", "echo", "print", "function", "class", "public", "private", "protected", "static", "const", "new", "throw", "catch", "try", "finally", "as", "instanceof"}
	for _, kw := range keywords {
		re := regexp.MustCompile(`\b` + kw + `\b([^\s])`)
		result = re.ReplaceAllString(result, kw+" $1")
	}

	return strings.TrimSpace(result)
}
