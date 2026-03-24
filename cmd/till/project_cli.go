package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/config"
	"github.com/hylla/tillsyn/internal/domain"
)

// projectDiscoveryError returns a discoverability hint for missing project ids.
func projectDiscoveryError(command string) error {
	return fmt.Errorf("--project-id is required for %s; run till project list to discover a project id, then run till project discover --project-id <project-id> to review collaboration readiness, or run till project create --name \"Example Project\" to create one", command)
}

// requireProjectID validates one project-scoped command input.
func requireProjectID(command, projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return projectDiscoveryError(command)
	}
	return nil
}

// locateProjectForCLI resolves one project by id and preserves archived-project guidance.
func locateProjectForCLI(ctx context.Context, svc *app.Service, projectID string, includeArchived bool, command string) (domain.Project, error) {
	if svc == nil {
		return domain.Project{}, fmt.Errorf("app service is not configured")
	}
	projects, err := svc.ListProjects(ctx, includeArchived)
	if err != nil {
		return domain.Project{}, fmt.Errorf("%s: list projects: %w", command, err)
	}
	projectID = strings.TrimSpace(projectID)
	for _, project := range projects {
		if project.ID == projectID {
			return project, nil
		}
	}
	if !includeArchived {
		allProjects, err := svc.ListProjects(ctx, true)
		if err != nil {
			return domain.Project{}, fmt.Errorf("%s %q: list archived projects: %w", command, projectID, err)
		}
		for _, project := range allProjects {
			if project.ID == projectID {
				return domain.Project{}, fmt.Errorf("%s %q: archived project is hidden by default; rerun with --include-archived", command, projectID)
			}
		}
	}
	return domain.Project{}, fmt.Errorf("%s %q: not found; run till project list to discover a project id", command, projectID)
}

// runProjectList lists projects and writes a human-readable table.
func runProjectList(ctx context.Context, svc *app.Service, opts projectListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	projects, err := svc.ListProjects(ctx, opts.includeArchived)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}
	slices.SortFunc(projects, compareProjectsForCLI)
	emptyGuidance := `Next step: till project create --name "Example Project"`
	if !opts.includeArchived && len(projects) == 0 {
		allProjects, err := svc.ListProjects(ctx, true)
		if err != nil {
			return fmt.Errorf("list projects including archived: %w", err)
		}
		if len(allProjects) > 0 {
			emptyGuidance = "Next step: till project list --include-archived"
		}
	}
	return writeProjectList(stdout, projects, emptyGuidance)
}

// runProjectCreate creates one project and writes a human-readable detail view.
func runProjectCreate(ctx context.Context, svc *app.Service, cfg config.Config, opts projectCreateCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	metadata, err := buildProjectMetadata(opts)
	if err != nil {
		return err
	}
	ctx = cliMutationContext(ctx, cfg)
	project, err := svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{
		Name:        opts.name,
		Description: opts.description,
		Kind:        domain.KindID(opts.kind),
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return writeProjectDetail(stdout, project, "Created Project")
}

// runProjectShow shows one project and writes a human-readable detail view.
func runProjectShow(ctx context.Context, svc *app.Service, opts projectShowCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project show", opts.projectID); err != nil {
		return err
	}
	project, err := locateProjectForCLI(ctx, svc, opts.projectID, opts.includeArchived, "show project")
	if err != nil {
		return err
	}
	return writeProjectDetail(stdout, project, "Project")
}

// runProjectDiscover shows one project collaboration-readiness summary and writes a human-readable detail view.
func runProjectDiscover(ctx context.Context, svc *app.Service, opts projectReadinessCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project discover", opts.projectID); err != nil {
		return err
	}
	project, err := locateProjectForCLI(ctx, svc, opts.projectID, opts.includeArchived, "discover project")
	if err != nil {
		return err
	}
	pendingRequests, err := svc.ListAuthRequests(ctx, domain.AuthRequestListFilter{
		ProjectID: project.ID,
		State:     domain.AuthRequestStatePending,
	})
	if err != nil {
		return fmt.Errorf("discover project %q: list pending auth requests: %w", project.ID, err)
	}
	activeSessions, err := svc.ListAuthSessions(ctx, app.AuthSessionFilter{
		ProjectID: project.ID,
		State:     "active",
	})
	if err != nil {
		return fmt.Errorf("discover project %q: list auth sessions: %w", project.ID, err)
	}
	leases, err := svc.ListCapabilityLeases(ctx, app.ListCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeProject,
	})
	if err != nil {
		return fmt.Errorf("discover project %q: list capability leases: %w", project.ID, err)
	}
	handoffs, err := svc.ListHandoffs(ctx, app.ListHandoffsInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			ScopeType: domain.ScopeLevelProject,
		},
	})
	if err != nil {
		return fmt.Errorf("discover project %q: list handoffs: %w", project.ID, err)
	}
	return writeProjectReadiness(stdout, project, pendingRequests, activeSessions, leases, handoffs)
}

// buildProjectMetadata merges optional JSON metadata with explicit flag overrides.
func buildProjectMetadata(opts projectCreateCommandOptions) (domain.ProjectMetadata, error) {
	metadata, err := parseOptionalProjectMetadataJSON(opts.metadataJSON)
	if err != nil {
		return domain.ProjectMetadata{}, err
	}
	if strings.TrimSpace(opts.owner) != "" {
		metadata.Owner = opts.owner
	}
	if strings.TrimSpace(opts.icon) != "" {
		metadata.Icon = opts.icon
	}
	if strings.TrimSpace(opts.color) != "" {
		metadata.Color = opts.color
	}
	if strings.TrimSpace(opts.homepage) != "" {
		metadata.Homepage = opts.homepage
	}
	if len(opts.tags) > 0 {
		metadata.Tags = append([]string(nil), opts.tags...)
	}
	if strings.TrimSpace(opts.standardsMarkdown) != "" {
		metadata.StandardsMarkdown = opts.standardsMarkdown
	}
	return metadata, nil
}

// parseOptionalProjectMetadataJSON parses an optional project metadata JSON document.
func parseOptionalProjectMetadataJSON(raw string) (domain.ProjectMetadata, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return domain.ProjectMetadata{}, nil
	}
	var metadata domain.ProjectMetadata
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return domain.ProjectMetadata{}, fmt.Errorf("parse --metadata-json: %w", err)
	}
	return metadata, nil
}

// writeProjectList renders projects as a stable table with names first.
func writeProjectList(stdout io.Writer, projects []domain.Project, emptyGuidance string) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NAME\tID\tKIND\tOWNER\tARCHIVED"); err != nil {
		return fmt.Errorf("write project list header: %w", err)
	}
	if len(projects) == 0 {
		if _, err := fmt.Fprintln(tw, "(none)\t-\t-\t-\t-"); err != nil {
			return fmt.Errorf("write empty project list row: %w", err)
		}
	}
	for _, project := range projects {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			compactText(project.Name),
			compactText(project.ID),
			compactText(string(project.Kind)),
			compactText(project.Metadata.Owner),
			projectArchivedText(project.ArchivedAt),
		); err != nil {
			return fmt.Errorf("write project list row: %w", err)
		}
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush project list: %w", err)
	}
	if len(projects) == 0 {
		if _, err := fmt.Fprintf(stdout, "\n%s\n", strings.TrimSpace(emptyGuidance)); err != nil {
			return fmt.Errorf("write empty project list guidance: %w", err)
		}
	}
	return nil
}

// writeProjectDetail renders one project as a readable key/value summary.
func writeProjectDetail(stdout io.Writer, project domain.Project, title string) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	rows := [][2]string{
		{"name", compactText(project.Name)},
		{"id", compactText(project.ID)},
		{"slug", compactText(project.Slug)},
		{"kind", compactText(string(project.Kind))},
		{"owner", compactText(project.Metadata.Owner)},
		{"icon", compactText(project.Metadata.Icon)},
		{"color", compactText(project.Metadata.Color)},
		{"homepage", compactText(project.Metadata.Homepage)},
		{"tags", compactText(strings.Join(project.Metadata.Tags, ", "))},
		{"archived", projectArchivedText(project.ArchivedAt)},
		{"description", compactText(project.Description)},
		{"standards_markdown", compactText(project.Metadata.StandardsMarkdown)},
	}
	if _, err := fmt.Fprintln(tw, strings.ToUpper(strings.TrimSpace(title))); err != nil {
		return fmt.Errorf("write project detail header: %w", err)
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", row[0], row[1]); err != nil {
			return fmt.Errorf("write project detail row: %w", err)
		}
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush project detail: %w", err)
	}
	return nil
}

// writeProjectReadiness renders one project collaboration summary with a next-step bridge.
func writeProjectReadiness(stdout io.Writer, project domain.Project, pendingRequests []domain.AuthRequest, activeSessions []app.AuthSession, leases []domain.CapabilityLease, handoffs []domain.Handoff) error {
	activeAgentSessions := countActiveAgentSessions(activeSessions)
	activeOrchestratorSessions := countActiveAgentRoleSessions(activeSessions, "orchestrator")
	openHandoffs := countOpenHandoffs(handoffs)
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "PROJECT COLLABORATION READINESS"); err != nil {
		return fmt.Errorf("write project readiness header: %w", err)
	}
	rows := [][2]string{
		{"name", compactText(project.Name)},
		{"id", compactText(project.ID)},
		{"slug", compactText(project.Slug)},
		{"kind", compactText(string(project.Kind))},
		{"owner", compactText(project.Metadata.Owner)},
		{"archived", projectArchivedText(project.ArchivedAt)},
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", row[0], row[1]); err != nil {
			return fmt.Errorf("write project readiness row: %w", err)
		}
	}
	if _, err := fmt.Fprintln(tw); err != nil {
		return fmt.Errorf("write project readiness spacer: %w", err)
	}
	if _, err := fmt.Fprintln(tw, "COORDINATION INVENTORY"); err != nil {
		return fmt.Errorf("write project readiness inventory header: %w", err)
	}
	inventoryRows := [][2]string{
		{"pending_auth_requests", fmt.Sprintf("%d", len(pendingRequests))},
		{"active_auth_sessions", fmt.Sprintf("%d", len(activeSessions))},
		{"active_agent_sessions", fmt.Sprintf("%d", activeAgentSessions)},
		{"active_orchestrator_sessions", fmt.Sprintf("%d", activeOrchestratorSessions)},
		{"project_leases", fmt.Sprintf("%d", len(leases))},
		{"open_project_handoffs", fmt.Sprintf("%d", openHandoffs)},
	}
	for _, row := range inventoryRows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", row[0], row[1]); err != nil {
			return fmt.Errorf("write project readiness inventory row: %w", err)
		}
	}
	if _, err := fmt.Fprintln(tw); err != nil {
		return fmt.Errorf("write project readiness spacer: %w", err)
	}
	command, reason := projectReadinessNextStep(project.ID, pendingRequests, activeOrchestratorSessions, len(leases), openHandoffs)
	if _, err := fmt.Fprintln(tw, "NEXT STEP"); err != nil {
		return fmt.Errorf("write project readiness next-step header: %w", err)
	}
	if _, err := fmt.Fprintf(tw, "command\t%s\n", command); err != nil {
		return fmt.Errorf("write project readiness next-step command: %w", err)
	}
	if _, err := fmt.Fprintf(tw, "reason\t%s\n", reason); err != nil {
		return fmt.Errorf("write project readiness next-step reason: %w", err)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush project readiness: %w", err)
	}
	return nil
}

// projectReadinessNextStep returns the recommended project collaboration next step.
func projectReadinessNextStep(projectID string, pendingRequests []domain.AuthRequest, activeOrchestratorSessions, leases, openHandoffs int) (string, string) {
	switch {
	case len(pendingRequests) == 1:
		return fmt.Sprintf("till auth request show --request-id %s", pendingRequests[0].ID), "Inspect the pending auth request, then approve or deny it before issuing a session, lease, or handoff."
	case len(pendingRequests) > 1:
		return fmt.Sprintf("till auth request list --project-id %s --state pending", projectID), "Multiple pending auth requests are visible; inspect them and then approve or deny the right one before issuing a session, lease, or handoff."
	case activeOrchestratorSessions == 0:
		return fmt.Sprintf("till auth request create --path project/%s --principal-id <agent-id> --principal-type agent --principal-role orchestrator --client-id <client-id> --client-type mcp-stdio --reason %q", projectID, "project collaboration setup"), "No active orchestrator session is visible for this project yet; request and approve one before issuing a project lease."
	case leases == 0:
		return fmt.Sprintf("till lease issue --project-id %s --role builder --agent-name <agent-name>", projectID), "An active orchestrator session is visible, so issue the project lease before creating the first handoff."
	case openHandoffs == 0:
		return fmt.Sprintf("till handoff create --project-id %s --summary %q --source-role builder --target-role qa", projectID, "project collaboration handoff"), "A session and lease are visible, so create the first handoff when the collaboration needs a durable checkpoint."
	default:
		return fmt.Sprintf("till handoff list --project-id %s", projectID), "Collaboration surfaces are populated; inspect the currently open handoffs before making the next change."
	}
}

// countActiveAgentSessions returns the number of active project sessions owned by agent principals.
func countActiveAgentSessions(sessions []app.AuthSession) int {
	count := 0
	for _, session := range sessions {
		if strings.EqualFold(strings.TrimSpace(session.PrincipalType), "agent") {
			count++
		}
	}
	return count
}

// countActiveAgentRoleSessions returns the number of active agent sessions matching one role label.
func countActiveAgentRoleSessions(sessions []app.AuthSession, role string) int {
	role = strings.TrimSpace(role)
	if role == "" {
		return 0
	}
	count := 0
	for _, session := range sessions {
		if !strings.EqualFold(strings.TrimSpace(session.PrincipalType), "agent") {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(session.PrincipalRole), role) {
			count++
		}
	}
	return count
}

// countOpenHandoffs returns the number of non-terminal handoffs visible for project coordination.
func countOpenHandoffs(handoffs []domain.Handoff) int {
	count := 0
	for _, handoff := range handoffs {
		if domain.IsTerminalHandoffStatus(handoff.Status) {
			continue
		}
		count++
	}
	return count
}

// compactText reduces multiline output to one line for table display.
func compactText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "-"
	}
	return strings.Join(strings.Fields(text), " ")
}

// projectArchivedText renders archived state for project tables.
func projectArchivedText(archivedAt *time.Time) string {
	if archivedAt == nil {
		return "active"
	}
	return "archived"
}

// compareProjectsForCLI sorts projects by name then id for stable operator discovery.
func compareProjectsForCLI(a, b domain.Project) int {
	leftName := strings.ToLower(strings.TrimSpace(a.Name))
	rightName := strings.ToLower(strings.TrimSpace(b.Name))
	if leftName != rightName {
		return strings.Compare(leftName, rightName)
	}
	return strings.Compare(strings.TrimSpace(a.ID), strings.TrimSpace(b.ID))
}
