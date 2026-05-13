# PLAN_QA_PROOF — Drop 4c.6.1 W1 TEMPLATE_RESOLUTION — Round 1

**Verdict:** PASS-WITH-NIT — 2 FF findings (load-bearing, must close before W1 dispatch), 4 NIT findings (cosmetic / authoring polish).

The L2 plan is well-grounded against current code (LSP / Read-verified): `loadProjectTemplate`, `loadProjectTemplateCandidate`, `bakeProjectKindCatalog`, `readProjectTierAgent`, `readUserTierAgent`, `assembleAgentFileBody`, `resolveAgentGroup`, `agentBodyDefaultGroup`, `projectAgentsSubdir`, `userAgentsSubdir`, `domain.ProjectMetadata`, and `platform.Paths` all resolve as claimed. Droplet count is internally consistent (3 narrative / 3 enumerated / 3 summary table / "3 atomic droplets"). `_BLOCKERS.toml` mirrors PLAN.md inline `Blocked by:` bullets one-to-one. The blocked_by graph is acyclic (`W4.D1 → D1 → D2; W4.D1 → D3`); shared-file/package pairs (D1+D2 on `service.go` / `internal/app`) are correctly blocked, and disjoint pairs (D1+D3, D2+D3) correctly have no blocker. AC1–AC6 each map to a covering droplet and its `mage test-pkg` validation target. The L1 → L2 package-scope expansion to include `internal/domain` is explicitly documented in three locations (header line 7, scope §C lines 141-142, D2 ContextBlocks lines 322-324) — valid L2 refinement, not scope creep.

Two findings are load-bearing because they affect dispatch correctness; four are cosmetic.

---

## 1. Findings — FF (Load-Bearing)

### FF1 — `_BLOCKERS.toml` references undefined "U1" coordination key for `agentBodyDefaultGroup` rename

**Axis:** specify-block-well-formedness / cross-planner-consistency

**Line citation:** `_BLOCKERS.toml` line 18 — `"...if W4.D1 renames agentBodyDefaultGroup ('till-go' → 'go'), D3 updates the constant in the same commit per U1"`

**Claim:** `_BLOCKERS.toml` cites "per U1" as the rationale for D3 updating `agentBodyDefaultGroup` in the same commit as W4.D1's rename. **There is no "U1" entry anywhere in the W1 PLAN.md, nor in the L1 PLAN.md's W1 section.** Searched both files end-to-end; PLAN.md does not declare an Unknowns list, has no `## Unknowns` heading, and never names `agentBodyDefaultGroup`.

**Evidence:**
- W1 `PLAN.md` lines 1–449: no occurrence of `U1`, `Unknown`, or `agentBodyDefaultGroup`. D3 RiskNotes (lines 377–379) and ContextBlocks (line 384) ONLY discuss `projectAgentsSubdir` (which stays unchanged).
- L1 `PLAN.md` W1 section (lines 188–217): does not mention `agentBodyDefaultGroup` or any "U1" coordination key.
- `internal/app/dispatcher/cli_claude/render/render.go` line 184 confirms `const agentBodyDefaultGroup = "till-go"` exists today.
- W4.D1's spec (L1 PLAN.md lines 322–381) does NOT include `render.go` in its paths, does NOT list `agentBodyDefaultGroup` among its modifications, and does NOT call out renaming the constant. W4.D1's "Drop `go-` prefix" language refers to filenames in `internal/templates/builtin/agents/till-go/` (deleting `go-builder-agent.md` etc.), not the `agentBodyDefaultGroup` runtime constant in render.go.

**Why load-bearing:** This is a "PLAN.md is truth; `_BLOCKERS.toml` mirrors" violation per WORKFLOW.md § "_BLOCKERS.toml — Sibling Blocker Ledger" (line 80: "if the two disagree, PLAN.md is truth and `_BLOCKERS.toml` is stale"). It also breaks PLAN-QA-DISCIPLINE-R1: a NEW behavior implication (rename `agentBodyDefaultGroup`) is referenced as a coordination unknown, but the runtime constant rename is NOT in W4.D1's spec NOR D3's spec. If a builder takes `_BLOCKERS.toml` at face value, D3 would silently widen scope to edit a constant W4.D1 never wired up. If the orchestrator instead reads only PLAN.md, the dispatch path is fine — but the artifacts disagree.

**Fix hint (pick one):**
- (a) **Drop the "per U1" clause** from `_BLOCKERS.toml` line 18 entirely. The blocker rationale stands on its own ("test fixtures use W4.D1-confirmed canonical group names"). The runtime constant `agentBodyDefaultGroup` is NOT being renamed in this drop — group strings are derived at call sites from `project.Language` / `project.Metadata.Groups` / `path.Dir(SystemPromptTemplatePath)`, not from this constant.
- (b) **Add a `## Unknowns` section** to W1 PLAN.md with a "U1" entry explicitly routing the `agentBodyDefaultGroup` rename question to either W4.D1's scope or a future drop, and update W4.D1's spec to match.
- (c) **Confirm with dev** whether the `till-go` → `go` group-name rename should land in W4.D1 (then route through D3 as a same-commit constant update) — and if YES, update both W4.D1 spec and W1 PLAN.md D3 spec to add the `agentBodyDefaultGroup = "go"` change explicitly. This is the most decisive resolution, but it expands W4.D1's scope (render.go added) and adds a path-set entry to D3 (no longer just a signature change).

Recommended: (a) — least scope drift. The "till-go" string in `agentBodyDefaultGroup` is render.go's dispatch-default for empty `SystemPromptTemplatePath`; it's orthogonal to the W4.D1 subdir restructure. Filenames in `internal/templates/builtin/agents/till-go/` are losing the `go-` prefix; the `till-go` SUBDIR NAME is unchanged in W4.D1's scope (read PLAN.md lines 332–353: W4.D1 still writes into `internal/templates/builtin/agents/till-go/...`).

---

### FF2 — D1/D2 testability-seam ownership ambiguous: `loadProjectTemplateWithHome` claimed by D2 but absent from D1's KindPayload

**Axis:** spec-conformance / atomic-decomposition

**Line citation:** D2 RiskNotes line 307–314; D1 KindPayload line 174; D1 Acceptance line 229–239.

**Claim:** D2's RiskNotes section (line 307–314) describes a coordination pattern: *"D1 adds a `loadProjectTemplateWithHome(project *domain.Project, homeDir string)` variant (accepting explicit homeDir for testability) and the published `loadProjectTemplate` calls it with `os.UserHomeDir()`. D2 then calls `loadProjectTemplateWithHome` per group."* But:

- D1's KindPayload (line 174) lists ONLY `{"file":"internal/app/service.go","symbol":"loadProjectTemplate","action":"modify",...}` — no `loadProjectTemplateWithHome` add.
- D1's Acceptance bullets (lines 229–239) require "4-tier resolution" + HOME-tier extension but say nothing about an injectable `homeDir string` parameter or a sister function.
- D1's RiskNotes (lines 246–252) describes test injection via "a seam in `loadProjectTemplate` or a helper that accepts an optional home override (e.g. `loadProjectTemplateCandidate` already takes an absolute path, so the walk loop can be given a fake home-derived path in tests without injecting an interface)" — explicitly proposing a NON-interface, non-`loadProjectTemplateWithHome` approach.

**Two contradictory designs are encoded in the same PLAN.md:**
1. D1 RiskNotes: walk loop is given a fake home-derived path; no new exported/private helper signature.
2. D2 RiskNotes: D1 adds `loadProjectTemplateWithHome(project, homeDir)`; D2 calls it.

**Evidence:**
- W1 PLAN.md D1 RiskNotes line 247–250 (lib-seam approach).
- W1 PLAN.md D2 RiskNotes line 309–313 (`loadProjectTemplateWithHome` approach).
- `internal/app/service.go` line 529: today's `loadProjectTemplate(project *domain.Project) (templates.Template, bool, error)` — no `homeDir` param.

**Why load-bearing:** Builders for D1 and D2 are the same agent invoked at different times (D2 blocked_by D1). When the D2 builder spawns, it expects D1 to have shipped a specific seam. If D1's builder followed D1's own RiskNotes (which describe a different seam) rather than D2's RiskNotes, D2 will have to refactor D1's surface area mid-droplet — silent scope creep. Atomic-decomposition rule: each droplet has a yes/no-verifiable acceptance set; today D1's "did you ship the right seam for D2?" is not in D1's acceptance.

**Fix hint:** Pick one design and inline it into BOTH droplets:
- (a) **Recommended:** Make `loadProjectTemplateWithHome(project, homeDir)` part of D1's KindPayload + acceptance. Add a fourth KindPayload entry under D1: `{"file":"internal/app/service.go","symbol":"loadProjectTemplateWithHome","action":"add","shape_hint":"new package-private; takes (project, homeDir) and runs the 4-tier walk; loadProjectTemplate calls it with os.UserHomeDir()"}`. Then update D1's Acceptance bullet 6 to require this helper to exist with a covering test. D2 then trivially consumes the seam.
- (b) Alternative: pick D1 RiskNotes' "walk-loop fake home path" approach, and rewrite D2 RiskNotes line 309–313 to match — D2 then constructs an array of candidate paths directly rather than calling `loadProjectTemplateWithHome` per group.

Either resolves the ambiguity; the D1 KindPayload version (a) is cleaner because it surfaces the testability seam in the acceptance contract.

---

## 2. Findings — NIT (Cosmetic)

### NIT1 — D2 paths line hedges test-file location (`project_test.go` if exists; or `domain_test.go`)

**Axis:** specify-block-well-formedness

**Line citation:** D2 Paths line 273 — `internal/domain/project_test.go (if exists; or internal/domain/domain_test.go)`.

**Reason:** D2 is a typed-field-only addition to `domain.ProjectMetadata` (no logic). A test verifying field presence isn't strictly required, but the path hedge ("if exists; or...") is the kind of thing a planner can resolve cheaply via a quick `ls internal/domain/`. The L2 planner had every tool needed to determine which file currently exists; leaving the choice to the builder is unnecessary deferral.

**Fix hint:** Run `ls internal/domain/*_test.go` and pin the exact filename. If a `project_test.go` exists, name it; if not, write the test in the file currently used for `ProjectMetadata` round-trip coverage. If no `ProjectMetadata` tests exist anywhere, state explicitly "D2 builder MAY add a marshal/unmarshal round-trip test in `internal/domain/project_test.go` (new file)." Don't ship the hedge.

---

### NIT2 — D2 `mergeTemplates` shape-hint mentions inspecting `internal/templates/schema.go` but does not pin the merge algorithm

**Axis:** spec-conformance

**Line citation:** D2 RiskNotes line 300–305.

**Reason:** "Builder must inspect the struct shape in `internal/templates/schema.go` before writing the merge helper. If a `templates.Merge` function exists, use it. If not, the builder writes a package-private `mergeTemplates(base, overlay templates.Template) templates.Template` in `service.go` that iterates map keys and overlays." This is correct but underspecified: `templates.Template` carries `AgentBindings`, `ChildRules`, `Gates`, `GateRulesRaw`, `Tillsyn`, `StewardSeeds`, `KindRules` (verified via `internal/templates/schema.go` lines 200–248). The plan says "iterates map keys" but `ChildRules` is a slice, not a map. `KindRules` likewise. `Tillsyn` is a struct.

**Fix hint:** Pin the merge semantic per field — e.g., "merge `AgentBindings` map by key (overlay wins on collision per per-kind agent name); merge `ChildRules` by append-then-dedup-on-(WhenParentKind, CreateChildKind, Title) tuple; merge `Gates` map by key (overlay wins); `Tillsyn` struct merged field-by-field with overlay-wins-on-nonzero." Leaving the merge algorithm under-specified is a likely build-QA round source. The plan's "LAST-GROUP-WINS on per-kind keys" only covers `AgentBindings`; the rest is undefined.

---

### NIT3 — D1 RiskNotes' test-seam paragraph and D2 RiskNotes' test-seam paragraph use different terminology for the same concept

**Axis:** specify-block-well-formedness

**Line citation:** D1 RiskNotes line 246–252 vs D2 RiskNotes line 307–314.

**Reason:** D1 calls it "a seam in `loadProjectTemplate`" / "the walk loop can be given a fake home-derived path." D2 calls it "`loadProjectTemplateWithHome(project, homeDir)`." Even after FF2 is resolved, the residual terminology drift will confuse a builder reading both droplets in sequence.

**Fix hint:** After FF2 is closed, use ONE term across both droplets (the chosen design's symbol name, e.g., `loadProjectTemplateWithHome` in D1's KindPayload AND in D2's RiskNotes).

---

### NIT4 — Summary table line 423 uses `*_test.go` glob for D2 paths

**Axis:** specify-block-well-formedness

**Line citation:** PLAN.md line 424 — `D2 | project.go, service.go, *_test.go | internal/domain, internal/app | D1 | —`.

**Reason:** The summary table's path column collapses `service_test.go` + `project_test.go` (or `domain_test.go`) into a glob `*_test.go`. This is a summary-only artifact and unlikely to cause builder confusion (the per-droplet Paths blocks are explicit), but it is a minor inconsistency with D1's row (which spells out `service.go, service_test.go`).

**Fix hint:** Spell the test file names out in the summary row to match D1's row style: e.g. `project.go, service.go, project_test.go, service_test.go`.

---

## 3. Numeric-Consistency Check

**Droplets narrated (Scope text, lines 22–37):** D1, D2, D3 = **3**.
**Droplets enumerated (### Droplets section, lines 218, 265, 334):** D1, D2, D3 = **3**.
**Summary table rows (lines 423–425):** D1, D2, D3 = **3**.
**"Total: **3 atomic droplets**." line 427.**

All counts agree. PLAN-QA-DISCIPLINE-R2: **PASS.**

**Acceptance criteria coverage (PLAN-QA-DISCIPLINE-R1):**

| AC | Behavior shipped by | Tested by (droplet) | Tested by (mage target) | Test-runner blocked_by ships behavior? |
|----|--------------------|---------------------|-------------------------|---------------------------------------|
| AC1 4-tier walk | D1 | D1 | `mage test-pkg ./internal/app` | Same droplet — PASS |
| AC2 HOME wins over embedded | D1 | D1 | `mage test-pkg ./internal/app` | Same droplet — PASS |
| AC3 multi-group + `Groups []string` | D2 | D2 | `mage test-pkg ./internal/domain` + `./internal/app` | D2 blocked_by D1 (D2 ships its own test) — PASS |
| AC4 subdir-per-group | D3 | D3 | `mage test-pkg ./internal/app/dispatcher/cli_claude/render` | Same droplet — PASS |
| AC5 cross-group `gen` fallback preserved | (untouched; D3 explicitly preserves) | existing tests pass unchanged | (existing) | N/A — preservation, not new behavior |
| AC6 `mage ci` green | all three | post-D1+D2+D3 | `mage ci` | Sequential gate — PASS |

PLAN-QA-DISCIPLINE-R1: **PASS** for new-behavior acceptance bullets.

---

## 4. `_BLOCKERS.toml` ↔ PLAN.md Mirroring Audit

| PLAN.md Blocked by: bullet | `_BLOCKERS.toml` row | Match? |
|---|---|---|
| D1 line 223: "Blocked by: W4.D1" | `_BLOCKERS.toml` lines 6–8: `W1.D1` blocked_by `["4c.6.1.W4.D1"]` | YES |
| D2 line 270: "Blocked by: D1" | `_BLOCKERS.toml` lines 10–13: `W1.D2` blocked_by `["W1.D1"]` | YES |
| D3 line 339: "Blocked by: W4.D1" | `_BLOCKERS.toml` lines 15–18: `W1.D3` blocked_by `["4c.6.1.W4.D1"]` | YES |

**Mirroring is one-to-one.** The "per U1" clause in `_BLOCKERS.toml` line 18 is the only discrepancy — captured in FF1.

---

## 5. Cross-Planner Consistency

- **L1 → L2 package expansion** (`internal/app, render` → `internal/app, internal/domain, render`): explicitly documented in three locations (W1 PLAN.md header line 7; Scope §C lines 141–142; D2 ContextBlocks 322–324). Valid L2 refinement, not scope creep. ✓
- **`mergeTemplates` placement in `internal/app/service.go`, not `internal/templates`**: explicitly stated D2 RiskNotes lines 302–305. ✓
- **`agentBodyDefaultGroup` constant rename coordination**: not an Unknown in PLAN.md; the `_BLOCKERS.toml` reference is orphaned. See FF1.

---

## 6. Conclusion

**Verdict: PASS-WITH-NIT.** Plan is structurally sound, well-grounded, internally consistent in droplet count, blocker graph, and acceptance coverage. Close FF1 + FF2 before W1 dispatches; NIT1–NIT4 are authoring polish and can land in the same revision pass or be deferred to the builder's discretion (but per `feedback_nits_are_first_class.md`, default to fixing them).
