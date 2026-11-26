# The design language's next form, on paper and tested by hand

## Context

The design language in `lab/engine/DESIGN.md` was tried, unchanged, on
a flow that crosses two systems of another organisation: a nightly
pipeline, and the API it reads. Both designs are kept outside this
repository. The trial worked, and it showed where the language stops.

- The API's design raises no question alone. Joined by hand to the
  pipeline's design at the places they share, 6 of the 8 ways the API
  can leave a response have no fault for them on the pipeline's side.
  The language has no way to join two designs.
- The pipeline's design raises 13 questions. 7 are false, caused by a
  workaround for a step that fails while the run goes on.
- 12 things could not be said at all.

`lab/engine/pk.json` shows the same gaps in pk's own design:

- It has 14 writes to lasting places. Read against what git does, 7
  add something, 6 overwrite and 1 removes. The design says so for 1.
- In all 28 plays of `changelog` where `fetch tags` runs, the fault "a
  tag does not name a version" is never asked. The engine treats a
  place the run has written as good, and a fetch adds tags from origin.
- Five faults listed by hand each guard a write that would lose
  something: a dirty tree, a commit already pushed, a branch that
  cannot be fast-forwarded or has diverged, a push origin rejects.
- Two facts listed by hand are what another run leaves: "the last
  commit is a release commit", and "the commit is pushed, tagged or
  merged".

Hamilton and Hackler, "Universal Systems Language for Preventative
Systems Engineering" (CSER 2007), read in full for this plan, has each
of these as part of the language. This plan writes the design language
in that form on paper and tests it by hand. It builds no engine. A
language that holds on paper is the input to a second plan, which
builds the engine.

## The five statements

Each is a change to what a definition says. Page numbers are the
paper's.

| # | Statement | Hamilton | What it replaces today |
|---|---|---|---|
| 1 | A place has a type. The type says what can be done to the place and what each operation refuses. | Every type owns its primitive operations (p. 7). "One cannot put a leg on a table where a leg already exists" (p. 11). | `kind`, `keyed`, `how`, and the three faults a keyed place brings |
| 2 | A place is made of places, with the three structures parts already use. | Type maps capture "containment of one object by another" (p. 7). The structures are "used generically in both" maps (p. 6). | The flat `places` list |
| 3 | A run starts from the states earlier runs produced. | Single reference, single assignment (p. 6). A function is started when the object state it depends on exists (p. 7). | Every run starts with each place untouched, and what can be wrong with it is listed by hand |
| 4 | A primitive in one definition is the top of a definition one layer down. | "Each primitive function (or type) can be realized as a top node of a map on a lower (more concrete) layer" (p. 3). | `outside` |
| 5 | A failure is a value returned to the parent, which decides what happens next. | "Reject values exist in all types, allowing the FMap user to recover" (p. 11). Axiom 2: a parent loses control when a child stops before completion (p. 5). | `refuse` and `fail` stop the whole run |

Three consequences are part of the statements:

- **A change makes a new state.** The paper has create, destroy, copy,
  reference and move (p. 11), and no overwrite. A state cannot be
  modified while something still refers to it. So statement 1 has no
  "overwrite" operation: a write that loses what was there is a
  destroy, and the type says what it refuses.
- **There is no cycle to check.** The output of a function cannot be
  its own input (p. 5). `release` reading the commit `changelog` added,
  and `changelog` reading the tags `release` added, is a sequence.
- **What a producer accepted is asked again of the reader.** Axiom 5: a
  parent rejects what is outside "its own, and only its own" input
  (p. 5). An `accept` is one run's decision, and a reader may need
  more.

The paper is an overview. It names the rules for types without giving
them in full. Which types exist, and what each refuses, is designed
here from what pk and the two outside designs need. The page says for
each rule whether it is Hamilton's or ours.

## The hand test

All files are written under `_sketch-language/`, which git ignores.

1. **The page.** Write the language with the five statements, for a
   model reading it cold, in the form `DESIGN.md` has: the idea, the
   parts of a definition, how each is read, the JSON. It keeps facts as
   the one idea, and keeps every finding kind it still needs.
2. **git, one layer down.** Write a definition of the git operations pk
   uses: fetch, commit, reset, switch, fast-forward merge, tag and
   push, over the working tree and its files, a branch's history, tags,
   and origin. Each operation is written once, with its type's
   refusals.
3. **pk on git.** Write `changelog`, `release` and `ship` again, as
   primitives over that layer. Record:
   - each fact `pk.json` lists by hand, and whether it is still listed,
     comes from a type, or comes from an earlier run;
   - each of the five guard faults, and whether it falls out of a type;
   - each of the 8 questions the engine raises today (1 `expect`, 5
     `unanswered`, 2 `left-behind`), traced by hand: still raised,
     explained, or gone;
   - each new question, starting with the tags a fetch adds.
4. **The two outside designs.** Write them again the same way, outside
   the repository. Record counts only: how many of the 6 unmatched
   outcomes are raised with no hand join, how many of the 7 false
   questions are gone, and how many of the 12 unsayable things can now
   be said.
5. **A cold read.** A new session gets the page, git's definition from
   step 2, and pk's own pages for one command, and writes that
   command's definition. It follows the method in
   `lab/engine/BLIND-TEST.md`, with the engine step left out, and
   reports where it had to guess. The page is fixed and the read
   repeated with a new session until a read reports no guess the page
   should have settled. The report gives the number of rounds.
6. **The report.** For each statement: it holds, it needs a named
   change, or it is dropped. The types found, each with its operations
   and refusals. What an engine needs in order to play the language,
   as the input to the second plan.

No play is counted by hand. Steps 3 and 4 test whether each thing can
be said, and trace named cases. The cold read is the check on my own
reading.

## What stops a second copy

- **The language page.** The sketch is not tracked beside `DESIGN.md`.
  The second plan replaces `DESIGN.md` with it; there are never two
  tracked pages that both say what the language is.
- **Hamilton's sources.** `control/axioms.md` is the owner's table of
  them, held by `TestAxiomsPrintsTheOwnersTable`. The page points at
  that table and does not restate the axioms.
- **What the checker already has.** `control/fields.md` passes a
  refusal to the parent with `"to": "parent"`, and `control/axioms.md`
  has the two type-map rows. Statement 5 and statement 1 take their
  meaning from there. The report says, for each statement, whether
  `control/` already holds it. That is evidence for the open decision
  in `DESIGN.md` about the two checkers.
- **git's facts.** They are written once, in git's definition. pk's
  definition names the operation and repeats nothing about it.
- **The outside designs.** Nothing from them enters this repository:
  no file, no name, no path. The report carries their counts only.

## Not changed

`lab/engine/engine.py`, `DESIGN.md`, `pk.json`, `pk-code.json`, the
tests, the lab, `control/`, and the pages under `docs/`. This plan
proposes no engine and no change to a decided design. Each change it
points to is the second plan's.

## Done when

- Each of the five statements has a verdict, with the case from pk that
  supports it.
- Each number in the report comes from a file written for the test or a
  command run for the report.
- Something the new form cannot say is a result, and the report shows
  it.
- The cold read has ended, and the report gives its rounds and what
  each changed on the page.
- `make test` passes before and after, since nothing it covers changes.

No commit is made. The sketch and the report are left for the owner to
read.
