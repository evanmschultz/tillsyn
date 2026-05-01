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

<To be filled by `go-planning-agent` in Phase 1. Planner emits droplets (or sub-drops) with `paths`, `packages`, `acceptance`, `blocked_by`, `state: todo` per `workflow/example/drops/WORKFLOW.md` § "Phase 1 — Plan" (used as a template-design reference; the project does not literally follow the 7-phase loop).>

## Notes

<Cross-droplet decisions, library choices made during planning, deferrals, YAGNI rulings — to be filled as planning proceeds.>
