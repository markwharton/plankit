# pk protect

Block Claude Code from editing or overwriting files in `docs/plans/`.

## Usage

`pk protect` runs as a **PreToolUse hook** on `Edit` and `Write` tool calls. It is not intended to be run manually.

## How it works

1. Reads the hook payload from stdin to get the target file path.
2. Checks if the file is under `docs/plans/` relative to the project directory.
3. If yes, outputs a block decision. If no, exits silently to allow the operation.

## Hook protocol

- **Input:** PreToolUse JSON on stdin (includes `tool_input.file_path`).
- **Output:** `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}` on stdout to block. No output to allow. `hookEventName` is required by the Claude Code hook schema whenever `hookSpecificOutput` is present.
- **Exit code:** Always 0.

## Environment

- **CLAUDE_PROJECT_DIR** — Used to resolve the project root. Falls back to `cwd` from the hook payload.
