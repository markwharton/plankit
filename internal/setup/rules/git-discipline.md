---
description: Verify before rewriting, commit with purpose, conventional commits, commit before risk
---

# Git Discipline

- **Don't push your work until you're happy with it.** Unpushed commits are yours to amend, reorder, and combine. Once pushed, history is shared and rewriting creates problems downstream.
- **Verify push state before any history rewrite.** Before `--amend`, soft reset, or any operation that rewrites a commit, run `git log --oneline @{push}..HEAD` (or compare against `origin/<branch>`) to confirm the target commit has not been pushed. If the command errors (no upstream) or the target commit is not in the output, it has been pushed — make a new commit instead. Never assume a recent commit is local; always check.
- **Commit, push, and release are separate decisions.** Commit when the work is ready; push when you're confident; release when it's time to publish. Release is not an ordinary push: `pk release` tags, merges, and pushes as one atomic action, because the tag must travel with the merge that anchors it, so never push by hand to publish a release.
- **Never force push.** If a pushed commit needs fixing, make a new commit.
- **Rewrite unpushed commits with soft reset.** To fold an edit into an earlier commit: `git log --oneline` (note hashes); confirm the target commit appears in `git log --oneline @{push}..HEAD` (unpushed); verify the target is the commit you intend to modify, not an unrelated commit that landed after it; `git reset --soft <target>~1`; `git restore --staged <files-for-later-commits>`; edit; `git add` + `git commit -C <target-hash>`; then re-stage and re-commit later files with their hashes. Reflog recovers mistakes within ~30 days.
- **Don't improvise git history rewrites.** The soft-reset procedure covers the common case. When it applies, follow it. Don't reach for interactive rebase, stash-based workarounds, or ad hoc alternatives.
- **Don't rewrite history between `pk changelog` and `pk release`.** The two commands are a coupled flow: changelog captures commit SHAs, release publishes them. Rewriting history mid-flow produces stale references.
- **Commit with purpose.** Each commit is one logical change. Follow Conventional Commits to make history scannable.
- **Configure automation that produces commits to follow the convention.** Dependabot, release bots, and any tool that opens PRs or pushes commits should set a conventional `commit-message: prefix:` (e.g., `chore(deps)`) so their work flows into `pk changelog` rather than getting silently skipped at release time.
- **Match message weight to change weight.** Substantive features (user-facing behavior, design decisions worth preserving) get a multi-paragraph body explaining why and shape. Focused small changes get one-line messages. Don't inherit the recent commits' style; match the message to this commit's content.
- **Never include BREAKING CHANGE** in commit messages unless there is an actual breaking change.
- **Commit before risk.** Before refactoring or trying an uncertain approach, commit what works. Git is your safety net, but only if you've saved your progress.
