package cmd

import "github.com/spf13/cobra"

func newMigrateCmd() *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migration commands",
	}

	migrateCmd.AddCommand(newMigrateExportCmd())

	return migrateCmd
}
