# pk guard

Block git mutations on protected branches.

## Usage

`pk guard` is a PreToolUse hook — it runs automatically when Claude Code invokes Bash. It is not intended to be called directly.

## How it works

1. Reads the Bash command from the hook payload.
2. Checks if the command is a git mutation (`commit`, `push`, `merge`, `rebase`).
3. If it is, reads the `guard.protectedBranches` array from `.pk.json`.
4. Gets the current branch via `git rev-parse --abbrev-ref HEAD`.
5. If the current branch is in the protected list, blocks the operation.

Read-only git commands (`status`, `log`, `diff`, `branch`, `fetch`) and non-git commands are always allowed.

## Configuration

Add a `guard` key to `.pk.json` in the project root:

```json
{
  "guard": {
    "protectedBranches": ["main"]
  }
}
```

Multiple branches can be protected:

```json
{
  "guard": {
    "protectedBranches": ["main", "production"]
  }
}
```

If `.pk.json` does not exist or has no `guard` key, `pk guard` is a no-op — the hook exits silently and allows all commands.

The `/init` skill can configure guard for you. When you specify protected branches during initialization, it creates the `.pk.json` guard configuration automatically.

## Hook protocol

- **Input:** PreToolUse JSON on stdin (includes `tool_input.command` for Bash).
- **Output:** `{"decision":"block","reason":"..."}` on stdout to block. No output to allow.
- **Exit code:** Always 0.
