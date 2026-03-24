package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/config"
	"github.com/hylla/tillsyn/internal/domain"
)

// projectDiscoveryError returns a discoverability hint for missing project ids.
func projectDiscoveryError(command string) error {
	return fmt.Errorf("%s requires --project-id; run till project list to discover a project id", command)
}

// requireProjectID validates one project-scoped command input.
func requireProjectID(command, projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return projectDiscoveryError(command)
	}
	return nil
}

// runProjectList lists projects and writes a human-readable table.
func runProjectList(ctx context.Context, svc *app.Service, opts *projectListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if opts == nil {
		return fmt.Errorf("project command state is not configured")
	}
	projects, err := svc.ListProjects(ctx, opts.includeArchived)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}
	return writeProjectList(stdout, projects)
}

// runProjectCreate creates one project and writes a human-readable detail view.
func runProjectCreate(ctx context.Context, svc *app.Service, cfg config.Config, opts *projectCreateCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if opts == nil {
		return fmt.Errorf("project command state is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	metadata := domain.ProjectMetadata{
		Owner:             opts.owner,
		Icon:              opts.icon,
		Color:             opts.color,
		Homepage:          opts.homepage,
		Tags:              append([]string(nil), opts.tags...),
		StandardsMarkdown: opts.standardsMarkdown,
	}
	project, err := svc.CreateProjectWithMetadata(ctx, app.CreateProjectInput{
		Name:        opts.name,
		Description: opts.description,
		Kind:        domain.KindID(opts.kind),
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return writeProjectDetail(stdout, project)
}

// runProjectShow shows one project and writes a human-readable detail view.
func runProjectShow(ctx context.Context, svc *app.Service, opts *projectShowCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if opts == nil {
		return fmt.Errorf("project command state is not configured")
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
			return writeProjectDetail(stdout, project)
		}
	}
	return fmt.Errorf("show project %q: not found", projectID)
}

// writeProjectList renders projects as a stable table with names first.
func writeProjectList(stdout io.Writer, projects []domain.Project) error {
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
	return nil
}

// writeProjectDetail renders one project as a readable key/value summary.
func writeProjectDetail(stdout io.Writer, project domain.Project) error {
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
	if _, err := fmt.Fprintln(tw, "PROJECT"); err != nil {
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
