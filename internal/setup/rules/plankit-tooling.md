---
description: Three-layer architecture (pk commands, hooks, skills) and hook behavior
kind: conduct
---

# Plankit Tooling

## Three Layers

- **pk commands:** Standalone CLI tools that power everything below. You don't run these directly; hooks and skills handle that.
- **Hooks:** Wire pk commands into Claude Code events. They run automatically and you receive their output (block decisions, ask prompts, notifications). Described below.
- **Skills:** User-invoked workflows (`/conventions`, `/preserve`, `/ship`). Each has its own instructions. Execute them only when the user asks. When suggesting `/ship` as a next step, write it bare: never append a version (`/ship`, not `/ship v0.4.0`). `/ship` takes no version; the version is computed by `pk changelog` and revealed by its dry-run.

## Hook Behavior

- **`pk guard` blocks git mutations on protected branches, and can guard `git push` on any branch.** If the project uses ask mode, you will be prompted instead; respect the user's decision either way. When a protected-branch mutation is blocked, switch to the development branch. When a push is blocked or prompted by push-guard, treat pushing as the developer's call: do not work around it. The developer pushes manually, or uses `pk preserve` / `pk release`, which publish through pk and pass the guard.
- **`pk protect` blocks edits to `docs/plans/`.** Preserved plans are immutable historical records. The block reason tells you why. Adjust your approach; don't try to work around it.
- **`pk preserve` runs after exiting plan mode.** Behavior depends on project configuration; it may preserve automatically or notify that a plan is ready. When it runs automatically, surface the outcome to the user, including any commits created or pushes attempted. If the user types `/preserve`, dispatch the skill as your next action. Never queue it behind implementation work. `/preserve` is an explicit request, not a go-signal for something else.

## Session Bootstrap

- **pk installs itself in cloud sandboxes.** The SessionStart hook downloads pk if it's not already available. If pk is already on PATH, the hook exits immediately. No action needed.

## Flag Conventions

- **`--push` exists only on `pk setup --baseline` and `pk preserve`.** On those commands it means "publish what I just produced, fully": pushing any refs needed to make it reachable on origin (for a tag, the branch it sits on), never a partial push. Without `--push` they stay local-only, because commit and push are separate decisions (git-discipline). No other pk command takes `--push`.
- **`--at <ref>` narrows `--push` to that ref.** When a command accepts `--at <ref>`, `--push` publishes only what was produced at that ref, not HEAD or its branch. The user picked the ref; pk doesn't assume the branch.
- **`pk release` has no `--push`; it publishes atomically.** It fast-forward merges into the release branch, tags, and pushes in one step; the only flag is `--dry-run` (preview). Passing `--push` errors.
- **Don't infer a pk flag from another command; check `pk <cmd> --help`.** Each command's `--help` is the authoritative, always-current flag list and can't drift from the binary the way a copied list can. Flag conventions aren't universal: a flag on one command, like `--push` on `pk setup --baseline`, may not exist on another, so when unsure run `pk <cmd> --help` rather than assuming.
