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

## Commands

Run `pk help` for the topic index. Each command's page is also a
/plankit: shortcut in Claude Code, so one name means the same thing in
the typeahead, the terminal, and this repository.

## Migrating from plankit v1

The plugin replaces everything `pk setup` used to copy into a
repository. In a repository configured by v1:

1. Remove the plankit hook entries (guard, protect, preserve, and the
   install-pk SessionStart entry) from `.claude/settings.json`. The
   plugin wires the same hooks itself, and leftover entries fire
   everything twice.
2. Delete `.claude/install-pk.sh` and the copied
   `.claude/skills/` and `.claude/rules/plankit/` files. The plugin
   ships current versions of the skills; the rules are retired.
3. Keep `.pk.json` and `docs/plans/` exactly as they are. The config
   schema is unchanged and the plans remain the immutable record.

Nothing else changes: the guard, protect, and preserve behavior is the
v1 behavior, now arriving with the plugin instead of per-repository
copies.
