# Plan: Condense methodology + simplify skills

## Context

The current `docs/methodology.md` (162 lines) reads as an essay — good for making the case, but hard to scan during sessions. A separate brainstorming session explored structural alternatives and identified a 14-section structure that works well. The agreed direction: use that structure but rewrite in the project's own voice, keeping the 3-4 narrative examples that earn the guidelines and cutting the rest to bullet+detail.

Separately, `docs/getting-started.md` duplicates README.md and pk-setup.md — no unique content survives removal. And the `/changelog` and `/release` skills are redundant now that `/ship` chains both; removing them simplifies the skill surface to `/init`, `/preserve`, `/ship`.

Two commits on `develop`, in this order.

---

## Commit 1: `docs: condense methodology and remove getting-started`

### Rewrite `docs/methodology.md`

14-section structure with clear hierarchy. Most sections are tight bullet+detail. Sections that keep condensed narrative (1-3 sentences, not full anecdotes):

- **Plan review** — silent semantic narrowing (unique practical insight)
- **Guidelines** — heft/gulp example + autonomous push example (why guidelines must be present, not just remembered)
- **Discipline** — Knuth/Aquino-Michaels reference (one sentence)
- **Use what you build** and **When the model shifts** — fold into section 13/14 as brief callouts

Target: ~140-160 lines. References to `/changelog` and `/release` skills in the current doc (line 151) are dropped since those skills are being removed in commit 2. Refer to `/ship` as the release workflow instead.

### Delete files

- `docs/getting-started.md`

### Update references to `getting-started.md`

| File | Line | Change |
|------|------|--------|
| `README.md` | 33 | Remove "or [Getting Started](docs/getting-started.md)" — keep "See [Setup](#setup) below for details." |
| `README.md` | 72 | Remove Getting Started entry from Documentation list |
| `CLAUDE.md` | 98 | "(README, getting-started, methodology)" → "(README, methodology)" |
| `.claude/rules/plankit-development.md` | 15 | Remove `docs/getting-started.md` from grep example list |

Note: `docs/plans/` files reference getting-started.md but plans are immutable — no changes.

### Verification

- All internal doc links resolve (no broken `getting-started.md` links)
- `make test` passes
- `make build` passes

---

## Commit 2: `chore: simplify skills to /init, /preserve, and /ship`

### Delete embedded skill sources

- `internal/setup/skills/changelog/SKILL.md` (delete directory)
- `internal/setup/skills/release/SKILL.md` (delete directory)

The `//go:embed skills/*/SKILL.md` glob at `setup.go:23` auto-excludes them. The `skills()` function at line 268 dynamically reads the embed.FS — no code change needed.

### Delete local skill copies

- `.claude/skills/changelog/SKILL.md` (delete directory)
- `.claude/skills/release/SKILL.md` (delete directory)

### Update `/ship` skill (both copies)

`internal/setup/skills/ship/SKILL.md` and `.claude/skills/ship/SKILL.md`:
- Change "Combined changelog + release workflow" → "The release workflow"
- Recompute `pk_sha256` for local copy per CLAUDE.md convention

### Update `plankit-tooling.md` rule (both copies)

`internal/setup/rules/plankit-tooling.md` line 11 and `.claude/rules/plankit-tooling.md` line 12:
- `` `/changelog`, `/release`, `/preserve` `` → `` `/ship`, `/preserve` ``
- Recompute `pk_sha256` for local copy

### Update documentation

| File | Line(s) | Change |
|------|---------|--------|
| `README.md` | 17 | `/init`, `/changelog`, `/release`, `/ship`, `/preserve` → `/init`, `/ship`, `/preserve` |
| `README.md` | 21 | Rewrite: `/ship` is the release workflow, chains `pk changelog` + `pk release` with preview+confirm |
| `CLAUDE.md` | 56 | `managed skills (changelog, init, preserve, release, ship)` → `managed skills (init, preserve, ship)` |
| `CLAUDE.md` | 65 | Update skill-managed commands line — only `/ship` wraps `pk changelog` and `pk release` |
| `docs/pk-setup.md` | 26 | Skill list: `/init`, `/preserve`, `/ship` |
| `docs/pk-setup.md` | 72 | "five managed skills" → "three managed skills", update list |
| `docs/creating-skills.md` | 36 | Installed skills list: `/preserve`, `/ship` |
| `docs/creating-skills.md` | 69 | "five skills" → "three skills", update directory list |
| `docs/creating-skills.md` | 73 | "/changelog and /release skills use this pattern" → "/ship skill uses this pattern" |
| `docs/creating-skills.md` | 336 | References list: `changelog`, `init`, `preserve`, `release`, `ship` → `init`, `preserve`, `ship` |
| `.claude/skills/release-notes/SKILL.md` | 107-108 | "run `/changelog` and `/release` to ship" → "run `/ship` to ship" |

### Update code

| File | Line | Change |
|------|------|--------|
| `internal/setup/setup_test.go` | 70 | `{"changelog", "preserve", "release"}` → `{"init", "preserve", "ship"}` |

### Files NOT changed

- `docs/plans/` — immutable historical records, references to /changelog and /release stay
- `internal/changelog/` and `internal/release/` packages — CLI commands remain, only skills removed
- `pk teardown` — dynamically scans for `pk_sha256` marker, no hardcoded skill names
- `docs/pk-changelog.md` and `docs/pk-release.md` — command docs for CLI commands, still valid
- `.pk.json` config keys — `changelog.*` and `release.*` configure the CLI commands, not skills

### Verification

- `make test` — confirms setup_test.go change works
- `make build` — confirms embedded FS compiles with removed skills
- `make lint` — no formatting drift
- Smoke: `./dist/pk setup --project-dir /tmp/test-project` installs exactly 3 skills (init, preserve, ship) and no changelog or release skill directories
