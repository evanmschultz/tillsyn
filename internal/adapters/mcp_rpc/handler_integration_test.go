package mcprpc

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	autent "github.com/evanmschultz/autent"
	"github.com/evanmschultz/tillsyn/internal/adapters/auth/autentauth"
	mcpcommon "github.com/evanmschultz/tillsyn/internal/adapters/mcp_common"
	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// newRealMCPAttentionHandlerForTest constructs one real auth-backed MCP handler plus its backing store.
func newRealMCPAttentionHandlerForTest(t *testing.T) (*Handler, *sqlite.Repository, *autentauth.Service, string) {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	nextID := 0
	svc := app.NewService(repo, func() string {
		nextID++
		return fmt.Sprintf("id-%03d", nextID)
	}, nil, app.ServiceConfig{
		AutoCreateProjectColumns: true,
	})
	project, err := svc.CreateProject(context.Background(), "Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	adapter := mcpcommon.NewAppServiceAdapter(svc, auth)
	handler, err := NewHandler(Config{}, adapter, adapter)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler, repo, auth, project.ID
}

// issueUserMCPTestSession issues one deterministic user session for real MCP auth-backed integration tests.
func issueUserMCPTestSession(t *testing.T, auth *autentauth.Service) autent.IssuedSession {
	t.Helper()

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	return issued
}

// approvedPathHandoffFixture stores one real MCP handler plus approved-path handoff fixtures.
// Post-Drop-1.75, auth scope mirrors kind and is always project-level for action-item-scoped rows
// (see internal/app/auth_scope.go authScopeContextFromActionItemLineage). The fixture therefore
// anchors approval at the project level and uses a second project for out-of-scope denial tests
// instead of the pre-collapse branch/phase narrowing.
type approvedPathHandoffFixture struct {
	handler             *Handler
	auth                *autentauth.Service
	repo                *sqlite.Repository
	projectID           string
	outOfScopeProjectID string
	branchID            string
	phaseID             string
	handoffID           string
	outOfScopeHandoffID string
}

// newApprovedPathHandoffFixture constructs one real MCP fixture for approved-path handoff updates.
func newApprovedPathHandoffFixture(t *testing.T) approvedPathHandoffFixture {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	nextID := 0
	service := app.NewService(repo, func() string {
		nextID++
		return fmt.Sprintf("id-%03d", nextID)
	}, nil, app.ServiceConfig{
		AutoCreateProjectColumns: true,
	})
	seedMCPOrphanKindsForTest(t, service)
	project, err := service.CreateProject(context.Background(), "Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	otherProject, err := service.CreateProject(context.Background(), "Other", "")
	if err != nil {
		t.Fatalf("CreateProject(other) error = %v", err)
	}
	columnID := firstMCPProjectColumnIDForTest(t, repo, project.ID)
	otherColumnID := firstMCPProjectColumnIDForTest(t, repo, otherProject.ID)
	branch, phase, actionItem := createMCPScopedActionItemChainForTest(t, service, project.ID, columnID)
	_, _, otherActionItem := createMCPScopedActionItemChainForTest(t, service, otherProject.ID, otherColumnID)
	handoff, err := service.CreateHandoff(context.Background(), app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: project.ID,
			BranchID:  branch.ID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   actionItem.ID,
		},
		SourceRole:  "builder",
		TargetRole:  "qa",
		Summary:     "handoff summary",
		NextAction:  "review",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff() error = %v", err)
	}
	outOfScopeHandoff, err := service.CreateHandoff(context.Background(), app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: otherProject.ID,
			ScopeType: domain.ScopeLevelActionItem,
			ScopeID:   otherActionItem.ID,
		},
		SourceRole:  "builder",
		TargetRole:  "qa",
		Summary:     "out-of-scope handoff",
		NextAction:  "review",
		CreatedBy:   "user-1",
		CreatedType: domain.ActorTypeUser,
	})
	if err != nil {
		t.Fatalf("CreateHandoff(out of scope) error = %v", err)
	}

	adapter := mcpcommon.NewAppServiceAdapter(service, auth)
	handler, err := NewHandler(Config{}, adapter, adapter)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return approvedPathHandoffFixture{
		handler:             handler,
		auth:                auth,
		repo:                repo,
		projectID:           project.ID,
		outOfScopeProjectID: otherProject.ID,
		branchID:            branch.ID,
		phaseID:             phase.ID,
		handoffID:           handoff.ID,
		outOfScopeHandoffID: outOfScopeHandoff.ID,
	}
}

// firstMCPProjectColumnIDForTest returns one auto-created project column for MCP integration fixtures.
func firstMCPProjectColumnIDForTest(t *testing.T, repo *sqlite.Repository, projectID string) string {
	t.Helper()

	columns, err := repo.ListColumns(context.Background(), projectID, true)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("ListColumns() returned no columns, want defaults")
	}
	return columns[0].ID
}

// seedMCPOrphanKindsForTest is retained for API compatibility with integration
// fixtures that used to seed branch/phase/subtask kind definitions. Post-Drop-1.75
// the built-in seed covers every valid kind, so this is now a no-op. Kept so
// existing call sites compile without churn.
func seedMCPOrphanKindsForTest(t *testing.T, svc *app.Service) {
	t.Helper()
	_ = svc
}

// createMCPScopedActionItemChainForTest creates one plan -> plan -> build chain for MCP auth-context tests.
// The chain's shape (3-level tree under one project) mirrors the pre-Drop-1.75 branch -> phase -> actionItem
// hierarchy for authorization traversal; the chain variables are still named branch/phase/actionItem to
// preserve the semantic role each level plays in approved-path assertions.
func createMCPScopedActionItemChainForTest(t *testing.T, service *app.Service, projectID, columnID string) (domain.ActionItem, domain.ActionItem, domain.ActionItem) {
	t.Helper()

	branch, err := service.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      projectID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		ColumnID:       columnID,
		Title:          "Branch",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(branch) error = %v", err)
	}
	phase, err := service.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      projectID,
		ParentID:       branch.ID,
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		ColumnID:       columnID,
		Title:          "Phase",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(phase) error = %v", err)
	}
	actionItem, err := service.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      projectID,
		ParentID:       phase.ID,
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		ColumnID:       columnID,
		Title:          "ActionItem",
		CreatedByActor: "user-1",
		CreatedByName:  "User One",
		UpdatedByActor: "user-1",
		UpdatedByName:  "User One",
		UpdatedByType:  domain.ActorTypeUser,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(actionItem) error = %v", err)
	}
	return branch, phase, actionItem
}

// issueApprovedPathMCPTestSession issues one MCP user session constrained to one project-scoped approved path.
// Post-Drop-1.75 auth scope collapses to project level for action-item rows, so branch/phase segments in the
// approved path would make the session narrower than any resolvable context. The branchID / phaseIDs parameters
// are retained for call-site compatibility but only the projectID contributes to the issued approved_path.
func issueApprovedPathMCPTestSession(t *testing.T, auth *autentauth.Service, projectID, branchID string, phaseIDs ...string) autent.IssuedSession {
	t.Helper()
	_ = branchID
	_ = phaseIDs

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
		Metadata: map[string]string{
			"approved_path": domain.AuthRequestPath{
				ProjectID: projectID,
			}.String(),
			"project_id": projectID,
		},
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	return issued
}

// TestHandlerAttentionMutationPersistsAuthenticatedAttribution verifies the real MCP mutation path persists authenticated attribution.
func TestHandlerAttentionMutationPersistsAuthenticatedAttribution(t *testing.T) {
	handler, repo, auth, projectID := newRealMCPAttentionHandlerForTest(t)
	issued := issueUserMCPTestSession(t, auth)

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, raiseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "till.attention_item", map[string]any{
		"operation":            "raise",
		"project_id":           projectID,
		"scope_type":           "project",
		"scope_id":             projectID,
		"kind":                 "risk_note",
		"summary":              "Raised by MCP auth",
		"session_id":           issued.Session.ID,
		"session_secret":       issued.Secret,
		"requires_user_action": false,
	}))
	raised := toolResultStructured(t, raiseResp.Result)
	raiseID, _ := raised["id"].(string)
	if raiseID == "" {
		t.Fatalf("raise attention id = empty, want persisted row id from %#v", raised)
	}

	var createdByActor string
	var createdByType string
	if err := repo.DB().QueryRowContext(
		context.Background(),
		`SELECT created_by_actor, created_by_type FROM attention_items WHERE id = ?`,
		raiseID,
	).Scan(&createdByActor, &createdByType); err != nil {
		t.Fatalf("QueryRow(created attribution) error = %v", err)
	}
	if createdByActor != "user-1" {
		t.Fatalf("created_by_actor = %q, want user-1", createdByActor)
	}
	if createdByType != "user" {
		t.Fatalf("created_by_type = %q, want user", createdByType)
	}

	_, resolveResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "till.attention_item", map[string]any{
		"operation":      "resolve",
		"id":             raiseID,
		"reason":         "approved",
		"session_id":     issued.Session.ID,
		"session_secret": issued.Secret,
	}))
	if resolveResp.Error != nil {
		t.Fatalf("resolve response error = %#v, want nil", resolveResp.Error)
	}

	var resolvedByActor string
	var resolvedByType string
	if err := repo.DB().QueryRowContext(
		context.Background(),
		`SELECT resolved_by_actor, resolved_by_type FROM attention_items WHERE id = ?`,
		raiseID,
	).Scan(&resolvedByActor, &resolvedByType); err != nil {
		t.Fatalf("QueryRow(resolved attribution) error = %v", err)
	}
	if resolvedByActor != "user-1" {
		t.Fatalf("resolved_by_actor = %q, want user-1", resolvedByActor)
	}
	if resolvedByType != "user" {
		t.Fatalf("resolved_by_type = %q, want user", resolvedByType)
	}
}

// TestHandlerUpdateHandoffResolvesApprovedPathContext verifies the real MCP transport now resolves by-id handoff scope before auth.
func TestHandlerUpdateHandoffResolvesApprovedPathContext(t *testing.T) {
	fixture := newApprovedPathHandoffFixture(t)
	issued := issueApprovedPathMCPTestSession(t, fixture.auth, fixture.projectID, fixture.branchID, fixture.phaseID)

	server := httptest.NewServer(fixture.handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, resp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(22, "till.handoff", map[string]any{
		"operation":       "update",
		"handoff_id":      fixture.handoffID,
		"status":          "resolved",
		"summary":         "resolved handoff",
		"next_action":     "none",
		"resolution_note": "done",
		"session_id":      issued.Session.ID,
		"session_secret":  issued.Secret,
	}))
	if resp.Error != nil {
		t.Fatalf("update_handoff response error = %#v, want nil", resp.Error)
	}
	if isError, _ := resp.Result["isError"].(bool); isError {
		t.Fatalf("update_handoff returned isError=true: %q", toolResultText(t, resp.Result))
	}

	var (
		status         string
		resolutionNote string
	)
	if err := fixture.repo.DB().QueryRowContext(
		context.Background(),
		`SELECT status, resolution_note FROM handoffs WHERE id = ?`,
		fixture.handoffID,
	).Scan(&status, &resolutionNote); err != nil {
		t.Fatalf("QueryRow(updated handoff) error = %v", err)
	}
	if status != "resolved" {
		t.Fatalf("handoff status = %q, want resolved", status)
	}
	if resolutionNote != "done" {
		t.Fatalf("handoff resolution_note = %q, want done", resolutionNote)
	}
}

// newRealMCPAuthRequestHandlerForTest constructs one real auth-backed MCP handler
// with AuthRequests and AuthBackend wired so auth-request lifecycle operations
// (create/approve/claim) work end-to-end. Returns the handler, the app.Service
// (for direct approve calls in tests), the autentauth.Service, and the project ID.
func newRealMCPAuthRequestHandlerForTest(t *testing.T) (*Handler, *app.Service, *autentauth.Service, string) {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	auth, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	nextID := 0
	svc := app.NewService(repo, func() string {
		nextID++
		return fmt.Sprintf("id-%03d", nextID)
	}, nil, app.ServiceConfig{
		AutoCreateProjectColumns: true,
		AuthRequests:             auth,
		AuthBackend:              auth,
	})
	project, err := svc.CreateProject(context.Background(), "Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	adapter := mcpcommon.NewAppServiceAdapter(svc, auth)
	handler, err := NewHandler(Config{}, adapter, adapter)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler, svc, auth, project.ID
}

// TestAuthRequestCreateWaitTimeoutBlocksAndWakesOnApproval verifies that
// till.auth_request operation=create with wait_timeout blocks until an
// approval arrives, then returns the approved state — proving E2E-8 is
// fixed at the MCP wire layer.
func TestAuthRequestCreateWaitTimeoutBlocksAndWakesOnApproval(t *testing.T) {
	handler, svc, _, projectID := newRealMCPAuthRequestHandlerForTest(t)

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	// approveAfter is the delay before the background goroutine approves the
	// pending request. The value is deliberately coarse (2s real-wall) so the
	// test stays robust on a loaded CI runner.
	const approveAfter = 2 * time.Second

	// approveErrCh carries any error from the background goroutine so the
	// main test body can surface it as a t.Fatal after the blocking call.
	approveErrCh := make(chan error, 1)

	go func() {
		time.Sleep(approveAfter)

		// Poll ListAuthRequests until the pending request appears. The
		// request is persisted before the live-wait begins, so it is
		// observable immediately after CreateAuthRequest writes to SQLite.
		var requestID string
		for attempt := 0; attempt < 20; attempt++ {
			requests, listErr := svc.ListAuthRequests(context.Background(), domain.AuthRequestListFilter{
				ProjectID: projectID,
				State:     domain.AuthRequestStatePending,
				Limit:     1,
			})
			if listErr != nil {
				approveErrCh <- fmt.Errorf("ListAuthRequests attempt %d: %w", attempt, listErr)
				return
			}
			if len(requests) > 0 {
				requestID = requests[0].ID
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if requestID == "" {
			approveErrCh <- fmt.Errorf("no pending auth request found after polling")
			return
		}

		_, approveErr := svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
			RequestID:    requestID,
			ResolvedBy:   "dev",
			ResolvedType: domain.ActorTypeUser,
		})
		approveErrCh <- approveErr
	}()

	// Call till.auth_request operation=create with wait_timeout. This blocks
	// until the goroutine approves the request or the 5s timeout fires.
	start := time.Now()
	_, createResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "till.auth_request", map[string]any{
		"operation":      "create",
		"path":           "project/" + projectID,
		"principal_id":   "builder-agent-1",
		"principal_type": "agent",
		"principal_role": "builder",
		"client_id":      "till-mcp-stdio",
		"reason":         "integration test wait_timeout",
		"wait_timeout":   "5s",
	}))
	elapsed := time.Since(start)

	// Surface background goroutine errors before other assertions.
	if approveErr := <-approveErrCh; approveErr != nil {
		t.Fatalf("background approve goroutine error = %v", approveErr)
	}

	// No protocol-level error.
	if createResp.Error != nil {
		t.Fatalf("create response protocol error = %#v, want nil", createResp.Error)
	}

	// Tool-level success (isError must be false or absent).
	if isErr, _ := createResp.Result["isError"].(bool); isErr {
		t.Fatalf("create response isError = true: %s", toolResultText(t, createResp.Result))
	}

	result := toolResultStructured(t, createResp.Result)

	// State must be approved after the goroutine's approval woke the live-waiter.
	if got, _ := result["state"].(string); got != "approved" {
		t.Fatalf("state = %q, want approved", got)
	}

	// resume_token must be non-empty (continuation ownership proof).
	if got, _ := result["resume_token"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("resume_token = empty, want non-empty")
	}

	// Elapsed time must reflect blocking until approval, not immediate return
	// and not a full timeout. Tolerance: [1500ms, 4500ms].
	const (
		minElapsed = 1500 * time.Millisecond
		maxElapsed = 4500 * time.Millisecond
	)
	if elapsed < minElapsed {
		t.Fatalf("elapsed = %v, want >= %v (call returned too fast — should have blocked)", elapsed, minElapsed)
	}
	if elapsed > maxElapsed {
		t.Fatalf("elapsed = %v, want <= %v (call timed out before approval arrived)", elapsed, maxElapsed)
	}

	t.Logf("wait_timeout blocking test: elapsed=%v, approved state confirmed, resume_token non-empty", elapsed)
}

// TestHandlerUpdateHandoffOutOfScopeApprovedPathDenied verifies the MCP transport still fails closed for out-of-scope approved-path sessions.
func TestHandlerUpdateHandoffOutOfScopeApprovedPathDenied(t *testing.T) {
	fixture := newApprovedPathHandoffFixture(t)
	issued := issueApprovedPathMCPTestSession(t, fixture.auth, fixture.projectID, fixture.branchID, fixture.phaseID)

	server := httptest.NewServer(fixture.handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, resp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(23, "till.handoff", map[string]any{
		"operation":       "update",
		"handoff_id":      fixture.outOfScopeHandoffID,
		"status":          "resolved",
		"summary":         "should fail",
		"next_action":     "none",
		"resolution_note": "denied",
		"session_id":      issued.Session.ID,
		"session_secret":  issued.Secret,
	}))
	if resp.Error != nil {
		t.Fatalf("update_handoff response error = %#v, want nil", resp.Error)
	}
	if isError, _ := resp.Result["isError"].(bool); !isError {
		t.Fatalf("update_handoff isError = %v, want true", resp.Result["isError"])
	}
	if got := toolResultText(t, resp.Result); !strings.HasPrefix(got, "auth_denied:") {
		t.Fatalf("update_handoff error text = %q, want auth_denied prefix", got)
	}
}
