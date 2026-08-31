# Design

plankit's documentation convention, and the decisions that shape the repository. The local rule ([docs.md](../.claude/rules/docs.md)) governs sentences and loads in this repository's sessions; another repository adopts it by copying the file. This convention governs which files exist, and is plankit's own.

## The documentation convention

**The set is closed.** The root holds `README.md`, `LICENSE`, `CONTRIBUTING.md`, `CLAUDE.md`, `CHANGELOG.md` (generated, never edited), and GitHub's two remaining reserved names: `SECURITY.md`, only as GitHub means it, the vulnerability disclosure policy the Security tab links (plankit's names the reporting channel; the security model is [security.md](security.md)), and `CODE_OF_CONDUCT.md`, which does not exist today. `docs/` holds the suite, the references, `docs/plans/` (immutable records), and `docs/archive/`. A new topic joins an existing file; it never becomes a new one. A decision goes into this file; a reference fact goes into the reference that owns it.

The suite:

| Document | Holds |
|---|---|
| [architecture.md](architecture.md) | components, boundaries, why |
| [security.md](security.md) | what is trusted, what is scanned, what is exposed |
| [design.md](design.md) | this file: the convention, and decisions with their reasons |
| [publishing.md](publishing.md) | how a release becomes binaries, a tap, and an update notice |
| glossary.md | domain terms; plankit has none that git's and pk's vocabulary does not cover, so the file does not exist |

The references are the working area: one `pk-<command>.md` per command, plus [pk-json.md](pk-json.md), [error-reference.md](error-reference.md), [environment-variables.md](environment-variables.md), and the procedure [changing-flow.md](changing-flow.md). They live beside the code because they change in the same commit as the flag or message they state, and a tagged checkout carries the reference that matches its binary.

`docs/archive/` holds essays and notes kept for reference, not maintained; its [README.md](archive/README.md) indexes them with one line each.

**Rewriting a document.** List the questions it answers and map each to a member of the set. Keep every fact, command, decision, and limit; move each to its home. Delete every other sentence. Search each surviving sentence across the project; a second copy becomes a link. Read the first sentence: it states what the thing is, or the file is not finished. The measure is fewer words and the same facts.

## Squash merge and release tags

A tag on a commit that is later squash-merged is orphaned. A tag is created at commit C on a source branch; the PR is squash-merged, creating a new commit S; the tag still points at C, which is not an ancestor of the target branch. `git log` on the target never shows it, `git describe` finds nothing, and CI looking for tags on the release branch sees no release.

`pk release`'s own flow is not affected: it tags the fast-forwarded release branch, so the tag is always an ancestor. The decision: merge commits and rebase only, for any branch that carries release tags; the shipped ruleset excludes squash ([branch-protection.md](branch-protection.md)). If squash is needed elsewhere, tag after the merge lands, never before.

## Line endings

plankit's root `.gitattributes` is `* text=auto eol=lf`. Two reasons. On Windows, `core.autocrlf=true` prints `LF will be replaced by CRLF` for every staged LF file when no `.gitattributes` exists; a `preCommit` hook or `git add -u` can print it dozens of times per release. And the managed files pk ships must stay byte-stable, or the hidden-character scan in `make test` would trip on a stray CR. `text=auto` leaves binary files alone; `eol=lf` normalises text in the repository and on checkout on every platform. Without a `.gitattributes`, `git config core.autocrlf input` silences the warning on one machine.

## No deploy command

pk ships releases and has no deploy command. pk itself is a distributed artifact: the shipped version is installed on machines its maintainer does not control, so there is no single running version to replace, and moving forward means publishing a release for consumers to pull.

For a service you operate, the tag list is still the whole deploy model: deploy ships the latest tag, rollback ships the tag before it, both idempotent, neither reading what is currently live. Rollback is safe only while the previous version works against the current data; a change that breaks that coexistence (a data migration) is rolled forward, not back. A release checkout verifies the build against exactly what ships, and a service that reports its own version makes "does live match the tag" answerable. Reopen if pk ever grows a hosted component.

## The em-dash check

`pk rules --lint --strict` rejects em dashes in rule files. The check began as a proxy, added when nothing checked content: the LLM tell was never the glyph but the vague clause it joined, and the glyph was cheap to scan for. The shipped documentation rule now checks the content directly. Revisit the check after a full rewrite pass has held; removing it changes the linter for every pk-managed project, so the removal is its own commit and release note, never a side effect.

## No pk preview command

Previewing a release is fifteen lines of a SKILL.md: the `/ship` skill runs each command's `--dry-run` and asks. A binary command would add surface without capability. Reopen if a preview ever needs logic a skill cannot carry.

## Guarantees live in the CLI

A rule can be ignored; a hook cannot. Whatever must hold in every session is a pk command, and a fix to how the model behaves is a change to pk, filed as an issue.
