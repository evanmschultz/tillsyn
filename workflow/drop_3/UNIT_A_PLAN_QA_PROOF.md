# DROP_3 UNIT A — Plan QA Proof, Round 1

**Verdict:** fail
**Date:** 2026-05-02

## 1. Findings

### 1.1 [BLOCKING] Droplet 3.A.3 — fictional INSERT/UPDATE call sites at lines 1414, 1452, 2500

**Severity:** blocking — the builder will follow these citations and find no code matching the description, then either invent new sites or skip the work.

**Evidence:**
`/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/adapters/storage/sqlite/repo.go` contains exactly **one** `INSERT INTO action_items` (line 1253) and **one** column-bearing `UPDATE action_items` (line 1347 in `UpdateActionItem`). The only other `UPDATE action_items` writes are the `created_by_name` / `updated_by_name` backfill migrations at lines 767-768, which do not touch the column-write contract.

The plan's bullet at 3.A.3 says: *"Same pattern applied to the 2 other INSERT statements at lines 1414 and 1452 and the soft-create-or-update path at line 2500 — each touches the same `action_items` row writes."*

- Line 1414-1418 is the `SELECT … FROM action_items` column-list inside `ListActionItems`.
- Line 1452-1456 is the `SELECT … FROM action_items` column-list inside `ListActionItemsByParent`.
- Line 2500 is the `SELECT … FROM action_items` column-list inside `getActionItemByID`.

These are **SELECTs that need a new column added to the projected list** — a real and necessary change — but they are NOT additional INSERT statements, and there is no soft-create-or-update path. The plan misclassifies them as writes.

**Suggested fix:** Rewrite the bullet to read approximately:

> *"Sweep the SELECT column-lists at `ListActionItems` (line 1414), `ListActionItemsByParent` (line 1452), and `getActionItemByID` (line 2500) — each must include `structural_type` in the same ordinal position as the new column declaration so `scanActionItem` reads the value correctly. There is one INSERT (line 1253) and one UPDATE (line 1347), both already enumerated above."*

This collapses the misnamed "2 other INSERT statements + soft-create-or-update" into the correct SELECT-projection sweep, which the plan ALREADY captures correctly in the next bullet ("Sweep all `SELECT ... FROM action_items` and `RETURNING ...` clauses…"). The misstatement is duplicative *and* wrong.

### 1.2 [NIT] Droplet 3.A.1 — regex character-class precedent slightly misstates `role_test.go` mirror

**Severity:** nit — mostly cosmetic but the builder will be confused for a minute.

**Evidence:**
`internal/domain/role.go:52` uses regex `(?m)^Role:\s*([a-z0-9-]+)\s*$` (with digits, because `qa-a11y` carries `11`). `role.go:46-50` documents the widening explicitly. The plan's 3.A.1 acceptance correctly says structural_type uses `[a-z-]+` (no digits — accurate, since none of `drop|segment|confluence|droplet` carry digits) but also says the test file mirrors `role_test.go` "exactly … line-for-line." The "capitalized value fails to match (regex captures [a-z-]+ only)" case at `role_test.go:120` carries a comment that is itself stale relative to `role.go:52`'s `[a-z0-9-]+`. For `structural_type`, the comment text `[a-z-]+` would be accurate, but the builder should NOT propagate the stale comment if they read role_test.go first and copy mechanically.

**Suggested fix:** Add a one-line acceptance note: *"The new test's character-class comment text should reflect `[a-z-]+` accurately; do NOT copy `role_test.go:120`'s comment verbatim if it conflicts with the actual regex."*

### 1.3 [NIT] Droplet 3.A.2 — sweep claim is broad without an enumerated list

**Severity:** nit — the planner correctly identifies the structural risk but leaves the actual locations unbounded.

**Evidence:**
3.A.2 says: *"Sweep `domain_test.go` for every other `domain.NewActionItem(...)` / `NewActionItem(ActionItemInput{...})` call site that omits `StructuralType` — each must be updated to supply a value, or each test must explicitly exercise the new `ErrInvalidStructuralType` rejection path."* `internal/domain/domain_test.go` is 819 lines and has many `NewActionItem` call sites. The same risk applies in `internal/app/snapshot_test.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/app/service_test.go`, and likely others — every test that constructs an `ActionItem` via `NewActionItem` will fail validation once `StructuralType` is required on create.

**Suggested fix:** Either enumerate the sweep targets (recommend: list each `_test.go` file with a `NewActionItem` call grep count) or explicitly call out cross-package sweep scope: *"Cross-package sweep: every `_test.go` under `internal/` that calls `domain.NewActionItem` must supply a `StructuralType` value. The blast radius extends beyond `internal/domain` — 3.A.3 + 3.A.4 + 3.A.5 droplets will hit this when their own tests run."*

This isn't a structural plan failure (the cascade ordering correctly puts 3.A.2 before 3.A.3/.4/.5, so when downstream droplets break their tests, that's caught at builder time), but flagging it explicitly avoids surprise.

### 1.4 [NIT] Droplet 3.A.6 — section anchor for "between line 19 and line 36"

**Severity:** nit — citation is correct but needs to read between subsections, not headings.

**Evidence:**
Plan 3.A.6 says: *"New `## Cascade Vocabulary` section inserted between `## The Tillsyn Model (Node Types)` (line 19) and `## Level Addressing (0-Indexed)` (line 36)."* `WIKI.md` has subsections `### Do Not Use Other Kinds Today` (line 28) and `### Do Not Use Templates Right Now` (line 32) inside `## The Tillsyn Model`. The plan's "Architectural Questions" notes block correctly disambiguates this (recommends placement after line 34, before line 36) but the acceptance text on the droplet itself is ambiguous — a builder following only the acceptance could insert mid-section above `### Do Not Use…`.

**Suggested fix:** Update 3.A.6 acceptance to reference the disambiguated placement: *"Insert as a new `## Cascade Vocabulary` h2 immediately after the `### Do Not Use Templates Right Now` subsection (between line 34 and line 36) — i.e., the new section sits as a sibling top-level h2 between `## The Tillsyn Model (Node Types)` and `## Level Addressing (0-Indexed)`, NOT nested under the model section."*

### 1.5 [NIT] Pre-MVP no-migration: 3.A.3 acceptance correctly forbids ALTER TABLE but still cites the migration block

**Severity:** nit — the bullet correctly forbids the work but gestures at it in a way that could be misread.

**Evidence:**
3.A.3 acceptance bullet 1 reads: *"Pre-MVP rule (per `feedback_no_migration_logic_pre_mvp.md`): no `ALTER TABLE` migration; dev fresh-DBs the database. Do NOT add an `ALTER TABLE ... ADD COLUMN structural_type` statement under the existing migration-block pattern at lines 515/548."*

This honors the no-migration rule correctly. The cited lines 515 and 548 actually point at `attention_items` and `auth_requests` migration blocks respectively (the action_items migration block lives at lines 518-532). Builder might be confused. Suggest: cite line 518 (the start of `workItemAlterStatements`) instead.

**Suggested fix:** Replace `lines 515/548` with `line 518 (start of workItemAlterStatements)` for accuracy.

## 2. Missing Evidence

### 2.1 No explicit verification of 4-value enum closure

**Evidence Needed:** The plan correctly enumerates 4 values (`drop|segment|confluence|droplet`) per PLAN.md § 19.3 line 1629. Acceptance check passes. Confirmed in `workflow/drop_3/UNIT_A_PLAN.md:5` ("closed 4-value enum") and `workflow/drop_3/UNIT_A_PLAN.md:19` ("four typed constants").

### 2.2 No retroactive SQL classification — confirmed honored

**Evidence Needed:** PLAN.md § 19.3 line 1636 ("Retroactive classification of existing action_items via one-shot SQL") is the bullet that pre-MVP rules REPLACE per `feedback_no_migration_logic_pre_mvp.md`. The plan correctly notes this in `workflow/drop_3/UNIT_A_PLAN.md:178` ("REPLACED with the fresh-DB rule. No `ALTER TABLE`, no SQL backfill, no `till migrate` subcommand"). No fictional retroactive SQL slipped in.

### 2.3 Cross-unit dependency to Unit D agent file is correctly surfaced

**Evidence Needed:** `workflow/drop_3/UNIT_A_PLAN.md:171` carries the explicit conflict warning between 3.A.7 and Unit D's frontmatter sweep, with two resolution options (sequencing 3.A.7 → Unit_D_agent_file_droplet, or merging both edits). This satisfies the cross-unit-dependency surfacing requirement.

### 2.4 Drop 2.3-2.6 mirror pattern — correctly applied

**Evidence Needed:** Verified the cascade matches Drop 2.3's `Role` rollout:

- **Domain enum** (3.A.1 ↔ Drop 2.2 `internal/domain/role.go`): identical shape.
- **ActionItem field** (3.A.2 ↔ Drop 2.3 `Role` field placement at `action_item.go:33`): plan cites the right line.
- **SQLite column** (3.A.3 ↔ `role TEXT NOT NULL DEFAULT ''` at `repo.go:174`): correct placement and DDL pattern.
- **App / common / mcpapi plumbing** (3.A.4 ↔ Drop 2.5): same five files, same touch points.
- **Snapshot serialization** (3.A.5 ↔ Drop 2.6): same omitempty pattern, same toDomain shape.

The mirror is faithful. Divergence (`StructuralType` REQUIRED on create vs `Role` permits empty) is documented per PLAN.md § 19.3.

### 2.5 Blocked-by chain acyclicity verified

**Evidence Needed:** Walking the `blocked_by` graph for Unit A:
- 3.A.1 → none
- 3.A.2 → 3.A.1
- 3.A.3 → 3.A.2 (transitively 3.A.1)
- 3.A.4 → 3.A.2
- 3.A.5 → 3.A.2
- 3.A.6 → none
- 3.A.7 → none

Acyclic. Same-package serialization holds: 3.A.1 and 3.A.2 share `internal/domain` and 3.A.2 explicitly `blocked_by` 3.A.1. 3.A.3 (`internal/adapters/storage/sqlite`) is the only droplet in its package. 3.A.4 spans `internal/app` + `internal/adapters/server/common` + `internal/adapters/server/mcpapi`; 3.A.5 sits in `internal/app`. **3.A.4 and 3.A.5 share `internal/app`** — both edit different files (3.A.4: `service.go`; 3.A.5: `snapshot.go`) but the package compile-unit overlaps. Per CLAUDE.md "Blocker Semantics" rule: *"Sibling build-tasks sharing … a package in `packages` MUST have an explicit `blocked_by` between them."* Neither 3.A.4 → 3.A.5 nor 3.A.5 → 3.A.4 is asserted. **Same-package serialization gap.** Severity nit-to-blocking depending on whether the orchestrator wants strict serialization or accepts that file-level non-overlap is sufficient post-Drop-1 — pre-Drop-1, the rule is package-level.

**Suggested fix:** Add `3.A.5 blocked_by: [3.A.2, 3.A.4]` (or `3.A.4 blocked_by: [3.A.2, 3.A.5]` whichever is the chosen serialization) to honor the package-level rule. Alternatively, document the carve-out explicitly in the Notes section: *"3.A.4 and 3.A.5 share `internal/app` but edit non-overlapping files (`service.go` vs `snapshot.go`); orchestrator accepts file-level non-overlap as sufficient because no shared symbol crosses the two."*

This is logged as a separate finding below at 1.6.

### 1.6 [BLOCKING-OR-DOCUMENT] Same-package serialization gap between 3.A.4 and 3.A.5

**Severity:** blocking unless explicitly documented as an orchestrator-accepted exception.

**Evidence:**
- 3.A.4 `Packages` line: `internal/app, internal/adapters/server/common, internal/adapters/server/mcpapi`.
- 3.A.5 `Packages` line: `internal/app`.

Both droplets edit `internal/app/*.go` files (`service.go` and `snapshot.go` respectively). Per CLAUDE.md "Blocker Semantics": *"sibling build-tasks sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them."* The single Go package `internal/app` is the shared compile unit even though the two files don't overlap.

**Suggested fix:** Either (a) add `3.A.5 blocked_by: [3.A.2, 3.A.4]` (keeps 3.A.5 strictly serialized after 3.A.4), or (b) explicitly document in the plan's Notes: *"3.A.4 ↔ 3.A.5 same-package non-overlap exception accepted by orchestrator. Both touch `internal/app` but edit disjoint files (`service.go` vs `snapshot.go`) and do NOT share any symbol; concurrent build is safe because the package compiles cleanly with both edits applied independently."*

Recommend option (a) for Unit A — strict same-package serialization keeps the rule honest and the cost is one droplet's wait time on a small change.

## Verdict Summary

The plan correctly mirrors the Drop 2.3 `Role` cascade end-to-end across 7 droplets, with the architectural divergence (REQUIRED on create) faithfully captured per PLAN.md § 19.3. The 4-value closed enum, pre-MVP no-migration rule, cross-unit dependency to Unit D's frontmatter sweep, and the WIKI canonical-source-with-pointer pattern are all honored.

Two findings block PASS:

1. **3.A.3 contains fictional INSERT/UPDATE call sites at lines 1414, 1452, 2500** — those are SELECT projections, not writes. The bullet's claim of "2 other INSERT statements + soft-create-or-update path" is wrong on every count. The needed work (sweep SELECT column-lists) is correctly captured in the next bullet, so this is a duplicative-and-wrong restatement that will derail the builder.
2. **3.A.4 and 3.A.5 share `internal/app` without an explicit `blocked_by`** — same-package serialization rule violated unless the orchestrator documents the carve-out.

Three nits should be cleaned up before builder spin-up: 3.A.1's regex-comment mirror caveat, 3.A.2's sweep-scope enumeration, and 3.A.6's WIKI-section placement disambiguation. 3.A.5 line citation for the migration block is also slightly off (line 518 not 515/548).

All other line citations spot-checked at HEAD verify cleanly: `action_item.go:33` (Role field), `action_item.go:150-158` (Role validation block), `action_item.go:201` (Role return assignment), `repo.go:174` (role column DDL), `repo.go:1252-1257` (INSERT), `repo.go:1346-1351` (UPDATE), `repo.go:2790` (scanActionItem), `repo.go:2846` (Role assignment in scan), `service.go:404` (CreateActionItemInput), `service.go:429` (UpdateActionItemInput), `service.go:574-595` (NewActionItem call), `service.go:784-794` (Role update block), `mcp_surface.go:57+78` (request structs), `app_service_adapter_mcp.go:666-684+708-720` (adapter literals), `extended_tools.go:860-866+1033+1092+1370+1416+1443` (handler + tool schema + legacy aliases), `extended_tools_test.go:429+455` (stub rejection logic), `snapshot.go:57+63+1060+1067+1267+1273-1278` (snapshot struct/round-trip/legacy fallback), `snapshot_test.go:442` (round-trip test), `domain_test.go:212` (Role validation test), `role_test.go` (mirror source), `role.go:52` (regex), `errors.go:28` (ErrInvalidRole), `WIKI.md:19+28+32+34+36` (insertion target), `go-qa-falsification-agent.md:95-108` (attack section).

Verdict: **fail**, recoverable with two fixes plus three nits. Re-spawn builder once the planner addresses 1.1 and 1.6.
