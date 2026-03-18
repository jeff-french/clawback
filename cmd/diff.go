package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jeff-french/clawback/internal/jsonutil"
	"github.com/jeff-french/clawback/internal/render"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var quiet bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between rendered output and current openclaw.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := render.Render(homeDir, cfg)
			if err != nil {
				return err
			}

			// Read current output file
			outputPath := cfg.OutputPath(homeDir)
			existingData, err := os.ReadFile(outputPath)
			if err != nil {
				if os.IsNotExist(err) {
					if quiet {
						os.Exit(1)
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Output file does not exist yet. Run 'clawback render' to create it.")
					os.Exit(1)
				}
				return fmt.Errorf("reading %s: %w", outputPath, err)
			}

			var existing map[string]any
			if err := json.Unmarshal(existingData, &existing); err != nil {
				return fmt.Errorf("parsing %s: %w", outputPath, err)
			}

			diffs := jsonutil.Compare(existing, result.Data)

			if len(diffs) == 0 {
				if !quiet {
					fmt.Fprintln(cmd.OutOrStdout(), "No differences found.")
				}
				return nil
			}

			if quiet {
				os.Exit(1)
			}

			if jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(diffs)
			}

			fmt.Fprint(cmd.OutOrStdout(), jsonutil.FormatDiffs(diffs))
			os.Exit(1)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Exit code only (0 = clean, 1 = drifted)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output diff as JSON")

	return cmd
}
