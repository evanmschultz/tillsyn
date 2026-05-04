package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestNewProjectAndSlug verifies behavior for the covered scenario.
func TestNewProjectAndSlug(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, err := NewProject("p1", "  My Big Project!  ", " desc ", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if p.Slug != "my-big-project" {
		t.Fatalf("unexpected slug %q", p.Slug)
	}
	if p.Name != "My Big Project!" {
		t.Fatalf("unexpected name %q", p.Name)
	}
	if p.Metadata.Owner != "" || len(p.Metadata.Tags) != 0 {
		t.Fatalf("expected empty metadata defaults, got %#v", p.Metadata)
	}
}

// TestNewProjectValidation verifies behavior for the covered scenario.
func TestNewProjectValidation(t *testing.T) {
	now := time.Now()
	if _, err := NewProject("", "ok", "", now); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
	if _, err := NewProject("id", "   ", "", now); err != ErrInvalidName {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

// TestProjectArchiveRestore verifies behavior for the covered scenario.
func TestProjectArchiveRestore(t *testing.T) {
	now := time.Now()
	p, err := NewProject("p1", "test", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	later := now.Add(time.Minute)
	p.Archive(later)
	if p.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}
	p.Restore(later.Add(time.Minute))
	if p.ArchivedAt != nil {
		t.Fatal("expected archived_at to be nil")
	}
}

// TestProjectUpdateDetailsWithMetadata verifies behavior for the covered scenario.
func TestProjectUpdateDetailsWithMetadata(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, err := NewProject("p1", "Original", "desc", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}

	err = p.UpdateDetails("  Updated Name ", "  Updated Desc ", ProjectMetadata{
		Owner:    "  Evan ",
		Icon:     ":rocket:",
		Color:    "62",
		Homepage: " https://example.com ",
		Tags:     []string{"Dev", "dev", "Roadmap"},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("UpdateDetails() error = %v", err)
	}
	if p.Name != "Updated Name" || p.Slug != "updated-name" {
		t.Fatalf("unexpected name/slug update %#v", p)
	}
	if p.Description != "Updated Desc" {
		t.Fatalf("unexpected description %q", p.Description)
	}
	if p.Metadata.Owner != "Evan" {
		t.Fatalf("unexpected owner %q", p.Metadata.Owner)
	}
	if p.Metadata.Homepage != "https://example.com" {
		t.Fatalf("unexpected homepage %q", p.Metadata.Homepage)
	}
	if len(p.Metadata.Tags) != 2 || p.Metadata.Tags[0] != "dev" || p.Metadata.Tags[1] != "roadmap" {
		t.Fatalf("unexpected metadata tags %#v", p.Metadata.Tags)
	}
}

// TestNewColumnValidation verifies behavior for the covered scenario.
func TestNewColumnValidation(t *testing.T) {
	now := time.Now()
	_, err := NewColumn("c1", "p1", "todo", -1, 0, now)
	if err != ErrInvalidPosition {
		t.Fatalf("expected ErrInvalidPosition, got %v", err)
	}
	_, err = NewColumn("c1", "p1", "todo", 0, -1, now)
	if err != ErrInvalidPosition {
		t.Fatalf("expected ErrInvalidPosition, got %v", err)
	}
}

// TestColumnMutations verifies behavior for the covered scenario.
func TestColumnMutations(t *testing.T) {
	now := time.Now()
	c, err := NewColumn("c1", "p1", "todo", 0, 5, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := c.Rename("  done ", now.Add(time.Minute)); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if c.Name != "done" {
		t.Fatalf("unexpected column name %q", c.Name)
	}
	if err := c.SetPosition(3, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("SetPosition() error = %v", err)
	}
	if c.Position != 3 {
		t.Fatalf("unexpected position %d", c.Position)
	}
}

// TestNewActionItemDefaultsAndLabels verifies behavior for the covered scenario.
func TestNewActionItemDefaultsAndLabels(t *testing.T) {
	now := time.Now()
	due := now.Add(24 * time.Hour)
	actionItem, err := NewActionItemForTest(ActionItemInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "  Ship feature ",
		Kind:      KindBuild,
		DueAt:     &due,
		Labels:    []string{"Backend", "backend", "  ", "Urgent"},
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if actionItem.Priority != PriorityMedium {
		t.Fatalf("expected default medium, got %q", actionItem.Priority)
	}
	if actionItem.Title != "Ship feature" {
		t.Fatalf("unexpected title %q", actionItem.Title)
	}
	if actionItem.Scope != KindAppliesToBuild {
		t.Fatalf("expected default scope to mirror kind, got %q", actionItem.Scope)
	}
	if len(actionItem.Labels) != 2 || actionItem.Labels[0] != "backend" || actionItem.Labels[1] != "urgent" {
		t.Fatalf("unexpected labels %#v", actionItem.Labels)
	}
}

// TestNewActionItemValidation verifies behavior for the covered scenario.
func TestNewActionItemValidation(t *testing.T) {
	now := time.Now()
	_, err := NewActionItem(ActionItemInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
		Kind:      KindBuild,
		Priority:  Priority("bad"),
	}, now)
	if err != ErrInvalidPriority {
		t.Fatalf("expected ErrInvalidPriority, got %v", err)
	}

	if _, err := NewActionItem(ActionItemInput{
		ID:        "t-missing-kind",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
	}, now); err != ErrInvalidKind {
		t.Fatalf("expected ErrInvalidKind for empty kind, got %v", err)
	}

	if _, err := NewActionItem(ActionItemInput{
		ID:        "t-bad-kind",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
		Kind:      Kind("bogus"),
	}, now); err != ErrInvalidKind {
		t.Fatalf("expected ErrInvalidKind for junk kind, got %v", err)
	}

	if _, err := NewActionItem(ActionItemInput{
		ID:        "t-mismatched-scope",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
		Kind:      KindBuild,
		Scope:     KindAppliesToPlan,
	}, now); err != ErrInvalidKindAppliesTo {
		t.Fatalf("expected ErrInvalidKindAppliesTo when scope mismatches kind, got %v", err)
	}
}

// TestNewActionItemRoleValidation covers the closed Role enum on the optional
// Role field — empty round-trips empty, every valid role round-trips, an
// unknown value rejects with ErrInvalidRole, and whitespace-only normalizes
// to the empty zero value.
func TestNewActionItemRoleValidation(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name     string
		input    Role
		wantRole Role
		wantErr  error
	}{
		{name: "empty", input: "", wantRole: "", wantErr: nil},
		{name: "whitespace", input: "   ", wantRole: "", wantErr: nil},
		{name: "builder", input: RoleBuilder, wantRole: RoleBuilder, wantErr: nil},
		{name: "qa-proof", input: RoleQAProof, wantRole: RoleQAProof, wantErr: nil},
		{name: "qa-falsification", input: RoleQAFalsification, wantRole: RoleQAFalsification, wantErr: nil},
		{name: "qa-a11y", input: RoleQAA11y, wantRole: RoleQAA11y, wantErr: nil},
		{name: "qa-visual", input: RoleQAVisual, wantRole: RoleQAVisual, wantErr: nil},
		{name: "design", input: RoleDesign, wantRole: RoleDesign, wantErr: nil},
		{name: "commit", input: RoleCommit, wantRole: RoleCommit, wantErr: nil},
		{name: "planner", input: RolePlanner, wantRole: RolePlanner, wantErr: nil},
		{name: "research", input: RoleResearch, wantRole: RoleResearch, wantErr: nil},
		{name: "unknown rejects", input: Role("foobar"), wantRole: "", wantErr: ErrInvalidRole},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actionItem, err := NewActionItem(ActionItemInput{
				ID:             "t-role",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "x",
				Kind:           KindBuild,
				StructuralType: StructuralTypeDroplet,
				Role:           tc.input,
			}, now)
			if err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if actionItem.Role != tc.wantRole {
				t.Fatalf("Role = %q, want %q", actionItem.Role, tc.wantRole)
			}
		})
	}
}

// TestNewActionItemStructuralTypeValidation covers the closed StructuralType
// enum on the mandatory StructuralType field. Unlike Role's permissive empty,
// StructuralType MUST be supplied — empty and whitespace-only inputs reject
// with ErrInvalidStructuralType. Each of the four enum members round-trips,
// and an unknown value rejects.
func TestNewActionItemStructuralTypeValidation(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name    string
		input   StructuralType
		wantST  StructuralType
		wantErr error
	}{
		{name: "drop", input: StructuralTypeDrop, wantST: StructuralTypeDrop, wantErr: nil},
		{name: "segment", input: StructuralTypeSegment, wantST: StructuralTypeSegment, wantErr: nil},
		{name: "confluence", input: StructuralTypeConfluence, wantST: StructuralTypeConfluence, wantErr: nil},
		{name: "droplet", input: StructuralTypeDroplet, wantST: StructuralTypeDroplet, wantErr: nil},
		{name: "empty rejects", input: "", wantST: "", wantErr: ErrInvalidStructuralType},
		{name: "whitespace rejects", input: "   ", wantST: "", wantErr: ErrInvalidStructuralType},
		{name: "unknown rejects", input: StructuralType("foobar"), wantST: "", wantErr: ErrInvalidStructuralType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actionItem, err := NewActionItem(ActionItemInput{
				ID:             "t-st",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "x",
				Kind:           KindBuild,
				StructuralType: tc.input,
			}, now)
			if err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if actionItem.StructuralType != tc.wantST {
				t.Fatalf("StructuralType = %q, want %q", actionItem.StructuralType, tc.wantST)
			}
		})
	}
}

// TestActionItemMoveUpdateArchiveRestore verifies behavior for the covered scenario.
func TestActionItemMoveUpdateArchiveRestore(t *testing.T) {
	now := time.Now()
	actionItem, err := NewActionItemForTest(ActionItemInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
		Kind:      KindBuild,
		Priority:  PriorityLow,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}

	if err := actionItem.Move("c2", 2, now.Add(time.Minute)); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if actionItem.ColumnID != "c2" || actionItem.Position != 2 {
		t.Fatalf("unexpected move state: %#v", actionItem)
	}

	due := now.Add(2 * time.Hour)
	err = actionItem.UpdateDetails("new", "desc", PriorityHigh, &due, []string{"A", "a", "B"}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("UpdateDetails() error = %v", err)
	}
	if actionItem.Title != "new" || actionItem.Priority != PriorityHigh {
		t.Fatalf("unexpected actionItem update state %#v", actionItem)
	}
	actionItem.Archive(now.Add(3 * time.Minute))
	if actionItem.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}
	actionItem.Restore(now.Add(4 * time.Minute))
	if actionItem.ArchivedAt != nil {
		t.Fatal("expected archived_at nil")
	}
}

// TestNewActionItemRichMetadataAndDefaults verifies behavior for the covered scenario.
func TestNewActionItemRichMetadataAndDefaults(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	lastVerified := now.Add(-time.Hour)
	actionItem, err := NewActionItemForTest(ActionItemInput{
		ID:        "t-rich",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "rich actionItem",
		Kind:      KindBuild,
		Priority:  PriorityMedium,
		Metadata: ActionItemMetadata{
			Objective: "  Ship feature  ",
			ContextBlocks: []ContextBlock{
				{Title: "rule", Body: "  always run tests  ", Type: ContextType("RUNBOOK"), Importance: ContextImportance("HIGH")},
			},
			ResourceRefs: []ResourceRef{
				{
					ID:             "res1",
					ResourceType:   ResourceType("URL"),
					Location:       " https://example.com/spec ",
					PathMode:       PathMode("ABSOLUTE"),
					Tags:           []string{"Spec", "spec"},
					LastVerifiedAt: &lastVerified,
				},
			},
			CompletionContract: CompletionContract{
				StartCriteria: []ChecklistItem{{Text: "ready", Complete: true}},
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if actionItem.Kind != KindBuild {
		t.Fatalf("expected kind build, got %q", actionItem.Kind)
	}
	if actionItem.Scope != KindAppliesToBuild {
		t.Fatalf("expected scope to mirror kind build, got %q", actionItem.Scope)
	}
	if actionItem.LifecycleState != StateTodo {
		t.Fatalf("expected default state todo, got %q", actionItem.LifecycleState)
	}
	if actionItem.UpdatedByType != ActorTypeUser {
		t.Fatalf("expected default actor type user, got %q", actionItem.UpdatedByType)
	}
	if actionItem.Metadata.Objective != "Ship feature" {
		t.Fatalf("expected normalized objective, got %q", actionItem.Metadata.Objective)
	}
	if len(actionItem.Metadata.ContextBlocks) != 1 || actionItem.Metadata.ContextBlocks[0].Type != ContextTypeRunbook {
		t.Fatalf("unexpected context blocks %#v", actionItem.Metadata.ContextBlocks)
	}
	if len(actionItem.Metadata.ResourceRefs) != 1 || actionItem.Metadata.ResourceRefs[0].ResourceType != ResourceTypeURL {
		t.Fatalf("unexpected resource refs %#v", actionItem.Metadata.ResourceRefs)
	}
	if len(actionItem.Metadata.ResourceRefs[0].Tags) != 1 || actionItem.Metadata.ResourceRefs[0].Tags[0] != "spec" {
		t.Fatalf("unexpected normalized resource tags %#v", actionItem.Metadata.ResourceRefs[0].Tags)
	}
}

// TestActionItemLifecycleTransitions verifies behavior for the covered scenario.
func TestActionItemLifecycleTransitions(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	actionItem, err := NewActionItemForTest(ActionItemInput{
		ID:        "t-state",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "stateful",
		Kind:      KindBuild,
		Priority:  PriorityLow,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}

	if err := actionItem.SetLifecycleState(StateInProgress, now.Add(time.Minute)); err != nil {
		t.Fatalf("SetLifecycleState(in_progress) error = %v", err)
	}
	if actionItem.StartedAt == nil || actionItem.LifecycleState != StateInProgress {
		t.Fatalf("expected started/in_progress state, got %#v", actionItem)
	}
	if err := actionItem.SetLifecycleState(StateComplete, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("SetLifecycleState(complete) error = %v", err)
	}
	if actionItem.CompletedAt == nil || actionItem.LifecycleState != StateComplete {
		t.Fatalf("expected completed/complete state, got %#v", actionItem)
	}
	if err := actionItem.Reparent("parent-1", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("Reparent() error = %v", err)
	}
	if actionItem.ParentID != "parent-1" {
		t.Fatalf("unexpected parent id %q", actionItem.ParentID)
	}
	if err := actionItem.Reparent(actionItem.ID, now.Add(4*time.Minute)); err != ErrInvalidParentID {
		t.Fatalf("expected ErrInvalidParentID, got %v", err)
	}
	actionItem.Archive(now.Add(5 * time.Minute))
	if actionItem.LifecycleState != StateArchived {
		t.Fatalf("expected archived state, got %q", actionItem.LifecycleState)
	}
	actionItem.Restore(now.Add(6 * time.Minute))
	if actionItem.LifecycleState != StateTodo {
		t.Fatalf("expected restore to todo, got %q", actionItem.LifecycleState)
	}

	// todo → failed is valid (discovered invalid before work starts).
	if err := actionItem.SetLifecycleState(StateFailed, now.Add(7*time.Minute)); err != nil {
		t.Fatalf("SetLifecycleState(failed from todo) error = %v", err)
	}
	if actionItem.LifecycleState != StateFailed {
		t.Fatalf("expected failed state, got %q", actionItem.LifecycleState)
	}
	if actionItem.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set when entering failed")
	}

	// Leaving a terminal state back to todo clears CompletedAt.
	if err := actionItem.SetLifecycleState(StateTodo, now.Add(8*time.Minute)); err != nil {
		t.Fatalf("SetLifecycleState(todo from failed) error = %v", err)
	}
	if actionItem.CompletedAt != nil {
		t.Fatal("expected CompletedAt to be nil after leaving failed to todo")
	}

	// in_progress → failed is valid (failure during work).
	if err := actionItem.SetLifecycleState(StateInProgress, now.Add(9*time.Minute)); err != nil {
		t.Fatalf("SetLifecycleState(in_progress) error = %v", err)
	}
	if err := actionItem.SetLifecycleState(StateFailed, now.Add(10*time.Minute)); err != nil {
		t.Fatalf("SetLifecycleState(failed from in_progress) error = %v", err)
	}
	if actionItem.LifecycleState != StateFailed {
		t.Fatalf("expected failed state from in_progress, got %q", actionItem.LifecycleState)
	}
	if actionItem.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set when entering failed from in_progress")
	}
}

// TestIsTerminalState verifies the IsTerminalState helper for all canonical states.
func TestIsTerminalState(t *testing.T) {
	if IsTerminalState(StateTodo) {
		t.Fatal("todo should not be terminal")
	}
	if IsTerminalState(StateInProgress) {
		t.Fatal("in_progress should not be terminal")
	}
	if !IsTerminalState(StateComplete) {
		t.Fatal("complete should be terminal")
	}
	if !IsTerminalState(StateFailed) {
		t.Fatal("failed should be terminal")
	}
	if IsTerminalState(StateArchived) {
		t.Fatal("archived should not be terminal")
	}
}

// TestChecklistItemUnmarshalRejectsLegacyDoneKey verifies that ChecklistItem JSON
// decode is strict-canonical: only the canonical "complete" key is honored, and
// the legacy "done" key produces an explicit error rather than being silently
// dropped to a zero-value Complete field.
func TestChecklistItemUnmarshalRejectsLegacyDoneKey(t *testing.T) {
	cases := []struct {
		name     string
		payload  string
		wantErr  bool
		errMatch string
		want     ChecklistItem
	}{
		{
			name:    "canonical complete=false decodes",
			payload: `{"id":"x","text":"y","complete":false}`,
			want:    ChecklistItem{ID: "x", Text: "y", Complete: false},
		},
		{
			name:    "canonical complete=true decodes",
			payload: `{"id":"x","text":"y","complete":true}`,
			want:    ChecklistItem{ID: "x", Text: "y", Complete: true},
		},
		{
			name:    "missing completion key defaults to Complete=false",
			payload: `{"id":"x","text":"y"}`,
			want:    ChecklistItem{ID: "x", Text: "y", Complete: false},
		},
		{
			name:     "legacy done=true rejected",
			payload:  `{"id":"x","text":"y","done":true}`,
			wantErr:  true,
			errMatch: "legacy",
		},
		{
			name:     "legacy done=false rejected",
			payload:  `{"id":"x","text":"y","done":false}`,
			wantErr:  true,
			errMatch: "legacy",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got ChecklistItem
			err := json.Unmarshal([]byte(tc.payload), &got)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Unmarshal(%q) error = nil, want error", tc.payload)
				}
				if tc.errMatch != "" && !strings.Contains(err.Error(), tc.errMatch) {
					t.Fatalf("Unmarshal(%q) error = %q, want substring %q", tc.payload, err.Error(), tc.errMatch)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal(%q) error = %v, want nil", tc.payload, err)
			}
			if got != tc.want {
				t.Fatalf("Unmarshal(%q) = %#v, want %#v", tc.payload, got, tc.want)
			}
		})
	}
}

// TestActionItemContractUnmetChecks verifies behavior for the covered scenario.
func TestActionItemContractUnmetChecks(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	actionItem, err := NewActionItemForTest(ActionItemInput{
		ID:        "t-contract",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "contract",
		Kind:      KindBuild,
		Priority:  PriorityHigh,
		Metadata: ActionItemMetadata{
			CompletionContract: CompletionContract{
				StartCriteria: []ChecklistItem{
					{ID: "s1", Text: "design approved", Complete: false},
					{ID: "s2", Text: "repo ready", Complete: true},
				},
				CompletionCriteria: []ChecklistItem{
					{ID: "c1", Text: "tests green", Complete: false},
				},
				CompletionChecklist: []ChecklistItem{
					{ID: "k1", Text: "docs updated", Complete: false},
				},
				Policy: CompletionPolicy{RequireChildrenComplete: true},
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	startUnmet := actionItem.StartCriteriaUnmet()
	if len(startUnmet) != 1 || startUnmet[0] != "design approved" {
		t.Fatalf("unexpected start unmet list %#v", startUnmet)
	}
	children := []ActionItem{
		{ID: "child-1", Title: "child", LifecycleState: StateInProgress},
	}
	doneUnmet := actionItem.CompletionCriteriaUnmet(children)
	if len(doneUnmet) < 3 {
		t.Fatalf("expected unmet completion checks, got %#v", doneUnmet)
	}
}

// TestNewActionItemRejectsInvalidMetadata verifies behavior for the covered scenario.
func TestNewActionItemRejectsInvalidMetadata(t *testing.T) {
	now := time.Now()
	_, err := NewActionItemForTest(ActionItemInput{
		ID:        "t-bad",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "bad",
		Kind:      KindBuild,
		Priority:  PriorityMedium,
		Metadata: ActionItemMetadata{
			ContextBlocks: []ContextBlock{
				{Body: "x", Type: ContextType("invalid")},
			},
		},
	}, now)
	if err == nil {
		t.Fatal("expected invalid context type error")
	}
}

// TestMergeProjectMetadataDefaults verifies conservative project metadata defaulting.
func TestMergeProjectMetadataDefaults(t *testing.T) {
	merged, err := MergeProjectMetadata(ProjectMetadata{
		Owner:       "Existing owner",
		Homepage:    "https://example.com/existing",
		Tags:        []string{"alpha", "shared"},
		KindPayload: jsonRaw(`{"shared":{"caller":"keep"},"existing":true}`),
		CapabilityPolicy: ProjectCapabilityPolicy{
			AllowEqualScopeDelegation: true,
		},
	}, &ProjectMetadata{
		Owner:             "Default owner",
		Icon:              ":rocket:",
		Color:             "62",
		Tags:              []string{"shared", "beta"},
		StandardsMarkdown: "default standards",
		KindPayload:       jsonRaw(`{"shared":{"template":"fill"},"default":true}`),
		CapabilityPolicy: ProjectCapabilityPolicy{
			AllowOrchestratorOverride: true,
		},
	})
	if err != nil {
		t.Fatalf("MergeProjectMetadata() error = %v", err)
	}
	if merged.Owner != "Existing owner" {
		t.Fatalf("Owner = %q, want Existing owner", merged.Owner)
	}
	if merged.Icon != ":rocket:" {
		t.Fatalf("Icon = %q, want :rocket:", merged.Icon)
	}
	if merged.Color != "62" {
		t.Fatalf("Color = %q, want 62", merged.Color)
	}
	if merged.Homepage != "https://example.com/existing" {
		t.Fatalf("Homepage = %q, want existing", merged.Homepage)
	}
	if merged.StandardsMarkdown != "default standards" {
		t.Fatalf("StandardsMarkdown = %q, want default standards", merged.StandardsMarkdown)
	}
	if !bytes.Equal(merged.KindPayload, jsonRaw(`{"default":true,"existing":true,"shared":{"caller":"keep","template":"fill"}}`)) {
		t.Fatalf("KindPayload = %s, want merged payload", string(merged.KindPayload))
	}
	if !merged.CapabilityPolicy.AllowEqualScopeDelegation {
		t.Fatal("expected existing equal-scope delegation to remain true")
	}
	if merged.CapabilityPolicy.AllowOrchestratorOverride {
		t.Fatal("expected capability policy defaults to stay explicit-only")
	}
	if merged.CapabilityPolicy.OrchestratorOverrideToken != "" {
		t.Fatalf("unexpected override token %q", merged.CapabilityPolicy.OrchestratorOverrideToken)
	}
	if len(merged.Tags) != 3 || merged.Tags[0] != "alpha" || merged.Tags[1] != "beta" || merged.Tags[2] != "shared" {
		t.Fatalf("unexpected merged tags %#v", merged.Tags)
	}
}

// TestMergeActionItemMetadataDefaults verifies conservative actionItem metadata defaulting.
func TestMergeActionItemMetadataDefaults(t *testing.T) {
	merged, err := MergeActionItemMetadata(ActionItemMetadata{
		Objective:       "Existing objective",
		CommandSnippets: []string{"make test"},
		DecisionLog:     []string{"decision-a"},
		KindPayload:     jsonRaw(`{"shared":{"caller":"keep"},"existing":true}`),
		CompletionContract: CompletionContract{
			CompletionChecklist: []ChecklistItem{{ID: "ck-existing", Text: "existing check", Complete: false}},
		},
	}, &ActionItemMetadata{
		ImplementationNotesUser:  "default user notes",
		ImplementationNotesAgent: "default agent notes",
		AcceptanceCriteria:       "default acceptance",
		DefinitionOfDone:         "default done",
		ValidationPlan:           "default validation",
		BlockedReason:            "default blocked",
		RiskNotes:                "default risk",
		CommandSnippets:          []string{"make test", "make fmt"},
		ExpectedOutputs:          []string{"output-a"},
		DecisionLog:              []string{"decision-b"},
		RelatedItems:             []string{"issue-1"},
		TransitionNotes:          "default transition",
		DependsOn:                []string{"dep-1"},
		BlockedBy:                []string{"block-1"},
		ContextBlocks: []ContextBlock{
			{Title: "runbook", Body: "always test", Type: ContextTypeRunbook, Importance: ContextImportanceHigh},
		},
		ResourceRefs: []ResourceRef{
			{ID: "doc-1", ResourceType: ResourceTypeDoc, Location: "docs/spec.md"},
		},
		KindPayload: jsonRaw(`{"shared":{"template":"fill"},"default":true}`),
		CompletionContract: CompletionContract{
			StartCriteria:       []ChecklistItem{{Text: "ready"}},
			CompletionCriteria:  []ChecklistItem{{ID: "ck-default", Text: "default check"}},
			CompletionChecklist: []ChecklistItem{{ID: "ck-default-2", Text: "default checklist"}},
			CompletionEvidence:  []string{"evidence-a"},
			CompletionNotes:     "default notes",
			Policy:              CompletionPolicy{RequireChildrenComplete: true},
		},
	})
	if err != nil {
		t.Fatalf("MergeActionItemMetadata() error = %v", err)
	}
	if merged.Objective != "Existing objective" {
		t.Fatalf("Objective = %q, want existing", merged.Objective)
	}
	if merged.ImplementationNotesUser != "default user notes" {
		t.Fatalf("ImplementationNotesUser = %q, want default user notes", merged.ImplementationNotesUser)
	}
	if merged.ImplementationNotesAgent != "default agent notes" {
		t.Fatalf("ImplementationNotesAgent = %q, want default agent notes", merged.ImplementationNotesAgent)
	}
	if merged.ValidationPlan != "default validation" {
		t.Fatalf("ValidationPlan = %q, want default validation", merged.ValidationPlan)
	}
	if len(merged.CommandSnippets) != 2 || merged.CommandSnippets[0] != "make test" || merged.CommandSnippets[1] != "make fmt" {
		t.Fatalf("unexpected command snippets %#v", merged.CommandSnippets)
	}
	if len(merged.DecisionLog) != 2 || merged.DecisionLog[0] != "decision-a" || merged.DecisionLog[1] != "decision-b" {
		t.Fatalf("unexpected decision log %#v", merged.DecisionLog)
	}
	if len(merged.ContextBlocks) != 1 || merged.ContextBlocks[0].Type != ContextTypeRunbook {
		t.Fatalf("unexpected context blocks %#v", merged.ContextBlocks)
	}
	if len(merged.ResourceRefs) != 1 || merged.ResourceRefs[0].Location != "docs/spec.md" {
		t.Fatalf("unexpected resource refs %#v", merged.ResourceRefs)
	}
	if !bytes.Equal(merged.KindPayload, jsonRaw(`{"default":true,"existing":true,"shared":{"caller":"keep","template":"fill"}}`)) {
		t.Fatalf("KindPayload = %s, want merged payload", string(merged.KindPayload))
	}
	if len(merged.CompletionContract.StartCriteria) != 1 || merged.CompletionContract.StartCriteria[0].Text != "ready" {
		t.Fatalf("unexpected start criteria %#v", merged.CompletionContract.StartCriteria)
	}
	if len(merged.CompletionContract.CompletionCriteria) != 1 || merged.CompletionContract.CompletionCriteria[0].ID != "ck-default" {
		t.Fatalf("unexpected completion criteria %#v", merged.CompletionContract.CompletionCriteria)
	}
	if len(merged.CompletionContract.CompletionChecklist) != 2 {
		t.Fatalf("unexpected completion checklist %#v", merged.CompletionContract.CompletionChecklist)
	}
	if len(merged.CompletionContract.CompletionEvidence) != 1 || merged.CompletionContract.CompletionEvidence[0] != "evidence-a" {
		t.Fatalf("unexpected completion evidence %#v", merged.CompletionContract.CompletionEvidence)
	}
	if merged.CompletionContract.CompletionNotes != "default notes" {
		t.Fatalf("CompletionNotes = %q, want default notes", merged.CompletionContract.CompletionNotes)
	}
	if !merged.CompletionContract.Policy.RequireChildrenComplete {
		t.Fatal("expected require_children_complete to be tightened by defaults")
	}
}

// jsonRaw returns one trimmed JSON payload for merge assertions.
func jsonRaw(raw string) []byte {
	return []byte(raw)
}

// TestIsValidKindCoversClosedEnum verifies every member of the 12-value Kind
// enum is recognized and non-member inputs are rejected.
func TestIsValidKindCoversClosedEnum(t *testing.T) {
	for _, kind := range []Kind{
		KindPlan,
		KindResearch,
		KindBuild,
		KindPlanQAProof,
		KindPlanQAFalsification,
		KindBuildQAProof,
		KindBuildQAFalsification,
		KindCloseout,
		KindCommit,
		KindRefinement,
		KindDiscussion,
		KindHumanVerify,
	} {
		if !IsValidKind(kind) {
			t.Fatalf("IsValidKind(%q) = false, want true", kind)
		}
	}
	for _, raw := range []string{"", "  ", "bogus", "actionItem", "subtask", "phase", "project"} {
		if IsValidKind(Kind(raw)) {
			t.Fatalf("IsValidKind(%q) = true, want false", raw)
		}
	}
}

// TestDefaultActionItemScopeMirrorsKind verifies scope mirrors kind for every
// member of the 12-value enum and empty scope for invalid kinds.
func TestDefaultActionItemScopeMirrorsKind(t *testing.T) {
	for _, kind := range []Kind{
		KindPlan,
		KindResearch,
		KindBuild,
		KindPlanQAProof,
		KindPlanQAFalsification,
		KindBuildQAProof,
		KindBuildQAFalsification,
		KindCloseout,
		KindCommit,
		KindRefinement,
		KindDiscussion,
		KindHumanVerify,
	} {
		got := DefaultActionItemScope(kind)
		want := KindAppliesTo(kind)
		if got != want {
			t.Fatalf("DefaultActionItemScope(%q) = %q, want %q", kind, got, want)
		}
	}
	if got := DefaultActionItemScope(Kind("bogus")); got != "" {
		t.Fatalf("DefaultActionItemScope(bogus) = %q, want empty", got)
	}
}

// TestNormalizeKindIDLowercaseAndTrim verifies the simplified normalizer only
// lowercases and trims; it no longer rewrites the actionItem token.
func TestNormalizeKindIDLowercaseAndTrim(t *testing.T) {
	tests := []struct {
		in   string
		want KindID
	}{
		{in: "  plan  ", want: KindID("plan")},
		{in: "BUILD", want: KindID("build")},
		{in: "Build-QA-Proof", want: KindID("build-qa-proof")},
		{in: "", want: KindID("")},
		{in: "   ", want: KindID("")},
		// actionItem and its variants are now preserved verbatim lowercased —
		// the old camelCase canonicalization is removed.
		{in: "actionItem", want: KindID("actionitem")},
		{in: "build-actionItem", want: KindID("build-actionitem")},
	}
	for _, tc := range tests {
		got := NormalizeKindID(KindID(tc.in))
		if got != tc.want {
			t.Fatalf("NormalizeKindID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNewActionItemOwnerNormalization covers the free-form Owner field on
// ActionItemInput — empty round-trips empty, whitespace-only collapses to
// empty, surrounding whitespace is trimmed, and arbitrary non-empty values
// (including STEWARD) round-trip without a closed-enum membership check.
func TestNewActionItemOwnerNormalization(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name      string
		input     string
		wantOwner string
	}{
		{name: "empty", input: "", wantOwner: ""},
		{name: "whitespace collapses to empty", input: "   ", wantOwner: ""},
		{name: "steward round-trips", input: "STEWARD", wantOwner: "STEWARD"},
		{name: "surrounding whitespace trimmed", input: "  STEWARD  ", wantOwner: "STEWARD"},
		{name: "arbitrary value round-trips", input: "DEV", wantOwner: "DEV"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actionItem, err := NewActionItem(ActionItemInput{
				ID:             "t-owner",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "x",
				Kind:           KindBuild,
				StructuralType: StructuralTypeDroplet,
				Owner:          tc.input,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			if actionItem.Owner != tc.wantOwner {
				t.Fatalf("Owner = %q, want %q", actionItem.Owner, tc.wantOwner)
			}
		})
	}
}

// TestNewActionItemDropNumberValidation covers the DropNumber field —
// zero round-trips as the zero value (treated as "not a numbered drop"),
// positive values round-trip, and negative values reject with
// ErrInvalidDropNumber.
func TestNewActionItemDropNumberValidation(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name           string
		input          int
		wantDropNumber int
		wantErr        error
	}{
		{name: "zero round-trips", input: 0, wantDropNumber: 0, wantErr: nil},
		{name: "positive round-trips", input: 5, wantDropNumber: 5, wantErr: nil},
		{name: "negative rejects", input: -1, wantDropNumber: 0, wantErr: ErrInvalidDropNumber},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actionItem, err := NewActionItem(ActionItemInput{
				ID:             "t-drop-number",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "x",
				Kind:           KindBuild,
				StructuralType: StructuralTypeDroplet,
				DropNumber:     tc.input,
			}, now)
			if err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if actionItem.DropNumber != tc.wantDropNumber {
				t.Fatalf("DropNumber = %d, want %d", actionItem.DropNumber, tc.wantDropNumber)
			}
		})
	}
}

// TestNewActionItemPersistentRoundTrip covers the Persistent bool — both
// the false zero-value (default) and explicit true round-trip without any
// validation.
func TestNewActionItemPersistentRoundTrip(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name  string
		input bool
	}{
		{name: "false default", input: false},
		{name: "true round-trips", input: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actionItem, err := NewActionItem(ActionItemInput{
				ID:             "t-persistent",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "x",
				Kind:           KindBuild,
				StructuralType: StructuralTypeDroplet,
				Persistent:     tc.input,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			if actionItem.Persistent != tc.input {
				t.Fatalf("Persistent = %v, want %v", actionItem.Persistent, tc.input)
			}
		})
	}
}

// TestNewActionItemDevGatedRoundTrip covers the DevGated bool — both the
// false zero-value (default) and explicit true round-trip without any
// validation.
func TestNewActionItemDevGatedRoundTrip(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name  string
		input bool
	}{
		{name: "false default", input: false},
		{name: "true round-trips", input: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actionItem, err := NewActionItem(ActionItemInput{
				ID:             "t-dev-gated",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "x",
				Kind:           KindBuild,
				StructuralType: StructuralTypeDroplet,
				DevGated:       tc.input,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			if actionItem.DevGated != tc.input {
				t.Fatalf("DevGated = %v, want %v", actionItem.DevGated, tc.input)
			}
		})
	}
}

// TestNewActionItemPathsNormalization covers the Paths []string field added
// in Drop 4a droplet 4a.5. Empty input round-trips as nil; single + multi
// path inputs round-trip with insertion order preserved (the dispatcher's
// lock manager reads the slice as ordered). Surrounding whitespace is
// trimmed. Duplicates after trim are silently deduped to match the Labels
// precedent. Whitespace-only / empty entries reject with ErrInvalidPaths
// (planner bug, not benign noise). Backslash-bearing entries reject with
// ErrInvalidPaths to enforce the forward-slash / git-ls-files convention.
// Path-exists is intentionally NOT enforced at the domain layer — paths
// often refer to files the build droplet will create. Drop 4a Wave 2 lock
// manager performs runtime validation.
func TestNewActionItemPathsNormalization(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name      string
		input     []string
		wantPaths []string
		wantErr   error
	}{
		{name: "nil round-trips empty", input: nil, wantPaths: nil, wantErr: nil},
		{name: "empty slice round-trips empty", input: []string{}, wantPaths: nil, wantErr: nil},
		{name: "single path round-trips", input: []string{"internal/domain/action_item.go"}, wantPaths: []string{"internal/domain/action_item.go"}, wantErr: nil},
		{name: "multi path preserves insertion order", input: []string{"a/b/c.go", "d/e/f.go"}, wantPaths: []string{"a/b/c.go", "d/e/f.go"}, wantErr: nil},
		{name: "surrounding whitespace trimmed", input: []string{"  a/b.go  ", "c.go"}, wantPaths: []string{"a/b.go", "c.go"}, wantErr: nil},
		{name: "duplicates after trim dedupe", input: []string{"a.go", "a.go", "  a.go  ", "b.go"}, wantPaths: []string{"a.go", "b.go"}, wantErr: nil},
		{name: "empty entry rejects", input: []string{"a.go", ""}, wantPaths: nil, wantErr: ErrInvalidPaths},
		{name: "whitespace-only entry rejects", input: []string{"   ", "a.go"}, wantPaths: nil, wantErr: ErrInvalidPaths},
		{name: "backslash rejects", input: []string{`internal\domain\action_item.go`}, wantPaths: nil, wantErr: ErrInvalidPaths},
		{name: "mixed slashes rejects on first backslash", input: []string{"a/b.go", `c\d.go`}, wantPaths: nil, wantErr: ErrInvalidPaths},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actionItem, err := NewActionItem(ActionItemInput{
				ID:             "t-paths",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "x",
				Kind:           KindBuild,
				StructuralType: StructuralTypeDroplet,
				Paths:          tc.input,
			}, now)
			if err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if len(actionItem.Paths) != len(tc.wantPaths) {
				t.Fatalf("Paths length = %d (%#v), want %d (%#v)", len(actionItem.Paths), actionItem.Paths, len(tc.wantPaths), tc.wantPaths)
			}
			for i := range tc.wantPaths {
				if actionItem.Paths[i] != tc.wantPaths[i] {
					t.Fatalf("Paths[%d] = %q, want %q (full = %#v)", i, actionItem.Paths[i], tc.wantPaths[i], actionItem.Paths)
				}
			}
		})
	}
}
