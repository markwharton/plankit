# plankit

[![CI](https://github.com/markwharton/plankit/actions/workflows/ci.yml/badge.svg)](https://github.com/markwharton/plankit/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/markwharton/plankit/graph/badge.svg?token=y1SS0kyj3v)](https://codecov.io/gh/markwharton/plankit)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**`pk` is a command-line tool for repositories worked on with [Claude Code](https://code.claude.com). It installs rules and skills, preserves approved plans as records, blocks git mutations on protected branches, and releases from the git tag. One Go binary; macOS, Linux, and Windows (Git Bash).**

## What it does

`pk setup` installs five things into a repository:

- **Rules:** `CLAUDE.md` with critical rules, and `.claude/rules/plankit/` covering craft, conduct, and documentation.
- **Skills:** `/pk-configure`, `/preserve`, `/ship`.
- **Plan preservation:** approved plans saved to `docs/plans/`, committed, and protected from edits.
- **Branch protection:** git mutations blocked by hooks, locally, before the commit or push exists.
- **Release management:** `CHANGELOG.md` from commits, and tagged releases.

<!-- shipped-footprint:start -->
Always-on rules footprint: ≈2,602 tokens (estimated, calibrated against claude-fable-5) for the rules and CLAUDE.md `pk setup` installs, loaded every session. Your edits and added rules change it; run `pk rules` for your own estimate.
<!-- shipped-footprint:end -->

## Install

```bash
brew tap markwharton/plankit && brew install plankit   # Homebrew, macOS and Linux
go install github.com/markwharton/plankit/cmd/pk@latest # Go
```

Or download a binary from the [releases page](https://github.com/markwharton/plankit/releases). Run `pk` from a terminal, not by double-clicking the binary.

## Use

```bash
cd your-project
pk setup        # existing repository: hooks, skills, rules, CLAUDE.md
pk init --push  # new repository: main + develop, managed files, v0.0.0, ruleset
```

Restart Claude Code after setup. The release flow is `/ship`, which runs `pk changelog` then `pk release` with a preview and confirmation at each step.

`pk setup` starts in trunk flow: `pk release` tags the default branch and pushes. Adding `release.branch` to `.pk.json` switches to merge flow, where `main` advances only by release. Modes and flags: [pk setup](docs/pk-setup.md); configuration: [.pk.json](docs/pk-json.md); established repositories: [Adoption](docs/adoption.md).

## Commands

| Command | Does |
|---------|------|
| `pk init` | Make a repository plankit-shaped: topology, managed files, `v0.0.0`, branches. [Details](docs/pk-init.md) |
| `pk setup` | Install hooks, skills, rules, and CLAUDE.md; `--baseline` anchors `pk changelog`. [Details](docs/pk-setup.md) |
| `pk status` | Report configuration state and release readiness. [Details](docs/pk-status.md) |
| `pk rules` | Report the always-on context footprint of `.claude/rules/` and CLAUDE.md; `--lint` scans for hidden characters. [Details](docs/pk-rules.md) |
| `pk teardown` | Remove plankit hooks, skills, and rules. [Details](docs/pk-teardown.md) |
| `pk changelog` | Write CHANGELOG.md from commits and commit it; the tag comes from `pk release`. [Details](docs/pk-changelog.md) |
| `pk release` | Validate, merge to the release branch, tag, and push. [Details](docs/pk-release.md) |
| `pk guard` | Block git mutations on protected branches; guard `git push`. [Details](docs/pk-guard.md) |
| `pk preserve` | Save the approved plan to `docs/plans/`. [Details](docs/pk-preserve.md) |
| `pk protect` | Block edits to `docs/plans/`. [Details](docs/pk-protect.md) |
| `pk pin` | Update a pinned version in a file. [Details](docs/pk-pin.md) |
| `pk version` | Print the version and check for updates. [Details](docs/pk-version.md) |

| Skill | Runs |
|-------|------|
| `/pk-configure` | Detects the git setup, asks about branch and release policy, writes `.pk.json` |
| `/preserve` | `pk preserve` |
| `/ship` | `pk changelog`, then `pk release`, each with preview and confirmation |

## Limits

- **Ultraplan (preview):** runs remotely and writes no local plan file, so `ExitPlanMode` does not fire and preservation does not trigger. Use `/plan`.
- **Claude Code on the web:** a SessionStart hook fetches the matching `pk` binary into the sandbox; hooks then work. Mobile has no shell, so hooks are no-ops there.
- **Windows** needs Git Bash, which Claude Code requires.

## Documentation

- The suite: [Architecture](docs/architecture.md), [Security](docs/security.md), [Design](docs/design.md), [Publishing](docs/publishing.md)
- Reference: [.pk.json](docs/pk-json.md), [Environment Variables](docs/environment-variables.md), [Error Reference](docs/error-reference.md), one `docs/pk-<command>.md` per command, [Branch protection](docs/branch-protection.md), [Changing flow](docs/changing-flow.md), [Adoption](docs/adoption.md)
- [docs/archive/](docs/archive/README.md): essays and notes kept for reference, not maintained

## License

MIT
