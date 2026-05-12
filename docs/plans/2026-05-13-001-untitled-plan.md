## Context

In the alert-monitor repo, `/init` asked branch configuration questions (develop/main) but never wrote `.pk.json`. When `/ship` ran later, `pk release` fell through to trunk flow — it tagged and pushed `develop` but never merged into `main`. The CI workflow triggers on `push: branches: [main]`, so the deploy never fired. Manual intervention was needed.

Two distinct failures combined:
1. The `/init` skill's step 7 (write `.pk.json`) was skipped by the model after the big CLAUDE.md confirmation flow in step 6
2. `pk release` silently treated the missing config as intentional trunk flow, with no indication in its output

## Changes

### 1. Init skill: reorder steps so .pk.json is written earlier

Move `.pk.json` creation from step 7 (after CLAUDE.md write) to step 5 (immediately after questions, before drafting conventions). The model currently asks questions in step 4, drafts conventions in step 5, shows and confirms CLAUDE.md in step 6, then is supposed to write `.pk.json` in step 7. After the big interactive confirmation + file write in step 6, the model treats the task as done and skips step 7.

With the reorder: questions (step 4) flow directly into config write (step 5), then convention drafting and CLAUDE.md follow. The `.pk.json` write gets the natural momentum of "you just asked these questions, now write the answers."

Add a rule: "Always write .pk.json immediately after config questions when any config was provided. Do not defer it past CLAUDE.md."

Files:
- `internal/setup/skills/init/SKILL.md` (embedded source)
- `.claude/skills/init/SKILL.md` (deployed copy, recompute `pk_sha256`)

Step renumbering:
- Step 4: Ask config questions (unchanged)
- Step 5: **Write .pk.json** (moved from old step 7) — same conditional logic, same merge behavior
- Step 6: Draft conventions (was step 5)
- Step 7: Show conventions, confirm, write CLAUDE.md (was step 6)
- Step 8: Baseline nudge (unchanged)

### 2. pk release: label trunk flow explicitly in output

When no `release.branch` is configured and trunk flow is active, add a label to the pre-flight output:

```
  Trunk flow (no release.branch in .pk.json)
  On develop branch
```

Current output only shows `On develop branch` — no indication that merge was skipped due to missing config vs. intentional trunk design. The label makes misconfig visible during `pk release --dry-run` (which `/ship` shows before confirming).

Files:
- `internal/release/release.go` — line 208-211, add flow label before the branch line
- `internal/release/release_test.go` — update trunk flow tests to check for the new label

### 3. No changes to /ship or pk changelog

- `/ship` already shows `pk release --dry-run` output for user confirmation — the new trunk flow label will be visible there
- `pk changelog` is not affected by `release.branch` — it doesn't use this config for its behavior
- Adding a `.pk.json` existence check to `/ship` would be heuristic (trunk flow legitimately doesn't need it)

## Verification

- `make test` — updated release tests pass
- `make build`
- Smoke: run `pk release --dry-run` in a repo on `develop` with no `.pk.json` — output shows "Trunk flow (no release.branch in .pk.json)"
- Read both skill files to confirm step reordering and `pk_sha256` update
