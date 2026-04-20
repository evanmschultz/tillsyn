package app

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// boolPtr returns a pointer to one bool value.
func boolPtr(v bool) *bool {
	return &v
}

// newDeterministicService builds a service with deterministic IDs and clock values for tests.
func newDeterministicService(repo *fakeRepo, now time.Time, cfg ServiceConfig) *Service {
	idCounter := 0
	return NewService(repo, func() string {
		idCounter++
		return "id-" + time.Unix(int64(idCounter), 0).UTC().Format("150405")
	}, func() time.Time {
		return now
	}, cfg)
}

// TestServiceSetAndListProjectAllowedKindsValidation verifies allowlist write and list behavior.
func TestServiceSetAndListProjectAllowedKindsValidation(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Kinds", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := svc.SetProjectAllowedKinds(context.Background(), SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   nil,
	}); !errors.Is(err, domain.ErrKindNotAllowed) {
		t.Fatalf("SetProjectAllowedKinds(empty) error = %v, want ErrKindNotAllowed", err)
	}
	if err := svc.SetProjectAllowedKinds(context.Background(), SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   []domain.KindID{"unknown-kind"},
	}); !errors.Is(err, domain.ErrKindNotFound) {
		t.Fatalf("SetProjectAllowedKinds(unknown) error = %v, want ErrKindNotFound", err)
	}
	if err := svc.SetProjectAllowedKinds(context.Background(), SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   []domain.KindID{"actionItem", "phase", "actionItem"},
	}); err != nil {
		t.Fatalf("SetProjectAllowedKinds(valid) error = %v", err)
	}
	kinds, err := svc.ListProjectAllowedKinds(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds() error = %v", err)
	}
	want := []domain.KindID{"actionItem", "phase"}
	if !slices.Equal(kinds, want) {
		t.Fatalf("ListProjectAllowedKinds() = %#v, want %#v", kinds, want)
	}
	if _, err := svc.ListProjectAllowedKinds(context.Background(), ""); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("ListProjectAllowedKinds(empty id) error = %v, want ErrInvalidID", err)
	}
}

// TestServiceListKindDefinitionsAndUpsert verifies upsert and deterministic list sorting behavior.
func TestServiceListKindDefinitionsAndUpsert(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	if _, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "zeta",
		DisplayName: "Zeta",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToActionItem},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(create) error = %v", err)
	}
	updated, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "zeta",
		DisplayName: "Alpha",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToActionItem},
	})
	if err != nil {
		t.Fatalf("UpsertKindDefinition(update) error = %v", err)
	}
	if updated.DisplayName != "Alpha" {
		t.Fatalf("DisplayName = %q, want Alpha", updated.DisplayName)
	}
	kinds, err := svc.ListKindDefinitions(context.Background(), false)
	if err != nil {
		t.Fatalf("ListKindDefinitions() error = %v", err)
	}
	if len(kinds) == 0 {
		t.Fatal("ListKindDefinitions() expected non-empty catalog")
	}
	seen := false
	for _, kind := range kinds {
		if kind.ID == "zeta" {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatal("ListKindDefinitions() missing upserted kind zeta")
	}
	for idx := 1; idx < len(kinds); idx++ {
		prev := kinds[idx-1]
		next := kinds[idx]
		if prev.DisplayName > next.DisplayName {
			t.Fatalf("kinds not sorted at index %d: %q > %q", idx, prev.DisplayName, next.DisplayName)
		}
	}
}

// TestServiceCapabilityLeaseLifecycleAndRevokeAll verifies lease issue/heartbeat/renew/revoke flows.
func TestServiceCapabilityLeaseLifecycleAndRevokeAll(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		RequireAgentLease:  boolPtr(true),
		CapabilityLeaseTTL: time.Hour,
	})

	project, err := svc.CreateProject(context.Background(), "Leases", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	lease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1-instance",
		RequestedTTL:    30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}
	if _, err := svc.HeartbeatCapabilityLease(context.Background(), HeartbeatCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      "wrong-token",
	}); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("HeartbeatCapabilityLease(wrong token) error = %v, want ErrMutationLeaseInvalid", err)
	}
	heartbeatLease, err := svc.HeartbeatCapabilityLease(context.Background(), HeartbeatCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	if err != nil {
		t.Fatalf("HeartbeatCapabilityLease() error = %v", err)
	}
	if heartbeatLease.HeartbeatAt.IsZero() {
		t.Fatal("HeartbeatCapabilityLease() expected HeartbeatAt")
	}
	renewed, err := svc.RenewCapabilityLease(context.Background(), RenewCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
		TTL:             2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("RenewCapabilityLease() error = %v", err)
	}
	if !renewed.ExpiresAt.After(lease.ExpiresAt) {
		t.Fatalf("RenewCapabilityLease() expiry %v must be after %v", renewed.ExpiresAt, lease.ExpiresAt)
	}
	revoked, err := svc.RevokeCapabilityLease(context.Background(), RevokeCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		Reason:          "manual revoke",
	})
	if err != nil {
		t.Fatalf("RevokeCapabilityLease() error = %v", err)
	}
	if !revoked.IsRevoked() {
		t.Fatal("RevokeCapabilityLease() expected revoked lease")
	}
	if _, err := svc.HeartbeatCapabilityLease(context.Background(), HeartbeatCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	}); !errors.Is(err, domain.ErrMutationLeaseRevoked) {
		t.Fatalf("HeartbeatCapabilityLease(revoked) error = %v, want ErrMutationLeaseRevoked", err)
	}

	second, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "agent-2",
		AgentInstanceID: "agent-2-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(second) error = %v", err)
	}
	if err := svc.RevokeAllCapabilityLeases(context.Background(), RevokeAllCapabilityLeasesInput{
		ProjectID: "",
		ScopeType: domain.CapabilityScopeProject,
		ScopeID:   project.ID,
	}); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("RevokeAllCapabilityLeases(empty project) error = %v, want ErrInvalidID", err)
	}
	if err := svc.RevokeAllCapabilityLeases(context.Background(), RevokeAllCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeType("bad"),
		ScopeID:   project.ID,
	}); !errors.Is(err, domain.ErrInvalidCapabilityScope) {
		t.Fatalf("RevokeAllCapabilityLeases(bad scope) error = %v, want ErrInvalidCapabilityScope", err)
	}
	if err := svc.RevokeAllCapabilityLeases(context.Background(), RevokeAllCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeActionItem,
		ScopeID:   "missing-actionItem",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("RevokeAllCapabilityLeases(unknown actionItem scope) error = %v, want ErrNotFound", err)
	}
	// Guard against project-scoped root rows being treated as actionItem-scoped tuples.
	repo.tasks["project-root-item"] = domain.ActionItem{
		ID:        "project-root-item",
		ProjectID: project.ID,
		Scope:     domain.KindAppliesToProject,
	}
	if err := svc.RevokeAllCapabilityLeases(context.Background(), RevokeAllCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeActionItem,
		ScopeID:   "project-root-item",
	}); !errors.Is(err, domain.ErrInvalidCapabilityScope) {
		t.Fatalf("RevokeAllCapabilityLeases(project root as actionItem scope) error = %v, want ErrInvalidCapabilityScope", err)
	}
	if err := svc.RevokeAllCapabilityLeases(context.Background(), RevokeAllCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeProject,
		ScopeID:   project.ID,
	}); err != nil {
		t.Fatalf("RevokeAllCapabilityLeases() error = %v", err)
	}
	storedSecond, err := repo.GetCapabilityLease(context.Background(), second.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease(second) error = %v", err)
	}
	if !storedSecond.IsRevoked() {
		t.Fatal("RevokeAllCapabilityLeases() expected second lease to be revoked")
	}
}

// TestServiceEnforceMutationGuardBranches covers principal mutation-guard failure and success branches.
func TestServiceEnforceMutationGuardBranches(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		RequireAgentLease:  boolPtr(true),
		CapabilityLeaseTTL: time.Hour,
	})

	project, err := svc.CreateProject(context.Background(), "Guarded", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         "wrong-project",
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "bad-project",
		AgentInstanceID: "bad-project",
	}); !errors.Is(err, domain.ErrInvalidCapabilityScope) {
		t.Fatalf("IssueCapabilityLease(bad project scope) error = %v, want ErrInvalidCapabilityScope", err)
	}
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeBranch,
		ScopeID:         "missing-branch",
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "missing-branch",
		AgentInstanceID: "missing-branch",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("IssueCapabilityLease(missing branch) error = %v, want ErrNotFound", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if err := svc.enforceMutationGuard(context.Background(), project.ID, domain.ActorTypeUser, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); err != nil {
		t.Fatalf("enforceMutationGuard(user) error = %v", err)
	}
	if err := svc.enforceMutationGuard(context.Background(), project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); !errors.Is(err, domain.ErrMutationLeaseRequired) {
		t.Fatalf("enforceMutationGuard(no guard) error = %v, want ErrMutationLeaseRequired", err)
	}

	missingCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       "agent-x",
		AgentInstanceID: "missing",
		LeaseToken:      "missing-token",
	})
	if err := svc.enforceMutationGuard(missingCtx, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(missing lease) error = %v, want ErrMutationLeaseInvalid", err)
	}

	lease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "agent-y",
		AgentInstanceID: "agent-y-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}
	badIdentity := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       "other-name",
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	if err := svc.enforceMutationGuard(badIdentity, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(identity mismatch) error = %v, want ErrMutationLeaseInvalid", err)
	}

	validGuard := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       lease.AgentName,
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	if err := svc.enforceMutationGuard(validGuard, "wrong-project", domain.ActorTypeAgent, domain.CapabilityScopeProject, "wrong-project", domain.CapabilityActionEditNode); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(project mismatch) error = %v, want ErrMutationLeaseInvalid", err)
	}

	lease.Revoke("revoked", now)
	if err := repo.UpdateCapabilityLease(context.Background(), lease); err != nil {
		t.Fatalf("UpdateCapabilityLease(revoke) error = %v", err)
	}
	if err := svc.enforceMutationGuard(validGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); !errors.Is(err, domain.ErrMutationLeaseRevoked) {
		t.Fatalf("enforceMutationGuard(revoked) error = %v, want ErrMutationLeaseRevoked", err)
	}

	expired, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "agent-z",
		AgentInstanceID: "agent-z-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(expired) error = %v", err)
	}
	expired.ExpiresAt = now.Add(-time.Minute)
	if err := repo.UpdateCapabilityLease(context.Background(), expired); err != nil {
		t.Fatalf("UpdateCapabilityLease(expired) error = %v", err)
	}
	expiredGuard := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       expired.AgentName,
		AgentInstanceID: expired.InstanceID,
		LeaseToken:      expired.LeaseToken,
	})
	if err := svc.enforceMutationGuard(expiredGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); !errors.Is(err, domain.ErrMutationLeaseExpired) {
		t.Fatalf("enforceMutationGuard(expired) error = %v, want ErrMutationLeaseExpired", err)
	}

	branch, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.Kind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch A",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(branch) error = %v", err)
	}
	branchLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeBranch,
		ScopeID:         branch.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "agent-branch",
		AgentInstanceID: "agent-branch-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(branch) error = %v", err)
	}
	branchGuard := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       branchLease.AgentName,
		AgentInstanceID: branchLease.InstanceID,
		LeaseToken:      branchLease.LeaseToken,
	})
	if err := svc.enforceMutationGuard(branchGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(scope mismatch) error = %v, want ErrMutationLeaseInvalid", err)
	}
	if err := svc.enforceMutationGuard(branchGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeBranch, branch.ID, domain.CapabilityActionEditNode); err != nil {
		t.Fatalf("enforceMutationGuard(scope match) error = %v", err)
	}
	storedBranch, err := repo.GetCapabilityLease(context.Background(), branchLease.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease(branch) error = %v", err)
	}
	if storedBranch.HeartbeatAt.IsZero() {
		t.Fatal("enforceMutationGuard(scope match) expected heartbeat update")
	}
}

// TestCreateActionItemAppliesKindTemplateActions verifies checklist merge and child auto-create behavior.
func TestCreateActionItemAppliesKindTemplateActions(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	// Bootstrap built-in kinds first so project creation can resolve the default project kind.
	if _, err := svc.ListKindDefinitions(context.Background(), false); err != nil {
		t.Fatalf("ListKindDefinitions(bootstrap) error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "refactor",
		DisplayName: "Refactor",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToActionItem},
		Template: domain.KindTemplate{
			CompletionChecklist: []domain.ChecklistItem{
				{ID: "ck-run-tests", Text: "run package tests", Done: false},
			},
			AutoCreateChildren: []domain.KindTemplateChildSpec{
				{
					Title:       "Template Child",
					Description: "Auto-created child",
					Kind:        domain.KindID(domain.KindSubtask),
					AppliesTo:   domain.KindAppliesToSubtask,
					Labels:      []string{"templated"},
				},
			},
		},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(refactor) error = %v", err)
	}

	project, err := svc.CreateProject(context.Background(), "Template Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	parent, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Title:       "Parent ActionItem",
		Description: "Template parent",
		Kind:        domain.Kind("refactor"),
		Scope:       domain.KindAppliesToActionItem,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(refactor) error = %v", err)
	}
	storedParent, err := repo.GetActionItem(context.Background(), parent.ID)
	if err != nil {
		t.Fatalf("GetActionItem(parent) error = %v", err)
	}
	if len(storedParent.Metadata.CompletionContract.CompletionChecklist) != 1 {
		t.Fatalf("parent checklist len = %d, want 1", len(storedParent.Metadata.CompletionContract.CompletionChecklist))
	}

	tasks, err := svc.ListActionItems(context.Background(), project.ID, true)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	foundChild := false
	for _, actionItem := range tasks {
		if actionItem.ParentID == parent.ID && actionItem.Title == "Template Child" {
			foundChild = true
			if actionItem.Kind != domain.KindSubtask {
				t.Fatalf("child kind = %q, want subtask", actionItem.Kind)
			}
			if actionItem.Scope != domain.KindAppliesToSubtask {
				t.Fatalf("child scope = %q, want subtask", actionItem.Scope)
			}
		}
	}
	if !foundChild {
		t.Fatal("expected template-created child actionItem")
	}
}

// TestCreateProjectAppliesKindTemplateDefaultsAndChildren verifies project kinds seed metadata and root work.
func TestCreateProjectAppliesKindTemplateDefaultsAndChildren(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 11, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		DefaultDeleteMode:        DeleteModeArchive,
		AutoCreateProjectColumns: false,
	})

	if _, err := svc.ListKindDefinitions(context.Background(), false); err != nil {
		t.Fatalf("ListKindDefinitions(bootstrap) error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "go-service",
		DisplayName: "Go Service",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToProject},
		Template: domain.KindTemplate{
			ProjectMetadataDefaults: &domain.ProjectMetadata{
				Owner:             "platform",
				Tags:              []string{"go", "service"},
				StandardsMarkdown: "follow go test and qa defaults",
			},
			AutoCreateChildren: []domain.KindTemplateChildSpec{{
				Title:       "Main Branch",
				Description: "default implementation branch",
				Kind:        domain.KindID("branch"),
				AppliesTo:   domain.KindAppliesToBranch,
				Labels:      []string{"templated"},
			}},
		},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(go-service) error = %v", err)
	}

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name: "Go API",
		Kind: "go-service",
		Metadata: domain.ProjectMetadata{
			Tags: []string{"api"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if project.Metadata.Owner != "platform" {
		t.Fatalf("project owner = %q, want platform", project.Metadata.Owner)
	}
	if len(project.Metadata.Tags) != 3 || project.Metadata.Tags[0] != "api" || project.Metadata.Tags[1] != "go" || project.Metadata.Tags[2] != "service" {
		t.Fatalf("unexpected project tags %#v", project.Metadata.Tags)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected template root column creation")
	}
	tasks, err := svc.ListActionItems(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one template-created root actionItem, got %d", len(tasks))
	}
	if tasks[0].Kind != domain.Kind("branch") || tasks[0].Scope != domain.KindAppliesToBranch {
		t.Fatalf("unexpected root actionItem kind/scope %#v", tasks[0])
	}
}

// TestCreateActionItemCascadesChildKindTemplateDefaults verifies child auto-create goes back through create-time defaults.
func TestCreateActionItemCascadesChildKindTemplateDefaults(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 11, 30, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	if _, err := svc.ListKindDefinitions(context.Background(), false); err != nil {
		t.Fatalf("ListKindDefinitions(bootstrap) error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "qa-check",
		DisplayName: "QA Check",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToSubtask},
		Template: domain.KindTemplate{
			ActionItemMetadataDefaults: &domain.ActionItemMetadata{
				AcceptanceCriteria: "qa verifies the change",
				CompletionContract: domain.CompletionContract{
					CompletionChecklist: []domain.ChecklistItem{{ID: "qa-green", Text: "qa evidence attached"}},
				},
			},
		},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(qa-check) error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "implementation",
		DisplayName: "Implementation",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToActionItem},
		Template: domain.KindTemplate{
			ActionItemMetadataDefaults: &domain.ActionItemMetadata{
				ValidationPlan: "run package tests before handoff",
				CompletionContract: domain.CompletionContract{
					Policy: domain.CompletionPolicy{RequireChildrenDone: true},
				},
			},
			AutoCreateChildren: []domain.KindTemplateChildSpec{{
				Title:       "QA Check",
				Description: "verify implementation",
				Kind:        "qa-check",
				AppliesTo:   domain.KindAppliesToSubtask,
			}},
		},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(implementation) error = %v", err)
	}

	project, err := svc.CreateProject(context.Background(), "Cascade", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	parent, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "Implement auth flow",
		Kind:      "implementation",
		Scope:     domain.KindAppliesToActionItem,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(implementation) error = %v", err)
	}
	storedParent, err := repo.GetActionItem(context.Background(), parent.ID)
	if err != nil {
		t.Fatalf("GetActionItem(parent) error = %v", err)
	}
	if storedParent.Metadata.ValidationPlan != "run package tests before handoff" {
		t.Fatalf("ValidationPlan = %q, want template default", storedParent.Metadata.ValidationPlan)
	}
	if !storedParent.Metadata.CompletionContract.Policy.RequireChildrenDone {
		t.Fatal("expected require_children_done default on parent")
	}

	tasks, err := svc.ListActionItems(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	var child *domain.ActionItem
	for i := range tasks {
		if tasks[i].ParentID == parent.ID {
			child = &tasks[i]
			break
		}
	}
	if child == nil {
		t.Fatal("expected auto-created child actionItem")
	}
	if child.Kind != domain.Kind("qa-check") {
		t.Fatalf("child kind = %q, want qa-check", child.Kind)
	}
	if child.Metadata.AcceptanceCriteria != "qa verifies the change" {
		t.Fatalf("AcceptanceCriteria = %q, want cascaded child default", child.Metadata.AcceptanceCriteria)
	}
	if len(child.Metadata.CompletionContract.CompletionChecklist) != 1 {
		t.Fatalf("unexpected child completion checklist %#v", child.Metadata.CompletionContract.CompletionChecklist)
	}
}

// TestCreateActionItemRejectsExternalSystemBypass verifies public callers cannot fake the internal template path.
func TestCreateActionItemRejectsExternalSystemBypass(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		DefaultDeleteMode: DeleteModeArchive,
		RequireAgentLease: boolPtr(true),
	})

	project, err := svc.CreateProject(context.Background(), "Guarded", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if _, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:     project.ID,
		ColumnID:      column.ID,
		Title:         "Illicit system create",
		UpdatedByType: domain.ActorTypeSystem,
	}); !errors.Is(err, domain.ErrMutationLeaseRequired) {
		t.Fatalf("CreateActionItem(system without internal marker) error = %v, want ErrMutationLeaseRequired", err)
	}
}

// TestCreateActionItemRejectsRecursiveTemplateBeforePersistence verifies recursive templates fail closed.
func TestCreateActionItemRejectsRecursiveTemplateBeforePersistence(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 12, 10, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	if _, err := svc.ListKindDefinitions(context.Background(), false); err != nil {
		t.Fatalf("ListKindDefinitions(bootstrap) error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "loop",
		DisplayName: "Loop",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToActionItem, domain.KindAppliesToSubtask},
		Template: domain.KindTemplate{
			AutoCreateChildren: []domain.KindTemplateChildSpec{{
				Title:     "Loop Child",
				Kind:      "loop",
				AppliesTo: domain.KindAppliesToSubtask,
			}},
		},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(loop) error = %v", err)
	}

	project, err := svc.CreateProject(context.Background(), "Loop Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if _, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "Root",
		Kind:      "loop",
		Scope:     domain.KindAppliesToActionItem,
	}); !errors.Is(err, domain.ErrInvalidKindTemplate) {
		t.Fatalf("CreateActionItem(loop) error = %v, want ErrInvalidKindTemplate", err)
	}
	tasks, err := svc.ListActionItems(context.Background(), project.ID, true)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected no persisted tasks on recursive template failure, got %d", len(tasks))
	}
}

// TestIssueCapabilityLeaseParentDelegationPolicy verifies bounded parent-child delegation by role and scope.
func TestIssueCapabilityLeaseParentDelegationPolicy(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Delegation", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	branch, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Kind:      domain.Kind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Title:     "Branch A",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(branch) error = %v", err)
	}
	actionItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID: project.ID,
		ParentID:  branch.ID,
		ColumnID:  column.ID,
		Kind:      domain.KindActionItem,
		Scope:     domain.KindAppliesToActionItem,
		Title:     "ActionItem A",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(actionItem) error = %v", err)
	}

	parent, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleOrchestrator,
		AgentName:       "orch-1",
		AgentInstanceID: "orch-1",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(parent orchestrator) error = %v", err)
	}
	if got := parent.ScopeID; got != project.ID {
		t.Fatalf("parent ScopeID = %q, want normalized project id %q", got, project.ID)
	}
	child, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:        project.ID,
		ScopeType:        domain.CapabilityScopeBranch,
		ScopeID:          branch.ID,
		Role:             domain.CapabilityRoleBuilder,
		AgentName:        "builder-1",
		AgentInstanceID:  "builder-1",
		ParentInstanceID: parent.InstanceID,
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(child builder) error = %v", err)
	}
	if got := child.ParentInstanceID; got != parent.InstanceID {
		t.Fatalf("child ParentInstanceID = %q, want %q", got, parent.InstanceID)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:        project.ID,
		ScopeType:        domain.CapabilityScopeProject,
		ScopeID:          project.ID,
		Role:             domain.CapabilityRoleBuilder,
		AgentName:        "builder-project",
		AgentInstanceID:  "builder-project",
		ParentInstanceID: parent.InstanceID,
	}); !errors.Is(err, domain.ErrInvalidCapabilityDelegation) {
		t.Fatalf("IssueCapabilityLease(equal scope child) error = %v, want ErrInvalidCapabilityDelegation", err)
	}

	parentAllowed, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:                 project.ID,
		ScopeType:                 domain.CapabilityScopeBranch,
		ScopeID:                   branch.ID,
		Role:                      domain.CapabilityRoleOrchestrator,
		AgentName:                 "orch-allowed",
		AgentInstanceID:           "orch-allowed",
		AllowEqualScopeDelegation: true,
		OverrideToken:             "override-equal",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(parent allowed) error = %v", err)
	}
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:        project.ID,
		ScopeType:        domain.CapabilityScopeBranch,
		ScopeID:          branch.ID,
		Role:             domain.CapabilityRoleBuilder,
		AgentName:        "builder-branch-allowed",
		AgentInstanceID:  "builder-branch-allowed",
		ParentInstanceID: parentAllowed.InstanceID,
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(equal scope allowed) error = %v", err)
	}
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:        project.ID,
		ScopeType:        domain.CapabilityScopeActionItem,
		ScopeID:          actionItem.ID,
		Role:             domain.CapabilityRoleResearch,
		AgentName:        "research-child",
		AgentInstanceID:  "research-child",
		ParentInstanceID: parent.InstanceID,
	}); err != nil {
		t.Fatalf("IssueCapabilityLease(research child) error = %v", err)
	}

	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:        project.ID,
		ScopeType:        domain.CapabilityScopeActionItem,
		ScopeID:          actionItem.ID,
		Role:             domain.CapabilityRoleOrchestrator,
		AgentName:        "child-orch",
		AgentInstanceID:  "child-orch",
		ParentInstanceID: parent.InstanceID,
	}); !errors.Is(err, domain.ErrInvalidCapabilityDelegation) {
		t.Fatalf("IssueCapabilityLease(orchestrator child) error = %v, want ErrInvalidCapabilityDelegation", err)
	}

	builderParent, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeBranch,
		ScopeID:         branch.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "builder-parent",
		AgentInstanceID: "builder-parent",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(builder parent) error = %v", err)
	}
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:        project.ID,
		ScopeType:        domain.CapabilityScopeActionItem,
		ScopeID:          actionItem.ID,
		Role:             domain.CapabilityRoleQA,
		AgentName:        "qa-child",
		AgentInstanceID:  "qa-child",
		ParentInstanceID: builderParent.InstanceID,
	}); !errors.Is(err, domain.ErrInvalidCapabilityDelegation) {
		t.Fatalf("IssueCapabilityLease(builder parent child) error = %v, want ErrInvalidCapabilityDelegation", err)
	}
	if _, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleSystem,
		AgentName:       "system-1",
		AgentInstanceID: "system-1",
	}); !errors.Is(err, domain.ErrInvalidCapabilityRole) {
		t.Fatalf("IssueCapabilityLease(system) error = %v, want ErrInvalidCapabilityRole", err)
	}
}

// TestQALeaseActionPolicy verifies qa leases may comment and edit scoped nodes before template contracts narrow them.
func TestQALeaseActionPolicy(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 3, 21, 11, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		DefaultDeleteMode:  DeleteModeArchive,
		RequireAgentLease:  boolPtr(true),
		CapabilityLeaseTTL: time.Hour,
	})

	project, err := svc.CreateProject(context.Background(), "QA Policy", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	actionItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID: project.ID,
		ColumnID:  column.ID,
		Title:     "ActionItem A",
		Priority:  domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}
	qaLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		Role:            domain.CapabilityRoleQA,
		AgentName:       "qa-1",
		AgentInstanceID: "qa-1",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(qa) error = %v", err)
	}
	qaCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       qaLease.AgentName,
		AgentInstanceID: qaLease.InstanceID,
		LeaseToken:      qaLease.LeaseToken,
	})
	if _, err := svc.CreateComment(qaCtx, CreateCommentInput{
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeActionItem,
		TargetID:     actionItem.ID,
		BodyMarkdown: "qa note",
		ActorID:      "qa-1",
		ActorType:    domain.ActorTypeAgent,
	}); err != nil {
		t.Fatalf("CreateComment(qa) error = %v", err)
	}
	if _, err := svc.UpdateActionItem(qaCtx, UpdateActionItemInput{
		ActionItemID: actionItem.ID,
		Title:        "ActionItem A QA",
		Description:  "qa-edited",
		Priority:     domain.PriorityMedium,
		UpdatedBy:    "qa-1",
		UpdatedType:  domain.ActorTypeAgent,
	}); err != nil {
		t.Fatalf("UpdateActionItem(qa) error = %v", err)
	}
}

// TestKindCapabilityHelpers verifies deterministic helper behavior used by service methods.
func TestKindCapabilityHelpers(t *testing.T) {
	normalized := normalizeKindIDList([]domain.KindID{"ActionItem", "phase", "actionItem", "  ", "Phase"})
	wantIDs := []domain.KindID{"actionItem", "phase"}
	if !slices.Equal(normalized, wantIDs) {
		t.Fatalf("normalizeKindIDList() = %#v, want %#v", normalized, wantIDs)
	}

	hashA := hashSchema(`{"type":"object"}`)
	hashB := hashSchema(`{"type":"object"}`)
	hashC := hashSchema(`{"type":"string"}`)
	if hashA != hashB {
		t.Fatalf("hashSchema() expected deterministic hash, got %q vs %q", hashA, hashB)
	}
	if hashA == hashC {
		t.Fatalf("hashSchema() expected different hash for different schema, got %q", hashA)
	}

	existing := []domain.ChecklistItem{{ID: "a", Text: "existing"}}
	incoming := []domain.ChecklistItem{{ID: "a", Text: "duplicate"}, {ID: "b", Text: "new"}, {ID: "", Text: "skip"}}
	merged := mergeChecklistItems(existing, incoming)
	if len(merged) != 2 {
		t.Fatalf("mergeChecklistItems() len = %d, want 2", len(merged))
	}

	if _, err := normalizeActionItemMetadataFromKindPayload(json.RawMessage(`{`)); !errors.Is(err, domain.ErrInvalidKindPayload) {
		t.Fatalf("normalizeActionItemMetadataFromKindPayload(invalid) error = %v, want ErrInvalidKindPayload", err)
	}
	meta, err := normalizeActionItemMetadataFromKindPayload(json.RawMessage(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("normalizeActionItemMetadataFromKindPayload(valid) error = %v", err)
	}
	if string(meta.KindPayload) != `{"key":"value"}` {
		t.Fatalf("KindPayload = %s, want {\"key\":\"value\"}", string(meta.KindPayload))
	}
}
