---
name: guard
description: How pk guard decides what an agent may do to git, and how to tune it
---

# pk guard

`pk guard` is a PreToolUse hook on shell commands. Before the agent
runs a git mutation (`commit`, `merge`, `push`, `rebase`, `reset`), guard
reads `.pk.json` and answers with a permission decision. Three
policies apply; the strongest decision wins, and deny beats ask.

- branch policy (`guard.mode`): a mutation while a protected branch
  (`guard.branches`) is checked out is denied (`block`), questioned
  (`ask`), or ignored (`off`)
- push policy (`guard.push`): any `git push`, on any branch, is
  denied, questioned, or ignored, because pushing is the developer's
  action
- breaking policy (`guard.breaking`): a commit whose message carries a
  breaking marker is questioned (`ask`) or ignored (`off`)

The hook recognizes git through chains (`&&`, `;`, `|`), environment
prefixes, absolute paths, `git -C`, and `git.exe`. A compound command
is judged by its parts.

## What guard is

A guardrail against an agent following its defaults, not a security
boundary. An internal error fails open with a note on stderr.
An unconfigured repository (no `.pk.json`) is a no-op.

## Changing policy

Edit `.pk.json` and commit it. The keys, their values, and their
defaults are under Settings below.

```json
"guard": {
  "branches": ["main"],
  "breaking": "ask",
  "mode": "block",
  "push": "block"
}
```

## Breaking markers

A breaking marker (`!` after the type, or a `BREAKING CHANGE:` footer)
drives the next major version. The marker is the developer's claim,
and an agent never adds one on its own judgment.
The hook asks before a `git commit` whose inline message carries a
marker. `guard.breaking: "off"` disables the ask, not the rule.

Only `-m` and `--message` arguments are inspected; `-F` files and
editor commits pass through. That covers the case the rule exists
for, since agent commits are `-m` commits.

## Settings

<!-- generated: settings -->
The `guard` section of `.pk.json`:

```
"guard": {
  "branches": ["<branch>", ...],
  "breaking": "ask" | "off",
  "mode": "block" | "ask" | "off",
  "push": "block" | "ask" | "off"
}
```

- `guard.branches`: a list of branch names; default the release branch. The protected branches: a git mutation while one is checked out is judged by `guard.mode`.
- `guard.breaking`: `ask` or `off`; default `ask`. A commit whose message carries a breaking marker (`!` or `BREAKING CHANGE`) is questioned, or not.
- `guard.mode`: `block`, `ask`, or `off`; default `block`. A git mutation while a protected branch is checked out is denied, questioned, or ignored.
- `guard.push`: `block`, `ask`, or `off`; default `block`. Any `git push`, on any branch, is denied, questioned, or ignored.

An unknown key or a value outside these fails the whole file when it loads, with a message naming the key: `pk` commands exit 2, and each hook reports the message and takes no action until it is fixed. An absent key means its default. `pk status` reads the file back and reports the first problem.
<!-- /generated: settings -->
