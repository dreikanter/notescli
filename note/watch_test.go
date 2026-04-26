package note

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchCoalesceCreateDeleteEmitsNothing(t *testing.T) {
	pending := make(map[int]Event)
	var order []int

	order = coalesceWatchEvent(pending, order, Event{Type: EventCreated, ID: 1})
	order = coalesceWatchEvent(pending, order, Event{Type: EventDeleted, ID: 1})

	assert.Empty(t, pending)
	assert.Equal(t, []int{1}, order)
}

func TestWatchCoalesceLastEventWins(t *testing.T) {
	pending := make(map[int]Event)
	var order []int

	order = coalesceWatchEvent(pending, order, Event{Type: EventCreated, ID: 1})
	order = coalesceWatchEvent(pending, order, Event{Type: EventUpdated, ID: 1})
	order = coalesceWatchEvent(pending, order, Event{Type: EventDeleted, ID: 1})

	assert.Equal(t, Event{Type: EventDeleted, ID: 1}, pending[1])
	assert.Equal(t, []int{1}, order)
}

func TestWatchCoalesceDeleteCreateIsUpdate(t *testing.T) {
	pending := make(map[int]Event)
	var order []int

	order = coalesceWatchEvent(pending, order, Event{Type: EventDeleted, ID: 1})
	order = coalesceWatchEvent(pending, order, Event{Type: EventCreated, ID: 1})

	assert.Equal(t, Event{Type: EventUpdated, ID: 1}, pending[1])
	assert.Equal(t, []int{1}, order)
}

func TestOSStoreWatchCreatedUpdatedDeleted(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	watchRoot := filepath.Join(s.Root(), "2026", "01")
	require.NoError(t, os.MkdirAll(watchRoot, StoreDirMode(s.Root())))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := s.Watch(ctx)
	require.NoError(t, err)
	defer w.Close()

	entry, err := s.Put(Entry{Meta: Meta{Slug: "one", CreatedAt: created}, Body: "body"})
	require.NoError(t, err)
	assertWatchEvent(t, w, Event{Type: EventCreated, ID: entry.ID})

	entry.Body = "changed"
	_, err = s.Put(entry)
	require.NoError(t, err)
	assertWatchEvent(t, w, Event{Type: EventUpdated, ID: entry.ID})

	require.NoError(t, s.Delete(entry.ID))
	assertWatchEvent(t, w, Event{Type: EventDeleted, ID: entry.ID})
}

func TestOSStoreWatchDebouncesSameID(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	entry, err := s.Put(Entry{Meta: Meta{Slug: "one", CreatedAt: created}, Body: "body"})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := s.Watch(ctx)
	require.NoError(t, err)
	defer w.Close()

	entry.Body = "changed once"
	_, err = s.Put(entry)
	require.NoError(t, err)
	entry.Body = "changed twice"
	_, err = s.Put(entry)
	require.NoError(t, err)

	assertWatchEvent(t, w, Event{Type: EventUpdated, ID: entry.ID})
	assertNoWatchEvent(t, w, 150*time.Millisecond)
}

func TestOSStoreWatchCloseWhilePending(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	watchRoot := filepath.Join(s.Root(), "2026", "01")
	require.NoError(t, os.MkdirAll(watchRoot, StoreDirMode(s.Root())))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := s.Watch(ctx)
	require.NoError(t, err)

	_, err = s.Put(Entry{Meta: Meta{Slug: "one", CreatedAt: created}, Body: "body"})
	require.NoError(t, err)
	require.NoError(t, w.Close())

	select {
	case _, ok := <-w.Events():
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("watcher did not close")
	}
}

func assertWatchEvent(t *testing.T, w Watcher, want Event) {
	t.Helper()
	select {
	case got, ok := <-w.Events():
		require.True(t, ok, "watcher closed before event")
		assert.Equal(t, want, got)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for event %#v", want)
	}
}

func assertNoWatchEvent(t *testing.T, w Watcher, d time.Duration) {
	t.Helper()
	select {
	case got, ok := <-w.Events():
		require.True(t, ok, "watcher closed")
		t.Fatalf("unexpected event %#v", got)
	case <-time.After(d):
	}
}
