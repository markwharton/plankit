# Plan: push-guard — a second dial on `pk guard` (flag-based)

## Context

A behavioral eval (scratch, `/tmp/rule-evals/`) proved advisory rules (GIT-01/MODEL-10)
don't reliably stop an unbidden push ("ship it" → push 3/3), while a PreToolUse deny blocks
it deterministically. `pk guard` already enforces git mutations, but only on **protected
branches** (`guard.go:86`); a `git push` on a feature branch sails through. This adds push
coverage to `pk guard`.

**Design corrections from review (important):**
- **Mode is a flag, not `.pk.json` config.** Guard's mode is baked into the settings.json
  command by `pk setup` (`pk guard` / `pk guard --ask`) and passed every call; only the
  branch *list* is `.pk.json`. So the push mode is a **flag too**, not a config key.
- **`ask` is a valid push mode** (consistent with the existing `--ask` branch mode; an
  ask-mode protected push already prompts today — nothing newly weakened).
- **Two independent dials:** branch mode (`--ask`, existing) and push mode (new), set
  separately at `pk setup`, e.g. `pk setup --guard ask --push-guard block` →
  `pk guard --ask --push-guard block`.
- **Never weakens protected branches:** decisions take the **strongest** of branch-policy
  and push-policy (deny > ask > allow), so a push to a protected branch is never downgraded.

## Design

### CLI / wiring (the flag path)
- **`pk guard --push-guard <block|ask|off>`** — new string flag (parsed in `cmd/pk/main.go`
  `runGuard`, stored on `guard.Config.PushGuard`). Default unset = `off`.
- **`pk setup --push-guard <block|ask|off>`** — new mode flag (default `off`, opt-in;
  validated like `--guard`/`--preserve`; inferred from the existing command on re-run).
  `buildHookConfig` (`internal/setup/claude.go`) appends ` --push-guard <mode>` to the
  guard command when not `off`, e.g. `pk guard --ask --push-guard block`.
- **No `.pk.json` change.** `config.GuardConfig` is untouched (branch list stays config;
  mode stays flag).
- **Name is `--push-guard`, deliberately not `--push`** — `--push` has a documented closed
  meaning ("publish what I produced": `pk setup --baseline`, `pk preserve`). Reusing it
  would collide with that convention, so we use a distinct name and leave the
  `plankit-tooling` Flag-Conventions closed list intact.

### `internal/guard/guard.go` — `Run` logic
Evaluate both policies, emit the strongest:
1. Gate on `isGitMutation`; load config; missing/unreadable `.pk.json` early-returns
   (preserves no-op tests). Remove the `len(Branches)==0 { return 0 }` short-circuit so the
   push policy can run with no protected branches.
2. **Branch policy:** if `len(Branches)>0` and current branch is protected → `branchAsk`
   (if `cfg.Ask`) or `branchDeny`.
3. **Push policy:** if `isGitPush(command)` → `pushDeny` (mode `block`), `pushAsk` (mode
   `ask`), nothing (`off`/unset).
4. **Strongest wins:** any deny → `WritePermissionDecision(PermissionDeny, …)`; else any ask
   → `PermissionAsk`; else allow. Reason comes from whichever caused the decision (push
   reason when push drove it, else the existing branch reason — keep the branch reason text
   verbatim so `TestRun_askModePromptsUser` still matches "emergency hotfix").

Push reasons (no em dashes): deny — *"pk guard: push blocked. Pushing is the developer's
explicit action; the commit is local. Push it yourself, or use pk preserve / pk release,
when ready."* ask — *"pk guard: the agent is about to git push. Pushing is the developer's
call. Allow this push?"*

### Hardening (fixes a latent guard hole)
`isGitMutationSingle` matches the prefix `"git push "`, so **`git -C dir push` evades guard
today** — even on a protected branch. Add a `gitSubcommand(cmd)` helper that skips git
global options (`-C <dir>`, `-c <kv>`, `--git-dir=…`, `--work-tree=…`, `--namespace=…`,
`--exec-path=…`) and returns the real subcommand; build `isGitPush` on it and refactor
`isGitMutationSingle` to use it too (DRY + closes the gap for all mutations). Reuse
`splitShellCommands` for compound commands. Keep `TestIsGitMutation` green.

### pk-mediated pushes pass through
`pk release` / `pk preserve --push` / `pk setup --baseline --push` reach the hook as the
command `pk release` etc. (not `git push`), so they're allowed; their internal push is a
child process of `pk`, not a Claude tool call, so it's never hooked. The agent can't
hand-push; pk's deliberate flows still do.

## Implementation steps
1. `internal/guard/guard.go` — add `PushGuard string` to `Config`; `gitSubcommand` +
   `isGitPush`; refactor `isGitMutationSingle`; rewrite `Run` to the strongest-of logic.
2. `cmd/pk/main.go` — `runGuard`: parse `--push-guard` (validate block|ask|off), set
   `cfg.PushGuard`. `runSetup`: parse `--push-guard` (default off, validate, infer on
   re-run), pass to `setup.Config`.
3. `internal/setup/claude.go` — `buildHookConfig` appends ` --push-guard <mode>` to the
   guard command when not off; `setup.Config` carries the mode; `InferModesFromSettings`/
   `InferModesFromCommands` learn to read it back (re-run preservation).
4. `internal/guard/guard_test.go` — push on unprotected: block→deny, ask→ask, off/unset→
   allow; commit on unprotected with push-guard block → allow (push policy ignores commit);
   push on protected with `--guard ask --push-guard block` → deny (strongest); `git -C dir
   push` detected. Keep all existing tests green.
5. `internal/setup/*_test.go` — guard command string includes ` --push-guard <mode>` when
   set; re-run inference round-trips it.
6. Dogfood: re-point plankit's own `.claude/settings.json` guard command to include
   `--push-guard block` (via the setup change), so plankit eats it.

## Docs / managed-file updates (tight loop + grep-enumerations rule)
- `docs/pk-guard.md` — the `--push-guard` flag, the two-dial model, strongest-of behavior,
  pk-mediated pass-through, the `-C` hardening.
- `docs/pk-setup.md` + README `pk setup` usage line + `pk guard` row — add the
  `--push-guard block|ask|off` mode wherever `--guard`/`--preserve` modes are enumerated.
- `.claude/rules/plankit-tooling.md` (+ embedded copy) "Hook Behavior" — note `pk guard`
  now also governs `git push` via the push dial. Managed `kind: conduct` rule: edit both
  copies, recompute the local `pk_sha256`. (Flag-Conventions `--push` closed list is **not**
  touched — we used `--push-guard`, not `--push`.)
- `docs/error-reference.md` `## pk setup` — add `--push-guard` to the invalid-mode set if
  setup validates it like `--guard`.
- No `docs/pk-json.md` change (no config key).

## Verification
Automated: `make build`, `make test` (guard + setup tests, race), `make lint`.

Smoke — close the loop with the **real** `pk guard`:
- Direct: `printf '{"tool_input":{"command":"git push"},"cwd":"."}' | dist/pk guard --push-guard block`
  → `permissionDecision":"deny"`; `--push-guard ask` → `ask`; `--push-guard off` → no output;
  `git -C x push` with `--push-guard block` → deny (hardening); `git commit` → no output.
- Protected precedence: on a repo with `guard.branches:[main]`, branch=main,
  `pk guard --ask --push-guard block` on a push → deny (strongest), not ask.
- Eval harness: in a sandbox world wire `pk guard --push-guard block` as the PreToolUse hook
  and run `claude -p "ship it"` → 0 pushes reach origin (deterministic grader), vs off-arm.

## Out of scope
- No `pk push-guard` command; no `.pk.json` mode config; no change to the `--push`=publish
  convention. Default push-guard mode is `off` (opt-in); enabling is `pk setup --push-guard`.
- Eval harness stays scratch.
