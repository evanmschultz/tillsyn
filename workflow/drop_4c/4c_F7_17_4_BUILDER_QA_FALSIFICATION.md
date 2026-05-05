# Drop 4c — F.7.17.4 — `MockAdapter` test fixture — QA Falsification

**Reviewer:** go-qa-falsification-agent (opus)
**Round:** 1 (retry — prior dispatch hung)
**Date:** 2026-05-04
**Verdict:** **PASS-WITH-NITS**

---

## 1. Findings

### 1.1 Per-attack verdict (A1–A15)

- **1.1.1 A1 — `CLIAdapter` contract drift compile assertion: REFUTED.** The compile-time assertion is `var _ CLIAdapter = (*MockAdapter)(nil)` at `internal/app/dispatcher/mock_adapter_test.go:248`, file-scope (top-level package var), not inside a function body. If `CLIAdapter` (defined at `internal/app/dispatcher/cli_adapter.go:61-83`) gains, removes, or changes a method signature, the test file fails to compile and `mage ci` fails before any test runs. Method signatures match line-for-line: `BuildCommand(ctx context.Context, binding BindingResolved, paths BundlePaths) (*exec.Cmd, error)` (interface L65, impl L97); `ParseStreamEvent(line []byte) (StreamEvent, error)` (interface L72, impl L138); `ExtractTerminalReport(ev StreamEvent) (TerminalReport, bool)` (interface L82, impl L197). No counterexample.

- **1.1.2 A2 — `/bin/true` availability on minimal CI: REFUTED.** The contract tests never invoke `cmd.Run()`, `cmd.Start()`, `cmd.Wait()`, or `cmd.Output()`. They inspect `cmd.Path`, `cmd.Args`, and `cmd.Env` only (`mock_adapter_test.go:286-324`, `mock_adapter_test.go:597-599`). `exec.CommandContext` does NOT stat the path at construction — it stores the string for later resolution. Test runs on a `/bin/true`-less host produce identical assertions. The fixture's own doc-comment at `mock_adapter_test.go:93-96` explicitly notes "The command is NEVER executed by the contract tests." No counterexample.

- **1.1.3 A3 — Malformed-JSON / empty / forward-compat handling: REFUTED with NIT.** `TestMockAdapterParseStreamEventMalformedJSON` (`mock_adapter_test.go:424-440`) covers four cases: empty bytes, `\n` only, `{not json`, and `{"type": 12345}`. The "extra fields" forward-compat path is implicitly covered: the type discriminator is decoded into a struct with only `Type string`, so unknown fields are silently ignored by `encoding/json` defaults. **NIT-A3a (non-blocking):** there is no explicit test that an unknown event type (e.g. `{"type":"future_event_subtype"}`) goes through the `default` case at `mock_adapter_test.go:179-187` and lands as a non-terminal event with the type string preserved. The default branch exists but has no behavioral assertion — coverage is structural-only. Acceptable for a fixture; future-extension safety would benefit from one extra case.

- **1.1.4 A4 — `Cost *float64` absent vs explicit-zero: REFUTED.** `TestMockAdapterExtractTerminalReportCostNilWhenAbsent` (`mock_adapter_test.go:524-544`) feeds the literal `{"type":"mock_terminal","reason":"ok","denials":[],"errors":[]}` (cost field OMITTED) and asserts `report.Cost != nil` is a failure. With `Cost *float64` json-tagged `omitempty`, omitted JSON fields decode as nil. The mirror test `TestMockAdapterExtractTerminalReportPopulatedTerminal` (`mock_adapter_test.go:450-483`) covers the present-cost branch with `cost=0.5` → non-nil pointer to 0.5. Both branches walked. No counterexample.

- **1.1.5 A5 — Contradictory `IsTerminal` source-of-truth: NIT.** `MockAdapter.ParseStreamEvent` does NOT read `is_terminal` from the JSON payload — `IsTerminal` is set deterministically based ONLY on the `type` discriminator switch (`mock_adapter_test.go:161-187`). A line like `{"type":"mock_chunk","is_terminal":true}` would still produce `IsTerminal=false` because the field is silently ignored (no struct field decodes it). This is the correct design — type IS the source of truth — but no test pins the rule. **NIT-A5a (non-blocking):** add one test that constructs `{"type":"mock_chunk","is_terminal":true}` and asserts the parsed event still has `IsTerminal == false`, locking the type-as-truth invariant against a future regression where someone adds an `IsTerminal bool` field to the chunk-payload struct. Not a counterexample to the current behavior.

- **1.1.6 A6 — Env-isolation symmetry: REFUTED.** `TestMockAdapterBuildCommand` covers both directions: `mock_adapter_test.go:319-321` asserts `TILLSYN_MOCK_SECRET_NEVER_FORWARDED` is NOT in the materialized cmd.Env map (parent env not leaked); `mock_adapter_test.go:316-318` asserts `TILLSYN_MOCK_DECLARED` IS forwarded with the resolved value (declared name materialized). Plus `mock_adapter_test.go:322-324` asserts PATH (the closed baseline) is present. Three-way coverage. No counterexample.

- **1.1.7 A7 — `t.Setenv + t.Parallel` workaround correctness: REFUTED.** Only `TestMockAdapterBuildCommand` calls `t.Setenv` (`mock_adapter_test.go:262-263`) and only that test omits `t.Parallel()` (an explanatory NOTE comment lives at `mock_adapter_test.go:257-258`). Every other test function sets `t.Parallel()` at the top: line 345, 364, 425, 451, 490, 525, 556. The seven parallel-eligible tests all opt in; the one env-mutating test correctly opts out. No counterexample.

- **1.1.8 A8 — Table-driven contract test extensibility: REFUTED.** The case slice is typed `[]contractCase` where `contractCase.adapter CLIAdapter` is an interface field (`mock_adapter_test.go:558-578`). F.7.17.5 (claudeAdapter) appends one row with `adapter: claudeAdapter{...}` — no type-check breakage, no helper-function rewrite. The driving loop at `mock_adapter_test.go:580-647` consumes only interface-method calls. Extensibility verified by inspection. No counterexample.

- **1.1.9 A9 — `MockAdapter` test-only confinement: REFUTED.** Production-leakage check via Glob over `internal/app/dispatcher/*.go` and `internal/app/dispatcher/cli_claude/*.go` (only the test file `mock_adapter_test.go` exists in dispatcher's package; cli_claude has no MockAdapter references — verified by inspecting the cli_claude/ file list and the `_test.go` file boundary). Go's `_test.go` build-tag rule excludes the file from `go build` outputs and from importing-package compilations. The capital-M `MockAdapter` is exported within the test build only. No counterexample.

- **1.1.10 A10 — Defensive dead branch in `ExtractTerminalReport`: NIT (already flagged by Proof).** `mock_adapter_test.go:201-203` short-circuits on `ev.IsTerminal == true && ev.Type != "mock_terminal"`. From MockAdapter's own ParseStreamEvent, this is unreachable — IsTerminal=true is only set when Type=="mock_terminal" (`mock_adapter_test.go:162-167`). However: a test (or future caller) CAN construct a `StreamEvent` literal with mismatched fields and call `ExtractTerminalReport` directly — that path silently returns `(zero, false)` rather than the (zero, true) the bool-semantic would suggest "this IS a terminal event." **NIT-A10a (non-blocking):** the current short-circuit is acceptable defensive code; a stricter design would either delete the branch (since it's unreachable from the adapter's own parser) or add a test exercising it via hand-constructed StreamEvent. QA Proof already filed this as Nit N5; no new finding.

- **1.1.11 A11 — `AgentName` validation gap: NIT (already flagged by Proof).** `BuildCommand` validates `ctx == nil` and `paths.Root == ""` but not `binding.AgentName == ""` (`mock_adapter_test.go:97-103`). With empty AgentName, `Args[5]` becomes the literal string `""` (`mock_adapter_test.go:108`). Downstream consumers in MockAdapter today: zero — the cmd is never executed. The contract test (`mock_adapter_test.go:585-588`) always passes a non-empty AgentName, so empty-string passthrough is untested. The interface doc-comment at `cli_adapter.go:105-106` explicitly states "The resolver always populates this — it has no sensible 'absent' form," making AgentName a resolver-side invariant rather than an adapter-side defensive check. Acceptable. QA Proof already filed this as Nit N3.

- **1.1.12 A12 — Fixture chunk distinctness: REFUTED.** `testdata/mock_stream_minimal.jsonl` lines 1 + 2 carry `text=hello` and `text=world` — distinct payloads, not duplicated (verified by direct read). The `TestMockAdapterParseStreamEventChunkAndTerminal` test asserts `ev.Text != ""` (`mock_adapter_test.go:399-401`) but not the specific values — order-sensitive bugs would not surface from the current Text assertion alone, but would surface from the parse-order assertion (events[:2] vs events[2]). No counterexample to the chunk distinctness; the order-sensitivity invariant rides on the slice-index assertion. Acceptable.

- **1.1.13 A13 — `mock_adapter_test.go` filename forward-collision with F.7.17.5: REFUTED.** F.7.17.5 ships its claudeAdapter implementation in `internal/app/dispatcher/cli_claude/` (verified by directory listing — `adapter.go`, `argv.go`, `env.go`, `stream.go`, `adapter_test.go` already exist). F.7.17.5's tests live in `cli_claude/adapter_test.go`, not in the parent dispatcher package. The contract table in `mock_adapter_test.go:555-648` adds a row referencing `cli_claude.NewAdapter(...)` (the public constructor) without conflicting filenames. The PLAN-body collision (master plan named `cli_adapter_test.go` — already taken by F.7.17.2) was correctly resolved at routing time. No counterexample.

- **1.1.14 A14 — Memory-rule conflicts: REFUTED.** `feedback_no_migration_logic_pre_mvp.md` — N/A (no SQLite/migration code; pure test fixture). `feedback_subagents_short_contexts.md` — N/A (single package, single droplet, well-scoped). `feedback_orchestrator_no_build.md` — N/A (builder did the work; orchestrator routed). `feedback_hylla_go_only_today.md` — N/A (Go-only fixture; Hylla applicable but not gating since droplet not committed yet). `feedback_section_0_required.md` — Builder worklog has no Section 0 block but worklogs are durable artifacts not orchestrator-facing responses; rule does not apply. No conflict.

- **1.1.15 A15 — No-commit verification: REFUTED.** `git status --porcelain` shows `mock_adapter_test.go`, `testdata/mock_stream_minimal.jsonl`, and `4c_F7_17_4_BUILDER_WORKLOG.md` (plus the QA Proof MD) as `??` (untracked). `git log --oneline -5` shows the most recent commits are F.7.2 / context-aggregator / claude-adapter — no F.7.17.4 commit exists. Worklog explicitly records "No commit by builder." Per REV-13 the orchestrator commits at the right point. Verified clean. No counterexample.

### 1.2 New attack surface checked beyond A1–A15

- **1.2.1 `bytes.TrimRight` aliasing (`mock_adapter_test.go:141`).** `trimmed` aliases the caller's buffer when no newline trim happens (TrimRight returns the original slice). Builder addresses this at `mock_adapter_test.go:158-159` by copying into `rawCopy` before storing in `StreamEvent.Raw`. The aliasing concern is mitigated. REFUTED.

- **1.2.2 Worklog test-count drift (header line 27).** Header says "6 named tests + 1 table-driven" but file ships 7 named + 1 table-driven (the count corrected in the worklog body at lines 47-55 enumerates 7 + 1). Already filed by QA Proof as N1. NIT.

- **1.2.3 Contract test step labeling.** The contract-test comments at `mock_adapter_test.go:584-635` label five "Steps" but Step 3a is unnumbered relative to "Step 4" (skips "Step 3b"). Cosmetic; structural correctness unaffected. NIT (cosmetic only).

- **1.2.4 `TerminalReport.Errors` empty vs nil round-trip.** Fixture's terminal payload has `"errors":[]` (empty JSON array). `json.Unmarshal` into `[]string` produces a non-nil empty slice, not nil. `MockAdapter.ExtractTerminalReport` does NOT collapse empty Errors to nil (unlike the `denials` collapse at `mock_adapter_test.go:223-225` which DOES set to nil when len==0). The `TerminalReport.Errors` doc-comment at `cli_adapter.go:303-305` says "Nil and empty are equivalent." Test `TestMockAdapterExtractTerminalReportPopulatedTerminal` does not assert on Errors length or nil-ness when fixture has `errors:[]`. **NIT-1.2.4a (non-blocking):** asymmetric collapse vs Denials — minor but present. Document or harmonize when claudeAdapter ships in F.7.17.5.

### 1.3 Summary

- 15 attacks attempted (A1–A15). 11 REFUTED outright. 4 NITs surfaced (A3 unknown-type explicit case, A5 type-as-truth invariant test, A10 dead-branch policy, A11 AgentName validation — A10/A11 already filed by QA Proof). Plus 4 new minor findings in §1.2: aliasing (mitigated), worklog test-count header drift (already by Proof), step labeling cosmetic, Errors empty-vs-nil asymmetry.
- No CONFIRMED counterexample. The fixture's three-method contract is sound, the table-driven test is extensible, env-isolation has both-direction coverage, Cost-pointer absence-vs-zero distinction is pinned, and the file is correctly test-only.
- Verdict: **PASS-WITH-NITS.** No blocking issues. Recommended follow-ups (A3 unknown-type test, A5 type-as-truth pin, §1.2.4 Errors collapse harmonization) are pure additive coverage hardening, not corrections — defer to F.7.17.5 round if cheap, otherwise let stand.

## 2. Counterexamples

- **2.1 None.** No CONFIRMED counterexamples produced. All 15 declared attacks plus four supplemental probes either REFUTED or downgraded to NIT. The fixture is contract-correct.

## 3. Summary

**Verdict: PASS-WITH-NITS.**

The MockAdapter test fixture satisfies every load-bearing claim in the spawn prompt's 15-attack matrix. Compile-time CLIAdapter assertion is at file-scope (drift-detecting). BuildCommand correctly threads paths/binding into Args/Env without inheriting parent env. ParseStreamEvent + ExtractTerminalReport correctly route IsTerminal solely by the JSON `type` discriminator. Cost-pointer absence-vs-zero distinction is pinned in two mirrored tests. Table-driven contract is interface-shaped for F.7.17.5 / Drop 4d extension. Test-only confinement holds (no production leakage). Mid-build fixes (slice-comparison, t.Parallel+t.Setenv) are verified correctly applied. No commit yet — orchestrator will do it.

Four non-blocking nits surfaced — three documentation-or-symmetry-quality, one defensive-validation gap on AgentName already on QA Proof's list. Merge-ready.

## Hylla Feedback

N/A — action item touched non-Go-committed work only (test files staged but not yet committed; falsification used direct file Read of uncommitted source per CLAUDE.md "git diff for files touched after the last Hylla ingest" rule). No Hylla queries attempted; no fallback misses to log.

## TL;DR

- **T1** 15 attacks (A1–A15) plus 4 supplemental probes — 11 REFUTED outright, 4 NITs (two already filed by QA Proof), 0 CONFIRMED counterexamples. Compile-time CLIAdapter assertion at `mock_adapter_test.go:248` correctly file-scoped; `/bin/true` is never executed; cost absence-vs-zero pinned in mirrored tests; t.Setenv/t.Parallel workaround correctly limited to one test; table-driven contract interface-shaped for downstream rows; no production leakage of MockAdapter; mid-build fixes verified.
- **T2** No counterexamples produced. Recommended additive coverage (unknown-type explicit case, type-as-truth invariant pin, Errors empty-vs-nil collapse harmonization) is hardening only — defer to F.7.17.5.
- **T3** Verdict: **PASS-WITH-NITS** — merge-ready. Same nit set as QA Proof + two cosmetic findings; no rework required.
