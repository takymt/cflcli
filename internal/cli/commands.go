package cli

import "github.com/spf13/cobra"

func (a *App) newRootCommand(args []string, workdir string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "cfl",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return newUsageError(cmd, err)
	})
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
	authCmd := &cobra.Command{
		Use:   "auth",
		Args:  noArgsWithUsage(),
		Short: "Manage Confluence authentication",
	}

	var (
		newSpaceKey string
		newParentID string
		newWatch    bool
	)
	pageNewCmd := &cobra.Command{
		Use:     "new <title>.md",
		Args:    exactArgsWithUsage(1),
		PreRunE: requireFlagsWithUsage("space-key"),
		Short:   "Create a local markdown file and a Confluence page",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return a.runPageNew(cmd.Context(), workdir, cmdArgs[0], newSpaceKey, newParentID, newWatch)
		},
	}
	pageNewCmd.Flags().StringVar(&newSpaceKey, "space-key", "", "Confluence space key")
	pageNewCmd.Flags().StringVar(&newParentID, "parent-id", "", "Confluence parent page id")
	pageNewCmd.Flags().BoolVar(&newWatch, "watch", false, "Watch the created file and sync on changes")

	var syncWatch bool
	pageSyncCmd := &cobra.Command{
		Use:   "sync <path>",
		Args:  exactArgsWithUsage(1),
		Short: "Sync a local markdown file or directory to Confluence",
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
		Use:     "put <file>",
		Args:    exactArgsWithUsage(1),
		PreRunE: requireFlagsWithUsage("page-id"),
		Short:   "Upload or update an attachment on a Confluence page",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			filePath := resolvePath(workdir, cmdArgs[0])
			return a.runAttachmentPut(cmd.Context(), attachmentPutPageID, filePath)
		},
	}
	attachmentPutCmd.Flags().StringVar(&attachmentPutPageID, "page-id", "", "Confluence page id")

	attachmentDeleteCmd := &cobra.Command{
		Use:     "delete <filename>",
		Args:    exactArgsWithUsage(1),
		PreRunE: requireFlagsWithUsage("page-id"),
		Short:   "Delete an attachment from a Confluence page by filename",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return a.runAttachmentDelete(cmd.Context(), attachmentDelPageID, cmdArgs[0])
		},
	}
	attachmentDeleteCmd.Flags().StringVar(&attachmentDelPageID, "page-id", "", "Confluence page id")

	attachmentCmd.AddCommand(attachmentPutCmd, attachmentDeleteCmd)

	var authOpts authLoginOptions
	authCmd.RunE = func(cmd *cobra.Command, cmdArgs []string) error {
		return a.runAuthLogin(cmd.Context(), authOpts.domain, authOpts.email, authOpts.apiToken, authOpts.noValidate)
	}
	bindAuthLoginFlags(authCmd, &authOpts)

	var authLoginOpts authLoginOptions
	authLoginCmd := &cobra.Command{
		Use:   "login",
		Args:  noArgsWithUsage(),
		Short: "Save Confluence credentials",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return a.runAuthLogin(cmd.Context(), authLoginOpts.domain, authLoginOpts.email, authLoginOpts.apiToken, authLoginOpts.noValidate)
		},
	}
	bindAuthLoginFlags(authLoginCmd, &authLoginOpts)

	authLogoutCmd := &cobra.Command{
		Use:   "logout",
		Args:  noArgsWithUsage(),
		Short: "Clear saved Confluence credentials",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return a.runAuthLogout()
		},
	}

	authCmd.AddCommand(authLoginCmd, authLogoutCmd)
	rootCmd.AddCommand(pageCmd, attachmentCmd, authCmd)
	return rootCmd
}

type authLoginOptions struct {
	domain     string
	email      string
	apiToken   string
	noValidate bool
}

func bindAuthLoginFlags(cmd *cobra.Command, opts *authLoginOptions) {
	cmd.Flags().StringVar(&opts.domain, "domain", "", "Confluence domain")
	cmd.Flags().StringVar(&opts.email, "email", "", "Confluence email")
	cmd.Flags().StringVar(&opts.apiToken, "api-token", "", "Confluence API token")
	cmd.Flags().BoolVar(&opts.noValidate, "no-validate", false, "Skip online credential validation before saving")
}
