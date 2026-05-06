# DROP_4c.5 — TEMPLATE_ERGONOMICS_AND_AUDIT_DEBT

**State:** planning
**Blocked by:** DROP_4c (shipped at `49da561`)
**Paths (expected):** `internal/app/`, `internal/templates/`, `internal/adapters/server/mcpapi/`, `internal/adapters/server/common/`, `internal/domain/`, `cmd/till/`, `internal/app/dispatcher/`, `internal/platform/gitenv/` (NEW), `WIKI.md`, `go.mod`, `go.sum`, `.tillsyn/template.toml` (NEW), `~/.claude/agents/*.md` cross-refs only.
**Packages (expected):** `internal/app`, `internal/templates`, `internal/adapters/server/mcpapi`, `internal/adapters/server/common`, `internal/adapters/storage/sqlite`, `internal/domain`, `internal/app/dispatcher`, `internal/tui/gitdiff`, `internal/platform/gitenv` (NEW), `cmd/till`.
**PLAN.md ref:** project-root `PLAN.md` (Drop 4c.5 row to be added when this drop closes; pre-Drop-2 PLAN.md isn't currently authoritative).
**Workflow:** `workflow/example/drops/WORKFLOW.md`.
**Cascade concept:** `AGENT_CASCADE_DESIGN.md`.
**Started:** 2026-05-05.
**Closed:** —

## Scope

Template ergonomics + audit-debt sweep so the cascade-on-itself dogfood loop (Drop 5) doesn't fight silent-data-loss bugs, missing escape hatches, or broken template ergonomics. Bundles deferred work from the original Drop 4c SKETCH (Themes A/B/C/D + F.1/F.2/F.3/F.5/F.6) plus accumulated 4a/4b refinement residue (Theme E). F.4 marketplace CLI deferred to Drop 4d-prime; F.7 spawn pipeline already shipped in Drop 4c. Full scope + open questions in `REVISION_BRIEF.md`. **Total droplets: 34.**

## Per-Theme Source-of-Truth PLANs

The master plan below indexes per-droplet specs in the per-theme PLAN MDs. Builders read the per-theme MD for full paths/packages/acceptance/test-scenarios/falsification mitigations:

- **Theme A** (silent-data-loss + agent-surface hardening, 4 droplets): `workflow/drop_4c_5/THEME_A_PLAN.md`.
- **Theme B+D** (escape hatches + go.mod/vet hygiene, 4 droplets): `workflow/drop_4c_5/THEME_BD_PLAN.md`.
- **Theme C+E** (STEWARD refinements + 4a/4b residue, 13 droplets): `workflow/drop_4c_5/THEME_CE_PLAN.md`.
- **Theme F** (template ergonomics F.1/F.2/F.3/F.5/F.6, 13 droplets): `workflow/drop_4c_5/THEME_F_PLAN.md`.

## Planner

### Open Question Resolutions (from REVISION_BRIEF §9)

- **Q1 — Theme F.2 default-fe.toml defer.** **RESOLVED: DEFER.** Ship `default-generic.toml` + `default-go.toml` only; `LoadDefaultTemplateForLanguage("fe")` returns `ErrLanguageNotSupported`. Rationale: pre-MVP, no FE adopters; FE templates land via F.4 marketplace CLI post-Drop-4d-prime.
- **Q2 — F.5 `validateAgentBindingFiles` warn vs error.** **RESOLVED: WARN-ONLY.** Dev-machine state ≠ template-correctness. Adopters who want strict-fail can wrap the warn-logger to escalate.
- **Q3 — Theme E doc-only NIT scope.** **RESOLVED: CORRECTNESS MANDATORY; DOC-ONLY OPPORTUNISTIC.** Pure-doc NITs fold into correctness droplets when they ride along (E.1, E.4, E.5, E.8). Pure-doc-only items dropped: `goleak.VerifyTestMain` (out-of-scope test-infra hygiene), S2 `mage testPkg` PLAN-doc (process MD, route to refinements memory), A13 conflict-detector single-flight (Drop 4b daemon-mode follow-up). C.4 (WIKI Cross-Subtree Exception clarification) retained because §3.3 explicitly carves it out.
- **Q4 — Theme A `client_type` server-infer dispatcher coverage.** **RESOLVED: ADAPTER-STAMP IS SUFFICIENT.** Dispatcher today provisions auth via the CLI path (which now stamps `"cli"`), so cascade subagents inherit `"cli"`. If dispatcher gains a direct `Service.CreateAuthRequest` call, that path stamps `"cli-cascade"` — out of scope for 4c.5; tracked as a Drop 4d / Drop 5 follow-up.
- **Q5 — Drop 5 readiness gate.** **RESOLVED: A + B MANDATORY; C/D/E/F MAY LAND IN PARALLEL OR DEFER.** Drop 5 dogfood requires Theme A (silent-data-loss closed) + Theme B (escape hatches landed). The remaining themes can land in this drop or in parallel with Drop 5. Drop 4c.5 still aims to land all 7 themes; the Q5 ruling is a contingency for if the drop runs long.

### Cross-Theme Cross-Cutting Decisions

- **No new domain types added by reusing existing fields.** B.1's supersede uses existing `metadata.outcome = "superseded"` (already recognized by `app_service_adapter_mcp.go:1163`) and `metadata.transition_notes` (existing free-form field) — no new `Metadata.SupersedeReason` field.
- **A.4's strict-failure-outcome-enum check (rejecting `"success"` on `→failed`) — INCLUDE.** Builder should add the switch on `metadata.outcome ∈ {"failure", "blocked", "superseded"}` for `→failed`. Cost is one switch + one test row; semantic value is high (`"success"` on `→failed` is nonsense).
- **E.6 fix-path: post-decode canonicalization (NOT exact-match).** Per Theme C+E planner's Note 2 reasoning: `domain.IsValidKind` already case-folds, validation contract is "case-tolerant," forcing exact-match diverges value-validation from key-validation surfaces, and post-canonicalization the collision case (`[gates.BUILD]` AND `[gates.build]`) becomes detectable. Plan-QA may flip if reasoning is rejected.
- **E.9 placement: `internal/platform/gitenv` (NOT `internal/utils/`).** CLAUDE.md "Project Structure" lists `internal/platform — OS-specific paths`; gitenv fits the platform-isolation theme. No `internal/utils/` exists in current project layout.
- **F.6.1 + Theme A `service.go` collision: F.6.1 lands in the chain.** F.6.1's 5-line inline refactor (replace `mergeActionItemMetadataWithKindTemplate(base, _)` call with `mergedMetadata := in.Metadata`) sits in the `internal/app` chain at a slot that doesn't conflict with concurrent A.x edits. Per `internal/app` package-lock chain below.

### Master Droplet Table

Droplets are grouped by package-lock chain. Within each chain droplets serialize via `blocked_by`. Across chains droplets parallelize except where cross-theme blockers apply.

#### Chain 1 — `internal/app` package-lock chain (12 droplets)

| ID    | Title (short)                                       | Files (primary)                                                                  | Source PLAN MD             | Blocked by              |
| ----- | --------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------- | ----------------------- |
| A.1   | Pointer-sentinel PATCH on `UpdateActionItem`        | `service.go` (664-763, 1201-1388), tests + adapter mappings                      | `THEME_A_PLAN.md`          | —                       |
| A.4   | Require `metadata.outcome` on `→failed`             | `service.go` (1043-1127), `domain/errors.go`, tests                              | `THEME_A_PLAN.md`          | A.1                     |
| B.1   | `till action_item supersede` CLI + service method   | `service.go` (new method), `app_service_adapter_mcp.go`, `cmd/till/*`            | `THEME_BD_PLAN.md`         | A.4                     |
| B.2   | `till action_item list --state failed` CLI          | `service.go` (new `ListActionItemsByState`), `cmd/till/*`                         | `THEME_BD_PLAN.md`         | B.1                     |
| C.2   | Idempotency: `raiseRefinementsGateForgottenAttention` doc vs impl | `auto_generate_steward.go`, `auto_generate_steward_test.go`             | `THEME_CE_PLAN.md`         | B.2                     |
| C.3   | Tighten `isRefinementsGate` predicate (title-shape) | `auto_generate_steward.go`, `auto_generate_steward_test.go`                      | `THEME_CE_PLAN.md`         | C.2                     |
| E.8   | Auth auto-revoke ScopeType guard + reason-string    | `auth_requests.go` (938), `auth_requests_test.go`                                | `THEME_CE_PLAN.md`         | C.3                     |
| F.6.1 | Inline `mergeActionItemMetadataWithKindTemplate`    | `service.go:897`, `kind_capability.go:1002`                                      | `THEME_F_PLAN.md`          | E.8                     |
| F.1.1 | Wire `loadProjectTemplate` embedded fallback        | `service.go` (427), `service_test.go`                                            | `THEME_F_PLAN.md`          | F.6.1, F.2.1            |
| F.1.2 | `loadProjectTemplate` filesystem walk               | `service.go` (extend F.1.1), tests                                               | `THEME_F_PLAN.md`          | F.1.1, F.1.3            |
| F.2.4 | Caller audit + cross-package tests for language-aware loading | `service.go` callers, tests                                            | `THEME_F_PLAN.md`          | F.1.3, F.2.1, F.2.2 (transitively F.1.2 via Chain 1) |
| E.9   | Git-status pre-check NITs + `internal/platform/gitenv` | `git_status.go`, `service.go` (1015-1019), new `internal/platform/gitenv/` | `THEME_CE_PLAN.md`         | F.2.4                   |

#### Chain 2 — `internal/adapters/server/mcpapi` package-lock chain (6 droplets)

| ID    | Title (short)                                                  | Files (primary)                                            | Source PLAN MD     | Blocked by              |
| ----- | -------------------------------------------------------------- | ---------------------------------------------------------- | ------------------ | ----------------------- |
| A.2   | Reject unknown JSON keys at MCP boundary (`bindArgumentsStrict`) | `handler.go`, `extended_tools.go`, `handoff_tools.go`, new `strict_decode.go` | `THEME_A_PLAN.md` | A.1 (cross-chain — wire shape coordination) |
| A.3   | Server-infer / require non-empty `client_type`                 | `handler.go` (113, 199), CLI (cmd/till/main.go)            | `THEME_A_PLAN.md`  | A.2                     |
| E.5   | `mapToolError` adds `ErrOrchSelfApprovalDisabled` sharp prefix | `handler.go` (891-948), `handler_test.go`                  | `THEME_CE_PLAN.md` | A.3                     |
| F.3.1 | `till.template` MCP tool: `get` + `list_builtin` operations    | `extended_tools.go` (new `registerTemplateTools`), `handler.go`, `template_service.go` (NEW) | `THEME_F_PLAN.md` | E.5, F.2.1, F.2.2, F.1.2 |
| F.3.2 | `till.template` MCP tool: `validate` operation                 | `extended_tools.go`, `template_service.go`                 | `THEME_F_PLAN.md`  | F.3.1, F.5.1            |
| F.3.3 | `till.template` MCP tool: `set` operation (atomic install)     | `extended_tools.go`, `template_service.go`                 | `THEME_F_PLAN.md`  | F.3.2, F.1.2            |

#### Chain 3 — `internal/app/dispatcher` package-lock chain (5 droplets)

| ID    | Title (short)                                                                 | Files (primary)                                              | Source PLAN MD     | Blocked by  |
| ----- | ----------------------------------------------------------------------------- | ------------------------------------------------------------ | ------------------ | ----------- |
| E.1   | Lock manager doc + test contract: input-order + duplicate-input               | `locks_file.go`, `locks_package.go` + tests                  | `THEME_CE_PLAN.md` | —           |
| E.2   | Tree walker test rigor: archived-parent + ListColumns error path              | `walker.go`, `walker_test.go`                                | `THEME_CE_PLAN.md` | E.1         |
| E.3   | Conflict detector: assert both file+package overlap entries                   | `conflict.go`, `conflict_test.go`                            | `THEME_CE_PLAN.md` | E.2         |
| E.4   | Process monitor: `Track` doc + atomicity edge case + `for-range int`          | `monitor.go`, `monitor_test.go`                              | `THEME_CE_PLAN.md` | E.3         |
| E.7   | `gate_mage_test_pkg` test rigor: no-dedup + halt-call + empty-string element  | `gate_mage_test_pkg_test.go`                                 | `THEME_CE_PLAN.md` | E.4         |

#### Chain 4 — `internal/templates` package-lock chain (6 droplets)

| ID    | Title (short)                                                                 | Files (primary)                                          | Source PLAN MD     | Blocked by  |
| ----- | ----------------------------------------------------------------------------- | -------------------------------------------------------- | ------------------ | ----------- |
| F.2.1 | Rebadge `default.toml` → `default-go.toml`                                    | `embed.go`, `builtin/default-go.toml` (RENAME), `embed_test.go` | `THEME_F_PLAN.md` | —           |
| F.2.2 | Add `default-generic.toml` (language-agnostic showcase)                       | `builtin/default-generic.toml` (NEW), `embed.go`, tests  | `THEME_F_PLAN.md`  | F.2.1       |
| F.1.3 | Language-aware embedded resolver                                              | `embed.go`, `embed_test.go`                              | `THEME_F_PLAN.md`  | F.2.1, F.2.2 |
| E.6   | `validateMapKeys` post-decode canonicalization                                | `load.go` (284-301), `load_test.go`                      | `THEME_CE_PLAN.md` | F.1.3       |
| F.5.1 | `validateAgentBindingFiles` (warn-only) + `validateRequiredChildRules`        | `load.go`, `load_test.go`                                | `THEME_F_PLAN.md`  | E.6         |
| F.5.2 | `validateChildRuleReachability` + `validateKindStructuralCoherence`           | `load.go`, `load_test.go`                                | `THEME_F_PLAN.md`  | F.5.1       |

#### Chain 5 — `internal/adapters/server/common/app_service_adapter_mcp.go` file-lock chain (1 droplet, plus A.1 + B.1 cross-chain)

A.1 and B.1 (already in Chain 1) ALSO edit `internal/adapters/server/common/app_service_adapter_mcp.go` as secondary file. C.1's primary edits target this file. Per CLAUDE.md "File- and package-level blocking" — droplets sharing a file MUST have explicit `blocked_by`. C.1 lands AFTER both A.1 and B.1 to avoid mechanical merge conflicts on this file.

| ID    | Title (short)                                                          | Files (primary)                                                                                              | Source PLAN MD     | Blocked by                              |
| ----- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ------------------ | --------------------------------------- |
| C.1   | Extend `assertOwnerStateGateUpdateFields` to Persistent / DevGated     | `app_service_adapter_mcp.go`, `app_service_adapter_steward_gate_test.go` (`internal/adapters/server/common`) | `THEME_CE_PLAN.md` | B.1 (transitively A.1 via Chain 1)      |

#### Independent (no package-lock collision)

| ID    | Title (short)                                                            | Files (primary)                                                | Source PLAN MD     | Blocked by |
| ----- | ------------------------------------------------------------------------ | -------------------------------------------------------------- | ------------------ | ---------- |
| C.4   | WIKI Cross-Subtree Exception kind-choice clarification                   | `WIKI.md` (markdown only)                                      | `THEME_CE_PLAN.md` | —          |
| D.1   | Strip non-fantasy-fork `go.mod` `replace` directives                     | `go.mod`, `go.sum`, possibly `third_party/teatest_v2/`         | `THEME_BD_PLAN.md` | —          |
| D.2   | Sweep accumulated vet / gopls / `mage ci` hints                          | `D2_HINT_SWEEP.md` (NEW per-droplet) + TBD per sweep          | `THEME_BD_PLAN.md` | D.1        |
| F.2.3 | Self-host `<repo_root>/.tillsyn/template.toml` for tillsyn dogfood       | `.tillsyn/template.toml` (NEW), `.gitignore` verification     | `THEME_F_PLAN.md`  | F.2.1      |

### Cross-Theme Blocked-By Justifications

- **A.2 → A.1.** Both touch the wire-format for action-item update (A.1 makes `description` distinguishable absent-vs-empty via `*string`; A.2 strict-decoder must NOT reject null-pointer fields from A.1's struct shape change). Cross-chain blocker because A.1 is in the `internal/app` chain and A.2 is in the `mcpapi` chain.
- **C.1 → B.1 (cross-chain Chain 1 → Chain 5).** A.1 + B.1 both edit `internal/adapters/server/common/app_service_adapter_mcp.go` as secondary surface (UpdateActionItem mapping in A.1; SupersedeActionItem passthrough in B.1). C.1's primary surface is the same file (`assertOwnerStateGateUpdateFields` body + caller). File-collision rule per CLAUDE.md "Paths and Packages" requires explicit `blocked_by`. Chosen ordering: C.1 lands after B.1 (which transitively follows A.1 in Chain 1), so the adapter file is in stable A+B shape before C.1's gate-extension edits. Per Theme C+E plan acceptance #3, C.1 also semantically depends on A.1's pointer-sentinel framework: "Caller (`UpdateActionItem` at line 845) extends pre-fetch trigger to include `in.Persistent != nil || in.DevGated != nil`."
- **E.6 → F.1.3 (cross-chain Theme F → Theme E).** F.1.3 lands the language-aware resolver that loads embedded TOML files; E.6's canonicalization changes the validator chain that runs during `Load`. E.6 lands AFTER F.1.3 to avoid the canonicalization touching F.1.3's walk semantics. Could potentially flip; plan-QA assesses.
- **F.5.1 → E.6.** Both edit `internal/templates/load.go` validator chain; sequential edits to avoid merge conflict.
- **F.5.1 transitively → F.2.1.** F.5.1's tests load the embedded default; the rebadge from `default.toml` → `default-go.toml` (F.2.1) must land first. Transitive via E.6 → F.1.3 → F.2.1, but called out for traceability.
- **F.1.1 → F.6.1 (intra-Chain-1).** Inserted in the `internal/app` chain after F.6.1 lands; F.6.1's small refactor lands first since it has no pre-Chain-1 dependencies.
- **F.1.1 → F.2.1 (cross-chain Templates → App).** F.1.1's `LoadDefaultTemplate()` thin-wrapper depends on F.2.1's rename (`default.toml` → `default-go.toml`) so the embedded fallback resolves correctly.
- **F.2.4 → F.1.3 + F.2.1 + F.2.2 (cross-chain Chain 4 → Chain 1).** F.2.4's caller audit redirects every `LoadDefaultTemplate()` call to `LoadDefaultTemplateForLanguage(...)` (F.1.3); requires the rebadge (F.2.1) and generic template (F.2.2) to exist.
- **F.3.1 → F.2.1 + F.2.2 + F.1.2 (cross-chain).** `till.template list_builtin` enumerates the renamed files; `get` op's bake-source provenance string depends on F.1.2's walk landing.
- **F.3.2 → F.5.1 (cross-chain).** `validate` op surfaces F.5.1's warn-logger output in its envelope.
- **F.3.3 → F.1.2 (cross-chain).** `set` op's atomic-install path writes to the same destination F.1.2's walk reads from.

### Wave Structure (Plan-QA Refines)

- **Wave A — Foundation parallel-launches** (no `blocked_by` edges; all 5 are true Wave A heads):
  - A.1 (Chain 1 head)
  - C.4 (markdown only, no code edits)
  - D.1 (go.mod, no Go package)
  - E.1 (Chain 3 head)
  - F.2.1 (Chain 4 head)

- **Wave B — Second-tier launches** (depends only on Wave A items):
  - A.4 (Chain 1 progresses; blocked_by: A.1)
  - A.2 (Chain 2 head, blocked_by: A.1)
  - D.2 (blocked_by: D.1)
  - E.2 (Chain 3 progresses; blocked_by: E.1)
  - F.2.2 (Chain 4 progresses; blocked_by: F.2.1)
  - F.2.3 (independent of chains; blocked_by: F.2.1)

- **Wave C — Mid-drop sequential:**
  - B.1, B.2 progressing in Chain 1
  - A.3, E.5 progressing in Chain 2
  - E.3, E.4 progressing in Chain 3
  - F.1.3, E.6 progressing in Chain 4
  - F.5.1, F.5.2 starting in Chain 4

- **Wave D — Tail droplets:**
  - F.6.1, F.1.1, F.1.2, F.2.4, E.9 in Chain 1 (last few)
  - F.3.1, F.3.2, F.3.3 in Chain 2 (template MCP tool)
  - E.7 (Chain 3 tail)

- **Wave E — Final:**
  - C.2, C.3, E.8 in Chain 1 (remaining mid-chain droplets — see chain ordering above)
  - C.1 (Chain 5; blocked_by: B.1 — must follow B.1's adapter-file edits)

(Wave structure is approximate; cross-chain blockers create natural barriers. Plan-QA may re-sequence. Chain 1's serial path through 12 droplets is the wall-clock bottleneck.)

## Notes

### Chain Length Trade-offs

Chain 1 (`internal/app`) is the bottleneck at 12 droplets serial. Chains 2-4 run parallel where blockers permit. Independent droplets (C.1, C.4, D.1, D.2, F.2.3) consume no chain capacity. Estimated wall-clock: ~1.5-2 hours per chain at ~10-15min per droplet. With 4 parallel chains, full drop is ~2-3 hours plus QA + commit overhead.

### Out-of-Scope Items Routed Away

- A13 conflict-detector single-flight (Drop 4b daemon-mode).
- R10 / R11 / R6 / R3 (already addressed in earlier drops; memory-marked).
- `goleak.VerifyTestMain` + S2 mage doc (test-infra hygiene; route to Drop 5+).
- F.4 marketplace CLI (Drop 4d-prime).
- Codex adapter (Drop 4d).
- Drop 5 dogfood validation (this drop is the prerequisite).
- Drop 4.5 TUI overhaul (separate FE/TUI track).

### Pre-MVP Rules (per REVISION_BRIEF §6)

Builders run `model: opus`. Filesystem-MD mode, no Tillsyn-runtime per-droplet plan items, no closeout MD rollups, single-line conventional commits ≤72 chars, never raw `go test` / `go build` / `go vet` / `mage install`. Builder spawn prompts MUST include "do NOT commit" directive (per F.7-CORE REV-13). Each builder reads REVISIONS POST-AUTHORING section first if the sub-plan adds one. Section 0 SEMI-FORMAL REASONING in every subagent response.

### Locked Architectural Decisions (inherited from Drop 4c, REVISION_BRIEF §5)

L1 (no secrets), L2 (no command override), L3 (POSIX-only), L4 (closed env baseline), L11 (CLI-agnostic monitor), L13 (context aggregator OPTIONAL), L20 (commit + push gates default OFF). All non-negotiable.

### Builder Spawn-Prompt Template

Each builder spawn for a Drop 4c.5 droplet uses this template. Substitute `<X>` placeholders per droplet.

```
You are the builder for droplet <DROPLET_ID> of Drop 4c.5 in the Tillsyn repo (`/Users/evanschultz/Documents/Code/hylla/tillsyn/main`). HEAD `<CURRENT_HEAD_SHA>` on `main`.

**Paradigm override:** filesystem-MD coordination mode. NO Tillsyn runtime calls. NO Hylla calls (stale post-Drop-4c-merge until reingest). Use Read / Grep / Glob / LSP / Bash / git diff for evidence.

**REQUIRED PRE-WORK READING:**
1. `workflow/drop_4c_5/PLAN.md` (master plan).
2. `workflow/drop_4c_5/<SOURCE_THEME_PLAN>.md` — your droplet's source-of-truth spec.
3. `workflow/drop_4c_5/REVISION_BRIEF.md` § <RELEVANT_SECTION>.
4. `CLAUDE.md` for orchestration discipline.

**YOUR DROPLET:** `<DROPLET_ID>` — `<DROPLET_TITLE>`.
- Source spec: `<SOURCE_THEME_PLAN>.md` § "<DROPLET_HEADING>".
- Files to modify: `<FILES>` (per spec).
- Packages: `<PACKAGES>` (per spec).
- Acceptance: see source spec; the spec is authoritative.
- Test scenarios: implement every entry in the source spec's table-driven scenarios.
- Falsification mitigations: pre-empt every mitigation listed in the source spec.

**HARD RULES:**
- Builders run `model: opus` (already set on this spawn).
- DO NOT commit. DO NOT push. Orchestrator drives commits AFTER QA pair returns green (per F.7-CORE REV-13).
- NEVER raw `go test` / `go build` / `go vet` / `mage install`. Always `mage <target>` (discover via `mage -l`).
- Single-line conventional commits ≤72 chars (orchestrator uses this format when committing your work).
- Append `## Droplet <DROPLET_ID> — Round <N>` section to `workflow/drop_4c_5/BUILDER_WORKLOG.md` documenting files touched + targets run + design notes + Hylla feedback (or "None — Hylla unused this droplet").
- Set droplet `state: in_progress` at start, `state: done` at end. Mutate the source theme PLAN MD's droplet row directly.

**Render your response beginning with a `# Section 0 — SEMI-FORMAL REASONING` block containing `## Proposal`, `## QA Proof`, `## QA Falsification`, `## Convergence` passes. Each pass uses the 5-field certificate (Premises / Evidence / Trace or cases / Conclusion / Unknowns) where applicable. Convergence must declare (a) QA Falsification found no unmitigated counterexample, (b) QA Proof confirmed evidence completeness, (c) remaining Unknowns are routed back to the orchestrator. Section 0 stays in your response only — NEVER write Section 0 into PLAN/WORKLOG/QA artifacts.**

**Tillsyn-flow output style** for your final response: numbered sections + addressable bullets + TL;DR.
```

QA spawn prompts mirror this shape with the role swapped to qa-proof or qa-falsification, target output paths swapped to `BUILDER_QA_PROOF.md` / `BUILDER_QA_FALSIFICATION.md`, and instructions to NOT edit production code.
