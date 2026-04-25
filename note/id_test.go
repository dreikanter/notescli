package note

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadID(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`{"last_id": 9218}`), 0o644); err != nil {
		t.Fatal(err)
	}

	idf, err := readID(dir)
	require.NoError(t, err)
	if idf.LastID != 9218 {
		t.Errorf("got LastID=%d, want 9218", idf.LastID)
	}
}

func TestReadIDMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := readID(dir)
	require.Error(t, err)
}

func TestReadIDInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`not json`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := readID(dir)
	require.Error(t, err)
}

func TestNextID(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`{"last_id": 9218}`), 0o644); err != nil {
		t.Fatal(err)
	}

	id, err := NextID(dir)
	require.NoError(t, err)
	assert.Equal(t, 9219, id)

	// Verify file was updated
	idf, err := readID(dir)
	if err != nil {
		t.Fatalf("unexpected error reading back: %v", err)
	}
	if idf.LastID != 9219 {
		t.Errorf("got LastID=%d after write, want 9219", idf.LastID)
	}
}

func TestNextIDConsecutive(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`{"last_id": 100}`), 0o644); err != nil {
		t.Fatal(err)
	}

	id1, _ := NextID(dir)
	id2, _ := NextID(dir)

	assert.Equal(t, 101, id1)
	assert.Equal(t, 102, id2)
}

func TestNextIDConcurrent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`{"last_id": 0}`), 0o644); err != nil {
		t.Fatal(err)
	}

	const n = 32
	ids := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, err := NextID(dir)
			if err != nil {
				t.Errorf("NextID failed: %v", err)
				return
			}
			ids[i] = id
		}(i)
	}
	wg.Wait()

	sort.Ints(ids)
	for i, id := range ids {
		if id != i+1 {
			t.Fatalf("ids not contiguous after concurrent NextID calls: %v", ids)
		}
	}

	// The dir should contain only id.json — no lockfile or temp files.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "id.json" {
			t.Errorf("unexpected file left behind: %s", e.Name())
		}
	}
}

func TestWriteIDAtomic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`{"last_id": 0}`), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeID(dir, idFile{LastID: 42})
	require.NoError(t, err)

	// No temp files should remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "id.json" {
			t.Errorf("unexpected file left behind: %s", e.Name())
		}
	}
}
