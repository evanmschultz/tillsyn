# DROP_2 — Plan QA Proof, Round 2

**Verdict:** fail
**Date:** 2026-05-01

This Round 2 review verifies that `workflow/drop_2/PLAN.md` has resolved Round 1's blockers (5 PROOF + 9 FALSIFICATION) and is now evidence-complete, internally consistent, and that every claim in droplet acceptance criteria is supported by current code at HEAD. Major Round 1 issues are RESOLVED. Four new blocking gaps surfaced; nits enumerated below.

---

## Round 1 PROOF Blocker Resolution

| R1 Finding | Status | Evidence |
| --- | --- | --- |
| F1 (Unit B Paths underscope — cross-package consumers) | **resolved** | Round 2 collapses Unit B into one atomic droplet 2.7. Paths enumerate `internal/domain/{workitem.go,action_item.go,domain_test.go,kind_capability_test.go}`, `internal/app/{service.go,service_test.go,snapshot.go,snapshot_test.go,attention_capture.go,attention_capture_test.go}`, `internal/adapters/server/common/{capture.go,capture_test.go,app_service_adapter.go,app_service_adapter_test.go,app_service_adapter_lifecycle_test.go,app_service_adapter_mcp.go,types.go}`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/tui/{model.go,model_test.go,options.go,thread_mode.go}`, `internal/config/{config.go,config_test.go}`. Cross-checked via `git grep -nE "StateDone|StateProgress" -- '*.go'` — every consumer file now in 2.7 Paths. |
| F2 (canonicalLifecycleState second coercion site) | **resolved** | `internal/adapters/server/common/capture.go:296-312` now in 2.7 with explicit "rewrite `canonicalLifecycleState` to accept ONLY canonical inputs" acceptance bullet. Verified at HEAD: `func canonicalLifecycleState(state domain.LifecycleState) domain.LifecycleState` lives at `:297` with legacy switch cases at `:301, 303`. |
| F3 (defaultStateTemplates seed) | **resolved** | 2.7 Paths cover `internal/app/service.go:1873-1881` with explicit "flip `defaultStateTemplates()` so the seed ID column emits `\"in_progress\"` and `\"complete\"`" acceptance bullet. Verified at HEAD: `defaultStateTemplates` at `:1874` returns `{ID: "progress"}` and `{ID: "done"}`. |
| F4 (acceptance grep too broad) | **resolved** | 2.7 acceptance now uses scope-narrowed greps: `git grep -nE 'domain\.StateDone|domain\.StateProgress'`, `git grep -nE 'lifecycle_state.*"done"\|lifecycle_state.*"progress"'`, `git grep -nE 'json:"done_tasks"\|json:"done_items"'`, plus per-file checks scoped to state-machine source files. The unconstrained `'"done"'` whole-tree grep is gone. |
| F5 (parent PLAN.md drift) | **resolved** | Verified `main/PLAN.md:1602` at HEAD: line now reads "**Pre-MVP rule: no migration logic in Go code, no `till migrate` CLI subcommand, no SQL backfill — dev fresh-DBs `~/.tillsyn/tillsyn.db` after the unit lands.**" The `internal/app/migrations/role_hydration.go` reference and D2 hydration-runner language are gone. |

## Round 1 FALSIFICATION Blocker Resolution

| R1 Finding | Status | Evidence |
| --- | --- | --- |
| §1 (Unit B Paths omit ≥4 production files) | **resolved** | All 14 files identified in R1 §2.2 are now in 2.7 Paths. |
| §3 (`internal/tui/thread_mode.go` missing) | **resolved** | 2.7 Paths explicitly includes `internal/tui/thread_mode.go` with cite `:151` (verified at HEAD: `domain.StateDone` at `:151`). |
| §4 (`capture.go` second coercion site) | **resolved** | See PROOF F2 above. |
| §5 (`internal/app/service.go` missing) | **resolved** | 2.7 Paths cover `service.go` with explicit acceptance for `defaultStateTemplates`, `normalizeStateID`, `lifecycleStateForColumnID`, plus 16+ state-symbol references. |
| §6 (`attention_capture.go`/`snapshot.go`/`service_test.go` missing) | **resolved** | All in 2.7 Paths. |
| §7 (`internal/config/config.go` missing) | **resolved** | 2.7 Paths cover `config.go:218, 550, 1092-1094` with explicit strict-canonical acceptance — no "external compatibility" carve-out. |
| §8 (`app_service_adapter_lifecycle_test.go` missing) | **resolved** | 2.7 Paths cover. Plan correctly classifies `:716, 721, 912` `RevokedReason: "done"` as free-form (verified: `lifecycle_test.go:716, 721, 912` are capability-lease revoke reasons, not state literals). |
| §9 (`extended_tools_test.go` missing) | **resolved** | 2.7 Paths cover `mcpapi/extended_tools_test.go` cites `:427, 446, 1114, 2587, 2600`. Verified hits at HEAD. |
| §1 mage-ci-green-between-units invariant | **resolved** | Unit B is one atomic droplet; the invariant cannot break mid-Unit-B. |

All 5 R1 PROOF blockers and 9 R1 FALSIFICATION blockers are addressed. Major rewrite is sound.

---

## New Round 2 Findings

### 1. `internal/app/kind_capability_test.go` missing from 2.7 Paths — `ChecklistItem.Done` field rename will compile-fail

**Severity:** blocking

**Evidence.** `internal/app/kind_capability_test.go:429` contains `{ID: "ck-run-tests", Text: "run package tests", Done: false}` — a literal of `ChecklistItem.Done`. Renaming the struct field `Done bool` → `Complete bool` at `internal/domain/workitem.go:84` (in scope of 2.7) compile-breaks this file. The file is **not** in 2.7's `Paths:` line.

`git grep -n 'Done: (true|false)' -- 'internal/'` shows hits at:
- `internal/adapters/server/common/capture_test.go:124` — in 2.7 Paths ✓
- `internal/app/kind_capability_test.go:429` — **NOT in 2.7 Paths** ✗
- `internal/app/service_test.go:3003, 3038, 3040, 4612` — `service_test.go` in Paths ✓ (but plan's specific cite `:3039` should be `:3040`; see Finding 5 nit)
- `internal/domain/domain_test.go:275, 421, 422, 425, 428, 536` — file in Paths; range `420-442` covers `:421-428` ✓
- `internal/domain/kind_capability_test.go:19` — file in 2.7 Paths but plan only enumerates `:35`; `:19` is a **distinct** `Done: false` site that needs renaming

**Why blocking.** Builder following 2.7 strictly may not edit `internal/app/kind_capability_test.go` because it's not declared. After 2.7 lands, `mage ci` lights up with a compile error in `internal/app` (`unknown field Done in struct literal of type domain.ChecklistItem`). The unit-boundary `mage ci` gate catches it but 2.7's per-droplet acceptance does not pre-flag the file.

**Suggested fix.** Add `internal/app/kind_capability_test.go` to 2.7's `Paths:` enumeration in the `internal/app/` block. Add `internal/domain/kind_capability_test.go:19` to the existing `:35` cite ("rename `Done: false` at `:19` and `RequireChildrenDone:` at `:35`").

---

### 2. `service_test.go` search-state legacy literals (`States:` / `StateID:`) at `:1561, 1567, 1574, 1580, 1766, 1775, 2953` not enumerated

**Severity:** blocking

**Evidence.** `git grep -nE '"progress"|"done"' internal/app/service_test.go` shows state-machine literals NOT covered by 2.7's enumeration:

- `:1561` — `States: []string{"progress"}` (search filter, state-vocab context)
- `:1567` — `matches[0].StateID != "progress"` (state-vocab assertion)
- `:1574` — `States: []string{"archived"}` (canonical, OK)
- `:1580` — `matches[0].StateID != "archived"` (canonical, OK)
- `:1766` — `States: []string{"archived"}` (canonical, OK)
- `:1775` — `archivedMatches[0].StateID != "archived"` (canonical, OK)
- `:2953` — `if got[0].ID != "progress" || got[1].ID != "todo"` (sanitized state-template ID assertion)

The non-canonical hits at `:1561, 1567, 2953` are legitimate state-vocab values that must rename to `"in_progress"` under strict-canonical. They are inside `service_test.go` (in 2.7 Paths) but the plan's specific "verified hits at" enumeration (PLAN.md line 167) did not include them. A builder reading the cite list literally may miss these and `mage ci` (which fails strict-canonical normalizer assertions when `States: []string{"progress"}` round-trips) lights up red.

**Why blocking.** Same package as 2.7 in scope, but the plan under-declares the specific lines. Builder may overlook them; acceptance grep `git grep -nE '"progress"|"done"' internal/app/service.go` returns empty — but the plan's per-file scoped grep enumeration **doesn't include `service_test.go`** in the scope-aware sweep (PLAN.md line 214 lists `service.go` but not `_test.go`).

**Suggested fix.** Add `:1561, 1567, 2953` to 2.7's `service_test.go` cite list (alongside the 16+ symbol-reference lines already enumerated). Alternatively, expand acceptance bullet to add `git grep -nE '"progress"|"done"' internal/app/service_test.go` returning empty (after the rename) as a per-file scoped check.

---

### 3. 2.7 missing same-package `blocked_by: 2.5` for `internal/adapters/server/common/` and `internal/adapters/server/mcpapi/`

**Severity:** blocking

**Evidence.** Plan line 226: `**Blocked by:** 2.6`. But:

- 2.5 (MCP role) Paths include: `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`
- 2.7 (state-vocab) Paths include: `internal/adapters/server/common/{capture.go, capture_test.go, app_service_adapter.go, app_service_adapter_test.go, app_service_adapter_lifecycle_test.go, app_service_adapter_mcp.go, types.go}`, `internal/adapters/server/mcpapi/extended_tools_test.go`

Both 2.5 and 2.7 edit `internal/adapters/server/common/app_service_adapter_mcp.go` (same file) and the same Go package `internal/adapters/server/common` AND the same Go package `internal/adapters/server/mcpapi`. The post-Drop-1 rule (CLAUDE.md "File- and package-level blocking") requires explicit `blocked_by` between siblings sharing a file or package.

Walking the chain: 2.5 → 2.4 → 2.3 → 2.2; 2.7 → 2.6 → 2.3 → 2.2. Both converge at 2.3 but 2.7 does not transitively block on 2.5. Under a parallel-spawn cascade scheduler (which Drop 4 lands but the plan must already be safe-against now), 2.5 and 2.7 could run concurrently after their respective ancestors close. Same-file edit on `app_service_adapter_mcp.go` is the load-bearing race.

**Why blocking.** Even pre-cascade (orchestrator manually serial-spawns), the plan's Blocked-by graph is the *contract* QA asserts — and the contract is wrong on this same-package edge. R1 plan FALSIFICATION (which I'm extending) flagged the same-package serialization rule explicitly; it must be enforced.

The plan's own Notes line 67 says "for testing-clarity they serialize 2.4 → 2.5 → 2.6" — but the `Blocked by:` lines on each droplet do NOT enforce that serialization (2.6 is `Blocked by: 2.3`, not `2.5`). Plan prose contradicts the actual chain.

**Suggested fix.** Either (a) add `Blocked by: 2.6, 2.5` to droplet 2.7, OR (b) add `Blocked by: 2.5` to droplet 2.6 (which would transitively force 2.7 → 2.6 → 2.5). Option (b) matches the prose at line 67. Either way, fix the chain so 2.7 is guaranteed to run AFTER 2.5 closes.

---

### 4. `internal/domain/domain_test.go:114` mis-classified as state-vocab-relevant

**Severity:** blocking (plan instruction is wrong; builder may make an unnecessary change)

**Evidence.** PLAN.md line 318 (Notes — Round 2 revision summary, item B2) says: "`internal/domain/domain_test.go:114` column-rename test (verified state-vocab-relevant — column name `\"done\"`)."

Verified at HEAD:
```
:111: if err := c.Rename("  done ", now.Add(time.Minute)); err != nil {
:114: if c.Name != "done" {
```

This test verifies that `Column.Rename` trims whitespace — the `"done"` is a free-form column name, not a state literal. Column names are user-supplied free-form strings (the boot-seed names them `"To Do"`, `"In Progress"`, `"Done"`, `"Failed"` per `defaultStateTemplates` line 1876-1879). The test asserts the trim behavior on `c.Name` regardless of vocabulary.

The plan's classification implies the builder must rename `:111`/`:114` from `"done"` to `"complete"`. Doing so weakens the trim-behavior test (still valid) but is not REQUIRED by strict-canonical — a free-form column name `"done"` is simply a column called "done", not a state literal.

**Why blocking.** A misclassification in plan prose can cause an over-edit. If the builder follows the prose, they rename test fixtures that don't actually need renaming. `mage ci` won't catch the misclassification (test still passes after over-edit).

**Suggested fix.** Remove `domain_test.go:114` from the "state-vocab-relevant" list in PLAN.md line 318. Add a note in Cross-droplet decisions: "Column names are free-form strings; `Rename(\"  done  \")` test fixtures stay as-is (they test whitespace trim, not state vocabulary)."

---

### 5. Cite drift — minor

**Severity:** nit

**Evidence (multiple).**

- `service_test.go:3039` cited as a `Done:` rename site (PLAN.md line 167). Actual `Done: true` literal is at `:3040`; `:3039` is `CompletionChecklist: []domain.ChecklistItem{` (open-brace). Off-by-one cite.
- `service_test.go:3797` listed in the symbol-reference sweep (line 167 grep enumeration). Verified at HEAD: `Reason: "done"` at `:3797` is a capability-lease revoke reason, NOT a state literal. The plan's Notes correctly classifies similar `Reason:` / `RevokedReason:` cases at lifecycle_test.go but inadvertently rolled `service_test.go:3797` into the rename sweep. Builder may over-edit.
- `domain.NormalizeLifecycleState` (PLAN.md line 219, acceptance bullet) is wrong on the symbol name. Actual symbol is **unexported** `domain.normalizeLifecycleState` (lowercase `n`) at `internal/domain/workitem.go:148`. The acceptance grep targeting `domain.NormalizeLifecycleState` would return empty regardless. Plan should reference the actual unexported symbol; alternatively, the builder must export it (out of scope for 2.7) or the acceptance refers to the right symbol name.

**Suggested fix.**
- PLAN.md line 167: change `:3039` to `:3040`; remove `:3797` from the sweep list, classify it explicitly under "verified no state-machine touches" alongside the lifecycle_test.go `Reason: "done"` cases.
- PLAN.md line 219: change `domain.NormalizeLifecycleState` to `domain.normalizeLifecycleState` (or note the function is unexported and acceptance wraps via the public surface — `domain.IsTerminalState`, `SetLifecycleState`, etc.).

---

### 6. `Done: true/false` literal greps not in 2.7 acceptance

**Severity:** nit

**Evidence.** PLAN.md acceptance lines 199-216 enumerate symbol greps (`StateDone`, `StateProgress`, `RequireChildrenDone`, `DoneActionItems`, `DoneItems`) and JSON-tag greps but does NOT include a `Done: true|false` field-literal grep. The field rename is a top-level scope item (line 160 — "rename `ChecklistItem.Done bool` → `ChecklistItem.Complete bool`") but the acceptance bullet does not surface a sweep grep for the field-literal pattern.

The R1 PROOF Finding 2 noted the original grep `git grep "ChecklistItem.*Done bool|.Done = true|.Done = false"` was malformed. Round 2 dropped that grep entirely. A correct grep would be `git grep -nE "ChecklistItem.*Done bool|\\bDone:\\s*(true|false)\\b" -- '*.go'`. Without it, build-QA can't verify the field-literal rename completeness.

**Suggested fix.** Add to 2.7 acceptance: `git grep -nE 'Done:\s*(true|false)' -- '*.go'` returns empty; only `Complete:` literals remain.

---

### 7. MD-cleanup carve-out boundary tightened — verified PASS

**Severity:** —

**Evidence.** PLAN.md line 392 carve-out reads: "delete the broken phrase or replace with `<deleted in Drop 2 — see PLAN.md § 19.3>`. No paraphrasing surrounding sentences. Anything beyond a delete-or-stub is out of scope and routes to a future MD-cleanup refinement drop." This is the tightened version per R1 FALSIFICATION §13 suggestion. Boundary is now: delete-or-stub-marker only, no paraphrasing. Build-QA verifies via `git diff`.

This addresses R1 FALSIFICATION nit §13. No new finding.

---

### 8. `mcp_surface.Completed` field independence confirmed — verified PASS

**Severity:** —

**Evidence.** Verified at HEAD via `Read internal/adapters/server/common/mcp_surface.go:215-229`: the `Completed bool json:"completed"` field at `:227` is on a struct containing `ScannedCount`, `QueuedCount`, `ReadyCount`, `FailedCount`, `StaleCount`, `RunningCount`, `PendingCount`, `Completed`, `TimedOut` — clearly an embedding-job / async-job status response shape, not a checklist field. Different type from `ChecklistItem`.

PLAN.md Notes (Cross-droplet decisions, line 372) correctly classifies this and explicitly excludes it from the rename. ✓

---

### 9. Blocked-by chain acyclic — verified PASS

**Severity:** —

**Evidence.** Walking the chain from PLAN.md: 2.1→ε, 2.2→ε, 2.3→2.2, 2.4→2.3, 2.5→2.4, 2.6→2.3, 2.7→2.6, 2.8→2.7, 2.9→2.8, 2.10→2.9, 2.11→2.10. Each edge points to a strictly-earlier droplet. No cycles. ✓

(Finding 3 above flags a missing `blocked_by` edge — that's about completeness, not cycles.)

---

### 10. `Repository.ListActionItemsByParent` decision in 2.10 — verified PASS

**Severity:** —

**Evidence.** PLAN.md droplet 2.10 Paths include `internal/app/ports.go` and `internal/adapters/storage/sqlite/repo.go`. Acceptance specifies the new method signature `ListActionItemsByParent(ctx context.Context, projectID, parentID string) ([]domain.ActionItem, error)` and constrains the SQLite implementation to use an indexed query (`WHERE project_id = ? AND parent_id = ?`).

Verified at HEAD: `internal/app/ports.go:11-53` `Repository` interface has `GetActionItem`, `ListActionItems` but no `ListActionItemsByParent`. Plan correctly adds the method.

This addresses R1 PROOF Finding 6 (nit — under-stated data-shape impact). ✓

---

### 11. Mutation-path enumeration in 2.11 — verified PASS

**Severity:** —

**Evidence.** PLAN.md droplet 2.11 acceptance lists `create|update|move|move_state|delete|restore|reparent` (7 mutations). Cross-checked against `internal/adapters/server/mcpapi/extended_tools.go:904-1282` action_item operation dispatch:
- `:904` get (read), `:918` list (read), `:945` search (read)
- `:981` create (mutation), `:1049` update (mutation), `:1100` move (mutation), `:1150` move_state (mutation), `:1196` delete (mutation), `:1241` restore (mutation), `:1282` reparent (mutation)

7 mutations enumerated; 3 read-only operations (get, list, search) are correctly read-allowed. ✓

---

### 12. No alias-tolerance leaks elsewhere in plan — verified PASS

**Severity:** —

**Evidence.** Re-read PLAN.md searching for any phrase contradicting strict-canonical. Found none. The plan repeatedly emphasizes:
- Line 150 — "Strict-canonical only (dev decision, 2026-05-01). No alias tolerance."
- Line 192 — `internal/config/config.go isKnownLifecycleState` is strict-canonical, no "external compatibility" carve-out.
- Lines 219, 221 — JSON unmarshal accepts ONLY canonical, decode error on legacy keys.
- Cross-droplet decisions (line 368) — every state-coercion site is strict-canonical.

No drift. ✓

---

## Verdict Summary

**FAIL.** Round 2 resolves all 5 R1 PROOF blockers and all 9 R1 FALSIFICATION blockers (atomic Unit B, full file enumeration, capture.go covered, narrowed greps, parent PLAN.md corrected). However, four new blocking gaps surfaced:

1. **`internal/app/kind_capability_test.go` missing from 2.7 Paths** — `Done: false` at `:429` will compile-fail after the field rename.
2. **`service_test.go:1561, 1567, 2953` legacy state literals not enumerated** — `States: []string{"progress"}` and `StateID == "progress"` need rewriting to canonical.
3. **2.7 missing `blocked_by: 2.5`** — both droplets edit `internal/adapters/server/common/app_service_adapter_mcp.go` (same file) and share the `internal/adapters/server/common` + `internal/adapters/server/mcpapi` packages. Plan's prose at line 67 promises serialization but the actual `Blocked by:` chain doesn't enforce it.
4. **`domain_test.go:114` mis-classified as state-vocab-relevant** — column name `"done"` is free-form, not a state literal; plan's classification will cause unnecessary over-edit.

Plus three nits: cite drift (`:3039` → `:3040`, `:3797` over-claimed), unexported `normalizeLifecycleState` symbol naming, and a missing `Done: true|false` field-literal acceptance grep.

The single most damaging Round 2 finding is **Finding 1** — an entire test file is missing from 2.7's Paths, which will compile-fail after the `ChecklistItem.Done → Complete` rename. The fail-loud unit-boundary `mage ci` gate catches it, but the plan's `Paths:` declaration is the file-locking contract and it's incomplete.

**Single fix path.** A focused Round 3 revision pass touching only:
- 2.7 `Paths:` block — add `internal/app/kind_capability_test.go`; expand `internal/domain/kind_capability_test.go` cite from `:35` to `:19, 35`.
- 2.7 `service_test.go` cite list — add `:1561, 1567, 2953`; remove `:3797`; correct `:3039` to `:3040`.
- 2.7 `Blocked by:` line — change to `Blocked by: 2.6, 2.5` (or add `Blocked by: 2.5` to 2.6).
- PLAN.md Notes line 318 — remove `domain_test.go:114` from "state-vocab-relevant" list; add carve-out note about column-name free-form-strings.
- 2.7 acceptance — add `git grep -nE 'Done:\s*(true|false)' -- '*.go'` returns empty; correct `domain.NormalizeLifecycleState` to `domain.normalizeLifecycleState` (or remove that bullet — it's redundant with `IsTerminalState` test in the same block).

These are surgical edits to an otherwise-sound plan. Round 1's heavy lifting (atomic Unit B, full file enumeration, narrowed greps, parent PLAN.md correction) is correctly applied.
