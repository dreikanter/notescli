package note

import "time"

// Entry is the single domain object that Store implementations work with.
type Entry struct {
	ID   int
	Meta Meta
	Body string
}

// Meta holds the user-domain metadata for a note. YAML serialisation details
// live inside OSStore.
//
// CreatedAt is the canonical authored timestamp. It always carries a value
// in memory: read from the frontmatter "date" field when present, otherwise
// derived from the filename's UID date prefix. It only round-trips back to
// the YAML "date" field when DateExplicit is true; otherwise the field is
// omitted on write and consumers fall back to the filename per SCHEMA.md.
//
// DateExplicit reports that CreatedAt came from a user-supplied frontmatter
// "date" (on read) or a user-driven CLI flag like `update --date` (on write).
// Auto-defaulted dates (e.g. `notes new` stamping time.Now into a fresh
// entry) leave it false so the redundant frontmatter line is suppressed.
//
// UpdatedAt is derived from the file's ModTime on read and is never written
// to YAML. OSStore.Put sets it to time.Now on every write — no file re-read
// needed because the OS updates ModTime when the file is rewritten.
//
// Tags is the merged set of frontmatter "tags" and body "#hashtag" tokens;
// OSStore performs the merge on read and consumers never distinguish between
// the two sources.
//
// Extra carries unknown frontmatter keys as map[string]any; OSStore handles
// conversion to/from yaml.Node at the serialisation boundary.
type Meta struct {
	Title        string
	Slug         string
	Type         string
	CreatedAt    time.Time
	DateExplicit bool
	UpdatedAt    time.Time
	Tags         []string
	Aliases      []string
	Description  string
	Public       bool
	Extra        map[string]any
}
