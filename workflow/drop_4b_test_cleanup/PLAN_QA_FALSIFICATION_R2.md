# PLAN_QA_FALSIFICATION — Drop 4b Test Cleanup (Round 2)

**Verdict:** PASS WITH NITS — 0 CONFIRMED counterexamples, 2 POSSIBLE findings, 3 NITs, 4 REFUTED, 1 EXHAUSTED (no counterexample).

Round-1's 4 CONFIRMED counterexamples (F1/F2/F3/F4) are all addressed in the round-2 PLAN.md. The unprompted round-2 discoveries (`Reason *string` arg + `"supersede_task"` action-string) hold up under fresh attack. Two material gaps remain (F5 future-refinement not durably filed; staleness of two production doc-comments after D1.6 lands), neither severe enough to block round-2 dispatch.

Severity legend: CONFIRMED = concrete counterexample with file:line evidence that defeats the plan as written; POSSIBLE = real risk with credible repro path, builder should resolve; NIT = stylistic / clarity / discoverability concern; REFUTED = attack does not land; EXHAUSTED = attack family explored, no counterexample found.

---

## 1. Attack Results

### Attack 1 — Supersede CLI flags beyond `--reason` (NULL / REFUTED)

The orchestrator's prompt suggested the planner may have missed `--dry-run` / `--note` / metadata-bag args on the existing supersede CLI. Investigation: `cmd/till/main.go:876` shows the supersede subcommand declares EXACTLY one flag — `--reason` (StringVar bind). No `--dry-run`, no `--note`, no `--override-*`. The plan's `Reason *string` args-struct field IS the complete arg set. Attack is NULL.

**Evidence:**
- `cmd/till/main.go:874-876`: `RunE: actionItemMutationRunE("supersede")` then `actionItemSupersedeCmd.Flags().StringVar(&actionItemOpts.reason, "reason", "", ...)` — only one flag.
- `internal/app/service.go:1815`: `Service.SupersedeActionItem(ctx, actionItemID, reason string)` — service signature also takes only those two args plus ctx.
- `internal/adapters/mcp_common/mcp_surface.go:354-358`: `SupersedeActionItemRequest` carries `ActionItemID`, `Reason`, `Actor` — no other transport fields.

**Verdict:** REFUTED. The plan's single-`Reason` arg-struct addition is complete.

---

### Attack 2 — `goleak.VerifyTestMain(m)` interaction with parallel siblings (POSSIBLE, MEDIUM)

The dispatcher package has 32 `_test.go` files containing `t.Parallel()` calls plus several tests that drive `d.Start(...)` which spawns long-lived per-project subscriber goroutines (`internal/app/dispatcher/subscriber.go:80-87` — one `go d.runSubscriberLoop` per project). `goleak.VerifyTestMain(m)` runs AFTER all tests in the package complete (per goleak's contract). If ANY test in the dispatcher package leaks a goroutine that does not drain before `os.Exit` fires, the verifier fails the entire `mage test-pkg internal/app/dispatcher` run — including tests unrelated to the new e2e file.

**Evidence:**
- `rg "func TestMain" internal/app/dispatcher/` → 0 matches. The package currently has NO `TestMain` function. Round-2 plan creates one in the new `dispatcher_e2e_test.go` file, which will be the ONLY `TestMain` in the package.
- `rg "\.Start\(" internal/app/dispatcher/{dispatcher_test.go,subscriber_test.go}` → 12 sites in the test suite call `d.Start(context.Background())`. Each starts long-lived subscriber goroutines.
- `rg -l "go func\(\)" internal/app/dispatcher` → 11 source/test files spawn goroutines via `go func()` (broker_sub_test.go, monitor_test.go, gates_test.go, subscriber_test.go, locks_*_test.go, commit_agent.go, etc.).
- `subscriber.go:124-127` and `subscriber.go:80-87` — `Start` spawns N goroutines, `Stop` drains via `sync.WaitGroup`. If any test forgets `t.Cleanup(d.Stop)` or a context.Background() spawn outlives its test (broker_sub_test.go line patterns), `VerifyTestMain` catches it package-wide.

**Why this matters:** R6.2's intent is to catch goroutine leaks in the renamed `TestAutoDispatch_NewDispatcherGateWiring` + `TestAutoDispatchE2EGateFailViaNewDispatcher` paths. But `VerifyTestMain(m)` casts a wider net — it asserts no leaks across ALL ~25 dispatcher test files. This is more rigorous than R6.2 requested, AND it may surface pre-existing leaks unrelated to the e2e tests. PLAN.md D1.3 line 84 already documents the `VerifyNone(t)` fallback if `TestMain` causes sibling inflation, so the planner anticipates this risk. The fallback is correct.

**Recommended mitigation (NIT-level enhancement to acceptance):** PLAN.md D1.3 acceptance bullet 5 says *"Goleak is wired — either via `TestMain` + `goleak.VerifyTestMain(m)` or via `goleak.VerifyNone(t)` at end of each test body."* Strengthen by adding: *"If `VerifyTestMain(m)` surfaces a pre-existing leak in a non-e2e dispatcher test, builder MUST fall back to per-test `VerifyNone(t)` rather than 'fix' the unrelated test inline — pre-existing leaks are out of scope for this drop."* This prevents scope creep where builder fixes an unrelated leak then breaks something else.

**Verdict:** POSSIBLE, MEDIUM. Builder has documented fallback; risk is bounded. Add scope-creep guard in acceptance.

---

### Attack 3 — Interface widening breaks `stubExpandedService` test fake (NIT, LOW)

Adding `SupersedeActionItem` to `mcpcommon.ActionItemService` widens the interface. The package `internal/adapters/mcp_rpc` has a fully-implementing test fake at `internal/adapters/mcp_rpc/extended_tools_test.go:33-665+` (`stubExpandedService`) which currently provides `GetActionItem`, `ListActionItems`, `CreateActionItem`, `UpdateActionItem`, `MoveActionItem`, `MoveActionItemState`, `DeleteActionItem`, `RestoreActionItem`, `ReparentActionItem`, `ListChildActionItems`, `ResolveActionItemID`, `GetProjectBySlug`. Once the interface gains `SupersedeActionItem`, this stub must also gain it OR the entire mcp_rpc package fails compile (`*stubExpandedService` no longer satisfies `mcpcommon.ActionItemService`).

**Evidence:**
- `rg "ActionItemService" -l` → ONE production implementer (`AppServiceAdapter` via `pickActionItemService` in `handler.go:1034-1042`), plus ONE test implementer (`stubExpandedService` in `extended_tools_test.go`). No other mocks.
- `instructions_tool_test.go:13` and `extended_tools_test.go:33` are the two consumers of `stubExpandedService` as an `ActionItemService`.
- `AppServiceAdapter` already has `SupersedeActionItem` at `app_service_adapter_mcp.go:1051-1075`, so the interface widening lands without adapter changes.

**Why this is NIT not CONFIRMED:** PLAN.md D1.6 already declares `extended_tools_test.go` as an edited path. The builder will discover the compile failure on first build attempt and add the stub method as part of D1.6's work. The compile failure is loud, not silent. BUT the acceptance criteria do NOT explicitly list "add `SupersedeActionItem` stub method to `stubExpandedService` to maintain interface satisfaction." A builder reading acceptance bullets verbatim could miss this and waste cycles diagnosing a compile error before realizing the stub needs widening.

**Recommended mitigation:** Add to D1.6 acceptance: *"`stubExpandedService` in `extended_tools_test.go` gains a `SupersedeActionItem(ctx, mcpcommon.SupersedeActionItemRequest) (domain.ActionItem, error)` method (deterministic stub return) so the package continues to compile after the interface widening in D1.5."*

**Verdict:** NIT. Compile-loud catch, but acceptance should pre-empt the diagnostic cost.

---

### Attack 4 — `"supersede_task"` collision with existing supersede CLI surface (REFUTED + NIT side effect)

The orchestrator's prompt asked whether the existing human-only supersede CLI uses an action-string that would collide with D1.6's new `"supersede_task"`. Investigation: the existing CLI does NOT route through `authorizeMCPMutation` at all — `runActionItemSupersede` (cmd/till/main.go:2604-2607) goes through the CLI's local capability-guard chain (`Service.SupersedeActionItem` calls `enforceMutationGuardAcrossScopes` with `CapabilityActionMarkComplete`, NOT through an MCP-action-string layer). No string-collision risk.

**Evidence:**
- `rg "supersede_task"` → 0 matches in production code (and 0 in tests). The string is genuinely new with D1.6.
- `cmd/till/main.go:2604-2607` shows the supersede CLI dispatch path; it does NOT call any `authorizeMCPMutation`-style helper.
- `internal/app/service.go:1832`: `enforceMutationGuardAcrossScopes(..., CapabilityActionMarkComplete)` — service-layer guard reuses an existing capability action; no string layer.

**However — NIT side effect (CONFIRMED minor):** TWO production source files contain doc-comments that become STALE the moment D1.6 lands:

1. `cmd/till/main.go:850`: `// invoking this path today — no MCP tool registration exposes supersede`
2. `internal/adapters/mcp_common/mcp_surface.go:351`: `// today; no MCP tool registration exposes supersede so agent-driven flows cannot reach it.`

D1.6 directly contradicts both claims. The plan does NOT include path updates to these files for the doc-comment correction. Code-reader will read these and conclude "MCP cannot reach supersede," then be confused by the MCP enum in `extended_tools.go:1440` that says otherwise.

**Recommended mitigation:** Add `cmd/till/main.go` (line 844-851 doc-comment block) and `internal/adapters/mcp_common/mcp_surface.go` (line 350-353 doc-comment block) to D1.6's paths. Update both comments to say something like "Pre-D1.6 the CLI was the only surface; D1.6 added `till.action_item operation=supersede` MCP exposure." Builder applies the textual update; no code logic changes.

**Verdict:** REFUTED on collision; NIT (CONFIRMED) on stale doc-comments. Add doc-comment update to D1.6 paths.

---

### Attack 5 — R7.3 stub-only acceptance leaves future refinement evaporating (CONFIRMED, LOW)

PLAN.md D1.4 Notes (line 115) says *"A future refinement adds a test in `cmd/till/main_test.go` (where `dispatcherTemplateResolver` lives) asserting per-project routing via the real resolver."* This deferral exists ONLY as prose inside PLAN.md.

**Evidence:**
- `rg "dispatcherTemplateResolver" REFINEMENTS.md` → 1 match (existing R8 entry from a DIFFERENT drop covering a DIFFERENT concern — D3-to-D4 test-tightness routing). The F5 future-refinement for the per-project routing test is NOT in REFINEMENTS.md.
- `rg "test_cleanup|4b_test_cleanup" REFINEMENTS.md` → 0 matches. No drop-4b-test-cleanup-scoped entries.
- The drop directory does not contain a refinements-aside file; CLOSEOUT.md exists but is empty per the `feedback_no_closeout_md_pre_dogfood` rule.

**Why this matters:** PLAN.md prose is technically durable (git-tracked file in `workflow/drop_4b_test_cleanup/`), but the discoverability is poor — future drops looking for parked refinements consult REFINEMENTS.md or memory entries, not drop-archive PLAN.md inline prose. The F5 refinement will be silently dropped on the floor.

**Recommended mitigation:** Before D1.4 ships, add a `REFINEMENTS.md` entry with shape:

```markdown
## 2026-05-18 — drop_4b_test_cleanup — Real-resolver per-project routing test not reachable from internal/app/dispatcher

### Context
Drop 4b test-cleanup parameterized `stubE2ETemplateResolver` for per-project routing. The real `dispatcherTemplateResolver` lives in `package main` at `cmd/till/main.go:2704` and cannot be imported from `internal/app/dispatcher`.

### Observation
`TestStubE2ETemplateResolverRoutesPerProject` proves the test stub routes per project. Production `dispatcherTemplateResolver` per-project routing is NOT covered. `TestDispatcherTemplateResolverAdapter` at `cmd/till/main_test.go:3363` tests the resolver positively but not per-project.

### Proposed fix
Add `TestDispatcherTemplateResolverPerProjectRouting` in `cmd/till/main_test.go` exercising the real resolver with two projects and asserting per-project return values.

### Target drop
Parking lot / opportunistic absorption when next touching `cmd/till/main.go` or `cmd/till/main_test.go`.

### Tags
`dispatcher-tests`, `cmd-till`, `parking-lot`
```

Alternatively, add a memory entry — but the project's convention is REFINEMENTS.md for cross-drop technical refinements.

**Verdict:** CONFIRMED, LOW. File the refinement durably before the drop closes.

---

### Attack 6 — D1.1 alias normalization silent vs loud (NIT, LOW)

PLAN.md D1.1 adds an `"actionitem"` → `"action_item"` alias inside `NormalizeCommentTargetType` to honor the schema-declared camelCase form. The plan does NOT add a deprecation `log.Warn` or even a `// DEPRECATED:` code comment.

**Evidence:**
- PLAN.md D1.1 Changes (line 32-34): adds a switch/map step but does not specify a log call.
- The proposed change makes BOTH `"actionItem"` (camelCase, alias) and `"action_item"` (canonical) valid forever — no migration plan to remove the alias.
- Code-readers 6 months out will see callers passing `"actionItem"` and won't know it's a pre-Drop-1.75 back-compat band-aid versus the canonical form.

**Why this matters:** Silent aliases are a known code-clarity drift pattern. The Drop 1.75 collapse already addressed pre-1.75 vocabulary drift; introducing a NEW silent alias re-creates the same class of confusion at a smaller scale. The dev rule `feedback_no_migration_logic_pre_mvp.md` argues against log warnings ("dev wipes DB on schema/state-vocab change"), but that rule covers persistence migration, not in-code naming-convention drift documentation.

**Recommended mitigation:** Add a code comment above the alias map step:
```go
// DEPRECATED: "actionItem" is the pre-Drop-1.75 schema-declared form,
// kept as an alias here for back-compat with stale MCP schema strings.
// Callers should use "action_item" (canonical). Remove the alias when
// the schema enum drops "actionItem" entirely.
```
This is purely a code comment — no functional change, no log call required.

**Verdict:** NIT. Code-clarity improvement, not a correctness gap.

---

### Attack 7 — `go get goleak` prereq ordering (REFUTED)

PLAN.md D1.3 line 78 explicitly declares: *"DEV-ACTION PREREQ: Before this droplet builds, the dev must run `go get go.uber.org/goleak` in the `main/` worktree shell. The goleak library is NOT in `go.mod` (confirmed). Builder blocks until the dep is available."*

**Evidence:**
- `rg "goleak" go.mod go.sum` → 0 matches. Confirmed absent.
- PLAN.md acceptance for D1.3 (line 90-95) doesn't explicitly list "go.mod contains goleak" as a check, but the package will fail to compile without the dep, so this is a structural pre-build gate.
- The orchestrator's attack prompt asks about "what if builder fires without dev running `go get`?" Answer: builder's first build attempt fails on missing module. PLAN.md's "Builder blocks until the dep is available" anticipates exactly this.

**Verdict:** REFUTED. Prereq is explicitly documented; orchestrator gates the dev-action before spawning D1.3.

---

### Attack 8 — `till.action_item operation` enum currently missing `"supersede"` (REFUTED)

PLAN.md D1.6 step 1 (line 170) explicitly adds `"supersede"` to the enum at `extended_tools.go:1440`. Plan acceptance bullet 2 (line 184) confirms.

**Evidence:**
- `extended_tools.go:1440`: current enum is `mcp.Enum("get", "list", "search", "create", "update", "move", "move_state", "delete", "restore", "reparent")` — confirmed missing `"supersede"`.
- PLAN.md D1.6 step 1 makes the exact change.

**Verdict:** REFUTED. Plan covers this correctly.

---

### Attack 9 — `mcp_surface.go:848-861` interface line range (REFUTED)

Verified by direct Read. Interface declaration at lines 847-861 (`ActionItemService` from line 847 doc-comment to line 861 closing brace). Line 857 is `ReparentActionItem`. PLAN.md's "add after `ReparentActionItem` at line 857" reference is accurate as of HEAD `2124d2c`.

**Evidence:**
- Direct read of `mcp_surface.go:840-862` shows `ActionItemService` interface spanning 847-861 with `ReparentActionItem` at 857.
- D1.5 has no upstream blockers that would shift these lines mid-drop.
- Within-drop, D1.6 is `blocked_by D1.5` (per `_BLOCKERS.toml`), so D1.6 reads the new interface state after D1.5 lands. No line-drift race.

**Verdict:** REFUTED. Line citations are accurate and protected by within-drop blocker chain.

---

### Attack 10 — Cross-drop file overlap with `drop_fe_1_bootstrap` (REFUTED)

`drop_fe_1_bootstrap/PLAN.md` touches `magefile.go` (lines 57-58, 64). This drop's paths:
- `internal/domain/comment.go`, `internal/domain/comment_test.go`
- `internal/adapters/mcp_rpc/extended_tools.go`, `extended_tools_test.go`
- `internal/app/dispatcher/subscriber_test.go`, `dispatcher_e2e_test.go` (new)
- `internal/adapters/mcp_common/mcp_surface.go`, `app_service_adapter_lifecycle_test.go`

Set intersection: empty. `magefile.go` is NOT in this drop's paths. FE drop's `ui/` path tree is disjoint from this drop's `internal/` tree.

**One caveat:** D1.3 implicitly requires `go.mod` + `go.sum` to be modified by the dev's `go get go.uber.org/goleak`. FE drop does not touch `go.mod` per its declared paths. No conflict.

**Evidence:**
- FE PLAN.md `rg "internal/app/dispatcher|internal/domain/comment|internal/adapters/mcp_rpc|internal/adapters/mcp_common"` → 0 production-path matches in FE PLAN.md (only references in FE QA reports analyzing cross-drop).
- FE drop's own QA falsification R2 explicitly recognized disjoint paths (per FE drop's `PLAN_QA_FALSIFICATION_R2.md:214`: "Set intersection: empty. No file overlaps; no package overlaps.").

**Verdict:** REFUTED. Cross-drop interaction is disjoint.

---

## 2. Summary of Findings

| Attack | Verdict | Severity | Action |
|---|---|---|---|
| 1 — Args struct missing fields beyond `Reason` | REFUTED | — | None |
| 2 — `VerifyTestMain` package-wide leak detection | POSSIBLE | MEDIUM | Add scope-creep guard to D1.3 acceptance |
| 3 — `stubExpandedService` interface widening | NIT | LOW | Add stub-method bullet to D1.6 acceptance |
| 4a — `"supersede_task"` collision | REFUTED | — | None |
| 4b — Stale "no MCP exposes supersede" doc-comments | CONFIRMED (NIT) | LOW | Add 2 file paths to D1.6 + textual update |
| 5 — F5 future refinement not filed durably | CONFIRMED | LOW | Add REFINEMENTS.md entry before drop closes |
| 6 — D1.1 alias silent vs loud | NIT | LOW | Add `// DEPRECATED:` code comment |
| 7 — `go get goleak` prereq ordering | REFUTED | — | None |
| 8 — `till.action_item` enum missing supersede | REFUTED | — | None |
| 9 — `mcp_surface.go:848-861` line range | REFUTED | — | None |
| 10 — Cross-drop overlap with `drop_fe_1_bootstrap` | REFUTED | — | None |

**Round-1 carry-forward verification:**
- F1 (round-1) addressed: PLAN.md D1.5 line 135-141 now explicitly states "adapter method ALREADY EXISTS" and scopes D1.5 to interface-addition-only.
- F2 (round-1) addressed: PLAN.md D1.4 R7.1 line 109 removes the pre-authorized scope-deflation and requires dev sign-off for fallback.
- F3 (round-1) addressed: PLAN.md D1.6 line 181 adds the 5th test case pinning `ErrTransitionBlocked` for non-failed items.
- F4 (round-1) addressed: PLAN.md D1.3 line 78 declares the `go get goleak` DEV-ACTION PREREQ explicitly; D1.3 wires goleak in.

All four round-1 CONFIRMED counterexamples are resolved.

**Round-2 unprompted-discovery verification:**
- `Reason *string` field added to args struct (PLAN.md D1.6 step 3, line 172): correct and complete per Attack 1.
- `"supersede_task"` action-string chosen (PLAN.md D1.6 step 4, line 173): naming-consistent with `restore_task`, `reparent_task`, `delete_task`, `create_task` precedents; no collision risk per Attack 4.

Both unprompted discoveries are sound.

---

## 3. Verdict

**PASS WITH NITS.**

The round-2 PLAN.md correctly addresses all four round-1 CONFIRMED counterexamples and lands two correct unprompted discoveries. No CONFIRMED counterexample against round-2 specifically.

The two CONFIRMED-NIT findings (F4b stale doc-comments; F5 unfiled refinement) are durable but small — they can be absorbed into the build phase by the orchestrator OR rolled into D1.6 acceptance text, NOT a re-planning round. Recommend the orchestrator:

1. Append paths `cmd/till/main.go` and `internal/adapters/mcp_common/mcp_surface.go` to D1.6 with a one-line directive to update the two stale doc-comments.
2. Stage a `REFINEMENTS.md` entry (text in F5 above) before this drop closes.
3. Optionally add the D1.3 scope-creep guard from Attack 2 mitigation and the D1.6 `stubExpandedService` stub-method bullet from Attack 3 mitigation.

If the orchestrator prefers a strict "no plan edits between QA rounds" policy, dispatch builders against round-2 PLAN.md as-is; the NITs surface during build-QA naturally (compile-loud for Attack 3, doc-comment review for Attack 4b, REFINEMENTS.md check for Attack 5).

---

## 4. Hylla Feedback

Hylla MCP is back on per 2026-05-18 directive. Tried `mcp__plugin_context7_context7__query-docs` against `/uber-go/goleak` for `VerifyTestMain` semantics — `library "/uber-go/goleak" not found`. Context7 doesn't cover goleak directly; resolved-id lookup returned unrelated uber-go libraries (zap, fx, guide, mock, h3) — none for goleak. Fell back to PLAN.md / REVISION_BRIEF / direct source inspection. This is a Context7 ergonomics issue, not Hylla.

**Hylla**: did NOT use Hylla MCP this round. The investigation was driven by `rg` keyword searches across the working tree + direct `Read` calls on specific files (PLAN.md, REVISION_BRIEF.md, source files at known line ranges, `_BLOCKERS.toml`, REFINEMENTS.md). For text-keyword scans across 30+ Go files in a single package and PLAN.md prose, `rg` was the natural fit — Hylla would have been slower for the bulk text scan and equivalent for the targeted source reads. No Hylla miss to log because no Hylla query was issued.

**Bash-grep blocked**: the environment blocked plain `grep -rn` and `grep -l` Bash invocations early in the round. `rg` (ripgrep) Bash invocations worked. This is a session-level permission quirk, not a Hylla concern. Recorded here for orchestrator awareness.

**One Context7 miss:** the `goleak` library appears genuinely absent from Context7's index. Suggestion: add `/uber-go/goleak` (or the canonical path) to Context7 if dev cares about future agents querying its docs. For this round the fallback (Reading PLAN.md + source code) sufficed.
