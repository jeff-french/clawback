package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeff-french/clawback/internal/config"
	"github.com/spf13/cobra"
)

// appContext holds resolved runtime state, replacing package-level globals.
type appContext struct {
	homeDir string
	cfg     *config.Config
}

// NewRootCmd creates the root command. Pass a version string (e.g. "1.0.0")
// to enable the built-in --version flag; pass "" to omit it.
func NewRootCmd(opts ...string) *cobra.Command {
	var ctx appContext
	ver := ""
	if len(opts) > 0 {
		ver = opts[0]
	}

	rootCmd := &cobra.Command{
		Use:     "clawback",
		Short:   "Manage modular OpenClaw configuration",
		Long:    "A CLI tool that manages modular OpenClaw configuration. It treats openclaw.json as a build artifact rendered from JSON5 source files via $include directives.",
		Version: ver,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if ctx.homeDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("determining home directory: %w", err)
				}
				ctx.homeDir = filepath.Join(home, ".openclaw")
			}

			var err error
			ctx.cfg, err = config.Load(ctx.homeDir)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if err := ctx.cfg.Validate(ctx.homeDir); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&ctx.homeDir, "home", "", "OpenClaw home directory (default: ~/.openclaw)")

	rootCmd.AddCommand(newRenderCmd(&ctx))
	rootCmd.AddCommand(newDiffCmd(&ctx))
	rootCmd.AddCommand(newSyncCmd(&ctx))

	return rootCmd
}
