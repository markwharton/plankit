# Review fixes, phase 6: tokens

## Context

A code review of the repository at v1.8.3 is being fixed in phases.
This is the last. The earlier phases fixed defects and gave each check
and message one home. What is left is the vocabulary pk and its site
are written in: values that every reader spells for itself where one
definition would do.

- The mode values (`off`, `ask`, `block`, `auto`, `manual`) are string
  literals at every reader in guard, brief, and preserve. `config`
  names only the defaults, so a misspelt value in a reader compiles.
- The help index hard-codes its two escape codes beside the style
  table the rest of `pk help` renders from.
- Three places join a list into a sentence, two with a serial comma
  and one without.
- Narration, the plain lines a command prints as it works, is written
  with `fmt` at each site and in two styles: `pk changelog` prints
  "no commits since v1.0.0" and "Found 3 conventional commit(s)", and
  three lines end in a full stop.
- A hook's permission reason has three shapes: guard opens three with
  "pk guard:" and two with "Branch ... is protected by pk guard";
  protect's has no prefix.
- The site writes the repository's address in three places and a
  platform count ("six platforms") copied by hand from the Makefile.
- The stylesheet tokenises its colours and nothing else: the monospace
  stack is written three times (and once as bare `monospace`), there
  are five radii and ten font sizes, the page gutter is repeated three
  times, and the small uppercase label is defined twice with
  near-equal values.

The outcome: seven local commits on `develop`, the suite green before
and after each. Nothing is pushed and no release is cut.

## 1. Mode values have names

`internal/config/config.go` names the five values (`ModeAsk`,
`ModeAuto`, `ModeBlock`, `ModeManual`, `ModeOff`). The defaults and
the settings table in `config` are written with them, and so is every
reader: guard's four policies, the brief's sentences, preserve's
switch. Claude Code's own permission words in `hookio`
(`PermissionAsk` and the others) are a different vocabulary and stay.

Commit: `refactor(config): mode values have names`

## 2. The help index is styled from the style table

`internal/help`: one function returns the style table for a style
(plain or ANSI); `Render` and the index both call it, and the index
takes its bold and reset from the table. The two escape codes written
into `help.go` go.

Commit: `refactor(help): the index is styled from the style table`

## 3. One way to join a list in a sentence

`internal/cli` exports `JoinList(items, conjunction)`: "a", "a and b",
"a, b, and c". The usage error for `--format`, the universal-flags
pointer line docgen writes into each page, and the allowed-values list
in each Settings section all use it.

The generated pages do not change: both docgen callers already wrote
the serial comma. The `--format` error would gain one for three or
more formats; no command has three.

Commit: `refactor(cli): one way to join a list in a sentence`

## 4. Narration goes through msg, in one style

`internal/msg` gains `Sayf`, one line of narration with no prefix:
what a command is doing or has done. The plain `fmt` lines on stderr
in `changelog`, `release`, `ship`, `preserve`, and `repo` are written
with it. The style is a capital at the start and no full stop, which
most lines already follow. The lines that change:

- `pk changelog`: "Fetched tags from origin: latest is %s", "No release
  yet: the first release covers all %d commits", "No commits since
  %s", and the two lines that ended in a full stop ("No conventional
  commits remain after --exclude", "No visible commits after grouping
  (all hidden or unmapped types)"). A line that opens with a number
  stays as it is.
- `pk preserve`: "No pending plan to preserve" and "No pending plan to
  discard" lose their full stops. The two `--dry-run` blocks, aligned
  by hand, become a `msg.Section` with `msg.Itemf` lines.
- `pk release --dry-run`: the last item is one sentence, "All checks
  passed; run without --dry-run to release".

A line that begins with a name keeps the name's spelling ("plankit
configured in ..."). Errors, warnings, notes, and hints are not
narration and keep their prefixes and wording.

`skills/changelog/SKILL.md` quotes two of these lines and is updated
with them.

Commit: `refactor: narration goes through msg, in one style`

## 5. Every permission reason opens with the hook's name

`internal/hookio`: `WritePermissionDecision` takes the hook's name and
writes the reason as "pk <hook>: <reason>", the prefix `msg.Hookf`
already gives a hook's stderr lines. guard's six reasons and protect's
one are written without their own prefix. The two branch reasons read
"pk guard: branch "main" is protected. Switch to ...", and protect's
reads "pk protect: docs/plans/ files are immutable ...".

Commit: `refactor(hooks): every permission reason opens with the hook's name`

## 6. The site's repository address is written once

- `tools/docgen/site.go`: one constant holds the repository's address.
  The "Design (GitHub)" link and the latest-release link are built
  from it, and the layout's footer link reads it from the page data.
- The phrase "six platforms" goes from the front page's release line:
  it is a count copied by hand, and the downloads are listed where
  they are served.

Commit: `refactor(site): the repository address is written once, and the platform count goes`

## 7. The stylesheet's sizes, radii, fonts, and gutter are tokens

`site/style.css`, beside the colour tokens in `:root`:

- `--mono`: the monospace stack, used by the sidebar links, code, and
  both labels.
- `--radius` (4px) for inline code and code blocks, `--radius-lg` (6px)
  for terminals and cards. The 999px pill stays.
- `--text-xs` (.78rem) for the two labels, `--text-sm` (.875rem) for
  the sidebar links, code blocks, card text, and the footer,
  `--text-lg` (1.15rem) for the wordmark and the lead paragraph.
  Headings and the `.92em` inline code stay as they are.
- `--gutter` (1.25rem), the horizontal padding of the header, the
  page, and the footer.
- The small uppercase label is one rule shared by the sidebar label
  and the terminal caption.

Nothing moves by more than a pixel: the largest change is a code
block's text going from 13.44px to 14px.

Commit: `style(site): the stylesheet's sizes, radii, fonts, and gutter are tokens`

## Where each fact lives after this phase

- A mode's values: `internal/config`.
- The help style table: `internal/help/engine.go`.
- Joining a list: `cli.JoinList`.
- A line of narration: `msg.Sayf`. A hook's prefix: `msg.Hookf` for
  stderr and `hookio.WritePermissionDecision` for a reason.
- The repository's address on the site: one constant in docgen.
- A size, a radius, the monospace stack, the gutter: `:root` in
  `site/style.css`.
- Check: a search of `internal/` outside `config` and `hookio` for the
  quoted mode values, and of `site/style.css` outside `:root` for
  `px` radii and `rem` font sizes other than the headings', shows no
  hits.

## Verification

- `make test` before the first change and after each of the seven,
  with the output shown.
- `make docs` leaves no diff after steps 1 to 3 and 5; after step 4 it
  rewrites only the changelog page's two quoted lines.
- The site, built before the phase with a fixed `VERSION` and copied
  aside, is compared after each step. Steps 1 to 5 show no difference
  beyond the changelog page's two lines. Step 6 shows the release line
  on the front page and nothing else. Step 7 shows `style.css` and the
  stylesheet's hash in each page's link, and nothing else.
- Smoke test against a fresh build, in scratch repositories whose bare
  origin is made by cloning, so nothing is pushed: `pk changelog` on a
  first release and with nothing to release, `pk preserve --dry-run`,
  a guard deny on a protected branch, and a protect deny, each printed
  to show the new line.
- The case that must fail: a `.pk.json` with `guard.mode` set to
  `blok` is still refused at load, naming the key and the allowed
  values.
- `claude plugin validate . --strict` passes; `gofmt -l cmd internal
  tools` prints nothing.
- The stylesheet is not checked by eye here. `make site-serve` serves
  the preview for a look before the site is next deployed.

## After this phase

Every finding of the review is fixed. `develop` then holds the whole
set, unpushed. The commits infer a minor release, so a release note in
`docs/notes/` is written before `pk ship`.
