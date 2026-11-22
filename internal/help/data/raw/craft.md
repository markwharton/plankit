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

- **Build an approved plan through, and report once.** The plan
  carries the decisions. Stop only for a decision the plan does not
  cover, a failure that cannot be resolved without changing a
  decision, or a change beyond the plan. Check: the report names every
  stop and why.
- **Test at the start of a session and report the status; test before
  and after each change.** A failure is then attributable to the
  change. Check: the suite output before and after appears in the
  session.
- **Grep before done.** When fixing a pattern or renaming, change every
  instance in the repository together. Check: a second search shows no
  hits (the `grep-done` prompt).
- **Work is done when the checks and a smoke test pass.** Build, tests,
  lint, then an end-to-end run with specific commands, their output,
  and one case that must fail. A proof whose output you did not see is
  not a proof. A pure internal refactor needs the suite only.
- **Write a diagnostic script before a second full rebuild while
  debugging.** Check: the script exists and reproduces the issue.
- **A failed text search means not found by this method.** When
  absence drives a change, confirm by parsing the structure. Check: the
  parse is shown, not the grep.

### Code

- **Preserve the structure you were given.** Do not flatten structured
  data into lists and reconstruct it with heuristics; the context is
  lost at the flattening. Check: none automated; caught only in review.
- **Fail fast, no silent fallbacks.** When something required is
  missing or wrong, stop with a message naming it. A documented
  default for an optional setting is not a fallback. Check: a test for
  the error path.

### Agents

- **Write shared code before briefing any agent.** Write the code that
  more than one agent's work will use, and one finished example of the
  work each agent will do. Agents working at the same time do not know
  what the others are writing. When shared code is missing, each agent
  writes its own version, and the versions differ. Check: the shared
  code and the example exist before the first brief.
- **Tell each agent to stop and report when it needs a change outside
  its files, or a decision its brief does not cover.** An agent that
  is not told this writes its own copy inside its files, or makes the
  decision itself. Both are faults: the copy passes every check and
  leaves the code with two versions, and the decision is made where
  the lead does not see it. Check: the brief states the rule.
- **Ask each agent to list what it added.** The report names every
  helper, table and constant the agent created, and every place its
  work differs from the example. Without the list, a second copy of a
  helper goes unnoticed. Check: the list is in the report.
- **Read an agent's code before accepting it.** Read where it meets
  the shared code and where it meets another agent's code. Search for
  a second version of each thing the report lists. Passing checks do
  not show duplication. Check: none automated; caught only in review.

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
  `BREAKING CHANGE`, `--bump`, and `--exclude` are the developer's.
  Check: guard asks before a marked commit and before a `--bump` lower
  than the commits infer; for `--exclude`, none automated.

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
