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
