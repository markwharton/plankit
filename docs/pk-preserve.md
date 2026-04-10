# pk preserve

Preserve an approved plan to `docs/plans/` as a timestamped file and commit it.

## Usage

```bash
pk preserve                # preserve and commit the most recent plan
pk preserve --push         # also push to origin after committing
pk preserve --dry-run      # preview without writing, committing, or pushing
pk preserve --notify       # check for a plan and notify without preserving
```

This command is designed to run as a **PostToolUse hook** on `ExitPlanMode`, but can also be invoked directly via the `/preserve` skill.

## How it works

1. Reads the hook payload from stdin (or falls back to the most recently modified plan in `~/.claude/plans/`).
2. Extracts the plan title from the first `# heading`.
3. Generates a filename: `{date}-{seq:03d}-{slug}.md` (e.g., `2026-04-05-001-add-auth-middleware.md`).
4. Checks for duplicate content (same plan already preserved today → skip).
5. Writes to `docs/plans/`, runs `git add` and `git commit`. If `--push` is set, also runs `git push origin HEAD`.
6. Outputs a `{"systemMessage": "..."}` JSON response on stdout.

## Flags

- **--dry-run** — Preview the plan title, destination file, and commit message without writing, committing, or pushing. Used by the `/preserve` skill for confirmation before proceeding.
- **--push** — Push to origin after committing. By default, `pk preserve` commits only — push when you're ready.
- **--notify** — Output a notification about the plan without preserving it. Used in manual mode to remind the user to run `/preserve` when ready. The response includes `additionalContext` so Claude is aware of the plan and can inform the user.

## Hook protocol

- **Input:** PostToolUse JSON on stdin (includes `tool_response` with the plan path).
- **Output:** `{"systemMessage": "..."}` on stdout (shown to user). In notify mode, also includes `{"hookSpecificOutput": {"additionalContext": "..."}}` to inject context into Claude's next turn.
- **Exit code:** Always 0. Errors are reported via stderr or systemMessage.

## Environment

- **CLAUDE_PROJECT_DIR** — Used to resolve the project root. Falls back to `cwd` from the hook payload.

## Details

### Team usage

The sequence number in filenames (e.g., `001`, `002`) is a local sort hint based on what exists in `docs/plans/` at the time of preservation. In a team setting, developers working in parallel may generate duplicate sequence numbers because each developer's local directory is a different snapshot. This is harmless — the slug portion ensures filenames are unique, and git will merge them without conflict. The sequence number provides useful ordering for a single developer; across a team, the date is the primary sort key.
