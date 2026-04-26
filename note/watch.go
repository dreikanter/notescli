package note

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	watchDebounce = 100 * time.Millisecond
	watchBufSize  = 64
)

// EventType describes the kind of note change a Watcher observed.
type EventType int

const (
	EventCreated EventType = iota + 1
	EventUpdated
	EventDeleted
)

// Event is an ID-keyed Store change notification.
type Event struct {
	Type EventType
	ID   int
}

// Watcher emits Store change events until its context is cancelled or Close is
// called.
type Watcher interface {
	Events() <-chan Event
	Close() error
}

// WatchOpt configures OSStore.Watch. No options are currently exposed; the
// parameter is reserved so options such as WithDebounce can be added without
// changing the Watch signature.
type WatchOpt func(*watchConfig)

type watchConfig struct{}

type osWatcher struct {
	events <-chan Event
	close  func() error
}

func (w *osWatcher) Events() <-chan Event { return w.events }
func (w *osWatcher) Close() error         { return w.close() }

// Watch returns a Watcher that emits ID-keyed events for note files under the
// store root. Events are going-forward only: Watch does not replay a snapshot.
// Long-running consumers should subscribe first, then call Store.All, so file
// changes that happen during the initial list are queued on the watcher.
//
// Implementations debounce internally. The current debounce window is fixed at
// 100 ms and coalesces by ID: create+delete in one window emits nothing,
// create+update emits create, and otherwise the last event in the window wins.
// Watch performs an initial filename scan before arming filesystem watches.
func (s *OSStore) Watch(ctx context.Context, opts ...WatchOpt) (Watcher, error) {
	cfg := watchConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	known, err := s.watchKnownIDs()
	if err != nil {
		return nil, err
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := addWatchDirs(fsw, s.root); err != nil {
		_ = fsw.Close()
		return nil, err
	}

	out := make(chan Event, watchBufSize)
	done := make(chan struct{})
	var once sync.Once
	closeFn := func() error {
		once.Do(func() { close(done) })
		return nil
	}

	go s.runWatch(ctx, fsw, known, out, done)

	return &osWatcher{events: out, close: closeFn}, nil
}

func (s *OSStore) runWatch(
	ctx context.Context,
	fsw *fsnotify.Watcher,
	known map[int]bool,
	out chan<- Event,
	done <-chan struct{},
) {
	defer close(out)
	defer fsw.Close()

	pending := make(map[int]Event)
	var order []int
	var timer *time.Timer
	var timerC <-chan time.Time

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
		timer = nil
		timerC = nil
	}
	armTimer := func() {
		if timer == nil {
			timer = time.NewTimer(watchDebounce)
			timerC = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(watchDebounce)
		timerC = timer.C
	}
	flush := func() bool {
		stopTimer()
		for _, id := range order {
			e, ok := pending[id]
			if !ok {
				continue
			}
			select {
			case out <- e:
			case <-ctx.Done():
				return false
			case <-done:
				return false
			}
		}
		pending = make(map[int]Event)
		order = nil
		return true
	}

	for {
		select {
		case <-ctx.Done():
			stopTimer()
			return
		case <-done:
			stopTimer()
			return
		case <-timerC:
			if !flush() {
				return
			}
		case err, ok := <-fsw.Errors:
			if !ok {
				stopTimer()
				return
			}
			// Watch has no error event in its initial API. Consumers that suspect
			// drift should recover by re-listing the store; a future WatchOpt can
			// surface backend errors without changing the Watch signature.
			_ = err
		case ev, ok := <-fsw.Events:
			if !ok {
				stopTimer()
				return
			}
			if ev.Op&fsnotify.Create != 0 && watchEventIsDir(ev.Name) {
				addCreatedDir(fsw, ev.Name)
				for _, e := range s.discoverWatchCreates(known) {
					order = coalesceWatchEvent(pending, order, e)
				}
			}
			e, ok := classifyWatchEvent(ev, known)
			if !ok {
				if len(pending) > 0 {
					armTimer()
				}
				continue
			}
			order = coalesceWatchEvent(pending, order, e)
			if len(pending) == 0 {
				stopTimer()
			} else {
				armTimer()
			}
		}
	}
}

func (s *OSStore) watchKnownIDs() (map[int]bool, error) {
	refs, err := s.scanFileRefs()
	if err != nil {
		return nil, err
	}
	known := make(map[int]bool, len(refs))
	for _, r := range refs {
		known[r.id] = true
	}
	return known, nil
}

func addWatchDirs(fsw *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		return fsw.Add(path)
	})
}

func watchEventIsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func addCreatedDir(fsw *fsnotify.Watcher, path string) {
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		_ = fsw.Add(p)
		return nil
	})
}

func (s *OSStore) discoverWatchCreates(known map[int]bool) []Event {
	refs, err := s.scanFileRefs()
	if err != nil {
		return nil
	}
	events := make([]Event, 0)
	for _, r := range refs {
		if known[r.id] {
			continue
		}
		known[r.id] = true
		events = append(events, Event{Type: EventCreated, ID: r.id})
	}
	return events
}

func classifyWatchEvent(ev fsnotify.Event, known map[int]bool) (Event, bool) {
	id, ok := watchEventID(ev.Name)
	if !ok {
		return Event{}, false
	}

	if ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		if !known[id] {
			return Event{}, false
		}
		delete(known, id)
		return Event{Type: EventDeleted, ID: id}, true
	}

	if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Chmod) == 0 {
		return Event{}, false
	}
	if info, err := os.Stat(ev.Name); err != nil || info.IsDir() {
		return Event{}, false
	}

	if known[id] {
		return Event{Type: EventUpdated, ID: id}, true
	}
	known[id] = true
	return Event{Type: EventCreated, ID: id}, true
}

func watchEventID(path string) (int, bool) {
	if filepath.Ext(path) != ".md" {
		return 0, false
	}
	base := strings.TrimSuffix(filepath.Base(path), ".md")
	ref, err := ParseFilename(base)
	if err != nil {
		return 0, false
	}
	id, err := strconv.Atoi(ref.ID)
	return id, err == nil
}

func coalesceWatchEvent(pending map[int]Event, order []int, next Event) []int {
	prev, ok := pending[next.ID]
	if !ok {
		pending[next.ID] = next
		return appendWatchOrder(order, next.ID)
	}

	if prev.Type == EventCreated && next.Type == EventUpdated {
		return order
	}
	if prev.Type == EventCreated && next.Type == EventDeleted {
		delete(pending, next.ID)
		return order
	}
	if prev.Type == EventDeleted && next.Type == EventCreated {
		pending[next.ID] = Event{Type: EventUpdated, ID: next.ID}
		return order
	}
	pending[next.ID] = next
	return order
}

func appendWatchOrder(order []int, id int) []int {
	for _, existing := range order {
		if existing == id {
			return order
		}
	}
	return append(order, id)
}
