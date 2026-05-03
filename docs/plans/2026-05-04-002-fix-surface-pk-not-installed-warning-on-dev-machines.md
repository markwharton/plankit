# Fix: surface "pk not installed" warning on dev machines

## Context

The `install-pk.sh` SessionStart hook warns when a developer clones a pk-configured repo without pk installed. But Claude Code suppresses output from SessionStart hooks that exit 0, so the warning is never seen. The session starts silently with broken hooks.

Validated on plankit.com: changing `exit 0` to `exit 1` causes Claude Code to surface the message as a non-blocking error. The session still starts normally, but the developer sees the warning.

## Files to Modify

### 1. `internal/setup/template/install-pk.sh` (line 37-39)

Two changes in the local-machine-without-pk code path:

1. **`exit 0` to `exit 1`** (line 39) so Claude Code surfaces the stderr output as a non-blocking error instead of swallowing it
2. **Drop the `⚠` from the echo** (line 37) since Claude Code already labels exit 1 as a hook error; the emoji was compensating for the silent exit

Current (lines 34-40):
```bash
# On local machines (no CLAUDE_ENV_FILE), pk isn't installed but we can't
# download it either. Warn so the developer knows hooks won't run.
if [ -z "${CLAUDE_ENV_FILE:-}" ]; then
  echo "⚠ pk is not installed. Hooks (guard, preserve, protect) will not run." >&2
  echo "Install: go install github.com/markwharton/plankit/cmd/pk@latest" >&2
  exit 0
fi
```

### 2. `.claude/install-pk.sh` (same code path)

Apply the same two changes to the local copy so it matches the template.

## Verification

1. `make build && make test && make lint` pass
2. Run `pk setup` in plankit.com to regenerate `install-pk.sh` and confirm the generated file has `exit 1` on that code path
