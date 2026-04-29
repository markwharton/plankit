---
name: release-notes
description: Generate a prompt for writing a "Notes from the workshop" entry on plankit.com from plankit's latest CHANGELOG
disable-model-invocation: true
---

Generate a prompt for writing a new "Notes from the workshop" entry on plankit.com. Reads plankit's own `CHANGELOG.md`, distills user-visible changes, and outputs a ready-to-paste prompt for a plankit.com session.

The skill operates on plankit only. It does not read or write to the plankit.com repo — output is a self-contained prompt the user pastes into a new plankit.com session. Push, not pull.

## Steps

1. Resolve inputs:
   - **Tool slug** — from a `tool` argument if supplied, else default to `pk`. plankit.com is multi-tool: `pk` → `/pk/notes/`, `mcp-bridge` → `/mcp-bridge/notes/`, `signals` → `/signals/notes/`. The slug drives every path in the emitted prompt.
   - **Version** — from a `version` argument if supplied (e.g., `v0.12.0`), else the most recent tag: `git describe --tags --abbrev=0`.
   - **Repo name** — `basename $(git rev-parse --show-toplevel)`. Fills the `<REPO>` slot in the emitted Context block so the receiving session sees both the source repo and the binary (they differ for plankit: repo `plankit`, binary `pk`).
2. Read `CHANGELOG.md` and extract the section for the target version (all categories: Added, Fixed, Documentation, Maintenance, etc.).
3. Distill each commit in the section:
   - **User-visible** — keep: new flags, behavior changes, stderr message changes, new commands, breaking changes.
   - **Rule updates** — case-by-case. Rules in `plankit-development.md` are maintainer-side and never appear in release notes (they don't ship to users and don't affect their projects). Rules in `model-behavior.md`, `plankit-tooling.md`, or other files that ship via `pk setup` can be mentioned — but only if the user will *feel* the difference (Claude behaves measurably differently for them). Most rule updates are infrastructure and should be skipped; the user experiences the effect without needing to know a rule landed.
   - **Internal** — drop: refactors, test-only changes, release-mechanism housekeeping (e.g., "pin install-pk.sh"), doc reorganizations that don't change behavior.
   When in doubt, ask: would a pk user do something differently tomorrow because of this commit? If no, drop it.
4. For each kept item, rewrite in the voice defined below — one short line about what the user can do differently now, not what code moved.
5. Assemble the prompt using the template in the "Prompt to emit" section. Print the entire prompt to stdout.

The user then copies the prompt into a new Claude session opened in `~/Projects/markwharton/plankit.com` and the plankit.com session writes the HTML entry.

## Voice for the entries (authoritative)

- **Practical, honest, concrete.** No "we are excited to announce" or launch-speak. plankit is a workshop, not a product launch.
- **Lead with what the user can do differently now**, not what commit landed.
- **One or two short paragraphs per entry, or a focused bullet list.** Not a wall of text.
- **Bold one key command or noun per entry** if it helps scanning.
- **No forced workshop metaphors.** "Notes from the workshop" is the section name — that's the extent of the metaphor. Do not try to make the content sound like a physical workshop. Avoid: "sharper edges," "tools on the bench," "pieces fitting together," "sawhorse," "jig," "lathe," "chisel," "grain" — they read as twee. Use words already in a developer's vocabulary: build, cut (as in "cut a release"), make, ship, land, polish, tighten, refactor. "Craft" is fine — it's found its way into AI-assisted development vocabulary. When no natural word fits, use plain descriptive language. No forced flavor.
- **Link to the full CHANGELOG on GitHub** for readers who want every detail.
- **Drop internal churn.** Refactors, tests, doc reorganizations that don't change behavior do not appear.

**Good example:**

> **v0.11.0 — `pk setup --baseline`.** Anchoring `pk changelog` just got one step simpler. If you use `pk changelog` / `pk release`, run `pk setup --baseline --push` — it tags `v0.0.0` on HEAD and publishes it. Works on new repos and existing ones; pass `--at <ref>` to fold prior commits into the first changelog entry. No more remembering raw git commands.

**Bad (do not produce output like this):**

> - soften version-tag tip and surface --at in changelog error (ed97860)
> - add --baseline to setup options enumerations (4df4838)
> - add Clarifications, Evolving pk Commands, and Tip Messages (255ce88)

That's the CHANGELOG. This skill's job is to turn it into something a human reads.

## Prompt to emit

After distilling, emit a prompt with this structure. Fill the `<BRACKETS>` with the distilled content.

```
# Write release notes for plankit.com — <TOOL> <VERSION>

Context: <TOOL> <VERSION> shipped (repo: <REPO>, binary: <TOOL>). Write
a new "Notes from the workshop" entry at `site/<TOOL>/notes/index.html`.
If that file doesn't exist yet, create it; if it exists, add the new
entry at the top (newest first).

## Entry content (already curated — use as the source for the entry text)

**<VERSION> — <DATE> — <SHORT THEME LABEL>**

<CURATED BULLETS OR PARAGRAPHS>

## Voice rules (don't deviate)

- Practical, honest, concrete. No launch-speak ("we're excited to announce").
- Lead with what the user can do differently now, not what commit landed.
- Short — one or two paragraphs OR a focused bullet list per entry.
- Bold one key command or noun per entry for scanning.
- Workshop metaphors where natural, not forced.
- Link to full CHANGELOG on GitHub for detail.

## Visual style

plankit.com pages share `site/style.css`. The notes page links to it
and uses the existing components (`.section`, `.features`, etc.) —
read an existing page for the pattern and mirror it. Do not invent new
tokens, new layouts, or new CSS files.

## Link points

After the notes page is in place, `site/<TOOL>/index.html` should have
a link to `/<TOOL>/notes/` in its Docs section. Verify the end state:
if the link is already present, no change needed — do not make an
empty commit. If missing, add it as a separate commit from the notes
entry. Do not link from `site/<TOOL>/start/index.html` — that page
stays focused on first-run.

## Constraints (explicitly NOT to build)

- No RSS feed, no `feed.xml`, no feed generators.
- No static site generator, no markdown-to-HTML build step.
- No raw CHANGELOG dump. This is curated, not transcribed.
- No third-party trackers or analytics (Google Analytics, Meta pixels, etc.) and no comment systems (Disqus, etc.). plankit signals — plankit's own metrics beacon, when present — is welcome; the notes page inherits it site-wide and shouldn't add its own.
- No "we are excited to announce" language — ever.

## Flow

1. Read 2–3 existing plankit.com pages to absorb style.
2. Draft `site/<TOOL>/notes/index.html` — show the user before writing files.
3. Iterate on voice/length if needed.
4. Check the end-state link from the tool landing page; add only if missing, as a separate commit.
5. Commit on develop with a clear message, then run `/ship` to release.

Draft → user review → commit → release. Don't bundle the notes page
and the link addition into one commit — they're two logical changes.
```

## Contract

- **Inputs:** optional `tool` argument (slug, default `pk` — e.g., `pk`, `mcp-bridge`, `signals`) and optional `version` argument (string like `v0.11.1`).
- **Output:** the full prompt printed to stdout, ready for the user to copy.
- **Side effects:** none. No writes, no cross-repo reads, no network calls.
