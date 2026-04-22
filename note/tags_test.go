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

func TestExtractTagsEmpty(t *testing.T) {
	root := t.TempDir()
	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no tags, got %v", got)
	}
}

func TestExtractTagsFrontmatterOnly(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [work, planning]\n---\n\nbody here.\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"---\ntags: [work]\n---\n\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"planning", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsBodyHashtagsOnly(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"Text with #alpha and #beta.\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"More text #alpha only.\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsMergedAndDeduped(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [work, shared]\n---\n\nBody #shared #body-only\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"no frontmatter here #work #another\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"another", "body-only", "shared", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsFrontmatterUniqueAcrossStore(t *testing.T) {
	// The unique tag comes only from frontmatter; body hashtag coverage
	// differs. A regression in ParseNote integration would drop fm-unique
	// and fail this test.
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [fm-unique]\n---\n\nbody mentions #body-unique only.\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"body-unique", "fm-unique"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsIgnoresCodeBlocks(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"real #kept\n```\n#ignored\n```\nafter #also-kept\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"also-kept", "kept"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsLowercasesMixedCase(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [Work, PLANNING]\n---\n\nbody #Coffee and #coffee.\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"no fm, #WORK here.\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"coffee", "planning", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsNonexistentRoot(t *testing.T) {
	_, err := ExtractTags(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing root")
	}
}
