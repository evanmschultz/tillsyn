# Drop 4c Plan-Doc Consistency Sweep — QA Bindings vs `parent_git_diff`

**Reviewer:** plan-qa-proof (consistency sweep).
**Date:** 2026-05-04.
**Scope:** verify every Drop 4c plan doc is consistent with the dev's SKETCH F.7.18 correction that QA agents (`build-qa-proof`, `build-qa-falsification`, `plan-qa-proof`, `plan-qa-falsification`) MUST NOT receive `parent_git_diff` pre-staged in their default-template `[context]` seeds.
**Mode:** read-only. No code edits, no plan edits — output is this findings MD only.

---

## Verdict

**CONFLICTS.**

One CONFIRMED conflict in `F7_18_CONTEXT_AGG_PLAN.md` F.7.18.5 acceptance criteria — the `[agent_bindings.build-qa-proof.context]` seed prescribes `parent_git_diff = true`, in direct contradiction of SKETCH.md:195 ("**NO `parent_git_diff`**"). By transitive identity (F.7.18.5 acceptance line 372: "identical to `build-qa-proof`"), `build-qa-falsification` carries the same defect.

One NIT in `SKETCH.md:207` — the bounded-mode framing prose mentions "git diff" without qualification, plausibly read as universal-to-bounded-mode rather than per-binding-conditional. Easy to misread.

Master `PLAN.md`, `F7_CORE_PLAN.md`, `F7_17_CLI_ADAPTER_PLAN.md`, the four `4c_F7_EXT_*` review/QA rounds (planner-review + QA-proof R1/R2 + QA-falsification R1/R2), and the four `PLAN_QA_FALSIFICATION_R*.md` rounds carry NO direct prescriptions of `parent_git_diff` for QA bindings. The conflict is localized to F.7.18.5's acceptance criteria; the SKETCH F.7.18 rationale section is internally consistent post-correction.

---

## 1. Conflicts (Must Fix)

### 1.1 F.7.18.5 acceptance criteria — `build-qa-proof` seed prescribes `parent_git_diff = true`

**File:** `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md`
**Lines:** 364–371 (`build-qa-proof` seed) and 372 (`build-qa-falsification` "identical to `build-qa-proof`" inheritance).

**Offending text (verbatim, line 364–371):**

```
- [ ] `[agent_bindings.build-qa-proof.context]` block:
  - `parent = true`.
  - `parent_git_diff = true`.
  - `siblings_by_kind = []` — empty placeholder; sibling-builder-worklog wiring depends on metadata plumbing not yet shipped (flagged as future refinement).
  - `ancestors_by_kind = ["plan"]`.
  - `delivery = "file"`.
  - `max_chars = 50000`.
  - `max_rule_duration = "500ms"`.
- [ ] `[agent_bindings.build-qa-falsification.context]` block: identical to `build-qa-proof` (per SKETCH:198 — same shape with falsification framing left to the agent's system prompt).
```

**Why this is a conflict:** SKETCH.md:195 explicitly says **"`build-qa-proof`: ... **NO `parent_git_diff`** — QA verifies independently by running `git diff` itself via Bash + `Read` tools. Pre-staging the diff would bias QA toward the builder's framing"**, and SKETCH.md:196 mirrors the rule for `build-qa-falsification`. F.7.18.5's acceptance criteria pin the OPPOSITE behavior into the default-template seed, which is what the builder will faithfully implement.

**Severity:** **CONFLICT — must be corrected.** A builder dispatched against F.7.18.5 today reads top-down and lands `parent_git_diff = true` in the default `default-go.toml` for both QA-on-build bindings. Round-2 falsification does NOT catch this (R1/R2/R3/R4 of `PLAN_QA_FALSIFICATION_R*.md` predate the dev's correction). REVISIONS POST-AUTHORING in F7_18_CONTEXT_AGG_PLAN.md (lines 495–512) covers REV-1 (command/args_prefix removal), REV-2 (env baseline), REV-3 (Tillsyn struct extension policy) — none address the QA-binding `parent_git_diff` correction.

**Recommended fix:**

1. Replace the `parent_git_diff = true` line in `build-qa-proof` seed (line 366) with:
   ```
   - **NO `parent_git_diff`** — QA verifies independently via Bash `git diff` + Read; pre-staging the diff biases QA toward the builder's framing. Per SKETCH.md:195.
   ```
2. Replace line 372's "identical to `build-qa-proof`" with an explicit-fields list mirroring the corrected `build-qa-proof` shape, OR keep "identical" but make sure both inherit the corrected (no-`parent_git_diff`) shape.
3. Add a new REV in F7_18_CONTEXT_AGG_PLAN.md REVISIONS POST-AUTHORING section pinning the correction:
   ```
   ### REV-4 — QA-binding seeds DO NOT pre-stage `parent_git_diff`
   F.7.18.5 acceptance criteria for `build-qa-proof`, `build-qa-falsification`,
   `plan-qa-proof`, `plan-qa-falsification` MUST NOT include
   `parent_git_diff = true`. QA agents verify independently by running
   `git diff` themselves via Bash + Read tools. Pre-staging the diff
   would bias QA toward the builder's framing. Per SKETCH.md:195–199.
   The build-qa-proof seed at line 364 + build-qa-falsification's "identical"
   inheritance at line 372 must be corrected before F.7.18.5 dispatches.
   ```
4. F.7.18.5 acceptance criterion line 391 ("`tpl.AgentBindings[domain.KindBuild].Context.Parent == true` (and equivalents for the other 5 in-scope bindings)") needs a companion negative assertion: `tpl.AgentBindings[domain.KindBuildQAProof].Context.ParentGitDiff == false` AND `tpl.AgentBindings[domain.KindBuildQAFalsification].Context.ParentGitDiff == false`.

---

## 2. NITs (Cosmetic / Risk-of-Misreading)

### 2.1 SKETCH.md "Bounded mode" framing prose — generic "git diff" phrasing borderline-misleading

**File:** `workflow/drop_4c/SKETCH.md`
**Line:** 207.

**Offending text (verbatim):**

```
- **Bounded mode** (declare `[context]`): agent receives pre-staged parent / siblings / ancestors / git diff at spawn, calls MCP only on completion. Predictable cost, lower latency, less round-tripping.
```

**Why this is a NIT (not a conflict):** the line is FRAMING the bounded-mode runtime semantic — describing the kinds of things bounded mode CAN provide, not prescribing every bounded-mode binding receives every kind. Read alongside SKETCH.md:194–199 (per-binding seed table that explicitly carves QA bindings OUT of `parent_git_diff`), the line is consistent. But a reader landing on line 207 first, without reading lines 194–199, could infer "every bounded-mode agent gets git diff" — which is exactly the defect that landed in F.7.18.5. The framing line was likely the source-text the F.7.18.5 planner read when seeding the QA bindings.

**Severity:** **NIT.** Internally consistent if read top-to-bottom. Misleading if read out-of-order.

**Recommended fix (optional):** add a parenthetical qualifier:

```
- **Bounded mode** (declare `[context]`): agent receives pre-staged parent / siblings / ancestors / git diff at spawn (per the seeds — git diff staged ONLY for builder bindings, NOT QA bindings; see lines 194–199), calls MCP only on completion. Predictable cost, lower latency, less round-tripping.
```

OR rephrase to "agent receives pre-staged context per the binding's `[context]` declaration" — drop the kind-list to avoid implying universality.

---

## 3. Files Verified Clean

The following plan docs were swept for `parent_git_diff` references in QA-binding default-template seeds. Each carries either NO QA-binding seed prescription, OR carries the correct (no-`parent_git_diff`) shape per SKETCH:195–197. None require correction.

### 3.1 `workflow/drop_4c/PLAN.md` (master) — clean

Master PLAN does not enumerate per-binding default-template seeds. §3 L13 declares the F.7.18 context aggregator OPTIONAL; §5 canonical-ID table cites F.7.18.5 by name as "Default-template seeds" without pinning specific seeds. NO `parent_git_diff` prescription anywhere. Pre-Drop-1 W3.2 vs 4a.25 NIT-1 documented in PLAN_QA_PROOF.md is unrelated.

### 3.2 `workflow/drop_4c/F7_CORE_PLAN.md` — clean

F.7-CORE plan covers F.7.1–F.7.16 (spawn pipeline, gates, commit/push). NO default-template seed prescription for QA bindings. F.7.16 acceptance covers `[gates.build]` expansion (line 944), not `[agent_bindings.<kind>.context]` seeds — those are F.7.18.5 territory. F.7-CORE F.7.4 (stream parser) line 351 mentions `permission_denials[]` but not `parent_git_diff`. Clean.

### 3.3 `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` — clean

F.7.17 plan covers CLI adapter seam (Schema-1, adapter scaffold, claudeAdapter, MockAdapter, dispatcher wiring, manifest cli_kind, permission_grants cli_kind, BindingResolved, monitor refactor, marketplace paper-spec, adapter docs). NO `[agent_bindings.<kind>.context]` content — that's owned entirely by F.7.18. Clean.

### 3.4 `workflow/drop_4c/4c_F7_EXT_PLANNER_REVIEW.md` — clean

Round-1 planner review predates the dev's correction. Mentions `parent_git_diff` ONCE at line 220 inside the F.7.18 schema-field enumeration (`parent`, `parent_git_diff`, `siblings_by_kind`, `ancestors_by_kind`, `descendants_by_kind`, `delivery`, `max_chars`, `include_round_history`). This is an SCHEMA-FIELD-NAME enumeration, NOT a per-binding default-template seed prescription. Acceptable.

### 3.5 `workflow/drop_4c/4c_F7_EXT_QA_PROOF.md` — clean

Round-1 QA-proof on F.7.17/F.7.18 architecture. Mentions `parent_git_diff` once at line 63 in the schema-field enumeration. Mentions `seed-default for build-qa-proof (line 175)` at line 64 referring to the THEN-current SKETCH at review time — historical citation, not a forward-prescription. Verdict ("PROOF GREEN-WITH-NITS") + recommended-revisions list at lines 144–148 do NOT prescribe `parent_git_diff` for QA. Clean.

### 3.6 `workflow/drop_4c/4c_F7_EXT_QA_PROOF_R2.md` — clean

Round-2 QA-proof reviews the SKETCH rework. Line 102 paraphrases the bounded-mode framing using the same generic "parent / siblings / ancestors / git diff" shape as SKETCH.md:207 (carries the same NIT 2.1 above) but does NOT prescribe `parent_git_diff` for QA bindings. NO per-binding seed prescription. Acceptable.

### 3.7 `workflow/drop_4c/4c_F7_EXT_QA_FALSIFICATION.md` — clean

Round-1 falsification, V1–V15. Mentions `parent_git_diff` at line 154 inside the schema-field enumeration (V6 token-budget priority discussion). Mentions `parent_git_diff` at line 261 / 291 in V10 ancestor-walk-cost discussion — both in SCHEMA-FIELD context, not seed prescription. NO per-binding seed prescription. Clean.

### 3.8 `workflow/drop_4c/4c_F7_EXT_QA_FALSIFICATION_R2.md` — clean

Round-2 falsification. Discusses V12/A5 (round-history deferred), A6 (greedy-fit), A7 (two-axis timeouts). NO `parent_git_diff` prescription for QA bindings. Clean.

### 3.9 `workflow/drop_4c/PLAN_QA_PROOF.md` — clean

Plan-QA-proof on master + sub-plans. Acceptance-testability sweep at line 181 cites F.7.18.5's `tpl.AgentBindings[domain.KindBuild].Context.Parent == true` without enumerating the specific QA-binding seed values. NO independent QA-binding `parent_git_diff` prescription. Clean.

### 3.10 `workflow/drop_4c/PLAN_QA_FALSIFICATION.md` (R1) — clean

Round-1 plan-falsification. Attack 14 line 449 explicitly cites F.7.18.5's `[agent_bindings.build.context]` seed `parent_git_diff = true` — but for the BUILD binding only (which IS allowed `parent_git_diff = true` per SKETCH:194). Does NOT cite the QA-binding seeds. Clean.

### 3.11 `workflow/drop_4c/PLAN_QA_FALSIFICATION_R2.md` — clean

R2 plan-falsification. NO mentions of `parent_git_diff` in QA-binding seed context. Clean.

### 3.12 `workflow/drop_4c/PLAN_QA_FALSIFICATION_R3.md` — clean

R3 plan-falsification. NO mentions of `parent_git_diff` in QA-binding seed context. Clean.

### 3.13 `workflow/drop_4c/PLAN_QA_FALSIFICATION_R4.md` — clean

R4 plan-falsification (final verification). NO mentions of `parent_git_diff` in QA-binding seed context. Clean.

### 3.14 SKETCH.md F.7.18 rationale section internally consistent

Cross-checking the dev's correction internally:

- **SKETCH.md:193 ("Spawn-prompt always-delivered baseline")** correctly describes action-item shape (`id, kind, parent_id, paths, packages, completion_contract, metadata.outcome / blocked_reason if present`), `session_id` for `till.*` MCP calls, working-directory + bundle paths. Does NOT include git diff in the always-delivered baseline. Consistent with the dev's correction.
- **SKETCH.md:194 (`build`)** ships `parent_git_diff` in the seed — correct (builder's lens).
- **SKETCH.md:195 (`build-qa-proof`)** explicitly says "**NO `parent_git_diff`**" with rationale.
- **SKETCH.md:196 (`build-qa-falsification`)** mirrors the rule.
- **SKETCH.md:197 (`plan-qa-proof` / `plan-qa-falsification`)** "No git diff needed (plan QA reviews planning artifacts, not code)."
- **SKETCH.md:198 (`plan` planner)** parent + ancestors only.
- **SKETCH.md:199 ("Why builder gets diff but QA doesn't")** rationale matches the per-binding seeds enumerated above it.

Internal rationale section is clean. The dev's correction landed coherently.

---

## 4. Hylla Feedback

`N/A — review touched non-Go files only` (12 plan-doc MDs across `workflow/drop_4c/`). No Hylla queries issued; cross-checks performed via direct `Read`. No miss to record.

---

## TL;DR

- **T1: Verdict CONFLICTS.** One CONFIRMED conflict in F7_18_CONTEXT_AGG_PLAN.md F.7.18.5 acceptance criteria (lines 364–372): the `build-qa-proof` and `build-qa-falsification` default-template seeds prescribe `parent_git_diff = true`, contradicting SKETCH.md:195–196's "**NO `parent_git_diff`**" rule. Recommended fix: drop the `parent_git_diff = true` line, add explicit "NO" markers, add REV-4 to F7_18 REVISIONS section, add negative-assertion test in F.7.18.5 acceptance.
- **T2: One NIT** at SKETCH.md:207 — bounded-mode framing prose mentions "git diff" without qualification, plausibly read as universal-to-bounded-mode. Probable source of the F.7.18.5 misseed. Easy fix: parenthetical qualifier or rephrase to "agent receives pre-staged context per the binding's `[context]` declaration."
- **T3: 13 plan-doc files swept clean** — master PLAN.md, F7_CORE_PLAN.md, F7_17_CLI_ADAPTER_PLAN.md, four 4c_F7_EXT_* review/QA rounds, four PLAN_QA_FALSIFICATION_R* rounds, PLAN_QA_PROOF.md, and SKETCH.md F.7.18 rationale section all carry NO conflicting QA-binding `parent_git_diff` prescriptions. Pre-existing schema-field enumerations (e.g. planner review line 220, QA proof R1 line 63) reference `parent_git_diff` as a SCHEMA FIELD NAME — not as a seed prescription — and are acceptable.
