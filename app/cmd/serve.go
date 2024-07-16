package cmd

import (
	"drawo/pkg/serve"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve application on dev server",
	Long:  "Application will be served on host and port defined in config.yml file",
	Run: func(cmd *cobra.Command, args []string) {
		serve.Serve()
	},
}
