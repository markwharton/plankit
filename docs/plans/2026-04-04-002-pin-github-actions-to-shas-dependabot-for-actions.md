# Pin GitHub Actions to SHAs + Dependabot for Actions

## Context

plankit's `.github/workflows/release.yml` references third-party actions by mutable major-version tags (`@v4`, `@v5`, `@v2`). If an upstream action repo is compromised, the tag could be repointed to malicious code, affecting the next legitimate release build. This is plankit's primary supply chain vector — the project has zero Go dependencies and a local-gated release flow.

The fix is two parts: pin actions to immutable commit SHAs, then add Dependabot scoped to `github-actions` only so the pins stay current.

## Changes

### 1. Pin actions to SHAs in `.github/workflows/release.yml`

Replace each `uses:` line with its full SHA, keeping a version comment:

```yaml
# Before
- uses: actions/checkout@v4
# After
- uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4.3.1

# Before
- uses: actions/setup-go@v5
# After
- uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff # v5.6.0

# Before
- uses: actions/upload-artifact@v4
# After
- uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02 # v4.6.2

# Before
- uses: actions/download-artifact@v4
# After
- uses: actions/download-artifact@d3f86a106a0bac45b974a628896c90dbdf5c8093 # v4.3.0

# Before
- uses: softprops/action-gh-release@v2
# After
- uses: softprops/action-gh-release@153bb8e04406b158c6c84fc1615b65b24149a1fe # v2.6.1
```

### 2. Add `.github/dependabot.yml`

Scoped exclusively to `github-actions` — no `gomod` or other ecosystems:

```yaml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
```

This means ~5 PRs per quarter at most (only when an action publishes a new release). Each PR will contain the updated SHA pin with version comment.

## Files

| File | Action |
|------|--------|
| `.github/workflows/release.yml` | Edit 5 `uses:` lines |
| `.github/dependabot.yml` | Create new file |

## Verification

1. Review the diff — each `uses:` line should have a 40-char SHA and a `# vX.Y.Z` comment
2. Confirm `.github/dependabot.yml` has only the `github-actions` ecosystem
3. Push to a branch, verify the workflow YAML parses correctly (GitHub will flag syntax errors on push)
4. Optionally trigger a dry-run release to confirm the pinned actions resolve correctly
