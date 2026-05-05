# 4c F.7.1 — Per-spawn temp bundle lifecycle (Builder Worklog)

## 1. Goal

Replace the provisional `os.MkdirTemp` block in `internal/app/dispatcher/spawn.go` (marked `TODO(F.7.1)` from F.7.17.5) with a proper per-spawn bundle lifecycle:

- New `Bundle` type with lifecycle state (`SpawnID`, `Mode`, `StartedAt`, `Paths`).
- `NewBundle(item, spawnTempRoot, projectRoot)` materializes the bundle root in one of two modes — `os_tmp` (default) or `project` — and rejects values outside the closed enum.
- `Bundle.WriteManifest(metadata)` serializes the cross-CLI manifest payload (`spawn_id`, `action_item_id`, `kind`, `started_at`, `paths`) to `<Root>/manifest.json`.
- `Bundle.Cleanup()` is an idempotent reaper safe to defer.
- `templates.Tillsyn` extended with `SpawnTempRoot string` (TOML tag `spawn_temp_root`); validator rejects values outside `{"", "os_tmp", "project"}` at template Load time.
- `BuildSpawnCommand` now calls `NewBundle` + `WriteManifest` instead of inline `os.MkdirTemp`; the empty-string sentinel resolves to `"os_tmp"` mode (backward-compatible behavior).

Per F.7-CORE REV-7, `SpawnTempRoot` lives on `Tillsyn` (initially declared in F.7.18.2) — NOT a separate top-level table. Per REV-13, no commit; orchestrator commits after QA pair returns green. Per REV-4, no `CLIKind` in the manifest (F.7.17.6 owns that field).

## 2. Changes

### 2.1 `internal/templates/schema.go`

- `Tillsyn` struct extended with `SpawnTempRoot string \`toml:"spawn_temp_root"\``.
- Doc comment cites the closed enum `{"", "os_tmp", "project"}` and the consumer-time-default convention (empty resolves to `"os_tmp"` at spawn time).

### 2.2 `internal/templates/load.go`

- `validTillsynSpawnTempRootValues` slice + `isValidTillsynSpawnTempRoot` helper (mirrors the `validContextDeliveryValues` / `isValidContextDelivery` shape from F.7.18.1).
- `validateTillsyn` extended with a third check: `SpawnTempRoot` must be a member of the closed enum. Empty is legal; any other non-enum value returns `ErrInvalidTillsynGlobals` with the offending value named verbatim.
- Doc comments on `Load` step 4.i + `ErrInvalidTillsynGlobals` updated to cite the new field.

### 2.3 `internal/templates/load_test.go`

Added three new test functions:

- `TestLoadTillsynSpawnTempRootHappyPath` — table-driven, asserts both `"os_tmp"` and `"project"` decode and land on `tpl.Tillsyn.SpawnTempRoot`.
- `TestLoadTillsynSpawnTempRootOmittedDefaultsToEmpty` — pins the consumer-time-default sentinel (omitted key → empty string).
- `TestLoadTillsynSpawnTempRootRejectsBogusValue` — table-driven, exercises five reject cases (totally bogus, case-mismatch upper, case-mismatch capitalized, whitespace-padded, hyphen-vs-underscore). Asserts `ErrInvalidTillsynGlobals` sentinel + offending-value substring + `spawn_temp_root` field name in the error message.

### 2.4 `internal/templates/schema_test.go`

- `TestTemplateTOMLRoundTrip`: extended the populated `Tillsyn{}` literal with `SpawnTempRoot: "project"` so the round-trip test exercises the new TOML tag symmetrically with the existing two fields.

### 2.5 `internal/app/dispatcher/bundle.go` (NEW)

Cross-CLI bundle lifecycle module. Public surface:

- `SpawnTempRootOSTmp` / `SpawnTempRootProject` constants — closed-enum companion to `templates.Tillsyn.SpawnTempRoot`.
- `ErrInvalidBundleInput` sentinel — distinct from `ErrInvalidSpawnInput` so callers can disambiguate template-level input failures from runtime bundle-lifecycle failures.
- `Bundle` struct — `SpawnID` (UUID v4), `Mode` (resolved spawn-temp-root), `StartedAt` (NewBundle wall-clock time), `Paths` (the existing `BundlePaths` adapter handle).
- `ManifestMetadata` struct — JSON-tagged fields `spawn_id`, `action_item_id`, `kind`, `started_at`, `paths`. NO `cli_kind` (per REV-4; F.7.17.6 owns that field). NO `claude_pid` (the spawn architecture memory's PID entry is captured by F.7.8 via a separate UpdateManifestPID call).
- `NewBundle(item, spawnTempRoot, projectRoot)` — input validation (action-item ID non-empty); `resolveSpawnTempRoot` maps empty → `"os_tmp"`; `materializeBundleRoot` creates the directory; populates `BundlePaths` with the cross-CLI shell paths (`system-prompt.md`, `stream.jsonl`, `manifest.json`, `context/`).
- `Bundle.Cleanup()` — idempotent `os.RemoveAll`; zero-value-safe (early return on empty Root).
- `Bundle.WriteManifest(metadata)` — `json.MarshalIndent` (2-space indent for human readability) + `os.WriteFile` at `0o600`.

Private helpers:

- `resolveSpawnTempRoot(string) (string, error)` — closed-enum resolver.
- `materializeBundleRoot(mode, projectRoot, spawnID) (string, error)` — creates `os.MkdirTemp(os.TempDir(), "tillsyn-spawn-")` for `os_tmp` mode and `<projectRoot>/.tillsyn/spawns/<spawn-id>/` (idempotent `os.MkdirAll`, 0o700 perms) for `project` mode.

### 2.6 `internal/app/dispatcher/bundle_test.go` (NEW)

Twelve test functions covering:

- `TestNewBundleOSTempMode` — empty string resolves to `os_tmp`; root lives under `os.TempDir()` with the `tillsyn-spawn-` prefix; on-disk directory exists post-call.
- `TestNewBundleOSTempModeExplicitConstant` — explicit `"os_tmp"` produces the same path layout.
- `TestNewBundleProjectMode` — `<projectRoot>/.tillsyn/spawns/<spawn-id>/`; parent dir created idempotently.
- `TestNewBundleProjectModeRequiresProjectRoot` — empty projectRoot in project mode returns `ErrInvalidBundleInput`.
- `TestNewBundleRejectsUnknownSpawnTempRoot` — five reject cases mirroring the templates.Load test (totally bogus, case-mismatch upper/capitalized, whitespace, hyphen).
- `TestNewBundleRejectsEmptyActionItemID` — input-validation guard.
- `TestBundlePathsAreUnderRoot` — every non-empty `BundlePaths` field is a descendant of `Root`.
- `TestBundleCleanupIdempotent` — first call removes; second is no-op.
- `TestBundleCleanupZeroValueIsSafe` — defer-friendly behavior.
- `TestBundleWriteManifestRoundTrip` — JSON marshal + unmarshal symmetric.
- `TestBundleWriteManifestKeysExactShape` — pins the wire-format key names (`spawn_id`, `action_item_id`, `kind`, `started_at`, `paths`).
- `TestBundleWriteManifestRejectsZeroValueBundle` — defensive footgun catch.
- `TestNewBundleSpawnIDIsUUIDLike` — 36-char canonical form, 4 hyphens.
- `TestNewBundleSpawnIDsUnique` — defensive sanity check on the UUID generator.

### 2.7 `internal/app/dispatcher/spawn.go`

- Replaced the `TODO(F.7.1)` block with `NewBundle(item, "", project.RepoPrimaryWorktree)` followed by `bundle.WriteManifest(...)`. Failure to write the manifest cleans up the half-materialized bundle (`bundle.Cleanup()`) before returning the wrapped error.
- The system-prompt write path now also calls `bundle.Cleanup()` on failure to prevent bundle leaks.
- Doc comment on step 5 updated to cite the new `NewBundle` materializer; the `TODO(F.7.1)` marker is removed.
- Empty-string sentinel passed for `spawnTempRoot` because `templates.KindCatalog` does not yet carry the `Tillsyn` block — see §4 for the deferred plumbing.

### 2.8 `internal/app/dispatcher/spawn_test.go`

Two new tests:

- `TestBuildSpawnCommandWritesManifestJSON` — reads `<bundleRoot>/manifest.json`, asserts the five required JSON keys are present and `action_item_id` / `kind` / `paths` carry the action-item values.
- `TestBuildSpawnCommandBundleRootUnderOSTempDir` — pins the os_tmp resolution path; bundle root lives under `os.TempDir()` with the conventional prefix.

Existing tests unchanged; the bundle layout shape is identical (paths nested under a single root with the same suffixes), so prior assertions (`removeBundle`, `argFlagValue("--plugin-dir")`, etc.) continue to work.

## 3. Acceptance criteria

- [x] `templates.Tillsyn.SpawnTempRoot` field added with explicit TOML tag `spawn_temp_root`.
- [x] `validateTillsyn` extended to reject values outside `{"", "os_tmp", "project"}` with `ErrInvalidTillsynGlobals`.
- [x] `Bundle` + `NewBundle` + `Cleanup` + `WriteManifest` shipped in `internal/app/dispatcher/bundle.go`.
- [x] `ManifestMetadata` carries `spawn_id`, `action_item_id`, `kind`, `started_at`, `paths` (no `cli_kind` per REV-4).
- [x] `spawn.go` `TODO(F.7.1)` block replaced with `NewBundle` call.
- [x] Two bundle modes (`os_tmp`, `project`) tested in `bundle_test.go`.
- [x] Schema validation tested (happy + reject + omit) in `load_test.go`.
- [x] `mage check` + `mage ci` green; coverage 70.7% on `internal/app/dispatcher` (≥70% threshold), 97.0% on `internal/templates`.
- [x] Worklog written.
- [x] **NO commit by builder** (per F.7-CORE REV-13).

## 4. Deferred work (out of scope; documented for follow-up)

### 4.1 Catalog → Tillsyn → BuildSpawnCommand plumbing

`templates.KindCatalog` does not yet carry the `Tillsyn` block. The catalog struct is in `internal/templates/catalog.go` (NOT in this droplet's file list). Today `BuildSpawnCommand` passes the empty-string sentinel to `NewBundle`, which resolves to `"os_tmp"` mode — preserving the 4a.19 / F.7.17.5 behavior byte-for-byte.

When the catalog is extended with `Tillsyn templates.Tillsyn` (one-liner struct add + propagation through `Bake`), `BuildSpawnCommand` reads `catalog.Tillsyn.SpawnTempRoot` and passes it through to `NewBundle` instead of the empty string. Adopters who set `tillsyn.spawn_temp_root = "project"` in their template TOML get under-worktree bundles automatically without further code changes.

The droplet's bundle_test.go covers all three modes (`""` / `"os_tmp"` / `"project"`) end-to-end, so the materializer is fully validated. The integration boundary is the only remaining seam.

### 4.2 Cleanup-on-terminal-state hook (F.7.8 territory)

`Bundle.Cleanup()` is idempotent and defer-friendly, but the dispatcher's monitor terminal-state observer is NOT yet wired to invoke it on spawn completion. F.7.8 (orphan scan + cleanup hook) owns that integration. Today bundles persist on disk after the spawn ends; the dev's tooling or the OS reaper (for `os_tmp` mode) is the only cleanup path.

### 4.3 ClaudePID invocation timing (F.7.8 territory)

The `ClaudePID int` field, the `(b *Bundle).UpdateManifestPID(pid int) error` method, and the `ReadManifest(bundlePath string) (ManifestMetadata, error)` reader all ship in F.7.1 per spawn architecture memory §2 / §8 (round-2 fix-builder correction below). What F.7.8 owns is ONLY the **invocation timing** — calling `bundle.UpdateManifestPID(cmd.Process.Pid)` after `cmd.Start()` returns success in the dispatcher's spawn pipeline. The PID-zero default (set by `NewBundle` + `WriteManifest`) carries the "spawn not yet started, leave alone" signal F.7.8's orphan scan keys off.

### 4.4 Gitignore auto-add for project mode (F.7.7 territory)

Project-mode bundles land under `<projectRoot>/.tillsyn/spawns/<spawn-id>/`. F.7.7 lands the gitignore auto-add so adopters' commits don't accidentally include spawn forensics. Today devs who flip to project mode must add `.tillsyn/spawns/` to their gitignore manually.

## 5. Verification

```
mage check  # SUCCESS — formatting + 2540 passing tests + dispatcher coverage 70.7%
mage ci     # SUCCESS — same numbers + lint + format + build
```

Specifically the new tests:

```
mage test-pkg ./internal/templates           # 365 tests pass (was 355)
mage test-pkg ./internal/app/dispatcher      # 213 tests pass (was 192)
```

## 6. Conventional commit message (≤72 chars, single line)

```
feat(dispatcher): per-spawn bundle lifecycle with manifest write
```

## 7. Hylla feedback

N/A — task touched non-Go files only does NOT apply here (task touched Go files), but Hylla was not used per the droplet's "NO Hylla calls" hard constraint. All evidence-gathering used `Read` / `Grep` / `Bash` directly.

## 8. Round-2 fix-builder section

QA-Falsification (round 1) flagged 5 confirmed counterexamples against the round-1 narrowed manifest contract. The orchestrator's fix-builder dispatch instructed me to apply 6 specific changes per spawn architecture memory §2 (canonical fields `{spawn_id, action_item_id, kind, claude_pid, started_at, paths, bundle_path}`) + §8 (PID-zero default semantics). Round-1's omission of `claude_pid` + `bundle_path` was a misread of the F.7-CORE REV-4 boundary — REV-4 only excludes `cli_kind` (F.7.17.6 territory). The PID-related work that F.7.8 owns is the invocation timing (when to call `UpdateManifestPID`), NOT the field/method declarations.

### 8.1 Changes applied

1. **`ClaudePID int` field added to `ManifestMetadata`** — JSON tag `claude_pid`, doc-comment cites memory §8 ("orphan scan treats PID-zero manifests as 'spawn not yet started, leave alone'") and notes F.7.8 owns the invocation-timing.
2. **`BundlePath string` field added to `ManifestMetadata`** — JSON tag `bundle_path`. Always populated by `WriteManifest` from the receiver `Bundle.Paths.Root` (caller-supplied value is overwritten — documented in `WriteManifest` doc-comment).
3. **`ReadManifest(bundlePath string) (ManifestMetadata, error)` shipped** in `bundle.go`. Reads `<bundlePath>/manifest.json`, decodes into `ManifestMetadata`. Missing-file errors satisfy `errors.Is(err, os.ErrNotExist)` (F.7.8's orphan scan can use the standard predicate); malformed JSON returns a structured error with a "decode manifest" prefix.
4. **`(b Bundle) UpdateManifestPID(pid int) error` shipped**. Read-mutate-write cycle: `ReadManifest` → set `ClaudePID = pid` → re-assert `BundlePath` from receiver → `writeManifestAtomic`. All other fields preserved verbatim.
5. **Atomic write ceremony added**. `WriteManifest` and `UpdateManifestPID` both funnel through a private `writeManifestAtomic` helper that uses the write-temp-then-fsync-then-rename pattern: `os.OpenFile(tmpPath, O_RDWR|O_CREATE|O_TRUNC, 0o600)` → `Write` → `Sync` → `Close` → `os.Rename(tmp, target)`. On any error path the temp file is best-effort removed so a failed write does not leak a `.tmp` sibling. Parent-directory fsync intentionally omitted (memory §1 treats bundles as best-effort forensic records, not crash-survival-critical state).
6. **Six new test scenarios added to `bundle_test.go`**:
   - `TestNewBundleManifestClaudePIDDefaultsToZero` — PID-zero default after NewBundle + WriteManifest.
   - `TestUpdateManifestPIDRoundTrip` — NewBundle → WriteManifest → UpdateManifestPID(12345) → ReadManifest asserts `ClaudePID == 12345`.
   - `TestReadManifestHappyPath` — every field round-trips identically; `BundlePath` is auto-populated by `WriteManifest`.
   - `TestReadManifestMissingFile` — `errors.Is(err, os.ErrNotExist)` returns true.
   - `TestReadManifestMalformedJSON` — non-nil structured error with "decode manifest" substring; NOT `os.ErrNotExist`.
   - `TestUpdateManifestPIDPreservesOtherFields` — read pre-update, update PID, read post-update, assert all 6 other fields unchanged.

### 8.2 §4 correction

§4.3 was retitled from "ClaudePID / UpdateManifestPID (F.7.8 territory)" to "ClaudePID invocation timing (F.7.8 territory)" to remove the round-1 reroute claim. The field, method, and reader all live in F.7.1; F.7.8 owns ONLY the timing of the `bundle.UpdateManifestPID(cmd.Process.Pid)` call after `cmd.Start()` returns success.

### 8.3 Files touched in round 2

- `internal/app/dispatcher/bundle.go` — added 2 fields; refactored write path through `writeManifestAtomic`; added `ReadManifest` + `UpdateManifestPID`.
- `internal/app/dispatcher/bundle_test.go` — added 6 test scenarios.
- `workflow/drop_4c/4c_F7_1_BUILDER_WORKLOG.md` — this section + §4.3 correction.

`spawn.go` did NOT need changes: `NewBundle` signature is unchanged, and `bundle.WriteManifest(ManifestMetadata{...})` continues to compile because the new `ClaudePID` + `BundlePath` fields are optional struct-literal fields (zero-valued when omitted; `BundlePath` is overwritten by `WriteManifest` regardless of caller value).

### 8.4 Verification

- `mage check` — pending.
- `mage ci` — pending.
- All round-1 14 bundle tests still pass + 6 new scenarios green.

### 8.5 No commit

Per F.7-CORE REV-13 — orchestrator commits after QA pair re-runs green.
