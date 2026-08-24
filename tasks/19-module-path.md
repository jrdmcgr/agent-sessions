# Task 19 — Fix `go.mod` module path for `go install`

## Context

Read `docs/plans/003-distribution.md`, "Manual steps" and the module-path note. `go install
github.com/jrdmcgr/agent-sessions@latest` needs the module's declared path to match the GitHub
import path. Today `go.mod` declares `module agent-sessions`, which only works for local builds.

Confirmed before this task was written: nothing in the codebase imports the module by path (it's
a single `package main`, no internal packages) — the only occurrence of the literal string
`"agent-sessions"` anywhere in `*.go` is a test fixture value in `table_test.go` (`Project:
"agent-sessions"`, an arbitrary project-name string, unrelated to the module declaration). Do not
touch that test fixture — it's not the module path, just a coincidentally identical string used
as sample data.

## Deliverable

In `go.mod`, change:

```
module agent-sessions
```

to:

```
module github.com/jrdmcgr/agent-sessions
```

Leave the `go 1.22` line and all `require`/`// indirect` blocks untouched.

## Verify the assumption above still holds

Before editing, re-run this check yourself (the codebase may have changed since this task was
written) and confirm it still returns only the `table_test.go` fixture line and nothing that
looks like an import path:

```sh
cd ~/Code/agent-sessions
grep -rn '"agent-sessions' . --include=*.go
```

If it returns anything else (an actual `import "agent-sessions/..."` of an internal package),
stop and report it in your summary instead of proceeding — that would mean this is no longer a
one-line change.

## Acceptance criteria (all must exit 0)

```sh
cd ~/Code/agent-sessions
go build ./...
go vet ./...
go test ./...
```

`go.mod`'s first line must read exactly `module github.com/jrdmcgr/agent-sessions`.

## Out of scope

Do not touch `go.sum`, dependency versions, or any `.go` file. Do not attempt to test `go install`
against the real GitHub path — the repo is still private at this point in the plan (that's a
manual step Jared does after all tasks land), so it would fail for an unrelated reason (auth/404,
not this change).
