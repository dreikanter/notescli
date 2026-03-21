package note

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadID(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`{"last_id": 9218}`), 0o644); err != nil {
		t.Fatal(err)
	}

	idf, err := ReadID(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idf.LastID != 9218 {
		t.Errorf("got LastID=%d, want 9218", idf.LastID)
	}
}

func TestReadIDMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadID(dir)
	if err == nil {
		t.Fatal("expected error for missing id.json")
	}
}

func TestReadIDInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`not json`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadID(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestNextID(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`{"last_id": 9218}`), 0o644); err != nil {
		t.Fatal(err)
	}

	id, err := NextID(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 9219 {
		t.Errorf("got id=%d, want 9219", id)
	}

	// Verify file was updated
	idf, err := ReadID(dir)
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

	if id1 != 101 {
		t.Errorf("first call: got %d, want 101", id1)
	}
	if id2 != 102 {
		t.Errorf("second call: got %d, want 102", id2)
	}
}

func TestWriteIDAtomic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(`{"last_id": 0}`), 0o644); err != nil {
		t.Fatal(err)
	}

	err := WriteID(dir, IDFile{LastID: 42})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No temp files should remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "id.json" {
			t.Errorf("unexpected file left behind: %s", e.Name())
		}
	}
}
