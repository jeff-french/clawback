package render

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeff-french/clawback/internal/config"
	"github.com/jeff-french/clawback/internal/json5"
	"github.com/jeff-french/clawback/internal/jsonutil"
)

// Result holds the output of a render operation.
type Result struct {
	Data    map[string]any
	JSON    []byte
	Sources map[string]string // top-level key → source file path
}

// Render performs the full render pipeline:
// 1. Parse master template
// 2. Resolve $include directives
// 3. Apply passthrough sections from existing output
// 4. Marshal to JSON
func Render(homeDir string, cfg *config.Config) (*Result, error) {
	templatePath := cfg.MasterTemplatePath(homeDir)
	templateData, err := json5.ParseFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("parsing master template %s: %w", templatePath, err)
	}

	baseDir := filepath.Dir(templatePath)
	resolved, sources, err := json5.ResolveIncludes(templateData, baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolving includes: %w", err)
	}

	// Apply passthrough sections from existing output file.
	// Silently skip only when the file doesn't exist yet (first render).
	outputPath := cfg.OutputPath(homeDir)
	existing, err := readExistingOutput(outputPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading existing output %s: %w", outputPath, err)
	}
	if existing != nil {
		applyPassthrough(resolved, existing, cfg)
	}

	jsonBytes, err := json.MarshalIndent(resolved, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}
	jsonBytes = append(jsonBytes, '\n')

	return &Result{
		Data:    resolved,
		JSON:    jsonBytes,
		Sources: sources,
	}, nil
}

func readExistingOutput(path string) (map[string]any, error) {
	data, err := json5.SafeReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func applyPassthrough(rendered, existing map[string]any, cfg *config.Config) {
	for _, path := range cfg.Passthrough {
		val, ok := jsonutil.GetPath(existing, path)
		if ok {
			jsonutil.SetPath(rendered, path, val)
		}
	}
}

// WriteOutput writes the rendered JSON to the output file.
func WriteOutput(homeDir string, cfg *config.Config, result *Result) error {
	outputPath := cfg.OutputPath(homeDir)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, result.JSON, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}
	return nil
}
