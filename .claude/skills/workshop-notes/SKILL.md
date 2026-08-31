---
name: workshop-notes
description: Emit a prompt for a plankit.com notes entry, either a release note distilled from plankit's CHANGELOG or a field note about a real incident
disable-model-invocation: true
argument-hint: [release [version]] | [field [ref]]
---

Generate a prompt for writing a notes entry on plankit.com. Two kinds of entry, one skill, one voice:

- **Release notes** (`/release-notes/` on the site): what a pk release changed and why, distilled from plankit's own `CHANGELOG.md`. Written when it ships.
- **Field notes** (`/field-notes/` on the site): a dated account of something that broke or nearly broke in a real repository, and what changed in pk or in how you work because of it. Written when it happens.

The skill operates on plankit only. It reads plankit's CHANGELOG and git history and never reads or writes the site repository; the output is a self-contained prompt the user pastes into a Claude Code session opened in the site repository (`~/Projects/markwharton/plankit-www` until it is renamed `plankit.com`). Push, not pull.

## Usage

- `/workshop-notes` or `/workshop-notes release`: release note for the latest tag.
- `/workshop-notes release v0.29.2`: release note for a named version.
- `/workshop-notes field <ref>`: field note whose fix landed at `<ref>` (a commit or tag). The skill reads the commits for the "Change" paragraph.
- `/workshop-notes field`: field note for an incident that produced no code change (a guard doing its job, a recovery worth recording). The skill asks for everything.

## Release mode

### When to use

Only for releases that contain at least one `feat`, `fix` a user would notice, `security`, or breaking change. Pure refactor/chore/docs releases do not warrant an entry. Check the CHANGELOG section for the target version first; if nothing qualifies, say so and stop.

### Steps

1. Resolve the version: the argument if given, else `git describe --tags --abbrev=0`.
2. Read `CHANGELOG.md` and extract the section for that version, every category.
3. Distill each commit:
   - **User-visible**, keep: new flags, behaviour changes, stderr message changes, new commands, breaking changes.
   - **Rule updates**, case by case. Rules in `development.md` are maintainer-side and never appear. Rules that ship via `pk setup` (`craft.md`, `conduct.md`) may be mentioned only if the user will feel the difference; most are infrastructure and are skipped.
   - **Internal**, drop: refactors, test-only changes, release housekeeping (pinning `install-pk.sh`, footprint updates), doc reorganisation that changes no behaviour.
   When in doubt: would a pk user do something differently tomorrow because of this commit? If not, drop it.
4. Rewrite each kept item in the voice below: what the user can do differently now, not what code moved.
5. Emit the release prompt (template below).

## Field mode

### When to use

Something happened in a real repository that is worth other people knowing: a tool caught a mistake, a tool failed to, a recovery had a non-obvious order, a guarantee was added because of it. Not for hypotheticals, and not for pk bugs with no story (those are the CHANGELOG's job).

### Steps

1. If a `ref` was given: read `git log --format='%h %s%n%b' <base>..<ref>` where `<base>` is the tag before `<ref>` (or `<ref>~1` for a single commit) and note the pk version as the nearest tag reachable from `<ref>` (`git describe --tags --abbrev=0 <ref>`). These feed the "Change" paragraph and the version label; they are the only parts the skill can derive.
2. Ask the user for the rest, as plain bullets under a short heading:

   Tell me about the incident:
   - What happened, in the order it happened, and what pk printed at each step?
   - What was the cause?
   - How was it recovered, if it needed recovering?
   - What changed because of it (in pk, in a doc, or in how you work), if the commits above do not already say?
   - Which site page does it belong beside (Construct, Adopt, Ship, Plans, Hooks)?

3. Draft the entry from the answers, in the field-note shape (below), and show it in the emitted prompt as the curated source. Do not soften it into a general lesson; the value is the specific sequence.
4. Emit the field prompt (template below).

## Voice (authoritative for both kinds)

- **Practical, honest, concrete.** No launch-speak, no "we're excited". plankit is small tools, carefully made, and the notes read that way.
- **Release notes lead with what the user can do differently now.** Field notes lead with what happened, in order, with the messages pk printed.
- **First person is fine in field notes** ("I ran", "we merged"). It is a real account, not a case study.
- **Short.** A release entry is one or two paragraphs or a focused bullet list; a field entry is four short paragraphs at most.
- **Bold one key command or noun per release entry** for scanning. Field entries use the fixed labels instead (Cause, Recovery, Change).
- **No forced workshop metaphors.** Use words already in a developer's vocabulary: build, cut, ship, land, polish, tighten. No sawhorses, jigs, or grain.
- **Prose says what and why; anything to type is a code block; anything a tool printed is a code block; inline code is for names only** (files, branches, keys, command names, commit messages). This is the site's house rule and the entry must follow it.
- **No private project names or paths.** A field note describes the shape of a repository, never which one it was, unless it is one of plankit's own.
- **Link to the CHANGELOG on GitHub** from release entries; link to the relevant site page from field entries.

**Good release entry:**

> **v0.11.0 — 2026-04-18 — `pk setup --baseline`.** Anchoring `pk changelog` is now one command. If you use `pk changelog` / `pk release`, run `pk setup --baseline --push`: it tags `v0.0.0` on HEAD and publishes it. Works on new repos and existing ones; pass `--at <ref>` to fold prior commits into the first changelog entry.

**Bad (this is the CHANGELOG, not a note):**

> - soften version-tag tip and surface --at in changelog error (ed97860)
> - add --baseline to setup options enumerations (4df4838)

**Field entry shape** (as it appears on the site):

> **The ruleset guarded the wrong branch** — 15 August 2026, pk 0.29.0
>
> The emitted `.github/protect-main.json` targeted `~DEFAULT_BRANCH`. In a repository where `develop` had been made the default, the ruleset moved with it and left `main` open. A pull request opened against `main` by mistake merged straight in.
>
> **Change.** The ruleset is rendered from `release.branch`: `refs/heads/main` by name, and all three of name, ref, and file follow a non-default release branch. See Construct, step 5.

Labels used in a field entry, in this order, each starting its own paragraph: the account (no label), **Cause.**, **Recovery.** (omit if none was needed), **Change.** (say "None to pk" if that is the point).

## Prompt to emit: release

```
# Write a release note for plankit.com: pk <VERSION>

Context: pk <VERSION> shipped (repo: plankit, binary: pk). Add a new
entry at the top of `site/release-notes/index.html` (newest first).

## Entry content (already curated; use as the source for the entry text)

**<VERSION> — <DATE> — <SHORT THEME LABEL>**

<CURATED BULLETS OR PARAGRAPHS>

## Form

The page is a sequence of `<article class="note">` blocks. Each has an
`<h2>` reading `vX.Y.Z &mdash; YYYY-MM-DD &mdash; theme` followed by one
or more `<p>`. Copy the form of the entry above yours exactly; do not
introduce new markup, classes, or CSS. Existing entries are the style
guide for length and voice.

## Voice rules (do not deviate)

- Practical, honest, concrete. No launch-speak.
- Lead with what the user can do differently now, not what commit landed.
- One or two short paragraphs or a focused bullet list.
- Bold one key command or noun for scanning.
- Prose says what and why; commands and printed output go in code
  blocks; inline code is for names only.
- No workshop metaphors; plain developer words.
- Link to the CHANGELOG on GitHub for the full list.

## Constraints

- No RSS feed, no build step, no CHANGELOG dump, no analytics or
  comment systems, no external requests. The site is self-contained.

## Flow

1. Read the top two entries of `site/release-notes/index.html`.
2. Draft the new entry and show it before writing the file.
3. Write it, commit on develop as `docs(site): add pk <VERSION> release notes`.
4. `/ship` when the user says so.
```

## Prompt to emit: field

```
# Write a field note for plankit.com: <TITLE>

Context: a real incident, dated <DATE>, pk <VERSION>. Add a new entry
at the top of `site/field-notes/index.html` (newest first).

## Entry content (already curated; use as the source for the entry text)

**<TITLE>** — <DATE>, pk <VERSION>

<ACCOUNT PARAGRAPH: what happened, in order, with what pk printed>

**Cause.** <one paragraph>

**Recovery.** <one paragraph, or omit>

**Change.** <one paragraph; "None to pk" if that is the point> See <PAGE>.

## Form

The page is a sequence of `<article class="note">` blocks. Each has an
`<h2>` holding the title and a `<small>` with `<time datetime="YYYY-MM-DD">`
and the pk version, then the paragraphs above with the labels in bold.
Copy the form of the entry above yours exactly; no new markup, classes,
or CSS. Anything pk printed goes in a `<pre><code>` block, not inline.

## Voice rules (do not deviate)

- First person, concrete, in the order it happened.
- Prose says what and why; commands and printed output go in code
  blocks; inline code is for names only.
- No private project names or paths.
- Do not generalise it into a lesson; the specific sequence is the value.
- Link to the site page it belongs beside.

## Constraints

- No new pages, no CSS changes, no external requests.

## Flow

1. Read the top entry of `site/field-notes/index.html`.
2. Draft the new entry and show it before writing the file.
3. Write it, commit on develop as `docs(site): field note: <short title>`.
4. `/ship` when the user says so.
```

## Contract

- **Inputs:** `release [version]` (default: latest tag) or `field [ref]`.
- **Output:** the full prompt printed to stdout, ready to paste into a session in the site repository.
- **Side effects:** none. Reads plankit's CHANGELOG and git log only; no writes, no cross-repo reads, no network.
