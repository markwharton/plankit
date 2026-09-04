---
name: changelog
description: Generate CHANGELOG.md from conventional commits and stage a release with the Release-Tag trailer
---

# pk changelog

Reads the conventional commits since the last version tag, infers the
next version, writes the section into `CHANGELOG.md`, and commits with
a `Release-Tag` trailer. No git tag is created here; that is
`pk release`'s moment, so the release stays a two-step: generate and
review, then tag and push.

## Version inference

A breaking change (`type!:` or a `BREAKING CHANGE:` footer) bumps
major, `feat` bumps minor, anything else bumps patch. `--bump
major|minor|patch` overrides. Types map to sections via
`changelog.types` in `.pk.json`; hidden types (like `plan`) are
tracked but never listed.

## Usage

```bash
pk changelog --dry-run
pk changelog
pk changelog --exclude abc1234,def5678
pk changelog --undo
```

`--dry-run` prints the section to stdout, redirectable, and works on a
dirty tree. `--exclude` drops commits by short hash before bump
inference. `--undo` unwinds an unpushed release commit, restoring
`CHANGELOG.md` and version files.

`changelog.versionFiles` names JSON files whose root `version` field is
spliced in place, formatting preserved. `postVersion` and `preCommit`
hooks run with `$VERSION` (no v prefix).

Run it from your working branch: the release commit belongs there, and
a protected branch is refused.
