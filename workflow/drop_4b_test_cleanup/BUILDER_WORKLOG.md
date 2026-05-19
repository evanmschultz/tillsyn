# DROP_4B_TEST_CLEANUP — Builder Worklog

Append a `## Droplet N.M — Round K` section per build attempt. See `drops/WORKFLOW.md` § "Phase 4 — Build (per droplet)" for what each section should contain.

## Droplet N.1 — Round 1

- **Builder:** <builder-agent-type> (e.g. `go-builder-agent` / `fe-builder-agent`)
- **Started:** YYYY-MM-DD HH:MM
- **Files touched:** <list>
- **Build-tool targets run:** <e.g. `mage build` (pass), `mage test-pkg <pkg>` (pass), `npm run test -- <path>` (pass), …>
- **Notes:** <design choices, surprises, library quirks, references to Context7 / language doc / LSP queries that mattered>

### Hylla Feedback

<For Go projects using Hylla. For other languages, rename to the code-understanding index feedback you're collecting, or delete this subsection if you don't use one. Record any case where the index missed and a fallback (LSP / Read / Grep) was needed. Format: Query → Missed because → Worked via → Suggestion. Aggregated into `HYLLA_FEEDBACK.md` (or equivalent) at closeout.>

## Droplet 1.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-18
- **Files touched:**
  - `internal/domain/comment.go`
  - `internal/domain/comment_test.go`
- **Build-tool targets run:**
  - `mage test-func ./internal/domain TestNormalizeCommentTargetTypeAlias` (RED — 5 failures before fix, as expected)
  - `mage test-func ./internal/domain TestNormalizeCommentTargetTypeAlias` (GREEN — 6/6 after fix)
  - `mage test-pkg ./internal/domain` (pass — 309 tests, 0 failures; regression-clean)
- **Notes:**
  - **TDD cycle:** Wrote `TestNormalizeCommentTargetTypeAlias` (5 subtests: camelCase, mixed-case, all-caps, canonical, whitespace-padded) BEFORE the production change. Confirmed RED with exact failure messages showing `"actionitem"` returned instead of `"action_item"`. Added alias map `commentTargetTypeAliases` and pre-normalization step in `NormalizeCommentTargetType`. Confirmed GREEN.
  - **Design choice — package-level alias map:** Used a `var commentTargetTypeAliases = map[string]CommentTargetType{...}` at the package level rather than an inline `switch`, so adding future aliases is a single-line addition to the map with no branching logic change. The map is small (one entry today) and lookup is O(1).
  - **`IsValidCommentTargetType` fix is implicit:** Confirmed from source that `IsValidCommentTargetType` calls `NormalizeCommentTargetType` first (line 158 of `comment.go`), so the alias fix propagates to validity checks automatically with no additional change.
  - **mage test-pkg path format discovery:** `mage test-func internal/domain` (no `./`) runs `go test internal/domain` which Go interprets as a module import path and silently produces 0 tests + exit 1. Must use `mage test-func ./internal/domain` (with `./`) for filesystem-local packages. Noted for future builder sessions.
  - **F6 (persistence round-trip):** Per PLAN.md notes, no persistence regression test added. This drift is accepted under the pre-MVP no-migration-logic rule. Noted in worklog only.
  - **Scope:** Stayed strictly within declared paths (`internal/domain/comment.go` + `internal/domain/comment_test.go`). No other files touched.

### Hylla Feedback

- **Query:** `mcp__hylla__hylla_search_keyword NormalizeCommentTargetType`
- **Missed because:** `mcp__hylla__hylla_search_keyword` tool was not available (tool not found error). Hylla MCP tools not present in this agent session despite CLAUDE.md indicating Hylla is back on.
- **Worked via:** `Read` on `internal/domain/comment.go` and `internal/domain/comment_test.go` directly.
- **Suggestion:** Ensure the Hylla MCP server is registered and tools (`hylla_search_keyword`, `hylla_search`, `hylla_node_full`, `hylla_refs_find`, `hylla_graph_nav`) are exposed in the agent's MCP config. The `mcp__hylla__*` namespace was entirely absent — either the server wasn't started or its tools weren't in the agent's allowlist.

## Droplet 1.2 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-18
- **Files touched:**
  - `internal/adapters/mcp_rpc/extended_tools.go` (line 2243 — enum and description string)
  - `internal/adapters/mcp_rpc/extended_tools_test.go` (appended `TestIsValidCommentTargetTypeLegacyTokensRejected`)
- **Build-tool targets run:**
  - `mage test-func ./internal/adapters/mcp_rpc TestIsValidCommentTargetTypeLegacyTokensRejected` (GREEN — 1/1 pass; regression test for legacy-token rejection)
  - `mage test-pkg ./internal/adapters/mcp_rpc` (pass — 232 tests, 0 failures; regression-clean)
- **Notes:**
  - **Schema enum change:** Updated `registerCommentTools` line 2243. Changed both the `mcp.Description(...)` string from `"project|branch|phase|actionItem|subtask|decision|note"` to `"project|action_item|actionItem"`, and the `mcp.Enum(...)` values from the 7-token pre-1.75 set to the 3-token post-1.75 set.
  - **TDD cycle notes:** This droplet modifies a string literal in an existing function, not a new function. "RED before GREEN" applies to new production functions; for a schema-string replacement, the behavior is declared (enum values are schema metadata), not computed. The regression test `TestIsValidCommentTargetTypeLegacyTokensRejected` documents existing-correct domain behavior. It went GREEN immediately because `IsValidCommentTargetType` already rejects legacy tokens — this is expected (regression guard, not new behavior).
  - **Sibling D1.1 coordination:** `internal/domain/comment_test.go` was dirty (D1.1 modified it). Per PLAN.md fallback rule, added the `IsValidCommentTargetType` legacy-token regression test to `internal/adapters/mcp_rpc/extended_tools_test.go` instead of the domain file. This is within the declared `paths` for D1.2.
  - **`mage test-pkg` path format (same discovery as D1.1):** Bare path `internal/adapters/mcp_rpc` (no `./`) causes go to treat it as a non-existent module import path and silently fail with 0 tests + exit 1. Must use `./internal/adapters/mcp_rpc`. Acceptance criteria in PLAN.md uses the bare form — the intent is the directory-local form. Used `./` form for all test runs.
  - **Pre-existing `internal/app/dispatcher` compile failure:** When running `mage ci`, `internal/app/dispatcher` shows a compile/build failure (0.00s, 0 tests). This is a pre-existing failure due to sibling D1.3 working in progress (it removed symbols from `subscriber_test.go` and created `dispatcher_e2e_test.go` as an untracked file, leaving the dispatcher package in an incomplete state). This is orthogonal to D1.2's scope; `internal/adapters/mcp_rpc` passes cleanly (232 tests).
  - **Scope:** Stayed strictly within declared paths. No other files touched except appending the regression test to `extended_tools_test.go`.

### Hylla Feedback

- **Query:** `mcp__hylla__hylla_search_keyword` for `registerCommentTools target_type enum`
- **Missed because:** Hylla MCP tools (`mcp__hylla__*`) not available in this agent session — same as D1.1 builder.
- **Worked via:** `Read` on `internal/adapters/mcp_rpc/extended_tools.go` at known line range (2220–2270) and `internal/domain/comment.go` for `IsValidCommentTargetType`.
- **Suggestion:** Same as D1.1: ensure `mcp__hylla__*` tools are in the agent's MCP allowlist. The tool namespace was entirely absent.

## Droplet 1.3 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-18
- **Files touched:**
  - `internal/app/dispatcher/subscriber_test.go` (removed `stubE2ETemplateResolver`, `TestAutoDispatchE2EGatePassViaNewDispatcher`, `TestAutoDispatchE2EGateFailViaNewDispatcher`, `templates` import)
  - `internal/app/dispatcher/dispatcher_e2e_test.go` (NEW file — `TestMain` + goleak wiring + moved + renamed e2e symbols)
- **Build-tool targets run:**
  - `mage test-func ./internal/app/dispatcher TestAutoDispatch_NewDispatcherGateWiring` (GREEN — 1/1 pass)
  - `mage test-func ./internal/app/dispatcher TestAutoDispatchE2EGateFailViaNewDispatcher` (GREEN — 1/1 pass)
  - `mage test-func ./internal/app/dispatcher TestAutoDispatchE2EGatePassViaNewDispatcher` (0 tests — old name confirmed absent)
  - `mage test-pkg ./internal/app/dispatcher` (pass — 389 tests, 0 failures; goleak found no leaks)
- **Notes:**
  - **R6.1 rename:** `TestAutoDispatchE2EGatePassViaNewDispatcher` → `TestAutoDispatch_NewDispatcherGateWiring`. Moved to new file as part of R7.4.
  - **R6.2 goleak approach — `TestMain` via `goleak.VerifyTestMain`:** Used `TestMain` (not per-test `VerifyNone(t)`) because no goroutine leaks were found in the full package run. The `TestMain` approach was confirmed correct: `goleak.VerifyTestMain(m)` returns `void` and handles `os.Exit` internally — callers must NOT wrap it in `os.Exit(...)`. Initial attempt used `os.Exit(goleak.VerifyTestMain(m))` which caused a build error ("used as value"). Fixed to plain `goleak.VerifyTestMain(m)` per goleak source at `testmain.go`.
  - **R6.3 clarifying comment:** Added to the `lister.calls.Load()` assertion in `TestAutoDispatch_NewDispatcherGateWiring` (the moved test): `// "state transitions" in the D5 spec means dispatcher lifecycle (Start/Stop), // not action-item state. This lister-calls pin is the lifecycle-transition signal.`
  - **R7.4 file split:** Created `dispatcher_e2e_test.go`. `stubE2ETemplateResolver` moved verbatim. Both e2e tests moved; gate test renamed. `templates` import removed from `subscriber_test.go` (no remaining usage); `errors` import kept (used by other tests in `subscriber_test.go`).
  - **goleak TestMain scope-package-wide:** Confirmed the `TestMain` applies to all 389 tests in the dispatcher package, not just the 2 e2e tests. All pass without goroutine leaks. The scope-creep guard was not triggered — no unrelated leaks surfaced.
  - **`mage test-pkg` path format:** Same pre-existing issue as D1.1/D1.2. `mage test-pkg internal/app/dispatcher` (no `./`) fails with 0 tests + exit 1 because Go interprets the bare path as a non-existent module import path. `mage test-pkg ./internal/app/dispatcher` (with `./`) passes correctly. The acceptance criteria in PLAN.md uses the bare format — the intent is the directory-local form; using `./` satisfies it.

## Out-of-Scope Leak Findings

None. `goleak.VerifyTestMain` ran against all 389 tests in the dispatcher package and found no goroutine leaks. The `TestMain` approach was not downgraded to per-test `VerifyNone(t)`.

### Hylla Feedback

- **Query:** `mcp__hylla__*` tools not available in this agent session (same as D1.1/D1.2).
- **Missed because:** Hylla MCP tool namespace absent.
- **Worked via:** `Read` on `subscriber_test.go` (line ranges 509-667), `dispatcher_e2e_test.go` (new file), plus goleak source at `~/go/pkg/mod/go.uber.org/goleak@v1.3.0/testmain.go` via `Bash cat` to verify the `VerifyTestMain` return type.
- **Suggestion:** Same as D1.1/D1.2: ensure `mcp__hylla__*` tools are in the agent's MCP allowlist.

## Droplet 1.2 — Round 2

- **Builder:** go-builder-agent
- **Started:** 2026-05-18
- **Files touched:**
  - `internal/adapters/mcp_rpc/extended_tools_test.go` (replaced `TestIsValidCommentTargetTypeLegacyTokensRejected`; renamed `TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes`)
- **Build-tool targets run:**
  - `mage test-func ./internal/adapters/mcp_rpc TestHandlerCommentToolTargetTypeEnumSchemaGuard` (GREEN — 1/1 pass)
  - `mage test-func ./internal/adapters/mcp_rpc TestHandlerCommentToolForwardsArbitraryTargetTypesToService` (GREEN — 1/1 pass)
  - `mage test-func ./internal/adapters/mcp_rpc TestIsValidCommentTargetTypeLegacyTokensRejected` (0 tests — old name confirmed absent)
  - `mage test-func ./internal/adapters/mcp_rpc TestHandlerExpandedCommentToolsForwardHierarchyTargetTypes` (0 tests — old name confirmed absent)
  - **Red-green verification:** Temporarily reverted D1.2 enum to pre-1.75 form (`"project", "branch", "phase", "actionItem", "subtask", "decision", "note"`); ran `mage test-func ./internal/adapters/mcp_rpc TestHandlerCommentToolTargetTypeEnumSchemaGuard` → confirmed FAIL at line 5996 (`till.comment target_type enum = [project branch phase actionItem subtask decision note], want exactly [project action_item actionItem]`). Restored correct enum. Confirmed GREEN.
  - `mage testPkg ./internal/adapters/mcp_rpc` (pass — 232 tests, 0 failures)
- **Notes:**
  - **Attack 5 fix:** Replaced `TestIsValidCommentTargetTypeLegacyTokensRejected` (domain-layer test, wrong layer) with `TestHandlerCommentToolTargetTypeEnumSchemaGuard` (schema-introspection test). The new test calls `tools/list`, extracts the `till.comment` schema, and uses `schemaPropertyEnumStrings` to assert the `target_type` enum equals exactly `["project", "action_item", "actionItem"]`. Also asserts the description string contains `"project|action_item|actionItem"` and does not contain stale tokens (`branch`, `phase`, `subtask`, `decision`, `note`). Red-green cycle confirmed — the test correctly fails when the D1.2 enum is reverted, proving it guards the actual schema change.
  - **Attack 2 fix — Option C:** Chose Option C (rename + annotate). The test body intentionally sends `"branch"` and `"phase"` to verify the MCP handler layer does NOT filter `target_type` — any value passes through to the service. This is transport-layer permissiveness testing, not a backward-compat retention test. The test body is semantically correct as-is; only the name and doc comment needed updating. Renamed to `TestHandlerCommentToolForwardsArbitraryTargetTypesToService`; doc comment now explicitly documents the transport-layer-permissiveness intent and explains why legacy tokens are used.
  - **Option A vs C decision rationale:** Option A would have replaced `"branch"`/`"phase"` with `"project"`/`"action_item"`, but that would destroy the test's ability to assert "handler does not validate at the MCP layer." Using current-vocab tokens would make the test indistinguishable from a schema-compliance test. Option C preserves the test's unique value: proving the handler is not the validation boundary. Renamed name and doc comment make the intent explicit so the next reader does not mistake it for a bug.
  - **Scope:** Only `extended_tools_test.go` was edited. `extended_tools.go` was not touched (the temporary revert was immediately restored; the file ends this round in its D1.2 post-round-1 state).

### Hylla Feedback

N/A — task touched no committed Go files via Hylla (Hylla MCP tools remain absent from this session per the 2026-05-18 disable notice). Used `Read` on the test file and `extended_tools.go` at known line ranges. Same absence as prior rounds.
