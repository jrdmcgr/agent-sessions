#!/usr/bin/env bash
# Cut a new agent-sessions release and update the Homebrew formula that
# tracks it, so `brew upgrade agent-sessions` picks it up.
#
# The version lives in a git tag, nowhere else: `go build -ldflags
# -X main.version=...` is filled in by the Homebrew formula at build time
# from the tag in its `url`, so "bump the version" means: tag this repo,
# push it, then point the formula at the new tag with a fresh sha256.
#
# Usage:
#   scripts/release.sh <patch|minor|major> ["release notes for the tag"]
#   scripts/release.sh --dry-run <patch|minor|major>
#
# Env:
#   FORMULA_REPO   path to the homebrew-formulae checkout
#                  (default: ~/Code/formulae)
#   FORMULA_FILE   formula file within that repo
#                  (default: Formula/agent-sessions.rb)
#
# semver convention for this tool: patch = bugfix, no interface change;
# minor = backward-compatible addition (new field, new flag); major = a
# removed/renamed field or flag, or changed default behavior.
set -euo pipefail

FORMULA_REPO="${FORMULA_REPO:-$HOME/Code/formulae}"
FORMULA_FILE="${FORMULA_FILE:-Formula/agent-sessions.rb}"
REPO_SLUG="jrdmcgr/agent-sessions"

dry_run=false
if [[ "${1:-}" == "--dry-run" ]]; then
  dry_run=true
  shift
fi

bump="${1:-}"
notes="${2:-}"
case "$bump" in
  patch|minor|major) ;;
  *)
    echo "usage: $(basename "$0") [--dry-run] <patch|minor|major> [\"release notes\"]" >&2
    exit 2
    ;;
esac

function run {
  if $dry_run; then
    printf '[dry-run] %s\n' "$*"
  else
    "$@"
  fi
}

function require_clean_repo {
  local repo="$1" label="$2"
  if [[ ! -d "$repo/.git" ]]; then
    echo "error: $label repo not found at $repo" >&2
    exit 1
  fi
  if [[ -n "$(git -C "$repo" status --porcelain)" ]]; then
    echo "error: $label repo ($repo) has uncommitted changes; commit or stash first" >&2
    exit 1
  fi
  local branch
  branch=$(git -C "$repo" symbolic-ref --short HEAD)
  if [[ "$branch" != "main" && "$branch" != "master" ]]; then
    echo "error: $label repo is on branch '$branch', not main/master" >&2
    exit 1
  fi
}

function latest_tag {
  git -C "$sessions_repo" tag --list 'v*' --sort=-v:refname | head -1
}

# next_version computes vX.Y.Z from a vX.Y.Z tag and a bump kind.
function next_version {
  local tag="$1" kind="$2"
  local ver="${tag#v}"
  IFS='.' read -r major minor patch <<<"$ver"
  case "$kind" in
    major) echo "v$((major + 1)).0.0" ;;
    minor) echo "v${major}.$((minor + 1)).0" ;;
    patch) echo "v${major}.${minor}.$((patch + 1))" ;;
  esac
}

sessions_repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

require_clean_repo "$sessions_repo" "agent-sessions"
require_clean_repo "$FORMULA_REPO" "formula"

current_tag="$(latest_tag)"
if [[ -z "$current_tag" ]]; then
  echo "error: no existing vX.Y.Z tag found in $sessions_repo" >&2
  exit 1
fi
new_tag="$(next_version "$current_tag" "$bump")"

echo "agent-sessions: $current_tag -> $new_tag ($bump)"

# 1. Tag and push agent-sessions.
tag_msg="$new_tag"
if [[ -n "$notes" ]]; then
  tag_msg="$new_tag: $notes"
fi
run git -C "$sessions_repo" push origin HEAD
run git -C "$sessions_repo" tag -a "$new_tag" -m "$tag_msg"
run git -C "$sessions_repo" push origin "$new_tag"

# 2. Fetch the tag's release tarball and hash it — the exact artifact
#    Homebrew will download, so the sha256 must come from this, not from
#    `git archive` (GitHub's archive tarballs are not byte-identical to a
#    local git archive of the same tree).
tarball_url="https://github.com/${REPO_SLUG}/archive/refs/tags/${new_tag}.tar.gz"
if $dry_run; then
  echo "[dry-run] curl -fsSL $tarball_url | shasum -a 256"
  sha256="<dry-run-sha256>"
else
  echo "Downloading $tarball_url ..."
  tmp_tarball="$(mktemp)"
  trap 'rm -f "$tmp_tarball"' EXIT
  # GitHub needs a moment after a tag push before the archive endpoint
  # reflects it; retry briefly instead of failing on the first 404.
  for attempt in 1 2 3 4 5; do
    if curl -fsSL "$tarball_url" -o "$tmp_tarball"; then
      break
    fi
    if [[ "$attempt" == 5 ]]; then
      echo "error: could not download $tarball_url after 5 attempts" >&2
      exit 1
    fi
    sleep 3
  done
  sha256="$(shasum -a 256 "$tmp_tarball" | cut -d' ' -f1)"
fi
echo "sha256: $sha256"

# 3. Point the formula at the new tag + sha256.
formula_path="$FORMULA_REPO/$FORMULA_FILE"
if [[ ! -f "$formula_path" ]]; then
  echo "error: formula not found at $formula_path" >&2
  exit 1
fi

version_no_v="${new_tag#v}"
if $dry_run; then
  echo "[dry-run] would rewrite $formula_path url -> tags/${new_tag}.tar.gz, sha256 -> $sha256"
else
  sed -i.bak \
    -e "s|archive/refs/tags/v[0-9][0-9.]*\.tar\.gz|archive/refs/tags/${new_tag}.tar.gz|" \
    -e "s|sha256 \".*\"|sha256 \"${sha256}\"|" \
    "$formula_path"
  rm -f "$formula_path.bak"
fi

run git -C "$FORMULA_REPO" add "$FORMULA_FILE"
run git -C "$FORMULA_REPO" commit -m "agent-sessions ${version_no_v}"
run git -C "$FORMULA_REPO" push origin HEAD

if $dry_run; then
  echo "[dry-run] done. Re-run without --dry-run to actually cut $new_tag."
else
  echo
  echo "Done. To pick it up:"
  echo "  brew update && brew upgrade agent-sessions"
fi
