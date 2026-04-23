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
#   The body of this script never runs locally.
# - In a Claude Code cloud sandbox (ephemeral VM with no pk installed):
#   downloads the matching pk release into $HOME/.local/bin so the
#   protective hooks (pk guard, pk preserve, pk protect) can run.

set -euo pipefail

command -v pk >/dev/null 2>&1 && exit 0

if ! command -v sha256sum >/dev/null 2>&1; then
  echo "pk install: sha256sum not found — cannot verify binary integrity" >&2
  exit 1
fi

PK_VERSION="{{VERSION}}"
install_dir="$HOME/.local/bin"
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
mv "$tmp" "$install_dir/pk"

[ -n "${CLAUDE_ENV_FILE:-}" ] && echo "export PATH=\"$install_dir:\$PATH\"" >> "$CLAUDE_ENV_FILE"
