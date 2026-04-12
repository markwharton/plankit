# Plan: Cloud sandbox compatibility for plankit

## Context

plankit's protective hooks (`pk guard`, `pk preserve`, `pk protect`) currently degrade silently in any environment where the `pk` binary is not on PATH. Local CLI, Desktop app, and VS Code extension all work because they execute on the developer's machine and inherit the shell PATH (`make install` puts `pk` in `$GOPATH/bin`). The web sandbox is where it breaks: a fresh Ubuntu 24.04 VM per session, no `pk` installed, hooks return command-not-found exit 127 — which is a non-blocking error by design, so they degrade rather than alert.

This plan addresses the gap in two phases:

- **Phase 1** — harden the build for static linkage (so the linux release artifacts are guaranteed to run in any sandbox) and update the docs to honestly describe the current degradation. Low risk, no testing dependency, ships as a patch.
- **Phase 2** — write a SessionStart bootstrap that `pk setup` installs unconditionally, so opening any plankit-configured repo in the web sandbox automatically downloads the matching `pk` binary. Requires real-sandbox testing before shipping; ships as a minor.

The two phases are bundled in one plan because phase 1 is the foundation for phase 2 (CGO_ENABLED=0 is what guarantees the linux binary downloaded in phase 2 actually runs without a glibc dependency).

## Design principles (apply to both phases)

- **`pk setup` configures, users don't hand-write.** Anything plankit needs in a repo must be writable by `pk setup` — no copy-paste setup instructions. Captured in memory: `project_pk_setup_configures.md`.
- **Idempotent install scripts are the enabler.** They let `pk setup` write the bootstrap unconditionally because the script itself decides when to act (no-op locally, install on web).
- **Pinned version, never "latest".** Version drift between the repo's committed script and the binary it downloads is a class of bug we should never ship.

## Phase 1 — Build hardening + docs honesty (commit now, ships as v0.7.1)

### Files to modify

1. **`Makefile`** — add after the `VERSION?=dev` line (line 2):

   ```makefile
   # Force pure-Go static binaries; prevents implicit glibc dependency on linux.
   export CGO_ENABLED := 0
   ```

   `export` makes every recipe inherit it; no need to modify each `go build`/`go install` line individually. Verified: plankit's current import set already produces a pure-Go binary, so this is a guardrail (preventing future imports of `net`/`os/user` from silently introducing dynamic linkage), not a behavior change. Native and cross-compiled binaries are byte-for-byte identical with and without the flag today.

2. **`.github/workflows/release.yml`** — add `CGO_ENABLED: 0` to the build step's `env:` block (currently line 42–44, alongside `GOOS` and `GOARCH`). Same guardrail in CI, applies to all five matrix targets.

3. **`CLAUDE.md`** — add one bullet under "Language & Build" (after "All user messages to stderr..."):

   > - `CGO_ENABLED=0` enforced via Makefile — pure-Go static binaries, no implicit glibc dependency on linux.

   This is the hidden-invariant rule that catches future drift if a contributor removes the flag or imports a cgo-dependent stdlib package.

4. **`README.md`** — append to "Known Limitations" (after the Ultraplan bullet, line 69):

   ```markdown
   - **Claude Code on web/mobile**: plankit's protective hooks (`pk guard`, `pk preserve`, `pk protect`) require the `pk` binary on PATH. The web sandbox doesn't ship `pk`, and mobile has no shell environment — hooks degrade to no-ops rather than failing loudly. Skills, rules, and CLAUDE.md still load from `.claude/`.
   ```

   Honest description of the gap. Phase 2 fixes it; phase 1 documents it.

### Verification (phase 1)

- `make build` succeeds; `dist/pk` is byte-identical to a v0.7.0 build (same SHA after stripping version-stamp differences).
- `make build-all` succeeds for all five targets.
- `make test` passes.
- README and CLAUDE.md render correctly (visual check).
- `pk changelog --dry-run` shows the expected v0.7.1 patch entries.

## Phase 2 — SessionStart bootstrap (deferred until sandbox testing, ships as v0.8.0)

### What `pk setup` writes into the repo

**A. New embedded script: `internal/setup/template/install-pk.sh`**

```bash
#!/usr/bin/env bash
# .claude/install-pk.sh — bootstrap pk in cloud sandboxes; no-op locally.
# Written by `pk setup`. Re-run `pk setup` to update the pinned version.
set -euo pipefail

command -v pk >/dev/null 2>&1 && exit 0

PK_VERSION="{{.Version}}"
install_dir="$HOME/.local/bin"
mkdir -p "$install_dir"

arch="$(uname -m)"
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 0 ;;
esac

url="https://github.com/markwharton/plankit/releases/download/${PK_VERSION}/pk-linux-${arch}"
curl -fsSL "$url" -o "$install_dir/pk"
chmod +x "$install_dir/pk"

echo "export PATH=\"$install_dir:\$PATH\"" >> "$CLAUDE_ENV_FILE"
```

The `{{.Version}}` is template-substituted at write time from `internal/version` — not hard-coded. This means:
- Embedded source contains the placeholder, never a literal version.
- When `pk setup` runs, it reads its own version and writes the literal into the committed script.
- When plankit releases a new version, the version package gets bumped via the existing release flow, and the next `pk setup` run picks it up automatically. **There is nothing in the plankit codebase to manually bump.**

**B. Settings.json hook entry**

`pk setup` adds a SessionStart entry to `.claude/settings.json` alongside the existing PreToolUse / PostToolUse:

```json
"SessionStart": [
  {
    "matcher": "*",
    "hooks": [
      { "type": "command", "command": ".claude/install-pk.sh", "timeout": 30 }
    ]
  }
]
```

### Code changes for phase 2

- **`internal/setup/setup.go`** — add a new `//go:embed template/install-pk.sh` block alongside the existing `templateFS`. Add a `SessionStart []HookEntry` field to `HooksConfig` (after `PostToolUse`). Update `buildHookConfig()` to include the SessionStart entry. Add a `writeInstallScript()` function that template-substitutes `{{.Version}}` from `internal/version` and writes to `.claude/install-pk.sh` with `0755` permissions.
- **`internal/setup/template/install-pk.sh`** — new file, the script template above.
- **`internal/setup/setup_test.go`** — add tests covering: script is written, script has correct version substituted, SessionStart hook entry is present, idempotent re-run doesn't corrupt.
- **`internal/version/version.go`** — no changes; we read from it.
- **`internal/version` consumer (new)** — `pk version` gains a "drift detection" check: if `.claude/install-pk.sh` exists in the current directory, parse its `PK_VERSION` and compare against `internal/version.version`. If local > script, warn: `your committed .claude/install-pk.sh pins <X> but you're running <Y> — re-run 'pk setup' to update`. Enforced detection, not just a doc note.
- **`docs/pk-setup.md`** — document the SessionStart hook and the upgrade flow ("re-run `pk setup` after upgrading plankit to update the pinned version in `.claude/install-pk.sh`").
- **`docs/pk-version.md`** — document the drift warning.
- **`README.md`** — replace the "Known Limitations" bullet from phase 1 with a positive description ("On Claude Code on the web, `pk setup` installs a SessionStart hook that fetches the matching `pk` binary into the sandbox at session start").

### Behavior matrix (phase 2)

| Surface | What happens at session start |
|---|---|
| Local CLI / Desktop / VS Code | SessionStart fires → `command -v pk` finds local install → script exits 0 immediately. No download, no PATH change. User never notices. |
| Web sandbox | Fresh VM → repo cloned → SessionStart fires → no `pk` on PATH → downloads `pk-linux-{amd64\|arm64}` from GitHub releases into `$HOME/.local/bin` → appends `export PATH=...` to `$CLAUDE_ENV_FILE` → subsequent `pk guard`, `pk preserve`, `pk protect` hooks find it. |
| Mobile | Presumed to share the web-sandbox code path. Unverified. Lowest priority. |

### Open verification (must test in a real web sandbox before shipping phase 2)

These are the questions the docs don't answer. Each requires running the SessionStart hook in an actual cloud session and inspecting behavior:

1. **`$HOME/.local/bin` writability.** Confirm the SessionStart hook user can `mkdir` and `chmod +x` here. The docs don't specify what user SessionStart hooks run as in cloud (only setup scripts are documented as running as root).
2. **Architecture detection.** Confirm `uname -m` returns `x86_64` (presumed) or `aarch64`. Adjust the case statement if cloud is something unusual.
3. **`$CLAUDE_ENV_FILE` availability at SessionStart.** The docs mention `CLAUDE_ENV_FILE` for hooks generally; verify it's set specifically when SessionStart fires (not just for PreToolUse / PostToolUse).
4. **`curl` availability.** Documented as preinstalled in cloud, but verify in practice from the SessionStart hook context.
5. **Failure visibility.** If `curl` fails (network blip, GitHub rate limit, missing release artifact), confirm stderr surfaces somewhere user-visible. Hooks exit 0 by design, so failures must shout.
6. **Cold-start latency.** Measure the time added to first session start. If it's > a few seconds, consider caching or a more aggressive bootstrap.
7. **Idempotence on session resume.** Confirm the script no-ops cleanly when re-fired during a resumed session (not just on first start).

### Verification (phase 2, after sandbox testing)

- All seven open questions above answered and resolved.
- `make test` passes with new setup_test.go cases.
- `pk setup` in a fresh repo creates `.claude/install-pk.sh` with the correct embedded version.
- `pk setup` re-run after a version bump rewrites the script with the new version.
- `pk version` warns when local version > script version.
- Local smoke test: SessionStart hook fires on plankit's own repo, exits immediately (because `pk` is already on PATH), no PATH change observed.
- Web sandbox smoke test: open plankit in claude.ai/code, confirm `pk` is on PATH after SessionStart, confirm `pk guard` blocks a test commit on a protected branch.

## Out of scope

- **Setup-script integration** for the cloud sandbox. Setup scripts in Claude Code on the web are stored in the cloud-environment UI, not committed to the repo. `pk setup` cannot write them, so they violate the "configures, doesn't hand-write" principle. We use SessionStart hooks instead.
- **`CLAUDE_CODE_REMOTE` branching.** The script's idempotence covers both local and cloud cases without needing to detect environment. Simpler is safer.
- **Mobile-specific testing.** Same code path as web; lower priority.
- **Auto-bumping `PK_VERSION` in existing committed scripts** when plankit releases a new version. The drift detection in `pk version` warns developers; bumping is an explicit `pk setup` re-run, opt-in like everything else plankit-managed.
- **Migration of existing repos.** Users pick up the SessionStart hook by re-running `pk setup` after upgrading. No special migration logic.
- **Caching of the downloaded binary across sessions.** Sandboxes are ephemeral by design; trying to cache fights the platform. Re-download per session is the correct posture.

## Release pacing

- **v0.7.1 (patch)** — Phase 1. Includes the existing `dev` work: refactor `7c5db28` and docs/skills `b63c80e`, plus the four phase 1 edits. Ready when you decide to release.
- **v0.8.0 (minor)** — Phase 2. Ships after sandbox testing. SessionStart bootstrap is genuinely new functionality (`feat:`), so minor under semver.

The natural cadence is: cut v0.7.1 soon to lock in the foundation, then carry phase 2 on `dev` until you can test it in the actual web sandbox.

## Critical files to read before implementing

### Phase 1
- `Makefile` — current build flags and targets.
- `.github/workflows/release.yml` — current CI build configuration.
- `README.md` — current "Known Limitations" section.
- `CLAUDE.md` — current "Language & Build" section.

### Phase 2
- `internal/setup/setup.go` — embed pattern (lines 19–26), `HooksConfig` struct (lines 46–49), `buildHookConfig()` (lines 51–78), the existing PreToolUse / PostToolUse plumbing.
- `internal/setup/setup_test.go` — test patterns to follow for the new script-write tests.
- `internal/version/version.go` — version package, source of `{{.Version}}` substitution.
- `internal/setup/skills/release/SKILL.md` — example of an embedded file with frontmatter, for reference on embed patterns.
- `internal/guard/guard.go` — example of an existing pk-binary hook command, for the SessionStart entry's surrounding context in `.claude/settings.json`.
