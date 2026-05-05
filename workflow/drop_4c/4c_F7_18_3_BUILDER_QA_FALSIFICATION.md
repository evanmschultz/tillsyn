# 4c.F.7.18.3 — Context Aggregator Engine — QA Falsification

**Reviewer:** go-qa-falsification-agent (subagent-mode under STEWARD orchestrator).
**Build under review:** new package `internal/app/dispatcher/context/` — `aggregator.go` (~22K, 554 LOC), `rules.go` (~9.4K, 305 LOC), `aggregator_test.go` (~24K, 14 tests).
**Worklog:** `workflow/drop_4c/4c_F7_18_3_BUILDER_WORKLOG.md`.
**Plan source:** `workflow/drop_4c/F7_18_CONTEXT_AGG_PLAN.md` §F.7.18.3 + REVISIONS POST-AUTHORING (REV-13 inherits from `F7_CORE_PLAN.md:1078`).
**Mode:** Read-only adversarial review; no Hylla calls; no code edits.

## Round 1

### Hard preconditions verified

- Package `internal/app/dispatcher/context/` exists with the three claimed files (no extras).
- `git status --porcelain` shows the package + worklog + QA artifacts as **untracked** (`??`). `git log --oneline -5` shows last commit `e19e9f0 docs(drop-4c): add f.7.2 qa proof and falsification artifacts`. **No F.7.18.3 commit landed by the builder.** REV-13 honored.
- `mage testPkg ./internal/app/dispatcher/context/...` returns 14/14 PASS.

### Per-attack verdicts

#### A1 — Empty-rule silent-skip vs marker-silence — **NIT**

The builder skips empty rule output entirely (no file, no marker, no cap charge — `aggregator.go:362-368`). The plan's spec table at L201 (SKETCH) and the F7_18 acceptance bullets only mention markers for **truncation / skip-via-cap / timeout**. The "rule fired but produced empty content" case is **not specified**, so silent-skip is a defensible design choice. The plan's own §F.7.18.3 test scenarios at line 51 confirm the choice: "Parent without start/end commits → empty content, no file written, no error."

Adversarial concern: an adopter with `parent_git_diff = true` who hits an empty-commit case has no signal that the rule fired. Counterargument: the plan rejects "rule-fired-but-empty" markers because they would be noise on every empty-binding spawn.

Not a falsification — record as a design decision worth surfacing in a future refinement (`add a "[rule X fired with empty content]" marker behind an opt-in flag` if adopters request it).

#### A2 — Greedy-fit + per-rule full-file disk size — **REFUTED**

Per-rule full content is written to `Files["<rule>.full"]` regardless of bundle-cap pressure (`aggregator.go:397-399, 415-417`). For two oversized rules the spawn-bundle directory holds rendered + truncated + multiple full-content files. The spec at SKETCH:201 explicitly defines `.full` as off-budget ("full content at <bundle>/context/<rule>.full"). Downstream constraint analysis (claudeAdapter `--system-prompt-file` size limit, sandbox writability cap) is **out of scope for this droplet** — the aggregator's contract ends at producing the Bundle struct.

If a downstream constraint surfaces, that's an integration finding for F.7.18.4 / F.7.3b, not a defect here.

#### A3 — Inner-deadline clamp via outer context — **REFUTED**

`aggregator.go:331` uses `stdcontext.WithTimeout(outerCtx, ruleDuration)`. Go stdlib semantics for `WithTimeout` derived from a parent context that already has a deadline: the child's deadline is `min(parent.Deadline, time.Now()+ruleDuration)`. No bespoke clamp logic needed; the engine relies on stdlib correctly. `TestResolvePerBundleTimeout` exercises this path (BundleDuration=50ms, MaxRuleDuration=1s — outer wins).

#### A4 — `context.WithTimeout` cleanup / goroutine leaks — **REFUTED**

- Outer: `aggregator.go:277-278` — `defer cancelOuter()`.
- Per-rule: `aggregator.go:331-333` — `cancelRule()` is called **immediately** after `evaluateRule` returns, before any branching. Every error / non-error path runs **after** `cancelRule()`.

No leaked timer goroutines.

#### A5 — Resolver context observance under per-rule timeout — **REFUTED**

- `resolveParent` → `reader.GetActionItem(ctx, …)` (mock honors ctx via `select`/`time.After` at lines 44-48).
- `resolveSiblingsByKind` → `reader.ListSiblings(ctx, …)` (mock checks `ctx.Err()` at line 67-69).
- `resolveAncestorsByKind` → per-hop `reader.GetActionItem(ctx, …)`; ctx propagates per call.
- `resolveDescendantsByKind` → `ctx.Err()` polled at the **top of every pop iteration** (`rules.go:193-195`) before the next reader call.

Production reader contract MUST honor ctx (file-level or domain-port concern, not aggregator concern). Within the aggregator's contract surface, every ctx-aware call site is wired.

NIT: `resolveSiblingsByKind` does pure compute on the returned slice without re-polling ctx, but the slice is bounded by adapter limits (single SQL `SELECT children`); not a realistic blow-budget vector.

#### A6 — `siblings_by_kind` latest-round tie-break — **REFUTED, NIT on coverage**

`rules.go:96-101` tie-breaks on lex-greater ID when `CreatedAt.Equal()`. Deterministic across runs. **However: no test exercises the `CreatedAt.Equal && ID >` branch.** Implementation is correct; coverage is missing for the explicit tie path. Classify as coverage NIT — implementation REFUTED.

#### A7 — `ancestors_by_kind` cycle guard — **REFUTED, NIT on coverage**

`rules.go:144-162`: `visited` map keyed by UUID, checked before the next hop; cycle (A.parent=B, B.parent=A) terminates at the second visit of A with `fmt.Errorf("ancestors_by_kind: cycle detected at %s", currentID)`. Walk also caps at 256 hops as defense-in-depth.

**Counterexample attempt: cycle case.** Walking A→B→A would: hop=0 currentID=B, mark B; hop=1 currentID=A, mark A; hop=2 currentID=B, found in `visited`, return cycle error. Correct.

Cycle path lacks a unit test. Implementation REFUTED; coverage gap is a NIT.

#### A8 — `descendants_by_kind` DFS depth + uncancellable adapter — **REFUTED**

`rules.go:182-217`. `maxNodes=4096` cap; per-iteration `ctx.Err()` poll (line 193); standard DFS with explicit stack. A pathologically deep tree of 4096 nodes at 1ms reader latency = ~4s, well past the 500ms per-rule default — `ctx.Err()` catches it between calls and returns the ctx error, which the engine maps to a per-rule timeout marker.

ListChildren returning a huge slice is **not** uncancellable from the aggregator's perspective; the adapter is responsible for honoring ctx during slice materialization. That's an adapter contract, not an aggregator-layer bug.

#### A9 — Mock ActionItemReader thread-safety with `t.Parallel()` — **REFUTED**

Each test constructs its own `mockReader` via `newMockReader()` (lines 28-33). No shared mock state across tests. Map mutations only happen during test setup before `Resolve` is called; the mock's `GetActionItem` / `ListChildren` / `ListSiblings` only **read** the maps. Safe for `t.Parallel()`.

#### A10 — Builder commit policy — **REFUTED**

Per `F7_CORE_PLAN.md:1078-1082` REV-13: "Builder spawn prompts MUST explicitly forbid self-commit." Worklog explicitly states `**NO commit** by builder (per F.7-CORE REV-13)`. `git log --oneline -5` confirms last commit is `e19e9f0` (F.7.2 QA artifacts), pre-dating this droplet. `git status --porcelain` shows the new files as `??` (untracked).

REV-13 honored.

#### A11 — Marker accumulation ordering — **REFUTED**

Markers are appended to `bundle.Markers` in encounter order:
1. Truncation marker (`aggregator.go:377-380`) — emitted **before** the greedy-fit check.
2. Skip-via-cap marker (`aggregator.go:389-392`) — emitted if a rule busts after truncation.
3. Per-rule-timeout marker (`aggregator.go:352-355`) — emitted on inner ctx.DeadlineExceeded with outer alive.
4. Outer-timeout marker (`aggregator.go:344-347`) — emitted on outer-dead, then `break`.

For `delivery=inline`, markers append after rendered content (lines 439-449). For `delivery=file`, markers ARE the inline payload. `TestResolveDeclarationOrderStable` directly exercises declaration-order marker emission across all 5 rules.

#### A12 — `Bundle.Files` filename collision — **REFUTED**

Filenames are `<rule>.md` (rendered) and `<rule>.full` (pre-truncation full content). The five rule names are constants (`parent`, `parent_git_diff`, `siblings_by_kind`, `ancestors_by_kind`, `descendants_by_kind`) — distinct, no collision possible from a single binding.

`<rule>.full` is written **either** on the skip-with-truncation path (line 398) **or** in the post-loop stitch (line 416), never both — they're mutually exclusive. No double-write.

#### A13 — Memory-rule conflicts — **REFUTED**

- `feedback_no_migration_logic_pre_mvp.md`: pure-Go function, no SQLite migration, no `till migrate` CLI. ✓
- `feedback_subagents_short_contexts.md`: single package, ~600 LOC + ~300 LOC, single round ✓
- `feedback_opus_builders_pre_mvp.md`: plan specifies opus model (`F7_18_CONTEXT_AGG_PLAN.md:195`); worklog inherits ✓
- `feedback_no_closeout_md_pre_dogfood.md`: no CLOSEOUT/LEDGER/WIKI_CHANGELOG written ✓
- `feedback_never_remove_workflow_files.md`: only NEW files added; no deletions ✓

#### A14 — Coverage 81.8% — what's the 18.2% — **NIT**

Mock fields `getErr` (`aggregator_test.go:25`) and `diffErr` (`aggregator_test.go:78`) exist but are **never set in any test**. Consequence: every error-bubble-up path (`rules.go:24-26`, `:43-45`, `:53-55`, `:77-79`, `:154-156`, `:208-210`) is uncovered. The non-timeout error branch in `aggregator.go:358-359` is also uncovered.

These are straightforward `fmt.Errorf("...: %w", err)` wrappings — low risk, but the worklog overstates "above 70% gate" without mentioning the uncovered branches all share a category (port-error propagation). A future NIT-fix round adds three table-driven tests:

- Reader-returns-error during `resolveParent` → `Resolve` returns wrapped error.
- DiffReader-returns-error during `resolveParentGitDiff` → `Resolve` returns wrapped error.
- ListChildren-returns-error during `resolveDescendantsByKind` → `Resolve` returns wrapped error.

Cycle-detection (A7) and equal-CreatedAt-different-ID tie-break (A6) also lack tests; same NIT bucket.

#### A15 (added during self-review) — Concurrent rule evaluation / Bundle race — **REFUTED**

`Resolve` is single-threaded — the rule-iteration loop processes one rule at a time, builds `renderedRules` slice serially, and stitches the output post-loop. No goroutines spawned for parallel rule evaluation. `Bundle.Files` and `Bundle.Markers` only ever mutated by the calling goroutine. Safe.

### CONFIRMED counterexamples

**None.**

### NITs

- **A1** — Empty-rule output is silently skipped; consider opt-in marker for adopters who want fired-but-empty signal.
- **A6** — Latest-round tie-break (equal CreatedAt + lex-greater ID) is implemented but has no unit test.
- **A7** — Cycle guard in `resolveAncestorsByKind` works correctly but has no unit test.
- **A14** — Coverage 81.8%; uncovered branches concentrate in port-error-propagation paths (mock `getErr` / `diffErr` fields exist but unused). Three table-driven tests would close the gap.

### Summary

**Verdict: PASS-WITH-NITS.**

No CONFIRMED counterexamples. The four NITs (A1 design choice, A6/A7 untested code paths, A14 port-error coverage) are coverage / design-discussion items, not blockers. Builder honored REV-13 (no self-commit), the empty-binding fast path is correctly wired, the greedy-fit cap matches the SKETCH:202 spec, and the two-axis wall-clock contract is enforceable through stdlib `context.WithTimeout`.

Recommend the orchestrator dispatch a small follow-up build-drop (or fold into F.7.18.4) to land:

1. Mock-error tests (`getErr` / `diffErr` paths).
2. Cycle-guard test for `resolveAncestorsByKind`.
3. Equal-CreatedAt tie-break test for `resolveSiblingsByKind`.

These are non-blocking; the aggregator engine ships as-is for F.7.18.4 to layer the cap algorithm on top.

## Hylla Feedback

N/A — review touched Go source files but resolved every cross-reference via `Read` on cited line numbers from the worklog + plan + recent-commit context. No Hylla queries issued. The active source files are uncommitted (untracked), so Hylla wouldn't have indexed them yet anyway — `Read` is the correct tool.
