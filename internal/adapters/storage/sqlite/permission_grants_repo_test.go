package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestRepositoryPermissionGrantsSchemaAndIndex verifies the durable
// permission_grants schema (columns + UNIQUE composite + lookup index)
// exists after migration.
func TestRepositoryPermissionGrantsSchemaAndIndex(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	var tableCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='permission_grants'`).Scan(&tableCount); err != nil {
		t.Fatalf("query permission_grants table error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected permission_grants table to exist, got %d", tableCount)
	}

	rows, err := repo.db.QueryContext(ctx, `PRAGMA table_info(permission_grants)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(permission_grants) error = %v", err)
	}
	t.Cleanup(func() {
		_ = rows.Close()
	})
	columns := map[string]string{}
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal *string
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &primaryKey); err != nil {
			t.Fatalf("rows.Scan() error = %v", err)
		}
		columns[name] = colType
	}
	for _, want := range []string{"id", "project_id", "kind", "rule", "cli_kind", "granted_by", "granted_at"} {
		if _, ok := columns[want]; !ok {
			t.Fatalf("expected permission_grants.%s in schema, got %#v", want, columns)
		}
	}

	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_permission_grants_lookup'`).Scan(&tableCount); err != nil {
		t.Fatalf("query idx_permission_grants_lookup error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected idx_permission_grants_lookup to exist, got %d", tableCount)
	}

	// Verify the UNIQUE composite is enforced as an auto-generated index
	// on the four columns. SQLite auto-creates an index named
	// "sqlite_autoindex_permission_grants_*" for each UNIQUE constraint;
	// we only check that exactly one such index exists for this table.
	var uniqueCount int
	if err := repo.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND tbl_name='permission_grants' AND name LIKE 'sqlite_autoindex_permission_grants_%'
	`).Scan(&uniqueCount); err != nil {
		t.Fatalf("query unique-composite autoindex error = %v", err)
	}
	if uniqueCount < 1 {
		t.Fatalf("expected UNIQUE composite to materialize as an autoindex, got %d", uniqueCount)
	}
}

// TestRepositoryPermissionGrantsRoundTrip verifies insert + list + delete
// for one project on the happy path.
func TestRepositoryPermissionGrantsRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "permission_grants_round_trip.db")
	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	project := mustCreateProject(t, ctx, repo, "p-grants-1", now)

	first, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-1",
		ProjectID: project.ID,
		Kind:      domain.KindBuild,
		Rule:      "Bash(npm run *)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
	}, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant(first) error = %v", err)
	}
	second, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-2",
		ProjectID: project.ID,
		Kind:      domain.KindBuild,
		Rule:      "Read(./.zshrc)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewPermissionGrant(second) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, first); err != nil {
		t.Fatalf("InsertGrant(first) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, second); err != nil {
		t.Fatalf("InsertGrant(second) error = %v", err)
	}

	got, err := repo.ListGrantsForKind(ctx, project.ID, domain.KindBuild, "claude")
	if err != nil {
		t.Fatalf("ListGrantsForKind() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 grants, got %d (%#v)", len(got), got)
	}
	// Determinism: granted_at ASC, id ASC.
	if got[0].ID != first.ID || got[1].ID != second.ID {
		t.Fatalf("expected ordering [first, second], got [%s, %s]", got[0].ID, got[1].ID)
	}
	if got[0].Rule != first.Rule {
		t.Errorf("got[0].Rule = %q, want %q", got[0].Rule, first.Rule)
	}
	if !got[0].GrantedAt.Equal(first.GrantedAt) {
		t.Errorf("got[0].GrantedAt = %v, want %v", got[0].GrantedAt, first.GrantedAt)
	}
	if got[0].Kind != domain.KindBuild {
		t.Errorf("got[0].Kind = %q, want %q", got[0].Kind, domain.KindBuild)
	}

	// Delete first; list should now contain only second.
	if err := repo.DeleteGrant(ctx, first.ID); err != nil {
		t.Fatalf("DeleteGrant(first) error = %v", err)
	}
	got, err = repo.ListGrantsForKind(ctx, project.ID, domain.KindBuild, "claude")
	if err != nil {
		t.Fatalf("ListGrantsForKind() post-delete error = %v", err)
	}
	if len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("expected only second grant after delete, got %#v", got)
	}

	// Delete a non-existent id returns ErrNotFound.
	err = repo.DeleteGrant(ctx, "g-does-not-exist")
	if !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("DeleteGrant(nonexistent) err = %v, want ErrNotFound", err)
	}
}

// TestRepositoryPermissionGrantsIdempotentInsert verifies that re-inserting
// the same (project_id, kind, rule, cli_kind) tuple is a noop and that the
// original row's granted_at + granted_by stay untouched.
func TestRepositoryPermissionGrantsIdempotentInsert(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "permission_grants_idempotent.db")
	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	project := mustCreateProject(t, ctx, repo, "p-grants-idemp", now)

	original, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-orig",
		ProjectID: project.ID,
		Kind:      domain.KindBuild,
		Rule:      "Bash(mage *)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
	}, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant(original) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, original); err != nil {
		t.Fatalf("InsertGrant(original) error = %v", err)
	}

	// Construct a second grant with a DIFFERENT id, granted_at, and
	// granted_by but the SAME (project_id, kind, rule, cli_kind) tuple.
	// The UNIQUE composite must reject the second row, leaving the
	// original intact.
	dup, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-dup",
		ProjectID: project.ID,
		Kind:      domain.KindBuild,
		Rule:      "Bash(mage *)",
		CLIKind:   "claude",
		GrantedBy: "OTHER_PRINCIPAL",
	}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("NewPermissionGrant(dup) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, dup); err != nil {
		t.Fatalf("InsertGrant(dup) expected nil (idempotent), got %v", err)
	}

	got, err := repo.ListGrantsForKind(ctx, project.ID, domain.KindBuild, "claude")
	if err != nil {
		t.Fatalf("ListGrantsForKind() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 grant after idempotent re-insert, got %d (%#v)", len(got), got)
	}
	if got[0].ID != original.ID {
		t.Errorf("expected original.ID %q to survive, got %q", original.ID, got[0].ID)
	}
	if got[0].GrantedBy != original.GrantedBy {
		t.Errorf("GrantedBy = %q, want %q (original must not be overwritten)", got[0].GrantedBy, original.GrantedBy)
	}
	if !got[0].GrantedAt.Equal(original.GrantedAt) {
		t.Errorf("GrantedAt = %v, want %v (original must not be overwritten)", got[0].GrantedAt, original.GrantedAt)
	}
}

// TestRepositoryPermissionGrantsCrossProjectIsolation verifies that grants
// scoped to project A do not leak into project B's lookups.
func TestRepositoryPermissionGrantsCrossProjectIsolation(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "permission_grants_cross_project.db")
	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	projectA := mustCreateProject(t, ctx, repo, "p-A", now)
	projectB := mustCreateProject(t, ctx, repo, "p-B", now)

	grantA, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-A",
		ProjectID: projectA.ID,
		Kind:      domain.KindBuild,
		Rule:      "Bash(npm run *)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
	}, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant(A) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, grantA); err != nil {
		t.Fatalf("InsertGrant(A) error = %v", err)
	}

	// Project A has the grant; project B has none.
	gotA, err := repo.ListGrantsForKind(ctx, projectA.ID, domain.KindBuild, "claude")
	if err != nil {
		t.Fatalf("ListGrantsForKind(A) error = %v", err)
	}
	if len(gotA) != 1 || gotA[0].ID != grantA.ID {
		t.Fatalf("project A: expected [%q], got %#v", grantA.ID, gotA)
	}

	gotB, err := repo.ListGrantsForKind(ctx, projectB.ID, domain.KindBuild, "claude")
	if err != nil {
		t.Fatalf("ListGrantsForKind(B) error = %v", err)
	}
	if len(gotB) != 0 {
		t.Fatalf("project B: expected empty, got %#v", gotB)
	}
}

// TestRepositoryPermissionGrantsCrossCLIIsolation verifies that grants
// approved for one CLI (e.g. claude) do not leak into spawns for a
// different CLI even within the same project + kind. This is the
// guarantee F.7.17.7's UNIQUE composite + lookup index is built to
// uphold for Drop 4d's codex landing.
func TestRepositoryPermissionGrantsCrossCLIIsolation(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "permission_grants_cross_cli.db")
	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	project := mustCreateProject(t, ctx, repo, "p-cli", now)

	// Grant approved under cli_kind="claude".
	claudeGrant, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-claude",
		ProjectID: project.ID,
		Kind:      domain.KindBuild,
		Rule:      "Bash(mage *)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
	}, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant(claude) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, claudeGrant); err != nil {
		t.Fatalf("InsertGrant(claude) error = %v", err)
	}

	// Looking up under "claude" returns the grant.
	gotClaude, err := repo.ListGrantsForKind(ctx, project.ID, domain.KindBuild, "claude")
	if err != nil {
		t.Fatalf("ListGrantsForKind(claude) error = %v", err)
	}
	if len(gotClaude) != 1 {
		t.Fatalf("expected 1 grant under claude, got %#v", gotClaude)
	}

	// Looking up under "codex" (the future Drop 4d CLI) MUST return
	// empty, even though project + kind + rule match.
	gotCodex, err := repo.ListGrantsForKind(ctx, project.ID, domain.KindBuild, "codex")
	if err != nil {
		t.Fatalf("ListGrantsForKind(codex) error = %v", err)
	}
	if len(gotCodex) != 0 {
		t.Fatalf("expected empty under codex (cross-CLI isolation), got %#v", gotCodex)
	}

	// Also verify the same rule CAN be granted independently under a
	// different cli_kind — the UNIQUE composite is per-CLI.
	codexGrant, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-codex",
		ProjectID: project.ID,
		Kind:      domain.KindBuild,
		Rule:      "Bash(mage *)",
		CLIKind:   "codex",
		GrantedBy: "STEWARD",
	}, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant(codex) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, codexGrant); err != nil {
		t.Fatalf("InsertGrant(codex) error = %v", err)
	}
	gotCodex, err = repo.ListGrantsForKind(ctx, project.ID, domain.KindBuild, "codex")
	if err != nil {
		t.Fatalf("ListGrantsForKind(codex) post-insert error = %v", err)
	}
	if len(gotCodex) != 1 || gotCodex[0].ID != codexGrant.ID {
		t.Fatalf("expected codex grant post-insert, got %#v", gotCodex)
	}
}

// TestRepositoryPermissionGrantsKindFilter verifies that ListGrantsForKind
// returns only rows matching the requested kind even when other kinds
// exist for the same project + cli_kind.
func TestRepositoryPermissionGrantsKindFilter(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "permission_grants_kind_filter.db")
	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	project := mustCreateProject(t, ctx, repo, "p-kind", now)

	buildGrant, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-build",
		ProjectID: project.ID,
		Kind:      domain.KindBuild,
		Rule:      "Bash(mage *)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
	}, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant(build) error = %v", err)
	}
	qaGrant, err := domain.NewPermissionGrant(domain.PermissionGrantInput{
		ID:        "g-qa",
		ProjectID: project.ID,
		Kind:      domain.KindBuildQAProof,
		Rule:      "Bash(mage *)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
	}, now)
	if err != nil {
		t.Fatalf("NewPermissionGrant(qa) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, buildGrant); err != nil {
		t.Fatalf("InsertGrant(build) error = %v", err)
	}
	if err := repo.InsertGrant(ctx, qaGrant); err != nil {
		t.Fatalf("InsertGrant(qa) error = %v", err)
	}

	gotBuild, err := repo.ListGrantsForKind(ctx, project.ID, domain.KindBuild, "claude")
	if err != nil {
		t.Fatalf("ListGrantsForKind(build) error = %v", err)
	}
	if len(gotBuild) != 1 || gotBuild[0].ID != buildGrant.ID {
		t.Fatalf("expected only build grant under KindBuild, got %#v", gotBuild)
	}

	gotQA, err := repo.ListGrantsForKind(ctx, project.ID, domain.KindBuildQAProof, "claude")
	if err != nil {
		t.Fatalf("ListGrantsForKind(qa) error = %v", err)
	}
	if len(gotQA) != 1 || gotQA[0].ID != qaGrant.ID {
		t.Fatalf("expected only qa grant under KindBuildQAProof, got %#v", gotQA)
	}
}

// TestRepositoryPermissionGrantsValidationErrors verifies fail-closed
// validation on the storage adapter for missing or zero-value fields. The
// domain constructor catches most of these before they reach the
// adapter; the adapter's own checks defend against direct callers that
// hand-build a PermissionGrant struct without going through
// NewPermissionGrant.
func TestRepositoryPermissionGrantsValidationErrors(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	mustCreateProject(t, ctx, repo, "p-valid", now)

	base := domain.PermissionGrant{
		ID:        "g-x",
		ProjectID: "p-valid",
		Kind:      domain.KindBuild,
		Rule:      "Bash(mage *)",
		CLIKind:   "claude",
		GrantedBy: "STEWARD",
		GrantedAt: now,
	}

	// Empty ID rejected.
	bad := base
	bad.ID = ""
	if err := repo.InsertGrant(ctx, bad); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("InsertGrant(empty ID) err = %v, want ErrInvalidID", err)
	}

	// Zero GrantedAt rejected.
	bad = base
	bad.GrantedAt = time.Time{}
	if err := repo.InsertGrant(ctx, bad); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("InsertGrant(zero GrantedAt) err = %v, want ErrInvalidID", err)
	}

	// Empty cliKind on List rejected.
	if _, err := repo.ListGrantsForKind(ctx, "p-valid", domain.KindBuild, ""); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("ListGrantsForKind(empty cliKind) err = %v, want ErrInvalidID", err)
	}

	// Empty id on Delete rejected.
	if err := repo.DeleteGrant(ctx, ""); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("DeleteGrant(empty id) err = %v, want ErrInvalidID", err)
	}
}

// mustCreateProject is a small helper that creates one project row for
// permission_grants tests so the FK constraint on permission_grants.project_id
// is satisfied. Returns the persisted Project.
func mustCreateProject(t *testing.T, ctx context.Context, repo *Repository, id string, now time.Time) domain.Project {
	t.Helper()
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: id, Name: id}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput(%q) error = %v", id, err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject(%q) error = %v", id, err)
	}
	return project
}
