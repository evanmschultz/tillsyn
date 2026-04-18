package common

import (
	"errors"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestComputeCaptureSummaryHashIgnoresCapturedAt verifies the hash excludes capture timestamp jitter.
func TestComputeCaptureSummaryHashIgnoresCapturedAt(t *testing.T) {
	summary := app.CaptureStateSummary{
		CapturedAt: time.Date(2026, 2, 25, 2, 32, 58, 600610000, time.UTC),
		Level: domain.LevelTuple{
			ProjectID: "p1",
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   "p1",
		},
		GoalOverview: "scope=project:p1 project=p1 view=full",
		AttentionOverview: app.CaptureStateAttentionOverview{
			UnresolvedCount: 1,
			Items: []app.CaptureStateAttentionItem{
				{
					ID:                 "a1",
					Kind:               domain.AttentionKindApprovalRequired,
					State:              domain.AttentionStateOpen,
					Summary:            "needs approval",
					RequiresUserAction: true,
					CreatedAt:          time.Date(2026, 2, 25, 2, 32, 54, 0, time.UTC),
				},
			},
		},
		WorkOverview: app.CaptureStateWorkOverview{
			TotalItems:      2,
			ActiveItems:     2,
			InProgressItems: 1,
			DoneItems:       0,
			BlockedItems:    0,
			OpenChildItems:  0,
		},
		FollowUpPointers: app.CaptureStateFollowUpPointers{
			ListAttentionItems:      "list_attention_items(project_id=\"p1\")",
			ListProjectChangeEvents: "list_project_change_events(project_id=\"p1\")",
		},
	}

	hashA, err := computeCaptureSummaryHash(summary)
	if err != nil {
		t.Fatalf("computeCaptureSummaryHash(first) error = %v", err)
	}
	summary.CapturedAt = time.Date(2026, 2, 25, 2, 32, 58, 607449000, time.UTC)
	hashB, err := computeCaptureSummaryHash(summary)
	if err != nil {
		t.Fatalf("computeCaptureSummaryHash(second) error = %v", err)
	}
	if hashA != hashB {
		t.Fatalf("hash mismatch when only captured_at changed: %q != %q", hashA, hashB)
	}
}

// TestComputeCaptureSummaryHashSortsAttentionItems verifies hash stability across item ordering differences.
func TestComputeCaptureSummaryHashSortsAttentionItems(t *testing.T) {
	older := app.CaptureStateAttentionItem{
		ID:                 "a1",
		Kind:               domain.AttentionKindBlocker,
		State:              domain.AttentionStateOpen,
		Summary:            "older",
		RequiresUserAction: false,
		CreatedAt:          time.Date(2026, 2, 25, 2, 31, 0, 0, time.UTC),
	}
	newer := app.CaptureStateAttentionItem{
		ID:                 "a2",
		Kind:               domain.AttentionKindApprovalRequired,
		State:              domain.AttentionStateOpen,
		Summary:            "newer",
		RequiresUserAction: true,
		CreatedAt:          time.Date(2026, 2, 25, 2, 32, 0, 0, time.UTC),
	}
	base := app.CaptureStateSummary{
		Level: domain.LevelTuple{
			ProjectID: "p1",
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   "t1",
		},
		GoalOverview: "scope=actionItem:t1 project=p1 view=summary",
		AttentionOverview: app.CaptureStateAttentionOverview{
			UnresolvedCount: 2,
		},
		WorkOverview: app.CaptureStateWorkOverview{
			TotalItems:      4,
			ActiveItems:     3,
			InProgressItems: 1,
			DoneItems:       1,
			BlockedItems:    1,
			FocusItemID:     "t1",
			OpenChildItems:  1,
		},
		FollowUpPointers: app.CaptureStateFollowUpPointers{
			ListAttentionItems:      "list_attention_items(project_id=\"p1\",scope_id=\"t1\")",
			ListProjectChangeEvents: "list_project_change_events(project_id=\"p1\")",
			ListChildActionItems:    "list_child_tasks(project_id=\"p1\",parent_id=\"t1\")",
		},
	}
	first := base
	first.AttentionOverview.Items = []app.CaptureStateAttentionItem{older, newer}
	second := base
	second.AttentionOverview.Items = []app.CaptureStateAttentionItem{newer, older}

	hashA, err := computeCaptureSummaryHash(first)
	if err != nil {
		t.Fatalf("computeCaptureSummaryHash(first order) error = %v", err)
	}
	hashB, err := computeCaptureSummaryHash(second)
	if err != nil {
		t.Fatalf("computeCaptureSummaryHash(second order) error = %v", err)
	}
	if hashA != hashB {
		t.Fatalf("hash mismatch for equivalent attention sets: %q != %q", hashA, hashB)
	}
}

// TestNormalizeCommonHelpers exercises the common transport normalization helpers and their failure modes.
func TestNormalizeCommonHelpers(t *testing.T) {
	t.Parallel()

	projectID, scopeType, scopeID, err := normalizeScopeTuple("p1", "", "")
	if err != nil {
		t.Fatalf("normalizeScopeTuple(project) error = %v", err)
	}
	if projectID != "p1" || scopeType != ScopeTypeProject || scopeID != "p1" {
		t.Fatalf("normalizeScopeTuple(project) = %q %q %q, want p1 project p1", projectID, scopeType, scopeID)
	}

	projectID, scopeType, scopeID, err = normalizeScopeTuple("p1", ScopeTypeBranch, "b1")
	if err != nil {
		t.Fatalf("normalizeScopeTuple(branch) error = %v", err)
	}
	if projectID != "p1" || scopeType != ScopeTypeBranch || scopeID != "b1" {
		t.Fatalf("normalizeScopeTuple(branch) = %q %q %q, want p1 branch b1", projectID, scopeType, scopeID)
	}

	if _, _, _, err := normalizeScopeTuple("p1", ScopeTypeActionItem, ""); err == nil {
		t.Fatal("normalizeScopeTuple(actionItem) expected error for missing scope_id")
	}

	if state, err := normalizeAttentionStateFilter("acknowledged"); err != nil || state != AttentionStateAcknowledged {
		t.Fatalf("normalizeAttentionStateFilter() = %q, %v, want acknowledged, nil", state, err)
	}
	if _, err := normalizeAttentionStateFilter("invalid"); !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("normalizeAttentionStateFilter(invalid) error = %v, want ErrInvalidCaptureStateRequest", err)
	}

	raise, err := normalizeRaiseAttentionItemRequest(RaiseAttentionItemRequest{
		ProjectID:    "p1",
		ScopeType:    ScopeTypeProject,
		ScopeID:      "p1",
		Kind:         string(domain.AttentionKindBlocker),
		Summary:      "  summary  ",
		BodyMarkdown: "  body  ",
	})
	if err != nil {
		t.Fatalf("normalizeRaiseAttentionItemRequest() error = %v", err)
	}
	if raise.Summary != "summary" || raise.BodyMarkdown != "body" {
		t.Fatalf("normalizeRaiseAttentionItemRequest() = %#v", raise)
	}
	if _, err := normalizeResolveAttentionItemRequest(ResolveAttentionItemRequest{}); err == nil {
		t.Fatal("normalizeResolveAttentionItemRequest() expected error for empty id")
	}
}

// TestMapAppErrorAndCommentMarkdown exercises error mapping and comment-body helpers used by common transport adapters.
func TestMapAppErrorAndCommentMarkdown(t *testing.T) {
	t.Parallel()

	if got := buildCommentBodyMarkdown("summary", "details"); got != "summary\n\ndetails" {
		t.Fatalf("buildCommentBodyMarkdown() = %q, want summary\\n\\ndetails", got)
	}
	if got := buildCommentBodyMarkdown("summary", ""); got != "summary" {
		t.Fatalf("buildCommentBodyMarkdown(summary only) = %q, want summary", got)
	}
	if got := commentSummaryFromMarkdown("  # heading\n\nbody line"); got != "heading" {
		t.Fatalf("commentSummaryFromMarkdown() = %q, want heading", got)
	}

	if err := mapAppError("op", app.ErrNotFound); !errors.Is(err, ErrNotFound) {
		t.Fatalf("mapAppError(app not found) = %v, want ErrNotFound", err)
	}
	if err := mapAppError("op", domain.ErrInvalidID); !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("mapAppError(invalid id) = %v, want ErrInvalidCaptureStateRequest", err)
	}
	if err := mapAppError("op", domain.ErrMutationLeaseRequired); !errors.Is(err, ErrGuardrailViolation) {
		t.Fatalf("mapAppError(lease required) = %v, want ErrGuardrailViolation", err)
	}
}
