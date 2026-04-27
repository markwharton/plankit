# Resources

## Claude Code

- [Best practices](https://code.claude.com/docs/en/best-practices) — Effective CLAUDE.md, hooks, context management
- [How Claude remembers your project](https://code.claude.com/docs/en/memory) — CLAUDE.md, imports, `.claude/rules/`, auto memory
- [When to use plan mode](https://code.claude.com/docs/en/common-workflows#when-to-use-plan-mode) — Separating exploration from implementation

## Git

- [Git Tools — Rewriting History](https://git-scm.com/book/en/v2/Git-Tools-Rewriting-History) — Why commit and push are separate decisions

> **Don't push your work until you're happy with it.** One of the cardinal rules of Git is that, since so much work is local within your clone, you have a great deal of freedom to rewrite your history locally. However, once you push your work, it is a different story entirely, and you should consider pushed work as final unless you have good reason to change it. In short, you should avoid pushing your work until you're happy with it and ready to share it with the rest of the world.

### Mental model

Git is an elegant data structure, not a weird tool with confusing commands. Three concepts explain almost everything:

- **Commits are immutable nodes.** Each commit is a snapshot with a SHA, a parent pointer, and a message. Once created, it never changes — "rewriting history" means creating new commits and moving pointers, not editing existing ones.
- **Branches are pointers.** A branch is a label that points to the latest commit in a line of work. Renaming, deleting, or creating a branch doesn't touch the commits — it only changes where the label points. That's why branch operations are instant and safe.
- **Merges are ancestry links.** A merge commit has two parents, connecting two lines of history. It's how git knows one branch descends from another. Without the merge commit, git can't see the relationship — even if the content is identical (which is why cherry-pick doesn't establish ancestry).

These three concepts explain the techniques below:

- **Soft replay.** `git reset --soft <target>` moves the branch pointer back but leaves all changes staged. You re-deal the cards — same content, different commits. Only works while unpushed, because new commits get new SHAs.
- **Merge for branch syncing.** Use `git merge` to bring commits from another branch into yours. Merge establishes ancestry — git knows the branches are connected, which is what fast-forward merges (like `pk release`) require. Avoid cherry-pick for this purpose: it copies content but doesn't connect the branches, so fast-forward merges will fail later.
- **Merge commits as invisible infrastructure.** A merge commit that isn't a conventional commit won't appear in the changelog — but it's load-bearing in the git graph. It establishes the ancestry that makes `pk release` fast-forward work, without cluttering user-facing output.

### Release workflows

See [pk release — Workflows](pk-release.md#workflows) for the reference table. The examples below show the actual command sequences for each flow.

**Merge flow** (protected `main`, work on `develop`, `release.branch: "main"` in `.pk.json`):

```bash
pk changelog                          # on develop: generate changelog and commit (no tag yet)
pk release                            # tag, merge to main, validate, push, switch back, push develop
```

**Trunk flow** (single branch, no `release.branch` configured):

```bash
pk changelog                          # on the working branch: generate changelog and commit (no tag yet)
pk release                            # tag HEAD, push current branch + tag
```

`pk changelog` adds a `Release-Tag:` trailer to the commit message body. `pk release` reads that trailer to know which version to tag, then creates the tag just before pushing. If something goes wrong between `pk changelog` and `pk release`, run `pk changelog --undo` to cleanly unwind the release commit (refuses if HEAD has already been pushed).

## GitHub CLI

- [gh reference](https://cli.github.com/manual/) — Full command reference

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

## Well-Architected Framework

AWS, Azure, and Google Cloud each publish a Well-Architected Framework — a structured set of best practices organized around the same core pillars. The names differ slightly but the concepts align. The patterns transfer across providers.

| Pillar | AWS | Azure | Google Cloud |
|--------|-----|-------|-------------|
| Operations | Operational Excellence | Operational Excellence | Operational Excellence |
| Security | Security | Security | Security, Privacy, and Compliance |
| Reliability | Reliability | Reliability | Reliability |
| Performance | Performance Efficiency | Performance Efficiency | Performance Optimization |
| Cost | Cost Optimization | Cost Optimization | Cost Optimization |
| Sustainability | Sustainability | — | Sustainability |

### AWS

- [AWS Well-Architected Framework](https://docs.aws.amazon.com/wellarchitected/latest/framework/welcome.html)
- Pillars: [Operational Excellence](https://docs.aws.amazon.com/wellarchitected/latest/operational-excellence-pillar/welcome.html) | [Security](https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/welcome.html) | [Reliability](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/welcome.html) | [Performance Efficiency](https://docs.aws.amazon.com/wellarchitected/latest/performance-efficiency-pillar/welcome.html) | [Cost Optimization](https://docs.aws.amazon.com/wellarchitected/latest/cost-optimization-pillar/welcome.html) | [Sustainability](https://docs.aws.amazon.com/wellarchitected/latest/sustainability-pillar/sustainability-pillar.html)

### Azure

- [Azure Well-Architected Framework](https://learn.microsoft.com/en-us/azure/well-architected/)
- Pillars: [Reliability](https://learn.microsoft.com/en-us/azure/well-architected/reliability/) | [Security](https://learn.microsoft.com/en-us/azure/well-architected/security/) | [Cost Optimization](https://learn.microsoft.com/en-us/azure/well-architected/cost-optimization/) | [Operational Excellence](https://learn.microsoft.com/en-us/azure/well-architected/operational-excellence/) | [Performance Efficiency](https://learn.microsoft.com/en-us/azure/well-architected/performance-efficiency/)

### Google Cloud

- [Google Cloud Well-Architected Framework](https://docs.cloud.google.com/architecture/framework)
- Pillars: [Operational Excellence](https://docs.cloud.google.com/architecture/framework/operational-excellence) | [Security, Privacy, and Compliance](https://docs.cloud.google.com/architecture/framework/security) | [Reliability](https://docs.cloud.google.com/architecture/framework/reliability) | [Cost Optimization](https://docs.cloud.google.com/architecture/framework/cost-optimization) | [Performance Optimization](https://docs.cloud.google.com/architecture/framework/performance-optimization) | [Sustainability](https://docs.cloud.google.com/architecture/framework/sustainability)
