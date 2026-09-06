# Contributing

## Build and test

```bash
make build      # docgen + go build -> ./pk
make test       # go vet + go test ./... (repo and tools/docgen)
make docs       # compile skills/ into internal/help/data (committed)
make site       # render the website into site/dist (not committed)
make fmt        # gofmt the tree
make bin-local  # build bin/pk-<os>-<arch> behind the bin/pk shim
make dist       # cross-compile every shim target into bin/
```

CI runs `make test`, checks gofmt and docgen drift, and runs
`claude plugin validate . --strict`. `make docs` rewrites the
generated regions in `skills/` (each command's Flags section, the
overview's universal flags, the Settings sections) from the command
list and the settings table, then compiles the pages. A change to a skill or to a command's flags must be committed
with the rewritten pages and the recompiled `internal/help/data`, or
the drift check fails. docgen also rejects hidden, control, and
bidirectional characters in skills, because skills ship verbatim into
other people's model contexts.

## Workflow

All changes go through `develop`; never commit directly to `main`.

In Claude Code, `pk guard` enforces this. In a terminal, GitHub branch
protection rejects the push at the server, after the commit exists
locally. Check the branch before committing.

After merging PRs on GitHub, sync the local branch with
`git pull --rebase`, only while the local commits are unpushed.

## Pull requests

- **Rebase and merge** for most PRs: linear history, and each
  conventional commit is preserved individually for `pk changelog`.
- **Merge commit** when the PR branch has tags: rebase creates new
  SHAs, which would orphan tags pointing at the originals.
- **Squash is disabled**: it collapses all commits into one, losing
  the conventional commit messages `pk changelog` depends on.

## Release

The repository releases itself with its own machinery:

```bash
pk ship --dry-run        # preview the section (or rehearse the release when one is pending)
pk ship                  # changelog then release as one command
```

or step by step, when you want to review between the halves:

```bash
pk changelog --dry-run   # preview the section and version bump
pk changelog             # on develop: CHANGELOG.md, plugin.json version, Release-Tag commit
pk release --dry-run     # rehearse the flow
pk release               # merge to main, tag, push main + tag, push develop
```

The pushed tag triggers `.github/workflows/release.yml`. It
cross-compiles the platform binaries and assembles the plugin archive.
It publishes the GitHub release with the archive (versioned and as
`plankit.zip`), the binaries, and the published `marketplace.json`.
The release commits nothing to a source branch; see docs/design.md,
The release as one derivation chain.

Publishing prerequisites, once per repository:

- Branch protection on `main` must admit the maintainer's
  fast-forward push from `pk release`. No bot ever pushes to a source
  branch.
- plankit.com deploys from `.github/workflows/site.yml` to a Cloudflare
  Pages project named `plankit`; set the `CLOUDFLARE_API_TOKEN` and
  `CLOUDFLARE_ACCOUNT_ID` repository secrets. Without them the job
  builds the site and skips the deploy.
- Until the first release exists there is no published marketplace;
  `.claude-plugin/marketplace.json` in the tree is the development
  manifest for `claude --plugin-dir` and validation only. Use that
  with `make bin-local` for local work before the first release.

Pre-release checklist:

- `make test` green and `claude plugin validate . --strict` clean.
- `make bin-local`, then in a scratch repository
  `claude --plugin-dir <this checkout>`: `/plankit:status` answers,
  and a `git commit` on a protected branch is blocked.
- **Bashless Windows gate**: on a Windows machine or VM without Git
  for Windows, so Claude Code falls back to PowerShell, install the
  release candidate and verify the guard hook fires. Plugin `bin/`
  PATH injection is a Bash-tool behavior, so a PowerShell session
  reaches pk only through the hook wiring's explicit
  `${CLAUDE_PLUGIN_ROOT}` path, and this is the run that proves it.

## Contributions & AI

plankit is a solo/small-team toolkit. Pull requests are welcome and
reviewed by the maintainer.

- **No third-party Go dependencies in the main module.** The one
  build-time exception is goldmark inside `tools/docgen`, a separate
  module that never links into pk.
- **Describe intent, not just the diff.** For substantive or
  behavior-changing PRs, say what you're changing and why in the PR
  body. No formal plan is required.
