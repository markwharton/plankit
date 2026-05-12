# pk guard

Block git mutations on protected branches.

## Usage

`pk guard` is a PreToolUse hook — it runs automatically when Claude Code invokes Bash. It is not intended to be called directly.

## How it works

1. Reads the Bash command from the hook payload.
2. Splits compound commands (`&&`, `||`, `;`) and checks each subcommand for git mutations (`commit`, `push`, `merge`, `rebase`).
3. If any subcommand is a mutation, reads the `guard.branches` array from `.pk.json`.
4. Gets the current branch via `git rev-parse --abbrev-ref HEAD`.
5. If the current branch is in the protected list, blocks the command.

Read-only git commands (`status`, `log`, `diff`, `branch`, `fetch`) and non-git commands are always allowed.

By default, guard blocks outright — the command is denied and Claude Code must switch to the development branch. With `--ask`, guard prompts the user to confirm or reject instead, allowing legitimate overrides (emergency hotfix, manual recovery) without disabling the hook. The normal path — `pk release` — bypasses guard entirely because `pk release` uses `exec.Command` directly, not Bash.

## Flags

- **--ask** — Prompt the user instead of blocking. Use this for teams that want emergency override capability without disabling the hook.

## Configuration

Add a `guard` key to `.pk.json` in the project root:

```json
{
  "guard": {
    "branches": ["main"]
  }
}
```

Multiple branches can be protected:

```json
{
  "guard": {
    "branches": ["main", "production"]
  }
}
```

If `.pk.json` does not exist or has no `guard` key, `pk guard` is a no-op — the hook exits silently and allows all commands.

The `/conventions` skill can configure guard for you. When you specify protected branches, it creates the `.pk.json` guard configuration automatically.

## Hook protocol

- **Input:** PreToolUse JSON on stdin (includes `tool_input.command` for Bash).
- **Output:** `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}` on stdout to block. With `--ask`, uses `"permissionDecision":"ask"` to prompt the user instead. No output to allow. `hookEventName` is required by the Claude Code hook schema whenever `hookSpecificOutput` is present.
- **Exit code:** Always 0.

## Environment

- **CLAUDE_PROJECT_DIR** — Used to resolve the project root. Falls back to `cwd` from the hook payload.
