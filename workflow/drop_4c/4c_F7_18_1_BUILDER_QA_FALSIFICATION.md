# Drop 4c F.7.18.1 — Schema-2 Context Sub-Struct: Builder QA Falsification

**Droplet:** `4c.F.7.18.1` — closed `Context ContextRules` sub-struct on `templates.AgentBinding` plus `validateAgentBindingContext` Load-time validator.

**Reviewer mode:** read-only adversarial QA falsification. Counterexample-search pass over 13 attack vectors (A1–A13) plus self-attack from §4.4 sweep.

**Verdict legend:** CONFIRMED (counterexample produced — builder must rework) | REFUTED (attack landed but no real defect) | NIT (cosmetic / non-blocking observation).

---

## Round 1

### A1. `max_rule_duration = "0s"` adopter-surprise footgun?

**Claim attacked:** Builder picked "zero legal, negative fails" interpretation. Construct cases where adopter writes `"0s"` deliberately, expecting "no timeout / infinite", but consumer treats zero as "use 500ms default."

**Evidence:**
- `internal/templates/schema.go:466-472` `MaxRuleDuration` doc-comment: *"Engine-time default = 500ms when zero (F.7.18.4 wires context.WithTimeout per rule)."*
- `internal/templates/schema.go:309-310` `MaxBudgetUSD` doc-comment + `Validate()` rule: *"`MaxBudgetUSD: >= 0 (zero permitted; means unlimited at dispatcher's choice)."* (line 497).
- `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md:96` plan body: *"If `MaxRuleDuration` is set (`> 0`) it MUST be positive; zero means 'use bundle-global default' (engine-time substitution)."*

**Trace:**
- The schema convention in this repo is INCONSISTENT across the AgentBinding fields:
  - `MaxBudgetUSD == 0` → "unlimited" (per existing Validate doc).
  - `BlockedRetryCooldown == 0` → "no cooldown" (zero is a valid duration meaning "fire next retry immediately").
  - `MaxRuleDuration == 0` → "use 500ms default" (per F.7.18.1 doc-comment).
  - `MaxChars == 0` → "use 50000 default" (per F.7.18.1 doc-comment).
- An adopter familiar with `MaxBudgetUSD` semantics writing `max_rule_duration = "0s"` thinking "no timeout / unlimited per-rule budget" would in fact get a 500ms cap.
- Conversely, `BlockedRetryCooldown = "0s"` means "zero wait" (literal), not "use default cooldown".
- This is a documented design choice, not a defect. The `MaxRuleDuration` and `MaxChars` doc-comments DO state the engine-time default-substitution explicitly. The schema layer does not, and cannot, prevent adopter misreading.

**Conclusion:** **NIT.** The footgun is real but documented. Builder's interpretation matches the F.7.18 plan body L96 + the empty-`[context]` happy-path test (which requires `MaxRuleDuration == 0` to load cleanly). A more defensive design would use a sentinel like `-1` for "no timeout" and treat `0` as "use default" — but that conflicts with the validator's "negative rejected" rule. Sentinel via positive constant (`MaxDurationUnlimited = math.MaxInt64`?) is also possible but out-of-scope for F.7.18.1.

**Routing:** Document the zero-vs-nil convention asymmetry in F.7.18.4's engine-time substitution doc-comment so the adopter-surprise risk is captured at exactly one place. If the dev wants stronger schema-layer guidance, surface as a refinement against F.7.18.4.

---

### A2. `descendants_by_kind` on `kind=plan` allow-test exercises template loading or cargo-cult passes?

**Claim attacked:** Test claims the allow-rule fires; verify it actually exercises the loader.

**Evidence:** `internal/templates/context_rules_test.go:287-317`, specifically:

```
[agent_bindings.plan]
agent_name = "go-planning-agent"
model = "opus"

[agent_bindings.plan.context]
parent = true
descendants_by_kind = ["build", "plan"]
delivery = "file"
```

The test calls `Load(strings.NewReader(src))`, asserts `err == nil`, and then verifies `binding.Context.DescendantsByKind == ["build", "plan"]` element-by-element.

**Trace:**
- The TOML payload routes through the full validator chain (the same `Load` function called by every other test).
- The validator chain runs `validateAgentBindingContext` (load.go:129), which in turn calls `validateContextKindList(kind, "descendants_by_kind", ctx.DescendantsByKind)` (load.go:491-493). The check on `descendants_by_kind` is identical to the check on `siblings_by_kind` and `ancestors_by_kind` — there is no `kind=plan`-specific gate.
- Therefore: there is no "schema rule against `descendants_by_kind` on `kind=plan`" to gate against — this is the absence-of-rule asserted by the test, not the existence-of-rule. The test correctly verifies the absence.

**Conclusion:** **REFUTED.** Test exercises real loader code via the standard `Load` entry point and asserts both no-error and verbatim-decode. Future tightening of the validator (e.g. "descendants forbidden on plan") would fail this test, which is the regression bar's purpose.

---

### A3. `Delivery` enum closure: does validator case-fold `"Inline"` / `"INLINE"`?

**Claim attacked:** Verify case sensitivity at validator layer.

**Evidence:** `internal/templates/load.go:498-509`:

```go
func isValidContextDelivery(v string) bool {
	for _, candidate := range validContextDeliveryValues {
		if v == candidate {
			return true
		}
	}
	return false
}
```

`v == candidate` is exact string equality. No `strings.ToLower` / `TrimSpace`.

**Trace:**
- `delivery = "Inline"` (capitalized) → `v == "inline"` is false; `v == "file"` is false; `v == ""` is false → returns false → ErrInvalidContextRules.
- `delivery = "INLINE"` (uppercase) → same. Rejected.
- `delivery = "file "` (trailing space) → not in closed set. Rejected.
- The doc-comment on `isValidContextDelivery` (load.go:498-501) explicitly cites the rationale (*"Exact-match — no whitespace trimming or case folding, mirroring the IsValidGateKind rationale (silent case-fold matching would mask 'Inline' / 'FILE' typos at load time)"*).
- However, the test only exercises `"stream"` (`internal/templates/context_rules_test.go:165`). Capitalized variants like `"Inline"` are NOT covered by the test suite.

**Conclusion:** **NIT (test coverage).** The implementation correctly rejects case-mismatched delivery values. The failing-input test pool is narrow — `"stream"` is the only case. A future contributor who decides to "improve UX" by adding `strings.ToLower(v)` to the comparison would find no test guarding against it. Compare to `TestGateKindClosedEnum` (`schema_test.go:132-166`) which has 7 invalid cases including `"MAGE_CI"` and `" mage_ci "` — that's the regression pattern this validator should mirror.

**Routing:** Add a follow-on test in F.7.18.1 round-2 OR a refinement-drop note: extend `TestLoadAgentBindingContextRejectsInvalidDelivery` into a table-driven test covering `"Inline"`, `"FILE"`, `"file "`, ` "file"` (leading space), `"in-line"` (hyphen vs underscore confusion). The mirror of the gate-kind test is the canonical pattern.

---

### A4. `MaxChars = 0` adopter-surprise footgun (same shape as A1)?

**Claim attacked:** `max_chars = 0` legal at schema, but consumer treats zero as "use 50KB default". Adopter setting `0` thinking "unlimited" gets unexpected 50KB cap.

**Evidence:** Same as A1 — `MaxChars` doc-comment line 459-460: *"Engine-time default = 50000 when zero (F.7.18.3 territory). Negative values are rejected at load time."*

**Trace:** Identical to A1. The zero-vs-nil convention is inconsistent across the repo. The consumer-side adopter-surprise is real, but the design choice is documented.

**Conclusion:** **NIT.** Same disposition as A1 — documented, consistent with sibling field semantics within `ContextRules`, but conflicts with the broader `MaxBudgetUSD = 0 → unlimited` convention. Route to the same refinement.

---

### A5. Cross-binding kind reference uses `domain.IsValidKind`?

**Claim attacked:** If validator uses string equality vs hardcoded list, future kind additions break it.

**Evidence:** `internal/templates/load.go:511-523` `validateContextKindList`:

```go
func validateContextKindList(kind domain.Kind, fieldName string, kinds []domain.Kind) error {
	for _, k := range kinds {
		if !domain.IsValidKind(k) {
			return fmt.Errorf("%w: agent_bindings[%q].context.%s entry %q",
				ErrUnknownKindReference, kind, fieldName, k)
		}
	}
	return nil
}
```

Direct call to `domain.IsValidKind` (`internal/domain/kind.go:50-52`) which uses `slices.Contains(validKinds, ...)` against the canonical 12-value enum.

**Trace:** Future kind additions to the closed enum land in `internal/domain/kind.go:34-47` (the `validKinds` slice); `validateContextKindList` automatically picks them up via the shared `IsValidKind` helper. No hardcoded list duplication.

**Conclusion:** **REFUTED.** Validator delegates correctly.

---

### A6. `[agent_bindings.<kind>.context]` strict-decode unknown-key rejection actually fires?

**Claim attacked:** Verify the strict-decode chain catches unknown sub-table keys.

**Evidence:** `internal/templates/context_rules_test.go:319-347` `TestLoadAgentBindingContextStrictDecodeRejectsUnknownKey`:

```go
[agent_bindings.build.context]
bogus_field = true
```

Asserts: `errors.Is(err, ErrUnknownTemplateKey)` AND `strings.Contains(err.Error(), "bogus_field")`.

**Trace:**
- `internal/templates/load.go:99-106` `strictDecoder.DisallowUnknownFields()` is invoked unconditionally on the full template.
- pelletier/go-toml/v2 walks every level of nested tables; an unknown key under `[agent_bindings.build.context]` decodes into `ContextRules` and trips `StrictMissingError` because every `ContextRules` field has an explicit TOML tag.
- The test's `errors.Is(err, ErrUnknownTemplateKey)` assertion confirms the wrap chain.

**Conclusion:** **REFUTED.** Strict decode demonstrably fires for the new sub-struct.

---

### A7. `fullyPopulatedAgentBinding` extension via `omitempty` instead of populating Context?

**Claim attacked:** Builder noted pelletier/go-toml/v2 emits `[context]` table even for zero. Could `toml:"context,omitempty"` cleanly prevent emission?

**Evidence:**
- `internal/templates/schema.go:389` field tag: `Context ContextRules \`toml:"context"\`` — no `omitempty`.
- `internal/templates/agent_binding_test.go:32-58` populates Context fully.

**Trace:**
- pelletier/go-toml/v2 `omitempty` on a struct field: the encoder OMITS the table when ALL fields are at their zero value.
- However, the Schema-2 design INTENDS `[context]` to be optional at the TOML layer (omitted = fully-agentic mode per master PLAN L13). `omitempty` would change the round-trip semantics: a fully-populated Context decoded then re-encoded would still emit; a zero-valued Context would NOT emit. This is actually CORRECT for the use case.
- BUT: the round-trip test (`TestAgentBindingTOMLRoundTrip`) populates Context fully precisely to exercise every TOML tag. With `omitempty`, the round-trip on a zero-value Context would produce slightly different bytes (no `[context]` block) — the existing happy-path test (`TestLoadAgentBindingContextEmptyTablePresent`) already proves that an empty `[context]` block is equivalent to absence.
- The current design (no `omitempty`) is also valid — pelletier emits the empty table, which decodes back identically. Neither approach produces incorrect behavior; the choice is pure stylistic preference for default-marshaled output.

**Conclusion:** **NIT.** `omitempty` would marginally improve marshaled-output ergonomics (no empty `[context]` table when Context is zero) but is not required for correctness. Either approach satisfies the F.7.18.1 acceptance contract. Route as cosmetic-only refinement if the dev wants tidier emit.

---

### A8. `Duration` type cross-leak — package's `Duration` or `time.Duration`?

**Claim attacked:** Verify which Duration type is used.

**Evidence:** `internal/templates/schema.go:472`: `MaxRuleDuration Duration \`toml:"max_rule_duration"\``.

**Trace:**
- `Duration` (line 33) is the package's own type alias `type Duration time.Duration` with `MarshalText` / `UnmarshalText` methods.
- This is the correct choice: pelletier/go-toml/v2 cannot decode a TOML duration string into a bare `time.Duration` (the latter has no `TextUnmarshaler`). Using the package's `Duration` enables the wire form `max_rule_duration = "500ms"`.
- `BlockedRetryCooldown` (line 331) uses the same `Duration` type — symmetric pattern.
- Existing `TestAgentBindingDurationStringWireForm` proves the wrapper round-trips canonically.

**Conclusion:** **REFUTED.** Correct type.

---

### A9. Validator chain ordering: env-fail short-circuits context check?

**Claim attacked:** If `validateAgentBindingEnvNames` fails fast on a malformed env entry, `validateAgentBindingContext` is unchecked.

**Evidence:** `internal/templates/load.go:126-131`:

```go
if err := validateAgentBindingEnvNames(tpl); err != nil {
    return Template{}, err
}
if err := validateAgentBindingContext(tpl); err != nil {
    return Template{}, err
}
```

Each validator returns on the first offending entry; chain returns on the first non-nil error.

**Trace:**
- A template with BOTH a malformed env entry AND a malformed context block would surface the env error first, masking the context error until the env is fixed.
- This is the standard "fail-fast first error" pattern matching every other validator in the chain (validateMapKeys → validateChildRuleKinds → validateChildRuleCycles → validateChildRuleReachability → validateGateKinds → validateAgentBindingEnvNames → validateAgentBindingContext).
- Each validator iterates the same `tpl.AgentBindings` map. There is no shared state mutation; if env passes, context runs against the same data.
- Error aggregation (returning all errors at once instead of fail-fast) is explicitly NOT a F.7.18.1 acceptance criterion. The plan body L100 specifies single-error-return: *"Future drops that want exhaustive reporting can switch to error aggregation."*

**Conclusion:** **REFUTED.** Standard fail-fast chain. Adopters fix one error, retry, find next. No defect.

---

### A10. `MaxRuleDuration = "-0s"` (negative zero) handling?

**Claim attacked:** Verify negative-zero parsing.

**Evidence:** Validator check is `time.Duration(ctx.MaxRuleDuration) < 0` (load.go:481).

**Trace:**
- `time.ParseDuration("-0s")` returns `time.Duration(0)` with `nil` error. (Verified by Go semantic: `time.Duration` is an `int64` count of nanoseconds; `-0` and `0` are the same int64 value. Negative-zero floating-point semantics do not apply to integer types.)
- Therefore `"-0s"` decodes to `Duration(0)`, which `time.Duration(...)<0` evaluates as false → passes validation → engine substitutes 500ms default at runtime.
- This is consistent with `"0s"` semantics (A1 attack). Adopter writing `"-0s"` thinking "negative" gets the same default-substitution as `"0s"`, which is the expected behavior — the validator does not surface a confusing reject for negative-zero.

**Conclusion:** **REFUTED.** No surprise. `"-0s"` parses to zero, treated as zero. Builder's interpretation is correct.

---

### A11. `Validate()` method on AgentBinding does NOT validate Context?

**Claim attacked:** Programmatic in-memory AgentBinding construction calling `Validate()` skips Context-level checks.

**Evidence:** `internal/templates/schema.go:509-532` `(b AgentBinding) Validate()` body:

```go
if strings.TrimSpace(b.AgentName) == "" { ... }
if strings.TrimSpace(b.Model) == "" { ... }
if b.MaxTries <= 0 { ... }
if b.MaxTurns <= 0 { ... }
if b.MaxBudgetUSD < 0 { ... }
if b.BlockedRetries < 0 { ... }
if time.Duration(b.BlockedRetryCooldown) < 0 { ... }
return nil
```

NO call to `b.Context.Validate()` or any context-validation helper.

**Trace:**
- Context-level validation lives ONLY in `validateAgentBindingContext` (load.go:470), which is called from `Load` and iterates `tpl.AgentBindings`.
- A caller that constructs an `AgentBinding` programmatically (e.g. test fixtures, dispatcher in-memory mocks) and calls `b.Validate()` will get nil for invalid Context fields:
  - `binding.Context.Delivery = "stream"` → `b.Validate()` returns nil.
  - `binding.Context.MaxChars = -1` → `b.Validate()` returns nil.
  - `binding.Context.SiblingsByKind = []domain.Kind{"bogus"}` → `b.Validate()` returns nil.
- The dispatcher's expected consumer path is `templates.Load → tpl.AgentBindings[kind]`, where the load chain has already validated. Programmatic construction is a test-fixture / mock concern.
- The asymmetry is consistent with how Env validation works: `validateAgentBindingEnvNames` lives only in the load chain, not on `Validate()`.

**Conclusion:** **NIT (asymmetric API surface).** The current design treats `Validate()` as a fast struct-level check (used by the table-driven tests in `agent_binding_test.go:88-210`) and `validateAgentBindingContext` / `validateAgentBindingEnvNames` as load-chain validators. A programmer who reads `Validate()` and assumes "this is the complete validation surface for AgentBinding" would be wrong.

**Routing:** Either (a) document on `Validate()` doc-comment that Context + Env validation lives in the load chain only, or (b) extend `Validate()` to call into context-validation. Option (a) is the lower-cost change and matches the existing Env precedent. Surface as a doc-comment refinement against F.7.18.1.

---

### A12. Empty `siblings_by_kind = []` semantics — slice-vs-omitted equivalence?

**Claim attacked:** Verify empty slice and omitted field both produce zero-value behavior at consumer.

**Evidence:** `internal/templates/context_rules_test.go:120-148` `assertContextRulesZero` helper:

```go
if len(ctx.SiblingsByKind) != 0 {
    t.Fatalf("Context.SiblingsByKind = %v; want empty", ctx.SiblingsByKind)
}
```

Uses `len(...) != 0` rather than `== nil` — treats nil and empty slice as equivalent.

**Trace:**
- pelletier/go-toml/v2 decoding behavior:
  - `siblings_by_kind = []` (declared empty) → field decodes as a non-nil empty `[]domain.Kind{}`.
  - field omitted → field stays nil (`var SiblingsByKind []domain.Kind` zero value).
- The validator (`validateContextKindList`) iterates `for _, k := range kinds` — both nil and empty-slice produce zero iterations, no error.
- Consumer code (F.7.18.3 aggregator) presumably also iterates with `for _, k := range ctx.SiblingsByKind` — same equivalence.
- Round-trip: a fully-populated AgentBinding with `SiblingsByKind = nil` would encode WITHOUT `siblings_by_kind = []` (pelletier/go-toml/v2 default behavior for nil slices); with `SiblingsByKind = []domain.Kind{}` would encode WITH the empty-array literal. This is a marshaled-output asymmetry, NOT a consumer-behavior difference.

**Conclusion:** **REFUTED.** No defect. Consumer treats both as zero. Marshaled-output asymmetry is unavoidable with pelletier/go-toml/v2 (and irrelevant to F.7.18.1's contract).

---

### A13. Memory-rule conflicts.

**Evidence:**
- `feedback_no_migration_logic_pre_mvp.md` — schema struct addition is in-memory only; no SQL migration; no `till migrate` CLI; clean.
- `feedback_subagents_short_contexts.md` — single-surface task, ~5 file edits, well within scope.
- `feedback_opus_builders_pre_mvp.md` — F.7.18.1 used opus per spawn (worklog confirms).
- `feedback_no_closeout_md_pre_dogfood.md` — no CLOSEOUT.md / LEDGER.md edits in this droplet (worklog confirms — only the worklog itself).
- `feedback_never_remove_workflow_files.md` — no workflow file removal.

**Conclusion:** **REFUTED.** No memory-rule conflicts.

---

## Self-attack — §4.4 sweep extensions

### A-α. `domain.IsValidKind` case-folds — kind-list normalization gap?

**Claim:** `domain.IsValidKind` lowercases before checking enum membership (`internal/domain/kind.go:50-52`). So `siblings_by_kind = ["BUILD"]` passes validation. But the validated kind value lands in `Context.SiblingsByKind` as the original literal `Kind("BUILD")` (uppercase) — NOT normalized to `Kind("build")`.

**Evidence:**
- `internal/domain/kind.go:50-52`:
  ```go
  func IsValidKind(kind Kind) bool {
      return slices.Contains(validKinds, Kind(strings.TrimSpace(strings.ToLower(string(kind)))))
  }
  ```
  Returns true for any case variant.
- `internal/templates/load.go:515-522` `validateContextKindList` checks membership but does NOT mutate the slice elements.
- Therefore: `binding.Context.SiblingsByKind[0] == Kind("BUILD")` literally. A downstream comparison `if k == domain.KindBuild { ... }` (where `KindBuild == "build"` lowercase) would be FALSE.

**Trace:**
- The F.7.18.3 aggregator engine (not yet built) will iterate `ctx.SiblingsByKind` and call something like `repo.GetSiblings(parentID, kind)`. If the SQLite layer compares case-sensitively (which the existing kind storage does — schema.go uses lowercase canonical kinds), the lookup returns zero rows.
- The dispatcher's existing `validateMapKeys` path has the same gap: `[agent_bindings.BUILD]` would pass `domain.IsValidKind` but the resulting `tpl.AgentBindings` map would have a key `Kind("BUILD")` that doesn't match `tpl.AgentBindings[domain.KindBuild]` lookups elsewhere.
- This is a PRE-EXISTING repo-wide gap, not a F.7.18.1 defect. F.7.18.1 inherits the convention by delegating to `domain.IsValidKind`.

**Conclusion:** **NIT (pre-existing).** F.7.18.1 inherits a repo-wide canonicalization gap rather than introducing one. The right fix is to change `domain.IsValidKind` (or add a `domain.NormalizeKind` helper) and apply it across all validators that store kinds; that's a refinement targeting `internal/domain/kind.go`, not F.7.18.1.

**Routing:** Surface as cross-cutting refinement against `internal/domain/kind.go`. Track separately so a future kind-canonicalization drop captures all 4+ touch points (`validateMapKeys`, `validateChildRuleKinds`, `validateContextKindList`, every test fixture using uppercase kinds).

---

### A-β. Round-trip TOML test — does pelletier decode-into uppercase Kind?

**Claim:** `TestAgentBindingTOMLRoundTrip` builds a fully-populated AgentBinding with `SiblingsByKind: []domain.Kind{domain.KindBuildQAProof, domain.KindBuildQAFalsification}` (lowercase canonical). What if pelletier decode happens to lowercase the resulting key, masking A-α?

**Evidence:** `internal/templates/agent_binding_test.go:50-55` populates with the canonical lowercase enum constants. Round-trip asserts `reflect.DeepEqual(original, decoded)`.

**Trace:**
- Encode: `Kind("build-qa-proof")` → TOML literal `"build-qa-proof"`.
- Decode: TOML literal `"build-qa-proof"` → `Kind("build-qa-proof")`.
- Symmetric; `reflect.DeepEqual` passes.
- The round-trip test does NOT exercise uppercase-kind input; A-α gap remains uncovered.

**Conclusion:** **NIT.** Round-trip test is correct for canonical input. A-α gap is not a F.7.18.1 concern.

---

### A-γ. Sentinel chain on Delivery error wrap.

**Claim:** Verify `ErrInvalidContextRules` correctly chains to `ErrInvalidAgentBinding`.

**Evidence:** `internal/templates/load.go:212`:
```go
ErrInvalidContextRules = fmt.Errorf("%w: context", ErrInvalidAgentBinding)
```
Test (`context_rules_test.go:174-178`):
```go
if !errors.Is(err, ErrInvalidAgentBinding) {
    t.Fatalf("Load: errors.Is(_, ErrInvalidAgentBinding) = false; err = %v", err)
}
```

**Conclusion:** **REFUTED.** Sentinel chain explicit and tested.

---

## Hylla Feedback

`N/A — action item touched only Go code that I read directly via `Read` rather than Hylla; the falsification surface is the new Go file plus four edited files, all known by exact path from the spawn prompt + worklog. No Hylla queries issued, no fallback tracked.`

---

## Summary

**Verdict: PASS-WITH-NITS.**

| Attack | Verdict | Severity |
|---|---|---|
| A1 — `max_rule_duration = "0s"` adopter footgun | NIT | Low — documented |
| A2 — `descendants_by_kind` allow-test exercises loader | REFUTED | — |
| A3 — `Delivery` enum case-fold attack | NIT | Low — test-coverage gap |
| A4 — `MaxChars = 0` adopter footgun | NIT | Low — documented |
| A5 — Cross-binding kind ref uses `domain.IsValidKind` | REFUTED | — |
| A6 — `[context]` strict-decode unknown-key | REFUTED | — |
| A7 — `omitempty` on Context field | NIT | Cosmetic |
| A8 — Duration type cross-leak | REFUTED | — |
| A9 — Validator chain fail-fast | REFUTED | — |
| A10 — `"-0s"` negative-zero handling | REFUTED | — |
| A11 — `Validate()` method skips Context | NIT | Asymmetric API — doc-comment fix |
| A12 — Empty slice vs omitted | REFUTED | — |
| A13 — Memory-rule conflicts | REFUTED | — |
| A-α — `domain.IsValidKind` case-fold gap | NIT (pre-existing) | Cross-cutting |
| A-β — Round-trip uppercase-kind coverage | NIT | Test-coverage gap |
| A-γ — Sentinel chain explicit | REFUTED | — |

**Zero CONFIRMED counterexamples.** Five NITs:

1. **A3 — Delivery case-mismatch test coverage.** Extend `TestLoadAgentBindingContextRejectsInvalidDelivery` to a table-driven test mirroring `TestGateKindClosedEnum`'s 7-row pool (`"Inline"`, `"FILE"`, `"file "`, ` "file"`, `"in-line"`, `"INLINE"`). Low-cost; matches existing pattern.
2. **A1 / A4 — Zero-as-default vs zero-as-disabled adopter-surprise.** Document the convention explicitly when F.7.18.4 lands the engine-time default-substitution. NOT a F.7.18.1 defect; route to F.7.18.4.
3. **A7 — `omitempty` on Context field.** Cosmetic-only; no behavior change. Optional.
4. **A11 — `Validate()` method skips Context.** Add a doc-comment on `Validate()` clarifying that Context + Env validation lives in `templates.Load` only, mirroring the existing Env asymmetry. Low-cost.
5. **A-α — `domain.IsValidKind` case-fold gap.** Pre-existing repo-wide. Surface as cross-cutting refinement against `internal/domain/kind.go`; not a F.7.18.1 concern.

**Recommended action:** Pass the droplet. Land NIT 1 (A3 test extension) as a one-test-file addendum — reviewer's choice whether to require it pre-merge or land as follow-up. NITs 2-5 are routing-only.
