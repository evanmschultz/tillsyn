package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/asg017/sqlite-vec-go-bindings/ncruces"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
)

// driverName defines a package constant value.
const driverName = "sqlite3"
const defaultEmbeddingSearchLimit = 200
const defaultBusyTimeout = 60 * time.Second

const (
	// globalAuthProjectSlug stores the internal hidden slug that backs global auth routing.
	globalAuthProjectSlug = "all-projects-internal"
	// globalAuthProjectName stores the internal hidden project name that backs global auth routing.
	globalAuthProjectName = "All Projects (Internal)"
	// globalAuthProjectCreatedAt stores a deterministic timestamp for the hidden auth-routing project row.
	globalAuthProjectCreatedAt = "1970-01-01T00:00:00Z"
)

// errSQLiteVecUnavailable reports that sqlite-vec functions are unavailable in the active runtime.
var errSQLiteVecUnavailable = errors.New("sqlite vec capability unavailable")

func init() {
	cfg := wazero.NewRuntimeConfig()
	if bits.UintSize < 64 {
		cfg = cfg.WithMemoryLimitPages(512) // 32MB, aligns with ncruces 32-bit default.
	} else {
		cfg = cfg.WithMemoryLimitPages(4096) // 256MB, aligns with ncruces 64-bit default.
	}
	// sqlite-vec's ncruces wasm build uses atomic instructions; enable thread features
	// in wazero so ncruces can compile the embedded module at runtime while preserving
	// ncruces' bounded default memory limits.
	sqlite3.RuntimeConfig = cfg.WithCoreFeatures(
		api.CoreFeaturesV2 | experimental.CoreFeaturesThreads,
	)
}

// Repository represents repository data used by this package.
type Repository struct {
	db           *sql.DB
	vecAvailable bool
}

// DB returns the underlying SQLite handle used by the repository.
func (r *Repository) DB() *sql.DB {
	if r == nil {
		return nil
	}
	return r.db
}

// Open opens the requested operation.
func Open(path string) (*Repository, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}
	db, err := sql.Open(driverName, path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := applySQLiteConnectionPragmas(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	repo := &Repository{db: db}
	if err := repo.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

// OpenInMemory opens in memory.
func OpenInMemory() (*Repository, error) {
	db, err := sql.Open(driverName, "file::memory:?cache=shared")
	if err != nil {
		return nil, fmt.Errorf("open sqlite memory: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := applySQLiteConnectionPragmas(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	repo := &Repository{db: db}
	if err := repo.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

// Close closes the requested operation.
func (r *Repository) Close() error {
	return r.db.Close()
}

// applySQLiteConnectionPragmas configures the one-connection pool with the pragmas required for local dogfooding.
func applySQLiteConnectionPragmas(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("sqlite db is required")
	}
	pragmas := []string{
		fmt.Sprintf("PRAGMA busy_timeout = %d", defaultBusyTimeout/time.Millisecond),
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("apply sqlite pragma %q: %w", pragma, err)
		}
	}
	return nil
}

// migrate applies schema and data migrations required for compatibility.
func (r *Repository) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL DEFAULT 'project',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS columns_v1 (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			wip_limit INTEGER NOT NULL DEFAULT 0,
			position INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			parent_id TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL DEFAULT 'task',
			scope TEXT NOT NULL DEFAULT 'task',
			lifecycle_state TEXT NOT NULL DEFAULT 'todo',
			column_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL,
			due_at TEXT,
			labels_json TEXT NOT NULL DEFAULT '[]',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user',
			created_by_name TEXT NOT NULL DEFAULT 'tillsyn-user',
			updated_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user',
			updated_by_name TEXT NOT NULL DEFAULT 'tillsyn-user',
			updated_by_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT,
			archived_at TEXT,
			canceled_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(column_id) REFERENCES columns_v1(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS work_items (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			parent_id TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL DEFAULT 'task',
			scope TEXT NOT NULL DEFAULT 'task',
			lifecycle_state TEXT NOT NULL DEFAULT 'todo',
			column_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL,
			due_at TEXT,
			labels_json TEXT NOT NULL DEFAULT '[]',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user',
			created_by_name TEXT NOT NULL DEFAULT 'tillsyn-user',
			updated_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user',
			updated_by_name TEXT NOT NULL DEFAULT 'tillsyn-user',
			updated_by_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT,
			archived_at TEXT,
			canceled_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(column_id) REFERENCES columns_v1(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS task_embeddings (
			task_id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			embedding BLOB NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(task_id) REFERENCES work_items(id) ON DELETE CASCADE,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_task_embeddings_project ON task_embeddings(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_embeddings_updated_at ON task_embeddings(updated_at);`,
		`CREATE TABLE IF NOT EXISTS embedding_documents (
			subject_type TEXT NOT NULL,
			subject_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			search_target_type TEXT NOT NULL,
			search_target_id TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			embedding BLOB NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(subject_type, subject_id),
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_embedding_documents_project ON embedding_documents(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_embedding_documents_target ON embedding_documents(search_target_type, search_target_id, updated_at DESC, subject_type, subject_id);`,
		`CREATE INDEX IF NOT EXISTS idx_embedding_documents_updated_at ON embedding_documents(updated_at);`,
		`CREATE TABLE IF NOT EXISTS embedding_jobs (
			subject_type TEXT NOT NULL,
			subject_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			desired_content_hash TEXT NOT NULL DEFAULT '',
			indexed_content_hash TEXT NOT NULL DEFAULT '',
			model_provider TEXT NOT NULL DEFAULT '',
			model_name TEXT NOT NULL DEFAULT '',
			model_dimensions INTEGER NOT NULL DEFAULT 0,
			model_signature TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 0,
			last_enqueued_at TEXT NOT NULL,
			last_started_at TEXT,
			last_heartbeat_at TEXT,
			last_succeeded_at TEXT,
			last_failed_at TEXT,
			next_attempt_at TEXT,
			claimed_by TEXT NOT NULL DEFAULT '',
			claim_expires_at TEXT,
			last_error_code TEXT NOT NULL DEFAULT '',
			last_error_message TEXT NOT NULL DEFAULT '',
			last_error_summary TEXT NOT NULL DEFAULT '',
			stale_reason TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(subject_type, subject_id),
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_embedding_jobs_project_status_updated_at ON embedding_jobs(project_id, status, updated_at DESC, subject_type, subject_id);`,
		`CREATE INDEX IF NOT EXISTS idx_embedding_jobs_project_next_attempt ON embedding_jobs(project_id, next_attempt_at, updated_at DESC, subject_type, subject_id);`,
		`CREATE INDEX IF NOT EXISTS idx_embedding_jobs_claim_expires ON embedding_jobs(claim_expires_at, status, subject_type, subject_id);`,
		`CREATE TABLE IF NOT EXISTS change_events (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				project_id TEXT NOT NULL,
				work_item_id TEXT NOT NULL,
				operation TEXT NOT NULL,
				actor_id TEXT NOT NULL,
				actor_name TEXT NOT NULL DEFAULT 'tillsyn-user',
				actor_type TEXT NOT NULL,
				metadata_json TEXT NOT NULL DEFAULT '{}',
				created_at TEXT NOT NULL,
				FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
			);`,
		// comments.target_id is polymorphic, so only project_id is enforced as a foreign key.
		`CREATE TABLE IF NOT EXISTS comments (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
				target_type TEXT NOT NULL,
				target_id TEXT NOT NULL,
				summary TEXT NOT NULL DEFAULT '',
				body_markdown TEXT NOT NULL,
				actor_id TEXT NOT NULL DEFAULT 'tillsyn-user',
				actor_name TEXT NOT NULL DEFAULT 'tillsyn-user',
				actor_type TEXT NOT NULL DEFAULT 'user',
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
			);`,
		`CREATE TABLE IF NOT EXISTS kind_catalog (
			id TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			description_markdown TEXT NOT NULL DEFAULT '',
			applies_to_json TEXT NOT NULL DEFAULT '[]',
			allowed_parent_scopes_json TEXT NOT NULL DEFAULT '[]',
			payload_schema_json TEXT NOT NULL DEFAULT '',
			template_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS project_allowed_kinds (
			project_id TEXT NOT NULL,
			kind_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY(project_id, kind_id),
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(kind_id) REFERENCES kind_catalog(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS template_libraries (
			id TEXT PRIMARY KEY,
			scope TEXT NOT NULL,
			project_id TEXT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			source_library_id TEXT,
			builtin_managed INTEGER NOT NULL DEFAULT 0,
			builtin_source TEXT NOT NULL DEFAULT '',
			builtin_version TEXT NOT NULL DEFAULT '',
			revision INTEGER NOT NULL DEFAULT 1,
			revision_digest TEXT NOT NULL DEFAULT '',
			created_by_actor_id TEXT NOT NULL DEFAULT '',
			created_by_actor_name TEXT NOT NULL DEFAULT '',
			created_by_actor_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			approved_by_actor_id TEXT NOT NULL DEFAULT '',
			approved_by_actor_name TEXT NOT NULL DEFAULT '',
			approved_by_actor_type TEXT NOT NULL DEFAULT '',
			approved_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_template_libraries_scope_status ON template_libraries(scope, status, name, id);`,
		`CREATE INDEX IF NOT EXISTS idx_template_libraries_project ON template_libraries(project_id, status, name, id);`,
		`CREATE TABLE IF NOT EXISTS template_node_templates (
			library_id TEXT NOT NULL,
			id TEXT NOT NULL,
			scope_level TEXT NOT NULL,
			node_kind_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			description_markdown TEXT NOT NULL DEFAULT '',
			project_metadata_defaults_json TEXT NOT NULL DEFAULT '',
			task_metadata_defaults_json TEXT NOT NULL DEFAULT '',
			PRIMARY KEY(library_id, id),
			UNIQUE(library_id, scope_level, node_kind_id),
			FOREIGN KEY(library_id) REFERENCES template_libraries(id) ON DELETE CASCADE,
			FOREIGN KEY(node_kind_id) REFERENCES kind_catalog(id) ON DELETE RESTRICT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_template_node_templates_library ON template_node_templates(library_id, scope_level, node_kind_id, id);`,
		`CREATE TABLE IF NOT EXISTS template_child_rules (
			library_id TEXT NOT NULL,
			node_template_id TEXT NOT NULL,
			id TEXT NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			child_scope_level TEXT NOT NULL,
			child_kind_id TEXT NOT NULL,
			title_template TEXT NOT NULL,
			description_template TEXT NOT NULL DEFAULT '',
			responsible_actor_kind TEXT NOT NULL,
			orchestrator_may_complete INTEGER NOT NULL DEFAULT 0,
			required_for_parent_done INTEGER NOT NULL DEFAULT 0,
			required_for_containing_done INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY(library_id, node_template_id, id),
			FOREIGN KEY(library_id, node_template_id) REFERENCES template_node_templates(library_id, id) ON DELETE CASCADE,
			FOREIGN KEY(child_kind_id) REFERENCES kind_catalog(id) ON DELETE RESTRICT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_template_child_rules_template ON template_child_rules(library_id, node_template_id, position, id);`,
		`CREATE TABLE IF NOT EXISTS template_child_rule_editor_kinds (
			library_id TEXT NOT NULL,
			node_template_id TEXT NOT NULL,
			child_rule_id TEXT NOT NULL,
			actor_kind TEXT NOT NULL,
			PRIMARY KEY(library_id, node_template_id, child_rule_id, actor_kind),
			FOREIGN KEY(library_id, node_template_id, child_rule_id) REFERENCES template_child_rules(library_id, node_template_id, id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS template_child_rule_completer_kinds (
			library_id TEXT NOT NULL,
			node_template_id TEXT NOT NULL,
			child_rule_id TEXT NOT NULL,
			actor_kind TEXT NOT NULL,
			PRIMARY KEY(library_id, node_template_id, child_rule_id, actor_kind),
			FOREIGN KEY(library_id, node_template_id, child_rule_id) REFERENCES template_child_rules(library_id, node_template_id, id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS project_template_bindings (
			project_id TEXT PRIMARY KEY,
			library_id TEXT NOT NULL,
			library_name TEXT NOT NULL DEFAULT '',
			bound_revision INTEGER NOT NULL DEFAULT 1,
			bound_revision_digest TEXT NOT NULL DEFAULT '',
			bound_library_updated_at TEXT NOT NULL DEFAULT '',
			bound_library_snapshot_json TEXT NOT NULL DEFAULT '',
			bound_by_actor_id TEXT NOT NULL DEFAULT '',
			bound_by_actor_name TEXT NOT NULL DEFAULT '',
			bound_by_actor_type TEXT NOT NULL DEFAULT 'user',
			bound_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(library_id) REFERENCES template_libraries(id) ON DELETE RESTRICT
		);`,
		`CREATE TABLE IF NOT EXISTS node_contract_snapshots (
			node_id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			source_library_id TEXT NOT NULL,
			source_node_template_id TEXT NOT NULL,
			source_child_rule_id TEXT NOT NULL,
			created_by_actor_id TEXT NOT NULL DEFAULT '',
			created_by_actor_type TEXT NOT NULL DEFAULT 'system',
			responsible_actor_kind TEXT NOT NULL,
			orchestrator_may_complete INTEGER NOT NULL DEFAULT 0,
			required_for_parent_done INTEGER NOT NULL DEFAULT 0,
			required_for_containing_done INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY(node_id) REFERENCES work_items(id) ON DELETE CASCADE,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_node_contract_snapshots_project ON node_contract_snapshots(project_id, created_at, node_id);`,
		`CREATE TABLE IF NOT EXISTS node_contract_editor_kinds (
			node_id TEXT NOT NULL,
			actor_kind TEXT NOT NULL,
			PRIMARY KEY(node_id, actor_kind),
			FOREIGN KEY(node_id) REFERENCES node_contract_snapshots(node_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS node_contract_completer_kinds (
			node_id TEXT NOT NULL,
			actor_kind TEXT NOT NULL,
			PRIMARY KEY(node_id, actor_kind),
			FOREIGN KEY(node_id) REFERENCES node_contract_snapshots(node_id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS capability_leases (
			instance_id TEXT PRIMARY KEY,
			lease_token TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			project_id TEXT NOT NULL,
			scope_type TEXT NOT NULL,
			scope_id TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL,
			parent_instance_id TEXT NOT NULL DEFAULT '',
			allow_equal_scope_delegation INTEGER NOT NULL DEFAULT 0,
			issued_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			heartbeat_at TEXT NOT NULL,
			revoked_at TEXT,
			revoke_reason TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS attention_items (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			branch_id TEXT NOT NULL DEFAULT '',
			scope_type TEXT NOT NULL,
			scope_id TEXT NOT NULL,
			state TEXT NOT NULL,
			kind TEXT NOT NULL,
			summary TEXT NOT NULL,
			body_markdown TEXT NOT NULL DEFAULT '',
			requires_user_action INTEGER NOT NULL DEFAULT 0,
			created_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user',
			created_by_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			acknowledged_by_actor TEXT NOT NULL DEFAULT '',
			acknowledged_by_type TEXT NOT NULL DEFAULT '',
			acknowledged_at TEXT,
			resolved_by_actor TEXT NOT NULL DEFAULT '',
			resolved_by_type TEXT NOT NULL DEFAULT '',
			resolved_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS auth_requests (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			branch_id TEXT NOT NULL DEFAULT '',
			phase_ids_json TEXT NOT NULL DEFAULT '[]',
			path TEXT NOT NULL,
			scope_type TEXT NOT NULL,
			scope_id TEXT NOT NULL,
			principal_id TEXT NOT NULL,
			principal_type TEXT NOT NULL,
			principal_role TEXT NOT NULL DEFAULT '',
			principal_name TEXT NOT NULL DEFAULT '',
			client_id TEXT NOT NULL,
			client_type TEXT NOT NULL DEFAULT '',
			client_name TEXT NOT NULL DEFAULT '',
			requested_session_ttl_seconds INTEGER NOT NULL,
			approved_path TEXT NOT NULL DEFAULT '',
			approved_session_ttl_seconds INTEGER NOT NULL DEFAULT 0,
			reason TEXT NOT NULL DEFAULT '',
			continuation_json TEXT NOT NULL DEFAULT '{}',
			state TEXT NOT NULL,
			requested_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user',
			requested_by_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			resolved_by_actor TEXT NOT NULL DEFAULT '',
			resolved_by_type TEXT NOT NULL DEFAULT '',
			resolved_at TEXT,
			resolution_note TEXT NOT NULL DEFAULT '',
			issued_session_id TEXT NOT NULL DEFAULT '',
			issued_session_secret TEXT NOT NULL DEFAULT '',
			issued_session_expires_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS handoffs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			branch_id TEXT NOT NULL DEFAULT '',
			scope_type TEXT NOT NULL,
			scope_id TEXT NOT NULL,
			target_branch_id TEXT NOT NULL DEFAULT '',
			target_scope_type TEXT NOT NULL DEFAULT '',
			target_scope_id TEXT NOT NULL DEFAULT '',
			source_role TEXT NOT NULL DEFAULT '',
			target_role TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			summary TEXT NOT NULL,
			next_action TEXT NOT NULL DEFAULT '',
			missing_evidence_json TEXT NOT NULL DEFAULT '[]',
			related_refs_json TEXT NOT NULL DEFAULT '[]',
			created_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user',
			created_by_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			updated_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user',
			updated_by_type TEXT NOT NULL DEFAULT 'user',
			updated_at TEXT NOT NULL,
			resolved_by_actor TEXT NOT NULL DEFAULT '',
			resolved_by_type TEXT NOT NULL DEFAULT '',
			resolved_at TEXT,
			resolution_note TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_columns_project_position ON columns_v1(project_id, position);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_column_position ON tasks(project_id, column_id, position);`,
		`CREATE INDEX IF NOT EXISTS idx_work_items_project_column_position ON work_items(project_id, column_id, position);`,
		`CREATE INDEX IF NOT EXISTS idx_work_items_project_parent ON work_items(project_id, parent_id);`,
		`CREATE INDEX IF NOT EXISTS idx_change_events_project_created_at ON change_events(project_id, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_project_target_created_at ON comments(project_id, target_type, target_id, created_at ASC, id ASC);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_project_created_at ON comments(project_id, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_project_allowed_kinds_project ON project_allowed_kinds(project_id, kind_id);`,
		`CREATE INDEX IF NOT EXISTS idx_capability_leases_scope ON capability_leases(project_id, scope_type, scope_id, role);`,
		`CREATE INDEX IF NOT EXISTS idx_capability_leases_expiry ON capability_leases(expires_at, revoked_at);`,
		`CREATE INDEX IF NOT EXISTS idx_attention_scope_state_created_at ON attention_items(project_id, scope_type, scope_id, state, requires_user_action, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_attention_project_state_kind_created_at ON attention_items(project_id, state, kind, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_auth_requests_project_state_created_at ON auth_requests(project_id, state, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_auth_requests_expiry_state ON auth_requests(state, expires_at, created_at DESC);`,
		`DROP INDEX IF EXISTS idx_handoffs_project_status_created_at;`,
		`CREATE INDEX IF NOT EXISTS idx_handoffs_project_status_updated_at ON handoffs(project_id, status, updated_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_handoffs_project_scope_created_at ON handoffs(project_id, scope_type, scope_id, updated_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_handoffs_project_target_scope_created_at ON handoffs(project_id, target_scope_type, target_scope_id, updated_at DESC, id DESC);`,
	}

	for _, stmt := range stmts {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate sqlite: %w", err)
		}
	}
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE projects ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}'`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("migrate sqlite add projects.metadata_json: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE projects ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("migrate sqlite add projects.kind: %w", err)
	}
	taskAlterStatements := []string{
		`ALTER TABLE tasks ADD COLUMN parent_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN kind TEXT NOT NULL DEFAULT 'task'`,
		`ALTER TABLE tasks ADD COLUMN scope TEXT NOT NULL DEFAULT 'task'`,
		`ALTER TABLE tasks ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'todo'`,
		`ALTER TABLE tasks ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE tasks ADD COLUMN created_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user'`,
		`ALTER TABLE tasks ADD COLUMN created_by_name TEXT NOT NULL DEFAULT 'tillsyn-user'`,
		`ALTER TABLE tasks ADD COLUMN updated_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user'`,
		`ALTER TABLE tasks ADD COLUMN updated_by_name TEXT NOT NULL DEFAULT 'tillsyn-user'`,
		`ALTER TABLE tasks ADD COLUMN updated_by_type TEXT NOT NULL DEFAULT 'user'`,
		`ALTER TABLE tasks ADD COLUMN started_at TEXT`,
		`ALTER TABLE tasks ADD COLUMN completed_at TEXT`,
		`ALTER TABLE tasks ADD COLUMN canceled_at TEXT`,
	}
	for _, stmt := range taskAlterStatements {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil && !isDuplicateColumnErr(err) {
			return fmt.Errorf("migrate sqlite tasks: %w", err)
		}
	}
	workItemAlterStatements := []string{
		`ALTER TABLE work_items ADD COLUMN parent_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE work_items ADD COLUMN kind TEXT NOT NULL DEFAULT 'task'`,
		`ALTER TABLE work_items ADD COLUMN scope TEXT NOT NULL DEFAULT 'task'`,
		`ALTER TABLE work_items ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'todo'`,
		`ALTER TABLE work_items ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE work_items ADD COLUMN created_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user'`,
		`ALTER TABLE work_items ADD COLUMN created_by_name TEXT NOT NULL DEFAULT 'tillsyn-user'`,
		`ALTER TABLE work_items ADD COLUMN updated_by_actor TEXT NOT NULL DEFAULT 'tillsyn-user'`,
		`ALTER TABLE work_items ADD COLUMN updated_by_name TEXT NOT NULL DEFAULT 'tillsyn-user'`,
		`ALTER TABLE work_items ADD COLUMN updated_by_type TEXT NOT NULL DEFAULT 'user'`,
		`ALTER TABLE work_items ADD COLUMN started_at TEXT`,
		`ALTER TABLE work_items ADD COLUMN completed_at TEXT`,
		`ALTER TABLE work_items ADD COLUMN canceled_at TEXT`,
	}
	for _, stmt := range workItemAlterStatements {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil && !isDuplicateColumnErr(err) {
			return fmt.Errorf("migrate sqlite work_items: %w", err)
		}
	}
	if err := r.migrateCommentsOwnershipTuple(ctx); err != nil {
		return err
	}
	if err := r.migrateCommentSummary(ctx); err != nil {
		return err
	}
	if err := r.migrateChangeEventsActorName(ctx); err != nil {
		return err
	}
	authRequestAlterStatements := []string{
		`ALTER TABLE auth_requests ADD COLUMN principal_role TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE auth_requests ADD COLUMN approved_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE auth_requests ADD COLUMN approved_session_ttl_seconds INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range authRequestAlterStatements {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil && !isDuplicateColumnErr(err) {
			return fmt.Errorf("migrate sqlite auth_requests: %w", err)
		}
	}
	if err := r.migrateTemplateLifecycle(ctx); err != nil {
		return err
	}
	if err := r.migrateTaskActorNames(ctx); err != nil {
		return err
	}
	if err := r.migratePhaseScopeContract(ctx); err != nil {
		return err
	}
	if err := r.ensureCommentIndexes(ctx); err != nil {
		return err
	}
	if err := r.migrateLegacyEmbeddingDocuments(ctx); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_tasks_project_parent ON tasks(project_id, parent_id)`); err != nil {
		return fmt.Errorf("migrate sqlite task parent index: %w", err)
	}
	if err := r.bridgeLegacyTasksToWorkItems(ctx); err != nil {
		return err
	}
	if err := r.seedDefaultKindCatalog(ctx); err != nil {
		return err
	}
	if err := r.ensureGlobalAuthProject(ctx); err != nil {
		return err
	}
	if err := r.probeVecCapability(ctx); err != nil {
		if errors.Is(err, errSQLiteVecUnavailable) {
			return nil
		}
		return fmt.Errorf("migrate sqlite vec capability probe: %w", err)
	}
	return nil
}

// migrateLegacyEmbeddingDocuments copies legacy task-only vectors into the generic document table.
func (r *Repository) migrateLegacyEmbeddingDocuments(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO embedding_documents (
			subject_type, subject_id, project_id, search_target_type, search_target_id, content_hash, content, embedding, updated_at
		)
		SELECT ?, task_id, project_id, ?, task_id, content_hash, content, embedding, updated_at
		FROM task_embeddings
	`, string(app.EmbeddingSubjectTypeWorkItem), string(app.EmbeddingSearchTargetTypeWorkItem)); err != nil {
		return fmt.Errorf("migrate sqlite legacy embedding documents: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `
		UPDATE embedding_jobs
		SET subject_type = ?
		WHERE subject_type = ?
	`, string(app.EmbeddingSubjectTypeWorkItem), "task"); err != nil {
		return fmt.Errorf("migrate sqlite embedding job subject types: %w", err)
	}
	return nil
}

// migratePhaseScopeContract rewrites legacy subphase markers into the canonical phase contract.
func (r *Repository) migratePhaseScopeContract(ctx context.Context) error {
	rewriteStatements := []struct {
		name string
		sql  string
		args []any
	}{
		{name: "tasks.scope", sql: `UPDATE tasks SET scope = ? WHERE scope = ?`, args: []any{string(domain.KindAppliesToPhase), "subphase"}},
		{name: "work_items.scope", sql: `UPDATE work_items SET scope = ? WHERE scope = ?`, args: []any{string(domain.KindAppliesToPhase), "subphase"}},
		{name: "comments.target_type", sql: `UPDATE comments SET target_type = ? WHERE target_type = ?`, args: []any{string(domain.CommentTargetTypePhase), "subphase"}},
		{name: "capability_leases.scope_type", sql: `UPDATE capability_leases SET scope_type = ? WHERE scope_type = ?`, args: []any{string(domain.CapabilityScopePhase), "subphase"}},
		{name: "attention_items.scope_type", sql: `UPDATE attention_items SET scope_type = ? WHERE scope_type = ?`, args: []any{string(domain.ScopeLevelPhase), "subphase"}},
		{name: "change_events.metadata_json", sql: `UPDATE change_events SET metadata_json = replace(metadata_json, '\"subphase\"', '\"phase\"') WHERE metadata_json LIKE '%\"subphase\"%'`, args: nil},
	}
	for _, stmt := range rewriteStatements {
		if _, err := r.db.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			return fmt.Errorf("migrate phase scope contract %s: %w", stmt.name, err)
		}
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE work_items SET scope = ? WHERE kind = ? AND scope = ?`, string(domain.KindAppliesToPhase), string(domain.WorkKindPhase), string(domain.KindAppliesToTask)); err != nil {
		return fmt.Errorf("migrate phase scope contract work_items project phase scope: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE tasks SET scope = ? WHERE kind = ? AND scope = ?`, string(domain.KindAppliesToPhase), string(domain.WorkKindPhase), string(domain.KindAppliesToTask)); err != nil {
		return fmt.Errorf("migrate phase scope contract tasks project phase scope: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `SELECT id, applies_to_json, allowed_parent_scopes_json FROM kind_catalog`)
	if err != nil {
		return fmt.Errorf("migrate phase scope contract query kind_catalog: %w", err)
	}
	defer rows.Close()

	type kindRow struct {
		id          string
		appliesTo   []domain.KindAppliesTo
		parentScope []domain.KindAppliesTo
	}
	toUpdate := make([]kindRow, 0)
	for rows.Next() {
		var id, appliesRaw, parentRaw string
		if err := rows.Scan(&id, &appliesRaw, &parentRaw); err != nil {
			return fmt.Errorf("migrate phase scope contract scan kind_catalog: %w", err)
		}

		var appliesTo []domain.KindAppliesTo
		if err := json.Unmarshal([]byte(appliesRaw), &appliesTo); err != nil {
			return fmt.Errorf("migrate phase scope contract decode kind applies_to %q: %w", id, err)
		}
		var parentScopes []domain.KindAppliesTo
		if err := json.Unmarshal([]byte(parentRaw), &parentScopes); err != nil {
			return fmt.Errorf("migrate phase scope contract decode kind parent scopes %q: %w", id, err)
		}

		normalizedApplies := rewriteSubphaseKindAppliesTo(appliesTo)
		normalizedParents := rewriteSubphaseKindAppliesTo(parentScopes)
		if kindAppliesToEqual(appliesTo, normalizedApplies) && kindAppliesToEqual(parentScopes, normalizedParents) {
			continue
		}
		toUpdate = append(toUpdate, kindRow{id: id, appliesTo: normalizedApplies, parentScope: normalizedParents})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate phase scope contract iterate kind_catalog: %w", err)
	}

	now := time.Now().UTC()
	for _, row := range toUpdate {
		appliesJSON, err := json.Marshal(row.appliesTo)
		if err != nil {
			return fmt.Errorf("migrate phase scope contract encode kind applies_to %q: %w", row.id, err)
		}
		parentJSON, err := json.Marshal(row.parentScope)
		if err != nil {
			return fmt.Errorf("migrate phase scope contract encode kind parent scopes %q: %w", row.id, err)
		}
		if _, err := r.db.ExecContext(ctx, `UPDATE kind_catalog SET applies_to_json = ?, allowed_parent_scopes_json = ?, updated_at = ? WHERE id = ?`, string(appliesJSON), string(parentJSON), ts(now), row.id); err != nil {
			return fmt.Errorf("migrate phase scope contract update kind_catalog %q: %w", row.id, err)
		}
	}
	return nil
}

// rewriteSubphaseKindAppliesTo replaces the removed subphase marker with phase and de-duplicates.
func rewriteSubphaseKindAppliesTo(values []domain.KindAppliesTo) []domain.KindAppliesTo {
	out := make([]domain.KindAppliesTo, 0, len(values))
	seen := map[domain.KindAppliesTo]struct{}{}
	for _, raw := range values {
		scope := domain.NormalizeKindAppliesTo(raw)
		if scope == "" {
			continue
		}
		if scope == "subphase" {
			scope = domain.KindAppliesToPhase
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

// migrateCommentsOwnershipTuple rewrites comments to the canonical ownership tuple columns.
func (r *Repository) migrateCommentsOwnershipTuple(ctx context.Context) error {
	hasActorID, err := r.tableHasColumn(ctx, "comments", "actor_id")
	if err != nil {
		return fmt.Errorf("migrate sqlite comments actor_id check: %w", err)
	}
	hasActorName, err := r.tableHasColumn(ctx, "comments", "actor_name")
	if err != nil {
		return fmt.Errorf("migrate sqlite comments actor_name check: %w", err)
	}
	hasAuthorName, err := r.tableHasColumn(ctx, "comments", "author_name")
	if err != nil {
		return fmt.Errorf("migrate sqlite comments author_name check: %w", err)
	}
	if hasActorID && hasActorName && !hasAuthorName {
		return nil
	}
	hasActorType, err := r.tableHasColumn(ctx, "comments", "actor_type")
	if err != nil {
		return fmt.Errorf("migrate sqlite comments actor_type check: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate sqlite comments begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `ALTER TABLE comments RENAME TO comments_legacy`); err != nil {
		return fmt.Errorf("migrate sqlite comments rename legacy table: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `
		CREATE TABLE comments (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			body_markdown TEXT NOT NULL,
			actor_id TEXT NOT NULL DEFAULT 'tillsyn-user',
			actor_name TEXT NOT NULL DEFAULT 'tillsyn-user',
			actor_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("migrate sqlite comments create canonical table: %w", err)
	}

	actorIDExpr := `'tillsyn-user'`
	switch {
	case hasActorID:
		actorIDExpr = `NULLIF(TRIM(actor_id), '')`
	case hasAuthorName:
		actorIDExpr = `NULLIF(TRIM(author_name), '')`
	}
	actorNameExpr := `NULL`
	switch {
	case hasActorName:
		actorNameExpr = `NULLIF(TRIM(actor_name), '')`
	case hasAuthorName:
		actorNameExpr = `NULLIF(TRIM(author_name), '')`
	}
	actorTypeExpr := `'user'`
	if hasActorType {
		actorTypeExpr = `COALESCE(NULLIF(TRIM(actor_type), ''), 'user')`
	}
	actorIDSelect := fmt.Sprintf("COALESCE(%s, 'tillsyn-user')", actorIDExpr)
	actorNameSelect := fmt.Sprintf("COALESCE(%s, %s, 'tillsyn-user')", actorNameExpr, actorIDSelect)
	copyStmt := fmt.Sprintf(`
		INSERT INTO comments(id, project_id, target_type, target_id, summary, body_markdown, actor_id, actor_name, actor_type, created_at, updated_at)
		SELECT id, project_id, target_type, target_id, '', body_markdown, %s, %s, %s, created_at, updated_at
		FROM comments_legacy
	`, actorIDSelect, actorNameSelect, actorTypeExpr)
	if _, err = tx.ExecContext(ctx, copyStmt); err != nil {
		return fmt.Errorf("migrate sqlite comments copy rows: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DROP TABLE comments_legacy`); err != nil {
		return fmt.Errorf("migrate sqlite comments drop legacy table: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("migrate sqlite comments commit: %w", err)
	}
	return nil
}

// migrateCommentSummary adds and backfills the canonical comments.summary column.
func (r *Repository) migrateCommentSummary(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE comments ADD COLUMN summary TEXT NOT NULL DEFAULT ''`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("migrate sqlite add comments.summary: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, body_markdown
		FROM comments
		WHERE NULLIF(TRIM(summary), '') IS NULL
	`)
	if err != nil {
		return fmt.Errorf("migrate sqlite list comments missing summary: %w", err)
	}
	defer rows.Close()

	type summaryBackfill struct {
		commentID string
		summary   string
	}
	backfillRows := make([]summaryBackfill, 0)

	for rows.Next() {
		var (
			commentID    string
			bodyMarkdown string
		)
		if err := rows.Scan(&commentID, &bodyMarkdown); err != nil {
			return fmt.Errorf("migrate sqlite scan comments summary row: %w", err)
		}
		backfillRows = append(backfillRows, summaryBackfill{
			commentID: commentID,
			summary:   commentSummaryFromBody(bodyMarkdown),
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate sqlite iterate comments summary rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("migrate sqlite close comments summary rows: %w", err)
	}
	for _, row := range backfillRows {
		if _, err := r.db.ExecContext(ctx, `UPDATE comments SET summary = ? WHERE id = ?`, row.summary, row.commentID); err != nil {
			return fmt.Errorf("migrate sqlite backfill comments.summary for %q: %w", row.commentID, err)
		}
	}
	return nil
}

// migrateChangeEventsActorName adds and backfills the actor_name ownership column.
func (r *Repository) migrateChangeEventsActorName(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE change_events ADD COLUMN actor_name TEXT NOT NULL DEFAULT ''`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("migrate sqlite add change_events.actor_name: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `
		UPDATE change_events
		SET actor_name = COALESCE(NULLIF(TRIM(actor_id), ''), 'tillsyn-user')
		WHERE NULLIF(TRIM(actor_name), '') IS NULL
			OR (
				TRIM(actor_name) = 'tillsyn-user'
				AND NULLIF(TRIM(actor_id), '') IS NOT NULL
				AND TRIM(actor_id) <> 'tillsyn-user'
			)
	`); err != nil {
		return fmt.Errorf("migrate sqlite backfill change_events.actor_name: %w", err)
	}
	return nil
}

// migrateTaskActorNames adds and backfills readable task actor-name columns on both task tables.
func (r *Repository) migrateTaskActorNames(ctx context.Context) error {
	statements := []struct {
		name string
		sql  string
	}{
		{name: "tasks.created_by_name", sql: `UPDATE tasks SET created_by_name = COALESCE(NULLIF(TRIM(created_by_actor), ''), 'tillsyn-user') WHERE NULLIF(TRIM(created_by_name), '') IS NULL`},
		{name: "tasks.updated_by_name", sql: `UPDATE tasks SET updated_by_name = COALESCE(NULLIF(TRIM(updated_by_actor), ''), NULLIF(TRIM(created_by_name), ''), COALESCE(NULLIF(TRIM(created_by_actor), ''), 'tillsyn-user')) WHERE NULLIF(TRIM(updated_by_name), '') IS NULL`},
		{name: "work_items.created_by_name", sql: `UPDATE work_items SET created_by_name = COALESCE(NULLIF(TRIM(created_by_actor), ''), 'tillsyn-user') WHERE NULLIF(TRIM(created_by_name), '') IS NULL`},
		{name: "work_items.updated_by_name", sql: `UPDATE work_items SET updated_by_name = COALESCE(NULLIF(TRIM(updated_by_actor), ''), NULLIF(TRIM(created_by_name), ''), COALESCE(NULLIF(TRIM(created_by_actor), ''), 'tillsyn-user')) WHERE NULLIF(TRIM(updated_by_name), '') IS NULL`},
	}
	for _, stmt := range statements {
		if _, err := r.db.ExecContext(ctx, stmt.sql); err != nil {
			return fmt.Errorf("migrate sqlite backfill %s: %w", stmt.name, err)
		}
	}
	return nil
}

// ensureCommentIndexes restores comment indexes that may be dropped during table rewrite.
func (r *Repository) ensureCommentIndexes(ctx context.Context) error {
	statements := []string{
		`CREATE INDEX IF NOT EXISTS idx_comments_project_target_created_at ON comments(project_id, target_type, target_id, created_at ASC, id ASC);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_project_created_at ON comments(project_id, created_at DESC, id DESC);`,
	}
	for _, stmt := range statements {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate sqlite comments indexes: %w", err)
		}
	}
	return nil
}

func (r *Repository) migrateTemplateLifecycle(ctx context.Context) error {
	alterStatements := []string{
		`ALTER TABLE template_libraries ADD COLUMN builtin_managed INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE template_libraries ADD COLUMN builtin_source TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE template_libraries ADD COLUMN builtin_version TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE template_libraries ADD COLUMN revision INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE template_libraries ADD COLUMN revision_digest TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE project_template_bindings ADD COLUMN library_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE project_template_bindings ADD COLUMN bound_revision INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE project_template_bindings ADD COLUMN bound_revision_digest TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE project_template_bindings ADD COLUMN bound_library_updated_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE project_template_bindings ADD COLUMN bound_library_snapshot_json TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alterStatements {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil && !isDuplicateColumnErr(err) {
			return fmt.Errorf("migrate sqlite template lifecycle: %w", err)
		}
	}
	if err := r.backfillTemplateLibraryRevisions(ctx); err != nil {
		return err
	}
	if err := r.backfillProjectTemplateBindingSnapshots(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Repository) backfillTemplateLibraryRevisions(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM template_libraries ORDER BY id ASC`)
	if err != nil {
		return fmt.Errorf("migrate sqlite list template libraries for revision backfill: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("migrate sqlite scan template library id: %w", err)
		}
		ids = append(ids, domain.NormalizeTemplateLibraryID(id))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate sqlite iterate template library ids: %w", err)
	}
	for _, id := range ids {
		library, err := r.GetTemplateLibrary(ctx, id)
		if err != nil {
			return fmt.Errorf("migrate sqlite load template library %q: %w", id, err)
		}
		revision := max(library.Revision, 1)
		digest := strings.TrimSpace(library.RevisionDigest)
		if digest == "" {
			digest = library.RevisionFingerprint()
		}
		if digest == "" {
			return fmt.Errorf("migrate sqlite compute template library digest for %q", id)
		}
		if _, err := r.db.ExecContext(ctx, `
			UPDATE template_libraries
			SET revision = ?, revision_digest = ?
			WHERE id = ?
		`, revision, digest, id); err != nil {
			return fmt.Errorf("migrate sqlite backfill template library revision for %q: %w", id, err)
		}
	}
	return nil
}

func (r *Repository) backfillProjectTemplateBindingSnapshots(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `SELECT project_id FROM project_template_bindings ORDER BY project_id ASC`)
	if err != nil {
		return fmt.Errorf("migrate sqlite list project template bindings for snapshot backfill: %w", err)
	}
	defer rows.Close()

	var projectIDs []string
	for rows.Next() {
		var projectID string
		if err := rows.Scan(&projectID); err != nil {
			return fmt.Errorf("migrate sqlite scan project binding id: %w", err)
		}
		projectIDs = append(projectIDs, strings.TrimSpace(projectID))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate sqlite iterate project template binding ids: %w", err)
	}
	for _, projectID := range projectIDs {
		binding, err := r.GetProjectTemplateBinding(ctx, projectID)
		if err != nil {
			return fmt.Errorf("migrate sqlite load project template binding for %q: %w", projectID, err)
		}
		library, err := r.GetTemplateLibrary(ctx, binding.LibraryID)
		if err != nil {
			return fmt.Errorf("migrate sqlite load bound template library %q for project %q: %w", binding.LibraryID, projectID, err)
		}
		if binding.LibraryName == "" {
			binding.LibraryName = library.Name
		}
		if binding.BoundRevision == 0 {
			binding.BoundRevision = max(library.Revision, 1)
		}
		if binding.BoundRevisionDigest == "" {
			binding.BoundRevisionDigest = library.RevisionDigest
		}
		if binding.BoundLibraryUpdatedAt.IsZero() {
			binding.BoundLibraryUpdatedAt = library.UpdatedAt.UTC()
		}
		if binding.BoundLibrarySnapshot == nil {
			binding.BoundLibrarySnapshot = &library
		}
		if err := r.UpsertProjectTemplateBinding(ctx, binding); err != nil {
			return fmt.Errorf("migrate sqlite backfill project template binding snapshot for %q: %w", projectID, err)
		}
	}
	return nil
}

// tableHasColumn reports whether one table currently contains a named column.
func (r *Repository) tableHasColumn(ctx context.Context, tableName, columnName string) (bool, error) {
	switch strings.TrimSpace(tableName) {
	case "comments", "change_events", "template_libraries", "project_template_bindings":
	default:
		return false, fmt.Errorf("unsupported sqlite table for schema introspection %q", tableName)
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	columnName = strings.TrimSpace(strings.ToLower(columnName))
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &primaryKey); err != nil {
			return false, err
		}
		if strings.TrimSpace(strings.ToLower(name)) == columnName {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

// bridgeLegacyTasksToWorkItems copies legacy task rows into canonical work_items rows.
func (r *Repository) bridgeLegacyTasksToWorkItems(ctx context.Context) error {
	// Keep migration idempotent and non-destructive so existing tasks databases remain readable.
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO work_items(
			id, project_id, parent_id, kind, scope, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
			metadata_json, created_by_actor, created_by_name, updated_by_actor, updated_by_name, updated_by_type, created_at, updated_at, started_at, completed_at, archived_at, canceled_at
		)
		SELECT
			t.id,
			t.project_id,
			t.parent_id,
			t.kind,
			t.scope,
			t.lifecycle_state,
			t.column_id,
			t.position,
			t.title,
			t.description,
			t.priority,
			t.due_at,
			t.labels_json,
			t.metadata_json,
			t.created_by_actor,
			COALESCE(NULLIF(TRIM(t.created_by_name), ''), NULLIF(TRIM(t.created_by_actor), ''), 'tillsyn-user'),
			t.updated_by_actor,
			COALESCE(NULLIF(TRIM(t.updated_by_name), ''), NULLIF(TRIM(t.updated_by_actor), ''), COALESCE(NULLIF(TRIM(t.created_by_name), ''), NULLIF(TRIM(t.created_by_actor), ''), 'tillsyn-user')),
			t.updated_by_type,
			t.created_at,
			t.updated_at,
			t.started_at,
			t.completed_at,
			t.archived_at,
			t.canceled_at
		FROM tasks t
		WHERE NOT EXISTS (
			SELECT 1
			FROM work_items wi
			WHERE wi.id = t.id
		)
	`)
	if err != nil {
		return fmt.Errorf("bridge legacy tasks to work_items: %w", err)
	}
	return nil
}

// seedDefaultKindCatalog inserts built-in kind catalog entries when absent.
func (r *Repository) seedDefaultKindCatalog(ctx context.Context) error {
	type seedRecord struct {
		id          domain.KindID
		displayName string
		description string
		appliesTo   []domain.KindAppliesTo
		parentScope []domain.KindAppliesTo
	}
	records := []seedRecord{
		{id: domain.DefaultProjectKind, displayName: "Project", description: "Built-in project kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToProject}},
		{id: domain.KindID(domain.WorkKindTask), displayName: "Task", description: "Built-in task kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{id: domain.KindID(domain.WorkKindSubtask), displayName: "Subtask", description: "Built-in subtask kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}, parentScope: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToSubtask, domain.KindAppliesToPhase, domain.KindAppliesToBranch}},
		{id: domain.KindID(domain.WorkKindPhase), displayName: "Phase", description: "Built-in phase kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}, parentScope: []domain.KindAppliesTo{domain.KindAppliesToBranch, domain.KindAppliesToPhase}},
		{id: domain.KindID("branch"), displayName: "Branch", description: "Built-in branch kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToBranch}, parentScope: []domain.KindAppliesTo{domain.KindAppliesToBranch}},
		{id: domain.KindID(domain.WorkKindDecision), displayName: "Decision", description: "Built-in decision kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToPhase, domain.KindAppliesToSubtask}},
		{id: domain.KindID(domain.WorkKindNote), displayName: "Note", description: "Built-in note kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToPhase, domain.KindAppliesToSubtask}},
	}

	now := time.Now().UTC()
	for _, record := range records {
		appliesJSON, err := json.Marshal(record.appliesTo)
		if err != nil {
			return fmt.Errorf("encode kind applies_to for %q: %w", record.id, err)
		}
		parentJSON, err := json.Marshal(record.parentScope)
		if err != nil {
			return fmt.Errorf("encode kind parent scopes for %q: %w", record.id, err)
		}
		templateJSON, err := json.Marshal(domain.KindTemplate{})
		if err != nil {
			return fmt.Errorf("encode kind template for %q: %w", record.id, err)
		}
		_, err = r.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO kind_catalog(
				id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		`, string(record.id), record.displayName, record.description, string(appliesJSON), string(parentJSON), "", string(templateJSON), ts(now), ts(now))
		if err != nil {
			return fmt.Errorf("seed kind_catalog %q: %w", record.id, err)
		}

		existing, err := r.GetKindDefinition(ctx, record.id)
		if err != nil {
			return fmt.Errorf("load seeded kind_catalog %q: %w", record.id, err)
		}
		mergedApplies := mergeKindAppliesTo(existing.AppliesTo, record.appliesTo)
		mergedParentScopes := mergeKindAppliesTo(existing.AllowedParentScopes, record.parentScope)
		if kindAppliesToEqual(existing.AppliesTo, mergedApplies) && kindAppliesToEqual(existing.AllowedParentScopes, mergedParentScopes) {
			continue
		}

		appliesJSON, err = json.Marshal(mergedApplies)
		if err != nil {
			return fmt.Errorf("encode merged kind applies_to for %q: %w", record.id, err)
		}
		parentJSON, err = json.Marshal(mergedParentScopes)
		if err != nil {
			return fmt.Errorf("encode merged kind parent scopes for %q: %w", record.id, err)
		}
		if _, err = r.db.ExecContext(ctx, `
			UPDATE kind_catalog
			SET applies_to_json = ?, allowed_parent_scopes_json = ?, updated_at = ?
			WHERE id = ?
		`, string(appliesJSON), string(parentJSON), ts(now), string(record.id)); err != nil {
			return fmt.Errorf("update seeded kind_catalog %q: %w", record.id, err)
		}
	}
	return nil
}

// mergeKindAppliesTo appends required values into existing values without duplicates.
func mergeKindAppliesTo(existing, required []domain.KindAppliesTo) []domain.KindAppliesTo {
	out := make([]domain.KindAppliesTo, 0, len(existing)+len(required))
	seen := map[domain.KindAppliesTo]struct{}{}
	for _, raw := range existing {
		scope := domain.NormalizeKindAppliesTo(raw)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	for _, raw := range required {
		scope := domain.NormalizeKindAppliesTo(raw)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

// kindAppliesToEqual reports whether two applies_to slices are identical.
func kindAppliesToEqual(a, b []domain.KindAppliesTo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if domain.NormalizeKindAppliesTo(a[i]) != domain.NormalizeKindAppliesTo(b[i]) {
			return false
		}
	}
	return true
}

// CreateProject creates project.
func (r *Repository) CreateProject(ctx context.Context, p domain.Project) error {
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("encode project metadata: %w", err)
	}
	kindID := domain.NormalizeKindID(p.Kind)
	if kindID == "" {
		kindID = domain.DefaultProjectKind
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO projects(id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Slug, p.Name, p.Description, string(kindID), string(metaJSON), ts(p.CreatedAt), ts(p.UpdatedAt), nullableTS(p.ArchivedAt))
	return err
}

// UpdateProject updates state for the requested operation.
func (r *Repository) UpdateProject(ctx context.Context, p domain.Project) error {
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("encode project metadata: %w", err)
	}
	kindID := domain.NormalizeKindID(p.Kind)
	if kindID == "" {
		kindID = domain.DefaultProjectKind
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE projects
		SET slug = ?, name = ?, description = ?, kind = ?, metadata_json = ?, updated_at = ?, archived_at = ?
		WHERE id = ?
	`, p.Slug, p.Name, p.Description, string(kindID), string(metaJSON), ts(p.UpdatedAt), nullableTS(p.ArchivedAt), p.ID)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// DeleteProject deletes project and all dependent rows through foreign-key cascades.
func (r *Repository) DeleteProject(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM projects
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetProject returns project.
func (r *Repository) GetProject(ctx context.Context, id string) (domain.Project, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at
		FROM projects
		WHERE id = ?
	`, id)
	return scanProject(row)
}

// ListProjects lists projects.
func (r *Repository) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	query := `
		SELECT id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at
		FROM projects
	`
	if !includeArchived {
		query += ` WHERE id != ? AND archived_at IS NULL`
	} else {
		query += ` WHERE id != ?`
	}
	query += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, domain.AuthRequestGlobalProjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Project{}
	for rows.Next() {
		var (
			p           domain.Project
			kindRaw     string
			metadataRaw string
			createdRaw  string
			updatedRaw  string
			archived    sql.NullString
		)
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &kindRaw, &metadataRaw, &createdRaw, &updatedRaw, &archived); err != nil {
			return nil, err
		}
		p.Kind = domain.NormalizeKindID(domain.KindID(kindRaw))
		if p.Kind == "" {
			p.Kind = domain.DefaultProjectKind
		}
		if strings.TrimSpace(metadataRaw) == "" {
			metadataRaw = "{}"
		}
		if err := json.Unmarshal([]byte(metadataRaw), &p.Metadata); err != nil {
			return nil, fmt.Errorf("decode project metadata_json: %w", err)
		}
		p.CreatedAt = parseTS(createdRaw)
		p.UpdatedAt = parseTS(updatedRaw)
		p.ArchivedAt = parseNullTS(archived)
		out = append(out, p)
	}
	return out, rows.Err()
}

// ensureGlobalAuthProject creates the hidden project row that backs global auth requests and notifications.
func (r *Repository) ensureGlobalAuthProject(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO projects(id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at)
		VALUES (?, ?, ?, '', ?, '{}', ?, ?, NULL)
		ON CONFLICT(id) DO NOTHING
	`,
		domain.AuthRequestGlobalProjectID,
		globalAuthProjectSlug,
		globalAuthProjectName,
		string(domain.DefaultProjectKind),
		globalAuthProjectCreatedAt,
		globalAuthProjectCreatedAt,
	)
	if err != nil {
		return fmt.Errorf("ensure global auth project: %w", err)
	}
	return nil
}

// SetProjectAllowedKinds replaces one project's kind allowlist.
func (r *Repository) SetProjectAllowedKinds(ctx context.Context, projectID string, kindIDs []domain.KindID) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.ErrInvalidID
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM project_allowed_kinds WHERE project_id = ?`, projectID); err != nil {
		return err
	}

	now := ts(time.Now().UTC())
	seen := map[domain.KindID]struct{}{}
	for _, raw := range kindIDs {
		kindID := domain.NormalizeKindID(raw)
		if kindID == "" {
			continue
		}
		if _, ok := seen[kindID]; ok {
			continue
		}
		seen[kindID] = struct{}{}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO project_allowed_kinds(project_id, kind_id, created_at)
			VALUES (?, ?, ?)
		`, projectID, string(kindID), now); err != nil {
			return err
		}
	}

	err = tx.Commit()
	return err
}

// ListProjectAllowedKinds lists one project's explicit kind allowlist.
func (r *Repository) ListProjectAllowedKinds(ctx context.Context, projectID string) ([]domain.KindID, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT kind_id
		FROM project_allowed_kinds
		WHERE project_id = ?
		ORDER BY kind_id ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.KindID, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		kindID := domain.NormalizeKindID(domain.KindID(raw))
		if kindID == "" {
			continue
		}
		out = append(out, kindID)
	}
	return out, rows.Err()
}

// CreateKindDefinition creates one kind catalog entry.
func (r *Repository) CreateKindDefinition(ctx context.Context, kind domain.KindDefinition) error {
	appliesJSON, err := json.Marshal(kind.AppliesTo)
	if err != nil {
		return fmt.Errorf("encode kind applies_to_json: %w", err)
	}
	parentJSON, err := json.Marshal(kind.AllowedParentScopes)
	if err != nil {
		return fmt.Errorf("encode kind allowed_parent_scopes_json: %w", err)
	}
	templateJSON, err := json.Marshal(kind.Template)
	if err != nil {
		return fmt.Errorf("encode kind template_json: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO kind_catalog(
			id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		string(domain.NormalizeKindID(kind.ID)),
		strings.TrimSpace(kind.DisplayName),
		strings.TrimSpace(kind.DescriptionMarkdown),
		string(appliesJSON),
		string(parentJSON),
		strings.TrimSpace(kind.PayloadSchemaJSON),
		string(templateJSON),
		ts(kind.CreatedAt),
		ts(kind.UpdatedAt),
		nullableTS(kind.ArchivedAt),
	)
	return err
}

// UpdateKindDefinition updates one kind catalog entry.
func (r *Repository) UpdateKindDefinition(ctx context.Context, kind domain.KindDefinition) error {
	appliesJSON, err := json.Marshal(kind.AppliesTo)
	if err != nil {
		return fmt.Errorf("encode kind applies_to_json: %w", err)
	}
	parentJSON, err := json.Marshal(kind.AllowedParentScopes)
	if err != nil {
		return fmt.Errorf("encode kind allowed_parent_scopes_json: %w", err)
	}
	templateJSON, err := json.Marshal(kind.Template)
	if err != nil {
		return fmt.Errorf("encode kind template_json: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE kind_catalog
		SET display_name = ?, description_markdown = ?, applies_to_json = ?, allowed_parent_scopes_json = ?, payload_schema_json = ?, template_json = ?, updated_at = ?, archived_at = ?
		WHERE id = ?
	`,
		strings.TrimSpace(kind.DisplayName),
		strings.TrimSpace(kind.DescriptionMarkdown),
		string(appliesJSON),
		string(parentJSON),
		strings.TrimSpace(kind.PayloadSchemaJSON),
		string(templateJSON),
		ts(kind.UpdatedAt),
		nullableTS(kind.ArchivedAt),
		string(domain.NormalizeKindID(kind.ID)),
	)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetKindDefinition loads one kind catalog entry by id.
func (r *Repository) GetKindDefinition(ctx context.Context, kindID domain.KindID) (domain.KindDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
		FROM kind_catalog
		WHERE id = ?
	`, string(domain.NormalizeKindID(kindID)))
	return scanKindDefinition(row)
}

// ListKindDefinitions lists kind catalog entries.
func (r *Repository) ListKindDefinitions(ctx context.Context, includeArchived bool) ([]domain.KindDefinition, error) {
	query := `
		SELECT id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
		FROM kind_catalog
	`
	if !includeArchived {
		query += ` WHERE archived_at IS NULL`
	}
	query += ` ORDER BY display_name ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.KindDefinition, 0)
	for rows.Next() {
		kind, scanErr := scanKindDefinition(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, kind)
	}
	return out, rows.Err()
}

// UpsertTemplateLibrary creates or replaces one template library and all nested rules.
func (r *Repository) UpsertTemplateLibrary(ctx context.Context, library domain.TemplateLibrary) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO template_libraries(
			id, scope, project_id, name, description, status, source_library_id,
			builtin_managed, builtin_source, builtin_version, revision, revision_digest,
			created_by_actor_id, created_by_actor_name, created_by_actor_type,
			created_at, updated_at, approved_by_actor_id, approved_by_actor_name,
			approved_by_actor_type, approved_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			scope = excluded.scope,
			project_id = excluded.project_id,
			name = excluded.name,
			description = excluded.description,
			status = excluded.status,
			source_library_id = excluded.source_library_id,
			builtin_managed = excluded.builtin_managed,
			builtin_source = excluded.builtin_source,
			builtin_version = excluded.builtin_version,
			revision = excluded.revision,
			revision_digest = excluded.revision_digest,
			created_by_actor_id = excluded.created_by_actor_id,
			created_by_actor_name = excluded.created_by_actor_name,
			created_by_actor_type = excluded.created_by_actor_type,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			approved_by_actor_id = excluded.approved_by_actor_id,
			approved_by_actor_name = excluded.approved_by_actor_name,
			approved_by_actor_type = excluded.approved_by_actor_type,
			approved_at = excluded.approved_at
	`,
		library.ID,
		string(domain.NormalizeTemplateLibraryScope(library.Scope)),
		nullableString(library.ProjectID),
		strings.TrimSpace(library.Name),
		strings.TrimSpace(library.Description),
		string(domain.NormalizeTemplateLibraryStatus(library.Status)),
		nullableString(library.SourceLibraryID),
		boolToInt(library.BuiltinManaged),
		strings.TrimSpace(library.BuiltinSource),
		strings.TrimSpace(library.BuiltinVersion),
		max(library.Revision, 1),
		strings.TrimSpace(library.RevisionDigest),
		strings.TrimSpace(library.CreatedByActorID),
		strings.TrimSpace(library.CreatedByActorName),
		normalizeOptionalActorType(library.CreatedByActorType),
		ts(library.CreatedAt),
		ts(library.UpdatedAt),
		strings.TrimSpace(library.ApprovedByActorID),
		strings.TrimSpace(library.ApprovedByActorName),
		normalizeOptionalActorType(library.ApprovedByActorType),
		nullableTS(library.ApprovedAt),
	)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM template_node_templates WHERE library_id = ?`, library.ID); err != nil {
		return err
	}
	for _, nodeTemplate := range library.NodeTemplates {
		projectDefaultsJSON, err := marshalTemplateProjectMetadata(nodeTemplate.ProjectMetadataDefaults)
		if err != nil {
			return err
		}
		taskDefaultsJSON, err := marshalTemplateTaskMetadata(nodeTemplate.TaskMetadataDefaults)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO template_node_templates(
				library_id, id, scope_level, node_kind_id, display_name, description_markdown,
				project_metadata_defaults_json, task_metadata_defaults_json
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`,
			library.ID,
			nodeTemplate.ID,
			string(domain.NormalizeKindAppliesTo(nodeTemplate.ScopeLevel)),
			string(domain.NormalizeKindID(nodeTemplate.NodeKindID)),
			strings.TrimSpace(nodeTemplate.DisplayName),
			strings.TrimSpace(nodeTemplate.DescriptionMarkdown),
			projectDefaultsJSON,
			taskDefaultsJSON,
		)
		if err != nil {
			return err
		}
		for _, childRule := range nodeTemplate.ChildRules {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO template_child_rules(
					library_id, node_template_id, id, position, child_scope_level, child_kind_id,
					title_template, description_template, responsible_actor_kind, orchestrator_may_complete,
					required_for_parent_done, required_for_containing_done
				)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				library.ID,
				nodeTemplate.ID,
				childRule.ID,
				childRule.Position,
				string(domain.NormalizeKindAppliesTo(childRule.ChildScopeLevel)),
				string(domain.NormalizeKindID(childRule.ChildKindID)),
				strings.TrimSpace(childRule.TitleTemplate),
				strings.TrimSpace(childRule.DescriptionTemplate),
				string(domain.NormalizeTemplateActorKind(childRule.ResponsibleActorKind)),
				boolToInt(childRule.OrchestratorMayComplete),
				boolToInt(childRule.RequiredForParentDone),
				boolToInt(childRule.RequiredForContainingDone),
			)
			if err != nil {
				return err
			}
			if err = insertTemplateActorKinds(ctx, tx, "template_child_rule_editor_kinds", library.ID, nodeTemplate.ID, childRule.ID, childRule.EditableByActorKinds); err != nil {
				return err
			}
			if err = insertTemplateActorKinds(ctx, tx, "template_child_rule_completer_kinds", library.ID, nodeTemplate.ID, childRule.ID, childRule.CompletableByActorKinds); err != nil {
				return err
			}
		}
	}

	err = tx.Commit()
	return err
}

// GetTemplateLibrary loads one template library and all nested rules.
func (r *Repository) GetTemplateLibrary(ctx context.Context, libraryID string) (domain.TemplateLibrary, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, scope, project_id, name, description, status, source_library_id,
			builtin_managed, builtin_source, builtin_version, revision, revision_digest,
			created_by_actor_id, created_by_actor_name, created_by_actor_type,
			created_at, updated_at, approved_by_actor_id, approved_by_actor_name,
			approved_by_actor_type, approved_at
		FROM template_libraries
		WHERE id = ?
	`, domain.NormalizeTemplateLibraryID(libraryID))
	library, err := scanTemplateLibrary(row)
	if err != nil {
		return domain.TemplateLibrary{}, err
	}
	nodeTemplates, err := loadTemplateNodeTemplates(ctx, r.db, library.ID)
	if err != nil {
		return domain.TemplateLibrary{}, err
	}
	library.NodeTemplates = nodeTemplates
	return library, nil
}

// ListTemplateLibraries lists template libraries with optional filters.
func (r *Repository) ListTemplateLibraries(ctx context.Context, filter domain.TemplateLibraryFilter) ([]domain.TemplateLibrary, error) {
	query := `
		SELECT
			id, scope, project_id, name, description, status, source_library_id,
			builtin_managed, builtin_source, builtin_version, revision, revision_digest,
			created_by_actor_id, created_by_actor_name, created_by_actor_type,
			created_at, updated_at, approved_by_actor_id, approved_by_actor_name,
			approved_by_actor_type, approved_at
		FROM template_libraries
		WHERE 1 = 1
	`
	args := make([]any, 0, 3)
	if scope := domain.NormalizeTemplateLibraryScope(filter.Scope); scope != "" {
		query += ` AND scope = ?`
		args = append(args, string(scope))
	}
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}
	if status := domain.NormalizeTemplateLibraryStatus(filter.Status); status != "" {
		query += ` AND status = ?`
		args = append(args, string(status))
	}
	query += ` ORDER BY scope ASC, project_id ASC, name ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.TemplateLibrary, 0)
	for rows.Next() {
		library, err := scanTemplateLibrary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, library)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for idx := range out {
		out[idx].NodeTemplates, err = loadTemplateNodeTemplates(ctx, r.db, out[idx].ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// UpsertProjectTemplateBinding sets the active template-library binding for one project.
func (r *Repository) UpsertProjectTemplateBinding(ctx context.Context, binding domain.ProjectTemplateBinding) error {
	boundLibrarySnapshotJSON, err := marshalTemplateLibrarySnapshot(binding.BoundLibrarySnapshot)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO project_template_bindings(
			project_id, library_id, library_name, bound_revision, bound_revision_digest, bound_library_updated_at,
			bound_library_snapshot_json, bound_by_actor_id, bound_by_actor_name, bound_by_actor_type, bound_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET
			library_id = excluded.library_id,
			library_name = excluded.library_name,
			bound_revision = excluded.bound_revision,
			bound_revision_digest = excluded.bound_revision_digest,
			bound_library_updated_at = excluded.bound_library_updated_at,
			bound_library_snapshot_json = excluded.bound_library_snapshot_json,
			bound_by_actor_id = excluded.bound_by_actor_id,
			bound_by_actor_name = excluded.bound_by_actor_name,
			bound_by_actor_type = excluded.bound_by_actor_type,
			bound_at = excluded.bound_at
	`,
		strings.TrimSpace(binding.ProjectID),
		domain.NormalizeTemplateLibraryID(binding.LibraryID),
		strings.TrimSpace(binding.LibraryName),
		max(binding.BoundRevision, 1),
		strings.TrimSpace(binding.BoundRevisionDigest),
		ts(binding.BoundLibraryUpdatedAt),
		boundLibrarySnapshotJSON,
		strings.TrimSpace(binding.BoundByActorID),
		strings.TrimSpace(binding.BoundByActorName),
		normalizeOptionalActorType(binding.BoundByActorType),
		ts(binding.BoundAt),
	)
	return err
}

// GetProjectTemplateBinding loads one project's active template-library binding.
func (r *Repository) GetProjectTemplateBinding(ctx context.Context, projectID string) (domain.ProjectTemplateBinding, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT project_id, library_id, library_name, bound_revision, bound_revision_digest, bound_library_updated_at,
			bound_library_snapshot_json, bound_by_actor_id, bound_by_actor_name, bound_by_actor_type, bound_at
		FROM project_template_bindings
		WHERE project_id = ?
	`, strings.TrimSpace(projectID))
	return scanProjectTemplateBinding(row)
}

// DeleteProjectTemplateBinding removes one project's active template-library binding.
func (r *Repository) DeleteProjectTemplateBinding(ctx context.Context, projectID string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM project_template_bindings
		WHERE project_id = ?
	`, strings.TrimSpace(projectID))
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return app.ErrNotFound
	}
	return nil
}

// CreateNodeContractSnapshot persists one generated-node contract snapshot.
func (r *Repository) CreateNodeContractSnapshot(ctx context.Context, snapshot domain.NodeContractSnapshot) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO node_contract_snapshots(
			node_id, project_id, source_library_id, source_node_template_id, source_child_rule_id,
			created_by_actor_id, created_by_actor_type, responsible_actor_kind, orchestrator_may_complete,
			required_for_parent_done, required_for_containing_done, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		strings.TrimSpace(snapshot.NodeID),
		strings.TrimSpace(snapshot.ProjectID),
		domain.NormalizeTemplateLibraryID(snapshot.SourceLibraryID),
		domain.NormalizeTemplateLibraryID(snapshot.SourceNodeTemplateID),
		domain.NormalizeTemplateLibraryID(snapshot.SourceChildRuleID),
		strings.TrimSpace(snapshot.CreatedByActorID),
		normalizeOptionalActorType(snapshot.CreatedByActorType),
		string(domain.NormalizeTemplateActorKind(snapshot.ResponsibleActorKind)),
		boolToInt(snapshot.OrchestratorMayComplete),
		boolToInt(snapshot.RequiredForParentDone),
		boolToInt(snapshot.RequiredForContainingDone),
		ts(snapshot.CreatedAt),
	)
	if err != nil {
		return err
	}
	if err = insertNodeContractActorKinds(ctx, tx, "node_contract_editor_kinds", snapshot.NodeID, snapshot.EditableByActorKinds); err != nil {
		return err
	}
	if err = insertNodeContractActorKinds(ctx, tx, "node_contract_completer_kinds", snapshot.NodeID, snapshot.CompletableByActorKinds); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// GetNodeContractSnapshot loads one generated-node contract snapshot.
func (r *Repository) GetNodeContractSnapshot(ctx context.Context, nodeID string) (domain.NodeContractSnapshot, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			node_id, project_id, source_library_id, source_node_template_id, source_child_rule_id,
			created_by_actor_id, created_by_actor_type, responsible_actor_kind, orchestrator_may_complete,
			required_for_parent_done, required_for_containing_done, created_at
		FROM node_contract_snapshots
		WHERE node_id = ?
	`, strings.TrimSpace(nodeID))
	snapshot, err := scanNodeContractSnapshot(row)
	if err != nil {
		return domain.NodeContractSnapshot{}, err
	}
	snapshot.EditableByActorKinds, err = loadNodeContractActorKinds(ctx, r.db, "node_contract_editor_kinds", snapshot.NodeID)
	if err != nil {
		return domain.NodeContractSnapshot{}, err
	}
	snapshot.CompletableByActorKinds, err = loadNodeContractActorKinds(ctx, r.db, "node_contract_completer_kinds", snapshot.NodeID)
	if err != nil {
		return domain.NodeContractSnapshot{}, err
	}
	return snapshot, nil
}

// CreateColumn creates column.
func (r *Repository) CreateColumn(ctx context.Context, c domain.Column) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO columns_v1(id, project_id, name, wip_limit, position, created_at, updated_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.ProjectID, c.Name, c.WIPLimit, c.Position, ts(c.CreatedAt), ts(c.UpdatedAt), nullableTS(c.ArchivedAt))
	return err
}

// UpdateColumn updates state for the requested operation.
func (r *Repository) UpdateColumn(ctx context.Context, c domain.Column) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE columns_v1
		SET name = ?, wip_limit = ?, position = ?, updated_at = ?, archived_at = ?
		WHERE id = ?
	`, c.Name, c.WIPLimit, c.Position, ts(c.UpdatedAt), nullableTS(c.ArchivedAt), c.ID)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// ListColumns lists columns.
func (r *Repository) ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	query := `
		SELECT id, project_id, name, wip_limit, position, created_at, updated_at, archived_at
		FROM columns_v1
		WHERE project_id = ?
	`
	if !includeArchived {
		query += ` AND archived_at IS NULL`
	}
	query += ` ORDER BY position ASC`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Column{}
	for rows.Next() {
		var (
			c          domain.Column
			createdRaw string
			updatedRaw string
			archived   sql.NullString
		)
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.WIPLimit, &c.Position, &createdRaw, &updatedRaw, &archived); err != nil {
			return nil, err
		}
		c.CreatedAt = parseTS(createdRaw)
		c.UpdatedAt = parseTS(updatedRaw)
		c.ArchivedAt = parseNullTS(archived)
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateTask creates task.
func (r *Repository) CreateTask(ctx context.Context, t domain.Task) error {
	labelsJSON, err := json.Marshal(t.Labels)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return err
	}

	scope := domain.NormalizeKindAppliesTo(t.Scope)
	if scope == "" {
		scope = domain.DefaultTaskScope(t.Kind, t.ParentID)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_items(
			id, project_id, parent_id, kind, scope, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
			metadata_json, created_by_actor, created_by_name, updated_by_actor, updated_by_name, updated_by_type, created_at, updated_at, started_at, completed_at, archived_at, canceled_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID,
		t.ProjectID,
		t.ParentID,
		string(t.Kind),
		string(scope),
		string(t.LifecycleState),
		t.ColumnID,
		t.Position,
		t.Title,
		t.Description,
		t.Priority,
		nullableTS(t.DueAt),
		string(labelsJSON),
		string(metadataJSON),
		t.CreatedByActor,
		t.CreatedByName,
		t.UpdatedByActor,
		t.UpdatedByName,
		string(t.UpdatedByType),
		ts(t.CreatedAt),
		ts(t.UpdatedAt),
		nullableTS(t.StartedAt),
		nullableTS(t.CompletedAt),
		nullableTS(t.ArchivedAt),
		nullableTS(t.CanceledAt),
	)
	if err != nil {
		return err
	}

	actorID, actorName, actorType := resolveChangeEventActor(ctx, t.CreatedByActor, t.CreatedByName, t.UpdatedByType, t.UpdatedByActor, t.UpdatedByName)
	err = insertTaskChangeEvent(ctx, tx, domain.ChangeEvent{
		ProjectID:  t.ProjectID,
		WorkItemID: t.ID,
		Operation:  domain.ChangeOperationCreate,
		ActorID:    actorID,
		ActorName:  actorName,
		ActorType:  actorType,
		Metadata: map[string]string{
			"column_id":  t.ColumnID,
			"position":   strconv.Itoa(t.Position),
			"title":      t.Title,
			"item_kind":  string(t.Kind),
			"item_scope": string(scope),
		},
		OccurredAt: t.CreatedAt,
	})
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// UpdateTask updates state for the requested operation.
func (r *Repository) UpdateTask(ctx context.Context, t domain.Task) error {
	labelsJSON, err := json.Marshal(t.Labels)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return err
	}

	scope := domain.NormalizeKindAppliesTo(t.Scope)
	if scope == "" {
		scope = domain.DefaultTaskScope(t.Kind, t.ParentID)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	prev, err := getTaskByID(ctx, tx, t.ID)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE work_items
		SET parent_id = ?, kind = ?, scope = ?, lifecycle_state = ?, column_id = ?, position = ?, title = ?, description = ?, priority = ?, due_at = ?,
		    labels_json = ?, metadata_json = ?, updated_by_actor = ?, updated_by_name = ?, updated_by_type = ?, updated_at = ?, started_at = ?, completed_at = ?, archived_at = ?, canceled_at = ?
		WHERE id = ?
	`,
		t.ParentID,
		string(t.Kind),
		string(scope),
		string(t.LifecycleState),
		t.ColumnID,
		t.Position,
		t.Title,
		t.Description,
		t.Priority,
		nullableTS(t.DueAt),
		string(labelsJSON),
		string(metadataJSON),
		t.UpdatedByActor,
		t.UpdatedByName,
		string(t.UpdatedByType),
		ts(t.UpdatedAt),
		nullableTS(t.StartedAt),
		nullableTS(t.CompletedAt),
		nullableTS(t.ArchivedAt),
		nullableTS(t.CanceledAt),
		t.ID,
	)
	if err != nil {
		return err
	}
	if err := translateNoRows(res); err != nil {
		return err
	}

	op, metadata := classifyTaskTransition(prev, t)
	metadata["title"] = t.Title
	metadata["item_kind"] = string(t.Kind)
	metadata["item_scope"] = string(scope)
	actorID, actorName, actorType := resolveChangeEventActor(ctx, t.UpdatedByActor, t.UpdatedByName, t.UpdatedByType, prev.UpdatedByActor, prev.UpdatedByName)
	err = insertTaskChangeEvent(ctx, tx, domain.ChangeEvent{
		ProjectID:  t.ProjectID,
		WorkItemID: t.ID,
		Operation:  op,
		ActorID:    actorID,
		ActorName:  actorName,
		ActorType:  actorType,
		Metadata:   metadata,
		OccurredAt: t.UpdatedAt,
	})
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// GetTask returns task.
func (r *Repository) GetTask(ctx context.Context, id string) (domain.Task, error) {
	return getTaskByID(ctx, r.db, id)
}

// ListTasks lists tasks.
func (r *Repository) ListTasks(ctx context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	query := `
		SELECT
			id, project_id, parent_id, kind, scope, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
			metadata_json, created_by_actor, created_by_name, updated_by_actor, updated_by_name, updated_by_type, created_at, updated_at, started_at, completed_at, archived_at, canceled_at
		FROM work_items
		WHERE project_id = ?
	`
	if !includeArchived {
		query += ` AND archived_at IS NULL`
	}
	query += ` ORDER BY column_id ASC, position ASC`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

// DeleteTask deletes task.
func (r *Repository) DeleteTask(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM work_items WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if err := translateNoRows(res); err != nil {
		return err
	}
	actorID, actorName, actorType := resolveChangeEventActor(ctx, task.UpdatedByActor, task.UpdatedByName, task.UpdatedByType, task.CreatedByActor, task.CreatedByName)

	err = insertTaskChangeEvent(ctx, tx, domain.ChangeEvent{
		ProjectID:  task.ProjectID,
		WorkItemID: task.ID,
		Operation:  domain.ChangeOperationDelete,
		ActorID:    actorID,
		ActorName:  actorName,
		ActorType:  actorType,
		Metadata: map[string]string{
			"column_id":  task.ColumnID,
			"position":   strconv.Itoa(task.Position),
			"title":      task.Title,
			"item_kind":  string(task.Kind),
			"item_scope": string(task.Scope),
		},
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// UpsertEmbeddingDocument writes one indexed subject document row for semantic retrieval.
func (r *Repository) UpsertEmbeddingDocument(ctx context.Context, in app.EmbeddingDocument) error {
	subjectType := strings.TrimSpace(string(in.SubjectType))
	subjectID := strings.TrimSpace(in.SubjectID)
	projectID := strings.TrimSpace(in.ProjectID)
	searchTargetType := strings.TrimSpace(string(in.SearchTargetType))
	searchTargetID := strings.TrimSpace(in.SearchTargetID)
	contentHash := strings.TrimSpace(in.ContentHash)
	if subjectType == "" || subjectID == "" || projectID == "" || searchTargetType == "" || searchTargetID == "" || contentHash == "" {
		return domain.ErrInvalidID
	}
	if len(in.Vector) == 0 {
		return domain.ErrInvalidID
	}
	if err := r.requireVecCapability(); err != nil {
		return err
	}
	vectorJSON, err := json.Marshal(in.Vector)
	if err != nil {
		return fmt.Errorf("marshal embedding vector: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO embedding_documents (
			subject_type, subject_id, project_id, search_target_type, search_target_id, content_hash, content, embedding, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, vec_f32(?), ?)
		ON CONFLICT(subject_type, subject_id) DO UPDATE SET
			project_id = excluded.project_id,
			search_target_type = excluded.search_target_type,
			search_target_id = excluded.search_target_id,
			content_hash = excluded.content_hash,
			content = excluded.content,
			embedding = excluded.embedding,
			updated_at = excluded.updated_at
	`, subjectType, subjectID, projectID, searchTargetType, searchTargetID, contentHash, strings.TrimSpace(in.Content), string(vectorJSON), ts(in.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert embedding document: %w", err)
	}
	return nil
}

// DeleteEmbeddingDocument deletes one subject document row by subject key.
func (r *Repository) DeleteEmbeddingDocument(ctx context.Context, subjectType app.EmbeddingSubjectType, subjectID string) error {
	return deleteEmbeddingDocument(ctx, r.db, string(subjectType), subjectID)
}

// SearchEmbeddingDocuments executes one vector similarity search query for indexed subject rows.
func (r *Repository) SearchEmbeddingDocuments(ctx context.Context, in app.EmbeddingSearchInput) ([]app.EmbeddingSearchMatch, error) {
	projectIDs := normalizedStringSet(in.ProjectIDs)
	if len(projectIDs) == 0 || len(in.Vector) == 0 {
		return []app.EmbeddingSearchMatch{}, nil
	}
	if err := r.requireVecCapability(); err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultEmbeddingSearchLimit
	}
	vectorJSON, err := json.Marshal(in.Vector)
	if err != nil {
		return nil, fmt.Errorf("marshal search vector: %w", err)
	}
	query := `
		SELECT subject_type, subject_id, search_target_type, search_target_id, (1.0 - distance) AS similarity, updated_at
		FROM (
			SELECT
				subject_type,
				subject_id,
				search_target_type,
				search_target_id,
				vec_distance_cosine(embedding, vec_f32(?)) AS distance,
				updated_at
			FROM embedding_documents
			WHERE project_id IN (` + queryPlaceholders(len(projectIDs)) + `)
	`
	args := make([]any, 0, len(projectIDs)+len(in.SubjectTypes)+len(in.SearchTargetTypes)+2)
	args = append(args, string(vectorJSON))
	for _, projectID := range projectIDs {
		args = append(args, projectID)
	}
	if subjectTypes := normalizeEmbeddingSubjectTypeSet(in.SubjectTypes); len(subjectTypes) > 0 {
		query += ` AND subject_type IN (` + queryPlaceholders(len(subjectTypes)) + `)`
		for _, subjectType := range subjectTypes {
			args = append(args, subjectType)
		}
	}
	if searchTargetTypes := normalizeEmbeddingSearchTargetTypeSet(in.SearchTargetTypes); len(searchTargetTypes) > 0 {
		query += ` AND search_target_type IN (` + queryPlaceholders(len(searchTargetTypes)) + `)`
		for _, searchTargetType := range searchTargetTypes {
			args = append(args, searchTargetType)
		}
	}
	query += `
		)
		ORDER BY distance ASC, subject_type ASC, subject_id ASC
		LIMIT ?
	`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search embedding documents: %w", err)
	}
	defer rows.Close()

	out := make([]app.EmbeddingSearchMatch, 0, limit)
	for rows.Next() {
		var (
			subjectType      string
			subjectID        string
			searchTargetType string
			searchTargetID   string
			similarity       float64
			updatedAt        string
		)
		if err := rows.Scan(&subjectType, &subjectID, &searchTargetType, &searchTargetID, &similarity, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan embedding search match: %w", err)
		}
		out = append(out, app.EmbeddingSearchMatch{
			SubjectType:      app.EmbeddingSubjectType(strings.TrimSpace(subjectType)),
			SubjectID:        strings.TrimSpace(subjectID),
			SearchTargetType: app.EmbeddingSearchTargetType(strings.TrimSpace(searchTargetType)),
			SearchTargetID:   strings.TrimSpace(searchTargetID),
			Similarity:       similarity,
			SearchedAt:       parseTS(updatedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate embedding search matches: %w", err)
	}
	return out, nil
}

func deleteEmbeddingDocument(ctx context.Context, q execer, subjectType, subjectID string) error {
	subjectType = strings.TrimSpace(strings.ToLower(subjectType))
	subjectID = strings.TrimSpace(subjectID)
	if subjectType == "" || subjectID == "" {
		return nil
	}
	_, err := q.ExecContext(ctx, `DELETE FROM embedding_documents WHERE subject_type = ? AND subject_id = ?`, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("delete embedding document: %w", err)
	}
	return nil
}

func normalizeEmbeddingSubjectTypeSet(values []app.EmbeddingSubjectType) []string {
	if len(values) == 0 {
		return nil
	}
	raw := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ToLower(string(value)))
		if normalized == "" {
			continue
		}
		raw = append(raw, normalized)
	}
	return normalizedStringSet(raw)
}

func normalizeEmbeddingSearchTargetTypeSet(values []app.EmbeddingSearchTargetType) []string {
	if len(values) == 0 {
		return nil
	}
	raw := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ToLower(string(value)))
		if normalized == "" {
			continue
		}
		raw = append(raw, normalized)
	}
	return normalizedStringSet(raw)
}

// CreateComment persists one normalized comment row.
func (r *Repository) CreateComment(ctx context.Context, comment domain.Comment) error {
	commentID := strings.TrimSpace(comment.ID)
	if commentID == "" {
		return domain.ErrInvalidID
	}

	target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  comment.ProjectID,
		TargetType: comment.TargetType,
		TargetID:   comment.TargetID,
	})
	if err != nil {
		return err
	}

	bodyMarkdown := strings.TrimSpace(comment.BodyMarkdown)
	if bodyMarkdown == "" {
		return domain.ErrInvalidBodyMarkdown
	}
	summary := commentSummaryFromBody(bodyMarkdown)

	actorID := chooseActorID(comment.ActorID, "tillsyn-user")
	actorName := chooseActorName(actorID, comment.ActorName)
	if actorName == "" {
		actorName = actorID
	}
	createdAt := comment.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := comment.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO comments(id, project_id, target_type, target_id, summary, body_markdown, actor_id, actor_name, actor_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		commentID,
		target.ProjectID,
		string(target.TargetType),
		target.TargetID,
		summary,
		bodyMarkdown,
		actorID,
		actorName,
		string(normalizeActorType(comment.ActorType)),
		ts(createdAt),
		ts(updatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

// ListCommentsByTarget lists comments for a concrete project target.
func (r *Repository) ListCommentsByTarget(ctx context.Context, target domain.CommentTarget) ([]domain.Comment, error) {
	target, err := domain.NormalizeCommentTarget(target)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, target_type, target_id, summary, body_markdown, actor_id, actor_name, actor_type, created_at, updated_at
		FROM comments
		WHERE project_id = ? AND target_type = ? AND target_id = ?
		ORDER BY created_at ASC, id ASC
	`, target.ProjectID, string(target.TargetType), target.TargetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Comment, 0)
	for rows.Next() {
		comment, scanErr := scanComment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, comment)
	}
	return out, rows.Err()
}

// ListCommentTargets lists unique comment targets for one project in deterministic order.
func (r *Repository) ListCommentTargets(ctx context.Context, projectID string) ([]domain.CommentTarget, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT project_id, target_type, target_id
		FROM comments
		WHERE project_id = ?
		ORDER BY target_type ASC, target_id ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list comment targets: %w", err)
	}
	defer rows.Close()

	out := make([]domain.CommentTarget, 0)
	for rows.Next() {
		var (
			rowProjectID string
			targetType   string
			targetID     string
		)
		if err := rows.Scan(&rowProjectID, &targetType, &targetID); err != nil {
			return nil, fmt.Errorf("scan comment target: %w", err)
		}
		target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
			ProjectID:  rowProjectID,
			TargetType: domain.CommentTargetType(targetType),
			TargetID:   targetID,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment targets: %w", err)
	}
	return out, nil
}

// ListProjectChangeEvents lists recent project events for activity-log consumption.
func (r *Repository) ListProjectChangeEvents(ctx context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, work_item_id, operation, actor_id, actor_name, actor_type, metadata_json, created_at
		FROM change_events
		WHERE project_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.ChangeEvent, 0)
	for rows.Next() {
		var (
			event       domain.ChangeEvent
			opRaw       string
			actorType   string
			metadataRaw string
			createdRaw  string
		)
		if err := rows.Scan(&event.ID, &event.ProjectID, &event.WorkItemID, &opRaw, &event.ActorID, &event.ActorName, &actorType, &metadataRaw, &createdRaw); err != nil {
			return nil, err
		}
		event.Operation = normalizeChangeOperation(opRaw)
		event.ActorID = chooseActorID(event.ActorID, "tillsyn-user")
		event.ActorName = chooseActorName(event.ActorID, event.ActorName)
		event.ActorType = normalizeActorType(domain.ActorType(actorType))
		event.OccurredAt = parseTS(createdRaw)
		if strings.TrimSpace(metadataRaw) == "" {
			metadataRaw = "{}"
		}
		if err := json.Unmarshal([]byte(metadataRaw), &event.Metadata); err != nil {
			return nil, fmt.Errorf("decode change_events.metadata_json: %w", err)
		}
		if event.Metadata == nil {
			event.Metadata = map[string]string{}
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

// CreateAttentionItem creates one scoped attention-item row.
func (r *Repository) CreateAttentionItem(ctx context.Context, item domain.AttentionItem) error {
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		return domain.ErrInvalidID
	}
	level, err := domain.NewLevelTuple(domain.LevelTupleInput{
		ProjectID: item.ProjectID,
		BranchID:  item.BranchID,
		ScopeType: item.ScopeType,
		ScopeID:   item.ScopeID,
	})
	if err != nil {
		return err
	}

	state := domain.NormalizeAttentionState(item.State)
	if state == "" {
		state = domain.AttentionStateOpen
	}
	if !domain.IsValidAttentionState(state) {
		return domain.ErrInvalidAttentionState
	}
	kind := domain.NormalizeAttentionKind(item.Kind)
	if !domain.IsValidAttentionKind(kind) {
		return domain.ErrInvalidAttentionKind
	}
	summary := strings.TrimSpace(item.Summary)
	if summary == "" {
		return domain.ErrInvalidSummary
	}

	createdBy := strings.TrimSpace(item.CreatedByActor)
	if createdBy == "" {
		createdBy = "tillsyn-user"
	}
	createdByType := normalizeActorType(item.CreatedByType)
	createdAt := item.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	ackBy := strings.TrimSpace(item.AcknowledgedByActor)
	ackByType := normalizeOptionalActorType(item.AcknowledgedByType)
	resolvedBy := strings.TrimSpace(item.ResolvedByActor)
	resolvedByType := normalizeOptionalActorType(item.ResolvedByType)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO attention_items(
			id, project_id, branch_id, scope_type, scope_id, state, kind, summary, body_markdown, requires_user_action,
			created_by_actor, created_by_type, created_at, acknowledged_by_actor, acknowledged_by_type, acknowledged_at,
			resolved_by_actor, resolved_by_type, resolved_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		level.ProjectID,
		level.BranchID,
		string(level.ScopeType),
		level.ScopeID,
		string(state),
		string(kind),
		summary,
		strings.TrimSpace(item.BodyMarkdown),
		boolToInt(item.RequiresUserAction),
		createdBy,
		string(createdByType),
		ts(createdAt),
		ackBy,
		ackByType,
		nullableTS(item.AcknowledgedAt),
		resolvedBy,
		resolvedByType,
		nullableTS(item.ResolvedAt),
	)
	if err != nil {
		return fmt.Errorf("insert attention item: %w", err)
	}
	return nil
}

// GetAttentionItem returns one attention-item row by id.
func (r *Repository) GetAttentionItem(ctx context.Context, attentionID string) (domain.AttentionItem, error) {
	return getAttentionItemByID(ctx, r.db, attentionID)
}

// ListAttentionItems lists scoped attention items in deterministic order.
func (r *Repository) ListAttentionItems(ctx context.Context, filter domain.AttentionListFilter) ([]domain.AttentionItem, error) {
	filter, err := domain.NormalizeAttentionListFilter(filter)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, project_id, branch_id, scope_type, scope_id, state, kind, summary, body_markdown, requires_user_action,
			created_by_actor, created_by_type, created_at, acknowledged_by_actor, acknowledged_by_type, acknowledged_at,
			resolved_by_actor, resolved_by_type, resolved_at
		FROM attention_items
		WHERE project_id = ? AND scope_type = ? AND scope_id = ?
	`
	args := []any{filter.ProjectID, string(filter.ScopeType), filter.ScopeID}

	if filter.UnresolvedOnly {
		query += ` AND state != ?`
		args = append(args, string(domain.AttentionStateResolved))
	}
	if len(filter.States) > 0 {
		query += ` AND state IN (` + queryPlaceholders(len(filter.States)) + `)`
		for _, state := range filter.States {
			args = append(args, string(state))
		}
	}
	if len(filter.Kinds) > 0 {
		query += ` AND kind IN (` + queryPlaceholders(len(filter.Kinds)) + `)`
		for _, kind := range filter.Kinds {
			args = append(args, string(kind))
		}
	}
	if filter.RequiresUserAction != nil {
		query += ` AND requires_user_action = ?`
		args = append(args, boolToInt(*filter.RequiresUserAction))
	}

	query += ` ORDER BY created_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.AttentionItem, 0)
	for rows.Next() {
		item, scanErr := scanAttentionItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// ResolveAttentionItem resolves one attention item by id and returns the updated row.
func (r *Repository) ResolveAttentionItem(ctx context.Context, attentionID string, resolvedBy string, resolvedByType domain.ActorType, resolvedAt time.Time) (domain.AttentionItem, error) {
	attentionID = strings.TrimSpace(attentionID)
	if attentionID == "" {
		return domain.AttentionItem{}, domain.ErrInvalidID
	}

	// Acquire the write lock up front so the read/modify/write sequence does not
	// start as a deferred transaction and then fail on lock upgrade under cross-process contention.
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return domain.AttentionItem{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	item, err := getAttentionItemByID(ctx, tx, attentionID)
	if err != nil {
		return domain.AttentionItem{}, err
	}
	if err := item.Resolve(resolvedBy, resolvedByType, resolvedAt); err != nil {
		return domain.AttentionItem{}, err
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE attention_items
		SET state = ?, resolved_by_actor = ?, resolved_by_type = ?, resolved_at = ?
		WHERE id = ?
	`, string(item.State), item.ResolvedByActor, string(normalizeActorType(item.ResolvedByType)), nullableTS(item.ResolvedAt), attentionID)
	if err != nil {
		return domain.AttentionItem{}, err
	}
	if err := translateNoRows(res); err != nil {
		return domain.AttentionItem{}, err
	}

	err = tx.Commit()
	if err != nil {
		return domain.AttentionItem{}, err
	}
	return item, nil
}

// CreateAuthRequest creates one persisted auth request row.
func (r *Repository) CreateAuthRequest(ctx context.Context, request domain.AuthRequest) error {
	phaseIDsJSON, err := json.Marshal(request.PhaseIDs)
	if err != nil {
		return fmt.Errorf("encode auth request phase ids: %w", err)
	}
	continuationJSON, err := json.Marshal(request.Continuation)
	if err != nil {
		return fmt.Errorf("encode auth request continuation: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO auth_requests(
			id, project_id, branch_id, phase_ids_json, path, scope_type, scope_id,
			principal_id, principal_type, principal_role, principal_name, client_id, client_type, client_name,
			requested_session_ttl_seconds, approved_path, approved_session_ttl_seconds, reason, continuation_json, state,
			requested_by_actor, requested_by_type, created_at, expires_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note,
			issued_session_id, issued_session_secret, issued_session_expires_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		strings.TrimSpace(request.ID),
		strings.TrimSpace(request.ProjectID),
		strings.TrimSpace(request.BranchID),
		string(phaseIDsJSON),
		strings.TrimSpace(request.Path),
		string(domain.NormalizeScopeLevel(request.ScopeType)),
		strings.TrimSpace(request.ScopeID),
		strings.TrimSpace(request.PrincipalID),
		strings.TrimSpace(request.PrincipalType),
		strings.TrimSpace(request.PrincipalRole),
		strings.TrimSpace(request.PrincipalName),
		strings.TrimSpace(request.ClientID),
		strings.TrimSpace(request.ClientType),
		strings.TrimSpace(request.ClientName),
		int(request.RequestedSessionTTL.Seconds()),
		strings.TrimSpace(request.ApprovedPath),
		int(request.ApprovedSessionTTL.Seconds()),
		strings.TrimSpace(request.Reason),
		string(continuationJSON),
		string(domain.NormalizeAuthRequestState(request.State)),
		strings.TrimSpace(request.RequestedByActor),
		string(normalizeActorType(request.RequestedByType)),
		ts(request.CreatedAt),
		ts(request.ExpiresAt),
		strings.TrimSpace(request.ResolvedByActor),
		normalizeOptionalActorType(request.ResolvedByType),
		nullableTS(request.ResolvedAt),
		strings.TrimSpace(request.ResolutionNote),
		strings.TrimSpace(request.IssuedSessionID),
		strings.TrimSpace(request.IssuedSessionSecret),
		nullableTS(request.IssuedSessionExpiresAt),
	)
	if err != nil {
		return fmt.Errorf("insert auth request: %w", err)
	}
	return nil
}

// GetAuthRequest returns one persisted auth request row by id.
func (r *Repository) GetAuthRequest(ctx context.Context, requestID string) (domain.AuthRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, project_id, branch_id, phase_ids_json, path, scope_type, scope_id,
			principal_id, principal_type, principal_role, principal_name, client_id, client_type, client_name,
			requested_session_ttl_seconds, approved_path, approved_session_ttl_seconds, reason, continuation_json, state,
			requested_by_actor, requested_by_type, created_at, expires_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note,
			issued_session_id, issued_session_secret, issued_session_expires_at
		FROM auth_requests
		WHERE id = ?
	`, strings.TrimSpace(requestID))
	return scanAuthRequest(row)
}

// ListAuthRequests lists persisted auth requests in deterministic order.
func (r *Repository) ListAuthRequests(ctx context.Context, filter domain.AuthRequestListFilter) ([]domain.AuthRequest, error) {
	filter, err := domain.NormalizeAuthRequestListFilter(filter)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT
			id, project_id, branch_id, phase_ids_json, path, scope_type, scope_id,
			principal_id, principal_type, principal_role, principal_name, client_id, client_type, client_name,
			requested_session_ttl_seconds, approved_path, approved_session_ttl_seconds, reason, continuation_json, state,
			requested_by_actor, requested_by_type, created_at, expires_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note,
			issued_session_id, issued_session_secret, issued_session_expires_at
		FROM auth_requests
		WHERE 1 = 1
	`
	args := make([]any, 0, 3)
	if filter.ProjectID != "" {
		query += ` AND project_id = ?`
		args = append(args, filter.ProjectID)
	}
	if filter.State != "" {
		query += ` AND state = ?`
		args = append(args, string(filter.State))
	}
	query += ` ORDER BY created_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.AuthRequest, 0)
	for rows.Next() {
		request, scanErr := scanAuthRequest(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, request)
	}
	return out, rows.Err()
}

// UpdateAuthRequest updates one persisted auth request row.
func (r *Repository) UpdateAuthRequest(ctx context.Context, request domain.AuthRequest) error {
	phaseIDsJSON, err := json.Marshal(request.PhaseIDs)
	if err != nil {
		return fmt.Errorf("encode auth request phase ids: %w", err)
	}
	continuationJSON, err := json.Marshal(request.Continuation)
	if err != nil {
		return fmt.Errorf("encode auth request continuation: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE auth_requests
		SET project_id = ?, branch_id = ?, phase_ids_json = ?, path = ?, scope_type = ?, scope_id = ?,
			principal_id = ?, principal_type = ?, principal_role = ?, principal_name = ?, client_id = ?, client_type = ?, client_name = ?,
			requested_session_ttl_seconds = ?, approved_path = ?, approved_session_ttl_seconds = ?, reason = ?, continuation_json = ?, state = ?,
			requested_by_actor = ?, requested_by_type = ?, created_at = ?, expires_at = ?,
			resolved_by_actor = ?, resolved_by_type = ?, resolved_at = ?, resolution_note = ?,
			issued_session_id = ?, issued_session_secret = ?, issued_session_expires_at = ?
		WHERE id = ?
	`,
		strings.TrimSpace(request.ProjectID),
		strings.TrimSpace(request.BranchID),
		string(phaseIDsJSON),
		strings.TrimSpace(request.Path),
		string(domain.NormalizeScopeLevel(request.ScopeType)),
		strings.TrimSpace(request.ScopeID),
		strings.TrimSpace(request.PrincipalID),
		strings.TrimSpace(request.PrincipalType),
		strings.TrimSpace(request.PrincipalRole),
		strings.TrimSpace(request.PrincipalName),
		strings.TrimSpace(request.ClientID),
		strings.TrimSpace(request.ClientType),
		strings.TrimSpace(request.ClientName),
		int(request.RequestedSessionTTL.Seconds()),
		strings.TrimSpace(request.ApprovedPath),
		int(request.ApprovedSessionTTL.Seconds()),
		strings.TrimSpace(request.Reason),
		string(continuationJSON),
		string(domain.NormalizeAuthRequestState(request.State)),
		strings.TrimSpace(request.RequestedByActor),
		string(normalizeActorType(request.RequestedByType)),
		ts(request.CreatedAt),
		ts(request.ExpiresAt),
		strings.TrimSpace(request.ResolvedByActor),
		normalizeOptionalActorType(request.ResolvedByType),
		nullableTS(request.ResolvedAt),
		strings.TrimSpace(request.ResolutionNote),
		strings.TrimSpace(request.IssuedSessionID),
		strings.TrimSpace(request.IssuedSessionSecret),
		nullableTS(request.IssuedSessionExpiresAt),
		strings.TrimSpace(request.ID),
	)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// CreateCapabilityLease creates one capability lease row.
func (r *Repository) CreateCapabilityLease(ctx context.Context, lease domain.CapabilityLease) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO capability_leases(
			instance_id, lease_token, agent_name, project_id, scope_type, scope_id, role, parent_instance_id, allow_equal_scope_delegation, issued_at, expires_at, heartbeat_at, revoked_at, revoke_reason
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		strings.TrimSpace(lease.InstanceID),
		strings.TrimSpace(lease.LeaseToken),
		strings.TrimSpace(lease.AgentName),
		strings.TrimSpace(lease.ProjectID),
		string(domain.NormalizeCapabilityScopeType(lease.ScopeType)),
		strings.TrimSpace(lease.ScopeID),
		string(domain.NormalizeCapabilityRole(lease.Role)),
		strings.TrimSpace(lease.ParentInstanceID),
		boolToInt(lease.AllowEqualScopeDelegation),
		ts(lease.IssuedAt),
		ts(lease.ExpiresAt),
		ts(lease.HeartbeatAt),
		nullableTS(lease.RevokedAt),
		strings.TrimSpace(lease.RevokedReason),
	)
	return err
}

// UpdateCapabilityLease updates one capability lease row.
func (r *Repository) UpdateCapabilityLease(ctx context.Context, lease domain.CapabilityLease) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE capability_leases
		SET lease_token = ?, agent_name = ?, project_id = ?, scope_type = ?, scope_id = ?, role = ?, parent_instance_id = ?, allow_equal_scope_delegation = ?, issued_at = ?, expires_at = ?, heartbeat_at = ?, revoked_at = ?, revoke_reason = ?
		WHERE instance_id = ?
	`,
		strings.TrimSpace(lease.LeaseToken),
		strings.TrimSpace(lease.AgentName),
		strings.TrimSpace(lease.ProjectID),
		string(domain.NormalizeCapabilityScopeType(lease.ScopeType)),
		strings.TrimSpace(lease.ScopeID),
		string(domain.NormalizeCapabilityRole(lease.Role)),
		strings.TrimSpace(lease.ParentInstanceID),
		boolToInt(lease.AllowEqualScopeDelegation),
		ts(lease.IssuedAt),
		ts(lease.ExpiresAt),
		ts(lease.HeartbeatAt),
		nullableTS(lease.RevokedAt),
		strings.TrimSpace(lease.RevokedReason),
		strings.TrimSpace(lease.InstanceID),
	)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetCapabilityLease returns one capability lease row by instance id.
func (r *Repository) GetCapabilityLease(ctx context.Context, instanceID string) (domain.CapabilityLease, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT instance_id, lease_token, agent_name, project_id, scope_type, scope_id, role, parent_instance_id, allow_equal_scope_delegation, issued_at, expires_at, heartbeat_at, revoked_at, revoke_reason
		FROM capability_leases
		WHERE instance_id = ?
	`, strings.TrimSpace(instanceID))
	return scanCapabilityLease(row)
}

// ListCapabilityLeasesByScope lists scope-matching leases in deterministic order.
func (r *Repository) ListCapabilityLeasesByScope(ctx context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string) ([]domain.CapabilityLease, error) {
	projectID = strings.TrimSpace(projectID)
	scopeType = domain.NormalizeCapabilityScopeType(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	query := `
		SELECT instance_id, lease_token, agent_name, project_id, scope_type, scope_id, role, parent_instance_id, allow_equal_scope_delegation, issued_at, expires_at, heartbeat_at, revoked_at, revoke_reason
		FROM capability_leases
		WHERE project_id = ? AND scope_type = ?
	`
	args := []any{projectID, string(scopeType)}
	if scopeID != "" {
		query += ` AND scope_id = ?`
		args = append(args, scopeID)
	}
	query += ` ORDER BY issued_at ASC, instance_id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.CapabilityLease, 0)
	for rows.Next() {
		lease, scanErr := scanCapabilityLease(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, lease)
	}
	return out, rows.Err()
}

// RevokeCapabilityLeasesByScope revokes all leases matching one scope filter.
func (r *Repository) RevokeCapabilityLeasesByScope(ctx context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string, revokedAt time.Time, reason string) error {
	projectID = strings.TrimSpace(projectID)
	scopeType = domain.NormalizeCapabilityScopeType(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "revoked"
	}
	query := `
		UPDATE capability_leases
		SET revoked_at = ?, revoke_reason = ?
		WHERE project_id = ? AND scope_type = ?
	`
	args := []any{ts(revokedAt.UTC()), reason, projectID, string(scopeType)}
	if scopeID != "" {
		query += ` AND scope_id = ?`
		args = append(args, scopeID)
	}
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// queryRower represents a query-only DB contract used by DB and Tx implementations.
type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// queryRowser represents a read-only DB contract used by DB and Tx implementations.
type queryRowser interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// getTaskByID returns a task using the canonical work_items table.
func getTaskByID(ctx context.Context, q queryRower, id string) (domain.Task, error) {
	row := q.QueryRowContext(ctx, `
		SELECT
			id, project_id, parent_id, kind, scope, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
			metadata_json, created_by_actor, created_by_name, updated_by_actor, updated_by_name, updated_by_type, created_at, updated_at, started_at, completed_at, archived_at, canceled_at
		FROM work_items
		WHERE id = ?
	`, id)
	return scanTask(row)
}

// getAttentionItemByID returns one attention item using the canonical attention_items table.
func getAttentionItemByID(ctx context.Context, q queryRower, attentionID string) (domain.AttentionItem, error) {
	row := q.QueryRowContext(ctx, `
		SELECT
			id, project_id, branch_id, scope_type, scope_id, state, kind, summary, body_markdown, requires_user_action,
			created_by_actor, created_by_type, created_at, acknowledged_by_actor, acknowledged_by_type, acknowledged_at,
			resolved_by_actor, resolved_by_type, resolved_at
		FROM attention_items
		WHERE id = ?
	`, strings.TrimSpace(attentionID))
	return scanAttentionItem(row)
}

// execerContext represents a write-only DB contract used by DB and Tx implementations.
type execerContext interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// insertTaskChangeEvent inserts a change-event ledger record.
func insertTaskChangeEvent(ctx context.Context, execer execerContext, event domain.ChangeEvent) error {
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("encode change event metadata: %w", err)
	}
	actorID := chooseActorID(event.ActorID, "tillsyn-user")
	actorName := chooseActorName(actorID, event.ActorName)
	_, err = execer.ExecContext(ctx, `
		INSERT INTO change_events(project_id, work_item_id, operation, actor_id, actor_name, actor_type, metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.ProjectID,
		event.WorkItemID,
		string(event.Operation),
		actorID,
		actorName,
		string(normalizeActorType(event.ActorType)),
		string(metadataJSON),
		ts(normalizeEventTS(event.OccurredAt)),
	)
	if err != nil {
		return fmt.Errorf("insert change event: %w", err)
	}
	return nil
}

// resolveChangeEventActor merges task-level attribution with any context identity metadata.
func resolveChangeEventActor(ctx context.Context, actorID, actorName string, actorType domain.ActorType, fallbacks ...string) (string, string, domain.ActorType) {
	actorID = chooseActorID(append([]string{actorID}, fallbacks...)...)
	actorName = chooseActorName(actorID, actorName)
	actorType = normalizeActorType(actorType)
	if mutationActor, ok := app.MutationActorFromContext(ctx); ok {
		actorID = chooseActorID(mutationActor.ActorID, actorID)
		actorName = chooseActorName(actorID, mutationActor.ActorName, actorName)
		actorType = normalizeActorType(mutationActor.ActorType)
	}
	return actorID, actorName, actorType
}

// classifyTaskTransition derives the best operation category and metadata for a task update.
func classifyTaskTransition(prev, next domain.Task) (domain.ChangeOperation, map[string]string) {
	if prev.ArchivedAt == nil && next.ArchivedAt != nil {
		return domain.ChangeOperationArchive, map[string]string{
			"from_state": string(prev.LifecycleState),
			"to_state":   string(next.LifecycleState),
		}
	}
	if prev.ArchivedAt != nil && next.ArchivedAt == nil {
		return domain.ChangeOperationRestore, map[string]string{
			"from_state": string(prev.LifecycleState),
			"to_state":   string(next.LifecycleState),
		}
	}
	if prev.ColumnID != next.ColumnID || prev.Position != next.Position {
		return domain.ChangeOperationMove, map[string]string{
			"from_column_id": prev.ColumnID,
			"to_column_id":   next.ColumnID,
			"from_position":  strconv.Itoa(prev.Position),
			"to_position":    strconv.Itoa(next.Position),
		}
	}
	fields := changedTaskFields(prev, next)
	metadata := map[string]string{}
	if len(fields) > 0 {
		metadata["changed_fields"] = strings.Join(fields, ",")
	}
	return domain.ChangeOperationUpdate, metadata
}

// changedTaskFields identifies a deterministic set of meaningful changes for metadata.
func changedTaskFields(prev, next domain.Task) []string {
	changed := make([]string, 0)
	if prev.ParentID != next.ParentID {
		changed = append(changed, "parent_id")
	}
	if prev.Kind != next.Kind {
		changed = append(changed, "kind")
	}
	if prev.Scope != next.Scope {
		changed = append(changed, "scope")
	}
	if prev.LifecycleState != next.LifecycleState {
		changed = append(changed, "lifecycle_state")
	}
	if prev.Title != next.Title {
		changed = append(changed, "title")
	}
	if prev.Description != next.Description {
		changed = append(changed, "description")
	}
	if prev.Priority != next.Priority {
		changed = append(changed, "priority")
	}
	if !equalNullableTimes(prev.DueAt, next.DueAt) {
		changed = append(changed, "due_at")
	}
	if !equalStringSlices(prev.Labels, next.Labels) {
		changed = append(changed, "labels")
	}
	if !equalMetadata(prev.Metadata, next.Metadata) {
		changed = append(changed, "metadata")
	}
	if prev.UpdatedByActor != next.UpdatedByActor {
		changed = append(changed, "updated_by_actor")
	}
	if prev.UpdatedByType != next.UpdatedByType {
		changed = append(changed, "updated_by_type")
	}
	if !equalNullableTimes(prev.StartedAt, next.StartedAt) {
		changed = append(changed, "started_at")
	}
	if !equalNullableTimes(prev.CompletedAt, next.CompletedAt) {
		changed = append(changed, "completed_at")
	}
	if !equalNullableTimes(prev.CanceledAt, next.CanceledAt) {
		changed = append(changed, "canceled_at")
	}
	return changed
}

// equalStringSlices compares string slices by value and order.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// equalNullableTimes compares nullable timestamps using UTC normalization.
func equalNullableTimes(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.UTC().Equal(b.UTC())
}

// equalMetadata compares normalized JSON representations of task metadata.
func equalMetadata(a, b domain.TaskMetadata) bool {
	aJSON, aErr := json.Marshal(a)
	bJSON, bErr := json.Marshal(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// chooseActorID returns the first non-empty actor id or the default local actor.
func chooseActorID(candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return "tillsyn-user"
}

// chooseActorName returns the first non-empty actor name, else the canonical actor id.
func chooseActorName(actorID string, candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return "tillsyn-user"
	}
	return actorID
}

// commentSummaryFromBody returns the first non-empty markdown line as summary text.
func commentSummaryFromBody(bodyMarkdown string) string {
	for _, line := range strings.Split(bodyMarkdown, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// normalizeActorType applies a default when actor type is unset or unsupported.
func normalizeActorType(actorType domain.ActorType) domain.ActorType {
	switch strings.TrimSpace(strings.ToLower(string(actorType))) {
	case string(domain.ActorTypeUser):
		return domain.ActorTypeUser
	case string(domain.ActorTypeAgent):
		return domain.ActorTypeAgent
	case string(domain.ActorTypeSystem):
		return domain.ActorTypeSystem
	default:
		return domain.ActorTypeUser
	}
}

// normalizeChangeOperation canonicalizes persisted operation values.
func normalizeChangeOperation(raw string) domain.ChangeOperation {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case string(domain.ChangeOperationCreate):
		return domain.ChangeOperationCreate
	case string(domain.ChangeOperationUpdate):
		return domain.ChangeOperationUpdate
	case string(domain.ChangeOperationMove):
		return domain.ChangeOperationMove
	case string(domain.ChangeOperationArchive):
		return domain.ChangeOperationArchive
	case string(domain.ChangeOperationRestore):
		return domain.ChangeOperationRestore
	case string(domain.ChangeOperationDelete):
		return domain.ChangeOperationDelete
	default:
		return domain.ChangeOperationUpdate
	}
}

// normalizeEventTS ensures event timestamps are always populated and UTC-normalized.
func normalizeEventTS(in time.Time) time.Time {
	if in.IsZero() {
		return time.Now().UTC()
	}
	return in.UTC()
}

// scanner represents scanner data used by this package.
type scanner interface {
	Scan(dest ...any) error
}

// marshalTemplateProjectMetadata encodes optional project-metadata defaults for template storage.
func marshalTemplateProjectMetadata(defaults *domain.ProjectMetadata) (string, error) {
	if defaults == nil {
		return "", nil
	}
	raw, err := json.Marshal(defaults)
	if err != nil {
		return "", fmt.Errorf("encode template project metadata defaults: %w", err)
	}
	return string(raw), nil
}

// marshalTemplateTaskMetadata encodes optional task-metadata defaults for template storage.
func marshalTemplateTaskMetadata(defaults *domain.TaskMetadata) (string, error) {
	if defaults == nil {
		return "", nil
	}
	raw, err := json.Marshal(defaults)
	if err != nil {
		return "", fmt.Errorf("encode template task metadata defaults: %w", err)
	}
	return string(raw), nil
}

func marshalTemplateLibrarySnapshot(library *domain.TemplateLibrary) (string, error) {
	if library == nil {
		return "", nil
	}
	raw, err := json.Marshal(library)
	if err != nil {
		return "", fmt.Errorf("encode template library snapshot: %w", err)
	}
	return string(raw), nil
}

func unmarshalTemplateLibrarySnapshot(raw string) *domain.TemplateLibrary {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var library domain.TemplateLibrary
	if err := json.Unmarshal([]byte(raw), &library); err != nil {
		return nil
	}
	normalized, err := domain.NewTemplateLibrary(domain.TemplateLibraryInput{
		ID:                  library.ID,
		Scope:               library.Scope,
		ProjectID:           library.ProjectID,
		Name:                library.Name,
		Description:         library.Description,
		Status:              library.Status,
		SourceLibraryID:     library.SourceLibraryID,
		BuiltinManaged:      library.BuiltinManaged,
		BuiltinSource:       library.BuiltinSource,
		BuiltinVersion:      library.BuiltinVersion,
		Revision:            max(library.Revision, 1),
		RevisionDigest:      library.RevisionDigest,
		CreatedByActorID:    library.CreatedByActorID,
		CreatedByActorName:  library.CreatedByActorName,
		CreatedByActorType:  library.CreatedByActorType,
		ApprovedByActorID:   library.ApprovedByActorID,
		ApprovedByActorName: library.ApprovedByActorName,
		ApprovedByActorType: library.ApprovedByActorType,
		ApprovedAt:          library.ApprovedAt,
		NodeTemplates:       nodeTemplateInputsFromDomain(library.NodeTemplates),
	}, library.UpdatedAt)
	if err != nil {
		return nil
	}
	normalized.CreatedAt = library.CreatedAt.UTC()
	normalized.UpdatedAt = library.UpdatedAt.UTC()
	normalized.Revision = max(library.Revision, 1)
	if strings.TrimSpace(normalized.RevisionDigest) == "" {
		normalized.RevisionDigest = normalized.RevisionFingerprint()
	}
	return &normalized
}

func nodeTemplateInputsFromDomain(in []domain.NodeTemplate) []domain.NodeTemplateInput {
	if len(in) == 0 {
		return nil
	}
	out := make([]domain.NodeTemplateInput, 0, len(in))
	for _, nodeTemplate := range in {
		input := domain.NodeTemplateInput{
			ID:                  nodeTemplate.ID,
			ScopeLevel:          nodeTemplate.ScopeLevel,
			NodeKindID:          nodeTemplate.NodeKindID,
			DisplayName:         nodeTemplate.DisplayName,
			DescriptionMarkdown: nodeTemplate.DescriptionMarkdown,
		}
		if nodeTemplate.ProjectMetadataDefaults != nil {
			projectDefaults := *nodeTemplate.ProjectMetadataDefaults
			input.ProjectMetadataDefaults = &projectDefaults
		}
		if nodeTemplate.TaskMetadataDefaults != nil {
			taskDefaults := *nodeTemplate.TaskMetadataDefaults
			input.TaskMetadataDefaults = &taskDefaults
		}
		if len(nodeTemplate.ChildRules) > 0 {
			input.ChildRules = make([]domain.TemplateChildRuleInput, 0, len(nodeTemplate.ChildRules))
			for _, childRule := range nodeTemplate.ChildRules {
				input.ChildRules = append(input.ChildRules, domain.TemplateChildRuleInput{
					ID:                        childRule.ID,
					Position:                  childRule.Position,
					ChildScopeLevel:           childRule.ChildScopeLevel,
					ChildKindID:               childRule.ChildKindID,
					TitleTemplate:             childRule.TitleTemplate,
					DescriptionTemplate:       childRule.DescriptionTemplate,
					ResponsibleActorKind:      childRule.ResponsibleActorKind,
					EditableByActorKinds:      append([]domain.TemplateActorKind(nil), childRule.EditableByActorKinds...),
					CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), childRule.CompletableByActorKinds...),
					OrchestratorMayComplete:   childRule.OrchestratorMayComplete,
					RequiredForParentDone:     childRule.RequiredForParentDone,
					RequiredForContainingDone: childRule.RequiredForContainingDone,
				})
			}
		}
		out = append(out, input)
	}
	return out
}

// insertTemplateActorKinds persists one template child-rule actor-kind list.
func insertTemplateActorKinds(ctx context.Context, execer execerContext, table, libraryID, nodeTemplateID, childRuleID string, actorKinds []domain.TemplateActorKind) error {
	for _, actorKind := range actorKinds {
		normalized := domain.NormalizeTemplateActorKind(actorKind)
		if normalized == "" {
			continue
		}
		if _, err := execer.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s(library_id, node_template_id, child_rule_id, actor_kind)
			VALUES (?, ?, ?, ?)
		`, table), libraryID, nodeTemplateID, childRuleID, string(normalized)); err != nil {
			return err
		}
	}
	return nil
}

// insertNodeContractActorKinds persists one node-contract actor-kind list.
func insertNodeContractActorKinds(ctx context.Context, execer execerContext, table, nodeID string, actorKinds []domain.TemplateActorKind) error {
	for _, actorKind := range actorKinds {
		normalized := domain.NormalizeTemplateActorKind(actorKind)
		if normalized == "" {
			continue
		}
		if _, err := execer.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s(node_id, actor_kind)
			VALUES (?, ?)
		`, table), strings.TrimSpace(nodeID), string(normalized)); err != nil {
			return err
		}
	}
	return nil
}

// loadTemplateNodeTemplates loads one library's node templates and nested child rules.
func loadTemplateNodeTemplates(ctx context.Context, q queryRowser, libraryID string) ([]domain.NodeTemplate, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			id, scope_level, node_kind_id, display_name, description_markdown,
			project_metadata_defaults_json, task_metadata_defaults_json
		FROM template_node_templates
		WHERE library_id = ?
		ORDER BY scope_level ASC, node_kind_id ASC, id ASC
	`, domain.NormalizeTemplateLibraryID(libraryID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.NodeTemplate, 0)
	for rows.Next() {
		var (
			nodeTemplate       domain.NodeTemplate
			scopeLevelRaw      string
			nodeKindIDRaw      string
			projectDefaultsRaw string
			taskDefaultsRaw    string
		)
		if err := rows.Scan(
			&nodeTemplate.ID,
			&scopeLevelRaw,
			&nodeKindIDRaw,
			&nodeTemplate.DisplayName,
			&nodeTemplate.DescriptionMarkdown,
			&projectDefaultsRaw,
			&taskDefaultsRaw,
		); err != nil {
			return nil, err
		}
		nodeTemplate.LibraryID = domain.NormalizeTemplateLibraryID(libraryID)
		nodeTemplate.ScopeLevel = domain.NormalizeKindAppliesTo(domain.KindAppliesTo(scopeLevelRaw))
		nodeTemplate.NodeKindID = domain.NormalizeKindID(domain.KindID(nodeKindIDRaw))
		if strings.TrimSpace(projectDefaultsRaw) != "" {
			var projectDefaults domain.ProjectMetadata
			if err := json.Unmarshal([]byte(projectDefaultsRaw), &projectDefaults); err != nil {
				return nil, fmt.Errorf("decode template project_metadata_defaults_json: %w", err)
			}
			nodeTemplate.ProjectMetadataDefaults = &projectDefaults
		}
		if strings.TrimSpace(taskDefaultsRaw) != "" {
			var taskDefaults domain.TaskMetadata
			if err := json.Unmarshal([]byte(taskDefaultsRaw), &taskDefaults); err != nil {
				return nil, fmt.Errorf("decode template task_metadata_defaults_json: %w", err)
			}
			nodeTemplate.TaskMetadataDefaults = &taskDefaults
		}
		out = append(out, nodeTemplate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for idx := range out {
		out[idx].ChildRules, err = loadTemplateChildRules(ctx, q, libraryID, out[idx].ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// loadTemplateChildRules loads one node template's child rules and actor lists.
func loadTemplateChildRules(ctx context.Context, q queryRowser, libraryID, nodeTemplateID string) ([]domain.TemplateChildRule, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			id, position, child_scope_level, child_kind_id, title_template, description_template,
			responsible_actor_kind, orchestrator_may_complete, required_for_parent_done, required_for_containing_done
		FROM template_child_rules
		WHERE library_id = ? AND node_template_id = ?
		ORDER BY position ASC, id ASC
	`, domain.NormalizeTemplateLibraryID(libraryID), domain.NormalizeTemplateLibraryID(nodeTemplateID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.TemplateChildRule, 0)
	for rows.Next() {
		var (
			childRule             domain.TemplateChildRule
			childScopeLevelRaw    string
			childKindIDRaw        string
			responsibleActorRaw   string
			orchestratorMayRaw    int
			requiredParentRaw     int
			requiredContainingRaw int
		)
		if err := rows.Scan(
			&childRule.ID,
			&childRule.Position,
			&childScopeLevelRaw,
			&childKindIDRaw,
			&childRule.TitleTemplate,
			&childRule.DescriptionTemplate,
			&responsibleActorRaw,
			&orchestratorMayRaw,
			&requiredParentRaw,
			&requiredContainingRaw,
		); err != nil {
			return nil, err
		}
		childRule.NodeTemplateID = domain.NormalizeTemplateLibraryID(nodeTemplateID)
		childRule.ChildScopeLevel = domain.NormalizeKindAppliesTo(domain.KindAppliesTo(childScopeLevelRaw))
		childRule.ChildKindID = domain.NormalizeKindID(domain.KindID(childKindIDRaw))
		childRule.ResponsibleActorKind = domain.NormalizeTemplateActorKind(domain.TemplateActorKind(responsibleActorRaw))
		childRule.OrchestratorMayComplete = orchestratorMayRaw != 0
		childRule.RequiredForParentDone = requiredParentRaw != 0
		childRule.RequiredForContainingDone = requiredContainingRaw != 0
		out = append(out, childRule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for idx := range out {
		out[idx].EditableByActorKinds, err = loadTemplateChildRuleActorKinds(ctx, q, "template_child_rule_editor_kinds", libraryID, nodeTemplateID, out[idx].ID)
		if err != nil {
			return nil, err
		}
		out[idx].CompletableByActorKinds, err = loadTemplateChildRuleActorKinds(ctx, q, "template_child_rule_completer_kinds", libraryID, nodeTemplateID, out[idx].ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// loadTemplateChildRuleActorKinds loads one child rule's actor-kind list.
func loadTemplateChildRuleActorKinds(ctx context.Context, q queryRowser, table, libraryID, nodeTemplateID, childRuleID string) ([]domain.TemplateActorKind, error) {
	rows, err := q.QueryContext(ctx, fmt.Sprintf(`
		SELECT actor_kind
		FROM %s
		WHERE library_id = ? AND node_template_id = ? AND child_rule_id = ?
		ORDER BY actor_kind ASC
	`, table), domain.NormalizeTemplateLibraryID(libraryID), domain.NormalizeTemplateLibraryID(nodeTemplateID), domain.NormalizeTemplateLibraryID(childRuleID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.TemplateActorKind, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		normalized := domain.NormalizeTemplateActorKind(domain.TemplateActorKind(raw))
		if normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out, rows.Err()
}

// loadNodeContractActorKinds loads one node-contract actor-kind list.
func loadNodeContractActorKinds(ctx context.Context, q queryRowser, table, nodeID string) ([]domain.TemplateActorKind, error) {
	rows, err := q.QueryContext(ctx, fmt.Sprintf(`
		SELECT actor_kind
		FROM %s
		WHERE node_id = ?
		ORDER BY actor_kind ASC
	`, table), strings.TrimSpace(nodeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.TemplateActorKind, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		normalized := domain.NormalizeTemplateActorKind(domain.TemplateActorKind(raw))
		if normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out, rows.Err()
}

// scanProject handles scan project.
func scanProject(s scanner) (domain.Project, error) {
	var (
		p           domain.Project
		kindRaw     string
		metadataRaw string
		createdRaw  string
		updatedRaw  string
		archived    sql.NullString
	)
	if err := s.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &kindRaw, &metadataRaw, &createdRaw, &updatedRaw, &archived); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Project{}, app.ErrNotFound
		}
		return domain.Project{}, err
	}
	p.Kind = domain.NormalizeKindID(domain.KindID(kindRaw))
	if p.Kind == "" {
		p.Kind = domain.DefaultProjectKind
	}
	if strings.TrimSpace(metadataRaw) == "" {
		metadataRaw = "{}"
	}
	if err := json.Unmarshal([]byte(metadataRaw), &p.Metadata); err != nil {
		return domain.Project{}, fmt.Errorf("decode project metadata_json: %w", err)
	}
	p.CreatedAt = parseTS(createdRaw)
	p.UpdatedAt = parseTS(updatedRaw)
	p.ArchivedAt = parseNullTS(archived)
	return p, nil
}

// scanTask handles scan task.
func scanTask(s scanner) (domain.Task, error) {
	var (
		t            domain.Task
		dueRaw       sql.NullString
		labelsRaw    string
		metadataRaw  string
		createdRaw   string
		updatedRaw   string
		startedRaw   sql.NullString
		completedRaw sql.NullString
		archivedRaw  sql.NullString
		canceledRaw  sql.NullString
		priority     string
		kind         string
		scopeRaw     string
		state        string
		updatedType  string
	)
	if err := s.Scan(
		&t.ID,
		&t.ProjectID,
		&t.ParentID,
		&kind,
		&scopeRaw,
		&state,
		&t.ColumnID,
		&t.Position,
		&t.Title,
		&t.Description,
		&priority,
		&dueRaw,
		&labelsRaw,
		&metadataRaw,
		&t.CreatedByActor,
		&t.CreatedByName,
		&t.UpdatedByActor,
		&t.UpdatedByName,
		&updatedType,
		&createdRaw,
		&updatedRaw,
		&startedRaw,
		&completedRaw,
		&archivedRaw,
		&canceledRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, app.ErrNotFound
		}
		return domain.Task{}, err
	}
	t.Priority = domain.Priority(priority)
	t.Kind = domain.WorkKind(kind)
	t.Scope = domain.NormalizeKindAppliesTo(domain.KindAppliesTo(scopeRaw))
	t.LifecycleState = domain.LifecycleState(state)
	t.UpdatedByType = domain.ActorType(updatedType)
	t.CreatedAt = parseTS(createdRaw)
	t.UpdatedAt = parseTS(updatedRaw)
	t.StartedAt = parseNullTS(startedRaw)
	t.CompletedAt = parseNullTS(completedRaw)
	t.ArchivedAt = parseNullTS(archivedRaw)
	t.CanceledAt = parseNullTS(canceledRaw)
	t.DueAt = parseNullTS(dueRaw)
	if strings.TrimSpace(metadataRaw) == "" {
		metadataRaw = "{}"
	}
	if err := json.Unmarshal([]byte(metadataRaw), &t.Metadata); err != nil {
		return domain.Task{}, fmt.Errorf("decode metadata_json: %w", err)
	}
	if err := json.Unmarshal([]byte(labelsRaw), &t.Labels); err != nil {
		return domain.Task{}, fmt.Errorf("decode labels_json: %w", err)
	}
	if strings.TrimSpace(string(t.Kind)) == "" {
		t.Kind = domain.WorkKindTask
	}
	if t.Scope == "" {
		if strings.TrimSpace(t.ParentID) == "" {
			t.Scope = domain.KindAppliesToTask
		} else {
			t.Scope = domain.KindAppliesToSubtask
		}
	}
	if t.LifecycleState == "" {
		t.LifecycleState = domain.StateTodo
	}
	if strings.TrimSpace(t.CreatedByActor) == "" {
		t.CreatedByActor = "tillsyn-user"
	}
	if strings.TrimSpace(t.CreatedByName) == "" {
		t.CreatedByName = t.CreatedByActor
	}
	if strings.TrimSpace(t.UpdatedByActor) == "" {
		t.UpdatedByActor = t.CreatedByActor
	}
	if strings.TrimSpace(t.UpdatedByName) == "" {
		t.UpdatedByName = t.CreatedByName
	}
	if t.UpdatedByType == "" {
		t.UpdatedByType = domain.ActorTypeUser
	}
	return t, nil
}

// scanComment scans one comments row into a domain.Comment.
func scanComment(s scanner) (domain.Comment, error) {
	var (
		comment       domain.Comment
		targetTypeRaw string
		summaryRaw    string
		actorTypeRaw  string
		createdRaw    string
		updatedRaw    string
	)
	if err := s.Scan(
		&comment.ID,
		&comment.ProjectID,
		&targetTypeRaw,
		&comment.TargetID,
		&summaryRaw,
		&comment.BodyMarkdown,
		&comment.ActorID,
		&comment.ActorName,
		&actorTypeRaw,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Comment{}, app.ErrNotFound
		}
		return domain.Comment{}, err
	}
	comment.TargetType = domain.NormalizeCommentTargetType(domain.CommentTargetType(targetTypeRaw))
	if !domain.IsValidCommentTargetType(comment.TargetType) {
		return domain.Comment{}, fmt.Errorf("decode comment target_type %q: %w", targetTypeRaw, domain.ErrInvalidTargetType)
	}
	comment.ActorType = normalizeActorType(domain.ActorType(actorTypeRaw))
	comment.ActorID = chooseActorID(comment.ActorID, "tillsyn-user")
	comment.ActorName = chooseActorName(comment.ActorID, comment.ActorName)
	comment.BodyMarkdown = strings.TrimSpace(comment.BodyMarkdown)
	if comment.BodyMarkdown == "" {
		comment.BodyMarkdown = strings.TrimSpace(summaryRaw)
	}
	comment.CreatedAt = parseTS(createdRaw)
	comment.UpdatedAt = parseTS(updatedRaw)
	return comment, nil
}

// scanKindDefinition decodes one kind_catalog row.
func scanKindDefinition(s scanner) (domain.KindDefinition, error) {
	var (
		kind            domain.KindDefinition
		idRaw           string
		appliesRaw      string
		parentScopesRaw string
		templateRaw     string
		createdRaw      string
		updatedRaw      string
		archivedRaw     sql.NullString
	)
	if err := s.Scan(
		&idRaw,
		&kind.DisplayName,
		&kind.DescriptionMarkdown,
		&appliesRaw,
		&parentScopesRaw,
		&kind.PayloadSchemaJSON,
		&templateRaw,
		&createdRaw,
		&updatedRaw,
		&archivedRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.KindDefinition{}, app.ErrNotFound
		}
		return domain.KindDefinition{}, err
	}
	kind.ID = domain.NormalizeKindID(domain.KindID(idRaw))
	if kind.ID == "" {
		return domain.KindDefinition{}, domain.ErrInvalidKindID
	}
	if strings.TrimSpace(appliesRaw) == "" {
		appliesRaw = "[]"
	}
	if err := json.Unmarshal([]byte(appliesRaw), &kind.AppliesTo); err != nil {
		return domain.KindDefinition{}, fmt.Errorf("decode applies_to_json: %w", err)
	}
	if strings.TrimSpace(parentScopesRaw) == "" {
		parentScopesRaw = "[]"
	}
	if err := json.Unmarshal([]byte(parentScopesRaw), &kind.AllowedParentScopes); err != nil {
		return domain.KindDefinition{}, fmt.Errorf("decode allowed_parent_scopes_json: %w", err)
	}
	if strings.TrimSpace(templateRaw) == "" {
		templateRaw = "{}"
	}
	if err := json.Unmarshal([]byte(templateRaw), &kind.Template); err != nil {
		return domain.KindDefinition{}, fmt.Errorf("decode template_json: %w", err)
	}
	kind.CreatedAt = parseTS(createdRaw)
	kind.UpdatedAt = parseTS(updatedRaw)
	kind.ArchivedAt = parseNullTS(archivedRaw)
	return kind, nil
}

// scanTemplateLibrary decodes one template_libraries row.
func scanTemplateLibrary(s scanner) (domain.TemplateLibrary, error) {
	var (
		library              domain.TemplateLibrary
		scopeRaw             string
		projectIDRaw         sql.NullString
		statusRaw            string
		sourceLibraryIDRaw   sql.NullString
		builtinManagedRaw    int
		createdActorTypeRaw  string
		createdRaw           string
		updatedRaw           string
		approvedActorTypeRaw string
		approvedAtRaw        sql.NullString
	)
	if err := s.Scan(
		&library.ID,
		&scopeRaw,
		&projectIDRaw,
		&library.Name,
		&library.Description,
		&statusRaw,
		&sourceLibraryIDRaw,
		&builtinManagedRaw,
		&library.BuiltinSource,
		&library.BuiltinVersion,
		&library.Revision,
		&library.RevisionDigest,
		&library.CreatedByActorID,
		&library.CreatedByActorName,
		&createdActorTypeRaw,
		&createdRaw,
		&updatedRaw,
		&library.ApprovedByActorID,
		&library.ApprovedByActorName,
		&approvedActorTypeRaw,
		&approvedAtRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TemplateLibrary{}, app.ErrNotFound
		}
		return domain.TemplateLibrary{}, err
	}
	library.ID = domain.NormalizeTemplateLibraryID(library.ID)
	library.Scope = domain.NormalizeTemplateLibraryScope(domain.TemplateLibraryScope(scopeRaw))
	library.ProjectID = strings.TrimSpace(projectIDRaw.String)
	library.Status = domain.NormalizeTemplateLibraryStatus(domain.TemplateLibraryStatus(statusRaw))
	library.SourceLibraryID = domain.NormalizeTemplateLibraryID(sourceLibraryIDRaw.String)
	library.BuiltinManaged = builtinManagedRaw != 0
	library.Revision = max(library.Revision, 1)
	library.RevisionDigest = strings.TrimSpace(library.RevisionDigest)
	library.CreatedByActorType = domain.ActorType(normalizeOptionalActorType(domain.ActorType(createdActorTypeRaw)))
	library.CreatedAt = parseTS(createdRaw)
	library.UpdatedAt = parseTS(updatedRaw)
	library.ApprovedByActorType = domain.ActorType(normalizeOptionalActorType(domain.ActorType(approvedActorTypeRaw)))
	library.ApprovedAt = parseNullTS(approvedAtRaw)
	return library, nil
}

// scanProjectTemplateBinding decodes one project_template_bindings row.
func scanProjectTemplateBinding(s scanner) (domain.ProjectTemplateBinding, error) {
	var (
		binding                  domain.ProjectTemplateBinding
		actorTypeRaw             string
		boundLibraryUpdatedAtRaw string
		boundLibrarySnapshotJSON string
		boundAtRaw               string
	)
	if err := s.Scan(
		&binding.ProjectID,
		&binding.LibraryID,
		&binding.LibraryName,
		&binding.BoundRevision,
		&binding.BoundRevisionDigest,
		&boundLibraryUpdatedAtRaw,
		&boundLibrarySnapshotJSON,
		&binding.BoundByActorID,
		&binding.BoundByActorName,
		&actorTypeRaw,
		&boundAtRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ProjectTemplateBinding{}, app.ErrNotFound
		}
		return domain.ProjectTemplateBinding{}, err
	}
	binding.LibraryID = domain.NormalizeTemplateLibraryID(binding.LibraryID)
	binding.BoundRevision = max(binding.BoundRevision, 1)
	binding.BoundRevisionDigest = strings.TrimSpace(binding.BoundRevisionDigest)
	binding.BoundLibraryUpdatedAt = parseTS(boundLibraryUpdatedAtRaw)
	binding.BoundLibrarySnapshot = unmarshalTemplateLibrarySnapshot(boundLibrarySnapshotJSON)
	if normalized := normalizeOptionalActorType(domain.ActorType(actorTypeRaw)); normalized != "" {
		binding.BoundByActorType = domain.ActorType(normalized)
	} else {
		binding.BoundByActorType = domain.ActorTypeUser
	}
	binding.BoundAt = parseTS(boundAtRaw)
	return binding, nil
}

// scanNodeContractSnapshot decodes one node_contract_snapshots row.
func scanNodeContractSnapshot(s scanner) (domain.NodeContractSnapshot, error) {
	var (
		snapshot              domain.NodeContractSnapshot
		createdActorTypeRaw   string
		responsibleActorRaw   string
		orchestratorMayRaw    int
		requiredParentRaw     int
		requiredContainingRaw int
		createdAtRaw          string
	)
	if err := s.Scan(
		&snapshot.NodeID,
		&snapshot.ProjectID,
		&snapshot.SourceLibraryID,
		&snapshot.SourceNodeTemplateID,
		&snapshot.SourceChildRuleID,
		&snapshot.CreatedByActorID,
		&createdActorTypeRaw,
		&responsibleActorRaw,
		&orchestratorMayRaw,
		&requiredParentRaw,
		&requiredContainingRaw,
		&createdAtRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NodeContractSnapshot{}, app.ErrNotFound
		}
		return domain.NodeContractSnapshot{}, err
	}
	snapshot.SourceLibraryID = domain.NormalizeTemplateLibraryID(snapshot.SourceLibraryID)
	snapshot.SourceNodeTemplateID = domain.NormalizeTemplateLibraryID(snapshot.SourceNodeTemplateID)
	snapshot.SourceChildRuleID = domain.NormalizeTemplateLibraryID(snapshot.SourceChildRuleID)
	if normalized := normalizeOptionalActorType(domain.ActorType(createdActorTypeRaw)); normalized != "" {
		snapshot.CreatedByActorType = domain.ActorType(normalized)
	} else {
		snapshot.CreatedByActorType = domain.ActorTypeSystem
	}
	snapshot.ResponsibleActorKind = domain.NormalizeTemplateActorKind(domain.TemplateActorKind(responsibleActorRaw))
	snapshot.OrchestratorMayComplete = orchestratorMayRaw != 0
	snapshot.RequiredForParentDone = requiredParentRaw != 0
	snapshot.RequiredForContainingDone = requiredContainingRaw != 0
	snapshot.CreatedAt = parseTS(createdAtRaw)
	return snapshot, nil
}

// scanCapabilityLease decodes one capability_leases row.
func scanCapabilityLease(s scanner) (domain.CapabilityLease, error) {
	var (
		lease         domain.CapabilityLease
		scopeTypeRaw  string
		roleRaw       string
		allowEqualRaw int
		issuedRaw     string
		expiresRaw    string
		heartbeatRaw  string
		revokedRaw    sql.NullString
	)
	if err := s.Scan(
		&lease.InstanceID,
		&lease.LeaseToken,
		&lease.AgentName,
		&lease.ProjectID,
		&scopeTypeRaw,
		&lease.ScopeID,
		&roleRaw,
		&lease.ParentInstanceID,
		&allowEqualRaw,
		&issuedRaw,
		&expiresRaw,
		&heartbeatRaw,
		&revokedRaw,
		&lease.RevokedReason,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CapabilityLease{}, app.ErrNotFound
		}
		return domain.CapabilityLease{}, err
	}
	lease.ScopeType = domain.NormalizeCapabilityScopeType(domain.CapabilityScopeType(scopeTypeRaw))
	lease.Role = domain.NormalizeCapabilityRole(domain.CapabilityRole(roleRaw))
	lease.AllowEqualScopeDelegation = allowEqualRaw != 0
	lease.IssuedAt = parseTS(issuedRaw)
	lease.ExpiresAt = parseTS(expiresRaw)
	lease.HeartbeatAt = parseTS(heartbeatRaw)
	lease.RevokedAt = parseNullTS(revokedRaw)
	return lease, nil
}

// scanAttentionItem decodes one attention_items row.
func scanAttentionItem(s scanner) (domain.AttentionItem, error) {
	var (
		item               domain.AttentionItem
		scopeTypeRaw       string
		stateRaw           string
		kindRaw            string
		requiresUserAction int
		createdByTypeRaw   string
		createdRaw         string
		ackByTypeRaw       string
		ackAtRaw           sql.NullString
		resolvedByTypeRaw  string
		resolvedAtRaw      sql.NullString
	)
	if err := s.Scan(
		&item.ID,
		&item.ProjectID,
		&item.BranchID,
		&scopeTypeRaw,
		&item.ScopeID,
		&stateRaw,
		&kindRaw,
		&item.Summary,
		&item.BodyMarkdown,
		&requiresUserAction,
		&item.CreatedByActor,
		&createdByTypeRaw,
		&createdRaw,
		&item.AcknowledgedByActor,
		&ackByTypeRaw,
		&ackAtRaw,
		&item.ResolvedByActor,
		&resolvedByTypeRaw,
		&resolvedAtRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AttentionItem{}, app.ErrNotFound
		}
		return domain.AttentionItem{}, err
	}

	item.ScopeType = domain.NormalizeScopeLevel(domain.ScopeLevel(scopeTypeRaw))
	item.State = domain.NormalizeAttentionState(domain.AttentionState(stateRaw))
	item.Kind = domain.NormalizeAttentionKind(domain.AttentionKind(kindRaw))
	item.RequiresUserAction = requiresUserAction != 0
	item.CreatedByType = normalizeActorType(domain.ActorType(createdByTypeRaw))
	item.CreatedAt = parseTS(createdRaw)
	item.AcknowledgedByType = domain.ActorType(normalizeOptionalActorType(domain.ActorType(ackByTypeRaw)))
	item.AcknowledgedAt = parseNullTS(ackAtRaw)
	item.ResolvedByType = domain.ActorType(normalizeOptionalActorType(domain.ActorType(resolvedByTypeRaw)))
	item.ResolvedAt = parseNullTS(resolvedAtRaw)

	return item, nil
}

// scanAuthRequest decodes one auth_requests row.
func scanAuthRequest(s scanner) (domain.AuthRequest, error) {
	var (
		request                   domain.AuthRequest
		phaseIDsRaw               string
		scopeTypeRaw              string
		requestedSessionTTL       int
		approvedSessionTTL        int
		continuationRaw           string
		stateRaw                  string
		requestedByTypeRaw        string
		createdRaw                string
		expiresRaw                string
		resolvedByTypeRaw         string
		resolvedAtRaw             sql.NullString
		issuedSessionExpiresAtRaw sql.NullString
	)
	if err := s.Scan(
		&request.ID,
		&request.ProjectID,
		&request.BranchID,
		&phaseIDsRaw,
		&request.Path,
		&scopeTypeRaw,
		&request.ScopeID,
		&request.PrincipalID,
		&request.PrincipalType,
		&request.PrincipalRole,
		&request.PrincipalName,
		&request.ClientID,
		&request.ClientType,
		&request.ClientName,
		&requestedSessionTTL,
		&request.ApprovedPath,
		&approvedSessionTTL,
		&request.Reason,
		&continuationRaw,
		&stateRaw,
		&request.RequestedByActor,
		&requestedByTypeRaw,
		&createdRaw,
		&expiresRaw,
		&request.ResolvedByActor,
		&resolvedByTypeRaw,
		&resolvedAtRaw,
		&request.ResolutionNote,
		&request.IssuedSessionID,
		&request.IssuedSessionSecret,
		&issuedSessionExpiresAtRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthRequest{}, app.ErrNotFound
		}
		return domain.AuthRequest{}, err
	}
	if strings.TrimSpace(phaseIDsRaw) == "" {
		phaseIDsRaw = "[]"
	}
	if err := json.Unmarshal([]byte(phaseIDsRaw), &request.PhaseIDs); err != nil {
		return domain.AuthRequest{}, fmt.Errorf("decode auth_requests.phase_ids_json: %w", err)
	}
	if strings.TrimSpace(continuationRaw) == "" {
		continuationRaw = "{}"
	}
	if err := json.Unmarshal([]byte(continuationRaw), &request.Continuation); err != nil {
		return domain.AuthRequest{}, fmt.Errorf("decode auth_requests.continuation_json: %w", err)
	}
	request.ScopeType = domain.NormalizeScopeLevel(domain.ScopeLevel(scopeTypeRaw))
	request.RequestedSessionTTL = time.Duration(requestedSessionTTL) * time.Second
	request.ApprovedSessionTTL = time.Duration(approvedSessionTTL) * time.Second
	request.State = domain.NormalizeAuthRequestState(domain.AuthRequestState(stateRaw))
	request.RequestedByType = normalizeActorType(domain.ActorType(requestedByTypeRaw))
	request.CreatedAt = parseTS(createdRaw)
	request.ExpiresAt = parseTS(expiresRaw)
	request.ResolvedByType = domain.ActorType(normalizeOptionalActorType(domain.ActorType(resolvedByTypeRaw)))
	request.ResolvedAt = parseNullTS(resolvedAtRaw)
	request.IssuedSessionExpiresAt = parseNullTS(issuedSessionExpiresAtRaw)
	if request.State == domain.AuthRequestStateApproved {
		if strings.TrimSpace(request.ApprovedPath) == "" {
			request.ApprovedPath = request.Path
		}
		if request.ApprovedSessionTTL <= 0 {
			request.ApprovedSessionTTL = request.RequestedSessionTTL
		}
	}
	return request, nil
}

// translateNoRows handles translate no rows.
func translateNoRows(res sql.Result) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return app.ErrNotFound
	}
	return nil
}

// boolToInt converts boolean values into sqlite-friendly numeric flags.
func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// queryPlaceholders returns a comma-separated list of SQL parameter placeholders.
func queryPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimPrefix(strings.Repeat(",?", count), ",")
}

// normalizeOptionalActorType canonicalizes optional actor-type fields without defaults.
func normalizeOptionalActorType(actorType domain.ActorType) string {
	switch strings.TrimSpace(strings.ToLower(string(actorType))) {
	case string(domain.ActorTypeUser):
		return string(domain.ActorTypeUser)
	case string(domain.ActorTypeAgent):
		return string(domain.ActorTypeAgent)
	case string(domain.ActorTypeSystem):
		return string(domain.ActorTypeSystem)
	default:
		return ""
	}
}

// nullableString converts empty strings into SQL NULL values.
func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

// ts handles ts.
func ts(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// nullableTS handles nullable ts.
func nullableTS(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTS parses input into a normalized form.
func parseTS(v string) time.Time {
	ts, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return time.Time{}
	}
	return ts.UTC()
}

// parseNullTS parses input into a normalized form.
func parseNullTS(v sql.NullString) *time.Time {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	ts := parseTS(v.String)
	return &ts
}

// probeVecCapability records whether sqlite-vec scalar functions are available on this connection.
func (r *Repository) probeVecCapability(ctx context.Context) error {
	var version string
	if err := r.db.QueryRowContext(ctx, `SELECT vec_version()`).Scan(&version); err != nil {
		if isMissingFunctionErr(err, "vec_version") {
			r.vecAvailable = false
			return errSQLiteVecUnavailable
		}
		return fmt.Errorf("probe vec_version(): %w", err)
	}
	r.vecAvailable = strings.TrimSpace(version) != ""
	if !r.vecAvailable {
		return errSQLiteVecUnavailable
	}
	return nil
}

// requireVecCapability returns one stable sentinel when sqlite-vec support is unavailable.
func (r *Repository) requireVecCapability() error {
	if r.vecAvailable {
		return nil
	}
	return errSQLiteVecUnavailable
}

// isMissingFunctionErr reports whether sqlite returned one missing-function failure for the named function.
func isMissingFunctionErr(err error, fn string) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such function") && strings.Contains(msg, strings.ToLower(fn))
}

// isDuplicateColumnErr reports whether the expected condition is satisfied.
func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate column name")
}
