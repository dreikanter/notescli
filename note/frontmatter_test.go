package note

import (
	"testing"
)

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

func TestParseTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "multiple tags",
			input: "---\ntags: [work, planning]\n---\n\n# Note\n",
			want:  []string{"work", "planning"},
		},
		{
			name:  "single tag",
			input: "---\ntags: [journal]\n---\n\n# Note\n",
			want:  []string{"journal"},
		},
		{
			name:  "no frontmatter",
			input: "# Hello\n\nBody text.\n",
			want:  nil,
		},
		{
			name:  "frontmatter without tags",
			input: "---\ntitle: My Note\n---\n\n# Note\n",
			want:  nil,
		},
		{
			name:  "empty tags",
			input: "---\ntags: []\n---\n\n# Note\n",
			want:  nil,
		},
		{
			name:  "tags with other fields",
			input: "---\ntitle: Weekly Review\ntags: [review, work]\ndescription: Week 10\n---\n\n# Note\n",
			want:  []string{"review", "work"},
		},
		{
			name:  "roundtrip with BuildFrontmatter",
			input: BuildFrontmatter(FrontmatterFields{Tags: []string{"x", "y"}}) + "# Content\n",
			want:  []string{"x", "y"},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "unclosed frontmatter",
			input: "---\ntags: [a]\n# Hello\n",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTags([]byte(tt.input))
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseTags(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseTags(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("ParseTags(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
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
