# Changelog

## [0.3.7] - 2026-04-24

### Changed

- `internal/cli/append.go`: `notes append` now takes a single `<id>` integer argument. Load goes through `store.Get`, body is modified in-memory, and save goes through `store.Put`. Filter flags (`--type`, `--slug`, `--tag`, `--today`) are removed — users get IDs from `notes ls` or `notes resolve` ([#237]).

[#237]: https://github.com/dreikanter/notes-cli/pull/237

## [0.3.6] - 2026-04-24

### Changed

- **Breaking**: `notes ls` now prints one integer ID per line (newest first) instead of one absolute path per line. Scripts that piped `notes ls` into `notes read` / `notes rm` should switch to `notes resolve` if they need paths.
- `internal/cli/ls.go`: replace `note.Load` + filter pipeline with `store.IDs()` (fast directory-scan path) when no filter flags are set, and `store.All(...)` with composed `QueryOpt`s otherwise.
- Flag mapping: `--type` → `WithType` (now single-valued), `--slug` → `WithSlug`, `--tag` → `WithTag` (repeatable, AND), `--today` → `WithExactDate(time.Now())`.
- Removed the `--name` filename-fragment flag; it will return as a tag/title-fragment query option in a future phase ([#236]).

[#236]: https://github.com/dreikanter/notes-cli/pull/236

## [0.3.5] - 2026-04-24

### Changed

- `internal/cli/rm.go`: `notes rm` now takes a single `<id>` integer argument and deletes via `store.Delete(id)`. The `--today` flag is removed — users get today's ID from `notes ls --today` or `notes resolve`. Non-existent IDs surface `note.ErrNotFound` as a clear "not found" message ([#235]).

[#235]: https://github.com/dreikanter/notes-cli/pull/235

## [0.3.4] - 2026-04-24

### Changed

- `internal/cli/read.go`: `notes read` now takes a single `<id>` integer argument and resolves it via `store.Get(id)`. The filter flags (`--type`, `--slug`, `--tag`, `--today`) are removed — users discover IDs via `notes ls` or `notes resolve`. `--no-frontmatter` is preserved. Raw file bytes still come from disk (via `store.AbsPath`) so on-disk YAML formatting is unchanged ([#234]).

[#234]: https://github.com/dreikanter/notes-cli/pull/234

## [0.3.3] - 2026-04-24

### Changed

- `internal/cli/tags.go`: `notes tags` now calls `store.All()` instead of `note.Load` + index walk. `OSStore.All()` already returns entries with `Meta.Tags` populated as the merged frontmatter/body-hashtag union, so the command drops the two-source merge. Output format is unchanged ([#233]).

[#233]: https://github.com/dreikanter/notes-cli/pull/233

## [0.3.2] - 2026-04-24

### Added

- `note/os_store.go`: `OSStore` — the filesystem-backed `Store` implementation over the existing `YYYY/MM/YYYYMMDD_id_slug.md` layout. Reuses `ParseFilename`, `Filename`, `DirPath`, `WriteAtomic`, `NextID`, `ExtractHashtags`, `ParseNote`, and `FormatNote`. Filename scan sorts by `(date DESC, id DESC)` using the integer ID so `_11_` sorts newer than `_9_`. `Put` handles atomic rename on slug/date changes.
- `internal/cli/root.go`: `notesStore()` helper constructs an `*note.OSStore` from the resolved `--path` / `$NOTES_PATH` root. Threaded in now; individual commands adopt it in later phases ([#232]).

[#232]: https://github.com/dreikanter/notes-cli/pull/232

## [0.3.1] - 2026-04-23

### Added

- `note/mem_store.go`: `MemStore` — in-memory `Store` backed by `map[int]StoreEntry` with a `sync.RWMutex`. Test-only; validates the `Store` interface shape before `OSStore` is built. `IDs`, `All`, and `Find` sort newest-first by `Meta.CreatedAt` with a deterministic higher-ID tie-break. `Put` assigns IDs as `max(existing) + 1`, sets `Meta.CreatedAt` to now when zero, and always sets `Meta.UpdatedAt`. Includes a compile-time `var _ Store = (*MemStore)(nil)` assertion ([#231]).

[#231]: https://github.com/dreikanter/notes-cli/pull/231

## [0.3.0] - 2026-04-23

### Added

- `note/domain.go`: `StoreEntry` and `StoreMeta` domain types (temporary names; renamed to `Entry` / `Meta` in the cleanup phase).
- `note/storage.go`: `Store` interface, `QueryOpt` type, and filter constructors `WithType`, `WithSlug`, `WithTag`, `WithExactDate`, `WithBeforeDate`. The new `WithExactDate` coexists with the legacy `WithDate(string) ResolveOption`; it will be renamed to `WithDate` once the legacy Resolve path is removed.

No implementations and no behaviour changes — this PR only establishes the contract the subsequent migration phases build on ([#230]).

[#230]: https://github.com/dreikanter/notes-cli/pull/230

## [0.2.21] - 2026-04-23

### Changed

- `internal/cli/update.go` local vars renamed: `updateTags`→`tags`, `updateNoTags`→`noTags`, `updateTitle`→`title`, `updateDescription`→`description`, `updateSlug`→`slug`, `updateNoSlug`→`noSlug`, `updateType`→`noteType`, `updateNoType`→`noType`. The `update` prefix was redundant inside a file already scoped to the update command ([#213])

[#213]: https://github.com/dreikanter/notes-cli/pull/213

## [0.2.20] - 2026-04-23

### Changed

- `new.go`: `findUpsertNote` and `readStdinBody` extracted from `RunE`; the upsert lookup and stdin read are now named helpers
- `update.go`: `syncNoteFilename` extracted from `RunE`; the hard-link rename path is now a standalone function
- `annotate.go`: `invokeAnnotate` extracted; it wraps schema build, context deadline, `runClaude`, and result parse into one call ([#212])

[#212]: https://github.com/dreikanter/notes-cli/pull/212

## [0.2.19] - 2026-04-23

### Changed

- `runExternalSearch(cmd, args, tool, notInstalled, buildArgs)` extracted to `internal/cli/search.go`. Both `grep` and `rg` delegate to it; each command's `RunE` now only provides the tool-specific `buildArgs` closure. The `notInstalled` string triggers a `exec.LookPath` pre-check when non-empty ([#211])

[#211]: https://github.com/dreikanter/notes-cli/pull/211

## [0.2.18] - 2026-04-23

### Changed

- `resolveOrFilter(cmd, root, args, f, resolveOpts...)` added to `internal/cli/filter.go`. It handles the repeated "positional ref → `resolveRef`; filter flags → load+filter; neither → caller decides" pattern. `append` and `read` now delegate to it; `resolve` uses it for the filter-only path and keeps its positional-arg path inline since it allows `--today` alongside a positional argument ([#210])

[#210]: https://github.com/dreikanter/notes-cli/pull/210

## [0.2.17] - 2026-04-23

### Changed

- `lockStoreRoot` (the `syscall.Flock` helper used by `NextID`) moved from `note/id.go` into two build-tag files: `note/id_unix.go` (`//go:build unix`) and a no-op stub `note/id_other.go` (`//go:build !unix`). The package now compiles on non-Unix targets without a `syscall` dependency; behavior on Unix is unchanged ([#209])

[#209]: https://github.com/dreikanter/notes-cli/pull/209

## [0.2.16] - 2026-04-23

### Changed

- `Entry.MergedTags()` no longer recomputes on every call. The merged set (frontmatter tags ∪ body hashtags, lowercased, deduplicated, sorted) is now built once per entry during `Index.build()` and stored in an unexported `mergedTags` field; `MergedTags()` returns a fresh copy. `cloneEntry` clones the cached slice alongside the other slice fields ([#208])

[#208]: https://github.com/dreikanter/notes-cli/pull/208

## [0.2.15] - 2026-04-23

### Changed

- `Index.Snapshot()` added: returns a lightweight `Snapshot` value (slice-header copy, no deep copy) under a short read-lock. `Snapshot` exposes `Entries() []Entry` and `Len() int` and is safe to hold after the lock is released. Callers that need a stable view of the index after `Load` can use `Snapshot()` instead of `Entries()` ([#207])

[#207]: https://github.com/dreikanter/notes-cli/pull/207

## [0.2.14] - 2026-04-23

### Changed

- `note.Task` fields `Prefix`, `Marker`, and `Suffix` are now unexported — they were regex capture intermediates that leaked parse details to external consumers. Replaced with exported `State TaskState` (values: `TaskPending`, `TaskDone`, `TaskOther`) and `Text string` (trimmed task text after the bracket). `Reassembled` and `WithTag` continue to work; internal `RolloverTasks` uses the unexported captures ([#206])

[#206]: https://github.com/dreikanter/notes-cli/pull/206

## [0.2.13] - 2026-04-23

### Changed

- `note.FindLatestTodo` and `note.FindTodayTodo` removed from the `note` package. Both functions hardcode `Type == "todo"` and iterate `[]Entry` by date — CLI policy, not a library primitive. They are now unexported helpers in `internal/cli/new_todo.go`, their sole caller. `ParseTask`, `ExtractTasks`, `RolloverTasks`, and `FormatTodoContent` remain in `note` as reusable primitives ([#205])

[#205]: https://github.com/dreikanter/notes-cli/pull/205

## [0.2.12] - 2026-04-23

### Changed

- `parseEditor` and `isTerminalEditor` moved from `internal/cli/edit.go` to a new `internal/editor` package as exported `editor.Parse` and `editor.IsTerminal`. The new package is independently testable with no Cobra dependency ([#204])

[#204]: https://github.com/dreikanter/notes-cli/pull/204

## [0.2.11] - 2026-04-23

### Changed

- `writeAtomic` and `rootDirMode` moved from `internal/cli` to the `note` package as exported `note.WriteAtomic` and `note.StoreDirMode`. These are pure file-I/O primitives with no CLI dependency; exporting them makes them available to downstream consumers such as notes-pub / notes-view without duplication ([#203])

[#203]: https://github.com/dreikanter/notes-cli/pull/203

## [0.2.10] - 2026-04-23

### Changed

- `note.ResolveRef` removed. It called `Load(root, WithFrontmatter(false))` on every invocation, so external callers using it in a loop paid for a full store walk per call; the docs already steered callers toward `Index.Resolve`. External consumers should `Load` once and call `idx.Resolve(query, opts...)`, wrapping a `false`-bool miss in `note.ErrNotFound` when the caller's contract is `(_, error)`. The CLI now routes all seven call sites (`edit`, `append`, `annotate`, `read`, `resolve`, `update`, `rm`) through an internal `cli.resolveRef` helper that preserves the previous error surface ([#202])

[#202]: https://github.com/dreikanter/notes-cli/pull/202

## [0.2.9] - 2026-04-23

### Changed

- `note.ExtractTags(root)` removed. It ran a full `Load` on every call and hid body hashtags behind the unexported `bodyHashtags` field, so external consumers either paid for a re-walk or lost access. Callers that already hold an `Index` should combine `Index.Tags()` (frontmatter aggregate) with per-entry `Entry.BodyHashtags()` themselves; the `notes tags` CLI command routes through `Index` and is unchanged from the user's side. ([#201])
- `note.Entry.BodyHashtags() []string` exported as a defensive-copy accessor returning the lowercased, deduplicated hashtags extracted from the note body during `Load`. Returns nil when `Load` ran with `WithFrontmatter(false)` or the body had no hashtags. Mutating the returned slice does not affect the index ([#201])

[#201]: https://github.com/dreikanter/notes-cli/pull/201

## [0.2.8] - 2026-04-23

### Changed

- Rename `note.Filter` → `note.FilterByFilename` for symmetry with `FilterByTags`, `FilterByDate`, `FilterBySlug`, and `FilterByTypes`. The bare `Filter` name hid the fact that it only matches against the base filename; the `By…` suffix makes the axis explicit. Internal CLI call site (`internal/cli/ls.go`) updated. External callers importing `note.Filter` need a straight rename ([#200])

[#200]: https://github.com/dreikanter/notes-cli/pull/200

## [0.2.7] - 2026-04-23

### Changed

- `note.IsID` removed; it was a one-line alias for `note.IsDigits` with no stricter semantics to enforce. Internal callers (`ParseFilename`, `Index.Resolve`) now call `IsDigits` directly. External consumers that imported `note.IsID` for wikilink / CLI argument detection should switch to `note.IsDigits`, which keeps identical behavior (non-empty, ASCII digits only) ([#199])

[#199]: https://github.com/dreikanter/notes-cli/pull/199

## [0.2.6] - 2026-04-23

### Changed

- Rename `isFilenameCacheSafeType` → `filenameRoundtripSafeType` in `note/note.go`. The predicate has nothing to do with a cache; it reports whether a type round-trips cleanly through `Filename` / `ParseFilename`. Unexported helper, no external callers affected ([#198])

[#198]: https://github.com/dreikanter/notes-cli/pull/198

## [0.2.5] - 2026-04-23

### Changed

- `note.Index.Reload()` now returns `<-chan error` (was `<-chan struct{}`). A successful rebuild closes the channel with the zero value; a failing rebuild sends the error on the buffered channel before close, so `err := <-ch` returns the build error or nil. The logger installed via `WithLogger` still sees the same error. Long-lived services can now react to a specific reload's outcome instead of only being able to wait for "some build has finished" ([#197])

[#197]: https://github.com/dreikanter/notes-cli/pull/197

## [0.2.4] - 2026-04-23

### Changed

- `note.ErrNotFound = errors.New("note not found")` exported so callers can match misses with `errors.Is` instead of string-comparing. `ResolveRef` now wraps it on the priority-chain miss path (previously `fmt.Errorf("note not found: %s", …)` with no sentinel) and on the `resolveRelPath` EvalSymlinks miss. `Index.Resolve` keeps its `(Entry, bool, error)` shape — `bool=false` is a miss, `error` is reserved for I/O — and the `ErrNotFound` doc-comment spells out the convention so the two APIs stay distinguishable ([#196])

[#196]: https://github.com/dreikanter/notes-cli/pull/196

## [0.2.3] - 2026-04-23

### Changed

- `note.cloneEntry` now deep-copies `Frontmatter.Extra` — the map, each `yaml.Node` value, and the nested `Content` slices — so a web-service consumer that mutates `Extra` after a lookup cannot race other readers of the same `Index` entry. Previously only `Tags`, `Aliases`, and `bodyHashtags` were cloned, and the doc-comment warned that `Extra` was aliased; that footgun is gone ([#195])

[#195]: https://github.com/dreikanter/notes-cli/pull/195

## [0.2.2] - 2026-04-23

### Changed

- `note.TypesWithSpecialBehavior` unexported to `typesWithSpecialBehavior`; external importers can no longer `append` to the package-level slice and silently change CLI behavior globally. `note.HasSpecialBehavior(s)` remains the public predicate, and a new `note.SpecialBehaviorTypes()` returns a fresh copy of the list for callers that need the values. `SCHEMA.md` now references `HasSpecialBehavior` instead of the unexported slice ([#194])

[#194]: https://github.com/dreikanter/notes-cli/pull/194

## [0.2.1] - 2026-04-23

### Changed

- `note` package no longer writes to `os.Stderr`. Per-note frontmatter parse failures (`note/index.go`), `Index.Reload` build failures, and unreadable-subdirectory warnings during `Scan` now route through a new `note.Logger = func(error)`. Install one via `note.WithLogger` (LoadOption) or `note.WithScanLogger` (ScanOption); the default is a no-op so external importers (notes-pub, notes-view) can embed the package without inheriting its stderr output. The `notes` CLI wires a single `stderrLogger(cmd)` helper through every `note.Load` call, so user-visible output is unchanged ([#193])
- `.github/workflows/tag.yml` preserves the major.minor segment of the latest tag instead of hardcoding `v0.1.*`; `CLAUDE.md`'s Versioning and Changelog sections are updated to match. Bumping minor now requires a manual `v0.X.0` tag, after which the workflow continues patch-bumping within that series ([#193])

[#193]: https://github.com/dreikanter/notes-cli/pull/193

## [0.2.0] - 2026-04-23

### Changed

- Rename `note.Note` → `note.Ref` to drop the package/type stutter. `Entry` now embeds `Ref` instead of `Note`, and `ResolveRef` / `Scan` / `ParseFilename` now return `Ref`. The `Ref` field name replaces `Note` in `Entry` struct literals. No cross-package changes required — external callers only consume `note.Entry` and never reference `note.Note` by name. ([#164])

[#164]: https://github.com/dreikanter/notes-cli/pull/164

## [0.1.111] - 2026-04-23

### Changed

- `note.Scan` swaps its `opts ...ScanOptions` variadic (documented as "only the first is honored") for the functional-options pattern already used by `Load` and `ResolveRef`. New `ScanOption func(*ScanOptions)` and `WithStrict(b bool) ScanOption`. `Scan(root)` still defaults to strict; `Scan(root, WithStrict(false))` walks the full tree. The `ScanOptions` struct stays because it's still the argument to `Load`'s `WithScanOptions` and the watcher's `WithScanOptions`. Internal call sites and tests updated to the new form ([#168])
## [0.1.110] - 2026-04-23

### Changed

- `note/tags.go`: folded `*Index.mergedTagsSorted` back into `ExtractTags`. The helper was a method on `*Index` but declared in a different file from the rest of `Index`'s methods (`note/index.go`), and it had only one caller. Inlining drops the cross-file method and keeps `Index`'s surface in one place. No behavior change: nil on empty index, deduped/lowercased/sorted union of frontmatter tags and body hashtags, same locking discipline ([#167])

## [0.1.109] - 2026-04-23

### Changed

- `note.IsDigits` exported as a non-empty ASCII-digit predicate, carved out of the existing internal `isDigits`. `IsID` now delegates to it (same semantics, no behavior change). `note/watch/watch.go`'s `shouldWatchDir` and `strictNotePath` now call `note.IsDigits` instead of `note.IsID` — the check there is about a `YYYY` or `MM` directory segment being digits, not about the segment being a note ID. Internal `isDigits` callers (`ParseFilename` date check, `Scan`'s year/month directory filters, `ValidateSlug`'s all-digits rejection) follow the rename ([#166])
## [0.1.108] - 2026-04-23

### Changed

- `notes new` and `notes append` now read stdin via `cmd.InOrStdin()` instead of reading `os.Stdin` directly, so tests (or any caller) can inject input by setting `rootCmd.SetIn(...)`. The terminal-detection heuristic is now `stdinIsTerminal(io.Reader)` and only runs the `Stat()` check when the reader is an `*os.File`; any other reader (pipe, `strings.Reader`, `bytes.Buffer`, etc.) is treated as non-terminal. `new_test.go` and `append_test.go` drop the `os.Stdin = r` / `os.Pipe` dance and use `rootCmd.SetIn(strings.NewReader(...))` ([#165])

## [0.1.107] - 2026-04-23

### Changed

- `note.FilterByTags` no longer re-scans the store. Its signature is now `FilterByTags(entries []Entry, tags []string) []Entry` — the `root` argument and internal `Load` are gone; merged tags are read directly from `Entry.MergedTags()`. For `ls --tag foo` the prior two `WalkDir` passes (plus a second frontmatter read the first pass did not need) collapse to one walk with frontmatter ([#163])
- `note.Filter`, `FilterByDate`, `FilterBySlug`, `FilterByTypes`, `FindTodayTodo`, and `FindLatestTodo` now take `[]Entry` so the CLI pipeline is uniformly `Load → []Entry → …`. `Entry` embeds `Note`, so field access inside these helpers is unchanged ([#163])
- `internal/cli`: `applyFilters` takes and returns `[]note.Entry` (and no longer needs `root`). A new `loadOptsFor(f)` picks `note.WithFrontmatter(true)` only when a `--tag` filter is active, so commands that do not touch tags do not pay the frontmatter-read cost. Every CLI entry point (`ls`, `resolve`, `read`, `append`, `new --upsert`, `new-todo`) now does a single `note.Load` at the top and feeds `idx.Entries()` through the pipeline ([#163])

## [0.1.106] - 2026-04-23

### Changed

- `note.ResolveRef` and `note.ResolveRefDate` collapsed into a single `ResolveRef(root, query, opts...)` with a `WithDate` functional option, matching the `Load` options pattern. The `Date` suffix described a parameter rather than the operation, and `ResolveRef` was a zero-value wrapper over `ResolveRefDate(root, query, "")`. Date-aware call sites (`resolve`, `rm`) now pass `note.WithDate(date)`; plain callers keep their existing two-arg form. Adding future constraints (e.g. `WithType`) becomes a one-liner ([#161])
- `Index.Resolve` now accepts the same variadic `ResolveOption` set, so `WithDate` threads through the cached index and the by-ID / by-path map lookups stay O(1) even when date-filtered (the match is discarded after the fact if its `Date` does not match). The duplicated priority chain in `resolveInEntries` (which linear-scanned a pre-filtered entry slice because it had lost the index maps) is gone; `ResolveRef` is now a thin `Load` + `Index.Resolve` wrapper ([#161])

## [0.1.105] - 2026-04-23

### Changed

- `CLAUDE.md` adds an explicit Attribution rule: no AI/tool authorship lines ("Generated by Claude Code", "Co-authored-by: Claude", robot emoji, etc.) in PR titles/descriptions, commit messages, code comments, or issue comments ([#162])

## [0.1.103] - 2026-04-23

### Changed

- `note.NoteFilename` and `note.NoteDirPath` renamed to `note.Filename` and `note.DirPath` to drop the package-name stutter (`note.Note*`) that repeated at every call site. Tests and the one surviving doc-comment reference were updated in the same pass ([#159])
- `notes update` now reads the parsed bool value for both `--public` and `--private` instead of hardcoding `true`/`false` on `Changed()`. Previously `--public=false` would flip `Public` to `true` (the inverse of intent) and `--private=false` was a no-op. `MarkFlagsMutuallyExclusive("public","private")` still prevents both being set at once ([#159])
- `buildAnnotateSchema` in `internal/cli/annotate.go` now returns `(string, error)` and propagates the `json.Marshal` failure instead of silently discarding it with `_`. The input is controlled so today's callers can't trigger the error, but the pattern violated the "no silent error swallowing" rule ([#159])
- `CLAUDE.md` documents the CHANGELOG workflow explicitly: open the PR first, note the assigned number, then add the CHANGELOG entry referencing that number in a follow-up atomic commit. Avoids the chicken-and-egg of trying to predict the PR number before creation ([#159])

## [0.1.101] - 2026-04-23

### Changed

- `note.FormatNote` now returns `([]byte, error)` instead of panicking when `yaml.Marshal` fails. `Frontmatter.Extra` holds arbitrary `yaml.Node` values sourced from user input, which can in principle fail to re-encode (cycles, aliases), so the prior "impossible" panic was unsafe. All four production callers (`create`, `new_todo`, `annotate`, `update`) and the `frontmatter_test.go` suite (via a new `mustFormatNote` helper) handle the error ([#158])
- `note.DateFormat` exported as the canonical `"20060102"` layout constant. The literal was duplicated across 11 call sites in `note/` and `internal/cli/` (production and tests); every site now references the constant, giving a single source of truth for UID-derived and CLI-facing dates ([#158])

## [0.1.100] - 2026-04-23

### Changed

- `notes new` and `notes new-todo` now write the new note file via the existing `writeAtomic` helper (tmp + rename), matching every other note-write path in the CLI (`append`, `annotate`, `update`, and the rollover-update step of `new-todo`). A mid-write crash can no longer leave a truncated note at the target path; failure modes collapse to "nothing written" or "fully written" ([#134], [#156])
- `note/watch`: dropped the internal `strictDirPrefix` helper. Its strict-mode semantics were identical to `shouldWatchDir`, so `addTree`'s descent-pruning branch now simply returns `fs.SkipDir` whenever `shouldWatchDir` rejects a directory in strict mode. No behavior change; the fixed-depth YYYY/MM strict layout has nowhere deeper worth descending to ([#134], [#156])

## [0.1.99] - 2026-04-22

### Changed

- `note.ResolveEntryDate` now takes the `Entry` directly (`func ResolveEntryDate(e Entry, fi fs.FileInfo) (time.Time, string)`) instead of the explicit `Note` + `Frontmatter` pair it accepted when it landed in #149 before `Entry` existed. Priority, source labels, and `fi == nil` handling are unchanged. Callers holding an `Entry` from `note.Index` no longer need to unpack it ([#140])

## [0.1.98] - 2026-04-22

### Changed

- `note.FilterByTags`, `note.ExtractTags`, `note.ResolveRef`, and `note.ResolveRefDate` now route through `note.Load` + `Index` instead of re-walking and re-reading the store on every call. Behavior is unchanged: tag sources still merge frontmatter `tags:` with body hashtags, the resolve priority chain (ID → type → path → slug substring) is identical, and a per-note frontmatter parse error still logs to stderr and falls back to body hashtags only. Callers that already hold an `Index` can skip the wrappers and call `Index.Resolve` / `Entry.MergedTags` directly to avoid a second file-read pass. `Entry.MergedTags()` returns the sorted, lowercased, deduplicated union of frontmatter tags and body hashtags for a single entry ([#144])

## [0.1.97] - 2026-04-22

### Changed

- Documentation only: corrected the `CHANGELOG.md` reference for the `Frontmatter.Date` field (promoted in v0.1.90) from PR #146 to issue [#138]. No code change ([#153])

## [0.1.96] - 2026-04-22

### Added

- `note.Index.Reload() <-chan struct{}` requests a rebuild and returns a channel that closes once a walk completing at or after the call has swapped in. Scheduling: idle → start immediately; in-flight → queue at most one follow-up, and every caller arriving during the in-flight build receives the same queued `done` so they only observe completion after a walk that started after their request. Cleanup runs in a deferred block so a panicking build cannot leave waiters blocked. Pairs with the `note/watch` debouncer (step 7 of #134): watcher fires, consumer calls `Reload`, bursts collapse to at most one rebuild ([#143])

## [0.1.95] - 2026-04-22

### Added

- `note/watch` subpackage: an fsnotify-based `Watcher` that observes `.md` note activity under a store root and emits a single debounced signal on `Events()` after filesystem activity settles. Pairs with `Index.Reload` (step 7) — watcher fires, consumer reloads, index coalescer collapses bursts into at most one rebuild. `watch.WithScanOptions` mirrors `note.ScanOptions`: strict mode (default) ignores events outside `YYYY/MM/*.md`, lenient mode accepts any `.md` anywhere beneath root. Newly created directories are registered automatically. Placed in a subpackage so `fsnotify` stays out of the CLI binary's dependency graph ([#145])

## [0.1.94] - 2026-04-22

### Added

- `note.Entry`, `note.Index`, and `note.Load` consolidate the per-query `Scan` → `FilterByTags` → `ExtractTags` re-read chain into a single concurrent pass. `Load(root, opts...)` walks the store once, parses frontmatter in parallel (workers default to `runtime.NumCPU()`), and returns an `*Index` with `Root`, `Entries`, `ByID`, `ByRel`, `BySlug`, `ByTag`, `Tags`, and `Resolve` methods. `Entry` embeds `Note` and adds `Frontmatter`, `ModTime`, and `Size`. Options: `WithFrontmatter(bool)` (default true), `WithWorkers(int)`, and `WithScanOptions(ScanOptions)`. `Index` methods take an internal `RWMutex` and defensive-copy `Frontmatter.Tags` / `Frontmatter.Aliases` on return so future `Reload` can swap state atomically. Existing `Scan`/`ExtractTags`/`ResolveRef` APIs are unchanged ([#150])

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

- `note.Frontmatter` now has a reserved `Date time.Time` field (`yaml:"date,omitempty"`). Notes whose `date:` previously landed in `Frontmatter.Extra` now populate the typed field, and consumers no longer need to decode the `yaml.Node` themselves. Round-trip preserves the input format: date-only values (midnight UTC) serialize as `YYYY-MM-DD`; values with a non-zero time-of-day serialize as RFC3339. Consumers that need a date when `date:` is absent should fall back to the UID-derived date from the filename prefix, then file mtime — see `SCHEMA.md` ([#138])

## [0.1.89] - 2026-04-22

### Added

- `note.ExtractHashtags` is now exported (previously unexported `extractHashtags`). Downstream tools (notes-pub, notes-view) can reuse the same body-hashtag extraction rules — fenced code blocks, inline backticks, URL anchors, chained hashes — instead of re-implementing them ([#136])
- `note.IsID` reports whether a string is a valid notes-cli note ID (non-empty, ASCII digits only). Replaces the ad-hoc `isNoteID` / `IsUID` helpers currently duplicated in consumer projects ([#136])
- `note.NormalizeSlug` returns an ASCII-lowercase, URL-safe form of a string (non-alphanumeric runs collapse to `-`; leading/trailing dashes stripped). Shared normalization contract for filenames and URL path segments ([#136])
- `note.DeriveSlug` returns the normalized slug for a note using the fallback chain: frontmatter slug → stem with id prefix stripped → empty. Consolidates the slug-resolution logic that consumers were each inventing ([#136])

## [0.1.88] - 2026-04-22

### Removed

- Internal cleanup (no user-visible behavior change): drop the unused `Note.BaseName` field (assigned by `ParseFilename` but read only by two test assertions; `RelPath`/`Date`/`ID`/`Slug` already cover note identity) and a dead `_ = out` line in `TestRgExcludesNonMarkdown` ([#135])

## [0.1.87] - 2026-04-22

### Changed

- Internal refactor (no user-visible behavior change): per-command flag setup split into `registerXxxFlags()` helpers for `update`, `new`, `annotate`, `ls`, and `rm` so test setups can reuse them instead of duplicating flag wiring; `note.FilterByType` removed in favor of the existing multi-value `FilterByTypes`; `readID`, `writeID`, and `lockStoreRoot` unexported; `update` command's `contentChanged` initialization moved ahead of the conditional blocks ([#133])

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
[#133]: https://github.com/dreikanter/notes-cli/pull/133
[#135]: https://github.com/dreikanter/notes-cli/pull/135
[#136]: https://github.com/dreikanter/notes-cli/pull/136
[#153]: https://github.com/dreikanter/notes-cli/pull/153
[#139]: https://github.com/dreikanter/notes-cli/issues/139
[#141]: https://github.com/dreikanter/notes-cli/issues/141
[#138]: https://github.com/dreikanter/notes-cli/issues/138
[#149]: https://github.com/dreikanter/notes-cli/pull/149
[#150]: https://github.com/dreikanter/notes-cli/pull/150
[#145]: https://github.com/dreikanter/notes-cli/issues/145
[#143]: https://github.com/dreikanter/notes-cli/issues/143
[#144]: https://github.com/dreikanter/notes-cli/issues/144
[#140]: https://github.com/dreikanter/notes-cli/issues/140
[#134]: https://github.com/dreikanter/notes-cli/issues/134
[#156]: https://github.com/dreikanter/notes-cli/pull/156
[#158]: https://github.com/dreikanter/notes-cli/pull/158
[#159]: https://github.com/dreikanter/notes-cli/pull/159
[#162]: https://github.com/dreikanter/notes-cli/pull/162
[#161]: https://github.com/dreikanter/notes-cli/pull/161
[#163]: https://github.com/dreikanter/notes-cli/pull/163
[#165]: https://github.com/dreikanter/notes-cli/pull/165
[#166]: https://github.com/dreikanter/notes-cli/pull/166
[#167]: https://github.com/dreikanter/notes-cli/pull/167
[#168]: https://github.com/dreikanter/notes-cli/pull/168
