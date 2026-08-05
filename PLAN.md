# Plan: Rewrite `sessions` in Go

## Source

The reference implementation is `~/Code/dotfiles/bin/sessions` (Python, ~570 lines). It lists
agent coding sessions across two harnesses — pi (`~/.pi/agent/sessions/*/*.jsonl`) and Claude
Code (`~/.claude/projects/*/*.jsonl`) — by reading their JSONL transcripts, bucketing activity
into session-days, and rendering a table or JSON.

**The Python script is the spec.** Every task file cites the Python functions it ports. When in
doubt, match Python behavior exactly, including edge cases (malformed lines skipped, missing
timestamps skipped, unpriced models flagged with `?`).

## Target

- Go module `agent-sessions`, single `package main`, binary named `sessions`.
- No third-party dependencies — stdlib only (`encoding/json`, `flag` is NOT used; args are
  parsed by hand in task 10).
- Go 1.22+.
- One `.go` file per concern, one `_test.go` per file. Tests use only `testing` + `os.MkdirTemp`.

## File layout

| File          | Contents                                              | Task |
| ------------- | ----------------------------------------------------- | ---- |
| `go.mod`      | module decl                                           | 01   |
| `types.go`    | shared types, constants, `dayOf` helper               | 01   |
| `jsonl.go`    | JSONL iteration, timestamp parsing, map getters       | 02   |
| `pricing.go`  | pricing table, aliases, `shortModel`, `price`         | 03   |
| `pi.go`       | pi transcript reader                                  | 04   |
| `claude.go`   | Claude Code transcript reader                         | 05   |
| `discover.go` | transcript discovery + `decodeSlug`                   | 06   |
| `aggregate.go`| session-day bucketing + `fallbackName`                | 07   |
| `format.go`   | `humanizeTokens/Duration/Models`                      | 08   |
| `table.go`    | table rendering                                       | 09   |
| `cli.go`      | arg parsing + date-range resolution                   | 10   |
| `main.go`     | wiring, filters, sort, JSON output                    | 11   |
| `bin/parity`  | shell script diffing Go vs Python `--json` output     | 12   |

## Task order and dependencies

```
01 scaffold
├── 02 jsonl helpers
│   ├── 04 pi reader      (also needs 03)
│   └── 05 claude reader  (also needs 03)
├── 03 pricing
├── 06 discovery          (needs 02 signatures only)
├── 07 aggregation        (needs 01 types, 03 price)
├── 08 format helpers     (needs 01 only)
├── 09 table rendering    (needs 08)
└── 10 CLI parsing        (needs 01 only)
    └── 11 main wiring    (needs everything above)
        └── 12 parity check (run on Jared's machine, real data)
```

Tasks 02, 03, 06, 07, 08, 09, 10 are parallelizable after 01 lands. 04/05 need 02+03.
11 needs all. 12 is the final gate.

## Conventions every task must follow

- **Types come from `types.go` (task 01) verbatim.** Do not redefine or "improve" them.
- Timestamps: parse to `time.Time` in the **local** zone; a "date" is a `time.Time` at local
  midnight produced by `dayOf`. Never compare dates as strings.
- Missing/optional numbers from JSON arrive as `float64` via `map[string]any`; use the
  getter helpers from task 02, never raw type assertions.
- Nullable cost is `*float64` (nil = unpriced/unknown), matching Python's `None`.
- Errors reading files are silently skipped, exactly like the Python (`except OSError: return`).
- Each task's acceptance criteria are shell commands that must exit 0. Run them before
  declaring done. Do not modify files owned by other tasks.
- Every new test must be seen to fail first: temporarily break the code under test, observe
  the failure, restore, observe green.

## Acceptance for the whole project (task 12)

`sessions --all --json`, `--week`, `--yesterday`, a specific date, `--project`, `--harness`,
`--active`, and `--temp` produce output equivalent to the Python script on real data
(JSON compared with costs rounded to 4 decimals; table compared byte-for-byte).
