# pk protect

Block Claude Code from editing or writing files under `docs/plans/`.

## Usage

`pk protect` runs as a PreToolUse hook on the `Edit` and `Write` tools. It is not run by hand, and it has no mode: it is on whenever the hook is installed.

## How it works

1. Reads `tool_input.file_path` from the hook payload.
2. Resolves a relative path against the project directory and follows symlinks.
3. Denies a path under `docs/plans/`; otherwise writes nothing, which allows the edit.

## Hook protocol

- **Input:** PreToolUse JSON on stdin; `tool_input.file_path` is read.
- **Output:** `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}` to block; nothing to allow.
- **Exit code:** always 0.

## Environment

- **CLAUDE_PROJECT_DIR**: see [Environment variables](environment-variables.md).
