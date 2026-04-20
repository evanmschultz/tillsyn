package common

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// commonLifecycleFixture stores one real adapter stack for integration-style common package tests.
type commonLifecycleFixture struct {
	adapter *AppServiceAdapter
	repo    *sqlite.Repository
	svc     *app.Service
	now     time.Time
}

// newCommonLifecycleFixture constructs a real adapter + sqlite + app service stack for wrapper coverage.
func newCommonLifecycleFixture(t *testing.T) commonLifecycleFixture {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	now := time.Date(2026, 3, 20, 18, 0, 0, 0, time.UTC)
	nextID := 0
	svc := app.NewService(repo, func() string {
		nextID++
		return fmt.Sprintf("common-id-%d", nextID)
	}, func() time.Time { return now }, app.ServiceConfig{})
	return commonLifecycleFixture{
		adapter: NewAppServiceAdapter(svc, nil),
		repo:    repo,
		svc:     svc,
		now:     now,
	}
}

// containsStep reports whether one guidance step contains every required fragment.
func containsStep(steps []string, fragments ...string) bool {
	for _, step := range steps {
		matches := true
		for _, fragment := range fragments {
			if !strings.Contains(step, fragment) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

// TestAppServiceAdapterProjectActionItemCommentLifecycle verifies common adapter wrappers over project/actionItem/comment flows.
func TestAppServiceAdapterProjectActionItemCommentLifecycle(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{
		Name:        "Inbox",
		Description: "Project description",
		Metadata:    domain.ProjectMetadata{Color: "amber"},
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	project, err = fixture.adapter.UpdateProject(ctx, UpdateProjectRequest{
		ProjectID:   project.ID,
		Name:        "Inbox Updated",
		Description: "Updated description",
		Metadata:    domain.ProjectMetadata{Color: "gold"},
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if project.Name != "Inbox Updated" {
		t.Fatalf("UpdateProject() name = %q, want Inbox Updated", project.Name)
	}
	projects, err := fixture.adapter.ListProjects(ctx, false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("ListProjects() = %#v, want project %q", projects, project.ID)
	}

	todo, err := fixture.svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn(todo) error = %v", err)
	}
	done, err := fixture.svc.CreateColumn(ctx, project.ID, "Done", 1, 0)
	if err != nil {
		t.Fatalf("CreateColumn(done) error = %v", err)
	}

	actionItem, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:   project.ID,
		ColumnID:    todo.ID,
		Title:       "Parent actionItem",
		Description: "Start here",
		Priority:    "high",
		Labels:      []string{"docs"},
		DueAt:       fixture.now.Add(2 * time.Hour).Format(time.RFC3339),
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(parent) error = %v", err)
	}
	gotActionItem, err := fixture.adapter.GetActionItem(ctx, actionItem.ID)
	if err != nil {
		t.Fatalf("GetActionItem() error = %v", err)
	}
	if gotActionItem.ID != actionItem.ID {
		t.Fatalf("GetActionItem() id = %q, want %q", gotActionItem.ID, actionItem.ID)
	}
	actionItem, err = fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: actionItem.ID,
		Title:        "Parent actionItem updated",
		Description:  "Updated body",
		Priority:     "medium",
		Labels:       []string{"docs", "review"},
		Actor:        actor,
	})
	if err != nil {
		t.Fatalf("UpdateActionItem() error = %v", err)
	}
	actionItem, err = fixture.adapter.MoveActionItemState(ctx, MoveActionItemStateRequest{
		ActionItemID: actionItem.ID,
		State:        "done",
		Actor:        actor,
	})
	if err != nil {
		t.Fatalf("MoveActionItemState() error = %v", err)
	}
	if actionItem.ColumnID != done.ID {
		t.Fatalf("MoveActionItemState() column_id = %q, want %q", actionItem.ColumnID, done.ID)
	}
	if actionItem.LifecycleState != domain.StateDone {
		t.Fatalf("MoveActionItemState() lifecycle_state = %q, want %q", actionItem.LifecycleState, domain.StateDone)
	}

	child, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID: project.ID,
		ParentID:  actionItem.ID,
		Kind:      string(domain.KindSubtask),
		ColumnID:  done.ID,
		Title:     "Child actionItem",
		Priority:  "medium",
		Actor:     actor,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(child) error = %v", err)
	}
	child, err = fixture.adapter.ReparentActionItem(ctx, ReparentActionItemRequest{
		ActionItemID: child.ID,
		ParentID:     actionItem.ID,
		Actor:        actor,
	})
	if err != nil {
		t.Fatalf("ReparentActionItem() error = %v", err)
	}
	if child.ParentID != actionItem.ID {
		t.Fatalf("ReparentActionItem() parent_id = %q, want %q", child.ParentID, actionItem.ID)
	}

	tasks, err := fixture.adapter.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("ListActionItems() len = %d, want 2", len(tasks))
	}
	children, err := fixture.adapter.ListChildActionItems(ctx, project.ID, actionItem.ID, false)
	if err != nil {
		t.Fatalf("ListChildActionItems() error = %v", err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("ListChildActionItems() = %#v, want child %q", children, child.ID)
	}
	result, err := fixture.adapter.SearchActionItems(ctx, SearchActionItemsRequest{
		ProjectID: project.ID,
		Query:     "updated",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchActionItems() error = %v", err)
	}
	if len(result.Matches) != 1 || result.Matches[0].ActionItem.ID != actionItem.ID {
		t.Fatalf("SearchActionItems() = %#v, want updated parent actionItem", result)
	}

	comment, err := fixture.adapter.CreateComment(ctx, CreateCommentRequest{
		ProjectID:    project.ID,
		TargetType:   string(domain.CommentTargetTypeProject),
		TargetID:     project.ID,
		Summary:      "Project summary",
		BodyMarkdown: "Detailed comment body",
		Actor:        actor,
	})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	if comment.Summary != "Project summary" {
		t.Fatalf("CreateComment() summary = %q, want Project summary", comment.Summary)
	}
	comments, err := fixture.adapter.ListCommentsByTarget(ctx, ListCommentsByTargetRequest{
		ProjectID:  project.ID,
		TargetType: string(domain.CommentTargetTypeProject),
		TargetID:   project.ID,
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget() error = %v", err)
	}
	if len(comments) != 1 || comments[0].ID != comment.ID {
		t.Fatalf("ListCommentsByTarget() = %#v, want comment %q", comments, comment.ID)
	}
	guide, err := fixture.adapter.GetBootstrapGuide(ctx)
	if err != nil {
		t.Fatalf("GetBootstrapGuide() error = %v", err)
	}
	if guide.WhatTillsynIs == "" || len(guide.Recommended) == 0 {
		t.Fatalf("GetBootstrapGuide() = %#v, want populated guidance", guide)
	}
	if guide.Summary == "" || !strings.Contains(guide.Summary, "approved global agent session") || !strings.Contains(guide.Summary, "auth request") {
		t.Fatalf("GetBootstrapGuide() summary = %q, want auth-aware bootstrap guidance", guide.Summary)
	}
	if !strings.Contains(guide.Summary, "worklogs in Tillsyn itself") {
		t.Fatalf("GetBootstrapGuide() summary = %q, want Tillsyn-only coordination guidance", guide.Summary)
	}
	if !strings.Contains(guide.WhatTillsynIs, "AGENTS.md") || !strings.Contains(guide.WhatTillsynIs, "CLAUDE.md") {
		t.Fatalf("GetBootstrapGuide() what_tillsyn_is = %q, want AGENTS.md/CLAUDE.md policy guidance", guide.WhatTillsynIs)
	}
	if len(guide.NextSteps) < 4 {
		t.Fatalf("GetBootstrapGuide() next_steps = %#v, want at least 4 operational steps", guide.NextSteps)
	}
	if !containsStep(guide.NextSteps, "approved", "create a project") {
		t.Fatalf("GetBootstrapGuide() next_steps = %#v, want approved-session project guidance", guide.NextSteps)
	}
	if !containsStep(guide.NextSteps, "till.auth_request(operation=create)", "auto-generate", "resume_token") {
		t.Fatalf("GetBootstrapGuide() next_steps = %#v, want auth-request auto-generation guidance", guide.NextSteps)
	}
	if !containsStep(guide.NextSteps, "till.auth_request(operation=claim)", "till.project(operation=create)") {
		t.Fatalf("GetBootstrapGuide() next_steps = %#v, want claim -> project create guidance", guide.NextSteps)
	}
	if !containsStep(guide.NextSteps, "till.template(operation=list)", "till.project(operation=bind_template)") {
		t.Fatalf("GetBootstrapGuide() next_steps = %#v, want template-library binding guidance", guide.NextSteps)
	}
	if !containsStep(guide.NextSteps, "Keep active coordination inside Tillsyn itself") {
		t.Fatalf("GetBootstrapGuide() next_steps = %#v, want Tillsyn-only coordination step", guide.NextSteps)
	}
	if !containsStep(guide.NextSteps, "AGENTS.md", "CLAUDE.md") {
		t.Fatalf("GetBootstrapGuide() next_steps = %#v, want AGENTS.md/CLAUDE.md alignment step", guide.NextSteps)
	}
	if !containsStep(guide.NextSteps, "till.capture_state") {
		t.Fatalf("GetBootstrapGuide() next_steps = %#v, want capture-state guidance", guide.NextSteps)
	}
	for _, tool := range []string{
		"till.auth_request",
		"till.project",
		"till.template",
		"till.action_item",
		"till.capture_state",
	} {
		if !slices.Contains(guide.Recommended, tool) {
			t.Fatalf("GetBootstrapGuide() recommended = %#v, want %q", guide.Recommended, tool)
		}
	}

	attentionItem, err := fixture.adapter.RaiseAttentionItem(ctx, RaiseAttentionItemRequest{
		ProjectID:          project.ID,
		ScopeType:          ScopeTypeProject,
		ScopeID:            project.ID,
		Kind:               string(domain.AttentionKindConsensusRequired),
		Summary:            "needs approval",
		BodyMarkdown:       "requires user action",
		RequiresUserAction: true,
		Actor:              actor,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem() error = %v", err)
	}
	attentionItems, err := fixture.adapter.ListAttentionItems(ctx, ListAttentionItemsRequest{
		ProjectID: project.ID,
		ScopeType: ScopeTypeProject,
		ScopeID:   project.ID,
		State:     AttentionStateOpen,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems() error = %v", err)
	}
	if len(attentionItems) != 1 || attentionItems[0].ID != attentionItem.ID {
		t.Fatalf("ListAttentionItems() = %#v, want raised attention item", attentionItems)
	}
	capture, err := fixture.adapter.CaptureState(ctx, CaptureStateRequest{
		ProjectID: project.ID,
		View:      "full",
	})
	if err != nil {
		t.Fatalf("CaptureState() error = %v", err)
	}
	if capture.GoalOverview.ProjectID != project.ID || capture.CommentOverview.RecentCount != 1 || capture.AttentionOverview.OpenCount != 1 {
		t.Fatalf("CaptureState() = %#v, want project/comment/attention summary", capture)
	}
	resolvedAttention, err := fixture.adapter.ResolveAttentionItem(ctx, ResolveAttentionItemRequest{
		ID:    attentionItem.ID,
		Actor: actor,
	})
	if err != nil {
		t.Fatalf("ResolveAttentionItem() error = %v", err)
	}
	if resolvedAttention.State != AttentionStateResolved {
		t.Fatalf("ResolveAttentionItem() state = %q, want resolved", resolvedAttention.State)
	}

	changeEvents, err := fixture.adapter.ListProjectChangeEvents(ctx, project.ID, 10)
	if err != nil {
		t.Fatalf("ListProjectChangeEvents() error = %v", err)
	}
	if len(changeEvents) == 0 {
		t.Fatal("ListProjectChangeEvents() = 0, want recorded changes")
	}
	rollup, err := fixture.adapter.GetProjectDependencyRollup(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProjectDependencyRollup() error = %v", err)
	}
	if rollup.TotalItems == 0 {
		t.Fatalf("GetProjectDependencyRollup() = %#v, want populated rollup", rollup)
	}

	if err := fixture.adapter.DeleteActionItem(ctx, DeleteActionItemRequest{ActionItemID: actionItem.ID, Mode: "archive", Actor: actor}); err != nil {
		t.Fatalf("DeleteActionItem(archive) error = %v", err)
	}
	restored, err := fixture.adapter.RestoreActionItem(ctx, RestoreActionItemRequest{ActionItemID: actionItem.ID, Actor: actor})
	if err != nil {
		t.Fatalf("RestoreActionItem() error = %v", err)
	}
	if restored.ArchivedAt != nil {
		t.Fatalf("RestoreActionItem() archived_at = %#v, want nil", restored.ArchivedAt)
	}
}

// TestAppServiceAdapterBuiltinTemplateLifecycle verifies builtin template status and ensure wrappers over the real app service stack.
func TestAppServiceAdapterBuiltinTemplateLifecycle(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	status, err := fixture.adapter.GetBuiltinTemplateLibraryStatus(ctx, "default-go")
	if err != nil {
		t.Fatalf("GetBuiltinTemplateLibraryStatus() error = %v", err)
	}
	if status.State != domain.BuiltinTemplateLibraryStateMissing {
		t.Fatalf("GetBuiltinTemplateLibraryStatus() state = %q, want missing", status.State)
	}
	for _, spec := range []app.CreateKindDefinitionInput{
		{ID: "go-project", DisplayName: "Go Project", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToProject}},
		{ID: "project-setup-phase", DisplayName: "Project Setup Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "plan-phase", DisplayName: "Plan Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-phase", DisplayName: "Build Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "closeout-phase", DisplayName: "Closeout Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "branch-cleanup-phase", DisplayName: "Branch Cleanup Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "refactor-phase", DisplayName: "Refactor Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "dogfood-refactor-phase", DisplayName: "Dogfood Refactor Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-actionItem", DisplayName: "Build ActionItem", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToActionItem}},
		{ID: "refactor-actionItem", DisplayName: "Refactor ActionItem", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToActionItem}},
		{ID: "dogfood-refactor-actionItem", DisplayName: "Dogfood Refactor ActionItem", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToActionItem}},
		{ID: "qa-check", DisplayName: "QA Check", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
		{ID: "commit-and-reingest", DisplayName: "Commit and Reingest", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
	} {
		if _, err := fixture.svc.UpsertKindDefinition(ctx, spec); err != nil {
			t.Fatalf("UpsertKindDefinition(%q) error = %v", spec.ID, err)
		}
	}
	authCtx := app.WithAuthenticatedCaller(ctx, domain.AuthenticatedCaller{
		PrincipalID:   "dev-1",
		PrincipalName: "Dev One",
		PrincipalType: domain.ActorTypeUser,
	})
	result, err := fixture.adapter.EnsureBuiltinTemplateLibrary(authCtx, EnsureBuiltinTemplateLibraryRequest{
		LibraryID: "default-go",
	})
	if err != nil {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("EnsureBuiltinTemplateLibrary() changed = false, want true")
	}
	if result.Status.State != domain.BuiltinTemplateLibraryStateCurrent {
		t.Fatalf("EnsureBuiltinTemplateLibrary() state = %q, want current", result.Status.State)
	}
	if result.Library.ID != "default-go" || !result.Library.BuiltinManaged {
		t.Fatalf("EnsureBuiltinTemplateLibrary() library = %#v, want builtin default-go", result.Library)
	}
}

// TestAppServiceAdapterBuiltinTemplateLifecycleDefaultFrontend verifies builtin frontend template status and ensure wrappers over the real app service stack.
func TestAppServiceAdapterBuiltinTemplateLifecycleDefaultFrontend(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	status, err := fixture.adapter.GetBuiltinTemplateLibraryStatus(ctx, "default-frontend")
	if err != nil {
		t.Fatalf("GetBuiltinTemplateLibraryStatus() error = %v", err)
	}
	if status.State != domain.BuiltinTemplateLibraryStateMissing {
		t.Fatalf("GetBuiltinTemplateLibraryStatus() state = %q, want missing", status.State)
	}
	for _, spec := range []app.CreateKindDefinitionInput{
		{ID: "frontend-project", DisplayName: "Frontend Project", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToProject}},
		{ID: "project-setup-phase", DisplayName: "Project Setup Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "plan-phase", DisplayName: "Plan Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-phase", DisplayName: "Build Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "closeout-phase", DisplayName: "Closeout Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "branch-cleanup-phase", DisplayName: "Branch Cleanup Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "refactor-phase", DisplayName: "Refactor Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "dogfood-refactor-phase", DisplayName: "Dogfood Refactor Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase}},
		{ID: "build-actionItem", DisplayName: "Build ActionItem", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToActionItem}},
		{ID: "refactor-actionItem", DisplayName: "Refactor ActionItem", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToActionItem}},
		{ID: "dogfood-refactor-actionItem", DisplayName: "Dogfood Refactor ActionItem", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToActionItem}},
		{ID: "qa-check", DisplayName: "QA Check", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
		{ID: "visual-qa", DisplayName: "Visual QA", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
		{ID: "a11y-check", DisplayName: "Accessibility Check", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
		{ID: "design-review", DisplayName: "Design Review", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToActionItem}},
		{ID: "commit-and-reingest", DisplayName: "Commit and Reingest", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}},
	} {
		if _, err := fixture.svc.UpsertKindDefinition(ctx, spec); err != nil {
			t.Fatalf("UpsertKindDefinition(%q) error = %v", spec.ID, err)
		}
	}
	authCtx := app.WithAuthenticatedCaller(ctx, domain.AuthenticatedCaller{
		PrincipalID:   "dev-1",
		PrincipalName: "Dev One",
		PrincipalType: domain.ActorTypeUser,
	})
	result, err := fixture.adapter.EnsureBuiltinTemplateLibrary(authCtx, EnsureBuiltinTemplateLibraryRequest{
		LibraryID: "default-frontend",
	})
	if err != nil {
		t.Fatalf("EnsureBuiltinTemplateLibrary() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("EnsureBuiltinTemplateLibrary() changed = false, want true")
	}
	if result.Status.State != domain.BuiltinTemplateLibraryStateCurrent {
		t.Fatalf("EnsureBuiltinTemplateLibrary() state = %q, want current", result.Status.State)
	}
	if result.Library.ID != "default-frontend" || !result.Library.BuiltinManaged {
		t.Fatalf("EnsureBuiltinTemplateLibrary() library = %#v, want builtin default-frontend", result.Library)
	}
}

// TestAppServiceAdapterProjectTemplateReapplyPreview verifies the common adapter surfaces project reapply preview data from the real app stack.
func TestAppServiceAdapterProjectTemplateReapplyPreview(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	project, err := fixture.svc.CreateProject(ctx, "Template Preview", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := fixture.svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if _, err := fixture.svc.UpsertTemplateLibrary(ctx, app.UpsertTemplateLibraryInput{
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
		NodeTemplates: []app.UpsertNodeTemplateInput{{
			ID:         "actionItem-template",
			ScopeLevel: domain.KindAppliesToActionItem,
			NodeKindID: domain.KindID(domain.KindActionItem),
			ChildRules: []app.UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.KindSubtask),
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
	if _, err := fixture.svc.BindProjectTemplateLibrary(ctx, app.BindProjectTemplateLibraryInput{
		ProjectID:        project.ID,
		LibraryID:        "go-defaults",
		BoundByActorID:   "dev-1",
		BoundByActorName: "Dev",
		BoundByActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("BindProjectTemplateLibrary() error = %v", err)
	}
	if _, err := fixture.svc.CreateActionItem(ctx, app.CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.KindActionItem,
		Scope:     domain.KindAppliesToActionItem,
		Title:     "Implement preview",
		Priority:  domain.PriorityMedium,
	}); err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if _, err := fixture.svc.UpsertTemplateLibrary(ctx, app.UpsertTemplateLibraryInput{
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
		NodeTemplates: []app.UpsertNodeTemplateInput{{
			ID:         "actionItem-template",
			ScopeLevel: domain.KindAppliesToActionItem,
			NodeKindID: domain.KindID(domain.KindActionItem),
			ChildRules: []app.UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.KindSubtask),
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

	preview, err := fixture.adapter.GetProjectTemplateReapplyPreview(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProjectTemplateReapplyPreview() error = %v", err)
	}
	if preview.DriftStatus != domain.ProjectTemplateBindingDriftUpdateAvailable {
		t.Fatalf("GetProjectTemplateReapplyPreview() drift = %q, want update_available", preview.DriftStatus)
	}
	if preview.EligibleMigrationCount != 1 || len(preview.ChildRuleChanges) != 1 {
		t.Fatalf("GetProjectTemplateReapplyPreview() = %#v, want one eligible candidate and one rule change", preview)
	}
}

// TestAppServiceAdapterApproveProjectTemplateMigrations verifies the common adapter applies eligible existing-node migrations.
func TestAppServiceAdapterApproveProjectTemplateMigrations(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	project, err := fixture.svc.CreateProject(ctx, "Template Approval", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := fixture.svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if _, err := fixture.svc.UpsertTemplateLibrary(ctx, app.UpsertTemplateLibraryInput{
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
		NodeTemplates: []app.UpsertNodeTemplateInput{{
			ID:         "actionItem-template",
			ScopeLevel: domain.KindAppliesToActionItem,
			NodeKindID: domain.KindID(domain.KindActionItem),
			ChildRules: []app.UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.KindSubtask),
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
	if _, err := fixture.svc.BindProjectTemplateLibrary(ctx, app.BindProjectTemplateLibraryInput{
		ProjectID:        project.ID,
		LibraryID:        "go-defaults",
		BoundByActorID:   "dev-1",
		BoundByActorName: "Dev",
		BoundByActorType: domain.ActorTypeUser,
	}); err != nil {
		t.Fatalf("BindProjectTemplateLibrary() error = %v", err)
	}
	parent, err := fixture.svc.CreateActionItem(ctx, app.CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.KindActionItem,
		Scope:     domain.KindAppliesToActionItem,
		Title:     "Implement preview",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	tasks, err := fixture.svc.ListActionItems(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	var generated domain.ActionItem
	for _, actionItem := range tasks {
		if actionItem.ParentID == parent.ID {
			generated = actionItem
			break
		}
	}
	if generated.ID == "" {
		t.Fatal("expected generated QA child actionItem")
	}
	if _, err := fixture.svc.UpsertTemplateLibrary(ctx, app.UpsertTemplateLibraryInput{
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
		NodeTemplates: []app.UpsertNodeTemplateInput{{
			ID:         "actionItem-template",
			ScopeLevel: domain.KindAppliesToActionItem,
			NodeKindID: domain.KindID(domain.KindActionItem),
			ChildRules: []app.UpsertTemplateChildRuleInput{{
				ID:                      "qa-check",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToSubtask,
				ChildKindID:             domain.KindID(domain.KindSubtask),
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

	result, err := fixture.adapter.ApproveProjectTemplateMigrations(ctx, ApproveProjectTemplateMigrationsRequest{
		ProjectID:     project.ID,
		ActionItemIDs: []string{generated.ID},
		Actor: ActorLeaseTuple{
			ActorID:   "dev-2",
			ActorName: "Dev Two",
			ActorType: string(domain.ActorTypeUser),
		},
	})
	if err != nil {
		t.Fatalf("ApproveProjectTemplateMigrations() error = %v", err)
	}
	if result.AppliedCount != 1 {
		t.Fatalf("ApproveProjectTemplateMigrations() = %#v, want one applied migration", result)
	}
	updatedActionItem, err := fixture.svc.GetActionItem(ctx, generated.ID)
	if err != nil {
		t.Fatalf("GetActionItem() error = %v", err)
	}
	if updatedActionItem.Title != "QA PROOF REVIEW UPDATE" {
		t.Fatalf("updated actionItem title = %q, want QA PROOF REVIEW UPDATE", updatedActionItem.Title)
	}
	snapshot, err := fixture.svc.GetNodeContractSnapshot(ctx, generated.ID)
	if err != nil {
		t.Fatalf("GetNodeContractSnapshot() error = %v", err)
	}
	if !slices.Equal(snapshot.EditableByActorKinds, []domain.TemplateActorKind{domain.TemplateActorKindOrchestrator, domain.TemplateActorKindQA}) {
		t.Fatalf("snapshot.EditableByActorKinds = %#v, want orchestrator+qa", snapshot.EditableByActorKinds)
	}
}

// TestAppServiceAdapterGetEmbeddingsStatusValidatesInputs verifies MCP-facing embeddings inventory rejects bad filters and hidden archived scope.
func TestAppServiceAdapterGetEmbeddingsStatusValidatesInputs(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{
		Name:  "Embeddings",
		Actor: actor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	status, err := fixture.adapter.GetEmbeddingsStatus(ctx, EmbeddingsStatusRequest{
		ProjectID: project.ID,
	})
	if err != nil {
		t.Fatalf("GetEmbeddingsStatus() error = %v", err)
	}
	if status.RuntimeOperational {
		t.Fatal("RuntimeOperational = true, want false when embeddings runtime is not fully wired in the fixture")
	}
	if _, err := fixture.adapter.GetEmbeddingsStatus(ctx, EmbeddingsStatusRequest{
		ProjectID: project.ID,
		Statuses:  []string{"pendng"},
	}); err == nil || !strings.Contains(err.Error(), "unsupported embeddings status") {
		t.Fatalf("GetEmbeddingsStatus(invalid status) error = %v, want unsupported status guidance", err)
	}
	if _, err := fixture.svc.ArchiveProject(ctx, project.ID); err != nil {
		t.Fatalf("ArchiveProject() error = %v", err)
	}
	if _, err := fixture.adapter.GetEmbeddingsStatus(ctx, EmbeddingsStatusRequest{
		ProjectID: project.ID,
	}); err == nil || !strings.Contains(err.Error(), "include_archived") {
		t.Fatalf("GetEmbeddingsStatus(archived hidden) error = %v, want archived scope guidance", err)
	}
}

// TestAppServiceAdapterEmbeddingsStatusAndSearchExposeMixedSubjectFamilies verifies the adapter surfaces mixed lifecycle families and search metadata from a real sqlite-backed service.
func TestAppServiceAdapterEmbeddingsStatusAndSearchExposeMixedSubjectFamilies(t *testing.T) {
	t.Parallel()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn-embeddings.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	now := time.Date(2026, 3, 20, 18, 0, 0, 0, time.UTC)
	nextID := 0
	svc := app.NewService(repo, func() string {
		nextID++
		return fmt.Sprintf("embeddings-id-%d", nextID)
	}, func() time.Time { return now }, app.ServiceConfig{
		EmbeddingRuntime: app.EmbeddingRuntimeConfig{
			Enabled:        true,
			Provider:       "deterministic",
			Model:          "hash-bow-v1",
			Dimensions:     3,
			ModelSignature: app.BuildEmbeddingModelSignature("deterministic", "hash-bow-v1", "", 3),
			MaxAttempts:    5,
		},
	})
	adapter := NewAppServiceAdapter(svc, nil)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	project, err := adapter.CreateProject(ctx, CreateProjectRequest{
		Name:        "Embeddings",
		Description: "Project document content for embeddings inventory",
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, err := adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Title:       "Searchable actionItem",
		Description: "work item embeddings content",
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	if _, err := adapter.UpdateProject(ctx, UpdateProjectRequest{
		ProjectID:   project.ID,
		Name:        "Embeddings",
		Description: "Updated project document content for embeddings inventory",
		Actor:       actor,
	}); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if _, err := adapter.CreateComment(ctx, CreateCommentRequest{
		ProjectID:    project.ID,
		TargetType:   string(domain.CommentTargetTypeProject),
		TargetID:     project.ID,
		Summary:      "Project thread",
		BodyMarkdown: "project thread context content",
		Actor:        actor,
	}); err != nil {
		t.Fatalf("CreateComment(project) error = %v", err)
	}
	if _, err := adapter.CreateComment(ctx, CreateCommentRequest{
		ProjectID:    project.ID,
		TargetType:   string(domain.CommentTargetTypeActionItem),
		TargetID:     actionItem.ID,
		Summary:      "ActionItem thread",
		BodyMarkdown: "actionItem thread context content",
		Actor:        actor,
	}); err != nil {
		t.Fatalf("CreateComment(actionItem) error = %v", err)
	}

	status, err := adapter.GetEmbeddingsStatus(ctx, EmbeddingsStatusRequest{
		ProjectID: project.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("GetEmbeddingsStatus() error = %v", err)
	}
	if !slices.Contains(status.ProjectIDs, project.ID) {
		t.Fatalf("ProjectIDs = %#v, want %q", status.ProjectIDs, project.ID)
	}
	types := map[string]int{}
	for _, row := range status.Rows {
		types[row.SubjectType]++
	}
	for _, want := range []string{
		string(app.EmbeddingSubjectTypeProjectDocument),
		string(app.EmbeddingSubjectTypeThreadContext),
		string(app.EmbeddingSubjectTypeWorkItem),
	} {
		if types[want] == 0 {
			t.Fatalf("status rows = %#v, want subject type %q", status.Rows, want)
		}
	}
	if status.Summary.ReadyCount+status.Summary.PendingCount+status.Summary.RunningCount+status.Summary.FailedCount+status.Summary.StaleCount == 0 {
		t.Fatalf("status summary = %#v, want non-zero lifecycle counts", status.Summary)
	}

	search, err := adapter.SearchActionItems(ctx, SearchActionItemsRequest{
		ProjectID: project.ID,
		Query:     "searchable actionItem",
		Mode:      string(app.SearchModeSemantic),
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchActionItems() error = %v", err)
	}
	if len(search.Matches) == 0 {
		t.Fatal("SearchActionItems() returned no matches, want searchable actionItem metadata")
	}
	match := search.Matches[0]
	if match.ActionItem.ID != actionItem.ID {
		t.Fatalf("SearchActionItems() match action_item_id = %q, want %q", match.ActionItem.ID, actionItem.ID)
	}
	if match.EmbeddingSubjectType == "" || match.EmbeddingSubjectID == "" {
		t.Fatalf("SearchActionItems() embedding metadata = %#v, want subject type/id", match)
	}
	if match.EmbeddingStatus == "" {
		t.Fatalf("SearchActionItems() embedding status = %#v, want lifecycle state", match)
	}
}

// TestAppServiceAdapterKindAndAllowlistLifecycle verifies kind catalog wrappers and allowlist updates.
func TestAppServiceAdapterKindAndAllowlistLifecycle(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{Name: "Inbox"})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	kind, err := fixture.adapter.UpsertKindDefinition(ctx, UpsertKindDefinitionRequest{
		ID:                  "review",
		DisplayName:         "Review",
		DescriptionMarkdown: "Review item",
		AppliesTo:           []string{"actionItem"},
	})
	if err != nil {
		t.Fatalf("UpsertKindDefinition() error = %v", err)
	}
	if kind.ID != "review" {
		t.Fatalf("UpsertKindDefinition() kind id = %q, want review", kind.ID)
	}
	kinds, err := fixture.adapter.ListKindDefinitions(ctx, false)
	if err != nil {
		t.Fatalf("ListKindDefinitions() error = %v", err)
	}
	if len(kinds) == 0 {
		t.Fatal("ListKindDefinitions() = 0, want populated catalog")
	}
	if err := fixture.adapter.SetProjectAllowedKinds(ctx, SetProjectAllowedKindsRequest{
		ProjectID: project.ID,
		KindIDs:   []string{string(kind.ID)},
	}); err != nil {
		t.Fatalf("SetProjectAllowedKinds() error = %v", err)
	}
	allowed, err := fixture.adapter.ListProjectAllowedKinds(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds() error = %v", err)
	}
	if len(allowed) != 1 || allowed[0] != string(kind.ID) {
		t.Fatalf("ListProjectAllowedKinds() = %#v, want review", allowed)
	}
}

// TestAppServiceAdapterAttentionAndLeaseLifecycle verifies attention wrappers and capability lease lifecycle wrappers.
func TestAppServiceAdapterAttentionAndLeaseLifecycle(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{
		Name:  "Ops",
		Actor: actor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	item, err := fixture.adapter.RaiseAttentionItem(ctx, RaiseAttentionItemRequest{
		ProjectID:          project.ID,
		ScopeType:          ScopeTypeProject,
		ScopeID:            project.ID,
		Kind:               string(domain.AttentionKindConsensusRequired),
		Summary:            "Needs approval",
		BodyMarkdown:       "requires user action",
		RequiresUserAction: true,
		Actor:              actor,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem() error = %v", err)
	}
	if item.ID == "" || item.State != string(domain.AttentionStateOpen) {
		t.Fatalf("RaiseAttentionItem() = %#v, want open attention item", item)
	}

	items, err := fixture.adapter.ListAttentionItems(ctx, ListAttentionItemsRequest{
		ProjectID: project.ID,
		ScopeType: ScopeTypeProject,
		ScopeID:   project.ID,
		State:     AttentionStateOpen,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != item.ID {
		t.Fatalf("ListAttentionItems() = %#v, want raised attention item", items)
	}

	resolved, err := fixture.adapter.ResolveAttentionItem(ctx, ResolveAttentionItemRequest{
		ID:    item.ID,
		Actor: actor,
	})
	if err != nil {
		t.Fatalf("ResolveAttentionItem() error = %v", err)
	}
	if resolved.State != string(domain.AttentionStateResolved) {
		t.Fatalf("ResolveAttentionItem() state = %q, want resolved", resolved.State)
	}

	lease, err := fixture.adapter.IssueCapabilityLease(ctx, IssueCapabilityLeaseRequest{
		ProjectID:           project.ID,
		ScopeType:           string(domain.CapabilityScopeProject),
		ScopeID:             project.ID,
		Role:                string(domain.CapabilityRoleWorker),
		AgentName:           "agent-1",
		AgentInstanceID:     "agent-1-instance",
		RequestedTTLSeconds: 1800,
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}
	if lease.InstanceID == "" || lease.LeaseToken == "" {
		t.Fatalf("IssueCapabilityLease() = %#v, want issued lease", lease)
	}

	heartbeat, err := fixture.adapter.HeartbeatCapabilityLease(ctx, HeartbeatCapabilityLeaseRequest{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	if err != nil {
		t.Fatalf("HeartbeatCapabilityLease() error = %v", err)
	}
	if heartbeat.InstanceID != lease.InstanceID {
		t.Fatalf("HeartbeatCapabilityLease() instance = %q, want %q", heartbeat.InstanceID, lease.InstanceID)
	}

	renewed, err := fixture.adapter.RenewCapabilityLease(ctx, RenewCapabilityLeaseRequest{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
		TTLSeconds:      3600,
	})
	if err != nil {
		t.Fatalf("RenewCapabilityLease() error = %v", err)
	}
	if !renewed.ExpiresAt.After(lease.ExpiresAt) {
		t.Fatalf("RenewCapabilityLease() expiry = %s, want after %s", renewed.ExpiresAt, lease.ExpiresAt)
	}

	revoked, err := fixture.adapter.RevokeCapabilityLease(ctx, RevokeCapabilityLeaseRequest{
		AgentInstanceID: lease.InstanceID,
		Reason:          "done",
	})
	if err != nil {
		t.Fatalf("RevokeCapabilityLease() error = %v", err)
	}
	if revoked.RevokedAt == nil || revoked.RevokedReason != "done" {
		t.Fatalf("RevokeCapabilityLease() = %#v, want revoked lease", revoked)
	}

	second, err := fixture.adapter.IssueCapabilityLease(ctx, IssueCapabilityLeaseRequest{
		ProjectID:           project.ID,
		ScopeType:           string(domain.CapabilityScopeProject),
		ScopeID:             project.ID,
		Role:                string(domain.CapabilityRoleWorker),
		AgentName:           "agent-2",
		AgentInstanceID:     "agent-2-instance",
		RequestedTTLSeconds: 1800,
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(second) error = %v", err)
	}
	if err := fixture.adapter.RevokeAllCapabilityLeases(ctx, RevokeAllCapabilityLeasesRequest{
		ProjectID: project.ID,
		ScopeType: string(domain.CapabilityScopeProject),
		ScopeID:   project.ID,
		Reason:    "scope shutdown",
	}); err != nil {
		t.Fatalf("RevokeAllCapabilityLeases() error = %v", err)
	}
	loaded, err := fixture.repo.GetCapabilityLease(ctx, second.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease() error = %v", err)
	}
	if loaded.RevokedAt == nil || loaded.RevokedReason != "scope shutdown" {
		t.Fatalf("GetCapabilityLease() = %#v, want revoked-by-scope lease", loaded)
	}
}

// TestAppServiceAdapterAttentionAndCaptureLifecycle verifies attention wrappers and capture-state mapping over a real service stack.
func TestAppServiceAdapterAttentionAndCaptureLifecycle(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{
		Name:  "Inbox",
		Actor: actor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := fixture.svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Title:       "Blocked actionItem",
		Description: "Needs review",
		Priority:    "medium",
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	comment, err := fixture.adapter.CreateComment(ctx, CreateCommentRequest{
		ProjectID:    project.ID,
		TargetType:   string(domain.CommentTargetTypeActionItem),
		TargetID:     actionItem.ID,
		Summary:      "Review summary",
		BodyMarkdown: "## Follow up\nImportant approval needed",
		Actor:        actor,
	})
	if err != nil {
		t.Fatalf("CreateComment(actionItem) error = %v", err)
	}
	if comment.Summary != "Review summary" {
		t.Fatalf("CreateComment(actionItem) summary = %q, want Review summary", comment.Summary)
	}

	raised, err := fixture.adapter.RaiseAttentionItem(ctx, RaiseAttentionItemRequest{
		ProjectID:          project.ID,
		ScopeType:          ScopeTypeActionItem,
		ScopeID:            actionItem.ID,
		Kind:               string(domain.AttentionKindConsensusRequired),
		Summary:            "ActionItem needs user decision",
		BodyMarkdown:       "Please review the request",
		RequiresUserAction: true,
		Actor:              actor,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem() error = %v", err)
	}
	items, err := fixture.adapter.ListAttentionItems(ctx, ListAttentionItemsRequest{
		ProjectID: project.ID,
		ScopeType: ScopeTypeActionItem,
		ScopeID:   actionItem.ID,
		State:     AttentionStateOpen,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != raised.ID {
		t.Fatalf("ListAttentionItems() = %#v, want raised item %q", items, raised.ID)
	}

	capture, err := fixture.adapter.CaptureState(ctx, CaptureStateRequest{
		ProjectID: project.ID,
		ScopeType: ScopeTypeActionItem,
		ScopeID:   actionItem.ID,
		View:      "full",
	})
	if err != nil {
		t.Fatalf("CaptureState() error = %v", err)
	}
	if capture.GoalOverview.ProjectID != project.ID || capture.RequestedScopeType != ScopeTypeActionItem {
		t.Fatalf("CaptureState() = %#v, want project/actionItem scope", capture)
	}
	if capture.CommentOverview.RecentCount != 1 || capture.CommentOverview.ImportantCount != 1 {
		t.Fatalf("CaptureState() comment_overview = %#v, want one important actionItem comment", capture.CommentOverview)
	}
	if capture.AttentionOverview.OpenCount != 1 || capture.AttentionOverview.RequiresUserAction != 1 {
		t.Fatalf("CaptureState() attention_overview = %#v, want open actionable item", capture.AttentionOverview)
	}

	resolved, err := fixture.adapter.ResolveAttentionItem(ctx, ResolveAttentionItemRequest{
		ID:     raised.ID,
		Reason: "handled",
		Actor:  actor,
	})
	if err != nil {
		t.Fatalf("ResolveAttentionItem() error = %v", err)
	}
	if resolved.ResolvedAt == nil {
		t.Fatalf("ResolveAttentionItem() = %#v, want resolved timestamp", resolved)
	}
}

// TestAppServiceAdapterCapabilityLeaseLifecycle verifies lease wrappers round-trip app service state.
func TestAppServiceAdapterCapabilityLeaseLifecycle(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{Name: "Lease Project"})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	lease, err := fixture.adapter.IssueCapabilityLease(ctx, IssueCapabilityLeaseRequest{
		ProjectID:       project.ID,
		ScopeType:       string(domain.CapabilityScopeProject),
		ScopeID:         project.ID,
		Role:            string(domain.CapabilityRoleWorker),
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}
	if lease.InstanceID == "" || lease.LeaseToken == "" {
		t.Fatalf("IssueCapabilityLease() = %#v, want persisted lease values", lease)
	}

	hearted, err := fixture.adapter.HeartbeatCapabilityLease(ctx, HeartbeatCapabilityLeaseRequest{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	if err != nil {
		t.Fatalf("HeartbeatCapabilityLease() error = %v", err)
	}
	if hearted.InstanceID != lease.InstanceID {
		t.Fatalf("HeartbeatCapabilityLease() instance_id = %q, want %q", hearted.InstanceID, lease.InstanceID)
	}

	renewed, err := fixture.adapter.RenewCapabilityLease(ctx, RenewCapabilityLeaseRequest{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
		TTLSeconds:      172800,
	})
	if err != nil {
		t.Fatalf("RenewCapabilityLease() error = %v", err)
	}
	if want := fixture.now.Add(48 * time.Hour); !renewed.ExpiresAt.Equal(want) {
		t.Fatalf("RenewCapabilityLease() expires_at = %v, want %v", renewed.ExpiresAt, want)
	}

	revoked, err := fixture.adapter.RevokeCapabilityLease(ctx, RevokeCapabilityLeaseRequest{
		AgentInstanceID: lease.InstanceID,
		Reason:          "done",
	})
	if err != nil {
		t.Fatalf("RevokeCapabilityLease() error = %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("RevokeCapabilityLease() revoked_at = nil, want timestamp")
	}
	if err := fixture.adapter.RevokeAllCapabilityLeases(ctx, RevokeAllCapabilityLeasesRequest{
		ProjectID: project.ID,
		ScopeType: string(domain.CapabilityScopeProject),
		ScopeID:   project.ID,
		Reason:    "cleanup",
	}); err != nil {
		t.Fatalf("RevokeAllCapabilityLeases() error = %v", err)
	}
}

// TestNormalizeActionItemStateInputAcceptsFailed verifies that normalizeActionItemStateInput accepts "failed" as a valid move state.
func TestNormalizeActionItemStateInputAcceptsFailed(t *testing.T) {
	t.Parallel()
	state, err := normalizeActionItemStateInput("failed")
	if err != nil {
		t.Fatalf("normalizeActionItemStateInput(failed) error = %v", err)
	}
	if state != domain.StateFailed {
		t.Fatalf("normalizeActionItemStateInput(failed) = %q, want %q", state, domain.StateFailed)
	}
}

// TestActionItemLifecycleStateForColumnNameFailed verifies that actionItemLifecycleStateForColumnName maps "Failed" to StateFailed.
func TestActionItemLifecycleStateForColumnNameFailed(t *testing.T) {
	t.Parallel()
	state := actionItemLifecycleStateForColumnName("Failed")
	if state != domain.StateFailed {
		t.Fatalf("actionItemLifecycleStateForColumnName(Failed) = %q, want %q", state, domain.StateFailed)
	}
}

// TestMoveActionItemStateToFailed verifies that the adapter MoveActionItemState handles the failed state end-to-end.
func TestMoveActionItemStateToFailed(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	actor := ActorLeaseTuple{
		ActorID:   "user-1",
		ActorName: "User One",
		ActorType: string(domain.ActorTypeUser),
	}

	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{
		Name:        "FailedTest",
		Description: "Test project",
		Actor:       actor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	todo, err := fixture.svc.CreateColumn(ctx, project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn(To Do) error = %v", err)
	}
	failed, err := fixture.svc.CreateColumn(ctx, project.ID, "Failed", 3, 0)
	if err != nil {
		t.Fatalf("CreateColumn(Failed) error = %v", err)
	}

	actionItem, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID: project.ID,
		ColumnID:  todo.ID,
		Title:     "ActionItem to fail",
		Priority:  "medium",
		Actor:     actor,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	actionItem, err = fixture.adapter.MoveActionItemState(ctx, MoveActionItemStateRequest{
		ActionItemID: actionItem.ID,
		State:        "failed",
		Actor:        actor,
	})
	if err != nil {
		t.Fatalf("MoveActionItemState(failed) error = %v", err)
	}
	if actionItem.LifecycleState != domain.StateFailed {
		t.Fatalf("MoveActionItemState(failed) lifecycle_state = %q, want %q", actionItem.LifecycleState, domain.StateFailed)
	}
	if actionItem.ColumnID != failed.ID {
		t.Fatalf("MoveActionItemState(failed) column_id = %q, want %q", actionItem.ColumnID, failed.ID)
	}
}
