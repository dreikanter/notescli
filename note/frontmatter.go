package note

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

const frontmatterDelim = "---"

func yamlKindName(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	}
	return fmt.Sprintf("kind(%d)", k)
}

// Frontmatter holds optional fields for note frontmatter.
// Adding a field is a one-line struct addition — no other changes required.
type Frontmatter struct {
	Title       string               `yaml:"title,omitempty"`
	Slug        string               `yaml:"slug,omitempty"`
	Type        string               `yaml:"type,omitempty"`
	Date        time.Time            `yaml:"date,omitempty"`
	Tags        []string             `yaml:"tags,omitempty"`
	Aliases     []string             `yaml:"aliases,omitempty"`
	Description string               `yaml:"description,omitempty"`
	Public      bool                 `yaml:"public,omitempty"`
	Extra       map[string]yaml.Node `yaml:"-"`
}

// IsZero reports whether f has no fields set, including Extra.
func (f Frontmatter) IsZero() bool {
	return f.Title == "" && f.Slug == "" && f.Type == "" && f.Date.IsZero() &&
		len(f.Tags) == 0 && len(f.Aliases) == 0 && f.Description == "" && !f.Public && len(f.Extra) == 0
}

// UnmarshalYAML decodes a mapping node into f. Reserved keys populate the
// typed fields; unknown keys are captured in f.Extra as yaml.Node values.
// Duplicate top-level keys and non-scalar keys are rejected.
func (f *Frontmatter) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("frontmatter: expected mapping, got %s", yamlKindName(node.Kind))
	}
	seen := make(map[string]bool, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			return fmt.Errorf("frontmatter: non-scalar key (%s)", yamlKindName(key.Kind))
		}
		if seen[key.Value] {
			return fmt.Errorf("frontmatter: duplicate key %q", key.Value)
		}
		seen[key.Value] = true
		switch key.Value {
		case "title":
			if err := value.Decode(&f.Title); err != nil {
				return fmt.Errorf("frontmatter title: %w", err)
			}
		case "slug":
			if err := value.Decode(&f.Slug); err != nil {
				return fmt.Errorf("frontmatter slug: %w", err)
			}
		case "type":
			if err := value.Decode(&f.Type); err != nil {
				return fmt.Errorf("frontmatter type: %w", err)
			}
		case "date":
			if err := value.Decode(&f.Date); err != nil {
				return fmt.Errorf("frontmatter date: %w", err)
			}
		case "tags":
			if err := value.Decode(&f.Tags); err != nil {
				return fmt.Errorf("frontmatter tags: %w", err)
			}
		case "aliases":
			if err := value.Decode(&f.Aliases); err != nil {
				return fmt.Errorf("frontmatter aliases: %w", err)
			}
		case "description":
			if err := value.Decode(&f.Description); err != nil {
				return fmt.Errorf("frontmatter description: %w", err)
			}
		case "public":
			if err := value.Decode(&f.Public); err != nil {
				return fmt.Errorf("frontmatter public: %w", err)
			}
		default:
			if f.Extra == nil {
				f.Extra = make(map[string]yaml.Node)
			}
			f.Extra[key.Value] = *value
		}
	}
	return nil
}

// MarshalYAML composes a mapping node with reserved fields first (in fixed
// order) and Extra keys alpha-sorted. Zero-valued reserved fields are omitted,
// matching the `omitempty` struct-tag discipline.
func (f Frontmatter) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	appendString := func(key, value string) {
		if value == "" {
			return
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
		)
	}
	appendList := func(key string, value []string) {
		if len(value) == 0 {
			return
		}
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, v := range value {
			seq.Content = append(seq.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v},
			)
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			seq,
		)
	}
	appendBool := func(key string, value bool) {
		if !value {
			return
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		)
	}
	appendTime := func(key string, value time.Time) {
		if value.IsZero() {
			return
		}
		// Date-only values (midnight UTC) serialize as YYYY-MM-DD so inputs
		// written as `date: 2026-04-22` round-trip without gaining a time
		// component. Values with a non-zero time-of-day use RFC3339.
		var formatted string
		if value.Hour() == 0 && value.Minute() == 0 && value.Second() == 0 &&
			value.Nanosecond() == 0 && value.Location() == time.UTC {
			formatted = value.Format("2006-01-02")
		} else {
			formatted = value.Format(time.RFC3339)
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!timestamp", Value: formatted},
		)
	}

	appendString("title", f.Title)
	appendString("slug", f.Slug)
	appendString("type", f.Type)
	appendTime("date", f.Date)
	appendList("tags", f.Tags)
	appendList("aliases", f.Aliases)
	appendString("description", f.Description)
	appendBool("public", f.Public)

	if len(f.Extra) > 0 {
		keys := make([]string, 0, len(f.Extra))
		for k := range f.Extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := f.Extra[k]
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k},
				&v,
			)
		}
	}

	return node, nil
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
