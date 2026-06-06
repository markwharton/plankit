---
description: Rules for building plankit itself. CHANGELOG format, command evolution, tip messages. Not shipped to plankit-using projects.
kind: craft
---

# Plankit Development

These rules apply when working *on* plankit: authoring the CLI, writing runtime messages, maintaining the CHANGELOG. They are maintainer-side and live in plankit's `.claude/rules/` only. They are NOT embedded in `pk setup` and do NOT ship to other projects.

## CHANGELOG Format

- **Plain text, one link per version.** Entries are plain `- summary (abc1234)` with no clickable commit SHAs. Each version heading (`## [v0.10.1] - 2026-04-17`) is already a clickable link to a compare URL showing every commit in that release with full context. Don't link individual commits in any form, whether inline `[sha](url)` or reference-style. Don't pull in CHANGELOG generators (commit-and-tag-version, git-cliff). plankit is "small tools, carefully made": plain text by design, not by oversight.

## Evolving pk Commands

- **Grep existing flag/mode enumerations before declaring done.** Adding or renaming a `pk` command option is a concept change, not just a command-doc update. Before finishing, grep the repo for existing option lists (`README.md`, other command docs) and add the new option where others are enumerated. The mechanical "code → tests → command doc → reference docs" loop stops at the reference docs; the grep catches the higher-level docs that the CLAUDE.md documentation rule requires updating for concept changes. Reference docs (`docs/pk-json.md`, `docs/error-reference.md`, `docs/environment-variables.md`) centralize cross-command information: config keys, error messages, and environment variables respectively. New docs in `docs/` go in the right README section: guides (adoption, methodology, versioning) under Documentation, lookup references (config schema, error messages, env vars) under Reference.
- **Adding `--push` to a new command updates the closed list in the `plankit-tooling` rule.** The Flag Conventions section names the exact commands that take `--push` (`pk setup --baseline`, `pk preserve`) and states no others do, so a downstream session never assumes a flag that isn't there. A new `--push` is therefore a concept change: add the command to that list in both the embedded source (`internal/setup/rules/plankit-tooling.md`) and the local copy (`.claude/rules/plankit-tooling.md`), then recompute the local copy's `pk_sha256` per CLAUDE.md's "Updating pk-managed files". Same idea as the code → command-docs loop: the enumeration is the source of truth, keep it current.

## Tip Messages

- **Show the git equivalent when pk is a thin wrapper.** When stderr output suggests a pk command as a next step, follow with the git equivalent on the next line when pk is a thin wrap over 1–2 git commands (tag creation, push, add/commit). Format: the pk command on one line, `or: <git commands>` on the next. Skip the git line when pk adds substantial logic (pre-flight checks, hooks, multi-step flows, commit scanning) that would be lost in a direct translation. The pattern educates, builds trust, and gives power users an escape hatch.

## Skill Authoring

- **Keep skill questions conversational.** When a skill asks the user for input, list questions as plain bullets under a short heading. Move interpretation context (which config key maps to which answer, default values, command references) to the skill's Rules section. Dense instructional text around questions causes the model to dump it all as a wall of text instead of walking through questions naturally.

## Rule Authoring

- **Managed rules split by audience: developer craft vs agent conduct, and the two never mix in one file.** Craft files state standards for the work (`development-standards.md` for code, `git-discipline.md` for git history); they are developer-voiced and Claude inherits them the way a teammate inherits house style. Conduct files state how the agent behaves (`model-behavior.md`); they are Claude-voiced. The two blur easily: a developer-voiced line like "push when you're confident" gets misread as the agent's own license to push. When adding or editing a bullet, decide whose discipline it is and put it in that file; if a craft file has accumulated an agent-conduct bullet, move it. This is why `model-behavior.md` was split out from `development-standards.md`, and why git agent-conduct (don't originate a push; on unexpected state, defer to the developer) lives in `model-behavior.md`, not `git-discipline.md`.
- **Record the craft/conduct split as a `kind:` frontmatter key on each rule.** Every managed rule carries `kind: craft` or `kind: conduct` in its frontmatter, the machine-readable form of the split above. `pk rules` surfaces it in RULES.md but never writes or enforces it. The key changes only frontmatter, so it does not affect `pk_sha256` (hashed over the body); set it in both the embedded source and the local copy when adding a rule. `developer`/`model` was rejected as the label because it reads as "who reads it" (Claude reads every rule), whereas the real axis is voice: developer-voiced craft vs Claude-voiced conduct.

## Repo Checks

- **All commands resolve to the git root via `git.RepoRoot`.** Directory resolution is uniform: `git.RepoRoot(stat, dir)` walks parent directories for `.git` (no subprocess) and returns the root path. There is no separate subprocess verification step. Commands differ only in failure policy: `changelog` and `release` exit when no repo is found, while `setup` falls back to the given directory (`--allow-non-git`).

## Security Scanning

- **Embedded managed text is scanned for hidden characters by `make test`.** `internal/setup/embed_safety_test.go` walks the embedded managed files (`rulesFS`, `skillsFS`, `templateFS`, `installScriptTemplate`) and fails on control characters, Unicode format characters (Cf: zero-width, bidi overrides), bare CR, and invalid UTF-8: the "Trojan Source" class (CVE-2021-42574). These files ship into downstream repos via `pk setup` and are read by AI agents every session, so a hidden instruction in one would fan out to every consumer; `pk_sha256` detects *that* a file changed, not that the change is safe. The repo's root `.gitattributes` pins `eol=lf`, which both silences the Windows "LF will be replaced by CRLF" warning and keeps the scan's CR rule from false-failing. The scan is a test (runs under existing CI), not a new Makefile target or shipped subcommand, so no added command surface. When adding a managed-file category with its own `//go:embed`, add it to the walk in that test.
- **Vulnerability scanning is `govulncheck` in CI, not GitHub Dependabot settings.** `make vuln` runs `govulncheck` against the live vuln.go.dev database and gates CI in `.github/workflows/ci.yml`. Do not enable GitHub's Dependabot *security updates* or *security alerts* toggles. Security-update PRs ignore `dependabot.yml`'s `target-branch` and `commit-message` and open against the default branch, bypassing the develop-first flow and `pk changelog`. Keep the dependency-security surface code-driven and in one place: `dependabot.yml` handles GitHub Actions *version* updates (landing on `develop` as `chore(deps)`), and `make vuln` handles the Go side. This also means CI builds on a maintained Go toolchain. The `go` directive pins an exact patch (e.g. `go 1.26.3`) so local auto-download and CI scan the same toolchain rather than an unpatched `.0`; when `govulncheck` flags a fix in a newer patch, bump that one line.
