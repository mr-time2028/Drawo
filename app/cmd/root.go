package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		exitCode, _ := fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode)
	}
}
