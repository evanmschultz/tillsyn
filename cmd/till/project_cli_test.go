package main

import (
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
	if err := writeProjectList(&out, projects); err != nil {
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
	if err := writeProjectList(&out, nil); err != nil {
		t.Fatalf("writeProjectList(nil) error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "(none)") || !strings.Contains(got, "NAME") {
		t.Fatalf("expected empty project table row, got %q", got)
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
