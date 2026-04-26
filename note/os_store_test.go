package note

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOSTestStore returns an OSStore rooted at a fresh t.TempDir with an
// initialised id.json (last_id: 0, so NextID starts at 1).
func newOSTestStore(t *testing.T) *OSStore {
	t.Helper()
	dir := t.TempDir()
	data, _ := json.Marshal(map[string]int{"last_id": 0})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "id.json"), data, 0o644))
	return NewOSStore(dir)
}

func TestOSStore_SatisfiesInterface(t *testing.T) {
	var _ Store = (*OSStore)(nil)
}

func TestOSStore_IDsEmpty(t *testing.T) {
	s := newOSTestStore(t)
	ids, err := s.IDs()
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestOSStore_IDsOrderIntegerIDNotLexicographic(t *testing.T) {
	s := newOSTestStore(t)

	today := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// 11 entries on the same day → IDs 1..11. Lexicographic order would
	// sort 9 before 10/11; the integer-ID sort must put 11 first.
	for i := 0; i < 11; i++ {
		_, err := s.Put(Entry{Meta: Meta{Title: "", CreatedAt: today}, Body: "x"})
		require.NoError(t, err)
	}

	ids, err := s.IDs()
	require.NoError(t, err)
	require.Len(t, ids, 11)
	assert.Equal(t, []int{11, 10, 9}, ids[:3])
}

func TestOSStore_PutNewCreatesFile(t *testing.T) {
	s := newOSTestStore(t)

	created := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	entry, err := s.Put(Entry{
		Meta: Meta{Title: "hello", Slug: "hi", CreatedAt: created},
		Body: "body text\n",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, entry.ID)
	assert.False(t, entry.Meta.UpdatedAt.IsZero())

	expected := filepath.Join(s.Root(), "2026", "01", "20260115_1_hi.md")
	assertFileExists(t, expected)
}

func TestOSStore_PutSlugChangeRenames(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

	entry, err := s.Put(Entry{Meta: Meta{Slug: "old", CreatedAt: created}, Body: "b"})
	require.NoError(t, err)
	oldPath := filepath.Join(s.Root(), "2026", "01", "20260115_1_old.md")

	entry.Meta.Slug = "new"
	_, err = s.Put(entry)
	require.NoError(t, err)
	newPath := filepath.Join(s.Root(), "2026", "01", "20260115_1_new.md")
	assertFileExists(t, newPath)
	assertNoFile(t, oldPath)
}

func TestOSStore_PutDateChangeMovesToNewSubdir(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

	entry, err := s.Put(Entry{Meta: Meta{Slug: "x", CreatedAt: created}, Body: "b"})
	require.NoError(t, err)
	oldPath := filepath.Join(s.Root(), "2026", "01", "20260115_1_x.md")

	entry.Meta.CreatedAt = time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	_, err = s.Put(entry)
	require.NoError(t, err)
	newPath := filepath.Join(s.Root(), "2026", "03", "20260320_1_x.md")
	assertFileExists(t, newPath)
	assertNoFile(t, oldPath)
}

func TestOSStore_Get(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	_, err := s.Put(Entry{Meta: Meta{Title: "t", CreatedAt: created}, Body: "body"})
	require.NoError(t, err)

	got, err := s.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "t", got.Meta.Title)
	assert.Equal(t, "body", got.Body)

	_, err = s.Get(99)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestOSStore_AllFilterByTagIncludesBodyHashtags(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// entry 1: frontmatter tag alpha
	_, err := s.Put(Entry{Meta: Meta{Tags: []string{"alpha"}, CreatedAt: created}, Body: "x"})
	require.NoError(t, err)
	// entry 2: body-hashtag beta
	_, err = s.Put(Entry{Meta: Meta{CreatedAt: created}, Body: "#beta is a body hashtag"})
	require.NoError(t, err)
	// entry 3: neither tag
	_, err = s.Put(Entry{Meta: Meta{CreatedAt: created}, Body: "nothing"})
	require.NoError(t, err)

	gotAlpha, err := s.All(WithTag("alpha"))
	require.NoError(t, err)
	assertEntryIDs(t, []int{1}, gotAlpha)

	gotBeta, err := s.All(WithTag("beta"))
	require.NoError(t, err)
	assertEntryIDs(t, []int{2}, gotBeta)
}

func TestOSStore_FindStopsAtFirstMatch(t *testing.T) {
	s := newOSTestStore(t)
	// Three todo entries across different days; newest first.
	for i := 1; i <= 3; i++ {
		day := time.Date(2026, 1, i, 0, 0, 0, 0, time.UTC)
		_, err := s.Put(Entry{Meta: Meta{Type: "todo", CreatedAt: day}, Body: ""})
		require.NoError(t, err)
	}

	got, err := s.Find(WithType("todo"))
	require.NoError(t, err)
	assert.Equal(t, 3, got.ID)
}

func TestOSStore_FindNoMatch(t *testing.T) {
	s := newOSTestStore(t)
	_, err := s.Find(WithType("todo"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestOSStore_Delete(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	entry, err := s.Put(Entry{Meta: Meta{Slug: "x", CreatedAt: created}, Body: "b"})
	require.NoError(t, err)

	require.NoError(t, s.Delete(entry.ID))
	assertNoFile(t, s.AbsPath(entry))
	assert.ErrorIs(t, s.Delete(entry.ID), ErrNotFound)
}

func TestOSStore_AbsPathNoIO(t *testing.T) {
	root := t.TempDir()
	s := NewOSStore(root)

	entry := Entry{
		ID: 42,
		Meta: Meta{
			Slug:      "demo",
			CreatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	want := filepath.Join(root, "2026", "02", "20260201_42_demo.md")
	assert.Equal(t, want, s.AbsPath(entry))
	// no file should have been created as a side effect
	assertNoFile(t, want)
}

func TestOSStore_RoundTripPreservesFrontmatterAndTags(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	entry, err := s.Put(Entry{
		Meta: Meta{
			Title:       "Test",
			Slug:        "test",
			Type:        "note",
			CreatedAt:   created,
			Tags:        []string{"alpha", "beta"},
			Aliases:     []string{"t"},
			Description: "d",
		},
		Body: "body with #gamma\n",
	})
	require.NoError(t, err)

	got, err := s.Get(entry.ID)
	require.NoError(t, err)

	assert.Equal(t, "Test", got.Meta.Title)
	assert.Equal(t, "test", got.Meta.Slug)
	assert.Equal(t, "note", got.Meta.Type)
	assert.True(t, got.Meta.CreatedAt.Equal(created))
	// Tags should include both frontmatter values and the body hashtag.
	want := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for _, tag := range got.Meta.Tags {
		delete(want, tag)
	}
	assert.Empty(t, want)
}

func TestOSStore_AllFilterByPublic(t *testing.T) {
	s := newOSTestStore(t)

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	_, err := s.Put(Entry{Meta: Meta{Title: "pub", Public: true, CreatedAt: base}, Body: "p\n"})
	require.NoError(t, err)
	_, err = s.Put(Entry{Meta: Meta{Title: "priv-explicit", Public: false, CreatedAt: base.Add(24 * time.Hour)}, Body: "x\n"})
	require.NoError(t, err)
	_, err = s.Put(Entry{Meta: Meta{Title: "pub2", Public: true, CreatedAt: base.Add(48 * time.Hour)}, Body: "y\n"})
	require.NoError(t, err)

	pub, err := s.All(WithPublic(true))
	require.NoError(t, err)
	assertEntryIDs(t, []int{3, 1}, pub)
	for _, e := range pub {
		assert.True(t, e.Meta.Public)
	}

	priv, err := s.All(WithPublic(false))
	require.NoError(t, err)
	require.Len(t, priv, 1)
	assert.Equal(t, 2, priv[0].ID)
	assert.False(t, priv[0].Meta.Public)
}

func TestOSStore_ReconcileUnchangedSkipsFileReadAndParse(t *testing.T) {
	s := newOSTestStore(t)
	entry, err := s.Put(Entry{Meta: Meta{CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, Body: "body"})
	require.NoError(t, err)
	path := s.AbsPath(entry)
	modTime := fileModTime(t, path)

	// If Reconcile reads this file despite the matching mtime, the malformed
	// frontmatter will fail the test. Reset the mtime to simulate an unchanged
	// cache key.
	require.NoError(t, os.WriteFile(path, []byte("---\n: bad\n---\nbody"), 0o644))
	require.NoError(t, os.Chtimes(path, modTime, modTime))

	diff, err := s.Reconcile(map[int]time.Time{entry.ID: modTime.In(time.FixedZone("offset", 3600))})
	require.NoError(t, err)
	assert.Empty(t, diff.Added)
	assert.Empty(t, diff.Updated)
	assert.Empty(t, diff.Removed)
}

func TestOSStore_ReconcileUpdatedAddedAndRemoved(t *testing.T) {
	s := newOSTestStore(t)
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated, err := s.Put(Entry{Meta: Meta{CreatedAt: created}, Body: "old"})
	require.NoError(t, err)
	oldModTime := fileModTime(t, s.AbsPath(updated))

	added, err := s.Put(Entry{Meta: Meta{CreatedAt: created.Add(24 * time.Hour)}, Body: "new"})
	require.NoError(t, err)

	updatedPath := s.AbsPath(updated)
	require.NoError(t, os.WriteFile(updatedPath, []byte("changed"), 0o644))
	newModTime := oldModTime.Add(2 * time.Second)
	require.NoError(t, os.Chtimes(updatedPath, newModTime, newModTime))

	diff, err := s.Reconcile(map[int]time.Time{updated.ID: oldModTime, 99: oldModTime})
	require.NoError(t, err)
	assertEntryIDsUnordered(t, []int{added.ID}, diff.Added)
	assertEntryIDsUnordered(t, []int{updated.ID}, diff.Updated)
	assert.Equal(t, "changed", diff.Updated[0].Body)
	assert.Equal(t, []int{99}, diff.Removed)
}

func TestOSStore_ReconcileMTimeSetBackwardsStillDetected(t *testing.T) {
	s := newOSTestStore(t)
	entry, err := s.Put(Entry{Meta: Meta{CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, Body: "body"})
	require.NoError(t, err)
	path := s.AbsPath(entry)
	current := fileModTime(t, path)
	backwards := current.Add(-time.Hour)
	require.NoError(t, os.Chtimes(path, backwards, backwards))

	diff, err := s.Reconcile(map[int]time.Time{entry.ID: current})
	require.NoError(t, err)
	assertEntryIDsUnordered(t, []int{entry.ID}, diff.Updated)
}

func TestOSStore_ReconcileSkipsFileWithNoParseableID(t *testing.T) {
	s := newOSTestStore(t)
	dir := filepath.Join(s.Root(), "2026", "01")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "not-a-note.md"), []byte("body"), 0o644))

	diff, err := s.Reconcile(nil)
	require.NoError(t, err)
	assert.Empty(t, diff.Added)
	assert.Empty(t, diff.Updated)
	assert.Empty(t, diff.Removed)
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.NoError(t, err)
}

func assertEntryIDsUnordered(t *testing.T, want []int, entries []Entry) {
	t.Helper()
	got := make([]int, len(entries))
	for i, e := range entries {
		got[i] = e.ID
	}
	assert.ElementsMatch(t, want, got)
}

func fileModTime(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info.ModTime()
}

func assertNoFile(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.True(t, errors.Is(err, fs.ErrNotExist), "expected %s not to exist, got err=%v", path, err)
}
