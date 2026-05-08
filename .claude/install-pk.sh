#!/usr/bin/env bash
#
# install-pk.sh — bootstraps the pk binary for Claude Code sessions.
#
# Managed by `pk setup`. Regenerated on every setup run and removed on
# `pk teardown`. Don't edit this file — customizations will be overwritten.
#
# When it runs:
# - Configured as a SessionStart hook, so Claude Code invokes it at the
#   start of every session.
#
# What it does:
# - On a developer machine with pk already on PATH: exits immediately.
# - On a developer machine without pk: prints a warning and exits.
# - In a Claude Code cloud sandbox (ephemeral VM with no pk installed):
#   downloads the matching pk release into $HOME/.local/share/pk/<version>
#   and prepends that directory to PATH for the session, so the protective
#   hooks (pk guard, pk preserve, pk protect) can run. Each version lives in
#   its own directory; sessions on different pinned versions don't collide.

set -euo pipefail

# Older install-pk.sh versions wrote pk to $HOME/.local/bin/pk, and some
# cloud sandbox base images plant a stale pk at that path on every restart.
# Clear it unconditionally so the gate below can't be tripped by a leftover
# binary. set -e surfaces permission errors loud rather than silently
# leaving a stale pk in place.
rm -f "$HOME/.local/bin/pk"

# If pk is already on PATH, it's a developer's own install (e.g.,
# ~/go/bin/pk). Skip our install so theirs stays in use.
command -v pk >/dev/null 2>&1 && exit 0

# On local machines (no CLAUDE_ENV_FILE), pk isn't installed but we can't
# download it either. Warn so the developer knows hooks won't run.
if [ -z "${CLAUDE_ENV_FILE:-}" ]; then
  echo "pk is not installed. Hooks (guard, preserve, protect) will not run." >&2
  echo "Install: go install github.com/markwharton/plankit/cmd/pk@latest" >&2
  exit 1
fi

PK_VERSION="v0.19.2"
install_dir="$HOME/.local/share/pk/$PK_VERSION"
binary="$install_dir/pk"

# Append a PATH export to CLAUDE_ENV_FILE only if it isn't already there;
# guards against PATH growing on repeat invocations within the same session.
append_path_once() {
  local dir="$1"
  local file="${CLAUDE_ENV_FILE:-}"
  [ -z "$file" ] && return 0
  local line="export PATH=\"$dir:\$PATH\""
  if ! grep -qxF "$line" "$file" 2>/dev/null; then
    echo "$line" >> "$file"
  fi
}

if [ -x "$binary" ]; then
  append_path_once "$install_dir"
  exit 0
fi

if ! command -v sha256sum >/dev/null 2>&1; then
  echo "pk install: sha256sum not found — cannot verify binary integrity" >&2
  exit 1
fi

mkdir -p "$install_dir"

arch="$(uname -m)"
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 0 ;;
esac

base="https://github.com/markwharton/plankit/releases/download/${PK_VERSION}"
tmp="$(mktemp -t pk.XXXXXX)"
sums="$(mktemp -t pk-sums.XXXXXX)"
trap 'rm -f "$tmp" "$sums"' EXIT

curl -fsSL "$base/pk-linux-${arch}" -o "$tmp"
curl -fsSL "$base/checksums.txt" -o "$sums"

expected="$(awk -v n="pk-linux-${arch}" '$2 == n {print $1}' "$sums")"
if [ -z "$expected" ]; then
  echo "pk install: checksum not published for pk-linux-${arch}" >&2
  exit 1
fi

actual="$(sha256sum "$tmp" | awk '{print $1}')"
if [ "$expected" != "$actual" ]; then
  echo "pk install: checksum mismatch for pk-linux-${arch}" >&2
  echo "  expected: $expected" >&2
  echo "  actual:   $actual" >&2
  exit 1
fi

chmod +x "$tmp"
mv "$tmp" "$binary"

append_path_once "$install_dir"

# Sandboxes clone only the working branch, so version tags aren't present
# locally until we fetch them — pk changelog / pk release need them to
# anchor history. Best-effort; never block session start.
if git rev-parse --git-dir >/dev/null 2>&1 && git remote get-url origin >/dev/null 2>&1; then
  git fetch origin --tags --quiet 2>/dev/null \
    || echo "pk install: git fetch --tags origin failed (non-fatal)" >&2
fi
