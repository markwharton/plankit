# A certificate import, told a piece at a time

This example shows how a design grows when the truth about a system
arrives late. It follows a real system, with its owner left unnamed:
an import that reads aviation certificates (ARC and COC) from PDFs
into a register.

The requirement said the PDFs came from the business system, Business
Central. In practice, users edited them before uploading them, and
some came from other systems. Nobody said so at the start. Each new
kind of PDF came to light when a user noticed a problem, a piece at a
time. Nobody knows the whole truth even now: there may be another
format that nobody has mentioned.

Each stage below is a design in this directory, played by the engine.
To play one, run this from `lab/engine`:

```
python3 engine.py ../examples/certificate-import/1-as-told.design
```

| Stage | Design | Plays | Questions |
|---|---|---|---|
| 1 | `1-as-told.design` | 4 | 2 |
| 2 | `2-the-files-too.design` | 6 | 3 |
| 3 | `3-one-file-many-certificates.design` | 7 | 0 |
| 4 | `4-what-nobody-said.design` | 11 | 3 |
| 5 | `5-what-nobody-knows.design` | 14 | 2 |
| 6 | `6-a-known-wrong-form.design` | 16 | 2 |
| 7 | `7-a-corrected-file.design` | 28 | 3 |

## 1. As first told

A certificate PDF from the business system is uploaded, and the
import creates a record for its certificate number:

```
PLACE pdf ONE OUTSIDE
    SAYS a certificate PDF the business system made, uploaded by a user
PLACE register MANY LASTING
    SAYS one record for each certificate number

ACTION import
    READS pdf
    CREATE register MANY FROM pdf
    REFUSE WHEN the pdf cannot be read
```

The engine asks two questions, though the requirement mentions
neither:
- **The register already has the number.** What does the import do
  then?
- **The same number is given twice in one upload.** What does the
  import do then?

The engine asks both of any create into a place that holds many, so
they need no one to have pictured an edited PDF. Each one has to be
decided before the design has no questions left.

## 2. The file is kept too

The first version also saved each certificate's PDF in a library,
named by its number, before recording it. It skipped a number already
in the register and reported the upload as a success. In the design,
that skip is a decision with its reason:

```
ACTION record it
    READS pdf
    CREATE register MANY FROM pdf
    ACCEPT WHEN register already has the member
        BECAUSE the record is there already, so the import skips it and says it succeeded
```

The engine now asks three questions:
- **The library already has a file for the number.**
- **One upload gives the library the same number twice.**
- **One upload gives the register the same number twice.**

Each of them later happened:
- **In June,** one file reused a number on pages that followed one
  another. Each page was saved under the same name over the last, and
  nine certificates became two.
- **In November,** a new certificate carried a number already
  allocated to another. Its file was saved over the original's, the
  record was skipped, and the upload reported success. Three re-signed
  certificates, uploaded as new files, did the same.

All three questions were open in the first version. Played before the
code was written, the design would have asked them then.

## 3. One file, many certificates

The answers the history reached, in June and November:
- pages that reuse a number one after another are one certificate;
- a number already in the library, or in the register, is held for
  review, not saved over.

The engine asks no questions. That doesn't make the design right. It
means every case the design knows of has an answer, and the design
still assumes every PDF is what the business system made.

## 4. What nobody said

Over the following months, the history learnt more about the PDFs:
- some were scans, with no text to read;
- some had fields edited by hand into a form the import didn't read;
- some came from another system;
- some carried a number that wasn't the certificate's.

Each one is a fault of the PDF. The design adds them with only the
answer the code had then: a scan is refused. The engine asks about the
other three, and says where each one bites: "split it" is the first
action to change something lasting with it unanswered.

The library's file is saved before anything checks the PDF, and a
file saved stands. So a refusal at the register comes after the
library has the file. In November the fix put a check before the
upload, and an audit then found two other ways in that saved files
without it.

## 5. What nobody knows

The last design answers what can be answered:
- **A check comes first.** Nothing is saved until the PDF has passed
  it, and a field edited into a form the import doesn't read is held
  for review there.
- **A form the import doesn't know is refused, with the reason.** It
  is never reported as a success with nothing imported, which the code
  does today.
- **Another system is an open fact.** Nobody has decided what an
  import from another system may do, so the design says that, and the
  engine plays it both ways.

Two questions are left, and they are the ones nobody can answer yet:
- **What may an import from another system do?** 8 of the 14 plays
  depend on it.
- **What happens with a wrong number?** A number already allocated is
  caught at the register. A number that was never used isn't caught
  by anything, because nothing in the PDF shows it is wrong. The owner
  decides: accept it with a reason and a review elsewhere, or check it
  against something outside the PDF.

## 6. A known wrong form, accepted on purpose

Some uploads were known to be wrong and were needed in the register
all the same: a regulator's variant of the form, which the business
system did not make yet, because the proper form had not been agreed.
Rather than let them slip through, the import was written to accept
them, as a known condition to support properly later.

In the design, that is an acceptance with its reason:

```
FAULT the form is a variant the business system does not make yet ABOUT pdf
    SAYS a form known to be wrong, needed in the register all the same

ACTION check it
    ...
    ACCEPT WHEN the form is a variant the business system does not make yet
        BECAUSE the register needs these certificates now, and the proper form is not agreed yet
```

The engine plays the accepted form through every step after the check,
and both of its plays reach the register. It opens no new question:
the case is decided. That is the difference from stage 5's open fact.
There nobody has decided, so the engine plays both ways and asks; here
the owner has decided, and the reason is on record beside the
decision. The import has the wrong form under its control, rather than
the form having the import under its control.

When the proper form is agreed, the acceptance becomes an ordinary
answer, or a refusal of the old form: one change to the design, and the
engine shows every play it affects.

## 7. A corrected file replaces the first

Later, a user could replace a file already imported with a corrected
one, and the import rewrote the record from it. Added as a second run,
with the record rewritten from the file and nothing more, the engine
asked twelve questions. Two kinds stand out:
- **Every fault of the PDF, again.** The replace was a second way in
  that skipped the check. It is the same finding as stage 4's, in a
  new place.
- **"register holds what is kept nowhere".** The engine explains it:
  "its state has changed since it was last copied to another lasting
  place, so replacing it loses that state". A correction made by hand
  in the register is lost when the record is rewritten from the file.
  The history lists this as still open.

The design puts the check on the replace too, and refuses a corrected
file whose number belongs to another certificate. A file replaced in
the library is accepted, because the library keeps each file's earlier
versions: that is how a file overwritten in November was restored.

Three questions are left: the two from stage 5, and the hand
correction. Should the rewrite refuse, hold the record for review, or
keep what was corrected by hand? That is the owner's decision, and the
design will not let it go unmade.

## What it shows

- **A requirement leaves cases unsaid,** and the engine asks about
  them from the first design. Stages 1 and 2 asked, before any code,
  the questions that cost certificates in June and November.
- **Each surprise is a new fact.** The engine says which step meets it
  first with no answer, and so where the check has to go.
- **A format nobody has mentioned is already answered.** A form the
  import doesn't know is refused, so it is held for review, never
  imported wrongly.
- **What isn't known stays named.** It is an open question in the
  design, with the number of plays that depend on it, not a case left
  out.
- **A known exception is a decision, not a gap.** An acceptance with
  its reason is played like any other case, and it stays the owner's.
- **Every way in meets the same questions.** A second run, such as a
  replace, is asked about every fault the first one answered, so a way
  in without the check is found in the design, not in an audit.

These designs are small examples written from the history. They are
not a design of the real system, and they have not been played against
its code.
