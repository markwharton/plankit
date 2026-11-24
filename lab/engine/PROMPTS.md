# Prompts

What to paste into a session to have it write in the language. The lab
reads these from this file and shows them under Prompts. Fill in what
is in angle brackets.

## Write a design from what a system should do

```
Read lab/engine/DESIGN.md in the plankit repository. It gives a design
language and its JSON keys.

Write a design in that language for <the system>, from <the pages that
say what it should do> and nothing else. State what the pages say: the
places, the facts that change a run, the faults, each action in order
with what it does about each fault that matters to it, what happens to
a change when the run stops later, and each thing that must always be
true as an expectation. Put rules about how a value is computed under
"rules".

Do not guess. Where the pages are silent, leave the design silent in
the ways DESIGN.md says the engine will ask about, and use an open fact
for anything else the language makes you choose.

Save it as <the design file> and run:
python3 lab/engine/engine.py <the design file> --root <the repository>
Fix the file only until the engine accepts it. Do not change it to make
questions go away. Report each question, and each point where the
language could not say what the pages say.
```

## Describe what the code does

```
Read lab/engine/DESIGN.md in the plankit repository, and <the design
file> for its names only: the places, the facts and the names of the
actions. Do not read what its actions do.

Write a second design in the same language that says what the code of
<the system> does now, from the code alone. Use the design's exact
names for every place, fact and action the code has. Add a name of your
own for anything the code has and the design lacks, and leave out what
the code does not have. Write an answer only where the code really
makes that check, and "later" only where the code really undoes a
change or deliberately leaves it. Leave out "hold", "expects" and
"rules".

Save it as <the code's design file> and run:
python3 lab/engine/engine.py <the design file> --root <the repository> --against <the code's design file>
Report each difference. Change no code.
```
