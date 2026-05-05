# 4c F.7.17.5 — Dispatcher Wiring (QA Proof Review, Round 1)

**Reviewer:** go-qa-proof-agent (sonnet).
**Reviewed:** `4c_F7_17_5_BUILDER_WORKLOG.md` + production diff.
**Verdict:** PROOF GREEN-WITH-NITS.

---

## 1. Verdict

**PROOF GREEN-WITH-NITS.** Every spec acceptance criterion is satisfied with file:line evidence. The two scope expansions (`cli_claude/init.go` + the 1-line `cmd/till/main.go` blank import) are unavoidable per the import-cycle constraint and are the minimum-viable resolution. `mage ci` was re-run by QA and confirmed green (2496 passed / 1 skipped pre-existing / 23 packages). REV-13 honored — no builder commit. Two minor nits noted below; neither blocks the droplet from closing.

---

## 2. Check-by-check evidence

### 2.1 Cycle would have existed

`internal/app/dispatcher/cli_claude/adapter.go:32` imports `dispatcher`. The supporting files in the same package import dispatcher too: `cli_claude/argv.go:8`, `cli_claude/env.go:9`, `cli_claude/stream.go:8`. If `dispatcher/spawn.go` had imported `cli_claude` to populate `var adapters = map[CLIKind]CLIAdapter{CLIKindClaude: cli_claude.New()}`, the resulting `dispatcher → cli_claude → dispatcher` cycle would have been a hard Go compile error. Builder's deviation rationale at `4c_F7_17_5_BUILDER_WORKLOG.md:69-79` is correct.

### 2.2 `RegisterAdapter` exported

`internal/app/dispatcher/spawn.go:131-135` declares `func RegisterAdapter(kind CLIKind, adapter CLIAdapter)` — exported. Its private complement `lookupAdapter` lives at `spawn.go:140-145`. Both serialize through a process-wide `sync.RWMutex` (`spawn.go:103`) over `adaptersMap` (`spawn.go:119`). Concurrency-safe.

### 2.3 `cli_claude/init.go` calls `dispatcher.RegisterAdapter(CLIKindClaude, New())`

`internal/app/dispatcher/cli_claude/init.go:25-27`:

```go
func init() {
    dispatcher.RegisterAdapter(dispatcher.CLIKindClaude, New())
}
```

Direct match with the spawn-prompt requirement.

### 2.4 `cmd/till/main.go` blank import

Diff confirms the addition at `cmd/till/main.go:28-29`:

```go
_ "github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude"
```

Side-effect import is one line + a 3-line preceding doc-comment. Triggers `cli_claude.init()` at process start, populating the registry before the dispatcher ever dispatches.

### 2.5 `BuildSpawnCommand` uses `ResolveBinding` from F.7.17.8

`internal/app/dispatcher/spawn.go:220`: `resolved := ResolveBinding(rawBinding)`. `ResolveBinding` lives at `internal/app/dispatcher/binding_resolved.go:116-153` — the F.7.17.8 deliverable per the cross-reference at `binding_resolved.go:14-19`. Variadic-overrides shape (`overrides ...*BindingOverrides`) is forward-compatible; F.7.17.5 calls it with no overrides, which is correct since CLI/MCP/TUI override layers don't yet plumb.

### 2.6 `ErrUnsupportedCLIKind` sentinel for unknown CLIKind

Sentinel declared at `spawn.go:98`. Returned at `spawn.go:222-225`:

```go
adapter, ok := lookupAdapter(resolved.CLIKind)
if !ok {
    return nil, SpawnDescriptor{}, fmt.Errorf("%w: %q", ErrUnsupportedCLIKind, resolved.CLIKind)
}
```

`fmt.Errorf("%w: ...")` makes it `errors.Is`-able. `TestBuildSpawnCommandRejectsUnknownCLIKind` (`spawn_test.go:178-194`) asserts `errors.Is(err, dispatcher.ErrUnsupportedCLIKind)` for `cli_kind = "bogus"`. ✓

### 2.7 Provisional bundle has `TODO(F.7.1)` marker

`spawn.go:235`: `// TODO(F.7.1): replace inline os.MkdirTemp with bundle-materializer.` Surrounding doc-comment (`spawn.go:227-235`) explains the lifecycle gap (no manifest.json, no deferred cleanup, no project-mode root) and points at F.7-CORE F.7.1. Tests use `t.Cleanup(func() { os.RemoveAll(bundleRoot) })` via the `removeBundle` helper (`spawn_test.go:102-115`).

### 2.8 `system-prompt.md` written, no `hylla_artifact_ref`

- Path: `spawn.go:243` — `SystemPromptPath: filepath.Join(bundleRoot, "system-prompt.md")`.
- Body: `spawn.go:340-375` (`assemblePrompt`) writes `task_id`, `project_id`, `project_dir`, `kind`, `title`, `paths`, `packages`, plus the `move-state directive` line. There is no string-literal `hylla_artifact_ref` anywhere in `assemblePrompt`. Doc-comment at `spawn.go:326-330` explicitly confirms F.7.10's removal: "Hylla awareness was deliberately removed in Drop 4c F.7.10."
- Negative-assertion test: `TestBuildSpawnCommandWritesSystemPromptFile` (`spawn_test.go:201-255`) asserts `!strings.Contains(bodyStr, "hylla_artifact_ref")` at line 247-249 even when `fixtureProject().HyllaArtifactRef` is populated (line 59) — proves the leak path is closed at the source, not just by virtue of the project not declaring the field.
- Positive tokens covered by test (`spawn_test.go:230-238`): `task_id`, `project_id`, `project_dir`, `kind`, `title`, `paths`, `packages`, `move-state directive:`. Matches the spawn-prompt spec.
- Perms `0o600` (`spawn.go:258`) — adequately scoped.

### 2.9 All test scenarios pass

Worklog `§2.2` enumerates 11 scenarios. The dispatcher test package run shows:

```
[PKG PASS] github.com/evanmschultz/tillsyn/internal/app/dispatcher (0.03s)
```

Re-run by QA via `mage test-pkg ./internal/app/dispatcher`: 192 tests passed / 0 failed / 0 skipped. ✓

### 2.10 `mage ci` green per worklog

QA re-ran `mage ci` from `/Users/evanschultz/Documents/Code/hylla/tillsyn/main`. Result:

```
tests: 2497
passed: 2496
failed: 0
skipped: 1   (TestStewardIntegrationDropOrchSupersedeRejected — pre-existing, unrelated)
packages: 23
pkg passed: 23
```

Coverage: `internal/app/dispatcher` 74.9%, `internal/app/dispatcher/cli_claude` 95.7%, `cmd/till` 75.5%. All thresholds (≥70%) satisfied. Build succeeds. Format + lint clean.

**Nit:** worklog says "2496 tests across 23 packages" — strictly there are 2497 tests (2496 passed + 1 skipped), where the skipped test is `TestStewardIntegrationDropOrchSupersedeRejected` and unrelated to this droplet. Trivial mis-statement; not a defect.

### 2.11 No commit by builder per REV-13

`git log --oneline -5` shows `af51dec feat(templates): seed agent_bindings context blocks in default template` as HEAD — predates this droplet's work. `git status --short` shows `cmd/till/main.go`, `spawn.go`, `spawn_test.go` as modified (M) and `cli_claude/init.go` + the worklog as untracked (??). REV-13 (`F7_CORE_PLAN.md:1091-1095`: "every builder spawn prompt MUST include the directive 'You are NOT permitted to run git commit'") is honored.

### 2.12 Scope: 5 files touched (3 in-scope + 2 cycle-break)

In-scope per the F.7.17 plan §F.7.17.5 "Files to edit/create" line (PLAN line 388-390):
- `internal/app/dispatcher/spawn.go` — modified ✓
- `internal/app/dispatcher/spawn_test.go` — modified ✓
- (worklog implicit per drop convention) ✓

Cycle-break expansions per worklog §4:
- `internal/app/dispatcher/cli_claude/init.go` — new, 28 lines ✓
- `cmd/till/main.go` — 1-line blank import + 3-line doc-comment ✓

Total production-surface diff: 5 files. The `workflow/drop_4c/SKETCH.md` modification visible in `git status` is pre-existing dirt from earlier Drop-4c planning waves (unrelated to F.7.17.5 builder work).

---

## 3. Falsification attempts

Each attack mitigated or accepted:

- **A — Could `RegisterAdapter` accept a nil adapter and crash later?** Yes — `spawn.go:131-135` has no nil-guard. Lookup would return `(nil, true)` and `adapter.BuildCommand` would nil-deref at `spawn.go:264`. Mitigation: registry is internal API; only `init()` and tests touch it; both pass non-nil. Acceptable, but a defensive nil-guard would improve robustness. **Nit, see §4.**
- **B — Bundle dir leaks on adapter error?** `spawn.go:258-260` writes the prompt; `spawn.go:264-266` errors return without cleanup. Acknowledged in worklog §2.1 lines 232-233 — `TODO(F.7.1)` will own bundle lifecycle. Tests reap via `t.Cleanup`. Accepted.
- **C — `ResolveBinding` mutates rawBinding?** No — `binding_resolved.go:120-123` uses `cloneStringSlice`; the function returns a new struct. Pure. ✓
- **D — `descriptor.MCPConfigPath` points at a file that does not exist?** Yes — only the bundle root + `system-prompt.md` exist; the `plugin/.mcp.json` path is provisional. Doc-comment `spawn.go:270-276` explicitly says F.7-CORE F.7.1 will materialize. The descriptor field is for `--dry-run` JSON output (path-as-string), not for file-content consumers. Accepted.
- **E — `cmd.Dir = project.RepoPrimaryWorktree` clobbers any cmd.Dir an adapter sets internally.** Today's claude adapter does not set cmd.Dir, so no breakage. Future adapters need to know. Doc-comment at `spawn.go:266-268` could call this out. Architectural nit, not a blocker.
- **F — Is the registry concurrency-safe?** Yes — sync.RWMutex + map. `TestRegisterAdapterRoutesCustomCLIKind` runs without `t.Parallel()` (`spawn_test.go:519-520`) to avoid racing with the default-claude tests. Properly handled.
- **G — `derefString` nil branch ever exercised in production?** No — `ResolveBinding` always populates pointer fields. Helper exists for defensive symmetry. Doc-comment at `spawn.go:297-301` documents this. ✓
- **H — `TestRegisterAdapterRoutesCustomCLIKind` leaks `customKind` registration into the global map across test runs.** Yes — there is no `t.Cleanup` to deregister the fake adapter. Within one test binary run this is benign because no other test queries `"test-custom-kind"`. **Minor nit, see §4.**
- **I — Doc-comment at `spawn.go:115-117` says "Production wiring lives at internal/app/dispatcher/cli_register"** but the actual wiring lives at `internal/app/dispatcher/cli_claude/init.go`. The empty `cli_register/` directory exists from earlier scaffolding but has no Go source files. **Doc-drift nit, see §4.**
- **J — `TestRegisterAdapterRoutesCustomCLIKind` cleanup hook is a no-op.** `spawn_test.go:539-544` schedules a `t.Cleanup` whose body is `_ = os.Args` — a deliberate no-op. The test acknowledges (lines 541-543) that the bundle dir may leak. Bounded by OS tempdir reaping; acceptable for now. Will be subsumed by F.7-CORE F.7.1 cleanup.
- **K — `mage ci` red?** Re-ran by QA: GREEN. 2496 passed.

No unmitigated counterexample to the PROOF GREEN-WITH-NITS verdict.

---

## 4. Nits (non-blocking)

- **N1** — `spawn.go:115-117` doc-comment says "Production wiring lives at internal/app/dispatcher/cli_register" — actual wiring lives at `internal/app/dispatcher/cli_claude/init.go`. Suggested fix in a follow-up: update the comment OR delete the empty `cli_register/` directory + adjust the comment to point at `cli_claude/init.go`.
- **N2** — `spawn.go:131-135` `RegisterAdapter` accepts a nil adapter without checks. Consider a defensive `if adapter == nil { panic("dispatcher: RegisterAdapter called with nil adapter") }` — fail-fast at registration rather than nil-deref at spawn time. Internal API; minor.
- **N3** — `TestRegisterAdapterRoutesCustomCLIKind` leaves `dispatcher.CLIKind("test-custom-kind") -> *fakeAdapter` registered in the process-wide map for the rest of the test binary's lifetime. Adding `t.Cleanup(func() { dispatcher.RegisterAdapter(customKind, nil) })` (or a `DeregisterAdapter` helper if nil-registration is forbidden per N2) would make the test self-cleaning.
- **N4** — Worklog text "2496 tests across 23 packages" is technically off by one — there are 2497 tests; one (`TestStewardIntegrationDropOrchSupersedeRejected` in `mcpapi`) is a pre-existing skip. Trivia.

None of these block close-out. They are tracked here for the orchestrator to triage as drop-end refinements.

---

## 5. Summary

PROOF GREEN-WITH-NITS. Every numbered check (1–12) passes with file:line evidence. The cycle-break expansion is justified and minimum-viable. `mage ci` re-verified green by QA. REV-13 honored (no builder commit). The four nits in §4 are non-blocking and refine surface that already meets the spec.

---

## Hylla Feedback

`N/A — action item touched non-Go files only` is wrong here — this was a Go-code QA pass. Restating: **None — Hylla answered everything needed.** No Hylla queries were issued because the review's evidence-gathering was bounded to:

1. The diff itself (read via `Read` + `git diff` on five paths).
2. The plan + worklog MDs (read via `Read`).
3. The `mage ci` re-run (verification gate).
4. The committed-state symbols `BindingResolved`, `ResolveBinding`, `BundlePaths`, `CLIAdapter`, `CLIKindClaude` — all read directly from `binding_resolved.go` + `cli_adapter.go` since they are the same package as the diff under review and the file lookup was simpler than a Hylla round-trip.

No fallback miss to record.
