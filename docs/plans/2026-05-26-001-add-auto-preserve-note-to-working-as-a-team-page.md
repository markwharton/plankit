# Add auto-preserve note to Working as a Team page

## Context

In the busiest repos, preserve mode is set to `auto` via `pk setup --preserve auto`. The default `manual` mode prompts after plan approval, which interrupts the flow of both the developer and the session. Auto mode saves the plan in the background the moment it is approved, so work continues without a pause. This is a practical refinement worth documenting on the Working as a Team page.

## Change

**File:** `docs/working-as-a-team.md`

Append two sentences to the end of paragraph 2 (the one ending "The plan shows why."):

> By default, preservation is manual: Claude Code prompts after plan approval, and you run `/preserve` when ready. In our busiest repos we use `pk setup --preserve auto` so the plan is saved in the background the moment it is approved, and the session moves straight to implementation.

No other files need to change.

## Verification

- Read the page after editing to confirm flow and formatting.
- `make lint` to ensure no build issues (doc-only change, but good hygiene).
- `make test` to confirm nothing broke.
