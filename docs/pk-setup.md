# pk setup

Configure a project's hooks, skills, and CLAUDE.md for use with plankit.

## Usage

```bash
pk setup                              # default: block guard, manual preserve
pk setup --guard ask                  # prompt user instead of blocking on protected branches
pk setup --preserve auto              # auto-preserve plans on ExitPlanMode
pk setup --project-dir /path/to/dir   # specify project directory
pk setup --force                      # overwrite all managed skills
pk setup --allow-non-git              # proceed even if directory is not a git repo
pk setup --baseline                   # also create v0.0.0 tag if no version tag exists
pk setup --baseline --at <ref>        # tag <ref> as v0.0.0 instead of HEAD
pk setup --baseline --push            # also push the tag to origin
```

Setup refuses to install outside a git working tree by default — most pk commands require git. Run `git init` first, or pass `--allow-non-git` to proceed anyway. Monorepo subdirectories are correctly detected as inside a git repo (the check walks up parents looking for `.git`).

## How it works

1. **Configures `.claude/settings.json`** with PreToolUse, PostToolUse, and SessionStart hooks (guard, protect, preserve, bootstrap), and adds `Bash(pk:*)` permission for skill execution. Existing user hooks are preserved — only plankit hooks are added or updated.
2. **Creates `CLAUDE.md`** with critical rules if none exists. If a pk-managed CLAUDE.md exists and hasn't been modified, it is updated. User-modified or unmanaged files are left alone. CLAUDE.md is never force-overwritten — once customized, it is user-owned.
3. **Installs rules** to `.claude/rules/`: model-behavior, development-standards, git-discipline. These contain the detailed guidelines that Claude Code loads automatically.
4. **Installs skills** to `.claude/skills/`: `/init`, `/preserve`, `/ship`. User-modified skills are skipped unless `--force` is used.
5. **Writes `.claude/install-pk.sh`** — a bootstrap script that downloads `pk` into cloud sandboxes (Claude Code on the web). The script is pinned to the running `pk` version and is a no-op when `pk` is already on PATH. Skipped for development builds.
6. **Checks PATH** and warns if `pk` is not found.

After setup, restart Claude Code to apply changes.

## Flags

- **--guard** — Guard mode: `block` or `ask` (default: `block`). Controls whether `pk guard` blocks git mutations outright or prompts the user to confirm.
- **--preserve** — Plan preservation mode: `manual` or `auto` (default: `manual`).
- **--force** — Overwrite all managed skills regardless of user modifications. Does not affect CLAUDE.md.
- **--allow-non-git** — Proceed even if the project directory is not inside a git working tree. Setup refuses by default; this flag is the escalation for cases where pk is being installed before `git init`, or when only pk's non-git features (rules, skills, `pk protect`) are wanted.
- **--project-dir** — Project directory (default: current directory).
- **--baseline** — After setup, create a `v0.0.0` tag on HEAD if no valid semver tag exists in the repo. Idempotent: if any tag parses as semver (e.g. `v0.0.0`, `v1.2.3`), the step is a no-op. See [Baseline tag for pk changelog](#baseline-tag-for-pk-changelog).
- **--at** — Tag the given ref instead of HEAD. Requires `--baseline`. Use this to anchor an existing repo at its first commit so all prior work lands in the first changelog entry (`pk setup --baseline --at $(git rev-list --max-parents=0 HEAD)`).
- **--push** — After tagging, publish to `origin`. Requires `--baseline`. Pushes HEAD + tag by default, so the tagged commit is reachable from a branch on origin (matching `pk preserve --push`). With `--at`, pushes the tag only — the user picked the ref, pk doesn't assume which branch goes with it. Without `--push`, the tag stays local and pk prints the manual push command — consistent with the git-discipline rule that commit and push are separate decisions.

## Details

### Running without git

pk is designed for git repositories, but parts of it work without git. When `--allow-non-git` is used, setup installs everything but some commands will not function:

| Feature | Without git |
|---------|-------------|
| Rules (`.claude/rules/`) | Fully functional — Claude Code loads them regardless |
| `pk protect` | Fully functional — checks file paths only |
| `pk guard` hook | Silent no-op — nothing to guard |
| `pk preserve` hook | Silent skip — plan is not saved |
| `pk changelog` | Fails (exit 1) — needs `git log` |
| `pk release` | Fails (exit 1) — needs tags, branches |

The common non-git use case is getting pk's rules and skills in a scratch directory or during the gap between `pk setup` and `git init`.

### CLAUDE.md

The CLAUDE.md installed by `pk setup` contains critical rules — the non-negotiable behaviors that prevent the most common issues. Detailed guidelines for model behavior, development standards, and git discipline are installed as `.claude/rules/` files, which Claude Code loads automatically alongside CLAUDE.md.

Add a `## Project Conventions` section to make Claude productive from the first message of every session. Without project conventions, Claude follows the rules but has to rediscover the project each session. With them, it knows the project from the start.

### Customize your CLAUDE.md

After running `pk setup`, run `/init` to add project-specific conventions. The skill analyzes the codebase, discovers technical conventions and business rules, asks about branch protection, and proposes a `## Project Conventions` section for your approval.

### Add your own skills

`pk setup` installs three managed skills (`/init`, `/preserve`, `/ship`). You can add your own — see [Creating skills](creating-skills.md).

### Guard modes

- **block** (default) — Git mutations on protected branches are denied outright.
- **ask** — The user is prompted to confirm or reject, allowing emergency overrides.

### Preserve modes

- **manual** (default) — Use the `/preserve` skill when you're ready to save a plan.
- **auto** — Plans are automatically preserved when you exit plan mode.

Re-running `pk setup` preserves the existing mode configuration. Pass `--guard` or `--preserve` explicitly to change modes.

### Managed file protection

`pk setup` manages files with three update strategies:

- **CLAUDE.md** — starts managed, becomes user-owned once customized. Protected by a SHA256 marker (`<!-- pk:sha256:... -->`). Updated when pristine, skipped when modified. Never force-overwritten — once you add project conventions, it's yours.
- **Skills and rules** — protected by a `pk_sha256` field in YAML frontmatter. Updated when pristine, skipped when modified. `--force` reclaims them. When pk no longer ships a skill or rule, the local copy is removed on the next `pk setup` run if its hash still matches — user-modified copies are preserved with a warning, and skills you authored yourself (no `pk_sha256` marker) are skipped without notice.
- **install-pk.sh** — always overwritten with the latest template. No SHA protection. Users have no reason to customize it — it's infrastructure, not content. Script fixes ship to every project on the next `pk setup` run.

### Baseline tag for pk changelog

`pk changelog` reads every commit since the most recent semver tag. Without a tag, there is nothing to diff from and the command errors out. `--baseline` creates that anchor without you having to remember the raw git commands.

Three common scenarios:

**1. New repo, anchor from the initial commit.**

```bash
cd your-project
git init
# ... first commit ...
pk setup --baseline
```

Tags HEAD (your initial commit) as `v0.0.0`. Next conventional commits on `develop` become the first changelog entry.

**2. Existing repo, anchor from current state.**

```bash
cd your-repo
pk setup --baseline
```

Tags current HEAD as `v0.0.0`. Prior commits are treated as prior art — they do not appear in the first changelog entry.

**3. Existing repo, include prior commits in the first changelog.**

```bash
cd your-repo
pk setup --baseline --at $(git rev-list --max-parents=0 HEAD)
```

Tags the initial commit of the repo as `v0.0.0`. Every commit since (including any pre-pk history) will be scanned by `pk changelog` and categorized by its conventional-commit type.

**Idempotent.** If any valid semver tag exists in the repo, `--baseline` is a no-op and prints which tag was found. Running it repeatedly is safe.

**Tag is local by default.** After tagging, `pk setup --baseline` prints the push command for you to run when ready. Pass `--push` to tag and push in one step. The manual push keeps the commit/push separation — creating a tag is reversible, publishing it is not.

**Discoverability tip.** When `pk setup` runs in a git repo with no valid semver tag, it prints a tip suggesting `pk setup --baseline`. The tip is only shown when there is no tag — once anchored, setup is quiet.

### Cloud sandbox bootstrap

`pk setup` writes a SessionStart hook and `.claude/install-pk.sh` that together bootstrap `pk` into Claude Code on the web. The script downloads the `pk` binary from GitHub Releases into `$HOME/.local/bin` at session start, then runs a best-effort `git fetch --tags` so `pk changelog` and `pk release` see the repo's version tags — sandboxes clone only the working branch by default.

On local surfaces (CLI, Desktop, VS Code), the script detects the existing `pk` install and exits immediately — no download, no PATH change.

The script is pinned to the version of `pk` that ran `pk setup`. After upgrading plankit, re-run `pk setup` to update the pinned version. `pk version` warns when the pinned version falls behind the running version.
