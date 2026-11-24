# What pk's definition leaves in words

## Context

On `definition/pk.json` and its two description files, `control check`
prints 51 findings, and 38 of them are `unsaid`: things the code does
that the fields of the format cannot state. The definition itself keeps
21 of its rules as words in `unsaid` and states 19 in fields.

Before any field is added to the format, this study finds out what
kinds of thing are being left in words, and whether the format as it is
could state them. Its result is evidence for one decision: the format
needs nothing, an extension, or a different shape.

The earlier proposal, a `covers` field on `rejects`, is set aside until
then. It is one case of one kind below.

## What was counted

Done already, by reading. Nothing was changed.

The three files hold 59 different `unsaid` entries: 21 in the
definition, and 38 more that only the description files have. Each was
put in one kind.

| Kind | Entries | What it says | An example |
|---|---|---|---|
| Variant | 17 | A flag, a flow or a state changes what the run does | in trunk flow there is no merge |
| Computation | 13 | How an output is worked out from its inputs | a feat bumps minor |
| Recovery | 11 | What is undone, kept or only warned about after a failure | a failure after tagging rolls back |
| Order | 6 | When one thing happens relative to another | preRelease runs before the tag exists |
| Validity | 6 | A further reason for refusing, or when a refusal does not apply | --undo refuses a commit that is pushed |
| Interface | 5 | Exit codes, output streams, how a hook is run | a refusal exits 2 |
| End state | 1 | What is true when the run ends | a release ends on the working branch |

Three facts stand beside the table:

1. `definition/pk.json` has 17 leaves and 2 joins. It has no `or` and
   no `include`. The format's way to state a variant is `or`, and the
   largest kind of entry is the variant.
2. The sweep already holds evidence about recovery. Of the 20 faults of
   pk that end in a refusal, 7 end with a lasting place already
   written. In 4 of them the place is `version files`, which is the
   case the `unsaid` rule "nothing is committed and the tree goes back"
   is about. The checker reports none of the 7.
3. This is one system. The three trials and the baseline have one
   `unsaid` entry each.

## The two questions

- **Variants.** Can `or` state pk's variants at a cost a person would
  pay? If not, what exactly stops it? Two candidates to confirm or rule
  out: every element has exactly one parent, so a variant has to repeat
  its leaves under new names; and a branch of an `or` cannot be
  "nothing", so a step that is skipped cannot be said.
- **Recovery.** Is "a run refuses after a lasting place was written" a
  check the existing traces support, and how many of its reports are
  real?

The other kinds are sorted by reading in the report. Computation is
what a leaf's own tests hold. Order is the timing the standard already
says the checker does not cover.

## The experiment

1. **Variants.** Restate pk's definition as scenario files, each one
   stating a single variant with `or`: trunk flow, `--undo`, nothing to
   release, and `--dry-run`. The files live in the git-ignored sketch
   directory and are checked with the repository as their root. For
   each, record: elements added, `unsaid` entries that became fields,
   findings added and gone, the paths the sweep plays, and anything
   the checker refuses, with its reason.
2. **Recovery.** A throwaway script reads the model `control view`
   prints and lists every fault that ends in a refusal with a lasting
   place written, for pk and the three trials. Each row is read
   against the design and marked real, harmless, or wrong.
3. **Report.** One table for each question, then each of the seven
   kinds with a recommendation: it stays words, today's format can
   state it, it needs an extension that is named exactly, or it belongs
   to another tool. The counts and the result are added as one finding
   to `docs/control-findings.md`, the file where findings wait for a
   plan.

## Not changed

The checker's code, its pages, `definition/pk.json`, the trials and the
baseline. This plan proposes no change to the format. Any change it
points to is a later plan.

## Done when

- Each kind has a recommendation, and each number in the report comes
  from a command run for the report.
- Each scenario file is one the checker accepts, or the report shows
  the reason it gave for refusing it. A restatement the format cannot
  express is a result, and is shown with the checker's own words.
- `make test` passes before and after, since nothing it covers changes.

No commit is made. The one tracked change, the finding, is left for the
owner to read.
