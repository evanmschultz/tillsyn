# 4c F.7.17.5 — Dispatcher wiring (Builder QA Falsification)

**Droplet:** F.7.17.5 — Dispatcher wiring (`BuildSpawnCommand` rewrite)
**Reviewer role:** go-qa-falsification-agent (read-only adversarial review)
**Date:** 2026-05-04
**Mode:** Filesystem-MD (no Tillsyn action items)
**Scope evidence:**
- Spec: `workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md` §4c.F.7.17.5 + REVISIONS POST-AUTHORING (REV-1 / REV-2 / REV-3 / REV-5 / REV-6 / REV-7 / REV-8 / REV-9)
- Cross-cutting drop-wide rule: `workflow/drop_4c/F7_CORE_PLAN.md` §REV-13 (no builder self-commit)
- Worklog: `workflow/drop_4c/4c_F7_17_5_BUILDER_WORKLOG.md`
- Source files: `internal/app/dispatcher/spawn.go`, `internal/app/dispatcher/spawn_test.go`, `internal/app/dispatcher/cli_claude/init.go`, `cmd/till/main.go`
- Cross-references: `internal/app/dispatcher/cli_adapter.go` (interface contract), `internal/app/dispatcher/cli_claude/adapter.go` + `argv.go` (claude adapter), `internal/app/dispatcher/binding_resolved.go` (`ResolveBinding`), pre-edit baseline `git show 81c49f5:internal/app/dispatcher/spawn.go` (post-F.7.10 baseline)

---

## 1. Findings

### 1.1 Builder deviated from spec via registry pattern (justified)

Spec L394-411 outlined a literal `map[CLIKind]CLIAdapter` initializer plus a constructor-injected adapter map (`BuildSpawnCommand` "accepts an additional dependency: `adapters map[CLIKind]CLIAdapter`"). Builder shipped:

- A `sync.RWMutex` + `map[CLIKind]CLIAdapter` package-private pair plus exported `RegisterAdapter` / private `lookupAdapter` (`spawn.go:99-145`).
- `cli_claude/init.go` (NEW) calls `dispatcher.RegisterAdapter(dispatcher.CLIKindClaude, New())` at package import time.
- `cmd/till/main.go` blank-imports `cli_claude` (lines 28-31) so the production binary triggers the registration.

**Why this is acceptable:** The literal-map pseudocode in the plan body would have required `dispatcher` to import `cli_claude`, but `cli_claude` already imports `dispatcher` for the `BindingResolved` / `BundlePaths` / `CLIAdapter` / `StreamEvent` / `TerminalReport` types (see `cli_claude/adapter.go:32`, `argv.go:8`, `env.go:9`, `stream.go:8`). That is a hard import cycle Go refuses to compile. The registry-with-init pattern is the standard Go workaround (database/sql drivers ship the same shape) and preserves the plan's outcome: claude is the only registered adapter in Drop 4c; Drop 4d is purely additive (new `cli_codex/init.go` + new blank import in `cmd/till/main.go`). Builder also preserved the 4-arg signature `BuildSpawnCommand(item, project, catalog, AuthBundle)`, satisfying the plan's "do NOT change the signature" constraint at the cost of dropping the spec's "constructor-injected adapter map" knob.

The deviation is documented transparently in worklog §4 with rejected alternatives enumerated.

### 1.2 Spawn assembles the descriptor / cmd correctly per F.7.17 contract

- Adapter lookup happens AFTER `ResolveBinding` (`spawn.go:220-225`) — the resolver is what applies the L15 default-to-claude rule (`binding_resolved.go:129-131`), so an empty `rawBinding.CLIKind` produces `CLIKindClaude` BEFORE `lookupAdapter` runs. Spec L399 ("kind := ResolveCLIKind(string(binding.CLIKind))") is functionally satisfied via `ResolveBinding`.
- `ErrUnsupportedCLIKind` (L93-98) wraps the offending kind in the error message. Test `TestBuildSpawnCommandRejectsUnknownCLIKind` (spawn_test.go L178-194) asserts `errors.Is(err, ErrUnsupportedCLIKind)` for unregistered kinds.
- Provisional `BundlePaths` materialized via `os.MkdirTemp("", "tillsyn-spawn-")` (L236) with `TODO(F.7.1)` cross-reference at L235.
- `system-prompt.md` written at `<bundleRoot>/system-prompt.md` with `0o600` perms (L258); body assembled by `assemblePrompt` at L340-374.
- `adapter.BuildCommand(ctx, resolved, bundlePaths)` invoked at L264; `cmd.Dir = project.RepoPrimaryWorktree` set post-call at L268.
- `SpawnDescriptor.MCPConfigPath` set to `<bundleRoot>/plugin/.mcp.json` at L277, satisfying spec L67-69 + L411 commitment that the descriptor's MCP path stays surfaced for `till dispatcher run --dry-run`.

### 1.3 `assemblePrompt` matches F.7.10 + adds paths/packages per F.7.17.5 spec

Pre-F.7.17.5 baseline (`git show 81c49f5:internal/app/dispatcher/spawn.go`) emitted: `task_id`, `project_id`, `project_dir`, `kind`, `title (optional)`, `move-state directive`. Post-edit (L340-374) emits the same SIX fields PLUS `paths (optional)` and `packages (optional)`. Acceptance criterion "extends the body with paths + packages when set" (worklog §2.1) satisfied. **No `hylla_artifact_ref` token** at any line of the function body.

Test `TestBuildSpawnCommandWritesSystemPromptFile` (spawn_test.go L201-255) asserts every structural token present AND the explicit negative `strings.Contains(bodyStr, "hylla_artifact_ref")` substring check at L247-249. F.7.10's removal sweep stays load-bearing.

### 1.4 Registry pattern is concurrency-safe and overwrite-tolerant

`RegisterAdapter` (L131-135) holds `adaptersMu.Lock()` for the whole map write. `lookupAdapter` (L140-145) holds `RLock()`. Multiple concurrent `init()` calls would serialize correctly (Go runtime serializes `init()` per package, but the lock is defensive against future dynamic registration). Doc-comment at L123 documents "last writer wins" overwrite semantics.

### 1.5 Drop 4d is purely additive

Adding the codex adapter requires only:

- New `internal/app/dispatcher/cli_codex/` package (mirror of `cli_claude/`).
- New `internal/app/dispatcher/cli_codex/init.go` calling `dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, New())`.
- New blank import in `cmd/till/main.go`.
- New `const CLIKindCodex CLIKind = "codex"` in `cli_adapter.go` (currently absent — only mentioned in doc-comments at L21 + L37).

No edits to `spawn.go`, `BuildSpawnCommand`, or any existing test. The seam is clean.

### 1.6 Worklog accuracy / spec compliance

- §3 (acceptance criteria) all checkboxes match observed code shape.
- §4 (scope deviations) explicitly enumerates the two scope expansions (`cli_claude/init.go` NEW + `cmd/till/main.go` 1-line) and the rejected alternatives.
- §6 conventional commit message is single line, ≤72 chars (`feat(dispatcher): wire CLIAdapter registry through BuildSpawnCommand`). Sticks to repo convention.

### 1.7 Coverage above gate

Worklog reports 74.9% on `internal/app/dispatcher` (above 70% threshold) and 95.7% on `internal/app/dispatcher/cli_claude`. Spec F.7.17.5 has no specific coverage requirement beyond mage-ci's 70% gate; gate cleared.

---

## 2. Counterexamples (per-attack verdicts)

### 2.1 — A1 (compile-time check for missing registration): REFUTED

There is no compile-time check that catches a removed `cmd/till/main.go` blank import — if the import is removed, every spawn fails at runtime with `ErrUnsupportedCLIKind`. **However**, this is the standard Go init-registry pattern (database/sql drivers, image format codecs, etc.) and is widely understood. spawn.go L106-118 doc-comment explicitly documents the requirement; spawn_test.go L13-18 reuses the same blank import and would also fail loudly if the cli_claude package were removed. Not a counterexample to the droplet's correctness — the registry pattern is a defensible deviation from the literal spec.

### 2.2 — A2 (RegisterAdapter thread safety): REFUTED

`RegisterAdapter` holds `adaptersMu.Lock()` (write lock) for the whole map mutation (`spawn.go:131-135`). Concurrent calls are safe. Doc-comment at L100-102 explains the design choice.

### 2.3 — A3 (RegisterAdapter overwrite semantics): REFUTED

Doc-comment at L123-124: "Repeat registrations under the same kind overwrite — last writer wins." Verified at L134 (`adaptersMap[kind] = adapter`). No panic, no error. This is a deliberate semantic — the registry pattern's standard contract.

### 2.4 — A4 (4-arg signature preservation): REFUTED

Pre-edit `git show 81c49f5:internal/app/dispatcher/spawn.go` and post-edit (`spawn.go:185-190`) both declare `BuildSpawnCommand(item domain.ActionItem, project domain.Project, catalog templates.KindCatalog, authBundle AuthBundle) (*exec.Cmd, SpawnDescriptor, error)`. Both production callers `dispatcher.go:545` and `dispatcher.go:748` (also `cmd/till/dispatcher_cli_test.go:378` which only references the symbol in a doc-comment) compile unchanged.

### 2.5 — A5 (provisional bundle path leakage): NIT — known and tracked

`os.MkdirTemp` creates a temp dir; `BuildSpawnCommand` returns `*exec.Cmd` but the caller doesn't see the bundle handle. Production code: directory leaks until OS / dev cleanup. Worklog §2.1 acknowledges with `TODO(F.7.1)` cross-reference; spawn.go L227-238 documents the leak. Tests use `removeBundle` cleanup hook to avoid accumulation in CI. F.7.1 owns the proper lifecycle. Accepted with TODO marker.

### 2.6 — A6 (system-prompt.md fidelity to F.7.10): REFUTED

Pre-F.7.17.5 baseline (`81c49f5:spawn.go` `assemblePrompt`) emits: `task_id`, `project_id`, `project_dir`, `kind`, `title (optional)`, `move-state directive`. Post-edit (`spawn.go:340-374`) emits the same six PLUS `paths (optional)` and `packages (optional)`. No additions beyond what plan F.7.17.5 explicitly required (worklog §2.1). No deletions from F.7.10's contract.

### 2.7 — A7 (`hylla_artifact_ref` absent): REFUTED

Zero references to `hylla` in `assemblePrompt` body (`spawn.go:340-374`). Test `TestBuildSpawnCommandWritesSystemPromptFile` (spawn_test.go L247-249) pins the negative substring assertion. F.7.10 contract preserved.

### 2.8 — A8 (external test package + unexported access): REFUTED

`spawn_test.go` declares `package dispatcher_test` (L1) and accesses only exported symbols: `dispatcher.BuildSpawnCommand`, `dispatcher.AuthBundle`, `dispatcher.SpawnDescriptor`, `dispatcher.RegisterAdapter`, `dispatcher.ErrUnsupportedCLIKind`, `dispatcher.ErrNoAgentBinding`, `dispatcher.ErrInvalidSpawnInput`, `dispatcher.CLIKind`, `dispatcher.CLIAdapter`, `dispatcher.BindingResolved`, `dispatcher.BundlePaths`, `dispatcher.StreamEvent`, `dispatcher.TerminalReport`. Custom `fakeAdapter` is locally defined (L492-512) and registered via `dispatcher.RegisterAdapter` (L524). No private-symbol access.

### 2.9 — A9 (blank import placement in import block): REFUTED

`cmd/till/main.go` import group is one block (lines 4-39). The local-package imports starting at line 21 are alphabetically sorted; `internal/app/dispatcher/cli_claude` (line 31) sits correctly between `internal/app/dispatcher` (line 27) and `internal/buildinfo` (line 32) — alphabetic order satisfied (`cli_claude` < `buildinfo` alphabetically among the path tails after `internal/app/dispatcher/`... actually: `internal/app/dispatcher/cli_claude` < `internal/buildinfo` lexicographically because `app` < `buildinfo` at character index 9 — `b` > `a`, so `internal/buildinfo` correctly sorts AFTER `internal/app/...` paths). gofumpt-clean.

### 2.10 — A10 (init order: map initialized before init): REFUTED

`spawn.go:103` declares `var adaptersMu sync.RWMutex` (zero value, ready). `spawn.go:119` declares `var adaptersMap = map[CLIKind]CLIAdapter{}` (composite-literal init). Go spec: package-level `var` declarations with initializers run BEFORE any `init()` function. `cli_claude` imports `dispatcher`, so `dispatcher`'s package-level vars complete BEFORE `cli_claude.init()` fires. Order safe.

### 2.11 — A11 (coverage gaps): UNKNOWN — gate passed, granular gaps unmeasurable from MD evidence

Worklog §3 reports 74.9% on `internal/app/dispatcher`. Above the 70% gate. Without running the coverage profile myself, I cannot identify the specific 25.1% uncovered. The new tests cover `RegisterAdapter` (`TestRegisterAdapterRoutesCustomCLIKind`), `lookupAdapter` (transitively via `BuildSpawnCommand`), `ErrUnsupportedCLIKind` (`TestBuildSpawnCommandRejectsUnknownCLIKind`), default-to-claude (`TestBuildSpawnCommandUsesClaudeAdapterByDefault`), explicit-claude (`TestBuildSpawnCommandHonorsExplicitClaudeCLIKind`), prompt write + content (`TestBuildSpawnCommandWritesSystemPromptFile`), bundle propagation (`TestBuildSpawnCommandPropagatesBundlePaths`). All key functions exercised. Marked Unknown rather than refuted because I cannot rule out an unhit error branch.

### 2.12 — A12 (Drop 4d additivity): REFUTED

To add codex in Drop 4d:

- Declare `const CLIKindCodex CLIKind = "codex"` in `cli_adapter.go` (already mentioned in doc-comments at L21 + L37; the const itself isn't shipped yet — Drop 4d's job).
- Create `internal/app/dispatcher/cli_codex/{adapter,argv,env,stream,init}.go` mirroring `cli_claude/`.
- `cli_codex/init.go` calls `dispatcher.RegisterAdapter(dispatcher.CLIKindCodex, New())`.
- Add one blank import in `cmd/till/main.go`.

ZERO edits to `spawn.go`, `BuildSpawnCommand`, or any existing test. The registry seam is purely additive.

### 2.13 — A13 (no builder self-commit per REV-13): REFUTED

`git log --oneline -5` shows the most recent commit is `2339b10 docs(drop-4a): droplet 4a.15 qa green` — predates this droplet's work entirely. The four file changes are uncommitted in the working tree. Worklog §3 line 63 explicitly checks this: `[x] **NO commit by builder** (per F.7-CORE REV-13).`. F.7-CORE REV-13 directive observed.

### 2.14 — A14 (memory-rule conflicts): REFUTED

- `feedback_no_migration_logic_pre_mvp.md`: no SQL changes, no `till migrate` CLI, no migration code. ✓
- `feedback_subagents_short_contexts.md`: 5 files (4 modified/new + 1 worklog), single-purpose. ✓
- `feedback_orchestrator_no_build.md`: orchestrator dispatched the spawned builder; the builder edited code; no orchestrator code edits visible. ✓
- No `mage install` invocation in any source file (`/usr/bin/grep` confirms).
- No raw `go build` / `go test` / `go vet` / `go run` shell-outs in source.

### 2.15 — A15 (cmd/till/main.go scope minimal): REFUTED with sub-NIT

`git diff cmd/till/main.go` shows exactly `+4 lines` added (3-line doc-comment + 1 import line). No flag parsing changes, no init code changes, no `func main` mutation. **Sub-NIT:** worklog §2.4 says "1-line change" — strictly the import itself IS one line, but with the comment header the diff is 4 lines. Cosmetic imprecision; no functional issue.

### 2.16 — Bonus: spawn.go doc-comment drift on `cli_register` package: NIT (CONFIRMED)

`spawn.go:115-118` claims "Production wiring lives at internal/app/dispatcher/cli_register; cmd/till imports it for side-effects." `spawn.go:122` similarly references "cli_register today." But the actual production wiring lives in `internal/app/dispatcher/cli_claude/init.go`, NOT in any `cli_register/` package. The empty `internal/app/dispatcher/cli_register/` directory exists on disk but is git-untracked and contains zero files. A future contributor reading this doc-comment would search for `cli_register` and find an empty dir — wasted minutes.

**Suggested fix (one-line edit, builder territory if reopened):** replace "Production wiring lives at internal/app/dispatcher/cli_register" with "Production wiring lives at internal/app/dispatcher/cli_claude/init.go" and "cli_register today" with "the per-CLI adapter package's `init.go` today."

Severity: NIT. Doesn't affect runtime correctness; affects future-contributor discoverability.

### 2.17 — Bonus: SKETCH.md scope expansion (uncommitted, undeclared): NOTE

`git status --porcelain` shows `M workflow/drop_4c/SKETCH.md` — 83 lines of additions describing F.7.17 / F.7.18 architecture. Worklog §4 does NOT mention SKETCH.md. **Most likely** these are accumulated planner-side edits from earlier F.7.17 droplets (.1, .2, .3, .4) carried forward across the sequence — NOT this builder's deviation, since the diff is planner-domain content (theme descriptions, validation rules, sequencing constraints). Cannot be definitively attributed to this builder without git blame on uncommitted lines (impossible). The SKETCH.md content itself looks coherent with the broader F.7 plan and matches the plan's REV-1 / REV-2 / REV-3 / REV-7 supersessions. **Note for the orchestrator:** verify this SKETCH delta is intended pre-existing planner work and gets folded into the next commit's diff context.

### 2.18 — Bonus: F.7.17.4 MockAdapter unused by F.7.17.5 tests: NIT

Plan F.7.17.5 acceptance line "Mock-injection test: `adapters[CLIKindClaude] = newMockAdapter(...)` — `BuildSpawnCommand` returns the mock's `*exec.Cmd` verbatim" was supposed to consume the `MockAdapter` shipped in F.7.17.4 (`mock_adapter_test.go`). But because `spawn_test.go` switched from `package dispatcher` to `package dispatcher_test` (external) to support the side-effect import, it cannot reach `MockAdapter` (declared inside `mock_adapter_test.go`, internal-test-only — visible from `package dispatcher` test files but NOT from `package dispatcher_test` files). Builder shipped a local `fakeAdapter` (spawn_test.go L492-512) instead, which is functionally equivalent for proving the registry routes a custom CLIKind. The contract assertion lands; only the test-fixture reuse is lost.

Severity: NIT. F.7.17.4's MockAdapter remains useful for in-package contract tests in `cli_adapter_test.go`; the F.7.17.5 external-test world just gets its own thinner mock.

---

## 3. Summary

**Overall verdict: PASS-WITH-NITS.**

Builder's registry-pattern deviation from the plan's literal `var adapters = map[...]` initializer is JUSTIFIED — the literal pseudocode would have created a hard import cycle Go refuses to compile. The chosen pattern is a Go standard idiom (database/sql driver registration), preserves the 4-arg signature constraint, satisfies every functional acceptance criterion, and keeps Drop 4d purely additive. Worklog §4 transparently documents the deviation with rejected alternatives.

All 15 declared attack vectors landed as REFUTED (12) or NIT (3 — A5 known-and-tracked, A11 coverage-gaps-unmeasurable, A15 cosmetic-imprecision). Three additional bonus findings:
- 2.16 spawn.go doc-comment drift on `cli_register` (NIT — fixable in a one-line edit)
- 2.17 uncommitted SKETCH.md delta (NOTE — likely pre-existing planner work, orchestrator should verify)
- 2.18 F.7.17.4 MockAdapter unused by F.7.17.5 tests (NIT — fakeAdapter is functionally equivalent)

NONE of the findings constitute a counterexample requiring rework. The dispatcher wiring is correct, well-tested, and mergeable as-is. The two NITs that affect future-contributor experience (2.16 doc-comment, 2.18 MockAdapter reuse) are reasonable to fix in a follow-up edit but do not block this droplet's close-out.

**Recommendation:** PASS-WITH-NITS. Optional one-line cleanup of `spawn.go:115-122` doc-comment to reference `cli_claude/init.go` instead of the non-existent `cli_register/` package; orchestrator should sanity-check the SKETCH.md delta is intended planner work before commit.

---

## TL;DR

- **T1**: Findings sweep — registry-with-init pattern justified by import-cycle constraint; all functional acceptance criteria satisfied; assemblePrompt extends F.7.10 baseline with `paths` + `packages` fields and remains `hylla_artifact_ref`-free; Drop 4d additivity preserved; coverage 74.9% above the 70% gate; no builder self-commit per F.7-CORE REV-13.
- **T2**: Counterexamples — 12 attacks REFUTED, 3 NIT (provisional bundle leak known-and-tracked, coverage gaps unmeasurable from MD evidence, cmd/till/main.go diff was technically 4 lines not 1); 3 bonus findings (`cli_register` doc-comment drift NIT, uncommitted SKETCH.md delta NOTE, F.7.17.4 MockAdapter unused by F.7.17.5 tests NIT). No CONFIRMED counterexamples.
- **T3**: Verdict — PASS-WITH-NITS. Mergeable as-is; optional one-line `spawn.go` doc-comment cleanup recommended.

---

## Hylla Feedback

`N/A — review touched non-Go files (plan MD, worklog MD) and Go source already on disk; reviewed via `Read` per the spawn-prompt directive ("No Hylla calls").` No Hylla queries attempted; no miss to record.
