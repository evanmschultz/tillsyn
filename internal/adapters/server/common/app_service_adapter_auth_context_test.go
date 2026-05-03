package common

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/adapters/auth/autentauth"
	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// authScopeFixture stores one real auth-backed hierarchy fixture for adapter auth tests.
type authScopeFixture struct {
	adapter      *AppServiceAdapter
	auth         *autentauth.Service
	projectID    string
	approvedPath string
	actionItemA  domain.ActionItem
	actionItemB  domain.ActionItem
	handoffA     domain.Handoff
	handoffB     domain.Handoff
	attentionA   domain.AttentionItem
	attentionB   domain.AttentionItem
	leaseA       domain.CapabilityLease
	leaseB       domain.CapabilityLease
}

// newAuthScopeFixtureForTest constructs one real auth-backed scope fixture with in-scope and out-of-scope resources.
func newAuthScopeFixtureForTest(t *testing.T) authScopeFixture {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	requireAgentLease := false
	nextID := 0
	svc := app.NewService(repo, func() string {
		nextID++
		return fmt.Sprintf("id-%03d", nextID)
	}, nil, app.ServiceConfig{
		AutoCreateProjectColumns: true,
		RequireAgentLease:        &requireAgentLease,
	})
	seedOrphanKindsForTest(t, svc)

	project, err := svc.CreateProject(context.Background(), "Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	columns, err := repo.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("ListColumns() returned no columns, want default project columns")
	}
	columnID := columns[0].ID

	// Under scope-mirrors-kind, every action-item row is
	// ScopeLevelActionItem. The auth path for any per-row auth fixture lives
	// at project/<projectID> because no ancestor can anchor a deeper scope.
	// The two action items here stand in for the "in-scope A vs out-of-scope
	// B" pair the fixture models.
	_ = columnID
	planA := mustCreateActionItemForTest(t, svc, app.CreateActionItemInput{
		ProjectID:      project.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		ColumnID:       columnID,
		Title:          "Plan A",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	actionItemA := mustCreateActionItemForTest(t, svc, app.CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       planA.ID,
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		ColumnID:       columnID,
		Title:          "Build A",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	planB := mustCreateActionItemForTest(t, svc, app.CreateActionItemInput{
		ProjectID:      project.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		ColumnID:       columnID,
		Title:          "Plan B",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	actionItemB := mustCreateActionItemForTest(t, svc, app.CreateActionItemInput{
		ProjectID:      project.ID,
		ParentID:       planB.ID,
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		ColumnID:       columnID,
		Title:          "Build B",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})

	handoffA, err := svc.CreateHandoff(context.Background(), app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   actionItemA.ID,
		},
		Summary:     "handoff a",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff(actionItem A) error = %v", err)
	}
	handoffB, err := svc.CreateHandoff(context.Background(), app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   actionItemB.ID,
		},
		Summary:     "handoff b",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff(actionItem B) error = %v", err)
	}

	attentionA, err := svc.RaiseAttentionItem(context.Background(), app.RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   actionItemA.ID,
		},
		Kind:        domain.AttentionKindRiskNote,
		Summary:     "attention a",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem(actionItem A) error = %v", err)
	}
	attentionB, err := svc.RaiseAttentionItem(context.Background(), app.RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   actionItemB.ID,
		},
		Kind:        domain.AttentionKindRiskNote,
		Summary:     "attention b",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem(actionItem B) error = %v", err)
	}

	leaseA, err := svc.IssueCapabilityLease(context.Background(), app.IssueCapabilityLeaseInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeActionItem,
		ScopeID:   actionItemA.ID,
		Role:      domain.CapabilityRoleBuilder,
		AgentName: "Builder A",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(actionItem A) error = %v", err)
	}
	leaseB, err := svc.IssueCapabilityLease(context.Background(), app.IssueCapabilityLeaseInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeActionItem,
		ScopeID:   actionItemB.ID,
		Role:      domain.CapabilityRoleBuilder,
		AgentName: "Builder B",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(actionItem B) error = %v", err)
	}

	return authScopeFixture{
		adapter:   NewAppServiceAdapter(svc, auth),
		auth:      auth,
		projectID: project.ID,
		// Every action-item row maps to ScopeLevelActionItem post-Drop-1.75;
		// the approved path that used to narrow to a branch/phase now stays
		// at project-level and covers both action items in the fixture.
		approvedPath: "project/" + project.ID,
		actionItemA:  actionItemA,
		actionItemB:  actionItemB,
		handoffA:     handoffA,
		handoffB:     handoffB,
		attentionA:   attentionA,
		attentionB:   attentionB,
		leaseA:       leaseA,
		leaseB:       leaseB,
	}
}

// mustCreateActionItemForTest creates one fixture work item or fails the test.
// Empty StructuralType is defaulted to domain.StructuralTypeDroplet so legacy
// fixture rows (which predate droplet 3.4's required-on-create rule) keep
// round-tripping through the adapter chain. Tests that need a specific
// structural type still set it explicitly.
func mustCreateActionItemForTest(t *testing.T, svc *app.Service, in app.CreateActionItemInput) domain.ActionItem {
	t.Helper()

	if strings.TrimSpace(string(in.StructuralType)) == "" {
		in.StructuralType = domain.StructuralTypeDroplet
	}
	actionItem, err := svc.CreateActionItem(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateActionItem(%q) error = %v", in.Title, err)
	}
	return actionItem
}

// mustIssueApprovedPathSessionForTest issues one deterministic session carrying approved-path metadata.
func mustIssueApprovedPathSessionForTest(t *testing.T, auth *autentauth.Service, approvedPath string) (string, string) {
	t.Helper()

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
		Metadata: map[string]string{
			"approved_path": approvedPath,
			"project_id":    "",
		},
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	return issued.Session.ID, issued.Secret
}

// TestAppServiceAdapterAuthorizeMutationApprovedPathLookupBackedResources verifies by-id mutation auth derives project-rooted scope from persisted resources.
func TestAppServiceAdapterAuthorizeMutationApprovedPathLookupBackedResources(t *testing.T) {
	t.Parallel()

	fixture := newAuthScopeFixtureForTest(t)
	sessionID, sessionSecret := mustIssueApprovedPathSessionForTest(t, fixture.auth, fixture.approvedPath)

	cases := []struct {
		name    string
		req     MutationAuthorizationRequest
		wantErr error
	}{
		{
			name: "update actionItem in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "update_task",
				Namespace:     "tillsyn",
				ResourceType:  "actionItem",
				ResourceID:    fixture.actionItemA.ID,
				Context:       map[string]string{"action_item_id": fixture.actionItemA.ID},
			},
		},
		{
			// Post-Drop-1.75 approved-path narrowing cannot descend below
			// project because action-item rows are all ScopeLevelActionItem.
			// The fixture covers both A and B via the project-level path, so
			// the previously-denied "out of scope" case now authorizes.
			name: "update actionItem B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "update_task",
				Namespace:     "tillsyn",
				ResourceType:  "actionItem",
				ResourceID:    fixture.actionItemB.ID,
				Context:       map[string]string{"action_item_id": fixture.actionItemB.ID},
			},
		},
		{
			name: "move actionItem in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "move_task",
				Namespace:     "tillsyn",
				ResourceType:  "actionItem",
				ResourceID:    fixture.actionItemA.ID,
				Context:       map[string]string{"action_item_id": fixture.actionItemA.ID},
			},
		},
		{
			name: "delete actionItem B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "delete_task",
				Namespace:     "tillsyn",
				ResourceType:  "actionItem",
				ResourceID:    fixture.actionItemB.ID,
				Context:       map[string]string{"action_item_id": fixture.actionItemB.ID},
			},
		},
		{
			name: "restore actionItem in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "restore_task",
				Namespace:     "tillsyn",
				ResourceType:  "actionItem",
				ResourceID:    fixture.actionItemA.ID,
				Context:       map[string]string{"action_item_id": fixture.actionItemA.ID},
			},
		},
		{
			name: "reparent actionItem B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "reparent_task",
				Namespace:     "tillsyn",
				ResourceType:  "actionItem",
				ResourceID:    fixture.actionItemB.ID,
				Context: map[string]string{
					"action_item_id": fixture.actionItemB.ID,
					"parent_id":      fixture.actionItemB.ID,
				},
			},
		},
		{
			name: "update handoff in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "update_handoff",
				Namespace:     "tillsyn",
				ResourceType:  "handoff",
				ResourceID:    fixture.handoffA.ID,
				Context:       map[string]string{"handoff_id": fixture.handoffA.ID},
			},
		},
		{
			name: "update handoff B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "update_handoff",
				Namespace:     "tillsyn",
				ResourceType:  "handoff",
				ResourceID:    fixture.handoffB.ID,
				Context:       map[string]string{"handoff_id": fixture.handoffB.ID},
			},
		},
		{
			name: "resolve attention in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "resolve_attention_item",
				Namespace:     "tillsyn",
				ResourceType:  "attention_item",
				ResourceID:    fixture.attentionA.ID,
				Context:       map[string]string{"attention_id": fixture.attentionA.ID},
			},
		},
		{
			name: "resolve attention B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "resolve_attention_item",
				Namespace:     "tillsyn",
				ResourceType:  "attention_item",
				ResourceID:    fixture.attentionB.ID,
				Context:       map[string]string{"attention_id": fixture.attentionB.ID},
			},
		},
		{
			name: "renew lease in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "renew_capability_lease",
				Namespace:     "tillsyn",
				ResourceType:  "capability_lease",
				ResourceID:    fixture.leaseA.InstanceID,
				Context:       map[string]string{"agent_instance_id": fixture.leaseA.InstanceID},
			},
		},
		{
			name: "renew lease B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "renew_capability_lease",
				Namespace:     "tillsyn",
				ResourceType:  "capability_lease",
				ResourceID:    fixture.leaseB.InstanceID,
				Context:       map[string]string{"agent_instance_id": fixture.leaseB.InstanceID},
			},
		},
		{
			name: "heartbeat lease in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "heartbeat_capability_lease",
				Namespace:     "tillsyn",
				ResourceType:  "capability_lease",
				ResourceID:    fixture.leaseA.InstanceID,
				Context:       map[string]string{"agent_instance_id": fixture.leaseA.InstanceID},
			},
		},
		{
			name: "revoke lease B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "revoke_capability_lease",
				Namespace:     "tillsyn",
				ResourceType:  "capability_lease",
				ResourceID:    fixture.leaseB.InstanceID,
				Context:       map[string]string{"agent_instance_id": fixture.leaseB.InstanceID},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caller, err := fixture.adapter.AuthorizeMutation(context.Background(), tc.req)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("AuthorizeMutation() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("AuthorizeMutation() error = %v", err)
			}
			if caller.PrincipalID != "user-1" {
				t.Fatalf("AuthorizeMutation() principal_id = %q, want user-1", caller.PrincipalID)
			}
		})
	}
}

// TestAppServiceAdapterAuthorizeMutationApprovedPathExplicitScopeResources verifies explicit-scope mutations are narrowed before auth evaluates approved_path.
func TestAppServiceAdapterAuthorizeMutationApprovedPathExplicitScopeResources(t *testing.T) {
	t.Parallel()

	fixture := newAuthScopeFixtureForTest(t)
	sessionID, sessionSecret := mustIssueApprovedPathSessionForTest(t, fixture.auth, fixture.approvedPath)

	cases := []struct {
		name    string
		req     MutationAuthorizationRequest
		wantErr error
	}{
		{
			name: "create actionItem under in-scope parent",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_task",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "actionItem",
				ResourceID:    "new",
				Context: map[string]string{
					"project_id": fixture.projectID,
					"parent_id":  fixture.actionItemA.ID,
				},
			},
		},
		{
			name: "create actionItem under out-of-scope parent",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_task",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "actionItem",
				ResourceID:    "new",
				Context: map[string]string{
					"project_id": fixture.projectID,
					"parent_id":  fixture.actionItemB.ID,
				},
			},
		},
		{
			name: "create comment on in-scope target",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_comment",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "comment",
				ResourceID:    fixture.actionItemA.ID,
				Context: map[string]string{
					"project_id":  fixture.projectID,
					"target_type": "actionItem",
				},
			},
		},
		{
			name: "create comment on actionItem B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_comment",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "comment",
				ResourceID:    fixture.actionItemB.ID,
				Context: map[string]string{
					"project_id":  fixture.projectID,
					"target_type": "actionItem",
				},
			},
		},
		{
			name: "create handoff in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_handoff",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "handoff",
				ResourceID:    fixture.actionItemA.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "actionItem",
				},
			},
		},
		{
			name: "create handoff on actionItem B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_handoff",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "handoff",
				ResourceID:    fixture.actionItemB.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "actionItem",
				},
			},
		},
		{
			name: "raise attention in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "raise_attention_item",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "attention_item",
				ResourceID:    fixture.actionItemA.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "actionItem",
				},
			},
		},
		{
			name: "raise attention on actionItem B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "raise_attention_item",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "attention_item",
				ResourceID:    fixture.actionItemB.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "actionItem",
				},
			},
		},
		{
			name: "issue capability lease in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "issue_capability_lease",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "capability_lease",
				ResourceID:    fixture.actionItemA.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "actionItem",
				},
			},
		},
		{
			name: "issue capability lease on actionItem B covered by project path",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "issue_capability_lease",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "capability_lease",
				ResourceID:    fixture.actionItemB.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "actionItem",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caller, err := fixture.adapter.AuthorizeMutation(context.Background(), tc.req)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("AuthorizeMutation() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("AuthorizeMutation() error = %v", err)
			}
			if caller.PrincipalID != "user-1" {
				t.Fatalf("AuthorizeMutation() principal_id = %q, want user-1", caller.PrincipalID)
			}
		})
	}
}

// TestAppServiceAdapterAuthorizeMutationApprovedPathPolicySplit verifies the
// locked split between global-admin mutations and project-scoped workflow
// mutations.
func TestAppServiceAdapterAuthorizeMutationApprovedPathPolicySplit(t *testing.T) {
	t.Parallel()

	fixture := newAuthScopeFixtureForTest(t)
	globalSessionID, globalSessionSecret := mustIssueApprovedPathSessionForTest(t, fixture.auth, "global")

	cases := []struct {
		name          string
		sessionID     string
		sessionSecret string
		req           MutationAuthorizationRequest
		wantErr       error
	}{
		{
			name:          "global approval may create project",
			sessionID:     globalSessionID,
			sessionSecret: globalSessionSecret,
			req: MutationAuthorizationRequest{
				Action:       "create_project",
				Namespace:    "project:" + domain.AuthRequestGlobalProjectID,
				ResourceType: "project",
				ResourceID:   "new",
				Context: map[string]string{
					"project_id": domain.AuthRequestGlobalProjectID,
					"scope_type": "project",
					"scope_id":   domain.AuthRequestGlobalProjectID,
				},
			},
		},
		{
			name:          "global approval may not issue in-project lease",
			sessionID:     globalSessionID,
			sessionSecret: globalSessionSecret,
			req: MutationAuthorizationRequest{
				Action:       "issue_capability_lease",
				Namespace:    "project:" + fixture.projectID,
				ResourceType: "capability_lease",
				ResourceID:   fixture.actionItemA.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "actionItem",
				},
			},
			wantErr: ErrAuthorizationDenied,
		},
		{
			name:          "global approval may not update project",
			sessionID:     globalSessionID,
			sessionSecret: globalSessionSecret,
			req: MutationAuthorizationRequest{
				Action:       "update_project",
				Namespace:    "tillsyn",
				ResourceType: "project",
				ResourceID:   fixture.projectID,
				Context: map[string]string{
					"project_id": fixture.projectID,
				},
			},
			wantErr: ErrAuthorizationDenied,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			req.SessionID = tc.sessionID
			req.SessionSecret = tc.sessionSecret
			caller, err := fixture.adapter.AuthorizeMutation(context.Background(), req)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("AuthorizeMutation() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("AuthorizeMutation() error = %v", err)
			}
			if caller.PrincipalID != "user-1" {
				t.Fatalf("AuthorizeMutation() principal_id = %q, want user-1", caller.PrincipalID)
			}
		})
	}
}
