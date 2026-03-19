package json5

import (
	"strings"
	"testing"
)

func TestSetValueEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		key      string
		value    string
		wantSub  string
		wantKeep string
	}{
		{
			name:    "empty object appends key",
			content: `{}`,
			key:     "newKey",
			value:   `"hello"`,
			wantSub: `newKey`,
		},
		{
			name: "object with only comments",
			content: `{
  // just a comment
}`,
			key:     "key",
			value:   `true`,
			wantSub: `key`,
		},
		{
			name: "nested path-like key with dot",
			content: `{
  "a.b": "old",
}`,
			key:     "a.b",
			value:   `"new"`,
			wantSub: `"new"`,
		},
		{
			name: "key with special characters requires quoting on append",
			content: `{
  "existing": "value",
}`,
			key:     "special-key",
			value:   `42`,
			wantSub: `"special-key": 42`,
		},
		{
			name: "replace value that is an object",
			content: `{
  settings: { a: 1, b: 2 },
  other: true,
}`,
			key:     "settings",
			value:   `{"x": 99}`,
			wantSub: `{"x": 99}`,
		},
		{
			name: "replace value that is an array",
			content: `{
  items: ["a", "b", "c"],
  flag: true,
}`,
			key:     "items",
			value:   `["x","y"]`,
			wantSub: `["x","y"]`,
		},
		{
			name: "value after line comment",
			content: `{
  // This comment describes the key
  myKey: "original", // inline comment
}`,
			key:      "myKey",
			value:    `"updated"`,
			wantSub:  `"updated"`,
			wantKeep: "// This comment describes the key",
		},
		{
			name: "key in block comment should not be replaced",
			content: `{
  /* realKey: "ignored" */
  realKey: "actual",
}`,
			key:     "realKey",
			value:   `"changed"`,
			wantSub: `"changed"`,
		},
		{
			name: "single-character key",
			content: `{
  x: 1,
}`,
			key:     "x",
			value:   `2`,
			wantSub: `2`,
		},
		{
			name: "boolean value replaced with string",
			content: `{
  enabled: true,
}`,
			key:     "enabled",
			value:   `"yes"`,
			wantSub: `"yes"`,
		},
		{
			name: "key value with no trailing comma",
			content: `{
  only: "value"
}`,
			key:     "only",
			value:   `"replaced"`,
			wantSub: `"replaced"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SetValue(tt.content, tt.key, tt.value)
			if !strings.Contains(result, tt.wantSub) {
				t.Errorf("result missing %q:\n%s", tt.wantSub, result)
			}
			if tt.wantKeep != "" && !strings.Contains(result, tt.wantKeep) {
				t.Errorf("result lost %q:\n%s", tt.wantKeep, result)
			}
		})
	}
}

func TestAppendToObjectEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		key     string
		value   string
		wantSub string
	}{
		{
			name:    "append to empty object",
			content: `{}`,
			key:     "first",
			value:   `"value"`,
			wantSub: `first`,
		},
		{
			name: "append preserves trailing comment",
			content: `{
  key: "val",
  // trailing comment
}`,
			key:     "newKey",
			value:   `true`,
			wantSub: `newKey: true`,
		},
		{
			name: "append with quoted keys style",
			content: `{
  "quotedKey": "value",
}`,
			key:     "another",
			value:   `"data"`,
			wantSub: `"another": "data"`,
		},
		{
			name: "key requiring quoting (hyphen)",
			content: `{
  simple: true,
}`,
			key:     "my-key",
			value:   `"val"`,
			wantSub: `"my-key": "val"`,
		},
		{
			name: "key requiring quoting (starts with digit)",
			content: `{
  simple: true,
}`,
			key:     "0invalid",
			value:   `"val"`,
			wantSub: `"0invalid": "val"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendToObject(tt.content, tt.key, tt.value)
			if !strings.Contains(result, tt.wantSub) {
				t.Errorf("result missing %q:\n%s", tt.wantSub, result)
			}
		})
	}
}

func TestFindValueEnd(t *testing.T) {
	tests := []struct {
		name    string
		content string
		start   int
		want    int
	}{
		{
			name:    "past end of string",
			content: "abc",
			start:   5,
			want:    5,
		},
		{
			name:    "string value with escapes",
			content: `"hello \"world\""`,
			start:   0,
			want:    17,
		},
		{
			name:    "nested object",
			content: `{ a: { b: 1 } }`,
			start:   0,
			want:    15,
		},
		{
			name:    "array value",
			content: `[1, 2, 3], next`,
			start:   0,
			want:    9,
		},
		{
			name:    "primitive ends at comma",
			content: `42, next`,
			start:   0,
			want:    2,
		},
		{
			name:    "primitive ends at newline",
			content: "true\nnext",
			start:   0,
			want:    4,
		},
		{
			name:    "primitive ends at closing brace",
			content: `false}`,
			start:   0,
			want:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findValueEnd(tt.content, tt.start)
			if got != tt.want {
				t.Errorf("findValueEnd() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"simple", false},
		{"camelCase", false},
		{"with_underscore", false},
		{"$dollar", false},
		{"with-hyphen", true},
		{"with.dot", true},
		{"0startsWithDigit", true},
		{"", true},
		{"has space", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := needsQuoting(tt.key)
			if got != tt.want {
				t.Errorf("needsQuoting(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestDetectIndent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "two-space indent",
			content: "{\n  key: \"val\",\n}",
			want:    "  ",
		},
		{
			name:    "four-space indent",
			content: "{\n    key: \"val\",\n}",
			want:    "    ",
		},
		{
			name:    "tab indent",
			content: "{\n\tkey: \"val\",\n}",
			want:    "\t",
		},
		{
			name:    "no indent defaults to two spaces",
			content: "{}",
			want:    "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectIndent(tt.content)
			if got != tt.want {
				t.Errorf("detectIndent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindOutermostClosingBrace(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "simple object",
			content: `{ "a": 1 }`,
			want:    9,
		},
		{
			name:    "nested objects",
			content: `{ "a": { "b": 1 } }`,
			want:    18,
		},
		{
			name:    "brace in string is ignored",
			content: `{ "a": "}" }`,
			want:    11,
		},
		{
			name:    "brace in comment is ignored",
			content: "{ \"a\": 1 // }\n}",
			want:    14,
		},
		{
			name:    "no brace returns -1",
			content: `"no braces here"`,
			want:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findOutermostClosingBrace(tt.content)
			if got != tt.want {
				t.Errorf("findOutermostClosingBrace() = %d, want %d", got, tt.want)
			}
		})
	}
}
