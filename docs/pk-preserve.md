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

- **--dry-run** — Preview the plan title, destination file, and commit message without writing, committing, or pushing. When no plan is found, prints a diagnostic to stderr explaining why (e.g., path didn't match the `.claude/plans/*.md` pattern, file not found).
- **--push** — Push to origin after committing. By default, `pk preserve` commits only — push when you're ready.
- **--notify** *(deprecated)* — Force manual (notify) mode: output a notification about the plan without preserving it. The mode now lives in `.pk.json` (`preserve.mode`); this flag is kept only so an old `pk preserve --notify` hook keeps working until `pk setup` rewrites it bare.

## Configuration

`preserve.mode` in `.pk.json` controls how the automatic `ExitPlanMode` hook behaves:

- **auto** — commit the plan to `docs/plans/` immediately.
- **manual** *(default)* — notify you to run `/preserve` when ready; nothing is committed.
- **off** — do nothing.

An explicit `/preserve` always commits, regardless of the mode. Set it with `pk setup --preserve <mode>`. See [.pk.json](pk-json.md#preserve).

## Hook protocol

- **Input:** PostToolUse JSON on stdin. The plan path comes from `tool_response`, which the harness sends as an object with a `filePath` field (a legacy plain-string form is also accepted). The JSON is parsed structurally so Windows paths with escaped backslashes resolve correctly.
- **Output:** `{"systemMessage": "..."}` on stdout (shown to user). In notify mode, also includes `{"hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": "..."}}` to inject context into Claude's next turn. `hookEventName` is required by the Claude Code hook schema whenever `hookSpecificOutput` is present.
- **Exit code:** Always 0. Errors are reported via stderr or systemMessage.

## Environment

- **CLAUDE_PROJECT_DIR** — Used to resolve the project root. Falls back to `cwd` from the hook payload.

## Details

### Race safety across Claude sessions

`~/.claude/plans/` is shared — every Claude Code session writes plans there. When multiple sessions are active, mtime-based selection in `findLatestPlan()` can pick the wrong plan. To close that window, `pk preserve --notify` writes the absolute path of the approved plan to `<projectDir>/.git/pk-pending-plan`. When the `/preserve` skill later invokes `pk preserve` (no hook stdin), that invocation reads the pointer and preserves the exact plan that was approved — even if a rival session has since bumped the mtime on a different `*.md` in `~/.claude/plans/`. The pointer is deleted after successful preservation (or when it points at a missing file). No `.gitignore` coordination is needed because `.git/` is never tracked.

### Team usage

The sequence number in filenames (e.g., `001`, `002`) is a local sort hint based on what exists in `docs/plans/` at the time of preservation. In a team setting, developers working in parallel may generate duplicate sequence numbers because each developer's local directory is a different snapshot. This is harmless — the slug portion ensures filenames are unique, and git will merge them without conflict. The sequence number provides useful ordering for a single developer; across a team, the date is the primary sort key.
