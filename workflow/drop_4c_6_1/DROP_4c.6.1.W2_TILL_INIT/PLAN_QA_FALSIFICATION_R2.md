# PLAN-QA-FALSIFICATION — DROP_4c.6.1.W2_TILL_INIT (Round 2)

**Reviewer:** go-qa-falsification-agent (filesystem-MD mode, Hylla OFF)
**Round:** 2
**Verdict:** **PASS-WITH-NITs** — 0 FF, 4 NITs. All 3 round-1 FFs (FF1 W4 till-prefix, FF2 D7 KindPayload stopgap, FF3 _BLOCKERS.toml drift) and all 7 round-1 NITs absorbed correctly. The L2 plan is dispatchable for D1-D7 modulo the 4 nits below. No counterexample lands.

---

## 1. Cross-Wave Attack Verification

### 1.1 Attack — `Metadata.Groups = payload.Groups` consumption (W1.D2 typed-field landing)

**Attempt.** Verify W1.D2 actually ships `ProjectMetadata.Groups []string` with a JSON tag that round-trips W2.D7's write back to `Metadata.Groups = payload.Groups`. Verified W1 round-2 PLAN.md:

- L43-44 (Round 2 Changes — Fals FF2): "D2 acceptance reaffirmed with explicit routing note. W2/W3 PLAN.md updates are out of W1 scope — their round-2 planners absorb this."
- L104-108 (AC3): `domain.ProjectMetadata` carries a `Groups []string` field with JSON tag `"groups,omitempty"`. **W2.D7 and W3.D1 MUST consume this typed field directly (not `KindPayload` JSON).**
- L264 (D2 KindPayload): `add Groups []string field with json tag 'groups,omitempty'`.
- L375-376 (D2 AC1): `Groups []string` with JSON tag `json:"groups,omitempty"`.
- L408-411 (D2 AC6): "W2.D7 and W3.D1 MUST consume `project.Metadata.Groups` typed field directly. They MUST NOT use `KindPayload` JSON stopgap."

W2 round-2 PLAN.md D7 acceptance (L349, L361, L374-376):
- AC5: `Metadata.Groups = payload.Groups` — typed field from W1.D2.
- AC field-name match: `Groups`, slice element `string` (`[]string`), JSON tag `groups,omitempty`. **MATCH.**

**Verdict.** REFUTED — no counterexample. Field-name + slice-element-type + JSON-tag all match between W1.D2 and W2.D7. The wave-ordering (W1 Wave B → W2 Wave C) ensures W1.D2 is `complete` before W2.D7 dispatches. W2.D7 RiskNotes line 370 explicitly notes the verify-via-LSP step.

### 1.2 Attack — D1 `MCP *bool` accessor coerces explicit `false` to nil?

**Attempt.** D1 acceptance: `MCPRegistration() bool { if p.MCP == nil { return true }; return *p.MCP }`. Construct a `--json '{"name":"x","groups":["go"],"mcp":false}'` payload — does the unmarshal produce `MCP = &false` or `MCP = nil`?

Go JSON decoding: a JSON literal `false` into `*bool` field produces a non-nil pointer to `false` (not nil). Only OMITTED `"mcp"` key (or explicit `"mcp": null`) produces nil. With `"mcp":false`, `p.MCP` is non-nil, `*p.MCP = false`, `MCPRegistration()` returns `false`. Correct behavior.

CONSUMER-TIE test coverage:
- D1 AC line 74 case (b): `--json '{"name":"x","groups":["go","fe"],"mcp":false}'` — verifies explicit `false` round-trip.
- D1 AC line 74 case (a): no `mcp` key — verifies nil→true default.
- D4 AC line 215: three `run(--json)` paths (mcp:true, mcp:false, no-mcp-key). Explicit nil→true coverage.

**Verdict.** REFUTED — no counterexample. The accessor pattern is canonical (mirrors `OrchSelfApprovalEnabled *bool` per `internal/domain/project.go:128`). D1's three `run()` cases cover all three JSON-key states (omitted/false/true).

### 1.3 Attack — D6 partial-state warning vs blanket-skip semantics

**Attempt.** Round-1 fals NIT4 raised the partial-state idempotency gap: user with `template.toml` covering `go` only then runs `--group go --group fe`. Does D6's blanket-skip + fail-loud warning actually cover this case correctly, and is the remediation guidance accurate?

D6 round-2 AC line 302: "If the existing file is absent one or more `[<group>]` sections for the current selected groups, `runInitPipeline` prints a warning: `\"WARN: <destDir>/.tillsyn/template.toml already exists but is missing sections for group(s): [<missing-list>]. Remove it and re-run to regenerate.\"` (non-fatal — exits zero, warning only)."

Trace through user scenario: existing `template.toml` has `[go]` section only; user runs `till init --group go --group fe`.
- Blanket-skip applies (file exists).
- Detection heuristic (line 320): "check if the existing `template.toml` content contains `\"[<group>]\"` or `\"[<group>.\"` for each selected group." For `[go]` and `[fe]`:
  - `[go]` substring present → covered.
  - `[fe]` substring absent → missing.
- Warning fires: `missing sections for group(s): [fe]`.
- Remediation: `Remove it and re-run to regenerate.`

The remediation IS the simplest path consistent with the no-migration philosophy. But it's lossy: if the user has hand-edited `[go]` section, `rm` destroys that customization. That's acceptable pre-MVP per "fail-loud, no-migration" philosophy.

**Sub-concern:** the substring check `"[<group>]"` vs `"[<group>."` is a simple string-presence heuristic, not a TOML parse. A TOML comment `# this section is for [go]` would falsely match. But the heuristic is documented (line 320) and pre-MVP-acceptable.

**Verdict.** REFUTED — no counterexample on correctness. The heuristic limitation is documented; the remediation is consistent with no-migration philosophy. Flagged below as **NIT-R2-1** (low-severity) for stronger heuristic in a later drop.

### 1.4 Attack — `runInitPipeline` FLAT-detection placement survives D5's `copyAgentFiles` refactor

**Attempt.** D2 places `detectFLATLayout` + `detectOldSchemaAgentsTOML` inside `runInitPipeline` BEFORE `copyAgentFiles`. D5 refactors `copyAgentFiles` signature to `(destDir string, groups []string)`. Does the detection survive D5?

Current `runInitPipeline` (`cmd/till/init_cmd.go:404`) is structured as a top-level function calling `copyAgentFiles` at line 410. D2 inserts detection calls BEFORE line 410. D5 modifies the `copyAgentFiles` signature and call site at line 410 (`copyAgentFiles(destDir, payload.Groups)`), but does NOT touch the detection calls D2 inserted earlier in `runInitPipeline`. Independent edit points; no conflict.

D2 RiskNotes line 140-141 documents this exactly: "The FLAT check is placed in `runInitPipeline` (not inside `copyAgentFiles`) so it survives the D5 rewrite independently." Defensive constraint (line 142-143 in ContextBlocks): "D5 must NOT add FLAT detection into `copyAgentFiles` — that would create a double-check redundancy."

**Verdict.** REFUTED — no counterexample. D2's placement is D5-independent. D5 RiskNotes explicitly forbids re-adding the check inside `copyAgentFiles`.

### 1.5 Attack — Canonical group names mismatch between D1 hardcoded list and W4.D1's `git mv`

**Attempt.** D1 hardcodes `allowedInitGroups = ["gen", "go", "fe"]`. W4.D1 performs `git mv till-go → go` + `git mv till-gen → gen` and adds new `fe/` dir.

L1 PLAN.md W4.D1 acceptance (line 381): "Post-rename directory listing: `internal/templates/builtin/agents/` contains exactly `go/`, `gen/`, `fe/`, `till-gdd/` (4 subdirs). NO `till-go/`. NO `till-gen/`."

D1's `allowedInitGroups = ["gen", "go", "fe"]` matches the post-rename canonical names exactly. D5's embed path `builtin/agents/<group>/` for `<group>` in `{"gen", "go", "fe"}` resolves correctly post-W4.D1.

**Verdict.** REFUTED — no counterexample. W4.D1's renames + D1's hardcoded list + D5's embed paths all agree on `{"gen", "go", "fe"}`.

### 1.6 Attack — agents.toml multi-group fixture path (W4.D2 single fixture vs W2.D6 per-group expectations)

**Attempt.** L1 W4.D2 acceptance (line 433) says: "Both `[go]` and `[fe]` group sections present" in the single embedded `agents.example.toml`. Does W2's `runInitPipeline` (calling `copyAgentsTOML(destDir)` at line 414 of `init_cmd.go`) write this single fixture path correctly, or does it expect per-group separate fixtures?

`copyAgentsTOML` (current `init_cmd.go:585-602`) reads ONE static path: `const srcPath = "builtin/agents.example.toml"`. It copies that single fixture to `<destDir>/agents.toml`. The W2 round-2 PLAN.md does NOT modify `copyAgentsTOML` in any of D1-D7. Notes §agents.toml Gap (line 434-442) explicitly accepts this gap and recommends Option (a): "W4.D2 ships an embedded `agents.example.toml` containing `[go]` + `[fe]` + `[gen]` sections; `copyAgentsTOML` stays single-fixture."

**Sub-concern.** L1 W4.D2 acceptance line 433 says "Both `[go]` and `[fe]` group sections present" — it does NOT mention `[gen]`. W2 PLAN.md Notes §agents.toml Gap line 439 says "containing `[go]` + `[fe]` + `[gen]` sections." Slight gap: W4.D2 acceptance asserts 2 sections; W2's deferral note expects 3. If a user runs `till init --group gen`, the agents.toml fixture lacks `[gen]` and the user gets an inconsistent state (agent files copied but no agents.toml bindings).

This is a CROSS-WAVE inconsistency between L1 W4.D2 acceptance (2 sections: `[go]`, `[fe]`) and W2's reliance on 3 sections. But W2 is faithful to its L1 contract (L1 W2 acceptance doesn't include agents.toml multi-group aggregation). The gap is correctly surfaced to the orchestrator (Notes §agents.toml Gap, line 442: "This decision requires orchestrator adjudication"). Routed via the round-1 NIT6 DEFERRED disposition.

**Verdict.** REFUTED on falsification axis (no W2-internal counterexample). The cross-wave gap is correctly surfaced for orchestrator routing. Tracked as **NIT-R2-2** (medium) below to surface again pre-dispatch.

### 1.7 Attack — Numeric consistency (7 narrated + 7 enumerated + 7 _BLOCKERS rows)

**Attempt.** PLAN-QA-DISCIPLINE-R2 check.
- L37 narrative: "Seven atomic droplets in a strict serial chain."
- Table L41-49: 7 rows (W2.D1 through W2.D7).
- Body sections D1 (L55), D2 (L110), D3 (L150), D4 (L198), D5 (L244), D6 (L289), D7 (L333) — 7 enumerations.
- `_BLOCKERS.toml`: 7 `[[blockers]]` rows (W2.D1 through W2.D7).
- Blockers Reference table L389-396: 7 rows.

All five sources agree on 7. PASS.

**Verdict.** REFUTED — no counterexample.

### 1.8 Attack — `_BLOCKERS.toml` ↔ PLAN.md drift on W2.D1 + W2.D5 cross-wave entries

**Attempt.** Round-1 FF3 found W2.D1 entry missing from `_BLOCKERS.toml`. Round-2 PLAN.md L17 says "RESOLVED UNILATERALLY by orchestrator before round-2 dispatch."

Verified `_BLOCKERS.toml`:
- W2.D1 entry exists (L6-9): `blocked_by = ["4c.6.1.W4.D1"]`. ✓ Matches PLAN.md L43 (table) + L63 (D1 inline) + L390 (Blockers Reference).
- W2.D5 entry (L26-29): `blocked_by = ["W2.D4", "4c.6.1.W4.D1"]`. ✓ Matches PLAN.md L47 (table) + L252 (D5 inline) + L394 (Blockers Reference).
- W2.D3 (L17-19): `["W2.D2", "4c.6.1.W5"]`. ✓
- W2.D4 (L21-24): `["W2.D3", "4c.6.1.W5"]`. ✓
- Reasons in `_BLOCKERS.toml` align with PLAN.md inline rationale.

**Verdict.** REFUTED — no counterexample. Round-1 FF3 absorbed cleanly. Round-1 NIT5 (D5 missing explicit W4.D1) also absorbed.

### 1.9 Attack — `Disabled bool` field removal interacts incorrectly with TUI navigation logic

**Attempt.** D1 keeps `Disabled bool` field on `initTUIGroupRow` for D1→D3 interim. D3 removes it. But `nextEnabledGroupRow` / `prevEnabledGroupRow` helpers reference `row.Disabled` to skip rows. If D1 keeps the field but all rows are `Disabled: false`, the helpers become no-op (cursor advances over all 3 rows linearly). Good.

Then in D3, the picker_multi.go component handles its own navigation — D3 deletes `nextEnabledGroupRow`, `prevEnabledGroupRow`, `initTUIGroupRows`, `initTUIGroupRow` (including `Disabled` field), `groupCursor`. PLAN.md D3 acceptance line 165 explicitly enumerates these deletions.

Cross-check: in the D1→D3 interim, the existing TUI Update logic at `init_cmd.go:220-238` references `row.Disabled` on line 228 ("defense-in-depth: if cursor somehow lands on disabled row, Enter is no-op"). After D1, all rows are `Disabled: false`, so this branch is dead-but-correct (cursor never lands on a disabled row). After D3, the picker_multi.go takes over and the entire block is replaced — `row.Disabled` reference disappears.

Also: line 268 in View renders `(disabled — reserved for GDD)` suffix when `row.Disabled` is true. D1's spec on L73 says "All doc-comments referencing `Group string`, `till-gen`, `till-go`, `\"till-gdd\"`, `reservedInitGroups` are updated or removed." The View rendering of `(disabled — reserved for GDD)` references `"reserved for GDD"` — this label is a runtime string, not a doc-comment. D1's acceptance doesn't explicitly mention removing this rendering string.

**Sub-concern.** D1's spec says "till-gdd row removed" (L104) and all rows enabled (no `Disabled: true`). With all rows enabled, the View's `if row.Disabled` branch is dead code that won't fire — but the literal `"(disabled — reserved for GDD)"` string lives in the source until D3 wholesale-replaces the View. Not a falsification — dead code lives in source for the duration of the D1→D3 interim; D3's picker_multi.go replacement removes it.

**Verdict.** REFUTED — no counterexample. Interim dead-code state is intentional and tracked. Flagged below as **NIT-R2-3** (low) to surface the View string for cleanup awareness.

### 1.10 Attack — CONSUMER-TIE D3/D4 JSON-mode mirrors equivalent to teatest-direct?

**Attempt.** D3/D4 use `teatest_v2` for TUI behavior (real terminal unavailable in `go test`). JSON-mode mirrors test the same logic via `run(ctx, args, &out, io.Discard)`. Are they actually equivalent?

D3 multi-select TUI state machine: user toggles `go` + `fe` rows via space, presses Enter, payload becomes `Groups: ["go", "fe"]`. JSON-mode mirror: `run(--json '{"name":"x","groups":["go","fe"],"mcp":false}')` produces the same `payload.Groups = ["go", "fe"]` slice at the validation gate.

But the JSON-mode mirror BYPASSES the TUI's interactive logic — it doesn't exercise space-toggle, doesn't exercise the empty-selection rejection, doesn't exercise the default-`["gen"]` pre-selection. The JSON-mode mirror exercises only the DOWNSTREAM consumption of `Groups` after the picker would have produced it. So the two tests cover DIFFERENT layers:

- teatest direct-drive: TUI state machine + payload-shaping.
- JSON-mode mirror: payload-validation + downstream consumption.

D3 acceptance line 167 frames this exactly: "CONSUMER-TIE supplement: `run(..., '--json', '{"name":"x","groups":["go","fe"],"mcp":false}')` exercises the multi-group payload path **without entering the TUI**; this is the JSON-mode mirror of D3's TUI multi-select."

The word **"supplement"** is load-bearing. The teatest path is the PRIMARY gate for TUI behavior; the `run(--json)` is a SUPPLEMENT covering downstream wiring. They are NOT equivalent — they are complementary. The round-1 NIT3 absorption documented this correctly.

D4 same shape: teatest tests confirm step transitions (Enter=YES, n=NO, Esc=cancel); JSON-mode mirror tests the downstream `MCPRegistration()` accessor consumption.

**Verdict.** REFUTED — no counterexample. Tests are correctly framed as complementary, not equivalent. The L1 CONSUMER-TIE contract (line 251) says tests invoke `run()` end-to-end; D3/D4's JSON-mode mirror satisfies that contract for the non-interactive payload path while teatest covers the interactive path.

---

## 2. Findings (FF + NIT)

### 2.1 FF (none)

No confirmed counterexamples. All round-1 FFs (FF1, FF2, FF3) absorbed correctly. All 7 round-1 NITs absorbed inline as documented in round-2 changelog (PLAN.md L11-29).

### 2.2 NIT-R2-1 [Axis: heuristic-precision] [severity: low] — D6 partial-state detection heuristic susceptible to TOML-comment false-match

**Claim.** D6 detection heuristic (PLAN.md L320): "check if the existing `template.toml` content contains `\"[<group>]\"` or `\"[<group>.\"` for each selected group. Simple string check — not full TOML parse." A user with a TOML comment `# this section is for [go]` would falsely match `[go]` as a section header. The warning would suppress for genuine missing-group cases.

**Evidence.** PLAN.md L320 (D6 RiskNotes), L325 (D6 ContextBlocks `decision`).

**Fix hint.** Tighten the heuristic to look for `^\[<group>\]$` or `^\[<group>\.` (line-anchored). Or document the limitation more sharply. Or use `pelletier/go-toml/v2` to parse and check section presence — but that adds a TOML parse cost on every re-run.

**Disposition.** ABSORB at D6 dispatch — tighten the doc-comment to `// Heuristic: line-anchored prefix check, not full TOML parse; a comment containing "[<group>]" produces a false match` AND add a regex-based line-anchored check (cheap, no parse). This is a one-line tightening to D6's KindPayload `shape_hint` for `writeTemplateTOML`, not a re-plan.

---

### 2.3 NIT-R2-2 [Axis: cross-wave-coverage] [severity: medium] — agents.toml `[gen]` section absent from W4.D2's stated fixture; W2.D6 deferral assumes 3 sections

**Claim.** L1 W4.D2 acceptance (PLAN.md L433) asserts: "Both `[go]` and `[fe]` group sections present" in `agents.example.toml`. W2 round-2 PLAN.md Notes §agents.toml Gap (L439) describes Option (a) as: "containing `[go]` + `[fe]` + `[gen]` sections." Three-section vs two-section drift. A user running `till init --group gen` post-W4.D2 receives:

- Agent files copied to `<destDir>/.tillsyn/agents/gen/*.md` (10 files — D5).
- `<destDir>/template.toml` written aggregating `gen` template (D6).
- `<destDir>/agents.toml` copied from fixture — but the fixture lacks `[gen]` section per W4.D2 acceptance.

Result: inconsistent state. `gen` group is partially initialized: agent files + template.toml present, but agents.toml lacks the `[gen]` block.

**Evidence.**
- L1 PLAN.md L433 (W4.D2 acceptance): only `[go]` and `[fe]` sections specified.
- W2 round-2 PLAN.md L438-441: Notes §agents.toml Gap option (a) presumes `[go]` + `[fe]` + `[gen]`.
- W2 PLAN.md L442: "This decision requires orchestrator adjudication."

**Fix hint.** Orchestrator pre-W4.D2 dispatch should pick one:
- **(a)** Patch L1 W4.D2 acceptance to require `[go]` + `[fe]` + `[gen]` sections in `agents.example.toml`. Consistent with W2's deferral note. Simple — one additional section.
- **(b)** Pick Option (b) from W2 PLAN.md Notes — add a D6b droplet to W2 for multi-group agents.toml aggregation. Larger scope.

W2's planner recommended Option (a) (line 442). The fix is a single-line patch to L1 W4.D2 acceptance line 433.

**Disposition.** ROUTE TO ORCHESTRATOR — this is a CROSS-WAVE consistency item that round-2 W2 planner correctly surfaced. The orchestrator should patch L1 W4.D2 before W4.D2 dispatches. Not a W2 internal defect; W2 is faithful to its L1 contract.

---

### 2.4 NIT-R2-3 [Axis: dead-code-string-cleanup] [severity: low] — `(disabled — reserved for GDD)` View string lives in source during D1→D3 interim

**Claim.** D1 acceptance (PLAN.md L73): "All doc-comments referencing `Group string`, `till-gen`, `till-go`, `\"till-gdd\"`, `reservedInitGroups` are updated or removed." The literal runtime string `"(disabled — reserved for GDD)"` at `cmd/till/init_cmd.go:269` is a View-rendered label, not a doc-comment. It's dead string (no row has `Disabled: true` post-D1) but lives in source until D3 wholesale-replaces the View with `picker_multi.go`.

**Evidence.**
- `cmd/till/init_cmd.go:268-269` — current View renders this label conditional on `row.Disabled`.
- D1 PLAN.md L73 acceptance — covers doc-comments only, not runtime strings.
- D3 PLAN.md L165 — wholesale-replaces with picker_multi.go (deletes everything including the View).

**Fix hint.** Either:
- **(a)** Add a D1 acceptance bullet: "remove the `\"(disabled — reserved for GDD)\"` View-render label string from `init_cmd.go:269` (the `if row.Disabled` View branch). Since no row has `Disabled: true` post-D1, the branch is dead and the string is unreachable."
- **(b)** Leave the dead string until D3 replaces it. Pre-MVP-acceptable — dead string is harmless and D3 removes it within the same wave.

**Disposition.** ABSORB at D1 dispatch — either choice is fine. Recommend (a) (one-line edit) for cleanliness. Not load-bearing; either path produces the same end-state post-D3.

---

### 2.5 NIT-R2-4 [Axis: dispatch-optimization-tracking] [severity: low] — Fals NIT1 disposition (coarse blocked_by W5 for D3/D4) DOCUMENTED but no refinement entry tracks future tightening

**Claim.** W2 round-2 PLAN.md L424-426 documents: "D3 only needs `picker_multi.go` (W5.D4); D4 only needs `confirm.go` (W5.D2). Tightening to droplet-level external blockers is a dispatch-optimization; pre-Drop-4b dispatcher enforces at wave level. Kept at wave level for L2 simplicity. Future drop may tighten when the dispatcher supports droplet-level external-blocker granularity."

The Raised Refinements table (PLAN.md L402-405) lists only PLATFORM-TEMPLATES-R1 and W2-GROUPS-R1 (RESOLVED). The dispatch-optimization is mentioned in prose but not tracked as a refinement entry. If a future drop should tighten this, there's no refinement ID to surface in dogfood retrospectives.

**Evidence.**
- PLAN.md L424-426 (Notes §W5 Dependency Note dispatch-optimization paragraph).
- PLAN.md L402-405 (Raised Refinements — only 2 entries).

**Fix hint.** Add a refinement entry:
- `DISPATCH-EXTERNAL-BLOCKER-GRANULARITY-R1` — When Drop 4b dispatcher supports droplet-level external blockers, tighten W2.D3 `blocked_by` from `4c.6.1.W5` to `4c.6.1.W5.D4`, and W2.D4 from `4c.6.1.W5` to `4c.6.1.W5.D2`. Saves ~2-3h of unnecessary serialization in the cascade.

**Disposition.** ABSORB inline at L2 planner re-pass — add the refinement entry. One-line table addition.

---

## 3. Cross-Planner / Plan-QA-Discipline Confirmations

### 3.1 PLAN-QA-DISCIPLINE-R1 (test-runner droplet `blocked_by`) — PASS

Every D-series droplet's tests run within the droplet that lands the new behavior. R1's "test-runner droplet `blocked_by` includes the wave shipping the behavior" reduces to "each droplet `blocked_by` covers its own dependencies." Confirmed:
- D1 ← W4.D1 (canonical names).
- D2 ← D1 (Groups slice consumption).
- D3 ← D2, W5 (picker_multi).
- D4 ← D3, W5 (confirm.go).
- D5 ← D4, W4.D1 (embed path).
- D6 ← D5 (after copyAgentFiles).
- D7 ← D6 (after writeTemplateTOML).

All explicit. PASS.

### 3.2 PLAN-QA-DISCIPLINE-R2 (narrative ↔ enumeration) — PASS

7 narrative + 7 table + 7 body sections + 7 `_BLOCKERS.toml` rows + 7 Blockers Reference table rows. PASS.

### 3.3 Blocker-graph acyclicity — PASS

Topo-walk: W4.D1 → D1; W5 → D3, D4; D1 → D2 → D3 → D4 → D5 → D6 → D7. No back-edges. Acyclic.

### 3.4 Sibling-overlap-without-blocker — PASS

All 7 droplets share `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go` + package `cmd/till`. Fully serialized D1→D7. Cross-wave: W3 and W7.D3 share `cmd/till` package, `blocked_by W2` per L1. All package collisions resolved.

### 3.5 CONSUMER-TIE coverage — PASS (post-round-2 absorption)

D1 NIT2 absorbed (L74 explicit `run()` bullet); D3/D4 NIT3 absorbed (L167, L215 explicit `run(--json)` supplements); D2/D5/D6/D7 already explicit. All 7 droplets have explicit CONSUMER-TIE bullets.

### 3.6 Atomic-droplet sizing — PASS

D1: ~7 KindPayload changes (rename + delete + accessor + tests). D2: 4 changes (2 new fns + 1 mod + 2 tests). D3: 4 changes (TUI rewrite + dead-code deletion + 2 tests). D4: 4 changes (TUI step + accessor + 2 tests). D5: 3 changes (signature + caller + tests). D6: 3 changes (new fn + caller + tests). D7: 2 changes (createProjectDBRecord upgrade + tests). All within 1-4 code-blocks range.

### 3.7 R10 locked-decision compliance — PASS

- R10-D1 (canonical names `go`/`fe`/`gen`, no `till-` prefix): D1 hardcodes correctly; D5 embed path correctly; D7 Language mapping correctly.
- R10-D2 (typed `ProjectMetadata.Groups`): D7 uses typed field; KindPayload reflects this; W2-GROUPS-R1 marked RESOLVED.
- W5 component contract (no `tea.Quit`, `View() string` sub-component): D3 + D4 ContextBlocks correctly document these constraints.

---

## 4. Premises / Evidence / Trace / Conclusion / Unknowns

### Premises

- Round-1 PLAN_QA_PROOF.md (1 FF + 7 NITs) and PLAN_QA_FALSIFICATION.md (3 FFs + 7 NITs) are the absorption inventory.
- W1 round-2 PLAN.md ships typed `ProjectMetadata.Groups []string` per AC3.
- W5 round-2 PLAN.md ships `confirm.go` + `picker_multi.go` as Bubble Tea sub-components (NOT `tea.Model`) with accessor-based completion signaling.
- W4.D1 (Wave A) performs `git mv till-go → go` + `git mv till-gen → gen` + adds `fe/`.
- L1 PLAN.md R10 locked decisions (lines 910-938) are inherited constraints.
- `_BLOCKERS.toml` mirrors PLAN.md inline `Blocked by:` bullets; PLAN.md is truth on drift.

### Evidence

- W2 round-2 PLAN.md absorption changelog (L11-29) covers every round-1 finding with explicit disposition.
- W1 round-2 PLAN.md D2 acceptance + KindPayload (L264, L375-376) match W2.D7's consumption shape exactly (field name, slice element type, JSON tag).
- W5 round-2 PLAN.md AC4 + D4 acceptance (L86, L538-545) confirm `picker_multi.go` API contract matches W2.D3's blocker.
- `_BLOCKERS.toml` 7 rows cover all 7 droplets; cross-wave entries (W2.D1 ← W4.D1; D3/D4 ← W5; D5 ← W4.D1) all present and consistent.
- Existing `cmd/till/init_cmd.go` structure (line 404 `runInitPipeline`, line 410 `copyAgentFiles` call) confirms D2's FLAT-detection placement survives D5's signature refactor independently.
- `ProjectMetadata` current state (L119-155 of `internal/domain/project.go` — confirmed in round-1 proof) has NO `Groups` field today; W1.D2 adds it pre-W2.D7 dispatch.

### Trace / Cases

- **Round-1 FF1 absorption (till-prefix drift):** W4.D1's `git mv` lands canonical names; D1 hardcodes `["gen", "go", "fe"]`; D5 reads `builtin/agents/<group>/` with unprefixed names. Verified through-walk OK.
- **Round-1 FF2 absorption (D7 KindPayload stopgap):** D7 acceptance writes `Metadata.Groups = payload.Groups` directly; KindPayload shape_hint reflects this; W2-GROUPS-R1 marked RESOLVED. Verified.
- **Round-1 FF3 absorption (_BLOCKERS.toml drift):** W2.D1 entry present; all 7 entries match PLAN.md bullets. Verified.
- **All 7 round-1 NITs absorbed:** verified one-by-one against changelog L17-29.
- **MCP `*bool` accessor flow:** explicit `false` → non-nil pointer to false; omitted key → nil → `MCPRegistration()` returns true. Three CONSUMER-TIE cases cover the matrix. Verified.
- **D6 partial-state warning + remediation:** detection heuristic documented; remediation lossy-by-design (no-migration philosophy). Verified.
- **W4.D2 single-fixture vs W2.D6 deferral:** cross-wave gap on `[gen]` section absent from L1 W4.D2 acceptance; routed via NIT-R2-2.
- **CONSUMER-TIE complementarity:** teatest-direct + JSON-mode mirror cover complementary layers, NOT equivalent. Round-2 framing as "supplement" is correct.

### Conclusion

**PASS-WITH-NITs.** Round-2 plan is dispatchable. All round-1 FFs absorbed correctly. All round-1 NITs absorbed inline per changelog. Four new NITs surfaced for absorption at L2 planner re-pass or L1/W4 cross-wave routing:

1. **NIT-R2-1** (low): D6 detection heuristic line-anchoring. Absorb at D6 dispatch.
2. **NIT-R2-2** (medium): L1 W4.D2 acceptance line 433 `[gen]` section. Route to orchestrator for cross-wave fix.
3. **NIT-R2-3** (low): D1 dead View string `"(disabled — reserved for GDD)"`. Absorb at D1 dispatch or accept as D3-replaced.
4. **NIT-R2-4** (low): Add `DISPATCH-EXTERNAL-BLOCKER-GRANULARITY-R1` refinement entry. One-line table addition.

No counterexample lands. All ten attack vectors REFUTED.

### Unknowns

- **U-R2-1:** NIT-R2-2 disposition (Option a single-fixture-with-3-sections vs Option b new W2 droplet) requires dev/orchestrator adjudication. Surfaced via round-1 NIT6 DEFERRED + this round's reaffirmation.
- **U-R2-2:** Whether the partial-state warning's `"Remove it and re-run to regenerate"` remediation is acceptable when user has hand-edited `template.toml` (lossy `rm` destroys customization). Pre-MVP-acceptable per no-migration philosophy; revisit at MVP.

Both unknowns route via this MD; not blocking dispatch.

---

## 5. Hylla Feedback

N/A — Hylla was explicitly OFF for this review per spawn-prompt directive. All evidence-gathering used `Read` on filesystem-MD plans + targeted `Read` on `cmd/till/init_cmd.go` for placement verification.
