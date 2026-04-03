# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

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

### Documentation

- add changelog documentation and fix skill file consistency (4999270)
- preserve approved plan -- Template extensibility + `pk release` [skip ci] (7864046)
- articulate why guidelines matter for LLM-assisted development (be8f171)
- connect unbounded prompts to template constraints in methodology (5da5a5f)

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
