# Branch protection

`pk guard` blocks git mutations on protected branches at commit time in your local environment. GitHub Repository Rulesets block at push time on the server. Running both gives defense in depth — a mutation one layer does not cover is rejected by the other, and both enforce the same conceptual rule.

This doc describes the ruleset plankit's own repos use. Import it into your project for the server-side half.

[`pk init`](pk-init.md) writes a copy into new projects at `.github/protect-main.json` (`protect-<release>.json` for a non-default release branch), ready to apply. pk never applies it: that needs an authenticated GitHub call, and pk shells out only to git.

## Which branch it guards

The ruleset names the release branch: `refs/heads/main`. It does not use GitHub's `~DEFAULT_BRANCH`. The release branch is the one `pk release` advances and `pk guard` blocks, and it need not be the repository's default: many projects make the working branch (`develop`) the default so pull requests base there. With `~DEFAULT_BRANCH` that change would silently move the protection onto `develop` and leave `main` open. Pinning by name keeps the ruleset on the branch `pk release` advances, whatever the default branch is set to. If you rename the release branch, update `include` alongside `release.branch` in `.pk.json`.

## Import through the UI

1. Get the file: `pk init` writes `.github/protect-main.json`, or download [`protect-main.json`](protect-main.json) from the `docs/` directory next to this doc.
2. On GitHub: your repo → **Settings** → **Rules** → **Rulesets**.
3. Click **New ruleset ▾** → **Import a ruleset** → upload `protect-main.json`.
4. Review the rules shown, confirm, save.

## What the ruleset does

| Rule | Effect |
|---|---|
| `deletion` | Blocks deletion of the release branch |
| `non_fast_forward` | Blocks force-push |
| `pull_request` (review count 0) | Requires a PR to merge; self-approval is allowed — solo-dev flow |
| `allowed_merge_methods: [merge, rebase]` | Excludes squash |
| `bypass_actors: Admin` | Admin bypasses rules — required for `pk release` |

## Why the defaults

**No squash.** `pk changelog` reads conventional commits (`feat:`, `fix:`, etc.) from the git log to build the release section. Squashing a PR collapses the commit history into one commit, destroying the per-commit types the changelog generator needs. Merge commits and rebase keep the originals intact.

**Admin bypass.** `pk release` merges the development branch into the release branch directly — no PR. Without the admin bypass, the ruleset would block the release merge. The bypass preserves the automation; the trade-off is that admins can push directly to the release branch at any time. Local `pk guard` is what catches admin mistakes — the ruleset on its own won't block an admin who skips `pk release` and pushes to main. The bypass is the Repository admin role only: a Maintainer running `pk release` has their push rejected by the `pull_request` rule unless their role is added to `bypass_actors`, or that rule is dropped (`deletion` and `non_fast_forward` are the two that enforce pk's model on their own).

**PR required with zero approvals.** Solo developers self-approve their own PRs. Teams should change `required_approving_review_count` to 1 or more.

## Enforcement caveats

- **Public personal repos** — rulesets are enforced.
- **Org repos** — rulesets are enforced.
- **Private personal repos** — rulesets are **not enforced** unless you're on the GitHub Team plan. GitHub shows a warning on the Rulesets page for this case. `pk guard` still protects locally.

## Customization

Adapt the imported ruleset to match your project:

- **`required_approving_review_count: 1`** (or more) — require reviewers for team projects.
- **Add a `required_status_checks` rule** — gate merges on passing CI. See [GitHub's ruleset rules reference](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) for the full schema.
- **`required_reviewers`** — name specific review teams by id.
- **`allowed_merge_methods`** — both `merge` and `rebase` are fine for plankit: each preserves the underlying PR commits that `pk changelog` reads. `merge` adds a merge commit on top (`Merge pull request #N from ...`) which is non-conventional and silently skipped by `pk changelog`. Tighten to `["rebase"]` for linear history or `["merge"]` to preserve branch structure. Never add `"squash"` — it collapses the per-commit conventional types the changelog generator needs.
- **`bypass_actors`** — add specific usernames or teams alongside Admin, or remove the bypass if no pk-release-style automation is in use.

## Apply with gh

```bash
gh api --method POST repos/<owner>/<repo>/rulesets --input .github/protect-main.json
```

Use the copy `pk init` wrote, not `docs/protect-main.json`. The `docs/` copy is a UI export and carries `source` and `source_type` fields naming the repository it came from; the rulesets API rejects those. The `.github/` copy omits them and posts as-is. A repository shaped before it had a remote may have no `.github/` copy; import the `docs/` copy through the UI, or drop the two `source` fields and post it.

## Related

- [pk guard](pk-guard.md) — local-side guard that complements this ruleset.
- [pk init](pk-init.md) — writes `.github/protect-main.json` into a new project.
- [pk release](pk-release.md) — uses the admin bypass to merge the release branch.
