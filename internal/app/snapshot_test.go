package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestExportSnapshotIncludesExpectedData verifies behavior for the covered scenario.
func TestExportSnapshotIncludesExpectedData(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)

	p1, _ := domain.NewProject("p1", "Alpha", "", now)
	p1.Metadata = domain.ProjectMetadata{Owner: "team-a", Tags: []string{"alpha"}}
	p2, _ := domain.NewProject("p2", "Beta", "", now)
	p2.Archive(now.Add(time.Minute))
	repo.projects[p1.ID] = p1
	repo.projects[p2.ID] = p2

	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "Done", 0, 0, now)
	repo.columns[c1.ID] = c1
	repo.columns[c2.ID] = c2

	t1, _ := domain.NewTask(domain.TaskInput{ID: "t1", ProjectID: p1.ID, ColumnID: c1.ID, Position: 0, Title: "Task A", Priority: domain.PriorityLow}, now)
	t2, _ := domain.NewTask(domain.TaskInput{ID: "t2", ProjectID: p2.ID, ColumnID: c2.ID, Position: 0, Title: "Task B", Priority: domain.PriorityHigh}, now)
	t2.Archive(now.Add(2 * time.Minute))
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2

	kind, err := domain.NewKindDefinition(domain.KindDefinitionInput{
		ID:          "refactor",
		DisplayName: "Refactor",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToTask},
	}, now)
	if err != nil {
		t.Fatalf("NewKindDefinition() error = %v", err)
	}
	repo.kindDefs[kind.ID] = kind
	repo.projectAllowedKinds[p1.ID] = []domain.KindID{kind.ID}
	templateLibrary, err := domain.NewTemplateLibrary(domain.TemplateLibraryInput{
		ID:                  "global-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Global Defaults",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "tester",
		CreatedByActorName:  "Tester",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "tester",
		ApprovedByActorName: "Tester",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []domain.NodeTemplateInput{{
			ID:          "task-template",
			ScopeLevel:  domain.KindAppliesToTask,
			NodeKindID:  kind.ID,
			DisplayName: "Refactor Task",
		}},
	}, now)
	if err != nil {
		t.Fatalf("NewTemplateLibrary() error = %v", err)
	}
	repo.templateLibraries[templateLibrary.ID] = templateLibrary
	binding, err := domain.NewProjectTemplateBinding(domain.ProjectTemplateBindingInput{
		ProjectID:        p1.ID,
		LibraryID:        templateLibrary.ID,
		BoundByActorID:   "tester",
		BoundByActorName: "Tester",
		BoundByActorType: domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewProjectTemplateBinding() error = %v", err)
	}
	repo.projectBindings[p1.ID] = binding
	nodeContract, err := domain.NewNodeContractSnapshot(domain.NodeContractSnapshotInput{
		NodeID:                  t1.ID,
		ProjectID:               p1.ID,
		SourceLibraryID:         templateLibrary.ID,
		SourceNodeTemplateID:    "task-template",
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
		RequiredForParentDone:   true,
	}, now)
	if err != nil {
		t.Fatalf("NewNodeContractSnapshot() error = %v", err)
	}
	repo.nodeContracts[t1.ID] = nodeContract

	projectComment, err := domain.NewComment(domain.CommentInput{
		ID:           "comment-1",
		ProjectID:    p1.ID,
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     p1.ID,
		BodyMarkdown: "Project comment",
		ActorID:      "tester",
		ActorName:    "tester",
		ActorType:    domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewComment() error = %v", err)
	}
	commentKey := p1.ID + "|" + string(projectComment.TargetType) + "|" + projectComment.TargetID
	repo.comments[commentKey] = []domain.Comment{projectComment}

	lease, err := domain.NewCapabilityLease(domain.CapabilityLeaseInput{
		InstanceID: "lease-1",
		LeaseToken: "token-1",
		AgentName:  "orchestrator",
		ProjectID:  p1.ID,
		ScopeType:  domain.CapabilityScopeTask,
		ScopeID:    t1.ID,
		Role:       domain.CapabilityRoleOrchestrator,
		ExpiresAt:  now.Add(2 * time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("NewCapabilityLease() error = %v", err)
	}
	repo.capabilityLeases[lease.InstanceID] = lease
	handoff, err := domain.NewHandoff(domain.HandoffInput{
		ID:              "handoff-1",
		ProjectID:       p1.ID,
		ScopeType:       domain.ScopeLevelTask,
		ScopeID:         t1.ID,
		SourceRole:      "builder",
		TargetRole:      "qa",
		Status:          domain.HandoffStatusWaiting,
		Summary:         "Wait for QA",
		NextAction:      "QA reviews work",
		MissingEvidence: []string{"manual qa"},
		CreatedByActor:  "tester",
		CreatedByType:   domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewHandoff() error = %v", err)
	}
	repo.handoffs[handoff.ID] = handoff
	archivedHandoff, err := domain.NewHandoff(domain.HandoffInput{
		ID:             "handoff-2",
		ProjectID:      p2.ID,
		ScopeType:      domain.ScopeLevelTask,
		ScopeID:        t2.ID,
		SourceRole:     "builder",
		TargetRole:     "qa",
		Status:         domain.HandoffStatusWaiting,
		Summary:        "Wait for archived QA",
		CreatedByActor: "tester",
		CreatedByType:  domain.ActorTypeUser,
	}, now)
	if err != nil {
		t.Fatalf("NewHandoff(archived) error = %v", err)
	}
	repo.handoffs[archivedHandoff.ID] = archivedHandoff

	svc := NewService(repo, nil, func() time.Time { return now.Add(3 * time.Minute) }, ServiceConfig{})

	snapActive, err := svc.ExportSnapshot(context.Background(), false)
	if err != nil {
		t.Fatalf("ExportSnapshot(active) error = %v", err)
	}
	if snapActive.Version != SnapshotVersion {
		t.Fatalf("unexpected version %q", snapActive.Version)
	}
	if len(snapActive.Projects) != 1 || snapActive.Projects[0].ID != p1.ID {
		t.Fatalf("unexpected active projects %#v", snapActive.Projects)
	}
	if len(snapActive.Columns) != 1 || snapActive.Columns[0].ID != c1.ID {
		t.Fatalf("unexpected active columns %#v", snapActive.Columns)
	}
	if len(snapActive.Tasks) != 1 || snapActive.Tasks[0].ID != t1.ID {
		t.Fatalf("unexpected active tasks %#v", snapActive.Tasks)
	}
	if len(snapActive.Handoffs) != 1 || snapActive.Handoffs[0].ID != handoff.ID {
		t.Fatalf("expected only active-scope handoff in active snapshot, got %#v", snapActive.Handoffs)
	}

	snapAll, err := svc.ExportSnapshot(context.Background(), true)
	if err != nil {
		t.Fatalf("ExportSnapshot(all) error = %v", err)
	}
	if len(snapAll.Projects) != 2 || len(snapAll.Columns) != 2 || len(snapAll.Tasks) != 2 {
		t.Fatalf("unexpected all snapshot sizes p=%d c=%d t=%d", len(snapAll.Projects), len(snapAll.Columns), len(snapAll.Tasks))
	}
	if len(snapAll.KindDefinitions) != 1 {
		t.Fatalf("expected kind definition closure in snapshot, got %#v", snapAll.KindDefinitions)
	}
	if len(snapAll.ProjectAllowedKinds) != 1 || snapAll.ProjectAllowedKinds[0].ProjectID != p1.ID {
		t.Fatalf("expected project allowlist closure in snapshot, got %#v", snapAll.ProjectAllowedKinds)
	}
	if len(snapAll.TemplateLibraries) != 1 || snapAll.TemplateLibraries[0].ID != templateLibrary.ID {
		t.Fatalf("expected template library closure in snapshot, got %#v", snapAll.TemplateLibraries)
	}
	if len(snapAll.ProjectBindings) != 1 || snapAll.ProjectBindings[0].ProjectID != p1.ID {
		t.Fatalf("expected project template binding closure in snapshot, got %#v", snapAll.ProjectBindings)
	}
	if len(snapAll.NodeContracts) != 1 || snapAll.NodeContracts[0].NodeID != t1.ID {
		t.Fatalf("expected node contract closure in snapshot, got %#v", snapAll.NodeContracts)
	}
	if len(snapAll.Comments) != 1 || snapAll.Comments[0].ID != "comment-1" {
		t.Fatalf("expected comment closure in snapshot, got %#v", snapAll.Comments)
	}
	if snapAll.Comments[0].Summary != "Project comment" {
		t.Fatalf("expected comment summary in snapshot export, got %#v", snapAll.Comments[0])
	}
	if len(snapAll.CapabilityLeases) != 1 || snapAll.CapabilityLeases[0].InstanceID != "lease-1" {
		t.Fatalf("expected capability lease closure in snapshot, got %#v", snapAll.CapabilityLeases)
	}
	if len(snapAll.Handoffs) != 2 {
		t.Fatalf("expected archived-scope handoff in full snapshot, got %#v", snapAll.Handoffs)
	}
	if snapAll.Handoffs[0].ID != "handoff-1" || snapAll.Handoffs[1].ID != "handoff-2" {
		t.Fatalf("expected deterministic lexical handoff order in snapshot, got %#v", snapAll.Handoffs)
	}
	foundMeta := false
	for _, sp := range snapAll.Projects {
		if sp.ID == p1.ID && sp.Metadata.Owner == "team-a" {
			foundMeta = true
			break
		}
	}
	if !foundMeta {
		t.Fatalf("expected metadata to round-trip in export, got %#v", snapAll.Projects)
	}
}

// TestImportSnapshotCreatesAndUpdates verifies behavior for the covered scenario.
func TestImportSnapshotCreatesAndUpdates(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)

	existingProject, _ := domain.NewProject("p1", "Old Name", "", now)
	existingCol, _ := domain.NewColumn("c1", existingProject.ID, "Old Col", 0, 0, now)
	existingTask, _ := domain.NewTask(domain.TaskInput{ID: "t1", ProjectID: existingProject.ID, ColumnID: existingCol.ID, Position: 0, Title: "Old Task", Priority: domain.PriorityLow}, now)

	repo.projects[existingProject.ID] = existingProject
	repo.columns[existingCol.ID] = existingCol
	repo.tasks[existingTask.ID] = existingTask

	svc := NewService(repo, nil, func() time.Time { return now }, ServiceConfig{})
	due := now.Add(48 * time.Hour)
	snap := Snapshot{
		Version: SnapshotVersion,
		Projects: []SnapshotProject{
			{ID: "p1", Name: "New Name", Description: "updated", Slug: "new-name", Metadata: domain.ProjectMetadata{Owner: "owner-1", Tags: []string{"a"}}, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "p2", Name: "Project Two", Description: "new", Slug: "project-two", CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
		},
		Columns: []SnapshotColumn{
			{ID: "c1", ProjectID: "p1", Name: "Doing", Position: 1, WIPLimit: 2, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "c2", ProjectID: "p2", Name: "To Do", Position: 0, WIPLimit: 0, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
		},
		Tasks: []SnapshotTask{
			{ID: "t1", ProjectID: "p1", ColumnID: "c1", Position: 2, Title: "Updated Task", Description: "details", Priority: domain.PriorityHigh, DueAt: &due, Labels: []string{"a", "b"}, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "t2", ProjectID: "p2", ColumnID: "c2", Position: 0, Title: "New Task", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "phase-1", ProjectID: "p2", ColumnID: "c2", Position: 1, Title: "Phase", Priority: domain.PriorityMedium, Kind: domain.WorkKindPhase, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
		},
		KindDefinitions: []SnapshotKindDefinition{
			{
				ID:          "refactor",
				DisplayName: "Refactor",
				AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToTask},
				CreatedAt:   now,
				UpdatedAt:   now.Add(time.Minute),
			},
		},
		ProjectAllowedKinds: []SnapshotProjectAllowedKinds{
			{ProjectID: "p1", KindIDs: []domain.KindID{"refactor"}},
		},
		TemplateLibraries: []domain.TemplateLibrary{
			func() domain.TemplateLibrary {
				library, err := domain.NewTemplateLibrary(domain.TemplateLibraryInput{
					ID:                  "global-defaults",
					Scope:               domain.TemplateLibraryScopeGlobal,
					Name:                "Global Defaults",
					Status:              domain.TemplateLibraryStatusApproved,
					CreatedByActorID:    "importer",
					CreatedByActorName:  "Importer",
					CreatedByActorType:  domain.ActorTypeUser,
					ApprovedByActorID:   "importer",
					ApprovedByActorName: "Importer",
					ApprovedByActorType: domain.ActorTypeUser,
					NodeTemplates: []domain.NodeTemplateInput{{
						ID:          "task-template",
						ScopeLevel:  domain.KindAppliesToTask,
						NodeKindID:  domain.KindID("refactor"),
						DisplayName: "Refactor Task",
					}},
				}, now)
				if err != nil {
					t.Fatalf("NewTemplateLibrary(import) error = %v", err)
				}
				return library
			}(),
		},
		ProjectBindings: []domain.ProjectTemplateBinding{
			func() domain.ProjectTemplateBinding {
				binding, err := domain.NewProjectTemplateBinding(domain.ProjectTemplateBindingInput{
					ProjectID:        "p1",
					LibraryID:        "global-defaults",
					BoundByActorID:   "importer",
					BoundByActorName: "Importer",
					BoundByActorType: domain.ActorTypeUser,
				}, now)
				if err != nil {
					t.Fatalf("NewProjectTemplateBinding(import) error = %v", err)
				}
				return binding
			}(),
		},
		NodeContracts: []domain.NodeContractSnapshot{
			func() domain.NodeContractSnapshot {
				snapshot, err := domain.NewNodeContractSnapshot(domain.NodeContractSnapshotInput{
					NodeID:                  "t1",
					ProjectID:               "p1",
					SourceLibraryID:         "global-defaults",
					SourceNodeTemplateID:    "task-template",
					ResponsibleActorKind:    domain.TemplateActorKindQA,
					EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
					CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA, domain.TemplateActorKindHuman},
					RequiredForParentDone:   true,
				}, now)
				if err != nil {
					t.Fatalf("NewNodeContractSnapshot(import) error = %v", err)
				}
				return snapshot
			}(),
		},
		Comments: []SnapshotComment{
			{
				ID:           "comment-1",
				ProjectID:    "p1",
				TargetType:   domain.CommentTargetTypeProject,
				TargetID:     "p1",
				BodyMarkdown: "Imported project comment",
				ActorID:      "importer",
				ActorName:    "importer",
				ActorType:    domain.ActorTypeUser,
				CreatedAt:    now,
				UpdatedAt:    now.Add(time.Minute),
			},
		},
		CapabilityLeases: []SnapshotCapabilityLease{
			{
				InstanceID:  "lease-1",
				LeaseToken:  "token-1",
				AgentName:   "orchestrator",
				ProjectID:   "p1",
				ScopeType:   domain.CapabilityScopeTask,
				ScopeID:     "t1",
				Role:        domain.CapabilityRoleOrchestrator,
				IssuedAt:    now,
				ExpiresAt:   now.Add(24 * time.Hour),
				HeartbeatAt: now.Add(2 * time.Minute),
			},
		},
		Handoffs: []SnapshotHandoff{
			{
				ID:             "handoff-1",
				ProjectID:      "p1",
				ScopeType:      domain.ScopeLevelTask,
				ScopeID:        "t1",
				SourceRole:     "builder",
				TargetRole:     "qa",
				Status:         domain.HandoffStatusWaiting,
				Summary:        "Imported handoff",
				NextAction:     "Wait for QA",
				CreatedByActor: "importer",
				CreatedByType:  domain.ActorTypeUser,
				CreatedAt:      now,
				UpdatedByActor: "importer",
				UpdatedByType:  domain.ActorTypeUser,
				UpdatedAt:      now.Add(time.Minute),
			},
		},
	}

	if err := svc.ImportSnapshot(context.Background(), snap); err != nil {
		t.Fatalf("ImportSnapshot() error = %v", err)
	}

	if got := repo.projects["p1"]; got.Name != "New Name" || got.Description != "updated" {
		t.Fatalf("unexpected updated project %#v", got)
	}
	if got := repo.projects["p1"]; got.Metadata.Owner != "owner-1" {
		t.Fatalf("expected metadata owner updated, got %#v", got.Metadata)
	}
	if _, ok := repo.projects["p2"]; !ok {
		t.Fatal("expected new project p2")
	}
	if got := repo.columns["c1"]; got.Name != "Doing" || got.Position != 1 {
		t.Fatalf("unexpected updated column %#v", got)
	}
	if _, ok := repo.columns["c2"]; !ok {
		t.Fatal("expected new column c2")
	}
	if got := repo.tasks["t1"]; got.Title != "Updated Task" || got.Priority != domain.PriorityHigh {
		t.Fatalf("unexpected updated task %#v", got)
	}
	if _, ok := repo.tasks["t2"]; !ok {
		t.Fatal("expected new task t2")
	}
	if got := repo.tasks["phase-1"]; got.Kind != domain.WorkKindPhase || got.Scope != domain.KindAppliesToPhase {
		t.Fatalf("expected imported phase task to default to phase scope, got %#v", got)
	}
	if _, ok := repo.kindDefs[domain.KindID("refactor")]; !ok {
		t.Fatal("expected imported kind definition refactor")
	}
	allowed := repo.projectAllowedKinds["p1"]
	if len(allowed) != 1 || allowed[0] != domain.KindID("refactor") {
		t.Fatalf("expected imported project allowlist for p1, got %#v", allowed)
	}
	if _, ok := repo.templateLibraries["global-defaults"]; !ok {
		t.Fatal("expected imported template library global-defaults")
	}
	if binding, ok := repo.projectBindings["p1"]; !ok || binding.LibraryID != "global-defaults" {
		t.Fatalf("expected imported project binding for p1, got %#v", repo.projectBindings["p1"])
	}
	if nodeContract, ok := repo.nodeContracts["t1"]; !ok || nodeContract.SourceLibraryID != "global-defaults" {
		t.Fatalf("expected imported node contract for t1, got %#v", repo.nodeContracts["t1"])
	}
	commentKey := "p1|project|p1"
	if len(repo.comments[commentKey]) != 1 || repo.comments[commentKey][0].ID != "comment-1" {
		t.Fatalf("expected imported project comment closure, got %#v", repo.comments[commentKey])
	}
	if repo.comments[commentKey][0].Summary != "Imported project comment" {
		t.Fatalf("expected imported comment summary fallback from body markdown, got %#v", repo.comments[commentKey][0])
	}
	if _, ok := repo.capabilityLeases["lease-1"]; !ok {
		t.Fatal("expected imported capability lease lease-1")
	}
	if _, ok := repo.handoffs["handoff-1"]; !ok {
		t.Fatal("expected imported handoff handoff-1")
	}
}

// TestImportSnapshotValidateErrors verifies behavior for the covered scenario.
func TestImportSnapshotValidateErrors(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, nil, time.Now, ServiceConfig{})

	badVersion := Snapshot{Version: "tillsyn.snapshot.v999"}
	if err := svc.ImportSnapshot(context.Background(), badVersion); err == nil {
		t.Fatal("expected version validation error")
	}
	missingVersion := Snapshot{}
	if err := svc.ImportSnapshot(context.Background(), missingVersion); err == nil {
		t.Fatal("expected missing version validation error")
	}

	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	badRefs := Snapshot{
		Version:  SnapshotVersion,
		Projects: []SnapshotProject{{ID: "p1", Name: "A", Slug: "a", CreatedAt: now, UpdatedAt: now}},
		Columns:  []SnapshotColumn{{ID: "c1", ProjectID: "missing", Name: "To Do", Position: 0, CreatedAt: now, UpdatedAt: now}},
	}
	if err := svc.ImportSnapshot(context.Background(), badRefs); err == nil {
		t.Fatal("expected reference validation error")
	}

	invalidPhaseParent := Snapshot{
		Version: SnapshotVersion,
		Projects: []SnapshotProject{
			{ID: "p1", Name: "A", Slug: "a", CreatedAt: now, UpdatedAt: now},
		},
		Columns: []SnapshotColumn{
			{ID: "c1", ProjectID: "p1", Name: "To Do", Position: 0, CreatedAt: now, UpdatedAt: now},
		},
		Tasks: []SnapshotTask{
			{ID: "t1", ProjectID: "p1", ColumnID: "c1", Position: 0, Title: "Task", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now},
			{ID: "p1", ProjectID: "p1", ParentID: "t1", Kind: domain.WorkKindPhase, Scope: domain.KindAppliesToPhase, ColumnID: "c1", Position: 1, Title: "Phase", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now},
		},
	}
	if err := svc.ImportSnapshot(context.Background(), invalidPhaseParent); err == nil || !strings.Contains(err.Error(), "invalid for phase parent scope") {
		t.Fatalf("expected invalid phase parent error, got %v", err)
	}

	validNestedPhase := Snapshot{
		Version: SnapshotVersion,
		Projects: []SnapshotProject{
			{ID: "p2", Name: "B", Slug: "b", CreatedAt: now, UpdatedAt: now},
		},
		Columns: []SnapshotColumn{
			{ID: "c2", ProjectID: "p2", Name: "To Do", Position: 0, CreatedAt: now, UpdatedAt: now},
		},
		Tasks: []SnapshotTask{
			{ID: "branch-1", ProjectID: "p2", Kind: domain.WorkKind("branch"), Scope: domain.KindAppliesToBranch, ColumnID: "c2", Position: 0, Title: "Branch", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now},
			{ID: "phase-1", ProjectID: "p2", ParentID: "branch-1", Kind: domain.WorkKindPhase, Scope: domain.KindAppliesToPhase, ColumnID: "c2", Position: 1, Title: "Phase", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now},
			{ID: "phase-2", ProjectID: "p2", ParentID: "phase-1", Kind: domain.WorkKindPhase, Scope: domain.KindAppliesToPhase, ColumnID: "c2", Position: 2, Title: "Nested Phase", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now},
		},
	}
	if err := svc.ImportSnapshot(context.Background(), validNestedPhase); err != nil {
		t.Fatalf("expected valid nested phase lineage to import, got %v", err)
	}

	orphanHandoff := Snapshot{
		Version: SnapshotVersion,
		Projects: []SnapshotProject{
			{ID: "p3", Name: "C", Slug: "c", CreatedAt: now, UpdatedAt: now},
		},
		Columns: []SnapshotColumn{
			{ID: "c3", ProjectID: "p3", Name: "To Do", Position: 0, CreatedAt: now, UpdatedAt: now},
		},
		Tasks: []SnapshotTask{
			{ID: "t3", ProjectID: "p3", ColumnID: "c3", Position: 0, Title: "Task", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now},
		},
		Handoffs: []SnapshotHandoff{
			{
				ID:             "handoff-missing",
				ProjectID:      "p3",
				ScopeType:      domain.ScopeLevelTask,
				ScopeID:        "missing-task",
				SourceRole:     "builder",
				TargetRole:     "qa",
				Status:         domain.HandoffStatusWaiting,
				Summary:        "Broken handoff",
				CreatedByActor: "importer",
				CreatedByType:  domain.ActorTypeUser,
				CreatedAt:      now,
				UpdatedByActor: "importer",
				UpdatedByType:  domain.ActorTypeUser,
				UpdatedAt:      now,
			},
		},
	}
	if err := svc.ImportSnapshot(context.Background(), orphanHandoff); err == nil || !strings.Contains(err.Error(), "unknown source scope") {
		t.Fatalf("expected orphan handoff validation error, got %v", err)
	}

	badTemplateRefs := Snapshot{
		Version: SnapshotVersion,
		Projects: []SnapshotProject{
			{ID: "p4", Name: "D", Slug: "d", CreatedAt: now, UpdatedAt: now},
		},
		Columns: []SnapshotColumn{
			{ID: "c4", ProjectID: "p4", Name: "To Do", Position: 0, CreatedAt: now, UpdatedAt: now},
		},
		Tasks: []SnapshotTask{
			{ID: "t4", ProjectID: "p4", ColumnID: "c4", Position: 0, Title: "Task", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now},
		},
		TemplateLibraries: []domain.TemplateLibrary{
			{
				ID:        "broken-library",
				Scope:     domain.TemplateLibraryScopeGlobal,
				Name:      "Broken",
				Status:    domain.TemplateLibraryStatusApproved,
				CreatedAt: now,
				UpdatedAt: now,
				NodeTemplates: []domain.NodeTemplate{{
					ID:          "broken-template",
					LibraryID:   "broken-library",
					ScopeLevel:  domain.KindAppliesToTask,
					NodeKindID:  domain.KindID("missing-kind"),
					DisplayName: "Broken Template",
				}},
			},
		},
	}
	if err := svc.ImportSnapshot(context.Background(), badTemplateRefs); err == nil || !strings.Contains(err.Error(), "unknown node_kind_id") {
		t.Fatalf("expected template reference validation error, got %v", err)
	}
}

// failingSnapshotRepo represents failing snapshot repo data used by this package.
type failingSnapshotRepo struct {
	*fakeRepo
	err error
}

// ListProjects lists projects.
func (f failingSnapshotRepo) ListProjects(context.Context, bool) ([]domain.Project, error) {
	return nil, f.err
}

// TestExportSnapshotPropagatesError verifies behavior for the covered scenario.
func TestExportSnapshotPropagatesError(t *testing.T) {
	expected := errors.New("boom")
	svc := NewService(failingSnapshotRepo{fakeRepo: newFakeRepo(), err: expected}, nil, time.Now, ServiceConfig{})
	_, err := svc.ExportSnapshot(context.Background(), false)
	if !errors.Is(err, expected) {
		t.Fatalf("expected error %v, got %v", expected, err)
	}
}

// TestSnapshotValidateAcceptsFailedState verifies that the failed lifecycle state is accepted by snapshot validation.
func TestSnapshotValidateAcceptsFailedState(t *testing.T) {
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	snap := Snapshot{
		Version:  SnapshotVersion,
		Projects: []SnapshotProject{{ID: "p1", Name: "A", Slug: "a", CreatedAt: now, UpdatedAt: now}},
		Columns:  []SnapshotColumn{{ID: "c1", ProjectID: "p1", Name: "Failed", Position: 3, CreatedAt: now, UpdatedAt: now}},
		Tasks: []SnapshotTask{
			{ID: "t1", ProjectID: "p1", ColumnID: "c1", Position: 0, Title: "Failed task", Priority: domain.PriorityMedium, LifecycleState: domain.StateFailed, CreatedAt: now, UpdatedAt: now},
		},
	}
	if err := snap.Validate(); err != nil {
		t.Fatalf("Validate() should accept failed lifecycle state, got error = %v", err)
	}
}

// TestSnapshotValidateRejectsInvalidState verifies the error message includes failed in the valid states list.
func TestSnapshotValidateRejectsInvalidState(t *testing.T) {
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	snap := Snapshot{
		Version:  SnapshotVersion,
		Projects: []SnapshotProject{{ID: "p1", Name: "A", Slug: "a", CreatedAt: now, UpdatedAt: now}},
		Columns:  []SnapshotColumn{{ID: "c1", ProjectID: "p1", Name: "To Do", Position: 0, CreatedAt: now, UpdatedAt: now}},
		Tasks: []SnapshotTask{
			{ID: "t1", ProjectID: "p1", ColumnID: "c1", Position: 0, Title: "Bad state", Priority: domain.PriorityMedium, LifecycleState: "invalid", CreatedAt: now, UpdatedAt: now},
		},
	}
	err := snap.Validate()
	if err == nil {
		t.Fatal("Validate() should reject invalid lifecycle state")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Fatalf("error message should include 'failed' in valid states list, got %q", err.Error())
	}
}
