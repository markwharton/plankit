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
```

Setup refuses to install outside a git working tree by default — most pk commands require git. Run `git init` first, or pass `--allow-non-git` to proceed anyway. Monorepo subdirectories are correctly detected as inside a git repo (the check walks up parents looking for `.git`).

## How it works

1. **Configures `.claude/settings.json`** with PreToolUse, PostToolUse, and SessionStart hooks (guard, protect, preserve, bootstrap), and adds `Bash(pk:*)` permission for skill execution. Existing user hooks are preserved — only plankit hooks are added or updated.
2. **Creates `CLAUDE.md`** with critical rules if none exists. If a pk-managed CLAUDE.md exists and hasn't been modified, it is updated. User-modified or unmanaged files are left alone. CLAUDE.md is never force-overwritten — once customized, it is user-owned.
3. **Installs rules** to `.claude/rules/`: model-behavior, development-standards, git-discipline. These contain the detailed guidelines that Claude Code loads automatically.
4. **Installs skills** to `.claude/skills/`: `/init`, `/changelog`, `/preserve`, `/release`. User-modified skills are skipped unless `--force` is used.
5. **Writes `.claude/install-pk.sh`** — a bootstrap script that downloads `pk` into cloud sandboxes (Claude Code on the web). The script is pinned to the running `pk` version and is a no-op when `pk` is already on PATH. Skipped for development builds.
6. **Checks PATH** and warns if `pk` is not found.

After setup, restart Claude Code to apply changes.

## Flags

- **--guard** — Guard mode: `block` or `ask` (default: `block`). Controls whether `pk guard` blocks git mutations outright or prompts the user to confirm.
- **--preserve** — Plan preservation mode: `manual` or `auto` (default: `manual`).
- **--force** — Overwrite all managed skills regardless of user modifications. Does not affect CLAUDE.md.
- **--allow-non-git** — Proceed even if the project directory is not inside a git working tree. Setup refuses by default; this flag is the escalation for cases where pk is being installed before `git init`, or when only pk's non-git features (rules, skills, `pk protect`) are wanted.
- **--project-dir** — Project directory (default: current directory).

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

`pk setup` installs four managed skills (`/changelog`, `/init`, `/preserve`, `/release`). You can add your own — see [Creating skills](creating-skills.md).

### Guard modes

- **block** (default) — Git mutations on protected branches are denied outright.
- **ask** — The user is prompted to confirm or reject, allowing emergency overrides.

### Preserve modes

- **manual** (default) — Use the `/preserve` skill when you're ready to save a plan.
- **auto** — Plans are automatically preserved when you exit plan mode.

Re-run setup anytime to switch modes.

### Managed file protection

`pk setup` manages files with three update strategies:

- **CLAUDE.md** — starts managed, becomes user-owned once customized. Protected by a SHA256 marker (`<!-- pk:sha256:... -->`). Updated when pristine, skipped when modified. Never force-overwritten — once you add project conventions, it's yours.
- **Skills and rules** — protected by a `pk_sha256` field in YAML frontmatter. Updated when pristine, skipped when modified. `--force` reclaims them.
- **install-pk.sh** — always overwritten with the latest template. No SHA protection. Users have no reason to customize it — it's infrastructure, not content. Script fixes ship to every project on the next `pk setup` run.

### Cloud sandbox bootstrap

`pk setup` writes a SessionStart hook and `.claude/install-pk.sh` that together bootstrap `pk` into Claude Code on the web. The script downloads the `pk` binary from GitHub Releases into `$HOME/.local/bin` at session start.

On local surfaces (CLI, Desktop, VS Code), the script detects the existing `pk` install and exits immediately — no download, no PATH change.

The script is pinned to the version of `pk` that ran `pk setup`. After upgrading plankit, re-run `pk setup` to update the pinned version. `pk version` warns when the pinned version falls behind the running version.
