package main

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/config"
	"github.com/hylla/tillsyn/internal/domain"
)

// TestWriteProjectList renders a stable human-scannable project table.
func TestWriteProjectList(t *testing.T) {
	projects := []domain.Project{
		{
			ID:          "p2",
			Name:        "Beta",
			Kind:        domain.KindID("project"),
			Metadata:    domain.ProjectMetadata{Owner: "team-b"},
			Description: "Second project",
		},
		{
			ID:          "p1",
			Name:        "Alpha",
			Kind:        domain.KindID("go-service"),
			Metadata:    domain.ProjectMetadata{Owner: "team-a"},
			Description: "First\nproject",
		},
	}
	var out strings.Builder
	if err := writeProjectList(&out, projects, `Next step: till project create --name "Example Project"`); err != nil {
		t.Fatalf("writeProjectList() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"Projects", "NAME", "ID", "OWNER", "Alpha", "p1", "go-service", "team-a", "Beta", "p2", "team-b"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in project list output, got %q", want, got)
		}
	}
}

// TestWriteProjectListEmpty guides operators toward project creation when none exist.
func TestWriteProjectListEmpty(t *testing.T) {
	var out strings.Builder
	if err := writeProjectList(&out, nil, `Next step: till project create --name "Example Project"`); err != nil {
		t.Fatalf("writeProjectList(nil) error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "No projects found.") || !strings.Contains(got, "Projects") || !strings.Contains(got, "till project create --name") {
		t.Fatalf("expected empty project table row, got %q", got)
	}
}

// TestWriteProjectListEmptyArchivedHint points archived-only operators toward the include-archived path.
func TestWriteProjectListEmptyArchivedHint(t *testing.T) {
	var out strings.Builder
	if err := writeProjectList(&out, nil, "Next step: till project list --include-archived"); err != nil {
		t.Fatalf("writeProjectList(nil, archived hint) error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "till project list --include-archived") {
		t.Fatalf("expected archived discovery hint, got %q", got)
	}
}

// TestWriteProjectDetail renders the primary name/id-first detail block.
func TestWriteProjectDetail(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Alpha", "First project", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	project.Kind = domain.KindID("go-service")
	project.Metadata.Owner = "team-a"
	project.Metadata.Tags = []string{"go", "cli"}

	var out strings.Builder
	if err := writeProjectDetail(&out, project, "Project"); err != nil {
		t.Fatalf("writeProjectDetail() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"Project", "name", "Alpha", "id", "p1", "slug", "alpha", "kind", "go-service", "description", "First project", "owner", "team-a", "tags", "go, cli"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in project detail output, got %q", want, got)
		}
	}
}

// TestWriteProjectReadiness renders the collaboration bridge and next-step guidance.
func TestWriteProjectReadiness(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Alpha", "First project", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	project.Metadata.Owner = "team-a"
	sessions := []app.AuthSession{
		{SessionID: "s-user", PrincipalType: "user"},
		{SessionID: "s-builder", PrincipalType: "agent", PrincipalRole: "builder"},
		{SessionID: "s-orchestrator", PrincipalType: "agent", PrincipalRole: "orchestrator"},
	}
	handoffs := []domain.Handoff{
		{ID: "h-open", Status: domain.HandoffStatusWaiting},
		{ID: "h-done", Status: domain.HandoffStatusResolved},
	}

	var out strings.Builder
	if err := writeProjectReadiness(&out, project, nil, sessions, nil, handoffs); err != nil {
		t.Fatalf("writeProjectReadiness() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Project Collaboration Readiness",
		"Coordination Inventory",
		"active_auth_sessions",
		"3",
		"active_agent_sessions",
		"2",
		"active_orchestrator_sessions",
		"1",
		"active_project_leases",
		"0",
		"open_project_handoffs",
		"1",
		"Next Step",
		"till lease issue --project-id p1 --role builder --agent-name AGENT_NAME",
		"An active orchestrator session is visible",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in project readiness output, got %q", want, got)
		}
	}
}

// TestProjectWithOwnerFallbackUsesDisplayName verifies local config identity fills empty owner labels.
func TestProjectWithOwnerFallbackUsesDisplayName(t *testing.T) {
	project := domain.Project{Metadata: domain.ProjectMetadata{}}
	project = projectWithOwnerFallback(project, "Evan")
	if got := project.Metadata.Owner; got != "Evan" {
		t.Fatalf("project owner fallback = %q, want %q", got, "Evan")
	}

	project.Metadata.Owner = "explicit-owner"
	project = projectWithOwnerFallback(project, "Evan")
	if got := project.Metadata.Owner; got != "explicit-owner" {
		t.Fatalf("project owner fallback overwrote explicit owner: %q", got)
	}
}

// TestRunProjectCreateUsesTemplateLibrary verifies the CLI create helper binds an approved global template library during project creation.
func TestRunProjectCreateUsesTemplateLibrary(t *testing.T) {
	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	svc := app.NewService(repo, func() string { return uuid.NewString() }, func() time.Time { return now }, app.ServiceConfig{})

	if _, err := svc.ListKindDefinitions(context.Background(), false); err != nil {
		t.Fatalf("ListKindDefinitions() error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(context.Background(), app.CreateKindDefinitionInput{
		ID:          "go-service",
		DisplayName: "Go Service",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToProject},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(go-service) error = %v", err)
	}
	if _, err := svc.UpsertTemplateLibrary(context.Background(), app.UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Description:         "Global defaults for Go projects",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "user-1",
		CreatedByActorName:  "User One",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "user-1",
		ApprovedByActorName: "User One",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []app.UpsertNodeTemplateInput{{
			ID:         "project-template",
			ScopeLevel: domain.KindAppliesToProject,
			NodeKindID: domain.KindID("go-service"),
			ProjectMetadataDefaults: &domain.ProjectMetadata{
				Owner:             "Platform",
				StandardsMarkdown: "Run Go validation",
			},
			ChildRules: []app.UpsertTemplateChildRuleInput{{
				ID:                      "main-branch",
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
	}); err != nil {
		t.Fatalf("UpsertTemplateLibrary(go-defaults) error = %v", err)
	}

	var out strings.Builder
	if err := runProjectCreate(context.Background(), svc, config.Config{}, projectCreateCommandOptions{
		name:              "Go Service",
		kind:              "go-service",
		templateLibraryID: "go-defaults",
	}, &out); err != nil {
		t.Fatalf("runProjectCreate() error = %v", err)
	}
	if got := out.String(); !strings.Contains(got, "Created Project") || !strings.Contains(got, "Go Service") || !strings.Contains(got, "owner") || !strings.Contains(got, "Platform") {
		t.Fatalf("unexpected runProjectCreate output: %q", got)
	}
	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(projects))
	}
	binding, err := svc.GetProjectTemplateBinding(context.Background(), projects[0].ID)
	if err != nil {
		t.Fatalf("GetProjectTemplateBinding() error = %v", err)
	}
	if binding.LibraryID != "go-defaults" {
		t.Fatalf("binding.LibraryID = %q, want go-defaults", binding.LibraryID)
	}
	tasks, err := svc.ListTasks(context.Background(), projects[0].ID, false)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Main Branch" {
		t.Fatalf("unexpected generated tasks %#v", tasks)
	}
}

// TestRequireProjectIDGuidesDiscovery points operators toward discovery before scoped commands run.
func TestRequireProjectIDGuidesDiscovery(t *testing.T) {
	err := requireProjectID("till capture-state", "")
	if err == nil {
		t.Fatal("expected missing project id error")
	}
	got := err.Error()
	for _, want := range []string{"--project-id is required", "till project list", "till project discover --project-id", "till project discover PROJECT_ID", "till project create --name", "till project create \"Example Project\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in project-id guidance, got %q", want, got)
		}
	}
}

// TestResolveProjectNameInput accepts either --name or one positional project name.
func TestResolveProjectNameInput(t *testing.T) {
	cases := []struct {
		name    string
		flag    string
		args    []string
		want    string
		wantErr string
	}{
		{
			name: "flag only",
			flag: "Inbox",
			want: "Inbox",
		},
		{
			name: "positional only",
			args: []string{"Inbox"},
			want: "Inbox",
		},
		{
			name: "matching flag and positional",
			flag: "Inbox",
			args: []string{"Inbox"},
			want: "Inbox",
		},
		{
			name:    "missing name",
			wantErr: "project name is required",
		},
		{
			name:    "conflicting inputs",
			flag:    "Inbox",
			args:    []string{"Roadmap"},
			wantErr: "either --name or one positional project name",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveProjectNameInput(tc.flag, tc.args)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("resolveProjectNameInput() error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveProjectNameInput() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveProjectNameInput() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestResolveProjectIDInput accepts either --project-id or one positional project id.
func TestResolveProjectIDInput(t *testing.T) {
	cases := []struct {
		name    string
		flag    string
		args    []string
		want    string
		wantErr string
	}{
		{
			name: "flag only",
			flag: "p1",
			want: "p1",
		},
		{
			name: "positional only",
			args: []string{"p1"},
			want: "p1",
		},
		{
			name: "matching flag and positional",
			flag: "p1",
			args: []string{"p1"},
			want: "p1",
		},
		{
			name:    "missing project id",
			wantErr: "--project-id is required",
		},
		{
			name:    "conflicting inputs",
			flag:    "p1",
			args:    []string{"p2"},
			wantErr: "either --project-id or one positional project id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveProjectIDInput("project show", tc.flag, tc.args)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("resolveProjectIDInput() error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveProjectIDInput() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveProjectIDInput() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestProjectReadinessNextStep selects the right collaboration bridge from inventory counts.
func TestProjectReadinessNextStep(t *testing.T) {
	cases := []struct {
		name               string
		pendingRequests    []domain.AuthRequest
		activeOrchestrator int
		leases             int
		openHandoffs       int
		wantCommandParts   []string
		wantReason         string
	}{
		{
			name: "single pending auth request first",
			pendingRequests: []domain.AuthRequest{
				{ID: "req-1"},
			},
			activeOrchestrator: 1,
			leases:             1,
			openHandoffs:       1,
			wantCommandParts:   []string{"till auth request show", "--request-id req-1"},
			wantReason:         "Inspect the pending auth request",
		},
		{
			name: "multiple pending auth requests list view",
			pendingRequests: []domain.AuthRequest{
				{ID: "req-1"},
				{ID: "req-2"},
			},
			activeOrchestrator: 1,
			leases:             1,
			openHandoffs:       1,
			wantCommandParts:   []string{"till auth request list", "--project-id p1", "--state pending"},
			wantReason:         "Multiple pending auth requests",
		},
		{
			name:             "request agent session next when none are active",
			wantCommandParts: []string{"till auth request create", "--path project/p1", "--principal-id AGENT_ID", "--principal-type agent", "--principal-role orchestrator", "--client-id CLIENT_ID", "--client-type mcp-stdio"},
			wantReason:       "No active orchestrator session is visible",
		},
		{
			name:               "request orchestrator when only non-orchestrator agent sessions exist",
			wantCommandParts:   []string{"till auth request create", "--principal-role orchestrator"},
			wantReason:         "No active orchestrator session is visible",
			activeOrchestrator: 0,
		},
		{
			name:               "lease after orchestrator session",
			activeOrchestrator: 1,
			wantCommandParts:   []string{"till lease issue", "--project-id p1", "--role builder", "--agent-name AGENT_NAME"},
			wantReason:         "issue the project lease",
		},
		{
			name:               "handoff after lease",
			activeOrchestrator: 1,
			leases:             1,
			wantCommandParts:   []string{"till handoff create", "--project-id p1", "--summary \"project collaboration handoff\"", "--source-role builder", "--target-role qa"},
			wantReason:         "first handoff",
		},
		{
			name:               "inspect handoffs once the bridge is populated",
			activeOrchestrator: 1,
			leases:             1,
			openHandoffs:       1,
			wantCommandParts:   []string{"till handoff list", "--project-id p1"},
			wantReason:         "Collaboration surfaces are populated",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotCommand, gotReason := projectReadinessNextStep("p1", tc.pendingRequests, tc.activeOrchestrator, tc.leases, tc.openHandoffs)
			for _, want := range tc.wantCommandParts {
				if !strings.Contains(gotCommand, want) {
					t.Fatalf("command = %q, want to contain %q", gotCommand, want)
				}
			}
			if !strings.Contains(gotReason, tc.wantReason) {
				t.Fatalf("reason = %q, want to contain %q", gotReason, tc.wantReason)
			}
		})
	}
}

// TestCountActiveAgentSessions counts only agent-owned sessions for readiness guidance.
func TestCountActiveAgentSessions(t *testing.T) {
	sessions := []app.AuthSession{
		{SessionID: "user", PrincipalType: "user"},
		{SessionID: "agent-a", PrincipalType: "agent"},
		{SessionID: "agent-b", PrincipalType: "AGENT"},
	}
	if got := countActiveAgentSessions(sessions); got != 2 {
		t.Fatalf("countActiveAgentSessions() = %d, want 2", got)
	}
}

// TestCountActiveAgentRoleSessions counts only agent sessions matching the requested role.
func TestCountActiveAgentRoleSessions(t *testing.T) {
	sessions := []app.AuthSession{
		{SessionID: "user", PrincipalType: "user", PrincipalRole: "orchestrator"},
		{SessionID: "builder", PrincipalType: "agent", PrincipalRole: "builder"},
		{SessionID: "orchestrator-a", PrincipalType: "agent", PrincipalRole: "orchestrator"},
		{SessionID: "orchestrator-b", PrincipalType: "agent", PrincipalRole: "ORCHESTRATOR"},
	}
	if got := countActiveAgentRoleSessions(sessions, "orchestrator"); got != 2 {
		t.Fatalf("countActiveAgentRoleSessions() = %d, want 2", got)
	}
}

// TestCountOpenHandoffs excludes terminal handoff states from readiness guidance.
func TestCountOpenHandoffs(t *testing.T) {
	handoffs := []domain.Handoff{
		{ID: "open-1", Status: domain.HandoffStatusReady},
		{ID: "open-2", Status: domain.HandoffStatusWaiting},
		{ID: "failed", Status: domain.HandoffStatusFailed},
		{ID: "resolved", Status: domain.HandoffStatusResolved},
		{ID: "superseded", Status: domain.HandoffStatusSuperseded},
	}
	if got := countOpenHandoffs(handoffs); got != 3 {
		t.Fatalf("countOpenHandoffs() = %d, want 3", got)
	}
}

// TestCountActiveCapabilityLeases excludes expired and revoked leases from readiness guidance.
func TestCountActiveCapabilityLeases(t *testing.T) {
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	active, err := domain.NewCapabilityLease(domain.CapabilityLeaseInput{
		InstanceID: "lease-active",
		LeaseToken: "token-active",
		AgentName:  "Builder",
		ProjectID:  "p1",
		ScopeType:  domain.CapabilityScopeProject,
		ScopeID:    "p1",
		Role:       domain.CapabilityRoleBuilder,
		ExpiresAt:  now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("NewCapabilityLease(active) error = %v", err)
	}
	expired := active
	expired.InstanceID = "lease-expired"
	expired.ExpiresAt = now.Add(-time.Minute)
	revoked := active
	revoked.InstanceID = "lease-revoked"
	revokedAt := now.Add(-time.Minute)
	revoked.RevokedAt = &revokedAt

	if got := countActiveCapabilityLeases([]domain.CapabilityLease{active, expired, revoked}, now); got != 1 {
		t.Fatalf("countActiveCapabilityLeases() = %d, want 1", got)
	}
}

// TestBuildProjectMetadataPrefersExplicitFlags verifies flag values override JSON defaults.
func TestBuildProjectMetadataPrefersExplicitFlags(t *testing.T) {
	metadata, err := buildProjectMetadata(projectCreateCommandOptions{
		metadataJSON:      `{"owner":"json-owner","tags":["json"],"homepage":"https://json.invalid"}`,
		owner:             "flag-owner",
		tags:              []string{"flag"},
		standardsMarkdown: "flag standards",
	})
	if err != nil {
		t.Fatalf("buildProjectMetadata() error = %v", err)
	}
	if metadata.Owner != "flag-owner" {
		t.Fatalf("metadata.Owner = %q, want flag-owner", metadata.Owner)
	}
	if len(metadata.Tags) != 1 || metadata.Tags[0] != "flag" {
		t.Fatalf("metadata.Tags = %#v, want []string{\"flag\"}", metadata.Tags)
	}
	if metadata.Homepage != "https://json.invalid" {
		t.Fatalf("metadata.Homepage = %q, want https://json.invalid", metadata.Homepage)
	}
	if metadata.StandardsMarkdown != "flag standards" {
		t.Fatalf("metadata.StandardsMarkdown = %q, want flag standards", metadata.StandardsMarkdown)
	}
}

// TestBuildProjectMetadataRejectsInvalidJSON verifies metadata-json parse failures stay operator-visible.
func TestBuildProjectMetadataRejectsInvalidJSON(t *testing.T) {
	_, err := buildProjectMetadata(projectCreateCommandOptions{metadataJSON: `{"owner":`})
	if err == nil {
		t.Fatal("expected invalid metadata json error")
	}
	if !strings.Contains(err.Error(), "parse --metadata-json") {
		t.Fatalf("expected parse error context, got %v", err)
	}
}

// TestCompareProjectsForCLI sorts names first and ids second for stable discovery output.
func TestCompareProjectsForCLI(t *testing.T) {
	projects := []domain.Project{
		{ID: "p2", Name: "Beta"},
		{ID: "p3", Name: "alpha"},
		{ID: "p1", Name: "Alpha"},
	}
	slices.SortFunc(projects, compareProjectsForCLI)
	got := []string{projects[0].ID, projects[1].ID, projects[2].ID}
	want := []string{"p1", "p3", "p2"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("sorted ids = %v, want %v", got, want)
	}
}
