package common

import (
	"context"
	"errors"
	"testing"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestValidateMetadataOutcome is a table-driven unit test for the
// validateMetadataOutcome adapter boundary function. It covers the full valid
// set (success, failure, blocked, superseded), empty/nil passthrough, case
// normalization, and rejection of unrecognized values.
func TestValidateMetadataOutcome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		metadata   *domain.TaskMetadata
		wantErr    bool
		wantOutcome string
	}{
		{
			name:        "success is valid",
			metadata:    &domain.TaskMetadata{Outcome: "success"},
			wantOutcome: "success",
		},
		{
			name:        "failure is valid",
			metadata:    &domain.TaskMetadata{Outcome: "failure"},
			wantOutcome: "failure",
		},
		{
			name:        "blocked is valid",
			metadata:    &domain.TaskMetadata{Outcome: "blocked"},
			wantOutcome: "blocked",
		},
		{
			name:        "superseded is valid",
			metadata:    &domain.TaskMetadata{Outcome: "superseded"},
			wantOutcome: "superseded",
		},
		{
			name:        "empty outcome is valid",
			metadata:    &domain.TaskMetadata{Outcome: ""},
			wantOutcome: "",
		},
		{
			name:        "uppercase SUCCESS normalizes to lowercase",
			metadata:    &domain.TaskMetadata{Outcome: "SUCCESS"},
			wantOutcome: "success",
		},
		{
			name:        "mixed case Blocked normalizes to lowercase",
			metadata:    &domain.TaskMetadata{Outcome: "Blocked"},
			wantOutcome: "blocked",
		},
		{
			name:        "whitespace-padded outcome normalizes",
			metadata:    &domain.TaskMetadata{Outcome: "  failure  "},
			wantOutcome: "failure",
		},
		{
			name:    "banana is rejected",
			metadata: &domain.TaskMetadata{Outcome: "banana"},
			wantErr: true,
		},
		{
			name:    "done is rejected",
			metadata: &domain.TaskMetadata{Outcome: "done"},
			wantErr: true,
		},
		{
			name:    "in_progress is rejected",
			metadata: &domain.TaskMetadata{Outcome: "in_progress"},
			wantErr: true,
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

// TestUpdateTaskRejectsInvalidOutcome verifies end-to-end that the UpdateTask
// adapter method rejects unrecognized metadata.outcome values before they reach
// the application service layer.
func TestUpdateTaskRejectsInvalidOutcome(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	// Create a project and column so we can create a real task.
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

	task, err := fixture.adapter.CreateTask(ctx, CreateTaskRequest{
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Title:     "Task for outcome test",
		Priority:  "medium",
		Actor:     actor,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Updating with an invalid outcome should fail.
	_, err = fixture.adapter.UpdateTask(ctx, UpdateTaskRequest{
		TaskID:   task.ID,
		Title:    "Task for outcome test",
		Metadata: &domain.TaskMetadata{Outcome: "banana"},
		Actor:    actor,
	})
	if err == nil {
		t.Fatal("UpdateTask() expected error for invalid outcome 'banana', got nil")
	}
	if !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("UpdateTask() error = %v, want wrapped ErrInvalidCaptureStateRequest", err)
	}

	// Updating with a valid outcome should succeed.
	updated, err := fixture.adapter.UpdateTask(ctx, UpdateTaskRequest{
		TaskID:   task.ID,
		Title:    "Task for outcome test",
		Metadata: &domain.TaskMetadata{Outcome: "success"},
		Actor:    actor,
	})
	if err != nil {
		t.Fatalf("UpdateTask() unexpected error = %v", err)
	}
	if updated.Metadata.Outcome != "success" {
		t.Fatalf("UpdateTask() outcome = %q, want %q", updated.Metadata.Outcome, "success")
	}

	// Updating with nil metadata should succeed (no validation needed).
	_, err = fixture.adapter.UpdateTask(ctx, UpdateTaskRequest{
		TaskID:   task.ID,
		Title:    "Task for outcome test updated",
		Metadata: nil,
		Actor:    actor,
	})
	if err != nil {
		t.Fatalf("UpdateTask() with nil metadata unexpected error = %v", err)
	}
}

// TestCreateTaskRejectsInvalidOutcome verifies end-to-end that the CreateTask
// adapter method rejects unrecognized metadata.outcome values before they reach
// the application service layer.
func TestCreateTaskRejectsInvalidOutcome(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	// Create a project and column so we can attempt task creation.
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

	// Creating a task with an invalid outcome should fail.
	_, err = fixture.adapter.CreateTask(ctx, CreateTaskRequest{
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Title:     "Task with invalid outcome",
		Priority:  "medium",
		Metadata:  domain.TaskMetadata{Outcome: "banana"},
		Actor:     actor,
	})
	if err == nil {
		t.Fatal("CreateTask() expected error for invalid outcome 'banana', got nil")
	}
	if !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("CreateTask() error = %v, want wrapped ErrInvalidCaptureStateRequest", err)
	}

	// Creating a task with a valid outcome should succeed.
	task, err := fixture.adapter.CreateTask(ctx, CreateTaskRequest{
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Title:     "Task with valid outcome",
		Priority:  "medium",
		Metadata:  domain.TaskMetadata{Outcome: "success"},
		Actor:     actor,
	})
	if err != nil {
		t.Fatalf("CreateTask() unexpected error = %v", err)
	}
	if task.Metadata.Outcome != "success" {
		t.Fatalf("CreateTask() outcome = %q, want %q", task.Metadata.Outcome, "success")
	}

	// Creating a task with empty outcome (default) should succeed.
	_, err = fixture.adapter.CreateTask(ctx, CreateTaskRequest{
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Title:     "Task with empty outcome",
		Priority:  "medium",
		Actor:     actor,
	})
	if err != nil {
		t.Fatalf("CreateTask() with empty outcome unexpected error = %v", err)
	}
}
