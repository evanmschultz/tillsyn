# F.7.8 — Builder QA Falsification Review (Round 2)

**Droplet:** F.7-CORE F.7.8 — orphan scan API.
**Round 1 verdict:** NEEDS-REWORK with 2 CONFIRMED (Attack 1 — PID reuse cmdline-match missing; Attack 6 — adapter-routing absent).
**Round 2 scope:** verify both fixes (Attack 1 mitigation; Attack 6 acceptance via REV-16) are airtight, hunt for new counterexamples introduced by the round-2 changes.

**Files reviewed:**

- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/orphan_scan.go` (production, post-round-2)
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/orphan_scan_test.go` (tests, post-round-2)
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/workflow/drop_4c/4c_F7_8_BUILDER_WORKLOG.md` (round-2 section appended)
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/workflow/drop_4c/F7_CORE_PLAN.md` REV-16 (lines 1064–1070)
- `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/internal/app/dispatcher/cli_adapter.go` lines 51–83 (3-method CLIAdapter lock)

---

## 1. Per-attack re-audit

### Attack 1 (round-1 CONFIRMED) — PID reuse via signal-0 alone — **NOW REFUTED**

**Premises (re-validated).** Spec L566 + L578 + L584 require cmdline-match against the `claude` binary path to defeat PID-reuse false-positives.

**Evidence.**

- `orphan_scan.go` L105–121 — `ProcessChecker.IsAlive(pid int, expectedCmdlineSubstring string) bool` — signature extended.
- `orphan_scan.go` L159–177 — `DefaultProcessChecker.CommLookup` injection seam plus `psCommLookup` shelling `ps -p <pid> -o comm=`.
- `orphan_scan.go` L182–229 — IsAlive does signal-0 → `CommLookup` → `strings.Contains(comm, expectedCmdlineSubstring)`. Empty substring opt-out at L203–209. Lookup error / empty-comm both treated as not-alive (L216–227).
- `orphan_scan.go` L337 — Scan derives substring via `expectedCmdlineForCLIKind(manifest.CLIKind)`.
- `orphan_scan.go` L383–392 — closed switch over `claude` / `codex` / fallthrough to `""`.
- Tests: `TestDefaultProcessChecker_LiveProcess` (L697–766) covers six branches (empty substring → no lookup; match → true; mismatch "vim" → false; lookup error → false; empty comm → false; pid <= 0 → no lookup, false). `TestOrphanScannerScan_PIDReuseRejectedByCmdlineMismatch` (L312–351) is the end-to-end Attack-1 acceptance. `TestOrphanScannerScan_PassesExpectedCmdlineFromCLIKind` (L255–295) pins the CLIKind→substring mapping.

**Trace.** Original counterexample: dispatcher recorded PID 12345, OS reused PID for `vim` post-restart, signal-0 returned nil, scanner falsely concluded "alive". With the round-2 fix:

1. Scanner calls `IsAlive(12345, "claude")` (substring derived from `manifest.CLIKind == "claude"`).
2. `DefaultProcessChecker.IsAlive` does signal-0 (returns nil — `vim` is alive).
3. Falls through to `CommLookup`. Production `psCommLookup` returns `"vim"`.
4. `strings.Contains("vim", "claude")` → false. IsAlive returns false.
5. Scanner classifies item as orphan, fires `OnOrphanFound`.

The round-1 counterexample no longer reaches the false-positive branch.

**Verdict:** REFUTED — original Attack-1 counterexample no longer reproduces.

---

### Attack 6 (round-1 CONFIRMED) — Adapter routing absent — **NOW REFUTED via REV-16**

**Evidence.**

- `F7_CORE_PLAN.md` REV-16 (lines 1064–1070) authors the deferral: "F.7.8 acceptance criteria L562 / L566 originally specified `OrphanScan(ctx, repo, adapterRegistry CLIAdapterRegistry)` routing PID-liveness through `adapterRegistry.Get(manifest.CLIKind).IsPIDAlive(pid)`. F.7.17.10's `CLIAdapter` interface is locked to three methods … The two specs are mutually inconsistent."
- Resolution: "F.7.8 ships generic `ProcessChecker` interface + `DefaultProcessChecker`. … Future codex adapter (Drop 4d) introduces a per-CLIKind ProcessChecker registry … without changing the OrphanScanner API."
- Rationale: "Adding `IsPIDAlive` (which is OS-level POSIX semantics, not CLI-specific) would conflate concerns."
- Architectural fact: `cli_adapter.go` L52–83 — CLIAdapter interface locked to exactly `BuildCommand`, `ParseStreamEvent`, `ExtractTerminalReport` per F.7.17 L10. No `IsPIDAlive`.
- Forward-compat hook present: `expectedCmdlineForCLIKind` (`orphan_scan.go` L383–392) already routes off `manifest.CLIKind`. The Drop-4d codex adapter only needs to add a `case "codex": return "codex"` arm (already shipped) — and, if codex liveness needs different OS semantics (e.g. ssh-tunneled), to introduce a registry without touching today's API.

**Verdict:** REFUTED — REV-16 cleanly authorizes the deferral; CLIAdapter L10 lock is preserved; OrphanScanner API stays adapter-routing-friendly because `manifest.CLIKind` is already plumbed into both `expectedCmdlineForCLIKind` and the `OnOrphanFound` callback context (callback receives `domain.ActionItem`; the manifest is on disk at `manifestPath` if needed).

---

### Attack R2-A — Substring-contains false-positive on `claude-installer` / `not-claude` (CONFIRMED)

**Premises.** The cmdline match is `strings.Contains(comm, "claude")`. POSIX `ps -p <pid> -o comm=` returns "the value of argv[0] as a string" (POSIX.1-2017 ps utility); on macOS this is the basename of argv[0], on Linux it's truncated to ~16 bytes (`/proc/<pid>/comm`).

**Evidence.** `orphan_scan.go` L228: `return strings.Contains(comm, expectedCmdlineSubstring)`. Substring containment, not equality, not whole-word match.

**Trace (counterexample).** PID 12345 is reused by an unrelated binary whose `comm` field contains the string "claude" as a substring:

1. `claude-installer` — a hypothetical installer binary the user ran. `comm = "claude-installer"`. `strings.Contains("claude-installer", "claude")` → **true**. IsAlive returns true. Real orphan NOT reaped.
2. `not-claude` — an adversarial / coincidental binary. `comm = "not-claude"`. `strings.Contains("not-claude", "claude")` → **true**. IsAlive returns true. Real orphan NOT reaped.
3. `claude-helper`, `claudette`, `claudemod`, etc. — every binary whose basename contains "claude" anywhere triggers the false-positive.

This is a narrowed-but-still-present version of round-1 Attack 1 — strict-equality (or basename-equality) would close it; substring-contains keeps the door cracked.

**Severity.** Lower than round-1 PID-reuse via signal-0-only (which was a 100% false-positive on every reused PID). The round-2 surface narrows the false-positive to the subset of reused PIDs whose new binary's `comm` happens to contain "claude" as a substring — still possible but vastly narrower, and on a typical dev / production machine the population of "claude*" binaries is ~1 (the actual `claude` binary).

**Why this matters anyway.** The orchestrator prompt explicitly asked: *"cmdline contains 'claude' but is actually `claude-installer` or `not-claude` — does substring contains accept it? Verify TrimSpace + Contains semantics."* The answer is yes — `strings.Contains` accepts substring anywhere in the string, including prefix-extended (`claude-installer`) and suffix-extended (`not-claude`) variants. The doc-comment at L138–148 acknowledges the "trimmed command name contains expectedCmdlineSubstring" semantics but does NOT flag the substring-extension false-positive class.

**Mitigation pathway (informational).** A tighter shape uses `strings.TrimSpace(comm) == expectedCmdlineSubstring` (whole-name equality) OR `strings.HasPrefix(comm, expectedCmdlineSubstring)` followed by a delimiter check (so `claude` matches `claude` and `claude-1.2.3` but not `claude-installer`). Tightest: a closed-set lookup keyed off `manifest.CLIKind` returning a strict allowlist (e.g. `{"claude", "claude-cli"}` for CLIKind="claude").

**Verdict:** CONFIRMED — substring-contains semantics admit a narrow class of PID-reuse false-positives the round-2 fix did not close. NIT-class severity given the narrow false-positive population, but it IS a counterexample.

---

### Attack R2-B — Race window between signal-0 and CommLookup (REFUTED — documented + correctly handled)

**Premises.** Between the signal-0 probe (`proc.Signal(syscall.Signal(0))`) and the `CommLookup` shellout, the live PID could exit and a third PID could be allocated.

**Evidence.** `orphan_scan.go` L216–221: `CommLookup` returning an error is treated as not-alive ("Most common cause: PID exited between signal-0 and the lookup. Treat as dead — reaping a since-exited process is a no-op."). Doc-comment L67–70 (file-level) also flags the start-time-tightening alternative as a future refinement.

**Trace.** PID 12345 alive at signal-0 → exits between signal-0 and `ps` → `ps` returns ESRCH → `psCommLookup` returns ("", err) → IsAlive returns false → scanner reaps. The reap is correct (the original PID's process is dead). If the OS allocates PID 12345 to a fresh `claude` process between `ps` invocations on a busy system, the next scan cycle will catch it, and meanwhile the orphan reap of the FIRST item is correct (its monitoring goroutine is dead either way).

**Verdict:** REFUTED — race is acknowledged, handled by treating lookup-error as not-alive, and produces the correct outcome (reap the orphan).

---

### Attack R2-C — Empty cmdline output (process between exec and comm write) (REFUTED — handled at L222–227)

**Premises.** A process between `fork` and `exec` (or between `exec` and the kernel's comm write on Linux) could have an empty `/proc/<pid>/comm` or `ps` could return empty output.

**Evidence.** `orphan_scan.go` L222–227: empty `comm` is treated as not-alive ("Empty output should not happen for a valid live process; defending against it keeps Contains('', 'claude') from returning a false-positive."). `strings.Contains("", "claude")` is false anyway, so the explicit guard is belt-and-suspenders — but this prevents a weird OS-corner-case where `ps` returns empty for a transient race.

A separate worry: `strings.Contains(s, "")` returns true for any `s`. In our path the substring is derived from `expectedCmdlineForCLIKind` and is non-empty for `claude` / `codex` and empty for `""`. Empty substring opts out via the L203 short-circuit BEFORE reaching `strings.Contains` — so the `strings.Contains(comm, "")` trapdoor is unreachable. Verified.

**Test coverage.** `TestDefaultProcessChecker_LiveProcess` empty-comm assertion (L746–751) pins the L222–227 behavior.

**Verdict:** REFUTED — empty-comm path handled.

---

### Attack R2-D — macOS vs Linux `ps -p <pid> -o comm=` portability (REFUTED with sub-NIT)

**Premises.** Production `psCommLookup` (L172) shells `ps -p <pid> -o comm=`. Per POSIX.1-2017 (verified via Context7 / pubs.opengroup.org/onlinepubs/9699919799/utilities/ps): the `-o` option provides format control and the `comm` field is "the name of the command being executed, which is the value of argv[0] as a string."

**Evidence.**

- macOS `ps -o comm=` is BSD-flavored; returns the full pathname or basename of argv[0] depending on the kernel's notion (typically the basename for binaries on PATH, full path otherwise). Either form contains "claude" for a `claude` binary.
- Linux `ps -o comm=` returns `/proc/<pid>/comm`, kernel-truncated to TASK_COMM_LEN=16 bytes (15 chars + NUL). The string "claude" fits well under 15 chars; not a truncation concern.
- Both implementations treat `-o comm=` as suppressing the header (per POSIX: "If the header text is null, … if all header text fields are null, no header line is written").

The shellout is portable. Pre-MVP rule "Tillsyn doesn't run on Windows" makes the POSIX-only restriction acceptable.

**Sub-NIT.** A future macOS-on-Apple-Silicon dev running a `claude` binary installed via npm at `/opt/homebrew/lib/node_modules/@anthropic-ai/claude-code/cli.js` invoked via a `claude` shim will have `comm = "node"` if the shim is `#!/usr/bin/env node`. Then `strings.Contains("node", "claude")` → false → false-orphan-reap of a HEALTHY claude spawn. This is a real risk: the Anthropic claude CLI shipped on dev machines today is precisely a node shim. The dispatcher's spawn path uses `exec.Command("claude", …)` (or whatever the resolver returns), so the actual ps comm value depends on whether the binary execs node directly (case 1) or replaces argv[0] (case 2).

The Drop 4c F.7.17.3 claude argv assembly emits `claude --headless ...` and the CLI binary itself is the entry point — but on macOS via npm, `which claude` resolves to a shell script that ultimately calls node, so PID's comm could be "node". This warrants a manual verification via `dispatcher run` + `ps -p <pid> -o comm=` on the dev's machine. Logging the round-1 false-positive flip without verifying the round-2 false-negative case is a gap.

**Verdict:** REFUTED for the portability question proper; flagged as a sub-NIT for the npm-claude-shim ps comm value (whether real claude spawns return "claude" or "node" or "sh" depends on the install method — needs empirical verification before next dispatcher dogfood). Logged below in §2.

---

### Attack R2-E — psCommLookup `exec.Command` PATH dependency (REFUTED — doc-commented)

**Premises.** `exec.Command("ps", …)` invokes Go's `exec.LookPath("ps")` which consults `PATH`. If the dispatcher inherits a malformed `PATH` (or PATH shadowing), `ps` could resolve to a non-`ps` shim.

**Evidence.** `orphan_scan.go` L155–158 explicitly addresses this: "ps is required on every supported POSIX system at /bin/ps and /usr/bin/ps; Go's exec.LookPath ('ps') consults the inherited PATH. Tillsyn's dispatcher inherits the dev's shell PATH so this is not a portability risk in practice."

**Verdict:** REFUTED — documented contract; pre-MVP risk acceptance is reasonable.

---

### Attack R2-F — `expectedCmdlineForCLIKind` empty-string fallthrough silently downgrades to signal-0-only (REFUTED — documented + correct semantics)

**Premises.** Legacy bundles written before F.7.17.6 landed `manifest.CLIKind` will have `CLIKind == ""`. The switch (L383–392) returns `""` for unknown CLIKinds. `IsAlive` then takes the empty-substring opt-out path (L203–209). Result: legacy bundles inherit round-1 (signal-0-only) semantics, including the original Attack-1 vulnerability.

**Evidence.** Doc-comment at L376–382 explicitly addresses this: "Unknown CLIKind values (including the empty string from older manifests written before F.7.17.6) fall back to the empty string, which DefaultProcessChecker interprets as 'skip cmdline check, signal-0 only' — preserving round-1 behaviour for legacy bundles rather than mis-reaping a healthy spawn that happens to predate the guard."

**Trade-off.** Two failure modes possible for legacy bundles:

- **Strict (cmdline-match required, `""` substring rejected):** legacy bundles ALL get reaped on dispatcher restart because no comm field can match `""` strictly. Bad — mis-reaps healthy spawns.
- **Lax (current shipped — fallthrough to signal-0 only):** legacy bundles get round-1 semantics. Bad — Attack 1 PID-reuse false-positive still hits legacy bundles.

The shipped choice (lax) is the safer side: Tillsyn's dogfood manifests after F.7.17.6 ALL set `CLIKind="claude"` (verified at `bundle.go` L146 — `ManifestMetadata.CLIKind` is populated by every spawn since F.7.17.6). The legacy-bundle window is bounded by "between F.7.17.6 landing and the next dispatcher restart that processes legacy bundles" — narrow and self-extinguishing.

**Verdict:** REFUTED — the lax fallthrough is the documented + correct trade-off given the bounded legacy window. Could be revisited as a refinement if production accumulates long-lived legacy bundles.

---

### Attack R2-G — REV-16 doc cleanly documents the deferral (REFUTED — REV-16 well-formed)

**Evidence.** Re-quoted from F7_CORE_PLAN.md L1064–1070:

- Title: "REV-16 — F.7.8 adapter-routing surface deferred to Drop 4d (codex)"
- Names the original spec lines (L562 / L566).
- Identifies the conflict (F.7.17.10 3-method lock vs adapterRegistry.Get(...).IsPIDAlive).
- States the resolution (generic ProcessChecker, manifest.CLIKind plumbed, future per-CLIKind registry in Drop 4d).
- Explains "why not extend CLIAdapter" (OS-level POSIX semantics, not CLI-specific concern).

The REV is structurally consistent with REV-5, REV-9, REV-13, REV-14, REV-15 (all in the same file). Reads cleanly.

**Verdict:** REFUTED — REV-16 is well-formed and authorizing.

---

### Attack R2-H — Memory-rule conflicts (REFUTED)

**Evidence.**

- Round-2 worklog `## Hylla Feedback` (L167–169): "N/A — round-1 worklog already noted Hylla is Go-only today and round-1 used Read-based authoritative inspection. Round-2 likewise relied on direct Read of the round-1-shipped surface plus `rg` / Grep for symbol enumeration; no Hylla query was attempted because the in-package symbol shapes were already on disk and authoritative under Read." Compliant with the Hylla-only-Go memory rule (noted in worklog) and with the Hylla feedback closing-comment requirement.
- Worklog L82 + L156–157: "**NO commit by builder** (per F.7-CORE REV-13)." Compliant.
- No migration logic; no `mage install`; no raw `go test` / `go build` / `go vet`.
- Substantive response Section 0 not written into Tillsyn artifacts (per `feedback_section_0_required.md` Tillsyn-boundary rule — applies to orchestrator-facing responses only).

**Verdict:** REFUTED.

---

## 2. Side findings (NITs / forward-watches)

- **N1 — Round-2 false-positive narrowing (Attack R2-A).** Substring-contains accepts `claude-installer` / `not-claude`. Tighten to whole-name equality OR closed-set allowlist. NIT-class severity but is a real residual counterexample to Attack 1.
- **N2 — npm-claude-shim ps comm value unverified (Attack R2-D sub-NIT).** macOS dev installs of @anthropic-ai/claude-code via npm may produce `comm = "node"` or `"sh"` instead of `"claude"` depending on the shim shape. Empirical verification needed before dispatcher dogfood — if the spawn's actual `ps -p <pid> -o comm=` output is `"node"`, the round-2 fix swings the failure mode from "reuse false-positive" (round-1 Attack 1) to "false-negative orphan reap of healthy node-shim spawns" (round-2 regression). Recommend: instrument dispatcher startup to log `manifest.ClaudePID` + `psCommLookup(pid)` for the next 1–2 real spawns, confirm the comm value contains "claude", THEN close. Could be a Drop-4c-end smoke test.
- **N3 — Round-1 NITs that survived round-2 (carry-over):**
  - N3.a — `recordingProcessChecker` declared at L771 (was L556 in round-1) below first use at L504. Same readability nit; trivial refactor.
  - N3.b — `var _ = time.Now` at L815 (was L575 in round-1). Same dead stub.
  - N3.c — Malformed-JSON test coverage gap (round-1 N4) still un-addressed. The branch is exercised at `orphan_scan.go` L327–328 but no test pins it. NIT-class.
- **N4 — Round-1 N5 (test-suite hang) re-surfaces in round-2 worklog L136–147.** Builder reports `mage check` blocked in their session by `monitor_test.go` `go build` shellout sandbox. Not introduced by F.7.8 changes (round-1 also flagged this). Orchestrator confirms they ran `mage ci` clean from their shell — the round-2 algorithm changes are themselves stub-driven (no goroutines, no exec, no race-sensitive primitives) so the test-file additions cannot be the hang cause. Acceptable; orchestrator's clean `mage ci` is authoritative.

---

## 3. Per-attack verdict matrix

| #     | Attack                                                                | Round 1   | Round 2                  |
| ----- | --------------------------------------------------------------------- | --------- | ------------------------ |
| 1     | PID reuse via signal-0 alone                                          | CONFIRMED | **REFUTED** (fix landed) |
| 2     | `OnOrphanFound` blocking                                              | REFUTED   | REFUTED (unchanged)      |
| 3     | Malformed-JSON branch                                                 | REFUTED+NIT | REFUTED+NIT (carry-over) |
| 4     | Concurrent scans                                                      | REFUTED   | REFUTED (unchanged)      |
| 5     | `bundle.path` / metadata read path                                    | REFUTED   | REFUTED (unchanged)      |
| 6     | Adapter routing absent                                                | CONFIRMED | **REFUTED** (REV-16)     |
| 7     | Memory-rule conflicts                                                 | REFUTED   | REFUTED (unchanged)      |
| R2-A  | Substring-contains false-positive (`claude-installer` / `not-claude`) | —         | **CONFIRMED (NIT)**      |
| R2-B  | Race signal-0 ↔ CommLookup                                            | —         | REFUTED                  |
| R2-C  | Empty cmdline output                                                  | —         | REFUTED                  |
| R2-D  | macOS vs Linux ps portability                                         | —         | REFUTED + sub-NIT (N2)   |
| R2-E  | psCommLookup PATH dependency                                          | —         | REFUTED                  |
| R2-F  | Empty-CLIKind fallthrough to signal-0                                 | —         | REFUTED                  |
| R2-G  | REV-16 wellformedness                                                 | —         | REFUTED                  |
| R2-H  | Memory-rule conflicts (re-check)                                      | —         | REFUTED                  |

---

## 4. Final verdict

**PASS-WITH-NITS.**

Round-1's two CONFIRMED counterexamples are both correctly resolved:

- Attack 1 (PID reuse via signal-0 alone) — fix landed via `IsAlive(pid, expectedCmdlineSubstring)` + `CommLookup` injection seam + `expectedCmdlineForCLIKind` derivation. Three new tests (`TestOrphanScannerScan_PIDReuseRejectedByCmdlineMismatch`, `TestOrphanScannerScan_PassesExpectedCmdlineFromCLIKind`, rewritten `TestDefaultProcessChecker_LiveProcess`) pin the algorithm. End-to-end Attack-1 reproduction no longer reaches the false-positive branch.
- Attack 6 (adapter routing absent) — accepted via REV-16 in F7_CORE_PLAN.md L1064–1070. CLIAdapter L10 lock preserved; future per-CLIKind ProcessChecker registry can land in Drop 4d without touching today's API. `manifest.CLIKind` is already plumbed through `expectedCmdlineForCLIKind` and into the `OnOrphanFound` callback context.

One CONFIRMED-NEW finding (Attack R2-A, substring-contains false-positive on `claude-installer` / `not-claude`) is NIT-class — narrows a real but vastly narrower false-positive window than round-1's vulnerability. Plus one forward-watch (N2, npm-claude-shim ps comm value) that needs empirical verification before dispatcher dogfood but does not block the droplet.

**Routing recommendation:** Accept PASS-WITH-NITS and route N1 (substring → equality / allowlist) and N2 (empirical ps-comm verification) as a follow-up Drop-4c-end smoke or Drop 4d refinement — neither rises to a NEEDS-REWORK gate. Round-1 N3.a/b (decl ordering, dead `time.Now` stub) and N3.c (malformed-JSON test) are pre-existing carry-overs not introduced by round-2.

---

## TL;DR

- T1 — Per-attack re-audit: Attacks 1 + 6 (round-1 CONFIRMED) both REFUTED in round-2; one new NIT-class CONFIRMED (Attack R2-A substring-contains false-positive); seven new attacks REFUTED (R2-B through R2-H).
- T2 — Side findings: N1 substring-contains tightening; N2 npm-claude-shim ps comm value verification; N3.a–c round-1 NITs carry-over; N4 round-1 N5 test-suite hang carry-over.
- T3 — Verdict matrix: round-1 2/7 CONFIRMED → round-2 1/15 CONFIRMED (NIT-class), 14/15 REFUTED.
- T4 — Final: PASS-WITH-NITS. Route N1 + N2 as follow-up; do not block droplet on either.
