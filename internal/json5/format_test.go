package json5

import (
	"strings"
	"testing"
)

func TestFormatValueString(t *testing.T) {
	got := FormatValue("hello", 0)
	if got != `"hello"` {
		t.Errorf("got %s, want %q", got, `"hello"`)
	}
}

func TestFormatValueBool(t *testing.T) {
	if got := FormatValue(true, 0); got != "true" {
		t.Errorf("got %s, want true", got)
	}
	if got := FormatValue(false, 0); got != "false" {
		t.Errorf("got %s, want false", got)
	}
}

func TestFormatValueNil(t *testing.T) {
	if got := FormatValue(nil, 0); got != "null" {
		t.Errorf("got %s, want null", got)
	}
}

func TestFormatValueNumber(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{"integer", float64(42), "42"},
		{"float", float64(3.14), "3.14"},
		{"zero", float64(0), "0"},
		{"negative", float64(-1), "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatValue(tt.val, 0)
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFormatObjectEmpty(t *testing.T) {
	got := FormatObject(map[string]any{})
	if got != "{}" {
		t.Errorf("got %s, want {}", got)
	}
}

func TestFormatObjectSimple(t *testing.T) {
	data := map[string]any{
		"name": "test",
		"count": float64(5),
	}
	got := FormatObject(data)

	if !strings.Contains(got, "count: 5,") {
		t.Errorf("expected unquoted key with value, got:\n%s", got)
	}
	if !strings.Contains(got, `name: "test",`) {
		t.Errorf("expected name key, got:\n%s", got)
	}
}

func TestFormatObjectSortedKeys(t *testing.T) {
	data := map[string]any{
		"zebra": float64(1),
		"alpha": float64(2),
		"mid":   float64(3),
	}
	got := FormatObject(data)

	alphaIdx := strings.Index(got, "alpha")
	midIdx := strings.Index(got, "mid")
	zebraIdx := strings.Index(got, "zebra")

	if alphaIdx > midIdx || midIdx > zebraIdx {
		t.Errorf("keys should be sorted alphabetically, got:\n%s", got)
	}
}

func TestFormatObjectQuotedKeys(t *testing.T) {
	data := map[string]any{
		"with-hyphen": "val",
		"normal":      "val",
	}
	got := FormatObject(data)

	if !strings.Contains(got, `"with-hyphen": "val"`) {
		t.Errorf("hyphenated key should be quoted, got:\n%s", got)
	}
	if !strings.Contains(got, `normal: "val"`) {
		t.Errorf("normal key should be unquoted, got:\n%s", got)
	}
}

func TestFormatObjectNested(t *testing.T) {
	data := map[string]any{
		"outer": map[string]any{
			"inner": "value",
		},
	}
	got := FormatObject(data)

	if !strings.Contains(got, "outer: {\n") {
		t.Errorf("expected nested object, got:\n%s", got)
	}
	if !strings.Contains(got, `    inner: "value",`) {
		t.Errorf("expected indented inner key, got:\n%s", got)
	}
}

func TestFormatValueShortArray(t *testing.T) {
	data := []any{"a", "b", "c"}
	got := FormatValue(data, 0)
	if got != `["a", "b", "c"]` {
		t.Errorf("short primitive array should be compact, got: %s", got)
	}
}

func TestFormatValueLongArray(t *testing.T) {
	data := []any{
		map[string]any{"key": "val1"},
		map[string]any{"key": "val2"},
	}
	got := FormatValue(data, 0)

	if !strings.Contains(got, "[\n") {
		t.Errorf("array of objects should be multi-line, got:\n%s", got)
	}
}

func TestFormatValueTrailingCommas(t *testing.T) {
	data := map[string]any{
		"a": float64(1),
		"b": float64(2),
	}
	got := FormatObject(data)

	lines := strings.Split(got, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "{" || trimmed == "}" {
			continue
		}
		if !strings.HasSuffix(trimmed, ",") {
			t.Errorf("expected trailing comma on line %q", trimmed)
		}
	}
}
