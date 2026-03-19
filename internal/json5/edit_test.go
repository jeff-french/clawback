package json5

import (
	"strings"
	"testing"
)

func TestSetValue(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		key      string
		value    string
		wantSub  string // substring that should be present
		wantKeep string // substring that must be preserved (e.g., comments)
	}{
		{
			name: "replace existing value",
			content: `{
  debug: true,
  logLevel: "info",
}`,
			key:     "debug",
			value:   "false",
			wantSub: "false",
		},
		{
			name: "replace string value",
			content: `{
  logLevel: "info",
  timeout: 30,
}`,
			key:     "logLevel",
			value:   `"debug"`,
			wantSub: `"debug"`,
		},
		{
			name: "preserve comments",
			content: `{
  // This is important
  debug: true,
  // Log level setting
  logLevel: "info",
}`,
			key:      "debug",
			value:    "false",
			wantSub:  "false",
			wantKeep: "// This is important",
		},
		{
			name: "add new key",
			content: `{
  debug: true,
}`,
			key:     "newKey",
			value:   `"newValue"`,
			wantSub: `newKey: "newValue"`,
		},
		{
			name: "replace quoted key value",
			content: `{
  "host": "localhost",
  "port": 5432,
}`,
			key:     "host",
			value:   `"remotehost"`,
			wantSub: `"remotehost"`,
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

func TestSetValueQuotedKeyNoDuplicates(t *testing.T) {
	content := `{
  "host": "localhost",
  "port": 5432,
}`
	result := SetValue(content, "host", `"newvalue"`)

	// Must NOT create a duplicate key — the existing quoted key should be updated in-place.
	if strings.Count(result, `"host"`) != 1 {
		t.Errorf("expected exactly 1 occurrence of '\"host\"', got %d in:\n%s",
			strings.Count(result, `"host"`), result)
	}
	if !strings.Contains(result, `"newvalue"`) {
		t.Errorf("expected value to be updated in:\n%s", result)
	}
}

func TestAppendToObject(t *testing.T) {
	content := `{
  // Existing comment
  key1: "value1",
}`
	result := AppendToObject(content, "key2", `"value2"`)

	if !strings.Contains(result, "key2") {
		t.Errorf("result should contain key2:\n%s", result)
	}
	if !strings.Contains(result, "// Existing comment") {
		t.Errorf("result should preserve comments:\n%s", result)
	}
	if !strings.Contains(result, "key1") {
		t.Errorf("result should preserve key1:\n%s", result)
	}
}

func TestCommentPreservation(t *testing.T) {
	content := `{
  // Database configuration
  // Connection settings
  host: "localhost",
  port: 5432,
  // Credentials
  user: "admin",
  /* Multi-line
     block comment */
  password: "secret",
}`

	// Change a value
	result := SetValue(content, "port", "3306")

	// Verify all comments preserved
	comments := []string{
		"// Database configuration",
		"// Connection settings",
		"// Credentials",
		"/* Multi-line",
		"block comment */",
	}
	for _, c := range comments {
		if !strings.Contains(result, c) {
			t.Errorf("lost comment %q in:\n%s", c, result)
		}
	}

	// Verify value was changed
	if !strings.Contains(result, "3306") {
		t.Errorf("value not updated in:\n%s", result)
	}
}
