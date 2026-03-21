package common

import (
	"errors"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestMCPHelperParsing verifies duration, continuation, and RFC3339 parsing helpers.
func TestMCPHelperParsing(t *testing.T) {
	t.Parallel()

	if got, err := parseOptionalDurationString(" 2h ", "ttl"); err != nil || got != 2*time.Hour {
		t.Fatalf("parseOptionalDurationString(2h) = %s, %v, want 2h nil", got, err)
	}
	if _, err := parseOptionalDurationString("nope", "ttl"); !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("parseOptionalDurationString(nope) error = %v, want ErrInvalidCaptureStateRequest", err)
	}

	continuation, err := parseContinuationJSON(`{"resume_tool":"till.capture_state","resume":{"path":"project/p1"}}`)
	if err != nil {
		t.Fatalf("parseContinuationJSON() error = %v", err)
	}
	if got, _ := continuation["resume_tool"].(string); got != "till.capture_state" {
		t.Fatalf("parseContinuationJSON() = %#v, want resume_tool", continuation)
	}
	resume, ok := continuation["resume"].(map[string]any)
	if !ok || resume["path"] != "project/p1" {
		t.Fatalf("parseContinuationJSON() resume = %#v, want nested path", continuation["resume"])
	}
	if _, err := parseContinuationJSON(`{"resume_tool":`); !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("parseContinuationJSON(bad) error = %v, want ErrInvalidCaptureStateRequest", err)
	}

	ts, err := parseOptionalRFC3339("2026-03-20T12:00:00Z")
	if err != nil || ts == nil || !ts.Equal(time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("parseOptionalRFC3339() = %#v, %v, want parsed UTC timestamp", ts, err)
	}
	if _, err := parseOptionalRFC3339("not-rfc3339"); !errors.Is(err, ErrInvalidCaptureStateRequest) {
		t.Fatalf("parseOptionalRFC3339(invalid) error = %v, want ErrInvalidCaptureStateRequest", err)
	}
}

// TestMCPHelperIdentityMapping verifies actor identity and kind-list helpers stay deterministic.
func TestMCPHelperIdentityMapping(t *testing.T) {
	t.Parallel()

	if got := requestedActorTypeFromPrincipalType("service"); got != domain.ActorTypeAgent {
		t.Fatalf("requestedActorTypeFromPrincipalType(service) = %q, want agent", got)
	}
	if got := requestedActorTypeFromPrincipalType("user"); got != domain.ActorTypeUser {
		t.Fatalf("requestedActorTypeFromPrincipalType(user) = %q, want user", got)
	}

	actorID, actorName := deriveMutationActorIdentity(ActorLeaseTuple{
		ActorID:         "",
		ActorName:       "",
		AgentName:       "Agent One",
		AgentInstanceID: "agent-1",
	})
	if actorID != "agent-1" || actorName != "Agent One" {
		t.Fatalf("deriveMutationActorIdentity() = %q / %q, want agent-1 / Agent One", actorID, actorName)
	}

	if got := normalizeActorType("AGENT"); got != domain.ActorTypeAgent {
		t.Fatalf("normalizeActorType(AGENT) = %q, want agent", got)
	}
	if got := normalizeActorType(""); got != domain.ActorTypeUser {
		t.Fatalf("normalizeActorType(\"\") = %q, want user", got)
	}
	if isValidActorType(domain.ActorType("service")) {
		t.Fatal("isValidActorType(service) = true, want false")
	}

	appliesTo := toKindAppliesToList([]string{"project", "phase"})
	if len(appliesTo) != 2 || appliesTo[1] != domain.KindAppliesTo("phase") {
		t.Fatalf("toKindAppliesToList() = %#v, want project/phase", appliesTo)
	}
	kindIDs := toKindIDList([]string{"bug", "feature"})
	if len(kindIDs) != 2 || kindIDs[0] != domain.KindID("bug") {
		t.Fatalf("toKindIDList() = %#v, want bug/feature", kindIDs)
	}
}

// TestMapAuthRequestAndCommentRecords verifies transport record mappers preserve key lifecycle fields.
func TestMapAuthRequestAndCommentRecords(t *testing.T) {
	t.Parallel()

	resolvedAt := time.Date(2026, 3, 20, 12, 30, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	record := mapAuthRequestRecord(domain.AuthRequest{
		ID:                     "req-1",
		ProjectID:              "p1",
		Path:                   "project/p1",
		PrincipalID:            "agent-1",
		PrincipalType:          "agent",
		ClientID:               "till-mcp-stdio",
		ClientType:             "mcp-stdio",
		RequestedSessionTTL:    2 * time.Hour,
		Reason:                 "manual review",
		Continuation:           map[string]any{"resume_tool": "till.capture_state", "resume": map[string]any{"path": "project/p1"}},
		State:                  domain.AuthRequestStateApproved,
		RequestedByActor:       "user-1",
		RequestedByType:        domain.ActorTypeUser,
		CreatedAt:              time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		ExpiresAt:              expiresAt,
		ResolvedByActor:        "user-1",
		ResolvedByType:         domain.ActorTypeUser,
		ResolvedAt:             &resolvedAt,
		ResolutionNote:         "approved",
		IssuedSessionID:        "sess-1",
		IssuedSessionSecret:    "secret-1",
		IssuedSessionExpiresAt: &expiresAt,
	})
	if got, _ := record.Continuation["resume_tool"].(string); record.ID != "req-1" || record.IssuedSessionID != "sess-1" || got != "till.capture_state" {
		t.Fatalf("mapAuthRequestRecord() = %#v, want persisted auth request fields", record)
	}

	comment := mapDomainCommentRecord(domain.Comment{
		ID:           "c1",
		ProjectID:    "p1",
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     "p1",
		BodyMarkdown: "# Summary\n\nmore details",
		ActorID:      "user-1",
		ActorName:    "User One",
		ActorType:    domain.ActorTypeUser,
		CreatedAt:    time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 20, 12, 5, 0, 0, time.UTC),
	})
	if comment.Summary != "Summary" || comment.ActorName != "User One" {
		t.Fatalf("mapDomainCommentRecord() = %#v, want summary extraction", comment)
	}
}
