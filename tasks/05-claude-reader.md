# Task 05 — Claude Code transcript reader

## Context

Read `PLAN.md`. Depends on tasks 01–02. Ports `read_claude_session` from
`~/Code/dotfiles/bin/sessions` — read that function first; it is the spec.

## Claude Code transcript shape

One JSONL file per session at `~/.claude/projects/<slug>/<uuid>.jsonl`. Entry types:

| type           | fields used                                                              |
| -------------- | ------------------------------------------------------------------------ |
| `custom-title` | `customTitle`                                                            |
| `ai-title`     | `aiTitle`                                                                |
| `user`         | `cwd`, `sessionId`, `timestamp`, `isSidechain`, `message.content`        |
| `assistant`    | same as user, plus `message.model`, `message.usage`                      |

`message.usage` keys: `input_tokens`, `output_tokens`, `cache_read_input_tokens`,
`cache_creation_input_tokens` (note: creation → `Usage.CacheWrite`).

## Deliverable: `claude.go`

```go
// readClaudeSession parses one Claude Code transcript. Returns nil if the
// file yields no user/assistant events.
func readClaudeSession(path string) *Session
```

Required behaviors (each is a test case):

1. Entries other than the four types above are ignored; `user`/`assistant` entries with
   `isSidechain: true` are skipped entirely.
2. **cwd**: first non-empty `cwd` seen wins (`cwd = cwd or entry.cwd` in Python — note this is
   first-wins, opposite of pi's last-wins). Fallback when never set:
   `decodeSlug(filepath.Base(filepath.Dir(path)))` (function owned by task 06 — same stub rule
   as task 04 behavior 2).
3. **ID**: filename stem initially; any entry's non-empty `sessionId` overrides (last wins).
4. **Name**: `customTitle` if any entry set one, else `aiTitle`, else `""`. Later non-empty
   titles override earlier ones within each kind.
5. **model**: `message.model`, except the literal string `"<synthetic>"` becomes `""`. Models
   are NOT sticky across events here (unlike pi) — each event carries only its own.
6. **usage**: map the four snake_case keys; missing usage → zero. `Cost` is always nil.
7. **Role**: `"user"` or `"assistant"` from the entry type. `Text` from
   `firstText(message.content)`. `TS` from the entry's top-level `timestamp`.
8. `Harness` is `HarnessClaude`, `Path` is the input path. Empty/unreadable file → nil.

## Deliverable: `claude_test.go`

One main fixture (temp dir, e.g. `projects/-Users-x-Code-proj/uuid1.jsonl`) with: a
`custom-title` entry, an `ai-title` entry, a sidechained assistant entry (must be excluded),
a user entry (string content), an assistant entry with full usage and a real model, and an
assistant entry with model `"<synthetic>"`. Assert the full `Session`, including that the
name is the custom title and the sidechain event is absent. Small extra fixtures for: ai-title
only; no cwd anywhere (slug fallback); empty file → nil.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run TestReadClaudeSession ./...
```

Break the sidechain filter once, watch a test fail, restore, confirm green.

## Out of scope

Only `claude.go` and `claude_test.go`.
