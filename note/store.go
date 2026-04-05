package note

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scan enumerates notes under root using the known YYYY/MM/ directory structure.
// Only directories matching year (all digits) and month (two-digit) patterns are visited.
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
			return nil, err
		}

		for _, m := range months {
			if !m.IsDir() || len(m.Name()) != 2 || !isDigits(m.Name()) {
				continue
			}

			monthPath := filepath.Join(yearPath, m.Name())
			files, err := os.ReadDir(monthPath)
			if err != nil {
				return nil, err
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
//  1. Numeric ID
//  2. Absolute or relative path (must exist and be under root)
//  3. Basename (exact filename without .md extension)
//  4. Slug
//  5. Type — most recent note of that type (e.g. "todo", "backlog", "weekly")
func ResolveRef(root, query string) (*Note, error) {
	return ResolveRefDate(root, query, "")
}

// ResolveRefDate works like ResolveRef but optionally restricts candidates to
// notes matching the given YYYYMMDD date string. Pass "" to skip date filtering.
func ResolveRefDate(root, query, date string) (*Note, error) {
	query = strings.TrimSpace(query)

	notes, err := Scan(root)
	if err != nil {
		return nil, err
	}

	if date != "" {
		notes = FilterByDate(notes, date)
	}

	// Step 1: numeric ID
	if query != "" && isDigits(query) {
		for i := range notes {
			if notes[i].ID == query {
				return &notes[i], nil
			}
		}
		// No early return: fall through to basename/slug/type resolution.
		// This allows all-digit slugs (e.g. "999") to be found when no ID matches.
	}

	// Step 2: absolute or relative path
	if strings.ContainsRune(query, filepath.Separator) {
		absPath, err := filepath.Abs(query)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve path: %w", err)
		}
		absRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve notes path: %w", err)
		}
		absPathResolved, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return nil, fmt.Errorf("note not found: %s", query)
		}
		if !strings.HasPrefix(absPathResolved, absRoot+string(filepath.Separator)) {
			return nil, fmt.Errorf("path is outside notes directory: %s", query)
		}
		rel, err := filepath.Rel(absRoot, absPathResolved)
		if err != nil {
			return nil, fmt.Errorf("cannot compute relative path: %w", err)
		}
		for i := range notes {
			if notes[i].RelPath == rel {
				return &notes[i], nil
			}
		}
		return nil, fmt.Errorf("note not found: %s", query)
	}

	stripped := strings.TrimSuffix(query, ".md")

	// Step 3: basename
	for i := range notes {
		if notes[i].BaseName == stripped {
			return &notes[i], nil
		}
	}

	// Step 4: slug
	for i := range notes {
		if notes[i].Slug != "" && notes[i].Slug == query {
			return &notes[i], nil
		}
	}

	// Step 5: type — most recent match
	for i := range notes {
		if notes[i].Type != "" && notes[i].Type == query {
			return &notes[i], nil
		}
	}

	return nil, fmt.Errorf("note not found: %s", query)
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

// FilterByTags returns notes that contain all of the given tags in their frontmatter.
func FilterByTags(notes []Note, root string, tags []string) ([]Note, error) {
	var results []Note
	for _, n := range notes {
		data, err := os.ReadFile(filepath.Join(root, n.RelPath))
		if err != nil {
			return nil, err
		}
		noteTags := ParseFrontmatterFields(data).Tags
		if hasAllTags(noteTags, tags) {
			results = append(results, n)
		}
	}
	return results, nil
}

func hasAllTags(noteTags []string, required []string) bool {
	set := make(map[string]struct{}, len(noteTags))
	for _, t := range noteTags {
		set[t] = struct{}{}
	}
	for _, r := range required {
		if _, ok := set[r]; !ok {
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

// ValidateSlug returns an error if the slug is ambiguous with note ID resolution.
// All-digit slugs are rejected because they conflict with numeric ID lookup.
func ValidateSlug(slug string) error {
	if slug != "" && isDigits(slug) {
		return fmt.Errorf("slug %q is all digits, which conflicts with note ID resolution", slug)
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
