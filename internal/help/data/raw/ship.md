---
name: ship
description: Cut the pending work as one release - changelog then release in a single command
---

# pk ship

Runs `pk changelog` then `pk release` as one invocation. Ship carries
no state of its own; the trailer decides what remains. A run
interrupted between the halves resumes at release on rerun.

## Usage

```bash
pk ship --dry-run
pk ship
pk ship --bump patch
```

`--bump` and `--exclude` pass through to changelog. `--dry-run`
previews the section when nothing is pending, or rehearses the release
when a Release-Tag commit exists. Limit: the release half cannot be
rehearsed before its changelog commit exists.

If the release half fails, the changelog commit stands. Rerun
`pk ship` to retry, or `pk changelog --undo` to unwind.

## Unattended releases

From Claude Code, run `pk ship --dry-run` first. If the inferred bump
is patch or minor, run `pk ship` and report the result. If the preview
shows a major, stop: list the commits carrying `!` or
`BREAKING CHANGE`, show the section, and wait for the developer's
confirmation before running `pk ship`. Decision: a major ships only
on the developer's go.
