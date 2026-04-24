# DROP_1_75 — Closeout

- **Closed:** 2026-04-20
- **Final commit:** `b8580db`
- **CI run:** https://github.com/evanmschultz/tillsyn/actions/runs/24703853607

Two-phase project: this file stages content on the drop branch pre-merge. STEWARD reads it on `main/` post-merge and splices into `LEDGER.md`, `REFINEMENTS.md`, `HYLLA_FEEDBACK.md`, `WIKI_CHANGELOG.md`, `HYLLA_REFINEMENTS.md`, and `WIKI.md`.

## Code-Understanding Index Feedback Aggregation

**Consolidated entry to append to `HYLLA_FEEDBACK.md`:**

- `DROP_1_75_KIND_COLLAPSE: no misses across 15 units and 45 commits.`

Rationale (for reviewer, not for the ledger line): every per-unit and per-QA `## Hylla Feedback` section across `BUILDER_WORKLOG.md` + `BUILDER_QA_PROOF.md` + `BUILDER_QA_FALSIFICATION.md` reported `None — Hylla answered everything needed.` or `N/A`. The drop's work was heavily rename/deletion-shaped: every edited file fell inside the post-last-ingest staging window where Hylla is stale per project rule #2, making `git diff` + `rg` + `Grep` + `mage` gates authoritative. Unit 1.14 was non-Go (SQL) and Hylla indexes Go only today. No query was forced into a fallback path because no query was semantically appropriate — the question shapes were lexical ("has-this-identifier-disappeared-from-these-files") not semantic ("who-calls-this-function").

## Refinements

Append to `REFINEMENTS.md`:

- **R-1.75.1 (tooling — repeat).** `go-planning-agent` lacks `Edit` / `Write` tool permissions in filesystem-MD coordination mode; planners had to emit full-PLAN.md payloads in their response for the orchestrator to paste in. The workflow should formalize the payload-handoff pattern (planner outputs `<<<PLAN.md>>>...<<<END>>>` delimiters; orchestrator extracts and writes) OR grant `go-planning-agent` Edit/Write against `workflow/**` only. Already tracked in memory `project_drop_1_75_refinements_raised.md`.
- **R-1.75.2 (grep surface).** PLAN §1.15 end-state invariant grep pattern conflates live-API drift with drop-scope narrative prose. Drop 1.75 patched §1.15 with per-file excludes for `CLAUDE.md` / `DROP_1_75_ORCH_PROMPT.md` / `AGENT_CASCADE_DESIGN.md`. Future drops should either (a) tighten the pattern to `template_library_id` (the live-API form) and drop the broader `template_librar` substring, (b) split the invariant into two greps (one code-scoped `--glob='*.go' --glob='*.sql'`; one docs-scoped against README.md only), or (c) add a generic `drop-descriptor-prose-allowlist` mechanism.
- **R-1.75.3 (README template-library prose).** `README.md` lines 273–382 still carry an extensive "Template-library operator examples" section describing `till template ...` CLI commands, `till.template` MCP operations, template-binding lifecycle prose, and project reapply/migration flows — all dead post-1.75. Out of scope for the §1.15 invariant-unblock fix (narrow to the 3 `template_library_id` hits). A dedicated docs-cleanup refinement drop should excise the whole section and replace it with a brief note that the template system will return in a later drop.
- **R-1.75.4 (CLAUDE.md pre-drop-rule staleness).** `CLAUDE.md` § "Pre-Drop-1.75 Creation Rule (Current HEAD)" is a block describing pre-drop state; now that Drop 1.75 ships, that section is historical. STEWARD post-merge should either delete the section or rename it "Post-Drop-1.75 Creation Rule" with updated prose reflecting the `{project, actionItem}` end state.
- **R-1.75.5 (orphan-via-collapse remnants).** Per PLAN.md § Scope bullet 8, five code sites are intentionally left as dead/partial post-collapse for a future refinement drop (memory rule `feedback_orphan_via_collapse_defer_refinement.md`). Sites: (1) `internal/domain/kind.go:22-28` `KindAppliesTo` constants — `Phase`/`Subtask` naturally unreachable, `Project`/`Branch`/`ActionItem` live; (2) `internal/domain/workitem.go:35-44` `WorkKind` variants — `KindSubtask`/`KindPhase`/`KindDecision`/`KindNote` naturally unreachable (post-rename the type moved to `kind.go`, but the dead constants still ship there); (3) `internal/app/kind_capability.go:409-423` `capabilityScopeTypeForActionItem` — `Branch` arm is runtime-live per the auth-path quirk, `Phase`/`Subtask` arms naturally unreachable; (4) `internal/domain/auth_request.go:43-49` `AuthRequestPathKind` constants — all live; (5) `internal/domain/kind.go:57-73` `KindDefinition.Template` + `KindTemplate` + `validateKindTemplateExpansion` + `normalizeKindTemplate` + `ErrInvalidKindTemplate` sentinel — naturally unreachable (kind_catalog bakes empty `auto_create_children`). A Hylla-assisted refinement drop should delete the naturally-unreachable rows and keep the runtime-live arms.
- **R-1.75.6 (commit-subject cap).** One commit this drop (`b8580db`) landed at 77 chars in the subject line, 5 over the ~72 soft cap. Tooling could warn or hard-block at commit time rather than requiring post-hoc log review.

## F3 Reminder — project_allowed_kinds Decision (SURFACE BEFORE DEV RUNS drops-rewrite.sql)

Per memory rule `project_drop_1_75_unit_1_14_f3_decision.md`, when dev runs `scripts/drops-rewrite.sql` against `~/.tillsyn/tillsyn.db` at drop-end, re-surface the three F3 options and the chosen one:

- **Option A (chosen):** assert-only. Phase 7 post-run assertion checks that `project_allowed_kinds` contains no orphan kind rows; if violated, manual cleanup. No DDL changes to the table itself.
- **Option B (rejected):** wipe `project_allowed_kinds` to an empty set on rewrite; re-seed from code on next binary start. More aggressive; discarded because project customizations would be lost.
- **Option C (rejected):** rewrite each row's `kind_id` column in-place from orphan → `actionItem`. Discarded because it would collapse genuine customization diversity into a single row.

Dev explicitly asked for the reminder at run time. If dev wants to switch to Option B or C, the rewrite needs a small PHASE-8-style block added before the assertions fire.

## Ledger Entry

Append to `LEDGER.md`:

- **Drop:** `DROP_1_75_KIND_COLLAPSE`
- **Closed:** 2026-04-20
- **Branch:** `drop/1.75`
- **Final commit:** `b8580db`
- **CI run:** https://github.com/evanmschultz/tillsyn/actions/runs/24703853607 (success: `ci (macos-latest)` + `release snapshot check`)
- **Droplets (units):** 15 (1.1 through 1.15), all passed QA before commit.
- **Commits on branch:** 45 (9 plan-QA rounds + 12 code refactors + 24 MD scaffolding/closeout).
- **Files touched:** 81 total (+5023 / -17324 LOC net).
  - Go: 63 files (+1145 / -16195) — deletion-heavy (template_libraries excision, `seedDefaultKindCatalog` deletion, legacy `tasks`-table excision).
  - SQL: 1 file (`scripts/drops-rewrite.sql`, +181 / -243) — full rewrite to schema-only collapse.
  - MD / other: 17 files (scaffolding, plan, worklog, QA, closeout, refinement markers).
- **Packages touched:** 8 (`internal/domain`, `internal/app`, `internal/adapters/storage/sqlite`, `internal/adapters/server/mcpapi`, `internal/adapters/server/common`, `internal/adapters/server/httpapi`, `internal/tui`, `cmd/till`).
- **Plan-QA rounds:** 9 (Round 1 pass-with-revisions, Rounds 2–8 incremental tightening against Hylla/LSP/Context7 evidence, Round 9 pass-clean).
- **Build-QA rounds:** 15 build-QA proof + 15 build-QA falsification rounds, one of each per unit; several units needed Round-2 respawn after partial-edit recovery from API-limit interruptions on units 1.11 + 1.12.
- **Coverage (drop-end `mage ci`):** 1259 tests / 20 packages / 0 failures / min 70.6% on `internal/tui`.
- **Description:** Collapsed `kind_catalog` to `{project, actionItem}`. Excised the `template_libraries` / `template_binding` / `node_contract_snapshots` subsystem across domain, app, MCP, CLI, TUI. Renamed Go identifiers `WorkKind → Kind`, `WorkItemID → ActionItemID`, `task.go → action_item.go`. Dropped `projects.kind` column via native SQLite `DROP COLUMN`. Deleted legacy `tasks` table, `bridgeLegacyActionItemsToWorkItems`, and `migratePhaseScopeContract`. Rewrote `scripts/drops-rewrite.sql` to a schema-only 7-phase collapse with CHECK-on-TEMP-TABLE assertion idiom. Deferred orphan-via-collapse cleanup of five sites to a future Hylla-assisted refinement drop per the orphan-via-collapse doctrine. This drop unblocks the post-Drop-2 cascade tree where `metadata.role` replaces kind-slug role encoding.

## Wiki Changelog

Append one-liner to `WIKI_CHANGELOG.md`:

- `DROP_1_75_KIND_COLLAPSE: kind_catalog collapsed to {project, actionItem}. template_libraries excised. WorkKind → Kind + task.go → action_item.go. projects.kind column dropped. Legacy tasks table deleted. scripts/drops-rewrite.sql rewrites the dev DB to match.`

**WIKI.md updates (STEWARD-applied post-merge):**

- § "Creation Rule" (if such section exists) — update to reflect that every plan item under a project is now created with `kind='actionItem', scope='actionItem'`; the broader kind enumeration (`build-actionItem`, `subtask`, `qa-check`, etc.) is dead. Role lives in description prose (`Role: builder`, `Role: qa-proof`, etc.) until Drop 2 promotes it to `metadata.role`.
- § "Schema" — remove any reference to `projects.kind`, `template_libraries`, `template_node_templates`, `template_child_rules`, `project_template_bindings`, `node_contract_snapshots`, and the legacy `tasks` table.
- § "Template System" (if such section exists) — replace with a short note that the template subsystem returns in a future drop with a re-designed contract (no `kind_catalog` coupling).

## Code-Understanding Index Ingest (DEFERRED TO ORCHESTRATOR POST-PR-MERGE)

Not yet triggered — pending PR merge + dev validation (TUI + MCP spot check + dev-run of `scripts/drops-rewrite.sql`). Will record here when orchestrator runs it.

- **Triggered:** TBD
- **Mode:** full_enrichment
- **Source:** `github.com/evanmschultz/tillsyn@main`
- **Result:** TBD

## Steward Post-Merge Checklist

- Read `workflow/drop_1_75/CLOSEOUT.md` on `main/`.
- Splice "Ledger Entry" into `LEDGER.md`.
- Splice "Refinements" (R-1.75.1 through R-1.75.6) into `REFINEMENTS.md`.
- Splice "Hylla Feedback Aggregation" one-liner into `HYLLA_FEEDBACK.md`.
- Splice "Wiki Changelog" one-liner into `WIKI_CHANGELOG.md`.
- Apply "WIKI.md updates" in place.
- Drop-orch deletes remote `drop/1.75` branch + local branch after merge; STEWARD runs `git worktree remove drop/1.75` from bare root.
- Record the post-merge ingest run id in `LEDGER.md` entry when the orchestrator completes the reingest.
