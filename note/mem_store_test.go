package note

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMemStore_IDsEmpty(t *testing.T) {
	s := NewMemStore()
	ids, err := s.IDs()
	if err != nil {
		t.Fatalf("IDs: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("IDs on empty store = %v, want empty", ids)
	}
}

func TestMemStore_IDsOrderNewestFirstThenIDDesc(t *testing.T) {
	s := NewMemStore()
	day1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)

	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day1}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{CreatedAt: day1}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{CreatedAt: day2}})

	ids, err := s.IDs()
	if err != nil {
		t.Fatalf("IDs: %v", err)
	}
	want := []int{3, 2, 1}
	if !sliceEqual(ids, want) {
		t.Fatalf("IDs = %v, want %v", ids, want)
	}
}

func TestMemStore_AllNoOpts(t *testing.T) {
	s := NewMemStore()
	for i := 1; i <= 3; i++ {
		mustPut(t, s, Entry{ID: i, Meta: Meta{CreatedAt: time.Date(2026, 1, i, 0, 0, 0, 0, time.UTC)}})
	}
	entries, err := s.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("All len = %d, want 3", len(entries))
	}
	if entries[0].ID != 3 || entries[2].ID != 1 {
		t.Fatalf("All order = %v, want [3 2 1]", entryIDs(entries))
	}
}

func TestMemStore_AllFilterByType(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Type: "todo", CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Type: "note", CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{Type: "todo", CreatedAt: day(2026, 1, 3)}})

	got, err := s.All(WithType("todo"))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(got) != 2 || got[0].ID != 3 || got[1].ID != 1 {
		t.Fatalf("All WithType(todo) = %v, want [3 1]", entryIDs(got))
	}
}

func TestMemStore_AllMultipleTagsAreAND(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Tags: []string{"a"}, CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Tags: []string{"a", "b"}, CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{Tags: []string{"a", "b", "c"}, CreatedAt: day(2026, 1, 3)}})

	got, err := s.All(WithTag("a"), WithTag("b"))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(got) != 2 || got[0].ID != 3 || got[1].ID != 2 {
		t.Fatalf("All WithTag(a)+WithTag(b) = %v, want [3 2]", entryIDs(got))
	}
}

func TestMemStore_TagMatchCaseInsensitive(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Tags: []string{"Alpha"}, CreatedAt: day(2026, 1, 1)}})
	got, err := s.All(WithTag("alpha"))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("All WithTag(alpha) len = %d, want 1", len(got))
	}
}

func TestMemStore_AllNoMatchEmptySliceNotError(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Type: "note", CreatedAt: day(2026, 1, 1)}})

	got, err := s.All(WithType("todo"))
	if err != nil {
		t.Fatalf("All unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("All no-match len = %d, want 0", len(got))
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("All no-match should not wrap ErrNotFound")
	}
}

func TestMemStore_AllFilterByExactDate(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{CreatedAt: time.Date(2026, 1, 1, 23, 59, 0, 0, time.UTC)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{CreatedAt: day(2026, 1, 2)}})

	got, err := s.All(WithExactDate(day(2026, 1, 1)))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("All WithExactDate len = %d, want 2", len(got))
	}
}

func TestMemStore_AllFilterByBeforeDate(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{CreatedAt: day(2026, 1, 3)}})

	got, err := s.All(WithBeforeDate(day(2026, 1, 3)))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 1 {
		t.Fatalf("All WithBeforeDate = %v, want [2 1]", entryIDs(got))
	}
}

func TestMemStore_FindReturnsNewest(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Type: "todo", CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Type: "todo", CreatedAt: day(2026, 1, 2)}})

	got, err := s.Find(WithType("todo"))
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.ID != 2 {
		t.Fatalf("Find ID = %d, want 2", got.ID)
	}
}

func TestMemStore_FindNoMatchErrNotFound(t *testing.T) {
	s := NewMemStore()
	_, err := s.Find(WithType("todo"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Find no-match err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_Get(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Title: "one", CreatedAt: day(2026, 1, 1)}})

	got, err := s.Get(1)
	if err != nil {
		t.Fatalf("Get hit: %v", err)
	}
	if got.Meta.Title != "one" {
		t.Fatalf("Get title = %q, want one", got.Meta.Title)
	}

	_, err = s.Get(99)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get miss err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_PutAssignsIDStartingAt1(t *testing.T) {
	s := NewMemStore()
	e, err := s.Put(Entry{Body: "hello"})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if e.ID != 1 {
		t.Fatalf("first Put ID = %d, want 1", e.ID)
	}
}

func TestMemStore_PutAssignsMaxPlusOne(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 5, Meta: Meta{CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{CreatedAt: day(2026, 1, 1)}})

	e, err := s.Put(Entry{Body: "new"})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if e.ID != 6 {
		t.Fatalf("Put new ID = %d, want 6", e.ID)
	}
}

func TestMemStore_PutExistingIDReplaces(t *testing.T) {
	s := NewMemStore()
	created := day(2026, 1, 1)
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Title: "old", CreatedAt: created}, Body: "old body"})

	e, err := s.Put(Entry{ID: 1, Meta: Meta{Title: "new", CreatedAt: created}, Body: "new body"})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if e.Meta.Title != "new" || e.Body != "new body" {
		t.Fatalf("Put replace = %+v, want title=new body=new body", e)
	}
	got, _ := s.Get(1)
	if got.Meta.Title != "new" {
		t.Fatalf("Get after replace title = %q, want new", got.Meta.Title)
	}
}

func TestMemStore_PutUpdateZeroCreatedAtErrors(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day(2026, 1, 1)}})

	_, err := s.Put(Entry{ID: 1, Meta: Meta{Title: "x"}})
	if err == nil {
		t.Fatal("Put update with zero CreatedAt: expected error, got nil")
	}
}

func TestMemStore_PutZeroCreatedAtSetsToNow(t *testing.T) {
	s := NewMemStore()
	before := time.Now()
	e, err := s.Put(Entry{Body: "hi"})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	after := time.Now()

	if e.Meta.CreatedAt.Before(before) || e.Meta.CreatedAt.After(after) {
		t.Fatalf("CreatedAt = %v, want between %v and %v", e.Meta.CreatedAt, before, after)
	}
}

func TestMemStore_PutAlwaysSetsUpdatedAt(t *testing.T) {
	s := NewMemStore()
	originalCreated := day(2026, 1, 1)
	e, err := s.Put(Entry{ID: 1, Meta: Meta{CreatedAt: originalCreated}})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !e.Meta.CreatedAt.Equal(originalCreated) {
		t.Fatalf("Put changed provided CreatedAt: got %v", e.Meta.CreatedAt)
	}
	if e.Meta.UpdatedAt.IsZero() {
		t.Fatalf("Put did not set UpdatedAt")
	}

	time.Sleep(time.Millisecond)
	e2, err := s.Put(Entry{ID: 1, Meta: Meta{CreatedAt: originalCreated}})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !e2.Meta.UpdatedAt.After(e.Meta.UpdatedAt) {
		t.Fatalf("UpdatedAt did not advance: first=%v second=%v", e.Meta.UpdatedAt, e2.Meta.UpdatedAt)
	}
}

func TestMemStore_Delete(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day(2026, 1, 1)}})

	if err := s.Delete(1); err != nil {
		t.Fatalf("Delete hit: %v", err)
	}
	if _, err := s.Get(1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete = %v, want ErrNotFound", err)
	}
	if err := s.Delete(1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete miss err = %v, want ErrNotFound", err)
	}
}

func TestMemStore_ConcurrentReads(t *testing.T) {
	s := NewMemStore()
	for i := 1; i <= 20; i++ {
		mustPut(t, s, Entry{ID: i, Meta: Meta{CreatedAt: day(2026, 1, i%28+1)}})
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if _, err := s.All(); err != nil {
					t.Errorf("All: %v", err)
					return
				}
				if _, err := s.IDs(); err != nil {
					t.Errorf("IDs: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func mustPut(t *testing.T, s *MemStore, e Entry) Entry {
	t.Helper()
	out, err := s.Put(e)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	return out
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func entryIDs(entries []Entry) []int {
	out := make([]int, len(entries))
	for i, e := range entries {
		out[i] = e.ID
	}
	return out
}

func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
