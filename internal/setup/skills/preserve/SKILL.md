---
name: preserve
description: Preserve the most recently approved plan to docs/plans/
disable-model-invocation: true
allowed-tools: Bash(pk:*)
---

Preserve the most recently approved plan to docs/plans/ and commit it.

Run:

pk preserve

This commits the plan locally with a `plan:` conventional commit. Do not push — the user decides when to push.

Report the result to the user.

With the plan preserved, proceed with its implementation.
