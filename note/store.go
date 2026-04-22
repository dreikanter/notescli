package note

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ScanOptions configures Scan's directory traversal.
//
// Strict=true (the default when no options are passed) restricts discovery to
// the canonical YYYY/MM/*.md layout used by notes-cli: only top-level directories
// whose name is all digits are considered years, and only their two-digit
// all-digit subdirectories are considered months. Other entries are ignored.
//
// Strict=false walks the entire tree under root with filepath.WalkDir and
// considers every *.md file whose base name parses via ParseFilename, regardless
// of nesting depth or parent directory naming. This is the layout downstream
// tools such as notes-view consume; opt in explicitly when you need it.
type ScanOptions struct {
	Strict bool
}

// Scan enumerates notes under root.
//
// Called as Scan(root) it preserves the historical strict YYYY/MM/*.md
// discipline. Pass ScanOptions{Strict: false} to walk every *.md file under
// root regardless of layout. Only the first option in opts is consulted;
// additional values are ignored.
//
// Unreadable subdirectories are logged to stderr and skipped in both modes,
// matching the per-note parse-error behavior, so a single permission glitch
// can't break ls/tags/resolve.
func Scan(root string, opts ...ScanOptions) ([]Note, error) {
	strict := true
	if len(opts) > 0 {
		strict = opts[0].Strict
	}
	if strict {
		return scanStrict(root)
	}
	return scanLenient(root)
}

func scanStrict(root string) ([]Note, error) {
	var notes []Note

	years, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, y := range years {
		if !y.IsDir() || !isDigits(y.Name()) {
			continue
		}

		yearPath := filepath.Join(root, y.Name())
		months, err := os.ReadDir(yearPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", yearPath, err)
			continue
		}

		for _, m := range months {
			if !m.IsDir() || len(m.Name()) != 2 || !isDigits(m.Name()) {
				continue
			}

			monthPath := filepath.Join(yearPath, m.Name())
			files, err := os.ReadDir(monthPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warn: %s: %v\n", monthPath, err)
				continue
			}

			for _, f := range files {
				if f.IsDir() || filepath.Ext(f.Name()) != ".md" {
					continue
				}

				base := strings.TrimSuffix(f.Name(), ".md")
				n, parseErr := ParseFilename(base)
				if parseErr != nil {
					continue
				}

				n.RelPath = filepath.Join(y.Name(), m.Name(), f.Name())
				notes = append(notes, n)
			}
		}
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].RelPath > notes[j].RelPath
	})

	return notes, nil
}

func scanLenient(root string) ([]Note, error) {
	var notes []Note

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) != ".md" {
			return nil
		}
		base := strings.TrimSuffix(d.Name(), ".md")
		n, parseErr := ParseFilename(base)
		if parseErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		n.RelPath = rel
		notes = append(notes, n)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].RelPath > notes[j].RelPath
	})

	return notes, nil
}

// ResolveRef resolves a note reference to a Note using the following priority:
//  1. Numeric ID — exact match; all-digit queries never fall through
//  2. Type with special behavior (todo, backlog, weekly) — most recent match
//  3. Path — absolute or relative path with separator, exact match under root
//  4. Slug substring — most recent note whose slug contains the query
//
// Implementation routes through Index.Resolve on a WithFrontmatter(false)
// load, so CLI commands that already hold an Index can call Index.Resolve
// directly and skip this wrapper.
func ResolveRef(root, query string) (Note, error) {
	return ResolveRefDate(root, query, "")
}

// ResolveRefDate works like ResolveRef but optionally restricts candidates to
// notes matching the given YYYYMMDD date string. Pass "" to skip date filtering.
func ResolveRefDate(root, query, date string) (Note, error) {
	idx, err := Load(root, WithFrontmatter(false))
	if err != nil {
		return Note{}, err
	}

	if date == "" {
		e, ok, err := idx.Resolve(query)
		if err != nil {
			return Note{}, err
		}
		if !ok {
			return Note{}, fmt.Errorf("note not found: %s", strings.TrimSpace(query))
		}
		return e.Note, nil
	}

	entries := idx.Entries()
	filtered := filterEntriesByDate(entries, date)
	e, ok, err := resolveInEntries(root, filtered, query)
	if err != nil {
		return Note{}, err
	}
	if !ok {
		return Note{}, fmt.Errorf("note not found: %s", strings.TrimSpace(query))
	}
	return e.Note, nil
}

// filterEntriesByDate returns entries whose Date field matches the given
// YYYYMMDD string, preserving input order.
func filterEntriesByDate(entries []Entry, date string) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Date == date {
			out = append(out, e)
		}
	}
	return out
}

// resolveInEntries applies the ResolveRef priority chain to an arbitrary
// (pre-filtered) entry slice. It mirrors Index.Resolve but linear-scans,
// because date-restricted subsets don't share the index's maps.
func resolveInEntries(root string, entries []Entry, query string) (Entry, bool, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		if len(entries) == 0 {
			return Entry{}, false, nil
		}
		return entries[0], true, nil
	}

	if IsID(query) {
		for _, e := range entries {
			if e.ID == query {
				return e, true, nil
			}
		}
		return Entry{}, false, nil
	}

	if HasSpecialBehavior(query) {
		for _, e := range entries {
			if e.Type == query {
				return e, true, nil
			}
		}
	}

	if filepath.IsAbs(query) || strings.ContainsAny(query, `/\`) {
		rel, err := resolveRelPath(root, query)
		if err != nil {
			return Entry{}, false, err
		}
		for _, e := range entries {
			if e.RelPath == rel {
				return e, true, nil
			}
		}
		return Entry{}, false, nil
	}

	for _, e := range entries {
		if e.Slug != "" && strings.Contains(e.Slug, query) {
			return e, true, nil
		}
	}

	return Entry{}, false, nil
}

// resolveRelPath converts a path-like query to a note RelPath under root.
// Returns an error if the path does not exist or escapes root.
func resolveRelPath(root, query string) (string, error) {
	queryPath := query
	if !filepath.IsAbs(queryPath) {
		abs, err := filepath.Abs(queryPath)
		if err != nil {
			return "", fmt.Errorf("cannot resolve path: %w", err)
		}
		queryPath = abs
	}
	absRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("cannot resolve notes path: %w", err)
	}
	absQuery, err := filepath.EvalSymlinks(queryPath)
	if err != nil {
		return "", fmt.Errorf("note not found: %s", query)
	}
	rel, err := filepath.Rel(absRoot, absQuery)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside notes directory: %s", query)
	}
	return rel, nil
}

// Filter returns all notes whose filename contains the fragment (case-insensitive).
func Filter(notes []Note, fragment string) []Note {
	fragment = strings.ToLower(fragment)
	var results []Note
	for _, n := range notes {
		if strings.Contains(strings.ToLower(filepath.Base(n.RelPath)), fragment) {
			results = append(results, n)
		}
	}
	return results
}

// FilterByTags returns notes that contain all of the given tags. Tag sources
// mirror ExtractTags: frontmatter `tags:` fields and body hashtags (#word).
// Comparison is case-insensitive.
//
// Implementation routes through Load so the tag index is built from a single
// concurrent file-read pass; the []Note signature is preserved as a shim for
// callers that don't yet hold an Index. A per-note frontmatter parse error is
// logged to stderr during Load and leaves the note's frontmatter tags empty;
// body hashtags are still considered.
func FilterByTags(notes []Note, root string, tags []string) ([]Note, error) {
	if len(notes) == 0 {
		return nil, nil
	}
	idx, err := Load(root)
	if err != nil {
		return nil, err
	}
	var results []Note
	for _, n := range notes {
		e, ok := idx.ByRel(n.RelPath)
		if !ok {
			continue
		}
		if hasAllTags(e.MergedTags(), tags) {
			results = append(results, n)
		}
	}
	return results, nil
}

// hasAllTags reports whether every entry in required appears in noteTags,
// case-insensitively. noteTags may be pre-lowercased (as from Entry.MergedTags)
// or mixed-case — both sides are folded for comparison.
func hasAllTags(noteTags []string, required []string) bool {
	set := make(map[string]struct{}, len(noteTags))
	for _, t := range noteTags {
		set[strings.ToLower(t)] = struct{}{}
	}
	for _, r := range required {
		if _, ok := set[strings.ToLower(r)]; !ok {
			return false
		}
	}
	return true
}

// FilterByDate returns notes whose Date field matches the given YYYYMMDD string.
func FilterByDate(notes []Note, date string) []Note {
	var results []Note
	for _, n := range notes {
		if n.Date == date {
			results = append(results, n)
		}
	}
	return results
}

// FilterBySlug returns notes with an exact slug match.
func FilterBySlug(notes []Note, slug string) []Note {
	var results []Note
	for _, n := range notes {
		if n.Slug == slug {
			results = append(results, n)
		}
	}
	return results
}

// FilterByTypes returns notes whose type matches any of the given values.
func FilterByTypes(notes []Note, types []string) []Note {
	set := toSet(types)
	var results []Note
	for _, n := range notes {
		if _, ok := set[n.Type]; ok {
			results = append(results, n)
		}
	}
	return results
}

var slugRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ValidateSlug returns an error if the slug cannot safely appear in a note
// filename. Empty slugs are accepted (they just omit the slug segment).
// All-digit slugs are rejected because they conflict with numeric ID lookup.
// Anything outside [A-Za-z0-9_-] is rejected to keep filenames portable and to
// avoid confusing ParseFilename's dot-suffix cache.
func ValidateSlug(slug string) error {
	if slug == "" {
		return nil
	}
	if isDigits(slug) {
		return fmt.Errorf("slug %q is all digits, which conflicts with note ID resolution", slug)
	}
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("slug %q contains invalid characters; only [A-Za-z0-9_-] are allowed", slug)
	}
	return nil
}

func toSet(vals []string) map[string]struct{} {
	m := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		m[v] = struct{}{}
	}
	return m
}
