package note

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExtractHashtagsBasic(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"simple", "hello #world", []string{"world"}},
		{"multiple", "#alpha and #beta here", []string{"alpha", "beta"}},
		{"digits and dashes", "#a-b_c #123 #x1", []string{"a-b_c", "123", "x1"}},
		{"slash terminates", "see #foo/bar", []string{"foo"}},
		{"punctuation after", "ok #tag, next.", []string{"tag"}},
		{"parens", "(#tag)", []string{"tag"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ExtractHashtags([]byte(c.in))
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestExtractHashtagsNegative(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"heading h1", "# heading not a tag"},
		{"heading h2", "## another heading"},
		{"indented heading", "   # still a heading"},
		{"word-prefixed", "foo#bar baz"},
		{"bare hash", "look here: # not-tag"},
		{"lone hash", "just # alone"},
		{"url anchor", "https://www.teamviewer.com/en/#screenshotsAnchor"},
		{"url anchor bare", "see example.com/path/#section for more"},
		{"backticked tag", "prose `#hashtag` continues"},
		{"chained hashes", "#one#two"},
		{"chained three", "prefix #one#two#three suffix"},
		{"domain anchor", "visit foo.bar/#frag here"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ExtractHashtags([]byte(c.in))
			if len(got) != 0 {
				t.Fatalf("expected no tags, got %v", got)
			}
		})
	}
}

func TestExtractHashtagsInlineCode(t *testing.T) {
	in := "real #out and `inline #in` and #back"
	want := []string{"out", "back"}
	got := ExtractHashtags([]byte(in))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractHashtagsFencedBlock(t *testing.T) {
	in := "before #a\n```\n#hidden\n#also-hidden\n```\nafter #b\n"
	want := []string{"a", "b"}
	got := ExtractHashtags([]byte(in))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractHashtagsFencedBlockWithInfoString(t *testing.T) {
	in := "top #ok\n```go\n// #comment\n```\nend #done\n"
	want := []string{"ok", "done"}
	got := ExtractHashtags([]byte(in))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractHashtagsCRLF(t *testing.T) {
	in := "before #a\r\n```\r\n#hidden\r\n```\r\nafter #b\r\n"
	want := []string{"a", "b"}
	got := ExtractHashtags([]byte(in))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractHashtagsBareHash(t *testing.T) {
	cases := []string{"#", "text # and #", "line #\nnext #"}
	for _, in := range cases {
		got := ExtractHashtags([]byte(in))
		if len(got) != 0 {
			t.Errorf("input %q: expected no tags, got %v", in, got)
		}
	}
}

func writeNote(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEntryBodyHashtags(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [fm]\n---\n\nBody #alpha and #beta, #alpha again.\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"---\n---\n\nno hashtags here.\n")
	writeNote(t, root, "2026/01/20260103_1003.md",
		"no-frontmatter body #gamma\n")

	idx, err := Load(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byRel := make(map[string][]string)
	for _, e := range idx.Entries() {
		byRel[e.RelPath] = e.BodyHashtags()
	}

	if got := byRel["2026/01/20260101_1001.md"]; !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Errorf("1001 BodyHashtags = %v, want [alpha beta]", got)
	}
	if got := byRel["2026/01/20260102_1002.md"]; got != nil {
		t.Errorf("1002 BodyHashtags = %v, want nil", got)
	}
	if got := byRel["2026/01/20260103_1003.md"]; !reflect.DeepEqual(got, []string{"gamma"}) {
		t.Errorf("1003 BodyHashtags = %v, want [gamma]", got)
	}
}

func TestEntryBodyHashtagsWithoutFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"body #alpha #beta\n")

	idx, err := Load(root, WithFrontmatter(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := idx.Entries()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if got := entries[0].BodyHashtags(); got != nil {
		t.Errorf("BodyHashtags = %v, want nil when WithFrontmatter(false)", got)
	}
}

func TestEntryBodyHashtagsReturnsCopy(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\n---\n\nbody #alpha #beta\n")

	idx, err := Load(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := idx.Entries()
	first := entries[0].BodyHashtags()
	if len(first) == 0 {
		t.Fatal("expected hashtags")
	}
	first[0] = "mutated"
	second := entries[0].BodyHashtags()
	if second[0] != "alpha" {
		t.Errorf("mutation leaked: second[0] = %q, want %q", second[0], "alpha")
	}
}
