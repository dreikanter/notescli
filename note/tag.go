package note

import (
	"bytes"
)

// extractHashtags scans body text and returns hashtag tokens (without the
// leading '#'), preserving source order and including duplicates. Rules:
//   - Lines whose first non-whitespace character is '#' are skipped (headings).
//   - Fenced code blocks (``` on a line, optionally indented, with optional
//     info string) are skipped until the next fence line.
//   - Inline backtick spans on a single line are skipped.
//   - A '#' preceded by a word character ([A-Za-z0-9_]) is not a tag.
//   - Tag characters are [A-Za-z0-9_-]; other characters terminate a tag.
func extractHashtags(body []byte) []string {
	var out []string
	inFence := false

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

		trim := 0
		for trim < len(line) && (line[trim] == ' ' || line[trim] == '\t') {
			trim++
		}

		if trim+3 <= len(line) && line[trim] == '`' && line[trim+1] == '`' && line[trim+2] == '`' {
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
			if j > 0 && isWordByte(line[j-1]) {
				continue
			}
			k := j + 1
			for k < len(line) && isTagByte(line[k]) {
				k++
			}
			if k > j+1 {
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
