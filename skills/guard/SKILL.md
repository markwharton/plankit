---
name: guard
description: How pk guard protects branches - the mode, push, and breaking-marker dials, and how to change them
---

# pk guard

Guard is a PreToolUse hook on shell commands. Before the agent runs a
git mutation (`commit`, `merge`, `push`, `rebase`, `reset`), guard
reads `.pk.json` and answers with a permission decision. Three
policies apply; the strongest decision wins, and deny beats ask.

- branch policy (`guard.mode`): a mutation while a protected branch
  (`guard.branches`) is checked out is denied (`block`), questioned
  (`ask`), or ignored (`off`)
- push policy (`guard.push`): any `git push`, on any branch, is
  denied, questioned, or ignored. Reason: pushing is the developer's
  action.
- breaking policy (`guard.breaking`): a commit whose message carries a
  breaking marker is questioned (`ask`) or ignored (`off`)

Guard recognizes git through chains (`&&`, `;`, `|`), environment
prefixes, absolute paths, `git -C`, and `git.exe`. A compound command
is judged by its parts.

## What guard is

A guardrail against an agent following its defaults, not a security
boundary. Limit: an internal error fails open with a note on stderr.
An unconfigured repository (no `.pk.json`) is a no-op.

## Changing policy

Edit `.pk.json` and commit it.

```json
"guard": { "branches": ["main"], "mode": "block", "push": "block", "breaking": "ask" }
```

## Breaking markers

A breaking marker (`!` after the type, or a `BREAKING CHANGE:` footer)
drives the next major version. Decision: the marker is the
developer's claim; an agent never adds one on its own judgment.
Guard asks before a `git commit` whose inline message carries a
marker. `guard.breaking: "off"` disables the ask, not the rule.

Limit: only `-m` and `--message` arguments are inspected. `-F` files
and editor commits pass through. Reason: agent commits are `-m`
commits.
