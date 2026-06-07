# Namespace pk-managed rules into `.claude/rules/plankit/`

## Context

`pk setup` installs four managed rules into an adopter's `.claude/rules/` as flat files
(`git-discipline.md`, `model-behavior.md`, `development-standards.md`, `plankit-tooling.md`).
`.claude/rules/` is a shared namespace, so a repo that already has its own rule of the same
name collides with plankit's. Today the collision is silent and has two failure layers:

- **(a) pk's rule never loads.** `shouldUpdate` (`internal/setup/managed.go:156`) sees the
  adopter's same-named file has no `pk_sha256` marker and **skips** it — protecting their file,
  but plankit's rule content never enters context. One quiet `skipped (not managed by pk)` line.
- **(b) a surviving pk rule points at the wrong file.** `model-behavior.md` references
  `git-discipline.md` by bare filename; under a collision that reference resolves to the
  adopter's unrelated file. Silent semantic corruption — live today, no rename needed.

Fix, decided with the user:
- **Namespace the shipped set into a subdirectory** `.claude/rules/plankit/` (fixes layer a;
  collision becomes impossible). Claude Code discovers `.claude/rules/**` recursively — confirmed
  in official docs (`code.claude.com/docs/en/memory#organize-rules-with-claude`: "All .md files
  are discovered recursively"). Always-on token cost is unchanged (same files, same content).
- **Reference rules by title, not filename** (fixes layer b permanently, independent of layout).
  The two references are prose, not links — all rules are already in context — so Claude resolves
  them by the `# Git Discipline` heading. Title references can't collide and survive any rename.

Scope: only the four **shipped** rules move. `plankit-development.md` is maintainer-only (never
installed, no `pk_sha256`) and **stays at top-level** `.claude/rules/` — a natural split:
top-level = this-repo-only, `plankit/` = the shipped set.

## Layout decision

- Embedded source stays flat: `internal/setup/rules/*.md` (the `//go:embed rules/*.md` is
  unchanged). Only the **install destination** gains the `plankit/` segment.
- plankit's own dogfood copies move: `.claude/rules/{four}.md` → `.claude/rules/plankit/{four}.md`.
  `.claude/rules/plankit-development.md` stays put.

## The core problem to get right

The flat-layout assumption is in **four** places. Miss one and we reintroduce a silent gap:

| File | Function | Today | Needs |
|---|---|---|---|
| `internal/setup/setup.go` | install loop (`:295`) + prune (`:303`) | writes/prunes `rules/<name>.md` | write to `rules/plankit/<name>.md`; prune the subdir; **migration-sweep** old top-level copies |
| `internal/rules/rules.go` | `collectRules` (`:78`) | flat `ReadDir`, skips dirs (`:90`) | recurse so `pk rules` reports subdir rules (and any nested user rules) |
| `internal/status/status.go` | `scanManaged` rules branch (`:347`) | flat `ReadDir`, skips dirs | recurse so `pk status` sees managed rules in the subdir |
| `internal/teardown/teardown.go` | `scanManagedFiles` "rules" (`:300`) + empty-dir check (`:376`) | flat; comment "Rules are flat" | remove subdir rules; remove empty `plankit/` then `rules/` |

## Changes

### 1. Install path + migration (`internal/setup/setup.go`)
- Install loop (`:294-302`): `ruleFile := filepath.Join(settingsDir, "rules", "plankit", rule.Name+".md")`.
  `writeManaged` already does `MkdirAll(filepath.Dir(path))`, so the `plankit/` dir is created.
- Prune the subdir: `pruneRules(cfg, filepath.Join(settingsDir, "rules", "plankit"), keptRules)`.
- **Migration sweep** (reuses existing logic, no new code path): call
  `pruneRules(cfg, filepath.Join(settingsDir, "rules"), map[string]bool{})` on the **top-level**.
  With an empty kept-set, `evaluateRemoval` (`managed.go:307`) removes pristine pk-marked rules
  left at the old location, preserves user-modified ones (warns), and skips non-pk files
  (`plankit-development.md`, the adopter's own rules). Idempotent and safe to re-run.
- After pruning, remove the `plankit/` dir if it ends up empty (mirror existing empty-dir handling).

### 2. Recursive discovery + always-on/conditional split (`internal/rules/rules.go`, `document.go`)
- `collectRules`: walk `.claude/rules/` recursively (descend via the injected `cfg.ReadDir` to keep
  DI intact — call `ReadDir` on each subdir; no new dependency). For each `*.md`, set `displayPath`
  to the path **relative to `.claude/rules/`** (e.g. `.claude/rules/plankit/git-discipline.md`),
  which makes the footprint report show provenance-by-location for free. Keep the existing
  sort/parse/estimate logic. Missing dir still yields empty, not error.
- **Why recursion matters beyond the move:** today's flat scan silently *undercounts* for any
  adopter who already groups rules into subdirs (the rules load but `pk rules` misses them).
  Recursion makes the footprint match what Claude Code actually loads, for any layout.
- **Honesty fix — separate always-on from conditional.** Claude Code loads a rule with `paths:`
  frontmatter only when a matching file is read, not every session. Recursion would otherwise
  sweep such rules into the "Always-on" total and *overcount*. Add `conditional bool` to the
  `rule` struct; in `collectRules` set it via presence-check `if _, ok := fields["paths"]; ok`
  (verified: `parseFrontmatter` at `rules.go:139` registers the `paths:` key; list items have no
  colon and are skipped, so the key is present with an empty/inline value either way).
- `document.go` (`writeFootprint`/`writeFootprintLines`): partition rules into always-on and
  conditional. Keep the existing `Always-on context: N files, … tokens` header counting CLAUDE.md
  + always-on rules only, then their rows. If any conditional rules exist, emit a second group:
  `Conditional (loads on matching files): M files, … tokens` followed by those rows (same aligned
  columns + `[provenance] kind` tags). The provenance tally stays one line over all rules.
  plankit's own four rules have no `paths:`, so plankit's report is unchanged by this split.

### 3. Recursive scan (`internal/status/status.go`)
- `scanManaged` rules branch (`nested=false`): descend into subdirectories so managed rules in
  `plankit/` are found. `checkSingleFile` already gates on the `pk_sha256` marker, so recursion
  only surfaces genuinely managed files; user rules without markers are ignored as before.

### 4. Teardown (`internal/teardown/teardown.go`)
- `scanManagedFiles` "rules" case (`:300-310`): descend into subdirs so `plankit/` rules are
  scheduled for removal (marker-gated, same safety as today).
- Empty-dir logic (`:376-385`): account for removing the now-empty `plankit/` subdir, then the
  parent `rules/` dir.

### 5. Title references (`model-behavior.md`, both copies) — the one `pk_sha256` recompute
- `internal/setup/rules/model-behavior.md` (and the moved `.claude/rules/plankit/model-behavior.md`):
  - `:10` `` `git-discipline`'s "push when you're happy" `` → "the **Git Discipline** rule's …".
  - `:32` "their discipline lives in `git-discipline.md`" → "… lives in the **Git Discipline** rule".
- Body changes, so recompute `pk_sha256` for model-behavior per CLAUDE.md "Updating pk-managed
  files":  `sed -n '/^---$/,/^---$/!p' internal/setup/rules/model-behavior.md | shasum -a 256`,
  write the new value into the local copy's frontmatter. (Only model-behavior rehashes; the other
  three move with bodies untouched, so their hashes are unchanged.)

### 6. Tests
- `internal/setup/setup_test.go`: assert install lands at `.claude/rules/plankit/<name>.md`; add a
  **migration test** — seed an old pristine top-level `git-discipline.md`, run setup, assert it's
  removed and the subdir copy exists; seed a user-modified top-level copy, assert it's preserved+warned.
- `internal/status/status_test.go`: fixtures/assertions for managed rules under `plankit/`.
- `internal/teardown/teardown_test.go`: assert subdir rules are removed and `plankit/`+`rules/` dirs cleaned.
- `internal/rules/rules_test.go`: add a case proving a rule in a subdir is discovered with the
  relative `displayPath`; add a case with a `paths:`-frontmatter rule asserting it appears under
  "Conditional", is excluded from the always-on total, and that a no-`paths:` rule stays always-on;
  keep existing fixtures. Existing `TestEstimateTokens`/`TestHumanInt` etc. still cover the
  now-exported helpers (rename references if needed).
- `evals/footprint/main_test.go`: still green after the helper extraction (it exercises the report
  + README rewrite, which now call `rules.FormatBytes`/`HumanInt`).

### 7. DRY: extract shared formatting helpers (`internal/rules` ↔ `evals/footprint`)
- `evals/footprint/main.go` carries byte-identical copies of `formatBytes` (`:235`), `humanInt`
  (`:243`), and `normalizeLF` (`:263`) that already exist in `internal/rules`. Export them from
  `internal/rules` (`FormatBytes`, `HumanInt`, `NormalizeLF`), update `internal/rules`' own callers
  (`document.go` formatBytes/humanInt; `rules.go`/`document.go` normalizeLF), and delete the copies
  in `evals/footprint`, which already imports the package — no import cycle (evals → internal/rules).
- Scope limit: do **not** touch the separate `normalizeLF` in `internal/setup/managed.go` or any
  copies in `internal/status` — folding those would make `setup`/`status` depend on `internal/rules`,
  a layering change beyond this fix. Leave the two **report renderers** separate too (footprint's
  always-on/on-demand-skills vs `pk rules`' provenance+kind / always-on+conditional are different
  shapes; merging is the wrong abstraction).
- `evals/footprint` does **not** recurse and does **not** need the conditional split: it reads the
  flat embedded source (`internal/setup/rules/*.md`), which this change leaves flat, and all four
  shipped rules are always-on.

### 8. Docs + prose references (no `pk_sha256` impact)
- `docs/pk-setup.md`: "Installs rules to `.claude/rules/`" → `.claude/rules/plankit/`; update the
  enumerated names/paths.
- `docs/pk-rules.md`: update sample footprint output to show `plankit/` paths, recursive
  discovery, and the always-on vs "Conditional (loads on matching files)" grouping.
- `CLAUDE.md` (project) `:60`: update the rules-dir description to note the `plankit/` subdir.
- `.claude/rules/plankit-development.md` (maintainer, no hash): the "Updating pk-managed files"
  and Rule Authoring sections name embedded/local rule paths — update to the `plankit/` local path.
- `.claude/skills/release-notes/SKILL.md` (local, no hash): prose mentions of rule files — update.
- `evals/cases.md`, `evals/README.md`, `evals/run-evals.sh`: update rule path/name mentions.

## Out of scope (verified)
- `pk-` filename prefix — superseded by the subdir decision.
- Loud setup-time collision warning — subdir makes collisions impossible, so unneeded.
- Embedded-source relayout — embed stays flat; only install destination changes.
- `evals/footprint` recursion/conditional logic — not needed (flat source, all always-on). It is
  touched only by the §7 helper extraction, not by the layout change.
- DRY-ing `normalizeLF` in `internal/setup`/`internal/status` — out (would create a layering
  dependency on `internal/rules`); only the `evals/footprint` ↔ `internal/rules` dup is folded.
- Token-cost / README footprint number — content unchanged, so the figure is unchanged.

## Verification

Automated:
- `make test` — new migration/subdir/recursion tests pass; existing suites green.
- `make lint` — gofmt + vet clean.

Smoke (deterministic):
- `make build`. Greenfield: `dist/pk setup` in a temp git repo → rules land in
  `.claude/rules/plankit/`; `dist/pk rules` reports all four with `plankit/` paths and the same
  always-on total as before; `dist/pk status` shows them managed.
- **Conditional split:** add a user rule `.claude/rules/scoped.md` with `paths:` frontmatter →
  `dist/pk rules` lists it under "Conditional (loads on matching files)", excluded from the
  always-on total; a user rule without `paths:` appears in the always-on group.
- **Collision case (the bug):** in a temp repo, create a user `.claude/rules/git-discipline.md`
  (no marker) → `dist/pk setup` installs `plankit/git-discipline.md` **and leaves the user file
  untouched**; both coexist; `dist/pk rules` lists both, distinct paths.
- **Migration case:** seed an old pristine top-level `git-discipline.md` (with a valid pk marker),
  run `dist/pk setup` → old file removed, `plankit/` copy created; re-run → byte-identical (idempotent).
- **Modified-old case (negative):** seed a top-level pk rule whose body was edited → setup
  preserves it with a warning, still installs the `plankit/` copy.
- `dist/pk teardown --confirm` in an installed repo → `plankit/` rules removed, empty dirs cleaned.
- Title references: `grep -rn "git-discipline" internal/setup/rules/model-behavior.md` returns
  nothing; the rule is referenced as **Git Discipline** instead.
