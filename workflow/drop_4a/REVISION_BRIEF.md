# Drop 4a — Dispatcher Core: Master Brief

**Working name:** Drop 4a (Dispatcher Core).
**Sequencing:** post-Drop-3, pre-Drop-4b. Drop 4b (gate execution + post-build pipeline) lands immediately after 4a closes.
**Parent target:** PLAN.md §19.4. Drop 4 was originally one drop; in Drop-3-end discussion the dev approved splitting into 4a (dispatch mechanism) + 4b (gate execution).
**Mode:** filesystem-MD only (no Tillsyn-runtime per-droplet plan items today; this brief + per-wave PLAN fragments + unified PLAN.md drive the work).

## Goal

Replace the orchestrator-as-dispatcher loop with a programmatic dispatcher. Today the parent Claude Code session IS the dispatcher: it picks the kind, picks the agent variant, spawns the subagent, provisions auth, and watches state transitions manually. Drop 4a delivers a manual-trigger dispatcher — the dispatcher takes over agent spawn + lock management + auto-promotion + auth provisioning, but git/commit/push/Hylla-reingest stay manual until Drop 4b.

After Drop 4a + 4b: MVP-feature-complete cascade. Dev creates a project, planners decompose work, dispatcher fires builders + QA on state transitions, post-build gates run automatically, the cascade runs end-to-end except where `human-verify` or `failed` states surface.

## Decomposition — five waves

Per memory `feedback_decomp_small_parallel_plans.md`, decomposition into ≤N parallel planners (≤15min each, one surface/package). Five waves; each wave's droplets serialize internally via `blocked_by`; waves serialize globally (Wave 0 → Wave 1 → Wave 2 → Wave 3 → Wave 4).

### Wave 0 — Dev hygiene infrastructure (~3-4 droplets)

Lands first so subsequent wave builders benefit from local pre-commit gates.

- `mage format-check` target — public wrapper around the existing private `formatCheck()` in `magefile.go:218-236`.
- `.githooks/pre-commit` — runs `mage format-check`.
- `.githooks/pre-push` — runs full `mage ci`.
- `mage install-hooks` target — sets `core.hooksPath = .githooks` so tracked hook scripts become active for fresh clones.
- `mage format` no-arg ergonomics fix — currently `func Format(path string) error` requires a positional arg making `mage format` (no arg) error out. Split into `Format()` (whole tree) and `FormatPath(path string)` (scoped), or adopt a variadic form.

### Wave 1 — Domain-field infrastructure (~8-9 droplets)

All mechanical first-class field additions on `ActionItem` + project node. Same surgical pattern per field: domain struct field + `CreateActionItemInput`/`UpdateActionItemInput` field + SQL column + MCP request/response field + snapshot field + validation tests.

- `paths []string` first-class on `ActionItem` — read by Wave-2 lock manager for file-level locking.
- `packages []string` first-class on `ActionItem` — read by Wave-2 lock manager for package-level locking. Validate that every file in `paths` maps to a package in `packages` (enforce coverage).
- `files []string` first-class on `ActionItem` — reference attachments (read-only, distinct from `paths`). TUI file-viewer pane in Drop 4.5 is the consumer; minimal validation today (`[]string` + path-exists). Pulled forward into 4a per dev's parallelization preference.
- `start_commit string` first-class on `ActionItem` — set on creation (current HEAD). Drop 4b consumer.
- `end_commit string` first-class on `ActionItem` — set on terminal-state transition. Drop 4b consumer.
- `state` accepted in place of `column_id` on `till.action_item(operation=create|move)` — column_id stays in DB but is hidden from agent surface. Adapter resolves column server-side via existing `resolveActionItemColumnIDForState` (`internal/adapters/server/common/app_service_adapter_mcp.go:884`). Reject the call only when both are empty.
- Always-on parent-blocks-on-failed-child — remove `RequireChildrenComplete bool` policy bit from `internal/domain/workitem.go:108`; make the invariant unconditional in the `CompletionCriteriaUnmet` path. Bypass only via the supersede CLI (post-MVP refinement).
- Project-node first-class fields on `Project`: `hylla_artifact_ref` (string), `repo_bare_root` (abs path), `repo_primary_worktree` (abs path), `language` (enum: `go`, `fe`), `build_tool` (string: `mage`, `npm`), `dev_mcp_server_name` (string). Domain-level fields, not metadata blob. Each with explicit validation. Wave-2 dispatcher reads these to spawn agents with correct `cd` + correct `{lang}-builder-agent` variant + correct artifact ref + correct MCP server registration.
- Verify auto-seeded column titles use post-Drop-2 vocabulary (`Todo` / `In Progress` / `Complete` / `Failed` / `Archived`) — not `Done`. Cleanup if drift found.

### Wave 2 — Dispatcher loop (~10 droplets)

The dispatcher itself. New package likely `internal/app/dispatcher/`.

- LiveWaitBroker subscription to action-item state changes. `internal/adapters/livewait/localipc` already exists; connect dispatcher to its event stream.
- Lock manager: file-level (per `paths` entry) + package-level (per `packages` entry). Acquire on `in_progress` transition; release on terminal state. Dynamic conflict detection inserts runtime `blocked_by` rather than letting parallel agents collide.
- Auto-promotion: walk the tree on every state change, find `todo` items whose blockers cleared, promote to `in_progress`. The state-trigger that fires the agent spawn.
- Agent spawn loop: read template `agent_bindings` (Drop 3 landed this), construct invocation, spawn subagent via `claude --agent` CLI or equivalent, hand the subagent its auth bundle.
- Conflict detection: when two siblings share `paths` or `packages` without explicit `blocked_by`, insert a runtime blocker rather than racing.
- State-trigger registration on `in_progress`: when the dispatcher promotes an item, the spawn fires immediately.
- Process monitoring: track spawned subagent PIDs, detect crashes vs clean exits.
- Cleanup on terminal state: release locks, revoke leases (ties to Wave-3 auth flow).
- Manual-trigger interface: dev-facing CLI/MCP command `till dispatcher run --action-item <id>` for manual one-shot dispatch (the "manual-trigger dispatcher" step before Drop 4b automates).

### Wave 3 — Auth integration (Drop 1.6 absorbed) (~5 droplets)

Programmatic auth approval so the dispatcher's auto-spawn loop doesn't bottleneck on the dev TUI.

- Auth-layer rule: orchs may approve non-orch subagent auth requests scoped within their own subtree. Reject orch-self-approval. Reject cross-orch approval.
- STEWARD cross-subtree exception: STEWARD's project-scoped lease covers all six persistent level_1 parents.
- Project opt-out toggle: `metadata.orch_self_approval_enabled: bool` (default `true` once capability lands). Backstop, not the everyday path.
- Audit trail: every orch-approved auth records the approving orch's `agent_instance_id` + `lease_token` + `principal_id`.
- MCP-layer test coverage: 4 cases — (1) orch-in-subtree approves non-orch in same subtree → success; (2) orch-in-subtree approves another orchestrator → reject; (3) orch-A approves orch-B's subagent in B's subtree → reject; (4) STEWARD approves under persistent parent → success.

### Wave 4 — Closeout (~2 droplets)

- CLAUDE.md updates reflecting dispatcher landing + new domain fields + new auth flow + always-on parent-blocks rule.
- Agent prompt updates (`~/.claude/agents/*.md`) — drop the S2 dev-fallback paragraph since orch-self-approval now lands; update `task_id` references where appropriate.
- `~/.claude/CLAUDE.md` parity sync.
- Memory updates: retire `feedback_steward_spawn_drop_orch_flow.md`'s S2 dev-fallback paragraph; update `project_steward_auth_bootstrap.md`.

## Out-of-scope (deferred to Drop 4b or later)

- **Drop 4b scope (lands immediately after 4a):** gate runner reading template `[gates]`, commit-agent (haiku) integration, `git commit` + `git push` automation, Hylla reingest hook on `closeout`, auth auto-revoke on terminal state, git-status-pre-check on action-item creation.
- **Pre-Drop-5 refinement scope:** PATCH semantics on update handlers, reject unknown keys at MCP boundary, supersede CLI, CLI failure listing, server-infer `client_type`, `go.mod` replace cleanup, require non-empty outcome on `failed`.
- **Drop 4.5 scope (concurrent with Drop 5):** TUI overhaul, columns-table retirement (Position rebinding), file-viewer pane consuming `files`.

## Locked architectural decisions (entering planning)

- **L1** — `state` on MCP create+move is the agent-facing API. column_id stays in DB. Columns table retirement deferred to 4.5.
- **L2** — Always-on parent-blocks-on-failed-child. Remove `RequireChildrenComplete` policy bit; make unconditional.
- **L3** — `paths` / `packages` / `files` / `start_commit` / `end_commit` are all first-class domain fields on `ActionItem`, not metadata blob.
- **L4** — Project-node first-class fields (`hylla_artifact_ref`, `repo_bare_root`, `repo_primary_worktree`, `language`, `build_tool`, `dev_mcp_server_name`) on `Project`, not metadata blob.
- **L5** — Drop 1.6 (auth approval cascade) absorbed into Wave 3. The dispatcher's auto-spawn is the only consumer; no reason to ship 1.6 standalone.
- **L6** — Wave 0 (dev hygiene) lands first so all subsequent wave builders benefit from `mage format-check` pre-commit gating.
- **L7** — Manual-trigger dispatcher is the Drop 4a deliverable. Git/commit/push/Hylla-reingest stay manual until Drop 4b.
- **L8** — Single drop, single PR, single closeout, single Hylla reingest. Not split into 4a-i / 4a-ii / 4a-iii.

## Pre-MVP rules in force (memory-anchored)

- **No migration logic in Go code.** Dev fresh-DBs `~/.tillsyn/tillsyn.db` between schema-touching wave landings.
- **No closeout MD rollups** (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK) — pre-dogfood. Per-drop worklog + PLAN/CLAUDE updates only.
- **Opus builders.** Every builder spawn carries `model: opus` until cascade dogfooding begins.
- **Filesystem-MD mode.** No Tillsyn-runtime per-droplet plan items. PLAN.md droplet rows + per-droplet QA artifacts in tree.
- **Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING** in every subagent response.
- **Single-line commits.** `chore`/`feat`/`fix`/`docs`/`test` conventional, ≤72 chars.
- **Hylla is Go-only today.** Markdown sweeps fall back to `Read` / `rg` without logging Hylla misses.

## Workflow phases (from `workflow/example/drops/WORKFLOW.md`)

- **Phase 1 — Drop dir scaffold.** Done (this brief).
- **Phase 2 — Parallel planners.** 5 planner subagents, one per wave, fired concurrently. Each writes `WAVE_N_PLAN.md` in this dir.
- **Phase 3 — Orch synthesis + plan-QA twins.** Orch reads all 5 wave plans + cross-cuts them into a unified `PLAN.md` with global droplet numbering (4a.1, 4a.2, ...) and explicit `blocked_by` chains. Then plan-QA-proof + plan-QA-falsification dispatch in parallel against the unified PLAN. **No builder fires until both plan-QA passes return green.**
- **Phase 4-5 — Builder + builder-QA per droplet.** Per droplet: build → build-QA-proof + build-QA-falsification (parallel) → fix loop → commit. Per memory rule, no commit/push without both QA passes complete.
- **Phase 6 — Drop close.** `mage ci` locally → push → `gh run watch --exit-status` green. No per-droplet push.
- **Phase 7 — Drop 4b kickoff** immediately after 4a's PR merges. (Drop 4b runs against `main` with 4a's dispatcher already in place.)

## Concrete planner spawn contract

Each parallel planner gets:

- This brief (full text).
- Their assigned wave's scope (the wave-section above).
- Output target: `workflow/drop_4a/WAVE_N_PLAN.md`.
- Section 0 directive verbatim.
- Output contract: per-droplet rows with `Title`, `Paths`, `Packages`, `Acceptance`, `Blocked by` columns. Droplet IDs scoped to their wave (`Wave 1.1`, `Wave 1.2`, ...) — orch renumbers to global (`4a.1`, ...) at synthesis.
- Constraint: no Hylla calls (Go-indexed primitives are stable; the dispatcher work is largely additive); use `Read` / `Grep` / `Glob` / `LSP` for code understanding.
- Constraint: planner does NOT write source code; output is the PLAN fragment only.

## Plan-QA contract (gate before any builder fires)

After orch synthesizes WAVE_*_PLAN.md → PLAN.md, two parallel plan-QA agents fire:

- **plan-QA-proof** (`go-qa-proof-agent`, `model: opus`): verify every droplet's acceptance criteria are concrete and measurable; verify `paths`/`packages` are populated; verify `blocked_by` chain is acyclic + complete; verify wave-internal serialization respects same-file-lock rules.
- **plan-QA-falsification** (`go-qa-falsification-agent`, `model: opus`): apply the 6 cascade-vocabulary attack vectors (5 attacks + §4.4 global L1 sweep) against the unified plan. Specifically: droplet-with-children, segment path/package overlap without `blocked_by`, empty-`blocked_by` confluence, confluence with partial upstream coverage, role/structural_type contradictions; plus the §4.4 sweep — blocker-graph acyclicity, sibling-overlap-without-blockers, leaf-acceptance-criteria-compose-into-L1-outcome, orphan-droplet check.

Both must return PASS (or PASS-WITH-NIT for non-blocking observations) before any builder spawns. CONFIRMED counterexamples loop back to revision.
