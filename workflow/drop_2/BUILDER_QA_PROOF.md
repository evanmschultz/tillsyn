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
