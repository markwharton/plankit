---
description: Commit with purpose, conventional commits, commit before risk
pk_sha256: da6346b8787c3d8abb50ee50265a02b16f3f9c2cb7b96d6163c77625b67e29f0
---

# Git Discipline

- **Don't push your work until you're happy with it.** Locally, you have full freedom: amend, reorder, combine. Once pushed, history is shared and rewriting creates problems downstream.
- **Commit and push are separate decisions.** Commit when the work is ready; push when you're confident.
- **Never force push.** If a pushed commit needs fixing, make a new commit.
- **Rewrite unpushed commits with soft reset.** To fold an edit into an earlier commit: `git log --oneline` (note hashes); verify the target is the commit you intend to modify, not an unrelated commit that landed after it; `git reset --soft <target>~1`; `git restore --staged <files-for-later-commits>`; edit; `git add` + `git commit -C <target-hash>`; then re-stage and re-commit later files with their hashes. Reflog recovers mistakes within ~30 days.
- **Don't improvise git history rewrites.** The soft-reset procedure covers the common case. When it applies, follow it. Don't reach for interactive rebase, stash-based workarounds, or ad hoc alternatives.
- **Commit with purpose.** Each commit is one logical change. Follow Conventional Commits to make history scannable.
- **Configure automation that produces commits to follow the convention.** Dependabot, release bots, and any tool that opens PRs or pushes commits should set a conventional `commit-message: prefix:` (e.g., `chore(deps)`) so their work flows into `pk changelog` rather than getting silently skipped at release time.
- **Match message weight to change weight.** Substantive features (user-facing behavior, design decisions worth preserving) get a multi-paragraph body explaining why and shape. Focused small changes get one-line messages. Don't inherit the recent commits' style; match the message to this commit's content.
- **Never include BREAKING CHANGE** in commit messages unless there is an actual breaking change.
- **Commit before risk.** Before refactoring or trying an uncertain approach, commit what works. Git is your safety net, but only if you've saved your progress.
