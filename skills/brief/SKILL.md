---
name: brief
description: What each session is told about this repository's plankit policy at start, generated from .pk.json
---

# pk brief

The repository's policy, rendered as instructions for a session. The
SessionStart hook runs it on every start, resume, and compaction and
injects the text as context. Nothing is copied into the repository:
the text is derived from `.pk.json` on each run.

## Usage

```bash
pk brief
pk brief --format json
```

At a terminal, `pk brief` prints the text a session receives.
`--format json` prints the SessionStart envelope the hook emits. In an
unconfigured repository the hook is silent and the command says so.

## What it says

The first and last sentences are constant. The rest follows the
dials. `changelog.types` lists the commit types, the default table
when the file leaves it empty. `guard.breaking` decides whether the
marker rule mentions the ask. `guard.mode` and `guard.push` describe
the protected branches and pushing. `release.branch` names where
releases merge. `preserve.mode` describes how plans are kept. A dial
set to off drops its paragraph.

## Flags

<!-- generated: flags -->
```
  --format <value>
        Output format: text or json (default text)
```
<!-- /generated: flags -->
