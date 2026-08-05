# Task 06 — Transcript discovery and slug decoding

## Context

Read `PLAN.md`. Depends on task 01 only — do not wait on tasks 04/05. Ports `decode_slug`
and `discover` from `~/Code/dotfiles/bin/sessions` — read them first. If tasks 04/05 landed a
temporary `decodeSlug` stub (their instructions allow one), delete the stub and own the
function here.

`discoverIn` takes the two readers as parameters instead of calling `readPiSession`/
`readClaudeSession` by name. That is the only deliberate deviation from the Python (which has
a module-level `discover()` that hardcodes both readers) — it exists so this file, and its
tests, build and pass standalone against task 01 alone, with no dependency on tasks 04/05.
Task 11 (main wiring) calls `discoverIn` directly, passing `readPiSession`/`readClaudeSession`
as the reader arguments — there is no separate `discover()` wrapper anywhere in this codebase.

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

// discoverIn scans piRoot and claudeRoot for "*/*.jsonl" files (exactly one
// directory deep, sorted lexically by full path within each root, pi root's
// files before claude root's) whose mtime is >= cutoff. A zero cutoff means
// no filtering. Roots that don't exist are skipped; files that can't be
// stat'ed are skipped. Every file under piRoot is paired with piReader in
// the result; every file under claudeRoot is paired with claudeReader.
func discoverIn(piRoot, claudeRoot string, piReader, claudeReader sessionReader, cutoff time.Time) []discoveredFile
```

Implementation notes:

- Use `filepath.Glob(filepath.Join(root, "*", "*.jsonl"))` — glob results are already sorted.
- mtime comparison: keep the file when `!info.ModTime().Before(cutoff)` (Python keeps
  `mtime >= cutoff`; "last write can't precede work").
- Do not add a `discover(since time.Time) []discoveredFile` wrapper function. It is
  deliberately not part of this task (see Context) — adding it anyway just reintroduces the
  04/05 build dependency this task exists to avoid.

## Deliverable: `discover_test.go`

Pass small stub `sessionReader` values as the two reader arguments — e.g. two distinct
closures like `func(string) *Session { return nil }` assigned to named variables, so you can
assert which stub ended up on which `discoveredFile` (compare with
`reflect.ValueOf(f.reader).Pointer()`, or simpler: give each stub a unique sentinel behavior
and call `f.reader("")` in the assertion, comparing results). `discoverIn` never calls the
readers itself; it only attaches them.

Build a temp tree:

```
tmp/pi/slugA/a.jsonl        (mtime: now)
tmp/pi/slugA/old.jsonl      (mtime: 3 days ago, via os.Chtimes)
tmp/pi/nested/deep/x.jsonl  (must NOT be found — too deep)
tmp/pi/top.jsonl            (must NOT be found — too shallow)
tmp/claude/slugB/b.jsonl    (mtime: now)
```

Assert:

- zero cutoff → exactly `a.jsonl`, `old.jsonl`, `b.jsonl`, pi files first, sorted; the pi
  files carry the pi stub reader, `b.jsonl` carries the claude stub reader.
- cutoff = yesterday midnight → `old.jsonl` excluded.
- nonexistent roots → empty slice, no panic.
- `decodeSlug("-Users-JaredMcGuire-Code-dotfiles")` → `/Users/JaredMcGuire/Code/dotfiles`;
  `decodeSlug("")` → `/`.

## Acceptance criteria (all must exit 0)

```sh
cd ~/Code/agent-sessions
go build ./... && go vet ./... && go test -run 'TestDiscoverIn|TestDecodeSlug' ./...
grep -c 'func decodeSlug' *.go   # must print 1 for exactly one file
grep -c 'func discover(' *.go    # must print 0 — no discover() wrapper in this codebase
```

## Out of scope

Only `discover.go` and `discover_test.go` (plus deleting a `decodeSlug` stub from
`pi.go`/`claude.go` if present).
