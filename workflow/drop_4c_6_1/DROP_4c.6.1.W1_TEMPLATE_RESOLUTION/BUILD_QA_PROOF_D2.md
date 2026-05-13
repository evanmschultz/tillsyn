# W1.D2 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

---

## Acceptance Bullet Coverage

### AC #1 — `ProjectMetadata.Groups` field declared with correct shape

**Quote:**
> `domain.ProjectMetadata.Groups` field exists: `Groups []string` with JSON tag `json:"groups,omitempty"`. Per Go `encoding/json`, `omitempty` on `[]string` omits BOTH `nil` AND empty-non-nil slices (`len(s) == 0` is treated as empty for slices). Only non-empty slices appear in marshaled output. Zero value is nil. Additive field — existing `ProjectMetadata` marshal/unmarshal round-trips are unaffected. Builder confirms no full-struct-literal comparisons on `ProjectMetadata` break via LSP `findReferences`.

**Evidence:**
- `internal/domain/project.go:168` — `Groups []string \`json:"groups,omitempty"\``.
- Doc-comment at lines 155-167 covers omitempty semantics and zero-value rationale.
- `mage test-pkg ./internal/domain` PASS (307/307).
- `mage test-pkg ./internal/tui` PASS — `TestProjectSchemaCoverageIsExplicit` updated to classify `Groups` as internal (model_test.go:15054-15060).
- `mage ci` PASS (3194/3194 tests, internal/domain coverage 81.8%).

**Verdict:** PASS.

---

### AC #2 — Round-trip test demonstrates omitempty behavior

**Quote:**
> `ProjectMetadata.Groups` marshal/unmarshal round-trip test in `project_test.go` confirms: `Groups=nil`→omitted; `Groups=["go","fe"]`→present; `Groups=[]string{}`→omitted (both nil and len==0 slices are omitted per Go encoding/json semantics).

**Evidence:**
- `internal/domain/project_test.go:269-349` — `TestProjectMetadata_Groups_RoundTrip` table-driven 3 cases:
  - `nil_groups_omitted`: marshal asserts substring `"groups"` is NOT present; decoded `Groups` must be nil.
  - `non_empty_groups_present`: marshal asserts `"groups":["go","fe"]` substring present; decoded value matches input.
  - `empty_slice_groups_omitted`: marshal asserts `"groups"` is NOT present; decoded `Groups` must be nil.
- `mage test-func ./internal/domain TestProjectMetadata_Groups_RoundTrip` → 4 tests passed (parent + 3 sub-tests). PASS.

The implementation matches the PLAN.md spec body (`empty_slice → omitted`) rather than the contradictory KindPayload line (`empty non-nil NOT omitted`). The spec body is authoritative per the spawn-prompt's instruction "resolved per spec body". RESOLVED.

**Verdict:** PASS.

---

### AC #3 — `loadProjectTemplatesForGroups` coordinator + empty-string guard

**Quote:**
> New `loadProjectTemplatesForGroups(project *domain.Project, homeDir string)` helper in `service.go` (package-private). Calls `loadProjectTemplateWithHome` (shipped by D1) with each non-empty group in `project.Metadata.Groups`. Guards against empty-string entries in `Groups` (skips entries where `strings.TrimSpace(group) == ""`). Merges resulting `templates.Template` values via `mergeTemplates`.

**Evidence:**
- `internal/app/service.go:490-519` — function defined with the exact signature `(project *domain.Project, homeDir string) (templates.Template, bool, error)`.
- Empty-string guard at lines 493-495: `if strings.TrimSpace(group) == "" { continue }`. This correctly catches both `""` AND whitespace-only entries (e.g., `"  "`).
- Calls `loadProjectTemplateWithHome(project, homeDir, group)` at line 497 — confirms D1-seam reuse.
- Calls `mergeTemplates(merged, tpl)` at line 505 for the merge step (first non-empty group seeds `merged`, subsequent groups merge in).
- Fallback to embedded default at lines 508-517 when all entries are empty/whitespace — defensive correctness.
- Test `empty_string_in_groups_skipped_without_error` at service_test.go:7118-7142 covers Groups containing both `""` and `"  "`. PASS.

**Verdict:** PASS.

---

### AC #4 — `mergeTemplates` per-field strategy for all 9 fields

**Quote:**
> New `mergeTemplates(base, overlay templates.Template) templates.Template` helper in `service.go` (package-private). Per-field merge strategy:
> - `SchemaVersion`: last-group-wins (overlay overwrites base).
> - `Kinds`: per-key last-group-wins.
> - `ChildRules`: append base + overlay; dedup on `(WhenParentKind, CreateChildKind)` tuple, overlay entry wins on collision.
> - `AgentBindings`: per-key last-group-wins (primary multi-group use case).
> - `Agents`: per-key last-group-wins.
> - `Gates`: per-key last-group-wins (overlay slice replaces base slice for same kind; NOT concat).
> - `GateRulesRaw`: per-key shallow merge, last-group-wins on collision.
> - `Tillsyn`: whole-struct last-group-wins; overlay `Tillsyn` replaces base if overlay is non-zero (`MaxContextBundleChars != 0 || MaxAggregatorDuration != 0 || SpawnTempRoot != ""`).
> - `StewardSeeds`: append base + overlay (no dedup; seeds are project-unique).
> - Doc-comment on `mergeTemplates` enumerates all 9 fields.

**Evidence:**
- `internal/app/service.go:548-641` — `mergeTemplates` body.
- Per-field implementation:
  - `SchemaVersion` (552-554): `if overlay.SchemaVersion != "" { out.SchemaVersion = overlay.SchemaVersion }` — last-group-wins. Matches.
  - `Kinds` (556-564): per-key copy-from-overlay. Matches.
  - `ChildRules` (566-586): builds an index keyed on `childRuleKey{WhenParentKind, CreateChildKind}` over base entries, then for each overlay entry either replaces in place (collision) or appends. Matches "append + dedup, overlay wins" semantics.
  - `AgentBindings` (588-596): per-key copy-from-overlay. Matches.
  - `Agents` (598-606): per-key copy-from-overlay. Matches.
  - `Gates` (608-616): per-key copy-from-overlay (entire slice replaced, no concat). Matches.
  - `GateRulesRaw` (618-626): per-key shallow merge with overlay-wins. Matches.
  - `Tillsyn` (628-633): replaces base if overlay's `MaxContextBundleChars`/`MaxAggregatorDuration`/`SpawnTempRoot` are non-zero. RequiresPlugins is explicitly excluded from the non-zero check per the doc-comment hedge. Matches spec.
  - `StewardSeeds` (635-637): `out.StewardSeeds = append(out.StewardSeeds, overlay.StewardSeeds...)` — no dedup. Matches.
- Doc-comment at lines 521-547 enumerates all 9 fields in order, with explicit semantics for each. The 9-field enumeration matches `templates.Template` struct fields (verified via `internal/templates/schema.go:150-248`).

**Verdict:** PASS.

---

### AC #5 — `bakeProjectKindCatalog` branches on non-empty `Groups`

**Quote:**
> `bakeProjectKindCatalog` branches: if `project.Metadata.Groups` is non-empty, call `loadProjectTemplatesForGroups`; else call `loadProjectTemplate` (existing path).

**Evidence:**
- `internal/app/service.go:422-431` — `bakeProjectKindCatalog` resolves `homeDir` from `os.UserHomeDir()` (skip on err / whitespace) and delegates to `bakeProjectKindCatalogWithHome`.
- `internal/app/service.go:443-470` — `bakeProjectKindCatalogWithHome` is the testability seam. Lines 447-456 implement the branch:
  ```
  if len(project.Metadata.Groups) > 0 {
      tpl, ok, err = loadProjectTemplatesForGroups(project, homeDir)
  } else {
      tpl, ok, err = loadProjectTemplate(project)
  }
  ```
- Test `both_groups_present_aggregated` at service_test.go:7005-7042 confirms multi-group path routes through coordinator.
- Pre-existing `TestBakeProjectKindCatalog_EmbeddedFallbackPopulatesCatalog` (service_test.go:7160+) confirms single-group path still works.

**Verdict:** PASS.

---

### AC #6 — W2.D7 + W3.D1 must consume typed field directly

**Quote:**
> W2.D7 and W3.D1 MUST consume `project.Metadata.Groups` typed field directly. They MUST NOT use `KindPayload` JSON stopgap. The orchestrator updates W2 + W3 PLAN.md before dispatching those droplets. (W2-GROUPS-R1 refinement RESOLVED inline by this droplet.)

**Evidence:**
- D2's responsibility here is to ship the typed field — which is done (AC#1).
- W2/W3 PLAN.md sync is out-of-W1.D2 scope; the orchestrator owns it.
- `Groups []string` field shipped at `internal/domain/project.go:168` is consumable by W2.D7/W3.D1 directly.

**Verdict:** PASS (the typed field exists and is consumable; the orchestrator-side W2/W3 sync is a separate responsibility).

---

### AC #7 — `mage test-pkg ./internal/domain` passes

**Quote:**
> `mage test-pkg ./internal/domain` passes.

**Evidence:**
- `mage test-pkg ./internal/domain` → `307 tests passed across 1 package`. PASS.
- `mage test-func ./internal/domain TestProjectMetadata_Groups_RoundTrip` → 4 tests passed (parent + 3 sub-tests). PASS.

**Verdict:** PASS.

---

### AC #8 — `mage test-pkg ./internal/app` passes + multi-group test coverage

**Quote:**
> `mage test-pkg ./internal/app` passes. New `TestBakeProjectKindCatalog_MultiGroup` covers:
> (a) 2 groups, both HOME files present → aggregated bindings contain entries from both groups;
> (b) 2 groups, one HOME file absent → absent group uses embedded fallback;
> (c) collision on same kind key → last group wins;
> (d) empty-string entry in `Groups` → skipped without error.

**Evidence:**
- `mage test-pkg ./internal/app` → `488 tests passed`. PASS.
- `mage test-func ./internal/app TestBakeProjectKindCatalog_MultiGroup` → 5 tests passed (parent + 4 sub-tests). PASS.
- Sub-test coverage:
  - (a) `both_groups_present_aggregated` (lines 7005-7042): seeds two HOME files for `go` (KindBuild binding) and `fe` (KindResearch binding); asserts both bindings present in catalog. PASS.
  - (b) `one_group_absent_uses_embedded_fallback` (lines 7043-7072): only `go` HOME file present; asserts no error and KindBuild binding from `go` present in catalog. PASS.
  - (c) `collision_same_kind_key_last_group_wins` (lines 7073-7113): both groups declare `[agent_bindings.build]` with different models; asserts `fe`'s `"opus"` model wins. PASS.
  - (d) `empty_string_in_groups_skipped_without_error` (lines 7114-7142): Groups = `["go", "", "  "]`; asserts no error and `go`'s KindBuild binding present. PASS.
- `mage ci` overall: 3194/3194 PASS; internal/app coverage 71.7% (above 70%); minimum coverage threshold met.

**Verdict:** PASS.

---

## NITs

### NIT 1 — Out-of-scope changes in working tree (cmd/till/init_cmd.go, cmd/till/init_cmd_test.go)
**Severity:** medium
**Recommended action:** Orchestrator must commit W1.D2 changes separately from W2.D1 changes.

The current working tree contains uncommitted changes to:
- `cmd/till/init_cmd.go` (+74 -53)
- `cmd/till/init_cmd_test.go` (+243 -71)

These files are NOT in W1.D2's declared `paths`. The diff content (`initJSONPayload.Groups` field with `[]string`, `initJSONPayload.MCP *bool`, `MCPRegistration()` helper, `allowedInitGroups = ["gen", "go", "fe"]`) belongs to W2.D1's scope per `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/PLAN.md`.

The W1.D2 build subset (per declared paths) is correct and passes QA. The cmd/till changes do not interfere with D2's acceptance criteria — `mage ci` is green with both — but they should be committed separately to keep the W1.D2 commit atomic per the drop's PLAN.md `Paths:` declaration.

This is the orchestrator's responsibility to handle at commit-staging time. No code defect.

---

### NIT 2 — `mergeTemplates` coverage matrix only exercises AgentBindings collision directly
**Severity:** low
**Recommended action:** Defer to dogfood refinement (already captured as `MERGE-FIELD-AXIS-R1` in spec).

AC #4 enumerates per-field merge strategy for all 9 fields. The implementation is correct on every field (verified by code-read above). However, the test matrix `TestBakeProjectKindCatalog_MultiGroup` only directly exercises:
- AgentBindings aggregation across two non-colliding kinds (case a)
- Embedded-fallback after-merge correctness (case b)
- AgentBindings collision last-wins (case c)
- Empty-group skip (case d)

The other 8 fields (`SchemaVersion`, `Kinds`, `ChildRules` collision-dedup, `Agents`, `Gates` slice-replace-not-concat, `GateRulesRaw` shallow merge, `Tillsyn` whole-struct replace + RequiresPlugins-exclusion, `StewardSeeds` append-no-dedup) are not exercised by dedicated test cases. The spec already raised `MERGE-FIELD-AXIS-R1` as a future refinement to revisit when multi-group projects exercise these fields in dogfood. Acceptance #4's wording focuses on the field-strategy spec + doc-comment, both of which are satisfied. Direct unit tests for each merge branch would harden the contract; this is a NIT not a FAIL because the spec explicitly defers it.

---

### NIT 3 — `mergeTemplates` no direct unit test (exercised only indirectly via `bakeProjectKindCatalogWithHome`)
**Severity:** low
**Recommended action:** Consider adding `TestMergeTemplates` table-driven unit test in a future iteration.

`mergeTemplates` is package-private and exercised only through `loadProjectTemplatesForGroups` -> `bakeProjectKindCatalogWithHome`. Adding a dedicated table-driven test would isolate merge semantics from the TOML-load + bake layer, making future per-field-strategy iterations (per `MERGE-FIELD-AXIS-R1`) easier to author and review. This compounds with NIT 2 (the same root cause: indirect coverage through baked catalog rather than direct merge assertions).

---

### NIT 4 — Doc-comment hedge re: RequiresPlugins non-zero check could surprise readers
**Severity:** low
**Recommended action:** Note acknowledged.

The doc-comment for `mergeTemplates` (line 537-540) explicitly states "RequiresPlugins does NOT contribute to the non-zero check per the spec-enumerated condition; the three named fields are the semantically meaningful non-zero signals."

This matches the spec body's enumeration (AC #4 lists only the three named fields). However, this means a hypothetical overlay declaring ONLY `[tillsyn].requires_plugins = [...]` (without the other three fields) would NOT replace base's `Tillsyn` struct. The behavior is documented but subtle; future dogfood may want to revisit, which the existing `MERGE-FIELD-AXIS-R1` refinement already covers.

---

## Verdict rationale

All 8 acceptance criteria are satisfied with file:line evidence. The build is `mage ci` green (3194/3194 tests pass, coverage thresholds met). The 9-field `mergeTemplates` doc-comment is complete and matches the spec. The 4-case multi-group test exercises aggregation, fallback, last-wins, and empty-string-skip behaviors as required. The round-trip test correctly demonstrates omitempty behavior for `[]string` (both nil and `[]string{}` omitted, matching the spec body's authoritative resolution of the KindPayload contradiction).

Two NIT categories surfaced:
1. **NIT 1** (medium): cmd/till changes in the working tree are W2.D1 scope, not W1.D2. Orchestrator must commit-stage atomically per droplet `paths`.
2. **NITs 2-4** (low): merge-semantics test coverage is narrow (only AgentBindings exercised directly); deferred per the spec-acknowledged `MERGE-FIELD-AXIS-R1` refinement.

No FAIL conditions. NIT 1 is a staging/commit-discipline concern, not a code-correctness defect. NITs 2-4 are test-coverage refinements already routed through `MERGE-FIELD-AXIS-R1`.

**Overall: PASS WITH NITS.**
