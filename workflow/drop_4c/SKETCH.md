# Drop 4c — Pre-Drop-5 Refinement Drop (Sketch)

**Status:** placeholder — NOT a full plan. Full PLAN.md authoring + parallel-planner dispatch + plan-QA twins land post-Drop-4b-merge before any builder fires.
**Author date:** 2026-05-03 (during Drop 4a planning phase).
**Purpose:** capture deferred-from-4a-and-4b items so nothing gets lost when planning starts post-4b.

## Naming

**Drop 4c** = the third drop in the 4-series. Drop 4a (dispatcher core) → Drop 4b (gate execution) → Drop 4c (pre-Drop-5 polish + audit-debt sweep) → Drop 4.5 (TUI overhaul, concurrent with Drop 5) + Drop 5 (dogfood validation).

Descriptive name: "pre-Drop-5 refinement drop." Functional name: "Drop 4c."

## Goal

Bundle the deferred polish + agent-facing hardening items that surfaced during the audit/build of Drops 4a + 4b but were explicitly out-of-scope for those drops. Lands BEFORE Drop 5 dogfooding so the cascade-on-itself loop isn't fighting silent-data-loss bugs or other agent-surface warts. Does NOT block Drop 4.5 (concurrent FE/TUI track).

End-state: cascade-on-itself loop has no known agent-surface footguns; pre-MVP audit gaps closed; dev-hygiene tooling complete.

## In-Scope Items (Captured From Audits)

The list grows as Drops 4a + 4b run. Initial seed items:

### From the Drop 1 audit (PLAN.md §19.1 items still missing post-Drop-4a-and-4b)

- **PATCH semantics on update handlers.** `Service.UpdateActionItem` calls `actionItem.UpdateDetails(...)` which writes every field unconditionally — `update(title="foo")` with empty `description` wipes the stored description. Silent-data-loss bug confirmed in pre-Drop-3 18.10B closeout. Fix: pointer-sentinel input fields (or explicit-replace-all flag) so omitted fields preserve.
- **Reject unknown keys at MCP boundary.** No `DisallowUnknownFields` across MCP server code. Schema-permissive create/update silently drops unknown fields. Surfaced in pre-Drop-3 audits. Fix: per-MCP-tool unknown-key rejection with structured error naming the offending key.
- **Server-infer `client_type` on auth request create.** Currently accepts empty `client_type`; only the approve-path rejects it (asymmetric validation bug). Fix: MCP-stdio adapter stamps `"mcp-stdio"`, TUI stamps `"tui"`, CLI stamps `"cli"`. Tighten `app.Service.CreateAuthRequest` to reject empty `ClientType`.
- **Supersede CLI**: `till action_item supersede <id> --reason "..."` — marks `failed` action item as `metadata.outcome: "superseded"` and transitions `failed → complete`. Bypasses the always-on parent-blocks invariant landed in Drop 4a Wave 1.7. Hard requirement before dogfood — without it, every `failed` child stuck-state requires dev fresh-DB.
- **CLI failure listing**: `till action_item list --state failed` (or `till failures list`). Dev visibility into `failed` action items pre-TUI-rendering (Drop 4.5).
- **Require non-empty outcome on `failed`.** `validateMetadataOutcome` accepts empty outcome. Domain-level validation: any transition to `failed` requires non-empty `metadata.outcome`.
- **`go.mod` `replace` directive cleanup.** Strip every `replace` directive except the fantasy-fork. Pre-cascade hygiene from PLAN.md §19.1 first bullet.

### From Drop 3 refinements memory (`project_drop_3_refinements_raised.md`)

- **R1 — STEWARD field-level guard symmetry on `Persistent` / `DevGated`.** `assertOwnerStateGateUpdateFields` rejects agent-principal Update calls that mutate Owner / DropNumber on STEWARD-owned items, but does NOT reject Persistent / DevGated mutations. Fix: extend to all four fields per L13 (domain primitives, not STEWARD-specific).
- **R2 — `raiseRefinementsGateForgottenAttention` doc-comment idempotency drift.** Helper godoc claims idempotent duplicate-insert handling that the implementation doesn't perform. Fix: either prepend `FindAttentionItemByExternalID`-style lookup OR trim the godoc claim.
- **R3 — `isRefinementsGate` future-cascade precision.** Today the predicate keys on `Owner=STEWARD + StructuralType=Confluence + DropNumber>0`; if a future cascade variant introduces a second STEWARD-owned numbered confluence kind, false-positive risk. Fix: tighten predicate with `Title` shape match OR add `Kind` discriminator.
- **R5 — WIKI Cross-Subtree Exception kind-choice hedge.** WIKI says `kind=closeout or kind=plan ... as appropriate` for ledger / wiki-changelog rollups — under-specified pending STEWARD's actual rollup-kind choices stabilizing. Fix: re-survey post-Drop-4 STEWARD usage; pick canonical rollup kinds.

### From Drop 4a/4b refinements memory (TBD, populated during builds)

- Any plan-QA-falsification PASS-WITH-NIT findings that don't get fixed in their drop.
- Any LSP / gopls-vet hints that surface during builds (mirrors Drop 3 R4 pattern).
- Any audit-gap accept items from outside-repo edits in 4a.32 + 4b's equivalent.
- Open questions Q1–Q12 from Drop 4a's PLAN.md §10 that resolve as "accepted as-is, document for future revisit" — those documented decisions belong here as candidate fixes if the dogfood window flags them.

## Out of Scope

- Anything that lands in Drops 4a or 4b before Drop 4c starts. Drop 4c is the deferred-residue, not duplicate work.
- Drop 4.5 scope (TUI overhaul, columns-table retirement, file-viewer pane).
- Drop 5 scope (dogfood validation).
- Migration logic — pre-MVP rule still in force.

## Tentative Wave / Item Structure

~10–12 droplets. Loose theme grouping (subject to revision):

### Theme A — Silent-data-loss + agent-surface hardening (~4 droplets)

- PATCH semantics on update handlers (per item ).
- Reject unknown keys at MCP boundary.
- Server-infer `client_type` on auth-request create.
- Require non-empty outcome on `failed` transitions.

### Theme B — Dev-facing escape hatches (~2 droplets)

- Supersede CLI (`till action_item supersede`).
- CLI failure listing (`till action_item list --state failed`).

### Theme C — STEWARD + cascade-precision refinements (~3 droplets)

- R1: extend `assertOwnerStateGateUpdateFields` to Persistent + DevGated.
- R2: tighten `raiseRefinementsGateForgottenAttention` doc-comment vs idempotency.
- R3: tighten `isRefinementsGate` predicate.
- R5: re-survey WIKI Cross-Subtree Exception kind-choice (may be MD-only sweep).

### Theme D — Pre-cascade hygiene (~1–2 droplets)

- `go.mod` `replace` directive cleanup.
- Any LSP / gopls-vet hints accumulated through 4a + 4b builds.

### Theme E — Drop-4a/4b-residue (TBD)

- Populated during 4a + 4b builds. Whatever surfaces in `project_drop_4a_refinements_raised.md` and the 4b equivalent that doesn't fit in those drops' scopes lands here.

## Pre-MVP Rules (carried forward)

- No migration logic in Go; dev fresh-DBs.
- No closeout MD rollups.
- Opus builders.
- Filesystem-MD mode (or Tillsyn-runtime if Drops 4b's auto-promotion has matured the runtime by then).
- Single-line commits.
- NEVER raw `go test` / `mage install`.

## Open Questions To Resolve At Full-Planning Time

- **Q1 — Pre-MVP rule transition.** Drops 4a + 4b plus dogfood-prep may justify flipping some pre-MVP rules. Specifically: `feedback_no_closeout_md_pre_dogfood.md` (skip rollups) — does Drop 4c want to start writing real LEDGER / REFINEMENTS entries to dogfood the rollup loop?
- **Q2 — Theme E sizing.** How many residue items will surface from 4a + 4b? Could be 0 (clean sweep) or 5+ (significant findings). Final droplet count drifts with this.
- **Q3 — Supersede CLI scope.** Just the basic `till action_item supersede` or also `till action_item list --state failed`? Possibly bundled in same droplet.
- **Q4 — Drop 4.5 coupling.** TUI overhaul (4.5) would benefit from CLI failure listing landing first (so 4.5 has the data layer to render). Soft sequencing — does Drop 4c block Drop 4.5, or run in parallel?
- **Q5 — Drop 5 readiness gate.** Drop 5 is "dogfood validation" — at what point during Drop 4c does Drop 5 become startable? After Theme A + B (silent-data-loss + escape hatches) but before C/D? Or wait for the full 4c close?

## Approximate Size

~10–12 droplets. Smaller than 4b. Most items are 1–3 file edits each (audit-finding fixes are typically narrow). Full planning at post-4b-merge time will refine the count + the Theme E residue list.

## Hard Prerequisites

- Drop 4a closes (dispatcher exists; can the Drop 4c items be dispatched-built or do they require manual orch builds? Decision deferred to planning time).
- Drop 4b closes (gate execution + commit pipeline exists; Drop 4c items benefit from the pipeline once they're dispatcher-eligible).
- `project_drop_4a_refinements_raised.md` exists and is populated with mid-build findings.
- `project_drop_4b_refinements_raised.md` (TBD) exists and is populated.

## Workflow Cross-References

- `workflow/drop_4a/PLAN.md` — Drop 4a's plan (deferrals from 4a end up here).
- `workflow/drop_4b/SKETCH.md` — Drop 4b's sketch (deferrals from 4b end up here).
- `project_drop_3_refinements_raised.md` — R1, R2, R3, R5 source.
- PLAN.md §19.1 — original Drop 1 audit-list source for items 1–7.
- Memory `feedback_no_closeout_md_pre_dogfood.md` — pre-MVP rules that may transition during this drop.

## Open Tasks Before Full Planning

1. Drop 4a closes; Drop 4b closes (in sequence).
2. Drop 4a + 4b refinements memories reviewed; Theme E populated with everything that didn't make it into the 4a/4b drops.
3. Drop 5 readiness gate decision: does Drop 5 start mid-Drop-4c (after Theme A + B), or wait for full Drop 4c close?
4. Full REVISION_BRIEF authored, parallel-planner dispatch (likely 4 theme planners, one per Theme A–D), unified PLAN.md synthesis, plan-QA twins → green → builder dispatch.

## Anti-Goals

- **Not a "fix everything" drop.** Drop 4c is bounded by the MVP-feature-complete gate. Items that aren't blocking Drop 5 dogfood readiness stay deferred to a later refinement.
- **Not a refactor drop.** Each item is a narrow fix on top of existing primitives. No package reorganizations, no API rewrites.
- **Not a TUI work.** TUI overhaul is Drop 4.5, runs concurrent with Drop 5.
