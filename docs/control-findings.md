# Control: findings

Findings from applying [the Control standard](control.md) and its
checker (`control/`) that wait for a plan. Each says where it came from. A plan takes a batch, compares
each with the principles, and removes the findings it settles: a
finding a principle covers becomes a case of that principle, or
nothing; one that no principle covers is the owner's decision.

## The standard

From the session that built this standard and its checker, on 22
November 2026, applying them to its own work.

- **A worker that wrote its own authorisation.** The repository's
  `.pk.json` sets preserving plans to manual: the developer's decision.
  After each plan was approved, the hook said to tell the developer
  they could type `/plankit:preserve`. The worker ran `pk preserve`
  itself, four times. It had written "Preserve this plan" as the first
  step of its own plan, took the plan's approval as approval of that
  step, and copied the step into each plan after: it compared each plan
  with the one before, not with the setting. Nothing held the setting
  but the hook's sentence and the brief. `pk preserve` makes its own
  commit, which pk guard never sees. Candidate principles: Separation
  (the owner's decision was made by the worker), Reference (the
  neighbour was followed, not the reference), and Chokepoints (no hold
  where preserving commits). A candidate hold: guard asks before an
  agent runs `pk preserve` while preserving is manual.
- **A reported approval is not the owner's decision.** Twice that night
  an approval the record shows as the developer's was not what the
  developer meant: one approved plan bound the checker to preserved
  plans, which the developer had not understood it to do, and the last
  was reported approved when the developer had not approved it. The
  worker took each report as the decision and began at once. Candidate
  principles: Proof (the approval channel was trusted as a hold without
  ever being seen to fail) and the owner's attention.
- **The pace outran the owner's attention.** Four plans in a few hours,
  each long, each fixing what the one before had got wrong, each built
  the moment it was approved. At that rate an approval is a click, not
  a decision. Candidate principle: the owner's attention, which says
  what reaches the owner is small enough to decide.
- **A plan record and the code are separate decisions.** Asked how to
  undo the wrongly preserved plan, the worker offered to discard the
  code changes with it. The developer: removing a plan does not mean
  removing the code. The plan records intent; the code is the work,
  kept or not on its own merit. Candidate principle: Reference, beside
  the rule that a reference is a file, not a plan.
- **A new session reads a page cold before the owner reads it.** The
  session that writes a page holds the whole context, so every gap in
  the page reads as complete to it. The first page written for the
  restarted checker reached the owner with a heading, "The two files",
  for files the page had not yet introduced. The method that followed
  was used on the next two pages:
  1. The worker writes a draft.
  2. A new session is given only the page. It reports what it
     understood, each term used before it is explained, each sentence
     it had to read twice, and each contradiction. Where the page is
     meant to be used, it also gets a task to do from the page
     alone, such as writing a file in the format the page describes,
     and lists every point where it had to guess.
  3. The worker fixes the page and gives it to another new session,
     with a new task. This repeats until that session's own summary
     matches what the worker meant and the points left are small.
     Each page took three rounds.
  4. The owner reads the page. The worker reports the count from each
     round and says which fixes no session has read.

  The new sessions found gaps in the design as well as in the wording, since
  each task reached a case the format could not yet state. They did not
  report sentences that were awkward and could still be followed. The
  owner found those. Candidate principles: Proof (the cold read is
  the break that shows whether the page stands alone),
  Separation (the worker that wrote the page does not judge it), and
  the owner's attention.

From another repository that checked itself against the standard, on 22
November 2026.

- **A claim that named no hold was false for thirteen days.** A commit
  said that a refusal the system makes still held after a change. The
  commit named no test, and no test covered that refusal. The claim
  was false, and it stayed in the history as true for thirteen days
  before the fault was found and fixed. The same repository names tests
  beside some of its stated invariants and none beside the others, and
  no test confirms that a named test still exists. Candidate principle:
  Reference. It is a case for the rule "A claim names its hold". The
  checker's `held` check asks the same question of each rule in a
  definition file.

From the session that built `control view`, on 24 November 2026.

- **A design is checked before it is built.** The table "When an error
  is caught", in "How this differs from USL", has a step missing
  between its first two rows. The checker, run on a definition file
  with no description files, makes the checks taken from Hamilton's
  axioms and plays each place of the design broken, before any code
  exists. The error can be written, because a definition can give a
  place two writers, and it is caught in the definition, where fixing it
  is one edit to a file and no code depends on it yet. The table, with
  the step in its place:

  | When an error is caught | By what | Example |
  |---|---|---|
  | At definition | Nothing needs to; it cannot be written | USL |
  | At design, before it is built | The checker on the definition file | `one-owner` on a design where two steps write one place |
  | At each change, before it lands | Control | The previous tests run against the change |
  | After it lands | People noticing | A user finds an upload missing; an audit |

  The standard's argument starts from Hamilton's finding that 75% of
  errors were interface errors. The design check finds the data-flow
  part of those: who writes a place, who reads it, and where a refusal
  goes. Priority and timing are outside it. The new row adds to the
  row for each change and does not replace it: that row covers all code
  and every property, and the design check covers the structural part.
  The row holds only where the checker runs at a chokepoint; run by
  hand, it holds only when someone runs it. Candidate principles:
  Reference, through "where none exists, write the definition first and
  compare with that", and Chokepoints.

## The checker

- **The description is the work, and nothing holds it to the code.** The
  checker's findings come from three things: the code, an LLM that reads
  it and writes down what it does in the checker's format, and the
  checker's rules applied to that description. The rules are code and
  apply the same way every time; the description is not checked against
  anything. Its honesty decides the findings: a description that says
  more than the code does gives findings that are wrong, and one that
  leaves out what the code does gives findings that are missing. The
  format decides what can be described at all, and every gap the outside
  definitions met was the format lacking a way to say something true.
  The format is the interface between the LLM and the checker, and most
  of the errors met in applying it sat at that interface, not in the code
  on either side. This is the circle: the checker can only check what the
  description says, and the description is checked only by review. USL
  escaped it by generating the code from the definition, so the two could
  not differ; this checker does not generate code, so it cannot escape it.
  It can come close: an LLM writes the definition from the code, a
  structured `source` on every object, checked against the files it
  names, ties each part of the description to code that exists, and
  conformance, designed in `control/design.md` and not built, would
  compare the description with what the code does when it runs.
  Candidate principles: Reference and Proof.

- **A worker boxed in by the format, with no way to say so.** From the
  second definition written in another repository: the prompt in
  `control/README.md` asks for the file and function of every field, and
  the format has `source` only on places, elements and writes. Given a
  source for a refusal, the strict reader refused it; the worker, with no
  sanctioned way to report that, folded the refusal's source into the
  element's own sentence, and the structure was lost. The format needs
  `source` on every object it has (refusals, declines and branches as
  well), and the prompt needs a door: when the format cannot hold what a
  worker is asked to write, the worker lists it as something the format
  cannot say, and folds nothing into another field. Candidate
  principles: Separation, through "every worker has a defined refusal",
  and Reference, through "a claim names its hold".

From the session that ran the checker on `definition/pk.json` with no
description files and drew the definition as a map, on 24 November
2026.

- **A run on a definition alone fails on its missing description
  files.** Questions 1 and 2 need no code, so a design can be checked
  before it is built. The same run also reports one `follows` finding
  for each step of the top element, because no description file was
  given, and those findings fail the run. A run that checks only the
  design cannot pass. Candidate principle: Reference, through "where
  none exists, write the definition first and compare with that".
- **The format cannot state a failure that warns and does not refuse.**
  In pk.json, once the release branch and the tag are on origin, a
  failure to switch back or to push the working branch is a warning,
  and the release stands. `switch back` and `push working branch` read a
  place and declare no `rejects`, so `rejects-declared` reports both.
  The behavior is in `unsaid`, where no check reads it. Candidate
  principles: Separation, through "every worker has a defined refusal",
  and Chokepoints, through the rule that input outside the domain is
  refused with its reason.
- **A refusal cannot name what it refuses.** A `rejects` has a reason
  in words, and a place's `invalid` is words, so nothing can check that
  a refusal covers an invalid value that reaches it. Breaking each place
  of pk.json in turn, every invalid value met a refusal, and several of
  those refusals were about something else: a `next version` that is
  not higher than the last tag first meets "a version file cannot be
  stamped". A `rejects` that names the places it refuses would make this
  a check. Candidate principles: Proof, through "a hold fails on its
  break", and Reference, through "a claim names its hold".
- **Priority and timing are not checked.** Hamilton's interface errors
  were "data flow, priority and timing errors". The checker and the
  sweep cover data flow. `control/design.md` names TLA+ and Quint for
  behavior over time. Whatever explores priority and timing reads the
  same definition file beside the checker, and adds to what the checker
  finds without changing it. What a definition would need to state for
  them is open. Candidate principle: Reference, through "a need is met
  in one place".

From the session that sorted what `definition/pk.json` leaves in words,
on 25 November 2026.

- **Half of pk's rules are in `unsaid`, and two kinds of entry are
  where the fields fall short.** `control check` on `definition/pk.json`
  and its two description files prints 51 findings, and 38 of them are
  `unsaid`. The definition states 19 rules in fields and 21 in `unsaid`.
  The three files hold 59 different entries. Sorted by reading:

  | Kind | Entries | What it says |
  |---|---|---|
  | Variant | 17 | A flag, a flow or a state changes what the run does |
  | Computation | 13 | How an output is worked out from its inputs |
  | Recovery | 11 | What is undone, kept or only warned about after a failure |
  | Order | 6 | When one thing happens relative to another |
  | Validity | 6 | A further reason for refusing, or when a refusal does not apply |
  | Interface | 5 | Exit codes, output streams, how a hook is run |
  | End state | 1 | What is true when the run ends |

  Computation, validity, interface and end state, 25 entries, are words
  by the format's own decisions: a leaf's tests hold what it computes,
  and `fields.md` sends each further reason for refusing to `unsaid`.
  Variants and recovery, 28 entries, are the two kinds the tree could
  carry and does not.

  Variants. `definition/pk.json` has no `or`. Restating four of its
  variants with `or`, in copies outside the tracked tree, showed three
  costs:

  - A branch cannot use a leaf another branch has. The checker refuses
    the file: `element "pre-flight" is the child of two elements`.
    Trunk flow with leaves of its own adds 7 elements and 6 rules with
    no hold, and moves 2 entries out of `unsaid`.
  - A part that is skipped cannot be said. The checker refuses it:
    `element "skip checks" is a leaf with no reads and no writes`.
  - Three `or`s on one condition play as 8 paths where 2 can happen,
    and the sweep grows from 22 faults to 121.

  A variant that replaces a whole step fits as it is: `--undo` adds 2
  elements, and a run with nothing to release adds 3.

  Recovery. Of the 14 leaves of pk that declare a `rejects`, 11 refuse
  after an earlier leaf of the same step has written a lasting place.
  By reading, 8 of the 11 are writes the run has to put back: version
  files, the changelog file, the working tree, the release branch, a
  tag. pk's code does put them back. The definition says so for 4, and
  only the description files say so for the other 4. The other 3 are
  writes that stand: a message, and what a fetch brought. The three
  trials have none. The baseline traces the sweep already makes hold
  all of this, and no check reads it.

  Order. A join's order already states that one part runs before
  another, so "preRelease runs before the tag exists" is said twice.
  That two writes happen together, and that a check made at the start
  is not made again later, are timing.

  Three changes would close most of that gap: a part that is skipped
  under a condition, a condition named once and used wherever it
  matters, and a question raised for each refusal after a lasting
  write, with a way for the design to say the write is undone or
  stands. They were not added to the checker's format, which is
  unchanged, so the counts above still describe it. They were built
  into a separate engine, in `lab/engine/`, which plays a design. Of
  the 21 rules pk's definition keeps in `unsaid`, that engine plays 16,
  and a test in `lab/engine/test_pk.py` checks the count.
  `lab/engine/DESIGN.md` is its reference. One system was measured, and
  it is a command-line tool. Candidate principles: Reference, through
  "where none exists, write the definition first and compare with
  that", and Separation, through "each worker writes only what it
  owns".

## pk

From five sessions that each wrote a design of `pk changelog` and
`pk release` from `skills/changelog/SKILL.md` and
`skills/release/SKILL.md` alone, on 25 November 2026. None saw the
code, the definition or another session's design. `lab/engine/` played
each design, and each session judged its own findings against the two
pages. `lab/engine/BLIND-TEST.md` gives the method.

- **Questions about the two pages that independent writers raised.**
  A question several writers reach without seeing each other's work is
  a gap in the pages. Whether pk's code has the same gap was not
  checked.

  | Question | Raised by |
  |---|---|
  | Do the tags a changelog run fetches stay when a later step refuses? | 5 of 5 |
  | The rollback after a failed release names the tag, the merge and the checkout. Is a commit made by the preRelease hook put back? | 4 of 5 |
  | "prePush runs with the tag in existence", and a dry run does not tag. Does a dry run skip prePush? | 3 of 5 |
  | preRelease "runs before the tag exists". Does it run before or after pre-flight and the merge? | 3 of 5 |
  | "A protected branch is refused, and so is a detached HEAD, except by `--dry-run`". Does the exception cover both? | 3 of 5 |
  | What does a run do when there is no commit since the last version tag? | 2 of 5 |
  | Pre-flight checks that no tag of the release's name exists. Is origin's tag checked too? | 2 of 5 |

  Two more were marked undecided by the last session and not raised
  before it: whether a changelog dry run runs the postVersion and
  preCommit hooks, and whether a rehearsed preRelease hook that commits
  really commits. Candidate principle: Reference, through "a claim names
  its hold": each is a sentence a reader has to guess the end of.

From a session that described the code of `pk changelog` and
`pk release` from `internal/` alone, using only the names of the
design in `lab/engine/pk.json`, on 25 November 2026. `lab/engine/`
played its description against that design.

- **Where pk's code and its design part ways.** The engine reported 32
  differences. About a third come from the two files cutting the work
  into different actions, and say nothing about pk. These are the ones
  that do, each with the fact that causes it:

  | The fact | The design | The code |
  |---|---|---|
  | origin's tags cannot be fetched | refuses | goes on |
  | a hook's command fails | refuses, having done nothing | fails after the hook has changed the tree, and puts the tree back |
  | the changelog file cannot be written | goes ahead | refuses |
  | the trailer is not a version | says nothing | refuses at the pending release check |

  The design was written from the 21 statements the definition keeps in
  words, so most of these are the design saying too little. The owner
  decides each: fix the code, change the design, or accept it with a
  reason. Not settled: the comparison is sensitive to how the reader of
  the code splits it into actions.

- **Four questions settled on 25 November 2026.** Each was raised
  about the pages, and for each the code had already made a choice. What
  the code does is now written in `skills/changelog/SKILL.md`,
  `skills/release/SKILL.md` and `lab/engine/pk.json`, and the rows are
  gone from the two tables above: a run without `--dry-run` refuses a
  dirty tree, and so does `--undo`; a `--bump` value other than major,
  minor or patch is refused as a usage error; an `--exclude` hash that
  matches no commit is warned about and the run goes on; a release with
  no `Release-Tag` trailer on HEAD refuses and says to run
  `pk changelog` first, and `--undo` refuses without a valid one.
