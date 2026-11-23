# Control: contain drift in the standard and the checker

## Context

The Control standard (`docs/control.md`, branch `docs-control`) and its
checker (`control/`, branch `control-checker`) drift the way the standard
describes: each change is judged against the one before it, not against
something fixed. In one day the standard's rules went from thirteen to
sixteen, one report at a time, until it was restructured around its
principles. A repository following the standard read a copy that had
since changed. The checker's reference table lives in a test the worker
can edit, so the worker could change both and the test would pass. And
no branch holds both the standard and the checker.

The same failure sits at a larger interface: between sessions over time.
A system built in one session is consistent because one worker held its
whole reference; later sessions each hold part of it, and the system
becomes the murmuration in time. The link that carries a design across
that interface is this: its invariants become holds, its decisions go
into the plan record, and each names the other.

This plan contains drift in both, and writes that link into the standard
and into craft. Each decision below names the hold that enforces it.

## Decisions

| Decision | Held by |
|---|---|
| One branch, `control`, from develop, holds the standard, the site page and the checker | `make test` passes on it: the docgen, main and control suites together |
| The branches are joined by merge, not rewritten: `@{push}` errors on both, so neither may be rewritten | none automated; the merge commits are in the log |
| This plan is preserved on `control` before the first change; it is the checker's reference | pk protect denies writes to `docs/plans`; the shell route is not held (see the protect case) |
| The table of Hamilton's sources below is the owner's reference; the checker's table is compared with it row for row, read from this preserved plan | `control/sources_test.go`, "TestAxiomsPrintsTheOwnersTable" |
| A finding from applying the standard goes to `docs/control-findings.md`; the standard changes only through a plan | none automated; caught in review |
| The standard's text is fixed between releases: the published page changes only when plankit is released or the site is rebuilt by hand | not held: the release workflow publishes it, and no test checks that |
| The standard's structure is held: every rule sits under a principle and ends with a Check, every case names a principle, every principle has a case | `control/standard_test.go`, "TestEveryRuleHasACheckAndEveryCaseAPrinciple" |
| Every "Held by" in the standard, the checker's pages and the plans names a test that exists | `control/format_test.go`, "TestEveryNamedHoldExists" |
| The two docgen test sites carry `docs/control.md`, and the site test asserts the page is built and in the navigation | `tools/docgen/site_test.go`, "TestBuildSite" |
| CI's gofmt step covers `control/` | the CI step itself |
| craft gains "Leave the reference behind" | `pk help craft` shows it; CI's drift check holds the compiled help |

## Reference: Hamilton's sources

| Source | Hamilton | Held by |
|---|---|---|
| Axiom 1 | A parent invokes its immediate children, and only those | well-formedness: one root, one parent each |
| Axiom 2 | A parent is responsible for its own output, and only its own | one-owner |
| Axiom 3 | A parent decides each child's output access | one-owner, replace-checks-owner |
| Axiom 4 | A parent decides each child's input access, "for reference only" | reads-for-reference |
| Axiom 5 | A parent rejects what is outside its own input | rejects-declared, partly: outside input only |
| Axiom 6 | A parent orders its immediate children | join-ordered |
| Theorem | Single reference, single assignment (from axioms 3 and 4) | one-owner with reads-for-reference |
| Type map | No two objects in one place | create-never-replaces, key-from-content |
| Type map | Reject values exist in all types | not held |
| Join | Each child depends on the ones before | join-ordered |
| Include | The children are independent | include-independent |
| Or | One child is chosen | or-conditions, partly: names conditions, not that they cover every case |

A later change to this table is a new approved plan with a section of the
same heading; the test reads the newest such plan.

## The standard's new text

**Drift**, a paragraph after the murmuration:

> The same happens between sessions. The session that built a system and
> the session that changes it later never meet; what passes between them
> is the code and what was written down. A system built in one session is
> consistent because one worker held its whole reference. Each later
> session holds part of it and judges each change against the code in
> front of it, and the system becomes the murmuration in time.

**Reference**, two rules:

> - **A session leaves its reference behind.** Before work is called
>   done, its invariants become holds and its decisions go into the plan
>   record. What lives only in the session is lost when it ends; what is
>   held is enforced on the next session's first change. Check: the
>   session's report lists each decision with the hold that enforces it.
> - **A decision names its hold, and a hold names its decision.** Each
>   decision in a plan names the test or check that enforces it, or says
>   it is not held. Each hold names the plan it carries. A decision with
>   no hold is a sentence in a record; a hold with no decision cannot be
>   judged by a later session, which then drops it. Check: a test finds
>   each hold a plan names and fails when it is gone; a hold that names
>   no plan is caught only in review.

**Changing this standard**, replacing its paragraph:

> A finding from applying the standard goes to `docs/control-findings.md`,
> with where it came from. The standard changes only through a plan that
> takes a batch of findings and compares each with the principles. If a
> principle covers a finding, it is a case of that principle: it goes in
> the cases, or nowhere. A new principle or a new rule is the owner's
> decision, shown against the principles it does not fit.

**Cases**, three, so every principle has one:

- **Chokepoints: a hook that watched one route.** plankit's protect hook
  denies the agent's Edit and Write tools any write to the plans
  directory, and a file written through the shell reaches it unseen.
- **Proof: a guard that read one form of its input.** plankit's guard
  asked before a commit whose message marked a breaking change, but read
  only messages given with `-m`. A message written as a here-document,
  the form agents use most, passed silently: the check had never been
  seen to fail on it. Applying this standard found it, four commands
  proved it, and tests that failed first fixed it.
- **The owner's attention: thirty-one choices put to the owner.** A
  mutation check needed, for each of its cases, the check expected to
  fail, and the owner was asked to choose all thirty-one, which needed
  knowledge of the test the owner did not have. The owner's decision was
  the rule for choosing; the worker applied it and explained the one pick
  that did not explain itself. In the same repository, stopping after
  each commit of an approved plan added the owner's time and no control.

**Applying it**, after the block: "The page changes only when plankit is
released or its site is rebuilt by hand, so a repository reads a fixed
text between releases. Not held: the release workflow publishes it, and
no test checks that."

**`docs/control-findings.md`**, new, with one finding: asking a person
before a replace is a door only if the question names each record it
would replace (from an import step that asked without checking which
files exist).

**craft, Working**, after "Build an approved plan through":

> - **Leave the reference behind.** Before work is called done, its
>   invariants become tests and its decisions go in the plan record, each
>   naming the other. Check: the report lists each decision with its test.

## Commits

1. `git switch -c control develop`; merge `docs-control`, then
   `control-checker` (both clean by `git merge-tree`).
2. Preserve this plan (`pk preserve`).
3. `test(docgen)`: both test sites write `docs/control.md`;
   TestBuildSite asserts `control.html` and the `/control` link.
   Fails first against today's test sites, passes after.
4. `test(control)`: the shared helpers below; `sources_test.go` reads
   the reference from the newest plan with the heading "Reference:
   Hamilton's sources", and its copy of the table is removed. Proven by
   a changed word in `sources.go` failing against the plan.
5. `docs(control)`: the new text above, the three cases, and
   `docs/control-findings.md`.
6. `test(control)`: `standard_test.go` (structure), and
   TestEveryNamedHoldExists widened to `docs/control.md` and
   `docs/plans/*.md`. Fails first on a scratch copy with a rule
   missing its Check and a case naming no principle.
7. `docs(craft)`: Leave the reference behind; `make docs`.
8. `ci`: gofmt covers `control`; the README says
   `cd control && go install .` puts `control` on the PATH, so another
   repository runs it whatever branch this checkout has.

Each commit carries no Co-Authored-By trailer.

## One copy of each thing

- **Markdown table rows:** one helper, `tableRows(text, heading)`, in a
  test helper file in `control/`, used by the README format test and the
  reference test. The README test's own row regexp goes.
- **Named holds:** one function reads "Held by `path`, "Test"" claims and
  resolves paths from the repository root; it serves the README, DESIGN,
  the standard and the plans.
- **Principle names:** read from the standard's headings, never listed
  in Go.
- **The reference table:** only in this plan. `sources.go` is the code,
  the plan is the reference, and no test holds a third copy.

## Verification

- `make test` on `control`: all three suites pass, the two docgen tests
  included.
- Mutation, in a scratch copy: a changed word in `sources.go` fails
  TestAxiomsPrintsTheOwnersTable; a rule without its Check and a case
  naming an unknown principle fail TestEveryRuleHasACheckAndEveryCaseAPrinciple;
  a renamed test fails TestEveryNamedHoldExists. The earlier check,
  parser and table mutants still fail by name.
- Smoke: `go run . axioms` matches the table above; `pk help craft`
  shows the new standard; `make site-preview` builds `control.html`
  with the navigation entry.

## Not in this plan

- The `store` field and the facts reader: designed in `control/DESIGN.md`,
  a plan of their own.
- Asking before a replace, in the checker's format: recorded as a
  finding, not decided here.
- Conformance with running code, per-record ownership, places that
  persist between runs, the checker's name, and whether it becomes its
  own product.
- Publishing and release: the owner's; a minor release needs its note
  in `docs/notes`.
- Folding the review-fix commits: neither branch may be rewritten.
- Closing protect's shell route: the new case records it.
