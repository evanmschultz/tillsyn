# Drop 4c F.7.16 — Builder QA Proof Review

**Droplet:** F.7-CORE F.7.16 (Default template `[gates.build]` expansion)
**Reviewer model:** opus
**Review date:** 2026-05-05
**Build artefacts under review:**
- `internal/templates/builtin/default.toml` — `[gates.build]` updated from Drop 4b's `["mage_ci"]` to `["mage_ci", "commit", "push"]` plus surrounding comment block expanded with toggle-default rationale.
- `internal/templates/embed_test.go` — `TestDefaultTemplateLoadsWithGates` updated; two new tests `TestDefaultTemplateGatesAllValidGateKinds`, `TestDefaultTemplateNoProjectMetadataOverrides`.
- `workflow/drop_4c/4c_F7_16_BUILDER_WORKLOG.md` (worklog).

## Round 1

### 1. Findings

#### 1.1 Verdict — PASS

The builder's claim is supported by the evidence on disk. All five spawn-prompt verification points hold:

1. **`[gates.build]` exact shape.** `default.toml:358` reads `build = ["mage_ci", "commit", "push"]` — exact 3-entry slice in the load-bearing order asserted by the diff against `HEAD` (`git diff HEAD internal/templates/builtin/default.toml`). No other `[gates.<kind>]` rows present. Slice order is the documented invariant: `mage_ci` precedes `commit` (green build precondition for committing) and `commit` precedes `push` (push without local commit is no-op or stale-state).
2. **Closed-enum gate-kind membership.** `internal/templates/schema.go:110-116` defines `validGateKinds` as `{GateKindMageCI, GateKindMageTestPkg, GateKindHyllaReingest, GateKindCommit, GateKindPush}`. `IsValidGateKind` (line 128) does exact-match against that slice. All three entries (`"mage_ci"`, `"commit"`, `"push"`) are members. `GateKindCommit = "commit"` (schema.go:94) and `GateKindPush = "push"` (schema.go:104) ship pre-F.7.16 from F.7.13 + F.7.14. The new test `TestDefaultTemplateGatesAllValidGateKinds` (embed_test.go:412-423) iterates every entry in every `tpl.Gates[<kind>]` slice and rejects any value `IsValidGateKind` flags as invalid — pins the closed-enum invariant at template load time.
3. **No project-metadata default overrides.** `internal/domain/project.go:182-187` (`IsDispatcherCommitEnabled`) and `:198-202` (`IsDispatcherPushEnabled`) both return `false` when their respective `*bool` field is `nil`. Zero-value `domain.ProjectMetadata{}` therefore reports both toggles disabled — Master PLAN.md L20's "default OFF until dogfood proves them safe" contract. The new test `TestDefaultTemplateNoProjectMetadataOverrides` (embed_test.go:438-455) loads the default template (the regression hook) and then asserts both `IsDispatcherCommitEnabled()` and `IsDispatcherPushEnabled()` remain `false` on a fresh zero-value `ProjectMetadata`. The `Template` struct itself carries no project-metadata-shaped fields (verified by inspection of schema.go); a future drop adding template-side toggle defaults would have to break this test before shipping. Scope is tight: the test does not assert *all* possible mutation paths, just the structural-invariant entry point — appropriate for the F.7.16 surface.
4. **Existing tests still aligned.** `TestDefaultTemplateLoadsWithGates` (embed_test.go:361-395) updated in place: the old `len == 1 && [0] == GateKindMageCI` assertion would mechanically fail post-edit and IS the contract that explicitly shifted; updating to `slices.Equal(gateSeq, []GateKind{GateKindMageCI, GateKindCommit, GateKindPush})` is correct, not a regression. The "absent kinds" half of the test is preserved verbatim — eleven sibling kinds (`plan`, `research`, `build-qa-proof`, `build-qa-falsification`, `plan-qa-proof`, `plan-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`) still asserted absent from `tpl.Gates`. The doc-comment was also rewritten to capture the F.7.16 expansion + toggle-default rationale; comment drift would otherwise have been a soft finding. None of the other 16 tests in `embed_test.go` touch `tpl.Gates` so they are insulated from this change.
5. **No commit by builder.** `git status --porcelain internal/` reports both files in unstaged-modified state (` M`). `git log --oneline -3` for the same paths shows the latest commits touching them are pre-existing (`5a195a7`, `af51dec`, `5ebfc55`, `198259f`) — no new commit landed in this droplet's window. REV-13 honored.

#### 1.2 Comment-block rationale matches code reality

The expanded comment block above `[gates]` (default.toml:320-356) names the toggle field paths (`ProjectMetadata.DispatcherCommitEnabled`, `ProjectMetadata.DispatcherPushEnabled`) and the TOML key paths (`dispatcher_commit_enabled`, `dispatcher_push_enabled`). Cross-checked against `internal/domain/project.go:149,154`: the TOML tags match exactly (`toml:"dispatcher_commit_enabled,omitempty"`, `toml:"dispatcher_push_enabled,omitempty"`). No drift between code and template-side prose.

#### 1.3 Doc-comment claim "Skipped at runtime unless toggle = &true"

The comment-block bullets state the runtime-skip behavior is conditioned on the toggle being explicitly `&true`. This is consistent with the accessor implementations: `IsDispatcherCommitEnabled` returns `true` ONLY when the pointer is non-nil AND dereferences to `true` (project.go:182-187). Both `nil` and `&false` collapse to disabled. The runtime gate-skip implementation lives in F.7.13 / F.7.14 surfaces and is NOT F.7.16 territory; the template-side prose accurately describes the behavior the prior droplets shipped.

### 2. Missing Evidence

#### 2.1 Local `mage ci` runtime green for the new tests

The orchestrator-shell `mage ci` exit-0 captured in the spawn prompt is the authoritative gate; the worklog (lines 56-83) reports a pre-existing 11-minute environment hang inside `mage testFunc ./internal/templates …` even on a stash-baseline. I attempted `mage testFunc ./internal/templates TestDefaultTemplateLoadsWithGates` in the QA shell as an independent re-confirmation and it hung in the same fashion (background task `br6xs0m9f` produced 0 bytes within the QA review window). This is not a regression introduced by F.7.16:
- The worklog reproduces the hang on a stash-baseline (no F.7.16 changes present), pinning it as environmental.
- The orchestrator's own `mage ci` exit-0 from a different shell is documented in the spawn prompt — that's the verified-green data point.
- The new tests are correct by inspection: `TestDefaultTemplateLoadsWithGates` is `slices.Equal` over a 3-entry `[]GateKind`; `TestDefaultTemplateGatesAllValidGateKinds` is a nested linear scan; `TestDefaultTemplateNoProjectMetadataOverrides` constructs a zero-value struct + calls two nil-checking accessors. No goroutines, no I/O, no concurrency — deterministic.

The missing-evidence slot is logged for the audit trail but does not block PASS — orchestrator-shell `mage ci` cleared the gate.

### 3. Summary

**PASS.** All five spawn-prompt verification points are supported by direct evidence in `default.toml`, `embed_test.go`, `schema.go`, `project.go`, and the `git diff` / `git status` output. The expanded `[gates.build]` slice, closed-enum gate-kind membership, project-metadata default-OFF invariant, existing-test alignment, and no-commit posture all check out. The orchestrator's `mage ci` exit-0 is the binding green; the QA-shell test-runner hang is a pre-existing environmental issue, not a regression — independently corroborated by stash-baseline reproduction in the worklog.

## Hylla Feedback

N/A — review touched non-Go files (TOML template) plus Go test additions and surrounding accessors that were already cited in the spawn prompt by file path. Direct `Read` of the cited files plus `git diff` / `git status` was sufficient. No Hylla queries attempted.
