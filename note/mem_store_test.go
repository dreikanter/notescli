package note

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStore_IDsEmpty(t *testing.T) {
	s := NewMemStore()
	ids, err := s.IDs()
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestMemStore_IDsOrderNewestFirstThenIDDesc(t *testing.T) {
	s := NewMemStore()
	day1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)

	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day1}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{CreatedAt: day1}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{CreatedAt: day2}})

	ids, err := s.IDs()
	require.NoError(t, err)
	assert.Equal(t, []int{3, 2, 1}, ids)
}

func TestMemStore_AllNoOpts(t *testing.T) {
	s := NewMemStore()
	for i := 1; i <= 3; i++ {
		mustPut(t, s, Entry{ID: i, Meta: Meta{CreatedAt: time.Date(2026, 1, i, 0, 0, 0, 0, time.UTC)}})
	}
	entries, err := s.All()
	require.NoError(t, err)
	assertEntryIDs(t, []int{3, 2, 1}, entries)
}

func TestMemStore_AllFilterByType(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Type: "todo", CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Type: "note", CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{Type: "todo", CreatedAt: day(2026, 1, 3)}})

	got, err := s.All(WithType("todo"))
	require.NoError(t, err)
	assertEntryIDs(t, []int{3, 1}, got)
}

func TestMemStore_AllMultipleTagsAreAND(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Tags: []string{"a"}, CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Tags: []string{"a", "b"}, CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{Tags: []string{"a", "b", "c"}, CreatedAt: day(2026, 1, 3)}})

	got, err := s.All(WithTag("a"), WithTag("b"))
	require.NoError(t, err)
	assertEntryIDs(t, []int{3, 2}, got)
}

func TestMemStore_TagMatchCaseInsensitive(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Tags: []string{"Alpha"}, CreatedAt: day(2026, 1, 1)}})
	got, err := s.All(WithTag("alpha"))
	require.NoError(t, err)
	assertEntryIDs(t, []int{1}, got)
}

func TestMemStore_AllNoMatchEmptySliceNotError(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Type: "note", CreatedAt: day(2026, 1, 1)}})

	got, err := s.All(WithType("todo"))
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.NotErrorIs(t, err, ErrNotFound)
}

func TestMemStore_AllFilterByExactDate(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{CreatedAt: time.Date(2026, 1, 1, 23, 59, 0, 0, time.UTC)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{CreatedAt: day(2026, 1, 2)}})

	got, err := s.All(WithExactDate(day(2026, 1, 1)))
	require.NoError(t, err)
	assertEntryIDs(t, []int{2, 1}, got)
}

func TestMemStore_AllFilterByBeforeDate(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{CreatedAt: day(2026, 1, 3)}})

	got, err := s.All(WithBeforeDate(day(2026, 1, 3)))
	require.NoError(t, err)
	assertEntryIDs(t, []int{2, 1}, got)
}

func TestMemStore_FindReturnsNewest(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Type: "todo", CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Type: "todo", CreatedAt: day(2026, 1, 2)}})

	got, err := s.Find(WithType("todo"))
	require.NoError(t, err)
	assert.Equal(t, 2, got.ID)
}

func TestMemStore_FindNoMatchErrNotFound(t *testing.T) {
	s := NewMemStore()
	_, err := s.Find(WithType("todo"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMemStore_Get(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Title: "one", CreatedAt: day(2026, 1, 1)}})

	got, err := s.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "one", got.Meta.Title)

	_, err = s.Get(99)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMemStore_PutAssignsIDStartingAt1(t *testing.T) {
	s := NewMemStore()
	e, err := s.Put(Entry{Body: "hello"})
	require.NoError(t, err)
	assert.Equal(t, 1, e.ID)
}

func TestMemStore_PutAssignsMaxPlusOne(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 5, Meta: Meta{CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{CreatedAt: day(2026, 1, 1)}})

	e, err := s.Put(Entry{Body: "new"})
	require.NoError(t, err)
	assert.Equal(t, 6, e.ID)
}

func TestMemStore_PutExistingIDReplaces(t *testing.T) {
	s := NewMemStore()
	created := day(2026, 1, 1)
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Title: "old", CreatedAt: created}, Body: "old body"})

	e, err := s.Put(Entry{ID: 1, Meta: Meta{Title: "new", CreatedAt: created}, Body: "new body"})
	require.NoError(t, err)
	assert.Equal(t, "new", e.Meta.Title)
	assert.Equal(t, "new body", e.Body)
	got, err := s.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "new", got.Meta.Title)
}

func TestMemStore_PutUpdateZeroCreatedAtErrors(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day(2026, 1, 1)}})

	_, err := s.Put(Entry{ID: 1, Meta: Meta{Title: "x"}})
	require.Error(t, err)
}

func TestMemStore_PutZeroCreatedAtSetsToNow(t *testing.T) {
	s := NewMemStore()
	before := time.Now()
	e, err := s.Put(Entry{Body: "hi"})
	require.NoError(t, err)
	after := time.Now()

	assert.False(t, e.Meta.CreatedAt.Before(before))
	assert.False(t, e.Meta.CreatedAt.After(after))
}

func TestMemStore_PutAlwaysSetsUpdatedAt(t *testing.T) {
	s := NewMemStore()
	originalCreated := day(2026, 1, 1)
	e, err := s.Put(Entry{ID: 1, Meta: Meta{CreatedAt: originalCreated}})
	require.NoError(t, err)
	assert.True(t, e.Meta.CreatedAt.Equal(originalCreated))
	assert.False(t, e.Meta.UpdatedAt.IsZero())

	time.Sleep(time.Millisecond)
	e2, err := s.Put(Entry{ID: 1, Meta: Meta{CreatedAt: originalCreated}})
	require.NoError(t, err)
	assert.True(t, e2.Meta.UpdatedAt.After(e.Meta.UpdatedAt))
}

func TestMemStore_Delete(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{CreatedAt: day(2026, 1, 1)}})

	require.NoError(t, s.Delete(1))
	_, err := s.Get(1)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.ErrorIs(t, s.Delete(1), ErrNotFound)
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

func TestMemStore_AllFilterByPublic(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Public: true, CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Public: false, CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{Public: true, CreatedAt: day(2026, 1, 3)}})

	pub, err := s.All(WithPublic(true))
	require.NoError(t, err)
	assertEntryIDs(t, []int{3, 1}, pub)

	priv, err := s.All(WithPublic(false))
	require.NoError(t, err)
	assertEntryIDs(t, []int{2}, priv)
}

func TestMemStore_AllPublicComposesWithTag(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Public: true, Tags: []string{"x"}, CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Public: true, Tags: []string{"y"}, CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{Public: false, Tags: []string{"x"}, CreatedAt: day(2026, 1, 3)}})

	got, err := s.All(WithPublic(true), WithTag("x"))
	require.NoError(t, err)
	assertEntryIDs(t, []int{1}, got)
}

func mustPut(t *testing.T, s *MemStore, e Entry) Entry {
	t.Helper()
	out, err := s.Put(e)
	require.NoError(t, err)
	return out
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func assertEntryIDs(t *testing.T, want []int, entries []Entry) {
	t.Helper()
	assert.Equal(t, want, entryIDs(entries))
}

func entryIDs(entries []Entry) []int {
	out := make([]int, len(entries))
	for i, e := range entries {
		out[i] = e.ID
	}
	return out
}
