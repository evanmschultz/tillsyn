package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
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
	allowedKinds, err := svc.ListProjectAllowedKinds(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds() error = %v", err)
	}
	if got, want := allowedKinds, []domain.KindID{"branch", "go-service"}; !slices.Equal(got, want) {
		t.Fatalf("ListProjectAllowedKinds() = %#v, want %#v", got, want)
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

// TestBindProjectTemplateLibraryRefreshesDefaultAllowlist verifies binding tightens the default catalog-wide allowlist to the library-scoped kinds.
func TestBindProjectTemplateLibraryRefreshesDefaultAllowlist(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Template Policy", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	defaultKinds, err := svc.ListProjectAllowedKinds(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds(default) error = %v", err)
	}
	if !slices.Contains(defaultKinds, domain.KindID("note")) {
		t.Fatalf("default allowlist = %#v, want generic kind note present", defaultKinds)
	}
	library, err := svc.UpsertTemplateLibrary(ctx, UpsertTemplateLibraryInput{
		ID:                  "workflow-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Workflow Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "user-1",
		CreatedByActorName:  "Operator",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "user-1",
		ApprovedByActorName: "Operator",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "phase-template",
			ScopeLevel: domain.KindAppliesToPhase,
			NodeKindID: domain.KindID("phase"),
			ChildRules: []UpsertTemplateChildRuleInput{{
				ID:                      "build-child",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToTask,
				ChildKindID:             domain.KindID("task"),
				TitleTemplate:           "Build child",
				ResponsibleActorKind:    domain.TemplateActorKindBuilder,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindBuilder},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindBuilder, domain.TemplateActorKindHuman},
			}},
		}},
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
	allowedKinds, err := svc.ListProjectAllowedKinds(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds(bound) error = %v", err)
	}
	if got, want := allowedKinds, []domain.KindID{"phase", "project", "task"}; !slices.Equal(got, want) {
		t.Fatalf("ListProjectAllowedKinds() = %#v, want %#v", got, want)
	}
}

// TestBindProjectTemplateLibraryPreservesCustomizedAllowlist verifies binding does not overwrite an explicitly curated project allowlist.
func TestBindProjectTemplateLibraryPreservesCustomizedAllowlist(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 11, 10, 30, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(ctx, "Custom Policy", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := svc.SetProjectAllowedKinds(ctx, SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   []domain.KindID{"note", "project"},
	}); err != nil {
		t.Fatalf("SetProjectAllowedKinds() error = %v", err)
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
		NodeTemplates: []UpsertNodeTemplateInput{{
			ID:         "task-template",
			ScopeLevel: domain.KindAppliesToTask,
			NodeKindID: domain.KindID("task"),
		}},
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
	allowedKinds, err := svc.ListProjectAllowedKinds(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds() error = %v", err)
	}
	if got, want := allowedKinds, []domain.KindID{"note", "project"}; !slices.Equal(got, want) {
		t.Fatalf("ListProjectAllowedKinds() = %#v, want %#v", got, want)
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
	if got, want := len(status.RequiredKindIDs), 16; got != want {
		t.Fatalf("len(status.RequiredKindIDs) = %d, want %d", got, want)
	}
	if got, want := len(status.MissingKindIDs), 13; got != want {
		t.Fatalf("len(status.MissingKindIDs) = %d, want %d", got, want)
	}
	for _, want := range []domain.KindID{"branch", "task", "subtask", "go-project", "project-setup-phase", "plan-phase", "build-phase", "closeout-phase", "branch-cleanup-phase", "refactor-phase", "dogfood-refactor-phase", "build-task", "refactor-task", "dogfood-refactor-task", "qa-check", "commit-and-reingest"} {
		if !slices.Contains(status.RequiredKindIDs, want) {
			t.Fatalf("status.RequiredKindIDs missing %q: %#v", want, status.RequiredKindIDs)
		}
	}
	for _, want := range []domain.KindID{"go-project", "project-setup-phase", "plan-phase", "build-phase", "closeout-phase", "branch-cleanup-phase", "refactor-phase", "dogfood-refactor-phase", "build-task", "refactor-task", "dogfood-refactor-task", "qa-check", "commit-and-reingest"} {
		if !slices.Contains(status.MissingKindIDs, want) {
			t.Fatalf("status.MissingKindIDs missing %q: %#v", want, status.MissingKindIDs)
		}
	}
}

// TestDefaultGoBuiltinTemplateLibrarySpecLoadsRepoSource verifies the builtin default-go contract loads from the repo-visible template source.
func TestDefaultGoBuiltinTemplateLibrarySpecLoadsRepoSource(t *testing.T) {
	spec, err := defaultGoBuiltinTemplateLibrarySpec(builtinTemplateActor{})
	if err != nil {
		t.Fatalf("defaultGoBuiltinTemplateLibrarySpec() error = %v", err)
	}
	if spec.ID != "default-go" {
		t.Fatalf("spec.ID = %q, want default-go", spec.ID)
	}
	if spec.BuiltinSource != "builtin://tillsyn/default-go" {
		t.Fatalf("spec.BuiltinSource = %q, want builtin://tillsyn/default-go", spec.BuiltinSource)
	}
	if spec.BuiltinVersion != "2026-04-13.1" {
		t.Fatalf("spec.BuiltinVersion = %q, want 2026-04-13.1", spec.BuiltinVersion)
	}
	if got, want := len(spec.NodeTemplates), 12; got != want {
		t.Fatalf("len(spec.NodeTemplates) = %d, want %d", got, want)
	}
	projectDefaults := spec.NodeTemplates[0].ProjectMetadataDefaults
	if projectDefaults == nil || strings.TrimSpace(projectDefaults.StandardsMarkdown) == "" {
		t.Fatalf("spec.NodeTemplates[0].ProjectMetadataDefaults = %#v, want standards markdown", projectDefaults)
	}
	if got := spec.NodeTemplates[0].ChildRules[0].TitleTemplate; got != "PROJECT SETUP" {
		t.Fatalf("project root child title = %q, want PROJECT SETUP", got)
	}
	branchTemplate := spec.NodeTemplates[2]
	if branchTemplate.NodeKindID != "branch" {
		t.Fatalf("branch template node kind = %q, want branch", branchTemplate.NodeKindID)
	}
	if got, want := len(branchTemplate.ChildRules), 4; got != want {
		t.Fatalf("len(branchTemplate.ChildRules) = %d, want %d", got, want)
	}
	if got := []string{
		branchTemplate.ChildRules[0].TitleTemplate,
		branchTemplate.ChildRules[1].TitleTemplate,
		branchTemplate.ChildRules[2].TitleTemplate,
		branchTemplate.ChildRules[3].TitleTemplate,
	}; !slices.Equal(got, []string{"PLAN", "BUILD", "CLOSEOUT", "BRANCH CLEANUP"}) {
		t.Fatalf("branch child titles = %#v, want PLAN/BUILD/CLOSEOUT/BRANCH CLEANUP", got)
	}
	if got := childRuleTitles(findNodeTemplateByKind(t, spec.NodeTemplates, "refactor-phase").ChildRules); !slices.Equal(got, []string{"HYLLA-FIRST REFACTOR BASELINE", "PHASE METRICS ROLLUP", "PHASE PARITY VALIDATION PLAN", "PHASE PUSH AND REINGEST CONFIRMATION", "REFACTOR METRICS BASELINE AND REPORT PATH", "REFACTOR SUBPHASE AND SLICE TREE"}) {
		t.Fatalf("refactor-phase child titles = %#v", got)
	}
	if got := childRuleTitles(findNodeTemplateByKind(t, spec.NodeTemplates, "dogfood-refactor-phase").ChildRules); !slices.Equal(got, []string{"CONFIRM LOCAL USED VERSION UPDATED", "DEV VERSION VALIDATION PLAN", "HYLLA-FIRST REFACTOR BASELINE", "PHASE METRICS ROLLUP", "PHASE PUSH AND REINGEST CONFIRMATION", "REFACTOR METRICS BASELINE AND REPORT PATH", "REFACTOR SUBPHASE AND SLICE TREE"}) {
		t.Fatalf("dogfood-refactor-phase child titles = %#v", got)
	}
	if got := childRuleTitles(findNodeTemplateByKind(t, spec.NodeTemplates, "refactor-task").ChildRules); !slices.Equal(got, []string{"COMMIT PUSH AND REINGEST", "METRICS CAPTURE AND REPORT", "PARITY VALIDATION IN ACTION", "QA FALSIFICATION REVIEW", "QA PROOF REVIEW"}) {
		t.Fatalf("refactor-task child titles = %#v", got)
	}
	if got := childRuleTitles(findNodeTemplateByKind(t, spec.NodeTemplates, "dogfood-refactor-task").ChildRules); !slices.Equal(got, []string{"COMMIT PUSH AND REINGEST", "CONFIRM LOCAL USED VERSION UPDATED", "METRICS CAPTURE AND REPORT", "QA FALSIFICATION REVIEW", "QA PROOF REVIEW", "TEST AGAINST DEV VERSION"}) {
		t.Fatalf("dogfood-refactor-task child titles = %#v", got)
	}
}

// TestGetBuiltinTemplateLibraryStatusMissingDefaultFrontend verifies builtin lifecycle status reports a missing install and missing frontend kind prerequisites.
func TestGetBuiltinTemplateLibraryStatusMissingDefaultFrontend(t *testing.T) {
	ctx := context.Background()
	svc := newDeterministicService(newFakeRepo(), time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), ServiceConfig{})

	status, err := svc.GetBuiltinTemplateLibraryStatus(ctx, "default-frontend")
	if err != nil {
		t.Fatalf("GetBuiltinTemplateLibraryStatus() error = %v", err)
	}
	if status.State != domain.BuiltinTemplateLibraryStateMissing {
		t.Fatalf("status.State = %q, want missing", status.State)
	}
	if status.Installed {
		t.Fatal("status.Installed = true, want false")
	}
	if got, want := len(status.RequiredKindIDs), 19; got != want {
		t.Fatalf("len(status.RequiredKindIDs) = %d, want %d", got, want)
	}
	if got, want := len(status.MissingKindIDs), 16; got != want {
		t.Fatalf("len(status.MissingKindIDs) = %d, want %d", got, want)
	}
	for _, want := range []domain.KindID{"branch", "task", "subtask", "frontend-project", "project-setup-phase", "plan-phase", "build-phase", "closeout-phase", "branch-cleanup-phase", "refactor-phase", "dogfood-refactor-phase", "build-task", "refactor-task", "dogfood-refactor-task", "qa-check", "visual-qa", "a11y-check", "design-review", "commit-and-reingest"} {
		if !slices.Contains(status.RequiredKindIDs, want) {
			t.Fatalf("status.RequiredKindIDs missing %q: %#v", want, status.RequiredKindIDs)
		}
	}
	for _, want := range []domain.KindID{"frontend-project", "project-setup-phase", "plan-phase", "build-phase", "closeout-phase", "branch-cleanup-phase", "refactor-phase", "dogfood-refactor-phase", "build-task", "refactor-task", "dogfood-refactor-task", "qa-check", "visual-qa", "a11y-check", "design-review", "commit-and-reingest"} {
		if !slices.Contains(status.MissingKindIDs, want) {
			t.Fatalf("status.MissingKindIDs missing %q: %#v", want, status.MissingKindIDs)
		}
	}
}

// TestDefaultFrontendBuiltinTemplateLibrarySpecLoadsRepoSource verifies the builtin default-frontend contract loads from the repo-visible template source.
func TestDefaultFrontendBuiltinTemplateLibrarySpecLoadsRepoSource(t *testing.T) {
	spec, err := defaultFrontendBuiltinTemplateLibrarySpec(builtinTemplateActor{})
	if err != nil {
		t.Fatalf("defaultFrontendBuiltinTemplateLibrarySpec() error = %v", err)
	}
	if spec.ID != "default-frontend" {
		t.Fatalf("spec.ID = %q, want default-frontend", spec.ID)
	}
	if spec.BuiltinSource != "builtin://tillsyn/default-frontend" {
		t.Fatalf("spec.BuiltinSource = %q, want builtin://tillsyn/default-frontend", spec.BuiltinSource)
	}
	if spec.BuiltinVersion != "2026-04-13.1" {
		t.Fatalf("spec.BuiltinVersion = %q, want 2026-04-13.1", spec.BuiltinVersion)
	}
	if got, want := len(spec.NodeTemplates), 12; got != want {
		t.Fatalf("len(spec.NodeTemplates) = %d, want %d", got, want)
	}
	projectDefaults := spec.NodeTemplates[0].ProjectMetadataDefaults
	if projectDefaults == nil || strings.TrimSpace(projectDefaults.StandardsMarkdown) == "" {
		t.Fatalf("spec.NodeTemplates[0].ProjectMetadataDefaults = %#v, want standards markdown", projectDefaults)
	}
	if got := spec.NodeTemplates[0].ChildRules[0].TitleTemplate; got != "PROJECT SETUP" {
		t.Fatalf("project root child title = %q, want PROJECT SETUP", got)
	}
	var buildTaskTemplate UpsertNodeTemplateInput
	for _, nodeTemplate := range spec.NodeTemplates {
		if nodeTemplate.NodeKindID == "build-task" {
			buildTaskTemplate = nodeTemplate
			break
		}
	}
	if buildTaskTemplate.ID == "" {
		t.Fatal("expected build-task node template in default-frontend builtin")
	}
	gotChildTitles := make([]string, 0, len(buildTaskTemplate.ChildRules))
	for _, childRule := range buildTaskTemplate.ChildRules {
		gotChildTitles = append(gotChildTitles, childRule.TitleTemplate)
	}
	slices.Sort(gotChildTitles)
	if want := []string{"ACCESSIBILITY CHECK", "COMMIT PUSH AND REINGEST", "QA FALSIFICATION REVIEW", "QA PROOF REVIEW", "VISUAL QA"}; !slices.Equal(gotChildTitles, want) {
		t.Fatalf("build-task child titles = %#v, want %#v", gotChildTitles, want)
	}
	if got := childRuleTitles(findNodeTemplateByKind(t, spec.NodeTemplates, "refactor-task").ChildRules); !slices.Equal(got, []string{"ACCESSIBILITY CHECK", "COMMIT PUSH AND REINGEST", "METRICS CAPTURE AND REPORT", "PARITY VALIDATION IN ACTION", "QA FALSIFICATION REVIEW", "QA PROOF REVIEW", "VISUAL QA"}) {
		t.Fatalf("refactor-task child titles = %#v", got)
	}
	if got := childRuleTitles(findNodeTemplateByKind(t, spec.NodeTemplates, "dogfood-refactor-task").ChildRules); !slices.Equal(got, []string{"ACCESSIBILITY CHECK", "COMMIT PUSH AND REINGEST", "CONFIRM LOCAL USED VERSION UPDATED", "METRICS CAPTURE AND REPORT", "QA FALSIFICATION REVIEW", "QA PROOF REVIEW", "TEST AGAINST DEV VERSION", "VISUAL QA"}) {
		t.Fatalf("dogfood-refactor-task child titles = %#v", got)
	}
}

// TestEnsureBuiltinTemplateLibraryInstallsDefaultGo verifies the supported builtin library installs explicitly once required kinds exist.
func TestEnsureBuiltinTemplateLibraryInstallsDefaultGo(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newDeterministicService(repo, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), ServiceConfig{})
	seedBuiltinTemplateKinds(t, ctx, svc)
	spec, err := defaultGoBuiltinTemplateLibrarySpec(builtinTemplateActor{
		ID:   "dev-1",
		Name: "Dev",
		Type: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("defaultGoBuiltinTemplateLibrarySpec() error = %v", err)
	}

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
	if result.Library.BuiltinSource != spec.BuiltinSource {
		t.Fatalf("result.Library.BuiltinSource = %q, want %q", result.Library.BuiltinSource, spec.BuiltinSource)
	}
	if result.Library.BuiltinVersion != spec.BuiltinVersion {
		t.Fatalf("result.Library.BuiltinVersion = %q, want %q", result.Library.BuiltinVersion, spec.BuiltinVersion)
	}
	if got, want := len(result.Library.NodeTemplates), 12; got != want {
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

// TestEnsureBuiltinTemplateLibraryInstallsDefaultFrontend verifies the supported builtin frontend library installs explicitly once required kinds exist.
func TestEnsureBuiltinTemplateLibraryInstallsDefaultFrontend(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newDeterministicService(repo, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), ServiceConfig{})
	seedDefaultFrontendBuiltinTemplateKinds(t, ctx, svc)
	spec, err := defaultFrontendBuiltinTemplateLibrarySpec(builtinTemplateActor{
		ID:   "dev-1",
		Name: "Dev",
		Type: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("defaultFrontendBuiltinTemplateLibrarySpec() error = %v", err)
	}

	result, err := svc.EnsureBuiltinTemplateLibrary(ctx, EnsureBuiltinTemplateLibraryInput{
		LibraryID: "default-frontend",
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
	if result.Library.BuiltinSource != spec.BuiltinSource {
		t.Fatalf("result.Library.BuiltinSource = %q, want %q", result.Library.BuiltinSource, spec.BuiltinSource)
	}
	if result.Library.BuiltinVersion != spec.BuiltinVersion {
		t.Fatalf("result.Library.BuiltinVersion = %q, want %q", result.Library.BuiltinVersion, spec.BuiltinVersion)
	}
	if got, want := len(result.Library.NodeTemplates), 12; got != want {
		t.Fatalf("len(result.Library.NodeTemplates) = %d, want %d", got, want)
	}
	loaded, err := svc.GetTemplateLibrary(ctx, "default-frontend")
	if err != nil {
		t.Fatalf("GetTemplateLibrary() error = %v", err)
	}
	if !loaded.BuiltinManaged {
		t.Fatal("loaded.BuiltinManaged = false, want true")
	}
}

// TestEnsureBuiltinTemplateLibraryReportsBootstrapRequiredWhenBuiltinKindsMissing verifies builtin ensure
// fails with bootstrap guidance instead of a misleading not-found error when the active DB lacks typed kinds.
func TestEnsureBuiltinTemplateLibraryReportsBootstrapRequiredWhenBuiltinKindsMissing(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newDeterministicService(repo, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), ServiceConfig{})

	_, err := svc.EnsureBuiltinTemplateLibrary(ctx, EnsureBuiltinTemplateLibraryInput{
		LibraryID: "default-go",
		ActorID:   "dev-1",
		ActorName: "Dev",
		ActorType: domain.ActorTypeUser,
	})
	if !errors.Is(err, domain.ErrBuiltinTemplateBootstrapRequired) {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v, want ErrBuiltinTemplateBootstrapRequired", err)
	}
	for _, want := range []string{
		`builtin template "default-go"`,
		"active runtime DB",
		"confirm you are on the intended stable or dev runtime",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("EnsureBuiltinTemplateLibrary() error = %q, want substring %q", err.Error(), want)
		}
	}
}

// TestDefaultGoBuiltinTemplateLibraryAppliesExpandedWorkflow verifies the shipped builtin default-go contract
// generates project setup at project creation, branch lifecycle phases at lane creation, and QA work for build tasks.
func TestDefaultGoBuiltinTemplateLibraryAppliesExpandedWorkflow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{AutoCreateProjectColumns: false})
	seedBuiltinTemplateKinds(t, ctx, svc)

	if _, err := svc.EnsureBuiltinTemplateLibrary(ctx, EnsureBuiltinTemplateLibraryInput{
		LibraryID: "default-go",
		ActorID:   "dev-1",
		ActorName: "Dev",
		ActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v", err)
	}

	project, err := svc.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:              "Tillsyn",
		Kind:              "go-project",
		TemplateLibraryID: "default-go",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	tasks, err := svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(project roots) error = %v", err)
	}
	rootTasks := make([]domain.Task, 0)
	for _, task := range tasks {
		if strings.TrimSpace(task.ParentID) == "" {
			rootTasks = append(rootTasks, task)
		}
	}
	if got, want := len(rootTasks), 1; got != want {
		t.Fatalf("len(rootTasks) = %d, want %d", got, want)
	}
	projectSetup := rootTasks[0]
	if projectSetup.Title != "PROJECT SETUP" || projectSetup.Kind != domain.WorkKind("project-setup-phase") || projectSetup.Scope != domain.KindAppliesToPhase {
		t.Fatalf("project setup root = %#v, want PROJECT SETUP/project-setup-phase/phase", projectSetup)
	}
	projectSetupContract, ok := repo.nodeContracts[projectSetup.ID]
	if !ok {
		t.Fatalf("repo.nodeContracts missing project setup %q", projectSetup.ID)
	}
	if projectSetupContract.SourceChildRuleID != "project-setup" {
		t.Fatalf("project setup contract source child rule = %q, want project-setup", projectSetupContract.SourceChildRuleID)
	}

	columns, err := svc.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected template root column for branch lane creation")
	}

	branch, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "MAIN DOGFOOD LANE",
	})
	if err != nil {
		t.Fatalf("CreateTask(branch) error = %v", err)
	}

	tasks, err = svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(branch children) error = %v", err)
	}
	branchChildren := make([]domain.Task, 0)
	branchPhaseKinds := map[string]domain.WorkKind{}
	for _, task := range tasks {
		if task.ParentID != branch.ID {
			continue
		}
		branchChildren = append(branchChildren, task)
		branchPhaseKinds[task.Title] = task.Kind
	}
	gotBranchTitles := make([]string, 0, len(branchChildren))
	for _, task := range branchChildren {
		gotBranchTitles = append(gotBranchTitles, task.Title)
	}
	slices.Sort(gotBranchTitles)
	wantBranchTitles := []string{"BRANCH CLEANUP", "BUILD", "CLOSEOUT", "PLAN"}
	if !slices.Equal(gotBranchTitles, wantBranchTitles) {
		t.Fatalf("branch child titles = %#v, want %#v", gotBranchTitles, wantBranchTitles)
	}
	if branchPhaseKinds["PLAN"] != domain.WorkKind("plan-phase") ||
		branchPhaseKinds["BUILD"] != domain.WorkKind("build-phase") ||
		branchPhaseKinds["CLOSEOUT"] != domain.WorkKind("closeout-phase") ||
		branchPhaseKinds["BRANCH CLEANUP"] != domain.WorkKind("branch-cleanup-phase") {
		t.Fatalf("branch phase kinds = %#v, want plan/build/closeout/branch-cleanup phase kinds", branchPhaseKinds)
	}

	var buildPhase domain.Task
	for _, task := range branchChildren {
		if task.Title == "BUILD" {
			buildPhase = task
			break
		}
	}
	if buildPhase.ID == "" {
		t.Fatal("expected generated BUILD phase")
	}
	if got, want := childTitles(tasks, buildPhase.ID), []string{"PHASE PUSH AND REINGEST CONFIRMATION"}; !slices.Equal(got, want) {
		t.Fatalf("build phase child titles = %#v, want %#v", got, want)
	}

	buildTask, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  buildPhase.ID,
		ColumnID:  buildPhase.ColumnID,
		Kind:      domain.WorkKind("build-task"),
		Scope:     domain.KindAppliesToTask,
		Title:     "IMPLEMENT TEMPLATE UPDATE",
	})
	if err != nil {
		t.Fatalf("CreateTask(build-task) error = %v", err)
	}

	tasks, err = svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(build-task children) error = %v", err)
	}
	buildTaskChildren := make([]domain.Task, 0)
	for _, task := range tasks {
		if task.ParentID == buildTask.ID {
			buildTaskChildren = append(buildTaskChildren, task)
		}
	}
	gotQATitles := make([]string, 0, len(buildTaskChildren))
	for _, task := range buildTaskChildren {
		gotQATitles = append(gotQATitles, task.Title)
	}
	slices.Sort(gotQATitles)
	if want := []string{"COMMIT PUSH AND REINGEST", "QA FALSIFICATION REVIEW", "QA PROOF REVIEW"}; !slices.Equal(gotQATitles, want) {
		t.Fatalf("build-task child titles = %#v, want %#v", gotQATitles, want)
	}
	for _, task := range buildTaskChildren {
		snapshot, ok := repo.nodeContracts[task.ID]
		if !ok {
			t.Fatalf("repo.nodeContracts missing generated QA child %q", task.ID)
		}
		if !snapshot.RequiredForParentDone {
			t.Fatalf("snapshot.RequiredForParentDone for %q = false, want true", task.Title)
		}
		if task.Title == "COMMIT PUSH AND REINGEST" {
			if snapshot.ResponsibleActorKind != domain.TemplateActorKindBuilder {
				t.Fatalf("commit-and-reingest responsible actor = %q, want builder", snapshot.ResponsibleActorKind)
			}
			if !slices.Contains(snapshot.EditableByActorKinds, domain.TemplateActorKindBuilder) || !slices.Contains(snapshot.EditableByActorKinds, domain.TemplateActorKindOrchestrator) {
				t.Fatalf("commit-and-reingest editable actors = %#v, want builder+orchestrator", snapshot.EditableByActorKinds)
			}
			if !slices.Contains(snapshot.CompletableByActorKinds, domain.TemplateActorKindHuman) || !slices.Contains(snapshot.CompletableByActorKinds, domain.TemplateActorKindBuilder) {
				t.Fatalf("commit-and-reingest completable actors = %#v, want builder+human", snapshot.CompletableByActorKinds)
			}
			continue
		}
		if snapshot.ResponsibleActorKind != domain.TemplateActorKindQA {
			t.Fatalf("snapshot.ResponsibleActorKind = %q, want qa", snapshot.ResponsibleActorKind)
		}
		if !slices.Contains(snapshot.EditableByActorKinds, domain.TemplateActorKindQA) {
			t.Fatalf("snapshot.EditableByActorKinds for %q = %#v, want qa", task.Title, snapshot.EditableByActorKinds)
		}
		if !slices.Contains(snapshot.CompletableByActorKinds, domain.TemplateActorKindHuman) || !slices.Contains(snapshot.CompletableByActorKinds, domain.TemplateActorKindQA) {
			t.Fatalf("snapshot.CompletableByActorKinds for %q = %#v, want qa+human", task.Title, snapshot.CompletableByActorKinds)
		}
	}
}

// TestDefaultFrontendBuiltinTemplateLibraryAppliesExpandedWorkflow verifies the shipped builtin default-frontend contract
// generates project setup, branch lifecycle phases, plan guidance, and frontend QA work for build tasks.
func TestDefaultFrontendBuiltinTemplateLibraryAppliesExpandedWorkflow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{AutoCreateProjectColumns: false})
	seedDefaultFrontendBuiltinTemplateKinds(t, ctx, svc)

	if _, err := svc.EnsureBuiltinTemplateLibrary(ctx, EnsureBuiltinTemplateLibraryInput{
		LibraryID: "default-frontend",
		ActorID:   "dev-1",
		ActorName: "Dev",
		ActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v", err)
	}

	project, err := svc.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:              "Frontend",
		Kind:              "frontend-project",
		TemplateLibraryID: "default-frontend",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	tasks, err := svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(project roots) error = %v", err)
	}
	rootTasks := make([]domain.Task, 0)
	for _, task := range tasks {
		if strings.TrimSpace(task.ParentID) == "" {
			rootTasks = append(rootTasks, task)
		}
	}
	if got, want := len(rootTasks), 1; got != want {
		t.Fatalf("len(rootTasks) = %d, want %d", got, want)
	}
	projectSetup := rootTasks[0]
	if projectSetup.Title != "PROJECT SETUP" || projectSetup.Kind != domain.WorkKind("project-setup-phase") || projectSetup.Scope != domain.KindAppliesToPhase {
		t.Fatalf("project setup root = %#v, want PROJECT SETUP/project-setup-phase/phase", projectSetup)
	}

	columns, err := svc.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected template root column for branch lane creation")
	}

	branch, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "FRONTEND LANE",
	})
	if err != nil {
		t.Fatalf("CreateTask(branch) error = %v", err)
	}

	tasks, err = svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(branch children) error = %v", err)
	}
	branchChildren := make([]domain.Task, 0)
	for _, task := range tasks {
		if task.ParentID == branch.ID {
			branchChildren = append(branchChildren, task)
		}
	}
	if got, want := childTitles(tasks, branch.ID), []string{"BRANCH CLEANUP", "BUILD", "CLOSEOUT", "PLAN"}; !slices.Equal(got, want) {
		t.Fatalf("branch child titles = %#v, want %#v", got, want)
	}

	planPhase := findTaskByTitle(t, branchChildren, "PLAN")
	if got, want := childTitles(tasks, planPhase.ID), []string{
		"BRANCH SETUP",
		"BUILD TASK TREE",
		"CLOSEOUT AND CLEANUP EXPECTATIONS",
		"CONTEXT7 AND BROWSER RESEARCH",
		"DESIGN EXPLORATION",
		"HYLLA-FIRST CODE UNDERSTANDING",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"SCOPE CONFIRMATION WITH DEV",
		"VALIDATION PLAN",
	}; !slices.Equal(got, want) {
		t.Fatalf("plan child titles = %#v, want %#v", got, want)
	}

	buildPhase := findTaskByTitle(t, branchChildren, "BUILD")
	if got, want := childTitles(tasks, buildPhase.ID), []string{"PHASE PUSH AND REINGEST CONFIRMATION"}; !slices.Equal(got, want) {
		t.Fatalf("build phase child titles = %#v, want %#v", got, want)
	}
	closeoutPhase := findTaskByTitle(t, branchChildren, "CLOSEOUT")
	if got, want := childTitles(tasks, closeoutPhase.ID), []string{
		"DEV REVIEW",
		"HYLLA REFRESHED AND CURRENT TO GIT",
		"LOCAL COMMIT RECORDED",
		"ORCHESTRATOR AND DEV COLLABORATIVE TESTING",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"PUSH PR HANDOFF READINESS",
		"QA FALSIFICATION REVIEW",
		"QA PROOF REVIEW",
		"REQUIRED BUILD GATES GREEN",
	}; !slices.Equal(got, want) {
		t.Fatalf("closeout child titles = %#v, want %#v", got, want)
	}
	cleanupPhase := findTaskByTitle(t, branchChildren, "BRANCH CLEANUP")
	if got, want := childTitles(tasks, cleanupPhase.ID), []string{
		"CONFIRM CLOSEOUT TRUTHFULLY COMPLETE",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"REMOVE FINISHED BRANCH",
		"REMOVE LINKED WORKTREE",
	}; !slices.Equal(got, want) {
		t.Fatalf("cleanup child titles = %#v, want %#v", got, want)
	}
	buildTask, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  buildPhase.ID,
		ColumnID:  buildPhase.ColumnID,
		Kind:      domain.WorkKind("build-task"),
		Scope:     domain.KindAppliesToTask,
		Title:     "IMPLEMENT FRONTEND UPDATE",
	})
	if err != nil {
		t.Fatalf("CreateTask(build-task) error = %v", err)
	}

	tasks, err = svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(build-task children) error = %v", err)
	}
	buildTaskChildren := make([]domain.Task, 0)
	for _, task := range tasks {
		if task.ParentID == buildTask.ID {
			buildTaskChildren = append(buildTaskChildren, task)
		}
	}
	if want := []string{"ACCESSIBILITY CHECK", "COMMIT PUSH AND REINGEST", "QA FALSIFICATION REVIEW", "QA PROOF REVIEW", "VISUAL QA"}; !slices.Equal(childTitles(tasks, buildTask.ID), want) {
		t.Fatalf("build-task child titles = %#v, want %#v", childTitles(tasks, buildTask.ID), want)
	}
	for _, task := range buildTaskChildren {
		snapshot, ok := repo.nodeContracts[task.ID]
		if !ok {
			t.Fatalf("repo.nodeContracts missing generated child %q", task.ID)
		}
		if !snapshot.RequiredForParentDone {
			t.Fatalf("snapshot.RequiredForParentDone for %q = false, want true", task.Title)
		}
		switch task.Title {
		case "COMMIT PUSH AND REINGEST":
			if snapshot.ResponsibleActorKind != domain.TemplateActorKindBuilder {
				t.Fatalf("commit-and-reingest responsible actor = %q, want builder", snapshot.ResponsibleActorKind)
			}
		case "VISUAL QA", "ACCESSIBILITY CHECK", "QA PROOF REVIEW", "QA FALSIFICATION REVIEW":
			if snapshot.ResponsibleActorKind != domain.TemplateActorKindQA {
				t.Fatalf("%s responsible actor = %q, want qa", task.Title, snapshot.ResponsibleActorKind)
			}
		default:
			t.Fatalf("unexpected build-task child %q", task.Title)
		}
	}
}

// TestDefaultGoBuiltinTemplateLibraryGeneratesRefactorWorkflowKinds verifies the shipped default-go runtime
// generates refactor and dogfood-refactor phase/task children with the expected role contracts.
func TestDefaultGoBuiltinTemplateLibraryGeneratesRefactorWorkflowKinds(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{AutoCreateProjectColumns: false})
	seedBuiltinTemplateKinds(t, ctx, svc)

	if _, err := svc.EnsureBuiltinTemplateLibrary(ctx, EnsureBuiltinTemplateLibraryInput{
		LibraryID: "default-go",
		ActorID:   "dev-1",
		ActorName: "Dev",
		ActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v", err)
	}

	project, err := svc.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:              "Go Refactor",
		Kind:              "go-project",
		TemplateLibraryID: "default-go",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if !strings.Contains(project.Metadata.StandardsMarkdown, "does not auto-create or force that repair item today") {
		t.Fatalf("project standards missing repair-item caveat: %q", project.Metadata.StandardsMarkdown)
	}
	if !strings.Contains(project.Metadata.StandardsMarkdown, "does not auto-verify every metric field or rollup total today") {
		t.Fatalf("project standards missing metrics caveat: %q", project.Metadata.StandardsMarkdown)
	}
	columns, err := svc.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	branch, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "REFACTOR LANE",
	})
	if err != nil {
		t.Fatalf("CreateTask(branch) error = %v", err)
	}

	refactorPhase, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("refactor-phase"),
		Scope:     domain.KindAppliesToPhase,
		Title:     "REFACTOR PHASE",
	})
	if err != nil {
		t.Fatalf("CreateTask(refactor-phase) error = %v", err)
	}
	dogfoodPhase, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("dogfood-refactor-phase"),
		Scope:     domain.KindAppliesToPhase,
		Title:     "DOGFOOD REFACTOR PHASE",
	})
	if err != nil {
		t.Fatalf("CreateTask(dogfood-refactor-phase) error = %v", err)
	}

	tasks, err := svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(phase children) error = %v", err)
	}
	if got, want := childTitles(tasks, refactorPhase.ID), []string{
		"HYLLA-FIRST REFACTOR BASELINE",
		"PHASE METRICS ROLLUP",
		"PHASE PARITY VALIDATION PLAN",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"REFACTOR METRICS BASELINE AND REPORT PATH",
		"REFACTOR SUBPHASE AND SLICE TREE",
	}; !slices.Equal(got, want) {
		t.Fatalf("refactor-phase child titles = %#v, want %#v", got, want)
	}
	if got, want := childTitles(tasks, dogfoodPhase.ID), []string{
		"CONFIRM LOCAL USED VERSION UPDATED",
		"DEV VERSION VALIDATION PLAN",
		"HYLLA-FIRST REFACTOR BASELINE",
		"PHASE METRICS ROLLUP",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"REFACTOR METRICS BASELINE AND REPORT PATH",
		"REFACTOR SUBPHASE AND SLICE TREE",
	}; !slices.Equal(got, want) {
		t.Fatalf("dogfood-refactor-phase child titles = %#v, want %#v", got, want)
	}

	refactorPhasePush := findChildTaskByTitle(t, tasks, refactorPhase.ID, "PHASE PUSH AND REINGEST CONFIRMATION")
	refactorPhasePushSnapshot := mustNodeContractSnapshot(t, repo, refactorPhasePush.ID)
	if refactorPhasePushSnapshot.ResponsibleActorKind != domain.TemplateActorKindOrchestrator {
		t.Fatalf("refactor phase push responsible actor = %q, want orchestrator", refactorPhasePushSnapshot.ResponsibleActorKind)
	}
	if !slices.Contains(refactorPhasePushSnapshot.EditableByActorKinds, domain.TemplateActorKindBuilder) || !slices.Contains(refactorPhasePushSnapshot.EditableByActorKinds, domain.TemplateActorKindResearch) {
		t.Fatalf("refactor phase push editable actors = %#v, want builder+research+orchestrator", refactorPhasePushSnapshot.EditableByActorKinds)
	}
	dogfoodPhaseLocal := findChildTaskByTitle(t, tasks, dogfoodPhase.ID, "CONFIRM LOCAL USED VERSION UPDATED")
	dogfoodPhaseLocalSnapshot := mustNodeContractSnapshot(t, repo, dogfoodPhaseLocal.ID)
	if dogfoodPhaseLocalSnapshot.ResponsibleActorKind != domain.TemplateActorKindHuman {
		t.Fatalf("dogfood phase local-version responsible actor = %q, want human", dogfoodPhaseLocalSnapshot.ResponsibleActorKind)
	}
	if !slices.Equal(dogfoodPhaseLocalSnapshot.CompletableByActorKinds, []domain.TemplateActorKind{domain.TemplateActorKindHuman}) {
		t.Fatalf("dogfood phase local-version completable actors = %#v, want human only", dogfoodPhaseLocalSnapshot.CompletableByActorKinds)
	}

	refactorTask, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  refactorPhase.ID,
		ColumnID:  refactorPhase.ColumnID,
		Kind:      domain.WorkKind("refactor-task"),
		Scope:     domain.KindAppliesToTask,
		Title:     "DECOUPLE FLOW",
	})
	if err != nil {
		t.Fatalf("CreateTask(refactor-task) error = %v", err)
	}
	dogfoodTask, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  dogfoodPhase.ID,
		ColumnID:  dogfoodPhase.ColumnID,
		Kind:      domain.WorkKind("dogfood-refactor-task"),
		Scope:     domain.KindAppliesToTask,
		Title:     "DOGFOOD DECOUPLE FLOW",
	})
	if err != nil {
		t.Fatalf("CreateTask(dogfood-refactor-task) error = %v", err)
	}

	tasks, err = svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(task children) error = %v", err)
	}
	if got, want := childTitles(tasks, refactorTask.ID), []string{
		"COMMIT PUSH AND REINGEST",
		"METRICS CAPTURE AND REPORT",
		"PARITY VALIDATION IN ACTION",
		"QA FALSIFICATION REVIEW",
		"QA PROOF REVIEW",
	}; !slices.Equal(got, want) {
		t.Fatalf("refactor-task child titles = %#v, want %#v", got, want)
	}
	if got, want := childTitles(tasks, dogfoodTask.ID), []string{
		"COMMIT PUSH AND REINGEST",
		"CONFIRM LOCAL USED VERSION UPDATED",
		"METRICS CAPTURE AND REPORT",
		"QA FALSIFICATION REVIEW",
		"QA PROOF REVIEW",
		"TEST AGAINST DEV VERSION",
	}; !slices.Equal(got, want) {
		t.Fatalf("dogfood-refactor-task child titles = %#v, want %#v", got, want)
	}

	refactorCommit := findChildTaskByTitle(t, tasks, refactorTask.ID, "COMMIT PUSH AND REINGEST")
	refactorCommitSnapshot := mustNodeContractSnapshot(t, repo, refactorCommit.ID)
	if refactorCommitSnapshot.ResponsibleActorKind != domain.TemplateActorKindBuilder {
		t.Fatalf("refactor commit responsible actor = %q, want builder", refactorCommitSnapshot.ResponsibleActorKind)
	}
	if !slices.Contains(refactorCommitSnapshot.CompletableByActorKinds, domain.TemplateActorKindBuilder) || !slices.Contains(refactorCommitSnapshot.CompletableByActorKinds, domain.TemplateActorKindHuman) {
		t.Fatalf("refactor commit completable actors = %#v, want builder+human", refactorCommitSnapshot.CompletableByActorKinds)
	}
	refactorQA := findChildTaskByTitle(t, tasks, refactorTask.ID, "QA PROOF REVIEW")
	refactorQASnapshot := mustNodeContractSnapshot(t, repo, refactorQA.ID)
	if refactorQASnapshot.ResponsibleActorKind != domain.TemplateActorKindQA || refactorQA.Kind != domain.WorkKind("qa-check") {
		t.Fatalf("refactor QA child = %#v snapshot=%#v, want qa-check owned by qa", refactorQA, refactorQASnapshot)
	}
	parityValidation := findChildTaskByTitle(t, tasks, refactorTask.ID, "PARITY VALIDATION IN ACTION")
	paritySnapshot := mustNodeContractSnapshot(t, repo, parityValidation.ID)
	if paritySnapshot.ResponsibleActorKind != domain.TemplateActorKindBuilder || parityValidation.Kind != domain.WorkKind("subtask") {
		t.Fatalf("parity validation child = %#v snapshot=%#v, want subtask owned by builder", parityValidation, paritySnapshot)
	}
	refactorMetrics := findChildTaskByTitle(t, tasks, refactorTask.ID, "METRICS CAPTURE AND REPORT")
	if !strings.Contains(refactorMetrics.Description, "does not auto-verify every metric field or rollup total today") {
		t.Fatalf("refactor metrics description missing caveat: %q", refactorMetrics.Description)
	}
	refactorPhaseRollup := findChildTaskByTitle(t, tasks, refactorPhase.ID, "PHASE METRICS ROLLUP")
	if !strings.Contains(refactorPhaseRollup.Description, "does not auto-verify every field or rollup total today") {
		t.Fatalf("refactor phase rollup description missing caveat: %q", refactorPhaseRollup.Description)
	}
	dogfoodLocalTask := findChildTaskByTitle(t, tasks, dogfoodTask.ID, "CONFIRM LOCAL USED VERSION UPDATED")
	dogfoodLocalTaskSnapshot := mustNodeContractSnapshot(t, repo, dogfoodLocalTask.ID)
	if dogfoodLocalTaskSnapshot.ResponsibleActorKind != domain.TemplateActorKindHuman || dogfoodLocalTask.Kind != domain.WorkKind("subtask") {
		t.Fatalf("dogfood local-version child = %#v snapshot=%#v, want subtask owned by human", dogfoodLocalTask, dogfoodLocalTaskSnapshot)
	}
	if !slices.Equal(dogfoodLocalTaskSnapshot.CompletableByActorKinds, []domain.TemplateActorKind{domain.TemplateActorKindHuman}) {
		t.Fatalf("dogfood local-version completable actors = %#v, want human only", dogfoodLocalTaskSnapshot.CompletableByActorKinds)
	}
	dogfoodMetrics := findChildTaskByTitle(t, tasks, dogfoodTask.ID, "METRICS CAPTURE AND REPORT")
	if !strings.Contains(dogfoodMetrics.Description, "does not auto-verify every metric field or rollup total today") {
		t.Fatalf("dogfood metrics description missing caveat: %q", dogfoodMetrics.Description)
	}
}

// TestDefaultFrontendBuiltinTemplateLibraryGeneratesRefactorWorkflowKinds verifies the shipped default-frontend runtime
// generates refactor and dogfood-refactor phase/task children with the expected role contracts.
func TestDefaultFrontendBuiltinTemplateLibraryGeneratesRefactorWorkflowKinds(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 3, 10, 15, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{AutoCreateProjectColumns: false})
	seedDefaultFrontendBuiltinTemplateKinds(t, ctx, svc)

	if _, err := svc.EnsureBuiltinTemplateLibrary(ctx, EnsureBuiltinTemplateLibraryInput{
		LibraryID: "default-frontend",
		ActorID:   "dev-1",
		ActorName: "Dev",
		ActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v", err)
	}

	project, err := svc.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:              "Frontend Refactor",
		Kind:              "frontend-project",
		TemplateLibraryID: "default-frontend",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if !strings.Contains(project.Metadata.StandardsMarkdown, "does not auto-create or force that repair item today") {
		t.Fatalf("frontend project standards missing repair-item caveat: %q", project.Metadata.StandardsMarkdown)
	}
	if !strings.Contains(project.Metadata.StandardsMarkdown, "does not auto-verify every metric field or rollup total today") {
		t.Fatalf("frontend project standards missing metrics caveat: %q", project.Metadata.StandardsMarkdown)
	}
	columns, err := svc.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	branch, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "FRONTEND REFACTOR LANE",
	})
	if err != nil {
		t.Fatalf("CreateTask(branch) error = %v", err)
	}

	refactorPhase, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("refactor-phase"),
		Scope:     domain.KindAppliesToPhase,
		Title:     "FRONTEND REFACTOR PHASE",
	})
	if err != nil {
		t.Fatalf("CreateTask(refactor-phase) error = %v", err)
	}
	dogfoodPhase, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("dogfood-refactor-phase"),
		Scope:     domain.KindAppliesToPhase,
		Title:     "FRONTEND DOGFOOD REFACTOR PHASE",
	})
	if err != nil {
		t.Fatalf("CreateTask(dogfood-refactor-phase) error = %v", err)
	}

	tasks, err := svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(phase children) error = %v", err)
	}
	if got, want := childTitles(tasks, refactorPhase.ID), []string{
		"HYLLA-FIRST REFACTOR BASELINE",
		"PHASE METRICS ROLLUP",
		"PHASE PARITY VALIDATION PLAN",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"REFACTOR METRICS BASELINE AND REPORT PATH",
		"REFACTOR SUBPHASE AND SLICE TREE",
	}; !slices.Equal(got, want) {
		t.Fatalf("frontend refactor-phase child titles = %#v, want %#v", got, want)
	}
	if got, want := childTitles(tasks, dogfoodPhase.ID), []string{
		"CONFIRM LOCAL USED VERSION UPDATED",
		"DEV VERSION VALIDATION PLAN",
		"HYLLA-FIRST REFACTOR BASELINE",
		"PHASE METRICS ROLLUP",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"REFACTOR METRICS BASELINE AND REPORT PATH",
		"REFACTOR SUBPHASE AND SLICE TREE",
	}; !slices.Equal(got, want) {
		t.Fatalf("frontend dogfood-refactor-phase child titles = %#v, want %#v", got, want)
	}

	refactorTask, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  refactorPhase.ID,
		ColumnID:  refactorPhase.ColumnID,
		Kind:      domain.WorkKind("refactor-task"),
		Scope:     domain.KindAppliesToTask,
		Title:     "REDUCE UI COUPLING",
	})
	if err != nil {
		t.Fatalf("CreateTask(refactor-task) error = %v", err)
	}
	dogfoodTask, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  dogfoodPhase.ID,
		ColumnID:  dogfoodPhase.ColumnID,
		Kind:      domain.WorkKind("dogfood-refactor-task"),
		Scope:     domain.KindAppliesToTask,
		Title:     "DOGFOOD REDUCE UI COUPLING",
	})
	if err != nil {
		t.Fatalf("CreateTask(dogfood-refactor-task) error = %v", err)
	}

	tasks, err = svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(task children) error = %v", err)
	}
	if got, want := childTitles(tasks, refactorTask.ID), []string{
		"ACCESSIBILITY CHECK",
		"COMMIT PUSH AND REINGEST",
		"METRICS CAPTURE AND REPORT",
		"PARITY VALIDATION IN ACTION",
		"QA FALSIFICATION REVIEW",
		"QA PROOF REVIEW",
		"VISUAL QA",
	}; !slices.Equal(got, want) {
		t.Fatalf("frontend refactor-task child titles = %#v, want %#v", got, want)
	}
	if got, want := childTitles(tasks, dogfoodTask.ID), []string{
		"ACCESSIBILITY CHECK",
		"COMMIT PUSH AND REINGEST",
		"CONFIRM LOCAL USED VERSION UPDATED",
		"METRICS CAPTURE AND REPORT",
		"QA FALSIFICATION REVIEW",
		"QA PROOF REVIEW",
		"TEST AGAINST DEV VERSION",
		"VISUAL QA",
	}; !slices.Equal(got, want) {
		t.Fatalf("frontend dogfood-refactor-task child titles = %#v, want %#v", got, want)
	}

	visualQA := findChildTaskByTitle(t, tasks, refactorTask.ID, "VISUAL QA")
	visualQASnapshot := mustNodeContractSnapshot(t, repo, visualQA.ID)
	if visualQA.Kind != domain.WorkKind("visual-qa") || visualQASnapshot.ResponsibleActorKind != domain.TemplateActorKindQA {
		t.Fatalf("visual QA child = %#v snapshot=%#v, want visual-qa owned by qa", visualQA, visualQASnapshot)
	}
	a11yCheck := findChildTaskByTitle(t, tasks, refactorTask.ID, "ACCESSIBILITY CHECK")
	a11ySnapshot := mustNodeContractSnapshot(t, repo, a11yCheck.ID)
	if a11yCheck.Kind != domain.WorkKind("a11y-check") || a11ySnapshot.ResponsibleActorKind != domain.TemplateActorKindQA {
		t.Fatalf("a11y child = %#v snapshot=%#v, want a11y-check owned by qa", a11yCheck, a11ySnapshot)
	}
	frontendCommit := findChildTaskByTitle(t, tasks, refactorTask.ID, "COMMIT PUSH AND REINGEST")
	frontendCommitSnapshot := mustNodeContractSnapshot(t, repo, frontendCommit.ID)
	if frontendCommitSnapshot.ResponsibleActorKind != domain.TemplateActorKindBuilder {
		t.Fatalf("frontend commit responsible actor = %q, want builder", frontendCommitSnapshot.ResponsibleActorKind)
	}
	frontendRefactorMetrics := findChildTaskByTitle(t, tasks, refactorTask.ID, "METRICS CAPTURE AND REPORT")
	if !strings.Contains(frontendRefactorMetrics.Description, "does not auto-verify every metric field or rollup total today") {
		t.Fatalf("frontend refactor metrics description missing caveat: %q", frontendRefactorMetrics.Description)
	}
	frontendPhaseRollup := findChildTaskByTitle(t, tasks, refactorPhase.ID, "PHASE METRICS ROLLUP")
	if !strings.Contains(frontendPhaseRollup.Description, "does not auto-verify every field or rollup total today") {
		t.Fatalf("frontend phase rollup description missing caveat: %q", frontendPhaseRollup.Description)
	}
	frontendDogfoodLocal := findChildTaskByTitle(t, tasks, dogfoodTask.ID, "CONFIRM LOCAL USED VERSION UPDATED")
	frontendDogfoodLocalSnapshot := mustNodeContractSnapshot(t, repo, frontendDogfoodLocal.ID)
	if frontendDogfoodLocal.Kind != domain.WorkKind("subtask") || frontendDogfoodLocalSnapshot.ResponsibleActorKind != domain.TemplateActorKindHuman {
		t.Fatalf("frontend dogfood local-version child = %#v snapshot=%#v, want subtask owned by human", frontendDogfoodLocal, frontendDogfoodLocalSnapshot)
	}
	if !slices.Equal(frontendDogfoodLocalSnapshot.CompletableByActorKinds, []domain.TemplateActorKind{domain.TemplateActorKindHuman}) {
		t.Fatalf("frontend dogfood local-version completable actors = %#v, want human only", frontendDogfoodLocalSnapshot.CompletableByActorKinds)
	}
	frontendDogfoodMetrics := findChildTaskByTitle(t, tasks, dogfoodTask.ID, "METRICS CAPTURE AND REPORT")
	if !strings.Contains(frontendDogfoodMetrics.Description, "does not auto-verify every metric field or rollup total today") {
		t.Fatalf("frontend dogfood metrics description missing caveat: %q", frontendDogfoodMetrics.Description)
	}
}

// TestEnsureBuiltinTemplateLibraryCreatesExpandedDefaultGoWorkflow verifies the shipped builtin
// generates project setup, branch lifecycle phases, and build-task QA work end to end.
func TestEnsureBuiltinTemplateLibraryCreatesExpandedDefaultGoWorkflow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	now := time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		AutoCreateProjectColumns: false,
	})
	seedBuiltinTemplateKinds(t, ctx, svc)

	if _, err := svc.EnsureBuiltinTemplateLibrary(ctx, EnsureBuiltinTemplateLibraryInput{
		LibraryID: "default-go",
		ActorID:   "dev-1",
		ActorName: "Dev",
		ActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v", err)
	}

	project, err := svc.CreateProjectWithMetadata(ctx, CreateProjectInput{
		Name:              "TILLSYN",
		Kind:              "go-project",
		TemplateLibraryID: "default-go",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	binding, err := svc.GetProjectTemplateBinding(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProjectTemplateBinding() error = %v", err)
	}
	if binding.LibraryID != "default-go" {
		t.Fatalf("binding.LibraryID = %q, want default-go", binding.LibraryID)
	}

	tasks, err := svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(project) error = %v", err)
	}
	projectSetup := findTaskByTitle(t, tasks, "PROJECT SETUP")
	if projectSetup.Kind != domain.WorkKind("project-setup-phase") || projectSetup.Scope != domain.KindAppliesToPhase {
		t.Fatalf("project setup kind/scope = %q/%q, want project-setup-phase/phase", projectSetup.Kind, projectSetup.Scope)
	}
	if got, want := childTitles(tasks, projectSetup.ID), []string{
		"CREATE OR CONFIRM FIRST BRANCH LANE",
		"CREATE OR CONFIRM FIRST PLAN PHASE",
		"HYLLA INGEST MODE DECISION",
		"HYLLA INITIAL INGEST OR REFRESH",
		"HYLLA VS DB STATE REVIEW",
		"HYLLA VS GIT FRESHNESS CHECK",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"PROJECT METADATA AND STANDARDS LOCK",
		"TEMPLATE FIT REVIEW",
	}; !slices.Equal(got, want) {
		t.Fatalf("project setup child titles = %#v, want %#v", got, want)
	}

	columns, err := svc.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected a root column for template-generated work")
	}

	branch, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "MAIN WORKTREE LANE",
	})
	if err != nil {
		t.Fatalf("CreateTask(branch) error = %v", err)
	}

	tasks, err = svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(branch lifecycle) error = %v", err)
	}
	if got, want := childTitles(tasks, branch.ID), []string{"BRANCH CLEANUP", "BUILD", "CLOSEOUT", "PLAN"}; !slices.Equal(got, want) {
		t.Fatalf("branch child titles = %#v, want %#v", got, want)
	}
	planPhase := findChildTaskByTitle(t, tasks, branch.ID, "PLAN")
	if got, want := childTitles(tasks, planPhase.ID), []string{
		"BRANCH AND WORKTREE SETUP",
		"BUILD TASK TREE",
		"CLOSEOUT AND CLEANUP EXPECTATIONS",
		"CONTEXT7 AND GO DOC RESEARCH",
		"HYLLA-FIRST CODE UNDERSTANDING",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"SCOPE CONFIRMATION WITH DEV",
		"VALIDATION PLAN",
	}; !slices.Equal(got, want) {
		t.Fatalf("plan child titles = %#v, want %#v", got, want)
	}
	buildPhase := findChildTaskByTitle(t, tasks, branch.ID, "BUILD")
	if got, want := childTitles(tasks, buildPhase.ID), []string{"PHASE PUSH AND REINGEST CONFIRMATION"}; !slices.Equal(got, want) {
		t.Fatalf("build phase child titles = %#v, want %#v", got, want)
	}
	closeoutPhase := findChildTaskByTitle(t, tasks, branch.ID, "CLOSEOUT")
	if got, want := childTitles(tasks, closeoutPhase.ID), []string{
		"DEV REVIEW",
		"HYLLA REFRESHED AND CURRENT TO GIT",
		"LOCAL COMMIT RECORDED",
		"ORCHESTRATOR AND DEV COLLABORATIVE TESTING",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"PUSH PR HANDOFF READINESS",
		"QA FALSIFICATION REVIEW",
		"QA PROOF REVIEW",
		"REQUIRED MAGE GATES GREEN",
	}; !slices.Equal(got, want) {
		t.Fatalf("closeout child titles = %#v, want %#v", got, want)
	}
	cleanupPhase := findChildTaskByTitle(t, tasks, branch.ID, "BRANCH CLEANUP")
	if got, want := childTitles(tasks, cleanupPhase.ID), []string{
		"CONFIRM CLOSEOUT TRUTHFULLY COMPLETE",
		"CONFIRM STALE MCP SERVER GONE",
		"PHASE PUSH AND REINGEST CONFIRMATION",
		"REFRESH CODEX MCP LIST",
		"REMOVE FINISHED BRANCH",
		"REMOVE LANE GOPLS MCP ENTRY",
		"REMOVE LINKED WORKTREE",
	}; !slices.Equal(got, want) {
		t.Fatalf("cleanup child titles = %#v, want %#v", got, want)
	}
	buildTask, err := svc.CreateTask(ctx, CreateTaskInput{
		ProjectID: project.ID,
		ParentID:  buildPhase.ID,
		ColumnID:  columns[0].ID,
		Kind:      domain.WorkKind("build-task"),
		Scope:     domain.KindAppliesToTask,
		Title:     "IMPLEMENT SHIPPED DEFAULT-GO TEMPLATE",
	})
	if err != nil {
		t.Fatalf("CreateTask(build-task) error = %v", err)
	}

	tasks, err = svc.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(build task) error = %v", err)
	}
	if got, want := childTitles(tasks, buildTask.ID), []string{"COMMIT PUSH AND REINGEST", "QA FALSIFICATION REVIEW", "QA PROOF REVIEW"}; !slices.Equal(got, want) {
		t.Fatalf("build-task QA child titles = %#v, want %#v", got, want)
	}
}

// TestGetBuiltinTemplateLibraryStatusDetectsUpdateAvailable verifies status reports update availability when the installed library predates builtin provenance metadata.
func TestGetBuiltinTemplateLibraryStatusDetectsUpdateAvailable(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	svc := newDeterministicService(repo, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), ServiceConfig{})
	seedBuiltinTemplateKinds(t, ctx, svc)

	spec, err := defaultGoBuiltinTemplateLibrarySpec(builtinTemplateActor{
		ID:   "dev-1",
		Name: "Dev",
		Type: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("defaultGoBuiltinTemplateLibrarySpec() error = %v", err)
	}
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
				TitleTemplate:           "QA PROOF REVIEW",
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
				TitleTemplate:           "QA PROOF REVIEW UPDATE",
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
				TitleTemplate:           "QA PROOF REVIEW",
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
				TitleTemplate:           "QA PROOF REVIEW UPDATE",
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
	if updatedTask.Title != "QA PROOF REVIEW UPDATE" {
		t.Fatalf("updated task title = %q, want QA PROOF REVIEW UPDATE", updatedTask.Title)
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
				TitleTemplate:           "QA PROOF REVIEW",
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
				TitleTemplate:           "QA PROOF REVIEW UPDATE",
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
		{ID: "project-setup-phase", DisplayName: "Project Setup Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "plan-phase", DisplayName: "Plan Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-phase", DisplayName: "Build Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "closeout-phase", DisplayName: "Closeout Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "branch-cleanup-phase", DisplayName: "Branch Cleanup Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "refactor-phase", DisplayName: "Refactor Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "dogfood-refactor-phase", DisplayName: "Dogfood Refactor Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-task", DisplayName: "Build Task", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: "refactor-task", DisplayName: "Refactor Task", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: "dogfood-refactor-task", DisplayName: "Dogfood Refactor Task", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: "qa-check", DisplayName: "QA Check", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
		{ID: "commit-and-reingest", DisplayName: "Commit and Reingest", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
	} {
		if _, err := svc.UpsertKindDefinition(ctx, spec); err != nil {
			t.Fatalf("UpsertKindDefinition(%q) error = %v", spec.ID, err)
		}
	}
}

// seedDefaultFrontendBuiltinTemplateKinds installs the builtin default-frontend prerequisite kinds used by lifecycle tests.
func seedDefaultFrontendBuiltinTemplateKinds(t *testing.T, ctx context.Context, svc *Service) {
	t.Helper()
	for _, spec := range []CreateKindDefinitionInput{
		{ID: "frontend-project", DisplayName: "Frontend Project", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToProject}},
		{ID: "project-setup-phase", DisplayName: "Project Setup Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "plan-phase", DisplayName: "Plan Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-phase", DisplayName: "Build Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "closeout-phase", DisplayName: "Closeout Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "branch-cleanup-phase", DisplayName: "Branch Cleanup Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "refactor-phase", DisplayName: "Refactor Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "dogfood-refactor-phase", DisplayName: "Dogfood Refactor Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-task", DisplayName: "Build Task", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: "refactor-task", DisplayName: "Refactor Task", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: "dogfood-refactor-task", DisplayName: "Dogfood Refactor Task", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: "qa-check", DisplayName: "QA Check", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
		{ID: "visual-qa", DisplayName: "Visual QA", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
		{ID: "a11y-check", DisplayName: "Accessibility Check", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
		{ID: "design-review", DisplayName: "Design Review", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: "commit-and-reingest", DisplayName: "Commit and Reingest", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
	} {
		if _, err := svc.UpsertKindDefinition(ctx, spec); err != nil {
			t.Fatalf("UpsertKindDefinition(%q) error = %v", spec.ID, err)
		}
	}
}

// childTitles returns stable sorted child titles for one parent task id.
func childTitles(tasks []domain.Task, parentID string) []string {
	out := make([]string, 0)
	for _, task := range tasks {
		if task.ParentID != parentID {
			continue
		}
		out = append(out, task.Title)
	}
	slices.Sort(out)
	return out
}

func childRuleTitles(rules []UpsertTemplateChildRuleInput) []string {
	out := make([]string, 0, len(rules))
	for _, rule := range rules {
		out = append(out, rule.TitleTemplate)
	}
	slices.Sort(out)
	return out
}

func findNodeTemplateByKind(t *testing.T, templates []UpsertNodeTemplateInput, kind domain.KindID) UpsertNodeTemplateInput {
	t.Helper()
	for _, template := range templates {
		if template.NodeKindID == kind {
			return template
		}
	}
	t.Fatalf("expected node template for kind %q", kind)
	return UpsertNodeTemplateInput{}
}

func mustNodeContractSnapshot(t *testing.T, repo *fakeRepo, nodeID string) domain.NodeContractSnapshot {
	t.Helper()
	snapshot, ok := repo.nodeContracts[nodeID]
	if !ok {
		t.Fatalf("repo.nodeContracts missing node %q", nodeID)
	}
	return snapshot
}

// findTaskByTitle returns one task with the requested title or fails the test.
func findTaskByTitle(t *testing.T, tasks []domain.Task, title string) domain.Task {
	t.Helper()
	for _, task := range tasks {
		if task.Title == title {
			return task
		}
	}
	t.Fatalf("missing task with title %q", title)
	return domain.Task{}
}

// findChildTaskByTitle returns one child task with the requested title or fails the test.
func findChildTaskByTitle(t *testing.T, tasks []domain.Task, parentID, title string) domain.Task {
	t.Helper()
	for _, task := range tasks {
		if task.ParentID == parentID && task.Title == title {
			return task
		}
	}
	t.Fatalf("missing child task %q under parent %q", title, parentID)
	return domain.Task{}
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
