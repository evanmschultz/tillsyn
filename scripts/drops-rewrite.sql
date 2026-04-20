-- =============================================================================
-- drops-rewrite.sql — Drop 1.75 kind-collapse migration.
--
-- Scope (7 phases, all wrapped in a single transaction):
--   1. Pre-flight counts — audit starting state.
--   2. DROP TABLE the template cluster (9 tables) BEFORE kind_catalog delete so
--      RESTRICT FKs from template_*.node_kind_id / .child_kind_id → kind_catalog
--      cannot trip during Phase 3.
--   3. DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem').
--      `project_allowed_kinds.kind_id → kind_catalog(id) ON DELETE CASCADE`
--      clears allowlist rows pointing at deleted legacy kinds automatically.
--   4. ALTER TABLE projects DROP COLUMN kind — native SQLite form (3.35.0+).
--      `projects.kind` is a plain column with no PK / UNIQUE / FK / index /
--      trigger / view dependencies; the 12-step rebuild is unnecessary.
--   5. DROP TABLE tasks — legacy empty table excised in Unit 1.7.
--   6. UPDATE action_items SET kind='actionItem', scope='actionItem' — unify
--      any rows still on the old default.
--   7. Assertion block — 8 end-state invariants guarded via a TEMP TABLE with
--      CHECK(expected = actual). Any violation raises a CHECK-constraint error;
--      combined with `.bail on` at the top of the script the sqlite3 CLI exits
--      before COMMIT fires, and the transaction rolls back implicitly when the
--      connection closes without a successful COMMIT.
--
-- Why TEMP TABLE CHECK + `.bail on` instead of RAISE(ROLLBACK, ...) at top level:
--   RAISE() may only appear inside a trigger program (SQLite parser error
--   otherwise). Without `.bail on`, sqlite3's default behavior is to continue
--   past a CHECK-constraint error — the failed INSERT is reverted but the
--   transaction stays open, so the trailing COMMIT still commits every DDL/DML
--   statement that executed before the failure. `.bail on` flips the CLI to
--   abort on first error, which leaves the BEGIN-open transaction unwrapped
--   by COMMIT and SQLite rolls it back when the connection closes.
--
-- What this rewrite DROPS from the prior main-branch script (pre-drop cleanup
-- 2026-04-18 already removed every row the deleted phases would have touched):
--   - Phase 1 "delete doomed projects" — TILLSYN-OLD + HYLLA_OLD already gone.
--   - Phase 2 "unbind Sjal" — Sjal project already gone.
--   - Phase 5 "introduce `drop` kind" — replaced by `actionItem` kind (already
--     present in dev DB as of the 2026-04-18 identifier rename).
--   - Phase 6 "hydrate metadata.role" — no legacy kinds survive in action_items
--     after the pre-drop purge; all rows are already `kind='task'`.
--   - Phase 7 "collapse kind + scope" — subsumed by Phase 6 of this script.
--   - Phase 8 "normalize project.kind" — eliminated by Phase 4 DROP COLUMN.
--
-- Run model:
--   - Dev-run only. Never invoked by CI. Never called by Go code.
--   - One-shot; once committed, historical reference only.
--   - Dev MUST back up `~/.tillsyn/tillsyn.db` before running.
--   - Intended to run AFTER Drop 1.75 Go code ships (which deletes `Project.Kind`
--     and every Go-side legacy-kind reference).
--
-- How to run:
--   cp ~/.tillsyn/tillsyn.db ~/.tillsyn/tillsyn.db.pre-drop-1-75
--   sqlite3 ~/.tillsyn/tillsyn.db < scripts/drops-rewrite.sql
--
-- On assertion failure the transaction rolls back; the DB is unchanged.
--
-- F3 Option A (project_allowed_kinds residue, per PLAN §1.14):
--   Assertion 8 guards against allowlist rows referencing kinds outside
--   ('project','actionItem'). Under the current schema the DELETE at Phase 3
--   CASCADE-clears those rows, so the assertion is a diagnostic safety net
--   catching any future schema drift. If it fires, the script rolls back
--   cleanly and the dev handles the residue manually.
-- =============================================================================

-- `.bail on` makes the sqlite3 CLI abort on the first error. Combined with the
-- Phase 7 CHECK-on-TEMP-TABLE assertions, a failing invariant leaves the
-- BEGIN-open transaction without a COMMIT and SQLite rolls it back cleanly.
.bail on

PRAGMA foreign_keys = ON;

BEGIN TRANSACTION;

-- -----------------------------------------------------------------------------
-- PHASE 1 — pre-flight counts (informational; printed by sqlite3)
-- -----------------------------------------------------------------------------
SELECT 'pre:projects'                 AS label, COUNT(*) AS n FROM projects;
SELECT 'pre:action_items'             AS label, COUNT(*) AS n FROM action_items;
SELECT 'pre:kind_catalog'             AS label, COUNT(*) AS n FROM kind_catalog;
SELECT 'pre:project_allowed_kinds'    AS label, COUNT(*) AS n FROM project_allowed_kinds;
SELECT 'pre:template_libraries'       AS label, COUNT(*) AS n FROM template_libraries;
SELECT 'pre:template_node_templates'  AS label, COUNT(*) AS n FROM template_node_templates;
SELECT 'pre:template_child_rules'     AS label, COUNT(*) AS n FROM template_child_rules;
SELECT 'pre:project_template_bindings' AS label, COUNT(*) AS n FROM project_template_bindings;
SELECT 'pre:node_contract_snapshots'  AS label, COUNT(*) AS n FROM node_contract_snapshots;
SELECT 'pre:tasks_legacy'             AS label, COUNT(*) AS n FROM tasks;
SELECT 'pre:projects_kind_col'        AS label, COUNT(*) AS n FROM pragma_table_info('projects') WHERE name = 'kind';

-- -----------------------------------------------------------------------------
-- PHASE 2 — DROP TABLE the template cluster (9 tables).
--
-- Order: drop dependent tables before their parents so RESTRICT FKs don't trip.
--   1. template_child_rule_completer_kinds → template_child_rules
--   2. template_child_rule_editor_kinds    → template_child_rules
--   3. template_child_rules                → template_node_templates (ON DELETE CASCADE)
--                                           + kind_catalog (ON DELETE RESTRICT)
--   4. template_node_templates             → template_libraries (ON DELETE CASCADE)
--                                           + kind_catalog (ON DELETE RESTRICT)
--   5. project_template_bindings           → template_libraries (ON DELETE RESTRICT)
--   6. node_contract_completer_kinds       → node_contract_snapshots
--   7. node_contract_editor_kinds          → node_contract_snapshots
--   8. node_contract_snapshots             → projects + action_items (ON DELETE CASCADE)
--   9. template_libraries                  → projects (ON DELETE CASCADE)
--
-- Ordering the nine drops before Phase 3 is the Round-5 editorial contract:
-- once the template_* and project_template_bindings tables are gone, the
-- RESTRICT FKs from those tables to kind_catalog no longer exist, and Phase 3
-- can safely CASCADE through project_allowed_kinds.
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS template_child_rule_completer_kinds;
DROP TABLE IF EXISTS template_child_rule_editor_kinds;
DROP TABLE IF EXISTS template_child_rules;
DROP TABLE IF EXISTS template_node_templates;
DROP TABLE IF EXISTS project_template_bindings;
DROP TABLE IF EXISTS node_contract_completer_kinds;
DROP TABLE IF EXISTS node_contract_editor_kinds;
DROP TABLE IF EXISTS node_contract_snapshots;
DROP TABLE IF EXISTS template_libraries;

-- -----------------------------------------------------------------------------
-- PHASE 3 — clean the kind catalog.
--
-- Leave only ('project', 'actionItem'). The ON DELETE CASCADE FK from
-- project_allowed_kinds.kind_id → kind_catalog(id) clears allowlist rows
-- pointing at deleted legacy kinds automatically.
-- -----------------------------------------------------------------------------
DELETE FROM kind_catalog WHERE id NOT IN ('project', 'actionItem');

-- -----------------------------------------------------------------------------
-- PHASE 4 — drop `kind` column from projects.
--
-- Per PLAN §1.14 Round-7 F2: use SQLite's native DROP COLUMN (3.35.0+, March
-- 2021). The dev sqlite3 CLI is 3.51.0; well past the floor. `projects.kind`
-- is a plain column with no PK / UNIQUE / FK / index / trigger / view
-- dependencies, so the native form works directly.
--
-- No PRAGMA foreign_keys = OFF/ON wrapper: PRAGMA foreign_keys inside an open
-- BEGIN TRANSACTION is a silent no-op per SQLite docs, and no table rebuild
-- happens so there's no CASCADE risk.
-- -----------------------------------------------------------------------------
ALTER TABLE projects DROP COLUMN kind;

-- -----------------------------------------------------------------------------
-- PHASE 5 — drop legacy `tasks` table (Unit 1.7 excised its Go references).
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS tasks;

-- -----------------------------------------------------------------------------
-- PHASE 6 — unify action_items kind + scope.
--
-- Every row becomes kind='actionItem', scope='actionItem'. Pre-drop cleanup
-- left rows on the old 'task' default; this rewrites them in a single UPDATE.
-- -----------------------------------------------------------------------------
UPDATE action_items SET kind = 'actionItem', scope = 'actionItem';

-- -----------------------------------------------------------------------------
-- PHASE 7 — assertion block. 8 end-state invariants.
--
-- A TEMP TABLE with CHECK(expected = actual) is the guard. Any INSERT whose
-- (expected, actual) pair disagrees raises a CHECK-constraint error, which
-- aborts the transaction and rolls back every DDL/DML since BEGIN. When all
-- pairs agree the TEMP TABLE holds the 8 diagnostic rows and is dropped on
-- COMMIT (TEMP tables disappear with the connection regardless).
--
-- SQL 3-valued-logic note (Round-5 O2): `NOT IN (...)` silently passes a NULL
-- input, so 7.7 and 7.8 add an explicit `OR <col> IS NULL` arm.
-- -----------------------------------------------------------------------------
CREATE TEMP TABLE drop_1_75_assertions (
    label    TEXT PRIMARY KEY,
    expected INTEGER NOT NULL,
    actual   INTEGER NOT NULL,
    CHECK(expected = actual)
);

-- 7.1 kind_catalog should have exactly 2 rows.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('kind_catalog_rows_equals_2', 2,
        (SELECT COUNT(*) FROM kind_catalog));

-- 7.2 no template_% tables remain.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('template_percent_tables_gone', 0,
        (SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'template_%'));

-- 7.3 no node_contract_% tables remain.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('node_contract_percent_tables_gone', 0,
        (SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'node_contract_%'));

-- 7.4 project_template_bindings table is gone.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('project_template_bindings_gone', 0,
        (SELECT COUNT(*) FROM sqlite_master WHERE name = 'project_template_bindings'));

-- 7.5 legacy tasks table is gone.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('tasks_table_gone', 0,
        (SELECT COUNT(*) FROM sqlite_master WHERE name = 'tasks'));

-- 7.6 projects.kind column is gone.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('projects_kind_column_gone', 0,
        (SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'kind'));

-- 7.7 no action_items carry an off-catalog kind.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('action_items_kinds_canonical', 0,
        (SELECT COUNT(*) FROM action_items
         WHERE kind NOT IN ('project','actionItem') OR kind IS NULL));

-- 7.8 F3 Option A — project_allowed_kinds has no residue pointing at a
--     deleted legacy kind. CASCADE from Phase 3 should clear these
--     automatically, so firing this assertion implies schema drift.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('project_allowed_kinds_canonical', 0,
        (SELECT COUNT(*) FROM project_allowed_kinds
         WHERE kind_id NOT IN ('project','actionItem') OR kind_id IS NULL));

-- Diagnostic echo of every assertion row (all pairs equal on success).
SELECT 'assert:' || label AS label, expected, actual FROM drop_1_75_assertions;

-- -----------------------------------------------------------------------------
-- PHASE 7-post — post-flight snapshot (informational, printed after assertions).
-- -----------------------------------------------------------------------------
SELECT 'post:projects'              AS label, COUNT(*) AS n FROM projects;
SELECT 'post:action_items'          AS label, COUNT(*) AS n FROM action_items;
SELECT 'post:kind_catalog'          AS label, COUNT(*) AS n FROM kind_catalog;
SELECT 'post:project_allowed_kinds' AS label, COUNT(*) AS n FROM project_allowed_kinds;

COMMIT;

-- -----------------------------------------------------------------------------
-- END
-- -----------------------------------------------------------------------------
