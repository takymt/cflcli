package cli

import "github.com/spf13/cobra"

func (a *App) newRootCommand(args []string, workdir string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "cfl",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.SetOut(a.stdout)
	rootCmd.SetErr(a.stdout)
	rootCmd.SetArgs(args)

	pageCmd := &cobra.Command{
		Use:   "page",
		Short: "Manage Confluence pages",
	}
	attachmentCmd := &cobra.Command{
		Use:   "attachment",
		Short: "Manage page attachments",
	}

	var (
		newSpaceKey string
		newParentID string
		newWatch    bool
	)
	pageNewCmd := &cobra.Command{
		Use:   "new <title>.md",
		Args:  cobra.ExactArgs(1),
		Short: "Create a local markdown file and a Confluence page",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return a.runPageNew(cmd.Context(), workdir, cmdArgs[0], newSpaceKey, newParentID, newWatch)
		},
	}
	pageNewCmd.Flags().StringVar(&newSpaceKey, "space-key", "", "Confluence space key")
	pageNewCmd.Flags().StringVar(&newParentID, "parent-id", "", "Confluence parent page id")
	pageNewCmd.Flags().BoolVar(&newWatch, "watch", false, "Watch the created file and sync on changes")
	_ = pageNewCmd.MarkFlagRequired("space-key")

	var syncWatch bool
	pageSyncCmd := &cobra.Command{
		Use:   "sync <file>.md",
		Args:  cobra.ExactArgs(1),
		Short: "Sync a local markdown file to Confluence",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return a.runPageSync(cmd.Context(), workdir, cmdArgs[0], syncWatch)
		},
	}
	pageSyncCmd.Flags().BoolVar(&syncWatch, "watch", false, "Watch the file and sync on changes")
	pageCmd.AddCommand(pageNewCmd, pageSyncCmd)

	var (
		attachmentPutPageID string
		attachmentDelPageID string
	)
	attachmentPutCmd := &cobra.Command{
		Use:   "put <file>",
		Args:  cobra.ExactArgs(1),
		Short: "Upload or update an attachment on a Confluence page",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			filePath := resolvePath(workdir, cmdArgs[0])
			return a.runAttachmentPut(cmd.Context(), attachmentPutPageID, filePath)
		},
	}
	attachmentPutCmd.Flags().StringVar(&attachmentPutPageID, "page-id", "", "Confluence page id")
	_ = attachmentPutCmd.MarkFlagRequired("page-id")

	attachmentDeleteCmd := &cobra.Command{
		Use:   "delete <filename>",
		Args:  cobra.ExactArgs(1),
		Short: "Delete an attachment from a Confluence page by filename",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return a.runAttachmentDelete(cmd.Context(), attachmentDelPageID, cmdArgs[0])
		},
	}
	attachmentDeleteCmd.Flags().StringVar(&attachmentDelPageID, "page-id", "", "Confluence page id")
	_ = attachmentDeleteCmd.MarkFlagRequired("page-id")

	attachmentCmd.AddCommand(attachmentPutCmd, attachmentDeleteCmd)
	rootCmd.AddCommand(pageCmd, attachmentCmd)
	return rootCmd
}
