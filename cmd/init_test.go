package cmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitFromMonolith(t *testing.T) {
	dir := setupFixture(t, "monolith")

	out, err := executeCmd([]string{"--home", dir, "init"})
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, out)
	}

	// Verify key files were created.
	for _, f := range []string{
		".clawback.json5",
		"config/openclaw.json5",
		"config/env.json5",
		"config/agents.json5",
		"config/channels.json5",
		"config/plugins.json5",
		"config/meta.json5",
		"config/wizard.json5",
	} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify round-trip message.
	if !strings.Contains(out, "Round-trip verification passed") {
		t.Errorf("expected round-trip success message, got:\n%s", out)
	}

	// Verify diff is clean after init.
	diffOut, err := executeCmd([]string{"--home", dir, "diff"})
	if err != nil {
		t.Errorf("expected clean diff after init, got error: %v\nOutput: %s", err, diffOut)
	}
}

func TestInitDryRun(t *testing.T) {
	dir := setupFixture(t, "monolith")

	out, err := executeCmd([]string{"--home", dir, "init", "--dry-run"})
	if err != nil {
		t.Fatalf("init --dry-run failed: %v", err)
	}

	if !strings.Contains(out, "Would create") {
		t.Errorf("expected 'Would create' in output, got:\n%s", out)
	}

	// No files should be written.
	configDir := filepath.Join(dir, "config")
	if _, err := os.Stat(configDir); !errors.Is(err, fs.ErrNotExist) {
		t.Error("config directory should not exist after dry-run")
	}
}

func TestInitAlreadyExists(t *testing.T) {
	dir := setupFixture(t, "monolith")

	// Create config/ to trigger the conflict.
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := executeCmd([]string{"--home", dir, "init"})
	if err == nil {
		t.Fatal("expected error when config/ already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestInitForce(t *testing.T) {
	dir := setupFixture(t, "monolith")

	// Create config/ to trigger the conflict, then use --force.
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := executeCmd([]string{"--home", dir, "init", "--force"})
	if err != nil {
		t.Fatalf("init --force failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Round-trip verification passed") {
		t.Errorf("expected round-trip success, got:\n%s", out)
	}
}

func TestInitNoOpenclaw(t *testing.T) {
	dir := t.TempDir()

	_, err := executeCmd([]string{"--home", dir, "init"})
	if err == nil {
		t.Fatal("expected error when openclaw.json is missing")
	}
}

func TestInitWithArrays(t *testing.T) {
	dir := setupFixture(t, "monolith")

	out, err := executeCmd([]string{"--home", dir, "init"})
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, out)
	}

	// "bindings" is an array — it should be inlined in the master template,
	// not extracted to a separate file.
	bindingsFile := filepath.Join(dir, "config", "bindings.json5")
	if _, err := os.Stat(bindingsFile); !errors.Is(err, fs.ErrNotExist) {
		t.Error("bindings.json5 should not exist — arrays should be inlined")
	}

	// Check that bindings appears in the master template.
	template, err := os.ReadFile(filepath.Join(dir, "config", "openclaw.json5"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(template), "bindings:") {
		t.Error("master template should contain inlined bindings key")
	}
	if strings.Contains(string(template), `bindings: { $include`) {
		t.Error("bindings should not use $include — it's an array")
	}
}

func TestInitRoundTrip(t *testing.T) {
	dir := setupFixture(t, "monolith")

	// Run init (which now also writes the rendered openclaw.json).
	out, err := executeCmd([]string{"--home", dir, "init"})
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, out)
	}

	// Use diff command for definitive deep comparison.
	_, diffErr := executeCmd([]string{"--home", dir, "diff"})
	if diffErr != nil {
		t.Errorf("diff should be clean after init, got: %v", diffErr)
	}
}
