# pk init

Make a repository plankit-shaped: branch topology, managed files, the `v0.0.0` baseline tag, and the working branch.

## Usage

```bash
pk init                      # shape the repo you are in
pk init --push               # and publish the branches and tag to origin (also on a shaped repo, from any branch)
pk init --dry-run            # preview without changing anything
pk init --no-setup           # release management only; no .claude footprint
pk init --source work        # name the working branch something other than develop
pk init --release trunk      # release branch is trunk, not the current branch
```

`pk init` runs on a repository that already exists and has at least one commit. It does not create the GitHub repository: that needs an authenticated GitHub call, and pk shells out only to git. Create the repo first with `gh repo create` or the web UI, then run `pk init` inside the clone.

**`pk init` and [`pk setup`](pk-setup.md) act on different things.** `pk setup` sets up *pk* in a project: managed files, hooks, and modes. `pk init` sets up the *project*: branch topology, the baseline tag, the working branch, protection. That is why `pk init` runs `pk setup` internally: initializing a repository includes installing pk into it. It is also why `pk init` runs once, while you keep running `pk setup` on every pk upgrade. A second `pk init` writes nothing; see [Idempotency](#idempotency).

## How it works

1. Resolves the git repository root and the release branch (the branch currently checked out, unless `--release` names one).
2. Pre-flights: at least one commit, a clean working tree, the release branch checked out, and an `origin` remote when `--push` was given. On any failure, `pk init` refuses before writing anything.
3. Decides whether the branch-protection ruleset applies. It is skipped only when `origin` is known not to be GitHub (a local path, say). No `origin` yet is not that case: the ruleset is written, since a later run cannot add it.
4. Writes the branch topology into `.pk.json`: `guard.branches` naming the release branch, and `release.branch` naming it again. Existing keys are field-merged, not replaced.
5. Runs the `pk setup` path: managed files and modes, plus the `v0.0.0` baseline tag on the current commit. With `--no-setup`, only the baseline tag: the managed files and modes are skipped.
6. Writes `.github/protect-<release>.json` (`protect-main.json` for the default) unless the project already has one. The ruleset guards `refs/heads/<release>` by name.
7. Commits everything written so far as `chore: pk setup` (`chore: pk init` with `--no-setup`).
8. Creates the working branch (`develop` by default) from the release branch and switches to it.
9. With `--push`: pushes the release branch, then the tag, then the working branch.
10. Prints how to apply the ruleset, and what to do next.

Every step is a no-op when already satisfied, so a re-run after a partial failure completes the job. Once the working branch exists the run takes a different path entirely: nothing is written or committed. See [Idempotency](#idempotency).

## Flags

- **--push** — Publish what `pk init` produced: the release branch, the `v0.0.0` tag, and the working branch, as three pushes in that order; a push that fails stops the run, and the pushes before it stand. Requires an `origin` remote. Shaping is the action; publishing is a separate decision. `--push` also works later, on a shaped repository, from any branch.
- **--no-setup** — Skip the `pk setup` step: no managed files (CLAUDE.md, rules, skills, settings) and no hook modes. Everything that is repository shape still happens: the `.pk.json` topology, the `v0.0.0` tag, the working branch, the ruleset file, and `--push` if given. The commit is labelled `chore: pk init`. See [Release management without Claude Code](#release-management-without-claude-code).
- **--source `<name>`** — Working branch to create. Defaults to `develop`.
- **--release `<name>`** — Release branch. Defaults to `release.branch` in `.pk.json`, then to the branch currently checked out.
- **--dry-run** — Print what would happen and exit without creating, writing, tagging, committing, pushing, or applying anything.
- **--project-dir `<dir>`** — Project directory. Defaults to the current directory; resolves up to the repository root either way.

## Configuration

`pk init` writes `.pk.json` rather than reading it. It sets the topology:

```json
{
  "guard": {
    "branches": ["main"]
  },
  "release": {
    "branch": "main"
  }
}
```

The `pk setup` step then field-merges `guard.mode`, `guard.push`, and `preserve.mode` alongside. With `--no-setup` the topology is all that is written: the modes only mean something to the hooks, and no hooks were installed. An existing `.pk.json` keeps every other key, so running `pk init` on a repo that already has `changelog` config does not lose it.

See [pk-json.md](pk-json.md) for the full schema.

## Details

### Release management without Claude Code

`pk changelog` and `pk release` are standalone: they read git (the tag history, the branches, origin) and `.pk.json`, never the managed files. A repository shaped with `pk init --no-setup` therefore supports the full release flow. It carries nothing but standard git artifacts (conventional commits, a Keep-a-Changelog `CHANGELOG.md`, semver tags) plus a small `.pk.json`. Nothing under `.claude/`, no `CLAUDE.md`.

That shape fits a repository that will be handed over, published, or otherwise managed by people who do not use pk. You keep the release discipline while developing; they receive a plain repository. It also fits any project that wants versioned releases without the Claude Code wiring.

What still lands:

- The `.pk.json` topology. `guard.branches` matters standalone: `pk changelog` refuses to run on a guarded branch by reading it directly.
- The `v0.0.0` anchor and the working branch.
- One commit, labelled `chore: pk init` rather than `chore: pk setup`, because setup never ran.

The upgrade path is `pk setup`, run at any time from any branch. It field-merges the hook modes into the existing `.pk.json` alongside the preserved topology, and installs the managed files.

### Idempotency

`pk init` makes exactly one commit on the release branch, on the first run, before the working branch is created. That commit is the last thing both branches share by construction. Any later commit on the release branch is one the working branch lacks, and `pk release` cannot fast-forward past it.

So the working branch's existence selects the path. If it does not exist, the first-run steps above run, each a no-op when already satisfied: no re-tag, no re-created branch, no empty commit. If it exists, `pk init` writes and commits nothing. It confirms the other two parts of the shape: a version tag, and `release.branch` in `.pk.json` naming this release branch. It prints `Already shaped`, publishes with `--push`, and switches to the working branch if you are on the release branch. Nothing else is updated: not the release branch, not the tag, not the working tree.

Two consequences:

- A shaped repository takes `pk init --push` from any branch, since nothing is anchored on this run. Shape locally, publish the working branch with `gh repo create --source . --push`, then `pk init --push` from where you are to publish the release branch and the tag.
- Managed-file updates never arrive through `pk init`. On a shaped repository that is `pk setup`'s job, on the working branch, committed on its own.

A working branch on a repository that is *not* otherwise shaped is refused. A repository in that state is an established project; shaping it here would tag and commit on the release branch, unreachable from the working branch. See [adoption.md](adoption.md).

### The ruleset is written, not applied

Applying a ruleset needs an authenticated GitHub call, and pk shells out only to git. `pk init` writes `.github/protect-<release>.json` (`.github/protect-main.json` for the default) and prints both ways to apply it: the GitHub UI import, and the `gh api` equivalent. The file on disk is the source of truth, so a project that customizes it keeps its own policy.

The ruleset guards the release branch by name, `refs/heads/<release>`, not `~DEFAULT_BRANCH`. The release branch is the one `pk release` advances and `pk guard` blocks. It need not be the repository's default branch. A project that makes its working branch the default (so pull requests base there) must still have the release branch guarded. See [branch-protection.md](branch-protection.md).

The ruleset is written on the first run only, when the setup commit is made. It is skipped when `origin` is known not to be GitHub (a local path, say). There it would be an inert file nobody can apply, so pk says so with a `Note:`. No `origin` yet is different: the host is unknown. A later run cannot add the file without a commit on the release branch, so it is written now. The summary says how to apply it once the repository is on GitHub. A repository shaped without one gets a `Note:` on re-run; add the ruleset by hand from [branch-protection.md](branch-protection.md).

### Where v0.0.0 lands

`v0.0.0` tags the commit that was HEAD when shaping began, which is the project's own first commit. The managed files land in the *next* commit, `chore: pk setup`. The one-commit gap is deliberate: `pk changelog`'s first real release starts from where the project's code history begins, not from plankit's scaffolding.

### Why pk init creates a commit

`--push` is only meaningful if the pushed branches carry the files `pk init` just wrote. Committing before pushing is what makes the published repository match the local one. Without it, `--push` would publish branches and a tag containing none of the setup.

The commit is deliberately separate from the project's own work, matching the shipped craft rule that `pk setup` updates are committed on their own.

### Where the release branch comes from

By precedence: the `--release` flag, then `release.branch` in `.pk.json`, then the branch currently checked out. Reading `.pk.json` is what stops a re-run from redefining the project. `pk init` leaves you on the working branch, so inferring from there would rewrite `release.branch` and `guard.branches` to that branch, silently unguarding the real one. With `.pk.json` read first, a re-run from the working branch resolves the right release branch and takes the shaped path.

`guard.branches` is seeded only when the project has not set it, since a project may legitimately guard more than the release branch.

### Established repositories

`pk init` targets a fresh repository. For an established project that already has history, releases, or another release tool, see [adoption.md](adoption.md). It covers anchoring the baseline somewhere other than the first commit, and migrating existing configuration.
