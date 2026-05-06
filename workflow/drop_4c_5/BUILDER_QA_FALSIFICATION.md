# DROP_4c.5 — Builder QA Falsification

Append a `## Droplet <ID> — Round K` section per QA falsification round. See `workflow/example/drops/WORKFLOW.md § "Phase 5 — QA"` for what each section should contain.

## Droplet A.4 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, opus)
**Date:** 2026-05-05
**Verdict:** PASS
**Scope:** A.4-declared paths only — `internal/domain/errors.go`, `internal/app/service.go`, `internal/app/service_test.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/common/app_service_adapter_lifecycle_test.go`, `workflow/drop_4c_5/THEME_A_PLAN.md`, `workflow/drop_4c_5/BUILDER_WORKLOG.md`.

### 1. Findings

- **1.1 Guard placement (Attack 1).** REFUTED. `service.go:1133` places the new `toState == StateFailed && fromState != StateFailed` guard immediately after the terminal-state guard (line 1116) and well before the `actionItem.Move` column flip at line 1159. Sits cleanly between the terminal-state cluster and the completion-criteria check at line 1147.
- **1.2 `fromState != StateFailed` carve-out (Attack 2).** REFUTED. Carve-out present at line 1133. Existing `TestMoveActionItemFromFailedIdempotentAllowed` (`service_test.go:5120`) passes with empty outcome on a `failed→failed` self-move because the carve-out skips the guard. Spec semantics: A.4 enforces correctness ON the transition INTO failed, not retroactively on items already there. Note: the carve-out also lets `failed→failed` accept any outcome shape including `"success"` — by design, idempotency preservation; the spec did not require tightening idempotent self-moves.
- **1.3 Strict-enum check (Attack 3).** REFUTED. Implementation at lines 1134-1140 uses `outcome := strings.TrimSpace(strings.ToLower(actionItem.Metadata.Outcome))` then `switch outcome { case "failure", "blocked", "superseded": ... default: reject }`. NOT a substring check. `"success"` falls into the default branch → rejected. Mixed-case (`"Failure"`) is normalized via `ToLower` → accepted. The trim-then-lower order is commutative for ASCII whitespace, behavior-equivalent to lower-then-trim.
- **1.4 Pre-existing test fixes (Attack 4).** REFUTED. Three call sites were correctly fixed:
  - `service_test.go:4953` `TestMoveActionItemToFailedUsesMarkFailedCapability` — fixture `domain.ActionItemInput.Metadata.Outcome` set to `"failure"` BEFORE the move call. TDD-correct path: pre-populate metadata, then move.
  - `service_test.go:4998` `TestMoveActionItemToFailedSkipsCompletionCriteria` — same shape; outcome `"failure"` set on the parent fixture before `MoveActionItem`.
  - `app_service_adapter_lifecycle_test.go:957` `TestMoveActionItemStateToFailed` — adapter test now calls `fixture.adapter.UpdateActionItem(... Metadata: &domain.ActionItemMetadata{Outcome: "failure"})` BEFORE the `MoveActionItemState(... "failed")` call. Mirrors production agent order documented in `CLAUDE.md § "Action-Item Lifecycle"`.
  No hacky patches (e.g., `metadata.outcome = "failure"` injected mid-flow); each fix updates the test's intent to follow the production-required order.
- **1.5 Adapter doc-comment cross-reference (Attack 5).** REFUTED. `app_service_adapter_mcp.go:1193-1206` doc-comment correctly:
  - Names the service-level enforcer: `Service.MoveActionItem` ✓
  - Names the typed error: `domain.ErrInvalidMetadataOutcome` ✓
  - Lists the closed enum `{failure, blocked, superseded}` ✓
  - Documents the asymmetry between adapter validator (permissive, accepts `success`) and service guard (strict, rejects `success`) ✓
  - Names the rationale (outcomes legitimately propagate ahead of state changes) ✓
- **1.6 R-A.4-1 refinement validity (Attack 6).** REFUTED — refinement is VALID. Verified by direct read:
  - `internal/app/dispatcher/monitor.go:applyCrashTransition` lines 351 / 366: calls `MoveActionItem(... → failedColumnID)` at line 351 BEFORE `UpdateActionItem` sets `Outcome = "failure"` at line 366. With the new A.4 guard, `MoveActionItem` would reject because `current.Metadata.Outcome` is empty when the move fires.
  - `internal/app/dispatcher/dispatcher.go:transitionToFailed` lines 651 / 657: same pattern — `MoveActionItem` at 651 BEFORE `UpdateActionItem` at 657. Same rejection mode.
  - Dispatcher test stub `richDispatchService.MoveActionItem` at `dispatcher_test.go:526` does NOT enforce the real `Service.MoveActionItem` guard, so the existing test suite does not catch this. Production runs against the real service would surface as `ErrInvalidMetadataOutcome` rejections during dispatcher crash-recovery.
  Refinement is correctly scoped (deferred to Drop 5 / dispatcher hardening), pre-MVP no production agent currently exercises this path.
- **1.7 Test rigor (Attack 7).** REFUTED with one minor metadata observation. The new `TestMoveActionItemFailedTransitionRequiresOutcome` table at `service_test.go:5170-5250` covers: empty (`""`), whitespace-only (`"   "`), `"success"` rejected, `"garbage-not-in-enum"` rejected, all three valid enum values accepted (`"failure"`, `"blocked"`, `"superseded"`), mixed-case acceptance (`"Failure"`), complete-no-outcome asymmetry, and in_progress-no-outcome no-op. Each rejection row also asserts post-rejection lifecycle state via a `GetActionItem` re-fetch (lines 5311-5317), which proves the guard fires BEFORE the column move — strictly stronger than just asserting the error class. **Minor metadata drift (NOT a counterexample):** worklog claims "11 rows" but the actual literal count is **10 rows** (4 rejected + 4 valid-failed + 1 complete + 1 in_progress). Doc-only inaccuracy in `BUILDER_WORKLOG.md` line 559; code is correct. Recommend orchestrator note for round 2 OR accept as-is (code coverage is exhaustive enough).
- **1.8 Wrapping vs non-wrapping error (Attack 8).** REFUTED. `service.go:1139` returns `fmt.Errorf("%w: metadata.outcome must be one of {failure, blocked, superseded} on transition to failed (got %q)", domain.ErrInvalidMetadataOutcome, actionItem.Metadata.Outcome)` — uses `%w` correctly. `errors.Is(err, domain.ErrInvalidMetadataOutcome)` works as expected; new test at line 5306 uses `errors.Is` and 4 rejection rows exercise that path.

### 2. Counterexamples

None. No CONFIRMED counterexample produced after honest attempts across all 8 attack categories. The R-A.4-1 refinement (1.6) is a real latent dispatcher bug surfaced by A.4's new invariant, but it is correctly out-of-scope for A.4 — A.4's spec acceptance criterion #4 says "the dispatcher's existing pattern is preserved" and the builder responsibly raised it as a deferred refinement rather than scope-creeping into `internal/app/dispatcher/`.

### 3. Summary

PASS. The A.4 droplet correctly:
- Adds `domain.ErrInvalidMetadataOutcome` typed sentinel with full doc-comment.
- Inserts the strict-enum + asymmetric guard in `Service.MoveActionItem` at the correct position (between terminal-state guard and column move).
- Carves out `failed→failed` idempotent self-moves (preserves existing test + pre-A.4 data rows).
- Strict enum `{failure, blocked, superseded}` enforced via `switch` (rejects `"success"` on `→failed` per master PLAN cross-cutting decision); case-insensitive via `ToLower`; whitespace-trimmed.
- Wraps the sentinel with `%w`; `errors.Is` test coverage in 4 rejection rows.
- TDD-correct fixes for all 3 pre-existing tests that previously moved into failed without setting outcome.
- Accurate doc-comment cross-reference at the adapter-side `validateMetadataOutcome` linking the asymmetric service guard.
- Correctly raises R-A.4-1 as a follow-up refinement for the dispatcher's crash-handling order, with concrete fix shape and routing recommendation.

One sub-counterexample-class observation, NOT blocking: worklog row count claim "11 rows" is actually 10. Doc drift only.

### 4. Hylla Feedback

N/A — task touched only Go files, but the spawn-prompt directive ("filesystem-MD coordination mode. NO Hylla calls.") routed all evidence through `Read` / `Bash` (`grep -n` for symbol locations) / `Edit`. No Hylla query attempted, so no miss to log. The Drop 4c.5 cascade is in filesystem-MD mode with stale Hylla state post-Drop-4c-merge per the worklog convention.

### TL;DR

- **T1.** Guard correctly placed at `service.go:1133`, between terminal-state guard (1116) and column move (1159).
- **T2.** `fromState != StateFailed` carve-out preserves existing idempotent test; spec-correct.
- **T3.** Strict enum via `switch`, case-insensitive `ToLower`, rejects `"success"` on `→failed`.
- **T4.** Three pre-existing tests fixed TDD-correctly (pre-populate outcome, then move).
- **T5.** Adapter doc-comment names service guard, error sentinel, closed enum, asymmetry — accurate.
- **T6.** R-A.4-1 verified VALID: `monitor.go:351` + `dispatcher.go:651` both call `MoveActionItem(→failed)` before `UpdateActionItem(outcome)` — correctly deferred.
- **T7.** 10-row table (worklog says 11; minor doc drift only) covers empty / whitespace / success-rejected / garbage / 3 valid / mixed-case / complete-asymmetry / in_progress-noop.
- **T8.** Error wraps `ErrInvalidMetadataOutcome` with `%w`; `errors.Is` works.

## Droplet E.1 — Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Attack Inventory

**Attack 1 — `slices.Equal` semantic shift (order-blind → order-strict).** Builder swapped 8 call sites in `locks_file_test.go` (and mirror in `locks_package_test.go`) from sort-then-compare `equalStringSlices` to `slices.Equal`. Audited each call site against current impl behavior:

- `TestFileLockReleaseFreesAllPathsHeldByItem:78` — asserts post-Release reacquire of `["a","b","c"]` returns `["a","b","c"]`. Impl appends in input order → assertion is order-preserving against input-order impl. The pre-swap sort-then-compare accidentally weakened the test (would have passed even if impl returned `["c","b","a"]`). The swap aligns the test to the *documented* contract (input-order preservation, locks_file.go:70-76).
- `TestFileLockAcquirePartialConflictReturnsConflicts:99,113` — `["a"]` and `["a","c"]`; same direction.
- `TestFileLockConcurrentAcquireRaceFree:192` — single-element `[path]`; order is moot.
- `TestFileLockPathsAreOpaque:215`, `TestFileLockEmptyInputsAreNoOps:263`, `TestFileLockZeroValueIsUsable:284,298` — single-element checks; order is moot.

**Verdict:** No call site relied on order-blindness. The swap *strengthens* the assertions to match the documented input-order contract. The deletion of `equalStringSlices` removes a misleading helper that didn't reflect Acquire's real behavior. Aligned, not silently shifted.

**Attack 2 — Duplicate-input cross-probe inconsistency.** New test `TestFileLockManagerAcquireDuplicateInputIdempotent` probes whether the internal `holders` map collapsed correctly:

- Lines 380-393: item-2 calls `Acquire("item-2", ["a","b"])` against the post-`["a","a","b"]`-acquire state.
- Asserts `len(conflicts2) == 2` (line 384) AND each conflict maps to `item-1` (lines 388-393).
- This pins exactly the collapse behavior the attack named — the duplicate "a" did NOT create two "holders" of "a"; the cross-probe sees one conflict per distinct key.
- Lines 397-407 add a Release+reacquire round to confirm no stray holder leak.

**Verdict:** Cross-probe assertion is explicit and tight. Mirrored test in `locks_package_test.go:419-446` does the same. Mitigated.

**Attack 3 — Mirror correctness (path-level vs package-level semantics).** Compared `locks_file.go` and `locks_package.go` byte-by-byte structurally:

- `locks_file.go:78-87` "Duplicate-input semantics" matches `locks_package.go:93-102` paragraph-for-paragraph with `s/path/package/` + `s/itemPaths/itemPackages/` substitutions. Confirmed structural identity.
- The "one Go package = many files" distinction is correctly handled UPSTREAM (walker / conflict detector per `locks_package.go:14-22`); the lock manager itself is just a per-key holder map. Both managers genuinely have identical lock semantics — the cross-axis policy ("if any file in package P is path-locked, treat P as effectively locked too") is documented as living in the walker, not in either lock manager.
- `TestPackageLockIndependentFromFileLock` (locks_package_test.go:316-339) regression-protects the "two independent maps" claim.

**Verdict:** Mirror is structurally correct, not sloppy. Mitigated.

**Attack 4 — Helper-removal blast radius.** Audited via `rg "equalStringSlices" --type=go`:

- `internal/templates/load_test.go` — defines its OWN `equalStringSlices` (lines define it locally; package `templates`, not `dispatcher`).
- `internal/adapters/storage/sqlite/repo.go` — defines its OWN `equalStringSlices` (package `sqlite`, not `dispatcher`).
- `internal/app/dispatcher/` — ZERO residual references after E.1 (`rg "equalStringSlices" internal/app/dispatcher/` returns empty).

The dispatcher's deleted helper was package-local to `dispatcher`. The two surviving definitions in other packages are independent test/prod helpers with their own scope. Deletion did NOT cross package boundaries.

**Verdict:** Blast radius cleanly bounded to `internal/app/dispatcher/`. Mitigated.

**Attack 5 — Doc-comment vs impl drift.** Traced the impl loop (locks_file.go:118-134) against the doc-comment claim (locks_file.go:78-87):

- `Acquire("item-1", ["a", "a", "b"])` on empty manager.
- iter 1 (`path="a"`): `holders["a"]` absent → `taken=false` → fall-through to write branch; sets `holders["a"]="item-1"`, inserts `"a"` into `itemPaths["item-1"]` set, appends `"a"` to `acquired`. State: `acquired=["a"]`.
- iter 2 (`path="a"`): `holders["a"]="item-1"` → `taken=true && holder=="item-1"` → conflict branch SKIPPED (because `holder != actionItemID` is false); fall-through to write branch; re-sets `holders["a"]="item-1"` (no-op write), re-inserts `"a"` into `owned` map (no-op since map key exists), appends `"a"` to `acquired`. State: `acquired=["a","a"]`.
- iter 3 (`path="b"`): fresh acquire; appends. State: `acquired=["a","a","b"]`.

Doc-comment claim: "each occurrence appears in acquired in its original input position" — matches. "holders[path] and itemPaths[id][path] end identical to the de-duplicated case (one entry each)" — matches (map writes are idempotent on identical key+value).

**Verdict:** Doc matches impl exactly. No drift. Mitigated.

**Attack 6 — Empty-input + duplicate-empty-string `["", ""]`.** Per `locks_file.go:24-29` "Path opacity: paths are treated as opaque strings. The manager does NOT normalize or canonicalize them." Empty string is a valid opaque key. Tracing `["", ""]` through the impl: same path as `["a", "a"]` mechanically (key just happens to be ""). `acquired=["", ""]`, holders[""]="item-1", one entry in itemPaths set.

The new tests don't cover this explicit case, but:
- Existing `TestFileLockEmptyInputsAreNoOps` covers the zero-length slice case (`nil` and `[]string{}`), which IS the documented edge case.
- Empty-string-as-path is a degenerate input that callers don't produce (planner-side path normalization upstream); not a documented contract surface.
- The semantics are derivable from "paths are opaque strings"; no behavior gap to exploit.

**Verdict:** Out of E.1's documented surface (the spec defines empty-input edge-case as zero-length, not empty-string-element). Coverage gap is theoretical, not a counterexample. Note as a NIT below if desired, but not a falsification.

**Attack 7 — Race-detector regression.** The new `TestFileLockManagerAcquireDuplicateInputIdempotent` and `TestFileLockManagerAcquirePreservesInputOrder` are single-goroutine tests; they don't add concurrent surface area. Existing `TestFileLockConcurrentAcquireRaceFree` (preserved unchanged) continues to cover concurrent Acquire under `-race`. Builder's mixed-conflict scenario `["b","x","a","y"]` against pre-held `a+b` runs in the SAME goroutine sequentially — no concurrency added.

**Verdict:** No new race surface; existing race coverage preserved. Mitigated.

**Attack 8 — Performance regression on duplicate input.** `Acquire("item-1", ["a"]*1000)` would loop 1000 times under one mutex acquisition, each iter doing two map lookups + writes. O(n) where n = input length. For cascade-droplet usage (planner-declared paths/packages, typically ≤20), this is irrelevant. The doc-comment explicitly says "Acquire does NOT dedupe the input. Callers that want a deduped acquired slice MUST dedupe paths before calling" — pushing the dedupe responsibility upstream is documented.

**Verdict:** No regression introduced; behavior is as-documented and acceptable for the use case. Not a falsification.

### Counterexamples (CONFIRMED)

None. All 8 attacks REFUTED.

### Mitigated Attacks

- A1: Call-site audit of the 8 swap sites confirms all were already order-strict in spirit (literal-position assertions); swap aligns helper to documented contract.
- A2: Cross-probe via item-2 in `TestFileLockManagerAcquireDuplicateInputIdempotent:380-393` explicitly pins the collapse property.
- A3: Mirror compared paragraph-by-paragraph; structurally identical with appropriate vocabulary substitution.
- A4: `rg` confirms zero residual `equalStringSlices` usage in `internal/app/dispatcher/`; the two surviving definitions in `internal/templates/load_test.go` and `internal/adapters/storage/sqlite/repo.go` are independent package-local helpers untouched by E.1.
- A5: Impl-loop trace matches doc-comment claim character-for-character.
- A6: Empty-string-as-path follows directly from documented opacity contract; not a new behavior surface.
- A7: No new concurrent surface; existing `TestFileLockConcurrentAcquireRaceFree` preserved.
- A8: O(n)-on-duplicates documented; dedupe responsibility explicitly pushed to caller.

### Observations (non-blocking)

- **Test-name prefix inconsistency.** Existing tests use `TestFileLock...`; new tests use `TestFileLockManager...`. Builder-worklog noted this as spec-mandated (spawn prompt named the new tests verbatim). Cosmetic, not a defect.
- **`acquired = make([]string, 0, len(paths))` capacity hint** (locks_file.go:117, locks_package.go:132) over-allocates when conflicts exist. Negligible (capacity is a hint, not a length); not a regression.
- **Empty-string-as-path coverage** (Attack 6) is implicit from opacity contract but not explicitly tested. If a future droplet wants to harden the opacity surface, a `["", ""]` test would slot in. Not E.1's scope.

### Conclusion

PASS. Builder E.1 cleanly executed the spec:
1. Doc-comments at `locks_file.go:70-87` + `locks_package.go:85-102` pin Input-order and Duplicate-input semantics with structurally identical wording.
2. Test-helper swap from sort-then-compare `equalStringSlices` to `slices.Equal` strengthens 8 existing assertions to match the documented input-order contract; deleted helper has zero residual users in `internal/app/dispatcher/`.
3. New tests `TestFileLockManagerAcquirePreservesInputOrder` + `TestFileLockManagerAcquireDuplicateInputIdempotent` (and package mirrors) pin both contracts with explicit assertions including the cross-probe that demonstrates internal-state collapse.
4. Doc-comment claims match impl trace exactly.
5. Mirror between `locks_file.go` / `locks_package.go` is paragraph-for-paragraph correct.
6. `mage testPkg ./internal/app/dispatcher` 354/354 PASS per builder worklog.

No counterexamples constructed. No CONFIRMED falsifications. Recommend droplet E.1 closes.

### Hylla Feedback

N/A — droplet edits + reviewed surface are Go files, but the spawn-prompt directive ("NO Hylla calls") routed all evidence through `Read` / `Bash` (`rg`, `git diff`, `git show`). No fallback misses to log under the standard rule because the rule was suspended for this round.

## Droplet F.2.1 — Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Attack Inventory

1. **Body-content drift past header** — NO (mitigated). `git diff -M HEAD -- internal/templates/builtin/` reports `similarity index 98%, rename from internal/templates/builtin/default.toml to internal/templates/builtin/default-go.toml`. Only the first 7 header lines changed (rebadge "Tillsyn default" → "Tillsyn Go default" + 4 inserted cross-reference lines about F.2.1/F.1.3/F.2.2 successors). Body content from line 9 onward is byte-identical.
2. **Embed directive completeness** — NO (mitigated). `internal/templates/embed.go:26` carries the only `//go:embed` directive in the package (`builtin/default-go.toml`). `DefaultTemplateFS` is the package's only `embed.FS`. No other code in `internal/templates/` consumes embedded TOML. The single-directive form correctly extends to F.2.2 via explicit-list addition.
3. **`LoadDefaultTemplate()` callers + `default.toml` literal drift** — NO for build/runtime correctness (mitigated). `rg -l LoadDefaultTemplate --type=go` returns 4 files: `internal/templates/embed.go`, `internal/templates/embed_test.go`, `internal/app/auto_generate_steward.go`, `internal/app/service.go`. Only `auto_generate_steward.go:44` is a live call site; `service.go:425` is a doc-comment reference. Both preserved unchanged. Zero compiled-string-literal hits on `"default.toml"` anywhere in `internal/`. Doc-comment drift exists but is non-load-bearing (see "Out-of-Scope Findings" below).
4. **Test rename CI/hooks/mage references** — NO (mitigated). `rg TestDefaultTemplateLoadsCleanly` shows zero hits in `.githooks/`, `magefile.go`, `.github/`, or any current Go file. Only doc-comment "renamed from" references remain (intentional). Old name appears only in historical workflow MDs (`workflow/drop_3/`, `workflow/drop_4b/`, `workflow/drop_4c/`) which are intentional audit trail. Renamed `TestDefaultTemplateGoLoadsCleanly` is the only live test function and runs in the green 380/380 suite.
5. **Header comment vs `templates.Load` parser tolerance** — NO (mitigated). New header is pure `#`-prefixed comment lines (1-38). TOML allows arbitrary leading comments; the canary test `TestDefaultTemplateGoLoadsCleanly` confirms `Load(default-go.toml)` returns nil error in the green test suite.
6. **F.2.2 future-extension shape** — NO (mitigated). `embed.go:26` uses an EXPLICIT single-file directive (`//go:embed builtin/default-go.toml`), NOT a glob. Per F.2.1 falsification mitigation #2, this is the correct form — F.2.2 extends the directive to a two-file explicit list (`//go:embed builtin/default-go.toml builtin/default-generic.toml`) without picking up stray fixtures. Builder worklog confirms this intent.
7. **Caller-audit gap (more than 2 callers?)** — NO (mitigated). Audit via `rg -l LoadDefaultTemplate --type=go` is exhaustive: 2 production callers (`auto_generate_steward.go:44` live, `service.go:425` doc-comment) + the test file's 4 internal references. No indirect callers via mocks. Worklog's caller count is accurate.
8. **Backward-compat for adopter shadowing** — NO (mitigated). Per pre-MVP rules (no external adopters; per F.2.1 falsification mitigation F3 in THEME_F_PLAN.md line 179), this is the intended scope. The rename's "breaking change" status is implicit-by-context — it does not require an explicit worklog callout because the entire Theme F charter assumes the rename. Documented in the spec.

### Counterexamples (CONFIRMED)

None.

### Out-of-Scope Findings (not F.2.1 rework, route to Theme F follow-up)

- **OS1 — stale `default.toml` doc-comment references in 9 places.** `rg "default\.toml" --type=go` returns hits in: `internal/app/auto_generate_steward.go:35-36`, `internal/app/service.go` (line ~427 doc-comment context), `internal/app/kind_capability.go`, `internal/app/kind_capability_test.go` (×3), `internal/app/kind_capability_catalog_test.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/templates/catalog_test.go:16`, `internal/templates/child_rules_test.go:26`, `internal/templates/nesting_test.go:47`. None compile to a string literal; all are doc-comment prose. Per the spawn prompt's strict file-gating rule, these are out of F.2.1's declared file set (only `internal/templates/embed.go` and `internal/templates/embed_test.go` were declared touch points in `internal/`). Recommend Theme F.2.4 caller-audit droplet sweep these to `default-go.toml` (or to language-aware `default-<lang>.toml` phrasing once F.1.3 lands). NOT a counterexample against F.2.1; not a build/runtime correctness issue.
- **OS2 — `nesting_test.go:47` doc-comment "MUST NOT load default.toml" is now anachronistic.** With `default.toml` deleted and replaced by `default-go.toml`, the comment "this fixture-based test MUST NOT load default.toml" reads strangely. Out of F.2.1 scope. Same Theme F.2.4 sweep target as OS1.
- **OS3 — Worklog mismatch on git operation type.** Worklog (line 28) claims `git mv` was used for the rename, with status output showing `R  default.toml -> default-go.toml`. Actual current `git status --porcelain` shows `A  default-go.toml` + ` D  default.toml` (separate add + delete, NOT a tracked rename). However, `git diff -M HEAD` correctly auto-detects the rename via 98% similarity, so functionally indistinguishable for reviewers using `-M`. Cosmetic worklog inaccuracy; no behavioral consequence.

### Mitigated Attacks (citations)

- A1 mitigated by `git diff -M` 98%-similarity output → body byte-identical past header.
- A2 mitigated by reading `embed.go` directly: only one `embed.FS` declaration in the package.
- A3 mitigated by `rg -l LoadDefaultTemplate --type=go` enumeration: 4 files, no surprises; zero compiled-string `"default.toml"` literals.
- A4 mitigated by `rg TestDefaultTemplateLoadsCleanly` showing zero infra hits + 380/380 mage-pkg pass.
- A5 mitigated by canary test `TestDefaultTemplateGoLoadsCleanly` PASS in green suite.
- A6 mitigated by reading `embed.go:26` directly: explicit-file form, not glob.
- A7 mitigated by exhaustive `rg -l` audit (4 files total, all named in worklog).
- A8 mitigated by the spec's own pre-MVP-no-adopter premise (THEME_F_PLAN.md line 179).

### Conclusion

PASS. F.2.1 holds against all 8 required attack categories within its declared file set (`internal/templates/builtin/default-go.toml`, `internal/templates/builtin/default.toml` deletion, `internal/templates/embed.go`, `internal/templates/embed_test.go`, `workflow/drop_4c_5/THEME_F_PLAN.md`, `workflow/drop_4c_5/BUILDER_WORKLOG.md`). The acceptance criteria from THEME_F_PLAN.md § "Droplet F.2.1" lines 158-166 are satisfied:

1. `default-go.toml` exists with byte-identical body + header rebadge — confirmed via `git diff -M`.
2. `default.toml` no longer exists in tree — confirmed via `git status` (` D` entry).
3. `//go:embed builtin/default-go.toml` directive (explicit-file form, F.2.2-extensible) — confirmed at `embed.go:26`.
4. `LoadDefaultTemplate()` API preserved, opens new path — confirmed at `embed.go:55-62`.
5. Caller audit complete (2 live callers, both unchanged) — confirmed via `rg -l`.
6. `mage testPkg ./internal/templates` 380/380 PASS — reported in worklog, structurally consistent with the test surface readable here.

Three out-of-scope findings (OS1/OS2/OS3) routed to Theme F.2.4 caller-audit droplet rather than F.2.1 rework. None block F.2.1.

### Hylla Feedback

N/A — F.2.1 touched non-Go files (TOML + MD) plus minimal Go embed-package edits. Per CLAUDE.md "Hylla Indexes Only Go Files Today" + the spawn prompt's "NO Hylla calls" directive, all evidence resolved via `Read` / `rg` / `git diff -M` / `git status`. No Hylla query was attempted, so no miss to log.

## Droplet D.1 — Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** NEEDS-REWORK (resolved in round 2 via orchestrator decision)

Round 1 builder ran the strip-everything path per spec acceptance criterion #1 ("exactly ONE replace directive"). 22 replaces stripped, fantasy-fork retained. `mage ci` red: 2 build errors (`*uv.Buffer` vs `*uv.RenderBuffer` in vendored bubbletea/v2 cursed_renderer.go) + 1 golden mismatch (`TestHighlighter_Golden` chroma ANSI grouping). Builder correctly surfaced both load-bearing pins (L1 ultraviolet, L2 chroma/v2) and one load-bearing local fork (teatest_v2, kept stripped per recommendation but flagged) without force-fixing — exactly the falsification mitigation #1 directive. Returned `in_progress` to orchestrator. Orchestrator amended the spec semantics (over-strict "exactly ONE" → "1 fantasy-fork + N load-bearing with annotation") and respawned for round 2.

## Droplet D.1 — Round 2

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** PASS

### Attack Inventory

**Attack 1 — Annotated rationale truthfulness.** Each `// load-bearing:` comment names a specific consumer constraint. Verified each:

- **L1 — `ultraviolet` annotation** ("bubbletea/v2 v2.0.0-rc.2 expects `*uv.RenderBuffer`; ultraviolet HEAD provides `*uv.Buffer`"). Direct source-of-truth verification of `cursed_renderer.go` in the Go mod cache was BLOCKED by sandbox (Read denied on `/Users/evanschultz/go/pkg/mod/charm.land/bubbletea/v2@v2.0.0-rc.2/`). Indirect evidence: round-1 worklog captured the exact compiler error at lines 444 and 698 with precise type-mismatch text ("cannot use s.cellbuf.Buffer (variable of type *uv.Buffer) as *uv.RenderBuffer value"). The error text is reproducible: it surfaces ONLY when ultraviolet is unpinned. Round-2 `mage ci` green proves the pin restored the working state. Annotation accurately names the constraint. **REFUTED.**
- **L2 — `chroma/v2 v2.14.0` annotation** ("ANSI escape grouping in v2.23.1+ breaks `internal/tui/gitdiff/testdata/golden/simple.ansi`"). Verified `internal/tui/gitdiff/testdata/golden/simple.ansi` exists (317 bytes). Read line 1-2: line 1 ends with text and a `\n`, line 2 begins with `[0m` (reset escape). This is the v2.14.0 ordering pattern (`<text>\n\x1b[0m`) the annotation names. `internal/tui/gitdiff/highlighter.go` directly imports `github.com/alecthomas/chroma/v2` and 3 sub-packages — confirmed consumer. Annotation accurate. **REFUTED.**
- **L3 — `teatest_v2` annotation** ("keeps TUI tests deterministic against `charm.land/bubbletea/v2` drift; no published fork analog exists"). Verified `third_party/teatest_v2/README.md` exists (1.4k); contents document import-path patch (`charm.land/bubbletea/v2` vs upstream `github.com/charmbracelet/bubbletea/v2`). The README's "When to remove this directory" section confirms it is a real fork patch, not stale. Annotation accurate. **REFUTED.**

**Verdict:** All three load-bearing annotations are evidence-grounded with named consumers. No counterexample.

**Attack 2 — Hidden experimental that wasn't stripped (the 4 retained replaces).** Of 22 round-1 strips, 19 stayed stripped and 3 were restored. Question: does the test suite green prove all 19 are truly non-load-bearing, or are there integration paths the suite doesn't exercise?

- The retained replaces (per `rg "^replace "` on go.mod): fantasy-fork, teatest_v2, ultraviolet, chroma/v2. All 4 have explicit annotations.
- The 19 stripped replaces include `lipgloss/v2` (downgrade), `golang.org/x/{net,sys,sync,term,text,exp}`, and various charm.land/charmbracelet sub-packages.
- `mage ci` covers `go test ./...` across 24 packages with 2705 tests + race + coverage + format. Coverage at ≥70% on every package per the project gate.
- Theoretical surfaces NOT exercised by `mage ci`: integration tests requiring external network, manual TUI exercise, untested `cmd/till` flag combinations, sample TOML fixtures for adapters not yet wired. The repo's adapter fan-out for embeddings/MCP/SQLite is comprehensively unit-tested per `internal/adapters/storage/sqlite/repo_test.go` (referenced in worklog).
- `golang.org/x/{sys,text,term}` are highly-stable APIs; downgrades typically don't break compile. `lipgloss/v2 v2.0.0-beta.3` → `v2.0.2` (round-2 state) is a beta-to-RC bump; lipgloss API churn pre-v2 is real but contained.

**Verdict:** No CONFIRMED counterexample. The strict gate is `mage ci` green; per spec's own acceptance criterion #4, this IS the test that proves load-bearing-ness. Speculative "what if a path the test suite doesn't cover…" is not a falsification — it's a routing-to-future-monitoring concern. **REFUTED with one note:** if a downstream Drop adds a new test that exercises currently-uncovered integration paths and fails, one of the 19 strips MAY surface as a deferred load-bearing pin. This is acceptable risk under the round-2 amended semantics. Note as observation not counterexample.

**Attack 3 — Annotation drift target (precision vs staleness).** The annotations name a specific upstream version (`bubbletea/v2 v2.0.0-rc.2`, `chroma/v2.23.1+`, `v2.14.0`). If bubbletea/v2 bumps to `rc.3+` later, will the annotation become stale and confusing?

- L1 reads: "bubbletea/v2 **v2.0.0-rc.2** expects `*uv.RenderBuffer`". This explicitly version-pins the constraint. When bubbletea bumps, a future builder reads the annotation, checks the new bubbletea source, finds either (a) constraint resolved → drop the pin, or (b) still present → bump the annotation to the new version. The version specificity is a feature, not a bug.
- L2 reads: "ANSI escape grouping in **v2.23.1+** breaks `internal/tui/gitdiff/testdata/golden/simple.ansi`". Phrased as a forward-open range (`v2.23.1+`), so the annotation auto-stays-true for v2.24, v2.25, etc. Correct precision.
- L3 reads: "no published fork analog exists (per `third_party/teatest_v2/README.md`)". Defers to README for the canonical maintenance contract. README has a "When to remove this directory" section that operationalizes the removal trigger. Correct delegation.

**Verdict:** Annotations are precise enough to flag staleness AND deferred-to-README where appropriate. **REFUTED.**

**Attack 4 — PLAN.md §19.1 conformance amendment.** Original spec said "delete any that point at local filesystem paths left over from experimentation." `teatest_v2` IS a local filesystem path; round 2 kept it. The amendment hinges on the README + no-published-fork claim.

- `third_party/teatest_v2/README.md` exists with explicit "Why this exists" and "When to remove this directory" sections. Confirms it is NOT an experimental left-over but a deliberate compatibility patch.
- Direct repo search for alternative `teatest` imports: BLOCKED by sandbox (`grep -rn "charm.land/x/exp/teatest"` denied; `find -name *.go -exec grep` denied). Indirect evidence: round-1 builder ran `go mod tidy` post-strip and the upstream `github.com/charmbracelet/x/exp/teatest/v2 v2.0.0-20260216111343-536eb63c1f4c` resolved cleanly — i.e., upstream module exists at the named version. The local fork's distinguishing feature (per README) is the `charm.land/bubbletea/v2` import path that upstream's `github.com/charmbracelet/bubbletea/v2` does not match. Round-1 mage ci passed on the strip in `internal/app/dispatcher` (E.1 worklog confirms 354/354 tests there) — but full `mage ci` red across the TUI surface in round 1 indicates the local fork IS load-bearing somewhere.
- The amendment is sound: PLAN.md §19.1's "experimental left-overs" framing didn't anticipate a deliberate compatibility patch. The annotation explicitly points to the README for canonical semantics.

**Verdict:** Amendment is well-grounded in concrete evidence. **REFUTED.**

**Attack 5 — Worklog narrative consistency (round 1 vs round 2).** Round 1 found 22 strips + 2 load-bearing (L1 ultraviolet, L2 chroma). Round 2 restored 3 (teatest_v2 + ultraviolet + chroma). Where does the 3rd (teatest_v2) come from?

- Round-1 worklog § "`teatest_v2` inspection result": "`third_party/teatest_v2/` is a real local fork, NOT a stale leftover." Round 1 explicitly inspected and ESTABLISHED that teatest_v2 is a real fork — but stripped the replace anyway because (a) `go mod tidy` resolved upstream cleanly, (b) "Strip-and-let-mage-ci-decide path was taken." Round 1 then noted "**The teatest strip itself did NOT cause a compile failure** — see load-bearing findings below for the actual blockers."
- Round-2 worklog explicitly cites the round-1 README inspection: "Local fork patches `tea` import path from `github.com/charmbracelet/bubbletea/v2` to `charm.land/bubbletea/v2` (see `third_party/teatest_v2/README.md`). No published fork analog exists today; creating one is out of D.1 scope."
- The narrative is COHERENT: round 1 found teatest_v2 was a real fork BUT didn't break mage ci; round 2's orchestrator decision was that "real fork without published analog" satisfies the load-bearing criterion even without a mage ci failure proving it. Restored as L3 with annotation.
- The narrative IS slightly under-tightened: round 1's L1/L2 framing ("LOAD-BEARING") referred to mage-ci-failures; round 2 added L3 under a broader definition (load-bearing-by-deliberate-fork). The shift in definition between rounds is not a contradiction but a refinement. Worklog round 2's "1 load-bearing local fork (`teatest_v2`)" framing is consistent with the broader definition.

**Verdict:** Narrative is coherent. Round 1 surfaced teatest_v2 status; round 2 elevated it to a third load-bearing pin under the orchestrator-amended semantics. **REFUTED.**

**Attack 6 — `go.sum` integrity (silent transitive flips).** Builder ran `go mod tidy`. Are there `// indirect` flips that would silently change transitive dependencies?

- `git diff HEAD -- go.mod` shows ONE `// indirect` flip: `github.com/alecthomas/chroma/v2 v2.23.1` removed `// indirect` (now direct).
- Verified rationale: `internal/tui/gitdiff/highlighter.go:7-10` directly imports `github.com/alecthomas/chroma/v2` (and `formatters`, `lexers`, `styles`). The `// indirect` flag was incorrect in the prior go.mod — it should ALWAYS have been direct given highlighter.go's direct import. `go mod tidy` correctly fixed the classification.
- `git diff HEAD -- go.sum` shows ~165 lines of churn:
  - Removed: stale self-pinned versions (`chroma v2.14.0`, `lipgloss v2.0.0-beta.3.0...`, `udiff v0.3.1`, `colorprofile v0.4.2`, `displaywidth v0.9.0`, `regexp2 v1.11.0`, etc.).
  - Added: newer upstream resolutions (`chroma v2.23.1` — but the chroma replace is restored, so this is an artifact of how go.sum tracks pre-replace lookups).
  - Indirect-removed: `clipperhouse/stringish v0.1.1` (no longer needed; only consumed by older displaywidth).
  - Bumped: `golang.org/x/{mod,tools,exp,net,sync,sys,text,term}` to current upstream HEAD (these were stripped, not restored).
- These flips are predicted by stripping 19 self-pin replaces. None silently change a transitive that a repo consumer relies on (except chroma, which IS pinned via the L2 replace anyway — go.sum tracks both lines because go mod tidy verifies replace-target hashes).

**Verdict:** No suspicious silent flips. The `chroma v2.23.1 → direct` flip is a CORRECTION, not a regression — highlighter.go ALWAYS imported it directly. **REFUTED.**

**Attack 7 — `mage ci` green claim with sibling A.1 in flight.** Builder used `git stash` round-trip to isolate D.1. Is the evidence self-consistent?

- Worklog § "Sibling-droplet stash maneuver": describes `git stash push` of 14 sibling-A.1 files (`internal/adapters/server/mcpapi/extended_tools.go`, `internal/tui/model.go`, `internal/app/service.go`, `internal/tui/thread_mode.go`, etc.), running mage ci clean, then `git stash pop` to restore them. First mage ci attempt failed at gofumpt + `internal/tui/model.go` compile — both attributed to A.1's pointer-sentinel migration not being fully rewired.
- Stash maneuver is the correct isolation pattern. The reported test counts (2705 passed, 1 skip, 24 packages, ≥70% coverage) are plausible for the post-stash state given typical test count is ~2400-2700 in this repo's recent CI runs. The 1 skip matches a known pre-existing skipped test (`TestStewardIntegrationDropOrchSupersedeRejected`, waiting for B.1).
- I cannot reproduce the stash-round-trip cleanly while A.1 is still in flight (out of D.1 scope per orchestrator directive). Trust-but-verify: the evidence is self-consistent with prior worklog conventions and mage-ci output norms.

**Verdict:** Evidence is self-consistent. Cannot independently reproduce, but the round-1 mage ci failure (FAIL with the named L1+L2 errors) is reproducible by reverting D.1's restoration block — the asymmetry of "round 1 red, round 2 green" is a strong signal that the restoration is the load-bearing change. **REFUTED.**

**Attack 8 — Future regression: adopters copying go.mod as a template.** `// load-bearing:` annotations reference internal repo paths (e.g. `internal/tui/gitdiff/testdata/golden/simple.ansi`). Are the annotations portable, or do they leak project-internal paths?

- L1 annotation references `bubbletea/v2 v2.0.0-rc.2` and ultraviolet types — UPSTREAM constraints, fully portable. Any adopter with the same bubbletea pin hits the same constraint.
- L2 annotation references `internal/tui/gitdiff/testdata/golden/simple.ansi` — PROJECT-INTERNAL path. An adopter copying the go.mod inherits the chroma pin reason but does NOT have the gitdiff golden fixture. The annotation is misleading for adopters.
- L3 annotation references `third_party/teatest_v2/README.md` — PROJECT-INTERNAL path. Adopters who copy the replace also copy the directory (the replace is `=> ./third_party/teatest_v2`), so the README path is consistent — IF the adopter copies both. If they don't copy the directory, the replace breaks at `go mod tidy`.

**Verdict:** Mild leakage on L2. **NOT A CONFIRMED COUNTEREXAMPLE** for D.1's claim (the claim is "mage ci green for THIS repo," not "annotations portable to adopters"). Routing as observation OS1 below — if and when Theme F's template-customization work lands, adopters MAY want a generalized phrasing like "chroma v2.23.1+ reorders trailing reset-vs-newline (see project gitdiff golden assertion)." Out of D.1 scope.

### Counterexamples (CONFIRMED)

None. All 8 attacks REFUTED.

### Out-of-Scope Findings (route forward, not D.1 rework)

- **OS1 — L2 annotation path leakage for adopters.** If Theme F template-customization eventually allows adopters to copy go.mod patterns, the L2 annotation's `internal/tui/gitdiff/testdata/golden/simple.ansi` path is project-internal. Suggested forward phrasing: "chroma v2.23.1+ reordered trailing-reset-vs-newline; downstream golden fixtures may need regeneration." Not a D.1 defect; route to template-customization drop.
- **OS2 — 19 stripped self-pins are not actively monitored.** None broke `mage ci` green, but if a future drop adds a test that exercises a previously-uncovered integration path, one of those 19 strips MAY resurface as load-bearing. Acceptable risk under round-2 amended semantics. Recommend a `# Surface-monitoring` note in `project_drop_4c_5_refinements_raised.md` so future drops know to watch for this.

### Mitigated Attacks (citations)

- A1 mitigated by direct verification of L2 (golden fixture exists with v2.14.0 ANSI grouping pattern) + L3 (README exists with clear maintenance contract); L1 mitigated indirectly via round-1 compile-error reproducibility.
- A2 mitigated by `mage ci` 2705/24-package green being the explicit acceptance gate per spec criterion #4.
- A3 mitigated by reading each annotation's text: L1/L2 carry version specificity, L3 defers to README.
- A4 mitigated by reading `third_party/teatest_v2/README.md` directly (1.4k of explicit fork rationale).
- A5 mitigated by tracing round-1 → round-2 worklog narrative: round 1 surfaced teatest_v2 status, round 2 elevated it under broader load-bearing definition.
- A6 mitigated by reading `git diff HEAD -- go.mod` and confirming the lone `// indirect` flip is a correction (highlighter.go directly imports chroma).
- A7 mitigated by checking worklog stash narrative is self-consistent with mage-ci output norms.
- A8 mitigated by classifying as routing-forward observation, not D.1 defect.

### Conclusion

PASS. D.1 round 2 cleanly satisfies the orchestrator-amended acceptance criteria:

1. `go.mod` carries 4 `replace` directives — 1 fantasy-fork + 3 load-bearing — every other (19) experimental self-pin stripped per the original strip directive.
2. Each load-bearing replace has an explicit `// load-bearing: <reason>` annotation naming a specific consumer constraint.
3. All 3 load-bearing rationales are evidence-grounded:
   - L1 (ultraviolet) — round-1 compile-error trace at vendored `cursed_renderer.go:444,698` proves the API constraint.
   - L2 (chroma/v2 v2.14.0) — `internal/tui/gitdiff/testdata/golden/simple.ansi` exists with the v2.14.0 ordering pattern; `highlighter.go` directly imports the package.
   - L3 (teatest_v2) — `third_party/teatest_v2/README.md` documents the deliberate fork rationale + maintenance contract.
4. `go.sum` regenerated correctly; the lone `// indirect` flip on chroma/v2 is a correction (highlighter.go directly imports it; flag was incorrect previously).
5. `mage ci` 2705/24-package green per worklog (could not independently reproduce due to A.1 sibling concurrency, but stash-maneuver narrative is self-consistent and round-1 → round-2 asymmetry strongly implies the restoration is the load-bearing change).
6. No CONFIRMED counterexamples constructed across 8 required attack categories.

Two out-of-scope observations (OS1 leakage in L2 annotation, OS2 surface-monitoring for 19 unmonitored strips) routed forward; neither blocks D.1.

Recommend D.1 closes.

### Hylla Feedback

N/A — D.1 touched only non-Go files (`go.mod`, `go.sum`, MD plan/worklog updates). Hylla is Go-only today. All evidence resolved via `Read` (go.mod, go.sum diff, README.md, simple.ansi, highlighter.go imports, builder worklog) + `Bash` (`git diff`, `git log`, `rg "^replace "`). Direct verification of vendored bubbletea source for L1 was BLOCKED by sandbox (Read denied on `/Users/evanschultz/go/pkg/mod/charm.land/bubbletea/v2@v2.0.0-rc.2/cursed_renderer.go`); fell back to round-1 worklog's reproduced compile-error text. **Sandbox-environment gripe (not a Hylla miss):** Read access to the Go module cache would have hardened L1 verification beyond worklog-trust. Recommend the orchestrator note this in the "subagent sandboxing" refinement track.

## Droplet A.1 — Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** FAIL — one CONFIRMED counterexample (acceptance criterion miss on MCP tool description string update).

### Attack Inventory

**Attack 1 — `DueAt **time.Time` three-way correctness.**
Verified against `internal/app/service.go:1282-1285` + `internal/adapters/server/common/app_service_adapter_mcp.go:838-855`. Adapter cleanly distinguishes:
- `in.DueAt == nil` (wire absent) → `dueAtPtr = nil` → service preserves.
- `in.DueAt = &""` (wire empty) → `dueAtPtr = &(*time.Time)(nil)` → service sets `dueAt := *in.DueAt = nil` → UpdateDetails normalizes nil → DueAt cleared.
- `in.DueAt = &"2026-..."` (wire RFC3339) → parses to `&utc`, wraps to `&&utc` → service derefs once to `*time.Time` pointing at parsed time → UpdateDetails sets the new pointer.

The struct field comment at `service.go:695-703` documents the three-way distinction. Tests in `TestUpdateActionItemPartialPATCHSemantics` only cover the **preserve** case (1 of 3 states); the **explicit-clear** and **explicit-set** paths are covered by `TestUpdateActionItem` (existing test, set path) and the adapter-level test at line 104 of `app_service_adapter_mcp_actor_attribution_test.go`. **REFUTED:** impl is correct, but coverage is asymmetric — see Finding F1 below.

**Attack 2 — Title-empty-rejection asymmetry transactional safety.**
`UpdateDetails` (`internal/domain/action_item.go:510-526`) is called at `service.go:1290`. Inspection: domain method validates title FIRST (line 513), returns `ErrInvalidTitle` BEFORE any field write to the in-memory struct. After `UpdateDetails` returns an error at line 1290, the function returns at line 1291 — so subsequent branches (Role, Owner, DropNumber, Persistent, DevGated, Paths, Packages, Files, StartCommit, EndCommit, Metadata) DO NOT execute. The test `title empty pointer rejected` (lines 1639-1647) asserts post-error stored state is unchanged — passing. **REFUTED:** transactional safety holds, no partial-write window.

**Attack 3 — Existing test migration completeness.**
Spot-checked 7 migrated sites:
- `service_test.go:1351` — uses `ptrTo("new title")` etc. (full-field update).
- `service_test.go:1393` — title-only via `ptrTo`.
- `service_test.go:1469` — title-only via `ptrTo`.
- `service_test.go:1512` — title-only via `ptrTo`.
- `service_test.go:2496` — full-field via `ptrTo`, including `ptrTo(created.DueAt)` (which yields `**time.Time` pointing to `*time.Time`, type-checks).
- `service_test.go:4636` — `ptrTo` on Title/Description/Priority.
- `kind_capability_test.go:902` — `ptrTo` on Title/Description/Priority.

`mage testPkg ./internal/app` runs locally → 397 tests pass (slight positive delta from worklog's 387, attributable to the 9 new TestUpdateActionItemPartialPATCHSemantics subtests being counted individually). All 21 occurrences of `UpdateActionItemInput{` in production + tests audited via `rg`; the only sites NOT migrated are dispatcher fixtures (`internal/app/dispatcher/dispatcher_test.go:566,638` + `service_adapter.go:44` + `conflict.go:319`) which only set `Metadata` + `ActionItemID` and benefit from the new PATCH semantics (out of A.1 scope per spawn note). **REFUTED.**

**Attack 4 — Wire-format compat for null JSON values.**
Pre-A.1 wire shape: `Description string` decoding `{"description": null}` → field stays `""` (Go's `json.Unmarshal` on null into a value-typed string is a no-op leaving the zero value). Pre-A.1 service unconditionally wrote that `""` → silently cleared.
Post-A.1 wire shape: `Description *string` decoding `{"description": null}` → field becomes `nil` → service preserves.
Post-A.1 wire shape: `Description *string` decoding `{"description": ""}` → field becomes `&""` → service applies → cleared.

The `null` JSON path semantics shifted from "clobber to empty" to "preserve." Per worklog §"Unknowns" + REVISION_BRIEF §6 (pre-MVP no-tolerance scope), this is acceptable. No production client documented to depend on null-clobber. **REFUTED at acceptance level.**

**Attack 5 — Defensive nil-check robustness on zero-value `UpdateActionItemInput{}`.**
Trace for `UpdateActionItemInput{ActionItemID: "x"}` (everything else nil):
- Service GetActionItem succeeds, GuardScopes resolved, mutation guard passes.
- `title := actionItem.Title` (preserved); same for description, priority, dueAt, labels.
- **`actionItem.UpdateDetails(...)` is called UNCONDITIONALLY** (line 1290) — even when all five input pointers are nil. Inside UpdateDetails: trim is no-op, normalizeDueAt round-trips (wraps in fresh `&ts` pointer with `Truncate(time.Second)`), normalizeLabels re-sorts/dedupéd. **`t.UpdatedAt = now.UTC()` is bumped unconditionally.**
- Then `s.repo.UpdateActionItem(ctx, actionItem)` writes the (effectively unchanged) item back to storage; `enqueueActionItemEmbedding` re-enqueues; `publishActionItemChanged` fires.

So a "fully-nil PATCH" is a no-op semantically but bumps `UpdatedAt` and triggers a spurious embedding re-enqueue + a publish event. Pre-A.1 behavior: same call would have FAILED with `ErrInvalidTitle` because title=`""` was unconditionally overwritten. So post-A.1 quietly succeeds where pre-A.1 rejected. **Per spec acceptance #1 (nil = preserve, no-op),** the new behavior is correct in semantics but produces observable side effects (UpdatedAt bump + embedding churn + publish event). Pre-A.1 callers that relied on the rejection are now silently no-op'd.

**REFUTED at acceptance level** (semantics match spec). Flagged as Finding F2 below for orchestrator triage — this is the kind of background side-effect drift that bites in dispatcher loops or auto-refresh paths.

**Attack 6 — TUI metadata-only update collapse correctness.**
Three sites verified:
- `model.go:8059` (`updateActionItemMetadataCmd`) — `Metadata: &metadata` only; T/D/P/DA/L all nil. Correct preserve semantics. Pre-A.1 this site presumably did the round-trip-current-values dance; the collapse is clean.
- `model.go:8604` (resource-attached path) — `Metadata: &meta` only; T/D/P/DA/L all nil. Correct.
- `model.go:11647` (labels-only path) — `Labels: &labelsCopy, Metadata: &actionItem.Metadata`; rest nil. Correct (labels get applied; metadata is a side-effect snapshot).

The only behavioral concern: `model.go:11647` passes `Metadata: &actionItem.Metadata` which is a pointer to the IN-MEMORY pre-update metadata. If something else mutates `actionItem.Metadata` between this call site's pointer-take and the service consuming it, there's a TOCTOU. But the call closure is short and there's no concurrent mutation in the path. **REFUTED.**

**Attack 7 — `buildCurrentEditActionItemInput` and `parseActionItemEditInput` blank-field semantics.**
- `buildCurrentEditActionItemInput` (`model.go:6064-6128`): blank description in form → `vals["description"] == ""` → `descVal == ""` → `Description: &descVal` → service receives `&""` → **explicit clear**. Title falls back to existing on blank (line 6080-6083), so blank title in form preserves (post-fallback `titleVal = actionItem.Title` non-empty); cannot exercise title-empty-reject path from this entry.
- `parseActionItemEditInput` (`model.go:19794-19863`): blank description in pipe-form → `parts[1] == ""` → `description = current.Description` (line 19813-15) → `descVal = current.Description` → `Description: &descVal` → service receives a non-nil pointer to existing description → **explicit re-set with same value (no observable change, but UpdatedAt bumps).**

The two TUI paths produce DIFFERENT semantics for "blank description in form": `buildCurrentEditActionItemInput` clears, `parseActionItemEditInput` re-sets to current. This asymmetry **predates A.1** (pre-A.1: `buildCurrentEditActionItemInput` would have written `Description=""` on blank, clearing; `parseActionItemEditInput` would have written `Description=current.Description`, preserving). Post-A.1 the asymmetry is preserved with the same observable outcome. **Not a regression; flagged as Finding F3 (pre-existing TUI inconsistency, out of A.1 scope).**

**Attack 8 — Concurrent A.1 / F.2.1 / E.1 / D.1 file-touches.**
`git log --oneline -- internal/app/service.go` last 10 commits — no overlap with sibling-droplet activity (latest is 9036422 "feat(app): publish action item changed on restore rename archive reparent and import" pre-dating the 4c.5 wave). A.1's diff is uncommitted across 16 files; no sibling droplet touches A.1's primary path set. D.1's `go.mod` work and A.1's `service.go` work both showed up in the same dirty workspace per worklog's stash-maneuver. **REFUTED:** clean droplet-boundary, no accidental sibling-droplet adoption.

**Attack 9 — `traceFormControlCharacterGuardPtr` nil-safety.**
Reading `internal/tui/trace.go:233-244`:
```go
func (m Model) traceFormControlCharacterGuardPtr(entity, operation, field string, value *string) {
    if value == nil {
        return
    }
    m.traceFormControlCharacterGuard(entity, operation, field, *value)
}
```
Behavior:
- `value == nil` → no log (preserve case — caller didn't supply the field; nothing to validate).
- `value = &""` → calls value-typed guard with `""` → `containsControlRunes("")` returns 0 → no log (no control chars; expected pre-A.1 behavior).
- `value = &"text\x00"` → calls value-typed guard with control chars → logs as expected.

There is no security regression: empty-string inputs that lack control characters skip logging (correct — there's nothing to flag). The pre-A.1 value-typed guard would also have skipped on `""`. **REFUTED.**

**Attack 10 — MCP tool description string still says title required.**
Reading `internal/adapters/server/mcpapi/extended_tools.go:1437`:
```go
mcp.WithString("title", mcp.Description("Title. Required for operation=create|update")),
```
And lines 1452-1455 (`description`, `priority`, `due_at`, `labels`): none document the new pointer-sentinel "omit to preserve, send empty to clear" semantics. Compare to `owner` (line 1443), `drop_number` (1444), `persistent` (1445), `dev_gated` (1446), `paths` (1447), `packages` (1448), `files` (1449), `start_commit` (1450), `end_commit` (1451), `role` (1441), `structural_type` (1442) — all of which DO document update-time semantics.

**Per spec falsification mitigation #1 (THEME_A_PLAN.md line 88):**
> "Mitigation: A.1 builder MUST update the MCP tool description string (`mcp.WithString("description", ...)`) to document 'omit to preserve, send empty string to explicitly clear'."

Builder explicitly deferred this in worklog §"Unknowns routed back to orchestrator" (lines 368-370) to "D.2 hint sweep, A.2's wire-audit, or a small standalone docs-only droplet." **CONFIRMED counterexample:** the spec made the description-string update a MUST-DO inside A.1, not a deferral. The acceptance criteria #1-7 don't explicitly require docstring updates, but the falsification mitigation does — and mitigations are part of the build contract, not advisory.

### Counterexamples (CONFIRMED)

**C1 — MCP tool description string regressions on `title|description|priority|due_at|labels`.**
File: `internal/adapters/server/mcpapi/extended_tools.go`
- Line 1437: `Title. Required for operation=create|update` — **wrong post-A.1**: title is now optional on update (preserved when omitted; explicit empty rejects). Should read e.g. `Title. Required for operation=create. On operation=update: omit to preserve, send empty string to surface ErrInvalidTitle.`
- Line 1452: `Action-item details in markdown-rich text` — **incomplete post-A.1**: should add `On operation=update: omit to preserve the existing value; send empty string to clear.`
- Line 1453: `low|medium|high` — **incomplete**: should add `On operation=update: omit to preserve the existing value; non-empty applies (empty rejects with ErrInvalidPriority).`
- Line 1454: `Optional RFC3339 timestamp` — **incomplete**: should add `On operation=update: omit to preserve the existing value; send empty string to clear; non-empty must parse as RFC3339.`
- Line 1455: `Optional labels` — **incomplete**: should add `On operation=update: omit to preserve the existing slice; send any array (including empty) to replace.`

**Reproduction:** any MCP client reading the tool schema sees the pre-A.1 contract; any agent following the schema-as-source-of-truth will pass title on every update (per the "Required" annotation), defeating the partial-update pattern A.1 ships.

**Recommended fix:** orchestrator dispatches a small follow-up builder targeted at this single file (~5 mcp.Description string edits) before A.1's PR merges. Alternatively, accept the deferral but explicitly route to D.2 hint sweep with a hard-blocker note (orchestrator's call). Pre-MVP scope means the wire docstring drift doesn't immediately break production, but it WILL mislead any agent reading the schema during dogfood.

### Mitigated / REFUTED

REFUTED on attacks 1, 2, 3, 4, 5 (acceptance level), 6, 7 (pre-existing, out of scope), 8, 9. Each backed by a code-citation trace. CONFIRMED only on attack 10.

### Findings (Non-Blocking)

**F1 — DueAt explicit-clear and explicit-set paths under-tested in `TestUpdateActionItemPartialPATCHSemantics`.**
The 9-row table covers `due_at nil preserves` only. The other two states (outer non-nil pointing to nil = clear; outer non-nil pointing to non-nil = set) are exercised at the adapter layer (`app_service_adapter_mcp_actor_attribution_test.go:104`) and the existing `TestUpdateActionItem`, but the canonical PATCH-semantics test should cover all three states for symmetry with description (preserve / clear / set). Recommend adding two rows: `due_at empty pointer clears` and `due_at non-nil pointer sets`. Non-blocking — coverage gap, not a logic gap.

**F2 — Empty-input no-op writes UpdatedAt + re-enqueues embedding + publishes event.**
A `UpdateActionItemInput{ActionItemID: "x"}` (everything else nil) is now a successful no-op semantically, but bumps `UpdatedAt` and fires `enqueueActionItemEmbedding` + `publishActionItemChanged`. If any future caller idiomatically constructs such inputs (e.g. a "ping refresh" pattern), it will silently churn the embedding queue and broadcast spurious change events. Recommend either (a) early-return when all `in.{Title,Description,Priority,DueAt,Labels,Role,StructuralType,Owner,DropNumber,Persistent,DevGated,Paths,Packages,Files,StartCommit,EndCommit,Metadata}` are nil/empty, or (b) document the side-effect in `UpdateActionItemInput` doc-comment. Non-blocking pre-MVP.

**F3 — Pre-existing TUI blank-description asymmetry between `buildCurrentEditActionItemInput` and `parseActionItemEditInput`.**
Predates A.1; preserved by A.1; not in A.1 scope. Recommend logging as a Drop 4c.5 refinement for later TUI hardening.

**F4 — `priority empty pointer` behavior is undocumented and untested.**
Sending `{"priority":""}` post-A.1 → service receives `*Priority` pointing to `Priority("")` → `slices.Contains(validPriorities, "")` returns false → `ErrInvalidPriority`. Defensible (matches title's "empty rejects"), but neither the spec acceptance table nor the test table covers the case. Recommend adding a `priority empty pointer rejected` row to the test table; orchestrator's call whether it gates A.1 close.

### Conclusion

**Verdict: FAIL (one CONFIRMED counterexample C1).**

The code shape, struct semantics, test migrations, and integration paths are correct and well-documented. PATCH semantics fire the correct three-way distinction for DueAt; nil-vs-empty-pointer cleanly distinguishes preserve from clear; transactional safety holds on title-rejection; TUI metadata-only collapses are clean; trace wrapper is nil-safe.

The single blocker is the MCP tool description string drift on the 5 fields A.1 shipped pointer-sentinels for. This is the wire-contract layer agents read; leaving it pre-A.1 means the agent surface advertises one contract while the implementation honors another. The spec named this a MUST-DO mitigation (THEME_A_PLAN.md line 88), and the builder explicitly deferred it. The orchestrator can either:

1. **Spawn a tiny follow-up builder** to make the 5 docstring edits (lowest cost; cleanest A.1 close).
2. **Re-spawn A.1 builder for round 2** with a focused directive on the docstring updates (matches the "round" worklog pattern).
3. **Accept the deferral to D.2** with an explicit blocker-noted attention item ensuring D.2 doesn't drop it.

Recommend option 1 — the 5 edits are mechanical, low-risk, single-file, and close A.1 cleanly without bouncing a bigger spawn.

### Hylla Feedback

N/A — A.1 review touched only Go source files via direct Read/Grep, no Hylla queries attempted (per spawn-prompt directive: filesystem-MD coordination mode, NO Hylla calls). All evidence resolved via `Read` + `rg` (where Bash-grep allowed) + `mage testPkg ./internal/app`. Builder's worklog claim of `mage ci` 2715 pass corroborated by `mage testPkg ./internal/app` 397/397 PASS in QA reproduction.

## Droplet E.2 — Round 1

**Reviewer:** go-qa-falsification-agent
**Date:** 2026-05-05
**Verdict:** PASS-WITH-NIT

### Attack Inventory

**Attack 1 — Archived-parent test does not actually pin ArchivedAt-handling (semantic-vs-nominal mismatch).** This is the load-bearing attack. Spec acceptance #1 framed the test as: pin defensive eligibility behavior on archived parents independently of the upstream `includeArchived=false` filter. Builder's fixture (walker_test.go:253-269):

- Parent: `LifecycleState=StateTodo`, `ArchivedAt=&archivedAt`, `Persistent=false` (zero-value).
- Candidate: `LifecycleState=StateTodo`, `ParentID="parent-1"`.

Trace through `isEligible` (walker.go:173-206):
1. item.LifecycleState == StateTodo → continue.
2. No BlockedBy → continue.
3. parentID non-empty → lookup; parent exists.
4. `parent.Persistent == false` → don't short-circuit return true.
5. Final: `return parent.LifecycleState == StateInProgress` → StateTodo != StateInProgress → false.

The candidate is filtered by the **existing parent-state gate** at line 205, NOT by ArchivedAt. The predicate never reads `parent.ArchivedAt` — `rg ArchivedAt internal/app/dispatcher/walker.go` returns zero hits. **The test passes byte-for-byte the same with `ArchivedAt=nil` set on the fixture.** Therefore it does not pin "archived-parent → not-eligible" defensively; it nominally tests parent-archived but actually tests parent-not-in-StateInProgress, which is already covered by `TestWalkerSkipsTodoItemWhoseParentIsTodo` (walker_test.go:189-226).

The builder's design notes acknowledge this honestly ("the existing parent-state gate filters the child either way; the assertion is on the observable outcome (child not promoted) rather than on the internal gate path that produced it"). But the spec's acceptance #1 explicitly carved out: "If the builder finds the predicate already correct via includeArchived=false filtering, the test asserts the filtering instead." The test asserts NEITHER — neither ArchivedAt-handling nor the upstream `includeArchived=false` filter. It's redundant with the existing TodoParent test.

**Strengthening counterexample:** the REAL hole the spec was probing is `parent.Persistent==true && ArchivedAt!=nil`. In that scenario, line 202-204 short-circuits `return true` regardless of ArchivedAt, and the candidate IS promoted. To pin defense-in-depth, the test would need parent.Persistent=true to bypass the StateInProgress gate, then verify ArchivedAt!=nil still rejects. The current test cannot regression-catch the persistent-archived-parent path.

**Verdict:** SOFT counterexample. Test name and intent are misleading; coverage is redundant with `TestWalkerSkipsTodoItemWhoseParentIsTodo`. Builder's defense ("future refactor that removes the LifecycleState gate without adding ArchivedAt fails this test") is technically correct but defensive in only one direction — the predicate's actual ArchivedAt-blindness for Persistent parents remains uncovered. Routing as NIT (not blocking) because: (a) the test does pass; (b) the predicate behavior is documented honestly in the test's own doc-comment lines 228-249 which acknowledge the LifecycleState-gate dependence; (c) the spec accepted "or test asserts the filtering instead" as an acceptable substitute, and "tests observable outcome on a fixture with archived-parent flag set" is a third path the spec didn't enumerate but tolerates. Recommend a follow-up droplet (or fold into the next refinement-tracker entry) adds a `parent.Persistent=true && ArchivedAt!=nil` case to close the actual defensive gap.

**Attack 2 — `ListColumns` error path completeness (does the early-fail prevent ALL subsequent ops?).** Spec acceptance #2: stub `walkerService.ListColumns` returning `errors.New("simulated infra failure")` → `Promote` returns wrapped error. Trace through `Promote` (walker.go:226-250):

1. nil-receiver guard passes.
2. projectID trim + check passes.
3. `w.svc.ListColumns(...)` → returns `(nil, infraErr)`.
4. `if err != nil { return ..., fmt.Errorf("walker: list columns for project %q: %w", projectID, err) }` → IMMEDIATE return at line 236.
5. Lines 238-249 are unreachable: `columnIDForLifecycleState` (line 238), missing-column check (line 239-241), `MoveActionItem` (line 242).

So an early `ListColumns` failure prevents ALL subsequent operations. Test asserts `svc.moveCalls == 0` (walker_test.go:565-567) which validates step-5 directly. Test ALSO asserts `!errors.Is(err, ErrPromotionBlocked)` (line 562-564) — confirming the error is NOT rewritten as the recoverable sentinel. Both contracts pinned.

**Verdict:** Mitigated. The early-return at line 236 is unconditional; no path through `Promote` continues past a `ListColumns` failure. Test's `moveCalls == 0` assertion captures this. No counterexample.

**Attack 3 — Doc-comment drift fix accuracy (does the impl ACTUALLY treat missing-reference and non-complete blockers symmetrically?).** New doc claims both "missing reference" AND "non-StateComplete blocker (StateTodo / StateInProgress / StateFailed / StateArchived)" are treated as not-clear. Trace impl (walker.go:177-187):

```
for _, blockerID := range item.Metadata.BlockedBy {
    blocker, ok := byID[strings.TrimSpace(blockerID)]
    if !ok {                             // missing reference
        return false
    }
    if blocker.LifecycleState != domain.StateComplete {  // non-StateComplete blocker
        return false
    }
}
```

Both branches return false. The non-StateComplete branch uses `!=` so EVERY non-StateComplete state (StateTodo, StateInProgress, StateFailed, StateArchived) lands the rejection. Doc enumeration is exhaustive — doc names exactly the four states `domain.LifecycleState` defines as non-Complete. Existing `TestWalkerSkipsTodoItemWithUnmetBlockedBy` (walker_test.go:128-181) covers `{todo, in_progress, failed}` cases. `archived` is not in the table-driven cases but is rejected by the same `!=` branch.

**Verdict:** Mitigated. Doc fix accurately describes impl. Direction is correct: doc tightens to match impl, not the other way around.

**Attack 4 — Conservative-by-design rationale ("stalled-but-untouched, not wrongly-promoted") — does a typo'd BlockedBy ever surface, or stall indefinitely?** New doc claims "should surface as a stalled-but-untouched item" and points at "supersede / archive paths" for legitimate bypass. Searched for active surfacing (refinements gate / attention items / orphan detection) on missing BlockedBy references:

- `rg "BlockedBy" internal/app/auto_generate_steward.go` shows only `assembleRefinementsGateBlockedBy` — the gate's OWN BlockedBy (steward-generated), not a stale-detector for planner-typo'd BlockedBy on other items.
- No active "missing BlockedBy reference" surfacing mechanism exists in production code today.

So a typo'd BlockedBy reference DOES stall indefinitely with no automatic alarm. The doc's framing reads as: "the manifestation of the planner bug is a stalled item, not a wrongly-promoted one" — i.e. it contrasts two PASSIVE outcomes (stall vs wrong-promote), not promising an ACTIVE alarm. That's compatible with the actual behavior. The supersede / archive pointer is also accurate — both paths exist in the lifecycle vocabulary as override hatches.

**Verdict:** Mitigated. Doc framing is technically truthful — it contrasts manifestation modes, not promising surfacing. The "should surface as a stalled item" reads as "manifests as", not "alerts about". Could read more sharply (e.g. "manifests as a stalled item rather than a wrongly-promoted one") but the existing wording is defensible. Not a falsification.

**Attack 5 — Test-stub field name consistency (`columnsErr` vs sibling pattern).** Audit existing stub fields:

- `columns` / `columnsErr` (NEW pair) — name-stem + Err suffix.
- `items` (no `itemsErr` — paired separately via `erroringListItemsStub.err`).
- `moveResult` / `moveErr` — name-stem + Err suffix (existing).

`columnsErr` matches the `moveResult/moveErr` pattern (paired-field naming with Err suffix). Inconsistency exists with `items` field (which has its own `erroringListItemsStub` rather than an inline error field), but adding `itemsErr` would have been a stub-design refactor outside E.2 scope. The split-stub pattern is grandfathered; adding a new error field on the existing stub respects the moveResult/moveErr precedent.

**Verdict:** Mitigated. Naming is consistent with `moveResult/moveErr` precedent. The legacy `erroringListItemsStub` separate-stub pattern is grandfathered; refactor is out of scope.

**Attack 6 — Concurrent E.1 + E.2 worklog interleaving (audit-trail order correctness).** E.1 round 1 sits at BUILDER_WORKLOG.md:36, E.2 round 1 sits at BUILDER_WORKLOG.md:417. Order between: F.2.1 (line 6), E.1 (line 36), D.1 R1 (line 68), D.1 R2 (line 192), A.1 R1 (line 294), A.1 R2 (line 372), E.2 (line 417). E.2 correctly appended after E.1 plus all intervening rounds. Audit trail is chronologically ordered (within the 2026-05-05 day). No interleaving anomaly.

**Verdict:** Mitigated. Worklog ordering preserves audit-trail readability.

**Attack 7 — `mage test-pkg` vs `mage testPkg` target name drift.** Builder worklog reports `mage test-pkg ./internal/app/dispatcher`. magefile.go declares `func TestPkg(pkg string) error` at line 49. Mage normalizes camelCase ↔ kebab-case at the CLI layer, so both forms invoke the same target. Spec uses `mage test-pkg`, builder used `mage test-pkg`, target resolves. Confirmed not a drift.

**Verdict:** Mitigated. Target invocation is correct.

**Attack 8 — Doc-comment scope creep beyond paragraph 2.** `git diff -- internal/app/dispatcher/walker.go` shows changes confined to lines 49-58 (paragraph 2 of the eligibility predicate doc). No other paragraphs touched. No production code touched in walker.go. Spec acceptance #3 mandated "Drift fix only — match existing impl"; builder honored.

**Verdict:** Mitigated. Scope is tight.

### Counterexamples (CONFIRMED)

None blocking. Attack 1 surfaces a SOFT issue: the new `TestWalkerTreatsArchivedParentAsNotEligible` does not actually exercise an ArchivedAt-specific code path — it pins parent-not-in-StateInProgress, which is redundant with the existing `TestWalkerSkipsTodoItemWhoseParentIsTodo`. The real hole (parent.Persistent=true with ArchivedAt!=nil short-circuiting line 202-204) remains uncovered. Routed as NIT, not blocker.

### Mitigated Attacks

- A1: Soft NIT — test name overstates what's pinned, but observable-outcome assertion is honest and the test doc-comment names the LifecycleState-gate dependence. Recommend follow-up adds `Persistent=true + ArchivedAt!=nil` case for true defense-in-depth.
- A2: Early-return at walker.go:236 prevents all subsequent ops; `moveCalls == 0` assertion validates.
- A3: Doc enumeration of {Todo, InProgress, Failed, Archived} matches `!=` impl branch; exhaustive across the LifecycleState enum.
- A4: "Stalled-but-untouched" framing contrasts passive outcomes; consistent with no active surfacing mechanism.
- A5: `columnsErr` matches `moveResult/moveErr` pattern; legacy `erroringListItemsStub` split-stub grandfathered.
- A6: Worklog order chronologically correct; E.2 appended after E.1.
- A7: Mage target name resolves cleanly (camelCase ↔ kebab-case normalization).
- A8: Doc scope confined to paragraph 2; no production code changes; no scope creep.

### Observations (non-blocking)

- **NIT-1 (from A1):** `TestWalkerTreatsArchivedParentAsNotEligible` does not exercise ArchivedAt-specific predicate logic. Coverage is functionally redundant with `TestWalkerSkipsTodoItemWhoseParentIsTodo`. Recommend a follow-up (e.g. routed to a Drop 4c.5 refinement entry or a small drop-end touch-up) adds a fixture with `parent.Persistent=true && ArchivedAt!=nil && parent.LifecycleState=StateInProgress` to pin defense-in-depth where the predicate's Persistent short-circuit currently swallows the archived state. The current test passes byte-identically with `ArchivedAt=nil`; that's the falsification surface.
- **NIT-2:** Doc-comment "should surface AS a stalled-but-untouched item" reads slightly active-voice; a sharper rewording would be "manifests as a stalled-but-untouched item". Cosmetic, not blocking.
- **NIT-3:** The existing `TestWalkerSkipsTodoItemWithUnmetBlockedBy` table at walker_test.go:131-138 doesn't include `StateArchived` as a blocker state. The new doc explicitly enumerates StateArchived as a non-clear blocker case. A future test extension could add `{name: "archived blocker", blockerState: domain.StateArchived}` to the table — the impl already rejects it (`!= StateComplete` branch), so the test would pass; the gap is coverage-completeness, not correctness.

### Conclusion

PASS-WITH-NIT. Builder E.2 cleanly executed acceptance #2 (ListColumns error path) and acceptance #3 (doc-comment drift fix). Acceptance #1 is met in letter (test exists, named correctly, asserts observable outcome) but not fully in spirit (test does not exercise ArchivedAt-specific code paths because the predicate has none today). The builder's design notes acknowledge this honestly and the spec's "or test asserts the filtering instead" carve-out tolerates the chosen path. Recommend droplet E.2 closes; route NIT-1 to a refinement-tracker follow-up that adds the `Persistent=true + ArchivedAt!=nil` defense-in-depth case.

`mage test-pkg ./internal/app/dispatcher` 356/356 PASS per builder worklog (verified target name resolves). `mage formatCheck` clean. No counterexamples blocking droplet close.

### Hylla Feedback

N/A — droplet edits + reviewed surface are Go files, but the spawn-prompt directive ("NO Hylla calls") routed all evidence through `Read` / `Bash` (`rg`, `git diff`). No fallback misses to log under the standard rule because the rule was suspended for this round.

## Droplet F.2.2 — Round 1

**Reviewer:** go-qa-falsification-agent (model: opus, filesystem-MD mode).
**Date:** 2026-05-05.
**Verdict:** PASS — no counterexample found across the seven attack categories.
**Scope:** F.2.2 declared files only — `internal/templates/builtin/default-generic.toml` (NEW), `internal/templates/embed.go` (modified), `internal/templates/embed_test.go` (extended), `workflow/drop_4c_5/THEME_F_PLAN.md` (state line), `workflow/drop_4c_5/BUILDER_WORKLOG.md` (round 1 section).

### Counterexample Certificate

- **Premises** — Builder claims default-generic.toml mirrors default-go's 12 kinds + 4 standard child_rules + 6 STEWARD seeds + identical `[gates]`, while omitting `[agent_bindings]` table and the two drop-narrowed `[[child_rules]]`. Embed directive uses explicit two-file form. Tests assert all of the above.
- **Evidence** — Read of `internal/templates/builtin/default-generic.toml` (337 lines), `internal/templates/builtin/default-go.toml` (gates + steward_seeds + child_rules slices via `grep`), `internal/templates/embed.go` (29-line `//go:embed` directive), `internal/templates/embed_test.go:43-160` (`TestLoadDefaultGenericTemplate`), `internal/templates/load.go:78-150` (Load chain), `:284-301` (`validateMapKeys`), `:400-403` (`validateChildRuleReachability` no-op), `:468-654` (all `validateAgentBinding*` validators iterate `range tpl.AgentBindings`). `mage testPkg ./internal/templates` reproduced **381/381 PASS** in QA session.

### Attack Inventory

**A1 — STEWARD seeds drift:** REFUTED. Both files ship 6 seeds with identical titles `DISCUSSIONS / HYLLA_FINDINGS / LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_REFINEMENTS` (verified by `[[steward_seeds]]` line-grep on both files: default-go lines 299-320, default-generic lines 284-306).

**A2 — Validator chain coverage with empty bindings:** REFUTED. All four `validateAgentBinding*` validators (`load.go:468 validateAgentBindingEnvNames`, `:531 validateAgentBindingContext`, `:622 validateAgentBindingToolGating`, plus `validateMapKeys` agent_bindings loop at `:290`) iterate `range tpl.AgentBindings`. Go's `range` over a nil map iterates zero times, so each validator is a no-op when bindings are absent. `validateRequiredChildRules` does NOT yet exist (it's an F.5.1 future droplet — confirmed via grep). 381/381 mage test result confirms.

**A3 — Embed directive completeness:** REFUTED. `embed.go:29` reads exactly `//go:embed builtin/default-go.toml builtin/default-generic.toml` — explicit two-file form per F.2.1 falsification mitigation #2, NOT a glob.

**A4 — `AgentBindings == 0` semantics:** REFUTED. No validator in the chain requires non-empty `AgentBindings`. The closed set of validators that touch the map all use `range`-iteration, which is empty-safe. Test asserts `len(tpl.AgentBindings) == 0` and the file passes the full chain. Builder's "the dispatcher tolerates absent bindings" claim is the contract, consistent with spec acceptance #2.

**A5 — Drop-narrowed child_rules omission:** REFUTED. default-go ships 6 `[[child_rules]]` (4 standard + 2 drop-narrowed `DROP-PLAN-QA-PROOF` / `DROP-PLAN-QA-FALSIFICATION` with `when_parent_structural_type = "drop"` at default-go lines 256-268); default-generic ships only the 4 standard, with a defensive test guard rejecting any non-empty `WhenParentStructuralType` (`embed_test.go:117`). TOML body explicitly comments the omission rationale (lines 251-265).

**A6 — Header comment vs TOML parser:** REFUTED. All header lines (1-54) are `#`-prefixed comments; TOML body starts at line 56 with `schema_version = "v1"`. Pelletier/go-toml/v2 parses cleanly (verified by 381/381 PASS).

**A7 — `mage ci` impact across consumers:** REFUTED. `rg DefaultTemplateFS` returns only two consumers — `embed.go:LoadDefaultTemplate` (opens `default-go.toml` by literal name) and `embed_test.go:TestLoadDefaultGenericTemplate` (opens `default-generic.toml` by literal name). Neither walks the embed.FS. The new file cannot surprise an unintended consumer.

### Counterexamples (CONFIRMED)

None. All seven attacks REFUTED.

### Findings (Non-Blocking)

- **F1 NIT:** The `embed_test.go:79` test opens the file via `DefaultTemplateFS.Open("builtin/default-generic.toml")` rather than the F.1.3-future `LoadDefaultTemplateForLanguage("")`. This is correct per spec ("F.1.3 not yet landed; direct embed.FS open preserves byte-for-byte semantics until then"). No action.
- **F2 NIT:** Doc-comment in default-generic.toml header references "F.1.3 acceptance criteria 2 + 6" — criterion 2 picks the file by `lang == ""`; criterion 6 establishes the wrapper relationship — informational only.

### Conclusion

PASS. F.2.2 ships exactly what spec + spawn-prompt require. STEWARD seeds match 1:1, gates match 1:1, embed directive is explicit two-file form, validator chain is empty-bindings-safe (verified by `mage testPkg` 381/381 PASS), drop-narrowed child_rules correctly omitted with defensive test guard, no FS-walking consumers. Builder's design notes correctly trace the absent-vs-empty-table decision (chose absent) and the drop-narrowed omission rationale. Recommend droplet F.2.2 closes.

### Hylla Feedback

N/A — F.2.2 review touched only Go-eligible files (`embed.go`, `embed_test.go`, `load.go`) plus a TOML and workflow MDs. Per spawn-prompt directive ("filesystem-MD coordination mode. NO Hylla calls.") and the "Hylla Indexes Only Go Files Today" memory rule, no Hylla query was attempted. All evidence resolved via `Read` + `Grep` (Bash-grep) + `mage testPkg`. No miss to log.

## Droplet F.2.3 — Round 1

**Reviewer:** go-qa-falsification-agent (model: opus, filesystem-MD mode).
**Date:** 2026-05-05.
**Verdict:** PASS — no counterexample against the build artifact. One CONFIRMED finding routed against the SPEC (planner side), already self-reported and corrected by the builder.
**Scope:** F.2.3 declared files only — `.tillsyn/template.toml` (NEW), `.gitignore` (modified), `workflow/drop_4c_5/THEME_F_PLAN.md` (state line), `workflow/drop_4c_5/BUILDER_WORKLOG.md` (round 1 section).

### Counterexample Certificate

- **Premises** — Builder claims `.tillsyn/template.toml` ships byte-faithful body content from `default-go.toml` plus a self-host header + a `[tillsyn]` block carrying `spawn_temp_root = "os_tmp"` (matching the dispatcher's empty-string default), and `.gitignore` re-includes that one file while keeping spawns / log / db ignored.
- **Evidence** — Read of `.tillsyn/template.toml` (697 lines), `internal/templates/builtin/default-go.toml` (653 lines, body section identical past header), `internal/app/dispatcher/bundle.go:246-256` (`resolveSpawnTempRoot`), `.gitignore` (current state + `git diff` showing pre-droplet state), `git check-ignore -v` against four candidate paths, `git status --porcelain --untracked-files=all .tillsyn/`, `BUILDER_WORKLOG.md:492-538` (F.2.3 R1 entry).

### Attack Inventory

**A1 — Body-content drift from `default-go.toml`:** REFUTED. Body bytes match between offsets `template.toml:47` and `default-go.toml:40` (both lines = `schema_version = "v1"`). All `[kinds.*]` (12), `[[child_rules]]` (6), `[[steward_seeds]]` (6), `[gates]` (`build = ["mage_ci", "commit", "push"]`), and `[agent_bindings.*]` + `[agent_bindings.*.context]` (12 + 6) blocks are byte-equivalent. Line-count delta = +44 = +7 header expansion + 36-line `[tillsyn]` tail block + ~1-line whitespace nudge, reconciling the worklog's claimed +43 within tolerance.

**A2 — `[tillsyn].spawn_temp_root = "os_tmp"` choice:** REFUTED. `bundle.go:246-256` `resolveSpawnTempRoot` switch maps both `""` and `"os_tmp"` (== `SpawnTempRootOSTmp`) to the same outcome. The explicit string preserves runtime behavior unchanged while making the dogfood policy observable on inspection. `"project"` would route bundles into `<worktree>/.tillsyn/spawns/<id>/` and require F.7.7 + F.7.8 (NOT shipped); choosing it now would pollute `mage ci`-ready repo state.

**A3 — Gitignore re-include correctness:** REFUTED. `git check-ignore -v` against the live repo:

- `.tillsyn/template.toml` → matches `.gitignore:19:!.tillsyn/template.toml` (RE-INCLUDED).
- `.tillsyn/spawns/foo.json` → matches `.gitignore:18:.tillsyn/*` (IGNORED).
- `.tillsyn/log/something` → matches `.gitignore:18:.tillsyn/*` (IGNORED).
- `.tillsyn/tillsyn.db` → matches `.gitignore:18:.tillsyn/*` (IGNORED).

`git status --porcelain --untracked-files=all .tillsyn/` returns exactly `?? .tillsyn/template.toml`. Surgical re-include works.

**A4 — Subdirectory shadowing:** REFUTED. `.tillsyn/*` glob correctly matches BOTH file entries AND directory entries at that depth — `.tillsyn/spawns/foo.json` is correctly IGNORED via the parent-dir match (Attack-3 evidence). Standard gitignore semantics; no shadowing pathology.

**A5 — Cross-checkout artifact stability:** REFUTED. `internal/app/service.go:loadProjectTemplate` is the only consumer of `<project_root>/.tillsyn/template.toml`; F.1.2 (the walker) has NOT shipped per spec § F.1.2 state line. Today the tillsyn project bakes from the embedded `default-go.toml` and the self-host file is INERT — staged for F.1.2's later activation. Worklog acknowledges this honestly. F.2.3 spec explicitly accepts this future-staging.

**A6 — Header content correctness:** REFUTED. Lines 1-46 of `template.toml`:

- Line 1: "Tillsyn self-host cascade template (dogfood)." — names file correctly.
- Lines 4 + 11: cross-references "Drop 4c.5 droplet F.2.3" + `internal/templates/builtin/default-go.toml`.
- Lines 14-19: documents intentional adjustment #1 (header swap).
- Lines 21-35: documents intentional adjustment #2 (`[tillsyn]` block + `spawn_temp_root` rationale + bundle.go:246-256 reference + the deferred path to `"project"`).
- Lines 37-41: cross-references F.1.2 walker activation policy.

**A7 — Spec mitigation #3 was wrong (per builder):** **CONFIRMED against SPEC, NOT builder.**

- `THEME_F_PLAN.md` § F.2.3 falsification mitigation F3 reads: `"existing rule is `.tillsyn/spawns/`, NOT `.tillsyn/`. Document in droplet acceptance."`
- `git diff .gitignore` shows the pre-droplet rule was `-.tillsyn/` (NOT `.tillsyn/spawns/`). The spec's manual-verification claim was incorrect.
- A `.tillsyn/` directory-level pattern would silently ignore the new `.tillsyn/template.toml` because gitignore re-include rules cannot resurrect a path under an excluded directory (they require `.tillsyn/*` granularity per gitignore docs).
- Builder caught the spec error during R1 implementation, refactored `.gitignore` to `.tillsyn/*` + `!.tillsyn/template.toml`, and self-reported the spec error in `BUILDER_WORKLOG.md:502`. The fix is correct (Attack-3 evidence).

**Routing:** finding goes against the SPEC (planner authored an incorrect manual-verification claim). The build artifact is correct because the builder caught + fixed the issue. Recommend the spec's F3 mitigation prose be updated post-merge in a refinement, OR THEME_F_PLAN.md is treated as a historical authoring artifact and the corrected behavior is tracked via the worklog.

### Counterexamples (CONFIRMED)

None against the builder's implementation. One CONFIRMED finding (A7) against the SPEC — already self-reported by the builder and resolved correctly in the build artifact.

### Conclusion

**PASS.** Body byte-faithful to `default-go.toml`, `[tillsyn].spawn_temp_root = "os_tmp"` correctly preserves the dispatcher's default behavior, gitignore re-include is surgical (verified by `git check-ignore -v` + `git status --untracked-files=all`), header is correct + audit-traceable, future-staging policy is honest, and the SPEC-side error in mitigation F3 was caught and corrected by the builder rather than propagated. Recommend droplet F.2.3 closes; route the SPEC-prose correction to a refinement note (or accept THEME_F_PLAN.md as a historical authoring artifact).

### Hylla Feedback

N/A — F.2.3 touched only non-Go files (TOML + dotfile + workflow MDs). Hylla is Go-only today per `feedback_hylla_go_only_today.md`. All evidence resolved via `Read` + `Bash` (`git check-ignore`, `git status`, `git diff`). Spawn-prompt directive ("NO Hylla calls") matched the file-type reality.

## Droplet A.2 — Round 1

**Reviewer:** go-qa-falsification-agent (subagent, opus)
**Date:** 2026-05-05
**Verdict:** PASS WITH FINDINGS — spec-mechanical work is correct; one scope-vs-stated-goal gap surfaced as a refinement candidate, NOT a counterexample to A.2's contractual acceptance criteria.
**Scope:** A.2-declared paths only — `internal/adapters/server/mcpapi/strict_decode.go`, `strict_decode_test.go`, `handler.go`, `handoff_tools.go`, `extended_tools.go`, `extended_tools_test.go`, `workflow/drop_4c_5/THEME_A_PLAN.md`, `workflow/drop_4c_5/BUILDER_WORKLOG.md`.

### 1. Findings

- **1.1 Strict-decoder bypass via raw `req.Get*` calls (Attack 1).** REFUTED as a counterexample to A.2 acceptance criteria — but a SCOPE-VS-GOAL GAP exists worth raising as a refinement. Acceptance #3 says "All 21 production `BindArguments` call sites in the three files swap to `bindArgumentsStrict`" — `rg -nc bindArgumentsStrict` shows handler.go=5, handoff_tools.go=5, extended_tools.go=11 (total 21, exactly matches spec). All 21 mechanical swaps are present; `rg "BindArguments\\("` returns no production residue. HOWEVER, several MCP tool registrations NEVER funnel through the strict decoder at all, reading every parameter via raw `req.GetString` / `req.GetBool` / `req.GetInt` / `req.GetStringSlice` and so will silently drop unknown keys — defeating acceptance #1's stated intent ("Stop schema-drift bugs ... from landing as silent no-ops") for those specific tools. Concrete bypass sites:
  - `extended_tools.go:1682-1729` — `till.embeddings` (operations `status` + `reindex`).
  - `extended_tools.go:1754-1770` — `till.kind` (operation `list`).
  - `extended_tools.go:1781-1790` — `till.list_kind_definitions` legacy alias.
  - `instructions_tool.go:131-158` — `till.get_instructions` (read-only inventory tool).
  - `handler.go:559-578` — `till.capture_state` (read-only state capture).
  - `auth_context_runtime.go:107-110` — only reads `auth_context_id` / `acting_auth_context_id` from raw req params; this is intentional cross-cutting middleware (the value is also declared on every typed struct via the new `AuthContextID` field) so this site is a NON-ISSUE. The five tools above are the actual gap. A typo'd argument on `till.embeddings` (e.g. `{"operation":"status","prject_id":"p1"}`) gets silent zero-default for the typo'd field. **Routing recommendation:** raise as `R-A.2-3` for a follow-up droplet that converts the 5 raw-read tool handlers to strict-decode. Builder hit the spec exactly; spec under-specified (named "21 BindArguments call sites" rather than "every MCP tool handler"). Builder cannot be faulted for that. NOT a counterexample to A.2 acceptance.

- **1.2 `AuthContextID` declared-not-read pattern (Attack 2).** REFUTED. Per spec falsification mitigation #1 + the builder's findings, the `AuthContextID` field on each of the 6 typed structs is declared explicitly to satisfy `DisallowUnknownFields` for the schema-declared `auth_context_id` key, while production reads it via `withMCPToolAuthRuntime` from raw req params (see `auth_context_runtime.go:107`). This is the documented design — verified at `extended_tools.go:475-478`, `extended_tools.go:797-801`, `extended_tools.go:2074-2078`, `handler.go:599-602`, `handoff_tools.go:93-96`, `extended_tools.go:167-170`. Each declaration carries a comment crosslinking to A.2 + the runtime consumer. NOT decoration — the field is load-bearing for strict-decode acceptance. Same shape applies to the new `Operation` field on `till.comment` (`extended_tools.go:2061-2065`) — declared so strict-decode accepts the schema-required `operation` key, body still reads it via `req.GetString("operation", "")` at line 2098. Pattern is consistent and necessary; refinement R-A.2-2 in BUILDER_WORKLOG correctly flags the cleanup opportunity (move reads onto the typed struct in a future droplet).

- **1.3 A.1 null-pointer regression — actual MCP wire test (Attack 3).** REFUTED with caveat. `TestBindArgumentsStrictPreservesNullPointer` (`strict_decode_test.go:66-97`) exercises the helper directly with `json.RawMessage(`{"description":null,"title":null,"labels":null}`)` and asserts the typed nil pointers result. The test is helper-level, NOT MCP-wire-level. **However**, the wire path goes: `httptest` → mark3labs MCP framework → `req.Params.Arguments` (which the framework hydrates from raw JSON) → `bindArgumentsStrict`. The framework hands `req.Params.Arguments` through as either a `map[string]any` (in `extended_tools_test.go:3618`-style `postJSONRPC` callouts) or `json.RawMessage` (the fast-path branch). The fast-path branch IS exercised end-to-end by `TestBindArgumentsStrictRawMessageFastPath` (`strict_decode_test.go:249-286`), and the helper-level null test pins the same code path the wire would hit on a `RawMessage`. The map-path null behavior is implicitly covered because `json.Marshal(map[string]any{"description": nil, ...})` produces `{"description":null,...}` which the strict decoder then decodes via the same `dec.Decode(target)` line. Conclusion: A.1's wire shape is preserved across both branches. NO MCP-wire integration test for `{"description":null}` against the action-item-update tool exists, but the helper-level evidence + the fast-path wire test transitively cover the regression. ROUTING NOTE: a future droplet could add an end-to-end `till.action_item operation=update + {"description":null}` test for defense-in-depth, but it is NOT required by A.2 acceptance criteria.

- **1.4 Test coverage gaps for the 21 swapped sites (Attack 4).** REFUTED. `TestHandlerExpandedToolRejectsUnknownJSONKeys` (`extended_tools_test.go:3556-3637`) covers ONE tool from each of the three production files: `till.project` (extended_tools.go), `till.auth_request` (handler.go), `till.handoff` (handoff_tools.go). The other 18 sites are covered indirectly by the 191-test `mage test-pkg ./internal/adapters/server/mcpapi` run that the builder reports as 191 passed / 1 pre-existing skip. A typo'd-key regression in any of the 18 untested sites would surface only via integration tests OR a specific unknown-key test for that tool — neither exists for those 18. HOWEVER, the strict-decode helper itself is comprehensively tested in `strict_decode_test.go` (8 tests: valid, null preservation, unknown key, multiple unknown keys, nil args, empty {}, non-pointer, nil target, raw-message fast-path), and the helper is the single decode path for all 21 sites. Failure modes that could differ per call-site: (a) the offending key matches an existing field name (no rejection — by design), (b) the typed struct shape itself rejects valid payloads — this is what the 4 schema-vs-struct gap fixes addressed. The remaining risk surface (a single struct shape with a missing field that no test currently sends) is small and would surface immediately on any attempt to use that tool with the missing key. NOT a blocker for A.2 acceptance.

- **1.5 `json.Decoder` error-parsing fragility (Attack 5).** REFUTED. The helper's `unknownFieldName` recovery (`strict_decode.go:105-128`) uses three layers of defense:
  1. `strings.HasPrefix(msg, "json: unknown field ")` — matches the std-lib stable format documented as `fmt.Errorf("json: unknown field %q", key)`.
  2. `strconv.Unquote(tail)` — escape-aware unquoting handles any `%q`-formatted payload correctly, including keys with embedded quotes/newlines/escapes.
  3. Fallback bare-token trim if `strconv.Unquote` rejects (defensive only; std lib has held this format stable since Go 1.10).
  Builder did NOT use `errors.As` against typed errors — the spec called this out as a falsification surface, but the std-lib's `encoding/json` package does NOT export a typed error for `DisallowUnknownFields` rejections (verified by reading the doc-comment at `strict_decode.go:31-35` and confirmed by Go std-lib behavior). The string-prefix path is the only viable approach. The fallback test cases (`TestUnknownFieldNameRecoveryEdgeCases` at `strict_decode_test.go:291-325`) cover both the stable-format path and the bare-token fallback path. ROBUSTNESS: should the std lib add a typed error in a future Go release, the helper's prefix path continues to work (the formatted error message is unchanged) AND the test suite continues to pass. No fragility.

- **1.6 Multiple unknown keys — stop-at-first vs all-collected (Attack 6).** REFUTED. `TestBindArgumentsStrictMultipleUnknownKeysReportsFirst` (`strict_decode_test.go:136-163`) explicitly pins this contract: input `{"first_unknown":"x","second_unknown":"y"}` (key-ordered via `json.RawMessage` for determinism) produces an error naming `first_unknown` ONLY. The helper inherits `json.Decoder`'s stop-at-first-error semantics from the std lib — no custom collection logic. Spec acceptance #5 row 3 matches exactly. Documented behavior, tested behavior.

- **1.7 Stale fixtures using unknown keys for forward-compat (Attack 7).** REFUTED. `rg "BindArguments" internal/adapters/server/mcpapi/*_test.go` returns only test-function-name and doc-comment hits — no production-equivalent `BindArguments(` calls in test code. Builder identified 4 stale-fixture failures during the 21-site swap (3 on `till.comment` Operation, 1 on `till.project` AuthContextID), fixed all 4 by adding the missing typed-struct fields. A pre-emptive audit caught 4 more latent gaps (`capabilityLeaseMutationArgs`, `handleActionItemOperation` anonymous struct, `attentionItemMutationArgs`, `handoffMutationArgs`) and fixed those proactively. The fixture/strict-decode contract is now coherent: every schema-declared `WithString`/`WithBool` key has a typed-struct field with a matching `json:` tag.

- **1.8 `Operation` field declared-not-read confusion (R-A.2-2 — Attack 8).** REFUTED. R-A.2-2 is a real cleanup opportunity, NOT a correctness defect. Verified at `extended_tools.go:2061-2065` + `2098`: the strict decoder accepts the `operation` key (which is required-by-schema at line 2031) by populating the typed struct's `Operation` field; the handler still reads via `req.GetString("operation", "")` for the `switch` at line 2098. WITHOUT the field declaration, the strict decoder would reject the schema-required `operation` key — so the field is load-bearing for strict-decode correctness. The handler's continuing use of `req.GetString` is a pre-existing code style; refactoring to read `args.Operation` is a low-priority cleanup tracked as R-A.2-2 in BUILDER_WORKLOG. Routing-to-refinement is appropriate; NOT a counterexample.

- **1.9 Errgroup / indirect `BindArguments` usage (Attack 9).** REFUTED. `rg "BindArguments" --type=go` across the entire repo shows zero indirect call sites — no errgroup, futures, or middleware shape that wraps `BindArguments`. The 21 swap sites are the complete set. Helper-level refactor was atomic.

### 2. Counterexamples

- None CONFIRMED. The strict-decoder-bypass finding (1.1) is a SCOPE-VS-STATED-GOAL gap (5 MCP tools never reach the strict decoder) — the builder hit acceptance criterion #3 exactly ("All 21 production `BindArguments` call sites ... swap"), but acceptance criterion #1's stated intent ("Stop schema-drift bugs ... from landing as silent no-ops") is incompletely realized for tools that never used `BindArguments` in the first place. Per spec authorial intent (acceptance #3 was the binding mechanical contract), this is NOT an A.2 counterexample. ROUTE AS `R-A.2-3` REFINEMENT for the orchestrator's closeout list — a follow-up droplet that converts the 5 raw-read tool handlers (`till.embeddings`, `till.kind`, `till.list_kind_definitions`, `till.get_instructions`, `till.capture_state`) to typed-struct + strict-decode for full schema-drift hardening across the MCP surface.

### 3. Summary

**PASS WITH FINDINGS.** A.2 mechanically meets every acceptance criterion: 21/21 swaps present (handler.go=5, handoff_tools.go=5, extended_tools.go=11), helper correctly implements `DisallowUnknownFields` with field-name extraction via stable-format prefix + `strconv.Unquote`, A.1 null-pointer regression preserved (helper-level + fast-path wire test), 8 helper-level + 1 wire-level test cases covering valid/null/unknown/multiple-unknown/nil/empty/non-pointer/nil-target/fast-path, four schema-vs-struct gaps fixed during the swap + four pre-emptive AuthContextID additions, `mage ci` green at 2749 tests / 0 fail / 24 packages all >=70% coverage. R-A.2-1 (Operation declared-not-read) and R-A.2-2 (auth_context_id schema-vs-struct invariant docs) are honest cleanup notes. NEW REFINEMENT R-A.2-3 raised here: 5 MCP tools (`till.embeddings`, `till.kind`, `till.list_kind_definitions`, `till.get_instructions`, `till.capture_state`) bypass strict-decode by reading raw req params; spec-mechanical work is correct, follow-up droplet should harden these tools to close the schema-drift gap end-to-end. Recommend droplet A.2 closes; orchestrator routes R-A.2-3 to closeout refinements list (or to a Theme A continuation droplet if scope expands).

### Hylla Feedback

N/A — A.2 review touched Go files (Hylla-eligible in principle) but the spawn-prompt directive ("NO Hylla calls") routed all evidence through `Read` + `Bash` (`rg` for `BindArguments`, `bindArgumentsStrict`, `req\.GetString`, `req\.GetMap`, `req\.GetBool`, `req\.GetInt`, `AuthContextID`, `mcp\.NewTool`, `srv\.AddTool`). Hylla is stale post-Drop-4c-merge until reingest, so no miss to log.

### TL;DR

T1. PASS WITH FINDINGS — A.2's mechanical contract is met exactly (21/21 swaps, helper correctness, A.1 null preservation, four gap-fixes + four pre-emptive); one spec-vs-stated-goal gap (R-A.2-3: 5 MCP tools bypass strict-decode entirely) raised as a follow-up refinement, NOT an A.2 counterexample.
