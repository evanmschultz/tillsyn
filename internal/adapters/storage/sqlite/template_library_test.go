package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestRepository_TemplateLibraryBindingAndContractRoundTrip verifies template-library storage, binding, and snapshot persistence.
func TestRepository_TemplateLibraryBindingAndContractRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 3, 29, 16, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("project-1", "Example", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := domain.NewColumn("column-1", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	library, err := domain.NewTemplateLibrary(domain.TemplateLibraryInput{
		ID:          "library-1",
		Scope:       domain.TemplateLibraryScopeProject,
		ProjectID:   project.ID,
		Name:        "Example Library",
		Description: "Applies QA follow-up work",
		Status:      domain.TemplateLibraryStatusApproved,
		NodeTemplates: []domain.NodeTemplateInput{
			{
				ID:         "actionItem-template",
				ScopeLevel: domain.KindAppliesToActionItem,
				NodeKindID: "actionItem",
				ActionItemMetadataDefaults: &domain.ActionItemMetadata{
					ValidationPlan: "Run actionItem validation",
				},
				ChildRules: []domain.TemplateChildRuleInput{
					{
						ID:                      "qa-check",
						Position:                1,
						ChildScopeLevel:         domain.KindAppliesToSubtask,
						ChildKindID:             "subtask",
						TitleTemplate:           "QA review",
						DescriptionTemplate:     "Verify the implementation",
						ResponsibleActorKind:    domain.TemplateActorKindQA,
						EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
						CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
						RequiredForParentDone:   true,
					},
				},
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("NewTemplateLibrary() error = %v", err)
	}
	if err := repo.UpsertTemplateLibrary(ctx, library); err != nil {
		t.Fatalf("UpsertTemplateLibrary() error = %v", err)
	}

	loadedLibrary, err := repo.GetTemplateLibrary(ctx, library.ID)
	if err != nil {
		t.Fatalf("GetTemplateLibrary() error = %v", err)
	}
	if len(loadedLibrary.NodeTemplates) != 1 {
		t.Fatalf("len(loadedLibrary.NodeTemplates) = %d, want 1", len(loadedLibrary.NodeTemplates))
	}
	if len(loadedLibrary.NodeTemplates[0].ChildRules) != 1 {
		t.Fatalf("len(loadedLibrary.NodeTemplates[0].ChildRules) = %d, want 1", len(loadedLibrary.NodeTemplates[0].ChildRules))
	}
	if loadedLibrary.NodeTemplates[0].ActionItemMetadataDefaults == nil || loadedLibrary.NodeTemplates[0].ActionItemMetadataDefaults.ValidationPlan != "Run actionItem validation" {
		t.Fatalf("loaded actionItem metadata defaults = %#v, want validation plan", loadedLibrary.NodeTemplates[0].ActionItemMetadataDefaults)
	}
	libraries, err := repo.ListTemplateLibraries(ctx, domain.TemplateLibraryFilter{
		Scope:     domain.TemplateLibraryScopeProject,
		ProjectID: project.ID,
		Status:    domain.TemplateLibraryStatusApproved,
	})
	if err != nil {
		t.Fatalf("ListTemplateLibraries() error = %v", err)
	}
	if len(libraries) != 1 || libraries[0].ID != library.ID {
		t.Fatalf("ListTemplateLibraries() = %#v, want library-1", libraries)
	}

	binding, err := domain.NewProjectTemplateBinding(domain.ProjectTemplateBindingInput{
		ProjectID:        project.ID,
		LibraryID:        library.ID,
		BoundByActorID:   "user-1",
		BoundByActorName: "Operator",
		BoundByActorType: domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewProjectTemplateBinding() error = %v", err)
	}
	if err := repo.UpsertProjectTemplateBinding(ctx, binding); err != nil {
		t.Fatalf("UpsertProjectTemplateBinding() error = %v", err)
	}
	loadedBinding, err := repo.GetProjectTemplateBinding(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProjectTemplateBinding() error = %v", err)
	}
	if loadedBinding.LibraryID != library.ID {
		t.Fatalf("loadedBinding.LibraryID = %q, want %q", loadedBinding.LibraryID, library.ID)
	}
	if err := repo.DeleteProjectTemplateBinding(ctx, project.ID); err != nil {
		t.Fatalf("DeleteProjectTemplateBinding() error = %v", err)
	}
	if _, err := repo.GetProjectTemplateBinding(ctx, project.ID); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("GetProjectTemplateBinding() error = %v, want ErrNotFound after delete", err)
	}
	if err := repo.UpsertProjectTemplateBinding(ctx, binding); err != nil {
		t.Fatalf("UpsertProjectTemplateBinding() restore error = %v", err)
	}

	actionItem, err := domain.NewActionItem(domain.ActionItemInput{
		ID:        "actionItem-1",
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Position:  0,
		Title:     "Build feature",
		Priority:  domain.PriorityHigh,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, actionItem); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	snapshot, err := domain.NewNodeContractSnapshot(domain.NodeContractSnapshotInput{
		NodeID:                    actionItem.ID,
		ProjectID:                 project.ID,
		SourceLibraryID:           library.ID,
		SourceNodeTemplateID:      "actionItem-template",
		SourceChildRuleID:         "qa-check",
		CreatedByActorID:          "tillsyn-system-template",
		CreatedByActorType:        domain.ActorTypeSystem,
		ResponsibleActorKind:      domain.TemplateActorKindQA,
		EditableByActorKinds:      []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds:   []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
		RequiredForParentDone:     true,
		RequiredForContainingDone: true,
	}, now)
	if err != nil {
		t.Fatalf("NewNodeContractSnapshot() error = %v", err)
	}
	if err := repo.CreateNodeContractSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("CreateNodeContractSnapshot() error = %v", err)
	}
	loadedSnapshot, err := repo.GetNodeContractSnapshot(ctx, actionItem.ID)
	if err != nil {
		t.Fatalf("GetNodeContractSnapshot() error = %v", err)
	}
	if loadedSnapshot.SourceLibraryID != library.ID {
		t.Fatalf("loadedSnapshot.SourceLibraryID = %q, want %q", loadedSnapshot.SourceLibraryID, library.ID)
	}
	if len(loadedSnapshot.CompletableByActorKinds) != 2 {
		t.Fatalf("len(loadedSnapshot.CompletableByActorKinds) = %d, want 2", len(loadedSnapshot.CompletableByActorKinds))
	}
	if !loadedSnapshot.RequiredForContainingDone {
		t.Fatal("loadedSnapshot.RequiredForContainingDone = false, want true")
	}
}
