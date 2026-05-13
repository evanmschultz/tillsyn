# W2.D1 — BUILD-QA-FALSIFICATION Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-falsification-agent (opus)
**Overall verdict:** PASS WITH NITS

## Attack Hypotheses Tested

### H1. *bool JSON semantics — `omitempty` for nil pointer; explicit false unmarshal

- **Hypothesis:** Does `omitempty` on `*bool` actually omit nil-pointer on marshal? Does explicit `false` correctly unmarshal to `*false` (not nil)?
- **Test:** Read `initJSONPayload` struct + `MCPRegistration()` accessor at `cmd/till/init_cmd.go:35-52`. Verified `TestMCPRegistration` table at `cmd/till/init_cmd_test.go:1047-1065` covers `nil_defaults_true`, `explicit_true`, `explicit_false` cases — all green. Stdlib `encoding/json` documented semantics: `omitempty` omits when value is the zero value for its type; for pointers the zero is nil. Non-nil pointer (including pointer to `false`) always emits. Verified via 300/300 `mage test-pkg ./cmd/till` PASS.
- **Finding:** Behavior matches Go stdlib JSON semantics. `MCP: nil` omits the key; `MCP: &false` emits `"mcp":false`; `MCP: &true` emits `"mcp":true`. Round-trip is correct.
- **Verdict:** REFUTED — no counterexample.

### H2. `MCPRegistration()` accessor invariants

- **Hypothesis:** nil→true; `*false`→false; `*true`→true. Test edge: `*bool` pointing to freshly-zeroed bool.
- **Test:** `TestMCPRegistration` cases exercise all three states. A freshly-zeroed `bool` value is `false`; `*bool` pointing to it returns `*p.MCP = false`, accessor returns `false` — same as `explicit_false` case. Confirmed via test pass.
- **Finding:** Accessor is correct. The freshly-zeroed-bool case is identical to explicit-false.
- **Verdict:** REFUTED.

### H3. `Groups` length zero — nil vs explicit []

- **Hypothesis:** Does `validateInitPayload` reject both `Groups: nil` and `Groups: []string{}` (explicit empty slice)?
- **Test:** Validator at `init_cmd.go:702` checks `len(p.Groups) == 0`. Go semantics guarantee `len(nil) == 0` and `len([]string{}) == 0` — same code path. `TestValidateInitPayload_W2D1` `empty_groups` case uses `Groups: nil`. `TestInit_JSONParse_TableDriven` `missing_groups` case omits the `groups` key entirely (also nil).
- **Finding:** Both nil and empty-slice paths hit the same `len == 0` branch and produce "groups required" error. Functionally correct. **NIT (low)**: explicit `{"groups":[]}` JSON-shape test absent; semantic equivalence to `missing_groups` is implicit, not asserted.
- **Verdict:** REFUTED (functional). NIT raised.

### H4. `Groups` entry validation — case / whitespace / duplicates

- **Hypothesis:** What about `Groups: ["GO"]` (uppercase)? `Groups: ["go "]` (whitespace)? `Groups: ["go","go"]` (duplicates)?
- **Test:** Validator at `init_cmd.go:706-717` does strict `g == allowed` comparison — no `TrimSpace`, no case-fold.
  - `"GO"` → not in `["gen","go","fe"]` → "invalid group(s) [GO]" error. Correct rejection.
  - `"go "` (trailing space) → not equal to `"go"` → "invalid group(s) [go ]" error. Correct rejection.
  - `"go","go"` → both pass `found=true` → SILENTLY ACCEPTED (no de-dup, no rejection). D1's stub `copyAgentFiles(destDir, payload.Groups[0])` only reads index 0 so duplicates are inert in D1. D5 will iterate over the slice — if D5 doesn't de-dup, `["go","go"]` could attempt two writes of the same files (which the idempotent skip would absorb cleanly, but with confusing `added/skipped` counts).
- **Finding:** Strict matching for case/whitespace is acceptable (clear rejection with remediation). Duplicate handling is SILENT ACCEPT — spec-consistent for D1 but a latent D5 concern. **NIT (medium)**: spec doesn't pin duplicate semantics; D5 builder should explicitly handle (de-dup or document acceptance).
- **Verdict:** REFUTED for D1 scope. NIT raised for D5 forward-compat.

### H5. `reservedInitGroups` deletion — every consumer updated?

- **Hypothesis:** `git grep "reservedInitGroups"` should return zero hits in production code; tests that asserted reservation logic should be deleted or repurposed.
- **Test:** `rg "reservedInitGroups" cmd/till/` → 1 hit at `init_cmd_test.go:95` in a doc-comment explaining the deletion. No production-code references. Repo-wide sweep excluding workflow MDs returns the same single doc-comment hit.
- **Finding:** Deletion is clean. The single remaining mention is intentional historical commentary in a test doc-comment.
- **Verdict:** REFUTED.

### H6. `till-gdd` reference sweep in cmd/till/

- **Hypothesis:** `git grep "till-gdd" cmd/till/` → zero hits expected per builder claim of "all doc-comments cleaned."
- **Test:** `git grep "till-gdd" cmd/till/` returns 8 hits across `init_cmd.go` (3 doc-comment historical mentions) and `init_cmd_test.go` (5 hits: 3 doc-comments referencing the deletion, 2 test cases asserting `till-gdd` correctly rejects as invalid post-W2.D1). Repo-wide hits in `internal/app/dispatcher/cli_claude/render/` and `internal/templates/embed.go` are NOT W2.D1 scope (paths outside `cmd/till/`); the brief constrained W2.D1 to `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go`.
- **Finding:** All cmd/till `till-gdd` mentions are either (a) historical doc-comments explaining the deletion or (b) test-case payloads asserting `till-gdd` now invalid. Builder's claim "all doc-comments cleaned" is technically slightly imprecise — historical commentary remains, which is acceptable.
- **Verdict:** REFUTED (the historical references are deliberate). **NIT (low)**: builder's claim phrasing slightly overstates the cleanup; spec says "All doc-comments referencing ... `till-gdd`, `reservedInitGroups` are updated or removed" — historical commentary is "updated" not "removed."

### H7. Bridge code marked for D2-D5 builders

- **Hypothesis:** Are the bridge code locations clearly marked (with comments) for D2-D5 builders to find?
- **Test:** Read bridge sites:
  - `init_cmd.go:247-253` (TUI bridge) — explicit comments: `// D1: single-value Groups slice as a bridge until D3 ships multi-select via picker_multi.go.` and `// D1->D3 interim: TUI hardwires MCP to false. D4 removes this hardwire and adds the confirm step that sets MCP via user input.`
  - `init_cmd.go:427-429` (pipeline `copyAgentFiles` single-group call) — explicit: `// D1: pass single group stub until D5 upgrades to the full multi-group loop (copyAgentFiles signature takes []string in D5).`
  - `init_cmd.go:471` (Laslig `"group"` key) — NO bridge comment marking this as a D5 update site.
- **Finding:** TUI bridge and pipeline-call bridge are well-documented. Laslig summary key at line 471 (`{"group", payload.Groups[0]}`) lacks a bridge comment despite D5's spec changing this to `"groups"` (comma-joined). **NIT (low)**: missing in-line marker for D5 builder.
- **Verdict:** PASS with NIT.

### H8. D5 forward-compat — Laslig key still "group"

- **Hypothesis:** `TestInit_SuccessMessage_Format` asserts `"group"`. D5 will change to `"groups"`. Bridge or bug?
- **Test:** PLAN.md D5 acceptance explicitly states: "Laslig summary row updated: `groups` key (comma-joined list) replaces `group` key." Bridge is intentional per spec.
- **Finding:** Intentional bridge. Test will be updated by D5. **Side-effect NIT (low)**: when a TUI/JSON user supplies multi-group (e.g. `["go","fe"]`), the Laslig summary shows only the first group — small UX info-loss until D5 ships. Acceptable for an intermediate droplet but the user may be momentarily confused if they hit this state via JSON-mode dogfooding before D5 lands.
- **Verdict:** REFUTED (intentional). NIT raised.

### H9. JSON tag round-trip — old "group" key drift

- **Hypothesis:** `{"groups":["go"]}` unmarshals correctly; old `{"group":"go"}` — does it fail with clear error or silently drop?
- **Test:** `runInitJSON` uses `json.Unmarshal` (NOT `json.NewDecoder().DisallowUnknownFields()`). Confirmed via `rg "DisallowUnknownFields"` → 0 hits. Unknown JSON keys silently drop.
  - Old `{"group":"go"}` unmarshals to `initJSONPayload{Name:"", Groups: nil}` — `validateInitPayload` catches `len(Groups) == 0` and returns "groups required (must supply at least one group)" error.
- **Finding:** Old-payload-shape user sees "groups required" error, NOT "field renamed from group to groups" error. Error message is correct-but-not-helpful for migration scenarios. **NIT (medium)**: a small migration-friendliness improvement would be to detect a stray top-level `"group"` field and surface "groups field renamed from singular group; supply groups: [...]" — but per project memory `feedback_no_migration_logic_pre_mvp.md`, "no migration code" is the policy. Acceptable per project policy.
- **Verdict:** REFUTED (policy-consistent). NIT raised under "could be friendlier post-MVP".

### H10. CONSUMER-TIE coverage gaps

- **Hypothesis:** 3 sub-cases for valid single-group no-mcp / valid multi-group mcp-false / invalid group. Gaps: valid multi-group mcp-true? mcp-omitted-defaults-true? Empty Groups array?
- **Test:** Read `TestInit_ConsumerTie_W2D1` at `init_cmd_test.go:1156-1213`. Sub-cases:
  - (a) `valid_single_group_no_mcp_key`: `{"name":"ct-a","groups":["go"]}` — covers nil→true MCP default.
  - (b) `valid_multi_group_mcp_false`: `{"name":"ct-b","groups":["go","fe"],"mcp":false}`.
  - (c) `invalid_group_bogus`: `{"name":"ct-c","groups":["bogus"]}` → error.
  - Missing: **valid multi-group mcp:true** scenario. Single-group mcp:true coverage exists in `TestInit_MCPJSON_FreshFile` and `TestInit_MCPJSON_PreservesHTTPTransport`. Multi-group + mcp:true combination is not exercised end-to-end.
  - Missing: **explicit `{"groups":[]}`** (empty-slice JSON shape, distinct from `missing_groups` which omits the key entirely). Same code path semantically; cosmetic test gap.
- **Finding:** Functional coverage is adequate — the missing combinations exercise no new code paths (multi-group + mcp:true goes through same `MCPRegistration() == true` branch as single-group + mcp:true; the `copyAgentFiles(destDir, payload.Groups[0])` stub treats multi-group identically to single-group). **NIT (low)**: spec-listed CONSUMER-TIE scenarios are met; broader coverage matrix would strengthen the test bed for D5/D4 inheritance.
- **Verdict:** REFUTED (per-spec coverage met). NIT raised.

### H11. `copyAgentFiles(destDir, payload.Groups[0])` — Groups[0] safety assumption

- **Hypothesis:** D5 expands this to iterate Groups. Today's single-index bridge ASSUMES `Groups` has at least one element. The validation gate `len >= 1` makes this safe — but is the assumption documented?
- **Test:** Two entry points to `runInitPipeline`:
  - `runInitJSON` → calls `validateInitPayload` → guarantees `len(Groups) >= 1` before `runInitPipeline`.
  - `runInitTUI` → TUI only reaches `Done()` state when `Enter` pressed on enabled row → sets `Groups = []string{row.Name}` (always len 1). The Done/Cancelled check at lines 372-376 prevents incomplete-walk payloads from reaching the pipeline. `Payload()` doc-comment at lines 312-315 notes "reading Payload() on a cancelled or in-progress walk returns the zero value (and the Groups field will be nil/empty)."
- **Finding:** Safety invariant holds via two independent gates (JSON validation OR TUI Done-gate), but `runInitPipeline`'s `payload.Groups[0]` access at line 429 has no in-line comment explaining why the access is safe. The bridge comment at lines 427-428 explains the D5 upgrade plan but not the safety invariant. **NIT (low)**: implicit invariant; an explicit comment like `// Safe: validateInitPayload (JSON path) or TUI Done() gate guarantees len(Groups) >= 1` would help.
- **Verdict:** REFUTED (invariant holds). NIT raised.

### H12. YAGNI — anything beyond L2 D1 spec?

- **Hypothesis:** Did builder add scope beyond what D1 required?
- **Test:** Walk the diff against the spec's KindPayload at PLAN.md line 106:
  - `initJSONPayload` Group→Groups + MCP→*bool + MCPRegistration() ✓
  - `allowedInitGroups` updated ✓
  - `reservedInitGroups` deleted (var + doc-comment + branch) ✓
  - `initTUIGroupRows` updated to 3 rows ✓
  - `validateInitPayload` updated ✓
  - `Disabled bool` retained per spec ✓
  - Tests: `TestMCPRegistration`, `TestValidateInitPayload_W2D1`, `TestInit_ConsumerTie_W2D1` (3 sub-cases) ✓
  - Existing tests updated to reflect new schema (no removal of tests outside scope; replacements documented in comments) ✓
- **Finding:** No YAGNI. Changes match KindPayload line-for-line.
- **Verdict:** REFUTED.

### H13. Cross-droplet bleed

- **Hypothesis:** W1.D2, W1.D1, W4.D2 also uncommitted. Did W2.D1 touch anything outside its declared `paths`?
- **Test:** `git status --short` → 7 files modified: `cmd/till/init_cmd.go`, `cmd/till/init_cmd_test.go` (W2.D1 declared paths) + `internal/app/service.go`, `internal/app/service_test.go`, `internal/domain/project.go`, `internal/domain/project_test.go`, `internal/tui/model_test.go`. Spot-checked `internal/app/service.go` diff: changes are `bakeProjectKindCatalog` + `loadProjectTemplatesForGroups` (multi-group HOME-tier template loading) — explicitly W1.D2 scope per `// Drop 4c.6.1 W1.D2:` comment at line 414. Brief confirmed "uncommitted W1.D2 + W2.D1 changes."
- **Finding:** No W2.D1 bleed into other paths. The non-cmd/till changes are W1.D2 in-flight work. `.tillsyn/agents/fe/` + `.tillsyn/agents/go/` untracked are gitignored test-run artifacts from manual `till init` dogfooding, not source bleed.
- **Verdict:** REFUTED.

### H14. Hermeticity — new `os.UserHomeDir()` / `$HOME` reads?

- **Hypothesis:** Did W2.D1 introduce HOME reads that tests don't fake?
- **Test:** `rg "UserHomeDir|os.Getenv\(.HOME" cmd/till/init_cmd.go` → 1 hit at line 786 in `registerMCPJSON` (pre-existing, not added by W2.D1; this code was shipped in earlier W2 work). All tests that exercise `mcp:true` use `t.Setenv("HOME", tmp)` for isolation (`TestInit_MCPJSON_FreshFile`, `TestInit_MCPJSON_AppendsToExisting`, etc.). W2.D1 introduces no new HOME reads. `TestInit_ConsumerTie_W2D1` sub-cases use `t.Setenv("HOME", tmp)` for hermeticity in (a) which has no mcp key (default true triggers `registerMCPJSON`).
- **Finding:** Hermetic. No new HOME reads.
- **Verdict:** REFUTED.

## Unmitigated Counterexamples

None found.

## NITs

| # | Severity | Topic | Recommended action |
|---|----------|-------|---------------------|
| 1 | low | Missing explicit `{"groups":[]}` JSON-shape test in `TestInit_JSONParse_TableDriven` / `TestValidateInitPayload_W2D1`. The behavior is identical to `groups: nil` (same `len == 0` branch), but the JSON-shape distinction is not asserted. | Add one table-case `empty_groups_array_explicit` with payload `{"name":"x","groups":[]}` expecting the same "groups required" error. Builder may absorb in a follow-up commit or D5 builder may add when expanding tests. |
| 2 | medium | Duplicate-group silent accept in `validateInitPayload`: `Groups: ["go","go"]` passes. D1 stub ignores via `Groups[0]` so this is inert in D1, but D5 will iterate and could double-process. | D5 builder should explicitly handle: either de-duplicate before `copyAgentFiles` loop OR document that duplicates pass through (idempotent skip absorbs second write but `added/skipped` counts become confusing). Surface as a D5 spec clarification. |
| 3 | low | `till-gdd` historical doc-comments remain in `init_cmd.go:34`, `init_cmd.go:58`, `init_cmd.go:142`, `init_cmd_test.go:94/96/261`. Builder's claim "all doc-comments cleaned" is slightly imprecise — spec said "updated or removed." Historical commentary is "updated" (explains the deletion), which is acceptable but the wording in the builder claim could mislead a reader expecting zero mentions. | No code change required. Builder-claim phrasing nit only. |
| 4 | low | Laslig summary key at `init_cmd.go:471` (`{"group", payload.Groups[0]}`) lacks a bridge comment marking it as a D5 update site (where D3-pipeline-call bridge and TUI bridge both have explicit `// D1...` markers). | Optional: D5 builder will rewrite this line. A one-line `// D1->D5 bridge: single-group display; D5 changes to "groups" comma-joined` would aid discoverability. |
| 5 | low | Multi-group + mcp:true CONSUMER-TIE combination not exercised end-to-end. Other tests cover (single-group + mcp:true) and (multi-group + mcp:false); the missing intersection is functionally redundant with existing paths but a gap in the coverage matrix. | Optional: add a fourth sub-case to `TestInit_ConsumerTie_W2D1` for `{"groups":["go","fe"],"mcp":true}`. Low priority — no new code paths exercised. |
| 6 | low | Implicit safety invariant for `payload.Groups[0]` at `init_cmd.go:429` (`runInitPipeline`). The invariant holds via two gates (JSON-mode `validateInitPayload` and TUI-mode `Done()` check) but lacks an in-line comment. | Optional: add `// Safe: validateInitPayload (JSON path) or TUI Done-gate guarantees len(Groups) >= 1.` just before line 429. |
| 7 | low (post-MVP) | Old-payload-shape (`{"group":"go"}`) produces "groups required" error instead of a more helpful "field renamed" message. Strict per project's `feedback_no_migration_logic_pre_mvp.md` policy. | Defer to post-MVP. Acceptable today. |

## Verdict rationale

W2.D1 satisfies all spec acceptance criteria. `mage ci` green (3194/3194). `mage test-pkg ./cmd/till` green (300/300). `mage test-func ./cmd/till TestMCPRegistration` (4/4), `TestValidateInitPayload_W2D1` (10/10), `TestInit_ConsumerTie_W2D1` (4/4) — all green. Bridge code is clearly marked at the two highest-leverage sites (TUI + pipeline-call). Hermeticity preserved via existing `t.Setenv("HOME", ...)` patterns. Cross-droplet bleed: none — the other modified files are explicit W1.D2 scope per the brief and confirmed via diff inspection. `till-gdd` and `reservedInitGroups` sweep: clean in production code, historical commentary in doc-comments is intentional.

Seven NITs identified — six are minor / cosmetic / forward-compat, none block dispatch. NIT 2 (duplicate-group silent accept) is the highest-value finding because it propagates to D5 — surface as a D5 spec clarification before that builder dispatches.

No unmitigated counterexamples. Verdict: **PASS WITH NITS**.
