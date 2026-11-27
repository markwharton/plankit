# control lab

A local page for designing a system as a definition file. It runs a
design on every change and draws the result five ways. It opens the
checker's definition files and designs you can run, and it changes
nothing in the checker.

## Running it

```
python3 lab/lab.py
```

That builds the checker from `control/`, opens the bundled examples and
the designs in `lab/designs/`, and opens the page in a browser.
Ctrl-C stops it. It needs Python 3 and Go, and nothing else.

To open a design of your own:

```
python3 lab/lab.py path/to/designs/
python3 lab/lab.py --root path/to/repo path/to/definition.json path/to/description.json
```

A directory is searched for definition files. Each one is a design, the
description files beside it are its descriptions, and its own directory
is its root. `--root` is for a definition file named directly, where
holds and sources are paths from a repository's root.

## Two kinds of design

The lab opens both. A definition file in the checker's format is checked
by `control`. A design you can run, in the form `engine/DESIGN.md` gives,
is played by the engine in `engine/`. The page tells them apart by the
file: a design you can run has `parts`. For one of those the controls
read "Given" and "Wrong" where the checker's read "Path" and "Break",
and `engine/view.py` lays the engine's plays out for the same five views.

## What it does

- **Checks as you type.** The editor on the left holds the definition
  file. Each change is sent to the checker, and nothing is written to
  disk until you choose Save. Text the checker refuses is shown with
  its reasons, and the views keep the last version it accepted.
- **Follows the file.** When the definition file, a description file,
  an acceptances file, or a file a hold or source names changes on
  disk, the page checks again. A session or an editor can change the
  design while the page is open.
- **Draws it five ways.** Rail, Bus, Ring, Grid and Matrix. Under each
  the page says what that view is good at and weak at. Pick a place
  under Break to give it an invalid value in every view.
- **Compares.** "Compare with" takes the saved file, a snapshot, or
  another design, and the Compare panel lists the findings that are new
  or gone and the sweep verdicts that changed.
- **Hands work to a session.** "Tests to write" copies each missing
  test as a prompt. "Prompts" fills in the checker's own prompt for
  writing a description file, read from `control/README.md`.

Designs listed in `examples/lab.json` are read-only, so that a slip of
Save cannot change the checker's own examples or `definition/pk.json`.
Files under `designs/` can be saved.

## What it does not do

- It holds no rule. Every finding, verdict and trace comes from
  `control view` and `control check`. The page works out three things
  for drawing: the state of each place after each leaf, which element
  reports a refusal, and the leaves under a parent.
- It reads the checker's model out of the page `control view` writes,
  from the script element with the id `view-data`. The checker has no
  command that prints that model directly.
- It listens on 127.0.0.1 only, answers only its own page, and writes
  only a design's own file or a new file beside it.

## The files

| File | What it is |
|---|---|
| `lab.py` | The server: finds designs, runs the checker, saves a file. |
| `lab.html` | The page: the editor, the five views and the panels. |
| `examples/` | The ten stages of the contacts design, and `lab.json`, which also lists the checker's trials and `definition/pk.json`. |
| `designs/` | Designs you can save. It starts with a copy of the contacts design, and a design written as text, `notes/notes.design`. |
| `engine/` | The engine that plays a design you can run, its reference `DESIGN.md`, pk's design, and `BLIND-TEST.md`. |
| `test_lab.py` | Tests of the server against the real checker and the engine. |
| `e2e.mjs` | A test of the page in headless Chrome. |

## Tests

```
cd lab
python3 -m unittest test_lab
node e2e.mjs
```

`make test` runs the first two with the repository's own tests, and
`make lab-e2e` runs the third. `e2e.mjs` needs Node 22 or later and Google Chrome. Both suites copy
`designs/` to a scratch directory first, so they change nothing here.

## Adding a view

A view is one function in `lab.html` that takes the current selection
and fills `#stage`. Add it to `DRAW` and to `VIEWS`, with a line each
for what it is good at and weak at.
