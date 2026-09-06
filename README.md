# plankit

Plan-driven development for Claude Code: guarded branches, immutable
plan records, and conventional-commit releases, shipped as a plugin.
plankit is the plugin; pk is the command it installs.

The skills are the documentation. The same pages load as `/plankit:`
shortcuts in Claude Code and render in a terminal as `pk help`. The pk
binary holds every decision: the hooks that guard branches and
preserve plans, and the changelog and release machinery. Uninstall the
plugin and pk still does everything from a terminal.

## Install

```
/plugin marketplace add https://plankit.com/marketplace.json
/plugin install plankit
```

The same marketplace file is attached to every GitHub release, so
`https://github.com/markwharton/plankit/releases/latest/download/marketplace.json`
works as the source too.

Then, in a repository you want plankit to manage:

```
pk init
```

A configured repository carries `.pk.json`, the committed policy.
`docs/plans/` appears when the first plan is preserved. No `.pk.json`
means off: every hook exits immediately.

## The loop

Plan in Claude Code; on plan approval the preserve hook commits the
plan into `docs/plans/`, where the protect hook keeps it immutable.
The guard hook blocks git mutations on protected branches, so work
lands on your development branch. When it is time to release:

```
pk changelog    # generate CHANGELOG.md, commit with a Release-Tag trailer
pk release      # tag from the trailer, merge or trunk flow, push
```

or as one command: `pk ship` runs both, resuming at release if a
Release-Tag commit is already pending.

`pk help` lists every topic; `pk status` reports the repository's
configuration and state.

## pk outside Claude Code

The guard, changelog, release, and pin commands are plain git
discipline and work in any terminal. With Go installed:

```
go install github.com/markwharton/plankit/cmd/pk@latest
```

On macOS or Linux with Homebrew:

```
brew tap markwharton/plankit
brew install plankit
```

Without Go or Homebrew, download your platform's binary from the latest GitHub
release (six are attached: `pk-darwin-arm64`, `pk-darwin-amd64`,
`pk-linux-amd64`, `pk-linux-arm64`, `pk-windows-amd64.exe`,
`pk-windows-arm64.exe`). On macOS or Linux:

```
chmod +x pk-<os>-<arch>
mv pk-<os>-<arch> ~/.local/bin/pk    # any PATH directory
```

On Windows, rename `pk-windows-<arch>.exe` to `pk.exe` and place it
in a directory on your PATH. Verify with `pk version`, then `pk init`
in the repository you want plankit to manage.

## Building from source

```
make build      # docgen + go build -> ./pk
make test
make bin-local  # build the platform binary behind the bin/pk shim
```

The main module is standard-library Go; the markdown compiler
(goldmark) lives in `tools/docgen` and runs at build time only.

The pre-plugin plankit (v0.x) and its documentation live in git
history before the plugin-first rewrite.
