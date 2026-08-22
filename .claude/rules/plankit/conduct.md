---
description: Hook responses, skill dispatch, pk flag lookup, and git/session conduct
kind: conduct
pk_sha256: 2d2be96d26a9ed5b039c63e678a60330753191fdc75e0ef43a5440e459b3b228
---

# Plankit Conduct

## Tooling

- **pk has three layers: commands, hooks, skills.** Hooks wire pk commands into Claude Code events and run automatically; skills (`/pk-configure`, `/preserve`, `/ship`) are user-invoked workflows. Execute a skill only when the user asks.
- **`pk guard` blocks git mutations on protected branches, and can guard `git push` on any branch.** In ask mode you are prompted instead; respect the user's decision either way. When a protected-branch mutation is blocked, switch to the development branch. When a push is blocked or prompted, pushing is the developer's call: never work around it. The developer pushes manually, or uses `pk preserve` / `pk release`, which publish through pk and pass the guard.
- **`pk protect` blocks edits to `docs/plans/`.** Preserved plans are immutable historical records. Adjust your approach; don't try to work around the block.
- **`pk preserve` runs after exiting plan mode.** It may preserve automatically or notify that a plan is ready. When it runs automatically, surface the outcome to the user, including any commits created or pushes attempted. If the user types `/preserve`, dispatch the skill as your next action; it is an explicit request, never a go-signal for something else.
- **Suggest `/ship` bare, never with a version.** `/ship` takes no version; the version is computed by `pk changelog` and revealed by its dry-run.
- **pk installs itself in cloud sandboxes.** The SessionStart hook downloads pk if it's not already on PATH; no action needed.

## Flags

- **`--push` exists only on `pk init`, `pk setup --baseline`, and `pk preserve`.** On those commands it means "publish what I just produced, fully"; without it they stay local-only, because commit and push are separate decisions. No other pk command takes `--push`.
- **`--at <ref>` narrows `--push` to that ref.** `--push` then publishes only what was produced at that ref, not HEAD or its branch.
- **`pk release` has no `--push`; its only flag is `--dry-run`.** Release publishes as one atomic step, so there is nothing separate to push.
- **Don't infer a pk flag from another command; run `pk <cmd> --help`.** Each command's `--help` is the authoritative flag list.

## Git and Session Conduct

- **Git decisions are the developer's; carry them out, don't originate them.** A request to commit is never a request to push, and approval to push once is not approval for the next push.
- **On unexpected git state, stop and defer to the developer.** Diverged branches, "behind remote", anything you didn't anticipate: don't reflexively `git pull`, `git pull --rebase`, `git merge`, or `git reset`. Diagnose with `git log --oneline --graph HEAD origin/<branch>`, report what you see, and wait for instructions.
- **Surface state-changing failures immediately.** When an operation partially succeeds (commit created but push rejected, file written but validation failed), tell the user what happened, what state changed, and the remediation step. Never silently continue.
- **Clarifications are information, not instructions.** When the user corrects your interpretation or brings you up to date on state, acknowledge and wait for the explicit next step. Never execute an option from your prior analysis just because the clarified state now matches it, especially destructive operations.
- **Test at the start of each session and report the status.** Test before and after changes so failures are attributable.
