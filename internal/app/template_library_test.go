package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestCreateTaskUsesBoundTemplateLibrary verifies bound node templates override legacy kind templates and persist node contracts.
func TestCreateTaskUsesBoundTemplateLibrary(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 17, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Template Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	library, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "project-library",
		Scope:               domain.TemplateLibraryScopeProject,
		ProjectID:           project.ID,
		Name:                "Project Library",
		Description:         "Applies project-specific QA work",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "user-1",
		CreatedByActorName:  "Operator",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "user-1",
		ApprovedByActorName: "Operator",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{
			{
				ID:         "task-template",
				ScopeLevel: domain.KindAppliesToTask,
				NodeKindID: domain.KindID(domain.WorkKindTask),
				TaskMetadataDefaults: &domain.TaskMetadata{
					ValidationPlan: "Run template validation",
				},
				ChildRules: []UpsertTemplateChildRuleInput{
					{
						ID:                      "qa-check",
						Position:                1,
						ChildScopeLevel:         domain.KindAppliesToSubtask,
						ChildKindID:             domain.KindID(domain.WorkKindSubtask),
						TitleTemplate:           "QA review",
						DescriptionTemplate:     "Verify the parent task output",
						ResponsibleActorKind:    domain.TemplateActorKindQA,
						EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
						CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
						RequiredForParentDone:   true,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpsertTemplateLibrary() error = %v", err)
	}
	if _, err := svc.BindProjectTemplateLibrary(ctx, BindProjectTemplateLibraryInput{
		ProjectID:        project.ID,
		LibraryID:        library.ID,
		BoundByActorID:   "user-1",
		BoundByActorName: "Operator",
		BoundByActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("BindProjectTemplateLibrary() error = %v", err)
	}

	parent, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Build feature",
		Priority:  domain.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if parent.Metadata.ValidationPlan != "Run template validation" {
		t.Fatalf("parent.Metadata.ValidationPlan = %q, want template default", parent.Metadata.ValidationPlan)
	}

	if len(repo.tasks) != 2 {
		t.Fatalf("len(repo.tasks) = %d, want parent plus generated child", len(repo.tasks))
	}
	var generated domain.Task
	for _, candidate := range repo.tasks {
		if candidate.ID == parent.ID {
			continue
		}
		generated = candidate
	}
	if generated.ParentID != parent.ID {
		t.Fatalf("generated.ParentID = %q, want %q", generated.ParentID, parent.ID)
	}
	if generated.Title != "QA review" {
		t.Fatalf("generated.Title = %q, want QA review", generated.Title)
	}
	snapshot, ok := repo.nodeContracts[generated.ID]
	if !ok {
		t.Fatalf("repo.nodeContracts missing generated child %q", generated.ID)
	}
	if snapshot.SourceLibraryID != library.ID {
		t.Fatalf("snapshot.SourceLibraryID = %q, want %q", snapshot.SourceLibraryID, library.ID)
	}
	if !snapshot.RequiredForParentDone {
		t.Fatal("snapshot.RequiredForParentDone = false, want true")
	}
}

// TestCreateProjectUsesApprovedGlobalTemplateLibrary verifies project creation can bind a global library and prefer its project template over legacy kind defaults.
func TestCreateProjectUsesApprovedGlobalTemplateLibrary(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		AutoCreateProjectColumns: false,
	})

	if _, err := svc.ListKindDefinitions(ctx, false); err != nil {
		t.Fatalf("ListKindDefinitions() error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(ctx, CreateKindDefinitionInput{
		ID:          "go-service",
		DisplayName: "Go Service",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToProject},
		Template: domain.KindTemplate{
			ProjectMetadataDefaults: &domain.ProjectMetadata{
				Owner: "legacy-owner",
				Tags:  []string{"legacy"},
			},
			AutoCreateChildren: []domain.KindTemplateChildSpec{{
				Title:       "Legacy Branch",
				Description: "legacy root child",
				Kind:        domain.KindID("branch"),
				AppliesTo:   domain.KindAppliesToBranch,
			}},
		},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(go-service) error = %v", err)
	}
	library, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Description:         "Approved global defaults for Go projects",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "user-1",
		CreatedByActorName:  "Operator",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "user-1",
		ApprovedByActorName: "Operator",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "project-template",
			ScopeLevel: domain.KindAppliesToProject,
			NodeKindID: domain.KindID("go-service"),
			ProjectMetadataDefaults: &domain.ProjectMetadata{
				Owner:             "platform",
				Tags:              []string{"go", "service"},
				StandardsMarkdown: "Run gofmt, tests, and QA",
			},
			ChildRules: []UpsertTemplateChildRuleInput{{
				ID:                      "root-branch",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToBranch,
				ChildKindID:             domain.KindID("branch"),
				TitleTemplate:           "Main Branch",
				DescriptionTemplate:     "default implementation branch",
				ResponsibleActorKind:    domain.TemplateActorKindBuilder,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindBuilder},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindBuilder, domain.TemplateActorKindHuman},
				RequiredForParentDone:   true,
			}},
		}},
	})
	if err != nil {
		t.Fatalf("UpsertTemplateLibrary() error = %v", err)
	}

	project, err := svc.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:              "Go API",
		Kind:              "go-service",
		TemplateLibraryID: library.ID,
		Metadata: domain.ProjectMetadata{
			Tags: []string{"api"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if project.Metadata.Owner != "platform" {
		t.Fatalf("project.Metadata.Owner = %q, want platform", project.Metadata.Owner)
	}
	if got := project.Metadata.Tags; len(got) != 3 || got[0] != "api" || got[1] != "go" || got[2] != "service" {
		t.Fatalf("project.Metadata.Tags = %#v, want api+go+service", got)
	}
	binding, err := svc.GetProjectTemplateBinding(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProjectTemplateBinding() error = %v", err)
	}
	if binding.LibraryID != library.ID {
		t.Fatalf("binding.LibraryID = %q, want %q", binding.LibraryID, library.ID)
	}
	tasks, err := svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1 generated root task", len(tasks))
	}
	if tasks[0].Title != "Main Branch" {
		t.Fatalf("tasks[0].Title = %q, want Main Branch", tasks[0].Title)
	}
	if tasks[0].Kind != domain.WorkKind("branch") || tasks[0].Scope != domain.KindAppliesToBranch {
		t.Fatalf("unexpected generated root task %#v", tasks[0])
	}
	snapshot, ok := repo.nodeContracts[tasks[0].ID]
	if !ok {
		t.Fatalf("repo.nodeContracts missing generated root task %q", tasks[0].ID)
	}
	if snapshot.SourceLibraryID != library.ID {
		t.Fatalf("snapshot.SourceLibraryID = %q, want %q", snapshot.SourceLibraryID, library.ID)
	}
}

// TestUnbindProjectTemplateLibrary verifies project bindings can be removed cleanly for TUI edit flows.
func TestUnbindProjectTemplateLibrary(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Template Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	library, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "user-1",
		CreatedByActorName:  "Operator",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "user-1",
		ApprovedByActorName: "Operator",
		ApprovedByActorType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("UpsertTemplateLibrary() error = %v", err)
	}
	if _, err := svc.BindProjectTemplateLibrary(ctx, BindProjectTemplateLibraryInput{
		ProjectID:        project.ID,
		LibraryID:        library.ID,
		BoundByActorID:   "user-1",
		BoundByActorName: "Operator",
		BoundByActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("BindProjectTemplateLibrary() error = %v", err)
	}

	if err := svc.UnbindProjectTemplateLibrary(ctx, UnbindProjectTemplateLibraryInput{
		ProjectID: project.ID,
	}); err != nil {
		t.Fatalf("UnbindProjectTemplateLibrary() error = %v", err)
	}
	if _, err := svc.GetProjectTemplateBinding(ctx, project.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetProjectTemplateBinding() error = %v, want ErrNotFound", err)
	}
}

// TestCreateTaskFallsBackToLegacyKindTemplate verifies legacy kind-template child generation still works when no binding exists.
func TestCreateTaskFallsBackToLegacyKindTemplate(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 3, 29, 17, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Legacy Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(ctx, CreateKindDefinitionInput{
		ID:          "build-task",
		DisplayName: "Build Task",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToTask},
		Template: domain.KindTemplate{
			AutoCreateChildren: []domain.KindTemplateChildSpec{
				{
					Title:       "Legacy QA review",
					Description: "Verify the implementation",
					Kind:        domain.KindID(domain.WorkKindSubtask),
					AppliesTo:   domain.KindAppliesToSubtask,
				},
			},
		},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition() error = %v", err)
	}
	allowedKinds, err := svc.ListProjectAllowedKinds(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds() error = %v", err)
	}
	allowedKinds = append(allowedKinds, domain.KindID("build-task"))
	if err := svc.SetProjectAllowedKinds(ctx, SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   allowedKinds,
	}); err != nil {
		t.Fatalf("SetProjectAllowedKinds() error = %v", err)
	}
	column, err := svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	parent, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKind("build-task"),
		Scope:     domain.KindAppliesToTask,
		Title:     "Implement feature",
		Priority:  domain.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if len(repo.tasks) != 2 {
		t.Fatalf("len(repo.tasks) = %d, want parent plus generated child", len(repo.tasks))
	}
	for _, candidate := range repo.tasks {
		if candidate.ID == parent.ID {
			continue
		}
		if candidate.Title != "Legacy QA review" {
			t.Fatalf("legacy generated child title = %q, want Legacy QA review", candidate.Title)
		}
		if _, ok := repo.nodeContracts[candidate.ID]; ok {
			t.Fatalf("repo.nodeContracts unexpectedly contains legacy child %q", candidate.ID)
		}
	}
}
