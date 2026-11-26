# A design you can run

This is a design language and an engine that plays it.
It starts from one question: what does an owner need to say about a
system so that a machine can play it before any code exists, and hold
the code to it afterwards?

A design is a model of the system's runs. The engine plays every run
the design allows and reports each one that leaves a question open.

## Who does what

| Who | Does |
|---|---|
| A model | Writes the design, from what the system should do. Writes the code. A different session, reading the code cold, describes what the code does. |
| The owner | The developer who decides what the system does. Decides the design, and decides each question the engine raises. |
| The engine | Plays the design and the description. It gives the same answer every time, and no model writes it as part of the work it judges. |

The owner decides the design before it is guarded. A design a model
has just written is one reading of its source, and two models read the
same pages differently. Once the owner has decided it, the design is
the fixed thing the code is compared with, and a model does not change
it without the owner's say.

Four things judge the work: the design, its acceptances, the tests the
design names, and the engine. A worker that can change any one of them
is marking its own work. Each is kept out of the worker's reach in the
way that asks the owner least.

| What judges | How it is kept | Why that way |
|---|---|---|
| The design | A guard asks the owner before a session commits a change to it | It changes rarely, and each change is the owner's decision |
| Its acceptances | The same guard | The same reason |
| The tests the design names | The engine reports a named test that is no longer in its file | Test files change all the time, mostly by gaining tests. Asking about each change would be noise |
| The engine | A released copy is run from outside the tree being worked on | It changes constantly while it is built. A worker on another system cannot reach it at all |

A guard that asks often has failed. It is ignored first and switched
off second. So a guard asks only where every ask is a real decision for
the owner.

Not caught by any of these: a named test that is made weaker and keeps
its name. A test is trusted once it has been seen to fail when its rule
is broken.

Not held yet. The guard is plankit's `guard.references`, and it would
list `lab/engine/pk.json` and `*.accepted.json`. It is not set, because
no released pk has it and pk refuses a setting it does not know. The
engine is run from the tree, and no check runs it on every change.

## The one idea: facts

Everything that can differ from one run to the next is a fact: a named
thing that is true or false.

- A flag is a fact: `dry run`.
- A state of the world is a fact: `the last commit is a release commit`.
- Something wrong with an input is a fact: `the tree is dirty`.
- Something going wrong at a step is a fact: `origin rejects the push`.
- A person's answer is a fact: the engine makes one for each question.
- Which run it is, is a fact: `in ship`. The engine makes one for each
  run.
- A decision nobody has made yet is a fact: an `open` one.

A fact that means something is wrong is a **fault**. The engine lets one
fault be true in a play, unless the design sets `faults` higher.

A design then says, for each action, what it does about the facts that
matter to it. Everything else follows from that one mechanism:

| What the owner wants to say | How it is said |
|---|---|
| A flag changes what runs | an action, a read or a change runs only `when` a fact holds |
| A step is skipped in one flow | the same `when`, or a choice whose other branch is nothing |
| An action refuses bad input | it `handles` the fault with `refuse` |
| An action fails part way, leaving work behind | it handles the fault with `fail`, and its changes say what happens `later` |
| A failure only warns | it handles the fault with `warn` |
| A bad value is fine here | it handles the fault with `accept` |
| A person is asked first | it handles the fault with `ask`, and the engine plays yes and no |
| Two flags cannot go together | a fact is possible `only` when another holds |
| Something matters to one run and not another | the fact is possible `only` when `in` that run |
| A step changes the world | a change `makes` a fact true or false for the rest of the run |
| A failure after a write puts it back | the change says `later`: `undone`, or `stands` |
| Several things happen together | they are the changes of one action, which all happen or none do |
| Two checks have no order | they are answers of one action |
| Two parts do not depend on each other | they are parts of an `all`, where a failure in one does not stop the other |
| A failure is dealt with where it is caught | the part that holds the action `handles` it |
| Nobody has decided this yet | an `open` fact, used wherever the answer would matter. The engine plays it both ways and asks |
| Why something is fine | `because`, on the statement that lets it through |
| A rule holds for every run | an expectation, limited by `when` to the plays it is about |
| The code must do the same | the code is written in this language too, and played beside the design |

## The parts of a design

| Part | What it says |
|---|---|
| Place | Something that holds data, and where its value lives: `outside` the system, `lasting`, within one `run`, or a `report` to a person. A `keyed` place holds many records, each under a key. A place can name the part that owns it. |
| Fact | A name, what it `says` in words, whether it is a `fault` or still `open`, the place it is `about`, and what it is possible `only` with. |
| Action | One thing a run does: the places it `reads`, the `changes` it makes, and what it `handles`. |
| Change | A place and how it is changed: `set`, `create`, `replace` or `delete`. It can say what happens `later` if a failure leaves the part it is in, and what facts it `makes` true or false. |
| Answer | An entry of `handles`: `when` a fact holds, `then` one of six things. |
| Sequence, choice, `all` | Parts in order, each needing the ones before; one part chosen by a fact; parts that do not depend on each other. |
| Use | A part is defined once under `parts` and used by name, so a variant never needs a copy. |
| Expectation | Something that must be true of every play it covers. |
| Rule in words | What an action computes. The engine does not play it; a test holds it. |

The six answers:

| `then` | The action | Next | The person |
|---|---|---|---|
| `refuse` | does nothing | the failure goes to its parent | is told |
| `fail` | does its work | the failure goes to its parent | is told |
| `warn` | does nothing | the run goes on | is told |
| `tell` | does its work | the run goes on | is told |
| `accept` | does its work | the run goes on | is not told |
| `ask` | does its work on yes. On no, the failure goes to its parent, or with `"no": "skip"` the run goes on without it | | is asked |

## Failures go to the parent

A failure is a value an action returns to the part that holds it. That
part decides what happens next. Hamilton's axioms give a parent the
answer for its own children; the owner's table of them is
`control/axioms.md`.

| Part | Its children | When a child returns a failure |
|---|---|---|
| `seq` | depend on the ones before | The sequence ends. The children after it do not run. The sequence returns the failure to its own parent. |
| `all` | are independent | The other children still run. Then the `all` returns the failure to its own parent. |
| `choose` | one is chosen | It returns the failure of the child it ran. |

- A `seq`, an `all` or a `choose` can have `handles` of its own, for
  the failures its children return. It answers with `warn` or `tell`,
  and the person is told, or with `accept` and a `because`, and the
  person is not. The failure stops there, and the part counts as
  complete.
- A failure no part answers reaches the run. The run has failed, and
  the person is told at `reports`.
- A change can say what happens `later`, when a failure leaves a part
  that contains the change: `undone`, or `stands` with a `because`.
  Silence is a question.

Use `seq` where a part needs what the one before it produced. Use
`all` where it does not. That choice is what decides whether a failure
stops the part after it.

Where the source gives an order and a failure of one part does not
stop the next, use `seq` and have the earlier part answer its own
failure with `warn`.

A keyed place brings three faults with it, which the engine asks about
whenever an action creates or replaces a record there:

- `<place> already has the record`, on a `create`
- `<place> is given a key twice`, on a `create` that says `"many": true`
- `<place> holds another's record`, on a `replace`

## How the engine reads a design

- **Answers are tried in the order written.** The first whose `when`
  holds is the one that applies. An answer with no `when` always
  applies, as when a person is asked every time.
- **What can go wrong in writing to a place is a fault at the step.**
  A fault `about` a place is about the value an action reads there. A
  full disk, or a store that will not take a write, is a fault with no
  `about`, named by the answer of the action that writes.
- **A fault about a place is carried, and asked where something
  lasting is changed.** What an action reads carries the fault, and so
  does everything made from it. It becomes a question when an action
  changes a lasting place or a report with something that carries it,
  and no action up to then has answered it. So one action can read a
  place and a later one can answer for it, as long as nothing lasting
  is changed in between. An answer stands for the rest of the run.
  Where a bad value does not matter to a run, an action that reads it
  says `accept`.
- **Once a run changes a place, the faults about it are false for the
  rest of the run.** The value is then the run's own.
- **A fault needs no `about`.** A fault with none is something that
  goes wrong at a step. It can say the action it happens `at`, or a
  list of actions, and the engine then asks each of them about it. Use
  `at` where the source names a step and does not say what happens
  when it fails. Without `at`, the fault matters only to the actions
  whose answers name it.
- **`only` takes anything a `when` takes.**
- **`create` and `replace` are for keyed places.** A fetch or a copy
  that cannot collide is a `set`.
- **A place the system changes is not `outside`.** The remote a run
  pushes to is `lasting`.
- **A rehearsal is the same action with its changes left out.** Give
  each change a `when` that is false in the rehearsal.
- **Silence is a question, and a guess is not.** Three kinds of
  silence need nothing written: an action that says nothing about a
  fault, a change that says nothing about `later`, and parts put in an
  `all`. The engine asks about each. Anywhere else the language makes
  the writer choose, such as whether an action runs under a flag, and
  the source does not say, use an `open` fact in the `when`. Do not
  pick one.
- **What a design waves through says why.** An `accept` answer and a
  change that `stands` each carry a `because`. The engine asks for one
  that is missing.
- **A question is settled in the design.** There is no list for
  silencing questions. Say the change stands, accept the fault, decide
  the open fact, and give the reason there.
- **A keyed place's three faults need no listing.** An answer names
  them as they are written above, and so can a `when` or an `only`.
  Where one of them is normal for a place, as an existing record is for
  a run made a second time, list that fact yourself without `fault`.
  It is then a state of the world, and does not use up the play's one
  fault. Unlike a listed fault about a place, these stay as they were
  after the run changes the place.
- **A fact a change `makes` starts as any other fact does:** undecided,
  until a play asks for it or a change makes it. `only` looks at what
  was so when the run started, and not at what the run made since.
- **Parts in an `all` are played in the order written.** The engine
  asks about their order only where they touch a place one of them
  changes.
- **A change that was undone counts as no change** for `never changes`
  and for `unchanged`. Only a lasting place can be left behind.
- **`fail` leaves the action's own changes in place,** so each of them
  says what happens `later`, like any other change.
- **A part's `name` is what a finding calls it.** A part with none is
  called by the actions in it.
- **An action reads or changes at least one place.** The engine refuses
  one that does neither.
- **A fault that matters under one flag, or in one run, says so with
  `only`.** A fault about a place is otherwise asked of every run that
  reads the place.
- **An earlier action can answer for a later one.** That holds for a
  keyed place's own faults too: a check before a `create` can answer
  `<place> already has the record`.
- **An expectation covers a play when its `when` can hold there.** A
  flag the play never asked about made no difference to it, so the play
  counts. A fault counts only where some action met it.
- **`on` limits an expectation by how the play ended:** `success` for a
  run that completed, with or without a warning; `failed` for one a
  failure reached; `declined` for one a person's no ended.

## What the engine does

It plays each run once for every choice of facts that makes a
difference. A fact is decided only when a play first asks for it, so a
flag makes no extra plays of a run that never looks at it. An action
asks for the faults about each place it reads, and for the facts its
answers and its `when` name.

For each play the engine knows which actions ran, which were skipped,
which failure reached the run, what was undone, the state of every place at the
end, and which values rest on something wrong that nothing answered.

Each finding is one question for the owner.

| Finding | The question |
|---|---|
| `open` | The design uses a fact nobody has decided. What is so? The finding says what runs only one way. |
| `expect` | A play breaks something the design says must hold. Which is wrong? |
| `unanswered` | Something lasting is changed while a fault that no answer covers is carried into it, or a fault goes wrong at a step and nothing there answers it. What should happen? The finding names the action, and what the run then changes on the strength of it. |
| `left-behind` | A failure left a part after a lasting place was changed in it. Is the change undone, or does it stand? |
| `reads-unwritten` | An action reads a value nothing has written by then. Was its writer skipped? |
| `order` | Two parts in no stated order touch a place one of them changes. Which runs first? |
| `owner` | A place is changed by a part that does not own it, or comes from outside. |
| `silent` | The design names no place where a person learns of a failure. |
| `unexplained` | The design accepts something wrong, or lets a change stand, and does not say why. |
| `stale` | A difference from the code was accepted, and the code no longer differs that way. Should the acceptance go? |
| `unused` | A fact is listed, and no play asks about it and no expectation or rule names it. Does it change anything? |
| `differs` | The code parts ways with the design. Fix the code, change the design, or accept it with a reason? |

It also lists every statement a test should hold: each answer, each
`later`, each expectation and each rule in words, with whether a test is
named and found.

## Holding code to a design

The code is described in the same language, by someone reading the
code. `--against` plays the two side by side: every play of the design,
and the code's play of the same facts. A finding is the first action
where the two do different things, with the facts every such play
shares. A fact only the code knows is played too, so code that refuses
for a reason the design never names is found.

Each difference is the owner's to decide, and there are three ways to
settle one:

| The owner decides | What is done | What the engine then shows |
|---|---|---|
| The code is wrong | Fix the code, and describe it again | The difference is gone |
| The design is wrong, or says too little | Change the design | The difference is gone |
| Both are right as they are | Accept the difference, with a reason | It is printed with its reason, and is not a question |

A difference belongs to neither file, so the owner accepts one in a
file of its own, beside the code's design and named for it:
`pk-code.accepted.json` beside `pk-code.json`. It is
`{ "accepted": [ { "finding": ..., "because": ... } ] }`, where
`finding` is the difference's subject as the engine prints it. An
accepted difference is printed with its reason and is not a question.
When the code stops differing that way, the acceptance is reported as
`stale`.

## The JSON

A design is one JSON object. The engine refuses a key it does not know
and names it.

| Object | Keys |
|---|---|
| The design | `system`, `reports` (the place where a person learns of a failure), `faults` (how many may be true in one play; 1 when left out), `places`, `facts`, `parts` (an object: name to part), `runs` (names of parts to play), `expects`, `rules` |
| A place | `name`, `kind` (`outside`, `lasting`, `run` or `report`), `keyed` (`true` for a place of records), `owner` (a part) |
| A fact | `name`, `says`, `fault` (`true` when it means something is wrong), `open` (`true` when it is a decision nobody has made), `about` (a place), `at` (an action, or a list of them), `only` (a fact that must hold for this one to be possible) |
| An action | `do` (its name), `when`, `reads`, `changes`, `handles` |
| A read | a place's name, or `{ "place": ..., "when": ... }` |
| A change | a place's name, or `{ "place": ..., "how": ..., "when": ..., "later": ..., "because": ..., "makes": { fact: true or false }, "many": true }`. `how` is `set` when left out, and `because` says why a change that `stands` may |
| An answer | `{ "when": ..., "then": ... }` (`when` left out for one that always applies), with `because` (why, for an `accept`), `tells` (a place or a list of places, when it is not the design's `reports`) and, for `ask`, `no` (`stop` or `skip`) |
| A sequence | `seq` (a list of parts), `name`, `when`, `handles` |
| Parts that do not depend on each other | `all` (a list of parts), `name`, `when`, `handles` |
| A choice | `choose` (a list of `{ "when": ..., "run": part }`), `otherwise` (a part, or `null` for nothing), `name`, `when`, `handles` |
| A part's answer | `{ "when": ..., "then": ... }` where `then` is `warn`, `tell` or `accept`, with `because` and `tells` |
| A use of a part | the part's name as a string, or `{ "use": name, "when": ..., "says": ... }` |
| Where a fact is asked for | a name, `{ "not": f }`, `{ "all": [f, ...] }` or `{ "any": [f, ...] }` |
| An expectation | `name`, `in` (a run), `when`, `on` (`any`, `success`, `failed` or `declined`), and at least one of: `never changes` (places), `never creates` (places), `ends` (place to state), `runs` (actions), `never runs` (actions), or `before` (an action) with `place` and `is` |
| A state | `untouched`, `set`, `created`, `replaced`, `deleted`, `undone`; or `changed` for any of the middle four, and `unchanged` for `untouched` or `undone` |
| A rule in words | `rule`, `about` (an action), `when` (the fact it depends on, when there is one) |

An answer, a change, an expectation and a rule can each name a `hold`:
`{ "file": ..., "test": ... }`, the test that holds it. Any part can
carry `says`, a sentence it stands for.

An action is known by its `do`. That is the name an expectation uses in
`before`, `runs` and `never runs`.

A play that ends with a warning counts as a success.

## What it found on pk

`pk.json` is pk's changelog, release and ship in this form, written
from the 21 statements the checker's definition of pk keeps in words.
Each statement in the design carries the sentence it stands for. Held
by `lab/engine/test_pk.py`.

- Of the 21 statements, 16 are played by the engine and 5 stay words.
- The design has 364 plays: 79 of changelog, 52 of release, 233 of ship.
- It reports 8 questions:
  - "changelog with --dry-run writes nothing" is broken by fetching
    tags.
  - Five faults that some action goes on with, each with what the run
    then changes on the strength of it.
  - Two asking whether what a fetch brought stands when a later action
    stops the run.
- `pk-code.json` is that design changed in one place to what an
  earlier description of pk's code says the code does. Played against
  the design it gives 2 `differs` findings, each naming the facts that
  cause it.

## What it does not do yet

- Define a fact from other facts. A mode that depends on two flags and
  a state has to be written out wherever it is used.
- Say that an action tries again, or that an open fact has more than
  two outcomes.
- Play a run record by record. A fact holds for the whole run, so an
  import where some names are already there and some are not is one
  play or the other.
- Tell a question put to a person from a failure they are told of. Both
  reach the place the design `reports` to.
- Say that a run stops and the person is not told, or that a refusal
  still does something first, such as reading a list again.
- Say that a person is offered three things. `ask` has yes and no, so
  a third choice is a fact of its own.
- Say exit codes, what a message contains, the value an option was
  given, or that two names are one place.
- Ask about a failure nobody listed. Where a source names no way for a
  step to fail, the design has no fault for it and the engine has
  nothing to ask.
- Limit `later` to failures after a named action. A change is undone by
  any failure that leaves a part it is in, or stands through all of
  them.
- Play two runs at once, or anything else about time beyond the order
  of a sequence.
- Say that a value is bad in a way that depends on how it was computed.
  Rules in words carry that.
- Draw anything. With `--json` the engine prints every play, which is
  what the lab's views would read.

## What comes next, and how each would fit

Each of these came from a blind test. None is built. Each is written
here as a use of facts, so that it is designed in when it is built.

- **A fact defined from other facts.** `"is": <facts>` on a fact, in
  place of being decided. A mode that depends on two flags and a state
  is then named once. The engine must work such a fact out wherever it
  now asks for one, in a play and when it checks `only`, and must never
  decide it.
- **An open fact with more than two outcomes.** An open fact that
  lists its `outcomes`, played once for each.
- **An action that tries again.** An answer `retry`, with the fact
  that says whether the second try fails too. The engine plays both.
- **A failure during recovery.** A change's undo can itself fail. With
  `faults` set to 2 the engine already plays two things going wrong,
  and an undo has no fault of its own to name yet.
- **Comparing by what a person sees.** `--against` reports the first
  action where two plays part, so a reader who cuts the code into
  different actions makes differences that are not real. Comparing how
  the run ends and what each place holds first, and naming the action
  only to explain a difference, would remove them.
- **Evidence for a description of the code.** Each statement naming
  the function where the code does it, checked to exist, as the checker
  in `control/` does.
- **The reference held to the engine.** A test that fails when the
  table of keys on this page and the keys the engine accepts differ.

## Its home

plankit is the home for this, for three reasons.

- The guard is plankit's. A design works only if a model cannot change
  what judges it, and pk's guard is what keeps it so.
- plankit is already installed wherever the work is done. The engine
  has to run as a released copy from outside the tree being worked on,
  and plankit is how a released copy reaches every machine and session.
- It is the same idea. plankit keeps a plan fixed once it is approved.
  This keeps a design fixed once the owner has decided it.

It comes in by stages, the lightest first, so that nothing ships before
it has settled.

1. This page as a skill. It is written for a model reading it cold,
   which is what a skill is.
2. The engine as a `pk` command. plankit's main module is Go with no
   dependencies, and the engine is Python, so this is a port. The
   engine's tests are its specification.
3. The lab last. It is the part still changing most.

To decide before any of it ships:

- There are two checkers. The one in `control/` is Go and checks a
  definition file. This engine is Python and plays a design. Either the
  engine replaces the checker, or it takes on the two things the
  checker has and it lacks: evidence for each statement about the code,
  and a reference held to the code by tests.
- The guard's reference files have to be released first, since a design
  cannot be guarded without them.

The first real run is plankit's own next change: design it, have a
session build it, have another describe the code cold, and settle each
difference one of the three ways. Nine blind tests showed that a model
can write in the language. None showed what the owner's side is like.

## Running it

```
cd lab/engine
python3 engine.py pk.json --root ../..                      # plays it and prints each question
python3 engine.py pk.json --root ../.. --against pk-code.json   # and holds the code to it
python3 engine.py pk.json --root ../.. --lines              # one line for each finding
python3 -m unittest test_engine test_pk                     # the engine's tests, and what this page says of pk
```
