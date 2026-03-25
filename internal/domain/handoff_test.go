package domain

import (
	"errors"
	"testing"
	"time"
)

// TestNewHandoffNormalizesAndValidates verifies creation normalization and validation.
func TestNewHandoffNormalizesAndValidates(t *testing.T) {
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	handoff, err := NewHandoff(HandoffInput{
		ID:              " handoff-1 ",
		ProjectID:       " project-1 ",
		BranchID:        " branch-1 ",
		ScopeType:       ScopeLevelPhase,
		ScopeID:         " phase-1 ",
		SourceRole:      " Builder ",
		TargetBranchID:  " branch-qa ",
		TargetScopeType: ScopeLevelTask,
		TargetScopeID:   " qa-task ",
		TargetRole:      " QA ",
		Status:          HandoffStatusReady,
		Summary:         " Hand off to QA ",
		NextAction:      " Wait for verification ",
		MissingEvidence: []string{" package tests ", "package tests", "manual qa"},
		RelatedRefs:     []string{" task-1 ", "task-1", " task-2 "},
		CreatedByActor:  " user-1 ",
		CreatedByType:   ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewHandoff() error = %v", err)
	}
	if handoff.ID != "handoff-1" {
		t.Fatalf("ID = %q, want handoff-1", handoff.ID)
	}
	if handoff.ProjectID != "project-1" || handoff.BranchID != "branch-1" {
		t.Fatalf("unexpected source tuple %#v", handoff)
	}
	if handoff.SourceRole != "builder" || handoff.TargetRole != "qa" {
		t.Fatalf("unexpected role normalization %#v", handoff)
	}
	if len(handoff.MissingEvidence) != 2 || handoff.MissingEvidence[0] != "package tests" {
		t.Fatalf("unexpected missing evidence %#v", handoff.MissingEvidence)
	}
	if len(handoff.RelatedRefs) != 2 || handoff.RelatedRefs[1] != "task-2" {
		t.Fatalf("unexpected related refs %#v", handoff.RelatedRefs)
	}
	if handoff.CreatedAt != now || handoff.UpdatedAt != now {
		t.Fatalf("expected timestamps to match creation time, got created=%v updated=%v", handoff.CreatedAt, handoff.UpdatedAt)
	}
	if handoff.IsTerminal() {
		t.Fatal("new handoff should not be terminal")
	}
}

// TestNewHandoffRejectsTerminalCreate verifies terminal statuses are not allowed at creation time.
func TestNewHandoffRejectsTerminalCreate(t *testing.T) {
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	_, err := NewHandoff(HandoffInput{
		ID:            "handoff-1",
		ProjectID:     "project-1",
		ScopeType:     ScopeLevelProject,
		Summary:       "done",
		Status:        HandoffStatusResolved,
		CreatedByType: ActorTypeUser,
	}, now)
	if !errors.Is(err, ErrInvalidHandoffTransition) {
		t.Fatalf("NewHandoff() error = %v, want %v", err, ErrInvalidHandoffTransition)
	}
}

// TestHandoffUpdateTransitions verifies non-terminal and terminal update behavior.
func TestHandoffUpdateTransitions(t *testing.T) {
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	handoff, err := NewHandoff(HandoffInput{
		ID:             "handoff-1",
		ProjectID:      "project-1",
		ScopeType:      ScopeLevelTask,
		ScopeID:        "task-1",
		SourceRole:     "builder",
		TargetRole:     "qa",
		Summary:        "Need validation",
		CreatedByActor: "user-1",
		CreatedByType:  ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewHandoff() error = %v", err)
	}
	if handoff.SourceRole != "builder" || handoff.TargetRole != "qa" {
		t.Fatalf("expected role-only handoff to preserve source/target roles, got %#v", handoff)
	}

	waitingAt := now.Add(5 * time.Minute)
	if err := handoff.Update(HandoffUpdateInput{
		Status:          HandoffStatusWaiting,
		Summary:         "Waiting on QA",
		NextAction:      "Hold until QA responds",
		MissingEvidence: []string{"qa signoff"},
		UpdatedByType:   ActorTypeAgent,
	}, waitingAt); err != nil {
		t.Fatalf("Update(waiting) error = %v", err)
	}
	if handoff.UpdatedByActor != "user-1" {
		t.Fatalf("UpdatedByActor = %q, want user-1 fallback", handoff.UpdatedByActor)
	}
	if handoff.SourceRole != "builder" || handoff.TargetRole != "qa" {
		t.Fatalf("expected status-only update to preserve handoff roles, got %#v", handoff)
	}
	if handoff.ResolvedAt != nil {
		t.Fatalf("expected unresolved handoff, got resolved_at=%v", handoff.ResolvedAt)
	}

	retargetedAt := now.Add(7 * time.Minute)
	if err := handoff.Update(HandoffUpdateInput{
		Status:          HandoffStatusWaiting,
		TargetBranchID:  "branch-qa",
		TargetScopeType: ScopeLevelTask,
		TargetScopeID:   "task-qa-2",
		Summary:         "Waiting on QA retarget",
		UpdatedByActor:  "user-1",
		UpdatedByType:   ActorTypeUser,
	}, retargetedAt); err != nil {
		t.Fatalf("Update(retarget preserve role) error = %v", err)
	}
	if handoff.TargetBranchID != "branch-qa" || handoff.TargetScopeID != "task-qa-2" || handoff.TargetRole != "qa" {
		t.Fatalf("expected retarget update to preserve target role, got %#v", handoff)
	}

	resolvedAt := now.Add(10 * time.Minute)
	if err := handoff.Update(HandoffUpdateInput{
		Status:         HandoffStatusResolved,
		Summary:        "QA complete",
		NextAction:     "Return to orchestrator",
		UpdatedByActor: "qa-1",
		UpdatedByType:  ActorTypeAgent,
		ResolvedByType: ActorTypeAgent,
		ResolutionNote: "qa passed",
	}, resolvedAt); err != nil {
		t.Fatalf("Update(resolved) error = %v", err)
	}
	if !handoff.IsTerminal() {
		t.Fatal("expected resolved handoff to be terminal")
	}
	if handoff.ResolvedAt == nil || !handoff.ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("ResolvedAt = %v, want %v", handoff.ResolvedAt, resolvedAt)
	}
	if handoff.ResolvedByActor != "qa-1" || handoff.ResolutionNote != "qa passed" {
		t.Fatalf("unexpected resolution fields %#v", handoff)
	}
	if err := handoff.Update(HandoffUpdateInput{
		Status:         HandoffStatusReturned,
		Summary:        "should fail",
		UpdatedByActor: "qa-2",
		UpdatedByType:  ActorTypeAgent,
	}, resolvedAt.Add(time.Minute)); !errors.Is(err, ErrInvalidHandoffTransition) {
		t.Fatalf("Update(after terminal) error = %v, want %v", err, ErrInvalidHandoffTransition)
	}
}

// TestNormalizeHandoffListFilter verifies filter normalization and validation.
func TestNormalizeHandoffListFilter(t *testing.T) {
	filter, err := NormalizeHandoffListFilter(HandoffListFilter{
		ProjectID: " project-1 ",
		BranchID:  " branch-1 ",
		ScopeType: ScopeLevelTask,
		ScopeID:   " task-1 ",
		Statuses:  []HandoffStatus{" waiting ", HandoffStatusWaiting, HandoffStatusReady},
		Limit:     -1,
	})
	if err != nil {
		t.Fatalf("NormalizeHandoffListFilter() error = %v", err)
	}
	if filter.ProjectID != "project-1" || filter.ScopeID != "task-1" {
		t.Fatalf("unexpected normalized filter %#v", filter)
	}
	if len(filter.Statuses) != 2 || filter.Statuses[0] != HandoffStatusWaiting || filter.Statuses[1] != HandoffStatusReady {
		t.Fatalf("unexpected statuses %#v", filter.Statuses)
	}
	if filter.Limit != 0 {
		t.Fatalf("Limit = %d, want 0", filter.Limit)
	}
	if _, err := NormalizeHandoffListFilter(HandoffListFilter{ProjectID: "project-1", ScopeID: "task-1"}); !errors.Is(err, ErrInvalidScopeType) {
		t.Fatalf("NormalizeHandoffListFilter(scope without type) error = %v, want %v", err, ErrInvalidScopeType)
	}
}
