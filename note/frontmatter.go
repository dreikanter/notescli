package note

import "strings"

// BuildFrontmatter generates YAML frontmatter from the given fields.
// Returns empty string if no fields are provided.
func BuildFrontmatter(slug string, tags []string, description string) string {
	var lines []string

	if slug != "" {
		lines = append(lines, "slug: "+slug)
	}
	if len(tags) > 0 {
		lines = append(lines, "tags: ["+strings.Join(tags, ", ")+"]")
	}
	if description != "" {
		lines = append(lines, "description: "+description)
	}

	if len(lines) == 0 {
		return ""
	}

	return "---\n" + strings.Join(lines, "\n") + "\n---\n\n"
}
