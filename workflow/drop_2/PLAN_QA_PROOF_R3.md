# DROP_2 — Plan QA Proof, Round 3

**Verdict:** pass
**Date:** 2026-05-01

This Round 3 review verifies that `workflow/drop_2/PLAN.md` has resolved every Round 2 PROOF blocker (4) + nit (3) and every Round 2 FALSIFICATION blocker (5) + nit (6). Round 3 was a surgical-fix pass — no planner respawn — closing citation drift and the cross-unit serialization gap. The new Round 3 spec for the dotted-address resolver (project-less body, 0-indexed positions, `created_at ASC, id ASC` tie-break, ambiguity unreachable by construction) is also reviewed for internal consistency. **Result: every Round 2 finding is resolved or correctly downgraded; no new Round 3 blockers; one residual nit (Round 2 §2.10 capture_test.go format-string label) is unresolved but is downgrade-acceptable for Round 3 sign-off.**

---

## Round 2 PROOF Blocker Resolution

| R2 Finding | Status | Evidence at HEAD |
| --- | --- | --- |
| **R2-P1.** `internal/app/kind_capability_test.go` missing from 2.7 Paths | **resolved** | `internal/app/kind_capability_test.go:429` confirmed at HEAD as `{ID: "ck-run-tests", Text: "run package tests", Done: false}` (verified via Read). PLAN.md `:164` adds the file to 2.7 `Paths:` with explicit "rename `Done: false` test fixture at `:429`" + "**Without this file in scope, `mage test-pkg ./internal/app` compile-fails after the field rename.**" Round 3 patch R3-1 explicitly logs this addition. |
| **R2-P2.** `service_test.go:1561, 1567, 2953` legacy state literals not enumerated | **resolved** | All three lines confirmed at HEAD as state-vocab literals: `:1561` `States: []string{"progress"}`; `:1567` `matches[0].StateID != "progress"`; `:2953` `if got[0].ID != "progress"`. PLAN.md `:168` lists these explicitly: *"rewrite `States: []string{"progress"}` and `StateID == "progress"` legacy state literals at `:1561, 1567, 2953` to canonical (`"in_progress"`)."* Round 3 patch R3-2 logs this addition. |
| **R2-P3.** 2.7 missing `Blocked by: 2.5` (same-file edit on `app_service_adapter_mcp.go`) | **resolved** | Verified both droplets edit the same file: 2.5 at PLAN.md `:118` ("thread `Role` through `CreateActionItem` at `:620`" — confirmed in `app_service_adapter_mcp.go:620 CreateActionItem`); 2.7 at PLAN.md `:180` ("rename symbols at `:820, 854, 856`" — confirmed at `:820` via Read). 2.7 `Blocked by:` line at PLAN.md `:229` reads `Blocked by: 2.5, 2.6` with parenthetical *"2.5 added Round 3"* explanation. R3-3 logs the patch. |
| **R2-P4.** `domain_test.go:114` mis-classified as state-vocab-relevant | **resolved** | `:111-114` confirmed at HEAD as a free-form column-rename test (`Rename("  done ", ...)` then `c.Name != "done"` — testing whitespace trim on a column name). PLAN.md `:162` now reads: *"**NOT in scope:** `:114` references column name `"done"` as a free-form column-rename test input, NOT a lifecycle state literal — leave unchanged."* R3-4 logs the carve-out. |
| **R2-P5 nit.** `service_test.go:3039 → :3040` cite drift | **resolved** | `:3040` confirmed at HEAD as `{ID: "k1", Text: "docs updated", Done: true}` (a `ChecklistItem.Done` literal); `:3039` is `CompletionChecklist: []domain.ChecklistItem{` (open-brace). PLAN.md `:168` now reads `:3003, 3038, 3040, 4612` (`:3040`, not `:3039`). R3-7 logs the fix. |
| **R2-P5 nit.** `service_test.go:3797` over-claimed as state literal | **resolved** | `:3797` confirmed at HEAD as `Reason: "done"` capability-lease revoke (free-form). PLAN.md `:168` now reads: *"**NOT in scope:** `:3797` is `Reason: "done"` on a capability-lease revoke (free-form text), NOT a state literal — leave unchanged."* R3-7 logs. |
| **R2-P6 nit.** `Done: true|false` field-literal acceptance grep missing | **resolved** | PLAN.md `:211` adds: `git grep -nE '\.Done\s*=\s*(true|false)|Done:\s*(true|false)' -- '*.go'` returns only stdlib concurrency idioms (`ctx.Done()`, `wg.Done()`) — zero `ChecklistItem.Done` field-literal sites remain. R3-7 logs. (R2 also flagged the unexported `isValidLifecycleState` symbol naming — verified at HEAD `:166` as lowercase, and PLAN.md `:160` now matches.) |

All 4 Round 2 PROOF blockers + 3 nits are addressed. Citations confirmed by Read at HEAD on 2026-05-01.

---

## Round 2 FALSIFICATION Blocker Resolution

| R2 Finding | Status | Evidence at HEAD |
| --- | --- | --- |
| **R2-F1.** Droplet 2.10 will not compile — `service_test.go fakeRepo` missing from `Paths:` | **resolved** | `service_test.go:18` confirmed at HEAD as `type fakeRepo struct {...}`; `newFakeRepo()` constructor at `:37`. PLAN.md `:281` now lists `internal/app/service_test.go (extend `fakeRepo` to implement the new method — without this, `mage test-pkg ./internal/app` compile-fails on every test that constructs `fakeRepo`)`. R3-5 logs. Also, PLAN.md `:168` notes *"`fakeRepo` extension for new `Repository.ListActionItemsByParent` method belongs in Droplet 2.10, NOT 2.7"* — clean ownership. |
| **R2-F2.** Same-package compile race 2.5 + 2.7 (`app_service_adapter_mcp.go`) | **resolved** | See R2-P3 above. `Blocked by: 2.5, 2.6` on 2.7 closes the race. |
| **R2-F3.** Droplet 2.7 misses `extended_tools.go:1339` MCP tool description | **resolved** | `:1339` confirmed at HEAD as `mcp.WithString("state", mcp.Description("Lifecycle state target for operation=move_state (for example: todo|in_progress|done)"))`. PLAN.md `:184` adds the file to 2.7 `Paths:` with explicit "rewrite the MCP tool-description string at `:1339`" pointing at the canonical example. R3-6 logs. |
| **R2-F4.** Dotted-address resolver `N.M.K` semantics under-specified | **resolved** | PLAN.md `:285` now defines: *"Dotted body is project-LESS and 0-indexed at every level. Form: `<lvl1_pos>.<lvl2_pos>.<lvl3_pos>...` ... Body regex: `^\d+(\.\d+)*$`."* PLAN.md `:287`: *"`Repository.ListActionItemsByParent(ctx, projectID, parentID)` ... deterministically ordered by `created_at ASC, id ASC` (or `position ASC, created_at ASC, id ASC` if a `position` column exists)."* PLAN.md `:292`: *"No `ErrDottedAddressAmbiguous` error — by construction the deterministic ORDER BY + UUID tie-breaker yields a unique item per index, so ambiguity is unreachable."* R3-8 logs. (See "Internal consistency of dotted-address spec" below.) |
| **R2-F5.** `kind_capability_test.go:73` reader for `RequireChildrenDone` | **resolved** | `:73` confirmed at HEAD as `if !kind.Template.ActionItemMetadataDefaults.CompletionContract.Policy.RequireChildrenDone {`. PLAN.md `:163` now reads "rename `RequireChildrenDone:` test fixtures at `:35` AND `:73` to `RequireChildrenComplete:`." R3-7 logs. |
| **R2-F6 nit.** Cite drift `service.go:556, :694` (claimed state-rename, actually `StateTodo`) | **resolved** | `:556` confirmed at HEAD as `lifecycleState = domain.StateTodo`; `:694` as `restoredState = domain.StateTodo`. PLAN.md `:167` now reads: *"**NOT in scope:** `:556` and `:694` reference `domain.StateTodo` (unchanged by Drop 2) — leave unchanged."* R3-7 logs. |
| **R2-F7 nit.** Cite drift `service.go:1965-1975` (range half-correct) | **partially-resolved** | PLAN.md `:167` lists `:623, 627, 639, 644, 1817, 1965-1975` for `domain.StateDone/Progress`. The `1965-1975` range still over-claims (R2 noted only `:1967, 1969` are renamable). Acceptance grep `git grep -nE 'domain\.StateDone|domain\.StateProgress' -- '*.go'` returning empty WILL catch a builder leaving a renamable site behind, AND will not flag a `StateTodo`/`StateFailed`/`StateArchived` line as over-edit (the grep is symbol-scoped). Builder reading the cite range literally would touch `:1965, 1971-1973, 1975` and find no renamable symbol — no harm done, but the cite is imprecise. **Downgrade-acceptable nit** — does not block Round 3 sign-off. |
| **R2-F8 nit.** `IsValidLifecycleState` exported-vs-unexported casing | **resolved** | `:166` confirmed at HEAD as `func isValidLifecycleState(state LifecycleState) bool`. PLAN.md `:160` corrected to "rewrite `isValidLifecycleState` (unexported) at `:166`." R3-7 logs. |
| **R2-F9 nit.** `mcp_surface.go:227 Completed` field rationale wording | **partially-resolved** | PLAN.md `:391` (Cross-droplet decisions) reads: *"`mcp_surface.Completed` field is independent of lifecycle-state vocabulary. ... an MCP-response checklist-completion boolean on a different struct from `ChecklistItem`."* The framing still uses "checklist-completion" — Round 2 Falsification §2.8 noted that the field is actually on `ReindexEmbeddingsResult` (embeddings-job status), not a checklist response. Conclusion is right (no rename); rationale gloss imprecise but harmless. **Downgrade-acceptable nit.** |
| **R2-F10 nit.** `snapshot.go:421` error-message text not enumerated | **partially-resolved** | PLAN.md `:169` lists `:419` as the validation switch but does NOT enumerate `:421` (`return fmt.Errorf("tasks[%d].lifecycle_state must be todo|progress|done|failed|archived", i)`). However, the acceptance grep `git grep -nE 'lifecycle_state.*"done"|lifecycle_state.*"progress"' -- '*.go'` (PLAN.md `:214`) catches this format-string because the literal `lifecycle_state` and `"done"`/`"progress"` co-occur. Builder following the grep gate will rewrite. **Downgrade-acceptable** — covered by acceptance grep. |
| **R2-F11 nit.** `capture_test.go:199` debug-message field-rename label | **NOT resolved** | `:199` confirmed at HEAD as `t.Fatalf("WorkOverview counts = %#v, want todo=2 progress=1 done=1 failed=1 archived=1", capture.WorkOverview)`. The format-string contains debug labels `progress=1 done=1`. Round 3 PLAN.md does not flag this. The acceptance grep set does NOT catch a Go format-string literal because the labels lack `lifecycle_state` / `LifecycleState` context and the per-file legacy-literal sweep targets `capture.go` (production) but not `capture_test.go`. The test still functions if the format-string lies — it just produces confusing failure output on test failure. **Severity: nit (debug-only, no runtime correctness impact).** Suggest fixing in builder pass via routine consistency edit (drop carve-out at PLAN.md `:411` permits trivial in-section adjacent fixes), but does not block Round 3 sign-off. |

10 of 11 R2 FALSIFICATION findings are resolved or downgrade-acceptable. R2-F11 (capture_test.go format-string label) is the only finding that survives Round 3 unaddressed; severity is nit (debug output only) and the surrounding code IS in 2.7 Paths so a careful builder will edit the format-string for consistency. Recommend documenting this in the builder context as a "while-you're-there" cleanup but not gating the round.

---

## New Round 3 Findings

### 1. Internal consistency of the dotted-address spec — verified PASS

- **Severity:** —
- **Evidence.** Reading PLAN.md `:285` (body shape), `:286` (slug shorthand), `:287` (ORDER BY), `:292` (`ErrDottedAddressAmbiguous` removed), and PLAN.md `:345` (R3-8 summary): the spec is self-consistent.
  - Body regex `^\d+(\.\d+)*$` matches `0`, `0.0`, `2.5.1`, but rejects `1-2.3`, `tillsyn-2.1`, `1.`, `.1`, `1..2`, `abc`, leading-dash. ✓ (Slug shorthand `<slug>:<dotted>` is split into slug + body BEFORE the body regex applies.)
  - 0-indexed at every level matches the global Tillsyn vocabulary rule (memory `project_tillsyn_cascade_vocabulary.md` — "level_0 = project, level_1 = first child drop, level_N = N deep") and PLAN.md `:285` makes the project-vs-body distinction explicit ("Project NEVER appears as `0` in the body").
  - `ORDER BY created_at ASC, id ASC` (or `position ASC, created_at ASC, id ASC` if a `position` column exists): the UUID `id` tie-breaker renders ambiguity unreachable, validating the `ErrDottedAddressAmbiguous` removal at `:292`.
  - Acceptance test list at `:293` matches: valid `0`, `0.0`, `2.5.1`, slug-prefixed valid, slug-prefix mismatch, out-of-range, malformed inputs, UUID-input rejected. (The Round 2 FAL §2.4 demand for "position-tied children, created_at-tied children" tests is partially addressed — the deterministic ORDER BY makes those tied cases produce defined behavior, but the test list does not explicitly enumerate the tie-break case. **Mild nit but downgrade-acceptable** — the spec defines behavior; absent an explicit test, build-QA will catch any regression via the `ListActionItemsByParent` round-trip test at `repo_test.go`.)

### 2. Repository interface contract consistent with R3-5 — verified PASS

- **Severity:** —
- **Evidence.** PLAN.md `:284` declares `ResolveDottedAddress(ctx, repo, projectID, dotted string) (string, error)`. PLAN.md `:281` lists `internal/app/ports.go (add `ListActionItemsByParent(ctx context.Context, projectID, parentID string) ([]domain.ActionItem, error)` to the `Repository` interface)`. PLAN.md `:287` aligns the method signature: `Repository.ListActionItemsByParent(ctx, projectID, parentID)`. Signature matches across all three sites. The fakeRepo extension at `service_test.go` is in 2.10 Paths (R3-5). All implementers covered: production `*Repository` (`internal/adapters/storage/sqlite/repo.go`) AND test fake `fakeRepo`.

### 3. CLI `--project` flag handling spec is complete — verified PASS

- **Severity:** —
- **Evidence.** PLAN.md `:307` enumerates all three CLI input forms:
  - Bare dotted body without project: errors with clear message ("project is required").
  - Slug-prefix `<proj_slug>:<dotted>` (e.g., `till action_item get tillsyn:1.5.2`): accepted.
  - `--project <slug>` flag with bare body: accepted.
- The Round 2 FAL §2.4 spec request is fully addressed.

### 4. Mutation-rejection enumeration in 2.11 — verified PASS

- **Severity:** —
- **Evidence.** PLAN.md `:301` lists mutation operations `create|update|move|move_state|delete|restore|reparent` (7 total). Cross-checked via Read against `extended_tools.go` action_item dispatch at `:904` (get-read), `:918` (list-read), `:945` (search-read), `:981` (create-mutation), `:1049` (update-mutation), `:1100` (move-mutation), `:1150` (move_state-mutation), `:1196` (delete-mutation), `:1241` (restore-mutation), `:1282` (reparent-mutation). All 7 mutations enumerated; 3 read-only operations (get, list, search) correctly NOT in the rejection list.

### 5. No new strict-canonical / no-migration / no-removal-of-workflow-files contradictions introduced — verified PASS

- **Severity:** —
- **Evidence.** Re-read PLAN.md searching for any phrase contradicting Drop 2's invariants:
  - **Strict-canonical:** `:150` reaffirms "No alias tolerance"; `:387` reiterates strict-canonical-only. No drift.
  - **No migration logic:** `:24-26`, `:397-403` (Explicit YAGNI rulings) all consistent. No `internal/app/migrations/` reference, no `till migrate` subcommand.
  - **No removal of workflow files:** Round 3 PLAN.md edits are additive (adding files to existing 2.7 Paths, adding spec text). No existing workflow MD content removed.
  - **`Blocked by:` chain acyclic:** Edges 2.1→ε, 2.2→ε, 2.3→2.2, 2.4→2.3, 2.5→2.4, 2.6→2.3, 2.7→2.6+2.5, 2.8→2.7, 2.9→2.8, 2.10→2.9, 2.11→2.10. No cycles. Same-package serialization now complete (2.5 + 2.7 race closed).

---

## Verdict Summary

**PASS.** Every Round 2 PROOF blocker (4) + nit (3) is resolved. 5 of 5 Round 2 FALSIFICATION blockers are resolved. 5 of 6 Round 2 FALSIFICATION nits are resolved or downgrade-acceptable; the remaining nit (R2-F11 — `capture_test.go:199` format-string debug label `progress=1 done=1`) is debug-only and does not gate Round 3 sign-off.

**Most important Round 3 strength.** R3-3's `Blocked by: 2.5, 2.6` on Droplet 2.7 closes the same-file race (`app_service_adapter_mcp.go`) that Round 2 flagged as the second-most-damaging finding. The DAG now correctly enforces the cross-unit serialization the Notes prose at `:67` always promised but the actual chain didn't enforce.

**Most important Round 3 risk-mitigation strength.** R3-5 (fakeRepo extension under 2.10) closes the only structural compile-break that Round 2 found — adding `ListActionItemsByParent` to the `Repository` interface forces every implementer to add it, and `fakeRepo` is now declared in 2.10 Paths so `mage test-pkg ./internal/app` compiles.

**New Round 3 spec strength.** R3-8's project-less, 0-indexed, `created_at ASC, id ASC` ordering with ambiguity unreachable-by-construction is a clean simplification over Round 2's 3-tier (`position, created_at, id`) tie-break with `ErrDottedAddressAmbiguous`. The UUID-as-final-tie-breaker eliminates the ambiguity error class entirely. Aligns with the global Tillsyn cascade vocabulary (project NOT a level; 0-indexed levels).

**Single residual nit (non-blocking).** `internal/adapters/server/common/capture_test.go:199` debug-message labels `progress=1 done=1` are still legacy — Round 3 doesn't flag them. Builder may fix opportunistically under PLAN.md `:411`'s carve-out (single-phrase fix in the same file's section); or carry to the next round of MD-cleanup. Not a gate.

**Round 3 reflects a healthy planner-QA-orch loop:** Round 1 found 14 issues, Round 2 found 16 (some carried, some new from the heavy rewrite), Round 3 closes the surgical drift cleanly. The plan is now ready to enter Build phase.

---

## Verdict Block

```
Verdict: pass
Round 3 PROOF blockers (new): 0
Round 3 PROOF nits (new): 0 (1 carried-over R2-F11 nit, downgrade-acceptable, builder carve-out applies)
Round 2 PROOF resolutions confirmed: 4 of 4 blockers + 3 of 3 nits
Round 2 FALSIFICATION resolutions confirmed: 5 of 5 blockers + 5 of 6 nits (R2-F11 carried)
```
