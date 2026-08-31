# pk preserve

Save an approved plan to `docs/plans/` as a dated file and commit it.

## Usage

```bash
pk preserve                # preserve and commit the approved plan
pk preserve --push         # and push to origin
pk preserve --dry-run      # print the title, file and commit message; write nothing
pk preserve --notify       # deprecated; forces manual mode
```

`pk preserve` runs as a PostToolUse hook on `ExitPlanMode`, and directly through the `/preserve` skill.

## How it works

1. Reads the plan path from the hook payload on stdin, or, with no payload, from `.git/pk-pending-plan`.
2. Takes the title from the plan's first `# ` heading.
3. Names the file `{date}-{seq:03d}-{slug}.md` (`2026-04-05-001-add-auth-middleware.md`); `seq` counts the files already in `docs/plans/` for that date.
4. Skips a plan with the same content already preserved today.
5. Writes the file, runs `git add` and `git commit`; with `--push`, `git push origin HEAD`.
6. Prints `{"systemMessage": "..."}` on stdout.

In `manual` mode the hook does not commit; it writes the plan's absolute path to `.git/pk-pending-plan` and says the plan is ready. `/preserve` then reads that pointer, so the plan committed is the one approved, whichever session touched `~/.claude/plans/` since. The pointer is deleted after a successful preservation.

## Flags

- **--dry-run**: print what would be written and committed. With no plan found, prints why on stderr.
- **--push**: push after committing. Without it the commit stays local.
- **--notify** *(deprecated)*: force manual mode. The mode is read from `preserve.mode`; the flag is kept so an old hook line works until `pk setup` rewrites it bare.

## Configuration

`preserve.mode`: see [.pk.json](pk-json.md#preserve).

## Hook protocol

- **Input:** PostToolUse JSON on stdin. The plan path is `tool_response.filePath`; a plain-string `tool_response` is also accepted. The JSON is parsed, so Windows paths with escaped backslashes resolve.
- **Output:** `{"systemMessage": "..."}` on stdout. In manual mode, also `{"hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": "..."}}`, which reaches Claude's next turn.
- **Exit code:** always 0.

## Environment

- **CLAUDE_PROJECT_DIR**: see [Environment variables](environment-variables.md).

## Limits

- A plan produced remotely (Ultraplan) writes no local file and fires no `ExitPlanMode`, so nothing is preserved.
- In a team, two developers can produce the same sequence number for one date; the slug keeps the filenames distinct and git merges them without conflict.
