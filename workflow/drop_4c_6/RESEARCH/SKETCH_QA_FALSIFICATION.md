# SKETCH v2.8.3 — Plan QA Falsification Review

**Subject under attack**: `workflow/drop_4c_6/SKETCH.md` (v2.8.3 FINAL) + `workflow/drop_4c_6/PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md`.

**Scope**: First cascade-flavored QA pass on the methodology drop. Plan-QA-falsification axis (parent kind = `plan`, level-0 plan). Active counterexample construction, REFUTED / CONFIRMED labels.

**Verdict bar**: a confirmed counterexample is a concrete reproduction with cited evidence (file, line, memory entry, or sketch reference) — not speculation.

---

## 1. Findings

### 1.1 Wave-decomposition under-/over-sized — REFUTED with two notable hedges

**Attack**: are any waves obviously too big (warrant splitting) or too small (busywork)?

I walked all eleven Specify blocks in §26 (W0, W0.5, W1, W2, W3, W5, W6, W7, W8, W9, W10, W11, W4-A, W4-B, W4-C, W4-D). Most waves carry 4-9 AcceptanceCriteria spanning a single coherent surface (config schema, embed FS, init command, render layer, etc.). The atomic-droplet sizing in `feedback_plan_down_build_up.md` lives at BUILD level, not WAVE level — so the under/over-size attack here is "does a wave decompose cleanly into sub-plan + atomic builds, or is it a grab-bag?"

Two waves bear examination:

- **W3** carries 7 AcceptanceCriteria spanning (a) `BindingResolved.SystemPromptTemplatePath` plumbing, (b) 3-tier priority resolution, (c) frontmatter `model:`/`tools:` strip, (d) defense-in-depth env vars, (e) post-render validator, (f) doc-comment corrections, (g) sentinel-injection integration test. That's five distinct production surfaces (`internal/app/dispatcher/cli_adapter.go` for the field, `render/render.go` for resolution + body assembly + validator, `cli_claude/env.go` for env vars, `SPAWN_PIPELINE.md` for docs, integration test). REFUTED as "too big" because each is a sub-plan child the wave-level planner spawns; the wave is a containing segment, not a build droplet. But it's the densest wave — flag for the planner-agent to watch.
- **W11** carries 9 AcceptanceCriteria spanning MCP-tool removal, role-gated rejects on MCP boundary, file-write-outside-paths reject, comment role-gating, `Service.RestoreActionItem` verify, and template-defined warning schema. Again, that's a multi-package wave (mcpapi + service + policy + comments). REFUTED as "too big" for the same reason — a containing segment whose planner will decompose. But §26.W11's ContextBlocks list it at "critical" severity, which is appropriate.

No wave is obviously "1 AcceptanceCriterion that's trivial busywork." W6 (docs) is the smallest at 4 AcceptanceCriteria; not busywork — `CASCADE_METHODOLOGY.md` skeleton is load-bearing for `project_methodology_docs_tracker.md`'s MVP gate.

**Verdict**: REFUTED. Hedge — flag W3 + W11 as the densest waves; planner should start their decomposition early so plan-QA can run on each wave's child plan independently.

### 1.2 Missing dependencies — REFUTED with one explicit caveat called out by sketch

**Attack**: does any wave silently depend on another?

Sketch §25 explicitly states: "4c.6 → 4c.7 (sequential; W3 frontmatter strip + bundle full content needed by W8 context preload). 4c.7 → 4c.8 (sequential; W7+W8 must work for prompt drafts to assume auto-create + context preload)."

Within 4c.6 the dependencies are: W0 → W0.5 (validators consume schema types); W0 → W3 (frontmatter strip needs `agents.toml` resolution); W1 → W2 (`till init` copies from embed FS); W1 → W3 (render reads embed-default fallback). Within 4c.7: W7 → W10 (wipe-and-replan triggers fresh-spawn-with-auto-create); W8 → W10 (failure-context section is rendered via context-preload pipeline). Sketch §26.W10 explicitly cross-references W3 + W8 as "render-layer hooks depended on."

Where I attempted a hidden-dep counterexample:

- **W11 ↔ W7**: W11 introduces a "non-system actor attempting `till.action_item.update` on system-auto-created QA action item → reject" rule. The check needs to know "this QA was system-auto-created." That data only exists if W7's auto-create pipeline writes a marker (e.g. `metadata.system_managed: true`) on auto-created children. Sketch §26.W7 says nothing about writing a marker. **CONFIRMED minor**: W11 implicitly depends on W7 marking system-auto-created children with a discriminator; sketch doesn't say so. See §2.1 below — this is a genuine counterexample to "no missing dependencies." Promoted from §1 to §2.

- **W4-A "language-generic"**: depends on W4-B not existing yet for verification — except actually W4-A and W4-B are siblings, so this is fine.

**Verdict**: REFUTED at sketch level (sequencing called out); CONFIRMED at the W11 ↔ W7 fine-grain (§2.1). Promoted.

### 1.3 Section-numbering bug in SKETCH — CONFIRMED minor

**Attack**: sketch order should make sense at L1 read.

SKETCH §22 (line 642) is followed by §25 (line 662) with no §23/§24 in between. §23 + §24 then appear at lines 1133 + 1145 — AFTER §25 + §26 + an "OLD §25" residue (line 1083). Reader hitting the doc cold will be confused.

This is a documentation defect, not a methodology defect. CONFIRMED, but trivial to fix: re-number §25 → §23, §26 → §24, drop the "OLD §25" section (or move it to an appendix), and renumber §23 + §24 in their current positions appropriately.

**Verdict**: CONFIRMED minor. Listed in §1, not §2 — this is a sketch hygiene gripe, not a methodology counterexample.

### 1.4 Specify under-constraint — REFUTED across all 14 waves

**Attack**: Objective vague enough that multiple valid implementations satisfy it.

I read each Specify Objective in §26 against its AcceptanceCriteria. Every Objective is paired with concrete file paths, struct names, mage targets, or behavior statements that pin the implementation:

- W0 names `internal/config/agents.go` + `AgentRuntime` + `AgentsRegistry` types.
- W0.5 names six specific validators with output contract (TOML-line pointer + structured error message).
- W2 names `cmd/till/`, vendor sources (`fsatomic`, `configmerge`), and ten distinct UX behaviors.
- W3 names `BindingResolved.SystemPromptTemplatePath`, `render.go:assembleAgentFileBody`, four env vars by name, two file paths for doc-comment corrections.
- W7 names `Service.CreateActionItem` + `tpl.ChildRulesFor` + atomic transaction.
- W8 names `render.go:assembleSystemPromptBody` + `context.Resolve` + new `all_peer_children` rule.
- W10 names `Service.WipeChildrenAndRePlan(parent_id)` + `metadata.failure_history` field + N=3 default + escalation path.
- W11 names six hardcoded structural rejects + advisory-warning schema shape.

Verdict: REFUTED across all 14 waves. Every Objective is constrained enough that planner decomposition is well-bounded.

### 1.5 Specify over-constraint — REFUTED

**Attack**: AcceptanceCriteria rules out reasonable implementations.

Spot-checked W0, W3, W7, W11 (the most-detailed waves). No AcceptanceCriterion mandates a specific algorithm where multiple are equivalent (e.g., none says "use depth-first traversal" — W0.5's cycle detector is left to "graph walk detects A→B→A cycles"). The schema-shape AcceptanceCriteria do constrain to specific TOML field names (`system_prompt_template_path`, `tools_allow`, etc.) — appropriate, since adopters' templates depend on names being stable.

**Verdict**: REFUTED.

### 1.6 Untestable AcceptanceCriteria — REFUTED with two soft spots

**Attack**: bullets without code-inspection or `mage <target>` evidence path.

I scanned every Specify block. Every AcceptanceCriterion has an associated `mage <target>` invocation in its ValidationPlan, OR is verifiable by code inspection (struct exists, field exists, file copied). The closest thing to an untestable criterion:

- **W4-A / W4-B / W4-C**: "Prompts pass plan-QA review (separate pass before merge)." That's a meta-criterion — passing depends on a future plan-QA run, not a `mage` target. But it's testable via the actual run, so this is procedural-acceptance, not pseudo-acceptance. Mild — flag for clarity but not a counterexample.
- **W6**: "`AGENTS_CONFIG.md` written: schema, override semantics, env_set/env_from_shell, Bedrock/Vertex/OpenRouter/Ollama Cloud examples..." — what does "written" mean concretely? The criterion is shape-of-doc, not behavior. Verifiable by reading the doc. Not a counterexample but a soft spot.

**Verdict**: REFUTED with two soft spots flagged.

### 1.7 Plan-down/build-up violation — REFUTED at wave level; one nit at planner-agent draft

**Attack**: does any wave assume code that needs higher-level integration NOT designed in the wave above? Or is integration deferred to lower levels (build-up violation)?

Plan-down requires interface/API/integration to be designed at higher levels BEFORE atoms are built. Walking the dep graph:

- W0 designs the schema (data types) → W0.5 validates schema (consumer) → W3 reads schema at render (consumer #2) → all three coherent. ✓
- W7 designs `Service.CreateActionItem` integration of `ChildRulesFor` → W10's wipe-and-replan re-uses the same integration ✓.
- W8 designs context-preload pipeline → W10's failure-history rendering re-uses the pipeline ✓.

No "build-up violation" found at the wave level. **Nit at the planner-agent draft**: the draft says "you author ONE level of decomposition" + "you do NOT plan to atoms in a single spawn." That's correct per `feedback_plan_down_build_up.md`. But the draft's Cascade Design HARD RULES section uses absolute numbers — "1-4 code blocks, 80-120 LOC + tests, ideally one production file" — without naming the source-of-truth as the template's atomic-droplet sizing. Per `feedback_tillsyn_enforces_templates.md`: "the till-go default template encodes one specific set of rules." The numbers should be cited as till-go template values, not Tillsyn invariants. See §2.2.

**Verdict**: REFUTED at wave level; CONFIRMED minor on planner-agent draft framing (promoted to §2.2).

### 1.8 Atomic-droplet enforcement — REFUTED at sketch level

**Attack**: does the sketch impose caps on droplets-per-wave that violate `feedback_plan_down_build_up.md`'s "no cap on children per pass" rule?

Sketch §25 explicitly states: "Droplet counts are NOT specified at this sketch level per `feedback_plan_down_build_up.md` — the planner-agent decomposes each wave into however many droplets fit the work." §25.1 + §25.2 + §25.3 each restate "droplet count determined by planner during plan-down (no cap)." §25.2 even says: "Per dev's 'don't let stuff fall through cracks' guidance, the planner is biased toward smaller per-droplet sizes — but that's a quality guideline, not a count cap."

Per-wave Specify blocks (§26) carry no per-wave droplet count anywhere.

**Verdict**: REFUTED. The sketch correctly defers to the planner.

### 1.9 Tillsyn-enforces-templates violation — REFUTED at wave level; one promoted CONFIRMED at planner-agent draft

**Attack** (per `feedback_tillsyn_enforces_templates.md`): does any wave specify hardcoded behavior that should be template-respected? Does any wave specify template-respected behavior that should be hardcoded structural?

§26.W11 explicitly tracks the structural-vs-semantic split: hardcoded structural rejects (separation-of-concerns invariants) vs template-defined advisory warnings. ContextBlocks at "critical" severity tag separation-of-concerns as architecture invariant. W11 dropped the `allow` policy per dev §3.1 — only `reject` (hardcoded structural) and `warn` (template advisory) remain. That's the cleanest version of the principle.

§26.W0.5 calls out validators + load-time fail-loud — Tillsyn enforces; templates define. ✓

§26.W11 schema for `[[advisory_warnings]]` is appropriately template-defined.

**One CONFIRMED promotion**: see §2.2 — planner-agent draft hardcodes till-go template numbers as if they were Tillsyn invariants.

**Verdict**: REFUTED at wave level; CONFIRMED in planner-agent draft framing (§2.2).

### 1.10 Default templates treated as Tillsyn behavior — REFUTED at wave level

**Attack**: does the sketch over-couple to default-go specifics?

Sketch §3.5 explicitly distinguishes `till-gen` (language-generic) vs `till-go` (Go+mage) vs `till-gdd` (post-Hylla, placeholder). §16 distinguishes "till-go defaults" from "Tillsyn dogfood overrides at `<this-repo>/.tillsyn/agents/`" — the latter adds Hylla-first content that defaults explicitly do NOT carry. §11 and §15 explicitly note "till-go" specifics drop out for `till-gen`. W4-A AcceptanceCriteria require "NO Hylla / NO mage / NO Go specifics in till-gen."

This rule is well-respected at sketch level.

**Verdict**: REFUTED.

### 1.11 Separation-of-concerns role bleed — REFUTED

**Attack**: are there places where one role does another's work?

Per `feedback_prompt_injection_team.md`: "planners NEVER touch QA action items; QA reads but doesn't author planner content; builders read specs but don't decompose." The planner-agent draft's "What you do NOT do" list is the strictest version of this in the project — explicit "never create, edit, or archive any QA action item" + "never author commit messages" + "never edit code." §11.2 reinforces system-managed wipe (planner doesn't archive QA-twins). §26.W11 hardcodes structural rejects at the MCP boundary as defense-in-depth.

I tried to construct a counterexample where a wave assumes the planner must touch QA (e.g., wipe-and-replan): §26.W10 explicitly puts the wipe ON the SYSTEM (`Service.WipeChildrenAndRePlan`); planner is BLIND to archived children. Confirmed clean.

I tried where a wave assumes the builder must decompose: no counterexample found.

**Verdict**: REFUTED.

### 1.12 Eat-own-dogfood — REFUTED with explicit hedge

**Attack**: does §26 actually demonstrate the methodology, or are some Specify blocks thin/ceremonial?

§26 is the dogfood. Each wave's Specify carries Objective + AcceptanceCriteria + ValidationPlan + RiskNotes + ContextBlocks — the same shape `planning-agent.md` says action-item metadata will hold. Spot-check:

- **W0**: AcceptanceCriteria pins file path, struct names, mage target. ContextBlocks at `decision normal` ("`tools_deny` is NOT user-overridable") and `constraint high` ("override-merge semantics per §5 are load-bearing"). Substantive.
- **W11**: ContextBlocks distinguish `constraint critical` (hardcoded structural) from `constraint high` (advisory only never blocks) from `decision normal` (allow policy dropped). Real-world scope is encoded in the ContextBlocks shape. Substantive.
- **W9**: explicitly says "empty wave acceptable: if W7+W8 land cleanly, drop this wave; don't manufacture work." Honest engineering — not ceremony.

The Specify blocks aren't all the same depth, which is appropriate per "Spec scales with droplet size" in §10.3.

**Verdict**: REFUTED. Sketch eats its own dogfood credibly.

### 1.13 Memory-rule conflicts — REFUTED with one explicit cross-ref miss

**Attack**: cross-check sketch claims against memory entries.

- `feedback_plan_down_build_up.md`: "no cap on children per pass." ✓ Sketch respects.
- `feedback_tillsyn_enforces_templates.md`: "schema-shipped without consumer is anti-pattern." ✓ Sketch §22 + §25 audit + W7/W8 wiring waves.
- `feedback_prompt_injection_team.md`: "separation-of-concerns is structural defense; sanitization is belt-and-suspenders." ✓ §26.W11 hardcodes structural rejects; §26.W11 RiskNotes mentions team-aware extension.
- `feedback_cascade_model_policy.md`: planners + builders sonnet, QA opus, commit haiku. ✓ §26 W0 → W11 carries this in §4.2.
- `feedback_no_closeout_md_pre_dogfood.md`: "skip CLOSEOUT/LEDGER/HYLLA_FEEDBACK rollups while not dogfooding." Sketch does NOT mention closeout artifacts — neutral. ✓
- `feedback_orphan_via_collapse_defer_refinement.md`: when a collapse orphans downstream vocabularies, defer to refinement. Not directly relevant.
- `feedback_subagents_short_contexts.md`: "subagents lose spawn-prompt Tillsyn creds on mid-run compaction." Sketch §22 / §11.2 does NOT discuss compaction-resilience for the wipe-and-replan flow. **Soft spot but not a counterexample** — flag as Unknown for §3.

One explicit cross-ref miss: `feedback_section_0_required.md` requires Section 0 on substantive responses. The planner-agent draft mandates Section 0 in stdout. The QA agent drafts (qa-proof-agent.md / qa-falsification-agent.md) are listed as "drafts pending planner sign-off" (§17.2) — when those are written, they MUST mandate Section 0 too. Not a sketch defect; just a forward-looking risk.

**Verdict**: REFUTED. No unmitigated memory-rule conflicts.

### 1.14 Schema-without-consumer attack on the sketch's own waves — REFUTED

**Attack**: does W0.5 define template validators but the validation isn't wired into the actual template-loading code path? Does W7 wire `ChildRulesFor` but the integration test only unit-tests the consumer without the loader-to-spawn end-to-end coverage?

- **W0.5**: AcceptanceCriteria explicitly says "integration test confirms loader rejects each fixture with correct error shape." ValidationPlan: `mage test-pkg ./internal/templates`; integration test confirms loader rejects each fixture. So the validator IS wired into the loader code path. ✓
- **W7**: AcceptanceCriteria explicitly says "Integration test: creating `kind=plan` auto-fires `plan-qa-proof` + `plan-qa-falsification` with `blocked_by` pointing at the plan." That's end-to-end (template → resolver → consumer → observable behavior change). ✓
- **W8**: "Integration test: spawn each cascade kind; assert system-prompt content matches §11.1 declared bundle." End-to-end. ✓
- **W10**: "Integration test: QA fails plan → wipe fires → fresh planner spawn sees synthesized failure_context in system prompt → planner authors fresh decomposition without reading archived children → fresh QA-twins auto-fire on new children." End-to-end. ✓
- **W11**: "Integration tests for each rejection + each warning path." End-to-end. ✓

Every wiring wave has an end-to-end integration test in its AcceptanceCriteria. The sketch is self-aware about the schema-without-consumer anti-pattern (§22 audit).

**Verdict**: REFUTED.

### 1.15 Resolver-without-test (W0.5 cycle detector) — REFUTED

**Attack**: where does `mage ci` actually exercise these new wirings?

W0.5 ValidationPlan: "malformed-template-fixture test PER validator (one fixture per error case); `mage test-pkg ./internal/templates`; integration test confirms loader rejects each fixture with correct error shape." That's per-validator test fixture coverage — six validators, six fixtures minimum.

**Verdict**: REFUTED.

### 1.16 W11 hardcoded vs template-tunable boundary — REFUTED with one nit

**Attack**: in W11, are any of the listed structural rejects actually adopter-tunable that are being hardcoded? Conversely: in W11's template-defined warnings, are any actually structural that should be hardcoded?

W11 hardcoded structural rejects (per §26.W11):
1. Planner creating QA action items → reject. **Structural per `feedback_prompt_injection_team.md`** ✓.
2. Builder creating any action items → reject. **Structural** (builders implement; they don't decompose; this is the cascade-architecture invariant per `feedback_plan_down_build_up.md`) ✓.
3. Non-system actor updating system-auto-created QA → reject. **Structural** (audit-trail integrity) ✓.
4. File-write outside declared `paths` → reject. **Structural** (lock-graph correctness; see Drop 4a Wave 2 lock manager). ✓
5. Comment role-gating: QA only on own action item → reject. **Structural per `feedback_prompt_injection_team.md`** (separation-of-concerns) ✓.
6. `till.action_item.delete` from any agent → reject. **Structural** (audit-trail integrity) ✓.

W11 template-defined advisory warnings (e.g., builder runs `mage test-pkg` instead of `mage test-func`):
- This IS adopter-tunable. Some adopters' agents may legitimately run `mage test-pkg` after each function-level test. ✓

I tried to find a structural reject that's actually adopter-tunable: did not.

**One nit**: §26.W11 RiskNotes says "Hardcoded rejects must NOT include things adopters legitimately want to tune." That's a self-aware guardrail — but the criterion is not enumerable a priori; the planner has to apply judgment. Reasonable.

**Verdict**: REFUTED.

### 1.17 N=3 escalation default — REFUTED with hedge

**Attack** (W10): is N=3 too aggressive? Too lax?

Sketch §26.W10: "N=3 means: 1st planning attempt + 2 retries → escalate. Bounded token cost; humans don't get pinged for trivial failures." Configurable per template post-MVP.

A counterexample to "N=3 is right": a flaky LSP / Hylla / network condition causing transient QA failures could exhaust the budget on infrastructure flake. But the sketch makes N tunable post-MVP; MVP can ship with N=3 and adopters can tune.

**Verdict**: REFUTED with hedge — flag for measurement during dogfood.

### 1.18 Token cost claim for failure synthesis — REFUTED with explicit unknown

**Attack** (§11.2): "failure synthesis is ~200-500 tokens vs ~2000-5000 archived children dump."

I cannot verify these numbers without measuring on real data. The CLAIM is plausible — failure synthesis is "what was tried + why it failed + don't repeat" prose; archived children dump is full descriptions + full metadata + full KindPayload + full ContextBlocks. Order-of-magnitude difference is plausible; ground truth is dogfood-measurable.

**Verdict**: REFUTED at the directional claim ("synthesis < dump"). UNKNOWN at exact numbers — flag for dogfood measurement (§3).

### 1.19 Cross-orch / multi-actor scenarios — REFUTED with explicit deferral

**Attack**: does the sketch handle multi-orch scenarios (e.g., one orch's wipe-and-replan firing while another orch is reading the parent plan)?

Sketch defers to Drop 4a's manual-trigger dispatcher (Wave 2 file/package locks). Multi-orch coordination is out of scope. §26.W10's atomic-archive transaction handles in-process atomicity; cross-orch is not addressed.

**Verdict**: REFUTED — explicitly deferred.

### 1.20 Recursion-depth bound default = 5 (W0.5) — REFUTED

**Attack**: depth 5 too low? Too high?

Sketch §26.W0.5: "child rules cannot trigger more than N levels deep (default 5; configurable post-MVP via template)." Cascade tree shape per CLAUDE.md is `project → plan → (plan-qa-twins + builds + sub-plans + research) → (build-qa-twins + sub-sub-plans + ...)`. Real cascade depth in current Drop 4c.5 / 4c.6 is well under 5. 5 covers room to grow without runaway recursion. Configurable post-MVP.

**Verdict**: REFUTED.

---

## 2. Counterexamples

### 2.1 W11 ↔ W7 implicit dependency on a system-managed-marker — CONFIRMED minor

**The claim under attack** (§26.W11 AcceptanceCriterion): "Any non-system actor attempting `till.action_item.update` on system-auto-created QA action item → reject."

**Why it's a counterexample**: this rule requires the runtime to distinguish "system-auto-created" from "agent-created" QA action items. Today's `domain.ActionItem` does not carry such a discriminator (`internal/domain/action_item.go` has no `system_managed` / `auto_created` field; auto-create's not landed yet). W7 (`Service.CreateActionItem` → `ChildRulesFor` consumer) is where the marker would be written — but §26.W7 AcceptanceCriteria do not mention writing such a marker. §26.W7 RiskNotes mention "Children's `paths`/`packages`: inherit from parent or stay empty? Template specifies." — but no `metadata.system_managed: true` (or equivalent) flag.

**Reproduction**: W11 runtime-reject logic must inspect each QA action item to decide "system created or not." If W7 doesn't write the discriminator, W11's rule has no source of truth. Either:

- **Option A**: W7 must write a `metadata.system_managed: true` (or `metadata.auto_created_via_child_rule: true`) flag on every auto-created child, AND W11's reject must read this field. Requires adding the AcceptanceCriterion to W7's Specify block.
- **Option B**: W11 infers "auto-created" from "kind ∈ {plan-qa-*, build-qa-*}" — but this is too coarse, since the sketch contemplates future template-defined kinds that also need protection.
- **Option C**: W11's actor-role check is enough — if any non-system actor tries to update ANY QA-kind action item, reject. Then no marker is needed. But this collides with the sketch's own §26.W11 ("Any non-system actor attempting `till.action_item.update` on system-auto-created QA action item → reject") which qualifies the reject to "system-auto-created" specifically.

**Suggested fix**: extend W7 AcceptanceCriteria with: "Auto-created children carry `metadata.auto_created_via_child_rule: true` (typed bool) for downstream W11 enforcement." Add equivalent ContextBlocks `decision normal` ("auto-created marker is the discriminator W11 reads") and `reference normal` (cross-ref to W11). Adjust W11 AcceptanceCriteria to read this field.

**Severity**: MINOR. Easy to fix in the planner-spawn for 4c.7's W7 + W11 (or as a cross-wave update before plan-down). But unfixed it's a real implementation gap.

### 2.2 Planner-agent draft hardcodes till-go template numbers as universal cascade rules — CONFIRMED minor

**The claim under attack** (`PROPOSED_AGENT_DRAFTS/till-go/planning-agent.md` § "Atomic droplet sizing — hard limits"):

> A `kind=build` action item with `Irreducible: true` is an ATOMIC DROPLET. Hard sizing constraints:
> - **1-4 code blocks** of change.
> - **80-120 LOC of production code MAX**, plus its tests.
> - **Ideally one production file**.

**Why it's a counterexample to `feedback_tillsyn_enforces_templates.md`**: those numbers are till-go template values, NOT Tillsyn cascade-architecture invariants. Per memory: "the till-go default template encodes one specific set of rules (1-4 code blocks, 80-120 LOC, mage test-func per-function red-green, etc.). Adopters who fork the template OR write their own can set different rules." The planner-agent draft is the till-go variant, so the numbers are correct FOR till-go — but the framing "**non-negotiable**" + "**HARD RULES**" suggests these are invariants. They are not. They're template-defined caps.

**Reproduction**: an adopter forking till-go to till-rust may legitimately want different sizing (Rust code blocks tend to be larger; 80-120 LOC may be too tight). If the till-rust planner-agent draft inherits the till-go framing wholesale ("non-negotiable"), the adopter is misled.

**Suggested fix**: re-frame the sizing constraints as "till-go template values" — e.g., introduce them with: "The till-go template's atomic-droplet sizing caps are:" (then list the numbers), followed by "These caps come from the template's KindRule for `kind=build`. Tillsyn enforces whatever sizing the loaded template defines; your job is to respect this template's specific caps." This preserves the strong sizing discipline FOR till-go users while leaving the adopter latitude for `till-gen` / `till-gdd` / forks.

A second framing fix: replace "non-negotiable" + "HARD RULES" with "till-go template caps (your template; respect it)". Same enforcement; correct attribution.

**Severity**: MINOR. The planner-agent draft is in `till-go/`, so the numbers are right for users in scope. But the framing could mislead adopters reading the draft to inform their own template's planner-agent.

**Cross-reference**: `feedback_tillsyn_enforces_templates.md`: "When writing default agent prompts: don't hardcode rules that should be template-customizable. Assume Tillsyn enforces template rules; the prompt directs agent behavior WITHIN those rules."

### 2.3 SKETCH numbering ordering — CONFIRMED trivial

Already covered in §1.3. Not a methodology counterexample; a documentation hygiene defect. Listed here for completeness because it's a CONFIRMED finding, just trivial in severity.

**Suggested fix**: re-number §25 → §23, §26 → §24; remove or appendix the "OLD §25" residue at line 1083; renumber the §23 / §24 currently at lines 1133 / 1145.

---

## 3. Summary

**Verdict: PASS** with three CONFIRMED minor counterexamples flagged. None of them invalidate the overall sketch; all three are addressable as small Specify-block updates or doc-hygiene fixes during plan-down.

**CONFIRMED counterexamples (3, all MINOR):**
- §2.1 — W11 needs W7 to write a `system_managed` discriminator that W7's AcceptanceCriteria don't mention. Cross-wave dep gap.
- §2.2 — Planner-agent draft frames till-go template numbers as "non-negotiable HARD RULES" rather than as till-go template caps. Misleading for future adopters reading the draft.
- §2.3 — SKETCH §23 / §24 / "OLD §25" numbering is out of order. Documentation hygiene.

**REFUTED attacks (17):**
1.1 wave decomposition under/over-sized; 1.2 missing dependencies (at sketch level); 1.4 specify under-constraint; 1.5 specify over-constraint; 1.6 untestable AcceptanceCriteria; 1.7 plan-down/build-up violation (at wave level); 1.8 atomic-droplet caps imposed at sketch level; 1.9 tillsyn-enforces-templates (at wave level); 1.10 default-templates-as-Tillsyn-behavior; 1.11 separation-of-concerns role bleed; 1.12 eat-own-dogfood; 1.13 memory-rule conflicts; 1.14 schema-without-consumer in sketch's own waves; 1.15 resolver-without-test; 1.16 hardcoded-vs-template boundary; 1.17 N=3 escalation; 1.18 token-cost claim direction; 1.19 multi-actor scenarios (deferred); 1.20 recursion-depth-bound default.

**Unknowns routed to dogfood measurement:**
- §1.18 — exact failure-synthesis token cost vs archived-children dump cost. Sketch's order-of-magnitude claim is plausible; exact numbers measurable post-Drop-4c.7+4c.8.
- §1.17 — N=3 escalation budget right-sizing. Tunable post-MVP per template; measurable during dogfood.
- §1.13 — `feedback_subagents_short_contexts.md` compaction-resilience for the wipe-and-replan + fresh-planner-spawn flow. Sketch §11.2 does not address; flag for the W10 planner spawn to consider OR for a future refinement.

**Plan-QA-falsification axis is satisfied** — every wave has well-bounded Objective + AcceptanceCriteria; the parallelization graph is implicit in §25 dependencies; the Specify blocks scale with droplet size; the cascade-tree shape is consistent with `CLAUDE.md § Cascade Tree Structure`.

**Closing certificate:**

- **Premises**: sketch v2.8.3 claims to land Drop 4c.6/7/8 as a methodology-coherent + dogfood-ready cascade architecture; planner-agent draft claims to honor `feedback_plan_down_build_up.md` + `feedback_tillsyn_enforces_templates.md` + `feedback_prompt_injection_team.md`.
- **Evidence**: SKETCH.md §1-§26; planner-agent draft full body; current code state at `internal/templates/child_rules.go` (resolver shipped, zero call sites confirmed via `rg`); `internal/app/dispatcher/cli_claude/render/render.go:340-364` (1-line stub confirmed); `internal/app/dispatcher/cli_adapter.go:102-179` (BindingResolved missing `SystemPromptTemplatePath` field confirmed); `internal/app/dispatcher/context/aggregator.go:243` (Resolve exists; consumers absent confirmed); `internal/templates/load.go:907` (`siblings_by_kind` validator confirmed; "latest per kind" semantics confirmed via `internal/app/dispatcher/context/rules.go:81`); `internal/app/service.go:1483-1518` (RestoreActionItem has SOME role-gate via `enforceMutationGuardAcrossScopes`).
- **Trace or cases**: 17 REFUTED attacks across decomposition / Specify-shape / methodology-conformance / memory-rule axes. 3 CONFIRMED minor counterexamples (§2.1, §2.2, §2.3).
- **Conclusion**: PASS. The sketch survives cascade-flavored plan-QA-falsification. Three minor fixes recommended before plan-down begins.
- **Unknowns**: token-cost ratios + N=3 right-sizing + compaction-resilience under wipe-and-replan flow — all routed to dogfood-measurement / future refinement.

---

## TL;DR

- **T1**: 20 attack axes attempted; 17 REFUTED, 3 CONFIRMED minor (cross-wave dep gap, planner-draft framing, sketch numbering); zero CONFIRMED counterexamples invalidate methodology.
- **T2**: §2.1 (W11 needs W7 system-managed marker) + §2.2 (planner draft mis-frames till-go template numbers as "non-negotiable") + §2.3 (sketch §23/§24 numbering out-of-order) are addressable as small Specify-block + doc-hygiene fixes during plan-down.
- **T3**: PASS. Sketch v2.8.3 survives cascade-flavored plan-QA-falsification; three minor fixes recommended before decomposition begins; three Unknowns routed to dogfood measurement.
