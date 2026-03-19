package json5

import (
	"fmt"
	"os"

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
// the size limit. It is exported so other packages can reuse the same checks.
func SafeReadFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to read symlink: %s", path)
	}
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("file too large (%d bytes, limit %d): %s", info.Size(), maxFileSize, path)
	}
	return os.ReadFile(path)
}

// Parse parses JSON5 bytes into a map.
func Parse(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := json5.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
