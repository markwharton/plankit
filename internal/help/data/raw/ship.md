---
name: ship
description: Cut the pending work as one release - changelog then release in a single command
---

# pk ship

Runs `pk changelog` then `pk release` as one invocation. `pk ship`
carries no state of its own: what remains is the `Release-Tag` trailer
on HEAD and whether its tag exists. A run interrupted between the
halves resumes at release on rerun.

## Usage

```bash
pk ship --dry-run
pk ship
```

`--dry-run` previews the section when nothing is pending, or rehearses
the release when a Release-Tag commit exists; the release half cannot
be rehearsed before its changelog commit exists.

If the release half fails, the changelog commit stands. Rerun
`pk ship` to retry, or `pk changelog --undo` to unwind.

## Deciding the version

The commits decide the version. `pk ship --dry-run` shows the bump
they infer, and that bump is the release. A version named earlier, in
a plan or in the conversation, was a guess made before the commits
existed; it is never a target to make the release match. When the
inferred version is not the one that was expected, stop and say so,
with the commits that set it. A commit whose type is wrong, a `feat`
that is really a `fix` or a `chore`, is fixed by amending that commit's
type before shipping, while it is unpushed; the bump then follows.

`--bump` and `--exclude` are the developer's, and pass through to
changelog. `--bump` is for the case the commits cannot show, when the
developer knows the inferred bump is wrong for a reason outside the
messages. `--exclude` leaves commits out of the section, which is the
developer's call in the same way. Use either only when the developer
asks for it, with the exact value given.

```bash
pk ship --bump patch       # only when the developer asked for patch
```

guard asks before a `--bump` lower than the commits infer, patch over a
`feat` or minor over a breaking change, and lists the commits that call
for the higher bump; see `pk help guard`, Version bumps. The changelog
half names an override next to the version it generates.

## Releasing

Releasing is the developer's call. Run `pk ship` when the developer
asks for a release, not on your own reading of a preview. When the
inferred bump is a major, show the commits carrying `!` or
`BREAKING CHANGE` and the section, and wait for the developer's go.

## Flags

<!-- generated: flags -->
```
  --bump <value>
        Override the inferred bump: major, minor, or patch (the developer's call; passed to changelog)
  --dry-run
        Rehearse: preview the section, or the release when one is already pending
  --exclude <value>
        Comma-separated commit SHAs to drop from the section (the developer's call; passed to changelog)
```

Every command also accepts `--plain`, `--project-dir <value>`, and `--quiet`; see `pk help help`.
<!-- /generated: flags -->
