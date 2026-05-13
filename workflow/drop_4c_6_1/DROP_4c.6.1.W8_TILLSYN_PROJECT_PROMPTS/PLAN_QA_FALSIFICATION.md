# W8 Plan-QA Falsification Verdict

**Verdict:** PASS-WITH-ABSORB
**Author:** go-qa-falsification-agent (retry after API error)
**Date:** 2026-05-12
**Plan reviewed:** `workflow/drop_4c_6_1/DROP_4c.6.1.W8_TILLSYN_PROJECT_PROMPTS/PLAN.md`
**Sibling _BLOCKERS reviewed:** `workflow/drop_4c_6_1/DROP_4c.6.1.W8_TILLSYN_PROJECT_PROMPTS/_BLOCKERS.toml`
**L1 cross-ref:** `workflow/drop_4c_6_1/PLAN.md` lines 706-846, R3-NIT5 GUIDANCE block.

---

## Attack Surface Coverage

Every attack vector from the spawn prompt was attempted:

| # | Attack | Result |
|---|---|---|
| 1 | Missing `blocked_by` enumeration in D21 | EXHAUSTED — 22 entries match (D0-D20 + 4c.6.1.W1) |
| 2 | PLAN.md ↔ `_BLOCKERS.toml` drift | CONFIRMED — see FF1 below |
| 3 | YAGNI on 22 droplets / D0 split / batching | REFUTED — file-disjointness mandates one droplet per file; D0 atomicity bound by .gitignore commit gate |
| 4 | PLAN-QA-DISCIPLINE-R1 new-behavior coverage | EXHAUSTED — only D21 asserts NEW resolver behavior; D21 `blocked_by 4c.6.1.W1` present |
| 5 | PLAN-QA-DISCIPLINE-R2 numeric: narrated 22 vs enumerated | REFUTED — D-list at line 1132 and count at line 8 both = 22 |
| 6 | From-scratch sufficiency (D8/D9/D10/D18/D19/D20) | REFUTED — citation lists name CLAUDE.md sections, WORKFLOW.md phase, WIKI.md, specific memory entries; sufficient |
| 7 | D3-vs-D5 / D13-vs-D15 near-identical risk | REFUTED-WITH-NIT — see NIT3 |
| 8 | D21 fixture path matches W1 resolver shape | REFUTED — fixture at `<tmpdir>/.tillsyn/agents/go/builder-agent.md` matches W1's post-change resolver |
| 9 | bindings.json ID collision with stil baseline | REFUTED — local IDs {dispatch, plan, archive, settings, help} disjoint from stil's {new-drop, complete-drop, handoff, comment} |
| 10 | `.gitignore` re-include ordering correctness | REFUTED — parent uses `.tillsyn/*` (wildcard); explicit `!.tillsyn/agents/` dir un-exclude is required per git docs and present |
| 11 | 20-prompt parallel dispatch file-disjointness | REFUTED — each D1-D20 declares exactly one path; zero overlap |
| 12 | Migration marker per-droplet hard gate | REFUTED — every D1-D20 AcceptanceCriteria carries "Migration marker present" as bullet |
| 13 | Section 0 leakage acceptance bullet | REFUTED — explicitly forbidden in Common Builder Constraints §4 + every droplet's AcceptanceCriteria |
| 14 | Cross-wave bindings.json schema consistency | REFUTED — `schema_version: 1` + `product_extensions.tillsyn.commands` shape matches W5 (line 100) and W6 (line 556) consumer expectations |
| 15 | Model field correctness vs cascade-model-policy memory | CONFIRMED — see FF2 below (stale `claude-opus-4-5` / `claude-sonnet-4-5` / `claude-haiku-4-5` model IDs) |

---

## Findings

### FF1 — `_BLOCKERS.toml` ↔ PLAN.md mirror node-naming drift (CONFIRMED)

**Disposition:** ABSORB.

The actual `_BLOCKERS.toml` file uses fully-qualified node IDs `W8.D0` / `W8.D1` / ... / `W8.D21`:

```toml
node = "W8.D1"
blocked_by = ["W8.D0"]
```

The mirror block inside PLAN.md (lines 1019-1126) uses bare `D0` / `D1` / ... / `D21`:

```toml
node = "D1"
blocked_by = ["D0"]
```

Two different node-naming conventions, same data semantically. PLAN.md is declared "truth" (per the comment at line 1020 of PLAN.md and line 3 of `_BLOCKERS.toml`), but PLAN.md's mirror disagrees with the file it claims to mirror. Per `PLAN-QA-DISCIPLINE-R2`'s sweep-after-absorption discipline, this contradiction must be resolved.

**Recommendation:** standardize on one form. The bare-`D0` form is consistent with how every other L2 plan in this drop references its own droplets internally (e.g., W1 PLAN.md uses `D1`/`D2`/`D3`). The fully-qualified `W8.D0` form is consistent with how L1 PLAN.md references cross-wave nodes (`4c.6.1.W1`). Recommend: change `_BLOCKERS.toml` to bare `D0`/`D21` (matches PLAN.md mirror + matches sibling L2 plans), keep `4c.6.1.W1` as the only fully-qualified entry (cross-wave reference in D21's blockers).

This is intra-L2 within-wave naming; downstream tooling treats `_BLOCKERS.toml` as the machine-readable copy, so canonicalizing on the bare form keeps it parseable per-wave-directory.

### FF2 — Model IDs in frontmatter table use stale `4-5` aliases (CONFIRMED)

**Disposition:** ABSORB.

The Model/Tools table at PLAN.md lines 78-89 specifies model IDs as:

| Role | Model |
|---|---|
| planning-agent | `claude-opus-4-5` |
| builder-agent | `claude-sonnet-4-5` |
| plan-qa-proof-agent | `claude-opus-4-5` |
| ... | ... |
| commit-message-agent | `claude-haiku-4-5` |

These IDs are stale on two axes:

1. **vs `feedback_cascade_model_policy.md` memory (2026-05-09):** the policy memory specifies `claude-sonnet-4-6` (NOT 4-5) for planner+builder; `claude-opus-4-7` (NOT 4-5) for QA pair + research; `claude-haiku-4-5-20251001` (NOT bare `claude-haiku-4-5`) for commit-message.
2. **vs live system agents at `~/.claude/agents/go-*.md`:** verified via grep on 2026-05-12 — frontmatter uses bare `model: sonnet` and `model: opus` aliases (Claude Code's symbolic model-name aliases that resolve to the current generation), NOT explicit version IDs.

Both target conventions are valid (explicit-versioned IDs OR bare aliases), but `claude-opus-4-5` / `claude-sonnet-4-5` matches NEITHER. The `4-5` digits are inherited from an earlier memory snapshot that has since rolled forward.

**Recommendation:** Pick one of:
- (a) Use bare aliases (`opus` / `sonnet` / `haiku`) — matches live system-agent frontmatter; lowest maintenance, auto-tracks model generation; commit-message stays bare `haiku`.
- (b) Use explicit IDs per current memory: `claude-sonnet-4-6` / `claude-opus-4-7` / `claude-haiku-4-5-20251001` — matches `feedback_cascade_model_policy.md` verbatim.

Option (a) is preferred because (i) it matches the live system agents the prompts are LIFTED FROM (per the migration marker), (ii) bare aliases are stable across model-generation churn, and (iii) memory IDs roll forward routinely (the `4-5` IDs were CURRENT three days ago per the memory's age stamp). Option (b) is acceptable if the dev wants the explicit-version-pinned discipline encoded.

Builder instruction in the L2 plan must say which option and apply it uniformly across all 20 prompts. The table at PLAN.md lines 78-89 must be updated to match.

---

## NITs (each gets ABSORB or DEFERRED-AS-NIT with reason)

### NIT1 — D21 RiskNote typo: `~/tmp/tillsyn/main` should be `/tmp/tillsyn/main`

**Disposition:** ABSORB.

PLAN.md line 992 RiskNote for D21: "Do NOT use `~/tmp/tillsyn/main` as the RepoPrimaryWorktree value — use `t.TempDir()` for proper test isolation."

The `~/` prefix is wrong. The actual hardcoded path in `fixtureProject()` at `render_test.go:94` is `/tmp/tillsyn/main` (absolute, no home prefix). The RiskNote was clearly written from memory rather than a re-read; the lint-warning value of the bullet survives (don't reuse the fixture's worktree path; use `t.TempDir()`) but the literal string is incorrect.

**Recommendation:** Change `~/tmp/tillsyn/main` to `/tmp/tillsyn/main`. One-character fix, but verbatim-citation discipline matters for builder reproducibility.

### NIT2 — `_BLOCKERS.toml` D0 entry missing entirely

**Disposition:** ABSORB.

Inspecting the actual `_BLOCKERS.toml` file (the on-disk copy, NOT PLAN.md's mirror): lines 11-114 enumerate D1-D21 with their blockers, but there is NO `[[blockers]]` entry for D0. The header comment at lines 6-9 acknowledges D0 is the Wave A head ("D0 is the Wave A head (no blockers)").

PLAN.md's mirror block (lines 1022-1126) likewise omits D0 — there is no `[[blockers]] node = "D0"` block.

This is technically valid TOML (a head node with no upstream blockers doesn't need a `blocked_by` row), and the header comment makes the intent explicit. However, downstream tooling that walks `_BLOCKERS.toml` to construct the full droplet graph would need a separate enumeration source to discover D0 exists. The other L2 plans in this drop tend to enumerate ALL nodes in their `_BLOCKERS.toml`, including head nodes with an empty `blocked_by = []` array (e.g., W1 D1 has an explicit row with empty array).

**Recommendation:** Add explicit `[[blockers]] node = "D0" blocked_by = [] reason = "Wave A head — no upstream blockers"` rows to BOTH `_BLOCKERS.toml` (on-disk) AND PLAN.md's mirror. Improves graph-walker robustness and matches sibling L2 plans.

### NIT3 — "visibly DIFFERENT" vs precisely-specified diff in QA prompt droplets

**Disposition:** DEFERRED-AS-NIT.

D3 / D4 / D5 / D6 / D13 / D14 / D15 / D16 (the 8 QA prompt droplets) each have an AcceptanceCriterion bullet using qualitative language: "Body is visibly DIFFERENT from D5 (build-qa-proof) in its Evidence Sources and What To Check sections — NOT a near-identical copy."

The very-next paragraphs DO specify the difference concretely:
- D3 Evidence Sources: PLAN.md, REVISION_BRIEF.md, SKETCH.md (NOT Go source files).
- D5 Evidence Sources: Go source files, `git diff`, `mage test-pkg` output, PLAN.md.

The qualitative "visibly DIFFERENT" wording is harmless given the precise diff is right next door, but is methodologically softer than a verifiable check.

**Reason for deferral:** Risk of over-rigidifying the prompt-authoring task. The precise Evidence Sources list IS measurable (the builder can grep for the exact distinct section); adding a quantitative diff metric (e.g., "≥30% character-level difference") creates a new failure mode for QA agents to police that doesn't track the actual concern. The structural divergence between PLAN-evidence and CODE-evidence is what matters, and that IS specified.

### NIT4 — D8 / D18 reference WORKFLOW.md "Phase 7 — Closeout"; current WORKFLOW.md uses different phase naming

**Disposition:** DEFERRED-AS-NIT.

D8 (line 487) and D18 (line 869) reference `WORKFLOW.md §"Phase 7 — Closeout"`. The L1 plan and CLAUDE.md reference Phases 4-7 (Phase 4 builder; Phase 5 QA twins; Phase 6 push + CI; Phase 7 closeout). The reference is likely correct, but the builder citing it should confirm the section header matches verbatim before writing it into the prompt.

**Reason for deferral:** Low-risk reference-precision concern. The phase numbering is documented and stable; the builder is expected to verify section header strings via Read tool before quoting. Promoting this to ABSORB would add round-trip overhead for a check the builder will do anyway.

### NIT5 — `extends_path` relative-path correctness for runtime loader

**Disposition:** DEFERRED-AS-NIT.

D0 specifies `"extends_path": "../../../stil/main/src/bindings/baseline.json"` (PLAN.md line 140). This path resolves from the consuming loader's CWD at runtime. The path implies the loader CWD is `tillsyn/main/.tillsyn/` (three `..` steps → `tillsyn/`, → parent of `tillsyn/` and `stil/`, → into `stil/main/src/bindings/`).

But W6's FE loader (per W6 PLAN.md line 576) explicitly states it uses HTTP fetch in dev mode (`fetch('/.tillsyn/bindings.json')`) and does NOT consult `extends_path` — the baseline is loaded separately. W5's TUI loader may use `extends_path` differently.

The field is documentation-grade for v1, not load-bearing. RiskNote at PLAN.md line 168 already acknowledges "extends_path value is relative to the consuming loader's working directory at runtime."

**Reason for deferral:** Field has documentary intent and is not consumed by either v1 loader (W5 / W6 take their own paths). Hardening this would require coordinating the loader contract — out of scope for W8 prompt authoring.

### NIT6 — D-list count narration vs literal D-list at PLAN-QA-DISCIPLINE-R2 sub-clause

**Disposition:** DEFERRED-AS-NIT.

PLAN.md line 8: "This plan declares exactly 22 droplets (D0 + D1-D20 + D21). Count: 1 + 20 + 1 = 22."

PLAN.md line 1132 (verification block): "Enumerated D-list: D0, D1, D2, ..., D20, D21. Count: 22."

Both narrate "22" matching the L1 directive. The R2 sub-clause from Round 8 says: "narrative counts must match L2 spawn directive's enumerated D-list (captures the R7-FF1 failure pattern)."

There's no current drift, but the discipline rule is encoded in only ONE place (the line 8 statement). If a future round adjusts the droplet count, both locations + the verification block at line 1132 + the wave-graph at lines 22-46 + the L1 PLAN.md line 833 narration ("21st (smoke-test D21, blocked_by W1) lands at Wave C transitively") all need synchronous update.

**Reason for deferral:** Methodology refinement spilling-into-discipline territory. Promote to a fresh refinement row tracking "where the droplet count appears in this plan" rather than absorbing inline.

---

## Cross-Wave Concerns

These warrant orchestrator escalation (not L2-fixable):

### CW1 — Cascade-model-policy memory drift across all L2 plans (FF2 amplified)

FF2 is W8-specific in symptom but **the underlying problem is global**: the cascade-model-policy memory was set 2026-05-09 with explicit `claude-sonnet-4-6` / `claude-opus-4-7` / `claude-haiku-4-5-20251001` IDs. Live system agents use bare aliases. ALL L2 plans across W1-W8 are likely to have authored model IDs in some intermediate state.

**Recommendation:** orchestrator should decide globally (option a bare aliases OR option b explicit IDs) and propagate the decision across every L2 plan touching model frontmatter — NOT just W8 D1-D20.

### CW2 — D21 depends on W1's exact post-change `readProjectTierAgent` signature

Per `render.go` line 877-890 (verified 2026-05-12): the current signature is `readProjectTierAgent(projectWorktree, basename string) (string, bool, error)`. W1 D3 (per `workflow/drop_4c_6_1/DROP_4c.6.1.W1_TEMPLATE_RESOLUTION/PLAN.md` lines 192-194, 360-364) changes this to `readProjectTierAgent(projectWorktree, group, basename string) ...` and updates `assembleAgentFileBody` to pass `group`.

D21 builder must verify W1's actual signature matches the plan at build time. If W1 signature drift occurs (e.g., W1 builder adds a different param ordering or different argument), D21's test will need surgical adaptation. This is acknowledged in PLAN.md U2 (Unknowns) but should also surface to the orchestrator as a real synchronization point between W1's L2 close-out and W8 D21's L2 dispatch.

---

## Verdict Summary

- **Verdict line:** PASS-WITH-ABSORB.
- **FF count:** 2 CONFIRMED, both ABSORB.
- **NIT count:** 6 total — 2 ABSORB (NIT1, NIT2), 4 DEFERRED-AS-NIT-with-reason (NIT3, NIT4, NIT5, NIT6).
- **Cross-wave concerns escalated:** 2 (CW1, CW2).
- **Attack-vector exhaustion:** 15 prompt-specified attacks all attempted; 13 REFUTED or EXHAUSTED, 2 CONFIRMED.

The plan is internally consistent in droplet count, blocker structure, and from-scratch citation coverage. The two CONFIRMED FFs are:
1. Naming-convention drift between two co-located representations of the same blocker graph (mechanical).
2. Model-ID staleness against current policy (mechanical — pick a convention and propagate).

Neither defect blocks builder spawn; both are mechanical edits the next round can apply in under 5 minutes of orchestrator/L2-replanner work. Recommend round-N+1 absorbs FF1 + FF2 + NIT1 + NIT2 in a single surgical pass and proceeds to L2 build dispatch.
