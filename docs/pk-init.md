# pk init

Make a repository plankit-shaped: the branch topology in `.pk.json`, the managed files, the `v0.0.0` tag, the ruleset file, one commit, and the working branch.

## Usage

```bash
pk init                      # shape the repository you are in
pk init --push               # and push the release branch, the tag and the working branch
pk init --dry-run            # print the steps; change nothing
pk init --no-setup           # topology, tag and working branch only; nothing under .claude/
pk init --source work        # working branch named work, not develop
pk init --release trunk      # release branch named trunk, not the checked-out branch
```

`pk init` runs in an existing clone with at least one commit. It does not create the repository on GitHub; pk shells out only to git.

`pk setup` installs pk into a project and runs on every upgrade. `pk init` shapes the project and runs once; it calls `pk setup` on the way.

## How it works

1. Resolves the repository root and the release branch: `--release`, then `release.branch` in `.pk.json`, then the branch checked out.
2. Pre-flight: a commit exists, the tree is clean, the release branch is checked out, and an `origin` exists when `--push` was given. A failure refuses before anything is written.
3. Writes `guard.branches` (unless already set) and `release.branch` into `.pk.json`, field-merged.
4. Runs the `pk setup` path, then tags `v0.0.0` on the current commit unless a semver tag exists. With `--no-setup`, the tag only.
5. Writes `.github/protect-<release>.json` (`protect-main.json` by default) unless the project has one. Skipped when `origin` is known not to be GitHub; with no `origin` yet, written, since a later run cannot add it without a commit on the release branch.
6. Commits everything written as `chore: pk setup` (`chore: pk init` with `--no-setup`).
7. Creates the working branch (`develop`) from that commit and checks it out.
8. With `--push`: pushes the release branch, then the tag, then the working branch, as three pushes; a failed push stops the run and the earlier pushes stand.
9. Prints how to apply the ruleset, and what to do next.

Each step is a no-op when already satisfied. Once the working branch exists, a run writes and commits nothing: it confirms the tag and `release.branch`, prints `Already shaped`, publishes with `--push`, and switches to the working branch. A working branch on a repository with no tag or no `release.branch` is refused: `branch "develop" already exists but the repository is not plankit-shaped`; see [Adoption](adoption.md).

## Flags

- **--push**: publish the release branch, the tag and the working branch. Works on a shaped repository from any branch.
- **--no-setup**: skip the managed files and the modes. `.pk.json` gets the topology only; the commit is `chore: pk init`. `pk changelog` and `pk release` read git and `.pk.json` alone, so the repository releases with nothing under `.claude/`; `pk setup` later adds the rest.
- **--source `<name>`**: the working branch. Default `develop`.
- **--release `<name>`**: the release branch. Default: `release.branch`, then the checked-out branch.
- **--dry-run**: print what would happen; write, tag, commit and push nothing.
- **--project-dir `<dir>`**: where the search for the repository root starts.

## Configuration

`pk init` writes `.pk.json`:

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

The `pk setup` step adds the modes; `--no-setup` leaves them out. Other keys are kept. See [.pk.json](pk-json.md).

## Decisions

- **`v0.0.0` sits one commit before the managed files.** The tag marks the project's first commit; `chore: pk setup` is the next. `pk changelog` reads from the tag, so the first release starts where the project's history starts and lists the setup commit as one Maintenance entry.
- **The commit comes before the branch.** Both branches share `chore: pk setup`, the last commit they share by construction; a commit made on the release branch after it is one the working branch lacks, and `pk release` refuses the fast-forward. That is why a re-run writes nothing, and why `--push` publishes the commit the files are in.
- **The release branch is read from `.pk.json` before the checked-out branch.** A re-run from the working branch would otherwise rewrite `release.branch` and `guard.branches` to that branch.
- **The ruleset guards `refs/heads/<release>` by name**, not `~DEFAULT_BRANCH`: a project may make the working branch its default. See [Branch protection](branch-protection.md).
