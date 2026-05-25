package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestActionItemDetailDTOJSONRoundTrip verifies that ActionItemDetailDTO survives JSON marshal/unmarshal,
// including the critical blocked_by field from Metadata.
func TestActionItemDetailDTOJSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
	dueAt := now.Add(24 * time.Hour)
	startedAt := now.Add(1 * time.Hour)
	completedAt := now.Add(2 * time.Hour)

	// Construct a fully-populated domain.ActionItem.
	domainAI := domain.ActionItem{
		ID:             "ai-test-123",
		ProjectID:      "proj-abc",
		ParentID:       "ai-parent",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		Role:           domain.RolePlanner,
		StructuralType: domain.StructuralTypeDroplet,
		Irreducible:    true,
		Owner:          "STEWARD",
		DropNumber:     3,
		Persistent:     false,
		DevGated:       true,
		Paths:          []string{"internal/client/dto.go", "internal/client/dto_test.go"},
		Packages:       []string{"github.com/evanmschultz/tillsyn/internal/client"},
		Files:          []string{"CLAUDE.md"},
		StartCommit:    "abc123def",
		EndCommit:      "def456ghi",
		LifecycleState: domain.StateInProgress,
		ColumnID:       "col-todo",
		Position:       42,
		Title:          "Build rich DTOs",
		Description:    "Create wire-contract DTOs for daemon",
		Priority:       domain.PriorityHigh,
		DueAt:          &dueAt,
		Labels:         []string{"go", "wire-contract"},
		Metadata: domain.ActionItemMetadata{
			Objective:          "Define serialization-clean DTOs",
			AcceptanceCriteria: "All fields exported, JSON-tagged, no funcs/chans",
			BlockedReason:      "",
			BlockedBy:          []string{"ai-a3-prereq"},
			DependsOn:          []string{"ai-plan-root"},
			Outcome:            "",
		},
		CreatedByActor: "client-seam-planner",
		CreatedByName:  "CLIENT-SEAM-PLANNER",
		UpdatedByActor: "client-seam-builder",
		UpdatedByName:  "CLIENT-SEAM-BUILDER",
		UpdatedByType:  domain.ActorTypeAgent,
		CreatedAt:      now,
		UpdatedAt:      now.Add(time.Hour),
		StartedAt:      &startedAt,
		CompletedAt:    &completedAt,
		ArchivedAt:     nil,
		CanceledAt:     nil,
	}

	// Convert to DTO.
	dto := ActionItemDetailFromDomain(domainAI)

	// Verify critical fields in DTO.
	if dto.ID != "ai-test-123" {
		t.Errorf("DTO.ID = %q, want ai-test-123", dto.ID)
	}
	if len(dto.Paths) != 2 {
		t.Errorf("DTO.Paths length = %d, want 2", len(dto.Paths))
	}
	if dto.Persistent != false {
		t.Errorf("DTO.Persistent = %v, want false", dto.Persistent)
	}
	if dto.DevGated != true {
		t.Errorf("DTO.DevGated = %v, want true", dto.DevGated)
	}
	if len(dto.Metadata.BlockedBy) != 1 {
		t.Errorf("DTO.Metadata.BlockedBy length = %d, want 1", len(dto.Metadata.BlockedBy))
	}
	if dto.Metadata.BlockedBy[0] != "ai-a3-prereq" {
		t.Errorf("DTO.Metadata.BlockedBy[0] = %q, want ai-a3-prereq", dto.Metadata.BlockedBy[0])
	}

	// Marshal to JSON.
	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Unmarshal back to a fresh DTO.
	var unmarshalledDTO ActionItemDetailDTO
	if err := json.Unmarshal(jsonBytes, &unmarshalledDTO); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// Verify round-trip: all fields survived, especially blocked_by.
	if unmarshalledDTO.ID != dto.ID {
		t.Errorf("round-trip ID: got %q, want %q", unmarshalledDTO.ID, dto.ID)
	}
	if unmarshalledDTO.Title != dto.Title {
		t.Errorf("round-trip Title: got %q, want %q", unmarshalledDTO.Title, dto.Title)
	}
	if len(unmarshalledDTO.Paths) != len(dto.Paths) {
		t.Errorf("round-trip Paths length: got %d, want %d", len(unmarshalledDTO.Paths), len(dto.Paths))
	}
	if unmarshalledDTO.DevGated != dto.DevGated {
		t.Errorf("round-trip DevGated: got %v, want %v", unmarshalledDTO.DevGated, dto.DevGated)
	}

	// **Critical assertion**: BlockedBy must survive round-trip.
	if len(unmarshalledDTO.Metadata.BlockedBy) != 1 {
		t.Errorf("round-trip Metadata.BlockedBy length: got %d, want 1", len(unmarshalledDTO.Metadata.BlockedBy))
	}
	if unmarshalledDTO.Metadata.BlockedBy[0] != "ai-a3-prereq" {
		t.Errorf("round-trip Metadata.BlockedBy[0]: got %q, want ai-a3-prereq", unmarshalledDTO.Metadata.BlockedBy[0])
	}

	// Verify timestamps round-trip as RFC3339 strings.
	if unmarshalledDTO.CreatedAt != dto.CreatedAt {
		t.Errorf("round-trip CreatedAt: got %q, want %q", unmarshalledDTO.CreatedAt, dto.CreatedAt)
	}
	if unmarshalledDTO.UpdatedAt != dto.UpdatedAt {
		t.Errorf("round-trip UpdatedAt: got %q, want %q", unmarshalledDTO.UpdatedAt, dto.UpdatedAt)
	}
}

// TestCommentDTOMappingAndRoundTrip verifies CommentDTO construction and JSON stability.
func TestCommentDTOMappingAndRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 24, 14, 30, 0, 0, time.UTC)

	domainComment := domain.Comment{
		ID:           "comment-001",
		ProjectID:    "proj-abc",
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     "ai-123",
		Summary:      "Closing comment",
		BodyMarkdown: "## Summary\n\nWork completed successfully.",
		ActorID:      "builder-agent",
		ActorName:    "CLIENT-SEAM-D3-BUILDER",
		ActorType:    domain.ActorTypeAgent,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	dto := CommentFromDomain(domainComment)

	if dto.ID != "comment-001" {
		t.Errorf("DTO.ID = %q, want comment-001", dto.ID)
	}
	if dto.Summary != "Closing comment" {
		t.Errorf("DTO.Summary = %q, want Closing comment", dto.Summary)
	}
	if dto.TargetType != "action_item" {
		t.Errorf("DTO.TargetType = %q, want action_item", dto.TargetType)
	}

	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var unmarshalledDTO CommentDTO
	if err := json.Unmarshal(jsonBytes, &unmarshalledDTO); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if unmarshalledDTO.Summary != dto.Summary {
		t.Errorf("round-trip Summary: got %q, want %q", unmarshalledDTO.Summary, dto.Summary)
	}
	if unmarshalledDTO.BodyMarkdown != dto.BodyMarkdown {
		t.Errorf("round-trip BodyMarkdown: got %q, want %q", unmarshalledDTO.BodyMarkdown, dto.BodyMarkdown)
	}
}

// TestDependencyRollupDTOMapping verifies DependencyRollupDTO construction from domain.
func TestDependencyRollupDTOMapping(t *testing.T) {
	// Note: domain.DependencyRollup is constructed in buildDependencyRollup function.
	// We can't directly instantiate it here without importing internal/app, but we can
	// verify the mapper logic is correct by testing the DTO fields.

	// Create a mock rollup-like structure by building via app service.
	// For now, create a minimal rollup using zero values and verify mapping works.
	rollup := domain.DependencyRollup{}
	rollup.ProjectID = "proj-xyz"
	rollup.TotalItems = 50
	rollup.ItemsWithDependencies = 15
	rollup.DependencyEdges = 32
	rollup.BlockedItems = 8
	rollup.BlockedByEdges = 12
	rollup.UnresolvedDependencyEdges = 5

	dto := DependencyRollupFromDomain(rollup)

	if dto.ProjectID != "proj-xyz" {
		t.Errorf("DTO.ProjectID = %q, want proj-xyz", dto.ProjectID)
	}
	if dto.TotalItems != 50 {
		t.Errorf("DTO.TotalItems = %d, want 50", dto.TotalItems)
	}
	if dto.ItemsWithDependencies != 15 {
		t.Errorf("DTO.ItemsWithDependencies = %d, want 15", dto.ItemsWithDependencies)
	}
	if dto.UnresolvedDependencyEdges != 5 {
		t.Errorf("DTO.UnresolvedDependencyEdges = %d, want 5", dto.UnresolvedDependencyEdges)
	}

	// Verify JSON round-trip.
	jsonBytes, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var unmarshalledDTO DependencyRollupDTO
	if err := json.Unmarshal(jsonBytes, &unmarshalledDTO); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if unmarshalledDTO.ProjectID != dto.ProjectID {
		t.Errorf("round-trip ProjectID: got %q, want %q", unmarshalledDTO.ProjectID, dto.ProjectID)
	}
	if unmarshalledDTO.BlockedByEdges != dto.BlockedByEdges {
		t.Errorf("round-trip BlockedByEdges: got %d, want %d", unmarshalledDTO.BlockedByEdges, dto.BlockedByEdges)
	}
}

// TestCreateActionItemInputDTOToAppInput verifies conversion to app.CreateActionItemInput.
func TestCreateActionItemInputDTOToAppInput(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	dueAt := now.Add(24 * time.Hour)

	dto := CreateActionItemInputDTO{
		ProjectID:      "proj-123",
		ParentID:       "ai-parent",
		Kind:           "build",
		StructuralType: "droplet",
		Owner:          "STEWARD",
		DropNumber:     3,
		Persistent:     true,
		DevGated:       false,
		Paths:          []string{"internal/client/dto.go"},
		Packages:       []string{"github.com/evanmschultz/tillsyn/internal/client"},
		Files:          []string{"CLAUDE.md"},
		StartCommit:    "abc123",
		EndCommit:      "def456",
		ColumnID:       "col-1",
		Title:          "Test Build",
		Description:    "Test description",
		Priority:       "high",
		DueAt:          dueAtString(dueAt),
		Labels:         []string{"test"},
		CreatedByActor: "builder",
		CreatedByName:  "Builder Agent",
	}

	appInput := dto.ToAppInput()

	if appInput.ProjectID != "proj-123" {
		t.Errorf("appInput.ProjectID = %q, want proj-123", appInput.ProjectID)
	}
	if appInput.Kind != domain.KindBuild {
		t.Errorf("appInput.Kind = %v, want %v", appInput.Kind, domain.KindBuild)
	}
	if appInput.Priority != domain.PriorityHigh {
		t.Errorf("appInput.Priority = %v, want %v", appInput.Priority, domain.PriorityHigh)
	}
	if appInput.Title != "Test Build" {
		t.Errorf("appInput.Title = %q, want Test Build", appInput.Title)
	}
	if len(appInput.Paths) != 1 {
		t.Errorf("appInput.Paths length = %d, want 1", len(appInput.Paths))
	}
	if appInput.DueAt == nil || !appInput.DueAt.Equal(dueAt) {
		t.Errorf("appInput.DueAt mismatch")
	}
}

// TestUpdateActionItemInputDTOToAppInput verifies conversion to app.UpdateActionItemInput.
func TestUpdateActionItemInputDTOToAppInput(t *testing.T) {
	newTitle := "Updated Title"
	newDescription := "Updated description"
	newPriority := "low"
	newDueAtStr := "2026-05-31T10:00:00Z"
	newDueAtStrPtr := &newDueAtStr

	dto := UpdateActionItemInputDTO{
		ActionItemID: "ai-123",
		Title:        &newTitle,
		Description:  &newDescription,
		Priority:     &newPriority,
		DueAt:        &newDueAtStrPtr,
		Labels:       &[]string{"updated"},
		Role:         "qa-proof",
		Owner:        ptrStr("NEW_OWNER"),
		DropNumber:   ptrInt(4),
		Persistent:   ptrBool(true),
		DevGated:     ptrBool(false),
	}

	appInput := dto.ToAppInput()

	if appInput.ActionItemID != "ai-123" {
		t.Errorf("appInput.ActionItemID = %q, want ai-123", appInput.ActionItemID)
	}
	if appInput.Title == nil || *appInput.Title != "Updated Title" {
		t.Errorf("appInput.Title mismatch")
	}
	if appInput.Priority == nil || *appInput.Priority != domain.PriorityLow {
		t.Errorf("appInput.Priority mismatch")
	}
	if appInput.Owner == nil || *appInput.Owner != "NEW_OWNER" {
		t.Errorf("appInput.Owner mismatch")
	}
}

// TestCreateActionItemInputDTOToAppInputNilFields verifies nil-preservation for optional fields.
func TestCreateActionItemInputDTOToAppInputNilFields(t *testing.T) {
	// Minimal DTO with only required fields.
	dto := CreateActionItemInputDTO{
		ProjectID:      "proj-xyz",
		Kind:           "plan",
		StructuralType: "drop",
		ColumnID:       "col-1",
		Title:          "Minimal Plan",
	}

	appInput := dto.ToAppInput()

	if appInput.ProjectID != "proj-xyz" {
		t.Errorf("appInput.ProjectID = %q, want proj-xyz", appInput.ProjectID)
	}
	if appInput.ParentID != "" {
		t.Errorf("appInput.ParentID = %q, want empty", appInput.ParentID)
	}
	if appInput.DueAt != nil {
		t.Errorf("appInput.DueAt = %v, want nil", appInput.DueAt)
	}
	if len(appInput.Labels) != 0 {
		t.Errorf("appInput.Labels = %v, want empty", appInput.Labels)
	}
}

// Helper functions for tests
func dueAtString(t time.Time) *string {
	s := t.Format(time.RFC3339)
	return &s
}

func ptrStr(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}

func ptrBool(b bool) *bool {
	return &b
}
