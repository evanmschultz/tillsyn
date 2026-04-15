package app

import (
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestThreadContextSubjectIDRoundTrip verifies encoded thread-context identifiers decode back into canonical comment targets.
func TestThreadContextSubjectIDRoundTrip(t *testing.T) {
	target := domain.CommentTarget{
		ProjectID:  "project-a",
		TargetType: domain.CommentTargetTypeTask,
		TargetID:   "task-42",
	}
	subjectID := BuildThreadContextSubjectID(target)
	if subjectID == "" {
		t.Fatal("expected non-empty thread-context subject id")
	}
	decoded, err := ParseThreadContextSubjectID(subjectID)
	if err != nil {
		t.Fatalf("ParseThreadContextSubjectID() error = %v", err)
	}
	if decoded != target {
		t.Fatalf("decoded target = %#v, want %#v", decoded, target)
	}
}

// TestEmbeddingSearchTargetForCommentTarget verifies project and work-item threads resolve to the correct search target families.
func TestEmbeddingSearchTargetForCommentTarget(t *testing.T) {
	projectType, projectID, err := EmbeddingSearchTargetForCommentTarget(domain.CommentTarget{
		ProjectID:  "project-a",
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   "project-a",
	})
	if err != nil {
		t.Fatalf("EmbeddingSearchTargetForCommentTarget(project) error = %v", err)
	}
	if projectType != EmbeddingSearchTargetTypeProject || projectID != "project-a" {
		t.Fatalf("project target = (%q, %q), want (project, project-a)", projectType, projectID)
	}

	workItemType, workItemID, err := EmbeddingSearchTargetForCommentTarget(domain.CommentTarget{
		ProjectID:  "project-a",
		TargetType: domain.CommentTargetTypeTask,
		TargetID:   "task-42",
	})
	if err != nil {
		t.Fatalf("EmbeddingSearchTargetForCommentTarget(task) error = %v", err)
	}
	if workItemType != EmbeddingSearchTargetTypeWorkItem || workItemID != "task-42" {
		t.Fatalf("work-item target = (%q, %q), want (work_item, task-42)", workItemType, workItemID)
	}
}

// TestBuildProjectDocumentEmbeddingContentIncludesDescriptiveFields verifies project-level semantic documents carry the intended descriptive surfaces.
func TestBuildProjectDocumentEmbeddingContentIncludesDescriptiveFields(t *testing.T) {
	project := domain.Project{
		ID:          "project-a",
		Name:        "Ops Search",
		Description: "Semantic indexing rollout",
		Metadata: domain.ProjectMetadata{
			Tags:              []string{"embeddings", "search"},
			StandardsMarkdown: "Use markdown-first project docs.",
		},
	}
	content := buildProjectDocumentEmbeddingContent(project)
	for _, want := range []string{
		project.Name,
		project.Description,
		"embeddings, search",
		project.Metadata.StandardsMarkdown,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("project document content missing %q: %q", want, content)
		}
	}
}
