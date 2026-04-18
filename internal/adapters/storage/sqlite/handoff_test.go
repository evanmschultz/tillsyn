package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestRepositoryHandoffSchemaAndIndexes verifies the durable handoff schema exists after migration.
func TestRepositoryHandoffSchemaAndIndexes(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	var tableCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='handoffs'`).Scan(&tableCount); err != nil {
		t.Fatalf("query handoffs table error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected handoffs table to exist, got %d", tableCount)
	}

	rows, err := repo.db.QueryContext(ctx, `PRAGMA table_info(handoffs)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(handoffs) error = %v", err)
	}
	t.Cleanup(func() {
		_ = rows.Close()
	})

	columns := map[string]struct{}{}
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
			t.Fatalf("rows.Scan() error = %v", err)
		}
		columns[name] = struct{}{}
	}

	for _, name := range []string{
		"id",
		"project_id",
		"branch_id",
		"scope_type",
		"scope_id",
		"target_branch_id",
		"target_scope_type",
		"target_scope_id",
		"source_role",
		"target_role",
		"status",
		"summary",
		"next_action",
		"missing_evidence_json",
		"related_refs_json",
		"created_by_actor",
		"created_by_type",
		"created_at",
		"updated_by_actor",
		"updated_by_type",
		"updated_at",
		"resolved_by_actor",
		"resolved_by_type",
		"resolved_at",
		"resolution_note",
	} {
		if _, ok := columns[name]; !ok {
			t.Fatalf("expected handoffs.%s in schema, got %#v", name, columns)
		}
	}

	for _, indexName := range []string{
		"idx_handoffs_project_status_updated_at",
		"idx_handoffs_project_scope_created_at",
		"idx_handoffs_project_target_scope_created_at",
	} {
		if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, indexName).Scan(&tableCount); err != nil {
			t.Fatalf("query index %q error = %v", indexName, err)
		}
		if tableCount != 1 {
			t.Fatalf("expected index %q to exist, got %d", indexName, tableCount)
		}
	}
}

// TestRepositoryHandoffRoundTrip verifies create, get, list, and update handoff persistence.
func TestRepositoryHandoffRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "handoffs.db")
	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p-handoff", "Handoffs", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	later := now.Add(2 * time.Minute)
	first := domain.Handoff{
		ID:              "h-1",
		ProjectID:       project.ID,
		BranchID:        "branch-1",
		ScopeType:       domain.ScopeLevelPhase,
		ScopeID:         "phase-1",
		TargetBranchID:  "branch-qa",
		TargetScopeType: domain.ScopeLevelActionItem,
		TargetScopeID:   "actionItem-qa",
		SourceRole:      "builder",
		TargetRole:      "qa",
		Status:          domain.HandoffStatusWaiting,
		Summary:         "Implement the durable handoff substrate",
		NextAction:      "Wait for QA verification",
		MissingEvidence: []string{"package-level tests", "manual QA"},
		RelatedRefs:     []string{"actionItem-123"},
		CreatedByActor:  "orch-1",
		CreatedByType:   domain.ActorTypeAgent,
		CreatedAt:       now,
		UpdatedByActor:  "orch-1",
		UpdatedByType:   domain.ActorTypeAgent,
		UpdatedAt:       now,
	}
	second := domain.Handoff{
		ID:             "h-2",
		ProjectID:      project.ID,
		BranchID:       "branch-2",
		ScopeType:      domain.ScopeLevelBranch,
		ScopeID:        "branch-2",
		SourceRole:     "orchestrator",
		TargetRole:     "builder",
		Status:         domain.HandoffStatusReady,
		Summary:        "Queue the follow-up work",
		NextAction:     "Pick up next lane",
		CreatedByActor: "orch-1",
		CreatedByType:  domain.ActorTypeAgent,
		CreatedAt:      later,
		UpdatedByActor: "orch-1",
		UpdatedByType:  domain.ActorTypeAgent,
		UpdatedAt:      later,
	}
	if err := repo.CreateHandoff(ctx, first); err != nil {
		t.Fatalf("CreateHandoff(first) error = %v", err)
	}
	if err := repo.CreateHandoff(ctx, second); err != nil {
		t.Fatalf("CreateHandoff(second) error = %v", err)
	}

	loaded, err := repo.GetHandoff(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetHandoff() error = %v", err)
	}
	if loaded.Summary != first.Summary {
		t.Fatalf("unexpected summary %q", loaded.Summary)
	}
	if len(loaded.MissingEvidence) != 2 {
		t.Fatalf("unexpected missing evidence %#v", loaded.MissingEvidence)
	}
	if loaded.TargetScopeID != first.TargetScopeID {
		t.Fatalf("unexpected target scope id %q", loaded.TargetScopeID)
	}
	if loaded.SourceRole != "builder" || loaded.TargetRole != "qa" {
		t.Fatalf("expected lowercased roles, got %#v", loaded)
	}

	listed, err := repo.ListHandoffs(ctx, domain.HandoffListFilter{
		ProjectID: project.ID,
	})
	if err != nil {
		t.Fatalf("ListHandoffs() error = %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 handoffs, got %#v", listed)
	}
	if listed[0].ID != second.ID || listed[1].ID != first.ID {
		t.Fatalf("expected newest-first order, got %#v", listed)
	}

	waitingOnly, err := repo.ListHandoffs(ctx, domain.HandoffListFilter{
		ProjectID: project.ID,
		Statuses:  []domain.HandoffStatus{domain.HandoffStatusWaiting},
	})
	if err != nil {
		t.Fatalf("ListHandoffs(waiting) error = %v", err)
	}
	if len(waitingOnly) != 1 || waitingOnly[0].ID != first.ID {
		t.Fatalf("expected one waiting handoff, got %#v", waitingOnly)
	}

	branchOnly, err := repo.ListHandoffs(ctx, domain.HandoffListFilter{
		ProjectID: project.ID,
		BranchID:  "branch-2",
		ScopeType: domain.ScopeLevelBranch,
		ScopeID:   "branch-2",
	})
	if err != nil {
		t.Fatalf("ListHandoffs(branch) error = %v", err)
	}
	if len(branchOnly) != 1 || branchOnly[0].ID != second.ID {
		t.Fatalf("expected branch filter to return second handoff, got %#v", branchOnly)
	}

	multiStatus, err := repo.ListHandoffs(ctx, domain.HandoffListFilter{
		ProjectID: project.ID,
		Statuses:  []domain.HandoffStatus{domain.HandoffStatusReady, domain.HandoffStatusWaiting},
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("ListHandoffs(multi-status) error = %v", err)
	}
	if len(multiStatus) != 1 || multiStatus[0].ID != second.ID {
		t.Fatalf("expected limit/tie-break list result, got %#v", multiStatus)
	}

	first.Status = domain.HandoffStatusResolved
	first.UpdatedAt = later.Add(time.Minute)
	first.UpdatedByActor = "qa-1"
	first.UpdatedByType = domain.ActorTypeAgent
	first.ResolvedByActor = "qa-1"
	first.ResolvedByType = domain.ActorTypeAgent
	first.ResolutionNote = "verified and complete"
	if err := repo.UpdateHandoff(ctx, first); err != nil {
		t.Fatalf("UpdateHandoff() error = %v", err)
	}

	updated, err := repo.GetHandoff(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetHandoff(updated) error = %v", err)
	}
	if updated.Status != domain.HandoffStatusResolved {
		t.Fatalf("expected resolved status, got %q", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set")
	}
	if updated.ResolutionNote != first.ResolutionNote {
		t.Fatalf("unexpected resolution note %q", updated.ResolutionNote)
	}
}

// TestRepositoryHandoffValidationErrors verifies fail-closed validation for handoff writes and lookups.
func TestRepositoryHandoffValidationErrors(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p-handoff-validate", "Handoffs Validate", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	base := domain.Handoff{
		ID:             "h-valid",
		ProjectID:      project.ID,
		ScopeType:      domain.ScopeLevelActionItem,
		ScopeID:        "actionItem-1",
		SourceRole:     "builder",
		TargetRole:     "qa",
		Status:         domain.HandoffStatusBlocked,
		Summary:        "summary",
		CreatedByActor: "user-1",
		CreatedByType:  domain.ActorTypeUser,
		CreatedAt:      now,
		UpdatedByActor: "user-1",
		UpdatedByType:  domain.ActorTypeUser,
		UpdatedAt:      now,
	}

	t.Run("create invalid id", func(t *testing.T) {
		handoff := base
		handoff.ID = "   "
		if err := repo.CreateHandoff(ctx, handoff); !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf("CreateHandoff() error = %v, want %v", err, domain.ErrInvalidID)
		}
	})

	t.Run("create invalid status", func(t *testing.T) {
		handoff := base
		handoff.ID = "h-invalid-status"
		handoff.Status = domain.HandoffStatus("wat")
		if err := repo.CreateHandoff(ctx, handoff); !errors.Is(err, domain.ErrInvalidHandoffStatus) {
			t.Fatalf("CreateHandoff() error = %v, want %v", err, domain.ErrInvalidHandoffStatus)
		}
	})

	t.Run("create empty summary", func(t *testing.T) {
		handoff := base
		handoff.ID = "h-empty-summary"
		handoff.Summary = "   "
		if err := repo.CreateHandoff(ctx, handoff); !errors.Is(err, domain.ErrInvalidSummary) {
			t.Fatalf("CreateHandoff() error = %v, want %v", err, domain.ErrInvalidSummary)
		}
	})

	t.Run("create invalid actor type", func(t *testing.T) {
		handoff := base
		handoff.ID = "h-invalid-actor"
		handoff.CreatedByType = domain.ActorType("robot")
		if err := repo.CreateHandoff(ctx, handoff); !errors.Is(err, domain.ErrInvalidActorType) {
			t.Fatalf("CreateHandoff() error = %v, want %v", err, domain.ErrInvalidActorType)
		}
	})

	t.Run("create malformed target tuple", func(t *testing.T) {
		handoff := base
		handoff.ID = "h-bad-target"
		handoff.TargetScopeType = domain.ScopeLevelActionItem
		handoff.TargetScopeID = ""
		if err := repo.CreateHandoff(ctx, handoff); !errors.Is(err, domain.ErrInvalidScopeID) {
			t.Fatalf("CreateHandoff() error = %v, want %v", err, domain.ErrInvalidScopeID)
		}
	})

	t.Run("get missing", func(t *testing.T) {
		if _, err := repo.GetHandoff(ctx, "missing"); !errors.Is(err, app.ErrNotFound) {
			t.Fatalf("GetHandoff() error = %v, want %v", err, app.ErrNotFound)
		}
	})

	t.Run("list invalid project", func(t *testing.T) {
		if _, err := repo.ListHandoffs(ctx, domain.HandoffListFilter{}); !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf("ListHandoffs() error = %v, want %v", err, domain.ErrInvalidID)
		}
	})

	t.Run("list invalid status", func(t *testing.T) {
		_, err := repo.ListHandoffs(ctx, domain.HandoffListFilter{
			ProjectID: project.ID,
			Statuses:  []domain.HandoffStatus{"nope"},
		})
		if !errors.Is(err, domain.ErrInvalidHandoffStatus) {
			t.Fatalf("ListHandoffs() error = %v, want %v", err, domain.ErrInvalidHandoffStatus)
		}
	})

	t.Run("schema uses updated-at ordering index for status queries", func(t *testing.T) {
		rows, err := repo.db.QueryContext(ctx, `PRAGMA index_info(idx_handoffs_project_status_updated_at)`)
		if err != nil {
			t.Fatalf("PRAGMA index_info() error = %v", err)
		}
		defer rows.Close()

		columns := make([]string, 0, 4)
		for rows.Next() {
			var seqno, cid int
			var name string
			if err := rows.Scan(&seqno, &cid, &name); err != nil {
				t.Fatalf("rows.Scan() error = %v", err)
			}
			columns = append(columns, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err() error = %v", err)
		}
		if len(columns) != 4 || columns[2] != "updated_at" {
			t.Fatalf("unexpected status index columns %#v", columns)
		}
	})

	t.Run("scan preserves lowercased roles", func(t *testing.T) {
		handoff := base
		handoff.ID = "h-role-normalized"
		handoff.SourceRole = "Builder"
		handoff.TargetScopeType = domain.ScopeLevelActionItem
		handoff.TargetScopeID = "actionItem-target"
		handoff.TargetRole = "QA"
		if err := repo.CreateHandoff(ctx, handoff); err != nil {
			t.Fatalf("CreateHandoff() error = %v", err)
		}
		got, err := repo.GetHandoff(ctx, handoff.ID)
		if err != nil {
			t.Fatalf("GetHandoff() error = %v", err)
		}
		if got.SourceRole != "builder" || got.TargetRole != "qa" {
			t.Fatalf("expected normalized roles, got %#v", got)
		}
		if strings.Contains(got.SourceRole, "B") || strings.Contains(got.TargetRole, "Q") {
			t.Fatalf("expected lowercase stored roles, got %#v", got)
		}
	})
}
