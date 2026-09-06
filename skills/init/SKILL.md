---
name: init
description: Configure a repository for plankit - write .pk.json and baseline a tag
---

# pk init

Configures the current repository. One run creates:

- `.pk.json`, the committed policy: the commit-type table (listed
  under `pk help changelog`), guard and preserve modes stated in
  full, and the release branch guarded
- a `v0.0.0` baseline tag when the repository has commits but no tag

`docs/plans/` is not created here. Preserve creates it when the first
plan is preserved, so a repository that never preserves a plan never
gains the directory.

## Usage

```bash
pk init
pk init --release trunk
pk init --dry-run
```

`--release` names the branch to guard and release from; the default
is the branch checked out. `--no-baseline` skips the tag. `--dry-run`
previews without writing.

Init refuses to run twice; edit `.pk.json` once it exists. Commit the
created files.
