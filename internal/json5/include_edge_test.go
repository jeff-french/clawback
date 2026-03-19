package json5

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCircularIncludeSelfReference(t *testing.T) {
	dir := t.TempDir()

	// A file that includes itself.
	selfContent := `{ self: { $include: "./self.json5" } }`
	if err := os.WriteFile(filepath.Join(dir, "self.json5"), []byte(selfContent), 0o644); err != nil {
		t.Fatal(err)
	}

	data := map[string]any{
		"root": map[string]any{"$include": "./self.json5"},
	}
	_, _, err := ResolveIncludes(data, dir)
	if err == nil {
		t.Fatal("expected error for self-referencing circular include, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' in error message, got: %v", err)
	}
}

func TestCircularIncludeThreeWay(t *testing.T) {
	dir := t.TempDir()

	// a -> b -> c -> a
	os.WriteFile(filepath.Join(dir, "a.json5"), []byte(`{ next: { $include: "./b.json5" } }`), 0o644)
	os.WriteFile(filepath.Join(dir, "b.json5"), []byte(`{ next: { $include: "./c.json5" } }`), 0o644)
	os.WriteFile(filepath.Join(dir, "c.json5"), []byte(`{ next: { $include: "./a.json5" } }`), 0o644)

	data := map[string]any{
		"start": map[string]any{"$include": "./a.json5"},
	}
	_, _, err := ResolveIncludes(data, dir)
	if err == nil {
		t.Fatal("expected error for 3-way circular include, got nil")
	}
}

func TestDeeplyNestedIncludes(t *testing.T) {
	dir := t.TempDir()

	// Create a chain of 5 levels of includes.
	os.WriteFile(filepath.Join(dir, "level5.json5"), []byte(`{ value: "deep" }`), 0o644)
	os.WriteFile(filepath.Join(dir, "level4.json5"), []byte(`{ nested: { $include: "./level5.json5" } }`), 0o644)
	os.WriteFile(filepath.Join(dir, "level3.json5"), []byte(`{ nested: { $include: "./level4.json5" } }`), 0o644)
	os.WriteFile(filepath.Join(dir, "level2.json5"), []byte(`{ nested: { $include: "./level3.json5" } }`), 0o644)
	os.WriteFile(filepath.Join(dir, "level1.json5"), []byte(`{ nested: { $include: "./level2.json5" } }`), 0o644)

	data := map[string]any{
		"root": map[string]any{"$include": "./level1.json5"},
	}
	result, _, err := ResolveIncludes(data, dir)
	if err != nil {
		t.Fatalf("deeply nested includes should succeed: %v", err)
	}

	// Navigate the chain: root.nested.nested.nested.nested.value
	current := result["root"]
	for i := 0; i < 4; i++ {
		m, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected map at level %d, got %T", i, current)
		}
		current = m["nested"]
	}
	leaf, ok := current.(map[string]any)
	if !ok {
		t.Fatalf("expected map at leaf, got %T", current)
	}
	if leaf["value"] != "deep" {
		t.Errorf("expected leaf value='deep', got %v", leaf["value"])
	}
}

func TestIncludeNonStringValue(t *testing.T) {
	dir := t.TempDir()

	data := map[string]any{
		"bad": map[string]any{"$include": 42},
	}
	_, _, err := ResolveIncludes(data, dir)
	if err == nil {
		t.Fatal("expected error for non-string $include value, got nil")
	}
	if !strings.Contains(err.Error(), "must be a string") {
		t.Errorf("expected 'must be a string' in error, got: %v", err)
	}
}

func TestIncludeInvalidJSON5(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "bad.json5"), []byte(`{ invalid: }`), 0o644)

	data := map[string]any{
		"section": map[string]any{"$include": "./bad.json5"},
	}
	_, _, err := ResolveIncludes(data, dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON5 in included file, got nil")
	}
}

func TestIncludePreservesNonMapValues(t *testing.T) {
	dir := t.TempDir()

	// Non-object values should pass through unchanged.
	data := map[string]any{
		"str":   "hello",
		"num":   float64(42),
		"bool":  true,
		"arr":   []any{"a", "b"},
		"null":  nil,
	}
	result, sources, err := ResolveIncludes(data, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["str"] != "hello" {
		t.Errorf("string not preserved")
	}
	if result["num"] != float64(42) {
		t.Errorf("number not preserved")
	}
	if result["bool"] != true {
		t.Errorf("bool not preserved")
	}
	if result["null"] != nil {
		t.Errorf("null not preserved")
	}
	// No sources for non-included values.
	if len(sources) != 0 {
		t.Errorf("expected no sources, got %v", sources)
	}
}

func TestIncludeEmptyObject(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "empty.json5"), []byte(`{}`), 0o644)

	data := map[string]any{
		"section": map[string]any{"$include": "./empty.json5"},
	}
	result, sources, err := ResolveIncludes(data, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	section, ok := result["section"].(map[string]any)
	if !ok {
		t.Fatal("section should be a map")
	}
	if len(section) != 0 {
		t.Errorf("expected empty map, got %v", section)
	}
	if sources["section"] == "" {
		t.Error("expected section to have a source even when empty")
	}
}

func TestPathTraversalVariants(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../secret.json5"},
		{"double parent", "../../secret.json5"},
		{"absolute path", "/etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]any{
				"bad": map[string]any{"$include": tt.path},
			}
			_, _, err := ResolveIncludes(data, dir)
			if err == nil {
				t.Fatalf("expected error for path traversal with %q, got nil", tt.path)
			}
		})
	}
}

func TestSafeReadFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()

	realFile := filepath.Join(dir, "real.json5")
	os.WriteFile(realFile, []byte(`{ key: "value" }`), 0o644)

	link := filepath.Join(dir, "link.json5")
	if err := os.Symlink(realFile, link); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	_, err := SafeReadFile(link)
	if err == nil {
		t.Fatal("expected error when reading symlink, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected 'symlink' in error, got: %v", err)
	}
}

func TestSafeReadFileNonexistent(t *testing.T) {
	_, err := SafeReadFile("/nonexistent/path/file.json5")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}
