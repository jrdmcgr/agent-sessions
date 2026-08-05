# Task 02 — JSONL iteration, timestamp parsing, map getters

## Context

Read `PLAN.md`. Depends on task 01 (`types.go` exists). Ports `iter_jsonl` and `parse_ts` from
`~/Code/dotfiles/bin/sessions` (read that file first — the docstrings are the spec), plus
typed getters used by the readers in tasks 04/05.

## Deliverable: `jsonl.go` with exactly these functions

```go
// iterJSONL reads path line by line, decoding each non-blank line as a JSON
// object into map[string]any. Blank lines and lines that fail to decode are
// skipped. If the file cannot be opened, returns nil. Lines that decode to a
// non-object (e.g. a bare string) are skipped.
func iterJSONL(path string) []map[string]any

// parseTS parses an ISO-8601 string (usually Z-suffixed, e.g.
// "2026-08-05T14:03:22.123Z", but bare "2026-08-05T14:03:22" and offset forms
// like "2026-08-05T14:03:22+02:00" must also work) and converts it to a
// time.Time in time.Local. Returns the zero time.Time for a non-string value,
// empty string, or unparseable string. A timestamp with no zone is taken as
// already local (matching Python's naive-datetime behavior).
func parseTS(value any) time.Time

// getString returns m[key] if it is a non-empty string, else "".
func getString(m map[string]any, key string) string

// getMap returns m[key] if it is a map[string]any, else nil.
func getMap(m map[string]any, key string) map[string]any

// getInt64 returns m[key] as int64 if it is a JSON number, else 0.
// (encoding/json decodes numbers into float64.)
func getInt64(m map[string]any, key string) int64

// getBool returns m[key] if it is a bool, else false.
func getBool(m map[string]any, key string) bool

// firstText extracts the user-visible text from a message content field:
//   - if content is a string, return it as-is
//   - if content is a []any, return the "text" of the first element that is a
//     map with "type" == "text" (empty string if that text is missing/nil)
//   - otherwise return ""
// Ports first_text from the Python.
func firstText(content any) string
```

Use `time.Parse` with these layouts tried in order for `parseTS`:
`time.RFC3339Nano`, then `"2006-01-02T15:04:05.999999999"` parsed with
`time.ParseInLocation(..., time.Local)`. Convert zoned results with `.In(time.Local)`.

## Deliverable: `jsonl_test.go`

Table-driven tests covering at minimum:

- `iterJSONL`: temp file containing a valid object line, a blank line, a garbage line
  (`{not json`), a bare-string line (`"hi"`), then a valid line → returns exactly the 2 objects
  in order. Nonexistent path → nil.
- `parseTS`: Z-suffixed with millis, offset form, naive form (result must equal the same
  wall-clock in `time.Local`), `nil`, `""`, `"not-a-date"`, integer `5` → zero time for the
  last four.
- `firstText`: plain string; list with a `tool_use` block before a `text` block; list with no
  text block; nil.
- One test each for the four getters covering present/absent/wrong-type.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run 'TestIterJSONL|TestParseTS|TestFirstText|TestGet' ./...
```

Before finishing: break `parseTS` deliberately (e.g. return zero always), confirm the test
fails, then restore and confirm green.

## Out of scope

Do not touch `types.go`, `main.go`, or create any other file.
