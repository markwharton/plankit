# Versioning

How pk manages version numbers across a release.

## Source of truth

Git tags are the single source of truth for version. `pk changelog` reads the latest tag, computes the next version from conventional commits, and writes that version into whatever files need it. Nothing else determines the version — not a constant in source, not a field in package.json, not a variable in a script. Those are all downstream targets that receive the version at release time.

## Flowing the version into artifacts

Several mechanisms exist for getting the tag-derived version into files. Choose based on your project's build model.

### versionFiles — JSON packages

For projects with a root-level `"version"` field in a JSON file (Node.js, Deno, VS Code extensions).

`pk changelog` updates these files directly using structural JSON parsing — it finds the root `"version"` key and splices in the new value, preserving formatting, key order, and indentation.

```json
{
  "changelog": {
    "versionFiles": [
      {"path": "package.json", "type": "json"},
      {"path": "package-lock.json", "type": "json"}
    ]
  }
}
```

**When to use:** The version lives in a JSON file that tools read at runtime or publish time (`npm publish` reads `package.json`). No build step needed to surface the version.

### pk pin --name — source constants

For projects where the version is a constant or variable in source code (Go, Python, Rust, TOML).

A `preCommit` hook calls `pk pin` to find the named identifier and replace its quoted value. The v-prefix is inferred from whatever's already in the file.

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file cmd/myapp/main.go --name version $VERSION"
    }
  }
}
```

Examples across languages:

| Language | Source line | Hook |
|----------|-----------|------|
| Go | `const version = "0.1.0"` | `pk pin --file cmd/myapp/main.go --name version $VERSION` |
| Python | `__version__ = "0.1.0"` | `pk pin --file mypackage/__init__.py --name __version__ $VERSION` |
| Rust | `version = "0.1.0"` | `pk pin --file Cargo.toml --name version $VERSION` |

`pk pin --name` matches the first occurrence in the file. For TOML files like `Cargo.toml`, this works when `[package]` appears before `[dependencies]`, which is the conventional layout. If the file structure puts a different `version` key first, the match will be wrong. Structural TOML support via `versionFiles` would eliminate this limitation.

**When to use:** The binary or package reads its version from a source constant at runtime, and the build is a simple `go build` / `pip install` / `cargo build` with no ldflags or injection step. The version lives in the source file and `pk pin` keeps it in sync with the tag.

### ldflags — build-time injection

For projects where the version is injected at build time and never appears in source. Common in Go projects with cross-compilation or CI-driven releases.

The source declares a variable with a default:

```go
var version = "dev"
```

The Makefile or CI script injects the real value:

```makefile
VERSION := $(shell git describe --tags --abbrev=0 | sed 's/^v//')
go build -ldflags "-X main.version=$(VERSION)" ./cmd/myapp
```

The `preCommit` hook pins a shell-variable script (for cloud sandbox bootstrap) rather than a source constant:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file .claude/install-pk.sh $VERSION"
    }
  }
}
```

**When to use:** The project cross-compiles for multiple platforms, uses CI to build release binaries, or has a Makefile that already derives the version from git. The version is never committed to source — it's computed from the tag at build time. This avoids a source-level version that could drift from the tag.

### Single tag, many files

Monorepos with a unified-version policy release every package at the same version, governed by one tag. The `preCommit` hook fans the tag-derived version out to every file that needs it:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pnpm -r exec npm version $VERSION --no-git-tag-version && git add 'packages/**/package.json' pnpm-lock.yaml"
    }
  }
}
```

This works when the ecosystem handles cross-package references natively (pnpm's `workspace:*` protocol never resolves a pinned version, so bumping each package independently is safe). Not every monorepo tool handles this cleanly — `npm version --workspaces`, for example, bumps each package's own version but doesn't rewrite cross-refs, then fails trying to resolve the old version from the registry. When the ecosystem command doesn't cover cross-refs, use a [hook script](#hook-scripts--custom-version-sync) instead.

Strategies compose freely. A project might use `versionFiles` for package.json, `pk pin --name` for a Python constant, and `pk pin` (without `--name`) for a bootstrap script, all in the same release.

### Hook scripts — custom version sync

For projects where the framework has no built-in version propagation command. You write a script that receives `$VERSION`, transforms it to the framework's format, updates every file that needs it, and stages the results.

```json
{
  "changelog": {
    "versionFiles": [
      {"path": "package.json", "type": "json"}
    ],
    "hooks": {
      "postVersion": "node scripts/sync-spfx-version.js $VERSION"
    }
  }
}
```

The script handles whatever the framework requires: SPFx uses 4-part versions (`X.Y.Z.0`) spread across `package-solution.json` and multiple `*.manifest.json` files. No CLI command exists to sync these, so the script receives the version, appends `.0`, writes every manifest, and stages the results.

**When to use:** The framework stores versions in a non-standard format or across multiple files with no ecosystem command to update them. `versionFiles` handles the root `package.json`; the hook script handles everything else.

## Choosing a strategy

| Project shape | Strategy | Why |
|--------------|----------|-----|
| npm package, VS Code extension | versionFiles | `npm publish` and `vsce` read `package.json` directly |
| Simple Go/Python/Rust CLI | pk pin --name | `go build ./cmd/...` is the build command; no Makefile, no ldflags |
| Cross-compiled Go binary with CI | ldflags | Version derived from tag at build time; no constant in source to drift |
| Monorepo with unified version | versionFiles + preCommit hook | Ecosystem provides propagation commands; verify cross-ref handling |
| SPFx, custom frameworks | versionFiles + hook script | No ecosystem command; script handles format conversion and file discovery |

## Multiple pins in one release

Chain commands in the `preCommit` hook:

```json
{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file .claude/install-pk.sh $VERSION && pk pin --file cmd/myapp/main.go --name version $VERSION"
    }
  }
}
```

All pinned files are staged by `git add -u` before the changelog commit. `pk changelog --undo` reverts everything in one step.

## Cross-platform hooks

`pk changelog` pre-expands `$VERSION` and `${VERSION}` in hook commands before passing them to the shell. This means the same hook syntax works on macOS, Linux, and Windows without platform-specific variable references (`%VERSION%` in cmd, `$env:VERSION` in PowerShell). The expansion happens inside pk, not in the shell.

This is also why `pk pin` exists rather than using `sed` — `sed -i` behaves differently on macOS vs Linux, and doesn't exist on Windows. `pk pin` is a cross-platform replacement that handles the quoting, prefix conventions, and file writing portably.

Bash-specific parameter expansion like `${VAR#pattern}` is not supported — only simple `$VAR` and `${VAR}` substitution.

## Related

- [pk changelog](pk-changelog.md) — release workflow, hooks, versionFiles configuration
- [pk pin](pk-pin.md) — command reference
