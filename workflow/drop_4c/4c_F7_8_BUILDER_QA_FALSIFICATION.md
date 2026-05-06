# F.7.8 — Builder QA Falsification Review

**Droplet:** F.7-CORE F.7.8 — orphan scan API.
**Files reviewed:**
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/orphan_scan.go` (production)
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/orphan_scan_test.go` (tests)
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/workflow/drop_4c/4c_F7_8_BUILDER_WORKLOG.md`
- Spec: `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/workflow/drop_4c/F7_CORE_PLAN.md` §F.7.8 (lines 542–591)
- Bundle/Manifest contract: `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/bundle.go` (ManifestMetadata at L118–176, ReadManifest at L366–383)
- F.7.9 metadata field: `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/domain/workitem.go` L201–210
- Adapter contract: `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_adapter.go` L51–83

**Sibling carve-out:** F.7.14 attribution explicitly avoided. All findings are against shipped F.7.8 code or the F.7.8 acceptance contract.

---

## 1. Attack Walkthrough

### Attack 1 — PID reuse (CONFIRMED — counterexample reproducible by spec read)

**Premises.** Spec (`F7_CORE_PLAN.md` L566) requires liveness check via `os.FindProcess` + `process.Signal(syscall.Signal(0))` **plus cmdline match against `claude` binary path**. Spec L578 explicitly enumerates the edge case: "cmdline mismatch (PID alive but reused by unrelated process) → treated as dead." Spec L584 lists "cmdline mismatch detection prevents PID-reuse false-positives" as a falsification mitigation to bake in. Same requirement in `SKETCH.md` L139.

**Evidence.** `orphan_scan.go` L113–132 (`DefaultProcessChecker.IsAlive`) implements signal-0 ONLY. The doc-comment at L48–57 explicitly disclaims cmdline match as a "refinement" and "known limitation," contradicting the acceptance criterion's explicit inclusion.

**Trace.**
1. Dispatcher process A spawns claude subagent at PID 12345; manifest records `claude_pid: 12345`, `cli_kind: "claude"`.
2. Dispatcher A dies. PID 12345 dies. OS reuses 12345 for an unrelated process B (e.g. `ssh`, `vim`).
3. New dispatcher restarts. Orphan scan reads manifest, calls `DefaultProcessChecker.IsAlive(12345)`. signal-0 returns nil (B is alive). Scanner concludes "still alive; skipping."
4. Real orphan never reaped. Action item stuck `in_progress` forever (or until next PID reuse coincidentally hits a dead PID).

**Conclusion.** CONFIRMED counterexample against an explicit spec acceptance criterion + an explicit spec falsification-mitigation requirement. Builder reduced this to a "known limitation" inside doc-comments without an authorizing REV in `F7_CORE_PLAN.md` REV-5 (the only F.7.8 revision, which only addressed the F.7.17.6 cli_kind dependency, not the cmdline-match requirement).

**Mitigation pathway (informational).** A correct shipping shape is one of:
- (a) Builder adds a cmdline-match step inside `DefaultProcessChecker` — read `/proc/<pid>/comm` (Linux) or `ps -o command= -p <pid>` (macOS) and substring-match `claude`.
- (b) Builder adds a planner-acknowledged REV-record in `F7_CORE_PLAN.md` deferring cmdline match to a follow-up droplet, with rationale.
Today neither (a) nor (b) is in the tree.

**Side question raised in prompt — "Verify if memory §8 cmdline-match was supposed to be implemented."** Spec L566 + L578 + L584 are all unambiguous: cmdline-match is in the acceptance criteria, the test-scenario list, AND the falsification-mitigations-to-bake-in list. The builder's reading of "memory §8 accepts this race as a known limitation" does not match the planner's actual acceptance criteria. No REV walks back the requirement.

**Verdict:** CONFIRMED.

---

### Attack 2 — `OnOrphanFound` callback never returning (REFUTED)

**Premises.** Scanner contract: callback runs synchronously inside the scan loop (`orphan_scan.go` L249). A blocking callback would stall the entire scan.

**Evidence.** L213–215 — the loop checks `ctx.Err()` between items, but NOT mid-callback. Callback execution is not ctx-aware (the scanner passes `ctx` through to the callback at L249, but does not enforce timeout).

**Trace.** A pathological caller that never returns from `OnOrphanFound` will hang `Scan` forever.

**Conclusion.** REFUTED as a falsification finding. The scanner's doc-comment at L196 explicitly states "callback execution is the caller's concern." This is documented contract. A blocking callback is the caller's bug, not the scanner's. The contract IS the timeout policy: caller owns ctx-respecting behavior, and the scanner already passes ctx through. There is no spec requirement (acceptance criteria L562–571) for the scanner to enforce a callback timeout. Adding one would be YAGNI.

**Verdict:** REFUTED — documented caller contract.

---

### Attack 3 — Malformed-JSON manifest branch (REFUTED, with NIT)

**Premises.** Spec (L573–580) lists six edge cases. Malformed-JSON is NOT in that list — only "manifest.json missing" is. But `orphan_scan.go` doc-comment L181–185 promises a malformed-JSON branch.

**Evidence.** `orphan_scan.go` L224–232 — the production code DOES correctly handle malformed JSON: `ReadManifest` returns a `decode manifest` error wrapping `json.Unmarshal` (per `bundle.go` L379–380); `errors.Is(err, os.ErrNotExist)` is false on that path; the scanner falls into the warning-log + continue branch at L230. `Scan` does not halt — it skips and logs. Behavior is correct.

**Test coverage.** No test exercises this branch. `orphan_scan_test.go` L266–303 covers manifest-missing only. Builder's worklog Unknowns line acknowledges this gap.

**Conclusion.** Production behavior is correct ("skip+log, do not halt"); the gap is test coverage only. The spec did NOT require a malformed-JSON test, and the builder explicitly raised the gap in their Unknowns. This is a legitimate NIT — adding a 15-line test using `os.WriteFile(manifestPath, []byte("{not json"), 0o600)` would close the loop — but it's not a falsification of the production claim.

**Verdict:** REFUTED, with NIT.

---

### Attack 4 — Concurrent scans on the same scanner (REFUTED)

**Premises.** `OnOrphanFound` could be called concurrently across two `Scan` invocations. The scanner has no internal locking.

**Evidence.** `orphan_scan.go` L138–142 (struct doc-comment): "Concurrent calls to Scan on the same scanner are not safe — the scanner has no internal locking — but callers do not need locking today because Scan runs exactly once per dispatcher startup."

**Trace.** Two goroutines calling `(s *OrphanScanner).Scan(ctx)` on the same `*OrphanScanner` would race on `orphans` and `callbackErrs` slices. Concurrent appends to a slice in Go without synchronization is a data race (unsafe; `-race` would flag).

**Conclusion.** Documented as not-safe-for-concurrent-use, with a justified caller-contract explanation. The dispatcher startup hook (out-of-scope per builder's "SCOPE" comment at L42–46) calls `Scan` exactly once. Adding a sync.Mutex would be premature given the documented single-shot lifecycle. REFUTED — explicit doc-comment contract.

**Verdict:** REFUTED — documented single-use contract.

---

### Attack 5 — `bundle.path` / `metadata.spawn_bundle_path` extraction routing (REFUTED)

**Premises.** Spec (L564) requires reading `metadata.spawn_bundle_path`. F.7.9 owns the field. F.7.8 reads it.

**Evidence.**
- `internal/domain/workitem.go` L201–210: `ActionItemMetadata.SpawnBundlePath string \`json:"spawn_bundle_path,omitempty"\`` — F.7.9-owned.
- `orphan_scan.go` L218: `bundlePath := strings.TrimSpace(item.Metadata.SpawnBundlePath)` — direct read off the metadata struct.
- Trim-empty fallthrough at L219–222: skip with debug log. Matches spec L575 ("`metadata.spawn_bundle_path` empty → log + leave alone").

**Conclusion.** Read path is correct, points at the right field, handles whitespace and empty per spec. REFUTED.

**Verdict:** REFUTED.

---

### Attack 6 — Adapter routing absent; generic `ProcessChecker` instead (CONFIRMED — design deviation)

**Premises.** Spec L566: "If `claude_pid > 0`, route to `adapterRegistry.Get(manifest.CLIKind).IsPIDAlive(pid)` — claude adapter's check uses [...]. Codex adapter's check is similar (Drop 4d)." Spec acceptance criteria L562: `OrphanScan(ctx, repo ActionItemRepo, adapterRegistry CLIAdapterRegistry) error`.

**Evidence.**
- `orphan_scan.go` L143–169 — `OrphanScanner` struct has `ActionItemReader`, `ProcessChecker`, `Logger`, `OnOrphanFound`. No `CLIAdapterRegistry` field. No adapter dispatch.
- `cli_adapter.go` L61–83 — `CLIAdapter` interface has exactly 3 methods (BuildCommand, ParseStreamEvent, ExtractTerminalReport) per F.7.17 locked decision L10. **No `IsPIDAlive` method.**
- `ManifestMetadata.CLIKind` (`bundle.go` L146) IS preserved on the on-disk manifest (per F.7.17.6) but `orphan_scan.go` reads `manifest.ClaudePID` only and ignores `manifest.CLIKind` at L240 — never routes off it.

**Trace.** When Drop 4d adds codex adapter, the orphan scan must distinguish "claude PID liveness" from "codex PID liveness" because the codex spawn's PID is recorded under a different process tree. Today's signal-0 check on `manifest.ClaudePID` will work for codex bundles too only by coincidence (PID is PID) — not by design. The codex liveness check might want different semantics (e.g. ssh-tunneled codex, where local PID is meaningless and the check should probe a remote endpoint instead). The shipped surface forces a future builder either to add `IsAlive` to `CLIAdapter` (violates F.7.17 L10's 3-method lock) OR to refactor `OrphanScanner` to take a registry (re-shapes today's API). Either way the current shape is a known future-rework hotspot.

**Conclusion.** This is a real design deviation from spec. The builder's defense is implicit in the file-level comment at L33–40: "today every recorded manifest is `claude` so the scanner uses a uniform PID-liveness check, but the field is preserved on the ManifestMetadata payload so future codex bundles route adapter-specific liveness logic without a scanner-API change." That explanation is internally inconsistent — preserving the field on disk does NOT enable adapter routing without an API change, because the scanner has no place to plug a per-adapter checker today.

**However:** the F.7.17 L10 lock (3-method `CLIAdapter` interface) is genuinely binding on F.7.8. Adding `IsPIDAlive` to `CLIAdapter` would require an F.7.17 REV — outside F.7.8's scope. So the builder's choice (generic `ProcessChecker`, defer per-adapter routing to Drop 4d) is defensible AS LONG AS the planner acknowledged the deviation. There is no such REV in `F7_CORE_PLAN.md`.

This is a CONFIRMED finding but its severity is "spec-vs-architecture-conflict that should have been REV'd by the planner before the builder shipped." The orchestrator's choice: route this back to the planner for an explicit REV (preferred) or accept the as-shipped shape with an inline doc-pointer to the future Drop 4d adapter-routing refactor.

**Verdict:** CONFIRMED — design deviation from spec without an authorizing REV. The simpler shape DOES miss something (codex adapter routing), and the deferral is undocumented at the planner level.

---

### Attack 7 — Memory rule conflicts (REFUTED)

**Premises.** No Hylla query, no commit, no migration logic.

**Evidence.**
- Worklog `## Hylla Feedback` section (L92–94): "N/A — task touched non-Go evidence-gathering only via direct file Reads." Correct stance; F.7-era Go is not in the current Hylla snapshot (verified: `hylla_search_keyword` for `SpawnBundlePath` returned 0 hits at snapshot 5). NIT: the phrasing "Hylla is Go-only today, and the task was test authoring" is awkward (the task IS Go); the intent is "the in-package symbols I needed were authoritative via Read of files in the same package."
- Worklog L82: "**NO commit by builder** (per F.7-CORE REV-13)." Compliant.
- No migration logic in `orphan_scan.go` (no `till migrate`, no SQL). Compliant.
- No `mage install` invocation. No raw `go test` / `go build` / `go vet`. Worklog cites `mage check` + `mage ci`.

**Conclusion.** REFUTED.

**Verdict:** REFUTED.

---

## 2. Side findings (NITs that are not falsifications)

- **N1 — `DefaultProcessChecker` is POSIX-only and silently degrades on Windows.** Doc-comment L58–61 acknowledges this. Pre-MVP rule says "Tillsyn doesn't run on Windows." Defensible.
- **N2 — `recordingProcessChecker` is declared *after* its first use** (`orphan_scan_test.go` L355 references it; declaration at L556). Compiles fine in Go (package-scope declarations are unordered) but reads awkwardly. Recommend moving the declaration up next to `stubProcessChecker` for readability.
- **N3 — `var _ = time.Now` at L575** is a defensive stub for a removed assertion. Either delete the import + the stub OR add the actual assertion. Today it's noise.
- **N4 — Test coverage gap for malformed-JSON branch.** See Attack 3.
- **N5 — `mage test-pkg ./internal/app/dispatcher` ran for 11 min and was killed with "test killed with quit: ran too long"** in this QA round (output captured: `0 test failures and 0 build errors across 1 package`, killed by mage's 11m guard). The hang is NOT in the F.7.8 test file (no infinite loops, no unbounded channels visible; each test uses isolated `t.TempDir`). This is a pre-existing dispatcher-package symptom unrelated to F.7.8. Worklog claimed `mage check + mage ci` green; that may have been at a different moment / with different parallel scheduling. Flagging for the orchestrator — not a F.7.8 falsification but should be triaged before drop closeout.

---

## 3. Per-attack verdict matrix

| # | Attack | Verdict |
|---|---|---|
| 1 | PID reuse (cmdline-match missing) | **CONFIRMED** |
| 2 | `OnOrphanFound` blocking | REFUTED — documented caller contract |
| 3 | Malformed-JSON branch | REFUTED with NIT (test coverage gap) |
| 4 | Concurrent scans | REFUTED — documented single-use |
| 5 | `bundle.path` / metadata read path | REFUTED |
| 6 | Adapter routing absent | **CONFIRMED** — design deviation, no REV |
| 7 | Memory-rule conflicts | REFUTED |

## 4. Final verdict

**NEEDS-REWORK** — two CONFIRMED counterexamples (Attacks 1 and 6) point at acceptance-criteria items the builder downgraded to "known limitations" or implicit deferrals without authorizing REVs in `F7_CORE_PLAN.md`. The orchestrator has two routes:

- **Route A (preferred):** Send the planner a falsification-driven REV request to either (a) walk back the cmdline-match + adapter-routing acceptance criteria for F.7.8 with explicit deferral to a Drop 4d droplet, OR (b) keep the requirements and route a builder respawn to add cmdline-match + a path for adapter routing (likely via an additional `OrphanScanner` field that does NOT touch the F.7.17 L10 3-method lock — e.g. a `func(cliKind CLIKind, pid int) bool` knob).
- **Route B:** Accept Attack 6 with an inline TODO + Drop-4d-routing pointer; respawn builder to add cmdline-match (Attack 1) since that one is in scope today and well-bounded (~25 lines in `DefaultProcessChecker`).

Plus N5 (test-suite hang) needs triage — the worklog's mage-ci-green claim doesn't match this round's measurement.

---

## TL;DR

- T1 — Attack walkthrough: 7 attacks executed, 2 CONFIRMED (PID reuse cmdline-match missing, adapter routing absent).
- T2 — Side NITs: 5 (Windows-only, decl ordering, dead var, malformed-JSON test gap, dispatcher test-suite hang).
- T3 — Verdict matrix: 2 CONFIRMED / 5 REFUTED.
- T4 — Final: NEEDS-REWORK; route via planner REV or builder respawn (cmdline-match + adapter-routing seam).
