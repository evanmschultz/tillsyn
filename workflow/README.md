# workflow/

Pre-dogfood MD-only coordination substrate for Tillsyn drops. Until the
Tillsyn cascade dispatcher (`main/PLAN.md` Drop 4) is live, numbered drops
coordinate via filesystem + Markdown files rather than Tillsyn action items.

**For the design rationale — droplet atomic units, planner-calls-planner,
package-level automated gates, planner-level LLM QA, ancestor re-QA on
blocker failure, metrics catalog, benchmarking plan (arxiv 2603.01896) —
read `../AGENT_CASCADE_DESIGN.md`.** This directory is the operational
substrate; the design doc is the concept + rationale.

## Directory Map

- `example/` — generic reference implementation of the cascade workflow
  using plain Markdown files instead of a coordination runtime. Suitable
  for projects that want to run the cascade without Tillsyn (or before
  their runtime is ready). Every file uses `<PROJECT>` / `<package>` /
  `<org>` placeholders — copy into your project and replace.
  - `example/CLAUDE.md` — generic project-level CLAUDE.md (orchestrator
    role boundaries, agent bindings, evidence sources, language quality
    rule scaffold).
  - `example/drops/WORKFLOW.md` — generic 7-phase per-drop lifecycle
    (plan → plan-QA → discuss → build → build-QA → verify → closeout)
    with cascade-aware sub-drop recursion, package-level gates, and
    ancestor re-QA.
  - `example/drops/_TEMPLATE/` — per-drop scaffold (`PLAN.md`,
    `BUILDER_WORKLOG.md`, `CLOSEOUT.md`) the orchestrator copies at the
    start of each new drop.
  - `example/drops/DROP_N_EXAMPLE/` — concrete pedagogical walkthrough of
    one closed drop (scaffold a CLI + mage + CI in a fictional generic Go
    project). Shows how `PLAN.md`, `PLAN_QA_PROOF.md`, `BUILDER_WORKLOG.md`,
    and `CLOSEOUT.md` content evolves across a drop's lifecycle.
- `drop_1_5/` — Tillsyn's own active Drop 1.5 TUI work. First per-drop
  subdir under the **post-2026-04-19 doctrine**: git-tracked on the drop
  branch, flowing to `main` via PR merge. Under that doctrine the
  drop-orch writes artifact content directly into `drop_N/` on the drop
  branch; post-merge, the integrating orch (STEWARD in this repo) reads
  `main/workflow/drop_N/` and splices into the top-level MDs. Per-drop
  subdirs are **retained** after close — they are the permanent on-disk
  audit trail, not transient scratch. See `../PLAN.md` §15.9 +
  `../AGENT_CASCADE_DESIGN.md` §8.3.

## `failures/` Subdir Rule (Forward-Only From 2026-04-19)

Each branched level of `drop_N/` carries a `failures/` subdir. When a
plan, QA, or build round fails, its artifact content moves into
`failures/` at that level so the next iteration can read + count the
prior failure. **Never delete QA / plan / build artifacts.** Retention =
forever. **Forward-only**: pre-2026-04-19 drops are not retroactively
backfilled.

## Scope Note

The `example/` tree is the generic adopter-facing reference — public-release
content showing the MD-only cascade variant. `drop_N/` subdirs under this
project's `workflow/` are real per-drop artifact trails for Tillsyn's own
drops, git-tracked and durable.

Once the Tillsyn cascade dispatcher ships (Drop 4) and Tillsyn can dogfood
against itself via action items, the Tillsyn-specific runtime state for
planning + execution moves into the action-item DB, but the `workflow/drop_N/`
on-disk audit trail stays — it is STEWARD's post-merge input and the
permanent history. See `../PLAN.md` §19.10 for the
"Split `AGENT_CASCADE_DESIGN.md` into concept + operations before MVP"
refinement bullet, which is coupled with the final shape of this directory.
