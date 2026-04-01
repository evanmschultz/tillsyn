package app

import (
	"context"
	"errors"
	"slices"
	"strings"
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

// TestGetBuiltinTemplateLibraryStatusMissing verifies builtin lifecycle status reports a missing install and missing kind prerequisites.
func TestGetBuiltinTemplateLibraryStatusMissing(t *testing.T) {
	ctx := context.Background()
	svc := newDeterministicService(newFakeRepo(), time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), ServiceConfig{})

	status, err := svc.GetBuiltinTemplateLibraryStatus(ctx, "default-go")
	if err != nil {
		t.Fatalf("GetBuiltinTemplateLibraryStatus() error = %v", err)
	}
	if status.State != domain.BuiltinTemplateLibraryStateMissing {
		t.Fatalf("status.State = %q, want missing", status.State)
	}
	if status.Installed {
		t.Fatal("status.Installed = true, want false")
	}
	if got, want := len(status.RequiredKindIDs), 4; got != want {
		t.Fatalf("len(status.RequiredKindIDs) = %d, want %d", got, want)
	}
	if got, want := len(status.MissingKindIDs), 4; got != want {
		t.Fatalf("len(status.MissingKindIDs) = %d, want %d", got, want)
	}
}

// TestEnsureBuiltinTemplateLibraryInstallsDefaultGo verifies the supported builtin library installs explicitly once required kinds exist.
func TestEnsureBuiltinTemplateLibraryInstallsDefaultGo(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newDeterministicService(repo, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), ServiceConfig{})
	seedBuiltinTemplateKinds(t, ctx, svc)

	result, err := svc.EnsureBuiltinTemplateLibrary(ctx, EnsureBuiltinTemplateLibraryInput{
		LibraryID: "default-go",
		ActorID:   "dev-1",
		ActorName: "Dev",
		ActorType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("result.Changed = false, want true")
	}
	if result.Status.State != domain.BuiltinTemplateLibraryStateCurrent {
		t.Fatalf("result.Status.State = %q, want current", result.Status.State)
	}
	if len(result.Status.MissingKindIDs) != 0 {
		t.Fatalf("result.Status.MissingKindIDs = %#v, want none", result.Status.MissingKindIDs)
	}
	if !result.Library.BuiltinManaged {
		t.Fatal("result.Library.BuiltinManaged = false, want true")
	}
	if result.Library.BuiltinSource != defaultGoBuiltinLibrarySource {
		t.Fatalf("result.Library.BuiltinSource = %q, want %q", result.Library.BuiltinSource, defaultGoBuiltinLibrarySource)
	}
	if result.Library.BuiltinVersion != defaultGoBuiltinLibraryVersion {
		t.Fatalf("result.Library.BuiltinVersion = %q, want %q", result.Library.BuiltinVersion, defaultGoBuiltinLibraryVersion)
	}
	if got, want := len(result.Library.NodeTemplates), 2; got != want {
		t.Fatalf("len(result.Library.NodeTemplates) = %d, want %d", got, want)
	}
	loaded, err := svc.GetTemplateLibrary(ctx, "default-go")
	if err != nil {
		t.Fatalf("GetTemplateLibrary() error = %v", err)
	}
	if !loaded.BuiltinManaged {
		t.Fatal("loaded.BuiltinManaged = false, want true")
	}
}

// TestGetBuiltinTemplateLibraryStatusDetectsUpdateAvailable verifies status reports update availability when the installed library predates builtin provenance metadata.
func TestGetBuiltinTemplateLibraryStatusDetectsUpdateAvailable(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newDeterministicService(repo, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), ServiceConfig{})
	seedBuiltinTemplateKinds(t, ctx, svc)

	spec := defaultGoBuiltinTemplateLibrarySpec(builtinTemplateActor{
		ID:   "dev-1",
		Name: "Dev",
		Type: domain.ActorTypeUser,
	})
	spec.BuiltinManaged = false
	spec.BuiltinSource = ""
	spec.BuiltinVersion = ""
	if _, err := svc.UpsertTemplateLibrary(ctx, spec); err != nil {
		t.Fatalf("UpsertTemplateLibrary() error = %v", err)
	}

	status, err := svc.GetBuiltinTemplateLibraryStatus(ctx, "default-go")
	if err != nil {
		t.Fatalf("GetBuiltinTemplateLibraryStatus() error = %v", err)
	}
	if status.State != domain.BuiltinTemplateLibraryStateUpdateAvailable {
		t.Fatalf("status.State = %q, want update_available", status.State)
	}
	if !status.Installed {
		t.Fatal("status.Installed = false, want true")
	}
}

// TestGetProjectTemplateReapplyPreviewReportsEligibleGeneratedNodes verifies drift preview surfaces changed rules and conservative eligible migration candidates.
func TestGetProjectTemplateReapplyPreviewReportsEligibleGeneratedNodes(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Template Preview", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if _, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "dev-1",
		CreatedByActorName:  "Dev",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "dev-1",
		ApprovedByActorName: "Dev",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "task-template",
			ScopeLevel: domain.KindAppliesToTask,
			NodeKindID: domain.KindID(domain.WorkKindTask),
			ChildRules: []UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.WorkKindSubtask),
				TitleTemplate:           "QA PASS 1",
				DescriptionTemplate:     "Verify the original contract",
				ResponsibleActorKind:    domain.TemplateActorKindQA,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
				RequiredForParentDone:   true,
			}},
		}},
	}); err != nil {
		t.Fatalf("UpsertTemplateLibrary(rev1) error = %v", err)
	}
	if _, err := svc.BindProjectTemplateLibrary(ctx, BindProjectTemplateLibraryInput{
		ProjectID:        project.ID,
		LibraryID:        "go-defaults",
		BoundByActorID:   "dev-1",
		BoundByActorName: "Dev",
		BoundByActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("BindProjectTemplateLibrary() error = %v", err)
	}
	parent, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Implement preview",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	var generated domain.Task
	for _, task := range repo.tasks {
		if task.ParentID == parent.ID {
			generated = task
			break
		}
	}
	if generated.ID == "" {
		t.Fatal("expected generated QA child task")
	}
	if _, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "dev-1",
		CreatedByActorName:  "Dev",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "dev-1",
		ApprovedByActorName: "Dev",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "task-template",
			ScopeLevel: domain.KindAppliesToTask,
			NodeKindID: domain.KindID(domain.WorkKindTask),
			ChildRules: []UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.WorkKindSubtask),
				TitleTemplate:           "QA PASS 1 REVIEW",
				DescriptionTemplate:     "Verify the latest contract",
				ResponsibleActorKind:    domain.TemplateActorKindQA,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindOrchestrator},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
				RequiredForParentDone:   true,
			}},
		}},
	}); err != nil {
		t.Fatalf("UpsertTemplateLibrary(rev2) error = %v", err)
	}

	preview, err := svc.GetProjectTemplateReapplyPreview(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProjectTemplateReapplyPreview() error = %v", err)
	}
	if preview.DriftStatus != domain.ProjectTemplateBindingDriftUpdateAvailable {
		t.Fatalf("preview.DriftStatus = %q, want update_available", preview.DriftStatus)
	}
	if got := len(preview.ChildRuleChanges); got != 1 {
		t.Fatalf("len(preview.ChildRuleChanges) = %d, want 1", got)
	}
	if got := preview.ChildRuleChanges[0].ChangeKinds; !slices.Equal(got, []string{"title", "description", "editable_by"}) {
		t.Fatalf("preview.ChildRuleChanges[0].ChangeKinds = %#v, want title+description+editable_by", got)
	}
	if got := preview.EligibleMigrationCount; got != 1 {
		t.Fatalf("preview.EligibleMigrationCount = %d, want 1", got)
	}
	if got := preview.IneligibleMigrationCount; got != 0 {
		t.Fatalf("preview.IneligibleMigrationCount = %d, want 0", got)
	}
	if len(preview.MigrationCandidates) != 1 || preview.MigrationCandidates[0].TaskID != generated.ID {
		t.Fatalf("preview.MigrationCandidates = %#v, want generated child %q", preview.MigrationCandidates, generated.ID)
	}
	if preview.MigrationCandidates[0].Status != domain.ProjectTemplateReapplyCandidateEligible {
		t.Fatalf("preview.MigrationCandidates[0].Status = %q, want eligible", preview.MigrationCandidates[0].Status)
	}
}

// TestApproveProjectTemplateMigrationsUpdatesEligibleGeneratedNodes verifies explicit migration approval rewrites the task and stored node contract.
func TestApproveProjectTemplateMigrationsUpdatesEligibleGeneratedNodes(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 1, 11, 10, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Template Approval", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if _, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "dev-1",
		CreatedByActorName:  "Dev",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "dev-1",
		ApprovedByActorName: "Dev",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "task-template",
			ScopeLevel: domain.KindAppliesToTask,
			NodeKindID: domain.KindID(domain.WorkKindTask),
			ChildRules: []UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.WorkKindSubtask),
				TitleTemplate:           "QA PASS 1",
				DescriptionTemplate:     "Verify the original contract",
				ResponsibleActorKind:    domain.TemplateActorKindQA,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
				RequiredForParentDone:   true,
			}},
		}},
	}); err != nil {
		t.Fatalf("UpsertTemplateLibrary(rev1) error = %v", err)
	}
	if _, err := svc.BindProjectTemplateLibrary(ctx, BindProjectTemplateLibraryInput{
		ProjectID:        project.ID,
		LibraryID:        "go-defaults",
		BoundByActorID:   "dev-1",
		BoundByActorName: "Dev",
		BoundByActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("BindProjectTemplateLibrary() error = %v", err)
	}
	parent, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Implement preview",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	var generated domain.Task
	for _, task := range repo.tasks {
		if task.ParentID == parent.ID {
			generated = task
			break
		}
	}
	if generated.ID == "" {
		t.Fatal("expected generated QA child task")
	}
	originalSnapshot := repo.nodeContracts[generated.ID]
	if _, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "dev-1",
		CreatedByActorName:  "Dev",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "dev-1",
		ApprovedByActorName: "Dev",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "task-template",
			ScopeLevel: domain.KindAppliesToTask,
			NodeKindID: domain.KindID(domain.WorkKindTask),
			ChildRules: []UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.WorkKindSubtask),
				TitleTemplate:           "QA PASS 1 REVIEW",
				DescriptionTemplate:     "Verify the latest contract",
				ResponsibleActorKind:    domain.TemplateActorKindQA,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindOrchestrator},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
				RequiredForParentDone:   true,
			}},
		}},
	}); err != nil {
		t.Fatalf("UpsertTemplateLibrary(rev2) error = %v", err)
	}

	result, err := svc.ApproveProjectTemplateMigrations(ctx, ApproveProjectTemplateMigrationsInput{
		ProjectID:      project.ID,
		TaskIDs:        []string{generated.ID},
		ApprovedBy:     "dev-2",
		ApprovedByName: "Dev Two",
		ApprovedByType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("ApproveProjectTemplateMigrations() error = %v", err)
	}
	if result.AppliedCount != 1 || len(result.Approvals) != 1 {
		t.Fatalf("ApproveProjectTemplateMigrations() = %#v, want one applied migration", result)
	}
	updatedTask := repo.tasks[generated.ID]
	if updatedTask.Title != "QA PASS 1 REVIEW" {
		t.Fatalf("updated task title = %q, want QA PASS 1 REVIEW", updatedTask.Title)
	}
	if updatedTask.Description != "Verify the latest contract" {
		t.Fatalf("updated task description = %q, want latest contract", updatedTask.Description)
	}
	if updatedTask.UpdatedByActor != "dev-2" {
		t.Fatalf("updated task actor = %q, want dev-2", updatedTask.UpdatedByActor)
	}
	updatedSnapshot := repo.nodeContracts[generated.ID]
	if !slices.Equal(updatedSnapshot.EditableByActorKinds, []domain.TemplateActorKind{domain.TemplateActorKindOrchestrator, domain.TemplateActorKindQA}) {
		t.Fatalf("updated snapshot editable kinds = %#v, want orchestrator+qa", updatedSnapshot.EditableByActorKinds)
	}
	if !updatedSnapshot.CreatedAt.Equal(originalSnapshot.CreatedAt) {
		t.Fatalf("updated snapshot created_at = %v, want preserved %v", updatedSnapshot.CreatedAt, originalSnapshot.CreatedAt)
	}
	preview, err := svc.GetProjectTemplateReapplyPreview(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProjectTemplateReapplyPreview() error = %v", err)
	}
	if preview.EligibleMigrationCount != 0 {
		t.Fatalf("preview.EligibleMigrationCount = %d, want 0 after approval", preview.EligibleMigrationCount)
	}
}

// TestGetProjectTemplateReapplyPreviewMarksModifiedGeneratedNodesIneligible verifies non-system edits block automatic migration eligibility.
func TestGetProjectTemplateReapplyPreviewMarksModifiedGeneratedNodesIneligible(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 1, 11, 30, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Template Preview", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if _, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "dev-1",
		CreatedByActorName:  "Dev",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "dev-1",
		ApprovedByActorName: "Dev",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "task-template",
			ScopeLevel: domain.KindAppliesToTask,
			NodeKindID: domain.KindID(domain.WorkKindTask),
			ChildRules: []UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.WorkKindSubtask),
				TitleTemplate:           "QA PASS 1",
				DescriptionTemplate:     "Verify the original contract",
				ResponsibleActorKind:    domain.TemplateActorKindQA,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
				RequiredForParentDone:   true,
			}},
		}},
	}); err != nil {
		t.Fatalf("UpsertTemplateLibrary(rev1) error = %v", err)
	}
	if _, err := svc.BindProjectTemplateLibrary(ctx, BindProjectTemplateLibraryInput{
		ProjectID:        project.ID,
		LibraryID:        "go-defaults",
		BoundByActorID:   "dev-1",
		BoundByActorName: "Dev",
		BoundByActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("BindProjectTemplateLibrary() error = %v", err)
	}
	parent, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Title:     "Implement preview",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	var generated domain.Task
	for _, task := range repo.tasks {
		if task.ParentID == parent.ID {
			generated = task
			break
		}
	}
	if generated.ID == "" {
		t.Fatal("expected generated QA child task")
	}
	if _, err := svc.UpdateTask(ctx, UpdateTaskInput{
		TaskID:        generated.ID,
		Title:         generated.Title,
		Description:   generated.Description,
		Priority:      generated.Priority,
		DueAt:         generated.DueAt,
		Labels:        generated.Labels,
		Metadata:      &generated.Metadata,
		UpdatedBy:     "dev-2",
		UpdatedByName: "Dev Two",
		UpdatedType:   domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if _, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "dev-1",
		CreatedByActorName:  "Dev",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "dev-1",
		ApprovedByActorName: "Dev",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "task-template",
			ScopeLevel: domain.KindAppliesToTask,
			NodeKindID: domain.KindID(domain.WorkKindTask),
			ChildRules: []UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.WorkKindSubtask),
				TitleTemplate:           "QA PASS 1 REVIEW",
				DescriptionTemplate:     "Verify the latest contract",
				ResponsibleActorKind:    domain.TemplateActorKindQA,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindOrchestrator},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
				RequiredForParentDone:   true,
			}},
		}},
	}); err != nil {
		t.Fatalf("UpsertTemplateLibrary(rev2) error = %v", err)
	}

	preview, err := svc.GetProjectTemplateReapplyPreview(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProjectTemplateReapplyPreview() error = %v", err)
	}
	if got := preview.EligibleMigrationCount; got != 0 {
		t.Fatalf("preview.EligibleMigrationCount = %d, want 0", got)
	}
	if got := preview.IneligibleMigrationCount; got != 1 {
		t.Fatalf("preview.IneligibleMigrationCount = %d, want 1", got)
	}
	if len(preview.MigrationCandidates) != 1 {
		t.Fatalf("len(preview.MigrationCandidates) = %d, want 1", len(preview.MigrationCandidates))
	}
	if preview.MigrationCandidates[0].Status != domain.ProjectTemplateReapplyCandidateIneligible {
		t.Fatalf("preview.MigrationCandidates[0].Status = %q, want ineligible", preview.MigrationCandidates[0].Status)
	}
	if got := preview.MigrationCandidates[0].Reason; !strings.Contains(got, "updated since generation") {
		t.Fatalf("preview.MigrationCandidates[0].Reason = %q, want updated since generation", got)
	}
}

// seedBuiltinTemplateKinds installs the builtin default-go prerequisite kinds used by lifecycle tests.
func seedBuiltinTemplateKinds(t *testing.T, ctx context.Context, svc *Service) {
	t.Helper()
	for _, spec := range []CreateKindDefinitionInput{
		{ID: "go-project", DisplayName: "Go Project", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToProject}},
		{ID: "implementation-phase", DisplayName: "Implementation Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-task", DisplayName: "Build Task", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: "qa-check", DisplayName: "QA Check", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
	} {
		if _, err := svc.UpsertKindDefinition(ctx, spec); err != nil {
			t.Fatalf("UpsertKindDefinition(%q) error = %v", spec.ID, err)
		}
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
