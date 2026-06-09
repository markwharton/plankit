# Plan: unify mode inference (push-guard in `pk status`) + add a migration prompt to docs/adoption.md

## Context

Two gaps, surfaced by the question "does `pk status` list the new guard push?" and by friction adopting plankit into **established** repos.

1. **`pk status` hides push-guard, and the inference layer is split in two.** status prints the `guard:` branch mode (block/ask/off) but ignores the `--push-guard` argument carried in the *same* hook command, so a project with push-guard on looks identical to one without it. The root cause is structural: mode inference lives in two parallel function families — `InferModes*` returns `(guard, preserve)`, while `InferPushGuard*` returns push-guard separately. Even setup's own mode-persistence path (`cmd/pk/main.go:206–219`) calls them as two separate steps with two `if` blocks. Push-guard *is* a mode; the durable fix is one inference call that returns all three.

2. **Established-repo migration has no interactive home.** New projects already work out of the box (`/new-plankit-project`). Established repos hit open-ended, judgment-heavy work — a foreign `CHANGELOG.md`, which ecosystem hooks a `package.json` needs, where to anchor the baseline, removing the old release tool. `adoption.md` Layer 4 documents this as prose, but nothing walks the user through it.

**Decision (with the user):** A deterministic `pk doctor`/`--fix` was considered and **rejected** — it would duplicate plankit's existing fail-fast (`pk changelog`/`guard`/`release` already error loudly at the moment a missing tag/branch matters) and the `/conventions` skill's interactive diagnosis, and a check list is always a step behind the genuinely hard migration cases. Migration is LLM-shaped, not check-list-shaped. A shipped `/adopt` **skill** was also considered and rejected: shipped skills install into *every* downstream repo via `pk setup`, justified only for **recurring** work (`conventions`/`preserve`/`ship`); adoption is a **one-time** task, so a shipped skill is permanent dead footprint everywhere to serve a task run once. The lightest mechanism that fits is a **copy-pasteable migration prompt in `docs/adoption.md`** — structurally the same instructions a skill carries, with zero install cost. So: Part 1 fixes the inference layer (Go); Part 2 is a documentation prompt. `/conventions` stays scoped to convention discovery. A "verify configured branches exist" check was also rejected — "exists where (local/origin)?" is ambiguous, a missing branch is often expected (about to be created), and `pk release` already fails clearly when it matters; configure branch *intent*, don't audit repo *state*.

---

## Part 1 — Unify mode inference, surface push-guard everywhere

A small refactor of the inference layer, not a print-site patch. Push-guard becomes a first-class field, so every consumer (setup persistence, status, the future adopt flow) gets all three modes from one call.

### Unify the family (`internal/setup/claude.go`)

```go
type Modes struct {
    Guard     string // "block" | "ask" | "off" | ""
    Preserve  string // "auto" | "manual" | "off" | ""
    PushGuard string // "block" | "ask" | "" (meaningful only when Guard is block/ask)
}

func InferModesFromCommands(commands []string) Modes   // was (guard, preserve string)
func InferModes(settings *OrderedObject) Modes
func InferModesFromSettings(readFile, dir) Modes
```

- Fold the `--push-guard <mode>` parse (currently `InferPushGuardFromCommands`, claude.go:240) into the command loop in `InferModesFromCommands`, which already walks every command.
- **Remove** the redundant `InferPushGuardFromCommands` / `InferPushGuardFromSettings`; their tests fold into the `Modes` tests. (Both internal — no external API concern.)

### Update the three callers

- **`cmd/pk/main.go:206–219`** (setup mode-persistence): collapse the two `if` blocks into one — `m := setup.InferModesFromSettings(os.ReadFile, dir)`, apply `m.Guard` / `m.Preserve` / `m.PushGuard` under their `!explicit` guards. Deletes the parallel push-guard call that prompted this refactor.
- **`internal/status/status.go`**: `inferModes(hooks)` (line 319) returns `setup.Modes`. In the **Modes** block (131–142) print `guard:` / `preserve:` as today, add a `push:` line when `Guard` is `block`/`ask` (show `PushGuard`, or `off` when guard is active but no `--push-guard` flag — mirroring how `guard`/`preserve` render `off`); omit when guard is off/unset. In **brief mode** (213–224) append `push=<mode>` to `parts` when set.
- **`internal/setup/claude_test.go`**: update the `InferModes*` tests to the struct return; migrate the `InferPushGuard*` sub-tests to assert `Modes.PushGuard`.

**Files:** `internal/setup/claude.go`, `internal/setup/claude_test.go`, `cmd/pk/main.go`, `internal/status/status.go`, `internal/status/status_test.go` (fixture carries `--push-guard block`; assert the `push:` line). `docs/pk-status.md` "Mode inference" section.

---

## Part 2 — Migration prompt in `docs/adoption.md`

No code, no skill, no install footprint. Add a focused, copy-pasteable prompt to the existing Layer 4 (Migration) section of `docs/adoption.md`. A user migrating an established repo gives this prompt to Claude once; it distills the Layer 4 prose into a single ready-to-run instruction. The surrounding Layer 4 prose stays as the reference the prompt draws from.

Adoption targets **experienced developers bringing an established project onto plankit** — not new builders. The prompt can assume familiarity with the project's build, and should ask the developer to make the judgment calls rather than guess — above all the version-file / `preCommit` wiring in step 3, which is the genuinely hard part.

### The prompt (a fenced block under Layer 4)

Phrased as instructions to Claude, concise, covering the four migration tasks already documented in Layer 4:

1. **Baseline placement** — check `git tag --list 'v*' --sort=-v:refname`; if no semver tag, explain `--at <ref>` (fold prior history) vs. fresh start and recommend `pk setup --baseline --push`. Advisory — let the developer run the remote/tagging command.
2. **Foreign `CHANGELOG.md`** — default to **leave-as-is and continue**: `pk changelog` appends new entries in its format above the existing content, which stays untouched (new releases in plankit format, old entries unchanged). This is the simplest, lossless path — recommend it. Offer a full rewrite to plankit's format *only* if the user wants uniformity, and warn it is **lossy**: constructs plankit omits by design (e.g. per-SHA commit links) would be dropped. Show the change and confirm before writing.
3. **Version propagation + commit-time hooks — the complex core, for a developer who knows the build.** This is where established-project migration actually lives. The taxonomy below is drawn from a survey of ~21 real `.pk.json` files (described generically — no project names or proprietary script names). Walk the developer through it; don't guess or auto-emit a hook.

   **First, does the project even need this?** About a third of real configs use nothing here — tags are the sole version source, just `guard` + `release`. If no file carries a version string that must track the tag, stop; adding hooks would be over-engineering.

   Otherwise map each version-bearing file to the right mechanism:
   - **JSON manifests/lockfiles** (`package.json`, nested/workspace `package.json`, `package-lock.json`) → `changelog.versionFiles` (declarative; **JSON-only**).
   - **Non-JSON version strings** (`pyproject.toml`, Python `__version__`, a Go `const version`, etc.) → `pk pin --file <path> --name <ident> $VERSION` chained in `changelog.hooks.preCommit`, because `versionFiles` can't touch them. (This is how Python/Go projects bump — a whole category `versionFiles` doesn't cover.)
   - **Files derived from the bump** (lockfiles, generated docs/specs, monorepo cross-refs) → regenerate in `preCommit` and `git add` the ones pk won't already stage. Real shapes: `npm install --package-lock-only` (lockfile resync, sometimes `cd <subdir> &&`); `pnpm -r exec npm version $VERSION --no-git-tag-version` then stage the workspace manifests + lockfile; `npm run generate:*` then `git add` the output; a framework-specific version-sync script.
   - **`postVersion`** is the alternative `$VERSION`-aware hook (runs right after the bump, before the changelog is written) for deriving a value into another file; `preCommit` (runs just before the commit) is the more common choice.
   - **`release.hooks.preRelease`** — a validation gate before publishing: clean install + lint + test + build (e.g. `npm ci && npm run lint && npm test && npm run build`, sometimes `+ go test`).

   Both hooks receive `$VERSION` as an env var. Merge into `.pk.json`, never overwrite; sort top-level keys. Rare opt-ins to mention only if relevant: custom `changelog.types` (non-default type→section mapping) and `showScope: true`. **Before writing the prompt's `git add` guidance, confirm what `pk changelog` stages automatically vs. what the hook must stage** (the real configs differ — some `git add` explicitly, some don't), so the instruction is precise.
4. **Remove the old tool** — detect `commit-and-tag-version` / `standard-version` / `semantic-release` in devDeps/CI and advise disabling them to avoid conflicting tag/changelog writes.

Keep it tight (a short numbered prompt, not a wall of text) and self-contained so it works pasted on its own. It does **not** audit branch existence (configure intent, not state) and stays advisory on anything remote/destructive.

### Light pointer

- `README.md` — if the Migration/adoption row or link doesn't already surface this, add a one-line pointer ("Migrating from another release tool? See [Adoption — Migration](docs/adoption.md#layer-4-migration)"). No skill-list or skills-table change (this is not a skill).

---

## Files

**Create:** none.

**Modify:**
- Part 1: `internal/setup/claude.go` + `claude_test.go` (`Modes` struct, remove `InferPushGuard*`); `cmd/pk/main.go` (collapse persistence calls); `internal/status/status.go` + `status_test.go` (consume `Modes`, print `push:`); `docs/pk-status.md`.
- Part 2: `docs/adoption.md` (migration prompt under Layer 4; also extend the Layer 4 "NPM projects" prose, which today only shows replacing release scripts, to mention the `versionFiles` + `preCommit` regen/`git add` + `preRelease` pattern so prose and prompt agree); `README.md` (one-line pointer, only if not already present).

No code is added for Part 2 — no new skill, command, flag, or `.pk.json` schema change. `internal/setup/skills/`, `setup_test.go`, `pk-setup.md`'s skill count, `CLAUDE.md`'s skill enumeration, the `plankit-tooling` `--push` list, and the reference docs are all unchanged.

---

## Verification

**Automated (Part 1):** `make test` (updated `claude_test.go`/`status_test.go`), `make lint`, `make vuln`. Part 2 is docs-only — no automated tests; `make test` still must pass unchanged.

**Smoke — Part 1 (`make build` → `dist/pk`):**
1. In a project configured with `--push-guard block`: `dist/pk status` shows a `push:` line under `guard:`; `dist/pk status --brief` includes `push=block`.
2. Negative: a project with guard off/unset shows no `push:` line.
3. Re-run `dist/pk setup` (no `--push-guard` flag) on a push-guarded project → the collapsed persistence path preserves `--push-guard` in the hook command (mode not silently reset).

**Smoke — Part 2 (docs):**
4. Render/read `docs/adoption.md` Layer 4 — the prompt is self-contained and copy-pasteable; the `README.md` pointer resolves to the right anchor.
5. End-to-end sanity across the taxonomy: paste the prompt to Claude against (a) a tags-only repo — confirm it adds *no* version hooks; (b) a Node repo with nested `package.json` + lockfile — confirm `versionFiles` + `preCommit` lockfile resync; (c) a Python/Go repo — confirm it reaches for `pk pin` in `preCommit` (not `versionFiles`). In each: recommends leaving the CHANGELOG as-is (rewrite only on request, with the lossy warning), treats hook wiring as developer judgment calls, merges `.pk.json` without clobbering, and stays advisory on baseline/remote actions.
