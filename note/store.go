package note

import (
	"fmt"
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
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
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

// ResolveRef resolves a note reference to a Note using the following priority:
//  1. Numeric ID
//  2. Absolute or relative path (must exist and be under root)
//  3. Basename (exact filename without .md extension)
//  4. Slug
//  5. Type — most recent note of that type (e.g. "todo", "backlog", "weekly")
func ResolveRef(root, query string) (*Note, error) {
	query = strings.TrimSpace(query)

	notes, err := Scan(root)
	if err != nil {
		return nil, err
	}

	// Step 1: numeric ID
	if query != "" && isDigits(query) {
		for i := range notes {
			if notes[i].ID == query {
				return &notes[i], nil
			}
		}
		return nil, fmt.Errorf("note not found: %s", query)
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
