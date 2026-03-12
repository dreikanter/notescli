package note

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Scan walks the archive directory and returns all valid notes, sorted newest first.
func Scan(root string) ([]Note, error) {
	var notes []Note

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		base := strings.TrimSuffix(filepath.Base(path), ".md")
		n, parseErr := ParseFilename(base)
		if parseErr != nil {
			return nil // skip files that don't match the naming convention
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}

		n.RelPath = rel
		notes = append(notes, n)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].RelPath > notes[j].RelPath
	})

	return notes, nil
}

// Resolve finds a single note matching the query by ID, slug, or base filename.
// Returns the first (most recent) match, or nil if not found.
func Resolve(notes []Note, query string) *Note {
	// Strip .md extension if provided
	query = strings.TrimSuffix(query, ".md")

	// Try exact ID match first
	for i := range notes {
		if notes[i].ID == query {
			return &notes[i]
		}
	}

	// Try exact slug match
	for i := range notes {
		if notes[i].Slug != "" && notes[i].Slug == query {
			return &notes[i]
		}
	}

	// Try exact base filename match
	for i := range notes {
		if notes[i].BaseName == query {
			return &notes[i]
		}
	}

	return nil
}

// Filter returns all notes whose base filename contains the fragment (case-insensitive).
func Filter(notes []Note, fragment string) []Note {
	fragment = strings.ToLower(fragment)
	var results []Note
	for _, n := range notes {
		if strings.Contains(strings.ToLower(n.BaseName), fragment) {
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
