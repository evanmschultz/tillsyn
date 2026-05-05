# Drop 4b — Gate Execution + Auth Auto-Revoke + Git-Status Pre-Check: Unified Plan

**Working name:** Drop 4b — Gate Execution
**Sequencing:** post-Drop-4a-merge, pre-Drop-4c
**Total droplets:** 7 across 2 waves (Wave B deferred to Drop 4c per Option β)
**Mode:** filesystem-MD only (no per-droplet Tillsyn plan items today)
**Plan-QA gate:** plan-QA-proof + plan-QA-falsification fire AGAINST this unified plan before any builder spawns

---

## 1. Goal

Wire deterministic post-build gate execution + auto-promotion-on-state-change + auth auto-revoke + git-status pre-check on top of Drop 4a's manual-trigger dispatcher. NO LLM in any 4b code path — commit-agent invocation deferred to Drop 4c F.7. End state: dispatcher promotes `todo → in_progress` automatically on `LiveWaitEventActionItemChanged`; builder reports success; gate runner fires configured gates (`mage_ci`, `mage_test_pkg`, `hylla_reingest`); auth revokes on terminal state; git-status pre-check rejects creation on dirty paths.

Combined Drop 4a + Drop 4b + Drop 4c = MVP-feature-complete cascade.

---

## 2. Locked Architectural Decisions (L1–L7)

Locked at REVISION_BRIEF authoring time:

- **L1** — Gates are deterministic; no LLM in any 4b gate path. Drop 4b ships gate IMPLEMENTATIONS that are spawn-pipeline-independent only. Drop 4c F.7 lands commit-agent invocation + commit/push gates.
- **L2** — `[gates.<kind>]` table extends template TOML schema. Closed enum of gate kinds in 4b: `mage_ci`, `mage_test_pkg`, `hylla_reingest`. Drop 4c adds `commit`, `push`. Templates reference gates by name.
- **L3** — Auth auto-revoke on terminal state lands here. Replaces 4a.22 cleanup-hook stub with real `Service.RevokeSessionForActionItem`. Fires on `complete` / `failed` / `archived`.
- **L4** — Git-status pre-check on `Service.CreateActionItem`. Rejects when any declared `Paths` entry is dirty in `git status --porcelain <path>`. Always-on; bypass requires post-MVP supersede CLI.
- **L5** — Auto-promotion-on-state-change is the cascade's continuous-mode loop. Subscribes to `LiveWaitEventActionItemChanged` (4a.15), walks tree via 4a.18 walker, promotes via existing 4a.23's `RunOnce`. Folds into existing `till serve` daemon.
- **L6** — Default template `[gates.build] = ["mage_ci"]` in 4b. Drop 4c expands to `["mage_ci", "commit", "push"]`.
- **L7** — Hylla reingest gate runs at `closeout` action-item completion, not per-`build`. Pre-MVP fallback: if Hylla MCP not connected, gate logs warning + skips with worklog note.

---

## 3. Pre-MVP Rules In Force

- No migration logic in Go. Schema-changing droplets note "Dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`."
- No closeout MD rollups. Per-droplet worklogs only.
- Opus builders. Every builder spawn carries `model: opus`.
- Filesystem-MD mode. No Tillsyn-runtime per-droplet plan items.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING in every subagent response.
- Single-line conventional commits. ≤72 chars.
- NEVER raw `go test` / `go build` / `go vet` / `mage install`. Always `mage <target>`.
- Hylla is Go-only today. Markdown sweeps fall back to `Read` / `rg` without logging Hylla misses.

---

## 4. Wave Structure

| Wave | Theme                                              | Droplet IDs       | Count | Sequence |
| ---- | -------------------------------------------------- | ----------------- | ----- | -------- |
| A    | Gate runner mechanism                              | 4b.1 – 4b.4       | 4     | First    |
| B    | DEFERRED TO DROP 4C (commit-agent + commit + push) | —                 | 0     | (4c F.7) |
| C    | Auth auto-revoke + git-status + auto-promotion     | 4b.5 – 4b.7       | 3     | After A  |

Total: **7 droplets**.

---

## 5. Wave-Internal-Plan Cross-References

Each wave's full per-droplet acceptance criteria, test scenarios, falsification mitigations, and verification gates live in the per-wave plan files:

- `workflow/drop_4b/WAVE_A_PLAN.md` — gate runner mechanism (4 droplets, ~1200 LOC)
- `workflow/drop_4b/WAVE_C_PLAN.md` — auth auto-revoke + git-status + auto-promotion (3 droplets, ~490 LOC)

Builders spawn against the unified PLAN's droplet row PLUS the wave plan's full detail.

---

## 6. Wave-to-Global ID Mapping

| Wave-internal | Global   | Title                                                                   |
| ------------- | -------- | ----------------------------------------------------------------------- |
| WA.1          | **4b.1** | `[GATES]` TABLE SCHEMA + CLOSED-ENUM GATE-KIND PRIMITIVE                |
| WA.2          | **4b.2** | GATE RUNNER + REGISTRY                                                  |
| WA.3          | **4b.3** | `mage_ci` GATE IMPLEMENTATION                                           |
| WA.4          | **4b.4** | `mage_test_pkg` GATE IMPLEMENTATION                                     |
| WC.1          | **4b.5** | AUTH AUTO-REVOKE WIRING (REPLACE 4A.22 STUB)                            |
| WC.2          | **4b.6** | GIT-STATUS PRE-CHECK ON `Service.CreateActionItem`                      |
| WC.3          | **4b.7** | AUTO-PROMOTION SUBSCRIBER + `hylla_reingest` GATE STUB                  |

---

## 7. Per-Droplet Rows

Wave-plan cross-references give the full acceptance detail. Rows below are the global view: title, paths summary, `blocked_by` with global IDs, one-line notes.

### Wave A — Gate Runner Mechanism (4 droplets)

#### 4b.1 — `[GATES]` TABLE SCHEMA + CLOSED-ENUM GATE-KIND PRIMITIVE

- **Paths:** `internal/templates/schema.go`, `internal/templates/load.go`, `internal/templates/builtin/default.toml`, schema test files.
- **Packages:** `internal/templates`.
- **Acceptance:** Add `GateKind` closed enum (`mage_ci`, `mage_test_pkg`, `hylla_reingest`); `Template.Gates map[domain.Kind][]GateKind` field with TOML key `gates` (NOT `gate_rules` — that key is reserved per Drop 3); new `validateGateKinds` validator hooked into `Load`; default.toml ships `[gates.build] = ["mage_ci"]`. See WAVE_A_PLAN.md §4b.1.
- **Blocked by:** Drop 4a merge.
- **Notes:** Wave A anchor. Per-droplet pre-MVP fresh-DB note (kind-catalog envelope expands).

#### 4b.2 — GATE RUNNER + REGISTRY

- **Paths:** `internal/app/dispatcher/gates.go` (NEW), `internal/app/dispatcher/gates_test.go` (NEW).
- **Packages:** `internal/app/dispatcher`.
- **Acceptance:** `gateRunner` struct with `Register(name, gateFunc)` + `Run(ctx, item, project, template)` returning `[]GateResult`. Halt on first failure; record failed gate + last-100-lines-or-8KB-shorter output in `GateResult.Output`. Runner does NOT mutate action item — caller (subscriber, Drop 4c gate-failure routing) handles state transitions. See WAVE_A_PLAN.md §4b.2.
- **Blocked by:** **4b.1** (consumes `Template.Gates` map + `GateKind` enum).
- **Notes:** Linear chain anchor; subsequent gate impls register against this runner.

#### 4b.3 — `mage_ci` GATE IMPLEMENTATION

- **Paths:** `internal/app/dispatcher/gate_mage_ci.go` (NEW), `gate_mage_ci_test.go` (NEW).
- **Packages:** `internal/app/dispatcher`.
- **Acceptance:** `gateMageCI(ctx, item, project) GateResult` wraps `exec.CommandContext("mage", "ci")` with `cmd.Dir = project.RepoPrimaryWorktree`; package-private `commandRunner` test seam (interface for fake injection); empty-worktree guard mirrors `dispatcher.go:392`; output capture last-100-lines-or-8KB-shorter. See WAVE_A_PLAN.md §4b.3.
- **Blocked by:** **4b.2** (registers against runner; same-package compile lock).
- **Notes:** `commandRunner` indirection lives here; 4b.4 reuses.

#### 4b.4 — `mage_test_pkg` GATE IMPLEMENTATION

- **Paths:** `internal/app/dispatcher/gate_mage_test_pkg.go` (NEW), `gate_mage_test_pkg_test.go` (NEW).
- **Packages:** `internal/app/dispatcher`.
- **Acceptance:** `gateMageTestPkg(ctx, item, project) GateResult` iterates over `item.Packages`; runs `mage test-pkg <pkg>` per package via `commandRunner`; halts on first failure; output capture aggregated across packages. Empty `item.Packages` documented behavior: gate emits info-level log + returns success (silent-success documented as PQA-4 routed open question). See WAVE_A_PLAN.md §4b.4.
- **Blocked by:** **4b.3** (shares `commandRunner` indirection; same-package compile lock).
- **Notes:** Wave A terminal droplet; 4b.7 (Wave C) registers `hylla_reingest` gate next.

### Wave C — Auth Auto-Revoke + Git-Status Pre-Check + Auto-Promotion (3 droplets)

#### 4b.5 — AUTH AUTO-REVOKE WIRING

- **Paths:** `internal/app/auth_requests.go` (extend with `RevokeSessionForActionItem`), `internal/app/dispatcher/cleanup.go` (replace `revokeAuthBundleStub`), test files in both packages.
- **Packages:** `internal/app`, `internal/app/dispatcher`.
- **Acceptance:** Add `Service.RevokeSessionForActionItem(ctx, actionItemID) error` that filters via `AuthSessionFilter` + parses `session.ApprovedPath` for `ScopeID == actionItemID`; revokes session + lease; returns `errors.Join` of failures. `cleanup.go:253-256` `revokeAuthBundleStub` replaced by call into the new method. Existing `errors.Join` aggregation in cleanup hook (line 218-237) survives unchanged. See WAVE_C_PLAN.md §4b.5.
- **Blocked by:** **4b.4** (Wave A close; sequencing per `feedback_md_update_qa.md` self-QA-budget rule even though file-disjoint).
- **Notes:** L3. Tests verify session + lease both revoked; revoke errors don't block lock release.

#### 4b.6 — GIT-STATUS PRE-CHECK ON `Service.CreateActionItem`

- **Paths:** `internal/app/service.go` (extend `CreateActionItem`), `internal/app/service_test.go` or `action_items_test.go`.
- **Packages:** `internal/app`.
- **Acceptance:** When `input.Paths` non-empty, run `git status --porcelain <path>` per path against `project.RepoPrimaryWorktree`; reject creation if any path dirty. Error includes dirty path list. Always-on; bypass requires post-MVP supersede CLI. Per-path invocation (path count typically <10); batched form deferred. **DB action note:** dev fresh-DBs (existing `in_progress` items would retroactively fail validation if updated; not strictly needed for fresh-DBs). See WAVE_C_PLAN.md §4b.6.
- **Blocked by:** **4b.4** (Wave A close; file-disjoint from 4b.5 → can run parallel with 4b.5 after Wave A).
- **Notes:** L4. Domain-level guard before repo write.

#### 4b.7 — AUTO-PROMOTION SUBSCRIBER + `hylla_reingest` GATE STUB

- **Paths:** `internal/app/dispatcher/dispatcher.go` (Start/Stop bodies replace `ErrNotImplemented`), `internal/app/dispatcher/gate_hylla_reingest.go` (NEW), `cmd/till/main.go` (wire dispatcher subscriber into `runServe`).
- **Packages:** `internal/app/dispatcher`, `cmd/till`.
- **Acceptance:** `dispatcher.Start(ctx)` spins subscriber goroutine that calls `subscribeBroker(ctx, projectID)` for each project (4b.7 author recommends Option B per WAVE_C_PLAN.md Q3: `s.repo.ListProjects` at Start time, one goroutine per project — plan-QA falsification arbitrates). On every `LiveWaitEventActionItemChanged`, walk tree via 4a.18 walker, promote eligible items via existing 4a.23's `RunOnce`. `dispatcher.Stop(ctx)` cancels ctx + waits for goroutines. `cmd/till serve` wires Start at startup, Stop on shutdown. Plus: `gateHyllaReingest` stub — programmatic `hylla_ingest` call wrapper; pre-MVP fallback when Hylla MCP not connected (logs warning + returns success; doesn't fail closeout). See WAVE_C_PLAN.md §4b.7.
- **Blocked by:** **4b.5** (shares `dispatcher.go` edits with auth-revoke wiring), **4b.2** (registers `hylla_reingest` against gate runner).
- **Notes:** L5 + L7. Continuous-mode subscriber lives in `till serve` daemon. Spawn invocation uses 4a.19 stub — Drop 4c F.7 replaces; subscriber loop survives unchanged.

---

## 8. DAG Summary

```
Drop 4a merge (committed)
        ↓
       4b.1 (schema)
        ↓
       4b.2 (gate runner)
        ↓
       4b.3 (mage_ci)
        ↓
       4b.4 (mage_test_pkg)
        ↓
   ┌───┴───┐
   ↓       ↓
  4b.5   4b.6     (auth-revoke + git-status pre-check; PARALLEL after Wave A)
   ↓
  4b.7   (auto-promotion + hylla_reingest stub; blocks on 4b.5 + 4b.2)
```

W4 closeout MD sweeps not in scope — no closeout MD rollup pre-dogfood per memory rule.

---

## 9. Approximate Size

~7 droplets total. ~1690 LOC (Wave A ~1200, Wave C ~490). ~3 days build work given Drop 4a's pace. Plan-QA twins after this PLAN.md commits.

---

## 10. Open Questions for Plan-QA Review

Wave A surfaced (per WAVE_A_PLAN.md §PQA-1 to §PQA-4):
- **Q1** — `GateInput.Template` may be unused dead surface (YAGNI).
- **Q2** — `commandRunner` relocation: lives in 4b.3 today; could move to `gates.go` for shared use earlier.
- **Q3** — Output-capture "shorter of 100 lines or 8KB" semantic: which wins on edge cases (long lines, binary output)?
- **Q4** — Empty `Packages` for `mage_test_pkg` is silent-success — security/QA gap?

Wave C surfaced (per WAVE_C_PLAN.md §Q1, §Q3, §Q5):
- **Q5** — Lease-vs-session revocation cascade in 4b.5 (does revoking session auto-revoke lease, or must 4b.5 do both explicitly)?
- **Q6** — Multi-project subscription option in 4b.7 (Option A: single subscriber + project_id filter; Option B: one goroutine per project; Option C: lazy on-demand). Author recommends B; plan-QA arbitrates.
- **Q7** — Gate-registry shape: 4b.2 owns the type; 4b.7's `hylla_reingest` registration validates the contract.
- **Cross-wave Q8** — Drop 4c F.7's spawn pipeline replaces 4a.19's stub. 4b.7's subscriber calls `RunOnce` which uses the stub. Does 4b's auto-promotion need to wait for Drop 4c, or is the stub-driven path acceptable for 4b's MVP-feature-complete scope? Recommendation: stub is acceptable; 4b proves the subscriber + promotion logic; Drop 4c swaps the spawn underneath without disturbing the subscriber.

---

## 11. Out Of Scope

- Commit-agent integration / `commit` gate / `push` gate (deferred to Drop 4c F.7.12-F.7.16).
- Project-metadata toggles `dispatcher_commit_enabled` / `dispatcher_push_enabled` (deferred — they exist to gate the deferred items).
- Spawn pipeline redesign (Drop 4c F.7).
- Template ergonomics (Drop 4c Theme F.1-F.6).
- Marketplace (Drop 4c Theme G post-MVP).
- Drop 4.5 TUI overhaul.
- Closeout MD rollups (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK) — pre-dogfood.

---

## 12. Cross-References

- `workflow/drop_4b/REVISION_BRIEF.md` — locked decisions L1–L7.
- `workflow/drop_4b/SKETCH.md` — original sketch + Option β scope split.
- `workflow/drop_4a/PLAN.md` — Drop 4a's plan (hard prereqs).
- `workflow/drop_4c/SKETCH.md` § Theme F.7 — absorbed Wave B (commit + push) + spawn redesign.
- Memory `feedback_orchestrator_runs_ingest.md` — Hylla reingest contract.
- Memory `project_drop_4c_spawn_architecture.md` — canonical spawn redesign architecture.
