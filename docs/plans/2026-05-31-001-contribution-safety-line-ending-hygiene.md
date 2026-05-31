# Contribution safety + line-ending hygiene

## Context

Two threads, folded into one change.

**Thread A — the jqwik "booby trap" story.** A maintainer hid instructions in shipped
text (ANSI escape codes) so AI agents reading his project would silently delete code.
plankit's classic supply-chain exposure is already well defended (stdlib-only, no
`go.sum`; Actions pinned to SHAs; `govulncheck` gating CI; race tests + lint + no-squash
+ protected `main`). The risk that actually maps is the **mirror image**: plankit
*ships* text to AI agents — `pk setup` fans the embedded managed files into downstream
repos where Claude reads them every session. A contributor who slipped hidden
instructions, zero-width characters, or bidi overrides (the "Trojan Source" class,
CVE-2021-42574) into one of those files would plant a jqwik-style trap that plankit then
distributes. The `pk_sha256` marker detects *that* a file changed, not that the change
is malicious, and a fresh install has no baseline. **Nothing currently scans the
embedded markdown for hidden characters.**

**Thread B — the CRLF mess Windows users actually hit.** Real symptom: on Windows with
`core.autocrlf=true` and **no `.gitattributes`**, git prints "LF will be replaced by
CRLF the next time Git touches it" whenever anything stages LF-stored files (e.g. a
`postVersion` hook, or pk's own `git add -u` at `internal/changelog/changelog.go:308`).
Harmless, but noisy and alarming (18 files in one report). The durable fix is a
`.gitattributes` pinning line endings so the attribute governs instead of the autocrlf
heuristic.

The two threads collide: the hidden-char scan in Thread A would flag `\r`, so a Windows
contributor's CRLF would false-fail it. Pinning LF via `.gitattributes` removes that
footgun and is itself the worked example for the Thread B guidance.

Decisions taken with the user:
- **Both threads, folded together.**
- **No mandatory preserved plan for PRs** — describing intent/approach in the PR body for
  substantive changes stays advisory.
- **pk does not impose line endings on user projects.** The downstream answer is
  *guidance* in `docs/resources.md` (using plankit's own `.gitattributes` as the
  example), not a `pk setup`-written file.
- **Keep** the scan + CONTRIBUTING section + PR template; make the scan CRLF-safe.

## Embedded managed files the scan covers

All in package `setup`, reachable from a test in `internal/setup/`:
- `internal/setup/template/CLAUDE.md` — `templateFS` (`embed.FS`)
- `internal/setup/rules/*.md` (4 files) — `rulesFS` (`embed.FS`)
- `internal/setup/skills/*/SKILL.md` (3 files) — `skillsFS` (`embed.FS`)
- `internal/setup/template/install-pk.sh` — `installScriptTemplate` (`string`)

## Change 1 — `.gitattributes` (repo root, new)

Pin line endings so CRLF never enters the repo and the scan's CR rule is consistent:

```gitattributes
# Normalize all text to LF in the repo and on checkout, every platform.
# Keeps managed files (shipped to downstream repos and read by AI agents) byte-stable,
# and silences "LF will be replaced by CRLF" for contributors on Windows.
* text=auto eol=lf
```

`text=auto` lets git skip detected-binary files; `eol=lf` forces LF on checkout
everywhere, so `//go:embed` captures LF bytes regardless of the build platform. This file
doubles as the worked example referenced from `docs/resources.md`.

## Change 2 — Trojan-source / hidden-character scan (CRLF-safe)

Add `internal/setup/embed_safety_test.go` (package `setup`) that walks each embedded value
above and fails on any disallowed rune, reporting `path:line:col` + `U+XXXX`. A Go **test**
rather than a Makefile target or shipped subcommand: it runs under `make test` (already
gates CI on every PR), scans exactly what ships (the embedded bytes), auto-covers new
managed files matching the existing embed globs, and adds no user-facing surface.

`disallowedRune` policy (belt-and-suspenders with the `.gitattributes` LF pinning):
- **Allow** `\n` (0x0A) and `\t` (0x09).
- **Allow** `\r` (0x0D) **only** when immediately followed by `\n` (a normal CRLF line
  ending) — a bare/lone CR is disallowed. This is what makes the scan CRLF-safe.
- **Disallow** any other `unicode.IsControl(r)` — catches ANSI ESC (0x1B) and the rest of
  C0/C1.
- **Disallow** `unicode.In(r, unicode.Cf)` — zero-width (U+200B–200D, U+FEFF), directional
  marks/overrides/isolates (U+202A–202E, U+2066–2069), word joiners.
- **Disallow** invalid UTF-8 (`utf8.RuneError` with width 1).

Mechanics: `fs.WalkDir` over each `embed.FS`; scan `installScriptTemplate` directly. The CR
look-ahead needs byte-level iteration (track whether the next byte is `\n`), not a plain
`for range` over runes. First run must pass against current files (already verified
CR-free). Add a table-driven `t.Run` over crafted byte strings (ANSI ESC, U+200B, lone CR,
valid CRLF) to lock the policy independently of the embedded fixtures.

Add a bullet to the maintainer-only rule `.claude/rules/plankit-development.md` under its
existing **Security Scanning** section, mirroring the govulncheck bullet: embedded managed
text is scanned for hidden/control characters by `make test` because those files ship into
downstream repos and are read by AI agents. (This rule is *not* embedded in `pk setup`, so
no `pk_sha256` recompute.)

## Change 3 — `docs/resources.md` line-endings guidance

Add a `### Line endings (LF vs CRLF)` subsection under the existing **Git** section
(after "Release workflows"). Cover, as downstream-facing guidance:
- The "LF will be replaced by CRLF the next time Git touches it" warning — what triggers it
  (`core.autocrlf=true` on Windows + no `.gitattributes`), and that it's informational
  noise, not an error or data loss.
- The fix: a `.gitattributes` with `* text=auto eol=lf` (or per-type), which makes the
  policy explicit so git stops guessing; alternatively `git config core.autocrlf input`.
- Point at plankit's own root `.gitattributes` as the worked example.

`docs/resources.md` is already linked from the README Reference area, so no README change.

## Change 4 — CONTRIBUTING.md "Contributions & AI" section

Append a short, factual, advisory section:
- plankit is a solo/small-team toolkit; PRs welcome, reviewed by the maintainer.
- No third-party Go dependencies — PRs adding one won't be accepted.
- Changes to **managed files** (`internal/setup/rules`, `skills`, `template/CLAUDE.md`) get
  extra scrutiny: they ship into downstream repos and are read by AI agents; they are
  LF-normalized via `.gitattributes` and scanned for hidden characters by `make test`. Note
  the paired `pk_sha256` update (already in CLAUDE.md's "Updating pk-managed files").
- Advisory: for substantive/behavior-changing PRs, describe intent and approach in the PR
  body, not just a diff. No preserved plan required.

## Change 5 — PR template

Add `.github/PULL_REQUEST_TEMPLATE.md`, advisory (no blocking gates):
- **What & why** (intent/approach for substantive changes).
- **Conventional commit type** used.
- Checklist: `make test`, `make lint`, `make vuln` pass; if managed files were touched,
  confirm `pk_sha256` updated and the embed-safety check passes.

## Verification

- `make test` — `embed_safety_test.go` passes against current managed files; the crafted-
  input `t.Run` cases assert ANSI ESC, U+200B, and lone CR fail while LF/CRLF/tab pass.
- Negative case (manual): inject `\x1b[2J` into a copy of a managed `.md`, run `make test`,
  confirm failure with the right `path:line:col` + `U+001B`; revert.
- `.gitattributes`: `git check-attr eol -- internal/setup/rules/git-discipline.md` reports
  `eol: lf`; `git add --renormalize .` produces no unexpected diff (repo already LF).
- `make build && make lint` clean.
- Render `docs/resources.md`, `CONTRIBUTING.md`, and `.github/PULL_REQUEST_TEMPLATE.md`;
  confirm the PR template appears when opening a PR.

## Out of scope (per decisions)

- No mandatory preserved plan for PRs; no PR-blocking status check beyond existing CI.
- No `pk setup`-written `.gitattributes` in user projects — guidance only.
- No CODEOWNERS / CODE_OF_CONDUCT / governance docs; no new shipped `pk` subcommand.
