# notescli

A CLI tool for interacting with a date-based notes archive.

## Build & Install

```sh
make build       # produces ./notes binary
make install     # installs to ~/go/bin/notes
```

Make sure `~/go/bin` is on your `PATH`:

```sh
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

## Versioning

The patch version is auto-incremented by GitHub Actions on each PR merge to
`main` (e.g. `v0.1.0` → `v0.1.1`). After merging, pull and reinstall locally:

```sh
git pull --tags
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

## License

MIT
