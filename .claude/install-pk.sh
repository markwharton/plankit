#!/usr/bin/env bash
set -euo pipefail

command -v pk >/dev/null 2>&1 && exit 0

PK_VERSION="v0.9.0"
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
