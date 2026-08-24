# Task 16 — CI test workflow

## Context

Read `docs/plans/003-distribution.md`, decision 9. Ordinary push/PR gate: build, vet, test.
Separate from the release workflow (task 15) because it triggers differently (every push, not
just tags) and should never block on release-only tooling like GoReleaser. Fully independent of
every other task.

## Deliverable: `.github/workflows/test.yml`

```yaml
name: test

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true
      - name: build
        run: go build ./...
      - name: vet
        run: go vet ./...
      - name: gofmt check
        run: |
          fmt_out="$(gofmt -l .)"
          if [ -n "$fmt_out" ]; then
            echo "gofmt needs to be run on:"
            echo "$fmt_out"
            exit 1
          fi
      - name: test
        run: go test ./... -v
```

Use `ubuntu-latest` here (unlike task 15's release workflow) — this job only needs to build for
the host platform to run the test suite, no cross-compilation involved.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
# Validate the YAML parses and the steps are sane by running the same commands locally:
go build ./... && go vet ./... && gofmt -l . && go test ./... -v
```

`gofmt -l .` must print nothing (if it doesn't, that's task 13's job to fix, not this one's — flag
it in your summary rather than editing other files here). If a GitHub Actions YAML linter
(`actionlint`) is available in your environment, run it against the new file and fix anything it
flags; if not available, skip that check and say so.

## Out of scope

No `.goreleaser.yml`, no `release.yml` (task 15). No README changes (task 17). Do not fix gofmt
issues in other files — only report them if `gofmt -l .` is non-clean when you check.
