package note

import (
	"io/fs"
	"time"
)

// Time parses Note.Date (the UID-derived YYYYMMDD prefix) to a time.Time at
// midnight UTC. It returns false when Date is not a valid YYYYMMDD value;
// values outside the canonical 8-character form (e.g. short or long years)
// are reported as malformed even though ParseFilename accepts them.
func (n Note) Time() (time.Time, bool) {
	t, err := time.ParseInLocation("20060102", n.Date, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// ResolveEntryDate picks a single canonical date for a note, returning a
// source label so callers can surface or override the choice.
//
// Priority (first match wins):
//  1. UID-derived date — Note.Date parses cleanly as YYYYMMDD ("uid").
//  2. Frontmatter date — fm.Date is non-zero ("frontmatter").
//  3. File mtime — fi is non-nil ("mtime").
//
// fi may be nil to skip the mtime fallback. When no source resolves, the
// zero time.Time and an empty source label are returned.
//
// The signature takes Note and Frontmatter explicitly until the Entry type
// from #142 lands; callers will then pass entry.Note and entry.Frontmatter.
func ResolveEntryDate(n Note, fm Frontmatter, fi fs.FileInfo) (time.Time, string) {
	if t, ok := n.Time(); ok {
		return t, "uid"
	}
	if !fm.Date.IsZero() {
		return fm.Date, "frontmatter"
	}
	if fi != nil {
		return fi.ModTime(), "mtime"
	}
	return time.Time{}, ""
}
