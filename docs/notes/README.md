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

Work from the facts, not from the voice. First list what the release
changed, as pk now behaves: a command, what it does, what it prints.
Then choose the two or three a reader would act on. Then write each as
its paragraph. A note written in the other order, voice first and facts
poured in, comes out as slogans with the changelog repeated under them.

## Voice

- Write forward, in the present, from the reader's side: what is true
  now and what they can do. Not what was broken, not what a commit
  landed.
- Practical, honest, concrete. No launch-speak, no "we are excited".
- The previous notes are the model; read two before writing one. Each
  paragraph says what is true now, why it matters to the reader, how
  they steer it, and which page has the rest. Neither a list of facts
  nor the page's flag walkthrough, and every clause checkable against
  the code, a command's output, or a page.
- Themed paragraphs, not one per commit. A minor is two to four short
  paragraphs; a major earns more.
- Open each paragraph with its claim in bold, one per paragraph. The
  claim names the thing and what it does now, so a reader can check it
  against the command or the page: `**\`pk status\` names the flow on
  its release line.**` Not a coined phrase: not "A refused step leaves
  nothing behind", not "keeps what you wrote". The rest of the
  paragraph earns the claim. The site sets that weight in
  `.note strong`.
- Before finishing, reread every bold claim and every "X, not Y"
  sentence, and rewrite any a reader has to decode. The rule, with its
  shapes, is "Say it straight" in `.claude/CLAUDE.md`.
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
