package json5

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, result map[string]any)
	}{
		{
			name:  "simple object",
			input: `{ "key": "value" }`,
			check: func(t *testing.T, result map[string]any) {
				if result["key"] != "value" {
					t.Errorf("expected key=value, got %v", result["key"])
				}
			},
		},
		{
			name:  "unquoted keys",
			input: `{ key: "value", num: 42 }`,
			check: func(t *testing.T, result map[string]any) {
				if result["key"] != "value" {
					t.Errorf("expected key=value, got %v", result["key"])
				}
			},
		},
		{
			name: "with comments",
			input: `{
				// line comment
				key: "value",
				/* block comment */
				other: true,
			}`,
			check: func(t *testing.T, result map[string]any) {
				if result["key"] != "value" {
					t.Errorf("expected key=value, got %v", result["key"])
				}
				if result["other"] != true {
					t.Errorf("expected other=true, got %v", result["other"])
				}
			},
		},
		{
			name:    "invalid json5",
			input:   `{ key: }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil && err == nil {
				tt.check(t, result)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json5")
	if err := os.WriteFile(path, []byte(`{ key: "value" }`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key=value, got %v", result["key"])
	}
}

func TestParseFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()

	// Create the real file
	real := filepath.Join(dir, "real.json5")
	if err := os.WriteFile(real, []byte(`{ key: "value" }`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink pointing to it
	link := filepath.Join(dir, "link.json5")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	_, err := ParseFile(link)
	if err == nil {
		t.Fatal("expected error when reading symlink, got nil")
	}
}

func TestSafeWriteFile(t *testing.T) {
	t.Run("happy path writes file with correct content and permissions", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.json5")
		content := []byte(`{"hello": "world"}`)

		if err := SafeWriteFile(path, content, 0o600); err != nil {
			t.Fatalf("SafeWriteFile() unexpected error: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading written file: %v", err)
		}
		if string(got) != string(content) {
			t.Errorf("content mismatch: got %q, want %q", got, content)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat written file: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("permissions = %o, want 0600", perm)
		}
	})

	t.Run("rejects symlink at target path", func(t *testing.T) {
		dir := t.TempDir()

		real := filepath.Join(dir, "real.json5")
		if err := os.WriteFile(real, []byte("original"), 0o644); err != nil {
			t.Fatal(err)
		}

		link := filepath.Join(dir, "link.json5")
		if err := os.Symlink(real, link); err != nil {
			t.Skip("symlinks not supported on this platform")
		}

		err := SafeWriteFile(link, []byte("malicious"), 0o600)
		if err == nil {
			t.Fatal("expected error when writing through symlink, got nil")
		}
		if !strings.Contains(err.Error(), "symlink") {
			t.Errorf("error should mention symlink, got: %v", err)
		}

		// Verify the original file was not modified.
		got, _ := os.ReadFile(real)
		if string(got) != "original" {
			t.Errorf("original file was modified through symlink")
		}
	})

	t.Run("non-existent directory fails gracefully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "no-such-dir", "file.json5")

		err := SafeWriteFile(path, []byte("data"), 0o600)
		if err == nil {
			t.Fatal("expected error for non-existent directory, got nil")
		}
	})

	t.Run("creates new file when target does not exist", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "brand-new.json5")

		if err := SafeWriteFile(path, []byte("new content"), 0o600); err != nil {
			t.Fatalf("SafeWriteFile() unexpected error: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading new file: %v", err)
		}
		if string(got) != "new content" {
			t.Errorf("content = %q, want %q", got, "new content")
		}
	})

	t.Run("overwrites existing regular file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "existing.json5")

		if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := SafeWriteFile(path, []byte("new"), 0o600); err != nil {
			t.Fatalf("SafeWriteFile() unexpected error: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading overwritten file: %v", err)
		}
		if string(got) != "new" {
			t.Errorf("content = %q, want %q", got, "new")
		}
	})
}

func TestSafeReadFile(t *testing.T) {
	t.Run("rejects file larger than maxFileSize", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "big.bin")

		// Create a file of exactly maxFileSize+1 bytes.
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Truncate(maxFileSize + 1); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()

		_, err = SafeReadFile(path)
		if err == nil {
			t.Fatal("expected error for oversized file, got nil")
		}
		if !strings.Contains(err.Error(), "too large") {
			t.Errorf("error should mention size, got: %v", err)
		}
	})

	t.Run("rejects symlink", func(t *testing.T) {
		dir := t.TempDir()

		real := filepath.Join(dir, "real.txt")
		if err := os.WriteFile(real, []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}

		link := filepath.Join(dir, "link.txt")
		if err := os.Symlink(real, link); err != nil {
			t.Skip("symlinks not supported on this platform")
		}

		_, err := SafeReadFile(link)
		if err == nil {
			t.Fatal("expected error when reading symlink, got nil")
		}
		if !strings.Contains(err.Error(), "symlink") {
			t.Errorf("error should mention symlink, got: %v", err)
		}
	})

	t.Run("happy path reads regular file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "normal.txt")
		want := []byte("hello world")

		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}

		got, err := SafeReadFile(path)
		if err != nil {
			t.Fatalf("SafeReadFile() unexpected error: %v", err)
		}
		if string(got) != string(want) {
			t.Errorf("content = %q, want %q", got, want)
		}
	})
}
