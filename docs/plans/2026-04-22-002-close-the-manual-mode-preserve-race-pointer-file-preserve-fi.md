# Close the manual-mode preserve race: pointer file + preserve-first rule

## Context

Shared plan space at `~/.claude/plans/` is unavoidable — every Claude Code session writes plans there. In manual mode (plankit's default), the PostToolUse hook on ExitPlanMode runs `pk preserve --notify`, which only surfaces a reminder. The bug scenario that prompted this plan:

1. Session A approves a plan via ExitPlanMode; the notify hook fires.
2. The user types `/preserve` immediately afterward.
3. Claude does **not** dispatch the skill — it treats the turn as a "go-ahead" signal and starts implementation. The `/preserve` invocation sits queued.
4. Implementation takes 10 minutes. During that window, Session B (another Claude session in another project) approves its own plan; the shared `~/.claude/plans/` directory now has a newer mtime.
5. Claude finally processes `/preserve`. The skill runs `pk preserve` with no hook stdin payload, so it falls back to `findLatestPlan()` at `internal/preserve/preserve.go:293`, which sorts by mtime — and picks Session B's plan.
6. Wrong plan preserved to Session A's `docs/plans/`.

Two independent fixes are needed:

1. **Selection robustness** — `/preserve` should reference the exact plan file that was approved, not do an mtime scan. The `--notify` hook has the path (via `extractPlanPath` at `internal/preserve/preserve.go:205`); it can hand that path to the skill via a project-local pointer file so the race window is closed even if `/preserve` executes minutes later.

2. **Timing** — Claude should honor `/preserve` as an immediate action when the user types it, not treat it as a go-signal and queue the actual skill dispatch behind implementation. Fixed via a plankit-tooling rule addition — Claude-behavior instruction, not a change to the notify message (which is addressed to the user, not to Claude).

## Changes

### 1. Add pointer-file read/write to `pk preserve`

File: `internal/preserve/preserve.go`

**Location and format.** Pointer is written to `<projectDir>/.git/pk-pending-plan`. Using `.git/` follows git's precedent for tooling state (`COMMIT_EDITMSG`, `MERGE_MSG`, `rebase-apply/`), is per-repo by construction, and is never tracked — no `.gitignore` coordination needed. `pk preserve` already bails when not in a git repo (`preserve.go:124`), so the `.git/` directory is guaranteed to exist. Content: one line, absolute path to the plan file in `~/.claude/plans/`.

**Write side — `--notify` mode** (currently `preserve.go:98-106`):

Move the project-dir resolution block (`preserve.go:108-121`) up so it runs *before* the notify block. Then extend the notify branch:

- If `extractPlanPath` returned a non-empty path and that file exists, write the path to `<projectDir>/.git/pk-pending-plan` (best-effort — log stderr on failure, do not change exit behavior).
- Continue with the existing `writeHookResponse` call.

**Read side — skill invocation (no stdin)** (currently `preserve.go:69-73`):

When `hooks.ReadInput()` fails (skill invocation has no hook payload), before falling back to `findLatestPlan()`:

- Determine `projectDir` from `CLAUDE_PROJECT_DIR` or `cfg.Getwd()`.
- Try reading `<projectDir>/.git/pk-pending-plan`. If present and the pointed-to file exists, use that path.
- Otherwise fall back to `findLatestPlan()` (existing behavior).

**Cleanup — after successful preservation** (currently `preserve.go:189-197`):

After the git commit succeeds, delete `<projectDir>/.git/pk-pending-plan` if it exists. Best-effort — stale pointer never blocks preservation. The next `--notify` cycle overwrites it anyway.

**Stale-pointer handling.** If the pointer is present but the pointed-to file no longer exists (e.g., Claude Code cleaned `~/.claude/plans/`), treat as absent and fall back to `findLatestPlan()`. Delete the stale pointer.

### 2. Notify message — leave as-is

The notify hook's `systemMessage` and `additionalContext` at `internal/preserve/preserve.go:101-104` stay unchanged. The notification's purpose is to inform the user that `/preserve` is available — it is not a place to instruct Claude to auto-run the skill. Respecting that contract is a design constraint, not just cosmetics.

### 3. Update the plankit-tooling rule to make `/preserve` dispatch-immediately

File: `internal/setup/rules/plankit-tooling.md` (embedded) and `.claude/rules/plankit-tooling.md` (local copy).

Current line 17:
> - **`pk preserve` runs after exiting plan mode.** Behavior depends on project configuration — it may preserve automatically or notify that a plan is ready.

New — add a trailing sentence directing Claude's behavior when the user invokes the skill (not when the hook fires):
> - **`pk preserve` runs after exiting plan mode.** Behavior depends on project configuration — it may preserve automatically or notify that a plan is ready. If the user types `/preserve`, dispatch the skill as your next action — never queue it behind implementation work. `/preserve` is an explicit request, not a go-signal for something else.

Recompute `pk_sha256` for the local copy:
```bash
sed -n '/^---$/,/^---$/!p' internal/setup/rules/plankit-tooling.md | shasum -a 256
```

### 4. Tests

File: `internal/preserve/preserve_test.go`

Add these subtests to `TestRun`:

- **`notify writes pointer file with plan path`** — run with `Notify: true` and a valid hook payload; assert `<projectDir>/.git/pk-pending-plan` exists and contains the expected path. No git commit should happen (existing notify assertion).

- **`skill invocation reads pointer, preserves correct plan under race`** — seed `~/.claude/plans/old.md` (approved plan, mtime T0) and `~/.claude/plans/newer.md` (rival plan, mtime T0+1m). Seed `<projectDir>/.git/pk-pending-plan` pointing at `old.md`. Run with no stdin (skill invocation) and `Notify: false`. Assert: the preserved plan content matches `old.md`, not `newer.md`. This is the regression guard for the original bug.

- **`pointer removed after successful preserve`** — run the same setup; assert `<projectDir>/.git/pk-pending-plan` does not exist after Run() returns successfully.

- **`stale pointer falls back to findLatestPlan`** — seed pointer → `/does/not/exist.md`, one real plan in `~/.claude/plans/`. Run as skill invocation. Assert: the real plan is preserved, and the stale pointer is deleted.

- **`no pointer, no stdin → fallback to latest`** — existing behavior; keep a test that covers this path explicitly (may already exist at `preserve_test.go:543`). Verify it still passes with new code.

### 5. Documentation

File: `docs/pk-preserve.md`

Add a short `### Race safety` subsection under `## Details` documenting:
- `~/.claude/plans/` is shared across Claude sessions; mtime-based selection is race-prone when multiple sessions are active.
- In manual mode, `pk preserve --notify` writes `<projectDir>/.git/pk-pending-plan` so the `/preserve` skill can reference the exact approved plan.
- The pointer is automatically cleaned up after successful preservation.

## Out of scope

- **Default-mode change.** Manual stays the default per plankit philosophy ("safe defaults, opt-in for escalation"). Users can still set `auto` mode in `.pk.json` if they want the hook to preserve immediately.
- **Positional arg `pk preserve <path>`.** Pointer handles the skill path automatically; no CLI change needed. Power users who want an explicit override can be added later if the need arises.
- **`.gitignore` updates.** Pointer lives in `.git/`, always untracked.
- **Cross-session mutual exclusion.** No locking — two sessions approving in the same project is still last-write-wins on the pointer, which is an acceptable edge case.
- **Hook config changes in `internal/setup/setup.go`.** Notify invocation and auto-preserve paths are unchanged; only the behavior of `pk preserve --notify` and the no-stdin path is extended.

## Verification

1. **Build**: `make build` — clean build, `dist/pk` produced.
2. **Unit tests**: `make test` — all tests pass, including new subtests.
3. **SHA regeneration**: after updating `internal/setup/rules/plankit-tooling.md`, recompute hash with the `sed | shasum` command and update `.claude/rules/plankit-tooling.md`. Verify `dist/pk setup --project-dir $(mktemp -d -t pk-test)` emits a rule file whose `pk_sha256` matches the local copy.
4. **Smoke test — race safety (the original bug)**:
   - Create two `.md` files in `~/.claude/plans/`: `plan-a.md` (mtime = now-1h) and `plan-b.md` (mtime = now).
   - Run `echo '{"tool_response":"/Users/.../.claude/plans/plan-a.md","cwd":"<projectDir>"}' | pk preserve --notify` to simulate Session A's ExitPlanMode hook.
   - Confirm `<projectDir>/.git/pk-pending-plan` exists and contains `plan-a.md`'s path.
   - Run `pk preserve` (no stdin, simulating the skill).
   - Confirm the commit references `plan-a.md`'s content, not `plan-b.md`.
   - Confirm `<projectDir>/.git/pk-pending-plan` is gone.
5. **Smoke test — fallback still works**:
   - Delete any pointer; confirm `pk preserve` with no stdin still falls back to the latest plan (no regression for fresh clones / first-time users).
6. **Smoke test — notify message unchanged**:
   - Simulate the notify hook as in (4) and inspect stdout. Confirm `additionalContext` still matches the original "Inform the user that they can type /preserve…" wording (regression guard against accidentally changing the user-facing message).
7. **Manual end-to-end**:
   - Rebuild + `pk setup` in a scratch project.
   - Open Claude Code, enter plan mode, approve via ExitPlanMode.
   - Type `/preserve`, verify Claude dispatches the skill immediately (no other work first).
   - Verify the approved plan is saved (not some stale plan from another project).
