---
name: init
description: Configure a repository for plankit - write .pk.json, create docs/plans, baseline a tag
---

# pk init

Configures the current repository for plan-driven development. One run
creates the entire per-repo footprint:

- `.pk.json`, the committed policy: the conventional-commit type table (listed under `pk help changelog`),
  guard and preserve modes stated explicitly, and the release branch
  guarded
- `docs/plans/`, where approved plans are preserved
- a `v0.0.0` baseline tag when the repository has commits but no tag,
  so release machinery has a starting point

## Usage

```bash
pk init
pk init --release trunk
pk init --dry-run
```

`--release` names the branch to guard and release from; the default is
the branch currently checked out. `--no-baseline` skips the tag.
`--dry-run` previews without writing.

Init refuses to run twice: edit `.pk.json` directly once it exists.
Commit the created files; the policy travels with the repository.
