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

// Entry is a fully-hydrated note record: the filename-derived Note plus
// frontmatter and file stat metadata. It is the unit Index exposes to
// downstream consumers (notes-view, notes-pub) that previously maintained
// their own in-memory indexes.
type Entry struct {
	Note
	Frontmatter Frontmatter
	ModTime     time.Time
	Size        int64
}

// Index is an in-memory, read-only snapshot of a notes store. Build one with
// Load; future reload semantics will swap state atomically under the RWMutex,
// so all read methods already take RLock today.
type Index struct {
	root string

	mu      sync.RWMutex
	entries []Entry
	byID    map[string]Entry
	byRel   map[string]Entry
	bySlug  map[string][]Entry
	byTag   map[string][]Entry
	allTags []string
}

type loadConfig struct {
	frontmatter bool
	workers     int
	scanOpts    ScanOptions
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

// Load walks root once, parses frontmatter concurrently, and returns a
// populated Index. A single concurrent pass replaces the Scan → FilterByTags
// → ExtractTags re-read chain that duplicated I/O for each query.
//
// Per-note frontmatter parse errors are logged to stderr (matching ParseNote's
// existing behavior) and leave that entry's Frontmatter zero; they never abort
// the load. Any file-read or stat error aborts the load.
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

	notes, err := Scan(root, cfg.scanOpts)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(notes))
	for i, n := range notes {
		entries[i] = Entry{Note: n}
	}

	if len(entries) > 0 {
		workers := cfg.workers
		if workers > len(entries) {
			workers = len(entries)
		}

		g, ctx := errgroup.WithContext(context.Background())
		jobs := make(chan int)

		g.Go(func() error {
			defer close(jobs)
			for i := range entries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case jobs <- i:
				}
			}
			return nil
		})

		for w := 0; w < workers; w++ {
			g.Go(func() error {
				for i := range jobs {
					path := filepath.Join(root, entries[i].RelPath)
					info, err := os.Stat(path)
					if err != nil {
						return err
					}
					entries[i].ModTime = info.ModTime()
					entries[i].Size = info.Size()
					if cfg.frontmatter {
						data, err := os.ReadFile(path)
						if err != nil {
							return err
						}
						fm, _, parseErr := ParseNote(data)
						if parseErr != nil {
							fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, parseErr)
							continue
						}
						entries[i].Frontmatter = fm
					}
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}
	}

	idx := &Index{
		root:    root,
		entries: entries,
		byID:    make(map[string]Entry, len(entries)),
		byRel:   make(map[string]Entry, len(entries)),
		bySlug:  make(map[string][]Entry),
		byTag:   make(map[string][]Entry),
	}

	tagSet := make(map[string]struct{})
	for _, e := range entries {
		if e.ID != "" {
			if _, dup := idx.byID[e.ID]; !dup {
				idx.byID[e.ID] = e
			}
		}
		idx.byRel[e.RelPath] = e
		if e.Slug != "" {
			idx.bySlug[e.Slug] = append(idx.bySlug[e.Slug], e)
		}
		for _, t := range e.Frontmatter.Tags {
			if t == "" {
				continue
			}
			lower := strings.ToLower(t)
			idx.byTag[lower] = append(idx.byTag[lower], e)
			tagSet[lower] = struct{}{}
		}
	}

	idx.allTags = make([]string, 0, len(tagSet))
	for t := range tagSet {
		idx.allTags = append(idx.allTags, t)
	}
	sort.Strings(idx.allTags)

	return idx, nil
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
func (i *Index) Resolve(query string) (Entry, bool, error) {
	query = strings.TrimSpace(query)

	i.mu.RLock()
	root := i.root
	entries := i.entries
	byID := i.byID
	byRel := i.byRel
	i.mu.RUnlock()

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
