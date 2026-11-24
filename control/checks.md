# The checks

This page says how the checker is run, what a finding looks like, what
each check reports, and when a run fails. Read `format.md` and
`fields.md` first.

## Running the checker

```
control check <definition file> [<description file> ...]
```

The checker is given the definition file and any number of description
files. Beside each of them it reads the acceptances file, when there is
one. It is run from the repository's root, so that the paths in holds
and sources resolve; `--root <directory>` names the root when it is run
from somewhere else.

A run does three things, in order:

1. It reads every file. When any file is refused, the checker prints
   each reason on standard error as `<file>: <reason>`, for every
   refused file, makes no checks, and the run fails. A refusal is not a
   finding and cannot be accepted.
2. It makes every check on this page and collects every finding.
3. It marks the findings the owner has accepted.

It prints one line for each finding on standard output: first the
findings not accepted, then the accepted ones. Each group is in this
order:

1. By check, in the order of this page: the checks of the first
   question in their table's order, then `held`, `follows`, `source`,
   `unsaid` and `stale`.
2. For a check made on description files, by description file, in the
   order the files were given. The `follows` finding for a child with
   no description file comes after those of every description file.
3. By subject, comparing character by character by Unicode code
   point. A double quote comes before every letter, a capital before
   every small letter, and a subject that is the start of another
   comes first.

`--format json` prints the same findings as JSON. Each finding there
also names the element and the place its subject is about, in the
fields `element` and `place`, as the last column of the table of
subjects below gives. A field the subject does not name is left out.

| Exit code | When |
|---|---|
| 0 | There is no finding, or the owner has accepted every one. |
| 2 | A finding is not accepted, or a file was refused. |
| 1 | The command was given wrongly. |

## A finding

```
<check>  <subject>: <message>
```

The check's name comes first, then two spaces, the subject, a colon,
and the message. This page fixes the check's name and the subject. The
message is the checker's own wording.

The subject says which statement the finding is about:

| Statement | Subject | Example | Fields in JSON |
|---|---|---|---|
| A place | `place <place>` | `place volume` | `place` |
| An element, of any shape | `element <element>` | `element import` | `element` |
| A read | `<element> reads <place>` | `import reads image` | `element`, `place` |
| A write | `<element> writes <place>` | `import writes volume` | `element`, `place` |
| A check before a write | the write, then `, check <check>` | `import writes volume, check repeated` | `element`, `place` |
| A `rejects` | `<element> rejects` | `import rejects` | `element` |
| A `declines` | `<element> declines` | `import declines` | `element` |
| A branch of an `or` | `<element> branch to <child>` | `save branch to say full` | `element`, the `or` |
| An entry in `unsaid` | its `what`, in double quotes | `"a long name is cut short"` | none |

A check reports one finding for each subject in each file it is made
on. Where several cases have the same subject, the one message names
them all.

An acceptance names a finding by its check and subject, as printed:
`"finding": "held  import declines"`. An accepted finding is printed
with the word `accepted` and two spaces before it, and the owner's
reason as its message:

```
accepted  held  import declines: the dialog is tested by hand
```

## When a file is refused

`fields.md` lists what makes the checker refuse a file as JSON. The
checker also refuses a file for any of these:

- Two places have the same name, two elements have the same name, a
  place is named `parent`, or two entries in one `unsaid` list have the
  same `what`.
- A name is used and the same file does not list it: in `system`,
  `describes`, `belongsTo`, a list of children, `do`, `otherwise`,
  `place`, or `to`. In `to`, the word `"parent"` is not a name.
- An element has more than one shape, or is the child of two elements,
  or is below itself, or is a leaf with no reads and no writes.
- A leaf reads one place twice or writes one place twice, or a write
  has two checks of the same kind, or two branches of an `or` have the
  same child.
- In a definition file, an element is not the top element and is not
  below it. In a description file, an element is not the described
  element and is not below it.
- In a definition file, an element other than the top element has
  `steps`, or the top element has another shape. In a description
  file, any element has `steps`.
- A field has a value its table in `fields.md` does not allow, such as
  `"create"` for a place that is not keyed.
- A `to` names a place, and that place is not written by the element or
  by any element below it. Or, in a definition file, the top element's
  `to` is `"parent"`.

The rules about children come from Hamilton's first axiom: a parent
calls its own children and no others.

## Is the design sound?

These checks are made on the definition file. Each comes from one of
Hamilton's axioms or rules, and `axioms.md` lists which.

In `include-independent` and `join-ordered`, a child counts together
with every element below it: the child reads what they read and writes
what they write. The other checks look at each leaf on its own.

| Check | A finding when | Subject |
|---|---|---|
| `one-owner` | A leaf writes a place that belongs to another element, and the leaf is not below that element. | The write |
| `one-owner` | A leaf writes a place from outside the system. | The write |
| `one-owner` | Two leaves write the same place, and the nearest element above both of them is an `include`, or is a `join` when the place is not lasting. | The place |
| `reads-for-reference` | A leaf reads a place that is not lasting and also writes it. A write whose `mode` is `"replace"` is allowed to, and so is one whose `mode` is `"create"` and has an `exists` check, because each has to read the record it is about to change. | The write |
| `create-never-replaces` | A write whose `mode` is `"create"` goes to a place whose `onExisting` is `"replace"`, and the write has no `exists` check. | The write |
| `create-never-replaces` | A write has an `exists` check, and the leaf does not read the place it writes. | The check |
| `replace-checks-owner` | A write whose `mode` is `"replace"` has no `belongs` check. | The write |
| `key-from-content` | A write has a `key` of `"content"`, can make many records in one run, and has no `repeated` check. | The write |
| `rejects-declared` | A leaf reads a place and has no `rejects`. | The leaf's `rejects` |
| `rejects-declared` | An element's `rejects` has a `to` of `"parent"`, and the parent has no `rejects`. | The parent's `rejects` |
| `declines-declared` | A leaf has a check whose `then` is `"ask"`, and the leaf has no `declines`. | The leaf's `declines` |
| `declines-declared` | An element's `declines` has a `to` of `"parent"`, and the parent has no `declines`. | The parent's `declines` |
| `reject-values` | A place has no `invalid`. | The place |
| `include-independent` | Two children of an `include` write the same place, or one reads a place the other writes. | The `include` element |
| `join-ordered` | A child of a `join` reads a place that is not lasting, which a later child writes and no earlier child writes. | The `join` element |
| `or-conditions` | An `or` has no `otherwise`. | The `or` element |
| `or-conditions` | Two branches of one `or` have the same `when`, letter for letter. | The `or` element |

A lasting place holds what earlier runs left. A part may read it before
a later part writes it, and parts that run in order may each write it.

The steps of the system run at separate times and share its places, so
`include-independent` and `join-ordered` are not made on the children
of the top element. `one-owner` is made on every leaf of the
definition file, whichever step it is in. By its first case, two steps
can both write a place only when the place belongs to the top element.

## Is the design held?

The check `held` is made on every rule in the definition file: each
check before a write, each `rejects`, each `declines`, each place whose
`onExisting` is `"refuse"`, and each entry in `unsaid`. The subject is
the rule's own: for such a place, `place <place>`.

| A finding when |
|---|
| The rule has no `hold`. |
| The hold's file does not exist. |
| The hold's file does not contain the test's name, exactly as written. |

## Does the code follow the design?

Three checks are made on each description file: `follows`, `source`
and `unsaid`. One more case of `follows` is made once for the run.

### `follows`

The checker finds the element that `describes` names in the definition
file. It then compares that element, and every element below it, in
the two files. In each file it looks only at the described element
and the elements below it in that file.

| Compared | Matched by | A finding when | Subject |
|---|---|---|---|
| Elements | Name | An element is in one file only. | The element |
| Shape | | One is a leaf and the other has children, or they combine their children in different ways, or the children differ. For a `join` the order counts. For an `or`, branches are matched by their child; the children and the `otherwise` count, and the order of the branches and the wording of `when` do not. | The element |
| Reads | Place | A read is in one file only. | The read |
| Writes | Place | A write is in one file only, or `mode`, `key` or `records` differs. | The write |
| Checks | `check` | A check is in one file only, or `then` differs. | The check |
| `rejects`, `declines` | | It is in one file only, or `to` differs. | The `rejects` or `declines` |
| Places | Name | A place listed in the description file is not in the definition file, or its `keyed` or `onExisting` differs. | The place |

When an element is in one file only, one finding covers the element,
the elements below it, and all their reads, writes, checks, `rejects`
and `declines`. In the same way, one
finding covers a write that is in one file only and its checks. The
parent of an element that is in one file only has its own finding,
because its children differ. A place is compared whenever the
description file lists it. The message of each finding says which file
has the statement, or how the two differ.

Two more cases are findings of `follows`:

| A finding when | Subject |
|---|---|
| `describes` names an element the definition file does not have, or one that is not a child of the top element. This is the one `follows` finding for that description file: its elements, statements and places are not compared. Its `source` and `unsaid` checks are still made. | The element |
| Once for the run: a child of the top element has no description file among the files given. | The element |

### `source`

Made on every statement in a description file, including each branch of
an `or`.

| A finding when |
|---|
| The statement has no `source`. |
| The source's file does not exist. |
| The source's file does not contain the function's name as a whole word: with no letter, digit or underscore directly before or after it. |

### `unsaid`

Each entry in a description file's `unsaid` is a finding, unless the
definition file's `unsaid` has an entry with exactly the same `what`.
A finding says the code does something the design does not state, and
the owner decides whether the design should require it.

## Accepted findings

| Findings | Accepted in the file beside |
|---|---|
| `follows`, `source` and `unsaid`, made on a description file | That description file |
| Every other finding, including a child of the top element with no description file | The definition file |

An acceptance matches a finding of the same check and subject that
belongs in its file. An acceptance that matches no finding in the run
is itself a finding. Its check is `stale`, its subject is the
acceptance's `finding`, and it cannot be accepted: the owner takes the
acceptance out.

```
stale  held  import declines: accepted, and no longer found
```

An acceptances file is refused when a `finding` in it does not begin
with the name of a check on this page and two spaces, or begins with
`stale`, or when a `reason` is empty.

## An example

For the two files at the end of `fields.md`, with the tests and
functions they name all in place, the run prints one line and fails:

```
follows  import writes volume, check repeated: in the definition file only
```

The owner then has the code changed to refuse a name given twice, or
changes the design, or accepts the finding with this beside the
description file:

```json
{ "accepted": [
  { "finding": "follows  import writes volume, check repeated",
    "reason": "a disk image cannot hold two files of one name" } ] }
```
