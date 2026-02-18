package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	profileFlag string
	outputFlag  string
	verboseFlag bool
)

// NewRootCmd creates the root command for cfl CLI.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "cfl",
		Short:        "CLI tool for Confluence Cloud REST API v2",
		Version:      version,
		SilenceUsage: true,
	}

	rootCmd.Flags().BoolP("version", "V", false, "version for cfl")
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "profile name (temporary override)")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "table", "output format (json | table)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newUseCmd())

	return rootCmd
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
