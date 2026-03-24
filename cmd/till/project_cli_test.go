package main

import (
	"slices"
	"strings"
	"testing"
	"time"

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
	for _, want := range []string{"NAME", "ID", "OWNER", "Alpha", "p1", "go-service", "team-a", "Beta", "p2", "team-b"} {
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
	if !strings.Contains(got, "(none)") || !strings.Contains(got, "NAME") || !strings.Contains(got, "till project create --name") {
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
	for _, want := range []string{"PROJECT", "name", "Alpha", "id", "p1", "slug", "alpha", "kind", "go-service", "description", "First project", "owner", "team-a", "tags", "go, cli"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in project detail output, got %q", want, got)
		}
	}
}

// TestRequireProjectIDGuidesDiscovery points operators toward discovery before scoped commands run.
func TestRequireProjectIDGuidesDiscovery(t *testing.T) {
	err := requireProjectID("till capture-state", "")
	if err == nil {
		t.Fatal("expected missing project id error")
	}
	got := err.Error()
	for _, want := range []string{"--project-id is required", "till project list", "till project create --name"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in project-id guidance, got %q", want, got)
		}
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
