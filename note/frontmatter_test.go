package note

import (
	"testing"
)

func TestParseFrontmatterFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  FrontmatterFields
	}{
		{
			name:  "empty input",
			input: "",
			want:  FrontmatterFields{},
		},
		{
			name:  "no frontmatter",
			input: "# Hello\n\nBody text.\n",
			want:  FrontmatterFields{},
		},
		{
			name:  "title only",
			input: "---\ntitle: My Note\n---\n\n# Content\n",
			want:  FrontmatterFields{Title: "My Note"},
		},
		{
			name:  "tags only",
			input: "---\ntags: [work, planning]\n---\n\n# Content\n",
			want:  FrontmatterFields{Tags: []string{"work", "planning"}},
		},
		{
			name:  "description only",
			input: "---\ndescription: Quick thought\n---\n\n# Content\n",
			want:  FrontmatterFields{Description: "Quick thought"},
		},
		{
			name:  "all fields",
			input: "---\ntitle: Weekly Review\ntags: [review, work]\ndescription: Week 10\n---\n\n# Content\n",
			want: FrontmatterFields{
				Title:       "Weekly Review",
				Tags:        []string{"review", "work"},
				Description: "Week 10",
			},
		},
		{
			name:  "unclosed frontmatter",
			input: "---\ntitle: Oops\n# Content\n",
			want:  FrontmatterFields{},
		},
		{
			name:  "roundtrip with BuildFrontmatter",
			input: BuildFrontmatter(FrontmatterFields{Title: "T", Tags: []string{"a", "b"}, Description: "D"}) + "body\n",
			want:  FrontmatterFields{Title: "T", Tags: []string{"a", "b"}, Description: "D"},
		},
		{
			name:  "slug only",
			input: "---\nslug: my-slug\n---\n\n# Content\n",
			want:  FrontmatterFields{Slug: "my-slug"},
		},
		{
			name:  "public true",
			input: "---\npublic: true\n---\n\n# Content\n",
			want:  FrontmatterFields{Public: true},
		},
		{
			name:  "public absent means false",
			input: "---\ntitle: T\n---\n\n# Content\n",
			want:  FrontmatterFields{Title: "T"},
		},
		{
			name:  "public non-true value means false",
			input: "---\npublic: yes\n---\n\n# Content\n",
			want:  FrontmatterFields{},
		},
		{
			name:  "all fields including slug and public",
			input: "---\ntitle: T\nslug: s\ntags: [a]\ndescription: D\npublic: true\n---\n\n# Content\n",
			want: FrontmatterFields{
				Title:       "T",
				Slug:        "s",
				Tags:        []string{"a"},
				Description: "D",
				Public:      true,
			},
		},
		{
			name: "roundtrip with slug and public",
			input: BuildFrontmatter(FrontmatterFields{
				Title:  "T",
				Slug:   "s",
				Tags:   []string{"a"},
				Public: true,
			}) + "body\n",
			want: FrontmatterFields{Title: "T", Slug: "s", Tags: []string{"a"}, Public: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFrontmatterFields([]byte(tt.input))
			if got.Title != tt.want.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.want.Title)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", got.Description, tt.want.Description)
			}
			if len(got.Tags) != len(tt.want.Tags) {
				t.Fatalf("Tags = %v, want %v", got.Tags, tt.want.Tags)
			}
			for i := range tt.want.Tags {
				if got.Tags[i] != tt.want.Tags[i] {
					t.Errorf("Tags[%d] = %q, want %q", i, got.Tags[i], tt.want.Tags[i])
				}
			}
			if got.Slug != tt.want.Slug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.want.Slug)
			}
			if got.Public != tt.want.Public {
				t.Errorf("Public = %v, want %v", got.Public, tt.want.Public)
			}
		})
	}
}

func TestBuildFrontmatter(t *testing.T) {
	tests := []struct {
		name   string
		fields FrontmatterFields
		want   string
	}{
		{
			name: "empty",
			want: "",
		},
		{
			name:   "tags only",
			fields: FrontmatterFields{Tags: []string{"journal", "idea"}},
			want:   "---\ntags: [journal, idea]\n---\n\n",
		},
		{
			name:   "description only",
			fields: FrontmatterFields{Description: "Quick thought"},
			want:   "---\ndescription: Quick thought\n---\n\n",
		},
		{
			name: "all fields",
			fields: FrontmatterFields{
				Title:       "Weekly Review",
				Tags:        []string{"review"},
				Description: "Week 10",
			},
			want: "---\ntitle: Weekly Review\ntags: [review]\ndescription: Week 10\n---\n\n",
		},
		{
			name:   "single tag",
			fields: FrontmatterFields{Tags: []string{"journal"}},
			want:   "---\ntags: [journal]\n---\n\n",
		},
		{
			name:   "title only",
			fields: FrontmatterFields{Title: "My Note"},
			want:   "---\ntitle: My Note\n---\n\n",
		},
		{
			name:   "slug only",
			fields: FrontmatterFields{Slug: "my-slug"},
			want:   "---\nslug: my-slug\n---\n\n",
		},
		{
			name:   "public true",
			fields: FrontmatterFields{Public: true},
			want:   "---\npublic: true\n---\n\n",
		},
		{
			name:   "public false omitted",
			fields: FrontmatterFields{Title: "T"},
			want:   "---\ntitle: T\n---\n\n",
		},
		{
			name: "all fields including slug and public",
			fields: FrontmatterFields{
				Title:       "T",
				Slug:        "s",
				Tags:        []string{"a"},
				Description: "D",
				Public:      true,
			},
			want: "---\ntitle: T\nslug: s\ntags: [a]\ndescription: D\npublic: true\n---\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFrontmatter(tt.fields)
			if got != tt.want {
				t.Errorf("BuildFrontmatter(%+v) =\n%q\nwant:\n%q", tt.fields, got, tt.want)
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
		{
			name:  "no frontmatter",
			input: "# Hello\n\nBody text.\n",
			want:  "# Hello\n\nBody text.\n",
		},
		{
			name:  "with frontmatter",
			input: "---\nslug: todo\ntags: [journal]\n---\n\n# Hello\n\nBody text.\n",
			want:  "# Hello\n\nBody text.\n",
		},
		{
			name:  "frontmatter only",
			input: "---\nslug: todo\n---\n",
			want:  "",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "unclosed frontmatter",
			input: "---\nslug: todo\n# Hello\n",
			want:  "---\nslug: todo\n# Hello\n",
		},
		{
			name:  "triple dash in body not at start",
			input: "# Hello\n\n---\n\nFooter.\n",
			want:  "# Hello\n\n---\n\nFooter.\n",
		},
		{
			name:  "preserves multiple blank lines after frontmatter",
			input: "---\nslug: todo\n---\n\n\n\nContent\n",
			want:  "\n\nContent\n",
		},
		{
			name:  "opening delimiter with trailing text",
			input: "---extra\nslug: x\n---\n\nBody\n",
			want:  "---extra\nslug: x\n---\n\nBody\n",
		},
		{
			name:  "opening delimiter only no newline",
			input: "---",
			want:  "---",
		},
		{
			name:  "opening delimiter only with newline",
			input: "---\nstuff\n",
			want:  "---\nstuff\n",
		},
		{
			name:  "empty frontmatter block",
			input: "---\n---\n\nBody\n",
			want:  "Body\n",
		},
		{
			name:  "malformed yaml in frontmatter",
			input: "---\n[bad: yaml\n---\n\nBody\n",
			want:  "Body\n",
		},
		{
			name:  "multiple closing delimiters",
			input: "---\na\n---\nb\n---\n\nBody\n",
			want:  "b\n---\n\nBody\n",
		},
		{
			name:  "roundtrip with BuildFrontmatter",
			input: BuildFrontmatter(FrontmatterFields{Tags: []string{"journal"}, Description: "A note"}) + "# Content\n",
			want:  "# Content\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(StripFrontmatter([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("StripFrontmatter(%q) =\n%q\nwant:\n%q", tt.input, got, tt.want)
			}
		})
	}
}
