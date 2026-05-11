---
name: preserve
description: Preserve the most recently approved plan to docs/plans/
disable-model-invocation: true
allowed-tools: Bash(pk:*)
pk_sha256: 44f6ba58cbce224fce791813b305cd709348fcd19dcec844fd382c4a6de1c6ba
---

Preserve the most recently approved plan to docs/plans/ and commit it. If you are in plan mode, exit plan mode first.

Run:

pk preserve

This commits the plan locally with a `plan:` conventional commit. Do not push — the user decides when to push.

Report the result to the user.

With the plan preserved, proceed with its implementation.
