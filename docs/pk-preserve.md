# pk preserve

Preserve an approved plan to `docs/plans/` as a timestamped file, committed and pushed.

## Usage

```bash
pk preserve                # preserve the most recent plan
pk preserve --dry-run      # preview without writing, committing, or pushing
pk preserve --notify       # check for a plan and notify without preserving
```

This command is designed to run as a **PostToolUse hook** on `ExitPlanMode`, but can also be invoked directly via the `/preserve` skill.

## How it works

1. Reads the hook payload from stdin (or falls back to the most recently modified plan in `~/.claude/plans/`).
2. Extracts the plan title from the first `# heading`.
3. Generates a filename: `{date}-{seq:03d}-{slug}.md` (e.g., `2026-04-05-001-add-auth-middleware.md`).
4. Checks for duplicate content (same plan already preserved today → skip).
5. Writes to `docs/plans/`, runs `git add`, `git commit`, and `git push origin HEAD`.
6. Outputs a `{"systemMessage": "..."}` JSON response on stdout.

## Flags

- **--dry-run** — Preview the plan title, destination file, and commit message without writing, committing, or pushing. Used by the `/preserve` skill for confirmation before proceeding.
- **--notify** — Output a notification about the plan without preserving it. Used in manual mode to remind the user to run `/preserve` when ready.

## Hook protocol

- **Input:** PostToolUse JSON on stdin (includes `tool_response` with the plan path).
- **Output:** `{"systemMessage": "..."}` on stdout.
- **Exit code:** Always 0. Errors are reported via stderr or systemMessage.
