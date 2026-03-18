package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ConfigDir != DefaultConfigDir {
		t.Errorf("expected default configDir %q, got %q", DefaultConfigDir, cfg.ConfigDir)
	}
	if cfg.OutputFile != DefaultOutputFile {
		t.Errorf("expected default outputFile %q, got %q", DefaultOutputFile, cfg.OutputFile)
	}
	if cfg.MasterTemplate != DefaultMasterTemplate {
		t.Errorf("expected default masterTemplate %q, got %q", DefaultMasterTemplate, cfg.MasterTemplate)
	}
	if len(cfg.Passthrough) != len(DefaultPassthrough) {
		t.Errorf("expected %d passthrough entries, got %d", len(DefaultPassthrough), len(cfg.Passthrough))
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	configContent := `{
  configDir: "./custom-config",
  outputFile: "./custom-output.json",
  masterTemplate: "./custom-config/master.json5",
  passthrough: ["meta"],
}`
	if err := os.WriteFile(filepath.Join(dir, ".clawback.json5"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ConfigDir != "./custom-config" {
		t.Errorf("expected configDir ./custom-config, got %q", cfg.ConfigDir)
	}
	if cfg.OutputFile != "./custom-output.json" {
		t.Errorf("expected outputFile ./custom-output.json, got %q", cfg.OutputFile)
	}
	if len(cfg.Passthrough) != 1 || cfg.Passthrough[0] != "meta" {
		t.Errorf("expected passthrough [meta], got %v", cfg.Passthrough)
	}
}

func TestIsPassthrough(t *testing.T) {
	cfg := &Config{
		Passthrough: []string{"meta", "wizard", "plugins.installs"},
	}

	if !cfg.IsPassthrough("meta") {
		t.Error("meta should be passthrough")
	}
	if !cfg.IsPassthrough("plugins.installs") {
		t.Error("plugins.installs should be passthrough")
	}
	if cfg.IsPassthrough("env") {
		t.Error("env should not be passthrough")
	}
}

func TestResolvePath(t *testing.T) {
	cfg := &Config{}
	homeDir := "/home/user/.openclaw"

	result := cfg.ResolvePath(homeDir, "./config")
	expected := filepath.Join(homeDir, "config")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	// Absolute path should be returned as-is
	abs := "/absolute/path"
	result = cfg.ResolvePath(homeDir, abs)
	if result != abs {
		t.Errorf("expected %q, got %q", abs, result)
	}
}
