# Design: Add `tags` command

**Date:** 2026-04-18

## Problem

There's no way to see what tags exist in the note store without grepping
frontmatter by hand. Users set frontmatter tags (`tags: [a, b]`) and also drop
inline hashtags (`#foo`) in note bodies, but neither is discoverable from the
CLI. A `tags` command should list the union of both, so it can be piped into
`fzf`, `grep`, or used as input to other commands.

Performance matters: on a store with 10k+ notesctl the command must finish quickly
enough to feel instant when piped.

## Design

### Command shape

New top-level command, no args, no flags:

```
notesctl tags
```

Prints unique tags, one per line, sorted alphabetically (byte order). Exits 0
on an empty store (prints nothing). No header, no counts, no formatting — pure
pipe-friendly output.

No filter flags (`--type`, `--slug`, `--today`, `--tag`) in v1. Scope is
always the entire store. If a future use case appears, filter flags can be
added without breaking the current output shape.

### Sources of tags

Two sources, merged into one deduplicated set:

1. **Frontmatter** — the existing `tags: [a, b]` field, parsed by
   `note.ParseFrontmatterFields`.
2. **Body hashtags** — `#word` tokens in the note body (after frontmatter
   stripping), with the leading `#` dropped.

### Hashtag extraction rules

A hashtag is `#` followed by one or more characters from `[A-Za-z0-9_-]`,
subject to:

- **Not a Markdown heading** — a line is treated as a heading (and skipped
  entirely) when its first non-whitespace content is a run of one or more
  `#` characters followed by whitespace or end-of-line (`# foo`, `## foo`,
  `   # foo`). A leading `#` with no space after it (e.g. `#alpha and
  #beta`) is not a heading and its hashtags are extracted normally.
- **Not preceded by a word character** — so `foo#bar` is not a tag, but
  `(foo) #bar` is. "Word character" means `[A-Za-z0-9_]`.
- **Not inside a fenced code block** — any region bounded by a line whose
  first non-whitespace content is ` ``` ` (three backticks, optional info
  string after). Opening and closing fences on their own lines. Nested fences
  are not handled (Markdown itself doesn't really nest them).
- **Not inside inline code** — any span between matching backticks on the
  same line. Simple: toggle a flag on each backtick encountered inside a
  non-code-block line.

The character class intentionally excludes `/` and `.` — `#foo/bar` yields
only `foo`. This matches the most common hashtag convention and avoids
tangling with file-path-like strings.

Extraction is done with a hand-rolled byte scanner (not regex) — it's both
faster and makes the code-block / inline-code state tracking natural.

### Extraction pipeline

1. `note.Scan(root)` enumerates notesctl (existing function; directory walk
   only, no file reads).
2. A bounded worker pool — `runtime.NumCPU()` goroutines — reads files and
   extracts tags in parallel. The workload is I/O-bound per file but CPU-
   bound across the scanner, so parallelism helps in both regimes.
3. Per-file work:
   - `os.ReadFile` the note.
   - `note.ParseFrontmatterFields` → collect frontmatter tags into a local
     set.
   - `note.StripFrontmatter` → run the body through the byte scanner,
     adding each hashtag to the same local set.
4. Workers send their local sets to a single collector goroutine, which
   merges them into a global `map[string]struct{}`.
5. Extract keys, `sort.Strings`, print one per line.

No caching. No index. One-shot scan per invocation. Matches the "no
database, no index" ethos of the rest of the codebase.

### File layout

- `note/tags.go` — new file. Contains `ExtractTags(root string) ([]string,
  error)` (parallel scan + merge) and `extractHashtags(body []byte) []string`
  (byte scanner). Unit tests in `note/tags_test.go`.
- `internal/cli/tags.go` — new file. Thin cobra wrapper that calls
  `note.ExtractTags(mustNotesPath())` and prints results. Tests in
  `internal/cli/tags_test.go`.

`ExtractTags` lives in the `note` package so it can be reused later (e.g., by
a future autocomplete or `tag --count` variant) without going through the CLI
layer.

### Tests

Unit tests for the byte scanner cover:

| Case | Expected |
|------|----------|
| `hello #world` | `[world]` |
| `#heading at start of line` | `[]` |
| `## second-level heading` | `[]` |
| `foo#bar` (no preceding space) | `[]` |
| `` `inline #code` not a tag`` | `[]` |
| Fenced block with `#tag` inside | `[]` |
| Mixed: frontmatter + body hashtags + code block | merged, deduped |
| `#foo/bar` | `[foo]` (slash terminates) |
| `#a-b_c`, `#123` | `[a-b_c, 123]` |

Integration tests (`internal/cli/tags_test.go`) cover:

| Case | Expected |
|------|----------|
| Empty store | exit 0, no output |
| Store with frontmatter-only tags | sorted unique list |
| Store with body-hashtag-only tags | sorted unique list |
| Store with both, overlapping | merged, deduplicated |
| Notes containing code blocks with `#` | code-block content ignored |

### Non-goals

- No `--count` flag.
- No source-splitting flag (frontmatter vs. body).
- No filter flags.
- No caching layer.
- No hierarchical tags (`#parent/child` collapses to `parent`).

### CHANGELOG

Add an entry under `v0.1.67` (next patch from current `v0.1.66`) referencing
the PR. Entry text along the lines of: "Add `tags` command to list tags from frontmatter and body hashtags
frontmatter and body hashtags ([#N])."
