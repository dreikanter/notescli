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
// CreatedAt maps to the YAML frontmatter "date" field and is both read from
// and written to disk.
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
	Title       string
	Slug        string
	Type        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Tags        []string
	Aliases     []string
	Description string
	Public      bool
	Extra       map[string]any
}
