---
description: Three-layer architecture (pk commands, hooks, skills) and hook behavior
---

# Plankit Tooling

## Three Layers

- **pk commands:** Standalone CLI tools that power everything below. You don't run these directly; hooks and skills handle that.
- **Hooks:** Wire pk commands into Claude Code events. They run automatically and you receive their output (block decisions, ask prompts, notifications). Described below.
- **Skills:** User-invoked workflows (`/init`, `/preserve`, `/ship`). Each has its own instructions. Execute them only when the user asks.

## Hook Behavior

- **`pk guard` blocks git mutations on protected branches.** If the project uses ask mode, you will be prompted instead; respect the user's decision either way. When blocked, switch to the development branch.
- **`pk protect` blocks edits to pk-managed files.** The block reason tells you why. Adjust your approach; don't try to work around it.
- **`pk preserve` runs after exiting plan mode.** Behavior depends on project configuration; it may preserve automatically or notify that a plan is ready. If the user types `/preserve`, dispatch the skill as your next action. Never queue it behind implementation work. `/preserve` is an explicit request, not a go-signal for something else.

## Session Bootstrap

- **pk installs itself in cloud sandboxes.** The SessionStart hook downloads pk if it's not already available. If pk is already on PATH, the hook exits immediately. No action needed.

## Committing pk Setup Changes

- **Commit `pk setup` updates on their own.** When `pk setup` creates or updates managed files (skills, rules, CLAUDE.md, install-pk.sh), commit those changes separately rather than folding them into feature work. Keeps history scannable and makes pk-upgrade churn distinguishable from project changes. Suggested message: `chore: update pk-managed files for v<VERSION>` where `<VERSION>` is the installed pk version.

## Flag Conventions

- **`--push` means "publish this, fully."** When a pk command supports `--push`, it publishes whatever that command produced, including any refs needed to make it reachable on origin. For a tagging command, that includes the branch the tag sits on. `--push` always means "push what I just did, complete," never a narrower partial push. The default behavior (no `--push`) is local-only, consistent with the git-discipline rule that commit and push are separate decisions.
- **`--at <ref>` narrows `--push` to the explicit target.** When a pk command accepts `--at <ref>` to operate on a specific ref rather than HEAD, `--push` publishes only the thing produced at that ref, not HEAD or its branch. The user picked the ref; pk doesn't assume which branch goes with it.
