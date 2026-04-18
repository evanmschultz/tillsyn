-- =============================================================================
-- rename-task-to-actionitem.sql — one-shot schema migration for the
-- Task → ActionItem identifier rename shipped pre-Drop-1.75 (2026-04-18).
--
-- Scope:
--   1. Rename table `work_items` → `action_items`.
--   2. Rename table `task_embeddings` → `action_item_embeddings`.
--   3. Rename column `task_id` → `action_item_id` inside the renamed
--      action_item_embeddings table.
--   4. Rebuild the `action_item_embeddings` indexes to match the new names.
--      (SQLite preserves PK/FK on RENAME TABLE but not the index name prefix.)
--
-- NOT touched intentionally:
--   - Legacy `tasks` table. It remains empty (or unused) and is dropped by
--     drops-rewrite.sql inside Drop 1.75. Leaving it alone here keeps the
--     two migrations independent.
--   - `lifecycle_state` string values (`done`/`progress`/`todo`/`archived`).
--   - kind_catalog rows (Drop 1.75's drops-rewrite.sql collapses them).
--   - Row-level `kind`/`scope` string values (the Go rename flipped the
--     code default from `'task'` to `'actionItem'`; existing rows keep their
--     historical `'task'` value until drops-rewrite.sql in Drop 1.75).
--
-- Run model:
--   - Dev-run only. Never invoked by CI. Never called by Go code.
--   - One-shot; once committed the script is historical reference only.
--   - Dev MUST back up `~/.tillsyn/tillsyn.db` before running.
--   - MUST run before booting the renamed `till` binary against an existing
--     database, otherwise the binary creates empty `action_items` /
--     `action_item_embeddings` tables alongside the old ones and orphans
--     all existing data.
--
-- How to run:
--   cp ~/.tillsyn/tillsyn.db ~/.tillsyn/tillsyn.db.pre-actionitem-rename
--   sqlite3 ~/.tillsyn/tillsyn.db < scripts/rename-task-to-actionitem.sql
--
-- On assertion failure the transaction rolls back; the DB is unchanged.
-- =============================================================================

PRAGMA foreign_keys = OFF;

BEGIN TRANSACTION;

-- -----------------------------------------------------------------------------
-- PHASE 0 — pre-flight snapshot (informational; printed by sqlite3)
-- -----------------------------------------------------------------------------
SELECT 'pre:work_items'                  AS label, COUNT(*) AS n FROM work_items;
SELECT 'pre:task_embeddings'             AS label, COUNT(*) AS n FROM task_embeddings;

-- -----------------------------------------------------------------------------
-- PHASE 1 — rename work_items → action_items
-- SQLite 3.25+ rewrites every FK reference to `work_items` in other tables.
-- -----------------------------------------------------------------------------
ALTER TABLE work_items RENAME TO action_items;

-- -----------------------------------------------------------------------------
-- PHASE 2 — rename task_embeddings → action_item_embeddings
-- Also rename its task_id column to action_item_id.
-- -----------------------------------------------------------------------------
ALTER TABLE task_embeddings RENAME TO action_item_embeddings;
ALTER TABLE action_item_embeddings RENAME COLUMN task_id TO action_item_id;

-- -----------------------------------------------------------------------------
-- PHASE 3 — rebuild indexes with new names
-- SQLite does NOT auto-rename indexes when RENAME TABLE fires. The old
-- `idx_task_embeddings_*` indexes (if they exist in this DB) still point at
-- the renamed table but keep their old names. Drop and recreate with the
-- new names so the schema matches what Go's CREATE INDEX IF NOT EXISTS
-- expects at boot.
-- -----------------------------------------------------------------------------
DROP INDEX IF EXISTS idx_task_embeddings_project;
DROP INDEX IF EXISTS idx_task_embeddings_updated_at;

CREATE INDEX IF NOT EXISTS idx_action_item_embeddings_project
    ON action_item_embeddings(project_id);
CREATE INDEX IF NOT EXISTS idx_action_item_embeddings_updated_at
    ON action_item_embeddings(updated_at);

-- -----------------------------------------------------------------------------
-- PHASE 4 — assert post-state
-- -----------------------------------------------------------------------------
SELECT 'post:action_items'               AS label, COUNT(*) AS n FROM action_items;
SELECT 'post:action_item_embeddings'     AS label, COUNT(*) AS n FROM action_item_embeddings;

-- Sanity: old names must be gone from sqlite_master.
SELECT 'post:work_items_still_exists'    AS label, COUNT(*) AS n
    FROM sqlite_master WHERE type='table' AND name='work_items';
SELECT 'post:task_embeddings_still_exists' AS label, COUNT(*) AS n
    FROM sqlite_master WHERE type='table' AND name='task_embeddings';

COMMIT;

PRAGMA foreign_keys = ON;
