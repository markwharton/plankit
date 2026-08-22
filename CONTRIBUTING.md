# Contributing

## Build

```bash
make build                    # Build for current platform -> dist/pk
make build VERSION=1.0.0      # Build with version injected
make build-all                # Cross-compile for all 5 platforms
make install                  # Install to GOPATH/bin (version: dev)
make install VERSION=1.0.0    # Install with version injected
```

The default version is `dev`. To see the installed version:

```bash
pk version    # Shows "pk dev" or "pk 1.0.0" etc.
```

## Test

```bash
make test       # Run tests with race detector
make lint       # Run go vet
make rules-lint # Lint .claude/rules: hidden chars + house style
make fmt        # Format code
```

## Documentation

The prose standard for `docs/` and `README.md` lives in `.claude/rules/plankit-development.md` (Documentation Prose) and loads into every Claude Code session in this repo. For a full audit-and-rewrite pass, paste this into a session:

```text
Audit and rewrite every file under docs/ except docs/plans/, plus
README.md, against the Documentation Prose rules (already loaded from
.claude/rules/). No fact may change in the style pass — only how it
is said.

1. Audit first: read every doc and list each violation with file:line,
   the exact sentence, the rule it breaks, and the fact it hides. Show
   me the full list before rewriting anything.
2. Verify before rewriting: confirm each behavior sentence in the Go
   source or reproduce it in a scratch repository. Where doc and code
   disagree, record it as a fact correction instead of restyling it.
3. Recurring phrases get the identical replacement everywhere. Grep the
   whole repo, including --help strings and error messages in internal/;
   where source strings share a doc's wording, list the pair with a
   recommendation — never desync them one-sidedly.
4. Re-read every touched section in full before showing the diff, not
   only the sentences you changed.
5. Commit fact corrections separately from style rewrites (both docs:),
   corrections first.
6. Rule bullets (.claude/rules/ and internal/setup/rules/) are
   audit-and-propose: show each proposed rewrite old and new, and I
   approve each bullet individually — these are behavioral instructions
   to the model, so meaning drift matters more than style. Shipped
   rules follow the pk-managed file procedure: update the embedded
   source and the local copy together and recompute pk_sha256 per
   CLAUDE.md. Run make rules-lint after.

Verify: make test and make lint pass; a final grep proves each replaced
phrase is gone from docs/; your report lists every fact correction and
every doc/source shared-wording pair.
```

The em dash check in `pk rules --lint` began as a proxy for this standard, added when nothing checked the content itself: the LLM tell was never the glyph, it was the vague clause the glyph joined. After a full rewrite pass has run under the standard, revisit the check. Removing it changes pk's shipped linter for every pk-managed project, so that is its own commit and release note, never a side effect.

## Workflow

All changes go through `develop` — never commit directly to `main`.

In Claude Code, `pk guard` enforces this automatically — it blocks git mutations on `main`. In the terminal, branch protection rules may exist but should not be the only safety net — discipline is on you.

Dependabot PRs target `develop` via `.github/dependabot.yml`. If a hotfix or emergency PR lands directly on `main`, merge main into develop before releasing:

```bash
git switch develop
git merge main
```

This ensures the changelog includes everything in the release and maintains the ancestry that `pk release` needs for fast-forward merges to main.

After merging PRs on GitHub, sync your local branch with rebase to avoid unnecessary merge commits:

```bash
git pull --rebase
```

This replays your unpushed local commits on top of the remote, keeping history linear. Only safe when your local commits haven't been pushed yet — which is exactly when you need it.

## Pull requests

When merging PRs through GitHub, choose the merge method based on the branch:

- **Rebase and merge** for most PRs (e.g., dependabot bumps) — replays commits on top of the target branch. Linear history, and each conventional commit is preserved individually for `pk changelog`.
- **Merge commit** when the PR branch has tags — rebase creates new SHAs which would orphan tags pointing at the originals.
- **Squash is disabled** — it collapses all commits into one, losing the conventional commit messages that `pk changelog` depends on. See [Squash Merge and Release Tags](docs/anti-patterns.md#squash-merge-and-release-tags).

## Release

With `release.branch` configured in `.pk.json`, the full release flow runs from Claude Code or terminal:

```bash
pk changelog --dry-run            # preview changelog and version bump
pk changelog                      # on develop: generate CHANGELOG.md and commit (no tag)
pk release --dry-run              # preview the release flow
pk release                        # read Release-Tag trailer, merge to main, tag, push main + tag, push develop
```

`pk release` merges the current branch into the release branch, validates, pushes, and switches back. See [pk release](docs/pk-release.md) for details.

**The release flow needs the Go toolchain on PATH**, not only for `make build`. Two release hooks compile from source: `pk changelog`'s `preCommit` runs `pk pin` on the bootstrap script and then `go run ./evals/footprint` (refreshes the always-on rules footprint line in the README so it lands in the release commit), and `pk release`'s `preRelease` runs `go test -race ./...`. So a release can't be cut on a machine without Go, even though the published binary itself has no runtime dependencies.

See [pk changelog](docs/pk-changelog.md) and [pk release](docs/pk-release.md) for details.

Monitor at: https://github.com/markwharton/plankit/actions

## Contributions & AI

plankit is a solo/small-team toolkit. Pull requests are welcome and reviewed by the maintainer.

- **No third-party Go dependencies.** plankit is standard-library only; a PR that adds a dependency won't be accepted.
- **Managed files get extra scrutiny.** Changes to the files `pk setup` ships downstream — `internal/setup/rules/`, `internal/setup/skills/`, and `internal/setup/template/CLAUDE.md` — are read by AI agents in every Claude Code session. They are line-ending normalized to LF via [`.gitattributes`](.gitattributes) and scanned for hidden/control characters by `make test` (the "Trojan Source" class: ANSI escapes, zero-width characters, bidi overrides). When you edit one, update its paired `pk_sha256` as described in [CLAUDE.md](CLAUDE.md) under "Updating pk-managed files".
- **Describe intent, not just the diff.** For substantive or behavior-changing PRs, say what you're changing and why in the PR body. No formal plan is required.
