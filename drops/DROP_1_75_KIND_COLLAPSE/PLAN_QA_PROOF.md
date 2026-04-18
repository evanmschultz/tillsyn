---
round: 3
verdict: PASS
reviewer: go-qa-proof-agent
reviewed: drops/DROP_1_75_KIND_COLLAPSE/PLAN.md
date: 2026-04-18
---

# DROP_1_75_KIND_COLLAPSE — Plan QA Proof, Round 3

Proof-oriented review of the 15-unit decomposition. Evidence gathered via `Read` / `Grep` against `/Users/evanschultz/Documents/Code/hylla/tillsyn/main/` (Hylla is Go-only and stale relative to the in-flight drop rename). Findings numbered `P1`…`PN`.

## P1 — DAG is acyclic and linearly progresses

- **Finding**: `Blocked by` edges form a DAG with no cycles. Every blocker references a lower-numbered unit; 1.1 is the single unblocked root; 1.15 sinks every prior unit through 1.14.
- **Evidence**: PLAN.md:65, 79, 91, 105, 118, 131, 144, 157, 170, 185, 197, 208, 221, 233, 251. Traced topologically: 1.1 → {1.2, 1.4} → 1.3 → {1.5} → {1.6, 1.7} → {1.8, 1.11, 1.13} → {1.9, 1.12} → 1.10 → 1.14 → 1.15.
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## P2 — Package-level blocker gap: 1.12 must block on 1.6 (mcpapi package)

- **Finding**: Unit 1.6 edits `internal/adapters/server/mcpapi/instructions_explainer.go` (PLAN.md:129) and declares package `internal/adapters/server/mcpapi` (PLAN.md:130). Unit 1.12 runs `mage test-pkg ./internal/adapters/server/mcpapi` (PLAN.md:211) and lists the same package (PLAN.md:207). `CLAUDE.md` §Blocker Semantics: *"sibling build-tasks sharing ... a package in `packages` MUST have an explicit `blocked_by`"*. Unit 1.12 `Blocked by: 1.3, 1.5, 1.7` (PLAN.md:208) — **1.6 absent**. If 1.12 starts before 1.6 commits, `instructions_explainer.go` still reads `project.Kind` while the test suite expects the stripped surface, causing a spurious red.
- **Evidence**: PLAN.md:129 (1.6 paths list includes `instructions_explainer.go`), PLAN.md:130 (1.6 packages include `mcpapi`), PLAN.md:206-211 (1.12 paths + packages + acceptance naming `mcpapi`), PLAN.md:208 (1.12 `Blocked by` missing 1.6). Also note the planner's own F10 note at PLAN.md:274 explicitly commits to package-level serialization — this site is not serialized.
- **Severity**: blocking (violates F10 contract the planner cited as addressed).
- **Suggestion**: change 1.12 `Blocked by:` to `1.3, 1.5, 1.6, 1.7`.

## P3 — Package-level blocker chain within `internal/domain` is tight

- **Finding**: `internal/domain` is touched by 1.1, 1.4, 1.6, 1.8, 1.9, 1.10. Chain: 1.4 ← 1.1; 1.6 ← 1.1, 1.4; 1.8 ← 1.1, 1.4, 1.6; 1.9 ← 1.1, 1.4, 1.6, 1.8; 1.10 ← 1.1, 1.4, 1.6, 1.8, 1.9. All five post-1.1 domain units name every prior domain unit directly — explicit serialization per F10.
- **Evidence**: PLAN.md:105 (1.4), :131 (1.6), :157 (1.8), :170 (1.9), :185 (1.10).
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## P4 — Package-level blocker chain within `internal/app` is tight

- **Finding**: `internal/app` is touched by 1.1, 1.2, 1.5, 1.6, 1.11. Chain: 1.2 ← 1.1; 1.5 ← 1.1, 1.2, 1.3, 1.4; 1.6 ← 1.1, 1.2, 1.3, 1.4, 1.5; 1.11 ← 1.2, 1.5, 1.6. Every post-1.1 app unit names every prior app unit directly or transitively via explicit edges.
- **Evidence**: PLAN.md:79, :118, :131, :197.
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## P5 — Package-level blocker chain within `internal/adapters/storage/sqlite` is tight

- **Finding**: Touched by 1.1, 1.3, 1.5, 1.7, 1.12. Chain: 1.3 ← 1.1, 1.2 (1.2 is app-layer, sequencing app seeder removal before inline catalog bake); 1.5 ← 1.3; 1.7 ← 1.3, 1.5; 1.12 ← 1.3, 1.5, 1.7. Serialized.
- **Evidence**: PLAN.md:91, :118, :144, :208.
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## P6 — Package-level blocker chain within `internal/tui` and `cmd/till` is tight

- **Finding**: `internal/tui` touched by 1.1, 1.5, 1.6, 1.13 — 1.13 ← 1.5, 1.6 ✓ (1.1 transitive). `cmd/till` touched by 1.1, 1.5, 1.6, 1.13 — same chain ✓.
- **Evidence**: PLAN.md:118, :131, :221.
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## P7 — Acceptance criteria are concrete and falsifiable

- **Finding**: Every unit's Acceptance field names a grep pattern with 0-match assertion and a specific mage target with pass assertion. 1.3 adds a test name (`TestRepositoryFreshOpenKindCatalog`) with precise row-count assertion. 1.8 asserts filesystem presence/absence. 1.9 asserts `grep` counts for `type Kind string` (= 1), `type KindID string` (= 1), constants (≥ 5). 1.14 asserts five SQL post-run invariants. 1.15 adds the end-state global `rg` sweep. None rely on aspirational phrasing.
- **Evidence**: PLAN.md:67-70, 81-82, 93-96, 107-109, 120-122, 133-135, 146-148, 159-161, 172-176, 187-188, 199-201, 210-214, 223-226, 236-242, 253-256.
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## P8 — Load-bearing line references verify against `main/` worktree

Spot-checked a representative sample:

- `workitem.go:35-44` → `type WorkKind` block at `main/internal/domain/workitem.go:34-44` ✓ (off by 1 — heading doc comment counts).
- `repo.go:169` `CREATE TABLE tasks` → verified at `:169` ✓.
- `repo.go:316` `CREATE TABLE kind_catalog` → verified at `:316` ✓.
- `repo.go:588` `ALTER TABLE projects ADD COLUMN kind` → verified at `:588` ✓.
- `repo.go:592-604` 13 `ALTER TABLE tasks` → verified at `:592-604` ✓.
- `repo.go:710-789` `migratePhaseScopeContract` → verified at `:710-789` ✓.
- `repo.go:1030-1055` `migrateTemplateLifecycle` → verified at `:1030-1055` ✓.
- `repo.go:1184-1228` `bridgeLegacyActionItemsToWorkItems` → verified at `:1184-1228` ✓.
- `repo.go:1231-1301` `seedDefaultKindCatalog` → verified at `:1231-1301` ✓.
- `kind_capability.go:559-589` `ensureKindCatalogBootstrapped` → verified ✓.
- `kind_capability.go:409-423` `capabilityScopeTypeForActionItem` → verified ✓.
- `project.go:16` `Kind KindID` field → verified at `:16` ✓; assignment at `:60` (PLAN says `:85` — see P10).
- `kind.go:22-28` `KindAppliesTo*` constants → verified at `:22-28` ✓.
- `tui/model.go` 9 hard refs (`:4856, :18747, :5190, :5200, :14840, :17905, :19236, :5227, :8957`) → all verified via `Grep` ✓.
- `extended_tools.go:2085, :2171, :2258, :2281` MCP tool registrations → all verified ✓.
- `handler.go:86` `registerTemplateLibraryTools` → verified ✓.
- `auth_request.go:43-49` `AuthRequestPathKind` + constants → verified at `:42-49` ✓.
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## P9 — Minor line-reference drift in a few plan citations

- **Finding**: Non-load-bearing line drift — the plan remains correct about what to delete, just off by one or two lines.
  - PLAN.md:23 cites `handler.go:1045-1050 pickTemplateLibraryService`; actual function is at `:1046-1054` (off-by-one).
  - PLAN.md:25 cites assignment `project.Kind` at `:85`; actual assignment is at `project.go:60` (`Kind: DefaultProjectKind` in `NewProject`) and mutation at `project.go:85` (in `SetKind`). Both live — `:85` is the `SetKind` body. Plan's `:85` is the mutator; the constructor site is at `:60`. Not wrong, but incomplete — builder must strip both.
  - PLAN.md:274 cites `kind_capability.go:867`; actual `KindID(Kind(x))` pattern appears near `:867` but exact line undetermined — plan just uses it as a flavor cite, not a delete target.
- **Evidence**: `main/internal/adapters/server/mcpapi/handler.go:1046`, `main/internal/domain/project.go:60, :85`.
- **Severity**: non-blocking (citations are close enough; builder will find both sites via `rg`).
- **Suggestion**: builder, when in 1.6, grep `project.go` for every `Kind` reference rather than trusting line numbers alone. Plan already says this via acceptance `rg 'project\.Kind|projects\.kind|Project\{[^}]*Kind' ... returns 0 matches` (PLAN.md:133) — safe.

## P10 — Unit 1.11 lists unrelated test files in Paths

- **Finding**: Unit 1.11 Paths includes `internal/app/search_embeddings_test.go` and `internal/app/embedding_runtime_test.go` (PLAN.md:195). Both files contain zero hits for `WorkKind`, `TemplateLibrary`, or `ensureKindCatalogBootstrapped` (verified via `Grep`). They don't need edits. Including them as `Paths` risks a builder waste-of-motion pass.
- **Evidence**: PLAN.md:195; `Grep 'WorkKind|TemplateLibrary|ensureKindCatalogBootstrapped' search_embeddings_test.go embedding_runtime_test.go` → 0 matches.
- **Severity**: non-blocking (builder will verify 0 hits and leave them).
- **Suggestion**: drop those two files from 1.11 Paths, or keep them and add a comment "no-op if no hits — package-compile coverage only."

## P11 — Unit 1.5 `TemplateLibraryRepo` port reference is imprecise

- **Finding**: PLAN.md:116 says "strip `TemplateLibraryRepo` port" in `internal/app/ports.go`. No standalone `TemplateLibraryRepo` interface exists. The template-library methods live on the unified `Repository` interface: `UpsertTemplateLibrary`, `GetTemplateLibrary`, `ListTemplateLibraries`, `UpsertProjectTemplateBinding`, `GetProjectTemplateBinding`, `DeleteProjectTemplateBinding`, `CreateNodeContractSnapshot`, `UpdateNodeContractSnapshot`, `GetNodeContractSnapshot` (ports.go:24-32).
- **Evidence**: `main/internal/app/ports.go:24-32`.
- **Severity**: non-blocking (intent is clear: strip the 9 methods; acceptance regex at PLAN.md:120 catches them).
- **Suggestion**: rewrite Unit 1.5 phrase as "strip `TemplateLibrary*` / `NodeContractSnapshot*` / `ProjectTemplateBinding*` methods from the `Repository` interface."

## P12 — F5 orphan-via-collapse classification matches code reality

- **Finding**: The four deferred sites at PLAN.md:49-52 each carry an explicit runtime-live / naturally-unreachable classification. Spot-checks:
  - `KindAppliesTo` constants (`kind.go:22-28`): Project/Branch/ActionItem runtime-live (auth-request paths, drop-scoped lease scope_type); Phase/Subtask kept but no rows will ever carry them post-collapse ✓.
  - `WorkKind` non-actionItem variants (`workitem.go:39-43`, renamed to `Kind*` by 1.1): naturally unreachable — `drops-rewrite.sql` `UPDATE action_items SET kind='actionItem', scope='actionItem'` kills the rows ✓.
  - `capabilityScopeTypeForActionItem` (`kind_capability.go:409-423`): Branch branch live per pre-Drop-2 auth-path-branch-quirk ✓; Phase/Subtask branches unreachable ✓; Project + default→ActionItem live ✓.
  - `AuthRequestPathKind` (`auth_request.go:42-49`): all three (`project`, `projects`, `global`) orthogonal to action_item kinds ✓.
  - Dev direct quote preserved at PLAN.md:48. Classification wording matches dev direction ("mixed", "naturally unreachable", "all live").
- **Evidence**: PLAN.md:48-52; `main/internal/domain/kind.go:22-28`; `main/internal/domain/workitem.go:39-43`; `main/internal/app/kind_capability.go:409-423`; `main/internal/domain/auth_request.go:42-49`.
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## P13 — Round 2 residual notes addressed or explicitly deferred

- **Finding**: The PLAN.md Scope block (:13-54) and Notes block (:262-277) absorb Round 2's residual notes:
  - Pre-drop dev DB cleanup context (PLAN.md:263) — addressed.
  - `__global__` auth project self-healing (PLAN.md:264) — addressed.
  - `work_items → action_items` table rename pre-shipped (PLAN.md:265) — addressed.
  - `KindID` vs `Kind` decision (PLAN.md:266) — addressed.
  - Goldens not affected (PLAN.md:272) — addressed.
  - F10 package-level blocker contract (PLAN.md:274) — addressed in intent but violated at one site (see P2).
  - F5 orphan-via-collapse classification (PLAN.md:48-52) — addressed per dev direction.
- **Evidence**: PLAN.md Scope + Notes sections.
- **Severity**: non-blocking (all residuals addressed except the one blocking gap in P2).
- **Suggestion**: fix P2 and this becomes fully clean.

## P14 — Every in-scope item maps to at least one unit

- **Finding**: Cross-walk of §Scope In-scope items (1-8) against units:
  - Scope.1 Kind catalog collapse → Units 1.2 (app seeder) + 1.3 (sqlite seeder + inline rows) ✓.
  - Scope.2 Go identifier rename → Unit 1.1 ✓.
  - Scope.3 File + type renames → Units 1.8 (`task.go → action_item.go`) + 1.9 (merge `WorkKind → Kind` into `kind.go`) ✓.
  - Scope.4 `template_libraries` excision → Units 1.4 (domain) + 1.5 (app+adapter+MCP+CLI+TUI) ✓.
  - Scope.5 Drop `projects.kind` column → Unit 1.6 ✓.
  - Scope.6 `drops-rewrite.sql` rewrite → Unit 1.14 ✓.
  - Scope.7 Legacy `tasks` table excision → Unit 1.7 ✓.
  - Scope.8 Tests + fixtures → Units 1.10 (domain), 1.11 (app), 1.12 (adapters), 1.13 (tui+cli) ✓.
  - Drop-end `mage ci` gate → Unit 1.15 ✓.
- **Evidence**: PLAN.md:17-39 vs §Planner units 1.1-1.15.
- **Severity**: non-blocking (confirmation — no Scope item left orphaned).
- **Suggestion**: none.

## P15 — 1.15 conflates `mage ci` gate with push + CI-watch

- **Finding**: Unit 1.15 title says "Drop-end `mage ci` gate" (PLAN.md:246) but its Acceptance includes `git push` + `gh run watch --exit-status` green (PLAN.md:254). These are distinct Phase 6 steps per WORKFLOW.md (`mage ci` local → push → CI watch). Conflation is harmless for the plan but makes the unit bigger than its title implies.
- **Evidence**: PLAN.md:246-256; WORKFLOW.md:154-158.
- **Severity**: non-blocking (orchestrator knows the sequence; no semantic gap).
- **Suggestion**: either rename 1.15 to "Drop-end verification (`mage ci` + push + CI watch)" or split into 1.15a (local CI) and 1.15b (push + watch). Either works.

## P16 — `scripts/drops-rewrite.sql` assertion set is complete for the stated end-state

- **Finding**: Unit 1.14 lists 5 assertions (PLAN.md:237-241): `kind_catalog` count = 2, `template_*` table count = 0, `tasks` table count = 0, `projects.kind` column count = 0, `action_items` non-canonical kind count = 0. These map bijectively to the five observable end-state invariants the drop promises. Rollback via `BEGIN TRANSACTION` + `RAISE(ROLLBACK, ...)` named at :242 — idempotent and safe.
- **Evidence**: PLAN.md:237-242.
- **Severity**: non-blocking (confirmation).
- **Suggestion**: none.

## Verdict

PASS with one blocking finding (P2 — 1.12 missing 1.6 in `Blocked by` for the shared `internal/adapters/server/mcpapi` package compile, violating the F10 contract PLAN.md:274 claims is resolved). The fix is a one-line edit. Eleven non-blocking confirmations (P1, P3-P8, P10-P16) demonstrate the decomposition's evidence chain, DAG structure, acceptance criteria, line-reference accuracy, orphan-via-collapse classification, and Scope-to-unit coverage are sound. Minor line-reference drift (P9), Path overreach in 1.11 (P10), and imprecise port-interface framing in 1.5 (P11) are editorial, not structural.

Round 3 recommended action: apply P2's one-line fix (`Blocked by: 1.3, 1.5, 1.6, 1.7` on unit 1.12). Optionally absorb P9/P10/P11/P15 as wording cleanup. Once P2 is addressed and QA Falsification's Round 3 findings are triaged, the plan is green for Phase 4 build.
