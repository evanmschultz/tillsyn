package app

import (
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

func TestAuthScopeContextFromActionItemLineageProjectDirectPhaseCollapsesToProject(t *testing.T) {
	t.Parallel()

	projectID := "p1"
	phase := domain.ActionItem{
		ID:        "phase-1",
		ProjectID: projectID,
		Scope:     "phase",
	}
	actionItem := domain.ActionItem{
		ID:        "actionItem-1",
		ProjectID: projectID,
		ParentID:  phase.ID,
		Scope:     "actionItem",
	}

	t.Run("phase scope", func(t *testing.T) {
		t.Parallel()

		got, err := authScopeContextFromActionItemLineage(projectID, domain.ScopeLevelPhase, phase.ID, []domain.ActionItem{phase})
		if err != nil {
			t.Fatalf("authScopeContextFromActionItemLineage() error = %v", err)
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

	t.Run("actionItem scope", func(t *testing.T) {
		t.Parallel()

		got, err := authScopeContextFromActionItemLineage(projectID, domain.ScopeLevelActionItem, actionItem.ID, []domain.ActionItem{phase, actionItem})
		if err != nil {
			t.Fatalf("authScopeContextFromActionItemLineage() error = %v", err)
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
