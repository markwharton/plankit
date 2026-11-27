# Evidence for pk's ten questions, 2026-11-27

A record of one run. The engine raised ten questions on
`lab/engine/pk.json`. Before the owner decided any of them, a new
session was asked to gather the evidence for each: what pk's pages say,
what pk's code does, and what git does when it is run. It was told to
decide nothing and to change no tracked file.

This is the step between the engine's questions and the owner's
decisions. Nothing here is agreed design. The options in the report are
that session's, and the decisions are the owner's.

## What is here

| File | What it is |
|---|---|
| `prompt.md` | The prompt, as it was given. |
| `pk-2026-11-27.md` | The report, as it came back. |
| `try.py`, `edits.py` | The harness the session wrote to try each option: it applies an edit to copies of `pk.json` and `git.json`, plays them, and prints the questions that went and the ones that came. |

The session ran on Opus 5.5, with git 2.54.0, against the designs at
commit `dc2275e`.

## Reading the report

- It marks each claim as read, with a file and line, or run, with the
  command and its output. Nothing in it was checked a second time.
- Its last lines name files in that session's scratch directory. Those
  are not kept. `try.py` and `edits.py` here are the copies.
- It suggests a script for measuring a raw `git push --atomic`. That
  script was not run and is not kept.

## Replaying an option

From the repository's root. Each option is played on its own, starting
from `lab/engine/pk.json` and `lab/engine/git.json` as they are now.

```
python3 lab/engine/evidence/try.py                      # every option
python3 lab/engine/evidence/try.py q9_stands            # one
python3 lab/engine/evidence/try.py q3_refuse q4_refuse  # several, each played separately
```

### Reading the output

Each option prints one line, and one more for each question it adds:

```
== q78_split_as_code: 8 findings (baseline 10); gone: ['unanswered  settings are invalid', ...]
   NEW unanswered  the working branch is behind origin: ...
```

- `baseline` is the number of questions the designs raise before the
  edit.
- `gone` lists the questions the option settles.
- `NEW` is a question the option raises.
- `engine refused` means the engine could not play the edited design,
  and its reason follows.

### The options

An option's name is the name of its function in `edits.py`. That file
is the list:

```
grep -n "^def q\|^def x\|^def all" lab/engine/evidence/edits.py
```

Which option is which in the report:

| Question | In the report | Name |
|---|---|---|
| 1 | A, pk fails after the tags arrive | `q1_true_pk_fails` |
| 1 | B, pk refuses | `q1_true_pk_refuses` |
| 1 | C, pk keeps its accept | `q1_true_pk_accepts` |
| 1 | pk's answer left out | `q1_true_pk_silent` |
| 2 | A, the expectation is narrowed | `q2_narrow_expectation` |
| 2 | B, a dry run reads origin's tags and fetches nothing | `q2_dry_run_reads_origin` |
| 3 | A, a state and a rule in words | `q3_state_with_rule` |
| 3 | B, refuse | `q3_refuse` |
| 3 | An accept, tried, which does not settle it | `q3_accept_as_code` |
| 4 | A, a state | `q4_state_not_fault` |
| 4 | B, refuse | `q4_refuse` |
| 5 | A, the push actions refuse | `q5_push_refuses` |
| 5 | B, pre-flight refuses | `q5_preflight_refuses` |
| 5 | C, A with git taking a tag on the same commit | `q5_same_commit_is_fine` |
| 6 | Settings are loaded first | `q6_load_settings_first` |
| 7 and 8 | The fact split in four, as the report's JSON has it | `q78_as_code` |
| 7 and 8 | The split, with a branch behind origin refused by changelog | `q78_refuse_everywhere` |
| 7 and 8 | Steps on the way to those two | `q78_split_as_code`, `q78_behind_accepted`, `q78_behind_refused` |
| 9 | A, stands | `q9_stands` |
| 10 | A, stands | `q10_stands` |
| 10 | B, undone | `q10_undone` |
| The extra git fact | The open fact decided | `x_merge_decided` |
| The extra git fact | A probe: pre-flight lets a dirty tree through | `x_preflight_lets_dirty_tree_through` |
| All ten | Every edit that matches the code | `all_as_code` |

Every option for questions 7 and 8 includes `q6_load_settings_first`.

`all_as_code` takes option B for question 1, pk refuses. The report
says option A, pk fails after the tags arrive, is what the code does.
Played alone, each settles question 1. The combination with option A
in place of B was not played.

### What it does not do

- **It does not combine options.** Naming several plays each alone.
  `all_as_code` is the one combination. For another, add a function to
  `edits.py` that calls the ones wanted, as `all_as_code` does.
- **It does not show where the edited designs are.** `try.py` writes
  them to a temporary directory and prints only the lines above, so an
  option cannot be opened in the lab from its output.
- **It follows the designs.** Once a question is decided and
  `pk.json` or `git.json` changes, an edit here may no longer apply,
  and the counts will differ from the report's.

## What was changed for keeping

`edits.py` is as the session wrote it. `try.py` has four lines changed
for its new home: it finds the repository from where it is, and it
writes the copies it plays to a temporary directory, not beside itself.
