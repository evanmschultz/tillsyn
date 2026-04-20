package tui

import (
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestCommentTargetTypeForKindSupportsHierarchyKinds verifies branch/phase kind coverage.
func TestCommentTargetTypeForKindSupportsHierarchyKinds(t *testing.T) {
	tests := []struct {
		name   string
		kind   domain.Kind
		want   domain.CommentTargetType
		wantOK bool
	}{
		{
			name:   "branch kind",
			kind:   domain.Kind(domain.KindAppliesToBranch),
			want:   domain.CommentTargetTypeBranch,
			wantOK: true,
		},
		{name: "phase kind", kind: domain.KindPhase, want: domain.CommentTargetTypePhase, wantOK: true},
		{
			name:   "unknown kind",
			kind:   domain.Kind("unknown"),
			want:   "",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		got, ok := commentTargetTypeForKind(tc.kind)
		if ok != tc.wantOK {
			t.Fatalf("%s: commentTargetTypeForKind() ok = %t, want %t", tc.name, ok, tc.wantOK)
		}
		if got != tc.want {
			t.Fatalf("%s: commentTargetTypeForKind() = %q, want %q", tc.name, got, tc.want)
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
			actionItem: domain.ActionItem{Kind: domain.KindActionItem, Scope: domain.KindAppliesToBranch},
			want:       domain.CommentTargetTypeBranch,
			wantOK:     true,
		},
		{
			name:       "phase remains phase",
			actionItem: domain.ActionItem{Kind: domain.KindPhase, Scope: domain.KindAppliesToPhase},
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
