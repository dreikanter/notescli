package note

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

// ExtractTags scans the note store under root and returns a sorted,
// deduplicated list of tags. Sources: frontmatter `tags:` fields and body
// hashtags (#word) in the prose. File reads run concurrently across
// runtime.NumCPU() workers. Returns a nil slice for an empty store. If any
// file read fails, the first such error is returned after all workers drain.
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

	jobs := make(chan Note)
	results := make(chan map[string]struct{}, workers)
	errCh := make(chan error, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make(map[string]struct{})
			var workerErr error
			for n := range jobs {
				data, err := os.ReadFile(filepath.Join(root, n.RelPath))
				if err != nil {
					if workerErr == nil {
						workerErr = err
					}
					continue
				}
				for _, t := range ParseFrontmatterFields(data).Tags {
					if t != "" {
						local[t] = struct{}{}
					}
				}
				for _, t := range extractHashtags(StripFrontmatter(data)) {
					local[t] = struct{}{}
				}
			}
			results <- local
			errCh <- workerErr
		}()
	}

	for _, n := range notes {
		jobs <- n
	}
	close(jobs)
	wg.Wait()
	close(results)
	close(errCh)

	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	merged := make(map[string]struct{})
	for local := range results {
		for t := range local {
			merged[t] = struct{}{}
		}
	}

	out := make([]string, 0, len(merged))
	for t := range merged {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}

// extractHashtags scans body text and returns hashtag tokens (without the
// leading '#'), preserving source order and including duplicates. Rules:
//   - Lines whose first non-whitespace content is a run of '#' followed by
//     whitespace or end-of-line are Markdown headings and are skipped entirely.
//   - Fenced code blocks (``` on a line, optionally indented, with optional
//     info string) are skipped until the next fence line. Tilde fences (~~~)
//     are not recognised.
//   - Inline backtick spans on a single line are skipped. An unclosed
//     backtick suppresses hashtags for the remainder of its line.
//   - A '#' preceded by a word byte ([A-Za-z0-9_]) is not a tag. The check is
//     byte-level, so hashtags adjacent to non-ASCII prose (e.g. `café#bar`)
//     may still be extracted.
//   - Tag characters are [A-Za-z0-9_-]; other bytes terminate a tag. A bare
//     '#' with no following tag byte produces no output.
func extractHashtags(body []byte) []string {
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
