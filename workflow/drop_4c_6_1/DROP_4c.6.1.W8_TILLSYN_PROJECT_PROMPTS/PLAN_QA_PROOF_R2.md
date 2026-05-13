# DROP_4c.6.1.W8 — PLAN-QA-PROOF (Round 2)

**Verdict:** PASS

Round-1 raised 2 FFs + 4 NITs (proof) and 2 FFs + 6 NITs (falsification). Round-2 planner absorbed all FFs and the 4 ABSORB-classed NITs; the 4 DEFERRED falsification NITs carry explicit rationale. No new FFs surfaced. Plan ships.

---

## 1. Findings

(none — all round-1 FFs + ABSORB NITs are resolved; no fresh FFs.)

---

## 2. Missing Evidence

(none — every round-2 change is verifiable against the on-disk PLAN.md + `_BLOCKERS.toml` + L1 PLAN R10-D4 directive at `workflow/drop_4c_6_1/PLAN.md:937` + render.go symbol shape at `render.go:327/646/860/877`.)

---

## 3. Round-2 Absorption Verification

### Proof FF1.1 / Fals FF2 (model values → bare aliases per R10-D4): RESOLVED ✓

- Common Model/Tools table at PLAN.md L98-110 lists bare aliases: `opus` (planning/qa-*/research, 7 roles), `sonnet` (builder, 1 role), `haiku` (commit-message, 1 role), `orchestrator-managed` (closeout + orchestrator-managed kinds, 2 roles).
- All 20 per-droplet AcceptanceCriteria spot-checked carry matching bare-alias `model:` bullets:
  - opus (12): D1/D3/D4/D5/D6/D7/D11/D13/D14/D15/D16/D17 → all `model: opus`.
  - sonnet (2): D2/D12 → `model: sonnet`.
  - haiku (2): D9/D19 → `model: haiku`.
  - orchestrator-managed (4): D8/D10/D18/D20 → `model: orchestrator-managed`.
- L1 PLAN.md L937 declares R10-D4 the locked decision: "W8 prompt frontmatter `model:` uses bare aliases ... matches live `~/.claude/agents/go-*-agent.md` system frontmatter."

### Proof FF1.2 (Hylla in tools): RESOLVED ✓

- Common table L98-106 lists `Read, Grep, Glob, Hylla` for all 6 Hylla-consuming roles (planning, plan-qa-proof, plan-qa-falsification, build-qa-proof, build-qa-falsification, research).
- Per-droplet AcceptanceCriteria spot-checked: D1, D3, D4, D5, D6, D7, D11, D13, D14, D15, D16, D17 all carry `tools: Read, Grep, Glob, Hylla` (12 droplets).
- Round-2 change note at L15 records the L1 PLAN.md L835 directive correctly: "Hylla-OFF applies to the current orchestration cycle only; authored prompts govern future dogfood when Hylla is operational."

### Proof NIT1 (D21 `binding.AgentName`): RESOLVED ✓

- D21 AcceptanceCriteria at L1000 now contains the explicit bullet: "Test sets `binding.AgentName = "builder-agent"` so rendered file path at `<bundle.Root>/plugin/agents/builder-agent.md` matches the asserted location. The filename is driven by `binding.AgentName` (per `render.go:327`), NOT by `SystemPromptTemplatePath`."
- Verified against render.go L327: `agentPath := filepath.Join(bundle.Paths.Root, pluginSubdir, agentsSubdir, binding.AgentName+".md")` — the filename source is `binding.AgentName`. Test will land file at the asserted path only if AgentName is set.

### Proof NIT2 + Fals FF1 (`_BLOCKERS.toml` naming): RESOLVED ✓

- On-disk `_BLOCKERS.toml` (verified at `_BLOCKERS.toml:13-122`): 22 entries, all use bare `D*` form. D0 head-row present (`node = "D0"`, `blocked_by = []`). D21 carries full cross-wave list `["D0","D1",...,"D20","4c.6.1.W1"]`. Header comment at L4-11 explicitly documents the convention.
- PLAN.md mirror at L1042-1155 also uses bare `D*` form. Round-2 change note at L17 records the canonicalization.
- Cross-wave reference `4c.6.1.W1` retains its fully-qualified form (graph-walker-friendly).

### Proof NIT3 (smart-quote risk): RESOLVED ✓

- CommonBuilderConstraint 7 at PLAN.md L79 added: "ASCII apostrophe in migration marker. The migration marker string uses an ASCII single-quote (U+0027) in `Tillsyn's` — do NOT let any editor or autocorrect convert it to a U+2019 right single quotation mark (curly/smart quote). Build-QA greps for the exact verbatim string; a U+2019 substitution silently breaks the grep check."

### Proof NIT4 (`extends_path` CWD): RESOLVED ✓

- D0 ContextBlocks at PLAN.md L195 now contains `reference` block documenting the loader CWD invariant: "`extends_path` resolves correctly ONLY when the loader's CWD is `tillsyn/main/.tillsyn/` (three `..` steps reach the directory containing both `tillsyn/` and `stil/`). W5 (Go TUI keybinding loader) and W6 (FE JS loader) build droplets MUST honor this CWD invariant when reading this file. If a loader runs from a different CWD, the path will silently miss. W5/W6 builders: verify your loader's CWD or resolve the path relative to the file location, not the process CWD."

### Fals NIT1 (~/tmp typo): RESOLVED ✓

- D21 RiskNote at L1014: "Do NOT use `/tmp/tillsyn/main` as the RepoPrimaryWorktree value — use `t.TempDir()` for proper test isolation." Leading `~/` removed; matches the literal hardcoded path in `fixtureProject()`.

### Fals NIT2 (D0 head-row in `_BLOCKERS.toml`): RESOLVED ✓

- Both on-disk `_BLOCKERS.toml:13-16` AND PLAN.md mirror L1046-1049 carry explicit `[[blockers]] node = "D0" blocked_by = []` row with reason "Wave A head — no upstream blockers".

### Fals NIT3, NIT4, NIT5, NIT6: DEFERRED-WITH-REASON ✓

- Documented at PLAN.md L22-25 with named rationale per finding:
  - NIT3 ("visibly DIFFERENT" qualitative wording): precise diff IS in next paragraph; quantitative metric risks over-rigidifying.
  - NIT4 (WORKFLOW.md §"Phase 7 — Closeout" reference): builder verifies section header at Read-time; low-risk.
  - NIT5 (`extends_path` loader robustness): covered by Proof NIT4 ContextBlock; full hardening needs W5/W6 coordination.
  - NIT6 (PLAN-QA-DISCIPLINE-R2 count sync methodology): promote to separate refinement row at drop closeout.

---

## 4. Other Proof Checks

### Droplet count (PLAN-QA-DISCIPLINE-R2): 22 ✓

- L8 narration: "1 + 20 + 1 = 22".
- Enumerated D-list at L1161: D0, D1, ..., D20, D21 = 22 entries.
- `_BLOCKERS.toml` enumeration: 22 entries (D0 head + D1-D20 prompt droplets + D21 smoke-test).
- Wave-graph at L40-65 enumerates D0 + D1-D20 + D21 = 22.
- All four count sources agree.

### `_BLOCKERS.toml` ↔ PLAN.md mirror consistency ✓

- Both files use bare `D*` form. Both contain D0 head-row. Both place D21 with the full cross-wave `blocked_by` list including `4c.6.1.W1`. PLAN.md mirror (L1042-1155) and on-disk `_BLOCKERS.toml` (L13-122) are graph-equivalent. Reason strings differ in wording (mirror uses terser "same as D1" for D2-D20; on-disk uses per-droplet specifics for D1-D20) but PLAN.md is declared truth (L1040), and the graph adjacency is identical across both surfaces.

### Migration marker discipline ✓

- CommonBuilderConstraint 5 at L77-78 declares the verbatim marker string. CommonBuilderConstraint 7 at L79 forbids smart-quote conversion. Each prompt droplet's AcceptanceCriteria includes "Migration marker present" bullet (D1 L225, D2 L280, D3 L326, D4 L369, D5 L406, D6 L442, D7 L478, D8 L514, D9 L560, D10 L599, D11 L638, D12 L674, D13 L710, D14 L745, D15 L780, D16 L815, D17 L850, D18 L883, D19 L918, D20 L953).

### Plan-QA vs Build-QA differentiation (D3↔D5, D4↔D6, D13↔D15, D14↔D16) ✓

- D3 (plan-qa-proof): Evidence Sources = PLAN.md / REVISION_BRIEF.md / SKETCH.md; What To Check = blocked_by graph / paths-packages / acceptance / structural_type (L320-330).
- D5 (build-qa-proof): Evidence Sources = Go source / `git diff` / `mage test-pkg` output / PLAN.md; What To Check = test pass rates / coverage / paths / acceptance / mage ci green (L408-410).
- D4 (plan-qa-falsification): Evidence Sources = PLAN.md / REVISION_BRIEF.md / SKETCH.md; Attack Vectors = missing blockers / cycles / drift / structural violations / scope creep (L371-373).
- D6 (build-qa-falsification): Evidence Sources = Go source / `git diff` / test output; Attack Vectors = counterexamples / race / edge cases / false-positive tests / security (L444-446).
- FE pairs (D13↔D15, D14↔D16) mirror the same asymmetry. Each pair carries an explicit `constraint (high): Must NOT be near-identical to D*` ContextBlock.

### FROM-SCRATCH droplets (D8, D9, D10, D18, D19, D20): 6 flagged ✓

- D8 L506: "NO SOURCE FILE EXISTS at `~/.claude/agents/closeout-agent.md` — author FROM SCRATCH" + explicit citation list (CLAUDE.md §"Cascade Tree Structure", WORKFLOW.md §"Phase 7 — Closeout", WIKI.md, CLAUDE.md §"Cascade Ledger + Hylla Feedback").
- D9 L552: "NO SOURCE FILE at `~/.claude/agents/commit-message-agent.md` — author FROM SCRATCH" + citations (CLAUDE.md §"Git Commit Format", §"Build-QA-Commit Discipline", memory `feedback_commit_style.md`).
- D10 L590: "NO SOURCE FILE — author FROM SCRATCH" + citations (CLAUDE.md §"Orchestrator-as-Hub Architecture", §"Cascade Tree Structure", WORKFLOW.md, WIKI.md).
- D18 L877: "NO SOURCE FILE — author FROM SCRATCH" + reference to D8 structural template.
- D19 L912: "NO SOURCE FILE — author FROM SCRATCH" + reference to D9.
- D20 L947: "NO SOURCE FILE — author FROM SCRATCH" + reference to D10.

### D21 path verified against W1 round-2 ✓

- D21 fixture path: `<tmpdir>/.tillsyn/agents/go/builder-agent.md` (subdir-per-group).
- W1 round-2 PLAN.md L188-191 confirms W1 D3 changes `readProjectTierAgent` from `(projectWorktree, basename)` → `(projectWorktree, group, basename)`. W1 D3 also renames constants `agentBodyDefaultGroup` from `"till-go"` → `"go"` and `agentBodyFallbackGroup` from `"till-gen"` → `"gen"`.
- Current pre-W1 resolver at `render.go:877-890` joins flat path `(projectWorktree, projectAgentsSubdir, basename)` — would miss the `go/` subdir. D21 correctly blocks on `4c.6.1.W1`.
- D21 RepoPrimaryWorktree via `t.TempDir()`; `SystemPromptTemplatePath = "go/builder-agent.md"` triggers `resolveAgentGroup` to return `"go"` (per render.go L860-867); `binding.AgentName = "builder-agent"` lands rendered file at `<bundle.Root>/plugin/agents/builder-agent.md` (per render.go L327).

### Migration marker on every prompt ✓

- Verbatim string at CommonBuilderConstraint 5: `<!-- Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8. -->`. Every per-droplet AcceptanceCriteria carries "Migration marker present" bullet.

### LSP-verifiable render.go symbols ✓

- `validateBundle` at L326 — confirms `agentPath` derived from `binding.AgentName`.
- `assembleAgentFileBody` at L646 — current call site for `readProjectTierAgent` is `(project.RepoPrimaryWorktree, basename)` (L666); W1 changes to `(project.RepoPrimaryWorktree, group, basename)`.
- `resolveAgentGroup` at L860 — derives group from `binding.SystemPromptTemplatePath` via `path.Dir`.
- `readProjectTierAgent` at L877 — current signature `(projectWorktree, basename string)`; W1 changes to `(projectWorktree, group, basename string)`.

### Bindings.json shape ✓

- 5 commands: `dispatch`, `plan`, `archive`, `settings`, `help` (PLAN.md L165-170).
- Verified against stil baseline at `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json:100-108`: baseline `product_extensions.tillsyn.commands` = `new-drop`, `complete-drop`, `handoff`, `comment`. No ID collision. Union = 9.

### `.gitignore` re-include order ✓

- Current `.gitignore` L11-19: comment block explaining `.tillsyn/*` (contents-not-dir) pattern; current re-include `!.tillsyn/template.toml`.
- PLAN.md L146-150 specifies adding three lines AFTER `!.tillsyn/template.toml`: `!.tillsyn/agents/` (dir un-exclude), `!.tillsyn/agents/**/*.md` (file glob), `!.tillsyn/bindings.json`. Order matches gitignore semantics: dir un-exclude must precede the glob.

### Mage discipline ✓

- CommonBuilderConstraint 1-2 at L73-74: never `mage install`; never raw `go test`/`go build`/`go vet`. Always `mage <target>`.
- Every droplet specifies `mage ci` (D0-D20) or `mage test-pkg ./internal/app/dispatcher/cli_claude/render` then `mage ci` (D21).

### Single-line commit constraint ✓

- CommonBuilderConstraint 3 at L75 enforces single-line conventional commits ≤72 chars, no body.

---

## 5. Section 0 — Orchestrator-Facing

# Section 0 — SEMI-FORMAL REASONING

## Proposal

- **Premises:** Round-2 planner absorbed every round-1 FF (proof FF1.1 + FF1.2; fals FF1 + FF2) and every ABSORB-classed NIT (proof NIT1-NIT4; fals NIT1, NIT2). Round-1 already cleared structural concerns (droplet count, wave graph, blocker topology, FROM-SCRATCH coverage, QA differentiation, bindings.json shape, `.gitignore` re-include order). Round-2 surface area is field-level edits only — no structural changes.
- **Evidence:** Direct reads of PLAN.md (1172 lines), `_BLOCKERS.toml` (122 lines), L1 PLAN.md L700-937 (W8 directive + R10-D4 locked decision at L937), W1 round-2 PLAN.md L1-40 + L188-204 (resolver signature change confirmation), `render.go` L320-337 / L640-693 / L850-867 / L870-890, `.gitignore` L11-19, `stil/main/src/bindings/baseline.json` L100-108. Round-1 verdicts read in full.
- **Trace or cases:** Each round-1 finding tracked through PLAN.md to its absorption locus. Each per-droplet `model:` and `tools:` AcceptanceCriterion spot-checked. D21 path triplet (fixture, RepoPrimaryWorktree, AgentName) verified against render.go locations. `_BLOCKERS.toml` enumeration confirmed.
- **Conclusion:** PASS. No FFs remain. No new FFs surfaced. 4 deferred fals NITs carry explicit rationale per `feedback_nits_are_first_class.md` discipline.
- **Unknowns:** None.

## QA Proof

- **Premises:** Round-2 changes are complete and traceable; every locked round-1 finding has a clear absorption locus or explicit deferral rationale; plan still satisfies PLAN-QA-DISCIPLINE-R1/R2.
- **Evidence:** Direct file reads above; line-precise pointers in §3 + §4.
- **Trace or cases:** 8 absorption claims (FF1.1, FF1.2, NIT1-NIT4 proof; NIT1, NIT2 fals) each pinned to PLAN.md line numbers. 4 deferred NITs each pinned to a deferral rationale at PLAN.md L22-25. 20 per-droplet `model:` + `tools:` spot-checks documented.
- **Conclusion:** Evidence is complete; verdict supported.
- **Unknowns:** None.

## QA Falsification

- **Premises:** Try to break the PASS verdict.
- **Attack vectors attempted:**
  - *Attack — round-2 missed a per-droplet `model:` bullet update:* spot-checked all 20 prompts (D1-D20). All carry the post-R10-D4 bare aliases. REFUTED.
  - *Attack — round-2 introduced new FF (e.g., Hylla added to commit-message-agent's tools):* checked D9/D19 (commit-message) — `tools: Read` only (no Hylla — correct for commit-only mechanical role). REFUTED.
  - *Attack — `_BLOCKERS.toml` D0 head-row missing in PLAN.md mirror despite being added to on-disk:* checked PLAN.md L1046-1049 — explicit `[[blockers]] node = "D0"` row present. REFUTED.
  - *Attack — D21 `binding.AgentName` bullet absent or wrong value:* checked PLAN.md L1000 — bullet present with explicit `"builder-agent"` value, render.go L327 line citation. REFUTED.
  - *Attack — D21 fixture path doesn't match W1 post-change resolver:* W1 round-2 confirmed signature `(projectWorktree, group, basename)`; D21 fixture at `<tmpdir>/.tillsyn/agents/go/builder-agent.md` is the post-W1 form. D21 blocks on `4c.6.1.W1`. REFUTED.
  - *Attack — smart-quote CommonBuilderConstraint forbids the wrong character:* checked PLAN.md L79 — forbids U+2019 conversion, requires U+0027 ASCII. Correct. REFUTED.
  - *Attack — D0 ContextBlock for extends_path CWD is too vague to act on:* checked PLAN.md L195 — names the exact invariant (CWD = `tillsyn/main/.tillsyn/`, three `..` steps), routes W5/W6 builders explicitly. REFUTED.
  - *Attack — bare-alias `orchestrator-managed` is not a real model:* L1 PLAN.md L937 explicitly enumerates `model: orchestrator-managed` as a closeout/orchestrator-managed kind value; the round-2 change note at PLAN.md L14 confirms "`model: orchestrator-managed` is a string value indicating orchestrator-managed scope — matches Tillsyn's orchestrator-managed-role convention." This is documentation-grade — `closeout` and `orchestrator-managed` kinds are orchestrator-handled, not model-dispatched. The string value is a stable convention, not a Claude API model name. REFUTED-AS-INTENTIONAL.
  - *Attack — 4 deferred fals NITs are not legitimately low-risk:* each carries a rationale tied to either (a) round-trip overhead vs builder-self-verify (NIT4 WORKFLOW header), (b) over-rigidification risk (NIT3 visibly-different), (c) cross-wave scope (NIT5 extends_path), (d) methodology-refinement-promotion (NIT6 count sync). Each deferral rationale is consistent with `feedback_nits_are_first_class.md` discipline ("Only skip with explicit reason"). REFUTED.
  - *Attack — droplet count drifted off 22:* L8 = 22; L1161 enumeration = 22; `_BLOCKERS.toml` = 22; wave-graph = 22. All four agree. REFUTED.
- **Conclusion:** No unmitigated counterexample to PASS.
- **Unknowns:** None.

## Convergence

- (a) QA Falsification produced no unmitigated counterexample to the PASS verdict.
- (b) QA Proof confirmed evidence completeness — every round-2 absorption claim is traceable to PLAN.md / `_BLOCKERS.toml` / render.go / L1 PLAN.md / W1 round-2 PLAN.md / stil baseline.
- (c) Remaining unknowns are routed: 4 deferred fals NITs each carry explicit deferral rationale; nothing is left dangling.

Converged.

---

## 6. Hylla Feedback

N/A — action item touched non-Go files only (PLAN.md, `_BLOCKERS.toml`). The Go file referenced (`render.go`) was read at known line numbers for symbol-shape verification, not Hylla-queried. Hylla is OFF per spawn directive.

---

## TL;DR

- **T1:** No new FFs surfaced.
- **T2:** No missing evidence; every claim is traceable to file:line.
- **T3:** 8 round-1 ABSORB items confirmed at PLAN.md absorption loci; 4 deferred fals NITs carry explicit deferral rationale.
- **T4:** Droplet count (22), wave graph, `_BLOCKERS.toml` mirror, migration markers, QA differentiation, FROM-SCRATCH set, D21 path triplet, bindings.json union, `.gitignore` order, mage discipline, single-line commits — all verified.
- **T5:** Section 0 5-pass certificate converged: proposal/proof/falsification/convergence; PASS verdict supported by direct file evidence; deferred NITs justified.
- **T6:** Hylla feedback N/A — non-Go files only.
