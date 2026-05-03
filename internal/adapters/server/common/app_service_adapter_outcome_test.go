package common

import (
	"context"
	"errors"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestValidateMetadataOutcome is a table-driven unit test for the
// validateMetadataOutcome adapter boundary function. It covers the full valid
// set (success, failure, blocked, superseded), empty/nil passthrough, case
// normalization, and rejection of unrecognized values.
func TestValidateMetadataOutcome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		metadata    *domain.ActionItemMetadata
		wantErr     bool
		wantOutcome string
	}{
		{
			name:        "success is valid",
			metadata:    &domain.ActionItemMetadata{Outcome: "success"},
			wantOutcome: "success",
		},
		{
			name:        "failure is valid",
			metadata:    &domain.ActionItemMetadata{Outcome: "failure"},
			wantOutcome: "failure",
		},
		{
			name:        "blocked is valid",
			metadata:    &domain.ActionItemMetadata{Outcome: "blocked"},
			wantOutcome: "blocked",
		},
		{
			name:        "superseded is valid",
			metadata:    &domain.ActionItemMetadata{Outcome: "superseded"},
			wantOutcome: "superseded",
		},
		{
			name:        "empty outcome is valid",
			metadata:    &domain.ActionItemMetadata{Outcome: ""},
			wantOutcome: "",
		},
		{
			name:        "uppercase SUCCESS normalizes to lowercase",
			metadata:    &domain.ActionItemMetadata{Outcome: "SUCCESS"},
			wantOutcome: "success",
		},
		{
			name:        "mixed case Blocked normalizes to lowercase",
			metadata:    &domain.ActionItemMetadata{Outcome: "Blocked"},
			wantOutcome: "blocked",
		},
		{
			name:        "whitespace-padded outcome normalizes",
			metadata:    &domain.ActionItemMetadata{Outcome: "  failure  "},
			wantOutcome: "failure",
		},
		{
			name:     "banana is rejected",
			metadata: &domain.ActionItemMetadata{Outcome: "banana"},
			wantErr:  true,
		},
		{
			name:     "done is rejected",
			metadata: &domain.ActionItemMetadata{Outcome: "done"},
			wantErr:  true,
		},
		{
			name:     "in_progress is rejected",
			metadata: &domain.ActionItemMetadata{Outcome: "in_progress"},
			wantErr:  true,
		},
		{
			name:     "nil metadata is valid",
			metadata: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateMetadataOutcome(tt.metadata)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateMetadataOutcome() expected error for outcome %q, got nil", tt.metadata.Outcome)
				}
				if !errors.Is(err, ErrInvalidCaptureStateRequest) {
					t.Fatalf("validateMetadataOutcome() error = %v, want wrapped ErrInvalidCaptureStateRequest", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateMetadataOutcome() unexpected error = %v", err)
			}
			if tt.metadata != nil && tt.metadata.Outcome != tt.wantOutcome {
				t.Fatalf("validateMetadataOutcome() outcome = %q, want %q", tt.metadata.Outcome, tt.wantOutcome)
			}
		})
	}
}

// TestUpdateActionItemRejectsInvalidOutcome verifies end-to-end that the UpdateActionItem
// adapter method rejects unrecognized metadata.outcome values before they reach
// the application service layer.
func TestUpdateActionItemRejectsInvalidOutcome(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	// Create a project and column so we can create a real actionItem.
	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{
		Name:        "OutcomeTest",
		Description: "Test project for outcome validation",
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	todo, err := fixture.svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	actionItem, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:      project.ID,
		ColumnID:       todo.ID,
		Title:          "ActionItem for outcome test",
		Priority:       "medium",
		Actor:          actor,
		StructuralType: string(domain.StructuralTypeDroplet),
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	// Updating with an invalid outcome should fail.
	_, err = fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: actionItem.ID,
		Title:        "ActionItem for outcome test",
		Metadata:     &domain.ActionItemMetadata{Outcome: "banana"},
		Actor:        actor,
	})
	if err == nil {
		t.Fatal("UpdateActionItem() expected error for invalid outcome 'banana', got nil")
	}
	if !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("UpdateActionItem() error = %v, want wrapped ErrInvalidCaptureStateRequest", err)
	}

	// Updating with a valid outcome should succeed.
	updated, err := fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: actionItem.ID,
		Title:        "ActionItem for outcome test",
		Metadata:     &domain.ActionItemMetadata{Outcome: "success"},
		Actor:        actor,
	})
	if err != nil {
		t.Fatalf("UpdateActionItem() unexpected error = %v", err)
	}
	if updated.Metadata.Outcome != "success" {
		t.Fatalf("UpdateActionItem() outcome = %q, want %q", updated.Metadata.Outcome, "success")
	}

	// Updating with nil metadata should succeed (no validation needed).
	_, err = fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: actionItem.ID,
		Title:        "ActionItem for outcome test updated",
		Metadata:     nil,
		Actor:        actor,
	})
	if err != nil {
		t.Fatalf("UpdateActionItem() with nil metadata unexpected error = %v", err)
	}
}

// TestCreateActionItemRejectsInvalidOutcome verifies end-to-end that the CreateActionItem
// adapter method rejects unrecognized metadata.outcome values before they reach
// the application service layer.
func TestCreateActionItemRejectsInvalidOutcome(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	// Create a project and column so we can attempt actionItem creation.
	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{
		Name:        "CreateOutcomeTest",
		Description: "Test project for create outcome validation",
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	todo, err := fixture.svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	// Creating a actionItem with an invalid outcome should fail.
	_, err = fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:      project.ID,
		ColumnID:       todo.ID,
		Title:          "ActionItem with invalid outcome",
		Priority:       "medium",
		Metadata:       domain.ActionItemMetadata{Outcome: "banana"},
		Actor:          actor,
		StructuralType: string(domain.StructuralTypeDroplet),
	})
	if err == nil {
		t.Fatal("CreateActionItem() expected error for invalid outcome 'banana', got nil")
	}
	if !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("CreateActionItem() error = %v, want wrapped ErrInvalidCaptureStateRequest", err)
	}

	// Creating a actionItem with a valid outcome should succeed.
	actionItem, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:      project.ID,
		ColumnID:       todo.ID,
		Title:          "ActionItem with valid outcome",
		Priority:       "medium",
		Metadata:       domain.ActionItemMetadata{Outcome: "success"},
		Actor:          actor,
		StructuralType: string(domain.StructuralTypeDroplet),
	})
	if err != nil {
		t.Fatalf("CreateActionItem() unexpected error = %v", err)
	}
	if actionItem.Metadata.Outcome != "success" {
		t.Fatalf("CreateActionItem() outcome = %q, want %q", actionItem.Metadata.Outcome, "success")
	}

	// Creating a actionItem with empty outcome (default) should succeed.
	_, err = fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:      project.ID,
		ColumnID:       todo.ID,
		Title:          "ActionItem with empty outcome",
		Priority:       "medium",
		Actor:          actor,
		StructuralType: string(domain.StructuralTypeDroplet),
	})
	if err != nil {
		t.Fatalf("CreateActionItem() with empty outcome unexpected error = %v", err)
	}
}
