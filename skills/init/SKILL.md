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
- the plugin entries in `.claude/settings.json`: the plankit
  marketplace under `extraKnownMarketplaces` and `plankit@plankit`
  under `enabledPlugins`, so a teammate who trusts the folder has the
  marketplace and is one `/plugin install` from the plugin. The file
  is the developer's: existing content keeps every byte, an entry
  already present under its key is left as it is, and a file with
  uncommitted changes or one that does not parse stops the run before
  anything is written.
- the commit `chore: configure plankit`, on the branch checked out
- a `develop` branch, created and checked out when the release branch
  was the only one; a repository with another branch keeps its own
- the release branch, created at the root commit when `--release`
  names one that does not exist, so it carries no unreleased work
- the brief, printed in full

No tag is placed: no version tag means nothing has been released, and
the first release covers the whole history. Work committed before init
is in it like everything after. Init says how many commits precede its
own and names the command that keeps them as history instead, `git tag
v0.0.0 <sha>` at the last of them. Nothing is pushed. The plans
directory is not created here. The preserve hook creates it when the
first plan is preserved, so a repository that never preserves a plan
never gains the directory.

## Starting a repository

```bash
git init -b main
pk init
```

`pk init --push` publishes `main` and `develop` once `origin` exists.
Set the remote's default branch to `develop` so clones start on the
working branch. A repository started on `develop` instead runs
`pk init --release main`, and `main` is created at the root.

## Procedure

Run `pk init` in the repository, or `pk init --project-dir <dir>` from
anywhere else, and relay its output. There is nothing left to commit.
`pk init --push` also pushes both branches, when the developer asks
for that.

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
is the branch checked out. When origin's default branch is another,
init refuses rather than guard a working branch, and names both. A
named branch that does not exist is created at the root commit.
`--branch` names the working branch to create; the default is
`develop`. `--no-commit` writes the files and stops, to read them
first; guard blocks a session from committing them on the release
branch, so the developer commits and branches by hand. `--push` pushes
the release branch and the working branch to `origin`, and refuses to
start without one. `--dry-run` previews every step without writing.
`--format json` emits one object: `root`, `release`, `created`,
`committed`, `branch`, `pushed`, and `dryRun`.

`pk init` refuses to run twice; edit `.pk.json` once it exists.
`pk status` reads it back and reports the first problem, and notes a
missing branch or plugin entry.

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
  --no-commit
        Leave .pk.json and .claude/settings.json uncommitted
  --push
        Push the release branch, the tag, and the working branch to origin
  --release <value>
        Release branch to guard (default: the branch currently checked out)
```

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->
