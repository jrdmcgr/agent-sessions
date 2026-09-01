#!/bin/sh
set -eu

# Detect OS
os_name=$(uname -s)
case "$os_name" in
  Darwin)
    os="darwin"
    ;;
  Linux)
    os="linux"
    ;;
  *)
    printf "Error: unsupported OS %s\n" "$os_name" >&2
    exit 1
    ;;
esac

# Detect architecture
arch_name=$(uname -m)
case "$arch_name" in
  x86_64|amd64)
    arch="amd64"
    ;;
  arm64|aarch64)
    arch="arm64"
    ;;
  *)
    printf "Error: unsupported architecture %s\n" "$arch_name" >&2
    exit 1
    ;;
esac

# Fetch latest release tag from GitHub API
printf "Fetching latest release...\n" >&2
release_response=$(curl -fsSL https://api.github.com/repos/jrdmcgr/agent-sessions/releases/latest)
tag=$(printf '%s' "$release_response" | grep -o '"tag_name":"[^"]*' | sed 's/"tag_name":"//' | head -n1)

if [ -z "$tag" ]; then
  printf "Error: could not determine latest release tag\n" >&2
  exit 1
fi

# Strip 'v' prefix from tag to get version number
version=$(printf '%s' "$tag" | sed 's/^v//')

# Build asset filename and URL
asset_filename="sessions_${version}_${os}_${arch}.tar.gz"
download_url="https://github.com/jrdmcgr/agent-sessions/releases/download/${tag}/${asset_filename}"
checksums_url="https://github.com/jrdmcgr/agent-sessions/releases/download/${tag}/checksums.txt"

printf "Installing %s (%s/%s, version %s)\n" "sessions" "$os" "$arch" "$version" >&2

# Create temp directory and set up cleanup trap
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

# Download asset and checksums
printf "Downloading %s...\n" "$asset_filename" >&2
curl -fsSL -o "$temp_dir/$asset_filename" "$download_url"

printf "Downloading checksums...\n" >&2
curl -fsSL -o "$temp_dir/checksums.txt" "$checksums_url"

# Verify checksum
printf "Verifying checksum...\n" >&2
cd "$temp_dir"

# Extract the checksum line for our asset and verify
checksum_line=$(grep "$asset_filename" checksums.txt)
if [ -z "$checksum_line" ]; then
  printf "Error: checksum for %s not found in checksums.txt\n" "$asset_filename" >&2
  exit 1
fi

# Use shasum if available, otherwise fall back to sha256sum
if command -v shasum >/dev/null 2>&1; then
  if ! printf '%s' "$checksum_line" | shasum -a 256 -c >/dev/null 2>&1; then
    printf "Error: checksum verification failed for %s\n" "$asset_filename" >&2
    exit 1
  fi
else
  if ! printf '%s' "$checksum_line" | sha256sum -c >/dev/null 2>&1; then
    printf "Error: checksum verification failed for %s\n" "$asset_filename" >&2
    exit 1
  fi
fi

printf "Checksum verified.\n" >&2

# Extract tarball
printf "Extracting %s...\n" "$asset_filename" >&2
tar -xzf "$asset_filename"

if [ ! -f "$temp_dir/sessions" ]; then
  printf "Error: sessions binary not found in extracted archive\n" >&2
  exit 1
fi

# Determine install location, respecting XDG conventions.
#
# There's no official XDG "bin" dir, but $XDG_BIN_HOME is the de facto
# convention used by installers that follow the XDG Base Directory spec
# (e.g. cargo, rustup-adjacent tooling). Fall back to
# $XDG_DATA_HOME/../bin (i.e. ~/.local/bin when XDG_DATA_HOME is unset,
# since XDG_DATA_HOME defaults to ~/.local/share), then ~/.local/bin
# directly, then /usr/local/bin.
install_path=""

if [ -n "${XDG_BIN_HOME:-}" ] && [ -d "$XDG_BIN_HOME" ] && printf '%s' "$PATH" | grep -q "$XDG_BIN_HOME"; then
  install_path="$XDG_BIN_HOME"
fi

if [ -z "$install_path" ] && [ -n "${XDG_DATA_HOME:-}" ]; then
  candidate="$(dirname "$XDG_DATA_HOME")/bin"
  if [ -d "$candidate" ] && printf '%s' "$PATH" | grep -q "$candidate"; then
    install_path="$candidate"
  fi
fi

if [ -z "$install_path" ] && [ -d "$HOME/.local/bin" ]; then
  if printf '%s' "$PATH" | grep -q "$HOME/.local/bin"; then
    install_path="$HOME/.local/bin"
  fi
fi

if [ -z "$install_path" ]; then
  if [ -w /usr/local/bin ]; then
    install_path="/usr/local/bin"
  else
    install_path="/usr/local/bin"
    use_sudo=1
  fi
fi

# Install binary
printf "Installing to %s...\n" "$install_path" >&2

if [ "${use_sudo:-0}" -eq 1 ]; then
  sudo install -m 755 "$temp_dir/sessions" "$install_path/sessions"
else
  install -m 755 "$temp_dir/sessions" "$install_path/sessions"
fi

# Verify installation and show version
printf "Installed at: %s\n" "$install_path/sessions"
"$install_path/sessions" --version
