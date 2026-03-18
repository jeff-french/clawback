package json5

import (
	"fmt"
	"os"

	"github.com/titanous/json5"
)

// ParseFile reads and parses a JSON5 file into a map.
// It refuses to follow symlinks to prevent symlink-based path traversal attacks.
func ParseFile(path string) (map[string]any, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to read symlink: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse parses JSON5 bytes into a map.
func Parse(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := json5.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
