# Changelog

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
[#106]: https://github.com/dreikanter/notes-cli/pull/106
[#107]: https://github.com/dreikanter/notes-cli/pull/107
[#108]: https://github.com/dreikanter/notes-cli/pull/108
