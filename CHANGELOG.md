# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [v0.19.6] - 2026-05-14

### Fixed

- resolve .pk.json and file paths from git root (77a57fa)

## [v0.19.5] - 2026-05-13

### Fixed

- rename /init to /conventions, reorder steps, label trunk flow (8419a2c)

## [v0.19.4] - 2026-05-12

### Fixed

- check branch exists on origin before committing (8896ce3)
- exit plan mode before executing action skills (6e9bd97)

## [v0.19.3] - 2026-05-11

### Fixed

- suggest the right fix for pinned version mismatch (0e87bc3)
- skip non-semver tags instead of failing (ded9235)

## [v0.19.2] - 2026-05-09

### Changed

- split setup.go into focused files by provider boundary (8b0dc6b)

### Documentation

- add reference docs for .pk.json, errors, and environment variables (fcfc2d6)

## [v0.19.1] - 2026-05-08

### Fixed

- address code review findings across correctness, shared config, and DRY (55e67e7)

## [v0.19.0] - 2026-05-07

### Added

- add private repo support to new-plankit-project skill (4c5441e)

### Fixed

- restore conversational question style (2019654)
- skip SessionStart hook for dev builds (3b7016c)

### Documentation

- add skill authoring rule for conversational questions (77d6699)

## [v0.18.0] - 2026-05-06

### Added

- add auto mode to skip confirmations when dry-runs pass (b4ecf47)

### Documentation

- add versioning guide (11b7caf)

## [v0.17.0] - 2026-05-05

### Added

- add --name flag for flexible version pinning (c422baa)

### Fixed

- remove /init hint from install-pk.sh (c691c7c)

## [v0.16.3] - 2026-05-05

### Fixed

- skip baseline tag when repo has no commits (179df9d)

### Maintenance

- hint to run /init when CLAUDE.md is still the template (c4c5637)

## [v0.16.2] - 2026-05-04

### Fixed

- exit 1 when pk is not installed so Claude Code surfaces the warning (aa94a4e)

### Documentation

- reframe messaging from deterministic outcomes to narrowing the solution space (ad7fea7)

## [v0.16.1] - 2026-05-03

### Fixed

- warn when pk is not installed on local machines (57fe9fc)

### Documentation

- add adoption guide (4718081)

## [v0.16.0] - 2026-05-02

### Added

- preserve guard and preserve modes on re-run (dff2a05)
- create Critical Rules header when CLAUDE.md is missing (231ba72)

### Fixed

- pre-expand hook variables for cross-platform compatibility (5cd06ca)

### Documentation

- fix "templates" in README, restore "When the model shifts" section (1001727)

### Maintenance

- add rules for git history rewrites and rule accountability (2b451af)

## [v0.15.3] - 2026-05-01

### Documentation

- condense methodology from 14 to 9 sections (d7ba140)
- update README with best practices link and cleaner formatting (cc8f598)
- limit release-notes skill to feat, security, and breaking changes (5b9d2c0)

### Maintenance

- add RunScript coverage (2fc0909)
- tighten rules files, remove all em dashes (91632ac)
- add rules for surfacing hook outcomes and system-reminder failures (c09e0b3)

## [v0.15.2] - 2026-04-30

### Fixed

- use cmd.exe for lifecycle hooks on Windows (ca65529)

## [v0.15.1] - 2026-04-30

### Documentation

- add bullet for commit-producing automation (bbf4122)
- condense methodology and remove getting-started (511f13b)
- add silent semantic narrowing to anti-patterns (0b0fcdb)

### Maintenance

- cover prune-helper directory and skip branches (d78f904)
- add chore(deps) prefix to dependabot config (03e5b9a)
- simplify skills to /init, /preserve, and /ship (82c3118)

## [v0.15.0] - 2026-04-28

### Added

- prune skills and rules no longer in the embed set (4750772)

### Changed

- rename "legacy" to "trunk" in code + tests (334f4ea)

### Documentation

- add "discipline as the multiplier" section to methodology (9d8b5cd)
- note Esc as the keyboard interrupt in "breaking the loop" (753239f)
- document the prune-on-setup behavior (cb0c2ea)
- make /ship workflow-agnostic (caeceb8)
- rename legacy flow → trunk flow and surface both in examples (238c694)
- doc pass — Reviewing the plan + four tightenings (0ccece6)

## [v0.14.3] - 2026-04-25

### Fixed

- clear legacy $HOME/.local/bin/pk before the install gate (be2006d)

### Documentation

- explain the command -v pk gate after the legacy rm (6f8b203)
- tighten the command -v pk gate comment (2e92d71)

## [v0.14.2] - 2026-04-25

### Fixed

- install pk per-version to prevent stale-binary leak (ddc5a14)

### Maintenance

- make commit-message tip self-describing (a6e4ac3)

## [v0.14.1] - 2026-04-24

### Fixed

- fetch tags at session start so pk changelog sees history (6d94c1d)

### Documentation

- note pk changelog --undo in ship and changelog flows (88c9abb)

## [v0.14.0] - 2026-04-23

### Added

- suggest commit message after managed-file updates (ee84a5d)

### Fixed

- surface malformed hooks JSON to stderr (5a2ace9)

### Security

- verify pk binary checksum against published manifest (c30f3f4)

### Changed

- drop legacy protectedBranches config shim (82ce7c1)

### Documentation

- add "when the model shifts" section to methodology (7315fa8)
- unify (bool, error) doc-comment style (8b2b62b)

## [v0.13.1] - 2026-04-22

### Fixed

- close manual-mode race with pointer file (186d73e)
- preserve settings.json key order across pk setup (1813b68)
- preserve unknown fields on user hook objects (8b2d05d)

### Changed

- drop dry-run preview from /preserve skill (4975ffe)

### Documentation

- refresh stale enumerations (8802dca)

## [v0.13.0] - 2026-04-21

### Added

- add managed /ship skill chaining changelog and release (82492bd)

### Maintenance

- bump codecov/codecov-action from 5.5.4 to 6.0.0 (d3c2e20)
- parameterize tool slug for multi-tool repos (6147eff)

## [v0.12.0] - 2026-04-19

### Added

- nudge pk setup --baseline when no tag exists and versioned releases planned (0ec2df9)
- suggest pk setup alongside install in update notice (f423f82)
- add maintainer skill for plankit.com notes prompts (1d1d9d9)
- add maintainer skill for project scaffolding (eee9a47)

### Fixed

- detect non-git repo up front with a friendly error (247581a)
- normalize Version() output to bare semver (6460ca3)
- create and push develop at init (052d33c)
- clear error when source branch is not on origin (4fc914a)

### Changed

- split maintainer-side rules out of plankit-tooling.md (b63e73a)

### Documentation

- add branch protection ruleset template and companion guide (5e1727e)
- document repo-check heuristic in plankit-development.md (1de8eb2)

### Maintenance

- read Go version from go.mod instead of pinning in workflows (a564a66)
- apply gofmt across six files (c4e160f)
- fail make lint on gofmt drift (3fa6a16)

## [v0.11.1] - 2026-04-19

### Fixed

- soften version-tag tip and surface --at in changelog error (ed97860)

### Documentation

- add --baseline to setup options enumerations (4df4838)
- add Clarifications, Evolving pk Commands, and Tip Messages (255ce88)

### Maintenance

- remove site/ and pages.yml for repo move to plankit.com (190d0b8)

## [v0.11.0] - 2026-04-18

### Added

- continue implementation after preserving the plan (656cb34)
- add --baseline flag to anchor pk changelog (44688ed)

### Documentation

- harmonize footer label and fix rules listing (b12e4a6)
- recommend pk setup --baseline on the start page (f247dbe)
- match commit message weight to change weight (a2e865a)
- protect CHANGELOG format from tooling drift (c76943e)
- clarify tag-as-source and add monorepo example (931f06f)
- establish --push and --at flag conventions (931d425)

## [v0.10.1] - 2026-04-17

### Documentation

- update canonical URLs and og tags to plankit.com (91c3b37)

### Maintenance

- add CNAME file claiming plankit.com for GitHub Pages (6b528b0)

## [v0.10.0] - 2026-04-16

### Added

- block by default, add --ask flag for user confirmation (b4f286e)
- add plankit-tooling rule for pk-setup projects (fb6ff05)
- flag unpinned GitHub Actions and missing Dependabot config (f6cb4b8)
- add pk teardown command to remove plankit from a project (3fd5296)
- add pk status command and non-git awareness (9ce9dc0)
- split pk feature configuration into three opt-in questions (685a418)

### Fixed

- validate version argument as semver (42b6355)
- add target verification step to soft reset guidance (0576063)
- preserve unknown hook categories and hook fields during merge (97ce733)

### Documentation

- align workflow steps across site and docs (fb366aa)
- add /rollback tutorial and harden /preview (8ec000a)
- add SEO metadata, design note, and README positioning line (3b8f95d)
- add --guard option to setup examples and clarify doc update trigger (723f8e5)
- document three command layers and their flag patterns (3b9a8bd)
- restructure site with /pk home and simplified landing (1f66617)

## [v0.9.0] - 2026-04-13

### Added

- add pk pin command for automated version pinning (3073f0a)

### Fixed

- enable cgo for race detector in test target (4803a9b)

### Documentation

- add SessionStart note, PR merge example, site bootstrap tree (2670df6)
- add git pull --rebase guidance for local sync (07b1b76)

### Maintenance

- bump codecov/codecov-action (b71720b)
- bump softprops/action-gh-release from 2.6.1 to 3.0.0 (1484d2f)

## [v0.8.0] - 2026-04-13

### Added

- bootstrap pk into cloud sandboxes via SessionStart hook (5f3bc4f)

### Documentation

- add /start and /guide pages, link from landing page (d2be5e1)
- update branch references from dev to develop (a12e318)
- target dependabot at develop, merge main for releases (dae5dc2)
- add git mental model and PR merge guidance (4c0e365)

### Maintenance

- bump actions/upload-artifact from 7.0.0 to 7.0.1 (4488d0c)
- bump actions/upload-pages-artifact from 4.0.0 to 5.0.0 (183348f)
- add SessionStart hook and bootstrap script (e2c53c1)
- add CI workflow with test, lint, and Codecov coverage (878fb89)

## [v0.7.1] - 2026-04-12

### Changed

- deduplicate trailer reading, clean-tree, hooks, and config types (7c5db28)

### Documentation

- tighten review-staged example, rename review→review-code (b63c80e)
- note hook degradation on web/mobile (6455a0c)
- use human-readable URL form for Claude Code docs (764167a)
- restore verification in development-standards description (dbc3307)

### Maintenance

- enforce CGO_ENABLED=0 for pure-Go static binaries (0dc3243)

## [v0.7.0] - 2026-04-11

### Added

- implement Semver 2.0.0 spec for version parsing and comparison (7db188e)
- add showScope option to include conventional commit scope in changelog (72e2609)
- harden managed skill permissions (4bca4c6)
- add soft-reset rule for rewriting unpushed commits (149661b)
- show commit SHA and dirty state in verbose output (5da2af9)
- add --undo to unwind an unpushed release commit (1ada6cc)
- add --exclude to drop commits from the release section (677b92f)

### Fixed

- reject dirty working tree in pk changelog (faaa8a1)
- include required hookEventName in hookSpecificOutput (dee71c6)

### Changed

- extract parseRepoURL to internal/git package (283423b)
- move tag creation from pk changelog to pk release via Release-Tag trailer (6098ca6)
- migrate to hookSpecificOutput.permissionDecision (4f0b631)
- migrate to hookSpecificOutput.permissionDecision (ask) (7a5f2d6)

### Documentation

- document release workflows, squash merge warning, new flags (b51fc7e)
- rewrite plan review section, add session chaining and failure examples (c05ebc3)
- replace Franklin quote with Lakein and add HBR and Horowitz quotes (dc1afb4)
- add site/humans.txt (7d28c38)
- add command doc template and normalize command docs (b6059ea)
- add Creating skills guide (23f38a5)
- add managed-files update guidance (7157362)
- add exploration-becomes-editing example to methodology (5446a3e)
- restructure Release workflows and clarify --push scope (73847ea)
- refine site, README, and methodology (review refresh) (5829dc9)
- update command docs, skills, and README for trailer-driven release flow (0753f12)
- add two-tier verification standard to development-standards (e1fbdca)
- add /preview tutorial example to creating-skills (a64a02f)

### Maintenance

- add preserve tests for push failure, dry-run push, and Getwd fallback (dcc0e08)
- add preserve error path tests for git and directory fallback (0702042)
- add coverage for git URL formats, large hook payloads, and cache TTL boundary (0edac81)
- add guard rev-parse failure and release tag-lost-after-merge tests (687a7dc)

## [v0.6.2] - 2026-04-08

### Fixed

- error on malformed .pk.json instead of silent fallback to defaults (073afec)

### Changed

- move preRelease hook from changelog to release config (37b9e72)
- eliminate DRY violations, harden security, unify design patterns (f8b6cba)

### Maintenance

- remove pk_sha256 marker from user-managed review skill (90c198c)

## [v0.6.1] - 2026-04-08

### Fixed

- prevent ReadInput from blocking when stdin is a terminal (cc5415a)

### Documentation

- add real-world example of guidelines being ignored (5f5d183)

## [v0.6.0] - 2026-04-08

### Added

- pk release merges to release branch, bypassing guard (d408632)

### Fixed

- handle compound commands in pk guard (bd37130)

### Documentation

- add release flow and GitHub CLI commands to resources (c05588b)
- add prerequisites and update introductory flow (af50f7a)
- update site with stronger narrative and substance (60934c2)
- fix stale terminology in methodology.md (2f5e243)

## [v0.5.0] - 2026-04-07

### Added

- broaden init prompt to discover business and domain rules (d7ae137)
- default pk preserve to commit-only, add --push flag (aa039a4)
- add plan as hidden commit type in changelog config (1f80bd6)
- slim CLAUDE.md template, move guidelines to .claude/rules/ (e754db8)
- add pk guard to block git mutations on protected branches (063c834)
- ship /init as managed skill, remove inline prompt from docs (c047940)
- add project conventions and guard config for protected branches (55a993d)

### Fixed

- merge hooks instead of replacing in pk setup (4319731)
- show skill name in setup output, apply new template and hooks (eef51e4)

### Changed

- rename .changelog.json to .pk.json with nested structure (123e19c)

### Documentation

- add release skill guardrail and documentation conventions (29d6347)
- add plankit.life landing page and GitHub Pages deployment (01fadea)
- add commit/push guideline and fix convention format wording (0b0bbad)
- add Benjamin Franklin quote to site references (234de37)
- list plan as a hidden type in CLAUDE.md conventions (30381e4)
- add pk guard documentation and update architecture (1330f1b)
- add resources.md for external references (01209af)
- add rules conversation with Claude Code self-assessment (a48e4f6)
- fix completeness and consistency across pk command docs (3e0adcd)

## [v0.4.0] - 2026-04-06

### Added

- use plan: commit type for preserved plans, exclude from changelog (26b1e6d)
- add --dry-run to pk preserve (32898aa)
- add additionalContext to notify hook and elevate grep-before-done guideline (ca86724)

### Fixed

- force IPv4 in update checker to avoid AAAA DNS timeouts (c5f743b)
- correct version ldflags case in CI and clean up code review findings (96e6790)
- use %v for all error formatting in release.go (6128e2d)

### Changed

- simplify plan commit message to just the title (ac13310)
- remove templates and non-core skills (init, review) (a3d29f9)

### Documentation

- restructure methodology and rewrite plan review section (aeb4449)
- add dogfooding term to methodology (a1095b7)
- document duplicate sequence numbers in team usage (657f181)

### Maintenance

- bump actions/upload-artifact from 4.6.2 to 7.0.0 (023ac22)
- bump actions/download-artifact from 4.3.0 to 8.0.1 (4fd4858)
- bump actions/checkout from 4.3.1 to 6.0.2 (386b3a0)
- remove plan files for re-commit with plan: type (411b2bf)
- bump actions/setup-go from 5.6.0 to 6.4.0 (5e7752c)
- ignore stray pk binary in project root (194be67)

## [v0.3.0] - 2026-04-05

### Added

- add --verbose flag to pk version for build details (1c82c00)
- add universal CLAUDE.md, /init skill, managed file protection, and command docs (1ea9324)

### Fixed

- remove duplicate changelog section and guard against duplicate ref links (0e623a1)
- pin GitHub Actions to commit SHAs and add Dependabot for actions (6e34030)
- default preserve mode to manual (opt-in for auto) (5bafe8f)
- add push hint to no-tags error message in pk changelog (4ba0936)

### Documentation

- warn against commits between changelog and release steps (87273a9)
- add Known Limitations section for Ultraplan compatibility (55404ef)
- clarify install-to-setup flow and reorder methodology sections (926cfd2)
- merge universal guidelines into plankit's own CLAUDE.md (74a3bee)
- add 'use what you build' section to methodology (e345d22)

### Maintenance

- add preserve compatibility tests for root home and JSON tool_response (bc71e90)
- add changelog and release skills from pk setup (8ef944e)
- update installed skills via pk setup --force (c223b4e)

## [v0.2.0] - 2026-04-04

### Added

- add comparison links to changelog and update documentation (a8d41b6)
- add pk release subcommand and template extensibility (ed70067)
- add /release skill and tighten documentation (4a3c934)

### Fixed

- use debug.ReadBuildInfo() for version in go install path (6667259)
- add dry-run preview to /changelog skill before committing (fc3b2fd)
- standardize --dry-run flag in release.sh to match pk convention (b49da72)
- use synchronous hook for manual preserve --notify mode (5a220e0)
- use Tagged instead of Released and compact reference links (38d0d6a)

### Documentation

- add changelog documentation and fix skill file consistency (4999270)
- articulate why guidelines matter for LLM-assisted development (be8f171)
- connect unbounded prompts to template constraints in methodology (5da5a5f)

### Maintenance

- release v0.2.0 (717f95a)

## [v0.1.0] - 2026-04-03

### Added

- add build/release workflow and contributing guide (132f6da)
- add project hooks, skills, and gitignore for backup files (adfca0f)
- add pk changelog command and /changelog skill (c3f48d8)

### Fixed

- add 2s HTTP timeout to update checker (b625b53)

### Changed

- embed skill files via go:embed instead of hardcoded strings (45d5824)

### Documentation

- add skills, templates, methodology docs, and README (e147405)

[v0.1.0]: https://github.com/markwharton/plankit/compare/v0.0.0...v0.1.0
[v0.2.0]: https://github.com/markwharton/plankit/compare/v0.1.0...v0.2.0
[v0.3.0]: https://github.com/markwharton/plankit/compare/v0.2.0...v0.3.0
[v0.4.0]: https://github.com/markwharton/plankit/compare/v0.3.0...v0.4.0
[v0.5.0]: https://github.com/markwharton/plankit/compare/v0.4.0...v0.5.0
[v0.6.0]: https://github.com/markwharton/plankit/compare/v0.5.0...v0.6.0
[v0.6.1]: https://github.com/markwharton/plankit/compare/v0.6.0...v0.6.1
[v0.6.2]: https://github.com/markwharton/plankit/compare/v0.6.1...v0.6.2
[v0.7.0]: https://github.com/markwharton/plankit/compare/v0.6.2...v0.7.0
[v0.7.1]: https://github.com/markwharton/plankit/compare/v0.7.0...v0.7.1
[v0.8.0]: https://github.com/markwharton/plankit/compare/v0.7.1...v0.8.0
[v0.9.0]: https://github.com/markwharton/plankit/compare/v0.8.0...v0.9.0
[v0.10.0]: https://github.com/markwharton/plankit/compare/v0.9.0...v0.10.0
[v0.10.1]: https://github.com/markwharton/plankit/compare/v0.10.0...v0.10.1
[v0.11.0]: https://github.com/markwharton/plankit/compare/v0.10.1...v0.11.0
[v0.11.1]: https://github.com/markwharton/plankit/compare/v0.11.0...v0.11.1
[v0.12.0]: https://github.com/markwharton/plankit/compare/v0.11.1...v0.12.0
[v0.13.0]: https://github.com/markwharton/plankit/compare/v0.12.0...v0.13.0
[v0.13.1]: https://github.com/markwharton/plankit/compare/v0.13.0...v0.13.1
[v0.14.0]: https://github.com/markwharton/plankit/compare/v0.13.1...v0.14.0
[v0.14.1]: https://github.com/markwharton/plankit/compare/v0.14.0...v0.14.1
[v0.14.2]: https://github.com/markwharton/plankit/compare/v0.14.1...v0.14.2
[v0.14.3]: https://github.com/markwharton/plankit/compare/v0.14.2...v0.14.3
[v0.15.0]: https://github.com/markwharton/plankit/compare/v0.14.3...v0.15.0
[v0.15.1]: https://github.com/markwharton/plankit/compare/v0.15.0...v0.15.1
[v0.15.2]: https://github.com/markwharton/plankit/compare/v0.15.1...v0.15.2
[v0.15.3]: https://github.com/markwharton/plankit/compare/v0.15.2...v0.15.3
[v0.16.0]: https://github.com/markwharton/plankit/compare/v0.15.3...v0.16.0
[v0.16.1]: https://github.com/markwharton/plankit/compare/v0.16.0...v0.16.1
[v0.16.2]: https://github.com/markwharton/plankit/compare/v0.16.1...v0.16.2
[v0.16.3]: https://github.com/markwharton/plankit/compare/v0.16.2...v0.16.3
[v0.17.0]: https://github.com/markwharton/plankit/compare/v0.16.3...v0.17.0
[v0.18.0]: https://github.com/markwharton/plankit/compare/v0.17.0...v0.18.0
[v0.19.0]: https://github.com/markwharton/plankit/compare/v0.18.0...v0.19.0
[v0.19.1]: https://github.com/markwharton/plankit/compare/v0.19.0...v0.19.1
[v0.19.2]: https://github.com/markwharton/plankit/compare/v0.19.1...v0.19.2
[v0.19.3]: https://github.com/markwharton/plankit/compare/v0.19.2...v0.19.3
[v0.19.4]: https://github.com/markwharton/plankit/compare/v0.19.3...v0.19.4
[v0.19.5]: https://github.com/markwharton/plankit/compare/v0.19.4...v0.19.5
[v0.19.6]: https://github.com/markwharton/plankit/compare/v0.19.5...v0.19.6
