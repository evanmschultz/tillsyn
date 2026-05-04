# Drop 4a — Dispatcher Core: Unified Plan

**Working name:** Drop 4a — Dispatcher Core
**Sequencing:** post-Drop-3, pre-Drop-4b
**Total droplets:** 32 across 5 waves
**Mode:** filesystem-MD only (no per-droplet Tillsyn plan items today)
**Plan-QA gate:** plan-QA-proof + plan-QA-falsification fire AGAINST this unified plan before any builder spawns

---

## 1. Goal

Replace the orchestrator-as-dispatcher loop with a programmatic dispatcher in a new `internal/app/dispatcher/` package. Drop 4a delivers the **manual-trigger dispatcher** — `till dispatcher run --action-item <id>` reads template `agent_bindings`, acquires file/package locks, walks tree eligibility, spawns subagents via `claude --agent`, and provisions auth via the new orch-self-approval flow. Git/commit/push/Hylla-reingest stay manual; Drop 4b automates them.

The spawn invocation itself splits across three droplets for testability without `claude` on PATH: 4a.19 constructs the `*exec.Cmd`, 4a.21 executes + monitors, 4a.23 (CLI) orchestrates the full RunOnce path. The split is documented in Q3 and is a deliberate testability choice, not a planning gap.

Combined Drop 4a + 4b = **MVP-feature-complete cascade**.

---

## 2. Locked Architectural Decisions (L1–L8)

Locked at REVISION_BRIEF authoring time, confirmed by parallel planners:

- **L1** — `state` accepted on `till.action_item(operation=create|move)` is the agent-facing API. `column_id` stays in DB but is hidden from agents (resolves via existing `resolveActionItemColumnIDForState`). Columns table retirement deferred to Drop 4.5's TUI overhaul.
- **L2** — Always-on parent-blocks-on-failed-child. `RequireChildrenComplete` policy bit removed; the `CompletionCriteriaUnmet` invariant is unconditional. Bypass via supersede CLI (post-MVP refinement).
- **L3** — `paths` / `packages` / `files` / `start_commit` / `end_commit` are first-class domain fields on `ActionItem`, NOT metadata blob entries.
- **L4** — Project-node first-class fields on `Project`: `hylla_artifact_ref`, `repo_bare_root`, `repo_primary_worktree`, `language`, `build_tool`, `dev_mcp_server_name`. NOT metadata blob.
- **L5** — Drop 1.6 (auth-approval cascade) absorbed into Wave 3. Dispatcher's auto-spawn loop is the only consumer; no reason to ship 1.6 standalone.
- **L6** — Wave 0 (dev hygiene) lands FIRST. Subsequent waves benefit from `.githooks/pre-commit` (`mage format-check`) + `.githooks/pre-push` (`mage ci`) gating from droplet 1.
- **L7** — Drop 4a delivers the **manual-trigger** dispatcher. Git/commit/push/Hylla-reingest stay manual. Auto-promotion-on-state-change wiring + post-build gates land in Drop 4b.
- **L8** — Single drop, single PR, single closeout, single Hylla reingest. Not split into sub-drops.

---

## 3. Pre-MVP Rules In Force

- **No migration logic in Go.** Schema-changing droplets note "Dev fresh-DBs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`."
- **No closeout MD rollups** (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK) — pre-dogfood.
- **Opus builders.** Every builder spawn carries `model: opus`.
- **Filesystem-MD mode.** No Tillsyn-runtime per-droplet plan items.
- **Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING** in every subagent response.
- **Single-line conventional commits.** ≤72 chars.
- **NEVER raw `go test` / `go build` / `go vet` / `mage install`.** Always `mage <target>`. Test-helper fakeagent compilation in Wave 2.8 is a documented carve-out.
- **Hylla is Go-only today.** Markdown sweeps fall back to `Read` / `rg` without logging Hylla misses.

---

## 4. Wave Structure

| Wave | Theme                              | Droplet IDs        | Count | Sequence |
| ---- | ---------------------------------- | ------------------ | ----- | -------- |
| 0    | Dev hygiene infrastructure         | 4a.1 – 4a.4        | 4     | First    |
| 1    | Domain-field infrastructure        | 4a.5 – 4a.13       | 9     | After 0  |
| 2    | Dispatcher loop                    | 4a.14 – 4a.23      | 10    | After 0; cross-wave deps on Wave 1 |
| 3    | Auth integration (Drop 1.6 abs.)   | 4a.24 – 4a.28      | 5     | After 0; can parallel Wave 2 |
| 4    | Closeout MD updates                | 4a.29 – 4a.32      | 4     | After 1+2+3 close |

Total: **32 droplets**.

---

## 5. Wave-Internal-Plan Cross-References

Each wave's full per-droplet acceptance criteria, test scenarios, falsification mitigations, and verification gates live in the per-wave plan files:

- `workflow/drop_4a/WAVE_0_PLAN.md` — dev hygiene (4 droplets)
- `workflow/drop_4a/WAVE_1_PLAN.md` — domain fields (9 droplets)
- `workflow/drop_4a/WAVE_2_PLAN.md` — dispatcher loop (10 droplets)
- `workflow/drop_4a/WAVE_3_PLAN.md` — auth integration (5 droplets)
- `workflow/drop_4a/WAVE_4_PLAN.md` — closeout sweeps (4 droplets)

The droplet rows below carry global IDs + title + summary + cross-wave `blocked_by`. Builders spawn against the unified PLAN's droplet row PLUS the wave plan's full detail.

---

## 6. Wave-to-Global ID Mapping

| Wave-internal | Global   | Title                                                                   |
| ------------- | -------- | ----------------------------------------------------------------------- |
| W0.1          | **4a.1** | `MAGE FORMAT` ERGONOMICS + `MAGE FORMAT-CHECK`                          |
| W0.2          | **4a.2** | `.GITHOOKS/` PRE-COMMIT + PRE-PUSH SCRIPTS                              |
| W0.3          | **4a.3** | `MAGE INSTALL-HOOKS` TARGET                                             |
| W0.4          | **4a.4** | CONTRIBUTING.MD HOOKS DOCS                                              |
| W1.1          | **4a.5** | `paths []string` FIRST-CLASS ON `ActionItem`                            |
| W1.2          | **4a.6** | `packages []string` FIRST-CLASS ON `ActionItem`                         |
| W1.3          | **4a.7** | `files []string` FIRST-CLASS ON `ActionItem`                            |
| W1.4          | **4a.8** | `start_commit string` FIRST-CLASS ON `ActionItem`                       |
| W1.5          | **4a.9** | `end_commit string` FIRST-CLASS ON `ActionItem`                         |
| W1.6          | **4a.10**| `state` ACCEPTED ON MCP CREATE + MOVE                                   |
| W1.7          | **4a.11**| ALWAYS-ON PARENT-BLOCKS; REMOVE `RequireChildrenComplete` POLICY BIT    |
| W1.8          | **4a.12**| PROJECT-NODE FIRST-CLASS FIELDS (6 BUNDLED)                             |
| W1.9          | **4a.13**| VERIFY DEFAULT-COLUMN TITLES USE POST-DROP-2 VOCABULARY                 |
| W2.1          | **4a.14**| DISPATCHER PACKAGE SKELETON + INTERFACE                                 |
| W2.2          | **4a.15**| LIVEWAITBROKER SUBSCRIPTION                                             |
| W2.3          | **4a.16**| FILE-LEVEL LOCK MANAGER                                                 |
| W2.4          | **4a.17**| PACKAGE-LEVEL LOCK MANAGER                                              |
| W2.5          | **4a.18**| TREE WALKER + AUTO-PROMOTION                                            |
| W2.6          | **4a.19**| AGENT SPAWN                                                             |
| W2.7          | **4a.20**| CONFLICT DETECTOR — SIBLING OVERLAP                                     |
| W2.8          | **4a.21**| PROCESS MONITORING                                                      |
| W2.9          | **4a.22**| TERMINAL-STATE CLEANUP                                                  |
| W2.10         | **4a.23**| MANUAL-TRIGGER CLI (`till dispatcher run`)                              |
| W3.1          | **4a.24**| AUTH ROLE ENUM WIDENING + ORCH SELF-APPROVAL GATE + STEWARD EXCEPTION   |
| W3.2          | **4a.25**| PROJECT-METADATA OPT-OUT TOGGLE `orch_self_approval_enabled`            |
| W3.3          | **4a.26**| AUDIT TRAIL: APPROVING-ORCH IDENTITY ON `auth_requests`                 |
| W3.4          | **4a.27**| MCP-LAYER GOLDEN TESTS — 4 APPROVE-PATH CASES                           |
| W3.5          | **4a.28**| DELETE S2 DEV-FALLBACK FROM PROMPTS + MEMORY                            |
| W4.1          | **4a.29**| `MAIN/CLAUDE.MD` DISPATCHER-AWARE SWEEP                                 |
| W4.2          | **4a.30**| `MAIN/WIKI.MD` POST-DISPATCHER COHERENCE CHECK                          |
| W4.3          | **4a.31**| `MAIN/STEWARD_ORCH_PROMPT.MD` §8.1 S2 FALLBACK SWEEP                    |
| W4.4          | **4a.32**| OUTSIDE-REPO AGENT + GLOBAL CLAUDE + MEMORY SWEEP                       |

---

## 7. Per-Droplet Rows

Wave-plan cross-references give the full acceptance detail. The rows below are the global view: title, paths summary, `blocked_by` with global IDs, and one-line notes.

### Wave 0 — Dev Hygiene (4 droplets)

#### 4a.1 — `MAGE FORMAT` ERGONOMICS + `MAGE FORMAT-CHECK`

- **Paths:** `magefile.go`
- **Packages:** `main` (magefile)
- **Acceptance:** Split existing `Format(path)` into no-arg `Format()` (whole tree) + `FormatPath(path)`. Add public `FormatCheck()` wrapper around private `formatCheck()`. Update `Aliases` map. See WAVE_0_PLAN.md §W0.1.
- **Blocked by:** —
- **Notes:** Wave 0 anchor. Single-file, ~25 LOC.

#### 4a.2 — `.GITHOOKS/` PRE-COMMIT + PRE-PUSH SCRIPTS

- **Paths:** `.githooks/pre-commit` (NEW), `.githooks/pre-push` (NEW)
- **Packages:** —
- **Acceptance:** POSIX `sh` scripts, executable mode 0755. pre-commit runs `mage format-check`; pre-push runs `mage ci`. See WAVE_0_PLAN.md §W0.2.
- **Blocked by:** **4a.1** (pre-commit invokes `mage format-check`).
- **Notes:** New `.githooks/` directory.

#### 4a.3 — `MAGE INSTALL-HOOKS` TARGET

- **Paths:** `magefile.go`
- **Packages:** `main`
- **Acceptance:** `InstallHooks()` runs `git config core.hooksPath .githooks` after pre-flight `os.Stat` on hook files. Idempotent. NOT auto-run during `mage ci`. See WAVE_0_PLAN.md §W0.3.
- **Blocked by:** **4a.2** (pre-flight depends on hook files; same-file lock with 4a.1 satisfied transitively).
- **Notes:** Single-file edit, ~30 LOC.

#### 4a.4 — CONTRIBUTING.MD HOOKS DOCS

- **Paths:** `CONTRIBUTING.md`
- **Packages:** —
- **Acceptance:** Replace existing `## Recommended Pre-Push Hook` section (lines 42–57) with new `## Local Git Hooks` section. Add `mage format-check` + `mage install-hooks` to mage target list (lines 19–26). See WAVE_0_PLAN.md §W0.4.
- **Blocked by:** **4a.3** (docs reference the new mage target).
- **Notes:** Pure markdown.

---

### Wave 1 — Domain-Field Infrastructure (9 droplets)

Wave 1's same-file-lock chain anchored at `internal/domain/action_item.go` + `internal/adapters/storage/sqlite/repo.go` + `internal/adapters/server/common/mcp_surface.go` + `internal/app/snapshot.go`. Six droplets (4a.5–4a.11) serialize linearly. 4a.12 (project fields) and 4a.13 (column verify) run in parallel.

#### 4a.5 — `paths []string` FIRST-CLASS ON `ActionItem`

- **Paths:** `internal/domain/{action_item.go,domain_test.go,errors.go}`, `internal/app/{service.go,snapshot.go,snapshot_test.go}`, `internal/adapters/server/common/{mcp_surface.go,app_service_adapter_mcp.go}`, `internal/adapters/server/mcpapi/{extended_tools.go,extended_tools_test.go}`, `internal/adapters/storage/sqlite/{repo.go,repo_test.go}`
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/adapters/storage/sqlite`
- **Acceptance:** Domain field + `CreateActionItemInput` + `UpdateActionItemInput` + SQL `paths_json TEXT NOT NULL DEFAULT '[]'` + MCP wire + snapshot. Domain validation: trim + dedup; reject whitespace-only / backslash-paths with `ErrInvalidPaths`. See WAVE_1_PLAN.md §1.1.
- **Blocked by:** **4a.4** (Wave 0 close).
- **Notes:** Wave 1 anchor; subsequent action-item field droplets serialize behind it. **Storage shape: JSON-encoded text column** (not side-table). DB action: dev fresh-DBs.

#### 4a.6 — `packages []string` FIRST-CLASS ON `ActionItem`

- **Paths:** Same surface as 4a.5.
- **Packages:** Same.
- **Acceptance:** Domain field after `Paths`. SQL `packages_json TEXT NOT NULL DEFAULT '[]'`. Domain-light coverage rule: when `Paths` is non-empty, `Packages` MUST be non-empty (`ErrInvalidPackages` on violation). Strict path→package mapping deferred to Wave 2 lock manager. See WAVE_1_PLAN.md §1.2.
- **Blocked by:** **4a.5** (same-file compile lock).
- **Notes:** DB action: dev fresh-DBs.

#### 4a.7 — `files []string` FIRST-CLASS ON `ActionItem`

- **Paths:** Same surface as 4a.5.
- **Packages:** Same.
- **Acceptance:** Domain field after `Packages`. SQL `files_json TEXT NOT NULL DEFAULT '[]'`. Disjoint-axis rule: `Files` and `Paths` NOT cross-checked (legitimate overlap for read-then-edit workflows). Path-exists validation deferred to consumer. See WAVE_1_PLAN.md §1.3.
- **Blocked by:** **4a.6** (same-file compile lock).
- **Notes:** Drop 4.5 file-viewer is the consumer; pulled forward into 4a per dev's parallelization preference. DB action: dev fresh-DBs.

#### 4a.8 — `start_commit string` FIRST-CLASS ON `ActionItem`

- **Paths:** Same surface as 4a.5.
- **Packages:** Same.
- **Acceptance:** Domain field after `Files`. SQL `start_commit TEXT NOT NULL DEFAULT ''`. Trim-only validation (no hex-format check; supports short-SHA + full-SHA + empty). **Opaque-domain field** — caller supplies the value (orchestrator pre-cascade; dispatcher post-Wave-2). See WAVE_1_PLAN.md §1.4.
- **Blocked by:** **4a.7** (same-file compile lock).
- **Notes:** Drop 4b commit-agent is the consumer. DB action: dev fresh-DBs.

#### 4a.9 — `end_commit string` FIRST-CLASS ON `ActionItem`

- **Paths:** Same surface as 4a.5.
- **Packages:** Same.
- **Acceptance:** Domain field after `StartCommit`. SQL `end_commit TEXT NOT NULL DEFAULT ''`. Trim-only validation, opaque-domain. **Population-timing decision:** `SetLifecycleState` does NOT auto-populate; caller (Wave 2 dispatcher) populates via `UpdateActionItem` before `MoveActionItemState`. See WAVE_1_PLAN.md §1.5.
- **Blocked by:** **4a.8** (same-file compile lock).
- **Notes:** Drop 4b commit-agent consumer. DB action: dev fresh-DBs.

#### 4a.10 — `state` ACCEPTED ON MCP CREATE + MOVE

- **Paths:** `internal/adapters/server/common/{mcp_surface.go,app_service_adapter_mcp.go}`, `internal/adapters/server/mcpapi/{extended_tools.go,extended_tools_test.go}`
- **Packages:** `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
- **Acceptance:** `CreateActionItemRequest` + `MoveActionItemRequest` gain `State string` alongside existing `ColumnID`. Adapter resolves `state` server-side via existing `resolveActionItemColumnIDForState`. Reject when both empty (existing behavior); reject when both non-empty ("specify exactly one"). 8 test cases (4×create + 4×move). See WAVE_1_PLAN.md §1.6.
- **Blocked by:** **4a.9** (same-file compile lock on `mcp_surface.go` + `app_service_adapter_mcp.go`).
- **Notes:** L1. column_id stays in DB; columns table retirement → Drop 4.5. **NO DB action** (adapter-only).

#### 4a.11 — ALWAYS-ON PARENT-BLOCKS; REMOVE `RequireChildrenComplete` POLICY BIT

- **Paths:** `internal/domain/{workitem.go,action_item.go,domain_test.go,kind_capability_test.go}`, `internal/app/{snapshot.go,snapshot_test.go}`, `internal/adapters/server/{common/capture_test.go,mcpapi/instructions_explainer.go,mcpapi/extended_tools.go}`, `internal/templates/builtin/default.toml` (sweep references), `internal/adapters/storage/sqlite/repo.go` (vestigial JSON cleanup)
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/templates`
- **Acceptance:** Delete `CompletionPolicy` struct + field + JSON tag. Make `CompletionCriteriaUnmet`'s children-walk unconditional. `failed` children also block parent close. **Builder MUST run `LSP findReferences` on `RequireChildrenComplete`** before editing — the path list is starting-point lower bound. See WAVE_1_PLAN.md §1.7.
- **Blocked by:** **4a.10** (same-file compile lock on `app_service_adapter_mcp.go` test file + `mcp_surface.go` shared with 4a.10; `action_item.go` chain shared with 4a.5–4a.9).
- **Notes:** L2. Stuck-parent failure mode (no supersede CLI in 4a) is an explicit pre-MVP cost — dev fresh-DBs is the escape valve. DB action: dev fresh-DBs (vestigial `policy.require_children_complete` JSON cleanup).

#### 4a.12 — PROJECT-NODE FIRST-CLASS FIELDS (6 BUNDLED)

- **Paths:** `internal/domain/{project.go,domain_test.go,errors.go}`, `internal/app/{service.go,snapshot.go,snapshot_test.go}`, `internal/adapters/server/common/{mcp_surface.go,app_service_adapter.go}`, `internal/adapters/server/mcpapi/{extended_tools.go,extended_tools_test.go}`, `internal/adapters/storage/sqlite/{repo.go,repo_test.go}`
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
- **Acceptance:** Six fields on `Project`: `HyllaArtifactRef`, `RepoBareRoot`, `RepoPrimaryWorktree`, `Language` (closed enum: `""` | `"go"` | `"fe"`), `BuildTool`, `DevMcpServerName`. Validation: `Language` against closed enum (`ErrInvalidLanguage`); `RepoBareRoot` + `RepoPrimaryWorktree` must be absolute (`ErrInvalidRepoPath`). Bundled per-Drop-3-3.21 precedent. See WAVE_1_PLAN.md §1.8.
- **Blocked by:** **4a.4** (Wave 0 close). **Independent of 4a.5–4a.11; runs parallel.**
- **Notes:** L4. **Irreducible: true** (6 fields × 6 wire surfaces; splitting doubles diff churn). Wave 2's spawn (4a.19) reads ALL six. DB action: dev fresh-DBs. **Same-package parallelization rationale (added post-plan-QA-falsification round-1 NIT):** 4a.12 shares packages `internal/adapters/server/common` (file `mcp_surface.go`) and `internal/adapters/server/mcpapi` (file `extended_tools.go`) with the 4a.5–4a.11 chain. Author-judged textually disjoint — different struct definitions in `mcp_surface.go` (project-level vs action-item-level), different `mcp.WithString` registrations in `extended_tools.go` (`till.project` vs `till.action_item` tools). Parallelization is intentional per Drop 3 droplet 3.21 precedent (separate struct extensions in the same wire-surface files lands cleanly). Builder verifies textual disjointness pre-edit; escalates to serialization on Wave 1 close if a hit surfaces.

#### 4a.13 — VERIFY DEFAULT-COLUMN TITLES USE POST-DROP-2 VOCABULARY

- **Paths:** `internal/app/service.go` (verify only), `internal/adapters/storage/sqlite/repo.go` (verify only); edits only if drift found.
- **Packages:** `internal/app`, `internal/adapters/storage/sqlite` (only if Branch B fires).
- **Acceptance:** Verify `defaultStateTemplates` returns `{"To Do", "In Progress", "Complete", "Failed"}`. Pre-read at HEAD shows it does. **Branch A** (no drift): worklog records evidence, no code change. **Branch B** (drift found): flip stray `"Done"` → `"Complete"` in seeding code. See WAVE_1_PLAN.md §1.9.
- **Blocked by:** **4a.4** (Wave 0 close). **Independent of 4a.5–4a.11; runs parallel.**
- **Notes:** **Irreducible: true** (cross-check droplet). `"To Do"` (display name with space) is intentional UI text; canonical state-ID is `todo`. Worklog records the disambiguation.

---

### Wave 2 — Dispatcher Loop (10 droplets)

Wave 2 lives in new package `internal/app/dispatcher/` (and `cmd/till/dispatcher_cli.go` for the CLI). Linear chain through 4a.14 (skeleton) with three parallel branches reconverging at the CLI.

#### 4a.14 — DISPATCHER PACKAGE SKELETON + INTERFACE

- **Paths:** `internal/app/dispatcher/{dispatcher.go,dispatcher_test.go,doc.go}` (NEW)
- **Packages:** `internal/app/dispatcher` (NEW)
- **Acceptance:** `Dispatcher` interface (`RunOnce`, `Start`, `Stop`). `dispatcher` impl + `NewDispatcher` constructor. `DispatchOutcome` struct + `Result` enum. `Start`/`Stop` may return `ErrNotImplemented` (Drop 4b wires). See WAVE_2_PLAN.md §2.1.
- **Blocked by:** **4a.4** (Wave 0 close). Cross-wave: none.
- **Notes:** Package boundary anchor. Subsequent Wave-2 droplets serialize behind via package-compile lock. **No** Drop 4b coupling (gate-runner / commit-agent / push / reingest fields).

#### 4a.15 — LIVEWAITBROKER SUBSCRIPTION

- **Paths:** `internal/app/dispatcher/{broker_sub.go,broker_sub_test.go}` (NEW), `internal/app/{live_wait.go,coordination_live_wait.go,service.go}`
- **Packages:** `internal/app/dispatcher`, `internal/app`
- **Acceptance:** Add `LiveWaitEventActionItemChanged` event. Add `Service.publishActionItemChanged` helper. `MoveActionItem` / `CreateActionItem` / `UpdateActionItem` publish on success. Dispatcher subscribes via re-subscribe loop. 100 ms-bounded test scenarios. See WAVE_2_PLAN.md §2.2.
- **Blocked by:** **4a.14** (package-compile lock), **4a.11** (same-file lock on `internal/app/service.go` — Wave 1 chain edits the same `Service.{Move,Create,Update}ActionItem` methods that 4a.15 extends with `publishActionItemChanged` calls; serializing through Wave 1's terminal node prevents merge conflicts on the publisher additions).
- **Notes:** Plan-QA-falsification round-1 CONFIRMED counterexample fix (2026-05-03). Original sketch flagged `coordination_live_wait.go` as a "soft cross-wave concern" but missed that Wave 1's chain edits `service.go` `Service.{Move,Create,Update}ActionItem` methods that 4a.15 extends. Same-file lock now explicit.

#### 4a.16 — FILE-LEVEL LOCK MANAGER

- **Paths:** `internal/app/dispatcher/{locks_file.go,locks_file_test.go}` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:** `fileLockManager` with `Acquire(actionItemID, paths)` + `Release(actionItemID)`. In-process `sync.Mutex` + `map[string]string`. Race-free; 100% coverage on file. **Opaque API** — paths are normalized by caller, not the manager. See WAVE_2_PLAN.md §2.3.
- **Blocked by:** **4a.14** (package-compile lock).
- **Notes:** No Drop 4b SQLite-persist; in-process for 4a.

#### 4a.17 — PACKAGE-LEVEL LOCK MANAGER

- **Paths:** `internal/app/dispatcher/{locks_package.go,locks_package_test.go}` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:** `packageLockManager` mirroring `fileLockManager` shape. Independent of file-lock map; cross-locking semantics live in walker (4a.18) + conflict detector (4a.20). 100% coverage. See WAVE_2_PLAN.md §2.4.
- **Blocked by:** **4a.14**, **4a.16** (package-compile lock).
- **Notes:** No premature `lockManager[K]` generic — Drop 4b evolves package-lock distinct from file-lock.

#### 4a.18 — TREE WALKER + AUTO-PROMOTION

- **Paths:** `internal/app/dispatcher/{walker.go,walker_test.go}` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:** `treeWalker` with `EligibleForPromotion` (returns `[]ActionItem` in `todo` whose `BlockedBy` are all `complete`, parent in `in_progress` or `Persistent=true`). `Promote` calls `Service.MoveActionItem` with state-resolved column. `ErrPromotionBlocked` typed error wrapping inner. See WAVE_2_PLAN.md §2.5.
- **Blocked by:** **4a.14**, **4a.15** (package-compile + needs subscriber alignment), **4a.6** (reads `Packages` + `Paths`), **4a.10** (uses `state` MCP surface), **4a.11** (eligibility relies on always-on parent-block invariant).
- **Notes:** Walker is read-only on the tree; promote is a separate method so 4a.20 (conflict) can intercede.

#### 4a.19 — AGENT SPAWN

- **Paths:** `internal/app/dispatcher/{spawn.go,spawn_test.go}` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:** `BuildSpawnCommand` constructs `*exec.Cmd` (does NOT execute) with `Dir = project.RepoPrimaryWorktree`, full argv per `claude --agent <name> --bare …`. Reads `templates.KindCatalog.LookupAgentBinding` for agent variant. **`AuthBundle` is a stub** — Wave 3 fills it. `ErrNoAgentBinding` for unbound kinds. See WAVE_2_PLAN.md §2.6.
- **Blocked by:** **4a.14**, **4a.12** (reads project-node fields).
- **Notes:** Auth-bundle stub is the explicit Wave-3 seam. `--mcp-config` placeholder path documented.

#### 4a.20 — CONFLICT DETECTOR — SIBLING OVERLAP

- **Paths:** `internal/app/dispatcher/{conflict.go,conflict_test.go}` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:** `conflictDetector.DetectSiblingOverlap` returns `[]SiblingOverlap` (file or package). `InsertRuntimeBlockedBy` adds to `BlockedBy` via `Service.UpdateActionItem`; idempotent. **Posts attention_item** on insertion. Tied: by `Position` then `ID` lex. See WAVE_2_PLAN.md §2.7.
- **Blocked by:** **4a.14**, **4a.18** (walker — runs after eligibility but before promote), **4a.6** (reads `Packages`).
- **Notes:** Drop 4a treats sibling-only (cousins → Drop 4b). Inserted runtime blockers stay (the runtime blocker IS the dependency edge).

#### 4a.21 — PROCESS MONITORING

- **Paths:** `internal/app/dispatcher/{monitor.go,monitor_test.go,testdata/fakeagent.go}` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:** `processMonitor.Track` starts the process + returns `Handle`. `Handle.Wait` returns `TerminationOutcome`. On crash: `MoveActionItem` to `failed` + `metadata.outcome = "failure"` + `metadata.failure_reason`. **Concurrent-safe**; goroutine-leak-free. Test uses `testdata/fakeagent.go` compiled via `go build` (test-helper carve-out). See WAVE_2_PLAN.md §2.8.
- **Blocked by:** **4a.14**, **4a.19** (consumes `*exec.Cmd` from spawn).
- **Notes:** **`metadata.failure_reason` shape: free-form string for 4a.** Drop 4b refactors to structured type. **Test-helper `go build` carve-out** is the one exception to "never raw `go`."

#### 4a.22 — TERMINAL-STATE CLEANUP

- **Paths:** `internal/app/dispatcher/{cleanup.go,cleanup_test.go}` (NEW)
- **Packages:** `internal/app/dispatcher`
- **Acceptance:** `cleanupHook.OnTerminalState` (`complete` | `failed` | `archived`): release file + package locks; **revoke auth-bundle stub** (`// Wave 3 fills this in`); unsubscribe monitor's PID map. Idempotent. `errors.Join` for partial failures. See WAVE_2_PLAN.md §2.9.
- **Blocked by:** **4a.14**, **4a.16**, **4a.17**, **4a.19** (touches both lock managers + auth-bundle stub from spawn).
- **Notes:** Auth-revoke is stub — Drop 4b lands real revoke. Documented audit-gap.

#### 4a.23 — MANUAL-TRIGGER CLI (`till dispatcher run`)

- **Paths:** `cmd/till/{dispatcher_cli.go,dispatcher_cli_test.go,main.go}`
- **Packages:** `cmd/till`
- **Acceptance:** `till dispatcher run --action-item <id>` cobra subcommand. Flags: `--action-item` (required), `--project` (optional), `--dry-run`. RunE: instantiate `Dispatcher`, `RunOnce`, print `DispatchOutcome`. Dry-run: print spawn descriptor JSON without exec. Exits non-zero on `Result=Failed`. See WAVE_2_PLAN.md §2.10.
- **Blocked by:** **4a.14**, **4a.18**, **4a.19**, **4a.20**, **4a.21**, **4a.22**, **4a.12** (project fields).
- **Notes:** L7 manual-trigger milestone deliverable. Per-CLI bootstrap (no shared broker with `till serve`); Drop 4b lands daemon variant.

---

### Wave 3 — Auth Integration (5 droplets)

Strict linear within wave: 4a.24 → 4a.25 → 4a.26 → 4a.27 → 4a.28. **Can run parallel with Wave 2** since 4a.19 (spawn) ships an auth-bundle stub that 4a.24 fills in via Wave 3's flow.

#### 4a.24 — AUTH ROLE ENUM WIDENING + ORCH SELF-APPROVAL GATE + STEWARD CROSS-SUBTREE EXCEPTION

- **Paths:** `internal/domain/{auth_request.go,auth_request_test.go}`, `internal/app/{auth_requests.go,auth_requests_test.go}`, `internal/adapters/auth/autentauth/service.go`, `internal/adapters/server/common/{mcp_surface.go,app_service_adapter_mcp.go,app_service_adapter_auth_requests_test.go}`, `internal/adapters/server/mcpapi/handler.go`
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/auth/autentauth`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
- **Acceptance:** Widen `AuthRequestRole` enum 4→7 values (`orchestrator`, `planner`, `qa-proof`, `qa-falsification`, `builder`, `research`, `commit`); bare `qa` rejected. Add `approve` operation to `till.auth_request` MCP tool. Service-layer self-approval gate (5 reject conditions + STEWARD persistent-parent exception via metadata-driven detection). 7+ test cases. See WAVE_3_PLAN.md §W3.1.
- **Blocked by:** **4a.4** (Wave 0 close). **Can run parallel with Wave 2.**
- **Notes:** Largest single droplet (~150 LOC across 5 packages). Builder MUST run package-by-package `mage test-pkg` early to surface enum-change ripple. DB action: dev fresh-DBs (legacy `qa` rows fail normalize). **Same-file parallelization rationale (added post-plan-QA-falsification round-2 NIT):** 4a.24 shares files `internal/adapters/server/common/{mcp_surface.go,app_service_adapter_mcp.go}` with the Wave 1 chain (4a.5–4a.11) and with 4a.12. 4a.24 adds NEW struct `ApproveAuthRequestRequest` + NEW method `ApproveAuthRequest` at file-end; Wave 1 + 4a.12 extend existing structs/methods at different line ranges. Textual disjointness same as 4a.12 ↔ Wave 1 case (per Drop 3 droplet 3.21 precedent). Builder verifies disjointness pre-edit; escalates to serialization on Wave 1 close if a hit surfaces. Wave 0's `.githooks/pre-commit` (`mage format-check`) + `pre-push` (`mage ci`) catch any rebase-induced gofmt collision before push.

#### 4a.25 — PROJECT-METADATA OPT-OUT TOGGLE `orch_self_approval_enabled`

- **Paths:** `internal/domain/{project.go,project_test.go}`, `internal/app/{auth_requests.go,auth_requests_test.go}`, `internal/adapters/server/common/{app_service_adapter_mcp.go,mcp_surface.go,app_service_adapter_auth_requests_test.go}`
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/common`
- **Acceptance:** `OrchSelfApprovalEnabled *bool` on `ProjectMetadata` (pointer; nil = enabled = default-true). `(ProjectMetadata).OrchSelfApprovalIsEnabled()` helper. `ErrOrchSelfApprovalDisabled` sentinel. **Toggle is total** — disables ALL orch-self-approval including STEWARD's cross-subtree path. See WAVE_3_PLAN.md §W3.2.
- **Blocked by:** **4a.24** (same `app_service_adapter_mcp.go` file + gate logic must exist before toggle wraps it).
- **Notes:** Pointer-bool defeats falsification attack 3 (silent-disable on missing JSON key). **Same-file parallelization rationale (added post-plan-QA-falsification round-2 NIT):** 4a.25 shares file `internal/domain/project.go` with 4a.12 (Wave 1 project-fields). 4a.12 extends `Project` struct; 4a.25 extends `ProjectMetadata` struct. Different structs, different line ranges, textually disjoint. Same parallelization rationale as 4a.24 ↔ Wave 1 chain. Wave 0 hooks catch any rebase collision before push.

#### 4a.26 — AUDIT TRAIL: APPROVING-ORCH IDENTITY ON `auth_requests`

- **Paths:** `internal/domain/auth_request.go`, `internal/adapters/auth/autentauth/{service.go,service_test.go}`, `internal/adapters/server/common/{app_service_adapter_mcp.go,mcp_surface.go,app_service_adapter_auth_requests_test.go}`
- **Packages:** `internal/domain`, `internal/adapters/auth/autentauth`, `internal/adapters/server/common`
- **Acceptance:** Three new fields on `AuthRequest`: `ApprovingPrincipalID`, `ApprovingAgentInstanceID`, `ApprovingLeaseToken`. `(*AuthRequest).Approve` signature widened. SQLite columns added via existing `ALTER TABLE ADD COLUMN` precedent (`ensureAuthRequestSchema`). Surfaced in `AuthRequestRecord` via `omitempty`. Domain-layer assertion: non-empty audit fields when approver is `agent` actor type. See WAVE_3_PLAN.md §W3.3.
- **Blocked by:** **4a.25** (same `app_service_adapter_mcp.go` `mapAuthRequestRecord` lock).
- **Notes:** Audit columns are top-level (not JSON blob). DB action: dev fresh-DBs.

#### 4a.27 — MCP-LAYER GOLDEN TESTS — 4 APPROVE-PATH CASES

- **Paths:** `internal/adapters/server/mcpapi/{handler_test.go,handler_steward_integration_test.go}`
- **Packages:** `internal/adapters/server/mcpapi`
- **Acceptance:** 5 test cases: (a) orch-in-subtree approves non-orch → success; (b) orch approves another orchestrator → reject; (c) cross-orch subtree approval → reject; (d) STEWARD approves under persistent parent → success; (e) project toggle disabled → reject. **Tool-schema-shape assertion:** `till.auth_request` `approve` op accepts ONLY documented args (no configurability knobs). See WAVE_3_PLAN.md §W3.4.
- **Blocked by:** **4a.26** (audit fields exercised in case (a) + (d) assertions).
- **Notes:** Brief item 6 verbatim. Coverage threshold (`mage ci`) requires ≥70% line coverage on the new approve-path code.

#### 4a.28 — DELETE S2 DEV-FALLBACK FROM PROMPTS + MEMORY

- **Paths (in-repo):** `STEWARD_ORCH_PROMPT.md`
- **Paths (outside-repo, audit-gap accept):** `~/.claude/agents/{go-builder,go-planning,go-qa-proof,go-qa-falsification,go-research}-agent.md`, memory `feedback_steward_spawn_drop_orch_flow.md`, memory `project_steward_auth_bootstrap.md`
- **Packages:** —
- **Acceptance:** Delete S2 dev-fallback paragraphs from STEWARD §8.1 + agent files + memory caveats. Replace with single "Wave 3 of Drop 4a landed orch-self-approval as canonical" sentence. `rg "S2 dev-fallback|pre-§19.1.6"` returns zero hits. See WAVE_3_PLAN.md §W3.5.
- **Blocked by:** **4a.27** (tested-green capability before docs flip).
- **Notes:** Markdown-only droplet; outside-repo edits recorded in worklog only (audit-gap acceptance per drop_3/3.27 finding 5.D.10).

---

### Wave 4 — Closeout MD Updates (4 droplets)

#### 4a.29 — `MAIN/CLAUDE.MD` DISPATCHER-AWARE SWEEP

- **Paths:** `main/CLAUDE.md`
- **Packages:** —
- **Acceptance:** Surgical sweep: `## Cascade Plan` adds dispatcher-landing sentence. `## Cascade Tree Structure` § "State-Trigger Dispatch" + "Agent Bindings" + "Post-Build Gates" gain post-Wave-2 sentences. `## Action-Item Lifecycle` flips Drop-1 references to past tense. `## Paths and Packages` section heading drops `(Drop-1 Target)` and rewrites body for first-class fields. `## Auth and Leases` retires "auto-revoke is Drop-1 item" (lands in Drop 4b) + adds Wave-3 orch-self-approval. See WAVE_4_PLAN.md §W4.1.
- **Blocked by:** **4a.11**, **4a.12**, **4a.13** (Wave 1 close — three terminal nodes), **4a.23** (Wave 2 close), **4a.28** (Wave 3 close).
- **Notes:** **Irreducible: true.** Orch-driven by default; escalate to builder if hits > ~15. **Description-symbol verification** required before writing post-Drop-4a sentences naming code symbols. **Q9 pre-spawn resolution (added post-plan-QA-proof round-1 NIT):** before this droplet starts, orchestrator pre-resolves the "Drop 1 of the cascade plan" prose references in `main/CLAUDE.md` § Action-Item Lifecycle against actual code state via LSP workspace symbol search for `StateFailed` / `RequireChildrenComplete`. The Drop-N references in the prose may not match the actual plan number that landed `failed` (Drop 4a Wave 1 vs original Drop 1). Spawn prompt names the resolved Drop-N references explicitly so the closeout vocabulary stays consistent across W4.1 / W4.2 / W4.4.

#### 4a.30 — `MAIN/WIKI.MD` POST-DISPATCHER COHERENCE CHECK

- **Paths:** `main/WIKI.md`
- **Packages:** —
- **Acceptance:** Pointer-only edits (canonical glossary owned by 3.6). `## Drop Decomposition Rules` § "Ordering" past-tenses dispatcher-runtime-blocker bullet. `## Build-QA-Commit Loop (Pre-Cascade)` lead sentence rewrites for manual-trigger dispatcher. `## Auth Approval Cascade` line 242 retires `Pre-fix vs post-fix state` paragraph. `## Drop-End Closeout Checklist` adds project-node first-class note. See WAVE_4_PLAN.md §W4.2.
- **Blocked by:** **4a.11**, **4a.12**, **4a.13**, **4a.23**, **4a.28**.
- **Notes:** **Irreducible: true.** Orch-driven; small edit set (~3-5 hits).

#### 4a.31 — `MAIN/STEWARD_ORCH_PROMPT.MD` §8.1 S2 FALLBACK SWEEP

- **Paths:** `main/STEWARD_ORCH_PROMPT.md`
- **Packages:** —
- **Acceptance:** Delete S2 fallback paragraph at line 302. Lead paragraph at line 261 drops "(pre-§19.1.6 fix drop)" parenthetical. Cross-references to §19.1.6 reviewed; rewrite to past-tense or delete. `rg "§19\.1\.6|pre-§19\.1\.6"` returns either zero or only past-tense hits. See WAVE_4_PLAN.md §W4.3.
- **Blocked by:** **4a.28** (Wave 3 close only — STEWARD prompt is auth-flow scoped).
- **Notes:** **Irreducible: true.** Orch-driven; ~3-5 hits all in §8.1 + cross-references.

#### 4a.32 — OUTSIDE-REPO AGENT + GLOBAL CLAUDE + MEMORY SWEEP

- **Paths (NOT git-tracked; audit-gap accept):** `~/.claude/agents/go-{builder,planning,qa-proof,qa-falsification}-agent.md`, `~/.claude/CLAUDE.md`, memory `feedback_steward_spawn_drop_orch_flow.md`, memory `project_steward_auth_bootstrap.md`
- **Packages:** —
- **Acceptance:** Per agent file: search for S2-fallback paragraph, retire if present, "no paragraph found" is valid worklog outcome. `task_id` rename NOT in scope (REVISION_BRIEF anticipatory; Wave 1 didn't actually rename). Cross-doc consistency: post-Wave-3 canonical sentence chosen in 4a.29 mirrored across all 7 outside-repo paths. See WAVE_4_PLAN.md §W4.4.
- **Blocked by:** **4a.28** (Wave 3 close), **4a.29** (canonical post-Wave-3 sentence chosen first).
- **Notes:** **Irreducible: true.** **Builder-driven** (7 files + cross-doc consistency check too heavy for in-orch attention). Per drop_3/3.27 finding 5.D.10 audit-gap acceptance.

---

## 8. Cross-Wave Blocker Wiring (DAG)

The hard `blocked_by` edges across waves. Acyclicity verified by topological sort below.

| Blocker (must complete) | Blocked droplet | Reason |
| --- | --- | --- |
| 4a.4                    | 4a.5 (Wave 1.1)        | Wave 0 close — pre-commit gating active before Wave-1 builders start. |
| 4a.4                    | 4a.12 (Wave 1.8)       | Wave 0 close. |
| 4a.4                    | 4a.13 (Wave 1.9)       | Wave 0 close. |
| 4a.4                    | 4a.14 (Wave 2.1)       | Wave 0 close — dispatcher skeleton can start parallel with Wave 1. |
| 4a.4                    | 4a.24 (Wave 3.1)       | Wave 0 close — auth code can start parallel with Wave 1+2. |
| 4a.11                   | 4a.15 (broker sub)     | Same-file lock on `internal/app/service.go` — Wave 1 chain edits `Service.{Move,Create,Update}ActionItem`; 4a.15 extends them with `publishActionItemChanged` calls. (Added post-plan-QA-falsification round-1 fix.) |
| 4a.6                    | 4a.18 (walker)         | Walker reads `actionItem.Paths` AND `actionItem.Packages`. |
| 4a.6                    | 4a.20 (conflict)       | Conflict detector reads `Paths` + `Packages` for sibling-overlap detection. |
| 4a.10                   | 4a.18 (walker)         | Walker uses `state` MCP API for promotion. |
| 4a.11                   | 4a.18 (walker)         | Eligibility predicate relies on always-on parent-block invariant. |
| 4a.12                   | 4a.19 (spawn)          | Spawn reads `RepoPrimaryWorktree`, `Language`, `HyllaArtifactRef`, `DevMcpServerName`. |
| 4a.12                   | 4a.23 (CLI)            | CLI bootstrap constructs Service against project with project fields populated. |
| 4a.11, 4a.12, 4a.13     | 4a.29 (CLAUDE.md)      | Wave 1 close — docs reflect first-class field landing + always-on block. |
| 4a.11, 4a.12, 4a.13     | 4a.30 (WIKI.md)        | Wave 1 close. |
| 4a.23                   | 4a.29 (CLAUDE.md)      | Wave 2 close — docs reflect dispatcher landing. |
| 4a.23                   | 4a.30 (WIKI.md)        | Wave 2 close. |
| 4a.28                   | 4a.29 (CLAUDE.md)      | Wave 3 close — docs reflect orch-self-approval landing. |
| 4a.28                   | 4a.30 (WIKI.md)        | Wave 3 close. |
| 4a.28                   | 4a.31 (STEWARD)        | Wave 3 close — STEWARD prompt is auth-flow scoped. |
| 4a.28                   | 4a.32 (outside-repo)   | Wave 3 close. |
| 4a.29                   | 4a.32 (outside-repo)   | Cross-doc consistency — canonical sentence chosen first. |

Wave-internal `blocked_by` edges (same-file / same-package locks) are listed in each droplet's row above.

---

## 9. Topological Order

One valid full-graph topological sort (parallelism opportunities annotated):

```
Wave 0 (strict serial):
  4a.1 → 4a.2 → 4a.3 → 4a.4

Wave 1 (linear chain on action_item.go + parallel branches):
  4a.5 → 4a.6 → 4a.7 → 4a.8 → 4a.9 → 4a.10 → 4a.11
  4a.12 (parallel from 4a.4)
  4a.13 (parallel from 4a.4)

Wave 2 (linear chain on dispatcher package + parallel branches):
  4a.14 → 4a.15 (after 4a.11 lands — same-file lock on service.go) → 4a.18    (broker → walker, after 4a.10/4a.11 land)
  4a.14 → 4a.16 → 4a.17    (file-lock → package-lock)
  4a.14 → 4a.19 (after 4a.12 lands) → 4a.21 (monitor)
  4a.18 + 4a.6 → 4a.20 (conflict)
  4a.16 + 4a.17 + 4a.19 → 4a.22 (cleanup)
  4a.18 + 4a.19 + 4a.20 + 4a.21 + 4a.22 + 4a.12 → 4a.23 (CLI)

Wave 3 (strict serial; parallel-OK with Wave 2):
  4a.24 → 4a.25 → 4a.26 → 4a.27 → 4a.28

Wave 4 (closeout, after waves 1+2+3 close):
  4a.29 (after Wave 1 + Wave 2 + Wave 3 close)
  4a.30 (after Wave 1 + Wave 2 + Wave 3 close)
  4a.31 (after Wave 3 close only)
  4a.32 (after Wave 3 close + 4a.29)
```

Wave 1 + Wave 2 + Wave 3 can overlap meaningfully:

- After 4a.4 closes, 4a.5 + 4a.12 + 4a.13 + 4a.14 + 4a.24 all start in parallel.
- After 4a.11 closes, 4a.15 unblocks (with 4a.14 also required); the publisher additions to `Service.{Move,Create,Update}ActionItem` only become safe to land once Wave 1's chain has finished editing those same methods.
- After 4a.6 closes, 4a.18 unblocks (with 4a.10 + 4a.11 + 4a.15 also required).
- Wave 3's auth code is independent of Wave 2's dispatcher code (the auth-bundle is a stub from 4a.19's perspective until W3 fills it post-merge).

---

## 10. Open Questions Routed To Plan-QA-Falsification

Surfaced inline by the parallel planners for plan-QA's adversarial pass:

- **Q1 — Wave 1 stuck-parent failure mode (4a.11).** No supersede CLI in 4a; dev fresh-DBs is the escape valve. Acceptable pre-MVP cost or blocker?
- **Q2 — Wave 2 same-package vs sub-package locks (`internal/app/dispatcher/locks/`).** 10 droplets in one package serialize via package-lock. Sub-package adds navigation cost; flat keeps the chain. Author's stance: keep flat.
- **Q3 — Wave 2 cmd-construction vs cmd-execution split (4a.19 vs 4a.21).** Splits required for testability without `claude` on PATH. 4a.21 owns both execution + monitoring (wider scope than brief implied). Plan-QA review.
- **Q4 — Wave 2 auth-bundle stub seam (4a.19).** Stub interface is `AuthBundle` zero-value + placeholder `--mcp-config` path. Sufficient for Wave 3 to plug into, or does 4a.19 need a more concrete contract?
- **Q5 — Wave 2 `metadata.failure_reason` shape (4a.21).** Free-form string today; Drop 4b's PLAN.md L76 spec ("`failure` concrete type with `failure_kind`/`diagnostic`/`fix_directive`") is deferred. Author's stance: free-form for 4a; 4b refactors.
- **Q6 — Wave 2 test-helper `go build` carve-out (4a.21 fakeagent).** Mage rule says "never raw `go build`" — but `testdata/fakeagent.go` compilation in a test setup IS a `go build`. Precedent in `cmd/till/main_test.go`. Acceptable as test-helper carve-out?
- **Q7 — Wave 2 CLI bootstrap symmetry with `till serve` daemon (4a.23).** 4a.23 bootstraps Service + Broker per CLI invocation. Drop 4b lands daemon variant. Should 4a.23 already structure for daemon, or is that forward-engineering YAGNI? Author's stance: minimal CLI today; 4b refactors.
- **Q8 — Wave 3 4a.24 scope size.** Largest single droplet (~150 LOC across 5 packages). Builder discipline: package-by-package `mage test-pkg` early to surface enum-change ripple. Splittable, or accept the size?
- **Q9 — Wave 4 4a.29 description-symbol verification.** `## Action-Item Lifecycle` references "Drop 1 of the cascade plan" — verify against actual code state which Wave landed `failed` (Wave 1 of Drop 4a vs Drop 1 of original plan). Plan numbers shifted; orch confirms before edit.
- **Q10 — Wave 4 4a.32 outside-repo audit-gap.** Per drop_3/3.27 finding 5.D.10, outside-repo edits documented in worklog only. Reconfirm acceptance for 4a or revisit.
- **Q11 — Wave 1.7's stuck-parent test for `failed` children (4a.11).** Brief-default is `failed` blocks parent close. Plan-QA falsification attack: is this the right semantics, or does `failed` get a separate "stuck" treatment?
- **Q12 — Wave 1.6 column_id back-compat (4a.10).** Both `state` and `column_id` accepted. "Specify exactly one" rejection. Is the dual-acceptance a YAGNI hazard, or load-bearing for TUI drag-and-drop?

---

## 11. Drop 4b Kickoff (Post-4a Merge)

Drop 4b runs immediately after 4a's PR merges. 4b builds on 4a's manual-trigger dispatcher and adds:

- Gate runner (template `[gates]` execution).
- Commit-agent (haiku) integration.
- `git commit` + `git push` automation.
- Hylla reingest hook on `closeout`.
- Auth auto-revoke on terminal state.
- Git-status-pre-check on action-item creation.
- Auto-promotion-on-state-change (continuous mode; complements 4a's manual-trigger).

After 4a + 4b: MVP-feature-complete cascade. Drop 5 (dogfooding) is the validation.

Drop 4.5 (TUI overhaul + columns-table retirement + file-viewer pane) runs concurrent with Drop 5.
