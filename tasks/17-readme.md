# Task 17 — README rewrite

## Context

Read `docs/plans/003-distribution.md` in full — this task documents the install paths and
naming contracts that plan defines, including ones other tasks (15, 18) are building in parallel.
Current `README.md` is a one-line stub (`# agent-sessions`). This task replaces it entirely.
Independent of every other task — it only needs to *describe* the contracts (asset naming,
install script URL, `go install` path), not their implementations to exist yet.

## What the tool does (for the intro paragraph)

`sessions` lists coding-agent sessions (pi and Claude Code) worked on in a date range, reading
their JSONL transcripts from `~/.pi/agent/sessions/` and `~/.claude/projects/`, bucketing
activity into session-days, and rendering a table (or JSON) with tokens, cost, and duration.
Read `main.go`'s `usageText` constant and `show.go` for the exact current flag set and the `show`
subcommand — don't invent flags or behavior; transcribe what's there.

## Deliverable: `README.md`

Sections, in order:

1. **Title + one-line description.**
2. **Install** — three ways, in this order (most people hitting the README from a release
   announcement want option A or B, not to clone the repo):
   - **A. Install script** (no Go required):
     ```
     curl -fsSL https://raw.githubusercontent.com/jrdmcgr/agent-sessions/main/scripts/install.sh | sh
     ```
   - **B. `go install`** (if you have Go 1.22+):
     ```
     go install github.com/jrdmcgr/agent-sessions@latest
     ```
   - **C. Manual download** — link to the Releases page
     (`https://github.com/jrdmcgr/agent-sessions/releases`), and note the asset naming:
     `sessions_<version>_<os>_<arch>.tar.gz` for `darwin`/`linux` × `amd64`/`arm64`, plus a
     `checksums.txt` to verify against.
3. **Usage** — reproduce the `usageText` block from `main.go` verbatim (read the file, copy it,
   don't retype from memory), plus a short paragraph on the `sessions show <path.jsonl>
   [--messages]` subcommand for inspecting a single transcript.
4. **Example** — one realistic invocation and a short sample of its table output. If you can run
   the real binary against your own `~/.pi/agent/sessions` or `~/.claude/projects` data, use real
   (but not personally sensitive — redact any project paths/names you'd rather not publish)
   output; otherwise construct a plausible example and label it as illustrative.
5. **Building from source** — `git clone`, `go build -o sessions .`, one line pointing at
   `go test ./...` for the test suite.
6. **Project history** — one or two sentences: this was ported from a Python script and built
   task-by-task with subagents; point at `PLAN.md` and `docs/` for anyone curious how, and note
   that `docs/TODO.md` and `tasks/*.md` are build-process artifacts, not user documentation.
7. **License** — one line: "MIT, see `LICENSE`."

## Acceptance criteria

- Every flag in the `usageText` constant (`main.go`) appears in the README's Usage section.
- The three install methods are present with the exact URLs/commands above.
- `LICENSE` is referenced (it may not exist yet in your worktree if task 13 hasn't landed — that's
  fine, just write the line; don't block on it).
- Run a spellcheck/read-through: no first-person "I" (this is user-facing docs, not a diary entry
  like `PLAN.md`/`TODO.md`).

## Out of scope

No changes to `PLAN.md`, `docs/TODO.md`, or any `tasks/*.md` file — those stay as the build
diary. No code changes.
