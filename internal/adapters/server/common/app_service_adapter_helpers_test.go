package common

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// TestNormalizeScopeTuple verifies scope tuple normalization and validation rules.
func TestNormalizeScopeTuple(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		projectID string
		scopeType string
		scopeID   string
		wantType  string
		wantID    string
		wantErr   error
	}{
		{name: "default project scope", projectID: "p1", wantType: ScopeTypeProject, wantID: "p1"},
		{name: "explicit branch scope", projectID: "p1", scopeType: ScopeTypeBranch, scopeID: "b1", wantType: ScopeTypeBranch, wantID: "b1"},
		{name: "project scope id mismatch", projectID: "p1", scopeType: ScopeTypeProject, scopeID: "p2", wantErr: ErrUnsupportedScope},
		{name: "missing non-project scope id", projectID: "p1", scopeType: ScopeTypePhase, wantErr: ErrUnsupportedScope},
		{name: "missing project id", wantErr: ErrInvalidCaptureStateRequest},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotProjectID, gotScopeType, gotScopeID, err := normalizeScopeTuple(tc.projectID, tc.scopeType, tc.scopeID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("normalizeScopeTuple() error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if gotProjectID != tc.projectID {
				t.Fatalf("normalizeScopeTuple() project_id = %q, want %q", gotProjectID, tc.projectID)
			}
			if gotScopeType != tc.wantType {
				t.Fatalf("normalizeScopeTuple() scope_type = %q, want %q", gotScopeType, tc.wantType)
			}
			if gotScopeID != tc.wantID {
				t.Fatalf("normalizeScopeTuple() scope_id = %q, want %q", gotScopeID, tc.wantID)
			}
		})
	}
}

// TestNormalizeRaiseAttentionItemRequest verifies request validation and normalization for mutation inputs.
func TestNormalizeRaiseAttentionItemRequest(t *testing.T) {
	t.Parallel()

	req, err := normalizeRaiseAttentionItemRequest(RaiseAttentionItemRequest{
		ProjectID:          "p1",
		ScopeType:          ScopeTypeProject,
		ScopeID:            "p1",
		Kind:               string(domain.AttentionKindApprovalRequired),
		Summary:            "needs review",
		BodyMarkdown:       "  details  ",
		RequiresUserAction: true,
		Actor: ActorLeaseTuple{
			ActorID:   "user-1",
			ActorType: string(domain.ActorTypeUser),
		},
	})
	if err != nil {
		t.Fatalf("normalizeRaiseAttentionItemRequest() error = %v", err)
	}
	if req.BodyMarkdown != "details" {
		t.Fatalf("normalizeRaiseAttentionItemRequest() body_markdown = %q, want details", req.BodyMarkdown)
	}
	if req.ScopeType != ScopeTypeProject || req.ScopeID != "p1" {
		t.Fatalf("normalizeRaiseAttentionItemRequest() scope = (%q, %q), want project/p1", req.ScopeType, req.ScopeID)
	}

	_, err = normalizeRaiseAttentionItemRequest(RaiseAttentionItemRequest{
		ProjectID: "p1",
		ScopeType: ScopeTypeProject,
		ScopeID:   "p1",
		Summary:   "missing kind",
	})
	if !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("normalizeRaiseAttentionItemRequest() missing kind error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestNormalizeResolveAttentionItemRequest verifies resolve request normalization keeps ids and trims reasons.
func TestNormalizeResolveAttentionItemRequest(t *testing.T) {
	t.Parallel()

	req, err := normalizeResolveAttentionItemRequest(ResolveAttentionItemRequest{
		ID:     "att-1",
		Reason: "  operator resolved  ",
	})
	if err != nil {
		t.Fatalf("normalizeResolveAttentionItemRequest() error = %v", err)
	}
	if req.ID != "att-1" || req.Reason != "operator resolved" {
		t.Fatalf("normalizeResolveAttentionItemRequest() = %#v, want trimmed values", req)
	}
}

// TestNormalizeAttentionStateFilter verifies supported state filters and invalid input handling.
func TestNormalizeAttentionStateFilter(t *testing.T) {
	t.Parallel()

	for _, state := range []string{"", AttentionStateOpen, AttentionStateAcknowledged, AttentionStateResolved} {
		if _, err := normalizeAttentionStateFilter(state); err != nil {
			t.Fatalf("normalizeAttentionStateFilter(%q) error = %v", state, err)
		}
	}
	if _, err := normalizeAttentionStateFilter("bad-state"); !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("normalizeAttentionStateFilter(bad-state) error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestBuildScopePathFromLevel verifies project and nested scope nodes are deterministic.
func TestBuildScopePathFromLevel(t *testing.T) {
	t.Parallel()

	projectOnly := buildScopePathFromLevel(domain.LevelTuple{
		ProjectID: "p1",
		ScopeType: domain.ScopeLevelProject,
		ScopeID:   "p1",
	}, "")
	if len(projectOnly) != 1 || projectOnly[0].Name != "p1" {
		t.Fatalf("buildScopePathFromLevel(project) = %#v, want project fallback name", projectOnly)
	}

	taskScope := buildScopePathFromLevel(domain.LevelTuple{
		ProjectID: "p1",
		ScopeType: domain.ScopeLevelTask,
		ScopeID:   "t1",
	}, "Inbox")
	if len(taskScope) != 2 {
		t.Fatalf("buildScopePathFromLevel(task) len = %d, want 2", len(taskScope))
	}
	if taskScope[1].ScopeType != string(domain.ScopeLevelTask) || taskScope[1].ScopeID != "t1" {
		t.Fatalf("buildScopePathFromLevel(task) tail = %#v, want task t1", taskScope[1])
	}
}

// TestBuildResumeHintsFromFollowUps verifies follow-up pointers map onto transport resume hints in stable order.
func TestBuildResumeHintsFromFollowUps(t *testing.T) {
	t.Parallel()

	hints := buildResumeHintsFromFollowUps(app.CaptureStateFollowUpPointers{
		ListAttentionItems:      "till.attention_item(operation=list,project_id=\"p1\")",
		ListProjectChangeEvents: "till.project(operation=list_change_events,project_id=\"p1\")",
		ListChildTasks:          "till.plan_item(operation=list,project_id=\"p1\",parent_id=\"t1\")",
	})
	if len(hints) != 3 {
		t.Fatalf("buildResumeHintsFromFollowUps() len = %d, want 3", len(hints))
	}
	if hints[0].Rel != "till.attention_item" || hints[2].Rel != "till.plan_item" {
		t.Fatalf("buildResumeHintsFromFollowUps() = %#v, want stable rel ordering", hints)
	}

	defaultHints := buildResumeHintsFromFollowUps(app.CaptureStateFollowUpPointers{})
	if len(defaultHints) != 1 || !strings.Contains(defaultHints[0].Note, "view=full") {
		t.Fatalf("buildResumeHintsFromFollowUps(default) = %#v, want capture_state fallback", defaultHints)
	}
}

// TestMapDomainAttentionItem verifies transport mapping preserves normalized state and timestamps.
func TestMapDomainAttentionItem(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 3, 20, 17, 12, 0, 0, time.UTC)
	resolvedAt := createdAt.Add(time.Hour)
	item := mapDomainAttentionItem(domain.AttentionItem{
		ID:                 "att-1",
		ProjectID:          "p1",
		ScopeType:          domain.ScopeLevelProject,
		ScopeID:            "p1",
		State:              domain.AttentionStateOpen,
		Kind:               domain.AttentionKindApprovalRequired,
		Summary:            "needs approval",
		BodyMarkdown:       "body",
		RequiresUserAction: true,
		CreatedAt:          createdAt,
		ResolvedAt:         &resolvedAt,
	})
	if item.State != string(domain.AttentionStateOpen) || item.Kind != string(domain.AttentionKindApprovalRequired) {
		t.Fatalf("mapDomainAttentionItem() normalized values = %#v", item)
	}
	if item.CreatedAt != createdAt.UTC() || item.ResolvedAt == nil || !item.ResolvedAt.Equal(resolvedAt.UTC()) {
		t.Fatalf("mapDomainAttentionItem() timestamps = %#v", item)
	}
}

// TestSummarizeCommentOverview verifies important markdown signals increment the important counter.
func TestSummarizeCommentOverview(t *testing.T) {
	t.Parallel()

	overview := summarizeCommentOverview([]domain.Comment{
		{BodyMarkdown: "plain note"},
		{BodyMarkdown: "Important: operator decision pending"},
		{BodyMarkdown: "urgent blocker requires user action"},
	})
	if overview.RecentCount != 3 {
		t.Fatalf("summarizeCommentOverview() recent = %d, want 3", overview.RecentCount)
	}
	if overview.ImportantCount != 2 {
		t.Fatalf("summarizeCommentOverview() important = %d, want 2", overview.ImportantCount)
	}
}

// TestMapAppError verifies transport sentinels remain stable for common app/domain failures.
func TestMapAppError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "not found", err: app.ErrNotFound, wantErr: ErrNotFound},
		{name: "invalid auth request ttl", err: domain.ErrInvalidAuthRequestTTL, wantErr: ErrInvalidCaptureStateRequest},
		{name: "lease required", err: domain.ErrMutationLeaseRequired, wantErr: ErrGuardrailViolation},
		{name: "bootstrap", err: ErrBootstrapRequired, wantErr: ErrBootstrapRequired},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := mapAppError("op", tc.err)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("mapAppError() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
