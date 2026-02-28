// Package watcher monitors file system changes for watch mode operation.
package watcher

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DefaultDebounce is the default debounce duration.
const DefaultDebounce = 300 * time.Millisecond

// OnChange is the callback type invoked when a watched file changes.
type OnChange func(changedFile string)

// Watcher monitors specific files for write changes with debounce support.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	debounce  time.Duration
	onChange  OnChange
	done      chan struct{}
	syncing   bool
	mu        sync.Mutex
}

// New creates a Watcher that monitors the given files. The onChange callback
// is invoked after the debounce duration elapses following a write event.
func New(files []string, debounce time.Duration, onChange OnChange) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if err := fsw.Add(f); err != nil {
			_ = fsw.Close()
			return nil, err
		}
	}

	return &Watcher{
		fsWatcher: fsw,
		debounce:  debounce,
		onChange:  onChange,
		done:      make(chan struct{}),
	}, nil
}

// Start begins listening for file change events in a background goroutine.
func (w *Watcher) Start() error {
	go w.loop()
	return nil
}

// Stop signals the watcher to shut down and closes the underlying fsnotify watcher.
func (w *Watcher) Stop() {
	close(w.done)
	_ = w.fsWatcher.Close()
}

// SetSyncing sets the syncing flag. While true, file change events are ignored
// to prevent re-triggering from the watcher's own writes.
func (w *Watcher) SetSyncing(v bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.syncing = v
}

func (w *Watcher) isSyncing() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.syncing
}

func (w *Watcher) loop() {
	var timer *time.Timer
	var lastFile string

	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			if !event.Has(fsnotify.Write) {
				continue
			}
			if w.isSyncing() {
				continue
			}

			lastFile = event.Name

			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(w.debounce, func() {
				if !w.isSyncing() {
					w.onChange(lastFile)
				}
			})

		case _, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
		}
	}
}
