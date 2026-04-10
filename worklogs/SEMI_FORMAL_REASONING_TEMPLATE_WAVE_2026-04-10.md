# Semi-Formal Reasoning Template Wave

Created: 2026-04-10
Status: completed
Lane: `agent/semi-formal-template-wave`

## Objective

Tighten the shipped `default-go` builtin so Tillsyn's normal Go workflow enforces better semi-formal reasoning discipline without introducing a new top-level reasoning phase or a parallel reasoning template library.

This wave is complete when:
- `PLAN` explicitly requires Hylla-first premises, Context7 only where needed, counterexample thinking, and a written conclusion/unknowns shape for semantic or high-risk work,
- `QA PASS 1` and `QA PASS 2` stop being duplicate placeholders and become asymmetric verification gates,
- `CLOSEOUT` reconciles reasoning against final code plus fresh Hylla state,
- `build-task` auto-generates a required `COMMIT AND REINGEST` subtask so downstream reasoning is not built on stale Hylla premises,
- builtin default-go tests and dogfood setup docs reflect the new contract.

## Locked Scope

Implement now:
- update the shipped `templates/builtin/default-go.json`,
- add one new builtin prerequisite kind: `commit-and-reingest`,
- add one generated `COMMIT AND REINGEST` subtask under `build-task`,
- update the default-go dogfood setup markdown so bootstrap/install flows know about the new kind and child task,
- update builtin lifecycle/template tests to match the contract.

Defer for a later wave:
- a separate `reasoning-core` template library,
- dedicated `premise` / `case` / `conclusion` kinds,
- hard schema enforcement for Hylla / git-diff / Context7 source anchors,
- phase-level `END-OF-PHASE COMMIT AND REINGEST`,
- patch-equivalence-specific first-class template kinds.

## Design Direction

Keep the current lifecycle:
- `PLAN`
- `BUILD`
- `CLOSEOUT`
- generated `QA PASS 1`
- generated `QA PASS 2`

Do not add a new reasoning phase.

Instead, sharpen the reasoning contract inside the nodes that already exist:
- `PLAN` gathers evidence and shapes the task tree,
- `QA PASS 1` checks proof completeness,
- `QA PASS 2` tries to falsify the conclusion,
- `CLOSEOUT` reconciles final code, final Hylla state, and remaining risk.

## Operating Rules

Use these source rules consistently:
- use Hylla for committed repo-local evidence,
- use git diff for uncommitted local deltas,
- use Context7 only for external semantics the repository cannot prove,
- turn unresolved uncertainty into explicit comments, handoffs, or attention instead of optimistic completion.

Use this reasoning shape for semantic, high-risk, or ambiguous work:
- premises,
- evidence,
- trace or cases,
- conclusion,
- unknowns.

Keep the current workflow shape:
- `PLAN`
- `BUILD`
- `CLOSEOUT`
- generated `QA PASS 1`
- generated `QA PASS 2`
- generated `COMMIT AND REINGEST`

Do not add a new top-level reasoning phase in this wave.

## Skills To Build Next

These are the first-wave skills this template pass is meant to support:

1. `semi-formal-reasoning`
- shared convention skill for premises, evidence, trace or cases, conclusion, and unknowns.

2. `plan-from-hylla`
- fills PLAN work with repo-grounded evidence and current-state understanding.

3. `qa-sweep-reasoner`
- makes closeout QA challenge claims instead of rubber-stamping them.

4. `commit-and-reingest`
- runs the post-build freshness loop so downstream reasoning uses current Hylla state.

Expected execution surfaces:
- skill/shared guidance: `semi-formal-reasoning`,
- multi-step agent or worker flow: `plan-from-hylla`,
- multi-step agent or worker flow: `qa-sweep-reasoner`,
- orchestrator-triggered workflow or slash command: `commit-and-reingest`.

## Contract Changes

### Project Standards

Add explicit project standards that say:
- Hylla is the first source for repo-local evidence,
- Context7 is only for external semantics the repo cannot prove,
- semantic or high-risk work needs a written reasoning certificate shape,
- unresolved uncertainty must become explicit coordination or risk state,
- confirmed-good build work must be committed and re-ingested into Hylla before downstream reasoning relies on it.

### Plan Phase

Strengthen `plan-phase-template` so semantic/high-risk work must produce:
- premises,
- evidence,
- counterexample checks,
- conclusion/confidence/unknowns,
- a validation plan that can challenge the intended answer.

Update child descriptions so:
- Hylla work traces symbols, callers, and invariants,
- Context7 work only pins required external contracts,
- validation planning explicitly covers falsification and hidden dependencies.

### Build Task

Keep the existing build-task shape, but:
- clarify that semantic claims must match final code, not just intended code,
- add one required `COMMIT AND REINGEST` subtask.

`COMMIT AND REINGEST` means:
- commit confirmed-good work,
- trigger Hylla refresh,
- wait until Hylla is current to that commit,
- only then allow downstream reasoning to assume the graph is fresh.

### QA Passes

Keep the titles.

Make the passes asymmetric:
- `QA PASS 1`: verify evidence completeness and reasoning coherence,
- `QA PASS 2`: attempt falsification via counterexamples, alternate traces, hidden dependencies, or contract mismatches.

### Closeout

Strengthen `closeout-phase-template` so readiness requires:
- final checks green,
- Hylla current to final git state,
- reasoning still matches final code,
- unresolved risks are explicit instead of hand-waved away.

## Concrete File Changes

1. `templates/builtin/default-go.json`
- bump builtin version,
- update project standards markdown,
- tighten `plan-phase-template`,
- tighten `build-phase-template`,
- tighten `closeout-phase-template`,
- update `build-task-template` descriptions/defaults,
- add generated `COMMIT AND REINGEST`.

2. `TILLSYN_DEFAULT_GO_DOGFOOD_SETUP.md`
- add `commit-and-reingest` to the required kind list,
- document the generated `COMMIT AND REINGEST` child under each `build-task`,
- add it to the MCP-only setup and validation checklist.

3. Tests
- update builtin required-kind expectations,
- seed the new builtin kind in lifecycle tests,
- update generated child-title expectations for `build-task`,
- update any pinned builtin version strings.

## Validation Plan

Run:
- `mage test-pkg ./internal/app`
- `mage test-pkg ./internal/adapters/server/common`
- `mage ci`

## Result Summary

Implemented in this wave:
- sharper semi-formal reasoning expectations in the shipped `default-go` builtin,
- one required `COMMIT AND REINGEST` child task for every `build-task`,
- updated dogfood/bootstrap docs and lifecycle tests.

Still deferred:
- reasoning-specific libraries/kinds,
- source-anchor schema enforcement,
- phase-level reingest sweeps,
- patch-equivalence-first task templates.
