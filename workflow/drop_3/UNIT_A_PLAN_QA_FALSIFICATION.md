# DROP_3 UNIT A — Plan QA Falsification, Round 1

**Verdict:** fail
**Date:** 2026-05-02

## Counterexamples

### C1 — CONFIRMED — Required-on-create cascade breaks ~96 NewActionItem call-sites across 6 packages; 3.A.2 paths gate only `internal/domain`

`NewActionItem` is called from 12 distinct files spanning 6 packages:

| File | Calls |
| --- | --- |
| `internal/app/service_test.go` | 56 |
| `internal/adapters/storage/sqlite/repo_test.go` | 22 |
| `internal/app/snapshot_test.go` | 7 |
| `internal/app/service.go` (production caller) | 1 |
| `internal/domain/domain_test.go` | (multiple) |
| `internal/app/attention_capture_test.go` | 1 |
| `internal/app/dotted_address_test.go` | 2 |
| `internal/app/embedding_runtime_test.go` | 2 |
| `internal/adapters/storage/sqlite/embedding_jobs_test.go` | 2 |
| `cmd/till/embeddings_cli_test.go` | 2 |
| `internal/tui/model_test.go` | 2 (via `newActionItemForTest` wrapper) |
| `internal/domain/action_item.go` | (definition) |

Verified via `git grep -ln "NewActionItem("`. Concrete spot-check at `cmd/till/embeddings_cli_test.go:115-127`: a fixture constructs `domain.NewActionItem(domain.ActionItemInput{ID, ProjectID, Kind, ColumnID, Position, Title, ...})` with no `StructuralType`. Once 3.A.2 lands the `if in.StructuralType == "" { return ActionItem{}, ErrInvalidStructuralType }` branch, that fixture immediately starts returning `ErrInvalidStructuralType` and the test fatal-errors at the `t.Fatalf("NewActionItem() error = %v", err)` line.

The 3.A.2 droplet:
- declares `Paths: internal/domain/action_item.go, internal/domain/domain_test.go` only;
- declares `Acceptance: mage test-pkg ./internal/domain green` only;
- declares `Sweep domain_test.go for every other domain.NewActionItem call site...` — but `domain_test.go` is one file out of 12.

A builder taking 3.A.2's spec literally edits two files, runs `mage test-pkg ./internal/domain` green, and ships the droplet "complete." Then the very next droplet 3.A.3 fails at `mage test-pkg ./internal/adapters/storage/sqlite` because `repo_test.go`'s 22 fixtures all reject. 3.A.4's `mage test-pkg ./internal/app && ./internal/adapters/server/common && ./internal/adapters/server/mcpapi` similarly cascade-fails on stale fixtures from `service_test.go` (56 calls), `attention_capture_test.go`, `dotted_address_test.go`, `embedding_runtime_test.go`. 3.A.5's snapshot package fails on `snapshot_test.go`. The `cmd/till/embeddings_cli_test.go` and `internal/tui/model_test.go` fixtures only get caught by `mage ci` at drop end.

**Why this matters now:** subsequent droplets (3.A.3 / 3.A.4 / 3.A.5) all `blocked_by: 3.A.2` but each one's own builder will be confronted with a broken sibling-package test suite that the prior droplet's "green" gate didn't catch. The test-fixture sweep is fundamentally **3.A.2's job** because 3.A.2 is the droplet that introduces the breaking change.

**Mitigations the planner needs to add to 3.A.2 specifically (preferred order):**

1. **Expand 3.A.2's `Paths` to enumerate every test file calling `NewActionItem`** across all 6 affected packages, OR
2. **Add a `newActionItemForTest`-style helper** mirroring the Drop 1.75 `Kind` precedent at `internal/tui/model_test.go:14674-14687` (planner-missed precedent — see C2 below) so test fixtures default `StructuralType` to a sensible value (e.g. `domain.StructuralTypeDroplet`), letting the migration land in a single file, OR
3. **Explicitly require `mage ci` (not just `mage test-pkg ./internal/domain`) as 3.A.2's acceptance gate** so the cascade breakage surfaces inside 3.A.2's own QA round rather than leaking into 3.A.3+.

Without one of these, 3.A.2's "complete" state is a false-positive that weaponizes every downstream droplet.

### C2 — CONFIRMED — Drop 1.75 `Kind` precedent of `newActionItemForTest` wrapper not consulted for `StructuralType`

`internal/tui/model_test.go:14674-14687` contains a wrapper:

```go
func newActionItemForTest(in domain.ActionItemInput, now time.Time) (domain.ActionItem, error) {
    if strings.TrimSpace(string(in.Kind)) == "" {
        in.Kind = domain.KindPlan
    }
    if strings.TrimSpace(string(in.Scope)) == "" {
        in.Scope = domain.KindAppliesTo(in.Kind)
    }
    return domain.NewActionItem(in, now)
}
```

This is the exact pattern Drop 1.75 used to absorb the `Kind` collapse without rewriting every TUI test fixture. Unit A's `## Architectural Decisions` and `## Architectural Questions` notes do not mention this precedent or whether to adopt it. The `## Notes` cite `feedback_no_migration_logic_pre_mvp.md` for fresh-DB rule but skip the test-fixture-shim precedent that is materially relevant.

Either the wrapper pattern is the right answer (low-risk, single-file, mirrors a precedent the codebase already endorses), or the planner needs to explicitly reject it with reasoning. Silence is the failure mode.

### C3 — CONFIRMED — Cross-unit conflict on `~/.claude/agents/go-qa-falsification-agent.md` is THREE-way, planner only flags TWO-way

Unit A's `## Notes` (lines 171) flags the conflict with Unit D 5.D.1 (frontmatter pointer) and proposes `5.D.1 blocked_by 3.A.7`. Correct as far as it goes.

Missed: **Unit D 5.D.5 also edits `~/.claude/agents/*.md`** at line 163 of `UNIT_D_PLAN.md`: *"`~/.claude/agents/*.md` — full pass after 5.D.1's frontmatter reminder is added"*. So `go-qa-falsification-agent.md` receives THREE writes:

1. Unit A 3.A.7 — adds the 5 cascade-vocabulary attack vectors (~lines 95-108 of the agent file).
2. Unit D 5.D.1 — adds a one-line frontmatter pointer.
3. Unit D 5.D.5 — full sweep pass for `slice` / `build-task` / `plan-task` / `qa-check` legacy vocabulary, which is dense in the agent prompts.

The ordering must be `3.A.7 → 5.D.1 → 5.D.5` (Unit D's plan already wires `5.D.5 blocked_by: 5.D.4` and 5.D.5 implicitly waits on 5.D.1 by acceptance text). But Unit A only commits to enforcing the `3.A.7 → 5.D.1` edge. The orchestrator synthesis brief needs **`3.A.7 → 5.D.5`** as a second explicit cross-unit blocker to prevent 5.D.5 from shipping a sweep that overwrites the 3.A.7 attack-vector block before 3.A.7 lands.

**Mitigation:** add an explicit second bullet to Unit A's "CONFLICT WARNING" note: *"Unit D's 5.D.5 sweep on `~/.claude/agents/go-qa-falsification-agent.md` must also blocked_by 3.A.7."*

### C4 — CONFIRMED — `[a-z-]+` regex narrowing is incoherent

3.A.1 acceptance specifies regex `(?m)^StructuralType:\s*([a-z-]+)\s*$` and explicitly justifies *"Character class is `[a-z-]+` not `[a-z0-9-]+` because none of the four values contain digits — keeps the regex narrower than `Role`'s."*

The four values are `drop`, `segment`, `confluence`, `droplet`. None contain hyphens. None contain digits. The regex ought to be `[a-z]+`. Including `-` is a no-op for matching valid values and matches arbitrary garbage like `--` or `-segment-` which then routes through `IsValidStructuralType` and rejects.

This isn't a build-breaking bug — the validator catches anything the regex over-matches. But the planner's stated rationale ("keeps the regex narrower than Role's") is internally inconsistent: narrower would be `[a-z]+`, not `[a-z-]+`. The droplet's hyphen-only test case at `role_test.go:185-190` (which asserts `Role: -` matches the regex but rejects in `IsValidRole`) carries through to the structural-type test only because `[a-z-]+` is left in. Cleanly, the test should not exist for structural_type because the regex shouldn't match `-` in the first place.

**Mitigation:** either (a) tighten regex to `[a-z]+` and drop the hyphen-only test case, or (b) keep `[a-z-]+` but rewrite the rationale comment to say "kept `-` for symmetry with `Role`'s regex; safe because `IsValidStructuralType` rejects garbage." Either resolution is fine; the current rationale is wrong and should be corrected before the builder authors the comment.

### C5 — CONFIRMED — 3.A.4 `stubExpandedService` rejection logic breaks every existing test fixture that calls Create without `structural_type`

3.A.4 line 96 says: *"`stubExpandedService.CreateActionItem` (line 429): adds rejection logic mirroring the `Role` rejection at lines 431-433: if `args.StructuralType` is empty OR not in the closed enum, return ..."*

The current `Role` rejection at `extended_tools_test.go:431-433` is `if trimmed != "" && !IsValidRole(...)` — empty is **permitted** through the stub. Diverging from this for `StructuralType` (rejecting empty) means every existing `extended_tools_test.go` test fixture that currently calls `CreateActionItem` via the stub without supplying a `structural_type` field would suddenly start receiving `ErrInvalidStructuralType`.

The planner adds a single new test (`TestActionItemMCPRejectsEmptyOrInvalidStructuralType`) but does not enumerate which existing tests need fixture updates to supply a valid `StructuralType`. Same shape of gap as C1 but localized to the MCP test surface.

**Mitigation:** 3.A.4 acceptance must explicitly require sweeping every existing `extended_tools_test.go` test that calls `CreateActionItem` / `UpdateActionItem` through the stub and supplying a valid `StructuralType` in the request fixture, OR — preferred — defaulting `args.StructuralType` to `StructuralTypeDroplet` when empty inside the stub for backward fixture compat (the production code rejects empty; the stub is allowed to be more permissive since it's only used for boundary error-mapping tests). Match the stub's explicit role-laxity precedent.

### C6 — Nit — `## Architectural Questions` includes a question already locked

The third bullet under `## Architectural Questions (Unresolved — Route to Orchestrator)` (line 185) declares *"`StructuralType` capitalization and stored form … Confirmed."* Marked "Confirmed" inside the "Unresolved" section. Either move it up to `## Architectural Decisions (Confirmed)` or drop it. Cosmetic but adds noise to the orchestrator's synthesis pass.

### C7 — Nit — `## Hylla Feedback` ergonomic gripe is mis-classified

The planner's `## Hylla Feedback` section (line 187-189) says "None — Hylla MCP was allowed but I leaned on direct file reads." That is a non-miss only if Hylla was not the right tool here. The planner then flags an ergonomic finding ("`LSP findReferences` may be a more ergonomic starting point than `hylla_refs_find`"). Per CLAUDE.md §"Code Understanding Rules" rule 1, *"All Go code: use Hylla MCP as the primary source for committed-code understanding. If Hylla does not return the expected result on the first search, exhaust every Hylla search mode … before falling back to LSP/Read/Grep/Glob."* The planner appears to have skipped the Hylla-first attempt entirely and gone straight to direct reads + `LSP findReferences`. That's a CLAUDE.md-rule violation, not an "ergonomic-only signal."

This is not a Plan-QA-falsification of the droplets themselves — it is a process nit on how the planner sourced evidence. Logging here so the orchestrator can correct on the next planner spawn. Not material to the plan's correctness; the line-number citations themselves verify out.

## Non-Counterexamples (Attempted, Refuted)

### N1 — Hidden symbol leak (Required Attack Vector 1) — REFUTED

`git grep -l "structural_type\|StructuralType"` across `*.go *.toml *.md *.sql` returns only the four planning docs (`PLAN.md`, `workflow/drop_3/{PLAN,UNIT_A_PLAN,UNIT_B_PLAN,UNIT_C_PLAN,UNIT_D_PLAN}.md`). No existing Go code consumes `StructuralType`. Greenfield introduction confirmed.

### N2 — Closed-enum exhaustiveness (Required Attack Vector 3) — REFUTED

All 4 values (`drop | segment | confluence | droplet`) are explicitly enumerated in:
- 3.A.1 acceptance (typed constants + `validStructuralTypes` slice).
- 3.A.5 snapshot test (4 cases — one per enum value).
- 3.A.6 WIKI per-value definitions.
- 3.A.7 attack-vector block (every value referenced by name).

No droplet ships an exhaustive switch / map / slice that omits a value.

### N3 — 5 plan-QA-falsification attack vectors (Required Attack Vector 5) — REFUTED

Cross-walk of PLAN.md §19.3 lines 1644-1649 against 3.A.7 acceptance:

| PLAN.md §19.3 vector | 3.A.7 sub-bullet |
| --- | --- |
| Droplet-with-children | 1 ✓ |
| Segment overlap without `blocked_by` | 2 ✓ |
| Empty-`blocked_by` confluence | 3 ✓ |
| Confluence partial upstream coverage | 4 ✓ |
| Role/structural_type contradiction | 5 ✓ |

All five present. Sub-bullet 5 even adds 4 specific contradiction shapes (`role=qa-* on non-droplet`, `role=builder on confluence`, `role=planner on dangling droplet`, `role=commit on non-droplet`) which is a strict superset of PLAN.md's three examples. Defensible expansion.

### N4 — WIKI insertion correctness (Required Attack Vector 6) — REFUTED

Read of `main/WIKI.md` lines 1-60 confirms:
- Line 19: `## The Tillsyn Model (Node Types)` — exists exactly as cited.
- Line 36: `## Level Addressing (0-Indexed)` — exists exactly as cited.
- Lines 28-34: `### Do Not Use Other Kinds Today` and `### Do Not Use Templates Right Now` sub-sections — exist exactly as cited.

Insertion between line 34 (end of "Do Not Use Templates Right Now") and line 36 (start of "Level Addressing") is a clean h2-sibling slot. No conflict with surrounding prose. The pedagogical flow ("understand nodes" → "understand cascade vocabulary" → "understand level addressing") is internally consistent.

### N5 — `tc := tc` (Required Attack Vector 8) — REFUTED

`role_test.go` does NOT use `tc := tc` — Go 1.22+ makes the per-iteration loop variable the default. The planner's "mirror role_test.go exactly" inherits the no-shadowing pattern, which is correct under the Tillsyn project's Go version. Test parallelism via `t.Parallel()` is safe without the shadow. Verified by reading `role_test.go:33-40`.

### N6 — Required-on-create vs optional-on-update divergence (Required Attack Vector 4) — REFUTED with caveat

The asymmetry mirrors `Role`'s precedent (validation-on-create at the domain layer, preserve-on-empty at the service-layer update path). No semantic gap I can construct breaks this:
- Create path: `domain.NewActionItem` validates non-empty + valid-enum.
- Update path: empty input preserves prior; non-empty input validates.
- "Clearing" structural_type is impossible — closed-enum required field has no valid empty terminal state.
- MCP boundary stub at 3.A.4 routes empty-on-create to rejection.

Caveat: 3.A.4 says nothing about whether `update` operations from MCP that omit `structural_type` (sending `""`) could be exploited to ship a row whose `structural_type` got corrupted to empty via a different path (snapshot toDomain, SQLite scan-of-pre-3.A.2-row). The planner does address this — 3.A.5 explicitly says `(SnapshotActionItem).toDomain` does NOT default-fill empty (catches it on next mutation), and 3.A.3 says `scanActionItem` does NOT default-fill empty. So the only paths that can produce an empty-`structural_type` `ActionItem` post-3.A.5 are scan-of-legacy-row and deserialize-of-legacy-snapshot, both of which fail-closed on the next write. Defensible.

### N7 — Same-package blockers (Required Attack Vector 2) — REFUTED for intra-Unit-A; partially confirmed cross-unit (already covered in C3 + C5)

Intra-Unit-A:
- 3.A.1 → 3.A.2 (same `internal/domain` package) — explicit `blocked_by` ✓
- 3.A.2 → 3.A.3 (different packages) — explicit `blocked_by` ✓ (semantic dep)
- 3.A.2 → 3.A.4 (different packages) — explicit `blocked_by` ✓
- 3.A.2 → 3.A.5 (different packages, but app pkg) — explicit `blocked_by` ✓
- 3.A.3 ↔ 3.A.4: 3.A.3 edits `internal/adapters/storage/sqlite`; 3.A.4 edits `internal/app, internal/adapters/server/common, internal/adapters/server/mcpapi` — disjoint package locks. No `blocked_by` needed at the package level. SEMANTIC dep: 3.A.4's app-service plumbing technically only needs the domain field (3.A.2), not the SQLite column (3.A.3); 3.A.3 and 3.A.4 could parallelize. The planner serializes them via `Blocked by: 3.A.2` only on each, which is correct.
- 3.A.5 (`internal/app/snapshot.go`) ↔ 3.A.4 (`internal/app/service.go`): SAME `internal/app` package. Both modify `.go` files in `internal/app`. Different files, but same package — Go package-level test compilation. 3.A.5 declares `Blocked by: 3.A.2` only. 3.A.4 also declares `Blocked by: 3.A.2`. **Risk**: parallel build of 3.A.4 + 3.A.5 against the same `internal/app` package could race on test compilation if both touch test files (3.A.4 doesn't directly, but 3.A.5's `snapshot_test.go` IS in `internal/app`).

Looking again: 3.A.4 paths list `internal/app/service.go` (production only, no test). 3.A.5 paths list `internal/app/snapshot.go` and `internal/app/snapshot_test.go`. Both packages compile together. If 3.A.4 lands first and breaks `internal/app`'s compilation (because `service.go` references `domain.StructuralType` which lands in 3.A.2 — already cleared), then 3.A.5's builder picks up a green package. Reverse order also works. So serialization is unnecessary but parallelization is also safe given both depend on 3.A.2.

Same-package guideline ("sibling tasks sharing a Go package MUST have explicit blocked_by") is, strictly read, a guarantee not currently met for 3.A.4 ↔ 3.A.5. **Defensible only because both depend on 3.A.2 and neither writes to the other's file.** The planner should add a one-line note: *"3.A.4 and 3.A.5 share `internal/app` package but write to disjoint files — parallel-safe; orchestrator may run them in either order or in parallel."* Cosmetic, not blocking.

## Verdict Summary

**Verdict: fail.**

5 CONFIRMED counterexamples (C1–C5) and 2 nits (C6, C7).

**Most damaging:** C1 — required-on-create at the domain layer breaks ~96 `NewActionItem` call-sites across 6 packages, but 3.A.2's `Paths` and `Acceptance` gate only on `internal/domain`. The droplet would ship "complete" (passes its own `mage test-pkg ./internal/domain`) while leaving the rest of the build broken for downstream droplets. The fix is one of three options listed in C1, and 3.A.2's spec must explicitly choose one before the builder fires.

**Second-most damaging:** C3 — three-way edit conflict on `go-qa-falsification-agent.md` only flagged as two-way. Unit D 5.D.5 sweep needs an explicit `blocked_by: 3.A.7` edge.

**Recommended path forward:**

1. Add C1 mitigation to 3.A.2 acceptance text: enumerate every test file calling `NewActionItem` across all 6 packages OR introduce a `newActionItemForTest`-style wrapper OR upgrade the gate to `mage ci`.
2. Add C5 mitigation to 3.A.4 acceptance text: enumerate the existing `extended_tools_test.go` Create-via-stub fixtures that need `StructuralType` supplied, or default empty-in-stub to `StructuralTypeDroplet`.
3. Add a second cross-unit conflict bullet to Unit A's `## Notes`: *"Unit D 5.D.5 sweep on `~/.claude/agents/go-qa-falsification-agent.md` must also `blocked_by 3.A.7`."*
4. Tighten or comment-correct C4 regex rationale.
5. Reclassify C6 nit (move "Confirmed" item out of "Unresolved Questions").
6. Optional: address C7 by re-running planner pass with Hylla-first evidence sourcing per CLAUDE.md.
