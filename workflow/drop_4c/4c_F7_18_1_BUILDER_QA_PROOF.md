# Drop 4c F.7.18.1 — Builder QA Proof Review

**Verdict: PROOF GREEN-WITH-NITS.**

The builder shipped exactly the F.7.18.1 surface — `Context ContextRules` sub-struct on `AgentBinding`, eight TOML-tagged fields, closed-enum delivery constants, a Load-time validator wired into the chain at the correct position, and nine acceptance tests that match the spawn-prompt scenario list one-for-one. `mage ci` is green per worklog. Two minor nits are doc-comment-citation drift (no code-behavior impact) and the lack of a builder-side capture of the `mage ci` log; both are flagged below as N1/N2 and do not block.

---

## 1. Per-Criterion Evidence

### 1.1 Criterion 1 — `Context ContextRules` field on `AgentBinding` with TOML tag `context` + doc-comment citing master PLAN L13

PASS. `internal/templates/schema.go:389` — `Context ContextRules \`toml:"context"\``. Doc-comment block at `schema.go:373-388` cites master PLAN L13 (line 379) and the F.7.18 plan acceptance + REV-3. Field placed at the end of the struct after the F.7.17.1 `Env` / `CLIKind` additions, preserving REV-1 ordering.

**N1 (citation drift, non-blocking)** — `schema.go:379` reads `// Per Drop 4c F.7.18 (master PLAN.md L13)` but `main/PLAN.md` line 13 is the table-of-contents heading "The Cascade Model". The actual "FLEXIBLE not REQUIRED" framing lives at line 13 of the **F.7.18 sub-plan** (`workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md:13`), not the master `PLAN.md`. Recommend a future doc-comment refresh to read `(F.7.18 sub-plan L13)` for precision. No code-behavior impact.

### 1.2 Criterion 2 — `ContextRules` struct with eight TOML-tagged fields

PASS. `schema.go:410-473`:
- `Parent bool \`toml:"parent"\`` — line 415.
- `ParentGitDiff bool \`toml:"parent_git_diff"\`` — line 423.
- `SiblingsByKind []domain.Kind \`toml:"siblings_by_kind"\`` — line 429.
- `AncestorsByKind []domain.Kind \`toml:"ancestors_by_kind"\`` — line 435.
- `DescendantsByKind []domain.Kind \`toml:"descendants_by_kind"\`` — line 445.
- `Delivery string \`toml:"delivery"\`` — line 456.
- `MaxChars int \`toml:"max_chars"\`` — line 464.
- `MaxRuleDuration Duration \`toml:"max_rule_duration"\`` — line 472.

Every field carries an explicit TOML tag — closed-struct strict-decode unknown-key rejection (Criterion 9 path) is mechanically guaranteed.

Closed-enum delivery constants at `schema.go:479-488`: `ContextDeliveryInline = "inline"`, `ContextDeliveryFile = "file"`. Note the type is plain `string`, not a named alias type like `ContextDelivery`. The F.7.18.1 plan (line 86–89) suggested a closed-enum string type with `IsValidContextDelivery` helper, but the builder's plain-string approach is functionally equivalent — `validateAgentBindingContext` performs the closed-enum check, and `isValidContextDelivery` exists as the unexported helper at `load.go:502-509`. **N2 (design choice, non-blocking)**: the plan-described named type would have given consumers (F.7.18.3 engine) a slightly stronger compile-time discipline, but the plan body's wording was "MAY mirror the IsValidGateKind pattern" rather than MUST, and the existing `string` field will not block the engine. Do not regress.

### 1.3 Criterion 3 — `validateAgentBindingContext` wired in correct order

PASS. `load.go:129-131` — invoked from `Load()` immediately after `validateAgentBindingEnvNames(tpl)` (lines 126-128) and before the function returns. The validator-order doc-comment at `load.go:55-59` was extended with step 4(g) describing the new validator. The closed declared order `validateMapKeys → validateChildRuleKinds → validateChildRuleCycles → validateChildRuleReachability → validateGateKinds → validateAgentBindingEnvNames → validateAgentBindingContext` matches the plan's "after `validateAgentBindingEnvNames`" placement exactly.

### 1.4 Criterion 4 — Nine acceptance tests pass

PASS. `internal/templates/context_rules_test.go` declares exactly nine top-level `Test*` functions plus the `assertContextRulesZero` helper:

| # | Test | Line | Asserts |
|---|---|---|---|
| 1 | `TestLoadAgentBindingContextHappyPath` | `:17` | bounded-mode block decodes verbatim; `parent=true`, `parent_git_diff=true`, `ancestors_by_kind=["plan"]`, `delivery="file"`, `max_chars=50000`, `max_rule_duration="500ms"` |
| 2 | `TestLoadAgentBindingContextEmptyTablePresent` | `:71` | `[context]` heading with no nested keys → zero value, validator passes |
| 3 | `TestLoadAgentBindingContextOmittedAltogether` | `:97` | no `[context]` table at all → zero value, FLEXIBLE-not-REQUIRED path |
| 4 | `TestLoadAgentBindingContextRejectsInvalidDelivery` | `:156` | `delivery = "stream"` → `ErrInvalidContextRules` AND `errors.Is(_, ErrInvalidAgentBinding)` |
| 5 | `TestLoadAgentBindingContextRejectsNegativeMaxChars` | `:188` | `max_chars = -1` → `ErrInvalidContextRules` |
| 6 | `TestLoadAgentBindingContextRejectsNegativeMaxRuleDuration` | `:224` | `max_rule_duration = "-1s"` → `ErrInvalidContextRules` |
| 7 | `TestLoadAgentBindingContextRejectsInvalidKindReference` | `:253` | `siblings_by_kind = ["bogus_kind"]` → `ErrUnknownKindReference` |
| 8 | `TestLoadAgentBindingContextAllowsDescendantsOnPlanKind` | `:287` | `[agent_bindings.plan.context] descendants_by_kind = ["build", "plan"]` loads clean |
| 9 | `TestLoadAgentBindingContextStrictDecodeRejectsUnknownKey` | `:326` | `bogus_field = true` → `ErrUnknownTemplateKey` |

Sentinel chain assertion in test 4 (`load.go:212` defines `ErrInvalidContextRules = fmt.Errorf("%w: context", ErrInvalidAgentBinding)`) — `errors.Is(_, ErrInvalidAgentBinding)` returns true for context errors. Test 4 lines 171–178 prove both sentinels fire.

The `assertContextRulesZero` helper (`:122-148`) uses field-by-field comparison instead of `==` because `ContextRules` carries slices. The comment correctly notes nil-vs-empty-slice asymmetry from pelletier/go-toml/v2 is treated as equivalent.

### 1.5 Criterion 5 — No schema rule rejecting `descendants_by_kind` on `kind=plan`

PASS. Test 8 at `context_rules_test.go:287-317` is the explicit regression bar. The validator at `load.go:470-496` runs `validateContextKindList` for all three kind-walk slices uniformly with no kind-specific gating — there is no `if kind == domain.KindPlan` branch anywhere in `validateAgentBindingContext`. Cross-checked at `load.go:515-523` (`validateContextKindList`): the helper takes a `kind domain.Kind` argument only for error-UX naming, not for any rule branching.

### 1.6 Criterion 6 — No top-level `Tillsyn` struct (F.7.18.2 territory)

PASS. `rg "type Tillsyn struct" internal/templates/` returned no matches via the worklog's "NOT edited" list (line 46). Confirmed by reading `schema.go` end-to-end (lines 1-533): the `Template` struct (lines 127-199) declares `SchemaVersion / Kinds / ChildRules / AgentBindings / Gates / GateRulesRaw / StewardSeeds` with no `Tillsyn` field. F.7.18.2 will add it.

### 1.7 Criterion 7 — No aggregator engine (F.7.18.3 territory)

PASS. `git status --porcelain` shows no `internal/app/dispatcher/context/` package — the only untracked dispatcher files are `cli_adapter.go` + `cli_adapter_test.go` from F.7.17.2. No `Resolve(ctx, ...)` entry point, no `Bundle` struct, no per-rule renderers. F.7.18.3 will add them.

### 1.8 Criterion 8 — No `[context]` seed edits in `default.toml` (F.7.18.5 territory)

PASS. `git status --porcelain` shows `internal/templates/builtin/default.toml` is NOT in the modified list. F.7.18.5 will add the six seed bindings. The schema-2 + seed split is preserved per the SKETCH:238–243 sequencing rule (no droplet ships seed TOML referencing fields whose schema droplet hasn't landed yet — Schema-2 lands without seeds in this droplet, and F.7.18.5's seeds will be added after F.7.18.2's `[tillsyn]` block exists too).

### 1.9 Criterion 9 — `mage ci` green

PASS-BY-WORKLOG. The builder's worklog (`4c_F7_18_1_BUILDER_WORKLOG.md:58-60`) reports `mage check` (alias of `mage ci`) green, 277 tests in `internal/templates` (268 pre-existing + 9 new), 95.8% coverage on the `internal/templates` package. Coverage exceeds the 70% gate.

**N3 (verification gap, non-blocking)** — the worklog claims `mage ci` green but does not embed the captured log output. The QA-proof role is read-only and cannot independently re-run `mage ci`; verification leans on the worklog's claim + the .githooks/pre-push gate that ran on commit. If the post-build orch wants extra defense-in-depth, a follow-up `mage ci` run before the drop's PR open would tighten this. Not a defect.

### 1.10 Criterion 10 — Scope: only `internal/templates/` files + worklog

PASS. `git status --porcelain` confirms the F.7.18.1 footprint:

Modified (Schema-2 surface):
- `M internal/templates/schema.go` — `ContextRules` struct + delivery constants + Context field on AgentBinding.
- `M internal/templates/load.go` — `validateAgentBindingContext` + `isValidContextDelivery` + `validateContextKindList` + `ErrInvalidContextRules` + `validContextDeliveryValues` + `time` import.
- `M internal/templates/agent_binding_test.go` — `fullyPopulatedAgentBinding` extended with the Context fixture.
- `M internal/templates/schema_test.go` — `TestTemplateTOMLRoundTrip`'s populated `AgentBindings` literal extended with the same Context fixture (lines 88-100 confirm).

New (Schema-2 acceptance + worklog):
- `?? internal/templates/context_rules_test.go` — nine acceptance tests.
- `?? workflow/drop_4c/4c_F7_18_1_BUILDER_WORKLOG.md` — builder worklog.

Other modified / untracked entries (`SKETCH.md`, `cli_adapter.go`, `4c_F7_17_2_BUILDER_WORKLOG.md`, `F7_*_PLAN.md`, `PLAN_QA_*.md`) are F.7.17.2 / planning artifacts unrelated to this droplet's surface — they are not part of F.7.18.1's diff and do not violate scope.

### 1.11 Criterion 11 — Spawn-prompt drift resolution

PASS. The spawn-prompt scenario list said "`max_rule_duration = "0s"` MUST fail" while the validators rule said "if non-zero, must be positive." The builder resolved in favor of the validators rule + the F.7.18 plan body L96 (zero means "use bundle-global default" / engine-time substitution). The resolution is documented:

- **Test doc-comment** at `context_rules_test.go:213-223` explicitly names the contradiction and cites the F.7.18 plan body L96 + the empty-`[context]` happy-path requirement (which itself requires zero-valued `MaxRuleDuration` to load cleanly — Test 2 + Test 3 above).
- **Worklog** at `4c_F7_18_1_BUILDER_WORKLOG.md:38` documents the same drift + resolution.

**Cross-check against F.7.18.4 cap-algorithm consumer (cited by Criterion 11):** the F.7.18 sub-plan at line 302 declares `binding.Context.MaxRuleDuration == 0` → use 500ms (default substitution), and line 96 says "zero means 'use bundle-global default' (engine-time substitution)." The builder's "zero is legal sentinel" reading is therefore the consistent reading across all three artifacts (validators rule + happy-path tests + F.7.18.4 cap-algorithm consumer). Builder's call is sound.

---

## 2. Section 0 Convergence

### 2.1 QA Proof — evidence completeness

Every spawn-prompt criterion has at least one file:line citation; the nine acceptance tests are individually tabulated and cross-referenced against the spawn prompt's scenario list; the four "out-of-scope" criteria (6/7/8) are each verified via direct file reads + `git status` triangulation.

### 2.2 QA Falsification — counterexample attempts

I attempted these counterexamples; none survived:

- **Hidden `if kind == domain.KindPlan` branch?** Read `load.go:470-523` end-to-end; no kind-specific branching.
- **`ContextRules` field missing TOML tag, breaking strict-decode?** Read all eight fields at `schema.go:415-472`; every one carries an explicit tag.
- **Validator ordering sneakily breaks `Env` validation?** Read `load.go:111-131`; `validateAgentBindingEnvNames` runs first, then `validateAgentBindingContext`. No ordering regression.
- **Spawn-prompt-drift resolution leaves `0s` accepted but the F.7.18.4 consumer expects `0s` rejected?** F.7.18 sub-plan lines 96 + 302 + the test 2/3 happy paths all converge on "zero is the default-substitution sentinel." Resolution coherent across the three downstream consumers.
- **Doc-comment citation at `schema.go:379` claims "master PLAN.md L13" but L13 is the TOC heading — does this break a downstream tool?** Pure prose; no code parses doc-comment line cites. Captured as N1 advisory.
- **`reflect.DeepEqual` round-trip would fail on nil-vs-empty-slice asymmetry from pelletier/go-toml/v2?** The `fullyPopulatedAgentBinding` fixture at `agent_binding_test.go:47-56` populates every kind-walk slice with at least one valid kind, sidestepping the asymmetry; the worklog at line 30 calls this out explicitly.

### 2.3 Convergence

(a) QA Falsification produced no unmitigated counterexample. (b) QA Proof confirmed evidence completeness across all 11 criteria. (c) Remaining Unknowns: N1 doc-comment citation precision (advisory only), N3 `mage ci` log capture absent from worklog (advisory only). Both routed via this MD; neither blocks the verdict.

---

## 3. Hylla Feedback

`N/A — droplet review touched non-Go-only artifacts (Go schema/validator + plan MDs); no Hylla queries needed because direct `Read` on the F.7.18.1 surface (5 files in `internal/templates/`) was the bounded fast path. No Hylla miss to log.`

---

## 4. Summary

**Verdict: PROOF GREEN-WITH-NITS.** All 11 spawn-prompt criteria pass with file:line evidence. Two non-blocking advisories (N1 citation drift, N3 mage-ci log absence) and one design-choice note (N2 plain string vs named type). Build is ready for falsification sibling + dev review.
