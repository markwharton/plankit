# Review fixes, phase 3: gates and supply chain

## Context

A code review of the repository at v1.8.3 is being fixed in phases.
Phases 1 and 2 fixed the defects that were reproduced by running a
build. Phase 3 closes the gaps around the code: a test suite that no
gate runs, third-party code fetched at a floating version, secrets
visible to steps that do not need them, a dependency nothing watches,
and one path check that can be walked around.

1. `make test` vets `tools/docgen` and never runs its tests. CI runs
   `make test`, so docgen's four test files run nowhere automated,
   while `CONTRIBUTING.md` and `.claude/CLAUDE.md` say `make test`
   covers both modules.
2. The site deploy runs `npx --yes wrangler@3`, and the site's layout
   imports `mermaid@11` from a CDN. Each follows whatever is published
   next under that major version. GitHub Actions in this repository are
   already pinned to commit SHAs for the same reason.
3. `TAP_DISPATCH_TOKEN` is set for the whole release job and the
   Cloudflare token for the whole site job, so every step sees them,
   including the ones that build with a third-party module.
4. Dependabot watches GitHub Actions only. `tools/docgen` depends on
   goldmark, and nothing reports a new version of it.
5. `pk preserve --plan` accepts a path under the plans directory that
   climbs back out of it with `..`.

The outcome: five local commits on `develop`, the suite green before
and after each. Nothing is pushed and no release is cut.

## Fix 1: `make test` runs the docgen tests

`Makefile`, the `test` target: after vetting `tools/docgen`, run its
tests too. CI, which runs `make test`, gains them with no change of
its own. The docgen tests build everything they need in temporary
directories, with their own git identity, so they do not depend on the
checkout's history or tags.

`CONTRIBUTING.md` and `.claude/CLAUDE.md` already describe `make test`
this way; the change makes them true.

Commit: `build: make test runs the docgen tests`

## Fix 2: a plan path cannot climb out of the plans directory

`internal/preserve/preserve.go`, `matchPlanPath`: the matched path is
cleaned, and a path that no longer sits under the plans directory once
cleaned matches nothing. A path that wanders and comes back
(`plans/sub/../widget.md`) is the cleaned path. Both readers of a plan
path go through this function, the hook's payload and `--plan`, so
both get the check.

Tests in `internal/preserve/preserve_test.go`: `--plan` with
`<plans>/../outside.md`, for a file that exists there, is refused as
"not a Claude Code plan" and commits nothing; a path that returns
inside the directory is preserved.

Commit: `fix(preserve): a plan path that climbs out of the plans directory is refused`

## Fix 3: wrangler and mermaid run at exact versions

- `.github/workflows/site.yml`: `wrangler@3` becomes `wrangler@3.114.17`,
  the newest 3.x the registry lists. The 3.x line is the registry's
  `legacy` tag; moving to 4 is a change of its own and is not made
  here.
- `site/layout.html`: the import names `mermaid@11.17.2`, the newest
  11.x. Subresource integrity is not added: mermaid's entry module
  loads its own chunks, which an integrity value on the entry would
  not cover.
- `tools/docgen/site_test.go`, `TestBuildSite`: one more assertion on
  the page that carries a diagram, that the import names a full
  `major.minor.patch` version, so the layout cannot drift back to a
  floating one.
- `.github/workflows/ci.yml`: the Claude Code CLI install stays on the
  latest version, with a comment giving the reason: the newest CLI is
  what shows a change in plugin validation the day it lands, and that
  job has a read-only token and no secrets.

Dependabot cannot see a version written inside a `run` step or an
HTML file, so these two are bumped by hand.

Commit: `security: wrangler and mermaid run at exact versions`

## Fix 4: a token reaches only the step that uses it

A job-level `env` hands a secret to every step. The job-level entries
exist only so a step's `if` can test whether the secret is set; the
step's own script can make that test.

- `.github/workflows/release.yml`: `TAP_DISPATCH_TOKEN` moves from the
  job's `env` to the "Notify the Homebrew tap" step. The step's `if`
  goes; its script exits 0 with a line saying so when the token is
  empty. `continue-on-error` stays.
- `.github/workflows/site.yml`: `CLOUDFLARE_API_TOKEN` and
  `CLOUDFLARE_ACCOUNT_ID` move from the job's `env` to the deploy
  step. The "Deploy" and "Skipped deploy" steps become one step whose
  script prints the skipped line and exits 0 when the token is empty,
  and deploys otherwise.

The build steps (`make dist`, `make site`), which run docgen and its
third-party module, no longer see either token.

Commit: `security(ci): the deploy and tap tokens reach only the step that uses them`

## Fix 5: Dependabot watches the docgen module

`.github/dependabot.yml`: a second entry, `gomod` in `/tools/docgen`,
with the schedule, target branch, and commit prefix of the existing
GitHub Actions entry. The main module has no dependencies to watch.

Commit: `ci: Dependabot watches the docgen module`

## Not changed, and why

- `.pk.json`'s `release.hooks.preRelease` still runs the main module's
  tests only (`go test -race ./...`). It is this repository's release
  policy, and CI gates the docgen tests on every push once fix 1 lands.
- The Claude Code CLI in CI is left on latest, as fix 3 says.

## Where each fact lives after this phase

- The wrangler version is written once, in `site.yml`.
- The mermaid version is written once, in `site/layout.html`, and a
  test holds it to a full version.
- Each secret is named in the one step that uses it.
- "Inside the plans directory" is decided in `matchPlanPath` for both
  the hook and `--plan`.

## Verification

- `make test` before the first change and after each of the five, with
  the output shown. After fix 1 the output lists the docgen package.
- The case that must fail, for fix 1: `make test` run with a failing
  docgen test supplied through a build overlay, so the tree is not
  touched, exits non-zero.
- The preserve test is run and seen to fail before fix 2, then pass;
  the same for the new assertion in `TestBuildSite` before fix 3.
- Each workflow file and `dependabot.yml` is parsed as YAML after its
  change, and the parsed step is printed to show the `env` sits on the
  step and not the job.
- Each step script that now tests for its token is run locally with
  the token unset: it prints its skipped line and exits 0. The path
  with a token set runs only in GitHub Actions; it is first exercised
  by the next release, or by dispatching the site workflow by hand.
- `make site-preview` builds, and the how-it-works page carries the
  exact mermaid version.
- `gofmt -l cmd internal tools` prints nothing; `make docs` leaves no
  diff.

## Later phases, each with its own plan

- Phase 4, duplication and dead code: the IR types, command-page
  detection, breaking-marker detection, release pre-flight checks,
  config loading, current-branch lookup, hook call sites, pin, docgen's
  semver and URL copies, the unused message catalogue, late policy
  validation, swallowed errors, leftovers. Decisions there: delete or
  adopt the catalogue, and the exit code for an unreachable origin.
- Phase 5, tokens: named mode values, the help style table, narration
  through `msg`, guard's reason prefixes, the stylesheet. Decision
  there: which sizes and radii the stylesheet keeps.
