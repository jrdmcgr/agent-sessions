# Task 09 — Table rendering

## Context

Read `PLAN.md`. Depends on tasks 01 and 08. Ports `render_table` from
`~/Code/dotfiles/bin/sessions` — read it first; output must be byte-identical to the Python
for the same rows (task 12 diffs them).

## Deliverable: `table.go`

```go
// renderTable writes the session table to w (stdout in production).
// showDate prepends a DATE column. The unpriced-model footnote goes to errW
// (stderr in production). Ports render_table.
func renderTable(w, errW io.Writer, rows []Row, showDate bool)
```

Behaviors (mirror the Python exactly):

1. No rows → print `No sessions found.\n` to w and return.
2. Headers: optional `DATE`, then `PROJECT SESSION TIME DUR HARNESS MODEL TOKENS COST`.
3. Body cells per row:
   - DATE: `Mon 08-05` style — Go layout `"Mon 01-02"`.
   - PROJECT: `r.Project`.
   - SESSION: `r.Name` plus `" *"` when `r.Active`.
   - TIME: `"15:04-15:04"` from Start/End.
   - DUR: `humanizeDuration(End.Sub(Start))`.
   - HARNESS, MODEL (`humanizeModels(r.Models)`), TOKENS (`humanizeTokens(r.Tokens)`).
   - COST: `fmt.Sprintf("$%.2f", r.Cost)`, with `"?"` appended when `!r.Priced`.
4. Total line: first cell `TOTAL`; SESSION column gets `"N session-days"` when showDate else
   `"N sessions"`; DUR sums all (End-Start); TOKENS sums; COST sums, `"?"` appended unless
   every row is Priced. Other cells empty.
5. Column widths: max of header/body/total cell byte lengths. Cells left-justified, joined by
   two spaces, each line right-trimmed of trailing spaces. Separator lines are dashes of each
   column width joined by two spaces (NOT trimmed differently — Python doesn't rstrip these,
   but dash lines have no trailing spaces anyway).
6. Order written: header, separator, body rows, separator, total.
7. If any row is unpriced, write `"\n? = includes an unpriced model; cost is a lower bound.\n"`
   to errW.

## Deliverable: `table_test.go`

Golden-string tests using `bytes.Buffer` for both writers:

- Empty rows → exactly `"No sessions found.\n"`.
- Two rows (one active, one unpriced, different name lengths so widths are exercised),
  showDate=true → compare the full output to a hand-written golden string; assert errW got
  the footnote.
- Same rows all-priced, showDate=false → no DATE column, `"2 sessions"` label, empty errW.

To build the golden strings correctly, run the Python script's `render_table` mentally or via
a scratch Python snippet with the same inputs; do not guess column widths — compute them.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run TestRenderTable ./...
```

Break the width calculation once (off-by-one), watch the golden test fail, restore.

## Out of scope

Only `table.go` and `table_test.go`.
