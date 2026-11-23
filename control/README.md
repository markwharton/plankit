# control

A checker for the design of a system and the code that is meant to
follow it. The design is written down in a definition file. What the
code does is written down, one element at a time, in description files,
by an LLM that reads the code. The checker is the deterministic part:
it asks whether the design is sound, whether each of its rules names a
test, and whether the description files match the design, and the owner
reads what it finds.

It is a separate Go module, standard library only, and is not part of
pk.

## How it works

```
 the owner decides                     a worker reads the code
       │                                         │
       ▼                                         ▼
┌─────────────────┐                    ┌──────────────────┐
│ definition file │                    │ description file │
│ what the system │                    │ what the code    │
│ must do         │                    │ does now         │
└────────┬────────┘                    └─────────┬────────┘
         │                                       │
         │  1. is the design sound?              │
         │  2. does each rule name a test?       │
         │  3. do the two files match? ◄─────────┘
         ▼
┌─────────────────────────────────────────────────────────┐
│ the checker: compares the files, looks names up in code │
└────────────────────────────┬────────────────────────────┘
                             ▼
                         findings
                             │
      ┌──────────────┬───────┴──────┬─────────────────┐
      ▼              ▼              ▼                 ▼
 fix the code   write a test   change the      accept it,
                               design          with a reason
```

The checker does three plain things:

- **It does not read code to understand it.** It looks for a name in a
  file: a test's name for a hold, a function's name for a source.
- **It does not judge.** It compares the two files field by field and
  applies fixed rules to the definition file. The same files give the
  same lines every time.
- **It leaves one job to the LLM:** writing down what the code does.
  Every statement the LLM writes points at a function that can be
  opened.

Each finding traces back to a line in one of the two files.

## The pages

| Page | What it says |
|---|---|
| [format.md](format.md) | The two kinds of file, how they differ, and what the checker asks. Read this first. |
| [fields.md](fields.md) | Every field of the files, with an example. |
| [checks.md](checks.md) | How the checker is run, the form of a finding, each check, and accepted findings. |
| [axioms.md](axioms.md) | The owner's table of Hamilton's axioms and the check for each. |
| [design.md](design.md) | Why it is built this way, what it does not do, and what is designed and not built. |

The first three are the reference the checker is built from. The code
is held to them:

- Every key the reader accepts is in a table of `fields.md`, and the
  tables name no other. Held by `control/pages_test.go`,
  "TestTheFieldsPageListsEveryKeyOfTheFormat".
- Every check is named in `checks.md`, in the order the checker prints
  them, and the subjects printed are the ones in its table. Held by
  `control/pages_test.go`, "TestTheChecksPageListsEveryCheck" and
  "TestTheSubjectsAreThePages".
- The example in `fields.md`, read from the page and run, prints the
  line `checks.md` shows, and passes with the acceptance shown there.
  Held by `control/main_test.go`,
  "TestThePagesExamplePrintsTheLineThePageShows".
- `control axioms` prints the table in `axioms.md` row for row. Held by
  `control/sources_test.go`, "TestAxiomsPrintsTheOwnersTable".

## Running it

```
cd control
go run . check --root examples/import examples/import/saved-files.json examples/import/import.json
go run . checks
go run . axioms
```

To run it from another repository, install it once from this
directory: `go install .` puts `control` on the PATH, in Go's bin
directory.

`check` prints one line for each finding on standard output and a
summary on standard error. It exits 0 when there is no finding or the
owner has accepted every one, 2 when a finding is not accepted or a
file was refused, and 1 when the command was given wrongly. Held by
`control/main_test.go`, "TestACleanRunExitsZeroWithNothingOnStdout",
"TestFindingsGoToStdoutAndTheSummaryToStderr",
"TestARefusedFileStopsTheRun" and "TestACommandGivenWronglyExitsOne".

## How the checks are proven

The tests follow the proof principle of the Control standard,
[docs/control.md](../docs/control.md): a hold fails on its break and
passes on its baseline.

- A baseline in `testdata/baseline`, one definition file and two
  description files that use every shape and every field, has no
  finding. Held by `control/check_test.go`,
  "TestBaselinePassesEveryCheck".
- Each check of the first question has at least one break of the
  baseline, and each break gives the findings named for it and no
  others. Held by `control/check_test.go`,
  "TestEverySoundnessCheckFailsOnItsBreak" and
  "TestEverySoundnessCheckHasABreak".
- A rule without its hold, each difference between the two files, and a
  statement without its source are each reported for their own break.
  Held by `control/check_test.go`,
  "TestHeldReportsARuleWithoutItsHold",
  "TestFollowsReportsEachDifference" and
  "TestSourceReportsAStatementWithoutItsSource".
- Every finding is reported, and not only the first. Held by
  `control/check_test.go`, "TestEveryFindingIsReported".
- Each thing the pages say is refused is refused, with a reason that
  names it, and every reason is reported. Held by
  `control/refuse_test.go`, "TestAFileTheFormatDoesNotAllowIsRefused"
  and "TestEveryReasonForRefusingIsReported".
- An accepted finding is printed last and does not fail the run, an
  acceptance that matches no finding is a finding, and an acceptance
  counts only beside the file its finding belongs to. Held by
  `control/main_test.go`,
  "TestAnAcceptedFindingIsPrintedLastAndDoesNotFail",
  "TestAStaleAcceptanceIsAFinding" and
  "TestAnAcceptanceBelongsBesideItsFile".

## The trials

Each directory in `examples/` holds a design, the code as it once was,
and small stand-in files for the tests and functions they name.

| Trial | What it is | What the run finds |
|---|---|---|
| `upload` | A pipeline that files uploaded documents, with a second route the design does not have. | 12 findings: two routes into one store, a key that can repeat, a rule with no test, and the places where the code makes none of the design's checks. |
| `import` | A page that puts the files of a disk image on a volume. | 2 findings: the code asks before replacing without reading the volume, and does not refuse a name given twice. |
| `save` | A game's Save and Save As, with two acceptances files. | 14 findings and 2 accepted: a replace with no check, readers that refuse nothing, a step the design does not have, and an acceptance that no longer matches. |

The lines each run prints are fixed by a test. For `upload` and `save`
they are the lines that new sessions, reading only the three pages,
worked out by hand before the checker was written. Held by
`control/main_test.go`, "TestTheTrialsPrintTheirKnownFindings".

## Writing a description file

A description file says what the code does now. Paste this into a
session in the repository the element belongs to:

```
Read these three pages, in order: format.md, fields.md and checks.md,
in the control directory of plankit.

Write a description file, in the format those pages give, for one
element of this repository: <the step>. Work from the code. Every
statement names its source, and anything the code does that the fields
cannot state goes in `unsaid`. Use the names in the definition file
<path> for places and elements; where there is no definition file yet,
choose the names yourself.

Then report each point where the pages did not tell you how to write
something, and each thing you put in `unsaid`. Change no code.
```

See [design.md](design.md) for what the checker does not do.
