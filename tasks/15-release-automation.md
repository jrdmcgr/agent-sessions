# Task 15 — GoReleaser config + release workflow

## Context

Read `docs/plans/003-distribution.md`, decisions 2–4 and 9. This produces the "no Go toolchain"
distribution path: on every `v*` tag push, build binaries for four OS/arch combos and publish
them as a GitHub Release. Independent of every other task — it does not need task 14's
`version.go` to exist yet to be written correctly, only to target the symbol name `main.version`
(already fixed by the plan).

## Deliverable 1: `.goreleaser.yml` (repo root)

GoReleaser schema v2. Use exactly this shape (adjust only if `goreleaser check` — see acceptance
— reports a schema error, in which case fix the minimum needed and note what changed):

```yaml
version: 2

project_name: sessions

builds:
  - id: sessions
    binary: sessions
    main: .
    ldflags:
      - -s -w -X main.version={{.Version}}
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64

archives:
  - id: sessions
    formats: [tar.gz]
    name_template: "sessions_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - none*

checksum:
  name_template: "checksums.txt"

release:
  github:
    owner: jrdmcgr
    name: agent-sessions

changelog:
  disable: true
```

Notes:
- `files: [none*]` keeps the tarball to just the binary — no README/LICENSE bloat in the archive.
- `changelog.disable: true` because this plan doesn't set up conventional commits; a generated
  changelog would be noise. Revisit later if that changes.
- The archive name template must produce exactly `sessions_<version>_<os>_<arch>.tar.gz` (decision
  4 in the plan) — verify the rendered name in the acceptance check below, not just that the file
  builds.

## Deliverable 2: `.github/workflows/release.yml`

```yaml
name: release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: "~> 2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Use `macos-latest` as the runner (not `ubuntu-latest`) since this build needs to cross-compile for
both `darwin` and `linux` targets and Go's cross-compilation works identically either way — but if
you have a documented reason `ubuntu-latest` is preferable for this repo, use that instead and say
why in your summary.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
# Install goreleaser if not present: `brew install goreleaser` or `go install github.com/goreleaser/goreleaser/v2@latest`
goreleaser check                          # validates .goreleaser.yml, must exit 0
goreleaser release --snapshot --clean     # local dry-run build, no publish, must exit 0
ls dist/ | grep -E 'sessions_.*_(darwin|linux)_(amd64|arm64)\.tar\.gz'   # all 4 present
tar -tzf dist/sessions_*_darwin_arm64.tar.gz   # contains exactly one file: sessions
```

If `goreleaser` isn't installed and can't be installed in your environment, do your best to
hand-verify the YAML against the v2 schema (https://goreleaser.com/customization/) and say so
plainly in your summary — do not claim the acceptance criteria passed if you couldn't run them.

## Out of scope

No changes to `main.go`/`version.go` (task 14 owns those — this task only needs the symbol name
`main.version` to exist by release time, not now). No `test.yml` (task 16). No README (task 17).
