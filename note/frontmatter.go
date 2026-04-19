package note

import (
	"bytes"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

const frontmatterDelim = "---"

// Frontmatter holds optional fields for note frontmatter.
// Adding a field is a one-line struct addition — no other changes required.
type Frontmatter struct {
	Title       string   `yaml:"title,omitempty"`
	Slug        string   `yaml:"slug,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Public      bool     `yaml:"public,omitempty"`
}

// IsZero reports whether f has no fields set.
func (f Frontmatter) IsZero() bool {
	return reflect.ValueOf(f).IsZero()
}

// ParseNote splits a note file into its frontmatter and body.
// If no frontmatter block is present, the zero Frontmatter is returned along
// with the full input as body and a nil error.
// If the frontmatter block is present but malformed, a non-nil error is
// returned along with the zero Frontmatter; the body is still returned as
// a sub-slice so bulk readers can fall back to body-only processing.
// The returned body is always a sub-slice of the input — no allocation.
func ParseNote(data []byte) (Frontmatter, []byte, error) {
	bodyStart, fmEnd, ok := frontmatterEnd(data)
	if !ok {
		return Frontmatter{}, data, nil
	}
	yamlStart := len(frontmatterDelim) + 1
	var f Frontmatter
	if err := yaml.Unmarshal(data[yamlStart:fmEnd], &f); err != nil {
		return Frontmatter{}, data[bodyStart:], fmt.Errorf("parse frontmatter: %w", err)
	}
	return f, data[bodyStart:], nil
}

// FormatNote serialises frontmatter followed by body. Omits the frontmatter
// block entirely when f.IsZero(). yaml.Marshal cannot fail for this struct,
// so marshal errors are treated as impossible and cause a panic.
func FormatNote(f Frontmatter, body []byte) []byte {
	if f.IsZero() {
		return body
	}
	out, err := yaml.Marshal(f)
	if err != nil {
		panic(fmt.Sprintf("yaml.Marshal Frontmatter: %v", err))
	}
	const prefix = "---\n"
	const suffix = "---\n\n"
	buf := make([]byte, 0, len(prefix)+len(out)+len(suffix)+len(body))
	buf = append(buf, prefix...)
	buf = append(buf, out...)
	buf = append(buf, suffix...)
	buf = append(buf, body...)
	return buf
}

// StripFrontmatter returns data with any leading frontmatter block removed.
// If no valid frontmatter block is present, data is returned unchanged.
// Convenience for callers that want the body without parsing (e.g.
// `notes read --no-frontmatter`).
func StripFrontmatter(data []byte) []byte {
	bodyStart, _, ok := frontmatterEnd(data)
	if !ok {
		return data
	}
	return data[bodyStart:]
}

// frontmatterEnd locates the YAML frontmatter block at the start of data.
// Returns fmEnd (end of the YAML content — i.e. start of the closing "---"
// line, exclusive), bodyStart (index after the closing delimiter line and
// one optional blank line), and ok=true if a valid block was found.
func frontmatterEnd(data []byte) (bodyStart, fmEnd int, ok bool) {
	delim := []byte(frontmatterDelim)
	if !bytes.HasPrefix(data, delim) {
		return 0, 0, false
	}
	rest := data[len(delim):]
	firstNL := bytes.IndexByte(rest, '\n')
	if firstNL < 0 {
		return 0, 0, false
	}
	if len(bytes.TrimRight(rest[:firstNL], "\r")) > 0 {
		return 0, 0, false
	}
	offset := len(delim) + firstNL + 1
	for offset < len(data) {
		nl := bytes.IndexByte(data[offset:], '\n')
		var line []byte
		if nl < 0 {
			line = data[offset:]
		} else {
			line = data[offset : offset+nl]
		}
		if bytes.Equal(bytes.TrimRight(line, "\r"), delim) {
			fmEnd = offset
			if nl < 0 {
				bodyStart = len(data)
			} else {
				bodyStart = offset + nl + 1
			}
			if bodyStart < len(data) && data[bodyStart] == '\n' {
				bodyStart++
			} else if bodyStart+1 < len(data) && data[bodyStart] == '\r' && data[bodyStart+1] == '\n' {
				bodyStart += 2
			}
			return bodyStart, fmEnd, true
		}
		if nl < 0 {
			return 0, 0, false
		}
		offset += nl + 1
	}
	return 0, 0, false
}
