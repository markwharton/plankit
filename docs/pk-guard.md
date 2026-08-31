# pk guard

Block git mutations on protected branches, and `git push` on any branch, inside a Claude Code session.

## Usage

`pk guard` runs as a PreToolUse hook on the Bash and PowerShell tools. It is not run by hand.

## How it works

1. Reads the command from the hook payload.
2. Splits it on `&&`, `||`, `|`, `|&`, `;` and newlines, ignoring operators inside quotes. In each part, skips leading `VAR=value` assignments and a leading `command`, matches git by path basename (`/usr/bin/git`), and skips git's global options (`git -C dir push`, `git -c k=v commit`).
3. **Branch policy**: a part that runs `commit`, `push`, `merge`, `rebase` or `reset` while the current branch (`git rev-parse --abbrev-ref HEAD`) is in `guard.branches` is decided by `guard.mode`.
4. **Push policy**: a part that runs `git push`, on any branch, is decided by `guard.push`.
5. When both apply, the stronger decision is emitted: deny, then ask, then allow.

Anything else is allowed. `pk release`, `pk preserve --push`, `pk setup --baseline --push` and `pk init --push` reach the hook as `pk …`, not `git push`; their pushes are pk's own child processes and are never hooked.

## Flags

Deprecated overrides, kept so a hook written before the modes moved to `.pk.json` keeps working until `pk setup` rewrites it bare:

- **--ask**: force `ask` for the branch policy. Use `guard.mode`.
- **--push-guard `<block|ask|off>`**: force the push policy. Use `guard.push`.

## Configuration

`guard.branches`, `guard.mode` and `guard.push`: see [.pk.json](pk-json.md#guard).

## Hook protocol

- **Input:** PreToolUse JSON on stdin; `tool_input.command` is read.
- **Output:** `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}` to block, `"ask"` to prompt, nothing to allow.
- **Exit code:** always 0. A crash before the response exits 2, which Claude Code treats as a block with stderr shown to Claude. A missing `pk` exits 127, which Claude Code treats as non-blocking: the command runs unguarded.

## Environment

- **CLAUDE_PROJECT_DIR**: see [Environment variables](environment-variables.md).

## Limits

- The guard reads the command text, not its intent: a heredoc line beginning `git push` is blocked.
- Command substitution and subshells are not parsed.
- It fails open: a missing binary leaves the session unguarded.
