---
description: Three-layer architecture (pk commands, hooks, skills) and hook behavior
pk_sha256: 2b03aa6a5e2aee9daaf8a244462d8f4d87ec4bfe0355d10e6e6c177e46443a7c
---

# Plankit Tooling

## Three Layers

- **pk commands** — standalone CLI tools that power everything below. You don't run these directly — hooks and skills handle that.
- **Hooks** — wire pk commands into Claude Code events. They run automatically and you receive their output (block decisions, ask prompts, notifications). Described below.
- **Skills** — user-invoked workflows (`/changelog`, `/release`, `/preserve`). Each has its own instructions. Execute them only when the user asks.

## Hook Behavior

- **`pk guard` blocks git mutations on protected branches.** If the project uses ask mode, you will be prompted instead — respect the user's decision either way. When blocked, switch to the development branch.
- **`pk protect` blocks edits to pk-managed files.** The block reason tells you why — adjust your approach, don't try to work around it.
- **`pk preserve` runs after exiting plan mode.** Behavior depends on project configuration — it may preserve automatically or notify that a plan is ready.

## Session Bootstrap

- **pk installs itself in cloud sandboxes.** The SessionStart hook downloads pk if it's not already available. If pk is already on PATH, the hook exits immediately. No action needed.

## Flag Conventions

- **`--push` means "publish this, fully."** When a pk command supports `--push`, it publishes whatever that command produced — and any refs needed to make it reachable on origin. For a tagging command, that includes the branch the tag sits on. `--push` always means "push what I just did, complete," never a narrower partial push. The default behavior (no `--push`) is local-only, consistent with the git-discipline rule that commit and push are separate decisions.
- **`--at <ref>` narrows `--push` to the explicit target.** When a pk command accepts `--at <ref>` to operate on a specific ref rather than HEAD, `--push` publishes only the thing produced at that ref — not HEAD or its branch. The user picked the ref; pk doesn't assume which branch goes with it.

## CHANGELOG Format

- **Plain text, one link per version.** Entries are plain `- summary (abc1234)` — no clickable commit SHAs. Each version heading (`## [v0.10.1] - 2026-04-17`) is already a clickable link to a compare URL showing every commit in that release with full context. Don't link individual commits in any form — inline `[sha](url)` or reference-style. Don't pull in CHANGELOG generators (commit-and-tag-version, git-cliff). plankit is "small tools, carefully made" — plain text by design, not by oversight.

## Evolving pk Commands

- **Grep existing flag/mode enumerations before declaring done.** Adding or renaming a `pk` command option is a concept change, not just a command-doc update. Before finishing, grep the repo for existing option lists (`README.md`, `docs/getting-started.md`, other command docs) and add the new option where others are enumerated. The mechanical "code → tests → command doc" loop stops at the command doc; the grep catches the higher-level docs that the CLAUDE.md documentation rule requires updating for concept changes.

## Tip Messages

- **Show the git equivalent when pk is a thin wrapper.** When stderr output suggests a pk command as a next step, follow with the git equivalent on the next line when pk is a thin wrap over 1–2 git commands (tag creation, push, add/commit). Format: the pk command on one line, `or: <git commands>` on the next. Skip the git line when pk adds substantial logic (pre-flight checks, hooks, multi-step flows, commit scanning) that would be lost in a direct translation. The pattern educates, builds trust, and gives power users an escape hatch.
