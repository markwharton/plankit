# preserve.dir: the plans directory is a dial, docs/plans by default

## Context

The plans directory is hard-coded as `docs/plans` in `internal/paths`, and preserve, protect, brief, and status read it there. A repository with a different documentation convention (a `documentation/` tree, say) cannot put its records where its other documents live. The pk init plan of 2026-09-20 recorded this as a future dial named `preserve.dir` with default `docs/plans`; this plan builds it.

Outcome: `.pk.json` gains `preserve.dir`, a directory relative to the repository root. Every reader resolves it through the config, the brief tells each session the configured directory, protect guards it, status reports it, and the pages say "the plans directory" with the default stated once, in the generated Settings block. A repository with no `dir` key behaves exactly as today.

Suite at start of this session: green.

## Change 1: the dial (`feat(preserve)`)

**`internal/config/config.go`**

- `DefaultPreserveDir = "docs/plans"` beside `DefaultPreserveMode`, with the `// preserve.dir` tag the others carry.
- `PreserveConfig` gains `Dir string \`json:"dir,omitempty"\`` above `Mode` (key order; `TestWrittenConfigIsSorted` holds it).
- `ResolvedDir()` applies the default. `DirPath(root string) string` returns `filepath.Join(root, filepath.FromSlash(ResolvedDir()))`, the one place the filesystem path is built, as `config.Path` is for the policy file. The forward-slash relative form for `git add` and messages is `path.Join(cfg.Preserve.ResolvedDir(), filename)` at the call site; no third helper.
- `Default()` writes `Dir: DefaultPreserveDir` explicitly, as it writes the modes, so the file pk init writes reads whole.
- `Validate()` adds one rule, after the enum loop: a non-empty `preserve.dir` must be a clean relative path with forward slashes that stays inside the repository. Refused by name, in one message: absolute paths, a backslash, `.`, `..` or a `../` prefix, a value that `path.Clean` would change (a trailing slash, a `./`, a doubled slash). Windows drive letters fall under "absolute" via `filepath.IsAbs` and a leading-letter-colon check.

**`internal/config/settings.go`**: the row, sorted before `preserve.mode`:

```
{Key: "preserve.dir", Kind: "a directory, relative to the repository root", Shape: `"<dir>"`, Default: "`docs/plans`",
    Doc: "Where preserved plans are written and kept immutable; forward slashes, inside the repository."},
```

docgen then emits it into the preserve page's Settings block and into the JSON Schema as a string property with that description. Nothing else for the schema.

**`internal/paths`**: deleted. Its symbols move to the config methods above; `PlansRel` has no successor because the default is `DefaultPreserveDir`. CHANGELOG history is the only remaining mention and stays.

**Readers**

- `internal/preserve/preserve.go`: `preserve()` takes the `PreserveConfig` and uses `DirPath` and `path.Join(ResolvedDir(), filename)`. The explicit paths (typed, `--plan`) now load the config too: `ErrNotConfigured` returns `cli.Statef` with the hint "run pk init to write .pk.json", exit 2; another load error returns `cli.Statef` with the message. Today an explicit run in an unconfigured repository commits to `docs/plans`; that ends, and it is the only behaviour change. The hook path is unchanged. The manual-mode `additionalContext` names the configured directory instead of `docs/plans/`. `Summary` becomes "Hook: preserve the approved plan as a record and commit it". The package comment stops naming `docs/plans/`.
- `internal/protect/protect.go`: `config.Load` already runs; keep the config and pass `cfg.Preserve.DirPath(root)` into `isUnderPlansDir`. The deny reason names the configured directory: `"%s/ files are immutable historical records. ..."`. `Summary` becomes "Hook: keep preserved plans immutable". Package comment likewise.
- `internal/brief/brief.go`: the three preserve sentences start with `cfg.Preserve.ResolvedDir() + "/"`.
- `internal/repo/repo.go`: drop the `PlansDir` constant. `runStatus` reports `line("protect", cfg.Preserve.ResolvedDir()+"/ immutable")` and counts plans in `cfg.Preserve.DirPath(root)` when configured; unconfigured reports zero plans (status already exits 2 there). The init comment says "the plans directory".

**Tests**

- `internal/config/config_test.go`: `TestPreserveDirIsValidated`: `documentation/plans` loads and resolves; each of `/abs`, `C:/abs`, `../out`, `docs/../x`, `docs/plans/`, `./docs`, `.`, `docs\plans` fails with a message containing `preserve.dir`. `TestAbsentModesResolveToDefaults` also checks `ResolvedDir() == "docs/plans"` and `DirPath`.
- `internal/preserve/preserve_test.go`: `scratch` takes the config it writes as today; add `TestConfiguredDirIsUsed` (dir `documentation/plans`: the file lands there, the message and commit path say so, `docs/` is not created) and `TestTypedRunRefusesAnUnconfiguredRepository` (remove `.pk.json`, typed run and `--plan` both exit 2 naming `.pk.json`). Replace `paths.Plans(dir)` with `config.Default("main").Preserve.DirPath(dir)` or a small helper.
- `internal/protect/protect_test.go`: `TestDeniesWritesUnderConfiguredDir`: with dir `documentation/plans`, a write there is denied and names that directory, and a write under `docs/plans/` is allowed.
- `internal/brief/brief_test.go`: a case with `Dir = "documentation/plans"` wants "documentation/plans/ is immutable".
- `internal/repo/repo_test.go`: the status test sets `Dir`, seeds a plan there, and expects the count and the protect line; the `PlansDir` references become the config default.

**Generated**: `make docs` rewrites the preserve page's Settings block and `internal/help/data`; `make site` regenerates the schema under `site/dist`, which is not committed. Committed with the change.

## Change 2: the pages (`docs: pages say the plans directory`)

The literal `docs/plans` leaves authored prose entirely. The words are "the plans directory" everywhere; the default is stated by the generated Settings block on the preserve page, which owns the dial, and by the brief in each repository. A page that needs to say where the directory is named says `preserve.dir` in `.pk.json`, once.

- `skills/preserve/SKILL.md`: description "Keep each approved plan as a record in the plans directory, on approval or on request"; the opening sentence names `preserve.dir` once as the directory's home.
- `skills/protect/SKILL.md`: description "Why preserved plans are immutable and how pk protect enforces it"; body says "under the plans directory, `preserve.dir` in `.pk.json`" and "the sequence in the plans directory".
- `skills/overview/SKILL.md`: the Model paragraph, the footprint paragraph, and the migration step say "the plans directory".
- `skills/init/SKILL.md`: "The plans directory is not created here."
- `README.md`: the Install paragraph and the loop paragraph say "the plans directory"; Install points at `pk help preserve` for the key.
- `docs/architecture.md`: the record is "the plans directory, `preserve.dir`"; the Plans section says "the plans directory".
- `docs/design.md`: the policy-file bullet lists "preserve's mode and directory". The protect debugging example is a payload run against a scratch repository with the default policy, so its `docs/plans/` path is example data, like `example.md`, not a statement of the default; it stays.
- `docs/notes/v1.0.0.md`: history, unchanged.
- A grep for `docs/plans` over `skills/`, `README.md`, `docs/architecture.md` afterwards shows no hits; the description one-liners are reread against "purpose, not features".

## Commits

Two commits on `develop`, no breaking marker, no Co-Authored-By trailer, no push:

1. `feat(preserve): preserve.dir names the plans directory, docs/plans by default`
2. `docs: pages say the plans directory`

`make test` before and after each, reading the fail count. The feature makes the next release a minor, which needs a note under `docs/notes/` before `pk changelog`; that is a release-time task, not part of this change.

## Verification

- `make test` green at each step; `gofmt -l` empty; `make docs` leaves the tree clean after commit; `claude plugin validate . --strict` passes.
- `make build`, then in a scratch repository under the scratchpad: `./pk init`, edit `.pk.json` to `"dir": "documentation/plans"`, then:
  - `./pk brief < /dev/null` says `documentation/plans/ is immutable`.
  - `./pk preserve --plan ~/.claude/plans/wise-crafting-dijkstra.md` commits under `documentation/plans/`; `./pk status` shows the protect line and one plan.
  - Pipe a protect payload for `documentation/plans/x.md`: deny naming that directory; for `docs/plans/x.md`: silence.
  - Must fail: set `"dir": "../elsewhere"`; `./pk status` exits 2 with a message naming `preserve.dir`; `./pk preserve` in a repository without `.pk.json` exits 2 with the init hint.
- `./pk help preserve` shows the `preserve.dir` row with its default.
