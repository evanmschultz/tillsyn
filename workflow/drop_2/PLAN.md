# DROP_2 — HIERARCHY REFACTOR

**State:** planning
**Blocked by:** —
**Paths (expected):** `internal/domain/`, `internal/app/`, `internal/adapters/storage/sqlite/`, `internal/adapters/server/common/`, `internal/adapters/server/mcpapi/`, `internal/tui/`, `internal/config/`, `templates/builtin/` (deletion), `cmd/till/`
**Packages (expected):** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/tui`, `internal/config`, `cmd/till`
**PLAN.md ref:** `main/PLAN.md` § 19.2 — drop 2 — Hierarchy Refactor
**Started:** 2026-05-01
**Closed:** —

## Scope

Drop 2 is the hierarchy-refactor drop. Four units of work, all grounded in `main/PLAN.md` § 19.2:

1. **Promote `metadata.role` to a first-class domain field.** Closed-enum `Role` type with 9 values (`builder`, `qa-proof`, `qa-falsification`, `qa-a11y`, `qa-visual`, `design`, `commit`, `planner`, `research`). Pure parser (`ParseRoleFromDescription`) lives in `internal/domain/role.go`. `Role` field added to `ActionItem` struct with validation. SQLite schema column. MCP `role` field on action-item create/update/get + snapshot serialization. **No hydration runner, no `till migrate` CLI subcommand, no SQL backfill — pre-MVP, dev deletes `~/.tillsyn/tillsyn.db` after the unit lands.**
2. **State-vocabulary rename: `done → complete` AND `progress → in_progress`** (bundled). Touches every file in the tree that names `domain.StateDone` / `domain.StateProgress` symbols, every legacy state literal at a state-machine site, and `ChecklistItem.Done bool → ChecklistItem.Complete bool` field including JSON serialization key. **Pre-step: delete `templates/builtin/*.json` and the `templates/` package entirely (Drop 3 will overhaul the template system from scratch).** No state-rewrite SQL script; dev deletes DB.
3. **Strip hardwired nesting defaults from the domain catalog (mechanism stays).** Set every `KindDefinition.AllowedParentScopes` to empty in boot-seed payloads (`internal/adapters/storage/sqlite/repo.go`). The `AllowsParentScope` enforcement path at `internal/app/kind_capability.go:566` continues to work — empty defaults make it return true for every parent (universal-allow), per the empty-list early return at `internal/domain/kind.go:227-229`. Delete the speculative `domain.AllowedParentKinds(Kind) []Kind` function (zero production callers per PLAN.md). One DB UPDATE script for any existing rows' `allowed_parent_scopes_json` is also OUT — dev fresh-DBs.
4. **Dotted-address fast-nav reads.** Pure resolver in `internal/app` taking a dotted string + project context, returns UUID or ambiguity/missing error. Wire into `till.action_item(operation=get)` MCP read + CLI read commands. Mutation paths reject dotted form. TUI bindings deferred to Drop 4.5.

**Order matters per PLAN.md § 19.2:** template deletion (Unit B-zero, Droplet 2.1) → role promotion (Unit A, no state-machine changes) → state rename (Unit B, ONE atomic droplet that flips every reference) → strip nesting defaults (Unit C, orthogonal) → dotted-address reads (Unit D, lands last so rename churn settles before resolver tests).

**Out of scope (explicit, per PLAN.md § 19.2):** commit cadence rules, reverse-hierarchy prohibitions, auto-create rules, template wiring (all Drop 3); dispatcher (Drop 4); TUI overhaul (Drop 4.5); `scope` column removal (deferred to a future refinement drop).

**Pre-MVP rules in effect (per memory):**

- No migration logic in Go code, no `till migrate` subcommands, no one-shot SQL scripts. Dev deletes `~/.tillsyn/tillsyn.db` between schema or state-vocab-changing units.
- No `CLOSEOUT.md`, no `LEDGER.md` entry, no `WIKI_CHANGELOG.md` entry, no `REFINEMENTS.md` entry, no `HYLLA_FEEDBACK.md` rollup, no `HYLLA_REFINEMENTS.md` rollup. Worklog MDs (this `PLAN.md`, `BUILDER_WORKLOG.md`, `BUILDER_QA_*.md`, `PLAN_QA_*.md`) DO happen.
- Drop 2 closes when all `main/PLAN.md` § 19.2 checkboxes are checked. No separate state-bearing row.

## Planner

Decomposition into 11 atomic droplets. Order: Unit B-zero → Unit A → Unit B (single atomic droplet) → Unit C → Unit D, per the PLAN.md § 19.2 ordering paragraph and the pre-MVP rules in the Scope section above. Each unit lands `mage ci` green before the next unit's first droplet starts. Within a unit, droplets that share a Go package carry an explicit `Blocked by:` to the prior package-touching droplet — same-package-parallel-edits break each other's compile.

Acceptance verification target throughout: `mage test-pkg <pkg>` (per droplet) and `mage ci` (per unit boundary). **Never `mage install`** — dev-only.

**Unit B is one atomic droplet (Round 2 revision).** Round 1 split Unit B into three droplets (2.7 domain / 2.8 TUI / 2.9 MCP); Plan-QA proof + falsification both flagged that the symbol rename has whole-tree fanout that breaks the mage-ci-green-between-droplets invariant if partitioned. Strict-canonical is a single atomic invariant. Unit B (now Droplet 2.7) flips every reference in one commit — every `StateDone` / `StateProgress` symbol, every legacy state literal at a state-machine site, every `ChecklistItem.Done` field reference, every aggregate-counter field tied to state vocab.

---

### Unit B-zero — Delete builtin template JSON + neutralize loader (prerequisite to all other units)

The pre-step from Scope item 2. Lands first because Unit B's state rename would otherwise have to sweep ~80 `"done": false` and `"progress"` literals across `templates/builtin/default-go.json` (76k file) + `templates/builtin/default-frontend.json` (71k file). Drop 3 overhauls templates from scratch — keeping these JSON files alive through Drop 2 only to delete them at Drop 3 start is wasted churn.

**Loader-coupling investigation result (see `## Notes` for full analysis):** `templates/embed.go` is the only Go file in the `templates` package and uses `//go:embed builtin/*.json`. Zero importers anywhere in the Go tree (verified: `git grep "evanmschultz/tillsyn/templates"` returns empty). Per Go embed semantics, deleting all matching files makes `//go:embed builtin/*.json` a build error. **Therefore Unit B-zero must delete the entire `templates/` package — both the JSON files AND `templates/embed.go`** in one droplet. Going with full deletion: simpler, no orphan code, Drop 3 reintroduces a fresh `templates/` package on its own terms.

#### Droplet 2.1 — Delete `templates/` package outright

- **State:** done
- **Paths:** `templates/builtin/default-go.json` (delete), `templates/builtin/default-frontend.json` (delete), `templates/embed.go` (delete), `templates/builtin/` (delete dir if empty), `templates/` (delete dir if empty)
- **Packages:** `github.com/evanmschultz/tillsyn/templates` (deletion)
- **Acceptance:**
  - `git rm` removes all four paths above (plus their parent dirs if empty).
  - `git grep "evanmschultz/tillsyn/templates"` returns empty across the whole repo (no orphan imports).
  - `git grep "templates/builtin"` returns only MD references in `README.md`, `PLAN.md`, `CLAUDE.md`, and `workflow/drop_2/PLAN.md` (those are MD content edits, NOT Go-tree references, and may stay until Drop 3 cleanup or be touched by builder if trivially in scope — see Notes).
  - `mage ci` green.
  - DB action: NONE.
- **Blocked by:** —

---

### Unit A — Promote `metadata.role` to first-class domain field

Closed-enum 9 values: `builder`, `qa-proof`, `qa-falsification`, `qa-a11y`, `qa-visual`, `design`, `commit`, `planner`, `research`. Regex parser: `(?m)^Role:\s*([a-z-]+)\s*$`. `Role` is **optional** on the domain `ActionItem` (empty-string is valid; only non-empty values are validated against the closed enum). No hydration runner, no `till migrate` subcommand, no SQL backfill — see Pre-MVP rules.

Five droplets. **2.2** (pure parser + Role enum) is independent. **2.3** (domain `ActionItem.Role` + validation) depends on 2.2 (uses the same `Role` type). **2.4** (SQLite schema column + scanner + write paths) depends on 2.3 (needs the field on the struct). **2.5** (MCP request/response + app-service plumbing) depends on 2.3 + 2.4. **2.6** (snapshot field) depends on 2.3.

Same-package-blocking constraints inside Unit A: 2.2 + 2.3 share `internal/domain` → 2.3 blocked-by 2.2. 2.6 also touches `internal/app/snapshot.go` (package `internal/app`); 2.5 also touches `internal/adapters/server/common`. 2.4 touches `internal/adapters/storage/sqlite`. Disjoint packages between 2.4 / 2.5 / 2.6 — they can theoretically build in parallel after 2.3, but for testing-clarity they serialize 2.4 → 2.5 → 2.6.

#### Droplet 2.2 — Pure `Role` enum + `ParseRoleFromDescription` parser in `internal/domain`

- **State:** todo
- **Paths:** `internal/domain/role.go` (new), `internal/domain/role_test.go` (new), `internal/domain/errors.go` (add `ErrInvalidRole = errors.New("invalid role")`)
- **Packages:** `internal/domain`
- **Acceptance:**
  - New `Role` string type with 9 typed constants (`RoleBuilder`, `RoleQAProof`, `RoleQAFalsification`, `RoleQAA11y`, `RoleQAVisual`, `RoleDesign`, `RoleCommit`, `RolePlanner`, `RoleResearch`).
  - `IsValidRole(r Role) bool` returns true only for those 9 values; empty string is **invalid** at this validator level (caller decides whether to permit empty).
  - `NormalizeRole(r Role) Role` lowercases + trims; returns empty for empty input.
  - `ParseRoleFromDescription(desc string) (Role, error)` applies regex `(?m)^Role:\s*([a-z-]+)\s*$`, returns the **first** matching `Role` if its value is one of the 9 closed values; returns `("", nil)` if no `Role:` line is found; returns `("", ErrInvalidRole)` if a `Role:` line is present but its value is not in the closed enum.
  - `internal/domain/errors.go` adds `ErrInvalidRole = errors.New("invalid role")` to the existing var-block (same style as `ErrInvalidKind`).
  - Table-driven tests cover: each of 9 valid values; empty desc; desc with no `Role:` line; multiline desc with `Role:` mid-paragraph (regex anchors require start-of-line); two `Role:` lines (asserts first wins); whitespace variants (`Role:  builder  ` → `RoleBuilder`); unknown value (`Role: foobar` → `ErrInvalidRole`); case sensitivity (`Role: Builder` should fail since the regex captures `[a-z-]+`); `Role: qa-proof` round-trip.
  - `mage test-pkg ./internal/domain` green.
  - DB action: NONE.
- **Blocked by:** —

#### Droplet 2.3 — Add `Role` field to `ActionItem` + `ActionItemInput` + `NewActionItem` validation

- **State:** todo
- **Paths:** `internal/domain/action_item.go` (add `Role Role` field to both structs; add validation block in `NewActionItem`), `internal/domain/domain_test.go` (extend existing `NewActionItem` table-driven tests — confirmed via Read: no `internal/domain/action_item_test.go` file exists today; tests live in `domain_test.go`)
- **Packages:** `internal/domain`
- **Acceptance:**
  - `ActionItem` struct gains `Role Role` field (zero-value empty string allowed).
  - `ActionItemInput` struct gains `Role Role` field.
  - `NewActionItem` normalizes `in.Role` via `NormalizeRole`; if non-empty, calls `IsValidRole`; on failure returns `ErrInvalidRole`. Empty role is permitted (returns the zero-value Role on the constructed `ActionItem`).
  - Table-driven test additions: empty role round-trips empty; each of 9 valid roles round-trips; unknown role rejected with `ErrInvalidRole`; whitespace-only role normalizes to empty.
  - All existing `domain_test.go` tests remain green (no regressions on the 12-value `Kind` validation path).
  - `mage test-pkg ./internal/domain` green.
  - DB action: NONE.
- **Blocked by:** 2.2

#### Droplet 2.4 — SQLite `action_items.role` column + scanner + insert/update paths

- **State:** todo
- **Paths:** `internal/adapters/storage/sqlite/repo.go` (add `role TEXT NOT NULL DEFAULT ''` to the `CREATE TABLE IF NOT EXISTS action_items` block at `:168`; add `role` to `scanActionItem` at `:2738`; add `role` to insert + update SQL inside the action-item write paths), `internal/adapters/storage/sqlite/repo_test.go` (extend round-trip test to set + read a `Role` value)
- **Packages:** `internal/adapters/storage/sqlite`
- **Acceptance:**
  - New column `role TEXT NOT NULL DEFAULT ''` appears in the `action_items` `CREATE TABLE` statement at `:168`.
  - `scanActionItem` reads the new column into `domain.ActionItem.Role`.
  - Insert + update SQL include the `role` column. Existing tests with empty `Role` still pass (empty-string default).
  - One new test in `repo_test.go` writes `domain.RoleBuilder`, reads it back, asserts equality.
  - **Pre-MVP rule honored:** no `ALTER TABLE` migration, no SQL backfill — dev fresh-DBs. The schema-creation block is the only schema source.
  - `mage test-pkg ./internal/adapters/storage/sqlite` green.
  - **DB action:** DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci` for this droplet (schema change).
- **Blocked by:** 2.3

#### Droplet 2.5 — MCP `role` field on action-item create/update/get + app-service plumbing

- **State:** todo
- **Paths:** `internal/adapters/server/common/mcp_surface.go` (add `Role string` to `CreateActionItemRequest` + `UpdateActionItemRequest` request structs and to the response shape), `internal/adapters/server/common/app_service_adapter_mcp.go` (thread `Role` through `CreateActionItem` at `:620` and `UpdateActionItem` at `:661` into the underlying `app.CreateActionItemInput` / `app.UpdateActionItemInput`), `internal/adapters/server/mcpapi/extended_tools.go` (add `mcp.WithString("role", mcp.Description("optional role tag — see allowed values"))` to the `till.action_item` tool's create + update operation schemas; thread the parsed value into the `Create/Update` request), `internal/adapters/server/mcpapi/extended_tools_test.go` (add a test case asserting role round-trip through MCP)
- **Packages:** `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
- **Acceptance:**
  - `till.action_item(operation=create, role=...)` accepts and persists the role; reading via `operation=get` returns it.
  - `till.action_item(operation=update, role=...)` updates the role on an existing action item.
  - Empty role is accepted on create and update (no-op for update).
  - Invalid role returns a 400-class MCP error (carries `ErrInvalidRole` semantics — match the existing pattern for kind-invalid errors).
  - Test in `extended_tools_test.go` covers: create with valid role, create without role, update role, get returns role, create with invalid role rejects.
  - `mage test-pkg ./internal/adapters/server/common` and `mage test-pkg ./internal/adapters/server/mcpapi` both green.
  - DB action: NONE (data-shape change rides on 2.4's schema change).
- **Blocked by:** 2.4

#### Droplet 2.6 — Snapshot serialization for `Role`

- **State:** todo
- **Paths:** `internal/app/snapshot.go` (add `Role domain.Role \`json:"role,omitempty"\`` to `SnapshotActionItem` struct at `:57`; thread the field through `snapshotActionItemFromDomain` at `:1057` and `(t SnapshotActionItem) toDomain()` at `:1263`), `internal/app/snapshot_test.go` if it exists, otherwise extend whichever test exercises `SnapshotActionItem` round-trip
- **Packages:** `internal/app`
- **Acceptance:**
  - Snapshot round-trip preserves a non-empty `Role` value across all 9 valid roles (table-driven).
  - Snapshot with empty role round-trips empty (omitempty drops the JSON key on serialize).
  - JSON shape: `{"role":"builder"}` when set, key absent when empty.
  - **No `SnapshotVersion` bump required** — field uses `omitempty` and `encoding/json` ignores unknown keys by default. Old `v5` snapshots load forward-compatibly.
  - `mage test-pkg ./internal/app` green.
  - DB action: NONE.
- **Blocked by:** 2.3

---

### Unit B — State-vocabulary rename `done → complete`, `progress → in_progress` (single atomic droplet)

Bundled in one droplet per Round 2 dev decision (see `## Notes` → "Round 2 revision summary"). After Unit B-zero deleted `templates/builtin/*.json`, the rename only touches Go code + Go tests + checklist JSON keys (no template JSON sweep). The rename is intrusive enough that splitting risks per-droplet `mage ci` red between commits — every consumer file MUST flip in one commit.

**Strict-canonical only (dev decision, 2026-05-01).** No alias tolerance. The rename is a hard cutover — only canonical values (`complete`, `in_progress`) are accepted on every code path (constants, alias normalizers, JSON unmarshal, MCP coercion, CLI input, config). Legacy values (`done`, `progress`, `completed`, `doing`, `in-progress`) are REJECTED, not coerced. Pre-MVP every caller is the dev; broken callers fail loud and get fixed at the source. Rationale: pre-MVP cleanliness — no lingering legacy-tolerance trash that needs a separate cleanup drop later.

Cross-unit ordering: 2.7 is `Blocked by: 2.6` to honor PLAN.md § 19.2's explicit "role promotion → state rename" ordering.

#### Droplet 2.7 — State-vocabulary rename across the whole tree (atomic)

- **State:** todo
- **Paths:** the rename touches every state-machine site in the tree. Enumerated below by package; every file is in one commit. **All file:line cites verified at HEAD via `git grep` for this Round 2 revision.**

  **`internal/domain/` (state-machine truth source + struct fields):**
  - `internal/domain/workitem.go` — rename `StateDone` → `StateComplete` and `StateProgress` → `StateInProgress` constants at `:18-19`; rewrite normalization at `:147-163` so ONLY canonical values (`"complete"`, `"in_progress"`) are accepted — every legacy value (`"done"`, `"completed"`, `"progress"`, `"doing"`, `"in-progress"`) returns the unknown-state error path; flip `IsTerminalState` at `:174` to test against `StateComplete`/`StateFailed`; rename `ChecklistItem.Done bool` → `ChecklistItem.Complete bool` at `:81-85` including the JSON tag `\`json:"complete"\``; rewrite `isValidLifecycleState` (unexported) at `:166` to enumerate the canonical values; rename `CompletionPolicy.RequireChildrenDone bool` at `:89` → `CompletionPolicy.RequireChildrenComplete bool` including the JSON tag `\`json:"require_children_done"\`` → `\`json:"require_children_complete"\``.
  - `internal/domain/action_item.go` — rename `StateProgress`/`StateDone` symbol references at `:268, 275, 278, 315`; rename `item.Done` field access at `:357` to `item.Complete` (production code); rename `policy.RequireChildrenDone` reader at `:310` to `policy.RequireChildrenComplete`.
  - `internal/domain/domain_test.go` — rename test references to renamed constants and `ChecklistItem.Done` fields throughout (e.g. `:275, 324, 327, 330, 333, 374, 393, 396, 420-442, 536, 561-566`); rename `RequireChildrenDone:` test fixtures at `:430, 566, 614-615` to `RequireChildrenComplete:`. **NOT in scope:** `:114` references column name `"done"` as a free-form column-rename test input, NOT a lifecycle state literal — leave unchanged.
  - `internal/domain/kind_capability_test.go` — rename `Done: false` `ChecklistItem` field literal at `:19` to `Complete: false`; rename `RequireChildrenDone:` test fixtures at `:35` AND `:73` to `RequireChildrenComplete:`. **Without `:19` in scope, `mage test-pkg ./internal/domain` compile-fails after the field rename.**
  - `internal/app/kind_capability_test.go` — rename `Done: false` test fixture at `:429` (and any other `ChecklistItem.Done` field references in this file) to `Complete: false`. **Without this file in scope, `mage test-pkg ./internal/app` compile-fails after the field rename.**

  **`internal/app/` (transition rules, snapshot validation, attention overview, default state seed):**
  - `internal/app/service.go` — rename `domain.StateDone`/`StateProgress` symbol references at `:623, 627, 639, 644, 1817, 1965-1975`; flip `defaultStateTemplates()` at `:1873-1881` so the seed ID column emits `"in_progress"` and `"complete"` (NOT `"progress"`/`"done"`); rewrite `normalizeStateID` at `:1922-1955` to accept ONLY canonical inputs (every legacy alias case at `:1948-1951` removed); rewrite `lifecycleStateForColumnID` at `:1958-1979` to switch on canonical column names only. **NOT in scope:** `:556` and `:694` reference `domain.StateTodo` (unchanged by Drop 2) — leave unchanged.
  - `internal/app/service_test.go` — sweep all symbol references and state-literal test inputs (verified hits at `:1561, 1567, 2467, 2953, 3035, 3055, 3065, 3092, 3108, 3186, 3196, 4573, 4609, 4626, 4693`; full grep sweep required); rewrite `States: []string{"progress"}` and `StateID == "progress"` legacy state literals at `:1561, 1567, 2953` to canonical (`"in_progress"`); rename `Done: true/false` checklist literals at `:3003, 3038, 3040, 4612` to `Complete:`; rename `RequireChildrenDone:` test fixtures at `:3042, 3095, 4613` to `RequireChildrenComplete:`. **NOT in scope:** `:3797` is `Reason: "done"` on a capability-lease revoke (free-form text), NOT a state literal — leave unchanged. **`fakeRepo` extension for new `Repository.ListActionItemsByParent` method belongs in Droplet 2.10, NOT 2.7.**
  - `internal/app/snapshot.go` — rename symbols in the validation switch at `:419` and any other site (e.g., `:1267` fallback uses `StateTodo`, unchanged); flip the doc comment that names `domain.AllowedParentKinds` at `:448` is OUT OF SCOPE for 2.7 — it lives under Droplet 2.9 (Unit C, 2.9 in the Round 2 renumbering) — but `:419` IS in 2.7.
  - `internal/app/snapshot_test.go` — sweep any state-symbol references touched by the rename.
  - `internal/app/attention_capture.go` — rename `domain.StateProgress`/`StateDone` references at `:350, 353, 356, 371`. Field renames `InProgressItems` → consider rename to `InProgressItems` (already canonical in field name; only the JSON tag at `:95` `json:"in_progress_items"` is canonical-friendly already). `DoneItems` field at `:96` (`json:"done_items"`) → rename field to `CompleteItems` and JSON tag to `json:"complete_items"`. Update increments at `:351, 354`. (See `## Notes` → "Aggregate counter rename" for rationale.)
  - `internal/app/attention_capture_test.go` — sweep references at `:272, 377-378, 386-390` (rename `DoneItems` → `CompleteItems`).

  **`internal/adapters/server/common/` (capture state, MCP coercion, lifecycle adapter, types):**
  - `internal/adapters/server/common/capture.go` — rename symbols at `:258, 260, 302, 304`; **rewrite `canonicalLifecycleState` at `:296-312` to accept ONLY canonical inputs** (this is the SECOND coercion site Round 1 missed; verified at HEAD `:296-312`); rename increments `overview.InProgressActionItems++` at `:259` (canonical-friendly) and `overview.DoneActionItems++` at `:261` → rename field to `CompleteActionItems` (see types.go below).
  - `internal/adapters/server/common/capture_test.go` — rename symbol references at `:111, 136, 268-269`; rewrite `canonicalLifecycleState("doing")` test at `:268` to verify rejection (no longer coercion to `StateProgress`); rename `Done:` checklist field at `:123` to `Complete:`; rename `RequireChildrenDone:` test fixture at `:126` to `RequireChildrenComplete:`; rename `WorkOverview.DoneActionItems` assertion at `:198` to `CompleteActionItems`.
  - `internal/adapters/server/common/app_service_adapter.go` — rename `summary.WorkOverview.DoneItems` → `CompleteItems` at `:409`; rename struct-field assignment `DoneActionItems:` → `CompleteActionItems:` at `:421` (target field name in `types.go`).
  - `internal/adapters/server/common/app_service_adapter_test.go` — rename `DoneItems` field literals at `:39, 95`.
  - `internal/adapters/server/common/app_service_adapter_lifecycle_test.go` — rewrite `State: "done"` test input at `:180` to `State: "complete"` (state-machine input — strict-canonical applies); leave `Reason: "done"` / `RevokedReason: "done"` at `:716, 721, 912` unchanged (capability-lease revoke reason — incidental free-form string, not a state literal). Rename `domain.StateDone` symbol references at `:189-190`.
  - `internal/adapters/server/common/app_service_adapter_mcp.go` — rename symbols at `:820, 854, 856`; rewrite `actionItemLifecycleStateForColumnName` at `:849-864` to accept ONLY canonical column-names (`"complete"` → `domain.StateComplete`, `"in_progress"` → `domain.StateInProgress`); rewrite `normalizeStateLikeID` at `:866-901` (legacy alias cases at `:892, 894-895`) to accept ONLY canonical inputs and reject every legacy form with a clear error.
  - `internal/adapters/server/common/types.go` — rename struct field `InProgressActionItems` (already canonical) at `:142` and JSON tag `json:"in_progress_tasks"` (already canonical); rename `DoneActionItems` at `:143` → `CompleteActionItems` and JSON tag `json:"done_tasks"` → `json:"complete_tasks"`.

  **`internal/adapters/server/mcpapi/`:**
  - `internal/adapters/server/mcpapi/extended_tools.go` — rewrite the MCP tool-description string at `:1339` (`"Lifecycle state target for operation=move_state (for example: todo|in_progress|done)"`) to use canonical values only (`"... todo|in_progress|complete"`). LLM agents read tool descriptions as canonical examples; leaving the legacy literal contradicts strict-canonical at the external-API contract surface.
  - `internal/adapters/server/mcpapi/extended_tools_test.go` — rename `domain.StateProgress`/`StateDone` symbol references at `:427, 446`; rewrite `"state": "done"` test inputs at `:1114, 2587` to `"state": "complete"`; rewrite `service.lastMoveActionItemStateReq.State` assertion at `:2600` from `"done"` to `"complete"`.

  **`internal/tui/` (state literals, label maps, default-state lists, all surfaces):**
  - `internal/tui/model.go` — rename `domain.StateDone`/`StateProgress` symbol references at `:12249, 14052, 14058, 17188, 17199, 17370, 17641, 17977, 17979, 18017, 18019, 18066, 18074, 18171-18172, 18210`; update `canonicalSearchStatesOrdered` at `:305` from `["todo", "progress", "done", "archived"]` to `["todo", "in_progress", "complete", "archived"]`; update `canonicalSearchStateLabels` at `:317-318` and the `"done"`/`"progress"` keys at `:321` plus the lookups at `:18016, 18018, 18020`; update `searchStates`/`searchDefaultStates`/`dependencyStates` literal lists at `:1231-1236`; update label-map switch cases at `:13692, 14151, 17978`; rewrite `normalizeColumnStateID` at `:17934-17967` (strict-canonical only — legacy alias case at `:17960-17963` removed); rewrite `lifecycleStateForColumnName` at `:17971-17985` and `lifecycleStateLabel` at `:18012-18029` strict-canonical; update `item.Done` access at `:17742` → `item.Complete`.
  - `internal/tui/model_test.go` — sweep all references at `:627, 685-686, 967-970, 4549, 5734-5740, 11463-11472, 13004, 13186, 13247, 13267, 13541-13549`; rewrite `"in-progress", "progress", "doing"` test cases at `:685, 967` and `"done", "complete", "completed"` test case at `:969` so they verify rejection (no longer coercion).
  - `internal/tui/options.go` — update default-state list at `:147` from `["todo", "progress", "done"]` to `["todo", "in_progress", "complete"]`.
  - `internal/tui/thread_mode.go` — rename `domain.StateDone` symbol reference at `:151` (Round 1 plan missed this file; Round 2 includes it).

  **`internal/config/` (search states + validator):**
  - `internal/config/config.go` — flip `Search.States` default at `:218` from `["todo", "progress", "done"]` to `["todo", "in_progress", "complete"]`; flip fallback at `:550`; rewrite `isKnownLifecycleState` at `:1092-1094` from `["todo", "progress", "done", "failed", "archived"]` to `["todo", "in_progress", "complete", "failed", "archived"]` — strict-canonical, no legacy alias tolerance for "external config compatibility."
  - `internal/config/config_test.go` — rewrite `states = ["todo", "progress", "archived"]` TOML fixture at `:326` to canonical (`"in_progress"`); rewrite `isKnownLifecycleState` table-driven assertions at `:811-820` to verify strict-canonical (only canonical values return true; `"progress"`/`"done"` now return false).

  **Out of `Paths:` — verified no state-machine touches:** `cmd/till/main.go` (no `StateDone`/`StateProgress` symbol or legacy state literal — `cfg.Search.States` is just a slice copy at `:3262`), `cmd/till/embeddings_cli.go` (`status = "completed"` at `:242` is embedding-job status, unrelated to lifecycle state), `internal/adapters/storage/sqlite/repo.go` (only `domain.StateTodo` reference at `:2819`, unchanged). `internal/adapters/server/common/app_service_adapter_outcome_test.go:71` (`Outcome: "done"` is a sample bad value for the `outcome` validator, NOT a lifecycle state — test passes unchanged after rename because it asserts rejection). `internal/adapters/server/mcpapi/handler_integration_test.go:380, 405` (`resolution_note: "done"` is free-form text, not a state). `internal/adapters/storage/sqlite/repo_test.go:1749`, `internal/app/capability_inventory_test.go:50`, `internal/domain/comment_test.go:95`, `internal/domain/handoff_test.go:64` (all `Reason`/`Summary`/body-text free-form `"done"`, not state literals). `internal/app/service_test.go:2467` `{ID: "doing", Name: "Doing", Position: 1}` is a test fixture verifying `normalizeStateID` legacy-alias coercion — under strict-canonical the test must be rewritten to assert rejection, so it IS in scope (covered above under `service_test.go`).

- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite` (test only — covered by 2.4's earlier touch but state-vocab leaves it untouched), `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/tui`, `internal/config`
- **Acceptance:**
  - **Symbol-grep checks (whole-tree, scope-aware):**
    - `git grep -nE "\\bStateDone\\b" -- '*.go'` returns empty.
    - `git grep -nE "\\bStateProgress\\b" -- '*.go'` returns empty.
    - `git grep -nE "\\bStateComplete\\b" -- '*.go'` returns non-empty (the new canonical symbol is present).
    - `git grep -nE "\\bStateInProgress\\b" -- '*.go'` returns non-empty.
    - `git grep -nE "\\bChecklistItem\\b.*\\bDone\\b" -- '*.go'` returns empty (struct field renamed).
    - `git grep -nE "\\.Done\\b" -- 'internal/domain/' 'internal/tui/' 'internal/app/' 'internal/adapters/'` shows zero ChecklistItem.Done access remaining (free-form `ctx.Done()`, `wg.Done()`, `start.Done()` matches are unchanged stdlib/concurrency idioms — these are NOT state-vocab and are explicitly allowed; verify the only `.Done` hits remaining are stdlib concurrency calls, not domain field access).
    - `git grep -nE "\\bDoneItems\\b|\\bDoneActionItems\\b" -- '*.go'` returns empty (aggregate counter fields renamed to `CompleteItems`/`CompleteActionItems`).
    - `git grep -nE "\\bRequireChildrenDone\\b" -- '*.go'` returns empty (field renamed to `RequireChildrenComplete`).
    - `git grep -nE 'json:"require_children_done"' -- '*.go'` returns empty (JSON tag renamed to `require_children_complete`).
    - `git grep -nE '\\.Done\\s*=\\s*(true|false)|Done:\\s*(true|false)' -- '*.go'` returns only stdlib concurrency idioms (`ctx.Done()`, `wg.Done()`) — zero `ChecklistItem.Done` field-literal sites remain.
  - **State-machine literal checks (scope-narrowed to state-machine contexts, NOT broad string sweeps):**
    - `git grep -nE 'domain\\.StateDone|domain\\.StateProgress' -- '*.go'` returns empty.
    - `git grep -nE 'lifecycle_state.*"done"|lifecycle_state.*"progress"' -- '*.go'` returns empty.
    - `git grep -nE 'LifecycleState.*"done"|LifecycleState.*"progress"' -- '*.go'` returns empty.
    - `git grep -nE '"in-progress"|"doing"' -- 'internal/domain/' 'internal/app/' 'internal/adapters/server/' 'internal/tui/' 'internal/config/'` returns empty (legacy aliases gone from every state-machine file).
    - In `internal/domain/workitem.go`, `internal/app/service.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/common/capture.go`, `internal/tui/model.go`, `internal/config/config.go`, `git grep -E '"done"|"progress"|"completed"' <file>` returns empty (the state-machine source files have zero legacy literals).
    - `git grep -nE 'json:"done"|json:"progress"|json:"completed"|json:"in-progress"|json:"doing"' -- '*.go'` returns empty (no JSON tags carry legacy state vocab).
    - `git grep -nE 'json:"done_tasks"|json:"done_items"' -- '*.go'` returns empty (aggregate counter JSON tags renamed to `complete_tasks`/`complete_items`).
  - **Behavior checks (test-driven):**
    - `IsTerminalState(StateComplete)` returns true; `IsTerminalState(StateFailed)` returns true; all other states return false.
    - **Strict-canonical normalization (every site):** input `"complete"` → `StateComplete`; input `"in_progress"` → `StateInProgress`. Input `"done"`, `"completed"`, `"progress"`, `"in-progress"`, `"doing"` returns the unknown-state error path (NOT coerced). This applies to `domain.NormalizeLifecycleState`, `app.normalizeStateID`, `app.lifecycleStateForColumnID`, `common.canonicalLifecycleState`, `common.actionItemLifecycleStateForColumnName`, `common.normalizeStateLikeID`, `tui.normalizeColumnStateID`, `tui.lifecycleStateForColumnName`, `config.isKnownLifecycleState`.
    - `defaultStateTemplates()` returns `[{ID: "todo"}, {ID: "in_progress"}, {ID: "complete"}, {ID: "failed"}]` (canonical IDs).
    - `ChecklistItem` JSON marshal emits `"complete":` not `"done":`. JSON unmarshal accepts ONLY `"complete"` — `"done"` keys produce a decode error (no fallback alias). Table-driven test asserts both directions: canonical accepted, legacy rejected.
    - `WorkOverview` JSON emits `complete_tasks` (was `done_tasks`); `AttentionWorkOverview` JSON emits `complete_items` (was `done_items`).
    - The `till.action_item` MCP create/list/move-state tool accepts ONLY canonical state values (`"complete"`, `"in_progress"`, `"todo"`, `"archived"`); legacy values produce a clear error response. Canonical values round-trip through reads.
  - **Whole-tree CI gate:** `mage ci` green (the unified CI run is the unit-boundary gate — any missed file produces a compile error or test failure here).
  - **DB action:** DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci` for this droplet (state-vocab change — column IDs change from `progress`/`done` to `in_progress`/`complete`).
  - **Snapshot version:** `internal/app/snapshot.go:16 SnapshotVersion = "tillsyn.snapshot.v5"` STAYS at `v5` despite the JSON-tag-breaking changes in this droplet (`json:"done" → json:"complete"`, `json:"done_tasks" → json:"complete_tasks"`, `json:"done_items" → json:"complete_items"`, `json:"require_children_done" → json:"require_children_complete"`). Pre-MVP rule per `feedback_no_migration_logic_pre_mvp.md`: dev fresh-DBs and fresh-snapshots between schema-vocabulary changes; no migration code, no decoder aliases, no version bump. Post-MVP, snapshot version becomes load-bearing on every breaking change.
- **Blocked by:** 2.5, 2.6  *(2.5 added Round 3: 2.5 + 2.7 both edit `internal/adapters/server/common/app_service_adapter_mcp.go` — same-package serialization required to prevent cross-unit compile race once Unit B fans out from Unit A's tail)*

---

### Unit C — Strip hardwired nesting defaults from domain catalog (mechanism stays)

Two droplets. Independent of Unit B per PLAN.md § 19.2 ("orthogonal, can run in parallel with the rename if convenient"); ordered after Unit B here only because B leaves no Go-tree fallout for C to step on. **2.8** flips the boot-seed payloads (the live production data path). **2.9** deletes the speculative `AllowedParentKinds` function (the dead-code path).

Same-package-blocking: 2.8 + 2.9 are in different packages (`internal/adapters/storage/sqlite` vs `internal/domain`), so independent. 2.9 also touches a doc-comment in `internal/app/snapshot.go:448` and `internal/adapters/storage/sqlite/repo.go:300` — still no compile-overlap conflict with 2.8 because comment edits do not race compile units, and 2.8 owns its own edit window on `repo.go` (boot-seed payloads at `:304-375`).

#### Droplet 2.8 — Empty `AllowedParentScopes` for every kind in boot-seed

- **State:** todo
- **Paths:** `internal/adapters/storage/sqlite/repo.go` (change every `INSERT OR IGNORE INTO kind_catalog` row at `:304-375` so `allowed_parent_scopes_json` is `'[]'` instead of `'["plan"]'` or `'["build"]'` — 12 rows total: `plan`, `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`), `internal/adapters/storage/sqlite/repo_test.go` and `internal/app/kind_capability_test.go` and `internal/domain/kind_capability_test.go` (update any test that asserted the old `["plan"]`/`["build"]` defaults; assert universal-allow behavior of `AllowsParentScope` per `internal/domain/kind.go:224-236` with the empty-list early return at `:227-229`)
- **Packages:** `internal/adapters/storage/sqlite`, `internal/app`, `internal/domain` (test-only edit)
- **Acceptance:**
  - All 12 `INSERT OR IGNORE INTO kind_catalog` payloads carry `allowed_parent_scopes_json = '[]'`.
  - **Untouched fields:** `applies_to_json` and every other column on each row remain unchanged. ONLY `allowed_parent_scopes_json` flips to `'[]'`.
  - `KindDefinition.AllowsParentScope(any-scope)` returns `true` for every seeded kind (universal-allow), per the existing `internal/domain/kind.go:224-236` early-return at `:227-229` on empty `AllowedParentScopes`.
  - The `AllowsParentScope` enforcement path at `internal/app/kind_capability.go:566` is unchanged — only the input data shape changed.
  - **Pre-MVP rule honored:** no DB UPDATE script for any existing rows' `allowed_parent_scopes_json`; dev fresh-DBs.
  - `mage test-pkg ./internal/adapters/storage/sqlite` and `mage test-pkg ./internal/app` and `mage test-pkg ./internal/domain` all green.
  - **DB action:** DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci` for this droplet (boot-seed data change).
- **Blocked by:** 2.7

#### Droplet 2.9 — Delete `domain.AllowedParentKinds` function + test fixture + doc-comment cleanup

- **State:** todo
- **Paths:** `internal/domain/kind.go` (delete `AllowedParentKinds` function at `:94-117`), `internal/domain/domain_test.go` (delete `TestAllowedParentKindsEncodesHierarchy` at `:680-714`), `internal/app/snapshot.go` (update doc comment at `:448` referencing `domain.AllowedParentKinds` — replace with reference to `KindDefinition.AllowedParentScopes` + `AllowsParentScope`), `internal/adapters/storage/sqlite/repo.go` (update doc comment at `:300` referencing `domain.AllowedParentKinds` — same replacement)
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`
- **Acceptance:**
  - `git grep "AllowedParentKinds"` returns empty across the whole repo (function deleted, test deleted, both doc comments updated).
  - `internal/app/kind_capability.go:566` enforcement path still compiles and tests still pass — that path uses `AllowedParentScopes` + `AllowsParentScope`, NOT the deleted `AllowedParentKinds`.
  - `mage ci` green (unit boundary — confirms whole-tree no orphans).
  - DB action: NONE (code-deletion, no schema or data shape change).
- **Blocked by:** 2.8

---

### Unit D — Dotted-address fast-nav reads (CLI + MCP read paths)

Pure resolver, single package. Two droplets. Lands last per PLAN.md § 19.2 ordering ("zero coupling, lands last so rename churn settles before resolver tests"). TUI bindings deferred to Drop 4.5 per PLAN.md § 19.2 explicit out-of-scope.

**Decision: resolver lives in `internal/app`** (not `internal/domain`). Rationale: resolution requires a project-context lookup against the action-items repo, which is an application-service concern, not a pure-domain concern. Keep the domain layer free of repo dependencies.

**Repository interface decision (Round 2):** `Repository` (`internal/app/ports.go:11-53`) exposes `GetActionItem(ctx, id)` and `ListActionItems(ctx, projectID, includeArchived)` — but no list-children-by-parent operation. Resolving `N.M.K` requires walking the parent→child tree level by level. With only `ListActionItems`, the resolver would pull every action item in the project and filter by `ParentID` in memory — O(depth × N) per resolve. **Add `ListActionItemsByParent(ctx context.Context, projectID, parentID string) ([]domain.ActionItem, error)` to `Repository`** with a SQLite-side implementation, alongside the resolver. This is a clean-interface decision over an O(N) walk; the new method is a consumer-side requirement of the resolver, justified by the use-case.

Same-package-blocking: 2.10 owns the resolver in `internal/app` + the `Repository` interface change in `internal/app/ports.go` + the SQLite-side implementation in `internal/adapters/storage/sqlite`. 2.11 wires CLI + MCP — disjoint packages from 2.10's resolver (`cmd/till` + `internal/adapters/server/...`), so no compile race; serialize anyway because 2.11 calls into the resolver 2.10 created.

#### Droplet 2.10 — Pure dotted-address resolver in `internal/app` + `Repository.ListActionItemsByParent`

- **State:** todo
- **Paths:** `internal/app/dotted_address.go` (new — function `ResolveDottedAddress(ctx, repo, projectID, dotted string) (actionItemID string, err error)` with sentinel errors `ErrDottedAddressNotFound`, `ErrDottedAddressInvalidSyntax`), `internal/app/dotted_address_test.go` (new — table-driven tests using an in-memory fake or the existing test SQLite fixture), `internal/app/ports.go` (add `ListActionItemsByParent(ctx context.Context, projectID, parentID string) ([]domain.ActionItem, error)` to the `Repository` interface), `internal/app/service_test.go` (extend `fakeRepo` to implement the new method — without this, `mage test-pkg ./internal/app` compile-fails on every test that constructs `fakeRepo`), `internal/adapters/storage/sqlite/repo.go` (add `ListActionItemsByParent` method on `*Repository` alongside existing `ListActionItems` at `:1393`), `internal/adapters/storage/sqlite/repo_test.go` (round-trip test for the new method)
- **Packages:** `internal/app`, `internal/adapters/storage/sqlite`
- **Acceptance:**
  - Function signature: `func ResolveDottedAddress(ctx context.Context, repo Repository, projectID string, dotted string) (string, error)` — `Repository` is the existing app-layer interface, now extended with `ListActionItemsByParent`. **`projectID` is supplied by the caller** (CLI `--project <slug>` flag, slug-prefix shorthand, or MCP session inference); the resolver does NOT parse a project component out of the dotted body.
  - **Dotted body is project-LESS and 0-indexed at every level.** Form: `<lvl1_pos>.<lvl2_pos>.<lvl3_pos>...` — `0` = first child of project (level_1 position 0), `0.0` = first grandchild, `2.5` = level_1 position 2 then level_2 position 5. **Project NEVER appears as `0` in the body.** Body regex: `^\d+(\.\d+)*$`. (Names are not in play — addresses are positional only; renaming a drop does not change its position-based address.)
  - **CLI shorthand:** `<proj_slug>:<dotted>` (slug-prefix-colon, e.g. `tillsyn:1.5.2`) accepted as a CLI ergonomic; the resolver parses the slug, looks up `projectID` for the slug, then resolves the body. CLI also accepts `--project <slug>` flag with bare body. Slug verified against the supplied/inferred `projectID`.
  - `Repository.ListActionItemsByParent(ctx, projectID, parentID)` returns the list of action items whose `ParentID == parentID` within `projectID`, **deterministically ordered by `created_at ASC, id ASC`** — UUID `id` is a globally unique tie-breaker, so the listing is total. Empty `parentID` returns level-1 children (no parent). Position N in this ordering IS the dotted index. **NOT to be confused with the existing column-scoped `position` field** at `internal/adapters/storage/sqlite/repo.go` (Kanban-column ordering keyed by `(project_id, column_id, position)` per index `idx_action_items_project_column_position`) — that field is TUI-arrangement-only and would shift dotted addresses under user drag-reordering. The resolver MUST use `created_at ASC, id ASC` exclusively. If a future drop adds a per-parent `tree_position` column, that drop revisits the resolver.
  - SQLite implementation uses an indexed query (`WHERE project_id = ? AND parent_id = ? ORDER BY ...`); no per-call full-table scan.
  - Returns the resolved action-item UUID on match.
  - Returns `ErrDottedAddressNotFound` when any level's index is out-of-range for the listing at that level.
  - Returns `ErrDottedAddressInvalidSyntax` when the input fails the body shape check (or slug-prefix shape check).
  - **No `ErrDottedAddressAmbiguous` error** — by construction the deterministic ORDER BY + UUID tie-breaker yields a unique item per index, so ambiguity is unreachable.
  - Table-driven tests cover: valid `0` (single-level), valid `0.0` (two-level), valid `2.5.1` (three-level), slug-prefixed valid (`tillsyn:1.5.2`), slug-prefix mismatch (slug doesn't match `projectID`), out-of-range path at each level, malformed inputs (empty, `1.`, `.1`, `1..2`, `abc`, leading-dash, deep nesting), leading-zero accepted (`007` → 7 per `strconv.Atoi`), UUID input rejected (the resolver expects the dotted form; UUID-vs-dotted detection is a caller-side concern in 2.11).
  - `mage test-pkg ./internal/app` and `mage test-pkg ./internal/adapters/storage/sqlite` both green.
  - DB action: NONE (additive method, no schema change).
- **Blocked by:** 2.9

#### Droplet 2.11 — Wire resolver into CLI + MCP read paths; mutation paths reject dotted form

- **State:** todo
- **Paths:** `internal/adapters/server/common/app_service_adapter_mcp.go` (in `till.action_item(operation=get)`, accept either UUID or dotted form for `action_item_id` — when input doesn't parse as UUID, call `ResolveDottedAddress`; mutation operations `create|update|move|move_state|delete|restore|reparent` reject dotted form with a clear error), `internal/adapters/server/mcpapi/extended_tools.go` (mirror the get-vs-mutate distinction in tool-level argument validation), `cmd/till/main.go` (CLI read commands accept dotted form via the same resolver; CLI mutation commands reject dotted form), test files for each path
- **Packages:** `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `cmd/till`
- **Acceptance:**
  - `till.action_item(operation=get, action_item_id="2.1")` resolves through the resolver — project inferred from MCP auth-gated session — and returns the matching action item.
  - `till.action_item(operation=get, action_item_id="<UUID>")` continues to work unchanged. UUID-vs-dotted detection: caller checks for UUID shape (e.g. `regexp.MustCompile`-based UUID check) and routes to direct `GetActionItem` if UUID, else `ResolveDottedAddress`.
  - `till.action_item(operation=update, action_item_id="2.1", ...)` returns a 400-class error explaining that mutations require UUIDs.
  - **CLI `till action_item get` accepts both forms:** explicit `--project <slug>` flag with bare dotted body (`till action_item get --project tillsyn 1.5.2`), AND slug-prefix shorthand (`till action_item get tillsyn:1.5.2`). Bare dotted form without project (`till action_item get 1.5.2`) errors with a clear message — project is required.
  - CLI `till action_item update 2.1 ...` rejects with the same mutations-require-UUID error class.
  - **MCP tool description for `till.action_item` explicitly documents:** `action_item_id` accepts UUID OR dotted form (project inferred from session); list of mutations that reject dotted form.
  - Unknown / out-of-range dotted addresses propagate `ErrDottedAddressNotFound` upward as MCP/CLI errors with descriptive messages naming the level + index that failed.
  - `mage ci` green (drop boundary — full validation that all four units composed cleanly).
  - DB action: NONE.
- **Blocked by:** 2.10

---

## Notes

### Round 2 revision summary

This is the Round 2 revision of `workflow/drop_2/PLAN.md`. Round 1 was reviewed by `PLAN_QA_PROOF.md` (5 blockers, 4 nits, FAIL) and `PLAN_QA_FALSIFICATION.md` (9 blockers, 6 nits, FAIL). Brief items applied (per orchestrator):

- **B1.** Collapsed Round 1's three-droplet Unit B (2.7 domain / 2.8 TUI / 2.9 MCP) into ONE atomic droplet (new 2.7) that flips every reference in one commit. Preserves the mage-ci-green-between-droplets invariant.
- **B2.** Enumerated all 14 cross-package files in Unit B's `Paths:` (originals: `internal/domain/action_item.go`, `internal/tui/thread_mode.go`, `internal/adapters/server/common/capture.go` + `_test.go` + `app_service_adapter_lifecycle_test.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/app/service.go` + `_test.go` + `snapshot.go` (state-symbol section) + `snapshot_test.go` + `attention_capture.go` + `_test.go`, `internal/config/config.go` + `_test.go`). Plus additions discovered during verification: `internal/app/service.go:2467` (test fixture), `internal/adapters/server/common/types.go` (counter struct fields), `internal/adapters/server/common/app_service_adapter.go` + `_test.go` (counter-field references), `internal/domain/domain_test.go:114` column-rename test (verified state-vocab-relevant — column name `"done"`).
- **B3.** Added `canonicalLifecycleState` site at `internal/adapters/server/common/capture.go:296-312` to Unit B scope. This was Round 1's most-damaging miss (independent coercion site).
- **B4.** Narrowed acceptance greps to state-machine context. Replaced unbounded `git grep -E '"done"' -- '*.go'` (which catches 9+ legitimate non-state uses: capability-lease revoke reasons, comment bodies, handoff summaries, embedding-job status, outcome validator) with scope-aware regexes: `git grep "domain.StateDone\b"`, `git grep "lifecycle_state.*\"done\""`, `git grep "json:\"done\""`, plus per-file checks scoped to state-machine source files (`internal/domain/workitem.go`, `internal/app/service.go`, etc.).
- **B5.** Added `Repository.ListActionItemsByParent` requirement to droplet 2.10 (renumbered from Round 1's 2.12). Pulled `internal/app/ports.go` and `internal/adapters/storage/sqlite/repo.go` into the droplet's `Paths:`.
- **B6.** Confirmed `internal/config/config.go isKnownLifecycleState` is strict-canonical (no legacy alias tolerance for "external config compatibility"). Added `internal/config/` to Unit B's `Paths:` and acceptance.
- **B7.** Renumbered droplets after Unit B collapse: 13 → 11. New chain: 2.1 (Unit B-zero), 2.2-2.6 (Unit A, 5 droplets), 2.7 (Unit B, atomic), 2.8-2.9 (Unit C), 2.10-2.11 (Unit D). Every `Blocked by:` reference updated.
- **B8.** Kept `ParseRoleFromDescription` parser as domain helper (dev decision — ~50 LOC, costs nothing, available for future opportunistic callers even though no production caller exists post-no-migration-decision).
- **B9.** Confirmed `internal/adapters/server/common/mcp_surface.go:227 Completed bool json:"completed"` is independent of lifecycle state — it's a checklist-item-completed boolean on an MCP response shape, unrelated to `ChecklistItem.Done`. No rename, no acceptance criterion touches it. (Verified via Read at HEAD.)
- **B10.** Updated `## Notes` to reflect every change above (this section, plus updates to Cross-droplet decisions, Explicit YAGNI rulings, Deferrals).

### Round 3 revision summary

This is the Round 3 revision following `PLAN_QA_PROOF_R2.md` (FAIL — 4 surgical blockers + 3 nits) and `PLAN_QA_FALSIFICATION_R2.md` (FAIL — 5 surgical blockers + 6 nits). All Round 1 blockers stayed resolved across Round 2; Round 3 closes surgical drift. Patches applied (orch self-edit, no planner respawn — surgical only):

- **R3-1.** Added `internal/app/kind_capability_test.go` to Droplet 2.7 `Paths:` — `Done: false` test fixture at `:429` would compile-fail post-rename without it.
- **R3-2.** Added `service_test.go:1561, 1567, 2953` legacy state literals (`States: []string{"progress"}`, `StateID == "progress"`) to 2.7's enumeration; rewrite to canonical (`"in_progress"`).
- **R3-3.** Added `Blocked by: 2.5, 2.6` to Droplet 2.7. Both 2.5 and 2.7 edit `internal/adapters/server/common/app_service_adapter_mcp.go`; cross-unit serialization was in prose only, not in the DAG. Now enforced.
- **R3-4.** Removed `domain_test.go:114` from 2.7's rename scope. Column name `"done"` at that line is free-form column-rename test input, NOT a lifecycle state literal.
- **R3-5.** Added `internal/app/service_test.go fakeRepo` extension to **Droplet 2.10** `Paths:`. Adding `ListActionItemsByParent` to `Repository` interface compile-breaks every test using `fakeRepo` (25+ sites) without the fake also implementing the new method.
- **R3-6.** Added `internal/adapters/server/mcpapi/extended_tools.go:1339` to 2.7 `Paths:` — MCP tool description string `"...todo|in_progress|done"` rewrites to `"...todo|in_progress|complete"`. LLM agents read tool descriptions as canonical examples; legacy literal contradicts strict-canonical at the external-API contract surface.
- **R3-7.** Cite drift sweep: `service.go:556, :694` reference `domain.StateTodo` (NOT renamable) — removed from rename instructions. `IsValidLifecycleState` corrected to lowercase `isValidLifecycleState` (unexported, line `:166`). `kind_capability_test.go` cite extended from `:35` to `:35, :73` (both have `RequireChildrenDone`). `service_test.go:3039 → :3040` cite drift fix. `service_test.go:3797` is `Reason: "done"` lease-revoke (free-form, NOT state literal) — removed from over-claim. Added `Done: true|false` field-literal acceptance grep.
- **R3-8.** Dotted-address resolver final spec applied (Droplet 2.10): **project is NOT in the dotted body** — supplied separately (CLI `--project <slug>` flag, slug-prefix shorthand `<proj_slug>:<dotted>`, or MCP session inference). Body is 0-indexed positions among parent's children at each level. `ORDER BY created_at ASC, id ASC` (or `position ASC, ...` if a position column exists — quick schema check at build time). `ErrDottedAddressAmbiguous` removed (unreachable by construction with deterministic ordering). Memory `project_tillsyn_cascade_vocabulary.md` updated to match. **(R3-8's `position ASC` conditional was superseded by R4-1 below — committed exclusively to `created_at ASC, id ASC` after falsification proved the existing `position` column is column-scoped, not parent-scoped.)**

### Round 4 revision summary

This is the Round 4 revision following `PLAN_QA_FALSIFICATION_R3.md` (FAIL — 2 surgical blockers + 3 carryover nits). PROOF Round 3 passed clean. Round 4 closes the two falsification blockers; no architectural change.

- **R4-1.** **Removed the `position ASC` conditional from R3-8** (PLAN.md `:287`). Falsification verified at HEAD: the existing `position` column on `action_items` is COLUMN-scoped (Kanban TUI arrangement), keyed by `idx_action_items_project_column_position` at `internal/adapters/storage/sqlite/repo.go:487`, NOT parent-scoped (tree). Reusing it would produce a resolver where dotted addresses shift under user drag-reordering of children across TUI columns even when the tree is unchanged. R4-1 commits to `ORDER BY created_at ASC, id ASC` exclusively (UUID `id` is a globally unique tie-breaker — total ordering). If a future drop adds a per-parent `tree_position` column, that drop revisits the resolver.
- **R4-2.** **Added `internal/domain/kind_capability_test.go:19` `Done: false` field literal to 2.7 scope.** Falsification spotted that R3-1 fixed only the sibling `internal/app/kind_capability_test.go:429` and missed the `internal/domain/` version. Without `:19` in 2.7's `Paths:`, `mage test-pkg ./internal/domain` compile-fails after the field rename.

Carryover nits accepted as downgrade-acceptable (PROOF + FALSIFICATION agreed):

- **R2-F11.** `internal/adapters/server/common/capture_test.go:199` debug-message format-string label `"WorkOverview counts = %#v, want todo=2 progress=1 done=1 failed=1 archived=1"` — debug-only (failure-message label, no runtime impact). Routed for opportunistic builder fix during 2.7 under the MD-adjacent carve-out.
- **R2-F7 / R3 nit.** `internal/app/service.go:1965-1975` cite range over-claim (covered by 2.7 acceptance grep sweep).
- **R2-F9 / R3 nit.** `mcp_surface.Completed` rationale gloss inaccurate but the field's independence from lifecycle state IS confirmed (covered by Notes "Cross-droplet decisions").
- **R3 nit.** R3-8 test list missing explicit same-`created_at` tie-break case — covered by `id ASC` deterministic tie-breaker; `repo_test.go` round-trip catches in build-QA.

### Aggregate counter rename (Round 2 discovery)

Strict-canonical state vocabulary implies aggregate-counter field names tracking a state should follow the rename. Verified hits:

- `internal/adapters/server/common/types.go:142-143` — struct fields `InProgressActionItems` (already canonical name) and `DoneActionItems` (rename → `CompleteActionItems`); JSON tags `in_progress_tasks` (canonical) and `done_tasks` (rename → `complete_tasks`).
- `internal/app/attention_capture.go:95-96` — fields `InProgressItems` (canonical) and `DoneItems` (rename → `CompleteItems`); JSON tags `in_progress_items` (canonical) and `done_items` (rename → `complete_items`).
- Increments at `internal/adapters/server/common/capture.go:259, 261` and `internal/app/attention_capture.go:351, 354`, plus reader sites at `internal/adapters/server/common/app_service_adapter.go:409, 421` and tests.

These are folded into Droplet 2.7. Rationale: the field name is part of the state-vocabulary surface — a `DoneActionItems` field tied to "items in `domain.StateDone` state" should rename to `CompleteActionItems` when `StateDone` becomes `StateComplete`. Half-renaming would leave a confusing tree.

### `RequireChildrenDone` field renamed (Round 2 dev decision)

`internal/domain/workitem.go:89` defines `CompletionPolicy.RequireChildrenDone bool` with JSON tag `json:"require_children_done"`. Under strict-canonical this renames to `RequireChildrenComplete` / `json:"require_children_complete"` — folded into Droplet 2.7. Rationale: the field semantically maps to "children in `complete` state"; leaving it as `Done` while every state-machine site renames creates a vocab island. Pre-MVP, dev fresh-DBs after 2.7 lands; no persisted-snapshot back-compat concern.

Sites updated in Droplet 2.7 (all already in 2.7's `Paths:`, file-line cites verified):

- **Definition:** `internal/domain/workitem.go:89` — field rename + JSON tag rename.
- **Reader:** `internal/domain/action_item.go:310` — `policy.RequireChildrenDone` → `policy.RequireChildrenComplete`.
- **Test fixtures:** `internal/adapters/server/common/capture_test.go:126`; `internal/app/service_test.go:3042, 3095, 4613`; `internal/domain/domain_test.go:430, 566, 614-615`; `internal/domain/kind_capability_test.go:35, :73`.

### Template-loader-coupling investigation (Unit B-zero)

**Question:** Does deleting `templates/builtin/*.json` require also deleting/stubbing Go loader code?

**Answer: Yes — delete the entire `templates/` package** (`templates/embed.go` + the JSON files + the directory).

**Evidence:**

1. `templates/embed.go` (the only Go file in `templates/`) uses `//go:embed builtin/*.json`. Per Go embed semantics (verified via Context7 / `pkg.go.dev/embed@go1.25.3`), `//go:embed` patterns must match files; a directive matching no files is a build error.
2. `git grep "evanmschultz/tillsyn/templates"` returns **empty** across the whole tree — zero importers of the `templates` package. The package exists only to expose `embed.FS`, but nothing in the live runtime reads it. (Verified via `git grep "templates.ReadFile"`, `git grep "templates.Files"`, and full-module-path search.)
3. Boot-seed of `kind_catalog` happens entirely in `internal/adapters/storage/sqlite/repo.go:286-375` via inline SQL — NOT via JSON template loading. The 12-kind enum is hard-coded in Go.
4. `instructions_tool.go:329,379` mention `default-go` only as instructional prose strings (not as JSON loading).
5. `README.md`, `CLAUDE.md`, `PLAN.md` reference `templates/builtin/default-go.json` and `default-frontend.json` only as documentation. Drop 3 will rewrite the template story; the surviving MD references are not load-bearing for Drop 2.

**Implication for Droplet 2.1:** delete the entire `templates/` package — not just the JSON files. This is a one-droplet operation with zero downstream Go-tree fallout. The MD references in README.md / CLAUDE.md / PLAN.md may be touched at builder discretion if the surrounding MD section is small and obviously-broken-by-deletion; otherwise they live as-is until Drop 3's template overhaul rewrites those sections.

### Cross-droplet decisions

- **Role field on `ActionItem` is optional** (empty allowed at the constructor; `IsValidRole` rejects empty as part of the closed-enum check, but the `NewActionItem` validator only invokes it on non-empty input). Rationale: pre-MVP, existing description-prose `Role:` lines aren't backfilled into a column; rows without a parsable role land with empty role.
- **`ParseRoleFromDescription` is kept as a domain helper** (Round 2 dev decision). No production caller exists post-no-migration-decision; the parser is ~50 LOC pure code with table-driven tests, costs nothing, and is available for future opportunistic callers (Drop 3+ MCP / CLI paths may want to lift `Role:` lines from description prose at create time).
- **Strict-canonical state literals (Unit B, dev decision 2026-05-01):** every state-coercion site in the tree (`domain.NormalizeLifecycleState`, `app.normalizeStateID`, `app.lifecycleStateForColumnID`, `common.canonicalLifecycleState`, `common.actionItemLifecycleStateForColumnName`, `common.normalizeStateLikeID`, `tui.normalizeColumnStateID`, `tui.lifecycleStateForColumnName`, `config.isKnownLifecycleState`) accepts ONLY canonical inputs (`"complete"`, `"in_progress"`, `"todo"`, `"archived"`, `"failed"`). Every legacy form (`"done"`, `"progress"`, `"completed"`, `"doing"`, `"in-progress"`) returns the unknown-state error path or false. NO legacy alias tolerance for "external config compatibility" — `internal/config/config.go` is strict-canonical too. Pre-MVP every caller is the dev; broken callers fail loud and get fixed at the source. Rationale: no lingering legacy-tolerance trash that would need cleanup later.
- **`ChecklistItem.Done bool → ChecklistItem.Complete bool` (strict-canonical):** JSON tag changes from `"done"` to `"complete"`. Unmarshal accepts ONLY `"complete"` — `"done"` keys produce a decode error. No fallback alias, no `UnmarshalJSON` shim. Pre-MVP no persisted-snapshot rows exist; any test fixture with legacy keys gets rewritten to canonical in the same droplet.
- **Aggregate counter rename:** `DoneActionItems → CompleteActionItems`, `DoneItems → CompleteItems`, plus their JSON tag mirrors. See "Aggregate counter rename" section above.
- **`CompletionPolicy.RequireChildrenDone → RequireChildrenComplete` rename** (Round 2 dev decision). Field rename + JSON tag rename `json:"require_children_done"` → `json:"require_children_complete"` + every reader/test fixture site. See "RequireChildrenDone field renamed" section above.
- **`mcp_surface.Completed` field is independent** of lifecycle-state vocabulary. `internal/adapters/server/common/mcp_surface.go:227 Completed bool json:"completed"` is an MCP-response checklist-completion boolean on a different struct from `ChecklistItem`. NO rename, NO acceptance criterion in Drop 2 touches this field. Verified via Read at HEAD.
- **Resolver location (Unit D):** `internal/app/dotted_address.go` (not `internal/domain`). Resolution requires a project-context repo lookup, which is an application-service concern. Keep `internal/domain` free of repo dependencies.
- **`Repository.ListActionItemsByParent` is added** in Droplet 2.10 alongside the resolver. Clean-interface decision over an O(N) walk via existing `ListActionItems`. New method on the `Repository` interface + a SQLite-side implementation.

### Explicit YAGNI rulings

- **No `mage migrate` or `till migrate` CLI subcommand** for any of the four units. Pre-MVP rule.
- **No SQL migration scripts** under `main/scripts/` for any of the four units. Pre-MVP rule.
- **No `internal/app/migrations/` package.** The `ParseRoleFromDescription` helper is a domain helper (`internal/domain/role.go`), not a migration runner.
- **No JSON-decoder alias for `ChecklistItem.Done` legacy key.** `templates/builtin/*.json` is being deleted in Unit B-zero, which removes the only on-disk source of `"done": false` checklist items in this tree.
- **No partial / shim path on Unit B-zero.** Delete the whole `templates/` package outright; do not leave an orphan loader pointing at a stub file.
- **No partial split of Unit B's strict-canonical sweep.** Round 1's three-droplet split (2.7/2.8/2.9) is reversed in Round 2 — strict-canonical is one atomic invariant; partitioning breaks `mage ci`-green-between-droplets.
- **`main/PLAN.md` § 19.2 vs no-migration constraint:** PROOF Round 1 finding 5 noted that `main/PLAN.md` § 19.2 still names `internal/app/migrations/role_hydration.go` (Round 1 plan also flagged the orchestrator already corrected the line; this Round 2 plan trusts the orchestrator's correction). If the parent PLAN.md still has that text, it's a follow-up to patch out — the executable plan in this file is canonical for Drop 2.

### Deferrals to later drops

- **Drop 3 — full template overhaul.** Reintroduces `templates/` from scratch with closed TOML schema, `[child_rules]` validator, default `templates/builtin/default.toml`. PLAN.md § 19.3.
- **Drop 4 — dispatcher.** Reads template-bound role + kind axes, fans out subagents.
- **Drop 4.5 — TUI overhaul.** Includes dotted-address bindings in TUI (resolver lands in Drop 2; TUI consumption lands later).
- **Future refinement drop — strip `scope` column from `action_items`.** Mirroring `scope` from `kind` lives until then per PLAN.md § 19.2 explicit OOS.
- **Future refinement drop — MD content cleanup.** Drop 2's Unit B-zero deletion + Unit B state rename will leave stale references in `README.md` / `CLAUDE.md` / `PLAN.md`. **Drop 2 carve-out (dev decision 2026-05-01, this drop only):** builders MAY make trivial in-section MD edits adjacent to their droplet's `paths` if the change is a single-sentence / single-phrase fix obviously broken by their code change. **Tightened boundary:** delete the broken phrase or replace with `<deleted in Drop 2 — see PLAN.md § 19.3>`. No paraphrasing surrounding sentences. Anything beyond a delete-or-stub is out of scope and routes to a future MD-cleanup refinement drop. Build-QA (proof + falsification) MUST verify any MD edits via `git diff` and confirm correctness. **Whole-document or whole-section MD sweeps remain out of scope here.** Future drops route MD cleanup to planner+QA, not builder. This carve-out does not establish a precedent.
