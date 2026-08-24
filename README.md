# sessions

List pi and Claude Code coding-agent sessions by date range, with token counts, costs, and duration.

## Install

### A. Install script (no Go required)

```bash
curl -fsSL https://raw.githubusercontent.com/jrdmcgr/agent-sessions/main/scripts/install.sh | sh
```

Detects your OS and architecture, downloads the matching prebuilt binary from the latest GitHub Release, verifies its checksum, and installs `sessions` to `~/.local/bin` (or `/usr/local/bin` if needed).

### B. `go install` (Go 1.22+)

```bash
go install github.com/jrdmcgr/agent-sessions@latest
```

### C. Manual download

Download a prebuilt binary from the [Releases page](https://github.com/jrdmcgr/agent-sessions/releases). Binaries are named `sessions_<version>_<os>_<arch>.tar.gz` for `darwin`/`linux` × `amd64`/`arm64`. Each release includes a `checksums.txt` file to verify the asset:

```bash
tar -xzf sessions_0.1.0_darwin_arm64.tar.gz
shasum -a 256 -c checksums.txt   # macOS; use sha256sum -c checksums.txt on Linux
```

## Usage

```
usage: sessions [-h] [--week [OFFSET] | --yesterday | --all] [--since YYYY-MM-DD]
                 [--until YYYY-MM-DD] [--project SUBSTR] [--harness {pi,claude}]
                 [--active] [--temp] [--json] [date]

List pi and Claude Code sessions worked on in a date range.

positional arguments:
  date                  a single day (YYYY-MM-DD)

options:
  -h, --help            show this help message and exit
  --week [OFFSET]       calendar week, Monday-based; OFFSET -1 is last week
  --yesterday           yesterday only
  --all                 every session on disk
  --since YYYY-MM-DD
  --until YYYY-MM-DD
  --project SUBSTR      filter by cwd/repo substring
  --harness {pi,claude}  filter by harness
  --active              only sessions touched in the last 2h
  --temp                include sessions run from temp dirs (prompt-eval fixtures)
  --json                emit JSON instead of a table

Defaults to today. Rows are session-days; totals sum the range.
```

### Inspecting a single transcript

Use the `sessions show` subcommand to inspect a single session transcript and extract metadata:

```bash
sessions show <path/to/transcript.jsonl> [--messages]
```

This reads a pi or Claude Code session file and emits a JSON record with session metadata, including date, harness, models used, tokens, cost, message count, and (with `--messages`) the full message history. Useful for integrating session data into scripts, archival systems, or analysis pipelines.

## Example

List all sessions on a given day (illustrative output — your projects and session names will differ):

```bash
$ sessions 2026-08-24

Wednesday 2026-08-24

│ Harness │ ID       │ Name              │ Project    │ Started │ Duration │ Messages │ Tokens  │ Cost   │
├─────────┼──────────┼───────────────────┼────────────┼─────────┼──────────┼──────────┼─────────┼────────┤
│ claude  │ a1b2c3d4 │ Fix flaky test    │ widget-api │ 08:11   │ 12 min   │ 8        │ 8,240   │ $0.13  │
│ claude  │ e5f6g7h8 │ Refactor CLI      │ dotfiles   │ 11:54   │ 1h 58m   │ 54       │ 312,415 │ $4.67  │
│ pi      │ i9j0k1l2 │ (unnamed)         │ my-project │ 15:40   │ 4 min    │ 2        │ 1,240   │ —      │

Total: 3 session-days │ 64 messages │ 321,895 tokens │ $4.80
```

List sessions from the last week, grouped by harness:

```bash
sessions --week
```

Filter to the past 2 hours:

```bash
sessions --active
```

Export today's sessions as JSON:

```bash
sessions --json > sessions.json
```

## Building from source

Clone the repo:

```bash
git clone https://github.com/jrdmcgr/agent-sessions.git
cd agent-sessions
```

Build the binary:

```bash
go build -o sessions .
```

Run the test suite:

```bash
go test ./...
```

## Project history

This tool was ported from a Python script and built task-by-task with subagents to establish repeatable distribution workflows for Go CLI tools. For background on the design and build process, see `PLAN.md` and the `docs/` directory. Build artifacts (`docs/TODO.md` and `tasks/*.md`) are the development log, not user documentation.

## License

MIT, see `LICENSE`.
