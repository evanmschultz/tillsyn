package tui

import (
	"testing"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestCommentTargetTypeForWorkKindSupportsHierarchyKinds verifies branch/subphase kind coverage.
func TestCommentTargetTypeForWorkKindSupportsHierarchyKinds(t *testing.T) {
	tests := []struct {
		name   string
		kind   domain.WorkKind
		want   domain.CommentTargetType
		wantOK bool
	}{
		{
			name:   "branch kind",
			kind:   domain.WorkKind(domain.KindAppliesToBranch),
			want:   domain.CommentTargetTypeBranch,
			wantOK: true,
		},
		{
			name:   "subphase kind",
			kind:   domain.WorkKind(domain.KindAppliesToSubphase),
			want:   domain.CommentTargetTypeSubphase,
			wantOK: true,
		},
		{
			name:   "unknown kind",
			kind:   domain.WorkKind("unknown"),
			want:   "",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		got, ok := commentTargetTypeForWorkKind(tc.kind)
		if ok != tc.wantOK {
			t.Fatalf("%s: commentTargetTypeForWorkKind() ok = %t, want %t", tc.name, ok, tc.wantOK)
		}
		if got != tc.want {
			t.Fatalf("%s: commentTargetTypeForWorkKind() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestCommentTargetTypeForTaskUsesScopeOverrides verifies scope-aware subphase/branch mapping.
func TestCommentTargetTypeForTaskUsesScopeOverrides(t *testing.T) {
	tests := []struct {
		name   string
		task   domain.Task
		want   domain.CommentTargetType
		wantOK bool
	}{
		{
			name:   "subphase scope on phase kind",
			task:   domain.Task{Kind: domain.WorkKindPhase, Scope: domain.KindAppliesToSubphase},
			want:   domain.CommentTargetTypeSubphase,
			wantOK: true,
		},
		{
			name:   "branch scope on task kind",
			task:   domain.Task{Kind: domain.WorkKindTask, Scope: domain.KindAppliesToBranch},
			want:   domain.CommentTargetTypeBranch,
			wantOK: true,
		},
		{
			name:   "phase remains phase",
			task:   domain.Task{Kind: domain.WorkKindPhase, Scope: domain.KindAppliesToPhase},
			want:   domain.CommentTargetTypePhase,
			wantOK: true,
		},
	}

	for _, tc := range tests {
		got, ok := commentTargetTypeForTask(tc.task)
		if ok != tc.wantOK {
			t.Fatalf("%s: commentTargetTypeForTask() ok = %t, want %t", tc.name, ok, tc.wantOK)
		}
		if got != tc.want {
			t.Fatalf("%s: commentTargetTypeForTask() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
