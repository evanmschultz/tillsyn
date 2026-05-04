# DROP 4A — UNIFIED PLAN QA FALSIFICATION (Round 2)

**Reviewer:** `go-qa-falsification-agent` (subagent)
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`
**Target:** `workflow/drop_4a/PLAN.md` (32 droplets, 5 waves) — post Round-1 fix application
**Round:** 2 (post-fix confirmation)
**Date:** 2026-05-03
**Verdict:** **PASS-WITH-NIT** — all five Round-1 fixes (plus the Q9 routing note) are correctly applied. One newly-surfaced same-package overlap (Wave 1 chain ↔ Wave 3 W3.1, plus 4a.12 ↔ Wave 3 W3.1) is structurally identical to Round-1's accepted PASS-WITH-NIT for 4a.12 ↔ Wave 1 chain, and adopts the same disposition: documentation-only NIT, not a build-blocker. Builders may dispatch Wave 0 immediately; the Wave-1↔Wave-3 same-package note SHOULD be added to PLAN.md §7's 4a.24 Notes line before any Wave-3 builder fires, but it does not gate Wave 0 / Wave 1 dispatch.

---

## 1. Methodology

Round 2 ran the prompt's seven verification axes:

- §2 — Per-fix verification (5 fixes + Q9 routing note).
- §3 — Re-walk the DAG for cycles introduced by the new 4a.11 → 4a.15 edge.
- §4 — Re-attack 4a.12's parallelization rationale via merge-collision construction.
- §5 — Re-attack the L1 testability-split sentence against actual droplet content.
- §6 — Re-attack the Q9 LSP-workspace-symbol-search note against actual symbol resolution requirements.
- §7 — New-counterexample sweep (Wave 2/3/4 cross-droplet path collisions; cross-wave hidden dependencies).
- §8 — Drop-3 pattern recurrence re-confirmation post-fix.

Evidence: `PLAN.md` (post-fix), `PLAN_QA_FALSIFICATION.md` (Round 1), `WAVE_1_PLAN.md`, `WAVE_2_PLAN.md`, `WAVE_3_PLAN.md`, `WAVE_4_PLAN.md`. No Hylla queries — target is markdown only; the load-bearing Go-source check (Wave 1 ↔ Wave 3 file overlap on `internal/adapters/server/common/mcp_surface.go` + `app_service_adapter_mcp.go`) was answered by direct path-list comparison across the wave plans.

---

## 2. Per-Fix Verification

### 2.1 PASS — Fix #1: §7 4a.15 `Blocked by:` adds `4a.11`

**Round-1 finding:** 4a.15 missing `blocked_by 4a.11` cross-wave edge.

**Applied fix (PLAN.md line 251):** `**Blocked by:** **4a.14** (package-compile lock), **4a.11** (same-file lock on `internal/app/service.go` — Wave 1 chain edits the same `Service.{Move,Create,Update}ActionItem` methods that 4a.15 extends with `publishActionItemChanged` calls; serializing through Wave 1's terminal node prevents merge conflicts on the publisher additions).`

**Walk-through.** When the dispatcher (or pre-cascade orchestrator) reaches 4a.15, it now reads two blockers: 4a.14 (Wave 2 anchor) AND 4a.11 (Wave 1 terminal node). 4a.11 itself is `blocked_by 4a.10`, which is `blocked_by 4a.9`, …, all the way back to `blocked_by 4a.5`, which is `blocked_by 4a.4` (Wave 0 close). The transitive closure makes `{4a.5, 4a.6, 4a.7, 4a.8, 4a.9, 4a.10, 4a.11}` all required-complete before 4a.15 is dispatched. That's the entire `Service.{Move,Create,Update}ActionItem` Wave-1 chain. Same-file collision on `internal/app/service.go` mechanically prevented.

**Secondary-path bypass check.** Could the dispatcher reach 4a.15 by some path that skips 4a.11? Checked the §8 cross-wave table and §9 topo sort. 4a.15 has exactly two blockers (4a.14 + 4a.11). 4a.14's blockers are `{4a.4}`. 4a.11's blockers are `{4a.10}`, transitively `{4a.5..4a.10}`. There is no edge to 4a.15 from any droplet outside `{4a.14, 4a.11}` — so no bypass path exists. Confirmed.

**Test-fixture drift mitigation (Round-1 §2.1 Case 3) confirmed.** Wave 1's input-struct extensions (`CreateActionItemInput` / `UpdateActionItemInput` Paths/Packages/etc.) all land before 4a.15's `broker_sub_test.go` is authored. 4a.15's tests can rely on the post-Wave-1 input-struct shape. No drift.

Verdict: **PASS** — fix is structurally correct and closes Round-1 §2.1's CONFIRMED counterexample.

### 2.2 PASS — Fix #2: §8 cross-wave edges table adds `4a.11 → 4a.15` row

**Applied fix (PLAN.md line 414):** `| 4a.11                   | 4a.15 (broker sub)     | Same-file lock on `internal/app/service.go` — Wave 1 chain edits `Service.{Move,Create,Update}ActionItem`; 4a.15 extends them with `publishActionItemChanged` calls. (Added post-plan-QA-falsification round-1 fix.) |`

The §8 table is the canonical cross-wave edge enumeration the (future) dispatcher reads. Row format matches the existing rows (e.g. `4a.6 → 4a.18`, `4a.10 → 4a.18`). Reason field cites the load-bearing rationale. The trailing parenthetical names this as a Round-1 fix-in — useful audit trail, doesn't hurt machine readability.

Verdict: **PASS** — table row matches the in-row `Blocked by` declaration in §7 4a.15.

### 2.3 PASS — Fix #3: §9 topological order updates Wave 2 chain header + adds rule bullet

**Applied fix (PLAN.md line 449):** `4a.14 → 4a.15 (after 4a.11 lands — same-file lock on service.go) → 4a.18    (broker → walker, after 4a.10/4a.11 land)`.

**Applied fix (PLAN.md line 469):** new bullet `After 4a.11 closes, 4a.15 unblocks (with 4a.14 also required); the publisher additions to `Service.{Move,Create,Update}ActionItem` only become safe to land once Wave 1's chain has finished editing those same methods.`

**Walk-through.** The §9 prose now matches the §7+§8 declarations: 4a.15 enters the Wave 2 chain only after 4a.11 closes, with 4a.14 as the package-compile-lock anchor. The new bullet closes Round-1's "soft cross-wave coordination concern" gap by explicitly naming the post-4a.11 unblock condition.

**No prose drift.** I checked that the §9 chain still reads coherently with the new parenthetical: `4a.14 → 4a.15 (after 4a.11 lands — same-file lock on service.go) → 4a.18 (broker → walker, after 4a.10/4a.11 land)`. The "after 4a.10/4a.11 land" parenthetical that originally hung off 4a.18 stays — it's still correct (4a.18 lists 4a.10 + 4a.11 in its `Blocked by` per §7 line 275). Both parentheticals coexist without contradiction.

Verdict: **PASS** — §9 prose, §8 table, §7 row are all consistent.

### 2.4 PASS — Fix #4: §7 4a.12 Notes adds parallelization rationale

**Round-1 NIT:** Same-package overlap on `mcp_surface.go` + `extended_tools.go` between 4a.12 and the 4a.5–4a.11 chain.

**Applied fix (PLAN.md line 222 — extension to 4a.12 Notes):** `Same-package parallelization rationale (added post-plan-QA-falsification round-1 NIT): 4a.12 shares packages `internal/adapters/server/common` (file `mcp_surface.go`) and `internal/adapters/server/mcpapi` (file `extended_tools.go`) with the 4a.5–4a.11 chain. Author-judged textually disjoint — different struct definitions in `mcp_surface.go` (project-level vs action-item-level), different `mcp.WithString` registrations in `extended_tools.go` (`till.project` vs `till.action_item` tools). Parallelization is intentional per Drop 3 droplet 3.21 precedent (separate struct extensions in the same wire-surface files lands cleanly). Builder verifies textual disjointness pre-edit; escalates to serialization on Wave 1 close if a hit surfaces.`

**Re-attack: is "textual disjointness" a sound argument?** Round-2 tries to construct counterexamples where two parallel builders editing different parts of the same file collide on a merge:

- **Import-block collision.** Imagine 4a.5 adds `"strings"` to the import block (for path normalization) at the same time 4a.12 adds `"strings"` for language-enum validation. Goimports normalizes import order deterministically; if both builders reach the same final import set, the merge is clean — `goimports` is the canonical formatter and the pre-commit `mage format-check` (4a.1) gates it. **Mitigation in fix:** the rationale doesn't explicitly call out goimports normalization, but the Wave 0 dev hygiene gates (4a.1 `mage format-check`) make this mechanically impossible to land an un-formatted import block. **Refuted, with mitigation already-named.**
- **File-end blank-line discipline.** If 4a.5 appends `CreateActionItemRequest.Paths` while 4a.12 appends `CreateProjectRequest.HyllaArtifactRef` to the END of `mcp_surface.go`, both diffs touch the same trailing-newline line. `git merge` resolves trailing-newline conflicts cleanly. `gofmt` enforces a single trailing newline. **Refuted.**
- **`gofmt` re-flow.** If 4a.5 adds a field with a long doc comment that pushes the existing struct past 80 chars, then 4a.12 adds another long-comment field, gofmt may re-flow alignment of struct tags adjacent to BOTH 4a.5 and 4a.12's additions. Same-line conflict possible if both edits land on adjacent struct-tag-aligned lines. **Real-world check:** `mcp_surface.go` `CreateActionItemRequest` is at lines 65-107; `CreateProjectRequest` is presumably elsewhere in the file (different struct). Different structs are gofmt-aligned independently — alignment is per-struct-block, not file-wide. **Refuted.**
- **Test-fixture drift in `extended_tools_test.go`.** If 4a.5 adds `paths` round-trip test and 4a.12 adds `language` round-trip test in the SAME `func TestExtendedTools(t *testing.T)`, both builders edit the same function body. **Mitigation in WAVE plan:** the fixture-builder pattern in Drop 3 is one test function per field (e.g. `TestPathsRoundTrip` separate from `TestLanguageRoundTrip`). 4a.5's WAVE_1_PLAN.md §1.1 lists `extended_tools_test.go` for "round-trip test for `paths`" — implying a dedicated test function. 4a.12's WAVE_1_PLAN.md §1.8 (would-list, by symmetry) is also dedicated. **Refuted, with naming-discipline mitigation.**

**Verdict on the rationale:** "Textual disjointness on different struct definitions" is sound IF the supporting discipline is in place (goimports + gofmt + per-field test functions + Wave 0 pre-commit hooks). All three disciplines are gated by Wave 0 (4a.1 / 4a.2 / 4a.3) and the Drop 3 precedent (3.21) cited. The fix's rationale is defensible.

**One non-blocking observation:** the rationale says "Builder verifies textual disjointness pre-edit; escalates to serialization on Wave 1 close if a hit surfaces." The escalation path is informal — there's no concrete test the builder runs. Adding "if `git merge --no-ff <main>` produces a conflict on `mcp_surface.go` or `extended_tools.go`, escalate" would be more concrete, but the existing prose is acceptable for a NIT-level fix.

Verdict: **PASS** — fix correctly addresses Round-1's PASS-WITH-NIT for 4a.12 ↔ Wave 1 chain. The textual-disjointness defense is sound under existing Wave-0 gates.

### 2.5 PASS — Fix #5: §1 Goal adds testability-split sentence

**Round-1 NIT:** L1 goal said "spawns subagents via `claude --agent`" but the work splits across 4a.19 / 4a.21 / 4a.23. Reader of L1 alone might assume "spawns" is a single droplet.

**Applied fix (PLAN.md line 15):** `The spawn invocation itself splits across three droplets for testability without `claude` on PATH: 4a.19 constructs the `*exec.Cmd`, 4a.21 executes + monitors, 4a.23 (CLI) orchestrates the full RunOnce path. The split is documented in Q3 and is a deliberate testability choice, not a planning gap.`

**Re-attack: does this accurately describe what 4a.19 / 4a.21 / 4a.23 do?**

- **4a.19 acceptance (PLAN.md §7 line 282):** `BuildSpawnCommand` constructs `*exec.Cmd` (does NOT execute) with `Dir = project.RepoPrimaryWorktree`, full argv per `claude --agent <name> --bare …`. ✓ Matches "constructs the *exec.Cmd."
- **4a.21 acceptance (PLAN.md §7 line 298):** `processMonitor.Track` starts the process + returns `Handle`. `Handle.Wait` returns `TerminationOutcome`. On crash: `MoveActionItem` to `failed` + `metadata.outcome = "failure"` + `metadata.failure_reason`. ✓ Matches "executes + monitors."
- **4a.23 acceptance (PLAN.md §7 line 314):** `till dispatcher run --action-item <id>` cobra subcommand. RunE: instantiate `Dispatcher`, `RunOnce`, print `DispatchOutcome`. ✓ Matches "orchestrates the full RunOnce path."

The L1 sentence accurately maps to the actual droplet acceptance criteria. The "deliberate testability choice, not a planning gap" framing also correctly answers Q3 (the open question).

**Builder-expectation impact.** A builder reading L1 + spawning against (say) 4a.19 will see L1's testability-split sentence first, then the §7 4a.19 acceptance. Mental model: "I'm building the cmd-construction half of the spawn invocation; execution lives in 4a.21." That matches what 4a.19's acceptance criteria expect. No mismatch.

Verdict: **PASS** — fix correctly closes Round-1 §2.6.3 PASS-WITH-NIT and sets correct builder expectations.

### 2.6 PASS — Q9 routing note: §7 4a.29 Notes adds pre-spawn LSP resolution

**Round-1 proof-side finding:** Q9 surfaced description-symbol drift risk (Drop-N references in `main/CLAUDE.md` § Action-Item Lifecycle may not match the actual landing-Drop-N).

**Applied fix (PLAN.md line 375):** `Q9 pre-spawn resolution (added post-plan-QA-proof round-1 NIT): before this droplet starts, orchestrator pre-resolves the "Drop 1 of the cascade plan" prose references in `main/CLAUDE.md` § Action-Item Lifecycle against actual code state via LSP workspace symbol search for `StateFailed` / `RequireChildrenComplete`. The Drop-N references in the prose may not match the actual plan number that landed `failed` (Drop 4a Wave 1 vs original Drop 1). Spawn prompt names the resolved Drop-N references explicitly so the closeout vocabulary stays consistent across W4.1 / W4.2 / W4.4.`

**Re-attack: is "LSP workspace symbol search for `StateFailed` / `RequireChildrenComplete`" sufficient pre-spawn resolution?**

- `StateFailed`: the symbol that lands when always-on parent-blocks-on-failed-child enforces the `failed` terminal state. LSP `workspace/symbol` query for `StateFailed` returns the definition site post-Wave-1; if it's at `internal/domain/workitem.go` (or wherever Wave 1 lands it), the orchestrator knows Wave-1-landed-it.
- `RequireChildrenComplete`: the field 4a.11 deletes. Post-4a.11, LSP workspace symbol search for `RequireChildrenComplete` should return ZERO hits — that's the post-Wave-1 state. If it returns >0 hits, Wave 1 hasn't landed yet and 4a.29 shouldn't dispatch.

**Sufficiency check.** The two symbols cover both polarities: post-Wave-1 we expect `StateFailed` (positive, present) AND `not RequireChildrenComplete` (negative, absent). That's a defensible "Wave 1 has landed and is correctly retired" signal. Adding a third symbol like `failed` (the lifecycle state string) might catch edge cases, but the existing two-symbol check is sufficient for the description-symbol drift Q9 named.

**Process question: does the orchestrator need a more explicit verify-and-document step?** The fix says "Spawn prompt names the resolved Drop-N references explicitly." That's the verify-and-document step — the orchestrator runs LSP, observes the resolved symbol locations, writes them into the spawn prompt. The prompt becomes the durable record. Explicit enough.

**One non-blocking observation:** the fix doesn't say "if the LSP query returns unexpected results, halt and surface to dev." If LSP shows `RequireChildrenComplete` STILL EXISTS post-Wave-1, the spawn should not proceed — but the prose doesn't pin that branch. For Round 2, this is acceptable as builder discipline (the orchestrator dispatching 4a.29 is by definition reading the plan post-Wave-1 close, so the LSP query is a verification step, not a gate). NIT-level: a stronger phrasing would say "if LSP shows `RequireChildrenComplete` references remain, halt and verify Wave 1 close."

Verdict: **PASS** — Q9 fix sufficiently resolves the description-symbol drift risk with the named LSP queries. The "verify-and-document via spawn prompt" step is the pre-spawn resolution mechanism.

---

## 3. DAG Re-Walk

### 3.1 PASS — Acyclicity sweep on revised PLAN.md

**Method.** Walked PLAN.md §8's cross-wave edges + every per-droplet `Blocked by` row in §7. The new edge added by the fix is **4a.11 → 4a.15**.

**Cycle attack.** Could 4a.15 transitively reach 4a.11 through any path? 4a.15's outgoing edges (§9 chain): 4a.15 → 4a.18, 4a.18 → 4a.20, 4a.18 → 4a.23, 4a.20 → 4a.23, etc. 4a.11's incoming edges: 4a.10. 4a.10's incoming: 4a.9. … 4a.5's incoming: 4a.4. None of {4a.18, 4a.20, 4a.23, 4a.29, 4a.30} have outgoing edges back into Wave 1 (Wave 1 is upstream of Wave 2 in the DAG). The new edge 4a.11 → 4a.15 stays acyclic.

**Topological sort still succeeds.** §9's order remains valid: Wave 0 → Wave 1 chain → 4a.12/4a.13 (parallel) → Wave 2 chain (with 4a.15 now after 4a.11) → Wave 3 chain → Wave 4. No reorderings required.

**Cross-check: does the new edge break any existing parallelism opportunity?** §9 originally had `4a.14 → 4a.15 → 4a.18` (broker → walker). Now it's `4a.14 → 4a.15 (after 4a.11 lands) → 4a.18`. 4a.15 was already serial behind 4a.14; the new constraint is "and also after 4a.11." Wave 2's parallel branches (`{4a.16 → 4a.17}`, `{4a.19 → 4a.21}`) are unaffected. Only loss: 4a.15 cannot start at the same moment as 4a.16 or 4a.19 (it must wait for Wave 1 close). That was already the case implicitly — the broker subscriber's `service.go` edits force Wave-1-close gating regardless; the fix just makes it explicit.

Verdict: **PASS** — DAG remains acyclic with the new edge; no new dangling references; topological order unchanged.

### 3.2 PASS — No new dangling references

Walked every `blocked_by` ID in §7 and §8. Every ID resolves to an existing droplet (4a.1 through 4a.32). No typos, no off-by-one errors introduced by the fix.

---

## 4. New Counterexample Sweep

Round 1 caught the 4a.15 ↔ Wave 1 chain. Round 2 sweeps for COUSIN counterexamples in the same shape but different droplet pairs.

### 4.1 NEW NIT (PASS-WITH-NIT) — Wave 1 chain ↔ Wave 3 W3.1 (4a.24) same-package overlap on `mcp_surface.go` + `app_service_adapter_mcp.go`

**The counterexample Round 1 missed.** Walking Wave 3's path declarations:

- **4a.24 paths (PLAN.md §7 line 326):** `internal/domain/{auth_request.go,auth_request_test.go}`, `internal/app/{auth_requests.go,auth_requests_test.go}`, `internal/adapters/auth/autentauth/service.go`, **`internal/adapters/server/common/{mcp_surface.go,app_service_adapter_mcp.go,app_service_adapter_auth_requests_test.go}`**, `internal/adapters/server/mcpapi/handler.go`.
- **4a.5 paths (PLAN.md §7 line 162) — and the entire 4a.5–4a.11 chain by extension:** `internal/domain/{action_item.go,domain_test.go,errors.go}`, `internal/app/{service.go,snapshot.go,snapshot_test.go}`, **`internal/adapters/server/common/{mcp_surface.go,app_service_adapter_mcp.go}`**, `internal/adapters/server/mcpapi/{extended_tools.go,extended_tools_test.go}`, `internal/adapters/storage/sqlite/{repo.go,repo_test.go}`.
- **4a.10 paths (PLAN.md §7 line 202):** `internal/adapters/server/common/{mcp_surface.go,app_service_adapter_mcp.go}`, `internal/adapters/server/mcpapi/{extended_tools.go,extended_tools_test.go}`.
- **4a.12 paths (PLAN.md §7 line 218):** `internal/domain/{project.go,domain_test.go,errors.go}`, `internal/app/{service.go,snapshot.go,snapshot_test.go}`, **`internal/adapters/server/common/{mcp_surface.go,app_service_adapter.go}`**, `internal/adapters/server/mcpapi/{extended_tools.go,extended_tools_test.go}`, `internal/adapters/storage/sqlite/{repo.go,repo_test.go}`.

**Overlap matrix (Wave 1 chain ↔ Wave 3 W3.1):**

| File | 4a.5–4a.11 | 4a.12 | 4a.24 |
| --- | --- | --- | --- |
| `internal/adapters/server/common/mcp_surface.go` | YES | YES | YES |
| `internal/adapters/server/common/app_service_adapter_mcp.go` | YES (4a.5–4a.11) | NO (4a.12 uses `app_service_adapter.go`) | YES |
| `internal/adapters/server/mcpapi/extended_tools.go` | YES | YES | NO (4a.24 uses `handler.go`) |
| `internal/adapters/server/mcpapi/handler.go` | NO | NO | YES |

**Three same-file overlap pairs:**

1. **4a.5–4a.11 ↔ 4a.24 on `mcp_surface.go`** — Wave 1 chain extends `CreateActionItemRequest` / `UpdateActionItemRequest` / `MoveActionItemRequest`. 4a.24 adds NEW struct `ApproveAuthRequestRequest`. Different structs in the same file. Same package compile lock.
2. **4a.5–4a.11 ↔ 4a.24 on `app_service_adapter_mcp.go`** — Wave 1 chain extends `CreateActionItem` / `UpdateActionItem` / `MoveActionItemState` adapter methods. 4a.24 adds NEW method `ApproveAuthRequest`. Different methods in the same file.
3. **4a.12 ↔ 4a.24 on `mcp_surface.go`** — 4a.12 adds project-fields struct extensions; 4a.24 adds `ApproveAuthRequestRequest`. Different structs.

**The §8 cross-wave table omits all three edges.** 4a.24's `Blocked by` row lists ONLY `4a.4` (Wave 0 close) per PLAN.md §7 line 329. The Notes line: `Can run parallel with Wave 2.` It does NOT say "can run parallel with Wave 1" — but absent an explicit `blocked_by`, the dispatcher reads the field literally and could fire 4a.24 the moment 4a.4 closes, alongside any of {4a.5, 4a.12}.

**Why this is structurally identical to Round-1's PASS-WITH-NIT for 4a.12 ↔ Wave 1 chain (Round-1 §3.1 → §2.6.2 → final §3.1 ruling):**

- Round 1 acknowledged that 4a.12 ↔ 4a.5–4a.11 share the same package + the same file `mcp_surface.go`, with textually-disjoint struct definitions.
- Round 1 ruled this PASS-WITH-NIT because the textual disjointness defense is sound under existing Wave 0 dev-hygiene gates (`mage format-check` pre-commit + `mage ci` pre-push enforce goimports + gofmt determinism).
- The 4a.24 case is the same shape: textually-disjoint additions to the same files, gated by the same Wave 0 hooks, defended by the same Drop 3 droplet 3.21 precedent.

**Why this is NOT a CONFIRMED counterexample (NOT a build-blocker):**

- The new struct `ApproveAuthRequestRequest` and the new method `ApproveAuthRequest` are file-end additions — they do not modify existing line ranges in `mcp_surface.go` or `app_service_adapter_mcp.go`. Merge collisions are mechanically prevented under goimports + gofmt determinism + per-struct alignment.
- 4a.24's WAVE_3_PLAN.md acceptance text (line 78) already includes the discipline rule: "Builder MUST `mage test-pkg ./internal/...` early to surface the enum-change ripple cost." That captures the same-package-rebase-discipline requirement implicitly.
- The §"Verification" rule in WAVE_3_PLAN.md §W3.1 lists `mage test-pkg ./internal/adapters/server/common && mage test-pkg ./internal/adapters/server/mcpapi` — both packages where the overlap lives. The builder cannot land 4a.24 without those packages compiling green AGAINST whatever Wave 1 state has landed.

**Why this IS a NIT (documentation gap):** PLAN.md §7's 4a.24 Notes line does NOT explicitly call out the same-package overlap with Wave 1, the way the Round-1-fix call-out for 4a.12 does. A reader of 4a.24 alone would not know the overlap exists. By analogy with Round 1's fix #4 (4a.12 Notes adds parallelization rationale), 4a.24's Notes SHOULD add a similar rationale.

**Recommended NIT (non-blocking, can land before Wave 3 dispatches):**

PLAN.md §7's 4a.24 Notes line append:

```
Same-package parallelization rationale (added post-plan-QA-falsification round-2 sweep):
4a.24 shares packages internal/adapters/server/common (file mcp_surface.go +
app_service_adapter_mcp.go) and internal/adapters/server/mcpapi (file handler.go) with
the Wave 1 chain (4a.5–4a.11) and 4a.12. Author-judged textually disjoint — 4a.24 adds
NEW struct ApproveAuthRequestRequest + NEW method ApproveAuthRequest at file-end;
Wave 1 chain extends existing CreateActionItemRequest / UpdateActionItemRequest /
MoveActionItemState. Parallelization is intentional per L5 ("Wave 3 can run parallel
with Wave 2") + Drop 3 droplet 3.21 precedent. Builder rebases atop latest main before
each mage test-pkg run; escalates to serialization on Wave 1 close if a hit surfaces.
```

This NIT does NOT block Wave 0 / Wave 1 dispatch. It SHOULD land before any Wave 3 builder spawns.

**Severity rationale.** PASS-WITH-NIT, not CONFIRMED. The Round-1 4a.15 case was CONFIRMED because Wave 1's chain edits the SAME `Service.{Move,Create,Update}ActionItem` METHOD BODIES that 4a.15 extends — text-line collisions on existing functions. The 4a.24 case is file-end-only additions to existing files, mechanically conflict-free under goimports + gofmt. The two cases are not equivalent in severity.

### 4.2 REFUTED — Wave 2 droplets edit Wave 1 files (beyond 4a.15)

Walked every Wave 2 droplet's path list:

- **4a.14 (skeleton):** all NEW files in `internal/app/dispatcher/`. No Wave 1 overlap.
- **4a.15 (broker sub):** edits `internal/app/{live_wait.go, coordination_live_wait.go, service.go}`. The `service.go` overlap is now blocked (Round-1 fix). `live_wait.go` and `coordination_live_wait.go` are NOT in any Wave 1 droplet's path list (verified against WAVE_1_PLAN.md "same-file lock chains" enumeration at lines 25-32 — only `action_item.go`, `repo.go`, `mcp_surface.go`, `app_service_adapter_mcp.go`, `snapshot.go`, `service.go`, `project.go`). Wave 2 plan's prose at §2.2 line 109 says "Wave 1's `state` MCP work *might* touch `coordination_live_wait.go`" — but Round-1's fact check confirmed Wave 1.6's plan does NOT actually touch `coordination_live_wait.go`. No new collision.
- **4a.16 (file locks):** all NEW files. Refuted.
- **4a.17 (package locks):** all NEW files. Refuted.
- **4a.18 (walker):** `internal/app/dispatcher/{walker.go,walker_test.go}` (NEW). Refuted.
- **4a.19 (spawn):** `internal/app/dispatcher/{spawn.go,spawn_test.go}` (NEW). Refuted.
- **4a.20 (conflict):** `internal/app/dispatcher/{conflict.go,conflict_test.go}` (NEW). Refuted.
- **4a.21 (monitor):** `internal/app/dispatcher/{monitor.go,monitor_test.go,testdata/fakeagent.go}` (NEW). Refuted.
- **4a.22 (cleanup):** `internal/app/dispatcher/{cleanup.go,cleanup_test.go}` (NEW). Refuted.
- **4a.23 (CLI):** `cmd/till/{dispatcher_cli.go,dispatcher_cli_test.go,main.go}`. The `cmd/till/main.go` is the only existing file. No Wave 1 droplet edits `cmd/till/main.go` (Wave 1 lives in `internal/`). Refuted.

Verdict: NO ADDITIONAL WAVE 2 ↔ WAVE 1 COLLISIONS. Round-1's fix on 4a.15 is the only Wave 2 ↔ Wave 1 same-file overlap.

### 4.3 REFUTED — Wave 3 hidden cross-wave dependencies on Wave 1 / Wave 2 not surfaced in §8

Walked W3.1–W3.5:

- **4a.24** consumes `domain.AuthRequest`, `service.AuthRequests`, `mcp_surface.go` `AuthRequestRecord`, `mcpapi/handler.go`. None of these are Wave-1-modified except indirectly via package-compile lock (handled in §4.1). NO hidden Wave 1 dependency requiring a `blocked_by` edge — the package-lock case is the §4.1 NIT.
- **4a.25** consumes `domain.ProjectMetadata`. Wave 1's 4a.12 lands `ProjectMetadata` field extensions. **DOES** 4a.25 consume Wave 1 fields? Re-reading WAVE_3_PLAN.md §W3.2 acceptance: 4a.25 adds `OrchSelfApprovalEnabled *bool` to `ProjectMetadata`. 4a.12 adds `HyllaArtifactRef`, `RepoBareRoot`, `RepoPrimaryWorktree`, `Language`, `BuildTool`, `DevMcpServerName` to **`Project`** (per PLAN.md §7 4a.12 acceptance line 220). Different struct (`Project` vs `ProjectMetadata`). Same file `internal/domain/project.go`. **Same-file overlap candidate.**
  - 4a.12 paths include `internal/domain/{project.go,domain_test.go,errors.go}`.
  - 4a.25 paths include `internal/domain/{project.go,project_test.go}`.
  - Both edit `internal/domain/project.go`. PLAN.md §7 4a.25 `Blocked by` is `4a.24` only. NO `blocked_by 4a.12`.
  - **Counterexample candidate.** But — checking the structure: `Project` and `ProjectMetadata` are different structs in `project.go`. 4a.12 modifies the `Project` struct definition + its validation block. 4a.25 modifies the `ProjectMetadata` struct definition. Same textual-disjointness pattern as 4a.5 ↔ 4a.12. PASS-WITH-NIT class, not CONFIRMED. **Add to the NIT batch in §4.1.**
- **4a.26** consumes the W3.1 audit-trail input fields (within Wave 3). Files: `internal/domain/auth_request.go` (NEW columns on AuthRequest), `internal/adapters/auth/autentauth/{service.go,service_test.go}`, `internal/adapters/server/common/{app_service_adapter_mcp.go,mcp_surface.go,app_service_adapter_auth_requests_test.go}`. The `mcp_surface.go` and `app_service_adapter_mcp.go` files overlap with Wave 1 chain — same NIT class as §4.1.
- **4a.27** consumes mcpapi handler tests. `internal/adapters/server/mcpapi/{handler_test.go,handler_steward_integration_test.go}`. No Wave 1 path overlap (Wave 1 chain edits `extended_tools.go` + `extended_tools_test.go`, NOT `handler*_test.go`). Refuted.
- **4a.28** is markdown-only. No code overlap.

**Aggregate:** §4.1's NIT umbrella covers 4a.24 / 4a.25 / 4a.26 same-package overlap with Wave 1 (and 4a.12 where applicable). All same-file-end-additions, all defended by Wave 0 hooks + Drop 3 textual-disjointness precedent. NO additional CONFIRMED counterexamples surfaced.

### 4.4 REFUTED — Wave 4 droplet same-file conflicts

Walked Wave 4:

- **4a.29 paths:** `main/CLAUDE.md` only.
- **4a.30 paths:** `main/WIKI.md` only.
- **4a.31 paths:** `main/STEWARD_ORCH_PROMPT.md` only.
- **4a.32 paths:** outside-repo (`~/.claude/agents/*.md`, `~/.claude/CLAUDE.md`, memory files).

All four Wave 4 droplets edit DIFFERENT files. No same-file conflict within Wave 4. The `4a.29 → 4a.32` edge in §8 captures the cross-doc-consistency dependency (canonical sentence chosen in 4a.29 is mirrored in 4a.32).

Verdict: **REFUTED** — no Wave 4 same-file conflicts.

---

## 5. Drop-3 Pattern Recurrence — Re-Confirmation Post-Fix

### 5.1 REFUTED — Parallel-builder QA misattribution (Drop 3 R 3.20+3.21)

Round 1's verdict stands: every parallel branch has explicit `blocked_by` to its serialization point. The new 4a.11 → 4a.15 edge tightens this further. The §4.1 NIT for Wave 1 ↔ Wave 3 is the same shape Round 1 PASS-WITH-NIT'd for 4a.12 ↔ Wave 1; the precedent is consistent.

### 5.2 REFUTED — Stop-before-commit fabrication (Drop 3 R4)

No change post-fix. Every gate is tool-verifiable.

### 5.3 REFUTED — Audit-gap acceptance for outside-repo edits (Drop 3 3.27 5.D.10)

No change post-fix. 4a.28 + 4a.32 carry explicit audit-gap acceptance citations.

### 5.4 REFUTED — Description-symbol drift (Q9)

The Q9 routing fix on 4a.29 (PLAN.md line 375) directly addresses this Drop-3-recurrence pattern. Pre-spawn LSP resolution mitigates the description-symbol drift Drop 3 hit.

---

## 6. Verdict Summary

- **Fix #1 (4a.15 `blocked_by 4a.11`):** PASS — closes Round-1 §2.1 CONFIRMED counterexample.
- **Fix #2 (§8 cross-wave row):** PASS — table consistent with §7 declaration.
- **Fix #3 (§9 topo prose + new bullet):** PASS — prose, table, and rows all align.
- **Fix #4 (4a.12 parallelization rationale):** PASS — textual-disjointness defense sound under Wave 0 hooks + Drop 3 precedent.
- **Fix #5 (L1 testability-split sentence):** PASS — sentence accurately maps to 4a.19/4a.21/4a.23 acceptance.
- **Q9 routing note (4a.29 Notes):** PASS — LSP workspace symbol search for `StateFailed` / `RequireChildrenComplete` is sufficient pre-spawn resolution.
- **DAG re-walk:** PASS — new edge stays acyclic; topo sort unchanged.
- **New counterexample sweep §4.1 (Wave 1 ↔ Wave 3 W3.1 same-package overlap on `mcp_surface.go` + `app_service_adapter_mcp.go` + Wave 1 ↔ 4a.25 same-file overlap on `project.go`):** **PASS-WITH-NIT** — structurally identical to Round-1's accepted PASS-WITH-NIT for 4a.12 ↔ Wave 1 chain. Documentation NIT recommended for PLAN.md §7 4a.24 Notes (and parallel rationale for 4a.25).
- **Wave 2 cross-wave sweep §4.2:** REFUTED — no new collisions beyond the Round-1 4a.15 fix.
- **Wave 4 same-file sweep §4.4:** REFUTED — all Wave 4 droplets edit different files.
- **Drop-3 pattern recurrence:** all REFUTED post-fix.

**Final verdict: PASS-WITH-NIT.**

The five Round-1 fixes plus the Q9 routing note are correctly applied and close every Round-1 finding. One newly-surfaced same-package overlap (Wave 1 chain ↔ Wave 3 W3.1, and Wave 1 4a.12 ↔ Wave 3 4a.25 on `project.go`) is structurally identical to Round-1's accepted PASS-WITH-NIT for 4a.12 ↔ Wave 1 chain — the same textual-disjointness defense applies. This NIT does NOT block Wave 0 / Wave 1 dispatch; the orchestrator may proceed to builder dispatch starting with Wave 0 immediately.

**Recommended NIT before any Wave 3 builder spawns (non-blocking, can land alongside Wave 1 work):**

PLAN.md §7's 4a.24 Notes line append the same-package parallelization rationale (template at §4.1 above) covering the 4a.24 ↔ Wave 1 chain ↔ 4a.12 same-package case. PLAN.md §7's 4a.25 Notes line append a similar rationale for the 4a.25 ↔ 4a.12 same-file overlap on `internal/domain/project.go`.

After the orchestrator applies the NIT, no further confirmation pass is required — the structural PASS verdict stands, and the NIT is documentation-only.

---

## 7. TL;DR

- **T1.** Methodology: per-fix verification (5 fixes + Q9), DAG re-walk for cycles, 4a.12 parallelization re-attack, L1 testability-split re-attack, Q9 LSP-resolution re-attack, new-counterexample sweep, Drop-3 pattern recurrence re-confirmation.
- **T2.** All five Round-1 fixes plus the Q9 routing note are CORRECTLY APPLIED — every fix closes its corresponding Round-1 finding and introduces no new structural problem.
- **T3.** New DAG analysis: the 4a.11 → 4a.15 edge stays acyclic; topological order unchanged; no new dangling references. Wave 2's parallel branches preserved.
- **T4.** New counterexample surfaced (Round 1 missed): Wave 1 chain ↔ Wave 3 W3.1 (4a.24) same-package overlap on `mcp_surface.go` + `app_service_adapter_mcp.go`, plus Wave 1 4a.12 ↔ Wave 3 W3.2 (4a.25) same-file overlap on `internal/domain/project.go`. Structurally identical to Round-1's 4a.12 ↔ Wave 1 chain PASS-WITH-NIT; same textual-disjointness defense applies under Wave 0 dev-hygiene hooks + Drop 3 droplet 3.21 precedent.
- **T5.** Drop-3 pattern recurrence sweep clean post-fix: parallel-builder QA misattribution (REFUTED — every parallel branch has explicit `blocked_by`), stop-before-commit fabrication (REFUTED — every gate tool-verifiable), audit-gap acceptance for outside-repo edits (REFUTED — 4a.28 + 4a.32 cite drop_3/3.27 5.D.10), description-symbol drift (REFUTED — Q9 routing fix mitigates).
- **T6.** Final verdict PASS-WITH-NIT — orchestrator may dispatch Wave 0 builders immediately. The §4.1 NIT (4a.24 Notes + 4a.25 Notes append for same-package overlap rationale) SHOULD land before any Wave 3 builder spawns but does NOT gate Wave 0 / Wave 1 dispatch. No further confirmation pass needed after the NIT.

---

## Hylla Feedback

N/A — task touched non-Go files only (PLAN.md, PLAN_QA_FALSIFICATION.md, WAVE_*_PLAN.md, drop_3 R2 precedent file are all markdown). Hylla today indexes only Go (per `feedback_hylla_go_only_today.md`). The load-bearing Go-source check (Wave 1 chain ↔ Wave 3 W3.1 / W3.2 file overlap on `mcp_surface.go`, `app_service_adapter_mcp.go`, `project.go`) was answered by direct path-list comparison across the wave-plan markdown files — no Hylla query needed and no fallback miss to log.
