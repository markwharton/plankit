# An engine for the design language's next form

## Context

The hand test of the design language's next form is done, and its
report is in `_sketch-language/REPORT.md`. All five statements held on
paper. Nothing was played, because no engine plays that form:
`lab/engine/engine.py` refuses it at the first key it does not know.
So the definitions written for the test, pk on git and the two outside
definitions, cannot be opened in the lab.

This plan changes `lab/engine/engine.py` to play the new form, so that
a definition standing on another can be played, stepped through and
drawn in the lab like any design today.

`_sketch-language/LANGUAGE.md` is the specification. This plan does
not restate it. Each commit below builds one part of that page, and
moves that part of the page into `lab/engine/DESIGN.md` in the same
commit, so the tracked page and the engine never disagree.

## How it is built

Five decisions shape the code. Each is here with its reason.

- **In place.** `engine.py` and `DESIGN.md` are changed where they
  are, on a new branch from `runnable-design`. A second engine or a
  second page would each claim to be the language.
- **A failure is returned, not raised.** Today `Play.exec` raises
  `Stop` and the run ends. It will return the failure to the part that
  called it. A `seq` ends and passes it up, an `all` runs its other
  children first, and a part's own `handles` can answer it. `later`
  is applied to the changes inside each part the failure leaves. This
  is what makes `warn`-and-made-fact workarounds unnecessary.
- **A fault is carried, and asked at the lasting change.** `Play.rests`
  already tracks what each place's value rests on. An unanswered fault
  will be carried through what is made from it and become a question
  when an action changes a lasting place or a report with it. A later
  action can then answer for an earlier read.
- **What a place brings is made of facts the engine already has.** The
  five brought facts are built-in facts, as the three keyed ones are
  today. "Kept nowhere" and "held elsewhere" are kept current through
  `Now.made`: a change `from` a place marks it copied, and a change to
  a place marks it not. A member's brought fact gets an `only` on its
  container's, so one answer about the working tree covers the files
  in it, through the `only` rule that exists.
- **A layer is loaded, not copied.** Loading a definition loads the
  ones it is `on` into one set of names and refuses a name used twice.
  An action that `uses` a run plays that run's part, with the bound
  places in place of the `given` ones. The same play loop runs both
  layers, and nothing of the lower definition is restated.

## Commits

Each is test-first: the tests are written, seen to fail, then made to
pass. `make test` runs before and after each. Nothing is pushed.

1. **Failures return to the parent.** `Play.exec` and `Play.act` in
   `engine.py`; `handles` on `seq`, `all` and `choose`; `later` applied
   as a failure leaves each part; the outcome `failed` and `"on":
   "failed"` in place of `refused` and `stopped`. The strings that
   name outcomes in `lab/engine/view.py`, `lab/lab.html` and the
   design files change with it.
2. **A fault is asked at the lasting change.** The carried set; `at`
   on a fault; `tells` as a list; `says` on a use. `unanswered`
   findings name the action that makes the lasting change.
3. **Shapes, operations, and what a place brings.** `is` (`one`,
   `many`, `series`) and `in` on a place; `create`, `replace` and
   `destroy`; `from` on a change; the five brought facts. `keyed` and
   `set` are removed. `lab/engine/convert.py`, a kept script with a
   test, converts the six design files that are not rewritten by hand:
   a keyed place becomes `many` and any other `one`; `set` becomes
   `create` on a run or report place and `replace` on a lasting one.
   The lab's vocabulary in `view.py` and `lab.html` changes with it.
4. **Layers.** `on`, `uses`, `with`, `given`, `rehearsal`, and `makes`
   and `later` on an action that uses a run. `git.json` and `pk.json`
   from the sketch become `lab/engine/git.json` and
   `lab/engine/pk.json`. `lab/engine/pk-code.json` is that `pk.json`
   with the one difference it has from the design today. `lab/lab.py`
   passes the definition's directory, so `on` paths resolve.
5. **A run starts from what earlier runs left.** A fact that changes
   make is false at the start of a run unless some run of the model
   can make it true. For each lasting place, the faults a run accepted
   or told and left it resting on are found by playing the runs until
   nothing new appears; another run that reads the place is asked
   about each.
6. **The page.** What remains of `DESIGN.md`: what the engine found on
   pk, with every number from a command, and what it does not do yet,
   which is the list of eleven in the sketch report.

## The owner's read before commit 4

`lab/engine/pk.json` is a design the owner has decided. Commit 4
replaces it with a new reading, on git. Before that commit is made,
the work stops and the owner reads `git.json` and `pk.json`. The
engine's findings on them are shown beside the hand trace in the
sketch report, which expects seven of today's eight questions still
raised, one new question about a tag origin already has, and the bad
tag now asked where tags were fetched. Each difference between the
engine and the hand trace is explained before `test_pk.py` is given
its new counts. A count is never taken from the engine's output alone.

## What stops a second copy

- **The page.** `DESIGN.md` is the one tracked page. The sketch page is
  its source during the build and is not tracked.
- **The play loop.** `Play.exec` plays both layers. There is no second
  walker for a lower definition.
- **Conditions.** `holds`, `assume` and `names_in` stay the only code
  that reads a `when`, an `only`, or an answer. Binding a `given`
  place renames a brought fact through the table that made it, never
  by matching text.
- **Brought facts.** One table gives each its name, the operation that
  brings it and its sentence. `engine.py` builds the facts from it and
  `DESIGN.md`'s table is checked against it by a test.
- **Vocabulary.** `KEYS`, the shapes, the operations, the answers and
  the outcomes are each declared once in `engine.py`. `view.py` reads
  them from there. A test fails when `DESIGN.md`'s table of keys and
  `KEYS` differ.
- **The conversion.** `convert.py` holds the rule for an old design
  once, with its test. No design file is converted by hand twice.
- **git's facts.** Stated in `git.json`. `pk.json` names a run of
  git's and repeats nothing about it.
- **What comes up.** `_sketch-language/check.py` lists it statically.
  Its figures for pk, 28 rows and 2 not answered, are a cross-check on
  commit 4. After that the engine reports it, and `check.py` stays in
  the sketch.

## Not in this plan

- The eleven things the new form still cannot say. They are listed in
  `DESIGN.md` by commit 6 and built by later plans.
- A blind test of the finished page with the engine, by the method in
  `lab/engine/BLIND-TEST.md`. It is the next step after this plan, and
  it takes the place of a third cold read.
- New views in the lab. The five views keep working on the new form.
  Drawing layers is later work.
- The checker in `control/`, the Go port, and `guard.references`.
- Anything from the two outside definitions. No file, name or path of
  theirs enters the repository.

## Verification

- **Each commit.** `make test`, before and after. New tests in
  `lab/engine/test_engine.py` use small designs written in the test,
  as the file does today, with one case that must fail for each rule:
  a failure in an `all` that stops a sibling, a fault answered after
  the lasting change, a replace of a lasting place with no answer for
  what is lost, a failure from below that nothing answers.
- **pk, after commit 4.** From `lab/engine`:
  `python3 engine.py pk.json --root ../..` and
  `python3 engine.py pk.json --root ../.. --against pk-code.json`.
  Then one rule is broken in a scratch copy, `undo` no longer refusing
  a commit held elsewhere, and the engine must report it.
- **The lab.** `python3 lab/lab.py` opens every tracked design in the
  five views, and `make lab-e2e` passes.
- **The outside pair.** Played from outside the repository, and opened
  in the lab by path. The hand trace expects the six unmatched
  outcomes raised and none of the seven false questions. This is a
  check by hand and is not in the test suite.

## Done when

- `engine.py` plays `pk.json` on `git.json`, and each finding is
  accounted for against the hand trace.
- Every tracked design loads, and the lab draws each.
- `DESIGN.md` says what the engine does, and the two tests that hold
  the page to the engine pass.
- The two outside definitions play and open in the lab.
- Six commits are on the new branch, each with `make test` passing.
