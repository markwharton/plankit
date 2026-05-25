# Working as a Team

This is how a small team uses plankit day to day. The [Methodology](methodology.md) explains the model: the developer controls the system, and the model executes within it. This is what that looks like in practice.

The plan is a shared artifact. One or two of us review and approve an approach, and that becomes the record. We spend our review there, up front, because that's the gate that matters: catching a wrong approach in the plan costs minutes, catching it after the code is written costs hours. Once it's approved, we build against it. The hard review already happened, so no one is blocked waiting on sign-offs afterward. That is what lets a small team move fast. Not less review. Review where it's cheapest.

The approved plan is preserved to `docs/plans/` and protected from edits. It won't match the final result exactly, and it isn't meant to; implementation always surfaces changes. We treat it as alignment, not enforcement: something to call up when work drifts, not a contract to argue over. The commit history shows what changed. The plan shows why.

What lets us trust the speed is that pk sits between the model and the repo (see [Architecture](architecture.md)). Git guards block mutations on the branches we protect, locally, before anything lands, so a fast-moving session can't quietly push a mistake somewhere that matters. The guarantee lives in the tooling, not in remembering to be careful.

The skills give everyone the same path through it. `/conventions`, `/preserve`, and `/ship` are the same commands and the same workflow whether you are a senior developer who wants the guardrails out of the way or someone newer who needs a path to follow before they have internalized every rule.

None of this works alone. Plans, guidelines, review, guards: discipline is the multiplier, and the result belongs to the system, not any single part. For a small team that is the payoff. You get the discipline of a much larger process without the overhead of one.
