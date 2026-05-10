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
