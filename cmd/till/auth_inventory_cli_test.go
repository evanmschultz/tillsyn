package main

import (
	"strings"
	"testing"
	"time"
)

// TestWriteAuthRequestListHuman renders a deterministic name-first request table.
func TestWriteAuthRequestListHuman(t *testing.T) {
	now := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	requests := []authRequestPayloadJSON{
		{
			ID:                  "req-b",
			State:               "pending",
			PrincipalID:         "review-b",
			PrincipalName:       "Zulu Review",
			PrincipalType:       "agent",
			PrincipalRole:       "builder",
			ClientID:            "client-b",
			ClientName:          "Till MCP STDIO",
			RequestedSessionTTL: "8h0m0s",
			Path:                "project/p2",
			RequestedByActor:    "lane-user",
			RequestedByType:     "user",
			CreatedAt:           now,
			ExpiresAt:           now.Add(8 * time.Hour),
		},
		{
			ID:                  "req-a",
			State:               "approved",
			PrincipalID:         "review-a",
			PrincipalName:       "Alpha Review",
			PrincipalType:       "agent",
			PrincipalRole:       "qa",
			ClientID:            "client-a",
			ClientName:          "Till MCP STDIO",
			RequestedSessionTTL: "2h0m0s",
			ApprovedSessionTTL:  "1h0m0s",
			Path:                "project/p1",
			ApprovedPath:        "project/p1/branch/narrowed",
			RequestedByActor:    "lane-orchestrator",
			RequestedByType:     "agent",
			IssuedSessionID:     "sess-a",
			CreatedAt:           now,
			ExpiresAt:           now.Add(2 * time.Hour),
		},
	}

	var out strings.Builder
	if err := writeAuthRequestListHuman(&out, requests); err != nil {
		t.Fatalf("writeAuthRequestListHuman() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Auth Requests",
		"NAME",
		"REQUEST ID",
		"STATE",
		"CLIENT",
		"REQUESTED PATH",
		"APPROVED PATH",
		"REQUESTED BY",
		"REQUESTED TTL",
		"APPROVED TTL",
		"RESULT SESSION",
		"Alpha Review [review-a] • qa",
		"Zulu Review [review-b] • builder",
		"req-a",
		"req-b",
		"sess-a",
		"project/p1/branch/narrowed",
		"project/p2",
		"1h",
		"8h",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in request list output, got %q", want, got)
		}
	}
	if strings.Index(got, "Alpha Review [review-a] • qa") > strings.Index(got, "Zulu Review [review-b] • builder") {
		t.Fatalf("expected name-first sorted output, got %q", got)
	}
}

// TestWriteAuthRequestDetailHuman renders requested and approved auth context without leaking secrets.
func TestWriteAuthRequestDetailHuman(t *testing.T) {
	now := time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)
	expires := now.Add(45 * time.Minute)
	resolvedAt := now.Add(10 * time.Minute)
	request := authRequestPayloadJSON{
		ID:                     "req-1",
		State:                  "approved",
		PrincipalID:            "review-1",
		PrincipalName:          "Review Agent",
		PrincipalType:          "agent",
		PrincipalRole:          "builder",
		ClientID:               "till-mcp-stdio",
		ClientName:             "Till MCP STDIO",
		RequestedSessionTTL:    "8h0m0s",
		ApprovedSessionTTL:     "2h0m0s",
		Path:                   "project/p1/branch/b1",
		ApprovedPath:           "project/p1/branch/narrowed",
		Reason:                 "inventory review",
		RequestedByActor:       "lane-user",
		RequestedByType:        "user",
		CreatedAt:              now,
		ExpiresAt:              now.Add(30 * time.Minute),
		ResolvedByActor:        "lane-orchestrator",
		ResolvedByType:         "agent",
		ResolvedAt:             &resolvedAt,
		ResolutionNote:         "approved for the narrower branch",
		IssuedSessionID:        "sess-1",
		IssuedSessionExpiresAt: &expires,
		IssuedSessionSecret:    "super-secret",
	}

	var out strings.Builder
	if err := writeAuthRequestDetailHuman(&out, request); err != nil {
		t.Fatalf("writeAuthRequestDetailHuman() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Auth Request",
		"name",
		"Review Agent [review-1] • builder",
		"request id",
		"req-1",
		"state",
		"approved",
		"client",
		"Till MCP STDIO",
		"requested path",
		"project/p1/branch/b1",
		"approved path",
		"project/p1/branch/narrowed",
		"requested ttl",
		"8h",
		"approved ttl",
		"2h",
		"requested by",
		"lane-user (user)",
		"reason",
		"inventory review",
		"issued session",
		"sess-1",
		"issued session expires",
		"resolved by",
		"lane-orchestrator (agent)",
		"resolved at",
		"resolution note",
		"approved for the narrower branch",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in request detail output, got %q", want, got)
		}
	}
	if strings.Contains(got, "super-secret") {
		t.Fatalf("expected auth detail output to omit session secret, got %q", got)
	}
}

// TestWriteAuthRequestResultHumanIncludesIssuedSecret renders the issued session secret only for mutation results.
func TestWriteAuthRequestResultHumanIncludesIssuedSecret(t *testing.T) {
	request := authRequestPayloadJSON{
		ID:                  "req-1",
		State:               "approved",
		ProjectID:           "p1",
		ScopeType:           "project",
		ScopeID:             "p1",
		PrincipalID:         "review-1",
		PrincipalName:       "Review Agent",
		PrincipalRole:       "builder",
		ClientID:            "till-mcp-stdio",
		ClientName:          "Till MCP STDIO",
		Path:                "project/p1",
		ApprovedPath:        "project/p1/branch/review",
		RequestedSessionTTL: "8h0m0s",
		ApprovedSessionTTL:  "2h0m0s",
		HasContinuation:     true,
		IssuedSessionID:     "sess-1",
		IssuedSessionSecret: "super-secret",
	}

	var out strings.Builder
	if err := writeAuthRequestResultHuman(&out, request); err != nil {
		t.Fatalf("writeAuthRequestResultHuman() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"Auth Request", "issued session", "sess-1", "issued session secret", "super-secret", "has continuation", "yes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in request result output, got %q", want, got)
		}
	}
}

// TestWriteAuthSessionListHuman renders a deterministic name-first session table.
func TestWriteAuthSessionListHuman(t *testing.T) {
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	revokedAt := now.Add(15 * time.Minute)
	sessions := []authSessionPayloadJSON{
		{
			SessionID:     "sess-b",
			State:         "active",
			ProjectID:     "p2",
			PrincipalID:   "review-b",
			PrincipalName: "Zulu Review",
			PrincipalType: "agent",
			PrincipalRole: "builder",
			ClientID:      "client-b",
			ClientName:    "Till MCP STDIO",
			ApprovedPath:  "project/p2",
			ExpiresAt:     now.Add(3 * time.Hour),
		},
		{
			SessionID:        "sess-a",
			State:            "revoked",
			ProjectID:        "p1",
			PrincipalID:      "review-a",
			PrincipalName:    "Alpha Review",
			PrincipalType:    "agent",
			PrincipalRole:    "qa",
			ClientID:         "client-a",
			ClientName:       "Till MCP STDIO",
			ApprovedPath:     "project/p1/branch/narrowed",
			ExpiresAt:        now.Add(2 * time.Hour),
			RevokedAt:        &revokedAt,
			RevocationReason: "cleanup",
		},
	}

	var out strings.Builder
	if err := writeAuthSessionListHuman(&out, sessions); err != nil {
		t.Fatalf("writeAuthSessionListHuman() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Auth Sessions",
		"NAME",
		"SESSION ID",
		"STATE",
		"CLIENT",
		"PROJECT",
		"APPROVED PATH",
		"EXPIRES",
		"REVOCATION",
		"Alpha Review [review-a] • qa",
		"Zulu Review [review-b] • builder",
		"sess-a",
		"sess-b",
		"revoked",
		"active",
		"cleanup",
		"project/p1/branch/narrowed",
		"project/p2",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in session list output, got %q", want, got)
		}
	}
	if strings.Index(got, "Alpha Review [review-a] • qa") > strings.Index(got, "Zulu Review [review-b] • builder") {
		t.Fatalf("expected name-first sorted output, got %q", got)
	}
}

// TestWriteAuthSessionDetailHuman renders one session detail block and includes the optional secret only when supplied.
func TestWriteAuthSessionDetailHuman(t *testing.T) {
	revokedAt := time.Date(2026, 3, 23, 12, 15, 0, 0, time.UTC)
	session := authSessionPayloadJSON{
		SessionID:        "sess-1",
		State:            "revoked",
		ProjectID:        "p1",
		AuthRequestID:    "req-1",
		ApprovedPath:     "project/p1/branch/review",
		PrincipalID:      "review-1",
		PrincipalName:    "Review Agent",
		PrincipalType:    "agent",
		PrincipalRole:    "builder",
		ClientID:         "till-mcp-stdio",
		ClientType:       "mcp-stdio",
		ClientName:       "Till MCP STDIO",
		ExpiresAt:        revokedAt.Add(time.Hour),
		RevokedAt:        &revokedAt,
		RevocationReason: "operator cleanup",
	}

	var out strings.Builder
	if err := writeAuthSessionDetailHuman(&out, session, "secret-1"); err != nil {
		t.Fatalf("writeAuthSessionDetailHuman() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Auth Session",
		"name",
		"Review Agent [review-1] • builder",
		"session id",
		"sess-1",
		"state",
		"revoked",
		"project",
		"p1",
		"auth request",
		"req-1",
		"approved path",
		"project/p1/branch/review",
		"session secret",
		"secret-1",
		"revocation reason",
		"operator cleanup",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in session detail output, got %q", want, got)
		}
	}
}

// TestWriteAuthRequestListHumanShowsCollisionSafeLabels verifies ids stay visible when names collide.
func TestWriteAuthRequestListHumanShowsCollisionSafeLabels(t *testing.T) {
	requests := []authRequestPayloadJSON{
		{
			ID:            "req-1",
			State:         "pending",
			PrincipalID:   "builder-a",
			PrincipalName: "Builder Lane",
			PrincipalRole: "builder",
			ClientID:      "client-a",
			Path:          "project/p1",
		},
		{
			ID:            "req-2",
			State:         "pending",
			PrincipalID:   "builder-b",
			PrincipalName: "Builder Lane",
			PrincipalRole: "builder",
			ClientID:      "client-b",
			Path:          "project/p1",
		},
	}

	var out strings.Builder
	if err := writeAuthRequestListHuman(&out, requests); err != nil {
		t.Fatalf("writeAuthRequestListHuman() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Builder Lane [builder-a] • builder",
		"Builder Lane [builder-b] • builder",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in collision-safe request list output, got %q", want, got)
		}
	}
}
