package note

import (
	"fmt"
	"path/filepath"
	"strings"
)

// TypesWithSpecialBehavior lists note types that trigger notes-cli-specific
// handling (e.g., daily rollover, weekly review conventions). Any string is a
// valid `type` value; this list is a soft registry, not a validation gate.
var TypesWithSpecialBehavior = []string{"todo", "backlog", "weekly"}

// HasSpecialBehavior reports whether s is a type with special notes-cli behavior.
func HasSpecialBehavior(s string) bool {
	for _, t := range TypesWithSpecialBehavior {
		if s == t {
			return true
		}
	}
	return false
}

// Note represents a single note file in the store.
type Note struct {
	RelPath string // relative path from store root, e.g. "2026/01/20260106_8823.md"
	Date    string // date as Y...YMMDD, e.g. "20260106"
	ID      string // "8823"
	Slug    string // descriptive slug, e.g. "api-redesign", or ""
	Type    string // type reported by the filename dot-suffix; any string accepted. Frontmatter type is canonical when available.
}

// isFilenameCacheSafeType reports whether a note type can round-trip through
// the filename-suffix cache. Values containing '.', '/', or '\' cannot —
// ParseFilename would mis-split them — so we omit them from the filename
// entirely and rely on frontmatter as canonical.
func isFilenameCacheSafeType(noteType string) bool {
	return noteType != "" && !strings.ContainsAny(noteType, `./\`)
}

// ParseFilename parses a note base filename (without .md extension) into its components.
// Expected format: Y...YMMDD_ID[_slug][.TYPE], where MM and DD are zero-padded.
// The dot-suffix is extracted as the filename-reported Type only when it round-
// trips cleanly (see isFilenameCacheSafeType). Frontmatter `type` is canonical.
func ParseFilename(baseName string) (Note, error) {
	noteType := ""
	remaining := baseName

	// Only treat the dot-suffix as a type if the remaining base is itself
	// dot-free — i.e. the suffix round-trips through Filename. Otherwise
	// leave Type empty and let the caller rely on frontmatter.
	if idx := strings.LastIndex(baseName, "."); idx >= 0 {
		suffix := baseName[idx+1:]
		prefix := baseName[:idx]
		if isFilenameCacheSafeType(suffix) && !strings.Contains(prefix, ".") {
			noteType = suffix
			remaining = prefix
		}
	}

	parts := strings.SplitN(remaining, "_", 3)
	if len(parts) < 2 {
		return Note{}, fmt.Errorf("invalid note filename: %s", baseName)
	}

	date := parts[0]
	if len(date) < 5 || !IsDigits(date) {
		return Note{}, fmt.Errorf("invalid date in filename: %s", baseName)
	}

	id := parts[1]
	if !IsID(id) {
		return Note{}, fmt.Errorf("invalid id in filename: %s", baseName)
	}

	slug := ""
	if len(parts) == 3 {
		slug = parts[2]
	}

	return Note{
		Date: date,
		ID:   id,
		Slug: slug,
		Type: noteType,
	}, nil
}

// Filename generates a note filename from date, id, optional slug, and optional type.
// Type is encoded as a secondary file extension (e.g. ".todo.md") only when it's
// safe to round-trip through ParseFilename; values with '.' or path separators
// are omitted from the filename, with frontmatter remaining canonical.
func Filename(date string, id int, slug, noteType string) string {
	base := fmt.Sprintf("%s_%d", date, id)
	if slug != "" {
		base = fmt.Sprintf("%s_%s", base, slug)
	}
	if isFilenameCacheSafeType(noteType) {
		return base + "." + noteType + ".md"
	}
	return base + ".md"
}

// DirPath returns the year/month directory path for a given date string (Y...YMMDD),
// where MM and DD are zero-padded.
func DirPath(root, date string) string {
	year := date[:len(date)-4]
	month := date[len(date)-4 : len(date)-2]
	return filepath.Join(root, year, month)
}

// IsID reports whether s is a valid notes-cli note ID: a non-empty string
// consisting only of ASCII digits. Downstream tools use this to detect
// numeric ID references (e.g. wikilinks, CLI query arguments) without
// re-implementing the predicate.
func IsID(s string) bool {
	return IsDigits(s)
}

// IsDigits reports whether s is non-empty and every rune is an ASCII digit.
// Use this when the caller cares about the digit-only shape of s (e.g.
// YYYY/MM path segments), not about whether s is a valid note ID.
func IsDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
