# Drop 4c F.7-CORE F.7.12 — Builder QA Proof Review

## Round 1

### Verdict

**PASS** — with one non-blocking documentation-drift NIT and one minor goroutine-leak risk flagged for follow-up.

### Files reviewed

- `internal/app/dispatcher/commit_agent.go` (NEW, 418 lines)
- `internal/app/dispatcher/commit_agent_test.go` (NEW, 684 lines)
- `workflow/drop_4c/4c_F7_12_BUILDER_WORKLOG.md` (NEW)

Cross-checked against (already-committed):

- `internal/app/dispatcher/spawn.go` — `BuildSpawnCommand`, `SpawnDescriptor`, `AuthBundle`, `lookupAdapter`, `ErrUnsupportedCLIKind`.
- `internal/app/dispatcher/cli_adapter.go` — `CLIAdapter`, `CLIKind`, `CLIKindClaude`, `StreamEvent`, `TerminalReport`, `BundlePaths`.
- `internal/app/dispatcher/monitor.go` — `Monitor` struct, `NewMonitor`, `Monitor.Run`.
- `internal/app/dispatcher/binding_resolved.go` — `ResolveBinding`, `BindingResolved`.
- `internal/app/dispatcher/mock_adapter_test.go` — `MockAdapter`, `newMockAdapter`.
- `internal/domain/kind.go` — `KindCommit` constant.
- `internal/domain/action_item.go` — `StartCommit` / `EndCommit` first-class fields.
- `internal/domain/project.go` — `Project.RepoPrimaryWorktree`.
- `internal/templates/catalog.go` — `KindCatalog.LookupAgentBinding`.
- `internal/templates/schema.go` — `AgentBinding` shape.

### Premises proven

| # | Premise | Evidence |
|---|---------|----------|
| P1 | `CommitAgent` struct exists with three required dependency fields. | `commit_agent.go:158-188` declares the struct + `GitDiff` / `BuildSpawnCommand` / `Monitor` plus three unexported test seams (`runCmd`, `openStream`, `lookupAdapterFn`). |
| P2 | `GenerateMessage(ctx, item, project, catalog, auth) (string, error)` is the documented public API. | `commit_agent.go:226-232` matches exactly. |
| P3 | All four sentinels are declared and wrapped via `%w` at every error-return site. | `ErrNoCommitDiff` line 62, `ErrCommitMessageTooLong` line 73, `ErrCommitAgentMisconfigured` line 80, `ErrCommitSpawnNoTerminal` line 88. Wrapping at lines 234, 237, 243, 246, 249, 257, 276, 279, 288, 292, 300, 304, 314, 323, 333, 345, 381, 403, 413. |
| P4 | `CommitMessageMaxLen = 72` is exported. | `commit_agent.go:45`. |
| P5 | Length+newline enforcement matches the spec (`>72` chars OR contains newline → reject). | `commit_agent.go:412` — `strings.ContainsRune(message, '\n') \|\| len(message) > CommitMessageMaxLen`. |
| P6 | Three-tier message extraction handles both production-claude and unit-test paths. | `commit_agent.go:395-401` walks sink-captured assistant text → `report.Reason` → `report.Errors[0]`. |
| P7 | Diff context is materialized at `<bundleRoot>/context/git_diff.patch`. | `commit_agent.go:298-305`; canonical filename via `commitDiffPatchFilename` constant at line 53. |
| P8 | Bundle root recovery from `descriptor.MCPConfigPath` matches the canonical layout. | `commit_agent.go:290` — `filepath.Dir(filepath.Dir(descriptor.MCPConfigPath))` matches `<root>/plugin/.mcp.json` documented at `spawn.go:62-67`. Defensive guard at 287-293 catches empty path. |
| P9 | Synthetic action item carries `Kind=KindCommit` so the catalog binding resolves correctly. | `commit_agent.go:262-272` plus `KindCatalog.LookupAgentBinding(syntheticItem.Kind)` at line 312. |
| P10 | All 7 spec scenarios + 8 robustness edges have tests. | Counted in `commit_agent_test.go`: 15 functions, 17 effective scenarios (one is a 3-row table). |
| P11 | `mage testPkg ./internal/app/dispatcher` 299/299 pass. | Re-run during this review. |
| P12 | `mage ci` 2642 pass / 1 skip (pre-existing) / 0 fail across 24 packages, dispatcher coverage 75.1%. | Re-run during this review. |
| P13 | No commit was made (REV-13). | `git status` shows commit_agent.go, commit_agent_test.go, and worklog as untracked (`??`). |

### Contract drifts validated

The builder's worklog flagged four drifts; each was verified against current code reality:

- **Drift 1** — `BuildSpawnCommand` signature. The spec assumed `(ctx, item, project, catalog, auth)`. **Reality** at `spawn.go:306-311`: `(item, project, catalog, auth) (*exec.Cmd, SpawnDescriptor, error)`. No leading ctx; the function uses `context.Background()` internally with a TODO for ctx-plumbing. The `SpawnBuilder` type alias at `commit_agent.go:114-119` mirrors production reality.
- **Drift 2** — `StreamEvent.Text` extraction. The canonical `TerminalReport` at `cli_adapter.go:287-306` carries `Cost` / `Denials` / `Reason` / `Errors` — no `Text` field. Production claude routes assistant text through non-terminal "assistant" events; the terminal "result" event carries telemetry only. The three-tier extraction (sink → Reason → Errors[0]) is the correct shape; pinned by `TestCommitAgentGenerateMessageAssistantTextWins`.
- **Drift 3** — bundle-root walk. `filepath.Dir(filepath.Dir(descriptor.MCPConfigPath))` matches the canonical `<root>/plugin/.mcp.json` layout. Defensive empty-path guard at lines 287-293. Refinement (move to `SpawnDescriptor.BundleRoot` field) acknowledged in worklog and is appropriate for a future droplet.
- **Drift 4** — diff-context path. Hardcoded as `commitDiffPatchFilename` constant. F.7.13's prompt template (not yet authored) will read from the same canonical name. Coupling is documented at lines 47-53; acceptable forward-coupling for the F.7.13 author to inherit.

### Findings

#### F1 — NIT: documentation-drift on trailing-newline tolerance (non-functional)

`commit_agent.go:408-411` doc-comment claims:

> *"Strip a single trailing newline before the length check so an agent that helpfully appended '\\n' at the end does not trip the cap by one character."*

The implementation at line 412 does NOT strip a trailing newline:

```go
if strings.ContainsRune(message, '\n') || len(message) > CommitMessageMaxLen {
```

A message `"feat: x\n"` (single trailing newline, ≤72 chars) is rejected with `ErrCommitMessageTooLong`. Current behavior is the stricter "no newlines at all" rule — this is correct per the project's "Single-Line Commits" rule. The doc-comment overpromises tolerance the code doesn't deliver.

**Recommendation**: tighten the doc-comment to match the code (drop the strip-trailing-newline language) OR add a `strings.TrimRight(message, "\n")` call before the check (probably the original intent). Either fix is in-place. **Not blocking.**

#### F2 — MINOR RISK: sink-reader goroutine leaks on Monitor panic

The sink-reader goroutine at `commit_agent.go:363-375` reads `for ev := range sink` and exits when `close(sink)` runs at line 378. If `c.Monitor` panics, `close(sink)` never runs and the goroutine leaks.

**Risk profile**: production `Monitor.Run` doesn't panic on documented inputs; the test suite's Monitor mocks either return cleanly or return errors. So no current panic path triggers this. Flag for future hardening if Monitor evolves a panic-able implementation.

**Recommendation (future)**: defer the close — `defer close(sink)` immediately after the goroutine spawn at line 362, before the `c.Monitor` call. **Not blocking.**

### Test inventory (vs. worklog claim)

Worklog claims 14 functions, 16 effective scenarios. Counted in `commit_agent_test.go`:

| # | Test function | Scenarios |
|---|---------------|-----------|
| 1 | `TestCommitAgentGenerateMessageHappyPath` | 1 |
| 2 | `TestCommitAgentGenerateMessageMissingStartCommit` | 1 |
| 3 | `TestCommitAgentGenerateMessageMissingEndCommit` | 1 |
| 4 | `TestCommitAgentGenerateMessageEmptyDiff` | 1 |
| 5 | `TestCommitAgentGenerateMessageMessageTooLong` | 1 |
| 6 | `TestCommitAgentGenerateMessageMultilineMessageRejected` | 1 |
| 7 | `TestCommitAgentGenerateMessageSpawnBuildFails` | 1 |
| 8 | `TestCommitAgentGenerateMessageMonitorFails` | 1 |
| 9 | `TestCommitAgentGenerateMessageAssistantTextWins` | 1 |
| 10 | `TestCommitAgentGenerateMessageNoTerminalText` | 1 |
| 11 | `TestCommitAgentGenerateMessageNilReceiver` | 1 |
| 12 | `TestCommitAgentGenerateMessageNilDeps` (table) | 3 |
| 13 | `TestCommitAgentGenerateMessageGitDiffFails` | 1 |
| 14 | `TestCommitAgentGenerateMessageMissingMCPConfigPath` | 1 |
| 15 | `TestCommitAgentGenerateMessageNoBindingForKind` | 1 |
| **Total** | **15 functions** | **17 effective scenarios** |

Worklog undercounted by one (claimed 14 functions / 16 scenarios). Not a falsification; just a tally drift.

### Verification gates

- `mage testPkg ./internal/app/dispatcher` — 299/299 pass.
- `mage ci` — 2642 pass / 1 skip (`TestStewardIntegrationDropOrchSupersedeRejected`, pre-existing and unrelated) / 0 fail across 24 packages.
- Coverage — dispatcher 75.1% (above the 70% threshold). Every package at or above threshold.
- Format — `gofumpt` clean (gate ran inside `mage ci`).
- Build — `till` binary built clean.
- Commit — none made; files untracked per REV-13.

### Hylla feedback

- **Query**: `hylla_search_keyword` for `BuildSpawnCommand`, `type StreamEvent struct`, `type SpawnDescriptor struct`, `type TerminalReport struct`, `KindCommit`, `LookupAgentBinding` — all returned zero matches.
- **Missed because**: stale ingest. Hylla snapshot 5 predates the Drop 4a Wave 2 dispatcher work and the Drop 4c F.7.17 adapter seam. The symbols exist on disk in the working tree but are not yet in the indexed snapshot.
- **Worked via**: direct `Read` of `internal/app/dispatcher/{spawn,cli_adapter,monitor,binding_resolved,mock_adapter_test}.go`, `internal/domain/{kind,action_item,project}.go`, `internal/templates/{catalog,schema}.go`. Plus `mage testPkg` and `mage ci` for verification.
- **Suggestion**: drop-end ingest cadence is too coarse for QA-of-fresh-code passes — every droplet that lands new symbols starts the next droplet's QA without those symbols searchable. Either (a) Hylla learns an "uncommitted-on-main-tree" fallback layer, or (b) reingest cadence drops below drop-end (per-merge? per-set-of-droplets?). The F.7.12 workaround (read ~10 known files) was straightforward but the gap will hit every fresh-code QA until reingest.

### Section 0 reasoning

Section 0 reasoning lives in the orchestrator-facing response only, per project CLAUDE.md / `SEMI-FORMAL-REASONING.md`. Not duplicated here.
