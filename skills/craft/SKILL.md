---
name: craft
description: Engineering standards and reusable prompts for working in a plankit repository - load the standards, or run a named prompt
argument-hint: [prompt]
---

# craft

Standards that were paid for, and prompts that apply them. Invoked
bare, this page loads the standards for a session about to do real
work. Invoked with a name, run that prompt against the current work:
`/plankit:craft review`. At a terminal, `pk help craft` prints the
same page.

## Standards

### Working

- **Test at the start of a session and report the status.** Then test
  before and after each change, so a failure is attributable to the
  change and not to what came before it.
- **Grep before done.** When fixing a pattern or renaming, find every
  instance in the repository and change them together. One fix is not
  done until every occurrence is done.
- **Work is done when the checks and a smoke test pass.** Build, tests,
  and lint; then an end-to-end check with specific commands and
  observable outcomes, including one negative case, whenever the
  change alters observable behavior. A proof whose output you did not
  see is not a proof. Skip the smoke test for a pure internal refactor.
- **Diagnostic scripts over rebuild cycles.** Before a second full
  rebuild while debugging, stop and write a minimal script that tests
  the specific issue.
- **A failed text search means not found by this method, never not
  present.** When absence drives a root cause or a change, confirm by
  parsing the structure, not by matching the text.

### Code

- **Preserve the structure you were given.** Let the data model drive
  the code. Never flatten structured data into lists and reconstruct
  it with heuristics; the context is already lost.
- **Fail fast, no silent fallbacks.** When something required is
  missing or wrong, stop with a clear message. Never substitute a
  made-up value that hides the problem. A documented default for an
  optional setting is not a fallback; a guess in place of a required
  one is.

### Git

- **Git decisions are the developer's.** Carry them out, do not
  originate them. A request to commit is not a request to push, and
  approval to push once is not approval for the next push.
- **On unexpected git state, stop and report.** A diverged branch, a
  branch behind its remote, anything you did not expect: do not pull,
  rebase, merge, or reset on your own. Show `git log --oneline --graph
  HEAD origin/<branch>`, say what you see, and wait.
- **Surface partial failures at once.** When an operation half
  succeeds, say what happened, what state changed, and the remedy.
  Never continue silently.
- **Rewrite only unpushed commits.** Confirm the target appears in
  `git log --oneline @{push}..HEAD`; if the command errors or the
  target is absent, it has been pushed. Make a new commit instead.
- **Never mark a change breaking on your own judgment.** `!` and
  `BREAKING CHANGE` drive the next major version and are the
  developer's claim. The same holds for `--bump`: it is the developer's
  override, never yours.

## Prompts

### review

Code review: DRY violations, anti-patterns, design tokens, security.

Review the current change, not the whole repository. Report each
finding with file and line, the rule it breaks, and the smallest
change that fixes it. Duplicated logic is a finding even when each
copy is correct. Hard-coded values that a design token or a config
key already names are findings. For security, look for input that
reaches a shell, a query, or a file path unescaped, and for secrets in
the tree.

### audit-docs

Audit the documentation against the code before rewriting a word.

List every claim in the affected documents that describes behavior,
and verify each one against the source or by running the command.
Where a document and the code disagree, record it as a fact
correction, separately from any style change. Commit corrections
before style. Then, and only then, rewrite for clarity without
changing a fact.

### smoke

Prove the change end to end with commands and observable outcomes.

Name the commands you will run, run them, and show the output. Include
one case that must fail and show it failing. If the change is a pure
internal refactor, say so and run the test suite instead.

### grep-done

Before declaring a rename or a pattern fix done, find every occurrence.

Search the repository for the old form, list every hit, and change
them together. Then search again and show the empty result.
