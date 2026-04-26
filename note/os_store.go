package note

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// OSStore is the filesystem-backed Store. It wraps the existing
// root/YYYY/MM/YYYYMMDD_id_slug.md layout and reuses the package's existing
// filename, frontmatter, and atomic-write helpers.
//
// The Store interface never exposes filesystem paths. Callers that need the
// absolute path of an entry (e.g. the resolve command) type-assert to
// *OSStore and call AbsPath.
type OSStore struct {
	root string
}

var _ Store = (*OSStore)(nil)

// NewOSStore returns an OSStore rooted at root. The directory must already
// exist; use os.Stat at the caller if validation is required.
func NewOSStore(root string) *OSStore {
	return &OSStore{root: root}
}

// Root returns the absolute path the store is rooted at.
func (s *OSStore) Root() string { return s.root }

// fileRef is the filename-derived metadata for a single note. It is used
// internally to avoid re-parsing filenames repeatedly.
type fileRef struct {
	id       int
	date     string // YYYYMMDD
	slug     string
	noteType string // from filename dot-suffix; may be overridden by frontmatter
	relPath  string // path relative to root
}

// absPath returns the absolute filesystem path of r under root.
func (r fileRef) absPath(root string) string { return filepath.Join(root, r.relPath) }

// sortByRecency sorts refs newest-first by (date DESC, id DESC). Pure
// lexicographic ordering on the filename is not reliable because "_9_" sorts
// after "_10_" while ID 10 is newer.
func sortByRecency(refs []fileRef) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].date != refs[j].date {
			return refs[i].date > refs[j].date
		}
		return refs[i].id > refs[j].id
	})
}

// scanFileRefs walks the YYYY/MM subtree under root and returns every note
// filename that parses successfully. Unreadable subdirectories are skipped;
// they do not abort the scan.
func (s *OSStore) scanFileRefs() ([]fileRef, error) {
	var out []fileRef

	years, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	for _, y := range years {
		if !y.IsDir() || !IsDigits(y.Name()) {
			continue
		}
		months, err := os.ReadDir(filepath.Join(s.root, y.Name()))
		if err != nil {
			continue
		}
		for _, m := range months {
			if !m.IsDir() || len(m.Name()) != 2 || !IsDigits(m.Name()) {
				continue
			}
			monthDir := filepath.Join(s.root, y.Name(), m.Name())
			files, err := os.ReadDir(monthDir)
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() || filepath.Ext(f.Name()) != ".md" {
					continue
				}
				base := strings.TrimSuffix(f.Name(), ".md")
				ref, err := ParseFilename(base)
				if err != nil {
					continue
				}
				id, err := strconv.Atoi(ref.ID)
				if err != nil {
					continue
				}
				out = append(out, fileRef{
					id:       id,
					date:     ref.Date,
					slug:     ref.Slug,
					noteType: ref.Type,
					relPath:  filepath.Join(y.Name(), m.Name(), f.Name()),
				})
			}
		}
	}
	sortByRecency(out)
	return out, nil
}

// IDs returns every stored ID newest-first by (date DESC, id DESC) using
// only the filename layout — no file reads.
func (s *OSStore) IDs() ([]int, error) {
	refs, err := s.scanFileRefs()
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(refs))
	for i, r := range refs {
		ids[i] = r.id
	}
	return ids, nil
}

// Reconcile returns the delta between known and the current on-disk state.
// known maps note ID to the file mtime the caller last observed, usually from
// Entry.Meta.UpdatedAt returned by All, Get, Find, or a previous Reconcile.
// Files whose mtimes match known are skipped entirely: no file read and no YAML
// parse. Files whose mtimes differ are read and parsed, even when the on-disk
// mtime moved backwards; mtime equality is the cache key. Do not seed known
// from the Entry returned by Put: Put avoids an extra stat and its UpdatedAt is
// the write time, not necessarily the exact filesystem mtime.
//
// Caveats: filesystem mtime resolution can coalesce rapid writes on some
// filesystems, so a rewrite inside that resolution may be missed; tools such as
// rsync or touch can set mtimes backwards, which is why Reconcile compares
// equality rather than newer-than; a rename that changes the ID-bearing
// filename prefix appears as Removed+Added.
func (s *OSStore) Reconcile(known map[int]time.Time) (Diff, error) {
	refs, err := s.scanFileRefs()
	if err != nil {
		return Diff{}, err
	}

	seen := make(map[int]struct{}, len(refs))
	var diff Diff
	for _, ref := range refs {
		path := ref.absPath(s.root)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Diff{}, err
		}

		modTime := info.ModTime()
		if knownTime, ok := known[ref.id]; ok && knownTime.Equal(modTime) {
			seen[ref.id] = struct{}{}
			continue
		}

		entry, err := s.readEntry(ref)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Diff{}, err
		}
		seen[ref.id] = struct{}{}
		if _, ok := known[ref.id]; ok {
			diff.Updated = append(diff.Updated, entry)
		} else {
			diff.Added = append(diff.Added, entry)
		}
	}

	for id := range known {
		if _, ok := seen[id]; !ok {
			diff.Removed = append(diff.Removed, id)
		}
	}
	sort.Ints(diff.Removed)
	return diff, nil
}

// refMatchesFilename evaluates the subset of q that can be answered from the
// filename alone: WithType, WithSlug, WithExactDate, WithBeforeDate.
// Body-dependent filters (WithTag) are always passed as matching at this
// layer; readEntry decides.
func refMatchesFilename(r fileRef, q query) bool {
	if q.typeSet && r.noteType != q.noteType {
		return false
	}
	if q.slugSet && r.slug != q.slug {
		return false
	}
	if q.dateSet {
		if r.date != q.date.Format(DateFormat) {
			return false
		}
	}
	if q.beforeSet {
		if r.date >= q.beforeDate.Format(DateFormat) {
			return false
		}
	}
	return true
}

// All returns every entry matching opts, newest-first. Type/slug/date filters
// are evaluated from filenames; tag filters require reading file bodies.
func (s *OSStore) All(opts ...QueryOpt) ([]Entry, error) {
	return s.collect(opts, false)
}

// Find returns the newest entry matching opts, or ErrNotFound.
func (s *OSStore) Find(opts ...QueryOpt) (Entry, error) {
	entries, err := s.collect(opts, true)
	if err != nil {
		return Entry{}, err
	}
	if len(entries) == 0 {
		return Entry{}, ErrNotFound
	}
	return entries[0], nil
}

// collect is the shared read path for All and Find. When firstOnly is true
// it stops at the first body-matched entry; refs are already sorted newest-
// first so the first match is also the newest.
func (s *OSStore) collect(opts []QueryOpt, firstOnly bool) ([]Entry, error) {
	q := buildQuery(opts)

	refs, err := s.scanFileRefs()
	if err != nil {
		return nil, err
	}

	filtered := refs[:0:0]
	for _, r := range refs {
		if refMatchesFilename(r, q) {
			filtered = append(filtered, r)
		}
	}

	if firstOnly {
		for _, r := range filtered {
			entry, err := s.readEntry(r)
			if err != nil {
				return nil, err
			}
			if matches(entry, q) {
				return []Entry{entry}, nil
			}
		}
		return nil, nil
	}

	entries, err := s.readConcurrent(filtered)
	if err != nil {
		return nil, err
	}

	out := entries[:0]
	for _, e := range entries {
		if matches(e, q) {
			out = append(out, e)
		}
	}
	return out, nil
}

// readConcurrent reads each fileRef via a worker pool and returns entries
// in the same order as refs.
func (s *OSStore) readConcurrent(refs []fileRef) ([]Entry, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	workers := runtime.NumCPU()
	if workers > len(refs) {
		workers = len(refs)
	}

	entries := make([]Entry, len(refs))
	g, ctx := errgroup.WithContext(context.Background())
	jobs := make(chan int)

	g.Go(func() error {
		defer close(jobs)
		for i := range refs {
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
				entry, err := s.readEntry(refs[i])
				if err != nil {
					return err
				}
				entries[i] = entry
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return entries, nil
}

// readEntry loads a single file and converts it to a Entry. It populates
// Meta.Tags with the merged frontmatter+body-hashtag set and Meta.UpdatedAt
// from the file's ModTime.
func (s *OSStore) readEntry(r fileRef) (Entry, error) {
	path := r.absPath(s.root)
	data, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Entry{}, err
	}
	fm, body, err := ParseNote(data)
	if err != nil {
		return Entry{}, fmt.Errorf("%s: %w", path, err)
	}
	meta := frontmatterToMeta(fm, r, info.ModTime(), body)
	return Entry{
		ID:   r.id,
		Meta: meta,
		Body: string(body),
	}, nil
}

// Get returns the entry with the given ID.
func (s *OSStore) Get(id int) (Entry, error) {
	r, err := s.findFileRef(id)
	if err != nil {
		return Entry{}, err
	}
	return s.readEntry(r)
}

// Delete removes the file for the given ID.
func (s *OSStore) Delete(id int) error {
	r, err := s.findFileRef(id)
	if err != nil {
		return err
	}
	if err := os.Remove(r.absPath(s.root)); err != nil {
		return fmt.Errorf("cannot remove note: %w", err)
	}
	return nil
}

// findFileRef scans for a file whose parsed ID matches id.
func (s *OSStore) findFileRef(id int) (fileRef, error) {
	refs, err := s.scanFileRefs()
	if err != nil {
		return fileRef{}, err
	}
	for _, r := range refs {
		if r.id == id {
			return r, nil
		}
	}
	return fileRef{}, fmt.Errorf("%w: %d", ErrNotFound, id)
}

// Put writes entry. See Store.Put for the full contract. When entry.ID is
// zero a new ID is allocated via NextID (id.json + flock) and CreatedAt is
// defaulted to time.Now if zero. On updates with a changed slug or date the
// file is renamed atomically.
func (s *OSStore) Put(entry Entry) (Entry, error) {
	now := time.Now()

	if entry.ID == 0 {
		id, err := NextID(s.root)
		if err != nil {
			return Entry{}, err
		}
		entry.ID = id
		if entry.Meta.CreatedAt.IsZero() {
			entry.Meta.CreatedAt = now
		}
	}

	if entry.Meta.CreatedAt.IsZero() {
		return Entry{}, fmt.Errorf("note %d: CreatedAt is zero", entry.ID)
	}

	var oldPath string
	if prev, err := s.findFileRef(entry.ID); err == nil {
		oldPath = prev.absPath(s.root)
	} else if !errors.Is(err, ErrNotFound) {
		return Entry{}, err
	}

	newRelPath, newAbsPath := s.pathFor(entry)
	dirAbs := filepath.Dir(newAbsPath)
	if err := os.MkdirAll(dirAbs, StoreDirMode(s.root)); err != nil {
		return Entry{}, fmt.Errorf("cannot create %s: %w", filepath.Dir(newRelPath), err)
	}

	fm := metaToFrontmatter(entry.Meta)
	data, err := FormatNote(fm, []byte(entry.Body))
	if err != nil {
		return Entry{}, err
	}

	if oldPath != "" && oldPath != newAbsPath {
		if err := os.Rename(oldPath, newAbsPath); err != nil {
			return Entry{}, fmt.Errorf("cannot rename note: %w", err)
		}
	}

	if err := WriteAtomic(newAbsPath, data); err != nil {
		return Entry{}, err
	}

	entry.Meta.UpdatedAt = now
	return entry, nil
}

// AbsPath returns the absolute path the store would use for entry given its
// current Meta.CreatedAt, ID, and Meta.Slug. It derives the path purely from
// the entry's fields — no I/O.
func (s *OSStore) AbsPath(entry Entry) string {
	_, abs := s.pathFor(entry)
	return abs
}

// pathFor returns the rel/abs path the filename layout produces for entry.
func (s *OSStore) pathFor(entry Entry) (rel, abs string) {
	date := entry.Meta.CreatedAt.Format(DateFormat)
	name := Filename(date, entry.ID, entry.Meta.Slug, entry.Meta.Type)
	dir := DirPath(s.root, date)
	abs = filepath.Join(dir, name)
	rel, _ = filepath.Rel(s.root, abs)
	return rel, abs
}

// frontmatterToMeta converts the on-disk frontmatter into the public
// Meta. Body hashtags are merged into Meta.Tags; Meta.UpdatedAt is set
// from the file ModTime; Meta.CreatedAt falls back to the filename date when
// the frontmatter has no date.
func frontmatterToMeta(fm frontmatter, r fileRef, modTime time.Time, body []byte) Meta {
	created := fm.Date
	if created.IsZero() {
		if t, err := time.Parse(DateFormat, r.date); err == nil {
			created = t
		}
	}
	noteType := fm.Type
	if noteType == "" {
		noteType = r.noteType
	}
	slug := fm.Slug
	if slug == "" {
		slug = r.slug
	}

	return Meta{
		Title:       fm.Title,
		Slug:        slug,
		Type:        noteType,
		CreatedAt:   created,
		UpdatedAt:   modTime,
		Tags:        computeMergedTags(fm.Tags, normalizeHashtags(ExtractHashtags(body))),
		Aliases:     append([]string(nil), fm.Aliases...),
		Description: fm.Description,
		Public:      fm.Public,
		Extra:       extraFromYAML(fm.Extra),
	}
}

// metaToFrontmatter converts Meta into the on-disk frontmatter. Body
// hashtags are *not* stripped from Meta.Tags — they round-trip through the
// frontmatter alongside the originals. UpdatedAt is never written.
func metaToFrontmatter(m Meta) frontmatter {
	var aliases []string
	if len(m.Aliases) > 0 {
		aliases = append([]string(nil), m.Aliases...)
	}
	var tags []string
	if len(m.Tags) > 0 {
		tags = append([]string(nil), m.Tags...)
	}
	return frontmatter{
		Title:       m.Title,
		Slug:        m.Slug,
		Type:        m.Type,
		Date:        m.CreatedAt,
		Tags:        tags,
		Aliases:     aliases,
		Description: m.Description,
		Public:      m.Public,
		Extra:       extraToYAML(m.Extra),
	}
}

// extraFromYAML converts the internal yaml.Node map into the public
// map[string]any used by Meta.
func extraFromYAML(in map[string]yaml.Node) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, n := range in {
		var v any
		if err := n.Decode(&v); err != nil {
			out[k] = n
			continue
		}
		out[k] = v
	}
	return out
}

// extraToYAML converts a map[string]any back into the yaml.Node
// representation the frontmatter type expects on write.
func extraToYAML(in map[string]any) map[string]yaml.Node {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]yaml.Node, len(in))
	for k, v := range in {
		if n, ok := v.(yaml.Node); ok {
			out[k] = n
			continue
		}
		var node yaml.Node
		if err := node.Encode(v); err != nil {
			continue
		}
		out[k] = node
	}
	return out
}
