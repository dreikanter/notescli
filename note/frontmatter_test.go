package note

import "testing"

func TestBuildFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		slug        string
		tags        []string
		description string
		want        string
	}{
		{
			name: "empty",
			want: "",
		},
		{
			name: "slug only",
			slug: "todo",
			want: "---\nslug: todo\n---\n\n",
		},
		{
			name: "tags only",
			tags: []string{"journal", "idea"},
			want: "---\ntags: [journal, idea]\n---\n\n",
		},
		{
			name:        "description only",
			description: "Quick thought",
			want:        "---\ndescription: Quick thought\n---\n\n",
		},
		{
			name:        "all fields",
			slug:        "weekly",
			tags:        []string{"review"},
			description: "Week 10",
			want:        "---\nslug: weekly\ntags: [review]\ndescription: Week 10\n---\n\n",
		},
		{
			name: "single tag",
			tags: []string{"journal"},
			want: "---\ntags: [journal]\n---\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFrontmatter(tt.slug, tt.tags, tt.description)
			if got != tt.want {
				t.Errorf("BuildFrontmatter(%q, %v, %q) =\n%q\nwant:\n%q", tt.slug, tt.tags, tt.description, got, tt.want)
			}
		})
	}
}
