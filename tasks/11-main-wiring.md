# Task 11 — Main wiring: filters, sort, JSON output, header

## Context

Read `PLAN.md`. Depends on ALL previous tasks (01–10 merged; `go test ./...` green before you
start). Ports `main()` and the `--json` branch from `~/Code/dotfiles/bin/sessions` — read the
whole Python file before writing code.

## Deliverable: replace the stub `main.go`

Structure it as a testable `run` plus a thin `main`:

```go
// run executes the program against explicit roots and writers so tests can
// inject fixtures. Returns an exit code.
func run(argv []string, piRoot, claudeRoot string, stdout, stderr io.Writer, now time.Time) int

func main() {
	os.Exit(run(os.Args[1:], piSessionsDir(), claudeSessionsDir(), os.Stdout, os.Stderr, time.Now()))
}
```

`run` behaviors, in order (mirroring Python `main`):

1. Parse args (task 10). On parse error: print the error to stderr, return 2. On help:
   print a usage text summarizing the flags (content is yours; include the "Defaults to
   today. Rows are session-days; totals sum the range." epilog) and return 0.
2. `resolveRange` → start/end. Build `days map[time.Time]bool` covering start..end inclusive
   (nil for --all).
3. Discover files: call task 06's `discoverIn(piRoot, claudeRoot, readPiSession, readClaudeSession, start)`
   using the roots injected into `run` (not `piSessionsDir()`/`claudeSessionsDir()` directly —
   those are only the defaults `main()` passes in). Parse each discovered file with its
   paired reader, skip nils, then apply session-level filters:
   - `--harness` mismatch → skip.
   - `--project`: case-insensitive substring match against `s.CWD` (`strings.Contains` on
     lowered strings).
   - temp dirs: unless `--temp`, skip sessions whose CWD has any `TempCwdPrefixes` prefix.
4. `sessionDays` per session; drop rows with zero `Start`; drop non-active rows when
   `--active`.
5. Sort rows by (Date, Start) ascending. Use `sort.SliceStable`.
6. `--json`: emit a JSON array (2-space indent) where each element has the keys, in this
   order (define a struct with json tags in this order):
   `date` ("2006-01-02"), `harness`, `id`, `name`, `project`, `cwd`,
   `start` ("2006-01-02T15:04:05"), `end` (same), `models` ([]string, must encode as `[]`
   not `null` when empty — initialize to non-nil), `tokens`, `usage` (the Usage struct),
   `cost` (float), `priced` (bool), `messages`, `active`, `path`,
   `duration_minutes` (int, floor of (end-start) in minutes). Follow with a trailing
   newline. Empty rows → `[]`. Then return 0 (no table, no header).
7. Table mode: print the header line then a blank line to stdout —
   `"all sessions"` when start is zero; `start.Format("Monday 2006-01-02")` when start==end;
   otherwise `"2026-08-01 .. 2026-08-07"` (`"2006-01-02"` both sides, joined by `" .. "`).
   Then `renderTable(stdout, stderr, rows, showDate)` where
   `showDate = (distinct dates in rows) > 1 || len(days) > 1` (days non-nil).
8. Return 0.

## Deliverable: `main_test.go` (integration tests)

Build a temp fixture tree with one pi transcript and one Claude transcript (reuse the fixture
shapes from tasks 04/05; give them timestamps on two known days and set file mtimes with
`os.Chtimes`). Also one pi transcript with `cwd` under `/tmp` (temp-dir filtering) and one
claude transcript in a different project dir (project filtering). Drive `run` with a fixed
`now` and assert on captured stdout:

- default (today's date arg = a fixture day passed positionally): only that day's rows;
  header is `"Wednesday 2026-08-05"` style; no DATE column for a single-day, single-date
  result set.
- `--all --json`: unmarshal the output, assert row count, key presence, date/start string
  formats, `duration_minutes`, and that the `/tmp` session is absent.
- `--temp --all --json`: `/tmp` session present.
- `--project <substr>` and `--harness pi` filters.
- unknown flag → exit code 2, message on stderr.
- a JSON-order check: assert the raw output's first row has `"date"` before `"harness"`
  before `"id"` (string index comparison), guarding key order.

## Also port the signal/pipe behavior

In `main`, ignore `SIGPIPE`-style broken-pipe write errors (Go mostly handles this; wrapping
stdout writes to swallow `syscall.EPIPE` is sufficient) and exit 130 on interrupt is NOT
required — Go's default ^C behavior is acceptable. Note this divergence in a comment.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test ./...
go build -o /tmp/sessions-go . && /tmp/sessions-go --all >/dev/null   # runs on real data without panicking
```

## Out of scope

Do not modify other tasks' files except to fix a genuine bug their tests missed — if you do,
add the missing test there and name the fix in your report.
