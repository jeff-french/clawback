package config

import (
	"os"
	"path/filepath"
	"strings"
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
	if len(cfg.Passthrough) != len(DefaultPassthrough()) {
		t.Errorf("expected %d passthrough entries, got %d", len(DefaultPassthrough()), len(cfg.Passthrough))
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

func TestLoadRejectsSymlink(t *testing.T) {
	dir := t.TempDir()

	// Create a real file and a symlink pointing to it
	real := filepath.Join(dir, "real.json5")
	if err := os.WriteFile(real, []byte(`{ configDir: "./config" }`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".clawback.json5")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error when .clawback.json5 is a symlink, got nil")
	}
}

func TestValidate(t *testing.T) {
	homeDir := t.TempDir()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid relative paths",
			cfg: &Config{
				ConfigDir:      "./config",
				OutputFile:     "./openclaw.json",
				MasterTemplate: "./config/openclaw.json5",
			},
			wantErr: false,
		},
		{
			name: "path traversal in outputFile",
			cfg: &Config{
				ConfigDir:      "./config",
				OutputFile:     "../../../etc/crontab",
				MasterTemplate: "./config/openclaw.json5",
			},
			wantErr: true,
		},
		{
			name: "absolute path in masterTemplate",
			cfg: &Config{
				ConfigDir:      "./config",
				OutputFile:     "./openclaw.json",
				MasterTemplate: "/etc/passwd",
			},
			wantErr: true,
		},
		{
			name: "absolute path in configDir",
			cfg: &Config{
				ConfigDir:      "/tmp/evil",
				OutputFile:     "./openclaw.json",
				MasterTemplate: "./config/openclaw.json5",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(homeDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadRejectsOversizedConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".clawback.json5")

	// Write a file that exceeds the 10MB SafeReadFile limit (10<<20 + 1 bytes).
	// config.Load now delegates to SafeReadFile which enforces the 10MB cap.
	data := make([]byte, 10<<20+1)
	for i := range data {
		data[i] = ' '
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for oversized config, got nil")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected error to contain %q, got %q", "too large", err.Error())
	}
}

func TestLoadInvalidJSON5(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".clawback.json5")

	// Write malformed JSON5 content
	if err := os.WriteFile(path, []byte(`{this is not valid json5: [}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON5, got nil")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected error to contain %q, got %q", "parsing", err.Error())
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
