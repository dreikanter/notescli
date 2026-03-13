# notescli

A CLI tool for interacting with a date-based notes archive.

## Prerequisites

- Go 1.21+
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

## Setup

```sh
git clone https://github.com/dreikanter/notescli.git
cd notescli
go mod download
```

## Build

```sh
make build       # produces ./notes binary
```

## Install

```sh
make install     # installs to ~/go/bin/notes
```

Make sure `~/go/bin` is on your `PATH`:

```sh
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

The version is derived from git tags at build time via `git describe`. Without
tags, the binary reports the short commit hash as its version.

## Versioning

Versions follow `v0.{PR_number}.0` format (e.g. PR #5 → `v0.5.0`).

After merging a PR to `main`, tag and push:

```sh
git tag v0.X.0    # where X is the merged PR number
git push origin v0.X.0
```

Then reinstall to pick up the new version:

```sh
make install
notes --version
```

## Usage

```sh
# Show help
notes --help

# Read a note by ID
notes read 8823

# Read a note by slug
notes read todo

# Read a note by filename
notes read 20260106_8823.md

# Filter notes by fragment
notes filter todo
notes filter 2026

# List recent notes
notes ls
notes ls --limit 10
notes ls --type todo

# Override notes archive path
notes --path /path/to/notes read 8823
```

The notes archive path is resolved in this order:

1. `--path` flag
2. `NOTES_PATH` environment variable
3. `~/Dropbox/Notes` (default)

## Development

```sh
make test        # run all tests
make lint        # run golangci-lint
make clean       # remove built binary
```

Run a single test:

```sh
go test ./note/ -run TestParseFilename -v
```

## Project structure

```
cmd/notes/main.go    # binary entrypoint → produces "notes"
internal/cli/        # cobra command definitions
  root.go            # root command, --path flag, path resolution
  read.go            # notes read
  filter.go          # notes filter
  ls.go              # notes ls
note/                # domain logic (parsing, scanning, matching)
  note.go            # Note struct, ParseFilename
  archive.go         # Scan, Resolve, Filter, FilterBySlug
  note_test.go       # unit tests for parsing
  archive_test.go    # tests for scanning and matching
testdata/            # fixture notes for tests
```

## License

MIT
