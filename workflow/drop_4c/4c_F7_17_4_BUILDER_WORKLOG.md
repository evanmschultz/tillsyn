# Drop 4c — F.7.17.4 — `MockAdapter` test fixture — Builder Worklog

**Builder:** go-builder-agent (opus)
**Round:** 1
**Date:** 2026-05-04

## Goal

Land a `MockAdapter` test fixture exercising the `CLIAdapter` interface
contract WITHOUT touching real `claude` / `codex` binaries. Per master
PLAN L19 + F.7.17 REV-6, MockAdapter is the load-bearing proof that the
adapter seam is multi-adapter-ready BEFORE Drop 4d adds the second real
adapter.

## REVISIONS-first read

REV-1 (Command/ArgsPrefix dropped), REV-5 (ExtractTerminalReport rename),
REV-7 (F.7.17.9 merged into F.7-CORE F.7.4), REV-8 (REVISIONS-first
directive) — all applied. The PLAN body's `cli_adapter_test.go` filename
collided with droplet 4c.F.7.17.2's already-shipped file; per the spawn
prompt's forward-collision note, this droplet uses
`mock_adapter_test.go` instead.

## Files created

- `internal/app/dispatcher/mock_adapter_test.go` (NEW) — `MockAdapter`
  struct + 6 named tests + 1 table-driven contract test.
- `internal/app/dispatcher/testdata/mock_stream_minimal.jsonl` (NEW) —
  3-line recorded fixture: 2 `mock_chunk` non-terminal events + 1
  `mock_terminal` event with cost / denials / reason / errors populated.
- `workflow/drop_4c/4c_F7_17_4_BUILDER_WORKLOG.md` (NEW, this file).

No production code touched. MockAdapter lives entirely in the `_test.go`
file — invisible to non-test builds.

## What MockAdapter ships

- **Type**: `MockAdapter` struct with mutex-guarded `calls []mockBuildCommandCall` for test inspection.
- **Constructor**: `newMockAdapter() *MockAdapter`.
- **Methods** (all three from `CLIAdapter`):
  - `BuildCommand(ctx, BindingResolved, BundlePaths) (*exec.Cmd, error)` — returns `*exec.Cmd` whose `Path = "/bin/true"`, `Args` thread `--bundle-root` + `--agent-name` verbatim, and `Env` is set explicitly (NOT `os.Environ()`-inherited per F.7.17 L8) to a closed minimal baseline (`PATH`) plus the resolved values for binding's `Env` names. Records every call for assertion.
  - `ParseStreamEvent(line []byte) (StreamEvent, error)` — parses two event types: `mock_chunk` (non-terminal, carries `Text`) and `mock_terminal` (terminal). Unknown types pass through as non-terminal. Empty / malformed lines return wrapped errors.
  - `ExtractTerminalReport(StreamEvent) (TerminalReport, bool)` — returns `(populated, true)` for `mock_terminal` events and `(zero, false)` for non-terminal events. Cost-pointer semantics per L11: nil when payload omits cost, non-nil pointer when present.
- **Compile-time assertion**: `var _ CLIAdapter = (*MockAdapter)(nil)` — interface drift fails the build.

## Tests shipped

1. `TestMockAdapterBuildCommand` — asserts `*exec.Cmd.Path` / `Args` / `Env`. Critically: a parent-shell secret env var (`TILLSYN_MOCK_SECRET_NEVER_FORWARDED`) is NOT carried into `cmd.Env` unless the binding's `Env` list opts it in (L8 isolation proof). Declared name (`TILLSYN_MOCK_DECLARED`) IS forwarded with the resolved value.
2. `TestMockAdapterBuildCommandRejectsBadInput` — defensive-validation: nil context and zero-value `BundlePaths.Root` produce wrapped errors.
3. `TestMockAdapterParseStreamEventChunkAndTerminal` — round-trips `testdata/mock_stream_minimal.jsonl` (3 lines). Asserts `IsTerminal` flips correctly between chunk events and terminal event; `Raw` preserved verbatim.
4. `TestMockAdapterParseStreamEventMalformedJSON` — empty / lone-newline / unclosed-brace / wrong-typed-discriminator lines all return wrapped errors.
5. `TestMockAdapterExtractTerminalReportPopulatedTerminal` — terminal event with `cost=0.5` produces non-nil `Cost` pointer pointing to 0.5; `Reason="ok"`; one parsed `ToolDenial` with `ToolName="Bash"` and preserved raw `ToolInput`.
6. `TestMockAdapterExtractTerminalReportNonTerminalReturnsFalse` — non-terminal event returns `(zero, false)`.
7. `TestMockAdapterExtractTerminalReportCostNilWhenAbsent` — terminal event WITHOUT `cost` field produces `Cost == nil` (NOT pointer to 0.0). This is the F.7.17 L11 load-bearing absence-vs-zero distinction.
8. `TestCLIAdapterContractTableDriven` — table-driven over `[]CLIAdapter{newMockAdapter()}`. Each row exercises the full BuildCommand → ParseStreamEvent (non-terminal) → ExtractTerminalReport (false) → ParseStreamEvent (terminal) → ExtractTerminalReport (populated, true) sequence end-to-end. Future F.7.17.5 / Drop 4d adapters extend this table.

## Issues hit + fixed mid-build

1. **Slice-comparison compile error**: initial code did `if (report != TerminalReport{})` to assert zero-value. `TerminalReport` contains slice fields (`Denials`, `Errors`), so `!=` is invalid Go. Replaced with field-by-field zero-checks (Cost == nil, Reason == "", len(Denials) == 0, len(Errors) == 0).
2. **`t.Parallel` + `t.Setenv` panic**: Go 1.26's testing framework panics if a test calls both. `TestMockAdapterBuildCommand` needs `t.Setenv` to control parent env for the L8-isolation assertion, so the `t.Parallel` was removed from that test only — every other test stays parallel. Added a `NOTE` comment on the test for future maintainers.
3. **Mage stderr suppression**: the default `mage testPkg` swallows stderr panics. Used `mage -v testPkg ./internal/app/dispatcher` to surface the actual panic message and isolate the t.Parallel issue. (Worth mentioning for future builders working in this package.)

## Verification

- `mage check` — green. 22/22 packages pass. Dispatcher package coverage 73.2% (≥ 70% gate). No formatting drift.
- `mage ci` — green. Build succeeds. Coverage threshold met across every package.
- Total tests in dispatcher package: 176 (up from 169 before this droplet's addition).

## Acceptance criteria status

- [x] `MockAdapter` struct implements `CLIAdapter` interface (compile-time assertion via `var _ CLIAdapter = (*MockAdapter)(nil)`).
- [x] All four contract methods produce sensible outputs for mock fixture inputs. (Three methods per F.7.17 L10; `Calls()` is the test-fixture-only fourth accessor.)
- [x] Fixture file at `testdata/mock_stream_minimal.jsonl` has 3 lines (2 chunks + 1 terminal).
- [x] Table-driven contract test asserts the BuildCommand → ParseStreamEvent → ExtractTerminalReport sequence works end-to-end with MockAdapter.
- [x] Tests confirm `Cost *float64` semantics: terminal event WITH cost populates pointer non-nil; terminal event WITHOUT cost leaves pointer nil.
- [x] Tests confirm `IsTerminal` flag flips correctly between non-terminal and terminal events.
- [x] `mage check` + `mage ci` green.
- [x] Worklog written.
