# Extract session-parsing out of `archive-session`

## Why

`~/Code/dotfiles/bin/archive-session` (Python) does three jobs in one script:

| Concern | Should live in | Notes |
|---|---|---|
| Parse/normalize a transcript into session facts | **agent-sessions** (this tool) | Harness knowledge (pi↔claude shapes, tool-name mapping) gets one home |
| Render + write a note to `~/Notes` | note-archiver consumer | Vault policy: frontmatter, sentinels, delta, index, filenames |
| Write a daily log line to `~/Code/agent` | memory-logger consumer | Agent-repo policy: upsert, git commit |

Because the one script owns all three, `~/Code/agent` depends on `dotfiles`. The fix is to give
the shared parsing a home in `sessions`, then let each consumer shell out to it. **This plan is
only the extraction** — the first row. Splitting the two consumers comes after, once the tool
supplies everything they need.

Scope discipline: `sessions` stays a pure reader of what is in the transcript. It does not compute
facts that aren't there (no `git` subprocess) and does not format output (no markdown). It
normalizes and emits structured JSON; consumers render.

## What the tool is missing today

Per-session metadata `archive-session` extracts that `sessions` does not expose (or conflates):

| Field | State in `sessions` |
|---|---|
| `git_branch` | ✗ — see decision 1 |
| `provider` | ✗ (pi records it; claude is "anthropic") |
| `slug`, `version` | ✗ |
| `custom_title` vs `ai_title` | conflated into `Name` |
| session-level `started_at`/`ended_at` | only per-day buckets exist |
| full message content + tool calls | ✗ — `Event.Text` keeps only the first text block; tool_use dropped |
| per-message `uuid` | ✗ — needed for the note's append-only delta mode |

Two structural gaps:
- **Single-file mode.** Hooks hand the tool one transcript path (Claude via stdin JSON, pi via
  argv). `sessions` only discovers-from-roots today.
- **Per-session output.** Notes are one-per-session (delta-appended across days); the daily log is
  one line bucketed by last timestamp. Both want session-level aggregation, not the session-*day*
  rows `sessions` emits.

## Decisions

1. **pi `git_branch` → pi extension, not a subprocess in the tool.** Claude records `gitBranch` on
   every entry; a `git rev-parse` at parse time answers the wrong question (archiving runs at
   session end, and multi-day sessions change branch). The branch is a fact only the harness knows
   at session time, so capture it there: a pi extension stamps the branch into pi's session data,
   and `sessions` reads one field from both harnesses. Caveats: forward-only (old pi transcripts
   never get a branch); a new moving part to install via `bin/pi-hook`.

   **Verified feasible (2026-08-12).** pi's `session` header carries no branch, and
   `ctx.sessionManager` is a `ReadonlySessionManager` (no append). But the `pi` ExtensionAPI
   exposes `pi.appendEntry(customType, data)` — "append a custom entry for state persistence, not
   sent to LLM" (used by `preset.ts`) — and `pi.exec` runs git (as `git-checkpoint.ts` does). So
   the extension resolves the branch and stamps it without polluting LLM context. Mechanism: on
   `session_start` and `turn_start`, run `git rev-parse --abbrev-ref HEAD`; when the branch differs
   from the last recorded, `pi.appendEntry("git-branch", { branch })`. This gives Claude-like
   mid-session change tracking. `sessions` reads `type:"custom"` entries with
   `customType:"git-branch"` and takes the last `data.branch`.
2. **No markdown in the tool; no separate markdown binary yet.** `sessions` emits structured,
   normalized blocks (text + `tool_use` with canonical names + inputs). The note-archiver renders
   markdown internally from that JSON. The normalization/formatting seam:
   - Normalize tool identity (pi `bash` / claude `Bash` → one canonical block) → **tool**.
   - Render a block to `` > [tool] Bash: `cmd` ``, `## User`/`## Claude`, sentinels → **consumer**.
   A standalone `session-md` filter is deferred until a second consumer (e.g. terminal viewing)
   exists; the JSON boundary makes that a cheap later extraction, not a rewrite.
3. **Emit both message counts.** `Messages` (all events, for the table) and a renderable count
   (non-empty, non-noise — what Python's `render_blocks` counts). The tool shouldn't decide which a
   consumer wants; since it already walks the messages, expose both.
4. **Deferred:** the daily-log line references the vault note filename (`note ...`), a live coupling
   between the two consumers. Resolve it when refactoring `archive-session`, not now.
5. **Emit per-message `uuid`** (claude `uuid`, pi `id`). It's the note delta-state key, but it's a
   session fact, so it belongs in the tool's output.

## Per-session JSON schema (`sessions show <path>`)

The contract both future consumers bind to. Metadata is always emitted; message blocks only when
`--messages` is passed, so the memory-logger's call stays cheap and only the note-archiver pays for
the body. The argument is a transcript **path** (the hook's case); id-lookup is a cheap later add.

```jsonc
{
  "harness": "pi",
  "session_id": "019ff189…",
  "path": "/…/sessions/…/<ts>_<uuid>.jsonl",
  "cwd": "/Users/JaredMcGuire/Code/agent",
  "project": "agent",                 // basename(cwd)
  "git_branch": "main",               // "" if absent
  "provider": "anthropic",            // "" if absent
  "model": "claude-sonnet-5",         // last model used
  "models": ["claude-sonnet-5"],      // order of first use
  "slug": "",                         // claude only
  "version": "",                      // claude_version; "" if absent
  "custom_title": "",                 // raw, "" if absent
  "ai_title": "",                     // raw, "" if absent
  "name": "resolved display title",   // custom_title || ai_title || fallbackName
  "summary": "first substantive user message, tags stripped, ≤120",
  "started_at": "2026-08-11T11:57:00",// session-level, full precision
  "ended_at":   "2026-08-11T12:23:00",
  "message_count": 28,                // all events (matches the table's Messages)
  "renderable_count": 24,             // non-empty, non-noise user+assistant
  "usage": { "input": 0, "output": 0, "cache_read": 0, "cache_write": 0 },
  "tokens": 0, "cost": 0.0, "priced": true,

  // only when --messages is passed:
  "messages": [
    { "uuid": "a1b2c3d4", "role": "user", "ts": "…",
      "blocks": [ { "type": "text", "text": "…" } ] },
    { "uuid": "b2c3d4e5", "role": "assistant", "ts": "…", "model": "claude-sonnet-5",
      "blocks": [
        { "type": "text", "text": "…" },
        { "type": "tool_use", "name": "Bash", "input": { "command": "ls" } }
      ] }
  ]
}
```

Schema notes:
- **Blocks normalized to Claude spelling.** The pi→Claude map remaps both the tool name
  (`bash`→`Bash`) and arg keys (read's `path`→`file_path`) — that's normalization, so it lives in
  the tool. `thinking` and `tool_result` blocks are dropped, matching `archive-session`.
- **`tool_use` carries `name` + raw `input`, never a rendered string** (decision 2); the archiver
  formats the summary line itself.
- **`--messages` gates the `messages` array.** A 195-message session's blocks are large and the
  memory-logger doesn't need them; default output is metadata + counts only.
- **Per-assistant-message `model`** is included: a free session fact, and the model can change
  mid-session. Consumers may ignore it.

## Phases

Each phase's new tests must be seen to fail first (repo convention; `AGENTS.md §5`).

### Phase 1 — enrich parsing (no output change)
- `Session`: add `GitBranch`, `Provider`, `Slug`, `Version`, `CustomTitle`, `AITitle`; keep `Name`
  as the resolved display title.
- `Event`: add `UUID`, and normalized content blocks (text + `tool_use{name,input}`) alongside the
  existing `Text` (which the table/`fallbackName` path keeps using untouched).
- Populate all of the above in `pi.go` and `claude.go`. Move the pi→Claude tool-name map out of
  the Python into the tool.

### Phase 2 — single-file, per-session output
- `sessions show <path>`: detect harness from the file (port `detect_harness`), parse one
  transcript.
- Emit one JSON record per session: full metadata + ordered messages (role, ts, uuid, normalized
  blocks) + both message counts + session-level `started_at`/`ended_at` + derived `summary`
  (port `extract_summary`: first substantive user message, tags stripped, truncated 120).

### Phase 3 — verify by parity
- Golden test: new per-session JSON matches Python's `collect_metadata` + normalized blocks on
  shared fixtures. Break the code to see it fail, then restore.

## Follow-on (named, not in this plan)
Two thin consumers each call `sessions show`:
- **note-archiver** stays in `dotfiles`: frontmatter, callout, sentinels/delta, vault paths, index,
  markdown rendering of the message blocks.
- **memory-logger** moves into `~/Code/agent`: daily line, upsert, git commit — the move that
  finally cuts the `dotfiles` dependency.
