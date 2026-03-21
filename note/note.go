package note

import (
	"fmt"
	"path/filepath"
	"strings"
)

// KnownTypes lists the well-known note types encoded as secondary file extensions.
var KnownTypes = []string{"todo", "backlog", "weekly"}

// IsKnownType reports whether s is a recognized note type.
func IsKnownType(s string) bool {
	for _, t := range KnownTypes {
		if s == t {
			return true
		}
	}
	return false
}

// Note represents a single note file in the store.
type Note struct {
	RelPath  string // relative path from store root, e.g. "2026/01/20260106_8823.md"
	Date     string // date as Y...YMMDD, e.g. "20260106"
	ID       string // "8823"
	Slug     string // descriptive slug, e.g. "api-redesign", or ""
	Type     string // known type from file extension, e.g. "todo", "backlog", or ""
	BaseName string // filename without extensions, e.g. "20260106_8823" or "20260102_8814_standup"
}

// ParseFilename parses a note base filename (without .md extension) into its components.
// Expected format: Y...YMMDD_ID[_slug][.TYPE], where MM and DD are zero-padded.
// If the base name ends with a known type suffix (e.g. ".todo"), it is extracted as the Type.
func ParseFilename(baseName string) (Note, error) {
	noteType := ""
	remaining := baseName

	// Check for known type as a dot-suffix, e.g. "20260102_8814.todo"
	if idx := strings.LastIndex(baseName, "."); idx >= 0 {
		suffix := baseName[idx+1:]
		if IsKnownType(suffix) {
			noteType = suffix
			remaining = baseName[:idx]
		}
	}

	parts := strings.SplitN(remaining, "_", 3)
	if len(parts) < 2 {
		return Note{}, fmt.Errorf("invalid note filename: %s", baseName)
	}

	date := parts[0]
	if len(date) < 5 || !isDigits(date) {
		return Note{}, fmt.Errorf("invalid date in filename: %s", baseName)
	}

	id := parts[1]
	if !isDigits(id) || id == "" {
		return Note{}, fmt.Errorf("invalid id in filename: %s", baseName)
	}

	slug := ""
	if len(parts) == 3 {
		slug = parts[2]
	}

	return Note{
		Date:     date,
		ID:       id,
		Slug:     slug,
		Type:     noteType,
		BaseName: remaining,
	}, nil
}

// NoteFilename generates a note filename from date, id, optional slug, and optional type.
// Type is encoded as a secondary file extension (e.g. ".todo.md").
func NoteFilename(date string, id int, slug, noteType string) string {
	base := fmt.Sprintf("%s_%d", date, id)
	if slug != "" {
		base = fmt.Sprintf("%s_%s", base, slug)
	}
	if noteType != "" {
		return base + "." + noteType + ".md"
	}
	return base + ".md"
}

// NoteDirPath returns the year/month directory path for a given date string (Y...YMMDD),
// where MM and DD are zero-padded.
func NoteDirPath(root, date string) string {
	year := date[:len(date)-4]
	month := date[len(date)-4 : len(date)-2]
	return filepath.Join(root, year, month)
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
