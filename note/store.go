package note

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ErrNotFound is returned (wrapped) from resolveRelPath when a path-like
// query cannot be followed under the store root. Callers that wrap a miss
// from Index.Resolve into an error should reuse this sentinel so users can
// match with errors.Is:
//
//	if errors.Is(err, note.ErrNotFound) { … }
//
// Index.Resolve and Index.ByID/ByRel/BySlug keep the (value, bool) miss
// convention — the bool distinguishes "no match" from I/O failure without a
// sentinel comparison.
var ErrNotFound = errors.New("note not found")

// ScanOptions configures Scan's directory traversal.
//
// Strict=true (the default) restricts discovery to the canonical YYYY/MM/*.md
// layout used by notes-cli: only top-level directories whose name is all
// digits are considered years, and only their two-digit all-digit
// subdirectories are considered months. Other entries are ignored.
//
// Strict=false walks the entire tree under root with filepath.WalkDir and
// considers every *.md file whose base name parses via ParseFilename,
// regardless of nesting depth or parent directory naming. This is the layout
// downstream tools such as notes-view consume; opt in explicitly when you
// need it.
type ScanOptions struct {
	Strict bool
	logger Logger
}

// ScanOption configures Scan. All options are optional; pass zero or more.
type ScanOption func(*ScanOptions)

// WithStrict sets Scan's strict-layout mode. Default true. Set false to walk
// every *.md file under root regardless of layout.
func WithStrict(b bool) ScanOption {
	return func(o *ScanOptions) { o.Strict = b }
}

// WithScanLogger installs a Logger for non-fatal warnings from Scan — today
// that is "subdirectory unreadable, skipping" in both strict and lenient
// modes. Default: no-op (the scan silently skips, matching the package rule
// that note/ does not write to os.Stderr).
func WithScanLogger(l Logger) ScanOption {
	return func(o *ScanOptions) { o.logger = l }
}

// Scan enumerates notes under root.
//
// Called as Scan(root) it preserves the historical strict YYYY/MM/*.md
// discipline. Pass WithStrict(false) to walk every *.md file under root
// regardless of layout.
//
// Unreadable subdirectories are skipped in both modes so a single permission
// glitch can't break ls/tags/resolve; pass WithScanLogger to surface those
// warnings to the caller.
func Scan(root string, opts ...ScanOption) ([]Ref, error) {
	cfg := ScanOptions{Strict: true}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Strict {
		return scanStrict(root, cfg.logger)
	}
	return scanLenient(root, cfg.logger)
}

func scanStrict(root string, log Logger) ([]Ref, error) {
	var notes []Ref

	years, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, y := range years {
		if !y.IsDir() || !IsDigits(y.Name()) {
			continue
		}

		yearPath := filepath.Join(root, y.Name())
		months, err := os.ReadDir(yearPath)
		if err != nil {
			log.log(fmt.Errorf("%s: %w", yearPath, err))
			continue
		}

		for _, m := range months {
			if !m.IsDir() || len(m.Name()) != 2 || !IsDigits(m.Name()) {
				continue
			}

			monthPath := filepath.Join(yearPath, m.Name())
			files, err := os.ReadDir(monthPath)
			if err != nil {
				log.log(fmt.Errorf("%s: %w", monthPath, err))
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

func scanLenient(root string, log Logger) ([]Ref, error) {
	var notes []Ref

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			log.log(fmt.Errorf("%s: %w", path, err))
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

// ResolveOption configures Index.Resolve. All options are optional; pass
// zero or more.
type ResolveOption func(*resolveConfig)

type resolveConfig struct {
	date string
}

// WithDate restricts Index.Resolve candidates to notes matching the given
// YYYYMMDD date string. An empty string disables the filter (the default).
func WithDate(date string) ResolveOption {
	return func(c *resolveConfig) { c.date = date }
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
		return "", fmt.Errorf("%w: %s", ErrNotFound, query)
	}
	rel, err := filepath.Rel(absRoot, absQuery)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside notes directory: %s", query)
	}
	return rel, nil
}

// FilterByFilename returns entries whose filename contains the fragment (case-insensitive).
func FilterByFilename(entries []Entry, fragment string) []Entry {
	fragment = strings.ToLower(fragment)
	var results []Entry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(filepath.Base(e.RelPath)), fragment) {
			results = append(results, e)
		}
	}
	return results
}

// FilterByTags returns entries whose merged tags (frontmatter `tags:` plus body
// hashtags, case-folded) include every tag in tags. The caller supplies the
// entries — typically from a single Load — so no additional filesystem walk
// or frontmatter read happens here.
//
// An entry whose frontmatter failed to parse during Load still considers its
// body hashtags (Load logs the parse error and falls back to body-only);
// entries from Load(WithFrontmatter(false)) have empty MergedTags and match
// only if tags is also empty.
func FilterByTags(entries []Entry, tags []string) []Entry {
	if len(entries) == 0 {
		return nil
	}
	var results []Entry
	for _, e := range entries {
		if hasAllTags(e.MergedTags(), tags) {
			results = append(results, e)
		}
	}
	return results
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

// FilterByDate returns entries whose Date field matches the given YYYYMMDD string.
func FilterByDate(entries []Entry, date string) []Entry {
	var results []Entry
	for _, e := range entries {
		if e.Date == date {
			results = append(results, e)
		}
	}
	return results
}

// FilterBySlug returns entries with an exact slug match.
func FilterBySlug(entries []Entry, slug string) []Entry {
	var results []Entry
	for _, e := range entries {
		if e.Slug == slug {
			results = append(results, e)
		}
	}
	return results
}

// FilterByTypes returns entries whose type matches any of the given values.
func FilterByTypes(entries []Entry, types []string) []Entry {
	set := toSet(types)
	var results []Entry
	for _, e := range entries {
		if _, ok := set[e.Type]; ok {
			results = append(results, e)
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
	if IsDigits(slug) {
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
