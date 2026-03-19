package json5

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveIncludes processes $include directives in a parsed JSON5 map.
// baseDir is the directory containing the file that was parsed.
// It returns the resolved map and a mapping of top-level keys to their source file paths.
// Includes are restricted to files within baseDir to prevent path traversal attacks.
// Circular includes are detected and reported as errors.
func ResolveIncludes(data map[string]any, baseDir string) (map[string]any, map[string]string, error) {
	root := filepath.Clean(baseDir)
	visited := make(map[string]bool)
	return resolveIncludesInternal(data, root, root, visited)
}

func resolveIncludesInternal(data map[string]any, baseDir, root string, visited map[string]bool) (map[string]any, map[string]string, error) {
	result := make(map[string]any, len(data))
	sources := make(map[string]string)

	for key, val := range data {
		resolved, source, err := resolveValue(val, baseDir, root, visited)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving %q: %w", key, err)
		}
		result[key] = resolved
		if source != "" {
			sources[key] = source
		}
	}

	return result, sources, nil
}

func resolveValue(val any, baseDir, root string, visited map[string]bool) (any, string, error) {
	obj, ok := val.(map[string]any)
	if !ok {
		return val, "", nil
	}

	// Check if this is an $include directive
	if len(obj) == 1 {
		if inc, ok := obj["$include"]; ok {
			path, ok := inc.(string)
			if !ok {
				return nil, "", fmt.Errorf("$include value must be a string, got %T", inc)
			}
			absPath := filepath.Clean(filepath.Join(baseDir, path))

			// Path traversal check: included files must stay within root directory.
			rel, err := filepath.Rel(root, absPath)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, "", fmt.Errorf("$include path %q escapes config directory", path)
			}

			// Circular include detection using DFS grey-marking.
			if visited[absPath] {
				return nil, "", fmt.Errorf("circular $include detected: %s", absPath)
			}
			visited[absPath] = true
			defer delete(visited, absPath)

			included, err := ParseFile(absPath)
			if err != nil {
				return nil, "", fmt.Errorf("including %q: %w", path, err)
			}
			// Recursively resolve includes in the included file
			resolved, _, err := resolveIncludesInternal(included, filepath.Dir(absPath), root, visited)
			if err != nil {
				return nil, "", err
			}
			return resolved, absPath, nil
		}
	}

	// Not an include — recursively resolve nested objects
	resolved := make(map[string]any, len(obj))
	for k, v := range obj {
		r, _, err := resolveValue(v, baseDir, root, visited)
		if err != nil {
			return nil, "", err
		}
		resolved[k] = r
	}
	return resolved, "", nil
}
