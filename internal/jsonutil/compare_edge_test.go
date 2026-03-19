package jsonutil

import (
	"strings"
	"testing"
)

func TestCompareEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		old       map[string]any
		new       map[string]any
		wantDiffs int
		check     func(t *testing.T, diffs []Diff)
	}{
		{
			name:      "both empty maps",
			old:       map[string]any{},
			new:       map[string]any{},
			wantDiffs: 0,
		},
		{
			name:      "nil value to non-nil",
			old:       map[string]any{"key": nil},
			new:       map[string]any{"key": "value"},
			wantDiffs: 1,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Type != DiffChanged {
					t.Errorf("expected changed, got %s", diffs[0].Type)
				}
			},
		},
		{
			name:      "non-nil value to nil",
			old:       map[string]any{"key": "value"},
			new:       map[string]any{"key": nil},
			wantDiffs: 1,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Type != DiffChanged {
					t.Errorf("expected changed, got %s", diffs[0].Type)
				}
			},
		},
		{
			name:      "nil to nil is equal",
			old:       map[string]any{"key": nil},
			new:       map[string]any{"key": nil},
			wantDiffs: 0,
		},
		{
			name:      "empty array to empty array",
			old:       map[string]any{"arr": []any{}},
			new:       map[string]any{"arr": []any{}},
			wantDiffs: 0,
		},
		{
			name:      "empty array vs non-empty array",
			old:       map[string]any{"arr": []any{}},
			new:       map[string]any{"arr": []any{"item"}},
			wantDiffs: 1,
		},
		{
			name:      "type mismatch string vs number",
			old:       map[string]any{"val": "42"},
			new:       map[string]any{"val": float64(42)},
			wantDiffs: 1,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Type != DiffChanged {
					t.Errorf("expected changed for type mismatch, got %s", diffs[0].Type)
				}
			},
		},
		{
			name:      "type mismatch map vs string",
			old:       map[string]any{"val": map[string]any{"nested": true}},
			new:       map[string]any{"val": "flat"},
			wantDiffs: 1,
		},
		{
			name:      "type mismatch string vs map",
			old:       map[string]any{"val": "flat"},
			new:       map[string]any{"val": map[string]any{"nested": true}},
			wantDiffs: 1,
		},
		{
			name:      "type mismatch array vs map",
			old:       map[string]any{"val": []any{1, 2}},
			new:       map[string]any{"val": map[string]any{"a": 1}},
			wantDiffs: 1,
		},
		{
			name:      "deeply nested change",
			old:       map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": "old"}}}},
			new:       map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": "new"}}}},
			wantDiffs: 1,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Path != "a.b.c.d" {
					t.Errorf("expected path a.b.c.d, got %s", diffs[0].Path)
				}
			},
		},
		{
			name: "multiple changes sorted by path",
			old: map[string]any{
				"z": "old",
				"a": "old",
				"m": "old",
			},
			new: map[string]any{
				"z": "new",
				"a": "new",
				"m": "new",
			},
			wantDiffs: 3,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Path != "a" || diffs[1].Path != "m" || diffs[2].Path != "z" {
					t.Errorf("diffs not sorted: %v", diffs)
				}
			},
		},
		{
			name:      "numeric type normalization (int vs float)",
			old:       map[string]any{"num": float64(1)},
			new:       map[string]any{"num": float64(1.0)},
			wantDiffs: 0,
		},
		{
			name: "added and removed in same comparison",
			old: map[string]any{
				"kept":    "same",
				"removed": "gone",
			},
			new: map[string]any{
				"kept":  "same",
				"added": "new",
			},
			wantDiffs: 2,
			check: func(t *testing.T, diffs []Diff) {
				types := map[DiffType]bool{}
				for _, d := range diffs {
					types[d.Type] = true
				}
				if !types[DiffAdded] {
					t.Error("expected an added diff")
				}
				if !types[DiffRemoved] {
					t.Error("expected a removed diff")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := Compare(tt.old, tt.new)
			if len(diffs) != tt.wantDiffs {
				t.Fatalf("expected %d diffs, got %d: %+v", tt.wantDiffs, len(diffs), diffs)
			}
			if tt.check != nil {
				tt.check(t, diffs)
			}
		})
	}
}

func TestGetPathEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		data   map[string]any
		path   string
		wantOK bool
		want   any
	}{
		{
			name:   "top-level key",
			data:   map[string]any{"key": "value"},
			path:   "key",
			wantOK: true,
			want:   "value",
		},
		{
			name:   "missing intermediate key",
			data:   map[string]any{"a": map[string]any{}},
			path:   "a.b.c",
			wantOK: false,
		},
		{
			name:   "path through non-map value",
			data:   map[string]any{"a": "string"},
			path:   "a.b",
			wantOK: false,
		},
		{
			name:   "nil value is found",
			data:   map[string]any{"key": nil},
			path:   "key",
			wantOK: true,
			want:   nil,
		},
		{
			name:   "empty string path retrieves top-level empty key",
			data:   map[string]any{"": "empty"},
			path:   "",
			wantOK: true,
			want:   "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := GetPath(tt.data, tt.path)
			if ok != tt.wantOK {
				t.Fatalf("GetPath() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && val != tt.want {
				t.Errorf("GetPath() = %v, want %v", val, tt.want)
			}
		})
	}
}

func TestSetPathEdgeCases(t *testing.T) {
	t.Run("overwrite non-map intermediate", func(t *testing.T) {
		data := map[string]any{"a": "string"}
		SetPath(data, "a.b", "value")
		val, ok := GetPath(data, "a.b")
		if !ok || val != "value" {
			t.Errorf("expected a.b='value', got %v (ok=%v)", val, ok)
		}
	})

	t.Run("single segment path", func(t *testing.T) {
		data := map[string]any{}
		SetPath(data, "key", "value")
		if data["key"] != "value" {
			t.Errorf("expected key='value', got %v", data["key"])
		}
	})

	t.Run("deeply nested creation", func(t *testing.T) {
		data := map[string]any{}
		SetPath(data, "a.b.c.d.e", "deep")
		val, ok := GetPath(data, "a.b.c.d.e")
		if !ok || val != "deep" {
			t.Errorf("expected a.b.c.d.e='deep', got %v", val)
		}
	})
}

func TestOwningFile(t *testing.T) {
	sources := map[string]string{
		"env":     "/home/config/env.json5",
		"plugins": "/home/config/plugins.json5",
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"top-level match", "env", "/home/config/env.json5"},
		{"nested path", "env.debug", "/home/config/env.json5"},
		{"deeply nested", "plugins.settings.autoUpdate", "/home/config/plugins.json5"},
		{"unknown key", "unknown.path", ""},
		{"empty path", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OwningFile(sources, tt.path)
			if got != tt.want {
				t.Errorf("OwningFile(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFormatDiffsEdgeCases(t *testing.T) {
	t.Run("empty diffs", func(t *testing.T) {
		result := FormatDiffs(nil)
		if !strings.Contains(result, "No differences") {
			t.Errorf("expected 'No differences', got: %s", result)
		}
	})

	t.Run("all diff types", func(t *testing.T) {
		diffs := []Diff{
			{Path: "added", Type: DiffAdded, NewValue: map[string]any{"nested": true}},
			{Path: "removed", Type: DiffRemoved, OldValue: []any{1, 2, 3}},
			{Path: "changed", Type: DiffChanged, OldValue: "old", NewValue: "new"},
		}
		result := FormatDiffs(diffs)
		if !strings.Contains(result, "+ added") {
			t.Errorf("missing added line in: %s", result)
		}
		if !strings.Contains(result, "- removed") {
			t.Errorf("missing removed line in: %s", result)
		}
		if !strings.Contains(result, "~ changed") {
			t.Errorf("missing changed line in: %s", result)
		}
	})
}
