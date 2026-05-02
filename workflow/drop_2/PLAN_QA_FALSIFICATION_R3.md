# DROP_2 — Plan QA Falsification, Round 3

**Verdict:** fail
**Date:** 2026-05-01

This Round 3 falsification attacks the surgical-fix patches the orchestrator applied (R3-1..R3-8) against Round 2's 4 PROOF + 5 FALSIFICATION blockers, and the new project-less dotted-address spec at PLAN.md `:285-293`. Round 3 closes the major Round 2 findings — fakeRepo extension under 2.10, 2.5+2.7 same-package race, MCP description rewrite, dotted-address `N.M.K` semantics, `kind_capability_test.go:73` reader, cite drift sweep. **But two new blocking counterexamples surfaced**: one structural (the `position ASC` conditional in R3-8 is unsafe because the existing `position` column is column-scoped, not parent-scoped — semantic mismatch, not just an absence question), one cite-completeness (Round 3 fixed the `Done: false` literal at `internal/app/kind_capability_test.go:429` per R3-1 but missed the identical `Done: false` literal at `internal/domain/kind_capability_test.go:19`). Plus three carry-over nits from Round 2.

Severity ladder (highest first):
- **F1** — R3-8's `position ASC` conditional fallback is unsafe. The `position` column on `action_items` exists but is **column-scoped (`idx_action_items_project_column_position` keys on `(project_id, column_id, position)`)**, not parent-scoped. A builder following the conditional literally produces a resolver where dotted addresses are unstable across user-driven column reordering.
- **F2** — `internal/domain/kind_capability_test.go:19` carries `Done: false` (a `ChecklistItem.Done` field literal) and is missed by the Round 3 cite list at PLAN.md `:163`. Same-package compile-break shape as Round 2 R2-P1 (`internal/app/kind_capability_test.go:429`) which Round 3 R3-1 fixed only for the `internal/app` sibling.
- **F3** — Round 3 PROOF's PASS verdict is contradicted by F1 and F2. PROOF accepted the `position ASC` conditional at face value without checking the column's actual schema semantics, and didn't sweep the `Done: false` literal pattern across packages.

Three nits below — `service.go:1965-1975` cite range over-claim (R2-F7 carry-over), `mcp_surface.go:227 Completed` rationale gloss (R2-F9 carry-over), and a missing tie-break test case in 2.10's enumerated test list.

---

## 1. Round 2 Blocker Resolution Summary

| R2 # | R2 Title (paraphrased)                                                  | R3 Status              | Notes                                                                                          |
| ---- | ----------------------------------------------------------------------- | ---------------------- | ---------------------------------------------------------------------------------------------- |
| R2-P1 | `internal/app/kind_capability_test.go` missing from 2.7 Paths          | resolved               | PLAN.md `:164` adds the file with explicit `:429` `Done: false → Complete: false` cite.       |
| R2-P2 | `service_test.go:1561, 1567, 2953` legacy literals not enumerated       | resolved               | PLAN.md `:168` lists all three explicitly.                                                     |
| R2-P3 | 2.7 missing `Blocked by: 2.5` (same-file edit on `app_service_adapter_mcp.go`) | resolved        | PLAN.md `:229` reads `Blocked by: 2.5, 2.6` with R3-3 commentary.                              |
| R2-P4 | `domain_test.go:114` mis-classified as state-vocab                      | resolved               | PLAN.md `:162` carve-out: column-name free-form, NOT lifecycle state.                          |
| R2-P5 | Cite drift `:3039 → :3040`, `:3797` over-claim, unexported `normalizeLifecycleState` | resolved        | All three corrections applied.                                                                  |
| R2-P6 | `Done: true|false` field-literal acceptance grep missing                 | resolved               | PLAN.md `:211` adds `\.Done\s*=\s*(true|false)|Done:\s*(true|false)` grep.                     |
| R2-F1 | Droplet 2.10 `fakeRepo` extension missing from `Paths:`                  | resolved               | PLAN.md `:281` adds `internal/app/service_test.go (extend fakeRepo to implement the new method)`. |
| R2-F2 | Same-package race 2.5 + 2.7 (`app_service_adapter_mcp.go`)               | resolved               | See R2-P3 above.                                                                                |
| R2-F3 | 2.7 misses `extended_tools.go:1339` MCP tool description                 | resolved               | PLAN.md `:184` adds the file with explicit `:1339` rewrite cite.                                |
| R2-F4 | Dotted-address `N.M.K` semantics under-specified                          | partially-resolved     | R3-8 commits to project-less body, 0-indexed, `created_at ASC, id ASC` tie-break, no ambiguity error — clean. **But the `position ASC` conditional fallback is unsafe: see Finding 2.1 below.** |
| R2-F5 | `kind_capability_test.go:73` reader for `RequireChildrenDone`             | resolved               | PLAN.md `:163` extends cite from `:35` to `:35, :73`.                                           |
| R2-F6 | Cite drift `service.go:556, :694` (StateTodo, not renamable)              | resolved               | PLAN.md `:167` adds `NOT in scope` carve-out for `:556, :694`.                                  |
| R2-F7 | Cite drift `service.go:1965-1975` half-correct                            | partially-resolved     | Range still over-claims. PROOF downgraded to nit (acceptance grep catches it). See Nit 2.4 below. |
| R2-F8 | `IsValidLifecycleState` exported-vs-unexported casing                     | resolved               | Corrected to `isValidLifecycleState` at `:166`.                                                 |
| R2-F9 | `mcp_surface.go:227 Completed` rationale wording                          | partially-resolved     | Rationale gloss still inaccurate — see Nit 2.6 below. Conclusion (no rename) correct.            |
| R2-F10 | `snapshot.go:421` error-message text not enumerated                       | downgrade-acceptable   | Acceptance grep `lifecycle_state.*"done"|lifecycle_state.*"progress"` catches it.               |
| R2-F11 | `capture_test.go:199` debug-message field labels                          | unresolved (carry-over) | Debug-only; PROOF downgraded; acceptable per the orchestrator's adjacent-fix carve-out.         |

10 of 11 R2 FALSIFICATION findings + 7 of 7 R2 PROOF findings are resolved or downgrade-acceptable. R2-F4 is **partially-resolved** — the high-level spec is sound but the position-column conditional is broken (Finding 2.1).

---

## 2. Round 3 Findings

### 2.1 R3-8's `position ASC` conditional is unsafe — `position` column on `action_items` is column-scoped, not parent-scoped

- **Severity:** blocking
- **Attack vector:** 3 (`position` column existence + semantics).
- **Counterexample evidence (file:line, verified at HEAD):**
  - `internal/adapters/storage/sqlite/repo.go:176` — `position INTEGER NOT NULL` is on the `action_items` `CREATE TABLE`. Column DOES exist.
  - `internal/adapters/storage/sqlite/repo.go:487` — `CREATE INDEX IF NOT EXISTS idx_action_items_project_column_position ON action_items(project_id, column_id, position)`. Index keys on `(project_id, column_id, position)` — **per-column**, not per-parent.
  - `internal/adapters/storage/sqlite/repo.go:488` — `CREATE INDEX IF NOT EXISTS idx_action_items_project_parent ON action_items(project_id, parent_id)`. Parent-lookup index has no position component.
  - `internal/adapters/storage/sqlite/repo.go:1404` — existing `ListActionItems` orders `ORDER BY column_id ASC, position ASC`. Confirms position semantics are per-column-arrangement, not per-parent-stable-index.
  - `internal/domain/action_item.go:32` — `Position int` field on the `ActionItem` struct (per-row scalar; the column-scoping comes from the index + ORDER BY conventions, not the field itself).
  - PLAN.md `:287` — *"`Repository.ListActionItemsByParent(ctx, projectID, parentID)` ... deterministically ordered by `created_at ASC, id ASC` (or `position ASC, created_at ASC, id ASC` if a `position` column exists on `action_items` — quick schema check at build time, prefer position if available)."*

- **Why this is blocking.** A builder following the conditional literally will:
  1. Run a schema check (the column exists), prefer `position`.
  2. Write `ORDER BY position ASC, created_at ASC, id ASC` for `ListActionItemsByParent`.
  3. Get **incorrect dotted addresses** when:
     - Two siblings of one parent live in different columns (parent A: child X in `todo` at position 0, child Y in `in_progress` at position 0). Both rows have `position = 0`. The fallback `created_at ASC, id ASC` resolves the tie — but the resolver's *meaning* becomes "Nth child by column-arrangement order," not "Nth child by stable parent-ordering." If the dev later moves Y to position 3 within `in_progress`, X's dotted address shifts even though X never moved.
     - The user creates child X at position 5 in column `todo`, then drag-reorders to position 2. X's `position` on disk is now `2`. Other children of the same parent live at positions 0, 1, 3 in `todo`. The dotted address `parent_idx.2` now resolves to whichever child has `position = 2` — which is X by virtue of a UI gesture, not by its place in the children-listing.
  4. **The mental model dotted addresses convey is "stable, position-by-creation-or-explicit-order, parent-scoped indexing."** R3-8 says "0-indexed positions among parent's children at each level." A builder reading the conditional sees the SQL `position` column and reasonably conflates the two meanings of "position." This is a *correctness bug*, not a compile-break — `mage ci` will be green.

- **Compounding lexical hazard.** R3-8's prose at PLAN.md `:285` says *"0-indexed positions among parent's children at each level."* The next sentence at PLAN.md `:287` says *"if a position column exists ... prefer position if available."* Same word, two meanings, adjacent sentences. **A builder will conflate them.**

- **Suggested mitigation.** Drop the `position ASC` conditional fallback entirely. PLAN.md `:287` should commit to `ORDER BY created_at ASC, id ASC` unconditionally:

  > *"`Repository.ListActionItemsByParent(ctx, projectID, parentID)` returns the list of action items whose `ParentID == parentID` within `projectID`, **deterministically ordered by `created_at ASC, id ASC`**. The dotted-address index is the 0-indexed slot in this ordering, NOT the value of any `position` column on the `action_items` table — the existing `position` column is column-scoped (per `idx_action_items_project_column_position`), changes when the user reorders within a column, and would make dotted addresses unstable across user gestures."*

  Add an acceptance bullet:

  > *"`ListActionItemsByParent` SQL must be `SELECT ... FROM action_items WHERE project_id = ? AND parent_id = ? AND archived_at IS NULL ORDER BY created_at ASC, id ASC`. **Do NOT use `position ASC`.** Test fixture: parent P with children X (column `todo`, position 0, created at t=0) and Y (column `in_progress`, position 0, created at t=1) — `0` resolves to X, `1` resolves to Y, regardless of column or position."*

  And rename one of the two "position" usages so they don't shadow each other.

---

### 2.2 Round 3 missed `internal/domain/kind_capability_test.go:19` for `ChecklistItem.Done → Complete` rename

- **Severity:** blocking
- **Attack vector:** 2 (literal/symbol leak), 15 (JSON-tag rename completeness).
- **Counterexample evidence (file:line, verified at HEAD):**
  - `internal/domain/kind_capability_test.go:19` — `CompletionChecklist: []ChecklistItem{{ID: "c1", Text: "run tests", Done: false}}`. A `ChecklistItem.Done bool` field literal that compile-breaks after 2.7 renames the field to `Complete bool`.
  - PLAN.md `:163` — *"`internal/domain/kind_capability_test.go` — rename `RequireChildrenDone:` test fixtures at `:35` AND `:73` to `RequireChildrenComplete:`."* Plan's per-file enumeration mentions only `:35, :73` (the `RequireChildrenDone` field), NOT `:19` (the `Done: false` field literal).
  - This is the EXACT shape Round 2 R2-P1 caught for the sibling file `internal/app/kind_capability_test.go:429`. Round 3 R3-1 fixed the `internal/app` site but did not extend the same fix to `internal/domain`.
  - Verified via `git grep -nE "Done: (true|false)" -- 'internal/domain/'`: `:19` and `:275, 421, 422, 425, 428, 536` in `domain_test.go` (latter all in 2.7 cite range `:420-442, :536`). Only the `internal/domain/kind_capability_test.go:19` site is missed.

- **Why this is blocking.** Compile-break inside `internal/domain` after 2.7 renames `ChecklistItem.Done bool → ChecklistItem.Complete bool`. `mage test-pkg ./internal/domain` fails with `unknown field Done in struct literal of type domain.ChecklistItem`. The unit-boundary `mage ci` gate catches it, AND the acceptance grep `git grep -nE 'Done:\s*(true|false)' -- '*.go'` from PLAN.md `:211` would catch it post-rename — so a careful builder running both gates will fix it. But the plan's `Paths:` per-file enumeration is the file-locking contract and the builder's surgical-edit guide. Missing this site under the explicit cite list contradicts Round 3's "surgical drift cleanly closed" claim.

- **Suggested mitigation.** Extend PLAN.md `:163` to: *"`internal/domain/kind_capability_test.go` — rename `Done: false` field literal at `:19` to `Complete: false`; rename `RequireChildrenDone:` test fixtures at `:35` AND `:73` to `RequireChildrenComplete:`."*

---

### 2.3 PROOF Round 3's PASS verdict is contradicted by F1 and F2

- **Severity:** blocking
- **Attack vector:** 17 (attack PROOF Round 3's PASS verdict).
- **Counterexample evidence:**
  - `PLAN_QA_PROOF_R3.md:55` — *"`ORDER BY created_at ASC, id ASC` (or `position ASC, created_at ASC, id ASC` if a `position` column exists): the UUID `id` tie-breaker renders ambiguity unreachable, validating the `ErrDottedAddressAmbiguous` removal at `:292`."* PROOF accepted the `position ASC` conditional at face value without checking the existing `position` column's *semantics* — only its *existence*. Falsification §2.1 above shows the column exists but has wrong semantics for parent-scoped indexing.
  - `PLAN_QA_PROOF_R3.md:14` — PROOF's R2-P1 resolution check verified `internal/app/kind_capability_test.go:429`. PROOF did not sweep for sibling-package occurrences of the same `Done: false` pattern. The `internal/domain/kind_capability_test.go:19` site (Falsification §2.2 above) was missed.
  - PROOF's verdict at `PLAN_QA_PROOF_R3.md:88-99` claims "0 new R3 blockers." With Falsification §2.1 + §2.2 above, the count is at least 2 new R3 blockers.

- **Why this is blocking.** PROOF's PASS verdict is the gate that flips Drop 2 to "build phase." With §2.1 unaddressed, builders enter the round with an unsafe optional ORDER BY. With §2.2 unaddressed, builders may make a surgical edit pass that misses a compile-break which only the unit-boundary `mage ci` will catch — burning a build round on an avoidable miss.

- **Suggested mitigation.** Fixing §2.1 + §2.2 above naturally addresses PROOF's blind spot. PROOF's evidence-coverage methodology should also flag any *conditional* in plan prose ("if X, then Y, else Z") for explicit *X-existence-AND-X-semantics* verification, not just X-existence.

---

### 2.4 (nit) `service.go:1965-1975` cite range still over-claims state-rename sites

- **Severity:** nit (R2-F7 carry-over, PROOF downgraded; flagged for builder ergonomics).
- **Attack vector:** 9 (`isValidLifecycleState` casing) + general planner accuracy.
- **Counterexample evidence:**
  - PLAN.md `:167` — *"rename `domain.StateDone`/`StateProgress` symbol references at `:623, 627, 639, 644, 1817, 1965-1975`"*.
  - Verified at HEAD via `git grep -n "StateProgress\|StateDone" -- 'internal/app/service.go'`: only `:1967, 1969` in the `:1965-1975` range carry renamable symbols. `:1965, 1971-1975` are `StateTodo`/`StateFailed`/`StateArchived` (NOT renamable).
- **Why nit.** Acceptance grep `git grep -nE 'domain\.StateDone|domain\.StateProgress'` returns empty post-rename, so the over-claim doesn't cause a missed site or unnecessary edit. A builder reading the cite range literally inspects lines that have nothing to rename — minor lost time, no correctness issue.
- **Suggested mitigation.** Tighten cite to `:1967, 1969` only. Or replace `:1965-1975` with `:1965-1975 (only :1967, :1969 carry renamable symbols)`.

---

### 2.5 (nit) R3-8's acceptance test list does not enumerate the deterministic-tie-break test case

- **Severity:** nit (R2 falsification §2.4 carry-over partially-addressed by R3-8 by-construction).
- **Attack vector:** 3 (dotted-address spec test enumeration).
- **Counterexample evidence:**
  - PLAN.md `:293` lists tests: *"valid `0`, valid `0.0`, valid `2.5.1`, slug-prefixed valid, slug-prefix mismatch, out-of-range, malformed inputs (empty, `1.`, `.1`, `1..2`, `abc`, leading-dash, deep nesting), UUID input rejected ..."*
  - R3-8 made tie-resolution unreachable-by-construction (UUID `id` is the final tie-breaker; UUID lex-ordering is total). The test isn't load-bearing for correctness — but a fixture with two children sharing identical `created_at` (manually constructed test fixture, e.g., `now := time.Date(...); fixture(now); fixture(now)`) would EXERCISE the `id`-tie-break path and lock in determinism.
- **Why nit.** Absent the test, the deterministic ordering still works at runtime. A future engineer who drops the `id` tie-breaker would silently re-introduce ambiguity, which the test would catch.
- **Suggested mitigation.** Add to PLAN.md `:293` test list: *"two children with identical `created_at` (test fixture forces equality) — assert deterministic resolution by `id` tie-break."*

---

### 2.6 (nit) `mcp_surface.go:227 Completed` rationale gloss still inaccurate

- **Severity:** nit (R2-F9 carry-over, PROOF downgrade-acceptable).
- **Attack vector:** general planner-prose-accuracy.
- **Counterexample evidence:**
  - PLAN.md `:391` reads *"`mcp_surface.go:227 Completed bool json:"completed"` is an MCP-response checklist-completion boolean on a different struct from `ChecklistItem`."*
  - Verified at HEAD: the field is on `ReindexEmbeddingsResult` (an embeddings-reindex job-status struct, sibling to `ScannedCount`, `QueuedCount`, `FailedCount`, `RunningCount`, `PendingCount`, `Completed`, `TimedOut`). NOT a "checklist-completion boolean."
- **Why nit.** The conclusion (no rename, no acceptance touches the field) is correct. Rationale gloss is wrong but harmless to the build.
- **Suggested mitigation.** Rewrite to: *"`mcp_surface.go:227 Completed bool json:"completed"` lives on `ReindexEmbeddingsResult` (embeddings-reindex job status response shape). Independent of `ChecklistItem.Done` and of `LifecycleState`. NO rename, NO acceptance criterion in Drop 2 touches this field."*

---

## 3. Acyclic + Same-Package Re-Verification (Round 3 incremental)

- **Cycle check (DAG).** Edges: 2.1→ε, 2.2→ε, 2.3→2.2, 2.4→2.3, 2.5→2.4, 2.6→2.3, 2.7→{2.5, 2.6}, 2.8→2.7, 2.9→2.8, 2.10→2.9, 2.11→2.10. No cycles. ✓
- **2.5 + 2.7 same-package edit on `app_service_adapter_mcp.go`:** Round 3 added `2.5` to 2.7's `Blocked by:` line (PLAN.md `:229`). Race closed. ✓
- **2.6 + 2.7 same-file edits on `internal/app/snapshot.go`:** 2.6 edits `SnapshotActionItem` struct at `:57` + `snapshotActionItemFromDomain` at `:1057` + `toDomain` at `:1263`. 2.7 edits the validation switch at `:419`. Different sections of the same file, but same Go file. Same-package serialization is enforced (2.7 → 2.6 via `Blocked by`). ✓
- **2.7 → {2.5, 2.6}, both transitively pull 2.3:** No transitive race or cycle.
- **Conclusion.** Same-package serialization is clean. Round 2's 2.5 + 2.7 race is closed.

---

## 4. JSON-Tag Aggregate Counter Sweep (Round 3 re-verify)

- `git grep -n "DoneActionItems\|DoneItems\|complete_items\|complete_tasks" -- '*.go'` shows only the WorkOverview / AttentionWorkOverview members at the sites Plan 2.7 enumerates. No new sites. ✓
- `internal/domain/workitem.go:129 DefinitionOfDone string json:"definition_of_done"` — semantic field ("Definition of Done" is a Scrum/agile idiom independent of the lifecycle `done` state). NOT touched by 2.7. **Verified PASS.** Could optionally be documented as an explicit out-of-scope carve-out for completeness.
- `internal/app/snapshot.go:80 CompletedAt *time.Time json:"completed_at,omitempty"` — already canonical (`Completed`, not `Done`); no rename. ✓

---

## 5. Slug-Prefix Syntax Conflict Sweep

- **Slug character set verified at HEAD:** `internal/domain/project.go:154-178 normalizeSlug` produces strings matching `[a-z0-9-]+` with no leading/trailing dashes. **Colons (`:`) are NOT in the slug character set** — the slug-prefix shorthand `<slug>:<dotted>` is syntactically unambiguous. ✓
- **Default-project mechanism check:** `git grep -nE "DefaultProject|default_project|defaultProject|TILLSYN_PROJECT|TILL_PROJECT" -- 'cmd/till/' 'internal/config/'` returns no default-project mechanism. PLAN.md `:307`'s "bare dotted form without project errors" rule is correct. ✓
- **Shell-escape conflict:** `<slug>:<dotted>` doesn't collide with shell metacharacters (`:` is not special in Bash), nor with environment-variable expansion (`$slug` is unrelated syntax). No shell-quoting hazard. ✓
- **MCP session inference:** PLAN.md `:307` says project is inferred from the MCP auth-gated session for tool calls. Existing auth-gated patterns expose project context. ✓

---

## 6. Repository Contract Re-Check

- **`ListActionItemsByParent` signature:** `(ctx context.Context, projectID, parentID string) ([]domain.ActionItem, error)` — `projectID` required (PLAN.md `:287`); `parentID == ""` convention for "level-1 children" matches the existing `ParentID == ""` domain convention at `internal/domain/action_item.go:91-103`. PLAN.md `:287` documents the empty-string semantics explicitly. ✓
- **`fakeRepo` extension scope:** `internal/app/service_test.go fakeRepo` (declared at `:18`) holds `tasks map[string]domain.ActionItem`. Existing methods include `ListActionItems` at `:607`. Plan 2.10 `Paths:` includes `internal/app/service_test.go (extend fakeRepo to implement the new method)`. The extension pattern: filter `f.tasks` by `ParentID == parentID && ProjectID == projectID`, sort by `(CreatedAt, ID)` to mirror the SQLite ordering. PLAN.md doesn't explicitly require fakeRepo's ordering match the SQL ORDER BY — **mild nit**: should be added so dotted-address tests against fakeRepo stay consistent with the production repo. Downgrade-acceptable; builder will figure it out from the symmetry.

---

## Verdict Summary

**FAIL.** Two new blocking findings (Falsification §2.1, §2.2) plus PROOF's wrong PASS verdict (§2.3, structurally derivative of §2.1 + §2.2). Three nits (§2.4, §2.5, §2.6) carry over from Round 2 with PROOF downgrades but are worth recording.

**Most damaging counterexample.** Falsification §2.1 — the R3-8 `position ASC` conditional in PLAN.md `:287` is unsafe. Verified at HEAD: `internal/adapters/storage/sqlite/repo.go:487` shows the existing `position` column index is `(project_id, column_id, position)` — column-scoped, not parent-scoped. A builder who picks the position-prefer branch produces a resolver where dotted addresses shift under user-driven column reordering even when the addressed item never moved. The fallback `created_at ASC, id ASC` saves the build from compile-break, but the resolver's *output semantics* are wrong. This is a *correctness bug*, not a compile-break — `mage ci` will be green.

**Second-most damaging.** Falsification §2.2 — `internal/domain/kind_capability_test.go:19` carries `Done: false` (a `ChecklistItem.Done` field literal) and is missed by Round 3's cite list. Same shape as Round 2 R2-P1 (which Round 3 R3-1 fixed for `internal/app/kind_capability_test.go:429` but not for the `internal/domain` sibling).

**Round 2 → Round 3 progress.** 10 of 11 R2 falsification findings + 7 of 7 R2 proof findings are resolved or downgrade-acceptable. R3 introduces 2 new blockers — one from the new dotted-address spec (R3-8's position-column conditional), one from incomplete propagation of the R3-1 fix. PROOF's PASS verdict misses both.

**Single fix path.** Round 4 surgical pass:

- PLAN.md `:287` — drop the `position ASC` conditional; commit to `ORDER BY created_at ASC, id ASC` only; add explicit prose carve-out distinguishing "list-position-among-children" from "the SQL `position` column."
- PLAN.md `:163` — extend `internal/domain/kind_capability_test.go` cite list to `:19, :35, :73` with "rename `Done: false` field at `:19`" + the existing `RequireChildrenDone → RequireChildrenComplete` at `:35, :73`.
- PLAN.md `:391` — rewrite the `mcp_surface.Completed` rationale to name `ReindexEmbeddingsResult` correctly (R2-F9 carry-over).
- PLAN.md `:167` — tighten `:1965-1975` cite to `:1967, 1969` (R2-F7 carry-over, builder ergonomics).
- PLAN.md `:293` — append a `created_at`-tied-children test case to lock in deterministic resolution under tie-break paths.

These are surgical edits to an otherwise-sound plan. R3-8's spec is internally consistent — only the `position`-column conditional is unsafe.

---

## Verdict Block

```
Verdict: fail
Round 3 FALSIFICATION blockers (new): 2
Round 3 FALSIFICATION nits (new): 0 (3 carry-over R2 nits flagged for closure clarity)
Round 2 FALSIFICATION resolutions confirmed: 5 of 5 blockers + 5 of 6 nits (R2-F11 carried, downgrade-acceptable)
Round 2 PROOF resolutions confirmed: 4 of 4 blockers + 3 of 3 nits
PROOF Round 3's PASS verdict: contradicted by Falsification §2.1 + §2.2
```
