package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	if len(args) < 2 || args[0] != "page" {
		a.println("usage: cfl page <new|sync> ...")
		return 1
	}

	switch args[1] {
	case "new":
		if err := a.runPageNew(ctx, args[2:], workdir); err != nil {
			a.println(err.Error())
			return 1
		}
		return 0
	case "sync":
		if err := a.runPageSync(ctx, args[2:], workdir); err != nil {
			a.println(err.Error())
			return 1
		}
		return 0
	default:
		a.println("usage: cfl page <new|sync> ...")
		return 1
	}
}

func (a *App) runPageNew(ctx context.Context, args []string, workdir string) error {
	fileArg, options, err := splitArgs(args)
	if err != nil {
		return err
	}
	if fileArg == "" {
		return errors.New("page new requires exactly one markdown file argument")
	}

	spaceID := options["space-id"]
	parentID := options["parent-id"]
	if spaceID == "" {
		return errors.New("--space-id is required")
	}

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

	a.println(created.URL)
	return nil
}

func (a *App) runPageSync(ctx context.Context, args []string, workdir string) error {
	fileArg, options, err := splitArgs(args)
	if err != nil {
		return err
	}
	if fileArg == "" {
		return errors.New("page sync requires exactly one markdown file argument")
	}
	watch := options["watch"] == "true"

	path := resolvePath(workdir, fileArg)
	if !watch {
		updated, err := a.syncFile(ctx, path)
		if err != nil {
			return err
		}
		a.println(updated.URL)
		return nil
	}

	first, err := a.syncFile(ctx, path)
	if err != nil {
		a.println(colorRed("!") + " " + err.Error())
	} else {
		a.println(first.URL)
	}

	watcher, err := newPollingWatcher(path, a.watchPollInterval)
	if err != nil {
		return err
	}
	defer func() {
		_ = watcher.Close()
	}()

	return a.watchLoop(ctx, watcher.Events(), path)
}

func (a *App) watchLoop(ctx context.Context, events <-chan struct{}, path string) error {
	var (
		timer  *time.Timer
		timerC <-chan time.Time
	)

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
				a.print(colorGreen("."))
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
	updated, err := a.client.UpdatePage(ctx, frontmatter.PageID, title, converted)
	if err != nil {
		return page.Page{}, err
	}

	return updated, nil
}

func (a *App) println(s string) {
	_, _ = fmt.Fprintln(a.stdout, s)
}

func (a *App) print(s string) {
	_, _ = fmt.Fprint(a.stdout, s)
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

func splitArgs(args []string) (string, map[string]string, error) {
	options := make(map[string]string)
	var fileArg string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			name := strings.TrimPrefix(arg, "--")
			if name == "watch" {
				options[name] = "true"
				continue
			}
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("missing value for --%s", name)
			}
			options[name] = args[i+1]
			i++
			continue
		}
		if fileArg != "" {
			return "", nil, errors.New("expected exactly one markdown file argument")
		}
		fileArg = arg
	}

	return fileArg, options, nil
}
