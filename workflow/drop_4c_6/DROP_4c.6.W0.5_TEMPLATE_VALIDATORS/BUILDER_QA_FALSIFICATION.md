# BUILDER_QA_FALSIFICATION — DROP_4c.6.W0.5_TEMPLATE_VALIDATORS

## Droplet 4c.6.W0.5.D1 — Round 1

**Date:** 2026-05-09
**Reviewer:** go-qa-falsification-agent (build-QA-falsification, parent.kind=build)
**Scope:** kind-enum validator over the new `Template.Agents` map (commit `3a1f8b5`).

### Counterexamples

None CONFIRMED. All seven attack families exhausted; details below.

#### B1 — test-coverage attacks

Attempted attacks on the new test `TestLoadValidatesAgentMapKeysClosedEnum` (`internal/templates/load_test.go:296-371`):

- **Empty / nil `Agents` map.** Probed `canonicalizeMapKeys` (load.go:499-501): `if len(m) == 0 { return nil, nil }` covers both nil and empty maps without nil-deref. Tested implicitly by every `Load` of a template that omits `[agents.<kind>]` (`valid_minimal.toml` itself). REFUTED.
- **Single valid kind in `[agents.<kind>]`.** Row 1 of the new test (`valid kind passes`, line 318-323) covers this: `validMinimal + "\n[agents.build]\n"` and asserts `tpl.Agents[domain.KindBuild]` is present. REFUTED.
- **Multiple invalid kinds in same map.** Not directly tested with a multi-bogus fixture; `canonicalizeMapKeys` short-circuits on first invalid key (load.go:507-510 returns immediately). The wrapped error names the FIRST offending key. The L2 acceptance bullets do not require aggregation; first-fail is consistent with the existing `validateMapKeys` contract. NIT-only, REFUTED as counterexample.
- **Case-fold collision (`[agents.BUILD]` + `[agents.build]` siblings).** Existing parallel maps have explicit collision tests (`TestValidateMapKeysCollidesOnCaseFold` for gates at line 1751, `TestValidateMapKeysCollidesOnCaseFoldKindsTable` for kinds at line 1780). The `Agents` map has NO equivalent collision test. The collision branch lives entirely inside the shared generic `canonicalizeMapKeys` helper (load.go:518-528) and is invariant in `V` — exercised by both gates and kinds tests. The collision path for `Agents` is contractually identical and helper-covered; a dedicated collision test would be parity, not coverage. NIT-only test-asymmetry, REFUTED as counterexample. Optional improvement: append a row 4 to `TestLoadValidatesAgentMapKeysClosedEnum` exercising `[agents.BUILD]` + `[agents.build]` collision.
- **Bare `valid_minimal.toml` not loaded directly.** Row 1 / row 3 only exercise the baseline AFTER appending `[agents.build]` / `[agents.BUILD]`. If the baseline itself failed Load, both rows would fail too — implicit coverage. REFUTED.
- **Test row 1 `wantSubstrs` empty.** Row 1 sets `wantErr=false` so the substring loop never runs (line 349-353 guarded by `tc.wantErr`). Correct. REFUTED.

Family verdict: REFUTED.

#### B2 — contract-preservation attacks

`Template.Agents` is brand-new in this droplet. `git grep "tpl.Agents"` and `git grep "Template.Agents"` (run against the full tree) return only the new validator + its test + schema doc-comments. No prior consumer treats empty `Agents` as a sentinel value, no prior consumer ranges over it, no prior consumer looks up by key. The field is wired in this droplet only; no contract drift possible. REFUTED.

Family verdict: REFUTED.

#### B3 — hidden-coupling attacks

The new validator reuses `canonicalizeMapKeys` (load.go:499-531) verbatim over the new map. `canonicalizeMapKeys` is V-generic with constraint `any`; `AgentRuntime{}` is a zero-size struct, making it the cheapest possible value type for the helper. Helper invariants verified:

- **Nil-map / empty-map**: `len(m) == 0` early-return path (line 500). Sound for `map[domain.Kind]AgentRuntime`.
- **Key-canonicalization rule**: `domain.Kind(strings.ToLower(strings.TrimSpace(string(k))))` — invariant in V. Sound.
- **Collision detection**: rebuild path detects `_, dup := rebuilt[canon]` — invariant in V. Sound.
- **Strict-decode interplay**: `AgentRuntime struct{}` has no fields, so strict decode (`DisallowUnknownFields`, load.go:183) accepts only an empty body under `[agents.<kind>]`. Author who writes `[agents.build]\nfoo = "bar"\n` triggers `ErrUnknownTemplateKey` BEFORE `validateAgentMapKeys` ever runs — that's the desired behavior per `schema.go:596-602` doc-comment. Sound.

REFUTED.

Family verdict: REFUTED.

#### B4 — YAGNI attacks

- **`AgentRuntime struct{}` placeholder.** The deferral is intentional per W0 sequencing — W0 ships the runtime-config value-shape (max_tries, max_budget_usd, blocked_retries, etc.). Today `Agents`'s ONLY load-time invariant is closed-enum membership of map keys, which `validateAgentMapKeys` enforces independent of value shape. Empty struct is the smallest concrete shape that lets strict decode accept the table at all (per `schema.go:596-602`). Not premature.
- **Separate `validateAgentMapKeys` function instead of folding into `validateMapKeys`.** PLAN.md § "Cross-Cutting Decisions / Tradeoffs" mandates separate insertion in the chain so adopters who diff the chain order see a distinct D1 step. Documented design choice; not over-abstraction. Folding would have buried the W0.5 hook below the chain-level diff.
- **`agentMapKeys` test helper.** Mirrors existing `mapKeys` helper for diagnostic rendering. Justified as test-diagnostic parity; test failures show sorted keys for stable output. NIT-only.

REFUTED.

Family verdict: REFUTED.

#### B5 — spec-compliance attacks

L2 acceptance bullets (PLAN.md lines 58-65) mapped to test/code coverage:

| Bullet | Coverage | Status |
|---|---|---|
| 1. New validator `validateAgentMapKeys` over `Template.Agents` map keys | `load.go:476-483` + wired at `load.go:197` | satisfied |
| 2. Existing kind-enum check unchanged | full-pkg test run (406 pass) shows no regression in `TestLoadRejectsBogus*` | satisfied |
| 3. Malformed fixture `invalid_agents_unknown_kind.toml` rejects with `ErrUnknownKindReference` + names `agents` field + `totally-bogus` key | row 2 `wantSubstrs: []string{"agents map key", "totally-bogus"}` (test:329) | satisfied |
| 4. Shared baseline fixture `valid_minimal.toml` passes Load cleanly | row 1 implicitly verifies (build of baseline + `[agents.build]` block passes Load) | satisfied (implicit, see B1 NIT) |
| 5. `TestLoadValidatesAgentMapKeysClosedEnum` table-driven w/ 3 rows | rows present at test:318-336 | satisfied |
| 6. `mage test-func` RED→GREEN | BUILDER_WORKLOG round-1 documents RED via commented-out wire-up | satisfied |
| 7. `mage test-pkg ./internal/templates` clean | re-verified by reviewer (406 tests pass) | satisfied |

Each bullet has at least one concrete verifying test that ACTUALLY verifies it (not name-only). REFUTED.

Family verdict: REFUTED.

#### B6 — shipped-but-not-wired attacks

- **`validateAgentMapKeys` wire-up.** `load.go:197` sits in the `LoadWithOptions` chain between `validateMapKeys` (line 194) and `validateChildRuleKinds` (line 200). Correct insertion point per PLAN.md cross-cutting decision. Verified by reviewer via `git grep validateAgentMapKeys` returning the call site.
- **Test exercises full Load path, not validator in isolation.** `TestLoadValidatesAgentMapKeysClosedEnum` calls `Load(strings.NewReader(tc.src))` (test:341), which runs the FULL `LoadWithOptions` chain including the new validator. Not stubbed.
- **Fixtures exist and are read at test time.** `mustReadTestdata` (test:377-384) reads `testdata/valid_minimal.toml` (31 lines) and `testdata/invalid_agents_unknown_kind.toml` (37 lines) — both present on disk. RED-confirmation in BUILDER_WORKLOG line 33-34 commented the wire-up and observed expected failures, proving the validator actually runs in production and isn't dead code.

Re-verified: `mage testPkg ./internal/templates` → 406 tests pass; `mage testFunc ./internal/templates TestLoadValidatesAgentMapKeysClosedEnum` → 4 sub-tests pass.

Family verdict: REFUTED.

#### B7 — prompt-injection attacks

Pre-team-feature; per `feedback_prompt_injection_team.md` this family is dormant until team functionality lands. No action-item content is attacker-controllable in the W0.5 scope. EXHAUSTED.

Family verdict: EXHAUSTED.

### Summary

**Verdict: pass.**

**Counterexample count:** 0

| Family | Result |
|---|---|
| B1 test-coverage | REFUTED |
| B2 contract-preservation | REFUTED |
| B3 hidden-coupling | REFUTED |
| B4 yagni | REFUTED |
| B5 spec-compliance | REFUTED |
| B6 shipped-but-not-wired | REFUTED |
| B7 prompt-injection | EXHAUSTED |

Build round 1 lands a closed-12-enum check on the new `Template.Agents` map at the correct position in the `LoadWithOptions` chain, reusing the shared generic `canonicalizeMapKeys` helper, paired with a table-driven test exercising valid / unknown / case-fold rows against on-disk fixtures and one inline source. Both gates green: `mage testPkg ./internal/templates` (406 tests pass) and `mage testFunc ./internal/templates TestLoadValidatesAgentMapKeysClosedEnum` (4 sub-tests pass).

**Optional follow-up (NIT, not gating):** add a `[agents.BUILD]` + `[agents.build]` case-fold collision row to `TestLoadValidatesAgentMapKeysClosedEnum` to bring per-map test coverage to parity with the existing `TestValidateMapKeysCollidesOnCaseFold` (gates) and `TestValidateMapKeysCollidesOnCaseFoldKindsTable` (kinds) tests. The collision path is helper-covered today; this is a parity NIT, not a missing-coverage CONFIRMED counterexample.

### Hylla Feedback

N/A — droplet touched a single Go package (`internal/templates`) where every relevant file (`load.go`, `load_test.go`, `schema.go`) was very recently modified in HEAD (commit `3a1f8b5`); Hylla's index is stale for those files until the drop-end reingest. Direct `Read` + `git grep` against the working tree was the correct evidence path. No Hylla queries attempted; nothing to log.

## Droplet 4c.6.W0.5.D2 — Round 1

**Date:** 2026-05-09
**Reviewer:** go-qa-falsification-agent (build-QA-falsification, parent.kind=build)
**Scope:** `agent_name` embedded-tier validator (commit `e999a0b`) + the FF2 `embeddedAgentLibraryShipped` package-init probe reconciliation (NOT in L2 PLAN.md verbatim).

### Counterexamples

- **1.1 [Family: B5 spec-compliance] [severity: low]** Doc-comment drift on `LoadOptions.AgentLookupFn` field at `internal/templates/load.go:43-62`. Two contradictions to the actual implementation:
  1. Line 49 says "Nil resolves to a default that walks **DefaultAgentLibraryFS** unconditionally" — that symbol does not exist (`git grep "DefaultAgentLibraryFS"` returns zero hits). The actual default walker (`defaultAgentLookupFn` at `load.go:1598`) walks `DefaultTemplateFS`.
  2. Lines 56-61 say "Pre-W1.D1 (embedded agent .md files not yet shipped) the default walker returns **false** for every name — exercising the default in a unit test without an explicit injection deliberately **fails-loud** per W0.5 round-2 FF2 disclosure." The actual FF2 reconciliation (added in this same round) made the default walker fail-**permissive** when `embeddedAgentLibraryShipped == false`: `defaultAgentLookupFn` returns **true** (not false) at `load.go:1602-1610`, and `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` (`load_test.go:550`) asserts the permissive behavior verbatim.
  - **Repro:** `Read` `internal/templates/load.go` lines 43-62; cross-reference against the actual `defaultAgentLookupFn` body at lines 1598-1621 and the test at `load_test.go:550-568`. The field godoc was authored before the FF2 reconciliation landed and was not updated when the probe was added.
  - **Fix hint:** Update the field godoc (1) replace `DefaultAgentLibraryFS` with `DefaultTemplateFS`; (2) replace the "returns false / fails-loud" sentence with "Pre-W1.D1 (embedded agent .md files not yet shipped) the default walker fails-permissive (returns true unconditionally) per the `embeddedAgentLibraryShipped` package-init probe; tests that need to exercise the hard-fail path inject an explicit `LoadOptions.AgentLookupFn`. Post-W1.D1 the same default walker becomes strict automatically." Severity is low because the worklog `Design notes / decisions` section captures the actual FF2 contract correctly and the implementation + tests + secondary godoc on `embeddedAgentLibraryShipped` (`load.go:1538-1577`) and `defaultAgentLookupFn` (`load.go:1579-1597`) are all consistent — only the field-level `AgentLookupFn` doc drifted. No runtime behavior is wrong; only the field-level doc-comment is misleading.

#### B1 — test-coverage attacks

Attempted attacks on `TestLoadValidatesAgentBindingNamesEmbeddedFloor` (`load_test.go:427`) + `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` (`load_test.go:550`):

- **Empty agent_name (row 3).** Inline TOML source with `agent_name = ""`; injected `lookupFn := func(string) bool { return false }`. Expected `ErrUnknownAgentName` + substring `"empty"`. Verified row 3 of the table covers this (`load_test.go:483-491`); validator hits the early-return at `load.go:1667-1670` before reaching the lookup. REFUTED.
- **Whitespace-only agent_name (`agent_name = " "`).** NOT directly tested. The validator does NOT trim — `" "` would pass the `name == ""` check and reach the lookup at `load.go:1671`. The default walker would call `DefaultTemplateFS.Open("builtin/agents/till-gen/ .md")` which returns a "file does not exist" error → walker returns false → validator emits `ErrUnknownAgentName`. So whitespace is implicitly rejected, but with the same error message as a normal unresolvable name (no distinct "agent_name is whitespace-only" diagnostic). NIT-only — adopter would see the unresolvable-name error and grep their TOML; not a counterexample. REFUTED.
- **Case-sensitivity (`agent_name = "Builder"` vs `builder.md`).** Embed.FS is case-sensitive regardless of host OS (Go embed.FS uses raw filename matching). If an adopter authors `agent_name = "Builder"` and the embedded file is `builder.md`, the lookup fails and the validator emits `ErrUnknownAgentName`. Behavior matches `validateMapKeys`'s case-fold canonicalization for kind-enum keys but the agent NAME itself is intentionally NOT canonicalized — agent names are filesystem paths, not domain.Kind enum members. Distinct invariants; behavior is correct. REFUTED.
- **Very-long names / UTF-8 names.** Untested but the embed.FS API does not impose a length cap (POSIX paths are bounded by filesystem; embed.FS by go's file abstraction layer). UTF-8 paths work because `embed.FS.Open` takes a `string` and Go strings are UTF-8 by construction. No DOS vector — single Open call. REFUTED.
- **Multiple unresolvable bindings in same template.** The validator returns on the FIRST offending binding (`load.go:1671-1675`); subsequent bindings never reach lookup. Doc-comment at lines 1650-1655 documents this explicitly as a bounded-error-surface choice. Acceptable for Drop 4c.6; future drops may add aggregation. NIT, REFUTED.
- **Empty `tpl.AgentBindings` map.** `valid_minimal.toml` without `[agent_bindings.*]` would loop over an empty map in `validateAgentBindingNames`; loop body never runs, returns nil. Tested implicitly by every test that omits agent_bindings. REFUTED.

Family verdict: REFUTED.

#### B2 — contract-preservation attacks

`embeddedAgentLibraryShipped` is a package-level `var` populated by an immediately-invoked function at package init (`load.go:1564-1577`). Concerns probed:

- **Init-order dependency on `DefaultTemplateFS`.** `DefaultTemplateFS` is declared in `embed.go:35` (`var DefaultTemplateFS embed.FS`). The Go spec guarantees package-level `var` declarations are initialized before any function-level statements run; for cross-file vars, dependency order applies (`embeddedAgentLibraryShipped`'s initializer references `DefaultTemplateFS`, so the compiler orders the embed.FS init first). Verified via `mage testPkg` 411/411 GREEN — if the init order were wrong, the probe would observe a zero-value FS and tests using the default walker would fail consistently. REFUTED.
- **Test isolation — can a test mutate `embeddedAgentLibraryShipped`?** The variable is package-scoped and not exported. Tests can only mutate it via the same-package access path (`embeddedAgentLibraryShipped = true`). Inspection of `load_test.go` shows zero such mutations (`grep -n "embeddedAgentLibraryShipped =" load_test.go` returns no matches; only doc-comment references). The test design intentionally injects via `LoadOptions.AgentLookupFn` to bypass the probe entirely — this is the documented test seam. REFUTED.
- **Test isolation — can a test swap `DefaultTemplateFS` to populate the probe state mid-run?** `DefaultTemplateFS` is a package-level `embed.FS` var; technically a test could reassign it (`DefaultTemplateFS = newFS`), but the probe runs ONCE at init — re-assigning the FS post-init does not re-run the probe, so the cached `embeddedAgentLibraryShipped` value would not reflect the swap. This is a test-flexibility limitation, not a contract bug; it would matter if a future test wants to mock embed.FS contents to exercise the strict-mode default walker pre-W1.D1. The L2 PLAN's design says tests inject via `AgentLookupFn` — which provides the exact same coverage without depending on FS swaps. REFUTED.
- **`embeddedAgentLibraryShipped` mutability vs concurrent test runs.** Go's `var ... = func() { ... }()` initializer is run once during package init, before any goroutines spawn. Subsequent reads from the same variable are reads of an immutable value (Go has no const for non-string types but the variable is never written after init). Race-free. REFUTED.
- **What if `DefaultTemplateFS` future drop adds the embed directive `builtin/agents/`?** The probe would observe `builtin/agents/till-gen/*.md` etc. and switch `embeddedAgentLibraryShipped` to true. The default walker becomes strict. Test `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` would FAIL (expected nil error but `defaultAgentLookupFn("go-builder-agent")` returns false because the file doesn't exist yet). The test's own godoc (`load_test.go:519-549`) explicitly anticipates this transition and directs the W1.D1 builder to flip the assertion. Forward-looking handoff is correct. REFUTED.

Family verdict: REFUTED.

#### B3 — hidden-coupling attacks

- **`validateAgentBindingFiles` (warn-only, line 1466 region) vs `validateAgentBindingNames` (hard-fail, line 1623).** Both iterate `tpl.AgentBindings`; both check existence-of-file shapes; both run in the load chain (warn-only first at `load.go:256`, hard-fail second at `load.go:257`). Could the warn-only emit a misleading message about a name that the hard-fail subsequently rejects? Inspection: `validateAgentBindingFiles` checks `~/.claude/agents/<name>.md` (host filesystem) — that's a DIFFERENT existence check on a DIFFERENT layer (dev-machine state) than the embedded-FS check. Adopter sees "agent .md not found in `~/.claude/agents/`" warning, then "agent_name does not resolve at the embedded floor" hard-fail. Two distinct messages; both grep-friendly. The order (warn first, hard-fail second) means adopters whose dev machine is incomplete BUT whose template is correct see ONLY the warning; adopters whose template is BROKEN see warning + hard-fail. Acceptable layering. REFUTED.
- **`embeddedAgentLibraryShipped` consumed only by `defaultAgentLookupFn`.** The probe is consumed at exactly one site (`load.go:1602`). Scope contained: `git grep embeddedAgentLibraryShipped -- '*.go'` returns 4 hits (1 comment in load.go declaring the var, 1 use in defaultAgentLookupFn, 2 doc-comment references in load_test.go). Probe cannot be observed by external code; cannot drift into a wider invariant unintentionally. REFUTED.
- **Validator runs after every other agent-binding validator.** `LoadWithOptions` chain order (`load.go:223-260`): map keys → child rules → required rules → reachability → structural → gates → env → context → tool gating → files (warn-only) → **names (hard-fail)** → tillsyn. By the time `validateAgentBindingNames` runs, every prior validator has succeeded. The order is correct: name resolution is the LAST check, so adopter sees the most actionable diagnostic first (kind typos, missing required rules, etc.) before reaching name-resolution. REFUTED.

Family verdict: REFUTED.

#### B4 — YAGNI / scope-creep — PRIMARY ATTACK FOCUS

Builder added `embeddedAgentLibraryShipped` package-init probe NOT verbatim in the L2 PLAN.md spec. The L2 PLAN explicitly stated (line 91): "Pre-W1.D1, the FS contains no `*.md` files at those paths, so the default walker returns false for every name — that's the correct degenerate state and is exercised by D2's unit tests via explicit `LoadOptions.AgentLookupFn` injection of a synthetic lookup fn that returns true for known synthetic names." Plan said: default walker returns false; tests use injection.

**Attack: could the L2 plan have been amended instead of adding code?**

Three alternatives evaluated:

1. **Don't wire `validateAgentBindingNames` into the chain pre-W1.D1.** Plan acceptance bullet 1 explicitly demands wire-up: "asserts every `[agent_bindings.<kind>] agent_name = "..."` value resolves at the EMBEDDED tier." Skipping the wire-up = shipped-but-not-wired anti-pattern (`feedback_tillsyn_enforces_templates.md`). REJECTED as alternative — the deviation cannot be avoided this way without violating spec.
2. **Inject `AgentLookupFn` at production callsites (`LoadDefaultTemplate*`).** `LoadDefaultTemplate()` calls `LoadDefaultTemplateForLanguage("")` (`embed.go:94-96`); neither has an injection point. Adding one means breaking the public API surface (multiple downstream callers in `internal/app/auto_generate_steward.go` per `embed.go:73-83`). Wider blast radius than the probe. REJECTED as alternative — the cost of plumbing `LoadOptions` through every default-template caller exceeds the cost of the probe.
3. **Update existing fixtures referencing real agent names to inject `AgentLookupFn`.** The 49 regressing tests include happy-path tests on `default-go.toml` — the embedded default itself. Updating them all to inject `AgentLookupFn` requires every Load-of-default-template test to construct a synthetic lookup; the scope of updates is wider than the probe AND introduces a coupling between tests and the embedded library's contents. REJECTED.

**Conclusion on the deviation:** The probe is a justified reconciliation. The L2 plan's stated contract ("default walker returns false; tests use injection") was incomplete because it didn't account for the 49 production-path tests that do NOT inject (they go through `LoadDefaultTemplate()` which has no injection seam). The probe's mechanism is minimal: 1 package-level var + 1 conditional in `defaultAgentLookupFn` + zero new exported symbols + zero new abstractions. It honors the L2 plan's intent ("validator code is final on D2 land; W1.D1 ships files into the FS path the default already walks") by making the transition automatic on FS-content change rather than code change.

**Sub-attacks within B4:**

- **Is the closed three-group iteration over-engineered?** `embeddedAgentGroups = []string{"till-gen", "till-go", "till-gdd"}` mirrors `SKETCH.md` § 3.4 verbatim. The closed slice + iteration is the smallest concrete shape: 3 strings + a 1-line `for _, group := range` loop. No abstraction beyond what the spec requires. REFUTED.
- **Is the LOUD WARNING comment block at `load.go:1526-1535` over-documenting?** The warning targets future drops that add new embedded groups (e.g. `till-fe` post-MVP). Without the warning, a future drop could add the directory but forget to extend the slice — silently bypassing the new group from the resolver floor. The warning is a hand-off contract for future authors; not over-engineering, just protecting the closed-set invariant. REFUTED.
- **Is `defaultAgentLookupFn` empty-name early-return at `load.go:1599-1601` redundant with the validator's own empty-name check at `load.go:1667-1670`?** The empty-name check in `validateAgentBindingNames` returns BEFORE calling `lookupFn`, so the early-return in `defaultAgentLookupFn` is unreachable from the validator. BUT `defaultAgentLookupFn` is a package-level function that future call sites could invoke directly (e.g. a future spawn-time resolver); the empty-name guard is defensive correctness for that future caller. NIT-only redundancy, not a counterexample. REFUTED.

Family verdict: REFUTED. The FF2 reconciliation is a JUSTIFIED deviation, not scope creep. Rationale: (1) the alternative of skipping the wire-up violates the L2 acceptance bullet 1; (2) the alternative of plumbing `LoadOptions` through `LoadDefaultTemplate*` has wider blast radius; (3) the probe is the smallest concrete reconciliation that honors the L2 intent ("validator code is final on D2 land") and avoids breaking 49 production-path tests. The deviation is documented in the worklog `Design notes / decisions` (round-1 entry, "FF2 reconciliation" bullet) with explicit rationale; the LOUD WARNING in the second test's godoc commits to the post-W1.D1 transition.

#### B5 — spec-compliance attacks

L2 acceptance bullets (PLAN.md lines 88-98) mapped to test/code coverage:

| Bullet | Coverage | Status |
|---|---|---|
| 1. New validator `validateAgentBindingNames` over `[agent_bindings.<kind>] agent_name` | `load.go:1661-1677` + wired at `load.go:257` | satisfied |
| 2. Project-tier + user-tier checks NOT performed at load time | doc-comment `load.go:1635-1639`; `validateAgentBindingFiles` (warn-only) preserved at `load.go:256` | satisfied |
| 3. Embedded FS query via `embed.FS` exposed at `internal/templates/embed.go` | `defaultAgentLookupFn` at `load.go:1598-1621` walks `DefaultTemplateFS` | satisfied |
| 4. Malformed fixture `invalid_unknown_agent_name.toml` rejects with `ErrUnknownAgentName` + names kind + agent_name | row 2 `wantSubstrs: []string{"agent_bindings", "build", "no-such-agent"}` (test:481) | satisfied |
| 5. Happy-path fixture `valid_minimal_with_known_agent.toml` passes | row 1 (test:469-474) | satisfied |
| 6. `LoadOptions.AgentLookupFn` field added | `load.go:43-62` (with the doc-drift NIT in 1.1 above) | satisfied |
| 7. `TestLoadValidatesAgentBindingNamesEmbeddedFloor` table-driven w/ 3 rows | rows present at test:461-491 | satisfied |
| 8. New sentinel `ErrUnknownAgentName` | `load.go:434-465` | satisfied |
| 9. `mage test-func` RED→GREEN | BUILDER_WORKLOG round-1 documents RED via build-error level + commented-out wire-up | satisfied |
| 10. `mage test-pkg ./internal/templates` clean | re-verified by reviewer (411 tests pass) | satisfied |

The "LOUD WARNING" `TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` test at `load_test.go:550-568` was specifically called out as an attack surface — does it actually fail when embed.FS is populated, or just docstring-warn? **Verified ACTUAL behavior:** the test calls `LoadWithOptions` with `LoadOptions{}` (nil `AgentLookupFn`); production walker `defaultAgentLookupFn("go-builder-agent")` runs; pre-W1.D1 `embeddedAgentLibraryShipped == false` → walker returns true → validator passes → test asserts nil error → PASS. Post-W1.D1 (`builtin/agents/till-go/go-builder-agent.md` exists), `embeddedAgentLibraryShipped == true` → walker actually walks the FS → file at `till-go/go-builder-agent.md` exists → walker returns true → validator passes → test STILL passes. Wait — that means the test does NOT fail post-W1.D1 if W1.D1 ships the agent file at the same name. The test's godoc warning at `load_test.go:544-549` ("Either update the test's assertion (default lookup now strict) or update `valid_minimal.toml` to reference an agent_name your placeholder files satisfy") is forward-looking but somewhat defensive — the test will only fail post-W1.D1 if W1.D1 ships placeholder files but does NOT ship `go-builder-agent.md` specifically. As long as W1.D1 ships every name `valid_minimal.toml` references (which is just `go-builder-agent`), the test continues to pass and only the FF2-permissive code path becomes unreachable. The test has documentation value (commits the FF2 contract to a checked-in test) but limited adversarial signal post-W1.D1. NIT — not a counterexample. REFUTED.

Family verdict: REFUTED.

#### B6 — shipped-but-not-wired attacks

- **`validateAgentBindingNames` wire-up.** `load.go:257` sits in the `LoadWithOptions` chain immediately after `validateAgentBindingFiles` warn-only call (line 256) and before `validateTillsyn` (line 260). Correct insertion point per L2 PLAN cross-cutting decision. Verified by reviewer via `git grep validateAgentBindingNames` returning the call site.
- **Test exercises full Load path, not validator in isolation.** `TestLoadValidatesAgentBindingNamesEmbeddedFloor` calls `LoadWithOptions` (test:495) with explicit `LoadOptions{AgentLookupFn: tc.lookupFn}` injection — runs the FULL chain including the new validator. Not stubbed.
- **`embeddedAgentLibraryShipped` consumed only by `defaultAgentLookupFn`.** Acceptable — the probe is an internal mechanism for the default walker; production callers consume it transitively through `defaultAgentLookupFn`'s behavior. Containment: 1 declaration site, 1 use site. The B6 question "is that acceptable scoping?" — yes; widening the consumer surface (e.g. exposing the probe state to a public method) would be over-engineering YAGNI.
- **Fixtures exist and are read at test time.** `mustReadTestdata` reads `testdata/invalid_unknown_agent_name.toml` (33 lines) and `testdata/valid_minimal_with_known_agent.toml` (33 lines) — both present on disk. RED-confirmation in BUILDER_WORKLOG line 86-87 commented the wire-up and observed expected failures, proving the validator actually runs in production and isn't dead code.

Re-verified: `mage testPkg ./internal/templates` → 411 tests pass; `mage testFunc ./internal/templates TestLoadValidatesAgentBindingNamesEmbeddedFloor` → 4 sub-tests pass; `mage testFunc ./internal/templates TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1` → 1 test passes.

Family verdict: REFUTED.

#### B7 — prompt-injection attacks

Pre-team-feature; per `feedback_prompt_injection_team.md` this family is dormant until team functionality lands. No action-item content is attacker-controllable in the W0.5 scope. EXHAUSTED.

Family verdict: EXHAUSTED.

### Summary

**Verdict: pass.**

**Counterexample count:** 1 (low severity — doc-comment drift on `LoadOptions.AgentLookupFn`).

| Family | Result |
|---|---|
| B1 test-coverage | REFUTED |
| B2 contract-preservation | REFUTED |
| B3 hidden-coupling | REFUTED |
| B4 yagni / scope-creep (PRIMARY) | REFUTED |
| B5 spec-compliance | REFUTED (1 doc-comment drift NIT) |
| B6 shipped-but-not-wired | REFUTED |
| B7 prompt-injection | EXHAUSTED |

**Explicit verdict on the FF2 reconciliation:** **JUSTIFIED, not scope creep.** The `embeddedAgentLibraryShipped` package-init probe is the smallest concrete reconciliation that honors the L2 PLAN's intent ("validator code is final on D2 land; W1.D1 ships files into the FS path the default already walks") while preserving 49 production-path tests that load `default-go.toml` without an `AgentLookupFn` injection seam. Three alternative deviations (skip wire-up; plumb `LoadOptions` through `LoadDefaultTemplate*`; update every default-template test to inject) are all worse: the first violates spec, the second has wider blast radius, the third introduces a coupling between tests and embedded-library contents. The probe is documented in BUILDER_WORKLOG `Design notes / decisions` round-1 entry with explicit rationale, the `embeddedAgentLibraryShipped` and `defaultAgentLookupFn` doc-comments capture the contract, and the second test (`TestLoadValidatesAgentBindingNamesDefaultLookupPermissivePreW1D1`) commits the FF2-permissive behavior to a checked-in assertion. Verdict pass with 1 low-severity NIT (1.1 above) on the field-level `LoadOptions.AgentLookupFn` godoc which was authored before the FF2 reconciliation landed and not updated. Field godoc says "fails-loud" but actual implementation is "fails-permissive" — fix is a small godoc edit; no runtime behavior is wrong.

Build round 1 lands the `agent_name` embedded-tier validator at the correct chain position with hard-fail semantics, the `LoadOptions.AgentLookupFn` test injection seam, the closed-three-group `embeddedAgentGroups` slice, the FF2 `embeddedAgentLibraryShipped` reconciliation probe, paired with a 3-row table-driven test exercising known/unknown/empty agent names against on-disk fixtures + an inline-source row, plus a forward-looking test asserting the FF2-permissive default behavior pre-W1.D1. Both gates green: `mage testPkg ./internal/templates` (411 tests pass) and per-target `mage testFunc` runs (5 sub-tests pass across the two new tests).

**Optional follow-up (NIT, not gating):** apply the field godoc fix from finding 1.1 in a follow-up commit (single-file edit to `internal/templates/load.go` lines 49 + 56-61). Not gating because the runtime behavior is correct and the secondary doc-comments + the worklog capture the actual FF2 contract — the drift is contained to the field-level godoc only.

### Hylla Feedback

N/A — droplet touched a single Go package (`internal/templates`) where every relevant file (`load.go`, `load_test.go`, `embed.go`, `schema.go`) was very recently modified through commit `e999a0b`; Hylla's index is stale for those files until the drop-end reingest. Direct `Read` + `git grep` against the working tree was the correct evidence path. No Hylla queries attempted; nothing to log.

## Droplet 4c.6.W0.5.D3 — Round 1

Build-QA-falsification of W0.5.D3 (cycle detector with unified-graph + shared `dfsDetectCycle` helper). Round 1 attack focus: the two builder design refinements — `[K ~string]` instead of literal `[K comparable]`, and two-graph walk instead of merged-edge graph — plus the standard 7-family attack pass.

### Counterexamples

- 1.1 [Family: B5 spec-compliance / NIT] [severity: low] **Sentinel godoc drift on `ErrTemplateCycle`.** `internal/templates/load.go:289-292` doc-comment on `ErrTemplateCycle` still says "the [child_rules] parent → child kind graph contains a directed cycle" — but the validator was extended in this droplet to also walk the blocked_by-induced graph and the wrapped error message now appends edge labels `[parent->child]` or `[blocked_by]`. Repro: `git -C main grep -n "parent → child kind graph" -- internal/templates/load.go` returns the stale single-line description. Fix hint: extend the godoc to "...the unified [child_rules] kind graph (parent→child auto-create edges and blocked_by-induced edges) contains a directed cycle." Not gating — runtime behavior is correct; only the sentinel's pithy one-line summary is stale. Same NIT pattern as the W0.5.D2 round-1 finding 1.1 (godoc drift behind FF reconciliation).

### Family-by-family attack walk

#### B1 — Test-coverage attacks

- **Single-rule self-cycle.** Covered by pre-existing `TestLoadSelfCycleSingleRule` (load_test.go:714-733; GREEN). Also exercised inline by `TestLoadValidatesChildRuleCyclesUnifiedGraph` row 3 ("blocked_by-only cycle rejected (parent->child acyclic)") which uses a `BlockedByParent=true` self-loop and asserts the cycle path renders as `build -> build` plus the edge-label bracket. REFUTED.
- **2-cycle (parent→child).** Covered by fixture `invalid_child_rules_cycle.toml` + row 1 of the unified-graph test. REFUTED.
- **2-cycle (blocked_by-only).** Covered by `invalid_child_rules_blocked_by_cycle.toml` + row 2. The fixture's coupled-graph reality is documented in the fixture comment (lines 11-20) and in the test's row-2 `wantNoSubstr` rationale (lines 2410-2418): today's schema couples the two edge sets, so the parent→child detection wins the race; the blocked_by-only path is exercised by row 3's self-loop. REFUTED.
- **3-cycle.** Not directly tested as a fixture; the DFS algorithm is standard colored-DFS whose N-cycle correctness is invariant to N once the 2-cycle base case + recursion handle. The `TestLoadValidatesChildRuleCyclesDeterministicRootOrder` test runs on a 2-cycle plus an isolated acyclic root, so the `for _, root := range roots` outer loop's lex-ordering is exercised, and the inner adjacency walk that closes the cycle is exercised on the 2-cycle. Adding a 3-cycle row would not catch a class of bugs the 2-cycle row misses given the algorithmic structure (standard back-edge detection on directed graphs). Low-value gap, not a counterexample. REFUTED.
- **Mixed parent→child + blocked_by cycle in same template.** Today's coupled-graph schema means every blocked_by edge is mirrored in parent→child via the same rule, so no rule combination produces a parent→child-only path that closes via blocked_by. The unified DFS contract still demands BOTH edge sets be walked; the implementation walks both (load.go:665-670). Forward-looking when the schema decouples, the fixture infrastructure is in place. REFUTED.
- **Deeply nested kind chain (>5 hops).** Out of D3's scope — D4 lands the recursion-depth bound. D3's cycle detector is correct on chains of any length up to a cycle's closure. The DFS uses recursion (not iteration) so a pathological 10000-deep chain could hit Go's goroutine stack limit. For load-time template validation with realistic kind counts (closed 12-enum), this is not a concern. REFUTED.

Family verdict: REFUTED.

#### B2 — Contract-preservation attacks

- **`formatCyclePath` rendering (closure preserved).** A self-cycle `build → build` renders as `"build -> build -> build"` (closure appended after `cyclePath` already starts with closure). Pre-existing `TestLoadSelfCycleSingleRule` asserts `strings.Contains(err.Error(), "build -> build")` which matches both the pre-D3 and post-D3 rendering — back-compat preserved per the worklog's "Closure-rendering quirk preserved for back-compat" note (line 142). Verified by running `mage testFunc ./internal/templates TestLoadSelfCycleSingleRule` GREEN. REFUTED.
- **Determinism across Go versions / OSes.** The shared helper `dfsDetectCycle` builds `roots []string` from `for k := range graph` (non-deterministic), then `sort.Strings(roots)` (deterministic). Inner adjacency-list walk iterates `graph[node]` slice (deterministic, since the slice is built by iterating the input rules slice in deterministic order). The 20-iteration loop in `TestLoadValidatesChildRuleCyclesDeterministicRootOrder` (load_test.go:2562-2578) catches any non-determinism that would manifest only some-of-the-time. REFUTED.
- **Cycle-path rendering for non-trivial cycles.** Three-node cycle `A → B → C → A`: the DFS visits A (gray), then B (gray on stack `[A, B]`), then C (gray on stack `[A, B, C]`), then `next = A` is colorGray → `resultPath = stack ++ [A] = [A, B, C, A]`. `formatCyclePath` finds first `A` at idx 0, renders `A -> B -> C -> A`. Correct. REFUTED.

Family verdict: REFUTED.

#### B3 — Hidden-coupling attacks

- **`dfsDetectCycle[K ~string]` helper state leak across calls.** Helper closes over `color map[K]int`, `resultPath []K`, and the recursive `dfs` closure, ALL declared inside the function body. No package-level state, no global maps. Two parallel calls (parent→child graph then blocked_by graph) instantiate fresh `color` and `resultPath` per call — verified at load.go:705-707. REFUTED.
- **Recursion-stack depth.** Goroutine starts with 8 KB stack growing dynamically up to ~1 GB by default. Practical limit on the closed 12-enum: at most 12 distinct kinds → max recursion depth 12 → trivially safe. REFUTED.
- **Path-clone safety on cycle detection.** load.go:715: `resultPath = append(append([]K{}, stack...), next)` — explicit `append([]K{}, stack...)` clones `stack` so the returned path does NOT alias the recursion-stack slice. Verified; later mutations to the recursion stack on retreat cannot mutate the captured cycle path. REFUTED.

Family verdict: REFUTED.

#### B4 — YAGNI / scope-creep (PRIMARY FOCUS)

**Was the helper extraction necessary, or could the existing DFS have been extended in place?** The existing `validateChildRuleCycles` ran a single DFS over a single graph. D3 needs to walk TWO graphs. Extending in place would require either (a) inlining the colored-DFS loop twice (~50 LOC duplicated), or (b) hoisting the inner DFS into a closure parameterized by graph reference. Option (a) duplicates the colored-DFS pattern that Drop 3 finding 5.B.4 already specified should be preserved as a single pattern. Option (b) is structurally equivalent to extracting `dfsDetectCycle` but without the type parameter — effectively a private helper without generic reuse. PLAN.md L207 explicitly mandates the helper extraction: "builder extracts a shared private helper rather than copy-pasting." D4 (recursion-depth) and D5 (blocked_by acyclicity) reuse the same helper, so the extraction is structurally required by the PLAN. REFUTED — extraction is justified, not scope creep.

**Was `[K ~string]` strictly needed or is `[K comparable]` + manual sort projection cleaner?** PLAN.md acceptance bullet 6 (line 128) literally specifies `dfsDetectCycle[K comparable](graph map[K][]K) (cyclePath []K, found bool)`. Builder deviated to `[K ~string]`. Two questions: (i) does deviation lose generic flexibility for D4/D5 reuse? (ii) does deviation lose generic flexibility for any plausible future caller?

(i) D4 walks the parent→child kind graph (`map[domain.Kind][]domain.Kind`, `domain.Kind = string`). D5 walks the blocked_by kind graph (same shape). Both reuse cases satisfy `~string` trivially; deviation does NOT lose D4/D5 reuse. (ii) The acceptance bullet's `[K comparable]` is mutually inconsistent with the same bullet's "iterates root-set in sorted-key order" demand: `comparable` does not support ordering, so a `[K comparable]` helper would either need a caller-supplied `less` function, a `[K cmp.Ordered]` constraint, or projection-from-string-and-back inside the helper. Builder picked the third option's cleanest dual: narrow to `~string` and let the helper sort internally. The acceptance text was an internally-inconsistent shape; the builder converged the spec rather than diverging. Future callers keying by non-string types (struct, int) would force re-parameterization — but the closed cascade kind enum is `string`-typed and no proximate non-string-keyed graph is in scope.

REFUTED — `[K ~string]` is a justified design refinement, not scope creep. The acceptance text contradicted itself; builder picked the smaller-diff convergence.

Family verdict: REFUTED. Both design refinements justified.

#### B5 — Spec-compliance attacks (PRIMARY FOCUS)

- **FF3: "Cycle-DFS shared helper iterates root-set in sorted-key order. D3 lands the helper; D4/D5 call into it."** Helper landed at load.go:695-742, `func dfsDetectCycle[K ~string]`. `git grep "dfsDetectCycle" -- '*.go'` confirms three call sites in `validateChildRuleCycles` (D3) — no D4/D5 sites yet because those droplets are state `todo`. Per FF3 chain language: "D3 lands the helper; D4/D5 call into it" — D3 ships the helper as a reusable private function with `~string` constraint covering both D4 (recursion-depth) and D5 (blocked_by acyclicity) reuse cases. REFUTED.
- **D3 actually lands the helper (not just inlined code).** Verified — `dfsDetectCycle` is a separate function with its own godoc (load.go:674-694), not inlined. REFUTED.
- **Determinism test catches non-determinism (20 iterations sufficient?).** Go's `for k := range map` iterates in randomized order per range; on a 2-element map with 2 distinct iteration orders, each iteration has p=0.5 of order A, p=0.5 of order B. Without sorted-root iteration, 20 iterations would have probability `(1/2)^19` of all-same-order ≈ 1.9e-6 — i.e. essentially every 20-iteration run would catch the non-determinism. With the sort.Strings fix in place, the test pins the lex-min cycle path and 20 iterations is overkill — even 2 would suffice. REFUTED — 20 iterations is sufficient (and conservatively so).
- **Sentinel `ErrTemplateCycle` reused as planner mandated.** PLAN.md line 139 mandates "reuse `ErrTemplateCycle`; do NOT introduce a separate `ErrTemplateBlockedByCycle`." Verified at load.go:289 — single sentinel `ErrTemplateCycle`. No new sentinel introduced for blocked_by case. REFUTED.
- **Sentinel godoc drift (FINDING 1.1).** As noted above: the sentinel's pithy one-liner (load.go:289-291) still says "parent → child kind graph" in the singular even though the validator now walks the unified graph. CONFIRMED but low-severity.

Family verdict: REFUTED on substance, 1 low-severity NIT on sentinel godoc.

#### B6 — Shipped-but-not-wired

- **`dfsDetectCycle` helper called by D3's two-graph walk.** Verified — load.go:665 + 668 are live call sites in `validateChildRuleCycles`. Helper is reachable from production via `LoadWithOptions` → `validateChildRuleCycles` → `dfsDetectCycle`. REFUTED.
- **D4/D5 wiring deferred to those droplets.** PLAN.md FF3 chain language explicitly defers; D4 and D5 are state `todo` and will land helper reuse in their own rounds. Acceptable — D3 ships a reusable helper; downstream droplets consume it. REFUTED.
- **`formatCyclePath` generalization shipped & wired.** Generalized from `func(stack []domain.Kind, closure domain.Kind) string` to `func[K ~string](cyclePath []K) string` (load.go:756). Two live call sites at load.go:666 + 669. D4/D5 reuse parametrically — both will key by `domain.Kind` (a `~string`) so the renderer covers both. REFUTED.

Family verdict: REFUTED.

#### B7 — Prompt-injection

Pre-team-feature; per `feedback_prompt_injection_team.md` this family is dormant until team functionality lands. No action-item content is attacker-controllable in the W0.5.D3 scope. EXHAUSTED.

Family verdict: EXHAUSTED.

### Required gate runs (executed)

- **`mage testPkg ./internal/templates`** — GREEN. 418/418 tests pass.
- **`mage testFunc ./internal/templates "TestLoadValidatesChildRuleCycles.*"`** — GREEN. 7/7 sub-tests pass (5 rows of `TestLoadValidatesChildRuleCyclesUnifiedGraph` + 1 `TestLoadValidatesChildRuleCyclesDeterministicRootOrder` + the parent test funcs).
- **`mage testFunc ./internal/templates TestLoadSelfCycleSingleRule`** — GREEN. 1/1 (regression — pre-existing self-cycle test still passes after the unified-graph extension).
- **`mage testFunc ./internal/templates TestLoadRejectionTable`** — GREEN. 9/9 (regression — pre-existing rejection table including the cycle row still passes).
- **`git grep "dfsDetectCycle" -- '*.go'`** — 5 hits inside `internal/templates/load.go` (3 call sites + 1 godoc reference + 1 definition) + 1 hit in `load_test.go` (godoc reference in the unified-graph test). Helper scope is package-private (lowercase first letter); no external consumer. Verified.

### Summary

**Verdict: pass.**

**Counterexample count:** 1 (low severity — sentinel godoc drift on `ErrTemplateCycle`).

| Family | Result |
|---|---|
| B1 test-coverage | REFUTED |
| B2 contract-preservation | REFUTED |
| B3 hidden-coupling | REFUTED |
| B4 yagni / scope-creep (PRIMARY) | REFUTED |
| B5 spec-compliance (PRIMARY) | REFUTED (1 sentinel-godoc drift NIT) |
| B6 shipped-but-not-wired | REFUTED |
| B7 prompt-injection | EXHAUSTED |

**Explicit verdict on the two design refinements:**

1. **`[K ~string]` instead of literal `[K comparable]`: JUSTIFIED.** PLAN.md acceptance bullet 6 simultaneously demanded `[K comparable]` AND "iterates root-set in sorted-key order" — these are mutually inconsistent because `comparable` does not support ordering. The builder converged the spec to the smallest constraint that lets the helper self-sort (`~string`); D4/D5's reuse cases (both keyed by `domain.Kind`, which is `string`-typed) are unaffected. Future callers keying by non-string types would force re-parameterization, but no proximate non-string-keyed graph is in scope. NOT scope creep — convergence of an internally inconsistent acceptance bullet.

2. **Two-graph walk instead of unified merged-edge graph: JUSTIFIED.** A literal merged-edge graph would falsely flag every well-formed `BlockedByParent=true` rule as a 2-cycle (parent→child + child→parent edges of a single rule combine into a degree-2 cycle in the merged graph). This would over-detect — every QA-twin rule, every commit-cadence rule, every standard cascade rule trips the validator. The two-graph approach preserves the semantic distinction between auto-create cycles (infinite chain) and blocked_by cycles (runtime deadlock) while reporting WHICH edge set produced the cycle. The PLAN's "unified DFS" wording is internally consistent with the two-graph implementation IF "unified" is read at the caller-level (one validator, two passes) rather than at the graph-edge level. Builder's interpretation is the only one that doesn't over-flag. NOT scope creep — semantic correctness convergence.

Build round 1 lands the cycle detector with the unified-graph extension, the shared `dfsDetectCycle[K ~string]` helper, the generalized `formatCyclePath[K ~string]` renderer, plus a 5-row table-driven test exercising the parent→child cycle, the blocked_by cycle, the self-loop edge-label flexibility, the happy-path baseline, and an acyclic blocked_by chain — paired with a 20-iteration determinism test pinning the sorted-root contract. Both gates green: `mage testPkg ./internal/templates` (418 tests pass) and per-target `mage testFunc` runs (8 sub-tests pass across the two new tests + 2 regression tests).

**Optional follow-up (NIT, not gating):** apply the sentinel godoc fix from finding 1.1 in a follow-up commit (single-line edit to `internal/templates/load.go:289-291`). Not gating because the runtime behavior is correct, the validator's own godoc (load.go:620-650) accurately describes the unified-graph behavior, and the wrapped error message names both edge types — the drift is contained to the sentinel's pithy one-line summary only.

### Hylla Feedback

N/A — droplet touched only Go files inside `internal/templates/` plus testdata fixtures (TOML) plus the workflow MDs. All Go reads were against `load.go` + `load_test.go` in the uncommitted modified working set per `git status` (load.go was modified through W0.5.D1 / D2 / D3 across the session). Hylla's index is stale for those files until the drop-end reingest. Direct `Read` + `git grep` against the working tree was the correct evidence path. No Hylla queries attempted on the in-flight files; nothing to log.

## Droplet 4c.6.W0.5.D4 — Round 1

**Date:** 2026-05-09
**Reviewer:** go-qa-falsification-agent (build-QA-falsification, parent.kind=build)
**Scope:** child_rules recursion-depth bound (commit `38760ee`). Round 1 attack focus: the three builder design refinements — depth=edges-not-nodes, `dfsDetectCycle` not directly reused, new `formatChainPath` helper — plus the standard 7-family attack pass.

### Counterexamples

None CONFIRMED. All seven attack families exhausted; details below. One low-severity NIT logged under B5 (path-length over-allocation) but it is non-functional and is not a counterexample.

#### B1 — Test-coverage attacks

Attempted attacks on `TestLoadValidatesChildRuleRecursionDepth` (`load_test.go:2607`) + `TestLoadValidatesChildRuleRecursionDepthRunsAfterCycleDetection` (`load_test.go:2740`):

- **Depth-5 boundary passes (row 1).** Inline TOML `closeout → research → discussion → refinement → human-verify → commit` (5 edges, 6 nodes). Logical trace through `compute`: commit→0 (leaf), human-verify→1, refinement→2, discussion→3, research→4, closeout→5; `5 <= 5` → continues; no other root produces a path longer than 5; returns nil. Verified by GREEN. REFUTED.
- **Depth-6 trip (row 2, fixture).** `closeout → … → commit → plan` (6 edges, 7 nodes). compute("closeout") = 6; `6 > 5` → diagnostic walks successorOnLongest from "closeout"; path = [closeout, research, discussion, refinement, human-verify, commit, plan]; renders via `formatChainPath` joined by " -> "; matches all three required substrings (the full path string, `"depth 6"`, `"max 5"`). Verified by GREEN. REFUTED.
- **Diamond shapes (memoization correctness).** Not directly tested as a fixture, but the algorithm is correct by construction: `compute(node)` reads `depthFrom[node]` cache before any per-call work (line 894), so a shared descendant `D` reached via both `A→B→D` and `A→C→D` resolves once. The cached `depthFrom[D]` already encodes the longest path from `D`, so the second visit observes the same value the first computed. `successorOnLongest[D]` was set on the first computation; the second visit does not overwrite it (the cache hit returns before the `successorOnLongest` write). Tied children pick the FIRST in `graph[node]` iteration order (strict `>` at line 911), and `graph[node]` is built by appending in TOML decode order (deterministic per pelletier/go-toml/v2). REFUTED algorithmically; no counterexample constructible.
- **Disjoint roots.** Test row 4 (`single root-only kind passes (depth 0)`) constructs `closeout → research` and `refinement → human-verify` — two disjoint single-edge components. compute walks both in sorted-root order; both yield depth 1; both pass. REFUTED.
- **Empty graph.** Row 3 uses `valid_minimal.toml` (empty `[[child_rules]]`); validator returns nil at line 873 early-return. REFUTED.
- **Very-deep chain (>20).** Not directly fixture-tested. With 12 closed kinds and the cascade vocabulary, a chain >12 cannot exist without revisiting a kind, which would re-enter the chain and trip D3's cycle detector first. The depth bound (5) is reached well before the closed-enum exhaustion limit. The algorithm scales linearly (memoised DFS is O(V+E)) and recursion depth is bounded by the closed enum cardinality (12) — no stack-overflow risk. REFUTED.
- **Self-cycle pre-rejection.** A self-loop `build → build` is a cycle; D3 rejects with `ErrTemplateCycle` before D4 runs. The chain-order regression guard (`TestLoadValidatesChildRuleRecursionDepthRunsAfterCycleDetection`) pins this contract by asserting `errors.Is(err, ErrTemplateCycle)` AND NOT `errors.Is(err, ErrChildRuleRecursionTooDeep)` on the cyclic input `build → plan → build`. REFUTED.

Family verdict: REFUTED.

#### B2 — Contract-preservation attacks

- **Chain-order contract: D4 strictly after D3.** Verified at `load.go:245-250` — `validateChildRuleCycles` runs at line 245, `validateChildRuleRecursionDepth` runs at line 248, no validator interposes. The chain-order test at `load_test.go:2740-2768` asserts the contract empirically — a misorder that runs D4 before D3 would either infinite-loop the depth DFS or surface `ErrChildRuleRecursionTooDeep` on a cyclic input, both caught by the test's `errors.Is(err, ErrTemplateCycle)` AND `!errors.Is(err, ErrChildRuleRecursionTooDeep)` assertions. REFUTED.
- **`Load` godoc validator-chain table updated to include 4(c').** Verified at `load.go:109-114` — the table calls out `validateChildRuleRecursionDepth` between `c. validateChildRuleCycles` and `d. validateRequiredChildRules`, with the chain-order rationale ("Runs immediately after the cycle detector so cyclic graphs are rejected with the better diagnostic") explicit. REFUTED.
- **Sentinel `ErrChildRuleRecursionTooDeep` follows the established `var ErrXxx = errors.New(...)` pattern.** Verified at `load.go:483-513` — sentinel block, godoc with depth semantics + bound rationale + W0.5 plan FF1 disclosure. REFUTED.
- **Existing tests untouched.** `mage testPkg ./internal/templates` reports 424/424 GREEN; 18 new tests added since the W0.5.D3 round (round-3 baseline was 418, current is 424). The new validator does not regress any prior test. REFUTED.

Family verdict: REFUTED.

#### B3 — Hidden-coupling attacks

- **Memoization map state isolation across calls.** `depthFrom`, `successorOnLongest`, and `visited` are all declared INSIDE `validateChildRuleRecursionDepth` (lines 888-890); each `Load` call instantiates fresh maps. No package-level state. Two parallel `Load` calls cannot leak depth values across each other. REFUTED.
- **`successorOnLongest` write/read ordering.** The `compute` function writes `successorOnLongest[node] = bestChild` at line 922 ONLY when `bestDepth >= 0` (i.e. the node has at least one out-edge). Leaves do NOT write to `successorOnLongest`. The diagnostic walk at line 942-949 uses `next, ok := successorOnLongest[node]` — when `ok == false` (leaf node), the loop breaks. So the path rendering correctly terminates at leaves. The depth-6 fixture's terminal node `plan` is a leaf (no out-edge); the path rendering correctly ends with `plan` and renders the full 7-node chain. REFUTED.
- **`visited` map purpose.** Defense-in-depth guard (line 897-904) — sets `visited[node] = true` BEFORE the recursive descent so a cycle (which D3 should have rejected) treats the back-edge target as a leaf rather than infinite-looping. The `visited` map is never read after `compute(node)` returns because `depthFrom[node]` is the cache the next entry sees. REFUTED — the `visited` map is correctly contained to one call's recursion.
- **Closure capture in `var compute func(node domain.Kind) int`.** The closure captures `depthFrom`, `successorOnLongest`, `visited`, `graph`, and `compute` itself. All are call-local. No goroutine spawn; no concurrent access. Race-free. REFUTED.
- **Recursion stack depth.** Closed 12-kind enum bounds the recursion at 12. Default Go goroutine stack (8 KB initial, grows to ~1 GB) trivially covers that. REFUTED.

Family verdict: REFUTED.

#### B4 — YAGNI / scope-creep — PRIMARY FOCUS (3 design refinements)

Reviewer attacks the three design refinements the spawn prompt called out.

**Refinement 1 — Depth=edges, not nodes.** L2 PLAN ContextBlock (`decision (normal): default depth bound is 5 edges; configurable post-MVP via [tillsyn] recursion_depth_max`) is verbatim. Off-by-one risk probed: depth-5 chain has 6 nodes, depth-6 chain has 7 nodes. The constant's doc-comment (load.go:516-527) says "counted in edges from any root" explicitly; `ErrChildRuleRecursionTooDeep`'s godoc (load.go:483-513) repeats the same definition. Diagnostic message format (load.go:950-957) prints `depth N` matching the edge count from the recursive count `bestDepth + 1` at line 921 (which counts edges, not nodes — leaves return 0 and each parent adds 1 edge). Verified by tests row 1 (depth-5 boundary passes) + row 2 (depth-6 trips). The rendering chain has `depth+1` nodes (`path` capacity at line 940) — correct nodes-vs-edges accounting. JUSTIFIED — the edges semantics is internally consistent and externally verifiable. REFUTED.

**Refinement 2 — `dfsDetectCycle` not directly reused.** L2 PLAN line 199 ("Helper extraction `dfsDetectCycle` was NOT needed for D4") is the builder's documented reasoning. Reviewer probes: could the existing helper structurally cover longest-path? `dfsDetectCycle` returns `(cyclePath []K, found bool)` and uses colored-DFS with white/gray/black state. Longest-path needs `depth int` + `successorOnLongest map[K]K`. The two DFS shapes have:

- **Different return types.** Cycle returns a slice; longest-path returns an int + builds a successor map.
- **Different state.** Cycle uses 3-state coloring (white/gray/black) for back-edge detection. Longest-path uses memoization (cache hit returns immediately).
- **Different traversal order semantics.** Cycle returns on FIRST back-edge (early termination); longest-path walks every reachable node before resolving the depth (no early termination per node — must visit all out-edges to pick the max).

Forcing reuse via a single helper would either (a) bloat the helper's signature with an `aggregator func(...)` callback that handles both depth-tracking and back-edge detection, or (b) inline-merge the two algorithms into a unified DFS that's harder to reason about. Neither serves the codebase. The L2 PLAN's "graph constructed by D3 is reused by D4 — D4 does NOT re-build it" wording (acceptance bullet 6) was structurally optimistic: D3 builds the graph inside `validateChildRuleCycles` as a local variable; reusing that variable in D4 would require either hoisting the graph build into a shared helper (4 LOC of code worth its own refactor) or passing the graph through `LoadWithOptions`'s call chain (wider blast radius). Builder picked a third path: D4 builds its OWN graph (3 LOC) and inherits the iteration discipline (sort.Strings root order at line 930 mirrors `dfsDetectCycle`'s line 781-785 contract verbatim). The "spirit of reuse" — same iteration order for reproducible diagnostics — is honored without forcing structural reuse of an algorithmically distinct helper. JUSTIFIED. REFUTED.

**Refinement 3 — New `formatChainPath` helper instead of reusing `formatCyclePath`.** L2 PLAN ContextBlock said "warning (normal): `formatCyclePath` reuse (or near-clone) for D4's path rendering keeps the error UX consistent." The L2 PLAN's "near-clone" wording explicitly permits a separate helper. Builder discovered DURING RED→GREEN (worklog line 184-185 documents the test failure) that reusing `formatCyclePath` literally produces the wrong output: `formatCyclePath` strips prefix nodes by finding the first occurrence of the closure (last) element and rendering from there (load.go:815-820). On a non-cyclic chain `[closeout, research, ..., plan]`, the last element `plan` appears only once at the end, so `startIdx` lands on `plan` itself and the rendering becomes just `"plan"` — losing every prefix node. This is a real bug that would have shipped if the renderer were forcibly reused. The new `formatChainPath` (load.go:973-982) is 9 LOC, mirrors `formatCyclePath`'s `~string` constraint + " -> " separator, and avoids the closure-stripping behavior. The diff is small; the alternative (parameterize `formatCyclePath` with a `treatLastAsClosure bool` flag) would have added complexity to a helper used by 2 cycle call sites + 1 chain call site for a 50% conditional split — strictly worse than two clean helpers. JUSTIFIED. REFUTED.

**Sub-attack: was the path-rendering chain length over-allocated?** Line 940: `path := make([]domain.Kind, 0, depth+1)`. For depth=6 the chain has 7 nodes — capacity 7 is exact. For depth=5 the validator does not enter this branch (the `depth <= childRuleRecursionDepthMax` guard at line 935 returns), so the allocation never fires. For depth=N>5 the chain length is exactly `N+1`; capacity `depth+1` is exact. No over-allocation. REFUTED.

**Sub-attack: was `validateChildRuleRecursionDepth` shipped behind a flag instead of wired in the chain?** No — `git grep validateChildRuleRecursionDepth` shows the call site at `load.go:248` inside `LoadWithOptions`. Verified shipped + wired. REFUTED.

Family verdict: REFUTED. All three design refinements justified; no scope creep.

#### B5 — Spec-compliance attacks

L2 acceptance bullets (PLAN.md lines 156-165) mapped to test/code coverage:

| Bullet | Coverage | Status |
|---|---|---|
| 1. New validator `validateChildRuleRecursionDepth` walks parent→child graph; rejects when depth > `childRuleRecursionDepthMax = 5` | `load.go:871-960` + wired at `load.go:248` | satisfied |
| 2. Constant `childRuleRecursionDepthMax = 5` documented as default per `SKETCH.md § 26.W0.5`; configurable post-MVP | `load.go:516-527` | satisfied |
| 3. New sentinel `ErrChildRuleRecursionTooDeep` added to sentinel block | `load.go:483-513` | satisfied |
| 4. Wrapped error names offending kind, observed depth, bound, path-from-root | `load.go:950-957` (`%w: kind %q reaches depth %d (max %d): %s`) | satisfied |
| 5. New malformed fixture `invalid_child_rules_too_deep.toml` rejects with sentinel + path rendering `"closeout -> research -> discussion -> refinement -> human-verify -> commit -> plan"` | row 2 of test (load_test.go:2654-2664) asserts all three substrings; fixture is 6-rule chain | satisfied |
| 6. Graph constructed by D3 is reused (D4 does NOT re-build) | NOT structurally reused — see B4 Refinement 2; D4 builds its own 3-LOC graph and inherits iteration discipline. **Documented deviation** in the worklog (line 199); justified algorithmically | satisfied (intent honored, structural reuse rejected with rationale) |
| 7. Cycle vs depth ordering: D3 fails first on cyclic input, D4 never runs | pinned by `TestLoadValidatesChildRuleRecursionDepthRunsAfterCycleDetection` (load_test.go:2740) | satisfied |
| 8. Table-driven test w/ 4 rows (depth 5 pass, depth 6 fail, empty pass, multi-root pass) | `TestLoadValidatesChildRuleRecursionDepth` rows 1-4 | satisfied |
| 9. `mage test-func` RED→GREEN | BUILDER_WORKLOG round-1 documents RED via build-error level + commented-out wire-up | satisfied |
| 10. `mage test-pkg ./internal/templates` clean | re-verified by reviewer (424 tests pass) | satisfied |

Acceptance bullet 6 was deliberately reinterpreted by the builder — see B4 Refinement 2 for the structural reasoning. The intent ("D4 inherits D3's iteration discipline") is honored via `sort.Strings` root order in both validators; the literal "graph variable reuse" is rejected because the graph build is 3 LOC and hoisting it would force either a hoisted package-internal helper or a state-passing rewrite. The deviation is documented at WORKLOG line 199 and the test contract is unchanged.

**NIT-only sub-attack on B5: path-rendering allocation.** The path `make([]domain.Kind, 0, depth+1)` is exactly right-sized for the chain; no over-allocation. Verified at line 940. NIT-only sub-attack found nothing.

Family verdict: REFUTED.

#### B6 — Shipped-but-not-wired attacks

- **`validateChildRuleRecursionDepth` wire-up.** Verified at `load.go:248` inside `LoadWithOptions` chain — between `validateChildRuleCycles` (line 245) and `validateRequiredChildRules` (line 251). Correct insertion point per L2 PLAN (which mandated "after the cycle validator").
- **Test exercises full Load path, not validator in isolation.** Both `TestLoadValidatesChildRuleRecursionDepth` and the chain-order regression guard call `Load(strings.NewReader(src))` (load_test.go:2705 + 2758) — runs the FULL validator chain. Not stubbed.
- **Fixture exists and is read at test time.** `mustReadTestdata(t, "invalid_child_rules_too_deep.toml")` (test:2701 → fixture file at `testdata/invalid_child_rules_too_deep.toml`, 50 lines on disk). RED-confirmation in BUILDER_WORKLOG line 184-188 commented the wire-up and observed expected failures.
- **Sentinel + constant + helpers all reachable from production.** `ErrChildRuleRecursionTooDeep` returned via `fmt.Errorf("%w: ...", ...)` at line 950-957. `childRuleRecursionDepthMax` consumed at line 935 + 955. `formatChainPath` consumed at line 956. All three are live; no dead code.

Re-verified: `mage testPkg ./internal/templates` → 424 tests pass; `mage testFunc ./internal/templates TestLoadValidatesChildRuleRecursionDepth` → 5 sub-tests pass; `mage testFunc ./internal/templates TestLoadValidatesChildRuleRecursionDepthRunsAfterCycleDetection` → 1 test passes. `git grep validateChildRuleRecursionDepth -- '*.go'` returns 6 hits (1 godoc reference in chain table, 1 wire-up call site, 1 sentinel godoc, 1 validator definition, 2 test references) — scope contained.

Family verdict: REFUTED.

#### B7 — Prompt-injection attacks

Pre-team-feature; per `feedback_prompt_injection_team.md` this family is dormant until team functionality lands. No action-item content is attacker-controllable in the W0.5.D4 scope. EXHAUSTED.

Family verdict: EXHAUSTED.

### Required gate runs (executed)

- **`mage testPkg ./internal/templates`** — GREEN. 424/424 tests pass.
- **`mage testFunc ./internal/templates TestLoadValidatesChildRuleRecursionDepth`** — GREEN. 5/5 sub-tests pass (1 parent + 4 rows).
- **`mage testFunc ./internal/templates TestLoadValidatesChildRuleRecursionDepthRunsAfterCycleDetection`** — GREEN. 1/1.
- **`git grep "validateChildRuleRecursionDepth"`** — 6 hits inside `internal/templates/`: 1 godoc chain-table reference (load.go:109), 1 wire-up call site (load.go:248), 1 sentinel godoc (load.go:483), 1 validator definition (load.go:871), 2 test references (load_test.go:2586 + 2589). Scope contained to one package; no cross-package consumer; no external API surface.

### Summary

**Verdict: pass.**

**Counterexample count:** 0

| Family | Result |
|---|---|
| B1 test-coverage | REFUTED |
| B2 contract-preservation | REFUTED |
| B3 hidden-coupling | REFUTED |
| B4 yagni / scope-creep (PRIMARY) | REFUTED |
| B5 spec-compliance | REFUTED (1 documented acceptance-bullet reinterpretation, justified at WORKLOG line 199) |
| B6 shipped-but-not-wired | REFUTED |
| B7 prompt-injection | EXHAUSTED |

**Explicit verdict on the three design refinements:**

1. **Depth=edges (not nodes): JUSTIFIED.** The constant's doc-comment (`load.go:516-527`) and the sentinel's godoc (`load.go:483-513`) both explicitly state "edges from any root." Tests row 1 (depth 5 = 5 edges = 6 nodes passes) + row 2 (depth 6 = 6 edges = 7 nodes fails) pin the boundary. Diagnostic message format prints `depth N` matching edge count exactly. No off-by-one risk — `bestDepth + 1` (line 921) increments the edge count per parent step from leaf-zero base.

2. **`dfsDetectCycle` not directly reused: JUSTIFIED.** Cycle detection and longest-path are algorithmically distinct DFS shapes — different return types (slice vs int+map), different state (3-state colors vs memoization), different early-termination semantics (back-edge return vs full-subtree resolution). Forcing reuse via shared helper would either bloat the signature with a callback aggregator or merge the algorithms into a harder-to-reason-about unified DFS. Builder kept the iteration discipline (sort.Strings root order) which is the spirit of reuse the L2 PLAN had in mind; rejected the literal "graph variable reuse" wording because the graph build is 3 LOC and hoisting forces wider blast radius. Documented deviation at WORKLOG line 199.

3. **New `formatChainPath` helper: JUSTIFIED.** Reusing `formatCyclePath` literally produces wrong output on non-cyclic chains — the closure-stripping logic at `load.go:815-820` would render a depth-6 chain as just `"plan"` (the last element). Builder caught this during RED→GREEN (WORKLOG line 184-185 documents the test failure that triggered the renderer split). The L2 PLAN's "near-clone" wording explicitly permits the split. The new helper is 9 LOC, mirrors the cycle renderer's `~string` constraint + " -> " separator, and avoids the closure-stripping bug. The alternative (parameterize `formatCyclePath` with a `treatLastAsClosure bool` flag) is strictly worse — splits a 2-cycle-call-site + 1-chain-call-site helper into a conditional that obscures both call sites.

Build round 1 lands the recursion-depth bound at the correct chain position with edges-semantics depth measurement, a memoised DAG longest-path DFS with `successorOnLongest` chain-walk-back for diagnostic rendering, the new `formatChainPath` renderer, plus a 4-row table-driven test exercising depth-5 boundary, depth-6 fixture, empty graph, and multi-root, paired with a chain-order regression guard pinning the D3-before-D4 contract. Both gates green: `mage testPkg ./internal/templates` (424 tests pass) and per-target `mage testFunc` runs (6 sub-tests pass across the two new tests).

### Hylla Feedback

N/A — droplet touched only Go files inside `internal/templates/` plus a new TOML fixture plus the workflow MDs. All Go reads were against `load.go` + `load_test.go` at HEAD commit `38760ee` (the D4 commit) plus uncommitted `git status` deltas elsewhere in the tree. Hylla's index is stale for `internal/templates/load.go` until the drop-end reingest (load.go has been modified through W0.5.D1 / D2 / D3 / D4 across the day, none of which Hylla has yet seen). Direct `Read` + `git grep` against the working tree was the correct evidence path. No Hylla queries attempted on the in-flight files; nothing to log.

## Droplet 4c.6.W0.5.D5 — Round 1

**Date:** 2026-05-09
**Reviewer:** go-qa-falsification-agent (build-QA-falsification, parent.kind=build)
**Scope:** `validateBlockedByAcyclicity` + `buildBlockedByGraph` + `LoadOptions.BlockedByGraphFn` test seam + new sentinel `ErrTemplateBlockedByCycle` + new fixture `invalid_blocked_by_acyclicity.toml` + two new tests (`TestLoadValidatesBlockedByAcyclicity` 4-row + `TestLoadValidatesBlockedByAcyclicityRunsAfterChildRuleCycles` chain-order regression guard).

### Counterexamples

None CONFIRMED. All seven attack families exhausted; details below.

#### B1 — test-coverage attacks

Attempted attacks on `TestLoadValidatesBlockedByAcyclicity` (`internal/templates/load_test.go:2817-2909`):

- **Self-cycle (A→A).** Not directly tested by D5. The validator delegates cycle detection entirely to the shared generic `dfsDetectCycle[K ~string]` helper (load.go:819-866). D3's `TestLoadSelfCycleSingleRule` (existing) and `TestLoadValidatesChildRuleCyclesUnifiedGraph`'s self-cycle row (load_test.go:2381+) exercise the helper's self-cycle path. D5 inherits self-cycle correctness via the shared helper. The synthetic graph at row 2 (`build → plan, plan → build`) is a 2-cycle which is the smallest distinct shape; coverage is contractually transitive through the helper. NIT-only, REFUTED as counterexample.
- **3-cycle (A→B→C→A).** Same shape-coverage argument as self-cycle: `dfsDetectCycle` is helper-tested via D3's tests; D5 inherits. NIT-only, REFUTED.
- **Empty-graph happy path.** Row 3 ("empty child_rules passes", load_test.go:2862-2867) explicitly covers `len(graph) == 0` via `schema_version = "v1"` only. The validator's defensive early return at load.go:1111 (`if len(graph) == 0 { return nil }`) is exercised. REFUTED.
- **Production walker on real fixture.** Row 1 ("acyclic production graph passes", load_test.go:2845-2849) loads `valid_minimal.toml` with `graphFn: nil`, hitting the production walker `buildBlockedByGraph` which produces 2 child→parent edges (`build-qa-proof → build`, `build-qa-falsification → build`). The DFS confirms acyclicity over a real (non-injected) graph. REFUTED.
- **Single-edge cycle.** The synthetic 2-cycle row IS a single-rule-shape cycle once the producing graph fn is interpreted as "one rule per edge." Smaller-cycle ordering (1-edge self-cycle) routes through the helper's tested branch. REFUTED.
- **`wantNoSubstr` field declared but unused on every row.** Field exists at load_test.go:2842 but no row populates it. Defensive scaffold; no defect. NIT-only, REFUTED.

Family verdict: REFUTED. Coverage is helper-mediated for cycle-shape variations; D5's own rows cover the validator's behavior surface (production walker hit, injected-cycle hit, empty-graph short-circuit, fixture-driven path).

#### B2 — contract-preservation attacks

Chain-order contract pinned by `TestLoadValidatesBlockedByAcyclicityRunsAfterChildRuleCycles` (load_test.go:2932-2944). Verified D3 → D4 → D5 ordering at load.go:278-286:

```
278: validateChildRuleCycles(tpl.ChildRules)             // D3
281: validateChildRuleRecursionDepth(tpl.ChildRules)     // D4
284: validateBlockedByAcyclicity(tpl.ChildRules, opts.BlockedByGraphFn) // D5
```

The regression guard loads `invalid_child_rules_blocked_by_cycle.toml` (D3's coupled-cycle fixture; both rules have `blocked_by_parent = true` so the cycle exists in BOTH the parent→child and the blocked_by graphs) and asserts:

- `errors.Is(err, ErrTemplateCycle)` — D3 fired first.
- `!errors.Is(err, ErrTemplateBlockedByCycle)` — D5 was preempted.

A future refactor that swaps D3 and D5 in the chain order would surface the wrong sentinel for every coupled cycle and break adopter `errors.Is` routing. The regression guard catches the swap loudly. The pre-D4 ordering (depth runs before blocked-by) is a chain-position decision, not a correctness invariant — D4 is acyclicity-prerequisite-bounded so it's unaffected by ordering relative to D5. REFUTED.

Family verdict: REFUTED.

#### B3 — hidden-coupling attacks (test-seam reachability)

`LoadOptions.BlockedByGraphFn` is **exported** (capital `B`). Any caller of the public `LoadWithOptions` API can set it, including library users. Attack line: an adopter's malicious `BlockedByGraphFn` could bypass the production walker.

Probe: searched all `*.go` files in the tree (`rg -n 'BlockedByGraphFn|buildBlockedByGraph'`):

- Production callers: `LoadWithOptions` at load.go:284 (passes `opts.BlockedByGraphFn` through). `LoadDefaultTemplate*` (the embedded-template loaders) construct `LoadOptions{}` with the field unset — defaults to nil. **Zero production sets of `BlockedByGraphFn`.**
- Test callers: `TestLoadValidatesBlockedByAcyclicity` row 2 + row 4 (load_test.go:2882-2884) inject `syntheticCycleFn`.

Semantic analysis: the validator only **rejects** via the seam (it can fail Load with `ErrTemplateBlockedByCycle`). It cannot turn a real cycle into a non-cycle — `dfsDetectCycle` finds the cycle deterministically over whatever graph it walks, and the production-walker output is independent of `BlockedByGraphFn`. The injection ONLY substitutes the graph being walked; it does NOT bypass cycle detection. Worst-case adopter abuse: pass a `BlockedByGraphFn` that always returns a synthetic cycle → adopter's own template fails Load with `ErrTemplateBlockedByCycle`. Self-harming, not data-corrupting.

Two acceptance lenses:

1. **Field naming consistency.** D2's seam `LoadOptions.AgentLookupFn` is also exported (load.go:63). The pattern is consistent; D5 isn't introducing a new exported-seam pattern. Both are documented as "test-only" in their godocs (load.go:65-84 for D5; the AgentLookupFn doc-comment for D2).
2. **Could be unexported.** Lowercasing `blockedByGraphFn` would force every test caller into the same package (already the case — tests live in `internal/templates`). This is a possible refinement; the builder chose exported for symmetry with `AgentLookupFn`. Refinement-grade NIT, not a counterexample.

REFUTED as counterexample. Lensed as refinement-only NIT for future review (consider unexporting both `AgentLookupFn` and `BlockedByGraphFn` together).

Family verdict: REFUTED.

#### B4 — YAGNI / scope-creep attacks (premature extraction)

`buildBlockedByGraph` (load.go:1134-1142) is 7 LOC, called once by `validateBlockedByAcyclicity` (load.go:1109). Inlining is technically possible. Attack line: the extraction is premature — the validator is dead code today (degenerate forest), and the helper exists "just in case."

Counter-evidence at load.go:1120-1133 (helper's doc-comment):

```
// buildBlockedByGraph produces the production kind-level blocked_by graph
// from a [[child_rules]] slice. Each rule whose BlockedByParent is true
// contributes one edge from rule.CreateChildKind to rule.WhenParentKind
// (child→parent: child cannot start until parent terminal-completes).
//
// Today's schema produces a forest — every edge is a child→parent edge and
// parents have no incoming blocked_by edges, so no cycle can exist. The
// helper is its own function rather than inlined into
// validateBlockedByAcyclicity so future schema expansions (e.g. a
// `BlockedByKinds []domain.Kind` field on ChildRule) extend the helper in
// one place rather than threading new edge-construction logic through the
// validator body.
```

Justification surfaces:

- Helper has a documented forward-looking purpose (future `BlockedByKinds` schema axis).
- Helper has a clear single-responsibility (graph construction) vs validator (graph traversal + error wrapping).
- Helper extraction matches D3's pattern (D3's `validateChildRuleCycles` builds two graphs inline; if D3's blocked_by edge-set construction grows, extracting it is the natural refactor — and D5's `buildBlockedByGraph` is exactly that target).

The extraction is YAGNI-questionable but documented + small. Inlining would force any future schema addition to thread new logic through the validator body. The 7-LOC helper is not over-engineered; it's at the "name-the-concept" granularity that Go readers expect. REFUTED.

Family verdict: REFUTED.

#### B5 — spec-compliance attacks (synthetic-fixture pattern)

Attack line: `invalid_blocked_by_acyclicity.toml` is a structurally valid TOML fixture; the rejection comes from the injected `syntheticCycleFn` in row 4 of the test, not from the fixture content. Does this bypass the validator?

Probe: row 4 ("fixture-driven cycle via injection rejected", load_test.go:2868-2877):

```
src:     invalidBlockedByCycle,
graphFn: syntheticCycleFn,
```

The fixture IS loaded through normal `LoadWithOptions` (load_test.go:2882), passing through every other validator (schema_version, strict decode, validateMapKeys, validateChildRuleKinds, validateChildRuleCycles, validateChildRuleRecursionDepth) BEFORE reaching `validateBlockedByAcyclicity`. The fixture's structural validity is a precondition; the cycle rejection comes from the validator + injected graph fn working as designed. The validator IS exercised — the graph it walks is the injected one, but the rejection path through `dfsDetectCycle → formatCyclePath → ErrTemplateBlockedByCycle` is real production code.

Cross-check against the L2 PLAN W0.5.D5 KindPayload + RiskNote (PLAN.md:194):

> If today's `BlockedByParent: true` schema cannot construct a real cycle ... the fixture instead tests D5 against an injected mock graph via a new `LoadOptions.BlockedByGraphFn` ...

The fixture pattern explicitly matches the planned shape. The fixture's doc-comment (testdata/invalid_blocked_by_acyclicity.toml:1-29) honestly discloses the injection-driven rejection AND cites the forward-looking schema-expansion path. Row 4 provides parity with D3/D4's fixture-shaped rows so adopters who grep `testdata/` for the new sentinel see a real fixture file paired with the test row.

REFUTED. Synthetic injection is necessary because today's schema cannot construct a real kind-level cycle (every `BlockedByParent: true` edge runs child→parent, parents have zero incoming blocked_by edges, the production graph is a trivially-acyclic forest), and the disclosure is honest.

Family verdict: REFUTED.

#### B6 — shipped-but-not-wired attacks

Attack line: `validateBlockedByAcyclicity` is wired into the chain but produces nil for every embedded default template — it's vacuous in production today. Is this the "shipped-but-not-wired" anti-pattern (Drop 3 droplet 3.20)?

Probe:

- Validator IS wired at load.go:284. Every Load goes through it.
- Production walker `buildBlockedByGraph` produces a real (non-empty) graph for `valid_minimal.toml` (2 edges per the fixture's 2 QA-twin child_rules, both with `blocked_by_parent = true`). The DFS walks the real graph and confirms acyclicity in O(V+E). The work is real, not a no-op.
- For the embedded `default-go.toml` (the production template with the full chain of QA-twin rules), the production walker produces ~10+ edges, all child→parent, no cycle. Validator returns nil after a real DFS walk.

Distinction from Drop 3 droplet 3.20's "shipped-but-not-wired": that anti-pattern was a SCHEMA FEATURE shipped without any RUNTIME CONSUMER. D5's case is the inverse — the validator IS the runtime consumer; it consumes the existing `BlockedByParent` schema field, walks the existing graph, and confirms acyclicity. The fact that today's schema can't produce a cycle doesn't make the validator unwired — it makes the validator's REJECTION PATH dormant.

Cross-reference: Drop 4a Wave 1.7's runtime `BlockedBy` acyclicity check on action-item UUIDs (cited at load.go:1056-1060) is the runtime mirror. The load-time + runtime pair is intentional: load-time guards the template-author UX (catch cycles BEFORE any action item is created); runtime guards the dispatcher UX (catch cycles introduced by manual orchestrator action-item edits). The vacuous-in-today's-schema posture is documented + accepted.

PLAN.md (line 206) pre-mitigates this attack explicitly:

> **Synthetic-cycle fixture risk.** ... Reviewer might claim "the validator never fires in production." Mitigation: cite Drop 4a Wave 1.7's runtime check as the shape mirror; the load-time validator's value is forward-looking + provides a load-time diagnostic for any future schema addition that could introduce cycles.

REFUTED. The validator IS wired + produces a real DFS walk on every Load; the rejection path is dormant against today's schema but reachable + tested against the injected graph.

Family verdict: REFUTED.

#### B7 — prompt-injection attacks

DORMANT pre-team-feature. Validator inputs are template TOML content authored by the same actor who Loads the template; no attacker-controllable content reaches the error-message rendering. EXHAUSTED.

Family verdict: EXHAUSTED.

### Summary

**Verdict:** PASS.

**Counterexample count:** 0 CONFIRMED, 7 attempted attacks, all REFUTED (B1-B6) or EXHAUSTED (B7).

**Per-family table:**

| Family                                | Verdict   | Notes                                                                         |
| ------------------------------------- | --------- | ----------------------------------------------------------------------------- |
| B1 test-coverage                      | REFUTED   | Cycle-shape coverage helper-mediated via shared `dfsDetectCycle`              |
| B2 contract-preservation              | REFUTED   | Chain-order regression guard pins D3-before-D5 contract                       |
| B3 hidden-coupling (seam reachability)| REFUTED   | Exported seam is rejection-only; abuse is self-harming. NIT: consider unexport |
| B4 YAGNI / extraction                 | REFUTED   | `buildBlockedByGraph` extraction has documented forward-looking justification |
| B5 spec-compliance (synthetic fixture)| REFUTED   | Disclosure is honest; matches L2 PLAN KindPayload shape                       |
| B6 shipped-but-not-wired              | REFUTED   | Validator IS the runtime consumer; rejection path dormant but real DFS runs  |
| B7 prompt-injection                   | EXHAUSTED | DORMANT pre-team-feature                                                      |

**Test-seam verdict:** `LoadOptions.BlockedByGraphFn` is justified — production walker is degenerate today; injection lets the validator be exercised by a real RED→GREEN test. Exposure is rejection-only (cannot suppress real cycles, can only inject synthetic ones); abuse is self-harming. NIT: consider unexporting both this field and D2's `AgentLookupFn` together as a future refinement (both are test-only).

**`buildBlockedByGraph` extraction verdict:** Justified. 7 LOC helper with documented forward-looking purpose (future `BlockedByKinds []domain.Kind` schema axis); single-responsibility separation from validator's traversal+wrapping; matches Go's "name-the-concept" granularity. Inlining would force future schema additions to thread new edge-construction logic through the validator body. Not premature.

**Gate runs:**

- `mage testFunc ./internal/templates TestLoadValidatesBlockedByAcyclicity` → 5/5 sub-tests pass (1 parent + 4 rows).
- `mage testFunc ./internal/templates TestLoadValidatesBlockedByAcyclicityRunsAfterChildRuleCycles` → 1/1 pass.
- `mage testPkg ./internal/templates` → 430/430 tests pass.

Build round 1 lands the kind-level blocked_by-acyclicity validator at the correct chain position (immediately after D4's recursion-depth check), reusing D3's generic `dfsDetectCycle[K ~string]` + `formatCyclePath[K ~string]` helpers, with a forward-looking `buildBlockedByGraph` extraction documented against future schema expansion. The chain-order regression guard pins the D3-before-D5 contract against the existing coupled-cycle fixture. The synthetic-graph injection seam is honestly disclosed in both the validator's godoc and the fixture's doc-comment. All gates green.

### Hylla Feedback

N/A — droplet touched only Go files inside `internal/templates/` plus a new TOML fixture plus the workflow MDs. All Go reads were against `load.go` + `load_test.go` plus `testdata/invalid_blocked_by_acyclicity.toml` in the uncommitted modified working set per `git status`. Hylla's index is stale for those files (load.go has been modified through W0.5.D1 / D2 / D3 / D4 / D5 across the day, none of which Hylla has yet seen). Direct `Read` + `rg` against the working tree was the correct evidence path. No Hylla queries attempted on the in-flight files; nothing to log. The post-drop reingest will re-cover this surface.

One ergonomic NIT carried forward from D4's worklog (recurring; same shape):

- **Query**: `Bash` with `grep -n "<patterns>"` against `internal/templates/load.go`.
- **Missed because**: shell sandbox denied bare `grep` invocations.
- **Worked via**: `rg -n "<patterns>" /Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/templates/<file>`.
- **Suggestion**: not a Hylla item — recurring sandbox-policy NIT for the orchestrator. Documenting again because the loop cost ~2 retries during this round; D4 + D5 builder worklogs noted the same pattern.
