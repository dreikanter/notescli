package note

import (
	"bytes"
	"sort"
	"strings"
)

// ExtractHashtags scans body text and returns hashtag tokens (without the
// leading '#'), preserving source order and including duplicates. Rules:
//   - Lines whose first non-whitespace content is a run of '#' followed by
//     whitespace or end-of-line are Markdown headings and are skipped entirely.
//   - Fenced code blocks (``` on a line, optionally indented, with optional
//     info string) are skipped until the next fence line. Tilde fences (~~~)
//     are not recognised.
//   - Inline backtick spans on a single line are skipped. An unclosed
//     backtick suppresses hashtags for the remainder of its line.
//   - A '#' preceded by a word byte ([A-Za-z0-9_]) or a URL-path byte
//     (`/`, `:`, `.`, `?`, `=`, `&`, `~`, `#`) is not a tag. This prevents
//     matches inside URLs (`example.com/#anchor`) and inline chains
//     (`#one#two`). The check is byte-level, so hashtags adjacent to
//     non-ASCII prose (e.g. `café#bar`) may still be extracted.
//   - Tag characters are [A-Za-z0-9_-]; other bytes terminate a tag. A bare
//     '#' with no following tag byte produces no output. A tag immediately
//     followed by another '#' (e.g. `#one#two`) is rejected.
func ExtractHashtags(body []byte) []string {
	var out []string
	inFence := false
	fence := []byte("```")

	for len(body) > 0 {
		nl := bytes.IndexByte(body, '\n')
		var line []byte
		if nl < 0 {
			line = body
			body = nil
		} else {
			line = body[:nl]
			body = body[nl+1:]
		}
		line = bytes.TrimRight(line, "\r")

		trim := 0
		for trim < len(line) && (line[trim] == ' ' || line[trim] == '\t') {
			trim++
		}

		if bytes.HasPrefix(line[trim:], fence) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		if trim < len(line) && line[trim] == '#' {
			k := trim
			for k < len(line) && line[k] == '#' {
				k++
			}
			if k == len(line) || line[k] == ' ' || line[k] == '\t' {
				continue
			}
		}

		inInline := false
		for j := 0; j < len(line); j++ {
			c := line[j]
			if c == '`' {
				inInline = !inInline
				continue
			}
			if c != '#' || inInline {
				continue
			}
			if j > 0 && !isHashtagLeadingByte(line[j-1]) {
				continue
			}
			k := j + 1
			for k < len(line) && isTagByte(line[k]) {
				k++
			}
			if k > j+1 && (k == len(line) || line[k] != '#') {
				out = append(out, string(line[j+1:k]))
			}
			j = k - 1
		}
	}
	return out
}

func isTagByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-'
}

func isWordByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// isHashtagLeadingByte reports whether c may legally precede a '#' that
// starts a hashtag. Word bytes (so `foo#bar` is not a tag) and URL-path
// bytes (so `example.com/#anchor` is not a tag) are excluded.
func isHashtagLeadingByte(c byte) bool {
	if isWordByte(c) {
		return false
	}
	switch c {
	case '/', ':', '.', '?', '=', '&', '~', '#':
		return false
	}
	return true
}

// hasAllTags reports whether every entry in required appears in noteTags,
// case-insensitively. Used by both MemStore and OSStore for WithTag filtering.
func hasAllTags(noteTags []string, required []string) bool {
	set := make(map[string]struct{}, len(noteTags))
	for _, t := range noteTags {
		set[strings.ToLower(t)] = struct{}{}
	}
	for _, r := range required {
		if _, ok := set[strings.ToLower(r)]; !ok {
			return false
		}
	}
	return true
}

// computeMergedTags builds the sorted, lowercased, deduplicated union of
// frontmatter tags and body hashtags. bodyHashtags is assumed already
// lowercased (as produced by normalizeHashtags). Returns nil when the
// union is empty.
func computeMergedTags(fmTags, bodyHashtags []string) []string {
	set := make(map[string]struct{}, len(fmTags)+len(bodyHashtags))
	for _, t := range fmTags {
		if t == "" {
			continue
		}
		set[strings.ToLower(t)] = struct{}{}
	}
	for _, t := range bodyHashtags {
		set[t] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// normalizeHashtags lowercases and deduplicates a hashtag list from
// ExtractHashtags into the canonical form merged into Meta.Tags by
// OSStore.
func normalizeHashtags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(raw))
	for _, t := range raw {
		if t == "" {
			continue
		}
		set[strings.ToLower(t)] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
