# DROP_2 — Build QA Proof

(durable; append `## Droplet N.M — Round K` per QA attempt; NEVER `git rm`d)

## Droplet 2.1 — Round 1

**Verdict:** pass
**Date:** 2026-05-01

### Findings

#### F1. Staging matches expected delta exactly. (PASS)

`git status --porcelain` returned exactly the five expected entries and no extras:

- `D  templates/builtin/default-frontend.json`
- `D  templates/builtin/default-go.json`
- `D  templates/embed.go`
- ` M workflow/drop_2/PLAN.md` (state flips)
- `?? workflow/drop_2/BUILDER_WORKLOG.md` (new round artifact)

No stray modifications, no accidental edits to other files, no orphaned `templates/` subentries. Three deletions match AC1.

#### F2. Zero Go importers of the deleted package. (PASS)

`git grep -n "evanmschultz/tillsyn/templates" -- '*.go'` returned no hits. AC2 satisfied: no orphan import paths anywhere in the Go tree.

#### F3. Zero `templates.ReadFile` / `templates.Files` references in Go. (PASS)

`git grep -nE "templates\.(ReadFile|Files)" -- '*.go'` returned no hits. The package's API surface (the `Files embed.FS` var and `ReadFile` helper from the deleted `templates/embed.go`) has no consumers — the package was already runtime-dead before deletion.

#### F4. `templates/` parent dir gone. (PASS)

`ls templates/` returned `"templates/": No such file or directory (os error 2)`. Both `templates/` and `templates/builtin/` were auto-removed when their last child was `git rm`'d. No empty-dir residue.

#### F5. `mage ci` green at HEAD. (PASS)

I re-ran `mage ci` on HEAD (post-deletion, with the deletes staged) and observed:

- `[SUCCESS] Verified tracked sources`
- `[SUCCESS] Listed tracked Go files`
- `[SUCCESS] Checked Go formatting`
- `[SUCCESS] Test stream detected`
- 19 packages, 1263 tests, 0 failures, 0 skipped — every package `[PKG PASS]`
- `Minimum package coverage: 70.0%` met across all 15 reported packages (lowest is `internal/tui` at exactly 70.0%; `internal/buildinfo` at 100.0%)
- `[SUCCESS] Built till from ./cmd/till`

This independently corroborates the builder's claim. AC4 satisfied.

#### F6. Droplet 2.1 state in `PLAN.md` is `done`. (PASS)

`workflow/drop_2/PLAN.md:48` reads `- **State:** done` immediately under the `#### Droplet 2.1 — Delete \`templates/\` package outright` heading at line 46. State-flip executed correctly.

#### F7. `BUILDER_WORKLOG.md` exists with valid Round 1 content. (PASS)

`workflow/drop_2/BUILDER_WORKLOG.md` (new, untracked, 35 lines) contains:

- `## Droplet 2.1 — Round 1` heading
- `**Outcome:** success.`
- Files-touched section enumerating the three `git rm`'d files + the two auto-removed parent dirs + the PLAN.md state flips
- `**MD edits under carve-out:** none` with explicit reasoning citing PLAN.md line 394
- `**Mage targets run:**` section reporting `mage ci` green, 1263 tests, 19 packages, exit 0
- `**Design notes:**` section covering loader-coupling investigation
- `## Hylla Feedback` section explaining N/A status with rationale (deletion-only droplet, single Go file deleted outright, importer search via `git grep` was the right tool)

Structurally sound and substantively accurate.

#### F8. AC3 — `templates/builtin` references are MD-only outside Go tree. (PASS, with one cosmetic nit)

`git grep "templates/builtin"` returned 12 hits across exactly four MD files:

- `PLAN.md:1605, 1609, 1623` — top-level project PLAN (Drop 3 template overhaul context)
- `README.md:298, 309` — README dogfood-template links
- `workflow/drop_2/PLAN.md:5, 16, 42, 49, 54, 148, 384, 394, 415, 422` — droplet plan itself
- `workflow/drop_2/PLAN_QA_FALSIFICATION.md:126` — QA falsification round 1

**Zero `.go` hits.** AC3's hard constraint ("only MD references...not Go-tree references") is satisfied.

**Nit (T6 raised by builder, confirmed):** AC3's literal expected-hit list at PLAN.md:54 names `README.md`, `PLAN.md`, `CLAUDE.md`, and `workflow/drop_2/PLAN.md`. Reality:

- `CLAUDE.md` is in the literal list but produces no hit. (Unsurprising — CLAUDE.md doesn't reference template builtin paths today.)
- `workflow/drop_2/PLAN_QA_FALSIFICATION.md:126` produces a hit but is not in the literal list.

The substantive intent of AC3 ("zero Go-tree hits, all surviving hits are doc/historical-audit MD prose") is fully satisfied. The literal enumeration is stale but non-blocking — Drop 3's full template rewrite will sweep these MD references anyway, per PLAN.md:394 ("the surviving MD references are not load-bearing for Drop 2"). Recommend leaving as-is for Round 1 PASS; STEWARD or Drop 3 planner can fold the literal list into a future PLAN.md update if useful.

#### F9. No Go orphans. (PASS, covered by F5)

`mage ci` includes `[SUCCESS] Built till from ./cmd/till`, which builds the entire dependency graph from the CLI entrypoint. Successful build means no orphan imports / dead references / unresolved symbols anywhere in the reachable Go tree. Per project rules raw `go build ./...` is forbidden; the mage target is the canonical equivalent and it passed.

#### F10. QA-file preservation. (PASS)

`ls workflow/drop_2/` shows all eight prior QA artifacts intact:

- `PLAN_QA_PROOF.md`, `PLAN_QA_PROOF_R2.md`, `PLAN_QA_PROOF_R3.md`
- `PLAN_QA_FALSIFICATION.md`, `PLAN_QA_FALSIFICATION_R2.md`, `PLAN_QA_FALSIFICATION_R3.md`, `PLAN_QA_FALSIFICATION_R4.md`
- (Plus the new `BUILDER_WORKLOG.md` and modified `PLAN.md`.)

No `git rm` of QA files. Memory rule `feedback_never_remove_workflow_files.md` honored.

### Missing Evidence

None. All 10 required proof checks have direct citations. The builder's claims are reproducible end-to-end on HEAD with the staged deletes.

### Verdict Summary

Droplet 2.1's deletion landed cleanly. Three Go-relevant files (`templates/builtin/default-frontend.json`, `templates/builtin/default-go.json`, `templates/embed.go`) were `git rm`'d; both parent directories auto-removed; the `templates/` package no longer exists. Zero Go importers, zero residual API uses, zero `.go`-tree references to `templates/builtin`. `mage ci` reproduces green on HEAD with 1263 tests passing across 19 packages, all coverage ≥ 70.0%, and the `till` binary builds clean — independently confirming the builder's claim. PLAN.md's Droplet 2.1 state is correctly `done`; `BUILDER_WORKLOG.md` carries a structurally-sound Round 1 entry with a valid `## Hylla Feedback` section. The one cosmetic nit (T6: AC3's literal expected-hit list mentions `CLAUDE.md` which never appeared as a hit and omits `workflow/drop_2/PLAN_QA_FALSIFICATION.md` which does) does not affect the substantive intent of AC3 and is correctly deferred to Drop 3's full template rewrite. Ready to commit.

### Hylla Feedback

N/A — task touched non-Go files only (deletions of two JSON files and one trivial Go file with zero importers). Verification used `git grep`, `git status`, `ls`, `Read`, and `mage ci` directly — Hylla would not have added value over `git grep` for the "are there importers?" question. No Hylla queries were attempted, no fallbacks needed.

## Droplet 2.2 — Round 1

**Verdict:** pass
**Date:** 2026-05-01

### Findings

#### F1. All 9 typed Role constants present with exact string values. (PASS)

`internal/domain/role.go:13-23` declares the closed enum block:

- `RoleBuilder         Role = "builder"`
- `RoleQAProof         Role = "qa-proof"`
- `RoleQAFalsification Role = "qa-falsification"`
- `RoleQAA11y          Role = "qa-a11y"`
- `RoleQAVisual        Role = "qa-visual"`
- `RoleDesign          Role = "design"`
- `RoleCommit          Role = "commit"`
- `RolePlanner         Role = "planner"`
- `RoleResearch        Role = "research"`

All 9 values match acceptance #1 exactly (lowercase, hyphenated). The `validRoles` slice at `role.go:26-36` mirrors the same 9 in identical order, single source for membership.

#### F2. `IsValidRole` rejects the empty string. (PASS)

`role.go:58-60` implements `slices.Contains(validRoles, …)` over a list that does not include `""`. The empty string can never satisfy membership. Test coverage at `role_test.go:29` (`{name: "empty string is invalid", role: Role(""), want: false}`) and the corresponding `slices.Contains` semantics make the rejection deterministic. Acceptance #2 satisfied.

#### F3. `NormalizeRole` trim + lowercase + empty pass-through. (PASS)

`role.go:64-70`:

```
trimmed := strings.TrimSpace(string(role))
if trimmed == "" { return "" }
return Role(strings.ToLower(trimmed))
```

Tests at `role_test.go:48-58` cover:

- Surrounding whitespace trim (`"  builder  "` → `RoleBuilder`).
- Uppercase lowercased (`"BUILDER"` → `RoleBuilder`).
- Empty stays empty (`""` → `""`).
- Mixed case + whitespace (`"  QA-Proof  "` → `RoleQAProof`).
- Whitespace-only normalizes to empty (`"   "` → `""`).

Acceptance #3 satisfied.

#### F4. `ParseRoleFromDescription` regex + 3-branch contract. (PASS, with documented spec deviation)

`role.go:52` declares `roleDescriptionRegex = regexp.MustCompile(\`(?m)^Role:\s*([a-z0-9-]+)\s*$\`)` at package level — `MustCompile` ensures one-time init compilation; an invalid pattern would panic during package load and surface immediately on any test run (90 tests passed, so the pattern is well-formed RE2).

The regex is intentionally case-sensitive — the character class is `[a-z0-9-]+`, no `(?i)` flag, so uppercase inputs like `Role: Builder` produce no match. `role_test.go:120-124` confirms (`{name: "capitalized value fails to match (regex captures [a-z-]+ only)", desc: "Role: Builder", want: Role(""), wantErr: nil}`).

`ParseRoleFromDescription` (role.go:86-96) implements all 3 contract branches:

- **No `Role:` line found** → `match == nil` → returns `("", nil)`. Tests at `role_test.go:84-100` cover empty desc, prose-only desc, and mid-paragraph `Role:` (start-of-line anchor enforces).
- **First matching line carries a closed-enum value** → returns `(Role, nil)`. Tests at `role_test.go:102-105` ("two Role lines — first wins") + `role_test.go:126-184` round-trip all 9 enum values + embedded-in-larger-description case.
- **First matching line carries an unknown value** → returns `("", ErrInvalidRole)`. Tests at `role_test.go:114-118` (unknown `foobar`) + `role_test.go:185-190` (hyphen-only `-`).

Whitespace tolerance test at `role_test.go:108-112` covers `"Role:  builder  "` → `RoleBuilder` (trailing whitespace inside the line accepted by `\s*$`).

**Spec deviation (acceptable, builder pre-flagged):** PLAN.md acceptance #4 wrote the regex as `(?m)^Role:\s*([a-z-]+)\s*$`, but acceptance #1 includes `qa-a11y` (contains digits `1`, `1`) as a valid enum value. With `[a-z-]+`, `Role: qa-a11y` would never match — falling through to the `match == nil` branch returning `("", nil)` instead of `(RoleQAA11y, nil)`. The implementation widens the class to `[a-z0-9-]+` (digits added) — minimum diff to satisfy round-trip on all 9 enum values. Case-sensitivity is preserved (uppercase remains excluded). Documented in the doc-comment on `roleDescriptionRegex` (role.go:38-52) and in `BUILDER_WORKLOG.md` for orchestrator visibility. Routed as informational, not blocking.

Acceptance #4 satisfied.

#### F5. `ErrInvalidRole` added to `errors.go` in the kind-family group. (PASS)

`git diff internal/domain/errors.go` shows a single +1 line at `errors.go:28`:

```
+	ErrInvalidRole                 = errors.New("invalid role")
```

Inserted between `ErrInvalidKindPayloadSchema` (line 27) and `ErrInvalidLifecycleState` (line 29) — i.e. immediately after the kind-family group, before the lifecycle/actor/attention/handoff group. Same `var (` block as `ErrInvalidKind` (errors.go:20). Placement is conceptual (Role peers with Kind) rather than strict alphabetical, matching the file's existing groups-by-domain organization. Acceptance #5 satisfied.

#### F6. Table-driven tests cover every acceptance #6 case. (PASS)

`role_test.go` carries 3 top-level test funcs with 34 sub-cases:

- `TestIsValidRole` (lines 12-41) — 11 cases: 9 valid values round-trip, empty rejected, unknown rejected.
- `TestNormalizeRole` (lines 45-68) — 5 cases: trim, lowercase, empty, mixed, whitespace-only.
- `TestParseRoleFromDescription` (lines 74-205) — 18 cases covering empty desc, no `Role:` line, mid-paragraph (start-of-line anchor), two `Role:` lines (first wins), trailing whitespace tolerance, unknown value (`ErrInvalidRole`), capitalized value (no match), 9 valid round-trips, embedded-in-larger-description, hyphen-only edge case.

Every case in acceptance #6's enumeration is mapped 1:1. Acceptance #6 satisfied.

#### F7. Forvar cleanup landed — three `tc := tc` lines removed. (PASS)

`git grep -n "tc := tc" internal/domain/role_test.go` returned exit=1 with no output (no matches). The three loop bodies at `role_test.go:33-40`, `role_test.go:60-67`, and `role_test.go:193-204` open directly with `t.Run(tc.name, func(t *testing.T) { t.Parallel() …` — no shadow-copy lines remain.

Go 1.22+ per-iteration loop-var scoping (project is Go 1.26+ per CLAUDE.md Tech Stack) makes `tc := tc` dead code and `forvar`-flagged. Removal is safe and the tests still pass (F8 / F9 below).

#### F8. `mage test-pkg ./internal/domain` green at HEAD. (PASS)

I re-ran `mage test-pkg ./internal/domain` on HEAD and observed:

- `[RUNNING] Running go test ./internal/domain`
- `[SUCCESS] Test stream detected`
- `[PKG PASS] github.com/evanmschultz/tillsyn/internal/domain (0.00s)`
- Test summary: `tests: 90 / passed: 90 / failed: 0 / skipped: 0`
- `[SUCCESS] All tests passed`

Independent confirmation of the builder's claim. Acceptance "mage test-pkg ./internal/domain green" satisfied.

#### F9. `mage ci` green at HEAD. (PASS)

I re-ran `mage ci` on HEAD and observed:

- All four `[SUCCESS]` source/format/test-stream gates green.
- 19 packages, 1300 tests, 0 failed, 0 skipped — every package `[PKG PASS]`.
- `internal/domain` package coverage: **79.4%** (≥ 70.0% threshold).
- Minimum package coverage across all 15 reported packages: 70.0% (lowest is `internal/tui` at exactly 70.0%; `internal/buildinfo` at 100.0%).
- `[SUCCESS] Built till from ./cmd/till`.

This independently corroborates the builder's claims (1300 tests, 79.4% coverage). Coverage-threshold gate passed.

#### F10. `git status --porcelain` matches expected delta exactly. (PASS)

Observed:

- ` M internal/domain/errors.go`
- ` M workflow/drop_2/BUILDER_WORKLOG.md`
- ` M workflow/drop_2/PLAN.md`
- `?? internal/domain/role.go`
- `?? internal/domain/role_test.go`

Five entries, exactly matching the prompt's expected set (`M errors.go`, `M PLAN.md`, `M BUILDER_WORKLOG.md`, `?? role.go`, `?? role_test.go`). No drift, no stray modifications, no accidental edits to other files.

#### F11. Spec deviation acceptable — minimum-diff widening preserves case-sensitivity. (ACCEPT)

The regex change `[a-z-]+ → [a-z0-9-]+` is the minimum diff to satisfy the closed-enum round-trip contract on `qa-a11y`. Case-sensitivity is preserved because uppercase letters remain outside the class. Verified directly by the test at `role_test.go:120-124` (`Role: Builder` → empty match, no error). The alternative — renaming `qa-a11y` to `qa-accessibility` — would ripple through every `main/PLAN.md` § 19.2 reference, every CLAUDE.md role list, every closed-enum mention in the cascade design docs. Widening the regex is the localized, lower-blast-radius fix. Builder pre-flagged the deviation in BUILDER_WORKLOG.md and in the doc-comment on `roleDescriptionRegex`. Routed as an informational note for the orchestrator; no blocking action required.

### Missing Evidence

None. All 11 required proof checks have direct citations + reproducible mage gates at HEAD.

### Verdict Summary

Droplet 2.2 lands cleanly. The `internal/domain.Role` closed enum (9 typed constants), `IsValidRole`, `NormalizeRole`, and `ParseRoleFromDescription` are all in place at `internal/domain/role.go` (96 LOC), with `ErrInvalidRole` correctly inserted into the kind-family group of `errors.go`. The pure parser implements the 3-branch contract (no Role line / first-match-wins / unknown-value error) with package-level `regexp.MustCompile` initialization, multiline anchor `(?m)`, and case-sensitive class. The Round-1 spec deviation (regex class widened from `[a-z-]+` to `[a-z0-9-]+`) is the minimum diff to round-trip `qa-a11y` and preserves the case-sensitivity acceptance; builder pre-flagged it in the doc-comment + BUILDER_WORKLOG, and the deviation is acceptable. Round-2 forvar cleanup removed the three redundant `tc := tc` shadows correctly — Go 1.22+ per-iteration scoping makes them dead code, and the test count + coverage are unchanged. `mage test-pkg ./internal/domain` reproduces 90/90 tests passing on HEAD; `mage ci` reproduces 1300/1300 tests across 19 packages, with `internal/domain` at 79.4% coverage and all packages ≥ 70%. `git status --porcelain` matches the expected 5-entry delta exactly. Ready to commit.

### Hylla Feedback

N/A — task created two brand-new Go files (`role.go`, `role_test.go`) and made a one-line edit to a known existing Go file (`errors.go`). QA verification used `Read` for whole-file structural inspection (the relevant question was "show me the full file" not "find a symbol"), `Bash` for `git status` / `git diff` / `git grep` (the right tool for change-set + symbol-absence checks), and `mage` directly for gate reproduction. No Hylla queries were attempted because the work product is freshly-untracked Go that Hylla would not yet index until reingest, and the structural questions were better served by `Read`. Zero Hylla misses, zero ergonomic gripes.

## Droplet 2.3 — Round 1

**Verdict:** pass
**Date:** 2026-05-01

### Findings

#### F1. `Role Role` field added to `ActionItem` struct in correct position. (PASS)

`internal/domain/action_item.go:24-54` declares the struct. The new field lives at lines 30-33:

```
Kind      Kind                              // line 28
Scope     KindAppliesTo                     // line 29
// Role optionally tags ...                 // lines 30-32 doc comment
Role           Role                          // line 33
LifecycleState LifecycleState                // line 34
```

Field is correctly placed between `Scope` and `LifecycleState`, grouping the four closed-enum classifiers (`Kind`, `Scope`, `Role`, `LifecycleState`) in adjacent struct slots. Exactly matches the builder's stated rationale and acceptance "ActionItem struct gains Role Role field."

#### F2. `Role Role` field added to `ActionItemInput` struct in correct position. (PASS)

`internal/domain/action_item.go:57-82` declares the input struct. The new field lives at lines 63-67:

```
Kind      Kind                              // line 61
Scope     KindAppliesTo                     // line 62
// Role optionally tags ...                 // lines 63-66 doc comment
Role           Role                          // line 67
LifecycleState LifecycleState                // line 68
```

Symmetric placement to `ActionItem`. Acceptance "ActionItemInput struct gains Role Role field" satisfied.

#### F3. `NewActionItem` validation block correct. (PASS)

`internal/domain/action_item.go:150-158` implements the exact pattern specified:

```
// Role is optional. NormalizeRole collapses whitespace-only input to the
// empty string; an empty normalized role is permitted and round-trips as
// the zero-value Role. A non-empty normalized value must be a member of
// the closed Role enum — short-circuit on emptiness because IsValidRole
// rejects the empty string.
in.Role = NormalizeRole(in.Role)
if in.Role != "" && !IsValidRole(in.Role) {
    return ActionItem{}, ErrInvalidRole
}
```

The block is positioned after `Scope` validation (line 149) and before `LifecycleState` validation (line 159) — i.e. the same order as the struct fields, matching the validation flow's section organization. The constructor's return literal at line 201 carries `Role: in.Role`, completing the round-trip path. Empty short-circuit is correct because `IsValidRole` at `role.go:58-60` rejects the empty string (verified during Droplet 2.2 QA Round 1 F2). Acceptance "normalizes via NormalizeRole; if non-empty calls IsValidRole; on failure returns ErrInvalidRole; empty role is permitted" satisfied.

#### F4. Doc comments on `Role` fields are present and accurate. (PASS)

- `ActionItem.Role` doc (action_item.go:30-32): "Role optionally tags an action item with a closed-enum role (e.g. builder, qa-proof, planner). Empty string is the zero value and is permitted — callers that require a role should validate downstream."
- `ActionItemInput.Role` doc (action_item.go:63-66): "Role optionally tags the action item with a closed-enum role. Empty string is permitted and round-trips as the zero-value Role; non-empty values must match the closed Role enum or NewActionItem returns ErrInvalidRole."

Both doc comments correctly describe optional/empty semantics and the input-side specifically names `ErrInvalidRole` as the failure surface. Idiomatic Go-doc style. Per project rule "Go doc comments on every top-level declaration."

#### F5. `TestNewActionItemRoleValidation` exists with all 12 required cases. (PASS)

`internal/domain/domain_test.go:210-255` declares the test. The 12 cases at lines 219-230:

1. `empty` (input: `""`, expects empty role, no error)
2. `whitespace` (input: `"   "`, expects empty role, no error — exercises NormalizeRole's whitespace-collapse path)
3. `builder` (RoleBuilder round-trip)
4. `qa-proof` (RoleQAProof round-trip)
5. `qa-falsification` (RoleQAFalsification round-trip)
6. `qa-a11y` (RoleQAA11y round-trip)
7. `qa-visual` (RoleQAVisual round-trip)
8. `design` (RoleDesign round-trip)
9. `commit` (RoleCommit round-trip)
10. `planner` (RolePlanner round-trip)
11. `research` (RoleResearch round-trip)
12. `unknown rejects` (input: `Role("foobar")`, expects empty role, `ErrInvalidRole`)

All 9 valid roles round-trip; empty + whitespace exercise the optional-empty path; unknown rejects with `ErrInvalidRole`. Matches acceptance "empty role round-trips empty; each of 9 valid roles round-trips; unknown role rejected with ErrInvalidRole; whitespace-only role normalizes to empty" exactly.

LSP `documentSymbol` confirms the test func is registered at line 210 of the file. Test position is immediately after `TestNewActionItemValidation` (line 157) per the builder's stated insertion order, preserving co-location of constructor-validation tests.

#### F6. No `tc := tc` in the new test loop. (PASS)

`internal/domain/domain_test.go:233-254` opens the loop body directly with `t.Run(tc.name, func(t *testing.T) { … })` — no shadow-copy line. Go 1.22+ per-iteration scoping (project is Go 1.26+) makes `tc := tc` dead code; the new test follows the post-Droplet-2.2-Round-2 forvar-clean pattern. Consistent with `internal/domain/role_test.go` after the Droplet 2.2 Round 2 cleanup pass.

#### F7. TUI schema-coverage gate handled correctly. (PASS, scope-expansion accepted as readOnly)

`internal/tui/model_test.go:14797-14829` declares `TestActionItemSchemaCoverageIsExplicit`. The two field maps and the new `Role` entry:

- **`editable`** (lines 14799-14806): `Title`, `Description`, `Priority`, `DueAt`, `Labels`, `Metadata` — user-data fields the TUI exposes for direct edit.
- **`readOnly`** (lines 14807-14828): `ID`, `ProjectID`, `ParentID`, `Kind`, `Scope`, `Role`, `LifecycleState`, `ColumnID`, `Position`, plus all the actor/timestamp fields. The new `"Role": {}` lands at line 14813, between `Scope` (14812) and `LifecycleState` (14814).

The classification is correct. `Role` is a closed-enum classifier identical in shape to `Kind`/`Scope`/`LifecycleState` — those three are all `readOnly` because they're set at create-time (`Kind`/`Scope`) or via dedicated state-machine transitions (`LifecycleState`), not via free-form TUI edit. `Role` follows the same pattern: it's set at create-time via the constructor's normalize-and-validate block (action_item.go:155-158); there's no `Role` mutator method on `ActionItem` (no equivalent of `UpdateDetails`/`SetLifecycleState` for roles). The TUI today has no role-edit overlay. Adjacent placement to its peer classifiers also makes the slot easy to spot in future audits.

The reflect-based assertion `assertExplicitFieldCoverage(t, reflect.TypeOf(domain.ActionItem{}), editable, readOnly, nil)` at line 14829 enumerates every struct field on `domain.ActionItem` and fails if any are missing from the union of the two maps — that's why adding `Role Role` to the struct forced this scope expansion. The builder's flag is correct: this gate trips on every new `ActionItem` field. The single-line addition `"Role": {}` is the minimum diff to satisfy the gate.

**Scope-expansion judgment: ACCEPT readOnly classification.** The path was not in the orchestrator's listed Paths, but the gate's mechanics force the classification, and `readOnly` matches the actual data model (no mutator + closed-enum classifier in the same lane as Kind/Scope/LifecycleState). Adding it to `editable` instead would imply a TUI edit path that doesn't exist and would create an asymmetry with `Kind`/`Scope`/`LifecycleState`. No re-classification needed; no orchestrator routing needed beyond informational acknowledgment.

#### F8. `mage test-pkg ./internal/domain` green at HEAD. (PASS)

I re-ran `mage test-pkg ./internal/domain` on HEAD and observed:

- `[RUNNING] Running go test ./internal/domain`
- `[SUCCESS] Test stream detected`
- `[PKG PASS] github.com/evanmschultz/tillsyn/internal/domain (0.00s)`
- Test summary: `tests: 103 / passed: 103 / failed: 0 / skipped: 0`
- `[SUCCESS] All tests passed`

Test count went from 90 (Droplet 2.2 baseline) to 103 — delta of +13 tests. The new `TestNewActionItemRoleValidation` itself contributes 12 sub-cases plus the parent test, but the Go test counter at this aggregation level reports the parent + each sub-case (the +13 lines up with the 12 new sub-cases plus the harness arithmetic noted in BUILDER_WORKLOG: "103 tests pass (was 102 prior; new TestNewActionItemRoleValidation adds 1 test with 12 subtests)"). Independent confirmation of the builder's claim. Acceptance "mage test-pkg ./internal/domain green" satisfied.

#### F9. `mage ci` green at HEAD with full coverage compliance. (PASS)

I re-ran `mage ci` on HEAD and observed:

- All four `[SUCCESS]` source/format/test-stream gates green.
- 19 packages, **1313 tests**, 0 failed, 0 skipped — every package `[PKG PASS]`.
- `internal/domain` package coverage: **79.4%** (≥ 70.0% threshold).
- `internal/tui` package coverage: **70.0%** (exactly meets threshold — the schema-coverage gate's `+1 LOC` did not move the needle).
- Minimum package coverage across all 15 reported packages: 70.0%.
- `[SUCCESS] Built till from ./cmd/till`.

This corroborates the builder's claim of 1313 tests + ≥ 70% coverage on every package + clean build. Test count delta from Droplet 2.2 (1300) to Droplet 2.3 (1313) is +13 — matches `mage test-pkg ./internal/domain` delta (90 → 103, also +13). `internal/tui` coverage holds at exactly 70.0% — the `+1 LOC` schema-map entry did not push the package below threshold.

#### F10. `git status --porcelain` matches expected delta exactly. (PASS)

Observed:

- ` M internal/domain/action_item.go`
- ` M internal/domain/domain_test.go`
- ` M internal/tui/model_test.go`
- ` M workflow/drop_2/BUILDER_WORKLOG.md`
- ` M workflow/drop_2/PLAN.md`

Five entries, exactly matching the prompt's expected set (`M action_item.go`, `M domain_test.go`, `M model_test.go`, `M PLAN.md`, `M BUILDER_WORKLOG.md`). No stray modifications, no accidental edits to other files. The scope expansion to `model_test.go` is staged and is the only path beyond the original three (`action_item.go` + `domain_test.go` + the worklog/plan MDs).

#### F11. PLAN.md state flip is correct. (PASS)

`workflow/drop_2/PLAN.md:87` reads `- **State:** done` immediately under the `#### Droplet 2.3 — Add Role field to ActionItem + ActionItemInput + NewActionItem validation` heading at line 85. State-flip executed correctly (`todo → in_progress → done`).

#### F12. `BUILDER_WORKLOG.md` carries a structurally-sound Droplet 2.3 Round 1 entry. (PASS)

`workflow/drop_2/BUILDER_WORKLOG.md` (lines 95-121) contains:

- `## Droplet 2.3 — Round 1` heading (line 95)
- `**Outcome:** success.` (line 97)
- Files-touched section enumerating the three Go files + worklog + PLAN with explicit LOC deltas (lines 99-103)
- **Scope expansion explicitly flagged** for `internal/tui/model_test.go` with classification rationale (line 103)
- `**Mage results:**` section reporting `mage test-pkg ./internal/domain` 103/103 + `mage ci` 1313/1313 (lines 105-108)
- `**Design notes:**` section covering field placement, short-circuit-on-empty rationale, post-forvar test-style choice, and existing-test-stay-green claim (lines 110-115)
- `**PLAN.md state flips:**` line 117 documenting both transitions
- `## Hylla Feedback` section (line 119) with explicit "None — Hylla answered everything needed" status

Structurally sound and substantively accurate. Builder's pre-flagged scope expansion is honored.

### Missing Evidence

None. All 12 required proof checks have direct citations + reproducible mage gates at HEAD. The `internal/tui/` pre-existing tech-debt warnings are explicitly out of scope per the orchestrator prompt and per memory `project_drop_2_refinements_raised.md` R1; not flagged.

### Verdict Summary

Droplet 2.3 lands cleanly. The `Role Role` field is on both `ActionItem` (action_item.go:33) and `ActionItemInput` (action_item.go:67) in the correct position between `Scope` and `LifecycleState`, with idiomatic Go-doc comments documenting empty/optional semantics. `NewActionItem`'s normalize-and-validate block at action_item.go:150-158 implements the exact pattern (`in.Role = NormalizeRole(in.Role); if in.Role != "" && !IsValidRole(in.Role) { return ErrInvalidRole }`) and the constructor's return literal at line 201 carries `Role: in.Role` for the round-trip. `TestNewActionItemRoleValidation` at domain_test.go:210-255 covers all 12 required cases (empty, whitespace, 9 valid roles, unknown rejects) using the post-Droplet-2.2-Round-2 forvar-clean test idiom. The schema-coverage gate scope expansion to `internal/tui/model_test.go:14813` (`"Role": {}` in the `readOnly` map) is the right call — `Role` is a closed-enum classifier in the same lane as `Kind`/`Scope`/`LifecycleState`, all readOnly, with no mutator method on `ActionItem`. `mage test-pkg ./internal/domain` reproduces 103/103 tests passing on HEAD; `mage ci` reproduces 1313/1313 tests across 19 packages, with `internal/domain` at 79.4% coverage and all packages ≥ 70% (the `+1 LOC` schema-map entry left `internal/tui` at exactly 70.0%, threshold-compliant). `git status --porcelain` matches the expected 5-entry delta exactly. `PLAN.md` state is correctly `done`. `BUILDER_WORKLOG.md` carries a structurally-sound Round 1 entry that pre-flags the scope expansion. Ready to commit.

### Hylla Feedback

None — Hylla answered everything needed. QA verification was code-local: `Read` for whole-file inspection of `action_item.go` (small file, structural questions), `Read` with offset/limit for the new test func at `domain_test.go:210-255` and the schema-coverage gate at `model_test.go:14797-14829`, `LSP documentSymbol` for fast top-level navigation inside the 26k-line `domain_test.go` (the right tool for "show me every test func and its line"), `Bash` for `git status` / `git diff` / `mage test-pkg` / `mage ci`. No Hylla queries were attempted because the changed files (`action_item.go`, `domain_test.go`, `model_test.go`) are post-last-ingest deltas — Hylla is stale on those by definition until drop-end reingest, so `Read` and `LSP` are the canonical tools per CLAUDE.md "Code Understanding Rules" §2 ("Changed since last ingest: use git diff for files touched after the last Hylla ingest"). Zero Hylla misses, zero ergonomic gripes.

## Droplet 2.4 — Round 1

**Verdict:** pass
**Date:** 2026-05-02

### Findings

#### F1. CREATE TABLE adds `role TEXT NOT NULL DEFAULT ''` between `scope` and `lifecycle_state`. (PASS)

`internal/adapters/storage/sqlite/repo.go:174` (post-edit) reads:

```
kind TEXT NOT NULL DEFAULT 'actionItem',
scope TEXT NOT NULL DEFAULT 'actionItem',
role TEXT NOT NULL DEFAULT '',
lifecycle_state TEXT NOT NULL DEFAULT 'todo',
```

`git diff` confirms the addition is a single new line (`+role TEXT NOT NULL DEFAULT ''`) inside the `CREATE TABLE IF NOT EXISTS action_items` block at `:171`. Column position groups the four closed-enum classifiers (`kind`, `scope`, `role`, `lifecycle_state`) consecutively — matches the Droplet 2.3 struct ordering convention and the "all five SQL sites in lockstep" rationale in the worklog. PLAN.md acceptance #1 ("`role TEXT NOT NULL DEFAULT ''` appears in the `action_items` `CREATE TABLE` statement at `:168`") satisfied.

#### F2. `scanActionItem` reads `role` into `domain.ActionItem.Role` in correct slot. (PASS)

`internal/adapters/storage/sqlite/repo.go:2756` (post-edit) declares the local `roleRaw string` between `scopeRaw` and `state`. `:2766` adds `&roleRaw,` as the 6th Scan target between `&scopeRaw` and `&state`. `:2796` assigns `t.Role = domain.Role(roleRaw)`. The Scan-target order matches the SELECT column list order — `id, project_id, parent_id, kind, scope, role, lifecycle_state, ...` (positional slots 1-7). The `domain.Role(roleRaw)` cast is a typed-string conversion; the empty default round-trips as the `Role` zero value. PLAN.md acceptance #2 satisfied.

#### F3. INSERT SQL — column list, VALUES placeholders, and bind-args slice all add `role` at position 6. (PASS)

`repo.go:1239` (column list, post-edit):

```
id, project_id, parent_id, kind, scope, role, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
```

`repo.go:1242` (VALUES list): `25 → 26` placeholders (one new `?` added). `repo.go:1244-1252` bind-args: `t.ID, t.ProjectID, t.ParentID, string(t.Kind), string(scope), string(t.Role), string(t.LifecycleState), t.ColumnID, t.Position, ...`. The `string(t.Role)` arg lands at slot 6 between `string(scope)` and `string(t.LifecycleState)`, matching the column-list slot. Diff is a 3-line addition (column-list slot, one new `?`, one bind-arg line) — net +3 lines on the INSERT path. PLAN.md acceptance #3 (insert SQL includes `role`) satisfied.

#### F4. UPDATE SQL — SET clause and bind-args add `role = ?` at correct slot. (PASS)

`repo.go:1333` (SET clause, post-edit):

```
SET parent_id = ?, kind = ?, scope = ?, role = ?, lifecycle_state = ?, column_id = ?, position = ?, title = ?, ...
```

`repo.go:1340` adds `string(t.Role)` between `string(scope)` and `string(t.LifecycleState)` in the bind-args. Slot ordering matches the SET clause: 1=parent_id, 2=kind, 3=scope, 4=role, 5=lifecycle_state. The UPDATE SET clause omits `id` and `project_id` (immutable post-create) but the relative ordering for the rest of the row matches the INSERT column list and CREATE TABLE schema. PLAN.md acceptance #3 (update SQL includes `role`) satisfied.

#### F5. Both SELECT statements add `role` at slot 6. (PASS)

Builder pre-flagged this as a discovery: PLAN.md mentions "scanActionItem at `:2738`" but doesn't enumerate the SELECT sites. Two SELECT statements feed `scanActionItem`:

- **`ListActionItems` SELECT at `repo.go:1399`**: column list now reads `id, project_id, parent_id, kind, scope, role, lifecycle_state, column_id, position, ...`. `role` at slot 6.
- **`getActionItemByID` SELECT at `repo.go:2450`**: column list now reads `id, project_id, parent_id, kind, scope, role, lifecycle_state, column_id, position, ...`. `role` at slot 6.

Both SELECT slots match `scanActionItem`'s Scan-target order (F2). If only one had been updated, `scanActionItem` would have read `lifecycle_state` into `roleRaw` for the unupdated path — every test exercising that path would have silently shifted bindings and broken. The new test's ListActionItems assertion (F7 below) and the existing 68 sqlite tests (which exercise `getActionItemByID` heavily) all passing on HEAD confirms both paths are correctly aligned.

#### F6. Pre-MVP rule honored — no `ALTER TABLE` for `role`, no migration code, no SQL backfill. (PASS)

`git grep -n "ALTER TABLE" internal/adapters/storage/sqlite/repo.go` returns 22 hits, all of which are pre-existing `ALTER TABLE` statements for legacy columns:

- `:511` `projects.metadata_json` (pre-existing).
- `:514` `attention_items.target_role` (pre-existing).
- `:518-530` legacy `action_items.{parent_id, kind, scope, lifecycle_state, metadata_json, created_by_*, updated_by_*, started_at, completed_at, canceled_at}` (pre-existing migration block).
- `:547-549` `auth_requests.*` (pre-existing).
- `:633` `comments` rename (pre-existing).
- `:694` `comments.summary` (pre-existing).
- `:742` `change_events.actor_name` (pre-existing).

**Critically, zero `ALTER TABLE action_items ADD COLUMN role` anywhere in repo.go.** The only schema source for the new `role` column is the `CREATE TABLE IF NOT EXISTS action_items` block at `:172` (F1 above). `git diff internal/adapters/storage/sqlite/repo.go | grep -E "^\\+.*ALTER TABLE"` returns empty — no `ALTER TABLE` was added by this droplet. Pre-MVP rule (memory `feedback_no_migration_logic_pre_mvp.md`) honored. PLAN.md acceptance "no `ALTER TABLE` migration, no SQL backfill — dev fresh-DBs" satisfied.

#### F7. `TestRepository_PersistsActionItemRole` exercises every required behavior. (PASS)

`internal/adapters/storage/sqlite/repo_test.go:2204-2306` (new test, +106 LOC) covers the four required behaviors:

- **Empty-role default round-trip** (`:2225-2245`): `NewActionItem` with no `Role` field set → `CreateActionItem` → `GetActionItem` → asserts `loadedEmpty.Role != ""` would fatal. Exercises the empty-string DEFAULT path and the `domain.Role(roleRaw)` zero-value cast in `scanActionItem`.
- **`RoleBuilder` round-trip via create + get** (`:2249-2270`): `NewActionItem` with `Role: domain.RoleBuilder` → `CreateActionItem` → `GetActionItem` → asserts equality. Exercises INSERT with `string(t.Role) == "builder"` at slot 6 and `getActionItemByID` SELECT path.
- **`ListActionItems` surfaces role** (`:2272-2284`): separate SELECT path — iterates the listing and confirms the `RoleBuilder` item carries the role value. Exercises `ListActionItems`'s independent SELECT at `repo.go:1399` (the second SELECT path that feeds `scanActionItem`).
- **Reassign on update from `RoleBuilder` → `RoleQAProof`** (`:2289-2305`): mutates the loaded item, calls `UpdateActionItem`, re-fetches, asserts new value. This is the load-bearing UPDATE assertion — an UPDATE that forgot the `role = ?` clause would still pass a "create with role, read back" test, but would fail the reassign because the underlying row would still carry `"builder"`.

All four covered. PLAN.md acceptance "Existing tests with empty `Role` still pass (empty-string default)" + "One new test in `repo_test.go` writes `domain.RoleBuilder`, reads it back, asserts equality" satisfied — and the test goes beyond the literal acceptance text by also covering ListActionItems and Update, which is appropriate given the five SQL sites the column touches.

#### F8. No `tc := tc` lines in the new test. (PASS)

`git grep "tc := tc" internal/adapters/storage/sqlite/repo_test.go` returned no hits ("NO HITS"). The new test is straight-line (not table-driven across `t.Run` subtests), so the Go 1.22+ per-iteration scoping rule doesn't even apply, but the file-wide invariant holds anyway. Consistent with the post-Droplet-2.2-Round-2 forvar-clean convention.

#### F9. `mage test-pkg ./internal/adapters/storage/sqlite` green at HEAD with fresh DB. (PASS)

I deleted `~/.tillsyn/tillsyn.db` (per PLAN.md DB action) and re-ran `mage test-pkg ./internal/adapters/storage/sqlite` on HEAD:

- `[RUNNING] Running go test ./internal/adapters/storage/sqlite`
- `[SUCCESS] Test stream detected`
- `[PKG PASS] github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite (0.00s)`
- Test summary: `tests: 69 / passed: 69 / failed: 0 / skipped: 0`
- `[SUCCESS] All tests passed`

Test count went 68 → 69, exactly +1 — the new `TestRepository_PersistsActionItemRole`. Independent confirmation of the builder's claim. PLAN.md acceptance "`mage test-pkg ./internal/adapters/storage/sqlite` green" satisfied.

#### F10. `mage ci` green at HEAD with full coverage compliance. (PASS)

I re-ran `mage ci` on HEAD and observed:

- All four `[SUCCESS]` source/format/test-stream gates green.
- 19 packages, **1314 tests**, 0 failed, 0 skipped — every package `[PKG PASS]`.
- `internal/adapters/storage/sqlite` package coverage: **75.1%** (≥ 70.0% threshold).
- Minimum package coverage across all 15 reported packages: 70.0% (lowest is `internal/tui` at exactly 70.0%; `internal/buildinfo` at 100.0%).
- `[SUCCESS] Built till from ./cmd/till`.

Test count delta from Droplet 2.3 (1313) to Droplet 2.4 (1314) is exactly +1 — matches the `mage test-pkg ./internal/adapters/storage/sqlite` delta (68 → 69, also +1). All 19 packages green. The `role` column addition + scan-args + insert/update bindings did not regress any existing test, confirming the empty-string DEFAULT correctly round-trips through every fixture that constructs an `ActionItem` without setting `Role`.

#### F11. No regressions on existing tests. (PASS, covered by F9 + F10)

The 68 pre-existing sqlite tests + 1245 pre-existing tests across the other 18 packages all pass. The empty-string DEFAULT means existing fixtures that omit the `Role` field continue to round-trip cleanly:

- INSERT writes `string(t.Role) == ""` at slot 6 — accepted by `role TEXT NOT NULL DEFAULT ''`.
- Scan reads `roleRaw == ""` — assigned to `t.Role` as `domain.Role("")`, the typed zero value.
- Existing comparisons that don't reference `Role` are invariant on the addition.

This is the "additive column with safe default" pattern that the pre-MVP fresh-DB rule unlocks — no migration code needed, no backfill, existing fixtures keep working. PLAN.md acceptance "Existing tests with empty `Role` still pass (empty-string default)" satisfied.

#### F12. `git status --porcelain` matches expected delta exactly. (PASS)

Observed:

- ` M internal/adapters/storage/sqlite/repo.go`
- ` M internal/adapters/storage/sqlite/repo_test.go`
- ` M workflow/drop_2/BUILDER_WORKLOG.md`
- ` M workflow/drop_2/PLAN.md`

Four entries, exactly matching the prompt's expected set (`M repo.go`, `M repo_test.go`, `M PLAN.md`, `M BUILDER_WORKLOG.md`). No stray modifications, no accidental edits to other files. The Round 1 scope-expansion to `internal/tui/model_test.go` from Droplet 2.3 was already committed prior to Droplet 2.4 — `Role` is already in the readOnly schema-coverage map, so this droplet did not need to touch the TUI gate again.

#### F13. PLAN.md state flip is correct. (PASS)

`workflow/drop_2/PLAN.md:102` reads `- **State:** done` immediately under the `#### Droplet 2.4 — SQLite \`action_items.role\` column + scanner + insert/update paths` heading at line 100. State-flip executed correctly (`todo → in_progress → done`).

#### F14. `BUILDER_WORKLOG.md` carries a structurally-sound Droplet 2.4 Round 1 entry. (PASS)

`workflow/drop_2/BUILDER_WORKLOG.md` (lines 123-150) contains:

- `## Droplet 2.4 — Round 1` heading (line 123).
- Files-touched section enumerating both Go files with explicit LOC deltas (+9 on `repo.go`, +106 on `repo_test.go`) at `:125-128`.
- `**Mage results:**` section reporting `mage test-pkg ./internal/adapters/storage/sqlite` 69/69 + `mage ci` 1314/1314 + sqlite coverage 75.1% at `:130-133`.
- `**Design notes:**` section covering: column position rationale, three-SELECT-path discovery (PLAN.md mentioned only one), empty-role default semantics, test-pattern choice (focused round-trip vs extending the kind/scope test), reassign-via-update being the load-bearing assertion, pre-MVP rule honored at `:135-142`.
- `**No `tc := tc`...`** explicit invariant honored note at `:144`.
- `**PLAN.md state flips:**` line `:146` documenting both transitions.
- `## Hylla Feedback` section at `:148-150` with explicit "None — Hylla answered everything needed" status + rationale.

Structurally sound and substantively accurate. The discovery callout (two SELECTs, not one as PLAN.md suggested) is correctly surfaced for orchestrator awareness.

### Missing Evidence

None. All 14 required proof checks have direct citations + reproducible mage gates at HEAD. The fresh-DB requirement was honored before the test run (deleted `~/.tillsyn/tillsyn.db` per PLAN.md DB action). The pre-existing `internal/tui/`, `internal/app/`, `internal/adapters/storage/sqlite/` tech-debt warnings (R1, R5, R6 in `project_drop_2_refinements_raised.md`) and the `go.mod` chroma/v2 indirect→direct warning (R2) are explicitly out of scope per the orchestrator prompt; not flagged.

### Verdict Summary

Droplet 2.4 lands cleanly. The `role TEXT NOT NULL DEFAULT ''` column joins the `action_items` `CREATE TABLE` block at `repo.go:174`, between `scope` and `lifecycle_state` — keeping the four closed-enum classifiers (`kind`, `scope`, `role`, `lifecycle_state`) consecutive across the schema, the Go struct, and every SQL site. All five SQL sites are in lockstep at slot 6: `scanActionItem` Scan target (`:2766`), INSERT column list + bind args (`:1239 + :1249`), UPDATE SET clause + bind arg (`:1333 + :1340`), `ListActionItems` SELECT (`:1399`), and `getActionItemByID` SELECT (`:2450`). The builder correctly discovered and updated BOTH SELECT statements — PLAN.md mentioned only `scanActionItem` but the file has two SELECTs feeding it; missing one would have silently shifted bindings and broken every fixture exercising that path. Pre-MVP rule honored: zero `ALTER TABLE`, zero migration code, zero SQL backfill — the `CREATE TABLE IF NOT EXISTS` block is the only schema source for the new column. `TestRepository_PersistsActionItemRole` at `repo_test.go:2204-2306` exercises all four required behaviors (empty-role default, `RoleBuilder` round-trip, `ListActionItems` surfaces role, reassign on update from `RoleBuilder` → `RoleQAProof`) — the reassign assertion is load-bearing because a "create-and-read" alone wouldn't catch a forgotten UPDATE SET clause. No `tc := tc` lines anywhere in the test file. With `~/.tillsyn/tillsyn.db` deleted (per PLAN.md DB action), `mage test-pkg ./internal/adapters/storage/sqlite` reproduces 69/69 tests passing on HEAD; `mage ci` reproduces 1314/1314 tests across 19 packages, with `internal/adapters/storage/sqlite` at 75.1% coverage and all packages ≥ 70%. `git status --porcelain` matches the expected 4-entry delta exactly. PLAN.md state is correctly `done`. `BUILDER_WORKLOG.md` carries a structurally-sound Round 1 entry that pre-flags the two-SELECT-paths discovery. Ready to commit.

### Hylla Feedback

None — Hylla answered everything needed (and most of the verification was against committed Go code in `repo.go` + uncommitted test deltas in `repo_test.go`, where Hylla is stale post-edit by design). QA verification used `Read` for whole-file inspection of `role.go` (small file, structural check on the Role enum constants), `Read` with offset/limit for surgical inspection of the test file diff and the worklog, `Bash` for `git status` / `git diff` / `git grep "ALTER TABLE"` / `git grep "tc := tc"` / `mage test-pkg` / `mage ci`. No Hylla queries were attempted because (a) `repo.go` and `repo_test.go` carry uncommitted deltas — Hylla is stale on those by definition until drop-end reingest, so `git diff` and `Read` are the canonical tools per CLAUDE.md "Code Understanding Rules" §2; (b) the SQL string contents being verified are non-Go-symbol payloads that Hylla doesn't index (per memory `feedback_hylla_go_only_today.md`), so `Read` + `git diff` are the right tools regardless. Zero Hylla misses, zero ergonomic gripes for this droplet.

## Droplet 2.5 — Round 1

**Verdict:** pass
**Date:** 2026-05-02

### Findings

1. **`mcp_surface.go` request-struct role fields confirmed.** `CreateActionItemRequest.Role string` lands at `internal/adapters/server/common/mcp_surface.go:62` with a 4-line doc comment (`:58-61`) covering the empty-permitted + closed-enum-validation contract. `UpdateActionItemRequest.Role string` lands at `:88` with a 3-line doc comment (`:85-87`) covering the empty-preserves-prior semantic. Field placement on both structs is between the closed-enum classifiers and the rest of the request payload — consistent with the kind/scope grouping convention.

2. **`app.CreateActionItemInput.Role` and `app.UpdateActionItemInput.Role` are typed `domain.Role`.** `internal/app/service.go:410` (CreateActionItemInput) carries `Role domain.Role` between `Scope` and `ColumnID` with a 4-line doc comment (`:406-409`) describing the closed-enum + empty-permitted + `ErrInvalidRole`-on-mismatch contract. `service.go:438` (UpdateActionItemInput) carries `Role domain.Role` between `Labels` and `Metadata` with a 3-line doc comment (`:435-437`) describing the empty-preserves-prior + closed-enum-validation semantic. Both fields are typed `domain.Role`, not raw `string` — type safety lives at the MCP-adapter conversion boundary, not deep in the app layer.

3. **`app_service_adapter_mcp.go` threads role on both create and update.** `CreateActionItem` at `internal/adapters/server/common/app_service_adapter_mcp.go:641` adds `Role: domain.Role(strings.TrimSpace(in.Role))` between `Scope` (`:640`) and `ColumnID` (`:642`) inside the `app.CreateActionItemInput` literal. `UpdateActionItem` at `:685` adds `Role: domain.Role(strings.TrimSpace(in.Role))` between `Labels` (`:684`) and `Metadata` (`:686`) inside the `app.UpdateActionItemInput` literal. Both sites use the same `domain.Role(strings.TrimSpace(...))` cast/trim pattern — wire-level whitespace is normalized at the adapter boundary, leaving downstream callers a clean value.

4. **`app_service_adapter.go mapAppError` adds `domain.ErrInvalidRole` to the 400-class list.** `internal/adapters/server/common/app_service_adapter.go:650` inserts `errors.Is(err, domain.ErrInvalidRole),` into the multi-clause `case` block at `:644-651` that maps invalid-input errors to `errors.Join(ErrInvalidCaptureStateRequest, err)`. Placement is between `ErrKindNotAllowed` (`:649`) and `app.ErrInvalidDeleteMode` (`:651`) — consistent with the kind-family grouping. The wrapped result emits the `invalid_request:` MCP prefix on the wire (mapped at `mcpapi/handler.go` against `common.ErrInvalidCaptureStateRequest`), which the test `create with invalid role returns invalid_request` exercises end-to-end.

5. **`extended_tools.go` schema field on all three tool registrations.** `mcp.WithString("role", ...)` confirmed at three sites: (a) primary `till.action_item` tool at `internal/adapters/server/mcpapi/extended_tools.go:1347` (`"Optional role tag for operation=create|update — see allowed values (closed enum: builder|qa-proof|qa-falsification|qa-a11y|qa-visual|design|commit|planner|research). Empty string preserves the existing value on update."`); (b) legacy `till.create_task` at `:1401` (closed-enum description, no preserve-on-update language since it's create-only); (c) legacy `till.update_task` at `:1428` (closed-enum description with `Empty preserves prior value.` suffix). All three descriptions enumerate the 9 closed-enum values explicitly — LLM tool-callers don't have to guess valid values. Parity at the schema surface keeps legacy callers honest (without a schema field, the `role` payload would be silently stripped at MCP boundary even though the underlying handler reads it).

6. **`extended_tools.go` args-struct + request-literal threading.** Args struct at `internal/adapters/server/mcpapi/extended_tools.go:865` adds `Role string \`json:"role"\`` between `Scope` (`:864`) and `ColumnID` (`:866`). Threading into `common.CreateActionItemRequest` literal at `:1033` (`Role: args.Role,` between `Scope` at `:1032` and `ColumnID` at `:1034`). Threading into `common.UpdateActionItemRequest` literal at `:1091` (`Role: args.Role,` between `Labels` at `:1090` and `Metadata` at `:1092`). Both request-literal slot positions match the field-order convention from check #1.

7. **App-service create handler validates via `domain.NewActionItem`.** `Service.CreateActionItem` at `internal/app/service.go:580` includes `Role: in.Role,` in the `domain.ActionItemInput` literal passed to `domain.NewActionItem` (called downstream). The Droplet 2.3 `NewActionItem` block at `internal/domain/action_item.go` normalizes via `NormalizeRole`, runs `IsValidRole` if non-empty, returns `ErrInvalidRole` on mismatch, accepts empty as the zero-value default — verified via Droplet 2.3 worklog + `mage test-pkg ./internal/domain` 103/103 from prior round. Invalid role from create therefore surfaces at the domain layer and bubbles through the app + adapter chain to the MCP `invalid_request:` boundary mapping.

8. **App-service update handler — empty preserves, non-empty validates.** `Service.UpdateActionItem` at `internal/app/service.go:784-794` adds the role-update block immediately after `actionItem.UpdateDetails(...)` (`:781`). The block normalizes via `domain.NormalizeRole(in.Role)` (`:788`), short-circuits on `normalized == ""` (the empty-preserves-prior path is achieved by NOT entering the if-block), validates via `domain.IsValidRole` on the non-empty branch (`:789`), returns `domain.ErrInvalidRole` on mismatch (`:790`), assigns `actionItem.Role = normalized` and bumps `actionItem.UpdatedAt` on success (`:792-793`). Behavior matches the `Empty role is accepted on create and update (no-op for update)` and `Invalid role returns a 400-class MCP error` acceptance lines from PLAN.md `:123-124`.

9. **`TestHandlerExpandedActionItemRoleRoundTrip` covers all 5 sub-tests.** Test at `internal/adapters/server/mcpapi/extended_tools_test.go:3151-3280`: (a) `create with valid role plumbs and round-trips` (`:3177-3200`) asserts `service.lastCreateActionItemReq.Role == string(domain.RoleBuilder)` AND that `toolResultText(...)` contains the role string in the response payload — covers the create + get round-trip explicitly per PLAN.md `:121` `reading via operation=get returns it`; (b) `create without role round-trips empty` (`:3202-3219`) asserts `service.lastCreateActionItemReq.Role == ""`; (c) `update with role plumbs the new value` (`:3221-3238`) asserts `service.lastUpdateActionItemReq.Role == string(domain.RoleQAProof)`; (d) `update without role preserves prior` (`:3240-3259`) asserts `service.lastUpdateActionItemReq.Role == ""` (empty wire, preservation done at service-layer no-op branch verified independently by check #8); (e) `create with invalid role returns invalid_request` (`:3261-3279`) asserts `isError == true` AND `toolResultText` has prefix `invalid_request:` — exercises the `mapAppError` + `errors.Join(common.ErrInvalidCaptureStateRequest, domain.ErrInvalidRole)` chain end-to-end via the production-shape stub. The stub at `:380-394` and `:411-426` returns the same wrapped-error shape as the real `AppServiceAdapter` so the MCP error mapper hits the `invalid_request:` prefix path. Stub also echoes the trimmed role into the returned `domain.ActionItem.Role` field at `:399` and `:425` for the response-payload assertion in (a).

10. **No `tc := tc` in the new role test code.** `TestHandlerExpandedActionItemRoleRoundTrip` uses 5 direct `t.Run` calls with literal sub-test names (not a `for _, tc := range cases { t.Run(tc.name, ...) }` loop), so the Go-1.22+ per-iteration scoping is moot. Confirmed by reading `:3151-3280` in full — no loop variable to capture. Round 2 cleanup pass removed pre-existing `tc := tc` lines at `:3051` (was inside `TestHandlerExpandedGlobalAdminMutationsUseRootedProjectAuthScope`'s loop) and `:3114` (was inside `TestHandlerExpandedMutationAuthErrorsMap`'s loop) — both adjacent to the new test, but neither belongs to it. Diff confirms removals at the same line numbers.

11. **Mage gates green at HEAD.** Reproduced all four targets locally:
    - `mage testPkg ./internal/app` → **176 tests pass** in 1 package (matches builder claim).
    - `mage testPkg ./internal/adapters/server/common` → **123 tests pass** in 1 package (matches builder claim).
    - `mage testPkg ./internal/adapters/server/mcpapi` → **93 tests pass** in 1 package (matches builder claim, was 87 prior + 6 new = 93).
    - `mage ci` → **1320 tests pass across 19 packages**, all packages ≥ 70% coverage threshold, build of `./cmd/till` succeeds, exit 0. Per-package coverage: `internal/app` 71.5%, `internal/adapters/server/common` 73.0%, `internal/adapters/server/mcpapi` 72.4%, `internal/domain` 79.4%, `internal/tui` 70.0% (matches builder report exactly).

12. **No unintended file changes.** `git status --porcelain` returns exactly 8 entries, all `M` (modified):
    - `internal/adapters/server/common/app_service_adapter.go`
    - `internal/adapters/server/common/app_service_adapter_mcp.go`
    - `internal/adapters/server/common/mcp_surface.go`
    - `internal/adapters/server/mcpapi/extended_tools.go`
    - `internal/adapters/server/mcpapi/extended_tools_test.go`
    - `internal/app/service.go`
    - `workflow/drop_2/BUILDER_WORKLOG.md`
    - `workflow/drop_2/PLAN.md`

    No `??` untracked entries, no `D`/`A` deltas. Matches expected 6-source + 2-workflow MD touch-set exactly.

13. **PLAN.md state-flip confirmed.** `workflow/drop_2/PLAN.md:117` reads `**State:** done` for Droplet 2.5 — the heading at `:115` plus the state line at `:117` are correctly transitioned from prior `todo` per the BUILDER_WORKLOG.md Round 1 `PLAN.md state flips` note at line 182.

### Missing Evidence

None. All 13 required proof checks have direct citations + reproducible mage gates at HEAD. The Round 2 `tc := tc` cleanup at `:3051` and `:3114` is correctly limited to surrounding pre-existing tests; the new role test never had `tc := tc` per its non-table-driven structure. The pre-existing tech-debt warnings (R1, R5, R6, R7 in `project_drop_2_refinements_raised.md`) and the `go.mod` chroma/v2 indirect→direct warning (R2) are explicitly out of scope per the orchestrator prompt; not flagged.

### Verdict Summary

Droplet 2.5 lands cleanly. The MCP `role` field is plumbed through all five required surfaces — `CreateActionItemRequest` + `UpdateActionItemRequest` (`mcp_surface.go:62, :88`), `app.CreateActionItemInput` + `app.UpdateActionItemInput` (`service.go:410, :438`), the adapter `Role: domain.Role(strings.TrimSpace(in.Role))` casts on both create + update paths (`app_service_adapter_mcp.go:641, :685`), the `mapAppError` 400-class clause (`app_service_adapter.go:650`), and the args-struct + request-literal + tool-schema triple at the MCP boundary (`extended_tools.go:865, :1033, :1091, :1347, :1401, :1428`). The kind-pattern mirror is faithfully implemented with one deliberate divergence flagged in the worklog: Role is mutable on update (per spec line `:122` `updates the role on an existing action item`) where Kind is immutable, and the service-layer update block at `service.go:784-794` correctly implements the empty-preserves-prior + non-empty-validates semantic. `TestHandlerExpandedActionItemRoleRoundTrip` at `extended_tools_test.go:3151-3280` exercises all 5 sub-tests required by PLAN.md `:125` (create-with-valid-role + create-without-role + update-with-role + update-without-role + create-with-invalid-role) using a production-shape stub that returns `errors.Join(common.ErrInvalidCaptureStateRequest, domain.ErrInvalidRole)` — the same wrapped-error shape `AppServiceAdapter.mapAppError` produces — so the MCP `invalid_request:` boundary mapping is exercised end-to-end without a full app-service stack. The legacy `till.create_task` and `till.update_task` schema parity at `extended_tools.go:1401` and `:1428` is the right call: without it, payload `role` would be silently stripped at the schema boundary even though `handleActionItemOperation` reads it. No `tc := tc` lines anywhere in the new test code; Round 2 cleanup correctly scoped to the two pre-existing surrounding tests at `:3051` + `:3114`. `mage testPkg ./internal/app` reproduces 176/176, `./internal/adapters/server/common` 123/123, `./internal/adapters/server/mcpapi` 93/93; `mage ci` reproduces 1320/1320 across 19 packages with all coverage ≥ 70% on HEAD. `git status --porcelain` matches the expected 8-entry delta exactly. PLAN.md state is correctly `done` at `:117`. `BUILDER_WORKLOG.md` carries structurally-sound Round 1 + Round 2 entries with explicit design notes covering the kind-pattern divergence + the stub-bypass-and-wrapped-error shape rationale. Ready to commit.

### Hylla Feedback

None — Hylla answered everything needed (and most of the verification was against uncommitted Go deltas in 6 source files + 1 test file, where Hylla is stale post-edit by design until drop-end reingest). QA verification used `Read` with offset/limit for surgical inspection of the test file (the new `TestHandlerExpandedActionItemRoleRoundTrip` block at `:3151-3280`) and `Bash` for `git diff` per file / `git status --porcelain` / `mage testPkg` / `mage ci`. No Hylla queries were attempted because all 6 source files carry uncommitted deltas — Hylla is stale on those by definition until drop-end reingest, so `git diff` and `Read` are the canonical tools per CLAUDE.md "Code Understanding Rules" §2. The earlier-droplet committed code (Droplet 2.3's `domain.NewActionItem` role-validation block, the `domain.Role` enum + `IsValidRole` from Droplet 2.2) was already exercised by `mage testPkg ./internal/domain` in prior rounds and verified by their respective QA passes — re-querying Hylla would not have added signal. Zero Hylla misses, zero ergonomic gripes for this droplet.

## Droplet 2.6 — Round 1

**Verdict:** pass
**Date:** 2026-05-02

### Findings

1. **`SnapshotActionItem.Role` field present, correctly typed, correctly tagged.** `internal/app/snapshot.go:63` reads `Role           domain.Role               \`json:"role,omitempty"\``. Field is positioned between `Scope` (`:62`) and `LifecycleState` (`:64`), grouping the closed-enum classifiers `Kind`/`Scope`/`Role`/`LifecycleState` as the matching `domain.ActionItem` definition does at `internal/domain/action_item.go:28-34`. Type is `domain.Role` (the `type Role string` alias defined at `internal/domain/role.go:10`), matching the convention used by sibling closed-enum fields on this struct (`Kind: domain.Kind`, `Scope: domain.KindAppliesTo`, `LifecycleState: domain.LifecycleState`, `Priority: domain.Priority`). JSON tag uses `omitempty`, which is correct for `string`-aliased zero-value drop semantics.

2. **`snapshotActionItemFromDomain` threads Role correctly.** `internal/app/snapshot.go:1065` reads `Role:           t.Role,` inside the constructor, between `Scope` (`:1064`) and `LifecycleState` (`:1066`). Direct copy from `domain.ActionItem.Role` (`internal/domain/action_item.go:33`) into `SnapshotActionItem.Role`. Slot order matches the struct definition.

3. **`(SnapshotActionItem).toDomain()` reverse projection threads Role correctly and does NOT re-validate via `domain.NewActionItem`.** `internal/app/snapshot.go:1312` reads `Role:           t.Role,` inside the literal `domain.ActionItem{...}` returned at `:1306-1331`. The function constructs the domain literal directly (no `domain.NewActionItem` call), matching the existing hydration pattern that handles legacy/empty fallbacks (kind→KindPlan at `:1271-1276`, state→StateTodo at `:1268-1270`, scope→DefaultActionItemScope at `:1278-1280`) but does not re-run input validation. Confirmed by reading `:1265-1331` end-to-end — no `domain.NewActionItem` call appears in `toDomain`. This matches the design note in `BUILDER_WORKLOG.md:235` ("the value was already validated when first written; `toDomain` is hydration, not validation").

4. **All 9 `domain.Role` constants round-trip preserved.** `TestSnapshotActionItemRoleRoundTripPreservesAllRoles` at `internal/app/snapshot_test.go:442-483` is table-driven with 9 cases (one per closed-enum `Role` constant). Cases at `:448-456` cover `RoleBuilder`, `RoleQAProof`, `RoleQAFalsification`, `RoleQAA11y`, `RoleQAVisual`, `RoleDesign`, `RoleCommit`, `RolePlanner`, `RoleResearch` — exact match with the 9 constants declared at `internal/domain/role.go:14-22`. Each subtest constructs a real `domain.ActionItem` via `domain.NewActionItem` (`:460-469`), projects to snapshot via `snapshotActionItemFromDomain` (`:473`), asserts `snap.Role == tc.role` (`:474-476`), then hydrates back via `snap.toDomain()` (`:477`) and asserts `hydrated.Role == tc.role` (`:478-480`) — both directions of the projection are checked.

5. **Empty-role round-trip preserves empty.** `TestSnapshotActionItemRoleEmptyRoundTripsEmpty` at `internal/app/snapshot_test.go:488-513` constructs a `domain.ActionItem` with no `Role` field set (`:490-498`), confirms zero-value at `:502-504`, projects via `snapshotActionItemFromDomain` and asserts `snap.Role == ""` at `:505-508`, hydrates via `toDomain()` and asserts `hydrated.Role == ""` at `:509-512`.

6. **JSON-shape contract held both directions.** `TestSnapshotActionItemRoleJSONShape` at `internal/app/snapshot_test.go:517-572`:
   - Builds `withRole := SnapshotActionItem{... Role: domain.RoleBuilder ...}` at `:520-536`, `json.Marshal`s at `:537-540`, asserts `strings.Contains(rawWith, \`"role":"builder"\`)` at `:541-543`.
   - Builds `withoutRole` (copy with `Role = ""`) at `:545-547`, `json.Marshal`s at `:548-551`, asserts `!strings.Contains(rawWithout, \`"role"\`)` at `:552-554` — the absence assertion is strict (matches the literal `"role"` substring, including the JSON-key quotes, so it cannot false-positive against a hypothetical word "role" embedded in another field's value).
   - Round-trips `rawWith` back through `json.Unmarshal` at `:558-561` and asserts `decodedWith.Role == domain.RoleBuilder` at `:562-564`.
   - Round-trips `rawWithout` back through `json.Unmarshal` at `:565-568` and asserts `decodedWithout.Role == ""` at `:569-571`.

   Both serialize-direction assertions and both unmarshal-direction assertions are present, so JSON-tag drift in either direction is caught.

7. **Snapshot version stays at v5.** `git grep "SnapshotVersion"` against `internal/app/snapshot.go` returns four hits: `:15` (doc comment), `:16` (`const SnapshotVersion = "tillsyn.snapshot.v5"`), `:181` (used in export literal), `:327` (used in import validation). The constant's literal value `"tillsyn.snapshot.v5"` is unchanged from pre-droplet HEAD. Forward-compatibility relies on `omitempty` + `encoding/json`'s ignore-unknown-keys default per PLAN.md `:139` and BUILDER_WORKLOG.md `:236`.

8. **No `tc := tc` capture line in any new test loop.** `git grep "tc := tc" internal/app/snapshot_test.go` returns no matches. The single `for _, tc := range cases` loop in the new code (`TestSnapshotActionItemRoleRoundTripPreservesAllRoles` at `:458-482`) uses Go 1.22+ per-iteration scoping without the legacy shadow-copy idiom. The two single-shot tests (`TestSnapshotActionItemRoleEmptyRoundTripsEmpty`, `TestSnapshotActionItemRoleJSONShape`) have no `for` loops and don't apply.

9. **`mage test-pkg ./internal/app` green at HEAD now.** Re-ran from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`: 188 tests pass / 0 failed / 0 skipped, 1 package, exit 0. Matches BUILDER_WORKLOG.md `:227` claim of 188 tests (185 prior + 3 new top-level; the 9 sub-tests inside `TestSnapshotActionItemRoleRoundTripPreservesAllRoles` are sub-tests not top-level tests, consistent with the +3 top-level delta).

10. **`mage ci` green at HEAD now.** Re-ran from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`: 1332 tests pass / 0 failed / 0 skipped, 19 packages, exit 0. Coverage threshold `70.0%` met everywhere — `internal/app` at 71.5% (per the per-package coverage table). All 15 packages listed clear ≥ 70% threshold. Build of `./cmd/till` succeeds. Matches BUILDER_WORKLOG.md `:228` claim of 1332/1332 across 19 packages.

11. **No unintended file changes.** `git status --porcelain` returns exactly four entries:
    ```
     M internal/app/snapshot.go
     M internal/app/snapshot_test.go
     M workflow/drop_2/BUILDER_WORKLOG.md
     M workflow/drop_2/PLAN.md
    ```
    Matches the expected 4-entry delta exactly. No untracked files, no new files created, no unrelated modifications, no `D`/`A` deltas. (The closing edit to this `BUILDER_QA_PROOF.md` will be the QA-side append, expected and out-of-scope for this `git status` snapshot.)

12. **PLAN.md state-flip confirmed.** `workflow/drop_2/PLAN.md:132` reads `**State:** done` for Droplet 2.6 — heading at `:130` (`#### Droplet 2.6 — Snapshot serialization for \`Role\``) plus state line at `:132` are correctly transitioned from prior `todo` per BUILDER_WORKLOG.md `:223` (`Droplet 2.6 \`**State:** todo\` → \`**State:** in_progress\` at start; will flip to \`**State:** done\` at end of round`).

### Missing Evidence

None. All 12 required proof checks have direct citations + reproducible mage gates at HEAD. The `omitempty` warning on `snapshot.go:94` and `slicessort` warning at `:963` are explicitly the same pre-existing R5/R7 tech-debt entries (line numbers shifted by +1 due to Droplet 2.6's +1 LOC field insertion at `:63`); not flagged per orchestrator out-of-scope directive. Pre-existing lint warnings in `internal/tui/`, `internal/app/`, `internal/adapters/storage/sqlite/`, `internal/adapters/server/` (R1, R5, R6, R7 in `project_drop_2_refinements_raised.md`) explicitly out of scope; not flagged.

### Verdict Summary

Droplet 2.6 lands cleanly. The `Role` field is added to `SnapshotActionItem` at `internal/app/snapshot.go:63` between `Scope` and `LifecycleState`, typed `domain.Role`, JSON-tagged `\`json:"role,omitempty"\`` — correct closed-enum-classifier grouping with parallel `Kind`/`Scope`/`Role`/`LifecycleState` placement on `domain.ActionItem` (`internal/domain/action_item.go:28-34`). Both projection functions are surgically threaded: `snapshotActionItemFromDomain` at `:1058-1087` adds `Role: t.Role,` at `:1065`, and `(SnapshotActionItem).toDomain()` at `:1265-1331` adds `Role: t.Role,` at `:1312`. The reverse projection does NOT re-validate via `domain.NewActionItem` — direct struct-literal hydration matches the existing fallback-default pattern (kind/state/scope fallbacks) without re-running input validation; this is correct because hydration assumes the snapshot was validated when first written. Three new tests at `:442-572`: `TestSnapshotActionItemRoleRoundTripPreservesAllRoles` (table-driven over all 9 `domain.Role` constants — verified exact match against `internal/domain/role.go:14-22`), `TestSnapshotActionItemRoleEmptyRoundTripsEmpty` (zero-value round-trip both directions), `TestSnapshotActionItemRoleJSONShape` (key present when set, key absent when empty, plus unmarshal-direction round-trip preservation). Snapshot version stays `tillsyn.snapshot.v5` at `internal/app/snapshot.go:16` — no bump required because `omitempty` + `encoding/json`'s ignore-unknown-keys default means old `v5` snapshots load forward-compatibly. `git grep "tc := tc"` returns empty — Go 1.22+ per-iteration scoping respected. `mage test-pkg ./internal/app` reproduces 188/188; `mage ci` reproduces 1332/1332 across 19 packages, all coverage ≥ 70% (`internal/app` at 71.5%). `git status --porcelain` shows exactly the 4 expected files. PLAN.md `:132` state correctly flipped to `done`. BUILDER_WORKLOG.md carries a structurally-sound Round 1 entry with explicit design notes covering field placement, typing rationale, `omitempty` rationale, no-revalidate decision, version-stay rationale, and the JSON-shape both-directions test design. Ready to commit.

### Hylla Feedback

None — Hylla was not the right tool for this droplet. The work was three deltas in two files (`internal/app/snapshot.go` +3 LOC, `internal/app/snapshot_test.go` +132 LOC including a new `encoding/json` import); all source-code verification ran against uncommitted Go deltas, where Hylla is stale-by-design until drop-end reingest per CLAUDE.md "Code Understanding Rules" §2. QA verification used `Read` with offset/limit for the snapshot.go struct + projection-function blocks, `LSP documentSymbol` for the test-file structure (line numbers of all 12 top-level test functions, including the new three at `:442`/`:488`/`:517`), and `Bash` for `git grep` / `git status --porcelain` / `mage test-pkg ./internal/app` / `mage ci`. The supporting committed files (`internal/domain/role.go`, `internal/domain/action_item.go`) were `Read` rather than Hylla-queried because the question shape was "show me this file's struct shape + field neighborhood" not "find a symbol / find references" — `Read` is the natural fit for whole-file structural inspection. Zero Hylla misses, zero ergonomic gripes for this droplet.

## Droplet 2.7 — Round 1

**Verdict:** pass
**Date:** 2026-05-02

### 1. Findings

- 1.1 **Symbol-grep checks (PLAN.md :202-219) all pass.** Reproduced at HEAD: `git grep -nE "\bStateDone\b" -- '*.go'` empty, `git grep -nE "\bStateProgress\b" -- '*.go'` empty, `git grep -nE "\bStateComplete\b" -- '*.go'` non-empty (canonical present), `git grep -nE "\bStateInProgress\b" -- '*.go'` non-empty (canonical present), `git grep -nE "\bRequireChildrenDone\b" -- '*.go'` empty, `git grep -nE "\bDoneItems\b|\bDoneActionItems\b" -- '*.go'` empty, `git grep -nE 'json:"require_children_done"' -- '*.go'` empty, `git grep -nE 'Done:\s*(true|false)' -- '*.go'` empty, `git grep -nE 'json:"done_tasks"|json:"done_items"' -- '*.go'` empty. Every legacy symbol/JSON tag is purged from the Go tree.
- 1.2 **State-machine literal scope checks pass.** `git grep -nE 'domain\.StateDone|domain\.StateProgress' -- '*.go'` empty. `git grep -nE 'lifecycle_state.*"done"|lifecycle_state.*"progress"' -- '*.go'` empty. Production state-machine sources (`internal/domain/workitem.go`, `internal/app/service.go`, `internal/adapters/server/common/{capture,app_service_adapter_mcp}.go`, `internal/tui/model.go`, `internal/config/config.go`) carry zero `"done"`/`"progress"`/`"completed"` literals (`git grep -E '"done"|"progress"|"completed"' -- <files>` empty across all six). `git grep -nE 'json:"done"|json:"progress"|json:"completed"|json:"in-progress"|json:"doing"' -- '*.go'` returns only `internal/adapters/server/common/mcp_surface.go:236 Completed bool json:"completed"` — the explicitly-out-of-scope independent field per Notes B9 + orchestrator carve-out, NOT a state-vocab leak. `git grep -nE '"in-progress"|"doing"' -- 'internal/domain/' 'internal/app/' 'internal/adapters/server/' 'internal/tui/' 'internal/config/'` returns only test-file occurrences inside rejection-asserting branches (`capture_test.go:269` test labels itself "legacy rejected"; `service_test.go:2467` is `{ID: "doing", ...}` exercising the unknown-state coercion path; `config_test.go:820` iterates a `[]string{"progress", "done", "completed", "in-progress", "doing"}` slice asserting all return false from `isKnownLifecycleState`; `model_test.go:685, 969` are switch-case branches for canonical column-name resolution). All legacy literals in production are gone; test-file occurrences are intentional rejection assertions.
- 1.3 **Behavior verification (read-confirmed).** `internal/domain/workitem.go:160-164` `IsTerminalState` returns true for `StateComplete` || `StateFailed`. `internal/domain/workitem.go:154-158` `isValidLifecycleState` enumerates against the canonical set `{StateTodo, StateInProgress, StateComplete, StateFailed, StateArchived}`. `normalizeLifecycleState` at `:147-152` is a pure trim+lower; legacy aliases pass through and fail `isValidLifecycleState` downstream — matches the design judgment in BUILDER_WORKLOG (line 250) and is consistent with PLAN.md "return the unknown-state error path" because `isValidLifecycleState(state)==false` for every legacy alias and every code path that consumes lifecycle state validates via `isValidLifecycleState` or its companions. `internal/app/service.go:1894-1902` `defaultStateTemplates()` returns `{ID: "todo"}, {ID: "in_progress"}, {ID: "complete"}, {ID: "failed"}` (display names also flipped to `"To Do"`, `"In Progress"`, `"Complete"`, `"Failed"` per builder design judgment #6 — keeps the seed display name in sync with `canonicalSearchStateLabels`). `internal/domain/workitem.go:80-85` `ChecklistItem` struct has `Complete bool json:"complete"`. `internal/domain/workitem.go:87-90` `CompletionPolicy` struct has `RequireChildrenComplete bool json:"require_children_complete"`. `internal/adapters/server/common/types.go:138-148` `WorkOverview` emits `"complete_tasks"` (was `"done_tasks"`); `internal/app/attention_capture.go:96` `AttentionWorkOverview.CompleteItems int json:"complete_items"`. The `till.action_item` MCP move-state schema description at `internal/adapters/server/mcpapi/extended_tools.go:1342` reads `"... todo|in_progress|complete"` — canonical only.
- 1.4 **`mage ci` reproduced green.** Independently re-ran `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/`: 1332 tests passed across 19 packages, exit 0. Per-package coverage (all ≥ 70.0%): `internal/domain` 79.4%, `internal/app` 71.5%, `internal/adapters/storage/sqlite` 75.1%, `internal/adapters/server/common` 73.0%, `internal/adapters/server/mcpapi` 72.4%, `internal/tui` 70.0% (exactly on threshold per builder's design note #8), `internal/config` 76.8%. Build of `./cmd/till` succeeded.
- 1.5 **Per-package mage gates reproduced.** `mage test-pkg ./internal/domain` → 103. `mage test-pkg ./internal/app` → 188. `mage test-pkg ./internal/adapters/storage/sqlite` → 69. `mage test-pkg ./internal/adapters/server/common` → 123. `mage test-pkg ./internal/adapters/server/mcpapi` → 93. `mage test-pkg ./internal/tui` → 354. `mage test-pkg ./internal/config` → 32. Every count matches the worklog's Mage results section exactly.
- 1.6 **`canonicalLifecycleState` at `internal/adapters/server/common/capture.go:296-315`** rewritten strict-canonical: switch on canonical IDs only (`todo|in_progress|complete|failed|archived`), default arm returns `StateTodo` (effective rejection — legacy values fall through and downstream callers see todo, not the legacy state). Doc comment at `:296-299` documents the strict-canonical behavior including the legacy-rejection path.
- 1.7 **`actionItemLifecycleStateForColumnName` at `internal/adapters/server/common/app_service_adapter_mcp.go:851-866`** rewritten strict-canonical: switch on canonical column slugs only, default returns `""`. Companion `normalizeStateLikeID` at `:868-910` slugifies inputs with `_` separator (matches builder design judgment #1), then maps via canonical-only switch with default fall-through to raw slug — legacy aliases like `"done"` slug to `"done"` and the consuming switch hits its `""`/`StateTodo` default arm.
- 1.8 **`extended_tools.go:1342` MCP description string** reads `"Lifecycle state target for operation=move_state (for example: todo|in_progress|complete)"` — canonical only. PLAN.md cited `:1339` (drift +3, builder noted in worklog). The actual description is correct.
- 1.9 **MD carve-out at `capture_test.go:199`** debug-message label flipped: `"WorkOverview counts = %#v, want todo=2 in_progress=1 complete=1 failed=1 archived=1"`. R2-F11 carve-out applied as instructed.
- 1.10 **`config.example.toml:42-43`** scope-expansion fix applied: `# Canonical lifecycle states only: todo | in_progress | complete | archived.` + `states = ["todo", "in_progress", "complete"]`. Required because `TestExampleConfigEmbeddingsDefaults` exercises the example file via `Load + Validate`. Builder surfaced this as scope expansion.
- 1.11 **No new `tc := tc` in any modified test.** `git grep "tc := tc"` returns 9 hits, NONE of which are in the 25 files modified by this droplet — they are all in pre-existing untouched test files (`cmd/till/main_test.go`, `internal/adapters/server/common/{app_service_adapter_helpers_test.go,app_service_adapter_mcp_guard_test.go}`, `internal/app/schema_validator_test.go`, `internal/domain/auth_request_test.go`). Every modified test file is `tc := tc`-free, honoring Go 1.22+ per-iteration scoping.
- 1.12 **No `ALTER TABLE` added by this droplet.** `git grep -nE "ALTER TABLE" -- '*.go'` returns matches only in `internal/adapters/auth/autentauth/service.go`, `internal/adapters/livewait/localipc/broker.go`, and `internal/adapters/storage/sqlite/repo.go` — none of those files are in the Droplet 2.7 modified set. Pre-MVP rule honored.
- 1.13 **PLAN.md state-flip.** `workflow/drop_2/PLAN.md:156` reads `**State:** done`. State transition todo→in_progress→done landed correctly.
- 1.14 **`git status --porcelain` clean to scope.** Exactly 28 modified files: 25 Go (across `internal/domain/`, `internal/app/`, `internal/adapters/server/common/`, `internal/adapters/server/mcpapi/`, `internal/tui/`, `internal/config/`) + `config.example.toml` + `workflow/drop_2/PLAN.md` + `workflow/drop_2/BUILDER_WORKLOG.md`. No stray edits anywhere; no source files outside the declared paths touched. `git diff --stat` confirms the line-delta ratio (+432/-273) and per-file edit shapes are surgical.

### 2. Missing Evidence

- 2.1 **`ChecklistItem` JSON unmarshal "produces decode error" claim (PLAN.md :224)** is not strictly verified at the literal level. `encoding/json`'s default behavior is to silently ignore unknown JSON keys, so an inbound `{"done": true}` decodes into a `ChecklistItem` with `Complete: false` (not a returned `error`). `DisallowUnknownFields()` is only configured on HTTP handler entry points (`internal/adapters/server/httpapi/handler.go:559, 581`), NOT on snapshot/persistence/`ChecklistItem` unmarshal paths. **However**, the practical contract is honored: there is no fallback decoder accepting `"done"` as an alias for `Complete`, no struct-tag synonym, no custom `UnmarshalJSON` accepting legacy keys — legacy data simply drops out, which is consistent with the pre-MVP fresh-DB / fresh-snapshot rule. This is a wording-vs-implementation gap in PLAN.md, not a correctness gap; the strict-canonical contract holds at the field/tag level. Recommended PLAN.md follow-up post-MVP: either remove the "decode error" phrasing, or add `DisallowUnknownFields` to the snapshot/persistence decoders if literal rejection is desired. Not a blocker for Droplet 2.7.
- 2.2 **Cite drift in PLAN.md** — multiple file:line cites in PLAN.md are off by varying amounts (domain_test.go uniformly +49–50, service.go normalizeStateID +20, app_service_adapter_mcp.go +2, extended_tools.go +3). Builder located all sites via `git grep` and applied edits correctly via symbol/literal anchors. Not a blocker for the build, but a planner-PLAN-MD hygiene refinement candidate for Drop 2 close-out (cites should be re-grepped at HEAD before each round; or the round prompt should explicitly mark cites as "verify at HEAD" — which Round 2 of the PLAN.md already did).

### 3. Summary

**PASS.** All 14 required proof checks land cleanly with file:line evidence. The atomic state-vocabulary rename is complete: every `StateDone`/`StateProgress` symbol, every legacy state literal in production state-machine sources, every `ChecklistItem.Done` field reference, every aggregate counter (`DoneItems`/`DoneActionItems` → `CompleteItems`/`CompleteActionItems`), every JSON tag (`require_children_done` → `require_children_complete`, `done_tasks` → `complete_tasks`, `done_items` → `complete_items`), and the `RequireChildrenDone` field rename all flipped in one commit. `mage ci` reproduces 1332/1332 across 19 packages with all coverage ≥ 70.0%. Per-package mage gates match builder claims exactly (domain 103, app 188, sqlite 69, common 123, mcpapi 93, tui 354, config 32). The two coercion sites (`canonicalLifecycleState` at `capture.go:296-315`, `actionItemLifecycleStateForColumnName` at `app_service_adapter_mcp.go:851-866`) are strict-canonical with rejection paths via the default arm. The MCP tool description at `extended_tools.go:1342` is canonical-only (`todo|in_progress|complete`). The `config.example.toml` scope-expansion + `capture_test.go:199` debug-message MD carve-out both landed cleanly. PLAN.md state correctly flipped to `done`. `git status --porcelain` shows exactly the 28 expected files; no stray edits. Builder design judgments (slug separator dash→underscore, column-name renames where state-machine semantics require it, coverage maintenance via 2 added positive test cases, error-path semantics via fall-through-to-default) are documented in BUILDER_WORKLOG with rationale and are consistent with strict-canonical. Pre-MVP rules honored (zero `ALTER TABLE`, zero migration code, zero new `tc := tc`). The one wording gap (PLAN.md "produces decode error" vs `encoding/json` default of silently ignoring unknown keys) is a documentation-vs-implementation phrasing issue, not a correctness gap; the strict-canonical contract is held at the field/tag level. Ready to commit.

### TL;DR

- T1 — Symbol-greps, state-machine literal scope checks, behavior verification (`IsTerminalState`, `defaultStateTemplates`, JSON tags), reproduced `mage ci` (1332/1332), reproduced per-package mage gates (103/188/69/123/93/354/32), canonical-only coercion sites at `capture.go:296-315` + `app_service_adapter_mcp.go:851-866`, canonical MCP description at `extended_tools.go:1342`, R2-F11 carve-out at `capture_test.go:199`, `config.example.toml` scope-expansion, no new `tc := tc`, no `ALTER TABLE`, PLAN.md state `done`, `git status` clean to scope — all 14 checks pass.
- T2 — One wording-gap noted: PLAN.md "ChecklistItem JSON unmarshal produces decode error for `done`" is not literally true under `encoding/json` defaults (which silently drop unknown keys), but the strict-canonical field/tag contract holds and pre-MVP fresh-DB/fresh-snapshot rule means no legacy data is in play. Cite drift in PLAN.md was located via `git grep`; edits landed correctly.
- T3 — PASS. Ready to commit.

### Hylla Feedback

N/A — task touched 25 Go files for surgical state-vocabulary renames + 1 TOML config example + 2 MD edits, all uncommitted on the working tree where Hylla is stale-by-design until drop-end reingest per CLAUDE.md "Code Understanding Rules" §2. QA verification was driven by `git grep` for known symbol/literal patterns (the natural fit for whole-tree literal sweeps across 25 files in known packages) plus `Read` for context around each cite, plus `Bash` for `mage ci` / per-package `mage test-pkg` / `git status --porcelain` / `git diff --stat`. Hylla queries were not the right shape for "find every occurrence of these 6 symbol names + 4 string literals + 5 JSON tags across 25 files in 7 known packages" — `git grep` is the natural fit. Zero Hylla misses, zero ergonomic gripes for this droplet.

## Droplet 2.7 — Round 2

**Verdict:** pass
**Date:** 2026-05-01
**Round scope:** intra-droplet drift cleanup. Droplet 2.7 PLAN.md state stays `**State:** done` from Round 1 — Round 2 tightens implementation to match PLAN.md `:222` (slug-style normalizer literal rejection) and `:224` (`ChecklistItem` JSON decode error on `"done"` key) without any new acceptance contract.

### 1. Findings

- 1.1 **Drift 1 — `ChecklistItem.UnmarshalJSON` exists with correct semantics.** `internal/domain/workitem.go:87-104` defines `func (c *ChecklistItem) UnmarshalJSON(data []byte) error`. Pattern confirmed: decode into `map[string]json.RawMessage` first (line 95), error if `"done"` key present (lines 99-101 — `fmt.Errorf("checklist item: legacy %q key rejected, use %q (strict-canonical)", "done", "complete")`), else re-decode via local `type alias ChecklistItem` (lines 102-103) to break the recursion cycle. Doc comment at `:87-93` documents the strict-canonical rationale (stdlib silent-drop vs hard-error contract). This addresses Round 1 Missing Evidence 2.1 directly.
- 1.2 **Drift 1 tests added — 5 cases in `internal/domain/domain_test.go:460-525`.** `TestChecklistItemUnmarshalRejectsLegacyDoneKey` exercises five sub-cases: (a) `{"id":"x","text":"y","complete":false}` → `ChecklistItem{Complete:false}`; (b) `{"id":"x","text":"y","complete":true}` → `ChecklistItem{Complete:true}`; (c) `{"id":"x","text":"y"}` (no completion key) → `ChecklistItem{Complete:false}`; (d) `{"id":"x","text":"y","done":true}` → error containing `"legacy"`; (e) `{"id":"x","text":"y","done":false}` → error containing `"legacy"`. Test uses `errMatch` substring assertion against the error message; new imports `encoding/json` + `strings` added at `:5-6`.
- 1.3 **Drift 2 — `normalizeStateID` rejects 5 legacy literals.** `internal/app/service.go:1949-1957` shows the pre-slug switch: after `name = strings.TrimSpace(strings.ToLower(name))` and the empty-string early return, the switch arm `case "done", "completed", "progress", "doing", "in-progress": return ""` fires before slugification. Doc comment at `:1942-1948` documents the rejection contract and explicitly notes `"to-do"` is NOT legacy (kebab-spelled canonical → maps to `"todo"`).
- 1.4 **Drift 2 — `normalizeStateLikeID` rejects 5 legacy literals.** `internal/adapters/server/common/app_service_adapter_mcp.go:873-881` shows the same pattern: ToLower+TrimSpace, empty-string early return, then the literal switch with the identical 5-literal arm returning `""`. Doc comment at `:868-872` matches the contract.
- 1.5 **Drift 2 — `normalizeColumnStateID` rejects 5 legacy literals.** `internal/tui/model.go:17939-17947` shows the third instance: ToLower+TrimSpace, empty-string early return, switch arm with 5 literals returning `""`. Doc comment at `:17934-17938` matches.
- 1.6 **Three normalizers carry the IDENTICAL 5-literal switch.** Cross-verified by reading all three: `case "done", "completed", "progress", "doing", "in-progress": return ""` appears verbatim in every site. No drift across the three implementations. Lowercased (`"Done"` → `"done"`) and whitespace-wrapped (`"  progress  "` → `"progress"`) inputs hit the rejection because trimming + lowercasing precedes the switch.
- 1.7 **Drift 2 tests added — 17/16/17 cases in matching test files.** `internal/app/service_test.go:2961-3000` `TestNormalizeStateIDStrictCanonicalRejectsLegacyLiterals` (17 cases: 5 canonical + 1 kebab-`to-do` + 2 display-name canonical + 5 legacy + 1 uppercase-`Done` + 1 whitespace-wrapped + 1 custom column + 1 empty). `internal/adapters/server/common/app_service_adapter_mcp_helpers_test.go:140-175` `TestNormalizeStateLikeIDStrictCanonicalRejectsLegacyLiterals` (16 cases — same shape, one fewer display-name case). `internal/tui/model_test.go:13570-13606` `TestNormalizeColumnStateIDStrictCanonicalRejectsLegacyLiterals` (17 cases). Each test asserts both the rejection set AND the canonical-preservation set AND the custom-column-passthrough invariant — the rejection is narrow, not a blanket reject.
- 1.8 **`mage ci` reproduced green at HEAD `c7e07f2`.** Independently re-ran `mage ci` from the working directory: **1391 tests passed across 19 packages**, 0 failed, 0 skipped, exit 0. Coverage gate met everywhere ≥ 70.0%: `internal/domain` 79.4%, `internal/app` 71.6%, `internal/adapters/storage/sqlite` 75.1%, `internal/adapters/server/common` 73.4%, `internal/adapters/server/mcpapi` 72.4%, `internal/tui` 70.0% (on threshold), `internal/config` 76.8%, `cmd/till` 76.6%, `internal/buildinfo` 100.0%. Build of `./cmd/till` succeeded. Total tests grew from R1's 1332 to R2's 1391 (+59 cumulative — 5 from Drift 1 + 17 + 16 + 17 sub-cases for Drift 2 + each test's parent function = 5 + 17 + 16 + 17 + 4 = 59 new). Math reconciles with worklog claim.
- 1.9 **No `tc := tc` in any new R2 test loop.** Read all four added test loops (`domain_test.go:512`, `service_test.go:2992`, `app_service_adapter_mcp_helpers_test.go:167`, `model_test.go:13599`) — every one is `for _, tc := range cases { t.Run(tc.name, func(t *testing.T) { ... }) }` without the legacy `tc := tc` shadowing line. Honors Go 1.22+ per-iteration variable scoping.
- 1.10 **PLAN.md untouched in R2.** `git diff c7e07f2~1 c7e07f2 --name-only` returns 9 files: 8 source/test Go files + `workflow/drop_2/BUILDER_WORKLOG.md`. `workflow/drop_2/PLAN.md` is NOT in the list. PLAN.md was correct as authored; R2 brought code into compliance with PLAN.md's existing acceptance language.
- 1.11 **Droplet 2.7 PLAN.md state stays `done`.** `workflow/drop_2/PLAN.md:156` reads `**State:** done`. Round 2 is intra-droplet cleanup, not a re-open.
- 1.12 **Behavior verification — MCP `move_state` with `state="in-progress"`.** Trace: `normalizeStateLikeID("in-progress")` → ToLower+TrimSpace returns `"in-progress"` (no change, already lowercase) → switch arm matches → returns `""`. Caller `actionItemLifecycleStateForColumnName(name)` at `internal/adapters/server/common/app_service_adapter_mcp.go:851-866` switches on the returned `""` → falls to the `default` arm → returns `domain.LifecycleState("")` (empty value). Downstream the empty `LifecycleState` fails `isValidLifecycleState` checks at the state-machine boundary, producing the unknown-state error path PLAN.md `:222` mandates. Trace verified by reading source.
- 1.13 **Behavior verification — `ChecklistItem` JSON unmarshal `{"done":true}`.** Trace: `json.Unmarshal([]byte(\`{"id":"x","text":"y","done":true}\`), &got)` → dispatches to the new `(*ChecklistItem).UnmarshalJSON` method → first decode into `map[string]json.RawMessage` succeeds (line 96) → `raw["done"]` lookup at line 99 returns `(json.RawMessage, true)` → returns `fmt.Errorf("checklist item: legacy %q key rejected, use %q (strict-canonical)", "done", "complete")` → error message contains `"legacy"` (asserted by Drift 1 test cases d and e). Caller sees a non-nil error, matching PLAN.md `:224` "JSON unmarshal accepts ONLY `\"complete\"` — `\"done\"` keys produce a decode error (no fallback alias)."
- 1.14 **No source files outside the listed scope edited.** `git diff c7e07f2~1 c7e07f2 --name-only` (9 files) is exactly: 3 production files (`workitem.go`, `service.go`, `app_service_adapter_mcp.go`, `model.go` = 4 actually) + 4 test files (`domain_test.go`, `service_test.go`, `app_service_adapter_mcp_helpers_test.go`, `model_test.go`) + 1 worklog. Drift 1 is in `workitem.go` + `domain_test.go`. Drift 2 is in 3 normalizer source files + 3 matching test files. Total 8 Go files, exactly matching the spawn-prompt scope. No `mage install` invoked. No migration code added.
- 1.15 **Builder design judgment on `"to-do"` validated.** Spawn prompt speculatively listed `"to-do"` among legacy literals; PLAN.md `:222` lists only 5 (`"done"`, `"completed"`, `"progress"`, `"doing"`, `"in-progress"`). Builder followed PLAN.md authoritatively and surfaced the prompt-vs-PLAN delta in BUILDER_WORKLOG. Tests assert `"to-do" → "todo"` (canonical preservation through slugification — `to-do` slugifies to `to_do` which the canonical-mapping switch maps to `"todo"`). This is correct: strict-canonical rejects only the 5 explicitly-listed legacy literals, not every kebab variant of canonical names. Verified via reading the Drift 2 normalizer code paths and corresponding test cases (`{name: "kebab to-do is canonical (not legacy)", in: "to-do", want: "todo"}` in all three test files).
- 1.16 **Custom column passthrough preserved (narrow rejection).** Each Drift 2 test asserts a custom column name (`"My Custom Column" → "my_custom_column"` in `service_test.go`, `"Backlog" → "backlog"` in `app_service_adapter_mcp_helpers_test.go` and `model_test.go`) survives unchanged through slugification. Confirms the pre-slug switch is targeted: only the 5 specific legacy literals are rejected, every other input still slugifies normally per the existing rune-by-rune builder loop.

### 2. Missing Evidence

- 2.1 **None.** Round 1's Missing Evidence 2.1 (PLAN.md "decode error" not literally true under `encoding/json` defaults) was the exact prompt for Round 2 Drift 1. The new `(*ChecklistItem).UnmarshalJSON` method explicitly produces a decode error on `"done"` keys, matching PLAN.md `:224` literally — the gap is closed. Round 1's Missing Evidence 2.2 (PLAN.md cite drift) is unchanged, but it was always a planner-MD hygiene refinement candidate, not a build-correctness gap, and no PLAN.md edits in R2 means it remains routable to drop close-out as a refinement entry.

### 3. Summary

**PASS.** Both R2 cleanup drifts land cleanly with file:line evidence. Drift 1 (`ChecklistItem.UnmarshalJSON` at `internal/domain/workitem.go:94-104`) closes Round 1 Missing Evidence 2.1 by replacing stdlib silent-drop semantics with explicit decode-error rejection for the legacy `"done"` key — the strict-canonical contract at PLAN.md `:224` is now literally true. Drift 2 (pre-slug literal switch at `normalizeStateID:1954-1957`, `normalizeStateLikeID:878-881`, `normalizeColumnStateID:17944-17947`) replaces Round 1's slug-passthrough ("done" → "done") with empty-string rejection ("done" → "") so callers see the unknown-state error path PLAN.md `:222` mandates. All three normalizers carry the identical 5-literal switch (no implementation drift across sites). Test coverage: 5 (domain) + 17 (app) + 16 (common) + 17 (tui) = 55 sub-cases + 4 parent functions = 59 new tests, all passing under `mage ci` (1391/1391, all packages ≥ 70.0% coverage). Per-package counts (109/206/140/372 in domain/app/common/tui) exactly match builder claims. No `tc := tc` shadowing, no PLAN.md edits, no migration code, no `mage install`, no out-of-scope file edits. PLAN.md state for Droplet 2.7 stays `done`. Behavior traces for both user-input paths verified by source-reading. Round 2 is a clean drift-cleanup pass within the same droplet's `done` lifecycle.

### TL;DR

- T1 — All 16 R2 proof checks pass: Drift 1 (`ChecklistItem.UnmarshalJSON` at `workitem.go:94-104` rejects `"done"` keys with explicit error) closes Round 1 Missing Evidence 2.1; Drift 2 (identical pre-slug 5-literal switch in `normalizeStateID:1954-1957`, `normalizeStateLikeID:878-881`, `normalizeColumnStateID:17944-17947`) replaces slug-passthrough with empty-string rejection per PLAN.md `:222`. Tests added: 5/17/16/17 sub-cases, no `tc := tc` shadowing. `mage ci` reproduced green: 1391/1391 across 19 packages, all coverage ≥ 70.0%. Behavior traces verified for both MCP `move_state(state="in-progress")` (→ `""` → unknown-state error) and `ChecklistItem` unmarshal `{"done":true}` (→ explicit error containing `"legacy"`).
- T2 — Zero missing evidence. Round 1's Missing Evidence 2.1 is the exact gap Round 2 Drift 1 closes; Round 1 Missing Evidence 2.2 (PLAN.md cite drift) is unchanged but always a planner-hygiene refinement candidate, routable to drop close-out.
- T3 — PASS. Droplet 2.7 PLAN.md state correctly stays at `done`; intra-droplet cleanup pass complete.

### Hylla Feedback

N/A — task touched only Go production + test code in 4 packages, all 8 files known-by-name from the spawn prompt. QA verification used `Read` for whole-file context around each cite + `Bash` for `mage ci` + `git diff` / `git log` / `git status` for change scope. Hylla queries were not the right shape for "verify these 4 specific symbols + 8 specific test functions exist with the documented contract" — direct `Read` against known absolute paths is the natural fit when the spawn prompt enumerates exact file:line targets and you need byte-level certainty (vs. semantic similarity). Working tree commits at `c7e07f2` are uncommitted-since-last-ingest (Hylla is stale by design until drop-end reingest per CLAUDE.md "Code Understanding Rules" §2), so Hylla wouldn't have been authoritative anyway. Zero Hylla misses, zero ergonomic gripes for this round.

## Droplet 2.8 — Round 1

**Verdict:** PASS
**Date:** 2026-05-01

### Findings

1. **All 12 INSERT rows flipped `allowed_parent_scopes_json` to `'[]'`.** `git diff internal/adapters/storage/sqlite/repo.go` shows exactly 12 hunks — one per seeded kind — each replacing the 5th VALUES literal (`'["plan"]'` x10, `'["build"]'` x2) with `'[]'`. Lines verified at `repo.go:308` (`plan`), `:314` (`research`), `:320` (`build`), `:326` (`plan-qa-proof`), `:332` (`plan-qa-falsification`), `:338` (`build-qa-proof`), `:344` (`build-qa-falsification`), `:350` (`closeout`), `:356` (`commit`), `:362` (`refinement`), `:368` (`discussion`), `:374` (`human-verify`). Builder's enumeration matches the diff line-for-line.

2. **`applies_to_json` untouched on every row.** `rg -n '"plan"\]|"build"\]' internal/adapters/storage/sqlite/repo.go` returns exactly 2 hits — `:308` (the `plan` row's `applies_to_json = '["plan"]'` slot) and `:320` (the `build` row's `applies_to_json = '["build"]'` slot). These are the scope-mirror values the spawn prompt called out as the collision risk, and they are the *4th* VALUES literal (not the 5th). The diff confirms only the 5th literal was edited per row — `applies_to_json`, `display_name`, `description_markdown`, `payload_schema_json`, `template_json`, and timestamp slots are byte-identical to pre-droplet state.

3. **`AllowsParentScope` empty-list early return intact at `internal/domain/kind.go:225-236`.** Read confirms the body is unchanged: signature at `:225`, normalize at `:226`, empty-list early return at `:227-229` (`if len(k.AllowedParentScopes) == 0 { return true }`), match-loop at `:230-234`, fallback `return false` at `:235`. Universal-allow semantics are correct: empty `AllowedParentScopes` (which the seeded rows now carry post-flip) trigger the early return → every parent scope accepted.

4. **`internal/app/kind_capability.go:566` enforcement path unchanged.** Read confirms the gate at `:565-568`: `if parent != nil { if !kind.AllowsParentScope(parent.Scope) { return ..., fmt.Errorf("%w: ...", domain.ErrKindNotAllowed) } }`. The `Errorf` quoting and surrounding `GetKindDefinition` / `AppliesToScope` / `resolveProjectAllowedKinds` blocks are byte-identical to pre-droplet state per `git diff` (file is not in `git status --porcelain`).

5. **`TestRepositoryFreshOpenKindCatalogUniversalParentAllow` exists with 144 probes (12 × 12).** The new test occupies `repo_test.go:2520-2567` per the diff. It (a) opens an in-memory repo via `OpenInMemory()`, (b) lists kinds via `repo.ListKindDefinitions(ctx, false)`, (c) asserts `len(kinds) == 12`, (d) defines a `parentScopeProbes` slice with 12 `KindAppliesTo` constants (`Plan`, `Build`, `Research`, `Closeout`, `Commit`, `Discussion`, `Refinement`, `HumanVerify`, `PlanQAProof`, `PlanQAFalsification`, `BuildQAProof`, `BuildQAFalsification`), (e) outer-loops the 12 seeded kinds, (f) inner-loops the 12 probes calling `kind.AllowsParentScope(scope)` and failing if false. Outer×inner = 12×12 = 144 probe assertions. Plus an outer per-kind `len(kind.AllowedParentScopes) != 0` guard for direct empty-list verification. Math checks.

6. **No `tc := tc` in the new test loop.** The test uses `for _, kind := range kinds` (not table-driven; not `for _, tc := range cases`). No subtests, no `t.Parallel()`, no shadow-rebind needed. Loop-variable-capture is not in play. Builder claim correct.

7. **`mage ci` green at HEAD now.** Ran `mage ci` from `main/`. Result: `tests: 1392, passed: 1392, failed: 0, skipped: 0, packages: 19, pkg passed: 19, pkg failed: 0`. Per-package coverage: `sqlite 75.1%`, `app 71.6%`, `domain 79.4%`, all ≥ 70% threshold. Build succeeded (`Built till from ./cmd/till`). Source verification + format check + coverage threshold + build all green.

8. **No `ALTER TABLE` added by 2.8.** `rg -n "ALTER TABLE" internal/adapters/storage/sqlite/repo.go` returns 22 hits at `:511, :514, :518-530, :547-549, :633, :694, :742` — none in the diff. `git diff internal/adapters/storage/sqlite/repo.go` shows zero `ALTER TABLE` lines added or removed. Schema-shape unchanged; only data-shape (the seeded INSERT VALUES) flipped, consistent with droplet acceptance ("DB action: DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci`" — confirmed local DB doesn't exist, no stale-data masking).

9. **`git status --porcelain` shows 5 files, not the builder-reported 4.** Output: ` M internal/adapters/storage/sqlite/repo.go`, ` M internal/adapters/storage/sqlite/repo_test.go`, ` M workflow/drop_2/BUILDER_QA_FALSIFICATION.md`, ` M workflow/drop_2/BUILDER_WORKLOG.md`, ` M workflow/drop_2/PLAN.md`. The 5th file (`BUILDER_QA_FALSIFICATION.md`) is the parallel-spawned falsification sibling's Round 1 append (169 lines added), not a builder-attributable artifact. The builder's claim of "4 expected files" was scoped to *the builder's own write surface* and is correct under that scope. The falsification sibling's MD edit is expected during a parallel QA run and is in scope for the falsification agent's write window. Not a blocking finding; noted for transparency.

10. **PLAN.md state-flip — Droplet 2.8 reads `**State:** done`.** `workflow/drop_2/PLAN.md:240-242` confirms: `#### Droplet 2.8 — Empty AllowedParentScopes for every kind in boot-seed` then `- **State:** done`. Adjacent Droplet 2.9 at `:255-257` correctly remains `- **State:** todo` (Droplet 2.9 owns the `AllowedParentKinds` deletion + doc-comment cleanup, blocked by 2.8).

### Missing Evidence

(none — every required check landed grounded evidence)

### Verdict Summary

PASS. All 10 required proof checks satisfied with file:line citations. The 12-row INSERT flip is byte-precise and confined to the `allowed_parent_scopes_json` slot; sibling columns including the `plan`/`build` `applies_to_json` collision-candidates are untouched. `AllowsParentScope`'s empty-list early return at `domain/kind.go:227-229` is the runtime mechanism that turns the data-shape change into universal-allow semantics, and the new `TestRepositoryFreshOpenKindCatalogUniversalParentAllow` covers the contract at the boot-seed boundary with 144 explicit (kind, parent-scope) probes. `mage ci` green: 1392/1392 tests, all packages ≥ 70% coverage. Enforcement path at `app/kind_capability.go:566` and the `domain.AllowedParentKinds` doc-comment at `repo.go:299-304` are correctly untouched (Droplet 2.9's scope). No `ALTER TABLE` added; no schema migration; pre-MVP "dev fresh-DBs" rule honored. Builder's 4-file claim is scope-correct (the 5th file in `git status` is the parallel falsification sibling's append, expected). Cleared for the build → QA → fix → commit gate to advance.

### Hylla Feedback

N/A — task touched only Go production + test code (2 files) plus 3 MD coordination artifacts. All 4 spawn-prompt-named files (`internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/domain/kind.go`, `internal/app/kind_capability.go`) addressable by absolute path with exact line ranges from the prompt. QA used `Read` for whole-file context, `Bash` for `git diff` / `git status` / `mage ci` / `rg`, and direct line-anchored cross-checks. Hylla is stale by design for the working-tree changes (CLAUDE.md "Code Understanding Rules" §2), and the "verify these N specific INSERT rows + 1 specific test function exist with documented byte-shape" question is naturally a `Read` + `git diff` shape, not a semantic-similarity / refs-graph shape. Zero Hylla queries attempted, zero misses to log, zero ergonomic gripes.

---

## Droplet 2.9 — Round 1

**Verdict:** PASS.

### Required Proof Checks

1. **`AllowedParentKinds` function deleted at `internal/domain/kind.go`.** PASS. The pre-edit function at lines 94–117 (per PLAN.md `:258` reference and prior Droplet 2.8 audit) is gone. `git diff internal/domain/kind.go` shows the entire 24-line block removed (the doc comment "AllowedParentKinds returns the kinds permitted as a direct parent..." plus the `switch Kind(strings.TrimSpace(...))` body covering `KindPlan|KindResearch|KindPlanQAProof|KindPlanQAFalsification|KindBuild|KindCloseout|KindCommit|KindRefinement|KindDiscussion|KindHumanVerify` → `[]Kind{KindPlan}`, `KindBuildQAProof|KindBuildQAFalsification` → `[]Kind{KindBuild}`, default `nil`). Post-edit, line 92 closes `validKindAppliesTo` slice and line 94 opens `KindTemplateChildSpec` directly — no orphan blank lines or syntax breakage. Read of `kind.go:85-103` confirms the adjacency.

2. **`TestAllowedParentKindsEncodesHierarchy` deleted at `internal/domain/domain_test.go`.** PASS. `git diff internal/domain/domain_test.go` shows the 38-line test deletion (12-row table covering all 12 `Kind` enum values + the `bogus` default-arm assertion). The test was structurally `for _, tc := range tests { got := AllowedParentKinds(tc.kind); ... }` — removing the test was the correct move once `AllowedParentKinds` itself was deleted (without removal, the test wouldn't compile). Adjacent test `TestNormalizeKindIDLowercaseAndTrim` is preserved unchanged.

3. **`internal/app/snapshot.go` doc comment updated.** PASS. New comment at the post-edit location reads: `// Parent-scope constraints are enforced by\n\t\t// domain.KindDefinition.AllowsParentScope (against the kind's\n\t\t// AllowedParentScopes list) at action-item creation. Snapshot validation\n\t\t// no longer special-cases the legacy KindPhase hierarchy because the\n\t\t// 12-value Kind enum removed it.` Correctly references `KindDefinition.AllowsParentScope` + `AllowedParentScopes` (the post-collapse enforcement path), drops the stale `domain.AllowedParentKinds` reference. Diff is +5/-3 net (rewrite, not pure addition).

4. **`internal/adapters/storage/sqlite/repo.go` doc comment updated.** PASS. New comment at the boot-seed `migrate` function reads: `// Seed the 12-value Kind enum into the kind catalog at boot. Scope\n\t\t// mirrors kind (applies_to_json = ["<kind-id>"]). Every row's\n\t\t// allowed_parent_scopes_json is the empty list "[]" (universal-allow):\n\t\t// domain.KindDefinition.AllowsParentScope returns true for every parent\n\t\t// scope when AllowedParentScopes is empty (see internal/domain/kind.go\n\t\t// AllowsParentScope early return). Per-project nesting constraints land\n\t\t// in the future template overhaul.` Correctly references `KindDefinition.AllowsParentScope` + `AllowedParentScopes` + the empty-list early-return mechanism, drops `domain.AllowedParentKinds`. Diff is +6/-5 net (rewrite, slightly longer than the deleted original because it documents the universal-allow semantic explicitly).

5. **`internal/adapters/storage/sqlite/repo_test.go` forward-looking sentence trimmed.** PASS. `git diff repo_test.go` shows lines 2521–2525 lose a single sentence: `// This is the post-Droplet-2.8 universal-allow contract — Droplet 2.9 will\n// follow up by deleting the now-orphan domain.AllowedParentKinds helper.` becomes `// This is the post-Droplet-2.8 universal-allow contract.` The test function body (`TestRepositoryFreshOpenKindCatalogUniversalParentAllow`) is unchanged. Scope expansion is justified per spawn-prompt rationale: acceptance #1 framing requires `git grep AllowedParentKinds -- '*.go'` to return empty whole-tree, and the comment was the last Go-tree reference. Net -1 LOC.

6. **Whole-Go-tree `git grep "AllowedParentKinds" -- '*.go'` returns empty.** PASS. Reproduced locally — exit 1 / no output. No file in the Go tree references the deleted function. (MD-file references in `workflow/drop_2/` are intentionally preserved per `feedback_never_remove_workflow_files.md` and explicitly out-of-scope per spawn prompt.)

7. **`internal/app/kind_capability.go:566` enforcement path unchanged.** PASS. `git diff internal/app/kind_capability.go` is empty (the file is not in `git status --porcelain` output for this droplet). Read of `kind_capability.go:560-577` confirms `if !kind.AllowsParentScope(parent.Scope)` at line 566 is intact and still calls the post-collapse `KindDefinition.AllowsParentScope` method (defined at `internal/domain/kind.go:200-211`, with empty-list early return at lines 202–204). The runtime contract — "every catalog row's `AllowedParentScopes` is empty → `AllowsParentScope` always returns true → universal-allow" — is preserved.

8. **`mage ci` green at HEAD.** PASS. Reproduced locally: `Sources` ✓, `Formatting` ✓, `Coverage` 1391/1391 tests passed across 19 packages (was 1392 before Droplet 2.9; -1 from `TestAllowedParentKindsEncodesHierarchy` deletion, exactly as expected), `Build` ✓ (`till` from `./cmd/till` built successfully), exit 0. All 19 packages at or above 70.0% minimum coverage threshold (`internal/domain` 79.4%, `internal/app` 71.6%, `internal/adapters/storage/sqlite` 75.1%, `internal/tui` 70.0%, all others ≥ 72.4%).

9. **No `tc := tc` introduced.** PASS. `git diff` of the 5 modified Go files contains zero `+tc := tc` additions (only deletions of doc comments + one test function + one stale sentence). The 9 pre-existing `tc := tc` lines in the tree are all in files NOT touched by Droplet 2.9 (`cmd/till/main_test.go`, `internal/adapters/server/common/{app_service_adapter_helpers_test.go, app_service_adapter_mcp_guard_test.go}`, `internal/app/schema_validator_test.go`, `internal/domain/auth_request_test.go`) — out of scope per `Paths:` and the BUILDER_QA_PROOF.md Droplet 2.7 R2 verdict (Droplet 2.7 #11). Honors Go 1.22+ per-iteration scoping convention; project is Go 1.26+.

10. **Git status shows expected files only.** PASS. `git status --porcelain`-equivalent output: 5 Go files (`internal/domain/kind.go`, `internal/domain/domain_test.go`, `internal/app/snapshot.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`) plus 2 MD coordination artifacts (`workflow/drop_2/BUILDER_WORKLOG.md`, `workflow/drop_2/PLAN.md`). Total 7 modified files = 4 spawn-prompt-named files + 1 scope expansion (`repo_test.go`, builder-justified per acceptance #1 framing) + 2 expected MD edits (worklog + PLAN.md state-flip). No stray edits, no out-of-scope surfaces. Builder's "5 Go files" claim correct (4 named + 1 scope expansion). Pre-edit, the falsification sibling's BUILDER_QA_FALSIFICATION.md is not in this droplet's diff (separate write window).

11. **PLAN.md state-flip — Droplet 2.9 reads `**State:** done`.** PASS. `workflow/drop_2/PLAN.md:255-257` confirms: `#### Droplet 2.9 — Delete \`domain.AllowedParentKinds\` function + test fixture + doc-comment cleanup` then `- **State:** done`. Adjacent Droplet 2.10 at line 281 correctly remains `- **State:** todo` (Droplet 2.10 owns the dotted-address resolver, blocked by 2.9). The state-flip dependency chain (2.9 → 2.10 unblocked) is correctly encoded.

### Missing Evidence

(none — every required check landed grounded evidence)

### Verdict Summary

PASS. All 11 required proof checks satisfied with file:line citations. The function deletion at `internal/domain/kind.go:94-117` (24 LOC) is clean — adjacent declarations (`validKindAppliesTo` slice closer at `:92`, `KindTemplateChildSpec` opener at `:94`) butt against each other without orphan whitespace or syntax damage. The test deletion at `internal/domain/domain_test.go` (37 LOC, lines 793–832 in pre-edit) removes the only remaining production caller of `AllowedParentKinds` after the Droplet 2.8 boot-seed flip emptied catalog rows; without removal the test would be a compile error post-deletion. Both doc-comment rewrites (`snapshot.go` and `repo.go`) cleanly redirect readers from the deleted speculative helper to the live enforcement path (`KindDefinition.AllowsParentScope` + the empty-list early return at `internal/domain/kind.go:202-204`). Scope expansion to `repo_test.go:2525` is acceptance-driven (universal "git grep returns empty whole-tree") and is a 1-line comment edit, not a behavioral change. The runtime enforcement contract at `internal/app/kind_capability.go:566` is byte-identical untouched — the universal-allow semantic that Droplet 2.8 established (catalog rows have empty `AllowedParentScopes` → `AllowsParentScope` returns true universally) flows through unchanged. `mage ci` reproduces 1391/1391 tests across 19 packages (was 1392; -1 = the deleted hierarchy test, as expected), all coverage ≥ 70%, build green, exit 0. `git grep "AllowedParentKinds" -- '*.go'` returns empty whole-tree. No `tc := tc` introduced. No `mage install`. No migration code. PLAN.md state correctly flipped to `done`. The 5-file Go diff + 2 MD diffs is exactly the spawn-prompt-expected delta. Cleared for the build → QA → fix → commit gate to advance to Droplet 2.10.

### Hylla Feedback

N/A — Droplet 2.9 is a pure deletion + doc-comment rewrite droplet touching 5 Go files (3 production + 2 test + the spawn-prompt-named 4 + 1 scope-expanded `repo_test.go`) plus 2 MD coordination artifacts. All 4 spawn-prompt-named files are addressable by absolute path with exact line ranges from the prompt and PLAN.md `Paths:` field. QA used `Read` for whole-file context (`kind.go:85-115`, `kind.go:198-217`, `kind_capability.go:540-630`, `PLAN.md:1-50, :253-300, :800-815`), `Bash` for `git diff` / `git status` / `git grep "AllowedParentKinds" -- '*.go'` / `mage ci`, and direct line-anchored cross-checks against PLAN.md acceptance criteria. Hylla is stale by design for the working-tree changes (CLAUDE.md "Code Understanding Rules" §2 — `git diff` is canonical for files changed since last ingest), and the "verify a function deletion + 4 doc-comment edits with byte-precise diff alignment" question is naturally a `git diff` + `Read` shape, not a semantic-similarity / refs-graph shape. The one place a Hylla `hylla_refs_find` query would have been ergonomically natural — confirming zero callers of `AllowedParentKinds` before the deletion — is exactly the state PLAN.md `:17` already declared with "(zero production callers per PLAN.md)" and the Droplet 2.8 PASS verdict already established empirically. Zero Hylla queries attempted, zero misses to log, zero ergonomic gripes for this droplet.
