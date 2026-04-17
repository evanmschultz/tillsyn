-- =============================================================================
-- drops-rewrite.sql — one-shot Tillsyn DB migration to drops-all-the-way-down.
--
-- Scope:
--   1. Delete TILLSYN-OLD (a0cfbf87-...) and HYLLA_OLD (19c997f5-...).
--   2. Collapse all non-project work_item kinds in keeper projects to `drop`.
--   3. Collapse all non-project work_item scopes to `task`.
--   4. Normalize project.kind to `project` (go-project, astro-solid-project → project).
--   5. Hydrate metadata.role on every work_item from its previous kind.
--   6. Add `drop` to kind_catalog; delete the 18 legacy kinds.
--   7. Delete all template_libraries (templates are dead post-collapse).
--   8. Delete stale node_contract_snapshots.
--   9. Drop the empty legacy `tasks` table.
--
-- NOT touched intentionally:
--   - lifecycle_state values (`done`/`progress`/`todo`/`archived` stay as-is).
--     Renaming `done → complete` is a runtime state-machine change and must ship
--     with Drop-1 Go code, not here, or the running `till` binary breaks.
--   - autent_* tables (no project_id FK; 0 grants reference doomed projects,
--     0 active sessions, payloads are opaque BLOBs — safe to leave).
--   - archived work_items in keeper projects (0 in keepers per recon; all
--     archived rows are in doomed projects and die via CASCADE).
--
-- Run model:
--   - Dev-run only. Never invoked by CI. Never called by Go code.
--   - One-shot; once committed, the script is historical reference only.
--   - Dev MUST back up `~/.tillsyn/tillsyn.db` before running.
--   - Intended to run AFTER Drop 2 Go code ships (which renames the Task
--     domain entity → Drop and removes legacy kind lookups). Running against
--     an older binary will leave the binary able to read the DB (kind/scope
--     are plain TEXT), but the binary won't understand `kind='drop'` until
--     Drop 2 ships.
--
-- How to run:
--   cp ~/.tillsyn/tillsyn.db ~/.tillsyn/tillsyn.db.pre-drops-rewrite
--   sqlite3 ~/.tillsyn/tillsyn.db < scripts/drops-rewrite.sql
--
-- On assertion failure the transaction rolls back; the DB is unchanged.
-- =============================================================================

PRAGMA foreign_keys = ON;

BEGIN TRANSACTION;

-- -----------------------------------------------------------------------------
-- PHASE 0 — pre-flight snapshot (informational; printed by sqlite3)
-- -----------------------------------------------------------------------------
SELECT 'pre:projects'      AS label, COUNT(*) AS n FROM projects;
SELECT 'pre:work_items'    AS label, COUNT(*) AS n FROM work_items;
SELECT 'pre:kind_catalog'  AS label, COUNT(*) AS n FROM kind_catalog;
SELECT 'pre:templates'     AS label, COUNT(*) AS n FROM template_libraries;
SELECT 'pre:snapshots'     AS label, COUNT(*) AS n FROM node_contract_snapshots;
SELECT 'pre:tasks_legacy'  AS label, COUNT(*) AS n FROM tasks;

-- -----------------------------------------------------------------------------
-- PHASE 1 — delete doomed projects (cascades 16 project-scoped tables)
--
-- Cascade coverage: work_items, tasks, columns_v1, project_allowed_kinds,
-- project_template_bindings, attention_items, comments, handoffs, auth_requests,
-- capability_leases, change_events, embedding_documents, embedding_jobs,
-- task_embeddings, node_contract_snapshots (and its editor/completer kind edges).
-- -----------------------------------------------------------------------------
DELETE FROM projects WHERE id IN (
    'a0cfbf87-b470-45f9-aae0-4aa236b56ed9',  -- TILLSYN-OLD
    '19c997f5-0c17-4a36-bd94-ebc06ff7f1cf'   -- HYLLA_OLD
);

-- -----------------------------------------------------------------------------
-- PHASE 2 — unbind Sjal from astro-solid template
-- Without this, PHASE 3 fails (project_template_bindings.library_id is RESTRICT).
-- -----------------------------------------------------------------------------
DELETE FROM project_template_bindings
WHERE project_id = '7823610e-57b5-43d7-bd13-0d861db74bb1';

-- -----------------------------------------------------------------------------
-- PHASE 3 — wipe all template libraries
-- Cascades: template_node_templates, template_child_rules,
-- template_child_rule_editor_kinds, template_child_rule_completer_kinds.
-- Frees the RESTRICT FK from template_*.node_kind_id/child_kind_id → kind_catalog,
-- so PHASE 8 can clear legacy kinds.
-- -----------------------------------------------------------------------------
DELETE FROM template_libraries;

-- -----------------------------------------------------------------------------
-- PHASE 4 — delete stale node_contract_snapshots
-- All remaining snapshots (Sjal's 387) reference now-deleted
-- template_libraries + template_node_templates + template_child_rules by ID.
-- Cascades node_contract_editor_kinds + node_contract_completer_kinds.
-- -----------------------------------------------------------------------------
DELETE FROM node_contract_snapshots;

-- -----------------------------------------------------------------------------
-- PHASE 5 — introduce the `drop` kind
-- applies_to_json = ["task"]: drops live at task scope.
-- allowed_parent_scopes_json = ["project","task"]: drops attach directly under
-- a project or nested under another drop.
-- -----------------------------------------------------------------------------
INSERT INTO kind_catalog (
    id,
    display_name,
    description_markdown,
    applies_to_json,
    allowed_parent_scopes_json,
    payload_schema_json,
    template_json,
    created_at,
    updated_at
) VALUES (
    'drop',
    'Drop',
    'Universal work unit at every level below a project. Drops nest infinitely. Role lives on metadata.role.',
    '["task"]',
    '["project","task"]',
    '',
    '{}',
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
)
ON CONFLICT(id) DO NOTHING;

-- -----------------------------------------------------------------------------
-- PHASE 6 — hydrate metadata.role BEFORE collapsing kind
-- Order matters: role inference reads the old `kind` value.
-- Uses json_set on metadata_json (empty string safety via NULLIF+COALESCE).
-- -----------------------------------------------------------------------------

-- 6a — builder
UPDATE work_items
SET metadata_json = json_set(COALESCE(NULLIF(metadata_json, ''), '{}'), '$.role', 'builder')
WHERE kind = 'build-task';

-- 6b — qa-falsification (title heuristic: substring match on FALSIFICATION)
UPDATE work_items
SET metadata_json = json_set(COALESCE(NULLIF(metadata_json, ''), '{}'), '$.role', 'qa-falsification')
WHERE kind = 'qa-check'
  AND UPPER(title) LIKE '%FALSIFICATION%';

-- 6c — qa-proof (explicit PROOF matches, plus the single ambiguous outlier
-- in TILLSYN: "18.5 QA — STATE + GAPS ASSESSMENT (READ-ONLY)" defaults here)
UPDATE work_items
SET metadata_json = json_set(COALESCE(NULLIF(metadata_json, ''), '{}'), '$.role', 'qa-proof')
WHERE kind = 'qa-check'
  AND UPPER(title) NOT LIKE '%FALSIFICATION%';

-- 6d — qa-a11y (FE accessibility check)
UPDATE work_items
SET metadata_json = json_set(COALESCE(NULLIF(metadata_json, ''), '{}'), '$.role', 'qa-a11y')
WHERE kind = 'a11y-check';

-- 6e — qa-visual (FE visual QA)
UPDATE work_items
SET metadata_json = json_set(COALESCE(NULLIF(metadata_json, ''), '{}'), '$.role', 'qa-visual')
WHERE kind = 'visual-qa';

-- 6f — design (FE design review)
UPDATE work_items
SET metadata_json = json_set(COALESCE(NULLIF(metadata_json, ''), '{}'), '$.role', 'design')
WHERE kind = 'design-review';

-- 6g — commit (commit + reingest gate)
UPDATE work_items
SET metadata_json = json_set(COALESCE(NULLIF(metadata_json, ''), '{}'), '$.role', 'commit')
WHERE kind = 'commit-and-reingest';

-- 6h — planner (legacy plan-phase)
UPDATE work_items
SET metadata_json = json_set(COALESCE(NULLIF(metadata_json, ''), '{}'), '$.role', 'planner')
WHERE kind = 'plan-phase';

-- (kinds `task`, `subtask`, `branch`, `phase`, `build-phase`, `closeout-phase`,
--  `project-setup-phase`, `branch-cleanup-phase`, `decision`, `note` get NO role
--  hydrated — role on these is ambiguous and a planner can set it later when
--  the drop is revisited.)

-- -----------------------------------------------------------------------------
-- PHASE 7 — collapse work_item kind + scope
-- Single UPDATE rewrites every row in keeper projects to drop/task.
-- (Doomed projects' rows already gone via PHASE 1 cascade.)
-- -----------------------------------------------------------------------------
UPDATE work_items
SET kind = 'drop', scope = 'task';

-- -----------------------------------------------------------------------------
-- PHASE 8 — normalize project.kind
-- go-project / astro-solid-project → project. `__global__` already `project`.
-- -----------------------------------------------------------------------------
UPDATE projects
SET kind = 'project'
WHERE kind <> 'project';

-- -----------------------------------------------------------------------------
-- PHASE 9 — clean the kind catalog
-- Remove the 18 legacy kinds. project_allowed_kinds rows pointing at them
-- cascade out via kind_catalog → project_allowed_kinds ON DELETE CASCADE.
-- Nothing else references kind_catalog at this point: templates wiped,
-- snapshots wiped, work_items.kind/projects.kind are plain TEXT without FK.
-- -----------------------------------------------------------------------------
DELETE FROM kind_catalog
WHERE id NOT IN ('project', 'drop');

-- -----------------------------------------------------------------------------
-- PHASE 10 — drop empty legacy `tasks` table
-- Superseded by `work_items`; confirmed 0 rows; no inbound FKs reference it.
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS tasks;

-- -----------------------------------------------------------------------------
-- PHASE 11 — hard assertions (abort-on-fail via CHECK constraint)
-- Each row inserted into migration_check must satisfy expected = actual;
-- any failure raises CHECK constraint error and rolls back the transaction.
-- -----------------------------------------------------------------------------
CREATE TEMP TABLE migration_check (
    label    TEXT PRIMARY KEY,
    expected INTEGER NOT NULL,
    actual   INTEGER NOT NULL,
    CHECK(expected = actual)
);

INSERT INTO migration_check (label, expected, actual) VALUES
    ('projects_total',
        4,
        (SELECT COUNT(*) FROM projects)),
    ('projects_doomed_gone',
        0,
        (SELECT COUNT(*) FROM projects
         WHERE id IN ('a0cfbf87-b470-45f9-aae0-4aa236b56ed9',
                      '19c997f5-0c17-4a36-bd94-ebc06ff7f1cf'))),
    ('projects_all_kind_project',
        0,
        (SELECT COUNT(*) FROM projects WHERE kind <> 'project')),
    ('work_items_all_drop',
        0,
        (SELECT COUNT(*) FROM work_items WHERE kind <> 'drop')),
    ('work_items_all_scope_task',
        0,
        (SELECT COUNT(*) FROM work_items WHERE scope <> 'task')),
    ('work_items_only_in_keepers',
        0,
        (SELECT COUNT(*) FROM work_items
         WHERE project_id NOT IN (
             'a5e87c34-3456-4663-9f32-df1b46929e30',  -- TILLSYN
             '7823610e-57b5-43d7-bd13-0d861db74bb1',  -- Sjal
             'cb2779e8-9167-4df9-8bab-d25ee7cfb8b4',  -- HYLLA
             '__global__'
         ))),
    ('kind_catalog_only_project_and_drop',
        2,
        (SELECT COUNT(*) FROM kind_catalog)),
    ('kind_catalog_has_drop',
        1,
        (SELECT COUNT(*) FROM kind_catalog WHERE id = 'drop')),
    ('kind_catalog_has_project',
        1,
        (SELECT COUNT(*) FROM kind_catalog WHERE id = 'project')),
    ('template_libraries_empty',
        0,
        (SELECT COUNT(*) FROM template_libraries)),
    ('template_node_templates_empty',
        0,
        (SELECT COUNT(*) FROM template_node_templates)),
    ('template_child_rules_empty',
        0,
        (SELECT COUNT(*) FROM template_child_rules)),
    ('project_template_bindings_empty',
        0,
        (SELECT COUNT(*) FROM project_template_bindings)),
    ('node_contract_snapshots_empty',
        0,
        (SELECT COUNT(*) FROM node_contract_snapshots)),
    ('project_allowed_kinds_only_valid',
        0,
        (SELECT COUNT(*) FROM project_allowed_kinds
         WHERE kind_id NOT IN ('project', 'drop')));

-- -----------------------------------------------------------------------------
-- PHASE 12 — role-hydration sanity snapshot (informational)
-- -----------------------------------------------------------------------------
SELECT
    COALESCE(json_extract(metadata_json, '$.role'), '(none)') AS role,
    COUNT(*) AS n
FROM work_items
GROUP BY role
ORDER BY n DESC;

-- -----------------------------------------------------------------------------
-- PHASE 13 — post-flight snapshot
-- -----------------------------------------------------------------------------
SELECT 'post:projects'     AS label, COUNT(*) AS n FROM projects;
SELECT 'post:work_items'   AS label, COUNT(*) AS n FROM work_items;
SELECT 'post:kind_catalog' AS label, COUNT(*) AS n FROM kind_catalog;

COMMIT;

-- -----------------------------------------------------------------------------
-- END
-- -----------------------------------------------------------------------------
