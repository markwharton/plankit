---
name: plankit
description: Plan-driven development for Claude Code - the model, the modes, and the per-repo footprint
---

# plankit

plankit preserves approved Claude Code plans and protects the flow they
describe. It ships as a Claude Code plugin: these skills are the
documentation, the hooks call the pk binary, and `pk help` renders the
same pages in the terminal.

## Model

Claude plans the work. On approval, the preserve hook copies the plan,
byte for byte, into `docs/plans/` under a dated, sequenced filename.
The protect hook keeps that directory immutable, and the guard hook
blocks git mutations on protected branches, so work lands through the
planned flow.

## Per-repo footprint

A configured repository carries exactly two things:

- `.pk.json`, committed repo policy: modes, protected branches,
  changelog sections, release hooks
- `docs/plans/`, the preserved plans

No `.pk.json` means off: every hook exits immediately in an
unconfigured repository.

## Commit conventions

Work commits follow Conventional Commits: `type(scope): subject`, with
`!` or a `BREAKING CHANGE:` footer marking a breaking change. The type
table lives in `.pk.json` under `changelog.types`; `pk init` writes
the default set, listed under `pk help changelog`, and whatever the
file says is what `pk changelog` reads on release day, so the
convention is what turns daily work into the changelog.
`pk changelog --dry-run` previews how the current commits will land.

One rule stands over the table: never add a breaking marker (`!` or a
`BREAKING CHANGE:` footer) on your own judgment. Markers drive the
next major version and are the developer's claim to make; write one
only on explicit user direction. Guard asks before a marked commit is
created.

## Commands

Run `pk help` for the topic index. Each command's page is also a
/plankit: shortcut in Claude Code, so one name means the same thing in
the typeahead, the terminal, and this repository.

## Migrating from pre-plugin plankit (v0.x)

The plugin replaces everything `pk setup` used to copy into a
repository. In a repository configured by a v0.x release:

1. Remove the plankit hook entries (guard, protect, preserve, and the
   install-pk SessionStart entry) from `.claude/settings.json`. The
   plugin wires the same hooks itself, and leftover entries fire
   everything twice.
2. Delete `.claude/install-pk.sh` and the copied
   `.claude/skills/` and `.claude/rules/plankit/` files. The plugin
   ships current versions of the skills; the rules are retired.
3. Keep `.pk.json` and `docs/plans/` exactly as they are. The config
   schema is unchanged and the plans remain the immutable record.

Nothing else changes: the guard, protect, and preserve behavior is
unchanged, now arriving with the plugin instead of per-repository
copies.

## Windows without Git Bash

When Git for Windows is absent, Claude Code's shell tool is PowerShell,
and the plugin's `bin/` directory is not on that shell's PATH. Invoke
pk by its full path instead:

```
& "${CLAUDE_PLUGIN_ROOT}\bin\pk.cmd" status
```

Claude Code substitutes `${CLAUDE_PLUGIN_ROOT}` when it loads this
page, so inside a session the line above already carries the real
install location. In the terminal, `pk help` shows the placeholder
verbatim.
