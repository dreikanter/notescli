package note

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

// ExtractTags scans the note store under root and returns a sorted,
// deduplicated, lowercased list of tags. Sources: frontmatter `tags:` fields
// and body hashtags (#word) in the prose. File reads run concurrently across
// runtime.NumCPU() workers. Returns a nil slice for an empty store.
// A per-note frontmatter parse error is written to stderr and the
// note's frontmatter tags are skipped (body hashtags are still collected).
// Any file-read error aborts the scan.
func ExtractTags(root string) ([]string, error) {
	notes, err := Scan(root)
	if err != nil {
		return nil, err
	}
	if len(notes) == 0 {
		return nil, nil
	}

	workers := runtime.NumCPU()
	if workers > len(notes) {
		workers = len(notes)
	}

	g, ctx := errgroup.WithContext(context.Background())
	jobs := make(chan Note)
	var mu sync.Mutex
	merged := make(map[string]struct{})

	g.Go(func() error {
		defer close(jobs)
		for _, n := range notes {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case jobs <- n:
			}
		}
		return nil
	})

	for i := 0; i < workers; i++ {
		g.Go(func() error {
			local := make(map[string]struct{})
			for n := range jobs {
				path := filepath.Join(root, n.RelPath)
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				fm, body, parseErr := ParseNote(data)
				if parseErr != nil {
					fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, parseErr)
				}
				for _, t := range fm.Tags {
					if t != "" {
						local[strings.ToLower(t)] = struct{}{}
					}
				}
				for _, t := range ExtractHashtags(body) {
					local[strings.ToLower(t)] = struct{}{}
				}
			}
			mu.Lock()
			for t := range local {
				merged[t] = struct{}{}
			}
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(merged))
	for t := range merged {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}

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
