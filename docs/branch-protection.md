# Branch protection

`pk guard` blocks git mutations on protected branches at commit time in your local environment. GitHub Repository Rulesets block at push time on the server. Running both gives defense in depth — a mistake dodged by one is caught by the other, and both enforce the same conceptual rule.

This doc describes the ruleset plankit's own repos use. Import it into your project for the server-side half.

## Import

1. Download [`protect-main.json`](protect-main.json) — the file sits next to this doc in the `docs/` directory.
2. On GitHub: your repo → **Settings** → **Rules** → **Rulesets**.
3. Click **New ruleset ▾** → **Import a ruleset** → upload `protect-main.json`.
4. Review the rules shown, confirm, save.

## What the ruleset does

| Rule | Effect |
|---|---|
| `deletion` | Blocks deletion of the default branch |
| `non_fast_forward` | Blocks force-push |
| `pull_request` (review count 0) | Requires a PR to merge; self-approval is allowed — solo-dev flow |
| `allowed_merge_methods: [merge, rebase]` | Excludes squash |
| `bypass_actors: Admin` | Admin bypasses rules — required for `pk release` |

## Why the defaults

**No squash.** `pk changelog` reads conventional commits (`feat:`, `fix:`, etc.) from the git log to build the release section. Squashing a PR collapses the commit history into one commit, destroying the per-commit types the changelog generator needs. Merge commits and rebase keep the originals intact.

**Admin bypass.** `pk release` merges the development branch into the release branch directly — no PR. Without the admin bypass, the ruleset would block the release merge. The bypass preserves the automation; the trade-off is that admins can push directly to the default branch at any time. Local `pk guard` is what catches admin mistakes — the ruleset on its own won't block an admin who skips `pk release` and pushes to main.

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

## Related

- [pk guard](pk-guard.md) — local-side guard that complements this ruleset.
- [pk release](pk-release.md) — uses the admin bypass to merge the release branch.
