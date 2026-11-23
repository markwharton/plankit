# The fields

This page lists the fields of the files the checker reads. Read
`format.md` first: it says what the files are, and it defines place,
element, statement, rule, hold, source and finding.

The page covers the two kinds of file `format.md` describes, the
definition file and the description file, and the acceptances file
that can sit beside either. A definition file or a description file
can have any name and sit anywhere in the repository. The checker is
given their paths.

## All the files

Each file is one JSON object. The checker refuses a file that has a key
this page does not list, lacks a key this page requires, spells a key
in another case, gives a key twice in one object, has a value of the
wrong type, or has anything after the object's end.

Every top-level key is required. In the tables below, a field marked
"always" is required. Any other field is required in the case its row
gives and is not written in any other case. In an object shown in
braces, every key is required unless the text says it can be left out. A field whose value is `true` is left out when it is not true.
An empty list is left out, except at the top level.

Four fields are expected, and the checker still reads a file where one
is missing: `hold`, `source`, `invalid` and `otherwise`. `checks.md`
says when a missing one is a finding.

A name is any text. No two places share a name, and no two elements
share a name. A place and an element may share one. No place is named
`parent`, which has a meaning below.

Some fields are in words: `reason`, `when`, `what` and `invalid`. They
are for people to read. The checker does not compare the words in one
file with the words in another.

## Words this page adds

| Word | Meaning |
|---|---|
| Store | What keeps a place's data: a folder, a database table, a browser's storage. |
| Record, key | A keyed place holds many records, each under a key: files under their names, rows under their ids. |

## Evidence

| Field | Value | Used in |
|---|---|---|
| `hold` | `{ "file": ..., "test": ... }`: the test's file and the test's name as written in that file | The definition file |
| `source` | `{ "file": ..., "function": ... }`: the function's file and the function's name | The description file |

A file is given as its path from the repository's root. A source's
`function` is the name the code gives the thing: a function, a method,
or a named value that holds a function or is passed in as one. Its
`file` is the file where that thing is defined.

Every rule should have a `hold`, and every statement in a description
file should have a `source`.

## The definition file

| Key | Value |
|---|---|
| `system` | The name of the top element. |
| `places` | A list of every place. |
| `elements` | A list of every element, the top element among them. |
| `unsaid` | A list of what the design requires and the fields cannot state. Each entry is `{ "what": ..., "hold": ... }`. |

### A place

| Field | When | Value |
|---|---|---|
| `name` | Always | The place's name. |
| `belongsTo` | Unless the place is from outside the system | The name of the element the place belongs to. |
| `outside` | For a place from outside the system | `true`. In the definition file a place has exactly one of `belongsTo` and `outside`. |
| `keyed` | For a place that holds many records, each under a key | `true` |
| `lasting` | For a place that keeps its data from one run to the next: a file, a branch, a table | `true`. A value made during a run and passed to the next step is not lasting. Three checks in `checks.md` treat a lasting place differently. |
| `onExisting` | For every keyed place | What the store does when it is asked to create a record at a key that already has one. `"replace"`: it puts the new record in place of the old. `"refuse"`: it fails and keeps the old. |
| `hold` | With `onExisting` of `"refuse"` | The store refusing is a rule, so it has a hold. |
| `invalid` | For every place | In words, what makes a value of this place invalid. Hamilton gives every kind of data such a value. For a keyed place it is about one record. |

### An element

An element has one of five shapes. A leaf reads and writes places and
has no children. A `join`, an `include` and an `or` have children, and
so does the top element, which has `steps`. `format.md` says how each
combines its children. An element with children does
no reading or writing itself; the elements below it do.

The top element has `steps`, and it is the only element that does.
Every other element is the child of exactly one element.

| Field | When | Value |
|---|---|---|
| `name` | Always | The element's name. |
| `reads` | For a leaf that reads | A list of the places it reads, each `{ "place": ... }`. |
| `writes` | For a leaf that writes | A list of its writes, described below. |
| `join` | For a join | A list of at least two children's names, in the order they run. When one refuses, the ones after it do not run. |
| `include` | For an include | A list of at least one child's name. |
| `steps` | For the top element | A list of at least one child's name: the steps of the system. |
| `or` | For an or | A list of at least one branch, each `{ "when": ..., "do": ... }`: a condition in words, and the name of the child that runs when it is true. |
| `otherwise` | With every `or` | The name of the child that runs when no condition is true. |
| `rejects` | Where the element refuses bad input | `{ "reason": ..., "to": ..., "hold": ... }`. `to` says where the refusal is reported. It is the name of a place, or `"parent"`. |
| `declines` | Where the element asks a person, who can say no | `{ "reason": ..., "to": ..., "hold": ... }`. `to` is as for `rejects`. It is left out when the person's answer is not reported: no place is written and the caller is not told. Where the caller is told, `to` is `"parent"`. An element has at most one `declines`. |

How `to` works:

- For a leaf, a place in `to` is one of the places the leaf writes. A
  leaf's `writes` list holds what it writes when it does its work, and
  also the place it writes when it refuses.
- `"parent"` passes the refusal up. The parent then has its own
  `rejects`, which says where the refusal goes next. It is a rule like
  any other, and its hold is a test that the refusal arrives there.
- For a step of the system, a child of the top element, `"parent"`
  means the refusal leaves the step and the rest of the system
  receives it. In a description file, a refusal returned to a caller
  outside the step has a `to` of `"parent"`.
- For an element with children, a place in `to` is one that an element
  below it writes, since the element does not write places itself.
- The top element has no parent, so its `to` is a place.

An element has at most one `rejects`. It says once that the element
refuses bad input, and where the refusal goes, and its hold is one
test. When a check before a write refuses, that refusal also goes where
the element's `rejects` says. A design that needs a separate test for
each reason for refusing lists each reason in `unsaid`, with its hold.

### A write

| Field | When | Value |
|---|---|---|
| `place` | Always | The name of the place written. |
| `mode` | Always | For a keyed place, `"create"` when the element is making a record it takes to be new, and `"replace"` when it writes at a key it takes to have a record already, as a save of an open file does. `mode` says what the code assumes, whether or not the record is there. The checks in `checks.md` test whether the assumption is safe. For a place that is not keyed, `"write"`. |
| `key` | For a keyed place | Where the record's key comes from. `"content"`: it comes from what the element reads, such as a file's name inside a document or a name a person typed. `"assigned"`: the system makes it. |
| `records` | For a keyed place, when one run of the element makes one record | `"one"`. Left out, one run can make many records. |
| `before` | When the element checks before this write | A list of the checks. |

### A check before a write

Each is `{ "check": ..., "then": ..., "hold": ... }`.

| `check` | The element asks | `then` |
|---|---|---|
| `"exists"` | Is there already a record at this key? | `"refuse"`: it refuses. `"ask"`: it asks a person, and writes only when the person says yes. The question is part of the check and is not listed as a write, and the person's answer is not listed as a read. A question the element asks whatever is at the key is not a check, and goes in `unsaid`. |
| `"repeated"` | Does this key appear more than once among the records of this run? | `"refuse"` |
| `"belongs"` | Does the record at this key belong to what is replacing it: the same upload, the same person, the same game? The element compares what the system recorded when the record was made, which is kept with the record and is not a place of its own. | `"refuse"` when it does not |

### A part of the system not yet described

A place can be kept by a part of the system that the definition file
does not yet describe. That part is written as a leaf under the top
element that writes the place, and the place belongs to it. The leaf
has no description file, and the checker reports that until it has one
or the owner accepts the finding.

### What the fields do not say

- When a check refuses, whether the other records of the same run are
  still written.
- What an `or` reads to test a condition. The condition is in words.
- Which of the elements below a parent writes the refusal the parent
  reports.

A design that needs to say one of these uses `unsaid`.

## The description file

A description file has the same fields as the definition file, with
these differences.

| Difference | In the description file |
|---|---|
| `describes` | The file has the key `describes` where the definition file has `system`. Its value is the name of the element the file covers. |
| What is listed | `elements` holds that element and every element below it. `places` holds the places those elements read or write. |
| Fields it does not have | `belongsTo`, `outside`, `lasting`, `invalid` and `hold`. The checker refuses them. Where the system ends, which element a place belongs to, and whether a place is lasting are the design's to say. Every place is listed the same way, whoever writes it. |
| `source` | Every statement has one: each place, element, read, write, check, `rejects`, `declines`, branch, and entry in `unsaid`. `otherwise` is a name and does not have a source of its own; the element's source covers it. |
| `unsaid` | A list of what the code does and the fields cannot state. Each entry is `{ "what": ..., "source": ... }`. Where the definition file's `unsaid` states the same thing, the entry uses exactly its words. |

A description file says what the code does. A check, a `rejects` or a
`declines` is written only where the code does it. Where the code never
refuses, the element has no `rejects`.

The source of each statement is the function where the code does it:

| Statement | Its source is the function that |
|---|---|
| A place | holds the place's data as this element meets it: it opens the store, declares the variables, takes in the input, shows the message, or builds the value passed to the next step. A store built in layers is described as the element meets it: the source is the layer the element's code calls, even when that layer is in the same file, and `onExisting` is what that layer does. |
| A leaf | carries out the step |
| An element with children | calls the children |
| A read, a write | reads or writes the place |
| A check | makes the check |
| `rejects`, `declines` | refuses the input, or acts when the person says no |
| A branch | makes the choice: the one that holds the `if`, even where another function works out the value |

## The acceptances file

`{ "accepted": [ ... ] }`, where each entry is `{ "finding": ...,
"reason": ... }`. `finding` is the finding's check and subject, as the
checker prints them. An acceptances file is named for the file it sits
beside, as `format.md` says. `checks.md` gives the form of a finding.

## An example

A system with one step: an import that puts the files of a disk image
on a volume. The definition file:

```json
{
  "system": "saved files",
  "places": [
    { "name": "image", "outside": true,
      "invalid": "a file that is not a disk image" },
    { "name": "volume", "belongsTo": "import",
      "keyed": true, "onExisting": "replace",
      "invalid": "a file the title cannot load" },
    { "name": "message", "belongsTo": "import",
      "invalid": "a message with no text" }
  ],
  "elements": [
    { "name": "saved files", "steps": ["import"] },
    { "name": "import",
      "reads": [ { "place": "image" }, { "place": "volume" } ],
      "writes": [
        { "place": "volume", "mode": "create", "key": "content",
          "before": [
            { "check": "exists", "then": "ask",
              "hold": { "file": "site/import.test.ts", "test": "asks before replacing a file" } },
            { "check": "repeated", "then": "refuse",
              "hold": { "file": "site/import.test.ts", "test": "refuses a name given twice" } }
          ] },
        { "place": "message", "mode": "write" }
      ],
      "rejects": { "reason": "the file is not a disk image", "to": "message",
        "hold": { "file": "site/import.test.ts", "test": "refuses a file that is not an image" } },
      "declines": { "reason": "the person chose not to import",
        "hold": { "file": "site/import.test.ts", "test": "changes nothing when the person says no" } }
    }
  ],
  "unsaid": []
}
```

A description file for `import`, written from code that does not check
for a name given twice:

```json
{
  "describes": "import",
  "places": [
    { "name": "image",
      "source": { "file": "site/import.ts", "function": "readImage" } },
    { "name": "volume", "keyed": true, "onExisting": "replace",
      "source": { "file": "play/disk.ts", "function": "write" } },
    { "name": "message",
      "source": { "file": "site/import.ts", "function": "tell" } }
  ],
  "elements": [
    { "name": "import",
      "source": { "file": "site/import.ts", "function": "importImage" },
      "reads": [
        { "place": "image",
          "source": { "file": "site/import.ts", "function": "readImage" } },
        { "place": "volume",
          "source": { "file": "site/import.ts", "function": "confirmImport" } }
      ],
      "writes": [
        { "place": "volume", "mode": "create", "key": "content",
          "source": { "file": "site/import.ts", "function": "importImage" },
          "before": [
            { "check": "exists", "then": "ask",
              "source": { "file": "site/import.ts", "function": "confirmImport" } }
          ] },
        { "place": "message", "mode": "write",
          "source": { "file": "site/import.ts", "function": "tell" } }
      ],
      "rejects": { "reason": "the file is not a disk image", "to": "message",
        "source": { "file": "site/import.ts", "function": "readImage" } },
      "declines": { "reason": "the person chose not to import",
        "source": { "file": "site/import.ts", "function": "confirmImport" } }
    }
  ],
  "unsaid": []
}
```

The two differ in one statement: the definition file has a `repeated`
check on the write to `volume`, and the description file does not.
