package app

import (
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

func TestAuthScopeContextFromTaskLineageProjectDirectPhaseCollapsesToProject(t *testing.T) {
	t.Parallel()

	projectID := "p1"
	phase := domain.Task{
		ID:        "phase-1",
		ProjectID: projectID,
		Scope:     "phase",
	}
	task := domain.Task{
		ID:        "task-1",
		ProjectID: projectID,
		ParentID:  phase.ID,
		Scope:     "task",
	}

	t.Run("phase scope", func(t *testing.T) {
		t.Parallel()

		got, err := authScopeContextFromTaskLineage(projectID, domain.ScopeLevelPhase, phase.ID, []domain.Task{phase})
		if err != nil {
			t.Fatalf("authScopeContextFromTaskLineage() error = %v", err)
		}
		if got.ScopeType != domain.ScopeLevelProject {
			t.Fatalf("scope_type = %q, want %q", got.ScopeType, domain.ScopeLevelProject)
		}
		if got.ScopeID != projectID {
			t.Fatalf("scope_id = %q, want %q", got.ScopeID, projectID)
		}
		if len(got.PhaseIDs) != 0 {
			t.Fatalf("phase_ids = %#v, want empty after project collapse", got.PhaseIDs)
		}
	})

	t.Run("task scope", func(t *testing.T) {
		t.Parallel()

		got, err := authScopeContextFromTaskLineage(projectID, domain.ScopeLevelTask, task.ID, []domain.Task{phase, task})
		if err != nil {
			t.Fatalf("authScopeContextFromTaskLineage() error = %v", err)
		}
		if got.ScopeType != domain.ScopeLevelProject {
			t.Fatalf("scope_type = %q, want %q", got.ScopeType, domain.ScopeLevelProject)
		}
		if got.ScopeID != projectID {
			t.Fatalf("scope_id = %q, want %q", got.ScopeID, projectID)
		}
		if len(got.PhaseIDs) != 0 {
			t.Fatalf("phase_ids = %#v, want empty after project collapse", got.PhaseIDs)
		}
	})
}
