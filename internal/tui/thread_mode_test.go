package tui

import (
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestCommentTargetTypeForKindCollapsesToActionItem verifies every valid kind maps to the ActionItem
// comment target after the Drop-1.75 kind collapse. The pre-collapse branch/phase target taxonomy
// is gone; the only surviving distinction is the project vs action-item split (project targets are
// not produced by this helper).
func TestCommentTargetTypeForKindCollapsesToActionItem(t *testing.T) {
	tests := []struct {
		name   string
		kind   domain.Kind
		want   domain.CommentTargetType
		wantOK bool
	}{
		{name: "plan kind", kind: domain.KindPlan, want: domain.CommentTargetTypeActionItem, wantOK: true},
		{name: "discussion kind", kind: domain.KindDiscussion, want: domain.CommentTargetTypeActionItem, wantOK: true},
		{name: "build kind", kind: domain.KindBuild, want: domain.CommentTargetTypeActionItem, wantOK: true},
		{name: "unknown kind", kind: domain.Kind("unknown"), want: "", wantOK: false},
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

// TestCommentTargetTypeForActionItemCollapsesToActionItem verifies the actionItem mapper also
// collapses to the single ActionItem target post-Drop-1.75.
func TestCommentTargetTypeForActionItemCollapsesToActionItem(t *testing.T) {
	tests := []struct {
		name       string
		actionItem domain.ActionItem
		want       domain.CommentTargetType
		wantOK     bool
	}{
		{
			name:       "plan kind",
			actionItem: domain.ActionItem{Kind: domain.KindPlan, Scope: domain.KindAppliesToPlan},
			want:       domain.CommentTargetTypeActionItem,
			wantOK:     true,
		},
		{
			name:       "discussion kind",
			actionItem: domain.ActionItem{Kind: domain.KindDiscussion, Scope: domain.KindAppliesToDiscussion},
			want:       domain.CommentTargetTypeActionItem,
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
