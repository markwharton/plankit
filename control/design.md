# Design

Why the checker is built the way it is, what it does not do, and what
has been designed and not built. The format and the checks themselves
are in `format.md`, `fields.md` and `checks.md`.

## Why the checks are structural

USL's guarantee came from its analyser, which refused definitions that
broke Hamilton's axioms. Most of those axioms are structural: one owner
for each output, inputs read for reference only, a rejection for what
is outside the input, and children combined only as dependent,
independent, or chosen. Structural rules are checks on a tree, so the
checker needs no theorem prover and nothing installed. It is Go,
standard library only, proven by its own tests. `control axioms` prints
each of Hamilton's sources with the check for it.

## What the checker looks at

The checks of the first question look at three things.

| Group | Checks | The question |
|---|---|---|
| Writes | `one-owner`, `reads-for-reference`, `create-never-replaces`, `replace-checks-owner`, `key-from-content` | Who may write a place, and does a write check before it replaces something? |
| Refusals | `rejects-declared`, `declines-declared`, `reject-values` | Does each step that reads refuse bad input, and does the refusal reach somewhere? |
| How steps combine | `include-independent`, `join-ordered`, `or-conditions` | Do steps run in an order that works, are independent steps independent, and does every choice have a default? |

Every check is in one group. Held by `control/pages_test.go`,
"TestTheDesignPageGroupsEveryCheck".

A place is anything that holds data: a file store, a table, a variable,
a message shown to a person. The other checks, `held`, `follows` and
`source`, ask whether the design has tests and whether the code matches
it. They look at the same three things.

The checker has nothing to say about whether a calculation gives the
right answer, how a screen looks or behaves, or timing, speed and two
things happening at once. Hamilton's axioms are about how the parts of
a system use each other's data, which is where she found three quarters
of the errors. The checker covers that part.

The Control standard reaches further. Its rules about tests,
references, proof and the owner's attention apply to any work in any
repository. The checker is one tool for one part of the standard.

### Where to point it

Point the checker at the steps where data moves between parts: a save,
an open, an import, an upload. A step that writes nothing lasting gives
it nothing to find, and tests cover that step better.

- Describe the steps where a silent overwrite or a dropped refusal
  would do harm.
- Describe shared code once. Where several parts save through the same
  code, one description file at that layer covers them.
- Keep the definition file, the description files and the acceptances
  files together in one folder.

A description file can be larger than the code it describes. A
repository that follows this advice ends with tens of these files. More
than that suggests the checker is pointed at code it has nothing to say
about.

## What it does not do

- A place belongs to one element, as a whole. A step that replaces only
  the records its own upload made still writes a place that belongs to
  another step, and `one-owner` reports it. Saying which records belong
  to whom is an open question.
- It checks structure and says little about behavior over time: no
  states, retries, or concurrency. Those need a model checker such as
  TLA+ or Quint.
- A store's `onExisting` is a statement about the store. When it is
  `"refuse"` it is a rule and names its hold. When it is `"replace"`
  nothing tests it against the real store; facts, below, are the design
  for doing so.
- Whether a place is lasting is the design's statement. Nothing checks
  it against the code.
- A refusal's `to` is a place the refusing element writes, or its
  parent. Nothing checks that whoever sent the bad input reads the
  place where the refusal lands.
- An element has exactly one parent. A step that several other steps
  call, such as a save reached from two menu commands and from
  quitting, is written once, as a step of its own, and its callers are
  not its parents. What each caller does with that step's refusal is
  not stated in the fields. A refusal that leaves a step arrives at the
  top element, where the design can require that it is reported, with a
  hold. What one caller does differently goes in `unsaid`.
- `invalid` is words. The checker confirms that every place has one and
  every leaf that reads has a `rejects`. It does not confirm that the
  refusal matches the words.
- The conditions of an `or` are words, so the checker cannot confirm
  that they exclude each other. The `otherwise` takes every case they
  leave.
- The fields have no delete, so a step that removes a record goes in
  `unsaid`.
- A person's answer that is not reported changes nothing the checker
  can see. Whether a no should be shown to the person is not checked.
- An acceptance is the owner's decision only where the repository's
  `.pk.json` lists the acceptances files in `guard.references`.
  Elsewhere a worker can write one, and only review sees it.
- A description file can be wrong, as `format.md` says under Limits.
  Conformance, below, is what would compare it with the running code.
- The run that disabled each check to prove its break test is not part
  of the suite.

## Conformance: designed, not built

A description file is written by a worker reading the code. It holds
only as well as its match with the code as it runs. The design for
comparing the two:

1. **A write trace.** The element's own tests run it against its real
   inputs and record each write: the place, where the key came from,
   and the mode. Recording goes through the inputs the system declares
   and does not patch its code, by the standard's rule that control
   does not change the work.
2. **A conformance check.** `control conform <description> <trace>`
   would compare the trace with the description file: every write
   observed is stated, and every write stated is observed at least
   once, or is marked as not covered by the trace.
3. **Store probes.** Each store's behavior is a fact with a probe; see
   the next section.
4. **A chokepoint.** CI runs `check` and `conform` on every change.

The hard part is the first step in each stack: producing a write trace
from Python, PowerShell, or a flow tool without changing the code under
test.

## Facts about services we do not own: sketched, not built

A place states how its store behaves, in `onExisting`, and the checker
reasons from that statement. When the store belongs to someone else,
with no source code to read, the statement is an assumption until
something tests it. If a file says a store refuses an existing key and
the store in fact replaces, the checker passes a write that overwrites
silently: a false hold. A fact is the statement together with the probe
that tests it.

### Rules

- **A fact is about one operation, with its parameters.** "SharePoint
  replaces an existing name" is too broad to be true; one operation,
  called one way, does one thing.
- **The probe is the fact's hold.** It is a short procedure anyone can
  run against their own environment, and it records what it observed.
  A fact with no passing result is not held, however well it is
  evidenced.
- **Results are produced by running the probe and are not written by
  hand.** A worker that writes a passing result has written its own
  judge.
- **A result has a date, and goes stale.** Services change without
  notice, so each fact states how long a result holds. An expired
  result is not held.
- **Evidence is kept apart from proof.** What was observed in the field,
  such as a version history read after an incident, is why the fact
  exists. It is not a result.
- **Facts are shared as probes.** Another person's result is a
  reference that has not been checked; the fact is held in an
  environment only once its probe has passed there.

### Format

A fact is strict JSON, read the same way as the other files. A place
would name its fact with `"store": "<id>"` in place of writing
`onExisting` by hand, and the checker would report a place whose fact
has no current passing result.

```json
{
  "id": "sharepoint-connector/create-file",
  "service": "SharePoint document library",
  "operation": "Create file action of the Power Automate SharePoint connector",
  "parameters": "a file name that already exists in the target folder",
  "behavior": { "onExisting": "replace" },
  "detail": "The existing file gets a new version holding the new content; the earlier content stays in the file's version history.",
  "conditions": ["the library keeps version history"],
  "probe": {
    "steps": [
      "In a test library, create probe.txt containing A.",
      "Create probe.txt again in the same folder, containing B.",
      "Read probe.txt and its version history."
    ],
    "expect": "probe.txt contains B, and its version history holds a version containing A.",
    "baseline": "Creating a name that does not exist makes a new file with one version.",
    "noVerdict": "Any error, or a run that cannot read the version history, proves nothing."
  },
  "holdsFor": "90 days",
  "evidence": [
    {
      "date": "2026-11-20",
      "kind": "observed",
      "what": "Reading the version history of every row in a production library showed four files replaced by later uploads of the same name, each earlier version still present."
    }
  ],
  "results": []
}
```

### The first entry, and what it does not yet know

The entry above is that first fact. Its evidence is strong: the
behavior was observed in production, the earlier versions were used to
restore four files, and another flow relies on the replacement on
purpose. It has no result, because no probe has run, so by these rules
it is evidenced and not held.

Two questions are open in the entry itself:

- **The condition.** The detail depends on the library keeping version
  history. Whether a library with versioning switched off replaces
  without keeping the earlier content, or behaves some other way, has
  not been observed. A second probe run, against such a library, would
  settle it and might split the fact in two.
- **Who runs the probe, and where.** It needs credentials and a test
  library, so it cannot run in ordinary CI. Run by hand, its result
  holds until `holdsFor` expires and then it must run again; run on a
  schedule, the schedule is the probe's chokepoint.

### Open questions

- Where facts live: next to the files that use them, or in a shared
  collection that the files point to.
- How a result is recorded so that it cannot be written by hand: for
  example, the probe runner writes it and signs it, or the result is
  only ever the probe's own output file.
- Which behaviors beyond `onExisting` are worth a fact: whether names
  are compared without regard to case, whether a trigger fires on the
  flow's own writes, whether a write is visible to the next read, and
  whether a retry repeats the action.

## Other formal tools

| Tool | When to reach for it |
|---|---|
| Alloy | Richer structural models than these checks: relations, records that belong to different owners. |
| TLA+, Quint | Behavior over time: states, retries, concurrency. |
| P | Services exchanging messages. |
| Event-B | Refinement from an abstract definition to a concrete one. |
| Dafny, Verus, Lean | Code proven against its specification. |

This checker was built small because the structural rules need nothing
more, and a checker in the repository's own language is one its owner
can read.

## Open questions

- The name: `control` for now.
- Whether it becomes its own product or a plankit command. Neither is
  forced.
- Records that belong to different owners within one place.
- Whether the mutation run that proves the checks moves into the suite.
