package domain

import (
	"testing"
	"time"
)

// TestNewCommentDefaultsAndNormalization verifies behavior for the covered scenario.
func TestNewCommentDefaultsAndNormalization(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	comment, err := NewComment(CommentInput{
		ID:           "comment-1",
		ProjectID:    " project-1 ",
		TargetType:   CommentTargetType(" ACTION_ITEM "),
		TargetID:     " item-1 ",
		BodyMarkdown: " **done** ",
	}, now)
	if err != nil {
		t.Fatalf("NewComment() error = %v", err)
	}
	if comment.ProjectID != "project-1" {
		t.Fatalf("expected trimmed project id, got %q", comment.ProjectID)
	}
	if comment.TargetType != CommentTargetTypeActionItem {
		t.Fatalf("expected normalized action-item target type, got %q", comment.TargetType)
	}
	if comment.TargetID != "item-1" {
		t.Fatalf("expected trimmed target id, got %q", comment.TargetID)
	}
	if comment.BodyMarkdown != "**done**" {
		t.Fatalf("expected trimmed markdown body, got %q", comment.BodyMarkdown)
	}
	if comment.Summary != "**done**" {
		t.Fatalf("expected default summary from first non-empty body line, got %q", comment.Summary)
	}
	if comment.ActorType != ActorTypeUser {
		t.Fatalf("expected default actor type user, got %q", comment.ActorType)
	}
	if comment.ActorID != "tillsyn-user" {
		t.Fatalf("expected default actor id tillsyn-user, got %q", comment.ActorID)
	}
	if comment.ActorName != "tillsyn-user" {
		t.Fatalf("expected default actor name tillsyn-user, got %q", comment.ActorName)
	}
	if !comment.CreatedAt.Equal(now.UTC()) || !comment.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("expected UTC timestamps at input time, got created=%s updated=%s", comment.CreatedAt, comment.UpdatedAt)
	}
}

// TestNewCommentUsesProvidedSummary verifies explicit summary normalization behavior.
func TestNewCommentUsesProvidedSummary(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	comment, err := NewComment(CommentInput{
		ID:           "comment-1",
		ProjectID:    "project-1",
		TargetType:   CommentTargetTypeActionItem,
		TargetID:     "item-1",
		Summary:      "  Explicit summary  ",
		BodyMarkdown: "\n\n# Heading\nbody detail",
	}, now)
	if err != nil {
		t.Fatalf("NewComment() error = %v", err)
	}
	if comment.Summary != "Explicit summary" {
		t.Fatalf("expected explicit summary to be trimmed and preserved, got %q", comment.Summary)
	}
}

// TestNewCommentDerivesSummaryFromFirstNonEmptyBodyLine verifies fallback summary behavior.
func TestNewCommentDerivesSummaryFromFirstNonEmptyBodyLine(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	comment, err := NewComment(CommentInput{
		ID:           "comment-1",
		ProjectID:    "project-1",
		TargetType:   CommentTargetTypeActionItem,
		TargetID:     "item-1",
		BodyMarkdown: "\n\n  ## Heading \n\nMore details",
	}, now)
	if err != nil {
		t.Fatalf("NewComment() error = %v", err)
	}
	if comment.Summary != "## Heading" {
		t.Fatalf("expected fallback summary from first non-empty markdown line, got %q", comment.Summary)
	}
}

// TestNewCommentDefaultsActorNameFromActorID verifies actor-name fallback behavior.
func TestNewCommentDefaultsActorNameFromActorID(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	comment, err := NewComment(CommentInput{
		ID:           "comment-1",
		ProjectID:    "project-1",
		TargetType:   CommentTargetTypeActionItem,
		TargetID:     "item-1",
		BodyMarkdown: "done",
		ActorID:      "agent-7",
	}, now)
	if err != nil {
		t.Fatalf("NewComment() error = %v", err)
	}
	if comment.ActorID != "agent-7" {
		t.Fatalf("expected actor id agent-7, got %q", comment.ActorID)
	}
	if comment.ActorName != "agent-7" {
		t.Fatalf("expected actor name fallback to actor id, got %q", comment.ActorName)
	}
}

// TestNewCommentValidation verifies behavior for the covered scenario.
func TestNewCommentValidation(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input CommentInput
		want  error
	}{
		{
			name: "missing id",
			input: CommentInput{
				ProjectID:    "p1",
				TargetType:   CommentTargetTypeProject,
				TargetID:     "p1",
				BodyMarkdown: "body",
			},
			want: ErrInvalidID,
		},
		{
			name: "missing project id",
			input: CommentInput{
				ID:           "c1",
				TargetType:   CommentTargetTypeProject,
				TargetID:     "p1",
				BodyMarkdown: "body",
			},
			want: ErrInvalidID,
		},
		{
			name: "missing target id",
			input: CommentInput{
				ID:           "c1",
				ProjectID:    "p1",
				TargetType:   CommentTargetTypeProject,
				BodyMarkdown: "body",
			},
			want: ErrInvalidTargetID,
		},
		{
			name: "invalid target type",
			input: CommentInput{
				ID:           "c1",
				ProjectID:    "p1",
				TargetType:   CommentTargetType("branch"),
				TargetID:     "c1",
				BodyMarkdown: "body",
			},
			want: ErrInvalidTargetType,
		},
		{
			name: "missing body",
			input: CommentInput{
				ID:           "c1",
				ProjectID:    "p1",
				TargetType:   CommentTargetTypeProject,
				TargetID:     "p1",
				BodyMarkdown: " \n\t ",
			},
			want: ErrInvalidBodyMarkdown,
		},
		{
			name: "invalid actor type",
			input: CommentInput{
				ID:           "c1",
				ProjectID:    "p1",
				TargetType:   CommentTargetTypeActionItem,
				TargetID:     "t1",
				BodyMarkdown: "body",
				ActorType:    ActorType("robot"),
			},
			want: ErrInvalidActorType,
		},
	}

	for _, tc := range tests {
		_, err := NewComment(tc.input, now)
		if err != tc.want {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}
}

// TestNormalizeCommentTarget verifies behavior for the covered scenario.
func TestNormalizeCommentTarget(t *testing.T) {
	target, err := NormalizeCommentTarget(CommentTarget{
		ProjectID:  " p1 ",
		TargetType: CommentTargetType(" ACTION_ITEM "),
		TargetID:   " t2 ",
	})
	if err != nil {
		t.Fatalf("NormalizeCommentTarget() error = %v", err)
	}
	if target.ProjectID != "p1" || target.TargetType != CommentTargetTypeActionItem || target.TargetID != "t2" {
		t.Fatalf("unexpected normalized target %#v", target)
	}
}

// TestNormalizeCommentTargetRejectsLegacyScopeTypes verifies the collapsed
// target-type vocabulary rejects the pre-12-kind branch / phase / subtask
// target types.
func TestNormalizeCommentTargetRejectsLegacyScopeTypes(t *testing.T) {
	for _, legacy := range []CommentTargetType{"branch", "phase", "subtask", "decision", "note"} {
		if _, err := NormalizeCommentTarget(CommentTarget{
			ProjectID:  "p1",
			TargetType: legacy,
			TargetID:   "item-1",
		}); err != ErrInvalidTargetType {
			t.Fatalf("NormalizeCommentTarget(%q) error = %v, want ErrInvalidTargetType", legacy, err)
		}
	}
}
