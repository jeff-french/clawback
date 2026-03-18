package json5

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, result map[string]any)
	}{
		{
			name:  "simple object",
			input: `{ "key": "value" }`,
			check: func(t *testing.T, result map[string]any) {
				if result["key"] != "value" {
					t.Errorf("expected key=value, got %v", result["key"])
				}
			},
		},
		{
			name:  "unquoted keys",
			input: `{ key: "value", num: 42 }`,
			check: func(t *testing.T, result map[string]any) {
				if result["key"] != "value" {
					t.Errorf("expected key=value, got %v", result["key"])
				}
			},
		},
		{
			name: "with comments",
			input: `{
				// line comment
				key: "value",
				/* block comment */
				other: true,
			}`,
			check: func(t *testing.T, result map[string]any) {
				if result["key"] != "value" {
					t.Errorf("expected key=value, got %v", result["key"])
				}
				if result["other"] != true {
					t.Errorf("expected other=true, got %v", result["other"])
				}
			},
		},
		{
			name:    "invalid json5",
			input:   `{ key: }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil && err == nil {
				tt.check(t, result)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json5")
	if err := os.WriteFile(path, []byte(`{ key: "value" }`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key=value, got %v", result["key"])
	}
}
