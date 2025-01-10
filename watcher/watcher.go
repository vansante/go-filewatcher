package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const deduplicationTime = time.Millisecond * 500

type Watcher struct {
	watcher    *fsnotify.Watcher
	paths      map[string]struct{}
	extensions []string
	prepCmd    string
	runCmd     string
	runCancel  context.CancelFunc
	changes    chan string
	ctx        context.Context
}

func New(ctx context.Context, prepCmd, runCmd string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("error creating watcher: %w", err)
	}

	return &Watcher{
		watcher: w,
		paths:   make(map[string]struct{}),
		prepCmd: prepCmd,
		runCmd:  runCmd,
		changes: make(chan string, 1024),
		ctx:     ctx,
	}, nil
}

func (w *Watcher) SetExtensions(extensions []string) {
	w.extensions = extensions
}

func (w *Watcher) AddPath(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error getting path info: %w", err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("error getting absolute path for %s: %w", path, err)
	}
	path = abs

	_, ok := w.paths[path]
	if ok {
		// Skip duplicates
		return nil
	}
	w.paths[path] = struct{}{}

	err = w.watcher.Add(path)
	if err != nil {
		return fmt.Errorf("error adding watch entry for %s: %w", path, err)
	}

	err = w.addRecursive(path, fi)
	if err != nil {
		return err
	}
	return nil
}

func (w *Watcher) addRecursive(path string, fi os.FileInfo) error {
	if !fi.IsDir() {
		return nil // we are done
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("error getting absolute path for %s: %w", path, err)
		}
		path = abs

		// Skip duplicates
		_, ok := w.paths[path]
		if ok {
			return nil
		}
		w.paths[path] = struct{}{}

		// Skip hidden directories
		if w.isHiddenDirectory(path) {
			return filepath.SkipDir
		}

		err = w.watcher.Add(path)
		if err != nil {
			return fmt.Errorf("error adding watch entry for %s: %w", path, err)
		}
		return nil
	})
	return err
}

func (w *Watcher) isHiddenDirectory(path string) bool {
	if strings.HasPrefix(filepath.Base(path), ".") {
		return true
	}

	dirs := filepath.SplitList(path)
	for _, dir := range dirs {
		if strings.HasPrefix(dir, ".") {
			return true
		}
	}
	return false
}

func (w *Watcher) shouldHandleEvent(evt fsnotify.Event) bool {
	if w.isHiddenDirectory(evt.Name) {
		return false
	}

	ext := filepath.Ext(evt.Name)
	if len(w.extensions) > 0 && !slices.Contains(w.extensions, ext) {
		return false
	}

	return true
}

func (w *Watcher) Watch() {
	lastChange := time.Now()

	go w.handleEvents()

	for {
		select {
		case <-w.ctx.Done():
			return
		case evt := <-w.watcher.Events:
			// Treat multiple events at same time as one
			if time.Since(lastChange) < deduplicationTime {
				continue
			}

			if !w.shouldHandleEvent(evt) {
				continue
			}

			lastChange = time.Now()

			w.changes <- evt.Name
		}
	}
}

func (w *Watcher) handleEvents() {
	for path := range w.changes {
		_, _ = fmt.Fprintf(os.Stderr, "--- Update: %s\n", path)
		w.runChangeCommand()
	}

	w.runCancel()
}

func (w *Watcher) runChangeCommand() {
	// Create child context so we can cancel this command
	// without cancelling the entire program
	commandCtx, commandCancel := context.WithCancel(w.ctx)
	defer commandCancel()

	// Cancel and rerun the command if the file changes
	// while we run the command
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		name, ok := <-w.changes
		// The channel was closed, shut down
		if !ok {
			return
		}

		commandCancel()
		// Send the file change back on the channel
		// to trigger `runChangeCommand` again
		w.changes <- name
	}()

	if w.prepCmd != "" {
		RunCommand(commandCtx, w.prepCmd, true)
	}

	if commandCtx.Err() != nil {
		return
	}

	if w.runCancel != nil {
		// Kill existing process
		w.runCancel()
	}

	var runCtx context.Context
	runCtx, w.runCancel = context.WithCancel(w.ctx)
	RunCommand(runCtx, w.runCmd, false)

	wg.Wait()
}
