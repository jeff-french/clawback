package json5

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/titanous/json5"
)

// maxFileSize is the largest config file we're willing to read (10 MB).
const maxFileSize = 10 << 20

// ParseFile reads and parses a JSON5 file into a map.
// It refuses to follow symlinks to prevent symlink-based path traversal attacks.
func ParseFile(path string) (map[string]any, error) {
	data, err := SafeReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// SafeReadFile reads a file after verifying it is not a symlink and is within
// the size limit. It opens the file first, then uses Fstat on the open
// descriptor to eliminate the TOCTOU window between check and read.
func SafeReadFile(path string) ([]byte, error) {
	// Lstat first to reject symlinks before opening.
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to read symlink: %s", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Fstat the open fd to verify properties haven't changed since Lstat.
	finfo, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !finfo.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	if finfo.Size() > maxFileSize {
		return nil, fmt.Errorf("file too large (%d bytes, limit %d): %s", finfo.Size(), maxFileSize, path)
	}
	return io.ReadAll(f)
}

// SafeWriteFile writes data to path atomically, refusing to follow symlinks.
// It writes to a temp file in the same directory, then renames it into place.
func SafeWriteFile(path string, data []byte, perm os.FileMode) error {
	// Check that the target is not a symlink (if it already exists).
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlink: %s", path)
		}
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".clawback-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up temp file on any failure path.
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("setting permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Re-check symlink right before rename to narrow the TOCTOU window.
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlink: %s", path)
		}
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}
	success = true
	return nil
}

// Parse parses JSON5 bytes into a map.
func Parse(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := json5.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
