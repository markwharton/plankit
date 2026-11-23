# Control

A system built with AI workers changes faster than its owner can
check by reading. This page describes a way of
working that keeps such a system from drifting: what must not change
is held by mechanisms that run on every change. Any worker can add
one; only the owner can remove one.

## Terms

| Term | Meaning |
|---|---|
| Owner | The person who decides what the system does. |
| Worker | Anything that changes the system: a person, an agent, a flow, a script. |
| Control | The set of holds on a system. |
| Hold | A mechanism that fails when its property breaks, in the system as it runs: a test, a check, a guard, a comparison with a reference. |
| Under control | Every named property of the system is held. |
| Drop | Removing or loosening a hold. |
| Drift | The result of drops nobody saw. |
| False hold | A hold that passes whether or not its property holds. |
| Reference | What the work is compared with: an older system, recorded behavior, or a definition written first. |
| Baseline | The unbroken system, on which a hold must pass. |

The owner decides, the worker changes, and control holds.

## Hamilton and USL

Margaret Hamilton led the on-board flight software for Apollo. After
Apollo her team analysed the errors found in final testing
(Hamilton & Hackler 2007):

> "Interface errors (data flow, priority and timing errors from the
> highest to the lowest levels of a system to the finest grain)
> accounted for ~75% of all errors found"

An interface error sits between two parts that each meet their own
contract. From that study she derived six axioms of control, the
relations of a parent over its children: it starts only its own
children, owns only its own output, decides what each child may read
and write, rejects what is outside its own input, and orders its own
children. A child reads its inputs "for reference only". The axioms
became the Universal Systems Language (USL), in which a system is
defined and its code generated, so that an interface error cannot be
written.

Her argument was against approaches that support "'fixing wrong
things up' rather than 'doing things in the right way in the first
place'": errors found by testing and by people noticing, after they
are already in the system.

In other accounts Hamilton tells of the Apollo 8 flight, when a
crew member selected the pre-launch program P01 in flight and erased
the navigation data. She had wanted the software to guard against it
and was told the astronauts were trained not to; a note in the
documentation said not to select P01 in flight. A rule that relies on
the operator remembering it is not held.

## How this differs from USL

In USL, control is part of the definition. The language cannot say
that a child writes its parent's input, so the error never exists and
nothing needs to catch it. Control prevents an error from being
written.

Today's workers write in general-purpose languages, where anything can
be said: a test can rewrite the production code it is testing.
Systems also meet services whose behavior we can describe but not
define. Control cannot live in the language, because we do not own the
language. It lives in mechanisms that run against each change. The
worker can still write the error; the change fails a hold before it
lands. Control prevents an error from passing.

| When an error is caught | By what | Example |
|---|---|---|
| At definition | Nothing needs to; it cannot be written | USL |
| At each change, before it lands | Control | The previous tests run against the change |
| After it lands | People noticing | A user finds an upload missing; an audit |

The owner's part changes with it. In USL the parent controls by
defining. Here the owner controls by deciding which holds exist and
which are dropped, and control carries out those decisions on every
change, including the ones the owner never sees.

USL's guarantee is complete because the language is the proof. A hold
is ordinary code: it can be edited, it can rewrite what it measures,
and it can be false.

## Drift

A system drifts when three things are true at once: something that
should be fixed can be changed by whoever is doing the work, there is
nothing to compare it with, and nobody looks before the next change is
built on it. Control answers each one. What judges the work is outside
the worker's reach, every change is compared with a reference, and
the owner decides each plan before it is built.

Review without a reference does not hold. Each piece can be small,
seen by the owner and approved, and the whole still drifts, because
each piece is judged against the one before it rather than against
anything fixed. A starling murmuration moves this way: each bird
matches its nearest neighbours, every move looks right locally, and
the flock goes wherever it goes. The owner's review holds only as well
as the reference it is compared with.

The same happens between sessions. The session that built a system and
the session that changes it later never meet; what passes between them
is the code and what was written down. A system built in one session is
consistent because one worker held its whole reference. Each later
session holds part of it and judges each change against the code in
front of it, and the system becomes the murmuration in time.

Each drop happens in one change: a commit that deletes an assertion,
an edit to what a test sets up, a summary that leaves out a standing
rule. Reading that change finds it. Drift is spread across many
changes, so it shows only after enough drops have added up, usually
when someone notices the system behaving wrongly. Control works at the
drop.

## Principles and rules

Five principles carry the standard. Each is followed by the rules that
come from it. Each rule ends with how it is checked, and a rule whose
check reads "none automated" is held only by review.

### Separation

Each part writes only its own. The worker does not write what judges
the work, what judges the work does not write it, and one worker does
not write another's. A worker that can change its judge, or a judge
that changes the work, measures nothing.

- **A worker may add holds; only the owner drops or changes one.**
  Adding a test or a guard tightens the system. Deleting an assertion,
  loosening an expected value, changing what a test sets up, or
  removing a guard is the owner's decision. The decision reaches the
  change through something the worker cannot write, such as a
  permission prompt or the owner's own commit; a note the worker writes
  in the commit message is the worker's. A standing rule from the
  owner sets the direction, and each drop under it is still the
  owner's to decide. Check: the previous tests run against the change,
  and any failure goes to the owner before a test is edited.
- **Control does not change the work.** A test reaches the system only
  through the inputs the system declares. It never patches, replaces
  or rewrites the system's own code; a test that does is measuring a
  program that never runs. Check: no test replaces production code
  outside the inputs the system declares.
- **Each worker writes only what it owns.** Its outputs are named
  before the work starts: files for an agent, records for a flow.
  Ownership comes from the system's own record of who wrote what,
  never from what the input says about itself. A write that would
  replace something existing is an explicit replace by its owner,
  never a side effect of a create. Check: the brief or the definition
  lists the outputs, and each write only creates or first compares the
  target's recorded owner with the writer.
- **A need is met in one place.** Code that more than one piece of
  work will use is written before that work starts, and work that uses
  another's output runs after it. A worker that starts without the
  history, such as a new session or an agent, does not know what is
  there, so before writing code for a need it finds the code that
  already meets it and uses that. Two ways of meeting one need drift
  apart, and a hold on one does not hold the other. New code for a
  need already met is the owner's decision, shown beside the existing
  code. Check: the inputs of each parallel piece exist before it
  starts; a second way of meeting a need is caught only in review.
- **Every worker has a defined refusal.** Stopping and reporting is a
  valid result. A worker without one, facing a standard it cannot
  meet, changes the test, writes its own copy, or reports done. Check:
  the brief says what a refusal looks like.

### Reference

Every change is compared with something fixed: an older system,
recorded behavior, a definition written first, or the code that
already meets the need. Without it, each change is judged against the
last one.

- **All work is compared with a reference.** Where one exists, compare
  with it. Where none exists, write the definition first and compare
  with that. Where the reference is itself wrong, each known
  difference is the owner's decision, recorded where the comparison
  reads it, so the comparison stays automatic and the difference stays
  visible. Check: the comparison runs before the work is called done.
- **A claim names its hold.** A sentence in a document or a comment
  that says what the system does names the hold that proves it, or
  says it is not held. A claim with neither can drift from the code
  unseen. Naming the hold shows where the proof is; the hold itself is
  proven like any other. Check: a test finds each named hold and fails
  when it is gone; a claim that names nothing is caught only in
  review.
- **A reference is a file, not a plan.** A plan records what was
  intended and why, and plans are superseded as the work changes; a
  test that reads one holds the code to an intention that may since
  have changed. What must hold is kept in a reference file beside what
  it describes, read by the holds, and a change to it is the owner's.
  Check: guard asks before a commit that changes a file in
  `guard.references`; no test reads a plan.
- **A session leaves its reference behind.** Before work is called
  done, its invariants become holds, what must stay fixed is written to
  reference files the holds read, and its decisions and their reasons
  go into the plan. What lives only in the session is lost when it
  ends; what is held is enforced on the next session's first change.
  Check: the session's report lists each decision with the hold that
  enforces it.
- **A decision names its hold, and a hold names its decision.** Each
  decision in a plan names the test or check that enforces it, or says
  it is not held. Each hold names the plan it carries. A decision with
  no hold is a sentence in a record; a hold with no decision cannot be
  judged by a later session, which then drops it. Check: a test finds
  each hold the standard and the checker's pages name, and fails when
  it is gone; whether a plan's decisions each name a hold is caught
  only in review, since nothing reads a plan.
- **A standard is fixed text.** Each choice it allows names its
  condition and who makes it: "nothing changes unless the owner
  decided it" can be checked, and "a difference is shown before
  release" permits the difference. Briefs, handovers and summaries
  quote the standing rules word for word, because a summary keeps
  decisions and loses standards. Check: quoted text matches its
  source; an exception in a standard is caught only in review.

### Proof

A hold is known to work only once it has been seen to fail when its
property breaks and to pass when it does not. Until then it may be a
false hold.

- **A hold fails on its break and passes on its baseline.** Break the
  property it guards, and the check that names that property fails.
  Run it on the unbroken system in the same setting, and it passes.
  Any other result, such as a crash, a different check failing, or no
  result in time, proves nothing and counts as neither. Mutation
  testing does this for code. A hold that runs its own break every
  time stays proven as the code changes; a proof run once goes stale.
  The parser or comparator a hold uses needs the same proof, since one
  that misreads its input makes the hold fail or pass for the wrong
  reason. Check: each hold names its break, the check expected to
  fail, and its baseline run.
- **Each hold reports on its own.** A hold that crashes ends the run
  and hides every hold after it, so each records its failure by name
  and the run continues. Check: a run that ends early names the holds
  it did not run.

### Chokepoints

A hold sits where every change has to pass: a commit, a push, a
release, CI, a deploy, or the receiving side of an interface. Between
chokepoints a worker can do anything; at one, every change is
checked. A hold placed anywhere else can be routed around.

- **A hold states where and when it runs.** A hold that runs on every
  change holds on every change. One that runs only at release, or only
  when someone starts it, holds only then, and a change between is not
  held. A hold that watches one route to a place does not hold another
  route to it. Check: each hold names what starts it and the route it
  watches.
- **Input outside the domain is refused at the receiver, with its
  reason, to whoever sent it.** It is never accepted as empty and never
  turned into something else. A report of "done" without its proof is
  input outside the domain. Check: the refusal reaches the sender with
  its reason intact.

### The owner's attention

What reaches the owner is small enough to decide, and is a decision
only the owner can make. An owner given more than that stops reading
and becomes the error detector.

- **The plan is the unit the owner decides.** The owner approves a
  plan, which carries the decisions. The worker builds it through, each
  commit held by a check, and reports once. It stops only for a
  decision the plan does not cover, a failure it cannot resolve without
  changing a decision, or a change beyond the plan. Check: each commit
  is held by a check, and the report names every stop and why.
- **The owner decides the rule for a choice; the worker applies it.**
  The worker shows each application and explains any that do not
  explain themselves. Check: none automated; caught only in review.

## Cases

- **Separation: a pipeline that replaced records it did not own.**
  Replacing was the design: an updated document replaced the record it
  updated. The trap was trusting the document's content to say which
  record that was. Documents were edited after they were generated,
  and some came from other systems, so a document could name a record
  that belonged to another upload. Each stage met its own contract,
  and the upload reported success. The system's record of who owns a
  target decides, not the document.
- **Separation: a port where the tests were changed to fit.** When a
  ported test failed, the worker changed the test instead of the code,
  and behaviors were deleted without the owner deciding. Later changes
  to tests cited a standing rule from the owner and named each
  decision in the commit message, and every one was still decided by
  the worker. Had the expected values come from the old system, there
  would have been none to edit.
- **Separation: a suite that patched the production code.** Every test
  replaced a production builder at runtime, so the tests measured a
  program that never ran. They were false holds, and their first
  failure was the first true thing they reported. The proof principle
  finds such holds.
- **Proof: a mutation check that counted a crash as a catch.** Each
  case broke the code in one way, and a case read as caught whenever
  the smoke test exited non-zero. One case was caught only because a
  helper threw on a missing file before the assertion for that
  behavior could run, and the output could not tell that from a
  mutant that failed to parse. The crash also ended the run, so every
  check after it went unrun.
- **Separation: a fix that would have added a second way to pace a
  loop.** A speed setting made one game's loop run too fast, and a
  session starting fresh proposed a new rule for paced loops. The
  runtime already met that need: a routine called while another runs
  is part of it, and only the outer step is sped up. Another game
  already kept its timing that way.
- **Reference: this standard, grown one approved rule at a time.** Each
  report from applying the standard to a repository produced a new
  rule. Each was small and approved, and in three commits the rules
  grew from thirteen to sixteen, with groups stretched to hold them.
  Each was judged against the report it came from, not against the
  standard's principles. The principles are now its reference, as the
  section on changing it sets out.
- **Reference: a plan the code rightly left behind.** A plan said every
  declined ask must reach a place. During the build the owner decided
  that a no which changes nothing needs no place, and the code follows
  the owner. The plan, kept unchanged as the record of what was
  approved, now states a rule the code does not follow, and is right to.
  A test that read the plan would have pulled the code back to the rule
  the owner set aside.
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
- **Chokepoints: an import that asked without looking.** The step
  asked the person before putting files on a volume and said a file of
  the same name would be replaced, without reading which files the
  volume held, so the person could not tell what the answer would
  replace. A question is a door only when it names what it lets
  through.
- **Proof: a proof run that replayed.** A mutation run's scratch
  copies kept their files' modification times, and the test runner
  reused a cached pass for changed documents, so five breaks read as
  passes until the run was repeated with the cache off. The runner is
  part of the proof's instrument.

## Limits

Control cannot hold a property nobody named. If the requirements are
wrong, a system under control is reliably wrong. Where the system
meets a service it does not own, the hold rests on an assumption about
that service; a probe that tests the assumption against the real
service makes the assumption visible, and does not make it true.

## Changing this standard

A finding from applying the standard goes to `docs/control-findings.md`,
with where it came from. The standard changes only through a plan that
takes a batch of findings and compares each with the principles. If a
principle covers a finding, it is a case of that principle: it goes in
the cases, or nowhere. A new principle or a new rule is the owner's
decision, shown against the principles it does not fit.

## Control in plankit

plankit's hooks are holds on what an agent does to git. They run
whatever the session was told, so they do not rely on the agent
remembering a rule.

- pk guard holds the developer's git decisions: a git mutation while a
  protected branch is checked out, and any push on any branch, is
  denied or questioned, as `.pk.json` sets. Held by
  `internal/guard/guard_test.go`, "TestBlocksMutationOnProtectedBranch"
  and "TestPushIsReadThroughShellForms".
- pk protect holds the plans directory against the agent's Edit and
  Write tools, so a preserved plan is a record those tools cannot
  rewrite. Held by `internal/protect/protect_test.go`,
  "TestDeniesWritesUnderPlans". A write made through the shell is not
  held.
- The version is computed from the tag and the commits, so there is
  no version for a worker to write. Held by
  `internal/changelog/changelog_test.go`, "TestBumpInference".
- pk brief gives each session the repository's policy, generated from
  `.pk.json`, so the policy arrives word for word rather than as a
  summary. Held by `internal/brief/brief_test.go`,
  "TestTextFollowsTheDials".
- pk guard asks before a commit while a file listed in
  `guard.references` has changes, naming the files, so a reference
  changes only on the developer's word. Held by
  `internal/guard/guard_test.go`, "TestReferenceChangesAsk".

## Applying it

Paste this into a session in the repository:

```
Read https://plankit.com/control and report how this repository stands
against it, rule by rule, with evidence from the code. Change nothing.
```

The page changes only when plankit is released or its site is rebuilt by
hand, so a repository reads a fixed text between releases. Not held: the
release workflow publishes it, and no test checks that.

What to do about each finding is the owner's decision.

## References

- Hamilton, M. H. and Hackler, W. R., "Universal Systems Language for
  Preventative Systems Engineering", *Proceedings CSER 2007*, Stevens
  Institute of Technology, Hoboken, NJ, March 14–16 2007, Paper #40.
  <http://www.htius.com/Articles/36.pdf>
- Hamilton, M. and Zeldin, S., "Higher Order Software: A Methodology
  for Defining Software", *IEEE Transactions on Software Engineering*,
  vol. SE-2, no. 1, March 1976.
