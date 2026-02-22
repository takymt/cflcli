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

// ProfileFlag returns the active profile flag.
func ProfileFlag() string {
	return profileFlag
}

// OutputFlag returns the active output flag.
func OutputFlag() string {
	return outputFlag
}

// SetProfileFlag sets the profile flag.
func SetProfileFlag(value string) {
	profileFlag = value
}

// SetOutputFlag sets the output flag.
func SetOutputFlag(value string) {
	outputFlag = value
}

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

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newUseCmd())
	rootCmd.AddCommand(newPageCmd())
	rootCmd.AddCommand(newMigrateCmd())

	return rootCmd
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
