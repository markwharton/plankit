---
name: brief
description: What each session is told about this repository's plankit policy at start, generated from .pk.json
---

# pk brief

The repository's policy, rendered as instructions for a session. The
SessionStart hook runs it on every start, resume, and compaction and
injects the text as context, so a session knows the commit
convention, the breaking-marker rule, the protected branches, and how
plans are kept before its first command. Nothing is copied into the
repository: the text is derived from `.pk.json` each time, so it
cannot disagree with what guard, protect, and preserve then enforce.

## Usage

```bash
pk brief
```

At a terminal it prints the same text a session receives, so you can
read exactly what your sessions are told; `--format json` prints the
SessionStart envelope the hook emits. In an unconfigured repository
the hook is silent and the command says so.

## What it says

Two sentences are constant, the first and the last. Everything
between follows the dials: `changelog.types` lists the commit types
(the default table when the file leaves it empty), `guard.breaking`
decides whether the marker rule mentions the ask, `guard.mode` and
`guard.push` describe the protected branches and pushing,
`release.branch` names where releases merge, and `preserve.mode`
describes how plans are kept. A dial set to off drops its paragraph.
