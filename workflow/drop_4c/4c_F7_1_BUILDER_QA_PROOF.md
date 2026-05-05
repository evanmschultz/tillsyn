# Drop 4c F.7.1 — Per-spawn temp bundle lifecycle (QA Proof)

**Verdict:** GREEN-WITH-NITS

The 10 acceptance criteria pass with file:line evidence. The proof bar (evidence supports the claim) is met for every named contract — schema extension, validator, Bundle/NewBundle/Cleanup/WriteManifest, two materialization modes, manifest-payload shape, spawn.go integration, test surface, mage ci, no commit. The NIT is a real **scope-creep finding** — the diff bundles F.7.6 work (RequiresPlugins schema field, validator, plugin pre-flight call site, plus untracked `plugin_preflight.go` + `plugin_preflight_test.go` + SKETCH.md edits) that is explicitly out-of-scope per `F7_CORE_PLAN.md` REV-7. The F.7.1 contract itself is intact and proven; the falsification sibling will own attacking the conclusion + the scope question.

## 1. Per-criterion evidence

### 1.1 C1 — `Tillsyn.SpawnTempRoot string` extension with TOML tag `spawn_temp_root`

- `internal/templates/schema.go:258` — `SpawnTempRoot string \`toml:"spawn_temp_root"\`` field declaration on the `Tillsyn` struct.
- `internal/templates/schema.go:240-257` — doc comment cites the closed enum `{"", "os_tmp", "project"}` and consumer-time-default convention (`""` resolves to `"os_tmp"` at spawn time).
- `internal/templates/schema_test.go:126-130` — round-trip test populates `SpawnTempRoot: "project"` so the TOML tag round-trips symmetrically with the existing two `Tillsyn` fields.

PASS.

### 1.2 C2 — `validateTillsyn` rejects values outside `{"", "os_tmp", "project"}` with structured error

- `internal/templates/load.go:761-779` — `validTillsynSpawnTempRootValues` slice + `isValidTillsynSpawnTempRoot` exact-match helper (no whitespace trim, no case fold — mirrors the IsValidGateKind / isValidContextDelivery pattern).
- `internal/templates/load.go:819-822` — `validateTillsyn` membership check returns `ErrInvalidTillsynGlobals` wrapping the offending value verbatim plus the closed-enum trio in the message.
- `internal/templates/load.go:256-273` — `ErrInvalidTillsynGlobals` doc-comment updated to enumerate the SpawnTempRoot rule alongside the existing two.
- `internal/templates/load_test.go:1207-1305` (approx) — `TestLoadTillsynSpawnTempRootHappyPath` (table: `os_tmp`, `project`), `TestLoadTillsynSpawnTempRootOmittedDefaultsToEmpty` (consumer-time-default sentinel), `TestLoadTillsynSpawnTempRootRejectsBogusValue` (5-row reject table: bogus, upper, capitalized, whitespace-padded, hyphen-vs-underscore) — all three test functions added by F.7.1.

PASS.

### 1.3 C3 — `Bundle` + `NewBundle(item, spawnTempRoot, projectRoot)` + `Cleanup` + `WriteManifest` shipped in `bundle.go`

All in `internal/app/dispatcher/bundle.go`:

- `bundle.go:50-54` — `SpawnTempRootOSTmp` / `SpawnTempRootProject` constants (closed-enum companion to `templates.Tillsyn.SpawnTempRoot`).
- `bundle.go:64` — `ErrInvalidBundleInput` sentinel (distinct from `ErrInvalidSpawnInput` per the worklog rationale — runtime bundle-lifecycle vs template-load disambiguation).
- `bundle.go:76-98` — `Bundle` struct with `SpawnID`, `Mode`, `StartedAt`, `Paths` (BundlePaths handle).
- `bundle.go:109-132` — `ManifestMetadata` struct (see C5 for shape).
- `bundle.go:157-190` — `NewBundle(item domain.ActionItem, spawnTempRoot string, projectRoot string) (Bundle, error)` matches the prompt's signature exactly.
- `bundle.go:264-269` — `Bundle.Cleanup() error` idempotent reaper, zero-value-safe (early return when `Paths.Root == ""`).
- `bundle.go:287-299` — `Bundle.WriteManifest(ManifestMetadata) error` with `json.MarshalIndent` (2-space) + `os.WriteFile` 0o600 perms; defensive guard on empty `ManifestPath`.

PASS.

### 1.4 C4 — Two bundle modes resolve correctly

- `bundle.go:202-212` — `resolveSpawnTempRoot(string) (string, error)` switch maps `""` and `"os_tmp"` → `SpawnTempRootOSTmp`; `"project"` → `SpawnTempRootProject`; default → `ErrInvalidBundleInput` wrap.
- `bundle.go:223-249` — `materializeBundleRoot(mode, projectRoot, spawnID)`:
  - `os_tmp` mode (lines 225-230): `os.MkdirTemp("", "tillsyn-spawn-")` — uses `os.TempDir()` via the empty `dir` param convention; matches "os.TempDir()/tillsyn-spawn-<id>/".
  - `project` mode (lines 231-243): `filepath.Join(projectRoot, ".tillsyn", "spawns", spawnID)` + `os.MkdirAll(root, 0o700)`; exits early with `ErrInvalidBundleInput` when `projectRoot` is empty/whitespace.
- `bundle_test.go:31-71` (`TestNewBundleOSTempMode`) — pins os_tmp resolution: empty string → `Mode == SpawnTempRootOSTmp`, root under `os.TempDir()`, basename has `tillsyn-spawn-` prefix, directory exists on disk.
- `bundle_test.go:96-126` (`TestNewBundleProjectMode`) — pins project mode: `<projectRoot>/.tillsyn/spawns/<spawnID>` layout, idempotent parent-dir creation, root is a directory.
- `bundle_test.go:131-144` (`TestNewBundleProjectModeRequiresProjectRoot`) — pins the project-mode-needs-projectRoot guard; empty projectRoot → `ErrInvalidBundleInput`.

PASS.

### 1.5 C5 — Manifest payload includes spawn_id, action_item_id, kind, started_at, paths; omits cli_kind + claude_pid

- `bundle.go:109-132` — `ManifestMetadata` struct fields with JSON tags:
  - `SpawnID string \`json:"spawn_id"\`` (line 111)
  - `ActionItemID string \`json:"action_item_id"\`` (line 116)
  - `Kind domain.Kind \`json:"kind"\`` (line 121)
  - `StartedAt time.Time \`json:"started_at"\`` (line 125)
  - `Paths []string \`json:"paths"\`` (line 131)
- `bundle.go:104-108` — explicit doc-comment confirming **NO `CLIKind`** (REV-4; F.7.17.6 owns) and **NO `ClaudePID`** (F.7.8 owns via separate `UpdateManifestPID`).
- `bundle_test.go:330-366` (`TestBundleWriteManifestKeysExactShape`) — round-trips through `map[string]any` and asserts the exact 5-key wire format with no extras.
- `bundle_test.go:272-324` (`TestBundleWriteManifestRoundTrip`) — full marshal/unmarshal symmetry with `time.Time.Equal` for nanosecond-drift safety.

PASS.

### 1.6 C6 — `spawn.go` `TODO(F.7.1)` block replaced with NewBundle + WriteManifest + cleanup-on-failure

- `internal/app/dispatcher/spawn.go:253-288` — replacement block:
  - Line 273: `bundle, err := NewBundle(item, "", project.RepoPrimaryWorktree)` — empty-string sentinel resolves to os_tmp mode; projectRoot threaded unconditionally so the project-mode codepath is wired for the eventual catalog-plumbing follow-up.
  - Lines 277-287: `bundle.WriteManifest(ManifestMetadata{SpawnID, ActionItemID, Kind, StartedAt, Paths})` followed by `_ = bundle.Cleanup()` on error and `return` with the wrapped failure.
- `spawn.go:288` — `bundlePaths := bundle.Paths` replaces the inline BundlePaths literal of the prior implementation.
- `spawn.go:296-299` — system-prompt write path also calls `_ = bundle.Cleanup()` on failure (per the worklog's bundle-leak prevention claim).
- `spawn.go:160-165` — step-5 doc-comment in `BuildSpawnCommand` updated to cite `NewBundle` materializer; the `TODO(F.7.1)` marker confirmed removed (no remaining `TODO(F.7.1)` references in spawn.go).
- `git diff internal/app/dispatcher/spawn.go` — the diff hunk at `@@ -224,30 +250,42 @@` shows the inline `os.MkdirTemp` block removed and the NewBundle + WriteManifest + cleanup path inserted in its place.

PASS.

### 1.7 C7 — 14 bundle tests + 2 spawn integration tests cover the surface

- `bundle_test.go` — 14 `func Test...` declarations at lines 31, 77, 96, 131, 149, 185, 206, 238, 260, 272, 330, 372, 388, 407 (verified by literal `grep -c "^func Test"` returning `14`):
  1. `TestNewBundleOSTempMode`
  2. `TestNewBundleOSTempModeExplicitConstant`
  3. `TestNewBundleProjectMode`
  4. `TestNewBundleProjectModeRequiresProjectRoot`
  5. `TestNewBundleRejectsUnknownSpawnTempRoot` (5 sub-cases)
  6. `TestNewBundleRejectsEmptyActionItemID`
  7. `TestBundlePathsAreUnderRoot`
  8. `TestBundleCleanupIdempotent`
  9. `TestBundleCleanupZeroValueIsSafe`
  10. `TestBundleWriteManifestRoundTrip`
  11. `TestBundleWriteManifestKeysExactShape`
  12. `TestBundleWriteManifestRejectsZeroValueBundle`
  13. `TestNewBundleSpawnIDIsUUIDLike`
  14. `TestNewBundleSpawnIDsUnique`
- `spawn_test.go` — 2 new `func Test...` declarations:
  - `TestBuildSpawnCommandWritesManifestJSON` at line 569 (asserts F.7.1 bundle integration end-to-end: manifest exists, has 5 keys, action_item_id / kind / paths sourced from item).
  - `TestBuildSpawnCommandBundleRootUnderOSTempDir` at line 633 (pins os_tmp resolution path).

NIT: worklog §2.6 prose says "Twelve test functions" but the bullet list directly below enumerates all 14 names. The actual count and the prompt's claim of 14 are correct; the prose count drifted. Not material to acceptance.

PASS.

### 1.8 C8 — `mage ci` green per worklog (2540 passed, 70.7% dispatcher, 97.0% templates)

- `mage ci` re-run: 2541 tests, 2540 passed, 1 skipped (`TestStewardIntegrationDropOrchSupersedeRejected` in `mcpapi`), 0 failed.
- Coverage table (from `mage ci` output):
  - `internal/app/dispatcher` = **70.6%** (worklog claims 70.7% — off by 0.1pp; minor non-material drift; both above the 70% threshold).
  - `internal/templates` = **97.0%** (matches worklog exactly).
- Coverage threshold message: `Minimum package coverage: 70.0%. [SUCCESS] Coverage threshold met`.

NIT: an initial `mage ci` run during evidence-gathering reported a transient `[PKG FAIL] internal/app/dispatcher` build error with 2327 passing tests (race-instrumented compile cache cold). Re-run produced the green verdict cited above. Behavior is consistent with race-build-cache transients on the F.7.6 untracked files joining the package; not a F.7.1 regression. Worklog claim is accurate on a steady-state run.

PASS.

### 1.9 C9 — NO commit by builder per REV-13

- `git log --oneline -5` shows HEAD at `d3fbb14 feat(dispatcher): wire build spawn command to cli adapter registry` (the F.7.17.5 baseline). No F.7.1 commits at HEAD.
- `git status --porcelain` shows all F.7.1-related changes as `M` (modified) or `??` (untracked) — none staged, none committed.
- `F7_CORE_PLAN.md:1103-1107` — REV-13 cited verbatim: builder spawn prompts MUST forbid `git commit` / `git add ... && git commit`; orchestrator drives commits AFTER QA pair returns green.

PASS.

### 1.10 C10 — Scope: only the 9 listed files touched

The 9 files in the spawn prompt are all present:

1. `internal/templates/schema.go` — modified.
2. `internal/templates/load.go` — modified.
3. `internal/templates/load_test.go` — modified.
4. `internal/templates/schema_test.go` — modified.
5. `internal/app/dispatcher/bundle.go` — NEW (untracked).
6. `internal/app/dispatcher/bundle_test.go` — NEW (untracked).
7. `internal/app/dispatcher/spawn.go` — modified.
8. `internal/app/dispatcher/spawn_test.go` — modified.
9. `workflow/drop_4c/4c_F7_1_BUILDER_WORKLOG.md` — NEW (untracked).

**FINDING — scope creep, NIT-grade**: the working tree carries non-trivial out-of-scope changes that the worklog does not declare:

- `internal/app/dispatcher/plugin_preflight.go` (untracked, NEW, ~328 lines) — F.7.6 work.
- `internal/app/dispatcher/plugin_preflight_test.go` (untracked, NEW, ~447 lines) — F.7.6 work.
- `internal/templates/schema.go:262-292` — `RequiresPlugins []string \`toml:"requires_plugins,omitempty"\`` field added to `Tillsyn` struct. F.7.6 work.
- `internal/templates/load.go:823-825` + `:850-883` — `validateTillsynRequiresPlugins(...)` call wired into `validateTillsyn` and the helper itself (~33 lines) added. F.7.6 work.
- `internal/templates/load_test.go:1336-1505` (approx) — five new `TestLoadTillsynRequiresPlugins*` test functions (HappyPath, OmittedZeroValue, EmptySliceAllowed, RejectionTable, CaseSensitiveDistinct). F.7.6 work.
- `internal/app/dispatcher/spawn.go:218-240` — F.7.6 plugin pre-flight call site (`if hook := RequiredPluginsForProject; hook != nil { ... CheckRequiredPlugins(...) }`) inserted before binding resolve. F.7.6 work.
- `workflow/drop_4c/SKETCH.md` — modified with substantial F.7-CORE prose updates (~83 line additions). Architectural / planning content; not part of F.7.1 acceptance.

`F7_CORE_PLAN.md:1039-1048` (REV-7) explicitly partitions the `Tillsyn` struct extension policy: F.7.1 owns `SpawnTempRoot`; F.7.6 owns `RequiresPlugins`. Both droplets land AFTER F.7.18.2 — but they remain SEPARATE droplets with independent acceptance contracts. The F.7.6 droplet has its own future build/QA cycle.

The prompt's Hard Constraint says "Verify scope: only the 9 listed files touched." That hard constraint is violated. However:

- The F.7.1-scoped contract itself is **complete and correct** — all 10 criteria pass on F.7.1 surface alone.
- The F.7.6 work is internally consistent (struct field + validator + plugin_preflight package + spawn.go call site + test surface) and `mage ci` is green with all of it in place.
- No commit was made (REV-13), so the orchestrator can decide whether to:
  (a) split the working tree into two commits (F.7.1 + F.7.6) and run F.7.6's QA twins separately, or
  (b) accept the bundled work as one logical unit and adjust the action-item tree accordingly.

The decision is orchestrator-owned. From a proof-of-claim standpoint, the F.7.1 evidence is intact; from a scope-discipline standpoint, this is a clear NIT.

GREEN-WITH-NITS.

## 2. Missing evidence

- 2.1 None on the F.7.1 contract itself. Every criterion has direct file:line evidence.
- 2.2 The worklog does not acknowledge the F.7.6 scope creep — that omission is itself missing evidence the dev/orchestrator should be aware of.

## 3. Summary

**Verdict: GREEN-WITH-NITS (PROOF GREEN on F.7.1 contract; NIT on out-of-scope F.7.6 bundling).**

F.7.1's 10 acceptance criteria all hold. The schema extension, closed-enum validator, Bundle lifecycle (`NewBundle` / `Cleanup` / `WriteManifest`), os_tmp + project mode resolution, manifest payload shape (5 keys, no `cli_kind`, no `claude_pid`), spawn.go integration with cleanup-on-failure, 14+2 test surface, mage ci green, and no-commit-by-builder are all directly evidenced. The orchestrator should weigh the scope NIT (F.7.6 RequiresPlugins schema + plugin pre-flight + 2 untracked files + SKETCH.md) before committing — either split into two commits with separate F.7.6 QA twins, or treat the bundled work as one logical chunk with explicit acceptance.

## TL;DR

- T1 — Per-criterion evidence: 10/10 criteria pass on F.7.1 surface with file:line citations; one NIT on bundle-test prose-vs-actual count (Twelve vs 14 — actual count is 14 and matches the prompt's claim); one NIT on coverage drift (worklog 70.7% vs measured 70.6%, both above 70% threshold); transient mage ci flake on cold race-build cache resolved on re-run.
- T2 — Missing evidence: none on the F.7.1 contract; the worklog's omission of the F.7.6 scope creep is itself a missing acknowledgement.
- T3 — Summary: GREEN-WITH-NITS — F.7.1 proof bar met cleanly; orchestrator decision required on the F.7.6 bundling (untracked plugin_preflight.go + plugin_preflight_test.go, RequiresPlugins schema + validator + 5 tests, spawn.go pre-flight call site, SKETCH.md edits).
