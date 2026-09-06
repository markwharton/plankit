# Contributing

## Build and test

```bash
make build      # docgen + go build -> ./pk
make test       # go vet + go test ./... (repo and tools/docgen)
make docs       # compile skills/ into internal/help/data (committed)
make fmt        # gofmt the tree
make bin-local  # build bin/pk-<os>-<arch> behind the bin/pk shim
make dist       # cross-compile every shim target into bin/
```

CI runs `make test`, checks gofmt and docgen drift, and runs
`claude plugin validate . --strict`. A change to any skill must be
committed together with its recompiled `internal/help/data` output or
the drift check fails. docgen also rejects hidden, control, and
bidirectional characters in skills: they ship verbatim into other
people's model contexts, so nothing a reader cannot see may compile.

## Workflow

All changes go through `develop`; never commit directly to `main`.

In Claude Code, `pk guard` enforces this automatically. In the
terminal, GitHub branch protection rejects the push only at the
server, after the commit exists locally, so check the branch yourself.

After merging PRs on GitHub, sync your local branch with
`git pull --rebase`. It replays your unpushed local commits on top of
the remote, keeping history linear, and is only safe when your local
commits haven't been pushed yet, which is exactly when you need it.

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

The pushed tag triggers `.github/workflows/release.yml`, which
cross-compiles the platform binaries, assembles the plugin archive,
publishes the GitHub release, and commits the archive URL and sha256
pin into `.claude-plugin/marketplace.json` on main. That pin bump is
what makes installers see the update.

Publishing prerequisites, once per repository:

- The release workflow pushes two things to `main`: the fast-forward
  from `pk release`, and its own marketplace-pin commit as
  `github-actions[bot]`. Branch protection on `main` must let both
  through (allow the maintainer and the Actions bot to push, or add
  them to the rule's bypass list) or the release fails at the push.
- The maiden release is what flips `marketplace.json` from the `./`
  development source to the archive-plus-sha256 form. Until it runs,
  marketplace installs fetch a repo with an empty `bin/`; use
  `claude --plugin-dir` with `make bin-local` for local work before
  the first release.

Pre-release checklist:

- `make test` green and `claude plugin validate . --strict` clean.
- `make bin-local`, then in a scratch repository
  `claude --plugin-dir <this checkout>`: `/plankit:status` answers,
  and a `git commit` on a protected branch is blocked.
- **Bashless Windows gate**: on a Windows machine or VM without Git
  for Windows (so Claude Code falls back to PowerShell), install the
  release candidate and verify the guard hook fires. The plugin bin/
  PATH injection is a Bash-tool behavior; PowerShell sessions reach pk
  through the hook wiring's explicit `${CLAUDE_PLUGIN_ROOT}` path, and
  this gate is what proves that path end to end.

## Contributions & AI

plankit is a solo/small-team toolkit. Pull requests are welcome and
reviewed by the maintainer.

- **No third-party Go dependencies in the main module.** The one
  build-time exception is goldmark inside `tools/docgen`, a separate
  module that never links into pk.
- **Describe intent, not just the diff.** For substantive or
  behavior-changing PRs, say what you're changing and why in the PR
  body. No formal plan is required.
