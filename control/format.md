# The format

The checker is a program. It reads two kinds of JSON file about a
software system, a definition file and a description file, and reports
findings. This page says what the two files are, how they differ, and
what the checker does with them. It does not list their fields.

## Words

| Word | Meaning |
|---|---|
| Owner | The person who decides what the system does. |
| Worker | Anything that changes the system: a person or an LLM session. |
| Place | Something that holds data: a file store, a table, a message shown to a person. |
| Element | One step of the system, such as an upload or a save. It reads and writes places. An element can be made of smaller elements, its children. The system itself is the top element. |
| Statement | One thing a file says: that a place exists, which element a place belongs to, that an element reads or writes a place, how an element's children combine, that an element checks something before it writes, that it refuses bad input, or what it does when a person it asked says no. |
| Rule | A statement in the definition file that says the system stops something: a check before a write, a refusal of bad input, what an element does when a person it asked says no, or a store that refuses a second record at a key. |
| Hold | A test meant to fail when a rule is broken. It is named by its file and the test's name. |
| Source | The function where the code does what a statement says. It is named by its file and the function's name. |
| Finding | One line the checker reports, for the owner to decide. |

Owner, worker and hold come from the Control standard,
`docs/control.md`.

Margaret Hamilton led the flight software for Apollo and later designed
the Universal Systems Language. "Definition" is her word: in her
language the code is generated from the definition. Here a worker
writes the code, and the checker compares a description of that code
with the definition, which is a wider use of her word. The three ways
children combine are hers and keep her meaning:

- `join`: the children run in order, each using what the one before
  produced.
- `include`: the children are independent of each other.
- `or`: one child runs, chosen by a condition.

The top element combines its children in a fourth way, which is ours
and is called `steps`. Its children are the steps of the system. They
run at separate times and share the system's places.

## The two files

| | Definition file | Description file |
|---|---|---|
| How many | One for the system | One for each child of the top element; it covers that child and everything below it |
| Says | What the system must do | What the code of that element does now |
| Written by | A worker drafts it; the owner decides it | A worker, usually an LLM session, that reads the code |
| Changes | Only on the owner's word, and before the code does | Whenever the code changes: it is written again from the code |
| Evidence | Each rule names its hold. Statements that are not rules have no evidence in this file; question 3 below ties them to the code | Each statement names its source |

A description file uses the definition file's names for places and
elements, and the checker matches the two files by name. The worker
writing a description file reads the definition file for those names
and for the words of its `unsaid` entries, and for nothing else. Where the code does something the definition file does not name,
the worker gives it a new name, and the comparison reports it.

The checker refuses a file that has a key the format does not have. It
reports what is wrong with the file, makes no checks, and the run fails.

## What the checker asks

1. **Is the design sound?** The checker applies checks taken from
   Hamilton's axioms to the definition file. They say how elements may
   use places: for example, only the element a place belongs to, or an
   element below it, writes that place, and an element does not change
   what it was given to read. `control/axioms.md` lists her axioms and
   the check for each.
2. **Does each rule name its hold?** The checker confirms that the
   hold's file exists and contains the test's name. A rule that names
   no hold is a finding.
3. **Does the code follow the design?** The checker compares each
   description file with the definition file, statement by statement,
   and each difference is a finding. An element that no description
   file covers is a finding. The checker also confirms each source: the
   file exists and contains the function's name.

A run fails when it has a finding the owner has not accepted.

The owner decides what resolves each finding: a change to the code, a
new test, a change to the definition file, or a description file
written again. A difference found by question 3 means the code is at
fault, the description is mistaken, or the design needs to change. The
statement's source names the function to read.

The owner can also accept a finding. Each of the two kinds of file can
have an acceptances file beside it, with the same name ending
`.accepted.json`: `save.accepted.json` beside a description file
`save.json`. A finding made on a description file is accepted beside
that description file, and any other finding beside the definition file.
Each acceptance names the finding by its check and its subject, as the
checker prints them, and gives the reason. The checker reports an
accepted finding as accepted. An acceptance that matches no finding is
itself a finding.

## Six decisions about the files

1. **Each read, write, check and refusal is its own object in the
   JSON.** Each can then carry its own evidence. Without this, a worker
   that has evidence for a refusal and no field for it puts it in the
   field of something else.
2. **Evidence is structured.** A hold is a file and a test's name. A
   source is a file and a function's name. The checker cannot confirm
   evidence written as a sentence.
3. **A rule that names no hold stays a finding until the owner accepts
   it.** That a rule has no test is then on record as the owner's
   decision.
4. **Each file has one list, `unsaid`, for what the format cannot
   state.** In a description file an entry is something the code does,
   and it names its source. In the definition file an entry is
   something the design requires; it is treated as a rule and names its
   hold. The checker confirms the evidence. It reports each entry of a
   description file, unless the definition file states the same thing
   in the same words. Without this list, a worker that finds
   something the format cannot state puts it in a field that means
   something else.
5. **Every place belongs to one element, or comes from outside the
   system.** The element a place belongs to is the one responsible for
   it: only that element and the elements below it write it. Input from outside
   the system, such as a file a person chooses, does not belong to an element.
6. **Only the definition file says which element a place belongs to.**
   A description file says which elements write the place. Which
   element a place belongs to is the owner's decision, and the code
   shows only which elements write it.

## How the definition file stays fixed

- It changes before the code. A description file written from the new
  code must then match the changed definition file.
- Every run checks the whole definition file, so a new element is
  judged against the whole design.
- The definition file is the checker's only account of what the system
  should do. The checker does not read the request a session was given
  or any record of earlier work.
- The checker does not stop anyone changing the definition file.
  plankit, the toolkit this checker sits beside, can: with the file
  listed under `guard.references` in a repository's `.pk.json`, plankit
  asks the owner before an LLM session commits a change to it. A
  person's change is not stopped.

## Limits

- A source shows that the function's name is in the file. It does not
  show what the function does.
- A hold shows that the test's name is in the file. It does not show
  that the test fails when its rule is broken.
- A description file is written by a worker reading code and can be
  wrong.
