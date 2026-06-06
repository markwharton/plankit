# pk guard

Block git mutations on protected branches, and guard `git push` against unbidden pushes on any branch.

## Usage

`pk guard` is a PreToolUse hook — it runs automatically when Claude Code invokes Bash. It is not intended to be called directly.

## How it works

1. Reads the Bash command from the hook payload.
2. Splits compound commands (`&&`, `||`, `;`) and parses each subcommand's git operation, skipping git global options so `git -C dir push` and `git -c k=v commit` are recognized (not just `git push`/`git commit`).
3. **Branch policy:** if any subcommand mutates (`commit`, `push`, `merge`, `rebase`, `reset`), reads `guard.branches` from `.pk.json`, gets the current branch (`git rev-parse --abbrev-ref HEAD`), and if it is protected, decides per the branch mode (`--ask` → ask, else deny).
4. **Push policy:** if any subcommand is a `git push`, decides per `--push-guard` (`block` → deny, `ask` → ask, `off` → allow) regardless of branch.
5. **Strongest wins:** when both policies apply (e.g. a push on a protected branch), the strongest decision is emitted (deny > ask > allow), so a protected-branch push is never downgraded.

Read-only git commands (`status`, `log`, `diff`, `branch`, `fetch`) and non-git commands are always allowed.

The branch policy defaults to blocking outright; with `--ask` it prompts instead, allowing legitimate overrides (emergency hotfix, manual recovery) without disabling the hook. The push policy is independent and off unless `--push-guard` is set.

**pk-mediated pushes pass through.** `pk release`, `pk preserve --push`, and `pk setup --baseline --push` reach the hook as the command `pk release` etc. (not `git push`), so they are allowed; their internal push is a child process of `pk` via `exec.Command`, not a Bash tool call, so it is never hooked. The agent cannot hand-push under `--push-guard block`; pk's deliberate publish flows still work.

## Flags

- **--ask** — Branch policy: prompt the user instead of blocking on a protected branch. Use this for teams that want emergency override capability without disabling the hook.
- **--push-guard `<block|ask|off>`** — Push policy for `git push` on any branch: `block` denies, `ask` prompts, `off` (default) allows. Set it via `pk setup --push-guard`.

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
