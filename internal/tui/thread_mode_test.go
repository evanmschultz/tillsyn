package tui

import (
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestCommentTargetTypeForWorkKindSupportsHierarchyKinds verifies branch/phase kind coverage.
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
		{name: "phase kind", kind: domain.WorkKindPhase, want: domain.CommentTargetTypePhase, wantOK: true},
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

// TestCommentTargetTypeForActionItemUsesScopeOverrides verifies scope-aware branch mapping.
func TestCommentTargetTypeForActionItemUsesScopeOverrides(t *testing.T) {
	tests := []struct {
		name       string
		actionItem domain.ActionItem
		want       domain.CommentTargetType
		wantOK     bool
	}{
		{
			name:       "branch scope on actionItem kind",
			actionItem: domain.ActionItem{Kind: domain.WorkKindActionItem, Scope: domain.KindAppliesToBranch},
			want:       domain.CommentTargetTypeBranch,
			wantOK:     true,
		},
		{
			name:       "phase remains phase",
			actionItem: domain.ActionItem{Kind: domain.WorkKindPhase, Scope: domain.KindAppliesToPhase},
			want:       domain.CommentTargetTypePhase,
			wantOK:     true,
		},
	}

	for _, tc := range tests {
		got, ok := commentTargetTypeForActionItem(tc.actionItem)
		if ok != tc.wantOK {
			t.Fatalf("%s: commentTargetTypeForActionItem() ok = %t, want %t", tc.name, ok, tc.wantOK)
		}
		if got != tc.want {
			t.Fatalf("%s: commentTargetTypeForActionItem() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
