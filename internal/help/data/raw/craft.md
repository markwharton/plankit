---
name: craft
description: Engineering standards and reusable prompts for working in a plankit repository - load the standards, or run a named prompt
argument-hint: [prompt]
---

# craft

craft is the engineering standards and prompts for working in a
plankit repository. `/plankit:craft` loads the standards into the
session. `/plankit:craft <prompt>` loads this page with the prompt name as
its argument, and the session runs that prompt. `pk help craft` prints this page.

## Standards

### Working

- **Test at the start of a session and report the status; test before
  and after each change.** Reason: a failure is then attributable to
  the change. Check: the suite output before and after appears in the
  session.
- **Grep before done.** When fixing a pattern or renaming, change every
  instance in the repository together. Check: a second search shows no
  hits (the `grep-done` prompt).
- **Work is done when the checks and a smoke test pass.** Build, tests,
  lint, then an end-to-end run with specific commands, their output,
  and one case that must fail. A proof whose output you did not see is
  not a proof. Limit: a pure internal refactor needs the suite only.
- **Write a diagnostic script before a second full rebuild while
  debugging.** Check: the script exists and reproduces the issue.
- **A failed text search means not found by this method.** When
  absence drives a change, confirm by parsing the structure. Check: the
  parse is shown, not the grep.

### Code

- **Preserve the structure you were given.** Do not flatten structured
  data into lists and reconstruct it with heuristics. Reason: the
  context is lost at the flattening. Check: none automated; caught only in review.
- **Fail fast, no silent fallbacks.** When something required is
  missing or wrong, stop with a message naming it. Limit: a documented
  default for an optional setting is not a fallback. Check: a test for
  the error path.

### Git

- **Git decisions are the developer's.** A request to commit is not a
  request to push; approval to push once is not approval for the next.
  Check: guard blocks or asks on pushes per `.pk.json`.
- **On unexpected git state, stop and report.** Do not pull, rebase,
  merge, or reset unprompted. Run `git log --oneline --graph HEAD
  origin/<branch>`, show it, wait. Check: none automated; caught only in review.
- **Surface partial failures at once.** Say what happened, what state
  changed, and the remedy. Check: none automated; caught only in review.
- **Rewrite only unpushed commits.** `git log --oneline @{push}..HEAD`
  must list the target; if it errors or the target is absent, make a
  new commit. Check: the command's output appears before the rewrite.
- **Never mark a change breaking on your own judgment.** `!`,
  `BREAKING CHANGE`, and `--bump` are the developer's. Check: guard
  asks before a marked commit; for `--bump`, none automated.

## Prompts

### review

Code review: DRY violations, anti-patterns, design tokens, security.

Run it as a second pass, after the first pass has the code working.
The first pass will contain DRY violations; that is expected.
Refactoring is pattern recognition, so the second pass is where the
model does its best work. Keep the prompt to the line above: it leaves
room to be comprehensive, and adding scope or examples narrows it. The
developer directs both passes and decides what ships.

### audit-docs

Audit the documentation against the code before rewriting a word.

List every claim in the affected documents that describes behavior.
Verify each against the source or by running the command. Record each
disagreement as a fact correction; commit corrections before any style
change. Then rewrite for the four sentence kinds without changing a
fact.

### smoke

Prove the change end to end.

Name the commands, run them, show the output, including one case that
must fail and its failure. For a pure internal refactor, run the suite
and say so.

### grep-done

Find every occurrence before declaring done.

Search for the old form, list every hit, change them together, search
again, show the empty result.
