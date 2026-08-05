# Task 03 — Pricing table, model aliases, `shortModel`, `price`

## Context

Read `PLAN.md`. Depends on task 01. Ports the `PRICING` / `MODEL_ALIASES` tables and the
`short_model` / `price` functions from `~/Code/dotfiles/bin/sessions` (read it first).

## Deliverable: `pricing.go`

```go
// rates are USD per million tokens.
type rates struct {
	Input, Output, CacheWrite, CacheRead float64
}

// Anthropic cache writes bill at 1.25x input, reads at 0.1x input.
var pricing = map[string]rates{ ... } // copy all 9 rows from the Python PRICING table

// Claude Code writes bare aliases when a session is started with one.
var modelAliases = map[string]string{
	"opus":   "claude-opus-5",
	"sonnet": "claude-sonnet-4-5",
	"haiku":  "claude-haiku-4-5",
}

// shortModel: "" -> ""; resolve alias; take the part after the last "/";
// strip a leading "claude-" prefix. Ports short_model.
func shortModel(model string) string

// price returns the cost in USD for usage under model's rates, or nil if the
// model (after alias resolution) isn't in the pricing table. Formula:
// (input*pIn + output*pOut + cacheWrite*pCW + cacheRead*pCR) / 1_000_000.
// Ports price.
func price(model string, u Usage) *float64
```

Note the Python tuple order is `(input, output, cache_write_5m, cache_read)` — map it onto the
struct fields carefully; getting CacheWrite/CacheRead swapped is the likely bug here.

## Deliverable: `pricing_test.go`

- `shortModel`: `""`→`""`; `"opus"`→`"opus-5"`; `"claude-sonnet-4-5"`→`"sonnet-4-5"`;
  `"anthropic/claude-opus-4-1"`→`"opus-4-1"`; `"gpt-5"`→`"gpt-5"`.
- `price`: known model with `Usage{Input: 1_000_000}` → exactly the input rate;
  `Usage{Input: 1e6, Output: 1e6, CacheWrite: 1e6, CacheRead: 1e6}` for
  `claude-sonnet-4-5` → `3.0+15.0+3.75+0.3 = 22.05`; alias `"sonnet"` gives the same;
  unknown model → nil; zero usage on a known model → non-nil `0.0`.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run 'TestShortModel|TestPrice' ./...
```

Break `price` deliberately once (swap CacheWrite/CacheRead rates), confirm a test catches it,
restore, confirm green. If no test catches it, add one that does.

## Out of scope

Only `pricing.go` and `pricing_test.go`.
