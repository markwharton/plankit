# Resources

## Claude Code

- [Best practices](https://code.claude.com/docs/en/best-practices) — Effective CLAUDE.md, hooks, context management
- [How Claude remembers your project](https://code.claude.com/docs/en/memory) — CLAUDE.md, imports, `.claude/rules/`, auto memory
- [When to use plan mode](https://code.claude.com/docs/en/common-workflows#when-to-use-plan-mode) — Separating exploration from implementation

## Git

- [Git Tools — Rewriting History](https://git-scm.com/book/en/v2/Git-Tools-Rewriting-History) — Why commit and push are separate decisions

> **Don't push your work until you're happy with it.** One of the cardinal rules of Git is that, since so much work is local within your clone, you have a great deal of freedom to rewrite your history locally. However, once you push your work, it is a different story entirely, and you should consider pushed work as final unless you have good reason to change it. In short, you should avoid pushing your work until you're happy with it and ready to share it with the rest of the world.

## GitHub CLI

- [gh reference](https://cli.github.com/manual/) — Full command reference

### Release flow (independent developer)

All changes go through `dev` — `main` is protected by `pk guard`. With
`release.branch` configured in `.pk.json`, `pk release` handles the merge
automatically:

```bash
pk changelog                          # on dev: generate changelog, commit, and tag version
pk release                            # merge to main, validate, push, switch back, push dev
```

Without `release.branch` configured, the manual git flow:

```bash
pk changelog                          # on dev: generate changelog, commit, and tag version
git switch main && git merge dev      # merge to main
pk release                            # on main: validate and push
git switch dev                        # switch back
git push origin dev                   # sync dev with origin
```


### Useful commands

```bash
# Workflow runs
gh run list --workflow=<file> --limit=1     # check status
gh run cancel <run-id>                      # kill a stuck run
gh run rerun <run-id>                       # retry it
gh run watch <run-id>                       # live follow

# PRs and issues
gh pr create --title "..." --body "..."     # create PR from current branch
gh pr list                                  # list open PRs
gh api repos/owner/repo/pulls/123/comments  # read PR comments
```
