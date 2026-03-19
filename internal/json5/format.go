package json5

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const defaultIndent = "  "

// FormatObject formats a map as a top-level JSON5 object string.
func FormatObject(data map[string]any) string {
	return FormatValue(data, 0)
}

// FormatValue converts a Go value to a JSON5 string with unquoted keys,
// trailing commas, and 2-space indentation.
func FormatValue(v any, depth int) string {
	switch val := v.(type) {
	case map[string]any:
		return formatMap(val, depth)
	case []any:
		return formatSlice(val, depth)
	case string:
		b, _ := json.Marshal(val)
		return string(b)
	case json.Number:
		return val.String()
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

func formatMap(data map[string]any, depth int) string {
	if len(data) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	indent := strings.Repeat(defaultIndent, depth+1)
	closeIndent := strings.Repeat(defaultIndent, depth)

	var b strings.Builder
	b.WriteString("{\n")
	for _, k := range keys {
		b.WriteString(indent)
		if NeedsQuoting(k) {
			fmt.Fprintf(&b, "%q", k)
		} else {
			b.WriteString(k)
		}
		b.WriteString(": ")
		b.WriteString(FormatValue(data[k], depth+1))
		b.WriteString(",\n")
	}
	b.WriteString(closeIndent)
	b.WriteString("}")
	return b.String()
}

func formatSlice(data []any, depth int) string {
	if len(data) == 0 {
		return "[]"
	}

	// Use compact form for short primitive-only arrays.
	if isShortPrimitiveSlice(data) {
		parts := make([]string, len(data))
		for i, v := range data {
			parts[i] = FormatValue(v, depth+1)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}

	indent := strings.Repeat(defaultIndent, depth+1)
	closeIndent := strings.Repeat(defaultIndent, depth)

	var b strings.Builder
	b.WriteString("[\n")
	for _, v := range data {
		b.WriteString(indent)
		b.WriteString(FormatValue(v, depth+1))
		b.WriteString(",\n")
	}
	b.WriteString(closeIndent)
	b.WriteString("]")
	return b.String()
}

func isShortPrimitiveSlice(data []any) bool {
	if len(data) > 5 {
		return false
	}
	for _, v := range data {
		switch v.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}
