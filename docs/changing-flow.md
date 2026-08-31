# Changing flow


Moving a repository between trunk flow (one branch, releases tagged on it) and merge flow (a working branch, `pk release` fast-forwards the release branch). The tags already made stay valid in both directions.

**`pk status` is the dashboard.** Its Readiness section evaluates the gap between what `.pk.json` declares and what the repository can actually do (baseline tag, branches on origin), and names the exact command that closes each gap. Re-run it after each step; when it reports `ready for pk changelog / pk release` and names the flow you intended, the transition is done. See [pk status](pk-status.md).

**`/pk-configure` walks the transition.** It detects the current setup (branches, remotes, tags, unpushed commits), asks where you want to land, writes `.pk.json`, and offers to create and publish a missing working branch — previewing the exact commands and acting only on your confirmation.

### Main-only to main/develop

Starting point: everything on `main`, pushed straight to origin. Target: `main` protected and release-managed, day-to-day work on `develop`.

1. **Configure the target** in `.pk.json` (or let `/pk-configure` do it):

   ```json
   {
     "guard": { "branches": ["main"] },
     "release": { "branch": "main" }
   }
   ```

2. **Create and publish `develop`.** One judgment call first: if local `main` is ahead of `origin/main`, decide where `develop` starts. Branching from local `main` carries the unpushed commits onto `develop` (they reach `main` again at the next release); branching from `origin/main` leaves them on `main`, which is now protected — so carrying them is usually what you want. Check with `git log origin/main..main --oneline`, then:

   ```bash
   git branch develop main      # or: git branch develop origin/main
   git switch develop
   git push -u origin develop
   ```

3. **Anchor the baseline tag** if there is none: `pk setup --baseline --push` (see [Layer 3](pk-setup.md#baseline) for `--at` placement).

4. **Confirm:** `pk status` → `Readiness: ready for pk changelog / pk release (merge flow into main)`. From here, work lands on `develop` and `/ship` publishes releases to `main`.

Until the transition is complete, each release-command refusal names the command that fixes it: `pk changelog` on a protected `main` with no other branch suggests creating `develop`, and `pk release` refuses with the same hint when the working branch is missing.

### Trunk flow to merge flow

Already releasing from a single branch (no `release.branch`)? Adding `release.branch` to `.pk.json` switches `pk release` from tag-and-push-current-branch to merge-then-tag-then-push. The steps are the same as above: add the config, create the working branch, confirm with `pk status`. Releases published from trunk flow remain valid; the next release simply merges instead.

### Merge flow to trunk flow

The reverse move: a project that no longer needs a working branch separate from the release branch. A Homebrew tap is the typical case: automated bump PRs, no code of its own, nothing to stage. `.pk.json` without `release.branch` selects trunk flow, and `guard.branches` has to go too, since a branch guard on the only branch would block every commit.

The order matters, because `pk release` reads `.pk.json` from the working tree at run time. Remove `release.branch` on the working branch and then release, and `pk release` runs in trunk mode *there*: it tags and pushes the working branch, and the release branch never receives the commits. So the config change is the last thing to land, and it lands on the release branch:

1. **On the working branch**, make every other change for the new shape (CI that checks out or targets the working branch, Dependabot `target-branch`, docs) and cut one last merge-flow release, so those commits reach the release branch through the normal path.
2. **Switch to the release branch.** Delete the working branch locally and on origin. Confirm only the release branch remains.
3. **From your own terminal, not a Claude Code session,** remove `release.branch` and `guard.branches` from `.pk.json` and commit. Until this commit lands, `guard.branches` still names the checked-out branch, so `pk guard` blocks the commit inside a session; the block is `guard.mode: block` working as configured, and this one commit is the developer's to make. Keep `guard.mode`, `guard.push`, and `preserve.mode`.
4. **Confirm:** `pk status` → `Readiness: ready for pk changelog / pk release (trunk flow; no release.branch in .pk.json)`. `pk changelog --dry-run` shows that one config commit; it is released with the next real change, not on its own. `pk setup` will now end each run with the reminder that `.pk.json` names no release branch; in trunk flow the reminder needs no action.

Tags made under merge flow remain valid; the next release tags the current branch and pushes it with the tag.