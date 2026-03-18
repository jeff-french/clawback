package jsonutil

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// DiffType represents the kind of difference found.
type DiffType string

const (
	DiffAdded   DiffType = "added"
	DiffRemoved DiffType = "removed"
	DiffChanged DiffType = "changed"
)

// Diff represents a single difference between two JSON structures.
type Diff struct {
	Path     string   `json:"path"`
	Type     DiffType `json:"type"`
	OldValue any      `json:"oldValue,omitempty"`
	NewValue any      `json:"newValue,omitempty"`
}

// Compare performs a deep structural comparison between two maps.
// It returns a list of differences found.
func Compare(old, new map[string]any) []Diff {
	var diffs []Diff
	compareValues("", old, new, &diffs)
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})
	return diffs
}

func compareValues(path string, old, new any, diffs *[]Diff) {
	oldMap, oldIsMap := old.(map[string]any)
	newMap, newIsMap := new.(map[string]any)

	if oldIsMap && newIsMap {
		compareMaps(path, oldMap, newMap, diffs)
		return
	}

	oldArr, oldIsArr := toSlice(old)
	newArr, newIsArr := toSlice(new)

	if oldIsArr && newIsArr {
		if !reflect.DeepEqual(normalizeJSON(old), normalizeJSON(new)) {
			*diffs = append(*diffs, Diff{
				Path:     path,
				Type:     DiffChanged,
				OldValue: old,
				NewValue: new,
			})
		}
		_ = oldArr
		_ = newArr
		return
	}

	if !reflect.DeepEqual(normalizeJSON(old), normalizeJSON(new)) {
		*diffs = append(*diffs, Diff{
			Path:     path,
			Type:     DiffChanged,
			OldValue: old,
			NewValue: new,
		})
	}
}

func compareMaps(path string, old, new map[string]any, diffs *[]Diff) {
	// Keys in old but not new
	for k, v := range old {
		childPath := joinPath(path, k)
		if newV, ok := new[k]; ok {
			compareValues(childPath, v, newV, diffs)
		} else {
			*diffs = append(*diffs, Diff{
				Path:     childPath,
				Type:     DiffRemoved,
				OldValue: v,
			})
		}
	}

	// Keys in new but not old
	for k, v := range new {
		childPath := joinPath(path, k)
		if _, ok := old[k]; !ok {
			*diffs = append(*diffs, Diff{
				Path:     childPath,
				Type:     DiffAdded,
				NewValue: v,
			})
		}
	}
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func toSlice(v any) ([]any, bool) {
	if s, ok := v.([]any); ok {
		return s, true
	}
	return nil, false
}

// normalizeJSON round-trips through JSON to normalize numeric types etc.
func normalizeJSON(v any) any {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return v
	}
	return result
}

// FormatDiffs returns a human-readable string of the diffs.
func FormatDiffs(diffs []Diff) string {
	if len(diffs) == 0 {
		return "No differences found."
	}

	var sb strings.Builder
	for _, d := range diffs {
		switch d.Type {
		case DiffAdded:
			sb.WriteString(fmt.Sprintf("+ %s: %s\n", d.Path, formatValue(d.NewValue)))
		case DiffRemoved:
			sb.WriteString(fmt.Sprintf("- %s: %s\n", d.Path, formatValue(d.OldValue)))
		case DiffChanged:
			sb.WriteString(fmt.Sprintf("~ %s: %s → %s\n", d.Path, formatValue(d.OldValue), formatValue(d.NewValue)))
		}
	}
	return sb.String()
}

func formatValue(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

// GetPath retrieves a value from a nested map using a dot-separated path.
func GetPath(data map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = data
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// SetPath sets a value in a nested map using a dot-separated path.
// It creates intermediate maps as needed.
func SetPath(data map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		next, ok := current[part]
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		m, ok := next.(map[string]any)
		if !ok {
			m = make(map[string]any)
			current[part] = m
		}
		current = m
	}
}

// OwningFile determines which source file owns a given path, based on the sources map.
func OwningFile(sources map[string]string, path string) string {
	// The sources map is keyed by top-level key → file path
	topKey := strings.SplitN(path, ".", 2)[0]
	if file, ok := sources[topKey]; ok {
		return file
	}
	return ""
}
