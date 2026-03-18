package cmd

import (
	"fmt"

	"github.com/jeff-french/clawback/internal/render"
	"github.com/spf13/cobra"
)

func newRenderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "render",
		Short: "Render openclaw.json from JSON5 source files",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := render.Render(homeDir, cfg)
			if err != nil {
				return err
			}

			if err := render.WriteOutput(homeDir, cfg, result); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Rendered %s\n", cfg.OutputPath(homeDir))
			return nil
		},
	}
}
