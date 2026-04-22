package note

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestFrontmatterIsZero(t *testing.T) {
	tests := []struct {
		name string
		f    Frontmatter
		want bool
	}{
		{"empty", Frontmatter{}, true},
		{"title set", Frontmatter{Title: "T"}, false},
		{"slug set", Frontmatter{Slug: "s"}, false},
		{"tags empty slice is zero", Frontmatter{Tags: []string{}}, true},
		{"tags with value", Frontmatter{Tags: []string{"a"}}, false},
		{"aliases empty slice is zero", Frontmatter{Aliases: []string{}}, true},
		{"aliases with value", Frontmatter{Aliases: []string{"a"}}, false},
		{"description set", Frontmatter{Description: "d"}, false},
		{"public true", Frontmatter{Public: true}, false},
		{"date set", Frontmatter{Date: time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNoteSuccess(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Frontmatter
		body  string
	}{
		{"empty input", "", Frontmatter{}, ""},
		{"no frontmatter", "# Hello\n\nBody text.\n", Frontmatter{}, "# Hello\n\nBody text.\n"},
		{"title only", "---\ntitle: My Note\n---\n\n# Content\n", Frontmatter{Title: "My Note"}, "# Content\n"},
		{"slug only", "---\nslug: my-slug\n---\n\n# Content\n", Frontmatter{Slug: "my-slug"}, "# Content\n"},
		{"tags only", "---\ntags: [work, planning]\n---\n\n# Content\n", Frontmatter{Tags: []string{"work", "planning"}}, "# Content\n"},
		{"description only", "---\ndescription: Quick thought\n---\n\n# Content\n", Frontmatter{Description: "Quick thought"}, "# Content\n"},
		{"public true", "---\npublic: true\n---\n\n# Content\n", Frontmatter{Public: true}, "# Content\n"},
		{"public absent false", "---\ntitle: T\n---\n\n# Content\n", Frontmatter{Title: "T"}, "# Content\n"},
		{
			name:  "all fields",
			input: "---\ntitle: T\nslug: s\ntags: [a]\ndescription: D\npublic: true\n---\n\n# Content\n",
			want:  Frontmatter{Title: "T", Slug: "s", Tags: []string{"a"}, Description: "D", Public: true},
			body:  "# Content\n",
		},
		{"unclosed frontmatter treated as no frontmatter", "---\ntitle: Oops\n# Content\n", Frontmatter{}, "---\ntitle: Oops\n# Content\n"},
		{"int coerced to string", "---\ntitle: 12345\n---\n", Frontmatter{Title: "12345"}, ""},
		{"null leaves field empty", "---\ntitle: null\nslug: s\n---\n", Frontmatter{Slug: "s"}, ""},
		{"empty frontmatter block", "---\n---\n\nBody\n", Frontmatter{}, "Body\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, body, err := ParseNote([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(f, tt.want) {
				t.Errorf("frontmatter: got %+v, want %+v", f, tt.want)
			}
			if string(body) != tt.body {
				t.Errorf("body: got %q, want %q", string(body), tt.body)
			}
		})
	}
}

func TestParseNoteErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"unclosed flow sequence", "---\ntitle: T\ntags: [a, b\n---\n\n# Content\n"},
		{"invalid bool value", "---\npublic: maybe\n---\n\n# Content\n"},
		{"bad field alongside good", "---\ntitle: T\npublic: maybe\ntags: [a, b]\n---\n\n# Content\n"},
		{"control character", "---\ntitle: \"A\x00B\"\nslug: s\n---\n"},
		{"non-mapping top level", "---\n[1, 2, 3]\n---\n"},
		{"duplicate keys rejected", "---\ntitle: A\ntitle: B\n---\n"},
		{
			name: "alias bomb",
			input: "---\n" +
				"a: &a [x]\n" +
				"b: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a,*a]\n" +
				"c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b,*b]\n" +
				"tags: *c\n" +
				"title: T\n" +
				"---\n",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			f, _, err := ParseNote([]byte(tt.input))
			if err == nil {
				t.Fatalf("expected error, got f=%+v", f)
			}
			if !f.IsZero() {
				t.Errorf("expected zero Frontmatter on error, got %+v", f)
			}
		})
	}
}

// On parse error the body is still returned as a sub-slice so bulk readers
// can fall back to body-only processing (e.g. body hashtags still collected).
func TestParseNoteErrorStillReturnsBody(t *testing.T) {
	input := []byte("---\npublic: maybe\n---\n\n# Content\n")
	_, body, err := ParseNote(input)
	if err == nil {
		t.Fatal("expected error")
	}
	if string(body) != "# Content\n" {
		t.Errorf("body = %q, want %q", string(body), "# Content\n")
	}
}

func TestParseNoteBodyIsSliceOfInput(t *testing.T) {
	input := []byte("---\ntitle: T\n---\n\nhello\n")
	_, body, err := ParseNote(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("body is empty")
	}
	if &body[0] != &input[len(input)-len(body)] {
		t.Error("body is not a sub-slice of input (extra allocation)")
	}
}

func TestFormatNoteSnapshotAllFields(t *testing.T) {
	f := Frontmatter{
		Title:       "T",
		Slug:        "s",
		Tags:        []string{"a"},
		Description: "D",
		Public:      true,
	}
	want := "---\ntitle: T\nslug: s\ntags:\n    - a\ndescription: D\npublic: true\n---\n\nbody\n"
	got := string(FormatNote(f, []byte("body\n")))
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatNoteEmptyFrontmatter(t *testing.T) {
	if got := string(FormatNote(Frontmatter{}, []byte("body\n"))); got != "body\n" {
		t.Errorf("got %q, want %q", got, "body\n")
	}
}

func TestRoundtrip(t *testing.T) {
	cases := []Frontmatter{
		{},
		{Title: "T"},
		{Tags: []string{"a", "b"}},
		{Tags: []string{"go", "rust, elixir"}},
		{Tags: []string{"foo: bar", "baz]"}},
		{Title: "Re: Project update"},
		{Title: "T", Slug: "s", Tags: []string{"a"}, Public: true},
		{Title: "T", Slug: "s", Tags: []string{"a"}, Description: "D", Public: true},
	}
	for i, fm := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			out := FormatNote(fm, []byte("body\n"))
			gotF, gotBody, err := ParseNote(out)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if !reflect.DeepEqual(gotF, fm) {
				t.Errorf("frontmatter: got %+v, want %+v", gotF, fm)
			}
			if string(gotBody) != "body\n" {
				t.Errorf("body: got %q, want %q", string(gotBody), "body\n")
			}
		})
	}
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no frontmatter", "# Hello\n\nBody text.\n", "# Hello\n\nBody text.\n"},
		{"with frontmatter", "---\nslug: todo\ntags: [journal]\n---\n\n# Hello\n\nBody text.\n", "# Hello\n\nBody text.\n"},
		{"frontmatter only", "---\nslug: todo\n---\n", ""},
		{"empty input", "", ""},
		{"unclosed frontmatter", "---\nslug: todo\n# Hello\n", "---\nslug: todo\n# Hello\n"},
		{"triple dash in body not at start", "# Hello\n\n---\n\nFooter.\n", "# Hello\n\n---\n\nFooter.\n"},
		{"preserves multiple blank lines after frontmatter", "---\nslug: todo\n---\n\n\n\nContent\n", "\n\nContent\n"},
		{"opening delimiter with trailing text", "---extra\nslug: x\n---\n\nBody\n", "---extra\nslug: x\n---\n\nBody\n"},
		{"opening delimiter only no newline", "---", "---"},
		{"opening delimiter only with newline", "---\nstuff\n", "---\nstuff\n"},
		{"empty frontmatter block", "---\n---\n\nBody\n", "Body\n"},
		{"malformed yaml still stripped", "---\n[bad: yaml\n---\n\nBody\n", "Body\n"},
		{"multiple closing delimiters", "---\na\n---\nb\n---\n\nBody\n", "b\n---\n\nBody\n"},
		{
			name:  "roundtrip with FormatNote",
			input: string(FormatNote(Frontmatter{Tags: []string{"journal"}, Description: "A note"}, []byte("# Content\n"))),
			want:  "# Content\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(StripFrontmatter([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// CRLF: interior body bytes must round-trip through ParseNote. Delimiter
// lines are LF-only on write; CRLF delimiter lines on read are tolerated.
func TestParseNoteCRLFInteriorPreserved(t *testing.T) {
	input := []byte("---\r\ntitle: T\r\ntags:\r\n  - a\r\n  - b\r\n---\r\n\r\nbody line\r\nsecond\r\n")
	f, body, err := ParseNote(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Title != "T" {
		t.Errorf("Title = %q", f.Title)
	}
	if len(f.Tags) != 2 || f.Tags[0] != "a" || f.Tags[1] != "b" {
		t.Errorf("Tags = %v", f.Tags)
	}
	want := "body line\r\nsecond\r\n"
	if string(body) != want {
		t.Errorf("body: got %q, want %q", string(body), want)
	}
}

func TestFormatNoteWritesLFOnly(t *testing.T) {
	out := FormatNote(Frontmatter{Title: "T"}, []byte("hello\r\nworld\r\n"))
	wantPrefix := "---\ntitle: T\n---\n\n"
	if string(out[:len(wantPrefix)]) != wantPrefix {
		t.Errorf("delimiter lines not LF-only: %q", string(out[:len(wantPrefix)]))
	}
	if string(out[len(wantPrefix):]) != "hello\r\nworld\r\n" {
		t.Errorf("body modified: %q", string(out[len(wantPrefix):]))
	}
}

func TestParseNoteExtraPreservesUnknownKeys(t *testing.T) {
	in := []byte("---\ntitle: T\nfeatured: true\ncustom: hello\n---\n\nbody\n")
	fm, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if fm.Title != "T" {
		t.Errorf("Title = %q, want %q", fm.Title, "T")
	}
	if string(body) != "body\n" {
		t.Errorf("body = %q, want %q", string(body), "body\n")
	}
	if _, ok := fm.Extra["featured"]; !ok {
		t.Error("Extra missing key 'featured'")
	}
	if _, ok := fm.Extra["custom"]; !ok {
		t.Error("Extra missing key 'custom'")
	}
	featuredNode := fm.Extra["featured"]
	var featured bool
	if err := featuredNode.Decode(&featured); err != nil {
		t.Fatalf("decode featured: %v", err)
	}
	if !featured {
		t.Errorf("featured = %v, want true", featured)
	}
}

func TestFormatNoteExtraPreservedInAlphaOrder(t *testing.T) {
	in := []byte("---\ntitle: T\nzebra: striped\nalpha: 1\nfeatured: true\n---\n\nbody\n")
	fm, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	out := string(FormatNote(fm, body))
	// Reserved "title" first; Extra keys alpha-sorted: alpha, featured, zebra.
	want := "---\ntitle: T\nalpha: 1\nfeatured: true\nzebra: striped\n---\n\nbody\n"
	if out != want {
		t.Errorf("FormatNote =\n%q\nwant:\n%q", out, want)
	}
}

func TestFormatNoteEmptyFrontmatterWithExtraOnly(t *testing.T) {
	fm := Frontmatter{Extra: map[string]yaml.Node{
		"featured": {Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
	}}
	want := "---\nfeatured: true\n---\n\nbody\n"
	got := string(FormatNote(fm, []byte("body\n")))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsZeroIncludesExtra(t *testing.T) {
	if (Frontmatter{}).IsZero() == false {
		t.Error("empty Frontmatter should be zero")
	}
	fm := Frontmatter{Extra: map[string]yaml.Node{
		"featured": {Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
	}}
	if fm.IsZero() {
		t.Error("Frontmatter with Extra should not be zero")
	}
}

func TestTypeRoundTrips(t *testing.T) {
	in := []byte("---\ntitle: T\ntype: meeting\n---\n\nbody\n")
	fm, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if fm.Type != "meeting" {
		t.Errorf("Type = %q, want meeting", fm.Type)
	}
	out := string(FormatNote(fm, body))
	want := "---\ntitle: T\ntype: meeting\n---\n\nbody\n"
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
}

func TestTypeFieldOrder(t *testing.T) {
	fm := Frontmatter{
		Title: "T", Slug: "s", Type: "meeting",
		Tags: []string{"a"}, Description: "D", Public: true,
	}
	got := string(FormatNote(fm, []byte("body\n")))
	want := "---\ntitle: T\nslug: s\ntype: meeting\ntags:\n    - a\ndescription: D\npublic: true\n---\n\nbody\n"
	if got != want {
		t.Errorf("FormatNote =\n%q\nwant:\n%q", got, want)
	}
}

func TestDateRoundTripDateOnly(t *testing.T) {
	in := []byte("---\ntitle: T\ndate: 2026-04-22\n---\n\nbody\n")
	fm, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	want := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	if !fm.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", fm.Date, want)
	}
	if _, ok := fm.Extra["date"]; ok {
		t.Error("Date should be on the typed field, not in Extra")
	}
	out := string(FormatNote(fm, body))
	wantOut := "---\ntitle: T\ndate: 2026-04-22\n---\n\nbody\n"
	if out != wantOut {
		t.Errorf("FormatNote =\n%q\nwant:\n%q", out, wantOut)
	}
}

func TestDateRoundTripRFC3339(t *testing.T) {
	in := []byte("---\ntitle: T\ndate: 2026-04-22T15:30:00Z\n---\n\nbody\n")
	fm, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	want := time.Date(2026, 4, 22, 15, 30, 0, 0, time.UTC)
	if !fm.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", fm.Date, want)
	}
	out := string(FormatNote(fm, body))
	wantOut := "---\ntitle: T\ndate: 2026-04-22T15:30:00Z\n---\n\nbody\n"
	if out != wantOut {
		t.Errorf("FormatNote =\n%q\nwant:\n%q", out, wantOut)
	}
}

func TestDateFieldOrder(t *testing.T) {
	fm := Frontmatter{
		Title: "T", Slug: "s", Type: "meeting",
		Date: time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Tags: []string{"a"}, Description: "D", Public: true,
	}
	got := string(FormatNote(fm, []byte("body\n")))
	want := "---\ntitle: T\nslug: s\ntype: meeting\ndate: 2026-04-22\ntags:\n    - a\ndescription: D\npublic: true\n---\n\nbody\n"
	if got != want {
		t.Errorf("FormatNote =\n%q\nwant:\n%q", got, want)
	}
}

// Migration check: a note whose `date:` previously landed in Extra (because
// `date` was not a reserved key) now populates the typed Date field instead.
func TestDateMigratesFromExtra(t *testing.T) {
	in := []byte("---\ntitle: Old note\ndate: 2025-01-15\nfeatured: true\n---\n\nbody\n")
	fm, _, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	want := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !fm.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", fm.Date, want)
	}
	if _, ok := fm.Extra["date"]; ok {
		t.Error("date key should not be in Extra after migration")
	}
	if _, ok := fm.Extra["featured"]; !ok {
		t.Error("non-reserved Extra keys should still round-trip")
	}
}

func TestDateInvalidRejected(t *testing.T) {
	in := []byte("---\ntitle: T\ndate: not-a-date\n---\n\nbody\n")
	_, _, err := ParseNote(in)
	if err == nil {
		t.Fatal("expected error for malformed date")
	}
}

func TestAliasesRoundTrip(t *testing.T) {
	in := []byte("---\ntitle: T\naliases:\n  - old-slug\n  - even-older\n---\n\nbody\n")
	fm, body, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if len(fm.Aliases) != 2 || fm.Aliases[0] != "old-slug" || fm.Aliases[1] != "even-older" {
		t.Errorf("Aliases = %v, want [old-slug even-older]", fm.Aliases)
	}
	if _, ok := fm.Extra["aliases"]; ok {
		t.Error("Aliases should be on the typed field, not in Extra")
	}
	out := string(FormatNote(fm, body))
	want := "---\ntitle: T\naliases:\n    - old-slug\n    - even-older\n---\n\nbody\n"
	if out != want {
		t.Errorf("FormatNote =\n%q\nwant:\n%q", out, want)
	}
}

func TestAliasesFieldOrder(t *testing.T) {
	fm := Frontmatter{
		Title: "T", Slug: "s", Type: "meeting",
		Date:        time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Tags:        []string{"a"},
		Aliases:     []string{"old"},
		Description: "D", Public: true,
	}
	got := string(FormatNote(fm, []byte("body\n")))
	want := "---\ntitle: T\nslug: s\ntype: meeting\ndate: 2026-04-22\ntags:\n    - a\naliases:\n    - old\ndescription: D\npublic: true\n---\n\nbody\n"
	if got != want {
		t.Errorf("FormatNote =\n%q\nwant:\n%q", got, want)
	}
}

// Migration check: a note whose `aliases:` previously landed in Extra now
// populates the typed Aliases field instead, and non-reserved Extra keys
// continue to round-trip.
func TestAliasesMigratesFromExtra(t *testing.T) {
	in := []byte("---\ntitle: Old note\naliases:\n  - prior-slug\n  - legacy-id\nfeatured: true\n---\n\nbody\n")
	fm, _, err := ParseNote(in)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	want := []string{"prior-slug", "legacy-id"}
	if !reflect.DeepEqual(fm.Aliases, want) {
		t.Errorf("Aliases = %v, want %v", fm.Aliases, want)
	}
	if _, ok := fm.Extra["aliases"]; ok {
		t.Error("aliases key should not be in Extra after migration")
	}
	if _, ok := fm.Extra["featured"]; !ok {
		t.Error("non-reserved Extra keys should still round-trip")
	}
}

func TestAliasesInvalidRejected(t *testing.T) {
	in := []byte("---\ntitle: T\naliases: not-a-list\n---\n\nbody\n")
	_, _, err := ParseNote(in)
	if err == nil {
		t.Fatal("expected error for non-list aliases")
	}
}

func TestRoundtripWithAliases(t *testing.T) {
	cases := []Frontmatter{
		{Aliases: []string{"a"}},
		{Title: "T", Aliases: []string{"old-slug", "prior"}},
		{Title: "T", Slug: "new", Aliases: []string{"old"}, Tags: []string{"x"}},
		{Aliases: []string{"contains: colon", "brackets]"}},
	}
	for i, fm := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			out := FormatNote(fm, []byte("body\n"))
			gotF, gotBody, err := ParseNote(out)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if !reflect.DeepEqual(gotF.Aliases, fm.Aliases) {
				t.Errorf("Aliases: got %v, want %v", gotF.Aliases, fm.Aliases)
			}
			if string(gotBody) != "body\n" {
				t.Errorf("body: got %q, want %q", string(gotBody), "body\n")
			}
		})
	}
}

func TestRoundtripWithDate(t *testing.T) {
	cases := []Frontmatter{
		{Date: time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)},
		{Title: "T", Date: time.Date(2026, 4, 22, 15, 30, 0, 0, time.UTC)},
		{Title: "T", Slug: "s", Date: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), Tags: []string{"a"}},
	}
	for i, fm := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			out := FormatNote(fm, []byte("body\n"))
			gotF, gotBody, err := ParseNote(out)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if !gotF.Date.Equal(fm.Date) {
				t.Errorf("Date: got %v, want %v", gotF.Date, fm.Date)
			}
			if string(gotBody) != "body\n" {
				t.Errorf("body: got %q, want %q", string(gotBody), "body\n")
			}
		})
	}
}
