---
description: Rules for building plankit itself. CHANGELOG format, command evolution, tip messages. Not shipped to plankit-using projects.
---

# Plankit Development

These rules apply when working *on* plankit: authoring the CLI, writing runtime messages, maintaining the CHANGELOG. They are maintainer-side and live in plankit's `.claude/rules/` only. They are NOT embedded in `pk setup` and do NOT ship to other projects.

## CHANGELOG Format

- **Plain text, one link per version.** Entries are plain `- summary (abc1234)` with no clickable commit SHAs. Each version heading (`## [v0.10.1] - 2026-04-17`) is already a clickable link to a compare URL showing every commit in that release with full context. Don't link individual commits in any form, whether inline `[sha](url)` or reference-style. Don't pull in CHANGELOG generators (commit-and-tag-version, git-cliff). plankit is "small tools, carefully made": plain text by design, not by oversight.

## Evolving pk Commands

- **Grep existing flag/mode enumerations before declaring done.** Adding or renaming a `pk` command option is a concept change, not just a command-doc update. Before finishing, grep the repo for existing option lists (`README.md`, other command docs) and add the new option where others are enumerated. The mechanical "code → tests → command doc → reference docs" loop stops at the reference docs; the grep catches the higher-level docs that the CLAUDE.md documentation rule requires updating for concept changes. Reference docs (`docs/pk-json.md`, `docs/error-reference.md`, `docs/environment-variables.md`) centralize cross-command information: config keys, error messages, and environment variables respectively. New docs in `docs/` go in the right README section: guides (adoption, methodology, versioning) under Documentation, lookup references (config schema, error messages, env vars) under Reference.

## Tip Messages

- **Show the git equivalent when pk is a thin wrapper.** When stderr output suggests a pk command as a next step, follow with the git equivalent on the next line when pk is a thin wrap over 1–2 git commands (tag creation, push, add/commit). Format: the pk command on one line, `or: <git commands>` on the next. Skip the git line when pk adds substantial logic (pre-flight checks, hooks, multi-step flows, commit scanning) that would be lost in a direct translation. The pattern educates, builds trust, and gives power users an escape hatch.

## Skill Authoring

- **Keep skill questions conversational.** When a skill asks the user for input, list questions as plain bullets under a short heading. Move interpretation context (which config key maps to which answer, default values, command references) to the skill's Rules section. Dense instructional text around questions causes the model to dump it all as a wall of text instead of walking through questions naturally.

## Repo Checks

- **All commands resolve to the git root via `git.RepoRoot`.** Directory resolution is consistent: `git.RepoRoot(stat, dir)` walks parent directories for `.git` (no subprocess) and returns the root path. This is the standard for determining where to operate. Commands that also need authoritative work-tree verification (worktree/GIT_DIR/submodule edge cases) additionally call `cfg.GitExec("", "rev-parse", "--is-inside-work-tree")` — but directory resolution itself always uses the stat-based walk.
