// Package watch provides a filesystem watcher primitive for notes stores.
//
// A Watcher monitors root for .md note activity and emits a single debounced
// signal after quiescence. It is intended to pair with note.Index.Reload: when
// the watcher fires, the consumer calls Reload and the index coalescer
// collapses bursts into at most one rebuild.
//
// The package is split out from the note core so the CLI binary does not pull
// fsnotify into its dependency graph.
package watch

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dreikanter/notes-cli/note"
	"github.com/fsnotify/fsnotify"
)

// DefaultDebounce is the quiet period used when WithDebounce is not set.
const DefaultDebounce = 100 * time.Millisecond

// Option configures a Watcher.
type Option func(*config)

type config struct {
	scan     note.ScanOptions
	debounce time.Duration
}

// WithScanOptions mirrors note.ScanOptions onto the watcher. In strict mode
// (the default), events whose path is not a YYYY/MM/*.md file under root are
// ignored, matching note.Scan's strict discipline. Lenient mode accepts any
// *.md file anywhere under root.
func WithScanOptions(o note.ScanOptions) Option {
	return func(c *config) { c.scan = o }
}

// WithDebounce sets the quiet period the watcher waits after the last
// relevant event before emitting a signal. Defaults to DefaultDebounce.
func WithDebounce(d time.Duration) Option {
	return func(c *config) { c.debounce = d }
}

// Watcher observes .md note activity under a store root and delivers a single
// signal on Events() after filesystem activity settles.
//
// The watcher adds directories to the underlying fsnotify recursively: strict
// mode subscribes only to root, year directories (digits-only names), and
// two-digit month directories beneath them; lenient mode subscribes to every
// subdirectory. New directories created while the watcher is running are
// registered as they appear.
type Watcher struct {
	root     string
	strict   bool
	debounce time.Duration

	fsw    *fsnotify.Watcher
	events chan struct{}
	errs   chan error
	done   chan struct{}

	closeOnce sync.Once
	closeErr  error
}

// New creates a Watcher rooted at root. The watcher starts observing
// immediately; call Close to release resources.
func New(root string, opts ...Option) (*Watcher, error) {
	cfg := config{
		scan:     note.ScanOptions{Strict: true},
		debounce: DefaultDebounce,
	}
	for _, o := range opts {
		o(&cfg)
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("watch: root %q is not a directory", root)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		root:     root,
		strict:   cfg.scan.Strict,
		debounce: cfg.debounce,
		fsw:      fsw,
		events:   make(chan struct{}, 1),
		errs:     make(chan error, 1),
		done:     make(chan struct{}),
	}

	if err := w.addTree(root); err != nil {
		_ = fsw.Close()
		return nil, err
	}

	go w.run()
	return w, nil
}

// Events returns a channel that receives a single struct{} after each quiet
// period following relevant activity. The channel is closed when the watcher
// is closed.
func (w *Watcher) Events() <-chan struct{} { return w.events }

// Errors returns a channel that receives non-fatal errors from the underlying
// fsnotify watcher. Errors are dropped when the consumer is not ready; treat
// this channel as best-effort diagnostics.
func (w *Watcher) Errors() <-chan error { return w.errs }

// Close stops the watcher and releases resources. Safe to call multiple times;
// only the first invocation returns the underlying fsnotify close error.
func (w *Watcher) Close() error {
	w.closeOnce.Do(func() {
		close(w.done)
		w.closeErr = w.fsw.Close()
	})
	return w.closeErr
}

// addTree registers dir and every relevant descendant directory with fsnotify.
// Errors from individual fsnotify.Add calls for subdirectories are swallowed
// (likely racy removals or permission glitches); failure to add dir itself is
// returned.
func (w *Watcher) addTree(dir string) error {
	if err := w.fsw.Add(dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == dir {
				return err
			}
			return nil
		}
		if !d.IsDir() || path == dir {
			return nil
		}
		if w.shouldWatchDir(path) {
			_ = w.fsw.Add(path)
			return nil
		}
		if w.strict {
			// shouldWatchDir returning false in strict mode means this path
			// doesn't conform to YYYY/MM at its depth; the strict layout is
			// fixed-depth, so there's nowhere deeper worth descending to.
			return fs.SkipDir
		}
		return nil
	})
}

// shouldWatchDir reports whether a directory should be registered with
// fsnotify. Root is always watched. In lenient mode every subdirectory is
// watched. In strict mode only year (digits-only) and two-digit month
// directories beneath year dirs are watched.
func (w *Watcher) shouldWatchDir(path string) bool {
	rel, err := filepath.Rel(w.root, path)
	if err != nil || rel == "." {
		return true
	}
	if !w.strict {
		return true
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	switch len(parts) {
	case 1:
		return note.IsID(parts[0])
	case 2:
		return note.IsID(parts[0]) && len(parts[1]) == 2 && note.IsID(parts[1])
	default:
		return false
	}
}

// strictNotePath reports whether path points at a YYYY/MM/*.md file under root.
func (w *Watcher) strictNotePath(path string) bool {
	rel, err := filepath.Rel(w.root, path)
	if err != nil {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 3 {
		return false
	}
	if !note.IsID(parts[0]) {
		return false
	}
	if len(parts[1]) != 2 || !note.IsID(parts[1]) {
		return false
	}
	return true
}

// run reads fsnotify events and drives the debounce timer. It exits when
// done is closed or the fsnotify channels are closed.
func (w *Watcher) run() {
	defer close(w.events)

	var (
		timer  *time.Timer
		timerC <-chan time.Time
	)
	reset := func() {
		if timer == nil {
			timer = time.NewTimer(w.debounce)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(w.debounce)
		}
		timerC = timer.C
	}

	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			select {
			case w.errs <- err:
			default:
			}
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if w.handle(ev) {
				reset()
			}
		case <-timerC:
			timerC = nil
			select {
			case w.events <- struct{}{}:
			default:
			}
		}
	}
}

// handle updates watch registrations in response to directory creations and
// reports whether ev is a relevant .md event that should (re)start the
// debounce timer.
func (w *Watcher) handle(ev fsnotify.Event) bool {
	if ev.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
			_ = w.addTree(ev.Name)
		}
	}
	if filepath.Ext(ev.Name) != ".md" {
		return false
	}
	if w.strict && !w.strictNotePath(ev.Name) {
		return false
	}
	return true
}
