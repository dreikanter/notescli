# Changelog

## [0.1.46] - 2026-04-04

### Added

- Add `edit` command to open a note in `$VISUAL` or `$EDITOR` ([#67])

[#67]: https://github.com/dreikanter/notescli/pull/67

## [0.1.45] - 2026-04-04

### Fixed

- `new-todo` no longer fails when no previous todo exists; creates an empty todo instead. `--force` works correctly when today's todo is the only one ([#58])

[#58]: https://github.com/dreikanter/notescli/pull/58

## [0.1.44] - 2026-04-04

### Fixed

- Reject conflicting `update` flags (`--slug`/`--no-slug`, `--type`/`--no-type`, `--tag`/`--no-tags`, `--public`/`--private`) instead of silently picking a winner ([#57])

[#57]: https://github.com/dreikanter/notescli/pull/57

## [0.1.43] - 2026-04-04

### Fixed

- Fix ref resolution for all-digit slugs; reject all-digit slugs in `new` and `update` commands ([#72])

[#72]: https://github.com/dreikanter/notescli/pull/72

## [0.1.42] - 2026-04-04

### Fixed

- Output absolute paths from `ls` to enable Unix pipelines like `notes ls | xargs notes read` ([#55])

[#55]: https://github.com/dreikanter/notescli/pull/55

## [0.1.41] - 2026-04-04

### Changed

- `update` command now returns an error when called with no flags instead of silently rewriting the file unchanged ([#69])

[#69]: https://github.com/dreikanter/notescli/pull/69
## [0.1.40] - 2026-04-04

### Added

- Add `--today` flag to `resolve` command for date-based note existence checks ([#53])

[#53]: https://github.com/dreikanter/notescli/pull/53

## [0.1.39] - 2026-04-04

### Changed

- Remove default limit from `ls`; output all notes unless `--limit` is specified. Handle SIGPIPE for clean pipe behavior ([#50])

[#50]: https://github.com/dreikanter/notescli/pull/50

## [0.1.38] - 2026-04-04

### Added

- Add `--today` flag to `append` for daily note rotation: appends to today's matching note or creates a new one ([#49])

[#49]: https://github.com/dreikanter/notescli/pull/49

## [0.1.37] - 2026-03-30

### Fixed

- Trim whitespace from `resolve` query to prevent lookup failures from trailing spaces or newlines ([#48])
- Restrict note scanning to known `YYYY/MM/` directory structure ([#48])

[#48]: https://github.com/dreikanter/notescli/pull/48

## [0.1.36] - 2026-03-29

### Added

- Add `Slug` and `Public` fields to `FrontmatterFields`; extend parser and builder; sync `slug:` frontmatter when `--slug`/`--no-slug` is used in `update` ([#46])

[#46]: https://github.com/dreikanter/notescli/pull/46

## [0.1.35] - 2026-03-28

### Added

- Add tests for `resolve` command, use `cmd.OutOrStdout()` in `read`, and minor test cleanup ([#45])

[#45]: https://github.com/dreikanter/notescli/pull/45

## [0.1.34] - 2026-03-28

### Added

- Add `resolve` command to print the absolute path of a note by ref ([#44])

[#44]: https://github.com/dreikanter/notescli/pull/44

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

[0.1.38]: https://github.com/dreikanter/notescli/releases/tag/v0.1.38
[0.1.37]: https://github.com/dreikanter/notescli/releases/tag/v0.1.37
[0.1.36]: https://github.com/dreikanter/notescli/releases/tag/v0.1.36
[0.1.35]: https://github.com/dreikanter/notescli/releases/tag/v0.1.35
[0.1.34]: https://github.com/dreikanter/notescli/releases/tag/v0.1.34
[0.1.32]: https://github.com/dreikanter/notescli/releases/tag/v0.1.32
[0.1.31]: https://github.com/dreikanter/notescli/releases/tag/v0.1.31
[0.1.30]: https://github.com/dreikanter/notescli/releases/tag/v0.1.30
[0.1.29]: https://github.com/dreikanter/notescli/releases/tag/v0.1.29
[0.1.28]: https://github.com/dreikanter/notescli/releases/tag/v0.1.28
[0.1.27]: https://github.com/dreikanter/notescli/releases/tag/v0.1.27
[0.1.26]: https://github.com/dreikanter/notescli/releases/tag/v0.1.26
[0.1.25]: https://github.com/dreikanter/notescli/releases/tag/v0.1.25
[0.1.24]: https://github.com/dreikanter/notescli/releases/tag/v0.1.24
[0.1.23]: https://github.com/dreikanter/notescli/releases/tag/v0.1.23
[0.1.19]: https://github.com/dreikanter/notescli/releases/tag/v0.1.19
[0.1.18]: https://github.com/dreikanter/notescli/releases/tag/v0.1.18
[0.1.17]: https://github.com/dreikanter/notescli/releases/tag/v0.1.17
[0.1.12]: https://github.com/dreikanter/notescli/releases/tag/v0.1.12
[0.1.11]: https://github.com/dreikanter/notescli/releases/tag/v0.1.11
[0.1.10]: https://github.com/dreikanter/notescli/releases/tag/v0.1.10
[0.1.9]: https://github.com/dreikanter/notescli/releases/tag/v0.1.9
[0.1.8]: https://github.com/dreikanter/notescli/releases/tag/v0.1.8
[0.1.7]: https://github.com/dreikanter/notescli/releases/tag/v0.1.7
[0.1.6]: https://github.com/dreikanter/notescli/releases/tag/v0.1.6
[0.1.5]: https://github.com/dreikanter/notescli/releases/tag/v0.1.5
[0.1.4]: https://github.com/dreikanter/notescli/releases/tag/v0.1.4
[0.1.2]: https://github.com/dreikanter/notescli/releases/tag/v0.1.2
[0.1.0]: https://github.com/dreikanter/notescli/releases/tag/v0.1.0
[#2]: https://github.com/dreikanter/notescli/pull/2
[#3]: https://github.com/dreikanter/notescli/pull/3
[#4]: https://github.com/dreikanter/notescli/pull/4
[#8]: https://github.com/dreikanter/notescli/pull/8
[#9]: https://github.com/dreikanter/notescli/pull/9
[#11]: https://github.com/dreikanter/notescli/pull/11
[#12]: https://github.com/dreikanter/notescli/pull/12
[#13]: https://github.com/dreikanter/notescli/pull/13
[#14]: https://github.com/dreikanter/notescli/pull/14
[#15]: https://github.com/dreikanter/notescli/pull/15
[#16]: https://github.com/dreikanter/notescli/pull/16
[#17]: https://github.com/dreikanter/notescli/pull/17
[#18]: https://github.com/dreikanter/notescli/pull/18
[#23]: https://github.com/dreikanter/notescli/pull/23
[#24]: https://github.com/dreikanter/notescli/pull/24
[#25]: https://github.com/dreikanter/notescli/pull/25
[#27]: https://github.com/dreikanter/notescli/pull/27
[#28]: https://github.com/dreikanter/notescli/pull/28
[#29]: https://github.com/dreikanter/notescli/pull/29
[#30]: https://github.com/dreikanter/notescli/pull/30
[#31]: https://github.com/dreikanter/notescli/pull/31
[#33]: https://github.com/dreikanter/notescli/pull/33
[#34]: https://github.com/dreikanter/notescli/pull/34
[#36]: https://github.com/dreikanter/notescli/pull/36
[#38]: https://github.com/dreikanter/notescli/pull/38
[#39]: https://github.com/dreikanter/notescli/pull/39
[#41]: https://github.com/dreikanter/notescli/pull/41
[#42]: https://github.com/dreikanter/notescli/pull/42
