# Task 06 — Transcript discovery and slug decoding

## Context

Read `PLAN.md`. Depends on task 01. Ports `decode_slug` and `discover` from
`~/Code/dotfiles/bin/sessions` — read them first. If tasks 04/05 landed a temporary
`decodeSlug` stub (their instructions allow one), delete the stub and own the function here.

## Deliverable: `discover.go`

```go
// decodeSlug converts "-Users-JaredMcGuire-Code-dotfiles" to
// "/Users/JaredMcGuire/Code/dotfiles" — strip leading/trailing '-', replace
// '-' with '/', prepend "/". Lossy for paths containing real dashes; that is
// accepted. Ports decode_slug.
func decodeSlug(slug string) string

// sessionReader parses one transcript file into a Session (or nil).
type sessionReader func(path string) *Session

// discoveredFile pairs a transcript path with the reader that understands it.
type discoveredFile struct {
	path   string
	reader sessionReader
}

// discoverIn scans the two roots for "*/*.jsonl" files (exactly one directory
// deep, sorted lexically by full path within each root, pi root first) whose
// mtime is >= cutoff. A zero cutoff means no filtering. Roots that don't
// exist are skipped; files that can't be stat'ed are skipped.
func discoverIn(piRoot, claudeRoot string, cutoff time.Time) []discoveredFile

// discover calls discoverIn with the real roots (piSessionsDir(),
// claudeSessionsDir()) and a cutoff of local midnight of `since`
// (dayOf(since)), or zero cutoff if since is the zero time.
func discover(since time.Time) []discoveredFile
```

Implementation notes:

- Use `filepath.Glob(filepath.Join(root, "*", "*.jsonl"))` — glob results are already sorted.
- mtime comparison: keep the file when `!info.ModTime().Before(cutoff)` (Python keeps
  `mtime >= cutoff`; "last write can't precede work").
- `discoverIn` exists so tests can point at temp roots; `discover` is a thin wrapper.
- The reader for files under `piRoot` is `readPiSession`; under `claudeRoot`,
  `readClaudeSession`. **Execute this task only after tasks 04 and 05 are merged** — those
  functions must already exist or the build fails.

## Deliverable: `discover_test.go`

Build a temp tree:

```
tmp/pi/slugA/a.jsonl        (mtime: now)
tmp/pi/slugA/old.jsonl      (mtime: 3 days ago, via os.Chtimes)
tmp/pi/nested/deep/x.jsonl  (must NOT be found — too deep)
tmp/pi/top.jsonl            (must NOT be found — too shallow)
tmp/claude/slugB/b.jsonl    (mtime: now)
```

Assert:

- zero cutoff → exactly `a.jsonl`, `old.jsonl`, `b.jsonl`, pi files first, sorted.
- cutoff = yesterday midnight → `old.jsonl` excluded.
- nonexistent roots → empty slice, no panic.
- `decodeSlug("-Users-JaredMcGuire-Code-dotfiles")` → `/Users/JaredMcGuire/Code/dotfiles`;
  `decodeSlug("")` → `/`.

## Acceptance criteria

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run 'TestDiscover|TestDecodeSlug' ./...
grep -c 'func decodeSlug' *.go   # must print 1 for exactly one file
```

## Out of scope

Only `discover.go` and `discover_test.go` (plus deleting a `decodeSlug` stub from
`pi.go`/`claude.go` if present).
