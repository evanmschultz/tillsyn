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
- `drop_1_5/` — Tillsyn's own active Drop 1.5 TUI work, using the
  filesystem-MD pattern because Tillsyn itself is pre-cascade-dispatcher.
  Not public-release content; active scratch + audit trail for the
  `DROP_1.5_ORCH` session. Directory moves to retire-or-relocate once
  Drop 1.5 closes.

## Scope Note (Transient)

Everything here is **pre-cascade / pre-MVP dogfood substrate**. Once the
Tillsyn cascade dispatcher ships (Drop 4) and Tillsyn can dogfood against
itself via action items instead of MD files, most contents move out of
`workflow/` and into Tillsyn's runtime state. The `example/` tree stays
(as a public-release reference showing the MD-only variant for users
without Tillsyn installed), but the `drop_1_5/`-style per-drop scratch
dirs retire.

See `../PLAN.md` §19.10 for the "Split `AGENT_CASCADE_DESIGN.md` into
concept + operations before MVP" refinement bullet, which is coupled with
the final shape of this directory.
