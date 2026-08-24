# Plan: distribute `sessions` to people without a Go toolchain

## Why

Today the only way to run this tool is `go build -o sessions .` from a clone. That's fine for a
Go developer; it's a wall for everyone else. Distribution needs two paths that both land on the
same binary:

1. **Has Go installed** → `go install github.com/jrdmcgr/agent-sessions@latest`.
2. **Doesn't** → download a prebuilt binary from a GitHub Release (via an install script or by
   hand), no compiler required.

Both paths require the repo to be **public** and to have **tagged releases**. Those two steps are
Jared's call, not a subagent's (§3: outward-facing) — see "Manual steps," last.

## Decisions

1. **Binary name is `sessions`**, regardless of the module/repo name `agent-sessions`. The
   `usageText` in `main.go` already says `usage: sessions …`; every build (goreleaser included)
   uses `-o sessions` / `binary: sessions`.
2. **GoReleaser** builds and publishes cross-platform binaries on tag push. It's the standard tool
   for "one Go repo, many OS/arch tarballs on GitHub Releases" — avoids hand-rolling a matrix build
   in raw Actions YAML.
3. **Release matrix:** `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`. No Windows for
   now (nobody asking for it; add a GOOS later if that changes).
4. **Asset naming (fixed contract, both the release config and the install script depend on it
   matching exactly):**
   ```
   sessions_<version>_<os>_<arch>.tar.gz
   ```
   e.g. `sessions_0.1.0_darwin_arm64.tar.gz`, containing one file: `sessions`. Also produced:
   `checksums.txt` (sha256, one line per asset) in the same release.
5. **Version is baked in at build time** via `-ldflags "-X main.version=..."`, surfaced as
   `sessions --version`. Local `go build` (no ldflags) falls back to `"dev"`.
6. **Install script** (`scripts/install.sh`) is the "don't have Go" path: detects OS/arch, pulls
   the matching asset from the latest GitHub Release, verifies its checksum, and drops `sessions`
   into `~/.local/bin` (falls back to prompting for `sudo` + `/usr/local/bin` if `~/.local/bin`
   isn't on `PATH` — decided by the script, not the user, at install time). Documented as:
   ```
   curl -fsSL https://raw.githubusercontent.com/jrdmcgr/agent-sessions/main/scripts/install.sh | sh
   ```
7. **No Homebrew tap yet.** A tap is a second repo to maintain in sync with every release; the
   install script covers the same need with less ongoing cost. Revisit if enough people ask for
   `brew install`.
8. **License: MIT.** Smallest-friction choice for a single-purpose CLI with no dependencies on
   copyleft code (`lipgloss`/`go-isatty` are both MIT).
9. **CI has two workflows, kept separate** because they trigger differently and fail differently:
   - `test.yml` — `go build`, `go vet`, `go test ./...` on every push/PR. Fast feedback, gates
     merges.
   - `release.yml` — GoReleaser, triggered only on `v*` tag push. Only runs at release time; a
     failure here doesn't block ordinary development.

## Task breakdown

Seven tasks, independent of each other's files (safe to farm to parallel subagents off the same
base commit):

| Task | Files touched | Depends on |
|---|---|---|
| 13 — repo hygiene | `LICENSE`, `.gitignore`, gofmt on 3 existing files | none |
| 14 — version flag | new `version.go`, one-line hooks in `main.go` + `cli.go` | none |
| 15 — release automation | new `.goreleaser.yml`, new `.github/workflows/release.yml` | none (reads the contract in this plan, not task 14's code) |
| 16 — CI test workflow | new `.github/workflows/test.yml` | none |
| 17 — README | rewrite `README.md` | none (reads the contract in this plan) |
| 18 — install script | new `scripts/install.sh` | none (reads the contract in this plan) |
| 19 — module path | `go.mod` only | none |

Every task gets the asset-naming and version contracts (decisions 4–6) restated inline so none of
them need to read another task's output to stay consistent — the plan is the shared source of
truth, not inter-task handoffs.

## Manual steps (Jared, after tasks 13–19 are merged)

These are outward-facing/irreversible (§3) and stay out of subagent scope:

1. Flip repo visibility to public: `gh repo edit jrdmcgr/agent-sessions --visibility public`.
2. Confirm `go install github.com/jrdmcgr/agent-sessions@latest` resolves once public.
3. Tag `v0.1.0` and push the tag — triggers `release.yml`, which runs GoReleaser.
4. Verify the release: download one asset by hand, run `sha256sum -c checksums.txt`, run the
   binary.
5. Test the install script end to end on a machine with no Go toolchain.

## Follow-on (named, not in this plan)

- **Homebrew tap**, if demand shows up (decision 7).
- **Windows build**, if demand shows up (decision 3).
