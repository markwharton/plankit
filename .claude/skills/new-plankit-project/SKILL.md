---
name: new-plankit-project
description: Emit the init commands for a new plankit-tooled project — GitHub repo, pk setup --baseline, v0.0.0 tag, develop branch, .pk.json release config
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
cat > .pk.json <<'JSON'
{
  "guard": {
    "branches": ["main"]
  },
  "release": {
    "branch": "main"
  }
}
JSON
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
cat > .pk.json <<'JSON'
{
  "guard": {
    "branches": ["main"]
  },
  "release": {
    "branch": "main"
  }
}
JSON
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
- **`.pk.json` written at init; `/conventions` deferred.** The branch-topology config (`release.branch: main`, `guard.branches: [main]`) is written now, because the script establishes those exact branches and `pk release` needs `release.branch` set or it silently falls back to trunk flow and ships on `develop`. CLAUDE.md project conventions are a separate matter: they need codebase shape to discover, so the user runs `/conventions` later, once the scaffold has enough shape for conventions to matter. A later `/conventions` run merges into this `.pk.json` rather than conflicting.

## Out of scope

- **Homebrew taps** (e.g., `homebrew-plankit`) are not plankit-tooled projects. They use a different init pattern — no `pk setup`, Formula directory, tap-specific README. This skill does not handle that case.

## Contract

- **Input:** `name` (required), `description` (required), `private` (optional), `org` (required when private).
- **Output:** shell script printed to stdout, ready to review and run.
- **Side effects:** none. The skill never runs the commands itself.
