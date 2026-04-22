package note

import "testing"

func TestNormalizeSlug(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"hello", "hello"},
		{"Hello", "hello"},
		{"HELLO", "hello"},
		{"hello world", "hello-world"},
		{"hello  world", "hello-world"},
		{"hello_world", "hello-world"},
		{"hello-world", "hello-world"},
		{"hello--world", "hello-world"},
		{"hello!@#world", "hello-world"},
		{"---leading", "leading"},
		{"trailing---", "trailing"},
		{"  spaces  ", "spaces"},
		{"café", "caf"},
		{"123abc", "123abc"},
		{"ABC123", "abc123"},
		{"API Redesign (v2)", "api-redesign-v2"},
		{"___", ""},
		{"!!!", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := NormalizeSlug(c.in); got != c.want {
				t.Errorf("NormalizeSlug(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestDeriveSlug(t *testing.T) {
	cases := []struct {
		name            string
		stem            string
		id              string
		frontmatterSlug string
		want            string
	}{
		{
			name:            "frontmatter wins over stem",
			stem:            "20260106_8823_old-slug",
			id:              "20260106_8823",
			frontmatterSlug: "New Slug!",
			want:            "new-slug",
		},
		{
			name: "uid prefix stripped from stem",
			stem: "20260106_8823_api-redesign",
			id:   "20260106_8823",
			want: "api-redesign",
		},
		{
			name: "stem with no trailing slug yields empty",
			stem: "20260106_8823",
			id:   "20260106_8823",
			want: "",
		},
		{
			name: "stem not matching id prefix used as-is",
			stem: "meeting-notes",
			id:   "20260106_8823",
			want: "meeting-notes",
		},
		{
			name: "empty id leaves stem untouched",
			stem: "raw_stem",
			id:   "",
			want: "raw-stem",
		},
		{
			name:            "frontmatter wins with empty stem",
			stem:            "",
			id:              "",
			frontmatterSlug: "fm-slug",
			want:            "fm-slug",
		},
		{
			name:            "frontmatter normalized",
			stem:            "",
			id:              "",
			frontmatterSlug: "Café Review",
			want:            "caf-review",
		},
		{
			name: "all inputs empty returns empty",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DeriveSlug(c.stem, c.id, c.frontmatterSlug)
			if got != c.want {
				t.Errorf("DeriveSlug(%q, %q, %q) = %q, want %q",
					c.stem, c.id, c.frontmatterSlug, got, c.want)
			}
		})
	}
}
