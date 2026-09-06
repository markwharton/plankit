---
name: overview
description: Plan-driven development for Claude Code - the model, the modes, and the per-repo footprint
---

# overview

plankit preserves approved Claude Code plans, guards protected
branches, and derives releases from commit messages. It ships as a
Claude Code plugin. These skills are the documentation; `pk help`
renders the same pages in a terminal.

## Model

Claude plans the work. On approval, the preserve hook copies the plan,
byte for byte, into `docs/plans/` under a dated, sequenced filename.
The protect hook denies edits under `docs/plans/`. The guard hook
denies or questions git mutations on protected branches.

## Per-repo footprint

A configured repository carries `.pk.json`, the committed policy:
modes, protected branches, changelog sections, release hooks.
`docs/plans/` appears when the first plan is preserved. Without
`.pk.json`, every hook exits without acting.

## Commit conventions

Work commits follow Conventional Commits: `type(scope): subject`. `!`
after the type or a `BREAKING CHANGE:` footer marks a breaking
change. The type table is `changelog.types` in `.pk.json`; `pk init`
writes the default table, listed under `pk help changelog`.
`pk changelog --dry-run` previews how the current commits will land.
Never add a breaking marker on your own judgment; see `pk help guard`,
Breaking markers.

## Commands

`pk help` prints the topic index. Each command's page is also a
`/plankit:` shortcut in Claude Code.

## Migrating from pre-plugin plankit (v0.x)

The plugin replaces the files `pk setup` copied into a repository. In
a repository configured by a v0.x release:

1. Remove the plankit hook entries (guard, protect, preserve, and the
   install-pk SessionStart entry) from `.claude/settings.json`. The
   plugin wires the same hooks, and leftover entries fire twice.
2. Delete `.claude/install-pk.sh`, the copied `.claude/skills/`, and
   `.claude/rules/plankit/`.
3. Keep `.pk.json` and `docs/plans/`. The config schema is unchanged.

## Windows without Git Bash

When Git for Windows is absent, Claude Code's shell tool is
PowerShell, and the plugin's `bin/` directory is not on its PATH.
Invoke pk by its full path:

```
& "${CLAUDE_PLUGIN_ROOT}\bin\pk.cmd" status
```

Claude Code substitutes `${CLAUDE_PLUGIN_ROOT}` when it loads this
page. In a terminal, `pk help` shows the placeholder verbatim.
