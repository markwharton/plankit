# Adoption

plankit is adopted in layers. Layer 1 is the foundation; Layers 2 and 3 are independent capabilities you add when your project needs them.

```mermaid
graph TD
    P["Prerequisites<br/>Claude Code + Git + pk"]
    L1["Layer 1: Foundation<br/>rules, skills, hooks"]
    L4["Layer 4: Migration<br/>from existing tools"]

    P --> L1

    subgraph " "
        direction LR
        L2["Layer 2: Branch Protection<br/>guard.branches in .pk.json"] --> L3["Layer 3: Release Management<br/>baseline tag + /ship"]
    end

    L1 --> L2
    L1 --> L3
    L3 --> L4

    style P fill:#f0f0f0,stroke:#999
    style L1 fill:#d4edda,stroke:#28a745
    style L2 fill:#cce5ff,stroke:#0d6efd
    style L3 fill:#cce5ff,stroke:#0d6efd
    style L4 fill:#fff3cd,stroke:#ffc107
```

## Prerequisites

- **[Claude Code](https://code.claude.com) provides the full experience.** Hooks, rules, skills, and `/ship` all run inside Claude Code. The release CLI (`pk changelog`, `pk release`) also works standalone from any terminal.
- **Git is required.** `pk setup` refuses to install outside a git working tree by default. Pass `--allow-non-git` for the narrow case where only rules and skills are needed before `git init`.
- **pk is required for hook features.** Install via `go install github.com/markwharton/plankit/cmd/pk@latest` or download a binary from the [releases page](https://github.com/markwharton/plankit/releases). See [When pk is not installed](#when-pk-is-not-installed) for what happens without it.

## Layer 1: Foundation

One command installs the full foundation:

```bash
cd your-project
pk setup
# restart Claude Code
```

**What you get:**

- **CLAUDE.md** with critical rules that prevent the most common issues
- **`.claude/rules/`** with detailed guidelines: model behavior, development standards, git discipline
- **Three skills:** `/init` (project conventions), `/preserve` (plan preservation), `/ship` (release workflow)
- **Hooks:** branch guard, plan protection, plan preservation

**No configuration needed.** No `.pk.json`, no tags, no additional setup. Guard is installed but is a no-op without branches to protect. Preserve works in manual mode by default; pass `--preserve auto` to `pk setup` to preserve plans automatically on exit from plan mode.

**Safe for existing projects.** `pk setup` never overwrites files it didn't create. Files without pk's SHA marker are skipped. Existing hooks in `.claude/settings.json` are preserved. See [Managed file protection](pk-setup.md#managed-file-protection) for details.

**Next step:** Run `/init` inside Claude Code to add project-specific conventions to CLAUDE.md. Without project conventions, Claude follows the rules but rediscovers the project each session. With them, it knows the project from the start. See [Customize your CLAUDE.md](pk-setup.md#customize-your-claudemd).

## Layer 2: Branch protection

**When to add:** You have a branch (typically `main`) that should never receive direct commits.

Create `.pk.json` in the project root:

```json
{
  "guard": {
    "branches": ["main"]
  }
}
```

`pk guard` now blocks git mutations on `main` during Claude Code sessions. The default mode is `block`; pass `--guard ask` to `pk setup` to prompt instead. See [pk guard](pk-guard.md) for details. `/init` offers to create this configuration for you.

**Server-side complement.** `pk guard` protects the local Claude Code session. A GitHub Ruleset covers the surfaces guard can't reach: pull requests, direct pushes via the GitHub UI, and other collaborators' machines. See [Branch protection](branch-protection.md) for an importable ruleset.

## Layer 3: Release management

**When to add:** You want automated changelogs and a structured release workflow.

This layer builds on three conventions:

- [Conventional Commits](https://www.conventionalcommits.org/) for commit message structure
- [Keep a Changelog](https://keepachangelog.com/) for CHANGELOG.md format
- [Semantic Versioning](https://semver.org/) for version numbering

**Tags are the version source of truth.** plankit reads the version from git tags, not from a version field in package.json, Makefile, or any other project file. This is universal across project types: Go, Node, Python, monorepos where all packages share a version. One source, no files to keep in sync.

**Anchor a baseline tag.** `pk changelog` reads commits since the most recent semver tag. Without one, it has nothing to diff from. Run `pk setup --baseline` to create a `v0.0.0` tag. See [Baseline tag for pk changelog](pk-setup.md#baseline-tag-for-pk-changelog) for the three common scenarios.

**Add release configuration to `.pk.json`:**

```json
{
  "guard": {
    "branches": ["main"]
  },
  "release": {
    "branch": "main"
  }
}
```

`release.branch` is the key config. `changelog.types` has sensible defaults (feat, fix, refactor, etc.) and only needs to be added if your project requires custom type-to-section mapping. See [pk changelog](pk-changelog.md#configuration) for the full reference. `/init` offers to create this configuration for you.

**`/ship` is the recommended release workflow.** It chains `pk changelog` and `pk release` with preview and confirm at each step, and handles the clean working tree requirement within the Claude session. Power users can run `pk changelog` and `pk release` directly in the terminal. See [pk release](pk-release.md#workflows) for merge flow vs. trunk flow.

**GitHub CLI is optional but useful.** pk commands use git directly, not `gh`. But `gh` helps with the workflow around releases: monitoring CI runs, creating PRs, checking workflow status. See [Resources](resources.md#github-cli) for common commands.

## Layer 4: Migration

**When:** Switching from commit-and-tag-version, standard-version, semantic-release, or similar tools.

**Existing CHANGELOG.md.** `pk changelog` writes [Keep a Changelog](https://keepachangelog.com/) format. It appends new entries from the baseline tag forward, preserving prior content below.

**Baseline tag placement.** Where you anchor `v0.0.0` determines what appears in your first pk-generated changelog entry. Use `--at` to include prior history or omit it to start fresh. See [Baseline tag for pk changelog](pk-setup.md#baseline-tag-for-pk-changelog) for the three scenarios.

**Commit type mapping.** The default types cover the standard Conventional Commits set. Add `changelog.types` to `.pk.json` only if your project uses custom types or needs different section names. See [pk changelog](pk-changelog.md#configuration).

**NPM projects.** Replace existing release scripts in `package.json`:

```json
{
  "scripts": {
    "release": "pk changelog && pk release",
    "release:dry": "pk changelog --dry-run"
  }
}
```

**Remove the old tool.** Disable or uninstall the previous release tool from CI and dev dependencies before switching, to avoid conflicting tag or changelog writes.

## When pk is not installed

When a developer clones a pk-configured repo without pk installed, hooks degrade gracefully:

| Feature | Without pk |
|---------|------------|
| CLAUDE.md | Works, Claude Code reads it regardless |
| `.claude/rules/` | Works, loaded automatically |
| `/init`, `/preserve`, `/ship` skills | Skills load but pk commands inside them fail |
| `pk guard` hook | Silent no-op, hook exits 127 (non-blocking) |
| `pk preserve` hook | Silent no-op, plans not preserved |
| `pk protect` hook | Silent no-op, plan edits not blocked |

**Cloud sandboxes are handled automatically.** The SessionStart hook runs `.claude/install-pk.sh`, which downloads the pinned pk version at session start. No action needed.

**Local machines get a session-start warning.** When pk is not on PATH, the SessionStart hook prints a warning with install instructions to stderr. Rules and skills still guide Claude, but the protective net (guard, preserve, protect) won't run until pk is installed.
