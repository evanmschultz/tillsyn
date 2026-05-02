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
