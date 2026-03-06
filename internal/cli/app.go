package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/takymt/cflcli/internal/page"
)

const defaultWatchDebounce = 800 * time.Millisecond

// App runs the page subcommands against a Confluence client.
type App struct {
	client            page.Client
	clientLoader      func() (page.Client, error)
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

// NewLazy constructs a CLI app that resolves a client only when needed.
func NewLazy(clientLoader func() (page.Client, error), stdout io.Writer) *App {
	return &App{
		clientLoader:      clientLoader,
		stdout:            stdout,
		watchDebounce:     defaultWatchDebounce,
		watchPollInterval: 100 * time.Millisecond,
	}
}

// Run executes the requested page command.
func (a *App) Run(ctx context.Context, args []string, workdir string) int {
	rootCmd := a.newRootCommand(args, workdir)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		a.println(err.Error())
		return 1
	}

	return 0
}

func (a *App) ensureClient() error {
	if a.client != nil {
		return nil
	}
	if a.clientLoader == nil {
		return errors.New("client is not configured")
	}
	client, err := a.clientLoader()
	if err != nil {
		return err
	}
	a.client = client
	return nil
}

func (a *App) runPageNew(ctx context.Context, workdir string, fileArg string, spaceKey string, parentID string, watch bool) error {
	path := resolvePath(workdir, fileArg)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("target file %q already exists", fileArg)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	title := page.TitleFromPath(fileArg)
	if err := a.ensureClient(); err != nil {
		return err
	}
	spaceID, err := a.client.ResolveSpaceIDByKey(ctx, spaceKey)
	if err != nil {
		return fmt.Errorf("resolve space id: %w", err)
	}
	if parentID == "" {
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
		SpaceKey: spaceKey,
		PageID:   created.ID,
		ParentID: parentID,
	}, "")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	if !watch {
		a.printPageURL("Created page URL", created.URL)
		return nil
	}
	return a.watchFile(ctx, path, fileArg, created.URL, false)
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

	return a.watchFile(ctx, path, fileArg, "", true)
}

func (a *App) runAttachmentPut(ctx context.Context, pageID string, filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		return err
	}
	if err := a.ensureClient(); err != nil {
		return err
	}
	return a.client.PutAttachment(ctx, pageID, filePath)
}

func (a *App) runAttachmentDelete(ctx context.Context, pageID string, filename string) error {
	if err := a.ensureClient(); err != nil {
		return err
	}
	return a.client.DeleteAttachment(ctx, pageID, filename)
}

func (a *App) watchFile(ctx context.Context, path string, displayPath string, initialURL string, syncFirst bool) error {
	if syncFirst {
		first, err := a.syncFile(ctx, path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return err
			}
			a.println(colorRed("!") + " " + err.Error())
		} else {
			a.printPageURL("Synced page URL", first.URL)
		}
	} else if initialURL != "" {
		a.printPageURL("Created page URL", initialURL)
	}
	a.println("Watching: " + displayPath + " (press q to quit)")

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
	a.println("Stopped watching: " + displayPath)
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
			updated, err := a.syncFile(ctx, path)
			if err != nil {
				a.println(colorRed("!") + " " + err.Error())
			} else {
				a.println(colorGreen(".") + " Synced page URL: " + updated.URL)
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
	if err := a.ensureClient(); err != nil {
		return page.Page{}, err
	}

	converted, generatedPaths, err := page.ConvertMarkdownToStorageWithMermaid(ctx, path, body, a.client.SiteBaseURL())
	if err != nil {
		return page.Page{}, err
	}
	defer func() {
		for _, generatedPath := range generatedPaths {
			_ = os.Remove(generatedPath)
		}
	}()

	if err := page.SyncAttachmentsFromStorage(ctx, a.client, frontmatter.PageID, path, converted); err != nil {
		return page.Page{}, err
	}

	title := page.TitleFromPath(path)
	updated, err := a.client.UpdatePage(ctx, frontmatter.PageID, title, converted)
	if err != nil {
		return page.Page{}, err
	}
	return updated, nil
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
