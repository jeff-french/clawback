package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupFixture copies a testdata fixture into a temporary directory so that
// tests can freely mutate files without affecting the repository.
func setupFixture(t *testing.T, fixture string) string {
	t.Helper()
	src := filepath.Join("..", "testdata", fixture)
	dst := t.TempDir()

	// Copy the config directory and any other files.
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
	if err != nil {
		t.Fatalf("copying fixture %s: %v", fixture, err)
	}
	return dst
}

// executeCmd creates a root command, sets --home, and executes the given args.
// It captures stdout and returns it along with any error.
func executeCmd(args []string) (string, error) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// mustRender renders the fixture and returns the home dir path.
func mustRender(t *testing.T, dir string) {
	t.Helper()
	if _, err := executeCmd([]string{"--home", dir, "render"}); err != nil {
		t.Fatalf("render failed: %v", err)
	}
}

// mutateOutput reads openclaw.json, applies fn to the parsed object, and writes it back.
func mutateOutput(t *testing.T, dir string, fn func(map[string]any)) {
	t.Helper()
	outputPath := filepath.Join(dir, "openclaw.json")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("parsing output: %v", err)
	}
	fn(obj)
	modified, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		t.Fatalf("marshaling output: %v", err)
	}
	if err := os.WriteFile(outputPath, modified, 0o600); err != nil {
		t.Fatalf("writing output: %v", err)
	}
}

// --- render command tests ---

func TestRenderSimple(t *testing.T) {
	dir := setupFixture(t, "simple")
	out, err := executeCmd([]string{"--home", dir, "render"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(out, "Rendered") {
		t.Errorf("expected 'Rendered' in output, got: %s", out)
	}

	// Verify the output file was created and is valid JSON.
	outputPath := filepath.Join(dir, "openclaw.json")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if result["name"] != "my-openclaw" {
		t.Errorf("expected name=my-openclaw, got %v", result["name"])
	}
	env, ok := result["env"].(map[string]any)
	if !ok {
		t.Fatal("expected env to be a map")
	}
	if env["debug"] != true {
		t.Errorf("expected env.debug=true, got %v", env["debug"])
	}
}

func TestRenderWithComments(t *testing.T) {
	dir := setupFixture(t, "with-comments")
	out, err := executeCmd([]string{"--home", dir, "render"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(out, "Rendered") {
		t.Errorf("expected 'Rendered' in output, got: %s", out)
	}

	outputPath := filepath.Join(dir, "openclaw.json")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if result["name"] != "test-app" {
		t.Errorf("expected name=test-app, got %v", result["name"])
	}
	plugins, ok := result["plugins"].(map[string]any)
	if !ok {
		t.Fatal("expected plugins to be a map")
	}
	if plugins["enabled"] != true {
		t.Errorf("expected plugins.enabled=true, got %v", plugins["enabled"])
	}
}

func TestRenderMatchesExpected(t *testing.T) {
	fixtures := []string{"simple", "with-comments"}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			dir := setupFixture(t, fixture)

			_, err := executeCmd([]string{"--home", dir, "render"})
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}

			// Read rendered output
			rendered, err := os.ReadFile(filepath.Join(dir, "openclaw.json"))
			if err != nil {
				t.Fatal(err)
			}
			// Read expected output
			expected, err := os.ReadFile(filepath.Join(dir, "expected.json"))
			if err != nil {
				t.Fatal(err)
			}

			// Compare as parsed JSON (ignoring whitespace differences)
			var renderedJSON, expectedJSON any
			if err := json.Unmarshal(rendered, &renderedJSON); err != nil {
				t.Fatalf("rendered output invalid JSON: %v", err)
			}
			if err := json.Unmarshal(expected, &expectedJSON); err != nil {
				t.Fatalf("expected output invalid JSON: %v", err)
			}

			renderedNorm, _ := json.MarshalIndent(renderedJSON, "", "  ")
			expectedNorm, _ := json.MarshalIndent(expectedJSON, "", "  ")
			if string(renderedNorm) != string(expectedNorm) {
				t.Errorf("rendered output does not match expected.\nGot:\n%s\nWant:\n%s", renderedNorm, expectedNorm)
			}
		})
	}
}

// --- diff command tests ---

func TestDiffClean(t *testing.T) {
	dir := setupFixture(t, "simple")

	// First render to create the output file.
	_, err := executeCmd([]string{"--home", dir, "render"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Diff should be clean (no error).
	out, err := executeCmd([]string{"--home", dir, "diff"})
	if err != nil {
		t.Fatalf("diff should be clean, got error: %v", err)
	}
	if !strings.Contains(out, "No differences") {
		t.Errorf("expected 'No differences' in output, got: %s", out)
	}
}

func TestDiffDrifted(t *testing.T) {
	dir := setupFixture(t, "simple")
	mustRender(t, dir)
	mutateOutput(t, dir, func(obj map[string]any) {
		obj["name"] = "drifted-name"
	})

	// Diff should detect drift.
	out, err := executeCmd([]string{"--home", dir, "diff"})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected ExitError with code 1, got: %v", err)
	}
	if !strings.Contains(out, "name") {
		t.Errorf("expected diff output to mention 'name', got: %s", out)
	}
}

func TestDiffQuiet(t *testing.T) {
	dir := setupFixture(t, "simple")
	mustRender(t, dir)
	mutateOutput(t, dir, func(obj map[string]any) {
		obj["extra"] = "key"
	})

	out, err := executeCmd([]string{"--home", dir, "diff", "--quiet"})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected ExitError with code 1, got: %v", err)
	}
	// Quiet mode should produce no output.
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no output in quiet mode, got: %s", out)
	}
}

func TestDiffJSON(t *testing.T) {
	dir := setupFixture(t, "simple")
	mustRender(t, dir)
	mutateOutput(t, dir, func(obj map[string]any) {
		obj["name"] = "changed"
	})

	out, err := executeCmd([]string{"--home", dir, "diff", "--json"})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected ExitError with code 1, got: %v", err)
	}
	// Output should be valid JSON.
	var diffs []any
	if err := json.Unmarshal([]byte(out), &diffs); err != nil {
		t.Fatalf("expected JSON output, got: %s (parse error: %v)", out, err)
	}
	if len(diffs) == 0 {
		t.Error("expected at least one diff in JSON output")
	}
}

func TestDiffNoOutputFile(t *testing.T) {
	dir := setupFixture(t, "simple")

	// Don't render — output file doesn't exist.
	out, err := executeCmd([]string{"--home", dir, "diff"})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected ExitError with code 1, got: %v", err)
	}
	if !strings.Contains(out, "does not exist") {
		t.Errorf("expected 'does not exist' message, got: %s", out)
	}
}

// --- sync command tests ---

func TestSyncAlreadyInSync(t *testing.T) {
	dir := setupFixture(t, "simple")

	// Render to create the output file.
	_, err := executeCmd([]string{"--home", dir, "render"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Sync when already in sync.
	out, err := executeCmd([]string{"--home", dir, "sync"})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if !strings.Contains(out, "Already in sync") {
		t.Errorf("expected 'Already in sync', got: %s", out)
	}
}

func TestSyncBackport(t *testing.T) {
	dir := setupFixture(t, "simple")
	mustRender(t, dir)
	mutateOutput(t, dir, func(obj map[string]any) {
		env := obj["env"].(map[string]any)
		env["debug"] = false
	})

	// Sync should backport changes.
	out, err := executeCmd([]string{"--home", dir, "sync"})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if !strings.Contains(out, "Updated") {
		t.Errorf("expected 'Updated' in output, got: %s", out)
	}
	if !strings.Contains(out, "Re-rendered") {
		t.Errorf("expected 'Re-rendered' in output, got: %s", out)
	}

	// Verify the source file was modified.
	envSrc, _ := os.ReadFile(filepath.Join(dir, "config", "env.json5"))
	if !strings.Contains(string(envSrc), "false") {
		t.Errorf("expected env.json5 to contain 'false' after sync, got:\n%s", envSrc)
	}
}

func TestSyncDryRun(t *testing.T) {
	dir := setupFixture(t, "simple")
	mustRender(t, dir)
	mutateOutput(t, dir, func(obj map[string]any) {
		env := obj["env"].(map[string]any)
		env["debug"] = false
	})

	// Dry-run should show what would change but not modify files.
	envBefore, _ := os.ReadFile(filepath.Join(dir, "config", "env.json5"))
	out, err := executeCmd([]string{"--home", dir, "sync", "--dry-run"})
	if err != nil {
		t.Fatalf("sync --dry-run failed: %v", err)
	}
	if !strings.Contains(out, "Would modify") {
		t.Errorf("expected 'Would modify' in output, got: %s", out)
	}

	// Source file should be unchanged.
	envAfter, _ := os.ReadFile(filepath.Join(dir, "config", "env.json5"))
	if string(envBefore) != string(envAfter) {
		t.Error("dry-run should not modify source files")
	}
}

func TestSyncAddedKeyInOutput(t *testing.T) {
	dir := setupFixture(t, "simple")
	mustRender(t, dir)
	mutateOutput(t, dir, func(obj map[string]any) {
		env := obj["env"].(map[string]any)
		env["newFeatureFlag"] = true
	})

	// Sync should backport the added key to the source file.
	out, err := executeCmd([]string{"--home", dir, "sync"})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if !strings.Contains(out, "Updated") {
		t.Errorf("expected 'Updated' in output, got: %s", out)
	}

	// Verify the source file now contains the new key.
	envSrc, _ := os.ReadFile(filepath.Join(dir, "config", "env.json5"))
	if !strings.Contains(string(envSrc), "newFeatureFlag") {
		t.Errorf("expected env.json5 to contain 'newFeatureFlag' after sync, got:\n%s", envSrc)
	}

	// Verify a subsequent diff is clean (sync + re-render produced consistent state).
	_, err = executeCmd([]string{"--home", dir, "diff"})
	if err != nil {
		t.Errorf("expected clean diff after sync, got error: %v", err)
	}
}

func TestSyncRemovedKeyIgnored(t *testing.T) {
	dir := setupFixture(t, "simple")
	mustRender(t, dir)

	// Capture the original source file content.
	envPath := filepath.Join(dir, "config", "env.json5")
	envBefore, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("reading env.json5: %v", err)
	}

	mutateOutput(t, dir, func(obj map[string]any) {
		env := obj["env"].(map[string]any)
		delete(env, "timeout")
	})

	// Run sync.
	_, err = executeCmd([]string{"--home", dir, "sync"})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// The source file should still contain the "timeout" key — removals in
	// openclaw.json should not strip keys from config sources.
	envAfter, _ := os.ReadFile(envPath)
	if !strings.Contains(string(envAfter), "timeout") {
		t.Errorf("expected env.json5 to still contain 'timeout' after sync, got:\n%s", envAfter)
	}

	// The rest of the source file should be unchanged for the timeout line.
	if !strings.Contains(string(envBefore), "timeout") {
		t.Fatal("precondition failed: env.json5 should contain 'timeout' before sync")
	}
}

// --- error case tests ---

func TestRenderMissingConfig(t *testing.T) {
	dir := t.TempDir()
	// No config files at all — master template will be missing.
	_, err := executeCmd([]string{"--home", dir, "render"})
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
}

func TestRenderInvalidHomePath(t *testing.T) {
	_, err := executeCmd([]string{"--home", "/nonexistent/path/that/does/not/exist", "render"})
	if err == nil {
		t.Fatal("expected error for invalid home path, got nil")
	}
}

func TestSyncNoOutputFile(t *testing.T) {
	dir := setupFixture(t, "simple")
	// Don't render — no output file exists.
	_, err := executeCmd([]string{"--home", dir, "sync"})
	if err == nil {
		t.Fatal("expected error when output file missing for sync")
	}
}

func TestExitError(t *testing.T) {
	e := &ExitError{Code: 42}
	if e.Error() != "exit status 42" {
		t.Errorf("ExitError.Error() = %q, want %q", e.Error(), "exit status 42")
	}
	if e.Code != 42 {
		t.Errorf("expected code 42, got %d", e.Code)
	}
}
