# DROP 4A — UNIFIED PLAN QA FALSIFICATION REVIEW

**Target:** `workflow/drop_4a/PLAN.md` (32 droplets, 5 waves)
**Reviewer mode:** filesystem-MD (read-only against PLAN; no code edits)
**Verdict:** **FAIL — one CONFIRMED counterexample requires plan revision before builder dispatch; one PASS-WITH-NIT-class observation; remaining attacks REFUTED.**
**Recommended action:** orchestrator amends PLAN.md §7 (4a.15 row) and §8 (cross-wave blocker table) per the fix in §2.1 below, then re-spawns this agent for a confirmation pass. Drop-3 pattern recurrence is clean. Open-questions Q1–Q12 are individually adjudicated in §4.

---

## 1. Methodology

Six attack vectors applied in order:

- **A1** — Droplet-with-children misclassification
- **A2** — Sibling path/package overlap without `blocked_by`
- **A3** — Empty-`blocked_by` confluence
- **A4** — Confluence with partial upstream coverage
- **A5** — Role / structural_type contradictions
- **A6** — §4.4 global L1 sweep (acyclicity / sibling-overlap / leaf-composes-into-L1 / orphan-droplet)

Then Drop-3-pattern-recurrence sweep (parallel-builder QA misattribution, stop-before-commit fabrication, audit-gap acceptance for outside-repo edits) and per-Q adjudication (Q1–Q12).

Evidence sourced from `PLAN.md`, `WAVE_0_PLAN.md` through `WAVE_4_PLAN.md`, `REVISION_BRIEF.md`, plus a directory listing of `internal/app/` to confirm `service.go` exists as a single shared file (the file the load-bearing counterexample turns on).

---

## 2. Counterexamples

### 2.1 CONFIRMED — 4a.15 edits `internal/app/service.go` AND the same package, but PLAN.md never blocks it on the Wave 1 chain that owns that file (A2 / A6.2 — sibling-overlap-without-blockers, cross-wave variant)

**The most damaging counterexample.** Walks the §"Project Structure" rule: every sibling sharing a `paths[]` entry OR a `packages[]` entry without an explicit `blocked_by` is a same-file / same-package race. This rule is what gates package-level locking (CLAUDE.md § "Cascade Tree Structure" → "Blocker Semantics"). Drop 4a's PLAN.md violates it cross-wave.

**Evidence:**

- **4a.15 (LIVEWAITBROKER SUBSCRIPTION)** PLAN.md §7 row lists Paths: `internal/app/dispatcher/{broker_sub.go,broker_sub_test.go}` (NEW), **`internal/app/{live_wait.go,coordination_live_wait.go,service.go}`**. Packages: `internal/app/dispatcher`, **`internal/app`**.
- **4a.15 acceptance** (cross-checked against WAVE_2_PLAN.md §2.2): "`Service.MoveActionItem`, `Service.CreateActionItem`, `Service.UpdateActionItem` each call `s.publishActionItemChanged(actionItem.ProjectID)` after a successful repo write." That edit is in **`internal/app/service.go`** — the same file Wave 1 droplets 4a.5–4a.11 each rewrite (Wave 1 acceptance: `CreateActionItemInput` / `UpdateActionItemInput` extension + thread-through service methods).
- **PLAN.md §7 4a.15 `Blocked by:` row** lists ONLY `4a.14` (the Wave 2 package skeleton). It does NOT list `4a.5`, `4a.6`, `4a.7`, `4a.8`, `4a.9`, `4a.10`, or `4a.11` — every one of which edits `internal/app/service.go` (and the `internal/app` package).
- **PLAN.md §8 (Cross-Wave Blocker Wiring)** does NOT carry any edge from Wave 1 to 4a.15. The section enumerates 4a.6→4a.18, 4a.6→4a.20, 4a.10→4a.18, 4a.11→4a.18, 4a.12→4a.19, 4a.12→4a.23, plus 4a.{4,11,12,13,23,28}→Wave-4 — but no `→4a.15` edge anywhere.
- **WAVE_2_PLAN.md §2.2 "Notes"** *itself surfaces this risk* but only as a "soft cross-wave coordination concern": "Wave 1's `state` MCP work *might* touch `coordination_live_wait.go`. Surface to plan-QA as a soft cross-wave coordination concern." The orchestrator at synthesis time misread "soft concern" as resolved — Wave 1.6's plan (WAVE_1_PLAN.md §1.6) does NOT touch `coordination_live_wait.go`, which is true, but the planner missed that **Wave 1.1–1.5 + 1.7 all touch `service.go`** (the entire Wave 1 input-struct chain), which 4a.15 also touches. That's the actual lock collision.

**Cases:**

- Case 1 (linear sequencing): orch dispatches 4a.5 (Wave 1.1) and 4a.15 (Wave 2.2) concurrently after 4a.4 + 4a.14 close. Both builders open `internal/app/service.go`. One commits first; the other lands a merge conflict on the same `Service` method body. This is the exact failure mode the file/package-blocking rule prevents.
- Case 2 (compile lock): even if file-disjoint within `service.go`, both builders modify the `Service` struct's method set in the same `internal/app` package — the second builder's `mage test-pkg ./internal/app` runs against an in-flight compile state from the first.
- Case 3 (test-fixture drift): Wave 1 droplets each adjust `service_test.go` fixtures around `CreateActionItemInput` / `UpdateActionItemInput`. 4a.15's tests in `broker_sub_test.go` instantiate a `*app.Service` and call `MoveActionItem` / `CreateActionItem` / `UpdateActionItem` — every Wave 1 input-struct extension changes the fixture-constructor call shape. With no `blocked_by` between the chains, 4a.15's tests can be authored against a stale `Input` shape that breaks the moment Wave 1.1 lands.

**Required fix:**

PLAN.md §7's 4a.15 `Blocked by:` must list **`4a.11`** (the terminal node of the Wave 1 `service.go` chain — once 4a.11 closes, every prior link 4a.5→4a.10 is also closed). Alternatively, list **`4a.5, 4a.6, 4a.7, 4a.8, 4a.9, 4a.10, 4a.11`** explicitly to mirror PLAN.md's normal cross-wave precision (e.g. `4a.10` blocks `4a.18` is listed even though `4a.11`'s parent-block invariant subsumes part of `4a.10`'s state-on-MCP work).

PLAN.md §8 must add a row:

```
| 4a.11 | 4a.15 (broker sub) | Same-file lock on internal/app/service.go (Service.{Move,Create,Update}ActionItem publish) + package compile lock on internal/app. |
```

PLAN.md §9 (Topological Order) must update the Wave 2 chain header. Currently:

```
4a.14 → 4a.15 → 4a.18    (broker → walker, after 4a.10/4a.11 land)
```

The "after 4a.10/4a.11 land" parenthetical implies 4a.18 needs them but is silent on 4a.15. Update to:

```
4a.14 → 4a.15 (after 4a.11 lands — same-file lock on service.go) → 4a.18    (broker → walker)
```

Severity: **build-blocker.** Without this fix, the dispatcher cannot dispatch 4a.5 and 4a.15 concurrently safely, but the §8 graph as drafted permits exactly that. CONFIRMED.

---

### 2.2 REFUTED — Droplets-with-children (A1)

Walked the 32-droplet matrix. Every leaf droplet (4a.1–4a.32) is `structural_type=droplet` per filesystem-MD mode. None spawn child plan items in their acceptance criteria — the closest is 4a.32's "builder spawns subagent" recommendation (Wave 4 builder-driven), but that's a build-time spawn, not a child plan item creation. The cascade tree is correctly leaf-shaped.

Note: PLAN.md is filesystem-MD (no Tillsyn runtime per-droplet plan items), so `structural_type` is implicit rather than declared per node — but the DAG semantics are equivalent. Attack does not land.

---

### 2.3 REFUTED — Empty-`blocked_by` confluence (A3)

Drop 4a's filesystem-MD mode does not declare confluences explicitly, but the cross-wave Wave 4 droplets (4a.29, 4a.30, 4a.31, 4a.32) act like confluences (multiple upstream waves feed in). Walked each:

- **4a.29** `Blocked by:` = {4a.11, 4a.12, 4a.13, 4a.23, 4a.28} — non-empty, complete (Wave 1 + Wave 2 + Wave 3 closure).
- **4a.30** `Blocked by:` = {4a.11, 4a.12, 4a.13, 4a.23, 4a.28} — non-empty, complete.
- **4a.31** `Blocked by:` = {4a.28} — non-empty (Wave 3 only, intentional — STEWARD prompt is auth-flow scoped per WAVE_4_PLAN.md §W4.3 rationale).
- **4a.32** `Blocked by:` = {4a.28, 4a.29} — non-empty.

No empty confluence. Attack does not land.

---

### 2.4 REFUTED — Confluence with partial upstream coverage (A4)

The strongest A4 candidate per the prompt: 4a.29's prose says "Wave 1 close, Wave 2 close, Wave 3 close." Wave 1 has THREE terminal nodes (4a.11 ending the action_item.go chain; 4a.12 the parallel project-fields branch; 4a.13 the parallel column-verify branch). Wave 2's terminal is 4a.23 (CLI). Wave 3's terminal is 4a.28.

PLAN.md §7's 4a.29 row lists `Blocked by: **4a.11**, **4a.12**, **4a.13** (Wave 1 close — three terminal nodes), **4a.23** (Wave 2 close), **4a.28** (Wave 3 close).` All three Wave-1 terminals are explicitly enumerated. Same for 4a.30. Coverage is complete — the "three terminal nodes" footnote is verbatim acknowledged in the row's Reason field.

Attack does not land. (Notable: the orchestrator preempted exactly this attack in the synthesis step.)

---

### 2.5 REFUTED — Role / structural_type contradictions (A5)

Drop 4a is filesystem-MD mode (no Tillsyn runtime, no `metadata.role` field today). Roles are surfaced per WAVE_4_PLAN.md §"Orchestrator-Driven vs Builder-Driven Per Droplet" table:

- **Orch-driven** (no builder spawn): 4a.29, 4a.30, 4a.31. Each is a small surgical MD edit (~3–10 hits in a single doc).
- **Builder-driven**: 4a.32 (cross-doc consistency + 7 outside-repo files), and implicitly all 4a.1–4a.28 (real code work).

No `qa-proof` or `qa-falsification` agent is assigned to a non-droplet parent (everything is a droplet here). No `builder` is assigned to a confluence-shaped item (Wave 4 closeout droplets are docs-only, not confluences). No `commit` kind appears (Drop 4a defers commit cadence to Drop 4b per L7).

Attack does not land.

---

### 2.6 §4.4 Global L1 Plan-QA Sweep

#### 2.6.1 REFUTED — Blocker-graph acyclicity

Walked PLAN.md §8 + every per-droplet `Blocked by:` row. Topological sort succeeds (PLAN.md §9 demonstrates one valid order). All Wave 1 edges flow upward; Wave 2 edges flow only from Wave 1 + within Wave 2; Wave 3 edges flow within Wave 3; Wave 4 edges flow from Wave 1 + Wave 2 + Wave 3 + within Wave 4 (W4.4 ← W4.1 only). No cycle exists.

The fix from §2.1 above (add 4a.11 → 4a.15) does NOT introduce a cycle: 4a.11 already has no outgoing edges to anything in Wave 2, and 4a.15's existing outgoing edge is to 4a.18 only. Adding 4a.11 → 4a.15 stays acyclic.

#### 2.6.2 CONFIRMED — Sibling-overlap-without-blockers

See §2.1 above (4a.15 ↔ Wave 1 chain). One CONFIRMED instance. No others found. Walked:

- **Wave 1 `action_item.go` chain (4a.5–4a.11):** every droplet `blocked_by` the prior — chain is fully serialized.
- **Wave 2 dispatcher package (4a.14–4a.23):** each droplet `blocked_by` 4a.14 plus the specific upstream droplets it consumes. No sibling pair within Wave 2 shares a NEW source file (each droplet creates its own `<name>.go` + `<name>_test.go`). Cross-droplet edits to existing `internal/app/` files: only 4a.15 (per §2.1) — and that's the leak.
- **Wave 3 (`app_service_adapter_mcp.go` triplet 4a.24/4a.25/4a.26):** PLAN.md §7 strict-linear chain (4a.24 → 4a.25 → 4a.26) explicitly serializes them on the same file. Correct.
- **Wave 4:** all four droplets file-disjoint (4a.29 ↔ CLAUDE.md, 4a.30 ↔ WIKI.md, 4a.31 ↔ STEWARD_ORCH_PROMPT.md, 4a.32 ↔ outside-repo). 4a.32 → 4a.29 cross-doc consistency edge is explicit.

Only one violation; documented at §2.1.

#### 2.6.3 PASS-WITH-NIT — Leaf acceptance criteria compose into L1 outcome

Drop 4a's L1 outcome (PLAN.md §1): "Replace the orchestrator-as-dispatcher loop with a programmatic dispatcher … `till dispatcher run --action-item <id>` reads template `agent_bindings`, acquires file/package locks, walks tree eligibility, spawns subagents via `claude --agent`, and provisions auth via the new orch-self-approval flow."

Mapped each L1 promise → owning droplet:

| L1 promise | Owning droplet | Notes |
| --- | --- | --- |
| New `internal/app/dispatcher/` package | 4a.14 | Skeleton + interface |
| `till dispatcher run --action-item <id>` CLI | 4a.23 | Manual-trigger CLI |
| Reads template `agent_bindings` | 4a.19 | `catalog.LookupAgentBinding` |
| Acquires file locks | 4a.16 | `fileLockManager` |
| Acquires package locks | 4a.17 | `packageLockManager` |
| Walks tree eligibility | 4a.18 | `treeWalker.EligibleForPromotion` |
| Spawns subagents via `claude --agent` | 4a.19 | `BuildSpawnCommand` |
| Provisions auth via orch-self-approval | 4a.24, 4a.27 | Domain + MCP wiring |

All eight promises have an owning droplet. Acceptance criteria each compose upward to the L1 promise. **NIT (non-blocking):** L1 says "spawns subagents via `claude --agent`," but 4a.19's acceptance is `BuildSpawnCommand` (constructs `*exec.Cmd`, does NOT execute) and 4a.21 owns execution + monitoring. The CLI (4a.23) is what stitches construction → execution. This is not a defect — it's the testability split surfaced in Q3 — but a reader of L1 alone might assume "spawns" means a single droplet. The split is documented in WAVE_2_PLAN.md §2.6 + §2.8 + Q3. NIT is a description-clarity polish for L1, not a planning hole.

#### 2.6.4 REFUTED — Orphan-droplet check

Every droplet has at least one upstream `blocked_by` (entry-point droplet 4a.1 is blocked-by-nothing per L6 anchor, which is correct — Wave 0 starts the chain). Every droplet has at least one downstream consumer:

- 4a.1 → 4a.2; 4a.2 → 4a.3; 4a.3 → 4a.4; 4a.4 → many.
- Every Wave 1 droplet → at least one Wave 2 / Wave 4 droplet.
- Every Wave 2 droplet → 4a.23 (CLI is the wave's confluence).
- Every Wave 3 droplet → 4a.27 (golden tests exercise W3.1–W3.3) → 4a.28 (docs flip).
- Every Wave 4 droplet either is a wave-terminal (4a.32) or is consumed by drop closeout.

No orphans.

---

## 3. Drop-3 Pattern-Recurrence Findings

### 3.1 REFUTED — Parallel-builder QA misattribution (Drop 3 R 3.20+3.21 commit split)

The PLAN explicitly serializes parallel branches via `blocked_by`. The two same-file/same-package-touching parallel branches I checked:

- **Wave 1 parallel:** 4a.5–4a.11 (action_item.go chain), 4a.12 (project.go chain), 4a.13 (column verify). Each branch edits a disjoint primary file (`action_item.go` vs `project.go` vs `service.go`-verify-only). 4a.12 DOES touch `mcp_surface.go` and `extended_tools.go` — same files as the 4a.5–4a.11 chain. **This is a second potential A2 violation** — but checking PLAN.md §7's 4a.12 row, Packages list overlaps `internal/adapters/server/common` and `internal/adapters/server/mcpapi` with the Wave-1 action-item chain. **Wait** — this needs closer inspection.

Re-reading 4a.12 acceptance: it adds `CreateProjectRequest` / `UpdateProjectRequest` fields in `mcp_surface.go` and `extended_tools.go`. Those are **DIFFERENT struct types** from `CreateActionItemRequest` / `UpdateActionItemRequest` that 4a.5–4a.11 extend. Same files, but disjoint struct surfaces — Go's lexical scoping makes the edits non-overlapping at the line level.

However, **both chains compile against the same `mcp_surface.go` package compile unit**. If 4a.5 and 4a.12 dispatch concurrently, both run `mage test-pkg ./internal/adapters/server/common` against an in-flight `mcp_surface.go`. This IS a package-lock collision per CLAUDE.md "Cascade Tree Structure" → "Blocker Semantics" → "File- and package-level blocking" rule (which says: "sibling build-tasks sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them").

Checking PLAN.md §7: 4a.12's `Blocked by` is `4a.4` (Wave 0 close). 4a.5's `Blocked by` is `4a.4`. They are siblings sharing `internal/adapters/server/common` package and the file `mcp_surface.go`. **No `blocked_by` edge exists between them.** The PLAN's §"Notes" on 4a.12 explicitly says: "**Independent of 4a.5–4a.11; runs parallel.**" That's the assertion under attack.

Possible mitigation: if Wave 1.8's edits to `mcp_surface.go` are textually disjoint from Wave 1.1–1.7's edits (different struct definitions, far apart in the file), parallel work is mechanically possible — but the package compile lock still forces serialization at `mage test-pkg` time, and both branches need to rebase past each other's commits. **This is a CONFIRMED secondary counterexample candidate.** However:

- WAVE_1_PLAN.md §1 lines 25–32 explicitly enumerates the same-file lock chain and lists `mcp_surface.go` as touched by "Wave 1.1–1.6" (action-item branch). Wave 1.8 (project-fields branch) is listed separately as touching `project.go` + `repo.go` + `mcp_surface.go` + `app_service_adapter.go` (note: `app_service_adapter.go`, not `app_service_adapter_mcp.go`).
- **WAVE_1_PLAN.md §"Wave-Internal Sequencing"** (line 304) explicitly says: "1.8 + 1.9 parallel from any point post-Wave-0."
- The parallelization is **author-intentional** with the textual-disjointness rationale documented (§"Decomposition" line 56–58: "Wave 1.8 (project fields) is independent of `action_item.go` work and parallelizes against Wave 1.1").

**Adjudication:** this is a **PASS-WITH-NIT, not a CONFIRMED counterexample.** The plan author considered the same-package question and concluded textual disjointness suffices. Given:

1. `CreateProjectRequest` / `UpdateProjectRequest` are distinct struct definitions in `mcp_surface.go` from `CreateActionItemRequest` / `UpdateActionItemRequest`.
2. The two builders only need to rebase past each other's package-level changes (no manual conflict resolution).
3. Wave 1.8 carries `Irreducible: true` rationale that explicitly cites Drop 3 droplet 3.21's same-pattern precedent.

The package-lock rule as drafted in CLAUDE.md is conservative; the author's parallelization stance is defensible. **NIT recommendation:** PLAN.md §7's 4a.12 Notes line should add: "Same package as 4a.5–4a.11 (`internal/adapters/server/common`, `internal/adapters/server/mcpapi`); textual disjointness on `mcp_surface.go` + `extended_tools.go` is the parallelization rationale. Builders must rebase atop each other's commits before each `mage test-pkg` run." This documents the trade-off rather than letting it remain implicit.

### 3.2 REFUTED — Stop-before-commit fabrication (Drop 3 R4 builder hallucinated tool calls)

Every per-droplet acceptance criteria in WAVE_0/1/2/3/4 explicitly lists `mage test-pkg <pkg>` per package, plus `mage ci` at droplet/wave end. Per-droplet verification gates are concrete and testable. WAVE_4 droplets (docs-only) explicitly require `mage ci` smoke + `Grep` post-condition checks (e.g. `Grep "metadata\.paths"` returns zero hits after edit) — these are tool-verifiable post-conditions, not narrative assertions.

The plan does not encode "and the builder reports success" as an acceptance criterion anywhere; every gate is a tool invocation with a deterministic exit/grep result.

### 3.3 REFUTED — Audit-gap acceptance for outside-repo edits (Drop 3 3.27 finding 5.D.10)

4a.28 (Wave 3) and 4a.32 (Wave 4) both contain explicit `Paths (outside-repo, audit-gap accept):` lines naming each non-tracked file (`~/.claude/agents/*.md`, memory files, `~/.claude/CLAUDE.md`). Each row carries an acceptance bullet citing drop_3/3.27 finding 5.D.10 by reference. Worklog-only audit trail is acknowledged as the trade-off, not silently absorbed. Q10 in PLAN.md §10 reconfirms acceptance.

---

## 4. Open-Questions Adjudication (PLAN.md §10)

For each, route is either **MITIGATE** (requires plan revision) or **ACCEPT** (route to plan-QA-falsification artifact as documented decision).

- **Q1 — Wave 1 stuck-parent failure mode (4a.11).** **ACCEPT.** Documented pre-MVP cost; dev fresh-DBs is the documented escape valve (CLAUDE.md "no migration logic pre-MVP" rule + WAVE_1_PLAN.md §1.7 Notes). Supersede CLI is post-MVP; until then, the failure mode is a recoverable nuisance, not a correctness hole. Plan-QA accepts.
- **Q2 — Wave 2 same-package vs sub-package locks.** **ACCEPT.** Author's stance defensible per memory `feedback_decomp_small_parallel_plans.md`'s "decomp into small parallel planners" — sub-packages would add navigation cost for 10 files in tight cohesion. WAVE_2_PLAN.md §"File layout decision" rationale stands.
- **Q3 — Wave 2 cmd-construction vs cmd-execution split (4a.19 vs 4a.21).** **ACCEPT** with NIT to L1 description (see §2.6.3). The split is testability-driven; testing 4a.19 without `claude` on PATH requires not-executing.
- **Q4 — Wave 2 auth-bundle stub seam (4a.19).** **ACCEPT.** WAVE_2_PLAN.md §2.6 + WAVE_3_PLAN.md §W3.1 both name the seam explicitly. The stub interface (`AuthBundle` zero-value + placeholder `--mcp-config` path) is sufficient for Wave 3 to plug into; Wave 3's W3.1 acceptance plumbs the four approver-identity fields back into `BuildSpawnCommand`'s call site. Verified seam alignment.
- **Q5 — Wave 2 `metadata.failure_reason` shape.** **ACCEPT.** Free-form for 4a; Drop 4b refactors. WAVE_2_PLAN.md §2.8 + Q5 explicitly defer the structured `failure` type. No falsification attack lands — the field is read by the orchestrator (human-readable string) in 4a, not parsed.
- **Q6 — Wave 2 test-helper `go build` carve-out.** **ACCEPT.** Precedent in `cmd/till/main_test.go` (per WAVE_2_PLAN.md §2.8 Notes). The carve-out is documented in PLAN.md §3 ("Test-helper fakeagent compilation in Wave 2.8 is a documented carve-out") + REVISION_BRIEF §"Pre-MVP rules in force" exception. Build-tag `//go:build ignore` excludes the file from the main package compile. Falsification attack: "this normalizes raw `go build` invocation" — REFUTED, the carve-out is bounded by build-tag + test-only context.
- **Q7 — Wave 2 CLI bootstrap symmetry with daemon.** **ACCEPT.** WAVE_2_PLAN.md §2.10 + Q7 author's stance: "minimal CLI today; 4b refactors." Forward-engineering for the daemon variant in 4a is YAGNI — Drop 4b will land it in one place.
- **Q8 — Wave 3 4a.24 scope size (~150 LOC across 5 packages).** **ACCEPT.** PLAN.md §7's 4a.24 Notes explicitly directs the builder: "MUST run package-by-package `mage test-pkg` early to surface enum-change ripple." Builder discipline mitigates risk. Splitting per-package would create same-file-lock + import-graph contention identical to Wave 1.8's "bundled" rationale (§W3.1 Notes line 78). Single droplet stands.
- **Q9 — Wave 4 4a.29 description-symbol verification.** **ACCEPT** with reinforcement. PLAN.md §7's 4a.29 Notes: "Description-symbol verification required before writing post-Drop-4a sentences naming code symbols." This is the load-bearing discipline rule; orch (or builder when 4a.29 escalates) MUST run `LSP findReferences` on `RequireChildrenComplete` / `StateFailed` / etc. before writing. Risk is real but mitigation is named.
- **Q10 — Wave 4 4a.32 outside-repo audit-gap.** **ACCEPT.** Reconfirmed per drop_3/3.27 5.D.10 (see §3.3 above).
- **Q11 — Wave 1.7 `failed` children block parent close.** **ACCEPT.** Same answer as Q1; correct semantics per L2 (failed is a terminal-non-success state, not a closeable state). Drop 1's always-on-parent-block invariant rules out the alternative interpretation. WAVE_1_PLAN.md §1.7 acceptance pins it explicitly with four test cases (complete / in_progress / failed / archived).
- **Q12 — Wave 1.6 column_id back-compat.** **ACCEPT.** Both `state` and `column_id` accepted; "specify exactly one" rejection is the disambiguator. The dual-acceptance is load-bearing for the columns table that survives until Drop 4.5's TUI overhaul (L1). Removing `column_id` acceptance now would force every existing TUI drag-and-drop call site to migrate before Drop 4.5 lands — premature. PLAN.md §7's 4a.10 Notes documents the deferral correctly.

All twelve open questions are ACCEPTED with reasoning; none require plan revision.

---

## 5. Verdict Summary

- **A1 (droplet-with-children):** REFUTED. Tree is leaf-shaped.
- **A2 (sibling overlap without `blocked_by`):** **CONFIRMED — 4a.15 ↔ Wave 1 service.go chain.** See §2.1.
- **A3 (empty-`blocked_by` confluence):** REFUTED.
- **A4 (confluence with partial upstream coverage):** REFUTED. 4a.29/4a.30 list all three Wave-1 terminals.
- **A5 (role/structural_type contradictions):** REFUTED.
- **A6.1 (acyclicity):** PASS.
- **A6.2 (sibling overlap):** see A2 (one CONFIRMED, one PASS-WITH-NIT for 4a.12 ↔ 4a.5–4a.11 same-package-textual-disjointness).
- **A6.3 (leaf composes into L1):** PASS-WITH-NIT (description-clarity NIT on L1 vs 4a.19/4a.21/4a.23 cmd-construction/execution split).
- **A6.4 (orphan droplets):** PASS.
- **Drop-3 pattern recurrence (parallel-builder QA, stop-before-commit, audit-gap):** all REFUTED.
- **Open questions Q1–Q12:** all ACCEPTED.

**Final verdict: FAIL.** The single CONFIRMED counterexample at §2.1 is a build-blocker — without the fix, the dispatcher (or pre-cascade orchestrator simulating it) can dispatch 4a.5 and 4a.15 concurrently and one builder will land a same-file-lock collision on `internal/app/service.go`. Plan must be amended before any builder spawns.

**Required fix (recap from §2.1):**

1. PLAN.md §7 — 4a.15 row `Blocked by:` add `4a.11`.
2. PLAN.md §8 — add row `4a.11 → 4a.15 | Same-file lock on internal/app/service.go (Service.{Move,Create,Update}ActionItem publish) + package compile lock on internal/app.`
3. PLAN.md §9 — Wave 2 chain header: `4a.14 → 4a.15 (after 4a.11 lands — same-file lock on service.go) → 4a.18`.

**Recommended NIT (PASS-WITH-NIT, not blocking):**

4. PLAN.md §7 — 4a.12 Notes line append: "Same package as 4a.5–4a.11 (`internal/adapters/server/common`, `internal/adapters/server/mcpapi`); textual disjointness on `mcp_surface.go` + `extended_tools.go` is the parallelization rationale. Builders must rebase atop each other's commits before each `mage test-pkg` run."
5. PLAN.md §1 (Goal) — clarify "spawns subagents via `claude --agent`" splits across 4a.19 (construct) + 4a.21 (execute + monitor) + 4a.23 (CLI orchestration), per Q3 split rationale.

After fix #1–#3 lands, re-spawn this agent for a confirmation pass before any builder dispatches.

---

## 6. TL;DR

- **T1.** Methodology: applied 5 cascade-vocabulary attacks + §4.4 four-part L1 sweep + Drop-3 pattern recurrence + Q1–Q12 adjudication.
- **T2.** ONE CONFIRMED counterexample: 4a.15 (LIVEWAITBROKER) edits `internal/app/service.go`, the same file the entire Wave 1 chain (4a.5–4a.11) edits, with no `blocked_by` between them. Build-blocker; PLAN.md §7 + §8 + §9 require three explicit edits.
- **T3.** Drop-3 pattern recurrence sweep clean: parallel-builder QA misattribution refuted (Wave 1 chain serialized; 4a.12 textual-disjoint NIT); stop-before-commit fabrication refuted (every gate is tool-verifiable); audit-gap acceptance for outside-repo edits explicitly cited per drop_3/3.27 5.D.10.
- **T4.** Open questions Q1–Q12 all ACCEPTED with documented rationale; none require plan revision.
- **T5.** Final verdict FAIL — dispatch one fix round to the orchestrator (three edits to PLAN.md), then re-spawn this agent for confirmation. PASS-WITH-NIT pathway available if §2.1 fix lands cleanly and the §3.1 / §2.6.3 NITs are also addressed; otherwise PASS-after-fix.

---

## Hylla Feedback

N/A — task touched non-Go files only (PLAN.md + WAVE_*_PLAN.md + REVISION_BRIEF.md are all markdown). Hylla today indexes only Go (per `feedback_hylla_go_only_today.md`). The single Go-source check needed (verifying `internal/app/service.go` exists as a single shared file the load-bearing counterexample turns on) was answered by a directory listing of `internal/app/`, which surfaced `service.go` (81k, the largest file in the package) plus `live_wait.go` and `coordination_live_wait.go` directly. No Hylla query issued; no fallback miss to log.
