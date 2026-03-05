package cli

import (
	"os"
	"time"
)

type pollingWatcher struct {
	events chan struct{}
	done   chan struct{}
}

func newPollingWatcher(path string, interval time.Duration) (*pollingWatcher, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	w := &pollingWatcher{
		events: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		lastModTime := info.ModTime()
		lastSize := info.Size()

		for {
			select {
			case <-w.done:
				close(w.events)
				return
			case <-ticker.C:
				current, err := os.Stat(path)
				if err != nil {
					continue
				}
				if current.ModTime() != lastModTime || current.Size() != lastSize {
					lastModTime = current.ModTime()
					lastSize = current.Size()
					select {
					case w.events <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	return w, nil
}

func (w *pollingWatcher) Events() <-chan struct{} {
	return w.events
}

func (w *pollingWatcher) Close() error {
	close(w.done)
	return nil
}
