---
name: spawn-subagent
description: >-
  Dispatch a task to a pi subagent running in an isolated git worktree, capturing
  its cost/duration (persisted session) and the commit it produces. Use when a
  plan has been split into independent task files and you want to run one or more
  in parallel, or when a single task's context is genuinely better handled in
  isolation (bulk mechanical edits, wide independent recon). One subagent per call;
  fan out by calling repeatedly.
metadata:
  scope: project
---

Run one task in a child pi session, isolated in a worktree, with its work
captured as a reviewable commit and its cost visible to the `sessions` report.

**When NOT to reach for this.** The standing rule (`MEMORY.md`) is *don't delegate
to subagents to protect context* — delegation is slow and pollutes nothing you'd
gain. This skill is the endorsed exception: the child's context is *better*, not
merely separate — a task-scoped worktree, no parent-conversation baggage. If the
work would be faster answered in-session, answer it in-session.

## Dispatch

One call per task; the parent fans out for parallelism.

```
bin/spawn-subagent tasks/03-pricing.md
bin/spawn-subagent tasks/04-pi-reader.md
```

- Task file → wrapped as `Read @<abs> and follow the instructions`; a bare string
  is used as a literal prompt. A commit directive is appended either way.
- Worktrees are **on by default** (each child gets its own branch + cwd, so
  parallel children never collide and each lands in its own session slug-dir).
  `--no-worktree` runs in place.
- The child writes a **persisted** session (never `--no-session`) so `sessions`
  picks up its cost/duration. Do not route it to a nested `--session-dir` — the
  report globs one directory deep and would miss it.

**Isolation is porous — task specs must not name absolute repo paths.** The child
is pinned to its worktree by cwd and by an explicit directive, but a task file
that says `cd ~/Code/<repo>` will walk it out onto the parent's live branch. Write
task specs to run in the current directory. The spawner guards this: if the parent
checkout's HEAD moves during a worktree run, it prints a `WARNING: the child
escaped its worktree` — treat that as a failed dispatch and reconcile the stray
commit before continuing.

The script prints a result block per child: `branch`, `worktree`, `base`/`head`
sha, whether it committed, and the cleanup command.

## Verify before trusting — do not merge on the commit alone

`AGENTS.md §5`: a commit that could not have failed is not evidence. For each child:

1. **Confirm it committed, and stayed put** — `committed: yes` in the result block
   and no escape `WARNING`. `NO` means the child did nothing; a `WARNING` means it
   committed onto the parent branch instead. Either way, investigate before retrying.
2. **Read the diff** — `git -C <worktree> diff <base>..<head>`.
3. **Run the repo's checks** in the worktree (its tests / lint), not just the
   parent's. Make sure they actually exercise the change.
4. **Integrate** — merge or rebase the child's `subagent/*` branch into your
   working branch once it passes.
5. **Clean up** — `git worktree remove <worktree>` (printed in the result block),
   then delete the branch if you're done with it.

## Promotion

While under test the script lives in this repo's `bin/`. When proven, it moves to
`~/Code/dotfiles/bin/` (on `PATH`) and this skill to `~/Code/agent/skills/`, at
which point `bin/spawn-subagent` becomes bare `spawn-subagent`.
