// Package cmd contains Cobra CLI commands.
//
// Responsibility:
//   - Define the root command and subcommands.
//   - Delegate to the appropriate application entry point.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"drawo/config"
)

var rootCmd = &cobra.Command{
	Use:   "drawo",
	Short: "Drawo multiplayer drawing game server",
	Long:  "Drawo is a production-quality multiplayer drawing game built with Go.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.Load()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
