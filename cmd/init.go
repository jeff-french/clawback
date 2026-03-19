package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jeff-french/clawback/internal/config"
	"github.com/jeff-french/clawback/internal/json5"
	"github.com/jeff-french/clawback/internal/jsonutil"
	"github.com/jeff-french/clawback/internal/render"
	"github.com/spf13/cobra"
)

type keyEntry struct {
	key   string
	value any
}

func newInitCmd(ctx *appContext) *cobra.Command {
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize modular config from an existing openclaw.json",
		Long:  "Decomposes a monolithic openclaw.json into modular JSON5 source files under config/, creates a master template with $include directives, and generates .clawback.json5.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, ctx.homeDir, dryRun, force)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be created without writing files")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config directory")

	return cmd
}

func runInit(cmd *cobra.Command, homeDir string, dryRun, force bool) error {
	// 1. Read the monolithic openclaw.json.
	outputPath := filepath.Join(homeDir, config.DefaultOutputFile)
	data, err := json5.SafeReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w\nRun this command from a directory containing an existing openclaw.json", outputPath, err)
	}

	// Use json.Number to preserve integer vs float distinction.
	var parsed map[string]any
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	if err := dec.Decode(&parsed); err != nil {
		return fmt.Errorf("parsing %s: %w", outputPath, err)
	}

	// 2. Check for existing config directory.
	configDir := filepath.Join(homeDir, config.DefaultConfigDir)
	if !force {
		if info, err := os.Stat(configDir); err == nil && info.IsDir() {
			return fmt.Errorf("config directory already exists: %s\nUse --force to overwrite", configDir)
		}
	}

	// 3. Classify top-level keys.
	var extracted []keyEntry // objects → separate files
	var inlined []keyEntry   // arrays/primitives → inline in template

	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := parsed[k]
		if _, isObj := v.(map[string]any); isObj {
			extracted = append(extracted, keyEntry{k, v})
		} else {
			inlined = append(inlined, keyEntry{k, v})
		}
	}

	if dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Would create:")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", filepath.Join(homeDir, ".clawback.json5"))
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", filepath.Join(configDir, "openclaw.json5"))
		for _, e := range extracted {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", filepath.Join(configDir, e.key+".json5"))
		}
		if len(inlined) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nInlined in master template (non-object keys):")
			for _, e := range inlined {
				fmt.Fprintf(cmd.OutOrStdout(), " %s", e.key)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
		return nil
	}

	// 4. Create config directory.
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// 5. Write section files.
	for _, e := range extracted {
		obj := e.value.(map[string]any)
		content := json5.FormatObject(obj) + "\n"
		path := filepath.Join(configDir, e.key+".json5")
		if err := json5.SafeWriteFile(path, []byte(content), 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
	}

	// 6. Write master template.
	template := buildMasterTemplate(extracted, inlined)
	templatePath := filepath.Join(configDir, "openclaw.json5")
	if err := json5.SafeWriteFile(templatePath, []byte(template), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", templatePath, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", templatePath)

	// 7. Write .clawback.json5.
	clawbackCfg := buildClawbackConfig()
	cfgPath := filepath.Join(homeDir, ".clawback.json5")
	if err := json5.SafeWriteFile(cfgPath, []byte(clawbackCfg), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", cfgPath, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", cfgPath)

	// 8. Round-trip verification.
	cfg, err := config.Load(homeDir)
	if err != nil {
		return fmt.Errorf("loading generated config: %w", err)
	}
	result, err := render.Render(homeDir, cfg)
	if err != nil {
		return fmt.Errorf("verifying round-trip: %w", err)
	}

	diffs := jsonutil.Compare(result.Data, parsed)
	if len(diffs) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nWarning: %d difference(s) detected in round-trip verification:\n", len(diffs))
		fmt.Fprint(cmd.OutOrStdout(), jsonutil.FormatDiffs(diffs))
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "\nRound-trip verification passed — rendered output matches original.")
	}

	return nil
}

func buildMasterTemplate(extracted []keyEntry, inlined []keyEntry) string {
	var b strings.Builder
	b.WriteString("{\n")

	for _, e := range extracted {
		key := e.key
		if json5.NeedsQuoting(key) {
			key = fmt.Sprintf("%q", key)
		}
		fmt.Fprintf(&b, "  %s: { $include: \"./%s.json5\" },\n", key, e.key)
	}

	for _, e := range inlined {
		key := e.key
		if json5.NeedsQuoting(key) {
			key = fmt.Sprintf("%q", key)
		}
		fmt.Fprintf(&b, "  %s: %s,\n", key, json5.FormatValue(e.value, 1))
	}

	b.WriteString("}\n")
	return b.String()
}

func buildClawbackConfig() string {
	return `{
  configDir: "./config",
  outputFile: "./openclaw.json",
  masterTemplate: "./config/openclaw.json5",
  passthrough: [
    "meta",
    "wizard",
    "plugins.installs",
  ],
}
`
}
