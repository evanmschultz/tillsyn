# DROP_3 ROUND 2 — PLAN-QA PROOF REVIEW

**Reviewer:** `go-qa-proof-agent` (Round 2 unified-plan PROOF pass)
**Target:** `workflow/drop_3/PLAN.md` `## Planner` section (28 droplets, ~707 lines)
**Date:** 2026-05-02
**Verdict:** **PASS WITH MITIGATION-REQUIRED FINDINGS**

The unified plan correctly synthesizes the Round 1 4-unit decomposition + 8 dev-locked decisions + cascade-methodology integration into a single 28-droplet shape. Line citations against the actual codebase are materially accurate (the Round 1 fictional-cite FAIL on `repo.go:1414/1452/2500` is corrected). Cross-unit blocker wiring is acyclic and the topological sort is valid. However, the plan ships at least one BLOCKING contradiction (double-claimed `Persistent`/`DevGated` field landing on droplets 3.2 AND 3.17), several BLOCKING file-level blocker discipline gaps (3.4 vs 3.21, 3.4 vs 3.19, 3.5 vs 3.21, 3.3 vs 3.18, 3.3 vs 3.15, 3.18 vs 3.15, 3.2 vs 3.17, 3.1 vs 3.17), and one MITIGATION gap on `Irreducible` SQLite persistence. None require re-spawning the planner — orchestrator can synthesize fixes directly into the droplet `Blocked by` lines and the field-landing assignment.

---

## 1. Required Proof Checks

### 1.1 Coverage of REVISION_BRIEF inputs — PASS-WITH-NIT

**§2 — 8 dev-locked decisions:** all 8 surface in the "Locked Architectural Decisions" block as L1–L8 with correct droplet pointers and resolutions verbatim from the brief.

| Brief decision | Plan label | Lands in droplet | Verified |
| --- | --- | --- | --- |
| 2.1 — UpdateActionItem field-level write guard | L1 | 3.19 | ✓ (explicit acceptance bullet at line 444) |
| 2.2 — autent boundary-map | L2 | 3.19 | ✓ (line 440) |
| 2.3 — STEWARD persistent parents seeded | L3 | 3.14 | ✓ (line 348) |
| 2.4 — `internal/templates/builtin/default.toml` | L4 | 3.14 | ✓ (line 343) |
| 2.5 — `KindCatalog` lazy-decode | L5 | 3.12 | ✓ (line 310) |
| 2.6 — `GateRule` defers to Drop 4 | L6 | 3.8 | ✓ (line 252) |
| 2.7 — `KindRule.Owner` | L7 | 3.8 + 3.14 + 3.20 | ✓ (lines 249, 348, 469) |
| 2.8 — `ReparentActionItem` gate | L8 | 3.19 + 3.22 | ✓ (lines 445, 512) |

**§3 — methodology integration:** all 5 items surface as L9–L13 in the methodology-integration block. NIT — see 1.5 below for placement-contradiction details on L9/L10/L11.

**§4 — 8 mechanical fixes:** all 8 surface in the "Mechanical Fix Log" at lines 651–658 with concrete droplet pointers; spot-checks against the actual content confirm each fix is genuinely applied (not just listed).

**§5 — per-unit findings (~30 items):** representative spot checks all map to concrete droplet acceptance criteria or planning-block decisions. Examples:
- 5.A.1 (fictional INSERT/UPDATE cites in 3.A.3) → 3.3 acceptance line 158 corrects to ONE INSERT (`:1253`) + ONE UPDATE (`:1347`) + 3 SELECT projection sweeps.
- 5.A.7 (CE1 ~96 NewActionItem call sites) → 3.2 line 143 introduces `newActionItemForTest` helper + cross-package sweep enumerated.
- 5.B.13 (CE8 wire-surface deletion) → 3.15 line 374 explicitly DELETEs `till.kind operation=upsert` + legacy alias + CLI.
- 5.C.15 (N3 field naming) → 3.19 line 439 renames to `AuthRequestPrincipalType`.
- 5.D.5 (F5 4 architectural questions) → 3.24/3.25/3.27/3.28 land each lock per the brief.

### 1.2 Line-citation accuracy — PASS

Round 1 had a known FAIL on `repo.go:1414, :1452, :2500` (Unit A 3.A.3 claimed they were INSERTs/UPDATEs; they are SELECT projections). The unified plan corrects this in droplet 3.3 acceptance bullet at line 158 — explicitly: "exactly ONE column-bearing INSERT (`:1253`) and ONE column-bearing UPDATE (`:1347`). The line cites at `:1414`, `:1452`, `:2500` are SELECT projections, NOT additional INSERT/UPDATE paths."

Spot-checked against actual `repo.go`:

| Plan cite | Actual content | Verdict |
| --- | --- | --- |
| `repo.go:168-197` — CREATE TABLE action_items | INSERT at line 168 starts `CREATE TABLE IF NOT EXISTS action_items`; ends `:197` | ✓ |
| `repo.go:1253` — INSERT | `INSERT INTO action_items(...)` confirmed | ✓ |
| `repo.go:1347` — UPDATE | `UPDATE action_items SET ...` confirmed | ✓ |
| `repo.go:1414` — `ListActionItems` SELECT | confirmed (SELECT column-list line) | ✓ |
| `repo.go:1452` — `ListActionItemsByParent` SELECT | confirmed | ✓ |
| `repo.go:2500` — `getActionItemByID` SELECT | confirmed | ✓ |
| `repo.go:518` — `workItemAlterStatements` start | confirmed | ✓ |
| `repo.go:2846` — `t.Role = domain.Role(roleRaw)` | confirmed | ✓ |
| `repo.go:286-377` — `kind_catalog` boot-seed | confirmed (CREATE TABLE + seed INSERTs span this range) | ✓ |
| `domain/action_item.go:33` — `Role` field | confirmed | ✓ |
| `domain/action_item.go:150-158` — Role validation block | confirmed (`NormalizeRole` + `IsValidRole` + `ErrInvalidRole`) | ✓ |
| `domain/role_test.go:120` — regex character-class comment | confirmed (`captures [a-z-]+ only`) | ✓ |
| `domain/kind.go:94-102` — `KindTemplateChildSpec` | confirmed | ✓ |
| `domain/kind.go:104-110` — `KindTemplate` | confirmed | ✓ |
| `domain/kind.go:118` — `AllowedParentScopes` field | confirmed | ✓ |
| `domain/kind.go:200-211` — `AllowsParentScope` method | confirmed | ✓ |
| `domain/kind.go:274-293` — `normalizeKindParentScopes` | confirmed | ✓ |
| `domain/kind.go:296-352` — `normalizeKindTemplate` | confirmed | ✓ |
| `domain/auth_request.go:596-608` — `normalizeAuthRequestPrincipalType` | confirmed (3 cases: user/agent/service) | ✓ |
| `domain/auth_request.go:393-405` — `NewAuthRequest` validation block | confirmed | ✓ |
| `app/auth_requests.go:35` — `AuthSessionIssueInput.PrincipalType` | confirmed | ✓ |
| `app/auth_requests.go:60` — `AuthSession.PrincipalType` | confirmed | ✓ |
| `app/service.go:404` — `CreateActionItemInput` | confirmed | ✓ |
| `app/service.go:429` — `UpdateActionItemInput` | confirmed | ✓ |
| `app/service.go:574` — `domain.NewActionItem` call | confirmed | ✓ |
| `app/service.go:784-794` — Role update block | confirmed | ✓ |
| `app/service.go:1106-1156` — `ReparentActionItem` | confirmed | ✓ |
| `app/snapshot.go:57-84` — `SnapshotActionItem` struct | confirmed (Role at `:63`) | ✓ |
| `app/snapshot.go:1059`/`:1060` — `snapshotActionItemFromDomain` | confirmed (declaration `:1060`; comment `:1059`) | minor (the plan cites `:1059` and `:1060` interchangeably; both refer to the same function — acceptable) |
| `app/snapshot.go:1267-1278` — `(SnapshotActionItem).toDomain` + Kind fallback | confirmed | ✓ |
| `app/snapshot_test.go:442` — `TestSnapshotActionItemRoleRoundTripPreservesAllRoles` | confirmed | ✓ |
| `app/kind_capability.go:545-578` — `resolveActionItemKindDefinition` | confirmed | ✓ |
| `app/kind_capability.go:566` — `kind.AllowsParentScope(parent.Scope)` | confirmed | ✓ |
| `app/kind_capability.go:750-766` — `mergeActionItemMetadataWithKindTemplate` | confirmed | ✓ |
| `app/kind_capability.go:771` — `validateKindTemplateExpansion` | confirmed | ✓ |
| `app/kind_capability.go:751-799` — recursive expansion | confirmed | ✓ |
| `mcpapi/extended_tools.go:860` — inline anonymous-struct args | confirmed | ✓ |
| `mcpapi/extended_tools.go:866` — `Role` field in args | confirmed | ✓ |
| `mcpapi/extended_tools.go:~1033` (case "create") | confirmed (case opens at `:987`; CreateActionItem call at `:1033`) | ✓ (the `~` qualifier on `~1033` is honest about line drift) |
| `mcpapi/extended_tools.go:~1092` (case "update" CreateActionItem call) | actual `case "update"` at `:1056`; UpdateActionItem call at `:1092` | ✓ |
| `mcpapi/extended_tools.go:1370` — `mcp.WithString("role", ...)` | confirmed | ✓ |
| `mcpapi/extended_tools.go:1417`/`:1444` — `till.create_task`/`till.update_task` | actual at `:1416`/`:1443` (1-line drift) | NIT (off-by-one but unambiguous; builder will find them) |
| `mcpapi/extended_tools.go:1682` — `Template domain.KindTemplate` (UpsertKindDefinition body) | confirmed | ✓ |
| `mcpapi/extended_tools.go:1778` — `Template domain.KindTemplate` (legacy alias body) | confirmed | ✓ |
| `mcpapi/extended_tools_test.go:429` — `stubExpandedService.CreateActionItem` | confirmed | ✓ |
| `common/app_service_adapter_mcp.go:666-684` — `CreateActionItem` | confirmed | ✓ |
| `common/app_service_adapter_mcp.go:708-720` — `UpdateActionItem` | confirmed | ✓ |
| `common/app_service_adapter_mcp.go:728` — `MoveActionItem` | confirmed | ✓ |
| `common/app_service_adapter_mcp.go:744` — `MoveActionItemState` | confirmed | ✓ |
| `common/app_service_adapter_mcp.go:760` — `GetActionItem` callsite inside MoveActionItemState | confirmed | ✓ |
| `common/app_service_adapter_mcp.go:810-823` — `ReparentActionItem` | confirmed | ✓ |
| `common/mcp_surface.go:57` — `CreateActionItemRequest` | confirmed | ✓ |
| `common/mcp_surface.go:78` — `UpdateActionItemRequest` | confirmed | ✓ |
| `common/mcp_surface.go:248` — `Template domain.KindTemplate` | confirmed | ✓ |
| `autentauth/service.go:191` — `ensurePrincipal` callsite | confirmed (inside `IssueSession`; correct boundary for `steward → agent` mapping) | ✓ |
| `autentauth/service.go:803-812` — `principalTypeToActorType` | confirmed | ✓ |
| `cmd/till/main.go:3042-3442` — `parseOptionalKindTemplateJSON` + `kind upsert` CLI | actual `parseOptionalKindTemplateJSON` declared at `:3041` (1-line drift); CLI subtree at `:895-1014`; `runOneShotCommand("kind.upsert", ...)` at `:2421` | NIT (off-by-one; but builder LSP-finds the symbols anyway) |
| `cmd/till/main.go:3617, 3619` — kindDefinitionPayload fields | confirmed | ✓ |
| `tui/model_test.go:14674-14687` — `newActionItemForTest` helper | confirmed | ✓ |
| `app/snapshot.go:94` — `Template domain.KindTemplate` in SnapshotKindDefinition | confirmed | ✓ |
| `repo_test.go:2470-2517` — `TestRepositoryFreshOpenKindCatalog` | function declared at `:2468`; ends `:2518` (range close enough; `:2470` is first body line) | ✓ |
| `repo_test.go:2520-2568` — `TestRepositoryFreshOpenKindCatalogUniversalParentAllow` | declared at `:2525`; ends `:2568` | ✓ |
| `repo_test.go:2563` — `kind.AllowsParentScope` callsite | confirmed | ✓ |
| `WIKI.md:34/:36` — placement between `### Do Not Use Templates Right Now` and `## Level Addressing` | confirmed | ✓ |
| `WIKI.md:19/36/47` — h2 hierarchy reference points | confirmed (all three are `## ` h2 sections) | ✓ |
| `~/.claude/agents/go-qa-falsification-agent.md:95-108` — `## Go Falsification Attacks` section | confirmed | ✓ |

Verdict: PASS. Aggregate accuracy is materially perfect; line drifts are off-by-one or `~` approximations clearly marked as such. The Round 1 fictional-cite issue is genuinely fixed.

### 1.3 Renumbering Map consistency — PASS

The renumbering map at lines 78–107 maps each `3.N` to its origin `3.A.k` / `3.B.k` / `3.C.k` / `3.D.k`. Every `Blocked by` reference inside droplet bodies uses the new `3.N` IDs consistently — verified by `rg -n "blocked_by.*3\.[ABCD]\." workflow/drop_3/PLAN.md` (no hits inside `Blocked by` lines except as origin tags in droplet headers — those are documentation, not blocker references). No leftover `5.D.k` from the Unit D renumbering.

### 1.4 Cross-unit blocker wiring acyclicity — PASS-WITH-NIT

Walked the full `Blocked by` graph from each droplet body (lines 127–622). All references point to lower-numbered droplets. No cycles. Topological sort `3.1 → 3.2 → ... → 3.28` is valid.

The three-way write conflict on `~/.claude/agents/go-qa-falsification-agent.md` (3.7 → 3.23 → 3.27) is explicitly documented at lines 535 (3.23 `Blocked by: 3.7`), 602 (3.27 `Blocked by: 3.23`), and the Mechanical Fix Log line 656.

NIT — table data row at line 633 reads `Blocker: 3.5 | Blocked: 3.4` with parenthetical "(Reverse-listed; effectively 3.5 ← 3.4 chain.)" admitting the row is inverted. The droplet body at line 198 says correctly `3.5 blocked by 3.4`. The reader has to do the mental flip to reconcile. Cleanup: rewrite the row as `Blocker: 3.4 | Blocked: 3.5` with no parenthetical.

### 1.5 Methodology compliance — FAIL (BLOCKING)

The 6th plan-QA-falsification attack vector (§4.4 global L1 sweep) lands in 3.7 acceptance line 234 with all four checks (acyclicity, sibling overlap, leaf composition, orphan detection) and a `ta-docs/cascade-methodology.md` §4.4 inline cite. ✓

The three new first-class fields per cascade-methodology §11.2 / §11.3 land — but with a contradiction:

- **L9 / L10 (Persistent + DevGated)** locked-decisions block (lines 64–66) say definitively: "Lands in **3.17** alongside `Owner` + `DropNumber` first-class fields."
- **L11 (Irreducible)** locked-decisions block line 68 says: "Lands as a domain field on `ActionItem` in **3.2** (extends Unit A's `StructuralType`-on-`ActionItem` droplet — same struct edit)."

So per the locked decisions, **3.2 should add `StructuralType + Irreducible` (2 fields)** and **3.17 should add `Owner + DropNumber + Persistent + DevGated` (4 fields)**.

But:

- **Droplet 3.2 acceptance** line 136: "ActionItem struct gains four fields after Role (line 33): `StructuralType StructuralType`, `Persistent bool`, `DevGated bool`, `Irreducible bool`." (4 fields, double-claims Persistent + DevGated)
- **Droplet 3.17 acceptance** line 404: "ActionItem struct gains four fields after Role (`:33`): `Owner string`, `DropNumber int`, `Persistent bool`, `DevGated bool` (per L9 + L10)." (4 fields, also claims Persistent + DevGated)
- **Renumbering map** lines 81 + 96 list both droplets as carrying Persistent + DevGated.

Both droplets cannot land Persistent + DevGated. This is an internal contradiction between the locked-decisions block (correct: 3.17) and the droplet acceptance bodies (wrong: lists Persistent+DevGated under both). **BLOCKING FINDING.**

The 5th methodology integration item (L13 — reframe Unit C scope language as domain primitive) is reflected in the droplet bodies (e.g., 3.17 line 406 explicitly says "all four are domain primitives. STEWARD is just one consumer"). ✓

### 1.6 Pre-MVP rule compliance — PASS

Spot-checked against memory rules:

- **No migration logic.** Out-of-scope confirmation at line 670 explicitly: "no Go code, no `till migrate` subcommand, no SQL backfill." Each schema-touching droplet (3.3, 3.12, 3.15, 3.18, 3.19, 3.20, 3.22) carries an explicit `**DB action:** dev DELETEs ~/.tillsyn/tillsyn.db` instruction. ✓
- **No closeout MD rollups.** Out-of-scope confirmation at line 671 explicitly disclaims `CLOSEOUT.md / LEDGER.md / WIKI_CHANGELOG.md / REFINEMENTS.md / HYLLA_FEEDBACK.md / HYLLA_REFINEMENTS.md`. ✓
- **Builders run opus.** Stated at lines 32 + 38. ✓
- **Never `mage install`.** Stated at line 38. ✓
- **Never `git rm` workflow files.** No droplet contains a `git rm` instruction; 3.27 + 3.28 sweeps add lines, never delete files. ✓

### 1.7 Same-package / same-file blocker discipline — FAIL (multiple BLOCKING findings)

Per CLAUDE.md §"Cascade Tree Structure" → "Blocker Semantics": *"sibling build-tasks sharing a file in `paths` OR a package in `packages` MUST have an explicit `blocked_by` between them."* Walked every pair of droplets sharing a file or package; the following gaps surfaced:

#### 1.7.a — `internal/domain/action_item.go` (3.2 ↔ 3.17). **BLOCKING.**

3.2 paths: `internal/domain/action_item.go`, `internal/domain/domain_test.go`, `internal/domain/action_item_test_helpers.go` (recommended).
3.17 paths: `internal/domain/action_item.go`, `internal/domain/domain_test.go`, `internal/domain/errors.go`.

Both droplets edit the same `ActionItem` struct (and `ActionItemInput`) at the same insertion point (after `Role`, line 33). 3.17 currently says `Blocked by: —` (line 415). Without an explicit `3.17 blocked_by 3.2` (or vice-versa) the two builders race on the same struct definition.

**Mitigation**: add `3.17 blocked_by 3.2`. After the Persistent/DevGated double-landing fix (1.5), 3.2 lands StructuralType+Irreducible and 3.17 appends Owner+DropNumber+Persistent+DevGated downstream — clean serialization.

#### 1.7.b — `internal/domain/errors.go` (3.1 ↔ 3.17). **BLOCKING.**

3.1 paths include `internal/domain/errors.go` (adds `ErrInvalidStructuralType`).
3.17 paths include `internal/domain/errors.go` (adds `ErrInvalidDropNumber`).

3.17 has no `Blocked by` edge to 3.1 currently. Both append a sentinel at the same conventional position. **Mitigation**: add `3.17 blocked_by 3.1` (transitively satisfied if 1.7.a's `3.17 blocked_by 3.2` lands, since `3.2 blocked_by 3.1`).

#### 1.7.c — `internal/adapters/storage/sqlite/repo.go` + `repo_test.go` (3.3 ↔ 3.18). **BLOCKING.**

3.3 modifies `CREATE TABLE action_items` at `:168-197` inserting `structural_type` after `role` at `:174`; updates INSERT (`:1253`), UPDATE (`:1347`), and 3 SELECTs.
3.18 modifies the same `CREATE TABLE` inserting `owner`, `drop_number`, `persistent`, `dev_gated` after `role` at `:174`; updates INSERT, UPDATE, and SELECTs.

If both fire concurrently they collide on the column ordinal positions inserted after `role`. After 3.3 lands, line 174 is still `role` but line 175 is `structural_type` (NEW); 3.18 must then insert AFTER `structural_type` (at `:176+`) to keep ordinals consistent with the SELECT lists 3.3 already extended. Currently 3.18 says "appended after `role` (`:174`)" without a blocker on 3.3 — race condition.

**Mitigation**: add `3.18 blocked_by 3.3`; rewrite 3.18 acceptance bullet from "appended after `role`" to "appended after `structural_type` (which 3.3 added)."

#### 1.7.d — `internal/adapters/storage/sqlite/repo.go` + `repo_test.go` (3.3 ↔ 3.15, 3.15 ↔ 3.18). **BLOCKING.**

3.15 deletes the `kind_catalog` boot-seed at `:286-377` and rewrites SELECT lists at `:1061, :1066, :1070, :1095, :1100, :1104, :1130, :1140, :1156, :2940, :2942, :2964, :2966, :2970, :2976, :2981, :2982, :2987, :2988`. Some of those SELECTs are the exact same `action_items` SELECTs that 3.3 + 3.18 already touched (at `:1414, :1452, :2500`).

3.15 has `Blocked by: 3.10, 3.11, 3.12` — no edge to 3.3 or 3.18. Race condition on SELECT column-list edits.

**Mitigation**: add `3.15 blocked_by 3.3` AND `3.18 blocked_by 3.15` (or some serialization that orders all three). Note 3.15's path list explicitly cites "starting-point lower bound, not closed enumeration" — but the file-level lock still applies.

#### 1.7.e — `internal/app/snapshot.go` + `snapshot_test.go` (3.5 ↔ 3.21). **BLOCKING.**

3.5 paths: `internal/app/snapshot.go`, `internal/app/snapshot_test.go` — adds `StructuralType` to `SnapshotActionItem` at `:63`, threads through `snapshotActionItemFromDomain` (`:1060`) and `(SnapshotActionItem).toDomain` (`:1267`).
3.21 paths: `internal/app/snapshot.go`, `internal/app/snapshot_test.go` (among others) — adds `Owner`, `DropNumber`, `Persistent`, `DevGated` to `SnapshotActionItem` after `Role` at `:63`, threads through the same two functions.

3.21 has `Blocked by: 3.18, 3.19` — no edge to 3.5. Both edit the same struct at the same insertion point and the same two helper functions.

**Mitigation**: add `3.21 blocked_by 3.5`.

#### 1.7.f — `internal/adapters/server/common/app_service_adapter_mcp.go` (3.4 ↔ 3.19, 3.4 ↔ 3.21). **BLOCKING.**

3.4 paths: `internal/app/service.go`, `internal/adapters/server/common/mcp_surface.go`, `internal/adapters/server/common/app_service_adapter_mcp.go`, `internal/adapters/server/mcpapi/extended_tools.go`, `internal/adapters/server/mcpapi/extended_tools_test.go` — threads `StructuralType`.

3.19 paths include `internal/adapters/server/common/app_service_adapter_mcp.go` (auth gates).
3.21 paths include the same `app_service_adapter_mcp.go` (`Owner` + `DropNumber` + `Persistent` + `DevGated` threading through `CreateActionItem` + `UpdateActionItem`) AND `mcp_surface.go`, `extended_tools.go`, `extended_tools_test.go`.

3.19 has `Blocked by: 3.18` — no edge to 3.4. 3.21 has `Blocked by: 3.18, 3.19` — no edge to 3.4. All three touch `app_service_adapter_mcp.go`'s `CreateActionItem`/`UpdateActionItem`/`MoveActionItem`/`MoveActionItemState`/`ReparentActionItem` block.

**Mitigation**: add `3.19 blocked_by 3.4` AND `3.21 blocked_by 3.4`. (3.4 → 3.5 → ... → 3.17 → 3.18 → 3.19 → 3.21 then becomes a clean chain.) Also same for shared files `mcp_surface.go`, `extended_tools.go`, `extended_tools_test.go` between 3.4 ↔ 3.21.

#### 1.7.g — `internal/templates/schema.go` + `schema_test.go` (3.8 ↔ 3.13). **PASS** (already wired)

3.13 says `Blocked by: 3.8 (file-level: same schema.go)` at line 335. ✓

#### 1.7.h — `internal/templates/builtin/default.toml` (3.14 ↔ 3.20). **PASS** (already wired transitively)

3.20 paths describe TOML body in the same `default.toml` 3.14 created. 3.20 has `Blocked by: 3.11, 3.19`. 3.20 also depends on 3.14 implicitly (the file must exist) — but no explicit edge. **NIT — minor**. Add `3.20 blocked_by 3.14` for explicit serialization on the same file.

### Summary of 1.7 mitigations

Adding 8 missing `blocked_by` edges: `3.17 ← 3.2`, `3.18 ← 3.3`, `3.15 ← 3.3`, `3.18 ← 3.15`, `3.21 ← 3.5`, `3.19 ← 3.4`, `3.21 ← 3.4`, `3.20 ← 3.14`. None creates a cycle (verified by walking the augmented DAG manually).

---

## 2. Findings

### 2.1 [BLOCKING] Persistent/DevGated double-landing on 3.2 + 3.17

3.2 acceptance (line 136) and 3.17 acceptance (line 404) both list `Persistent bool` + `DevGated bool` as fields landed on the `ActionItem` struct. The locked-decisions block lines 64–66 explicitly say L9 (Persistent) + L10 (DevGated) land in **3.17**, not 3.2. L11 (Irreducible) lands in 3.2.

**Resolution (orchestrator-applicable, no re-spawn):**
- Rewrite 3.2 acceptance line 136 to: "ActionItem struct gains TWO fields after Role (line 33): `StructuralType StructuralType`, `Irreducible bool`."
- Rewrite renumbering map entry 3.2 (line 81) to: "`StructuralType` + `Irreducible` fields on `ActionItem`".
- Rewrite the `[origin: ...]` tag on 3.2 (line 130) to drop L9 + L10: `[origin: 3.A.2 + L11]`.
- 3.17 stays as-is (Owner + DropNumber + Persistent + DevGated).
- Update §1.7.a mitigation accordingly.

### 2.2 [BLOCKING] 8 missing same-file `blocked_by` edges

See §1.7 above for the full list. Add the 8 edges: `3.17 ← 3.2`, `3.18 ← 3.3`, `3.15 ← 3.3`, `3.18 ← 3.15`, `3.21 ← 3.5`, `3.19 ← 3.4`, `3.21 ← 3.4`, `3.20 ← 3.14`.

### 2.3 [MITIGATION] `Irreducible` field has no SQLite persistence path

3.2 (post-1.5/2.1 fix) adds `Irreducible bool` to the `ActionItem` struct. 3.3 only adds `structural_type` column. 3.18 only adds `owner` + `drop_number` + `persistent` + `dev_gated` columns. There's no `irreducible` column anywhere. After the struct change + fresh-DB the field will round-trip as zero-value forever (no read path, no write path).

**Resolution:** either (a) extend 3.3 acceptance to also add `irreducible INTEGER NOT NULL DEFAULT 0` column + INSERT/UPDATE/SELECT/scanner threading; OR (b) explicit acceptance criterion in 3.2: "`Irreducible` is a struct-only flag for plan-QA-falsification consumption; no SQLite persistence in Drop 3 — defer column to Drop 4 where the dispatcher consumes it." Option (b) is simpler and matches the L11 wording ("dispatcher consumes; for now plan-QA-falsification's new attack-vector list includes 'irreducible-claim attack'").

### 2.4 [NIT] Reversed table row at line 633

The Cross-Unit Blocker Wiring table line 633 reads `Blocker: 3.5 | Blocked: 3.4` — reversed from the droplet body's actual `3.5 blocked_by 3.4`. The parenthetical "(Reverse-listed; ...)" admits the inversion but leaves it.

**Resolution:** rewrite the row as `Blocker: 3.4 | Blocked: 3.5 | Same-package serialization (internal/app) per finding 5.A.2 — explicit.` Drop the parenthetical.

### 2.5 [NIT] Misquoted methodology citation in L8

Plan L8 line 58 cites "Methodology §6.3 ('Failed nodes remain in their original tree position; they are not moved into a separate failed lane') confirms reparenting is a state-affecting mutation." The actual `ta-docs/cascade-methodology.md` §6.3 is titled "Planner Edits In Place" and contains no such quoted phrase. The conclusion (reparenting is a state-affecting mutation) is supportable on independent grounds — but the cited quote does not appear in the source.

**Resolution:** drop the quoted clause. Replace with: "Methodology §6.3 ('Planner Edits In Place') and §11.2's `owner` / state-transition framing make reparenting a state-affecting mutation."

### 2.6 [NIT] Off-by-one line drift on 1–2 cited lines

- `extended_tools.go:1417` actual `:1416` (`till.create_task`).
- `extended_tools.go:1444` actual `:1443` (`till.update_task`).
- `cmd/till/main.go:3042-3442` (`parseOptionalKindTemplateJSON`) actual `:3041-…`.
- `app/snapshot.go:1059`/`:1060` (`snapshotActionItemFromDomain`) the function declaration is `:1060`; comment header `:1059`. Plan cites both interchangeably.

These are all unambiguous — builder will LSP-find the symbols regardless. Listed for completeness.

### 2.7 [NIT] 3.20 implicitly depends on 3.14 without explicit edge

3.20 paths describe edits to `internal/templates/builtin/default.toml` (the file 3.14 creates). Currently 3.20 `Blocked by: 3.11, 3.19` — no explicit `3.14`.

**Resolution:** add `3.14` to 3.20's `Blocked by` line. Already in the §1.7.h / §2.2 mitigations list.

---

## 3. Coverage Matrix

| REVISION_BRIEF item | PLAN.md droplet / section | Verified |
| --- | --- | --- |
| §2.1 — UpdateActionItem field-level write guard | L1 → 3.19 (line 444) | ✓ |
| §2.2 — autent boundary-map | L2 → 3.19 (line 440) | ✓ |
| §2.3 — STEWARD persistent parents seeded | L3 → 3.14 (line 348) | ✓ |
| §2.4 — internal/templates/builtin/default.toml | L4 → 3.14 (line 343) | ✓ |
| §2.5 — KindCatalog lazy-decode | L5 → 3.12 (line 310) | ✓ |
| §2.6 — GateRule defers to Drop 4 | L6 → 3.8 (line 252) | ✓ |
| §2.7 — KindRule.Owner | L7 → 3.8 + 3.14 + 3.20 (lines 249, 348, 469) | ✓ |
| §2.8 — ReparentActionItem gate | L8 → 3.19 + 3.22 (lines 445, 512) | ✓ (NIT — see 2.5) |
| §3.1 — Persistent first-class field | L9 → 3.17 (line 64) | ✓ (BLOCKING — see 2.1) |
| §3.2 — DevGated first-class field | L10 → 3.17 (line 66) | ✓ (BLOCKING — see 2.1) |
| §3.3 — Irreducible flag | L11 → 3.2 (line 68) | ✓ (MITIGATION — see 2.3) |
| §3.4 — 6th plan-QA-falsification attack vector | L12 → 3.7 (line 234) | ✓ |
| §3.5 — Reframe Unit C scope language | L13 → 3.17, 3.20, 3.21, 3.22 | ✓ |
| §4.1 — fictional INSERT/UPDATE cites | 3.3 acceptance (line 158) | ✓ |
| §4.2 — 3.A.5 same-package serialization | 3.5 `Blocked by: 3.4` (line 198) | ✓ |
| §4.3 — 3.C.5 blocked_by 3.C.3 | 3.21 `Blocked by: 3.18, 3.19` (line 501) | ✓ (3.19 transitively pulls 3.18, satisfies the brief intent) |
| §4.4 — Unit D renumbering 5.D → 3.D | renumbering map lines 102–107 | ✓ |
| §4.5 — drop_1_5/ + drop_1_75/ exclusions | 3.27 line 587 + 3.28 line 607 | ✓ |
| §4.6 — three-way write conflict on go-qa-falsification-agent.md | chain 3.7 → 3.23 → 3.27 (lines 535, 602, 656) | ✓ |
| §4.7 — Unit B 3.B.8 LSP-found additional sites | 3.15 lines 359–375 | ✓ |
| §4.8 — Unit C 3.C.3 supersede path | 3.19 acceptance Test 7 (line 446) + 3.22 Test 5 (line 513) | ✓ |
| §5.A.1 — fictional INSERT/UPDATE cites | 3.3 (per §4.1) | ✓ |
| §5.A.2 — same-package serialization | 3.5 ← 3.4 | ✓ |
| §5.A.3 — regex-comment caveat | 3.1 acceptance line 125 | ✓ |
| §5.A.4 — sweep-scope enumeration | 3.2 acceptance line 144 | ✓ |
| §5.A.5 — WIKI placement | 3.6 acceptance line 206 | ✓ |
| §5.A.6 — migration block line cite | 3.3 acceptance line 162 | ✓ |
| §5.A.7 — CE1 newActionItemForTest helper | 3.2 acceptance line 143 | ✓ |
| §5.A.8 — CE3 three-way write conflict | chain in Mech Fix Log line 656 | ✓ |
| §5.A.9 — CE5 stub permissive default | 3.4 acceptance line 182 | ✓ |
| §5.A.10 — CE4 regex narrowing | 3.1 acceptance line 123 (builder choice) | ✓ |
| §5.A.11 — C6 placement fix | absorbed into Locked block (line 240 callout) | ✓ |
| §5.A.12 — Hylla-first process gap | not a plan correctness issue (process feedback) | ✓ |
| §5.B.1 — 3.B.8 path enumeration starting-point | 3.15 line 375 disclaimer | ✓ |
| §5.B.2 — 3.B.5 import-direction | L5 → 3.12 | ✓ |
| §5.B.3 — internal/templates/ package rationale | 3.8 — implicit (location asserted in the path) | ✓ |
| §5.B.4 — cycle-detection algorithm | 3.9 acceptance line 269 | ✓ |
| §5.B.5 — Tools []string validation deferred | 3.13 acceptance line 330 | ✓ |
| §5.B.6 — CE1 //go:embed ../.. | L4 → 3.14 | ✓ |
| §5.B.7 — CE2 missing call sites | 3.15 lines 359–375 (LSP-found extras) | ✓ |
| §5.B.8 — CE3 deletes kind_catalog tests | 3.14 acceptance line 351 + 3.15 acceptance line 371 | ✓ |
| §5.B.9 — CE4 KindCatalog import direction | L5 → 3.12 | ✓ |
| §5.B.10 — CE5 schema-version pre-pass | 3.9 acceptance line 267 | ✓ |
| §5.B.11 — CE6 GateRule undefined | L6 → 3.8 | ✓ |
| §5.B.12 — CE7 144-row drift | 3.10 acceptance line 283 + 3.14 acceptance line 350 | ✓ |
| §5.B.13 — CE8 wire-surface deletion | 3.15 acceptance line 374 | ✓ |
| §5.B.14 — N1 runtime-mutability | 3.12 acceptance line 315 | ✓ |
| §5.B.15 — N2 rejection-comment scope | 3.16 acceptance line 391 | ✓ |
| §5.B.16 — N3 explicit deny rows | 3.14 acceptance line 346 | ✓ |
| §5.B.17 — N4 AgentBinding split commit | 3.8 acceptance line 251 + 3.13 acceptance line 329 | ✓ |
| §5.C.1 — MoveActionItem column-only pre-fetch | 3.19 acceptance line 443 | ✓ |
| §5.C.2 — placeholder resolved to 3.B.4 | 3.20 acceptance line 482 | ✓ |
| §5.C.3 — 3.C.5 blocked_by 3.C.3 | 3.21 `Blocked by: 3.18, 3.19` | ✓ |
| §5.C.4 — bullet 9 cross-ref | L7 → 3.8 + 3.14 + 3.20 | ✓ |
| §5.C.5 — index coverage | 3.18 acceptance line 426 | ✓ |
| §5.C.6 — C1 UpdateActionItem state-lock | L1 → 3.19 | ✓ |
| §5.C.7 — C2 ReparentActionItem unguarded | L8 → 3.19 + 3.22 | ✓ |
| §5.C.8 — C3 autent collision | L2 → 3.19 | ✓ |
| §5.C.9 — C4 MoveActionItem no pre-fetch | 5.C.1 above | ✓ |
| §5.C.10 — C5 STEWARD parent seeding | L3 → 3.14 | ✓ |
| §5.C.11 — C6 refinements-gate forgetfulness | 3.22 Test 7 (line 515) | ✓ |
| §5.C.12 — C7 bullet 9 mechanism | L7 → 3.8 + 3.14 + 3.20 | ✓ |
| §5.C.13 — C8 supersede path | 3.19 Test 7 (line 446) + 3.22 Test 5 | ✓ |
| §5.C.14 — N1 state-neutral lock | 3.19 acceptance line 447 | ✓ |
| §5.C.15 — N3 field renaming | 3.19 acceptance line 439 (`AuthRequestPrincipalType`) | ✓ |
| §5.C.16 — N4 rollback-cost note | 3.17 acceptance line 411 | ✓ |
| §5.C.17 — N5 index design | 3.18 acceptance line 426 | ✓ |
| §5.D.1 — F1 exclusion list missing drops | 3.27 line 587 + 3.28 line 607 | ✓ |
| §5.D.2 — F2 split timing rationale | 3.28 acceptance line 617 | ✓ |
| §5.D.3 — F3 file-level race | chain 3.7 → 3.23 → 3.27 | ✓ |
| §5.D.4 — F4 subagent vs orch MD work | 3.27 acceptance line 598 | ✓ |
| §5.D.5 — F5 4 architectural questions | 3.24 + 3.25 + 3.27 + 3.28 (locked per brief) | ✓ |
| §5.D.6 — three-way conflict | per §4.6 / §5.D.3 | ✓ |
| §5.D.7 — drop_1_5/ + drop_1_75/ exclusion | per §4.5 / §5.D.1 | ✓ |
| §5.D.8 — ~/.claude/CLAUDE.md retired-vocab | 3.27 acceptance line 583 | ✓ |
| §5.D.9 — bootstrap WIKI scoping | 3.24 + 3.25 narrowed (lines 544, 558) | ✓ |
| §5.D.10 — worklog vs commit boundary | 3.27 acceptance line 597 | ✓ |
| §5.D.11 — workflow/example/CLAUDE.md insertion | 3.26 acceptance line 569 | ✓ |
| §5.D.12 — wrap-up timing | 3.28 acceptance line 617 | ✓ |
| §5.D.13 — skill frontmatter rationale | 3.24 acceptance line 546 | ✓ |
| §5.D.14 — slash-command files | 3.27 acceptance line 584 | ✓ |

---

## 4. Verdict Summary

**PASS WITH MITIGATION-REQUIRED FINDINGS.**

The unified plan has materially correct line citations (Round 1's fictional-cite FAIL is genuinely fixed), full coverage of all 8 dev-locked decisions, all 5 methodology-integration items, all 8 mechanical fixes, and all ~30 per-unit findings from REVISION_BRIEF §5. Pre-MVP rules are honored (no migration logic, no closeout MDs, builders=opus, never `mage install`, never `git rm` workflow files). The cross-unit blocker graph is acyclic and the topological sort is valid.

The plan ships TWO real BLOCKING contradictions and ONE MITIGATION gap:

1. **BLOCKING 2.1** — Persistent/DevGated double-landing on 3.2 + 3.17. Locked-decisions block (correct: lands in 3.17) contradicts droplet acceptance bodies (wrong: lists Persistent+DevGated under both 3.2 + 3.17). One-line orchestrator fix: rewrite 3.2 acceptance + renumbering map entry to `StructuralType + Irreducible` only.
2. **BLOCKING 2.2** — 8 missing same-file `blocked_by` edges (3.17 ← 3.2; 3.18 ← 3.3; 3.15 ← 3.3; 3.18 ← 3.15; 3.21 ← 3.5; 3.19 ← 3.4; 3.21 ← 3.4; 3.20 ← 3.14). All shared-file races between droplets that touch the same Go file. Orchestrator can patch each droplet's `Blocked by` line directly. The augmented DAG remains acyclic.
3. **MITIGATION 2.3** — `Irreducible` field has no SQLite persistence path. Either add a column to 3.3 / 3.18 OR document it as struct-only-for-Drop-3 with the column deferred to Drop 4.

Plus 4 NIT-level cleanups (table row reversal, methodology misquote, two off-by-one line cites, one implicit-but-unstated 3.20 ← 3.14 edge already covered in 2.2).

None of these requires re-spawning the planner. The orchestrator can synthesize all 3 BLOCKING/MITIGATION fixes into the existing droplet bodies + cross-unit table directly. The plan's structural decomposition, paths/packages, acceptance criteria depth, and traceability to REVISION_BRIEF inputs are all sound — these are surgical inconsistencies in the integration layer, not decomposition flaws.

---

## 5. Hylla Feedback

N/A — task touched non-Go files only (workflow/drop_3/PLAN.md, REVISION_BRIEF.md, ta-docs/cascade-methodology.md, WIKI.md, ~/.claude/agents/go-qa-falsification-agent.md). Go source verification went through `Read` + `LSP` for line-precise citation checks; no Hylla queries were issued because the proof-pass shape (verify a specific line/symbol exists at a specific cited location) is a direct-read pattern, not a search-and-discover pattern. Hylla would have been correct first-stop for "find every NewActionItem call site" or "find every reference to KindTemplate" — those queries did not arise during this proof pass because the plan's line citations were already symbol-precise.
