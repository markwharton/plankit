---
name: preserve
description: Preserve the most recently approved plan to docs/plans/
disable-model-invocation: true
allowed-tools: Bash(pk:*)
pk_sha256: a2c5d044b83ff3105610267085e3378c29abb2045061a8c24d46115b8e5ad508
---

Preserve the most recently approved plan to docs/plans/ and commit it. If you are in plan mode, exit plan mode first.

Run using the **Bash** tool (not PowerShell):

pk preserve

This commits the plan locally with a `plan:` conventional commit. Do not push — the user decides when to push.

Report the result to the user.

With the plan preserved, proceed with its implementation.

## Rules

- **Use the Bash tool for all commands.** pk requires a POSIX shell. Do not use the PowerShell tool.
