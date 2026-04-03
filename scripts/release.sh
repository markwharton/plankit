#!/bin/bash
set -euo pipefail

# Release script for plankit
#
# Usage:
#   ./scripts/release.sh            # Validate and push tag at HEAD to trigger CI release
#   ./scripts/release.sh --dry      # Run all checks without pushing

BINARY_NAME="pk"
PLATFORMS=("darwin-amd64" "darwin-arm64" "linux-amd64" "linux-arm64" "windows-amd64")

# --- Parse arguments ---

DRY_RUN=false
if [ "${1:-}" = "--dry" ]; then
  DRY_RUN=true
fi

# Find version tag at HEAD.
TAG=$(git tag --points-at HEAD | grep '^v' | head -1)
if [ -z "$TAG" ]; then
  echo "Error: no version tag at HEAD — run 'pk changelog' first"
  exit 1
fi
VERSION="${TAG#v}"

# Validate semver format.
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Error: tag $TAG is not valid semver"
  exit 1
fi

echo "=== Release ${TAG} ==="
echo ""

# --- Pre-flight checks ---

echo "--- Pre-flight checks ---"

# 1. Clean working tree
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: working tree is not clean"
  git status --short
  exit 1
fi
echo "  Clean working tree"

# 2. On main branch
BRANCH=$(git branch --show-current)
if [ "$BRANCH" != "main" ]; then
  echo "Error: not on main branch (on: $BRANCH)"
  exit 1
fi
echo "  On main branch"

# 3. Not behind remote (local may be ahead after pk changelog commit)
git fetch origin main --quiet
MERGE_BASE=$(git merge-base HEAD origin/main)
REMOTE=$(git rev-parse origin/main)
if [ "$MERGE_BASE" != "$REMOTE" ]; then
  echo "Error: local main is behind origin/main — pull first"
  exit 1
fi
echo "  Not behind origin/main"

echo "  Tag $TAG exists at HEAD"

# --- Run tests ---

echo ""
echo "--- Running tests ---"
go test -race ./...
echo "  All tests passed"

# --- Verify cross-compilation ---

echo ""
echo "--- Cross-compile verification ---"
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

LDFLAGS="-s -w -X github.com/markwharton/plankit/internal/version.Version=${VERSION}"

for PLATFORM in "${PLATFORMS[@]}"; do
  OS="${PLATFORM%-*}"
  ARCH="${PLATFORM#*-}"
  EXT=""
  if [ "$OS" = "windows" ]; then EXT=".exe"; fi

  OUTPUT="${BUILD_DIR}/${BINARY_NAME}-${PLATFORM}${EXT}"
  GOOS="$OS" GOARCH="$ARCH" go build -ldflags "$LDFLAGS" -o "$OUTPUT" ./cmd/pk
  SIZE=$(du -h "$OUTPUT" | cut -f1 | xargs)
  echo "  ${BINARY_NAME}-${PLATFORM}${EXT}  ${SIZE}"
done

echo "  All 5 platforms built successfully"

# --- Push ---

if [ "$DRY_RUN" = true ]; then
  echo ""
  echo "--- Dry run complete ---"
  echo "  All checks passed. Run without --dry to push."
  exit 0
fi

echo ""
echo "--- Pushing to origin ---"
git push origin main "$TAG"
echo "  Pushed main and $TAG"

echo ""
echo "=== Release ${TAG} started ==="
echo "  Monitor: https://github.com/markwharton/plankit/actions"
echo "  Release: https://github.com/markwharton/plankit/releases/tag/${TAG}"
