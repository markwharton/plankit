# Contributing

plankit is a standard-library Go module; `make build` writes `dist/pk`. Pull requests are reviewed by the maintainer.

## Build and test

```bash
make build                    # dist/pk, version dev
make build VERSION=1.0.0      # version injected by ldflags
make build-all                # darwin and linux on amd64 and arm64, windows on amd64
make install                  # GOPATH/bin
make test                     # go test -race ./...; scans the embedded managed files for hidden characters
make lint                     # go vet and gofmt drift
make rules-lint               # pk rules --lint --strict on .claude/rules/
make vuln                     # govulncheck
make cover                    # per-function coverage of .go files changed since the latest tag
```

`make cover` runs before `/ship`: Codecov's patch check (84%) reports on the release commit, after it is public.

## Documentation

The local rule `.claude/rules/docs.md` governs sentences; the house convention [docs/design.md](docs/design.md) governs which files exist; `.claude/rules/plankit-development.md`, Documentation Prose, adds the plankit-specific points.

Every `docs/pk-<command>.md` has these sections, in this order, each omitted when empty and never renamed: the one-line summary under the title, `## Usage`, `## How it works`, `## Flags`, `## Configuration`, `## Hook protocol`, `## Environment`, `## Decisions`, `## Limits`. Hook protocol states input, output and exit code; Environment lists the variables read, each as a link to `docs/environment-variables.md`. A change that adds a config key, an error message or an environment variable updates `docs/pk-json.md`, `docs/error-reference.md` or `docs/environment-variables.md` in the same commit; a change that adds a flag or mode updates every list that enumerates them (`grep` the repo for a sibling flag).

For a full audit-and-rewrite pass, paste this into a session:

```text
Audit and rewrite every file under docs/ except docs/plans/, plus
README.md, against .claude/rules/plankit/docs.md, following its
"Rewriting an existing document" steps, plus the Documentation Prose
points in .claude/rules/plankit-development.md (both already loaded).
No fact may change in the style pass; sentences that are not a fact,
a command, a decision, or a limit are deleted, not restyled.

1. Audit first: read every doc and list each violation with file:line,
   the exact sentence, the rule it breaks, and the fact it hides. Show
   me the full list before rewriting anything.
2. Verify before rewriting: confirm each behavior sentence in the Go
   source or reproduce it in a scratch repository. Where doc and code
   disagree, record it as a fact correction instead of restyling it.
3. Recurring phrases get the identical replacement everywhere. Grep the
   whole repo, including --help strings and error messages in internal/;
   where source strings share a doc's wording, list the pair with a
   recommendation; never desync them one-sidedly.
4. Re-read every touched section in full before showing the diff, not
   only the sentences you changed.
5. Commit fact corrections separately from style rewrites (both docs:),
   corrections first.
6. Rule bullets (.claude/rules/ and internal/setup/rules/) are
   audit-and-propose: show each proposed rewrite old and new, and I
   approve each bullet individually; these are behavioral instructions
   to the model, so meaning drift matters more than style. Shipped
   rules follow the pk-managed file procedure: update the embedded
   source and the local copy together and recompute pk_sha256 per
   CLAUDE.md. Run make rules-lint after.

Verify: make test and make lint pass; a final grep proves each replaced
phrase is gone from docs/; your report lists every fact correction and
every doc/source shared-wording pair.
```

`pk rules --lint --strict` rejects em dashes in rule files; that check is a proxy, to be revisited after a rewrite pass: [docs/design.md](docs/design.md#the-em-dash-check).

## Workflow

Work happens on `develop`; `main` advances only through `pk release`. Inside Claude Code, `pk guard` blocks a commit on `main`; in a terminal, only the GitHub ruleset rejects it, at push time, so check the branch yourself.

A commit that lands on `main` by another route (a hotfix PR) is merged back before the next release, which keeps the fast-forward possible and puts it in the changelog:

```bash
git switch develop
git merge main
```

After merging a PR on GitHub, `git pull --rebase` on `develop` keeps history linear; it is safe only while the local commits are unpushed.

Dependabot targets `develop` (`.github/dependabot.yml`). GitHub's Dependabot security updates stay off: they open PRs against the default branch and ignore `target-branch`; `make vuln` covers the Go side.

## Pull requests

- **Rebase and merge** for most PRs: each conventional commit lands as itself, for `pk changelog`.
- **Merge commit** when the PR branch carries tags: a rebase makes new SHAs and orphans them.
- **Squash** is disabled by the ruleset: [Squash merge](docs/design.md#squash-merge-and-release-tags).

## Release

```bash
pk changelog --dry-run            # the section and the version
pk changelog                      # on develop: CHANGELOG.md, footprint line, install-pk.sh pin; one commit, no tag
pk release --dry-run              # every check and the preRelease hook
pk release                        # fast-forward main, tag, push main and the tag, push develop
```

The release needs the Go toolchain: `changelog.hooks.preCommit` runs `pk pin` on the bootstrap script and `go run ./evals/footprint`; `release.hooks.preRelease` runs `go test -race ./...`. The published binary has no runtime dependency.

## Limits on contributions

- **No third-party Go dependency.** A PR that adds one is not accepted.
- **Managed files are read by an AI in every session downstream.** `internal/setup/rules/`, `internal/setup/skills/` and `internal/setup/template/CLAUDE.md` are LF-normalised by `.gitattributes` and scanned by `make test` for hidden characters (the Trojan Source class: ANSI escapes, zero-width, bidi overrides). Editing one means updating its `pk_sha256` as CLAUDE.md, Updating pk-managed files, states.
- **A PR body says what changed and why.** No plan is required.
