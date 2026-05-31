<!--
Advisory, not a gate. Delete sections that don't apply.
See CONTRIBUTING.md for the workflow (develop-first, conventional commits, no squash).
-->

## What & why

<!-- For substantive or behavior-changing PRs, describe the intent and approach, not just the diff. -->

## Conventional commit type

<!-- feat | fix | deprecate | revert | security | refactor | perf | docs | chore | test | build | ci | style -->

## Checklist

- [ ] `make test`, `make lint`, and `make vuln` pass
- [ ] No new third-party Go dependency (standard library only)
- [ ] If a managed file was touched (`internal/setup/rules/`, `internal/setup/skills/`, `internal/setup/template/CLAUDE.md`): `pk_sha256` updated and the embed-safety check passes
