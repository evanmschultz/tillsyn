# Drop 4c — F.7.17.4 — `MockAdapter` test fixture — QA Proof

**Reviewer:** go-qa-proof-agent (opus)
**Round:** 1
**Date:** 2026-05-04
**Verdict:** **PROOF GREEN-WITH-NITS**

# Section 0 — SEMI-FORMAL REASONING

## Proposal

- **Premises**:
  P1. `MockAdapter` satisfies `CLIAdapter` via compile-time assertion.
  P2. All three contract methods (BuildCommand, ParseStreamEvent, ExtractTerminalReport) are implemented per F.7.17 L8 / L10 / L11 spec.
  P3. The recorded JSONL fixture has 3 lines (2 chunks + 1 terminal).
  P4. A table-driven contract test exercises the full sequence end-to-end.
  P5. Cost-pointer semantics distinguish absent vs explicit-zero.
  P6. IsTerminal flips correctly between non-terminal and terminal events.
  P7. cmd.Env is set explicitly (L8 isolation), parent env not inherited.
  P8. MockAdapter lives in a `_test.go` file (test-only, invisible to prod).
  P9. `mage ci` green per worklog claim.
  P10. Scope == exactly 3 declared files (mock_adapter_test.go, fixture, worklog).
- **Evidence**:
  - `internal/app/dispatcher/mock_adapter_test.go` (read in full, 649 lines).
  - `internal/app/dispatcher/testdata/mock_stream_minimal.jsonl` (read in full, 3 lines).
  - `internal/app/dispatcher/cli_adapter.go` (CLIAdapter interface at L61-83, BindingResolved at L102-179, BundlePaths at L191-220, StreamEvent at L228-265, TerminalReport at L287-306).
  - `workflow/drop_4c/4c_F7_17_4_BUILDER_WORKLOG.md` (read in full, 79 lines).
  - `git status --porcelain` (3 untracked files match scope).
- **Trace or cases**: Each acceptance criterion (1–10 from spawn prompt) traced to file:line evidence — see numbered findings below.
- **Conclusion**: PROOF GREEN-WITH-NITS — every load-bearing claim is backed by file evidence; only doc-drift nits remain.
- **Unknowns**: Builder claim of `mage ci` 2328 passed / 22 packages / 73.2% dispatcher coverage cannot be re-run by a read-only QA-Proof agent. Routed to QA-Falsification sibling for any deeper challenge.

## QA Proof

- **Premises**: Every premise above must be evidenced by file:line citation, not asserted by builder narrative alone.
- **Evidence**: Every numbered finding below carries an explicit `file:line` reference. The CLIAdapter interface (3 methods, L10 of F.7.17 spec) is verified directly from `cli_adapter.go:61-83`; the worklog itself acknowledges the "4th method" framing in the spawn prompt is the test-fixture-only `Calls()` accessor.
- **Trace or cases**:
  - Compile-time assertion → `mock_adapter_test.go:248`.
  - BuildCommand stub → L97-126; Args threading → L105-110; explicit Env → L115-119.
  - ParseStreamEvent dispatch → L161-188 (mock_terminal, mock_chunk, default-tolerant).
  - ExtractTerminalReport → L197-233 (non-terminal short-circuit L198-203, terminal payload extract L205-232).
  - Fixture trace → 3 lines verbatim, 2 chunk + 1 terminal with cost/denials/reason/errors.
  - Table test trace → L555-648 walks 5 steps (BuildCommand, parse non-terminal, extract false, parse terminal, extract populated).
- **Conclusion**: All 10 acceptance criteria are evidenced by source. Coverage of `mage ci` claim is delegated downstream (read-only role).
- **Unknowns**: None within the proof-completeness scope.

## QA Falsification

- **Premises**: The PROOF verdict survives explicit attack on each premise.
- **Evidence**:
  - Attack A: "MockAdapter doesn't implement CLIAdapter — interface drifted in cli_adapter.go." Mitigation: `var _ CLIAdapter = (*MockAdapter)(nil)` (L248) is a compile-time check; the worklog reports `mage ci` green, which means the assertion compiled. Plus I diffed method signatures against `cli_adapter.go:61-83` — all 3 match.
  - Attack B: "Cost pointer semantic is broken — absence test is too narrow." Mitigation: `TestMockAdapterExtractTerminalReportCostNilWhenAbsent` (L524) explicitly omits the field from input JSON (L528) and asserts `report.Cost != nil` is reported as a failure (L538). The PopulatedTerminal test (L450) covers the present-cost branch.
  - Attack C: "Env-isolation test is paper-thin." Mitigation: L262 sets a uniquely-named secret env var via `t.Setenv`; L319-321 explicitly asserts that key is NOT in the materialized cmd.Env map; L316-318 asserts the declared name IS forwarded with the resolved value. Both branches of L8 isolation are covered.
  - Attack D: "Table-driven contract test only has one row, so it doesn't actually prove multi-adapter readiness." Mitigation: per F.7.17 master PLAN L19 the multi-adapter readiness PROOF is "future adapters can be appended as table rows without contract churn" — the table SHAPE is the load-bearing artifact, not row count. F.7.17.5 (claudeAdapter) lands the second row in the next droplet. The worklog at L55-56 articulates this explicitly.
  - Attack E: "Wrong test-count in worklog header (`6 named` at L27 vs 7 named in source)." Mitigation: minor doc drift; worklog "Tests shipped" subsection (L47-55) correctly enumerates 7+1. Filed as Nit N1 (non-blocking).
  - Attack F: "BuildCommand doesn't validate empty AgentName." Mitigation: `BindingResolved.AgentName` is documented (cli_adapter.go:105-106) as always-populated by the resolver — defensive validation here would be belt-and-suspenders, not a contract gap. Filed as Nit N3.
  - Attack G: "ExtractTerminalReport for `ev.IsTerminal=true && ev.Type!=mock_terminal` returns (zero, false) silently — wrong direction; should error." Mitigation: that combination is unreachable from MockAdapter's own ParseStreamEvent (which only sets IsTerminal=true when Type=="mock_terminal" at L162-167). Defensive code path; consistent with non-terminal short-circuit at L198. Filed as Nit N5.
  - Attack H: "Scope creep — `cli_claude/`, `4c_F7_17_3_*.md` are also untracked." Mitigation: those are sibling droplet 4c.F.7.17.3's deliverables, NOT this droplet's. Verified by inspecting `4c_F7_17_3_BUILDER_WORKLOG.md` existence in the same `git status` output.
- **Trace or cases**: A through H all mitigated or routed to non-blocking nit list.
- **Conclusion**: No counterexample to PROOF GREEN. Three doc/defensive-code nits acknowledged in body §1.3-1.5.
- **Unknowns**: `mage ci` re-execution is out of read-only QA-Proof scope; deferred to QA-Falsification sibling and the gating CI run.

## Convergence

- (a) QA Falsification produced no unmitigated counterexample.
- (b) QA Proof confirmed every load-bearing claim is backed by file:line evidence in `mock_adapter_test.go`, `testdata/mock_stream_minimal.jsonl`, and `cli_adapter.go`.
- (c) Remaining Unknown — independent re-run of `mage ci` — is explicitly deferred to QA-Falsification + CI gate.
- All checks passed; verdict GREEN-WITH-NITS.

## 1. Findings

### 1.1 Acceptance criteria (all 10 PASS)

- 1.1.1 **Compile-time assertion**: `var _ CLIAdapter = (*MockAdapter)(nil)` at `internal/app/dispatcher/mock_adapter_test.go:248`. PASS.
- 1.1.2 **BuildCommand contract** (`mock_adapter_test.go:97-126`):
  - Returns `*exec.Cmd` via `exec.CommandContext(ctx, "/bin/true", args...)` at L110.
  - Args thread `--bundle-root` (L107) and `--agent-name` (L108) verbatim from `paths.Root` and `binding.AgentName`.
  - cmd.Env is set explicitly at L119 to closed baseline `PATH=...` (L115) plus per-binding `binding.Env`-resolved names (L116-118). os.Environ() is NOT inherited.
  - PASS.
- 1.1.3 **ParseStreamEvent contract** (`mock_adapter_test.go:138-188`):
  - `mock_terminal` → `IsTerminal=true` (L165) at L162-167.
  - `mock_chunk` → `IsTerminal=false`, populates `Text` (L173-178) at L168-178.
  - Unknown types pass through as non-terminal at L179-187 (acceptable per L137 doc-comment).
  - Empty / malformed JSON → wrapped error at L142-144, L152-154.
  - PASS.
- 1.1.4 **ExtractTerminalReport contract** (`mock_adapter_test.go:197-233`):
  - Non-terminal → `(TerminalReport{}, false)` at L198-200.
  - Wrong-Type-but-IsTerminal-true → `(TerminalReport{}, false)` at L201-203 (defensive).
  - Terminal → decodes adapter-private payload (L205-214), populates Cost / Denials / Reason / Errors (L227-232), returns `(populated, true)`.
  - PASS.
- 1.1.5 **Fixture file** (`internal/app/dispatcher/testdata/mock_stream_minimal.jsonl`): 3 lines verified by direct read — line 1 `mock_chunk` "hello", line 2 `mock_chunk` "world", line 3 `mock_terminal` with `cost=0.5`, `reason="ok"`, one Bash denial, empty errors. PASS.
- 1.1.6 **Table-driven contract test** (`TestCLIAdapterContractTableDriven`, `mock_adapter_test.go:555-648`): single MockAdapter row at L569-577 walks the 5-step sequence — BuildCommand (L590), parse non-terminal (L603), extract → false (L616), parse terminal (L622), extract → populated true with Cost-pointer assertion (L635-645). PASS. Multi-adapter readiness is the table SHAPE (extensible at L578 by appending rows), not row count, per F.7.17 master PLAN L19.
- 1.1.7 **Cost-pointer semantics**:
  - WITH cost: `TestMockAdapterExtractTerminalReportPopulatedTerminal` at L450, asserts `report.Cost != nil` and `*report.Cost == 0.5` at L465-470. PASS.
  - WITHOUT cost: `TestMockAdapterExtractTerminalReportCostNilWhenAbsent` at L524, fixture string at L528 omits cost field, asserts `report.Cost == nil` at L538-540. PASS.
- 1.1.8 **IsTerminal flag**: `TestMockAdapterParseStreamEventChunkAndTerminal` at L363:
  - L396-398: events[:2] assert `IsTerminal == false`.
  - L412-414: events[2] asserts `IsTerminal == true`.
  - PASS.
- 1.1.9 **Env-isolation test**: `TestMockAdapterBuildCommand` at L256:
  - L262 sets `TILLSYN_MOCK_SECRET_NEVER_FORWARDED=leaked` via `t.Setenv`.
  - L263 sets `TILLSYN_MOCK_DECLARED=declared-value`.
  - L269 binding declares only `TILLSYN_MOCK_DECLARED` in `Env`.
  - L319-321 asserts the secret name is NOT present in `cmd.Env` map.
  - L316-318 asserts declared name IS forwarded with resolved value.
  - L322-324 asserts PATH (closed baseline) IS present.
  - PASS.
- 1.1.10 **MockAdapter is TEST-ONLY**: file lives at `internal/app/dispatcher/mock_adapter_test.go` — Go's `_test.go` build constraint excludes it from non-test compilation (no build tag needed). PASS.
- 1.1.11 **`mage ci` green**: builder worklog L65-67 reports 22/22 packages pass, dispatcher coverage 73.2% (≥70% gate). Spawn prompt notes 2328 passed / 0 failed / 22 packages. CONDITIONAL PASS — re-run is out of read-only scope; verdict deferred to QA-Falsification sibling + CI gate.
- 1.1.12 **Scope == 3 files + worklog**: `git status --porcelain` shows the three new files (`mock_adapter_test.go`, `testdata/mock_stream_minimal.jsonl`, `4c_F7_17_4_BUILDER_WORKLOG.md`); other untracked entries (`cli_claude/`, `4c_F7_17_3_BUILDER_WORKLOG.md`, etc.) belong to sibling droplet 4c.F.7.17.3 — confirmed by the parallel worklog file's existence. PASS.

### 1.2 Mid-build fix verification

- 1.2.1 **Slice-comparison fix**: per worklog L59, `TerminalReport != TerminalReport{}` was replaced with field-by-field zero checks. Verified at `mock_adapter_test.go:505-516` (TestMockAdapterExtractTerminalReportNonTerminalReturnsFalse) — uses `report.Cost != nil`, `report.Reason != ""`, `len(report.Denials) != 0`, `len(report.Errors) != 0`. PASS.
- 1.2.2 **`t.Parallel + t.Setenv` fix**: per worklog L60, `TestMockAdapterBuildCommand` removed `t.Parallel()` because `t.Setenv` is incompatible. Verified at L256-258 — no `t.Parallel()` call, NOTE comment present at L257-258 explaining the omission. Every other Test function calls `t.Parallel()` at the top (L345, L364, L425, L451, L490, L525, L556). PASS.

### 1.3 Nit N1 — Worklog test-count drift

- 1.3.1 Worklog header at line 27 says `MockAdapter struct + 6 named tests + 1 table-driven contract test`. Actual count in `mock_adapter_test.go`: 7 named (`TestMockAdapterBuildCommand`, `…RejectsBadInput`, `…ChunkAndTerminal`, `…MalformedJSON`, `…PopulatedTerminal`, `…NonTerminalReturnsFalse`, `…CostNilWhenAbsent`) + 1 table-driven (`TestCLIAdapterContractTableDriven`) = 8 total.
- 1.3.2 Worklog "Tests shipped" subsection at L47-55 correctly enumerates 7+1, so this is purely a header-line drift.
- 1.3.3 Non-blocking. Suggest one-line fix on the worklog header during the next round if a round happens; otherwise leave as historical drift artifact.

### 1.4 Nit N3 — Defensive validation gap on AgentName

- 1.4.1 `MockAdapter.BuildCommand` (L97-103) validates `ctx == nil` and `paths.Root == ""` but not `binding.AgentName == ""`. An empty AgentName produces a literal `""` Arg at L108.
- 1.4.2 `BindingResolved.AgentName` documents itself as always-populated by the resolver (`cli_adapter.go:105-106`: "The resolver always populates this — it has no sensible 'absent' form"). So this is acceptable per the upstream contract.
- 1.4.3 Non-blocking. Belt-and-suspenders only — would be a nice-to-have for future-proofing if the resolver invariant ever weakens.

### 1.5 Nit N5 — Defensive-code dead branch in ExtractTerminalReport

- 1.5.1 `mock_adapter_test.go:201-203` returns `(TerminalReport{}, false)` when `ev.IsTerminal == true && ev.Type != "mock_terminal"`.
- 1.5.2 This combination is unreachable from MockAdapter's own ParseStreamEvent (which only sets IsTerminal=true on `mock_terminal` events at L162-167). The branch protects against a hypothetical caller hand-constructing a StreamEvent with mismatched fields.
- 1.5.3 Acceptable defensive code. Symmetric with the non-terminal short-circuit at L198-200.

## 2. Missing Evidence

- 2.1 **None.** Every load-bearing claim in the spawn prompt's 10-item checklist resolves to a `file:line` citation in §1.1.
- 2.2 The one CONDITIONAL PASS (1.1.11, `mage ci` green) reflects the read-only role boundary — re-running mage is reserved for QA-Falsification + CI. Builder worklog evidence is present and consistent with spawn prompt.

## 3. Summary

**Verdict: PROOF GREEN-WITH-NITS.**

- All 10 acceptance criteria from the spawn prompt are evidenced at `file:line` granularity (§1.1).
- Both mid-build fixes are verified in source (§1.2).
- Three nits are non-blocking: worklog test-count header drift (N1), absent AgentName validation (N3), defensive dead branch in ExtractTerminalReport (N5). None block merge; N1 is the only one a follow-up round could trivially fix.
- `mage ci` re-run is deferred to QA-Falsification sibling per read-only role.

## TL;DR

- **T1** All 10 acceptance criteria PASS with file:line evidence; CLIAdapter compile-time assertion confirmed at `mock_adapter_test.go:248`; fixture is exactly 3 lines (2 chunks + 1 terminal); table-driven contract test walks BuildCommand → ParseStreamEvent → ExtractTerminalReport end-to-end; Cost-pointer absence-vs-zero distinction correctly tested; env-isolation test (L8) covered both directions; scope == 3 files + worklog; three non-blocking nits.
- **T2** No missing evidence within read-only QA-Proof scope; `mage ci` re-run is deferred to QA-Falsification sibling and the CI gate.
- **T3** Verdict: **PROOF GREEN-WITH-NITS** — merge-ready; nit N1 (worklog header says "6 named tests" but file ships 7+1) is the only one worth a follow-up round if cheap.
