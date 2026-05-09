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
