# DROP_2 — HIERARCHY REFACTOR

**State:** planning
**Blocked by:** —
**Paths (expected):** `internal/domain/`, `internal/app/`, `internal/adapters/storage/sqlite/`, `internal/adapters/server/common/`, `internal/tui/`, `templates/builtin/` (deletion), `cmd/till/`
**Packages (expected):** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/common`, `internal/tui`, `cmd/till`
**PLAN.md ref:** `main/PLAN.md` § 19.2 — drop 2 — Hierarchy Refactor
**Started:** 2026-05-01
**Closed:** —

## Scope

Drop 2 is the hierarchy-refactor drop. Four units of work, all grounded in `main/PLAN.md` § 19.2:

1. **Promote `metadata.role` to a first-class domain field.** Closed-enum `Role` type with 9 values (`builder`, `qa-proof`, `qa-falsification`, `qa-a11y`, `qa-visual`, `design`, `commit`, `planner`, `research`). Pure parser (`ParseRoleFromDescription`) lives in `internal/domain/role.go`. `Role` field added to `ActionItem` struct with validation. SQLite schema column. MCP `role` field on action-item create/update/get + snapshot serialization. **No hydration runner, no `till migrate` CLI subcommand, no SQL backfill — pre-MVP, dev deletes `~/.tillsyn/tillsyn.db` after the unit lands.**
2. **State-vocabulary rename: `done → complete` AND `progress → in_progress`** (bundled). Touches `internal/domain/workitem.go` (`StateDone → StateComplete`, `StateProgress → StateInProgress` constants, `IsTerminalState`, alias normalization), `ChecklistItem.Done bool → ChecklistItem.Complete bool` field including JSON serialization key, TUI state-string surfaces (`internal/tui/model.go` and `internal/tui/options.go`), MCP coercion at `internal/adapters/server/common/app_service_adapter_mcp.go`. **Pre-step: delete `templates/builtin/*.json` entirely (Drop 3 will overhaul the template system from scratch); also delete or neutralize the Go loader code that reads them.** No state-rewrite SQL script; dev deletes DB.
3. **Strip hardwired nesting defaults from the domain catalog (mechanism stays).** Set every `KindDefinition.AllowedParentScopes` to empty in boot-seed payloads (`internal/adapters/storage/sqlite/repo.go`). The `AllowsParentScope` enforcement path at `internal/app/kind_capability.go:566` continues to work — empty defaults make it return true for every parent (universal-allow). Delete the speculative `domain.AllowedParentKinds(Kind) []Kind` function (zero production callers per PLAN.md). One DB UPDATE script for any existing rows' `allowed_parent_scopes_json` is also OUT — dev fresh-DBs.
4. **Dotted-address fast-nav reads.** Pure resolver in `internal/domain` or `internal/app` taking a dotted string + project context, returns UUID or ambiguity/missing error. Wire into `till.action_item(operation=get)` MCP read + CLI read commands. Mutation paths reject dotted form. TUI bindings deferred to Drop 4.5.

**Order matters per PLAN.md § 19.2:** role promotion (no state-machine changes) → state rename (touches state machine + JSON template deletion + many files) → strip nesting defaults (orthogonal) → dotted-address reads (zero coupling, lands last so rename churn settles before resolver tests).

**Out of scope (explicit, per PLAN.md § 19.2):** commit cadence rules, reverse-hierarchy prohibitions, auto-create rules, template wiring (all Drop 3); dispatcher (Drop 4); TUI overhaul (Drop 4.5); `scope` column removal (deferred to a future refinement drop).

**Pre-MVP rules in effect (per memory):**

- No migration logic in Go code, no `till migrate` subcommands, no one-shot SQL scripts. Dev deletes `~/.tillsyn/tillsyn.db` between schema or state-vocab-changing units.
- No `CLOSEOUT.md`, no `LEDGER.md` entry, no `WIKI_CHANGELOG.md` entry, no `REFINEMENTS.md` entry, no `HYLLA_FEEDBACK.md` rollup, no `HYLLA_REFINEMENTS.md` rollup. Worklog MDs (this `PLAN.md`, `BUILDER_WORKLOG.md`, `BUILDER_QA_*.md`, `PLAN_QA_*.md`) DO happen.
- Drop 2 closes when all `main/PLAN.md` § 19.2 checkboxes are checked. No separate state-bearing row.

## Planner

Decomposition into atomic droplets. Order: B-zero → A → B → C → D, per the PLAN.md § 19.2 ordering paragraph and the pre-MVP rules in the Scope section above. Each unit lands `mage ci` green before the next unit's first droplet starts. Within a unit, droplets that share a Go package carry an explicit `Blocked by:` to the prior package-touching droplet — same-package-parallel-edits break each other's compile.

Acceptance verification target throughout: `mage test-pkg <pkg>` (per droplet) and `mage ci` (per unit boundary). **Never `mage install`** — dev-only.

---

### Unit B-zero — Delete builtin template JSON + neutralize loader (prerequisite to all other units)

The pre-step from Scope item 2. Lands first because Unit B's state rename would otherwise have to sweep ~80 `"done": false` and ~unknown `"progress"` literals across `templates/builtin/default-go.json` (76k file) + `templates/builtin/default-frontend.json` (71k file). Drop 3 overhauls templates from scratch — keeping these JSON files alive through Drop 2 only to delete them at Drop 3 start is wasted churn.

**Loader-coupling investigation result (see `## Notes` for full analysis):** `templates/embed.go` is the only Go file in the `templates` package and uses `//go:embed builtin/*.json`. Zero importers anywhere in the Go tree (verified: `git grep "evanmschultz/tillsyn/templates"` returns empty). Per Go embed semantics, deleting all matching files makes `//go:embed builtin/*.json` a build error. **Therefore Unit B-zero must delete the entire `templates/` package — both the JSON files AND `templates/embed.go`** in one droplet, or stub the embed directive to point at a placeholder. Going with full deletion: simpler, no orphan code, Drop 3 reintroduces a fresh `templates/` package on its own terms.

#### Droplet 2.1 — Delete `templates/` package outright

- **State:** todo
- **Paths:** `templates/builtin/default-go.json` (delete), `templates/builtin/default-frontend.json` (delete), `templates/embed.go` (delete), `templates/builtin/` (delete dir if empty), `templates/` (delete dir if empty)
- **Packages:** `github.com/evanmschultz/tillsyn/templates` (deletion)
- **Acceptance:**
  - `git rm` removes all four paths above (plus their parent dirs if empty).
  - `git grep "evanmschultz/tillsyn/templates"` returns empty across the whole repo (no orphan imports).
  - `git grep "templates/builtin"` returns only MD references in `README.md`, `PLAN.md`, `CLAUDE.md`, and `workflow/drop_2/PLAN.md` (those are MD content edits, NOT Go-tree references, and may stay until Drop 3 cleanup or be touched by builder if trivially in scope — see Notes).
  - `mage ci` green.
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
- **Blocked by:** —

#### Droplet 2.3 — Add `Role` field to `ActionItem` + `ActionItemInput` + `NewActionItem` validation

- **State:** todo
- **Paths:** `internal/domain/action_item.go` (add `Role Role` field to both structs; add validation block in `NewActionItem`), `internal/domain/action_item_test.go` or `internal/domain/domain_test.go` (extend existing `NewActionItem` table-driven tests)
- **Packages:** `internal/domain`
- **Acceptance:**
  - `ActionItem` struct gains `Role Role` field (zero-value empty string allowed).
  - `ActionItemInput` struct gains `Role Role` field.
  - `NewActionItem` normalizes `in.Role` via `NormalizeRole`; if non-empty, calls `IsValidRole`; on failure returns `ErrInvalidRole`. Empty role is permitted (returns the zero-value Role on the constructed `ActionItem`).
  - Table-driven test additions: empty role round-trips empty; each of 9 valid roles round-trips; unknown role rejected with `ErrInvalidRole`; whitespace-only role normalizes to empty.
  - All existing `domain_test.go` tests remain green (no regressions on the 12-value `Kind` validation path).
  - `mage test-pkg ./internal/domain` green.
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
- **Blocked by:** 2.4

#### Droplet 2.6 — Snapshot serialization for `Role`

- **State:** todo
- **Paths:** `internal/app/snapshot.go` (add `Role domain.Role \`json:"role,omitempty"\`` to `SnapshotActionItem` struct at `:57`; thread the field through `snapshotActionItemFromDomain` at `:1057` and `(t SnapshotActionItem) toDomain()` at `:1263`), `internal/app/snapshot_test.go` if it exists, otherwise extend whichever test exercises `SnapshotActionItem` round-trip
- **Packages:** `internal/app`
- **Acceptance:**
  - Snapshot round-trip preserves a non-empty `Role` value.
  - Snapshot with empty role round-trips empty (omitempty drops the JSON key on serialize).
  - JSON shape: `{"role":"builder"}` when set, key absent when empty.
  - `mage test-pkg ./internal/app` green.
- **Blocked by:** 2.3

---

### Unit B — State-vocabulary rename `done → complete`, `progress → in_progress`

Bundled in one sweep per PLAN.md § 19.2. After Unit B-zero deleted `templates/builtin/*.json`, the rename only touches Go code + Go tests + checklist JSON keys (no template JSON sweep). Three droplets. The rename is intrusive enough that a single droplet would balloon past the atomic-droplet ceiling; splitting by package layer keeps each droplet inspectable.

Cross-unit ordering: 2.7 is `Blocked by: 2.6` to honor PLAN.md § 19.2's explicit "role promotion → state rename" ordering. Same-package-blocking inside Unit B: 2.7 owns the `internal/domain` flip (constants + alias normalization + `IsTerminalState` + `ChecklistItem.Done → Complete` field rename). 2.8 owns the `internal/tui` sweep (model.go + options.go). 2.9 owns the MCP coercion + final repo-wide grep sweep. **2.7 unblocks 2.8 and 2.9** because the constants it renames are the truth source the others reference. 2.8 + 2.9 touch disjoint packages and could parallelize in principle, but serialize anyway: 2.9's grep sweep depends on 2.8's TUI rename being committed so the sweep can confirm zero stragglers.

#### Droplet 2.7 — Domain rename: `StateDone → StateComplete`, `StateProgress → StateInProgress`, `ChecklistItem.Done → Complete`

- **State:** todo
- **Paths:** `internal/domain/workitem.go` (rename `StateDone` → `StateComplete` and `StateProgress` → `StateInProgress` constants at `:17-22`; flip alias normalization at `:147-163` so canonical-out is `complete`/`in_progress` and aliases (`"done"`, `"completed"`, `"progress"`, `"doing"`, `"in-progress"`) coerce to canonical; flip `IsTerminalState` at `:171-175` to test against `StateComplete`/`StateFailed`; rename `ChecklistItem.Done bool` → `ChecklistItem.Complete bool` at `:80-85` including the JSON tag `\`json:"complete"\``), `internal/domain/workitem_test.go` and any other `internal/domain/*_test.go` files referencing the old constants/field name (rename references)
- **Packages:** `internal/domain`
- **Acceptance:**
  - `git grep "StateDone\b"` returns empty.
  - `git grep "StateProgress\b"` returns empty.
  - `git grep "ChecklistItem.*Done bool\|\.Done = true\|\.Done = false"` returns only test fixtures that the builder updated to use `Complete`.
  - `IsTerminalState(StateComplete)` returns true; `IsTerminalState(StateFailed)` returns true; all other states return false.
  - Alias normalization: input `"done"`, `"complete"`, `"completed"` → `StateComplete`. Input `"progress"`, `"in-progress"`, `"in_progress"`, `"doing"` → `StateInProgress`.
  - `ChecklistItem` JSON marshal emits `"complete":` not `"done":`. JSON unmarshal accepts both `"complete"` (canonical) AND legacy `"done"` (during alias normalization); decide explicit alias vs strict during build — the table-driven test should pin behavior either way.
  - Existing `internal/domain/*_test.go` tests still green after rename.
  - `mage test-pkg ./internal/domain` green.
- **Blocked by:** 2.6

#### Droplet 2.8 — TUI state-string sweep

- **State:** todo
- **Paths:** `internal/tui/model.go` (update `canonicalSearchStatesOrdered` at `:305` from `["todo", "progress", "done", "archived"]` to `["todo", "in_progress", "complete", "archived"]`; update `searchStates` at `:1231` and `dependencyStates` at `:1236` from `["todo", "progress", "done"]` to `["todo", "in_progress", "complete"]`; sweep label maps and any other `"done"` / `"progress"` literals — full grep sweep), `internal/tui/options.go` (update default-state list at `:147` from `["todo", "progress", "done"]` to `["todo", "in_progress", "complete"]`), `internal/tui/model_test.go` and any other `internal/tui/*_test.go` references to old state literals
- **Packages:** `internal/tui`
- **Acceptance:**
  - `git grep -F "\"done\"" internal/tui/` returns empty (or only test fixtures that explicitly check legacy-alias coercion if 2.7's normalizer kept the alias).
  - `git grep -F "\"progress\"" internal/tui/` returns empty (same caveat).
  - `git grep -F "\"in_progress\"" internal/tui/` and `git grep -F "\"complete\"" internal/tui/` return non-empty (canonical literals are present).
  - All `internal/tui/*_test.go` tests still green after literal sweep.
  - `mage test-pkg ./internal/tui` green.
- **Blocked by:** 2.7

#### Droplet 2.9 — MCP coercion + final repo-wide grep sweep

- **State:** todo
- **Paths:** `internal/adapters/server/common/app_service_adapter_mcp.go` (update `actionItemLifecycleStateForColumnName` at `:849-864` so column-name `"done"` resolves to `domain.StateComplete` and `"progress"` resolves to `domain.StateInProgress`; update `normalizeStateLikeID` at `:866-901` so the switch carries both canonical (`"in-progress"`/`"in_progress"`) and legacy (`"progress"`/`"doing"`) inputs to `"in_progress"`, and `"done"`/`"complete"`/`"completed"` to `"complete"` — alias tolerance for pre-rename callers; canonical writes use the new values), `cmd/till/main.go` (any `"done"`/`"progress"` literals in CLI state filtering or state-name display — full grep sweep), all tests touched by the rename
- **Packages:** `internal/adapters/server/common`, `cmd/till`
- **Acceptance:**
  - `git grep -F "StateDone\|StateProgress" -- '*.go'` returns empty across the WHOLE repo (including production + test).
  - `git grep -E '"done"' -- '*.go'` returns only intentional alias-tolerance cases (the MCP coercion switch + tests asserting alias behavior).
  - `git grep -E '"progress"' -- '*.go'` returns only intentional alias-tolerance cases.
  - The `till.action_item` MCP create/list/move-state tool accepts both legacy (`"done"`, `"progress"`) and canonical (`"complete"`, `"in_progress"`) state values; canonical values round-trip through reads.
  - `mage ci` green (this is the unit boundary — the unified CI run validates that 2.7 + 2.8 + 2.9 left no stragglers anywhere).
- **Blocked by:** 2.8

---

### Unit C — Strip hardwired nesting defaults from domain catalog (mechanism stays)

Two droplets. Independent of Unit B per PLAN.md § 19.2 ("orthogonal, can run in parallel with the rename if convenient"); ordered after Unit B here only because B leaves no Go-tree fallout for C to step on. **2.10** flips the boot-seed payloads (the live production data path). **2.11** deletes the speculative `AllowedParentKinds` function (the dead-code path).

Same-package-blocking: 2.10 + 2.11 are in different packages (`internal/adapters/storage/sqlite` vs `internal/domain`), so independent. 2.11 also touches a doc-comment in `internal/app/snapshot.go:448` and `internal/adapters/storage/sqlite/repo.go:300` — still no compile-overlap conflict with 2.10 because comment edits do not race compile units.

#### Droplet 2.10 — Empty `AllowedParentScopes` for every kind in boot-seed

- **State:** todo
- **Paths:** `internal/adapters/storage/sqlite/repo.go` (change every `INSERT OR IGNORE INTO kind_catalog` row at `:304-375` so `allowed_parent_scopes_json` is `'[]'` instead of `'["plan"]'` or `'["build"]'` — 12 rows total: `plan`, `research`, `build`, `plan-qa-proof`, `plan-qa-falsification`, `build-qa-proof`, `build-qa-falsification`, `closeout`, `commit`, `refinement`, `discussion`, `human-verify`), `internal/adapters/storage/sqlite/repo_test.go` and `internal/app/kind_capability_test.go` and `internal/domain/kind_capability_test.go` (update any test that asserted the old `["plan"]`/`["build"]` defaults; assert universal-allow behavior of `AllowsParentScope` per `internal/domain/kind.go:225-232`)
- **Packages:** `internal/adapters/storage/sqlite`, `internal/app`, `internal/domain` (test-only edit)
- **Acceptance:**
  - All 12 `INSERT OR IGNORE INTO kind_catalog` payloads carry `allowed_parent_scopes_json = '[]'`.
  - `KindDefinition.AllowsParentScope(any-scope)` returns `true` for every seeded kind (universal-allow), per the existing `internal/domain/kind.go:225-232` early-return on empty `AllowedParentScopes`.
  - The `AllowsParentScope` enforcement path at `internal/app/kind_capability.go:566` is unchanged — only the input data shape changed.
  - **Pre-MVP rule honored:** no DB UPDATE script for any existing rows' `allowed_parent_scopes_json`; dev fresh-DBs.
  - `mage test-pkg ./internal/adapters/storage/sqlite` and `mage test-pkg ./internal/app` and `mage test-pkg ./internal/domain` all green.
- **Blocked by:** 2.9

#### Droplet 2.11 — Delete `domain.AllowedParentKinds` function + test fixture + doc-comment cleanup

- **State:** todo
- **Paths:** `internal/domain/kind.go` (delete `AllowedParentKinds` function at `:94-117`), `internal/domain/domain_test.go` (delete `TestAllowedParentKindsEncodesHierarchy` at `:680-714`), `internal/app/snapshot.go` (update doc comment at `:448` referencing `domain.AllowedParentKinds` — replace with reference to `KindDefinition.AllowedParentScopes` + `AllowsParentScope`), `internal/adapters/storage/sqlite/repo.go` (update doc comment at `:300` referencing `domain.AllowedParentKinds` — same replacement)
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`
- **Acceptance:**
  - `git grep "AllowedParentKinds"` returns empty across the whole repo (function deleted, test deleted, both doc comments updated).
  - `internal/app/kind_capability.go:566` enforcement path still compiles and tests still pass — that path uses `AllowedParentScopes` + `AllowsParentScope`, NOT the deleted `AllowedParentKinds`.
  - `mage ci` green (unit boundary — confirms whole-tree no orphans).
- **Blocked by:** 2.10

---

### Unit D — Dotted-address fast-nav reads (CLI + MCP read paths)

Pure resolver, single package. Two droplets. Lands last per PLAN.md § 19.2 ordering ("zero coupling, lands last so rename churn settles before resolver tests"). TUI bindings deferred to Drop 4.5 per PLAN.md § 19.2 explicit out-of-scope.

**Decision: resolver lives in `internal/app`** (not `internal/domain`). Rationale: resolution requires a project-context lookup against the action-items repo, which is an application-service concern, not a pure-domain concern. Keep the domain layer free of repo dependencies.

Same-package-blocking: 2.12 owns the resolver in `internal/app`. 2.13 wires CLI + MCP — disjoint packages from 2.12 (`cmd/till` + `internal/adapters/server/...`), so no compile race; serialize anyway because 2.13 calls into the resolver 2.12 created.

#### Droplet 2.12 — Pure dotted-address resolver in `internal/app`

- **State:** todo
- **Paths:** `internal/app/dotted_address.go` (new — function `ResolveDottedAddress(ctx, repo, projectID, dotted string) (actionItemID string, err error)` with sentinel errors `ErrDottedAddressAmbiguous`, `ErrDottedAddressNotFound`, `ErrDottedAddressInvalidSyntax`), `internal/app/dotted_address_test.go` (new — table-driven tests using an in-memory fake or the existing test SQLite fixture)
- **Packages:** `internal/app`
- **Acceptance:**
  - Function signature: `func ResolveDottedAddress(ctx context.Context, repo Repository, projectID string, dotted string) (string, error)` (where `Repository` is the existing app-layer interface that already exposes the action-item read methods; consumer-side interface, not a new abstraction).
  - Accepts these forms: `N` (level-1), `N.M` (level-2), `N.M.K` (level-3), and `<proj_slug>-N.M.K` (slug-prefixed). The slug prefix is optional — when absent, the resolver scopes to the supplied `projectID`. When present, the resolver verifies the slug matches the project named by `projectID`.
  - Returns the resolved action-item UUID on unique match.
  - Returns `ErrDottedAddressNotFound` when the path doesn't lead to an action item.
  - Returns `ErrDottedAddressAmbiguous` when the path is non-unique (multiple matches at some level).
  - Returns `ErrDottedAddressInvalidSyntax` when the input fails the `^([a-z0-9-]+-)?\d+(\.\d+)*$` shape check.
  - Table-driven tests cover: valid `N`, valid `N.M`, valid `N.M.K`, slug-prefixed valid, slug-prefix mismatch, missing path, ambiguous path, malformed inputs (empty, `1.`, `.1`, `1..2`, `abc`, `1.2.3.4.5` deep nesting), UUID input rejected (must use the dotted form OR the caller is expected to skip the resolver).
  - `mage test-pkg ./internal/app` green.
- **Blocked by:** 2.11

#### Droplet 2.13 — Wire resolver into CLI + MCP read paths; mutation paths reject dotted form

- **State:** todo
- **Paths:** `internal/adapters/server/common/app_service_adapter_mcp.go` (in `till.action_item(operation=get)`, accept either UUID or dotted form for `action_item_id` — when input doesn't parse as UUID, call `ResolveDottedAddress`; mutation operations `create|update|move|move_state|delete|restore|reparent` reject dotted form with a clear error), `internal/adapters/server/mcpapi/extended_tools.go` (mirror the get-vs-mutate distinction in tool-level argument validation), `cmd/till/main.go` (CLI read commands accept dotted form via the same resolver; CLI mutation commands reject dotted form), test files for each path
- **Packages:** `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `cmd/till`
- **Acceptance:**
  - `till.action_item(operation=get, action_item_id="2.1")` resolves through the resolver and returns the matching action item.
  - `till.action_item(operation=get, action_item_id="<UUID>")` continues to work unchanged.
  - `till.action_item(operation=update, action_item_id="2.1", ...)` returns a 400-class error explaining that mutations require UUIDs.
  - CLI `till action_item get 2.1` works; CLI `till action_item update 2.1 ...` rejects with the same error class.
  - Unknown / ambiguous dotted addresses propagate `ErrDottedAddressNotFound` / `ErrDottedAddressAmbiguous` upward as MCP/CLI errors with descriptive messages.
  - `mage ci` green (drop boundary — full validation that all four units composed cleanly).
- **Blocked by:** 2.12

---

## Notes

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

- **Role field on `ActionItem` is optional** (empty allowed at the constructor; `IsValidRole` rejects empty as part of the closed-enum check, but the `NewActionItem` validator only invokes it on non-empty input). Rationale: pre-MVP, existing description-prose `Role:` lines aren't backfilled into a column; rows without a parsable role land with empty role. The `ParseRoleFromDescription` helper exists for callers who want to opportunistically lift the value out of description prose at create time.
- **Alias-tolerance for state literals (Unit B):** the `normalizeStateLikeID` switch keeps the legacy aliases (`"done"`, `"progress"`, `"completed"`, `"doing"`, `"in-progress"`) as inputs that coerce to canonical (`"complete"`, `"in_progress"`). Pre-MVP DB has no rows (dev fresh-DBs after the unit), so the alias is purely for client/CLI tolerance during the transition window. Canonical writes use the new values exclusively.
- **`ChecklistItem.Done bool → ChecklistItem.Complete bool`:** the JSON tag changes from `"done"` to `"complete"`. There are no persisted-snapshot rows to back-compat against pre-MVP, so the rename is straight — no JSON-decoder alias is required (decide during build whether to add one for forward-compat with externally-stored snapshots; default is no, dev fresh-DBs).
- **Resolver location (Unit D):** `internal/app/dotted_address.go` (not `internal/domain`). Resolution requires a project-context repo lookup, which is an application-service concern. Keep `internal/domain` free of repo dependencies.

### Explicit YAGNI rulings

- **No `mage migrate` or `till migrate` CLI subcommand** for any of the four units. Pre-MVP rule.
- **No SQL migration scripts** under `main/scripts/` for any of the four units. Pre-MVP rule.
- **No `internal/app/migrations/` package.** The `ParseRoleFromDescription` helper is a domain helper (`internal/domain/role.go`), not a migration runner.
- **No JSON-decoder alias for `ChecklistItem.Done` legacy key** unless build discovers a real consumer (none expected — `templates/builtin/*.json` is being deleted in Unit B-zero, which removes the only on-disk source of `"done": false` checklist items in this tree).
- **No partial / shim path on Unit B-zero.** Delete the whole `templates/` package outright; do not leave an orphan loader pointing at a stub file.

### Deferrals to later drops

- **Drop 3 — full template overhaul.** Reintroduces `templates/` from scratch with closed TOML schema, `[child_rules]` validator, default `templates/builtin/default.toml`. PLAN.md § 19.3.
- **Drop 4 — dispatcher.** Reads template-bound role + kind axes, fans out subagents.
- **Drop 4.5 — TUI overhaul.** Includes dotted-address bindings in TUI (resolver lands in Drop 2; TUI consumption lands later).
- **Future refinement drop — strip `scope` column from `action_items`.** Mirroring `scope` from `kind` lives until then per PLAN.md § 19.2 explicit OOS.
- **Future refinement drop — MD content cleanup.** Drop 2's Unit B-zero deletion + Unit B state rename will leave stale references in `README.md` / `CLAUDE.md` / `PLAN.md`. Trivial in-section MD edits are at builder discretion in Unit B-zero / Unit B; whole-document doc sweep is out of scope here.
