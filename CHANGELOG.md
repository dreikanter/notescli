# Changelog

## [Unreleased]

### Added

- Add `rg` command for searching note contents using ripgrep ([#27])
- Add default options for `rg` and `grep` subcommands ([#28])

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
