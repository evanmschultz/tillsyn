# PLAN_QA_PROOF — Drop 4c.6.1 W1 TEMPLATE_RESOLUTION — Round 2

**Verdict:** PASS-WITH-NIT — every round-1 finding (proof FF1/FF2 + proof NIT1-4 + falsification FF1/FF2/FF3 + falsification NIT1-10 = 16 dispositions total) is absorbed or explicitly deferred-with-reason in round-2 PLAN.md. R10-D1 and R10-D2 cross-cutting absorptions land cleanly in W1.D3 (constant renames + `blocked_by W4.D1`) and W1.D2 (typed `Groups []string`) respectively. `_BLOCKERS.toml` mirrors PLAN.md inline `Blocked by:` bullets one-for-one with no orphan references. LSP-verified symbols match the plan's "new, not yet in tree" claims (`loadProjectTemplateWithHome`, `loadProjectTemplatesForGroups`, `mergeTemplates`) and the "existing, to-be-modified" claims (`agentBodyDefaultGroup = "till-go"` at render.go:184; `agentBodyFallbackGroup = "till-gen"` at render.go:189; `readProjectTierAgent` at render.go:877 with current `(projectWorktree, basename string)` signature; `assembleAgentFileBody` at render.go:646; `loadProjectTemplate` at service.go:529; `bakeProjectKindCatalog` at service.go:416; `ProjectMetadata` at project.go:119 with no current `Groups` field).

Two NIT findings surface in round-2 that round-1 did not raise. Both are non-blocking for dispatch but should be folded into the builder's read-through.

---

## 1. Findings — FF (Load-Bearing)

None. All round-1 FFs (proof FF1, proof FF2, falsification FF1, falsification FF2, falsification FF3) are absorbed.

---

## 2. Findings — NIT (Cosmetic / Builder-Facing)

### NIT1 — `templates.Template` field count cited as "8" but struct has 9 fields

**Axis:** spec-conformance

**Line citation:** W1 PLAN.md line 150 (`templates.Template has 8 fields`), line 266 (D2 KindPayload `mergeTemplates` shape_hint: `doc-comment enumerates all 8 fields`), line 405 (D2 acceptance #4 prose `all 8 templates.Template fields`), line 424 (D2 RiskNotes `templates.Template has 8 fields (confirmed via internal/templates/schema.go)`).

**Evidence:** Direct read of `internal/templates/schema.go:150-248` enumerates exactly nine struct fields: `SchemaVersion` (154), `Kinds` (159), `ChildRules` (163), `AgentBindings` (167), `Agents` (184), `Gates` (200), `GateRulesRaw` (214), `Tillsyn` (223), `StewardSeeds` (247). PLAN.md line 153-168 itself enumerates nine bullets matching this list. The per-field merge specification is COMPLETE (no field is missing from the merge strategy); only the count text is off by one.

**Why NIT not FF:** The merge content is fully specified for all nine fields. The builder reading the bullets will write merge logic for all nine. The "8" is a textual count error that does not change behavior.

**Fix hint:** Replace "8 fields" with "9 fields" in all four occurrences (lines 150, 266 inside `shape_hint`, 405, 424). The bullet list itself is correct.

---

### NIT2 — D2 acceptance #1+#2 + RiskNotes + KindPayload claim `Groups: []string{}` is NOT omitted by `json:"groups,omitempty"`; standard Go `encoding/json` semantics omit it

**Axis:** spec-conformance / cross-language-semantics

**Line citation:** W1 PLAN.md line 184 (RiskNotes: *"`omitempty` on `[]string` omits nil slices only; an empty non-nil slice (`Groups: []string{}`) is NOT omitted"*), line 264 (D2 KindPayload `shape_hint`: *"omitempty omits nil slice only; empty non-nil slice is NOT omitted"*), line 268 (D2 KindPayload `TestProjectMetadata_Groups_RoundTrip` shape_hint: *"`Groups=[]string{}`→present (empty non-nil NOT omitted)"*), line 377-378 (D2 acceptance #1: *"`omitempty` omits nil slices only; empty non-nil slice (`Groups: []string{}`) is NOT omitted"*), line 382 (D2 acceptance #2: *"`Groups=[]string{}`→present (empty non-nil NOT omitted per JSON semantics)"*).

**Evidence:** Go `encoding/json` package documentation states: *"The 'omitempty' option specifies that the field should be omitted from the encoding if the field has an empty value, defined as false, 0, a nil pointer, a nil interface value, and any empty array, slice, map, or string."* The phrase "any empty array, slice, map, or string" includes BOTH `nil` AND `[]string{}` — both have `len(s) == 0`. The Go test verifying this is trivial:

```go
type T struct { G []string `json:"g,omitempty"` }
b, _ := json.Marshal(T{G: []string{}})  // produces `{}` — G OMITTED
```

The plan's claim "`Groups: []string{}` is NOT omitted" is factually wrong against `encoding/json`.

**Why NIT not FF (but worth flagging):** The behavior axis the plan cares about (additive field, no migration, round-trip safe) is preserved either way — both `nil` and `[]string{}` marshal/unmarshal cleanly. The D2 builder writing `TestProjectMetadata_Groups_RoundTrip` will write the test expecting `[]string{}` to round-trip as "present" and the test will FAIL against actual Go semantics. The builder will then either: (a) discover the misconception and fix the test to expect omission, or (b) switch `omitempty` off — which would be a wrong fix. This is a builder-trap.

**Fix hint:** Rewrite the three locations to state correct semantics:

- Line 184 (RiskNotes): *"`omitempty` on `[]string` omits both nil slices AND empty non-nil slices (`Groups: []string{}`) — per Go `encoding/json`, both have `len(s) == 0` and trigger omission. The zero value of `Groups` is nil, so freshly created projects marshal without the `groups` key; this is the desired additive behavior."*
- Line 264, 268, 377-378, 382 (KindPayload + acceptance): change `Groups=[]string{}`→present` to `Groups=[]string{}`→OMITTED (len-0 slice)`. The round-trip test should assert: nil→omitted; `["go","fe"]`→present; `[]string{}`→omitted.

The fix is one-line in each of the five citation spots and aligns the test contract with actual Go behavior.

---

## 3. Round-1 Finding Absorption Audit

Each round-1 finding mapped to its round-2 disposition + supporting evidence.

### Round-1 proof findings (PROOF_R1)

| Finding | Round-2 disposition | Evidence |
|---|---|---|
| Proof FF1 (orphan "per U1") | RESOLVED pre-round-2 by orchestrator | PLAN.md line 16; `_BLOCKERS.toml` line 18 no longer cites "per U1" |
| Proof FF2 (D1/D2 testability-seam ambiguous) | ABSORBED — pinned `loadProjectTemplateWithHome(project, homeDir, group)` as single seam | D1 KindPayload line 256 adds `loadProjectTemplateWithHome` as `action:"add"`; D1 acceptance #6 (line 330-333) requires the helper; D2 RiskNotes line 429-431 references D1's seam; D2 KindPayload line 265 calls `loadProjectTemplateWithHome` per group |
| Proof NIT1 (D2 paths hedge) | ABSORBED — pinned `internal/domain/project_test.go` | D2 Paths line 369; KindPayload line 268 references `project_test.go`; summary table line 581 |
| Proof NIT2 (`mergeTemplates` under-specified) | ABSORBED — all 9 fields enumerated (see NIT1 above) | RiskNotes line 153-168; D2 acceptance #4 lines 391-405 |
| Proof NIT3 (terminology drift between D1/D2 RiskNotes) | ABSORBED — single term `loadProjectTemplateWithHome` used throughout | D1 RiskNotes line 343; D2 RiskNotes line 429 |
| Proof NIT4 (summary table `*_test.go` glob) | ABSORBED — explicit filenames | Summary table line 580-581 spells out `project.go, project_test.go, service.go, service_test.go` |

### Round-1 falsification findings (FALSIFICATION_R1)

| Finding | Round-2 disposition | Evidence |
|---|---|---|
| Fals FF1 CRITICAL (till-prefix drift) | ABSORBED per R10-D1 | D3 acceptance #3+#4 lines 484-485; D3 KindPayload lines 276-277; D3 `blocked_by W4.D1` line 472 |
| Fals FF2 HIGH (W2/W3 stale on Groups) | ABSORBED — D2 acceptance #6 reaffirms typed field; W2/W3 PLAN.md updates routed via Round-2 changelog | D2 acceptance #6 lines 408-411 explicit cross-wave routing note; PLAN.md line 23 |
| Fals FF3 MEDIUM (mergeTemplates under-specified) | ABSORBED — per-field merge strategy enumerated for all 9 fields; MERGE-FIELD-AXIS-R1 raised | RiskNotes line 166-168; D2 acceptance #4 (391-405) |
| Fals NIT1 (D2 signature contradiction) | ABSORBED — resolved by FF2 fix; single signature `(project, homeDir, group)` | D2 acceptance #3 line 384; KindPayload line 265 |
| Fals NIT2 (D2 implicit signature change to D1) | ABSORBED — D2 calls D1 seam, no implicit change | D2 RiskNotes line 429-431 explicit |
| Fals NIT3 (`omitempty` for `[]string`) | ABSORBED — doc-comment note added (but semantics misstated; see NIT2 above) | D2 acceptance #1 line 377-378 |
| Fals NIT4 ("same author" prose) | DEFERRED-AS-NIT — `blocked_by` is structural; prose is informal | PLAN.md line 28 with reason |
| Fals NIT5 (render_test.go vague size) | ABSORBED — "1661 lines" | RiskNotes line 200; D3 ContextBlocks line 552-553 (warning critical) |
| Fals NIT6 (orphan "per U1") | RESOLVED — same as Proof FF1 | PLAN.md line 30 |
| Fals NIT7 (D3 KindPayload missing `agentBodyDefaultGroup` rename) | ABSORBED — folded into FF1 resolution | D3 KindPayload lines 276-277 |
| Fals NIT8 (D2 round-trip test) | ABSORBED — explicit round-trip test bullet | D2 KindPayload line 268 (`TestProjectMetadata_Groups_RoundTrip`); D2 acceptance #2 lines 380-383 |
| Fals NIT9 (whitespace-only Language) | DEFERRED-AS-NIT (REFUTED) — `strings.TrimSpace` already handles | PLAN.md line 33 with reason; D1 acceptance #2 line 323-324 |
| Fals NIT10 (D3 "one call site" not test gate) | DEFERRED-AS-NIT — LSP findReferences instruction sufficient | PLAN.md line 34 with reason; D3 RiskNotes line 509-511 |

All 16 round-1 findings disposed: 11 ABSORB, 3 DEFERRED-AS-NIT with reason, 2 RESOLVED pre-round-2.

---

## 4. R10 Cross-Cutting Absorption Audit

### R10-D1 (till-prefix subdir rename → canonical group names)

**Coverage in W1:**

- D3 scope EXPANDS to include both constant updates (`agentBodyDefaultGroup` `"till-go"`→`"go"` AND `agentBodyFallbackGroup` `"till-gen"`→`"gen"`).
- D3 `blocked_by W4.D1` correctly ordered (W4.D1 renames embedded FS dirs FIRST via `git mv till-go → go` + `git mv till-gen → gen` per L1 PLAN.md lines 351-352).
- D3 acceptance bullets #3 + #4 (PLAN.md lines 484-485) state both constant updates.
- D3 KindPayload entries (lines 276-277) state both constant modifications.
- D3 ContextBlocks (line 547-551) warning-critical flags `agentBodyFallbackGroup` rename dependency on W4.D1's `git mv till-gen → gen`.
- Acceptance #7 (line 121-122) explicitly preserves cross-group fallback to `gen` group via `readEmbeddedTierAgent` reading from `builtin/agents/gen/<basename>` AFTER W4.D1's rename.

**Verdict:** R10-D1 fully wired in W1. ✓

### R10-D2 (Groups typed field)

**Coverage in W1:**

- D2 ships `domain.ProjectMetadata.Groups []string` with `json:"groups,omitempty"` (PLAN.md lines 264, 374-378).
- D2 acceptance #6 (lines 408-411) explicitly states W2.D7 + W3.D1 consume typed field directly; NO `KindPayload` JSON stopgap; NO TODO fallback; W2-GROUPS-R1 RESOLVED inline.
- D2 RiskNotes line 460 cross-wave routing note for W2.D7 + W3.D1.

**Verdict:** R10-D2 fully wired in W1. ✓

---

## 5. `_BLOCKERS.toml` ↔ PLAN.md Mirroring Audit

| PLAN.md inline `Blocked by:` | `_BLOCKERS.toml` row | Match |
|---|---|---|
| D1 line 314: "W4.D1" | line 6-8: `W1.D1` blocked_by `["4c.6.1.W4.D1"]` | ✓ |
| D2 line 367: "D1" | line 10-13: `W1.D2` blocked_by `["W1.D1"]` | ✓ |
| D3 line 472: "W4.D1" | line 16-18: `W1.D3` blocked_by `["4c.6.1.W4.D1"]` | ✓ |

`_BLOCKERS.toml` has NO orphan "per U1" reference (RESOLVED pre-round-2). Mirroring is one-to-one and free of drift.

---

## 6. PLAN-QA-DISCIPLINE Checks

### PLAN-QA-DISCIPLINE-R1 (every new-behavior acceptance → test-runner blocked_by ships it)

| Acceptance | Behavior shipped by | Tested by (droplet) | Same-droplet test+behavior? |
|---|---|---|---|
| AC1 4-tier walk | D1 | D1 (`service_test.go` — `TestLoadProjectTemplate_HomeTier`) | ✓ |
| AC2 HOME wins over embedded | D1 | D1 (same test) | ✓ |
| AC3 multi-group + `Groups []string` | D2 | D2 (`project_test.go` round-trip + `service_test.go` multi-group) | ✓ |
| AC4 subdir-per-group | D3 | D3 (`render_test.go` — `TestReadProjectTierAgent_SubdirPerGroup`) | ✓ |
| AC5 constant renames | D3 | D3 (updated fixtures) | ✓ |
| AC6 cross-group `gen` fallback preserved | (preservation; W4.D1 dependency) | D3 (existing tests pass after constant + fixture updates) | ✓ |
| AC7 `mage ci` green | D1+D2+D3 | post-D1+D2+D3 sequential gate | ✓ |

**PASS.** Every new behavior is shipped by a droplet that also ships its test.

### PLAN-QA-DISCIPLINE-R2 (narrative droplet count = enumerated D-list count)

- Scope narrative (lines 50-71): D1, D2, D3 = **3**.
- ### Droplets enumerated (lines 309, 361, 466): D1, D2, D3 = **3**.
- Summary table (lines 578-582): D1, D2, D3 = **3**.
- Footer (line 584): `Total: **3 atomic droplets**.`

All counts agree. **PASS.**

---

## 7. LSP / Read Verification of Symbol References

| Symbol | Cited at | LSP/Read result |
|---|---|---|
| `loadProjectTemplate` | service.go:529 | EXISTS, current signature `(project *domain.Project) (templates.Template, bool, error)` |
| `loadProjectTemplateCandidate` | service.go:581 | EXISTS |
| `bakeProjectKindCatalog` | service.go:416 | EXISTS |
| `loadProjectTemplateWithHome` | (D1 ships) | NEW — not in tree; D1 authors |
| `loadProjectTemplatesForGroups` | (D2 ships) | NEW — not in tree; D2 authors |
| `mergeTemplates` | (D2 ships) | NEW — not in tree; D2 authors |
| `readProjectTierAgent` | render.go:877 | EXISTS, current signature `(projectWorktree, basename string) (string, bool, error)` |
| `readUserTierAgent` | render.go:898 | EXISTS, signature `(group, basename string)` — pattern to mirror confirmed |
| `assembleAgentFileBody` | render.go:646 | EXISTS, calls `readProjectTierAgent` at line 666 (CONFIRMED — single call site to update) |
| `resolveAgentGroup` | render.go:860 | EXISTS — returns `agentBodyDefaultGroup` (currently `"till-go"`) when binding path empty |
| `agentBodyDefaultGroup` | render.go:184 | EXISTS as `const agentBodyDefaultGroup = "till-go"` |
| `agentBodyFallbackGroup` | render.go:189 | EXISTS as `const agentBodyFallbackGroup = "till-gen"` |
| `projectAgentsSubdir` | render.go:193 | EXISTS as `const projectAgentsSubdir = ".tillsyn/agents"` |
| `ProjectMetadata` | project.go:119 | EXISTS — has NO `Groups []string` field currently (additive change for D2 confirmed) |
| `templates.Template` 9 fields | schema.go:150-248 | CONFIRMED — 9 fields (see NIT1 above; plan text says "8") |

All "existing" claims verified; all "new, not yet in tree" claims confirmed absent from current code.

---

## 8. Conclusion

**Verdict: PASS-WITH-NIT.**

Round-2 absorbs all 16 round-1 findings with appropriate dispositions. R10-D1 and R10-D2 cross-cutting decisions land cleanly in W1.D3 and W1.D2. `_BLOCKERS.toml` mirrors PLAN.md. PLAN-QA-DISCIPLINE-R1 and R2 both PASS. All LSP-claimed symbols verified.

Two NIT findings surface that round-1 did not catch:

1. **NIT1 (cosmetic):** `templates.Template` field count cited as "8" four times; the struct has 9 fields and the per-field bullet list itself enumerates 9. Content correct; count text off by one.
2. **NIT2 (load-bearing for builder test fidelity):** Multiple acceptance + RiskNote citations claim `Groups: []string{}` is NOT omitted by `json:"groups,omitempty"`; per Go `encoding/json` semantics, both nil AND len-0 slices are omitted. The builder writing `TestProjectMetadata_Groups_RoundTrip` will write a test that fails against actual Go behavior if these citations are followed verbatim. Fix is one-line in each of five citation spots and aligns the test contract with real `encoding/json` behavior.

Recommend: orchestrator-direct fix of both NITs before dispatching D1/D2/D3 (small edits, no round-3 planner pass required), then dispatch.
