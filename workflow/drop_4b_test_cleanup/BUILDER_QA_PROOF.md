# DROP_4B_TEST_CLEANUP — Builder QA Proof Review

Appended per build round. See `workflow/example/drops/WORKFLOW.md` § "Phase 5 — Build QA (per droplet)".

## Droplet 1.1 — Round 1

- **Reviewer:** go-build-qa-proof-agent
- **Date:** 2026-05-18
- **Scope:** R5 domain alias normalization — `internal/domain/comment.go` + `internal/domain/comment_test.go`
- **Verdict:** `pass`
- **Findings:** 0 blocking, 1 NIT (informational only)

### Evidence

**Premises (what must hold for D1.1 to be correct):**

- P1. `commentTargetTypeAliases` package-level map exists in `internal/domain/comment.go` with `"actionitem" → "action_item"`.
- P2. `NormalizeCommentTargetType` applies the alias lookup BEFORE the canonical-form range scan.
- P3. `NormalizeCommentTargetType("actionItem")` returns `"action_item"`.
- P4. `IsValidCommentTargetType("actionItem")` returns `true` (propagates via the existing `NormalizeCommentTargetType` call at line 174).
- P5. Tests cover `"actionItem"`, `"ActionItem"`, `"ACTIONITEM"`, canonical `"action_item"`, and whitespace-padded `" actionItem "`.
- P6. `mage test-pkg ./internal/domain` passes with 0 regressions (309 tests).
- P7. No deletion of existing tests; only additions.
- P8. Scope is limited to the two declared `paths`: `internal/domain/comment.go`, `internal/domain/comment_test.go`. Worklog + PLAN.md updates are out-of-scope-but-allowed (cascade artifact).

**Evidence:**

- **P1, P2** — `internal/domain/comment.go:144-146` declares `commentTargetTypeAliases` with single entry `"actionitem": CommentTargetTypeActionItem`. Lines 161-163 inside `NormalizeCommentTargetType` perform the alias lookup AFTER `strings.TrimSpace(strings.ToLower(...))` and AFTER the empty-string short-circuit, but BEFORE the `validCommentTargetTypes` range loop at lines 164-168. Order is correct: alias-resolve, then canonical-form fallthrough.
- **P3, P4** — Code trace (verified by reading lines 154-176):
  - `NormalizeCommentTargetType("actionItem")` → `lowered = "actionitem"` → alias map hits → returns `"action_item"`.
  - `IsValidCommentTargetType("actionItem")` → line 174 calls `NormalizeCommentTargetType("actionItem")` = `"action_item"` → line 175 `slices.Contains(validCommentTargetTypes, "action_item")` returns `true`.
- **P5** — `internal/domain/comment_test.go:218-222` lists exactly the five claimed cases:
  - `{"camelCase actionItem", "actionItem", CommentTargetTypeActionItem}`
  - `{"mixed case ActionItem", "ActionItem", CommentTargetTypeActionItem}`
  - `{"all caps ACTIONITEM", "ACTIONITEM", CommentTargetTypeActionItem}`
  - `{"canonical action_item unchanged", "action_item", CommentTargetTypeActionItem}`
  - `{"whitespace-padded actionItem", " actionItem ", CommentTargetTypeActionItem}`
  Each subtest (lines 225-233) asserts both `NormalizeCommentTargetType(input) == want` AND `IsValidCommentTargetType(input) == true`, so the test pins both prongs of the acceptance.
- **P6** — `mage test-pkg ./internal/domain` output: `tests: 309 / passed: 309 / failed: 0 / skipped: 0 / packages: 1 / pkg passed: 1`. `mage test-func ./internal/domain TestNormalizeCommentTargetTypeAlias` output: `tests: 6 / passed: 6 / failed: 0` (1 parent + 5 subtests).
- **P7** — `git diff HEAD -- internal/domain/comment_test.go` shows `+30 -0` (pure additions, no deletions). All five pre-existing tests (`TestNewCommentDefaultsAndNormalization`, `TestNewCommentUsesProvidedSummary`, `TestNewCommentDerivesSummaryFromFirstNonEmptyBodyLine`, `TestNewCommentDefaultsActorNameFromActorID`, `TestNewCommentValidation`, `TestNormalizeCommentTarget`, `TestNormalizeCommentTargetRejectsLegacyScopeTypes`) remain in the file unchanged.
- **P8** — `git diff HEAD --stat` for the D1.1 scope shows only `internal/domain/comment.go (+17/-1)` and `internal/domain/comment_test.go (+30/-0)`. Other modified files in `git status` (`internal/adapters/mcp_rpc/*`, `internal/app/dispatcher/subscriber_test.go`, `ui/`, etc.) belong to sibling parallel builders (D1.2, D1.3, D1.5, fe-1-bootstrap) — scope-isolated from D1.1.

### Trace coverage

Walked each claimed input through the production code:

| Input            | `strings.ToLower` + trim | Alias map hit?         | Return value     |
| ---------------- | ------------------------ | ---------------------- | ---------------- |
| `"actionItem"`   | `"actionitem"`           | yes → `action_item`    | `"action_item"`  |
| `"ActionItem"`   | `"actionitem"`           | yes → `action_item`    | `"action_item"`  |
| `"ACTIONITEM"`   | `"actionitem"`           | yes → `action_item`    | `"action_item"`  |
| `"action_item"`  | `"action_item"`          | no (not in alias map)  | falls through to range scan, matches `validCommentTargetTypes[1]` → `"action_item"` |
| `" actionItem "` | `"actionitem"`           | yes → `action_item`    | `"action_item"`  |

The canonical-form case is critical: it confirms the alias map does NOT shadow the existing canonical lookup, so pre-existing `"action_item"` callers continue to work via the range loop (regression-safe). The test case `"canonical action_item unchanged"` pins this explicitly.

`TestNormalizeCommentTargetRejectsLegacyScopeTypes` (lines 240-250) still passes — confirming that adding `"actionitem"` to the alias map did NOT accidentally widen the accepted set to include legacy tokens `"branch"`, `"phase"`, `"subtask"`, `"decision"`, `"note"`. These continue to fall through both the alias map and the canonical range scan, returning the lowercased form which fails `IsValidCommentTargetType`'s `slices.Contains` check.

### Conclusion

All eight premises hold. Acceptance criteria from PLAN.md droplet 1.1:

- `mage test-pkg internal/domain` passes — verified (309/309).
- `NormalizeCommentTargetType("actionItem")` returns `"action_item"` — verified by trace + table-driven test.
- `IsValidCommentTargetType("actionItem")` returns `true` — verified by trace + table-driven test.
- Existing tests unchanged — verified by `git diff` (`-0` deletions).

F6 (alias persistence round-trip) is documented in the PLAN as accepted under the pre-MVP no-migration-logic rule; builder correctly skipped a persistence regression test. The worklog acknowledges this. No finding.

### Findings

**NIT 1 (severity: low; axis: acceptance-criteria-coverage; non-blocking):**
The doc-comment for `commentTargetTypeAliases` (lines 139-143) describes the alias as "the camelCase form used in the **pre-Drop-1.75 schema**". Strictly the camelCase form `"actionItem"` is still the form declared by the **current** MCP schema enum at `internal/adapters/mcp_rpc/extended_tools.go:2243` (which D1.2 is updating to include both `"action_item"` and `"actionItem"`). So the doc-comment's "pre-Drop-1.75" framing slightly understates the alias's continued relevance after D1.2 lands. Suggested rewording: *"the camelCase form used in the MCP schema declaration that callers send; the canonical domain form is snake_case."* — accept-as-is acceptable; this is a doc-comment polish, not a correctness issue.

### Scope check

`git status` confirms the scope-isolated files for D1.1 are exactly the two declared paths. Sibling-builder WIP in `internal/adapters/mcp_rpc/*` (D1.2/D1.5), `internal/app/dispatcher/*` (D1.3), and `ui/*` (fe-1-bootstrap) is unrelated to D1.1 and is correctly outside the D1.1 review scope per the "parallel builders running" carve-out in the spawn prompt.

### Unknowns

None. All claims have explicit evidence in code, tests, or mage output.

### Hylla Feedback

- **Query:** Hylla MCP tools (`mcp__hylla__*`)
- **Missed because:** Per memory `feedback_hylla_disabled_for_now.md` (2026-05-18), Hylla MCP is OFF due to a DB issue. Tools were not available in this agent's surface; the builder's worklog also reported the same absence.
- **Worked via:** `Read`, `git diff`, `mage test-pkg`, `mage test-func` directly on the working tree.
- **Suggestion:** No Hylla-specific suggestion; the disabled-state is dev-managed. Once Hylla is re-enabled, a `mcp__hylla__hylla_search_keyword commentTargetTypeAliases` followed by `mcp__hylla__hylla_refs_find` on `NormalizeCommentTargetType` would have been the natural starting point.

## Droplet 1.2 — Round 1

- **Reviewer:** go-build-qa-proof-agent
- **Date:** 2026-05-18
- **Scope:** R5 MCP schema enum update for `till.comment` — `internal/adapters/mcp_rpc/extended_tools.go` + `internal/adapters/mcp_rpc/extended_tools_test.go`
- **Verdict:** `pass`
- **Findings:** 0 blocking, 1 NIT (informational only)

### Evidence

**Premises (what must hold for D1.2 to be correct):**

- P1. `extended_tools.go:2243` declares `target_type` schema with `mcp.Enum("project", "action_item", "actionItem")` — three tokens, in that order, no stale pre-1.75 tokens (`"branch"`, `"phase"`, `"subtask"`, `"decision"`, `"note"`).
- P2. The `target_type` description string at the same line is EXACTLY `"project|action_item|actionItem"` — no leading/trailing whitespace, no missing pipe, no stale tokens.
- P3. A new regression test `TestIsValidCommentTargetTypeLegacyTokensRejected` exists in `extended_tools_test.go` covering all 5 legacy tokens (`branch`, `phase`, `subtask`, `decision`, `note`), each asserting `domain.IsValidCommentTargetType(token) == false`.
- P4. `mage test-pkg ./internal/adapters/mcp_rpc` passes with 232/232 tests (no regressions; new test included in the count).
- P5. The targeted new test passes when invoked directly via `mage test-func ./internal/adapters/mcp_rpc TestIsValidCommentTargetTypeLegacyTokensRejected`.
- P6. Scope is limited to the declared `paths`: `internal/adapters/mcp_rpc/extended_tools.go`. The test addition lives in `internal/adapters/mcp_rpc/extended_tools_test.go` (same package; planner-authorized via "builder picks the natural location" in the acceptance criterion + worklog-documented dirty-domain-sibling fallback).
- P7. The canonical token `"action_item"` is preserved in both enum and description string (no regression for existing canonical callers).
- P8. Enum order matches description string order: `project`, `action_item`, `actionItem`.

**Evidence:**

- **P1, P2** — `internal/adapters/mcp_rpc/extended_tools.go:2243` literal read:
  ```go
  mcp.WithString("target_type", mcp.Description("project|action_item|actionItem"), mcp.Enum("project", "action_item", "actionItem")),
  ```
  Description string is exactly `"project|action_item|actionItem"`. Enum values are exactly `("project", "action_item", "actionItem")`. No stale tokens present.
- **P1, P2 (diff confirmation)** — `git diff` shows the single-line change:
  - Before: `mcp.WithString("target_type", mcp.Description("project|branch|phase|actionItem|subtask|decision|note"), mcp.Enum("project", "branch", "phase", "actionItem", "subtask", "decision", "note"))`
  - After:  `mcp.WithString("target_type", mcp.Description("project|action_item|actionItem"), mcp.Enum("project", "action_item", "actionItem"))`
  Single insertion / single deletion. No collateral damage.
- **P3** — `internal/adapters/mcp_rpc/extended_tools_test.go` appended `TestIsValidCommentTargetTypeLegacyTokensRejected` (+23 lines / -0). Test body:
  ```go
  tests := []struct{ token domain.CommentTargetType }{
      {token: "branch"},
      {token: "phase"},
      {token: "subtask"},
      {token: "decision"},
      {token: "note"},
  }
  for _, tc := range tests {
      if domain.IsValidCommentTargetType(tc.token) {
          t.Errorf("domain.IsValidCommentTargetType(%q) = true, want false (stale pre-1.75 token must be rejected)", tc.token)
      }
  }
  ```
  All 5 legacy tokens present; each asserts `false` via direct call to `domain.IsValidCommentTargetType`.
- **P4** — `mage test-pkg ./internal/adapters/mcp_rpc` output:
  - `tests: 232 / passed: 232 / failed: 0 / skipped: 0`
  - `pkg passed: 1 / pkg failed: 0`
- **P5** — `mage test-func ./internal/adapters/mcp_rpc TestIsValidCommentTargetTypeLegacyTokensRejected` output: `tests: 1 / passed: 1 / failed: 0` (`-race -count=1` flags applied by the mage target).
- **P6** — `git status --porcelain internal/adapters/mcp_rpc/` shows exactly two dirty files: `M internal/adapters/mcp_rpc/extended_tools.go` and `M internal/adapters/mcp_rpc/extended_tools_test.go`. Other repo-dirty files (`internal/domain/comment.go`, `internal/domain/comment_test.go`, `internal/app/dispatcher/subscriber_test.go`, `ui/*`, etc.) are sibling-builder WIP (D1.1, D1.3, fe-1-bootstrap) per the spawn-prompt parallel-builders carve-out.
- **P7** — Both literal reads confirm `"action_item"` is in the enum (position 2) and description string (between `project|` and `|actionItem`). The R5 acceptance criterion "canonical post-1.75 token preserved" holds.
- **P8** — Enum order: `(project, action_item, actionItem)`. Description order: `project|action_item|actionItem`. Match exact.

### Trace coverage

Walked each legacy token through `IsValidCommentTargetType` (the test's target function):

| Legacy token  | `NormalizeCommentTargetType` (ToLower + alias map miss + range-scan miss) | `slices.Contains(validCommentTargetTypes, ...)` | Test assertion |
| ------------- | ------------------------------------------------------------------------- | ------------------------------------------------ | -------------- |
| `"branch"`    | returns `"branch"` (no canonical match)                                   | `false`                                          | `false` (PASS) |
| `"phase"`     | returns `"phase"`                                                         | `false`                                          | `false` (PASS) |
| `"subtask"`   | returns `"subtask"`                                                       | `false`                                          | `false` (PASS) |
| `"decision"`  | returns `"decision"`                                                      | `false`                                          | `false` (PASS) |
| `"note"`      | returns `"note"`                                                          | `false`                                          | `false` (PASS) |

The alias-map step from D1.1 only adds `"actionitem" → "action_item"`; the 5 legacy tokens fall through to the canonical range scan and miss, so the lowercased input is returned and `slices.Contains` fails. Test behavior matches reality.

Sibling-coordination cross-check: the regression test uses `domain.IsValidCommentTargetType` (which is what D1.1's alias map propagates through), so D1.2's test is correctly coupled to D1.1's domain-layer enforcement of the shrunk enum. The MCP schema enum cannot independently widen what the domain validation allows — they are consistent post-D1.1+D1.2.

### Conclusion

All eight premises hold. Acceptance criteria from PLAN.md droplet 1.2:

- `mage test-pkg internal/adapters/mcp_rpc` passes — verified (232/232 with `./` prefix; bare form silently runs 0 tests per PLAN orch-correction).
- Schema enum contains `"project"`, `"action_item"`, `"actionItem"` and excludes stale pre-1.75 tokens — verified by literal read at line 2243.
- Description string exactly `"project|action_item|actionItem"` — verified by literal read at line 2243.
- `IsValidCommentTargetType("branch" / "phase" / "subtask" / "decision" / "note")` returns `false` — verified by table-driven test in `extended_tools_test.go` (passing).

Test-file-location: PLAN.md acceptance text says "builder picks the natural location" with explicit "may live in either domain or mcp_rpc". Builder chose mcp_rpc because the domain test file was dirty (sibling D1.1). The choice is acceptance-compliant and worklog-documented.

### Findings

**NIT 1 (severity: low; axis: spec-conformance; non-blocking):**
PLAN.md D1.2 `Paths:` lists only `internal/adapters/mcp_rpc/extended_tools.go`, but the builder appended the regression test to `internal/adapters/mcp_rpc/extended_tools_test.go` (consistent with the acceptance criterion's "builder picks the natural location" allowance and the worklog-documented dirty-domain-sibling fallback). The PLAN's `Paths:` field could have listed both `extended_tools.go` and `extended_tools_test.go` to match the acceptance criterion's flexibility. Accept-as-is — the builder's choice is acceptance-criterion-compliant and within the same package as the schema being changed; this is a PLAN.md documentation polish, not a correctness or scope issue. No blocking fix required.

### Scope check

`git status --porcelain internal/adapters/mcp_rpc/` confirms scope-isolation: only the two D1.2 files are dirty within the declared package. Sibling-builder WIP in `internal/domain/*` (D1.1 done), `internal/app/dispatcher/*` (D1.3 in-flight), `ui/*` (fe-1-bootstrap), and `workflow/*` (drop coordination MDs) are correctly outside D1.2's review scope.

### Unknowns

None. All claims have explicit evidence in code, diffs, mage output, or PLAN.md acceptance text.

### Hylla Feedback

- **Query:** Hylla MCP tools (`mcp__hylla__*`)
- **Missed because:** Per memory `feedback_hylla_disabled_for_now.md` (2026-05-18), Hylla MCP is OFF due to a DB issue. Tools were not available in this agent's surface; the builder's worklog also reported the same absence.
- **Worked via:** `Read`, `git diff`, `git status --porcelain`, `mage test-pkg ./internal/adapters/mcp_rpc`, `mage test-func ./internal/adapters/mcp_rpc TestIsValidCommentTargetTypeLegacyTokensRejected` directly on the working tree.
- **Suggestion:** No Hylla-specific suggestion; disabled-state is dev-managed. Once re-enabled, `mcp__hylla__hylla_search_keyword registerCommentTools` + `mcp__hylla__hylla_refs_find IsValidCommentTargetType` would have been the natural starting point.

## Droplet 1.3 — Round 1

- **Reviewer:** go-build-qa-proof-agent
- **Date:** 2026-05-18
- **Scope:** R6.1 rename + R6.2 goleak TestMain + R6.3 clarifying comment + R7.4 file split — `internal/app/dispatcher/subscriber_test.go` (modified) + `internal/app/dispatcher/dispatcher_e2e_test.go` (new)
- **Verdict:** `pass`
- **Findings:** 0 blocking, 0 NIT

### Evidence

**Premises (what must hold for D1.3 to be correct):**

- P1. `internal/app/dispatcher/dispatcher_e2e_test.go` exists as a new file with package declaration `package dispatcher`.
- P2. The new file imports `go.uber.org/goleak` and declares a single `TestMain(m *testing.M)` that invokes `goleak.VerifyTestMain(m)` as a plain statement (NOT wrapped in `os.Exit(...)` — goleak returns void).
- P3. `stubE2ETemplateResolver` type + `GetProjectTemplate` method are defined in the new file (moved from subscriber_test.go lines 509-522).
- P4. `TestAutoDispatch_NewDispatcherGateWiring` is defined in the new file (renamed from `TestAutoDispatchE2EGatePassViaNewDispatcher`).
- P5. `TestAutoDispatchE2EGateFailViaNewDispatcher` is defined in the new file (moved verbatim).
- P6. The three moved symbols and the `templates` import are absent from `subscriber_test.go`.
- P7. `subscriber_test.go` line count is approximately 506 (worklog claim: 667 → 506).
- P8. The old test name `TestAutoDispatchE2EGatePassViaNewDispatcher` appears in zero Go files repo-wide; the new name appears only in `dispatcher_e2e_test.go`.
- P9. Exactly one `TestMain` exists in the dispatcher package (Go requires uniqueness per package test binary).
- P10. The R6.3 clarifying comment lives on the `lister.calls.Load()` assertion in `TestAutoDispatch_NewDispatcherGateWiring` (per builder judgment under PLAN's "or whichever test pins subscriber lifecycle state" clause).
- P11. `mage test-pkg ./internal/app/dispatcher` passes 389/389 tests with `goleak.VerifyTestMain` reporting no goroutine leaks.
- P12. Scope is limited to the two D1.3 files within `internal/app/dispatcher/` per `git status --porcelain`.

**Evidence:**

- **P1** — `dispatcher_e2e_test.go:5` declares `package dispatcher`. File header (lines 1-4) explains its purpose and the R7.4 split rationale.
- **P2** — `dispatcher_e2e_test.go:13` imports `"go.uber.org/goleak"`. Lines 31-33:
  ```go
  func TestMain(m *testing.M) {
      goleak.VerifyTestMain(m)
  }
  ```
  No `os.Exit(...)` wrap. Builder's note in worklog confirms the API gotcha was caught: `VerifyTestMain` returns void and handles `os.Exit` internally; wrapping caused a build error that was fixed before commit.
- **P3** — `dispatcher_e2e_test.go:39-48` defines `type stubE2ETemplateResolver struct { tpl templates.Template }` plus `func (s *stubE2ETemplateResolver) GetProjectTemplate(_ context.Context, _ string) (templates.Template, error)`. Verbatim move from the previously-cited subscriber_test.go lines 509-522 (PLAN.md confirmed the source line range).
- **P4** — `dispatcher_e2e_test.go:69` declares `func TestAutoDispatch_NewDispatcherGateWiring(t *testing.T)`. The body (lines 70-142) preserves all prior assertions (gate runner returns `GateStatusPassed`, `lister.calls == 1`, no error).
- **P5** — `dispatcher_e2e_test.go:153` declares `func TestAutoDispatchE2EGateFailViaNewDispatcher(t *testing.T)`. Body retains `t.Parallel()` (line 154) per PLAN R6.2 spec. Failure assertions intact: `errors.Is(results[0].Err, ErrGateNotRegistered)`, `GateStatusFailed`, `GateName == GateKindMageTestPkg`.
- **P6** — `rg "templates|goleak" subscriber_test.go` → 0 matches. Imports block (lines 3-12) lists only `context`, `errors`, `sync/atomic`, `testing`, `time`, `app`, `domain`. `rg "stubE2ETemplateResolver"` → only the new file. `rg "TestAutoDispatchE2EGatePassViaNewDispatcher|TestAutoDispatch_NewDispatcherGateWiring|TestAutoDispatchE2EGateFailViaNewDispatcher"` → only the new file.
- **P7** — `wc -l internal/app/dispatcher/subscriber_test.go` returns 506. Exact match to worklog claim.
- **P8** — `rg "TestAutoDispatchE2EGatePassViaNewDispatcher" -g "*.go" /Users/.../tillsyn/main` → exit 1 (zero hits in any Go file). Cross-check: the old name still appears in PLAN/REVISION_BRIEF/QA artifacts in `workflow/drop_4b_test_cleanup/` (historical context — durable audit trail, NOT live code references), which is correct preservation. The new name appears in `dispatcher_e2e_test.go` at the comment (line 50) and func decl (line 69) only.
- **P9** — `rg "func TestMain" -g "*.go" /Users/.../internal/app/dispatcher` → one hit at `dispatcher_e2e_test.go:31`. Go test binary uniqueness invariant satisfied.
- **P10** — `dispatcher_e2e_test.go:117-119`:
  ```go
  // "state transitions" in the D5 spec means dispatcher lifecycle (Start/Stop),
  // not action-item state. This lister-calls pin is the lifecycle-transition signal.
  if got := lister.calls.Load(); got != 1 {
  ```
  Placement is on the `lister.calls.Load() != 1` assertion inside the moved+renamed gate-pass test. PLAN.md spec text "the `lister.calls` assertion ... or whichever test pins subscriber lifecycle state" — this assertion pins "Start enumerated projects exactly once," a lifecycle-state pin. Builder judgment matches PLAN intent.
- **P11** — `mage test-pkg ./internal/app/dispatcher` output: `tests: 389 / passed: 389 / failed: 0 / skipped: 0 / packages: 1 / pkg passed: 1`. Goleak reported no leaks (otherwise the package binary would have non-zero exit per `VerifyTestMain` contract). Worklog confirms the R6.2 scope-creep guard did NOT fire — no goroutine leak was surfaced anywhere in the 389-test surface, so the per-test `VerifyNone(t)` fallback was not needed.
- **P12** — `git status --porcelain internal/app/dispatcher/`:
  ```
   M internal/app/dispatcher/subscriber_test.go
  ?? internal/app/dispatcher/dispatcher_e2e_test.go
  ```
  Exactly the two declared D1.3 files. Sibling-builder WIP in `internal/adapters/mcp_rpc/` (D1.2) and `ui/` (FE) is outside this scope.

### Trace coverage

- **Compile trace** — All `_test.go` files in the dispatcher package compile together. The new file references `fakeCommandRunner` + `withFakeCommandRunner` (defined in `gate_mage_ci_test.go`), `stubProjectLister` + `subscriberWalkerStub` + `newTreeWalker` (defined in `subscriber_test.go`). Cross-file references resolve because the test binary builds the union. `mage test-pkg` success confirms compile.
- **goleak invocation trace** — `m.Run()` is invoked internally by `goleak.VerifyTestMain(m)`. After all 389 tests complete, goleak diffs the active goroutine set against its snapshot; any unexpected goroutine triggers a non-zero exit. The mage run finished with `tests: 389 / passed: 389`, so the diff was clean.
- **Rename safety trace** — Zero hits in Go for the old name confirms no test file, CI workflow, magefile, or build script references the stale identifier. The historical references in `workflow/drop_4b_test_cleanup/*.md` are intentional audit trail and are not consumed by the build / test toolchain.
- **R6.3 placement trace** — Two other `lister.calls.Load() != 1` assertions exist in subscriber_test.go (lines 156, 284) in unrelated tests; the comment is correctly scoped to the moved gate-pass test where the "state transitions" misreading was the falsification target in earlier QA rounds.

### Conclusion

All 12 premises hold. Acceptance criteria from PLAN.md droplet 1.3:

- R6.1 — `TestAutoDispatch_NewDispatcherGateWiring` exists; old `TestAutoDispatchE2EGatePassViaNewDispatcher` no longer exists in Go code. Verified.
- R6.2 — `goleak.VerifyTestMain(m)` wired in `dispatcher_e2e_test.go` `TestMain`. Scope-creep guard did not fire. Verified.
- R6.3 — clarifying comment placed on a lifecycle-pin assertion (`lister.calls.Load() != 1` in the moved gate-pass test). Verified.
- R7.4 — `dispatcher_e2e_test.go` created with `stubE2ETemplateResolver` + both e2e tests; symbols removed from `subscriber_test.go`; `templates` import cleaned up. Verified.
- `mage test-pkg ./internal/app/dispatcher` passes 389/389. Verified.

API gotcha (`os.Exit` wrap on `goleak.VerifyTestMain`) was caught and fixed before commit per worklog — no residual evidence in the final file. Good builder hygiene.

### Findings

None — 0 blocking, 0 NIT.

### Scope check

`git status --porcelain internal/app/dispatcher/` confirms exactly the two D1.3 files are dirty within the declared package. Sibling-builder WIP in `internal/adapters/mcp_rpc/` (D1.2 round 2 in flight) and `ui/` (FE D1.2/D1.3 dispatching) is correctly outside the D1.3 review surface.

### Unknowns

None. All claims have explicit evidence in code, grep results, or mage output.

### Hylla Feedback

- **Query:** Hylla MCP tools (`mcp__hylla__*`)
- **Missed because:** Per memory `feedback_hylla_disabled_for_now.md` (2026-05-18), Hylla MCP is OFF due to a DB issue. Tools were not available in this agent's surface; the builder's worklog also reported the same absence.
- **Worked via:** `Read` (worklog + both Go files), `rg` via `rtk proxy` for cross-repo symbol searches, `wc -l` for line count, `git status --porcelain`, and `mage test-pkg ./internal/app/dispatcher` for acceptance.
- **Suggestion:** No Hylla-specific suggestion; disabled-state is dev-managed. Once re-enabled, `mcp__hylla__hylla_refs_find TestMain` + `mcp__hylla__hylla_search_keyword "goleak.VerifyTestMain"` would have been the natural starting point for the rename + import-cleanup verification.

## Droplet 1.2 — Round 2

- **Reviewer:** go-build-qa-proof-agent
- **Date:** 2026-05-18
- **Scope:** Round-1 falsification absorption — Attack 5 (replace wrong-layer test with schema-introspection guard) + Attack 2 Option C (rename + annotate transport-layer permissiveness test). Test-only change.
- **Verdict:** `pass`
- **Findings:** 0 blocking, 0 NITs.

### Evidence

**Premises (what must hold for D1.2 Round 2 to be correct):**

- P1. `TestHandlerCommentToolTargetTypeEnumSchemaGuard` exists in `internal/adapters/mcp_rpc/extended_tools_test.go`, uses `tools/list` introspection via `findToolSchemaByName` + `schemaPropertyEnumStrings`, and asserts exact set equality on `["project", "action_item", "actionItem"]` plus stale-token absence (in BOTH the enum AND the description string).
- P2. Production state of `extended_tools.go` line 2243 carries the post-1.75 enum + description (`"project", "action_item", "actionItem"`) — round-1's production diff is preserved, not reverted.
- P3. `TestIsValidCommentTargetTypeLegacyTokensRejected` does not appear in any source file (workflow MDs may still reference it as historical context).
- P4. `TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes` does not appear in any source file.
- P5. `TestHandlerCommentToolForwardsArbitraryTargetTypesToService` exists, its body still sends `"branch"` and `"phase"` to verify transport-layer permissiveness, and its doc comment explicitly documents the transport-permissiveness intent.
- P6. `mage test-pkg ./internal/adapters/mcp_rpc` passes 232/232 (no regression).
- P7. D1.2 round-2 scope is limited to `internal/adapters/mcp_rpc/extended_tools_test.go`. `extended_tools.go` shows the round-1 production diff only (no further changes this round). Sibling-builder WIP in `ui/`, `internal/app/dispatcher/`, `internal/domain/` is out-of-scope parallel-builder activity.

**Evidence:**

- **P1** — `internal/adapters/mcp_rpc/extended_tools_test.go:5958-6021` defines `TestHandlerCommentToolTargetTypeEnumSchemaGuard`. Verified end-to-end:
  - Spins up a real `httptest.NewServer(handler)` and issues a `tools/list` request (lines 5977-5984).
  - Calls `findToolSchemaByName(t, toolsRaw, "till.comment")` at line 5990 — the canonical helper at lines 1140-1148 (not a hand-rolled equivalent).
  - Calls `schemaPropertyEnumStrings(t, commentSchema, "target_type")` at line 5994 — the canonical helper at lines 1172-1192 (not a hand-rolled equivalent).
  - **Exact set equality** is enforced via length check (line 5995-5997: `if len(gotEnum) != len(wantEnum)`) AND containment check (lines 5998-6002: every element of `wantEnum` must be in `gotEnum`). Combined, this is order-tolerant exact set equality — a superset (extra token) fails the length check; a subset (missing token) fails the containment check. Equivalent to `reflect.DeepEqual` modulo ordering.
  - **Stale-token absence in enum** is asserted independently at lines 6004-6009 — explicit list `{"branch", "phase", "subtask", "decision", "note"}` (matches the 5-of-7 pre-1.75 tokens that must be removed; `project` and `actionItem` are retained intentionally).
  - **Description-string assertion**: line 6012 fetches `target_type` description via `schemaStringPropertyDescription`; lines 6013-6015 require it to contain `"project|action_item|actionItem"`; lines 6016-6020 require it to NOT contain any of the same 5 stale tokens. Both prongs covered.
- **P2** — `internal/adapters/mcp_rpc/extended_tools.go:2243`:
  ```go
  mcp.WithString("target_type", mcp.Description("project|action_item|actionItem"), mcp.Enum("project", "action_item", "actionItem")),
  ```
  Production state is the post-1.75 form. Round-1's production diff stays. `git diff --stat` on `extended_tools.go` shows `+1/-1` — exactly the single-line enum + description fix from round 1.
- **P3, P4** — `rg -lrn "TestIsValidCommentTargetTypeLegacyTokensRejected|TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes" internal/ cmd/ magefile.go` exits 1 (zero matches). Both stale names are entirely absent from source.
- **P5** — `internal/adapters/mcp_rpc/extended_tools_test.go:3514-3574` defines `TestHandlerCommentToolForwardsArbitraryTargetTypesToService`:
  - **Doc comment (lines 3514-3519)** explicitly documents intent: "the MCP handler layer does NOT validate target_type values; any string passes through to the service. Domain validation is downstream. This test intentionally sends pre-Drop-1.75 legacy tokens (`branch`, `phase`) to confirm they transit the transport layer untouched — the schema enum excludes them from the JSON-Schema contract but the handler itself does not reject unknown values." Both the renamed function and the intent annotation are present.
  - **Body still sends `"branch"`/`"phase"`** — line 3538 (`"target_type": "branch"` for the create call), line 3558 (`"target_type": "phase"` for the list call). Verified via direct file Read.
  - **Service-forwarding asserts present** — lines 3548-3550 assert `service.lastCreateCommentReq.TargetType == "branch"`; lines 3565-3567 assert `service.lastListCommentReq.TargetType == "phase"`. Both stubbed-service assertions intact.
- **P6** — `mage test-pkg ./internal/adapters/mcp_rpc` output:
  ```
  [PKG PASS] github.com/evanmschultz/tillsyn/internal/adapters/mcp_rpc (0.00s)
  Test summary
    tests: 232
    passed: 232
    failed: 0
    skipped: 0
  [SUCCESS] All tests passed
    232 tests passed across 1 package.
  ```
  Matches the builder's worklog claim exactly.
- **P7** — `git status --porcelain internal/adapters/mcp_rpc/` shows:
  ```
   M internal/adapters/mcp_rpc/extended_tools.go
   M internal/adapters/mcp_rpc/extended_tools_test.go
  ```
  `git diff --stat` confirms `extended_tools.go` is `+1/-1` (the round-1 production fix carried through), and `extended_tools_test.go` is `+73/-3` (Round-1's regression test removed + Round-2's schema-guard test added + Round-2's rename + doc-comment expansion on the renamed test). Both files match D1.2's declared `paths`. The spawn prompt explicitly flagged sibling-builder dirty files in `ui/`, `internal/app/dispatcher/`, `internal/domain/` as expected parallel-builder activity outside D1.2 scope.

**Trace (round-trip the schema-guard's red-green claim, without re-running the revert):**

1. `registerCommentTools` at `extended_tools.go:2237-2254` registers the `till.comment` tool with the JSON-Schema-backed enum at line 2243.
2. The MCP server emits this schema in response to `tools/list` (per the MCP-Go library's standard behavior — verified by the test's successful introspection path at lines 5979-5990).
3. `TestHandlerCommentToolTargetTypeEnumSchemaGuard` introspects the live schema: any divergence from `{"project", "action_item", "actionItem"}` — extra token, missing token, or stale-token leak in description — triggers `t.Fatalf`.
4. Therefore: reverting the enum at line 2243 to the pre-1.75 form (`"project", "branch", "phase", "actionItem", "subtask", "decision", "note"`) would emit a 7-element enum in `tools/list`, fail the length check at line 5996, AND fail multiple iterations of the stale-token absence loop at lines 6004-6009. The builder worklog confirms this red-green cycle was run live with the expected failure message (`till.comment target_type enum = [project branch phase actionItem subtask decision note], want exactly [project action_item actionItem]`).
5. Conclusion: the test is correctly wired to the production change; it catches a revert; restoring the enum restores green.

**Conclusion:** Both round-1 falsification absorptions are correctly executed. Attack 5 is now defended by a transport-layer schema-introspection test using the canonical `findToolSchemaByName` + `schemaPropertyEnumStrings` helpers. Attack 2 Option C is now correctly named and doc-commented to express the transport-permissiveness intent without sacrificing test value. Acceptance run is green. Scope is clean. **PASS — round 2 ready for parallel falsification sweep.**

**Unknowns:** None. Every claim is backed by either direct file Read or live `mage` test output. The schema-guard's red-green verification was performed live by the builder (per worklog) and the post-condition (current enum = post-1.75 form, test PASS) is directly verified via Read of `extended_tools.go:2243` and the `mage test-pkg` 232/232 result.

### Hylla Feedback

N/A — Hylla MCP is OFF (`feedback_hylla_disabled_for_now.md`, 2026-05-18). Evidence gathering used `Read` on the test file + production file, `rg` for stale-name absence sweep, `git status --porcelain` + `git diff --stat` for scope, and `mage test-pkg ./internal/adapters/mcp_rpc` for the live acceptance run.
