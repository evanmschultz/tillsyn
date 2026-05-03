# DROP_3 — UNIT C — STEWARD AUTH MODEL + TEMPLATE AUTO-GENERATION

**State:** planning
**Blocked by:** Unit A (`metadata.structural_type` enum landed) — soft dep, see Cross-Unit Dependencies; Unit B (`[child_rules]` template infrastructure landed) — hard dep for Droplet 3.C.4
**Paths (expected):** `internal/domain/`, `internal/app/`, `internal/adapters/storage/sqlite/`, `internal/adapters/server/common/`, `internal/adapters/server/mcpapi/`
**Packages (expected):** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
**PLAN.md ref:** `main/PLAN.md` § 19.3 bullets 7–9; `main/PLAN.md` § 15.7 (STEWARD MD ownership split — context for the auth gate)
**Started:** 2026-05-02
**Closed:** —

## Scope

Unit C lands the auth-layer enforcement for STEWARD-owned action items + the data-shape primitives the template auto-generator needs. Three concrete primitives (`metadata.owner` first-class field, `metadata.drop_number` first-class field, `principal_type: steward` enum value), one auth-layer enforcement path (state-transition reject when `owner = STEWARD` and the caller's session principal_type is not `steward`), and one template-driven auto-generation hook that consumes Unit B's `[child_rules]` infrastructure to materialize the 5 STEWARD level_2 findings + the refinements-gate when a `DROP_N_ORCH` creates a level_1 numbered drop.

**Pre-Drop-3 honor-system state:** STEWARD owner-marking lives in description prose + ad-hoc `metadata` JSON keys (`metadata.persistent`, `metadata.owner` set as informational text per `STEWARD_ORCH_PROMPT.md` §5.0 line 149). No auth-layer enforcement — drop-orchs are trusted not to move STEWARD items through state. Unit C ends that honor-system.

**The four PLAN.md § 19.3 bullets Unit C covers:**

1. **`metadata.owner` first-class domain field on `ActionItem`** (orchestrator-locked: first-class string field on the struct, NOT metadata JSON). Empty default. Validation: trimmed string, no closed-enum (callers may store any principal name; `STEWARD` is the only value the auth gate currently keys on, but other owner names are permitted for future template-defined owned kinds).
2. **`principal_type: steward` enum value added to the auth model.** Today `normalizeAuthRequestPrincipalType` (`internal/domain/auth_request.go:597-608`) accepts `user | agent | service`. Unit C extends the closed enum with `steward` as a fourth value, distinct from `agent`. Sessions whose `principal_type = steward` carry the only credential the state-transition gate accepts on `owner = STEWARD` items.
3. **Auth-layer state-lock.** `MoveActionItem` and `MoveActionItemState` (the two MCP entry points that mutate `LifecycleState` — `internal/adapters/server/common/app_service_adapter_mcp.go:728, 744`) consult the loaded action item's `Owner` field. If `Owner == "STEWARD"` and the authenticated caller's session `PrincipalType != "steward"`, the request fails with `ErrAuthorizationDenied`. Other mutation paths (`UpdateActionItem` for description/details/metadata, `CreateActionItem` for child creation) stay open to drop-orchs — they edit content without transitioning state.
4. **Template auto-generation of STEWARD-scope items + refinements-gate.** When a drop-orch creates a level_1 numbered drop (`kind=plan` with parent = project root, plus the parent-context signal that says "this is a numbered DROP_N drop"), Unit B's `[child_rules]` engine fires the auto-generator: it creates the 5 level_2 findings drops under STEWARD's persistent parents (`DROP_N_HYLLA_FINDINGS` / `DROP_N_LEDGER_ENTRY` / `DROP_N_WIKI_CHANGELOG_ENTRY` / `DROP_N_REFINEMENTS_RAISED` / `DROP_N_HYLLA_REFINEMENTS_RAISED`) and the in-tree refinements-gate (`DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1`). Each auto-generated item lands with `Owner = "STEWARD"`, `DropNumber = N`, and the refinements-gate carries `blocked_by` pointing at every other drop N item plus the 5 level_2 findings drops.

**Out of scope for Unit C:**

- `structural_type` enum implementation (Unit A).
- TOML schema / parser / `[child_rules]` engine itself (Unit B). Unit C **consumes** the rule infrastructure; it does not build it.
- Adopter bootstrap docs (Unit D).
- `commit` cadence rules (Drop 3 broader scope; not Unit C).
- Dispatcher integration (Drop 4).

**Pre-MVP rules in effect:**

- Schema additions (`owner`, `drop_number` columns on `action_items`; `steward` value in the principal_type enum check) are pre-MVP — **dev fresh-DBs `~/.tillsyn/tillsyn.db`** between schema-touching droplets.
- No migration logic in Go, no `till migrate` subcommand, no SQL backfill script.
- No `CLOSEOUT.md` / `LEDGER.md` / `WIKI_CHANGELOG.md` / `REFINEMENTS.md` rollups for this drop. Worklog MDs (`PLAN.md`, `BUILDER_WORKLOG.md`, `BUILDER_QA_*.md`, `PLAN_QA_*.md`) DO happen.
- Builders run **opus**.
- Acceptance verification: `mage test-pkg <pkg>` per droplet; `mage ci` at the unit boundary (between droplets that share a package, the prior droplet's `mage ci` must be green before the next droplet starts). **Never `mage install`.**

## Cross-Unit Dependencies

Unit C is the third-positioned unit in Drop 3's planning fan-out. Two hard dependencies on sibling units:

| Dep | Source | What Unit C consumes | Why |
|---|---|---|---|
| Unit A `metadata.structural_type` enum field on `ActionItem` (closed 4-value: `drop | segment | confluence | droplet`) | Unit A planner's PLAN.md | Unit C's auto-generator (Droplet 3.C.4) writes `structural_type` on every auto-generated item — STEWARD persistent parents stay plain `action_item` with `metadata.persistent: true` (per § 19.3 line 1637); level_2 findings drops are plain `action_item` with `metadata.persistent: false, dev_gated: false` (line 1639); the refinements-gate is `structural_type: confluence` because it has non-empty `blocked_by` covering every Drop N item + the 5 level_2 findings (line 1633's confluence definition). Auto-generator must be able to set the field. | Soft — if Unit A delays, Unit C's Droplet 3.C.4 can still build against `metadata.structural_type` as a JSON key on the existing `ActionItemMetadata` struct, then refactor to first-class once Unit A lands. Orchestrator decides at synthesis whether 3.C.4 blocks-on-A or not. |
| Unit B `[child_rules]` template infrastructure (closed-TOML `[child_rules]` table + load-time validator + the runtime hook that fires children on parent creation) | Unit B planner's PLAN.md | Unit C's auto-generator (Droplet 3.C.4) IS a `[child_rules]` consumer — the auto-generator is encoded as a `[child_rules]` entry in the default template. Without the rule engine, there is no hook to fire on. | **Hard.** Droplet 3.C.4 is `Blocked by: 3.B.<final-rule-engine-droplet>` — orch wires this at synthesis after Unit B's PLAN.md surfaces the specific droplet ID. Droplets 3.C.1, 3.C.2, 3.C.3, 3.C.5, 3.C.6 do NOT depend on Unit B and can build in parallel with B. |

Unit D (adopter bootstrap) depends on Unit C indirectly — Unit D's CLAUDE.md / agent-frontmatter sweeps reference the new `Owner` field + `principal_type: steward` semantics. Unit D blocks on Unit C, not the other way around.

## Planner

Decomposition into 6 atomic droplets, ordered to land schema → enum → enforcement → auto-gen → tests in a deterministic sequence. Within each droplet the acceptance criteria are testable; cross-droplet `Blocked by` reflects same-package compile races + data-flow dependencies.

Acceptance verification target throughout: `mage test-pkg <pkg>` (per droplet) and `mage ci` (at unit boundary). **Never `mage install`** — dev-only.

---

### Droplet 3.C.1 — `Owner` and `DropNumber` first-class fields on `ActionItem` + `ActionItemInput` + validation

Mirrors Drop 2.3's `Role` field landing pattern (`workflow/drop_2/PLAN.md` Droplet 2.3, landed at `internal/domain/action_item.go:33, 67, 155-158, 201`). Two new fields on the same struct in the same droplet because they ship together for STEWARD-auto-generated items and there is no compile-race between them.

- **Paths:**
  - `internal/domain/action_item.go` — add `Owner string` and `DropNumber int` fields to both `ActionItem` (around `:33`, after `Role`) and `ActionItemInput` (around `:67`, after `Role`); add validation block in `NewActionItem` (after the existing `Role` validation at `:155-158`); thread the fields into the returned struct literal at `:201`.
  - `internal/domain/domain_test.go` — extend the existing `NewActionItem` table-driven tests to cover: empty-owner round-trips empty; non-empty owner round-trips; whitespace-only owner normalizes to empty; owner with surrounding whitespace is trimmed; `DropNumber = 0` round-trips zero (zero-value, treated as "not a numbered drop"); positive `DropNumber` round-trips; negative `DropNumber` rejected with `ErrInvalidDropNumber`.
- **Packages:** `internal/domain`
- **Acceptance:**
  - `ActionItem` struct gains `Owner string` field (zero-value empty string allowed) and `DropNumber int` field (zero-value 0 allowed).
  - `ActionItemInput` struct gains the same two fields.
  - `NewActionItem` trims `in.Owner` (no closed-enum — any non-empty trimmed value is valid; `"STEWARD"` is the value the auth gate keys on, but other principal names are permitted for future template-defined owned kinds).
  - `NewActionItem` rejects `in.DropNumber < 0` with a new `ErrInvalidDropNumber` sentinel in `internal/domain/errors.go` (mirror the `ErrInvalidRole` style already present from Drop 2.2).
  - Empty owner + zero drop_number is the dominant case (every action item that is not STEWARD-owned and not a numbered drop carries them as zero-value).
  - Table-driven tests added: empty-owner round-trip; `STEWARD` round-trip; whitespace-only owner → empty after normalize; `DropNumber = 1` round-trip; `DropNumber = 1.5` is integer-only (not allowed — it's `int` not `float`, so this is a syntactic rejection at the call site, but the test asserts `DropNumber = 0` and `DropNumber = 5` both round-trip and the negative case rejects).
  - All existing `domain_test.go` tests remain green (no regressions on Kind / Role / lifecycle paths).
  - `mage test-pkg ./internal/domain` green.
  - **DB action:** NONE (struct-only — schema column lands in 3.C.2).
- **Blocked by:** —

**Open question marker for Droplet 3.C.1 (orchestrator routes to dev at synthesis):** is `metadata.drop_number` a domain field or a metadata JSON key? Recommended (locked-in below by orchestrator decision pending dev sign-off): **first-class domain field**, mirroring `Owner` placement and Drop 2.3's `Role` precedent. Rationale: it gets consulted by the auto-generator at template-rule-fire time (frequent read path) and by the refinements-gate `blocked_by` resolver to find sibling Drop N items (cross-row query). A first-class column with an index lets the auto-generator query "all level_2 findings under STEWARD persistent parents WHERE drop_number = N" cleanly; metadata JSON would force every consumer to JSON-decode every row. **If dev pushes back to metadata JSON, only 3.C.1 + 3.C.2 change shape; the auth gate (3.C.3) reads only `Owner`, not `DropNumber`.**

---

### Droplet 3.C.2 — SQLite `action_items.owner` + `action_items.drop_number` columns + scanner + insert/update paths

- **Paths:**
  - `internal/adapters/storage/sqlite/repo.go` — add `owner TEXT NOT NULL DEFAULT ''` and `drop_number INTEGER NOT NULL DEFAULT 0` columns to the `CREATE TABLE IF NOT EXISTS action_items` block at `:168` (verified: column block runs `:168-197`; `role` column at `:174` is the Drop 2.4 precedent — append after `role` and before `lifecycle_state` for stylistic adjacency); add the two columns to `scanActionItem` (verified above the struct insert site at `:2738` — Drop 2.4's plan landed Role here); add the two columns to insert + update SQL inside the action-item write paths.
  - `internal/adapters/storage/sqlite/repo_test.go` — extend the existing round-trip test (Drop 2.4 added `Role` round-trip; Unit C extends with `Owner = "STEWARD"`, `DropNumber = 5` round-trip).
  - **New index for the auto-generator's cross-row queries:** `CREATE INDEX IF NOT EXISTS idx_action_items_project_owner_drop_number ON action_items(project_id, owner, drop_number);` — supports the auto-generator's "find every level_2 finding under STEWARD persistent parents for drop N" query and the refinements-gate `blocked_by` builder's "find every drop N item" query. Lands in the same `:488` index block as the existing `idx_action_items_*` indexes.
- **Packages:** `internal/adapters/storage/sqlite`
- **Acceptance:**
  - New columns appear in the `action_items` `CREATE TABLE` statement.
  - `scanActionItem` reads both columns into `domain.ActionItem.Owner` and `domain.ActionItem.DropNumber`.
  - Insert + update SQL include both columns. Existing tests with empty `Owner` + zero `DropNumber` still pass.
  - One new test in `repo_test.go` writes `Owner: "STEWARD"`, `DropNumber: 5`, reads back, asserts equality.
  - New index `idx_action_items_project_owner_drop_number` present on the `action_items` table.
  - **Pre-MVP rule honored:** no `ALTER TABLE` migration, no SQL backfill — dev fresh-DBs.
  - `mage test-pkg ./internal/adapters/storage/sqlite` green.
  - **DB action:** DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci` for this droplet (schema change).
- **Blocked by:** 3.C.1

---

### Droplet 3.C.3 — `principal_type: steward` enum value + auth-layer state-lock on `owner = STEWARD` items

The biggest semantic droplet in Unit C. Three coupled changes, all in the auth path; splitting them risks an intermediate compile state where the enum recognizes `steward` but no enforcement rejects non-steward state transitions on STEWARD-owned items, leaving a silent gap. Land them in one droplet.

- **Paths:**
  - `internal/domain/auth_request.go` — extend `normalizeAuthRequestPrincipalType` at `:596-608` (verified: 4-case switch today: `""/"user" → "user"; "agent" → "agent"; "service"/"system" → "service"; default → ErrInvalidActorType`) to add a fifth case: `"steward" → "steward", nil`. Update `NewAuthRequest` validation block at `:393-405`: today the block treats `principal_type == "agent"` specially (must carry a `principalRole`); `steward` follows the same "non-empty role required" pattern as `agent` (orchestrator-only role accepted; reject any other role). Update doc comment + table-driven tests in `auth_request_test.go` to assert `steward` is accepted, persists round-trip on the `AuthRequest` struct, and that `principal_role: orchestrator` is required when `principal_type == "steward"` (any other role rejects with `ErrInvalidAuthRequestRole`).
  - `internal/app/auth_requests.go` — propagate the new principal_type through `AuthSession.PrincipalType` field (already a string at `:60`; no shape change needed). Verify `AuthSessionIssueInput.PrincipalType` (`:35`) and `IssueAuthSession` correctly pass `"steward"` through to the issued session.
  - `internal/adapters/server/common/app_service_adapter_mcp.go` — add a new helper `assertOwnerStateGate(ctx, item)` invoked from `MoveActionItem` at `:728` and `MoveActionItemState` at `:744`, BEFORE the `withMutationGuardContext` already in place (or interleaved with it — the auth gate consults the loaded item's `Owner` field, which means the helper must run AFTER the action item is fetched but BEFORE the move SQL fires; on `MoveActionItemState` the existing `a.service.GetActionItem` call at `:760` is the natural fetch point — call the gate immediately after). The helper:
    1. Reads the authenticated caller from the context (the `domain.AuthenticatedCaller` set by `withMutationGuardContext` at `:1852`, plus the session's `PrincipalType` — which today the caller does not carry into the context, so this droplet **adds `PrincipalType` to `domain.AuthenticatedCaller`**, mirroring how Drop 2.3 added `Role` to `ActionItem`. The helper at `internal/domain/authenticated_caller.go:8-13` is currently `{PrincipalID, PrincipalName, PrincipalType ActorType, SessionID}` — `PrincipalType` is `ActorType` (user/agent/system) not the auth-request principal_type. Add a NEW field `AuthPrincipalType string` carrying the auth-request principal_type value (`user|agent|service|steward`), distinct from the existing `PrincipalType ActorType` actor-class field. Both fields are needed: `ActorType` is the user/agent/system attribution axis used by `change_events` + `created_by_type` columns; `AuthPrincipalType` is the auth-model principal-class axis used by the new STEWARD gate.).
    2. If `item.Owner == "STEWARD"` and the caller's `AuthPrincipalType != "steward"`, return `ErrAuthorizationDenied` wrapped with a context message (`"action item %q is owned by STEWARD; only steward-principal sessions can move state"`).
    3. Otherwise return nil.
  - `internal/adapters/server/common/auth.go` — extend the `MutationAuthorizer` interface contract: today `AuthorizeMutation` returns `domain.AuthenticatedCaller` with three fields. Verify the implementations populate the new `AuthPrincipalType` field from the underlying session's `PrincipalType` string. `internal/app/auth_requests.go` `ValidateAuthSession` at the existing surface returns `ValidatedAuthSession.Session.PrincipalType` — thread that into the caller.
  - **Tests in `internal/adapters/server/common/app_service_adapter_mcp_test.go`** (or whichever file already houses MoveActionItem tests — Read first):
    1. `MoveActionItemState` with `Owner = "STEWARD"` and caller `AuthPrincipalType = "agent"` → returns `ErrAuthorizationDenied`. The state DOES NOT change. Verify by re-fetching the action item.
    2. `MoveActionItemState` with `Owner = "STEWARD"` and caller `AuthPrincipalType = "steward"` → succeeds. State changes.
    3. `MoveActionItemState` with `Owner = ""` (empty owner) and caller `AuthPrincipalType = "agent"` → succeeds (gate only fires on `Owner == "STEWARD"`).
    4. `MoveActionItem` (column-level move, not state-level) on `Owner = "STEWARD"` with `AuthPrincipalType = "agent"` → must also reject (the state-lock applies to ANY `LifecycleState` transition, and `MoveActionItem` is the column-move path that can change state when the destination column maps to a different lifecycle).
    5. `UpdateActionItem` (description/details/metadata) on `Owner = "STEWARD"` with `AuthPrincipalType = "agent"` → SUCCEEDS. The gate is state-only; drop-orchs retain content-edit rights per § 19.3 bullet 7 line 1659 ("drop-orchs keep `create` + `update(description/details/metadata)` permissions").
    6. `CreateActionItem` with `parent_id = <STEWARD persistent parent>` and caller `AuthPrincipalType = "agent"` → SUCCEEDS. Drop-orchs create children under STEWARD parents (the level_2 findings) — that's the auto-generation pattern.
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/common`
- **Acceptance:**
  - `normalizeAuthRequestPrincipalType` accepts `"steward"` and returns it canonically; rejects unknown still falls through to `ErrInvalidActorType`.
  - `NewAuthRequest` requires `principal_role = "orchestrator"` when `principal_type = "steward"`; rejects any other role with `ErrInvalidAuthRequestRole`.
  - `domain.AuthenticatedCaller` gains an `AuthPrincipalType string` field. `NormalizeAuthenticatedCaller` (`internal/domain/authenticated_caller.go:16-34`) trims it. The mutation-guard context wiring at `internal/adapters/server/common/app_service_adapter_mcp.go:1852` populates the new field from the resolved session.
  - `MoveActionItem` and `MoveActionItemState` reject state transitions on `Owner = "STEWARD"` items when caller's `AuthPrincipalType != "steward"` with `ErrAuthorizationDenied`. Other mutation paths (`UpdateActionItem`, `CreateActionItem` for child creation, `ReparentActionItem`) are NOT gated.
  - All 6 test cases above pass in their respective `_test.go` files.
  - `mage test-pkg ./internal/domain`, `mage test-pkg ./internal/app`, `mage test-pkg ./internal/adapters/server/common` all green.
  - `mage ci` green at the unit boundary after this droplet.
  - **DB action:** DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci` (auth_request principal_type accepted set widens, although no schema column changes — defensive).
- **Blocked by:** 3.C.2

---

### Droplet 3.C.4 — Template `[child_rules]` consumer: STEWARD level_2 + refinements-gate auto-generation hook

Hard-blocked on Unit B's final droplet that lands the runtime `[child_rules]` engine. Consumer-side coding only — Unit B owns the rule-engine plumbing.

- **Paths:**
  - **Default template TOML (Unit B owns the file location; expected `templates/builtin/default.toml` per § 19.3):** Add the auto-generation `[child_rules]` entry. Schema is Unit B's; Unit C's contribution is the rule body. Sketch (final shape lands once Unit B's parser surfaces; orchestrator wires at synthesis):
    ```toml
    [[child_rules]]
    # Trigger: a level_1 plan-kind action item is created under the project root with metadata.drop_number set
    parent_kind = "plan"
    parent_filter = { parent_id = "{project_root_id}", drop_number_required = true }
    # Body: 6 children to materialize
    [[child_rules.spawns]]
    kind = "action_item"  # plain, not a drop / segment / confluence (stays under STEWARD persistent parent, not in this drop's tree)
    parent_id_lookup = "owner=STEWARD,title=DROP_${N}_HYLLA_FINDINGS-parent"
    title_template = "DROP_${N}_HYLLA_FINDINGS"
    owner = "STEWARD"
    drop_number = "${N}"
    description_template = "Level_2 findings drop for Drop ${N}'s Hylla feedback. Drop-orch populates from subagent closing-comment ## Hylla Feedback sections..."
    structural_type = ""  # explicitly empty per § 19.3 line 1639 (level_2 findings are plain action_item, no structural classification)
    persistent = false
    dev_gated = false
    # ...4 more spawns for LEDGER_ENTRY, WIKI_CHANGELOG_ENTRY, REFINEMENTS_RAISED, HYLLA_REFINEMENTS_RAISED — same shape, different STEWARD persistent parent
    [[child_rules.spawns]]
    # The refinements-gate inside the drop's own tree
    kind = "plan"
    parent_id = "{this_drop_id}"
    title_template = "DROP_${N}_REFINEMENTS_GATE_BEFORE_DROP_${N+1}"
    owner = "STEWARD"
    drop_number = "${N}"
    structural_type = "confluence"  # has non-empty blocked_by per § 19.3 line 1633
    blocked_by_lookup = "every_other_drop_${N}_item + the_5_level_2_findings_just_created"
    description_template = "STEWARD-owned refinements gate. Blocks Drop ${N}'s level_1 closure until STEWARD closes it after working the per-drop refinements pass..."
    ```
  - **Consumer-side Go code:** the rule engine fires the `child_rules.spawns` entries through `app.CreateActionItem`. Unit C verifies the resolution paths for the two non-trivial templates:
    1. `parent_id_lookup = "owner=STEWARD,title=..."` — finds the STEWARD persistent parent (`HYLLA_FINDINGS` / `LEDGER` / etc.) by querying `action_items WHERE owner = 'STEWARD' AND title = ?`. The new `idx_action_items_project_owner_drop_number` index (Droplet 3.C.2) covers this query. **New repository method:** `Repository.FindActionItemByOwnerAndTitle(ctx, projectID, owner, title) (domain.ActionItem, error)` in `internal/app/ports.go` + SQLite implementation.
    2. `blocked_by_lookup = "every_other_drop_N_item + 5_level_2_findings"` — the resolver runs at rule-fire time AFTER the 5 level_2 findings have been created (rule-engine ordering: spawn the findings first, then spawn the gate with a resolved blocked_by list). **New repository method:** `Repository.ListActionItemsByDropNumber(ctx, projectID, dropNumber) ([]domain.ActionItem, error)`. Same index covers it.
  - **Tests:** `internal/app/auto_generate_steward_test.go` (new file) — table-driven tests for the consumer-side resolution logic. Drives the rule-engine entry point through a fake repository. Cases:
    - Numbered drop `N=3` creation → 5 level_2 findings created under correct STEWARD parents + refinements-gate created with correct `blocked_by` covering every Drop 3 item + the 5 findings just spawned.
    - Numbered drop `N=3` creation when `Owner = "STEWARD"` is missing on a persistent parent → rule-engine returns a clear error (the auto-generator fails fast; STEWARD persistent parents must exist before any numbered drop spawns).
    - Non-numbered drop creation (`drop_number = 0`) → rule does NOT fire. Auto-gen is gated on `drop_number_required`.
    - Refinements-gate `blocked_by` correctness: the gate's `blocked_by` enumerates every action item with `drop_number = N` (excluding itself + including the 5 just-created findings).
- **Packages:** `internal/app`, `internal/adapters/storage/sqlite`, plus whatever package Unit B lands the `[child_rules]` engine in.
- **Acceptance:**
  - The 5 STEWARD level_2 findings auto-create on numbered-drop creation, each with the correct `Owner = "STEWARD"`, `DropNumber = N`, parented under the right STEWARD persistent parent.
  - The refinements-gate auto-creates inside the drop's tree with the correct `blocked_by` list.
  - Numbered drops that ARE NOT level_1 (i.e., a sub-`plan` 3 levels deep) do NOT trigger the rule — the rule fires only when the parent is the project root (template encodes this via `parent_filter`).
  - Tests cover the 4 cases above and pass.
  - `mage test-pkg ./internal/app` green.
  - `mage ci` green at the unit boundary after this droplet.
  - **DB action:** DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci` (rule-engine integration may add new rows on existing fixtures; safer to fresh-DB).
- **Blocked by:** 3.C.3, 3.B.<final-rule-engine-droplet> (orch wires the second blocker at synthesis once Unit B's PLAN.md surfaces the specific droplet ID).

---

### Droplet 3.C.5 — MCP `owner` + `drop_number` fields on action-item create/update/get + snapshot serialization

Mirrors Drop 2.5 (`Role` field MCP plumbing, `workflow/drop_2/PLAN.md:118-128`) and 2.6 (`Role` snapshot serialization, `:130-142`). Bundled because they share request/response struct edits and the snapshot work is small enough to ride with the MCP plumbing.

- **Paths:**
  - `internal/adapters/server/common/mcp_surface.go` — add `Owner string` and `DropNumber int` to `CreateActionItemRequest`, `UpdateActionItemRequest`, and the response shape.
  - `internal/adapters/server/common/app_service_adapter_mcp.go` — thread `Owner` + `DropNumber` through `CreateActionItem` (around `:620`, mirror Drop 2.5's `Role` plumbing) and `UpdateActionItem` (around `:661`).
  - `internal/adapters/server/mcpapi/extended_tools.go` — add `mcp.WithString("owner", ...)` and `mcp.WithNumber("drop_number", ...)` to the `till.action_item` tool's create + update operation schemas; thread parsed values into the `Create`/`Update` request.
  - `internal/adapters/server/mcpapi/extended_tools_test.go` — add a test case asserting owner + drop_number round-trip through MCP.
  - `internal/app/snapshot.go` — add `Owner string` (`json:"owner,omitempty"`) and `DropNumber int` (`json:"drop_number,omitempty"`) to `SnapshotActionItem` (struct at `:57-84`, fields go after `Role` at `:63`); thread the fields through `snapshotActionItemFromDomain` at `:1060` and `(t SnapshotActionItem) toDomain()`.
  - `internal/app/snapshot_test.go` (or wherever the round-trip test lives — Drop 2.6's plan referenced it) — extend round-trip to cover `Owner = "STEWARD"`, `DropNumber = 5`.
- **Packages:** `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/app`
- **Acceptance:**
  - `till.action_item(operation=create, owner=..., drop_number=...)` accepts and persists both fields; `operation=get` returns them.
  - `till.action_item(operation=update, owner=..., drop_number=...)` updates both fields on an existing item.
  - Empty owner + zero drop_number round-trip cleanly (`omitempty` drops the JSON keys).
  - Snapshot round-trip preserves both fields across all values; old `v5` snapshots load forward-compatibly (encoding/json ignores unknown keys).
  - **No `SnapshotVersion` bump required** — `omitempty` covers the legacy-format compatibility per Drop 2.6's precedent.
  - Test coverage: create with owner + drop_number, get returns them, update mutates them, snapshot round-trip.
  - `mage test-pkg ./internal/adapters/server/common`, `mage test-pkg ./internal/adapters/server/mcpapi`, `mage test-pkg ./internal/app` all green.
  - **DB action:** NONE (data-shape only; schema lands in 3.C.2).
- **Blocked by:** 3.C.2

**Parallel-with notes:** Droplet 3.C.5 touches `internal/adapters/server/common/app_service_adapter_mcp.go` — same file as 3.C.3's auth-gate edits. Same-package compile race per the standard rule, hence ordered AFTER 3.C.3, NOT in parallel. 3.C.4 (rule-engine consumer) does not edit `app_service_adapter_mcp.go`, so 3.C.4 + 3.C.5 are package-disjoint and could parallel — but since 3.C.4 is hard-blocked on Unit B and 3.C.5 is not, in practice 3.C.5 lands before 3.C.4 and 3.C.4 is `Blocked by: 3.C.5` to keep ordering deterministic. Orch decides at synthesis whether to relax the 3.C.5 → 3.C.4 ordering if Unit B lands earlier.

---

### Droplet 3.C.6 — Integration tests: drop-orch flow + STEWARD flow + refinements-gate close path

The end-to-end verification droplet. Lands last because it consumes every prior droplet's deliverable. Pure integration tests, no production code edits.

- **Paths:**
  - `internal/adapters/server/mcpapi/handler_integration_test.go` (or a new `..._steward_integration_test.go` if the file is too large to extend cleanly — Read first to decide) — add three end-to-end tests:
    1. **Drop-orch creates a numbered drop, auto-gen fires, drop-orch can edit findings descriptions but cannot move state.** Setup: project root + 5 STEWARD persistent parents seeded in the test fixture. Issue an agent-principal session. Create `DROP_3` as a level_1 plan with `drop_number = 3`. Assert: 5 level_2 findings auto-created under STEWARD parents; refinements-gate auto-created in DROP_3's tree with correct blocked_by. Drop-orch session calls `till.action_item(operation=update, action_item_id=<DROP_3_HYLLA_FINDINGS>, description=...)` → SUCCEEDS. Drop-orch session calls `till.action_item(operation=move_state, action_item_id=<DROP_3_HYLLA_FINDINGS>, state=complete)` → REJECTED with `ErrAuthorizationDenied`.
    2. **STEWARD principal session can move state on STEWARD-owned items.** Setup: same as case 1, plus a steward-principal session. STEWARD calls `till.action_item(operation=move_state, action_item_id=<DROP_3_HYLLA_FINDINGS>, state=complete)` → SUCCEEDS.
    3. **Refinements-gate close-path correctness.** STEWARD closes the 5 level_2 findings + every other Drop 3 item, then closes `DROP_3_REFINEMENTS_GATE_BEFORE_DROP_4` → SUCCEEDS (every blocker resolved). STEWARD then attempts to close `DROP_3` (level_1) → SUCCEEDS (refinements-gate is closed; parent-blocks-on-incomplete-child rule satisfied per Drop 1's always-on rule).
  - `internal/app/auth_requests_test.go` — extend the existing auth-request creation tests to cover: creating an auth request with `principal_type = "steward"` + `principal_role = "orchestrator"` → succeeds; with `principal_type = "steward"` + `principal_role = "builder"` → rejected with `ErrInvalidAuthRequestRole`.
- **Packages:** `internal/adapters/server/mcpapi`, `internal/app`
- **Acceptance:**
  - All 3 end-to-end tests + 2 auth-request unit tests pass.
  - The drop-close sequence in PLAN.md § 15.7 lines 1278-1287 is exercised end-to-end (modulo the per-drop MD-write side that lives outside Tillsyn).
  - `mage test-pkg ./internal/adapters/server/mcpapi` and `mage test-pkg ./internal/app` green.
  - **`mage ci` green at the unit boundary** — the final unit-C verification gate before merge.
  - **DB action:** DELETE `~/.tillsyn/tillsyn.db` BEFORE running `mage ci`.
- **Blocked by:** 3.C.4, 3.C.5

---

## Notes

### Architectural Decisions (Locked at Planner Time, Pending Dev Sign-Off at Synthesis)

1. **`Owner` is a first-class string field, NOT a closed enum.** Per the orchestrator-locked architectural decision in the spawn brief. `STEWARD` is the only value the auth gate currently keys on, but other principal-name owners (e.g., `DEV` for human-only items) are permitted for future template-defined owned kinds.
2. **`DropNumber` is a first-class int field, NOT a metadata JSON key.** Mirrors Drop 2.3's `Role` precedent. Read frequency at rule-fire + cross-row-query time justifies the column-with-index over JSON-decode-every-row. Open question routed to dev: confirm or push back to metadata JSON.
3. **`AuthPrincipalType` is a NEW field on `domain.AuthenticatedCaller`, distinct from the existing `PrincipalType ActorType`.** The two axes are orthogonal: `ActorType` is user/agent/system attribution (used by `change_events.actor_type`, `created_by_type`); `AuthPrincipalType` is the auth-model class (`user|agent|service|steward`) used by the new STEWARD gate. Conflating them would force the new `steward` value into `ActorType`, which would ripple into every `actor_type` column and `created_by_type` column — out of scope for Unit C and architecturally wrong (those columns describe who did the work, not which auth class signed the request).
4. **The auth gate fires ONLY on state-transition mutations (`MoveActionItem`, `MoveActionItemState`).** Drop-orchs retain `UpdateActionItem`, `CreateActionItem`, `ReparentActionItem` rights on STEWARD-owned items per § 19.3 bullet 7 line 1659. This matches the pre-Drop-3 honor-system semantics that Unit C codifies — drop-orchs populate findings descriptions, STEWARD owns state.
5. **Refinements-gate `blocked_by` is dynamic at rule-fire time, NOT static at create.** The auto-generator resolves "every other Drop N item" by querying `action_items WHERE drop_number = N` AT THE MOMENT the gate is created. Items that come into existence AFTER the gate is created (e.g., a mid-drop refinement plan-item) are NOT auto-added to the gate's blocked_by — drop-orch must manually update the gate's blocked_by list, OR the rule engine could later support rule-fire-on-drop-close (out of scope for Drop 3). **Open question routed to dev: dynamic-at-create + manual-update vs rule-fire-on-every-new-drop-N-child?** Recommended (locked-in at planner time): dynamic-at-create + manual-update. Rationale: simplest possible rule-engine semantics; matches the honor-system today (drop-orch curates the gate's blocked_by manually); fewer rule-engine triggers = fewer rule-engine bugs.

### Why 6 Droplets, Not 4 or 8

- 4 droplets would force bundling schema + auth-enforcement (3.C.2 + 3.C.3) — violates same-droplet-no-package-mixing for SQLite vs auth path edits. Or would force bundling rule-engine + tests (3.C.4 + 3.C.6) — violates the "one acceptance criterion per droplet" granularity rule and creates a too-big build target.
- 8 droplets would split `principal_type: steward` from the auth-gate enforcement, leaving a gap where the enum recognizes `steward` but no enforcement consults the value. The two MUST land atomically.
- 6 is the natural granularity: 1 (struct + validation) + 1 (schema) + 1 (auth, atomic) + 1 (rule-engine consumer) + 1 (MCP/snapshot plumbing) + 1 (integration tests).

### Same-Package Compile Race Wiring

- 3.C.1 and 3.C.3 both touch `internal/domain` — but 3.C.1 edits `action_item.go` (struct fields) and 3.C.3 edits `auth_request.go` + `authenticated_caller.go`. No file-level overlap. 3.C.3's `Blocked by: 3.C.2` chain transitively orders 3.C.1 → 3.C.3, so no parallel build risk.
- 3.C.3 and 3.C.5 both touch `internal/adapters/server/common/app_service_adapter_mcp.go`. Hard same-file race. 3.C.5 is `Blocked by: 3.C.3` (transitively via 3.C.2 → 3.C.3 → 3.C.5).
- 3.C.4 and 3.C.5 are package-disjoint at the file level but share `internal/app` at the package level — both can compile in parallel against different files (`auto_generate_steward.go` vs `snapshot.go`). But 3.C.4 is hard-blocked on Unit B's rule-engine, so the parallel opportunity is academic; orch serializes 3.C.5 → 3.C.4 for deterministic ordering.

### Hylla Feedback

None — Hylla answered everything needed for the planning pass via Read/LSP/grep on the working tree (Hylla today only indexes Go committed code, and the planning pass relied primarily on freshly-Drop-2-merged Go source + open MD specs for which Hylla has no value-add). Pure-MD specs (`PLAN.md`, `STEWARD_ORCH_PROMPT.md`, sibling `workflow/drop_2/PLAN.md`) are non-Go and explicitly out of Hylla's scope per `feedback_hylla_go_only_today.md`. Auth-domain Go code was read directly via `Read` on `internal/domain/auth_request.go`, `internal/domain/authenticated_caller.go`, `internal/domain/action_item.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/storage/sqlite/repo.go` — these reads grounded every concrete line-number citation in this PLAN.md. No Hylla queries were attempted because the orchestrator's spawn-brief described Unit C scope precisely enough that direct reads on the cited files were the most efficient evidence path.
