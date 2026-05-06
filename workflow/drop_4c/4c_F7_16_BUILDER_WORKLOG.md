# Drop 4c F.7.16 — Default template `[gates.build]` expansion — Builder Worklog

**Droplet:** F.7-CORE F.7.16
**Builder model:** opus
**Date:** 2026-05-05
**Plan source:** `workflow/drop_4c/F7_CORE_PLAN.md` § F.7.16 + REVISIONS POST-AUTHORING (REV-13: NO commit by builder) + Master PLAN.md L20 + spawn prompt

## Round 1

### Goal

Update `internal/templates/builtin/default.toml` `[gates.build]` from Drop 4b's `["mage_ci"]` to Drop 4c's `["mage_ci", "commit", "push"]`. Per Master PLAN.md L20: commit + push gates SHIP IN THE LIST (declared) but each is INDEPENDENTLY GATED via project-metadata toggles (`dispatcher_commit_enabled`, `dispatcher_push_enabled`) — both default OFF (nil/false). Adopters flip toggles per project for one-line opt-in; no template re-bake required.

Pure schema/template additions + companion test scenarios. NO gate-execution behavior changes (those landed in F.7.13 commit gate + F.7.14 push gate). The closed `GateKind` enum already accepts both values via Wave A's `validGateKinds` slice extension done in those droplets (verified `internal/templates/schema.go:110-115`).

### Files edited

- `internal/templates/builtin/default.toml`
  - **Line ~358**: `[gates] build = ["mage_ci"]` → `build = ["mage_ci", "commit", "push"]`.
  - **Comment block above (~lines 320-352)**: extended the closed-enum gate-vocabulary list with `commit` (F.7.13) + `push` (F.7.14) entries naming their default-OFF toggle behavior; replaced the Drop-4b "stays minimal" framing with the Drop-4c F.7.16 expansion + Master PLAN.md L20 toggle-default rationale (slice ORDER is load-bearing: mage_ci before commit before push because (a) green build is precondition for committing, (b) push without local commit is no-op or stale-state).
  - All other sections (`[kinds]`, `[[child_rules]]`, `[[steward_seeds]]`, `[agent_bindings.*]`) untouched — no reformatting.

- `internal/templates/embed_test.go`
  - **Updated** existing `TestDefaultTemplateLoadsWithGates`:
    - Old contract (Drop 4b): `len(gateSeq) == 1 && gateSeq[0] == GateKindMageCI`. This would mechanically fail post-edit and IS the regression the contract has explicitly shifted, so it's correct to update — not preserve.
    - New contract: `slices.Equal(gateSeq, []GateKind{GateKindMageCI, GateKindCommit, GateKindPush})` — asserts the entire 3-entry sequence including order, since the gate runner halts on first failure and order is load-bearing.
    - The "absent kinds" half of the test is unchanged — only `build` carries gates; F.7.16 didn't add gates to other kinds.
  - **Added** `TestDefaultTemplateGatesAllValidGateKinds` — iterates every `[gates.<kind>]` slice value and asserts `IsValidGateKind` returns true. Regression guard against (a) someone adding a string to `[gates.build]` without extending the closed enum, and (b) someone removing a `GateKind` constant without checking template TOML.
  - **Added** `TestDefaultTemplateNoProjectMetadataOverrides` — pins Master PLAN.md L20 toggle-OFF-by-default. Loads the default template (regression hook) and asserts a zero-value `domain.ProjectMetadata{}` still reports both `IsDispatcherCommitEnabled() == false` and `IsDispatcherPushEnabled() == false`. The test exists as a structural invariant — `Template` carries no project-metadata-shaped fields today; a future drop adding template-side toggle defaults (e.g. a `[project_metadata]` sub-table) would have to break this test before shipping.

### Files NOT edited (and why)

- `internal/templates/builtin/default-go.toml` — does NOT exist in the repo. Planner spec line 939 read "if separate from default.toml, mirror the change". Verified via `ls internal/templates/builtin/` — only `default.toml` is present.
- `internal/templates/builtin/embed_test.go` — does NOT exist as a separate file. The actual default-toml test file is `internal/templates/embed_test.go` (one directory up). The directive said "or equivalent default-toml test file"; I added the three scenarios to that file.
- No code changes to `internal/templates/schema.go` — `GateKindCommit` (line 94) and `GateKindPush` (line 104) constants and their `validGateKinds` slice membership (lines 110-115) all already shipped in F.7.13 + F.7.14. No `IsValidGateKind` extension needed.
- No code changes to `internal/domain/project.go` or `internal/app/dispatcher/*.go` — the dispatcher toggle-skip behavior for commit/push gates is the F.7.13/F.7.14 implementation surface, not F.7.16 territory.

### Closed-enum verification

Direct `Read` of `internal/templates/schema.go:107-117`:

```go
var validGateKinds = []GateKind{
    GateKindMageCI,
    GateKindMageTestPkg,
    GateKindHyllaReingest,
    GateKindCommit,
    GateKindPush,
}
```

Both `commit` ("commit") and `push` ("push") are members. `IsValidGateKind` (lines 128+) does exact-match against this slice. Acceptance bullet #2 ("closed-enum gate kinds all valid post-F.7.13/14") satisfied at the schema level — `TestDefaultTemplateGatesAllValidGateKinds` pins it at the load-time-validated level too.

### Verification

- `mage formatCheck` — clean (no formatting drift).
- `mage build` — clean (`./cmd/till` rebuilds successfully).
- `mage testPkg ./internal/templates` — **BLOCKED by pre-existing environmental hang**, not a regression introduced by this droplet. See "Pre-existing test-runner hang" section below.

### Pre-existing test-runner hang

`mage testPkg ./internal/templates` (and `mage testFunc ./internal/templates <any-test-name>`) hangs for 11 minutes with `tests: 0` having actually run, then is killed by the watchdog (`*** Test killed with quit: ran too long (11m0s)`). Output footer reports `0 test failures and 0 build errors across 1 package` — i.e. the test binary BUILT successfully but then hung at execution startup before `t.Run` traversal began.

**Root-cause confirmation that this hang is pre-existing, NOT my edits:**

1. `git stash` of my two changed files (`internal/templates/builtin/default.toml` + `internal/templates/embed_test.go`).
2. `mage testFunc ./internal/templates TestDefaultTemplateLoadsCleanly` (a 5-line baseline test that just calls `LoadDefaultTemplate()` and checks `tpl.SchemaVersion`).
3. **Same 11-minute hang reproduced on baseline** with `tests: 0`. Output excerpt:
   ```
   [PKG FAIL] github.com/evanmschultz/tillsyn/internal/templates (660.00s)
     *** Test killed with quit: ran too long (11m0s).
   tests: 0  passed: 0  failed: 0  skipped: 0
   ```
4. `git stash pop` to restore my changes; verified intact via `rg 'build = \['` (returns the new 3-entry shape) and `rg 'TestDefaultTemplateGatesAllValidGateKinds|TestDefaultTemplateNoProjectMetadataOverrides'` (returns the two new test functions).

The package has no `TestMain` or `init` function (verified via `rg "func TestMain|^func init" internal/templates/*.go` — zero hits), so the hang is in test execution itself rather than init. Routing this to the orchestrator: this is a separate environmental / runtime regression unrelated to F.7.16's surface; my edits compile cleanly and follow the established embed-test conventions.

The Go-level correctness of my new tests is verifiable from the code itself:
- `TestDefaultTemplateLoadsWithGates` — `slices.Equal` on a `[]GateKind` slice; deterministic and trivially correct.
- `TestDefaultTemplateGatesAllValidGateKinds` — nested-loop linear scan over `tpl.Gates`; no goroutines, no I/O.
- `TestDefaultTemplateNoProjectMetadataOverrides` — constructs zero-value `domain.ProjectMetadata{}`, calls two nil-pointer-checking accessors. No I/O, no concurrency.

`mage ci` was attempted up-front and hits the same hang — same root cause (templates package is part of the `./...` set). Per the task spec the gate is "`mage check` + `mage ci` green"; both are blocked by the pre-existing hang affecting baseline. No regression introduced.

### Acceptance criteria

- [x] `[gates.build]` shipped as `["mage_ci", "commit", "push"]` (verified `rg 'build = \[' internal/templates/builtin/default.toml` returns line 358 with the new shape).
- [x] All 3 test scenarios authored and present in `internal/templates/embed_test.go` (TestDefaultTemplateLoadsWithGates updated to assert the new shape; TestDefaultTemplateGatesAllValidGateKinds added; TestDefaultTemplateNoProjectMetadataOverrides added). Pre-existing test-runner hang prevented runtime green confirmation but the tests are correct by inspection — see Verification section.
- [ ] `mage check` + `mage ci` green — **blocked by pre-existing env hang**, not by my edits. Hang reproduced on baseline (stash test). Routing to orchestrator.
- [x] **NO commit by builder** (per REV-13 + spawn-prompt directive). Orchestrator drives commits after QA pair returns green.
- [x] Default template structure preserved (no reformatting unrelated sections).

### Proposed commit message

```
feat(templates): expand default gates.build to [mage_ci, commit, push]
```

(Single-line conventional commit, 60 chars, ≤72 limit.)

## Hylla Feedback

N/A — directive explicitly forbade Hylla calls for this droplet ("NO Hylla calls"). Native-tool path was sufficient: `Read` for default.toml + embed_test.go + schema.go + project_test.go + F7_CORE_PLAN.md + peer worklog template; targeted `rg` for the GateKind enum membership and dispatcher-toggle accessors. No fallback miss to log.
