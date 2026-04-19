# DROP_N — <NAME>

**State:** planning
**Blocked by:** <DROP_M> | —
**Paths (expected):** <broad file footprint at start; refined by planner into per-droplet paths>
**Packages (expected):** <broad package footprint; refined by planner into per-droplet packages>
**PLAN.md ref:** `<PROJECT>/PLAN.md` → `DROP_N_<NAME>` row
**Workflow:** `drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** YYYY-MM-DD
**Closed:** —

## Scope

<One paragraph. Lifted from the project-root `PLAN.md` container row + dev confirmation during Phase 1. Describes the what and why of this drop at the container level. Droplet-level detail belongs in the Planner section.>

## Planner

<Filled by the planner subagent in Phase 1. The planner either emits droplets directly (leaf drop) or emits sub-drop container rows (cascade recursion — each sub-drop gets its own `DROP_N.M_<NAME>/` directory and its own planner run). Each droplet's `state` is mutated in place by the builder during Phase 4. See `drops/WORKFLOW.md` § "Phase 1 — Plan" for deliverable rules.>

### Droplet N.1 — <title>

- **State:** todo
- **Paths:** <exact files this droplet writes to>
- **Packages:** <packages this droplet compiles under — droplets sharing a package MUST have explicit `blocked_by`>
- **Acceptance:** <yes/no-verifiable criteria a QA subagent can check against the code + test output>
- **Blocked by:** —

### Droplet N.2 — <title>

- **State:** todo
- **Paths:**
- **Packages:**
- **Acceptance:**
- **Blocked by:** N.1

<…repeat per droplet…>

<Alternatively, when this drop is decomposed into sub-drops rather than droplets:>

### Sub-drop N.1 — <title>

- **State:** planning
- **Directory:** `drops/DROP_N.1_<NAME>/`
- **Scope:** <one line>
- **Blocked by:** —

<…repeat per sub-drop…>

## Notes

<Optional. Cross-droplet decisions, library choices made during planning, deferrals to later drops, explicit YAGNI rulings, anything the planner wants reviewers to know but that doesn't belong inside a droplet row.>
