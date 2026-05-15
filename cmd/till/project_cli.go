package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// projectDiscoveryError returns a discoverability hint for missing project ids.
func projectDiscoveryError(command string) error {
	return fmt.Errorf("--project-id is required for %s; run till project list to discover a project id, then run till project discover --project-id PROJECT_ID (or till project discover PROJECT_ID) to review collaboration readiness, or run till project create --name \"Example Project\" (or till project create \"Example Project\") to create one", command)
}

// resolveProjectNameInput accepts either --name or one positional project name.
func resolveProjectNameInput(name string, args []string) (string, error) {
	name = strings.TrimSpace(name)
	if len(args) > 1 {
		return "", fmt.Errorf("project create accepts at most one positional project name")
	}
	if len(args) == 1 {
		positional := strings.TrimSpace(args[0])
		if positional == "" {
			return "", fmt.Errorf("project create positional project name cannot be empty")
		}
		if name == "" {
			return positional, nil
		}
		if name != positional {
			return "", fmt.Errorf("project create accepts either --name or one positional project name; received both %q and %q", name, positional)
		}
	}
	if name == "" {
		return "", fmt.Errorf("project name is required; pass --name \"Example Project\" or one positional name, or run till project create --help")
	}
	return name, nil
}

// resolveProjectIDInput accepts either --project-id or one positional project id.
func resolveProjectIDInput(command, projectID string, args []string) (string, error) {
	projectID = strings.TrimSpace(projectID)
	if len(args) > 1 {
		return "", fmt.Errorf("%s accepts at most one positional project id", command)
	}
	if len(args) == 1 {
		positional := strings.TrimSpace(args[0])
		if positional == "" {
			return "", projectDiscoveryError(command)
		}
		if projectID == "" {
			return positional, nil
		}
		if projectID != positional {
			return "", fmt.Errorf("%s accepts either --project-id or one positional project id; received both %q and %q", command, projectID, positional)
		}
	}
	if err := requireProjectID(command, projectID); err != nil {
		return "", err
	}
	return projectID, nil
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
func runProjectList(ctx context.Context, svc *app.Service, cfg config.Config, opts projectListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	projects, err := svc.ListProjects(ctx, opts.includeArchived)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}
	for idx := range projects {
		projects[idx] = projectWithOwnerFallback(projects[idx], cfg.Identity.DisplayName)
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
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return writeProjectDetail(stdout, project, "Created Project")
}

// runProjectUpdate reads the existing project, merges explicit flag values, then
// calls (*app.Service).UpdateProject. Fields not supplied in opts are preserved
// from the existing record; value-typed UpdateProjectInput has no pointer
// sentinels, so the read-first pattern is mandatory to avoid silently clobbering
// unchanged fields.
//
// Group membership changes:
//   - --add-group appends a value to Metadata.Groups when not already present (dedup
//     via linear scan); rejects values outside the allowedInitGroups set with a clear error.
//   - --remove-group filters the named value out of Metadata.Groups (no-op when absent).
func runProjectUpdate(ctx context.Context, svc *app.Service, cfg config.Config, opts projectUpdateCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project update", opts.projectID); err != nil {
		return err
	}
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}

	// Validate --add-group values before any read/write. Trim whitespace first
	// so the validation policy is consistent with applyGroupMutations, which
	// also trims each value before dedup/append. Over-rejecting trimmed-valid
	// input ("  go  " rejected as unknown) would be inconsistent and confusing.
	for _, g := range opts.addGroups {
		trimmed := strings.TrimSpace(g)
		if !isAllowedProjectGroup(trimmed) {
			return fmt.Errorf("project update: unknown group %q; allowed values: %s", g, strings.Join(allowedInitGroups, ", "))
		}
	}

	existing, err := locateProjectForCLI(ctx, svc, opts.projectID, false, "project update")
	if err != nil {
		return err
	}

	ctx = cliMutationContext(ctx, cfg)

	// Merge: start from all existing first-class fields, then overwrite flag-supplied values.
	name := existing.Name
	description := existing.Description
	hyllaArtifactRef := existing.HyllaArtifactRef
	repoBareRoot := existing.RepoBareRoot
	repoPrimaryWorktree := existing.RepoPrimaryWorktree
	buildTool := existing.BuildTool
	devMcpServerName := existing.DevMcpServerName

	if strings.TrimSpace(opts.description) != "" {
		description = opts.description
	}
	if strings.TrimSpace(opts.rootPath) != "" {
		repoPrimaryWorktree = opts.rootPath
	}
	if strings.TrimSpace(opts.bareRoot) != "" {
		repoBareRoot = opts.bareRoot
	}
	if strings.TrimSpace(opts.hyllaArtifactRef) != "" {
		hyllaArtifactRef = opts.hyllaArtifactRef
	}
	if strings.TrimSpace(opts.buildTool) != "" {
		buildTool = opts.buildTool
	}
	if strings.TrimSpace(opts.devMcpServerName) != "" {
		devMcpServerName = opts.devMcpServerName
	}

	// Merge metadata: preserve all existing fields, apply flag-driven overrides.
	metadata := existing.Metadata
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
		metadata.Tags = opts.tags
	}

	// Apply group mutations on top of existing groups.
	groups := applyGroupMutations(metadata.Groups, opts.addGroups, opts.removeGroups)
	metadata.Groups = groups

	project, err := svc.UpdateProject(ctx, app.UpdateProjectInput{
		ProjectID:           opts.projectID,
		Name:                name,
		Description:         description,
		Metadata:            metadata,
		HyllaArtifactRef:    hyllaArtifactRef,
		RepoBareRoot:        repoBareRoot,
		RepoPrimaryWorktree: repoPrimaryWorktree,
		BuildTool:           buildTool,
		DevMcpServerName:    devMcpServerName,
	})
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return writeProjectDetail(stdout, project, "Updated Project")
}

// applyGroupMutations returns a new groups slice with addGroups appended (dedup)
// and removeGroups filtered out. The original slice is not mutated.
func applyGroupMutations(existing, add, remove []string) []string {
	groups := append([]string(nil), existing...)
	for _, g := range add {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		found := false
		for _, eg := range groups {
			if eg == g {
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, g)
		}
	}
	if len(remove) == 0 {
		return groups
	}
	removeSet := make(map[string]struct{}, len(remove))
	for _, r := range remove {
		r = strings.TrimSpace(r)
		if r != "" {
			removeSet[r] = struct{}{}
		}
	}
	filtered := groups[:0:0]
	for _, g := range groups {
		if _, skip := removeSet[g]; !skip {
			filtered = append(filtered, g)
		}
	}
	return filtered
}

// isAllowedProjectGroup reports whether g is a member of allowedInitGroups.
func isAllowedProjectGroup(g string) bool {
	for _, allowed := range allowedInitGroups {
		if g == allowed {
			return true
		}
	}
	return false
}

// projectDeleteCommandOptions holds options for the project delete subcommand.
type projectDeleteCommandOptions struct {
	projectID string
	confirm   bool
}

// projectArchiveCommandOptions holds options for the project archive subcommand.
type projectArchiveCommandOptions struct {
	projectID string
}

// projectRestoreCommandOptions holds options for the project restore subcommand.
type projectRestoreCommandOptions struct {
	projectID string
}

// projectRenameCommandOptions holds options for the project rename subcommand.
type projectRenameCommandOptions struct {
	projectID string
	newName   string
}

// runProjectDelete hard-deletes one project. --confirm is required; omitting it
// returns a clear error because hard delete is irreversible.
func runProjectDelete(ctx context.Context, svc *app.Service, cfg config.Config, opts projectDeleteCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project delete", opts.projectID); err != nil {
		return err
	}
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if !opts.confirm {
		return fmt.Errorf("till project delete requires --confirm flag; hard delete is irreversible")
	}
	ctx = cliMutationContext(ctx, cfg)
	if err := svc.DeleteProject(ctx, opts.projectID); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	_, err := fmt.Fprintf(stdout, "Project %q deleted.\n", opts.projectID)
	return err
}

// runProjectArchive archives one project and writes the archived project detail.
func runProjectArchive(ctx context.Context, svc *app.Service, cfg config.Config, opts projectArchiveCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project archive", opts.projectID); err != nil {
		return err
	}
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	project, err := svc.ArchiveProject(ctx, opts.projectID)
	if err != nil {
		return fmt.Errorf("archive project: %w", err)
	}
	return writeProjectDetail(stdout, project, "Archived Project")
}

// runProjectRestore restores one archived project and writes the restored project detail.
func runProjectRestore(ctx context.Context, svc *app.Service, cfg config.Config, opts projectRestoreCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project restore", opts.projectID); err != nil {
		return err
	}
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	project, err := svc.RestoreProject(ctx, opts.projectID)
	if err != nil {
		return fmt.Errorf("restore project: %w", err)
	}
	return writeProjectDetail(stdout, project, "Restored Project")
}

// runProjectRename renames one project by calling (*Service).UpdateProject with
// a new Name while preserving all other first-class and metadata fields.
// --name is required and must be non-empty.
func runProjectRename(ctx context.Context, svc *app.Service, cfg config.Config, opts projectRenameCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project rename", opts.projectID); err != nil {
		return err
	}
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if strings.TrimSpace(opts.newName) == "" {
		return fmt.Errorf("project rename requires --name <new-name>; new name cannot be empty")
	}
	existing, err := locateProjectForCLI(ctx, svc, opts.projectID, false, "project rename")
	if err != nil {
		return err
	}
	ctx = cliMutationContext(ctx, cfg)
	project, err := svc.UpdateProject(ctx, app.UpdateProjectInput{
		ProjectID:           opts.projectID,
		Name:                strings.TrimSpace(opts.newName),
		Description:         existing.Description,
		Metadata:            existing.Metadata,
		HyllaArtifactRef:    existing.HyllaArtifactRef,
		RepoBareRoot:        existing.RepoBareRoot,
		RepoPrimaryWorktree: existing.RepoPrimaryWorktree,
		BuildTool:           existing.BuildTool,
		DevMcpServerName:    existing.DevMcpServerName,
	})
	if err != nil {
		return fmt.Errorf("rename project: %w", err)
	}
	return writeProjectDetail(stdout, project, "Renamed Project")
}

// runProjectShow shows one project and writes a human-readable detail view.
func runProjectShow(ctx context.Context, svc *app.Service, cfg config.Config, opts projectShowCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project show", opts.projectID); err != nil {
		return err
	}
	project, err := locateProjectForCLI(ctx, svc, opts.projectID, opts.includeArchived, "show project")
	if err != nil {
		return err
	}
	project = projectWithOwnerFallback(project, cfg.Identity.DisplayName)
	return writeProjectDetail(stdout, project, "Project")
}

// runProjectDiscover shows one project collaboration-readiness summary and writes a human-readable detail view.
func runProjectDiscover(ctx context.Context, svc *app.Service, cfg config.Config, opts projectReadinessCommandOptions, stdout io.Writer) error {
	if err := requireProjectID("project discover", opts.projectID); err != nil {
		return err
	}
	project, err := locateProjectForCLI(ctx, svc, opts.projectID, opts.includeArchived, "discover project")
	if err != nil {
		return err
	}
	project = projectWithOwnerFallback(project, cfg.Identity.DisplayName)
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
	rows := make([][]string, 0, len(projects))
	for _, project := range projects {
		rows = append(rows, []string{
			compactText(project.Name),
			compactText(project.ID),
			compactText(project.Metadata.Owner),
			projectArchivedText(project.ArchivedAt),
		})
	}
	printer := newCLIPrinter(stdout)
	if err := writeCLITableWithPrinter(printer, "Projects", []string{"NAME", "ID", "OWNER", "ARCHIVED"}, rows, "No projects found."); err != nil {
		return err
	}
	if len(projects) == 0 {
		if err := writeCLIPanelWithPrinter(printer, "Next Step", strings.TrimSpace(emptyGuidance), ""); err != nil {
			return fmt.Errorf("write empty project list guidance: %w", err)
		}
	}
	return nil
}

// writeProjectDetail renders one project as a readable key/value summary.
// Includes the Drop 4a first-class fields (root paths, build tool,
// dev MCP server name, Hylla artifact ref, and groups) so users can visually
// confirm what flag-driven updates changed.
func writeProjectDetail(stdout io.Writer, project domain.Project, title string) error {
	rows := [][2]string{
		{"name", compactText(project.Name)},
		{"id", compactText(project.ID)},
		{"slug", compactText(project.Slug)},
		{"owner", compactText(project.Metadata.Owner)},
		{"icon", compactText(project.Metadata.Icon)},
		{"color", compactText(project.Metadata.Color)},
		{"homepage", compactText(project.Metadata.Homepage)},
		{"tags", compactText(strings.Join(project.Metadata.Tags, ", "))},
		{"archived", projectArchivedText(project.ArchivedAt)},
		{"description", compactText(project.Description)},
		{"standards_markdown", compactText(project.Metadata.StandardsMarkdown)},
		{"root_path", compactText(project.RepoPrimaryWorktree)},
		{"bare_root", compactText(project.RepoBareRoot)},
		{"build_tool", compactText(project.BuildTool)},
		{"dev_mcp_server_name", compactText(project.DevMcpServerName)},
		{"hylla_artifact_ref", compactText(project.HyllaArtifactRef)},
		{"groups", compactText(strings.Join(project.Metadata.Groups, ", "))},
	}
	return writeCLIKV(stdout, strings.TrimSpace(title), rows)
}

// writeProjectReadiness renders one project collaboration summary with a next-step bridge.
func writeProjectReadiness(stdout io.Writer, project domain.Project, pendingRequests []domain.AuthRequest, activeSessions []app.AuthSession, leases []domain.CapabilityLease, handoffs []domain.Handoff) error {
	activeAgentSessions := countActiveAgentSessions(activeSessions)
	activeOrchestratorSessions := countActiveAgentRoleSessions(activeSessions, "orchestrator")
	activeLeases := countActiveCapabilityLeases(leases, time.Now().UTC())
	openHandoffs := countOpenHandoffs(handoffs)
	printer := newCLIPrinter(stdout)
	rows := [][2]string{
		{"name", compactText(project.Name)},
		{"id", compactText(project.ID)},
		{"slug", compactText(project.Slug)},
		{"owner", compactText(project.Metadata.Owner)},
		{"archived", projectArchivedText(project.ArchivedAt)},
		{"root_path", compactText(project.RepoPrimaryWorktree)},
		{"bare_root", compactText(project.RepoBareRoot)},
		{"build_tool", compactText(project.BuildTool)},
		{"dev_mcp_server_name", compactText(project.DevMcpServerName)},
		{"hylla_artifact_ref", compactText(project.HyllaArtifactRef)},
		{"groups", compactText(strings.Join(project.Metadata.Groups, ", "))},
	}
	if err := writeCLIKVWithPrinter(printer, "Project Collaboration Readiness", rows); err != nil {
		return err
	}
	inventoryRows := [][2]string{
		{"pending_auth_requests", fmt.Sprintf("%d", len(pendingRequests))},
		{"active_auth_sessions", fmt.Sprintf("%d", len(activeSessions))},
		{"active_agent_sessions", fmt.Sprintf("%d", activeAgentSessions)},
		{"active_orchestrator_sessions", fmt.Sprintf("%d", activeOrchestratorSessions)},
		{"active_project_leases", fmt.Sprintf("%d", activeLeases)},
		{"open_project_handoffs", fmt.Sprintf("%d", openHandoffs)},
	}
	if err := writeCLIKVWithPrinter(printer, "Coordination Inventory", inventoryRows); err != nil {
		return err
	}
	command, reason := projectReadinessNextStep(project.ID, pendingRequests, activeOrchestratorSessions, activeLeases, openHandoffs)
	if err := writeCLIPanelWithPrinter(printer, "Next Step", command, reason); err != nil {
		return fmt.Errorf("write project readiness next step: %w", err)
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
		return fmt.Sprintf("till auth request create --path project/%s --principal-id AGENT_ID --principal-type agent --principal-role orchestrator --client-id CLIENT_ID --reason %q", projectID, "project collaboration setup"), "No active orchestrator session is visible for this project yet; request and approve one before issuing a project lease."
	case leases == 0:
		return fmt.Sprintf("till lease issue --project-id %s --role builder --agent-name AGENT_NAME", projectID), "An active orchestrator session is visible, so issue the project lease before creating the first handoff."
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

// countActiveCapabilityLeases returns the number of currently active project leases.
func countActiveCapabilityLeases(leases []domain.CapabilityLease, now time.Time) int {
	count := 0
	for _, lease := range leases {
		if lease.IsActive(now) {
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

// projectWithOwnerFallback fills the local-MVP owner label from bootstrap identity when metadata is empty.
func projectWithOwnerFallback(project domain.Project, displayName string) domain.Project {
	if strings.TrimSpace(project.Metadata.Owner) != "" {
		return project
	}
	project.Metadata.Owner = strings.TrimSpace(displayName)
	return project
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
