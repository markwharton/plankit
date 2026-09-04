---
name: guard
description: How pk guard protects branches - modes, the push policy, and how to change them
---

# pk guard

Guard is a PreToolUse hook on shell commands. When the agent is about
to run a git mutation (`commit`, `merge`, `push`, `rebase`, `reset`),
guard checks `.pk.json` and answers with a permission decision. Two
policies apply, and the strongest decision wins:

- branch policy (`guard.mode`): mutations while a protected branch
  (`guard.branches`) is checked out are denied (`block`), questioned
  (`ask`), or ignored (`off`)
- push policy (`guard.push`): any `git push`, on any branch, is
  denied, questioned, or ignored. Pushing is the developer's explicit
  action; commits stay local until you push them.

Guard recognizes git through chains (`&&`, `;`, `|`), environment
prefixes, absolute paths, `git -C`, and `git.exe`, so a compound
command is judged by its parts.

## What guard is

A guardrail against an agent following its defaults, not a security
boundary. Internal errors fail open with a note on stderr, and an
unconfigured repository (no `.pk.json`) is always a no-op.

## Changing policy

Edit `.pk.json` and commit it; policy travels with the repository.

```json
"guard": { "branches": ["main"], "mode": "block", "push": "block" }
```
