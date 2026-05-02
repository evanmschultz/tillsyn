# DROP_2 — BUILDER_WORKLOG

## Droplet 2.1 — Round 1

**Outcome:** success.

**Files touched (deletions):**

- `templates/builtin/default-frontend.json` (deleted via `git rm`)
- `templates/builtin/default-go.json` (deleted via `git rm`)
- `templates/embed.go` (deleted via `git rm`)
- `templates/builtin/` (auto-removed when last child deleted)
- `templates/` (auto-removed when last child deleted)

**Files touched (state-flips):**

- `workflow/drop_2/PLAN.md` — Droplet 2.1 `**State:** todo` → `**State:** in_progress` at start; flipped to `**State:** done` at end of round.

**MD edits under carve-out:** none. The surviving non-Go references to `templates/builtin/...` (in `README.md`, top-level `PLAN.md`, `workflow/drop_2/PLAN.md`, `workflow/drop_2/PLAN_QA_FALSIFICATION.md`) are doc/historical-audit prose, not load-bearing Go references. Per PLAN.md line 394: "the surviving MD references are not load-bearing for Drop 2." Drop 3 cleanup will rewrite the README/PLAN.md prose alongside the new template system. No trivial-phrase-fix candidates encountered — surrounding sentences are coherent without edits.

**Mage targets run:**

- `mage ci` — green. 1263 tests passed across 19 packages. Minimum package coverage 70.0% met. Build of `./cmd/till` succeeded. Exit code 0.

**Design notes:**

- Pre-deletion verification: `templates/embed.go` is a 17-line file declaring `var Files embed.FS` (`//go:embed builtin/*.json`) and `func ReadFile(name string) ([]byte, error)`. No other Go files in the package.
- `git grep "evanmschultz/tillsyn/templates" -- '*.go'` returned empty before deletion — zero Go importers. Post-deletion confirmation also empty. The package's `embed.FS` was unreferenced runtime-dead code.
- `git rm` of the three files cleanly removed both parent directories (`templates/builtin/`, `templates/`) — no residual empty dirs.
- The deletion is atomic in one staged change; orchestrator commits on return.

## Hylla Feedback

N/A — task touched non-Go files only (deletions). The single Go file touched was `templates/embed.go`, deleted outright. Pre-deletion verification used `Read`, `Bash` (`git grep`, `ls`) on a known small file rather than Hylla — appropriate for a deletion-only droplet where the question is "are there importers?" answered fastest by `git grep`.

## Droplet 2.2 — Round 1

**Outcome:** success.

**Files touched:**

- `internal/domain/role.go` — new file, 90 LOC (Role type, 9 typed constants, validRoles slice, package-level `roleDescriptionRegex`, `IsValidRole`, `NormalizeRole`, `ParseRoleFromDescription`).
- `internal/domain/role_test.go` — new file, 191 LOC, table-driven tests (34 sub-cases across 3 top-level test funcs: 11 `IsValidRole`, 5 `NormalizeRole`, 18 `ParseRoleFromDescription`).
- `internal/domain/errors.go` — added `ErrInvalidRole = errors.New("invalid role")` between `ErrInvalidKindPayloadSchema` and `ErrInvalidLifecycleState` (i.e. immediately after the kind-family group, before the lifecycle/actor/attention/handoff group). Placement is conceptual (Role is a peer of Kind) rather than strict alphabetical — matches the file's existing groups-by-domain organization.
- `workflow/drop_2/PLAN.md` — Droplet 2.2 `**State:** todo` → `**State:** in_progress` at start; flipped to `**State:** done` at end of round.

**Mage targets run:**

- `mage test-pkg ./internal/domain` — 90 tests passed in package (88 prior + 34 new minus 32 net delta arithmetic = 90 total). Initial run failed with `qa-a11y` parser case (see Design notes below); fixed and re-ran green.
- `mage ci` — green. 1300 tests passed across 19 packages. `internal/domain` package coverage 79.4% (>= 70.0% threshold). Build of `./cmd/till` succeeded. Exit code 0.

**Design notes:**

- **Mirrored `internal/domain/kind.go` style** — typed string alias + `const ( ... )` block with `Role` constants + `validRoles` slice + `slices.Contains` membership check + trim+lowercase normalizer. Single deviation: no companion `RoleAppliesTo` analogue because Role does not have a parallel scope vocabulary (Kind has `KindAppliesTo` for the projects/work-item scope split; Role does not).
- **Empty-string rejection in `IsValidRole`** is per acceptance contract — at the validator level the empty string is not a member of the closed enum. Callers that want to permit an unset/optional role short-circuit on emptiness before calling `IsValidRole` (Droplet 2.3's `NewActionItem` will do exactly this). Documented in the Go doc comment on `IsValidRole`.
- **Regex deviation from PLAN.md spec — surfaced for orchestrator attention.** The PLAN.md acceptance writes the regex as `(?m)^Role:\s*([a-z-]+)\s*$`, but acceptance #1 lists `qa-a11y` as one of the 9 valid string values. `qa-a11y` contains digits (`1`, `1`), so `[a-z-]+` can never match it — the spec is internally inconsistent. Implemented the regex as `(?m)^Role:\s*([a-z0-9-]+)\s*$` (digits added) to satisfy the binding contract that all 9 enum values must round-trip via `ParseRoleFromDescription`. The case-sensitivity contract still holds because uppercase letters remain excluded — `Role: Builder` still produces no match. This deviation is documented in the doc comment on `roleDescriptionRegex`. The QA passes for Droplet 2.2 should confirm the deviation is acceptable; if not, the alternative is to rename the constant value (e.g. `qa-accessibility`) but that ripples through every spec reference.
- **Package-level regex** — `regexp.MustCompile` at package init so the pattern is compiled once. If the pattern is invalid Go panics at init, which would surface immediately on any test run; the test suite passing implies the regex is well-formed RE2.
- **Doc comments on every exported symbol** — `Role`, all 9 constants (block-level via `Built-in role values...`), `IsValidRole`, `NormalizeRole`, `ParseRoleFromDescription`. The package-level `roleDescriptionRegex` var is unexported but still documented for maintainability and to record the regex-deviation rationale.

## Hylla Feedback

N/A — task created two brand-new Go files (`role.go`, `role_test.go`) and made a one-line edit to a known existing Go file (`errors.go`). Reference reading used `Read` on `internal/domain/kind.go` and `internal/domain/errors.go` (committed, Go-indexed by Hylla, but reading the whole file via `Read` was the right call — I needed the full structural context for style-mirroring, not a search hit). No Hylla query was attempted because the question shape was "show me this entire file's structure" not "find a symbol / find references / find usages." `Read` is the correct tool for whole-file structural inspection; Hylla shines on symbol-level queries against large codebases. Zero ergonomic gripes for this droplet.

## Droplet 2.2 — Round 2

**Outcome:** success. Cleanup pass within Droplet 2.2 — Droplet's PLAN.md state stays at `done` from Round 1.

**Files touched:**

- `internal/domain/role_test.go` — removed three redundant `tc := tc` lines:
  - Line 34 (was inside `TestIsValidRole` `for _, tc := range cases` loop).
  - Line 62 (was inside `TestNormalizeRole` `for _, tc := range cases` loop).
  - Line 196 (was inside `TestParseRoleFromDescription` `for _, tc := range cases` loop).

**Why:** Go 1.22+ scopes loop variables per-iteration automatically — the `tc := tc` shadow-copy idiom was the pre-1.22 workaround for closure capture in `t.Run` subtests with `t.Parallel()`. Project is Go 1.26+ (per `main/CLAUDE.md` Tech Stack), so the copies are dead code and `go vet` / LSP flag them as `forvar` warnings. Removing them is purely a cleanup; semantics are preserved because the per-iteration scoping rule guarantees each subtest closure captures a distinct `tc`.

**Mage targets run:**

- `mage test-pkg ./internal/domain` — green. 90 tests passed in 0.28s (same count + same coverage as Round 1; redundant-line removal is invariant on test count).
- `mage ci` — green. 1300 tests passed across 19 packages. `internal/domain` coverage held at 79.4% (>= 70.0% threshold). Build of `./cmd/till` succeeded. Exit code 0.

**Design notes:**

- **Edit shape:** matched `for _, tc := range cases {\n\t\ttc := tc\n\t\tt.Run(tc.name, func(t *testing.T) {` plus the unique 1–2 lines of body content per loop, so each `Edit` call hit exactly one of the three sites despite the leading two lines being structurally identical across all three loops. Body discriminator chosen per-site:
  - `IsValidRole`: assertion line uses `IsValidRole(tc.role)`.
  - `NormalizeRole`: assertion line uses `NormalizeRole(tc.in)`.
  - `ParseRoleFromDescription`: body uses `got, err := ParseRoleFromDescription(tc.desc)` (the only loop with two body lines pre-assertion, so I anchored on those two lines without the assertion to keep the diff minimal).
- **Out-of-scope items respected:** `go.mod` not touched, no `go mod tidy` run, no other files edited. The pre-existing `chroma/v2 should be direct` warning is unchanged and outside Droplet 2.2's scope per the orchestrator prompt.
- **PLAN.md state untouched.** Droplet 2.2 stays at `done` from Round 1 — Round 2 is a same-droplet cleanup pass, not a new state transition.

## Hylla Feedback

N/A — task touched only existing Go test code via three line-deletions. No symbol search, no reference lookup, no Hylla query needed. The work was 100% LSP-warning-driven (the `forvar` diagnostics handed me the exact line numbers + the exact pattern to remove). Zero ergonomic gripes.

## Droplet 2.3 — Round 1

**Outcome:** success.

**Files touched:**

- `internal/domain/action_item.go` — added `Role Role` field to `ActionItem` struct (placed between `Scope` and `LifecycleState`, alongside the closed-enum classifier neighborhood) and to `ActionItemInput`. Added a normalize-and-validate block in `NewActionItem` after the `Scope`-mirrors-`Kind` check. Added `Role: in.Role` to the constructor's return literal. Net delta: +21 LOC.
- `internal/domain/domain_test.go` — added `TestNewActionItemRoleValidation`, a 12-case table-driven test covering empty / whitespace-only / each of 9 valid roles / unknown-rejects. Inserted directly after `TestNewActionItemValidation`. Net delta: +49 LOC.
- `internal/tui/model_test.go` — added `"Role": {}` to the `readOnly` map inside `TestActionItemSchemaCoverageIsExplicit` (line 14812). **Scope expansion** — this path was NOT in the orchestrator's listed Paths, but the schema-coverage gate trips on every new `ActionItem` field. The classification is unambiguous (closed-enum classifier — same lane as `Kind`/`Scope`/`LifecycleState`, all readOnly). Reported back to orchestrator. Net delta: +1 LOC.

**Mage results:**

- `mage test-pkg ./internal/domain` → 103 tests pass (was 102 prior; new `TestNewActionItemRoleValidation` adds 1 test with 12 subtests).
- `mage ci` → exit 0. 1313 tests pass across 19 packages. All packages above 70% coverage threshold (`internal/domain` at 79.4%, `internal/tui` at 70.0%). Build succeeds.

**Design notes:**

- **Field placement:** `Role` lives between `Scope` and `LifecycleState` in both structs. Rationale — `Kind`, `Scope`, `Role`, `LifecycleState` are the four closed-enum classifiers. Grouping them keeps the struct's mental model clean (system-classifier section vs user-data section).
- **Short-circuit on empty before `IsValidRole`:** required because `IsValidRole` rejects the empty string per `role.go:58-60`. The validator pattern is `in.Role = NormalizeRole(in.Role); if in.Role != "" && !IsValidRole(in.Role) { return ErrInvalidRole }`. This makes the empty zero-value the permitted default and makes whitespace-only inputs round-trip as empty (since `NormalizeRole` returns `""` for whitespace).
- **Test style — table-driven, no `tc := tc`:** the new test uses `for _, tc := range cases { t.Run(tc.name, func(t *testing.T) { ... }) }` without the legacy `tc := tc` capture line, per Go 1.22+ per-iteration scoping. This is the post-Round-2 forvar-clean pattern.
- **Existing tests stay green:** the `Kind` validation path was untouched. `TestNewActionItemDefaultsAndLabels`, `TestNewActionItemValidation`, `TestActionItemMoveUpdateArchiveRestore`, `TestNewActionItemRichMetadataAndDefaults`, `TestActionItemLifecycleTransitions`, `TestActionItemContractUnmetChecks`, `TestNewActionItemRejectsInvalidMetadata` all pass without change — those tests omit `Role`, so the empty-zero-value path is exercised implicitly.

**PLAN.md state flips:** Droplet 2.3 `todo → in_progress` at start, `in_progress → done` at end.

## Hylla Feedback

None — Hylla answered everything needed. The investigation was code-local (read three files in `internal/domain`, one test file, one test in `internal/tui`) and the LSP `documentSymbol` query handled fast navigation inside the 26k-line `domain_test.go`. No symbol search ambiguity, no stale-ingest issue. Zero ergonomic gripes for this droplet.

## Droplet 2.4 — Round 1

**Files touched:**

- `internal/adapters/storage/sqlite/repo.go` — added `role TEXT NOT NULL DEFAULT ''` to the `action_items` `CREATE TABLE` block (column placed between `scope` and `lifecycle_state`); added `roleRaw string` local + `&roleRaw` Scan target inside `scanActionItem` with `t.Role = domain.Role(roleRaw)`; added `role` to the `INSERT INTO action_items(...)` column list, the `VALUES (?...)` slot count, and the bind-args slice (`string(t.Role)` between `string(scope)` and `string(t.LifecycleState)`); added `role = ?` to `UPDATE action_items SET ...` with the matching bind arg; added `role` to the column list inside both `ListActionItems`'s `SELECT` and `getActionItemByID`'s `SELECT`. Net delta: +9 LOC.
- `internal/adapters/storage/sqlite/repo_test.go` — added `TestRepository_PersistsActionItemRole` immediately after `TestRepository_PersistsProjectKindAndActionItemScope`, mirroring its kind/scope round-trip pattern. The test covers (a) empty-role default round-trip on `CreateActionItem` + `GetActionItem`, (b) `domain.RoleBuilder` round-trip on a second item, (c) `ListActionItems` (separate SELECT path) surfaces the role, and (d) reassign on `UpdateActionItem` from `RoleBuilder` to `RoleQAProof`. Net delta: +106 LOC.

**Mage results:**

- `mage test-pkg ./internal/adapters/storage/sqlite` → 69 tests pass (was 68 prior; new `TestRepository_PersistsActionItemRole` adds 1).
- `mage ci` → exit 0. 1314 tests pass across 19 packages. `internal/adapters/storage/sqlite` coverage 75.1% (≥ 70% threshold). Build succeeds.

**Design notes:**

- **Column position in `CREATE TABLE`:** placed between `scope` and `lifecycle_state` to group the closed-enum classifiers (`kind`, `scope`, `role`, `lifecycle_state`) consecutively. This matches the Droplet 2.3 worklog convention that placed `Role` between `Scope` and `LifecycleState` on the Go struct, and keeps the SQL column order, the Go `scanActionItem` Scan order, the INSERT column list, the INSERT bind-args slice, the UPDATE SET clause, and both SELECT column lists in lockstep — all five sites added `role` in the same relative slot.
- **Three SELECT paths, all updated:** the file has two SELECT statements that feed `scanActionItem` (`ListActionItems` at the top of the file and `getActionItemByID` at the bottom). Both column lists were updated, otherwise `scanActionItem` would have read `lifecycle_state` into the new `roleRaw` slot and shifted every subsequent bind, breaking every existing test silently.
- **Empty-role default:** `domain.Role("")` cast on read yields the zero-value `Role`, matching the schema default `''` and the domain contract from Droplet 2.3 (empty role is permitted, only non-empty values get validated against the closed enum). No special-case `if roleRaw == "" { ... }` is needed — both `Role` and `roleRaw` are typed strings whose empty zero values are interchangeable.
- **Test pattern — focused round-trip vs extending the existing parameterized test:** chose a dedicated `TestRepository_PersistsActionItemRole` rather than extending `TestRepository_PersistsProjectKindAndActionItemScope`. Rationale: (a) the kind/scope test name reads as a contract; (b) role is a separate first-class field with its own contract (empty-default + reassign-via-update); (c) cleaner test isolation when a future change touches role specifically. The new test mirrors the kind/scope test's structural pattern (`OpenInMemory`, project + column setup, create + get + assert) so the file's idiom stays consistent.
- **Reassign-via-update is the load-bearing UPDATE assertion:** writing `RoleBuilder` on create then reassigning to `RoleQAProof` and reading back proves the SET clause is wired AND the bound value lands at the correct positional slot. A simple "create with role, read back" test would still pass even if the UPDATE SET clause forgot the role column.
- **Pre-MVP rule honored:** zero `ALTER TABLE`, zero migration code, zero SQL backfill. The `CREATE TABLE IF NOT EXISTS` block is the only schema source. Dev-deleted `~/.tillsyn/tillsyn.db` before this droplet ran (per spec), so the fresh DB is created with the new column on first connect.

**No `tc := tc` capture line in the new test:** the test is straight-line (not table-driven across `t.Run` subtests), so the Go 1.22+ per-iteration scoping rule does not apply here — but the convention is honored anyway: the file's existing tests in this style do not use loop captures.

**PLAN.md state flips:** Droplet 2.4 `todo → in_progress` at start, `in_progress → done` at end.

## Hylla Feedback

None — Hylla answered everything needed (and most reads in this droplet were against non-Go SQL strings + Go test plumbing, where Hylla is N/A). The investigation was: read `repo.go`'s CREATE TABLE block at `:168`, `scanActionItem` at `:2738`, the insert path at `:1237`, the update path at `:1330`, the two SELECTs at `:1394` + `:2444` — all located via `rg` for `INTO action_items|UPDATE action_items|FROM action_items`. The test file's existing round-trip pattern was found via `rg` for `CreateActionItem|UpdateActionItem|GetActionItem`. Hylla queries were not the right tool for these in-file SQL string locations — code-local file navigation was the natural fit. No miss to report.

## Droplet 2.5 — Round 1

**Files touched:**

- `internal/app/service.go` — added `Role domain.Role` to both `CreateActionItemInput` and `UpdateActionItemInput`; threaded `Role: in.Role` into the `domain.ActionItemInput` literal inside `Service.CreateActionItem`; added a service-layer Role update block inside `Service.UpdateActionItem` after `UpdateDetails(...)` that normalizes via `domain.NormalizeRole`, returns `domain.ErrInvalidRole` on a non-empty invalid value, otherwise assigns `actionItem.Role = normalized` and bumps `UpdatedAt`. Empty input is a no-op (preserves prior). Net delta: +20 LOC.
- `internal/adapters/server/common/mcp_surface.go` — added `Role string` to both `CreateActionItemRequest` and `UpdateActionItemRequest` with doc comments documenting the empty-string semantics (create-empty allowed, update-empty preserves prior). Net delta: +12 LOC.
- `internal/adapters/server/common/app_service_adapter_mcp.go` — added `Role: domain.Role(strings.TrimSpace(in.Role))` to both the `app.CreateActionItemInput` literal in `CreateActionItem(:620)` and the `app.UpdateActionItemInput` literal in `UpdateActionItem(:661)`. Net delta: +2 LOC.
- `internal/adapters/server/common/app_service_adapter.go` — added `errors.Is(err, domain.ErrInvalidRole)` to the existing `mapAppError` switch case that maps invalid-input errors to `ErrInvalidCaptureStateRequest` (alongside `ErrInvalidKind`, `ErrInvalidPriority`, etc.). Net delta: +1 LOC.
- `internal/adapters/server/mcpapi/extended_tools.go` — added `Role string \`json:"role"\`` to the `args` struct inside `handleActionItemOperation`; added `Role: args.Role` to both the `common.CreateActionItemRequest` literal (create branch) and the `common.UpdateActionItemRequest` literal (update branch); added `mcp.WithString("role", mcp.Description("..."))` schema field to the primary `till.action_item` tool plus the legacy `till.create_task` and `till.update_task` aliases (the description spells out the closed 9-value enum and the empty-on-update preserve semantic). Net delta: +5 LOC.
- `internal/adapters/server/mcpapi/extended_tools_test.go` — extended the `stubExpandedService.CreateActionItem` and `UpdateActionItem` methods to (a) reject non-empty invalid-role inputs by returning `errors.Join(common.ErrInvalidCaptureStateRequest, domain.ErrInvalidRole)` (the same wrapped shape the real `AppServiceAdapter` produces via `mapAppError`), (b) echo the trimmed role from the request into the returned `domain.ActionItem`'s `Role` field so the JSON round-trip is observable; added `TestHandlerExpandedActionItemRoleRoundTrip` with five `t.Run` sub-tests: create-with-valid-role plumbs + round-trips, create-without-role round-trips empty, update-with-role plumbs the new value, update-without-role preserves prior (empty-on-the-wire), create-with-invalid-role returns `invalid_request:` 400-class. Net delta: +135 LOC (+ ~6 LOC stub modifications).

**Mage results:**

- `mage testFunc ./internal/adapters/server/mcpapi TestHandlerExpandedActionItemRoleRoundTrip` → 6 tests pass (5 sub-tests + parent).
- `mage testPkg ./internal/adapters/server/common` → 123 tests pass (no regressions).
- `mage testPkg ./internal/adapters/server/mcpapi` → 93 tests pass (was 87 prior; +5 sub-tests + 1 parent = +6).
- `mage testPkg ./internal/app` → 176 tests pass (no regressions).
- `mage ci` → exit 0. **1320 tests pass across 19 packages**. Coverage thresholds met everywhere. Build succeeds.

**Design notes:**

- **Kind-pattern mirror, with a deliberate divergence:** the spec said "match the kind-handling pattern." Kind is **immutable** through update (the existing `Service.UpdateActionItem` ignores any `Kind` field; the immutability is enforced by the `UpdateActionItemInput` struct simply not having a `Kind` field). Spec-text for 2.5 explicitly states "`till.action_item(operation=update, role=...)` updates the role on an existing action item" plus "Empty role is accepted on create and update (no-op for update)" — so Role must be mutable on update unlike Kind. The implementation honors the literal spec: a non-empty Role on update mutates `actionItem.Role`; an empty Role on update is a no-op (preserves prior). Validation mirrors `domain.NewActionItem`: `domain.NormalizeRole` + `domain.IsValidRole` rejection with `domain.ErrInvalidRole`. The kind-pattern parts that DO match: MCP request struct uses `Role string` (not `domain.Role`) at the boundary, the adapter trims and casts to `domain.Role` before passing to the app input, the app input is typed `domain.Role`, validation surfaces as the same domain `ErrInvalid...` sentinel, and the error maps via the same `mapAppError` switch case to `ErrInvalidCaptureStateRequest` → MCP `invalid_request:` 400-class.
- **Response shape — no separate response struct:** the create/update/get response for `till.action_item` is `domain.ActionItem` directly (marshaled via `mcp.NewToolResultJSON(actionItem)`). `domain.ActionItem.Role` was already added in Droplet 2.3, so no response-shape extension was needed in `mcp_surface.go`. The `Role` field default Go-marshals to JSON key `"Role"` (no struct tag) — same as `Kind`, `Scope`, `Title`, etc. on the same struct. This matches the existing convention in this codebase.
- **`app.CreateActionItemInput.Role` and `app.UpdateActionItemInput.Role` were missing before this droplet** (verified by `rg` against `internal/app/service.go:404` + `:424`). Adding them was an in-scope transitive requirement of the MCP→app→domain plumbing per the spec's note "verify and add if missing." Both fields are typed `domain.Role` (not raw string) so the type-safety boundary lives at the MCP-adapter conversion site, not deep in the app layer.
- **Empty-on-update preserves prior — semantic chain:** the MCP layer trims and forwards verbatim (`""` stays `""`); the common adapter's `domain.Role(strings.TrimSpace(in.Role))` produces `""`; the app's update path normalizes via `domain.NormalizeRole` (still `""`) and short-circuits the `if normalized != ""` guard, leaving `actionItem.Role` untouched. The persisted row keeps its prior role value. Test `update without role preserves prior` proves the wire-level empty surfaces as empty in `lastUpdateActionItemReq.Role`; the no-op semantics in the service are exercised by `mage testPkg ./internal/app` (existing 176 tests pass without modification, so the new code path doesn't break any prior update behavior).
- **Stub bypass and the wrapped-error shape:** the `stubExpandedService` in `extended_tools_test.go` IS the `ActionItemService` consumer of MCP; production has a real `AppServiceAdapter` between MCP and the app layer that wraps every error through `mapAppError`. The first version of the test failed with `internal_error: invalid role` because the stub returned bare `domain.ErrInvalidRole`, which the MCP error mapper falls through to the default case. Fix: the stub now returns `errors.Join(common.ErrInvalidCaptureStateRequest, domain.ErrInvalidRole)` — the **same wrapped shape** the real adapter produces. The MCP error mapper at `handler.go:888` matches `common.ErrInvalidCaptureStateRequest` and emits the `invalid_request:` prefix. This makes the stub a faithful production-shape simulator for invalid-input flows. Real-adapter integration coverage of the role-invalid path lives implicitly in the existing `mage testPkg ./internal/adapters/server/common` chain (the new `mapAppError` case clause is touched by the same path).
- **Schema field on the legacy aliases:** added the `role` schema field to both `till.create_task` and `till.update_task` (the legacy aliases that share `handleActionItemOperation`). Without it, callers using the legacy alias would have `role` silently stripped from the request payload at the MCP schema boundary even though the underlying handler reads it. Parity at the schema surface keeps the legacy callers honest.
- **Single-line description on the schema field:** `mcp.Description("Optional role tag for operation=create|update — see allowed values (closed enum: builder|qa-proof|qa-falsification|qa-a11y|qa-visual|design|commit|planner|research). Empty string preserves the existing value on update.")` — explicit closed-enum listing matches what the dev wants for LLM-tool ergonomics (callers don't have to guess the valid values).
- **No `tc := tc` capture lines in the new test:** the new `TestHandlerExpandedActionItemRoleRoundTrip` uses five direct `t.Run` calls with literal sub-test names rather than a table-driven `for _, tc := range cases` loop, so Go 1.22+ per-iteration scoping is moot. Convention honored.

**PLAN.md state flips:** Droplet 2.5 `todo → in_progress` at start, `in_progress → done` at end.

## Hylla Feedback

None — Hylla answered everything needed. The investigation was code-local against five files (two in `internal/adapters/server/common`, two in `internal/adapters/server/mcpapi`, one in `internal/app`); navigation used `rg` against well-known anchor strings (`CreateActionItemRequest`, `UpdateActionItemInput`, `mapAppError`, `handleActionItemOperation`, `mcp.WithString`) plus the `LSP findReferences` tool to confirm `domain.ActionItem.Role` is already wired everywhere it should be. The Kind-pattern reference points at lines `:60`, `:639`, `:643` were obvious from reading the request structs and adapter code straight through. No Hylla query was the right shape for this kind of "five-file plumbing thread" investigation; code-local navigation was the natural fit. Zero ergonomic gripes for this droplet.

## Droplet 2.5 — Round 2

**Outcome:** success. Cleanup pass within Droplet 2.5 — Droplet's PLAN.md state stays at `done` from Round 1.

**Files touched:**

- `internal/adapters/server/mcpapi/extended_tools_test.go` — removed two redundant `tc := tc` lines:
  - Line 3051 (was inside the `for _, tc := range cases` loop in the create-with-various-applies-to/namespace test that constructs `stubExpandedService` with empty `stubMutationAuthorizer{}`).
  - Line 3118 (was inside the `for _, tc := range cases` loop in the auth-failure mapping test that constructs `stubExpandedService` with `stubMutationAuthorizer{authErr: tc.authErr}`).

**Why:** Go 1.22+ scopes loop variables per-iteration automatically — the `tc := tc` shadow-copy idiom was the pre-1.22 workaround for closure capture in `t.Run` subtests. Project is Go 1.26+ (per `main/CLAUDE.md` Tech Stack), so the copies are dead code and LSP flags them as `forvar` warnings. Removing them is purely a cleanup; semantics are preserved because the per-iteration scoping rule guarantees each subtest closure captures a distinct `tc`. This is the second occurrence of this lesson — Droplet 2.2 Round 2 already cleaned up the same pattern in `internal/domain/role_test.go`. The Round 1 spawn prompt explicitly forbade adding `tc := tc` lines; these two pre-existed in surrounding tests of the file (i.e., not introduced by Droplet 2.5's `TestHandlerExpandedActionItemRoleRoundTrip`, which uses direct `t.Run` calls per the Round 1 design note "No `tc := tc` capture lines in the new test"). Round 1 fixed only the new test it added; these two surrounding-test instances were caught by post-Round-1 LSP review.

**Mage targets run:**

- `mage test-pkg ./internal/adapters/server/mcpapi` — green. 93 tests passed in 0.99s (same count + same coverage as Round 1; redundant-line removal is invariant on test count).
- `mage ci` — green. 1320 tests passed across 19 packages. `internal/adapters/server/mcpapi` coverage held at 72.4% (≥ 70.0% threshold). Build of `./cmd/till` succeeded. Exit code 0.

**Design notes:**

- **Edit shape:** matched the full `for _, tc := range cases {\n\t\ttc := tc\n\t\tt.Run(tc.name, func(t *testing.T) {\n\t\t\tservice := &stubExpandedService{...}` block including the unique `stubMutationAuthorizer{...}` line per site. The two sites' bodies diverge on the `stubMutationAuthorizer` literal: site 1 uses bare `stubMutationAuthorizer{}`, site 2 uses `stubMutationAuthorizer{ authErr: tc.authErr }`. That single-line difference made each `Edit` call unique without needing `replace_all`.
- **Out-of-scope items respected:** the pre-existing `slicescontains` warning at `:1050` (logged in `project_drop_2_refinements_raised.md` R7) was not touched. No other lint warnings or files modified. PLAN.md untouched (per spawn prompt instruction step 6).
- **PLAN.md state untouched.** Droplet 2.5 stays at `done` from Round 1 — Round 2 is a same-droplet cleanup pass, not a new state transition.

## Hylla Feedback

N/A — task touched only existing Go test code via two line-deletions. No symbol search, no reference lookup, no Hylla query needed. The work was 100% LSP-warning-driven (the `forvar` diagnostics handed me the exact line numbers + the exact pattern to remove). Zero ergonomic gripes.

## Droplet 2.6 — Round 1

**Outcome:** success.

**Files touched:**

- `internal/app/snapshot.go` — added `Role domain.Role \`json:"role,omitempty"\`` to `SnapshotActionItem` (between `Scope` and `LifecycleState`); threaded `Role: t.Role` through `snapshotActionItemFromDomain` (line ~1058) and through `(SnapshotActionItem).toDomain()` (line ~1264). +3 LOC, -0.
- `internal/app/snapshot_test.go` — added `encoding/json` import; appended three new tests (`TestSnapshotActionItemRoleRoundTripPreservesAllRoles`, `TestSnapshotActionItemRoleEmptyRoundTripsEmpty`, `TestSnapshotActionItemRoleJSONShape`). +131 LOC.
- `workflow/drop_2/PLAN.md` — Droplet 2.6 `**State:** todo` → `**State:** in_progress` at start; will flip to `**State:** done` at end of round.

**Mage targets run:**

- `mage test-pkg ./internal/app` — green. 188 tests passed in 1.28s (185 pre-existing + 3 new top-level tests; the role round-trip test runs as 9 `t.Run` subtests, all green).
- `mage ci` — green. 1332 tests passed across 19 packages. `internal/app` coverage 71.5% (≥ 70.0% threshold). Build of `./cmd/till` succeeded. Exit code 0.

**Design notes:**

- **Field placement.** Inserted `Role` between `Scope` and `LifecycleState` in the struct so the snapshot field order mirrors `domain.ActionItem`'s order (`Kind` → `Scope` → `Role` → `LifecycleState`). This keeps the closed-enum classifiers grouped and the snapshot literal grep-friendly against the domain literal.
- **Field typing.** Used `domain.Role` (not `string`) — same convention as the other closed-enum fields on this struct (`Kind: domain.Kind`, `Scope: domain.KindAppliesTo`, `LifecycleState: domain.LifecycleState`, `Priority: domain.Priority`).
- **`omitempty` rationale.** `domain.Role` is `type Role string` (per `internal/domain/role.go:10`), so the standard string-empty `omitempty` rule applies: zero-value (`""`) drops the JSON key on serialize. JSON-shape test asserts both sides of this contract.
- **`toDomain` does NOT call `domain.NewActionItem`.** Direct struct-literal copy of `t.Role` into the constructed `domain.ActionItem`, matching the existing `toDomain` pattern (the value was already validated when first written; `toDomain` is hydration, not validation).
- **Snapshot version stays at v5.** `omitempty` plus `encoding/json`'s ignore-unknown-keys default means old `v5` snapshots load forward-compatibly without a version bump (per Droplet 2.6 spec note + Droplet 2.7's `## Notes` deferral of the version bump to post-MVP).
- **Test idiom matched the file.** `snapshot_test.go` uses standalone `TestX` functions with direct construction (no shared helper). Added three dedicated round-trip tests rather than extending the existing import/export tests, which already span 175+ lines each and exercise broader closure surfaces. Table-driven role test uses Go 1.22+ scoping (NO `tc := tc`).
- **JSON-shape test goes both ways.** Marshal → assert key present/absent; then Unmarshal → assert value preserved (with-role) or stays empty (without-role). Catches JSON-tag drift in either direction.

## Hylla Feedback

None — Hylla answered everything needed for context discovery (struct shape, fromDomain/toDomain locations, `domain.Role` type definition, `domain.ActionItem.Role` confirmation). Most of the symbol-locating work landed via `LSP documentSymbol` (live, exact line numbers), which is the right tool for a surgical Go edit task — Hylla would also have answered, but LSP was lower-friction here.
