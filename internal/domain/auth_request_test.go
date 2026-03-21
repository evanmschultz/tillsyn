package domain

import (
	"errors"
	"testing"
	"time"
)

// TestParseAuthRequestPath verifies the project-rooted auth request path contract.
func TestParseAuthRequestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		raw         string
		wantProject string
		wantBranch  string
		wantPhases  []string
		wantScope   ScopeLevel
		wantScopeID string
		wantErr     bool
	}{
		{
			name:        "project root",
			raw:         "project/p1",
			wantProject: "p1",
			wantScope:   ScopeLevelProject,
			wantScopeID: "p1",
		},
		{
			name:        "branch scope",
			raw:         "project/p1/branch/b1",
			wantProject: "p1",
			wantBranch:  "b1",
			wantScope:   ScopeLevelBranch,
			wantScopeID: "b1",
		},
		{
			name:        "nested phase scope",
			raw:         "project/p1/branch/b1/phase/ph1/phase/ph2",
			wantProject: "p1",
			wantBranch:  "b1",
			wantPhases:  []string{"ph1", "ph2"},
			wantScope:   ScopeLevelPhase,
			wantScopeID: "ph2",
		},
		{name: "missing project prefix", raw: "branch/b1", wantErr: true},
		{name: "phase before branch", raw: "project/p1/phase/ph1", wantErr: true},
		{name: "dangling segment", raw: "project/p1/branch", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseAuthRequestPath(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseAuthRequestPath(%q) error = nil, want error", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAuthRequestPath(%q) error = %v", tc.raw, err)
			}
			if got.ProjectID != tc.wantProject {
				t.Fatalf("ProjectID = %q, want %q", got.ProjectID, tc.wantProject)
			}
			if got.BranchID != tc.wantBranch {
				t.Fatalf("BranchID = %q, want %q", got.BranchID, tc.wantBranch)
			}
			if got.ScopeType != tc.wantScope {
				t.Fatalf("ScopeType = %q, want %q", got.ScopeType, tc.wantScope)
			}
			if got.ScopeID != tc.wantScopeID {
				t.Fatalf("ScopeID = %q, want %q", got.ScopeID, tc.wantScopeID)
			}
			if len(got.PhaseIDs) != len(tc.wantPhases) {
				t.Fatalf("PhaseIDs len = %d, want %d", len(got.PhaseIDs), len(tc.wantPhases))
			}
			for idx, phaseID := range tc.wantPhases {
				if got.PhaseIDs[idx] != phaseID {
					t.Fatalf("PhaseIDs[%d] = %q, want %q", idx, got.PhaseIDs[idx], phaseID)
				}
			}
		})
	}
}

// TestAuthRequestNormalizationHelpers verifies state, filter, id, and metadata normalization helpers.
func TestAuthRequestNormalizationHelpers(t *testing.T) {
	t.Parallel()

	if got := NormalizeAuthRequestState(" PENDING "); got != AuthRequestStatePending {
		t.Fatalf("NormalizeAuthRequestState() = %q, want pending", got)
	}
	if !IsValidAuthRequestState(AuthRequestStateDenied) {
		t.Fatal("IsValidAuthRequestState(denied) = false, want true")
	}
	if IsValidAuthRequestState(AuthRequestState("unknown")) {
		t.Fatal("IsValidAuthRequestState(unknown) = true, want false")
	}

	filter, err := NormalizeAuthRequestListFilter(AuthRequestListFilter{ProjectID: " p1 ", State: " pending ", Limit: 10})
	if err != nil {
		t.Fatalf("NormalizeAuthRequestListFilter() error = %v", err)
	}
	if filter.ProjectID != "p1" || filter.State != AuthRequestStatePending || filter.Limit != 10 {
		t.Fatalf("NormalizeAuthRequestListFilter() = %#v, want trimmed pending filter", filter)
	}
	if _, err := NormalizeAuthRequestListFilter(AuthRequestListFilter{Limit: -1}); !errors.Is(err, ErrInvalidPosition) {
		t.Fatalf("NormalizeAuthRequestListFilter() error = %v, want ErrInvalidPosition", err)
	}
	if _, err := normalizeAuthRequestPrincipalType("service"); err != nil {
		t.Fatalf("normalizeAuthRequestPrincipalType(service) error = %v", err)
	}
	if _, err := normalizeAuthRequestPrincipalType("robot"); !errors.Is(err, ErrInvalidActorType) {
		t.Fatalf("normalizeAuthRequestPrincipalType(robot) error = %v, want ErrInvalidActorType", err)
	}

	ids := normalizeAuthRequestIDs([]string{" a ", "", "b", " ", "c"})
	if got, want := len(ids), 3; got != want {
		t.Fatalf("normalizeAuthRequestIDs() len = %d, want %d", got, want)
	}
	clone := cloneAuthRequestObjectMap(map[string]any{" k ": map[string]any{"nested": "value"}, "": "drop"})
	if len(clone) != 1 {
		t.Fatalf("cloneAuthRequestObjectMap() = %#v, want trimmed single entry", clone)
	}
	nested, ok := clone["k"].(map[string]any)
	if !ok || nested["nested"] != "value" {
		t.Fatalf("cloneAuthRequestObjectMap() nested = %#v, want preserved nested object", clone["k"])
	}
}

// TestAuthRequestPathRoundTripAndLevelTuple verifies canonical path rendering and scope tuple conversion.
func TestAuthRequestPathRoundTripAndLevelTuple(t *testing.T) {
	t.Parallel()

	path, err := ParseAuthRequestPath(" /project/p1/branch/b1/phase/ph1/phase/ph2/ ")
	if err != nil {
		t.Fatalf("ParseAuthRequestPath() error = %v", err)
	}
	if got := path.String(); got != "project/p1/branch/b1/phase/ph1/phase/ph2" {
		t.Fatalf("Path.String() = %q, want canonical path", got)
	}
	level, err := path.LevelTuple()
	if err != nil {
		t.Fatalf("LevelTuple() error = %v", err)
	}
	if level.ScopeType != ScopeLevelPhase || level.ScopeID != "ph2" || level.BranchID != "b1" {
		t.Fatalf("LevelTuple() = %#v, want branch/phase tuple", level)
	}
	normalized, err := (AuthRequestPath{ProjectID: "p1", BranchID: "b1"}).Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if normalized.ScopeType != ScopeLevelBranch || normalized.ScopeID != "b1" {
		t.Fatalf("Normalize() = %#v, want branch scope", normalized)
	}
}

// TestAuthRequestLifecycleTransitions verifies request creation, approval, denial, cancelation, and expiration branches.
func TestAuthRequestLifecycleTransitions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	req, err := NewAuthRequest(AuthRequestInput{
		ID:                  "req-1",
		Path:                AuthRequestPath{ProjectID: "p1", BranchID: "b1", PhaseIDs: []string{"ph1"}},
		PrincipalID:         "agent-1",
		PrincipalType:       "agent",
		PrincipalName:       "Agent One",
		ClientID:            "till-mcp-stdio",
		ClientType:          "mcp-stdio",
		ClientName:          "Till MCP STDIO",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "needs review",
		Continuation:        map[string]any{" resume_tool ": " tool ", "resume": map[string]any{"path": "project/p1"}},
		RequestedByActor:    "lane-user",
		RequestedByType:     ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	if req.State != AuthRequestStatePending || req.RequestedByActor != "lane-user" {
		t.Fatalf("NewAuthRequest() = %#v, want pending request", req)
	}
	resume, ok := req.Continuation["resume"].(map[string]any)
	if !ok || resume["path"] != "project/p1" {
		t.Fatalf("NewAuthRequest() continuation resume = %#v, want nested path", req.Continuation["resume"])
	}
	if got, _ := req.Continuation["resume_tool"].(string); got != " tool " {
		t.Fatalf("NewAuthRequest() continuation resume_tool = %q, want original string payload", got)
	}
	if !req.IsTerminal() && req.IsExpired(now) {
		// sanity branch only
	}

	if err := req.Approve("approver-1", ActorTypeUser, "approved", "sess-1", "secret-1", now.Add(time.Hour), now); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if req.State != AuthRequestStateApproved || req.IssuedSessionID != "sess-1" {
		t.Fatalf("Approve() = %#v, want approved session", req)
	}
	if !req.IsTerminal() {
		t.Fatal("approved request should be terminal")
	}

	deniedReq, _ := NewAuthRequest(AuthRequestInput{
		ID:                  "req-2",
		Path:                AuthRequestPath{ProjectID: "p1"},
		PrincipalID:         "user-1",
		ClientID:            "till-tui",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "needs review",
		RequestedByActor:    "lane-user",
		RequestedByType:     ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err := deniedReq.Deny("approver-2", ActorTypeUser, "denied", now); err != nil {
		t.Fatalf("Deny() error = %v", err)
	}
	if deniedReq.State != AuthRequestStateDenied || deniedReq.IssuedSessionID != "" {
		t.Fatalf("Deny() = %#v, want denied without session", deniedReq)
	}

	canceledReq, _ := NewAuthRequest(AuthRequestInput{
		ID:                  "req-3",
		Path:                AuthRequestPath{ProjectID: "p1"},
		PrincipalID:         "user-1",
		ClientID:            "till-tui",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "needs review",
		RequestedByActor:    "lane-user",
		RequestedByType:     ActorTypeUser,
		Timeout:             30 * time.Minute,
	}, now)
	if err := canceledReq.Cancel("approver-3", ActorTypeUser, "canceled", now); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if canceledReq.State != AuthRequestStateCanceled {
		t.Fatalf("Cancel() = %#v, want canceled", canceledReq)
	}

	expiredReq, _ := NewAuthRequest(AuthRequestInput{
		ID:                  "req-4",
		Path:                AuthRequestPath{ProjectID: "p1"},
		PrincipalID:         "user-1",
		ClientID:            "till-tui",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "needs review",
		RequestedByActor:    "lane-user",
		RequestedByType:     ActorTypeUser,
		Timeout:             time.Millisecond,
	}, now)
	if !expiredReq.IsExpired(now.Add(10 * time.Millisecond)) {
		t.Fatal("expected request to expire")
	}
	if err := expiredReq.Expire(now.Add(10 * time.Millisecond)); err != nil {
		t.Fatalf("Expire() error = %v", err)
	}
	if expiredReq.State != AuthRequestStateExpired || expiredReq.ResolutionNote != "timed_out" {
		t.Fatalf("Expire() = %#v, want expired timed_out", expiredReq)
	}
}

// TestAuthRequestLifecycleRejectsInvalidStates verifies creation and mutation guards fail closed on bad inputs.
func TestAuthRequestLifecycleRejectsInvalidStates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	if _, err := NewAuthRequest(AuthRequestInput{}, now); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("NewAuthRequest() error = %v, want ErrInvalidID", err)
	}
	if _, err := NewAuthRequest(AuthRequestInput{
		ID:                  "req-1",
		Path:                AuthRequestPath{ProjectID: "p1"},
		PrincipalID:         "agent-1",
		ClientID:            "client-1",
		RequestedSessionTTL: 0,
		Timeout:             time.Minute,
	}, now); !errors.Is(err, ErrInvalidAuthRequestTTL) {
		t.Fatalf("NewAuthRequest() error = %v, want ErrInvalidAuthRequestTTL", err)
	}

	req, err := NewAuthRequest(AuthRequestInput{
		ID:                  "req-2",
		Path:                AuthRequestPath{ProjectID: "p1"},
		PrincipalID:         "agent-1",
		ClientID:            "client-1",
		RequestedSessionTTL: 2 * time.Hour,
		Timeout:             time.Minute,
	}, now)
	if err != nil {
		t.Fatalf("NewAuthRequest() error = %v", err)
	}
	if err := req.Approve("approver", ActorType("robot"), "note", "sess", "secret", now.Add(time.Hour), now); !errors.Is(err, ErrInvalidActorType) {
		t.Fatalf("Approve() error = %v, want ErrInvalidActorType", err)
	}
	if err := req.Deny("approver", ActorType("robot"), "note", now); !errors.Is(err, ErrInvalidActorType) {
		t.Fatalf("Deny() error = %v, want ErrInvalidActorType", err)
	}
	if err := req.Cancel("approver", ActorType("robot"), "note", now); !errors.Is(err, ErrInvalidActorType) {
		t.Fatalf("Cancel() error = %v, want ErrInvalidActorType", err)
	}

	expired := req
	expired.State = AuthRequestStatePending
	expired.ExpiresAt = now.Add(-time.Minute)
	if err := expired.ensurePending(now); !errors.Is(err, ErrAuthRequestExpired) {
		t.Fatalf("ensurePending() error = %v, want ErrAuthRequestExpired", err)
	}
}
