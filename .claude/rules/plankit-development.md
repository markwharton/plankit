---
description: Rules for building plankit itself — CHANGELOG format, command evolution, tip messages. Not shipped to plankit-using projects.
---

# Plankit Development

These rules apply when working *on* plankit — authoring the CLI, writing runtime messages, maintaining the CHANGELOG. They are maintainer-side and live in plankit's `.claude/rules/` only. They are NOT embedded in `pk setup` and do NOT ship to other projects.

## CHANGELOG Format

- **Plain text, one link per version.** Entries are plain `- summary (abc1234)` — no clickable commit SHAs. Each version heading (`## [v0.10.1] - 2026-04-17`) is already a clickable link to a compare URL showing every commit in that release with full context. Don't link individual commits in any form — inline `[sha](url)` or reference-style. Don't pull in CHANGELOG generators (commit-and-tag-version, git-cliff). plankit is "small tools, carefully made" — plain text by design, not by oversight.

## Evolving pk Commands

- **Grep existing flag/mode enumerations before declaring done.** Adding or renaming a `pk` command option is a concept change, not just a command-doc update. Before finishing, grep the repo for existing option lists (`README.md`, `docs/getting-started.md`, other command docs) and add the new option where others are enumerated. The mechanical "code → tests → command doc" loop stops at the command doc; the grep catches the higher-level docs that the CLAUDE.md documentation rule requires updating for concept changes.

## Tip Messages

- **Show the git equivalent when pk is a thin wrapper.** When stderr output suggests a pk command as a next step, follow with the git equivalent on the next line when pk is a thin wrap over 1–2 git commands (tag creation, push, add/commit). Format: the pk command on one line, `or: <git commands>` on the next. Skip the git line when pk adds substantial logic (pre-flight checks, hooks, multi-step flows, commit scanning) that would be lost in a direct translation. The pattern educates, builds trust, and gives power users an escape hatch.

## Repo Checks

- **Pick the repo check by command profile, not by habit.** Two patterns are in use. `git.IsRepo(stat, dir)` walks parent directories for `.git` with no subprocess — right for commands where the check is a pre-condition and may be the only git-adjacent op (`pk setup`, `pk status`). `cfg.GitExec("", "rev-parse", "--is-inside-work-tree")` shells to git and is authoritative on worktree/GIT_DIR/submodule edge cases — right for commands that already call GitExec extensively (`pk changelog`, `pk preserve`, `pk release`), where the check matches the rest of the command's git surface. The split is acceptable when each choice fits its command; don't standardize for its own sake.
