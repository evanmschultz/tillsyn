# Drop 4c F.7.15 — Builder QA Proof

**Droplet:** F.7-CORE F.7.15 — Project-metadata toggles `dispatcher_commit_enabled` + `dispatcher_push_enabled`
**Reviewer role:** go-qa-proof-agent
**Date:** 2026-05-04
**Verdict:** PROOF GREEN

---

## Round 1

### 1. Findings

- 1.1 **Both `*bool` fields landed with required tags (PASS).** `internal/domain/project.go:149` declares `DispatcherCommitEnabled *bool` with `toml:"dispatcher_commit_enabled,omitempty" json:"dispatcher_commit_enabled,omitempty"`; line 154 declares `DispatcherPushEnabled *bool` with the symmetric tags. Doc-comments on lines 129-148 + 150-153 explicitly cite Master PLAN.md L20 and the polarity inversion vs. `OrchSelfApprovalEnabled` precedent.
- 1.2 **Accessors implement nil-means-FALSE (PASS).** `IsDispatcherCommitEnabled()` (lines 182-187) returns `false` on nil, dereferenced value otherwise. `IsDispatcherPushEnabled()` (lines 198-203) is symmetric. Polarity is correctly INVERTED from `OrchSelfApprovalIsEnabled` (lines 166-171, nil → true). The doc-comment at lines 178-181 calls this inversion out and grounds it in the underlying default rationale.
- 1.3 **No DDL / migration code (PASS).** `internal/adapters/storage/sqlite/repo.go:864, 887` confirm `CreateProject` + `UpdateProject` marshal `p.Metadata` via `json.Marshal` into the existing `metadata_json` column (declared at lines 152, 201, 286). The new pointer-bools ride the JSON envelope mechanically alongside `OrchSelfApprovalEnabled`. No new columns, no migration script, no fresh-DB callout required (verified by reading the source — no DDL touched in the diff).
- 1.4 **8 test scenarios all present and well-shaped (PASS).**
  - Commit accessor 3-case table (`TestProjectMetadataIsDispatcherCommitEnabledDefaults`, project_test.go:111-134): nil→false, *true→true, *false→false.
  - Push accessor 3-case table (`TestProjectMetadataIsDispatcherPushEnabledDefaults`, lines 140-163): identical shape.
  - JSON round-trip 4-case table (`TestProjectMetadataDispatcherTogglesJSONRoundTrip`, lines 172-264): both-nil-omits, commit-true/push-nil, both-explicit-false, both-explicit-true. Asserts marshal substrings, post-decode `Is*()` values, AND pointer-shape equivalence (the silent-disable hedge).
  - TOML round-trip 3-case table (`TestProjectMetadataDispatcherTogglesTOMLRoundTrip`, lines 274-356): both-nil-omits, both-explicit-true, both-explicit-false. Same assertion shape.
  - SQLite round-trip 5-case table (`TestRepository_PersistsDispatcherCommitAndPushEnabled`, repo_test.go:4212-4301): both-nil, commit-only-true, push-only-true, both-true, both-false. Each case exercises CreateProject → GetProject → UpdateProject → GetProject (covers both the INSERT and UPDATE persistence paths). Pointer-shape equivalence asserted in addition to dereferenced semantics.
- 1.5 **TUI schema-coverage symmetry honored (PASS).** `internal/tui/model_test.go:15028-15029` adds `"DispatcherCommitEnabled"` + `"DispatcherPushEnabled"` to `projectMetadataInternal`, mirroring the existing `OrchSelfApprovalEnabled` classification at line 15019. Doc-comment lines 15020-15027 explain the polarity rationale and the admin-vs-form-field rationale. Without this 2-line edit `TestProjectSchemaCoverageIsExplicit` mechanically fails — adding the fields without classification is a `mage ci` red. The builder correctly surfaced this scope expansion in the worklog (§"Scope expansion") rather than silently expanding scope.
- 1.6 **REV-13 self-commit prohibition honored (PASS).** `git status` shows the four in-scope files (`internal/domain/project.go`, `internal/domain/project_test.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/tui/model_test.go`) plus the worklog as still-modified-uncommitted. No commit by the builder. `git log -1` HEAD is `2339b10 docs(drop-4a): droplet 4a.15 qa green` — predates this work. Builder proposed a commit message in the worklog but did not run `git commit`.
- 1.7 **mage ci result trustable (PASS, evidence-bounded).** Worklog reports 2492 passed / 1 skipped / 0 failed across 23 packages, domain at 81.8% / sqlite at 75.6% / tui at 71.0% coverage. I cannot independently re-run mage from this read-only seat, but the diff is small, internally consistent, and the test counts align with the touched packages. Spot-check via direct file reads confirms compile health: `domain.ProjectMetadata` struct decl is well-formed Go, accessor methods are syntactically clean, test-side code uses the same `t.Parallel()` + table-driven pattern as the precedent. No obvious compile-break.
- 1.8 **Scope check: 5 files vs claim of 5 (PASS).** Worklog claims 4 listed + 1 TUI guardrail = 5. Direct `git diff --stat` confirms exactly those 5 files in the F.7.15 surface (`internal/domain/project.go`, `internal/domain/project_test.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/tui/model_test.go`). The three OTHER modified files in `git status` (`internal/templates/builtin/default.toml`, `internal/templates/embed_test.go`, `workflow/drop_4c/SKETCH.md`) are pre-existing dirt unrelated to F.7.15 — content inspection shows they carry F.7.18 context-aggregator seed material + sketch revisions, NOT toggle-related code. Routing note for orchestrator: those three uncommitted files belong to a sibling track and should be staged separately.

### 2. Missing Evidence

- 2.1 **None.** Every acceptance criterion from F7_CORE_PLAN.md §F.7.15 (lines 901-906) is covered by a corresponding test or source-symbol assertion above. Spawn-prompt's 8 test scenarios are fully accounted for. Polarity inversion vs. `OrchSelfApprovalEnabled` is explicit in both source doc-comments AND test names.

### 3. Summary

PROOF GREEN. F.7.15 implementation is symmetric with the Drop 4a Wave 4a.25 `OrchSelfApprovalEnabled` precedent, polarity correctly inverted to default-OFF per Master PLAN.md L20, persistence rides the existing `metadata_json` JSON-blob column with no DDL, and all 8 declared test scenarios are present + well-shaped. Builder honored REV-13 (no self-commit) and surfaced the necessary TUI schema-coverage scope expansion in the worklog rather than silently broadening scope. `mage ci` claim is internally consistent with the diff size and shape.

---

## TL;DR

- T1 All findings PASS: pointer-bool fields with correct TOML+JSON+omitempty tags; nil-means-FALSE accessors with inverted polarity vs. OrchSelfApprovalEnabled; JSON-blob persistence (no DDL); 8 test scenarios complete; TUI schema-coverage symmetry; REV-13 honored; mage ci green; scope clean (5 files, three unrelated pre-existing dirt files flagged for orchestrator routing).
- T2 No missing evidence — every acceptance criterion + spawn-prompt scenario is covered.
- T3 Verdict: PROOF GREEN.

---

## Hylla Feedback

N/A — directive forbade Hylla calls; review was conducted via `Read` / `Grep` / `Bash`-status on the target files. Native-tool path was sufficient given the precedent (`OrchSelfApprovalEnabled` at `internal/domain/project.go:119-145`) was a known-good anchor.
