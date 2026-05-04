# Drop 4b — Gate Execution + Post-Build Pipeline (Sketch)

**Status:** placeholder — NOT a full plan. Full PLAN.md authoring + parallel-planner dispatch + plan-QA twins land post-Drop-4a-merge before any builder fires.
**Author date:** 2026-05-03 (during Drop 4a planning phase).
**Purpose:** capture scope so nothing gets forgotten when full planning starts.

## Naming

**Drop 4b** = the second half of the original PLAN.md §19.4 Drop 4. Drop 4a delivers the dispatcher mechanism (manual-trigger); Drop 4b delivers the gate execution + post-build pipeline that turns the manual-trigger dispatcher into a self-driving cascade.

Combined Drop 4a + 4b = MVP-feature-complete cascade.

## Hard Prerequisites (from Drop 4a)

Drop 4b cannot start until Drop 4a's PR merges and is on `main`. Specifically, 4b consumes:

- **`internal/app/dispatcher/` package** — exists with `Dispatcher` interface + `RunOnce` (4a.14, 4a.23).
- **`start_commit` / `end_commit` first-class on `ActionItem`** (4a.8, 4a.9) — commit-agent diff context.
- **`paths` / `packages` first-class on `ActionItem`** (4a.5, 4a.6) — gate runner reads them for the package-scoped `mage test-pkg <pkg>` calls.
- **Project-node first-class fields on `Project`** (4a.12) — gate runner reads `BuildTool` to pick `mage` vs `npm` invocations; reads `RepoPrimaryWorktree` for `cd` target.
- **Auth-bundle stub seam in `internal/app/dispatcher/spawn.go`** (4a.19) — Drop 4b's auth-bundle materialization fills the stub.
- **Cleanup hook with auth-revoke stub** (4a.22) — Drop 4b lands the real revoke.
- **Always-on parent-blocks-on-failed-child** (4a.11) — gate failure paths rely on the unconditional invariant (a `failed` build blocks parent close until supersede).
- **Orch-self-approval auth flow** (4a.24–4a.28) — gate runner uses dispatcher-spawned commit-agent which needs the new auth path.

## Goal

Wire the deterministic post-build pipeline that fires after a builder action item reports success. No LLM in the gate path except the commit-agent. Turn manual-trigger `till dispatcher run` into the auto-promotion-on-state-change loop the cascade plan envisions.

End-state: dispatcher promotes `todo → in_progress`, builder agent edits, builder reports `complete`, post-build gates fire (`mage ci` → commit-agent forms message → `git commit` → optionally `git push`), QA twins promote (parent done), QA agents fire in parallel, repeat across the tree until `closeout` triggers Hylla reingest.

## Locked Architectural Decisions

These ride forward from Drop 4a's REVISION_BRIEF and confirm the 4b shape:

- **L1 (4b)** — Gates are deterministic. No LLM in the gate path except the commit-agent (haiku) which forms a single-line conventional-commit message. Gate failures route to the action item's `metadata.outcome = "failure"` + `metadata.failure_reason`.
- **L2 (4b)** — Gates read template `[gates]` table per kind. Drop 3.14 already shipped the agent-bindings + child-rules schema in `internal/templates/builtin/default.toml`; the `[gates]` table extends that schema with kind-bound gate sequences (e.g., `[gates.build] = ["mage_ci", "commit", "push"]`).
- **L3 (4b)** — `git commit` + `git push` are dispatcher-driven post-MVP-confidence-window. Default for 4b is **opt-in** per project (`metadata.dispatcher_commit_enabled bool` on Project). Dev keeps git management manual until dogfood (Drop 5) builds confidence.
- **L4 (4b)** — Hylla reingest fires at `closeout` action-item completion, NOT per-`build`. Existing memory `feedback_orchestrator_runs_ingest.md` rule survives unchanged ("Once per drop during DROP N END — LEDGER UPDATE, full enrichment only, from remote, after push + gh run watch green").
- **L5 (4b)** — Auth auto-revoke on terminal state lands here (deferred from Drop 4a Wave 3 per REVISION_BRIEF "out of scope"). Cleanup hook (4a.22) calls real `revokeAuthBundle` instead of the stub.
- **L6 (4b)** — Git-status-pre-check on action-item creation lands here (deferred from Drop 4a Wave 1's L2 / Drop 1 audit). Action-item creation rejects when declared `paths` are dirty in `git status --porcelain`.
- **L7 (4b)** — Auto-promotion-on-state-change is the cascade's continuous-mode loop. Dispatcher subscribes to `LiveWaitEventActionItemChanged` (4a.15), walks tree on every event, promotes eligible `todo` items, spawns builders. Replaces 4a's manual-trigger CLI as the default cascade path; the manual CLI stays as a dev escape hatch.

## Tentative Wave Structure

~11 droplets across 3 waves. Subject to revision at full-planning time.

### Wave A — Gate runner mechanism (~4 droplets)

- **`[gates]` table schema in `internal/templates/schema.go`**: per-kind gate sequence (`[gates.build]`, `[gates.build-qa-proof]`, etc.). Closed enum of gate kinds: `mage_ci`, `mage_test_pkg`, `commit`, `push`, `hylla_reingest`. Each gate carries optional config (e.g., `mage_test_pkg` reads `paths` to derive package list).
- **Gate runner `internal/app/dispatcher/gates.go`**: reads template `[gates.<kind>]`, executes each gate in order, posts result. Halt on first failure; record failed gate name in `metadata.failure_reason`.
- **`mage_ci` gate**: shells out via `exec.Cmd` (or wraps existing `mage` binary call); captures exit code + last 100 lines of stdout/stderr for `metadata.gate_output`.
- **`mage_test_pkg` gate**: derives package list from action-item `packages`, runs `mage test-pkg <pkg>` per package. Optimization for sub-package work.

### Wave B — Commit + push pipeline (~4 droplets)

- **Commit-agent (haiku) integration `internal/app/dispatcher/commit_agent.go`**: spawns `claude --agent commit-message-agent --model haiku` against `git diff <start_commit>..<end_commit>` (using 4a's first-class commit fields). Returns single-line conventional commit message.
- **`commit` gate**: runs `git add <paths>` (action-item-declared paths only, NOT `git add -A`); runs `git commit -m "<message>"`; sets action-item `end_commit = git rev-parse HEAD`. Honors project `metadata.dispatcher_commit_enabled` toggle (default false in 4b; flip to true in dogfood).
- **`push` gate**: runs `git push origin <branch>`; gates on toggle. On push failure: action item moves to `failed` with `metadata.failure_reason = "git push: <error>"`.
- **Project-metadata toggles** for both: `dispatcher_commit_enabled bool`, `dispatcher_push_enabled bool`. Default false in 4b. Dev opts in per project.

### Wave C — Hylla reingest + auth auto-revoke + git-status-pre-check + closeout (~3 droplets)

- **`hylla_reingest` gate** at `closeout` completion: programmatic call to Hylla MCP `hylla_ingest` with project's `hylla_artifact_ref` + `enrichment_mode=full_enrichment`. Per memory rule, source from remote, not local. Ignored in pre-MVP if Hylla MCP not connected (matches current state).
- **Auth auto-revoke** in 4a.22's cleanup hook: replace stub with real `Service.RevokeSessionForActionItem(actionItemID)`. Fires on `complete` / `failed` / `archived`. Lease + session both revoked.
- **Git-status-pre-check** on action-item creation: `Service.CreateActionItem` rejects when any declared `paths` entry is dirty in `git status --porcelain <path>`. Always-on; bypass requires the post-MVP supersede CLI.
- **Auto-promotion-on-state-change**: dispatcher's continuous-mode subscriber loop. On `LiveWaitEventActionItemChanged` event, walk tree + promote. May fold into one of the above droplets if surface is small; could be a standalone droplet if testing demands.

## Pre-MVP Rules (carried forward)

- No migration logic in Go; dev fresh-DBs between schema-touching landings.
- No closeout MD rollups; per-drop worklog only.
- Opus builders.
- Filesystem-MD mode.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING.
- Single-line conventional commits.
- NEVER raw `go test` / `go build` / `mage install`.
- Hylla is Go-only today.

## Open Questions To Resolve At Full-Planning Time

These are sketch-time unknowns; full planning + plan-QA-falsification surfaces resolutions.

- **Q1 — Project toggle defaults.** `dispatcher_commit_enabled` and `dispatcher_push_enabled` default-false in 4b sketch. Should dogfood (Drop 5) flip these to default-true, or stay opt-in even in dogfood?
- **Q2 — `hylla_reingest` gate when Hylla MCP isn't connected.** Pre-MVP, the current orchestrator session doesn't have Hylla MCP tools. Should the gate skip silently with a worklog note, fail the closeout, or surface as an attention item?
- **Q3 — Commit-agent prompt contract.** What does the spawn prompt feed the haiku model? `git diff` blob plus action-item title + acceptance summary? Need a concrete spec at planning time.
- **Q4 — Push failure recovery.** A `push` gate failure leaves the local commit AHEAD of origin. Auto-retry, surface to dev, or rollback the commit? Initial sketch: surface to dev via attention item, no auto-rollback.
- **Q5 — Auth auto-revoke on `archived`.** `archived` is technically not a failure or success — it's a lifecycle move. Should auth still revoke? Sketch says yes (any terminal-shaped transition revokes).
- **Q6 — Git-status-pre-check granularity.** `git status --porcelain <path>` is path-by-path. For a droplet declaring 6 paths, is that 6 git invocations (slow) or one batched call (faster but path-set-bounded)?
- **Q7 — Continuous-mode dispatcher singleton.** Drop 4a's CLI bootstraps per-invocation. Drop 4b's continuous mode wants one daemon. Is that the existing `till serve` or a new `till dispatcher serve`? Naming + lifecycle decision deferred.
- **Q8 — Gate output capture size.** Bounded by what? 100 lines? Last 8KB? Full output to a side-table? Storage shape decision deferred.

## Approximate Size

~11 droplets. Smaller than 4a (32). Most droplets are surgical wiring on top of 4a's primitives — gate runner is the largest new component, ~5-6 source files. Full planning at post-4a-merge time will refine the count.

## Workflow Cross-References

- `workflow/drop_4a/PLAN.md` — Drop 4a's plan (this drop's hard prereqs).
- `workflow/drop_4a/REVISION_BRIEF.md` — locked decisions L1–L8 that ride forward into 4b's L1–L7.
- PLAN.md §19.4 — original Drop 4 spec; 4a + 4b together implement it.
- PLAN.md §9.7 — Hylla reingest drop-end-only rule.
- Memory `feedback_orchestrator_runs_ingest.md` — full-enrichment-from-remote rule.

## Open Tasks Before Full Planning

1. Drop 4a closes (PR merged on `main`).
2. Drop 4a's actual landings reviewed against this sketch — anything that drifted (e.g., auth-bundle stub turned out clunky, walker eligibility rules differ from sketch) gets reflected in 4b's REVISION_BRIEF.
3. `project_drop_4a_refinements_raised.md` reviewed for findings that should land in 4b vs Drop 4c (pre-Drop-5 refinement).
4. Full REVISION_BRIEF authored, parallel-planner dispatch (likely 3 wave planners), unified PLAN.md synthesis, plan-QA twins → green → builder dispatch.
