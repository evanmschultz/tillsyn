# Drop 4c.5 — Template Ergonomics + Audit-Debt Sweep (Revision Brief)

**Status:** revision-brief authoring 2026-05-05.
**Author:** orchestrator (post-Drop-4c-merge).
**Drop scope (LOCKED):** template ergonomics (F.1, F.2, F.3, F.5, F.6) + audit-debt themes (A, B, C, D) + Theme E residue carry-forward from Drop 4a/4b refinements memory.
**Out of scope:** F.4 marketplace CLI (Drop 4d-prime, post-Drop-5); F.7 (already shipped in Drop 4c); Theme G post-MVP marketplace evolution; codex adapter (Drop 4d, post-Drop-5).

## 1. Hard Prerequisites

- Drop 4a + 4b + 4c on `main`. HEAD `49da561` (Drop 4c close).
- F.7 spawn pipeline shipped — `internal/app/dispatcher/cli_claude/`, `cli_adapter.go`, `bundle.go`, `monitor.go`, `handshake.go`, `commit_agent.go`, `gate_commit.go`, `gate_push.go`, `orphan_scan.go`, `binding_resolved.go`, `permission_grants*` storage, `cli_claude/render/`. Theme A's strict-decode and Theme B's escape hatches CAN now reference these surfaces if needed.

## 2. Goal

Close pre-Drop-5-dogfood polish + audit-debt items so the cascade-on-itself loop in Drop 5 doesn't fight silent-data-loss bugs, missing escape hatches, or broken template ergonomics. Bundles the deferred work from the original Drop 4c SKETCH plus accumulated 4a/4b refinement residue.

## 3. Scope (~25-30 droplets across 7 themes)

### 3.1 Theme A — Silent-data-loss + agent-surface hardening (~4 droplets)

From original Drop 4c SKETCH §"Theme A":

- **PATCH semantics on update handlers.** `Service.UpdateActionItem` writes every field unconditionally; empty `description` wipes stored description. Fix: pointer-sentinel input fields OR explicit-replace-all flag so omitted fields preserve.
- **Reject unknown keys at MCP boundary.** No `DisallowUnknownFields` across MCP server code. Schema-permissive create/update silently drops unknown fields. Fix: per-MCP-tool unknown-key rejection with structured error naming the offending key.
- **Server-infer `client_type` on auth-request create.** Asymmetric validation bug — currently accepts empty `client_type`; only the approve-path rejects it. Fix: MCP-stdio adapter stamps `"mcp-stdio"`, TUI stamps `"tui"`, CLI stamps `"cli"`. Tighten `app.Service.CreateAuthRequest` to reject empty.
- **Require non-empty `metadata.outcome` on `failed` transitions.** `validateMetadataOutcome` accepts empty. Domain-level validation: any transition to `failed` requires non-empty `metadata.outcome`.

### 3.2 Theme B — Dev escape hatches (~2 droplets)

- **Supersede CLI**: `till action_item supersede <id> --reason "..."` — marks `failed` action item as `metadata.outcome: "superseded"` and transitions `failed → complete`. Bypasses always-on parent-blocks invariant. Hard requirement before dogfood.
- **Failure listing CLI**: `till action_item list --state failed`. Pre-TUI dev visibility into failed action items.

### 3.3 Theme C — STEWARD + cascade-precision refinements (~3 droplets)

From `project_drop_3_refinements_raised.md`:

- **R1**: extend `assertOwnerStateGateUpdateFields` to Persistent / DevGated fields (today only blocks Owner / DropNumber on STEWARD-owned items; should block all four per L13).
- **R2**: tighten `raiseRefinementsGateForgottenAttention` doc-comment vs idempotency claim drift (godoc claims idempotent duplicate-insert handling that the impl doesn't perform).
- **R3**: tighten `isRefinementsGate` predicate (currently keys on Owner=STEWARD + StructuralType=Confluence + DropNumber>0; future false-positive risk if a second STEWARD-owned numbered confluence kind lands).
- **R5**: re-survey WIKI Cross-Subtree Exception kind-choice hedge after STEWARD's actual rollup-kind choices stabilize.

### 3.4 Theme D — Pre-cascade hygiene (~1-2 droplets)

- **`go.mod` `replace` directive cleanup**: strip every `replace` except the fantasy-fork. Pre-cascade hygiene from PLAN.md §19.1 first bullet.
- **Accumulated gopls / vet hints**: any LSP/`mage ci` hints that surfaced through 4a + 4b + 4c builds that aren't yet addressed.

### 3.5 Theme E — Drop 4a/4b residue carry-forward (~5-7 droplets)

From `project_drop_4a_refinements_raised.md` + `project_drop_4b_refinements_raised.md`:

**From 4a memory** (post-F.7-supersession filter):

- **R4 + R7 (cross-cutting)**: file-lock + package-lock managers — un-pinned input-order test contract, undocumented duplicate-input semantics; `mage coverFile` target for per-file coverage runtime-verifiability.
- **R5**: tree walker test rigor (archived-parent test, ListColumns error-path, blocker-state doc-comment drift).
- **R8**: conflict detector A14 (test rigor) + A6 (path canonicalization doc clarification).
- **R9**: process monitor — six NITs that didn't get addressed during Drop 4b structured-failure work (PLAN.md row 4a.21 `failure_reason` vs `BlockedReason` alignment, `Track` doc-comment "defer Close" guidance, atomicity edge case, `for-range int` modernization, `goleak.VerifyTestMain` hardening, S2 `mage testPkg` ergonomics doc).
- **R12**: `mapToolError` add `domain.ErrOrchSelfApprovalDisabled` sharp-prefix case.

**From 4b memory**:

- **R1**: `validateMapKeys` case-fold footgun project-wide. Templates with `[gates.BUILD]` (uppercase) load + carry `Kind("BUILD")` literal that never matches `tpl.Gates[domain.KindBuild]` lookups. Fix: exact-match validation OR canonicalize keys post-decode.
- **R2**: `mage_test_pkg` gate test-rigor (no-dedup contract assertion, halt-test explicit `len(runner.calls) == 1`, empty-string package element gate-level test).
- **R3**: 4b.5 auth auto-revoke — `path.ScopeType == ScopeAction` guard + reason-string source decision.
- **R4**: 4b.6 git-status pre-check NITs (`filteredGitEnv` shared package extraction, defensive-nil-checker silent-skip → panic-or-document, file-location decision documentation).

### 3.6 Theme F — Template ergonomics (~13-16 droplets)

From original Drop 4c SKETCH §"Theme F" (excluding F.4 deferred + F.7 shipped):

- **F.1 — Project-template auto-discovery (~3 droplets).** Wire `internal/app/service.go` `loadProjectTemplate()` (currently returns `(zero, false, nil)` per Drop 3.14 deferral) to walk `<project.RepoBareRoot>/.tillsyn/template.toml` first → `<project.RepoPrimaryWorktree>/.tillsyn/template.toml` → embedded `default.toml`. Each candidate runs through `templates.Load(r io.Reader)` for full validation. Position-aware errors surface to project-create.
- **F.2 — Generic + Go + FE builtin separation (~4 droplets).** Refactor `internal/templates/builtin/`: `default-generic.toml` (language-agnostic showcase) + `default-go.toml` (current `default.toml` content rebadged) + `default-fe.toml` (FE agent bindings). `internal/templates/embed.go` resolver picks based on `project.Language` at bake time. This repo gets `<project_root>/.tillsyn/template.toml` for self-host dogfood.
- **F.3 — `till.template` MCP tool (~3 droplets).** Operations: `get` (current bake state), `validate` (run validation chain on candidate TOML), `set` (validate + install + re-bake catalog), `list_builtin`. TOML-in / TOML-out wire format.
- **F.5 — Extended validation (~2 droplets).** New checks layered into `templates.Load`: `validateAgentBindingFiles` (warn-only, check `~/.claude/agents/<name>.md` exists), `validateRequiredChildRules` (assert canonical QA-required parents have mandatory child rules), `validateChildRuleReachability` (kind-orphan detection — every kind reachable from `plan` via child_rules), `validateKindStructuralCoherence` (light cross-axis check on structural_type expectations).
- **F.6 — Cleanup of legacy KindTemplate stub (~1 droplet).** `internal/app/kind_capability.go:1002` `mergeActionItemMetadataWithKindTemplate` is a no-op pass-through stub; fold into its caller (`internal/app/service.go:716`) per the doc-comment promise.

## 4. Out of Scope

- F.4 marketplace CLI (Drop 4d-prime).
- F.7 spawn pipeline (shipped Drop 4c).
- Theme G post-MVP marketplace evolution (TUI marketplace browser, vector search, contribution flow, runtime simulation, template inheritance, template-bound agent prompts, versioned template references).
- Codex adapter (Drop 4d).
- Drop 4.5 TUI overhaul (concurrent FE/TUI track).
- Drop 5 dogfood validation.

## 5. Locked Architectural Decisions (Inherited from Drop 4c)

These are non-negotiable carried forward from the F.7 wave:

- **L1** Tillsyn never holds secrets. Env-var NAMES only.
- **L2** No Docker awareness, no OAuth registry, no `command` override knob. OS-level wrappers handle isolation.
- **L3** POSIX-only.
- **L4** Closed env baseline (process basics + network conventions); `os.Environ()` not inherited.
- **L11** Dispatcher monitor stays CLI-agnostic via adapter seam.
- **L13** F.7.18 context aggregator is OPTIONAL (Theme F.5 + F.1 templates may declare or omit `[context]`).
- **L20** Commit + push gates default OFF; default template ships gates listed-but-toggle-disabled.

## 6. Pre-MVP Rules In Force

Same as Drop 4c:

- No migration logic in Go. Schema additions ship inline in storage init path; dev fresh-DBs.
- No closeout MD rollups (LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_FEEDBACK) — pre-dogfood.
- Opus builders. Every builder spawn carries `model: opus`.
- Filesystem-MD mode. No Tillsyn-runtime per-droplet plan items.
- Tillsyn-flow output style + Section 0 SEMI-FORMAL REASONING in every subagent response.
- Single-line conventional commits ≤72 chars.
- NEVER raw `go test` / `go build` / `go vet` / `mage install`. Always `mage <target>`.
- Hylla is Go-only today.
- **Builder spawn prompts MUST include "do NOT commit" directive (per F.7-CORE REV-13).** Orchestrator drives commits AFTER QA pair returns green.
- **Each builder reads REVISIONS POST-AUTHORING section first** if the sub-plan has one.

## 7. Wave Structure (Tentative)

Plan-QA-twins will refine the wave structure. Initial sketch:

- **Wave A (~4-5 droplets):** Theme A (silent-data-loss + MCP hardening).
- **Wave B (~2 droplets):** Theme B (dev escape hatches).
- **Wave C (~5-7 droplets):** Theme E residue (4a/4b refinements that didn't fold into Drop 4c F.7).
- **Wave D (~3-4 droplets):** Theme C (STEWARD R1-R3, R5) + Theme D (go.mod cleanup, vet hints).
- **Wave E (~13-16 droplets):** Theme F template ergonomics — F.1 (auto-discovery) → F.2 (builtin separation) → F.3 (MCP tool) → F.5 (extended validation) → F.6 (KindTemplate cleanup).

Total: ~27-34 droplets. Plan-QA may absorb / split / re-sequence.

## 8. Concrete Planner Spawn Contract (For Wave Planners)

Multiple parallel planner spawns (one per theme or wave). Each planner:

- Reads this REVISION_BRIEF + relevant Drop 4a/4b refinements memory + 4c F.7 architectural docs (`SPAWN_PIPELINE.md`, `CLI_ADAPTER_AUTHORING.md`).
- Reads code surfaces for evidence (`internal/app/service.go`, `internal/templates/`, `internal/adapters/server/mcpapi/`, etc.).
- Authors a per-theme PLAN MD (`workflow/drop_4c_5/THEME_<X>_PLAN.md`) with per-droplet acceptance criteria, test scenarios, falsification mitigations, verification gates.
- NO Hylla calls (Hylla stale post-Drop-4c-merge).
- Section 0 SEMI-FORMAL REASONING required. Tillsyn-flow output style.

## 9. Open Questions for Plan-QA Review

- **Q1 — Theme F.2 builtin separation timing.** Does this drop need both `default-generic.toml` AND `default-fe.toml`, or is `default-go.toml` (rebadge of current `default.toml`) sufficient since this repo is Go-only? Lean: ship generic + go now, defer FE until an actual FE adopter materializes.
- **Q2 — Theme F.5 `validateAgentBindingFiles` warn vs error.** Soft warning when `~/.claude/agents/<name>.md` doesn't exist on dev's machine, OR hard reject? Lean: warn (dev's machine state vs template-correctness are different concerns).
- **Q3 — Theme E residue prioritization.** Some 4a/4b NITs are pure documentation; others are real correctness gaps (4b R1 case-fold footgun, 4a R12 mapToolError). Are doc-only NITs in scope, or only correctness gaps?
- **Q4 — Theme A `client_type` server-infer.** Adapter-stamped client type — does dispatcher path also need to stamp (cascade-spawned auth requests)? Lean: yes, for full cascade-on-itself coherence.
- **Q5 — Drop 5 readiness gate.** Does Drop 5 dogfood need ALL of Drop 4c.5 to land first, or only Themes A + B (silent-data-loss + escape hatches)? Lean: A + B mandatory; C/D/E/F can land in parallel with Drop 5 if needed.

## 10. Approximate Size

~27-34 droplets total. ~3-5 days at Drop 4a/4b/4c's pace (assuming similar parallelism + few fix-builder rounds). Larger than Drop 4b but smaller than Drop 4c (which had heavy F.7 architectural surfaces). Most items are 1-3 file edits each; F.2 builtin separation + F.3 MCP tool are the heaviest chunks at ~3-4 droplets each.

## 11. References

- `workflow/drop_4c/SKETCH.md` (post-Drop-4c-rework) — original Drop 4c SKETCH; Drop 4c.5 absorbs its Theme A/B/C/D + F.1-F.3/F.5-F.6 sections.
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4a_refinements_raised.md` — Drop 4a refinement residue (R1-R13 minus R6 absorbed by F.7).
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_4b_refinements_raised.md` — Drop 4b refinement residue (R1-R4).
- `~/.claude/projects/-Users-evanschultz-Documents-Code-hylla-tillsyn/memory/project_drop_3_refinements_raised.md` — Drop 3 refinements feeding Theme C (R1, R2, R3, R5).
- `SPAWN_PIPELINE.md`, `CLI_ADAPTER_AUTHORING.md` — Drop 4c F.7 reference docs.
- `WIKI.md`, `PLAN.md`, `CLAUDE.md` — project orchestration discipline.
- `workflow/example/drops/WORKFLOW.md` — drop lifecycle phases.
