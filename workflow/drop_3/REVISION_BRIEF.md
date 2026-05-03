# DROP_3 Round 2 Revision Brief

**Author:** orchestrator (Drop 3 planning, post-Round-1 synthesis)
**Date:** 2026-05-02
**Audience:** `go-planning-agent` Round 2 unified planner subagent
**Output target:** `workflow/drop_3/PLAN.md` — fill the `## Planner` section with the unified droplet decomposition (~28 droplets, renumbered `3.1`–`3.N` with original unit-letter retained inline as `3.A.k`/`3.B.k`/`3.C.k`/`3.D.k` for traceability and `blocked_by` clarity).

This brief is self-contained. Read the inputs listed in §1, apply the resolutions in §2–§5, and produce one unified `## Planner` section.

---

## 1. Inputs

Read in order:

1. **`workflow/drop_3/PLAN.md`** — the drop-level scope brief (§Scope + §Notes). Already has scope + locked architectural decisions + cross-cutting question routing.
2. **Round 1 unit plans** (4 files):
   - `workflow/drop_3/UNIT_A_PLAN.md` — Cascade vocabulary foundation, 7 droplets (3.A.1–3.A.7).
   - `workflow/drop_3/UNIT_B_PLAN.md` — Template system overhaul, 9 droplets (3.B.1–3.B.9).
   - `workflow/drop_3/UNIT_C_PLAN.md` — STEWARD auth + auto-gen, 6 droplets (3.C.1–3.C.6).
   - `workflow/drop_3/UNIT_D_PLAN.md` — Adopter bootstrap + doc sweep, 6 droplets (currently mis-numbered as `5.D.1`–`5.D.6` per Unit D's planner; renumber to `3.D.1`–`3.D.6`).
3. **Round 1 plan-QA artifacts** (8 files):
   - `workflow/drop_3/UNIT_{A,B,C,D}_PLAN_QA_PROOF.md`
   - `workflow/drop_3/UNIT_{A,B,C,D}_PLAN_QA_FALSIFICATION.md`
4. **Methodology spec (THE architectural canonical):**
   - `ta-docs/cascade-methodology.md` — §1 thesis, §2 droplet shape, §3 roles/models, §4 QA placement, §5 nesting, §6 failure handling, §7 re-QA invariants, §8 audit, §9 worked example, §11 canonical NodeBase + ActionItem fields.
5. **Drop-level scope source of truth:** `main/PLAN.md` § 19.3 (Template Configuration drop) — every droplet must trace to a § 19.3 bullet.

---

## 2. Architectural Decisions (LOCKED — Dev Approved 2026-05-02)

These decisions resolve cross-cutting open questions surfaced by Round 1 plan-QA. Bake them into the unified plan; do NOT re-route them.

- **2.1** **C1 — `UpdateActionItem` field-level write guard.** When `existing.Owner == "STEWARD"` and caller's `AuthPrincipalType != "steward"`, reject any `UpdateActionItemRequest` whose `Owner` or `DropNumber` differ from existing values. Without this, an agent-principal session clears `Owner` then transitions state — silent gate bypass. Land in Droplet 3.C.3 (or split into a coupled sub-droplet 3.C.3a if needed for atomicity).

- **2.2** **`principal_type: steward` autent boundary-map at the `autentauth` adapter layer.** Vendored `autent@v0.1.1` (`/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/autent@v0.1.1/domain/principal.go:22-26`) declares closed enum `{user, agent, service}` and rejects unknown values. **Tillsyn keeps `steward`** as a tillsyn-internal axis on its own `auth_requests` table + `AuthSession.PrincipalType` + new `AuthenticatedCaller.AuthPrincipalType`. **At the autentauth adapter boundary** (`internal/adapters/auth/autentauth/service.go:191`), map `steward → autentdomain.PrincipalTypeAgent` for autent's purposes; on the way back, `principalTypeToActorType` (`:803-812`) keeps mapping to `ActorTypeAgent`. The `steward` axis lives ONLY in tillsyn's own auth-request + session storage and the `AuthPrincipalType` field on the caller. Document the mapping in a doc comment at the adapter callsite.

- **2.3** **STEWARD persistent parents seeded via the default template's `[child_rules]`** at project creation time. The 6 STEWARD persistent parents (`DISCUSSIONS`, `HYLLA_FINDINGS`, `LEDGER`, `WIKI_CHANGELOG`, `REFINEMENTS`, `HYLLA_REFINEMENTS`) become rule-spawned children of the project root, marked with `persistent = true` (see §3.1 below). This closes the cold-start lockout that Round 1 falsification raised (Unit C C5) and makes the auto-generator's `parent_id_lookup` reliable. Add a new droplet **3.C.0 (or extend 3.C.4)** covering the default-template entry that seeds the 6 parents.

- **2.4** **`internal/templates/builtin/default.toml`** — the embedded TOML lives under `internal/templates/builtin/`, with `embed.go` colocated in `internal/templates/`. Resolves Unit B CE1 (`//go:embed ../..` build-stopping). Replaces Unit B 3.B.7's "repo-root sibling" claim. The repo-root `templates/` directory is NOT recreated.

- **2.5** **`KindCatalog` import direction — lazy-decode `KindCatalogJSON json.RawMessage` on `Project`.** `internal/domain/project.go` carries the raw JSON; decoding is done in `internal/app` or `internal/templates`, NEVER on the `Project` struct's methods. This avoids the `domain → templates` import cycle that Round 1 falsification surfaced (Unit B CE4). Update Unit B 3.B.5's "two options, builder picks" to "lazy-decode JSON RawMessage; decode lives outside `internal/domain`."

- **2.6** **`GateRule` defers to Drop 4** (the dispatcher consumer). Drop 3 leaves a `[gate_rules]` table hook in TOML schema for forward-compatibility but does NOT define `GateRule`'s fields or attachment point. Update Unit B 3.B.1 to remove `GateRule` from the schema struct list; instead document the schema-level placeholder (TOML `[gate_rules]` table reserved, no Go struct yet). Drop the `GateRule` struct stub from 3.B.1's acceptance.

- **2.7** **PLAN.md § 19.3 bullet 9 (template-defined STEWARD-owned kinds) — covered NOW.** Mechanism: `KindRule.SteWardOwned bool` (or `KindRule.Owner string` field on `KindRule`, accepting `"STEWARD"` as a value) on Unit B's schema. The auto-generator reads this when materializing children: any kind with `steward_owned = true` (or `owner = "STEWARD"`) gets `Owner = "STEWARD"` set on creation regardless of who creates it. Land as a sibling of bullet 8's `[child_rules]` work — extend Unit B 3.B.1 + 3.B.7 to cover the schema field; extend Unit C 3.C.4 (or new 3.C.7) to consume it at action-item-create time.

- **2.8** **`ReparentActionItem` gate — YES.** Add the same `assertOwnerStateGate` semantic to the reparent path (`internal/app/service.go:1106-1156` + `internal/adapters/server/common/app_service_adapter_mcp.go:810-823`). When `existing.Owner == "STEWARD"` and caller's `AuthPrincipalType != "steward"`, reject. Add to Droplet 3.C.3 acceptance + integration test in 3.C.6. Methodology §6.3 ("Failed nodes remain in their original tree position; they are not moved into a separate failed lane") confirms reparenting is a state-affecting mutation.

---

## 3. Methodology Integration (THREE NEW FIRST-CLASS FIELDS + ONE NEW ATTACK VECTOR)

`ta-docs/cascade-methodology.md` is now THE architectural spec for cascade nodes. The methodology's §11 canonical NodeBase + ActionItem field list controls. Drop 3 must conform to feature-parity for Tillsyn-as-substrate. The following additions are new scope on top of the Round 1 plans.

- **3.1** **`Persistent bool` first-class field on `ActionItem`.** Methodology §11.2. Replaces "STEWARD's persistent parents are special-cased in code" with "any node with `persistent = true` is retained as an anchor; child-rules and template seed it once at project creation." STEWARD's 6 persistent parents land with `Persistent = true`. Add to Unit A or Unit C — recommend **Unit C 3.C.1** (already lands `Owner` + `DropNumber` first-class; add `Persistent` alongside) since Unit C owns the STEWARD-related shape work.

- **3.2** **`DevGated bool` first-class field on `ActionItem`.** Methodology §11.2. Generalizes the STEWARD owner-gate to "any node where state transitions need human-reviewer approval." For Drop 3, used by the refinements-gate confluence (`DROP_N_REFINEMENTS_GATE_BEFORE_DROP_N+1`) to require dev sign-off before STEWARD can close it. Validation: `DevGated = true` items can be created/updated by agents but state transitions require an interactive human-approval signal (specific enforcement mechanism for Drop 4 dispatcher; Drop 3 lands the field + a passive auth-layer check that flags state transitions on `DevGated` items via `attention_item` rather than blocking, since the dispatcher isn't here yet). Add to **Unit C 3.C.1** alongside `Persistent`.

- **3.3** **`Irreducible bool` flag on `kind=build` droplets.** Methodology §2.3 + §11.3. Marks single-function-signature changes, single SQL migrations, single template edits — droplets that cannot decompose further. Plan-QA-falsification validates the claim (the methodology rule: "Planners default to decompose; irreducibility is the exception, not an escape hatch"). For Drop 3, land as a domain field on `ActionItem` (dispatcher consumes; for now plan-QA-falsification's new attack-vector list includes "irreducible-claim attack" — flag any droplet with `irreducible = true` whose acceptance criteria suggest it could decompose). Add to **Unit A** as a new droplet `3.A.8` or extend `3.A.2` (since `3.A.2` already adds `StructuralType` first-class to `ActionItem`).

- **3.4** **6th plan-QA-falsification attack vector — §4.4 global L1 plan-QA sweep.** Methodology §4.4 specifies a *second* plan-QA pass at L1 with full tree visibility when depth ≥ 3. It checks: (i) blocker graph acyclic; (ii) no two sibling droplets share `paths` or `packages` without explicit `blockers`; (iii) acceptance criteria at leaves compose into L1 outcome; (iv) no orphan droplets. **This is in addition** to the 5 attack vectors PLAN.md § 19.3 line 1644-1649 already enumerated. Extend Unit A 3.A.7's go-qa-falsification-agent prompt edits to include this 6th block: *"§4.4 global L1 plan-QA sweep — when invoked at L1 with depth ≥ 3 view, run blocker-acyclic, sibling-overlap-without-blockers, leaf-composition, and orphan checks across the full tree."*

- **3.5** **Re-frame Unit C scope language.** Replace "STEWARD persistent parents" mentions with "nodes with `Persistent = true` (the 6 STEWARD-owned anchor nodes seeded by the default template)." The mechanism is the field; STEWARD is just one consumer. Same with `dev_gated` — frame as a domain primitive, not a STEWARD-specific concept.

**Defer to Drop 4 / refinements (DO NOT add to Drop 3):** `failure` concrete type with `failure_kind` / `diagnostic` / `fix_directive`, `attempt_count` / `blocked_retry_count` / `last_failure_context` retry tracking, `start_commit` / `end_commit` git anchors, `context_blocks` array, per-kind droplet ceilings (`droplet_max_loc`, `droplet_max_files`), project onboarding fields (`mission`, `vocabulary`, `language`, `build_tool`, `standards_markdown_id`).

---

## 4. Mechanical Synthesis-Time Fixes (NO DEV DECISION NEEDED)

Apply each at unification:

- **4.1** **Drop fictional INSERT/UPDATE line cites in 3.A.3.** Per Unit A QA Proof §1.1: `repo.go:1414, :1452, :2500` are SELECT projections, NOT additional INSERTs / soft-create-or-update paths. There is exactly ONE INSERT (`:1253`) and ONE UPDATE (`:1347`) with column-bearing writes. Rewrite 3.A.3's bullet to: *"Sweep SELECT column-lists at `ListActionItems` (`:1414`), `ListActionItemsByParent` (`:1452`), and `getActionItemByID` (`:2500`) — each must include `structural_type` in the same ordinal position so `scanActionItem` reads the value correctly."*

- **4.2** **Add `3.A.5 blocked_by 3.A.4`** (or document the same-package non-overlap exception). Per Unit A QA Proof §1.6 — both droplets share `internal/app` package compile unit. Recommend the explicit blocker for strict same-package serialization.

- **4.3** **Change 3.C.5 `Blocked by: 3.C.2` → `Blocked by: 3.C.3`.** Per Unit C QA Proof §3.3 — both droplets edit `internal/adapters/server/common/app_service_adapter_mcp.go`; the prose acknowledges 3.C.5 must follow 3.C.3 but the formal `Blocked by` field reads only `3.C.2`. Tighten to `3.C.3` (which transitively pulls in `3.C.2`).

- **4.4** **Renumber Unit D droplets `5.D.N → 3.D.N`** throughout `UNIT_D_PLAN.md` and update the Cross-Unit Dependency Map. Per Unit D QA Proof §2.1 — purely clerical.

- **4.5** **Add `workflow/drop_1_5/**` and `workflow/drop_1_75/**`** to Unit D 3.D.5's "Paths (out of scope — explicitly excluded)" list. Per Unit D QA Proof F1 + Falsification §2.2 — both directories exist on disk and are historical audit trail. Update 3.D.6's exclusion list identically.

- **4.6** **Wire `3.D.1 blocked_by 3.A.7`** AND **`3.D.5 blocked_by 3.D.1`** for the three-way write conflict on `~/.claude/agents/go-qa-falsification-agent.md`. Per Unit A Falsification C3 + Unit D Falsification §2.1 — file is touched by 3.A.7 (5 attack-vector block), 3.D.1 (frontmatter pointer), 3.D.5 (legacy-vocab full sweep). Order: 3.A.7 → 3.D.1 → 3.D.5.

- **4.7** **Unit B 3.B.8 — add "starting-point lower bound" disclaimer + LSP-found additional call sites.** Per Unit B QA Proof §1.1 + Falsification CE2: builders must run `LSP findReferences` exhaustively on `KindTemplate`, `KindTemplateChildSpec`, `AllowedParentScopes`, `AllowsParentScope` before editing. Acceptance criterion: "`LSP findReferences` returns 0 hits" already covers this — but the path list should explicitly read as "a starting-point lower bound, not a closed enumeration." Add the LSP-found additional sites the falsification reviewer enumerated:
  - `internal/adapters/server/mcpapi/instructions_explainer.go:241-242, 391, 403, 528, 550`
  - `internal/adapters/server/common/app_service_adapter_mcp.go:1188, 1200, 1202, 1214`
  - `internal/adapters/server/mcpapi/extended_tools_test.go:703, 706, 712, 720, 723`
  - `internal/adapters/storage/sqlite/repo.go:1061, 1066, 1070, 1095, 1100, 1104, 1130, 1140, 1156, 2940, 2942, 2964, 2966, 2970, 2976, 2981, 2982, 2987, 2988`
  - `internal/app/snapshot.go:727, 1092, 1098, 1100, 1339, 1340, 1345, 1347` (beyond the cited `:94`)
  - `internal/tui/model.go:35, 754, 932, 1341` + `internal/tui/model_test.go:39, 87, 100, 104, 14661`
  - `internal/app/kind_capability.go:751-799` (recursive expansion, beyond the cited `:566, 750-766, 771`)
  - `cmd/till/main.go:3617, 3619` (beyond the cited `:3042-3442`)
  - `internal/domain/kind_capability_test.go:18-73` (entire test body, not just `:18, 20, 49`)

- **4.8** **Unit C 3.C.3 — add post-Drop-1 supersede path.** Per Unit C Falsification C8: Drop 1 (closed) landed always-on `failed` state + the supersede path (`till action_item supersede`). The STEWARD gate must fire on supersede identically to `MoveActionItemState`. Add to 3.C.3 acceptance: *"Test that `supersede` on a STEWARD-owned item is gated identically to MoveActionItemState — agent-principal session calling supersede on `Owner = STEWARD` rejects with `ErrAuthorizationDenied`; steward-principal session succeeds."*

---

## 5. Per-Unit Findings to Address

### 5.A — Unit A (Cascade Vocabulary Foundation)

**From PROOF (verdict: FAIL, recoverable):**
- 5.A.1 [BLOCKING] §1.1 — fictional INSERT/UPDATE cites in 3.A.3. Apply 4.1.
- 5.A.2 [BLOCKING-OR-DOC] §1.6 — same-package serialization gap 3.A.4 ↔ 3.A.5. Apply 4.2.
- 5.A.3 [NIT] §1.2 — 3.A.1 regex-comment mirror caveat. Add one-line: "the new test's character-class comment text reflects `[a-z-]+` — do NOT copy `role_test.go:120`'s comment verbatim if it conflicts with the actual regex."
- 5.A.4 [NIT] §1.3 — 3.A.2 sweep-scope enumeration. Tighten acceptance: "Cross-package sweep — every `_test.go` under `internal/` that calls `domain.NewActionItem` MUST supply a valid `StructuralType` value. Files affected: `internal/domain/domain_test.go`, `internal/app/service_test.go`, `internal/adapters/storage/sqlite/repo_test.go`, `internal/app/snapshot_test.go`, `internal/app/attention_capture_test.go`, `internal/app/dotted_address_test.go`, `internal/app/embedding_runtime_test.go`, `internal/adapters/storage/sqlite/embedding_jobs_test.go`, `cmd/till/embeddings_cli_test.go`, `internal/tui/model_test.go`."
- 5.A.5 [NIT] §1.4 — 3.A.6 WIKI placement disambiguation. Tighten to "insert as new `## Cascade Vocabulary` h2 between line 34 (end of `### Do Not Use Templates Right Now`) and line 36 (start of `## Level Addressing (0-Indexed)`) — sibling top-level h2, NOT nested."
- 5.A.6 [NIT] §1.5 — migration block line cite `lines 515/548` → `line 518 (start of workItemAlterStatements)`.

**From FALSIFICATION (verdict: FAIL, 5 CE + 2 nits):**
- 5.A.7 [HIGH — most damaging] CE1 — 3.A.2 required-on-create breaks ~96 NewActionItem call sites across 6 packages; 3.A.2's gate is `mage test-pkg ./internal/domain` only. **Fix:** upgrade 3.A.2 acceptance to `mage ci` AND apply Option (b) from CE1: introduce `newActionItemForTest` helper mirroring the Drop 1.75 `Kind` precedent at `internal/tui/model_test.go:14674-14687` (defaults `StructuralType = StructuralTypeDroplet` if empty); test fixtures call the helper instead of `domain.NewActionItem` directly. Document the precedent in 3.A.2 acceptance.
- 5.A.8 [HIGH] CE3 — three-way write conflict on `~/.claude/agents/go-qa-falsification-agent.md`. Apply 4.6.
- 5.A.9 [HIGH] CE5 — 3.A.4 `stubExpandedService` rejection logic breaks existing fixtures. **Fix:** in the test stub, default `args.StructuralType` to `StructuralTypeDroplet` when empty (the production code rejects empty; the stub may be more permissive for backward-fixture compat — matches the Role rejection pattern that explicitly permits empty in the stub).
- 5.A.10 [NIT] CE4 — `[a-z-]+` regex narrowing rationale incoherent. Tighten regex to `[a-z]+` and drop the hyphen-only test case (none of the 4 enum values contain hyphens), OR keep `[a-z-]+` but rewrite the rationale comment.
- 5.A.11 [NIT] C6 — "Confirmed" item inside "Unresolved Questions" section. Move to Architectural Decisions block.
- 5.A.12 [NIT — process] C7 — Hylla-first evidence sourcing process gap. Future planner spawns use Hylla MCP first per CLAUDE.md §"Code Understanding Rules"; not a plan correctness issue.

### 5.B — Unit B (Template System Overhaul)

**From PROOF (verdict: PASS WITH MITIGATION-REQUIRED, 2 mitigation + 3 nits):**
- 5.B.1 [MITIGATION] §1.1 — 3.B.8 path enumeration is a starting-point, not a closed set. Apply 4.7.
- 5.B.2 [MITIGATION] §1.2 — 3.B.5 import-direction decision routed to planner. Apply 2.5.
- 5.B.3 [NIT] §1.3 — `internal/templates/` package rationale. Tighten to "hexagonal core sub-package, distinct from `internal/app/` services that consume it" (NOT "adapter-adjacent" which is misleading).
- 5.B.4 [LOW] §2.1 — cycle-detection algorithm choice. Builder picks visited-set DFS; not blocking.
- 5.B.5 [LOW] §2.2 — `Tools []string` validation deferred to Drop 4. Forward-routed correctly; flag for orch to seed in Drop 4 planning.

**From FALSIFICATION (verdict: FAIL, 8 CE + 4 nits):**
- 5.B.6 [HIGH — most damaging] CE1 — `//go:embed ../../templates/builtin/default.toml` from `internal/templates/embed.go` is build-stopping (Go embed rejects `..`). Apply 2.4 — move TOML under `internal/templates/builtin/default.toml`, embed.go colocated in `internal/templates/`.
- 5.B.7 [HIGH] CE2 — 3.B.8 missing call sites. Apply 4.7.
- 5.B.8 [HIGH] CE3 — 3.B.8 deletes `kind_catalog` boot-seed but `repo_test.go:2470-2517` `TestRepositoryFreshOpen…` and `:2520-2568` `TestRepositoryFreshOpenKindCatalogUniversalParentAllow` pin the seeded values. **Fix:** 3.B.8 acceptance must explicitly delete or rewrite both test functions. Equivalent assertions move to `internal/templates/embed_test.go` (asserting `default.toml` covers all 12 kinds) — add to 3.B.7 acceptance.
- 5.B.9 [MEDIUM] CE4 — `KindCatalog` import-direction. Apply 2.5 (lazy-decode `KindCatalogJSON json.RawMessage` on `Project`; decoder lives outside `internal/domain`).
- 5.B.10 [MEDIUM] CE5 — `Load`'s strict-decode + schema-version-validate ordering. **Fix:** spec the order in 3.B.2 acceptance: tolerant pre-pass that decodes only `schema_version` (separate `Decoder` instance without `DisallowUnknownFields`), reject if `schema_version` is unknown, only then strict-decode the rest. Documents the version-aware error UX.
- 5.B.11 [MEDIUM] CE6 — `GateRule` undefined. Apply 2.6 (defer to Drop 4; remove from 3.B.1 acceptance; leave TOML `[gate_rules]` table reserved with no Go struct yet).
- 5.B.12 [MEDIUM] CE7 — 3.B.3's 144-row matrix vs 3.B.7's `default.toml` drift risk. **Fix:** 3.B.3 uses a hand-coded `Template` value as test fixture (NOT loaded from `default.toml`). 3.B.7's `embed_test.go` independently asserts the loaded `default.toml` round-trips against the same hand-coded fixture. Two distinct assertion paths against one source of truth.
- 5.B.13 [HIGH] CE8 — MCP/CLI wire surfaces (`till.kind operation=upsert`, `till.upsert_kind_definition` legacy alias, `till kind` CLI) carry `Template` + `AllowedParentScopes`. **Fix:** 3.B.8 must classify each: pre-MVP rule (`feedback_no_migration_logic_pre_mvp.md`) says no migration; recommend **delete the wire tool and CLI subcommand entirely** — the kind catalog is now read-only at runtime, mutated only via TOML at project-create. Document deprecation in 3.B.8 acceptance: remove `till.kind operation=upsert`, remove `till.upsert_kind_definition` legacy alias, remove `till kind` mutating subcommands (read-only `till kind list/get` may stay if Drop 4 needs them; otherwise also delete).
- 5.B.14 [LOW] N1 — `KindCatalog` runtime-mutability semantics implicit. Add explicit acceptance to 3.B.5: "edits to `<project_root>/.tillsyn/template.toml` after project create are ignored until dev fresh-DBs."
- 5.B.15 [LOW] N2 — rejection-comment authorship for dispatcher-driven auto-create. **Fix:** 3.B.9 acceptance scope-narrows: rejection-comments fire only on auth-gated creates (human/agent driven); dispatcher-internal auto-create rejections route differently (Drop 4 dispatcher specs: `failed` state on parent, no comment).
- 5.B.16 [LOW] N3 — `default.toml` "implicit-by-absence vs explicit deny rows" choice. **Fix:** force explicit `[child_rules]` deny rows in 3.B.7. Adding a 13th kind in a future drop is then an explicit opt-in to existing rules, not an implicit allow.
- 5.B.17 [LOW] N4 — 3.B.6 "convergence note" admits 3.B.6 might be no-op. **Fix:** commit explicitly: 3.B.1 ships skeletal `AgentBinding` (top-level fields declared but no field validation); 3.B.6 fills it in (validation + round-trip test). Builder doesn't read 3.B.1's output to decide.

### 5.C — Unit C (STEWARD Auth + Auto-Gen)

**From PROOF (verdict: PASS-with-nits, 5 nits):**
- 5.C.1 [NIT] §3.1 — `MoveActionItem` (column-only) gate fetch point. Tighten 3.C.3 to: "for `MoveActionItem`, builder adds a `GetActionItem` call before the gate fires (do NOT skip the gate just because the column-move path doesn't pre-fetch today)."
- 5.C.2 [NIT] §3.2 — `<final-rule-engine-droplet>` placeholder resolved at synthesis to `3.B.4` (`Template.ChildRulesFor`). Substitute throughout 3.C.4.
- 5.C.3 [NIT] §3.3 — 3.C.5 `Blocked by` should include 3.C.3 explicitly. Apply 4.3.
- 5.C.4 [NIT] §3.4 — PLAN.md § 19.3 bullet 9 cross-ref. Apply 2.7 (cover bullet 9 NOW via `KindRule.SteWardOwned bool` schema field + auto-generator consumer).
- 5.C.5 [NIT] §3.5 — index coverage. Tighten 3.C.2 acceptance: "the auto-generator's two cross-row queries are fully index-covered." Either swap index column order to `(project_id, drop_number, owner)` OR add a second index `(project_id, drop_number)`. Builder picks based on perf measurement (no preference at planning time — small expected row counts).

**From FALSIFICATION (verdict: FAIL, 8 CE + 5 nits):**
- 5.C.6 [HIGH — most damaging] C1 — `UpdateActionItem` state-lock bypass via `Owner` clearing. Apply 2.1.
- 5.C.7 [HIGH] C2 — `ReparentActionItem` unguarded. Apply 2.8.
- 5.C.8 [HIGH] C3 — `principal_type: steward` autent collision. Apply 2.2.
- 5.C.9 [MEDIUM] C4 — `MoveActionItem` no pre-fetch. Apply 5.C.1 above.
- 5.C.10 [HIGH] C5 — STEWARD persistent parent seeding. Apply 2.3 (default-template `[child_rules]` seeds the 6 parents at project creation).
- 5.C.11 [MEDIUM] C6 — refinements-gate `blocked_by` dynamic-at-create + manual-update has known forgetfulness failure mode. **Fix:** add 3.C.6 integration test case asserting the failure mode is documented behavior — drop_orch creates mid-drop child after gate, manual-update miss, gate closes prematurely. Either accept with runtime warning OR upgrade to rule-fire-on-every-new-drop-N-child (recommend ACCEPT with warning + add `attention_item` when gate's blocked_by closes if any drop_number=N item is still in_progress).
- 5.C.12 [MEDIUM] C7 — PLAN.md § 19.3 bullet 9 missing. Apply 2.7 (cover via `KindRule.SteWardOwned bool` + consumer in 3.C.4).
- 5.C.13 [LOW] C8 — post-Drop-1 supersede path. Apply 4.8.
- 5.C.14 [MEDIUM] N1 — `MoveActionItem` (column-only) state-neutral move semantics. **Fix:** lock semantics in 3.C.3 acceptance: gate on `Owner == "STEWARD"` regardless of state delta — drop-orchs cannot reorder STEWARD items. Stricter than § 19.3 bullet 7's literal "cannot move STEWARD items through state" but simpler/safer; matches "STEWARD owns state" intent. Document the choice.
- 5.C.15 [LOW] N3 — `AuthPrincipalType` field name confusable with existing `PrincipalType ActorType`. **Recommend:** rename new field to `AuthRequestPrincipalType` for clarity. Update 3.C.3 acceptance + tests.
- 5.C.16 [LOW] N4 — rollback-cost claim understated. Update 3.C.1 note: "If dev pushes back to metadata JSON, every consumer except the auth gate changes shape — non-trivial backout (3.C.1 + 3.C.2 + 3.C.4 + 3.C.5 + tests)."
- 5.C.17 [LOW] N5 — index design mismatch. Apply 5.C.5 above.

### 5.D — Unit D (Adopter Bootstrap + Doc Sweep)

**From PROOF (verdict: PASS-with-nits, 5 nits):**
- 5.D.1 [NIT] §3.2 F1 — exclusion list missing `drop_1_5/`, `drop_1_75/`. Apply 4.5.
- 5.D.2 [NIT] §3.2 F2 — 5.D.5 + 5.D.6 split timing. Defensible split kept (5.D.5 pre-A/B/C legacy sweep; 5.D.6 post-A/B/C new-vocab sweep). Document the split rationale explicitly in the unified plan.
- 5.D.3 [NIT] §3.2 F3 — file-level race risk on `go-qa-falsification-agent.md`. Apply 4.6.
- 5.D.4 [NIT] §3.2 F4 — "No subagents for MD work" memory-rule tension. **Resolution:** the rule applies to in-orch quick MD updates; for systematic doc sweeps in Drop 3 Unit D, builder subagent IS the work-doer. The self-QA + dev-approval gate before commit still applies. Document this in 3.D.5 + 3.D.6 acceptance.
- 5.D.5 [NIT] §3.2 F5 — 4 open architectural questions correctly parked, all locked at planner time:
  - Skill frontmatter convention: insert pointer in `references/template.md`, NOT YAML (description field consumed by autoloader for relevance scoring; pollution degrades skill discovery — 5.D.6 N2.8 confirms).
  - Per-drop final wrap-up timing: option (a) — sweep ships inside Drop 3 PR (preserves PR-review density; option (b) bypasses PR review).
  - Boilerplate `## Cascade Vocabulary` content owner: option (iii) — embed inline in `references/template.md` (bootstrap skill self-contained; no Unit B coordination needed).
  - Agent frontmatter insertion location: prose body immediately after YAML close (option (b) in 5.D.1's notes — YAML insertion would require schema-validated keys).

**From FALSIFICATION (verdict: FAIL, 7 CE + 2 lower-severity):**
- 5.D.6 [HIGH — most damaging] §2.1 — three-way write conflict on `go-qa-falsification-agent.md`. Apply 4.6.
- 5.D.7 [HIGH] §2.2 — exclusion list missing `drop_1_5/`, `drop_1_75/`. Apply 4.5.
- 5.D.8 [MEDIUM] §2.3 — `~/.claude/CLAUDE.md` retired-vocab edits. **Fix:** promote from "review only" to first-class in-scope edit target with three known retired-vocab hits enumerated:
  - Line 9 `slice-by-slice` → drop or rephrase per cascade vocabulary.
  - Lines 10, 121, 147 `build-task` → `build`.
  - Recommend orch confirms with dev pre-Phase-1 since `~/.claude/CLAUDE.md` is dev-personal global rules (LOCK: dev approved; treat as in-scope per memory rule "Tillsyn — No Slice Anywhere" applies globally to active docs).
- 5.D.9 [MEDIUM] §2.4 — bootstrap skills don't currently own WIKI.md authoring. **Fix:** narrow 3.D.2 + 3.D.3 acceptance to "CLAUDE.md pointer line only; defer WIKI scaffolding ownership to a follow-up refinement." This avoids scope expansion of bootstrap skills mid-Drop-3. Document the deferral in the plan's "Out Of Scope" block.
- 5.D.10 [MEDIUM] §2.5 — worklog vs commit boundary on `~/.claude/` edits. **Fix:** add explicit accept in `UNIT_D_PLAN.md` Notes section — "Audit gap accepted: `~/.claude/` edits are recorded in `BUILDER_WORKLOG.md` only; the workflow trades off git-tracked permanence for adopter-skill universality. Future maintainers reading the worklog see what edits landed; reconstruction against future filesystem state requires manual diff." (Option (c) from QA's three options.)
- 5.D.11 [LOW] §2.6 — 3.D.4 `workflow/example/CLAUDE.md` insertion site ambiguous. **Fix:** pick one — insert as new bullet inside `## Coordination Model — At a Glance` immediately after the line-26 reading-order bullet. Update 3.D.4 acceptance prose unambiguously.
- 5.D.12 [LOW] §2.7 — wrap-up timing rationale implicit. Add to plan notes: "Option (a) preserves PR-review density; option (b) bypasses PR review and accumulates documentation drift on `main`. Lock (a) for review-density preservation."
- 5.D.13 [LOW] §2.8 — skill frontmatter rationale thin. Add to plan notes: "description field is consumed by autoloader for relevance scoring; pollution with cross-references degrades skill discovery."
- 5.D.14 [LOW] §2.9 — slash-command files (`~/.claude/commands/*.md`) missing from sweep. **Fix:** add `~/.claude/commands/*.md` to 3.D.5's path list (with NOT-git-tracked / worklog-recording carve-out). If no slash-command files exist, document the absence.

---

## 6. Output Requirements

The unified `## Planner` section in `workflow/drop_3/PLAN.md` must include:

- **6.1** **Unified droplet decomposition: ~28 droplets renumbered `3.1` – `3.N`**, with original unit-letter retained inline as `3.A.k`/`3.B.k`/`3.C.k`/`3.D.k` for traceability. Each droplet has: state (`todo`), paths (concrete file list), packages (Go packages touched), acceptance (testable criteria), `blocked_by` (intra-unit + cross-unit explicit), open questions (if any remain after applying §2 + §3 decisions — should be zero or near-zero).

- **6.2** **Cross-unit `blocked_by` wiring** explicit and acyclic. Recommended hard edges:
  - 3.B.1 `blocked_by` 3.A.1 (StructuralType enum → schema struct field).
  - 3.B.4 `blocked_by` 3.A.1 (StructuralType axis → child_rules binding).
  - 3.B.7 `blocked_by` 3.A.1, 3.B.6 (default.toml needs full schema + StructuralType).
  - 3.C.4 `blocked_by` 3.B.4 (rule-engine consumer → rule-engine landed).
  - 3.D.1 `blocked_by` 3.A.7 (frontmatter pointer → after Unit A's attack-vector edits to same file).
  - 3.D.5 `blocked_by` 3.D.1 (legacy-vocab sweep → after frontmatter pointer added).
  - 3.D.6 `blocked_by` "Units A + B + C all closed" (the post-everything wrap-up).

- **6.3** **Renumbering mapping table** — explicit `3.A.k → 3.<new>` mapping so audit trail is preserved. Recommended scheme:
  - 3.1 = 3.A.1 (StructuralType domain enum)
  - 3.2 = 3.A.2 (StructuralType + Persistent + DevGated + Irreducible fields on ActionItem)
  - 3.3 = 3.A.3 (SQLite columns + scanner)
  - 3.4 = 3.A.4 (App + MCP plumbing)
  - 3.5 = 3.A.5 (Snapshot serialization)
  - 3.6 = 3.A.6 (WIKI § Cascade Vocabulary)
  - 3.7 = 3.A.7 (plan-QA-falsification 5+1 attack vectors; +6th = §4.4 sweep per 3.4)
  - 3.8 = 3.B.1 (TOML schema structs, GateRule excluded per 2.6)
  - 3.9 = 3.B.2 (TOML parser + load-time validator with schema-version pre-pass per 5.B.10)
  - 3.10 = 3.B.3 (`Template.AllowsNesting`)
  - 3.11 = 3.B.4 (`[child_rules]` consumer)
  - 3.12 = 3.B.5 (`KindCatalog` via lazy-decode JSON RawMessage on Project per 2.5)
  - 3.13 = 3.B.6 (Agent binding fields fill-in per 5.B.17)
  - 3.14 = 3.B.7 (`internal/templates/builtin/default.toml` per 2.4 + STEWARD parent seeds per 2.3)
  - 3.15 = 3.B.8 (Rewrite/delete old API + delete `till.kind operation=upsert` MCP wire per 5.B.13)
  - 3.16 = 3.B.9 (Audit-trail comment + attention)
  - 3.17 = 3.C.1 (Owner + DropNumber + Persistent + DevGated first-class fields per 3.1, 3.2)
  - 3.18 = 3.C.2 (SQLite columns + index)
  - 3.19 = 3.C.3 (`principal_type: steward` + auth gate on Move + Update field-guard per 2.1 + Reparent gate per 2.8 + supersede per 4.8 + autent boundary-map per 2.2)
  - 3.20 = 3.C.4 (template auto-gen consumer + `KindRule.SteWardOwned` consumer per 2.7)
  - 3.21 = 3.C.5 (MCP/snapshot plumbing for Owner + DropNumber + Persistent + DevGated)
  - 3.22 = 3.C.6 (integration tests + refinements-gate forgetfulness regression test per 5.C.11)
  - 3.23 = 3.D.1 (Agent file frontmatter sweep)
  - 3.24 = 3.D.2 (`go-project-bootstrap` skill update — narrow per 5.D.9)
  - 3.25 = 3.D.3 (`fe-project-bootstrap` skill update — narrow per 5.D.9)
  - 3.26 = 3.D.4 (CLAUDE.md template pointers)
  - 3.27 = 3.D.5 (in-repo legacy-vocab sweep)
  - 3.28 = 3.D.6 (post-A/B/C final wrap-up)

  Total: **28 droplets**. Open to a different mapping if integration reasoning argues for it; preserve the original per-unit boundaries for traceability.

- **6.4** **Locked architectural decisions block** at the top of the `## Planner` section, summarizing §2 (8 dev-approved decisions) so builders see them once, not threaded through 28 droplets.

- **6.5** **Methodology integration block** summarizing §3 (3 new fields + 6th attack vector + reframes). Cite `ta-docs/cascade-methodology.md` §11 as the canonical spec.

- **6.6** **Mechanical fix log** at end of `## Planner` section noting §4 (8 fixes) applied during synthesis.

---

## 7. Output Format

You MUST render your response beginning with a `# Section 0 — SEMI-FORMAL REASONING` block containing `## Planner`, `## Builder`, `## QA Proof`, `## QA Falsification`, and `## Convergence` passes before your final output. Each pass uses the 5-field certificate (Premises / Evidence / Trace or cases / Conclusion / Unknowns) where applicable. Convergence must declare (a) QA Falsification found no unmitigated counterexample, (b) QA Proof confirmed evidence completeness, (c) remaining Unknowns are routed. If any fail, loop back before Convergence.

After Section 0, render a `tillsyn-flow` numbered body with `## 1.`, `## 2.`, etc. and a `## TL;DR` with one `TN` item per top-level numbered section.

After your response is rendered, EDIT `workflow/drop_3/PLAN.md` to fill the `## Planner` section with the unified decomposition. Do NOT replace the `## Scope` or `## Notes` sections. Use the `Edit` tool with the existing placeholder text as `old_string` and your unified content as `new_string`.

---

## 8. Hard Constraints

- **NEVER** use `mage install` — dev-only target.
- **NEVER** use raw `go test` / `go build` / `go vet` / `go run` — always `mage <target>`.
- **NEVER** add `tc := tc` (Go 1.22+ loop-var capture).
- **NEVER** `git rm` any file in `workflow/drop_3/`. Files that don't apply are never created, not stamped-then-deleted.
- **NEVER** write `CLOSEOUT.md` / `LEDGER.md` / `WIKI_CHANGELOG.md` / `REFINEMENTS.md` / `HYLLA_FEEDBACK.md` / `HYLLA_REFINEMENTS.md` rollups — pre-MVP rule (`feedback_no_closeout_md_pre_dogfood.md`).
- **NEVER** add migration logic in Go, `till migrate` subcommands, or one-shot SQL scripts — pre-MVP rule (`feedback_no_migration_logic_pre_mvp.md`). Dev fresh-DBs `~/.tillsyn/tillsyn.db` between schema-touching droplets.
- **NEVER** include the `## Hylla Feedback` section content in any Tillsyn `description` / `metadata.*` / `completion_notes` / closing-comment surface — that section lives ONLY in your closing response (per memory `feedback_no_closeout_md_pre_dogfood.md` + Section 0 directive).
- Builders run **opus** per `feedback_opus_builders_pre_mvp.md`.
- Use Hylla MCP first for Go committed-code understanding (`mcp__hylla__hylla_*`) per CLAUDE.md §"Code Understanding Rules". Fall back to LSP / Read / Grep only after exhausting Hylla search modes; record any Hylla miss in your closing `## Hylla Feedback` section.
