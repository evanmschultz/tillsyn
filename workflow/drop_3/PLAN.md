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

<To be filled by `go-planning-agent`(s) in Phase 1. Decomposition strategy is an open question — see "Decomposition Strategy" in `## Notes` below.>

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
