# Drop 4c F.7.1 — BUILDER QA Falsification Review

## 1. Findings

- 1.1 The builder's `ManifestMetadata` struct OMITS `claude_pid int` (the spawn architecture memory §2 PID slot) despite the plan REVISIONS REV-4 (`workflow/drop_4c/F7_CORE_PLAN.md` line 1027) explicitly listing `claude_pid` IN the F.7.1 manifest field set: *"F.7.1 acceptance criteria: list manifest fields as `{spawn_id, action_item_id, kind, claude_pid, started_at, paths}` — NO `CLIKind`."* REV-4 strikes `cli_kind` only; the builder erroneously also struck `claude_pid` and pushed it to F.7.8 via worklog §4.3. CONFIRMED counterexample — the builder misread REV-4.
- 1.2 The plan's F.7.1 acceptance body (line 167-168) mandates `ReadManifest(bundlePath string) (Manifest, error)` (round-trip with wrapped `os.ErrNotExist`) AND fsync-on-close inside `WriteManifest`. Neither shipped: bundle.go has no `ReadManifest` function, and `WriteManifest` (line 287-299) uses plain `os.WriteFile` with the doc-comment (line 284-286) explicitly waiving fsync — *"The function does NOT fsync — fsync semantics for forensic durability are F.7.8's concern."* That deferral is unilateral; no REVISION authorizes it.
- 1.3 The plan's F.7.1 acceptance body (line 180) mandates `UpdateManifestPID(b, pid)` so the spawn caller can populate `ClaudePID` after `cmd.Start()` succeeds. Not shipped — coupled to the missing `ClaudePID` field. The plan's own falsification mitigation §"Falsification mitigations to bake in" line 184 is built around `UpdateManifestPID` existing. CONFIRMED.
- 1.4 File-decomposition deviation: plan line 154-157 mandates four files — `spawn_bundle.go`, `spawn_manifest.go`, `spawn_bundle_test.go`, `spawn_manifest_test.go`. Builder shipped two consolidated files — `bundle.go` and `bundle_test.go`. NIT: functionality preserved, but the plan's split-by-concern intent (bundle lifecycle vs manifest schema) is collapsed.
- 1.5 API-shape deviations from plan: `MkdirBundle(spawnID, tempRoot, projectWorktree)` → `NewBundle(item, spawnTempRoot, projectRoot)` (function name + signature), `RemoveBundle(b)` → `(Bundle).Cleanup()` (free-fn → method), `WriteManifest(b, m)` → `(Bundle).WriteManifest(m)` (free-fn → method). Plan also specifies a positional `spawnID string` input; builder generates the UUID inside NewBundle from `uuid.NewString()` instead. NITs: each substitution is reasonable Go style, but every deviation is unflagged in the worklog.
- 1.6 The plan acceptance line 165 specifies `Bundle struct exposes BundlePaths via accessor`. Builder exposes via direct field `Paths BundlePaths`. NIT.
- 1.7 Plan line 165 mandates manifest field `BundlePath string`. Builder omits this field. CONFIRMED — a third missing field beyond `claude_pid` / `cli_kind`.

## 2. Counterexamples

- 2.1 **CONFIRMED — REV-4 misreading: `claude_pid` MUST be on the F.7.1 manifest.** Evidence: `F7_CORE_PLAN.md` line 1027 (REV-4) lists the F.7.1 manifest fields as `{spawn_id, action_item_id, kind, claude_pid, started_at, paths}`. Plan body line 165 reinforces with the typed declaration `ClaudePID int (zero until cmd.Start succeeds)`. Plan body line 180 details the contract: `ClaudePID field is 0 in WriteManifest's initial call; spawn caller MUST update via UpdateManifestPID(b, pid) AFTER cmd.Start() returns successfully`. Builder shipped `bundle.go` line 109-132 with no `ClaudePID` field, and worklog §4.3 reroutes the field to F.7.8. The reroute contradicts REV-4 — only `cli_kind` was rerouted to F.7.17.6. Reproduction: read REV-4 + plan body line 180; observe that `claude_pid` is the load-bearing piece of the F.7.8 orphan-scan contract ("orphan scan treats PID-zero manifests as 'spawn not yet started'"); confirm the field is absent from the shipped struct.
- 2.2 **CONFIRMED — `ReadManifest` is missing.** Evidence: plan line 167 lists `ReadManifest(bundlePath string) (Manifest, error)` round-trips JSON; returns wrapped `os.ErrNotExist`. Search bundle.go: no `ReadManifest` declaration. Reproduction: `grep -n "func.*ReadManifest" internal/app/dispatcher/bundle.go` returns nothing. F.7.8's orphan-scan needs ReadManifest to load the manifest at scan-time; without it F.7.8 reaches into json.Unmarshal directly, which is a leakier seam.
- 2.3 **CONFIRMED — `UpdateManifestPID` is missing.** Evidence: plan body line 180 mandates the function. Search bundle.go: no `UpdateManifestPID` declaration. Reproduction: `grep -n "UpdateManifestPID" internal/app/dispatcher/bundle.go` returns nothing. Coupled to 2.1 — without `ClaudePID` in the struct there is nothing to update.
- 2.4 **CONFIRMED — fsync omission.** Evidence: plan line 168 specifies `WriteManifest writes JSON to <Root>/manifest.json; fsync on close`. bundle.go line 295 calls `os.WriteFile(b.Paths.ManifestPath, payload, 0o600)` — no fsync. Doc-comment line 284-286 explicitly waives fsync and reroutes to F.7.8. The reroute is again unilateral; no REVISION authorizes it. Reproduction: read bundle.go lines 287-299; confirm absence of any `*os.File` open + `f.Sync()` ceremony.
- 2.5 **CONFIRMED — `BundlePath string` manifest field missing.** Evidence: plan line 165 lists `BundlePath string` as a manifest field. Builder's `ManifestMetadata` (bundle.go line 109-132) omits this field. Reproduction: read the struct; observe five json-tagged fields, no `bundle_path`. Note: BundlePath duplicates information the file's own location carries (the manifest IS at `<bundle>/manifest.json`), but the plan still specifies it; if redundant, REV-X should have struck it. None did.

## 3. Summary

**Verdict: NEEDS-REWORK.**

Five CONFIRMED counterexamples (2.1, 2.2, 2.3, 2.4, 2.5) — three missing manifest fields (`claude_pid`, `cli_kind` is excused by REV-4 but `claude_pid` and `bundle_path` are not), one missing function (`ReadManifest`), one missing method (`UpdateManifestPID`), and a missing fsync acceptance criterion. The bulk of the gap is the builder unilaterally rerouting `ClaudePID`-related work to F.7.8 despite REV-4 explicitly keeping `claude_pid` in F.7.1's manifest — the reroute disposes of a load-bearing field for the F.7.8 orphan-scan contract that F.7.8 itself depends on F.7.1 already shipping.

The shipped surface (`NewBundle` mode resolution, idempotent `Cleanup`, manifest JSON write of the five-key payload, schema-layer enum validation, spawn.go integration with cleanup-on-failure) is solid in isolation — the test surface is thorough (14 bundle_test.go tests + 2 spawn_test.go tests + 3 load_test.go tests + 1 schema_test.go round-trip extension). All A1/A2/A4-A8/A10-A14 attack vectors REFUTED with file-cited evidence. The blocker is the manifest contract being narrower than REV-4 + plan body specify.

Recommended re-spin scope (single round):

1. Add `ClaudePID int` json-tagged field to `ManifestMetadata`.
2. Add `BundlePath string` json-tagged field to `ManifestMetadata` (mirror Bundle.Paths.Root).
3. Ship `ReadManifest(bundlePath string) (ManifestMetadata, error)` — wraps `os.ErrNotExist` on missing.
4. Ship `(*Bundle) UpdateManifestPID(pid int) error` — read-modify-write the manifest.json with the new PID.
5. Add `f.Sync()` ceremony inside `WriteManifest` so fsync-on-close is honored.
6. Extend bundle_test.go with: PID-zero default test, UpdateManifestPID round-trip test, ReadManifest happy-path + missing-file test, fsync verification (best-effort — fsync is hard to assert; presence of the call site is acceptable).
7. Worklog §4 deferred-work: drop the `claude_pid` reroute; F.7.8 still owns the *invocation* of `UpdateManifestPID` after `cmd.Start()`, but the *field + method* belong here per REV-4.

The NITs (1.4, 1.5, 1.6) are acceptable as-is — the file decomposition / API-shape deviations are reasonable Go-idiom substitutions and don't break any contract; the worklog should add a §2 note that they are intentional deviations from the plan-body file/API names so QA-Proof has a paper trail.

## TL;DR

- T1 Three counterexample groupings all converge on the builder's unilateral reroute of `claude_pid`-related work to F.7.8 (missing `ClaudePID` field, missing `UpdateManifestPID`, missing `BundlePath` field), one missing `ReadManifest`, and one missing fsync.
- T2 The reroute contradicts REVISIONS REV-4 which explicitly KEEPS `claude_pid` in the F.7.1 manifest while striking only `cli_kind`.
- T3 NEEDS-REWORK; single-round re-spin scope is concrete and bounded (6 changes listed).

## Hylla Feedback

`N/A — review touched non-Go code understanding only via direct Read on Go source` (per droplet hard constraint "No Hylla calls"). All evidence-gathering used `Read` directly on the source files; no Hylla queries issued. No miss to record.

---

# Round 2 Verification

## R2.1 Round-1 counterexample fix verification

- R2.1.V1 **`ClaudePID int` field** — VERIFIED PASS. `bundle.go` line 136: `ClaudePID int \`json:"claude_pid"\`` declared inside `ManifestMetadata`. JSON tag matches `claude_pid` exactly. Doc-comment lines 128-135 explicitly cite "spawn architecture memory §8" and document the zero-value semantics ("spawn not yet started, leave alone" signal F.7.8's orphan scan keys off, distinguishing pre-start / live / abandoned via PID==0 vs PID-alive vs PID-dead). The doc-comment ALSO clarifies invocation-timing ownership: "NewBundle never sets this field; only UpdateManifestPID writes it." Memory-§8 citation is concrete, not vague. CONFIRMED FIXED.

- R2.1.V2 **`BundlePath string` field with WriteManifest auto-population** — VERIFIED PASS. `bundle.go` line 155: `BundlePath string \`json:"bundle_path"\`` declared inside `ManifestMetadata`. JSON tag matches `bundle_path` exactly. WriteManifest (line 321-327) auto-populates: line 325 `metadata.BundlePath = b.Paths.Root` BEFORE the call to `writeManifestAtomic`. Defensive-against-caller-supplied-wrong-path: yes — even if the caller passes a populated `BundlePath` field, line 325 unconditionally overwrites with the receiver's Root. Doc-comment lines 152-155 document the contract precisely: "Always populated by WriteManifest at write time (sourced from the receiver Bundle.Paths.Root)." CONFIRMED FIXED.

- R2.1.V3 **`ReadManifest(bundlePath) (ManifestMetadata, error)`** — VERIFIED PASS. `bundle.go` line 346: `func ReadManifest(bundlePath string) (ManifestMetadata, error)` shipped at package level (free function, NOT a method — appropriate since reading a manifest from a bare path doesn't need a Bundle receiver). Missing-file branch: line 351-357 — `os.ReadFile` returns errors satisfying `errors.Is(err, os.ErrNotExist)`, and the `%w` wrap on line 356 preserves the sentinel through the wrapping layer. Test `TestReadManifestMissingFile` (bundle_test.go line 542-554) asserts `errors.Is(err, os.ErrNotExist)` directly. Malformed-JSON branch: line 358-361 wraps with "decode manifest" prefix. Test `TestReadManifestMalformedJSON` (bundle_test.go line 559-578) asserts both `!errors.Is(err, os.ErrNotExist)` AND `strings.Contains(err.Error(), "decode manifest")`. Empty-bundlePath defensive guard at line 347-349 returns `ErrInvalidBundleInput`. CONFIRMED FIXED.

- R2.1.V4 **`UpdateManifestPID(pid int) error` on Bundle** — VERIFIED PASS. `bundle.go` line 392: `func (b Bundle) UpdateManifestPID(pid int) error` — value-receiver method (consistent with Bundle's value-type design noted in line 73-75). Reads existing manifest via `ReadManifest(b.Paths.Root)` (line 396), updates `metadata.ClaudePID = pid` (line 400), re-asserts `metadata.BundlePath = b.Paths.Root` (line 404 — defensive self-heal per the doc-comment), writes back via the same `writeManifestAtomic` helper as WriteManifest (line 405). Other-fields preservation: `metadata` is the decoded struct from disk, so SpawnID/ActionItemID/Kind/StartedAt/Paths all survive verbatim. Test `TestUpdateManifestPIDPreservesOtherFields` (bundle_test.go line 584-646) directly asserts every field survives the cycle. Atomic write: shared `writeManifestAtomic` helper (line 418-447). Zero-value defensive guard at line 393-395. CONFIRMED FIXED.

- R2.1.V5 **fsync ceremony in `writeManifestAtomic`** — VERIFIED PASS. `bundle.go` lines 418-447. Sequence: line 424 `os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)` → line 428 `f.Write(payload)` → line 433 `f.Sync()` → line 438 `f.Close()` → line 442 `os.Rename(tmpPath, manifestPath)`. Order is correct: Sync MUST happen BEFORE Close (Sync on a closed fd errors), and Rename MUST happen AFTER Close (renaming an open file is OK on POSIX but ill-defined on Windows; Close-then-Rename is the portable pattern). Each fail-path also unwinds: Close + Remove on Write/Sync failure (lines 429-431, 434-436), Remove on Close failure (line 439), Remove on Rename failure (line 443). Both `WriteManifest` (line 326) AND `UpdateManifestPID` (line 405) funnel through `writeManifestAtomic`, so durability + atomicity contracts are consistent. CONFIRMED FIXED.

## R2.2 New Round-2 attacks

- R2.2.N1 **`UpdateManifestPID` concurrent-call race** — REFUTED with documentation. Two goroutines calling `b1.UpdateManifestPID(p1)` and `b2.UpdateManifestPID(p2)` concurrently on copies of the same Bundle (or even the same value) WOULD produce a lost-update if interleaved: read1 → read2 → write1 → write2 leaves `p2` on disk regardless of ordering. There is no mutex on `Bundle` (line 76-98) and the doc-comment on `UpdateManifestPID` lines 387-391 explicitly acknowledges this: *"Concurrency: there is no internal locking — the spawn architecture assumes a single writer per bundle (the dispatcher's monitor goroutine). Two concurrent UpdateManifestPID calls on the same bundle race; the rename-based write semantics guarantee no torn writes, but last-write-wins applies."* Per the falsification rubric, "Last-writer-wins is OK if documented" — and it is documented, with the single-writer assumption explicitly named. This is not a counterexample; it is an intentional design choice with a stated invariant. The torn-write concern is also addressed: the rename-based atomic-write pattern guarantees readers see either the pre-update OR post-update file in entirety, never a half-written merge. NIT-level observation only: the F.7.8 monitor goroutine integration (which lands later) MUST honor the single-writer-per-bundle invariant — but that is F.7.8's concern, not F.7.1's.

- R2.2.N2 **EXDEV cross-filesystem rename** — REFUTED. The temp file lives in the SAME directory as the target. `bundle.go` line 423: `tmpPath := manifestPath + ".tmp"`. `manifestPath` for an os_tmp-mode bundle is `<os.TempDir()>/tillsyn-spawn-XXXX/manifest.json`, so `tmpPath` is `<os.TempDir()>/tillsyn-spawn-XXXX/manifest.json.tmp` — same parent directory. Same for project-mode: `<projectRoot>/.tillsyn/spawns/<id>/manifest.json` and `manifest.json.tmp` share parent. POSIX `rename(2)` is atomic for same-directory renames and CANNOT fail with EXDEV when source + target are within the same filesystem (which they are, by virtue of being in the same directory). The doc-comment on line 410-411 makes this explicit: *"The temp file is created alongside the target with a `.tmp` suffix so the rename is atomic on POSIX filesystems (same-directory rename is atomic per POSIX rename(2))."* Compare to the EXDEV-vulnerable anti-pattern that uses `os.CreateTemp("", ...)` (which goes to `os.TempDir()`) and renames into a project-mode path — that anti-pattern is NOT used here. CONFIRMED safe.

- R2.2.N3 **`BundlePath` post-cleanup forensic readability** — REFUTED with confirmed-by-design intent. `(b Bundle) Cleanup()` (line 288-293) does `os.RemoveAll(b.Paths.Root)`, which removes the bundle directory and everything inside it including `manifest.json`. After Cleanup, `BundlePath` is unreadable from disk because the manifest itself is gone. This IS the intended design: F.7.8's orphan scan reads manifests of LIVE (pre-cleanup) bundles to identify abandoned spawns and reap them. Once a bundle is cleaned, there is nothing to scan — the scanner walks the spawns directory and finds no entry. The `BundlePath` field's value is for in-flight manifest interpretation (forensic tooling that loads a manifest fragment and needs to confirm its on-disk root location matches the path it was loaded from per the doc-comment lines 152-155), not post-cleanup archaeology. The droplet's "F.7.8 orphan scan reads manifests of LIVE spawns (pre-cleanup), so this is fine" framing is correct. CONFIRMED safe by design.

## R2.3 Round-2 summary

**Verdict: PASS.**

All 5 round-1 counterexamples (V1-V5) are CONFIRMED FIXED with line-cited evidence in `bundle.go`. All 3 new round-2 attacks (N1-N3) are REFUTED — N1 is documented as last-writer-wins (acceptable per rubric), N2 same-directory-rename invariant holds (no EXDEV exposure), N3 is intentional by design (F.7.8 only scans live bundles).

Test coverage on the 6 new test cases is rigorous:

- `TestNewBundleManifestClaudePIDDefaultsToZero` (line 410-441) — V1 + V2 default-zero PID + auto-populated BundlePath
- `TestUpdateManifestPIDRoundTrip` (line 447-478) — V4 happy-path PID flip
- `TestReadManifestHappyPath` (line 484-536) — V3 + V2 round-trip including `decoded.BundlePath != bundle.Paths.Root` assertion
- `TestReadManifestMissingFile` (line 542-554) — V3 `errors.Is(err, os.ErrNotExist)` assertion
- `TestReadManifestMalformedJSON` (line 559-578) — V3 "decode manifest" prefix + non-ErrNotExist assertion
- `TestUpdateManifestPIDPreservesOtherFields` (line 584-646) — V4 every-field-preservation cycle

**No NEEDS-REWORK signals.** The fix-builder's report matches reality exactly: the 6 listed changes all landed, and the test file's 6 new cases all exist with the assertions claimed.

**One PASS-WITH-NITS-grade observation** (does not block PASS, recorded for downstream droplet planners):

- R2.3.NIT-1 — `UpdateManifestPID`'s read-mutate-write sequence (line 396-405) reads a stale snapshot if a concurrent caller has already updated other fields. The single-writer invariant in the doc-comment line 387-391 makes this safe TODAY for F.7.1, but F.7.8's monitor-goroutine architecture (and any future refactor that moves manifest writes off the dispatcher's monitor goroutine) MUST preserve the single-writer-per-bundle invariant or introduce a Bundle-level mutex. This is downstream-droplet concern, not an F.7.1 gap.

## R2.4 TL;DR (Round 2)

- T1 (R2 verification) All 5 round-1 counterexamples CONFIRMED FIXED with file-and-line citations: V1 ClaudePID field present + memory-§8 cited; V2 BundlePath field present + WriteManifest auto-overrides; V3 ReadManifest shipped with proper os.ErrNotExist wrapping + decode-manifest prefix; V4 UpdateManifestPID shipped on value-receiver Bundle preserving all sibling fields; V5 writeManifestAtomic does OpenFile→Write→Sync→Close→Rename in correct order, used by both write paths.
- T2 (R2 new attacks) N1 race REFUTED (documented last-writer-wins with single-writer-per-bundle invariant); N2 EXDEV REFUTED (temp file lives in same directory as target via `manifestPath + ".tmp"`); N3 post-cleanup readability REFUTED (intentional — F.7.8 only scans live bundles).
- T3 (R2 verdict) PASS. No counterexamples remain. Single PASS-WITH-NITS-grade observation about single-writer-invariant honoring in F.7.8 is downstream concern, not an F.7.1 gap.

## R2.5 Hylla Feedback (Round 2)

`N/A — round-2 verification touched only Go code understanding via direct Read on Go source` (per droplet hard constraint "No Hylla calls"). All evidence-gathering used `Read` directly on `bundle.go` and `bundle_test.go`. No Hylla queries issued. No miss to record.
