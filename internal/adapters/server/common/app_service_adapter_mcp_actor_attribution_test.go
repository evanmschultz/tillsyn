package common

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// newActorAttributionAdapterFixture builds one adapter fixture with seeded project/actionItem rows.
func newActorAttributionAdapterFixture(t *testing.T) (*AppServiceAdapter, *app.Service, domain.Project, domain.ActionItem) {
	t.Helper()

	repo, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	nextID := 0
	idGen := func() string {
		nextID++
		return fmt.Sprintf("id-%03d", nextID)
	}
	clock := func() time.Time {
		return time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	}

	service := app.NewService(repo, idGen, clock, app.ServiceConfig{
		DefaultDeleteMode:        app.DeleteModeArchive,
		AutoCreateProjectColumns: true,
	})
	adapter := NewAppServiceAdapter(service, nil)

	project, err := service.CreateProject(context.Background(), "Actor Fixture", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	columns, err := service.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected auto-created project columns")
	}

	actionItem, err := service.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      project.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		ColumnID:       columns[0].ID,
		Title:          "Seed ActionItem",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "seed-user",
		UpdatedByActor: "seed-user",
		UpdatedByType:  domain.ActorTypeUser,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	return adapter, service, project, actionItem
}

// TestAppServiceAdapterRestoreActionItemAllowsUserAttributionWithoutGuardTuple verifies user+name restore attribution without lease tuple.
func TestAppServiceAdapterRestoreActionItemAllowsUserAttributionWithoutGuardTuple(t *testing.T) {
	adapter, service, _, actionItem := newActorAttributionAdapterFixture(t)
	if err := service.DeleteActionItem(context.Background(), actionItem.ID, app.DeleteModeArchive); err != nil {
		t.Fatalf("DeleteActionItem(archive) error = %v", err)
	}

	restored, err := adapter.RestoreActionItem(context.Background(), RestoreActionItemRequest{
		ActionItemID: actionItem.ID,
		Actor: ActorLeaseTuple{
			ActorType: string(domain.ActorTypeUser),
			AgentName: "EVAN",
		},
	})
	if err != nil {
		t.Fatalf("RestoreActionItem() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatal("expected restored actionItem to clear archived_at")
	}
	if restored.UpdatedByActor != "EVAN" {
		t.Fatalf("restored updated_by_actor = %q, want EVAN", restored.UpdatedByActor)
	}
	if restored.UpdatedByType != domain.ActorTypeUser {
		t.Fatalf("restored updated_by_type = %q, want %q", restored.UpdatedByType, domain.ActorTypeUser)
	}
}

// TestAppServiceAdapterUpdateActionItemRejectsAgentWithoutGuardTuple verifies agent mutations require a lease tuple.
func TestAppServiceAdapterUpdateActionItemRejectsNonUserWithoutGuardTuple(t *testing.T) {
	adapter, _, _, actionItem := newActorAttributionAdapterFixture(t)
	_, err := adapter.UpdateActionItem(context.Background(), UpdateActionItemRequest{
		ActionItemID: actionItem.ID,
		Title:        "Agent Update",
		Actor: ActorLeaseTuple{
			ActorType: string(domain.ActorTypeAgent),
			AgentName: "agent-1",
		},
	})
	if !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("UpdateActionItem() error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestAppServiceAdapterUpdateActionItemAllowsGuardedNonUserAttribution verifies guarded non-user mutations apply agent attribution.
func TestAppServiceAdapterUpdateActionItemAllowsGuardedNonUserAttribution(t *testing.T) {
	adapter, service, project, actionItem := newActorAttributionAdapterFixture(t)
	lease, err := service.IssueCapabilityLease(context.Background(), app.IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleWorker,
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}

	updated, err := adapter.UpdateActionItem(context.Background(), UpdateActionItemRequest{
		ActionItemID: actionItem.ID,
		Title:        "Guarded Agent Update",
		Actor: ActorLeaseTuple{
			ActorType:       string(domain.ActorTypeAgent),
			AgentName:       "agent-1",
			AgentInstanceID: lease.InstanceID,
			LeaseToken:      lease.LeaseToken,
		},
	})
	if err != nil {
		t.Fatalf("UpdateActionItem(guarded) error = %v", err)
	}
	if updated.UpdatedByActor != "agent-1" {
		t.Fatalf("updated updated_by_actor = %q, want agent-1", updated.UpdatedByActor)
	}
	if updated.UpdatedByName != "agent-1" {
		t.Fatalf("updated updated_by_name = %q, want agent-1", updated.UpdatedByName)
	}
	if updated.UpdatedByType != domain.ActorTypeAgent {
		t.Fatalf("updated updated_by_type = %q, want %q", updated.UpdatedByType, domain.ActorTypeAgent)
	}
}
