# Add custom humans.txt for plankit.life

## Context

The plankit.life site is deployed via GitHub Pages. GitHub Pages serves a default `humans.txt` for every Pages site ("Oh hello! Nice to see you. Made with ❤️ by humans.txt"), which is what showed up in the recent Pages deploy log. Since plankit is a project about human–AI collaboration — a developer directing Claude Code toward deterministic outcomes — the default file is a missed opportunity to carry the site's voice and credit the actual authorship honestly.

The goal is to ship a custom `site/humans.txt` that:
- credits the human author (Mark Wharton) as the builder,
- acknowledges Claude Code honestly under `/* TOOLS */` rather than `/* TEAM */` (lines up with the site's "developer directs, Claude executes" stance),
- thanks the three authors quoted on the landing page (HBR, Lakein, Horowitz),
- closes with a warm, personal note in the site's "close the laptop" voice — an invitation to check in with family and friends, rather than a clipped reference to the landing page line.

## Scope

One new file. No code changes, no workflow changes.

## Change

### New file: `site/humans.txt`

```
/* TEAM */
  Builder: Mark Wharton
  Site: https://plankit.life
  GitHub: @markwharton

/* THANKS */
  Harvard Business Review — "AI doesn't reduce work — it intensifies it"
  Alan Lakein — How to Get Control of Your Time and Your Life
  Ben Horowitz — The Hard Thing About Hard Things

/* TOOLS */
  Claude Code
  GitHub Pages
  Google Fonts (Source Serif 4, Inter)

/* SITE */
  Language: English
  Standards: HTML5, CSS3

/* If you're reading this,
   close the laptop and take a minute
   to check in with your family and friends. ❤️ */
```

Notes on the content:
- The `THANKS` section lists only the three people actually cited on the landing page (`site/index.html:200`, `site/index.html:225`, `site/index.html:237`), so the file stays truthful rather than performative.
- `TOOLS` reflects the actual stack of the *site* (not the `pk` CLI): it is hand-written HTML/CSS, no build step, Google Fonts loaded at runtime, hosted on GitHub Pages, authored with Claude Code.
- **No `Last update` field.** The humans.txt convention treats that field as the date the `humans.txt` file itself was last updated, but the field is ambiguous in practice and creates a maintenance trap — any hardcoded date will drift out of sync with the site and invite the question "do I need to bump this?" every time `index.html` changes. The field is optional in the humanstxt.org spec, so it is simply omitted. `git log -- site/humans.txt` is the authoritative source for when this file last changed.

## Why no workflow changes

`.github/workflows/pages.yml:31` uploads the entire `site/` directory as the Pages artifact via `actions/upload-pages-artifact`. Any new file dropped into `site/` is automatically served at the site root, so `site/humans.txt` will be live at `https://plankit.life/humans.txt` the next time `pages.yml` runs on `main`. No edits to `pages.yml` are needed.

## Files touched

- **Create**: `site/humans.txt` (new, ~20 lines)

No other files are modified.

## Verification

Local (before committing):
1. Create the file exactly as shown above.
2. Open it in a browser directly — `open site/humans.txt` — and confirm the text renders cleanly and the em-dashes are intact (UTF-8).
3. Optionally serve the `site/` directory locally to confirm the URL path works:
   ```
   cd site && python3 -m http.server 8000
   ```
   then visit `http://localhost:8000/humans.txt` and confirm it loads.

After release (separate step, user's call):
1. This is a `docs:` change on `dev`. Follow the normal flow the user prefers — the project convention per `CLAUDE.md` is `pk changelog` on `dev` then `pk release` to merge and push. Do **not** run either of those from this plan; they are release decisions for the user.
2. Once `main` is updated, the `Pages` workflow (`.github/workflows/pages.yml`) will trigger automatically because the change is under `site/**`.
3. After the workflow finishes (~20s based on the prior run), confirm:
   - `curl -sS https://plankit.life/humans.txt` returns the new file contents, not GitHub's default "Oh hello!" message.
   - The `Pages` job in the Actions tab shows a successful deploy.

## Out of scope

- No changes to `index.html`, no link from the landing page to `humans.txt` (it's a convention-discoverable file, not a navigation target).
- No changes to `pages.yml`, `release.yml`, or any other workflow.
- No changelog entry is written by this plan — `pk changelog` will pick up the `docs:` commit automatically on the next release.
