# plankit

Plan-driven development for Claude Code: guarded branches, immutable
plan records, and conventional-commit releases, shipped as a plugin.

The skills are the documentation. The same pages load as /plankit:
shortcuts in Claude Code and render in the terminal as `pk help`, so
one source describes every command in both places. The pk binary is
the deterministic kernel behind them: hooks that guard protected
branches and preserve plans, and the changelog and release machinery.

## Install

```
/plugin marketplace add markwharton/plankit
/plugin install plankit
```

Then, in a repository you want plankit to manage:

```
pk init
```

A configured repository carries exactly two things: `.pk.json`
(committed repo policy) and `docs/plans/` (the preserved plans).
No `.pk.json` means off: every hook exits immediately.

## The loop

Plan in Claude Code; on plan approval the preserve hook commits the
plan into `docs/plans/`, where the protect hook keeps it immutable.
The guard hook blocks git mutations on protected branches, so work
lands on your development branch. When it is time to release:

```
pk changelog    # generate CHANGELOG.md, commit with a Release-Tag trailer
pk release      # tag from the trailer, merge or trunk flow, push
```

`pk help` lists every topic; `pk status` reports the repository's
configuration and state.

## Building from source

```
make build      # docgen + go build -> ./pk
make test
make bin-local  # build the platform binary behind the bin/pk shim
```

The main module is standard-library Go; the markdown compiler
(goldmark) lives in `tools/docgen` and runs at build time only.

plankit v1 and its documentation live in git history before the
plugin-first rewrite.
