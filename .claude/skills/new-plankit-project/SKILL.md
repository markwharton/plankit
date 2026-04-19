---
name: new-plankit-project
description: Emit the init commands for a new plankit-tooled project — GitHub repo with MIT license, pk setup --baseline, v0.0.0 tag anchored at the license commit
disable-model-invocation: true
---

Generate the command sequence for initializing a new plankit-tooled project. The skill emits a ready-to-run shell script; the user reviews and runs it. The skill does not run the commands itself — creating a public GitHub repo and pushing to origin are side-effecting actions that deserve explicit human review.

## Usage

Takes two arguments:
- `name` — the repo name (becomes `markwharton/<name>`)
- `description` — short sentence for the GitHub repo description

If either is missing, ask the user before emitting the script.

## Steps

1. Read the `name` and `description` arguments.
2. Print the command sequence below, with the two placeholders filled in.
3. Tell the user to review and run it.

## Command template

```bash
cd ~/Projects/markwharton/

gh repo create markwharton/<NAME> \
  --public \
  --license MIT \
  --description "<DESCRIPTION>" \
  --clone

cd <NAME>
pk setup --baseline
git add -A
git commit -m "chore: pk setup"
git push -u origin main
git push origin v0.0.0
```

## Design notes

- **Parent directory:** `~/Projects/markwharton/`. All plankit-family projects live here.
- **Visibility:** `--public`. plankit projects are open by default.
- **License:** MIT. Consistent across plankit-family repos.
- **v0.0.0 lands on the license commit.** `gh repo create --license MIT` creates the initial commit (the LICENSE file). `pk setup --baseline` then tags that commit as `v0.0.0`. The setup files become the second commit. `pk changelog`'s first real release will start from that second commit — honest about where the project's code history begins.
- **Two explicit pushes** — `git push -u origin main` first, then `git push origin v0.0.0`. At the init stage, reviewing each step matters more than brevity. When the pattern is trusted and boring, this could collapse to `pk setup --baseline --push`. For now, keep the explicit pushes.
- **Setup commit message** — `"chore: pk setup"`. Short, accurate, no flavor.
- **No `/init` afterwards** — the project doesn't have conventions to discover yet. The user runs `/init` separately once the scaffold has enough shape for conventions to matter.

## Out of scope

- **Homebrew taps** (e.g., `homebrew-plankit`) are not plankit-tooled projects. They use a different init pattern — no `pk setup`, Formula directory, tap-specific README. This skill does not handle that case.
- **Private repos.** plankit-family projects are public by default. For a private one, edit the emitted script before running.

## Contract

- **Input:** `name` (required), `description` (required).
- **Output:** shell script printed to stdout, ready to review and run.
- **Side effects:** none. The skill never runs the commands itself.
