# PLAN_QA_PROOF — DROP_4c.6.1.W2_TILL_INIT (Round 1)

**Reviewer:** go-qa-proof-agent
**Round:** 1
**Verdict:** **PASS WITH CONDITIONS** — 1 FF + 7 NITs. Plan is structurally sound; the FF is a cross-wave consistency item the L2 builder MUST resolve at D7 dispatch time (after W1.D2 ships), and the NITs strengthen acceptance-bullet rigor without blocking dispatch of D1-D6.

---

## 0. Verification Inventory

Files read end-to-end:
- `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/PLAN.md`
- `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/_BLOCKERS.toml`
- `workflow/drop_4c_6_1/PLAN.md` (L1: lines 100-253, 800-846)
- `workflow/drop_4c_6_1/DROP_4c.6.1.W1_TEMPLATE_RESOLUTION/PLAN.md` (full file, including D2's `ProjectMetadata.Groups []string` field ship)
- `workflow/drop_4c_6_1/REVISION_BRIEF.md` §1-§2.12a
- `cmd/till/init_cmd.go` (current state — single-`Group` payload, `till-gen`/`till-go` allowed groups, `copyAgentsTOML` static-fixture write)
- `cmd/till/init_cmd_test.go` (CONSUMER-TIE `run(ctx, args, &out, io.Discard)` pattern confirmed in `TestInit_BareInvocation_ReturnsTUIStubError`, `TestInit_JSONInvocation_RoutesToValidParse`, `TestInit_JSONParse_TableDriven`, `TestInit_FreshDir_CopiesAllFiles`, etc.)
- `internal/domain/project.go` (LSP-equivalent — `ProjectMetadata` struct lines 119-155 has NO typed `Groups []string` today; W1.D2 ships it)
- `internal/app/service.go` lines 278-345 (`CreateProjectInput` + `CreateProjectWithMetadata` LSP-verified — both shipped pre-W2)
- `internal/tui/components/` — confirmed NOT in tree (W5 ships it)
- `workflow/example/drops/WORKFLOW.md` lines 22-90, 150-174

---

## 1. Findings

### 1.1 FF1 [Axis: shipped-but-not-wired] [severity: high] — D7 specifies `Metadata.KindPayload` JSON stopgap; W1.D2 ships typed `ProjectMetadata.Groups []string` field BEFORE W2.D7 builds

**Claim.** W2 PLAN.md lines 298, 318, 322, 326-327 (D7 Acceptance + RiskNotes + ContextBlocks + KindPayload) instruct the builder to persist groups via `Metadata.KindPayload` JSON `{"groups":[...]}` "since `ProjectMetadata` has no typed `Groups []string` field." This was an accurate observation **at planning time** (the current `internal/domain/project.go:119-155` ProjectMetadata struct has no Groups field), but it is **stale by D7 dispatch time**.

**Evidence pointer:**
- `workflow/drop_4c_6_1/DROP_4c.6.1.W1_TEMPLATE_RESOLUTION/PLAN.md` line 68-70 (AC3): "`domain.ProjectMetadata` carries a `Groups []string` field with JSON tag `\"groups,omitempty\"`."
- W1 PLAN.md line 182 (D2 KindPayload): `{"file":"internal/domain/project.go","symbol":"ProjectMetadata.Groups","action":"modify","shape_hint":"add Groups []string field with json tag 'groups,omitempty'"}`
- W1 PLAN.md line 278-280 (D2 AC1): "`domain.ProjectMetadata.Groups` field exists: `Groups []string` with JSON tag `json:\"groups,omitempty\"`."
- L1 PLAN.md line 821: `4c.6.1.W2 → 4c.6.1.W1` — W1 completes BEFORE W2 starts.
- W2 is `blocked_by 4c.6.1.W1` per L1 PLAN line 249 + W2 PLAN line 5.
- D7 is the LAST droplet in W2's serial chain (Wave C, after W1's W1.D2 has shipped in Wave B).

**Trace.**
W1 ships in Wave B → W1.D2 lands typed `Groups []string` on `ProjectMetadata` → W2 dispatches in Wave C → W2.D1 through D7 run serially. By the time D7 builder boots, `internal/domain/project.go:ProjectMetadata` ALREADY has the typed `Groups []string` field. D7's instruction to use `KindPayload` JSON stopgap will produce **technically working but semantically wrong** code: groups stored in JSON-blob form instead of the typed field W1.D2 explicitly added for this purpose.

REVISION_BRIEF §2.5 line 80 confirms the intent: "`Metadata.groups = [...]` (the selected groups, persisted on project record)." The intent was always the typed field; W1.D2 is the wave that lands it.

**Fix hint.** Update D7's spec at builder-dispatch time (after W1.D2 closes) to:
1. Set `Metadata.Groups = payload.Groups` directly on `ProjectMetadata` (the W1.D2 typed field).
2. Remove the `KindPayload` JSON stopgap path.
3. Remove the W2-GROUPS-R1 refinement entry (line 350) since the typed field exists.
4. Update D7 AcceptanceCriteria + RiskNotes + ContextBlocks accordingly.

This is a CROSS-WAVE consistency finding the L2 plan could not have foreseen at authoring time, but the L2 builder MUST resolve before authoring D7's diff. The fix is a documentation-only update to PLAN.md at D7 dispatch; no L2-replan needed.

**Severity = high** because shipping the stopgap when the typed field exists violates the "Tillsyn Enforces Templates, Doesn't Hardwire Behavior" + "shipped-but-not-wired" anti-pattern principle: the typed field exists (schema), but D7 wouldn't wire to it (consumer). This is exactly the failure mode the dev's MEMORY rule explicitly warns against.

---

## 2. NITs (First-Class Findings — All Address-At-Dispatch-Time)

### 2.1 NIT1 [Axis: parallelization-graph] [severity: low] — `_BLOCKERS.toml` omits D1's cross-wave blocker `4c.6.1.W4.D1` while D3/D4 cross-wave `4c.6.1.W5` blockers are present

**Claim.** `_BLOCKERS.toml` has rows for D2-D7 but **no row for W2.D1**. D1's inline `Blocked by: 4c.6.1.W4.D1` bullet (PLAN.md line 41) is not mirrored in `_BLOCKERS.toml`. D3's `4c.6.1.W5` and D4's `4c.6.1.W5` cross-wave blockers ARE mirrored. The convention is inconsistent.

**Evidence pointer:**
- `_BLOCKERS.toml` lines 6-34 (no `node = "W2.D1"` entry).
- PLAN.md line 41 (D1 inline `Blocked by: 4c.6.1.W4.D1`).
- WORKFLOW.md line 80: "if the two disagree, `PLAN.md` is truth and `_BLOCKERS.toml` is stale."
- WORKFLOW.md line 88: "Cross-subtree leaf-level ordering is NOT expressible in `_BLOCKERS.toml`" — partially applicable, but D3/D4's W5 cross-wave blockers ARE in the file, so the convention here is "mirror all blockers."

**Fix hint.** Add a `[[blockers]]` entry for W2.D1 with `blocked_by = ["4c.6.1.W4.D1"]` and reason "W4.D1 confirms canonical group names (gen/go/fe) before D1 hard-codes them." Or, document an explicit policy that cross-wave blockers belong only on PLAN.md (and remove W5 from D3/D4 in `_BLOCKERS.toml`). Either way, restore consistency.

---

### 2.2 NIT2 [Axis: acceptance-criteria-coverage] [severity: medium] — D1 acceptance bullets do not explicitly invoke `run(ctx, args, &out, io.Discard)` CONSUMER-TIE pattern

**Claim.** L1 PLAN.md line 251 (W2 spawn directive) and the W2 PLAN.md "Notes" section (line 354-356) require the CONSUMER-TIE TEST CONTRACT (`run(ctx, args, &out, io.Discard)` end-to-end pattern) as the **primary** acceptance gate for every D-series droplet. D1's acceptance bullets (PLAN.md lines 42-49) list `validateInitPayload` updates and `initTUIGroupRows` changes, but the only test-pattern reference is "`mage test-pkg ./cmd/till` passes." Without an explicit "tests invoke `run()` end-to-end" line, the D1 builder may reach for unit assertions on `validateInitPayload` directly (which CONSUMER-TIE explicitly says are acceptable as SUPPLEMENT but NOT as primary gate).

**Evidence pointer:**
- PLAN.md line 49 (D1 Acceptance final bullet): "`mage test-pkg ./cmd/till` passes with all existing and new tests green."
- PLAN.md line 354-356 (Notes / CONSUMER-TIE clause): "All tests in `init_cmd_test.go` invoke `run(ctx, args, &out, io.Discard)` end-to-end ... Unit assertions directly on internal helpers (`copyAgentFiles`, `validateInitPayload`, etc.) are acceptable as SUPPLEMENTS but not as the PRIMARY acceptance gate."
- Compare D2 acceptance line 89: "Tests: `run(ctx, args, &out, io.Discard)` end-to-end — one test for FLAT layout present ..." (explicit).
- Compare D5 acceptance line 214: explicit `run()` end-to-end. D6 line 255 explicit. D7 line 302 explicit.

**Fix hint.** Add one acceptance bullet to D1: "CONSUMER-TIE: validation behavior tested via `run(ctx, args, &out, io.Discard)` end-to-end — at minimum one new test driving `run(..., '--json', '{\"name\":\"x\",\"groups\":[\"go\"],\"mcp\":false}')` (valid) + one driving the multi-element variant + one expecting validation-error from an invalid group name. Unit-on-`validateInitPayload` assertions OK as supplement."

---

### 2.3 NIT3 [Axis: acceptance-criteria-coverage] [severity: medium] — D3/D4 acceptance bullets favor `teatest_v2` drive-model-directly tests over CONSUMER-TIE `run()` pattern

**Claim.** D3 line 130: "Tests use `teatest_v2` pattern (per existing init_cmd_test.go conventions): drive model directly to verify Done/Cancelled/Payload state; existing cobra end-to-end test still surfaces expected error in `go test` (no real terminal)." D4 line 172: "TUI model tests verify MCP step transitions (Enter=YES, n=NO, Esc=cancel); end-to-end test via `run()` for JSON mode with `\"mcp\":false` stays green." Both lean on teatest-drive-model — which is reasonable for TUI verification (real terminal not available in `go test`) — but the JSON-mode `run()` coverage of multi-group + new MCP step is implicit, not asserted.

**Evidence pointer:**
- PLAN.md lines 130 (D3) and 172 (D4): teatest-direct-drive language.
- L1 PLAN.md line 251: "CONSUMER-TIE TEST CONTRACT (R2-NIT3): tests invoke `run(ctx, args, &out, io.Discard)` end-to-end — flow-level assertions, not unit assertions on internal helpers. All D-series droplets sharing `init_cmd.go` follow this pattern."
- Existing test `cmd/till/init_cmd_test.go:191` `TestRunInitTUI_AcceptsDefaultNameAndSelectsTillGo` does drive teatest directly — pattern is established. The L1 contract still says "all D-series" → some friction.

**Fix hint.** Pre-MVP, the practical resolution is: teatest drive-model is the PRIMARY gate for TUI-step behavior (real terminal unavailable in CI), AND a CONSUMER-TIE `run(--json ...)` end-to-end test must cover the SAME state transitions as JSON mode. Add to D3: "CONSUMER-TIE supplement: `run(..., '--json', '{\"name\":\"x\",\"groups\":[\"go\",\"fe\"],\"mcp\":false}')` exercises the multi-group payload path without entering the TUI; this is the JSON-mode mirror of D3's TUI multi-select." Add to D4: "CONSUMER-TIE supplement: `run(..., '--json', '{\"name\":\"x\",\"groups\":[\"go\"],\"mcp\":true}')` exercises the MCP=true path; `'mcp':false` exercises MCP=false; both via `run()`." (D4 already has the partial line — strengthen it.)

---

### 2.4 NIT4 [Axis: spec-conformance] [severity: low] — D7 RiskNotes ContextBlocks `reference` line 325 cites `internal/domain/project.go:212` for `CreateProjectInput` — that's actually `ProjectInput`, not `CreateProjectInput`

**Claim.** PLAN.md line 325: "`internal/domain/project.go:212` (`CreateProjectInput` fields)." Verified by reading the file: line 212 is `type ProjectInput struct` (the inner domain input shape passed to `NewProjectFromInput`), NOT `CreateProjectInput`. `CreateProjectInput` lives in `internal/app/service.go:286-300`. The reference confuses two different shapes.

**Evidence pointer:**
- `internal/domain/project.go` line 212: `type ProjectInput struct {` (NOT CreateProjectInput).
- `internal/app/service.go` line 286: `type CreateProjectInput struct {` (the actual shape D7 builder constructs).
- W2 PLAN.md line 325 reference bullet.

**Fix hint.** Update D7 reference to: "`internal/app/service.go:286` (`CreateProjectInput` — the shape D7 constructs) wraps `internal/domain/project.go:212` (`ProjectInput` — internal validation shape called by `NewProjectFromInput`)."

---

### 2.5 NIT5 [Axis: acceptance-criteria-coverage] [severity: low] — D5 acceptance does not explicitly state `blocked_by W4.D1` is required for embedded `agents/<group>/` paths

**Claim.** D5 RiskNotes line 229: "W4.D1 restructures the embedded `internal/templates/builtin/agents/<group>/` directories. D5 ships after W4.D1 (via W2 blocked_by W4.D1) — the embedded paths MUST exist before D5 can be tested." This relies on transitive W2 → W4.D1 wave blockage. D5 itself only has `blocked_by W2.D4`. While transitively correct, explicit-redundant `blocked_by 4c.6.1.W4.D1` on D5 would mirror the D3/D4 pattern (which explicitly include W5 even though W2 wave is W5-blocked). PLAN-QA-DISCIPLINE-R1 prefers explicit blockers for new-behavior dependencies.

**Evidence pointer:**
- W2 PLAN.md line 205 (D5 Blocked by): `W2.D4` only.
- W2 PLAN.md line 4 (Wave): `blocked by W1 + W4.D1 + W5` (wave-level).
- D3 line 123 / D4 line 165: explicit cross-wave `4c.6.1.W5` blocker (redundant-but-explicit pattern).

**Fix hint.** OPTIONAL — depends on policy choice from NIT1: either (a) consistent inclusion (add `4c.6.1.W4.D1` to D5 explicitly) or (b) consistent omission (remove W5 from D3/D4 since wave-level covers it). Pick one, document in PLAN.md notes.

---

### 2.6 NIT6 [Axis: spec-conformance] [severity: medium] — L1 + L2 plans both omit multi-group aggregation of `agents.toml` (`<project>/agents.toml` `[<group>]` / `[<group>.<kind>]` sections)

**Claim.** REVISION_BRIEF §2.3 line 60: "Aggregate the group's bindings into `<project>/agents.toml` under `[<group>]` and `[<group>.<kind>]` sections." The L2 plan's D6 covers `<project>/.tillsyn/template.toml` aggregation but NOT `<project>/agents.toml` aggregation. Today's `copyAgentsTOML` (`cmd/till/init_cmd.go:585-602`) reads a single static embedded fixture `builtin/agents.example.toml` and writes it verbatim — single-group only. After W4.D2 ships the new schema, `copyAgentsTOML` still copies a single fixture; the multi-group aggregation REVISION_BRIEF specified is NOT in W2's L2 plan.

**Note on inheritance.** This gap is inherited from L1 — L1 PLAN.md lines 240-247 acceptance bullets do not include an `agents.toml` multi-group aggregation bullet. So the L2 plan is faithful to its L1 contract, but the L1 contract drifts from REVISION_BRIEF. Surfacing here so the orchestrator can route it back: either (a) explicit decision that `agents.toml` ships W4.D2's static template content (skipping per-group aggregation since the new TOML format is `[<group>]`-shaped and a single embedded fixture can contain ALL groups), or (b) D6 (or a new D6b) handles `agents.toml` aggregation. REVISION_BRIEF says (b); current plan effectively does (a).

**Evidence pointer:**
- REVISION_BRIEF.md line 60: aggregation intent.
- L1 PLAN.md lines 240-247: no `agents.toml` aggregation acceptance.
- W2 PLAN.md D6 line 252-256: only `<project>/.tillsyn/template.toml`, not `<project>/agents.toml`.
- `cmd/till/init_cmd.go:585-602` `copyAgentsTOML` — current static-fixture write.

**Fix hint.** Surface to orchestrator. If decision is (a) — W4.D2 ships a single `agents.example.toml` containing `[go]` + `[fe]` + `[gen]` sections and `copyAgentsTOML` stays single-fixture — add an acceptance bullet to W4.D2 confirming "embedded `agents.example.toml` contains all three group sections" and add a note to W2 confirming "single-fixture copy is sufficient post-W4.D2." If decision is (b) — actual multi-group aggregation from per-group sources — add a D6b droplet to W2.

---

### 2.7 NIT7 [Axis: spec-conformance] [severity: low] — D4 RiskNotes line 186 leaves `MCP bool` vs `*bool` builder choice unresolved; default-true semantics in JSON mode need a stronger contract

**Claim.** D4 RiskNotes line 186: "Builder decides: either (a) use `*bool` for `MCP` and default nil→true, or (b) keep `bool` and document that JSON callers must pass `\"mcp\":true` explicitly. Option (a) is more ergonomic; option (b) preserves backward compatibility. Builder picks and documents in test + doc comment." Per `feedback_nits_are_first_class.md`, leaving design choices to builder-discretion at L2 plan-QA time creates downstream ambiguity. REVISION_BRIEF §2.6 line 88: "JSON mode: respects `mcp` boolean as before; default true if absent." That's the contract — JSON mode default-true if absent. Pick option (a) at planning time.

**Evidence pointer:**
- W2 PLAN.md line 186 (D4 RiskNotes): builder-discretion clause.
- REVISION_BRIEF.md line 88: "default true if absent" — implies pointer-bool or sentinel.
- Existing `domain.ProjectMetadata.OrchSelfApprovalEnabled *bool` (lines 119-128 of `internal/domain/project.go`) is the established pattern for nil-means-default-on tristate booleans.

**Fix hint.** Lock option (a) at planning: change `initJSONPayload.MCP` from `bool` to `*bool` with JSON tag `"mcp,omitempty"`. Add a helper `func (p initJSONPayload) MCPRegistration() bool { if p.MCP == nil { return true }; return *p.MCP }` mirroring the `OrchSelfApprovalIsEnabled()` pattern. Update D4 acceptance to require this shape + a test asserting omitted `mcp` defaults to YES.

---

## 3. Cross-Planner Observations (Not Blocking)

### 3.1 PLAN-QA-DISCIPLINE-R1 (test-runner droplet blocked_by) — PASS with caveat

Every D-series droplet's tests run within the same droplet that lands the new behavior — there is no separate test-runner droplet downstream. So R1's "test-runner droplet's `blocked_by` includes the wave shipping the behavior" reduces to "each droplet's `blocked_by` covers its own new-behavior dependencies." That holds for D1-D7 with the caveat in NIT5 (D5 transitively gets W4.D1 via wave-level wave blockage; D3/D4 explicitly include W5).

### 3.2 PLAN-QA-DISCIPLINE-R2 (narrative droplet count vs enumerated list) — PASS

PLAN.md line 15 narrative: "Seven atomic droplets." Table lines 21-27: 7 rows. Body sections D1 (line 33), D2 (75), D3 (115), D4 (157), D5 (197), D6 (240), D7 (282) — 7 enumerations. Match.

### 3.3 Blocked_by acyclicity — PASS

Topo-walk:
- D1 ← {W4.D1}
- D2 ← {D1}
- D3 ← {D2, W5}
- D4 ← {D3, W5}
- D5 ← {D4}
- D6 ← {D5}
- D7 ← {D6}

D3 + D4 both name W5 explicitly. W5 is Wave A; W2.D1 starts Wave C (after W5 ships). Explicit blocker is REDUNDANT-but-not-cyclic. PASS.

### 3.4 Shared-file/package coverage — PASS

All 7 droplets share `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go` + package `cmd/till`. Fully serialized D1 → D2 → ... → D7. Cross-wave: W7.D3 shares `cmd/till` package, explicitly `blocked_by W2` per L1 line 823. W3 shares `cmd/till` package, `blocked_by W2` per L1 line 827. All package collisions resolved.

### 3.5 CONSUMER-TIE coverage — MIXED (see NIT2 + NIT3)

D2/D5/D6/D7: explicit `run()` end-to-end ✓.
D1: missing explicit `run()` mention → NIT2.
D3/D4: lean on teatest drive-model; `run()` mention partial → NIT3.

### 3.6 Closed-enum / smart-default coverage — PASS

D7 Language mapping (line 297, 319, 323) — closed enum `"" | "go" | "fe"` with explicit priority `go > fe > "" (gen)` documented. Mirrors `internal/domain/project.go:279-286` `isValidProjectLanguage`. PASS.

---

## 4. Premises / Evidence / Trace / Conclusion / Unknowns

### Premises

- W2.D1-D7 form a strict serial chain on `cmd/till/init_cmd.go` and `cmd/till/init_cmd_test.go`.
- W2 dispatches in Wave C, after W1 (Wave B) + W4.D1 (Wave A) + W5 (Wave A) all complete.
- L1 PLAN.md is the source-of-truth for wave-level acceptance; L2 PLAN.md decomposes within W2's declared scope.
- REVISION_BRIEF.md is the user-intent source; deviations between REVISION_BRIEF and L1 are flagged.
- `_BLOCKERS.toml` mirrors PLAN.md inline `Blocked by:` bullets; PLAN.md is truth on drift.

### Evidence

- All 7 droplets enumerated, each with KindPayload referencing specific symbols verified against `cmd/till/init_cmd.go` (line refs confirmed).
- W1 PLAN.md D2 ships `ProjectMetadata.Groups []string` (lines 68-70, 182, 278-280) — confirmed cross-wave.
- `internal/domain/project.go` (read end-to-end) — confirms current state has NO Groups field; W1.D2 lands it.
- `internal/app/service.go:286-345` — confirms `CreateProjectInput` + `CreateProjectWithMetadata` shipped pre-W2.
- `internal/tui/components/` confirmed not in tree — W5 ships it; D3/D4 blocked-by W5 is correct.
- Test patterns in `cmd/till/init_cmd_test.go` confirm `run(ctx, args, &out, io.Discard)` is the established end-to-end pattern.

### Trace / Cases

- **D1 → D7 dispatch order**: serial, no parallelism within wave. Verified against shared-file invariant.
- **Cross-wave dispatch order**: W4.D1 (Wave A) → W5 (Wave A) → W1 (Wave B, ships ProjectMetadata.Groups) → W2.D1 (Wave C) → W2.D2 → ... → W2.D7. By D7 dispatch time, ProjectMetadata.Groups EXISTS.
- **CONSUMER-TIE pattern**: established in `init_cmd_test.go`; required by L1 line 251; explicit in D2/D5/D6/D7, implicit/missing in D1/D3/D4.
- **Acyclic blocker graph**: topo-sort completes; no cycle.
- **`_BLOCKERS.toml` mirroring**: D2-D7 present; D1 cross-wave blocker missing (NIT1).

### Conclusion

**PASS WITH CONDITIONS.** The L2 plan is dispatchable today for D1-D6 with no blockers (FF1 affects only D7's spec wording, which the L2 builder/orchestrator must refresh after W1.D2 lands). NITs strengthen acceptance rigor but do not block dispatch.

Recommended orchestrator action before D7 dispatch:
1. After W1.D2 closes, re-spawn this plan-QA-proof (Round 2) OR have the orchestrator directly amend D7's Acceptance / RiskNotes / ContextBlocks / KindPayload to consume the typed `ProjectMetadata.Groups` field per FF1's fix hint.
2. At L2 planning convenience: absorb NIT1 (`_BLOCKERS.toml` D1 mirror), NIT2 (D1 CONSUMER-TIE bullet), NIT4 (D7 reference cleanup), NIT7 (lock D4 `*bool` choice).
3. Route NIT6 to dev for decision (single-fixture vs aggregated `agents.toml`).

### Unknowns

- **U1**: Whether `agents.toml` aggregation gap (NIT6) is intentional W4.D2-handles-it decision or L1 contract miss. Need dev / orchestrator adjudication.
- **U2**: Whether D3/D4 teatest-drive-model + `run(--json)` mirror approach satisfies CONSUMER-TIE strictness (NIT3) or requires stronger `run()` direct-TUI coverage (currently impossible in `go test` without real terminal).

Both Unknowns route via this MD; not blocking dispatch of D1-D6.

---

## 5. Hylla Feedback

N/A — Hylla was explicitly OFF for this review per spawn-prompt directive. All evidence gathering used LSP-equivalent `Read` + `Grep` against the live checkout.
