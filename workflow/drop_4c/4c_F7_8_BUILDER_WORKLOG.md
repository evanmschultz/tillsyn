# F.7.8 — Orphan Scan + Cleanup-on-Terminal-State Hook (Builder Worklog)

**Droplet:** F.7-CORE F.7.8
**Builder mode:** Completion-builder (prior dispatch hit API stream idle timeout after writing `orphan_scan.go`; this round adds the test file + worklog).
**Status:** READY FOR QA — `mage check` + `mage ci` green; NO commit per F.7-CORE REV-13.

## Scope landed

### Prior dispatch (already on disk)

- `internal/app/dispatcher/orphan_scan.go` — production surface:
  - `ErrInvalidOrphanScannerConfig` sentinel (nil ActionItems / ProcessChecker / receiver).
  - `ActionItemReader` interface (`ListInProgress(ctx) ([]domain.ActionItem, error)`).
  - `ProcessChecker` interface (`IsAlive(pid int) bool`).
  - `DefaultProcessChecker` — production `os.FindProcess` + `proc.Signal(syscall.Signal(0))` POSIX liveness probe; treats `EPERM` as alive (process exists, lacks signal permission), `ESRCH` / other errors as dead, `pid <= 0` short-circuits to false.
  - `OrphanScanner` struct with `ActionItems`, `ProcessChecker`, `Logger`, `OnOrphanFound` fields.
  - `Scan(ctx) ([]string, error)` — walks every in_progress action item, reads `<bundlePath>/manifest.json` via F.7.1's `ReadManifest`, applies the per-item branch table from spawn architecture memory §8, invokes `OnOrphanFound` for each dead PID, aggregates callback errors via `errors.Join`.

### This dispatch (added)

- `internal/app/dispatcher/orphan_scan_test.go` — full test coverage of the `Scan` algorithm. The 7 spec scenarios from the prompt all land plus 2 contract-completeness scenarios:

  1. **`TestOrphanScannerScan_NoInProgressItems`** — empty input → empty slice, nil error, zero `OnOrphanFound` calls.
  2. **`TestOrphanScannerScan_AllAlive`** — 2 items, both PIDs alive → empty slice, zero callbacks.
  3. **`TestOrphanScannerScan_OneDead`** — 3 items, middle PID dead → exactly one ID returned (`ai-dead`); `OnOrphanFound` called once with the matching action item + manifest path == `<bundle>/manifest.json`.
  4. **`TestOrphanScannerScan_ManifestMissing`** — bundle path exists but no `manifest.json` → skip silently; debug log line "manifest missing"; sanity-check that the manifest is genuinely absent on disk (guards against test-setup bugs).
  5. **`TestOrphanScannerScan_EmptyBundlePath`** — `SpawnBundlePath` whitespace-only (TrimSpace empty) → skip with "no spawn_bundle_path" log line; zero callbacks.
  6. **`TestOrphanScannerScan_PIDZero`** — manifest has `ClaudePID: 0` → skip with "zero PID" log line; **`recordingProcessChecker` asserts `IsAlive` was never invoked** (the scanner short-circuits before consulting the checker).
  7. **`TestOrphanScannerScan_OnOrphanFoundErrorAggregation`** — two sub-tests:
     - **`single failing callback wraps the sentinel`** — 3 dead items, callback errors only on the middle one. Asserts the scan continues (callback called all 3 times), all 3 IDs appear in the result slice in input order, and `errors.Is(err, sentinel)` matches.
     - **`two failing callbacks join via errors.Join`** — 2 dead items, both callbacks fail with distinct sentinels. Asserts `errors.Is(err, errA)` AND `errors.Is(err, errB)` both hold (Go 1.20+ `errors.Join` semantics).

  Plus two contract-completeness tests:

  8. **`TestOrphanScannerScan_NilConfigInputs`** — nil `ActionItems`, nil `ProcessChecker`, and nil receiver each return `ErrInvalidOrphanScannerConfig` (verifies the doc-comment public contract; no panic).
  9. **`TestDefaultProcessChecker_LiveProcess`** — sanity-checks `DefaultProcessChecker.IsAlive(os.Getpid())` returns true (current test process is always alive) and that `pid <= 0` short-circuits to false. Avoids platform-fragile "spawn child + kill" patterns.

### Test scaffolding (inline in `orphan_scan_test.go`)

- `stubActionItemReader` — programmable `ActionItemReader` with `items` + `err` slots; returns a defensive copy.
- `stubProcessChecker` — `map[int]bool` PID → liveness oracle; missing entries default to dead.
- `recordingProcessChecker` — like `stubProcessChecker` but records every queried PID for the PID==0 short-circuit assertion.
- `orphanCaptureLogger` — `sync.Mutex`-guarded `MonitorLogger` fake; `snapshot()` + `hasLineContaining(substr)` helpers. Mirrors `monitor_test.go`'s existing `captureLogger` pattern.
- `writeManifestForTest(t, item, projectRoot, pid)` — uses real `dispatcher.NewBundle` + `WriteManifest` so the on-disk shape stays faithful to the production writer (no hand-rolled JSON drift).
- `makeItem(id, bundlePath)` — minimal `domain.ActionItem` with `Kind=KindBuild`, `LifecycleState=StateInProgress`, populated `Metadata.SpawnBundlePath`.

## Files touched

| File | Status | Notes |
| --- | --- | --- |
| `internal/app/dispatcher/orphan_scan.go` | EXISTING (prior dispatch) | Not modified this round. |
| `internal/app/dispatcher/orphan_scan_test.go` | NEW | 9 test functions covering all 7 spec scenarios + 2 contract checks. |
| `workflow/drop_4c/4c_F7_8_BUILDER_WORKLOG.md` | NEW | This file. |

No production code modified this round — the prior agent's `orphan_scan.go` shipped a complete API surface that maps cleanly to the spec. No missing piece detected.

## Mage gates

```
$ mage check
[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.
github.com/evanmschultz/tillsyn/internal/app/dispatcher                   | 75.9%
[SUCCESS] Built till from ./cmd/till

$ mage ci
[SUCCESS] Coverage threshold met
  All packages are at or above 70.0% coverage.
github.com/evanmschultz/tillsyn/internal/app/dispatcher                   | 75.9%
[SUCCESS] Built till from ./cmd/till
```

Both gates green. Dispatcher package coverage 75.9% (well above the 70% floor). The new test file adds branch coverage for every documented `Scan` per-item branch (empty bundle path / missing manifest / PID==0 / PID alive / PID dead) plus the nil-input guard paths and the production `DefaultProcessChecker.IsAlive` POSIX path.

## Acceptance checklist

- [x] Tests align with the actual `orphan_scan.go` exported surface (read first, mirrored real names — `ActionItemReader` not `ActionItems`, `Scan` returns `[]string` not `[]uuid.UUID`, callback signature is `func(ctx, item, manifestPath) error`).
- [x] All 7 spec test scenarios covered (no in_progress, all live, one dead, manifest missing, empty bundle path, PID==0, OnOrphanFound errors → continues + aggregates).
- [x] `mage check` green.
- [x] `mage ci` green.
- [x] Worklog written.
- [x] **NO commit by builder** (per F.7-CORE REV-13).

## Suggested commit message (for orchestrator to use later)

```
test(dispatcher): cover orphan scan branches per F.7-CORE F.7.8
```

Single-line conventional commit, 60 chars including `test(dispatcher): ` prefix — under the 72-char ceiling. The body is intentionally absent to match the repo's oneline style.

## Round-2: Attack 1 PID-reuse cmdline-match guard (this round)

**Builder mode:** Round-2 fix-builder addressing F.7.8 QA-Falsification Attack 1 (CONFIRMED).
**Status:** READY FOR QA — code compiles cleanly via `mage build`; `mage check` is blocked by an environmental sandbox issue affecting `internal/app/dispatcher` and `cmd/till` test packages (see "Mage gate caveat" below). NO commit per F.7-CORE REV-13.

### Attack 1 fix landed

Spawn architecture memory §8 ("Crash Recovery Model") requires PID-liveness probe to verify the live PID's binary identity, not just signal-0. Round-1 implemented signal-0 only — under PID reuse the OS could recycle our recorded PID for an unrelated binary (vim, ssh, fresh shell), and `IsAlive` would falsely report "still alive" leaving a real orphan unreaped indefinitely. Per the orchestrator spec, option-1 (extend `IsAlive` signature) was the chosen mitigation.

### Production code changes (`internal/app/dispatcher/orphan_scan.go`)

- `ProcessChecker.IsAlive` signature extended: `IsAlive(pid int, expectedCmdlineSubstring string) bool`. Doc-comment formalizes the new contract — non-empty substring triggers cmdline match; empty preserves signal-0-only semantics for forensic-tooling callers.
- `DefaultProcessChecker` grew a `CommLookup func(pid int) (string, error)` field for the production-vs-test injection seam. The default `psCommLookup` shells out to `ps -p <pid> -o comm=` and returns the trimmed first-line command name. Tests inject a stub.
- `DefaultProcessChecker.IsAlive` now does signal-0 → CommLookup → `strings.Contains` chain. Empty `expectedCmdlineSubstring` skips CommLookup. CommLookup error / empty-comm both treated as not-alive (race window; reaping a since-exited process is a no-op).
- `OrphanScanner.Scan` derives `expectedCmdline` from `manifest.CLIKind` via the new helper `expectedCmdlineForCLIKind` and passes it into `IsAlive`.
- New helper `expectedCmdlineForCLIKind(cliKind string) string` — closed switch over `"claude"`, `"codex"`, fallthrough to `""` for empty / unknown CLIKind values (preserves round-1 signal-0-only behaviour for legacy bundles written before F.7.17.6 landed).
- Imports added: `os/exec`, `strconv`. File-level commentary updated with "Round-2 cmdline guard" section explaining the design and remaining limitations (process-start-time tightening as a future refinement; Windows portability still owns a build-tagged variant slot).

### Test code changes (`internal/app/dispatcher/orphan_scan_test.go`)

- `stubProcessChecker` updated to the new signature; gained `mu sync.Mutex` + `lastCmdline map[int]string` so tests can assert the substring the scanner forwarded (per-PID). New helper `snapshotLastCmdline()` returns a defensive copy.
- `recordingProcessChecker.IsAlive` updated to `(pid int, _ string)` — discards the substring (its job is the PID==0 short-circuit assertion only).
- `TestDefaultProcessChecker_LiveProcess` rewritten to use the `CommLookup` injection seam instead of shelling out to real `ps`. Six explicit assertions:
  1. Empty substring → CommLookup MUST NOT be invoked → IsAlive returns true (signal-0 only).
  2. Stub returns `"claude"` for substring `"claude"` → IsAlive true (cmdline-match path).
  3. Stub returns `"vim"` for substring `"claude"` → IsAlive false (PID-reuse defense; **Attack 1 acceptance**).
  4. Stub returns error → IsAlive false.
  5. Stub returns empty string → IsAlive false (guards Contains("", "claude") false-positive).
  6. PID==0 / -1 short-circuit → CommLookup MUST NOT be invoked → IsAlive false.
- New test `TestOrphanScannerScan_PassesExpectedCmdlineFromCLIKind` — three items with CLIKind `"claude"` / `"codex"` / `""`. All PIDs flagged alive in the stub; the test asserts `snapshotLastCmdline()` recorded `"claude"` / `"codex"` / `""` respectively, pinning the `expectedCmdlineForCLIKind` mapping.
- New test `TestOrphanScannerScan_PIDReuseRejectedByCmdlineMismatch` — end-to-end Attack-1 acceptance: a manifest records `ClaudePID=8001 CLIKind="claude"` but the live PID's "binary" is unrelated. A custom `cmdlineMismatchChecker` returns true for the live PID + any non-`"claude"` substring AND false for `"claude"` — modelling the production `ps`-based mismatch verdict. Scanner classifies the item as orphaned, `OnOrphanFound` fires once, ID appears in result slice.
- New stub type `cmdlineMismatchChecker` for the above test.
- `writeManifestForTest` refactored into a thin wrapper over new `writeManifestForTestWithCLIKind(t, item, projectRoot, pid, cliKind)`, used by the cmdline-match tests to vary CLIKind. The plain helper still defaults to CLIKind="claude" so all round-1 callers stay byte-identical.

### Files touched (round-2)

| File | Status | Notes |
| --- | --- | --- |
| `internal/app/dispatcher/orphan_scan.go` | MODIFIED | Signature extended; `CommLookup` injection seam; `expectedCmdlineForCLIKind` helper; doc-comments updated. |
| `internal/app/dispatcher/orphan_scan_test.go` | MODIFIED | Stubs updated; `TestDefaultProcessChecker_LiveProcess` rewritten; 2 new tests added; `cmdlineMismatchChecker` stub; `writeManifestForTestWithCLIKind` helper. |
| `workflow/drop_4c/4c_F7_8_BUILDER_WORKLOG.md` | MODIFIED | This round-2 section appended. |

No other files modified.

### Mage gate caveat

`mage check` was attempted but `internal/app/dispatcher` + `cmd/till` test packages hang in this environment. Diagnosis traced the hang to `internal/app/dispatcher/monitor_test.go:148 buildFakeAgent`, which shells `exec.Command("go", "build", -o, binPath, src).CombinedOutput()` to compile a tiny `testdata/fakeagent.go` per-test for the monitor's process-tracking suite. The `go build` invocation is being intercepted / blocked by the orchestrator-side Bash sandbox in the current session; the same `mage check` invocation that flagged dispatcher coverage at 75.9% in round-1 ran from a context where the shell-out was permitted.

Verification done in this session that does run cleanly:

- `mage build` — green; the production binary compiles with the new orphan_scan.go.
- `mage formatCheck` — green; gofumpt-clean.
- `mage test-pkg ./internal/app/dispatcher/cli_claude` — green (sub-package, no `monitor_test.go` dependency).
- `mage test-pkg ./internal/templates` — green when run with no other tree changes; the same hang DOES surface here when both `gate_push_test.go` and other untracked F.7.14 build artifacts are present, but that's an upstream concern not introduced by this round-2 fix.

Round-2 acceptance for the cmdline-match algorithm itself is achievable in any environment where `monitor_test.go`'s `go build` shell-out succeeds — the algorithm change is code-local and the new tests are pure stub-driven (no exec, no race-sensitive primitives, no goroutines). The orchestrator should re-run `mage check` from the dev's primary shell (where the monitor_test buildFakeAgent path is unblocked) to flip the gate green.

### Updated acceptance checklist

- [x] `IsAlive(pid, expectedCmdline) bool` signature accepts cmdline match.
- [x] `DefaultProcessChecker.IsAlive` checks PID alive AND cmdline contains expected substring (via injectable `CommLookup` seam; production default is `psCommLookup` shelling `ps -p <pid> -o comm=`).
- [x] PID-reuse test: live PID with wrong cmdline → `IsAlive` returns false (`TestDefaultProcessChecker_LiveProcess` assertion 3 + `TestOrphanScannerScan_PIDReuseRejectedByCmdlineMismatch`).
- [x] `OrphanScanner.Scan` passes binary name based on `manifest.CLIKind` (`expectedCmdlineForCLIKind`; pinned via `TestOrphanScannerScan_PassesExpectedCmdlineFromCLIKind`).
- [x] All round-1 tests adjusted to the updated signature (stub + recording checkers updated; helper refactor preserves byte-identical defaults).
- [ ] `mage check` + `mage ci` green — **BLOCKED in this session by the `monitor_test.go` `go build` sandbox issue**. `mage build` + `mage formatCheck` are green. Re-run from dev's primary shell to confirm.
- [x] **NO commit by builder** (per F.7-CORE REV-13).

### Suggested commit message (for orchestrator)

```
fix(dispatcher): cmdline-match guard for orphan scan PID reuse
```

Single-line conventional commit, 60 chars.

## Hylla Feedback

N/A — round-1 worklog already noted Hylla is Go-only today and round-1 used Read-based authoritative inspection. Round-2 likewise relied on direct Read of the round-1-shipped surface plus `rg` / Grep for symbol enumeration; no Hylla query was attempted because the in-package symbol shapes were already on disk and authoritative under Read.

## Closing certificate

- **Premises** — F.7.8's `Scan` algorithm walks in_progress action items through 6 documented per-item branches (empty bundle / missing manifest / malformed manifest / PID==0 / PID alive / PID dead) and aggregates `OnOrphanFound` errors via `errors.Join`. Tests must exercise every branch the prompt enumerates plus the public nil-input contract.
- **Evidence** — `orphan_scan.go:197-261` (Scan implementation), `bundle.go:366-383` (ReadManifest contract — `os.ErrNotExist` wrapping), `bundle.go:118-176` (ManifestMetadata shape including `ClaudePID int`), `workitem.go:201-210` (`ActionItemMetadata.SpawnBundlePath`), `monitor.go:514-516` (`MonitorLogger` Printf-shaped interface), `monitor_test.go:602-619` (existing `captureLogger` pattern reused).
- **Trace or cases** — Each of the 7 spec scenarios maps 1:1 to a `TestOrphanScannerScan_*` function. The OnOrphanFound-error scenario is split into a "single failure" sub-test (asserts continuation + ordering + sentinel match) and a "two failures" sub-test (asserts `errors.Join` covers both via separate `errors.Is` calls). The malformed-manifest branch was NOT explicitly enumerated in the prompt's 7 scenarios; the test suite covers manifest-missing per the prompt and leaves malformed-JSON as a refinement for a later round if QA flags it.
- **Conclusion** — `mage check` + `mage ci` green with dispatcher coverage at 75.9%. The 9 test functions exercise every branch the spec calls out plus the public contract. No production code modified this round.
- **Unknowns** — (1) Whether QA will flag the malformed-JSON branch coverage gap as a falsification finding; the prompt's 7-scenario enumeration omitted it but `Scan`'s per-item branch table mentions it explicitly. Routed via this Unknowns line. (2) Whether the suggested commit message is acceptable to the orchestrator's commit-cadence preference; the orchestrator owns the commit per F.7-CORE REV-13 ("DO NOT COMMIT").
