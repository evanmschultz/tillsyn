package common

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
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
	taskA        domain.Task
	taskB        domain.Task
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

	branchA := mustCreateTaskForTest(t, svc, app.CreateTaskInput{
		ProjectID:      project.ID,
		Kind:           domain.WorkKind("branch"),
		Scope:          domain.KindAppliesToBranch,
		ColumnID:       columnID,
		Title:          "Branch A",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	phaseA := mustCreateTaskForTest(t, svc, app.CreateTaskInput{
		ProjectID:      project.ID,
		ParentID:       branchA.ID,
		Kind:           domain.WorkKindPhase,
		Scope:          domain.KindAppliesToPhase,
		ColumnID:       columnID,
		Title:          "Phase A",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	taskA := mustCreateTaskForTest(t, svc, app.CreateTaskInput{
		ProjectID:      project.ID,
		ParentID:       phaseA.ID,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		ColumnID:       columnID,
		Title:          "Task A",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	branchB := mustCreateTaskForTest(t, svc, app.CreateTaskInput{
		ProjectID:      project.ID,
		Kind:           domain.WorkKind("branch"),
		Scope:          domain.KindAppliesToBranch,
		ColumnID:       columnID,
		Title:          "Branch B",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})
	taskB := mustCreateTaskForTest(t, svc, app.CreateTaskInput{
		ProjectID:      project.ID,
		ParentID:       branchB.ID,
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		ColumnID:       columnID,
		Title:          "Task B",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
	})

	handoffA, err := svc.CreateHandoff(context.Background(), app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   taskA.ID,
		},
		Summary:     "handoff a",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff(task A) error = %v", err)
	}
	handoffB, err := svc.CreateHandoff(context.Background(), app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   taskB.ID,
		},
		Summary:     "handoff b",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff(task B) error = %v", err)
	}

	attentionA, err := svc.RaiseAttentionItem(context.Background(), app.RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   taskA.ID,
		},
		Kind:        domain.AttentionKindRiskNote,
		Summary:     "attention a",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem(task A) error = %v", err)
	}
	attentionB, err := svc.RaiseAttentionItem(context.Background(), app.RaiseAttentionItemInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   taskB.ID,
		},
		Kind:        domain.AttentionKindRiskNote,
		Summary:     "attention b",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("RaiseAttentionItem(task B) error = %v", err)
	}

	leaseA, err := svc.IssueCapabilityLease(context.Background(), app.IssueCapabilityLeaseInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeTask,
		ScopeID:   taskA.ID,
		Role:      domain.CapabilityRoleBuilder,
		AgentName: "Builder A",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(task A) error = %v", err)
	}
	leaseB, err := svc.IssueCapabilityLease(context.Background(), app.IssueCapabilityLeaseInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeTask,
		ScopeID:   taskB.ID,
		Role:      domain.CapabilityRoleBuilder,
		AgentName: "Builder B",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(task B) error = %v", err)
	}

	return authScopeFixture{
		adapter:      NewAppServiceAdapter(svc, auth),
		auth:         auth,
		projectID:    project.ID,
		approvedPath: "project/" + project.ID + "/branch/" + branchA.ID + "/phase/" + phaseA.ID,
		taskA:        taskA,
		taskB:        taskB,
		handoffA:     handoffA,
		handoffB:     handoffB,
		attentionA:   attentionA,
		attentionB:   attentionB,
		leaseA:       leaseA,
		leaseB:       leaseB,
	}
}

// mustCreateTaskForTest creates one fixture work item or fails the test.
func mustCreateTaskForTest(t *testing.T, svc *app.Service, in app.CreateTaskInput) domain.Task {
	t.Helper()

	task, err := svc.CreateTask(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateTask(%q) error = %v", in.Title, err)
	}
	return task
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
			name: "update task in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "update_task",
				Namespace:     "tillsyn",
				ResourceType:  "task",
				ResourceID:    fixture.taskA.ID,
				Context:       map[string]string{"task_id": fixture.taskA.ID},
			},
		},
		{
			name: "update task out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "update_task",
				Namespace:     "tillsyn",
				ResourceType:  "task",
				ResourceID:    fixture.taskB.ID,
				Context:       map[string]string{"task_id": fixture.taskB.ID},
			},
			wantErr: ErrAuthorizationDenied,
		},
		{
			name: "move task in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "move_task",
				Namespace:     "tillsyn",
				ResourceType:  "task",
				ResourceID:    fixture.taskA.ID,
				Context:       map[string]string{"task_id": fixture.taskA.ID},
			},
		},
		{
			name: "delete task out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "delete_task",
				Namespace:     "tillsyn",
				ResourceType:  "task",
				ResourceID:    fixture.taskB.ID,
				Context:       map[string]string{"task_id": fixture.taskB.ID},
			},
			wantErr: ErrAuthorizationDenied,
		},
		{
			name: "restore task in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "restore_task",
				Namespace:     "tillsyn",
				ResourceType:  "task",
				ResourceID:    fixture.taskA.ID,
				Context:       map[string]string{"task_id": fixture.taskA.ID},
			},
		},
		{
			name: "reparent task out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "reparent_task",
				Namespace:     "tillsyn",
				ResourceType:  "task",
				ResourceID:    fixture.taskB.ID,
				Context: map[string]string{
					"task_id":   fixture.taskB.ID,
					"parent_id": fixture.taskB.ID,
				},
			},
			wantErr: ErrAuthorizationDenied,
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
			name: "update handoff out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "update_handoff",
				Namespace:     "tillsyn",
				ResourceType:  "handoff",
				ResourceID:    fixture.handoffB.ID,
				Context:       map[string]string{"handoff_id": fixture.handoffB.ID},
			},
			wantErr: ErrAuthorizationDenied,
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
			name: "resolve attention out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "resolve_attention_item",
				Namespace:     "tillsyn",
				ResourceType:  "attention_item",
				ResourceID:    fixture.attentionB.ID,
				Context:       map[string]string{"attention_id": fixture.attentionB.ID},
			},
			wantErr: ErrAuthorizationDenied,
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
			name: "renew lease out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "renew_capability_lease",
				Namespace:     "tillsyn",
				ResourceType:  "capability_lease",
				ResourceID:    fixture.leaseB.InstanceID,
				Context:       map[string]string{"agent_instance_id": fixture.leaseB.InstanceID},
			},
			wantErr: ErrAuthorizationDenied,
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
			name: "revoke lease out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "revoke_capability_lease",
				Namespace:     "tillsyn",
				ResourceType:  "capability_lease",
				ResourceID:    fixture.leaseB.InstanceID,
				Context:       map[string]string{"agent_instance_id": fixture.leaseB.InstanceID},
			},
			wantErr: ErrAuthorizationDenied,
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
			name: "create task under in-scope parent",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_task",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "task",
				ResourceID:    "new",
				Context: map[string]string{
					"project_id": fixture.projectID,
					"parent_id":  fixture.taskA.ID,
				},
			},
		},
		{
			name: "create task under out-of-scope parent",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_task",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "task",
				ResourceID:    "new",
				Context: map[string]string{
					"project_id": fixture.projectID,
					"parent_id":  fixture.taskB.ID,
				},
			},
			wantErr: ErrAuthorizationDenied,
		},
		{
			name: "create comment on in-scope target",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_comment",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "comment",
				ResourceID:    fixture.taskA.ID,
				Context: map[string]string{
					"project_id":  fixture.projectID,
					"target_type": "task",
				},
			},
		},
		{
			name: "create comment on out-of-scope target",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_comment",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "comment",
				ResourceID:    fixture.taskB.ID,
				Context: map[string]string{
					"project_id":  fixture.projectID,
					"target_type": "task",
				},
			},
			wantErr: ErrAuthorizationDenied,
		},
		{
			name: "create handoff in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_handoff",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "handoff",
				ResourceID:    fixture.taskA.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "task",
				},
			},
		},
		{
			name: "create handoff out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "create_handoff",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "handoff",
				ResourceID:    fixture.taskB.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "task",
				},
			},
			wantErr: ErrAuthorizationDenied,
		},
		{
			name: "raise attention in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "raise_attention_item",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "attention_item",
				ResourceID:    fixture.taskA.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "task",
				},
			},
		},
		{
			name: "raise attention out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "raise_attention_item",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "attention_item",
				ResourceID:    fixture.taskB.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "task",
				},
			},
			wantErr: ErrAuthorizationDenied,
		},
		{
			name: "issue capability lease in scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "issue_capability_lease",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "capability_lease",
				ResourceID:    fixture.taskA.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "task",
				},
			},
		},
		{
			name: "issue capability lease out of scope",
			req: MutationAuthorizationRequest{
				SessionID:     sessionID,
				SessionSecret: sessionSecret,
				Action:        "issue_capability_lease",
				Namespace:     "project:" + fixture.projectID,
				ResourceType:  "capability_lease",
				ResourceID:    fixture.taskB.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "task",
				},
			},
			wantErr: ErrAuthorizationDenied,
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
	projectSessionID, projectSessionSecret := mustIssueApprovedPathSessionForTest(t, fixture.auth, "project/"+fixture.projectID)

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
			name:          "global approval may bind project template library",
			sessionID:     globalSessionID,
			sessionSecret: globalSessionSecret,
			req: MutationAuthorizationRequest{
				Action:       "bind_project_template_library",
				Namespace:    "tillsyn",
				ResourceType: "project",
				ResourceID:   fixture.projectID,
				Context: map[string]string{
					"project_id": fixture.projectID,
				},
			},
		},
		{
			name:          "project approval may not bind project template library",
			sessionID:     projectSessionID,
			sessionSecret: projectSessionSecret,
			req: MutationAuthorizationRequest{
				Action:       "bind_project_template_library",
				Namespace:    "tillsyn",
				ResourceType: "project",
				ResourceID:   fixture.projectID,
				Context: map[string]string{
					"project_id": fixture.projectID,
				},
			},
			wantErr: ErrAuthorizationDenied,
		},
		{
			name:          "global approval may not issue in-project lease",
			sessionID:     globalSessionID,
			sessionSecret: globalSessionSecret,
			req: MutationAuthorizationRequest{
				Action:       "issue_capability_lease",
				Namespace:    "project:" + fixture.projectID,
				ResourceType: "capability_lease",
				ResourceID:   fixture.taskA.ID,
				Context: map[string]string{
					"project_id": fixture.projectID,
					"scope_type": "task",
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
