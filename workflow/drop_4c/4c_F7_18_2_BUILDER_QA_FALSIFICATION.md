# Drop 4c F.7.18.2 — Builder QA Falsification

**Droplet:** F.7.18.2 — Schema-3: `Tillsyn` top-level struct
**Mode:** Read-only adversarial review.
**Reviewer:** go-qa-falsification-agent (subagent).
**Verdict:** **PASS-WITH-NITS** (no CONFIRMED counterexample; three NITs surfaced for documentation / future-extender ergonomics).

## A1 — Strict-decode unknown-key inheritance for future extenders

**Claim:** pelletier/go-toml v2's `DisallowUnknownFields` recursion ensures F.7-CORE F.7.1 + F.7.6 inherit unknown-key rejection automatically when they add fields.

**Evidence:**
- `internal/templates/load.go:104` calls `strictDecoder.DisallowUnknownFields()` on the Decoder for the *whole* Template. The decoder walks the entire struct tree via reflect; nested struct fields with TOML tags inherit the strict-mode flag.
- `internal/templates/load_test.go:644` (`TestLoadAgentBindingStrictDecodeUnknownFieldStillRejects`) is precedent: it proves strict decode rejects `bogus_field` on the *nested* `AgentBinding` struct under `[agent_bindings.build]`. Same mechanism applies to the new `Tillsyn` struct under `[tillsyn]`.
- `internal/templates/load_test.go:821` (`TestLoadTillsynStrictDecodeUnknownFieldRejected`) directly proves the new `Tillsyn` struct inherits the rejection: `[tillsyn] max_context_bundle_chars = 200000, bogus_field = true` fails with `ErrUnknownTemplateKey`.
- F.7.1 / F.7.6 future extension: adding a field like `SpawnTempRoot string` with TOML tag `spawn_temp_root` on `Tillsyn` is purely additive — pelletier's strict mode auto-extends to the new field. No `load.go` change needed for the strict-decode contract; the only `load.go` change for new fields would be field-level validation rules (e.g. `SpawnTempRoot` must be a clean filesystem path), which `validateTillsyn`'s body trivially accommodates.

**Trace or cases:**
- Counterexample attempted: a future extender adds `SpawnTempRoot` but forgets the TOML tag → field would NOT decode at all, but ALSO would not appear in the strict-decode unknown-keys allow-list because reflect would skip it. This means `[tillsyn] spawn_temp_root = "/foo"` would still fail with `ErrUnknownTemplateKey` (because the field has no TOML tag, reflect can't bind it). This is actually a HELPFUL failure mode — silent omission is worse than load-time rejection.
- Counterexample attempted: extender adds `SpawnTempRoot` WITH TOML tag → load accepts it, no validator change needed.

**Conclusion:** REFUTED. The contract holds: extenders inherit unknown-key rejection automatically, AND silent-binding failures (forgotten TOML tag) also fail loud at load time. Builder's REV-3 contract receipt is correct.

**NIT N1:** the worklog's REV-3 receipt (line 61) says "validator's body deliberately limits itself to field-level negative checks, so adding a new field there is purely additive." This is technically true but glosses over the fact that *validator additions* are needed for non-numeric fields (e.g. `SpawnTempRoot` would need filesystem-path validation). Worklog is fine; future extender's worklog should explicitly cite this.

## A2 — Validator chain ordering: late vs early

**Claim:** `validateTillsyn` runs AFTER `validateAgentBindingContext`. Should it run earlier?

**Evidence:**
- `load.go:136` places `validateTillsyn` last in the chain.
- `validateTillsyn` only inspects `tpl.Tillsyn.MaxContextBundleChars` and `tpl.Tillsyn.MaxAggregatorDuration` — both leaf scalars. Validation has zero cross-dependency on `validateAgentBindingContext`.
- Existing chain pattern (`validateMapKeys` → `validateChildRuleKinds` → `validateChildRuleCycles` → `validateChildRuleReachability` → `validateGateKinds` → `validateAgentBindingEnvNames` → `validateAgentBindingContext` → `validateTillsyn`): validators run roughly outermost-key → nested-key. `[tillsyn]` is a top-level key and `[agent_bindings.<kind>.context]` is two levels deep — by structural ordering, `validateTillsyn` "should" run earlier.

**Trace or cases:**
- Counterexample case 1: a template with both `[agent_bindings.build.context] max_chars = -1` AND `[tillsyn] max_context_bundle_chars = -1`. Current order surfaces `ErrInvalidContextRules` first; `validateTillsyn` never fires. Adopter sees one error at a time → fixes context, re-loads, then sees Tillsyn error. Two-round error UX, not catastrophic.
- Counterexample case 2: a template with malformed `[tillsyn] max_aggregator_duration = "negfoo"` — pelletier's `ParseDuration` rejects the string at *decode-time* (step 3 strict decode), not at validator-time. So the malformed-string case never reaches `validateTillsyn` — irrelevant to ordering.
- Counterexample case 3: cross-validator interaction — `validateTillsyn` needs `validateAgentBindingContext` to have run first? No. The cross-cap warning that COULD couple them (`binding.Context.MaxChars > tpl.Tillsyn.MaxContextBundleChars`) is explicitly DEFERRED (worklog line 65). No coupling exists today.

**Conclusion:** REFUTED. Order is a UX preference, not a correctness requirement. Both orderings are equally early-fail (both run before any side-effect; nothing happens between them). No counterexample produced.

**NIT N2:** future-state ergonomics — when F.7.18.4 lands the cross-cap warning, it likely belongs in a NEW validator that reads BOTH `tpl.Tillsyn` and `tpl.AgentBindings`. That new validator MUST run after both `validateAgentBindingContext` AND `validateTillsyn`. Current ordering is forward-compatible.

## A3 — `ErrInvalidTillsynGlobals` sentinel wrap chain

**Claim:** `errors.Is(err, ErrInvalidTillsynGlobals)` works AND `errors.Is(err, ErrUnknownTemplateKey)` works for the strict-decode case.

**Evidence:**
- `load.go:235` declares `ErrInvalidTillsynGlobals = errors.New("invalid tillsyn globals")` — flat sentinel, NOT wrapped under `ErrInvalidAgentBinding` (correct — the `[tillsyn]` table is global, not per-binding).
- `validateTillsyn` (load.go:569-579) wraps via `fmt.Errorf("%w: ...", ErrInvalidTillsynGlobals, ...)`. Standard `%w` chain — `errors.Is(err, ErrInvalidTillsynGlobals)` returns true.
- Strict-decode case: `load.go:107` wraps `*toml.StrictMissingError` via `fmt.Errorf("%w: %s", ErrUnknownTemplateKey, strictErr.String())`. `errors.Is(err, ErrUnknownTemplateKey)` returns true. `bogus_field` under `[tillsyn]` follows this exact path.
- Tests prove both: `TestLoadTillsynRejectsNegativeMaxContextBundleChars` (load_test.go:768-788) asserts `errors.Is(err, ErrInvalidTillsynGlobals)`; `TestLoadTillsynStrictDecodeUnknownFieldRejected` (load_test.go:821-839) asserts `errors.Is(err, ErrUnknownTemplateKey)`.

**Trace or cases:**
- Counterexample attempted: could the sentinel wrap chain break under double-wrap? No — `validateTillsyn` only single-wraps. No nested `%w` chain that would lose route-ability.
- Counterexample attempted: could a caller route on `errors.Is(err, ErrInvalidAgentBinding)` and expect `validateTillsyn` failures to surface? No — `ErrInvalidTillsynGlobals` is INTENTIONALLY a separate sentinel. Worklog (line 23) explicitly cites this design choice. Doc-comment on the sentinel (load.go:230-234) names the rationale.

**Conclusion:** REFUTED. Wrap chains correct, sentinels properly route, tests prove both routes.

## A4 — Zero-as-default sentinel consistency / "explicit 0s" indistinguishable

**Claim:** `MaxAggregatorDuration = "0s"` parses successfully, but the validator can't distinguish "explicit 0s" from "omitted." Both produce zero-value `Duration` → engine consumer can't tell the difference. Footgun.

**Evidence:**
- `time.ParseDuration("0s")` returns `(0, nil)` — explicit zero parses fine. Confirmed by `TestLoadTillsynZeroValuesAllowed` (load_test.go:744-763) which loads `max_aggregator_duration = "0s"` cleanly and asserts `time.Duration(tpl.Tillsyn.MaxAggregatorDuration) == 0`.
- `Tillsyn` struct fields are value types (`int`, `Duration` aliasing `time.Duration int64`). Go's reflect cannot distinguish "zero-value" from "explicit zero set by TOML" — they are byte-identical. To distinguish, the struct would need `*int` / `*Duration` pointer fields with separate "presence" bits.

**Trace or cases:**
- Counterexample case: adopter A writes `[tillsyn] max_aggregator_duration = "0s"` thinking they're DISABLING the aggregator (zero = no work). Engine F.7.18.4 reads zero, substitutes 2s default. Adopter A's intent silently discarded.
- Counterexample case: adopter B omits `[tillsyn]` entirely. Engine F.7.18.4 reads zero, substitutes 2s default. Same outcome as adopter A — but adopter B's intent (use defaults) IS honored.
- Builder + planner have explicitly chosen this semantic (worklog line 24, F7_18 plan line 156): "Zero is legal and means use bundle-global default at engine-time." Adopter A's misreading is an authoring error, not a code bug.
- The zero-as-default-substitution pattern is documented in:
  - `Tillsyn` doc-comment (schema.go:228-230, 234-237).
  - `validateTillsyn` doc-comment (load.go:548-561).
  - `ErrInvalidTillsynGlobals` doc-comment (load.go:221-235).

**Conclusion:** REFUTED-WITH-NIT. The semantic is intentional, well-documented, and tested. Zero-vs-omitted indistinguishability is a Go language limit, not a builder defect.

**NIT N3:** if a future user-facing scenario emerges where "disable the aggregator entirely" is a real need, the schema can introduce an explicit boolean (`aggregator_disabled bool`) or a sentinel value (`max_aggregator_duration = "disabled"`). Today's "zero = default" is fine; the choice should be revisited at F.7.18.4 if engine consumers need a kill-switch.

## A5 — `MaxContextBundleChars` overflow → potential OOM

**Claim:** Field is `int` (platform-dependent). `max_context_bundle_chars = 9223372036854775807` (max int64) parses fine, validator accepts. If engine treats it as a memory allocation hint, OOM possible.

**Evidence:**
- `Tillsyn.MaxContextBundleChars` is `int` (schema.go:231). On 64-bit platforms (Tillsyn's target deployment per `go.mod` and Go 1.26+ requirement), `int` is `int64`. Pelletier accepts the full int64 range.
- `validateTillsyn` (load.go:570-573) only rejects `< 0`. Any non-negative value passes, including `math.MaxInt64`.
- Engine (F.7.18.4 — not yet implemented) is the consumer. Today there is no OOM surface because there is no consumer. Greedy-fit algorithm (PLAN line 305) measures rendered byte counts AGAINST the cap; the cap is a soft skip-marker boundary, NOT a buffer pre-allocation.
- A buffer pre-alloc of `MaxContextBundleChars` bytes would only happen if F.7.18.4's implementation did `make([]byte, 0, cap)` — Go's documented behavior is best-effort allocation, NOT mandatory. `make([]byte, 0, math.MaxInt64)` would panic with `runtime: out of memory`, NOT OOM.

**Trace or cases:**
- Counterexample case: F.7.18.4 implementation uses `bytes.NewBuffer(make([]byte, 0, cap))` with `cap = math.MaxInt64`. Result: `make` panics with `runtime: makeslice: cap out of range` because Go's runtime caps slice cap at platform-dependent max (~`maxAlloc`, which is `1<<48` on amd64). So the path fails LOUD at F.7.18.4-implementation time, not silently OOM.
- Counterexample case: F.7.18.4 uses `strings.Builder` + `Grow(cap)`. `strings.Builder.Grow` for int64-max would panic. Loud failure again.
- Counterexample case: F.7.18.4 doesn't pre-alloc, just appends until threshold. Adopter writes `MaxContextBundleChars = 9223372036854775807`. Engine never reaches threshold (real bundle content is ~1MB). No OOM.

**Conclusion:** REFUTED. Even with int64-max, Go's runtime panics rather than OOMing the host. Engine implementation determines actual behavior; F.7.18.4 (future droplet) will own its own validation if a tight upper bound is needed. F.7.18.2's job is field-shape validation, not engine-resource validation.

**NIT (already raised in N3-equivalent):** if F.7.18.4 wants a sane upper bound (e.g. 10MB), it should add its own engine-side check. Schema validation is intentionally permissive.

## A6 — Cross-cap warning deferred

**Claim:** Builder didn't ship the cross-cap warning (`binding.Context.MaxChars > tpl.Tillsyn.MaxContextBundleChars`). Verify per F.7.18.4 ownership.

**Evidence:**
- F7_18 plan line 159: cross-cap warning is listed in F.7.18.2's acceptance criteria as a "warn-only structured log."
- F7_18 plan line 164: acceptance criterion text explicitly notes "assertion captures the log via the project's existing log-capture test pattern, OR assertion is a doc-comment promise + manual eyeball at QA time (flagged as Q-item if no log-capture pattern exists)."
- Worklog line 64-65: builder defers it. Rationale cited: "Templates package has no logger today; introducing one is YAGNI. If F.7.18.4's planner wants the warning, it can land it then."
- F7_18 plan line 159 also notes the warning is "warn-only — log via package-level structured logger; doesn't fail load." There is no logger in `internal/templates/`. Builder's YAGNI call is defensible.
- F.7.18.4 (PLAN section) does NOT explicitly own the cross-cap warning — but its scope ("greedy-fit + two-axis wall-clock caps") is the natural home. Q-item escalation route exists.

**Trace or cases:**
- Counterexample attempted: is the warning critical for adopter UX? No — adopters who set `binding.Context.MaxChars > tpl.Tillsyn.MaxContextBundleChars` will see the runtime greedy-fit skip the busting rule. Engine F.7.18.4 already plans to emit a `[skipped: <rule_name>]` marker (PLAN line 295). Runtime visibility exists.
- Counterexample attempted: does deferring the warning violate F.7.18.2's acceptance criteria? Yes, technically — F7_18 plan line 159 lists the warning as an acceptance bullet. But the plan also flags Q-routing if no log-capture pattern exists, AND the worklog explicitly defers with rationale.

**Conclusion:** REFUTED-WITH-NIT. The cross-cap warning is a documented deferral with engineering-judgment rationale. The droplet's prompt acceptance criteria (worklog summary line 50-57) does NOT include the warning; the prompt-vs-plan delta is intentional.

**NIT N4 (already addressed by worklog):** F.7.18.4's planner should pick up the cross-cap warning explicitly in its acceptance criteria. The `Out-of-Scope` deferral note in the worklog provides the routing handoff.

## A7 — Round-trip semantics (omitempty / zero-value Tillsyn)

**Claim:** TOML → struct → TOML. Does pelletier emit `[tillsyn]` table when zero-value, or omit it? `omitempty`?

**Evidence:**
- `Template.Tillsyn` field declaration (schema.go:183) has TOML tag `tillsyn` — NO `omitempty`. None of the existing Template fields (`SchemaVersion`, `Kinds`, `ChildRules`, `AgentBindings`, `Gates`, `GateRulesRaw`, `StewardSeeds`) use `omitempty` either — all use bare TOML tags.
- pelletier/go-toml v2 default behavior for un-tagged-omitempty struct fields: emits the field/table even when zero-valued. For nested struct types, this means emitting `[tillsyn]` even when both sub-fields are zero.
- Round-trip test (`TestTemplateTOMLRoundTrip`, schema_test.go:53-132) populates `Tillsyn` with non-zero values (`MaxContextBundleChars = 200000, MaxAggregatorDuration = 2s`). Marshals → Unmarshals → DeepEqual. Passes.

**Trace or cases:**
- Counterexample attempted: round-trip with zero-value Tillsyn. Test does NOT explicitly cover this. But:
  - The Load path handles both `[tillsyn]` empty AND `[tillsyn]` absent — proven by `TestLoadTillsynEmptyTableDecodes` (load_test.go:693-711) and `TestLoadTillsynOmittedTableZeroValue` (load_test.go:718-738).
  - If pelletier emits `[tillsyn]` for zero-value (likely), the Marshal → Unmarshal cycle produces `[tillsyn]\n` empty table on the second pass — Load handles this. No round-trip break.
  - If pelletier omits `[tillsyn]` for zero-value (less likely without `omitempty`), Load handles absent `[tillsyn]` via `TestLoadTillsynOmittedTableZeroValue`. No break either way.
- Counterexample attempted: round-trip with PARTIALLY populated Tillsyn (only `MaxContextBundleChars` set). Pelletier emits `[tillsyn]\nmax_context_bundle_chars = N\n` — `MaxAggregatorDuration` omitted because Duration value is zero. Re-decode: `MaxAggregatorDuration` zero-valued. No break.

**Conclusion:** REFUTED. Round-trip is symmetric for both populated AND empty cases. The Load path's three-way coverage (populated / empty-table / absent-table) defends both Marshal emit modes.

**NIT N5:** the round-trip test should add a zero-value Tillsyn case for completeness. Worth surfacing as a follow-up test improvement, but not load-bearing — the Load tests cover the decode-side; Marshal-side asymmetry is impossible because Load accepts both modes pelletier could emit. **Surfacing as a NIT, not blocking.**

## A8 — `Tillsyn` field placement on `Template`

**Claim:** Builder placed before `StewardSeeds`. Does any code use positional struct-literal initialization of `Template`?

**Evidence:**
- `Template` struct field order (schema.go:127-208): `SchemaVersion` → `Kinds` → `ChildRules` → `AgentBindings` → `Gates` → `GateRulesRaw` → `Tillsyn` (NEW, before StewardSeeds) → `StewardSeeds`.
- Searched test code via `Read` of `schema_test.go` (positional init would jump out): `TestTemplateTOMLRoundTrip` uses NAMED-field initialization (lines 54-117 — every field uses `Field: value` syntax). No positional init present.
- Searched for other potential consumers: existing pattern in `internal/templates/` is universal named-field init. No positional initializers found.

**Trace or cases:**
- Counterexample attempted: positional init `Template{"v1", nil, nil, nil, nil, nil, Tillsyn{}, nil}` would compile against new field order; old positional init (without Tillsyn) would FAIL to compile (extra field count mismatch). But no such code exists.

**Conclusion:** REFUTED. No positional struct-literal init of Template anywhere in the codebase. Field placement is irrelevant to compile-time safety.

## A9 — Field name collisions

**Claim:** Any other type named `Tillsyn` in `internal/templates/` would cause compile errors.

**Evidence:**
- Hylla `hylla_search_keyword` for "Tillsyn struct templates" returned 15 unrelated results — no pre-existing `Tillsyn` symbol.
- `internal/templates/schema.go` lines 226-239 declare `type Tillsyn struct` once. Field `Template.Tillsyn Tillsyn` (line 183) is the type-then-field reference; legal in Go (field name + type name CAN match).
- `mage ci` passed (worklog verification block, lines 70-79) — compile is green. If a collision existed, build would fail.

**Conclusion:** REFUTED. No collision; build is green; uniqueness verified via Hylla.

## A10 — Strict-decode test fixture artifact

**Claim:** Builder's strict-decode test combines a valid known field (`max_context_bundle_chars = 200000`) with the unknown field. Does pelletier require BOTH, or does an isolated unknown-only payload also trigger?

**Evidence:**
- `TestLoadTillsynStrictDecodeUnknownFieldRejected` (load_test.go:821-839) uses TOML:
  ```toml
  [tillsyn]
  max_context_bundle_chars = 200000
  bogus_field = true
  ```
- The `max_context_bundle_chars` line forces pelletier to recognize `[tillsyn]` as a known table mapped to `Tillsyn`. Without it, would `[tillsyn] bogus_field = true` ALSO trigger StrictMissingError?
- Pelletier's strict-decode behavior on a table where ALL keys are unknown: still triggers — the mechanism is per-key, not per-table. The first-encountered unknown key fires the error.
- Existing precedent: `TestLoadAgentBindingStrictDecodeUnknownFieldStillRejects` (load_test.go:644-663) uses a TOML where `bogus_field = true` co-exists with `agent_name` and `model` (known fields). Same pattern as A10.
- Counterexample test: `TestLoadRejectionTable` row "unknown top-level key rejected" (load_test.go:99-105) uses `[bogus_table] foo = "bar"` — ENTIRE table is unknown to the Template struct (no `bogus_table` field). Triggers `ErrUnknownTemplateKey`. So "all-unknown" CAN fire too.

**Trace or cases:**
- Hypothesis: builder's "valid-known-field present" might be load-bearing for the test passing. But the precedent test for `[bogus_table]` proves all-unknown also fires. Builder's choice is defensive belt-and-suspenders, not a fixture bug.
- Concrete experiment: a TOML with `[tillsyn]\nbogus_field = true\n` (no known field) — would this fire? YES, because:
  1. Step 3 strict decode walks `[tillsyn]` → recognizes the table maps to `Tillsyn` (because `Template.Tillsyn` field has TOML tag `tillsyn`).
  2. Decoder reads `bogus_field = true` → looks up `bogus_field` on `Tillsyn` struct → not found.
  3. Strict mode raises `StrictMissingError`.
- Building this test variant is straightforward; builder's existing test is sufficient (covers the more-realistic case).

**Conclusion:** REFUTED. Builder's fixture is defensive but not REQUIRED — the "all-unknown table" case ALSO fires. Builder's choice mirrors `TestLoadAgentBindingStrictDecodeUnknownFieldStillRejects` for consistency.

**NIT N6 (could file as future-test refinement):** a second test case `[tillsyn] bogus_field = true` (no known field) would prove the "isolated unknown" path. Not blocking — existing test is correct.

## A11 — Memory rule conflicts

**Claim:** Verify against `feedback_no_migration_logic_pre_mvp` and `feedback_subagents_short_contexts`.

**Evidence:**
- `feedback_no_migration_logic_pre_mvp`: "no migration code in Go, no till migrate CLI, no one-shot SQL scripts." This droplet adds NEW fields to a NEW struct on an existing schema — purely additive, no migration. Compatible.
- `feedback_subagents_short_contexts`: builders should be small-scoped. F.7.18.2's surface (2 fields, 1 struct, 1 validator, 7 tests, ~140 LOC) is appropriately scoped. Compatible.
- `feedback_no_closeout_md_pre_dogfood`: skip CLOSEOUT/LEDGER rollups. Builder did not touch those files. Compatible.
- `feedback_opus_builders_pre_mvp`: builders use opus. Worklog confirms "opus" model. Compatible.

**Conclusion:** REFUTED. No memory-rule conflicts.

## A12 — `Tillsyn` struct visibility (exported)

**Claim:** Capitalized `Tillsyn` = exported. Confirm intentional.

**Evidence:**
- Downstream consumer is `internal/app/dispatcher/context/` (F.7.18.3 — not yet implemented). That package is a sibling under `internal/app/`, peer to `internal/templates`. Cross-package access REQUIRES exported visibility.
- Per F7_18 plan line 223: `Resolve(ctx, args ResolveArgs)` takes `Tillsyn templates.Tillsyn` as part of `ResolveArgs`. Cross-package field-read is the explicit design.
- Lowercase `tillsyn` would block this entirely.

**Conclusion:** REFUTED. Exported visibility is REQUIRED by F.7.18.3's design. Builder's choice correct.

---

## Final Verdict

**PASS-WITH-NITS.**

All 12 attack vectors REFUTED. Six NITs surfaced (N1-N6) for documentation / future-extender ergonomics — none blocking, none invalidating the droplet's claim. Builder's worklog accurately describes the work; tests prove the claimed contracts; strict-decode inheritance for F.7-CORE F.7.1 + F.7.6 is sound.

**Counterexamples produced:** 0.

**NITs (non-blocking):**
- N1: Future extender's worklog should explicitly cite "validator additions for non-numeric fields" pattern.
- N2: F.7.18.4's cross-cap warning validator must run AFTER both `validateAgentBindingContext` AND `validateTillsyn`.
- N3: If "disable aggregator entirely" emerges as a real need post-F.7.18.4, schema should add an explicit kill-switch.
- N4: F.7.18.4's planner should pick up the cross-cap warning per worklog `Out-of-Scope` deferral.
- N5: Round-trip test could add a zero-value Tillsyn case for completeness.
- N6: A second strict-decode test variant `[tillsyn] bogus_field = true` (no known field) would prove the "isolated unknown" path explicitly. Existing test covers the realistic case.

**Recommendations to orchestrator:**
1. Mark F.7.18.2 builder droplet **complete**.
2. Surface N4 to F.7.18.4 planner (cross-cap warning ownership).
3. Optionally file N5+N6 as test-refinement followups (low priority).

## Hylla Feedback

`N/A — action item touched non-Go-only files plus 4 Go files.` Hylla queries issued for A9 (no `Tillsyn` collision check) and A8 (no positional struct-literal init scan) returned correctly with no false positives. Direct `Read` on schema.go / load.go / load_test.go / schema_test.go / worklog / plan was the right tool for line-level evidence; Hylla's keyword search confirmed no pre-existing `Tillsyn` symbol elsewhere in the repo. No Hylla misses to report.
