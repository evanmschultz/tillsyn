# DROP_4c.6.1.W8 — PLAN-QA-PROOF (Round 1)

**Verdict:** PASS-WITH-NITS

Plan structure is sound and trace-complete. Two **FF (must-fix-before-build)** drifts found, plus 4 NITs. None of the FFs invalidate the wave graph or atomic-decomposition shape; all are field-level corrections the L2 planner should patch.

---

## 1. Findings

### FF (must-fix-before-build)

- 1.1 [Axis: specify-block-well-formedness] [severity: high] **`model:` frontmatter values disagree with both the active system-agent convention AND `feedback_cascade_model_policy.md` memory.** PLAN.md L80-89 prescribes `claude-opus-4-5` / `claude-sonnet-4-5` / `claude-haiku-4-5`. Verified against `~/.claude/agents/go-*-agent.md` frontmatter: existing agents use bare names (`model: sonnet` / `model: opus`). `feedback_cascade_model_policy.md` lists fully versioned forms `claude-sonnet-4-6` / `claude-opus-4-7` / `claude-haiku-4-5-20251001`. Neither matches the L2 plan's values. Builders authoring 20 files with a wrong model spec ship 20 broken artifacts. → fix_hint: pick one canonical list (preferred: bare `sonnet`/`opus`/`haiku` mirroring the live system agents) and update both the common table (L80-89) and every per-droplet AcceptanceCriterion (D1-D20 each cite their `model:` value inline). Document the choice in a CommonFrontmatter NIT-resolution note.

- 1.2 [Axis: spec-conformance] [severity: high] **`Hylla` tool omitted from QA / research / planning tools list.** L1 PLAN.md L787 directive: `Tools per role: qa-proof/qa-falsification/research/planning → Read, Grep, Glob, Hylla`. W8 PLAN.md L80-86 omits Hylla for all four roles (lists only `Read, Grep, Glob`). Builders authoring the 14 prompts (D1, D3, D4, D5, D6, D7, D11, D13, D14, D15, D16, D17 plus the 2 plan-QA proof + falsification in each group) using this table will ship prompts without Hylla in the tools allowlist — violating the L1 directive. → fix_hint: append `Hylla` (or the explicit MCP server slug, e.g., `mcp__hylla__*`) to the tools value for `planning-agent`, `plan-qa-proof-agent`, `plan-qa-falsification-agent`, `build-qa-proof-agent`, `build-qa-falsification-agent`, `research-agent` rows; update every per-droplet `tools:` AcceptanceCriterion accordingly. Decide before build whether the canonical token is `Hylla` (matches L1 directive prose), `hylla_*`, or the full MCP-namespace prefix (matches actual runtime tool naming).

### NIT (first-class per `feedback_nits_are_first_class.md` — address unless explicit skip rationale)

- 1.3 [Axis: specify-block-well-formedness] [severity: medium] **D21 `binding.AgentName` value left unspecified.** PLAN.md L975-979 spells out `SystemPromptTemplatePath = "go/builder-agent.md"` and asserts the rendered file at `<bundle.Root>/plugin/agents/builder-agent.md`. Verified in `render.go:327`: `agentPath := filepath.Join(bundle.Paths.Root, pluginSubdir, agentsSubdir, binding.AgentName+".md")` — the filename is driven by `binding.AgentName`, NOT by `SystemPromptTemplatePath`. To land the file at `builder-agent.md`, the test must set `binding.AgentName = "builder-agent"`. PLAN never says so. → fix_hint: add an explicit AcceptanceCriterion bullet: "Test sets `binding.AgentName = \"builder-agent\"` so the rendered file path matches the asserted location." This single missing field is the most likely cause of a builder confusion round.

- 1.4 [Axis: atomic-decomposition] [severity: low] **`_BLOCKERS.toml` ID convention differs between the on-disk file and the in-PLAN mirror.** On-disk `_BLOCKERS.toml` (verified) uses `W8.D1`, `W8.D2`, …; the inline `_BLOCKERS.toml Mirror` section in PLAN.md L1022-1124 uses bare `D1`, `D2`, …. Both shapes are internally consistent within their own file. PLAN.md states "PLAN.md is truth" (L1020), so the on-disk version's prefixed form is the durable one, but the in-PLAN mirror reads as if it diverges. → fix_hint: pick one convention and align both surfaces. Recommended: keep `W8.D*` in the on-disk file (preserves namespacing across waves) and update the in-PLAN mirror to match, OR explicitly document in the mirror that "the on-disk file uses `W8.D*` prefixes; this mirror uses bare `D*` IDs for local readability."

- 1.5 [Axis: spec-conformance] [severity: low] **Migration marker string contains a smart apostrophe.** L58, L204, and every per-droplet AcceptanceCriterion ("Migration marker present") cite the verbatim string `Tillsyn-project-local; lifted from ~/.claude/agents/ and adapted for Tillsyn's workflow. Future projects use embedded defaults shipped in Drop 4c.8.` The apostrophe in `Tillsyn's` is the ASCII `'` (verified by raw byte check). This is fine, but the per-droplet AcceptanceCriteria use the exact same string everywhere and build-QA will likely grep — make sure the builder doesn't auto-convert to U+2019 smart quote at write time. → fix_hint: add a CommonBuilderConstraint bullet: "Migration marker string uses ASCII single-quote in `Tillsyn's` — do not let any editor / autocorrect convert to U+2019."

- 1.6 [Axis: spec-conformance] [severity: low] **`extends_path` in `.tillsyn/bindings.json` is a relative path that breaks under any loader run from a non-`tillsyn/main/.tillsyn/` CWD.** PLAN.md L140 + L168 sets `extends_path = "../../../stil/main/src/bindings/baseline.json"`. RiskNote already flags this but offers no robustness directive. W5/W6 loaders consume the file at runtime; their CWD is not pinned. → fix_hint: either (a) add an explicit ContextBlock declaring the loader CWD invariant the path depends on (so W5/W6 build droplets honor it), or (b) decide now whether to make the value a marker like `${TILLSYN_ROOT}/../stil/main/src/bindings/baseline.json` with a resolver step. Don't fix in this drop, but record the decision so W5/W6 don't ship a broken loader.

---

## 2. Missing Evidence

(none — every claim in the L2 PLAN.md is traceable to one of: L1 PLAN.md W8 section, REVISION_BRIEF §2.18-2.20, SKETCH §3 / §10, the current `.gitignore` shape, the stil baseline, or render.go's `assembleAgentFileBody` / `readProjectTierAgent`.)

---

## 3. Trace Coverage Confirmation

- **Droplet count (PLAN-QA-DISCIPLINE-R2):** 22 droplets enumerated D0..D21. Narrative count (L8: "1 + 20 + 1 = 22") matches enumeration (L1132). ✔
- **Wave graph:** D0 (Wave A head, no blockers); D1-D20 each `blocked_by: D0` (Wave A, parallel); D21 `blocked_by: D0, D1, …, D20, 4c.6.1.W1` (Wave C). Verified acyclic by topo-sort. ✔
- **PLAN-QA-DISCIPLINE-R1 (D21 ties to W1 consumer):** D21 smoke-test asserts subdir-per-group resolver behavior — `<tmpdir>/.tillsyn/agents/go/builder-agent.md` (subdir form) — which is the shape W1 introduces. Current `readProjectTierAgent` is flat (`render.go:881` joins `projectWorktree + projectAgentsSubdir + basename` with no group segment). D21's `blocked_by` list explicitly includes `4c.6.1.W1`. ✔
- **`_BLOCKERS.toml` mirror:** on-disk file matches PLAN.md inline graph for `blocked_by` adjacencies; ID-prefix drift documented as NIT 1.4. ✔
- **FROM-SCRATCH droplets (D8, D9, D10, D18, D19, D20):** each is explicitly tagged FROM SCRATCH in PLAN.md (L30-32, L40-42), each cites the substitute source set (CLAUDE.md, WORKFLOW.md, WIKI.md, memory entries). Verified against `~/.claude/agents/` — only 10 system files present, none named `closeout-agent.md`, `commit-message-agent.md`, or `orchestrator-managed.md` (or their `fe-` variants). ✔
- **Plan-QA vs Build-QA differentiation (D3 vs D5; D4 vs D6; D13 vs D15; D14 vs D16):** each pair declares distinct Evidence Sources (PLAN/REVISION_BRIEF/SKETCH vs Go source/git diff/test output) and distinct What-To-Check / Attack-Vectors sections per SKETCH §3.1. ✔
- **Bindings 5-command shape:** PLAN.md L145-149 lists `dispatch`, `plan`, `archive`, `settings`, `help`. Verified against stil baseline `product_extensions.tillsyn.commands` (L100-108 of `/Users/evanschultz/Documents/Code/hylla/stil/main/src/bindings/baseline.json`): baseline ships `new-drop`, `complete-drop`, `handoff`, `comment` — disjoint from local's 5 IDs. Union = 9. ✔
- **`close` retirement:** PLAN.md L155 explicitly drops `close` (redundant with stil's canonical `complete-drop`). Matches SKETCH §10 row "Vim bindings merge semantic" + REVISION_BRIEF §2.19. ✔
- **`.gitignore` re-include pattern correctness:** Verified `.gitignore` L18 uses `.tillsyn/*` (excludes top-level contents, NOT the directory itself), per the in-file comment block L11-17. Plan correctly orders `!.tillsyn/agents/` before `!.tillsyn/agents/**/*.md` so the directory re-include precedes the glob. ✔
- **D21 smoke-test path shape:** PLAN.md L977-978 places fixture at `<tmpdir>/.tillsyn/agents/go/builder-agent.md` (subdir-per-group). Verified post-W1 resolver shape matches; pre-W1 resolver would miss. D21 blocks on W1 — correct. ✔
- **Render.go API references:** D21 ContextBlocks cite `readProjectTierAgent`, `assembleAgentFileBody`, `resolveAgentGroup` — all verified in `internal/app/dispatcher/cli_claude/render/render.go` (L877, L646, L860 respectively). ✔
- **Mage discipline:** every droplet specifies `mage ci` (D0-D20) or `mage test-pkg ./internal/app/dispatcher/cli_claude/render` then `mage ci` (D21). CommonBuilderConstraints L54-55 forbid `mage install` and raw `go` invocations. ✔
- **Single-line commit constraint:** CommonBuilderConstraints L56 enforces ≤72 char single-line conventional commits. Matches `feedback_commit_style.md`. ✔

---

## 4. Section 0 — Orchestrator-Facing

# Section 0 — SEMI-FORMAL REASONING

## Proposal

- **Premises:** L2 PLAN.md decomposes W8 into 22 atomic droplets; satisfies PLAN-QA-DISCIPLINE-R1/R2; aligns with REVISION_BRIEF §2.18-2.20, SKETCH §3 + §10, and L1 PLAN.md W8 directive lines 706-846.
- **Evidence:** Direct reads of PLAN.md (1142 lines), `_BLOCKERS.toml`, REVISION_BRIEF §2.18-2.20, SKETCH §3 + §10, `.gitignore`, `/stil/main/src/bindings/baseline.json`, `render.go` (`readProjectTierAgent` L877, `assembleAgentFileBody` L646, `resolveAgentGroup` L860, `validateBundle` L327), `render_test.go` fixtures, `~/.claude/agents/` listing (10 files; no closeout/commit-message/orchestrator-managed), `feedback_cascade_model_policy.md`, project CLAUDE.md.
- **Trace or cases:** 22-droplet enumeration verified; wave graph topo-sorted acyclic; PLAN-QA-R1 D21↔W1 link verified against current FLAT resolver; bindings union (4 stil + 5 local) verified; FROM-SCRATCH set (6 droplets) verified against `~/.claude/agents/` ground truth; QA-vs-build-QA differentiation verified per SKETCH §3.1.
- **Conclusion:** PASS-WITH-NITS. Two FFs (model values, Hylla tools omission) must land before build; 4 NITs should land but don't block dispatch.
- **Unknowns:** Whether the dev prefers `model:` bare names (matches current system agents) vs versioned IDs (matches memory) is a routing decision for the orchestrator/dev, not a planner choice.

## QA Proof

- **Premises:** Each acceptance criterion in D0-D21 is testable; each FROM-SCRATCH droplet has explicit citation list; each QA file specifies differentiated Evidence Sources + What-To-Check / Attack-Vectors; D21's W1 dependency is explicit.
- **Evidence:** Direct file reads (see above); cross-references to L1 PLAN.md sections at known line numbers.
- **Trace or cases:** Every droplet inspected for: AcceptanceCriteria yes/no testability, paths declared, packages declared, blocked_by present (or explicitly empty for Wave A head), KindPayload `changes` array shape, mage target named.
- **Conclusion:** Evidence is complete for the verdict. Findings recorded.
- **Unknowns:** None.

## QA Falsification

- **Premises:** Try to break the PASS-WITH-NITS verdict.
- **Evidence + attack vectors attempted:**
  - *Attack — D21 missing AgentName makes build impossible:* the test could fail or the builder could guess wrong. Mitigated: recorded as NIT 1.3 with explicit fix.
  - *Attack — model values are wrong and break all 20 prompts:* verified — recorded as FF 1.1. Builder MUST patch before authoring.
  - *Attack — Hylla missing from tools list makes QA prompts non-functional:* verified — recorded as FF 1.2.
  - *Attack — gitignore order constraint is wrong:* tested via in-file comment block (L11-17) explaining `.tillsyn/*` (children-not-dir) pattern; the proposed `!.tillsyn/agents/` + `!.tillsyn/agents/**/*.md` ordering is correct.
  - *Attack — bindings.json `extends_path` is fragile under arbitrary CWD:* real risk but doesn't block W8 build — recorded as NIT 1.6 routed at W5/W6.
  - *Attack — D21 fixture path mismatches actual W1 shape:* verified W1 changes `readProjectTierAgent` from `(projectWorktree, basename)` to `(projectWorktree, group, basename)`; fixture at `.tillsyn/agents/go/builder-agent.md` matches the post-W1 form. D21 correctly blocks on W1.
  - *Attack — count "1+20+1=22" doesn't match enumeration:* verified; the D-list (D0..D21) is 22 entries.
  - *Attack — stil baseline ID collision between local 5 and baseline 4:* verified — local IDs (`dispatch`, `plan`, `archive`, `settings`, `help`) are disjoint from baseline's (`new-drop`, `complete-drop`, `handoff`, `comment`). Note: `archive` exists in `ro-email` product_extensions but those are different namespaces; no collision in `tillsyn` ns.
  - *Attack — `close` should still be present:* verified — explicitly dropped in PLAN.md L155 and in SKETCH §10 "Vim bindings merge semantic" row. Canonical.
- **Conclusion:** No unmitigated counterexample to PASS-WITH-NITS. Both FFs are field-level and named precisely.
- **Unknowns:** None.

## Convergence

- (a) QA Falsification produced no unmitigated counterexample to the PASS-WITH-NITS verdict.
- (b) QA Proof confirmed evidence completeness — every claim in the L2 PLAN is traceable.
- (c) Remaining unknowns are routed: model-value choice → dev/orch decision; extends_path robustness → W5/W6 loader concern logged via NIT 1.6.

Converged.

---

## 5. Hylla Feedback

N/A — action item touched non-Go files only (PLAN.md, _BLOCKERS.toml, gitignore, JSON, MD prompts). The lone Go file referenced (`render.go`) was read for symbol-shape verification, not Hylla-queried. Hylla is OFF per spawn directive.
