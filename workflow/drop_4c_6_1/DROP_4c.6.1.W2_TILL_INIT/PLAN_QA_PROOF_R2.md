# PLAN_QA_PROOF — DROP_4c.6.1.W2_TILL_INIT (Round 2)

**Reviewer:** go-qa-proof-agent
**Round:** 2
**Verdict:** **PASS** — all round-1 findings (1 FF + 7 NITs from proof; 3 FFs + 7 NITs from falsification) absorbed; all R10 cross-cutting decisions landed. Zero new findings. Plan dispatchable today for D1; D3/D4 await W5 ship; D5 awaits W4.D1 ship; D7 verifies typed `ProjectMetadata.Groups` from W1.D2 via LSP at dispatch. No FFs. No NITs.

---

## 0. Verification Inventory

Files read end-to-end:

- `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/PLAN.md` (round-2, 443 lines)
- `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/_BLOCKERS.toml` (40 lines, 7 entries D1-D7)
- `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/PLAN_QA_PROOF.md` (round-1, 242 lines)
- `workflow/drop_4c_6_1/DROP_4c.6.1.W2_TILL_INIT/PLAN_QA_FALSIFICATION.md` (round-1, 266 lines)
- `workflow/drop_4c_6_1/PLAN.md` (L1: lines 1-80 Round-10 narrative; lines 200-549 incl. W1/W2/W4.D1/W4.D2/W5 specs)
- `workflow/drop_4c_6_1/REVISION_BRIEF.md` §2.3-§2.6 (referenced via L1)
- `workflow/drop_4c_6_1/SKETCH.md` §10 (referenced via R10-D1)
- `cmd/till/init_cmd.go` lines 1-80 (current state — confirms pre-W2.D1 baseline: `Group string`, `MCP bool`, `allowedInitGroups = ["till-gen", "till-go"]`, `reservedInitGroups` map present)
- `internal/domain/project.go` lines 110-275 (LSP-equivalent — `ProjectMetadata` struct lines 119-155 has NO `Groups []string` today; W1.D2 adds it. `ProjectInput` at line 212 confirmed; `OrchSelfApprovalIsEnabled()` accessor pattern at line 166 confirmed)
- `internal/app/service.go` lines 280-340 (`CreateProjectInput` at line 286 confirmed; wraps `domain.ProjectInput` via `NewProjectFromInput`; both shipped pre-W2)

---

## 1. Round-1 Absorption Verification

### 1.1 Proof FF1 (= Fals FF2) — D7 KindPayload JSON stopgap → **RESOLVED**

**Round-1 finding:** D7 specified `Metadata.KindPayload = {"groups":[...]}` JSON stopgap, ignoring the typed `Groups []string` field W1.D2 ships before D7 dispatch.

**Round-2 absorption (PLAN.md lines 11-16, 348-380):**
- Header narrative line 15: "Proof FF1 = Fals FF2 (D7 KindPayload JSON stopgap): ABSORBED per R10-D2. D7 acceptance + RiskNotes + ContextBlocks + KindPayload now specify `Metadata.Groups = payload.Groups` (typed field shipped by W1.D2). `KindPayload` stopgap removed. W2-GROUPS-R1 refinement RESOLVED inline."
- D7 Acceptance line 348: "`Metadata.Groups = payload.Groups` — typed field from W1.D2 (`internal/domain/project.go:ProjectMetadata.Groups []string`). Write directly. NO `KindPayload` JSON stopgap."
- D7 AcceptanceCriteria line 361: "`Metadata.Groups = payload.Groups` (typed `[]string` field from W1.D2 — verify via LSP `documentSymbol` on `internal/domain/project.go:ProjectMetadata` after W1.D2 ships; field name is `Groups`, JSON tag `groups,omitempty`)."
- D7 AcceptanceCriteria line 362: "NO `Metadata.KindPayload` JSON stopgap for groups. KindPayload left at its zero value."
- D7 RiskNotes line 370: "`Metadata.Groups` typed field: verify via LSP `documentSymbol` on `internal/domain/project.go:ProjectMetadata` after W1.D2 ships. The field was added by W1.D2 (`Groups []string` with JSON tag `groups,omitempty`). If W1.D2 has not shipped when D7 dispatches, this field will not exist — D7 MUST wait for W1 to be `complete` (already enforced by wave-level `W2 blocked_by W1`)."
- D7 ContextBlocks line 374: "constraint (critical): Use `Metadata.Groups = payload.Groups` (typed field from W1.D2). Do NOT use `Metadata.KindPayload = {\"groups\":[...]}` stopgap. The typed field exists post-W1.D2 and is the correct consumer surface."
- D7 ContextBlocks line 375: "constraint (critical): `Metadata.Groups` typed field requires W1.D2 to have shipped. D7 is Wave C (W2 blocked_by W1). By D7 dispatch time, W1.D2 is complete. Verify via LSP before writing."
- D7 KindPayload line 380: "...Metadata.Groups=payload.Groups (typed field from W1.D2 — NOT KindPayload)" — explicit anti-stopgap framing.
- Raised Refinements table line 405: "W2-GROUPS-R1 ... **RESOLVED** — R10-D2: W1.D2 ships `ProjectMetadata.Groups []string` typed field. D7 uses it directly. No stopgap."

**Verdict:** FULLY ABSORBED. The shipped-but-not-wired anti-pattern is decisively closed: typed field is the consumer, stopgap explicitly forbidden, LSP-verification gate at D7 dispatch, refinement closed inline.

---

### 1.2 Proof NIT1 (= Fals FF3) — `_BLOCKERS.toml` missing W2.D1 entry → **RESOLVED**

**Round-1 finding:** `_BLOCKERS.toml` had entries for D2-D7 but no row for W2.D1; cross-wave `4c.6.1.W4.D1` blocker not mirrored.

**Round-2 absorption:**
- Header narrative line 17: "Fals FF3 = Proof NIT1 (_BLOCKERS.toml W2.D1 entry missing): RESOLVED UNILATERALLY by orchestrator before round-2 dispatch. `_BLOCKERS.toml` already has the W2.D1 entry. Marked RESOLVED here."
- `_BLOCKERS.toml` lines 6-9: now contains explicit W2.D1 entry with `blocked_by = ["4c.6.1.W4.D1"]` and rationale.
- L1 PLAN.md Round-10 note line 27: "W2 `_BLOCKERS.toml`: missing W2.D1 entry. Orchestrator-fixed unilaterally."

**Verdict:** FULLY ABSORBED. `_BLOCKERS.toml` now mirrors PLAN.md for all 7 droplets including the cross-wave W4.D1 blocker on D1.

---

### 1.3 Proof NIT2 — D1 missing CONSUMER-TIE bullet → **RESOLVED**

**Round-1 finding:** D1 acceptance bullets did not include explicit `run(ctx, args, &out, io.Discard)` end-to-end test bullet (only said "`mage test-pkg ./cmd/till` passes").

**Round-2 absorption (PLAN.md line 74):**
"CONSUMER-TIE: validation behavior tested via `run(ctx, args, &out, io.Discard)` end-to-end — at minimum: (a) valid single-group `--json '{\"name\":\"x\",\"groups\":[\"go\"]}'` (no `mcp` key — verifies nil→true default); (b) valid multi-element `--json '{\"name\":\"x\",\"groups\":[\"go\",\"fe\"],\"mcp\":false}'`; (c) invalid group `--json '{\"name\":\"x\",\"groups\":[\"bogus\"]}'` expects non-zero exit + error substring \"invalid\". Unit assertions on `validateInitPayload` are acceptable as supplement."

D1 AcceptanceCriteria line 87 repeats: "CONSUMER-TIE: three `run()` end-to-end tests (valid single-group no-mcp-key, valid multi-group mcp-false, invalid group error)."

**Verdict:** FULLY ABSORBED. The (a)/(b)/(c) three-test contract is concrete, addresses the MCP `*bool` nil→true case explicitly, and matches D2/D5/D6/D7's CONSUMER-TIE pattern.

---

### 1.4 Proof NIT3 (= Fals NIT3 partial overlap) — D3/D4 JSON-mode run() mirror → **RESOLVED**

**Round-1 finding:** D3 and D4 leaned on `teatest_v2` drive-model tests for TUI behavior; explicit JSON-mode `run()` CONSUMER-TIE coverage was implicit/missing.

**Round-2 absorption:**
- D3 Acceptance line 167: "CONSUMER-TIE supplement: `run(..., '--json', '{\"name\":\"x\",\"groups\":[\"go\",\"fe\"],\"mcp\":false}')` exercises the multi-group payload path without entering the TUI; this is the JSON-mode mirror of D3's TUI multi-select."
- D3 AcceptanceCriteria line 179: "CONSUMER-TIE supplement: `run(--json '{\"name\":\"x\",\"groups\":[\"go\",\"fe\"],\"mcp\":false}')` passes."
- D3 KindPayload line 194: "`TestRunInit_JSONMode_MultiGroup` action=add — CONSUMER-TIE supplement; run() with groups:[go,fe]"
- D4 Acceptance line 215: "CONSUMER-TIE supplement: `run(..., '--json', '{\"name\":\"x\",\"groups\":[\"go\"],\"mcp\":true}')` (MCP=true path) + `run(..., '--json', '{\"name\":\"x\",\"groups\":[\"go\"],\"mcp\":false}')` (MCP=false path) + `run(..., '--json', '{\"name\":\"x\",\"groups\":[\"go\"]}')` (no `mcp` key — verifies nil→true default from D1's MCPRegistration) are all exercised and pass."
- D4 AcceptanceCriteria line 225: "CONSUMER-TIE supplement: three `run(--json)` paths (mcp:true, mcp:false, no-mcp-key) all pass."
- D4 KindPayload line 240: "`TestRunInit_JSONMode_MCPPaths` action=add — CONSUMER-TIE supplement; run() with mcp:true, mcp:false, no-mcp-key (nil→true)"

**Verdict:** FULLY ABSORBED. D3 mirrors multi-group, D4 mirrors three MCP modalities through the JSON path; teatest drive-model remains primary for TUI assertions per "real terminal unavailable in CI" pragmatic.

---

### 1.5 Proof NIT4 — D7 wrong cite for CreateProjectInput → **RESOLVED**

**Round-1 finding:** D7 referenced "`internal/domain/project.go:212` (`CreateProjectInput`)" but line 212 is `ProjectInput`, not `CreateProjectInput`.

**Round-2 absorption (PLAN.md line 378):**
"`reference` (normal): REVISION_BRIEF §2.5; `internal/app/service.go:286` (`CreateProjectInput` — the shape D7 constructs) wraps `internal/domain/project.go:212` (`ProjectInput` — internal validation shape called by `NewProjectFromInput`); `internal/domain/project.go:119` (`ProjectMetadata` — confirmed no Groups field TODAY; W1.D2 adds it)."

LSP-verification (today's tree):
- `internal/app/service.go:286` = `type CreateProjectInput struct {` ✓ confirmed.
- `internal/domain/project.go:212` = `type ProjectInput struct {` ✓ confirmed.
- Both cites correct and disambiguated.

**Verdict:** FULLY ABSORBED. The two-shape distinction is explicit; cite is line-accurate against today's tree.

---

### 1.6 Proof NIT5 — D5 missing explicit `blocked_by W4.D1` → **RESOLVED**

**Round-1 finding:** D5 only had `blocked_by W2.D4`; relied on transitive wave-level `W2 blocked_by W4.D1`. The D3/D4 explicit-redundant pattern preferred explicit `blocked_by 4c.6.1.W4.D1` on D5.

**Round-2 absorption:**
- D5 Blocked by line 252: "W2.D4, 4c.6.1.W4.D1 (W4.D1's `git mv` makes the canonical embedded paths `builtin/agents/go/`, `builtin/agents/fe/`, `builtin/agents/gen/` — D5 reads from these unprefixed paths; without W4.D1 the paths don't exist and `fs.ReadDir` returns ENOENT)"
- Decomposition Shape table row line 47: "W2.D5 ... Blocked by: W2.D4, 4c.6.1.W4.D1"
- Blockers Reference table row line 394: "W2.D5 | W2.D4, 4c.6.1.W4.D1"
- `_BLOCKERS.toml` lines 27-29: D5 entry `blocked_by = ["W2.D4", "4c.6.1.W4.D1"]` with explicit ENOENT rationale.

**Verdict:** FULLY ABSORBED. Three places (inline `Blocked by`, decomp table, Blockers Reference) consistent; `_BLOCKERS.toml` mirrors.

---

### 1.7 Proof NIT6 (= Fals NIT4 in part) — `agents.toml` multi-group aggregation gap → **RESOLVED PER DEV CALL**

**Round-1 finding:** REVISION_BRIEF §2.3 line 60 specified aggregation of `[<group>]` + `[<group>.<kind>]` into `<project>/agents.toml`; L2 D6 covered `template.toml` only, not `agents.toml`. Three options surfaced.

**Round-2 absorption — dev directive 2026-05-12 (L1 PLAN.md line 31, R10 Additional dev call):**
"W2 Proof NIT6 (agents.toml multi-group aggregation gap) — dev directive 2026-05-12: do NOT defer. RESOLVED per Option (a) — W4.D2 acceptance already specifies single `agents.example.toml` fixture with BOTH `[go]` and `[fe]` group sections present (line ~397 acceptance bullet). W2 D5/D6 consume this single fixture for multi-group projects (no separate per-group aggregation logic needed at init-time). Marked RESOLVED."

W2 PLAN.md line 22 (header narrative): "Proof NIT6 (agents.toml multi-group aggregation gap): DEFERRED. Gap is faithful to L1 contract (L1 acceptance bullets don't include `agents.toml` multi-group aggregation). Surfaced to orchestrator in closing summary. See Notes §agents.toml Gap."

W2 PLAN.md "agents.toml Multi-Group Aggregation Gap" notes section line 434-442 retains the option (a) / (b) discussion but marks the planner recommendation as (a).

L1 W4.D2 Acceptance line 433: "`agents.example.toml` sections: `[go]` replaces `[agents]`; `[go.plan-qa-proof]` replaces `[agents.plan-qa-proof]` etc. Full schema per REVISION_BRIEF §2.12. Both `[go]` and `[fe]` group sections present."

**Cross-reference verification:** L1 W4.D2 Acceptance (line 433) explicitly says "Both `[go]` and `[fe]` group sections present" — confirming the multi-group fixture lands in W4.D2 per dev call. W4.D1 ships `fe/` group agent dir; W4.D2 ships the multi-group TOML fixture. W2 consumes a single fixture (existing `copyAgentsTOML` static-write pattern) that already contains all groups — no new per-group aggregation logic in W2.

**Verdict:** FULLY ABSORBED per dev call. L1 PLAN.md Round-10 narrative + W4.D2 acceptance + W2 PLAN.md note alignment all coherent. The minor wording mismatch (W2 PLAN.md still says "DEFERRED" in line 22 while the L1 Round-10 note says "RESOLVED") is a narrative-history artifact: the W2 round-2 planner authored that line before the dev's R10 additional call. The OPERATIVE resolution lives in L1 R10 + W4.D2 acceptance; W2 has no new code obligation for `agents.toml` aggregation. This is not a finding — the planner recommendation in the W2 note section matches the dev's chosen Option (a). NOT raising as NIT (the operative resolution is consistent and the L2 plan has no behavioral gap).

---

### 1.8 Proof NIT7 (= Fals NIT3) — MCP `bool` → `*bool` builder-discretion ambiguity → **RESOLVED**

**Round-1 finding:** D4 RiskNote left `bool` vs `*bool` decision to builder; REVISION_BRIEF §2.6 mandates "default true if absent" — only `*bool` with nil→true is spec-consistent.

**Round-2 absorption:**
- Header narrative line 23: "Proof NIT7 = Fals NIT3 (MCP bool vs *bool): ABSORBED. `initJSONPayload.MCP` changes from `bool` to `*bool` in **D1** (first droplet touching `initJSONPayload`). `MCPRegistration() bool` accessor added (nil→true default). D4 reads via `payload.MCPRegistration()`. D1 KindPayload updated."
- D1 Acceptance line 66: "`initJSONPayload.MCP bool` changes to `MCP *bool` with JSON tag `\"mcp,omitempty\"`. A helper `func (p initJSONPayload) MCPRegistration() bool { if p.MCP == nil { return true }; return *p.MCP }` is added, mirroring `OrchSelfApprovalIsEnabled()` pattern (`internal/domain/project.go:157`). Omitting `\"mcp\"` from a `--json` payload defaults to YES (MCP registration enabled)."
- D1 AcceptanceCriteria line 80: "`initJSONPayload.MCP *bool` with JSON tag `\"mcp,omitempty\"`. `MCPRegistration() bool` accessor: nil→true."
- D1 RiskNotes line 95: "MCP `*bool` change: `payload.MCPRegistration()` accessor is the ONLY correct call site for reading MCP intent. Do NOT read `payload.MCP` directly anywhere in D1's edits. D4 and later droplets also use `MCPRegistration()`."
- D1 ContextBlocks line 102: "`decision` (normal): `MCP *bool` nil→true mirrors the `OrchSelfApprovalEnabled *bool` pattern at `internal/domain/project.go:128`. `MCPRegistration() bool` is the read accessor."
- D1 KindPayload line 106: "MCP bool → MCP *bool (json:mcp,omitempty); add MCPRegistration() bool accessor (nil→true)"
- D4 RiskNotes line 231: "`payload.MCPRegistration()` is the correct call site for reading MCP intent (defined in D1). Do NOT read `payload.MCP` directly — the `*bool` requires the accessor."
- D4 ContextBlocks line 237: "`decision` (normal): D4 reads `payload.MCPRegistration()` for all MCP-reading call sites — NOT `payload.MCP` directly."
- LSP-verification of cited pattern: `internal/domain/project.go:166` `func (m ProjectMetadata) OrchSelfApprovalIsEnabled() bool` confirmed — same nil→true shape (note: the W2 spec cites the pattern at line 157; actual function declaration is at line 166 — the doc-comment block starts at line 157. Minor cite drift but not material: the pattern is correctly identified).

**Verdict:** FULLY ABSORBED. The change moves to D1 (the first droplet touching `initJSONPayload`), accessor is mandated, D4 reads via accessor, mirror-pattern documented. The cite-line precision on the `OrchSelfApprovalEnabled` example is informational (158 vs 166); the SHAPE is correctly mirrored. Not raising as a NIT — the cited pattern is unambiguous to anyone running LSP.

---

### 1.9 Fals FF1 — till-prefix drift in W4.D1 → **RESOLVED PER R10-D1**

**Round-1 finding:** W2.D1 declared canonical names `go`, `fe`, `gen` (no prefix) but W4.D1's original spec retained `till-go/` + `till-gen/` subdir paths. Cross-wave contract mismatch.

**Round-2 absorption (L1 R10-D1 + W4.D1 update + W2 alignment):**
- L1 PLAN.md line 16 (R10-D1): "W4.D1 renames embedded subdirs `till-go/` → `go/` + `till-gen/` → `gen/` via `git mv` (preserves history); W1.D3 updates `agentBodyDefaultGroup` constant from `\"till-go\"` → `\"go\"` + `agentBodyFallbackGroup` from `\"till-gen\"` → `\"gen\"` in `render.go`. Canonical group names downstream: `go`, `fe`, `gen` — no `till-` prefix."
- L1 W4.D1 spec line 346: "Round 10 absorption (W1+W2 fals FF1): scope expanded to include `git mv till-go → go` + `git mv till-gen → gen` for canonical-group-name alignment."
- L1 W4.D1 Execution order line 350-353: `git mv` operations as Step 1 + Step 2.
- L1 W4.D1 Acceptance line 381: "Post-rename directory listing: `internal/templates/builtin/agents/` contains exactly `go/`, `gen/`, `fe/`, `till-gdd/` (4 subdirs). NO `till-go/`. NO `till-gen/`."
- L1 W4.D1 KindPayload line 416: explicit `git_mv` action entries.
- W2 PLAN.md line 16 (header narrative): "Fals FF1 (till-prefix drift in W4.D1): ABSORBED per R10-D1. W4.D1 performs `git mv till-go → go` + `git mv till-gen → gen`. Canonical group names are `go`, `fe`, `gen` (no `till-` prefix). D1's `allowedInitGroups` uses these names. D5's embed path uses `builtin/agents/<group>/` (unprefixed post-W4.D1). All references to `till-gen`/`till-go` removed from W2 spec."
- W2 D1 Acceptance line 67: "`allowedInitGroups` changes from `[\"till-gen\", \"till-go\"]` to `[\"gen\", \"go\"`, \"fe\"]`."
- W2 D1 AcceptanceCriteria line 81: "`allowedInitGroups = []string{\"gen\", \"go\", \"fe\"}`."
- W2 D1 RiskNotes line 96: "W4.D1 must be `complete` before D1 dispatches — confirms canonical group names `go`/`fe`/`gen` (no `till-` prefix)."
- W2 D5 Acceptance line 255: "Embed path: `builtin/agents/<group>/` (unprefixed — W4.D1's canonical names `go`/`fe`/`gen`, NOT `till-go`/`till-gen`)."
- W2 D5 RiskNotes line 276: explicit verification step via `fs.ReadDir`.
- W2 D5 ContextBlocks line 282: "`constraint` (critical): embed path uses UNPREFIXED group names (`go`/`fe`/`gen`). Builder must NOT use `till-go`/`till-gen` — those paths will return ENOENT after W4.D1's `git mv`."

**Verdict:** FULLY ABSORBED. Three-way coordination (R10-D1 decision + W4.D1 spec rewrite + W2 reference cleanup) consistent across all docs. The `blocked_by 4c.6.1.W4.D1` on both D1 and D5 ensures dispatch-time gate.

---

### 1.10 Fals NIT1 — D3/D4 coarse `blocked_by W5` → **DOCUMENTED AS DISPATCH-OPTIMIZATION**

**Round-1 finding:** D3 only needs W5.D4 (`picker_multi.go`); D4 only needs W5.D2 (`confirm.go`). Wave-level `blocked_by W5` over-serializes ~2-3 hours.

**Round-2 absorption (header narrative line 24 + Notes section line 426):**
- Header: "Fals NIT1 (coarse blocked_by W5 for D3/D4): DOCUMENTED. D3 only needs `picker_multi.go` (W5.D4); D4 only needs `confirm.go` (W5.D2). Tightening to droplet-level external blockers is a dispatch-optimization; pre-Drop-4b dispatcher enforces at wave level. Kept at wave level for L2 simplicity. Tracked as dispatch-optimization note."
- Notes "Dispatch-optimization note (Fals NIT1 disposition)" line 426: "D3 only needs `picker_multi.go` (W5.D4) and D4 only needs `confirm.go` (W5.D2). Tightening to droplet-level external blockers (`W2.D3 blocked_by W5.D4`; `W2.D4 blocked_by W5.D2`) would save ~2-3 hours of unnecessary serialization. Pre-Drop-4b the dispatcher enforces at wave level, making this documentation-only. Kept at wave level for L2 simplicity. Future drop may tighten when the dispatcher supports droplet-level external-blocker granularity."

The D3 Blocked by line 158 and D4 line 206 carry inline notes: "note: D3 specifically needs W5.D4 ... wave-level blocker is sufficient pre-Drop-4b dispatcher" / "note: D4 specifically needs W5.D2 ... wave-level blocker is sufficient pre-Drop-4b dispatcher".

**Verdict:** ABSORBED-AS-DOCUMENTATION. Explicit pre-Drop-4b dispatcher justification given; the optimization is not a correctness defect; the wave-level keep is intentional. Not raising as a NIT — the disposition is correctly classified as dispatch-optimization-deferred-with-reason per `feedback_nits_are_first_class.md`.

---

### 1.11 Fals NIT4 — D6 multi-group `template.toml` partial-state idempotency → **RESOLVED**

**Round-1 finding:** D6's blanket "skip if exists" silently fails to add new-group sections when user re-runs with new groups.

**Round-2 absorption:**
- Header narrative line 26: "Fals NIT4 (D6 partial-state idempotency): ABSORBED. D6 acceptance: blanket file-level skip + fail-loud warning if file exists AND one or more selected groups are absent from the existing file. Simpler than per-group block parsing; consistent with no-migration philosophy."
- D6 Acceptance line 302: "**Idempotency with partial-state warning:** If `<destDir>/.tillsyn/template.toml` already exists, the file is NOT overwritten (blanket skip). However: if the existing file is absent one or more `[<group>]` sections for the current selected groups, `runInitPipeline` prints a warning: `\"WARN: <destDir>/.tillsyn/template.toml already exists but is missing sections for group(s): [<missing-list>]. Remove it and re-run to regenerate.\"` (non-fatal — exits zero, warning only). This addresses the partial-state scenario without migration code."
- D6 AcceptanceCriteria line 313: "Blanket skip if file exists. Warning if file exists AND missing group sections. Non-fatal."
- D6 Acceptance line 305: CONSUMER-TIE test (d) "partial-state re-run (file exists, missing group section — verifies warning in output but zero exit)."
- D6 RiskNotes line 320: "Partial-state warning detection: check if the existing `template.toml` content contains `\"[<group>]\"` or `\"[<group>.\"` for each selected group. Simple string check — not full TOML parse."
- D6 ContextBlocks line 324: "`constraint` (critical): blanket skip — do NOT overwrite existing `template.toml`."
- D6 ContextBlocks line 325: "`decision` (normal): partial-state warning uses a simple string presence check (not full TOML parse)."

**Verdict:** FULLY ABSORBED. The fail-loud + non-fatal-warning pattern is consistent with the W2 philosophy elsewhere (D2 FLAT detection is fail-loud; this is warn-loud). The "no migration, fail loud" line is preserved.

---

### 1.12 Fals NIT5 — D7 Language mapping selection-order → **RESOLVED**

**Round-1 finding:** D7 used `go > fe > ""` fixed priority; user's `--group fe --group go` selection-order intent ignored.

**Round-2 absorption:**
- Header narrative line 27: "Fals NIT5 (D7 Language mapping selection-order): ABSORBED. D7 Language mapping uses `payload.Groups[0]` (selection-order wins). Tradeoff documented in RiskNotes."
- D7 Acceptance line 348: "`Language` = `payload.Groups[0]` mapped through language closed enum: `\"go\"` if first group is `\"go\"`, `\"fe\"` if first group is `\"fe\"`, `\"\"` if first group is `\"gen\"` (or any unmapped value). Selection-order wins: user's first group pick determines primary language. This respects user intent over fixed priority."
- D7 AcceptanceCriteria line 360: "`Language` = `payload.Groups[0]`-mapped value (go→\"go\", fe→\"fe\", gen→\"\"). Selection-order wins."
- D7 RiskNotes line 371: "`Language` mapping: `payload.Groups[0]` is the source. For a `[\"gen\"]` project, Language = `\"\"` (correct — gen has no language bias). For `[\"gen\", \"go\"]`, Language = `\"\"` because gen is first. User intent: if user selected gen before go, gen is primary. The fixed go-priority heuristic was explicitly rejected per NIT5 absorption — selection-order is the policy."
- D7 ContextBlocks line 376: "`decision` (normal): `Language` = `payload.Groups[0]` mapped value (selection-order wins). Documented as explicit policy: \"user's first group selection determines primary language; fixed-priority heuristic rejected per plan-QA NIT5.\""
- LSP-cross-check: `internal/domain/project.go:279` `isValidProjectLanguage` accepts `"" | "go" | "fe"` closed enum (verified above) — the mapping rule lands in the closed enum.

**Verdict:** FULLY ABSORBED. Selection-order policy is concrete; rejected alternative is named explicitly per `feedback_nits_are_first_class.md` audit-discipline.

---

### 1.13 Fals NIT6 — `reservedInitGroups` ambiguous fate → **RESOLVED**

**Round-1 finding:** D1 RiskNote left `reservedInitGroups` "remove or keep empty" to builder.

**Round-2 absorption:**
- Header narrative line 28: "Fals NIT6 (reservedInitGroups ambiguous fate): ABSORBED. D1 acceptance explicitly pins deletion of `reservedInitGroups` map and its doc-comment."
- D1 Acceptance line 68: "`reservedInitGroups` map AND its doc-comment are DELETED entirely. The reserved-group validation branch in `validateInitPayload` is also removed. If a future group needs reservation, the validation can re-introduce it."
- D1 AcceptanceCriteria line 82: "`reservedInitGroups` map + doc-comment DELETED. Reserved-group branch in `validateInitPayload` removed."
- D1 ContextBlocks line 99: "`constraint` (critical): `reservedInitGroups` — DELETE entirely. Do NOT keep empty. Rationale: the rationale for its existence (till-gdd reserved-but-not-shipped) evaporates with the new naming scheme. Empty map is dead code."
- D1 KindPayload line 106: "{\"file\":\"cmd/till/init_cmd.go\",\"symbol\":\"reservedInitGroups\",\"action\":\"delete\",\"shape_hint\":\"delete var + doc-comment + reserved-group branch in validateInitPayload\"}"

**Verdict:** FULLY ABSORBED. Three-place consistency (Acceptance + AC + ContextBlocks + KindPayload), delete-entirely policy pinned.

---

### 1.14 Fals NIT7 — D1 `Disabled bool` dead field fate → **RESOLVED**

**Round-1 finding:** D1 acceptance "all enabled" left the `Disabled bool` field's interim fate (D1→D3) unspecified.

**Round-2 absorption:**
- Header narrative line 29: "Fals NIT7 (Disabled bool dead field fate): ABSORBED. D1 acceptance: keep `Disabled bool` on `initTUIGroupRow` for D1→D3 interim (all rows enabled, field is inert). D3 deletes it with the full picker_multi.go replacement."
- D1 Acceptance line 70: "`initTUIGroupRows` updated to three rows (gen, go, fe), all enabled. The `Disabled bool` field on `initTUIGroupRow` is KEPT for the D1→D3 interim (all rows have `Disabled: false` — the field is inert but the struct shape is preserved; D3 removes it when `picker_multi.go` takes over)."
- D1 AcceptanceCriteria line 84: "`initTUIGroupRows` has three rows (gen/go/fe), all enabled. `Disabled bool` field KEPT on struct (inert, D3 removes it)."
- D1 ContextBlocks line 101: "`decision` (normal): `Disabled bool` field on `initTUIGroupRow` is kept for D1→D3 interim (all rows `Disabled: false` — inert). D3 removes it wholesale when `picker_multi.go` replaces the inline picker."
- D3 Acceptance line 165: "Dead code removed: `initTUIGroupRows []initTUIGroupRow`, `initTUIGroupRow` struct (including `Disabled bool` field), `nextEnabledGroupRow`, `prevEnabledGroupRow`, `groupCursor int` — all replaced by the `picker_multi.go` component."
- D3 AcceptanceCriteria line 177: "Dead code removed: `initTUIGroupRows`, `initTUIGroupRow` struct (and its `Disabled bool` field), `nextEnabledGroupRow`, `prevEnabledGroupRow`, `groupCursor int`."
- D3 KindPayload line 194: explicit struct + field deletion.

**Verdict:** FULLY ABSORBED. The keep-then-delete sequence is explicit at both D1 (keep) and D3 (delete) sites.

---

## 2. R10 Cross-Cutting Absorption Verification

### 2.1 R10-D1 (canonical group names `go`, `fe`, `gen`)

- D1 `allowedInitGroups = ["gen", "go", "fe"]` (line 67, 81). ✓
- D5 embed path `builtin/agents/<group>/` unprefixed (line 255, 269). ✓
- All references to `till-gen`/`till-go` removed from production code per line 86 + line 282. ✓
- Cross-wave coordination with W4.D1's `git mv` enforced via D1 + D5 explicit `blocked_by 4c.6.1.W4.D1`. ✓

### 2.2 R10-D2 (Groups typed field consumption)

- D7 reads `Metadata.Groups = payload.Groups` directly (line 349, 361, 374, 380). ✓
- No `KindPayload` JSON stopgap (line 362). ✓
- W2-GROUPS-R1 refinement marked RESOLVED inline (line 405). ✓
- LSP-verification gate at D7 dispatch (line 370). ✓

---

## 3. _BLOCKERS.toml ↔ PLAN.md Mirror Verification

| Droplet | PLAN.md `Blocked by` (inline) | Decomp table | Blockers Reference table | `_BLOCKERS.toml` | Match? |
|---------|-------------------------------|--------------|--------------------------|------------------|--------|
| W2.D1 | `4c.6.1.W4.D1` (line 63) | line 43 | line 390 | line 8 `["4c.6.1.W4.D1"]` | ✓ |
| W2.D2 | `W2.D1` (line 118) | line 44 | line 391 | line 13 `["W2.D1"]` | ✓ |
| W2.D3 | `W2.D2, 4c.6.1.W5` (line 158) | line 45 | line 392 | line 18 `["W2.D2", "4c.6.1.W5"]` | ✓ |
| W2.D4 | `W2.D3, 4c.6.1.W5` (line 206) | line 46 | line 393 | line 23 `["W2.D3", "4c.6.1.W5"]` | ✓ |
| W2.D5 | `W2.D4, 4c.6.1.W4.D1` (line 252) | line 47 | line 394 | line 28 `["W2.D4", "4c.6.1.W4.D1"]` | ✓ |
| W2.D6 | `W2.D5` (line 297) | line 48 | line 395 | line 33 `["W2.D5"]` | ✓ |
| W2.D7 | `W2.D6` (line 341) | line 49 | line 396 | line 38 `["W2.D6"]` | ✓ |

All 7 droplets agree across PLAN.md (3 places) + `_BLOCKERS.toml`. Perfect mirror.

---

## 4. PLAN-QA-DISCIPLINE Checks

### 4.1 R1 — Test-runner droplet `blocked_by` covers behavior shipped

Each D-series droplet ships its own tests in the same droplet (no separate test-runner droplets). Behavior-dependency check:

- D1's tests cover D1's rename + `MCPRegistration` + validator → D1's `blocked_by 4c.6.1.W4.D1` covers cross-wave canonical-name dependency. ✓
- D2's tests cover FLAT + old-schema detection → D2's `blocked_by W2.D1` covers Group→Groups rename dependency. ✓
- D3's tests cover multi-select TUI → D3's `blocked_by W2.D2, 4c.6.1.W5` covers `picker_multi.go` + serial chain. ✓
- D4's tests cover MCP confirm step → D4's `blocked_by W2.D3, 4c.6.1.W5` covers `confirm.go` + serial chain. ✓
- D5's tests cover subdir-per-group + multi-group copy → D5's `blocked_by W2.D4, 4c.6.1.W4.D1` covers serial chain + canonical embed paths. ✓
- D6's tests cover `template.toml` write + partial-state warning → D6's `blocked_by W2.D5` covers serial chain. ✓
- D7's tests cover full project record + typed `Metadata.Groups` → D7's `blocked_by W2.D6` (with transitive W2 `blocked_by W1` for typed field) covers serial chain + cross-wave typed-field dependency. ✓

**Verdict:** R1 PASS.

### 4.2 R2 — Narrative droplet count vs enumerated list

- Narrative line 38 (Decomposition Shape): "Seven atomic droplets in a strict serial chain."
- Decomposition Shape table lines 42-49: 7 rows (D1, D2, D3, D4, D5, D6, D7).
- Body droplets line 55, 110, 150, 198, 244, 289, 333: 7 explicit droplet headers.
- Blockers Reference table lines 388-396: 7 rows.
- `_BLOCKERS.toml` lines 7-39: 7 `[[blockers]]` entries.

All five sources agree on 7. **Verdict:** R2 PASS.

### 4.3 Acyclicity / serial-chain coverage

Topo-walk:
- D1 ← {W4.D1}
- D2 ← {D1}
- D3 ← {D2, W5}
- D4 ← {D3, W5}
- D5 ← {D4, W4.D1}
- D6 ← {D5}
- D7 ← {D6}

Strict serial D1→D2→…→D7 chain. External blockers (W4.D1, W5) acyclic. **Verdict:** PASS.

### 4.4 Shared-file/package coverage

All 7 droplets share `cmd/till/init_cmd.go` + `cmd/till/init_cmd_test.go` + package `cmd/till`. Fully serialized via D1→D7 chain. No sibling-overlap-without-blocker risk. **Verdict:** PASS.

---

## 5. CONSUMER-TIE Test Contract Verification

Per W2 PLAN.md Notes §CONSUMER-TIE (line 412): "All tests in `init_cmd_test.go` invoke `run(ctx, args, &out, io.Discard)` end-to-end ... The end-to-end `run()` call is the primary gate for each droplet's acceptance criteria."

Per-droplet verification:

| Droplet | Explicit `run()` end-to-end bullet? | Test count + cases |
|---------|-------------------------------------|--------------------|
| D1 | ✓ line 74 / 87 | 3: valid-single-no-mcp, valid-multi-mcp-false, invalid-group |
| D2 | ✓ line 124 / 132 | 3: FLAT-present, old-schema-present, clean-state |
| D3 | ✓ line 167 / 179 | 1 supplement: multi-group `--json` (teatest primary for TUI) |
| D4 | ✓ line 215 / 225 | 3 supplements: mcp:true, mcp:false, no-mcp-key (teatest primary for TUI step) |
| D5 | ✓ line 261 / 272 | 2: single-group + multi-group subdir creation |
| D6 | ✓ line 305 / 315 | 4: HOME-present, HOME-absent, idempotent, partial-state-warning |
| D7 | ✓ line 353 / 364 | 3: in-git-repo, not-in-git-repo, idempotent-rerun |

All 7 droplets carry explicit `run(ctx, args, &out, io.Discard)` CONSUMER-TIE bullets in Acceptance + AcceptanceCriteria. **Verdict:** PASS.

---

## 6. Premises / Evidence / Trace / Conclusion / Unknowns

### Premises

- All round-1 plan-QA findings (1 FF + 7 NITs proof; 3 FFs + 7 NITs falsification) must be absorbed in round-2 with explicit ABSORBED/RESOLVED/DEFERRED-WITH-REASON status (per `feedback_nits_are_first_class.md`).
- R10 cross-cutting decisions (R10-D1 canonical group names; R10-D2 Groups typed field) must land in W2.
- `_BLOCKERS.toml` must mirror PLAN.md for all 7 droplets including cross-wave blockers.
- PLAN-QA-DISCIPLINE R1 (test-runner blocked_by coverage) and R2 (narrative count = enumerated count) must hold.
- CONSUMER-TIE test contract (`run(ctx, args, &out, io.Discard)` primary gate) must be explicit on every droplet.
- D7 must consume the typed `ProjectMetadata.Groups` field from W1.D2 (not `KindPayload` JSON stopgap).
- D1 + D5 must reference W4.D1's `git mv` canonical paths (`go`/`fe`/`gen`, no `till-` prefix).

### Evidence

- W2 PLAN.md round-2 (443 lines) read end-to-end; every header-narrative absorption claim cross-verified against the body section it claims to update.
- `_BLOCKERS.toml` (40 lines) read end-to-end; mirror-table built (Section 3) confirms 7-of-7 droplet match across PLAN.md + TOML.
- L1 PLAN.md Round-10 narrative + W4.D1 + W4.D2 specs read; all R10 absorptions cross-verified.
- Round-1 PROOF (242 lines) and FALSIFICATION (266 lines) read end-to-end; each finding mapped to a round-2 absorption.
- LSP-equivalent confirmation:
  - `internal/app/service.go:286` = `type CreateProjectInput struct` ✓
  - `internal/domain/project.go:212` = `type ProjectInput struct` ✓
  - `internal/domain/project.go:119-155` = `ProjectMetadata` struct lacking `Groups []string` today (W1.D2 ships it) ✓
  - `internal/domain/project.go:166` = `func (m ProjectMetadata) OrchSelfApprovalIsEnabled() bool` (nil→true accessor pattern D1 mirrors) ✓
  - `cmd/till/init_cmd.go:34-52` = current pre-W2.D1 baseline confirms `Group string`, `MCP bool`, `allowedInitGroups = ["till-gen", "till-go"]`, `reservedInitGroups` map all present (W2.D1 will rename/delete these) ✓

### Trace / Cases

- **Round-1 finding → Round-2 disposition mapping**: 14 distinct round-1 findings (1+7 proof + 3+7 falsification — accounting for FF1=FF2 deduplication and NIT3 overlap) all addressed. Section 1.1-1.14 walks each.
- **R10 cross-cutting → W2 absorption mapping**: R10-D1 (canonical group names) lands at D1/D5 + W4.D1 spec; R10-D2 (Groups typed field) lands at D7 + W1.D2 spec. Section 2 walks each.
- **PLAN.md ↔ _BLOCKERS.toml mirror**: 7-row mirror table (Section 3) shows perfect agreement.
- **CONSUMER-TIE per droplet**: 7-row table (Section 5) shows explicit `run()` bullet on every droplet.
- **Acyclic serial chain**: D1→D2→…→D7 walked; external blockers (W4.D1, W5) acyclic.
- **PLAN-QA-DISCIPLINE R1/R2**: each droplet's test-shipping covers its own behavior; narrative count "7" matches all 5 enumeration sources.

### Conclusion

**PASS.** All round-1 plan-QA findings absorbed with explicit per-finding traceability. All R10 cross-cutting decisions landed at the correct droplets. `_BLOCKERS.toml` mirrors PLAN.md perfectly. PLAN-QA-DISCIPLINE R1 + R2 hold. CONSUMER-TIE contract explicit on every droplet. The serial chain is acyclic and the shared-file/package coverage is fully serialized. No FFs. No NITs.

The plan is dispatchable today for W2.D1 once W4.D1 reaches `complete`. D3/D4 await W5 ship; D5 awaits W4.D1 ship (same gate as D1); D7 awaits W1.D2's typed-field landing (transitively enforced by wave-level `W2 blocked_by W1`, with explicit LSP-verification gate at D7 dispatch).

### Unknowns

- **U1 (informational, not blocking):** W2 PLAN.md line 22 narrative says "Proof NIT6 ... DEFERRED" while the operative L1 R10 dev call (line 31) says "RESOLVED per Option (a)". The narrative-history line was written before the R10 additional dev call. Operative resolution is RESOLVED per Option (a) — single multi-group fixture in W4.D2 serves both groups. W2 has no new behavioral code obligation. Not a finding because (a) the operative resolution is consistent across L1 + W4.D2 spec + W2 planner recommendation in Notes §"agents.toml Multi-Group Aggregation Gap" (line 442) which itself recommends Option (a), and (b) the narrative-history line is audit-trail-only, not a build directive. Could optionally be swept to "RESOLVED per dev call" in a future cosmetic pass; not blocking.
- **U2 (informational, not blocking):** D1's RiskNote cites `OrchSelfApprovalEnabled` pattern at line 157, but the function `OrchSelfApprovalIsEnabled()` declaration is at line 166 (line 157 is mid-doc-comment block of the field above). Off-by-9 cite is cosmetic; the pattern is correctly identified and an LSP `documentSymbol` query lands the correct line. Not a finding because the cite SHAPE is right and any builder running LSP recovers the correct location.

Both Unknowns are informational; neither blocks dispatch.

---

## 7. Hylla Feedback

N/A — Hylla was explicitly OFF for this review per spawn-prompt directive. All evidence gathering used `Read` + LSP-equivalent line-precise file inspection against the live `main/` checkout.
