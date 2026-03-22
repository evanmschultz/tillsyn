package domain

import (
	"bytes"
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

// TestNewTaskDefaultsAndLabels verifies behavior for the covered scenario.
func TestNewTaskDefaultsAndLabels(t *testing.T) {
	now := time.Now()
	due := now.Add(24 * time.Hour)
	task, err := NewTask(TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "  Ship feature ",
		DueAt:     &due,
		Labels:    []string{"Backend", "backend", "  ", "Urgent"},
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.Priority != PriorityMedium {
		t.Fatalf("expected default medium, got %q", task.Priority)
	}
	if task.Title != "Ship feature" {
		t.Fatalf("unexpected title %q", task.Title)
	}
	if len(task.Labels) != 2 || task.Labels[0] != "backend" || task.Labels[1] != "urgent" {
		t.Fatalf("unexpected labels %#v", task.Labels)
	}
}

// TestNewTaskValidation verifies behavior for the covered scenario.
func TestNewTaskValidation(t *testing.T) {
	now := time.Now()
	_, err := NewTask(TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
		Priority:  Priority("bad"),
	}, now)
	if err != ErrInvalidPriority {
		t.Fatalf("expected ErrInvalidPriority, got %v", err)
	}
}

// TestTaskMoveUpdateArchiveRestore verifies behavior for the covered scenario.
func TestTaskMoveUpdateArchiveRestore(t *testing.T) {
	now := time.Now()
	task, err := NewTask(TaskInput{
		ID:        "t1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "x",
		Priority:  PriorityLow,
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	if err := task.Move("c2", 2, now.Add(time.Minute)); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if task.ColumnID != "c2" || task.Position != 2 {
		t.Fatalf("unexpected move state: %#v", task)
	}

	due := now.Add(2 * time.Hour)
	err = task.UpdateDetails("new", "desc", PriorityHigh, &due, []string{"A", "a", "B"}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("UpdateDetails() error = %v", err)
	}
	if task.Title != "new" || task.Priority != PriorityHigh {
		t.Fatalf("unexpected task update state %#v", task)
	}
	task.Archive(now.Add(3 * time.Minute))
	if task.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}
	task.Restore(now.Add(4 * time.Minute))
	if task.ArchivedAt != nil {
		t.Fatal("expected archived_at nil")
	}
}

// TestNewTaskRichMetadataAndDefaults verifies behavior for the covered scenario.
func TestNewTaskRichMetadataAndDefaults(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	lastVerified := now.Add(-time.Hour)
	task, err := NewTask(TaskInput{
		ID:        "t-rich",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "rich task",
		Priority:  PriorityMedium,
		Metadata: TaskMetadata{
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
				StartCriteria: []ChecklistItem{{Text: "ready", Done: true}},
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.Kind != WorkKindTask {
		t.Fatalf("expected default kind task, got %q", task.Kind)
	}
	if task.LifecycleState != StateTodo {
		t.Fatalf("expected default state todo, got %q", task.LifecycleState)
	}
	if task.UpdatedByType != ActorTypeUser {
		t.Fatalf("expected default actor type user, got %q", task.UpdatedByType)
	}
	if task.Metadata.Objective != "Ship feature" {
		t.Fatalf("expected normalized objective, got %q", task.Metadata.Objective)
	}
	if len(task.Metadata.ContextBlocks) != 1 || task.Metadata.ContextBlocks[0].Type != ContextTypeRunbook {
		t.Fatalf("unexpected context blocks %#v", task.Metadata.ContextBlocks)
	}
	if len(task.Metadata.ResourceRefs) != 1 || task.Metadata.ResourceRefs[0].ResourceType != ResourceTypeURL {
		t.Fatalf("unexpected resource refs %#v", task.Metadata.ResourceRefs)
	}
	if len(task.Metadata.ResourceRefs[0].Tags) != 1 || task.Metadata.ResourceRefs[0].Tags[0] != "spec" {
		t.Fatalf("unexpected normalized resource tags %#v", task.Metadata.ResourceRefs[0].Tags)
	}
}

// TestTaskLifecycleTransitions verifies behavior for the covered scenario.
func TestTaskLifecycleTransitions(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	task, err := NewTask(TaskInput{
		ID:        "t-state",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "stateful",
		Priority:  PriorityLow,
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	if err := task.SetLifecycleState(StateProgress, now.Add(time.Minute)); err != nil {
		t.Fatalf("SetLifecycleState(progress) error = %v", err)
	}
	if task.StartedAt == nil || task.LifecycleState != StateProgress {
		t.Fatalf("expected started/progress state, got %#v", task)
	}
	if err := task.SetLifecycleState(StateDone, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("SetLifecycleState(done) error = %v", err)
	}
	if task.CompletedAt == nil || task.LifecycleState != StateDone {
		t.Fatalf("expected completed/done state, got %#v", task)
	}
	if err := task.Reparent("parent-1", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("Reparent() error = %v", err)
	}
	if task.ParentID != "parent-1" {
		t.Fatalf("unexpected parent id %q", task.ParentID)
	}
	if err := task.Reparent(task.ID, now.Add(4*time.Minute)); err != ErrInvalidParentID {
		t.Fatalf("expected ErrInvalidParentID, got %v", err)
	}
	task.Archive(now.Add(5 * time.Minute))
	if task.LifecycleState != StateArchived {
		t.Fatalf("expected archived state, got %q", task.LifecycleState)
	}
	task.Restore(now.Add(6 * time.Minute))
	if task.LifecycleState != StateTodo {
		t.Fatalf("expected restore to todo, got %q", task.LifecycleState)
	}
}

// TestTaskContractUnmetChecks verifies behavior for the covered scenario.
func TestTaskContractUnmetChecks(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	task, err := NewTask(TaskInput{
		ID:        "t-contract",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "contract",
		Priority:  PriorityHigh,
		Metadata: TaskMetadata{
			CompletionContract: CompletionContract{
				StartCriteria: []ChecklistItem{
					{ID: "s1", Text: "design approved", Done: false},
					{ID: "s2", Text: "repo ready", Done: true},
				},
				CompletionCriteria: []ChecklistItem{
					{ID: "c1", Text: "tests green", Done: false},
				},
				CompletionChecklist: []ChecklistItem{
					{ID: "k1", Text: "docs updated", Done: false},
				},
				Policy: CompletionPolicy{RequireChildrenDone: true},
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	startUnmet := task.StartCriteriaUnmet()
	if len(startUnmet) != 1 || startUnmet[0] != "design approved" {
		t.Fatalf("unexpected start unmet list %#v", startUnmet)
	}
	children := []Task{
		{ID: "child-1", Title: "child", LifecycleState: StateProgress},
	}
	doneUnmet := task.CompletionCriteriaUnmet(children)
	if len(doneUnmet) < 3 {
		t.Fatalf("expected unmet completion checks, got %#v", doneUnmet)
	}
}

// TestNewTaskRejectsInvalidMetadata verifies behavior for the covered scenario.
func TestNewTaskRejectsInvalidMetadata(t *testing.T) {
	now := time.Now()
	_, err := NewTask(TaskInput{
		ID:        "t-bad",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "bad",
		Priority:  PriorityMedium,
		Metadata: TaskMetadata{
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
		KindPayload: jsonRaw(`{"existing":true}`),
		CapabilityPolicy: ProjectCapabilityPolicy{
			AllowEqualScopeDelegation: true,
		},
	}, &ProjectMetadata{
		Owner:             "Default owner",
		Icon:              ":rocket:",
		Color:             "62",
		Tags:              []string{"shared", "beta"},
		StandardsMarkdown: "default standards",
		KindPayload:       jsonRaw(`{"default":true}`),
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
	if !bytes.Equal(merged.KindPayload, jsonRaw(`{"existing":true}`)) {
		t.Fatalf("KindPayload = %s, want existing payload", string(merged.KindPayload))
	}
	if !merged.CapabilityPolicy.AllowOrchestratorOverride {
		t.Fatal("expected orchestrator override policy to be merged")
	}
	if !merged.CapabilityPolicy.AllowEqualScopeDelegation {
		t.Fatal("expected existing equal-scope delegation to remain true")
	}
	if merged.CapabilityPolicy.OrchestratorOverrideToken != "" {
		t.Fatalf("unexpected override token %q", merged.CapabilityPolicy.OrchestratorOverrideToken)
	}
	if len(merged.Tags) != 3 || merged.Tags[0] != "alpha" || merged.Tags[1] != "beta" || merged.Tags[2] != "shared" {
		t.Fatalf("unexpected merged tags %#v", merged.Tags)
	}
}

// TestMergeTaskMetadataDefaults verifies conservative task metadata defaulting.
func TestMergeTaskMetadataDefaults(t *testing.T) {
	merged, err := MergeTaskMetadata(TaskMetadata{
		Objective:       "Existing objective",
		CommandSnippets: []string{"make test"},
		DecisionLog:     []string{"decision-a"},
		CompletionContract: CompletionContract{
			CompletionChecklist: []ChecklistItem{{ID: "ck-existing", Text: "existing check", Done: false}},
		},
	}, &TaskMetadata{
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
		CompletionContract: CompletionContract{
			StartCriteria:       []ChecklistItem{{Text: "ready"}},
			CompletionCriteria:  []ChecklistItem{{ID: "ck-default", Text: "default check"}},
			CompletionChecklist: []ChecklistItem{{ID: "ck-default-2", Text: "default checklist"}},
			CompletionEvidence:  []string{"evidence-a"},
			CompletionNotes:     "default notes",
			Policy:              CompletionPolicy{RequireChildrenDone: true},
		},
	})
	if err != nil {
		t.Fatalf("MergeTaskMetadata() error = %v", err)
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
	if !merged.CompletionContract.Policy.RequireChildrenDone {
		t.Fatal("expected require_children_done to be tightened by defaults")
	}
}

// jsonRaw returns one trimmed JSON payload for merge assertions.
func jsonRaw(raw string) []byte {
	return []byte(raw)
}
