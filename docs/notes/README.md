# Release notes

One file per minor or major release, `v<version>.md`, read by
`tools/docgen` and rendered at plankit.com/notes. This file is not a
note: docgen reads only files named `v*.md`, so it ignores this one.

## What a note is

A short account of what a release changed, written for someone
deciding whether to update and what to do once they have. CHANGELOG.md
is the per-commit record and links every commit; a note is the human
reading of the same release, and never a prose restatement of the list.

## When one is written

Minor and major releases only. Patch releases get none. The
`changelog.hooks.postVersion` hook in `.pk.json` fails `pk changelog`
when the note for a minor or major is missing, so the note is written
and committed before the release is cut.

## What goes in

Distil, do not transcribe. For each commit in the release, ask whether
a reader would do something differently tomorrow because of it. If not,
it does not appear. Dropped as a matter of course: refactors,
test-only changes, release housekeeping, preserved plans, and
documentation that changed no behavior.

Six commits can make one paragraph, and one commit can be the whole
note.

## Voice

- Write forward, in the present, from the reader's side: what is true
  now and what they can do. Not what was broken, not what a commit
  landed.
- Practical, honest, concrete. No launch-speak, no "we are excited".
- Themed paragraphs, not one per commit. A minor is two to four short
  paragraphs; a major earns more.
- Open each paragraph with its claim in bold, one per paragraph:
  `**A release starts when you ask for one.**` The rest of the
  paragraph earns it. The site sets that weight in `.note strong`.
- Prose says what and why. Anything to type, and anything pk printed,
  goes in a code block. Inline code is for names: files, branches,
  keys, command names.
- Plain developer words: build, cut, ship, land, tighten. No workshop
  metaphors.
- No private repository names or paths. plankit's own are fine.
- No link to the CHANGELOG is needed: the site links each note's
  version to the compare view for that release.

## Form

```
---
version: v1.1.0
date: 2026-09-12
title: Releases start when you ask
---
```

Those three keys are required and are the only ones allowed. The
`version` must match the filename, and the title is a short claim, not
a label. The body is markdown.

The site renders notes whose tag exists, so a note written before its
release never appears early. `make site-preview` renders every note,
including the unreleased one, for a local look at
`site/dist/notes.html`.
