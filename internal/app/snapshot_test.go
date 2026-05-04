package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
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

	t1, _ := domain.NewActionItemForTest(domain.ActionItemInput{ID: "t1", ProjectID: p1.ID, ColumnID: c1.ID, Position: 0, Title: "ActionItem A", Priority: domain.PriorityLow, Kind: domain.KindPlan}, now)
	t2, _ := domain.NewActionItemForTest(domain.ActionItemInput{ID: "t2", ProjectID: p2.ID, ColumnID: c2.ID, Position: 0, Title: "ActionItem B", Priority: domain.PriorityHigh, Kind: domain.KindPlan}, now)
	t2.Archive(now.Add(2 * time.Minute))
	repo.tasks[t1.ID] = t1
	repo.tasks[t2.ID] = t2

	kind, err := domain.NewKindDefinition(domain.KindDefinitionInput{
		ID:          domain.KindID(domain.KindRefinement),
		DisplayName: "Refinement",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToRefinement},
	}, now)
	if err != nil {
		t.Fatalf("NewKindDefinition() error = %v", err)
	}
	repo.kindDefs[kind.ID] = kind
	repo.projectAllowedKinds[p1.ID] = []domain.KindID{kind.ID}

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
		ScopeType:  domain.CapabilityScopeActionItem,
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
		ScopeType:       domain.ScopeLevelActionItem,
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
		ScopeType:      domain.ScopeLevelActionItem,
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
	if len(snapActive.ActionItems) != 1 || snapActive.ActionItems[0].ID != t1.ID {
		t.Fatalf("unexpected active tasks %#v", snapActive.ActionItems)
	}
	if len(snapActive.Handoffs) != 1 || snapActive.Handoffs[0].ID != handoff.ID {
		t.Fatalf("expected only active-scope handoff in active snapshot, got %#v", snapActive.Handoffs)
	}

	snapAll, err := svc.ExportSnapshot(context.Background(), true)
	if err != nil {
		t.Fatalf("ExportSnapshot(all) error = %v", err)
	}
	if len(snapAll.Projects) != 2 || len(snapAll.Columns) != 2 || len(snapAll.ActionItems) != 2 {
		t.Fatalf("unexpected all snapshot sizes p=%d c=%d t=%d", len(snapAll.Projects), len(snapAll.Columns), len(snapAll.ActionItems))
	}
	// Post-Drop-1.75 the fake repo seeds all 12 kinds from the closed enum;
	// this test upserts a refinement kind in addition (duplicate ID; no extra
	// row), so the exported closure contains the 12 built-in definitions.
	if len(snapAll.KindDefinitions) != 12 {
		t.Fatalf("expected kind definition closure in snapshot, got %d definitions: %#v", len(snapAll.KindDefinitions), snapAll.KindDefinitions)
	}
	if len(snapAll.ProjectAllowedKinds) != 1 || snapAll.ProjectAllowedKinds[0].ProjectID != p1.ID {
		t.Fatalf("expected project allowlist closure in snapshot, got %#v", snapAll.ProjectAllowedKinds)
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
	existingActionItem, _ := domain.NewActionItemForTest(domain.ActionItemInput{ID: "t1", ProjectID: existingProject.ID, ColumnID: existingCol.ID, Position: 0, Title: "Old ActionItem", Priority: domain.PriorityLow, Kind: domain.KindPlan}, now)

	repo.projects[existingProject.ID] = existingProject
	repo.columns[existingCol.ID] = existingCol
	repo.tasks[existingActionItem.ID] = existingActionItem

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
		ActionItems: []SnapshotActionItem{
			{ID: "t1", ProjectID: "p1", ColumnID: "c1", Position: 2, Title: "Updated ActionItem", Description: "details", Priority: domain.PriorityHigh, DueAt: &due, Labels: []string{"a", "b"}, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "t2", ProjectID: "p2", ColumnID: "c2", Position: 0, Title: "New ActionItem", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
			{ID: "phase-1", ProjectID: "p2", ColumnID: "c2", Position: 1, Title: "Discussion", Priority: domain.PriorityMedium, Kind: domain.KindDiscussion, CreatedAt: now, UpdatedAt: now.Add(time.Minute)},
		},
		KindDefinitions: []SnapshotKindDefinition{
			{
				ID:          domain.KindID(domain.KindRefinement),
				DisplayName: "Refinement",
				AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToRefinement},
				CreatedAt:   now,
				UpdatedAt:   now.Add(time.Minute),
			},
		},
		ProjectAllowedKinds: []SnapshotProjectAllowedKinds{
			{ProjectID: "p1", KindIDs: []domain.KindID{domain.KindID(domain.KindRefinement)}},
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
				ScopeType:   domain.CapabilityScopeActionItem,
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
				ScopeType:      domain.ScopeLevelActionItem,
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
	if got := repo.tasks["t1"]; got.Title != "Updated ActionItem" || got.Priority != domain.PriorityHigh {
		t.Fatalf("unexpected updated actionItem %#v", got)
	}
	if _, ok := repo.tasks["t2"]; !ok {
		t.Fatal("expected new actionItem t2")
	}
	if got := repo.tasks["phase-1"]; got.Kind != domain.KindDiscussion || got.Scope != domain.KindAppliesToDiscussion {
		t.Fatalf("expected imported discussion actionItem to default to discussion scope, got %#v", got)
	}
	if _, ok := repo.kindDefs[domain.KindID(domain.KindRefinement)]; !ok {
		t.Fatal("expected imported kind definition refinement")
	}
	allowed := repo.projectAllowedKinds["p1"]
	if len(allowed) != 1 || allowed[0] != domain.KindID(domain.KindRefinement) {
		t.Fatalf("expected imported project allowlist for p1, got %#v", allowed)
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

	// The invalid-phase-parent and valid-nested-phase cases were ripped when
	// the 12-value Kind enum removed the Phase/Branch kinds and their
	// snapshot-side parent check.

	orphanHandoff := Snapshot{
		Version: SnapshotVersion,
		Projects: []SnapshotProject{
			{ID: "p3", Name: "C", Slug: "c", CreatedAt: now, UpdatedAt: now},
		},
		Columns: []SnapshotColumn{
			{ID: "c3", ProjectID: "p3", Name: "To Do", Position: 0, CreatedAt: now, UpdatedAt: now},
		},
		ActionItems: []SnapshotActionItem{
			{ID: "t3", ProjectID: "p3", ColumnID: "c3", Position: 0, Title: "ActionItem", Priority: domain.PriorityMedium, CreatedAt: now, UpdatedAt: now},
		},
		Handoffs: []SnapshotHandoff{
			{
				ID:             "handoff-missing",
				ProjectID:      "p3",
				ScopeType:      domain.ScopeLevelActionItem,
				ScopeID:        "missing-actionItem",
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
		ActionItems: []SnapshotActionItem{
			{ID: "t1", ProjectID: "p1", ColumnID: "c1", Position: 0, Title: "Failed actionItem", Priority: domain.PriorityMedium, LifecycleState: domain.StateFailed, CreatedAt: now, UpdatedAt: now},
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
		ActionItems: []SnapshotActionItem{
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

// TestSnapshotActionItemRoleRoundTripPreservesAllRoles verifies that every
// member of the closed Role enum survives a domain → snapshot → domain
// round-trip via snapshotActionItemFromDomain and (SnapshotActionItem).toDomain.
func TestSnapshotActionItemRoleRoundTripPreservesAllRoles(t *testing.T) {
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		role domain.Role
	}{
		{name: "builder", role: domain.RoleBuilder},
		{name: "qa-proof", role: domain.RoleQAProof},
		{name: "qa-falsification", role: domain.RoleQAFalsification},
		{name: "qa-a11y", role: domain.RoleQAA11y},
		{name: "qa-visual", role: domain.RoleQAVisual},
		{name: "design", role: domain.RoleDesign},
		{name: "commit", role: domain.RoleCommit},
		{name: "planner", role: domain.RolePlanner},
		{name: "research", role: domain.RoleResearch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:        "t-role",
				ProjectID: "p1",
				ColumnID:  "c1",
				Position:  0,
				Title:     "Role round-trip",
				Priority:  domain.PriorityMedium,
				Kind:      domain.KindBuild,
				Role:      tc.role,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if snap.Role != tc.role {
				t.Fatalf("snapshotActionItemFromDomain dropped role: got %q, want %q", snap.Role, tc.role)
			}
			hydrated := snap.toDomain()
			if hydrated.Role != tc.role {
				t.Fatalf("toDomain dropped role: got %q, want %q", hydrated.Role, tc.role)
			}
		})
	}
}

// TestSnapshotActionItemRoleEmptyRoundTripsEmpty verifies that an unset Role
// stays empty across the snapshot round-trip and that omitempty drops the
// JSON key on serialize.
func TestSnapshotActionItemRoleEmptyRoundTripsEmpty(t *testing.T) {
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	original, err := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:        "t-empty-role",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "Empty role",
		Priority:  domain.PriorityMedium,
		Kind:      domain.KindBuild,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItem() error = %v", err)
	}
	if original.Role != "" {
		t.Fatalf("expected zero-value Role on freshly constructed ActionItem, got %q", original.Role)
	}
	snap := snapshotActionItemFromDomain(original)
	if snap.Role != "" {
		t.Fatalf("snapshotActionItemFromDomain invented a role: got %q, want \"\"", snap.Role)
	}
	hydrated := snap.toDomain()
	if hydrated.Role != "" {
		t.Fatalf("toDomain invented a role: got %q, want \"\"", hydrated.Role)
	}
}

// TestSnapshotActionItemRoleJSONShape verifies the on-the-wire JSON shape:
// the role key is present when set and omitted when empty (omitempty contract).
func TestSnapshotActionItemRoleJSONShape(t *testing.T) {
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)

	withRole := SnapshotActionItem{
		ID:             "t-json-with",
		ProjectID:      "p1",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		Role:           domain.RoleBuilder,
		LifecycleState: domain.StateTodo,
		ColumnID:       "c1",
		Title:          "With role",
		Priority:       domain.PriorityMedium,
		Labels:         []string{},
		CreatedByActor: "tester",
		UpdatedByActor: "tester",
		UpdatedByType:  domain.ActorTypeUser,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	rawWith, err := json.Marshal(withRole)
	if err != nil {
		t.Fatalf("json.Marshal(withRole) error = %v", err)
	}
	if !strings.Contains(string(rawWith), `"role":"builder"`) {
		t.Fatalf("expected role key with builder value in JSON, got %s", rawWith)
	}

	withoutRole := withRole
	withoutRole.ID = "t-json-without"
	withoutRole.Role = ""
	rawWithout, err := json.Marshal(withoutRole)
	if err != nil {
		t.Fatalf("json.Marshal(withoutRole) error = %v", err)
	}
	if strings.Contains(string(rawWithout), `"role"`) {
		t.Fatalf("expected role key absent when empty (omitempty), got %s", rawWithout)
	}

	// Round-trip the on-the-wire form back through json.Unmarshal to confirm
	// the role tag matches on both directions of the wire boundary.
	var decodedWith SnapshotActionItem
	if err := json.Unmarshal(rawWith, &decodedWith); err != nil {
		t.Fatalf("json.Unmarshal(rawWith) error = %v", err)
	}
	if decodedWith.Role != domain.RoleBuilder {
		t.Fatalf("json round-trip dropped role: got %q, want %q", decodedWith.Role, domain.RoleBuilder)
	}
	var decodedWithout SnapshotActionItem
	if err := json.Unmarshal(rawWithout, &decodedWithout); err != nil {
		t.Fatalf("json.Unmarshal(rawWithout) error = %v", err)
	}
	if decodedWithout.Role != "" {
		t.Fatalf("expected empty role after unmarshal of role-less JSON, got %q", decodedWithout.Role)
	}
}

// TestSnapshotActionItemStructuralTypeRoundTripPreservesAllValues verifies
// that every member of the closed StructuralType enum survives the
// domain → snapshot → domain round-trip exactly. Mirrors the Drop 2.2
// Role round-trip precedent line-for-line so the structural-type axis
// gets the same regression coverage as the role axis.
func TestSnapshotActionItemStructuralTypeRoundTripPreservesAllValues(t *testing.T) {
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name           string
		structuralType domain.StructuralType
	}{
		{name: "drop", structuralType: domain.StructuralTypeDrop},
		{name: "segment", structuralType: domain.StructuralTypeSegment},
		{name: "confluence", structuralType: domain.StructuralTypeConfluence},
		{name: "droplet", structuralType: domain.StructuralTypeDroplet},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:             "t-structural-type",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "StructuralType round-trip",
				Priority:       domain.PriorityMedium,
				Kind:           domain.KindBuild,
				StructuralType: tc.structuralType,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if snap.StructuralType != tc.structuralType {
				t.Fatalf("snapshotActionItemFromDomain dropped structural_type: got %q, want %q", snap.StructuralType, tc.structuralType)
			}
			hydrated := snap.toDomain()
			if hydrated.StructuralType != tc.structuralType {
				t.Fatalf("toDomain dropped structural_type: got %q, want %q", hydrated.StructuralType, tc.structuralType)
			}
		})
	}
}

// TestSnapshotActionItemIrreducibleRoundTripPreservesBothStates verifies
// that the Irreducible bool survives the domain → snapshot → domain
// round-trip in both true and false states. The false case guards against
// `omitempty` silently corrupting the value: a missing JSON field
// deserializes to false, which happens to match the false input — but the
// in-memory domain → snapshot → domain path tested here does not pass
// through JSON, so a copy bug in either direction would still surface.
func TestSnapshotActionItemIrreducibleRoundTripPreservesBothStates(t *testing.T) {
	now := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name        string
		irreducible bool
	}{
		{name: "irreducible-true", irreducible: true},
		{name: "irreducible-false", irreducible: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:          "t-irreducible",
				ProjectID:   "p1",
				ColumnID:    "c1",
				Position:    0,
				Title:       "Irreducible round-trip",
				Priority:    domain.PriorityMedium,
				Kind:        domain.KindBuild,
				Irreducible: tc.irreducible,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if snap.Irreducible != tc.irreducible {
				t.Fatalf("snapshotActionItemFromDomain dropped irreducible: got %v, want %v", snap.Irreducible, tc.irreducible)
			}
			hydrated := snap.toDomain()
			if hydrated.Irreducible != tc.irreducible {
				t.Fatalf("toDomain dropped irreducible: got %v, want %v", hydrated.Irreducible, tc.irreducible)
			}
		})
	}
}

// TestSnapshotActionItemOwnerAndDropNumberRoundTrip verifies that the four
// new domain primitives added in Drop 3 droplet 3.21 — Owner, DropNumber,
// Persistent, DevGated — survive the domain → snapshot → domain round-trip
// in both their dominant zero-value cases (Owner="", DropNumber=0, false
// bools) and their representative non-zero cases. Legacy-format
// compatibility (pre-3.21 snapshots without these fields) is covered by the
// `omitempty` JSON tags: missing fields deserialize to their zero values,
// which match the legitimate domain defaults — no SnapshotVersion bump
// required. The in-memory domain → snapshot → domain path tested here does
// NOT pass through JSON, so a copy bug in either direction would still
// surface on every case (mirrors the Irreducible-round-trip rationale).
func TestSnapshotActionItemOwnerAndDropNumberRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		owner      string
		dropNumber int
		persistent bool
		devGated   bool
	}{
		{name: "all-zero-values", owner: "", dropNumber: 0, persistent: false, devGated: false},
		{name: "steward-anchor", owner: "STEWARD", dropNumber: 5, persistent: true, devGated: true},
		{name: "owner-only", owner: "STEWARD", dropNumber: 0, persistent: false, devGated: false},
		{name: "drop-number-only", owner: "", dropNumber: 3, persistent: false, devGated: false},
		{name: "persistent-only", owner: "", dropNumber: 0, persistent: true, devGated: false},
		{name: "dev-gated-only", owner: "", dropNumber: 0, persistent: false, devGated: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:         "t-owner-dropnumber",
				ProjectID:  "p1",
				ColumnID:   "c1",
				Position:   0,
				Title:      "Owner/DropNumber/Persistent/DevGated round-trip",
				Priority:   domain.PriorityMedium,
				Kind:       domain.KindBuild,
				Owner:      tc.owner,
				DropNumber: tc.dropNumber,
				Persistent: tc.persistent,
				DevGated:   tc.devGated,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if snap.Owner != tc.owner {
				t.Fatalf("snapshotActionItemFromDomain dropped owner: got %q, want %q", snap.Owner, tc.owner)
			}
			if snap.DropNumber != tc.dropNumber {
				t.Fatalf("snapshotActionItemFromDomain dropped drop_number: got %d, want %d", snap.DropNumber, tc.dropNumber)
			}
			if snap.Persistent != tc.persistent {
				t.Fatalf("snapshotActionItemFromDomain dropped persistent: got %v, want %v", snap.Persistent, tc.persistent)
			}
			if snap.DevGated != tc.devGated {
				t.Fatalf("snapshotActionItemFromDomain dropped dev_gated: got %v, want %v", snap.DevGated, tc.devGated)
			}
			hydrated := snap.toDomain()
			if hydrated.Owner != tc.owner {
				t.Fatalf("toDomain dropped owner: got %q, want %q", hydrated.Owner, tc.owner)
			}
			if hydrated.DropNumber != tc.dropNumber {
				t.Fatalf("toDomain dropped drop_number: got %d, want %d", hydrated.DropNumber, tc.dropNumber)
			}
			if hydrated.Persistent != tc.persistent {
				t.Fatalf("toDomain dropped persistent: got %v, want %v", hydrated.Persistent, tc.persistent)
			}
			if hydrated.DevGated != tc.devGated {
				t.Fatalf("toDomain dropped dev_gated: got %v, want %v", hydrated.DevGated, tc.devGated)
			}
		})
	}
}

// TestSnapshotActionItemOwnerLegacyFormatCompatibility verifies that a
// pre-droplet-3.21 snapshot — one whose JSON wire form OMITS the four new
// fields entirely — deserializes cleanly with all four set to their zero
// values. This is the production legacy-format path: the `omitempty` tags
// on Owner / DropNumber / Persistent / DevGated keep older snapshots
// forward-compatible without bumping SnapshotVersion. Failure here would
// indicate a copy bug at the JSON boundary that the in-memory round-trip
// above cannot catch.
func TestSnapshotActionItemOwnerLegacyFormatCompatibility(t *testing.T) {
	legacyJSON := []byte(`{
		"id": "t-legacy",
		"project_id": "p1",
		"kind": "build",
		"structural_type": "droplet",
		"lifecycle_state": "todo",
		"column_id": "c1",
		"position": 0,
		"title": "Legacy snapshot",
		"description": "pre-3.21 row, no owner/drop_number/persistent/dev_gated",
		"priority": "medium",
		"labels": [],
		"metadata": {},
		"created_by_actor": "u1",
		"updated_by_actor": "u1",
		"updated_by_type": "user",
		"created_at": "2026-04-01T00:00:00Z",
		"updated_at": "2026-04-01T00:00:00Z"
	}`)
	var snap SnapshotActionItem
	if err := json.Unmarshal(legacyJSON, &snap); err != nil {
		t.Fatalf("legacy snapshot unmarshal error = %v", err)
	}
	if snap.Owner != "" {
		t.Fatalf("legacy snapshot Owner = %q, want empty", snap.Owner)
	}
	if snap.DropNumber != 0 {
		t.Fatalf("legacy snapshot DropNumber = %d, want 0", snap.DropNumber)
	}
	if snap.Persistent != false {
		t.Fatalf("legacy snapshot Persistent = %v, want false", snap.Persistent)
	}
	if snap.DevGated != false {
		t.Fatalf("legacy snapshot DevGated = %v, want false", snap.DevGated)
	}
	hydrated := snap.toDomain()
	if hydrated.Owner != "" || hydrated.DropNumber != 0 || hydrated.Persistent != false || hydrated.DevGated != false {
		t.Fatalf("legacy snapshot toDomain leaked non-zero defaults: owner=%q drop_number=%d persistent=%v dev_gated=%v",
			hydrated.Owner, hydrated.DropNumber, hydrated.Persistent, hydrated.DevGated)
	}
}

// TestSnapshotActionItemPathsRoundTrip verifies that the Paths slice added
// in Drop 4a droplet 4a.5 survives the domain → snapshot → domain round-
// trip across the empty zero-value case and representative populated cases.
// Insertion order must be preserved end-to-end (the dispatcher's lock
// manager reads the slice as ordered). Legacy-format compatibility (pre-
// 4a.5 snapshots without the field) is covered by the json:"paths,omitempty"
// tag — missing field deserializes to nil, the legitimate zero value, with
// no SnapshotVersion bump.
func TestSnapshotActionItemPathsRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		paths []string
		want  []string
	}{
		{name: "nil-zero-value", paths: nil, want: nil},
		{name: "single-path", paths: []string{"internal/domain/action_item.go"}, want: []string{"internal/domain/action_item.go"}},
		{name: "multi-path-order-preserved", paths: []string{"a/b/c.go", "d/e/f.go", "g.go"}, want: []string{"a/b/c.go", "d/e/f.go", "g.go"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Supply a covering Packages entry whenever Paths is populated
			// so the Drop 4a droplet 4a.6 coverage invariant doesn't shadow
			// the Paths-round-trip assertions this test cares about.
			var pkgs []string
			if len(tc.paths) > 0 {
				pkgs = []string{"internal/domain"}
			}
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:        "t-paths",
				ProjectID: "p1",
				ColumnID:  "c1",
				Position:  0,
				Title:     "Paths round-trip",
				Priority:  domain.PriorityMedium,
				Kind:      domain.KindBuild,
				Paths:     tc.paths,
				Packages:  pkgs,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if len(snap.Paths) != len(tc.want) {
				t.Fatalf("snapshotActionItemFromDomain dropped paths: got %#v, want %#v", snap.Paths, tc.want)
			}
			for i := range tc.want {
				if snap.Paths[i] != tc.want[i] {
					t.Fatalf("snapshot Paths[%d] = %q, want %q", i, snap.Paths[i], tc.want[i])
				}
			}
			hydrated := snap.toDomain()
			if len(hydrated.Paths) != len(tc.want) {
				t.Fatalf("toDomain dropped paths: got %#v, want %#v", hydrated.Paths, tc.want)
			}
			for i := range tc.want {
				if hydrated.Paths[i] != tc.want[i] {
					t.Fatalf("hydrated Paths[%d] = %q, want %q", i, hydrated.Paths[i], tc.want[i])
				}
			}
		})
	}
}

// TestSnapshotActionItemPathsLegacyFormatCompatibility verifies that a
// pre-droplet-4a.5 snapshot — one whose JSON wire form OMITS the paths
// field entirely — deserializes cleanly with Paths=nil. The omitempty tag
// keeps older snapshots forward-compatible without a SnapshotVersion bump.
func TestSnapshotActionItemPathsLegacyFormatCompatibility(t *testing.T) {
	legacyJSON := []byte(`{
		"id": "t-legacy-paths",
		"project_id": "p1",
		"kind": "build",
		"structural_type": "droplet",
		"lifecycle_state": "todo",
		"column_id": "c1",
		"position": 0,
		"title": "Legacy snapshot, no paths",
		"description": "pre-4a.5 row",
		"priority": "medium",
		"labels": [],
		"metadata": {},
		"created_by_actor": "u1",
		"updated_by_actor": "u1",
		"updated_by_type": "user",
		"created_at": "2026-04-01T00:00:00Z",
		"updated_at": "2026-04-01T00:00:00Z"
	}`)
	var snap SnapshotActionItem
	if err := json.Unmarshal(legacyJSON, &snap); err != nil {
		t.Fatalf("legacy snapshot unmarshal error = %v", err)
	}
	if len(snap.Paths) != 0 {
		t.Fatalf("legacy snapshot Paths = %#v, want nil", snap.Paths)
	}
	hydrated := snap.toDomain()
	if len(hydrated.Paths) != 0 {
		t.Fatalf("legacy snapshot toDomain Paths = %#v, want nil", hydrated.Paths)
	}
}

// TestSnapshotActionItemPackagesRoundTrip verifies that the Packages slice
// added in Drop 4a droplet 4a.6 survives the domain → snapshot → domain
// round-trip across the empty zero-value case and representative populated
// cases. Insertion order must be preserved end-to-end (the dispatcher's
// lock manager reads the slice as ordered). Legacy-format compatibility
// (pre-4a.6 snapshots without the field) is covered by the
// json:"packages,omitempty" tag — missing field deserializes to nil, the
// legitimate zero value, with no SnapshotVersion bump. Each populated case
// supplies a covering Paths slice so the domain coverage invariant
// ("non-empty Paths requires non-empty Packages") doesn't reject the
// constructor.
func TestSnapshotActionItemPackagesRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		paths    []string
		packages []string
		want     []string
	}{
		{name: "nil-zero-value", paths: nil, packages: nil, want: nil},
		{name: "single-internal-package", paths: []string{"internal/domain/action_item.go"}, packages: []string{"internal/domain"}, want: []string{"internal/domain"}},
		{name: "multi-package-order-preserved", paths: []string{"a/b.go", "c/d.go", "e/f.go"}, packages: []string{"internal/domain", "internal/app", "github.com/foo/bar"}, want: []string{"internal/domain", "internal/app", "github.com/foo/bar"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:        "t-packages",
				ProjectID: "p1",
				ColumnID:  "c1",
				Position:  0,
				Title:     "Packages round-trip",
				Priority:  domain.PriorityMedium,
				Kind:      domain.KindBuild,
				Paths:     tc.paths,
				Packages:  tc.packages,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if len(snap.Packages) != len(tc.want) {
				t.Fatalf("snapshotActionItemFromDomain dropped packages: got %#v, want %#v", snap.Packages, tc.want)
			}
			for i := range tc.want {
				if snap.Packages[i] != tc.want[i] {
					t.Fatalf("snapshot Packages[%d] = %q, want %q", i, snap.Packages[i], tc.want[i])
				}
			}
			hydrated := snap.toDomain()
			if len(hydrated.Packages) != len(tc.want) {
				t.Fatalf("toDomain dropped packages: got %#v, want %#v", hydrated.Packages, tc.want)
			}
			for i := range tc.want {
				if hydrated.Packages[i] != tc.want[i] {
					t.Fatalf("hydrated Packages[%d] = %q, want %q", i, hydrated.Packages[i], tc.want[i])
				}
			}
		})
	}
}

// TestSnapshotActionItemPackagesLegacyFormatCompatibility verifies that a
// pre-droplet-4a.6 snapshot — one whose JSON wire form OMITS the packages
// field entirely — deserializes cleanly with Packages=nil. The omitempty
// tag keeps older snapshots forward-compatible without a SnapshotVersion
// bump.
func TestSnapshotActionItemPackagesLegacyFormatCompatibility(t *testing.T) {
	legacyJSON := []byte(`{
		"id": "t-legacy-packages",
		"project_id": "p1",
		"kind": "build",
		"structural_type": "droplet",
		"lifecycle_state": "todo",
		"column_id": "c1",
		"position": 0,
		"title": "Legacy snapshot, no packages",
		"description": "pre-4a.6 row",
		"priority": "medium",
		"labels": [],
		"metadata": {},
		"created_by_actor": "u1",
		"updated_by_actor": "u1",
		"updated_by_type": "user",
		"created_at": "2026-04-01T00:00:00Z",
		"updated_at": "2026-04-01T00:00:00Z"
	}`)
	var snap SnapshotActionItem
	if err := json.Unmarshal(legacyJSON, &snap); err != nil {
		t.Fatalf("legacy snapshot unmarshal error = %v", err)
	}
	if len(snap.Packages) != 0 {
		t.Fatalf("legacy snapshot Packages = %#v, want nil", snap.Packages)
	}
	hydrated := snap.toDomain()
	if len(hydrated.Packages) != 0 {
		t.Fatalf("legacy snapshot toDomain Packages = %#v, want nil", hydrated.Packages)
	}
}

// TestSnapshotActionItemFilesRoundTrip verifies that the Files slice added
// in Drop 4a droplet 4a.7 survives the domain → snapshot → domain round-
// trip across the empty zero-value case and representative populated
// cases. Insertion order must be preserved end-to-end (the Drop 4.5
// file-viewer pane reads the slice as ordered). Legacy-format
// compatibility (pre-4a.7 snapshots without the field) is covered by the
// json:"files,omitempty" tag — missing field deserializes to nil, the
// legitimate zero value, with no SnapshotVersion bump. Files is disjoint-
// axis with Paths so the populated cases also exercise legitimate
// overlap with Paths to assert no cross-axis check rejects the round-
// trip; the covering Packages entry is supplied so the Paths/Packages
// coverage invariant doesn't shadow the Files-round-trip assertion.
func TestSnapshotActionItemFilesRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 3, 11, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		paths    []string
		packages []string
		files    []string
		want     []string
	}{
		{name: "nil-zero-value", paths: nil, packages: nil, files: nil, want: nil},
		{name: "single-file", paths: nil, packages: nil, files: []string{"docs/README.md"}, want: []string{"docs/README.md"}},
		{name: "multi-file-order-preserved", paths: nil, packages: nil, files: []string{"docs/A.md", "docs/B.md", "docs/C.md"}, want: []string{"docs/A.md", "docs/B.md", "docs/C.md"}},
		{name: "files-overlap-with-paths-allowed", paths: []string{"internal/domain/action_item.go"}, packages: []string{"internal/domain"}, files: []string{"internal/domain/action_item.go"}, want: []string{"internal/domain/action_item.go"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:        "t-files",
				ProjectID: "p1",
				ColumnID:  "c1",
				Position:  0,
				Title:     "Files round-trip",
				Priority:  domain.PriorityMedium,
				Kind:      domain.KindBuild,
				Paths:     tc.paths,
				Packages:  tc.packages,
				Files:     tc.files,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if len(snap.Files) != len(tc.want) {
				t.Fatalf("snapshotActionItemFromDomain dropped files: got %#v, want %#v", snap.Files, tc.want)
			}
			for i := range tc.want {
				if snap.Files[i] != tc.want[i] {
					t.Fatalf("snapshot Files[%d] = %q, want %q", i, snap.Files[i], tc.want[i])
				}
			}
			hydrated := snap.toDomain()
			if len(hydrated.Files) != len(tc.want) {
				t.Fatalf("toDomain dropped files: got %#v, want %#v", hydrated.Files, tc.want)
			}
			for i := range tc.want {
				if hydrated.Files[i] != tc.want[i] {
					t.Fatalf("hydrated Files[%d] = %q, want %q", i, hydrated.Files[i], tc.want[i])
				}
			}
		})
	}
}

// TestSnapshotActionItemFilesLegacyFormatCompatibility verifies that a
// pre-droplet-4a.7 snapshot — one whose JSON wire form OMITS the files
// field entirely — deserializes cleanly with Files=nil. The omitempty
// tag keeps older snapshots forward-compatible without a SnapshotVersion
// bump.
func TestSnapshotActionItemFilesLegacyFormatCompatibility(t *testing.T) {
	legacyJSON := []byte(`{
		"id": "t-legacy-files",
		"project_id": "p1",
		"kind": "build",
		"structural_type": "droplet",
		"lifecycle_state": "todo",
		"column_id": "c1",
		"position": 0,
		"title": "Legacy snapshot, no files",
		"description": "pre-4a.7 row",
		"priority": "medium",
		"labels": [],
		"metadata": {},
		"created_by_actor": "u1",
		"updated_by_actor": "u1",
		"updated_by_type": "user",
		"created_at": "2026-04-01T00:00:00Z",
		"updated_at": "2026-04-01T00:00:00Z"
	}`)
	var snap SnapshotActionItem
	if err := json.Unmarshal(legacyJSON, &snap); err != nil {
		t.Fatalf("legacy snapshot unmarshal error = %v", err)
	}
	if len(snap.Files) != 0 {
		t.Fatalf("legacy snapshot Files = %#v, want nil", snap.Files)
	}
	hydrated := snap.toDomain()
	if len(hydrated.Files) != 0 {
		t.Fatalf("legacy snapshot toDomain Files = %#v, want nil", hydrated.Files)
	}
}

// TestSnapshotActionItemStartCommitRoundTrip verifies that the StartCommit
// string added in Drop 4a droplet 4a.8 survives the domain → snapshot →
// domain round-trip across the empty zero-value case and representative
// populated cases (short-SHA, full-SHA, free-form identifier). Surrounding
// whitespace is trimmed at NewActionItem time so the snapshot stage holds
// already-trimmed values; the round-trip preserves the trimmed form
// verbatim. Legacy-format compatibility (pre-4a.8 snapshots without the
// field) is covered by the json:"start_commit,omitempty" tag — missing
// field deserializes to "", the legitimate zero value, with no
// SnapshotVersion bump.
func TestSnapshotActionItemStartCommitRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 3, 11, 0, 0, 0, time.UTC)
	cases := []struct {
		name        string
		startCommit string
		want        string
	}{
		{name: "empty-zero-value", startCommit: "", want: ""},
		{name: "short-SHA", startCommit: "0cf5194", want: "0cf5194"},
		{name: "full-SHA", startCommit: "0cf5194d4cb6c8d4f9b9b1d7e1f9d3c2b4e5a6f7", want: "0cf5194d4cb6c8d4f9b9b1d7e1f9d3c2b4e5a6f7"},
		{name: "free-form-identifier", startCommit: "branch/feature@head", want: "branch/feature@head"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:          "t-startcommit",
				ProjectID:   "p1",
				ColumnID:    "c1",
				Position:    0,
				Title:       "StartCommit round-trip",
				Priority:    domain.PriorityMedium,
				Kind:        domain.KindBuild,
				StartCommit: tc.startCommit,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if snap.StartCommit != tc.want {
				t.Fatalf("snapshotActionItemFromDomain StartCommit = %q, want %q", snap.StartCommit, tc.want)
			}
			hydrated := snap.toDomain()
			if hydrated.StartCommit != tc.want {
				t.Fatalf("toDomain StartCommit = %q, want %q", hydrated.StartCommit, tc.want)
			}
		})
	}
}

// TestSnapshotActionItemStartCommitLegacyFormatCompatibility verifies that a
// pre-droplet-4a.8 snapshot — one whose JSON wire form OMITS the
// start_commit field entirely — deserializes cleanly with StartCommit="".
// The omitempty tag keeps older snapshots forward-compatible without a
// SnapshotVersion bump.
func TestSnapshotActionItemStartCommitLegacyFormatCompatibility(t *testing.T) {
	legacyJSON := []byte(`{
		"id": "t-legacy-startcommit",
		"project_id": "p1",
		"kind": "build",
		"structural_type": "droplet",
		"lifecycle_state": "todo",
		"column_id": "c1",
		"position": 0,
		"title": "Legacy snapshot, no start_commit",
		"description": "pre-4a.8 row",
		"priority": "medium",
		"labels": [],
		"metadata": {},
		"created_by_actor": "u1",
		"updated_by_actor": "u1",
		"updated_by_type": "user",
		"created_at": "2026-04-01T00:00:00Z",
		"updated_at": "2026-04-01T00:00:00Z"
	}`)
	var snap SnapshotActionItem
	if err := json.Unmarshal(legacyJSON, &snap); err != nil {
		t.Fatalf("legacy snapshot unmarshal error = %v", err)
	}
	if snap.StartCommit != "" {
		t.Fatalf("legacy snapshot StartCommit = %q, want empty string", snap.StartCommit)
	}
	hydrated := snap.toDomain()
	if hydrated.StartCommit != "" {
		t.Fatalf("legacy snapshot toDomain StartCommit = %q, want empty string", hydrated.StartCommit)
	}
}

// TestSnapshotActionItemEndCommitRoundTrip verifies that the EndCommit
// string added in Drop 4a droplet 4a.9 survives the domain → snapshot →
// domain round-trip across the empty zero-value case and representative
// populated cases (short-SHA, full-SHA, free-form identifier). Surrounding
// whitespace is trimmed at NewActionItem time so the snapshot stage holds
// already-trimmed values; the round-trip preserves the trimmed form
// verbatim. Legacy-format compatibility (pre-4a.9 snapshots without the
// field) is covered by the json:"end_commit,omitempty" tag — missing field
// deserializes to "", the legitimate zero value, with no SnapshotVersion
// bump.
func TestSnapshotActionItemEndCommitRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 3, 11, 0, 0, 0, time.UTC)
	cases := []struct {
		name      string
		endCommit string
		want      string
	}{
		{name: "empty-zero-value", endCommit: "", want: ""},
		{name: "short-SHA", endCommit: "0cf5194", want: "0cf5194"},
		{name: "full-SHA", endCommit: "0cf5194d4cb6c8d4f9b9b1d7e1f9d3c2b4e5a6f7", want: "0cf5194d4cb6c8d4f9b9b1d7e1f9d3c2b4e5a6f7"},
		{name: "free-form-identifier", endCommit: "branch/feature@head", want: "branch/feature@head"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := domain.NewActionItemForTest(domain.ActionItemInput{
				ID:        "t-endcommit",
				ProjectID: "p1",
				ColumnID:  "c1",
				Position:  0,
				Title:     "EndCommit round-trip",
				Priority:  domain.PriorityMedium,
				Kind:      domain.KindBuild,
				EndCommit: tc.endCommit,
			}, now)
			if err != nil {
				t.Fatalf("NewActionItem() error = %v", err)
			}
			snap := snapshotActionItemFromDomain(original)
			if snap.EndCommit != tc.want {
				t.Fatalf("snapshotActionItemFromDomain EndCommit = %q, want %q", snap.EndCommit, tc.want)
			}
			hydrated := snap.toDomain()
			if hydrated.EndCommit != tc.want {
				t.Fatalf("toDomain EndCommit = %q, want %q", hydrated.EndCommit, tc.want)
			}
		})
	}
}

// TestSnapshotActionItemEndCommitLegacyFormatCompatibility verifies that a
// pre-droplet-4a.9 snapshot — one whose JSON wire form OMITS the end_commit
// field entirely — deserializes cleanly with EndCommit="". The omitempty
// tag keeps older snapshots forward-compatible without a SnapshotVersion
// bump.
func TestSnapshotActionItemEndCommitLegacyFormatCompatibility(t *testing.T) {
	legacyJSON := []byte(`{
		"id": "t-legacy-endcommit",
		"project_id": "p1",
		"kind": "build",
		"structural_type": "droplet",
		"lifecycle_state": "todo",
		"column_id": "c1",
		"position": 0,
		"title": "Legacy snapshot, no end_commit",
		"description": "pre-4a.9 row",
		"priority": "medium",
		"labels": [],
		"metadata": {},
		"created_by_actor": "u1",
		"updated_by_actor": "u1",
		"updated_by_type": "user",
		"created_at": "2026-04-01T00:00:00Z",
		"updated_at": "2026-04-01T00:00:00Z"
	}`)
	var snap SnapshotActionItem
	if err := json.Unmarshal(legacyJSON, &snap); err != nil {
		t.Fatalf("legacy snapshot unmarshal error = %v", err)
	}
	if snap.EndCommit != "" {
		t.Fatalf("legacy snapshot EndCommit = %q, want empty string", snap.EndCommit)
	}
	hydrated := snap.toDomain()
	if hydrated.EndCommit != "" {
		t.Fatalf("legacy snapshot toDomain EndCommit = %q, want empty string", hydrated.EndCommit)
	}
}
