package json5

import (
	"fmt"
	"strings"
	"unicode"
)

// SetValue performs a surgical text edit on JSON5 content to update a key's value.
// It preserves comments and formatting. If the key doesn't exist, it appends it.
func SetValue(content string, key string, newValue string) string {
	// Try to find and replace existing key
	updated, found := replaceExistingKey(content, key, newValue)
	if found {
		return updated
	}

	// Key not found — append before the last closing brace
	return appendKey(content, key, newValue)
}

// replaceExistingKey finds a key in JSON5 text and replaces its value.
// Returns the updated content and whether the key was found.
func replaceExistingKey(content string, key string, newValue string) (string, bool) {
	// Look for the key in both quoted and unquoted forms
	patterns := []string{
		fmt.Sprintf(`"%s"`, key),
		key,
	}

	for _, pattern := range patterns {
		idx := findKeyInJSON5(content, pattern)
		if idx < 0 {
			continue
		}

		// Find the colon after the key
		colonIdx := strings.Index(content[idx+len(pattern):], ":")
		if colonIdx < 0 {
			continue
		}
		colonIdx += idx + len(pattern)

		// Find the start of the value (skip whitespace after colon)
		valueStart := colonIdx + 1
		for valueStart < len(content) && (content[valueStart] == ' ' || content[valueStart] == '\t') {
			valueStart++
		}

		// Find the end of the value
		valueEnd := findValueEnd(content, valueStart)

		return content[:valueStart] + " " + newValue + content[valueEnd:], true
	}

	return content, false
}

// findKeyInJSON5 finds a key pattern in JSON5 content, skipping occurrences in comments and strings.
// Pattern matching is checked before string-skipping so that quoted key patterns (e.g. `"foo"`)
// are found even though they start with a quote character.
func findKeyInJSON5(content string, pattern string) int {
	i := 0
	for i < len(content) {
		// Skip line comments
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '/' {
			for i < len(content) && content[i] != '\n' {
				i++
			}
			continue
		}
		// Skip block comments
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '*' {
			i += 2
			for i+1 < len(content) && (content[i] != '*' || content[i+1] != '/') {
				i++
			}
			if i+1 < len(content) {
				i += 2
			}
			continue
		}

		// Check for key pattern match BEFORE string skipping, so that
		// quoted key patterns like `"foo"` are not swallowed by the
		// string-literal skip below.
		if i+len(pattern) <= len(content) && content[i:i+len(pattern)] == pattern {
			// Verify this is a key position: preceded by whitespace/brace/comma and followed by whitespace/colon
			validBefore := i == 0 || isKeyBoundary(content[i-1])
			afterIdx := i + len(pattern)
			validAfter := afterIdx < len(content) && (content[afterIdx] == ':' || content[afterIdx] == ' ' || content[afterIdx] == '\t')
			if validBefore && validAfter {
				return i
			}
		}

		// Skip string literals
		if content[i] == '"' || content[i] == '\'' {
			quote := content[i]
			i++
			for i < len(content) && content[i] != quote {
				if content[i] == '\\' {
					i++
				}
				i++
			}
			if i < len(content) {
				i++
			}
			continue
		}

		i++
	}
	return -1
}

func isKeyBoundary(b byte) bool {
	return b == '{' || b == ',' || b == '\n' || b == '\r' || b == ' ' || b == '\t'
}

// findValueEnd finds the end of a JSON5 value starting at the given position.
func findValueEnd(content string, start int) int {
	if start >= len(content) {
		return start
	}

	ch := content[start]

	// String value
	if ch == '"' || ch == '\'' {
		return findStringEnd(content, start)
	}

	// Object or array
	if ch == '{' || ch == '[' {
		return findBracketEnd(content, start)
	}

	// Primitive value (number, boolean, null, unquoted string)
	i := start
	for i < len(content) {
		if content[i] == ',' || content[i] == '}' || content[i] == ']' || content[i] == '\n' || content[i] == '\r' {
			break
		}
		// Stop at line comment
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '/' {
			break
		}
		i++
	}
	// Trim trailing whitespace
	for i > start && (content[i-1] == ' ' || content[i-1] == '\t') {
		i--
	}
	return i
}

func findStringEnd(content string, start int) int {
	quote := content[start]
	i := start + 1
	for i < len(content) {
		if content[i] == '\\' {
			i += 2
			continue
		}
		if content[i] == quote {
			return i + 1
		}
		i++
	}
	return i
}

func findBracketEnd(content string, start int) int {
	open := content[start]
	var close byte
	if open == '{' {
		close = '}'
	} else {
		close = ']'
	}

	depth := 1
	i := start + 1
	for i < len(content) && depth > 0 {
		ch := content[i]
		// Skip comments
		if i+1 < len(content) && ch == '/' && content[i+1] == '/' {
			for i < len(content) && content[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(content) && ch == '/' && content[i+1] == '*' {
			i += 2
			for i+1 < len(content) && (content[i] != '*' || content[i+1] != '/') {
				i++
			}
			if i+1 < len(content) {
				i += 2
			}
			continue
		}
		// Skip strings
		if ch == '"' || ch == '\'' {
			i = findStringEnd(content, i)
			continue
		}
		switch ch {
		case open:
			depth++
		case close:
			depth--
		}
		i++
	}
	return i
}

// AppendToObject appends a key-value pair to a JSON5 object, matching existing style.
func AppendToObject(content string, key string, value string) string {
	return appendKey(content, key, value)
}

// findOutermostClosingBrace returns the index of the } that closes the
// outermost object in content, skipping strings and comments. Returns -1
// if no such brace is found.
func findOutermostClosingBrace(content string) int {
	depth := 0
	last := -1
	i := 0
	for i < len(content) {
		// Skip line comments
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '/' {
			for i < len(content) && content[i] != '\n' {
				i++
			}
			continue
		}
		// Skip block comments
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '*' {
			i += 2
			for i+1 < len(content) && (content[i] != '*' || content[i+1] != '/') {
				i++
			}
			if i+1 < len(content) {
				i += 2
			}
			continue
		}
		// Skip string literals
		if content[i] == '"' || content[i] == '\'' {
			i = findStringEnd(content, i)
			continue
		}
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				last = i
			}
		}
		i++
	}
	return last
}

func appendKey(content string, key string, value string) string {
	// Detect indentation style
	indent := detectIndent(content)

	// Find the } that closes the outermost object, skipping strings/comments.
	lastBrace := findOutermostClosingBrace(content)
	if lastBrace < 0 {
		return content
	}

	// Check if we need a comma before the new entry
	needsComma := false
	for i := lastBrace - 1; i >= 0; i-- {
		ch := content[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			continue
		}
		if ch != ',' && ch != '{' {
			needsComma = true
		}
		break
	}

	comma := ""
	if needsComma {
		comma = ","
	}

	// Format: use unquoted key if the file already uses that style, otherwise quote.
	quotedKey := fmt.Sprintf(`"%s"`, key)
	if !NeedsQuoting(key) && usesUnquotedKeys(content) {
		quotedKey = key
	}

	insertion := fmt.Sprintf("%s\n%s%s: %s,", comma, indent, quotedKey, value)
	return content[:lastBrace] + insertion + "\n" + content[lastBrace:]
}

func detectIndent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) > 0 && trimmed[0] != '{' && trimmed[0] != '}' && trimmed[0] != '/' {
			indent := line[:len(line)-len(trimmed)]
			if indent != "" {
				return indent
			}
		}
	}
	return "  "
}

func NeedsQuoting(key string) bool {
	if len(key) == 0 {
		return true
	}
	for i, r := range key {
		if i == 0 && !unicode.IsLetter(r) && r != '_' && r != '$' {
			return true
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
			return true
		}
	}
	return false
}

func usesUnquotedKeys(content string) bool {
	// Simple heuristic: check if there are unquoted keys (identifier followed by colon)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '/' || trimmed[0] == '{' || trimmed[0] == '}' {
			continue
		}
		if trimmed[0] != '"' && trimmed[0] != '\'' {
			if colonIdx := strings.Index(trimmed, ":"); colonIdx > 0 {
				return true
			}
		}
	}
	return false
}
