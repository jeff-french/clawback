package json5

import (
	"os"

	"github.com/titanous/json5"
)

// ParseFile reads and parses a JSON5 file into a map.
func ParseFile(path string) (map[string]any, error) {
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
