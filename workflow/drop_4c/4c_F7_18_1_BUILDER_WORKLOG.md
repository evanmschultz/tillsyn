# Drop 4c — F.7.18.1 Builder Worklog

## Droplet

`4c.F.7.18.1` — Schema-2: closed `Context ContextRules` sub-struct on `templates.AgentBinding` plus the `validateAgentBindingContext` Load-time validator.

REVISIONS-first compliance: read REV-1 + REV-3 in `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md` before reading the body. REV-1 is informational (F.7.17.1 already merged at commit `4177264` ships only `Env`/`CLIKind`). REV-3 confirms F.7.18.1 ships ONLY the `Context` sub-struct on `AgentBinding`; the top-level `Tillsyn` globals struct (formerly Schema-3) is F.7.18.2's territory and the aggregator engine is F.7.18.3's territory.

## Files edited

- `internal/templates/schema.go` — added `Context ContextRules` field at the end of the `AgentBinding` struct (TOML tag `context`), with full doc-comments anchoring it to master `PLAN.md` L13 and the F.7.18 plan acceptance criteria + REV-3. Declared the new closed `ContextRules` struct with eight fields and explicit TOML tags on every field:
  - `Parent bool` (`parent`)
  - `ParentGitDiff bool` (`parent_git_diff`)
  - `SiblingsByKind []domain.Kind` (`siblings_by_kind`)
  - `AncestorsByKind []domain.Kind` (`ancestors_by_kind`)
  - `DescendantsByKind []domain.Kind` (`descendants_by_kind`)
  - `Delivery string` (`delivery`)
  - `MaxChars int` (`max_chars`)
  - `MaxRuleDuration Duration` (`max_rule_duration`) — reuses the existing `templates.Duration` `TextUnmarshaler` so TOML duration strings ("500ms", "2s") decode into the field.
  Added closed-enum delivery constants `ContextDeliveryInline = "inline"` + `ContextDeliveryFile = "file"` so callers (and F.7.18.3's engine) reference the closed vocabulary by name rather than by literal string.
- `internal/templates/load.go`:
  - Added `time` import (the validator uses `time.Duration(ctx.MaxRuleDuration) < 0` to reject negative durations).
  - Extended the `Load` doc-comment validator-order list with step 4(g) describing `validateAgentBindingContext`.
  - Wired `validateAgentBindingContext(tpl)` into `Load` after `validateAgentBindingEnvNames`.
  - Added sentinel `ErrInvalidContextRules = fmt.Errorf("%w: context", ErrInvalidAgentBinding)` so callers using `errors.Is(err, ErrInvalidAgentBinding)` continue to route correctly.
  - Added `var validContextDeliveryValues = []string{"", ContextDeliveryInline, ContextDeliveryFile}` — closed three-value vocabulary including the empty string (which resolves to `"file"` at engine-time per master PLAN L13's "consumer-time default" framing).
  - Added `validateAgentBindingContext(tpl Template) error` that rejects (a) Delivery values outside the closed three-value vocabulary, (b) negative MaxChars (zero is legal — engine substitutes default at F.7.18.4 wiring time), (c) negative MaxRuleDuration (zero is legal — same default-substitution semantics), and (d) any kind reference in `SiblingsByKind` / `AncestorsByKind` / `DescendantsByKind` that is not a member of the closed 12-value `domain.IsValidKind` enum. Per-binding inner-checks run in a stable field order; outer-map iteration returns on the first offending field for bounded error surface.
  - Added `isValidContextDelivery(v string) bool` (exact-match — no whitespace trimming or case folding, mirroring `IsValidGateKind`'s rationale that silent fold-matching masks `"Inline"`/`"FILE"` typos at load time).
  - Added `validateContextKindList(kind, fieldName, kinds)` helper consolidating the three kind-walk slice checks so the validator stays at one screenful.
- `internal/templates/agent_binding_test.go` — extended `fullyPopulatedAgentBinding` to populate every `ContextRules` field with a non-zero validation-passing value. Updated header comment from "13 fields" to "14 fields" + added the F.7.18.1 extension paragraph. Added `domain` import (the Context fixture references `domain.KindBuildQAProof` etc.). Without this update the round-trip test would fail because pelletier/go-toml/v2 emits the `[context]` table even for a zero-valued sub-struct, and nil slices decode back as empty slices — the existing reflect.DeepEqual assertion would trip on the asymmetry.
- `internal/templates/schema_test.go` — extended `TestTemplateTOMLRoundTrip`'s populated `AgentBindings` literal with a fully-populated `Context` block so the Template-level round-trip assertion catches a TOML-tag drop on any of the eight new ContextRules fields.
- `internal/templates/context_rules_test.go` (NEW) — nine acceptance tests, one per spawn-prompt scenario:
  - `TestLoadAgentBindingContextHappyPath` — `[context]` block with `parent = true, parent_git_diff = true, ancestors_by_kind = ["plan"], delivery = "file", max_chars = 50000, max_rule_duration = "500ms"` decodes cleanly and every field lands verbatim.
  - `TestLoadAgentBindingContextEmptyTablePresent` — `[agent_bindings.build.context]` heading present but no nested keys → struct decodes to zero value, validator passes.
  - `TestLoadAgentBindingContextOmittedAltogether` — no `[context]` table at all → zero value, validator passes (master PLAN L13 fully-agentic-mode path).
  - `TestLoadAgentBindingContextRejectsInvalidDelivery` — `delivery = "stream"` rejected with `ErrInvalidContextRules` AND `errors.Is(_, ErrInvalidAgentBinding)` (sentinel chain).
  - `TestLoadAgentBindingContextRejectsNegativeMaxChars` — `max_chars = -1` rejected; error message contains `max_chars` for UX.
  - `TestLoadAgentBindingContextRejectsNegativeMaxRuleDuration` — `max_rule_duration = "-1s"` rejected; error message contains `max_rule_duration` for UX. **Important note on spawn-prompt drift**: the spawn prompt's "test scenarios" bullet listed `"0s"` as a "MUST fail" case alongside `"-1s"`, but its "validators" bullet says "if non-zero, must be positive duration" (i.e. zero is OK), and the empty-`[context]` happy-path test requires zero-valued `MaxRuleDuration` to load cleanly. Resolved per F.7.18 plan body L96 + the consistent reading: zero is legal (engine-time default-substitution at F.7.18.4); only strictly-negative durations fail at the schema layer. The test rejects `"-1s"` only.
  - `TestLoadAgentBindingContextRejectsInvalidKindReference` — `siblings_by_kind = ["bogus_kind"]` rejected with `ErrUnknownKindReference` (consistent with the existing kinds-map / child-rules vocabulary checks — context kind references use the same sentinel rather than introducing a context-specific one).
  - `TestLoadAgentBindingContextAllowsDescendantsOnPlanKind` — explicit allow-test: `[agent_bindings.plan.context] descendants_by_kind = ["build", "plan"]` MUST load cleanly. Per master PLAN L13's "template authors trusted" framing the schema does NOT reject descendants on `kind=plan`; round-history fix-planners + tree-pruners legitimately walk down from a plan parent.
  - `TestLoadAgentBindingContextStrictDecodeRejectsUnknownKey` — `[agent_bindings.build.context] bogus_field = true` rejected with `ErrUnknownTemplateKey`. Proves closed-struct unknown-key rejection (which depends on every `ContextRules` field carrying an explicit TOML tag) actually fires for the new sub-struct.
  - Plus an unexported `assertContextRulesZero(t, ctx)` helper that does a field-by-field comparison instead of `ctx == ContextRules{}` (the latter does not compile because ContextRules carries slices, which are not comparable in Go). The helper treats nil-vs-empty-slice asymmetry — which pelletier/go-toml/v2 introduces for empty arrays — as equivalent for the "zero-or-effectively-zero" contract.

## NOT edited (per REV-3 + droplet boundary discipline)

- No top-level `Tillsyn` struct on `Template`. That's F.7.18.2's territory.
- No `[tillsyn]` block validators (`MaxContextBundleChars` / `MaxAggregatorDuration` / cross-cap warning). F.7.18.2.
- No new `internal/app/dispatcher/context/` package. F.7.18.3.
- No greedy-fit cap algorithm + per-rule / per-bundle wall-clock timeouts. F.7.18.4.
- No `[context]` block edits in `internal/templates/builtin/default.toml`. F.7.18.5.
- No `metadata.spawn_history[]` doc-comment. F.7.18.6 (or absorbed into F.7.9 per Q1).
- No engine-time default-substitution constants (`50000`, `200000`, `500ms`, `2s`). Those land in the engine package alongside the cap algorithm (F.7.18.3 / F.7.18.4); the schema validator only enforces the field-shape contract (non-negative + closed-enum membership).

The diff is strictly the F.7.18.1 minimum surface.

## Verification

- `mage check` (alias of `mage ci`) — green. Test suite total: 277 tests in `internal/templates` (268 pre-existing + 9 new acceptance) all pass; full suite across 21 packages green with 1 pre-existing unrelated skip.
- `internal/templates` package coverage: **95.8%** (well above the 70% gate).
- `mage testPkg ./internal/templates` — 277 tests pass, no race regressions.
- All 9 spawn-prompt acceptance scenarios are real, named tests (verified individually via `mage testFunc`).

## Acceptance criteria — all met

- [x] `Context ContextRules` field on `AgentBinding` with TOML tag `context` + doc-comment citing master PLAN L13.
- [x] `ContextRules` struct with eight fields (Parent / ParentGitDiff / SiblingsByKind / AncestorsByKind / DescendantsByKind / Delivery / MaxChars / MaxRuleDuration), each with explicit TOML tag + doc-comment.
- [x] `ContextDeliveryInline` + `ContextDeliveryFile` closed-enum constants declared.
- [x] `validateAgentBindingContext` wired into `templates.Load` validator chain after `validateAgentBindingEnvNames`.
- [x] All 9 test scenarios pass.
- [x] No schema rule rejecting `descendants_by_kind` on `kind=plan` — verified by the explicit allow-test (`TestLoadAgentBindingContextAllowsDescendantsOnPlanKind`).
- [x] `mage check` + `mage ci` green.
- [x] Worklog written.

## Spawn-prompt drift documented

One concrete drift between the spawn prompt's "validators" rule (zero MaxRuleDuration legal) and its "test scenarios" rule (`max_rule_duration = "0s"` MUST fail) was resolved in favor of the validators rule, consistent with the empty-`[context]` happy-path requirement and the F.7.18 plan body L96 default-substitution semantics. Documented inline in the rejection test's doc-comment so a future reader hitting the same friction sees the rationale immediately.
