# Plan: move guard/preserve/push modes into `.pk.json`; hook commands go bare

## Context

plankit's guard/preserve config is **split and half-implicit**: the *branches* live in `.pk.json` (`guard.branches`), but the *modes* (guard block/ask/off, push-guard block/ask/off, preserve auto/manual/off) are encoded in the **hook command strings** in `.claude/settings.json` — by presence/bareness/flags (`pk guard` = block, `pk guard --ask` = ask, no `--push-guard` = push off, no preserve hook = preserve off). You can't read a mode off the command without knowing the conventions, and `pk status` has to *infer* modes back out of command strings (the `InferModesFromCommands` layer).

**Goal:** one source of truth. All guard/preserve config moves into `.pk.json`; every hook command becomes bare (`pk guard`, `pk preserve`, `pk protect`). `settings.json` becomes static wiring, identical for every project; `.pk.json` is the single, human-editable, legible config surface.

**Decisions already made with the user:**
- Modes live in `.pk.json` (not crammed into command flags) — chosen for editability (`"branches": ["main","develop"]` and `"mode": "ask"` beat flag strings buried in `settings.json`).
- `off` is a real value (`"mode": "off"`), not an absence.
- **Absent key → default**, and `pk setup` **writes every key explicitly**. Defaults: `guard.mode` → `block`, `guard.push` → `block`, `preserve.mode` → `manual`.
- Careful migration (early dev, but the blast radius is the user's own ~20 projects; never change a mode silently).

## End state

`.claude/settings.json` — static wiring, no flags anywhere:
```json
"PreToolUse":  [ {"matcher":"Bash|PowerShell","hooks":[{"command":"pk guard"}]},
                 {"matcher":"Edit|Write","hooks":[{"command":"pk protect"}]} ],
"PostToolUse": [ {"matcher":"ExitPlanMode","hooks":[{"command":"pk preserve"}]} ],
"SessionStart":[ {"matcher":"*","hooks":[{"command":".claude/install-pk.sh"}]} ]
```
`.pk.json` — the config (keys alphabetical, per the conventions skill):
```json
{
  "guard":    { "branches": ["main"], "mode": "block", "push": "block" },
  "preserve": { "mode": "manual" },
  "release":  { "branch": "main" }
}
```
Rule everywhere: **present = that value; absent = the default; `"off"` = deliberately disabled.**

---

## A. Config schema + defaults (`internal/config/config.go`)

- `GuardConfig`: add `Mode string` (`json:"mode,omitempty"`) and `Push string` (`json:"push,omitempty"`) alongside `Branches`.
- New `PreserveConfig{ Mode string }`; add `Preserve PreserveConfig` to `PkConfig`.
- **Single source of truth for the default values** (kills the "two defaults" duplication): exported consts `DefaultGuardMode="block"`, `DefaultGuardPush="block"`, `DefaultPreserveMode="manual"`, plus resolver methods that apply them: `GuardConfig.ResolvedMode()`, `GuardConfig.ResolvedPush()`, `PreserveConfig.ResolvedMode()`. Both the runtime (fallback) and `setup` (what it writes) reference these — the default literal exists once.

## B. Runtime reads modes from `.pk.json`

- **`internal/guard/guard.go`** — `loadGuardConfig` already returns `pk.Guard`. Replace `cfg.Ask` with `guardCfg.ResolvedMode()` (block/ask/**off** → off skips the branch deny/ask entirely, lines 84–103) and `cfg.PushGuard` with `guardCfg.ResolvedPush()` (lines 105–114). Net new capability: `mode:"off"` + `push:"block"` is now expressible (guard branches off, pushes still guarded) — wasn't before.
- **`internal/preserve/preserve.go`** — add a `loadPreserveConfig` (mirror `loadGuardConfig`) reading `pk.Preserve`. Today mode is `cfg.Notify` (manual) vs not (auto); off = no hook. New: read `ResolvedMode()` → `manual` writes the notify pointer + prompt (current `cfg.Notify==true` path, ~lines 139–150), `auto` commits, **`off` returns 0 immediately** (no-op).
- **Deprecated flag overrides (confirmed with user — back-compat for the transition):** `pk guard`/`pk preserve` still *accept* `--ask`/`--push-guard`/`--notify` and let an explicitly-passed flag override `.pk.json` (detect via `fs.Visit`, the pattern `runSetup` already uses). This keeps existing `pk guard --ask --push-guard block` hooks working the instant the binary upgrades, so there's **no window where guard is silently off** before re-setup normalizes them. Flags marked deprecated; removed in a later release once all projects are migrated.

## C. `pk setup` writes `.pk.json` + static hooks + migrates

New behavior: setup writes modes into `.pk.json` (today it never touches `.pk.json`). Reuse the existing ordered-merge machinery — `OrderedObject`, `ParseOrderedObject`, and the `addPermission`/`mergeHooks` template (`internal/setup/setup.go`, `claude.go`). Steps:

1. **Resolve each mode** by precedence: explicit setup flag (`--guard`/`--preserve`/`--push-guard`, via `fs.Visit`) > existing `.pk.json` value > **migrated from the old hook command** (`InferModesFromCommands` on the current `settings.json`) > default const. This preserves existing config and migrates old installs faithfully; defaults only apply to a genuinely fresh project.
2. **Write `.pk.json`** (new helper, mirrors `addPermission`): parse existing into an `OrderedObject`, **field-level merge** the `guard` object (`set mode`+`push`, keep `branches`), set `preserve.mode`, leave `release`/`changelog` untouched, re-sort top-level keys alphabetically, `MarshalIndent`, write. No SHA marker (user-owned). Replace today's "No .pk.json found — run /conventions" reminder (`setup.go:365`) with this write.
3. **Hooks go static/bare:** `buildHookConfigWithPush` → a `buildHookConfig` that always emits `pk guard`, `pk protect`, `pk preserve` (no `--ask`/`--push-guard`/`--notify`), regardless of mode. Guard/preserve hooks are **always installed** now (off lives in `.pk.json`; the bare command no-ops at runtime when off).

**Migration fix (required for faithful preservation):** `InferModesFromCommands` must decode an **absent** `--push-guard` (when a guard hook is present) as `"off"`, not `""` — so an existing bare `pk guard` migrates to `guard.push:"off"` and is *preserved*, never silently flipped to the new `block` default. (This is the latent Part-1 inconsistency; fixing it here is what makes migration safe. Default `block` then only reaches truly fresh installs, which have nothing to migrate.)

## D. `pk status` reads `.pk.json` (`internal/status/status.go`)

Drop the mode-inference-from-hooks (`inferModes`) and read `guard.ResolvedMode()`/`ResolvedPush()` + `preserve.ResolvedMode()` straight from `config.Load`. Print effective modes (defaults resolved). `InferModesFromCommands` stays in `internal/setup` purely as the migration reader — `status` no longer needs it.

## E. `cmd/pk/main.go` flags

- **`runSetup`** keeps `--guard`/`--preserve`/`--push-guard`; they now route to `.pk.json` writes instead of command-string encoding. Same UX (`pk setup --guard ask` still works).
- **`runGuard`/`runPreserve`** keep `--ask`/`--push-guard`/`--notify` as the deprecated overrides from §B.
- The Part-1 `InferModesFromSettings` persistence block becomes the migration read in §C.

## F. Skills, docs, the preserve-hook wrinkle

- **conventions skill** (shipped: embedded source + local copy + recomputed `pk_sha256`): now that `setup` writes `guard.mode`/`guard.push`/`preserve.mode`, the skill must **field-level merge** the `guard` object — preserve `mode`/`push` when it writes `branches`. Tighten step 5's "merge, don't overwrite" to be explicit about not clobbering mode keys.
- **new-plankit-project skill** (maintainer-only): its starter `.pk.json` can stay minimal — the `pk setup` it runs now writes the mode keys. Optionally show them for clarity.
- **Docs:** `pk-json.md` (new `guard.mode`/`guard.push`, new `preserve` section), `pk-guard.md` + `pk-preserve.md` (modes now in `.pk.json`; `--ask`/`--push-guard`/`--notify` deprecated), `pk-setup.md` (flags write to `.pk.json`; default `push` = block), `pk-status.md` (reads `.pk.json`), `adoption.md` (Layers 1–3 + the migration prompt/checklist gain `guard.mode`/`guard.push`/`preserve.mode`), `error-reference.md` if mode-validation messages move.
- **Open implementation decision (preserve hook entry):** today the preserve *entry* encodes mode via `async`/`timeout`/`statusMessage` (auto: async=true/60; manual: async=false/10) — a static single entry must pick one. Proposed: `pk preserve`, `async=false`, `timeout=30`, generic status text (committing one plan file is fast, so sync is fine and keeps the notify message surfacing reliably). Validate it doesn't block noticeably; this is the one behavior nuance to confirm during build.

---

## Migration & back-compat summary

- Existing project, next `pk setup`: modes lifted from old hook commands into `.pk.json` (faithfully, incl. absent-push→off), hooks rewritten bare. Visible in the `.pk.json`/`settings.json` diff; nothing silent.
- Binary upgrade *before* re-setup: old flag-bearing hooks keep working via the deprecated flag overrides (§B). No protection gap during the window.
- Fresh project: defaults written explicitly (`block`/`block`/`manual`); guard still dormant until `branches` added.

## Files

- **Code:** `internal/config/config.go` (fields, consts, resolvers); `internal/guard/guard.go` + `internal/preserve/preserve.go` (read config, off no-op, deprecated overrides); `internal/setup/setup.go` + `claude.go` (write `.pk.json`, static bare hooks, migration, absent-push→off); `internal/status/status.go` (read config); `cmd/pk/main.go` (flag routing).
- **Skills/docs:** `internal/setup/skills/conventions/SKILL.md` (+ local copy + hash); `docs/pk-json.md`, `pk-guard.md`, `pk-preserve.md`, `pk-setup.md`, `pk-status.md`, `adoption.md`, `error-reference.md`.
- **Tests:** `config` (resolvers/defaults), `guard`/`preserve` (modes from config incl. off), `setup` (`.pk.json` write + field-merge + migration + bare hooks + key order), `status` (reads config), `claude_test` (bare hooks).

## Verification

**Automated:** `make test`, `make lint`, `make vuln`. New/updated unit tests per above — especially: setup writes correct `.pk.json` and bare hooks; field-merge keeps `branches`; migration of `pk guard --ask --push-guard block` → `{mode:ask,push:block}` and of bare `pk guard` → `{mode:block,push:off}` (preserved, not flipped); guard/preserve honor `off`; deprecated flags override config.

**Smoke (`make build VERSION=v9.9.9` for a non-dev binary):**
1. Fresh `git init` + `pk setup` → `.pk.json` has `guard.mode/push` + `preserve.mode` written explicitly (`block`/`block`/`manual`); `settings.json` hooks are bare; `pk status` shows the modes.
2. Edit `.pk.json` `"mode":"off"` → `pk guard` on a protected-branch git mutation allows it (no-op); `"push":"ask"` → a `git push` prompts. (Drive via `printf '{"tool_input":{"command":"git push"},"cwd":"."}' | dist/pk guard`.)
3. Migration: seed a project with an old `pk guard --ask --push-guard block` hook and no mode keys → `pk setup` lifts `{mode:ask,push:block}` into `.pk.json` and rewrites the hook bare; a project with bare `pk guard` migrates to `push:"off"` (not block).
4. Field-merge: a `.pk.json` with `guard.branches` only → `pk setup` adds `mode`/`push` without dropping `branches`.
5. Back-compat: bare `dist/pk guard` with `guard.mode:"off"` in `.pk.json` → no-op; same binary with an explicit `--push-guard block` → blocks (override works).
