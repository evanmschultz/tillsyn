-- =============================================================================
-- drops-rewrite.sql — Drop 1.75 kind-collapse migration (12-kind enum).
--
-- Scope (8 phases, all wrapped in a single transaction):
--   1. Pre-flight counts — audit starting state.
--   2. DROP TABLE the template cluster (9 tables) BEFORE kind_catalog rewrite so
--      RESTRICT FKs from template_*.node_kind_id / .child_kind_id → kind_catalog
--      cannot trip during Phase 3.
--   3. DELETE FROM kind_catalog WHERE id NOT IN (12 new kinds).
--      `project_allowed_kinds.kind_id → kind_catalog(id) ON DELETE CASCADE`
--      clears allowlist rows pointing at deleted legacy kinds automatically.
--   3b. INSERT the 12 new kinds into kind_catalog with applies_to +
--       allowed_parent_scopes encoded per domain.AllowedParentKinds.
--   4. ALTER TABLE projects DROP COLUMN kind — native SQLite form (3.35.0+).
--      `projects.kind` is a plain column with no PK / UNIQUE / FK / index /
--      trigger / view dependencies; the 12-step rebuild is unnecessary.
--   5. DROP TABLE tasks — legacy empty table excised in Unit 1.7.
--   6. UPDATE action_items SET kind='plan', scope='plan' — remap every existing
--      row to the new default. Per dev 2026-04-23 decision (4.1): simplest
--      mapping — every row becomes a `plan`; manual retitling during dogfooding.
--   6b. UPDATE comments SET target_type='action_item' WHERE target_type NOT IN
--       ('project','action_item') — sweep legacy scope-level target types to
--       the new snake_case `action_item` value.
--   7. Assertion block — 10 end-state invariants guarded via a TEMP TABLE with
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
-- Run model:
--   - Dev-run only. Never invoked by CI. Never called by Go code.
--   - One-shot; once committed, historical reference only.
--   - Dev MUST back up `~/.tillsyn/tillsyn.db` before running.
--   - Intended to run AFTER Drop 1.75 Go code ships (12-kind enum in domain,
--     boot seeds in repo.go rewritten to seed the same 12 kinds).
--
-- How to run:
--   cp ~/.tillsyn/tillsyn.db ~/.tillsyn/tillsyn.db.pre-drop-1-75
--   sqlite3 ~/.tillsyn/tillsyn.db < scripts/drops-rewrite.sql
--
-- On assertion failure the transaction rolls back; the DB is unchanged.
--
-- F3 Option A (project_allowed_kinds residue, per PLAN §1.14):
--   Assertion 8 guards against allowlist rows referencing kinds outside the
--   12-value whitelist. Under the current schema the DELETE at Phase 3
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
SELECT 'pre:comments'                 AS label, COUNT(*) AS n FROM comments;
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
-- Delete every row whose id is not in the 12-value closed enum. The old
-- catalog had `project` + `actionItem` seeded at boot; both are being rewritten.
-- The ON DELETE CASCADE FK from project_allowed_kinds.kind_id → kind_catalog(id)
-- clears allowlist rows pointing at deleted kinds automatically.
-- -----------------------------------------------------------------------------
DELETE FROM kind_catalog
WHERE id NOT IN (
    'plan',
    'research',
    'build',
    'plan-qa-proof',
    'plan-qa-falsification',
    'build-qa-proof',
    'build-qa-falsification',
    'closeout',
    'commit',
    'refinement',
    'discussion',
    'human-verify'
);

-- -----------------------------------------------------------------------------
-- PHASE 3b — seed the 12 new kinds into kind_catalog.
--
-- Each row mirrors the Go-side boot seeds in internal/adapters/storage/sqlite/repo.go
-- (rewritten in Unit B of this drop). applies_to_json is ["<id>"] because
-- scope mirrors kind per row. allowed_parent_scopes_json encodes the parent
-- hierarchy rule from domain.AllowedParentKinds:
--   - plan  →  ["plan"] (nests; empty parent also allowed at root under project)
--   - research / build / closeout / commit / refinement / discussion /
--     human-verify / plan-qa-proof / plan-qa-falsification  →  ["plan"]
--   - build-qa-proof / build-qa-falsification  →  ["build"]
--
-- INSERT OR IGNORE keeps this idempotent if the Go boot seeder ran already.
-- -----------------------------------------------------------------------------
INSERT OR IGNORE INTO kind_catalog (
    id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
) VALUES
    ('plan',                   'Plan',                   'Planning-dominant kind; decomposes work into children.',             '["plan"]',                   '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('research',               'Research',               'Read-only investigation; compiles findings.',                        '["research"]',               '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('build',                  'Build',                  'Code-changing leaf; builder implements, tests, commits.',            '["build"]',                  '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('plan-qa-proof',          'Plan QA Proof',          'Proof-completeness QA pass on a plan parent.',                       '["plan-qa-proof"]',          '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('plan-qa-falsification',  'Plan QA Falsification',  'Falsification QA pass on a plan parent.',                            '["plan-qa-falsification"]',  '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('build-qa-proof',         'Build QA Proof',         'Proof-completeness QA pass on a build parent.',                      '["build-qa-proof"]',         '["build"]', '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('build-qa-falsification', 'Build QA Falsification', 'Falsification QA pass on a build parent.',                           '["build-qa-falsification"]', '["build"]', '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('closeout',               'Closeout',               'Drop-end coordination; aggregates ledger / refinements / findings.', '["closeout"]',               '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('commit',                 'Commit',                 'Commit action; template-triggered under plan at level >= 2.',        '["commit"]',                 '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('refinement',             'Refinement',             'Perpetual / long-lived tracking umbrella.',                          '["refinement"]',             '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('discussion',             'Discussion',             'Cross-cutting decision park.',                                       '["discussion"]',             '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL),
    ('human-verify',           'Human Verify',           'Dev sign-off hold point; no plan/QA children.',                      '["human-verify"]',           '["plan"]',  '', '{}', strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'), NULL);

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
-- PHASE 6 — remap action_items to the new kind vocabulary.
--
-- Every existing row becomes kind='plan', scope='plan'. Per dev direction
-- 2026-04-23: simplest mapping — every pre-collapse row becomes a `plan`
-- (the root/grouping kind). Dev retitles individual items to `build` /
-- `research` / other kinds manually during dogfooding as the project re-enters
-- active use. The scope mirror (scope = kind) is the post-Drop-1.75 invariant.
-- -----------------------------------------------------------------------------
UPDATE action_items SET kind = 'plan', scope = 'plan';

-- -----------------------------------------------------------------------------
-- PHASE 6b — sweep legacy comment target types.
--
-- Pre-collapse `comments.target_type` enum held `branch` / `phase` / `actionItem`
-- / `subtask` / `decision` / `note` / `project`. Post-collapse the domain enum
-- is only `project` + `action_item` (snake_case, changed from camelCase
-- `actionItem`). Remap every non-project row to `action_item`.
-- -----------------------------------------------------------------------------
UPDATE comments
SET target_type = 'action_item'
WHERE target_type NOT IN ('project', 'action_item');

-- -----------------------------------------------------------------------------
-- PHASE 7 — assertion block. 10 end-state invariants.
--
-- A TEMP TABLE with CHECK(expected = actual) is the guard. Any INSERT whose
-- (expected, actual) pair disagrees raises a CHECK-constraint error, which
-- aborts the transaction and rolls back every DDL/DML since BEGIN. When all
-- pairs agree the TEMP TABLE holds the diagnostic rows and is dropped on
-- COMMIT (TEMP tables disappear with the connection regardless).
--
-- SQL 3-valued-logic note (Round-5 O2): `NOT IN (...)` silently passes a NULL
-- input, so kind + scope + target_type assertions add explicit `OR <col> IS NULL` arms.
-- -----------------------------------------------------------------------------
CREATE TEMP TABLE drop_1_75_assertions (
    label    TEXT PRIMARY KEY,
    expected INTEGER NOT NULL,
    actual   INTEGER NOT NULL,
    CHECK(expected = actual)
);

-- 7.1 kind_catalog should have exactly 12 rows (the closed enum).
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('kind_catalog_rows_equals_12', 12,
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

-- 7.7 every action_items row carries one of the 12 canonical kinds.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('action_items_kinds_canonical', 0,
        (SELECT COUNT(*) FROM action_items
         WHERE kind NOT IN (
             'plan','research','build',
             'plan-qa-proof','plan-qa-falsification',
             'build-qa-proof','build-qa-falsification',
             'closeout','commit','refinement','discussion','human-verify'
         ) OR kind IS NULL));

-- 7.8 F3 Option A — project_allowed_kinds has no residue pointing at a
--     deleted legacy kind. CASCADE from Phase 3 should clear these
--     automatically, so firing this assertion implies schema drift.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('project_allowed_kinds_canonical', 0,
        (SELECT COUNT(*) FROM project_allowed_kinds
         WHERE kind_id NOT IN (
             'plan','research','build',
             'plan-qa-proof','plan-qa-falsification',
             'build-qa-proof','build-qa-falsification',
             'closeout','commit','refinement','discussion','human-verify'
         ) OR kind_id IS NULL));

-- 7.9 action_items.scope mirrors action_items.kind (post-collapse invariant).
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('action_items_scope_mirrors_kind', 0,
        (SELECT COUNT(*) FROM action_items
         WHERE scope IS NULL OR kind IS NULL OR scope != kind));

-- 7.10 comments.target_type is only in the new 2-value enum.
INSERT INTO drop_1_75_assertions (label, expected, actual) VALUES
    ('comments_target_type_canonical', 0,
        (SELECT COUNT(*) FROM comments
         WHERE target_type NOT IN ('project', 'action_item')
            OR target_type IS NULL));

-- Diagnostic echo of every assertion row (all pairs equal on success).
SELECT 'assert:' || label AS label, expected, actual FROM drop_1_75_assertions;

-- -----------------------------------------------------------------------------
-- PHASE 7-post — post-flight snapshot (informational, printed after assertions).
-- -----------------------------------------------------------------------------
SELECT 'post:projects'              AS label, COUNT(*) AS n FROM projects;
SELECT 'post:action_items'          AS label, COUNT(*) AS n FROM action_items;
SELECT 'post:kind_catalog'          AS label, COUNT(*) AS n FROM kind_catalog;
SELECT 'post:project_allowed_kinds' AS label, COUNT(*) AS n FROM project_allowed_kinds;
SELECT 'post:comments'              AS label, COUNT(*) AS n FROM comments;

COMMIT;

-- -----------------------------------------------------------------------------
-- END
-- -----------------------------------------------------------------------------
