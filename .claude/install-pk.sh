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

PK_VERSION="v0.11.1"
install_dir="$HOME/.local/bin"
mkdir -p "$install_dir"

arch="$(uname -m)"
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 0 ;;
esac

url="https://github.com/markwharton/plankit/releases/download/${PK_VERSION}/pk-linux-${arch}"
curl -fsSL "$url" -o "$install_dir/pk"
chmod +x "$install_dir/pk"

[ -n "${CLAUDE_ENV_FILE:-}" ] && echo "export PATH=\"$install_dir:\$PATH\"" >> "$CLAUDE_ENV_FILE"
