# Changelog

## [0.1.93] - 2026-04-22

### Added

- `note.Note.Time()` parses the UID-derived `Date` prefix (YYYYMMDD) into a `time.Time` at midnight UTC, returning `false` on malformed input. `note.ResolveEntryDate(n Note, fm Frontmatter, fi fs.FileInfo)` picks a canonical date for a note and returns its source label, walking the documented priority: UID-derived date (`"uid"`) → frontmatter `date` (`"frontmatter"`) → file mtime (`"mtime"`). Pass `fi == nil` to skip the mtime fallback. Downstream consumers (notes-pub, notes-view) no longer need to re-implement the chain ([#149])

## [0.1.92] - 2026-04-22

### Added

- `note.ScanOptions{Strict bool}` and a variadic `Scan(root string, opts ...ScanOptions) ([]Note, error)` signature let callers opt into a lenient walk. The default (no options, or `Strict: true`) preserves the existing YYYY/MM/*.md discipline; `Strict: false` walks every `.md` file under root with `filepath.WalkDir` regardless of nesting depth or parent-directory naming, matching the layout downstream tools like notes-view consume. Existing `Scan(root)` callers are unaffected ([#141])

## [0.1.91] - 2026-04-22

### Added

- `note.Frontmatter` now has a reserved `Aliases []string` field (`yaml:"aliases,omitempty"`). Notes whose `aliases:` previously landed in `Frontmatter.Extra` now populate the typed field, so downstream publishers (notes-pub permalink redirects, notes-view rename-history resolution) no longer need to decode the `yaml.Node` themselves. notes-cli does not itself consume `aliases` yet; the field is reserved to stabilize the contract and avoid future collisions — see `SCHEMA.md` ([#139])

## [0.1.90] - 2026-04-22

### Added

- `note.Frontmatter` now has a reserved `Date time.Time` field (`yaml:"date,omitempty"`). Notes whose `date:` previously landed in `Frontmatter.Extra` now populate the typed field, and consumers no longer need to decode the `yaml.Node` themselves. Round-trip preserves the input format: date-only values (midnight UTC) serialize as `YYYY-MM-DD`; values with a non-zero time-of-day serialize as RFC3339. Consumers that need a date when `date:` is absent should fall back to the UID-derived date from the filename prefix, then file mtime — see `SCHEMA.md` ([#146])

## [0.1.89] - 2026-04-22

### Added

- `note.ExtractHashtags` is now exported (previously unexported `extractHashtags`). Downstream tools (notes-pub, notes-view) can reuse the same body-hashtag extraction rules — fenced code blocks, inline backticks, URL anchors, chained hashes — instead of re-implementing them ([#136])
- `note.IsID` reports whether a string is a valid notes-cli note ID (non-empty, ASCII digits only). Replaces the ad-hoc `isNoteID` / `IsUID` helpers currently duplicated in consumer projects ([#136])
- `note.NormalizeSlug` returns an ASCII-lowercase, URL-safe form of a string (non-alphanumeric runs collapse to `-`; leading/trailing dashes stripped). Shared normalization contract for filenames and URL path segments ([#136])
- `note.DeriveSlug` returns the normalized slug for a note using the fallback chain: frontmatter slug → stem with id prefix stripped → empty. Consolidates the slug-resolution logic that consumers were each inventing ([#136])

## [0.1.86] - 2026-04-21

### Changed

- `note.ResolveRef` and `note.ResolveRefDate` now return `(Note, error)` instead of `(*Note, error)`, matching the value-return convention of the other store APIs. Callers accessing fields (`n.RelPath`, `n.ID`, etc.) need no changes; nil-vs-zero ambiguity is gone ([#132])
- `note.Scan` now logs a stderr warning and skips unreadable year/month subdirectories instead of aborting the whole scan. One permission glitch on a single month directory no longer breaks `ls`, `tags`, or `resolve`; root-level `ReadDir` failures still surface as hard errors ([#132])
- `Frontmatter.MarshalYAML` builds `yaml.Node` values directly (`Tag: "!!str"` / `"!!bool"` / `"!!seq"`) instead of routing strings, bools, and string lists through `(*yaml.Node).Encode` with panic-on-error wrappers. Output is byte-identical; the four impossible-to-reach panic paths are gone ([#132])

## [0.1.85] - 2026-04-21

### Changed

- `notes ls --tag`, `notes read --tag`, `notes append --tag`, and `notes resolve --tag` now match body hashtags (`#tag`) in addition to frontmatter `tags:`, mirroring the sources already used by `notes tags`. Tag-based filtering no longer silently ignores inline hashtags ([#131])

## [0.1.84] - 2026-04-20

### Changed

- `notes read`, `notes append`, and `notes resolve` now include the effective filters in the "no notes found" error (e.g. `no notes found matching filters: type=[todo] today=true`) so you can tell which filter narrowed too far ([#115])
- `notes new` and `notes new-todo` now inherit the notes-store root's directory permissions when creating date subdirectories, instead of hardcoding `0o755`, so a `0o700` root is no longer silently widened ([#115])

### Removed

- `notes read --no-frontmatter` no longer has a `-F` short form. Use the long flag ([#115])

## [0.1.83] - 2026-04-20

### Changed

- `note.NextID` now flocks the store root directory instead of a sibling `id.json.lock` file, so no lockfile artifact is left behind after `notes new` / `notes new-todo` runs. Serialization semantics are unchanged ([#115])
- `notes annotate --timeout 0` now disables the deadline (previously it caused the command to fail immediately), mirroring `--max-chars 0 = no limit` ([#115])

## [0.1.82] - 2026-04-20

### Changed

- `note.NextID` now serializes the id.json read-modify-write across processes via an exclusive `flock` on a sibling `id.json.lock`, so parallel `notes new` / `notes new-todo` runs can no longer duplicate IDs ([#115])
- `notes grep` and `notes rg` propagate the child process's exit code instead of collapsing every failure to `1`: "no match" (exit 1) is now distinguishable from real tool errors (exit 2+) by the caller ([#115])
- `notes annotate` now runs the `claude` CLI with a context-bound timeout (default 60s, configurable via `--timeout`), so a hung Claude binary no longer hangs the command indefinitely ([#115])

## [0.1.81] - 2026-04-20

### Changed

- `notes update --sync-filename` now reserves the target atomically with `os.Link` + `os.Remove`, closing a TOCTOU between `os.Stat` and `os.Rename` that could silently clobber a file created between the two syscalls ([#115])
- `mustNotesPath` replaced by `notesRoot() (string, error)`: the notes-store resolution no longer calls `os.Exit(1)` from inside `RunE` handlers, so errors now flow through Cobra's normal error pipeline (and respect `SilenceUsage`). Error output and exit code are unchanged ([#115])
- `notes annotate` error messages are more useful when the `claude` CLI fails with empty stderr: the exit code and the first 500 bytes of stdout are now included, replacing the previous opaque `exit status 1`. Successful runs and failures that write to stderr are unchanged ([#115])
- `notes new --public` and `--private` are now mutually exclusive (matching `notes update`). Passing both returns an error instead of silently letting `--private` override `--public`; the old silent-override logic is gone ([#115])

### Removed

- `notes new-todo --force` flag has been removed. Its help text promised to "regenerate today's todo even if it exists," but it actually allocated a new ID and wrote a *second* todo for the same day, which was never the intended behavior. If today's todo already exists, `notes new-todo` now unconditionally returns its path ([#115])

## [0.1.80] - 2026-04-20

### Changed

- `notes grep` no longer injects `-i`; searches are case-sensitive by default. Pass `-i` explicitly for case-insensitive search ([#115])
- `notes rg` now only injects `--glob *.md`; the previously forced `--sortr path`, `--heading`, `--no-line-number`, and `--ignore-case` defaults are gone, so the subcommand behaves like plain `rg` restricted to Markdown files. Pass those flags explicitly if you want the old output style ([#115])
- `ValidateSlug` now rejects anything outside `[A-Za-z0-9_-]` (previously only all-digit slugs were rejected), so slugs containing `/`, `\`, `.`, whitespace, or control characters can no longer reach `NoteFilename` and corrupt filesystem paths or the filename's dot-suffix cache ([#115])

## [0.1.79] - 2026-04-20

### Changed

- Internal cleanups from the code-review follow-up list (no user-visible behavior change): `notes update`'s "at least one flag" guard now walks `cmd.LocalFlags()` instead of a hand-maintained flag-name slice that had to stay in sync with registrations; `ParseTask`'s regex requires exactly one marker character (`[ ]`, `[x]`, …) instead of accepting zero-or-one, so stray `[]` no longer parses as a task ([#115])

## [0.1.78] - 2026-04-20

### Changed

- Drop the hardcoded `~/notes` fallback when resolving the notes store path. If neither `--path` nor `$NOTES_PATH` is set, `notes` now exits with `no notes store configured. Set $NOTES_PATH or pass --path` instead of silently scanning a `~/notes` directory that may exist for unrelated reasons. Set `NOTES_PATH` once (e.g. `export NOTES_PATH=~/notes`) to restore the previous behavior ([#123], [#117])

## [0.1.77] - 2026-04-20

### Changed

- `notes tags` output and `--tag` filter comparisons are now case-insensitive: tags are lowercased when extracted from frontmatter and body hashtags, and both sides are lowercased when matching `--tag` values against note frontmatter. On-disk frontmatter is left unchanged ([#120])

## [0.1.76] - 2026-04-20

### Changed

- Tighter inline hashtag matching: a `#` preceded by a URL-path byte (`/`, `:`, `.`, `?`, `=`, `&`, `~`, `#`) no longer starts a tag, so fragments like `example.com/#anchor` are left alone; and a tag immediately followed by another `#` (e.g. `#one#two`) is rejected to avoid mid-word false positives ([#119])

## [0.1.75] - 2026-04-20

### Changed

- Remove `signal.Reset(syscall.SIGPIPE)` from `main`: empirically verified a no-op on Go 1.25 (SIGPIPE behavior is identical with or without the call on both stdout and non-stdout fds), so the line and its `os/signal`/`syscall` imports are dead code. Go's default handler (terminate on fd 1/2, return EPIPE elsewhere) already provides the commented-for behavior ([#118])

## [0.1.74] - 2026-04-20

### Changed

- Small code-review follow-ups: `--path` help now documents the `$NOTES_PATH` / `~/notes` default; `notes resolve` Long help clarifies that `--today` is the only filter flag that can combine with a positional argument; `grep`/`rg` subcommands accept `-h` (not just `--help`) for help; frontmatter-parse warnings go to stderr directly instead of through `log.Printf` (no more timestamp prefix); `writeAtomic` is now shared across `update`, `annotate`, `append`, and the prev-todo rewrite in `new-todo` so partial writes never leave a corrupted file behind ([#116])

## [0.1.73] - 2026-04-19

### Changed

- Note frontmatter format: unknown keys are now preserved through `notes update` and any other format-rewriting command (via `Frontmatter.Extra`), enabling downstream tools and users to add custom fields without waiting for a notes-cli release. `type` moves from filename-only to a typed frontmatter field (filename still cached as a `.type` dot-suffix). `KnownTypes`/`IsKnownType` renamed to `TypesWithSpecialBehavior`/`HasSpecialBehavior` — the list is now a soft registry, not a validation gate; any string is a valid `type` value. `notes update` no longer auto-renames on `--slug`/`--type` changes; use the new `--sync-filename` flag to explicitly reconcile the filename with frontmatter. A repo-root `SCHEMA.md` documents reserved frontmatter keys. See [design spec](docs/superpowers/specs/2026-04-19-notes-schema-protocol-design.md) and [#104]. ([#114])

## [0.1.72] - 2026-04-19

### Changed

- `notes update` and `notes annotate` now fail with a clear error when the target note has malformed frontmatter, instead of silently dropping bad fields and rewriting the file ([#112])
- `notes ls --tag` and `notes tags` log a per-note warning to stderr for any note with unparseable frontmatter and skip it, instead of silently treating it as tagless ([#112])
- Stricter frontmatter parsing: duplicate keys, non-mapping top-level documents, control characters, and type mismatches are now rejected at the document level; previously the parser preserved siblings of a bad field ([#112])
- CRLF line endings inside the note body are now preserved verbatim through read/write round-trips ([#112])

## [0.1.71] - 2026-04-19

### Changed

- Switch frontmatter (de)serialization to `gopkg.in/yaml.v3`: tags and strings containing `,`, `]`, `:`, or other special characters now round-trip safely through write → read, and adding a new frontmatter field no longer requires parser changes ([#110])

## [0.1.70] - 2026-04-19

### Added

- `notes annotate <ref>` command that uses Claude Code CLI to fill empty frontmatter fields (`title`, `description`, `tags`). Defaults to `claude-haiku-4-5`; override with `--model`. Non-destructive: existing field values are never overwritten. ([#109])

## [0.1.69] - 2026-04-18

### Added

- Add `tags` command that lists unique tags from frontmatter and body hashtags across the store ([#108])

## [0.1.68] - 2026-04-18

### Changed

- Make note resolution less surprising: all-digit queries only match IDs (no fallthrough), and substring matching targets the slug only ([#107])

## [0.1.67] - 2026-04-18

### Changed

- Change daily task tag format from `[daily]` to `#daily` ([#106])

## [0.1.66] - 2026-04-09

### Changed

- Update all references from `dreikanter/notescli` to `dreikanter/notes-cli` to match the renamed repository ([#102])

## [0.1.65] - 2026-04-05

### Changed

- Rewrite README intro to explain the project's purpose and scope ([#100])

## [0.1.64] - 2026-04-05

### Changed

- Detach GUI editors in `edit` command so control returns to terminal immediately; terminal editors (vim, nano, etc.) still run in foreground ([#99])

## [0.1.63] - 2026-04-05

### Changed

- Replace `bin/update` script with `make update` target ([#98])

## [0.1.62] - 2026-04-05

### Changed

- `resolve` with no arguments returns the most recent note ([#97])

## [0.1.60] - 2026-04-05

### Changed

- Extract shared filter helper (`addFilterFlags`, `readFilterFlags`, `applyFilters`) to eliminate duplicated filter pipeline across `ls`, `resolve`, `read`, and `append` ([#92])
- Normalize `--type` flag to `StringSlice` on `read` and `append` for consistency with `ls` and `resolve` ([#92])

## [0.1.59] - 2026-04-05

### Removed

- Remove redundant `path` command ([#93])

## [0.1.58] - 2026-04-05

### Added

- Add `--upsert` flag to `new` command for idempotent create-or-return semantics ([#90])

### Changed

- Remove note creation logic from `append` command; `append` now only appends to existing notes ([#90])

### Removed

- Remove `--create`, `--today` (as creation trigger), `--title`, `--description`, `--public`, `--private` flags from `append` ([#90])

## [0.1.57] - 2026-04-05

### Changed

- Simplify `ResolveRef` priority chain from 5 steps to 3: ID → type → path substring ([#88])

## [0.1.55] - 2026-04-05

### Changed

- Merge `latest` into `resolve`; `resolve` now accepts `--type`, `--slug`, `--tag` filter flags as an alternative to the positional argument ([#85])
- Unify `Use` line to `<id|type|query>` across all ref-accepting commands ([#88])

### Removed

- Remove `latest` command (use `resolve --type`, `resolve --slug`, `resolve --tag` instead) ([#85])

### Fixed

- Fix broken `edit` and `rm` tests after testdata rename in [#72] ([#85])

## [0.1.54] - 2026-04-05

### Changed

- Simplify `--slug` flag to single-value on `ls` and `resolve` commands ([#85])

### Fixed

- Fix broken tests for `edit` and `rm` commands after testdata rename in [#72] ([#85])

## [0.1.41] - 2026-04-05

### Added

- Add `edit` command to open a note in `$VISUAL` or `$EDITOR` ([#76])
- Add `rm` command for deleting notes by ref ([#77])
- Add `--type`, `--slug`, `--tag`, and `--today` filter flags to `read`; mutually exclusive with the positional ref argument ([#81])
- Add `--today` flag to `latest` command ([#82])
- Add `--public` and `--private` flags to `append`, gated on `--create`/`--today` ([#83])

### Changed

- `update` command now returns an error when called with no flags instead of silently rewriting the file unchanged ([#71])
- Reject conflicting `update` flags (`--slug`/`--no-slug`, `--type`/`--no-type`, `--tag`/`--no-tags`, `--public`/`--private`) instead of silently picking a winner ([#74])
- Clarify `latest` command description to distinguish it from `resolve` ([#79])

### Fixed

- Output absolute paths from `ls` to enable Unix pipelines like `notes ls | xargs notes read` ([#73])
- Fix ref resolution for all-digit slugs; reject all-digit slugs in `new` and `update` commands ([#72])
- Fix `new-todo` when no previous todo exists; creates an empty todo instead. `--force` works correctly when today's todo is the only one ([#75])
- Fix `ls --type` and `--slug` flags to accept multiple values, matching `latest` behavior ([#78])
- Fix `grep` and `rg` commands passing `--help` to the subprocess instead of showing notes-specific help; improve `Long` descriptions to document injected default flags ([#80])

## [0.1.40] - 2026-04-04

### Added

- Add `--today` flag to `resolve` command for date-based note existence checks ([#53])

## [0.1.39] - 2026-04-04

### Changed

- Remove default limit from `ls`; output all notes unless `--limit` is specified. Handle SIGPIPE for clean pipe behavior ([#52])

## [0.1.38] - 2026-04-04

### Added

- Add `--today` flag to `append` for daily note rotation: appends to today's matching note or creates a new one ([#51])

## [0.1.37] - 2026-03-30

### Fixed

- Trim whitespace from `resolve` query to prevent lookup failures from trailing spaces or newlines ([#48])
- Restrict note scanning to known `YYYY/MM/` directory structure ([#48])

## [0.1.36] - 2026-03-29

### Added

- Add `Slug` and `Public` fields to `FrontmatterFields`; extend parser and builder; sync `slug:` frontmatter when `--slug`/`--no-slug` is used in `update` ([#46])

## [0.1.35] - 2026-03-28

### Added

- Add tests for `resolve` command, use `cmd.OutOrStdout()` in `read`, and minor test cleanup ([#45])

## [0.1.34] - 2026-03-28

### Added

- Add `resolve` command to print the absolute path of a note by ref ([#44])

## [0.1.32] - 2026-03-28

### Added

- Add `--today` flag to `ls` for filtering notes created today ([#42])

## [0.1.31] - 2026-03-28

### Changed

- Unify note ref resolution across `read`, `append`, and `update` via `note.ResolveRef`: accepts numeric ID, absolute/relative path, basename, slug, or type name ([#41])

## [0.1.30] - 2026-03-28

### Changed

- Migrate `new`, `ls`, `new-todo`, and `update` flag bindings from package-level vars to `GetString`/`GetBool` for cleaner test isolation ([#39])

## [0.1.29] - 2026-03-28

### Removed

- Remove `filter` command (superseded by `ls --name`) ([#38])

## [0.1.28] - 2026-03-28

### Added

- Add `--name` flag to `ls` for case-insensitive substring search on note filenames ([#36])

## [0.1.27] - 2026-03-28

### Added

- Add `update` command for updating frontmatter and renaming notes ([#34])

## [0.1.26] - 2026-03-28

### Changed

- Replace `[>]` forwarded state with `(moved)` tag in todo rollover ([#33])

## [0.1.25] - 2026-03-24

### Added

- Add `--create` flag to `append` subcommand ([#31])

## [0.1.24] - 2026-03-24

### Added

- Add `append` command for appending stdin text to existing notes ([#30])

## [0.1.23] - 2026-03-24

### Added

- Add `rg` command for searching note contents using ripgrep ([#27])
- Add default options for `rg` and `grep` subcommands ([#28])

### Changed

- Replace positional `[type]` argument in `latest` with `--type`, `--slug`, and `--tag` flags

## [0.1.19] - 2026-03-23

### Fixed

- Limit grep to `.md` files and exclude `.git` directory ([#25])

## [0.1.18] - 2026-03-21

### Fixed

- Support variable-length year in date format ([#24])

## [0.1.17] - 2026-03-21

### Changed

- Rename "archive" to "store" in all user-facing text ([#23])

## [0.1.12] - 2026-03-21

### Added

- Add `--tag` flag to `ls` command ([#18])

## [0.1.11] - 2026-03-20

### Fixed

- Fix `path` and `latest` output going to stderr ([#17])

## [0.1.10] - 2026-03-20

### Changed

- Change default notes path to `~/notes` ([#16])

## [0.1.9] - 2026-03-20

### Fixed

- Make `notes path` return absolute path ([#15])

## [0.1.8] - 2026-03-20

### Added

- Add `grep` command for searching note contents ([#14])

## [0.1.7] - 2026-03-20

### Added

- Add `path` command to print notes store location ([#13])

## [0.1.6] - 2026-03-20

### Added

- Add `latest` command to print path to most recent note ([#12])

## [0.1.5] - 2026-03-20

### Added

- Add `bin/update` script for convenient local updates ([#11])

## [0.1.4] - 2026-03-20

### Added

- Add `--title` flag to `new` command ([#9])

## [0.1.2] - 2026-03-20

### Changed

- Generalize root command description ([#8])

## [0.1.0] - 2026-03-13

### Added

- Add `new` and `new-todo` commands ([#2])
- Add `--no-frontmatter` flag to `read` command ([#3], [#4])

[0.1.71]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.71
[0.1.70]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.70
[0.1.69]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.69
[0.1.66]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.66
[0.1.63]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.63
[0.1.60]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.60
[0.1.59]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.59
[0.1.58]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.58
[0.1.57]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.57
[0.1.55]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.55
[0.1.54]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.54
[0.1.41]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.41
[0.1.40]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.40
[0.1.39]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.39
[0.1.38]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.38
[0.1.37]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.37
[0.1.36]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.36
[0.1.35]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.35
[0.1.34]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.34
[0.1.32]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.32
[0.1.31]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.31
[0.1.30]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.30
[0.1.29]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.29
[0.1.28]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.28
[0.1.27]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.27
[0.1.26]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.26
[0.1.25]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.25
[0.1.24]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.24
[0.1.23]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.23
[0.1.19]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.19
[0.1.18]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.18
[0.1.17]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.17
[0.1.12]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.12
[0.1.11]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.11
[0.1.10]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.10
[0.1.9]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.9
[0.1.8]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.8
[0.1.7]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.7
[0.1.6]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.6
[0.1.5]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.5
[0.1.4]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.4
[0.1.2]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.2
[0.1.0]: https://github.com/dreikanter/notes-cli/releases/tag/v0.1.0
[#2]: https://github.com/dreikanter/notes-cli/pull/2
[#3]: https://github.com/dreikanter/notes-cli/pull/3
[#4]: https://github.com/dreikanter/notes-cli/pull/4
[#8]: https://github.com/dreikanter/notes-cli/pull/8
[#9]: https://github.com/dreikanter/notes-cli/pull/9
[#11]: https://github.com/dreikanter/notes-cli/pull/11
[#12]: https://github.com/dreikanter/notes-cli/pull/12
[#13]: https://github.com/dreikanter/notes-cli/pull/13
[#14]: https://github.com/dreikanter/notes-cli/pull/14
[#15]: https://github.com/dreikanter/notes-cli/pull/15
[#16]: https://github.com/dreikanter/notes-cli/pull/16
[#17]: https://github.com/dreikanter/notes-cli/pull/17
[#18]: https://github.com/dreikanter/notes-cli/pull/18
[#23]: https://github.com/dreikanter/notes-cli/pull/23
[#24]: https://github.com/dreikanter/notes-cli/pull/24
[#25]: https://github.com/dreikanter/notes-cli/pull/25
[#27]: https://github.com/dreikanter/notes-cli/pull/27
[#28]: https://github.com/dreikanter/notes-cli/pull/28
[#29]: https://github.com/dreikanter/notes-cli/pull/29
[#30]: https://github.com/dreikanter/notes-cli/pull/30
[#31]: https://github.com/dreikanter/notes-cli/pull/31
[#33]: https://github.com/dreikanter/notes-cli/pull/33
[#34]: https://github.com/dreikanter/notes-cli/pull/34
[#36]: https://github.com/dreikanter/notes-cli/pull/36
[#38]: https://github.com/dreikanter/notes-cli/pull/38
[#39]: https://github.com/dreikanter/notes-cli/pull/39
[#41]: https://github.com/dreikanter/notes-cli/pull/41
[#42]: https://github.com/dreikanter/notes-cli/pull/42
[#44]: https://github.com/dreikanter/notes-cli/pull/44
[#45]: https://github.com/dreikanter/notes-cli/pull/45
[#46]: https://github.com/dreikanter/notes-cli/pull/46
[#48]: https://github.com/dreikanter/notes-cli/pull/48
[#51]: https://github.com/dreikanter/notes-cli/pull/51
[#52]: https://github.com/dreikanter/notes-cli/pull/52
[#53]: https://github.com/dreikanter/notes-cli/pull/53
[#71]: https://github.com/dreikanter/notes-cli/pull/71
[#72]: https://github.com/dreikanter/notes-cli/pull/72
[#73]: https://github.com/dreikanter/notes-cli/pull/73
[#74]: https://github.com/dreikanter/notes-cli/pull/74
[#75]: https://github.com/dreikanter/notes-cli/pull/75
[#76]: https://github.com/dreikanter/notes-cli/pull/76
[#77]: https://github.com/dreikanter/notes-cli/pull/77
[#78]: https://github.com/dreikanter/notes-cli/pull/78
[#79]: https://github.com/dreikanter/notes-cli/pull/79
[#80]: https://github.com/dreikanter/notes-cli/pull/80
[#81]: https://github.com/dreikanter/notes-cli/pull/81
[#82]: https://github.com/dreikanter/notes-cli/pull/82
[#83]: https://github.com/dreikanter/notes-cli/pull/83
[#85]: https://github.com/dreikanter/notes-cli/issues/85
[#88]: https://github.com/dreikanter/notes-cli/issues/88
[#90]: https://github.com/dreikanter/notes-cli/issues/90
[#92]: https://github.com/dreikanter/notes-cli/issues/92
[#93]: https://github.com/dreikanter/notes-cli/issues/93
[#97]: https://github.com/dreikanter/notes-cli/pull/97
[#98]: https://github.com/dreikanter/notes-cli/pull/98
[#99]: https://github.com/dreikanter/notes-cli/pull/99
[#100]: https://github.com/dreikanter/notes-cli/pull/100
[#102]: https://github.com/dreikanter/notes-cli/pull/102
[#104]: https://github.com/dreikanter/notes-cli/issues/104
[#106]: https://github.com/dreikanter/notes-cli/pull/106
[#107]: https://github.com/dreikanter/notes-cli/pull/107
[#108]: https://github.com/dreikanter/notes-cli/pull/108
[#109]: https://github.com/dreikanter/notes-cli/pull/109
[#110]: https://github.com/dreikanter/notes-cli/issues/110
[#112]: https://github.com/dreikanter/notes-cli/issues/112
[#114]: https://github.com/dreikanter/notes-cli/pull/114
[#116]: https://github.com/dreikanter/notes-cli/pull/116
[#118]: https://github.com/dreikanter/notes-cli/pull/118
[#119]: https://github.com/dreikanter/notes-cli/issues/119
[#120]: https://github.com/dreikanter/notes-cli/issues/120
[#117]: https://github.com/dreikanter/notes-cli/issues/117
[#123]: https://github.com/dreikanter/notes-cli/pull/123
[#115]: https://github.com/dreikanter/notes-cli/issues/115
[#131]: https://github.com/dreikanter/notes-cli/pull/131
[#132]: https://github.com/dreikanter/notes-cli/pull/132
[#136]: https://github.com/dreikanter/notes-cli/pull/135
[#139]: https://github.com/dreikanter/notes-cli/issues/139
[#141]: https://github.com/dreikanter/notes-cli/issues/141
[#146]: https://github.com/dreikanter/notes-cli/pull/146
[#149]: https://github.com/dreikanter/notes-cli/pull/149
