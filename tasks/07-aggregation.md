# Task 07 — Session-day aggregation and fallback naming

## Context

Read `PLAN.md`. Depends on tasks 01 and 03 (uses `price`, `shortModel`). Ports `session_days`,
`fallback_name`, and the `NOISE_PREFIXES` behavior from `~/Code/dotfiles/bin/sessions` — read
those functions first; they are the spec.

## Deliverable: `aggregate.go`

```go
// sessionDays splits a session into one Row per calendar day of activity.
// days is the set of selected days (keys are dayOf values); nil means all
// days. now is used for the Active flag. Rows are returned sorted by date
// ascending (Python relies on dict ordering; we sort explicitly — the final
// global sort in main makes this equivalent).
func sessionDays(s *Session, days map[time.Time]bool, now time.Time) []Row

// fallbackName returns the first meaningful user prompt as a name: skip
// non-user events, trim whitespace, skip empty text and text starting with
// any NoisePrefixes entry, collapse all whitespace runs to single spaces
// (strings.Fields + strings.Join), truncate to 48 bytes. If nothing
// qualifies, return "(unnamed)". Ports fallback_name.
func fallbackName(s *Session) string
```

`sessionDays` behaviors (each is a test case):

1. Events with zero `TS` are skipped. Events whose `dayOf(TS)` is not in `days` (when days is
   non-nil) are skipped.
2. Per day bucket: `Start` = min TS, `End` = max TS, `Messages` = event count, `Usage`
   accumulated across events, `Tokens` = `Usage.Total()`.
3. `Models`: distinct raw models in order of first appearance (empty models skipped), then
   each passed through `shortModel` for the Row.
4. Cost, only for events where `Usage.Any()`:
   - event `Cost` non-nil → add it;
   - else `price(event.Model, event.Usage)` non-nil → add that;
   - else mark the bucket `Priced = false` (Cost keeps accumulating from other events).
   Events with all-zero usage never affect Cost or Priced. Buckets start `Priced = true`.
5. `Name`: `s.Name` if non-empty, else `fallbackName(s)`. Same name on every Row.
6. `Project`: `filepath.Base(s.CWD)`; if that is "" or "/" or ".", use `s.CWD` as-is
   (Python: `Path(cwd).name or cwd`).
7. `Active`: `now.Sub(End) < ActiveWindow`.
8. `Date` is the `dayOf` value; `Harness`/`ID`/`CWD`/`Path` copied from the session.

## Deliverable: `aggregate_test.go`

Build sessions in Go literals (no files needed):

- A session spanning two days with three events (one per day plus one more) → two Rows with
  correct per-day Start/End/Messages/Usage; verify a `days` filter selecting only day 1
  yields one Row.
- Cost mixing: event with recorded Cost + event with priceable model + event with unknown
  model and nonzero usage → Priced false, Cost = recorded + computed. Same without the
  unknown-model event → Priced true.
- Unknown model with ZERO usage → Priced stays true.
- Model dedup and ordering, including an event with empty model between two named ones.
- `fallbackName`: assistant-first events skipped; `"<system-reminder>..."`, `"# header"`,
  `"Caveat: ..."`, `"[Request interrupted..."`, and whitespace-only texts skipped; a text of
  `"  fix   the\nbug  "` → `"fix the bug"`; a 60-char prompt truncated to 48; all-noise
  session → `"(unnamed)"`.
- `Active`: End 1h59m before now → true; 2h01m → false.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run 'TestSessionDays|TestFallbackName' ./...
```

Break the Priced logic once (treat unknown model as $0 but leave Priced true), watch a test
fail, restore, confirm green.

## Out of scope

Only `aggregate.go` and `aggregate_test.go`.
