# Task 10 — CLI argument parsing and date-range resolution

## Context

Read `PLAN.md`. Depends on task 01 only. Ports the `argparse` setup, `parse_day`,
`week_bounds`, and `resolve_range` from `~/Code/dotfiles/bin/sessions` — read them first.
Go's `flag` package can't express `--week [OFFSET]` with an optional value, so parse by hand.

## Deliverable: `cli.go`

```go
// options holds the parsed command line.
type options struct {
	date      time.Time // positional YYYY-MM-DD; zero if absent
	week      *int      // nil if --week absent; 0 for bare --week
	yesterday bool
	all       bool
	since     time.Time // zero if absent
	until     time.Time // zero if absent
	project   string
	harness   string // "", "pi", or "claude"
	active    bool
	temp      bool
	jsonOut   bool
}

// parseArgs parses argv (excluding the program name). On error, returns a
// non-nil error whose message names the offending flag/value. Handles -h and
// --help by returning a sentinel errHelp after usage is available to main.
func parseArgs(argv []string) (*options, error)

// parseDay parses "YYYY-MM-DD" strictly (time.ParseInLocation with layout
// "2006-01-02" in time.Local, then dayOf). Error message:
// `expected YYYY-MM-DD, got "<value>"`.
func parseDay(value string) (time.Time, error)

// weekBounds returns the Monday and Sunday (inclusive) of the calendar week
// `offset` weeks from the current one. Monday-based: Go's Weekday() has
// Sunday=0, Python's weekday() has Monday=0 — convert carefully:
// daysSinceMonday := (int(today.Weekday()) + 6) % 7. Ports week_bounds.
func weekBounds(offset int, today time.Time) (time.Time, time.Time)

// resolveRange maps options to an inclusive (start, end) day pair, both zero
// for --all. Precedence, matching Python resolve_range exactly:
// all → (zero, zero); positional date → (date, date); since/until set →
// (since or 1970-01-01, until or today); week non-nil → weekBounds;
// yesterday → (yesterday, yesterday); default → (today, today).
func resolveRange(o *options, today time.Time) (time.Time, time.Time)
```

Parsing rules (each is a test case):

1. Flags accept both `--flag value` and `--flag=value` forms.
2. `--week` optional value: bare `--week` → 0. `--week -1` → -1 (a following token that
   parses as an integer is consumed; a non-integer token like `--json` or a date is NOT).
   `--week=-1` → -1.
3. Exactly one non-flag positional allowed; it must parse as a day. A second positional or a
   bad date is an error.
4. Mutual exclusion (argparse group): at most one of `--week`, `--yesterday`, `--all`; a
   second one is an error naming both.
5. `--harness` value must be `pi` or `claude`; anything else is an error.
6. Unknown flag → error naming it.

## Deliverable: `cli_test.go`

Table-driven tests for every rule above plus:

- `weekBounds(0, wed)` for a known Wednesday (e.g. 2026-08-05) → Mon 2026-08-03..Sun 2026-08-09;
  `weekBounds(-1, monday)` → previous Monday..Sunday; a Sunday input stays in ITS week
  (Monday 6 days earlier).
- `resolveRange` precedence: verify each branch with a fixed `today`, including that
  `--since` alone gets until=today and `--until` alone gets since=1970-01-01.
- `parseDay("2026-02-30")` errors; `parseDay("2026-08-05")` equals
  `time.Date(2026,8,5,0,0,0,0,time.Local)`.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run 'TestParseArgs|TestParseDay|TestWeekBounds|TestResolveRange' ./...
```

## Out of scope

Only `cli.go` and `cli_test.go`. Do not touch `main.go` — task 11 wires parsing in and owns
the usage text.
