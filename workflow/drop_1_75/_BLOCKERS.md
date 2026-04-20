# Drop 1.75 — Blockers

Canonical `blocked_by` tracker for this drop's droplets. Source of truth for cross-sibling ordering when the planner or build-QA need to check the dispatch graph at a glance. In the cascade-proper workflow (`AGENT_CASCADE_DESIGN.md` §8 + `workflow/example/drops/WORKFLOW.md`) this file lives at every branched planner dir and tracks `blocked_by` between that dir's immediate children (sub-drops or droplets).

Once Tillsyn's Drop 1 `paths` / `packages` / `blocked_by` domain fields land on `ActionItem`, this MD becomes the one-shot migration source for that drop's wiring.

## Drop Shape

**Grandfathered flat.** Drop 1.75 was planned before the cascade doctrine fully landed. Its `PLAN.md` lists 15 droplets at one level — no L2 sub-drop dirs, no nested planners. Per dev directive (2026-04-19) the flat shape is kept; cascade-proper decomposition (multi-level planners, sub-drops per domain concern) applies from Drop 2 onward. Tracked as a refinement under `DROP_1_75_REFINEMENTS_RAISED` at drop close.

## Droplet Dependency Graph

Authoritative `blocked_by` wiring lives inline in each unit's section of `PLAN.md` (the `Blocked by:` bullet). This file mirrors that wiring in one readable block.

*To be populated at Round 10 plan-QA close — once the 15 units stabilize, extract each unit's `Blocked by:` into the table below.*

| Droplet | Blocked by | Reason |
|---|---|---|
| *(extracted at plan-lock)* | | |

## External Blockers

- None. Drop 1.75 has no cross-drop `blocked_by` — the script-rewrite and snapshot bump land as part of this drop's own unit sequencing.
