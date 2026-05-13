# W1.D2 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

W1.D2 acceptance criteria satisfied per per-package tests. Implementation matches spec acceptance #4 (9-field `mergeTemplates`). The two open items are (a) one planner-side spec gap (`Tillsyn` non-zero check omits `RequiresPlugins`) that the builder correctly followed, and (b) one hermeticity regression inherited from W1.D1 that now extends to two more test sites. Neither is a builder-side defect.

---

## Attack Hypotheses Tested

### H1 — `mergeTemplates` per-field correctness (5 map fields)

- **Hypothesis:** One of `Kinds`, `AgentBindings`, `Agents`, `Gates`, `GateRulesRaw` could be first-wins or merge-both instead of last-wins.
- **Test:** Read `internal/app/service.go:559-626`. Each map field uses the canonical `for k, v := range overlay.X { out.X[k] = v }` pattern, which is last-wins on key collision.
- **Finding:** Test `collision_same_kind_key_last_group_wins` (service_test.go:7079-7110) verifies AgentBindings explicitly: groups `["go", "fe"]` with model `"sonnet"` and `"opus"` respectively → decoded catalog returns `"opus"` (fe-group wins). Pattern is correct.
- **Verdict:** REFUTED.

### H2 — `ChildRules` dedup tuple correctness

- **Hypothesis:** Dedup might use only `WhenParentKind` or only `CreateChildKind` instead of the tuple; or overlay might not win on collision.
- **Test:** Read service.go:578-595. Inline struct key `{WhenParentKind, CreateChildKind}` is the dedup discriminator; overlay-wins via `out.ChildRules[i] = r`; non-matching entries preserved via `append`.
- **Finding:** Implementation matches spec. Not exercised by any test in service_test.go's multi-group block.
- **Verdict:** REFUTED on correctness; **NIT-1** raised for test coverage gap.

### H3 — `Tillsyn` whole-struct non-zero check

- **Hypothesis:** Overlay `Tillsyn` with only `RequiresPlugins` set (other 3 fields zero) should replace base if last-group-wins is the semantic, but the implementation's non-zero check would silently drop it.
- **Test:** Read service.go:628-633. Non-zero check: `overlay.Tillsyn.MaxContextBundleChars != 0 || overlay.Tillsyn.MaxAggregatorDuration != 0 || overlay.Tillsyn.SpawnTempRoot != ""`. `Tillsyn` struct has 4 fields including `RequiresPlugins` (schema.go:300+) — only 3 are checked.
- **Finding:** Spec acceptance #4 line 405-406 LITERALLY enumerates only these 3 fields: `"Tillsyn: whole-struct last-group-wins; overlay Tillsyn replaces base if overlay is non-zero (MaxContextBundleChars != 0 || MaxAggregatorDuration != 0 || SpawnTempRoot != "")"`. Builder explicitly called out this gap in doc-comment (service.go:545-548): `"RequiresPlugins does NOT contribute to the non-zero check per the spec-enumerated condition"`. Builder followed spec literally. This is a **planner-side spec gap** — concrete counterexample: `overlay.Tillsyn={RequiresPlugins:["plugin-a"]}` with other 3 fields zero will silently drop the RequiresPlugins value. The refinement `MERGE-FIELD-AXIS-R1` is already raised by the spec body to revisit Tillsyn semantics.
- **Verdict:** CONFIRMED gap but **spec-conforming**. Routed to NIT-2 + refinement-tracking.

### H4 — `StewardSeeds` append no-dedup

- **Hypothesis:** Seeds with identical content might be double-added on collision; or implementation might dedup unexpectedly.
- **Test:** Read service.go:635-638. `out.StewardSeeds = append(out.StewardSeeds, overlay.StewardSeeds...)`. Pure append, no dedup logic.
- **Finding:** Implementation matches spec ("no dedup; seeds are project-unique and append order is significant"). Identical content from both base and overlay would produce two entries. Not exercised by tests.
- **Verdict:** REFUTED on correctness; **NIT-1** raised for test coverage gap.

### H5 — Empty-string `Groups` entry handling (`Groups=["", "go"]`)

- **Hypothesis:** The skip predicate might short-circuit the entire loop on first empty string, missing valid groups after.
- **Test:** Read service.go:510-525. Loop body: `if strings.TrimSpace(group) == "" { continue }` — `continue` skips the empty entry and proceeds to next iteration. Test `empty_string_in_groups_skipped_without_error` (service_test.go:7113+) uses `Groups: ["go", "", "  "]` and asserts "go" group's binding made it into the catalog.
- **Finding:** Skip is per-entry, not loop-terminating. Whitespace-only strings (e.g. `"  "`) also skipped via `TrimSpace`. Note: test orders `Groups` with non-empty FIRST. A test with empty FIRST (e.g. `["", "go"]`) would more cleanly prove the non-short-circuit property.
- **Verdict:** REFUTED on correctness; **NIT-3** raised for ordering symmetry in tests.

### H6 — Empty-`Groups` vs `[""]` semantic distinction

- **Hypothesis:** `Groups=nil` and `Groups=[""]` might produce different behavior than intended.
- **Test:** Trace `bakeProjectKindCatalogWithHome`:
  - `Groups=nil`: `len(Groups) == 0` → routes to `loadProjectTemplate(project)` (single-group path).
  - `Groups=[""]`: `len(Groups) == 1` → routes to `loadProjectTemplatesForGroups`, loop skips empty, `hasMerged==false`, fallthrough to `templates.LoadDefaultTemplateForLanguage(project.Language)`.
- **Finding:** Different code paths, but semantically equivalent when `project.Language` matches an embedded default. Subtle inconsistency: `Groups=[""]` ignores `RepoBareRoot`/`RepoPrimaryWorktree` candidates entirely (the fallthrough goes straight to embedded), whereas `Groups=nil` walks all 4 tiers. If a dev sets `Groups=[""]` by accident (e.g., parse of a YAML field that produces a singleton empty), bare-root and primary-worktree templates would be ignored. Not tested.
- **Verdict:** REFUTED on hard correctness (no acceptance bullet violated); **NIT-4** raised for edge-case clarity and test gap.

### H7 — Hermeticity regression in `bakeProjectKindCatalog`

- **Hypothesis:** `bakeProjectKindCatalog` now calls `os.UserHomeDir()` at line 427. Existing tests that call this wrapper (NOT the new seam) become dependent on real `$HOME`.
- **Test:** `rtk grep` for callers identifies two test sites:
  - `TestBakeProjectKindCatalog_EmbeddedFallbackPopulatesCatalog` (service_test.go:7160) — `domain.Project{Language: "go"}`, asserts `SchemaVersion==V1` and `len(Kinds)>0`.
  - `TestBakeProjectKindCatalog_NonEmptyPathFallsThroughToEmbedded` (service_test.go:7188) — `domain.Project{RepoBareRoot: t.TempDir(), RepoPrimaryWorktree: t.TempDir(), Language: "go"}`, same assertions.
- **Finding:** Both tests rely on the embedded fallback. If the dev creates `~/.tillsyn/templates/go.toml` with a valid v1 template (the whole POINT of the W1 HOME tier feature), `loadProjectTemplate` (W1.D1) would resolve that file BEFORE the embedded fallback. `SchemaVersion==V1` and `len(Kinds)>0` would still pass coincidentally for any valid template, so the test passes but no longer tests what its name claims (the EMBEDDED fallback). This is an INHERITED hermeticity regression from W1.D1 — `BUILD_QA_FALSIFICATION_D1.md` Counterexample §1 already documents the dormant regression for `loadProjectTemplate` direct-callers. W1.D2's wrapper-level `os.UserHomeDir()` call is technically redundant when Groups is empty (the embedded path goes through `loadProjectTemplate` which re-calls `os.UserHomeDir()` itself) but doesn't add new exposure beyond what W1.D1 already established.
- **Verdict:** CONFIRMED but **inherited and dormant** on current dev + CI ($HOME has no `.tillsyn/templates/`). Routed to NIT-5.

### H8 — `omitempty` round-trip claim

- **Hypothesis:** Builder claims `omitempty` omits both nil AND `len==0` slices per Go encoding/json. Could be wrong about Go behavior.
- **Test:** Per Go encoding/json: `omitempty` on slice types omits when the value is "empty" — defined as `len == 0` for slices (which covers nil AND non-nil zero-length). Builder's round-trip test (project_test.go:299-303) asserts `Groups=[]string{}` produces `decoded.Groups == nil` (line 329-332). Ran `mage test-func ./internal/domain TestProjectMetadata_Groups_RoundTrip`: 4/4 PASS.
- **Finding:** Behavior correct. Both nil and empty-non-nil omitted. Decoded value is nil on unmarshal of JSON-without-key (encoding/json doesn't synthesize an empty slice).
- **Verdict:** REFUTED.

### H9 — `KindPayload` shape_hint contradiction

- **Hypothesis:** PLAN.md KindPayload says `Groups=[]string{}→present` while spec body says `→omitted`. Builder claimed to follow spec body.
- **Test:** Read PLAN.md:270 (KindPayload): `"Groups=[]string{}→present (empty non-nil NOT omitted)"`. Read PLAN.md:386 (acceptance #2): `"Groups=[]string{}→omitted (both nil and len==0 slices are omitted per Go encoding/json semantics)"`.
- **Finding:** PLAN.md is internally contradictory. KindPayload bullet (line 270) is WRONG per actual Go semantics. Spec acceptance (line 386) is correct. Builder followed correct spec, ignored incorrect KindPayload.
- **Verdict:** REFUTED on build. **NIT-6** (planner-side documentation fix to PLAN.md:270).

### H10 — TUI schema-coverage scope extension

- **Hypothesis:** `internal/tui/model_test.go` is touched outside W1.D2's declared `Packages: internal/domain, internal/app`. Could be illegitimate scope creep.
- **Test:** Read model_test.go:15054-15060 — adds `"Groups": {}` to `projectMetadataInternal` map. `TestProjectSchemaCoverageIsExplicit` asserts every `ProjectMetadata` field is classified as editable / readOnly / internal; an unclassified field fails the test. Adding `Groups` to `ProjectMetadata` WITHOUT classifying it would FAIL `mage ci`.
- **Finding:** Necessary scope extension forced by the schema-coverage invariant. Same pattern as `OrchSelfApprovalEnabled` / `DispatcherCommitEnabled` / `DispatcherPushEnabled` already in this map. Legitimate.
- **Verdict:** REFUTED. **NIT-7** raised for planner methodology: when modifying domain struct shapes, add `internal/tui` to the build's `Packages` so the schema-coverage update is in-scope from the start.

### H11 — Cross-package compile lock with in-flight Wave C builders

- **Hypothesis:** W1.D2 touches `internal/domain` + `internal/app` + (forced) `internal/tui`. A parallel Wave C builder might share these packages or files.
- **Test:** Inspected uncommitted state. `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go` carry W2-scope changes (`Group→Groups` schema rename + `MCP *bool` pointer) NOT in W1.D2's declared paths. These were not introduced by W1.D2's builder — they're pre-existing uncommitted work from a separate dispatch (likely a parallel W2.D1 builder per workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/PLAN.md:106 `initJSONPayload` action). No file collision with W1.D2's declared paths.
- **Finding:** Workspace state is mixed-build. Not a W1.D2 introduction. The pre-existing W2 work in init_cmd.go is correctly outside W1.D2's `paths` declaration. No compile lock issue.
- **Verdict:** REFUTED on attribution. **NIT-8** raised for orchestrator: the workspace had unrelated uncommitted W2 work when W1.D2 was dispatched. Future dispatch should checkpoint workspace cleanliness.

### H12 — YAGNI

- **Hypothesis:** Builder added abstractions beyond what acceptance criteria required.
- **Test:** Inspected new symbols: `bakeProjectKindCatalogWithHome`, `loadProjectTemplatesForGroups`, `mergeTemplates`, `writeHomeGroupTemplateFixture`. All four are explicitly named in PLAN.md KindPayload (lines 266-272) or acceptance criteria. No surplus generality.
- **Verdict:** REFUTED.

### H13 — Race / goroutine leak / interface misuse / hidden init state

- **Hypothesis:** Standard Go falsification axes.
- **Test:** Functions are pure (no goroutines, no shared state, no interfaces, no `init()` changes). `-race` runs included in `mage test-pkg`.
- **Verdict:** EXHAUSTED, no counterexample found.

### H14 — Slice-aliasing mutation in `mergeTemplates`

- **Hypothesis (additional attack):** `out := base` copies struct value but shares slice headers; mutation via `out.ChildRules[i] = r` mutates `base.ChildRules` backing array.
- **Test:** Read service.go:553 (`out := base`) and 588-589 (`out.ChildRules[i] = r`). Trace caller: `loadProjectTemplatesForGroups` (service.go:506-528) passes freshly-loaded `tpl` values from `loadProjectTemplateWithHome`. Each `tpl` is freshly decoded from its on-disk file (or a fresh embedded copy via `LoadDefaultTemplateForLanguage`), not shared with any caller after this function returns. Mutation of the first-loaded template's slice happens during the merge loop but no other caller holds a reference.
- **Finding:** Safe in current usage. Future direct callers of `mergeTemplates` could be surprised if they pass a `base` they intend to reuse. Defensive `out.ChildRules = slices.Clone(base.ChildRules)` would harden the seam.
- **Verdict:** REFUTED on current safety. **NIT-9** raised for future hardening.

### H15 — Raw `go` commands / `mage install` / file-gating bypass

- **Hypothesis:** Standard discipline checks.
- **Test:** Inspected all new code paths. No `go test`/`go build`/`go vet`/`mage install` invocations. No edits to declared paths outside `service.go`, `service_test.go`, `project.go`, `project_test.go`, `internal/tui/model_test.go` (forced).
- **Verdict:** EXHAUSTED, no counterexample found.

---

## Unmitigated Counterexamples

None.

The two CONFIRMED findings (H3 Tillsyn-non-zero excludes RequiresPlugins; H7 hermeticity regression) are both spec-conforming or inherited — neither is a W1.D2 builder-introduced defect. Both routed to NITs below.

---

## NITs

### NIT-1 — `mergeTemplates` test coverage gap on 7 non-AgentBindings fields (low → medium)

- **Severity:** medium. Spec acceptance #4 enumerates per-field semantics for all 9 fields. Spec acceptance #8 only mandates 4 test cases covering AgentBindings + AC3-empty-guard. Kinds, ChildRules (dedup), Agents, Gates, GateRulesRaw, Tillsyn-replacement, StewardSeeds-append all have ZERO multi-group test coverage. A future builder could regress any of these without `mage ci` catching it.
- **Recommendation:** Add a `TestMergeTemplates_PerFieldSemantics` (or extend `TestBakeProjectKindCatalog_MultiGroup` with sub-cases) covering:
  - Kinds key collision → last-wins.
  - ChildRules tuple dedup → overlay wins on tuple match; non-matching preserved from both.
  - Agents key collision → last-wins.
  - Gates key collision → overlay slice replaces base slice (NOT concat).
  - GateRulesRaw key collision → last-wins.
  - Tillsyn overlay non-zero → replaces.
  - StewardSeeds → append base + overlay, no dedup, order preserved.

### NIT-2 — `Tillsyn` non-zero check omits `RequiresPlugins` (medium, spec-side)

- **Severity:** medium. The non-zero check `MaxContextBundleChars != 0 || MaxAggregatorDuration != 0 || SpawnTempRoot != ""` omits the 4th field `RequiresPlugins`. Concrete counterexample: an overlay `Tillsyn{RequiresPlugins: ["plugin-a"]}` with the other 3 fields zero will silently NOT replace base — the new plugins list is dropped. Builder strictly followed spec acceptance #4 line 405-406, which literally enumerates only 3 fields.
- **Recommendation:** Refinement `MERGE-FIELD-AXIS-R1` is already raised by the spec to revisit Tillsyn merge semantics. The fix is either (a) extend the non-zero check to include `len(overlay.Tillsyn.RequiresPlugins) > 0`, OR (b) shift to per-field merge for Tillsyn (each Tillsyn sub-field last-wins independently), OR (c) different merge policy entirely (e.g. RequiresPlugins always appended). Decision belongs to the planner, not the W1.D2 builder.

### NIT-3 — Empty-string-first ordering not tested (low)

- **Severity:** low. The skip-empty test uses `Groups: ["go", "", "  "]` — non-empty FIRST. A test with `Groups: ["", "go"]` would more cleanly prove the loop continues past empty entries rather than short-circuiting.
- **Recommendation:** Add a sub-case `Groups: ["", "go"]` to `TestBakeProjectKindCatalog_MultiGroup`/`empty_string_in_groups_skipped_without_error` asserting the "go" group's binding still appears in the catalog.

### NIT-4 — `Groups=[""]`-only-empty-strings fallthrough not tested (low)

- **Severity:** low. When ALL Groups entries are empty/whitespace, `loadProjectTemplatesForGroups` falls through to `templates.LoadDefaultTemplateForLanguage(project.Language)`. This branch is reachable in production (a JSON payload that decodes to `Groups: [""]`) but has no test. Also: this path ignores `RepoBareRoot` and `RepoPrimaryWorktree` candidates entirely, which is a subtle behavior change vs `Groups=nil`.
- **Recommendation:** Add a sub-case `Groups: [""]` (or `["", "  "]`) to `TestBakeProjectKindCatalog_MultiGroup` asserting the catalog is populated from the embedded default. Optionally document the bare-root/primary-worktree-bypass behavior in `loadProjectTemplatesForGroups`'s doc-comment or revise the fallthrough to call `loadProjectTemplate(project)` instead of going straight to embedded.

### NIT-5 — `bakeProjectKindCatalog` direct-caller tests inherit W1.D1 hermeticity regression (low, dormant)

- **Severity:** low (dormant). `TestBakeProjectKindCatalog_EmbeddedFallbackPopulatesCatalog` (line 7160) and `TestBakeProjectKindCatalog_NonEmptyPathFallsThroughToEmbedded` (line 7188) call the wrapper. The wrapper now invokes `os.UserHomeDir()` (line 427). If `~/.tillsyn/templates/go.toml` exists, `loadProjectTemplate` (in the single-group branch) would consume it before the embedded fallback. The tests' `SchemaVersion==V1` and `len(Kinds)>0` assertions would pass coincidentally on any valid template. The tests' semantic intent ("EMBEDDED fallback") would silently break.
- **Repro (canonical):**
  ```
  mkdir -p ~/.tillsyn/templates
  cp internal/templates/builtin/till-fe.toml ~/.tillsyn/templates/go.toml
  mage test-func ./internal/app TestBakeProjectKindCatalog_EmbeddedFallbackPopulatesCatalog
  ```
  Test still passes (any valid v1 template satisfies the assertions) but no longer tests "embedded fallback was reached."
- **Recommendation:** Migrate both tests to call `bakeProjectKindCatalogWithHome(&project, "")` (forcing HOME tier skipped via empty homeDir). One-line change per test; the seam exists. Same pattern as W1.D1 BUILD_QA_FALSIFICATION_D1.md Counterexample §1's recommended fix for `loadProjectTemplate` direct-callers. May be batched with W1.D1's NIT or routed to a single hermeticity-cleanup follow-up build.

### NIT-6 — PLAN.md KindPayload-vs-spec-body contradiction on `Groups=[]string{}` (low, planner-side)

- **Severity:** low (planner-side documentation). PLAN.md:270 KindPayload bullet says `"Groups=[]string{}→present (empty non-nil NOT omitted)"`. PLAN.md:386 spec acceptance #2 correctly says `"Groups=[]string{}→omitted"`. Per actual Go `encoding/json` semantics (verified by round-trip test), `[]string{}` IS omitted by `omitempty`. KindPayload is wrong.
- **Recommendation:** Fix PLAN.md:270 to read `"Groups=[]string{}→omitted (per Go encoding/json semantics, len(s)==0 slices are empty)"`. Methodology note: when a PLAN.md states the same fact in both KindPayload `shape_hint` AND acceptance body, planners must keep them in sync; QA falsification should catch divergence (this round's catch).

### NIT-7 — `internal/tui` should be in `Packages` when modifying `ProjectMetadata` (low, methodology)

- **Severity:** low (methodology). W1.D2 builder was forced to touch `internal/tui/model_test.go` to keep `TestProjectSchemaCoverageIsExplicit` green after adding `Groups` to `ProjectMetadata`. This file is outside the declared `Packages: internal/domain, internal/app`. While the change is small (7 lines) and unambiguously correct, the planner should anticipate this when modifying any field on a struct covered by a schema-coverage assertion.
- **Recommendation:** Update planner methodology: when a build modifies a struct that has a schema-coverage test (e.g. `ProjectMetadata`, `Project`, `ActionItem`), include the package containing the coverage test (`internal/tui` today) in the build's `Packages`. Routes via `PLAN-QA-DISCIPLINE-R*` refinement family.

### NIT-8 — Workspace was non-clean at W1.D2 dispatch (low, orchestrator-side)

- **Severity:** low (orchestrator hygiene). At spawn time, the workspace had uncommitted W2-territory changes in `cmd/till/init_cmd.go` and `cmd/till/init_cmd_test.go` (Group→Groups schema rename + MCP*bool). These are NOT W1.D2's introduction; they were already-uncommitted from a parallel W2.D1 builder dispatch. Mixed-build workspaces are confusing for QA — initial diff inspection showed 7 files when only 5 were W1.D2's scope.
- **Recommendation:** Orchestrator should commit or stash unrelated uncommitted work before dispatching the next parallel wave's builder. Pre-dispatch checkpoint: `git status` clean for everything outside the new builder's `paths`. If the parallel builder must continue post-pause (e.g. mid-QA-cycle), route via per-builder topic branches or stashes.

### NIT-9 — `mergeTemplates` slice aliasing in `ChildRules` mutation path (low, future-proofing)

- **Severity:** low. `out := base` copies struct but shares slice headers. `out.ChildRules[i] = r` (line 588) mutates the shared backing array. Safe in current usage (caller `loadProjectTemplatesForGroups` passes freshly-loaded `tpl` values that aren't shared with any other caller after merge). But a future direct caller of `mergeTemplates` that retains the `base` value would observe silent mutation.
- **Recommendation:** Defensive `out.ChildRules = slices.Clone(base.ChildRules)` at line 575 (before the dedup loop). Same hardening for `Kinds`, `AgentBindings`, `Agents`, `Gates`, `GateRulesRaw` maps if the caller invariant is ever relaxed — current pattern of `if out.X == nil { out.X = make(...) }` doesn't protect against base mutation when the map is non-empty.

---

## Verdict rationale

W1.D2's implementation satisfies all 8 acceptance criteria as written:

1. `domain.ProjectMetadata.Groups []string` shipped with correct `omitempty` semantics — round-trip test 4/4 PASS.
2. `Groups=nil` and `Groups=[]string{}` both round-trip to `nil` (per Go encoding/json) — test asserts.
3. `loadProjectTemplatesForGroups` per-group iteration with empty-string skip — present + tested.
4. `mergeTemplates` 9-field merge — present, per-field semantics match spec literally.
5. `bakeProjectKindCatalog` branches on `len(Groups)>0` — present.
6. Typed `ProjectMetadata.Groups` consumed by downstream W2.D7/W3.D1 — typed field shipped; downstream consumption is W2/W3's responsibility (deferred per round-2 PLAN.md absorption).
7. `mage test-pkg ./internal/domain` — 307/307 PASS.
8. `mage test-pkg ./internal/app` — 488/488 PASS, 4 multi-group sub-cases included.

No CONFIRMED build-side counterexamples. The two findings that surfaced (Tillsyn non-zero gap + bakeProjectKindCatalog wrapper hermeticity) are spec-conforming and inherited respectively — neither is a W1.D2 introduction. 9 NITs routed for follow-up: 1 medium (test coverage on 7 unmerged fields), 1 medium (planner-side Tillsyn spec gap), 7 low.

**Verdict: PASS WITH NITS.**
