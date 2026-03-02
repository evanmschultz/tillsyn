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
		TargetType:   CommentTargetType(" TASK "),
		TargetID:     " item-1 ",
		BodyMarkdown: " **done** ",
	}, now)
	if err != nil {
		t.Fatalf("NewComment() error = %v", err)
	}
	if comment.ProjectID != "project-1" {
		t.Fatalf("expected trimmed project id, got %q", comment.ProjectID)
	}
	if comment.TargetType != CommentTargetTypeTask {
		t.Fatalf("expected normalized task target type, got %q", comment.TargetType)
	}
	if comment.TargetID != "item-1" {
		t.Fatalf("expected trimmed target id, got %q", comment.TargetID)
	}
	if comment.BodyMarkdown != "**done**" {
		t.Fatalf("expected trimmed markdown body, got %q", comment.BodyMarkdown)
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

// TestNewCommentDefaultsActorNameFromActorID verifies actor-name fallback behavior.
func TestNewCommentDefaultsActorNameFromActorID(t *testing.T) {
	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	comment, err := NewComment(CommentInput{
		ID:           "comment-1",
		ProjectID:    "project-1",
		TargetType:   CommentTargetTypeTask,
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
				TargetType:   CommentTargetType("column"),
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
				TargetType:   CommentTargetTypeTask,
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
		TargetType: CommentTargetType(" SUBTASK "),
		TargetID:   " t2 ",
	})
	if err != nil {
		t.Fatalf("NormalizeCommentTarget() error = %v", err)
	}
	if target.ProjectID != "p1" || target.TargetType != CommentTargetTypeSubtask || target.TargetID != "t2" {
		t.Fatalf("unexpected normalized target %#v", target)
	}
}

// TestNormalizeCommentTargetSupportsHierarchyNodes verifies branch/subphase target normalization.
func TestNormalizeCommentTargetSupportsHierarchyNodes(t *testing.T) {
	tests := []struct {
		name       string
		targetType CommentTargetType
		wantType   CommentTargetType
	}{
		{name: "branch", targetType: CommentTargetType(" BRANCH "), wantType: CommentTargetTypeBranch},
		{name: "subphase", targetType: CommentTargetType(" SUBPHASE "), wantType: CommentTargetTypeSubphase},
	}

	for _, tc := range tests {
		target, err := NormalizeCommentTarget(CommentTarget{
			ProjectID:  " p1 ",
			TargetType: tc.targetType,
			TargetID:   " item-1 ",
		})
		if err != nil {
			t.Fatalf("%s: NormalizeCommentTarget() error = %v", tc.name, err)
		}
		if target.TargetType != tc.wantType {
			t.Fatalf("%s: normalized target type = %q, want %q", tc.name, target.TargetType, tc.wantType)
		}
	}
}
