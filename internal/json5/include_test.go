package json5

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveIncludes(t *testing.T) {
	dir := t.TempDir()

	// Create included file
	envContent := `{ debug: true, logLevel: "info" }`
	if err := os.WriteFile(filepath.Join(dir, "env.json5"), []byte(envContent), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
		check   func(t *testing.T, result map[string]any, sources map[string]string)
	}{
		{
			name: "resolve include directive",
			data: map[string]any{
				"name": "test",
				"env":  map[string]any{"$include": "./env.json5"},
			},
			check: func(t *testing.T, result map[string]any, sources map[string]string) {
				env, ok := result["env"].(map[string]any)
				if !ok {
					t.Fatal("env should be a map")
				}
				if env["debug"] != true {
					t.Errorf("expected debug=true, got %v", env["debug"])
				}
				if env["logLevel"] != "info" {
					t.Errorf("expected logLevel=info, got %v", env["logLevel"])
				}
				if sources["env"] == "" {
					t.Error("expected env to have a source file")
				}
			},
		},
		{
			name: "non-include object preserved",
			data: map[string]any{
				"settings": map[string]any{"key": "value", "other": "data"},
			},
			check: func(t *testing.T, result map[string]any, sources map[string]string) {
				settings, ok := result["settings"].(map[string]any)
				if !ok {
					t.Fatal("settings should be a map")
				}
				if settings["key"] != "value" {
					t.Errorf("expected key=value, got %v", settings["key"])
				}
			},
		},
		{
			name: "missing include file",
			data: map[string]any{
				"bad": map[string]any{"$include": "./nonexistent.json5"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, sources, err := ResolveIncludes(tt.data, dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveIncludes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil && err == nil {
				tt.check(t, result, sources)
			}
		})
	}
}

func TestCircularInclude(t *testing.T) {
	dir := t.TempDir()

	// a.json5 includes b.json5, which includes a.json5 — a true cycle
	aContent := `{ b: { $include: "./b.json5" } }`
	bContent := `{ a: { $include: "./a.json5" } }`
	if err := os.WriteFile(filepath.Join(dir, "a.json5"), []byte(aContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.json5"), []byte(bContent), 0o644); err != nil {
		t.Fatal(err)
	}

	data := map[string]any{
		"section": map[string]any{"$include": "./a.json5"},
	}
	_, _, err := ResolveIncludes(data, dir)
	if err == nil {
		t.Fatal("expected error for circular include, got nil")
	}
}

func TestPathTraversalInclude(t *testing.T) {
	dir := t.TempDir()

	// Write a file outside the config dir
	parent := filepath.Dir(dir)
	if err := os.WriteFile(filepath.Join(parent, "secret.json5"), []byte(`{ secret: "value" }`), 0o644); err != nil {
		t.Fatal(err)
	}

	data := map[string]any{
		"bad": map[string]any{"$include": "../secret.json5"},
	}
	_, _, err := ResolveIncludes(data, dir)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestNestedIncludes(t *testing.T) {
	dir := t.TempDir()

	// Create nested include chain
	inner := `{ value: "nested" }`
	if err := os.WriteFile(filepath.Join(dir, "inner.json5"), []byte(inner), 0o644); err != nil {
		t.Fatal(err)
	}

	outer := `{ nested: { $include: "./inner.json5" }, local: "data" }`
	if err := os.WriteFile(filepath.Join(dir, "outer.json5"), []byte(outer), 0o644); err != nil {
		t.Fatal(err)
	}

	data := map[string]any{
		"section": map[string]any{"$include": "./outer.json5"},
	}

	result, _, err := ResolveIncludes(data, dir)
	if err != nil {
		t.Fatal(err)
	}

	section, ok := result["section"].(map[string]any)
	if !ok {
		t.Fatal("section should be a map")
	}

	nested, ok := section["nested"].(map[string]any)
	if !ok {
		t.Fatal("section.nested should be a map")
	}

	if nested["value"] != "nested" {
		t.Errorf("expected nested value, got %v", nested["value"])
	}
}
