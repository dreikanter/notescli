package note

import (
	"os"
	"path/filepath"
	"strings"
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
	if notes[0].Slug != "999" {
		t.Errorf("notes[0].Slug = %q, want \"999\"", notes[0].Slug)
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
		if strings.Contains(n.RelPath, "random_file") || strings.Contains(n.RelPath, "not-a-note") {
			t.Errorf("Scan should have skipped %q", n.RelPath)
		}
	}
}

// TestScanSkipsUnreadableDir verifies one unreadable month directory doesn't
// abort the whole scan — readable siblings still enumerate successfully.
func TestScanSkipsUnreadableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permission checks")
	}

	root := t.TempDir()
	good := filepath.Join(root, "2026", "01")
	if err := os.MkdirAll(good, 0o755); err != nil {
		t.Fatalf("mkdir good: %v", err)
	}
	goodNote := filepath.Join(good, "20260101_1_s.md")
	if err := os.WriteFile(goodNote, []byte("body\n"), 0o644); err != nil {
		t.Fatalf("write good: %v", err)
	}

	bad := filepath.Join(root, "2026", "02")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatalf("mkdir bad: %v", err)
	}
	if err := os.Chmod(bad, 0o000); err != nil {
		t.Fatalf("chmod bad: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(bad, 0o755) })

	notes, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan(%q) error: %v", root, err)
	}
	if len(notes) != 1 || notes[0].ID != "1" {
		t.Errorf("Scan = %+v, want 1 note with ID=1", notes)
	}
}

// TestScanStrictExplicit verifies passing WithStrict(true) matches the
// no-options default and continues to ignore non-YYYY layouts.
func TestScanStrictExplicit(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1.md", "body\n")
	writeNote(t, root, "drafts/20260102_2.md", "body\n")
	writeNote(t, root, "inbox/2026/01/20260103_3.md", "body\n")

	notes, err := Scan(root, WithStrict(true))
	if err != nil {
		t.Fatalf("Scan strict error: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != "1" {
		t.Fatalf("Scan strict = %+v, want only ID=1", notes)
	}
}

// TestScanLenient verifies Strict=false discovers any *.md note under root,
// regardless of nesting depth or parent directory naming.
func TestScanLenient(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1.md", "body\n")
	writeNote(t, root, "drafts/20260102_2_idea.md", "body\n")
	writeNote(t, root, "inbox/2026/01/20260103_3.md", "body\n")
	writeNote(t, root, "20260104_4.md", "body\n")
	writeNote(t, root, "deep/a/b/c/20260105_5_nested.md", "body\n")
	writeNote(t, root, "drafts/not-a-note.md", "body\n")
	writeNote(t, root, "drafts/random_file.txt", "body\n")

	notes, err := Scan(root, WithStrict(false))
	if err != nil {
		t.Fatalf("Scan lenient error: %v", err)
	}

	// Sort is descending lexicographic on RelPath.
	wantIDs := []string{"3", "2", "5", "4", "1"}
	if len(notes) != len(wantIDs) {
		t.Fatalf("Scan lenient returned %d notes (%+v), want %d", len(notes), notes, len(wantIDs))
	}
	for i, want := range wantIDs {
		if notes[i].ID != want {
			t.Errorf("notes[%d].ID = %q, want %q", i, notes[i].ID, want)
		}
	}

	// RelPath should be the path relative to root, preserving the discovered layout.
	for _, n := range notes {
		if filepath.IsAbs(n.RelPath) {
			t.Errorf("RelPath %q is absolute, want relative", n.RelPath)
		}
	}
}

// TestScanLenientDefaultIsStrict guards the contract that Scan(root) without
// options stays strict — only the canonical YYYY/MM/*.md layout is enumerated.
func TestScanLenientDefaultIsStrict(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1.md", "body\n")
	writeNote(t, root, "drafts/20260102_2.md", "body\n")

	notes, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan default error: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != "1" {
		t.Fatalf("default Scan = %+v, want only ID=1 (strict)", notes)
	}
}

// TestScanLenientSkipsUnreadableDir verifies one unreadable subdirectory is
// logged and skipped without aborting the lenient walk.
func TestScanLenientSkipsUnreadableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permission checks")
	}

	root := t.TempDir()
	writeNote(t, root, "drafts/20260101_1.md", "body\n")

	bad := filepath.Join(root, "locked")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatalf("mkdir bad: %v", err)
	}
	if err := os.Chmod(bad, 0o000); err != nil {
		t.Fatalf("chmod bad: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(bad, 0o755) })

	notes, err := Scan(root, WithStrict(false))
	if err != nil {
		t.Fatalf("Scan lenient error: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != "1" {
		t.Errorf("Scan lenient = %+v, want 1 note with ID=1", notes)
	}
}

func TestResolveRef(t *testing.T) {
	root := testdataPath(t)
	absPath := filepath.Join(root, "2026/01/20260106_8823_999.md")

	tests := []struct {
		name    string
		query   string
		wantID  string
		wantErr bool
	}{
		{"by id", "8823", "8823", false},
		{"by id todo", "8814", "8814", false},
		{"by type todo", "todo", "8814", false},
		{"by slug exact", "disable-letter_opener", "6973", false},
		{"by slug fragment", "letter_opener", "6973", false},
		{"by partial slug", "meeting", "8818", false},
		{"by absolute path", absPath, "8823", false},
		{"absolute path with trailing slash errors", absPath + "/", "", true},
		{"empty query returns most recent", "", "8823", false},
		{"numeric non-id errors", "999", "", true},
		{"numeric date fragment errors", "202601", "", true},
		{"basename query does not substring-match path", "20260106_8823_999", "", true},
		{"basename with md does not substring-match path", "20260106_8823_999.md", "", true},
		{"date fragment inside slug query does not match path", "20260106", "", true},
		{"not found id", "9999", "", true},
		{"not found query", "nonexistent", "", true},
		{"path outside root", "/tmp", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveRef(root, tt.query)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveRef(%q) expected error, got nil", tt.query)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveRef(%q) unexpected error: %v", tt.query, err)
			}
			if got.ID != tt.wantID {
				t.Errorf("ResolveRef(%q).ID = %q, want %q", tt.query, got.ID, tt.wantID)
			}
		})
	}
}

func TestResolveRefWithDateEmptyQueryFiltersByDate(t *testing.T) {
	root := testdataPath(t)
	got, err := ResolveRef(root, "", WithDate("20260104"))
	if err != nil {
		t.Fatalf("ResolveRef WithDate empty query error: %v", err)
	}
	if got.ID != "8818" {
		t.Errorf("ResolveRef empty + WithDate(20260104) = %q, want 8818", got.ID)
	}
}

func TestFilter(t *testing.T) {
	entries := []Entry{
		{Note: Note{RelPath: "2026/01/20260106_8823.md", Type: ""}},
		{Note: Note{RelPath: "2026/01/20260102_8814.todo.md", Type: "todo"}},
		{Note: Note{RelPath: "2024/12/20241203_6973_disable-letter_opener.md", Type: ""}},
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
			got := Filter(entries, tt.fragment)
			if len(got) != tt.wantLen {
				t.Errorf("Filter(%q) returned %d results, want %d", tt.fragment, len(got), tt.wantLen)
			}
		})
	}
}

func TestFilterByTags(t *testing.T) {
	root := testdataPath(t)
	idx, err := Load(root)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	entries := idx.Entries()

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
		{"uppercase query matches lowercase fm", []string{"WORK"}, 3, []string{"8823", "8818", "8814"}},
		{"mixed-case query matches", []string{"Meeting"}, 1, []string{"8818"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByTags(entries, tt.tags)
			if len(got) != tt.wantLen {
				t.Fatalf("FilterByTags(%v) returned %d entries, want %d", tt.tags, len(got), tt.wantLen)
			}
			for i, wantID := range tt.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("FilterByTags(%v)[%d].ID = %q, want %q", tt.tags, i, got[i].ID, wantID)
				}
			}
		})
	}
}

func TestFilterByTagsInlineHashtags(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [work]\n---\n\nbody mentions #inline here.\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"no frontmatter, just #inline body tag.\n")
	writeNote(t, root, "2026/01/20260103_1003.md",
		"---\ntags: [work]\n---\n\nno inline tags here.\n")

	idx, err := Load(root)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	entries := idx.Entries()

	tests := []struct {
		name    string
		tags    []string
		wantIDs []string
	}{
		{"inline tag matches fm-only and body-only", []string{"inline"}, []string{"1002", "1001"}},
		{"fm tag still matches", []string{"work"}, []string{"1003", "1001"}},
		{"AND across fm and body", []string{"work", "inline"}, []string{"1001"}},
		{"case-insensitive inline", []string{"INLINE"}, []string{"1002", "1001"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByTags(entries, tt.tags)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("FilterByTags(%v) returned %d entries, want %d", tt.tags, len(got), len(tt.wantIDs))
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
	entries := []Entry{
		{Note: Note{Slug: ""}},
		{Note: Note{Slug: "api-redesign"}},
		{Note: Note{Slug: "disable-letter_opener"}},
	}

	got := FilterBySlug(entries, "api-redesign")
	if len(got) != 1 {
		t.Errorf("FilterBySlug(api-redesign) returned %d, want 1", len(got))
	}

	got = FilterBySlug(entries, "")
	if len(got) != 1 {
		t.Errorf("FilterBySlug('') returned %d, want 1", len(got))
	}

	got = FilterBySlug(entries, "nope")
	if len(got) != 0 {
		t.Errorf("FilterBySlug(nope) returned %d, want 0", len(got))
	}
}

func TestFilterByDate(t *testing.T) {
	entries := []Entry{
		{Note: Note{Date: "20260106", ID: "8823"}},
		{Note: Note{Date: "20260104", ID: "8818"}},
		{Note: Note{Date: "20260102", ID: "8814"}},
		{Note: Note{Date: "20241203", ID: "6973"}},
	}

	got := FilterByDate(entries, "20260106")
	if len(got) != 1 || got[0].ID != "8823" {
		t.Errorf("FilterByDate(20260106) = %v, want [{ID:8823}]", got)
	}

	got = FilterByDate(entries, "20260104")
	if len(got) != 1 || got[0].ID != "8818" {
		t.Errorf("FilterByDate(20260104) = %v, want [{ID:8818}]", got)
	}

	got = FilterByDate(entries, "20991231")
	if len(got) != 0 {
		t.Errorf("FilterByDate(no match) = %v, want []", got)
	}
}

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"empty slug is valid", "", false},
		{"normal slug", "my-feature", false},
		{"slug with digits", "feature-123", false},
		{"slug with underscore", "snake_case", false},
		{"all-digit slug", "999", true},
		{"single digit", "0", true},
		{"slug with slash", "foo/bar", true},
		{"slug with backslash", `foo\bar`, true},
		{"slug with dot", "foo.bar", true},
		{"slug with space", "foo bar", true},
		{"slug with control char", "foo\tbar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSlug(%q) error = %v, wantErr %v", tt.slug, err, tt.wantErr)
			}
		})
	}
}

func TestFilterByTypes(t *testing.T) {
	entries := []Entry{
		{Note: Note{Type: ""}},
		{Note: Note{Type: "todo"}},
		{Note: Note{Type: "backlog"}},
		{Note: Note{Type: "todo"}},
	}

	tests := []struct {
		name    string
		types   []string
		wantLen int
	}{
		{"single type todo", []string{"todo"}, 2},
		{"single type backlog", []string{"backlog"}, 1},
		{"empty type matches untyped", []string{""}, 1},
		{"multiple types", []string{"todo", "backlog"}, 3},
		{"no match", []string{"nope"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByTypes(entries, tt.types)
			if len(got) != tt.wantLen {
				t.Errorf("FilterByTypes(%v) returned %d, want %d", tt.types, len(got), tt.wantLen)
			}
		})
	}
}
