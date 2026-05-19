# DROP_4B_TEST_CLEANUP — Drop 4b Deferred Refinements (R5/R6/R7/R8)

**State:** planning
**Blocked by:** —
**Paths (expected):** test files under `internal/app/dispatcher/` (R6/R7 D5 e2e tests), `internal/app/comments.go` + adjacent (R5 `till.comment` target_type fix), `internal/app/action_items.go` + MCP adapter layer (R8 supersede MCP operation). Planner narrows per droplet.
**Packages (expected):** `internal/app`, `internal/app/dispatcher`, `internal/adapters/server/mcpapi`. Planner sets per-droplet packages.
**PLAN.md ref:** Drop 4b deferred-refinement absorption — itemized in `REVISION_BRIEF.md`
**Workflow:** `workflow/example/drops/WORKFLOW.md`
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`
**Started:** 2026-05-18
**Closed:** —

## Scope

Land Drop 4b's deferred refinements R5/R6/R7/R8 from `project_drop_4b_refinements_raised.md` — name shorthand "test cleanup" because R6/R7 dominate the LOC count, but the drop also covers two coordination-surface MCP fixes (R5, R8). R1/R2/R3/R4 stay parked for a separate template-validation hardening drop later.

See `REVISION_BRIEF.md` for the four refinement breakdowns with file/line pointers, fix paths, and acceptance criteria.

## Planner

<Filled by `go-planning-agent` in Phase 1. Read `REVISION_BRIEF.md` first.>

## Notes

<Filled by planner if useful.>
