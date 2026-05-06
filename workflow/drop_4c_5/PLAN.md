# DROP_4c.5 — TEMPLATE_ERGONOMICS_AND_AUDIT_DEBT

**State:** planning
**Blocked by:** DROP_4c (shipped at `49da561`)
**Paths (expected):** `internal/app/`, `internal/templates/`, `internal/adapters/server/mcpapi/`, `internal/domain/`, `cmd/till/`, `go.mod`, `~/.claude/agents/*.md` cross-refs only.
**Packages (expected):** `internal/app`, `internal/templates`, `internal/adapters/server/mcpapi`, `internal/adapters/storage/sqlite`, `internal/domain`, `cmd/till`.
**PLAN.md ref:** project-root `PLAN.md` (Drop 4c.5 row to be added when this drop closes; pre-Drop-2 PLAN.md isn't currently authoritative).
**Workflow:** `workflow/example/drops/WORKFLOW.md`.
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`.
**Started:** 2026-05-05.
**Closed:** —

## Scope

Template ergonomics + audit-debt sweep so the cascade-on-itself dogfood loop (Drop 5) doesn't fight silent-data-loss bugs, missing escape hatches, or broken template ergonomics. Bundles deferred work from the original Drop 4c SKETCH (Themes A/B/C/D + F.1/F.2/F.3/F.5/F.6) plus accumulated 4a/4b refinement residue (Theme E). F.4 marketplace CLI deferred to Drop 4d-prime; F.7 spawn pipeline already shipped in Drop 4c. Full scope + open questions in `REVISION_BRIEF.md`.

## Planner

Per-theme planner outputs land in `THEME_<X>_PLAN.md` files (e.g. `THEME_A_PLAN.md`, `THEME_BD_PLAN.md`, `THEME_CE_PLAN.md`, `THEME_F_PLAN.md`). Master orchestration synthesizes the per-theme outputs into the droplet table below. Planning sub-streams kick off as parallel planner subagent spawns; their outputs feed plan-QA twins on the synthesized master plan.

### Planner Sub-Streams

- **`THEME_A_PLAN.md`** — Theme A silent-data-loss + agent-surface hardening (~4 droplets).
- **`THEME_BD_PLAN.md`** — Theme B dev escape hatches + Theme D pre-cascade hygiene (~3-4 droplets).
- **`THEME_CE_PLAN.md`** — Theme C STEWARD/cascade-precision refinements + Theme E 4a/4b residue carry-forward (~8-10 droplets).
- **`THEME_F_PLAN.md`** — Theme F template ergonomics F.1/F.2/F.3/F.5/F.6 (~13-16 droplets).

Per-theme PLAN MDs declare droplet IDs (`A.1`, `A.2`, …, `B.1`, `D.1`, `C.1`, `E.1`, …, `F.1.1`, `F.2.1`, …) with paths/packages/acceptance/blocked_by. Master plan-QA twins (`PLAN_QA_PROOF.md` + `PLAN_QA_FALSIFICATION.md`) attack the synthesized table.

### Droplet Table

<filled in after parallel theme planners + orchestrator synthesis returns>

## Notes

Open questions surfaced by REVISION_BRIEF §9 — Q1 default-fe.toml defer (lean: defer), Q2 validateAgentBindingFiles warn vs error (lean: warn), Q3 doc-only NIT scope (lean: correctness mandatory + doc opportunistic), Q4 client_type server-infer dispatcher coverage (lean: yes), Q5 Drop 5 readiness gate (lean: A+B mandatory) — resolved during plan-QA discussion if planners surface concrete tradeoffs.

Pre-MVP rules in force per REVISION_BRIEF §6: Opus builders, filesystem-MD mode, no Tillsyn per-droplet plan items, no closeout MD rollups, single-line conventional commits, never raw `go test` / `go build` / `go vet` / `mage install`, builder spawn prompts MUST include "do NOT commit" directive, REVISIONS-first reading where applicable, Section 0 SEMI-FORMAL REASONING in every subagent response.

Locked architectural decisions inherited from Drop 4c (REVISION_BRIEF §5): no `command` override, POSIX-only, closed env baseline, `os.Environ()` not inherited, dispatcher monitor stays CLI-agnostic, F.7.18 context aggregator OPTIONAL, commit + push gates default OFF.
