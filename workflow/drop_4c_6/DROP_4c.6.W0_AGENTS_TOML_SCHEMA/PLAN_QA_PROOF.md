# Plan-QA-Proof Round 1 — DROP_4c.6.W0

**Reviewed:** `workflow/drop_4c_6/DROP_4c.6.W0_AGENTS_TOML_SCHEMA/PLAN.md` + `_BLOCKERS.toml`.
**Parent contract:** `workflow/drop_4c_6/PLAN.md` lines 51-67 (W0 sub-plan container Scope + Acceptance + L2 directive).
**Source-of-truth:** `workflow/drop_4c_6/SKETCH.md` § 26.W0 (lines 755-773), § 4 (lines 172-289), § 5 (lines 291-300).
**Round:** 1.
**Verdict mechanic:** PASS = zero findings across all 5 axes. FAIL = ≥1 finding.

## 1. Findings

No findings. All five axes return clean. Per-axis trace below for the audit record.

### Axis trace

- **Atomic-decomposition (D1-D5 sizing)** — each droplet primary-edits one production file with co-located tests + 0-4 golden fixtures. D1 ships `agents.go` + `agents_test.go` + 1 fixture; D2 appends `Resolve` + 5 tests + 4 fixtures; D3 appends `MergeLocal` + 4 tests + 3 fixtures; D4 ships file-disjoint `frontmatter.go` + `frontmatter_test.go` + 6 tests; D5 appends `ConfigError` envelope + 4 tests (revises 2 prior assertions). PLAN.md `Notes § Decomposition rationale` (lines 218) cites the till-go template's 80-120 LOC per droplet target and explicitly defends "five, not three or seven." Sketch-level §25 / §26.W0 do not specify droplet counts — count is left to the planner per `feedback_plan_down_build_up.md` ("droplet counts NOT specified at sketch level"); the L2 sub-planner discharged that responsibility correctly.

- **Parallelization-graph (chain integrity)** — `_BLOCKERS.toml` mirrors PLAN.md inline `Blocked by:` bullets verbatim:
  - D2 `Blocked by: 4c.6.W0.D1` (PLAN.md:85) ↔ `_BLOCKERS.toml:16-18` ✓
  - D3 `Blocked by: 4c.6.W0.D2` (PLAN.md:121) ↔ `_BLOCKERS.toml:20-23` ✓
  - D4 `Blocked by: 4c.6.W0.D3` (PLAN.md:160) ↔ `_BLOCKERS.toml:26-28` ✓
  - D5 `Blocked by: 4c.6.W0.D4` (PLAN.md:196) ↔ `_BLOCKERS.toml:30-33` ✓
  - D1 `Blocked by: —` (PLAN.md:48); no `_BLOCKERS.toml` row needed for an unblocked head.
  Package-lock rationale (all 5 share `internal/config` Go package compile/test unit) is cited inline at PLAN.md:23 quoting `~/.claude/agents/go-planning-agent.md` HARD RULES verbatim. Serial chain is the correct shape per the cited rule. No parallelizable pair was incorrectly serialized — D1+D2 both edit `agents.go` (file-shared, must serial); D4 edits the file-disjoint `frontmatter.go` but the package-lock binds; PLAN.md:29 acknowledges D4's placement is "arbitrary on file-grounds" but defended by semantics-first ordering.

- **Specify-block well-formedness** — every droplet (D1-D5) carries Objective, AcceptanceCriteria (delegating to the inline Acceptance bullets above each Specify section), ValidationPlan (named `mage test-pkg` + per-test `mage test-func` invocations), RiskNotes (3-5 each, citing live API verification needs + alternative-path hedges), ContextBlocks (typed enum × severity per §10.1.7 — `decision`/`constraint`/`reference`/`warning` × `normal`/`high`/`critical`), and KindPayload (JSON-shaped `changes` array with `file`/`symbol`/`action`/`shape_hint` per droplet's modify-or-add scope). AcceptanceCriteria bullets are testable: each names a concrete test function (`TestLoadRegistry_Baseline`, `TestResolve_FullInherit`, `TestMergeLocal_ToolsDenyRejected`, `TestStripFrontmatterKeys_StripModel`, `TestConfigError_FormatsCorrectly`, etc.) with concrete assertion shape (e.g. "asserts `errors.Is(err, ErrToolsDenyNotOverridable)`," "asserts `configErr.Line > 0`"). No "verify it works" hand-waves.

- **Multi-level-decomposition (planner stayed at one level)** — sub-planner emitted FIVE atomic `kind=build` droplets (D1-D5) at L2; no L3 sub-drops. PLAN.md:33-35 declares this explicitly: "Decomposition shape — five atomic droplets serialized on `internal/config` package compile" and PLAN.md `Notes § Decomposition rationale` defends the granularity. No droplet hides further decomposition (e.g., D4's frontmatter strip helper could in principle warrant a sub-drop for YAML-lib choice + parse + emit, but at <80 LOC + 6 tests in one file it stays atomic). The W0 sub-plan container's L2 directive in `workflow/drop_4c_6/PLAN.md:67` proposed "Likely shape: D1 ... D5"; the L2 sub-planner adopted that suggested shape verbatim, which is a coincidence-of-good-fit, not laziness — the rationale in `Notes § Decomposition rationale` independently defends each split.

- **Shipped-but-not-wired** — every new symbol has a declared consumer, either intra-W0 or routed to a named downstream wave:
  - D1: `Preset` / `Override` / `AgentRuntime` / `AgentsRegistry` / `Kind` → consumed by D2 (`Resolve` walks Preset fields + Override pointers per `inheritance_*` test fixtures), D3 (`MergeLocal` operates on `*AgentsRegistry`), D5 (`ConfigError` envelope wraps decode + merge errors). `LoadRegistry` → consumed by D5 (wraps raw `*toml.DecodeError`); production consumer routed to W2 + W3 per `Notes § Out of scope for W0` line 228. ✓
  - D2: `Resolve` / `mergeMaps` / `replaceList` → `Resolve` consumed by D3 (PLAN.md:121 Blocked-by reason + `Notes § Decomposition rationale` line 222); `mergeMaps` consumed by D3 (PLAN.md:130 RiskNote: "reuse D2's `mergeMaps` helper to avoid drift"). Production consumer for `Resolve` routed to W3 per Out-of-scope §. ✓
  - D3: `MergeLocal` / `ErrToolsDenyNotOverridable` → consumed by D5 (D5's `TestMergeLocal_ToolsDenyPositionWrapped` exercises `MergeLocal`'s rejected-deny path wrapped in `ConfigError`); production consumer routed to W2's loader-flow + W3's render-layer per Out-of-scope §. ✓
  - D4: `StripFrontmatterKeys` → no W0-internal consumer (D4 is file-disjoint from D1-D3-D5 schema work); explicitly routed to W3 at PLAN.md:170 RiskNote: "W3 wires this helper at `render.go:assembleAgentFileBody` (per `SKETCH.md` § 26.W3); D4 ships the helper but does NOT wire it." This is a known-and-routed deferral matching §26.W0 Objective ("Foundation for everything else") and W3's §26.W3 AcceptanceCriteria which explicitly call out "Frontmatter strip" as W3 scope. ✓
  - D5: `ConfigError` / `WrapWithPosition` → consumed within D5 itself by `LoadRegistry` (D1) and `MergeLocal` (D3) modifications; D5's KindPayload (PLAN.md:212) names the modify-action on those two functions. Downstream consumers (W3 render layer + future W11 MCP boundary) routed at Out-of-scope § line 231. ✓
  
  No symbol ships into W0's `internal/config` package without either a W0-internal consumer or a documented downstream-wave consumer. The §26.W0 sketch contract demands "schema layer" foundation; downstream wiring (W2/W3) is the next wave's responsibility per the cascade-design principle of decomposing across waves rather than wiring everything at the schema layer.

## 2. Missing Evidence

None. Every claim in the L2 plan traces to one of:

- §26.W0 / §4 / §5 of the locked sketch (source-of-truth);
- The L1 W0 sub-plan container row (`workflow/drop_4c_6/PLAN.md:51-67`);
- The planner-agent HARD RULES the plan cites by section name;
- Concrete test names + fixture filenames inside the droplet's own ContextBlocks/KindPayload.

The two areas where evidence depends on **future** lookup (live API verification) are explicitly hedged in RiskNotes:
- D1 RiskNote (PLAN.md:54): "`pelletier/go-toml/v2` line-number-preservation API has changed across versions; ... verify the actual `*toml.DecodeError` shape via `go doc` before writing assertions."
- D5 RiskNote (PLAN.md:202): "`pelletier/go-toml/v2` position API (`DecodeError.Position()` / `.Key()`) — verify via `go doc github.com/pelletier/go-toml/v2.DecodeError` before authoring."

These are correctly RiskNotes, not silently-assumed claims — they route the unknown into the builder's evidence-gathering step at implementation time. That is the right shape: the L2 plan does not pretend to know the live `pelletier/go-toml/v2` API; it constrains the builder to verify before authoring assertions, and falls back to alternatives if the API doesn't expose what the test asserts.

## 3. Summary

**Verdict: pass**

Finding count: 0.

Rationale: The L2 plan satisfies all five plan-QA-proof axes against the wave-level Specify in `SKETCH.md` § 26.W0 and the L1 W0 sub-plan container contract at `workflow/drop_4c_6/PLAN.md:51-67`. Five atomic droplets serialized on the `internal/config` Go-package compile lock; chain integrity confirmed across PLAN.md inline `Blocked by:` bullets and `_BLOCKERS.toml`; every droplet's Specify block carries the six required fields (Objective / AcceptanceCriteria / ValidationPlan / RiskNotes / ContextBlocks / KindPayload) with testable acceptance bullets; the sub-planner authored ONE level of decomposition without spawning L3 sub-drops; and every new symbol has either an intra-W0 consumer (D1's types → D2/D3/D5; D2's `mergeMaps` → D3; D5's `ConfigError` → D1+D3 modifies) or an explicitly-named downstream-wave consumer (D4's `StripFrontmatterKeys` → W3 render layer; production wiring of `LoadRegistry` / `Resolve` / `MergeLocal` → W2/W3 per the explicit `Notes § Out of scope for W0` section).

Two areas of API-shape uncertainty (`pelletier/go-toml/v2` `DecodeError` position API, surveyed-vs-not YAML lib choice for D4) are correctly captured as RiskNotes that route the live verification to the builder's `go doc` / dep-survey step at implementation time, not silently assumed in acceptance criteria. This is the correct semi-formal shape: Unknowns explicitly routed, not buried.

The L2 plan can advance to plan-QA-falsification's parallel pass and (assuming falsification also clears) into Phase 4 builder spawning of D1.

## 4. Hylla Feedback

N/A — this review touched only Markdown / TOML inputs (`PLAN.md`, `_BLOCKERS.toml`, `SKETCH.md`, `WORKFLOW.md`, `CLAUDE.md`). Hylla indexes Go files only per project memory (`feedback_hylla_go_only_today.md`); no Go-symbol queries were warranted at the L2 plan-QA-proof scope. No fallbacks logged.
