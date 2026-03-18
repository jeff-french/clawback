package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jeff-french/clawback/internal/config"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantKeys []string
	}{
		{
			name:     "simple render with includes",
			fixture:  "simple",
			wantKeys: []string{"name", "version", "env"},
		},
		{
			name:     "render with comments fixture",
			fixture:  "with-comments",
			wantKeys: []string{"name", "plugins", "meta"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get testdata path relative to this test file
			fixtureDir := filepath.Join("..", "..", "testdata", tt.fixture)

			cfg := &config.Config{
				ConfigDir:      "./config",
				OutputFile:     "./openclaw.json",
				MasterTemplate: "./config/openclaw.json5",
				Passthrough:    []string{"meta", "wizard", "plugins.installs"},
			}

			result, err := Render(fixtureDir, cfg)
			if err != nil {
				t.Fatal(err)
			}

			for _, key := range tt.wantKeys {
				if _, ok := result.Data[key]; !ok {
					t.Errorf("missing expected key %q in rendered output", key)
				}
			}

			// Verify JSON is valid
			var check map[string]any
			if err := json.Unmarshal(result.JSON, &check); err != nil {
				t.Fatalf("rendered JSON is invalid: %v", err)
			}
		})
	}
}

func TestRenderWithPassthrough(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	os.MkdirAll(configDir, 0o755)

	// Write master template
	template := `{
  name: "test",
  meta: { version: 1 },
  env: { $include: "./env.json5" },
}`
	os.WriteFile(filepath.Join(configDir, "openclaw.json5"), []byte(template), 0o644)

	// Write env include
	env := `{ debug: true }`
	os.WriteFile(filepath.Join(configDir, "env.json5"), []byte(env), 0o644)

	// Write existing openclaw.json with passthrough data
	existing := map[string]any{
		"name": "test",
		"meta": map[string]any{"version": 2, "lastUpdated": "2024-01-01"},
		"env":  map[string]any{"debug": false},
	}
	existingJSON, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(dir, "openclaw.json"), existingJSON, 0o644)

	cfg := &config.Config{
		ConfigDir:      "./config",
		OutputFile:     "./openclaw.json",
		MasterTemplate: "./config/openclaw.json5",
		Passthrough:    []string{"meta"},
	}

	result, err := Render(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// meta should come from existing (passthrough)
	meta, ok := result.Data["meta"].(map[string]any)
	if !ok {
		t.Fatal("meta should be a map")
	}
	// meta.version should be 2 (from existing, not 1 from template)
	if v, ok := meta["version"]; !ok || v != float64(2) {
		t.Errorf("meta.version should be 2 (passthrough), got %v", meta["version"])
	}

	// env.debug should be true (from config, not passthrough)
	envData, envOk := result.Data["env"].(map[string]any)
	if !envOk {
		t.Fatal("env should be a map")
	}
	if envData["debug"] != true {
		t.Errorf("env.debug should be true (from config), got %v", envData["debug"])
	}
}

func TestRenderMissingTemplate(t *testing.T) {
	cfg := &config.Config{
		MasterTemplate: "./config/openclaw.json5",
	}

	_, err := Render(t.TempDir(), cfg)
	if err == nil {
		t.Error("expected error for missing template")
	}
}

func TestRenderRejectsSymlinkOutput(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	os.MkdirAll(configDir, 0o755)

	// Minimal valid template
	os.WriteFile(filepath.Join(configDir, "openclaw.json5"), []byte(`{ name: "test" }`), 0o644)

	// Replace openclaw.json with a symlink pointing elsewhere
	realTarget := filepath.Join(dir, "real-output.json")
	os.WriteFile(realTarget, []byte(`{"name":"old"}`), 0o644)
	symlinkPath := filepath.Join(dir, "openclaw.json")
	if err := os.Symlink(realTarget, symlinkPath); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	cfg := &config.Config{
		ConfigDir:      "./config",
		OutputFile:     "./openclaw.json",
		MasterTemplate: "./config/openclaw.json5",
		Passthrough:    []string{"meta"},
	}

	// Render should refuse to read the symlink as passthrough input
	_, err := Render(dir, cfg)
	if err == nil {
		t.Fatal("expected error when openclaw.json is a symlink, got nil")
	}
}
