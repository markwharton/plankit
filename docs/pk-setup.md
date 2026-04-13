# pk setup

Configure a project's hooks, skills, and CLAUDE.md for use with plankit.

## Usage

```bash
pk setup                              # default: block guard, manual preserve
pk setup --guard ask                  # prompt user instead of blocking on protected branches
pk setup --preserve auto              # auto-preserve plans on ExitPlanMode
pk setup --project-dir /path/to/dir   # specify project directory
pk setup --force                      # overwrite all managed skills
```

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
- **--project-dir** — Project directory (default: current directory).

## Details

### CLAUDE.md

The CLAUDE.md installed by `pk setup` contains critical rules — the non-negotiable behaviors that prevent the most common issues. Detailed guidelines for model behavior, development standards, and git discipline are installed as `.claude/rules/` files, which Claude Code loads automatically alongside CLAUDE.md.

Add a `## Project Conventions` section to make Claude productive from the first message of every session. Without project conventions, Claude follows the rules but has to rediscover the project each session. With them, it knows the project from the start.

### Customize your CLAUDE.md

After running `pk setup`, run `/init` to add project-specific conventions. The skill analyzes the codebase, discovers technical conventions and business rules, asks about branch protection, and proposes a `## Project Conventions` section for your approval.

### Add your own skills

`pk setup` installs four managed skills (`/changelog`, `/init`, `/preserve`, `/release`). You can add your own — see [Creating skills](creating-skills.md).

### Preserve modes

- **manual** (default) — Use the `/preserve` skill when you're ready to save a plan.
- **auto** — Plans are automatically preserved when you exit plan mode.

Re-run setup anytime to switch modes.

### Managed file protection

Files installed by `pk setup` include a SHA256 integrity marker. The format depends on the file type:

- **CLAUDE.md** — HTML comment on the first line: `<!-- pk:sha256:... -->`
- **Skills** — `pk_sha256` field in the YAML frontmatter

On re-run, `pk setup` checks the marker:

- **File is pristine** (SHA matches) — updated to the latest version.
- **File was modified by user** (SHA mismatch) — skipped with a warning.
- **File has no marker** (not managed by pk) — skipped.

`--force` overrides this for skills only. CLAUDE.md is never force-overwritten.

### Cloud sandbox bootstrap

`pk setup` writes a SessionStart hook and `.claude/install-pk.sh` that together bootstrap `pk` into Claude Code on the web. The script downloads the `pk` binary from GitHub Releases into `$HOME/.local/bin` at session start.

On local surfaces (CLI, Desktop, VS Code), the script detects the existing `pk` install and exits immediately — no download, no PATH change.

The script is pinned to the version of `pk` that ran `pk setup`. After upgrading plankit, re-run `pk setup` to update the pinned version. `pk version` warns when the pinned version falls behind the running version.
