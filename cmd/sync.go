package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeff-french/clawback/internal/config"
	"github.com/jeff-french/clawback/internal/json5"
	"github.com/jeff-french/clawback/internal/jsonutil"
	"github.com/jeff-french/clawback/internal/render"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Backport changes from openclaw.json to JSON5 source files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(cmd, homeDir, cfg, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without modifying files")

	return cmd
}

func runSync(cmd *cobra.Command, homeDir string, cfg *config.Config, dryRun bool) error {
	// Render what config files would produce
	result, err := render.Render(homeDir, cfg)
	if err != nil {
		return err
	}

	// Read current openclaw.json
	outputPath := cfg.OutputPath(homeDir)
	existingData, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", outputPath, err)
	}

	var existing map[string]any
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return fmt.Errorf("parsing %s: %w", outputPath, err)
	}

	// Compare: existing openclaw.json vs rendered
	diffs := jsonutil.Compare(result.Data, existing)

	if len(diffs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Already in sync.")
		return nil
	}

	// Group diffs by source file
	fileEdits := make(map[string][]jsonutil.Diff)
	for _, d := range diffs {
		// For passthrough sections, we copy from openclaw.json → config
		// For non-passthrough, we also backport new/changed keys from openclaw.json → config
		// But removed keys (in openclaw.json but not config) — keep config version
		if d.Type == jsonutil.DiffRemoved {
			// Key is in rendered (config) but not in openclaw.json — keep config version
			continue
		}

		sourceFile := jsonutil.OwningFile(result.Sources, d.Path)
		if sourceFile == "" {
			// Falls back to master template
			sourceFile = cfg.MasterTemplatePath(homeDir)
		}
		fileEdits[sourceFile] = append(fileEdits[sourceFile], d)
	}

	if dryRun {
		for file, edits := range fileEdits {
			rel, _ := filepath.Rel(homeDir, file)
			fmt.Fprintf(cmd.OutOrStdout(), "Would modify %s:\n", rel)
			for _, d := range edits {
				switch d.Type {
				case jsonutil.DiffAdded:
					fmt.Fprintf(cmd.OutOrStdout(), "  + %s\n", d.Path)
				case jsonutil.DiffChanged:
					fmt.Fprintf(cmd.OutOrStdout(), "  ~ %s\n", d.Path)
				}
			}
		}
		return nil
	}

	// Apply edits
	for file, edits := range fileEdits {
		if err := applyEdits(file, edits, result.Sources); err != nil {
			return fmt.Errorf("editing %s: %w", file, err)
		}
		rel, _ := filepath.Rel(homeDir, file)
		fmt.Fprintf(cmd.OutOrStdout(), "Updated %s\n", rel)
	}

	// Re-render after backporting
	reResult, err := render.Render(homeDir, cfg)
	if err != nil {
		return fmt.Errorf("re-rendering: %w", err)
	}
	if err := render.WriteOutput(homeDir, cfg, reResult); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Re-rendered %s\n", cfg.OutputPath(homeDir))
	return nil
}

func applyEdits(filePath string, diffs []jsonutil.Diff, sources map[string]string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	text := string(content)

	for _, d := range diffs {
		// Get the local key within this file
		localKey := localKeyForFile(d.Path, filePath, sources)

		valueJSON, err := json.Marshal(d.NewValue)
		if err != nil {
			return fmt.Errorf("marshaling value for %s: %w", d.Path, err)
		}

		switch d.Type {
		case jsonutil.DiffAdded:
			text = json5.AppendToObject(text, localKey, string(valueJSON))
		case jsonutil.DiffChanged:
			text = json5.SetValue(text, localKey, string(valueJSON))
		}
	}

	return os.WriteFile(filePath, []byte(text), 0o600)
}

// localKeyForFile extracts the local key name from a full JSON path,
// relative to the file that owns that section.
func localKeyForFile(path string, filePath string, sources map[string]string) string {
	// If the path has a dot, the first segment is the top-level key
	// that was $included from this file. The rest is the local path.
	parts := strings.SplitN(path, ".", 2)
	topKey := parts[0]

	// Check if this top-level key is sourced from this file
	if src, ok := sources[topKey]; ok && src == filePath {
		if len(parts) > 1 {
			return parts[1]
		}
		// The diff is for the entire included section — use the path as-is
		return path
	}

	// Key is in the master template directly
	return path
}

