# Plan-QA-Proof Round 1 — DROP_4c.6.W0.5

**Reviewer:** go-qa-proof-agent (subagent, opus, plan-QA mode)
**Round:** 1
**Parent kind:** `plan` (sub-plan container `4c.6.W0.5`)
**L2 PLAN under review:** `workflow/drop_4c_6/DROP_4c.6.W0.5_TEMPLATE_VALIDATORS/PLAN.md`
**Sibling blocker ledger:** `workflow/drop_4c_6/DROP_4c.6.W0.5_TEMPLATE_VALIDATORS/_BLOCKERS.toml`
**L1 container ref:** `workflow/drop_4c_6/PLAN.md` lines 71-85
**Sketch source-of-truth:** `workflow/drop_4c_6/SKETCH.md` § 26.W0.5 + § 25.1

## 1. Findings

(none)

All five plan-QA-proof axes (atomic-decomposition, parallelization-graph, specify-block-well-formedness, multi-level-decomposition, shipped-but-not-wired) verified clean against the L2 PLAN.md content + `_BLOCKERS.toml` content + live state of `internal/templates/load.go` + `internal/templates/schema.go`. Detailed axis-by-axis evidence below.

### 1.A Atomic-decomposition — verified

Each of D1-D6 declares one validator + one or two fixture(s) + one table-driven test. Per-droplet sizing per `KindPayload` in PLAN.md:

- **D1 (`validateAgentMapKeys`)** — extension of existing `canonicalizeMapKeys` over a new `Template.Agents` map; ~30-50 LOC validator + ~30 LOC test + 2 small TOML fixtures (`valid_minimal.toml` + `invalid_agents_unknown_kind.toml`). Reuse path is concrete: `canonicalizeMapKeys` is generic over `V any` (verified at `internal/templates/load.go:462`). Within atomic budget.
- **D2 (`validateAgentBindingNames` + `ErrUnknownAgentName` + `LoadOptions.AgentLookupFn`)** — new validator + new sentinel + new injection point + 2 TOML fixtures + 1 table-driven test with 3 rows (known/unknown/empty). ~50-80 LOC validator + ~40-60 LOC test + ~30 LOC fixture. Within atomic budget.
- **D3 (extend `validateChildRuleCycles` + `formatCyclePath`)** — surgical extension of the existing colored-DFS at `load.go:517-561`; ~20-30 LOC modify + 2 TOML fixtures + 1 table-driven test with 4 rows. Reuse path is concrete (existing `formatCyclePath` at `load.go:566-580`). Within atomic budget.
- **D4 (`validateChildRuleRecursionDepth` + `ErrChildRuleRecursionTooDeep` + `childRuleRecursionDepthMax = 5`)** — new validator + new sentinel + new constant + reuses graph helper from D3 + 1 TOML fixture (6-deep chain `k0→…→k6`) + 1 table-driven test with 4 rows (depth 5/6/empty/single). ~40-60 LOC validator + ~40-60 LOC test + ~30 LOC fixture. Within atomic budget.
- **D5 (`validateBlockedByAcyclicity` + `ErrTemplateBlockedByCycle` + `LoadOptions.BlockedByGraphFn`)** — new validator + new sentinel + new test-injection point + extracts shared private `dfsDetectCycle` generic helper across D3/D5 + 1 TOML fixture + 1 table-driven test with 3 rows. ~40-70 LOC validator + ~40-50 LOC test + ~30 LOC fixture. Within atomic budget.
- **D6 (`validateClaimVsImplCoherence` + `ErrClaimVsImplUnknownConsumer` + `knownWiredConsumers` empty map + `LoadOptions.ClaimedConsumersFn`)** — new validator + new sentinel + Go-internal empty closed set + new injection point + 1 structurally-valid fixture + 1 table-driven test with 3 rows. ~30-50 LOC validator + ~40-60 LOC test + ~20 LOC fixture. Within atomic budget.

Aggregate sizing claim by sub-planner in `PLAN.md:250` ("~30-80 LOC validator + ~30-50 LOC test + ~10-20 LOC fixture") matches the per-droplet KindPayload entries. No droplet exceeds the till-go atomic budget (1-4 code blocks, 80-120 LOC + tests; project CLAUDE.md § "Go Development Rules"). No L3 sub-plan needed.

### 1.B Parallelization-graph — verified

All 6 droplets edit the SAME 2 files (`internal/templates/load.go` + `internal/templates/load_test.go`) AND share the SAME Go package (`internal/templates`). Per project CLAUDE.md § "Blocker Semantics", sibling droplets sharing a file in `paths` OR a package in `packages` MUST have explicit `blocked_by`. The strict serial chain `D1 → D2 → D3 → D4 → D5 → D6` is the only legal shape.

**`_BLOCKERS.toml` ↔ PLAN.md mirroring**:

| PLAN.md droplet line | PLAN.md `Blocked by:` | `_BLOCKERS.toml` entry | Match |
| -------------------- | --------------------- | ---------------------- | ----- |
| D1 (line 66)         | `—`                   | (none — chain head; comment line 21 declares this) | OK |
| D2 (line 98)         | `4c.6.W0.5.D1`        | `[[blockers]] node = "4c.6.W0.5.D2"` blocked_by `D1` (lines 23-26) | OK |
| D3 (line 128)        | `4c.6.W0.5.D2`        | `[[blockers]] node = "4c.6.W0.5.D3"` blocked_by `D2` (lines 28-31) | OK |
| D4 (line 161)        | `4c.6.W0.5.D3`        | `[[blockers]] node = "4c.6.W0.5.D4"` blocked_by `D3` (lines 33-36) | OK |
| D5 (line 193)        | `4c.6.W0.5.D4`        | `[[blockers]] node = "4c.6.W0.5.D5"` blocked_by `D4` (lines 38-41) | OK |
| D6 (line 226)        | `4c.6.W0.5.D5`        | `[[blockers]] node = "4c.6.W0.5.D6"` blocked_by `D5` (lines 43-46) | OK |

Reasons in `_BLOCKERS.toml` are concrete (cite file-lock + package-lock + the specific helper-share rationale for D4/D5). No mismatch between the two ledgers; PLAN.md is truth and `_BLOCKERS.toml` mirrors it correctly.

Order rationale (`PLAN.md:21` + `_BLOCKERS.toml:16-19`) matches the L1 spawn directive verbatim (`workflow/drop_4c_6/PLAN.md:85`): "kind-enum + agent_name first → cycles → recursion-depth → blocked_by acyclicity → claim-vs-impl last." D3 produces the cycle-DFS helper that D4 reuses for depth + D5 extracts into a shared private function — this is internally consistent and explained in each droplet's RiskNotes.

### 1.C Specify-block-well-formedness — verified

Every droplet (D1-D6) carries the full Specify structure mandated by `~/.claude/agents/go-planning-agent.md` and `feedback_plan_down_build_up.md`:

- **Objective** — single load-bearing sentence (e.g. D1: "Extend the closed-12-enum kind-membership check to the new runtime-config `[agents.<kind>]` map shipped by W0.").
- **AcceptanceCriteria** — 6-10 numbered yes/no-verifiable bullets per droplet (D1: 7; D2: 10; D3: 7; D4: 10; D5: 9; D6: 10). Every bullet names a concrete file path, symbol, or `mage` target. Examples:
  - D1 bullet 5: `load_test.go` adds `TestLoadValidatesAgentMapKeysClosedEnum` table-driven test with row 1 = valid kind, row 2 = `agents.totally-bogus` rejected with `errors.Is(err, ErrUnknownKindReference)`, row 3 = canonicalization fold.
  - D4 bullet 1: depth bound named via `childRuleRecursionDepthMax = 5` constant.
  - D6 bullet 4: vacuous-pass on every embedded default + scaffold + sentinel test against synthetic injected claim.
- **ValidationPlan** — every droplet cites `mage test-func ./internal/templates <TestName>` for the per-droplet RED→GREEN cycle + `mage test-pkg ./internal/templates` for full-package suite + `mage ci`. No raw `go test` invocations (project CLAUDE.md § "Build Verification" rule 2 satisfied).
- **RiskNotes** — every droplet identifies 2-4 concrete risks with mitigations (D1: W0 ordering wrinkle, generic-reuse hazard, table-shape consistency; D2: pre-W1.D1 lookup-fn injection, open-ended agent-name set, hard-fail-vs-warn-only split; D3: colored-DFS preservation, degenerate-graph value, sentinel reuse; D4: boundary semantics, cycle-pre-rejection ordering, hardcoded-vs-configurable bound, helper-share; D5: D3-vs-D5 distinction, synthetic-cycle fixture, DFS-extraction discipline; D6: empty known-wired set rationale, test-only injection, runtime-parse rejection, future-drops update warning).
- **ContextBlocks** — typed list (`reference` / `decision` / `constraint` / `warning`) with severity tags (`normal` / `high` / `critical`). Every droplet carries 4-6 blocks pointing at concrete code locations (e.g. D1's `reference (normal): internal/templates/load.go:397-446`; D2's `reference (high): SKETCH.md § 3.4`; D3's `reference (high): internal/templates/load.go:517-580`).
- **KindPayload** — JSON-shaped change list with file/symbol/action/shape_hint per file touched. Every droplet's payload covers all `Paths` declared in the droplet header — no missing or extra entries.

**AcceptanceCriteria testability cross-check** (sub-planner's "10 testable bullets each" claim — verified): each droplet's table-driven test exercises exactly the rows the AcceptanceCriteria bullets call out (sentinel hit, sentinel miss, edge case). D1 row count (3) + D2 row count (3) + D3 row count (4) + D4 row count (4) + D5 row count (3) + D6 row count (3) all map 1:1 to the corresponding AcceptanceCriteria bullets. RED→GREEN cycle is reproducible.

### 1.D Multi-level-decomposition — verified

Sub-planner declared "no L3 needed" (`PLAN.md:250-251`) with the rationale that each droplet's aggregate sizing fits the till-go atomic budget. Verification:

- Each droplet ships ONE production file (`internal/templates/load.go`) + ONE test file (`internal/templates/load_test.go`) + 1-2 fixtures (`internal/templates/testdata/*.toml`).
- Aggregate ~70-160 LOC per droplet (validator + test + fixture) — within the 80-120 LOC + tests budget. The high end (D2 / D5 with new injection points) hovers around the boundary but does not exceed it.
- No droplet bundles multiple unrelated validators. Each one closes a single sentinel error path.

Decomposition shape is correct: ONE level of L2 droplets under the W0.5 sub-plan container; no recursive sub-planning required.

### 1.E Shipped-but-not-wired — verified

Per project anti-pattern ledger (`feedback_tillsyn_enforces_templates.md` + Drop 3 droplet 3.20 reference): every new sentinel introduced must have at least one production-or-test caller before the droplet ships. Audit:

| Sentinel introduced       | Producer (validator)                      | Consumer (test)                                                    | Wired? |
| ------------------------- | ----------------------------------------- | ------------------------------------------------------------------ | ------ |
| (D1 reuses `ErrUnknownKindReference` — already wired by `validateMapKeys` per `load.go:472`) | `validateAgentMapKeys` (extension) | `TestLoadValidatesAgentMapKeysClosedEnum` (3 rows) | YES |
| `ErrUnknownAgentName`     | `validateAgentBindingNames`               | `TestLoadValidatesAgentBindingNamesEmbeddedFloor` (3 rows incl. unknown row) | YES |
| (D3 reuses `ErrTemplateCycle` — already wired by `validateChildRuleCycles` per `load.go:542`) | `validateChildRuleCycles` (extended) | `TestLoadValidatesChildRuleCyclesUnifiedGraph` (4 rows) | YES |
| `ErrChildRuleRecursionTooDeep` | `validateChildRuleRecursionDepth`     | `TestLoadValidatesChildRuleRecursionDepth` (depth-6 row)           | YES |
| `ErrTemplateBlockedByCycle`    | `validateBlockedByAcyclicity`         | `TestLoadValidatesBlockedByAcyclicity` (synthetic-cycle row)       | YES |
| `ErrClaimVsImplUnknownConsumer` | `validateClaimVsImplCoherence`       | `TestLoadValidatesClaimVsImplCoherence` (injected-unknown-consumer row) | YES |

**D6's empty `knownWiredConsumers` set is intentional, not shipped-but-not-wired**:

- L1 W0.5 sub-plan container Acceptance bullet 4 (`workflow/drop_4c_6/PLAN.md:81`) explicitly declares "the set is empty (no consumers wired yet); the validator passes any template at load time without claim-vs-impl rejection. The set's existence + the test that exercises an 'unknown-consumer claim' sentinel ARE in scope; the actual consumer additions are deferred to 4c.7."
- SKETCH.md § 26.W7 AcceptanceCriteria (`SKETCH.md:908`) carries the explicit "W0.5's claim-vs-impl validator's known-wired set updated to include `ChildRulesFor` consumer" bullet → Drop 4c.7 W7 wires the first real entry.
- SKETCH.md § 26.W8 AcceptanceCriteria (`SKETCH.md:934`) carries the parallel bullet for `context.Resolve` → Drop 4c.7 W8 wires the second real entry.
- D6's RiskNotes carry an explicit "LOUD WARNING TO FUTURE DROPS" doc-comment requirement (`PLAN.md:235-236`): future drops adding a runtime consumer MUST update `knownWiredConsumers` in this file.
- D6's TestLoadValidatesClaimVsImplCoherence row 2 (synthetic injected claim → sentinel fail) IS the wired consumer of `ErrClaimVsImplUnknownConsumer`. The validator + sentinel + test ship together in 4c.6; the real-claim consumers ship in 4c.7. The test against the empty set IS the wiring.

D2's `LoadOptions.AgentLookupFn` default-resolves-false-for-every-name pre-W1.D1 wrinkle (`PLAN.md:90`) is also NOT shipped-but-not-wired: D2's tests inject a stub `lookupFn` that returns true for known names; W1.D1 explicitly carries the rewire-D2's-default-lookup bullet (`workflow/drop_4c_6/PLAN.md:101` declares W1.D1 blocked by W0.5; rewire is an L1 acceptance bullet on W1.D1). The injection point + tests + sentinel ship together; the production-default rewire happens in the immediate next droplet.

D5's `LoadOptions.BlockedByGraphFn` follows the same pattern: production default returns the degenerate child→parent graph (today's schema reality); test-only injection lets the synthetic-cycle row exercise the sentinel. Validator + sentinel + test ship together; future schema additions (`BlockedByKinds []domain.Kind` field) make the validator's value forward-real.

No shipped-but-not-wired hits.

## 2. Missing Evidence

(none)

All five plan-QA-proof axes verified directly against L2 PLAN.md content + `_BLOCKERS.toml` content + live `internal/templates/load.go` + `internal/templates/schema.go` state + L1 PLAN.md lines 71-85 + SKETCH.md § 26.W0.5 + § 25.1 + § 26.W7 + § 26.W8 + project CLAUDE.md § "Blocker Semantics" + `feedback_tillsyn_enforces_templates.md` + `feedback_plan_down_build_up.md`. No premise required external Hylla / Context7 / `go doc` lookups — every cite is a concrete file:line pointer.

One non-blocking observation routed back to the orchestrator (NOT a finding):

- **Builder-discretion note on D2's injection-point shape.** D2's AcceptanceCriteria bullet 6 says the builder picks between adding a new `LoadOptions.AgentLookupFn` field OR reusing the existing `LoadOptions.StatFn` with a path-shaped key. Both shapes are sound; the existing `StatFn` at `internal/templates/load.go:36-41` already takes `func(path string) bool`, so re-targeting it for the embedded-FS lookup is a one-line change vs the new field's two-line addition. The PLAN.md's "builder picks" framing is appropriate per till-go decomposition discipline (the planner does not over-specify mechanical-choice details). Not a finding.

## 3. Summary

**Verdict:** PASS

**Finding count:** 0

**Rationale:** The L2 PLAN.md decomposes Wave W0.5 into 6 atomic validator droplets (D1-D6), each within the till-go atomic budget (1-4 code blocks, 80-120 LOC + tests). The strict serial chain `D1 → D2 → D3 → D4 → D5 → D6` is mandated by shared file lock (`internal/templates/load.go` + `load_test.go`) + shared package lock (`internal/templates`); `_BLOCKERS.toml` mirrors PLAN.md exactly. Every droplet carries a complete Specify block (Objective + AcceptanceCriteria + ValidationPlan + RiskNotes + ContextBlocks + KindPayload) with concrete file paths, symbols, and `mage` targets — RED→GREEN cycle reproducible per droplet. Multi-level decomposition is correctly declared at one level (no L3 needed). Every new sentinel has a producer (its own validator) AND a consumer (its own table-driven test row); D6's intentionally-empty `knownWiredConsumers` set is explicitly grounded in L1 Acceptance bullet 4 + SKETCH.md § 26.W7 + § 26.W8 + a LOUD WARNING doc-comment for future drops — not a shipped-but-not-wired hit. The plan is ready to drive 6 sequential build-droplet spawns.

## 4. Hylla Feedback

N/A — this review used file-level reads only against the L2 PLAN.md + `_BLOCKERS.toml` + live `internal/templates/load.go` + `internal/templates/schema.go` + the L1 PLAN.md + SKETCH.md + project CLAUDE.md. No Hylla query was issued; per the spawn directive, Hylla may still be ingesting and the plan-QA-proof axes for an L2 PLAN are answered entirely by direct MD + code reads. No fallback bookkeeping required.
