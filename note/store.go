package note

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Scan enumerates notes under root using the known YYYY/MM/ directory structure.
// Only directories matching year (all digits) and month (two-digit) patterns are visited.
// Unreadable year/month subdirectories are logged to stderr and skipped, matching
// the per-note parse-error behavior, so a single permission glitch can't break ls/tags/resolve.
func Scan(root string) ([]Note, error) {
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

// ResolveRef resolves a note reference to a Note using the following priority:
//  1. Numeric ID — exact match; all-digit queries never fall through
//  2. Type with special behavior (todo, backlog, weekly) — most recent match
//  3. Path — absolute or relative path with separator, exact match under root
//  4. Slug substring — most recent note whose slug contains the query
func ResolveRef(root, query string) (Note, error) {
	return ResolveRefDate(root, query, "")
}

// ResolveRefDate works like ResolveRef but optionally restricts candidates to
// notes matching the given YYYYMMDD date string. Pass "" to skip date filtering.
func ResolveRefDate(root, query, date string) (Note, error) {
	query = strings.TrimSpace(query)

	notes, err := Scan(root)
	if err != nil {
		return Note{}, err
	}

	if date != "" {
		notes = FilterByDate(notes, date)
	}

	// Step 1: numeric ID — strict, no fallthrough
	if query != "" && isDigits(query) {
		for i := range notes {
			if notes[i].ID == query {
				return notes[i], nil
			}
		}
		return Note{}, fmt.Errorf("note not found: %s", query)
	}

	// Step 2: type — most recent match
	if HasSpecialBehavior(query) {
		for i := range notes {
			if notes[i].Type == query {
				return notes[i], nil
			}
		}
	}

	// Step 3: path (absolute, or relative containing a separator) — exact match
	if filepath.IsAbs(query) || strings.ContainsAny(query, "/\\") {
		rel, err := resolveRelPath(root, query)
		if err != nil {
			return Note{}, err
		}
		for i := range notes {
			if notes[i].RelPath == rel {
				return notes[i], nil
			}
		}
		return Note{}, fmt.Errorf("note not found: %s", query)
	}

	// Step 4: slug substring — most recent match
	for i := range notes {
		if strings.Contains(notes[i].Slug, query) {
			return notes[i], nil
		}
	}

	return Note{}, fmt.Errorf("note not found: %s", query)
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
// A per-note frontmatter parse error is written to stderr and the note's
// frontmatter tags are skipped (body hashtags are still considered).
func FilterByTags(notes []Note, root string, tags []string) ([]Note, error) {
	var results []Note
	for _, n := range notes {
		path := filepath.Join(root, n.RelPath)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fm, body, parseErr := ParseNote(data)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, parseErr)
		}
		hashtags := extractHashtags(body)
		noteTags := make([]string, 0, len(fm.Tags)+len(hashtags))
		noteTags = append(noteTags, fm.Tags...)
		noteTags = append(noteTags, hashtags...)
		if hasAllTags(noteTags, tags) {
			results = append(results, n)
		}
	}
	return results, nil
}

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

// FilterByType returns notes with an exact type match.
func FilterByType(notes []Note, noteType string) []Note {
	return FilterByTypes(notes, []string{noteType})
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
