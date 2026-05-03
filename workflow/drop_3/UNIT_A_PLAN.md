# DROP_3 — Unit A — Cascade Vocabulary Foundation

## Scope

Land `metadata.structural_type` as a closed 4-value enum (`drop | segment | confluence | droplet`) on every non-project action item, mirroring Drop 2.3's `Role` cascade end-to-end: pure domain enum + parser, first-class field on `ActionItem`, app-service plumbing, SQLite column + scan/insert/update, MCP request/handler/tool-schema plumbing, snapshot round-trip serialization. Add the `## Cascade Vocabulary` glossary section to `main/WIKI.md` as the single canonical source for waterfall semantics. Teach `~/.claude/agents/go-qa-falsification-agent.md` the five new attack vectors (droplet-with-children, segment overlap-without-blocked_by, empty-blocked_by confluence, partial-upstream-coverage confluence, role/structural_type contradictions). Pre-MVP rule: no migration logic — dev fresh-DBs `~/.tillsyn/tillsyn.db` after droplet 3.A.3 lands. Validation diverges from `Role` in one critical way: `structural_type` is REQUIRED on every non-project node (empty rejects), per PLAN.md § 19.3 "Default is NOT inferred — the creator chooses explicitly."

## Droplets

### Droplet 3.A.1 — Domain `StructuralType` enum + parser + tests

- **State:** todo
- **Paths:**
  - `internal/domain/structural_type.go` (NEW)
  - `internal/domain/structural_type_test.go` (NEW)
  - `internal/domain/errors.go` (add `ErrInvalidStructuralType`)
- **Packages:** `internal/domain`
- **Acceptance:**
  - New file `internal/domain/structural_type.go` (~50-60 LOC) defines:
    - `type StructuralType string` with four typed constants `StructuralTypeDrop = "drop"`, `StructuralTypeSegment = "segment"`, `StructuralTypeConfluence = "confluence"`, `StructuralTypeDroplet = "droplet"`.
    - `validStructuralTypes` package-level slice listing all four values in declaration order.
    - `IsValidStructuralType(StructuralType) bool` — uses `slices.Contains` with trim+lowercase normalization. Empty string returns `false` (matching `IsValidRole`'s contract).
    - `NormalizeStructuralType(StructuralType) StructuralType` — trim + lowercase. Whitespace-only collapses to empty.
    - `ParseStructuralTypeFromDescription(string) (StructuralType, error)` — regex `(?m)^StructuralType:\s*([a-z-]+)\s*$`. Empty desc → `("", nil)`. Unknown value → `("", ErrInvalidStructuralType)`. Match → `(typed-constant, nil)`. (Character class is `[a-z-]+` not `[a-z0-9-]+` because none of the four values contain digits — keeps the regex narrower than `Role`'s.)
  - `internal/domain/errors.go` adds `ErrInvalidStructuralType = errors.New("invalid structural type")` alongside `ErrInvalidRole`.
  - New `structural_type_test.go` mirrors `role_test.go` exactly: three table-driven `t.Parallel()` tests — `TestIsValidStructuralType`, `TestNormalizeStructuralType`, `TestParseStructuralTypeFromDescription`. Each StructuralType value gets an explicit case in each test. Whitespace, case, and empty-string carve-outs match `role_test.go`'s shape line-for-line.
  - Coverage: `mage test-pkg ./internal/domain` green; structural_type.go ≥ 90% covered.
- **Blocked by:** —

### Droplet 3.A.2 — `StructuralType` field on `ActionItem` + validation in `NewActionItem`

- **State:** todo
- **Paths:**
  - `internal/domain/action_item.go`
  - `internal/domain/domain_test.go`
- **Packages:** `internal/domain`
- **Acceptance:**
  - `ActionItem` struct gains `StructuralType StructuralType` field (placed after `Role`, mirroring Drop 2.3's placement at line 33).
  - `ActionItemInput` struct gains the same `StructuralType StructuralType` field with a doc comment matching `Role`'s style but stating: "StructuralType MUST be a member of the closed StructuralType enum. Empty is rejected — `NewActionItem` returns `ErrInvalidStructuralType`."
  - `NewActionItem` adds validation BETWEEN the existing `Role` validation block (lines 150-158) and the lifecycle-state initialization. Pattern:
    - `in.StructuralType = NormalizeStructuralType(in.StructuralType)`
    - `if in.StructuralType == "" { return ActionItem{}, ErrInvalidStructuralType }` ← **diverges from Role** (Role permits empty).
    - `if !IsValidStructuralType(in.StructuralType) { return ActionItem{}, ErrInvalidStructuralType }`
  - `NewActionItem` return statement assigns `StructuralType: in.StructuralType` (line ~201 in current file, alongside `Role`).
  - `domain_test.go` adds `TestNewActionItemStructuralTypeValidation` (mirroring `TestNewActionItemRoleValidation` at line 212) with cases: each of 4 values accepted; empty rejected with `ErrInvalidStructuralType`; whitespace-only rejected; unknown value rejected. Existing `TestNewActionItemRoleValidation` test cases must be UPDATED to supply a valid `StructuralType` (e.g. `StructuralTypeDroplet`) since `NewActionItem` now requires it — otherwise prior tests break with the new required-field validation.
  - Sweep `domain_test.go` for every other `domain.NewActionItem(...)` / `NewActionItem(ActionItemInput{...})` call site that omits `StructuralType` — each must be updated to supply a value, or each test must explicitly exercise the new `ErrInvalidStructuralType` rejection path.
  - `mage test-pkg ./internal/domain` green.
- **Blocked by:** 3.A.1 (same Go package — package-level lock; new symbols `StructuralType`, `IsValidStructuralType`, `NormalizeStructuralType`, `ErrInvalidStructuralType` must exist before this droplet compiles)

### Droplet 3.A.3 — SQLite `structural_type` column + scan + INSERT/UPDATE plumbing

- **State:** todo
- **Paths:**
  - `internal/adapters/storage/sqlite/repo.go`
  - `internal/adapters/storage/sqlite/repo_test.go`
- **Packages:** `internal/adapters/storage/sqlite`
- **Acceptance:**
  - `action_items` CREATE TABLE statement (line 168-197) gains a new column declaration: `structural_type TEXT NOT NULL DEFAULT ''` placed immediately after `role TEXT NOT NULL DEFAULT '',` (line 174). DDL-default `''` matches the `role` precedent — code-level validation in `NewActionItem` (3.A.2) is the canonical gate; the empty default exists only so legacy rows from older DBs scan without panicking. Pre-MVP rule (per `feedback_no_migration_logic_pre_mvp.md`): no `ALTER TABLE` migration; dev fresh-DBs the database. Do NOT add an `ALTER TABLE ... ADD COLUMN structural_type` statement under the existing migration-block pattern at lines 515/548.
  - INSERT statement at line 1252-1257: column list adds `structural_type` after `role`; VALUES tuple adds one more `?`; bind-arg list (lines 1259-1284) adds `string(t.StructuralType)` after `string(t.Role)`.
  - UPDATE statement at line 1346-1351: `SET` clause adds `structural_type = ?` after `role = ?`; bind-arg list (line 1352-1369) adds `string(t.StructuralType)` after `string(t.Role)`.
  - Same pattern applied to the 2 other INSERT statements at lines 1414 and 1452 and the soft-create-or-update path at line 2500 — each touches the same `action_items` row writes.
  - `scanActionItem` (or the relevant scanner around line 2806-2892): adds `structuralTypeRaw string` to the local var block, adds `&structuralTypeRaw` to the `Scan()` call alongside `&roleRaw`, and assigns `t.StructuralType = domain.StructuralType(structuralTypeRaw)` after the existing `t.Role = domain.Role(roleRaw)` at line 2846. The `t.StructuralType == ""` case is left untouched — domain-layer validation (3.A.2) catches it on next write.
  - SELECT column-lists in `getActionItemByID` and any `SELECT * FROM action_items` query must include `structural_type` in the proper ordinal position (matching the new column placement after `role`). Sweep all `SELECT ... FROM action_items` and `RETURNING ...` clauses in `repo.go` to confirm column count consistency.
  - `repo_test.go` adds a round-trip test asserting that an `ActionItem` with each of the four `StructuralType` values persists and rehydrates cleanly via `CreateActionItem` → `GetActionItem`. Existing repo tests that construct `domain.ActionItem` via `NewActionItem` need a valid `StructuralType` supplied — otherwise the domain-layer validation introduced in 3.A.2 fails at fixture-setup time.
  - `mage test-pkg ./internal/adapters/storage/sqlite` green.
- **Blocked by:** 3.A.2 (domain field must exist before the repo can scan/bind it)

### Droplet 3.A.4 — App-service + MCP plumbing for `structural_type`

- **State:** todo
- **Paths:**
  - `internal/app/service.go`
  - `internal/adapters/server/common/mcp_surface.go`
  - `internal/adapters/server/common/app_service_adapter_mcp.go`
  - `internal/adapters/server/mcpapi/extended_tools.go`
  - `internal/adapters/server/mcpapi/extended_tools_test.go`
- **Packages:** `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
- **Acceptance:**
  - `internal/app/service.go`:
    - `CreateActionItemInput` (line 404) gains `StructuralType domain.StructuralType` after `Role`. Doc comment notes empty is REJECTED (diverges from role).
    - `UpdateActionItemInput` (line 429) gains `StructuralType domain.StructuralType`. Doc comment notes empty preserves prior value (mirrors role's update-semantics, since update is partial — required-on-create, optional-on-update).
    - `CreateActionItem` (line 574) threads `StructuralType: in.StructuralType` into `domain.NewActionItem(...)` call (line 580).
    - `UpdateActionItem` (line ~775-794): adds a sibling block to the existing `Role` update at line 784-794 — if `domain.NormalizeStructuralType(in.StructuralType) != ""` then validate against `IsValidStructuralType` and assign to `actionItem.StructuralType`. Empty preserves prior value.
  - `internal/adapters/server/common/mcp_surface.go`:
    - `CreateActionItemRequest` (line 57) gains `StructuralType string` after `Role`. Doc comment: required closed-enum value (`drop|segment|confluence|droplet`); empty rejects with `ErrInvalidStructuralType`.
    - `UpdateActionItemRequest` (line 78) gains `StructuralType string`. Doc comment: empty preserves prior value.
  - `internal/adapters/server/common/app_service_adapter_mcp.go`:
    - `CreateActionItem` (line 666-684): threads `StructuralType: domain.StructuralType(strings.TrimSpace(in.StructuralType))` into the `app.CreateActionItemInput{}` literal alongside `Role`.
    - `UpdateActionItem` (line 708-720): same — threads `StructuralType:` into `app.UpdateActionItemInput{}`.
  - `internal/adapters/server/mcpapi/extended_tools.go`:
    - Inline anonymous-struct `args` (line 860): adds `StructuralType string \`json:"structural_type"\`` field after `Role` (line 866).
    - `case "create"` block (line ~1033): adds `StructuralType: args.StructuralType,` to the `common.CreateActionItemRequest{}` literal.
    - `case "update"` block (line ~1092): adds `StructuralType: args.StructuralType,` to the `common.UpdateActionItemRequest{}` literal.
    - Tool schema declaration (line 1370): adds `mcp.WithString("structural_type", mcp.Description("Required for operation=create — closed enum: drop|segment|confluence|droplet (waterfall metaphor — see WIKI.md §Cascade Vocabulary). Empty rejects on create. Empty preserves prior value on update."), mcp.Enum("drop", "segment", "confluence", "droplet"))` after the `mcp.WithString("role", ...)` line.
    - Legacy `till.create_task` (line 1417) and `till.update_task` (line 1444) tool registrations get the same `mcp.WithString("structural_type", ...)` parameter declaration.
  - `internal/adapters/server/mcpapi/extended_tools_test.go`:
    - `stubExpandedService.CreateActionItem` (line 429): adds rejection logic mirroring the `Role` rejection at lines 431-433: if `args.StructuralType` is empty OR not in the closed enum, return `errors.Join(common.ErrInvalidCaptureStateRequest, domain.ErrInvalidStructuralType)`. Returned `domain.ActionItem` literal (line 435-448) gets `StructuralType: domain.StructuralType(strings.TrimSpace(in.StructuralType))`.
    - `stubExpandedService.UpdateActionItem` (line 455): mirrors the same — empty preserves prior; non-empty must validate or rejects.
  - Add a `TestActionItemMCPRejectsEmptyOrInvalidStructuralType` test that exercises both empty (CONFIRMED rejection on create) and unknown value (CONFIRMED rejection on both create + update) through the MCP boundary.
  - `mage test-pkg ./internal/app && mage test-pkg ./internal/adapters/server/common && mage test-pkg ./internal/adapters/server/mcpapi` all green.
- **Blocked by:** 3.A.2 (domain field must exist before app/common/mcpapi can reference `domain.StructuralType` and `domain.IsValidStructuralType`)

### Droplet 3.A.5 — Snapshot serialization round-trip for `StructuralType`

- **State:** todo
- **Paths:**
  - `internal/app/snapshot.go`
  - `internal/app/snapshot_test.go`
- **Packages:** `internal/app`
- **Acceptance:**
  - `SnapshotActionItem` struct (line 57): adds `StructuralType domain.StructuralType \`json:"structural_type,omitempty"\`` field placed after `Role` (line 63). The `omitempty` tag matches `Role`'s tag — pre-3.A.2 snapshots without the field deserialize cleanly to empty, then 3.A.2's domain validation catches the empty value on next write (matches the legacy-`Kind` fallback pattern at line 1273-1278).
  - `snapshotActionItemFromDomain` (line 1059): adds `StructuralType: t.StructuralType,` to the literal alongside `Role`.
  - `(SnapshotActionItem).toDomain` (line 1267): adds `StructuralType: t.StructuralType,` to the returned `domain.ActionItem` literal alongside `Role`. NO empty-string fallback (unlike `Kind` at lines 1273-1278) — the closed enum has no sensible default; empty is intentionally surfaced so 3.A.2's validation catches it on next mutation.
  - `snapshot_test.go` adds `TestSnapshotActionItemStructuralTypeRoundTripPreservesAllValues` (mirroring `TestSnapshotActionItemRoleRoundTripPreservesAllRoles` at line 442). Four cases — one per enum value — assert domain → snapshot → domain round-trip preserves the typed constant exactly.
  - `mage test-pkg ./internal/app` green.
- **Blocked by:** 3.A.2 (`domain.ActionItem.StructuralType` field must exist before snapshot serializes it)

### Droplet 3.A.6 — `## Cascade Vocabulary` glossary section in WIKI

- **State:** todo
- **Paths:**
  - `main/WIKI.md`
- **Packages:** — (non-Go MD edit)
- **Acceptance:**
  - New `## Cascade Vocabulary` section inserted between `## The Tillsyn Model (Node Types)` (line 19) and `## Level Addressing (0-Indexed)` (line 36). Section structure:
    - **One-paragraph waterfall metaphor** orienting `drop / segment / confluence / droplet`. Cite Tillsyn Cascade branding alignment.
    - **`structural_type` enum reference** stating it is closed (4 values, not customizable initially), mandatory on every non-project node, validated at `till.action_item(operation=create|update)` boundary, default NOT inferred.
    - **Per-value definition + atomicity rules**:
      - `drop` — vertical cascade step; level-1 children of project are always drops; deeper drops are sub-cascades; parallelizable across siblings when path/package blockers allow.
      - `segment` — parallel execution stream within a drop (the fan-out unit); segments within a drop run in parallel; segments across drops coordinate via handoffs; may recurse.
      - `confluence` — merge/integration node pulling work from multiple segments or sibling drops; **MUST have non-empty `blocked_by`** naming every upstream contributor (definitional — empty `blocked_by` is a contradiction).
      - `droplet` — atomic, indivisible leaf action; **MUST have zero children** (any child indicates misclassification → should be segment or drop).
    - **Orthogonality with `metadata.role`** explicitly called out as a separate axis. Worked example: `(structural_type=droplet, role=builder)` is the canonical build leaf; `(structural_type=droplet, role=qa-proof)` is the canonical QA leaf; `(structural_type=confluence, role=…)` describes integration points; etc.
    - **Worked examples** (3-4): a level-1 drop fanning into 3 segments that converge at a confluence; a droplet under a segment with role=builder; a confluence with non-empty `blocked_by` listing every upstream segment; a sub-drop inside a segment when a stream needs its own fan-out.
    - **Single-canonical-source rule** stated explicitly: "This section is the canonical definition for cascade vocabulary. Every other doc — `PLAN.md`, `CLAUDE.md`, `STEWARD_ORCH_PROMPT.md`, agent prompt files, bootstrap skills — holds a pointer to this section, not a duplicate definition. The `plan-qa-falsification` agent attacks any cascade-vocabulary redefinition outside this section."
  - Section uses h2 (`##`) for the section title and h3 (`###`) for sub-sections (matching the rest of `WIKI.md`'s hierarchy at lines 19, 36, 47).
  - No edits to other docs (`PLAN.md`, `CLAUDE.md`, agent files, bootstrap skills) — those cross-doc pointer updates are Unit D's "Per-drop wrap-up cascade vocabulary sweep" scope (PLAN.md § 19.3 line 1651). Strict scope boundary.
  - No `mage` target — pure MD. Verify by re-reading the section + grep for surrounding heading order.
- **Blocked by:** —

### Droplet 3.A.7 — Plan-QA-falsification agent prompt + 5 new attack vectors

- **State:** todo
- **Paths:**
  - `~/.claude/agents/go-qa-falsification-agent.md`
- **Packages:** — (non-Go MD edit, agent prompt file outside `main/`)
- **Acceptance:**
  - `## Go Falsification Attacks` section (line 95-108) gains a new bullet block titled **Cascade-vocabulary attacks (post-Drop-3)** with five sub-bullets, each one a CONFIRMED-counterexample template:
    1. **Droplet-with-children.** Any action item with `structural_type=droplet` whose subtree contains one or more child action items is a misclassification. Real shape is `segment` (if the droplet was meant to fan out) or `drop` (if it was meant to be a vertical cascade step). PASS condition: every droplet has zero children. FAIL: produce the parent UUID + child list.
    2. **Segment path/package overlap without `blocked_by`.** Sibling segments within the same drop sharing any entry in `paths[]` or `packages[]` without an explicit `blocked_by` between them are a race waiting to fire. Same rule as file/package locking on `build` action items, applied at the segment level. PASS: every shared path/package is covered by an explicit `blocked_by`. FAIL: produce the two segment UUIDs + the overlapping path/package list.
    3. **Empty-`blocked_by` confluence.** Any action item with `structural_type=confluence` whose `blocked_by` list is empty is a definitional contradiction — confluences exist to merge upstream contributors, so they must name them. Flag and refuse. PASS: every confluence has non-empty `blocked_by`. FAIL: produce the confluence UUID.
    4. **Confluence with partial upstream coverage.** Any confluence whose `blocked_by` list does NOT name every segment / drop it claims to integrate is a partial-coverage failure — the planner must enumerate every upstream. PASS: every named upstream in description matches a `blocked_by` entry; every `blocked_by` entry is a valid segment / drop UUID. FAIL: produce the missing upstream UUID(s).
    5. **Role / structural_type contradictions.** Each combination has narrow legitimate shapes — flag the following CONFIRMED counterexamples:
       - `role=qa-proof` or `role=qa-falsification` on a non-droplet (QA work is always a leaf droplet).
       - `role=builder` on a confluence (integration is QA / orchestrator work, not builder).
       - `role=planner` on a droplet without a downstream integration target (planning produces decomposition; a planner droplet whose parent has no segments / drops to plan against is misclassified).
       - `role=commit` on anything other than a droplet (commit work is always atomic).
  - Each sub-bullet is detailed enough that a builder running this agent's prompt can reproduce the attack from the bullet alone, without external lookup.
  - At the top of the new attack block, add a one-line pointer: *"Cascade vocabulary canonical: `WIKI.md` §`Cascade Vocabulary`."*
  - The existing `**Plan-level attacks (on planning tasks).**` bullet (line 108) gains a sibling line referencing the new cascade-vocabulary block above it, so falsification agents reading top-down see the new attacks before the older plan-level attacks: "**Cascade-vocabulary attacks (post-Drop-3).** See block above — droplet-with-children, segment overlap-without-blocked_by, empty/partial-coverage confluence, role/structural_type contradictions."
  - No edits to `~/.claude/agents/go-qa-proof-agent.md` (proof QA verifies evidence completeness; falsification QA owns the attack surface).
  - No `mage` target — pure MD. Verify by re-reading the section + sanity-check that all 5 attack-vector keywords from PLAN.md § 19.3 lines 1644-1649 appear in the prompt.
- **Blocked by:** —

## Notes

### Cross-Unit Dependencies (For Orchestrator Synthesis)

- **3.A.1 produces `domain.StructuralType`, `IsValidStructuralType`, `NormalizeStructuralType`, `ErrInvalidStructuralType`.** Every other unit's droplet that imports `internal/domain` and references the structural-type axis must `blocked_by` 3.A.1. Concretely:
  - **Unit B (template system overhaul)** binds `child_rules` and gate rules on the `structural_type` axis. Unit B's first code droplet that consumes `domain.StructuralType` MUST `blocked_by` Unit A's 3.A.1.
  - **Unit C (STEWARD auth + template auto-generation)** auto-creates STEWARD level-2 items with `metadata.owner = STEWARD` AND a `structural_type` value. Unit C droplets touching action-item creation MUST `blocked_by` 3.A.2 (which is the field-on-`ActionItem` droplet).
  - **Unit D (adopter bootstrap + cross-doc sweep)** writes the cascade-vocabulary pointer into `~/.claude/agents/*.md` and into bootstrap skill output. **CONFLICT WARNING:** Unit D's edit to `~/.claude/agents/go-qa-falsification-agent.md` (adding a one-line frontmatter pointer per PLAN.md § 19.3 line 1650) collides with Unit A's 3.A.7 edit to the same file. Orchestrator must wire either `Unit_D_agent_file_droplet blocked_by 3.A.7` (preferred — let Unit A land the attack vectors first, Unit D adds the frontmatter pointer on top) OR merge both edits into a single droplet at synthesis time.
- **WIKI cross-doc pointers are Unit D, not Unit A.** Unit A's 3.A.6 ONLY writes the canonical glossary section in `main/WIKI.md`. Pointer lines in `main/PLAN.md` / `main/CLAUDE.md` / `main/STEWARD_ORCH_PROMPT.md` / `~/.claude/agents/*.md` / bootstrap skills are Unit D's "Per-drop wrap-up cascade vocabulary sweep" scope. Strict boundary — Unit A does not edit those files.

### Architectural Decisions (Confirmed)

- **First-class field, not metadata JSON.** Per PLAN.md § 19.3 + locked dev guidance in `workflow/drop_3/PLAN.md` line 67-68: `StructuralType` is a first-class domain field on `ActionItem`, mirroring Drop 2.3's `Role` field placement. Closed enum at the type-system level, not free-form JSON.
- **Required on create, optional on update.** Diverges from `Role`'s "empty permitted always" pattern. PLAN.md § 19.3 is explicit: "Default is NOT inferred — the creator (planner / orch / dev) chooses explicitly." 3.A.2's `NewActionItem` validation rejects empty; 3.A.4's `UpdateActionItem` preserves prior value on empty (so partial-update doesn't force callers to re-supply).
- **No migration logic.** Per `feedback_no_migration_logic_pre_mvp.md` + `workflow/drop_3/PLAN.md` line 30 + 60-62: dev fresh-DBs `~/.tillsyn/tillsyn.db` after 3.A.3 lands. The PLAN.md § 19.3 bullet at line 1636 ("Retroactive classification of existing action_items via one-shot SQL") is REPLACED with the fresh-DB rule. No `ALTER TABLE`, no SQL backfill, no `till migrate` subcommand.
- **Snapshot omitempty + no fallback.** `SnapshotActionItem.StructuralType` uses `omitempty` so pre-3.A.2 snapshots deserialize cleanly. `(SnapshotActionItem).toDomain` does NOT default an empty StructuralType (unlike `Kind`'s legacy fallback at snapshot.go:1273-1278) — empty surfaces via 3.A.2's validation on next mutation.

### Architectural Questions (Unresolved — Route to Orchestrator)

- **Should 3.A.4 split into separate "common surface" + "mcpapi handler" droplets?** Drop 2.5 bundled them. Bundle keeps the change atomic for one builder. Splitting adds blocker overhead with no clear benefit. Default: keep bundled. Flag for orchestrator if split is preferred.
- **Should the `## Cascade Vocabulary` WIKI section land BEFORE or AFTER the "Do Not Use Other Kinds Today" + "Do Not Use Templates Right Now" sub-sections of `## The Tillsyn Model (Node Types)` (lines 28-34)?** Recommendation: insert as a sibling top-level h2 BETWEEN line 34 (end of "Do Not Use Templates Right Now") and line 36 (start of "Level Addressing"). That places the cascade vocabulary as the section users read AFTER understanding what nodes exist (level 0 = project, deeper = drops in the pre-Drop-2 sense) but BEFORE level addressing — natural pedagogical flow. Confirmed in droplet 3.A.6 acceptance.
- **`StructuralType` capitalization and stored form.** Code uses Go `StructuralType` PascalCase symbol with `string` underlying type whose values are lowercase (`"drop"`, `"segment"`, `"confluence"`, `"droplet"`). MCP / SQLite store the lowercase string form. Matches `Role`'s convention (`RoleBuilder` typed constant carrying value `"builder"`). Confirmed.

### Hylla Feedback

None — Hylla MCP was allowed but I leaned on direct file reads (Read + LSP + native rg via Bash for non-Go MD) since I needed exact line numbers + full surface-area mapping for the description-symbol verification rule. The 7 droplets cite line-precise references to `internal/domain/role.go`, `internal/domain/action_item.go`, `internal/app/service.go`, `internal/app/snapshot.go`, `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/domain/role_test.go`, `internal/domain/domain_test.go`, `internal/app/snapshot_test.go`, `WIKI.md`, `~/.claude/agents/go-qa-falsification-agent.md`. No fallback misses to log — Hylla query would have returned the same surface but for one-shot symbol mapping the LSP `findReferences` from `Role` typed-constant covered the entire cascade in a single query (53 references across 10 files). For symbol-cascade-mapping work like this, `LSP findReferences` may be a more ergonomic starting point than `hylla_refs_find` because it returns line + character precision in a single query; flagging as ergonomic-only signal, not a Hylla bug.
