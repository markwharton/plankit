# Plan: Adoption Guide for plankit

## Context

plankit's per-command docs are thorough, but no document explains the layered nature of adoption. A new user reads the README, runs `pk setup`, and gets everything at once with no guidance about which features to engage first, which are optional, or what conventions they need to follow at each level. The information exists but is scattered across README.md, pk-setup.md, pk-changelog.md, pk-release.md, methodology.md, and branch-protection.md.

This matters for three audiences: someone starting a new project, someone integrating plankit into an existing project, and someone migrating from another release tool (e.g., commit-and-tag-version). Each needs a different subset of plankit, and each has different prerequisites.

Additionally, the "pk not installed" developer bootstrapping story has a known gap: on local machines, hooks silently degrade (exit 127, non-blocking), and there's no runtime warning beyond the initial `pk setup` PATH check. This should be documented honestly.

## Approach

Create `docs/adoption.md` and update `README.md` to reference it. The document describes four adoption layers, each building on the previous, with clear prerequisites and quickstart blocks. Links to existing command docs rather than duplicating content.

## New file: `docs/adoption.md`

### Structure

```
# Adoption
  (summary paragraph + Mermaid layer diagram)

## Prerequisites
  - Claude Code (hard)
  - Git (hard, --allow-non-git for edge cases)
  - pk installed (hard for hook features, graceful degradation without it)

## Layer 1: Foundation
  - What pk setup gives you: CLAUDE.md, rules, skills, hooks
  - No .pk.json needed, no tags needed
  - Safety: files without pk SHA markers are never overwritten
  - Next step: /init for project-specific conventions
  - Quickstart block

## Layer 2: Branch protection
  - When: you have a branch that shouldn't receive direct commits
  - .pk.json with guard.branches
  - Guard modes (block vs ask)
  - Server-side complement: GitHub Rulesets (link to branch-protection.md)
  - Quickstart: minimal .pk.json

## Layer 3: Release management
  - When: you want automated changelogs and structured releases
  - Conventions: Conventional Commits, Keep a Changelog, Semantic Versioning
  - Baseline tag (pk setup --baseline)
  - .pk.json: release.branch is the key config; changelog.types has sensible defaults and only needs to be added for customization
  - /ship is the recommended release workflow: handles preview, confirm, and the clean working tree requirement within the Claude session. Power users can run pk changelog and pk release directly in the terminal.
  - Merge flow vs trunk flow (link to pk-release.md)
  - GitHub CLI optional (pk prints git commands as fallback)
  - Quickstart: minimal .pk.json (guard + release.branch, no changelog.types unless customizing)

## Layer 4: Migration
  - Existing CHANGELOG.md: pk appends in Keep a Changelog format from baseline forward
  - Baseline tag placement: three --at scenarios (link to pk-setup.md)
  - Commit type mapping via changelog.types (only if the project's conventions differ from defaults)
  - NPM projects: replace existing scripts (e.g., "release": "pk changelog && pk release", "release:dry": "pk changelog --dry-run")
  - Remove the old tool from CI before switching

## When pk is not installed
  - Table: what works (CLAUDE.md, rules) vs what silently degrades (guard, protect, preserve)
  - Cloud sandboxes: install-pk.sh auto-bootstraps
  - Local machines: no runtime warning, hooks exit 127 non-blocking
  - Mitigation: document pk as prerequisite in project README/CONTRIBUTING
```

### Mermaid diagram

Flowchart showing the layers. Layer 2 and Layer 3 branch independently from Layer 1 (neither requires the other). Layer 4 follows from Layer 3 only.

### Conventions for Layer 3 (per user's addition)

The release management layer builds on three established conventions:
- [Conventional Commits](https://www.conventionalcommits.org/) for commit message structure
- [Keep a Changelog](https://keepachangelog.com/) for CHANGELOG.md format
- [Semantic Versioning](https://semver.org/) for version numbering

These are already referenced in plankit's generated output (CHANGELOG.md header, pk-changelog.md docs, version parsing code) but not called out as prerequisites for the adoption layer. The adoption guide makes this explicit.

### Style

- Bold principle then concise context (plankit doc convention)
- Quickstart code blocks at each layer
- Links to command docs for details, no duplication
- Concise: target ~130-150 lines

## Modified file: `README.md`

Add adoption guide as first item in the Documentation section:

```markdown
## Documentation

- [Adoption](docs/adoption.md): layered adoption from foundation to release management
- [Methodology](docs/methodology.md): plans, guidelines, compounding effect, model resilience
...
```

## Design decisions

- **Conventions stay in adoption guide only.** Keep a Changelog, Semantic Versioning, and Conventional Commits are linked in Layer 3 where they're contextually relevant. resources.md stays focused on Claude Code, Git, and cloud frameworks.
- **"pk not installed" gap: document honestly and note future improvement.** State what happens, explain why (non-blocking by design), suggest the README/CONTRIBUTING mitigation. Also note that a lighter-weight session check (e.g., a message when hooks can't find pk) could improve the developer experience in a future release.

## What this plan does NOT do

- Does not change any Go code or pk behavior
- Does not modify the new-plankit-project skill (that's personal automation, not general adoption guidance)
- Does not attempt to fix the "pk not installed" gap in code

## Verification

1. `docs/adoption.md` renders correctly on GitHub (Mermaid diagram, links, tables)
2. All internal links resolve to existing doc sections
3. README.md Documentation section updated
4. `make lint` still passes (no Go changes, but verify no stale references)
5. Smoke test: read the doc as a new user and confirm each layer's quickstart is self-contained

## Critical files

- `docs/adoption.md` (new)
- `README.md` (update Documentation section, line 72-76)
- Cross-references (read-only, link targets):
  - `docs/pk-setup.md` (baseline, managed files, modes)
  - `docs/pk-changelog.md` (types, conventional commits)
  - `docs/pk-release.md` (workflows, merge vs trunk)
  - `docs/branch-protection.md` (GitHub Rulesets)
  - `docs/methodology.md` (philosophy)
