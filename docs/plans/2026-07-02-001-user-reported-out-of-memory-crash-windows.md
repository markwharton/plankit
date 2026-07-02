# User-reported out-of-memory crash (Windows)

## Context

A user reported a Go runtime fatal OOM stack trace appearing on a `git checkout` Bash call in a Claude Code session on a Windows machine (a Node.js project, Git Bash environment). The environment is a VDI with 12 GB RAM running at ~80% used. The initial hypothesis was a mis-sized static 256-byte string/buffer in pk. The goal of this session: root-cause the crash and decide what, if anything, changes in plankit.

## Which binary crashed

Confirmed by the reporting session's own findings: the `pk guard` PreToolUse hook (wired to fire on every `Bash|PowerShell` call, `internal/setup/claude.go:116`). The session attributed the memory pressure to Chromium (Playwright E2E) + Vite dev servers running concurrently on the VDI. It was a one-off: the crash blocked one command mid-session and recovery was a simple retry once memory freed.

## Diagnosis (evidence-based)

**The crash is environmental (system-wide memory exhaustion on the reporter's machine), not a pk allocation bug.** Three independent lines of evidence:

1. **The trace contains no pk frame.** Every frame is Go toolchain source (`src/runtime/malloc.go`, `src/runtime/proc.go`, `src/internal/cpu/cpu_x86.go`, `src/runtime/asm_amd64.s`). The failing 256-byte (`0x100`) allocation is requested by `internal/cpu.doinit` — the Go runtime's own CPU-feature table — during `runtime.schedinit`, called from `rt0_go` (the process entry point). This runs before `main()` and before any package `init()`; no application string exists yet. Any Go binary would crash identically on that machine at that moment.
2. **No 256-byte buffer exists in pk.** `grep -rnE '256|make\(\[\]byte'` over `cmd/` + `internal/` (non-test): all "256" hits are SHA-256 naming; the repo's single sized buffer (`internal/changelog/changelog.go:699`) is exactly right-sized to its input.
3. **Mechanism:** `mallocgc` failing on a tiny request means Windows refused to commit heap pages (`VirtualAlloc` failure — commit charge = RAM + pagefile exhausted). A mis-sized 256-byte buffer cannot exhaust memory; guard lives milliseconds (reads one small JSON payload, string-scans the command, exits) and the OS reclaims everything at exit. Guard is simply the most frequently spawned process (every Bash/PowerShell call), so it was likely the canary.
4. **The environment fits:** a 12 GB VDI at ~80% used leaves ~2.4 GB of headroom, and VDIs commonly run small or fixed pagefiles, so the commit limit sits close at all times. A Claude Code session in a Node project routinely spikes a couple of GB (npm/build tooling, the CLI itself, background tasks); once commit is exhausted, the next process to spawn is refused memory — exactly a hook binary dying in runtime bootstrap.

Secondary findings from exploration:

- **Crash semantics (verified against current Claude Code docs):** for PreToolUse hooks, **exit 2 blocks the tool call** (stderr fed to Claude); any *other* non-zero exit (1, 127) is non-blocking. Go runtime fatal errors exit with status 2, so a crashed guard binary **fails closed** — the command is blocked and a retry succeeds once memory frees. This matches exactly what the crash session observed ("blocked one command mid-session; recovered on retry"). Missing binary (exit 127) remains fail-open, as `docs/adoption.md:217` already documents.
- **Doc inaccuracy found:** plankit's root `CLAUDE.md:114` states "Any non-zero exit (including command-not-found 127) is non-blocking" — wrong for exit 2. This file is maintainer-authored (no `pk:sha256` marker; the embedded template doesn't contain the Hook Protocol section), so it's a plain edit. `docs/pk-guard.md` Hook protocol (lines 48–52) covers only the happy path; `docs/error-reference.md` has no `pk guard` section.
- Embedded assets total ~44 KB and are untouched by guard; binary footprint is ordinary for static Go.

## Plan

### 1. Advice for the reporter (no repo change — a message to relay)

Cause is confirmed (Chromium/Playwright + Vite memory pressure on a 12 GB VDI), so this is mitigation, not diagnosis:

- The crash is benign for pk: guard failed closed (command blocked) and a retry succeeded; nothing was left unguarded and no state was corrupted.
- Check the pagefile isn't disabled or capped small (Advanced system settings → Performance → Virtual memory). On a VDI the pagefile is often fixed by the image; if it can't be raised locally, that's a request to the VDI admin (more RAM or a larger pagefile).
- If it recurs, Windows Event Viewer → System log → **Event ID 2004** (Resource-Exhaustion Detector) names the processes holding the most commit charge at the time.
- Practical mitigation on 12 GB: avoid running Playwright E2E (Chromium) while extra dev servers are up; close what isn't needed during sessions.

### 2. Correct and document hook crash semantics (only repo change; on `develop`)

- `CLAUDE.md:114` (Hook Protocol section, maintainer-authored) — correct the exit-code semantics: for PreToolUse, exit 2 blocks the tool call (stderr fed to Claude); any other non-zero exit (1, 127) is non-blocking. Check the adjacent PostToolUse line (115) for accuracy in the same pass (per current docs: PostToolUse cannot block; exit 2 shows stderr to Claude informationally).
- `docs/pk-guard.md` — add a short crash-semantics caveat to the Hook protocol section (lines 48–52): a crashed guard binary (Go fatal errors exit 2) blocks the command rather than failing open; retrying after the cause clears succeeds. A missing binary (exit 127) is non-blocking, consistent with `docs/adoption.md:217`.
- `docs/error-reference.md` — add a `## pk guard` section following the existing pattern (`### <error>` / fenced example / **Cause** / **Fix**) with an entry for the Go runtime OOM-at-startup signature (`runtime.mallocgc` / `runtime.schedinit` trace before any pk output): cause is system memory exhaustion (commit charge), not pk; fix is freeing memory / checking the pagefile, with the Event Viewer Event ID 2004 pointer; note the command is blocked and a retry succeeds once memory frees.
- Follow doc conventions: no em dashes in error-reference quoted messages; keep entries concise. README.md:116 already links error-reference; no concept change, so README body doesn't change. `docs/plans/` files that mention the old semantics are immutable and stay untouched.
- Commit as `docs: correct hook exit-code semantics and document guard OOM crash` (one logical change). No push — commit only.

### Out of scope

- Any pk code change: nothing to fix; no pk code executed. Buffer right-sizing/pooling would not have altered the outcome and isn't a Go idiom for millisecond-lived processes (GC owns deallocation).
- README Windows-memory caveat: not added unless requested; the error-reference entry carries the information.

## Verification

- `make test` and `make lint` before and after (docs-only change; both should stay green).
- Render-check the two edited docs; confirm the error-reference entry matches the existing section format (compare with the `pk changelog` section).
- No smoke tier needed: no runtime behavior changes (docs only).
