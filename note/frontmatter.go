package note

import (
	"bytes"
	"strings"
)

// FrontmatterFields holds optional fields for note frontmatter.
type FrontmatterFields struct {
	Title       string
	Tags        []string
	Description string
}

// BuildFrontmatter generates YAML frontmatter from the given fields.
// Returns empty string if no fields are provided.
func BuildFrontmatter(f FrontmatterFields) string {
	var lines []string

	if f.Title != "" {
		lines = append(lines, "title: "+f.Title)
	}
	if len(f.Tags) > 0 {
		lines = append(lines, "tags: ["+strings.Join(f.Tags, ", ")+"]")
	}
	if f.Description != "" {
		lines = append(lines, "description: "+f.Description)
	}

	if len(lines) == 0 {
		return ""
	}

	return "---\n" + strings.Join(lines, "\n") + "\n---\n\n"
}

// ParseFrontmatterFields extracts all frontmatter fields from data.
// Returns zero-value FrontmatterFields if no valid frontmatter is present.
func ParseFrontmatterFields(data []byte) FrontmatterFields {
	if !bytes.HasPrefix(data, frontmatterDelim) {
		return FrontmatterFields{}
	}

	rest := data[len(frontmatterDelim):]
	idx := bytes.IndexByte(rest, '\n')
	if idx < 0 {
		return FrontmatterFields{}
	}
	if len(bytes.TrimRight(rest[:idx], "\r")) > 0 {
		return FrontmatterFields{}
	}
	rest = rest[idx+1:]

	var f FrontmatterFields
	for {
		line, after, found := bytes.Cut(rest, []byte("\n"))
		trimmed := bytes.TrimRight(line, "\r")

		if bytes.Equal(trimmed, frontmatterDelim) {
			return f
		}

		s := string(trimmed)
		if t := strings.TrimPrefix(s, "title: "); t != s {
			f.Title = t
		} else if t := strings.TrimPrefix(s, "description: "); t != s {
			f.Description = t
		} else if bytes.HasPrefix(trimmed, []byte("tags: [")) && bytes.HasSuffix(trimmed, []byte("]")) {
			inner := s[len("tags: [") : len(s)-1]
			if inner != "" {
				f.Tags = strings.Split(inner, ", ")
			}
		}

		if !found {
			// Reached end without closing delimiter — not valid frontmatter.
			return FrontmatterFields{}
		}
		rest = after
	}
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
	// Opening delimiter must be exactly "---" on its own line.
	idx := bytes.IndexByte(rest, '\n')
	if idx < 0 {
		return data
	}
	if len(bytes.TrimRight(rest[:idx], "\r")) > 0 {
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
