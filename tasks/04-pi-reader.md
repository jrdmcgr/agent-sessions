# Task 04 — pi transcript reader

## Context

Read `PLAN.md`. Depends on tasks 01–02 (uses `iterJSONL`, `parseTS`, getters, `firstText`).
Ports `read_pi_session` from `~/Code/dotfiles/bin/sessions` — read that function line by line
before writing any code; the behaviors below restate it but the Python wins on conflict.

## pi transcript shape

One JSONL file per session at `~/.pi/agent/sessions/<slug>/<...>_<id>.jsonl`. Entry types:

| type           | fields used                                                        |
| -------------- | ------------------------------------------------------------------ |
| `session`      | `cwd`, `id`                                                        |
| `model_change` | `modelId`                                                          |
| `session_info` | `name`                                                             |
| `message`      | `message.role`, `message.model`, `message.usage`, `message.content`|

`message.usage` keys: `input`, `output`, `cacheRead`, `cacheWrite`, and optionally
`cost.total` (a number = exact recorded cost).

## Deliverable: `pi.go`

```go
// readPiSession parses one pi transcript. Returns nil if the file yields no
// message events.
func readPiSession(path string) *Session
```

Required behaviors (each is a test case):

1. **ID fallback**: initial ID is the filename stem after the first `_`
   (e.g. `2026-08-05_abc123.jsonl` → `abc123`; a stem with no `_` → whole stem). A `session`
   entry's non-empty `id` overrides it.
2. **cwd**: from the `session` entry; later entries never blank an earlier value. If no entry
   set cwd, fall back to `decodeSlug(filepath.Base(filepath.Dir(path)))` — task 06 owns
   `decodeSlug`; if task 06 hasn't landed, declare it in `pi.go` temporarily is NOT allowed;
   instead add this one-liner to `pi.go` is also not allowed — coordinate: if `decodeSlug` is
   missing from the tree, stub it in `pi.go` as `func decodeSlug(slug string) string` per the
   Python `decode_slug` and note in your report that task 06 must delete its own copy.
3. **model is sticky**: `model_change.modelId` and `message.message.model` update the current
   model; message events with no model inherit the last one.
4. **name**: last non-empty `session_info.name` wins.
5. **usage**: missing usage map → zero Usage. `cost.total` present and numeric → `Cost`
   points at that value; otherwise `Cost` is nil.
6. **events**: one Event per `message` entry, in file order, with `TS` from the entry's
   top-level `timestamp`, `Role` from `message.role`, `Text` from `firstText(message.content)`.
7. **empty**: a file with only `session`/`model_change` entries returns nil. Unreadable file
   returns nil.
8. `Harness` is `HarnessPi`, `Path` is the input path.

## Deliverable: `pi_test.go`

Write a fixture transcript to a temp dir in the test (use `os.MkdirTemp` +
`filepath.Join(dir, "someslug", "2026-08-05_sess1.jsonl")`) exercising all behaviors above:
a `session` header with cwd+id, a `model_change`, a `session_info`, three `message` entries
(one with usage+cost, one with usage but no cost, one user message with list content whose
first text block is `"hello world"`). Assert the full parsed `Session` including every Event
field. Add separate small fixtures for behaviors 1 (no-underscore stem), 2 (no session entry →
slug fallback), and 7.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run TestReadPiSession ./...
```

Break the model-stickiness logic once, watch a test fail, restore, confirm green.

## Out of scope

Only `pi.go` and `pi_test.go` (plus the `decodeSlug` stub caveat in behavior 2).
