# Three Updates: Cross-platform hooks, setup mode persistence, Windows install note

## Context

Three issues surfaced from user feedback:

1. **Cross-platform hook variable expansion**: Lifecycle hooks in `.pk.json` use `$VERSION` syntax. On Unix, `sh -c` expands this from the environment. On Windows, `cmd /c` needs `%VERSION%`. A single `.pk.json` can't use both, but the same project may be cloned on any platform. Additionally, the docs show `${VERSION#v}` (bash parameter expansion) which is unnecessary because VERSION is already set without the `v` prefix (confirmed: `changelog.go:251` does `strings.TrimPrefix(nextTag, "v")`).

2. **`pk setup` silently loses modes**: Running `pk setup --guard ask --preserve auto` then later `pk setup` (e.g., after upgrading pk) resets to defaults (`block`/`manual`). Flags have hardcoded defaults with no detection of existing configuration.

3. **Windows binary download confusion**: User downloaded the release binary into Downloads and double-clicked it expecting an app or installer. Since `pk` is a CLI tool, the transient console window opened, printed help to stderr, and closed immediately. The README needs to clarify that `pk` is a command-line tool to be run from a terminal.

## Plan

### 1. Cross-platform variable expansion in RunScript

**Problem:** `.pk.json` hook commands use `$VERSION` which only works on Unix. Windows `cmd /c` uses `%VERSION%`. A committed `.pk.json` must work on all platforms without per-OS hacks.

**Solution:** Pre-expand pk-provided variables in `RunScript` using Go's `os.Expand` before passing to the shell. This makes `$VERSION` syntax work identically on all platforms because Go does the expansion, not the shell.

Verified with `os.Expand` behavior tests:
- `$VERSION` -> value (primary use case)
- `${VERSION}` -> value (braced form)
- `$HOME` (not in env map) -> `$HOME` (preserved for shell expansion on Unix)
- `$VERSION_EXTRA` -> `$VERSION_EXTRA` (word-boundary correct, different var name)

**File: `internal/hooks/script.go`**

Add `os.Expand` pre-expansion before passing the command to the shell:

```go
func RunScript(command string, env map[string]string) error {
    // Pre-expand pk-provided variables so $VAR syntax works on all
    // platforms. sh -c expands $VAR natively; cmd /c does not. By
    // expanding before the shell sees the command, hook authors write
    // $VERSION once and it works everywhere.
    if len(env) > 0 {
        command = os.Expand(command, func(key string) string {
            if v, ok := env[key]; ok {
                return v
            }
            return "$" + key
        })
    }

    var cmd *exec.Cmd
    // ... rest unchanged
}
```

Need to add `"os"` to imports (currently not imported; `os.Stderr` is used via the `os` package but let me verify).

**File: `internal/hooks/script_test.go`**

Add tests for the pre-expansion behavior:
- `env var expansion in command`: verify `$VERSION` in the command string is expanded (e.g., `echo $VERSION` produces the value)
- `braced expansion in command`: verify `${VERSION}` works
- `unknown vars preserved`: verify `$HOME` passes through to the shell
- `no expansion without env`: verify nil env map leaves command unchanged

**File: `docs/pk-changelog.md`**

The "Single tag, many files" example (lines 141-155) shows a Node.js monorepo using lifecycle hooks. Fix factual errors in that example:
- Line 149: change `${VERSION#v}` to `$VERSION`. The `v`-stripping is unnecessary since VERSION is already set without the `v` prefix. This also removes the bash-specific syntax that won't survive `os.Expand` pre-expansion.
- Line 155: fix the description. Currently says VERSION is `v0.11.0`. Change to explain VERSION is `0.11.0` (without `v` prefix). Remove the `${VERSION#v}` explanation.
- After line 155: add a note that `$VERSION` works on all platforms because pk expands it before passing to the shell. Bash-specific parameter expansion like `${VAR#pattern}` is not supported cross-platform.

### 2. Preserve guard/preserve modes on re-run

**Problem:** `cmd/pk/main.go:148-149` has hardcoded defaults. Re-running `pk setup` without flags silently overwrites previous configuration.

**Solution:** Use `flag.FlagSet.Visit()` to detect which flags were explicitly set. If not explicit, read existing settings.json and infer current modes from hook commands. Fall back to hardcoded defaults only on first install.

**File: `internal/setup/setup.go`**

**Step 1: Define command constants** near the existing `buildHookConfig` function (around line 219). These replace the string literals in both setup and status:

```go
const (
    GuardBlockCommand     = "pk guard"
    GuardAskCommand       = "pk guard --ask"
    PreserveAutoCommand   = "pk preserve"
    PreserveManualCommand = "pk preserve --notify"
)
```

**Step 2: Update `buildHookConfig`** to use the constants instead of string literals.

**Step 3: Add `InferModesFromCommands`** that takes a flat list of command strings and returns modes. This is the shared logic, no magic strings:

```go
func InferModesFromCommands(commands []string) (guard, preserve string) {
    for _, cmd := range commands {
        switch cmd {
        case GuardBlockCommand:
            guard = "block"
        case GuardAskCommand:
            guard = "ask"
        case PreserveManualCommand:
            preserve = "manual"
        case PreserveAutoCommand:
            preserve = "auto"
        }
    }
    return guard, preserve
}
```

All exact matches, no `HasPrefix`.

**Step 4: Add `InferModes`** that extracts commands from parsed settings and delegates:

```go
func InferModes(settings *OrderedObject) (guard, preserve string) {
    hooksRaw, ok := settings.Get("hooks")
    if !ok {
        return "", ""
    }
    var hooks HooksConfig
    if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
        return "", ""
    }
    var commands []string
    for _, entries := range [][]HookEntry{hooks.PreToolUse, hooks.PostToolUse, hooks.SessionStart} {
        for _, entry := range entries {
            for _, h := range entry.Hooks {
                commands = append(commands, HookCommand(h))
            }
        }
    }
    return InferModesFromCommands(commands)
}
```

**File: `internal/status/status.go`**

Refactor `inferModes` (lines 326-345) to use setup's constants and exact match. Status already imports setup. Two options:

Delegate to `setup.InferModesFromCommands`, eliminating the duplicate switch:
```go
func inferModes(hooks []hookSummary) (string, string) {
    var commands []string
    for _, h := range hooks {
        commands = append(commands, h.commands...)
    }
    return setup.InferModesFromCommands(commands)
}
```

**File: `cmd/pk/main.go` `runSetup()` (lines 145-213)**

After `fs.Parse(args)`, add detection and inference:
1. Move dir resolution (lines 157-164) before the new block
2. Use `fs.Visit()` to detect which flags were explicitly set
3. If not explicit, read settings.json and call `setup.InferModes`
4. Add `"path/filepath"` to imports

```go
fs.Parse(args)

// Resolve project directory early — needed for both inference and Run.
dir := *projectDir
if dir == "." {
    var err error
    dir, err = os.Getwd()
    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }
}

// Detect explicit flags to preserve existing modes on re-run.
guardExplicit, preserveExplicit := false, false
fs.Visit(func(f *flag.Flag) {
    switch f.Name {
    case "guard":
        guardExplicit = true
    case "preserve":
        preserveExplicit = true
    }
})
if !guardExplicit || !preserveExplicit {
    settingsFile := filepath.Join(dir, ".claude", "settings.json")
    if data, err := os.ReadFile(settingsFile); err == nil {
        if parsed, err := setup.ParseOrderedObject(data); err == nil {
            g, p := setup.InferModes(parsed)
            if !guardExplicit && g != "" {
                *guardMode = g
            }
            if !preserveExplicit && p != "" {
                *preserveMode = p
            }
        }
    }
}
```

Remove the duplicate dir resolution block that currently exists later in the function.

**File: `internal/setup/setup_test.go`**

Add tests for both `InferModesFromCommands` and `InferModes`:
- `TestInferModesFromCommands_blockAndManual`: verify ["pk guard", "pk preserve --notify"] -> block, manual
- `TestInferModesFromCommands_askAndAuto`: verify ["pk guard --ask", "pk preserve"] -> ask, auto
- `TestInferModesFromCommands_empty`: verify [] -> "", ""
- `TestInferModes_blockAndManual`: build hooks via `buildHookConfig`, merge into settings, verify round-trip
- `TestInferModes_askAndAuto`: same with auto/ask modes
- `TestInferModes_noHooks`: empty settings, verify empty strings
- `TestInferModes_corruptHooks`: malformed JSON, verify graceful empty return
- `TestInferModes_userHooksOnly`: settings with only non-pk hooks, verify empty strings

**File: `docs/pk-setup.md` line 84**

Change "Re-run setup anytime to switch modes." to:
"Re-running `pk setup` preserves the existing mode configuration. Pass `--guard` or `--preserve` explicitly to change modes."

**File: `README.md` line 55**

Change "Re-run setup anytime to switch." to:
"Re-run setup to upgrade managed files. Pass `--guard` or `--preserve` explicitly to change modes."

### 3. Windows install note in README

**Problem:** User downloaded the release binary into their Downloads directory and double-clicked it, expecting an installer or GUI app. Since `pk` is a CLI tool, the console window flashed and closed immediately.

**File: `README.md` line 33**

After "Or download a binary from the releases page (no Go required).", add a note clarifying that `pk` is a command-line tool. The user should open a terminal (PowerShell, Command Prompt, or Git Bash), navigate to the binary's location or add it to PATH, and run `pk` from there.

## Verification

### Automated
- `make test` (includes new InferModes and script expansion tests)
- `make lint`
- `make build`

### Smoke: variable expansion
1. Build: `make build`
2. Create a test `.pk.json` with a `preCommit` hook using `$VERSION`
3. Run `dist/pk changelog --dry-run` to verify the hook would receive the expanded value
4. Verify the existing `"pk pin --file .claude/install-pk.sh $VERSION"` hook still works

### Smoke: mode persistence
1. In a temp project: `dist/pk setup --guard ask --preserve auto`
2. Verify: `dist/pk status` shows `guard=ask, preserve=auto`
3. Re-run: `dist/pk setup` (no flags)
4. Verify: `dist/pk status` still shows `guard=ask, preserve=auto`
5. Override: `dist/pk setup --guard block`
6. Verify: `dist/pk status` shows `guard=block, preserve=auto`

### Smoke: docs
- Verify pk-changelog.md VERSION description matches code behavior (`0.11.0`, not `v0.11.0`)

## Commit sequence

1. `fix: pre-expand hook variables for cross-platform compatibility` (script.go, script_test.go, pk-changelog.md)
2. `feat(setup): preserve guard and preserve modes on re-run` (setup.go, main.go, setup_test.go, pk-setup.md, README.md)
3. `docs: add Windows install note to README` (README.md)

## Critical files
- `internal/hooks/script.go` — add os.Expand pre-expansion
- `internal/hooks/script_test.go` — add expansion tests
- `internal/setup/setup.go` — add command constants, InferModesFromCommands, InferModes; update buildHookConfig to use constants
- `internal/setup/setup_test.go` — add InferModes/InferModesFromCommands tests
- `internal/status/status.go` — refactor inferModes to use setup.InferModesFromCommands (eliminates duplicate switch)
- `cmd/pk/main.go` — add Visit-based detection in runSetup, add path/filepath import
- `docs/pk-changelog.md` — fix VERSION docs, add cross-platform note
- `docs/pk-setup.md` — update re-run behavior description
- `README.md` — mode persistence wording, Windows install note
