package note

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Entry is a fully-hydrated note record: the filename-derived Ref plus
// frontmatter and file stat metadata. It is the unit Index exposes to
// downstream consumers (notes-view, notes-pub) that previously maintained
// their own in-memory indexes.
type Entry struct {
	Ref
	Frontmatter Frontmatter
	ModTime     time.Time
	Size        int64

	// bodyHashtags holds the lowercased, deduplicated body hashtags extracted
	// during Load. Read via MergedTags; the field is unexported because it
	// only feeds migration shims (FilterByTags, ExtractTags). Nil when Load
	// ran with WithFrontmatter(false).
	bodyHashtags []string
}

// MergedTags returns the lowercased, deduplicated union of the entry's
// frontmatter tags and body hashtags. This matches the tag source used by
// FilterByTags and ExtractTags: both frontmatter `tags:` values and in-body
// `#hashtag` tokens. Result is sorted.
func (e Entry) MergedTags() []string {
	set := make(map[string]struct{}, len(e.Frontmatter.Tags)+len(e.bodyHashtags))
	for _, t := range e.Frontmatter.Tags {
		if t == "" {
			continue
		}
		set[strings.ToLower(t)] = struct{}{}
	}
	for _, t := range e.bodyHashtags {
		set[t] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// Index is an in-memory, read-only snapshot of a notes store. Build one with
// Load; Reload swaps state atomically under the RWMutex, so all read methods
// take RLock today.
type Index struct {
	root string
	cfg  loadConfig

	mu      sync.RWMutex
	entries []Entry
	byID    map[string]Entry
	byRel   map[string]Entry
	bySlug  map[string][]Entry
	byTag   map[string][]Entry
	allTags []string

	// buildMu guards curDone and queuedDone — the Reload state machine.
	// Separate from mu so rebuild bookkeeping does not contend with read
	// lookups. See Reload for the scheduling semantics.
	buildMu    sync.Mutex
	curDone    chan struct{} // in-flight build's completion signal; nil when idle
	queuedDone chan struct{} // follow-up build queued while curDone runs; nil when none
}

type loadConfig struct {
	frontmatter bool
	workers     int
	scanOpts    ScanOptions
	logger      Logger
}

// LoadOption configures Load. All options are optional; pass zero or more.
type LoadOption func(*loadConfig)

// WithFrontmatter controls whether Load parses YAML frontmatter for each
// note. Default true. Setting false skips the file read entirely — Size and
// ModTime still populate via stat, but Frontmatter is zero and tag indexes
// are empty.
func WithFrontmatter(b bool) LoadOption {
	return func(c *loadConfig) { c.frontmatter = b }
}

// WithWorkers sets the number of concurrent file-parsing workers. Default
// runtime.NumCPU(). Values <=0 fall back to the default; the effective worker
// count is capped at the number of notes.
func WithWorkers(n int) LoadOption {
	return func(c *loadConfig) { c.workers = n }
}

// WithScanOptions forwards directory-traversal options to the underlying
// Scan. Default is the zero value (Strict: true), matching Scan(root).
func WithScanOptions(o ScanOptions) LoadOption {
	return func(c *loadConfig) { c.scanOpts = o }
}

// WithLogger installs a Logger that receives non-fatal errors from Load, the
// underlying Scan, and subsequent Index.Reload runs — per-note frontmatter
// parse failures, unreadable subdirectories, and reload-build errors. Default:
// no-op (the package does not write to os.Stderr; wire a logger at the
// application edge if you want that).
func WithLogger(l Logger) LoadOption {
	return func(c *loadConfig) { c.logger = l }
}

// Load walks root once, parses frontmatter concurrently, and returns a
// populated Index. A single concurrent pass replaces the Scan → FilterByTags
// → ExtractTags re-read chain that duplicated I/O for each query.
//
// Per-note frontmatter parse errors are forwarded to the logger installed via
// WithLogger (no-op by default) and leave that entry's Frontmatter zero; they
// never abort the load. Any file-read or stat error aborts the load.
func Load(root string, opts ...LoadOption) (*Index, error) {
	cfg := loadConfig{
		frontmatter: true,
		workers:     runtime.NumCPU(),
		scanOpts:    ScanOptions{Strict: true},
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.workers <= 0 {
		cfg.workers = runtime.NumCPU()
	}

	idx := &Index{root: root, cfg: cfg}
	if err := idx.build(); err != nil {
		return nil, err
	}
	return idx, nil
}

// build walks the notes tree once, reads each entry under the configured
// worker pool, and atomically swaps the new state in under i.mu. Called by
// Load for the initial population and by runBuild for subsequent reloads.
func (i *Index) build() error {
	notes, err := Scan(i.root, WithStrict(i.cfg.scanOpts.Strict), WithScanLogger(i.cfg.logger))
	if err != nil {
		return err
	}

	entries := make([]Entry, len(notes))
	for j, n := range notes {
		entries[j] = Entry{Ref: n}
	}

	if len(entries) > 0 {
		workers := i.cfg.workers
		if workers > len(entries) {
			workers = len(entries)
		}

		g, ctx := errgroup.WithContext(context.Background())
		jobs := make(chan int)

		g.Go(func() error {
			defer close(jobs)
			for j := range entries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case jobs <- j:
				}
			}
			return nil
		})

		for w := 0; w < workers; w++ {
			g.Go(func() error {
				for j := range jobs {
					path := filepath.Join(i.root, entries[j].RelPath)
					info, err := os.Stat(path)
					if err != nil {
						return err
					}
					entries[j].ModTime = info.ModTime()
					entries[j].Size = info.Size()
					if i.cfg.frontmatter {
						data, err := os.ReadFile(path)
						if err != nil {
							return err
						}
						fm, body, parseErr := ParseNote(data)
						if parseErr != nil {
							i.cfg.logger.log(fmt.Errorf("%s: %w", path, parseErr))
							body = data
						} else {
							entries[j].Frontmatter = fm
						}
						entries[j].bodyHashtags = normalizeHashtags(ExtractHashtags(body))
					}
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}
	}

	byID := make(map[string]Entry, len(entries))
	byRel := make(map[string]Entry, len(entries))
	bySlug := make(map[string][]Entry)
	byTag := make(map[string][]Entry)
	tagSet := make(map[string]struct{})
	for _, e := range entries {
		if e.ID != "" {
			if _, dup := byID[e.ID]; !dup {
				byID[e.ID] = e
			}
		}
		byRel[e.RelPath] = e
		if e.Slug != "" {
			bySlug[e.Slug] = append(bySlug[e.Slug], e)
		}
		for _, t := range e.Frontmatter.Tags {
			if t == "" {
				continue
			}
			lower := strings.ToLower(t)
			byTag[lower] = append(byTag[lower], e)
			tagSet[lower] = struct{}{}
		}
	}

	allTags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		allTags = append(allTags, t)
	}
	sort.Strings(allTags)

	i.mu.Lock()
	i.entries = entries
	i.byID = byID
	i.byRel = byRel
	i.bySlug = bySlug
	i.byTag = byTag
	i.allTags = allTags
	i.mu.Unlock()

	return nil
}

// Reload requests an index rebuild and returns a channel that closes when a
// build has completed that reflects the tree state at or after this call.
//
// Scheduling rules:
//   - Idle: start a new build immediately.
//   - Build in-flight: coalesce — queue at most one follow-up. Every caller
//     that arrives while the in-flight build runs receives the same follow-up's
//     done channel, so they only observe completion after a full walk that
//     started after their request.
//
// Callers that only need "the current build" can read the returned channel;
// callers that do not care (e.g. warmup on a navigation) may ignore it.
func (i *Index) Reload() <-chan struct{} {
	i.buildMu.Lock()
	if i.curDone == nil {
		done := make(chan struct{})
		i.curDone = done
		i.buildMu.Unlock()
		go i.runBuild(done)
		return done
	}
	if i.queuedDone == nil {
		i.queuedDone = make(chan struct{})
	}
	done := i.queuedDone
	i.buildMu.Unlock()
	return done
}

// runBuild executes one build and signals done; if another Reload request
// arrived during the build, it chains into the follow-up build in the same
// goroutine lineage.
//
// The state-machine cleanup runs in a deferred block so that even if build
// panics, waiters on done are released and any queued follow-up still gets
// scheduled — without this, waiting callers would block forever.
func (i *Index) runBuild(done chan struct{}) {
	defer func() {
		i.buildMu.Lock()
		next := i.queuedDone
		i.queuedDone = nil
		i.curDone = next
		i.buildMu.Unlock()

		close(done)

		if next != nil {
			go i.runBuild(next)
		}
	}()

	if err := i.build(); err != nil {
		i.cfg.logger.log(fmt.Errorf("index reload failed: %w", err))
	}
}

// Root returns the absolute path the index was built from.
func (i *Index) Root() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.root
}

// Entries returns all entries, newest first (descending RelPath). The slice
// is a fresh copy; Tags and Aliases slices on each entry are also copied so
// callers may mutate the result without affecting the index.
func (i *Index) Entries() []Entry {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]Entry, len(i.entries))
	for j, e := range i.entries {
		out[j] = cloneEntry(e)
	}
	return out
}

// ByID returns the entry with the given numeric ID, or false. When multiple
// entries share an ID the newest wins.
func (i *Index) ByID(id string) (Entry, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	e, ok := i.byID[id]
	if !ok {
		return Entry{}, false
	}
	return cloneEntry(e), true
}

// ByRel returns the entry whose RelPath exactly matches rel, or false. rel
// must be the forward- or OS-slash path used during the walk (filepath.Join
// of year, month, basename under strict scan).
func (i *Index) ByRel(rel string) (Entry, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	e, ok := i.byRel[rel]
	if !ok {
		return Entry{}, false
	}
	return cloneEntry(e), true
}

// BySlug returns all entries whose Slug exactly matches slug, newest first.
// A nil return means no match; the returned slice is a fresh copy.
func (i *Index) BySlug(slug string) []Entry {
	i.mu.RLock()
	defer i.mu.RUnlock()
	src := i.bySlug[slug]
	if len(src) == 0 {
		return nil
	}
	out := make([]Entry, len(src))
	for j, e := range src {
		out[j] = cloneEntry(e)
	}
	return out
}

// ByTag returns all entries whose frontmatter tags contain tag, newest first.
// Comparison is case-insensitive; body hashtags are not indexed here (use
// ExtractHashtags or FilterByTags for that source). A nil return means no
// match.
func (i *Index) ByTag(tag string) []Entry {
	i.mu.RLock()
	defer i.mu.RUnlock()
	src := i.byTag[strings.ToLower(tag)]
	if len(src) == 0 {
		return nil
	}
	out := make([]Entry, len(src))
	for j, e := range src {
		out[j] = cloneEntry(e)
	}
	return out
}

// Tags returns the sorted, lowercased, deduplicated set of frontmatter tags
// in the index. The returned slice is freshly allocated.
func (i *Index) Tags() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]string, len(i.allTags))
	copy(out, i.allTags)
	return out
}

// Resolve mirrors ResolveRef's priority chain against the cached index —
// no rescan, no re-read:
//  1. empty query → most recent entry
//  2. numeric ID → exact ByID match (strict; never falls through)
//  3. type with special behavior → most recent entry with that Type
//  4. path-like query (absolute or contains a separator) → ByRel after
//     filesystem resolution; errors on paths outside root or missing files
//  5. slug substring → most recent entry whose Slug contains the query
//
// Options narrow the candidate set before the chain runs; see WithDate.
// With a date filter the by-ID and by-path map lookups stay O(1); the match
// is discarded after the fact if its Date does not match.
//
// Returns (entry, true, nil) on match, (zero, false, nil) on no match, and
// (zero, false, err) only for genuine failures (path outside root, symlink
// resolution error). The bool-vs-error split lets callers distinguish
// "note not found" from "I/O broke."
//
// The RLock covers only the snapshot of the index state; the path-resolution
// syscalls and map/slice reads run after it's released, so a future Reload
// writer is not blocked on filesystem round-trips. This relies on the
// swap-only Reload discipline — the aliased slice and maps must remain
// immutable for the lifetime of the snapshot.
func (i *Index) Resolve(query string, opts ...ResolveOption) (Entry, bool, error) {
	cfg := resolveConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	query = strings.TrimSpace(query)

	i.mu.RLock()
	root := i.root
	entries := i.entries
	byID := i.byID
	byRel := i.byRel
	i.mu.RUnlock()

	if cfg.date != "" {
		filtered := make([]Entry, 0, len(entries))
		for _, e := range entries {
			if e.Date == cfg.date {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if query == "" {
		if len(entries) == 0 {
			return Entry{}, false, nil
		}
		return cloneEntry(entries[0]), true, nil
	}

	if IsID(query) {
		e, ok := byID[query]
		if !ok {
			return Entry{}, false, nil
		}
		if cfg.date != "" && e.Date != cfg.date {
			return Entry{}, false, nil
		}
		return cloneEntry(e), true, nil
	}

	if HasSpecialBehavior(query) {
		for _, e := range entries {
			if e.Type == query {
				return cloneEntry(e), true, nil
			}
		}
	}

	if filepath.IsAbs(query) || strings.ContainsAny(query, `/\`) {
		rel, err := resolveRelPath(root, query)
		if err != nil {
			return Entry{}, false, err
		}
		e, ok := byRel[rel]
		if !ok {
			return Entry{}, false, nil
		}
		if cfg.date != "" && e.Date != cfg.date {
			return Entry{}, false, nil
		}
		return cloneEntry(e), true, nil
	}

	for _, e := range entries {
		if e.Slug != "" && strings.Contains(e.Slug, query) {
			return cloneEntry(e), true, nil
		}
	}

	return Entry{}, false, nil
}

// cloneEntry returns e with Tags and Aliases deep-copied so callers can
// mutate the returned value without racing other readers of the same index
// entry. Frontmatter.Extra is shared by reference — callers treating Extra
// as mutable should copy it themselves.
func cloneEntry(e Entry) Entry {
	e.Frontmatter.Tags = cloneStrings(e.Frontmatter.Tags)
	e.Frontmatter.Aliases = cloneStrings(e.Frontmatter.Aliases)
	e.bodyHashtags = cloneStrings(e.bodyHashtags)
	return e
}

func cloneStrings(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}

// normalizeHashtags lowercases and deduplicates a hashtag list from
// ExtractHashtags into the canonical form used by Entry.bodyHashtags.
// Returns nil when the input is empty so equality checks against nil hold.
func normalizeHashtags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(raw))
	for _, t := range raw {
		if t == "" {
			continue
		}
		set[strings.ToLower(t)] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
