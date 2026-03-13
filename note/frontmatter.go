package note

import (
	"bytes"
	"strings"
)

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

var frontmatterDelim = []byte("---")

// StripFrontmatter removes YAML frontmatter from the beginning of data.
// Frontmatter must start on the first line with "---" and end with a
// subsequent "---" line. Leading whitespace after the closing delimiter
// is also removed.
func StripFrontmatter(data []byte) []byte {
	// Must start with "---"
	if !bytes.HasPrefix(data, frontmatterDelim) {
		return data
	}

	// Find closing "---" on its own line after the opening one.
	rest := data[len(frontmatterDelim):]
	// Skip to end of the opening delimiter line.
	idx := bytes.IndexByte(rest, '\n')
	if idx < 0 {
		return data
	}
	rest = rest[idx+1:]

	for {
		line, after, found := bytes.Cut(rest, []byte("\n"))
		if bytes.Equal(bytes.TrimRight(line, "\r"), frontmatterDelim) {
			// Trim exactly one leading blank line if present.
			if len(after) > 0 && after[0] == '\n' {
				after = after[1:]
			} else if len(after) > 1 && after[0] == '\r' && after[1] == '\n' {
				after = after[2:]
			}
			return after
		}
		if !found {
			// Reached end without closing delimiter.
			return data
		}
		rest = after
	}
}
