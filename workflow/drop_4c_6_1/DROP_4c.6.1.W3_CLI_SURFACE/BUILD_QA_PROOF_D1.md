# W3.D1 — BUILD-QA-PROOF Verdict
**Date:** 2026-05-14
**Reviewer:** go-qa-proof-agent (opus)
**Overall verdict:** PASS WITH NITS

## Acceptance Bullet Coverage

### AC1 — `till project update --project-id <id> --root-path /abs/path` updates `RepoPrimaryWorktree` and returns the updated project on stdout
- **Implementation:** `cmd/till/project_cli.go:200-202` merges `opts.rootPath` into `repoPrimaryWorktree`; line 229-241 passes it through to `svc.UpdateProject`.
- **Test:** `cmd/till/project_cli_test.go:541-547` — `TestRunProjectUpdate_UpdatesFirstClassFields/root-path update` asserts no error + "Updated Project" in output.
- **Verdict:** PASS. Note: `writeProjectDetail` does not surface `RepoPrimaryWorktree` directly in the K-V output, so the test only verifies success rather than the new value. Acceptable — the merge path is exercised, and the field is value-typed (no clobber risk by construction).

### AC2 — `--bare-root /abs/path` updates `RepoBareRoot`
- **Implementation:** `cmd/till/project_cli.go:203-205`.
- **Test:** `cmd/till/project_cli_test.go:550-556` — `bare-root update` sub-case.
- **Verdict:** PASS. Same `writeProjectDetail` caveat as AC1.

### AC3 — `--language go` updates `Language`; `--language invalid` fails with `ErrInvalidLanguage`
- **Implementation:** `cmd/till/project_cli.go:206-208` merges; the closed-enum check happens inside `domain/project.go:341-343` (`isValidProjectLanguage` → `ErrInvalidLanguage = "invalid language"` per `internal/domain/errors.go:33`).
- **Test:** `cmd/till/project_cli_test.go:609-655` — `TestRunProjectUpdate_LanguageValidation` with 4 sub-cases (invalid, go, fe, empty); invalid asserts `"invalid language"` substring; go/fe/empty assert no error.
- **Verdict:** PASS. Closed enum `"" | "go" | "fe"` matches `domain/project.go:293-300`.

### AC4 — `--hylla-artifact-ref github.com/org/repo@main` updates `HyllaArtifactRef`
- **Implementation:** `cmd/till/project_cli.go:209-211`.
- **Test:** `cmd/till/project_cli_test.go:582-588` — `hylla-artifact-ref update` sub-case.
- **Verdict:** PASS.

### AC5 — `--description "new desc"` updates `Description`
- **Implementation:** `cmd/till/project_cli.go:197-199`.
- **Test:** `cmd/till/project_cli_test.go:557-564` — `description update` sub-case asserts `"new desc"` appears in K-V output (description is surfaced by `writeProjectDetail` line 426).
- **Verdict:** PASS. Only field whose post-update value is directly asserted in output.

### AC6 — `--build-tool mage` updates `BuildTool`
- **Implementation:** `cmd/till/project_cli.go:212-214`.
- **Test:** `cmd/till/project_cli_test.go:566-572` — `build-tool update` sub-case.
- **Verdict:** PASS.

### AC7 — `--dev-mcp-server-name tillsyn-dev` updates `DevMcpServerName`
- **Implementation:** `cmd/till/project_cli.go:215-217`.
- **Test:** `cmd/till/project_cli_test.go:573-579` — `dev-mcp-server-name update` sub-case.
- **Verdict:** PASS.

### AC8 — `--owner "Evan"` updates `Metadata.Owner`
- **Implementation:** `cmd/till/project_cli.go:221-223` merges into `metadata.Owner` after `metadata := existing.Metadata` preserves all other metadata fields.
- **Test:** `cmd/till/project_cli_test.go:659-672` — `TestRunProjectUpdate_OwnerMetadata` asserts `"Evan"` in output (owner is surfaced by `writeProjectDetail` line 420).
- **Verdict:** PASS.

### AC9 — `--add-group fe` appends to `Metadata.Groups`; dedup no-op if already present
- **Implementation:** `cmd/till/project_cli.go:225-227` calls `applyGroupMutations`; lines 250-265 implement linear-scan dedup that only appends when not `found`.
- **Test:** `cmd/till/project_cli_test.go:677-720` — `TestRunProjectUpdate_AddGroupAppendsAndDeduplicates` adds `"go"` twice, reads back via `svc.ListProjects`, counts occurrences of `"go"` in `Metadata.Groups`, asserts exactly 1.
- **Verdict:** PASS. Real read-back assertion (not just stdout match).

### AC10 — `--remove-group go` removes from `Metadata.Groups`; no-op if not present
- **Implementation:** `cmd/till/project_cli.go` lines 266-282 — `removeSet` map filter; absent value silently skipped, no error path.
- **Test:** `cmd/till/project_cli_test.go:724-777` — `TestRunProjectUpdate_RemoveGroupFiltersAndIsNoopWhenAbsent` seeds `[go,fe]`, removes `go`, asserts `go` absent and `fe` remains; then removes absent `gen` and asserts no error.
- **Verdict:** PASS. Both present-remove and absent-remove paths exercised.

### AC11 — `--add-group` and `--remove-group` may be repeated
- **Implementation:** struct fields `addGroups []string` and `removeGroups []string` (`cmd/till/main.go:289-290`); `applyGroupMutations` iterates both slices.
- **Test:** `cmd/till/project_cli_test.go:800-833` — `TestRunProjectUpdate_MultipleAddRemoveGroups` supplies `[]string{"go", "fe"}` in one call, asserts both present; then removes `go` and asserts only `fe` remains.
- **Verdict:** PASS.

### AC12 — Missing `--project-id` fails with the canonical project-discovery error
- **Implementation:** `cmd/till/project_cli.go:165-167` calls `requireProjectID("project update", opts.projectID)`; the helper returns `projectDiscoveryError` (lines 17-20) which embeds `"--project-id is required"`, `"till project list"`, and `"till project discover"`.
- **Test:** `cmd/till/project_cli_test.go:513-525` — `TestRunProjectUpdate_MissingProjectIDReturnsDiscoveryError` asserts all three substrings.
- **Verdict:** PASS.

### AC13 — `mage test-pkg ./cmd/till/...` passes
- **Live verification:**
  - `mage testPkg ./cmd/till` → 351/351 PASS, cmd/till package PASS in 0.10s.
  - `mage testFunc ./cmd/till 'TestRunProjectUpdate.*'` → 18 tests PASS.
  - `mage ci` → 3236/3236 PASS, cmd/till coverage 77.2% (above 70.0% threshold), build clean.
- **Verdict:** PASS.

## Special-Focus Invariants

### Read-then-merge correctness (no zero-value clobber)
- **Implementation:** `project_cli.go:178` reads existing via `locateProjectForCLI` BEFORE any merge; lines 188-219 default each first-class field to `existing.X` then conditionally overwrite only when `strings.TrimSpace(opts.X) != ""`; `metadata := existing.Metadata` (line 220) preserves ALL nested metadata fields (Icon, Color, Homepage, Tags, StandardsMarkdown, etc.); `applyGroupMutations` deep-copies the existing groups slice (line 249) before mutation.
- **Verdict:** PASS by code inspection. Test rigor on this invariant is partial — see NIT N1.

### --add-group dedup
- **Implementation:** `project_cli.go:255-264` — linear scan against running `groups` slice (including prior additions in the same call).
- **Verdict:** PASS. Test `TestRunProjectUpdate_AddGroupAppendsAndDeduplicates` reads back and counts.

### --remove-group no-op on absent
- **Implementation:** `project_cli.go:266-282` — `removeSet` lookup returns `skip=false` for absent keys, value is preserved in `filtered`.
- **Verdict:** PASS. Test exercises both present-remove and absent-remove.

### --language invalid → ErrInvalidLanguage clear error
- **Implementation:** delegated to `domain.Project.UpdateDetails` (`internal/domain/project.go:341-343`); error message `"invalid language"` (`internal/domain/errors.go:33`).
- **Verdict:** PASS. Test asserts substring match.

### --add-group rejects unknown groups
- **Implementation:** `project_cli.go:171-176` validates EVERY `--add-group` value against `allowedInitGroups` (`cmd/till/init_cmd.go:63` → `{"gen", "go", "fe"}`) BEFORE the project read, returning a clear error that includes the offending value and the allowed set.
- **Test:** `cmd/till/project_cli_test.go:781-796` — `TestRunProjectUpdate_AddGroupRejectsUnknownGroup` with `"invalid-group"`, asserts the offending value appears in the error.
- **Verdict:** PASS.

## NITs

### N1 — Per-field non-clobber assertions only indirectly verified (severity: low)
- **Axis:** acceptance-criteria-coverage
- **Claim:** Tests for `--root-path`, `--bare-root`, `--language`, `--hylla-artifact-ref`, `--build-tool`, `--dev-mcp-server-name` only assert `"Updated Project"` in output — they do NOT seed multiple fields then read back to confirm OTHER fields are preserved across the update.
- **Evidence:** `cmd/till/project_cli_test.go:529-605` cases in `TestRunProjectUpdate_UpdatesFirstClassFields`.
- **Impact:** Implementation IS correct (the merge code path at `project_cli.go:188-219` preserves existing values), but the test rigor on the "no clobber" invariant is partial. Groups field has the strongest read-back coverage (counts after dedup/filter); description is asserted positively in output; the remaining fields have only success-signal assertions.
- **Fix hint:** Add one assertion per sub-case reading back via `svc.ListProjects` and verifying name + the unchanged fields equal the seeded baseline. Recommended for D1 round-2 if dev wants tighter guarantees; otherwise leave as-is (NIT, not blocker).

### N2 — `--icon`, `--color`, `--homepage`, `--tags` not present in struct, but listed in D1 KindPayload and required by W3.D7 (severity: medium)
- **Axis:** spec-conformance / shipped-but-not-wired
- **Claim:** D1 KindPayload shape_hint (line 273) lists `icon string; color string; homepage string; tags []string` as struct fields; W3.D7 AcceptanceCriteria (line 697) expects `--icon, --color, --homepage, --tags` flags on `till project update --help`. The shipped `projectUpdateCommandOptions` struct (`cmd/till/main.go:279-291`) does NOT include these fields, and `runProjectUpdate` does NOT plumb them through to `Metadata`.
- **Evidence:** `cmd/till/main.go:279-291` (struct fields); `cmd/till/project_cli.go:154-244` (runProjectUpdate body); D1 spec AcceptanceCriteria (PLAN.md:219-232) does NOT list these flags; D1 KindPayload (PLAN.md:273) lists them; W3.D7 AcceptanceCriteria (PLAN.md:697) expects them in `--help`.
- **Impact:** Builder correctly followed D1 AcceptanceCriteria bullets, which is the contract. However, W3.D7 will either (a) need to add these four flag fields to the struct (extending D1's surface from D7's side) or (b) drop the four flags from D7's `till project update` wiring. The cleanest disposition is to add the fields now during D1 round-2 to avoid D7-time struct churn.
- **Fix hint:** Either extend `projectUpdateCommandOptions` with `icon, color, homepage string; tags []string` plus the corresponding merge lines into `metadata.Icon/Color/Homepage/Tags`, OR document explicitly that D7 will not wire those flags and update D7's spec accordingly. Surfaced for orchestrator routing.

### N3 — `gen` group not exercised in tests (severity: low)
- **Axis:** acceptance-criteria-coverage
- **Claim:** `allowedInitGroups = {"gen", "go", "fe"}` (`cmd/till/init_cmd.go:63`); tests exercise `go` and `fe` adds but never `gen`.
- **Evidence:** `cmd/till/project_cli_test.go` ranges 677-833 — only `"go"` and `"fe"` literals in addGroups; `"invalid-group"` for the reject test.
- **Fix hint:** Add `"gen"` to one of the add-group cases (e.g. the multi-add test) for full enum coverage. NIT — `isAllowedProjectGroup` iterates the list so coverage is structurally adequate; this is belt-and-braces.

### N4 — Spec wording says "JSON blob on stdout" but implementation prints K-V text (severity: low)
- **Axis:** spec-conformance
- **Claim:** D1 AcceptanceCriteria bullet 1 (PLAN.md:220) says "returns the updated project as a JSON blob on stdout"; implementation uses `writeProjectDetail` (`project_cli.go:415-430`) which emits a K-V text table.
- **Evidence:** spec line 220 vs `project_cli.go:243` calling `writeProjectDetail(stdout, project, "Updated Project")`.
- **Impact:** The spec also explicitly says "Existing pattern to follow: runProjectCreate (project_cli.go:133)" which uses K-V text. Builder followed the existing pattern. NIT against spec wording, not implementation. No functional impact.
- **Fix hint:** Update spec phrasing to "human-readable detail view" or accept the drift as documentation-only. No code change needed.

### N5 — `Kind` field on `UpdateProjectInput` is dead/unread (severity: trivial — pre-existing, not D1's responsibility)
- **Axis:** spec-conformance (informational only)
- **Claim:** `internal/app/service.go:859` declares `Kind domain.KindID` on `UpdateProjectInput`; `Service.UpdateProject` (lines 872-913) never reads it.
- **Evidence:** grep of `in.Kind` in `service.go` returns no read.
- **Impact:** Pre-existing dead field. Not part of D1's scope. Recorded for a future refinement.

## Verdict Rationale

All 13 AcceptanceCriteria bullets are backed by implementation + at least one test path. Special-focus invariants (read-then-merge, dedup, no-op remove, language validation, unknown-group reject) each have explicit test coverage with read-back assertions on the group-mutation paths.

The implementation correctly:
- Reads existing project BEFORE any merge (`locateProjectForCLI` at line 178).
- Defaults every first-class field to the existing value (lines 188-195).
- Only overwrites when the flag value is non-empty after `strings.TrimSpace` (lines 197-218).
- Preserves the full `Metadata` struct including all nested fields (`metadata := existing.Metadata` at line 220).
- Deep-copies the existing groups slice before mutating (`applyGroupMutations` line 249).
- Validates `--add-group` against `allowedInitGroups` BEFORE the project read (lines 171-176), guaranteeing failure-fast without any side effects.
- Wires `cliMutationContext` AFTER the read so the audit-actor stamping applies only to the write (line 184).

Live verification: `mage ci` green (3236/3236, cmd/till 77.2% coverage); `mage testPkg ./cmd/till` 351/351 PASS; `mage testFunc ./cmd/till 'TestRunProjectUpdate.*'` 18/18 PASS.

Five NITs surface: (N1) per-field non-clobber test rigor is partial, (N2) icon/color/homepage/tags fields listed in KindPayload and required by W3.D7 but absent from D1's struct, (N3) `gen` group not exercised, (N4) spec wording "JSON blob" drift, (N5) pre-existing `UpdateProjectInput.Kind` dead field. N2 is the only one that could become a blocker at D7-time; N1 and N3 are test-rigor recommendations; N4 and N5 are documentation/refinement items.

**Overall verdict: PASS WITH NITS.**
