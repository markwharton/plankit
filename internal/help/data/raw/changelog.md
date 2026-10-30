---
name: changelog
description: Generate CHANGELOG.md from conventional commits and stage a release with the Release-Tag trailer
---

# pk changelog

Reads the conventional commits since the last version tag, infers the
next version, writes the section into `CHANGELOG.md`, and commits with
a `Release-Tag` trailer. No git tag is created; `pk release` creates
it, so the release stays a two-step: generate and review, then tag
and push.

## Version inference

A breaking change (`type!:` or a `BREAKING CHANGE:` footer) bumps
major, `feat` bumps minor, anything else bumps patch. The commits
decide the version; `--dry-run` shows the one they infer, and a version
named before the commits existed is a guess, never a target. When the
inferred version is not the one expected, stop and say so. A commit
with the wrong type is fixed by amending its type, while it is
unpushed, not by overriding the bump.

`--bump major|minor|patch` overrides the inference, and the run names
the override next to the version it generates:

```
Generating v0.8.1 (--bump patch; the commits infer minor)
```

`--bump` and `--exclude` are the developer's: `--bump` for the case
the commits cannot show, `--exclude` to leave commits out of the
section. Use either only when the developer asks, with the exact value
given. guard asks before a `--bump` lower than the commits infer; see
`pk help guard`, Version bumps.

Types map to sections via `changelog.types` in `.pk.json`; hidden
types are tracked but never listed. The default table `pk init`
writes:

```
feat       Added
fix        Fixed
deprecate  Deprecated
revert     Removed
security   Security
refactor   Changed
perf       Changed
docs       Documentation
chore      Maintenance
test       Maintenance
build      Maintenance
ci         Maintenance
style      Maintenance
plan       (hidden: written by pk preserve, never listed)
```

Edit `changelog.types` to rename sections, add types, or hide more.
An empty table means the default table.

## Usage

```bash
pk changelog --dry-run
pk changelog
pk changelog --exclude abc1234,def5678
pk changelog --undo
```

`--dry-run` prints the section to stdout and runs on a dirty tree.
`--exclude` drops commits by short hash before bump inference.
`--undo` unwinds an unpushed release commit, restoring `CHANGELOG.md`
and version files.

`changelog.versionFiles` names JSON files whose root `version` field
is spliced in place, formatting preserved. `postVersion` and
`preCommit` hooks run with `$VERSION` (no v prefix).

Run it from the working branch. A protected branch is refused. The
last version tag must be in the branch's history: a branch rewritten
after a release is refused, naming the tag, because the entry measured
from it would list that release's work again. Move the tag to the
matching commit in the new history, or delete it if it was never
meant, and update origin's copy.

## The first release

No version tag means nothing has been released. The first release
covers the whole history from an implied `v0.0.0`, and its entry links
to the tag's commits rather than a compare. The run says so first:
`no release yet: the first release covers all N commits`. To keep
earlier history out of it, tag `v0.0.0` at the last commit that is
history before running `pk changelog`; everything after that tag is
the first release.

A clone with no local tags is not a first release when origin has
them: a fresh clone that fetched only the working branch, as a cloud
sandbox does. `pk changelog` fetches origin's tags first and says so,
`fetched tags from origin: latest is v1.2.0`, then measures from that
tag. Only when the fetch itself fails does it stop, with the error and
the hint `git fetch --tags`.

## Flags

<!-- generated: flags -->
```
  --bump <value>
        Override the inferred bump: major, minor, or patch (the developer's call)
  --dry-run
        Print the section to stdout without writing or committing
  --exclude <value>
        Comma-separated commit SHAs to drop from the section (the developer's call)
  --undo
        Unwind the last pk changelog commit (must be unpushed)
```

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->

## Settings

<!-- generated: settings -->
The `changelog` section of `.pk.json`:

```
"changelog": {
  "hooks": {
    "postVersion": "<command>",
    "preCommit": "<command>"
  },
  "showScope": true | false,
  "types": [{ "hidden": true | false, "section": "<section>", "type": "<type>" }, ...],
  "versionFiles": [{ "path": "<path>", "type": "json" }, ...]
}
```

- `changelog.hooks.postVersion`: a shell command; default none. Runs after the version files are stamped and before CHANGELOG.md is written, with `$VERSION`.
- `changelog.hooks.preCommit`: a shell command; default none. Runs before the release commit, with `$VERSION`.
- `changelog.showScope`: `true` or `false`; default `false`. Includes each commit's scope in its changelog entry.
- `changelog.types`: a list of `{"type", "section", "hidden"}` rows; default the table under `pk help changelog`. Maps commit types to changelog sections; a hidden type is tracked and never listed; an empty list means the default table.
- `changelog.versionFiles`: a list of `{"path", "type"}` rows; default none. JSON files whose root `version` field is stamped in place at release.

An unknown key or a value outside these fails the whole file when it loads, with a message naming the key: `pk` commands exit 2, and each hook reports the message and takes no action until it is fixed. An absent key means its default. `pk status` reads the file back and reports the first problem.
<!-- /generated: settings -->
