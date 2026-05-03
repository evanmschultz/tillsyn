# DROP_3 — TEMPLATE CONFIGURATION

**State:** planning
**Blocked by:** DROP_2 (closed at `0a7ba80`)
**Paths (expected):** `internal/domain/`, `internal/app/`, `internal/adapters/storage/sqlite/`, `internal/adapters/server/common/`, `internal/adapters/server/mcpapi/`, `cmd/till/`, `templates/builtin/` (new), `~/.claude/agents/` (adopter bootstrap), `WIKI.md` (cascade glossary)
**Packages (expected):** `internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `cmd/till`, plus a new `internal/templates` package (or equivalent)
**PLAN.md ref:** `main/PLAN.md` § 19.3 — drop 3 — Template Configuration
**Started:** 2026-05-02
**Closed:** —

## Scope

Drop 3 is a **full template-system overhaul**, not an extension. Per `main/PLAN.md` § 19.3:

1. **Cascade vocabulary foundation.** Add `metadata.structural_type` as a closed-enum first-class field on every non-project node. Values: `drop | segment | confluence | droplet` (waterfall metaphor). Orthogonal to `metadata.role` (Drop 2.3) + the closed 12-kind enum (Drop 1.75). Templates `child_rules` and gate rules bind on `structural_type`. **NO retroactive SQL classification migration** — pre-MVP rule: dev fresh-DBs `~/.tillsyn/tillsyn.db` after the field lands.
2. **WIKI glossary as single canonical source.** New `## Cascade Vocabulary` section in `main/WIKI.md` owns the structural_type enum + atomicity rules (droplet has zero children; confluence has non-empty `blocked_by`; segment can recurse) + relationship to `metadata.role` + worked examples. Every other doc holds a pointer, not a duplicate.
3. **Template system overhaul.** Closed TOML schema at `<project_root>/.tillsyn/template.toml` with strict unknown-key rejection + versioned `schema_version`. Template baked into a `KindCatalog` value type at project-creation time (no runtime file lookups). Single `Template.AllowsNesting(parent, child Kind) (bool, reason string)` function as the one validation truth. `[child_rules]` table for auto-create with load-time validator catching unreachable rules / cycles / unknown kinds. Default template at `templates/builtin/default.toml` covering the conceptual prohibitions (closeout-no-closeout-parent, commit-no-plan-child, human-verify-no-build-child, build-qa-*-no-plan-child). Pure Go struct unmarshaling — no string-based DSL. Audit-trail `till.comment` + attention item on every nesting rejection. Existing `KindTemplate` + `KindTemplateChildSpec` + `AllowedParentScopes` + `AllowsParentScope` get **rewritten, not extended**.
4. **Agent binding fields on kind definitions.** `agent_name`, `model`, `effort`, `tools`, `max_tries`, `max_budget_usd`, `max_turns`, `auto_push`, `commit_agent`, `blocked_retries`, `blocked_retry_cooldown`. Drop 4's dispatcher reads these.
5. **Plan-QA-falsification new attack vectors.** Teach the plan-qa-falsification agent + checklist: droplet-with-children, segment path/package overlap without `blocked_by`, empty-`blocked_by` confluence, confluence with partial upstream coverage, role/structural_type contradictions.
6. **Adopter bootstrap updates.** Every `go-project-bootstrap` + `fe-project-bootstrap` skill + every `CLAUDE.md` template inherits the cascade glossary pointer at bootstrap time. WIKI scaffolding pre-fills `## Cascade Vocabulary`. Agent file frontmatter gets a one-line reminder pointing at the WIKI glossary.
7. **STEWARD auth-level state-lock.** New `principal_type: steward` in Tillsyn's auth model — distinct from `agent`. Auth layer rejects state transitions on `metadata.owner = STEWARD` items unless the session is `principal_type=steward`. Drop-orchs keep `create` + `update(description/details/metadata)` perms but cannot move STEWARD items through state. Replaces the pre-Drop-3 honor-system.
8. **Template auto-generation of STEWARD-scope items** on every numbered-drop creation. Template `child_rules` auto-create the 5 level_2 findings drops + the refinements-gate when `DROP_N_ORCH` creates a level_1 numbered drop. Auto-generated items land with `metadata.owner = STEWARD`, `metadata.drop_number = N`, correct `blocked_by` wiring.
9. **Template-defined STEWARD-owned drop kind(s).** Templates allow marking specific kinds as STEWARD-owned. Pairs with the `principal_type: steward` gate.
10. **Per-drop wrap-up cascade vocabulary sweep.** After the rename + enum + template binding land, sweep every lingering `action_item` / `action-item` / `action item` / `ActionItem` string across docs, agent prompts, slash-command files, skill files, memory files. Update `metadata.role` vs `metadata.structural_type` crosswalk wherever docs previously conflated role with kind.

**Out of scope (explicit, per PLAN.md § 19.3):** dispatcher implementation (Drop 4); TUI overhaul (Drop 4.5); cascade dogfooding (Drop 5); escalation (Drop 6); error handling + observability (Drop 7).

**Pre-MVP rules in effect (per memory):**

- No migration logic in Go code, no `till migrate` subcommands, no one-shot SQL scripts. Dev deletes `~/.tillsyn/tillsyn.db` between schema-touching units (`structural_type` field add, STEWARD `metadata.owner` migration, etc.).
- No `CLOSEOUT.md`, no `LEDGER.md` entry, no `WIKI_CHANGELOG.md` entry, no `REFINEMENTS.md` entry, no `HYLLA_FEEDBACK.md` rollup, no `HYLLA_REFINEMENTS.md` rollup. Worklog MDs (this `PLAN.md`, `BUILDER_WORKLOG.md`, `BUILDER_QA_*.md`, `PLAN_QA_*.md`) DO happen.
- Builders run **opus** (not sonnet default) per `feedback_opus_builders_pre_mvp.md`.
- Drop 3 closes when all `main/PLAN.md` § 19.3 checkboxes are checked.
- **Old "drops all the way down" framing is RETIRED** per dev direction 2026-05-02. Use waterfall metaphor + `structural_type` axis.

## Planner

**Decomposition:** 28 droplets, renumbered `3.1` – `3.28`, fanned across 4 work-units (A: cascade vocabulary foundation; B: template system overhaul; C: STEWARD auth + auto-gen; D: adopter bootstrap + cascade-vocabulary doc sweep). Original unit-letter retained inline as `3.A.k` / `3.B.k` / `3.C.k` / `3.D.k` for traceability and `blocked_by` clarity. Builders run **opus** (per `feedback_opus_builders_pre_mvp.md`); not repeated per droplet. Verification target throughout: `mage ci` at unit boundaries, `mage test-pkg <pkg>` per droplet. **Never `mage install`** (dev-only); **never** raw `go test` / `go build` / `go vet` / `go run`.

**Evidence-sourcing process (per CLAUDE.md §"Code Understanding Rules" + QA falsification §1.9):** every builder + every QA subagent spawn for Drop 3 uses **Hylla MCP first** (`mcp__hylla__hylla_search`, `mcp__hylla__hylla_node_full`, `mcp__hylla__hylla_search_keyword`, `mcp__hylla__hylla_refs_find`, `mcp__hylla__hylla_graph_nav`) for committed-Go code understanding. If Hylla does not return the expected result on the first search, exhaust every Hylla search mode — vector / keyword / graph-nav / refs — BEFORE falling back to LSP / Read / Grep. Every fallback miss is recorded in the subagent's closing comment under a `## Hylla Feedback` heading. This rule applies retroactively to Round 3 of any droplet that re-spawns; Round 1 + Round 2 planner spawns surfaced this gap (Round 1 §5.A.12 / C7; Round 2 §1.9) and Round 3 closes it.

### Locked Architectural Decisions (Round 1 Plan-QA Cross-Cutting)

The 8 decisions below were dev-approved on 2026-05-02 and resolve every cross-cutting open question Round 1 plan-QA surfaced. They are baked into the droplet decomposition; do NOT re-route them.

- **L1 — `UpdateActionItem` field-level write guard (Unit C C1).** Source: Unit C falsification CE C1 (silent state-lock bypass via clearing `Owner`). Resolution: when `existing.Owner == "STEWARD"` and caller's `AuthPrincipalType != "steward"`, reject any `UpdateActionItemRequest` whose `Owner` or `DropNumber` differ from existing values. Lands in droplet **3.19** (auth-gate atomic landing).

- **L2 — `principal_type: steward` autent boundary-map (Unit C C3).** Source: Unit C falsification CE C3 (vendored `autent@v0.1.1` declares closed `{user, agent, service}` enum). Resolution: Tillsyn keeps `steward` as an internal axis in its own `auth_requests` table + `AuthSession.PrincipalType` + new `AuthenticatedCaller.AuthPrincipalType`. At the autentauth adapter boundary (`internal/adapters/auth/autentauth/service.go:191`), map `steward → autentdomain.PrincipalTypeAgent`; on the way back, `principalTypeToActorType` (`:803-812`) keeps mapping to `ActorTypeAgent`. Lands in **3.19** with a doc-comment at the adapter callsite.

- **L3 — STEWARD persistent parents seeded via the default template's `[child_rules]` (Unit C C5).** Source: Unit C falsification CE C5 (auto-generator's `parent_id_lookup` fails before STEWARD parents exist — cold-start lockout). Resolution: 6 STEWARD persistent parents (`DISCUSSIONS`, `HYLLA_FINDINGS`, `LEDGER`, `WIKI_CHANGELOG`, `REFINEMENTS`, `HYLLA_REFINEMENTS`) become rule-spawned children of the project root via the default template, marked `Persistent = true` (per L9 below). Lands in **3.14** (`internal/templates/builtin/default.toml`).

- **L4 — Default template lives at `internal/templates/builtin/default.toml` (Unit B CE1).** Source: Unit B falsification CE1 (`//go:embed ../../templates/builtin/default.toml` from `internal/templates/embed.go` is build-stopping — Go embed rejects `..`). Resolution: TOML colocated under `internal/templates/builtin/`; `embed.go` lives in `internal/templates/`. The repo-root `templates/` directory is NOT recreated. Replaces Unit B 3.B.7's "repo-root sibling" claim. Lands in **3.14**.

- **L5 — `KindCatalog` import direction: lazy-decode `KindCatalogJSON json.RawMessage` on `Project` (Unit B CE4).** Source: Unit B falsification CE4 (circular import `internal/domain → internal/templates`). Resolution: `internal/domain/project.go` carries `KindCatalogJSON json.RawMessage`; decoding lives in `internal/app` or `internal/templates`, never on `Project`'s methods. Lands in **3.12**.

- **L6 — `GateRule` defers to Drop 4 (Unit B CE6).** Source: Unit B falsification CE6 (`GateRule` undefined in Round 1 schema). Resolution: Drop 3 leaves a `[gate_rules]` table hook in TOML schema for forward-compatibility but does NOT define `GateRule`'s Go struct or attachment point. Drop the `GateRule` struct stub from **3.8**. Drop 4 (dispatcher consumer) lands the actual struct + behavior.

- **L7 — Template-defined STEWARD-owned kinds covered NOW via `KindRule.Owner` field (Unit C C7).** Source: Unit C falsification CE C7 (PLAN.md § 19.3 bullet 9 has no concrete mechanism). Resolution: `KindRule.Owner string` field on Unit B's schema — accepting `"STEWARD"` as a value. The auto-generator reads this when materializing children: any kind with `owner = "STEWARD"` gets `Owner = "STEWARD"` set on creation regardless of who creates it. Schema field lands in **3.8** + **3.14**; auto-gen consumer lands in **3.20**.

- **L8 — `ReparentActionItem` is gated identically to `MoveActionItem` (Unit C C2).** Source: Unit C falsification CE C2 (`ReparentActionItem` reordering STEWARD items unguarded). Resolution: same `assertOwnerStateGate` semantic on the reparent path (`internal/app/service.go:1106-1156` + `internal/adapters/server/common/app_service_adapter_mcp.go:810-823`). Lands in **3.19** acceptance + integration test in **3.22**. Methodology §6.3 confirms reparenting is a state-affecting mutation.

### Methodology Integration (`ta-docs/cascade-methodology.md` §11 Canonical)

The cascade-methodology spec is now THE architectural canonical for cascade nodes. Drop 3 conforms to feature-parity for Tillsyn-as-substrate. Three new first-class fields and one new plan-QA-falsification attack vector land in this drop.

- **L9 — `Persistent bool` first-class field on `ActionItem` (methodology §11.2).** Replaces "STEWARD's persistent parents are special-cased in code" with "any node with `Persistent = true` is retained as an anchor; child-rules + template seed it once at project creation." STEWARD's 6 persistent parents land with `Persistent = true`. Lands in **3.17** alongside `Owner` + `DropNumber` first-class fields.

- **L10 — `DevGated bool` first-class field on `ActionItem` (methodology §11.2).** Generalizes the STEWARD owner-gate to "any node where state transitions need human-reviewer approval." For Drop 3, used by the refinements-gate confluence. Validation enforcement specifics live in Drop 4 dispatcher; Drop 3 lands the field + a passive auth-layer check that flags state transitions on `DevGated` items via `attention_item` (non-blocking — dispatcher isn't here yet). Lands in **3.17**.

- **L11 — `Irreducible bool` flag on `kind=build` droplets (methodology §2.3 + §11.3).** Marks single-function-signature changes, single SQL migrations, single template edits — droplets that cannot decompose further. Plan-QA-falsification validates the claim per the methodology rule ("Planners default to decompose; irreducibility is the exception, not an escape hatch"). Lands as a domain field on `ActionItem` in **3.2** (extends Unit A's `StructuralType`-on-`ActionItem` droplet — same struct edit).

- **L12 — 6th plan-QA-falsification attack vector: §4.4 global L1 plan-QA sweep.** Methodology §4.4 specifies a *second* plan-QA pass at L1 with full tree visibility when depth ≥ 3, checking: (i) blocker graph acyclic; (ii) no two sibling droplets share `paths` or `packages` without explicit `blockers`; (iii) acceptance criteria at leaves compose into L1 outcome; (iv) no orphan droplets. This is in addition to the 5 attack vectors PLAN.md § 19.3 line 1644-1649 enumerates. Lands in **3.7** (extends `~/.claude/agents/go-qa-falsification-agent.md` with the new block).

- **L13 — Reframe Unit C scope language as domain-primitive, not STEWARD-specific.** "STEWARD persistent parents" → "nodes with `Persistent = true` (the 6 STEWARD-owned anchor nodes seeded by the default template)." Same with `DevGated` — domain primitive, STEWARD is one consumer. Reflected throughout 3.17 / 3.20 / 3.21 / 3.22.

**Deferred to Drop 4 / refinements (NOT in Drop 3):** `failure` concrete type with `failure_kind` / `diagnostic` / `fix_directive`; `attempt_count` / `blocked_retry_count` / `last_failure_context` retry tracking; `start_commit` / `end_commit` git anchors; `context_blocks` array; per-kind droplet ceilings (`droplet_max_loc`, `droplet_max_files`); project onboarding fields (`mission`, `vocabulary`, `language`, `build_tool`, `standards_markdown_id`); `GateRule` Go struct + behavior (per L6); dispatcher consuming agent bindings; gate execution.

### Renumbering Map (Audit Trail)

| New     | Origin   | Title (short)                                                                  |
| ------- | -------- | ------------------------------------------------------------------------------ |
| 3.1     | 3.A.1    | Domain `StructuralType` enum + parser + tests                                  |
| 3.2     | 3.A.2    | `StructuralType` + `Irreducible` fields on `ActionItem`                        |
| 3.3     | 3.A.3    | SQLite `structural_type` column + scanner + INSERT/UPDATE                      |
| 3.4     | 3.A.4    | App-service + MCP plumbing for `structural_type`                               |
| 3.5     | 3.A.5    | Snapshot serialization round-trip for `StructuralType`                         |
| 3.6     | 3.A.6    | `## Cascade Vocabulary` glossary section in `WIKI.md`                          |
| 3.7     | 3.A.7    | Plan-QA-falsification agent prompt: 5 vocabulary attacks + 6th §4.4 sweep      |
| 3.8     | 3.B.1    | TOML schema structs (no `GateRule` per L6; carries `KindRule.Owner` per L7)    |
| 3.9     | 3.B.2    | TOML parser + load-time validator (schema-version pre-pass per CE5)            |
| 3.10    | 3.B.3    | `Template.AllowsNesting(parent, child Kind)` validation truth                  |
| 3.11    | 3.B.4    | `[child_rules]` consumer (`Template.ChildRulesFor`)                            |
| 3.12    | 3.B.5    | `KindCatalog` via lazy-decode `KindCatalogJSON json.RawMessage` (per L5)       |
| 3.13    | 3.B.6    | Agent binding fields fill-in (per N4 commit: 3.8 skeletal, 3.13 fills)         |
| 3.14    | 3.B.7    | `internal/templates/builtin/default.toml` (per L4) + STEWARD parent seeds (L3) |
| 3.15    | 3.B.8    | Rewrite/delete old API + delete `till.kind upsert` MCP/CLI (per CE8)           |
| 3.16    | 3.B.9    | Audit-trail `till.comment` + attention item on every nesting rejection         |
| 3.17    | 3.C.1    | `Owner` + `DropNumber` + `Persistent` + `DevGated` first-class fields          |
| 3.18    | 3.C.2    | SQLite `owner` + `drop_number` + `persistent` + `dev_gated` columns + index    |
| 3.19    | 3.C.3    | `principal_type: steward` + auth gates on Move + Update field-guard + Reparent + supersede + autent boundary-map |
| 3.20    | 3.C.4    | Template auto-gen consumer + `KindRule.Owner` consumer (per L7)                |
| 3.21    | 3.C.5    | MCP/snapshot plumbing for `Owner` + `DropNumber` + `Persistent` + `DevGated`   |
| 3.22    | 3.C.6    | Integration tests + refinements-gate forgetfulness regression                  |
| 3.23    | 3.D.1    | Agent file frontmatter cascade-glossary reminder (10 files)                    |
| 3.24    | 3.D.2    | `go-project-bootstrap` skill update (CLAUDE.md pointer line only per F4)       |
| 3.25    | 3.D.3    | `fe-project-bootstrap` skill update (CLAUDE.md pointer line only per F4)       |
| 3.26    | 3.D.4    | Cascade-glossary pointer in `main/CLAUDE.md` + `workflow/example/CLAUDE.md`    |
| 3.27    | 3.D.5    | In-repo legacy-vocabulary sweep (active canonical docs only)                   |
| 3.28    | 3.D.6    | Per-drop final wrap-up sweep (after Units A/B/C land)                          |

Total: 28 droplets.

### Droplet Decomposition

#### 3.1 — Domain `StructuralType` enum + parser + tests `[origin: 3.A.1]`

- **State:** todo
- **Paths:** `internal/domain/structural_type.go` (NEW), `internal/domain/structural_type_test.go` (NEW), `internal/domain/errors.go`
- **Packages:** `internal/domain`
- **Acceptance:**
  - `type StructuralType string` with four typed constants (`StructuralTypeDrop = "drop"`, `StructuralTypeSegment = "segment"`, `StructuralTypeConfluence = "confluence"`, `StructuralTypeDroplet = "droplet"`).
  - `validStructuralTypes` package-level slice in declaration order.
  - `IsValidStructuralType(StructuralType) bool` — `slices.Contains` with trim+lowercase normalize. Empty returns `false`.
  - `NormalizeStructuralType(StructuralType) StructuralType` — trim + lowercase.
  - `ParseStructuralTypeFromDescription(string) (StructuralType, error)` — regex `(?m)^StructuralType:\s*([a-z-]+)\s*$`. Rationale (per finding 5.A.10): builder may tighten to `[a-z]+` since none of the 4 enum values contain hyphens, OR keep `[a-z-]+` and document the choice. Either is acceptable; commit and document.
  - `errors.go` adds `ErrInvalidStructuralType`.
  - `structural_type_test.go` mirrors `role_test.go` shape: 3 `t.Parallel()` table-driven tests. Each enum value gets explicit cases; whitespace, case, empty-string carve-outs match `role_test.go` line-for-line. **Note:** new test's character-class comment text reflects the actual regex chosen — do NOT verbatim-copy `role_test.go:120`'s comment if it conflicts.
  - `mage test-pkg ./internal/domain` green; `structural_type.go` ≥ 90% covered.
- **Blocked by:** —
- **Notes:** New, not yet in tree. Mirror Drop 2.3 `Role` precedent.

#### 3.2 — `StructuralType` + `Irreducible` fields on `ActionItem` + validation `[origin: 3.A.2 + L11]`

- **State:** todo
- **Paths:** `internal/domain/action_item.go`, `internal/domain/domain_test.go`, plus `newActionItemForTest` helper (location TBD by builder — recommended `internal/domain/action_item_test_helpers.go` mirroring the Drop 1.75 `Kind` precedent at `internal/tui/model_test.go:14674-14687`).
- **Packages:** `internal/domain`
- **Acceptance:**
  - `ActionItem` struct gains two fields after `Role` (line 33): `StructuralType StructuralType`, `Irreducible bool`. (`Persistent` + `DevGated` land in 3.17 alongside `Owner` + `DropNumber` per L9 + L10 — single struct edit covers all four STEWARD-related domain primitives there; this droplet is Unit-A scope only.)
  - `ActionItemInput` struct gains the same two fields with doc comments. `StructuralType`'s comment: "MUST be a member of the closed StructuralType enum. Empty is rejected — `NewActionItem` returns `ErrInvalidStructuralType`." `Irreducible`'s comment: bool default `false`; semantics per `ta-docs/cascade-methodology.md` §2.3 + §11.3 (cited).
  - `NewActionItem` validation between existing `Role` block (lines 150-158) and lifecycle initialization:
    - `in.StructuralType = NormalizeStructuralType(in.StructuralType)`
    - Empty rejects with `ErrInvalidStructuralType` (diverges from Role's permissive empty).
    - Unknown value rejects with `ErrInvalidStructuralType`.
    - `Irreducible` is a bool with no validation (zero-value = `false` is the dominant case).
  - **Test-fixture helper** (per CE1 mitigation): introduce `newActionItemForTest(t *testing.T, in ActionItemInput) ActionItem` that defaults `StructuralType = StructuralTypeDroplet` if empty. Mirrors the Drop 1.75 `Kind` precedent. **All ~96 existing `domain.NewActionItem` call sites across 6 packages MUST be migrated to the helper OR explicitly supply a valid `StructuralType`.** Cross-package sweep enumerated:
    - `internal/domain/domain_test.go`, `internal/app/service.go` (production caller — per QA falsification §1.3), `internal/app/service_test.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/app/snapshot_test.go`, `internal/app/attention_capture_test.go`, `internal/app/dotted_address_test.go`, `internal/app/embedding_runtime_test.go`, `internal/adapters/storage/sqlite/embedding_jobs_test.go`, `cmd/till/embeddings_cli_test.go`, `internal/tui/model_test.go`.
  - Existing `TestNewActionItemRoleValidation` rows updated to supply valid `StructuralType` — otherwise prior tests break with new required-field validation.
  - New `TestNewActionItemStructuralTypeValidation` (each of 4 values accepted; empty/whitespace/unknown rejected with `ErrInvalidStructuralType`).
  - **Verification gate (per CE1 — upgrade to `mage ci`):** `mage ci` green, not just `mage test-pkg ./internal/domain`. The cross-package sweep means single-package verification under-covers.
- **Blocked by:** 3.1 (same Go package — package-level lock; new symbols must exist before this droplet compiles).
- **Notes:** Per L11, `Irreducible` lands here on `ActionItem` so plan-QA-falsification can attack the irreducible claim downstream. Builder may choose to migrate test helpers in a separate stub-PR pre-merge if the LOC delta is too large; defer the call to builder + per-droplet QA.

#### 3.3 — SQLite `structural_type` column + scanner + INSERT/UPDATE plumbing `[origin: 3.A.3 + finding 5.A.1 + 5.A.6]`

- **State:** todo
- **Paths:** `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`
- **Packages:** `internal/adapters/storage/sqlite`
- **Acceptance:**
  - `action_items` `CREATE TABLE` (line 168-197) gains TWO columns immediately after `role TEXT NOT NULL DEFAULT '',` (line 174): `structural_type TEXT NOT NULL DEFAULT ''` AND `irreducible INTEGER NOT NULL DEFAULT 0`. DDL-defaults match `role` precedent — code-level validation in 3.2 is the canonical gate. (`Persistent` + `DevGated` + `Owner` + `DropNumber` columns land in 3.18 alongside Unit-C struct fields.)
  - **Per finding 5.A.1 (corrected from Round 1 fictional cites):** there is exactly ONE column-bearing INSERT (`:1253`) and ONE column-bearing UPDATE (`:1347`). The line cites at `:1414`, `:1452`, `:2500` are SELECT projections, NOT additional INSERT/UPDATE paths. Sweep SELECT column-lists at `ListActionItems` (`:1414`), `ListActionItemsByParent` (`:1452`), and `getActionItemByID` (`:2500`) — each must include both new columns in the same ordinal position so `scanActionItem` reads the values correctly.
  - `INSERT` (`:1253`): column list adds `structural_type` + `irreducible` after `role`; VALUES tuple adds two more `?`; bind-arg list adds `string(t.StructuralType)` then `boolToInt(t.Irreducible)` (or equivalent SQLite-bool helper) after `string(t.Role)`.
  - `UPDATE` (`:1347`): SET clause adds `structural_type = ?` + `irreducible = ?` after `role = ?`; bind-arg list adds the same two values.
  - `scanActionItem`: adds `structuralTypeRaw string` and `irreducibleRaw int` to local var block, adds `&structuralTypeRaw, &irreducibleRaw` to `Scan()`, assigns `t.StructuralType = domain.StructuralType(structuralTypeRaw)` and `t.Irreducible = irreducibleRaw != 0` after existing `t.Role = domain.Role(roleRaw)` at `:2846`.
  - **Per finding 5.A.6 (line cite correction):** any migration block referenced lives at `line 518` (start of `workItemAlterStatements`), NOT lines 515/548 from Round 1.
  - **Pre-MVP rule (`feedback_no_migration_logic_pre_mvp.md`):** no `ALTER TABLE`, no SQL backfill — dev fresh-DBs.
  - `repo_test.go` round-trip per `StructuralType` value via `CreateActionItem` → `GetActionItem`. Existing repo-test fixtures supply valid `StructuralType` (via 3.2's `newActionItemForTest` helper).
  - `mage test-pkg ./internal/adapters/storage/sqlite` green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` (schema change).
- **Blocked by:** 3.2

#### 3.4 — App-service + MCP plumbing for `structural_type` `[origin: 3.A.4]`

- **State:** todo
- **Paths:** `internal/app/service.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`
- **Packages:** `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`
- **Acceptance:**
  - `app.CreateActionItemInput` (`:404`) gains `StructuralType domain.StructuralType` after `Role`. Doc: empty REJECTED.
  - `app.UpdateActionItemInput` (`:429`) gains `StructuralType`. Doc: empty preserves prior (mirrors Role's update semantics).
  - `app.CreateActionItem` (`:574`) threads `StructuralType: in.StructuralType` into `domain.NewActionItem`.
  - `app.UpdateActionItem` (`:~775-794`): sibling block to existing `Role` update at `:784-794` — if normalized non-empty, validate via `IsValidStructuralType` and assign.
  - `mcp_surface.go`: `CreateActionItemRequest` (`:57`) + `UpdateActionItemRequest` (`:78`) gain `StructuralType string`.
  - `app_service_adapter_mcp.go`: `CreateActionItem` (`:666-684`) + `UpdateActionItem` (`:708-720`) thread the field.
  - `extended_tools.go`: inline anonymous-struct `args` (`:860`) gains `StructuralType string \`json:"structural_type"\`` after `Role` (`:866`); `case "create"` block (`:~1033`) + `case "update"` block (`:~1092`) thread the field; tool schema declaration (`:1370`) adds `mcp.WithString("structural_type", mcp.Description(...), mcp.Enum("drop", "segment", "confluence", "droplet"))`. Legacy `till.create_task` (`:1416`) and `till.update_task` (`:1443`) get the same parameter.
  - `extended_tools_test.go`: `stubExpandedService.CreateActionItem` (`:429`) — **per finding 5.A.9 mitigation:** the stub defaults `args.StructuralType` to `StructuralTypeDroplet` when empty (matches the Role rejection-pattern that explicitly permits empty in the stub for backward fixture compat). Production code rejects empty; the stub is more permissive.
  - New `TestActionItemMCPRejectsEmptyOrInvalidStructuralType` exercises both empty (rejected on create) and unknown value (rejected on both create + update).
  - `mage test-pkg ./internal/app && mage test-pkg ./internal/adapters/server/common && mage test-pkg ./internal/adapters/server/mcpapi` all green.
- **Blocked by:** 3.2

#### 3.5 — Snapshot serialization round-trip for `StructuralType` `[origin: 3.A.5 + finding 5.A.2]`

- **State:** todo
- **Paths:** `internal/app/snapshot.go`, `internal/app/snapshot_test.go`
- **Packages:** `internal/app`
- **Acceptance:**
  - `SnapshotActionItem` struct (`:57`) gains `StructuralType domain.StructuralType \`json:"structural_type,omitempty"\`` after `Role` (`:63`). `omitempty` so pre-3.2 snapshots deserialize cleanly to empty.
  - `snapshotActionItemFromDomain` (`:1059`) adds `StructuralType: t.StructuralType` to the literal.
  - `(SnapshotActionItem).toDomain` (`:1267`) adds `StructuralType: t.StructuralType` — **NO** empty-string fallback (unlike `Kind` at `:1273-1278`); empty intentionally surfaces so 3.2's validation catches it on next mutation.
  - New `TestSnapshotActionItemStructuralTypeRoundTripPreservesAllValues` (mirrors `TestSnapshotActionItemRoleRoundTripPreservesAllRoles` at `:442`) — 4 cases, one per enum value.
  - `mage test-pkg ./internal/app` green.
- **Blocked by:** 3.4 (per finding 5.A.2 — explicit same-package serialization with 3.4; both touch `internal/app` package compile unit).

#### 3.6 — `## Cascade Vocabulary` glossary section in `WIKI.md` `[origin: 3.A.6 + finding 5.A.5]`

- **State:** todo
- **Paths:** `main/WIKI.md`
- **Packages:** — (non-Go MD edit)
- **Acceptance:**
  - **Per finding 5.A.5 (placement disambiguation):** insert as new `## Cascade Vocabulary` h2 between line 34 (end of `### Do Not Use Templates Right Now`) and line 36 (start of `## Level Addressing (0-Indexed)`) — sibling top-level h2, NOT nested.
  - Section structure:
    - One-paragraph waterfall metaphor orienting `drop / segment / confluence / droplet`.
    - `structural_type` enum reference: closed (4 values), mandatory on every non-project node, validated at `till.action_item(operation=create|update)` boundary, default NOT inferred.
    - Per-value definition + atomicity rules:
      - `drop` — vertical cascade step; level-1 children of project always drops; deeper drops are sub-cascades.
      - `segment` — parallel execution stream within a drop; segments within a drop run in parallel; may recurse.
      - `confluence` — merge/integration node; **MUST have non-empty `blocked_by`** (definitional).
      - `droplet` — atomic, indivisible leaf; **MUST have zero children**.
    - Orthogonality with `metadata.role` worked example: `(structural_type=droplet, role=builder)` canonical build leaf; `(structural_type=droplet, role=qa-proof)` canonical QA leaf; `(structural_type=confluence, role=…)` integration; etc.
    - 3-4 worked examples (drop fanning into 3 segments converging at confluence; droplet under segment with role=builder; confluence with non-empty `blocked_by`; sub-drop inside a segment).
    - Single-canonical-source rule explicit: every other doc holds a pointer, not a duplicate.
  - h2 for section title; h3 for sub-sections (matches `WIKI.md` hierarchy at lines 19, 36, 47).
  - No edits to other docs — those are 3.23/3.24/3.25/3.26 (Unit D) scope.
- **Blocked by:** —

#### 3.7 — Plan-QA-falsification agent prompt: 5 vocabulary attacks + 6th §4.4 sweep `[origin: 3.A.7 + L12]`

- **State:** todo
- **Paths:** `~/.claude/agents/go-qa-falsification-agent.md` (NOT git-tracked — diff recorded in worklog)
- **Packages:** —
- **Acceptance:**
  - `## Go Falsification Attacks` section (line 95-108) gains a new bullet block titled **Cascade-vocabulary attacks (post-Drop-3)** with five sub-bullets, each a CONFIRMED-counterexample template:
    1. **Droplet-with-children** — any item with `structural_type=droplet` that has children is misclassified.
    2. **Segment path/package overlap without `blocked_by`** — sibling segments sharing `paths[]` or `packages[]` without explicit `blocked_by` are a race.
    3. **Empty-`blocked_by` confluence** — definitional contradiction.
    4. **Confluence with partial upstream coverage** — every upstream named in description matches a `blocked_by` entry.
    5. **Role / structural_type contradictions** — qa-proof/qa-falsification on non-droplet; builder on confluence; planner on droplet without integration target; commit on non-droplet.
  - **Per L12 (6th attack vector):** add a sixth block titled **§4.4 global L1 plan-QA sweep**: when invoked at L1 with depth ≥ 3 view, run (i) blocker-graph acyclicity check; (ii) sibling-overlap-without-blockers check across `paths` and `packages`; (iii) leaf-acceptance-criteria-compose-into-L1-outcome check; (iv) orphan-droplet check. Cite `ta-docs/cascade-methodology.md` §4.4 inline.
  - One-line pointer at top of new block: *"Cascade vocabulary canonical: `WIKI.md` §`Cascade Vocabulary`. Methodology canonical: `ta-docs/cascade-methodology.md` §11."*
  - Existing `**Plan-level attacks (on planning tasks).**` bullet (line 108) gains a sibling line referencing the new cascade-vocabulary block above.
  - No edits to `~/.claude/agents/go-qa-proof-agent.md`.
  - Verify by re-reading the section + sanity-check that all 6 attack-vector keywords appear.
- **Blocked by:** —
- **Notes:** Inter-droplet write conflict on this same file with **3.23** + **3.27** — chain enforced via blockers (3.23 ← 3.7; 3.27 ← 3.23). Per finding 5.A.11: any "Confirmed" item from Round 1's "Unresolved Questions" section moves into the Architectural Decisions block of this PLAN (already done — see Locked block above).

#### 3.8 — TOML schema structs `[origin: 3.B.1 + L6 + L7 + N4]`

- **State:** todo
- **Paths:** `internal/templates/schema.go` (new), `internal/templates/schema_test.go` (new)
- **Packages:** `internal/templates` (new package)
- **Acceptance:**
  - `Template` struct with `SchemaVersion string`, `Kinds map[domain.Kind]KindRule`, `ChildRules []ChildRule`, `AgentBindings map[domain.Kind]AgentBinding` fields with `github.com/pelletier/go-toml/v2` tag conventions.
  - `KindRule` struct: `Owner string` (per L7 — accepts `"STEWARD"`), `AllowedParentKinds []domain.Kind`, `AllowedChildKinds []domain.Kind`, `StructuralType domain.StructuralType` (typed, enabled by 3.1's enum landing).
  - `ChildRule` struct: `WhenParentKind domain.Kind`, `CreateChildKind domain.Kind`, `Title string`, `BlockedByParent bool`, `WhenParentStructuralType domain.StructuralType`.
  - `AgentBinding` struct: skeletal (top-level fields declared but field-validation deferred to 3.13). Per N4 commitment: 3.8 ships skeletal `AgentBinding`; 3.13 fills validation + round-trip test. Builder of 3.13 doesn't read 3.8's output to decide.
  - **Per L6 (GateRule defer):** the schema reserves a `[gate_rules]` TOML table with no Go struct yet. `GateRule` struct stub is REMOVED from this droplet. Drop 4 lands the actual struct + behavior.
  - `SchemaVersionV1 = "v1"` constant exported.
  - All TOML tags `toml:"snake_case_name"`.
  - Every top-level type and field has Go doc comment.
  - `mage test-pkg ./internal/templates` passes (compile + skeletal type assertions).
- **Blocked by:** 3.1 (cross-unit — `domain.StructuralType` must exist).
- **Mage target:** `mage test-pkg ./internal/templates`.

#### 3.9 — TOML parser + load-time validator `[origin: 3.B.2 + finding 5.B.10]`

- **State:** todo
- **Paths:** `internal/templates/load.go` (new), `internal/templates/load_test.go` (new)
- **Packages:** `internal/templates`
- **Acceptance:**
  - `Load(io.Reader) (Template, error)` parses via `github.com/pelletier/go-toml/v2` decoder configured with `DisallowUnknownFields()` (strict).
  - **Per finding 5.B.10 (CE5 schema-version pre-pass):** tolerant pre-pass decodes ONLY `schema_version` (separate `Decoder` instance without `DisallowUnknownFields`). If `schema_version` unknown → reject with `ErrUnsupportedSchemaVersion` carrying clear UX message. Only THEN strict-decode the rest. Order: pre-pass → version-check → strict-decode.
  - Sentinel errors at package scope: `ErrUnknownTemplateKey`, `ErrUnsupportedSchemaVersion`, `ErrTemplateCycle`, `ErrUnreachableChildRule`, `ErrUnknownKindReference`.
  - Load-time validator: builds parent → child kind graph from `[child_rules]`; runs DFS (visited-set is fine, builder picks per finding 5.B.4) to detect cycles; asserts every referenced `Kind` is in `domain.validKinds` (closed 12-value enum); asserts no `[child_rules]` entry references an unreachable `WhenParentKind`.
  - Test coverage ≥ 80% on `load.go`. Table-driven tests follow project convention (no `tc := tc` per Go 1.22+).
  - Test cases: valid TOML; unknown key rejection; cycle; unreachable rule; unknown kind; missing schema_version; wrong schema_version (verify the pre-pass error UX path); malformed TOML.
  - `mage test-pkg ./internal/templates` green.
- **Blocked by:** 3.8.

#### 3.10 — `Template.AllowsNesting(parent, child Kind)` validation truth `[origin: 3.B.3 + finding 5.B.12]`

- **State:** todo
- **Paths:** `internal/templates/nesting.go` (new), `internal/templates/nesting_test.go` (new)
- **Packages:** `internal/templates`
- **Acceptance:**
  - `func (t Template) AllowsNesting(parent, child domain.Kind) (allowed bool, reason string)`.
  - Stable reason format: `"kind %q cannot nest under parent kind %q (rule: %s)"`. Tests assert exact text.
  - **Per finding 5.B.12 (CE7 drift mitigation):** test fixtures use a HAND-CODED `Template` value — NOT loaded from `default.toml`. 3.14's `embed_test.go` independently asserts the loaded `default.toml` round-trips against the SAME hand-coded fixture. Two distinct assertion paths against one source of truth.
  - 144-row table-driven test (12×12 Kind cartesian product) covering every combo.
  - Empty-template fallback (universal-allow) — captured in dedicated test row.
  - Coverage ≥ 90%.
  - `mage test-pkg ./internal/templates` green.
- **Blocked by:** 3.8, 3.9.

#### 3.11 — `[child_rules]` consumer (`Template.ChildRulesFor`) `[origin: 3.B.4 + L11 implications]`

- **State:** todo
- **Paths:** `internal/templates/child_rules.go` (new), `internal/templates/child_rules_test.go` (new)
- **Packages:** `internal/templates`
- **Acceptance:**
  - `func (t Template) ChildRulesFor(parent domain.Kind, parentType domain.StructuralType) []ChildRuleResolution`.
  - `ChildRuleResolution` struct: `Kind domain.Kind`, `Title string`, `BlockedByParent bool`, `StructuralType domain.StructuralType`, `Persistent bool`, `DevGated bool`, `Owner string` (per L7) — carries every field the auto-gen consumer (3.20) needs to materialize children.
  - Result deterministic — sorted stable order.
  - One level only — recursive expansion is dispatcher's job (Drop 4).
  - Test cases per PLAN.md § 19.3 line 1635: build → 2 children (build-qa-proof + build-qa-falsification); plan → 2 children (plan-qa-proof + plan-qa-falsification); structural_type=drop → 3 children (planner droplet + qa-proof droplet + qa-falsification droplet).
  - Coverage ≥ 85%.
- **Blocked by:** 3.8, 3.9, 3.10 (per QA falsification §1.2 — 3.10 + 3.11 share `internal/templates` Go-package compile unit; same-package serialization required). Cross-unit blocked by 3.1.

#### 3.12 — `KindCatalog` value type via lazy-decode `KindCatalogJSON json.RawMessage` `[origin: 3.B.5 + L5 + finding 5.B.14]`

- **State:** todo
- **Paths:** `internal/templates/catalog.go` (new), `internal/templates/catalog_test.go` (new), `internal/app/service.go`, `internal/app/kind_capability.go`, `internal/domain/project.go`
- **Packages:** `internal/templates`, `internal/app`, `internal/domain`
- **Acceptance:**
  - **Per L5 (resolves CE4 import-cycle):** `internal/domain/project.go` carries `KindCatalogJSON json.RawMessage` field — NOT typed `KindCatalog`. Decoder lives in `internal/app` or `internal/templates`, never on `Project`'s methods.
  - `KindCatalog` value type defined in `internal/templates/`. `Bake(t Template) KindCatalog` is pure — no I/O, no clock. `Lookup(kindID domain.KindID) (KindRule, bool)`.
  - JSON marshal/unmarshal for `Project.KindCatalogJSON` persistence.
  - `internal/app/service.go`: `CreateProjectWithMetadata` calls `templates.Bake` from per-project `<project_root>/.tillsyn/template.toml` if present, else from embedded `internal/templates/builtin/default.toml` (3.14).
  - `internal/app/kind_capability.go`: `resolveActionItemKindDefinition` (`:545-578`) replaces `s.repo.GetKindDefinition` calls with decoded-`KindCatalog.Lookup`. Repository fallback retained for projects with zero-value catalog (preserves boot compatibility per Drop 2.8 universal-nesting default).
  - **Per finding 5.B.14 (runtime-mutability):** explicit acceptance — edits to `<project_root>/.tillsyn/template.toml` AFTER project create are ignored until dev fresh-DBs. Document inline.
  - `internal/app/kind_capability_test.go` (31k LOC) still passes — adapter behavior preserved when `KindCatalog` is zero-valued.
  - `mage test-pkg ./internal/templates && mage test-pkg ./internal/app && mage test-pkg ./internal/adapters/storage/sqlite` all green.
  - `mage ci` green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` (project metadata shape change).
- **Blocked by:** 3.8, 3.9, 3.10, 3.11.
- **Mage target:** `mage ci`.

#### 3.13 — Agent binding fields fill-in `[origin: 3.B.6 + finding 5.B.17]`

- **State:** todo
- **Paths:** `internal/templates/schema.go`, `internal/templates/schema_test.go`, `internal/templates/agent_binding_test.go` (new)
- **Packages:** `internal/templates`
- **Acceptance:**
  - **Per N4 commit:** 3.8 shipped skeletal `AgentBinding` (top-level fields declared, no validation). 3.13 fills validation + round-trip test. Builder doesn't read 3.8's output to decide — the split is committed.
  - `AgentBinding` carries every field from PLAN.md § 19.3 lines 1653-1656: `AgentName string`, `Model string`, `Effort string`, `Tools []string` (per finding 5.B.5: `Tools` validation against actual MCP/Claude tool catalog deferred to Drop 4), `MaxTries int`, `MaxBudgetUSD float64`, `MaxTurns int`, `AutoPush bool`, `CommitAgent string`, `BlockedRetries int`, `BlockedRetryCooldown time.Duration`.
  - `BlockedRetryCooldown` parses from TOML duration strings (`"30s"`, `"5m"`).
  - Round-trip test asserts marshal→unmarshal stability for fully-populated `AgentBinding`.
  - Coverage ≥ 80%.
  - `mage test-pkg ./internal/templates` green.
- **Blocked by:** 3.8 (file-level: same `schema.go`).

#### 3.14 — `internal/templates/builtin/default.toml` + STEWARD parent seeds `[origin: 3.B.7 + L3 + L4 + finding 5.B.16]`

- **State:** todo
- **Paths:** `internal/templates/builtin/default.toml` (new — **per L4** under `internal/`, NOT repo-root), `internal/templates/embed.go` (new), `internal/templates/embed_test.go` (new)
- **Packages:** `internal/templates`
- **Acceptance:**
  - **Per L4 path correction:** TOML lives at `internal/templates/builtin/default.toml`. `embed.go` colocated in `internal/templates/` — `//go:embed builtin/default.toml` (no `..`). Repo-root `templates/` directory NOT recreated.
  - `default.toml` parses cleanly via `Load` (3.9).
  - Every closed-12-kind has a `[kinds.<kind>]` section.
  - **Per finding 5.B.16 (N3 explicit-deny):** reverse-hierarchy prohibitions are EXPLICIT `[child_rules]` deny rows — NOT implicit-by-absence. Adding a 13th kind in a future drop is then an explicit opt-in, not an implicit allow. Four PLAN.md prohibitions (closeout-no-closeout-parent; commit-no-plan-child; human-verify-no-build-child; build-qa-*-no-plan-child) each declared as a `[child_rules]` deny entry.
  - Auto-create rules per PLAN.md § 19.3 line 1635: build → build-qa-proof + build-qa-falsification; plan → plan-qa-proof + plan-qa-falsification; structural_type=drop → planner droplet + qa-proof droplet + qa-falsification droplet.
  - **Per L3 (STEWARD parent seeds):** the default template's `[child_rules]` seeds the 6 STEWARD persistent parents (`DISCUSSIONS`, `HYLLA_FINDINGS`, `LEDGER`, `WIKI_CHANGELOG`, `REFINEMENTS`, `HYLLA_REFINEMENTS`) at project creation time. Each marked with `persistent = true` (per L9), `owner = "STEWARD"` (per L7 — the auto-gen consumer in 3.20 reads this).
  - Agent bindings populated for every kind from CLAUDE.md "Agent Bindings" table.
  - **Per finding 5.B.8 (CE3):** `embed_test.go` independently asserts loaded `default.toml` covers all 12 kinds AND asserts the round-trip against the hand-coded `Template` value used in 3.10's `nesting_test.go`. Two test paths, one source of truth (per finding 5.B.12).
  - **Per finding 5.B.8 (CE3 also):** equivalent assertions from the soon-to-be-deleted `repo_test.go:2470-2517 TestRepositoryFreshOpen…` and `:2520-2568 TestRepositoryFreshOpenKindCatalogUniversalParentAllow` move HERE to `embed_test.go`. The `repo_test.go` tests are deleted in 3.15.
  - `mage test-pkg ./internal/templates` green.
- **Blocked by:** 3.8, 3.9, 3.10, 3.13. Cross-unit blocked by 3.1.

#### 3.15 — Rewrite/delete old API + delete `till.kind upsert` MCP/CLI `[origin: 3.B.8 + finding 5.B.7 + 5.B.8 + 5.B.13]`

- **State:** todo
- **Paths:**
  - `internal/domain/kind.go` — delete `KindTemplateChildSpec` (`:94-102`), `KindTemplate` (`:104-110`), `AllowedParentScopes` field (`:118`), `AllowsParentScope` method (`:200-211`), `normalizeKindTemplate` (`:296-352`), `normalizeKindParentScopes` (`:274-293`).
  - `internal/domain/kind_capability_test.go` — full file sweep (per finding 5.B.7 lower-bound starting points: `:18, 20, 49`; the entire `:18-73` test body is in scope, not just the cited lines).
  - `internal/domain/domain_test.go` — sweep test fixtures using deleted types.
  - `internal/app/kind_capability.go` — rewrite `:566` (`kind.AllowsParentScope(parent.Scope)` → `template.AllowsNesting(parent.Kind, kind.Kind)`); rewrite `validateKindTemplateExpansion` (`:771`); rewrite `mergeActionItemMetadataWithKindTemplate` (`:750-766`); per finding 5.B.7 also `:751-799` recursive expansion.
  - `internal/app/kind_capability_test.go` — extensive rewrites.
  - `internal/app/snapshot.go` — `:94`; per finding 5.B.7 also `:727, 1092, 1098, 1100, 1339, 1340, 1345, 1347`.
  - `internal/adapters/server/common/mcp_surface.go` — `:248`.
  - `internal/adapters/server/common/app_service_adapter_mcp.go` — per finding 5.B.7 `:1188, 1200, 1202, 1214`.
  - `internal/adapters/server/mcpapi/extended_tools.go` — `:1682, 1778`.
  - `internal/adapters/server/mcpapi/instructions_explainer.go` — per finding 5.B.7 `:241-242, 391, 403, 528, 550`.
  - `internal/adapters/server/mcpapi/extended_tools_test.go` — per finding 5.B.7 `:703, 706, 712, 720, 723`.
  - `internal/adapters/storage/sqlite/repo.go` — delete `kind_catalog` boot-seed (`:286-377`); per finding 5.B.7 also `:1061, 1066, 1070, 1095, 1100, 1104, 1130, 1140, 1156, 2940, 2942, 2964, 2966, 2970, 2976, 2981, 2982, 2987, 2988`. Schema column `template_json` becomes vestigial (pre-MVP rule — dev fresh-DBs).
  - `internal/adapters/storage/sqlite/repo_test.go` — `:2563`. Per finding 5.B.8 (CE3): DELETE `TestRepositoryFreshOpen…` (`:2470-2517`) and `TestRepositoryFreshOpenKindCatalogUniversalParentAllow` (`:2520-2568`). Equivalent assertions live in 3.14's `embed_test.go`.
  - `cmd/till/main.go` — `:3041-3442` LSP-confirmed plus per finding 5.B.7 `:3617, 3619`.
  - `internal/tui/model.go` — per finding 5.B.7 `:35, 754, 932, 1341`. `internal/tui/model_test.go:39, 87, 100, 104, 14661`.
  - **Per finding 5.B.13 (CE8 — wire surface deletion):** DELETE the `till.kind operation=upsert` MCP wire surface; DELETE the `till.upsert_kind_definition` legacy alias; DELETE the `till kind` mutating CLI subcommands. Read-only `till kind list/get` may stay if Drop 4 needs them; otherwise also delete. Document deprecation in this droplet's worklog. Pre-MVP rule: no migration.
  - **Disclaimer in builder spawn prompt:** the `Paths` list above is a **starting-point lower bound, not a closed enumeration**. Builder MUST run `LSP findReferences` exhaustively on every deleted symbol BEFORE editing.
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/adapters/storage/sqlite`, `cmd/till`, `internal/tui`
- **Acceptance:**
  - LSP `findReferences` on each deleted symbol returns 0 hits.
  - `mage test-pkg` for every touched package green; `mage ci` green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`.
- **Blocked by:** 3.3 (per QA proof §2.2 — both droplets edit `internal/adapters/storage/sqlite/repo.go`), 3.10, 3.11, 3.12.
- **Irreducible:** `true` (per QA falsification §1.7 + L11). Rationale: the rewrite-not-extend mandate (PLAN.md § 19.3 line 1623) forbids splitting `KindTemplate` deletion across multiple commits — every consumer must switch to the new API in one atomic commit, otherwise the codebase enters an intermediate state where some sites use the new API and others the old. LSP-driven sweep is the only correct approach. Methodology §2.3 ("a single function-signature change rippling through one file") generalizes here to a single closed-API deletion rippling through every consumer.

#### 3.16 — Audit-trail `till.comment` + attention item on every nesting rejection `[origin: 3.B.9 + finding 5.B.15]`

- **State:** todo
- **Paths:** `internal/app/kind_capability.go`, `internal/app/kind_capability_test.go`, `internal/domain/attention.go` (or `inbox.go` — verify via LSP `workspaceSymbol` for `AttentionCategory`), `internal/app/inbox_attention.go`
- **Packages:** `internal/app`, `internal/domain`
- **Acceptance:**
  - Every `Template.AllowsNesting → false` rejection at the create boundary creates a `till.comment` on the parent action item + an `attention_item` for the dev.
  - Comment uses existing `actor_type=user`/`actor_type=agent` actor model (no new principal type — Unit B's audit-trail is independent of Unit C STEWARD work).
  - **Per finding 5.B.15 (N2 scope-narrow):** rejection-comments fire ONLY on auth-gated creates (human/agent driven). Dispatcher-internal auto-create rejections route differently (Drop 4 dispatcher specs: `failed` state on parent, no comment). 3.16's path covers only the auth-gated create boundary.
  - Comment body markdown includes the rejection reason from `AllowsNesting`'s second return value verbatim.
  - Attention-item category `template_rejection`; subject names parent + child kind.
  - Test coverage: 4 cases end-to-end (closeout-no-closeout-parent; commit-no-plan-child; human-verify-no-build-child; build-qa-*-no-plan-child).
  - `mage test-pkg ./internal/app && mage test-pkg ./internal/domain` green; `mage ci` green.
- **Blocked by:** 3.10, 3.12, 3.14, 3.15.

#### 3.17 — `Owner` + `DropNumber` + `Persistent` + `DevGated` first-class fields `[origin: 3.C.1 + L9 + L10 + L13 + finding 5.C.16]`

- **State:** todo
- **Paths:** `internal/domain/action_item.go`, `internal/domain/domain_test.go`, `internal/domain/errors.go`
- **Packages:** `internal/domain`
- **Acceptance:**
  - `ActionItem` struct gains four fields after `Role` (`:33`): `Owner string`, `DropNumber int`, `Persistent bool`, `DevGated bool` (per L9 + L10).
  - `ActionItemInput` struct gains the same four fields.
  - **Per L13 reframe:** all four are domain primitives. STEWARD is just one consumer of `Owner`; the refinements-gate is one consumer of `DevGated`; the 6 anchor nodes are one consumer of `Persistent`. Doc comments name the methodology section: `ta-docs/cascade-methodology.md` §11.2.
  - `NewActionItem` trims `in.Owner` (no closed-enum — any non-empty trimmed value is valid; `"STEWARD"` is the value the auth gate keys on, but other principal-name owners are permitted).
  - `NewActionItem` rejects `in.DropNumber < 0` with new `ErrInvalidDropNumber` sentinel.
  - Empty `Owner` + zero `DropNumber` + `Persistent=false` + `DevGated=false` is the dominant zero-value case.
  - Table-driven tests: empty-owner round-trip; `STEWARD` round-trip; whitespace-only owner → empty after normalize; `DropNumber=0` round-trip; `DropNumber=5` round-trip; negative DropNumber rejected; `Persistent=true` / `DevGated=true` round-trip.
  - **Per finding 5.C.16 (N4 rollback-cost note):** if dev pushes back to metadata-JSON for `DropNumber`, every consumer except the auth gate changes shape — non-trivial backout (3.17 + 3.18 + 3.20 + 3.21 + tests). Document this in the droplet description.
  - All existing `domain_test.go` tests remain green.
  - `mage test-pkg ./internal/domain` green.
  - **DB action:** NONE (struct-only — schema column lands in 3.18).
- **Blocked by:** 3.2 (per QA proof §2.2 — both droplets edit the `ActionItem` struct at the same insertion point after `Role` line 33; same-file compile lock).

#### 3.18 — SQLite columns + index `[origin: 3.C.2 + finding 5.C.5 + 5.C.17]`

- **State:** todo
- **Paths:** `internal/adapters/storage/sqlite/repo.go`, `internal/adapters/storage/sqlite/repo_test.go`
- **Packages:** `internal/adapters/storage/sqlite`
- **Acceptance:**
  - `action_items` `CREATE TABLE` (`:168-197`) gains four columns appended after `role` (`:174`): `owner TEXT NOT NULL DEFAULT ''`, `drop_number INTEGER NOT NULL DEFAULT 0`, `persistent INTEGER NOT NULL DEFAULT 0`, `dev_gated INTEGER NOT NULL DEFAULT 0` (SQLite booleans as INTEGER 0/1).
  - `scanActionItem` reads all four columns into the struct.
  - INSERT + UPDATE SQL include all four columns.
  - **Per finding 5.C.5 + 5.C.17 (N5 index design):** the auto-generator (3.20) runs two cross-row queries — "find every level_2 finding under STEWARD persistent parents for drop N" and "find every drop N item." Both must be fully index-covered. Builder picks ONE: (a) swap index column order to `(project_id, drop_number, owner)` OR (b) add a second index `(project_id, drop_number)`. Pick based on perf measurement (small expected row counts; either is fine).
  - One new test in `repo_test.go` writes `Owner="STEWARD"`, `DropNumber=5`, `Persistent=true`, `DevGated=true`; reads back; asserts equality.
  - Pre-MVP rule honored: no `ALTER TABLE`, no SQL backfill — dev fresh-DBs.
  - `mage test-pkg ./internal/adapters/storage/sqlite` green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` (schema change).
- **Blocked by:** 3.3 (per QA proof §2.2 — both droplets edit `action_items` `CREATE TABLE` (`:168-197`)), 3.15 (per QA proof §2.2 — both droplets edit `internal/adapters/storage/sqlite/repo.go`), 3.17.

#### 3.19 — `principal_type: steward` + auth gates atomic landing `[origin: 3.C.3 + L1 + L2 + L8 + findings 5.C.1 + 5.C.13 + 5.C.14 + 5.C.15]`

- **State:** todo
- **Paths:**
  - `internal/domain/auth_request.go` — extend `normalizeAuthRequestPrincipalType` (`:596-608`) with fifth case `"steward" → "steward", nil`. Update `NewAuthRequest` validation block (`:393-405`): `steward` requires `principal_role = "orchestrator"`; reject any other role with `ErrInvalidAuthRequestRole`.
  - `internal/app/auth_requests.go` — verify `AuthSession.PrincipalType` (`:60`) and `AuthSessionIssueInput.PrincipalType` (`:35`) propagate `"steward"` correctly through `IssueAuthSession`.
  - `internal/domain/authenticated_caller.go` — **per finding 5.C.15 (N3 naming):** add new field with explicit name to avoid confusion with existing `PrincipalType ActorType` actor-class field. Field name: **`AuthRequestPrincipalType string`** (renamed from Round 1's `AuthPrincipalType` per 5.C.15 — clearer that it's the auth-request principal-class axis, not the actor-class). Both fields exist orthogonally: `PrincipalType ActorType` is user/agent/system attribution; `AuthRequestPrincipalType` is `user|agent|service|steward` for the new gates.
  - `internal/adapters/auth/autentauth/service.go` — **per L2 boundary-map:** at `:191` map `steward → autentdomain.PrincipalTypeAgent`. `principalTypeToActorType` (`:803-812`) keeps mapping to `ActorTypeAgent` for the actor-class axis. Doc-comment at the callsite documenting the mapping rationale.
  - `internal/adapters/server/common/app_service_adapter_mcp.go`:
    - New helper `assertOwnerStateGate(ctx, item)` invoked from `MoveActionItem` (`:728`) and `MoveActionItemState` (`:744`) AFTER the existing fetch (`:760`) and BEFORE the move SQL fires.
    - **Per finding 5.C.1 (NIT — column-only `MoveActionItem` pre-fetch):** for `MoveActionItem` (column-only path) the helper adds a `GetActionItem` call BEFORE the gate fires. Do NOT skip the gate just because the column-move path doesn't pre-fetch today.
    - **Per L1 (CE C1 field-level write guard):** extend `assertOwnerStateGate` to fire on `UpdateActionItemRequest` paths too. When `existing.Owner == "STEWARD"` and caller's `AuthRequestPrincipalType != "steward"`, REJECT any update whose `Owner` or `DropNumber` differ from existing values. (Prevents agent clearing `Owner` then transitioning state — silent gate bypass.)
    - **Per L8 (CE C2 reparent gate):** apply the same `assertOwnerStateGate` to `ReparentActionItem` (`:810-823` + `internal/app/service.go:1106-1156`). When `existing.Owner == "STEWARD"` and caller's `AuthRequestPrincipalType != "steward"`, REJECT.
    - **Per finding 5.C.13 (CE C8 supersede path):** Drop 1's always-on `failed` state introduced `till action_item supersede`. Apply the same gate identically: agent-principal session calling supersede on `Owner = STEWARD` REJECTS with `ErrAuthorizationDenied`; steward-principal session SUCCEEDS.
    - **Per finding 5.C.14 (N1 state-neutral semantic lock):** the gate fires on `Owner == "STEWARD"` regardless of state delta — drop-orchs cannot reorder or column-move STEWARD items even if no state changes. Stricter than § 19.3 bullet 7's literal text but matches "STEWARD owns state" intent. Document the choice.
  - `internal/adapters/server/common/auth.go` — `MutationAuthorizer` interface contract: implementations populate the new `AuthRequestPrincipalType` field from the underlying session.
  - `internal/adapters/server/common/app_service_adapter_mcp_test.go` (or whichever file already houses MoveActionItem tests — Read first):
    1. `MoveActionItemState(Owner=STEWARD, AuthRequestPrincipalType=agent)` → `ErrAuthorizationDenied`. State unchanged.
    2. `MoveActionItemState(Owner=STEWARD, AuthRequestPrincipalType=steward)` → SUCCEEDS.
    3. `MoveActionItemState(Owner="", AuthRequestPrincipalType=agent)` → SUCCEEDS.
    4. `MoveActionItem(Owner=STEWARD, AuthRequestPrincipalType=agent)` (column-level) → REJECTED.
    5. `UpdateActionItem(Owner=STEWARD, AuthRequestPrincipalType=agent)` changing description/details/metadata only → SUCCEEDS. Same call CHANGING `Owner` field → REJECTED (per L1).
    6. `ReparentActionItem(Owner=STEWARD, AuthRequestPrincipalType=agent)` → REJECTED (per L8).
    7. `Supersede(Owner=STEWARD, AuthRequestPrincipalType=agent)` → REJECTED (per 5.C.13).
    8. `CreateActionItem(parent_id=<STEWARD persistent parent>, AuthRequestPrincipalType=agent)` → SUCCEEDS (drop-orchs create children under STEWARD parents; that's the auto-gen pattern).
- **Packages:** `internal/domain`, `internal/app`, `internal/adapters/server/common`, `internal/adapters/auth/autentauth`
- **Acceptance:** all 8 test cases pass; `normalizeAuthRequestPrincipalType` accepts `"steward"`; `NewAuthRequest` requires `orchestrator` role for `steward` principal; `mage test-pkg ./internal/domain && mage test-pkg ./internal/app && mage test-pkg ./internal/adapters/server/common && mage test-pkg ./internal/adapters/auth/autentauth` all green; `mage ci` green.
- **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci` (auth_request principal_type accepted-set widens; defensive).
- **Blocked by:** 3.4 (per QA proof §2.2 — both droplets edit `internal/adapters/server/common/app_service_adapter_mcp.go`; same-file compile lock), 3.18.
- **Irreducible:** `true` (per QA falsification §1.7 + L11). Rationale: splitting risks an intermediate compile state where the enum recognizes `steward` but no enforcement consults the value, leaving a silent gap during partial deployment. The 8 test cases above (Move + MoveState + Update field-guard + Reparent + supersede + autent boundary-map) MUST land atomically. Methodology §2.3 ("the exception, not an escape hatch") justified by the cross-cutting auth-class extension touching domain + app + adapter layers in one coupled change.
- **Notes:** Same-file compile lock with **3.21** (which also touches `app_service_adapter_mcp.go`) — chain enforced via 3.21 ← 3.19.

#### 3.20 — Template auto-gen consumer + `KindRule.Owner` consumer `[origin: 3.C.4 + L7 + finding 5.C.2]`

- **State:** todo
- **Paths:**
  - **TOML body in `internal/templates/builtin/default.toml`** (the file landed in 3.14): the auto-generation `[child_rules]` entry that fires on level_1 numbered drop creation. Spawns 5 level_2 findings (parented under STEWARD persistent parents via `parent_id_lookup = "owner=STEWARD,title=..."`) + the refinements-gate confluence (with `blocked_by` enumerating every other Drop N item + the 5 just-created findings).
  - **Per L7 consumer:** the rule engine reads `KindRule.Owner` when materializing children. Any kind with `owner = "STEWARD"` gets `Owner = "STEWARD"` set on creation regardless of who creates it.
  - **Consumer-side Go code:**
    - New repository methods: `Repository.FindActionItemByOwnerAndTitle(ctx, projectID, owner, title) (domain.ActionItem, error)` + `Repository.ListActionItemsByDropNumber(ctx, projectID, dropNumber) ([]domain.ActionItem, error)`. Both index-covered by 3.18's index.
    - Auto-generator wiring in the rule-engine entry path (Unit B's 3.11 `ChildRulesFor` is the spec resolver; this droplet is the consumer that fires `app.CreateActionItem` for each resolution).
  - `internal/app/auto_generate_steward_test.go` (new) — table-driven cases:
    - Numbered drop `N=3` creation → 5 level_2 findings created under correct STEWARD parents + refinements-gate created with correct `blocked_by`.
    - Numbered drop `N=3` creation when STEWARD persistent parent missing → rule-engine returns clear error (auto-gen fails fast).
    - Non-numbered drop creation (`drop_number=0`) → rule does NOT fire.
    - Refinements-gate `blocked_by` enumerates every action item with `drop_number=N` (excluding itself) plus the 5 just-created findings.
- **Packages:** `internal/app`, `internal/adapters/storage/sqlite`, `internal/templates`
- **Acceptance:**
  - 5 STEWARD level_2 findings auto-create on numbered-drop creation, each `Owner="STEWARD"`, `DropNumber=N`, `Persistent=false`, `DevGated=false`, parented correctly.
  - Refinements-gate auto-creates inside drop's tree as `kind=plan, structural_type=confluence, Owner="STEWARD", DropNumber=N, DevGated=true` (per L10 — refinements-gate requires dev sign-off).
  - **Per finding 5.C.2 (NIT placeholder resolved):** `<final-rule-engine-droplet>` from Round 1 = **3.11** (`Template.ChildRulesFor`). Substitute throughout this droplet's description.
  - `mage test-pkg ./internal/app` green; `mage ci` green.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`.
- **Blocked by:** 3.11 (per finding 5.C.2 — the `Template.ChildRulesFor` engine droplet); 3.14 (per QA falsification §1.1 HIGH — 3.20 edits `internal/templates/builtin/default.toml`, the file 3.14 creates; load-bearing for Drop 4 dispatcher correctness); 3.19.

#### 3.21 — MCP/snapshot plumbing for `Owner` + `DropNumber` + `Persistent` + `DevGated` `[origin: 3.C.5]`

- **State:** todo
- **Paths:** `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go`, `internal/app/snapshot.go`, `internal/app/snapshot_test.go`
- **Packages:** `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/app`
- **Acceptance:**
  - `CreateActionItemRequest` + `UpdateActionItemRequest` + response shape gain `Owner string`, `DropNumber int`, `Persistent bool`, `DevGated bool`.
  - `app_service_adapter_mcp.go`: thread all four through `CreateActionItem` (`:~620`) + `UpdateActionItem` (`:~661`).
  - `extended_tools.go`: `mcp.WithString("owner", ...)`, `mcp.WithNumber("drop_number", ...)`, `mcp.WithBoolean("persistent", ...)`, `mcp.WithBoolean("dev_gated", ...)` on `till.action_item` create + update. Thread parsed values.
  - `extended_tools_test.go`: round-trip test for all four fields.
  - `snapshot.go`: `SnapshotActionItem` (`:57-84`) gains `Owner string \`json:"owner,omitempty"\``, `DropNumber int \`json:"drop_number,omitempty"\``, `Persistent bool \`json:"persistent,omitempty"\``, `DevGated bool \`json:"dev_gated,omitempty"\`` after `Role` (`:63`). Thread through `snapshotActionItemFromDomain` (`:1060`) and `(t SnapshotActionItem) toDomain()`.
  - `snapshot_test.go`: round-trip with `Owner="STEWARD", DropNumber=5, Persistent=true, DevGated=true`.
  - `omitempty` covers legacy-format compatibility — no `SnapshotVersion` bump.
  - `mage test-pkg` for all three packages green.
- **Blocked by:** 3.4 (per QA proof §2.2 — same-file compile lock on `app_service_adapter_mcp.go` and `extended_tools.go`), 3.5 (per QA proof §2.2 — same-file compile lock on `internal/app/snapshot.go`), 3.18, 3.19 (same-file compile lock on `app_service_adapter_mcp.go` with 3.19).
- **Irreducible:** `true` (per QA falsification §1.7 + L11). Rationale: 4 new fields × 6 wire surfaces (request struct + response struct + adapter create + adapter update + MCP tool schema + snapshot serialization) yields 24 coupled edit sites that all share the field-name vocabulary. Splitting per-field would require the wire surface to land twice (once per field pair) and risk JSON-tag drift between request/response. Methodology §2.3 fits ("single function-signature change rippling through one file" generalized to "a single struct-shape extension rippling through every wire surface"). Per-field decomposition is YAGNI churn.

#### 3.22 — Integration tests + refinements-gate forgetfulness regression `[origin: 3.C.6 + finding 5.C.11]`

- **State:** todo
- **Paths:** `internal/adapters/server/mcpapi/handler_integration_test.go` (or new `..._steward_integration_test.go`), `internal/app/auth_requests_test.go`
- **Packages:** `internal/adapters/server/mcpapi`, `internal/app`
- **Acceptance:**
  - **Test 1 — Drop-orch create + edit + cannot move state:** project root + 5 STEWARD persistent parents seeded via 3.14 fixture. Issue agent-principal session. Create `DROP_3` as level_1 plan with `drop_number=3`. Assert: 5 level_2 findings auto-created under STEWARD parents; refinements-gate auto-created with correct `blocked_by`. Drop-orch `update(description=...)` on `<DROP_3_HYLLA_FINDINGS>` SUCCEEDS. Drop-orch `move_state(state=complete)` on same → REJECTED with `ErrAuthorizationDenied`.
  - **Test 2 — STEWARD principal can move state:** STEWARD `move_state(state=complete)` on `<DROP_3_HYLLA_FINDINGS>` SUCCEEDS.
  - **Test 3 — Refinements-gate close path:** STEWARD closes 5 level_2 findings + every other Drop 3 item + `DROP_3_REFINEMENTS_GATE_BEFORE_DROP_4` SUCCEEDS (every blocker resolved). STEWARD then closes `DROP_3` → SUCCEEDS (parent-blocks-on-incomplete-child satisfied per Drop 1's always-on rule).
  - **Test 4 — Reparent gate (per L8):** drop-orch `reparent` on STEWARD-owned item → REJECTED.
  - **Test 5 — Supersede gate (per finding 5.C.13):** drop-orch `supersede` on STEWARD-owned item → REJECTED.
  - **Test 6 — Field-level guard (per L1):** drop-orch `update(action_item_id=<STEWARD_item>, owner="")` → REJECTED.
  - **Test 7 — Refinements-gate forgetfulness regression (per finding 5.C.11 + QA falsification §1.5):** drop-orch creates a mid-drop refinement plan-item for `drop_number=3` AFTER the gate is created. Drop-orch forgets to manually update the gate's `blocked_by` list. Gate close fires anyway. Test asserts BOTH:
    - **(a) The invariant:** a STEWARD attempt to close `DROP_3` (level_1) while a `drop_number=3` child is still `in_progress` REJECTS with parent-blocks-on-incomplete-child (per Drop 1's always-on rule). NOT just the safety-net warning — the underlying invariant must reject the close. Tightened per QA falsification §1.5 (Round 1's Test 7 only verified the warning surface; the invariant must be pinned independently).
    - **(b) The safety-net:** an `attention_item` is created on gate-close warning the dev that `drop_number=3` items remained `in_progress` when the gate closed. Documents the failure mode rather than papering over it (per 5.C.11 ACCEPT-with-warning resolution).
  - **Auth-request unit tests in `auth_requests_test.go`:** creating auth request with `principal_type="steward" + principal_role="orchestrator"` SUCCEEDS; with `principal_type="steward" + principal_role="builder"` REJECTED with `ErrInvalidAuthRequestRole`.
  - `mage test-pkg ./internal/adapters/server/mcpapi && mage test-pkg ./internal/app` green; `mage ci` green at unit boundary.
  - **DB action:** dev DELETEs `~/.tillsyn/tillsyn.db` BEFORE `mage ci`.
- **Blocked by:** 3.20, 3.21.

#### 3.23 — Agent file frontmatter cascade-glossary reminder `[origin: 3.D.1 + finding 5.D.5 (F5)]`

- **State:** todo
- **Paths (all NOT git-tracked — diff recorded in worklog):**
  - `~/.claude/agents/go-builder-agent.md`, `~/.claude/agents/go-planning-agent.md`, `~/.claude/agents/go-qa-proof-agent.md`, `~/.claude/agents/go-qa-falsification-agent.md`, `~/.claude/agents/go-research-agent.md`
  - `~/.claude/agents/fe-builder-agent.md`, `~/.claude/agents/fe-planning-agent.md`, `~/.claude/agents/fe-qa-proof-agent.md`, `~/.claude/agents/fe-qa-falsification-agent.md`, `~/.claude/agents/fe-research-agent.md`
- **Packages:** —
- **Acceptance:**
  - Each of 10 agent files gains a one-line cascade-glossary reminder placed **per finding 5.D.5 lock-in: prose body immediately after the YAML frontmatter `---` close marker** (option (b)). NOT inside YAML — the YAML schema uses `name` / `description` / `tools` only; unknown keys silently dropped.
  - Exact text: *"Structural classifications (`drop` | `segment` | `confluence` | `droplet`) live in the project WIKI's `## Cascade Vocabulary` section — never redefine here."*
  - Reminder lives BEFORE any "You are the …" identity sentence so it reads as agent-wide context.
  - Reminder identical across all 10 files.
  - Each file diff: exactly +1 line plus optional +1 blank-line padding.
  - Builder records 10-file diff in worklog.
- **Blocked by:** **3.7** (per mechanical fix 4.6 — three-way write conflict on `go-qa-falsification-agent.md` between 3.7, 3.23, 3.27. Order: 3.7 → 3.23 → 3.27. Inter-unit also: needs 3.6 (WIKI section authored).
- **Notes:** Builder spawn prompt MUST explicitly authorize `~/.claude/agents/*.md` edits since paths are outside the repo working dir.

#### 3.24 — `go-project-bootstrap` skill update (CLAUDE.md pointer line only) `[origin: 3.D.2 + finding 5.D.9 + 5.D.5 (F5 inline boilerplate)]`

- **State:** todo
- **Paths (NOT git-tracked):** `~/.claude/skills/go-project-bootstrap/SKILL.md`, `~/.claude/skills/go-project-bootstrap/references/template.md`
- **Packages:** —
- **Acceptance:**
  - **Per finding 5.D.9 (scope narrowing):** scope reduced to "CLAUDE.md pointer line only." Defer WIKI scaffolding ownership to a follow-up refinement. Avoids scope expansion of bootstrap skills mid-Drop-3.
  - **Per finding 5.D.5 (F5 lock):** boilerplate `## Cascade Vocabulary` content owner is **option (iii) — embed inline in `references/template.md`** (bootstrap skill self-contained; no Unit B coordination needed).
  - **Per finding 5.D.5 (F5 lock — frontmatter convention):** insert pointer in `references/template.md`, NOT in YAML frontmatter (description field consumed by autoloader for relevance scoring; YAML pollution degrades skill discovery).
  - `SKILL.md` workflow: new bullet under "3. Add Go-specific guidance": *"Add a CLAUDE.md pointer line: 'Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary`.'"*
  - `references/template.md`: new top-level rule: *"Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary` — never redefine in CLAUDE.md."*
  - Builder records diff in worklog.
- **Blocked by:** — (intra-unit). Inter-unit: 3.6.
- **Notes:** Disjoint files from 3.25 — can run in parallel.

#### 3.25 — `fe-project-bootstrap` skill update (CLAUDE.md pointer line only) `[origin: 3.D.3 + finding 5.D.9]`

- **State:** todo
- **Paths (NOT git-tracked):** `~/.claude/skills/fe-project-bootstrap/SKILL.md`, `~/.claude/skills/fe-project-bootstrap/references/template.md`
- **Packages:** —
- **Acceptance:** Identical pattern to 3.24, applied to FE variant. Same scope narrowing per 5.D.9. Same inline-boilerplate per 5.D.5 (F5).
- **Blocked by:** — (intra-unit). Inter-unit: 3.6.
- **Notes:** Disjoint files from 3.24 — runs in parallel with 3.24.

#### 3.26 — Cascade-glossary pointer in `main/CLAUDE.md` + `workflow/example/CLAUDE.md` `[origin: 3.D.4 + finding 5.D.11]`

- **State:** todo
- **Paths (git-tracked):** `main/CLAUDE.md`, `main/workflow/example/CLAUDE.md`
- **Packages:** —
- **Acceptance:**
  - `main/CLAUDE.md`: in existing "Cascade Plan" section, insert as second sentence of the section: *"Cascade vocabulary canonical: `WIKI.md` § `Cascade Vocabulary` — never redefine here."* No new section.
  - **Per finding 5.D.11 (LOW — disambiguation lock):** `main/workflow/example/CLAUDE.md`: insert as new bullet inside `## Coordination Model — At a Glance` immediately after the line-26 reading-order bullet. Exact text: *"Cascade vocabulary canonical: project `WIKI.md` § `Cascade Vocabulary` — every adopter project's CLAUDE.md MUST include this pointer and MUST NOT redefine the structural_type vocabulary locally."*
  - Both edits pure additions — no existing content removed.
  - Both files lint clean (markdown linter — no broken links, consistent code-fence usage).
  - Both land in Drop 3 PR.
- **Blocked by:** — (intra-unit). Inter-unit: 3.6.

#### 3.27 — In-repo legacy-vocabulary sweep (active canonical docs only) `[origin: 3.D.5 + findings 5.D.7 + 5.D.8 + 5.D.10 + 5.D.14]`

- **State:** todo
- **Paths (in-scope active canonical docs):**
  - `main/CLAUDE.md`, `main/PLAN.md` (surgical — only where prose conflates kind/role/structural_type), `main/WIKI.md` (pointer-only edits — canonical glossary owned by 3.6), `main/STEWARD_ORCH_PROMPT.md`, `main/AGENT_CASCADE_DESIGN.md`, `main/AGENTS.md`, `main/README.md`, `main/CONTRIBUTING.md`, `main/SEMI-FORMAL-REASONING.md`, `main/HYLLA_WIKI.md`, `main/tillsyn-project.md`, `main/DROP_1_75_ORCH_PROMPT.md`
  - `main/workflow/example/CLAUDE.md`, `main/workflow/example/drops/WORKFLOW.md`, `main/workflow/example/drops/_TEMPLATE/**`, `main/workflow/example/drops/DROP_N_EXAMPLE/**`
  - `~/.claude/agents/*.md` (full pass after 3.23's frontmatter reminder)
  - `~/.claude/skills/*/SKILL.md` + `~/.claude/skills/*/references/*.md` (global skills)
  - `~/.claude/CLAUDE.md` — **per finding 5.D.8 (CE3 lock):** promoted from "review only" to first-class in-scope edit. Four known retired-vocab hits enumerated: line 9 `slice-by-slice` → drop or rephrase; lines 10, 121, 147 `build-task` → `build`. Treat as in-scope per memory rule "Tillsyn — No Slice Anywhere" applied globally.
  - **Per finding 5.D.14 (LOW):** `~/.claude/commands/*.md` slash-command files added to sweep path list. NOT git-tracked; worklog-recorded. If no files exist, document the absence.
  - `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/MEMORY.md` — review only, no edits unless rule actually retired.
- **Paths (out of scope — explicitly excluded per finding 5.D.7):**
  - `main/workflow/drop_0/**`, `main/workflow/drop_1/**`, **`main/workflow/drop_1_5/**`** (per 4.5), **`main/workflow/drop_1_75/**`** (per 4.5), `main/workflow/drop_2/**` — historical worklog audit trail.
  - `main/workflow/drop_3/**` (except this PLAN.md when needed).
  - `main/LEDGER.md`, `main/WIKI_CHANGELOG.md`, `main/HYLLA_FEEDBACK.md`, `main/REFINEMENTS.md`, `main/HYLLA_REFINEMENTS.md` — historical audit trail.
  - All Go source under `internal/` and `cmd/` — Unit D is doc-only.
- **Packages:** —
- **Acceptance:**
  - Sweep pass 1: `Grep "drops all the way down"` and `"tasks all the way down"` across in-scope paths returns zero hits.
  - Sweep pass 2: `Grep "action_item|action-item|action item|ActionItem"` reviewed line-by-line. Hits where prose conflates kind with role get rewritten; correct schema usage stays.
  - Sweep pass 3: `Grep "slice|build-task|plan-task|qa-check"` rewritten where they appear in active prose to use closed 12-kind enum (`build`, `plan`, `build-qa-proof`, etc.). Audit-trail mentions of historical kinds in change-log entries stay.
  - Sweep pass 4: `metadata.role` × `metadata.structural_type` × `kind` orthogonal-axes design documented coherently.
  - **Per finding 5.D.10 (M5 audit-gap accept):** `~/.claude/` edits are recorded in `BUILDER_WORKLOG.md` only. The workflow trades off git-tracked permanence for adopter-skill universality. Future maintainers reading the worklog see what edits landed; reconstruction against future filesystem state requires manual diff. (Option (c) of QA's three options.) Document this audit-gap acceptance in the droplet description explicitly.
  - **Per finding 5.D.5 (F4 — subagent vs orch MD work):** memory rule "No subagents for MD work" applies to in-orch quick MD updates; for systematic sweeps in this droplet, builder subagent IS the work-doer. Self-QA + dev-approval gate before commit still applies.
  - Builder records per-file diff inventory in `BUILDER_WORKLOG.md`: `<path> <hits> <rewrites> <left-as-is>`.
  - `mage ci` green (no Go code touched; CI exit status is smoke test).
  - Self-QA: builder presents inventory + sample to orch → dev approves → commit.
- **Blocked by:** **3.23** (per 4.6 — same-file race on `go-qa-falsification-agent.md`); **3.26** (shares `main/CLAUDE.md` + `main/workflow/example/CLAUDE.md`).
- **Irreducible:** `true` (per QA falsification §1.7 + L11). Rationale: a single coherent vocabulary sweep across ~15 in-repo paths + the global `~/.claude/CLAUDE.md` + 10 `~/.claude/agents/*.md` + 12 `~/.claude/skills/**/*.md` files. Splitting the sweep risks vocabulary inconsistency between half-swept docs (e.g., `main/CLAUDE.md` says new vocab, `main/STEWARD_ORCH_PROMPT.md` still uses retired). One sweep, one self-QA pass, one dev-approval gate. Methodology §2.3 fits — "single SQL migration" generalizes here to "single coordinated vocabulary sweep across active canonical docs."

#### 3.28 — Per-drop final wrap-up sweep `[origin: 3.D.6 + finding 5.D.5 (F5) + 5.D.12]`

- **State:** todo
- **Paths:** Same in-scope set as 3.27, re-swept for new vocabulary introduced by 3.1–3.22:
  - `structural_type` + 4 enum values + atomicity rules + 6 plan-QA-falsification attack vectors (3.1–3.7).
  - TOML template-system terms (`Template.AllowsNesting`, `[child_rules]`, `KindCatalog`, `internal/templates/builtin/default.toml`); agent-binding kind fields (3.8–3.16).
  - `principal_type: steward`, `Owner = "STEWARD"`, `Persistent`, `DevGated`, auth-level state-lock, template auto-generation (3.17–3.22).
- **Packages:** —
- **Acceptance:**
  - Same sweep methodology as 3.27. Search terms = new vocabulary from A/B/C.
  - Verify every doc that references new vocabulary uses it consistently and points back to the canonical source (3.6 for cascade vocabulary; Unit B's template-system docs; Unit C's auth-model docs).
  - Verify `metadata.role` × `metadata.structural_type` × `kind` orthogonal-axes design is documented coherently in at least `main/CLAUDE.md`, `main/WIKI.md`, `main/PLAN.md`, `main/workflow/example/CLAUDE.md` (the four primary always-read docs).
  - Verify no new doc inadvertently re-introduces retired *"drops all the way down"* framing.
  - **Per finding 5.D.5 (F5 — wrap-up timing lock):** runs **before** Drop 3 PR merges (option (a)). Sweep ships inside Drop 3 PR — preserves PR-review density. Per finding 5.D.12: option (b) bypasses PR review and accumulates documentation drift on `main`; reject.
  - Builder records per-file diff inventory in `BUILDER_WORKLOG.md`.
  - `mage ci` green.
  - Self-QA + dev-approval before commit.
  - Commit message (per memory rule "Single-Line Commits"): `docs(drop-3): cascade vocabulary final sweep` — one line, no body.
- **Blocked by:** **3.27** (shares paths). Inter-unit: Units A + B + C all closed (`mage ci` green at each unit boundary). Concretely: 3.7, 3.16, 3.22 all complete.

### Cross-Unit Blocker Wiring (Acyclic DAG)

Hard cross-unit edges (every edge listed below; verify acyclicity by topological sort 3.1 → 3.28):

| Blocker (must complete) | Blocked droplet | Reason |
| ---                     | ---             | ---    |
| 3.1                     | 3.8             | `domain.StructuralType` typed in `KindRule.StructuralType` (Unit B schema). |
| 3.1                     | 3.11            | `child_rules` consumer references `domain.StructuralType` axis. |
| 3.1                     | 3.14            | `default.toml` auto-create rule on drop structural_type. |
| 3.2                     | 3.17            | Same-file `ActionItem` struct compile lock (per QA proof §2.2 — fix 4.1). |
| 3.3                     | 3.15            | Same-file `internal/adapters/storage/sqlite/repo.go` compile lock (per QA proof §2.2). |
| 3.3                     | 3.18            | Same-file `action_items` `CREATE TABLE` compile lock (per QA proof §2.2). |
| 3.4                     | 3.5             | Same-package serialization (`internal/app`) per finding 5.A.2. |
| 3.4                     | 3.19            | Same-file `app_service_adapter_mcp.go` compile lock (per QA proof §2.2). |
| 3.4                     | 3.21            | Same-file `app_service_adapter_mcp.go` + `extended_tools.go` compile lock (per QA proof §2.2). |
| 3.5                     | 3.21            | Same-file `internal/app/snapshot.go` compile lock (per QA proof §2.2). |
| 3.10                    | 3.11            | Same Go-package compile unit (`internal/templates`) per QA falsification §1.2. |
| 3.11                    | 3.20            | Auto-gen consumer reads `Template.ChildRulesFor` (per finding 5.C.2). |
| 3.14                    | 3.20            | 3.20 edits `internal/templates/builtin/default.toml`, the file 3.14 creates (per QA falsification §1.1 HIGH). |
| 3.15                    | 3.18            | Same-file `internal/adapters/storage/sqlite/repo.go` compile lock (per QA proof §2.2). |
| 3.7                     | 3.23            | `go-qa-falsification-agent.md` write conflict (per 4.6). |
| 3.6                     | 3.23, 3.24, 3.25, 3.26 | Pointer targets reference real WIKI section. |
| 3.23                    | 3.27            | Same-file `go-qa-falsification-agent.md` (per 4.6). |
| 3.26                    | 3.27            | Shares `main/CLAUDE.md` + `main/workflow/example/CLAUDE.md`. |
| 3.27                    | 3.28            | Shares paths. |
| 3.7, 3.16, 3.22         | 3.28            | "Units A + B + C all closed" prerequisite. |

Topological sort (one valid order, demonstrating acyclicity):
3.1 → 3.2 → 3.3 → 3.4 → 3.5 → 3.6 → 3.7 → 3.8 → 3.9 → 3.10 → 3.11 → 3.12 → 3.13 → 3.14 → 3.15 → 3.16 → 3.17 → 3.18 → 3.19 → 3.20 → 3.21 → 3.22 → 3.23 (parallel: 3.24, 3.25) → 3.26 → 3.27 → 3.28.

Parallelism: 3.24 + 3.25 disjoint and run in parallel. Within Unit B, 3.10 + 3.11 (post-3.9) are package-disjoint at file level but both compile against `internal/templates` — sequenced for deterministic ordering. Within Unit C, 3.20 + 3.21 are package-disjoint at file level but 3.21 same-file-locks with 3.19; sequenced 3.19 → 3.21, with 3.20 following 3.19 and 3.11.

### Mechanical Fix Log

**Round 1 → Round 2 (REVISION_BRIEF §4, 8 fixes applied during initial synthesis):**

- 4.1 (3.A.3 fictional INSERT/UPDATE cites corrected) → 3.3.
- 4.2 (3.A.5 same-package serialization with 3.A.4) → 3.5 `blocked_by: 3.4`.
- 4.3 (3.C.5 blocked_by 3.C.3) → 3.21 `blocked_by: 3.19`.
- 4.4 (Unit D renumbering 5.D.* → 3.D.*) → applied throughout 3.23-3.28.
- 4.5 (drop_1_5/ + drop_1_75/ exclusions in 3.D.5/3.D.6) → 3.27 + 3.28 out-of-scope lists.
- 4.6 (three-way write conflict on `go-qa-falsification-agent.md`) → chain 3.7 → 3.23 → 3.27.
- 4.7 (Unit B 3.B.8 LSP-found additional sites) → 3.15 paths list with explicit "starting-point lower bound" disclaimer.
- 4.8 (Unit C 3.C.3 supersede path) → 3.19 acceptance Test 7.

**Round 2 plan-QA → Round 3 (PLAN_QA_PROOF_R2 + PLAN_QA_FALSIFICATION_R2, 11 fixes applied dev-approved 2026-05-02):**

- R2.B1 (Persistent/DevGated double-landing on 3.2 + 3.17) → dropped from 3.2 (kept `StructuralType` + `Irreducible` only); 3.17 remains canonical landing per L9 + L10. Renumbering map row 3.2 title updated.
- R2.B2 (9 missing same-file `blocked_by` edges) → 3.17 ← 3.2; 3.18 ← 3.3; 3.18 ← 3.15; 3.15 ← 3.3; 3.19 ← 3.4; 3.21 ← 3.4; 3.21 ← 3.5; 3.20 ← 3.14; 3.11 ← 3.10. Cross-Unit Blocker Wiring table updated with all 9 edges.
- R2.M1 (`Irreducible` SQLite persistence path) → 3.3 acceptance gains `irreducible INTEGER NOT NULL DEFAULT 0` column + INSERT/UPDATE/scan plumbing.
- R2.M2 (3.2 sweep adds `internal/app/service.go` production caller) → enumerated sweep list updated.
- R2.M3 (3.22 Test 7 invariant assertion) → Test 7 now asserts BOTH the parent-blocks-on-incomplete-child invariant AND the safety-net `attention_item` warning.
- R2.N1 (reversed table row 3.5 / 3.4) → reordered as `3.4 → 3.5` in Cross-Unit Blocker Wiring table.
- R2.N2 (off-by-one line cites: `extended_tools.go:1417→:1416`, `:1444→:1443`, `cmd/till/main.go:3042→:3041`) → corrected.
- R2.N3 (mark oversized droplets `Irreducible: true` with rationale per QA falsification §1.7 + L11) → applied to 3.15, 3.19, 3.21, 3.27.
- R2.N4 ("Three known retired-vocab hits" → "Four") → corrected at the `~/.claude/CLAUDE.md` enumeration in 3.27.
- R2.N5 (Hylla-MCP-first directive for builder + QA subagent spawns) → added to header paragraph.
- R2.N6 (methodology §2.2 package-level gate) → added explicit deferred-to-Drop-4 entry in Out Of Scope.

(Round 1 finding 4.2 from PROOF — "L8 §6.3 misquote" — verified absent in current PLAN.md text; no fix required.)

### Out Of Scope Confirmations

Explicit non-scope, deferred to future drops or refinements per REVISION_BRIEF §3 / pre-MVP rules:

- **Drop 4 (dispatcher):** `GateRule` Go struct + behavior (per L6); dispatcher consuming agent bindings; gate execution; rule-fire-on-every-new-drop-N-child for the refinements-gate (3.22 Test 7 documents the manual-update path; rule-fire is Drop 4); **methodology §2.2 package-level build+test gate** as a first-class cascade primitive (per QA falsification §1.4 — Drop 3 plan does not model per-package QA gate nodes; the dispatcher in Drop 4 is the natural consumer).
- **Methodology fields deferred:** `failure` concrete type (`failure_kind`/`diagnostic`/`fix_directive`); `attempt_count`/`blocked_retry_count`/`last_failure_context`; `start_commit`/`end_commit` git anchors; `context_blocks` array; per-kind droplet ceilings; project onboarding fields (`mission`/`vocabulary`/`language`/`build_tool`/`standards_markdown_id`).
- **TUI overhaul:** Drop 4.5.
- **Cascade dogfooding:** Drop 5.
- **Escalation:** Drop 6.
- **Error handling + observability:** Drop 7.
- **Migration logic:** none in Drop 3 — no Go code, no `till migrate` subcommand, no SQL backfill. Dev fresh-DBs `~/.tillsyn/tillsyn.db` between schema-touching droplets (3.3, 3.12, 3.15, 3.18, 3.19, 3.20, 3.22).
- **Closeout MD artifacts:** none — no `CLOSEOUT.md` / `LEDGER.md` / `WIKI_CHANGELOG.md` / `REFINEMENTS.md` / `HYLLA_FEEDBACK.md` / `HYLLA_REFINEMENTS.md` rollups (per `feedback_no_closeout_md_pre_dogfood.md`). Worklog MDs (`PLAN.md`, `BUILDER_WORKLOG.md`, `BUILDER_QA_*.md`, `PLAN_QA_*.md`) DO happen.
- **Bootstrap WIKI scaffolding ownership:** narrowed in 3.24 / 3.25 to "CLAUDE.md pointer line only"; WIKI scaffolding ownership deferred to a follow-up refinement (per finding 5.D.9).
- **`till.kind upsert` MCP / `till.upsert_kind_definition` legacy alias / `till kind` mutating CLI:** DELETED in 3.15 (per finding 5.B.13). Read-only `till kind list/get` may stay if Drop 4 needs them; otherwise also delete.
- **`Tools []string` validation against actual MCP/Claude tool catalog:** deferred to Drop 4 (per finding 5.B.5).

## Notes

### Decomposition Strategy (Orchestrator-Routed Question Before Phase 1)

Drop 3 has **10 discrete work items** spanning at least 4 distinct surfaces. Per `feedback_decomp_small_parallel_plans.md`: "Default decomp for any drop with >1 package or >2 surfaces is ≤N small parallel planners (≤15min each, one surface/package) + orch synthesis + narrow build-drops with explicit blocked_by."

**Candidate parallel planner decomposition:**

- **Planner A — Cascade vocabulary foundation.** `metadata.structural_type` enum + WIKI `## Cascade Vocabulary` glossary + 5 new plan-QA-falsification attack vectors. Self-contained, foundational (other planners depend on the enum existing).
- **Planner B — Template system overhaul.** Closed TOML schema + parser + validator + `Template.AllowsNesting` + `[child_rules]` table + load-time validator + `templates/builtin/default.toml` + agent binding fields on kind definitions. Largest unit; biggest new code.
- **Planner C — STEWARD auth + template auto-generation.** `principal_type: steward` + auth-level state-lock + `metadata.owner` field + template auto-generation of STEWARD level_2 items on numbered-drop creation + template-defined STEWARD-owned drop kinds. Coupled to template system but separable.
- **Planner D — Adopter bootstrap + per-drop wrap-up.** `go-project-bootstrap` + `fe-project-bootstrap` skill updates + every `CLAUDE.md` template + agent file frontmatter reminders + cascade-vocabulary doc sweep. Mostly MD work.

**Alternative: single planner.** Drop 2 used one planner with a comprehensive scope brief and decomposed cleanly into 11 droplets. Drop 3 has more surfaces but a single planner could still handle it given clear briefing.

**Orchestrator recommendation:** parallel planners (A through D) running concurrently with read-only access to the shared spec. Each emits a sub-`PLAN.md` (under `workflow/drop_3/<UNIT>/PLAN.md` or similar). Orch synthesizes into the unified `## Planner` section. Build-droplets land with explicit cross-unit `blocked_by` wiring (e.g., Unit B depends on Unit A's structural_type enum being available; Unit C depends on Unit B's template system; Unit D depends on everything).

**Open question for dev:** parallel planners (A/B/C/D) or single planner? Pre-Phase-1 decision.

### Pre-MVP Constraints (Locked)

- No migration logic in Go code; dev fresh-DBs.
- Retroactive classification (PLAN.md § 19.3 bullet at line 1637) is REPLACED with: dev fresh-DBs after `structural_type` field lands. No SQL backfill.
- No closeout MD artifacts.
- Builders are opus.

### Cross-Cutting Architectural Decisions (Confirm Before Phase 1)

- **`metadata.structural_type` field placement.** First-class domain field on `ActionItem` (mirroring Drop 2.3's `Role` field) OR remain on `metadata` JSON? Recommend first-class for consistency with role.
- **Template directory location.** `<project_root>/.tillsyn/template.toml` per spec — confirm. Alternative: `templates/<project_slug>.toml` if multi-tenant.
- **Default template scope.** `templates/builtin/default.toml` ships in-tree (re-introduces the `templates/` package Drop 2.1 deleted). Confirm dev wants the package restored under the new schema, not a separate location.
- **STEWARD `metadata.owner` field.** First-class domain field OR `metadata` JSON? Recommend first-class for auth-layer enforcement.
