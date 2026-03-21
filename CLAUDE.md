# notescli

## Build & Install

```sh
make install     # builds and installs to ~/go/bin/notes
make build       # builds local ./notes binary
make test        # run tests
make lint        # run golangci-lint
```

## Versioning

Version is set at build time via git tags and `-ldflags`. The `Version` var in
`internal/cli/root.go` defaults to `"dev"` and is overridden by `make install`
/ `make build` using `git describe --tags`.

Patch version auto-increments on each PR merge via GitHub Actions
(`.github/workflows/tag.yml`), e.g. `v0.1.0` → `v0.1.1`.

After merging a PR, reinstall locally:

```sh
git checkout main && git pull --tags
make install
```

## Workflow

Run `make lint` before committing or creating a PR to catch issues early.

## Commits

- One logical change per commit (atomic commits)
- Commit message: one short line, no body
