# The blind test

The way to find out whether the language can say what a system needs,
and whether `DESIGN.md` is enough to write in it: give a new session
only the reference, the engine and a system's own pages, and have it
write the design cold.

Six rounds of this shaped the language. Each found gaps that the
writer of the language could not see.

## Setting it up

1. Make an empty directory. Copy `engine.py` into it, and `DESIGN.md`
   as `LANGUAGE.md` with the section "What it found on pk" taken out
   and its examples changed to ones that are not about the system under
   test. The page must give away no answer. Where the system stands on
   another, copy that design in too: it is the layer below, and the
   writer may read it.
2. Start a new session with the prompt below.
3. Run the engine on the file it wrote yourself, and check the numbers
   it reported.

## The prompt

```
You are testing a new design language cold. You will write a design for
<the system> from its documentation pages only, run an engine on it, and
report. Work only in <the directory> and write nothing anywhere else.

What you may read: LANGUAGE.md, engine.py (ONLY where LANGUAGE.md leaves
you unsure, and note each time what you looked up and why), and <the
pages>. Read nothing else: no source code, no tests, no other designs.

1. Read LANGUAGE.md.
2. From the pages alone, write one design file for <the runs>. State
   what the pages say: the places, the facts that change a run, the
   faults, each action in order with what it does about each fault that
   matters to it, what happens to a change when a failure leaves the
   part it is in, and each thing that must always be true as an
   expectation. Put rules
   about how a value is computed under "rules". Leave "hold" out.
3. Run the engine. Fix the file only until the engine accepts it. Then
   do NOT change the design to make findings go away. The findings are
   the result.

Model the pages faithfully. Do not guess: where the pages are silent,
leave the design silent in the ways LANGUAGE.md says the engine will ask
about, and use an open fact for anything else the language makes you
choose.

Report: the play counts and each finding's kind and subject; for each
finding whether it is (1) a real gap or contradiction in the pages, (2)
something the pages do not say, or (3) an artifact of your modelling;
every point where LANGUAGE.md did not tell you how to write something,
or the language could not say it; exactly what you looked up in
engine.py; every refusal or crash; and counts of what your design has.
```

## Reading the result

- A finding judged (3) points at the language or the reference, and is
  the one to act on first.
- A question that independent writers raise again and again is a real
  gap in the pages.
- If the writer had to read `engine.py`, the reference is missing
  something. Say it there.

`lab/examples/blind/pk-blind.json` is the design the fifth round wrote
for pk's changelog and release.

`pk-on-git-blind.json` beside it is the sixth round's, written for the
same two commands once a design could stand on another. It stands on
`git-at-round-6.json` beside it, a copy of git's design as it was in
that round, when the writer was given it at `lab/engine/git.json`. The
engine accepted it on the first run. Its `on` path is the one thing
changed since it was written, so that it finds that copy.

`pk-on-git-blind-7.json` beside it is the seventh round's, written for
changelog, release and ship once the language had named facts. It
stands on `git-at-code-read.json` beside it, git's design unchanged
since the code was read. Every example about pk in `LANGUAGE.md` was
changed to one about a library's lending desk, the named fact's among
them. The engine accepted the design on the first run: 4,743 plays and
14 questions. The writer looked one thing up in `engine.py`, when an
expectation's `when` is judged, which `DESIGN.md` now says. The round
cost about 132,000 tokens.

The writer named no fact. The conditions of more than one fact it
repeats are `in changelog or in ship`, five times, and `in release or
in ship`, twice. A writer drafting from pages does not yet know which
conditions repeat; pk's design named its three when it was revised.

Its questions are gaps in pk's pages: a preRelease hook that fails,
what pre-flight fetches, whether preRelease runs before the merge,
whether `--undo` makes the branch checks, and a commit or a tag that
cannot be made. Its expectation that changelog creates no tag failed
where changelog fetches origin's tags: the page said "No git tag is
created", and meant the release tag.

`data-pull.json`, `disk-import.json` and `timer-stop.json` beside it are
designs written the same way for one flow each of three other systems.
They are kept because each showed something pk could not: a run made a
second time over its own records, a person who is always asked, and a
person offered three things.

## Describing code blind

The same method holds code to a design. A new session reads the code
and nothing that says what the code should do. It is given the design's
names and no more: the places, the facts and the names of the actions.
`PROMPTS.md` has the prompt. The engine then plays the two with
`--against`.

`lab/examples/blind/pk-code-blind.json` is the description one session
wrote of pk's changelog and release this way. Played against
`lab/engine/pk.json` as it then was, before pk's design stood on git's,
it gave 32 differences. The real ones are in
`docs/control-findings.md`. About a third are noise: the reader split
the work into actions the design does not have, and the engine reports
the first action where two plays part.

`pk-code-on-git-blind.json` beside it is a second description, of
changelog, release and ship, written once pk's design stood on git's.
It stands on `git-at-code-read.json` beside it, a copy of git's design
as the reader was given it. The reader had `LANGUAGE.md` made as for a
design round, git's design, and a file of pk's names: the places, the
facts and the names of the parts and actions. Two facts' descriptions
there were cut back to what the fact is, since each also said what pk
does about it. The reader read pk's Go source and no tests, pages or
history. It did not run `--against`, because that would have shown it
what the design's actions do. The tester ran it, as in a design round.
The round cost about 180,000 tokens.

Played against `lab/engine/pk.json` it gave 32 differences. Seven are
ones already known. Twelve are things the code does that had not been
described, among them ship resuming a release already pending and
release making its checks before it fetches. Three mix both. Ten are
noise: the reader's own actions, and messages it wrote as `tell` that
only report progress. `lab/engine/pk-code.json` was then written again
from what it found, with the design's names and in the code's order.
