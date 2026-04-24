package note

import "errors"

// ErrNotFound is the package-wide "entry not found" sentinel. It is returned
// (wrapped) by Store.Get, Store.Find, and Store.Delete when no entry matches.
// Callers match with errors.Is:
//
//	if errors.Is(err, note.ErrNotFound) { … }
var ErrNotFound = errors.New("entry not found")

// Store is the backend abstraction the note package exposes. Implementations
// encapsulate the storage substrate (filesystem, in-memory, future cloud/DB)
// so CLI commands can target a single interface.
//
// Error contract for lookups:
//   - Get, Find, and Delete return a wrapped ErrNotFound when no entry
//     matches. Callers check with errors.Is(err, note.ErrNotFound).
//   - All returns an empty slice with a nil error when no entry matches;
//     zero results are not considered an error.
type Store interface {
	// IDs returns the IDs of every entry newest-first by Meta.CreatedAt.
	// Backends that can answer from a directory scan must not read file
	// contents. Returns an empty slice (nil error) when the store is empty.
	IDs() ([]int, error)

	// All returns every entry matching opts, newest-first by Meta.CreatedAt.
	// Returned entries are fully populated, including Meta.Tags merged from
	// frontmatter tags and body hashtags. Zero matches returns an empty
	// slice with a nil error.
	All(opts ...QueryOpt) ([]Entry, error)

	// Find returns the newest entry matching opts. Returns ErrNotFound when
	// no entry matches. Backends may terminate the scan after the first
	// match.
	Find(opts ...QueryOpt) (Entry, error)

	// Get returns the entry with the given ID, or ErrNotFound if no entry
	// has that ID.
	Get(id int) (Entry, error)

	// Put writes entry. When entry.ID is zero the store assigns a fresh ID
	// and defaults Meta.CreatedAt to time.Now if zero; otherwise Put performs
	// a full replace of the existing entry and requires Meta.CreatedAt to be
	// non-zero (returning an error otherwise). Meta.UpdatedAt is always set
	// to time.Now on write. Returns the stored entry with all store-assigned
	// fields populated.
	Put(entry Entry) (Entry, error)

	// Delete removes the entry with the given ID. Returns ErrNotFound when
	// no entry has that ID.
	Delete(id int) error
}
