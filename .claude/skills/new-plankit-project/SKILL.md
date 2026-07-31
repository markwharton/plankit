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
pk init --push
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
pk init --push
```

## Design notes

- **Parent directory:** `~/Projects/markwharton/` for public, `~/Projects/<org>/` for private.
- **Visibility:** Public by default. Private repos belong to an org and skip the license.
- **License:** MIT for public repos. Private repos have no license file.
- **The anchor commit.** `pk init` tags `v0.0.0` on whatever commit is at HEAD, so the repo needs one first. `gh repo create --license MIT` creates it (the LICENSE commit). Private repos have no license file and so no commit, which is why the private template adds `git commit --allow-empty`.
- **Branch protection.** `pk init` writes `.github/protect-main.json` but does not apply it; it prints the `gh api` command. Run that after the script, or import the file through the GitHub UI.
- **After init.** The script leaves you on `develop`. Restart Claude Code so the hooks load, then run `/conventions`.

## Out of scope

- **Homebrew taps** (e.g., `homebrew-plankit`) are not plankit-tooled projects. They use a different init pattern — no `pk setup`, Formula directory, tap-specific README. This skill does not handle that case.

## Contract

- **Input:** `name` (required), `description` (required), `private` (optional), `org` (required when private).
- **Output:** shell script printed to stdout, ready to review and run.
- **Side effects:** none. The skill never runs the commands itself.
