# pk guard

Block git mutations on protected branches, and guard `git push` against unbidden pushes on any branch.

## Usage

`pk guard` is a PreToolUse hook — it runs automatically when Claude Code invokes Bash. It is not intended to be called directly.

## How it works

1. Reads the Bash command from the hook payload.
2. Splits compound commands (`&&`, `||`, `;`) and parses each subcommand's git operation, skipping git global options so `git -C dir push` and `git -c k=v commit` are recognized (not just `git push`/`git commit`).
3. **Branch policy:** if any subcommand mutates (`commit`, `push`, `merge`, `rebase`, `reset`), reads `guard.branches` and `guard.mode` from `.pk.json`, gets the current branch (`git rev-parse --abbrev-ref HEAD`), and if it is protected, decides per `guard.mode` (`block` → deny, `ask` → ask, `off` → allow; default `block`).
4. **Push policy:** if any subcommand is a `git push`, decides per `guard.push` from `.pk.json` (`block` → deny, `ask` → ask, `off` → allow; default `block`) regardless of branch.
5. **Strongest wins:** when both policies apply (e.g. a push on a protected branch), the strongest decision is emitted (deny > ask > allow), so a protected-branch push is never downgraded.

Read-only git commands (`status`, `log`, `diff`, `branch`, `fetch`) and non-git commands are always allowed.

The branch policy defaults to `block`; `guard.mode: ask` prompts instead, allowing legitimate overrides (emergency hotfix, manual recovery) without disabling the hook. The push policy is independent and also defaults to `block`.

**pk-mediated pushes pass through.** `pk release`, `pk preserve --push`, and `pk setup --baseline --push` reach the hook as the command `pk release` etc. (not `git push`), so they are allowed; their internal push is a child process of `pk` via `exec.Command`, not a Bash tool call, so it is never hooked. The agent cannot hand-push under `--push-guard block`; pk's deliberate publish flows still work.

## Flags

`pk guard` reads its modes from `.pk.json`. These flags are **deprecated** overrides, kept only so an old hook that still passes them keeps working until `pk setup` rewrites the hook bare. Set `guard.mode` / `guard.push` in `.pk.json` instead.

- **--ask** *(deprecated)* — Force ask mode for the branch policy. Use `guard.mode: "ask"`.
- **--push-guard `<block|ask|off>`** *(deprecated)* — Force the push policy. Use `guard.push`.

## Configuration

The `guard` key in `.pk.json` holds the branches plus the two modes (`pk setup` writes `mode`/`push`; `/conventions` or you set `branches`):

```json
{
  "guard": {
    "branches": ["main", "production"],
    "mode": "block",
    "push": "block"
  }
}
```

Any absent key falls back to its default (`mode` `block`, `push` `block`); `"off"` is an explicit, distinct value. With no `guard.branches`, the **branch** policy is a no-op, but the **push** policy still applies (default `block`), so the agent's `git push` is blocked unless `guard.push` is `"off"`.

The `/conventions` skill can set `guard.branches` for you, field-merging it into the `guard` object without disturbing the modes.

## Hook protocol

- **Input:** PreToolUse JSON on stdin (includes `tool_input.command` for Bash).
- **Output:** `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}` on stdout to block. With `--ask`, uses `"permissionDecision":"ask"` to prompt the user instead. No output to allow. `hookEventName` is required by the Claude Code hook schema whenever `hookSpecificOutput` is present.
- **Exit code:** Always 0 by design. If the binary itself crashes before it can respond (Go fatal errors, such as running out of memory at startup, exit with status 2), Claude Code treats exit 2 as a blocking error: the command is blocked and stderr is fed back to Claude, so guard fails closed and a retry succeeds once the cause clears. A missing binary (exit 127) is non-blocking and the command proceeds unguarded — see [When pk is not installed](adoption.md#when-pk-is-not-installed). See [out of memory at startup](error-reference.md#out-of-memory-at-startup) in the error reference.

## Environment

- **CLAUDE_PROJECT_DIR** — Used to resolve the project root. Falls back to `cwd` from the hook payload.
