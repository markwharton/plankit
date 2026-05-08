# Environment Variables

Environment variables that pk reads or sets.

## Variables set by pk

### VERSION

Set by `pk changelog` when running `postVersion` and `preCommit` hooks. Contains the new version without the `v` prefix (e.g., `0.8.1`).

pk pre-expands `$VERSION` before passing the command to the shell, so hooks work on all platforms without relying on shell-specific variable expansion.

**Used by:** `pk changelog` hooks, `pk pin` (as positional argument in hook commands)

## Variables read by pk

### CLAUDE_PROJECT_DIR

Set by Claude Code. Contains the absolute path to the project root. Used by hook commands to resolve the project directory when the hook payload's `cwd` field might differ from the project root.

Falls back to the `cwd` field from the hook payload when not set.

**Used by:** `pk guard`, `pk preserve`, `pk protect`

## Filesystem paths

These are not environment variables but filesystem locations pk uses.

### ~/.cache/plankit/version-check.json

Daily cache for `pk version` update checks. Stores the latest version from GitHub Releases and the timestamp of the last check. Avoids repeated HTTP calls within 24 hours.

The parent directory follows `os.UserCacheDir()`: `~/Library/Caches` on macOS, `~/.cache` on Linux, `%LocalAppData%` on Windows.

### ~/.claude/plans/

Claude Code writes plan files here when exiting plan mode. `pk preserve` reads the most recent plan from this directory when no hook payload is available. Shared across all Claude Code sessions.

### .git/pk-pending-plan

Pointer file written by `pk preserve --notify` containing the absolute path to the approved plan. Read by the subsequent `pk preserve` invocation to preserve the exact plan that was approved, avoiding race conditions when multiple Claude Code sessions are active. Deleted after successful preservation.
