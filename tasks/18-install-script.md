# Task 18 — Install script for the no-Go-toolchain path

## Context

Read `docs/plans/003-distribution.md`, decisions 4 and 6. This is the script the README (task
17) tells people without Go to run. It must work standalone against the **asset naming contract**
in the plan without needing task 15's release workflow to have run yet — write it against the
contract, not against an actual release (there may not be one when you run this task).

## Deliverable: `scripts/install.sh`

POSIX `sh` (repo convention for shell in `bin/update-pricing` is POSIX `sh`, not bash — match it
here since this script is meant to run via `curl | sh` on an unknown shell). Requirements:

1. `set -eu` at the top.
2. Detect OS via `uname -s` → map `Darwin`→`darwin`, `Linux`→`linux`; anything else, print an
   error naming the unsupported OS and exit 1 (no Windows support, decision 3).
3. Detect arch via `uname -m` → map `x86_64`/`amd64`→`amd64`, `arm64`/`aarch64`→`arm64`; anything
   else, error and exit 1.
4. Resolve the latest release tag from GitHub's API:
   ```
   curl -fsSL https://api.github.com/repos/jrdmcgr/agent-sessions/releases/latest
   ```
   and extract `tag_name` (a `v`-prefixed tag, e.g. `v0.1.0`) without requiring `jq` (grep/sed is
   fine — this needs to run on a bare macOS/Linux box that may not have `jq`).
5. Build the asset URL for the detected OS/arch using the plan's naming contract, stripping the
   `v` prefix for the version number embedded in the filename:
   ```
   https://github.com/jrdmcgr/agent-sessions/releases/download/<tag>/sessions_<version>_<os>_<arch>.tar.gz
   ```
   e.g. tag `v0.1.0` → asset `sessions_0.1.0_darwin_arm64.tar.gz`.
6. Download the asset and `checksums.txt` from the same release into a temp dir
   (`mktemp -d`, cleaned up with a `trap ... EXIT`).
7. Verify the checksum: extract the line matching the asset filename from `checksums.txt` and
   check it with `shasum -a 256 -c` (macOS has `shasum`; prefer it, fall back to `sha256sum -c` if
   `shasum` isn't present). On mismatch, print an error and exit 1 — do not install.
8. Extract the tarball, and install the `sessions` binary:
   - If `~/.local/bin` exists and is on `$PATH`, install there, no `sudo`.
   - Otherwise, fall back to `/usr/local/bin`, using `sudo` only if the current user can't write
     to it directly (check with `[ -w /usr/local/bin ]` before reaching for `sudo`).
9. Print the installed path and `sessions --version` to confirm.
10. Support `curl -fsSL .../install.sh | sh` — no interactive prompts that would block under a
    pipe (decide the install location per point 8 automatically, don't ask).

## Acceptance criteria

Since there's no published release yet, you cannot run this end to end. Instead:

```sh
cd ~/Code/agent-sessions
sh -n scripts/install.sh          # syntax check, must exit 0
shellcheck scripts/install.sh     # if shellcheck is available; fix anything it flags at error/warning level
chmod +x scripts/install.sh       # must be executable
```

Then dry-run the logic that doesn't require a real release: comment out (in your own scratch
copy, not committed) the download step and confirm OS/arch detection prints the right
`sessions_<version>_<os>_<arch>.tar.gz` filename for your own machine's `uname -s`/`uname -m`.
Describe this manual check in your summary since it can't be a committed automated test — there's
nothing to assert against without a live release.

## Out of scope

No `.goreleaser.yml` (task 15) — this script only needs to agree with that task's output format,
not depend on its code. No README wiring (task 17 already includes the `curl | sh` line; don't
duplicate install instructions elsewhere).
