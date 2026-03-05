package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/page"
)

const defaultWatchDebounce = 800 * time.Millisecond

// App runs the page subcommands against a Confluence client.
type App struct {
	client            page.Client
	stdout            io.Writer
	watchDebounce     time.Duration
	watchPollInterval time.Duration
}

// New constructs a CLI app with the provided output writer.
func New(client page.Client, stdout io.Writer) *App {
	return &App{
		client:            client,
		stdout:            stdout,
		watchDebounce:     defaultWatchDebounce,
		watchPollInterval: 100 * time.Millisecond,
	}
}

// Run executes the requested page command.
func (a *App) Run(ctx context.Context, args []string, workdir string) int {
	rootCmd := &cobra.Command{
		Use:           "cfl",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.SetOut(a.stdout)
	rootCmd.SetErr(a.stdout)
	rootCmd.SetArgs(args)

	pageCmd := &cobra.Command{
		Use: "page",
	}

	var (
		newSpaceID  string
		newParentID string
		newWatch    bool
	)
	pageNewCmd := &cobra.Command{
		Use:   "new <title>.md",
		Args:  cobra.ExactArgs(1),
		Short: "Create a local markdown file and a Confluence page",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return a.runPageNew(cmd.Context(), workdir, cmdArgs[0], newSpaceID, newParentID, newWatch)
		},
	}
	pageNewCmd.Flags().StringVar(&newSpaceID, "space-id", "", "Confluence space id")
	pageNewCmd.Flags().StringVar(&newParentID, "parent-id", "", "Confluence parent page id")
	pageNewCmd.Flags().BoolVar(&newWatch, "watch", false, "Watch the created file and sync on changes")
	_ = pageNewCmd.MarkFlagRequired("space-id")

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
	rootCmd.AddCommand(pageCmd)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		a.println(err.Error())
		return 1
	}

	return 0
}

func (a *App) runPageNew(ctx context.Context, workdir string, fileArg string, spaceID string, parentID string, watch bool) error {
	path := resolvePath(workdir, fileArg)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("target file %q already exists", fileArg)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	title := page.TitleFromPath(fileArg)
	if parentID == "" {
		var err error
		parentID, err = a.client.ResolveSpaceRootPage(ctx, spaceID)
		if err != nil {
			return fmt.Errorf("resolve root parent: %w", err)
		}
	}

	exists, err := a.client.PageExists(ctx, spaceID, parentID, title)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("page %q already exists under parent %s", title, parentID)
	}

	created, err := a.client.CreatePage(ctx, spaceID, parentID, title, "")
	if err != nil {
		return err
	}

	data := page.FormatMarkdownFile(page.Frontmatter{
		SpaceID:  spaceID,
		PageID:   created.ID,
		ParentID: parentID,
	}, "")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	a.printPageURL("Created page URL", created.URL)
	if watch {
		return a.runPageSync(ctx, workdir, fileArg, true)
	}

	return nil
}

func (a *App) runPageSync(ctx context.Context, workdir string, fileArg string, watch bool) error {
	path := resolvePath(workdir, fileArg)
	if !watch {
		updated, err := a.syncFile(ctx, path)
		if err != nil {
			return err
		}
		a.printPageURL("Synced page URL", updated.URL)
		return nil
	}

	first, err := a.syncFile(ctx, path)
	if err != nil {
		a.println(colorRed("!") + " " + err.Error())
	} else {
		a.printPageURL("Synced page URL", first.URL)
	}

	watcher, err := newPollingWatcher(path, a.watchPollInterval)
	if err != nil {
		return err
	}
	watchErr := a.watchLoop(ctx, watcher.Events(), path)
	closeErr := watcher.Close()
	if watchErr != nil {
		return watchErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func (a *App) watchLoop(ctx context.Context, events <-chan struct{}, path string) error {
	var (
		timer  *time.Timer
		timerC <-chan time.Time
	)
	quitCh := startWatchQuitListener()

	stopTimer := func() {
		if timer == nil {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			stopTimer()
			return nil
		case <-quitCh:
			stopTimer()
			return nil
		case _, ok := <-events:
			if !ok {
				stopTimer()
				return nil
			}
			if timer == nil {
				timer = time.NewTimer(a.watchDebounce)
			} else {
				stopTimer()
				timer.Reset(a.watchDebounce)
			}
			timerC = timer.C
		case <-timerC:
			_, err := a.syncFile(ctx, path)
			if err != nil {
				a.println(colorRed("!") + " " + err.Error())
			} else {
				a.println(colorGreen("."))
			}
			timerC = nil
		}
	}
}

func (a *App) syncFile(ctx context.Context, path string) (page.Page, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return page.Page{}, err
	}

	frontmatter, body, err := page.ParseMarkdownFile(data)
	if err != nil {
		return page.Page{}, err
	}

	converted, err := page.ConvertMarkdownToStorage(body)
	if err != nil {
		return page.Page{}, err
	}

	title := page.TitleFromPath(path)
	return a.client.UpdatePage(ctx, frontmatter.PageID, title, converted)
}

func (a *App) println(s string) {
	_, _ = fmt.Fprintln(a.stdout, s)
}

func (a *App) printPageURL(prefix string, pageURL string) {
	a.println(prefix + ": " + pageURL)
}

func resolvePath(workdir string, file string) string {
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(workdir, file)
}

func colorGreen(s string) string {
	return "\x1b[32m" + s + "\x1b[0m"
}

func colorRed(s string) string {
	return "\x1b[31m" + s + "\x1b[0m"
}

func startWatchQuitListener() <-chan struct{} {
	ch := make(chan struct{}, 1)

	info, err := os.Stdin.Stat()
	if err != nil || (info.Mode()&os.ModeCharDevice) == 0 {
		return ch
	}

	go func() {
		buf := make([]byte, 1)
		for {
			n, readErr := os.Stdin.Read(buf)
			if readErr != nil {
				return
			}
			if n == 1 && (buf[0] == 'q' || buf[0] == 'Q') {
				ch <- struct{}{}
				return
			}
		}
	}()

	return ch
}
