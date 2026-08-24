# Task 13 — Repo hygiene: LICENSE, .gitignore, gofmt

## Context

Read `docs/plans/003-distribution.md`. This task has no code-logic changes, only the
paperwork/formatting a public repo needs. Independent of every other task in the plan.

## Deliverable 1: `LICENSE`

Add the standard MIT license text at the repo root, copyright line:

```
Copyright (c) 2026 Jared McGuire
```

Use the canonical MIT license wording (the one `gh repo create --license mit` or
choosealicense.com/licenses/mit would produce) — do not paraphrase it.

## Deliverable 2: `.gitignore`

Current contents are exactly:

```
/agent-sessions
```

Add `.DS_Store` (macOS Finder metadata; currently untracked but not ignored, so it can be
accidentally `git add`ed). Resulting file should ignore both, one entry per line. Do not touch
the existing `/agent-sessions` line.

## Deliverable 3: gofmt

Three files are not gofmt-clean today: `discover_test.go`, `jsonl.go`, `show_test.go`. Run:

```sh
gofmt -w discover_test.go jsonl.go show_test.go
```

Do not hand-edit formatting — let gofmt do it — and do not touch any other file's formatting.

## Acceptance criteria (all must exit 0)

```sh
cd ~/Code/agent-sessions
gofmt -l .        # must print nothing
go build ./...
go vet ./...
go test ./...
```

## Out of scope

No changes to any `.go` file's logic, no README changes, no CI/release files. If gofmt wants to
reformat a file not named above, leave it — flag it in your summary instead of fixing it silently.
