package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/takymt/cflcli/internal/page"
)

type collectResult struct {
	Path    string
	Display string
}

type skipResult struct {
	Display string
	Reason  string
}

type syncResult struct {
	Display string
	URL     string
	Err     error
}

const (
	defaultDirSyncMaxFiles    = 500
	defaultDirSyncConcurrency = 5
)

var errMarkdownFileLimitExceeded = errors.New("markdown file limit exceeded")

func collectMarkdownFiles(dir string, maxFiles int) ([]collectResult, []skipResult, error) {
	var (
		files   []collectResult
		skipped []skipResult
		seenMD  int
	)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == dir {
				return walkErr
			}
			skipped = append(skipped, skipResult{
				Display: displayPath(dir, path),
				Reason:  walkErr.Error(),
			})
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(path)
			if err == nil && info.IsDir() {
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}

		seenMD++
		if maxFiles > 0 && seenMD > maxFiles {
			return errMarkdownFileLimitExceeded
		}

		if data, readErr := os.ReadFile(path); readErr != nil {
			skipped = append(skipped, skipResult{
				Display: displayPath(dir, path),
				Reason:  readErr.Error(),
			})
		} else if _, _, parseErr := page.ParseMarkdownFile(data); parseErr != nil {
			skipped = append(skipped, skipResult{
				Display: displayPath(dir, path),
				Reason:  "no valid frontmatter",
			})
		} else {
			files = append(files, collectResult{
				Path:    path,
				Display: displayPath(dir, path),
			})
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errMarkdownFileLimitExceeded) {
			return nil, nil, fmt.Errorf("directory contains more than %d markdown files; use a more specific path", maxFiles)
		}
		return nil, nil, err
	}

	return files, skipped, nil
}

func (a *App) syncDir(ctx context.Context, files []collectResult, concurrency int) []syncResult {
	if concurrency < 1 {
		concurrency = 1
	}

	results := make([]syncResult, len(files))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, file := range files {
		if err := ctx.Err(); err != nil {
			for j := i; j < len(files); j++ {
				results[j] = syncResult{
					Display: files[j].Display,
					Err:     err,
				}
			}
			break
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(i int, file collectResult) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			updated, err := a.syncFile(ctx, file.Path)
			results[i] = syncResult{
				Display: file.Display,
				Err:     err,
			}
			if err == nil {
				results[i].URL = updated.URL
			}
		}(i, file)
	}

	wg.Wait()
	return results
}

func (a *App) printSyncSummary(results []syncResult, skipped []skipResult) int {
	var (
		synced  int
		failed  int
		printed bool
	)

	for _, result := range results {
		if result.Err != nil {
			a.println(fmt.Sprintf("Failed: %s: %v", result.Display, result.Err))
			failed++
		} else {
			a.println(fmt.Sprintf("Synced: %s -> %s", result.Display, result.URL))
			synced++
		}
		printed = true
	}

	if a.printSkipped(skipped) {
		printed = true
	}
	if printed {
		a.println("")
	}

	summary := fmt.Sprintf("Synced %d/%d files", synced, len(results))
	if len(skipped) > 0 {
		summary += fmt.Sprintf(" (%d skipped)", len(skipped))
	}
	a.println(summary)
	return failed
}

func (a *App) printSkipped(skipped []skipResult) bool {
	if len(skipped) == 0 {
		return false
	}
	for _, skip := range skipped {
		a.println(fmt.Sprintf("Skipped (%s): %s", skip.Reason, skip.Display))
	}
	return true
}

func displayPath(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return filepath.Base(path)
	}

	base := filepath.Base(filepath.Clean(root))
	switch base {
	case "", ".", string(filepath.Separator):
		return filepath.Clean(rel)
	default:
		return filepath.Join(base, rel)
	}
}
