# Task 08 — Humanizing helpers

## Context

Read `PLAN.md`. Depends on task 01 only. Ports `humanize_tokens`, `humanize_duration`,
`humanize_models` from `~/Code/dotfiles/bin/sessions` — read them first.

## Deliverable: `format.go`

```go
// humanizeTokens: >=1e9 -> "1.2B" (one decimal); >=1e6 -> "3.4M" (one
// decimal); >=1e3 -> "56k" (no decimal, rounded like Python's %.0f i.e.
// banker's-free: use strconv.FormatFloat(f, 'f', 0, 64)); else the plain
// integer. Ports humanize_tokens.
func humanizeTokens(n int64) string

// humanizeDuration: total whole minutes (floor). <60 -> "42m"; else
// "3h07m" (minutes zero-padded to 2). Ports humanize_duration.
func humanizeDuration(d time.Duration) string

// humanizeModels: empty -> "-". Join the first two with ","; if that string
// exceeds 22 bytes, cut to 21 bytes and append "~". If more than two models,
// append "+N" where N = len-2 (after any truncation of the head). Ports
// humanize_models.
func humanizeModels(models []string) string
```

Formatting parity notes (must match Python exactly):

- Python `f"{x:.1f}"` rounds half-to-even; Go `fmt.Sprintf("%.1f", x)` does the same. Use
  `fmt.Sprintf`.
- `humanize_tokens(1500)` in Python is `f"{1.5:.0f}k"` → `"2k"` (round-half-even: 1.5→2,
  2.5→2). Test both 1500→"2k" and 2500→"2k".

## Deliverable: `format_test.go`

Table-driven:

- tokens: 0→"0", 999→"999", 1000→"1k", 1500→"2k", 2500→"2k", 999_499→"999k",
  1_000_000→"1.0M", 12_345_678→"12.3M", 1_500_000_000→"1.5B".
- duration: 0→"0m", 59m59s→"59m", 60m→"1h00m", 3h7m→"3h07m", 25h5m→"25h05m".
- models: nil→"-", ["opus-5"]→"opus-5", ["a","b"]→"a,b", ["a","b","c"]→"a,b+1",
  ["a","b","c","d"]→"a,b+2", two long names whose join is 23+ bytes → 21 bytes + "~"
  (and with a third model, `...~+1`).

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run 'TestHumanize' ./...
```

## Out of scope

Only `format.go` and `format_test.go`.
