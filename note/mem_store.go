package note

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemStore is an in-memory Store backed by map[int]Entry with an
// RWMutex. It is never user-facing — it exists to validate the Store
// interface shape and to serve as the test double for command tests.
//
// MemStore skips the YAML and body-hashtag machinery that OSStore performs
// on read: Meta.Tags is exactly whatever the caller stored. Tag matching is
// case-insensitive.
type MemStore struct {
	mu      sync.RWMutex
	entries map[int]Entry
}

var _ Store = (*MemStore)(nil)

// NewMemStore returns an empty MemStore.
func NewMemStore() *MemStore {
	return &MemStore{entries: make(map[int]Entry)}
}

// IDs returns every stored ID newest-first by Meta.CreatedAt. Ties within
// the same timestamp break by higher ID first so the order is total and
// deterministic.
func (s *MemStore) IDs() ([]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]int, 0, len(s.entries))
	for id := range s.entries {
		ids = append(ids, id)
	}
	s.sortIDsLocked(ids)
	return ids, nil
}

// All returns every entry matching opts, newest-first by Meta.CreatedAt.
// See Store.All for the error contract: zero matches yield an empty slice
// and a nil error.
func (s *MemStore) All(opts ...QueryOpt) ([]Entry, error) {
	q := buildQuery(opts)

	s.mu.RLock()
	defer s.mu.RUnlock()

	matches := s.matchLocked(q)
	s.sortEntriesByRecency(matches)
	return matches, nil
}

// Find returns the newest entry matching opts, or ErrNotFound when no entry
// matches.
func (s *MemStore) Find(opts ...QueryOpt) (Entry, error) {
	q := buildQuery(opts)

	s.mu.RLock()
	defer s.mu.RUnlock()

	matches := s.matchLocked(q)
	if len(matches) == 0 {
		return Entry{}, ErrNotFound
	}
	s.sortEntriesByRecency(matches)
	return matches[0], nil
}

// Get returns the entry with the given ID, or ErrNotFound.
func (s *MemStore) Get(id int) (Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.entries[id]
	if !ok {
		return Entry{}, fmt.Errorf("%w: %d", ErrNotFound, id)
	}
	return e, nil
}

// Put stores entry. When entry.ID is zero a new ID is assigned as
// max(existing IDs) + 1 (1 for an empty store) and Meta.CreatedAt is
// defaulted to time.Now if zero. Updates (entry.ID != 0) must carry a
// non-zero Meta.CreatedAt. Meta.UpdatedAt is always set to time.Now.
func (s *MemStore) Put(entry Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if entry.ID == 0 {
		entry.ID = s.nextIDLocked()
		if entry.Meta.CreatedAt.IsZero() {
			entry.Meta.CreatedAt = now
		}
	}
	if entry.Meta.CreatedAt.IsZero() {
		return Entry{}, fmt.Errorf("note %d: CreatedAt is zero", entry.ID)
	}
	entry.Meta.UpdatedAt = now
	s.entries[entry.ID] = entry
	return entry, nil
}

// Delete removes the entry with the given ID, or returns ErrNotFound.
func (s *MemStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[id]; !ok {
		return fmt.Errorf("%w: %d", ErrNotFound, id)
	}
	delete(s.entries, id)
	return nil
}

// Reconcile returns the delta between known and the current in-memory state.
// known maps entry ID to the UpdatedAt timestamp the caller last observed.
func (s *MemStore) Reconcile(known map[int]time.Time) (Diff, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[int]struct{}, len(s.entries))
	var diff Diff
	for id, entry := range s.entries {
		seen[id] = struct{}{}
		knownTime, ok := known[id]
		if !ok {
			diff.Added = append(diff.Added, entry)
			continue
		}
		if !knownTime.Equal(entry.Meta.UpdatedAt) {
			diff.Updated = append(diff.Updated, entry)
		}
	}
	for id := range known {
		if _, ok := seen[id]; !ok {
			diff.Removed = append(diff.Removed, id)
		}
	}
	s.sortEntriesByRecency(diff.Added)
	s.sortEntriesByRecency(diff.Updated)
	sort.Ints(diff.Removed)
	return diff, nil
}

// matchLocked returns every entry that matches q. Caller holds s.mu for
// read (RLock or Lock).
func (s *MemStore) matchLocked(q query) []Entry {
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		if matches(e, q) {
			out = append(out, e)
		}
	}
	return out
}

// nextIDLocked returns max(existing IDs) + 1, or 1 when empty. Caller holds
// s.mu for write.
func (s *MemStore) nextIDLocked() int {
	highest := 0
	for id := range s.entries {
		if id > highest {
			highest = id
		}
	}
	return highest + 1
}

// sortIDsLocked sorts ids newest-first by the entries' CreatedAt, tie-breaking
// on higher ID first. Caller holds s.mu for read.
func (s *MemStore) sortIDsLocked(ids []int) {
	sort.Slice(ids, func(i, j int) bool {
		ei, ej := s.entries[ids[i]], s.entries[ids[j]]
		if !ei.Meta.CreatedAt.Equal(ej.Meta.CreatedAt) {
			return ei.Meta.CreatedAt.After(ej.Meta.CreatedAt)
		}
		return ids[i] > ids[j]
	})
}

// sortEntriesByRecency sorts entries newest-first by CreatedAt with the same
// tie-break as sortIDsLocked.
func (s *MemStore) sortEntriesByRecency(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].Meta.CreatedAt.Equal(entries[j].Meta.CreatedAt) {
			return entries[i].Meta.CreatedAt.After(entries[j].Meta.CreatedAt)
		}
		return entries[i].ID > entries[j].ID
	})
}
