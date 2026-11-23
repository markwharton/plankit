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
