# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

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
