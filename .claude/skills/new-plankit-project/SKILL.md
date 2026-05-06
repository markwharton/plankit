---
name: new-plankit-project
description: Emit the init commands for a new plankit-tooled project — GitHub repo, pk setup --baseline, v0.0.0 tag, develop branch
disable-model-invocation: true
argument-hint: <name> [private <org>]
---

Generate the command sequence for initializing a new plankit-tooled project. The skill emits a ready-to-run shell script; the user reviews and runs it. The skill does not run the commands itself — creating a GitHub repo and pushing to origin are side-effecting actions that deserve explicit human review.

## Usage

Takes two or four arguments:
- `name` (required) — the repo name
- `description` (required) — short sentence for the GitHub repo description
- `private` (optional) — makes the repo private (no license file)
- `org` (required when private) — the GitHub org (e.g., `HeliMods`)

If `name` or `description` is missing, ask the user before emitting the script. If `private` is specified without `org`, ask for it.

**Public (default):** `markwharton/<name>`, parent dir `~/Projects/markwharton/`, MIT license.
**Private:** `<org>/<name>`, parent dir `~/Projects/<org>/`, no license.

## Steps

1. Read the arguments.
2. Print the appropriate command sequence below, with placeholders filled in.
3. Tell the user to review and run it.

## Command template — public (default)

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
git checkout -b develop
git push -u origin develop
```

## Command template — private

```bash
cd ~/Projects/<ORG>/

gh repo create <ORG>/<NAME> \
  --private \
  --description "<DESCRIPTION>" \
  --clone

cd <NAME>
git commit --allow-empty -m "chore: init"
pk setup --baseline
git add -A
git commit -m "chore: pk setup"
git push -u origin main
git push origin v0.0.0
git checkout -b develop
git push -u origin develop
```

## Design notes

- **Parent directory:** `~/Projects/markwharton/` for public, `~/Projects/<org>/` for private.
- **Visibility:** Public by default. Private repos belong to an org and skip the license.
- **License:** MIT for public repos. Private repos have no license file.
- **v0.0.0 anchor commit.** Public repos: `gh repo create --license MIT` creates the initial commit (the LICENSE file). Private repos: `git commit --allow-empty -m "chore: init"` creates the anchor since there is no license file. In both cases, `pk setup --baseline` tags this commit as `v0.0.0`. The setup files become the next commit. `pk changelog`'s first real release starts from that commit — honest about where the project's code history begins.
- **Two explicit pushes** — `git push -u origin main` first, then `git push origin v0.0.0`. At the init stage, reviewing each step matters more than brevity. When the pattern is trusted and boring, this could collapse to `pk setup --baseline --push`. For now, keep the explicit pushes.
- **Setup commit message** — `"chore: pk setup"`. Short, accurate, no flavor.
- **`develop` created and pushed at init.** `pk release` expects `origin/develop` to exist (it runs `git fetch origin develop` and `merge-base HEAD origin/develop` as pre-flight). Establishing the branch on origin at init avoids a cryptic git error weeks later and keeps every plankit-tooled project in the same starting state.
- **No `/init` afterwards** — the project doesn't have conventions to discover yet. The user runs `/init` separately once the scaffold has enough shape for conventions to matter.

## Out of scope

- **Homebrew taps** (e.g., `homebrew-plankit`) are not plankit-tooled projects. They use a different init pattern — no `pk setup`, Formula directory, tap-specific README. This skill does not handle that case.

## Contract

- **Input:** `name` (required), `description` (required), `private` (optional), `org` (required when private).
- **Output:** shell script printed to stdout, ready to review and run.
- **Side effects:** none. The skill never runs the commands itself.
