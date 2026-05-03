package app

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
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
		KindIDs:   []domain.KindID{"plan", "build", "plan"},
	}); err != nil {
		t.Fatalf("SetProjectAllowedKinds(valid) error = %v", err)
	}
	kinds, err := svc.ListProjectAllowedKinds(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds() error = %v", err)
	}
	want := []domain.KindID{"build", "plan"}
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
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToPlan},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(create) error = %v", err)
	}
	updated, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "zeta",
		DisplayName: "Alpha",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToPlan},
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

// templateRejectionTestCase parameterizes one reverse-hierarchy prohibition
// covered by Drop 3 droplet 3.16's e2e contract. Each row spells out the
// parent kind, the child kind, and the catalog rule fragment that produces
// the rejection (mirroring the four prohibitions PLAN.md § 19.3 line 1638
// enumerates: closeout-no-closeout-parent, commit-no-plan-child,
// human-verify-no-build-child, build-qa-*-no-plan-child).
type templateRejectionTestCase struct {
	name        string
	parentKind  domain.Kind
	childKind   domain.Kind
	wantReason  string // substring assertion; AllowsNesting prepends `kind %q ...`
	commentBody string // substring assertion; recordTemplateRejectionAudit body
}

// templateRejectionDefaultCatalog bakes the four reverse-hierarchy
// prohibitions onto a project so droplet 3.16 e2e tests can exercise the
// rejection path without depending on the embedded default.toml file load
// order. The Kinds map mirrors the prohibitions encoded in
// internal/templates/builtin/default.toml — every kind enumerated in the
// closeout / commit / human-verify / build-qa-* allowed_child_kinds rows
// EXCEPT the prohibited child.
func templateRejectionDefaultCatalog() templates.KindCatalog {
	allKindsExcept := func(excluded domain.Kind) []domain.Kind {
		all := []domain.Kind{
			domain.KindPlan,
			domain.KindResearch,
			domain.KindBuild,
			domain.KindPlanQAProof,
			domain.KindPlanQAFalsification,
			domain.KindBuildQAProof,
			domain.KindBuildQAFalsification,
			domain.KindCloseout,
			domain.KindCommit,
			domain.KindRefinement,
			domain.KindDiscussion,
			domain.KindHumanVerify,
		}
		out := make([]domain.Kind, 0, len(all)-1)
		for _, k := range all {
			if k != excluded {
				out = append(out, k)
			}
		}
		return out
	}

	tpl := templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Kinds: map[domain.Kind]templates.KindRule{
			domain.KindPlan: {
				StructuralType: domain.StructuralTypeDroplet,
			},
			domain.KindBuild: {
				StructuralType: domain.StructuralTypeDroplet,
			},
			domain.KindCloseout: {
				Owner:             "STEWARD",
				AllowedChildKinds: allKindsExcept(domain.KindCloseout),
				StructuralType:    domain.StructuralTypeDroplet,
			},
			domain.KindCommit: {
				AllowedChildKinds: allKindsExcept(domain.KindPlan),
				StructuralType:    domain.StructuralTypeDroplet,
			},
			domain.KindHumanVerify: {
				AllowedChildKinds: allKindsExcept(domain.KindBuild),
				StructuralType:    domain.StructuralTypeDroplet,
			},
			domain.KindBuildQAProof: {
				AllowedChildKinds: allKindsExcept(domain.KindPlan),
				StructuralType:    domain.StructuralTypeDroplet,
			},
			domain.KindBuildQAFalsification: {
				AllowedChildKinds: allKindsExcept(domain.KindPlan),
				StructuralType:    domain.StructuralTypeDroplet,
			},
		},
	}
	return templates.Bake(tpl)
}

// TestCreateActionItemTemplateRejectionAuditTrail covers droplet 3.16's
// acceptance contract: every Template.AllowsNesting → false rejection at
// the auth-gated CreateActionItem boundary writes a till.comment on the
// parent + an attention_item with kind = template_rejection. Coverage
// hits all four reverse-hierarchy prohibitions PLAN.md § 19.3 line 1638
// names: closeout-no-closeout-parent, commit-no-plan-child,
// human-verify-no-build-child, build-qa-*-no-plan-child.
func TestCreateActionItemTemplateRejectionAuditTrail(t *testing.T) {
	cases := []templateRejectionTestCase{
		{
			name:        "closeout cannot parent another closeout",
			parentKind:  domain.KindCloseout,
			childKind:   domain.KindCloseout,
			wantReason:  "not in allowed_child_kinds",
			commentBody: "Parent kind: `closeout`",
		},
		{
			name:        "commit cannot parent a plan",
			parentKind:  domain.KindCommit,
			childKind:   domain.KindPlan,
			wantReason:  "not in allowed_child_kinds",
			commentBody: "Parent kind: `commit`",
		},
		{
			name:        "human-verify cannot parent a build",
			parentKind:  domain.KindHumanVerify,
			childKind:   domain.KindBuild,
			wantReason:  "not in allowed_child_kinds",
			commentBody: "Parent kind: `human-verify`",
		},
		{
			name:        "build-qa-falsification cannot parent a plan",
			parentKind:  domain.KindBuildQAFalsification,
			childKind:   domain.KindPlan,
			wantReason:  "not in allowed_child_kinds",
			commentBody: "Parent kind: `build-qa-falsification`",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
			svc := newDeterministicService(repo, now, ServiceConfig{})

			project, err := svc.CreateProject(context.Background(), "Template Rejection", "")
			if err != nil {
				t.Fatalf("CreateProject() error = %v", err)
			}

			// Bake the four-prohibition catalog onto the project so the
			// nesting check observes the same rules as
			// internal/templates/builtin/default.toml.
			catalog := templateRejectionDefaultCatalog()
			encoded, err := json.Marshal(catalog)
			if err != nil {
				t.Fatalf("json.Marshal(catalog) error = %v", err)
			}
			project.KindCatalogJSON = encoded
			if err := repo.UpdateProject(context.Background(), project); err != nil {
				t.Fatalf("UpdateProject() error = %v", err)
			}

			column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
			if err != nil {
				t.Fatalf("CreateColumn() error = %v", err)
			}

			// Create the parent at top level (no grandparent so the parent
			// itself bypasses the nesting check). The catalog still has a
			// matching kind for the parent so the resolver's project-
			// allowlist + AppliesToScope gates pass.
			parent, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
				ProjectID:      project.ID,
				ColumnID:       column.ID,
				Kind:           tc.parentKind,
				Title:          "PARENT-" + string(tc.parentKind),
				Priority:       domain.PriorityMedium,
				StructuralType: domain.StructuralTypeDroplet,
				CreatedByActor: "user-1",
				UpdatedByActor: "user-1",
				UpdatedByType:  domain.ActorTypeUser,
			})
			if err != nil {
				t.Fatalf("CreateActionItem(parent %q) error = %v", tc.parentKind, err)
			}

			// Now create the prohibited child — must fail with
			// ErrKindNotAllowed wrapping the catalog's rejection reason.
			_, err = svc.CreateActionItem(context.Background(), CreateActionItemInput{
				ProjectID:      project.ID,
				ParentID:       parent.ID,
				ColumnID:       column.ID,
				Kind:           tc.childKind,
				Title:          "CHILD-" + string(tc.childKind),
				Priority:       domain.PriorityMedium,
				StructuralType: domain.StructuralTypeDroplet,
				CreatedByActor: "user-1",
				UpdatedByActor: "user-1",
				UpdatedByType:  domain.ActorTypeUser,
			})
			if !errors.Is(err, domain.ErrKindNotAllowed) {
				t.Fatalf("CreateActionItem(child %q under %q) error = %v, want ErrKindNotAllowed", tc.childKind, tc.parentKind, err)
			}
			if !strings.Contains(err.Error(), tc.wantReason) {
				t.Fatalf("CreateActionItem error = %q, want substring %q", err.Error(), tc.wantReason)
			}

			// Assert audit-trail comment was written on the parent.
			comments, err := repo.ListCommentsByTarget(context.Background(), domain.CommentTarget{
				ProjectID:  project.ID,
				TargetType: domain.CommentTargetTypeActionItem,
				TargetID:   parent.ID,
			})
			if err != nil {
				t.Fatalf("ListCommentsByTarget() error = %v", err)
			}
			if len(comments) != 1 {
				t.Fatalf("ListCommentsByTarget() count = %d, want 1", len(comments))
			}
			comment := comments[0]
			if !strings.Contains(comment.BodyMarkdown, tc.commentBody) {
				t.Fatalf("comment body = %q, want substring %q", comment.BodyMarkdown, tc.commentBody)
			}
			if !strings.Contains(comment.BodyMarkdown, tc.wantReason) {
				t.Fatalf("comment body = %q, want rejection reason substring %q", comment.BodyMarkdown, tc.wantReason)
			}
			if comment.ActorType != domain.ActorTypeUser {
				t.Fatalf("comment ActorType = %q, want %q", comment.ActorType, domain.ActorTypeUser)
			}

			// Assert attention item with kind = template_rejection was
			// written scoped to the parent action item.
			items, err := repo.ListAttentionItems(context.Background(), domain.AttentionListFilter{
				ProjectID: project.ID,
				ScopeType: domain.ScopeLevelActionItem,
				ScopeID:   parent.ID,
				Kinds:     []domain.AttentionKind{domain.AttentionKindTemplateRejection},
			})
			if err != nil {
				t.Fatalf("ListAttentionItems() error = %v", err)
			}
			if len(items) != 1 {
				t.Fatalf("ListAttentionItems(template_rejection) count = %d, want 1", len(items))
			}
			item := items[0]
			if item.Kind != domain.AttentionKindTemplateRejection {
				t.Fatalf("attention Kind = %q, want %q", item.Kind, domain.AttentionKindTemplateRejection)
			}
			if !strings.Contains(item.BodyMarkdown, tc.wantReason) {
				t.Fatalf("attention body = %q, want rejection reason substring %q", item.BodyMarkdown, tc.wantReason)
			}
			if !item.RequiresUserAction {
				t.Fatal("attention RequiresUserAction = false, want true")
			}
		})
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
	// The legacy "project root rows disguised as action-item scope" guard is
	// gone with scope-mirrors-kind: every action_items row is
	// ScopeLevelActionItem now, so no tuple can slip through on a mismatched
	// scope coercion.
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

	// Under scope-mirrors-kind, every action-item row is ScopeLevelActionItem
	// (CapabilityScopeActionItem). Exercise the scope-match vs scope-mismatch
	// lease-guard paths using a plan-kind action item against an
	// action-item-scoped lease.
	planItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "Plan A",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(plan) error = %v", err)
	}
	planLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeActionItem,
		ScopeID:         planItem.ID,
		Role:            domain.CapabilityRoleBuilder,
		AgentName:       "agent-plan",
		AgentInstanceID: "agent-plan-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(plan) error = %v", err)
	}
	planGuard := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       planLease.AgentName,
		AgentInstanceID: planLease.InstanceID,
		LeaseToken:      planLease.LeaseToken,
	})
	if err := svc.enforceMutationGuard(planGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID, domain.CapabilityActionEditNode); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(scope mismatch) error = %v, want ErrMutationLeaseInvalid", err)
	}
	if err := svc.enforceMutationGuard(planGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeActionItem, planItem.ID, domain.CapabilityActionEditNode); err != nil {
		t.Fatalf("enforceMutationGuard(scope match) error = %v", err)
	}
	storedPlan, err := repo.GetCapabilityLease(context.Background(), planLease.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease(plan) error = %v", err)
	}
	if storedPlan.HeartbeatAt.IsZero() {
		t.Fatal("enforceMutationGuard(scope match) expected heartbeat update")
	}
}

// Note: Drop 3 droplet 3.15 deleted the legacy KindTemplate surface
// (AutoCreateChildren, ProjectMetadataDefaults, ActionItemMetadataDefaults,
// CompletionChecklist) along with mergeActionItemMetadataWithKindTemplate's
// merge behavior — the function is now a pass-through. The previously
// surviving "merges CompletionChecklist" test
// (TestCreateActionItemKindMergesCompletionChecklist) was retired in this
// droplet because the merge it covered no longer exists; templates v1's
// KindRule does not encode action-item metadata defaults. Future template-
// driven defaults will be reintroduced through a different mechanism.

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
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Illicit system create",
		UpdatedByType:  domain.ActorTypeSystem,
		StructuralType: domain.StructuralTypeDroplet,
	}); !errors.Is(err, domain.ErrMutationLeaseRequired) {
		t.Fatalf("CreateActionItem(system without internal marker) error = %v, want ErrMutationLeaseRequired", err)
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
	// Under scope-mirrors-kind, every action-item row lives at
	// CapabilityScopeActionItem. Exercise the delegation policy using a
	// project-scoped orchestrator parent delegating to an action-item-scoped
	// child, reflecting the only non-equal-scope path still reachable.
	planItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		Title:          "Plan A",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(plan) error = %v", err)
	}
	actionItem, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       planItem.ID,
		ColumnID:       column.ID,
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		Title:          "Build A",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(build) error = %v", err)
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
		ScopeType:        domain.CapabilityScopeActionItem,
		ScopeID:          planItem.ID,
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
		ScopeType:                 domain.CapabilityScopeActionItem,
		ScopeID:                   planItem.ID,
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
		ScopeType:        domain.CapabilityScopeActionItem,
		ScopeID:          planItem.ID,
		Role:             domain.CapabilityRoleBuilder,
		AgentName:        "builder-equal-allowed",
		AgentInstanceID:  "builder-equal-allowed",
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
		ScopeType:       domain.CapabilityScopeActionItem,
		ScopeID:         planItem.ID,
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
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "ActionItem A",
		Priority:       domain.PriorityMedium,
		StructuralType: domain.StructuralTypeDroplet,
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
	// NormalizeKindID now trims + lowercases (no camelCase rewriting).
	normalized := normalizeKindIDList([]domain.KindID{"Plan", "build", "plan", "  ", "Build"})
	wantIDs := []domain.KindID{"build", "plan"}
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
