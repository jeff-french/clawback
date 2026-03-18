package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeff-french/clawback/internal/config"
	"github.com/spf13/cobra"
)

var (
	homeDir string
	cfg     *config.Config
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "clawback",
		Short: "Manage modular OpenClaw configuration",
		Long:  "A CLI tool that manages modular OpenClaw configuration. It treats openclaw.json as a build artifact rendered from JSON5 source files via $include directives.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if homeDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("determining home directory: %w", err)
				}
				homeDir = filepath.Join(home, ".openclaw")
			}

			var err error
			cfg, err = config.Load(homeDir)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&homeDir, "home", "", "OpenClaw home directory (default: ~/.openclaw)")

	rootCmd.AddCommand(newRenderCmd())
	rootCmd.AddCommand(newDiffCmd())
	rootCmd.AddCommand(newSyncCmd())

	return rootCmd
}
