package note

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scan walks the store directory and returns all valid notes, sorted newest first.
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

// Resolve finds a single note matching the query by ID, slug, type, or base filename.
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

	// Try exact type match
	for i := range notes {
		if notes[i].Type != "" && notes[i].Type == query {
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

// Filter returns all notes whose base filename or type contains the fragment (case-insensitive).
func Filter(notes []Note, fragment string) []Note {
	fragment = strings.ToLower(fragment)
	var results []Note
	for _, n := range notes {
		if strings.Contains(strings.ToLower(n.BaseName), fragment) ||
			strings.Contains(strings.ToLower(n.Type), fragment) {
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
		noteTags := ParseTags(data)
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

// FilterBySlug returns notes with an exact slug match.
func FilterBySlug(notes []Note, slug string) []Note {
	return FilterBySlugs(notes, []string{slug})
}

// FilterBySlugs returns notes whose slug matches any of the given values.
func FilterBySlugs(notes []Note, slugs []string) []Note {
	set := toSet(slugs)
	var results []Note
	for _, n := range notes {
		if _, ok := set[n.Slug]; ok {
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

func toSet(vals []string) map[string]struct{} {
	m := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		m[v] = struct{}{}
	}
	return m
}
