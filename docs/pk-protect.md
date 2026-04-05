# pk protect

Block Claude Code from editing or overwriting files in `docs/plans/`.

## Usage

```bash
pk protect    # reads PreToolUse hook payload from stdin
```

This command runs as a **PreToolUse hook** on `Edit` and `Write` tool calls. It is not intended to be run manually.

## How it works

1. Reads the hook payload from stdin to get the target file path.
2. Checks if the file is under `docs/plans/` relative to the project directory.
3. If yes, outputs a block decision. If no, exits silently to allow the operation.

## Hook protocol

- **Input:** PreToolUse JSON on stdin (includes `tool_input.file_path`).
- **Output (blocked):** `{"decision": "block", "reason": "docs/plans/ files are immutable historical records..."}` on stdout.
- **Output (allowed):** No stdout output.
- **Exit code:** Always 0.

## Environment

- **CLAUDE_PROJECT_DIR** — Used to resolve the project root. Falls back to `cwd` from the hook payload.
