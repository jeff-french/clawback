package jsonutil

import (
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name      string
		old       map[string]any
		new       map[string]any
		wantDiffs int
		check     func(t *testing.T, diffs []Diff)
	}{
		{
			name:      "identical maps",
			old:       map[string]any{"a": "b", "c": float64(1)},
			new:       map[string]any{"a": "b", "c": float64(1)},
			wantDiffs: 0,
		},
		{
			name:      "added key",
			old:       map[string]any{"a": "b"},
			new:       map[string]any{"a": "b", "c": "d"},
			wantDiffs: 1,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Type != DiffAdded {
					t.Errorf("expected added, got %s", diffs[0].Type)
				}
				if diffs[0].Path != "c" {
					t.Errorf("expected path c, got %s", diffs[0].Path)
				}
			},
		},
		{
			name:      "removed key",
			old:       map[string]any{"a": "b", "c": "d"},
			new:       map[string]any{"a": "b"},
			wantDiffs: 1,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Type != DiffRemoved {
					t.Errorf("expected removed, got %s", diffs[0].Type)
				}
			},
		},
		{
			name:      "changed value",
			old:       map[string]any{"a": "old"},
			new:       map[string]any{"a": "new"},
			wantDiffs: 1,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Type != DiffChanged {
					t.Errorf("expected changed, got %s", diffs[0].Type)
				}
			},
		},
		{
			name: "nested diff",
			old: map[string]any{
				"parent": map[string]any{"child": "old"},
			},
			new: map[string]any{
				"parent": map[string]any{"child": "new"},
			},
			wantDiffs: 1,
			check: func(t *testing.T, diffs []Diff) {
				if diffs[0].Path != "parent.child" {
					t.Errorf("expected path parent.child, got %s", diffs[0].Path)
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

func TestGetSetPath(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "value",
			},
		},
	}

	// Test GetPath
	val, ok := GetPath(data, "a.b.c")
	if !ok || val != "value" {
		t.Errorf("GetPath(a.b.c) = %v, %v", val, ok)
	}

	_, ok = GetPath(data, "nonexistent")
	if ok {
		t.Error("GetPath should return false for nonexistent path")
	}

	// Test SetPath
	SetPath(data, "a.b.d", "new")
	val, ok = GetPath(data, "a.b.d")
	if !ok || val != "new" {
		t.Errorf("SetPath then GetPath(a.b.d) = %v, %v", val, ok)
	}

	// Test SetPath creates intermediate maps
	SetPath(data, "x.y.z", "deep")
	val, ok = GetPath(data, "x.y.z")
	if !ok || val != "deep" {
		t.Errorf("SetPath with intermediate maps: GetPath(x.y.z) = %v, %v", val, ok)
	}
}

func TestFormatDiffs(t *testing.T) {
	diffs := []Diff{
		{Path: "a", Type: DiffAdded, NewValue: "new"},
		{Path: "b", Type: DiffRemoved, OldValue: "old"},
		{Path: "c", Type: DiffChanged, OldValue: "old", NewValue: "new"},
	}

	output := FormatDiffs(diffs)
	if output == "" {
		t.Error("FormatDiffs should produce output")
	}
}
