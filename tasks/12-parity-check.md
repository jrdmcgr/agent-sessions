# Task 12 — Parity verification against the Python original

## Context

Read `PLAN.md`. Final gate; requires tasks 01–11 merged and real session data on this machine
(`~/.pi/agent/sessions`, `~/.claude/projects`). This task runs the Go binary and the Python
script (`~/Code/dotfiles/bin/sessions`) side by side and proves equivalence.

## Deliverable: `bin/parity` (executable shell script)

```sh
#!/bin/sh
# Compare Go and Python `sessions` output. Exits nonzero on any divergence.
set -eu
```

The script must:

1. `go build -o "$TMPDIR/sessions-go" .` from the repo root.
2. For each flag set in:
   `--all --json`, `--json` (today), `--week --json`, `--week -1 --json`,
   `--yesterday --json`, `--since 2026-01-01 --until 2026-06-30 --json`,
   `--project agent --json`, `--harness pi --json`, `--harness claude --json`,
   `--active --json`, `--temp --all --json`
   run both implementations and compare with jq normalization that:
   - sorts keys (`jq -S`),
   - rounds `cost` to 4 decimals (`.[] |= (.cost = ((.cost * 10000 | round) / 10000))`),
   so float formatting differences don't count as divergence. Print `OK <flags>` or
   `DIFF <flags>` plus the diff; any DIFF fails the run.
3. Table parity: for `--all`, `--week`, and today's default, diff stdout byte-for-byte
   (`diff <(python) <(go)`), EXCEPT rows whose Active flag could flip between the two
   invocations — avoid the race by running both within the same second and accepting that a
   rare flake means rerun. Any persistent diff fails.
4. Exit-code parity: both return 0 on success; both return 2 on `--harness bogus`
   (verify Python's argparse exit code first — argparse exits 2 — and assert both).

## Procedure (for the agent running this task)

1. Write the script, `chmod +x bin/parity`.
2. Run it. For every DIFF, diagnose whether Go or the spec-reading is wrong, fix in the
   owning file, add a unit test in that file's `_test.go` capturing the divergence, rerun.
3. Iterate until the script prints all OK.

## Known acceptable divergences (do not "fix")

- Float formatting inside JSON (handled by jq rounding).
- Exit 130 on ^C (Python) vs default signal death (Go).
- Usage/help text wording.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go test ./...
bin/parity        # prints OK for every flag set, exits 0
```

## Deliverable: report

List every divergence found and which task's file had the bug — this is the measure of how
well the task specs held up.
