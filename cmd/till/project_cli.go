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
	return fmt.Errorf("--project-id is required for %s; run till project list to discover a project id or till project create --name \"Example Project\" to create one", command)
}

// requireProjectID validates one project-scoped command input.
func requireProjectID(command, projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return projectDiscoveryError(command)
	}
	return nil
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
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if err := requireProjectID("project show", opts.projectID); err != nil {
		return err
	}
	projects, err := svc.ListProjects(ctx, opts.includeArchived)
	if err != nil {
		return fmt.Errorf("show project: %w", err)
	}
	projectID := strings.TrimSpace(opts.projectID)
	for _, project := range projects {
		if project.ID == projectID {
			return writeProjectDetail(stdout, project, "Project")
		}
	}
	if !opts.includeArchived {
		allProjects, err := svc.ListProjects(ctx, true)
		if err != nil {
			return fmt.Errorf("show project %q: list archived projects: %w", projectID, err)
		}
		for _, project := range allProjects {
			if project.ID == projectID {
				return fmt.Errorf("show project %q: archived project is hidden by default; rerun with --include-archived", projectID)
			}
		}
	}
	return fmt.Errorf("show project %q: not found; run till project list to discover a project id", projectID)
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
