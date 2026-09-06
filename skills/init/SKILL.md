---
name: init
description: Configure a repository for plankit - write .pk.json and baseline a tag
---

# pk init

Configures the current repository. One run creates:

- `.pk.json`, the committed policy: the commit-type table (listed
  under `pk help changelog`), guard and preserve modes stated in
  full, and the release branch guarded. Its first line names the
  file's JSON Schema, `https://plankit.com/pk.schema.json`, generated
  from the same table as each page's Settings section, so an editor
  validates the file as it is typed.
- a `v0.0.0` baseline tag when the repository has commits but no tag

`docs/plans/` is not created here. The preserve hook creates it when
the first plan is preserved, so a repository that never preserves a plan never
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

`pk init` refuses to run twice; edit `.pk.json` once it exists, and
`pk status` reads it back and reports the first problem. Commit the
created files.

## Flags

<!-- generated: flags -->
```
  --release <value>
        Release branch to guard (default: the branch currently checked out)
  --no-baseline
        Skip creating the v0.0.0 baseline tag
  --dry-run
        Preview without making any changes
```
<!-- /generated: flags -->
