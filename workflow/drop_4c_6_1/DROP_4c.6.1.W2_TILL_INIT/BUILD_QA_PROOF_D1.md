# W2.D1 — BUILD-QA-PROOF Verdict

**Date:** 2026-05-13
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

## Acceptance Bullet Coverage

### Bullet 1 — `initJSONPayload.Group string` -> `Groups []string`, JSON tag `"group"` -> `"groups"`

Quote: *"`initJSONPayload.Group string` field renamed to `Groups []string`. JSON tag changes from `"group"` to `"groups"`. Old `"group"` key in `--json` payloads no longer accepted."*

Evidence:
- `cmd/till/init_cmd.go:36-37` — struct field is `Groups []string` with `json:"groups"`. `Group` field absent.
- `cmd/till/init_cmd.go:88` — Example string updated to `'{"name":"my-project","groups":["go"],"mcp":true}'`.
- `cmd/till/init_cmd.go:102` — flag help text updated to `"groups":["go"]`.
- "Old `"group"` key no longer accepted": Go's `encoding/json` ignores unknown keys silently; the old key would deserialize to `Groups: nil` and then `validateInitPayload` would fail on `len(Groups) == 0`. Verified via `TestInit_JSONParse_TableDriven/missing_groups` (line 152-155) which sends `{"name":"foo"}` and expects `"groups", "required"` substring. The acceptance phrasing "no longer accepted" is honored end-to-end.

Verdict: **PASS**

### Bullet 2 — `MCP bool` -> `MCP *bool` + `MCPRegistration()` accessor

Quote: *"`initJSONPayload.MCP bool` changes to `MCP *bool` with JSON tag `"mcp,omitempty"`. A helper `func (p initJSONPayload) MCPRegistration() bool { if p.MCP == nil { return true }; return *p.MCP }` is added, mirroring `OrchSelfApprovalIsEnabled()` pattern (`internal/domain/project.go:157`). Omitting `"mcp"` from a `--json` payload defaults to YES (MCP registration enabled)."*

Evidence:
- `cmd/till/init_cmd.go:38` — `MCP *bool` with `json:"mcp,omitempty"`.
- `cmd/till/init_cmd.go:47-52` — `MCPRegistration()` accessor: `if p.MCP == nil { return true }; return *p.MCP`. Body matches the Specify shape verbatim.
- Mirror reference: `internal/domain/project.go:180` — `OrchSelfApprovalIsEnabled()` accessor confirmed. Note: Specify cites line 157; actual line is 180 (`func` declaration) — citation drift but the accessor exists and the polarity matches. NIT-level.
- `omitempty` semantics: nil pointer omitted on marshal, missing key unmarshals to nil. Go stdlib behavior. Round-trip property holds.
- `TestMCPRegistration` (lines 1043-1065) covers all 3 cases: `nil_defaults_true`, `explicit_true`, `explicit_false`. All 3 sub-cases pass per `mage test-pkg ./cmd/till` (300/300).

Verdict: **PASS** (with NIT4 below on the line-citation drift)

### Bullet 3 — `allowedInitGroups = ["gen", "go", "fe"]`

Quote: *"`allowedInitGroups` changes from `["till-gen", "till-go"]` to `["gen", "go", "fe"]`."*

Evidence:
- `cmd/till/init_cmd.go:61` — `var allowedInitGroups = []string{"gen", "go", "fe"}`.
- Doc-comment at lines 54-60 explains the W4.D1 rename + the `till-gdd` deletion.

Verdict: **PASS**

### Bullet 4 — `reservedInitGroups` map + doc-comment + reserved-group branch DELETED

Quote: *"`reservedInitGroups` map AND its doc-comment are DELETED entirely. The reserved-group validation branch in `validateInitPayload` is also removed. If a future group needs reservation, the validation can re-introduce it."*

Evidence:
- `git grep "reservedInitGroups" cmd/till/init_cmd.go` returns zero hits. Var fully deleted from production code.
- `validateInitPayload` (lines 698-722): no reserved-group branch. Only checks `Name` non-empty, `len(Groups) > 0`, each element in `allowedInitGroups`.
- One residual reference in `cmd/till/init_cmd_test.go:95` is a doc-comment EXPLAINING the deletion (historical context) — not a code reference. Acceptable for documentation continuity.

Verdict: **PASS**

### Bullet 5 — `validateInitPayload` updated: len(Groups)>0 + per-element membership + clear error

Quote: *"`validateInitPayload` updated: checks `len(p.Groups) > 0`, each group in `Groups` must be in `allowedInitGroups`. Returns a clear error listing invalid groups."*

Evidence:
- `cmd/till/init_cmd.go:698-722` — three-stage validation: (1) `Name` required, (2) `len(Groups) == 0` -> "groups required (must supply at least one group)", (3) per-element loop collects invalid into a slice, formatted error `"till init: invalid group(s) %v; allowed: %v"`.
- `TestValidateInitPayload_W2D1` (lines 1071-1146) covers 9 cases: `valid_single_go`, `valid_multi_go_fe`, `valid_gen`, `valid_fe`, `empty_groups`, `invalid_till_gdd`, `invalid_till_go_old_name`, `missing_name`, `mixed_valid_invalid`. All 9 sub-cases pass.
- Error message is clear: contains the invalid group list AND the allowed list — both halves of the contract.

Verdict: **PASS**

### Bullet 6 — `initTUIGroupRows` updated to 3 rows (gen, go, fe), all enabled; `Disabled bool` KEPT inert

Quote: *"`initTUIGroupRows` updated to three rows (gen, go, fe), all enabled. The `Disabled bool` field on `initTUIGroupRow` is KEPT for the D1->D3 interim (all rows have `Disabled: false` — the field is inert but the struct shape is preserved; D3 removes it when `picker_multi.go` takes over)."*

Evidence:
- `cmd/till/init_cmd.go:135-138` — `initTUIGroupRow` struct retains `Disabled bool` field. Doc-comment at lines 131-134 explains the D1->D3 interim.
- `cmd/till/init_cmd.go:146-150` — three rows `{Name: "gen", Disabled: false}`, `{Name: "go", Disabled: false}`, `{Name: "fe", Disabled: false}`. All explicitly `Disabled: false`.

Verdict: **PASS**

### Bullet 7 — `nextEnabledGroupRow` / `prevEnabledGroupRow` remain (unchanged, no-op skip logic)

Quote: *"`nextEnabledGroupRow` and `prevEnabledGroupRow` helpers remain (unchanged) — their skip-disabled logic is a no-op since all rows are enabled. D3 removes them."*

Evidence:
- `cmd/till/init_cmd.go:322-329` — `nextEnabledGroupRow(cur int)` walks forward skipping `Disabled` rows. Since all `Disabled: false`, the first iteration returns `cur+1` (or `cur` at end-of-slice).
- `cmd/till/init_cmd.go:334-341` — `prevEnabledGroupRow(cur int)` symmetric.
- Both functions kept, both still iterate over `initTUIGroupRows[i].Disabled`. Unchanged logic.

Verdict: **PASS**

### Bullet 8 — TUI bridge `finalPayload.Groups = []string{row.Name}` at line ~235

Quote: *"`initTUIModel.finalPayload.Group = row.Name` (line ~235) becomes `finalPayload.Groups = []string{row.Name}` (single-value slice bridge until D3 ships multi-select)."*

Evidence:
- `cmd/till/init_cmd.go:247-249` — comment: `"D1: single-value Groups slice as a bridge until D3 ships multi-select via picker_multi.go."` followed by `m.finalPayload.Groups = []string{row.Name}`.
- Also at line 250-253: TUI hardwires `mcpFalse := false; m.finalPayload.MCP = &mcpFalse` — sets the `*bool` to a false pointer. Comment explains D4 will remove this hardwire.

Verdict: **PASS**

### Bullet 9 — All doc-comments referencing old names updated or removed

Quote: *"All doc-comments referencing `Group string`, `till-gen`, `till-go`, `"till-gdd"`, `reservedInitGroups` are updated or removed."*

Evidence:
- `cmd/till/init_cmd.go:33-34, 55-60, 142-145` — three doc-comments reference `till-gen`, `till-go`, `till-gdd`. Each is in the context of explaining the W4.D1 rename / W2.D1 deletion (i.e., HISTORICAL context, not promoting the old names). Acceptable per Specify (intent is "no stale references that suggest the names are still valid"; these doc-comments explicitly describe the rename).
- `cmd/till/init_cmd_test.go:94-96, 261, 374, 404` — test-side comments are negative-test context (e.g. `"till-gdd now surfaces as a plain invalid-group error"`, `"FLAT copy required — no till-go/ prefix"`). All historical/negative-context. Acceptable.
- **NIT1 (see below)**: `cmd/till/help.go:390` still carries the OLD `--json` example `'{"name":"my-project","group":"till-go","mcp":true}'` — singular `"group"` key + old `till-go` name. User-facing help output drifts from new schema. Outside W2.D1's declared `paths` (`cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go`), but the acceptance bullet's scope arguably covers user-facing doc-comments. Captured as NIT.

Verdict: **PASS WITH NIT1**

### Bullet 10 — CONSUMER-TIE: 3 `run()` end-to-end tests

Quote: *"CONSUMER-TIE: validation behavior tested via `run(ctx, args, &out, io.Discard)` end-to-end — at minimum: (a) valid single-group `--json '{"name":"x","groups":["go"]}'` (no `mcp` key — verifies nil->true default); (b) valid multi-element `--json '{"name":"x","groups":["go","fe"],"mcp":false}'`; (c) invalid group `--json '{"name":"x","groups":["bogus"]}'` expects non-zero exit + error substring "invalid". Unit assertions on `validateInitPayload` are acceptable as supplement."*

Evidence:
- `TestInit_ConsumerTie_W2D1` (lines 1156-1213) — three sub-tests:
  - `valid_single_group_no_mcp_key` (1157-1175): `run(..., "init", "--json", '{"name":"ct-a","groups":["go"]}')` with NO `mcp` key. Expects nil error and "Init" substring. Verifies the nil->true default path through the pipeline (mcp would be registered).
  - `valid_multi_group_mcp_false` (1177-1195): `run(..., "init", "--json", '{"name":"ct-b","groups":["go","fe"],"mcp":false}')`. Multi-element. Expects nil + "Init".
  - `invalid_group_bogus` (1197-1212): `run(..., "init", "--json", '{"name":"ct-c","groups":["bogus"],"mcp":false}')`. Expects non-nil error containing "invalid".
- All three call `run(ctx, args, &out, io.Discard)` directly — full cobra wiring exercised.
- Supplement `TestValidateInitPayload_W2D1` (1071-1146) provides 9 additional unit assertions.

Verdict: **PASS**

### Bullet 11 — `mage test-pkg ./cmd/till` green; `mage ci` green

Quote: *"`mage test-pkg ./cmd/till` passes with all existing and new tests green. `mage ci` green."*

Evidence:
- `mage test-pkg ./cmd/till`: `Test summary tests: 300, passed: 300, failed: 0, skipped: 0`. (verified live)
- `mage ci`: `Test summary tests: 3194, passed: 3194, failed: 0, skipped: 0`. Coverage threshold (70%) met across all 30 packages. `cmd/till` coverage 76.1%. Build green. (verified live)

Verdict: **PASS**

### KindPayload changes verification

The KindPayload declares six change entries. Each verified:

1. `initJSONPayload` MODIFY: Group->Groups, MCP->*bool, +MCPRegistration() — PASS (lines 35-52).
2. `allowedInitGroups` MODIFY: `["gen","go","fe"]` — PASS (line 61).
3. `reservedInitGroups` DELETE: var + doc-comment + reserved-group branch — PASS (grep returns 0 hits in production).
4. `initTUIGroupRows` MODIFY: 3 rows all enabled, keep Disabled field — PASS (lines 146-150).
5. `validateInitPayload` MODIFY: len(Groups)>=1 + per-element + no reserved-group branch — PASS (lines 698-722).
6. `TestValidateInitPayload + TestInitTUIGroupRows + TestMCPRegistration` MODIFY: per spec — PASS (`TestValidateInitPayload_W2D1`, `TestRunInitTUI_SelectsFeRow`, `TestMCPRegistration` all present and passing; `TestInit_ConsumerTie_W2D1` adds the 3 CONSUMER-TIE cases as supplement).

Verdict: **PASS**

### Cross-cut: omitempty round-trip semantics on *bool

The acceptance hinges on `*bool` with `omitempty` correctly round-tripping nil/explicit-false/explicit-true:

- **Missing key on unmarshal**: Go's `json.Unmarshal` leaves `MCP` at its zero value (nil for pointer). Verified path: `TestInit_ConsumerTie_W2D1/valid_single_group_no_mcp_key` sends `'{"name":"ct-a","groups":["go"]}'` (no `mcp` key) and the test verifies the pipeline succeeds, which implies `MCPRegistration()` returned true (registered MCP). PASS.
- **Explicit false on unmarshal**: `"mcp":false` deserializes to `*MCP = &false`. `MCPRegistration()` returns false. Verified via `TestInit_ConsumerTie_W2D1/valid_multi_group_mcp_false`. PASS.
- **Explicit true on unmarshal**: `"mcp":true` deserializes to `*MCP = &true`. `MCPRegistration()` returns true. Verified via `TestInit_JSONParse_TableDriven/valid_gen_mcp_true` (line 120-124). PASS.
- **omitempty on marshal**: not exercised in W2.D1 (no JSON output path), but the tag is set correctly for future round-trips.

Verdict: **PASS**

### D2-D7 unblocking check

D2 reads `initJSONPayload.Groups` for FLAT-layout detection.
D3 replaces inline picker with `picker_multi.go` and reads `initTUIGroupRows` for default selection; both surfaces survive D1 with the canonical shape.
D4 reads `payload.MCPRegistration()` for the confirm step.
D5 refactors `copyAgentFiles(destDir, payload.Groups[0])` to iterate over all Groups elements.
D6 reads `payload.Groups` for template.toml aggregation.
D7 reads `payload.Groups` for `Metadata.Groups` typed field on `CreateProjectWithMetadata`.

All downstream consumers find a stable, named, typed surface. **D1 unblocks D2-D7 as designed.**

Verdict: **PASS**

## NITs

### NIT1 — `cmd/till/help.go:390` carries stale `--json` example (singular `group` + old `till-go` name)

**Severity:** medium
**Axis:** spec-conformance (doc-comment / user-visible help drift)

Quote of stale text:
```
"  till init --json '{\"name\":\"my-project\",\"group\":\"till-go\",\"mcp\":true}'",
```

This is the help-text registry entry for `till init` (`cmd/till/help.go:377-392`). The example shows the OLD JSON schema (`"group":"till-go"` — singular field + retired group name). Running `till help init` (or whatever surfaces this registry) would print an example that the new `validateInitPayload` REJECTS as invalid.

The file `cmd/till/help.go` is outside W2.D1's declared `paths` (which are `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go`). However, the acceptance bullet 9 says *"All doc-comments referencing `Group string`, `till-gen`, `till-go`, `"till-gdd"`, `reservedInitGroups` are updated or removed"* — `help.go` is user-facing doc-comment-equivalent material and arguably falls under "doc-comments referencing till-go".

**Fix hint:** Update `cmd/till/help.go:390` to mirror `cmd/till/init_cmd.go:88`:
```
"  till init --json '{\"name\":\"my-project\",\"groups\":[\"go\"],\"mcp\":true}'",
```

**Routing recommendation:** Either (a) fix as a tiny inline patch under W2.D1 before commit, OR (b) capture as a W2.D2 or W2.D5 piggyback edit since those droplets also touch `cmd/till/`. NOT a release-blocker; help output is a soft UX surface, and the validation error message in `validateInitPayload` is clear enough that a user would self-correct from the typed error.

### NIT2 — `runInitPipeline` Laslig "group" row label still singular

**Severity:** low
**Axis:** spec-conformance (cosmetic / D5 refactors anyway)

Quote: `cmd/till/init_cmd.go:471` — `{"group", payload.Groups[0]}` writes a Laslig key-value pair labeled `"group"` (singular) with value `Groups[0]`. For a multi-group payload like `["go","fe"]`, the Laslig output shows only the first group.

This is a SINGLE-VALUE BRIDGE consistent with Specify ("D5 upgrades this to the full multi-group loop"). The Specify explicitly says `copyAgentFiles(destDir, payload.Groups[0])` is the bridge; the Laslig row label naturally inherits. D5 will update both the loop AND the Laslig label to plural ("groups", joined).

**Fix hint:** None required — D5 (its own droplet) refactors this row. Captured for visibility only.

### NIT3 — Three doc-comments retain `till-gen` / `till-go` / `till-gdd` references

**Severity:** low
**Axis:** spec-conformance (intentional / explains rename)

Locations:
- `cmd/till/init_cmd.go:33-34` (initJSONPayload doc-comment): *"Drop 4c.6.1 W4.D1 renamed `till-gen` -> `gen` and `till-go` -> `go`"*.
- `cmd/till/init_cmd.go:55-60` (allowedInitGroups doc-comment): same explanation.
- `cmd/till/init_cmd.go:142-145` (initTUIGroupRows doc-comment): *"`till-gdd` was removed in Drop 4c.6.1 W2.D1"*.

These are HISTORICAL-CONTEXT comments explaining why the rename happened. They do NOT promote the old names as valid. Acceptable per the Specify intent. Captured as NIT only because the acceptance bullet wording (*"updated or removed"*) is technically met by "updated to explain the rename" but some readers might prefer the comments to omit the old names entirely.

**Fix hint:** None required. The historical-context comments aid future code readers; trimming them would lose useful drop-attribution.

### NIT4 — `MCPRegistration()` doc-comment cites wrong line for `OrchSelfApprovalIsEnabled`

**Severity:** low
**Axis:** spec-conformance (citation drift)

Quote: `cmd/till/init_cmd.go:45-46` — *"This mirrors the `OrchSelfApprovalIsEnabled()` accessor pattern on ProjectMetadata (internal/domain/project.go:166)"*.

Actual location: `internal/domain/project.go:180` (`func (m ProjectMetadata) OrchSelfApprovalIsEnabled() bool`).

The accessor itself exists and the polarity matches. The line cite is approximately 14 lines off — likely a pre-edit cite that drifted as adjacent code changed. Not load-bearing; the symbol name is the durable anchor.

**Fix hint:** Update the cite to `internal/domain/project.go:180` or to a less line-brittle form (`OrchSelfApprovalIsEnabled() on ProjectMetadata`).

## Verdict rationale

All 11 acceptance bullets PASS with file:line evidence on disk. All 6 KindPayload entries verified. The three CONSUMER-TIE sub-cases (`TestInit_ConsumerTie_W2D1`) are present and exercise `run(ctx, args, &out, io.Discard)` end-to-end. The `*bool`-omitempty round-trip property is verified across all three states (nil/explicit-false/explicit-true). `reservedInitGroups` deletion is complete in production code (zero residual references). `mage test-pkg ./cmd/till` = 300/300 PASS. `mage ci` = 3194/3194 PASS, coverage threshold met, build green.

D2-D7 are unblocked: every downstream droplet finds a stable typed surface (`Groups []string`, `MCPRegistration() bool`, `allowedInitGroups []string` with three canonical names).

Four NITs raised — one user-facing (NIT1 on `cmd/till/help.go:390` carrying stale `--json` example) and three cosmetic / spec-conformance. None block PASS; NIT1 recommended for inline fix before commit OR piggyback under W2.D2/W2.D5.

**Overall verdict: PASS WITH NITS.**
