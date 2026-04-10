---
description: Commit with purpose, conventional commits, commit before risk
pk_sha256: 61e4a20f38c46b7ac55965a9d9812cb4530cfa4768613f63d30fffbad7352685
---

# Git Discipline

- **Don't push your work until you're happy with it.** Locally, you have full freedom — amend, reorder, combine. Once pushed, history is shared and rewriting creates problems downstream.
- **Commit and push are separate decisions.** Commit when the work is ready; push when you're confident.
- **Never force push.** If a pushed commit needs fixing, make a new commit.
- **Rewrite unpushed commits with soft reset.** To fold an edit into an earlier commit: `git log --oneline` (note hashes) → `git reset --soft <target>~1` → `git restore --staged <files-for-later-commits>` → edit → `git add` + `git commit -C <target-hash>` → re-stage and re-commit later files with their hashes. Reflog recovers mistakes within ~30 days.
- **Commit with purpose.** Each commit is one logical change. Follow Conventional Commits to make history scannable.
- **Never include BREAKING CHANGE** in commit messages unless there is an actual breaking change.
- **Commit before risk.** Before refactoring or trying an uncertain approach — commit what works. Git is your safety net, but only if you've saved your progress.
