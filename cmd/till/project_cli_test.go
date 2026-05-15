package main

import (
	"bytes"
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/google/uuid"
)

// TestWriteProjectList renders a stable human-scannable project table.
func TestWriteProjectList(t *testing.T) {
	projects := []domain.Project{
		{
			ID:          "p2",
			Name:        "Beta",
			Metadata:    domain.ProjectMetadata{Owner: "team-b"},
			Description: "Second project",
		},
		{
			ID:          "p1",
			Name:        "Alpha",
			Metadata:    domain.ProjectMetadata{Owner: "team-a"},
			Description: "First\nproject",
		},
	}
	var out strings.Builder
	if err := writeProjectList(&out, projects, `Next step: till project create --name "Example Project"`); err != nil {
		t.Fatalf("writeProjectList() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"Projects", "NAME", "ID", "OWNER", "Alpha", "p1", "team-a", "Beta", "p2", "team-b"} {
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
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Alpha", Description: "First project"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	project.Metadata.Owner = "team-a"
	project.Metadata.Tags = []string{"go", "cli"}

	var out strings.Builder
	if err := writeProjectDetail(&out, project, "Project"); err != nil {
		t.Fatalf("writeProjectDetail() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"Project", "name", "Alpha", "id", "p1", "slug", "alpha", "description", "First project", "owner", "team-a", "tags", "go, cli"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in project detail output, got %q", want, got)
		}
	}
}

// TestWriteProjectReadiness renders the collaboration bridge and next-step guidance.
func TestWriteProjectReadiness(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Alpha", Description: "First project"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
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

// TestWriteProjectReadinessW2D7Fields asserts that the six W2.D7 first-class
// project fields appear in the collaboration readiness output when populated.
func TestWriteProjectReadinessW2D7Fields(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Alpha"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	project.RepoPrimaryWorktree = "/Users/evan/code/tillsyn/main"
	project.RepoBareRoot = "/Users/evan/code/tillsyn"
	project.BuildTool = "mage"
	project.DevMcpServerName = "tillsyn-dev"
	project.HyllaArtifactRef = "github.com/evanmschultz/tillsyn@main"
	project.Metadata.Groups = []string{"go", "fe"}

	var out strings.Builder
	if err := writeProjectReadiness(&out, project, nil, nil, nil, nil); err != nil {
		t.Fatalf("writeProjectReadiness() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"root_path",
		"bare_root",
		"build_tool",
		"dev_mcp_server_name",
		"hylla_artifact_ref",
		"groups",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected W2.D7 key %q in project readiness output, got %q", want, got)
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
			wantCommandParts: []string{"till auth request create", "--path project/p1", "--principal-id AGENT_ID", "--principal-type agent", "--principal-role orchestrator", "--client-id CLIENT_ID"},
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

// newServiceForProjectUpdateTest opens a fresh in-memory-equivalent SQLite repo and
// wraps it in a minimal *app.Service for runProjectUpdate unit tests. The
// repo is registered for cleanup via t.Cleanup.
func newServiceForProjectUpdateTest(t *testing.T) *app.Service {
	t.Helper()
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	noopLease := false
	svc := app.NewService(repo, func() string { return uuid.NewString() }, nil, app.ServiceConfig{
		RequireAgentLease: &noopLease,
	})
	return svc
}

// seedProjectForUpdateTest creates one minimal project via the service and returns it.
func seedProjectForUpdateTest(t *testing.T, svc *app.Service, name string) domain.Project {
	t.Helper()
	project, err := svc.CreateProjectWithMetadata(context.Background(), app.CreateProjectInput{
		Name: name,
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata(%q) error = %v", name, err)
	}
	return project
}

// TestRunProjectUpdate_MissingProjectIDReturnsDiscoveryError guards the canonical
// missing-project-id path before any service call runs.
func TestRunProjectUpdate_MissingProjectIDReturnsDiscoveryError(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	var out bytes.Buffer
	err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{}, &out)
	if err == nil {
		t.Fatal("expected missing project id error")
	}
	for _, want := range []string{"--project-id is required", "till project list", "till project discover"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in project-id guidance, got %v", want, err)
		}
	}
}

// TestRunProjectUpdate_UpdatesFirstClassFields verifies that flag-supplied first-class
// fields are written through to the project record without clobbering unchanged fields.
func TestRunProjectUpdate_UpdatesFirstClassFields(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "Alpha")

	cases := []struct {
		name    string
		opts    projectUpdateCommandOptions
		wantOut []string
	}{
		{
			// root-path stored in RepoPrimaryWorktree; writeProjectDetail does not
			// surface that field, so we verify success via the title only.
			name: "root-path update",
			opts: projectUpdateCommandOptions{
				projectID: project.ID,
				rootPath:  "/tmp/main",
			},
			wantOut: []string{"Updated Project"},
		},
		{
			// bare-root stored in RepoBareRoot; not surfaced by writeProjectDetail.
			name: "bare-root update",
			opts: projectUpdateCommandOptions{
				projectID: project.ID,
				bareRoot:  "/tmp/bare",
			},
			wantOut: []string{"Updated Project"},
		},
		{
			name: "description update",
			opts: projectUpdateCommandOptions{
				projectID:   project.ID,
				description: "new desc",
			},
			wantOut: []string{"Updated Project", "new desc"},
		},
		{
			name: "build-tool update",
			opts: projectUpdateCommandOptions{
				projectID: project.ID,
				buildTool: "mage",
			},
			wantOut: []string{"Updated Project"},
		},
		{
			name: "dev-mcp-server-name update",
			opts: projectUpdateCommandOptions{
				projectID:        project.ID,
				devMcpServerName: "tillsyn-dev",
			},
			wantOut: []string{"Updated Project"},
		},
		{
			name: "hylla-artifact-ref update",
			opts: projectUpdateCommandOptions{
				projectID:        project.ID,
				hyllaArtifactRef: "github.com/org/repo@main",
			},
			wantOut: []string{"Updated Project"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := runProjectUpdate(context.Background(), svc, config.Config{}, tc.opts, &out); err != nil {
				t.Fatalf("runProjectUpdate() error = %v", err)
			}
			got := out.String()
			for _, want := range tc.wantOut {
				if !strings.Contains(got, want) {
					t.Fatalf("expected %q in output, got %q", want, got)
				}
			}
		})
	}
}

// TestRunProjectUpdate_OwnerMetadata verifies --owner flag updates Metadata.Owner
// without clobbering other metadata fields.
func TestRunProjectUpdate_OwnerMetadata(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "OwnerProject")

	var out bytes.Buffer
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID: project.ID,
		owner:     "Evan",
	}, &out); err != nil {
		t.Fatalf("runProjectUpdate(--owner) error = %v", err)
	}
	if got := out.String(); !strings.Contains(got, "Evan") {
		t.Fatalf("expected owner %q in output, got %q", "Evan", got)
	}
}

// TestRunProjectUpdate_AddGroupAppendsAndDeduplicates verifies --add-group appends
// a new group and is a no-op when the group is already present.
func TestRunProjectUpdate_AddGroupAppendsAndDeduplicates(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "GroupProject")

	// First add: "go" should appear in Groups.
	var out bytes.Buffer
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID: project.ID,
		addGroups: []string{"go"},
	}, &out); err != nil {
		t.Fatalf("runProjectUpdate(--add-group go) error = %v", err)
	}

	// Second add: "go" again — should not duplicate.
	out.Reset()
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID: project.ID,
		addGroups: []string{"go"},
	}, &out); err != nil {
		t.Fatalf("runProjectUpdate(--add-group go again) error = %v", err)
	}

	// Verify the current state has exactly one "go" by reading the project.
	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	var found domain.Project
	for _, p := range projects {
		if p.ID == project.ID {
			found = p
			break
		}
	}
	goCount := 0
	for _, g := range found.Metadata.Groups {
		if g == "go" {
			goCount++
		}
	}
	if goCount != 1 {
		t.Fatalf("expected exactly 1 'go' group, got %d in %v", goCount, found.Metadata.Groups)
	}
}

// TestRunProjectUpdate_RemoveGroupFiltersAndIsNoopWhenAbsent verifies --remove-group
// removes a present group and is a no-op when the group is not present.
func TestRunProjectUpdate_RemoveGroupFiltersAndIsNoopWhenAbsent(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "RemoveGroupProject")

	// Seed: add "go" and "fe" groups.
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID: project.ID,
		addGroups: []string{"go", "fe"},
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(add go fe) error = %v", err)
	}

	// Remove "go".
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID:    project.ID,
		removeGroups: []string{"go"},
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(--remove-group go) error = %v", err)
	}

	// Remove absent group "gen" — should be no-op (no error).
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID:    project.ID,
		removeGroups: []string{"gen"},
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(--remove-group gen absent) error = %v", err)
	}

	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	var found domain.Project
	for _, p := range projects {
		if p.ID == project.ID {
			found = p
			break
		}
	}
	for _, g := range found.Metadata.Groups {
		if g == "go" {
			t.Fatalf("expected 'go' to be removed from Groups, got %v", found.Metadata.Groups)
		}
	}
	hasFe := false
	for _, g := range found.Metadata.Groups {
		if g == "fe" {
			hasFe = true
		}
	}
	if !hasFe {
		t.Fatalf("expected 'fe' to remain in Groups after removing 'go', got %v", found.Metadata.Groups)
	}
}

// TestRunProjectUpdate_AddGroupRejectsUnknownGroup verifies that unknown group
// values fail with a clear validation error before any service call.
func TestRunProjectUpdate_AddGroupRejectsUnknownGroup(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "ValidateGroupProject")

	var out bytes.Buffer
	err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID: project.ID,
		addGroups: []string{"invalid-group"},
	}, &out)
	if err == nil {
		t.Fatal("expected unknown group error")
	}
	if !strings.Contains(err.Error(), "invalid-group") {
		t.Fatalf("expected group name in error, got %v", err)
	}
}

// TestRunProjectUpdate_MultipleAddRemoveGroups verifies repeatable --add-group and
// --remove-group flags process all supplied values in one call.
func TestRunProjectUpdate_MultipleAddRemoveGroups(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "MultiGroupProject")

	// Add "go" and "fe" in one call.
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID: project.ID,
		addGroups: []string{"go", "fe"},
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(add go fe) error = %v", err)
	}

	// Remove "go" and verify "fe" remains.
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID:    project.ID,
		removeGroups: []string{"go"},
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(remove go) error = %v", err)
	}

	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	var found domain.Project
	for _, p := range projects {
		if p.ID == project.ID {
			found = p
			break
		}
	}
	if len(found.Metadata.Groups) != 1 || found.Metadata.Groups[0] != "fe" {
		t.Fatalf("expected Groups=[fe], got %v", found.Metadata.Groups)
	}
}

// TestRunProjectUpdate_MetadataFlagsIconColorHomepageTags verifies that
// --icon, --color, --homepage, and --tags (via opts.tags) update the
// corresponding Metadata fields via the read-then-merge pattern.
func TestRunProjectUpdate_MetadataFlagsIconColorHomepageTags(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "MetaProject")

	cases := []struct {
		name  string
		opts  projectUpdateCommandOptions
		check func(t *testing.T, p domain.Project)
	}{
		{
			name: "icon updates Metadata.Icon",
			opts: projectUpdateCommandOptions{
				projectID: project.ID,
				icon:      "star",
			},
			check: func(t *testing.T, p domain.Project) {
				t.Helper()
				if p.Metadata.Icon != "star" {
					t.Fatalf("expected Icon=%q, got %q", "star", p.Metadata.Icon)
				}
			},
		},
		{
			name: "color updates Metadata.Color",
			opts: projectUpdateCommandOptions{
				projectID: project.ID,
				color:     "#ff0000",
			},
			check: func(t *testing.T, p domain.Project) {
				t.Helper()
				if p.Metadata.Color != "#ff0000" {
					t.Fatalf("expected Color=%q, got %q", "#ff0000", p.Metadata.Color)
				}
			},
		},
		{
			name: "homepage updates Metadata.Homepage",
			opts: projectUpdateCommandOptions{
				projectID: project.ID,
				homepage:  "https://example.invalid",
			},
			check: func(t *testing.T, p domain.Project) {
				t.Helper()
				if p.Metadata.Homepage != "https://example.invalid" {
					t.Fatalf("expected Homepage=%q, got %q", "https://example.invalid", p.Metadata.Homepage)
				}
			},
		},
		{
			name: "tags updates Metadata.Tags",
			opts: projectUpdateCommandOptions{
				projectID: project.ID,
				tags:      []string{"a", "b", "c"},
			},
			check: func(t *testing.T, p domain.Project) {
				t.Helper()
				want := []string{"a", "b", "c"}
				if len(p.Metadata.Tags) != len(want) {
					t.Fatalf("expected Tags=%v, got %v", want, p.Metadata.Tags)
				}
				for i, tag := range want {
					if p.Metadata.Tags[i] != tag {
						t.Fatalf("Tags[%d]: expected %q, got %q", i, tag, p.Metadata.Tags[i])
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := runProjectUpdate(context.Background(), svc, config.Config{}, tc.opts, &out); err != nil {
				t.Fatalf("runProjectUpdate(%s) error = %v", tc.name, err)
			}
			projects, err := svc.ListProjects(context.Background(), false)
			if err != nil {
				t.Fatalf("ListProjects() error = %v", err)
			}
			var found domain.Project
			for _, p := range projects {
				if p.ID == project.ID {
					found = p
					break
				}
			}
			tc.check(t, found)
		})
	}
}

// TestRunProjectUpdate_SingleDescriptionFlagDoesNotClobberOthers verifies that
// running runProjectUpdate with only --description preserves all other
// first-class and metadata fields that were set at seed time.
func TestRunProjectUpdate_SingleDescriptionFlagDoesNotClobberOthers(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)

	// Seed with all first-class and key metadata fields populated.
	created, err := svc.CreateProjectWithMetadata(context.Background(), app.CreateProjectInput{
		Name:                "FullProject",
		Description:         "Original description",
		HyllaArtifactRef:    "github.com/org/repo@main",
		RepoBareRoot:        "/tmp/bare",
		RepoPrimaryWorktree: "/tmp/main",
		BuildTool:           "mage",
		DevMcpServerName:    "tillsyn-dev",
		Metadata: domain.ProjectMetadata{
			Groups: []string{"go", "fe"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	// Update ONLY the description.
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID:   created.ID,
		description: "updated-desc",
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(--description updated-desc) error = %v", err)
	}

	// Read back and assert description was updated and all other fields are unchanged.
	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	var found domain.Project
	for _, p := range projects {
		if p.ID == created.ID {
			found = p
			break
		}
	}
	if found.Description != "updated-desc" {
		t.Fatalf("expected Description updated to %q, got %q", "updated-desc", found.Description)
	}
	if found.HyllaArtifactRef != "github.com/org/repo@main" {
		t.Fatalf("expected HyllaArtifactRef preserved, got %q", found.HyllaArtifactRef)
	}
	if found.RepoBareRoot != "/tmp/bare" {
		t.Fatalf("expected RepoBareRoot preserved, got %q", found.RepoBareRoot)
	}
	if found.RepoPrimaryWorktree != "/tmp/main" {
		t.Fatalf("expected RepoPrimaryWorktree preserved, got %q", found.RepoPrimaryWorktree)
	}
	if found.BuildTool != "mage" {
		t.Fatalf("expected BuildTool preserved, got %q", found.BuildTool)
	}
	if found.DevMcpServerName != "tillsyn-dev" {
		t.Fatalf("expected DevMcpServerName preserved, got %q", found.DevMcpServerName)
	}
	if len(found.Metadata.Groups) != 2 {
		t.Fatalf("expected Groups preserved as [go fe], got %v", found.Metadata.Groups)
	}
}

// TestRunProjectUpdate_AddGroupIncludesGenGroup extends the add/remove group
// tests to cover the "gen" value in allowedInitGroups (belt-and-braces: the
// three-value enum has gen, go, fe; prior tests only exercised go and fe).
func TestRunProjectUpdate_AddGroupIncludesGenGroup(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "GenGroupProject")

	// Add "gen".
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID: project.ID,
		addGroups: []string{"gen"},
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(--add-group gen) error = %v", err)
	}

	// Remove "gen".
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID:    project.ID,
		removeGroups: []string{"gen"},
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(--remove-group gen) error = %v", err)
	}

	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	var found domain.Project
	for _, p := range projects {
		if p.ID == project.ID {
			found = p
			break
		}
	}
	for _, g := range found.Metadata.Groups {
		if g == "gen" {
			t.Fatalf("expected 'gen' removed from Groups, got %v", found.Metadata.Groups)
		}
	}
}

// TestWriteProjectDetail_IncludesFirstClassFields verifies that writeProjectDetail
// surfaces the Drop 4a first-class project fields so users can visually confirm
// flag-driven updates.
func TestWriteProjectDetail_IncludesFirstClassFields(t *testing.T) {
	project := domain.Project{
		ID:                  "proj-id",
		Name:                "TestProject",
		RepoPrimaryWorktree: "/tmp/main",
		RepoBareRoot:        "/tmp/bare",
		BuildTool:           "mage",
		DevMcpServerName:    "tillsyn-dev",
		HyllaArtifactRef:    "github.com/org/repo@main",
		Metadata: domain.ProjectMetadata{
			Groups: []string{"go", "fe"},
		},
	}
	var out bytes.Buffer
	if err := writeProjectDetail(&out, project, "Project"); err != nil {
		t.Fatalf("writeProjectDetail() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"/tmp/main",
		"/tmp/bare",
		"go",
		"mage",
		"tillsyn-dev",
		"github.com/org/repo@main",
		"go, fe",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in writeProjectDetail output, got:\n%s", want, got)
		}
	}
}

// newServiceForProjectLifecycleTest opens a fresh in-memory-equivalent SQLite repo
// and wraps it in a minimal *app.Service for delete/archive/restore/rename tests.
func newServiceForProjectLifecycleTest(t *testing.T) *app.Service {
	t.Helper()
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	noopLease := false
	svc := app.NewService(repo, func() string { return uuid.NewString() }, nil, app.ServiceConfig{
		RequireAgentLease: &noopLease,
	})
	return svc
}

// seedProjectForLifecycleTest creates one minimal project via the service and returns it.
func seedProjectForLifecycleTest(t *testing.T, svc *app.Service, name string) domain.Project {
	t.Helper()
	project, err := svc.CreateProjectWithMetadata(context.Background(), app.CreateProjectInput{
		Name: name,
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata(%q) error = %v", name, err)
	}
	return project
}

// TestRunProjectDelete_RequiresConfirm verifies that missing --confirm fails with a
// clear error before any service call runs.
func TestRunProjectDelete_RequiresConfirm(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)
	project := seedProjectForLifecycleTest(t, svc, "DeleteTarget")

	var out bytes.Buffer
	err := runProjectDelete(context.Background(), svc, config.Config{}, projectDeleteCommandOptions{
		projectID: project.ID,
		confirm:   false,
	}, &out)
	if err == nil {
		t.Fatal("expected confirm-required error")
	}
	for _, want := range []string{"--confirm", "hard delete is irreversible"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

// TestRunProjectDelete_SuccessPath verifies that --confirm=true deletes the project
// and writes a confirmation line to stdout.
func TestRunProjectDelete_SuccessPath(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)
	project := seedProjectForLifecycleTest(t, svc, "DeleteMe")

	var out bytes.Buffer
	if err := runProjectDelete(context.Background(), svc, config.Config{}, projectDeleteCommandOptions{
		projectID: project.ID,
		confirm:   true,
	}, &out); err != nil {
		t.Fatalf("runProjectDelete() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, project.ID) && !strings.Contains(got, "deleted") {
		t.Fatalf("expected deletion confirmation in output, got %q", got)
	}
	// Verify the project is actually gone.
	projects, err := svc.ListProjects(context.Background(), true)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	for _, p := range projects {
		if p.ID == project.ID {
			t.Fatalf("expected project %q to be deleted, but still found", project.ID)
		}
	}
}

// TestRunProjectDelete_MissingProjectIDReturnsDiscoveryError validates the
// missing-project-id guard before confirm is checked.
func TestRunProjectDelete_MissingProjectIDReturnsDiscoveryError(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)

	var out bytes.Buffer
	err := runProjectDelete(context.Background(), svc, config.Config{}, projectDeleteCommandOptions{
		confirm: true,
	}, &out)
	if err == nil {
		t.Fatal("expected missing project id error")
	}
	if !strings.Contains(err.Error(), "--project-id is required") {
		t.Fatalf("expected project-id guidance, got %v", err)
	}
}

// TestRunProjectArchive_ArchivesProject verifies the archive path returns the
// archived project detail with title "Archived Project".
func TestRunProjectArchive_ArchivesProject(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)
	project := seedProjectForLifecycleTest(t, svc, "ArchiveMe")

	var out bytes.Buffer
	if err := runProjectArchive(context.Background(), svc, config.Config{}, projectArchiveCommandOptions{
		projectID: project.ID,
	}, &out); err != nil {
		t.Fatalf("runProjectArchive() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"Archived Project", project.Name} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in archive output, got %q", want, got)
		}
	}
}

// TestRunProjectArchive_MissingProjectIDReturnsDiscoveryError guards the
// missing-project-id path before any service call runs.
func TestRunProjectArchive_MissingProjectIDReturnsDiscoveryError(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)

	var out bytes.Buffer
	err := runProjectArchive(context.Background(), svc, config.Config{}, projectArchiveCommandOptions{}, &out)
	if err == nil {
		t.Fatal("expected missing project id error")
	}
	if !strings.Contains(err.Error(), "--project-id is required") {
		t.Fatalf("expected project-id guidance, got %v", err)
	}
}

// TestRunProjectRestore_RestoresProject verifies the restore path returns the
// restored project detail with title "Restored Project".
func TestRunProjectRestore_RestoresProject(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)
	project := seedProjectForLifecycleTest(t, svc, "RestoreMe")

	// Archive first.
	if _, err := svc.ArchiveProject(context.Background(), project.ID); err != nil {
		t.Fatalf("ArchiveProject() error = %v", err)
	}

	var out bytes.Buffer
	if err := runProjectRestore(context.Background(), svc, config.Config{}, projectRestoreCommandOptions{
		projectID: project.ID,
	}, &out); err != nil {
		t.Fatalf("runProjectRestore() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"Restored Project", project.Name} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in restore output, got %q", want, got)
		}
	}
}

// TestRunProjectRestore_MissingProjectIDReturnsDiscoveryError guards the
// missing-project-id path before any service call runs.
func TestRunProjectRestore_MissingProjectIDReturnsDiscoveryError(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)

	var out bytes.Buffer
	err := runProjectRestore(context.Background(), svc, config.Config{}, projectRestoreCommandOptions{}, &out)
	if err == nil {
		t.Fatal("expected missing project id error")
	}
	if !strings.Contains(err.Error(), "--project-id is required") {
		t.Fatalf("expected project-id guidance, got %v", err)
	}
}

// TestRunProjectRename_MissingNameReturnsError verifies that omitting --name fails
// with a clear required-field error before any service call.
func TestRunProjectRename_MissingNameReturnsError(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)
	project := seedProjectForLifecycleTest(t, svc, "RenameMe")

	var out bytes.Buffer
	err := runProjectRename(context.Background(), svc, config.Config{}, projectRenameCommandOptions{
		projectID: project.ID,
		newName:   "",
	}, &out)
	if err == nil {
		t.Fatal("expected missing name error")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Fatalf("expected --name guidance in error, got %v", err)
	}
}

// TestRunProjectRename_MissingProjectIDReturnsDiscoveryError guards the
// missing-project-id path before name validation or any service call.
func TestRunProjectRename_MissingProjectIDReturnsDiscoveryError(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)

	var out bytes.Buffer
	err := runProjectRename(context.Background(), svc, config.Config{}, projectRenameCommandOptions{
		newName: "NewName",
	}, &out)
	if err == nil {
		t.Fatal("expected missing project id error")
	}
	if !strings.Contains(err.Error(), "--project-id is required") {
		t.Fatalf("expected project-id guidance, got %v", err)
	}
}

// TestRunProjectRename_PreservesAllOtherFields verifies that rename updates only
// the Name field and preserves all other first-class and metadata fields.
func TestRunProjectRename_PreservesAllOtherFields(t *testing.T) {
	svc := newServiceForProjectLifecycleTest(t)

	// Create a fully-populated project.
	created, err := svc.CreateProjectWithMetadata(context.Background(), app.CreateProjectInput{
		Name:                "OriginalName",
		Description:         "Original description",
		HyllaArtifactRef:    "github.com/org/repo@main",
		RepoBareRoot:        "/tmp/bare",
		RepoPrimaryWorktree: "/tmp/main",
		BuildTool:           "mage",
		DevMcpServerName:    "tillsyn-dev",
		Metadata: domain.ProjectMetadata{
			Owner: "original-owner",
		},
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}

	var out bytes.Buffer
	if err := runProjectRename(context.Background(), svc, config.Config{}, projectRenameCommandOptions{
		projectID: created.ID,
		newName:   "RenamedProject",
	}, &out); err != nil {
		t.Fatalf("runProjectRename() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "RenamedProject") {
		t.Fatalf("expected new name in output, got %q", got)
	}
	if !strings.Contains(got, "Renamed Project") {
		t.Fatalf("expected title 'Renamed Project' in output, got %q", got)
	}

	// Read back and confirm all other fields preserved.
	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	var found domain.Project
	for _, p := range projects {
		if p.ID == created.ID {
			found = p
			break
		}
	}
	if found.Name != "RenamedProject" {
		t.Fatalf("expected Name=%q, got %q", "RenamedProject", found.Name)
	}
	if found.Description != "Original description" {
		t.Fatalf("expected Description preserved, got %q", found.Description)
	}
	if found.HyllaArtifactRef != "github.com/org/repo@main" {
		t.Fatalf("expected HyllaArtifactRef preserved, got %q", found.HyllaArtifactRef)
	}
	if found.Metadata.Owner != "original-owner" {
		t.Fatalf("expected Owner preserved, got %q", found.Metadata.Owner)
	}
}

// TestRunProjectUpdate_WhitespaceTrimmedBeforeGroupValidation verifies that
// leading/trailing whitespace is trimmed from --add-group values BEFORE
// validation, so "  go  " is accepted rather than rejected as an unknown group.
func TestRunProjectUpdate_WhitespaceTrimmedBeforeGroupValidation(t *testing.T) {
	svc := newServiceForProjectUpdateTest(t)
	project := seedProjectForUpdateTest(t, svc, "TrimProject")

	// "  go  " should be accepted after trim.
	if err := runProjectUpdate(context.Background(), svc, config.Config{}, projectUpdateCommandOptions{
		projectID: project.ID,
		addGroups: []string{"  go  "},
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("runProjectUpdate(--add-group '  go  ') error = %v; expected trim before validation", err)
	}

	projects, err := svc.ListProjects(context.Background(), false)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	var found domain.Project
	for _, p := range projects {
		if p.ID == project.ID {
			found = p
			break
		}
	}
	hasGo := false
	for _, g := range found.Metadata.Groups {
		if g == "go" {
			hasGo = true
		}
	}
	if !hasGo {
		t.Fatalf("expected 'go' in Groups after trimmed add, got %v", found.Metadata.Groups)
	}
}
