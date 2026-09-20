---
name: init
description: Configure a repository for plankit
---

# pk init

Configures a repository in one run:

- `.pk.json`, the committed policy: the commit-type table (listed
  under `pk help changelog`), guard and preserve modes stated in
  full, and the release branch guarded. Its first line names the
  file's JSON Schema, `https://plankit.com/pk.schema.json`, generated
  from the same table as each page's Settings section, so an editor
  validates the file as it is typed.
- the commit `chore: configure plankit`, on the branch checked out
- the `v0.0.0` baseline tag, when the repository has no tag
- a `develop` branch, created and checked out when the release branch
  was the only one; a repository with another branch keeps its own
- the brief, printed in full

Nothing is pushed. `docs/plans/` is not created here. The preserve hook
creates it when the first plan is preserved, so a repository that never
preserves a plan never gains the directory.

## Procedure

Run `pk init` in the repository, or `pk init --project-dir <dir>` from
anywhere else, and relay its output. There is nothing left to commit.
`pk init --push` also pushes the branches and the tag, when the
developer asks for that.

## Usage

```bash
pk init
pk init --release main
pk init --branch dev
pk init --push
pk init --dry-run
pk init --format json
```

`--release` names the branch to guard and release into; the default
is the branch checked out. `--branch` names the branch to create; the
default is `develop`. `--no-commit` writes `.pk.json` and stops, to
read the file first; guard blocks a session from committing it on the
release branch, so the developer commits, tags, and branches by hand.
`--no-baseline` skips the tag. `--push` pushes the release branch, the
tag, and the new branch to `origin`, and refuses to start without one.
`--dry-run` previews every step without writing. `--format json`
emits one object: `root`, `release`, `created`, `committed`,
`baseline`, `branch`, `pushed`, and `dryRun`.

`pk init` refuses to run twice; edit `.pk.json` once it exists.
`pk status` reads it back and reports the first problem, and notes a
missing tag or branch.

In a repository still carrying the files a v0.x `pk setup` copied in,
the refusal says so and points at `pk help overview`: those files wire
the hooks the plugin already ships, so each one fires twice.

## Flags

<!-- generated: flags -->
```
  --branch <value>
        Working branch to create when the release branch is the only one (default develop)
  --dry-run
        Preview without making any changes
  --format <value>
        Output format: text or json (default text)
  --no-baseline
        Skip creating the v0.0.0 baseline tag
  --no-commit
        Leave .pk.json uncommitted
  --push
        Push the release branch, the tag, and the working branch to origin
  --release <value>
        Release branch to guard (default: the branch currently checked out)
```

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->
