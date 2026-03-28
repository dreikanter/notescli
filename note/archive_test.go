package note

import (
	"path/filepath"
	"testing"
)

func testdataPath(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../testdata")
	if err != nil {
		t.Fatalf("cannot resolve testdata path: %v", err)
	}
	return abs
}

func TestScan(t *testing.T) {
	root := testdataPath(t)
	notes, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan(%q) error: %v", root, err)
	}

	if len(notes) != 4 {
		t.Fatalf("Scan returned %d notes, want 4", len(notes))
	}

	// Should be sorted newest first (descending RelPath)
	if notes[0].ID != "8823" {
		t.Errorf("notes[0].ID = %q, want 8823 (newest)", notes[0].ID)
	}
	if notes[1].ID != "8818" {
		t.Errorf("notes[1].ID = %q, want 8818", notes[1].ID)
	}
	if notes[2].ID != "8814" {
		t.Errorf("notes[2].ID = %q, want 8814", notes[2].ID)
	}
	if notes[3].ID != "6973" {
		t.Errorf("notes[3].ID = %q, want 6973 (oldest)", notes[3].ID)
	}

	// Verify type is parsed from renamed testdata file
	if notes[2].Type != "todo" {
		t.Errorf("notes[2].Type = %q, want \"todo\"", notes[2].Type)
	}
	if notes[2].Slug != "" {
		t.Errorf("notes[2].Slug = %q, want \"\"", notes[2].Slug)
	}
}

func TestScanSkipsInvalidFiles(t *testing.T) {
	root := testdataPath(t)
	notes, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	for _, n := range notes {
		if n.BaseName == "random_file" || n.BaseName == "not-a-note" {
			t.Errorf("Scan should have skipped %q", n.BaseName)
		}
	}
}

func TestResolveRef(t *testing.T) {
	root := testdataPath(t)

	tests := []struct {
		name    string
		query   string
		wantID  string
		wantErr bool
	}{
		{"by id", "8823", "8823", false},
		{"by id todo", "8814", "8814", false},
		{"by type todo", "todo", "8814", false},
		{"by slug", "disable-letter_opener", "6973", false},
		{"by basename", "20260106_8823", "8823", false},
		{"by basename with md", "20260106_8823.md", "8823", false},
		{"by absolute path", "", "8823", false}, // set in test body
		{"not found id", "9999", "", true},
		{"not found slug", "nonexistent", "", true},
		{"path outside root", "/tmp", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.query
			if tt.name == "by absolute path" {
				query = filepath.Join(root, "2026/01/20260106_8823.md")
			}

			got, err := ResolveRef(root, query)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveRef(%q) expected error, got nil", query)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveRef(%q) unexpected error: %v", query, err)
			}
			if got.ID != tt.wantID {
				t.Errorf("ResolveRef(%q).ID = %q, want %q", query, got.ID, tt.wantID)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	notes := []Note{
		{RelPath: "2026/01/20260106_8823.md", BaseName: "20260106_8823", Type: ""},
		{RelPath: "2026/01/20260102_8814.todo.md", BaseName: "20260102_8814", Type: "todo"},
		{RelPath: "2024/12/20241203_6973_disable-letter_opener.md", BaseName: "20241203_6973_disable-letter_opener", Type: ""},
	}

	tests := []struct {
		name     string
		fragment string
		wantLen  int
	}{
		{"by id fragment", "882", 1},
		{"by type fragment", "todo", 1},
		{"by date fragment", "2026", 2},
		{"case insensitive type", "TODO", 1},
		{"no match", "zzz", 0},
		{"matches all with underscore", "_", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(notes, tt.fragment)
			if len(got) != tt.wantLen {
				t.Errorf("Filter(%q) returned %d results, want %d", tt.fragment, len(got), tt.wantLen)
			}
		})
	}
}

func TestFilterByTags(t *testing.T) {
	root := testdataPath(t)
	notes, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	tests := []struct {
		name    string
		tags    []string
		wantLen int
		wantIDs []string
	}{
		{"single shared tag", []string{"work"}, 3, []string{"8823", "8818", "8814"}},
		{"single unique tag", []string{"meeting"}, 1, []string{"8818"}},
		{"two tags AND", []string{"work", "planning"}, 1, []string{"8814"}},
		{"two tags AND meeting", []string{"work", "meeting"}, 1, []string{"8818"}},
		{"no match", []string{"nonexistent"}, 0, nil},
		{"one matching one not", []string{"work", "nonexistent"}, 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FilterByTags(notes, root, tt.tags)
			if err != nil {
				t.Fatalf("FilterByTags(%v) error: %v", tt.tags, err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("FilterByTags(%v) returned %d notes, want %d", tt.tags, len(got), tt.wantLen)
			}
			for i, wantID := range tt.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("FilterByTags(%v)[%d].ID = %q, want %q", tt.tags, i, got[i].ID, wantID)
				}
			}
		})
	}
}

func TestFilterBySlug(t *testing.T) {
	notes := []Note{
		{Slug: ""},
		{Slug: "api-redesign"},
		{Slug: "disable-letter_opener"},
	}

	got := FilterBySlug(notes, "api-redesign")
	if len(got) != 1 {
		t.Errorf("FilterBySlug(api-redesign) returned %d, want 1", len(got))
	}

	got = FilterBySlug(notes, "")
	if len(got) != 1 {
		t.Errorf("FilterBySlug('') returned %d, want 1", len(got))
	}

	got = FilterBySlug(notes, "nope")
	if len(got) != 0 {
		t.Errorf("FilterBySlug(nope) returned %d, want 0", len(got))
	}
}

func TestFilterByType(t *testing.T) {
	notes := []Note{
		{Type: ""},
		{Type: "todo"},
		{Type: "backlog"},
		{Type: "todo"},
	}

	got := FilterByType(notes, "todo")
	if len(got) != 2 {
		t.Errorf("FilterByType(todo) returned %d, want 2", len(got))
	}

	got = FilterByType(notes, "backlog")
	if len(got) != 1 {
		t.Errorf("FilterByType(backlog) returned %d, want 1", len(got))
	}

	got = FilterByType(notes, "")
	if len(got) != 1 {
		t.Errorf("FilterByType('') returned %d, want 1", len(got))
	}

	got = FilterByType(notes, "nope")
	if len(got) != 0 {
		t.Errorf("FilterByType(nope) returned %d, want 0", len(got))
	}
}
