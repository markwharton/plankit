#!/bin/bash
set -euo pipefail

# Release script for plankit
#
# Usage:
#   ./scripts/release.sh 1.0.0        # Tags v1.0.0 and pushes to trigger CI release
#   ./scripts/release.sh 1.0.0 --dry  # Run all checks without tagging or pushing

BINARY_NAME="pk"
PLATFORMS=("darwin-amd64" "darwin-arm64" "linux-amd64" "linux-arm64" "windows-amd64")

# --- Parse arguments ---

if [ $# -lt 1 ]; then
  echo "Usage: $0 <version> [--dry]"
  echo "  version: semver without 'v' prefix (e.g., 1.0.0)"
  echo "  --dry:   run checks only, don't tag or push"
  exit 1
fi

VERSION="$1"
DRY_RUN=false
if [ "${2:-}" = "--dry" ]; then
  DRY_RUN=true
fi

TAG="v${VERSION}"

# Validate semver format
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Error: version must be semver (e.g., 1.0.0), got: $VERSION"
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

# 3. Up to date with remote
git fetch origin main --quiet
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)
if [ "$LOCAL" != "$REMOTE" ]; then
  echo "Error: local main is not up to date with origin/main"
  echo "  local:  $LOCAL"
  echo "  remote: $REMOTE"
  exit 1
fi
echo "  Up to date with origin/main"

# 4. Tag doesn't already exist
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Error: tag $TAG already exists"
  exit 1
fi
echo "  Tag $TAG is available"

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

# --- Tag and push ---

if [ "$DRY_RUN" = true ]; then
  echo ""
  echo "--- Dry run complete ---"
  echo "  All checks passed. Run without --dry to tag and push."
  exit 0
fi

echo ""
echo "--- Tagging ${TAG} ---"
git tag "$TAG"
echo "  Created tag $TAG"

echo ""
echo "--- Pushing tag to origin ---"
git push origin "$TAG"
echo "  Pushed $TAG"

echo ""
echo "=== Release ${TAG} started ==="
echo "  Monitor: https://github.com/markwharton/plankit/actions"
echo "  Release: https://github.com/markwharton/plankit/releases/tag/${TAG}"
