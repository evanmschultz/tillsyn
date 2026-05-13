# PLAN_QA_PROOF — DROP 4c.6.1

**Drop:** `4c.6.1`
**Round:** 1 (plan-QA-proof)
**Reviewer:** go-qa-proof-agent
**Document under review:** `workflow/drop_4c_6_1/PLAN.md`
**Source-of-truth:** `workflow/drop_4c_6_1/REVISION_BRIEF.md` (16 subsections §2.1–2.16; 13 acceptance bullets §5.1–5.13), `workflow/drop_4c_6_1/SKETCH.md` (10 design sections), `CLAUDE.md`, `WIKI.md`, `workflow/example/drops/WORKFLOW.md`, `workflow/drop_4c_6/PLAN.md` (L1 pattern reference).

---

## Pass / Fail

**FAIL** — one must-fix (FF1) scope gap: REVISION_BRIEF §2.12's "agents.local.toml deep-merge logic updates to handle the new schema" is not assigned to any wave or droplet in PLAN.md. Plus 5 cosmetic NITs documented below. Wave grouping, blocked_by graph, sub-plan/direct-droplet split, MIGRATION TARGET propagation, refinement logging, and Section 0 hygiene are all clean.

---

## Findings

### FF1 — `agents.local.toml` deep-merge Go-code update missing from scope

**Location:** REVISION_BRIEF §2.12 last bullet: *"agents.local.toml deep-merge logic updates to handle the new schema."*

**Evidence:** PLAN.md W4.D2 ships the new `[<group>]` / `[<group>.<kind>]` schema in `agents.example.toml`, and CLI consumers in W3 reference the new shape. The bake walker (W1) handles `template.toml` resolution, distinct from `agents.toml`/`agents.local.toml` runtime config. The schema shift from `[agents]` + `[agents.<kind>]` (Drop 4c.6 shipped shape) → `[<group>]` + `[<group>.<kind>]` (this drop's target) requires a Go-side decoder change in `internal/config/agents.go` (the file that Drop 4c.6 W0 shipped containing `Preset` + per-kind override merge + `agents.local.toml` deep-merge). No droplet, sub-plan container, or wave in PLAN.md declares `internal/config/agents.go` in `paths` or `internal/config` in `packages`.

**Impact:** Without this work, `agents.toml`/`agents.local.toml` loading would either reject the new schema at decode (hard fail) or silently accept the old schema (regression). Tillsyn enforces templates — claim-vs-impl coherence demands the consumer match the schema.

**Fix:** Add a new wave (or extend W4.D2 / split into W4.D3) covering `internal/config/agents.go` + `internal/config/agents_test.go` updates: schema decode for `[<group>]` + `[<group>.<kind>]`, override merge semantics across groups, `agents.local.toml` deep-merge over the new shape, golden-fixture tests covering the multi-group merge edge cases. Likely declares `paths: internal/config/agents.go, internal/config/agents_test.go` and `packages: internal/config`. Must serialize against W4.D2 (or W4.D2's `agents.example.toml` rewrite tests will fail when the loader parses the new fixture).

**Severity:** high (acceptance criterion 5.12 `mage ci` green cannot be met if loader breaks on the new schema; 5.13 SQL-free dogfood end-to-end requires `agents.toml` to load cleanly).

---

## NITs

### NIT1 — Sub-plan vs direct-droplet count miscounted

**Location:** PLAN.md § "Sub-plan vs direct-droplet ratio + L2 spawn cadence" — first sentence: *"L1 emits 4 sub-plan containers (W1, W2, W3, W5, W6 — actually 5) and 4 direct droplets..."*

**Fix:** Replace "4 sub-plan containers (W1, W2, W3, W5, W6 — actually 5)" with "5 sub-plan containers (W1, W2, W3, W5, W6)".

### NIT2 — W4.D2 Packages declaration internally contradictory

**Location:** PLAN.md W4.D2 section, two consecutive bullets:
- `**Packages:** none (non-Go TOML files; no Go compile unit touched — ...)`
- `**Packages (Go compile):** internal/templates (embed.go update for till-fe.toml + agents.example.toml schema shift verification)`

**Fix:** Collapse to a single `**Packages:** internal/templates` line. The earlier "none" is contradicted by the immediate follow-up — `embed.go` is in `internal/templates` so the package IS compiled. The dispatcher's lock manager uses `packages` for serialization; the contradictory wording could confuse the L2 sub-planner.

### NIT3 — Acceptance criterion 5.13 (SQL-free dogfood end-to-end) has no explicit integration smoke droplet

**Location:** PLAN.md Acceptance criteria coverage map: *"5.13 SQL-free dogfood path end-to-end → W1+W2+W3+W4.D1 integration"*

**Evidence:** The integration is satisfied implicitly via the union of W1+W2+W3+W4.D1 acceptance bullets, but no droplet authors an explicit `mage test-integration` or end-to-end smoke test exercising `till init → till action_item create → till dispatcher run --dry-run`. REVISION_BRIEF §9.3 frames this as a hard dogfood-readiness gate.

**Fix:** Either (a) add an explicit smoke-test droplet to W3 (since W3 ships the final CLI surface for `till action_item create`) producing `cmd/till/dogfood_smoke_test.go` or similar; or (b) extend W3 acceptance to include the smoke test inline and require the L2 sub-planner to emit a droplet for it; or (c) explicitly defer to Drop 4c.7 (cascade wiring) and note in "Out-of-scope items" that 5.13's end-to-end pass is satisfied conceptually here but verified by drop 4c.7. Option (a) or (b) is preferable since 5.13 is in THIS drop's acceptance map.

**Severity:** medium-low — work is implied by per-wave acceptance bullets, but explicit smoke-test ownership prevents shipped-but-not-wired risk at the integration layer.

### NIT4 — W4.D1 borderline atomicity (~13 file operations)

**Location:** PLAN.md W4.D1 Paths block.

**Evidence:** Lists 5 deletes + 4 new QA files in `till-go/` + 1 delete + 2 splits + 4 new files in `till-gen/` + 9 new placeholder files in `fe/` + embed.go modify + embed_test.go modify = ~25 file ops. Per Drop 4c.6 sizing convention (1-4 code blocks, 80-120 LOC), this is high on file count but low on semantic code change ("structural file changes only — no semantic Go logic" per L1 table justification). Borderline.

**Fix:** Optional — could split into W4.D1a (delete `go-*` orphans + split `till-go/` qa files into 4), W4.D1b (split `till-gen/` qa files + delete `orchestrator-managed.md`), W4.D1c (add `fe/` 9-placeholder dir + extend `embed.go`). Each then is clearly atomic. OR accept current shape with the L1 table's justification (one mechanical structural pass). Either is defensible. **No blocking action required**; flag for L2 reconsideration if the builder finds the droplet too wide in practice.

### NIT5 — `## Hylla Feedback` section authored into PLAN.md committed artifact

**Location:** PLAN.md final section (lines 550–566).

**Evidence:** Drop 4c.6 PLAN.md (reference pattern) does NOT carry a Hylla Feedback section. Per `~/.claude/agents/go-planning-agent.md` § "Hylla Miss Reporting" + project CLAUDE.md § "Cascade Ledger + Hylla Feedback", Hylla misses belong in the planner's CLOSING COMMENT to the orchestrator, not in the committed PLAN.md artifact. The orchestrator aggregates closing comments at drop end into `workflow/drop_4c_6_1/CLOSEOUT.md` or `HYLLA_FEEDBACK.md`.

**Fix:** Move the `## Hylla Feedback` section out of `PLAN.md` and into the planner's orchestrator-facing closing response (or into a sibling `workflow/drop_4c_6_1/PLAN_HYLLA_FEEDBACK.md` if a separate file is preferred). Not blocking — content is useful and not harmful in PLAN.md; just inconsistent with the established pattern.

**Severity:** low (cosmetic / pattern conformance).

---

## Evidence Checked (per verification item)

1. **REVISION_BRIEF §2 coverage (2.1–2.16):**
   - 2.1 (HOME tier bake walker) → W1 ✓
   - 2.2 (group-aware resolver) → W1 ✓
   - 2.3 (multi-group `till init` flag + payload) → W2 ✓
   - 2.4 (`till init` writes `template.toml`) → W2 ✓
   - 2.5 (`CreateProjectWithMetadata` field population) → W2 ✓
   - 2.6 (TUI MCP confirm) → W2 ✓
   - 2.7 (template & agent save-flow CLIs) → W3 ✓
   - 2.8 (`till project update`) → W3 ✓
   - 2.9 (`till action_item create`) → W3 ✓
   - 2.10 (project lifecycle CLIs) → W3 ✓
   - 2.11 (agent set restructure: orphan delete + qa split + fe group) → W4.D1 ✓
   - 2.12 (agents.toml + template.toml schema shift) → W4.D2 **PARTIAL** — TOML files updated; the `agents.local.toml` deep-merge Go-side update (last bullet of §2.12) is NOT assigned. See **FF1**.
   - 2.13 (CLAUDE.md cascade-table corrections) → W7.D2 ✓
   - 2.14 (TUI components + style system) → W5 ✓
   - 2.15 (FE scaffold) → W6 ✓
   - 2.16 (delete `till serve`) → W7.D1 ✓

2. **Acceptance §5 coverage (5.1–5.13):** all 13 mapped in PLAN.md's "Acceptance criteria coverage map" table. 5.13 implicitly covered (see NIT3). 5.12 (`mage ci` green) depends on FF1 fix landing.

3. **`blocked_by` graph acyclicity:** walked the graph manually:
   - Wave A (parallel): W4.D1, W5, W6, W7.D1 — no internal blockers.
   - Wave B (after Wave A): W1 ← W4.D1; W4.D2 ← W4.D1; W7.D2 ← W4.D1 — parallel within Wave B; each references W4.D1 only.
   - Wave C: W2 ← {W1, W4.D1, W5} — W4.D1 already in Wave A so transitively satisfied; W1 and W5 must complete first.
   - Wave D: W3 ← {W2, W1} — W1 already in Wave B so transitively satisfied.
   - Topo-sort: W4.D1, W5, W6, W7.D1 → W1, W4.D2, W7.D2 → W2 → W3. **Acyclic.** Every blocked_by referent exists as a declared wave/droplet ID.

4. **Wave-grouping file-lock correctness:**
   - Wave A parallels (W4.D1, W5, W6, W7.D1): packages touched are `internal/templates` (W4.D1), `internal/tui/components`+`internal/tui/style` (W5, NEW packages), `fe/` (W6, NEW), `cmd/till` + `internal/adapters/server` (W7.D1). W7.D1's `cmd/till/main.go` edit + W4.D1's `internal/templates/embed.go` edit are disjoint. **No conflict.**
   - Wave B parallels (W1, W4.D2, W7.D2): W1 in `internal/app` + `internal/app/dispatcher/cli_claude/render`. W4.D2 in `internal/templates`. W7.D2 in `CLAUDE.md` (doc-only). **No conflict.**
   - W7.D1 (Wave A) and W2 (Wave C) both touch `cmd/till/main.go` — separated by the wave structure (Wave A complete before Wave C). Implicit but safe.
   - W3 (Wave D) touches `cmd/till/main.go` after W2 (Wave C) and W7.D1 (Wave A). Properly serialized.

5. **paths + packages per droplet:** every direct droplet declares both. Every sub-plan container declares expected paths + packages. W4.D2's Packages line is internally contradictory (see NIT2) but the L2/builder can infer.

6. **Migration markers in W5 + W6:**
   - W5 scope, acceptance, and L2 spawn directive all reference `// MIGRATION TARGET: github.com/hylla-org/lykta` as a hard requirement.
   - W6 scope, acceptance, and L2 spawn directive all reference `// MIGRATION TARGET: @hylla/stil-solid` as a hard requirement.
   - REVISION_BRIEF §10 + SKETCH §10 (EXTRACT-R1, EXTRACT-R2) constraint propagated correctly.

7. **No Tillsyn runtime references for drop-tracking:** PLAN.md uses CLI names (`till action_item create`, `till project update`) only as the work-product being BUILT in this drop, not as a tracking surface for the drop's own decomposition. Filesystem-MD-only paradigm preserved.

8. **Section 0 leakage check:** zero `# Section 0 — SEMI-FORMAL REASONING` headers or pass-titles (`## Planner`, `## Builder`, `## QA Proof`, `## QA Falsification`, `## Convergence`, `## Proposal`) in PLAN.md. The string "plan-qa-falsification" appears only as kind names, never as Section 0 pass titles. **Clean.**

9. **Refinements logged (8 expected per spawn prompt):** EXTRACT-R1 ✓, EXTRACT-R2 ✓, TILL-SERVE-R1 ✓, METHO-R1 ✓, A1-R1 ✓, D7-R5 ✓, FE-MOBILE-R1 ✓, FE-WEB-R1 ✓ — all present in PLAN.md § "Refinements to log at drop end" table.

10. **Drop 4c.6 pattern conformance:** PLAN.md header (State, Blocked by, Blocks, Paths, Packages, Workflow, Started), Scope section, Per-Wave Source-of-Truth, Planner section with decomposition shape table, per-wave sections with sub-plan containers / direct droplets, blocked_by graph summary, parallelism notes, Notes section, refinements table, out-of-scope list, acceptance coverage map — all conform to Drop 4c.6 L1 PLAN.md pattern. Section ordering matches.

11. **Sub-plan L2-spawn-directives:** every sub-plan container (W1, W2, W3, W5, W6) carries an `L2 sub-planner spawn directive` paragraph. Each names likely droplet shape (D1, D2, D3…), wires `blocked_by` recommendations between droplets sharing files/packages, calls out verification needs (LSP queries, Context7 checks, dependency-on-other-wave warnings). Lengths are appropriate — neither vague nor over-prescriptive. W2's directive correctly flags W2's hard dependency on W5 (TUI components must ship before W2's D2/D3 droplets can compile).

12. **Direct droplet atomicity (W4.D1, W4.D2, W7.D1, W7.D2):**
    - W7.D2 (CLAUDE.md doc-only) — atomic ✓
    - W7.D1 (delete `internal/adapters/server/` + 3 main.go-family edits) — atomic ✓ with documented critical-path verification gate
    - W4.D2 (4 TOML edits + new `till-fe.toml` + embed.go update) — atomic ✓
    - W4.D1 — borderline (see NIT4); justification stands at L1; L2 builder may split if needed.

---

## Hylla Feedback

**None — Hylla feedback for this QA pass is N/A.** Hylla today indexes Go only; this verification pass operated entirely against MD source-of-truth files (PLAN.md, REVISION_BRIEF.md, SKETCH.md, CLAUDE.md, prior-drop PLAN.md). No Go-symbol lookups required for the proof check. The PLAN.md planner DID record Hylla misses (enrichment-running errors) in its own `## Hylla Feedback` section — those should route via the planner's closing comment per NIT5.

---

## Notes

- Verdict gates on **FF1** (must-fix). Land an `internal/config/agents.go` schema-decoder update + tests as either a new W4.D3 droplet or a separate W4 sub-plan extension, then re-run plan-QA round 2.
- All 5 NITs are independently small. NIT1, NIT2, NIT5 are mechanical text/structure fixes. NIT3 + NIT4 deserve a brief planner judgement call but neither blocks the drop.
- Recommend the planner write `PLAN.md` Round 2 incorporating FF1 + NIT1 + NIT2 + NIT5 inline; NIT3 + NIT4 disposition (defer / accept / split) noted in Round 2's "Notes" section.
- Sibling QA pair (plan-qa-falsification) firing in parallel — both must pass before W4.D1 / W5 / W6 / W7.D1 (Wave A heads) can dispatch builders.

---

## Round 2 Verdict

**Drop:** `4c.6.1`
**Round:** 2 (plan-QA-proof)
**Reviewer:** go-qa-proof-agent
**Document under review:** `workflow/drop_4c_6_1/PLAN.md` (Round 2)
**Round 1 inputs verified against:** Round 1 PLAN_QA_PROOF.md (1 FF + 5 NITs), Round 1 PLAN_QA_FALSIFICATION.md (4 FFs + 6 NITs).
**Updated source-of-truth:** `REVISION_BRIEF.md` §2.3 / §2.9 / §2.11 / §2.12a (NEW) / §2.16 + refinements table; `SKETCH.md` §10 decisions.

### Pass / Fail

**PASS** — all 5 Round 1 must-fix findings (FF1 till mcp split, FF2 FLAT fail-loud, FF3 orchestrator-managed kept, FF4 --structural-type smart-default, proof-FF1 W0 config decoder) absorbed correctly into Round 2 PLAN.md. 9 of 11 NITs absorbed cleanly. Two residual NIT-level cosmetic findings recorded below (acceptance-map flag-name drift; ORCH-MANAGED-R1 missing from refinements-to-log table). Neither blocks Wave A dispatch.

### Findings

#### NIT1 [Axis: spec-conformance] [severity: low] — Acceptance criteria coverage map uses `--groups` (plural) while wave scope uses `--group` (singular, repeated)

**Location:** PLAN.md lines 654-655 (acceptance map): rows 5.1 + 5.2 read `till init --groups go` and `till init --groups go,fe`. W2 scope line 149 + acceptance lines 159-160 + Round 2 Changes line 21 all use `--group` (singular, repeated). Round 1 falsification NIT2 explicitly flagged this drift, and Round 2 Changes line 21 claims it is resolved consistently — partial absorption only; the coverage map text still uses the plural form inherited from REVISION_BRIEF §5.1.

**Fix:** Rewrite the two acceptance-map rows to `till init --group go` (5.1) and `till init --group go --group fe` (5.2) to match W2 scope + L2 directive. Also update REVISION_BRIEF §5.1 in the same pass for source-of-truth coherence.

**Severity:** low (text drift only; the L2 sub-planner inherits the correct singular-repeated shape from W2 scope, which is the canonical spec — the coverage-map drift cannot mis-direct a builder into shipping the wrong CLI surface, but it remains an audit-completeness gap).

#### NIT2 [Axis: completion-checklist-audit] [severity: low] — ORCH-MANAGED-R1 refinement listed in Notes/locked decisions but missing from "Refinements to log at drop end" table

**Location:** PLAN.md line 603 (locked architectural decisions) + W4.D1 RiskNotes line 283 both reference `ORCH-MANAGED-R1`. The refinements-to-log table at lines 674-684 enumerates 9 refinements (EXTRACT-R1, EXTRACT-R2, TILL-SERVE-R1, METHO-R1, A1-R1, D7-R5, D7-R6, FE-MOBILE-R1, FE-WEB-R1) — but no `ORCH-MANAGED-R1` row. REVISION_BRIEF §4 line 294 has it explicitly tracked. Drop closeout will miss logging ORCH-MANAGED-R1 unless this table is the canonical drop-end source.

**Fix:** Add a row to PLAN.md's "Refinements to log at drop end" table: `| ORCH-MANAGED-R1 | Split orchestrator-managed.md into role-specific agents (closeout-agent, refinement-agent, discussion-agent, human-verify-agent) in Drop 4c.8 prompt-authoring |`.

**Severity:** low (the refinement is tracked in REVISION_BRIEF §4 and the planner's notes; risk is purely a drop-end logging gap that the closeout aggregator catches).

### Evidence Checked (per verification item)

1. **`**Round:** 2` marker present:** PLAN.md line 4. **VERIFIED.**

2. **`## Round 2 Changes` section present with all 5 FFs + 11 NITs cataloged:** PLAN.md lines 12-21. All 5 FFs + 11 NITs explicitly enumerated. **VERIFIED.**

3. **FF1 (till serve / till mcp split):**
   - W7.D1 (lines 445-477) creates `internal/adapters/mcp_stdio/` (NEW package), moves `RunStdio` + helpers from `server.go`, updates `cmd/till/main.go` import. Wave A head, no blockers.
   - W7.D2 (lines 479-509) deletes remaining `internal/adapters/server/` + `till serve` cobra subcommand + tests; `blocked_by: 4c.6.1.W7.D1` (line 496).
   - W7.D3 (lines 511-537) renumbers CLAUDE.md cascade-table update; `blocked_by: 4c.6.1.W4.D1` (line 524).
   - Package-lock correctness: W7.D1 creates NEW `internal/adapters/mcp_stdio` package + modifies `internal/adapters/server` + modifies `cmd/till`. W7.D2 deletes `internal/adapters/server` + modifies `cmd/till`. Both touch `cmd/till/main.go` and `internal/adapters/server` — serialized via W7.D2 blocked_by W7.D1. **VERIFIED.**
   - Acceptance invariant "`till mcp` STILL WORKS" present in both W7.D1 (line 459) and W7.D2 (line 494). **VERIFIED.**

4. **FF2 (FLAT migration fail-loud):**
   - W2 scope line 150: explicit FAIL-LOUD path with error message `"FLAT agent layout detected at <project>/.tillsyn/agents/. Remove it and re-run: rm -rf <project>/.tillsyn/agents && till init --group <group>"` + NO migration code.
   - W2 scope line 151: old-schema `agents.toml` fail-loud detection (`[agents]` prefix check) + same remediation pattern.
   - W2 acceptance lines 161-162: both detection paths declared as test-acceptable behavior.
   - W2 L2 directive line 170: D2 droplet ownership of both fail-loud paths confirmed.
   - REVISION_BRIEF §2.3 line 63 + SKETCH §10 line 273 reinforce no-migration / fail-loud disposition. **VERIFIED.**

5. **FF3 (orchestrator-managed.md kept):**
   - W4.D1 paths line 255: `till-go/orchestrator-managed.md` ADD if absent (verify via Read first).
   - W4.D1 paths line 262: `till-gen/orchestrator-managed.md` KEEP — do NOT delete; FF3 disposition.
   - W4.D1 paths line 263: `fe/` ADD with 10 placeholder files (9 standard + orchestrator-managed.md).
   - W4.D1 acceptance lines 269-271: all three groups final at "10 files" (9 standard + orchestrator-managed).
   - W4.D1 RiskNotes lines 282-284 + ContextBlocks lines 292 reinforce FF3 disposition.
   - W4.D2 acceptance line 311 + RiskNotes line 329 + ContextBlocks line 334: 4 orchestrator-managed bindings (closeout/refinement/discussion/human-verify) continue to reference `orchestrator-managed` (NO `-agent` suffix). **VERIFIED.**

6. **FF4 (--structural-type smart-default):**
   - W3 scope lines 207-214 enumerate smart-default mapping: `plan`/`refinement` → `segment`; other 10 kinds → `droplet`. Override flag validates against closed enum `drop|segment|confluence|droplet`. Help text documents mapping.
   - W3 acceptance lines 226-228: `--kind plan` defaults to `segment`; `--kind build` defaults to `droplet`; invalid `--structural-type` fails with clear error.
   - REVISION_BRIEF §2.9 lines 114-122 source-of-truth match. **VERIFIED.**

7. **Proof-FF1 (W0 config decoder multi-group):**
   - NEW Wave W0 created (lines 73-105) as direct droplet `4c.6.1.W0`.
   - Paths: `internal/config/agents.go`, `internal/config/agents_test.go`, `internal/config/testdata/` (lines 79-83).
   - Packages: `internal/config` (line 84).
   - Acceptance: `AgentsRegistry` struct supports `map[group]GroupConfig`; `Resolve(registry, group, kind)` new signature; `Merge` deep-merge per-group; 5 golden-fixture test cases (a-e); `mage test-pkg ./internal/config` + `mage ci` green (lines 85-89).
   - W0 Wave A head, no blockers (line 90).
   - W4.D2 blocked_by includes W0 (line 319). **VERIFIED.**

8. **NIT absorption pass (all 11):**
   - Proof NIT1 (sub-plan count): line 588 reads "5 sub-plan containers (W1, W2, W3, W5, W6) and 6 direct droplets." **VERIFIED.**
   - Proof NIT2 (W4.D2 Packages collapsed): line 309 single `**Packages:** internal/templates`. **VERIFIED.**
   - Proof NIT3 (5.13 deferred to 4c.7): lines 618-620 + acceptance map line 666. **VERIFIED.**
   - Proof NIT4 (W4.D1 atomicity): lines 622-624 explicit note + L2 escape hatch. **VERIFIED.**
   - Proof NIT5 (Hylla Feedback removed from PLAN.md): grep PLAN.md — no `## Hylla Feedback` section present. **VERIFIED.**
   - Falsification NIT1 (-agent suffix): W4.D2 acceptance line 311 + KindPayload line 336. **VERIFIED.**
   - Falsification NIT2 (--group singular repeated): W2 scope + acceptance + L2 directive all consistent. Coverage map line 654-655 inconsistent (NIT1 above). **PARTIAL.**
   - Falsification NIT3 (agents.toml re-init detection): W2 scope line 151 + L2 directive line 170. **VERIFIED.**
   - Falsification NIT4 (~/.claude/agents/ split deferred): lines 630-632 explicit out-of-scope note. **VERIFIED.**
   - Falsification NIT5 (mage ci skipping fe/): W6 scope line 420 + acceptance line 436 + Notes line 616 + W6 L2 directive line 439. **VERIFIED.**
   - Falsification NIT6 (`## Planner` heading renamed): line 54 reads `## Per-Wave Plans`. **VERIFIED.**

9. **Acyclic graph check (W0 + restructured W7):**
   - Wave A (parallel, no blockers): W0, W4.D1, W5, W6, W7.D1.
   - Wave B (after Wave A): W1 ← W4.D1; W4.D2 ← W4.D1 + W0; W7.D2 ← W7.D1; W7.D3 ← W4.D1.
   - Wave C: W2 ← W1 + W4.D1 + W5.
   - Wave D: W3 ← W2 + W1.
   - Topo-sort: {W0, W4.D1, W5, W6, W7.D1} → {W1, W4.D2, W7.D2, W7.D3} → W2 → W3. **No cycle confirmed.**
   - Every blocked_by referent exists as declared wave/droplet ID. **VERIFIED.**

10. **W7 restructure file-lock correctness:**
    - W7.D1 NEW `internal/adapters/mcp_stdio/` package + MODIFY `internal/adapters/server/` (remove RunStdio) + MODIFY `cmd/till/main.go` (update import).
    - W7.D2 DELETE `internal/adapters/server/` remainder + MODIFY `cmd/till/main.go` (remove serve registration + server imports) + MODIFY `cmd/till/main_test.go` + MODIFY `cmd/till/help.go`.
    - Both touch `cmd/till/main.go` and `internal/adapters/server/` — but W7.D2 blocked_by W7.D1 serializes (line 496).
    - W7.D3 doc-only (`CLAUDE.md`), no Go compile, blocked_by W4.D1 (line 524).
    - W4.D1 (Wave A) and W7.D2 (Wave B) both modify `internal/templates/embed.go` and `cmd/till/main.go` respectively — disjoint packages.
    - No parallel file-lock conflict introduced. **VERIFIED.**

11. **REVISION_BRIEF §2.1–2.16 + §2.12a wave mapping:**
    - 2.1 → W1; 2.2 → W1; 2.3 → W2; 2.4 → W2; 2.5 → W2; 2.6 → W2; 2.7 → W3; 2.8 → W3; 2.9 → W3 (FF4); 2.10 → W3; 2.11 → W4.D1 (FF3); 2.12 → W4.D2; 2.12a → W0 (NEW, proof-FF1); 2.13 → W7.D3 (renumbered); 2.14 → W5; 2.15 → W6; 2.16 → W7.D1 + W7.D2 (FF1 split). All mapped per "Per-Wave Source-of-Truth" section (lines 45-52). **VERIFIED.**

12. **Acceptance §5.1–5.13 coverage:**
    - 5.1–5.12 mapped to waves in PLAN.md acceptance coverage map (lines 652-666).
    - 5.13 explicitly deferred to Drop 4c.7 with rationale per dev disposition (line 666). Acceptable per spawn prompt. **VERIFIED.**

13. **No Tillsyn-runtime references for drop tracking:** PLAN.md uses `till action_item create`, `till project update`, etc. only as work-products being built in W3, not as a tracking surface for the drop's decomposition. Filesystem-MD-only paradigm preserved. **VERIFIED.**

14. **Section 0 leakage check:** zero `# Section 0 — SEMI-FORMAL REASONING` headers in PLAN.md. No `## Proposal` / `## Builder` / `## QA Proof` / `## QA Falsification` / `## Convergence` pass titles present. Line 54 renamed to `## Per-Wave Plans` per Round-1 falsification NIT6. **CLEAN.**

15. **paths + packages per droplet:** every direct droplet (W0, W4.D1, W4.D2, W7.D1, W7.D2, W7.D3) declares both. Every sub-plan container (W1, W2, W3, W5, W6) declares expected paths + packages. **VERIFIED.**

16. **Migration markers (W5 + W6):** W5 scope + acceptance + L2 directive cite `// MIGRATION TARGET: github.com/hylla-org/lykta` as hard requirement; W6 same for `// MIGRATION TARGET: @hylla/stil-solid`. **VERIFIED.**

### Notes

- Round 2 PLAN.md is a clean absorption of all 5 must-fix dispositions. The single PASS-gating concern from Round 1 (proof-FF1 — `internal/config/agents.go` decoder unassigned) is now an entire dedicated wave (W0) at Wave A head with explicit blocked_by from W4.D2.
- The W7 two-step refactor-then-delete is correctly sequenced (W7.D1 Wave A; W7.D2 Wave B blocked_by W7.D1) with explicit `till mcp` STILL WORKS invariant on both droplets.
- Two NIT-level findings remain (NIT1 acceptance-map flag-name drift; NIT2 ORCH-MANAGED-R1 missing from refinements table). Both cosmetic; non-blocking; the L1→L2 contract is intact via W2 scope and Notes section respectively.
- Sibling plan-QA-falsification round 2 firing in parallel; both must pass before Wave A heads (W0 / W4.D1 / W5 / W6 / W7.D1) dispatch builders.

---

## Round 3 Verdict

**Drop:** `4c.6.1`
**Round:** 3 (plan-QA-proof)
**Reviewer:** go-qa-proof-agent
**Document under review:** `workflow/drop_4c_6_1/PLAN.md` (Round 3)
**Round 2 inputs verified against:** Round 2 PLAN_QA_PROOF.md (PASS — 2 cosmetic NITs), Round 2 PLAN_QA_FALSIFICATION.md (FAIL — 1 critical R2-FF1 + 3 NITs).
**Updated source-of-truth:** `REVISION_BRIEF.md` §2.14, §2.15, §2.16 (updated); §2.17, §2.18, §2.19, §2.20 (NEW); refinements table extended with 11 entries (BOOTSTRAP-R1, KEYBIND-R1/R2/R3, BIND-CONSIST-R1, NATIVE-MENU-R1, QA-SPLIT-R1, EMBED-PROMPTS-R1, CASCADE-WIRING-R1, FE-MOBILE-R1, FE-WEB-R1). `SKETCH.md` §10 extended with 10 new rows.

### Pass / Fail

**PASS** — R2-FF1 (W7.D1 dual extraction of `mcp_stdio/` + `mcp_common/`), R2-NIT1 (`mage ci-fe` wording), R2-NIT2 (Playwright in W6 L2), R2-NIT3 (CONSUMER-TIE in W2 L2), R1-NIT1 (`--group` singular in acceptance map), R1-NIT2 (ORCH-MANAGED-R1 row), and all four new scope expansions (W8 sub-plan, `till agents bootstrap`, W5 vim keybinding dispatcher, W6 vim engine + native menu) absorbed correctly into Round 3 PLAN.md. Topo-sort with W8 added remains acyclic; W8 file-lock disjoint from all other waves. 11 new refinements present. One cosmetic NIT recorded below (W8 path prefix POV). Non-blocking.

### Findings

#### R3-NIT1 [Axis: spec-conformance] [severity: low] — W8 paths use `tillsyn/main/.tillsyn/...` bare-root POV; from working-dir `tillsyn/main/` the actual paths are `.tillsyn/...`

**Location:** PLAN.md W8 Paths (lines 614-636). Every entry prefixed `tillsyn/main/.tillsyn/...`.

**Evidence:** The orchestrator + builders operate from the working directory `tillsyn/main/` (CLAUDE.md preamble: "This file lives in the `main/` worktree at `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`. ... real coding ... happens here. Drop orchs whose scope is the `main` branch launch from this directory."). From this CWD, the prompt files resolve to `.tillsyn/agents/go/builder-agent.md`, NOT `tillsyn/main/.tillsyn/agents/go/builder-agent.md`. The latter prefix is the bare-root view. The acceptance bullet on line 670 says "`git ls-files tillsyn/main/.tillsyn/agents/` shows 20 tracked files" — this command FAILS from inside `tillsyn/main/` (no such directory `tillsyn/main/`), but works from one directory up.

**Impact:** L2 sub-planner + builders reading Paths verbatim may try to create files at `tillsyn/main/.tillsyn/agents/...` from inside the `tillsyn/main/` CWD, producing a nested `tillsyn/main/tillsyn/main/.tillsyn/...` mistake. Mechanical fix; L2 can infer, but inconsistent with how every other wave's paths are recorded (W1-W7 all use repo-root-relative paths like `internal/...`, `cmd/till/...`, `fe/...`).

**Fix:** Strip the `tillsyn/main/` prefix from every W8 Paths entry (lines 614-636) and the acceptance bullet's `git ls-files` invocation (line 670). Use `.tillsyn/agents/go/builder-agent.md`, etc. Mirror this fix into REVISION_BRIEF.md §2.18 if the same prefix appears there.

**Severity:** low (cosmetic / path-resolution ambiguity; L2 builder can infer; non-blocking).

### Evidence Checked (per verification item)

1. **`**Round:** 3` marker present:** PLAN.md line 4. **VERIFIED.**

2. **`## Round 3 Changes` section present with all dispositions cataloged:** PLAN.md lines 12-23. R2-FF1, W8, W3 bootstrap, W5 keybindings, W6 vim engine + native menu, R2-NITs absorbed, 11 new refinements added, L1 shape updated (6 sub-plan + 6 direct = 12 nodes). **VERIFIED.**

3. **R2-FF1 absorption — W7.D1 dual extraction:**
   - Paths (lines 500-508): NEW `internal/adapters/mcp_stdio/` package (stdio.go + stdio_test.go); NEW `internal/adapters/mcp_common/` package (adapter.go + adapter_test.go); MODIFY `internal/adapters/server/server.go` (remove RunStdio + common/ uses); MODIFY `cmd/till/main.go` (update ALL servercommon callers at `:81-82`, `:2682`, `:2763-2764`); MODIFY `cmd/till/main_test.go` (12+ test sites).
   - Packages line 509: `mcp_stdio` (NEW), `mcp_common` (NEW), `internal/adapters/server`, `cmd/till`.
   - Acceptance (lines 511-520): both new packages exist; cmd/till has NO `servercommon` imports; `till mcp` works; `till capture-state` works; auth-mutation tests pass.
   - RiskNotes (line 527) cite R2-FF1 root cause + `cmd/till/main_test.go` ~1400+ lines + `git grep -n "servercommon\."` enumeration discipline.
   - KindPayload (line 538) covers both packages + 3 production + 12+ test sites. **VERIFIED.**

4. **R2-FF1 corollary — W7.D2 narrowed to HTTP residue only:**
   - Paths line 546: "DELETE remaining HTTP-only residue — `httpapi/`, HTTP-specific bits of `server.go`, `Run()` HTTP handler, any remaining HTTP test files; `common/` and stdio code already extracted by W7.D1 — do NOT re-delete those."
   - Acceptance line 551: "`internal/adapters/server/` directory does not exist post-build (all HTTP residue deleted)."
   - Pattern guidance (line 565): "Specify the RESIDUE to delete, not the whole directory." Lesson from R2-FF1 absorbed.
   - KindPayload action="delete" with shape_hint "git rm -r the HTTP-only residue after W7.D1 extractions; read directory first to confirm scope". **VERIFIED.**

5. **W8 sub-plan absorption — TILLSYN_PROJECT_AGENT_PROMPTS:**
   - Sub-plan container (line 612), Wave A entry, no blockers (line 672).
   - Paths (lines 614-636): 20 prompt files (10 go + 10 fe) + `.tillsyn/bindings.json` + `.gitignore` re-include.
   - Skip `gen/` group per disposition 7.6 (line 641).
   - Source material listed (lines 644-649): `~/.claude/agents/<group>-<role>-agent.md`, CLAUDE.md, WORKFLOW.md, WIKI.md, 9 memory entries.
   - Acceptance (lines 663-671): >= 1000 chars body, frontmatter complete (name + description + model + tools), role discipline encoded, post-render validator passes, no Section 0 leakage, bindings.json with 6 v1 commands, .gitignore re-includes.
   - L2 spawn directive (line 674): D0 .gitignore + bindings.json FIRST; D1-D8 go group; D9-D18 fe group; per-droplet QA pair (Proof verifies; Falsification attacks); source = `~/.claude/agents/` adapt; model assignments per cascade-model-policy; tools per role. **VERIFIED.**

6. **W3 expansion — `till agents bootstrap`:**
   - Scope text (line 213) describes `bootstrap --from <path> [--to <path>] [--dry-run] [--force]`; 2-into-4 QA fan-out explicit; group-agnostic agents copied to each known group dir; missing files reported; orchestrator-managed.md starter generated. Per §2.17.
   - Acceptance (line 239-240): `--dry-run` prints copy plan without writing; actual copy with 2-into-4 fan-out + missing-file reporting + orchestrator-managed.md starter.
   - L2 directive (line 251): D6 = bootstrap CLI; tests cover dry-run + actual copy + fan-out + missing-file reporting via CONSUMER-TIE pattern.
   - CONSUMER-TIE TEST CONTRACT explicit (line 251 end). **VERIFIED.**

7. **W5 expansion — vim keybinding dispatcher:**
   - Paths (lines 375-378): `internal/tui/keybindings/dispatcher.go,loader.go,modes.go,dispatcher_test.go`.
   - Package declared line 379: `internal/tui/keybindings` (NEW).
   - Migration marker (line 381, 402): `// MIGRATION TARGET: github.com/hylla-org/lykta` co-extracts with components + style.
   - Scope (lines 396-402): consumes stil `baseline.json` + Tillsyn-local `.tillsyn/bindings.json`; mode state machine (nav / insert / visual / visual-block / command / hint); per-binding dispatch.
   - Acceptance line 411: `mage test-pkg ./internal/tui/keybindings` passes.
   - L2 directive (line 414): D6 keybinding dispatcher; loader handles graceful fallback when `.tillsyn/bindings.json` absent. **VERIFIED.**

8. **W6 expansion — vim engine + wails-keys + palette + native menu + Playwright + mage ci-fe:**
   - Vim engine paths (lines 437-440): `fe/frontend/src/lib/vim/{engine.ts,types.ts,wails-keys.ts,palette.ts}`.
   - Migration markers (lines 456-459): `// MIGRATION TARGET: github.com/hylla-org/ro-vim` on engine.ts, wails-keys.ts, palette.ts.
   - Native menu: line 426 `fe/main.go` "DEFAULT NATIVE MENU"; line 463 "DEFAULT Wails v2 native menu (Quit, About, Hide, Minimize, Window controls). No custom menu items in v1." NATIVE-MENU-R1 tracks future integration.
   - Playwright: line 461 explicit "Playwright (via MCP `mcp__plugin_playwright_playwright__*`) for end-to-end keybinding tests"; line 489 same.
   - `mage ci-fe`: line 465 "IS ADDED to `magefile.go` in W6 (R2-NIT1 resolution — 'added' is authoritative; exact scope L2-decided)"; out-of-scope line 788 same. Single-decision wording. **VERIFIED.**

9. **W2 L2 directive — CONSUMER-TIE explicit:** Line 185 end of W2 L2 directive: "CONSUMER-TIE TEST CONTRACT (R2-NIT3): tests invoke `run(ctx, args, &out, io.Discard)` end-to-end — flow-level assertions, not unit assertions on internal helpers. All D-series droplets sharing `init_cmd.go` follow this pattern." **VERIFIED.**

10. **Acceptance map `--group` singular (R1-NIT1 absorption):** Lines 794-795: "5.1 `till init --group go` dispatcher-ready"; "5.2 `till init --group go --group fe` multi-group". Plural drift fixed. **VERIFIED.**

11. **Refinements table — ORCH-MANAGED-R1 + 9 new refinements:** Lines 816-834 (19 rows total). Inherited 9 from Round 2: EXTRACT-R1, EXTRACT-R2, TILL-SERVE-R1, METHO-R1, A1-R1, D7-R5, D7-R6, FE-MOBILE-R1, FE-WEB-R1. New Round 3 entries: ORCH-MANAGED-R1 (line 825 — R1-NIT2 absorption), BOOTSTRAP-R1 (826), KEYBIND-R1 (827), KEYBIND-R2 (828), KEYBIND-R3 (829), BIND-CONSIST-R1 (830), NATIVE-MENU-R1 (831), QA-SPLIT-R1 (832), EMBED-PROMPTS-R1 (833), CASCADE-WIRING-R1 (834). All 10 new entries present (9 net-new + ORCH-MANAGED-R1 absorption). **VERIFIED.**

12. **Topo-sort acyclic with W8 added:**
    - Wave A (parallel, no blockers): W0, W4.D1, W5, W6, W7.D1, **W8**.
    - Wave B (after Wave A): W1←{W4.D1}; W4.D2←{W4.D1, W0}; W7.D2←{W7.D1}; W7.D3←{W4.D1}.
    - Wave C: W2←{W1, W4.D1, W5}.
    - Wave D: W3←{W2, W1}.
    - Topo-sort line 707: {W0, W4.D1, W5, W6, W7.D1, W8} → {W1, W4.D2, W7.D2, W7.D3} → W2 → W3. **No cycle confirmed.**
    - W8 has no downstream blockers (line 707-713). **VERIFIED.**

13. **W8 file-lock disjoint:** W8 paths are `.tillsyn/agents/...` + `.tillsyn/bindings.json` + `.gitignore`. None of W0-W7 declare `.tillsyn/` or `.gitignore` in paths. W8's parallelism with the other 5 Wave A nodes is safe. **VERIFIED** (modulo R3-NIT1 path-prefix cosmetic).

14. **Source-of-truth pointers updated for §2.17–§2.20:**
    - Line 51 (scope summary): "REVISION_BRIEF §2 (subsections 2.1–2.20 + 2.12a)".
    - Line 61: W3 → §2.7–2.10 + §2.17.
    - Line 63: W5 → §2.14 (updated for keybindings).
    - Line 64: W6 → §2.15 (updated for vim engine).
    - Line 66: W8 → §2.18 + §2.19 + §2.20. **VERIFIED.**

15. **No Section 0 leakage:** `## Planner` heading replaced by `## Per-Wave Plans` (line 68 — kept from Round 2). No `# Section 0 — SEMI-FORMAL REASONING` headers in PLAN.md. No `## Proposal` / `## Builder` / `## QA Proof` / `## QA Falsification` / `## Convergence` pass titles. Line 810 mentions Section 0 only meta-commentarily ("Section 0 stays in the orchestrator-facing response — never in PLAN.md or QA files"). **CLEAN.**

16. **REVISION_BRIEF §2 + §5 wave mapping updated for §2.17–§2.20:**
    - §2.17 (`till agents bootstrap`) → W3 (line 61).
    - §2.18 (Tillsyn-project-local agent prompts) → W8 (line 66).
    - §2.19 (Tillsyn-local bindings.json) → W8 (line 66).
    - §2.20 (.gitignore re-includes) → W8 (line 66).
    - Pre-existing §2.1-2.16 mappings unchanged from Round 2. **VERIFIED.**

17. **L2 sub-planner spawn directives — prompt-authoring acceptance (W8):**
    - >= 1000 chars body (line 663).
    - Frontmatter complete with name + description + model + tools (line 664).
    - Role discipline encoded — mage targets, Section 0, plan-down/build-up, etc. (line 665).
    - Post-render validator (Drop 4c.6 W3.D5 reference — line 666).
    - No Section 0 leakage in committed prompt files (line 667).
    - Per-droplet QA pair: Proof verifies; Falsification attacks (line 674 end: "can a builder reading this prompt go wrong despite following it?"). **VERIFIED.**

18. **Sub-plan vs direct-droplet count (line 724):** "L1 emits **6 sub-plan containers** (W1, W2, W3, W5, W6, W8) and **6 direct droplets** (W0, W4.D1, W4.D2, W7.D1, W7.D2, W7.D3) = 12 L1 nodes." Decomposition shape table (lines 74-85) shows W0 direct, W1-W3 sub-plan, W4 direct droplets (2), W5-W6 sub-plan, W7 direct droplets (3), W8 sub-plan. Count consistent. **VERIFIED.**

19. **paths + packages per droplet:** every direct droplet (W0, W4.D1, W4.D2, W7.D1, W7.D2, W7.D3) declares both. Every sub-plan container (W1, W2, W3, W5, W6, W8) declares expected paths + packages (W8 packages declared as "none" — line 637, since all paths are non-Go). **VERIFIED.**

20. **Migration markers (W5 + W6):**
    - W5 components + style → `// MIGRATION TARGET: github.com/hylla-org/lykta` (line 381, 402).
    - W5 keybinding dispatcher → same target (line 402).
    - W6 FE components → `// MIGRATION TARGET: @hylla/stil-solid` (line 444).
    - W6 vim engine → `// MIGRATION TARGET: github.com/hylla-org/ro-vim` (lines 456, 458, 459). **VERIFIED.**

21. **Per-droplet QA expected (per spawn-prompt requirement for W8):** L2 directive line 674 explicit: "Per-droplet QA pair (build-qa-proof + build-qa-falsification) runs after each prompt droplet: Proof verifies >= 1000 chars + frontmatter complete + role discipline encoded; Falsification attacks: can a builder reading this prompt go wrong despite following it?" **VERIFIED.**

### Notes

- Round 3 PLAN.md is a clean absorption of R2-FF1 + R2-NITs 1/2/3 + R1 residual NITs 1/2 + 4 net-new scope expansions (W8, `till agents bootstrap`, W5 keybindings, W6 vim engine + native menu + Playwright + mage ci-fe). The R2-FF1 W7.D1 dual extraction is correctly executed: `mcp_stdio/` AND `mcp_common/` packages declared, all 3 production call sites + 12+ test sites enumerated in KindPayload, `till mcp` + `till capture-state` + auth-mutation tests preserved as critical-path invariants.
- W7.D2 deletion target narrowed to HTTP residue only — pattern lesson from R2-FF1 ("Specify the RESIDUE to delete, not the whole directory") absorbed in W7.D2 RiskNotes.
- W8 is a Wave A workstream fully disjoint from all other paths; the new file-lock surface (`.tillsyn/agents/`, `.tillsyn/bindings.json`, `.gitignore`) does not conflict with W0-W7.
- Single residual cosmetic NIT (R3-NIT1: W8 path prefix `tillsyn/main/.tillsyn/...` reads as bare-root POV; from the working-dir `tillsyn/main/` perspective should be `.tillsyn/...`). Non-blocking; the L2 sub-planner can infer; cosmetic consistency fix.
- Sibling plan-QA-falsification round 3 firing in parallel — both must pass before Wave A heads (W0, W4.D1, W5, W6, W7.D1, **W8**) dispatch builders.

---

## Round 5 — Plan-QA-Proof Verdict

**Drop:** `4c.6.1`
**Round:** 5 (plan-QA-proof)
**Pass / Fail:** **PASS**

### Findings

Round 5 is a surgical absorption pass for R3-NIT2 / R3-NIT3 / R3-NIT4 per the new "all NITs first-class" discipline (memory `feedback_nits_are_first_class.md`). All three findings carry explicit dispositions; no silent drops, no "judgment call" language. New process-change note binds future rounds to the same discipline.

- **R3-NIT2 (W7.D1 atomicity split)** — **ABSORB-AS-MOOT.** Disposition at PLAN.md line 16 cites W7's full 4-droplet restructure (Round 4): "The new W7.D1 is pure-read INVENTORY (no extraction), so the NIT2 split suggestion is void." Reason is non-circular — it cites a structural change already in tree (lines 521-636).
- **R3-NIT3 (W8 sub-plan decomposition shape)** — **DEFERRED-AS-NIT.** Disposition at PLAN.md line 17 cites falsifier's own verdict ("either shape is defensible," severity: low), preserves flat-19 at L1, grants L2 the option to split into W8.go + W8.fe internally. Tracked via new refinement row **W8-DECOMP-R1** at line 921. Reason is independent (the falsifier herself ruled it defensible); not a tautology.
- **R3-NIT4 (integration smoke test for prompt rendering)** — **ABSORB.** Disposition at PLAN.md line 18 + new W8 acceptance bullet at line 733 ("Integration smoke (R3-NIT4 absorption)") + new W8 L2 spawn-directive paragraph at line 751 ("R3-NIT4 smoke test requirement (REQUIRED in W8 L2)"). Smoke test scope is **unit-test only** — renders one W8-authored prompt through `internal/app/dispatcher/cli_claude/render/render.go:assembleAgentFileBody` with project-tier override and asserts the rendered body matches the W8-authored file (NOT the embedded default). Full end-to-end dispatcher smoke remains deferred to Drop 4c.7 §5.13, tracked via new refinement row **W8-SMOKE-R1** at line 922.
- **Process change (new rule for future rounds)** — captured at PLAN.md line 20: "Future plan-QA + build-QA rounds enumerate every finding (FF AND NIT) as ABSORB or DEFERRED-AS-NIT-with-reason. No 'judgment call' / 'as-is' / 'accepted' language without explicit absorb/defer disposition + reason." Sits at the bottom of the newest Round 5 Changes block — first thing future rounds read top-down.

No new findings raised in Round 5. No residual NITs from Round 4 left undisposed.

### Evidence checked

1. **Round marker** — PLAN.md line 4: `**Round:** 5`.
2. **Round 5 Changes section position** — present at lines 12-20, BEFORE Round 4 Changes (line 22). Newest-first ordering preserved.
3. **Round 4 Changes preserved verbatim** — lines 22-38 intact; round-4 R3-NIT2/NIT3/NIT4 entries (lines 33-35) preserved with their original "moot / unchanged / deferred" phrasing (which Round 5 surgically supersedes — old phrasings are not retroactively edited).
4. **R3-NIT2 MOOT disposition** — PLAN.md line 16; reason cites W7 restructure (W7.D1 = pure-read INVENTORY, no extraction → split suggestion void). Cross-checked against W7.D1 definition at lines 525-560 (pure-read; `Packages: none`; `mage ci` trivially green).
5. **R3-NIT3 DEFERRED-AS-NIT disposition** — PLAN.md line 17; reason cites falsifier verdict ("either shape is defensible," severity: low). W8-DECOMP-R1 refinement row at line 921.
6. **R3-NIT4 ABSORB disposition** — PLAN.md line 18; three downstream artifacts:
   - **W8 acceptance bullet** — line 733: "Integration smoke (R3-NIT4 absorption): at least one W8-authored prompt (e.g., `.tillsyn/agents/go/builder-agent.md`) is rendered through `internal/app/dispatcher/cli_claude/render/render.go:assembleAgentFileBody` with project-tier override, producing a body identical to the W8-authored file (NOT the embedded default). This is a unit test, NOT a full dispatch ..."
   - **W8 L2 spawn directive** — line 751: "R3-NIT4 smoke test requirement (REQUIRED in W8 L2): The LAST prompt droplet in the W8 L2 decomposition MUST include an integration smoke unit test ..." Explicit "last prompt droplet OR new test file in that package" satisfies the spawn-prompt's "designating the last prompt droplet OR a dedicated droplet" requirement.
   - **W8-SMOKE-R1 refinement row** — line 922: "Integration smoke only verifies ONE prompt's 3-tier resolver pickup (W8 acceptance bullet). Full end-to-end smoke (`till dispatcher run --dry-run`) is deferred to Drop 4c.7 acceptance §5.13 per round-2 dev disposition."
7. **Process change note** — line 20: explicit binding on all future plan-QA + build-QA rounds; "every finding (FF AND NIT) as ABSORB or DEFERRED-AS-NIT-with-reason."
8. **Refinements table** — both new rows present (lines 921-922); existing rows intact (902-920).
9. **No other PLAN.md sections modified beyond Round 5 surgical absorption points** — W0–W7 wave bodies unchanged; W8's wave body changed ONLY at the new acceptance bullet (line 733) and the L2 directive's R3-NIT4 paragraph (line 751). All other text in W8 (paths, scope, source material, bindings.json semantics, .gitignore re-includes, R3-NIT5 guidance) preserved from Round 4.
10. **Blocked-by graph acyclic and unchanged** — lines 753-794 identical to Round 4. No new edges; only W8 internal content expanded. Topo-sort `{W0, W4.D1, W5, W6, W7.D1, W8} → {W1, W4.D2, W7.D2, W7.D4} → {W2, W7.D3} → W3` still valid.
11. **No Section 0 leakage in PLAN.md** — grep-equivalent scan shows no `Section 0`, `## Proposal`, `## QA Proof`, `## QA Falsification`, `## Convergence` headings in PLAN.md body. Round 5 Changes section is descriptive narrative, not Section 0 reasoning.
12. **Disposition language explicit (per new discipline)** — Round 5 Changes uses `MOOT —`, `DEFERRED-AS-NIT —`, `ABSORB —` prefixes with reasons. No "accepted as-is" / "no change" / "judgment call" phrases.

### Notes

- Round 5 is the smallest possible round in this drop's history — three NIT dispositions + one process-change note + two refinement-table rows + one W8 acceptance bullet + one W8 L2 directive paragraph. Zero impact on wave bodies, paths, packages, or blocked_by edges.
- The process-change note (line 20) is durable across drops. Future plan-QA verdicts and build-QA verdicts for this drop AND for Drop 4c.7+ inherit the discipline.
- W8-DECOMP-R1 and W8-SMOKE-R1 land in the refinements table per the round's design. Both are tracked-not-blocking — they document optional L2 decisions and deferred end-to-end smoke respectively. Neither blocks Wave A dispatch.
- The W8 acceptance smoke bullet's exact test location ("`render_test.go` or a new test in W8's per-prompt-set test suite") leaves L2 a one-line decision — acceptable scope for an L2 sub-planner.
- Sibling plan-QA-falsification round 5 firing in parallel — both must pass before Wave A heads (W0, W4.D1, W5, W6, W7.D1, **W8**) dispatch builders.
- Hylla feedback for this round appears in the closing comment (none — Round 5 review was Markdown-only on PLAN.md / PLAN_QA_PROOF.md / PLAN_QA_FALSIFICATION.md / REVISION_BRIEF.md; no Hylla queries required and no Go code touched).

---

## Round 6 — Plan-QA-Proof Verdict

**Drop:** `4c.6.1`
**Round:** 6 (plan-QA-proof)
**Pass / Fail:** **PASS**

### Findings

Round 6 is a surgical absorption of R5-FF1 (CRITICAL) + explicit dispositions on R5-NIT1 / R5-NIT2 per the "NITs are first-class" discipline (memory `feedback_nits_are_first_class.md`). All three findings carry explicit absorb/defer language; no silent drops. A new refinement row captures the pattern for future plan-QA falsification.

- **R5-FF1 (W8 smoke-test cross-wave dependency on W1's group-aware resolver)** — **ABSORB.** PLAN.md line 14 confirms the disposition: "W8 smoke-test droplet declares blocked_by 4c.6.1.W1 explicitly. Other 19 W8 prompt droplets stay Wave A unblocked." Three downstream artifacts wire the absorption correctly:
  - **W8 acceptance bullet parenthetical** — PLAN.md line 743 ends with: "(smoke-test droplet blocked_by W1; see L2 spawn directive) [New, not yet in tree — W8 authors it.]" The parenthetical pointer is explicit + actionable; the L2 sub-planner sees the cross-wave dep at L1 acceptance level without burying it solely in the directive.
  - **W8 L2 spawn directive cross-wave-dep note** — PLAN.md line 763: "**Cross-wave dependency note**: this smoke-test droplet's `blocked_by` MUST include `4c.6.1.W1` because `assembleAgentFileBody`'s subdir-per-group resolver shape is shipped by W1, NOT in W8's Wave A window. The other 19 W8 prompt droplets do NOT require this blocker (they only AUTHOR `.md` files; they don't exercise the resolver). The smoke-test droplet is the SOLE W8 droplet that crosses waves." Note is bolded, names the exact symbol (`assembleAgentFileBody`), explicitly carves out the other 19 droplets, and identifies the smoke-test droplet as the SOLE cross-wave crosser. Builder + L2 sub-planner cannot miss it.
  - **W8 outer `Blocked by:` empty stays correct** — line 744 unchanged: "Blocked by: — (Wave A head; no blockers — paths are entirely disjoint from all other waves)". W8 (the sub-plan container) is still Wave A; the cross-wave blocker lives on the single L2 smoke-test droplet, not the L1 container. Wave graph acyclicity preserved (W1 → W4.D1, W4.D1 has no upstream blockers; smoke-test droplet → W1 ≡ Wave C window for that droplet only).
- **R5-NIT1 (paraphrase quote drift)** — **DEFERRED-AS-NIT.** PLAN.md line 17: "DEFERRED-AS-NIT — reason: paraphrase substance accurate (low-fidelity but not fabricated); fixing risks more drift than benefit." Reason is sound — the substance ("either shape is defensible" with severity:low) is preserved exactly; re-editing risks cascading drift in other rounds that already cite the paraphrase.
- **R5-NIT2 (bracketed editorial note in W8 acceptance bullet)** — **DEFERRED-AS-NIT.** PLAN.md line 18: "DEFERRED-AS-NIT — reason: stylistic editorial note; non-blocking; doesn't change builder behavior." Reason is sound — the note is editorial meta-commentary; non-blocking cosmetic.
- **Pattern observation (PLAN-QA-DISCIPLINE-R1)** — captured both as Round 6 Changes narrative (line 20: "Pattern observation worth capturing for future plan-QA falsification: when an acceptance bullet exercises NEW behavior shipped by ANOTHER wave, the testing droplet MUST `blocked_by` that wave...") and as new refinement-table row at line 935: "PLAN-QA-DISCIPLINE-R1 | Future plan-QA falsification spawn briefs include 'for every acceptance bullet asserting NEW behavior, verify the test-runner droplet's blocked_by includes the wave that ships that behavior' as an explicit attack angle | tracked; process refinement". This is durable methodology improvement — same class as the R4→R5 process-change note ("NITs are first-class").

No new findings raised in Round 6. No residual NITs from Round 5 left undisposed.

### Evidence checked

1. **Round marker** — PLAN.md line 4: `**Round:** 6`. **VERIFIED.**
2. **Round 6 Changes section position** — PLAN.md lines 12–20, BEFORE Round 5 Changes (line 22). Newest-first ordering preserved. **VERIFIED.**
3. **Round 5/4/3/2/1 Changes preserved verbatim** — lines 22–73 unchanged from Round 5; Round 5 Changes section (lines 22–30) intact including the process-change note at line 30. Round-4 NIT entries (lines 32–48), Round-3 absorptions (lines 50–62), Round-2 absorptions (lines 63–72), Round-1 implicit baseline all preserved. **VERIFIED.**
4. **R5-FF1 ABSORB language explicit** — PLAN.md line 16: "**R5-FF1**: ABSORB — W8 smoke-test droplet declares blocked_by 4c.6.1.W1 explicitly. Other 19 W8 prompt droplets stay Wave A unblocked. (See W8 L2 spawn directive cross-wave dependency note.)" Uses explicit "ABSORB —" prefix per the dev discipline directive. **VERIFIED.**
5. **W8 acceptance bullet parenthetical** — PLAN.md line 743 carries explicit "(smoke-test droplet blocked_by W1; see L2 spawn directive)" inline with the smoke-test acceptance bullet. Pointer is concrete + actionable for an L2 sub-planner reading the L1 acceptance criteria. **VERIFIED.**
6. **W8 L2 spawn-directive cross-wave-dep note** — PLAN.md line 763 contains the bolded `**Cross-wave dependency note**:` paragraph with three actionable points: (a) the smoke-test droplet `blocked_by` MUST include `4c.6.1.W1`; (b) the named symbol `assembleAgentFileBody` is the reason; (c) the other 19 W8 prompt droplets do NOT require this blocker — explicit carve-out. The note is the SOLE W8 droplet that crosses waves. **VERIFIED.**
7. **R5-NIT1 DEFERRED-AS-NIT with reason** — PLAN.md line 17: "**R5-NIT1**: DEFERRED-AS-NIT — reason: paraphrase substance accurate (low-fidelity but not fabricated); fixing risks more drift than benefit." Uses explicit "DEFERRED-AS-NIT — reason:" prefix per dev discipline. **VERIFIED.**
8. **R5-NIT2 DEFERRED-AS-NIT with reason** — PLAN.md line 18: "**R5-NIT2**: DEFERRED-AS-NIT — reason: stylistic editorial note; non-blocking; doesn't change builder behavior." Uses explicit "DEFERRED-AS-NIT — reason:" prefix per dev discipline. **VERIFIED.**
9. **PLAN-QA-DISCIPLINE-R1 refinement row** — PLAN.md line 935 carries the new row with id `PLAN-QA-DISCIPLINE-R1` and description matching the spawn-prompt's expected text: "Future plan-QA falsification spawn briefs include 'for every acceptance bullet asserting NEW behavior, verify the test-runner droplet's blocked_by includes the wave that ships that behavior' as an explicit attack angle | tracked; process refinement". **VERIFIED.**
10. **Pattern observation in Round 6 Changes narrative** — PLAN.md line 20 carries the observation explicitly: "Pattern observation worth capturing for future plan-QA falsification: when an acceptance bullet exercises NEW behavior shipped by ANOTHER wave, the testing droplet MUST `blocked_by` that wave..." with the attack-angle phrasing matching the refinement row. **VERIFIED.**
11. **Wave graph acyclicity with new cross-wave blocker** — PLAN.md lines 767–791 wave-graph block unchanged from Round 5. W8 (L1 sub-plan container) remains Wave A with empty `blocked_by`. The new cross-wave blocker lives on the L2 smoke-test droplet (one of the 20 W8 children), which inherits `blocked_by: 4c.6.1.W1`. W1 → W4.D1 (Wave B). Smoke-test droplet effectively becomes Wave C (after W1 closes). No cycle introduced: W1's blocker chain (W4.D1, Wave B) does not transitively depend on any W8 child. Topo-sort with the new edge: `{W0, W4.D1, W5, W6, W7.D1, W8-non-smoke-droplets}` → `{W1, W4.D2, W7.D2, W7.D4}` → `{W2, W7.D3, W8-smoke-droplet}` → W3. Acyclic. **VERIFIED.**
12. **Other 19 W8 droplets remain Wave A** — line 763 explicit carve-out: "The other 19 W8 prompt droplets do NOT require this blocker (they only AUTHOR `.md` files; they don't exercise the resolver)." W8 outer `Blocked by:` line 744 still empty. Disjointness invariant preserved for the 19 non-smoke droplets. **VERIFIED.**
13. **No round-1/2/3/4/5 regression** — wave graph (lines 765–791), W7 4-droplet structure (lines 535–676), refinement table base rows (lines 914–934 — 21 prior rows: EXTRACT-R1 through W8-SMOKE-R1), decomposition shape table (lines 112–123), acceptance coverage map (lines 890–904), locked architectural decisions (lines 824–847), out-of-scope items (lines 874–887) — all preserved from Round 5 verbatim. **VERIFIED.**
14. **No Section 0 leakage** — Round 6 Changes section (lines 12–20) is descriptive narrative; no `# Section 0`, `## Proposal`, `## Builder`, `## Planner`, `## QA Proof` (as Section-0 pass title), `## Convergence` headings appear. The only `Section 0` mention in PLAN.md is the meta-commentary at line 908 ("Section 0 SEMI-FORMAL REASONING in every subagent response, but Section 0 stays in the orchestrator-facing response — never in PLAN.md or QA files") which is policy-prose, not reasoning leakage. **VERIFIED.**
15. **Disposition language explicit (per new discipline)** — Round 6 Changes uses `ABSORB —` and `DEFERRED-AS-NIT — reason:` prefixes with reasons inline. No "judgment call" / "as-is" / "accepted" / "no change" phrases. Conforms to the Round 5 process-change directive at line 30. **VERIFIED.**

### Notes

- Round 6 is a surgical absorption pass — five total content changes relative to Round 5: (a) round marker bumped to 6; (b) Round 6 Changes section inserted; (c) W8 acceptance bullet parenthetical added; (d) W8 L2 spawn-directive cross-wave dependency note appended; (e) one new refinement row (PLAN-QA-DISCIPLINE-R1). Zero impact on wave graph topology (the new blocker lives at L2 inside the W8 sub-plan, not on the L1 container), W7 4-droplet structure, decomposition shape table, acceptance map, locked decisions, out-of-scope items, or any other surface.
- The cross-wave dep is **declared at three levels of fidelity** — Round 6 Changes summary (line 16), L1 acceptance parenthetical (line 743), and L2 spawn-directive bolded note (line 763). Triangulation across the three is intentional: even if an L2 sub-planner reads only the L1 paths or only the directive, the dependency surfaces. The bolded directive note is the most actionable form (names `assembleAgentFileBody` + carves out the other 19 droplets); the parenthetical is the quick-reference at acceptance level; the Round 6 Changes summary is the audit trail.
- PLAN-QA-DISCIPLINE-R1 is a durable methodology improvement. It captures the same root cause as the R1/R2/R3 W7 dependency-chasing AND the R5-FF1 absorption gap: any acceptance bullet that asserts NEW behavior must be tied to the wave that ships that behavior. The refinement formalizes this as an explicit attack angle for future plan-QA falsification rounds. This is the THIRD process-change-class refinement in this drop's history (joining "Specify the residue, not the deletion" from W7 inverted carving and "NITs are first-class" from R4→R5), demonstrating that the iterated plan-QA loop is producing reusable methodology.
- W8 outer `Blocked by:` remaining empty is correct. The L1 sub-plan container itself is unblocked; only ONE L2 droplet (the smoke-test) carries the W1 blocker. The W8 sub-plan can start its L2 decomposition immediately at drop start (Wave A window), and 19 of its 20 L2 droplets can run in Wave A; only the smoke-test droplet waits for W1.
- Sibling plan-QA-falsification round 6 firing in parallel — both must pass before Wave A heads (W0, W4.D1, W5, W6, W7.D1, **W8**) dispatch builders.
- Hylla feedback for this round appears in the closing comment (none — Round 6 review was Markdown-only on PLAN.md / PLAN_QA_PROOF.md / PLAN_QA_FALSIFICATION.md; no Hylla queries required and no Go code touched).

---

## Round 7 — Plan-QA-Proof Verdict

**Drop:** `4c.6.1`
**Round:** 7 (plan-QA-proof)
**Pass / Fail:** **PASS**

### Findings

Round 7 is a surgical absorption pass closing the three Round-6 findings (R6-FF1, R6-FF2, R6-NIT1) PLUS adding PLAN-QA-DISCIPLINE-R2 as a second-layer methodology refinement. All four findings carry explicit ABSORB / DEFERRED-AS-NIT-with-reason language per the dev's "NITs are first-class" discipline (memory `feedback_nits_are_first_class.md`). No silent drops.

- **R6-FF1 (L1 structural claims about W8's wave window not swept post R5-FF1 absorption)** — **ABSORB.** All three load-bearing L1 surfaces flagged in the falsifier verdict (lines 122, 793, 804 pre-Round-7) have been swept:
  - **Decomposition shape table W8 row (PLAN.md line 131 post-Round-7)** rewritten to acknowledge DUAL-WAVE: "~22 build droplets: 19 prompt-authoring droplets (Wave A) + `.tillsyn/bindings.json` + `.gitignore` re-includes (Wave A) + 1 dedicated smoke-test droplet (Wave C, `blocked_by W1`); prompt files are separate atomic droplets. DUAL-WAVE sub-plan: prompt-authoring droplets touch only `.tillsyn/` files; smoke-test droplet (D19) touches `internal/app/dispatcher/cli_claude/render/render_test.go`." The DUAL-WAVE framing + (D19) name + cross-wave path distinction are explicit + load-bearing. Verified at PLAN.md line 131.
  - **Wave-A roster (PLAN.md line 812 post-Round-7)** updated: "Wave A (parallel): W0, W4.D1, W5, W6, **W7.D1 (Inventory)**, W8 (Tillsyn-project-local prompts) — 19 prompt-authoring droplets are Wave A; the 20th (smoke-test D19, `blocked_by W1`) lands at Wave C transitively." The W8 entry now explicitly carves out the smoke-test droplet from pure Wave A. Verified at PLAN.md line 812.
  - **Parallelism note for W8 (PLAN.md line 823 post-Round-7)** rewritten: "W8 (Tillsyn-project-local prompts) is a DUAL-WAVE sub-plan — 19 prompt-authoring droplets touch only `.tillsyn/` files (Wave A, parallel with everything else); 1 smoke-test droplet (D19) touches `internal/app/dispatcher/cli_claude/render/render_test.go` and is `blocked_by 4c.6.1.W1` (Wave B), placing it in Wave C transitively. The W8 sub-plan container completion thus spans Wave A→Wave C." The "DUAL-WAVE sub-plan" framing replaces the prior "fully disjoint Wave A workstream" claim; "spans Wave A→Wave C" replaces "unblocked by anything in Waves B–D." Both formerly-false claims removed. Verified at PLAN.md line 823.
- **R6-FF2 ("LAST prompt droplet" structurally infeasible for smoke test — needs dedicated droplet)** — **ABSORB.** The W8 L2 spawn directive's R3-NIT4 smoke-test guidance has been rewritten:
  - PLAN.md line 770 now reads: "**R3-NIT4 smoke test requirement (REQUIRED in W8 L2) — DEDICATED D19 DROPLET**: Add a **new dedicated smoke-test droplet** (D19, or the next sequential index after all prompt-authoring droplets) AFTER the 19 prompt-authoring droplets..." The "LAST prompt droplet MUST include" framing from Round 6 (which was structurally infeasible per R6-FF2) is replaced with "DEDICATED D19 DROPLET" + "new dedicated smoke-test droplet". Verified.
  - PLAN.md lines 772–776 (path/package distinction) make explicit: "Prompt-authoring droplets: paths `.tillsyn/agents/<group>/<name>.md`, packages: none, atomicity: file-write-only. Smoke-test droplet (D19): paths `internal/app/dispatcher/cli_claude/render/render_test.go` (MODIFY — add test case; or a new test file in that package), packages: `internal/app/dispatcher/cli_claude/render`, atomicity: Go test addition. Different path + package locks; cannot live in the same droplet per atomic-droplet sizing + dispatcher lock semantics." The path/package boundary is named exactly, the lock-semantics justification is named exactly. L2 sub-planner cannot conflate the smoke test with a prompt droplet. Verified at PLAN.md lines 772–776.
  - PLAN.md lines 778–782 (blocked_by spec) name the smoke-test droplet's full blocker set: "Smoke-test droplet `blocked_by`: - All 19 prompt-authoring droplets (sequencing — smoke needs the prompt files written). - `4c.6.1.W1` (resolver subdir-per-group shape — smoke exercises `assembleAgentFileBody` per the new resolver). This makes the smoke-test droplet Wave C transitively (after W1 lands in Wave B + after the prompt droplets in Wave A). The other 19 W8 prompt droplets do NOT require the `blocked_by W1` blocker (they only AUTHOR `.md` files; they don't exercise the resolver)." Full blocker set (all 19 prompts + W1) is explicit; Wave-C transitive placement is named; carve-out for non-smoke droplets is preserved. Verified at PLAN.md lines 778–782.
- **R6-NIT1 (PLAN-QA-DISCIPLINE-R1 row 3 cells vs 2-column schema)** — **ABSORB.** PLAN.md line 954 (PLAN-QA-DISCIPLINE-R1 row) now has 2 cells, with the "(tracked; process refinement)" suffix inlined as a parenthetical at the tail of the Description cell. Schema-conformant. Verified at PLAN.md line 954.
- **PLAN-QA-DISCIPLINE-R2 (new second-layer methodology refinement)** — **ADDED** at PLAN.md line 955: "For every surgical cross-wave or cross-droplet absorption in round-N+1 planning, sweep all L1 structural claims (wave roster, parallelism notes, decomposition-shape table, dependency graph) to verify they still hold post-absorption. Captured after R6-FF1 (round-6 added W8 cross-wave dep but didn't update lines 122/793/804) (tracked; process refinement)." 2-cell row, schema-conformant, names the exact root cause (R6-FF1) + the exact line ranges (122/793/804) + the prescription (sweep all four L1 structural surfaces). This is the falsifier's recommended sibling-pattern refinement (line 977 of PLAN_QA_FALSIFICATION.md round-6 entry: "second-layer refinement may be worth capturing: 'for every surgical absorption of a cross-wave dep, verify all L1 structural claims about the affected wave / sub-plan window are still accurate.'"). Captured exactly. Verified at PLAN.md line 955.

No new findings raised in Round 7. No residual NITs from Round 6 left undisposed.

### Evidence checked

1. **Round marker** — PLAN.md line 4: `**Round:** 7`. **VERIFIED.**
2. **Round 7 Changes section position** — PLAN.md lines 12–19, BEFORE Round 6 Changes (line 21). Newest-first ordering preserved. **VERIFIED.**
3. **Round 6/5/4/3/2/1 Changes preserved verbatim** — Round 6 Changes (lines 21–29), Round 5 Changes (lines 31–39), Round 4 Changes (lines 41–57), Round 3 Changes (lines 59–70), Round 2 Changes (lines 72–81), and Round 1 baseline narrative all preserved unchanged from Round 6. **VERIFIED.**
4. **R6-FF1 ABSORB language explicit** — PLAN.md line 16: "**R6-FF1**: ABSORB — swept PLAN.md lines 122/793/804 to acknowledge W8 is now a DUAL-WAVE sub-plan (19 prompt droplets Wave A; 1 dedicated smoke-test droplet D19 Wave C transitively, blocked by W1)." Uses explicit "ABSORB —" prefix per dev discipline; names the three line ranges that were swept. **VERIFIED.**
5. **R6-FF1 — decomposition shape table W8 row swept** — PLAN.md line 131 (decomposition table) reads: "W8 — sub-plan container — ~22 build droplets: 19 prompt-authoring droplets (Wave A) + `.tillsyn/bindings.json` + `.gitignore` re-includes (Wave A) + 1 dedicated smoke-test droplet (Wave C, `blocked_by W1`); prompt files are separate atomic droplets. DUAL-WAVE sub-plan: prompt-authoring droplets touch only `.tillsyn/` files; smoke-test droplet (D19) touches `internal/app/dispatcher/cli_claude/render/render_test.go`." DUAL-WAVE language present; D19 name present; cross-wave dep documented at row level. **VERIFIED.**
6. **R6-FF1 — Wave-A roster swept** — PLAN.md line 812 reads: "Wave A (parallel): W0, W4.D1, W5, W6, **W7.D1 (Inventory)**, W8 (Tillsyn-project-local prompts) — 19 prompt-authoring droplets are Wave A; the 20th (smoke-test D19, `blocked_by W1`) lands at Wave C transitively." Pure-Wave-A claim about W8 is replaced with explicit 19-of-20 carve-out. **VERIFIED.**
7. **R6-FF1 — parallelism note swept** — PLAN.md line 823 reads: "W8 (Tillsyn-project-local prompts) is a DUAL-WAVE sub-plan — 19 prompt-authoring droplets touch only `.tillsyn/` files (Wave A, parallel with everything else); 1 smoke-test droplet (D19) touches `internal/app/dispatcher/cli_claude/render/render_test.go` and is `blocked_by 4c.6.1.W1` (Wave B), placing it in Wave C transitively. The W8 sub-plan container completion thus spans Wave A→Wave C." "DUAL-WAVE sub-plan" + "spans Wave A→Wave C" replace the formerly-false "fully disjoint Wave A workstream" + "unblocked by anything in Waves B–D" claims. **VERIFIED.**
8. **R6-FF2 ABSORB language explicit** — PLAN.md line 17: "**R6-FF2**: ABSORB — rewrote W8 L2 spawn directive cross-wave note: smoke test is a DEDICATED L2 droplet D19 (paths/packages distinct from prompt-authoring droplets; cannot live in 'LAST prompt droplet' per atomic-droplet sizing + path/package lock semantics)." Uses explicit "ABSORB —" prefix; names the structural reason (atomic-droplet sizing + path/package lock semantics). **VERIFIED.**
9. **R6-FF2 — W8 L2 spawn directive rewritten with DEDICATED D19** — PLAN.md line 770: "**R3-NIT4 smoke test requirement (REQUIRED in W8 L2) — DEDICATED D19 DROPLET**: Add a **new dedicated smoke-test droplet** (D19, or the next sequential index after all prompt-authoring droplets) AFTER the 19 prompt-authoring droplets." The Round-6 "LAST prompt droplet MUST include" framing is REPLACED with "DEDICATED D19 DROPLET" + "new dedicated smoke-test droplet AFTER the 19 prompt-authoring droplets." Structural infeasibility resolved. **VERIFIED.**
10. **R6-FF2 — path/package distinction documented** — PLAN.md lines 772–776: "Prompt-authoring droplets: paths `.tillsyn/agents/<group>/<name>.md`, packages: none, atomicity: file-write-only. Smoke-test droplet (D19): paths `internal/app/dispatcher/cli_claude/render/render_test.go` (MODIFY — add test case; or a new test file in that package), packages: `internal/app/dispatcher/cli_claude/render`, atomicity: Go test addition. Different path + package locks; cannot live in the same droplet per atomic-droplet sizing + dispatcher lock semantics." Both droplet types' paths + packages + atomicity are named exactly; lock-semantics justification is explicit. **VERIFIED.**
11. **R6-FF2 — smoke-test droplet `blocked_by` covers all 19 prompts + W1** — PLAN.md lines 778–782: "Smoke-test droplet `blocked_by`: - All 19 prompt-authoring droplets (sequencing — smoke needs the prompt files written). - `4c.6.1.W1` (resolver subdir-per-group shape — smoke exercises `assembleAgentFileBody` per the new resolver). This makes the smoke-test droplet Wave C transitively (after W1 lands in Wave B + after the prompt droplets in Wave A)." Full blocker set is enumerated; Wave-C transitive placement is named; resolver symbol `assembleAgentFileBody` named. **VERIFIED.**
12. **R6-NIT1 ABSORB language explicit** — PLAN.md line 18: "**R6-NIT1**: ABSORB — fixed PLAN-QA-DISCIPLINE-R1 refinements-table row from 3 cells to 2 cells to match 2-column table schema." Uses explicit "ABSORB —" prefix; names exact row + cell-count change. **VERIFIED.**
13. **R6-NIT1 — PLAN-QA-DISCIPLINE-R1 row is 2 cells** — PLAN.md line 954: `| PLAN-QA-DISCIPLINE-R1 | Future plan-QA falsification spawn briefs include "for every acceptance bullet asserting NEW behavior, verify the test-runner droplet's blocked_by includes the wave that ships that behavior" as an explicit attack angle (tracked; process refinement) |`. Two pipe-separated cells; the "(tracked; process refinement)" suffix is inlined as a parenthetical at the tail of the Description cell. Conforms to 2-column header `| ID | Description |` at line 931. **VERIFIED.**
14. **PLAN-QA-DISCIPLINE-R2 refinement-table row added** — PLAN.md line 955: `| PLAN-QA-DISCIPLINE-R2 | For every surgical cross-wave or cross-droplet absorption in round-N+1 planning, sweep all L1 structural claims (wave roster, parallelism notes, decomposition-shape table, dependency graph) to verify they still hold post-absorption. Captured after R6-FF1 (round-6 added W8 cross-wave dep but didn't update lines 122/793/804) (tracked; process refinement) |`. Two cells, schema-conformant. Description names: (a) the prescription (sweep all four L1 structural surfaces), (b) the root cause (R6-FF1 — round-6 added cross-wave dep but didn't sweep), (c) the exact line ranges (122/793/804) that drifted. Captured at the position the falsifier verdict suggested (line 977 of PLAN_QA_FALSIFICATION.md round-6 entry). **VERIFIED.**
15. **Round 7 Changes narrative summarizes all 4 dispositions** — PLAN.md lines 14–19: opening paragraph at line 14 cites R6-FF1 + R6-FF2 + R6-NIT1 origins; lines 16/17/18 give per-finding ABSORB dispositions with reasons; line 19 documents PLAN-QA-DISCIPLINE-R2 addition. All four artifacts of Round 7 are documented in the Changes section. **VERIFIED.**
16. **Acyclicity preserved with W8.D19 → W1 edge** — PLAN.md lines 801–810 wave graph block:  "Wave C (after Wave B): 4c.6.1.W2 → 4c.6.1.W1, 4c.6.1.W4.D1, 4c.6.1.W5; 4c.6.1.W7.D3 → 4c.6.1.W7.D2, 4c.6.1.W2." The W8.D19 → W1 edge is captured at lines 813–814 narrative ("the 20th (smoke-test D19, blocked_by W1) lands at Wave C transitively"; "Wave C (after Wave B): W2 ... + W7.D3 ... + W8.D19 smoke-test droplet (blocked by W1 + all 19 W8 prompt droplets)"). Topo-sort with the new W8.D19 edge: `{W0, W4.D1, W5, W6, W7.D1, W8-non-smoke-droplets}` → `{W1, W4.D2, W7.D2, W7.D4}` → `{W2, W7.D3, W8.D19}` → W3. W1's blocker chain is W4.D1 (Wave A); W8.D19 → W1 → W4.D1 has no cycle. **No cycle confirmed.** **VERIFIED.**
17. **No round-1/2/3/4/5/6 regression** — Round 6 Changes (lines 21–29) preserved verbatim post-Round-7; Round 5 Changes (lines 31–39) preserved; Round 4/3/2 Changes preserved; W7 4-droplet structure (lines 544–683) preserved; base refinements rows EXTRACT-R1 through PLAN-QA-DISCIPLINE-R1 (lines 933–954) preserved; W8 acceptance bullet smoke-test parenthetical "(smoke-test droplet blocked_by W1; see L2 spawn directive)" (line 752) preserved unchanged. **VERIFIED.**
18. **No Section 0 leakage** — Round 7 Changes section (lines 12–19) is descriptive narrative; no `# Section 0`, `## Proposal`, `## Builder` (as Section-0 pass), `## QA Proof`/`## QA Falsification` (as Section-0 pass titles), `## Convergence` headings appear. The only Section 0 mention in PLAN.md is policy-prose at line 927 ("Section 0 SEMI-FORMAL REASONING in every subagent response, but Section 0 stays in the orchestrator-facing response — never in PLAN.md or QA files"). **VERIFIED.**
19. **Disposition language explicit (per dev discipline)** — Round 7 Changes uses `ABSORB —` prefixes with reasons inline on all three R6 findings; the PLAN-QA-DISCIPLINE-R2 addition is labeled as "Added" narrative. No "judgment call" / "as-is" / "accepted" / "no change" phrases. Conforms to Round 5's process-change directive. **VERIFIED.**

### Notes

- Round 7 is the largest surgical absorption pass in this drop's history relative to the round-N-1 falsifier verdict — three L1 line-range sweeps (122/793/804) plus the W8 L2 spawn-directive rewrite covering 6+ paragraphs (lines 770–782) plus one refinement-row format fix plus one new refinement row. The size reflects the sibling-pattern lesson PLAN-QA-DISCIPLINE-R2 captures: surgical fixes in cross-wave-dep contexts have wide structural blast radius across L1 documentation surfaces.
- The DUAL-WAVE framing at line 131 (decomposition table), line 823 (parallelism note), and the carve-out at line 812 (Wave-A roster) form a triangulated description of W8's wave window. Each surface independently tells the reader "W8 is mostly-Wave-A with one Wave-C smoke-test droplet." Triangulation across three surfaces is the structural defense against future surgical-edit drift that PLAN-QA-DISCIPLINE-R2 prescribes.
- PLAN-QA-DISCIPLINE-R2 is the FOURTH process-change-class refinement in this drop's history (joining "Specify the residue, not the deletion" from W7 inverted carving, "NITs are first-class" from R4→R5, and PLAN-QA-DISCIPLINE-R1 from R5→R6). The iterated plan-QA loop continues to produce reusable methodology — each round's surgical absorption surfaces a new blind-spot pattern that becomes a future-attack-angle refinement.
- The smoke-test droplet D19's `blocked_by` set in line 778–782 spans both (a) sequencing-sibling (the 19 prompt droplets — the smoke test cannot run without the prompt files in tree) AND (b) cross-wave-resolver-dep (W1 — the resolver shape under test must ship first). Naming both classes of blocker explicitly defends against the dispatcher misdispatching the smoke test before either prerequisite class lands.
- The acyclicity check now has one additional edge to verify: W8.D19 → W1. W1's blocker chain (W1 → W4.D1; W4.D1 has no blockers) does NOT transit through any W8 child, so no cycle is introduced. The smoke-test droplet's transitive Wave-C placement is confirmed.
- Round 7 has zero impact on: W7 4-droplet structure, decomposition shape table's non-W8 rows, acceptance criteria coverage map, locked architectural decisions, out-of-scope items, Wave-D placement of W3, paths/packages declarations on any non-W8 droplet, blocked_by edges on any non-W8 droplet, embedded agent file set, FE scaffold, vim engine, TUI components.
- Sibling plan-QA-falsification round 7 firing in parallel — both must pass before Wave A heads (W0, W4.D1, W5, W6, W7.D1, **W8**) dispatch builders.
- Hylla feedback for this round appears in the closing comment (none — Round 7 review was Markdown-only on PLAN.md / PLAN_QA_PROOF.md / PLAN_QA_FALSIFICATION.md; no Hylla queries required and no Go code touched).

---

## Round 8 — Plan-QA-Proof Verdict

**Drop:** `4c.6.1`
**Round:** 8 (plan-QA-proof)
**Pass / Fail:** **PASS** (one cosmetic NIT — see below)

### Findings

Round 8 is a surgical absorption pass closing R7-FF1 (narrative droplet-count drift) + R7-NIT1 (D-range internal consistency) + extending PLAN-QA-DISCIPLINE-R2 with a numeric-consistency sub-clause. All three findings carry explicit ABSORB / DEFERRED-AS-NIT-with-reason language per the "NITs are first-class" discipline (memory `feedback_nits_are_first_class.md`). No silent drops.

- **R7-FF1 (narrative "19 prompt-authoring droplets" inconsistent with L2 spawn directive's enumerated D-list)** — **ABSORB.** Option α (un-batch D8 + D16) applied. All load-bearing narrative occurrences of the prior "19 prompt-authoring droplets" framing have been swept to "20":
  - **Decomposition shape table W8 row (PLAN.md line 140 post-Round-8)** rewritten: "~22 build droplets: 20 prompt-authoring droplets (Wave A) + `.tillsyn/bindings.json` + `.gitignore` re-includes (Wave A, D0) + 1 dedicated smoke-test droplet (Wave C, `blocked_by W1`); ... smoke-test droplet (D21) touches `internal/app/dispatcher/cli_claude/render/render_test.go`." The count + D21 smoke-test name + path/package distinction are explicit + load-bearing. **VERIFIED.**
  - **W8 L2 spawn-directive smoke-test heading (PLAN.md line 779 post-Round-8)** now reads: "**R3-NIT4 smoke test requirement (REQUIRED in W8 L2) — DEDICATED D21 DROPLET**: Add a **new dedicated smoke-test droplet** (D21, after D0 bindings + D1-D20 prompts) AFTER the 20 prompt-authoring droplets." D19→D21 rename complete; "20 prompt-authoring droplets" replaces the prior 19 count. **VERIFIED.**
  - **W8 L2 spawn-directive path/package distinction (PLAN.md line 781 post-Round-8)**: "Cross-wave dependency note: the smoke-test is a DEDICATED L2 droplet (separate from the 20 prompt-authoring droplets) because: - Prompt-authoring droplets: paths `.tillsyn/agents/<group>/<name>.md`, packages: none, atomicity: file-write-only. - Smoke-test droplet (D21): paths `internal/app/dispatcher/cli_claude/render/render_test.go` (MODIFY — add test case; or a new test file in that package), packages: `internal/app/dispatcher/cli_claude/render`, atomicity: Go test addition." Count + D21 name both correct. **VERIFIED.**
  - **W8 L2 spawn-directive blocked_by spec (PLAN.md line 788 post-Round-8)**: "Smoke-test droplet `blocked_by`: - All 20 prompt-authoring droplets (sequencing — smoke needs the prompt files written). - `4c.6.1.W1`..." Full blocker set with correct count (20 prompts + W1). **VERIFIED.**
  - **W8 L2 spawn-directive Wave-C transitive note (PLAN.md line 791 post-Round-8)**: "This makes the smoke-test droplet Wave C transitively (after W1 lands in Wave B + after the prompt droplets in Wave A). The other 20 W8 prompt droplets do NOT require the `blocked_by W1` blocker..." Count correct. **VERIFIED.**
  - **Wave-A roster (PLAN.md line 821 post-Round-8)** updated: "Wave A (parallel): W0, W4.D1, W5, W6, **W7.D1 (Inventory)**, W8 (Tillsyn-project-local prompts) — 20 prompt-authoring droplets are Wave A; the 21st (smoke-test D21, `blocked_by W1`) lands at Wave C transitively." Count + D21 name both correct. **VERIFIED.**
  - **Wave-C section (PLAN.md line 823 post-Round-8)**: "Wave C (after Wave B): W2 (blocked by W1 + W4.D1 + W5) + W7.D3 (Delete-residue, blocked by W7.D2 + W2 for cmd/till compile lock) + W8.D21 smoke-test droplet (blocked by W1 + all 20 W8 prompt droplets)." D21 name + 20 count both correct. **VERIFIED.**
  - **Parallelism note for W8 (PLAN.md line 832 post-Round-8)**: "W8 (Tillsyn-project-local prompts) is a DUAL-WAVE sub-plan — 20 prompt-authoring droplets touch only `.tillsyn/` files (Wave A, parallel with everything else); 1 smoke-test droplet (D21) touches `internal/app/dispatcher/cli_claude/render/render_test.go` and is `blocked_by 4c.6.1.W1` (Wave C transitively, after W1's Wave B resolver lands)." Count + D21 name both correct. **VERIFIED.**
- **R7-NIT1 (D-range internal consistency — "D9-D18 same shape" was a batched range that didn't enumerate the un-batched group)** — **ABSORB.** PLAN.md line 764 (W8 L2 spawn directive) now enumerates D1-D10 (go group) AND D11-D20 (fe group) explicitly with full file-name per-droplet: "D1 `go/planning-agent.md`; D2 `go/builder-agent.md`; D3 `go/plan-qa-proof-agent.md`; D4 `go/plan-qa-falsification-agent.md`; D5 `go/build-qa-proof-agent.md`; D6 `go/build-qa-falsification-agent.md`; D7 `go/research-agent.md`; D8 `go/closeout-agent.md`; D9 `go/commit-message-agent.md`; D10 `go/orchestrator-managed.md`; D11–D20 same shape for `fe/` group (D11 `fe/planning-agent.md`; ...; D20 `fe/orchestrator-managed.md`)." No batching language ("same shape for D9-D18"); fe group fully named D11-D20. **VERIFIED.**
- **PLAN-QA-DISCIPLINE-R2 numeric-consistency sub-clause** — **ADDED** at PLAN.md line 964: the existing R2 row was extended with "Includes verifying NUMERIC consistency — narrative droplet COUNTS in L1 must match the L2 spawn directive's enumerated D-list. Counts carried forward unverified from prior rounds are a recurring failure pattern (captured from R7-FF1)." Captures the exact failure pattern R7-FF1 surfaced (carried-forward stale counts post-surgical-edit). 2-cell row preserved, schema-conformant. **VERIFIED.**

No new high/critical findings raised in Round 8. One cosmetic NIT (see below).

### NITs

- **NIT-R8-1 — Round 8 Changes describes un-batching with `D8a/D8b/D8c` sub-letter notation, but the actual L2 spawn directive uses sequential `D8/D9/D10`.** PLAN.md line 16 says "un-batched D8 (closeout/commit-message/orchestrator-managed → D8a/D8b/D8c) and D16 (same for fe)." The actual directive at line 764 enumerates them as D8/D9/D10 (go) and D18/D19/D20 (fe) — sequential, not sub-lettered. The OUTCOME is correct (20 droplets total, properly enumerated, count consistent across all surfaces); only the descriptive shorthand in Round 8 Changes uses the sub-letter notation that doesn't appear in the implementation. **DEFERRED-AS-NIT** — reason: cosmetic, count + outcome are correct; Round 8 Changes is historical narrative explaining the option α absorption and the sub-letter notation is a legitimate description of the un-batching operation even though the linearized result uses sequential D-numbers. Fixing risks more drift than benefit (would force renaming D9/D10/D19/D20 to D8b/D8c/D16b/D16c which adds zero clarity and breaks the natural sequential enumeration). Severity: low.

### Evidence checked

1. **Round marker** — PLAN.md line 4: `**Round:** 8`. **VERIFIED.**
2. **Round 8 Changes section position** — PLAN.md lines 12–19, BEFORE Round 7 Changes (line 21). Newest-first ordering preserved. **VERIFIED.**
3. **Round 7/6/5/4/3/2/1 Changes preserved verbatim** — Round 7 Changes (lines 21–28) preserved; Round 6 Changes (lines 30–38) preserved (line 34 retains historical "19 W8 prompt droplets stay Wave A unblocked" — correct historical preservation); Round 5/4/3/2 Changes preserved. **VERIFIED.**
4. **R7-FF1 ABSORB language explicit** — PLAN.md line 16: "**R7-FF1**: ABSORB — un-batched D8 (closeout/commit-message/orchestrator-managed → D8a/D8b/D8c) and D16 (same for fe). Total = 20 prompt-authoring droplets (10 × 2 groups). Smoke renamed D19 → D21. Six narrative occurrences updated. R7-NIT1 (D-range inconsistency) folded in." Uses explicit "ABSORB —" prefix; names the option (α = un-batch D8 + D16); names the new total (20 = 10 × 2); names the smoke rename (D19 → D21). **VERIFIED.**
5. **W8 outer acceptance count consistent with new totals** — PLAN.md line 752: "All 20 prompt files exist (10 go + 10 fe) with non-stub bodies (>= 1000 chars each)." Count 20 + 10+10 split + standard placeholder size — all consistent. **VERIFIED.**
6. **Spawn directive enumerates D1-D20 explicitly, no batching language; fe group fully named D11-D20** — PLAN.md line 764 enumerates: D1 (go/planning) → D2 (go/builder) → D3 (go/plan-qa-proof) → D4 (go/plan-qa-falsification) → D5 (go/build-qa-proof) → D6 (go/build-qa-falsification) → D7 (go/research) → D8 (go/closeout) → D9 (go/commit-message) → D10 (go/orchestrator-managed); D11 (fe/planning) → D12 (fe/builder) → D13 (fe/plan-qa-proof) → D14 (fe/plan-qa-falsification) → D15 (fe/build-qa-proof) → D16 (fe/build-qa-falsification) → D17 (fe/research) → D18 (fe/closeout) → D19 (fe/commit-message) → D20 (fe/orchestrator-managed). No `D9-D18 same shape` / `D8a/D8b/D8c` / batching language. Full 20-droplet enumeration. **VERIFIED.**
7. **D21 smoke-test references throughout** — PLAN.md line 140 (decomp table) "smoke-test droplet (D21)"; line 779 "DEDICATED D21 DROPLET"; line 783 "Smoke-test droplet (D21)"; line 823 "W8.D21 smoke-test droplet"; line 832 "1 smoke-test droplet (D21)"; line 821 "the 21st (smoke-test D21, `blocked_by W1`)". Five+ smoke-test name references all consistent at D21. **VERIFIED.**
8. **PLAN-QA-DISCIPLINE-R2 has numeric-consistency sub-clause** — PLAN.md line 964: `| PLAN-QA-DISCIPLINE-R2 | For every surgical cross-wave or cross-droplet absorption in round-N+1 planning, sweep all L1 structural claims (wave roster, parallelism notes, decomposition-shape table, dependency graph) to verify they still hold post-absorption. Captured after R6-FF1 (round-6 added W8 cross-wave dep but didn't update lines 122/793/804). Includes verifying NUMERIC consistency — narrative droplet COUNTS in L1 must match the L2 spawn directive's enumerated D-list. Counts carried forward unverified from prior rounds are a recurring failure pattern (captured from R7-FF1) (tracked; process refinement) |`. 2-cell row, schema-conformant. The new sub-clause names: (a) the new attack angle (numeric consistency between narrative counts and enumerated D-lists), (b) the root cause (counts carried forward unverified post-surgical-edit), (c) the originating failure (R7-FF1). **VERIFIED.**
9. **Round 6 Changes line 34 (HISTORICAL) preserved correctly** — PLAN.md line 34 reads: "**R5-FF1**: ABSORB — W8 smoke-test droplet declares blocked_by 4c.6.1.W1 explicitly. Other 19 W8 prompt droplets stay Wave A unblocked. (See W8 L2 spawn directive cross-wave dependency note.)" This is HISTORICAL Round 6 narrative describing the R5-FF1 absorption state as of Round 6; the count "19" was correct at Round 6 (before the Round 8 un-batching). Preservation as historical narrative is correct. **VERIFIED.**
10. **Acyclic graph preserved** — Topo-sort with the Round-8 renaming W8.D19 → W8.D21: `{W0, W4.D1, W5, W6, W7.D1, W8-non-smoke-droplets (D0 + D1-D20)}` → `{W1, W4.D2, W7.D2, W7.D4}` → `{W2, W7.D3, W8.D21}` → `{W3}`. The Round-7 acyclicity proof still holds — only the smoke-test droplet's identifier changed (D19 → D21) and the prompt count grew (19 → 20). W1's blocker chain (W1 → W4.D1; W4.D1 has no blockers) does NOT transit through any W8 child, so no cycle is introduced. PLAN.md line 826: "Acyclicity check (topo-sort): {W0, W4.D1, W5, W6, W7.D1, W8} → {W1, W4.D2, W7.D2, W7.D4} → {W2, W7.D3} → W3. No cycle confirmed." (W8.D21 lives inside the W8 sub-plan; the topo-sort treats W8 as the container.) **VERIFIED.**
11. **No Section 0 leakage** — Round 8 Changes section (lines 12–19) is descriptive narrative; no `# Section 0`, `## Proposal`, `## Builder` (as Section-0 pass), `## QA Proof`/`## QA Falsification` (as Section-0 pass titles), `## Convergence` headings appear. The only Section 0 mention in PLAN.md is policy-prose at line 936 ("Section 0 SEMI-FORMAL REASONING in every subagent response, but Section 0 stays in the orchestrator-facing response — never in PLAN.md or QA files"). **VERIFIED.**
12. **Total droplet count holds at 22** — D0 (`.gitignore` + `bindings.json`) + D1-D20 (20 prompt-authoring droplets) + D21 (smoke-test) = 22 droplets total for W8 sub-plan. Matches PLAN.md line 140 decomposition table: "~22 build droplets." **VERIFIED.**
13. **Disposition language explicit (per dev discipline)** — Round 8 Changes uses `ABSORB —` prefix on R7-FF1 (line 16); the PLAN-QA-DISCIPLINE-R2 extension is labeled as "Extended" narrative (line 17). NIT-R8-1 above carries DEFERRED-AS-NIT with explicit reason. No "judgment call" / "as-is" / "accepted" / "no change" phrases. Conforms to Round 5's process-change directive. **VERIFIED.**

### Notes

- Round 8 is the second consecutive process-change-discipline round (Round 7 added PLAN-QA-DISCIPLINE-R2; Round 8 extends it with a numeric-consistency sub-clause). The iterated plan-QA loop continues to produce reusable methodology — each round's surgical absorption surfaces a new blind-spot pattern that becomes a future-attack-angle refinement.
- The R7-FF1 absorption is structurally clean: each load-bearing narrative surface (decomposition shape table, smoke-test heading, path/package distinction, blocked_by spec, Wave-C transitive note, Wave-A roster, Wave-C section, parallelism note) is independently updated to "20 prompt-authoring droplets" + "D21 smoke-test." Triangulation across 7+ surfaces ensures any future surgical edit will be caught by PLAN-QA-DISCIPLINE-R2's numeric-consistency sub-clause.
- The W8 L2 spawn directive at line 764 now enumerates all 20 prompt-authoring droplets with explicit file-name pairs. This eliminates the prior batching-language ambiguity (R7-NIT1) and gives the L2 sub-planner a complete authoritative D-list to act on. The fe group's D11-D20 are fully named (not just "same shape D11-D20") per the dev's explicit directive.
- NIT-R8-1 (D8a/D8b/D8c text vs D8/D9/D10 directive enumeration) is descriptive drift in the Round 8 Changes narrative, not in the implementation. The count + outcome are correct. Deferred per the dev discipline of weighing fix-vs-drift risk: forcing the sub-letter convention into the directive would break the natural sequential enumeration without adding clarity.
- The pattern observation in PLAN.md line 19 ("PLAN-QA-DISCIPLINE-R2 was added in round 7 but couldn't self-protect round 7's own absorption (discipline-added-in-round-N applies starting round-N+1). R8's plan-QA falsification should fully apply R2 to round 7's absorption + round 8's surgical edits.") captures a real meta-observation about plan-QA methodology: discipline rules apply *starting* from the round after they land, never retroactively. This is consistent with how Drop 4c.6.1's discipline rules have been propagating (NITs-first-class from R4→R5, PLAN-QA-DISCIPLINE-R1 from R5→R6, PLAN-QA-DISCIPLINE-R2 from R6→R7, R2-numeric-sub-clause from R7→R8).
- Sibling plan-QA-falsification round 8 firing in parallel — both must pass before Wave A heads (W0, W4.D1, W5, W6, W7.D1, **W8**) dispatch builders.
- Hylla feedback for this round appears in the closing comment (none — Round 8 review was Markdown-only on PLAN.md / PLAN_QA_PROOF.md / PLAN_QA_FALSIFICATION.md; no Hylla queries required and no Go code touched).

---

## Round 9 — Plan-QA-Proof Verdict

**Drop:** `4c.6.1`
**Round:** 9 (plan-QA-proof)
**Pass / Fail:** **PASS** (zero findings — minimal surgical absorption pass)

### Findings

Round 9 is a single-line surgical absorption of R8-FF1 from Round 8's plan-QA falsification PASS-WITH-ABSORB verdict, plus explicit DEFERRED-AS-NIT-with-reason language for three Round-8 NITs (R8-NIT1/2/3) and explicit REFUTED noting for one Round-8 self-flagged candidate (R8-FF2). All four dispositions carry explicit ABSORB / DEFERRED-AS-NIT-with-reason / REFUTED-with-reason language per the "NITs are first-class" discipline (memory `feedback_nits_are_first_class.md`). No silent drops.

- **R8-FF1 (locked-decisions line ~874 "~22 prompt files at `.tillsyn/agents/{go,fe}/`" mis-counts prompt total — load-bearing for future-drop architectural references)** — **ABSORB.** PLAN.md line 884 (post-Round-9 — was line 874 pre-insertion of the Round 9 Changes section) corrected to the falsifier's exact suggested rewording (PLAN_QA_FALSIFICATION.md:1254). The corrected text reads: `"Tillsyn-project-local prompts: 20 prompt files at `.tillsyn/agents/{go,fe}/` (10 per group) + `.tillsyn/bindings.json` + `.gitignore` re-includes (W8). Skip `gen/` per disposition 7.6."` Byte-perfect match to the suggested rewording. Internal consistency restored: line 884 now matches line 752 (acceptance: 20 prompts) and line 764 (spawn directive: D1-D20 prompts). **VERIFIED.**
- **R8-NIT1 (PLAN-QA-DISCIPLINE-R2 numeric-consistency sub-clause buried at end of long row — moderate-visibility positioning)** — **DEFERRED-AS-NIT** at PLAN.md line 17 with explicit reason: "R2 numeric sub-clause visibility is incremental methodology polish; not load-bearing for L2 spawn this drop. Future round may promote to separate refinement row." Reason names: (a) classification as incremental methodology polish, (b) non-blocking for L2 spawn, (c) explicit deferral path (future round promotion to separate R3 refinement row). **VERIFIED.**
- **R8-NIT2 (Round 7 Changes line 25 contains post-Round-8 numbering "D21" where R7 used "D19" — preserved-verbatim discipline strict-reading concern)** — **DEFERRED-AS-NIT** at PLAN.md line 18 with explicit reason: "Round 7 Changes retroactive D19→D21 substitution is in historically-consistent direction; reverting creates transient inconsistency for purity-only reasons." Reason names: (a) directional analysis (forward-propagation), (b) cost of reverting (transient inconsistency), (c) purity-vs-utility tradeoff weighted toward utility. **VERIFIED.**
- **R8-NIT3 (line ~73 "~22 prompts at `tillsyn/main/.tillsyn/agents/{go,fe}/`" same grammar as R8-FF1 — historical Round 3 narrative)** — **DEFERRED-AS-NIT** at PLAN.md line 19 with explicit reason: "line ~73 (Round 3 Changes block) has the same `~22 prompts at .tillsyn/agents/{go,fe}/` grammar as R8-FF1, but is historical Round 3 narrative; preservation discipline applies (per round-2 spawn-brief precedent for Round 6 line 34)." Reason names: (a) location classification (historical Changes block), (b) discipline applied (preservation per spawn-brief), (c) explicit precedent citation (Round 6 line 34 historical "19 prompt droplets" exemption). **VERIFIED.**
- **R8-FF2 (self-flagged "D8a/D8b/D8c shorthand vs D8/D9/D10 sequential D-list")** — **REFUTED** at PLAN.md line 20 with explicit reason: "D8a/D8b/D8c prose vs D8/D9/D10 sequential D-list is cosmetic shorthand; L2 D-list is authoritative." Reason names: (a) classification as cosmetic shorthand, (b) authoritative source (L2 D-list at line 764), (c) non-load-bearing nature. Aligns with falsifier's own REFUTED disposition. **VERIFIED.**

No new high/critical findings raised in Round 9. Zero NITs raised in this proof round.

### Evidence checked

1. **Round marker** — PLAN.md line 4: `**Round:** 9`. **VERIFIED.**
2. **Round 9 Changes section position** — PLAN.md lines 12–20, BEFORE Round 8 Changes (line 22). Newest-first ordering preserved. **VERIFIED.**
3. **Round 8/7/6/5/4/3/2 Changes preserved verbatim** — Round 8 Changes (lines 22–29), Round 7 Changes (lines 31–38), Round 6 Changes (lines 40–48), Round 5 Changes (lines 50–58), Round 4 Changes (lines 60–76), Round 3 Changes (lines 78–89), Round 2 Changes (lines 91–100) all preserved. Round 3 Changes line 83 retains historical `"~22 prompts at `tillsyn/main/.tillsyn/agents/{go,fe}/`"` per R8-NIT3 deferral. Round 6 Changes line 44 retains historical `"Other 19 W8 prompt droplets stay Wave A unblocked"` (preservation precedent intact). **VERIFIED.**
4. **R8-FF1 ABSORB language explicit + correct rewording at line 884** — PLAN.md line 884 reads: `"Tillsyn-project-local prompts: 20 prompt files at `.tillsyn/agents/{go,fe}/` (10 per group) + `.tillsyn/bindings.json` + `.gitignore` re-includes (W8). Skip `gen/` per disposition 7.6."`. Byte-perfect match to falsifier's suggested rewording at PLAN_QA_FALSIFICATION.md:1254. **VERIFIED.**
5. **R8-NIT1 DEFERRED-AS-NIT explicit with reason** — PLAN.md line 17 carries explicit "DEFERRED-AS-NIT — reason: …" prefix + falsifier's reasoning paraphrase. **VERIFIED.**
6. **R8-NIT2 DEFERRED-AS-NIT explicit with reason** — PLAN.md line 18 carries explicit "DEFERRED-AS-NIT — reason: …" prefix. **VERIFIED.**
7. **R8-NIT3 DEFERRED-AS-NIT explicit with reason** — PLAN.md line 19 carries explicit "DEFERRED-AS-NIT — reason: …" prefix + cites Round 6 line 34 historical exemption as preservation precedent. **VERIFIED.**
8. **R8-FF2 REFUTED explicit with reason** — PLAN.md line 20 carries explicit "REFUTED —" prefix with reason. **VERIFIED.**
9. **No prior-round Changes regression** — Round 8 Changes (lines 22–29) preserved verbatim post-Round-9. Round 7 Changes line 38 (PLAN-QA-DISCIPLINE-R2 captured) preserved. Round 6 Changes line 44 historical "19 W8 prompt droplets" preserved (precedent for R8-NIT3 deferral). Round 5 Changes lines 50-58 (R3-NIT2/NIT3/NIT4 dispositions + NITs-first-class process directive) preserved. Round 4 Changes lines 60-76 (W7 4-droplet R3-FF1 absorption) preserved. Round 3 Changes lines 78-89 preserved (including the historical "~22 prompts" wording at line 83). Round 2 Changes lines 91-100 (R2 absorptions) preserved. **VERIFIED.**
10. **Acyclic graph preserved** — Topo-sort unchanged from Round 8: `{W0, W4.D1, W5, W6, W7.D1, W8-non-smoke-droplets (D0 + D1-D20)}` → `{W1, W4.D2, W7.D2, W7.D4}` → `{W2, W7.D3, W8.D21}` → `{W3}`. Round 9 made zero structural edits to the wave graph or blocked_by edges. PLAN.md lines 826-836 (wave graph + parallelism notes) byte-identical to Round 8. **VERIFIED.**
11. **No Section 0 leakage** — `grep -n "^# Section 0\|^## Proposal\|^## Convergence" PLAN.md` returns no matches. Round 9 Changes section (lines 12–20) is descriptive narrative with no Section-0 pass titles. The only "Section 0" mention in PLAN.md is policy-prose at line 946 ("Section 0 SEMI-FORMAL REASONING in every subagent response, but Section 0 stays in the orchestrator-facing response — never in PLAN.md or QA files"). **VERIFIED.**
12. **Total droplet count holds at 22** — D0 (`.gitignore` + `bindings.json`) + D1-D20 (20 prompt-authoring droplets) + D21 (smoke-test) = 22 droplets total for W8 sub-plan. L2 spawn directive at PLAN.md line 774 enumerates all 22 droplets with explicit per-droplet file-name pairs for D1-D20 + dedicated D21 smoke test. Matches decomposition table line 150 "~22 build droplets". Matches W8 acceptance line 762 "All 20 prompt files exist (10 go + 10 fe)" (the 20-prompt subset of the 22 droplets). **VERIFIED.**
13. **Disposition language explicit (per dev discipline)** — Round 9 Changes uses explicit `ABSORB —` (R8-FF1, line 16), `DEFERRED-AS-NIT — reason:` (R8-NIT1/2/3, lines 17-19), and `REFUTED —` (R8-FF2, line 20) prefixes. Each carries an inline reason. No "judgment call" / "as-is" / "accepted" / "no change" / silent-drop phrases. Conforms to Round 5's process-change directive and `feedback_nits_are_first_class.md`. **VERIFIED.**
14. **Line-number narrative references use `~` approximation** — Round 9 Changes line 16 cites "line ~874" (pre-Round-9 line number; post-Round-9 actual line is 884 due to 10-line Round-9-Changes-section insertion at top). Round 9 Changes line 19 cites "line ~73" (pre-Round-9 line number; post-Round-9 actual line is 83). The "~" prefix is honest approximation; the falsifier's own verdict used the same pre-Round-9 numbers. Acceptable as audit-trail narrative. **VERIFIED.**

### Notes

- Round 9 is the smallest surgical absorption pass in this drop's history: 1 single-line fix at line 884 + 1 Round 9 Changes section addition. The minimality reflects the falsifier's PASS-WITH-ABSORB verdict (1 CONFIRMED + 3 DEFERRED + 1 REFUTED), where only the 1 CONFIRMED required actual textual edit. The 3 DEFERREDs + 1 REFUTED required only Round 9 Changes documentation per the "NITs are first-class" discipline.
- The R8-FF1 absorption demonstrates PLAN-QA-DISCIPLINE-R2's numeric-consistency sub-clause working as designed: the Round-8 plan-QA falsifier found a count-drift defect that Round 8's own R7-FF1 sweep missed; Round 9 absorbed surgically; the L1 narrative now triangulates 8+ surfaces all at "20 prompt files" (line 752, 762, 764, 774, 821, 823, 832, 884) with no further drift.
- The R8-NIT3 deferral is the structurally-cleanest example so far of historical-preservation discipline in action: line 83 (Round 3 Changes block) has the IDENTICAL grammatical defect as line 884 (locked decisions), but the LOAD-BEARING-NESS differs — line 884 is current-state load-bearing for future-drop architectural references; line 83 is read-only audit trail. Round 9 fixes the load-bearing one, preserves the historical one. The Round 6 line 44 "19 prompt droplets" precedent is cited explicitly.
- The R8-FF2 REFUTED disposition aligns with the falsifier's own REFUTED verdict. Both proof + falsification agree the D8a/D8b/D8c shorthand vs D8/D9/D10 sequential enumeration is cosmetic-only; the L2 D-list at line 774 is the authoritative source-of-truth and is unambiguous.
- This drop's plan-QA loop has now produced FOUR process-change-class methodology refinements: (1) "Specify the residue, not the deletion" from W7 inverted carving (Round 4), (2) "NITs are first-class" from R4→R5, (3) PLAN-QA-DISCIPLINE-R1 from R5→R6, (4) PLAN-QA-DISCIPLINE-R2 from R6→R7 (extended with numeric sub-clause R7→R8). Round 9 did NOT add a new methodology refinement — the existing R2 numeric sub-clause caught R8-FF1 successfully, and the R8-NIT1 polish to promote it to R3 was deferred as incremental.
- Sibling plan-QA-falsification Round 9 firing in parallel — both must converge PASS before Wave A heads (W0, W4.D1, W5, W6, W7.D1, **W8**) dispatch builders.
- Hylla feedback for this round appears in the closing comment (none — Round 9 review was Markdown-only on PLAN.md / PLAN_QA_PROOF.md / PLAN_QA_FALSIFICATION.md; no Hylla queries required and no Go code touched).
