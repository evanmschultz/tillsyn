# DROP 3 — UNIT B PLAN — TEMPLATE SYSTEM OVERHAUL

**Unit scope:** closed TOML schema + parser + validator + `Template.AllowsNesting` + `[child_rules]` + `templates/builtin/default.toml` + agent binding fields on kind definitions. Largest unit; biggest new code.
**Sibling units (parallel planners):** Unit A (cascade vocabulary / `structural_type` enum + WIKI glossary + plan-QA-falsification attack vectors), Unit C (STEWARD auth + `metadata.owner` + STEWARD-scope auto-generation), Unit D (adopter bootstrap + cascade-vocabulary doc sweep).
**State:** planning
**Author:** `go-planning-agent` (Phase 1, parallel planner B)
**PLAN.md ref:** `main/PLAN.md` § 19.3 — Template Overhaul Scope paragraph + agent-binding-field bullets at lines 1653-1658.

## Cross-Unit Dependencies

- **Hard dependency on Unit A's `StructuralType` enum landing first.** Templates bind `child_rules`, gate rules, validation, and agent bindings on the `structural_type` axis (PLAN.md line 1635). Until Unit A's `domain.StructuralType` value type exists in `internal/domain/`, Unit B's TOML schema cannot reference it. Orchestrator must wire `blocked_by: <Unit A's structural_type-landing droplet>` on every Unit B droplet that touches structural_type binding (droplets 3.B.4, 3.B.7).
- **Soft dependency on Unit C** for STEWARD-owned kind support — Unit B's schema must support `metadata.owner` semantics on kind definitions (e.g. a TOML `owner = "STEWARD"` field on a kind row). Unit C ships the auth-side enforcement; Unit B ships the schema slot. No `blocked_by` needed — Unit B lands the slot, Unit C reads it.
- **Drop 4 (dispatcher) consumes** the agent-binding fields Unit B writes into the schema. Out of scope for this unit; documented for handoff.

## Pre-MVP Constraints (Locked, Per Spawn Brief)

- No migration logic in Go code. Dev fresh-DBs `~/.tillsyn/tillsyn.db` after schema-touching droplets land.
- No closeout MD artifacts (`CLOSEOUT.md` / `LEDGER.md` / `WIKI_CHANGELOG.md` / `REFINEMENTS.md` / `HYLLA_FEEDBACK.md` / `HYLLA_REFINEMENTS.md`).
- Builders run **opus**.
- Default template path: `templates/builtin/default.toml` — re-introduce the `templates/` package Drop 2.1 deleted, under the new TOML schema.
- Per-project template path: `<project_root>/.tillsyn/template.toml`.
- **NEVER** `git rm` any `workflow/drop_3/` file.
- **NEVER** add `tc := tc` (Go 1.22+ loop-var capture).
- **NEVER** raw `go test` / `go build` / `go vet` / `go run` — always `mage <target>`. **NEVER** `mage install`. Build verification = `mage ci`.

## Architectural Decisions (Surfaced To Orchestrator)

- **Schema version starting value.** Recommend `schema_version = "v1"` (string, not integer). Forward-compat: future schema bumps land as `"v2"` etc., load-time error rejects unknown versions.
- **`KindCatalog` package location.** Recommend new package `internal/templates/` (not `internal/app/templates/` — keeps it adapter-adjacent and avoids `internal/app/` mass coupling). Drop 2.1 deleted the old `templates/builtin/*.json` package; Drop 3 reintroduces `templates/` at the repo root for embedded TOML defaults plus `internal/templates/` for the Go-side parser/loader/`KindCatalog` value type.
- **Audit-trail comment integration with auth.** Every `Template.AllowsNesting → false` rejection at the `till.action_item(operation=create)` boundary writes a `till.comment` on the **rejected parent** + creates an `attention_item` for the dev. The comment uses the existing `actor_type=user`/`actor_type=agent` actor model — no new principal type needed in Unit B. **Unit C does NOT need to gate this** — rejection-comments are write-as-the-rejecting-actor. Flag for orch sync with Unit C planner so the audit trail wiring doesn't collide with Unit C's STEWARD principal-type work.

## Decomposition

Nine droplets. Each is atomic: one PR-shaped change, one mage target, one acceptance check. Droplets 3.B.1, 3.B.6 share `internal/templates/schema.go` (the schema struct file) → 3.B.6 explicitly `blocked_by` 3.B.1. Droplets 3.B.7, 3.B.5 share project-create wiring → 3.B.7 `blocked_by` 3.B.5. Droplet 3.B.8 (rewrite/delete old API) is the last code droplet because it depends on every consumer (3.B.3, 3.B.4, 3.B.5) using the new API.

### 3.B.1 — Define TOML schema as Go structs

- **Scope.** Pure type definitions for the closed TOML schema. No behavior, no parsing, no validation. Defines `Template`, `KindRule`, `ChildRule`, `AgentBinding`, `GateRule`, plus the `schema_version` constant and a top-level `Template.SchemaVersion` field.
- **Paths.**
  - `internal/templates/schema.go` (new)
  - `internal/templates/schema_test.go` (new — pure type assertions, no parser tests yet)
- **Packages.** `internal/templates` (new package).
- **Acceptance.**
  - `Template` struct has `SchemaVersion string`, `Kinds map[domain.Kind]KindRule`, `ChildRules []ChildRule`, `AgentBindings map[domain.Kind]AgentBinding` fields with TOML struct tags using the `github.com/pelletier/go-toml/v2` tag conventions.
  - `KindRule` struct has fields `Owner string` (e.g. `"STEWARD"` — Unit C reads), `AllowedParentKinds []domain.Kind`, `AllowedChildKinds []domain.Kind`, `StructuralType domain.StructuralType` (Unit A enum — placeholder if Unit A hasn't landed; otherwise typed).
  - `ChildRule` struct has fields `WhenParentKind domain.Kind`, `CreateChildKind domain.Kind`, `Title string`, `BlockedByParent bool`, `WhenParentStructuralType domain.StructuralType` (optional; bind on structural_type per PLAN.md line 1635).
  - `AgentBinding` struct has every field from PLAN.md § 19.3 bullets 1653-1656: `AgentName string`, `Model string`, `Effort string`, `Tools []string`, `MaxTries int`, `MaxBudgetUSD float64`, `MaxTurns int`, `AutoPush bool`, `CommitAgent string`, `BlockedRetries int`, `BlockedRetryCooldown time.Duration`.
  - `GateRule` struct stub for Drop 4 dispatcher consumption — schema-only, no behavior.
  - `SchemaVersionV1 = "v1"` constant exported.
  - Every top-level type and field has a Go doc comment.
  - All TOML tags use `toml:"snake_case_name"` form.
  - `mage test-pkg ./internal/templates` passes (covers compile + skeletal test file).
- **Mage target.** `mage test-pkg ./internal/templates`.
- **Blocked by.** Unit A's `domain.StructuralType` enum landing (cross-unit). Orch wires `blocked_by` at synthesis. If Unit A hasn't landed when 3.B.1 starts, planner stubs `StructuralType` as `string` and 3.B.1 carries a TODO referencing Unit A's plan-item ID; 3.B.1 then re-blocks behind Unit A's enum droplet for a follow-up tightening.
- **Status.** todo.

### 3.B.2 — TOML parser + load-time validator

- **Scope.** Pure load-and-validate function `Load(reader io.Reader) (Template, error)` with strict unknown-key rejection, schema-version assertion, and load-time validation (cycle detection in `child_rules`, unreachable-rule detection, unknown-kind detection). No file I/O, no `KindCatalog` baking yet — that's 3.B.5.
- **Paths.**
  - `internal/templates/load.go` (new)
  - `internal/templates/load_test.go` (new — table-driven; valid TOML, unknown key rejection, cycle, unreachable rule, unknown kind, missing schema_version, wrong schema_version, malformed TOML)
- **Packages.** `internal/templates`.
- **Acceptance.**
  - `Load(io.Reader) (Template, error)` parses TOML via `github.com/pelletier/go-toml/v2` decoder configured with `DisallowUnknownFields()` (strict mode).
  - Returns sentinel errors `ErrUnknownTemplateKey`, `ErrUnsupportedSchemaVersion`, `ErrTemplateCycle`, `ErrUnreachableChildRule`, `ErrUnknownKindReference` declared at package scope.
  - Load-time validator: builds parent → child kind graph from `[child_rules]`, runs DFS to detect cycles, asserts every referenced `Kind` is in `domain.validKinds` (the closed 12-value enum), asserts no `[child_rules]` entry references a `WhenParentKind` that no other kind can reach (orphan rule).
  - Test coverage ≥ 80% on `internal/templates/load.go`.
  - Table-driven tests follow project convention (no `tc := tc`).
  - `mage test-pkg ./internal/templates` passes.
  - `mage ci` passes (sanity).
- **Mage target.** `mage test-pkg ./internal/templates` for unit work; `mage ci` for the final validation cycle.
- **Blocked by.** 3.B.1 (schema structs must exist before parser unmarshals into them).
- **Status.** todo.

### 3.B.3 — `Template.AllowsNesting(parent, child Kind) (bool, reason string)`

- **Scope.** The single validation truth function the rest of the codebase calls. Pure method on `Template`. Reads `KindRule.AllowedParentKinds` / `AllowedChildKinds` and returns `(true, "")` on allowed, `(false, "<reason>")` on rejected with a stable English reason string.
- **Paths.**
  - `internal/templates/nesting.go` (new)
  - `internal/templates/nesting_test.go` (new — table-driven over the cartesian product of the 12-value Kind enum, asserting each pairing matches expected outcome from a reference matrix; also covers reverse-hierarchy prohibitions in droplet 3.B.7's default.toml)
- **Packages.** `internal/templates`.
- **Acceptance.**
  - Method signature exactly: `func (t Template) AllowsNesting(parent, child domain.Kind) (allowed bool, reason string)`.
  - Returns stable reason strings (not formatted with random data) so tests can assert exact text.
  - Reason format: `"kind %q cannot nest under parent kind %q (rule: %s)"`.
  - 144-row table-driven test (12×12 Kind cartesian product) covering every combo from the default.toml shipped in 3.B.7. Reference matrix encoded inline in the test file.
  - Empty-template fallback: when `Template.Kinds[parent]` is nil, behavior is universal-allow (matches Drop 2.8's empty-`AllowedParentScopes` semantics) — captured in a dedicated test row.
  - Test coverage ≥ 90% on `internal/templates/nesting.go`.
  - `mage test-pkg ./internal/templates` passes.
- **Mage target.** `mage test-pkg ./internal/templates`.
- **Blocked by.** 3.B.1 (needs schema), 3.B.2 (test fixtures load via parser).
- **Status.** todo.

### 3.B.4 — `[child_rules]` auto-create implementation

- **Scope.** The `Template` method that, given a parent kind + parent's `metadata.structural_type`, returns the list of child specs to auto-create. Drop 4's dispatcher will fire them; Unit B ships the spec-resolution logic and tests. No I/O, no DB calls — pure function over `Template`.
- **Paths.**
  - `internal/templates/child_rules.go` (new)
  - `internal/templates/child_rules_test.go` (new — table-driven; cases include build → auto-creates build-qa-proof + build-qa-falsification, plan → auto-creates plan-qa-proof + plan-qa-falsification, structural_type=drop → auto-creates planner droplet + qa-proof droplet + qa-falsification droplet per PLAN.md line 1635)
- **Packages.** `internal/templates`.
- **Acceptance.**
  - Method signature: `func (t Template) ChildRulesFor(parent domain.Kind, parentType domain.StructuralType) []ChildRuleResolution`.
  - `ChildRuleResolution` struct (new in this droplet) carries: `Kind domain.Kind`, `Title string`, `BlockedByParent bool`, `StructuralType domain.StructuralType`.
  - Result is deterministic — sorted stable order for test assertions.
  - Cycle protection: callers cannot recurse — `ChildRulesFor` returns one level only. Recursive expansion is the dispatcher's job (Drop 4).
  - Tests cover the four reverse-hierarchy + auto-create rules from PLAN.md § 19.3 Template Overhaul Scope: build auto-creates build-qa-*, plan auto-creates plan-qa-*, drop structural_type auto-creates planner+qa droplets per PLAN.md line 1635.
  - Test coverage ≥ 85% on `internal/templates/child_rules.go`.
  - `mage test-pkg ./internal/templates` passes.
- **Mage target.** `mage test-pkg ./internal/templates`.
- **Blocked by.** 3.B.1, 3.B.2. Cross-unit blocked by Unit A's `StructuralType` enum (orch wires).
- **Status.** todo.

### 3.B.5 — `KindCatalog` value type baked at project-creation

- **Scope.** Introduce `KindCatalog` value type in `internal/templates/`. Built once at project-creation time from `Template` + `domain.validKinds`. Stored on `Project.Metadata` as JSON snapshot (since pre-MVP rule = no migration; on schema change dev fresh-DBs). Wire `app.Service.CreateProjectWithMetadata` to bake `KindCatalog` from the per-project `template.toml` (or `templates/builtin/default.toml` fallback) and persist it on the project row. Replace runtime `repo.GetKindDefinition` lookups in `internal/app/kind_capability.go` with `Project.KindCatalog.Lookup(kindID)` calls.
- **Paths.**
  - `internal/templates/catalog.go` (new — `KindCatalog` value type, `Bake(t Template) KindCatalog`, `Lookup(kindID)`, JSON marshal/unmarshal for project metadata persistence)
  - `internal/templates/catalog_test.go` (new — round-trip JSON test, lookup test, hot-path benchmark micro-test)
  - `internal/app/service.go` — wire `CreateProjectWithMetadata` to call `templates.Bake` (lines around the existing project-create path; ~10-15 LOC change, no LSP-resolved exact line until builder reads it)
  - `internal/app/kind_capability.go` — replace `s.repo.GetKindDefinition(ctx, kindID)` calls in `resolveActionItemKindDefinition` (`:545-578`) with `project.KindCatalog.Lookup(kindID)`. Repository fallback retained for projects without baked catalog (zero-value catalog → fall back to current path; preserves boot compatibility per PLAN.md universal-nesting Drop 2 default).
  - `internal/domain/project.go` — add `KindCatalog templates.KindCatalog` field on `Project` struct + JSON marshaling. (NOTE: import direction — `domain` cannot import `templates` if `templates` imports `domain`. Resolution: keep `KindCatalog` defined in `internal/templates/`, but the `Project` struct field is `KindCatalogJSON json.RawMessage` and accessor methods on `Project` decode lazily. Or: invert import direction and have `templates` declared as a sibling of `domain`. Builder droplet decides — both options noted, builder picks the one with cleaner dependency graph and documents in `BUILDER_WORKLOG.md`.)
- **Packages.** `internal/templates`, `internal/app`, `internal/domain`.
- **Acceptance.**
  - `KindCatalog.Lookup(kindID domain.KindID) (KindRule, bool)` returns the rule + ok-flag.
  - `Bake(t Template) KindCatalog` is pure — no I/O, no clock.
  - Project-create path bakes the catalog from `<project_root>/.tillsyn/template.toml` if present, else from `templates/builtin/default.toml` (loaded via Go embed in 3.B.7).
  - Existing `internal/app/kind_capability.go` test suite still passes — `internal/app/kind_capability_test.go` is 31k LOC, this droplet must NOT break it. Adapter behavior is preserved when `KindCatalog` is zero-valued (boot-time empty catalog → fall back to legacy `repo.GetKindDefinition` path).
  - `mage test-pkg ./internal/templates`, `mage test-pkg ./internal/app`, `mage test-pkg ./internal/adapters/storage/sqlite` all pass.
  - `mage ci` passes.
- **Mage target.** `mage ci` (final validation; this droplet touches multiple packages).
- **Blocked by.** 3.B.1, 3.B.2, 3.B.3, 3.B.4 (all `internal/templates/` API must exist before app wiring).
- **Status.** todo.

### 3.B.6 — Agent binding fields on kind definitions (TOML side)

- **Scope.** Extend the schema struct from 3.B.1 to fully cover every agent-binding field PLAN.md § 19.3 bullets 1653-1656 enumerate: `agent_name`, `model`, `effort`, `tools`, `max_tries`, `max_budget_usd`, `max_turns`, `auto_push`, `commit_agent`, `blocked_retries`, `blocked_retry_cooldown`. Drop 4's dispatcher reads these — Unit B only writes the schema and lands tests asserting parse-round-trip fidelity. No consumer wiring.
- **Paths.**
  - `internal/templates/schema.go` (extend `AgentBinding` if 3.B.1 left fields incomplete; if 3.B.1 already covered them, this droplet is a tightening pass — see "Convergence note" below)
  - `internal/templates/schema_test.go` (extend table)
  - `internal/templates/agent_binding_test.go` (new — round-trip TOML test asserting every field survives marshal → unmarshal cycle)
- **Packages.** `internal/templates`.
- **Acceptance.**
  - `AgentBinding` struct carries every field from PLAN.md lines 1653-1656.
  - `BlockedRetryCooldown` is `time.Duration` parsed from TOML duration string (e.g. `"30s"`, `"5m"`).
  - `MaxBudgetUSD` is `float64` (cents-precision unnecessary at this stage).
  - `Tools` is `[]string` (not enum) — Drop 4 validates against the actual MCP/Claude tool catalog.
  - Round-trip test asserts marshal→unmarshal stability for a fully-populated `AgentBinding`.
  - Test coverage ≥ 80% on `internal/templates/schema.go`.
  - `mage test-pkg ./internal/templates` passes.
  - **Convergence note for orchestrator:** if droplet 3.B.1 is scoped tightly (skeletal `AgentBinding`), droplet 3.B.6 fills it in. If 3.B.1's spawn brief lets the builder land all fields up front, 3.B.6 collapses into a tightening + test pass. Builder reads 3.B.1's output and adjusts. **Result: 3.B.6 always touches `schema.go` after 3.B.1 — file-level conflict requires explicit `blocked_by`.**
- **Mage target.** `mage test-pkg ./internal/templates`.
- **Blocked by.** 3.B.1 (file-level: same `schema.go`).
- **Status.** todo.

### 3.B.7 — `templates/builtin/default.toml`

- **Scope.** Author the default template at `templates/builtin/default.toml` covering the conceptual prohibitions (PLAN.md § 19.3 Template Overhaul Scope): closeout-no-closeout-parent, commit-no-plan-child, human-verify-no-build-child, build-qa-*-no-plan-child. Plus auto-create rules: build → build-qa-proof + build-qa-falsification; plan → plan-qa-proof + plan-qa-falsification; drop structural_type → planner droplet + qa-proof droplet + qa-falsification droplet per PLAN.md line 1635. Plus agent bindings for every kind in the closed 12-value enum (model, agent_name, effort tier — see CLAUDE.md "Cascade Tree Structure" Agent Bindings table). Embed via Go `embed.FS` in `internal/templates/embed.go`.
- **Paths.**
  - `templates/builtin/default.toml` (new — repo root, NOT under `internal/`; lives as a sibling of `cmd/`, `internal/`, `magefile.go`)
  - `internal/templates/embed.go` (new — Go `//go:embed` directive bringing the TOML file into the binary)
  - `internal/templates/embed_test.go` (new — load default.toml via `Load` from 3.B.2, assert no errors, assert every closed-12-kind has a `KindRule`, assert the 4 reverse-hierarchy prohibitions from PLAN.md are encoded, assert the 3 auto-create rules from PLAN.md line 1635 are encoded, assert `AllowsNesting` rejects every reverse-hierarchy combo and accepts every legitimate combo)
- **Packages.** `internal/templates`.
- **Acceptance.**
  - `default.toml` parses cleanly via `Load` from 3.B.2.
  - Every closed-12-kind has a `[kinds.<kind>]` section.
  - Reverse-hierarchy prohibitions are explicit (not implicit by absence): the four bullets from PLAN.md § 19.3 Template Overhaul Scope are each declared as a `[child_rules]` row OR as `allowed_parent_kinds` exclusions (planner picks the cleaner shape — both work; doc the choice in builder worklog).
  - Auto-create rules: build → 2 children, plan → 2 children, structural_type=drop → 3 children. PLAN.md line 1635 is the source of truth.
  - Agent bindings populated for every kind from CLAUDE.md "Agent Bindings" table.
  - `mage test-pkg ./internal/templates` passes.
- **Mage target.** `mage test-pkg ./internal/templates`.
- **Blocked by.** 3.B.1, 3.B.2, 3.B.3, 3.B.6 (needs full schema + parser + nesting + agent-binding fields). Cross-unit blocked by Unit A's `StructuralType` enum (auto-create rule on drop type).
- **Status.** todo.

### 3.B.8 — Rewrite/replace existing `KindTemplate` + `AllowedParentScopes` mechanism, delete dead code

- **Scope.** Per PLAN.md § 19.3: "Existing `KindTemplate` + `KindTemplateChildSpec` + `AllowedParentScopes` + `AllowsParentScope` get **rewritten, not extended**." Concretely: every consumer of those types now reads through `KindCatalog` (3.B.5). This droplet deletes the dead types from `internal/domain/kind.go:94-352` (`KindTemplate`, `KindTemplateChildSpec`, `AllowedParentScopes` field, `AllowsParentScope` method, `normalizeKindTemplate`, `normalizeKindParentScopes`) plus their test references and replaces every call site with the new `KindCatalog`/`Template` API.
- **Paths.**
  - `internal/domain/kind.go` (delete `KindTemplateChildSpec` `:94-102`, `KindTemplate` `:104-110`, `AllowedParentScopes` field `:118`, `AllowsParentScope` method `:200-211`, `normalizeKindTemplate` `:296-352`, `normalizeKindParentScopes` `:274-293`)
  - `internal/domain/kind_capability_test.go` (call-site rewrites at LSP-confirmed `:18,20,49`)
  - `internal/domain/domain_test.go` (test fixtures using the deleted types — full file sweep needed)
  - `internal/app/kind_capability.go` (rewrite call site at `:566` `kind.AllowsParentScope(parent.Scope)` → `template.AllowsNesting(parent.Kind, kind.Kind)`; rewrite `validateKindTemplateExpansion` `:771` to use `Template.ChildRulesFor`; rewrite `mergeActionItemMetadataWithKindTemplate` `:750-766` to use `KindRule` defaults)
  - `internal/app/kind_capability_test.go` (extensive test rewrites; this is the 31k LOC file; sweep every `KindTemplate` reference)
  - `internal/app/snapshot.go` (rewrite call site at `:94`)
  - `internal/adapters/server/common/mcp_surface.go` (rewrite at `:248`)
  - `internal/adapters/server/mcpapi/extended_tools.go` (rewrite at `:1682,1778`)
  - `internal/adapters/storage/sqlite/repo.go` (delete `kind_catalog` boot-seed `:286-377` — `KindCatalog` is now baked from TOML at project create, not seeded into SQLite. Schema column `template_json` becomes vestigial; planner accepts it stays per pre-MVP no-migration rule.)
  - `internal/adapters/storage/sqlite/repo_test.go` (rewrite call site at `:2563`)
  - `cmd/till/main.go` (rewrite call sites at LSP-confirmed `:3042,3045,3047,3049,3442`)
- **Packages.** `internal/domain`, `internal/app`, `internal/adapters/server/common`, `internal/adapters/server/mcpapi`, `internal/adapters/storage/sqlite`, `cmd/till`.
- **Acceptance.**
  - `internal/domain/kind.go` no longer declares `KindTemplate`, `KindTemplateChildSpec`, `AllowedParentScopes`, `AllowsParentScope`, `normalizeKindTemplate`, `normalizeKindParentScopes`.
  - LSP `findReferences` on each deleted symbol returns 0 hits.
  - Every call site listed above migrated to the new API.
  - `mage test-pkg ./internal/domain`, `mage test-pkg ./internal/app`, `mage test-pkg ./internal/adapters/storage/sqlite`, `mage test-pkg ./internal/adapters/server/common`, `mage test-pkg ./internal/adapters/server/mcpapi`, `mage test-pkg ./cmd/till` all pass.
  - `mage ci` passes.
- **Mage target.** `mage ci` (multi-package sweep).
- **Blocked by.** 3.B.3, 3.B.4, 3.B.5 (consumers must switch to new API before old API is deleted).
- **Status.** todo.

### 3.B.9 — Audit-trail `till.comment` + attention item on every nesting rejection

- **Scope.** Per PLAN.md § 19.3 Template Overhaul Scope: "Audit-trail `till.comment` + attention item on every nesting rejection." Wire into the rejection path at `internal/app/kind_capability.go:resolveActionItemKindDefinition` (`:545-578`). When `Template.AllowsNesting` returns false, the existing rejection path now (a) creates a `till.comment` on the parent action item with `actor_type=user`/`actor_type=agent` (whichever the rejecting context is) describing the rejection + reason string, and (b) creates an `attention_item` for the dev with category `template_rejection` (new category — schema constant + tests).
- **Paths.**
  - `internal/app/kind_capability.go` (extend rejection path with comment + attention_item creation; ~30 LOC change)
  - `internal/app/kind_capability_test.go` (new test cases asserting comment + attention_item land on rejection)
  - `internal/domain/attention.go` or `internal/domain/inbox.go` (add `AttentionCategoryTemplateRejection` constant — LSP `workspaceSymbol` for `AttentionCategory` to find exact file)
  - `internal/app/inbox_attention.go` (extend if attention-item creation flows through this file — current call sites at `:17,51`)
- **Packages.** `internal/app`, `internal/domain`.
- **Acceptance.**
  - Every `Template.AllowsNesting → false` rejection at the create boundary creates a comment + attention_item.
  - Comment body markdown includes the rejection reason from `AllowsNesting`'s second return value verbatim.
  - Attention-item category is `template_rejection`; subject names parent + child kind.
  - Test coverage: at least 4 cases (closeout-no-closeout-parent, commit-no-plan-child, human-verify-no-build-child, build-qa-*-no-plan-child) end-to-end through the create path.
  - `mage test-pkg ./internal/app`, `mage test-pkg ./internal/domain` pass.
  - `mage ci` passes.
- **Mage target.** `mage ci` (final droplet — full validation cycle).
- **Blocked by.** 3.B.3 (needs `AllowsNesting`), 3.B.5 (needs `KindCatalog` on `Project`), 3.B.7 (default.toml provides the rejection rules tested), 3.B.8 (rejection path migrated to new API).
- **Status.** todo.

## Droplet Order Summary

```
3.B.1 (schema structs)
  └─→ 3.B.2 (parser + load-time validator)
        ├─→ 3.B.3 (AllowsNesting)
        │     ├─→ 3.B.4 (child_rules)
        │     ├─→ 3.B.5 (KindCatalog + project-create wiring)
        │     │     └─→ 3.B.7 (default.toml + embed)  [also blocks on 3.B.6]
        │     └─→ 3.B.8 (rewrite/delete old API)  [also blocks on 3.B.4, 3.B.5]
        └─→ 3.B.6 (agent binding fields — same schema.go file as 3.B.1)
              └─→ 3.B.7 (default.toml needs full agent-binding schema)
                    └─→ 3.B.9 (audit-trail comment + attention)
```

Cross-unit `blocked_by` from Unit A's `StructuralType` enum: 3.B.1, 3.B.4, 3.B.7. Orchestrator wires at synthesis.

## File-Level / Package-Level Lock Audit

Per CLAUDE.md "Blocker Semantics": same-file or same-package edits require explicit `blocked_by`.

| File / package | Droplets touching | Resolution |
| --- | --- | --- |
| `internal/templates/schema.go` | 3.B.1, 3.B.6 | 3.B.6 `blocked_by` 3.B.1 |
| `internal/templates/load.go` | 3.B.2 | sole owner |
| `internal/templates/nesting.go` | 3.B.3 | sole owner |
| `internal/templates/child_rules.go` | 3.B.4 | sole owner |
| `internal/templates/catalog.go` | 3.B.5 | sole owner |
| `internal/templates/embed.go` | 3.B.7 | sole owner |
| `templates/builtin/default.toml` | 3.B.7 | sole owner |
| `internal/app/kind_capability.go` | 3.B.5, 3.B.8, 3.B.9 | 3.B.8 `blocked_by` 3.B.5; 3.B.9 `blocked_by` 3.B.8 |
| `internal/app/kind_capability_test.go` | 3.B.5, 3.B.8, 3.B.9 | same chain |
| `internal/domain/kind.go` | 3.B.8 | sole owner |
| `internal/domain/project.go` | 3.B.5 | sole owner |
| `internal/adapters/storage/sqlite/repo.go` | 3.B.8 | sole owner |
| `cmd/till/main.go` | 3.B.8 | sole owner |
| `internal/templates` package (Go-package lock) | 3.B.1, 3.B.2, 3.B.3, 3.B.4, 3.B.5, 3.B.6, 3.B.7 | every droplet under this package serializes via the dependency chain above (no parallel work allowed within this package) |
| `internal/app` package (Go-package lock) | 3.B.5, 3.B.8, 3.B.9 | already serialized via the chain |

## Notes

### Open Questions Routed Back To Orchestrator

1. **Unit A `StructuralType` placeholder strategy.** If Unit A's enum lands after Unit B's droplet 3.B.1 starts, planner recommends builder stubs `StructuralType` as a `string` type alias, marked TODO with a follow-up tightening pass once Unit A merges. Alternative: gate Unit B entirely behind Unit A's enum landing (slower but simpler). Orch decides.
2. **`KindCatalog` import direction.** Current draft (3.B.5) raises a circular-import risk: `internal/domain/project.go` referencing `internal/templates.KindCatalog` while `internal/templates` references `domain.Kind`. Two resolutions documented in 3.B.5 acceptance section; builder picks. Either is sound.
3. **`kind_catalog` SQLite boot-seed fate.** Drop 1.75 lands the SQLite `kind_catalog` table with INSERT-OR-IGNORE seeds (`internal/adapters/storage/sqlite/repo.go:286-377`). Once `KindCatalog` is baked from TOML at project-create time, the boot-seed becomes vestigial. Per pre-MVP no-migration rule, planner recommends 3.B.8 deletes the seed INSERTs but **keeps the table** so existing schema doesn't break. Dev fresh-DBs after 3.B.8 lands. Orch confirms.
4. **Template per-project file resolution at project-create.** When dev runs `till project create`, where does the per-project `<project_root>/.tillsyn/template.toml` live BEFORE the project exists? Answer in 3.B.5: project-create reads the TOML from `<project_metadata.repo_primary_worktree>/.tillsyn/template.toml` if `repo_primary_worktree` is supplied at create time, else falls back to embedded `templates/builtin/default.toml`. Drop 4 (PLAN.md § 19.4) tightens project-create with first-class `repo_primary_worktree` field — Unit B leaves this seam clean for Drop 4 to wire.
5. **Audit-trail comment auth coordination with Unit C.** Section "Architectural Decisions" above resolves this: Unit B's rejection comment is written as the rejecting actor (no STEWARD principal needed), so no Unit C blocker. Orch confirms with Unit C planner.

### Out Of Scope (Explicit)

- **Dispatcher consuming agent bindings** — Drop 4. Unit B writes the schema; Drop 4 reads it.
- **Gate execution** — Drop 4. Unit B's `GateRule` struct is a stub.
- **Per-project template authoring UX** — out of MVP scope.
- **Template upgrades / `schema_version` v2+ migration** — pre-MVP rule = no migration. Future drops handle.

## Hylla Feedback

N/A — planning touched non-Go files only (PLAN.md droplet authoring) plus targeted reads of Go files via Read + LSP, no Hylla queries needed for this planning pass. Read-tool was sufficient to map the `KindTemplate` / `AllowsParentScope` API surface; LSP `findReferences` covered call-site enumeration. No Hylla miss to record.

Ergonomic note for Hylla orchestrator-side aggregation: planner-only Hylla-Feedback sections that stay `N/A` are still useful signal — they confirm the planning surface is fully covered by Read+LSP without forcing Hylla into Go source ranges where it'd duplicate gopls work.
