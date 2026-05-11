# Plan-QA-Falsification Round 1 — DROP_4c.6.W3

**Verdict:** FAIL — five confirmed counterexamples (1 high, 3 medium, 1 medium). Two of them are load-bearing implementation traps (embed-path resolution, stripModel pointer-always-non-nil semantics) that would cause the build to not compile or to silently mis-strip every render, respectively. The plan is otherwise well-shaped: cycles clean, package locks honored, L1↔L2 contracts mostly aligned, atomicity within budget for 4 of 6 droplets.

## 1. Findings

- 1.1 [Family: A3-hidden-coupling / contract-mismatch] [severity: high] **D2's prescribed `//go:embed builtin/agents` directive in `internal/app/dispatcher/cli_claude/render/render.go` cannot resolve `internal/templates/builtin/agents/till-<group>/`.** Go's `//go:embed` resolves paths RELATIVE to the source file's package directory. From `cli_claude/render/render.go`, `builtin/agents` would resolve to `internal/app/dispatcher/cli_claude/render/builtin/agents/`, NOT the `internal/templates/builtin/agents/` directory W1.D1 ships placeholders into (per L1 PLAN.md line 141 + L2 line 27 + L2 D2 RiskNotes line 119). → **Repro:** D2 acceptance line 104 says `embeddedAgentFS` is declared at the top of `render.go` with `//go:embed builtin/agents`; line 119 says "match where W1.D1 placed the placeholder dirs" but the plan never reconciles the relative-path mismatch. A fresh build with this directive as written produces `pattern builtin/agents: no matching files found` at compile time. → **Fix hint:** re-anchor the embed in `internal/templates/builtin/` (or a new `internal/templates/builtin/agents/embed.go`) as `var AgentsFS embed.FS` with `//go:embed agents`, then have `render.go` import `internal/templates/builtin/agents` and reference `agents.AgentsFS`. Update D2 acceptance + paths + KindPayload accordingly. The W1.D1 sub-plan likely already exposes such a handle — the L2 D2 plan should pin the import name explicitly rather than re-declare a duplicate embed in the wrong package.

- 1.2 [Family: A2-contract-mismatch / hidden-coupling] [severity: medium] **D3's `stripModel = binding.Model != nil` predicate is ALWAYS-TRUE under the current resolver, so `model:` is stripped on every render regardless of `agents.toml`.** Verified in `internal/app/dispatcher/binding_resolved.go:143-150`: `ResolveBinding` populates pointer-typed fields via `resolveStringPtr`, which on the no-override path returns `&v` where `v := rawValue` (line 170-171). Net: `BindingResolved.Model` is NEVER nil after `ResolveBinding`. D3 acceptance lines 139, 159, and KindPayload all pin the strip to `binding.Model != nil`. The L2 plan's RiskNotes line 159 actually catches this confusion ("Wait — D3's flag is 'did `agents.toml` SET the key' — the actual flag should reflect 'is the resolved `AgentRuntime.Model` non-nil?'") but punts to sub-planner instead of fixing the acceptance. → **Repro:** Today's `ResolveBinding` always populates `Model = &""` (or whatever rawBinding.Model is). With `binding.Model != nil` as the strip predicate, every render strips `model:` regardless of agents.toml content. → **Fix hint:** settle this in D3's acceptance — either (a) use `*binding.Model != ""` (treat empty-string-pointer-target as "not set"), or (b) introduce an explicit `BindingResolved.ModelExplicit bool` companion field plumbed by the resolver to discriminate "agents.toml set this" from "rawBinding default of empty string promoted to pointer." Option (a) is cheaper but inverts the F.7.17 L9 "explicit-zero is meaningful" pointer semantic for `Model`. Option (b) is more honest and survives future use cases where empty model string carries explicit-meaning. The choice belongs at PLAN time, not sub-planner time.

- 1.3 [Family: A3-hidden-coupling / shipped-but-not-wired] [severity: medium] **D5's Signal C is a hardcoded blacklist of one F.7.3b stub phrase; future drops that introduce a different stub-shaped body silently bypass the validator.** D5 acceptance line 220: "**Signal C — F.7.3b stub-key-phrase absent**: body does NOT contain the literal substring `\"Behavior loaded from the canonical\"`." That literal is from `render.go:360` in today's tree. After D2 rewrites the function, the literal disappears from production code BUT survives as a hardcoded constant in `validateBundle`. There is no automated guard linking the validator's hardcoded phrase to any future stub-shaped body. → **Repro:** Imagine a future drop reintroduces a thin-stub fallback (e.g., when embedded FS is missing) that writes `"Subagent loaded from system path"` instead of the F.7.3b phrase. Signal C silently passes; Signals A + B may both happen to pass on a thin body that exceeds 200 chars and has frontmatter. The validator approves the new stub. The "guards against regressions" framing fails. → **Fix hint:** redesign Signal C as a positive-signal check rather than a negative blacklist — e.g., assert the body contains a sentinel role-section header that every legitimate substantive agent body carries (per W4 prompt drafts: a `## Role` or `# Section 0` marker). Or, route the stub literal through a single package-level constant referenced by ANY future stub-fallback code AND by validateBundle, so future drops can't drift them apart silently. Update D5 acceptance + RiskNotes + KindPayload.

- 1.4 [Family: A3-hidden-coupling / RiskNotes internal inconsistency] [severity: medium] **D5's calibration argument in RiskNotes line 236 is logically incoherent under the documented AND-combined signal semantics.** Acceptance line 221: "validateBundle returns a non-nil error if ANY signal fails (logical OR over failure conditions; equivalently, all three must hold for pass)." So Signal A failing alone → validator fails. RiskNotes line 236 then says: "False-positive risk on legitimately-short bodies is mitigated by the stub-phrase Signal C — even a hypothetical short legitimate body wouldn't contain the F.7.3b stub phrase." That mitigation argument requires Signal C absence to OVERRIDE Signal A failure, which contradicts the AND semantics. A short legit body fails Signal A regardless of Signal C. → **Repro:** Trace the validator: `body = "Read-only investigation. Posts findings via comment. Dies."` → length 60 chars → Signal A fails (< 200) → validator fails. Signal C is satisfied (no F.7.3b phrase) but irrelevant under AND. The plan greenlights a calibration without acknowledging that Signal A's threshold IS the actual constraint and Signal C's absence is no rescue. → **Fix hint:** either (a) lower Signal A's threshold to e.g. 100 (the genuine "is this an empty stub" floor), (b) make the threshold conditional on Signal C presence (Signal A only fires when Signal C ALSO triggers), or (c) accept the false-positive risk for legit-short bodies and document that all default agents must have body length > 200 chars (W4 prompt-drafting constraint). Option (c) is cleanest if W4 prompt drafts are guaranteed > 200 chars; that needs to be stated.

- 1.5 [Family: A2-contract-mismatch / under-specification] [severity: medium] **D2's `<group>` derivation algorithm is under-specified and punts to sub-planner; two reasonable sub-planner interpretations ship divergent runtime behavior.** D2 acceptance line 101 lists the user-tier as `<user-home>/.tillsyn/agents/<group>/<basename>` where `<group>` is "derived from the resolved template name." Empty `binding.SystemPromptTemplatePath` defaults to `till-go` per a parenthetical "preferred: derive `<group>` from `binding.SystemPromptTemplatePath` parent dir if non-empty, else `till-go` per dogfood default." The contract leaves three ambiguities: (1) does `SystemPromptTemplatePath` always carry `till-<group>/<name>.md` or just `<name>.md`? (2) is `till-go` the right default for a generic adopter (it isn't — `till-gen` is)? (3) what if the project is `till-gdd`? → **Repro:** Sub-planner A reads "preferred: derive from parent dir" and ships `filepath.Dir(SystemPromptTemplatePath)` → empty → falls back to `till-go`. Sub-planner B interprets "default to `till-go` per dogfood default" as a Go-only default and adds `till-gen` switching for non-Go projects. Both implementations satisfy the acceptance prose but produce different lookups. For a fresh `till-gen`-bound project (e.g., a Python adopter), Sub-planner A misroutes to `till-go`, returning ENOENT; Sub-planner B routes correctly. → **Fix hint:** settle in this droplet's acceptance, NOT in sub-planner. Either (a) require `SystemPromptTemplatePath` to be the form `till-<group>/<name>.md` and derive `<group>` via `filepath.Dir`, with explicit failure if path lacks a parent dir, OR (b) add a sibling field `BindingResolved.Group string` plumbed via D1 alongside `SystemPromptTemplatePath`, sourced from the template's `[agents]` group declaration. Option (b) requires a D1 scope expansion; surface as a planner-level decision before the sub-planner spawns.

## 2. Counterexamples

- 2.1 **Embed path resolution failure (Finding 1.1).** Compile-time error reproducible by writing the prescribed directive verbatim:

  ```go
  // render.go (per D2 acceptance line 104)
  //go:embed builtin/agents
  var embeddedAgentFS embed.FS
  ```

  → `pattern builtin/agents: no matching files found`. Reproduction confirmed by Go embed semantics: directives resolve relative to the package source file's directory, not the module root. The render package directory `internal/app/dispatcher/cli_claude/render/` has no `builtin/` subdir; W1.D1's placeholders live in a different module subtree. Concrete fix: import the W1.D1 embed handle. PLAN must specify the exact import symbol (e.g., `"github.com/evanmschultz/tillsyn/internal/templates/builtin/agents".AgentsFS`).

- 2.2 **Always-on `model:` strip (Finding 1.2).** Trace through the resolver:
  1. Template author leaves `[agent_bindings.builder]` without an explicit `model = ...`.
  2. `rawBinding.Model = ""` (zero value).
  3. `ResolveBinding` calls `resolveStringPtr(rawBinding.Model, overrides, …)`.
  4. No overrides supply Model → fall through to `v := rawValue; return &v` (line 170-171 of `binding_resolved.go`).
  5. `BindingResolved.Model = &""` — non-nil pointer to empty string.
  6. D3's strip predicate `binding.Model != nil` evaluates TRUE.
  7. `StripFrontmatterKeys` strips `model:` from the embedded MD frontmatter.
  8. Net: every render strips `model:` even when agents.toml never set one.

  Concrete fix per Finding 1.2 above.

- 2.3 **Stub-key-phrase drift (Finding 1.3).** Future-drop simulation:
  1. Drop 4c.9 adds emergency-fallback in `assembleAgentFileBody` for the `embeddedAgentFS` miss case: writes `"Default subagent stub. Definition loaded from runtime config."` (250 chars; satisfies Signal A).
  2. Frontmatter still contains `name:` + `description:` → Signal B passes.
  3. Body does not contain literal `"Behavior loaded from the canonical"` → Signal C passes.
  4. Validator returns nil → spawn proceeds with a thin stub.
  5. Same isolation regression as F.7.3b, undetected.

  Concrete fix per Finding 1.3 above.

- 2.4 **AND-semantics breaks calibration argument (Finding 1.4).** Walked above — short legit body fails Signal A despite Signal C being satisfied, contradicting RiskNotes claim.

- 2.5 **Group-derivation ambiguity (Finding 1.5).** Two-interpretation simulation:
  - Adopter project bound to `till-gen` template, `[agent_bindings.builder] system_prompt_template_path = "till-gen/builder-agent.md"`.
  - Sub-planner A's derivation: `filepath.Dir("till-gen/builder-agent.md")` → `"till-gen"`. User-tier read: `~/.tillsyn/agents/till-gen/builder-agent.md`. Correct.
  - Sub-planner B's derivation: dogfood default → `till-go`. User-tier read: `~/.tillsyn/agents/till-go/builder-agent.md`. ENOENT for adopter who only seeded `till-gen` overrides.
  - Both implementations match the prose acceptance.

  Concrete fix per Finding 1.5 above.

## 3. Summary

**Verdict:** FAIL.

Five confirmed counterexamples — one high (embed path, build-breaking as written), three medium (`stripModel` always-true, Signal C drift, group-derivation under-specification), one medium (RiskNotes calibration argument internally inconsistent). All five have concrete repro paths and fix hints; none are dispositional disagreements. The cascade-shape, blocker graph, atomicity (within +1 LOC margin on D2/D5), and L1-vs-L2 contract alignment all pass.

The blocker chain D1 → D2 → D3 → D5 → D6 is correct and necessary (package compile lock + cross-droplet field/body dependencies). D4's parallel claim is correct (different package `cli_claude` vs `cli_claude/render`, different file `env.go`). D6's chain dep on D2+D3+D5 is mechanical package-lock compliance plus the planner's reasonable "author the doc-comment with full knowledge of the post-D2-D3-D5 shape" rationale (line 50). 6 droplets matches L1's enumerated shape.

The HF8 wiring contract on D5 (line 215, 246) is well-stated; the residual concern is only that the test-injection path is left to sub-planner and could exercise fixture-mutation rather than real production wiring (Findings sub-thread, not promoted to formal counterexample).

| Family                        | Result | Findings                |
| ----------------------------- | ------ | ----------------------- |
| A1. Concurrency / blocked_by  | PASS   | Acyclic, locks honored. |
| A2. Contract-mismatch         | FAIL   | 1.2, 1.5                |
| A3. Hidden-coupling           | FAIL   | 1.1, 1.3, 1.4           |
| A4. YAGNI / scope-creep       | PASS   | D6's chain justified.   |
| A5. Shipped-but-not-wired     | NIT    | HF8 test-injection punt — borderline, not promoted. |
| A6. Atomicity                 | NIT    | D2/D5 ~5 blocks each — within +1 of budget. |
| A7. Prompt-injection          | EXHAUSTED | DORMANT pre-team-feature. |
| Phase-2 missing blocked_by    | PASS   | Cross-package deps captured via field/body chain. |
| Phase-2 cycles                | PASS   | DAG verified.           |
| Phase-2 drift (L1 ↔ L2)      | PASS   | Line numbers verified, doc-comment ownership split is acceptable. |

Recommended next round: planner re-authors D2 acceptance to import the W1.D1 embed handle (Finding 1.1); D3 acceptance to fix the strip predicate semantics (Finding 1.2); D5 acceptance to redesign Signal C as positive-signal or constant-shared (Finding 1.3) and to fix the calibration argument (Finding 1.4); D2 acceptance to pin the `<group>` derivation contract (Finding 1.5). The other ~95% of the plan is sound and does not need rework.

## 4. Hylla Feedback

- **Query:** `hylla_search_keyword(query="AgentBinding SystemPromptTemplatePath", fields=["content", "docstring"])` — zero results.
- **Missed because:** Same blind spot the upstream RESEARCH/ISOLATION_ENFORCEMENT_FIX.md called out — Hylla's keyword index does not appear to catch field-declaration tokens for `templates.AgentBinding.SystemPromptTemplatePath`. Two queries tried (with and without dotted path); both empty.
- **Worked via:** Cross-referenced the field's existence via L2 PLAN.md's RiskNotes citation chain (research deliverable + AGENT_ARCHITECTURE_TRUTH.md §2.3 cross-ref) plus direct `Read` of `internal/app/dispatcher/binding_resolved.go` and `internal/config/frontmatter.go` for the resolver and helper signatures the L2 plan depends on.
- **Suggestion:** Symbol-level field-name search would close this gap — `templates.AgentBinding.SystemPromptTemplatePath` is exactly the dotted-path identifier dev queries by, but the keyword index seems to tokenize on whole-word boundaries and miss field-declaration sites entirely. Same gripe as RESEARCH file; durable until indexer ergonomics change.

- **Ergonomic gripe:** Bash invocations of `grep`, `find`, `awk` were uniformly denied by the agent permission gate during this review, forcing fallback to `Read` for line-by-line confirmation. Reasonable for a falsification reviewer (tool discipline), but heightens the cost of cross-validating L2 line citations against actual code. Per-file `Read` calls scaled fine for this 286-line plan + ~1250 LOC across 4 source files; for a larger plan the cost would balloon.

---

## Round 2 Verdict

**Reviewer:** L2 plan-QA-falsification agent (round 2)
**Plan under review:** `workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md` (round-2 patch)
**Date:** 2026-05-09
**Verdict:** **FAIL** — three confirmed counterexamples (one HIGH build-breaking on the dogfood path, two MEDIUM on the same Signal-C / placeholder-content seam) plus three NIT-level findings on YAGNI / wiring drift. Round-1 findings W3-FF1, W3-FF2, W3-FF4, W3-FF5 are RESOLVED by the round-2 patch. Round-1 W3-FF3's "redesign Signal C as positive-presence" is partially addressed but the chosen markers (`# Section 0` / `## Role`) do not exist in any W1.D1-shipped placeholder, which converts the round-1 fail-open risk into a round-2 fail-CLOSED runtime regression.

### 1. Findings (Round 2)

- 1.1 **W3-FF6** [Family: A3-hidden-coupling / contract-mismatch] [severity: **HIGH**] **D5 Signal C as locked at PLAN line 224 fails-CLOSED on EVERY current W1.D1 placeholder, so the moment D5 lands the entire bundle render breaks for every default-template-bound spawn.** PLAN line 224 says Signal C requires the body to contain `"# Section 0"` OR `"## Role"`. I read every placeholder under `internal/templates/builtin/agents/till-go/` (12 files), `till-gen/` (8 files), `till-gdd/` (7 files) — **NONE contain either marker**. All 27 placeholders use the body header `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4`, NOT either Signal-C marker. → **Repro:** Once D5 lands, `Render()` calls `validateBundle()` at `render.go`'s exit; `validateBundle` reads the rendered `<bundle>/plugin/agents/<name>.md` (which D2 sourced from a W1.D1 placeholder); Signal C check `body contains "# Section 0" || "## Role"` returns false → validator fails → `Render()` returns `ErrInvalidAgentBody` → rollback → spawn aborts. Every default-template-bound spawn fails this way until W4 (Drop 4c.8) lands substantive prompt content. Also: PLAN line 224 hedges with "sub-planner verifies via Read of any one placeholder ... if the actual placeholder convention differs, sub-planner picks the actual marker present in the placeholders and routes the choice through this droplet's authoring decision" — but the actual placeholder convention is `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4`, which is a placeholder marker, not a "role" marker. Locking the validator on that string would itself be brittle (any change in W4's prompts that drops the literal "PLACEHOLDER" would re-introduce fail-open behavior). The right fix moves Signal C to a forward W4-deliverable contract or relaxes it to accept a forward-floor (e.g., "body contains AT LEAST one `## ` header line" + "body contains AT LEAST one `# ` h1") that any plausible substantive prompt would carry. → **Fix hint:** acknowledge in D5's Acceptance + RiskNotes that Signal C's positive markers MUST be present in BOTH the W1.D1 placeholder set AND the W4-drafted prompts. Two viable paths: (a) add `# Section 0` + `## Role` markers to ALL 27 W1.D1 placeholders as a SCOPE EXPANSION on D2 (or a new D2.5 droplet) so D5's check has live ground to stand on; (b) RELAX Signal C to a "body contains AT LEAST one `# ` markdown h1 header AFTER the closing frontmatter `---\n`" + a positive `# Section 0` / `## Role` requirement DEFERRED to Drop 4c.8 W4-D where W4 substantive prompts ship. Option (a) couples D5 to a placeholder-edit; Option (b) keeps W3 atomic but ships a weaker Signal C floor. Either way, the current PLAN's "every legitimate W4-drafted agent body MUST exceed 200 chars" line at RiskNote line 240 is a forward W4 requirement that does NOT mitigate the round-1 W3 regression on the W1.D1-placeholder-RENDERED bodies that D5 will validate THIS DROP.

- 1.2 **W3-FF7** [Family: A2-contract-mismatch] [severity: **HIGH**] **D2's `<group>` derivation rule combined with the SHIPPED `till-go.toml` `[agent_bindings.*]` rows produces an ENOENT-at-embedded-tier for every coordination-kind spawn (closeout / refinement / discussion / human-verify).** Verified in `internal/templates/builtin/till-go.toml`: NO `[agent_bindings.*]` row declares `system_prompt_template_path` — every binding leaves it empty. Per PLAN line 102 (LOCKED rule), empty `binding.SystemPromptTemplatePath` → `<group> = "till-go"`. The four coordination-kind bindings (`[agent_bindings.closeout]`, `[agent_bindings.refinement]`, `[agent_bindings.discussion]`, `[agent_bindings.human-verify]`) all set `agent_name = "orchestrator-managed"`. So D2 reads `builtin/agents/till-go/orchestrator-managed.md` from `templates.DefaultTemplateFS`. **But the embed.go directives at lines 77-103 place `orchestrator-managed.md` only at `builtin/agents/till-gen/orchestrator-managed.md`** (verified at `embed.go:103`) — the `till-go/` directory does NOT contain `orchestrator-managed.md`. Net: every coordination-kind spawn that exercises D2's resolver will return `ErrAgentBodyNotFound` from the embedded tier, render fails, spawn aborts. → **Repro:** Spawn dispatches through `Render` for any of the four orchestrator-managed kinds → `assembleAgentFileBody(project, binding)` called with `binding.SystemPromptTemplatePath = ""` and `binding.AgentName = "orchestrator-managed"` → `<group> = "till-go"` → `fs.ReadFile(templates.DefaultTemplateFS, "builtin/agents/till-go/orchestrator-managed.md")` → fs.ErrNotExist → resolver wraps as `ErrAgentBodyNotFound` → Render fails. → **Fix hint:** D2's `<group>` derivation must account for the cross-group resolution case where the same `agent_name` lives in a different group. Three viable paths: (a) extend the resolver to walk `till-go` first and then fall back to `till-gen` (cross-group fallback ordering — needs a deliberate decision); (b) thread the `<group>` through `BindingResolved` as a first-class field via D1.5 / D1 expansion (so `agents.toml` / template authors can pin `group = "till-gen"` on the orchestrator-managed bindings); (c) move `orchestrator-managed.md` placeholder into `till-go/` as well (duplicate-shipping per group, matching the existing `go-builder-agent.md` etc. pattern). Option (c) is the smallest change and preserves the "embedded tier resolved purely from `<group>`" rule. Option (b) is the most honest but expands D1's scope. Either fix needs a Decision ContextBlock + KindPayload update on D2. The plan currently silently assumes `agent_name` matches exactly one file under `till-<group>/`; that assumption breaks for the four coordination-kind bindings the SHIPPED template carries.

- 1.3 **W3-FF8** [Family: A3-hidden-coupling / shipped-but-not-wired] [severity: medium] **D5's "every legitimate W4-drafted agent body MUST exceed 200 chars" floor is declared in RiskNotes (PLAN line 240) but never propagates to any W4 acceptance contract that this plan could enforce.** PLAN line 240 says "the W4 prompt-length floor is documented as an ACCEPTED constraint, eliminating the AND-semantics calibration incoherence" and "W4 sub-planner verifies all 7 agent placeholder bodies clear the 200-char floor at authoring time." But: (a) W4 is Drop 4c.8 — not in this drop, not in any plan-QA-falsification's review surface today, no enforcement mechanism this plan can wire; (b) the assertion "all 7 agent placeholder bodies" mis-counts — the L1 PLAN's W1.D1 ships **21 standard placeholders + 6 legacy** (3 groups × 7 names + 5 legacy `go-*-agent.md` in `till-go/` + 1 legacy `orchestrator-managed.md` in `till-gen/` = 27 total — confirmed via `embed.go` directives at lines 77-103); (c) RiskNote line 240's "W4 sub-planner verifies" is a forward-honor-system requirement, not a verifiable contract. Net: if W4 ships any placeholder body that doesn't clear 200 chars body-length-after-frontmatter, D5's Signal A fails-closed identically to W3-FF6. → **Repro:** A future W4 sub-planner ships a deliberately-minimal `qa-falsification-agent.md` body (e.g., 150 chars) thinking it satisfies the "placeholder" framing → `validateBundle` Signal A fails → spawn aborts. The plan's only mitigation is a RiskNote that lives in W3, has no W4-enforceable mechanism, and is invisible to the W4 sub-planner unless someone manually surfaces it. → **Fix hint:** D5's Acceptance should add a dependency artifact: a `RESEARCH/W4_PROMPT_FLOOR_CONTRACT.md` or equivalent forward-pinned doc that the W4 sub-planner Reads as a hard-input. Or relax Signal A to a sentinel-stub-shape detector (length <= 50 chars body OR body equals `Tillsyn-spawned subagent stub. Behavior loaded ...` literal) so the validator only catches the actual stub regression that motivated D5, not legitimate-but-short prompts.

- 1.4 **W3-FF9** [Family: A2-contract-mismatch] [severity: medium] **D3's `stripModel = binding.Model != nil && *binding.Model != ""` LOCK changes pre-existing F.7.17 L9 pointer-semantics for the `model:` field.** PLAN line 143 + RiskNote line 163 lock the predicate as `binding.Model != nil && *binding.Model != ""`. The "&& *binding.Model != ""` clause treats EMPTY-STRING as "not set" — but per the F.7.17 L9 lock cited at `cli_adapter.go:96-101` (per the round-1 PLAN's own evidence chain), pointer-typed fields on `BindingResolved` are explicitly designed so EXPLICIT ZERO is meaningful (e.g., `MaxBudgetUSD = &0.0` means "no spend allowed", NOT "fall through to next layer"). Round-1 falsification 1.2's Fix Option (a) flagged this exact tension — round-2 picked Option (a) without acknowledging the F.7.17 L9 semantic inversion for `Model`. → **Repro:** Adopter sets `[agent_bindings.builder] model = ""` in their `agents.toml` deliberately to express "agents.toml chose empty model on purpose" (exotic but valid per F.7.17 L9 explicit-zero-is-meaningful). `ResolveBinding` populates `Model = ptr("")`. D3's predicate evaluates FALSE (because `*binding.Model == ""`). Net: `model:` from the embedded MD frontmatter is NOT stripped — embedded `model: opus` survives despite the adopter's intent to override. The plan accepts this as a "user-error case settled in QA" (line 143), but that's a behavior claim that the F.7.17 L9 lock doesn't carry. → **Fix hint:** EITHER (a) accept the F.7.17 L9 semantic inversion explicitly in D3's Decision ContextBlock — name it as "for `Model` only, empty-string-pointer-target is treated as 'not set' overriding F.7.17 L9 explicit-zero-is-meaningful, because empty model name has no runtime semantic"; OR (b) introduce the `BindingResolved.ModelExplicit bool` companion field per round-1 finding 1.2 Fix Option (b), which preserves F.7.17 L9 and adds a deterministic strip flag. Option (a) is cheaper but is a documented divergence from F.7.17 L9 — surface it explicitly so future readers don't trip on the inconsistency. Option (b) is more honest but expands D1's scope. Either way, the plan needs to NAME the divergence rather than pretend `*binding.Model != ""` is consistent with F.7.17 L9.

- 1.5 **W3-FF10** [Family: A3-hidden-coupling] [severity: nit] **The pre-existing happy-path render tests at `render_test.go:113`, `:144`, `:335`, `:370` use `binding.AgentName = "go-builder-agent"`; after D2 lands, those tests' rendered agent file bodies will be sourced from `internal/templates/builtin/agents/till-go/go-builder-agent.md` placeholder content. After D5 lands, those happy-path tests will FAIL because Signal C is unsatisfied per W3-FF6.** D5's Acceptance enumerates four NEW tests but does NOT enumerate the existing happy-path tests as needing updates. → **Repro:** `mage test-pkg ./internal/app/dispatcher/cli_claude/render` after D5 lands → `TestRenderHappyPathWritesAllFiveFiles` calls `Render()` → validator fails on placeholder body → test asserts `err == nil` → test fails. → **Fix hint:** D5 (or a D5-companion test-update droplet) audits the existing render-package tests AND lists the test-fixture mutations needed so they remain green. The HF8 wiring contract requires the test surface to assert validator behavior end-to-end via `Render()`; that means the existing happy-path tests are now part of the validator's test surface and need fixture bodies that pass Signals A+B+C. Most efficient fix: every existing happy-path test fixture's placeholder body gains a `## Role\n` header line so Signal C passes (touches placeholder MDs OR test-injection seam — sub-planner picks).

- 1.6 **W3-FF11** [Family: A3-hidden-coupling / RiskNotes drift] [severity: nit] **D6's RiskNote line 282 retains "F.7.2 + 4c.6 W3.D2 landed the field-and-resolver-wired version" wording but D2 itself is the consumer of `binding.SystemPromptTemplatePath`, not the wiring landing.** F.7.2 landed the field on `templates.AgentBinding`; D1 wires it through `BindingResolved`; D2 consumes it for resolution. D6's wording conflates "the field landed" with "the resolver landed" — both are accurate facts but the breadcrumb composes them oddly. → **Repro:** Future-reader trying to understand the architectural-history breadcrumb at `render.go:321-325` reads the breadcrumb claim "F.7.2 + 4c.6 W3.D2 landed the field-and-resolver-wired version" → looks up F.7.2 → sees only the field landing on `templates.AgentBinding`, not on `BindingResolved` → confused about WHERE the resolver lives. → **Fix hint:** D6's Acceptance line 265 + RiskNote line 282 wording: "F.7.2 landed `templates.AgentBinding.SystemPromptTemplatePath`; 4c.6 W3.D1 wired it through `dispatcher.BindingResolved`; 4c.6 W3.D2 landed the 3-tier resolver consuming the field. Comment retained as architectural-history breadcrumb." Three steps named, not collapsed into "F.7.2 + 4c.6 W3.D2."

### 2. Counterexamples (Round 2)

- 2.1 **W3-FF6 reproduction.** Verified by direct Read of all 27 W1.D1 placeholders in `internal/templates/builtin/agents/{till-go,till-gen,till-gdd}/`. Sample evidence:
  - `internal/templates/builtin/agents/till-go/builder-agent.md` (385 bytes total): body after closing `---\n\n` is `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4\n\nThis file is a Drop 4c.6 W1.D1 scaffolding placeholder. Its only purpose is to\nlet the embedded-FS resolver path land before Drop 4c.8 W4 authors the\nsubstantive prompt content.\n` — ~270 chars. No `# Section 0`, no `## Role`. Body length passes Signal A (270 > 200) but body fails Signal C.
  - `internal/templates/builtin/agents/till-go/closeout-agent.md` (387 bytes): identical body shape — passes A, fails C.
  - `internal/templates/builtin/agents/till-go/qa-proof-agent.md` (400 bytes): identical body shape — passes A, fails C.
  - All 21 standard + 6 legacy placeholder files share the exact same body header `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4`.

  Signal C as locked in the round-2 plan is `body contains "# Section 0" || body contains "## Role"`. Neither marker appears in any of the 27 placeholder files. Validator fail-closed for every default-template render after D5 lands. Concrete fix per Finding 1.1 above.

- 2.2 **W3-FF7 reproduction.** Trace through `till-go.toml` + `embed.go`:
  1. Coordination-kind spawn dispatched (e.g., `[agent_bindings.closeout]`).
  2. `binding.AgentName = "orchestrator-managed"`, `binding.SystemPromptTemplatePath = ""` (no row in `till-go.toml` declares it).
  3. D2 resolver: empty path → `<group> = "till-go"` per LOCKED rule (PLAN line 102).
  4. Project tier: `<project>/.tillsyn/agents/orchestrator-managed.md` — adopter probably hasn't shipped this; fall through.
  5. User tier: `~/.tillsyn/agents/till-go/orchestrator-managed.md` — adopter probably hasn't shipped this; fall through.
  6. Embedded tier: `fs.ReadFile(templates.DefaultTemplateFS, "builtin/agents/till-go/orchestrator-managed.md")` → fs.ErrNotExist (file is at `builtin/agents/till-gen/orchestrator-managed.md` per `embed.go:103`, NOT in `till-go/`).
  7. Resolver wraps as `ErrAgentBodyNotFound` → `Render` fails → spawn aborts.

  Concrete fix per Finding 1.2 above.

- 2.3 **W3-FF8 reproduction.** A future W4 sub-planner ships a thin placeholder for a research-agent body:
  ```
  ---
  name: research-agent
  description: Read-only investigation agent.
  ---

  Posts findings via comment. Dies cleanly.
  ```
  Body length: ~40 chars. Signal A fails. Validator returns `ErrInvalidAgentBody`. Render aborts. The plan's claim that W4's "200-char floor" mitigates this is a forward-honor-system claim with no enforcement seam.

- 2.4 **W3-FF9 reproduction.** Adopter writes `[agent_bindings.builder] model = ""` deliberately. `ResolveBinding` populates `Model = ptr("")`. D3 predicate `*binding.Model != ""` is FALSE. `model:` from embedded MD frontmatter is NOT stripped. Adopter expects strip; doesn't get it. The "user-error" framing in PLAN line 143 silently accepts this divergence from F.7.17 L9 explicit-zero-is-meaningful semantic without surfacing it.

- 2.5 **W3-FF10 reproduction.** `mage test-pkg ./internal/app/dispatcher/cli_claude/render` after D5 lands:
  1. `TestRenderHappyPathWritesAllFiveFiles` calls `render.Render(...)` with `fixtureBinding()` (`AgentName = "go-builder-agent"`).
  2. D2 resolves to `builtin/agents/till-go/go-builder-agent.md` (608 bytes per ls listing); body after frontmatter is the placeholder content `# PLACEHOLDER — substantive content lands in Drop 4c.8 W4\n\nThis placeholder satisfies the W0.5 ...`.
  3. D3 strip wiring runs — `binding.Model` is non-nil pointer to empty string per `fixtureBinding()` zero value → strip skipped per W3-FF2 fix.
  4. D5 validator runs → Signal C fails (no `# Section 0`, no `## Role` in placeholder body).
  5. Render returns `ErrInvalidAgentBody`.
  6. Test asserts `err == nil` → test FAILS.

### 3. Round-1 Resolution Status

| Round-1 Finding | Round-2 Patch Outcome | Notes |
| --------------- | -------------------- | ----- |
| W3-FF1 (embed path resolution) | RESOLVED | PLAN lines 97 + 105 + 120 + 129-130 explicitly consume `templates.DefaultTemplateFS` via `fs.ReadFile`; NO render-package-local `//go:embed` directive. Import path `github.com/evanmschultz/tillsyn/internal/templates` confirmed acyclic via `embed.go` import block (stdlib leafs only). |
| W3-FF2 (always-true `model:` strip) | RESOLVED | PLAN line 143 + RiskNote line 163 + Decision ContextBlock line 169 lock `stripModel = binding.Model != nil && *binding.Model != ""`. Note: introduces F.7.17 L9 semantic divergence — see new W3-FF9. |
| W3-FF3 (Signal C fail-open blacklist) | PARTIALLY ADDRESSED → introduces W3-FF6 fail-CLOSED runtime regression | Round-2 redesigned to positive-presence (`# Section 0` OR `## Role`) but the chosen markers are absent from EVERY current W1.D1 placeholder. Old fail-open replaced with new fail-closed. |
| W3-FF4 (AND-semantics calibration) | RESOLVED | PLAN RiskNote line 240 explicitly removes the incoherent claim ("the prior ROUND-1 RiskNote claim that 'Signal C mitigates short-body false positives on Signal A' was logically incoherent and is REMOVED"). Replaced with the W4-prompt-length-floor framing — but that framing is a forward-honor-system claim — see new W3-FF8. |
| W3-FF5 (`<group>` under-spec) | RESOLVED for the empty-vs-non-empty branch | PLAN line 102 + Decision ContextBlock line 128 lock the rule. But the LOCK exposes new W3-FF7 cross-group ENOENT for orchestrator-managed kinds. |

### 4. Severity Breakdown

| Severity | Count | IDs |
| -------- | ----- | --- |
| HIGH     | 2     | W3-FF6, W3-FF7 |
| MEDIUM   | 2     | W3-FF8, W3-FF9 |
| NIT      | 2     | W3-FF10, W3-FF11 |

### 5. Summary

**Verdict: FAIL.** Two HIGH-severity counterexamples (W3-FF6, W3-FF7) — both build-breaking on the dogfood path (`till-go` template + go-builder-agent / coordination-kind bindings) the moment D2+D5 land. Two MEDIUM (W3-FF8, W3-FF9) — forward-honor-system / semantic-divergence concerns. Two NIT (W3-FF10, W3-FF11) — test-update enumeration gaps + breadcrumb wording drift.

The round-2 patch successfully closed round-1 W3-FF1 (embed-path), W3-FF2 (model-strip predicate), W3-FF4 (AND-calibration), and W3-FF5 (`<group>`-empty-vs-non-empty). The remaining round-1 issue W3-FF3 (Signal C) was partially addressed but the redesign introduced a new fail-closed runtime regression (W3-FF6) that is more severe than the original fail-open: round-1's fail-open meant "future stub-regressions silently pass"; round-2's fail-closed means "EVERY current default-template render fails today." That's a strict regression on the dogfood spawn path.

Recommended next round: address W3-FF6 by either (a) adding `# Section 0` / `## Role` markers to ALL 27 W1.D1 placeholders as a SCOPE EXPANSION on D2 (or a new D2.5 droplet), or (b) RELAX Signal C to a marker-shape floor that any plausible substantive prompt would carry; address W3-FF7 by either (a) extending the resolver with cross-group fallback ordering, (b) plumbing `<group>` as a first-class field via D1 expansion, or (c) duplicate-shipping `orchestrator-managed.md` into `till-go/` placeholder set. The MEDIUM and NIT findings are absorbable on the same respin without new droplets.

### 6. Hylla Feedback

- **Query:** `hylla_search_keyword(query="binding.Model render", fields=["content"])` — returned only `internal/tui/Model.View` method, unrelated. Two queries against the dispatcher's `BindingResolved.Model` predicate site returned zero relevant hits.
- **Missed because:** Same field-declaration-tokens blind spot called out in round-1 + RESEARCH/ISOLATION_ENFORCEMENT_FIX.md. Hylla's keyword index doesn't tokenize on dotted-path field references like `binding.Model` or `BindingResolved.Model` — both queries returned the unrelated `tui/Model.View` method instead.
- **Worked via:** Direct `Read` of `internal/app/dispatcher/binding_resolved.go` and `internal/app/dispatcher/cli_claude/render/render_test.go` for the Model-pointer-semantics evidence chain.
- **Suggestion:** Same as round-1 — symbol-level field-declaration search would close this gap. Compounding suggestion: a "field-on-type" search mode where the query `BindingResolved.Model` returns the field declaration site + every read/write site within the artifact, with usage-context excerpts, would be high-value for round-trip semantic verification of the "is this pointer ever nil?" question.
- **Query:** `hylla_search(query="SystemPromptTemplatePath AgentBinding template field", search_types=["vector"])` — returned `enrichment still running for github.com/evanmschultz/tillsyn@main`.
- **Missed because:** Vector enrichment is per-snapshot and doesn't gracefully degrade to keyword fallback on miss; also the parameter shape error (`field` vs `fields` plural — error said "field must be summary, content, or docstring" when I passed `fields: ["content"]`).
- **Worked via:** Direct Read of `internal/templates/builtin/till-go.toml` to confirm absence of `system_prompt_template_path` rows.
- **Suggestion:** Hylla's search-types parameter shape inconsistency between `fields` (plural array) and `field` (singular string) is a footgun. Round-1 falsification didn't flag it; round-2 hit it on first-try.

- **Ergonomic gripe:** Bash `grep`, `find`, `awk` invocations were uniformly denied by the agent permission gate (same as round-1). Forced to direct-Read 27 placeholder files individually. Acceptable but expensive — `Bash` access for read-only commands like `grep -l "..." path/*.md` would have collapsed five Read calls into one. Round-1 raised this; round-2 confirms.

---

## Round 3 Verdict

**Reviewer:** L2 plan-QA-falsification agent (round 3)
**Plan under review:** `workflow/drop_4c_6/DROP_4c.6.W3_BUNDLE_AND_ISOLATION/PLAN.md` (round-3 in-place edit)
**Date:** 2026-05-09
**Verdict:** **PASS_WITH_FINDINGS** — round-3 patch closes round-2 W3-FF6 (3-marker disjunction validates today's W1.D1 placeholders), W3-FF7 (cross-group fallback ladder routes coordination-kind spawns), W3-FF10 (existing-test fixture preservation verified inline), W3-FF11 (Future-evolution breadcrumb expanded). One MEDIUM-severity counterexample (W3-FF12) on the strip-then-inject pipeline's silent-leak hole, plus three NIT-level findings on Signal C surface area, YAGNI on the 3-marker disjunction, and W3-FF9 forward-dep enforcement gap. None of the new findings are build-breaking; W3-FF12 fails the existing `TestRenderAgentFileWithoutToolGating` contract under a specific accident-prone disk-author scenario, so flagging as MEDIUM rather than NIT.

### 1. Findings (Round 3)

- 1.1 **W3-FF12** [Family: A2-contract-mismatch / hidden-coupling] [severity: **MEDIUM**] **D3's strip-then-inject pipeline does NOT strip stale `allowedTools:` / `disallowedTools:` from disk frontmatter when `binding.ToolsAllowed` AND `binding.ToolsDisallowed` are both empty (binding has no tool gates). This silently leaks accidentally-authored frontmatter from a placeholder MD into the rendered file, breaking the existing `TestRenderAgentFileWithoutToolGating` contract.** PLAN line 145 specifies `stripTools = len(binding.ToolsAllowed) > 0 || len(binding.ToolsDisallowed) > 0` — i.e. strip ONLY fires when binding has at least one tool-gate entry. The strip universe in `internal/config/frontmatter.go:51` is `frontmatterToolsKeys = []string{"tools", "allowedTools", "disallowedTools"}` — when `stripTools=false`, NONE of these keys are stripped; verified at `frontmatter.go:91-93` ("no-op short-circuit: return verbatim to preserve exact bytes" when both flags false). Per PLAN line 153, "Empty `binding.ToolsAllowed` / `binding.ToolsDisallowed` skip injection (mirroring `TestRenderAgentFileWithoutToolGating` contract)." Combined: if a future drop's W1.D1 placeholder author adds a stale `allowedTools: SomeTool` line to a placeholder MD frontmatter (e.g. accidentally during a copy-paste from a substantive prompt template), the strip pipeline does NOT remove it (binding has no gates → stripTools false → strip is no-op short-circuit), AND the inject pipeline does NOT run (binding has no gates → skip injection). Net: stale `allowedTools: SomeTool` survives verbatim into rendered file. → **Repro:** modify `internal/templates/builtin/agents/till-go/go-builder-agent.md` frontmatter to add `allowedTools: Bash` between line 2 and line 3 (between `name:` and the closing `---`); construct `BindingResolved{AgentName: "go-builder-agent", CLIKind: CLIKindClaude}` with empty `ToolsAllowed` + `ToolsDisallowed`; call `Render`. Post-D3 rendered file at `<bundle>/plugin/agents/go-builder-agent.md` contains `allowedTools: Bash` (leaked from disk verbatim). `TestRenderAgentFileWithoutToolGating` at `render_test.go:366-401` asserts `!strings.Contains(str, "allowedTools:")` — FAILS. The plan's "skip injection mirrors TestRenderAgentFileWithoutToolGating" claim is correct only when the disk source frontmatter ALSO has no `allowedTools:` / `disallowedTools:` keys; a defensive disk-source contract requires unconditional strip of the tool-gating keys regardless of binding state, with conditional inject layered on top. → **Fix hint:** decouple strip from binding state for the tool-gating keys. Two viable paths: (a) **always strip tool-gating keys** — D3 unconditionally calls `StripFrontmatterKeys(frontmatter, stripModel, true)` for the tools axis (always strip) regardless of binding.ToolsAllowed/ToolsDisallowed length, then conditionally inject from binding when non-empty. The strip universe is exactly the runtime-owned axis; defense-in-depth strip blocks accidental-disk-leak paths. (b) **add a per-strip-axis config** to `StripFrontmatterKeys` so callers can request "always strip tools, conditionally strip model" without overloading a single bool. Option (a) is the minimal change and matches the SKETCH § 4.4 "runtime owns model+tools surface" framing. Update D3's Acceptance bullets at PLAN lines 145-148, RiskNote line 171 (strip-then-inject ordering), and Decision ContextBlock line 178.

- 1.2 **W3-FF13** [Family: A3-hidden-coupling / over-permissive Signal C] [severity: nit] **D5's Signal C performs unanchored substring matching for `"# PLACEHOLDER"` / `"# Section 0"` / `"## Role"`, allowing a thin-stub body to bypass detection by quoting the marker inside a code-fence or paragraph.** PLAN line 234 + line 251 specify Signal C as `body contains AT LEAST ONE of "# PLACEHOLDER" OR "# Section 0" OR "## Role"` — `strings.Contains` semantics, no head-of-line anchoring, no exclusion of code-fenced content. → **Repro:** a future drop reintroduces a stub-shape body shaped like `"This is a stub. The substantive prompts use \`# Section 0\` headers — pending Drop 4c.8 W4. Behavior loaded from system path."` — body length ~150 chars, BUT the literal `"# Section 0"` substring is present (inside backticks, not as a heading). Signal A fails (length < 200) so this specific repro is caught by Signal A. The harder repro: body length 220 chars including a similar quoted-marker phrase. Signal A passes, Signal B passes (frontmatter intact), Signal C passes (substring match). Validator green-lights a stub body that happens to QUOTE the marker. → **Fix hint:** anchor Signal C to head-of-line — replace `strings.Contains(body, "# PLACEHOLDER")` with `strings.HasPrefix(line, "# PLACEHOLDER ")` over body lines (i.e., regex-equivalent `(?m)^# PLACEHOLDER `, `(?m)^# Section 0`, `(?m)^## Role`). Sub-planner picks line-by-line walk (no regex import needed). NOT load-bearing today (W1.D1 placeholders + drop 4c.8 W4 substantive prompts both use head-of-line markers), but tightening today eliminates a future false-pass surface. Update D5 Acceptance line 234 + RiskNote line 251.

- 1.3 **W3-FF14** [Family: A4-YAGNI / scope-creep] [severity: nit] **D5 Signal C's 3-marker disjunction (`# PLACEHOLDER` + `# Section 0` + `## Role`) is over-specified relative to the dual-state surface (today: placeholder; post-W4: substantive). Two markers would suffice; three increases the test-fixture matrix and the documentation burden without buying additional signal.** PLAN line 234 lists three positive markers; the rationale spans ~30 lines (lines 234, 251, 260). The role-split is: `# PLACEHOLDER` validates W1.D1 today AND any future placeholder-shipping drop; `# Section 0` validates Drop 4c.8 W4 substantive prompts; `## Role` validates "substantive prompts that use role-prose convention." But the third marker (`## Role`) has no concrete W4 design contract anchoring it — the W4 PLAN at the L1 level (`workflow/drop_4c_6/PLAN.md` lines ~155-200) does not pin role-prose-convention as a W4 deliverable. So `## Role` is speculative future-flexibility. → **Repro:** sub-planner Reads the L1 W4 plan, finds no `## Role` anchor, ships D5 with 2-marker disjunction (`# PLACEHOLDER` + `# Section 0`). Equivalent validator behavior, simpler test surface. → **Fix hint:** drop `## Role` from the disjunction unless a forward-pinned W4 sub-planner contract requires it. Plan can hedge with "open for W4 sub-planner to add additional markers if substantive prompts adopt a `## Role` convention" — but locking-the-marker-set today is plan-time-discipline. Sub-planner proposes 2-marker disjunction as the minimal-viable shape; orchestrator-respin gate decides. NOT blocking; reasonable people disagree. NIT level.

- 1.4 **W3-FF15** [Family: A5-shipped-but-not-wired / forward-dep enforcement] [severity: nit] **W3-FF9 round-3 resolution declares `model = ""` as user-error routed to "W0/W0.5 validator warning" (PLAN RiskNote line 169), but no W0/W0.5 validator pass actually exists for this case — the forward dep is unenforced and the plan does not name a refinement-routing seam.** Verified by Read of `internal/templates/load.go:1031-1055` (the AgentBinding validator entry point per AGENT_ARCHITECTURE_TRUTH.md § 2.3): `validateAgentBindingNames` validates AGENT_NAME existence in the embedded-tier (and project-tier post-W0.5), NOT the `model:` field's empty-string state. PLAN line 169's "W0/W0.5 validator warning flags `model = ""`" is a forward-honor-system claim with no implementation seam in W0/W0.5 (both shipped) and no refinement-tracker entry pinning a future implementation. → **Repro:** an adopter writes `[agent_bindings.builder] model = ""` in `agents.toml` deliberately (per W3-FF9 round-2 falsification's exotic-but-valid case) → no validator fires → adopter expects override behavior, gets pass-through (per round-3's locked predicate `*binding.Model != ""` evaluating FALSE) → adopter is confused; no diagnostic surfaces. → **Fix hint:** EITHER (a) raise the gap as a refinement-tracker entry routed to a future W0.5 backfill drop (cleanest), OR (b) add the warning to W3 itself as a small validator extension in D1's scope (expands D1; reasonable since D1 is the field-plumbing site), OR (c) ACCEPT explicitly that "no diagnostic fires for `model = ""`" and remove the "forward dep" framing from PLAN line 169 since no future drop is contracted to add it. Option (a) is the lowest-friction fix; sub-planner adds the refinement-tracker entry alongside D1's commit. NOT blocking W3 ship; the gap is documented somewhere else (or accepted) without changing W3 scope.

### 2. Counterexamples (Round 3)

- 2.1 **W3-FF12 reproduction.** Concrete trace through D3 + `StripFrontmatterKeys`:
  1. Future-drop placeholder author adds `allowedTools: Bash` to `internal/templates/builtin/agents/till-go/go-builder-agent.md` between line 2 (`name:`) and the closing `---` on line 4 — accidentally during copy-paste from a substantive prompt draft.
  2. Test invokes `render.Render(...)` with `BindingResolved{AgentName: "go-builder-agent", CLIKind: CLIKindClaude}` — empty `ToolsAllowed` + `ToolsDisallowed` (the `TestRenderAgentFileWithoutToolGating` shape at `render_test.go:374-378`).
  3. D2 resolves the body from disk: full content of `till-go/go-builder-agent.md` including the leaked `allowedTools: Bash` frontmatter line.
  4. D3 computes `stripTools = len([])>0 || len([])>0` = `false`. Computes `stripModel` = `binding.Model != nil && *binding.Model != ""` — `binding.Model` is nil (BindingResolved zero value with no resolver call) — so `stripModel = false`.
  5. D3 calls `config.StripFrontmatterKeys(frontmatter, false, false)` → no-op short-circuit at `frontmatter.go:91-93`, returns frontmatter verbatim.
  6. D3 inject step: `binding.ToolsAllowed` is empty, skip; `binding.ToolsDisallowed` is empty, skip. No injection.
  7. D3 emits frontmatter unchanged: still contains `allowedTools: Bash`.
  8. `TestRenderAgentFileWithoutToolGating` asserts `!strings.Contains(str, "allowedTools:")` — FAILS.

  Concrete fix per Finding 1.1 above (always-strip-tool-keys + conditional inject).

- 2.2 **W3-FF13 reproduction.** Future stub-shape body construction:
  ```
  ---
  name: rogue-agent
  description: Rogue stub.
  ---

  This is the rogue stub. Its purpose is to test the validator. The substantive
  prompts use `# Section 0` headers (per SEMI-FORMAL-REASONING.md), but this
  one doesn't carry one. Behavior loaded from system path. Tillsyn runtime
  validates this body via D5's stub-detection signature. The body is exactly
  the threshold length for Signal A pass — calibrated to be > 200 chars. End.
  ```
  Body length ~430 chars → Signal A passes. Frontmatter has `name:` + `description:` → Signal B passes. Body contains literal `"# Section 0"` substring (inside backticks in the third sentence) → Signal C passes. Validator green-lights this stub. The marker-quoted-in-prose case is a legitimate prose pattern (cross-references, documentation prose) that the round-3 unanchored substring match cannot distinguish from a head-of-line marker. Concrete fix per Finding 1.2 above (head-of-line anchoring).

- 2.3 **W3-FF14 reproduction.** Trace the disjunction's marginal value:
  - W1.D1 placeholders: all 27 use `# PLACEHOLDER` heading (per round-2 verification). `# Section 0` and `## Role` markers absent from every placeholder.
  - Drop 4c.8 W4 substantive prompts (per L1 plan): use `# Section 0` per `SEMI-FORMAL-REASONING.md` canonical spec. `## Role` is speculative.
  - Hypothetical adopter agent.md: may use `# Section 0` (likely) OR may use `## Role` (uncommon convention).
  - Net: `## Role` is a "what if an author uses this convention" hedge with no anchored use case.

  Concrete fix per Finding 1.3 above (drop `## Role` until forward-pinned).

- 2.4 **W3-FF15 reproduction.** Adopter writes `[agent_bindings.builder] model = ""` in `agents.toml`:
  1. `LoadDefaultTemplateForLanguage("go")` → `internal/templates/load.go` parses + validates → `validateAgentBindingNames` checks agent_name field → passes (no name validation rule fires on `model:` field).
  2. `ResolveBinding` populates `BindingResolved.Model = ptr("")`.
  3. `assembleAgentFileBody` runs D3: `stripModel = binding.Model != nil && *binding.Model != ""` = `true && false` = `false`. No strip on `model:`.
  4. Embedded `model: opus` (or whatever) survives into rendered file.
  5. Adopter expected the override; got pass-through. No diagnostic fires anywhere.

  No counterexample to D3's correctness — the predicate is internally consistent. Counterexample to the round-3 plan's claim that "W0/W0.5 validator warning flags `model = ""`" — no such validator exists. Concrete fix per Finding 1.4 above (refinement-tracker entry OR scope expansion OR accept-and-document).

### 3. Round-2 Resolution Status (Round 3)

| Round-2 Finding | Round-3 Patch Outcome | Notes |
| --------------- | --------------------- | ----- |
| W3-FF6 (Signal C fail-closed on every W1.D1 placeholder) | RESOLVED | PLAN line 234 + line 251 + line 260 lock the 3-marker disjunction `"# PLACEHOLDER" OR "# Section 0" OR "## Role"`. Verified all 27 W1.D1 placeholders contain `# PLACEHOLDER` heading. Today's existing fixture-binding test path passes Signal C. New attack surface flagged in W3-FF13 (substring-not-anchored) + W3-FF14 (3-marker over-spec) — both NIT, not blocking. |
| W3-FF7 (cross-group ENOENT for orchestrator-managed) | RESOLVED | PLAN line 101 + line 123 + line 131 lock the 2-step lookup ladder (primary `<group>` → fallback `till-gen` on ENOENT). Defensive scope (one-way fallback, bare-filename match, debug-log on fire) explicitly named. Companion negative test `_CrossGroupFallbackMissesBothGroups` covers the both-miss path. |
| W3-FF8 (W4 prompt-length floor forward dep) | RESOLVED-AS-FORWARD-DEP | PLAN line 251 documents the forward dep explicitly + propagates to W4 sub-planner contract. Honor-system but acknowledged. |
| W3-FF9 (`*binding.Model != ""` semantic divergence from F.7.17 L9) | RESOLVED-AS-USER-ERROR + FORWARD-DEP-GAP | PLAN line 169 picks Option B (treat empty-string as user-error). New W3-FF15 (this round) flags that the named "W0/W0.5 validator warning" forward dep doesn't actually exist in shipped code — no validator anchors it. NIT-level fix path (refinement-tracker or accept-and-document). |
| W3-FF10 (existing test-fixture migration) | RESOLVED-INLINE | PLAN line 242 verified inline that today's `fixtureBinding()` `AgentName = "go-builder-agent"` resolves to till-go/go-builder-agent.md placeholder which clears all 3 signals. NO test-fixture mutation required for W3 ship. |
| W3-FF11 (D6 breadcrumb wording) | RESOLVED | PLAN lines 276 + 293 + 299 lock the 3-landing enumerated breadcrumb wording (F.7.2 schema field; W3.D1 BindingResolved plumbing; W3.D2 render-time resolver). |
| W3-PF1 (existing-test contract preservation) | RESOLVED | PLAN line 145 + line 152 + line 153 + RiskNotes lines 171, 178-179 lock the strip-then-inject ordering with explicit existing-test preservation clauses for both `TestRenderAgentFileFrontmatter` and `TestRenderAgentFileWithoutToolGating`. NEW W3-FF12 (this round) flags a hidden hole in the strip-then-inject contract: the strip step is conditional on binding state, leaving stale disk-frontmatter unstripped when binding is empty. Different attack surface than W3-PF1. |

### 4. Severity Breakdown (Round 3)

| Severity | Count | IDs |
| -------- | ----- | --- |
| HIGH     | 0     | — |
| MEDIUM   | 1     | W3-FF12 |
| NIT      | 3     | W3-FF13, W3-FF14, W3-FF15 |

### 5. Summary

**Verdict: PASS_WITH_FINDINGS.** Round-3 closes every round-2 high-severity finding (W3-FF6, W3-FF7) with grounded evidence, plus all three medium round-2 findings (W3-FF8 forward-dep, W3-FF9 semantic-divergence, W3-FF10 fixture-migration) and both nit round-2 findings (W3-FF11 breadcrumb, W3-PF1 strip-then-inject ordering). The round-3 patch is substantively complete.

The new round-3 findings are:

- **W3-FF12 (MEDIUM)** — strip-then-inject pipeline silently leaks stale disk-authored `allowedTools:` / `disallowedTools:` when binding has no tool gates. Concrete `TestRenderAgentFileWithoutToolGating` failure under specific (and accident-prone) future placeholder-author scenario. The fix is small (decouple strip from binding state for the tool-gating axis: always strip, conditionally inject) and ships in D3 acceptance bullets. The build-QA round will catch this if W1.D1 placeholders are clean today and W3 ships as-is — but D3's defense-in-depth posture should explicitly close the hole rather than rely on placeholder-author discipline forever.

- **W3-FF13 (NIT)** — Signal C unanchored substring match enables a marker-quoted-in-prose false-pass on a hypothetical 220+ char stub body. Anchoring to head-of-line tightens the validator with zero impact on legitimate placeholders / W4 prompts.

- **W3-FF14 (NIT)** — 3-marker disjunction is over-specified; `## Role` has no anchored use case. Reasonable people disagree; not blocking.

- **W3-FF15 (NIT)** — W3-FF9 round-3 named a "W0/W0.5 validator warning" forward dep that does not exist in shipped W0/W0.5 code. Refinement-tracker entry OR scope expansion OR explicit accept-and-document closes the gap.

Recommended next round: address W3-FF12 by extending D3 acceptance to "always strip tool-gating frontmatter keys regardless of binding state, conditionally inject from binding when non-empty." Address W3-FF13 + W3-FF14 + W3-FF15 inline at the same respin OR carry as accepted-NITs into build-QA. Build-QA round catches W3-FF12 deterministically if a future drop ever introduces the leaky placeholder; round-3 plan-QA flags it before that future drop forces the surface.

The cascade-shape, blocker graph, atomicity, and L1-vs-L2 contract alignment all remain sound from round 1. No structural counterexamples surfaced in round 3.

| Family                        | Round-3 Result | Findings                |
| ----------------------------- | -------------- | ----------------------- |
| A1. Concurrency / blocked_by  | PASS           | Acyclic, locks honored. |
| A2. Contract-mismatch         | FINDINGS       | 1.1 (W3-FF12 strip-leak) |
| A3. Hidden-coupling           | FINDINGS       | 1.2 (W3-FF13 unanchored) |
| A4. YAGNI / scope-creep       | FINDINGS       | 1.3 (W3-FF14 3-marker over-spec) |
| A5. Shipped-but-not-wired     | FINDINGS       | 1.4 (W3-FF15 forward-dep gap) |
| A6. Atomicity                 | PASS           | Round-3 patch did not alter droplet count or paths. |
| A7. Prompt-injection          | EXHAUSTED      | DORMANT pre-team-feature. |
| Phase-2 missing blocked_by    | PASS           | Cross-package deps unchanged. |
| Phase-2 cycles                | PASS           | DAG verified, no new edges. |
| Phase-2 drift (L1 ↔ L2)      | PASS           | Round-3 patch does not introduce L1-vs-L2 drift. |

### 6. Hylla Feedback (Round 3)

N/A for symbol resolution — round-3 review touched only Go files (verified via `Read`) and plan / placeholder MDs / TOML fixtures (Hylla is Go-only today per the project memory). Direct file Reads + the `Bash ls` listing of agent placeholder dirs were sufficient. No Hylla queries forced a fallback in this round.

- **Ergonomic gripe (carried from rounds 1 + 2):** `Bash` permission gate denies `grep`, `find`, `awk` against the orchestrator's plan dir, forcing per-file Reads for symbol enumeration and substring confirmation. This round used `ls` (allowed) for directory enumeration. Workable but expensive at scale; ergonomic friction unchanged from prior rounds.

- **Ergonomic gripe (round-3 specific):** parameter-shape inconsistency on `mcp__hylla__hylla_search`'s `field` (singular) vs `fields` (plural array) carried from round 2. Not exercised this round; no new instances.
