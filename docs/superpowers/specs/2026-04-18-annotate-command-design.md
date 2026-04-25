# Design: `notes annotate <ref>` — AI-generated frontmatter

**Issue:** #105 — Annotate a note using AI

## Problem

Notes are created with minimal metadata. Users want the `title`, `description`, and `tags` fields filled in without retyping them by hand. The Claude Code CLI is already installed on the developer's machine; we can shell out to it non-interactively and merge its output into the note's frontmatter.

## Solution

Add a new command:

```
notes annotate <ref> [--model <name>]
```

It resolves the ref (same rules as `read`/`edit`/`update`), shells out to `claude` for structured metadata, and rewrites the note's frontmatter — filling only previously-empty fields.

## Behavior

1. Resolve `<ref>` via `note.ResolveRef`. Read the file, parse existing frontmatter, strip the body.
2. Compute the set of empty fields among `{title, description, tags}`. `tags` counts as empty when the slice is empty.
3. If no fields are empty, print the note path and return (no-op).
4. If the body is empty, return an error: `"note has no body content to annotate"`.
5. Build a JSON schema containing only the empty fields. Invoke `claude` with the schema, the fixed instructions, and the body as the prompt.
6. Parse the JSON response. Merge the returned values into the existing frontmatter (the non-empty fields are untouched).
7. Rewrite the file using the same "tmp file + rename" pattern as `update.go`. The filename does not change (slug untouched).
8. Print the note's absolute path (matches `update`).

## Scope of changes

Fields the command can generate: `title`, `description`, `tags`. Out of scope:

- `slug` — would rename the file on disk; user decision.
- `public` — privacy-sensitive signal; user decision.

Merge policy: **non-destructive**. A field that already has a value is never touched. There is no `--overwrite` flag in v1.

## Claude CLI invocation

```
claude -p --model <model> \
  --output-format json \
  --json-schema '<schema>' \
  --append-system-prompt '<instructions>' \
  '<note body>'
```

- `--model` defaults to `claude-haiku-4-5` and is overridable via the `--model` flag. No env var, no config file.
- `--output-format json` combined with `--json-schema` forces structured output we can parse directly.
- Schema is built at runtime from the set of empty fields:
  ```json
  {"type":"object","properties":{
    "title":{"type":"string"},
    "description":{"type":"string"},
    "tags":{"type":"array","items":{"type":"string"}}
  },"required":["<only empty fields>"],"additionalProperties":false}
  ```
  If only `tags` is empty, only `tags` appears in `properties` and `required`.
- The system prompt instructs Claude to produce concise frontmatter metadata for a personal note and to return only the requested fields.
- Binary resolved via `exec.LookPath("claude")`. If not found: `"claude CLI not found in PATH"`.
- `claude -p --output-format json` emits a JSON envelope on stdout. The exact shape of that envelope must be confirmed during implementation — run the command once by hand with a trivial schema, inspect stdout, and code the parser against the observed shape. The command extracts the schema-validated payload from the envelope and unmarshals it into a struct with `title`, `description`, `tags`. If extraction or unmarshalling fails, the parse error surfaces clearly and the file is untouched.

## Files

### New: `internal/cli/annotate.go`

- `annotateCmd` cobra command with `cobra.ExactArgs(1)` and `--model` flag.
- Package-level `claudeBinary = "claude"` (string) — swappable in tests (pattern from `edit.go`).
- Helpers:
  - `buildAnnotateSchema(empty []string) string`
  - `runClaude(bin, model, prompt string) ([]byte, error)` — executes and returns stdout.
  - `parseAnnotation(raw []byte, empty []string) (note.FrontmatterFields, error)` — unmarshals the envelope, then the nested fields.
  - `mergeAnnotation(existing note.FrontmatterFields, generated note.FrontmatterFields, empty []string) note.FrontmatterFields`

The `annotate` command does not call `update.go` directly; it implements its own read/merge/write because the semantics differ (only fill empties, no filename rename, no flag parsing).

### New: `internal/cli/annotate_test.go`

Follow the fake-binary pattern from `edit_test.go`. A helper writes a shell script named `claude` that echoes a canned JSON response, and the test overrides `claudeBinary` to point at that script.

Cases:

| Test | Scenario |
| --- | --- |
| `TestAnnotateFillsEmptyFields` | Note has only title set; script returns description+tags; file ends up with all three. |
| `TestAnnotateSkipsFilledFields` | Note has title+description+tags filled; command is a no-op, script is not called. |
| `TestAnnotateNoBodyErrors` | Note has frontmatter only; command returns the "no body" error, script not invoked. |
| `TestAnnotateClaudeNotFound` | `claudeBinary` points to a non-existent path; command returns the not-found error. |
| `TestAnnotateClaudeNonZeroExit` | Fake script `exit 1`; command surfaces stderr. |
| `TestAnnotateMalformedJSON` | Fake script prints invalid JSON; command errors; file untouched. |
| `TestAnnotateSchemaContainsOnlyEmptyFields` | Inspects the args passed to the fake script; schema's `required` matches the empty set. |
| `TestAnnotateModelFlag` | `--model foo` propagates as `--model foo` in the invocation args. |
| `TestAnnotatePreservesBody` | Body text after frontmatter is byte-identical after rewrite. |

No network calls; no real `claude` invocation.

### Updated: `CHANGELOG.md`

One entry for the next patch version, referencing the PR.

### Updated: `README.md`

Add `notes annotate <ref>` to the usage section, alongside `edit` and `update`.

## Error behavior summary

| Condition | Message |
| --- | --- |
| Ref not found | (delegated to `note.ResolveRef`) |
| All target fields already filled | (no error; prints path; exits 0) |
| Body empty after strip | `"note has no body content to annotate"` |
| `claude` not on PATH | `"claude CLI not found in PATH"` |
| `claude` returns non-zero | `"claude failed: <stderr>"` |
| Response not valid JSON or fails schema | `"cannot parse claude response: <err>"` |

In all error cases, the note file is left untouched.

## Out of scope

- Interactive preview or confirmation (`--dry-run`)
- `--overwrite` flag
- Slug or public field generation
- Prompt customization by the user
- `NOTES_ANNOTATE_MODEL` env var or config file
- Streaming/progress UI
- Batch annotation over multiple refs
