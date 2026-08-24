# Task 14 — `--version` flag with ldflags-injectable version string

## Context

Read `docs/plans/003-distribution.md`, decision 5. GoReleaser (task 15) will inject a real
version at build time via `-ldflags "-X main.version=..."`; a plain local `go build` (no ldflags)
must still work and report `dev`. This task is independent of task 15 — it only needs to expose
the right symbol name (`main.version`) for task 15 to target later; it does not need task 15's
files to exist.

## Deliverable 1: `version.go` (new file)

```go
package main

// version is set at build time via:
//   go build -ldflags "-X main.version=1.2.3"
// Local builds without ldflags report "dev".
var version = "dev"
```

## Deliverable 2: wire `--version` / `-v` into `main.go`

In `run` (in `main.go`), handle `--version`/`-v` the same way the existing `show` subcommand is
special-cased — as an early check before `parseArgs`, since `--version` is a standalone action,
not a filter flag that composes with others:

```go
if len(argv) > 0 && (argv[0] == "--version" || argv[0] == "-v") {
	fmt.Fprintln(stdout, "sessions "+version)
	return 0
}
```

Place it alongside (not replacing) the existing `if len(argv) > 0 && argv[0] == "show"` check at
the top of `run`.

## Deliverable 3: mention it in `usageText`

Add a line to the `options:` block in `usageText` (in `main.go`), near `-h, --help`:

```
  -v, --version         print the version and exit
```

## Deliverable 4: test in `main_test.go`

Add a test that calls `run` with `argv = []string{"--version"}` and asserts stdout is exactly
`"sessions dev\n"` (the default `version` value, since the test binary isn't built with ldflags).
Also test `-v` produces the same output. Follow the existing test style in `main_test.go` (look at
how it already constructs `run`'s arguments — e.g. temp dirs for `piRoot`/`claudeRoot` — and reuse
that pattern rather than inventing a new harness).

## Acceptance criteria (all must exit 0)

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test ./...
go build -o /tmp/sessions-dev . && /tmp/sessions-dev --version   # prints "sessions dev"
go build -ldflags "-X main.version=1.2.3" -o /tmp/sessions-v . && /tmp/sessions-v --version  # prints "sessions 1.2.3"
```

Break it once on purpose (e.g. comment out the `--version` check) to confirm your new test fails,
then restore and confirm green.

## Out of scope

No changes to `cli.go`'s flag parser — `--version` is handled before `parseArgs` runs, so it
never needs to know about mutual exclusion with `--week`/`--yesterday`/etc. No `.goreleaser.yml`,
no CI files — those are task 15/16.
