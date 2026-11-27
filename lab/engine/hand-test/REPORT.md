# Report: the design language's next form, tested by hand

The files of the test are in this directory: `LANGUAGE.md` (the page),
`git.json`, `pk.json` (pk on git), and `check.py`. The two outside
definitions are kept outside the repository; this report carries their
counts only. Nothing was played. `check.py` checks names and lists what
comes up from the layer below. Every other result is a trace by hand.

## Verdicts

| # | Statement | Verdict | The case from pk |
|---|---|---|---|
| 1 | A place has a type | Holds, changed | "The tree is dirty" is `working tree holds what is kept nowhere`. "The commit is pushed, tagged or merged" is `working branch is held elsewhere`. git's reset accepts both, and pk's `undo` refuses both. |
| 2 | A place is made of places | Holds, changed | The changelog file and the version files are `in` git's working tree. `undo` no longer lists them: reset replaces the tree. |
| 3 | A run starts from what earlier runs left | Holds on paper | "The last commit is a release commit" is made by the commit and is no longer a free fact. Needs an engine to test. |
| 4 | A primitive is the top of a definition below | Holds, changed | 9 of the 32 facts `lab/engine/pk.json` lists by hand are 6 faults in git's definition, written once. |
| 5 | A failure goes to the parent | Holds | `ship` is a sequence, so a failed release leaves it and the commit's `later: undone` applies. The outside pipeline is the stronger case, below. |

The changes, each against what the plan said:

- **Statement 1.** Three shapes (`one`, `many`, `series`) and three
  operations (`create`, `replace`, `destroy`). The plan said there
  would be no overwrite. `replace` stays: it is a destroy and a create
  in one step, and it is the operation that brings the loss question.
  A copy is a change that says `from`. Five facts are brought by the
  type; the page's table lists them.
- **Statement 2.** Containment is said by the member, with `in`. A
  higher definition has to be able to put its place in a lower one's,
  and it cannot edit the lower definition. The three structures were
  not needed for places. A place with no `is` is made of the places in
  it, and nothing needed a choice or a sequence of places.
- **Statement 4.** A run below acts on `given` places, and its user
  binds them with `with`. An action that uses a run also carries
  `makes`, `later` and `rehearsal`, since it states no changes.

## pk: where each hand-listed fact went

`lab/engine/pk.json` lists 32 facts. In `pk.json` here:

| Where | Facts |
|---|---|
| Still listed in pk | 19 |
| Made by an earlier run, and no longer free | 1 |
| Brought by a type | 3 |
| Listed once in git's definition | 9, as 6 faults |

pk's definition lists 20 facts and git's lists 8, by `check.py`.

## pk: the five guards

| Guard | Where it comes from now |
|---|---|
| The tree is dirty | A type: `holds what is kept nowhere`, brought by a replace |
| The commit is pushed, tagged or merged | A type: `is held elsewhere`, brought by a destroy |
| The release branch cannot be fast-forwarded | git, listed by hand once |
| The release branch has diverged | The same fault of git's, answered early by pre-flight |
| Origin rejects the push | git, listed by hand once |

Two of five come from a type. A rule for a `series`, that adding from
another series needs the other to contain its last member, would bring
the other three. It is not on the page.

## pk: the 8 questions the engine raises today

| Today's question | By hand, in the new form |
|---|---|
| `changelog --dry-run` writes nothing, broken by `fetch tags` | Still raised |
| A bad tag at `infer version` and `undo` | Still raised at `infer version`. Gone at `undo`, which no longer reads tags. Now also asked in the plays where tags were fetched |
| Origin does not answer, at `undo` | Gone. `undo` no longer reads origin; the type's fact replaces the check |
| Invalid settings at `build section` | Still raised |
| The branch is not the default, at `infer version` | Still raised |
| The working branch is unfit, at `infer version` | Still raised |
| What the pre-flight fetch brought, when the run stops later | Still raised |
| What `fetch tags` brought, when the run stops later | Still raised |

## pk: what comes up from git

By `check.py pk.json`: 28 rows come up to pk's actions across the three
runs. 20 are answered at the action, 6 by an earlier action, and 2 are
not answered.

New questions:

- **Origin already has the tag.** git's atomic push refuses it, and
  neither push action answers it. It is one question, at two actions.
- **A bad tag among fetched tags.** Today 28 plays never ask.
- **Two open facts in git's definition**, where the part of git's
  manual read does not say: whether a rejected tag update fails the
  fetch, and whether a fast-forward refuses a tree with changes it
  would overwrite.

Five faults git accepted or told came up and were answered again by
pk. Two are real guards (`undo`). One is a real reason (the fetch runs
only with no local tags). One repeats a reason pk's design already had
(the pre-flight fetch). One is noise: git's "a remote holds one tag of
each name" says a thing cannot happen, and pk had to repeat it.

## The outside pair, counts only

- 6 of the 6 outcomes the hand join found unmatched are raised with no
  hand join. One more is raised: a failure that cannot arise for this
  user of the run.
- 7 of the 7 false questions are gone. The pipeline is an `all` of two
  legs, each a `seq`, and needs no made facts and no start action.
- Of the 6 real questions, 5 are still raised. The sixth, what a
  skipped activity does to a link that only serialises, is no longer a
  question about the definition. The definition says the two legs are
  independent. Whether the pipeline does that is a question for holding
  the pipeline to it.
- 10 of the 12 unsayable things can be said. Not sayable: time, and one
  member of a `many` apart from the others. One of the 10, a check made
  by a later action than the first that reads the place, became sayable
  with a fix from the first cold read. Another, a control that acts in
  another run, is sayable by statement 3 and was not written.

## What the language still lacks

Found while writing the four definitions:

1. A binding that depends on a fact. pk's push is two actions, one for
   each flow, with the same answers.
2. An `accept` that says a thing cannot happen. It comes up like a
   tolerance and is asked again. The same holds for a failure that
   cannot arise for one user of a run.
3. A fact listed by hand about a `given` place. "The branch cannot be
   fast-forwarded" does not say which branch once two are bound.
4. A refusal that only the user of the run learns of. `refuse` says a
   person is told, and a lower layer has no person.
5. An action that uses a run and also writes. It had to be two
   actions: call, then land.
6. Parts that run one after another and do not depend on each other.
   `all` says no dependence and loses the order.
7. A `many` that is empty. "No local tags" is still listed by hand.
8. A change undone only by a failure after a named action. pk's page
   says a failure after tagging rolls back; `later` covers every
   failure that leaves the part.
9. A step whose place in the order the source does not say.
10. Two names that are one place. git's definition here has no place
    for which branch is checked out, and pk's working branch, default
    branch and checked-out branch can be the same one.
11. A fault below that joins two things. git's "origin does not answer"
    here covers a remote that cannot be reached and one that lacks
    what is asked for, and a user of the run can only answer both.

Items 8 to 11 came from the cold reads.

## Whether `control/` already holds each statement

| # | In `control/` |
|---|---|
| 1 | Partly. `control/axioms.md` has the two type-map rows, held by checks. There is no type field. |
| 2 | No. Elements nest; places do not. |
| 3 | Not looked at. |
| 4 | Not looked at. |
| 5 | Yes. `"to": "parent"` passes a refusal up, in `control/fields.md`. |

## What an engine needs to play this

- Load the definitions a definition is `on`, with one set of names, and
  refuse a name used twice.
- For each place: its shape, whether its state has been copied since
  it changed, and which places hold a copy.
- The brought facts, named for bound places.
- Failures returned through `seq`, `all` and `choose`, and a part's
  own `handles`.
- For each run and each lasting place, the ways the run can leave it,
  fed to the runs that read it until nothing new appears.
- Faults a run accepted or told, carried with the state it left.
- `rehearsal`.

What changes with it, from a survey of the tree: `lab/engine/view.py`
(place kinds, `how`, `then`, play statuses), the `THEN` map in
`lab/lab.html`, `lab/engine/test_engine.py`, the counts asserted in
`lab/engine/test_pk.py`, and 8 design files to write again.

## What this test did not do

- It played nothing. Statement 3 in particular is a rule for an engine,
  and a trace by hand cannot show it holds.
- git's behaviour was read from git's manual for six operations and
  from pk's code for the commands it runs. It was not run.
- `check.py` is static. It does not know when a `when` rules a fact
  out.

## The cold read

Two rounds were run, and the cold read has not ended. Each round was a
new session with the page, `git.json`, pk's page for `release` and
`check.py`, and nothing else. Each wrote `release.json`; both are in
`cold/`.

| Round | What the reader reported about the page | What changed on the page |
|---|---|---|
| 1 | 5 points it was unsure of or had to guess | A step fault can say the action it happens `at`. A fault about a place becomes a question when something lasting is changed on the strength of it, so a later action can answer for an earlier read. A fact whose maker is not modelled is listed. Expectations, states, `only` and `owner` are defined, with their JSON rows. A rehearsed action counts as run. |
| 2 | 9 points it was unsure of and 8 guesses. None of round 1's came back | What each answer means for something that comes up, and for a part. An order with no dependence. `when` against `rehearsal`. What `undone` says. `says` on every object. `at` as a list. Six smaller points stated. |

The round 2 fixes have not been read by a new session. The plan says
to repeat until a read reports no guess the page should have settled.
I stopped at two because each round costs about as much as the rest of
the test together. A third round is the next step if the page is to be
taken further.

Both readers, unprompted, wrote "clean tree" as `working tree holds
what is kept nowhere` and "not diverged" as git's `the branch cannot be
fast-forwarded`. Both left unanswered what pk's page does not answer,
9 of 24 in round 2.

One finding is about pk's own page, not the language. Round 1 reported
that `skills/release/SKILL.md` says `release.branch` defaults to the
branch `pk init` ran on, and also that empty selects trunk flow, and
does not say which applies when the key is absent. I read the two
lines and they say that. I did not check the code.

## `check.py`

It is kept. It sits in this git-ignored directory, so it is on disk
and not tracked. If it is to be used again, it needs a tracked home;
that is the owner's decision.
