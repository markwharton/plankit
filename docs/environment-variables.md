# Environment variables

## Set by pk

- **VERSION**: the version being released, without the `v` (`0.8.1`). Set for every hook; see the [hook timeline](pk-json.md#hook-timeline).
- **TAG**: the release tag, with the `v` (`v0.8.1`). Set for `preRelease` and `prePush`. During `preRelease` the tag ref does not exist yet.

pk expands `$VERSION`, `${VERSION}`, `$TAG` and `${TAG}` in the hook line before the shell runs it, so one line works on macOS, Linux and Windows. Shell-specific forms such as `${VAR#pattern}` are not expanded. Hooks inherit the parent environment.

## Read by pk

- **CLAUDE_PROJECT_DIR**: set by Claude Code; the project root. `pk guard`, `pk preserve` and `pk protect` use it, and fall back to the `cwd` field of the hook payload.

## Files

- **`<user cache dir>/plankit/version-check.json`**: `pk version`'s update check, refreshed daily. The directory is `os.UserCacheDir()`: `~/Library/Caches` on macOS, `~/.cache` on Linux, `%LocalAppData%` on Windows.
- **`~/.claude/plans/`**: where Claude Code writes plan files. `pk preserve` reads the approved plan's path from the hook payload or the pointer below; it never scans this directory.
- **`.git/pk-pending-plan`**: the absolute path of the approved plan, written by the `pk preserve` hook in `manual` mode and read by the next `pk preserve` with no payload (the `/preserve` skill). Deleted after a successful preservation. It lives under `.git/`, which is never tracked, so it needs no `.gitignore` entry.
