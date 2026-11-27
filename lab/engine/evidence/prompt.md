I need evidence for ten open questions about pk, so that I can decide each one.
You are gathering the evidence. You are not deciding, and you change no tracked
file.

## Background

This repository has a design language and an engine that plays it. The reference
is lab/engine/DESIGN.md; read it first. A design says what a system's runs do,
and the engine plays every run and reports each thing the design leaves
unsettled, as a question for the owner. I am the owner.

lab/engine/pk.json is the design of pk's changelog, release and ship commands.
It stands on lab/engine/git.json, a design of the git operations pk runs. The
work is on the branch design-engine-next, which is not pushed.

Print the ten questions in full, each with the plays that raise it:

    cd lab/engine && python3 engine.py pk.json --root ../..

lab/engine/test_pk.py holds the same list.

A question is one of four kinds:
- open: a fact nobody has decided, played both ways.
- expect: a play breaks something the design says must hold.
- unanswered: something lasting is changed while a fault no answer covers is
  carried into it, or a failure comes up from git and the pk action has no
  answer for it.
- left-behind: a lasting place was changed, a failure followed, and the design
  does not say whether the change is undone or stands.

## The ten questions

1. open: "a rejected tag update fails the fetch". git.json does not know
   whether `git fetch origin --tags` exits non-zero when it rejects an update
   to a tag that exists locally with a different target.
2. expect: "changelog with --dry-run writes nothing" is broken, because fetching
   origin's tags adds tags locally.
3. unanswered: "a tag does not name a version". Nothing says what version
   inference does with a tag that is not a version.
4. unanswered: "origin has no address". changelog accepts it and writes the
   section with no link. Later runs read what changelog left, and none says
   whether that is fine.
5. unanswered: "origin's tags already has the member". git's atomic push refuses
   a tag origin already has, and neither push action in pk.json answers.
6. unanswered: "settings are invalid". The design skips "branch checks" in a dry
   run, so in a dry run nothing refuses invalid settings.
7. unanswered: "the branch is not the default", in a dry run and at --undo.
8. unanswered: "the working branch is unfit", in a dry run and at --undo.
9. left-behind: "remote-tracking branch". The pre-flight fetch updates it, and
   nothing says what happens to that when a later step fails.
10. left-behind: "tags". The same, for the tags `fetch tags` brought.

Also check one thing the engine does not raise today: git.json's second open
fact, "a merge refuses a tree with changes it would overwrite", about
`git merge --ff-only` with uncommitted changes.

## What I need for each question

Three things, each with how you know it:

1. What pk's own pages say. They are skills/changelog/SKILL.md,
   skills/release/SKILL.md and skills/ship/SKILL.md. Quote the sentence, or say
   the pages are silent.
2. What pk's code does today. Start in internal/changelog/changelog.go,
   internal/release/release.go, internal/ship/ and internal/git/git.go. Give
   file and line, and name the test that holds the behaviour if there is one.
3. What git does, where the question turns on git. Do not answer from memory
   or from the manual alone. Run it in a scratch repository and show the
   command, the exit status and the output. Say which git version you ran.

Then, for each question, the ways I could settle it, with the edit each would
make to pk.json or git.json written out as JSON. DESIGN.md says what forms an
answer takes. Where the pages and the code disagree, say so plainly: that is a
finding in its own right. Do not recommend one option unless the pages and the
code agree and leave only one that is consistent with both.

## Leads, not conclusions

These are things I noticed in passing and have not verified. Confirm or correct
each.

- internal/changelog/changelog.go near line 151 appears to go on when the fetch
  of tags fails.
- internal/release/release.go near line 307 pushes the branch and the tag with
  `git push --atomic`.
- skills/changelog/SKILL.md says --undo with --dry-run "makes the same checks".
- An earlier reader reported that the pages say a failure "after tagging" rolls
  back, and say nothing of a commit a preRelease hook made; and that --undo
  needs a "valid" Release-Tag trailer where release speaks only of "no"
  trailer. Check both against the code while you are there.

## How to work

- Scratch repositories go in your scratchpad directory, never in this
  repository. Use a local bare repository as origin. Nothing here needs the
  network, and nothing touches this repository's real origin.
- To watch pk itself, run `make build` and use ./bin/pk in a scratch repository
  with its own .pk.json. Never run a pk command that writes in this repository.
- The existing tests already build such repositories; read them as evidence,
  and run one with `go test -run TestName ./internal/<pkg>` where that is the
  quickest proof.
- Change no tracked file. Do not edit pk.json, git.json, the pages or the code.
  Make no commit.
- If a question cannot be settled by reading and running, say what you tried
  and what would settle it.

## What to hand back

One section for each of the ten questions and one for the extra git fact, in
the order above. Each has: what the pages say, what the code does, what git
does where it applies, where they disagree, and the ways to settle it with the
JSON for each. Mark every claim as read (file and line) or run (command and
output). End with a short list of anything you could not verify.
