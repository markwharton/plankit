# Branch protection

`pk guard` blocks a git mutation on the local machine, before the command runs, inside a Claude Code session. A GitHub ruleset blocks it on the server, at push time, from any client. The two cover different surfaces; plankit repositories run both.

`docs/protect-main.json` is the ruleset plankit's own repositories use. [`pk init`](pk-init.md) writes a copy into a new project as `.github/protect-main.json` (`protect-<release>.json` for another release branch). pk never applies it: that is a GitHub API call, and pk shells out only to git.

## What it does

| Rule | Effect |
|---|---|
| `deletion` | the release branch cannot be deleted |
| `non_fast_forward` | no force-push |
| `pull_request`, review count 0 | a change reaches the branch through a PR; the author may approve it |
| `allowed_merge_methods: [merge, rebase]` | no squash; see [Squash merge](design.md#squash-merge-and-release-tags) |
| `bypass_actors: Repository admin` | `pk release` pushes the fast-forward without a PR |

## Decisions

- **The branch is named, `refs/heads/main`, not `~DEFAULT_BRANCH`.** A project may make the working branch its default so pull requests base there; `~DEFAULT_BRANCH` would then move the rules to `develop` and leave `main` open. Rename the release branch: update `include` and `release.branch` together.
- **Admin bypass.** `pk release` merges without a PR, so the admin role bypasses the rules; an admin can also push to the release branch by hand, which `pk guard` catches locally. A Maintainer running `pk release` is rejected by the `pull_request` rule until the role is added to `bypass_actors` or that rule is dropped; `deletion` and `non_fast_forward` enforce pk's model on their own.
- **Zero approvals.** A solo developer approves their own PR. Reopen with a second maintainer: set `required_approving_review_count` to 1.
- **No `required_status_checks`.** Add it when the repository has CI to gate on.

## Apply

Through the UI: Settings, Rules, Rulesets, New ruleset, Import a ruleset, upload the file. Through the CLI:

```bash
gh api --method POST repos/<owner>/<repo>/rulesets --input .github/protect-main.json
```

Post the `.github/` copy. The `docs/` copy is a UI export and carries `source` and `source_type`, which the API rejects; import it through the UI or drop the two fields.

## Limits

- A private personal repository does not enforce rulesets on the free plan; GitHub says so on the Rulesets page. `pk guard` still applies.
- A merge commit (`Merge pull request #N …`) is not a conventional commit and is skipped by `pk changelog`; the PR's own commits are read.
