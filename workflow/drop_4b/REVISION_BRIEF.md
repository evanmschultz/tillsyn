# Drop 4b — Gate Execution + Auth Auto-Revoke + Git-Status Pre-Check (Revision Brief)

**Status:** revision-brief authoring (post-Option-β scope split, 2026-05-04). Parallel-planner dispatch lands after dev approves this brief.
**Author:** orchestrator (post-Drop-4a-merge).
**Combined with Drop 4a + Drop 4c = MVP-feature-complete cascade.**

## 1. Hard Prerequisites

Drop 4a is on `main` (commit `618c7d2` — pre-push hook downgrade; CI green). Drop 4b consumes:

- `internal/app/dispatcher/` package + `Dispatcher` interface + `RunOnce` (4a.14, 4a.23).
- `paths` / `packages` / `start_commit` / `end_commit` first-class on `ActionItem` (4a.5, 4a.6, 4a.8, 4a.9).
- Project-node first-class fields (4a.12).
- Cleanup hook with auth-revoke stub (4a.22).
- Always-on parent-blocks-on-failed-child (4a.11).
- Orch-self-approval gate + audit trail (4a.24, 4a.26).
- LiveWaitBroker + ActionItemChanged event (4a.15).
- Tree walker + auto-promotion (4a.18).

## 2. Goal

Wire deterministic post-build gate execution + auto-promotion-on-state-change + auth auto-revoke + git-status pre-check. NO LLM in any 4b code path (commit-agent invocation deferred to Drop 4c). End state: dispatcher promotes `todo → in_progress` automatically on `LiveWaitEventActionItemChanged`; builder reports success; gate runner fires `mage_ci` (and any other configured gates); auth revokes on terminal state; git-status pre-check rejects creation on dirty paths.

## 3. Locked Architectural Decisions (L1–L7)

- **L1 — Gates are deterministic; no LLM in any 4b gate path.** Drop 4c lands the commit-agent invocation. Drop 4b ships gate IMPLEMENTATIONS that are spawn-pipeline-independent only: `mage_ci`, `mage_test_pkg`, `hylla_reingest`. The gate FRAMEWORK accepts arbitrary named gate kinds; Drop 4c registers commit + push gates atop the same framework.
- **L2 — `[gates.<kind>]` table extends template TOML schema.** Closed enum of gate kinds in 4b: `mage_ci`, `mage_test_pkg`, `hylla_reingest`. Drop 4c adds `commit`, `push`. Templates reference gates by name; gate framework dispatches via internal registry.
- **L3 — Auth auto-revoke on terminal state lands here.** Replaces the 4a.22 cleanup-hook stub with real `Service.RevokeSessionForActionItem(actionItemID)`. Fires on `complete` / `failed` / `archived`. Lease + session both revoked.
- **L4 — Git-status pre-check on action-item creation.** `Service.CreateActionItem` rejects when any declared `paths` entry is dirty in `git status --porcelain <path>`. Always-on; bypass requires the post-MVP supersede CLI (deferred). Wave 1 of Drop 4a's `paths` field is the input; the check is a domain-level guard.
- **L5 — Auto-promotion-on-state-change is the cascade's continuous-mode loop.** Dispatcher subscribes to `LiveWaitEventActionItemChanged` (4a.15), walks tree on every event, promotes eligible `todo` items via existing 4a.18 walker. Replaces 4a's manual-trigger CLI as the default cascade path; the manual CLI stays as a dev escape hatch. **Spawn invocation still uses 4a.19's stub** — Drop 4c F.7 replaces the spawn pipeline; 4b's auto-promotion just calls `RunOnce` programmatically on every walked-eligible item.
- **L6 — Default template `[gates.build] = ["mage_ci"]` only in 4b.** Drop 4c expands to `["mage_ci", "commit", "push"]` once those gates land. Project templates can override at 4b-time to add `mage_test_pkg` or `hylla_reingest` if their build flow needs it.
- **L7 — Hylla reingest gate runs at `closeout` action-item completion, not per-`build`.** Memory rule `feedback_orchestrator_runs_ingest.md` survives unchanged: full enrichment, from remote, post-CI-green. Pre-MVP fallback: if Hylla MCP not connected, gate logs warning + skips with worklog note (does NOT fail closeout).

## 4. Wave Structure

~7 droplets across 3 waves. Names align with the post-β scope.

### Wave A — Gate Runner Mechanism (~4 droplets)

- **4b.1 — `[gates]` table schema + closed-enum gate-kind primitive in `internal/templates/schema.go`.** Gate kinds: `mage_ci`, `mage_test_pkg`, `hylla_reingest`. Per-kind `[gates.<kind>]` array of gate names. Validation rejects unknown gate names at template-load time.
- **4b.2 — Gate runner `internal/app/dispatcher/gates.go`.** Reads template `[gates.<kind>]`, executes each gate in order via internal registry (`map[string]gateFunc`); halt on first failure; record failed gate name + last-100-lines output in `metadata.BlockedReason`. Successful gates record nothing (only failures are noisy).
- **4b.3 — `mage_ci` gate implementation.** Wraps `exec.Command("mage", "ci")` with `Dir = project.RepoPrimaryWorktree`. Captures exit code; on non-zero, last 100 lines of stdout/stderr land in `metadata.BlockedReason`. Concurrent-safe (gate runner serializes per-action-item).
- **4b.4 — `mage_test_pkg` gate implementation.** Derives package list from action-item `packages`; runs `mage test-pkg <pkg>` per package. Failure path mirrors `mage_ci`. Optimization for sub-package work — kind-bound (e.g., `[gates.build-qa-proof] = ["mage_test_pkg"]` runs only the package the QA pass owns).

### Wave B — DEFERRED to Drop 4c (commit + push)

Per Option β. Drop 4c F.7.12-F.7.16 absorb. Drop 4b ships nothing in Wave B.

### Wave C — Auth Auto-Revoke + Git-Status Pre-Check + Auto-Promotion + Hylla Reingest Gate (~3 droplets)

- **4b.5 — Auth auto-revoke wiring.** Cleanup hook (4a.22) replaces `revokeAuthBundleStub` with real `Service.RevokeSessionForActionItem(actionItemID)` (new method or repurpose existing `RevokeAuthSession` if signature aligns). Fires on `complete` / `failed` / `archived`. Tests verify session + lease both revoked; revoke errors don't block lock release (errors aggregated via `errors.Join`).
- **4b.6 — Git-status pre-check on `Service.CreateActionItem`.** Domain-level: when input has non-empty `Paths`, run `git status --porcelain <path>` per path (or batched via single `git status --porcelain --pathspec-from-file`). Reject creation if any declared path is dirty; error includes the dirty path list. Always-on. **Pre-MVP rule: dev fresh-DBs after this lands** (existing in_progress action items may have been created with dirty paths and would now retroactively fail validation if updated; not strictly needed for fresh-DBs).
- **4b.7 — Auto-promotion-on-state-change subscriber + `hylla_reingest` gate stub.** Dispatcher's continuous-mode loop: subscribes to `LiveWaitEventActionItemChanged` (4a.15), walks tree on every event via 4a.18 walker, promotes eligible `todo` items via `RunOnce`. Spawn invocation uses existing 4a.19 stub (Drop 4c replaces). Plus: `hylla_reingest` gate stub — programmatic `hylla_ingest` call wrapper; pre-MVP fallback when Hylla MCP not connected (logs warning, skips, doesn't fail closeout). Combined into one droplet because surface is small + concerns share the dispatcher subscriber lifecycle.

## 5. Pre-MVP Rules In Force

- **No migration logic in Go.** Schema-changing droplets (4b.1 adds `[gates]` table; 4b.7 may add fields to `ProjectMetadata` for the dispatcher-mode toggle if continuous-mode is opt-in) note "Dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`."
- **No closeout MD rollups** (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK) — pre-dogfood. Each droplet writes a per-droplet worklog only.
- **Opus builders.** Every builder spawn carries `model: opus`.
- **Filesystem-MD mode.** No Tillsyn-runtime per-droplet plan items.
- **Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING** in every subagent response.
- **Single-line conventional commits.** ≤72 chars.
- **NEVER raw `go test` / `go build` / `go vet` / `mage install`.** Always `mage <target>`.
- **Hylla is Go-only today.** Markdown sweeps fall back to `Read` / `rg` without logging Hylla misses.

## 6. Concrete Planner Spawn Contract (For Wave Planners)

Two parallel planner spawns: one for Wave A, one for Wave C. Wave B is deferred (no planner needed).

**Each planner:**
- Reads `workflow/drop_4b/REVISION_BRIEF.md` (this file) + `workflow/drop_4b/SKETCH.md` + relevant Drop 4a artifacts.
- Reads code surfaces for evidence: `internal/app/dispatcher/dispatcher.go`, `internal/app/dispatcher/cleanup.go`, `internal/templates/schema.go`, `internal/app/service.go`.
- Authors `workflow/drop_4b/WAVE_A_PLAN.md` (or `WAVE_C_PLAN.md`) with per-droplet acceptance criteria, test scenarios, falsification mitigations, verification gates.
- NO Hylla calls (Hylla is stale across uncommitted/post-merge code; planners use `Read` / `Grep` / `LSP` directly).
- Section 0 SEMI-FORMAL REASONING required. Tillsyn-flow output style.

## 7. Open Questions for Plan-QA Review

- **Q1 — Gate kind closed-enum vs extensible.** Sketch's L2 says closed enum. Should Drop 4b's schema reject unknown gate names at template-load (closed) or accept-and-warn (extensible for future drops)? Recommendation: closed; Drop 4c adds `commit` + `push` to the enum.
- **Q2 — `mage_test_pkg` vs `mage_ci` overlap.** A template that lists both runs the test suite twice (once via `mage ci`, once via `mage test-pkg`). Should the gate runner deduplicate? Recommendation: no — it's the template author's responsibility. Document as "no implicit deduplication."
- **Q3 — Continuous-mode dispatcher singleton.** Drop 4a's CLI bootstraps per-invocation. Drop 4b's continuous mode wants one daemon. Existing `till serve` (the MCP daemon) is the natural home, OR new `till dispatcher serve`. Recommendation: fold into existing `till serve` — fewer processes for the dev to manage.
- **Q4 — Hylla reingest gate when Hylla MCP not connected.** Pre-MVP, the orchestrator session may not have Hylla tools. Sketch L7 says skip with warning. Confirm.
- **Q5 — Auth auto-revoke on `archived`.** `archived` is a lifecycle move, not success/failure. Sketch says revoke (any terminal-shaped transition revokes). Confirm.
- **Q6 — Git-status pre-check granularity.** Per-path invocation (slow, simple) vs batched (fast, more complex parsing). Recommendation: per-path; the path count per droplet is typically <10.
- **Q7 — Gate output capture size.** 100 lines? Last 8KB? Bounded by what? Recommendation: last 100 lines OR last 8KB whichever shorter, captured into `metadata.BlockedReason` directly. Storage shape simple; fits in existing string field.

## 8. Out Of Scope

- Commit-agent integration / `commit` gate / `push` gate (deferred to Drop 4c F.7.12-F.7.16).
- Project-metadata toggles `dispatcher_commit_enabled` / `dispatcher_push_enabled` (deferred — they exist to gate the deferred items).
- Dispatcher daemon mode for continuous operation as a separate process — Drop 4b folds into existing `till serve`.
- Spawn pipeline redesign (Drop 4c F.7).
- Template ergonomics (Drop 4c Theme F.1-F.6).
- Marketplace (Drop 4c Theme G post-MVP).
- Drop 4.5 TUI overhaul.

## 9. Approximate Size

~7 droplets total. ~3 days of build work given Drop 4a's pace. Plan-QA twins after PLAN.md synthesis. Builders run opus.
