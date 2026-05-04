package mcpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/auth/autentauth"
	servercommon "github.com/evanmschultz/tillsyn/internal/adapters/server/common"
	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stewardIntegrationFixture wires real sqlite + autent + app.Service +
// AppServiceAdapter for the Drop 3 droplet 3.22 integration suite. The
// fixture mirrors the production wiring used by cmd/till/main.go: column
// auto-create + STEWARD anchor seeding + per-numbered-drop level_2 finding
// + refinements-gate auto-generation. Each test claims its own fresh
// fixture (no parallel sharing) so the per-fixture sqlite tempdir keeps
// state isolated.
type stewardIntegrationFixture struct {
	adapter   *servercommon.AppServiceAdapter
	repo      *sqlite.Repository
	svc       *app.Service
	projectID string

	// dropID is the level_1 plan action_item the drop-orch creates with
	// drop_number=3. Its creation triggers seedDropFindingsAndGate, which
	// materializes the 5 STEWARD-owned level_2 findings + the refinements
	// gate confluence inside the drop's tree.
	dropID string

	// findingIDs maps the canonical title suffix (e.g.
	// "DROP_3_HYLLA_FINDINGS") to the auto-seeded action_item id. Tests
	// reach into the fixture rather than re-deriving the id at every call
	// site — the seeder's title shape is asserted in
	// internal/app/auto_generate_steward_test.go and pinned by the
	// canonicalSixSeeds fixture there.
	findingIDs map[string]string

	// gateID is the auto-seeded refinements-gate confluence
	// (DROP_3_REFINEMENTS_GATE_BEFORE_DROP_4) inside the drop's tree.
	gateID string

	// columnIDs holds the auto-created project columns keyed by lifecycle
	// state (todo / in_progress / complete / failed) so MoveActionItem
	// callers do not have to repeat the ListColumns lookup.
	columnIDs map[domain.LifecycleState]string
}

// newStewardIntegrationFixture builds the per-test stack. Auto-create
// columns + AutoSeedStewardAnchors are both true so the post-create
// invariants land deterministically without separate setup calls.
func newStewardIntegrationFixture(t *testing.T) stewardIntegrationFixture {
	t.Helper()

	ctx := context.Background()
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
	if err := auth.EnsureDogfoodPolicy(ctx); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	nextID := 0
	idGen := func() string {
		nextID++
		return fmt.Sprintf("steward-int-id-%03d", nextID)
	}
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	svc := app.NewService(repo, idGen, func() time.Time { return now }, app.ServiceConfig{
		AutoCreateProjectColumns: true,
		AutoSeedStewardAnchors:   true,
	})
	adapter := servercommon.NewAppServiceAdapter(svc, auth)

	bootstrap := servercommon.ActorLeaseTuple{
		ActorID:                  "bootstrap-user",
		ActorName:                "Bootstrap User",
		ActorType:                string(domain.ActorTypeUser),
		AuthRequestPrincipalType: "user",
	}
	project, err := adapter.CreateProject(ctx, servercommon.CreateProjectRequest{
		Name:        "Drop 3 STEWARD integration",
		Description: "fixture project for droplet 3.22",
		Actor:       bootstrap,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	columnIDs := map[domain.LifecycleState]string{}
	cols, err := svc.ListColumns(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	for _, col := range cols {
		state := stewardIntegrationColumnState(col.Name)
		if state == "" {
			continue
		}
		columnIDs[state] = col.ID
	}
	if columnIDs[domain.StateTodo] == "" || columnIDs[domain.StateInProgress] == "" || columnIDs[domain.StateComplete] == "" {
		t.Fatalf("auto-created columns missing canonical states (got %#v)", columnIDs)
	}

	dropOrch := stewardIntegrationDropOrchActor()
	drop, err := adapter.CreateActionItem(ctx, servercommon.CreateActionItemRequest{
		ProjectID:      project.ID,
		ColumnID:       columnIDs[domain.StateTodo],
		Kind:           string(domain.KindPlan),
		StructuralType: string(domain.StructuralTypeDrop),
		DropNumber:     3,
		Title:          "DROP_3",
		Description:    "Drop 3 — STEWARD-gated integration fixture.",
		Priority:       string(domain.PriorityMedium),
		Actor:          dropOrch,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(DROP_3) error = %v", err)
	}

	wantSuffixes := []string{
		"DROP_3_HYLLA_FINDINGS",
		"DROP_3_LEDGER_ENTRY",
		"DROP_3_WIKI_CHANGELOG_ENTRY",
		"DROP_3_REFINEMENTS_RAISED",
		"DROP_3_HYLLA_REFINEMENTS_RAISED",
	}
	allItems, err := svc.ListActionItems(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("ListActionItems() error = %v", err)
	}
	itemsByTitle := map[string]domain.ActionItem{}
	for _, item := range allItems {
		itemsByTitle[item.Title] = item
	}
	findingIDs := map[string]string{}
	for _, suffix := range wantSuffixes {
		match, ok := itemsByTitle[suffix]
		if !ok {
			t.Fatalf("expected auto-seeded finding %q after DROP_3 creation; got items %v", suffix, itemsByTitle)
		}
		findingIDs[suffix] = match.ID
	}
	gateTitle := "DROP_3_REFINEMENTS_GATE_BEFORE_DROP_4"
	gate, ok := itemsByTitle[gateTitle]
	if !ok {
		t.Fatalf("expected auto-seeded refinements-gate %q after DROP_3 creation", gateTitle)
	}

	return stewardIntegrationFixture{
		adapter:    adapter,
		repo:       repo,
		svc:        svc,
		projectID:  project.ID,
		dropID:     drop.ID,
		findingIDs: findingIDs,
		gateID:     gate.ID,
		columnIDs:  columnIDs,
	}
}

// stewardIntegrationColumnState maps an auto-created column name onto its
// canonical lifecycle state. The package-private helper
// actionItemLifecycleStateForColumnName lives in
// internal/adapters/server/common; mirroring the four canonical mappings
// here keeps this file's column lookups self-contained without exporting
// the original.
func stewardIntegrationColumnState(name string) domain.LifecycleState {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "_")) {
	case "todo", "to_do":
		return domain.StateTodo
	case "in_progress", "doing":
		return domain.StateInProgress
	case "complete", "done":
		return domain.StateComplete
	case "failed":
		return domain.StateFailed
	case "archived":
		return domain.StateArchived
	default:
		return ""
	}
}

// stewardIntegrationDropOrchActor returns the canonical drop-orch actor
// tuple used by every drop-orch-driven assertion in this suite. The actor
// is an agent-principal session (AuthRequestPrincipalType="agent") which
// the STEWARD owner-state-lock REJECTS for state-affecting mutations on
// STEWARD-owned items.
func stewardIntegrationDropOrchActor() servercommon.ActorLeaseTuple {
	return servercommon.ActorLeaseTuple{
		ActorID:                  "drop-orch-1",
		ActorName:                "DROP_3_ORCH",
		ActorType:                string(domain.ActorTypeUser),
		AuthRequestPrincipalType: "agent",
	}
}

// stewardIntegrationStewardActor returns the canonical STEWARD actor tuple
// — a steward-principal session (AuthRequestPrincipalType="steward")
// authorized to move STEWARD-owned items through state.
func stewardIntegrationStewardActor() servercommon.ActorLeaseTuple {
	return servercommon.ActorLeaseTuple{
		ActorID:                  "STEWARD",
		ActorName:                "STEWARD orch",
		ActorType:                string(domain.ActorTypeUser),
		AuthRequestPrincipalType: "steward",
	}
}

// TestStewardIntegrationDropOrchCreateAndUpdateAllowedButCannotMoveState
// covers Test 1 of droplet 3.22:
//
//   - drop-orch creates DROP_3 (already done by the fixture), the seeder
//     materializes 5 level_2 findings + the refinements-gate;
//   - drop-orch update(description=...) on a STEWARD-owned finding
//     SUCCEEDS (description-only is permitted by L1's permissive side);
//   - drop-orch move_state(state=complete) on the same finding REJECTS
//     with ErrAuthorizationDenied (the L1 state-affecting gate fires).
func TestStewardIntegrationDropOrchCreateAndUpdateAllowedButCannotMoveState(t *testing.T) {
	fixture := newStewardIntegrationFixture(t)
	ctx := context.Background()
	dropOrch := stewardIntegrationDropOrchActor()

	// Sanity: 5 STEWARD-owned level_2 findings are seeded with the right
	// drop_number + parent linkage.
	for suffix, id := range fixture.findingIDs {
		got, err := fixture.adapter.GetActionItem(ctx, id)
		if err != nil {
			t.Fatalf("GetActionItem(%s) error = %v", suffix, err)
		}
		if got.Owner != "STEWARD" {
			t.Fatalf("finding %s Owner = %q, want STEWARD", suffix, got.Owner)
		}
		if got.DropNumber != 3 {
			t.Fatalf("finding %s DropNumber = %d, want 3", suffix, got.DropNumber)
		}
		if got.ParentID == "" {
			t.Fatalf("finding %s ParentID = empty, want STEWARD anchor id", suffix)
		}
	}
	gate, err := fixture.adapter.GetActionItem(ctx, fixture.gateID)
	if err != nil {
		t.Fatalf("GetActionItem(gate) error = %v", err)
	}
	if gate.ParentID != fixture.dropID {
		t.Fatalf("gate ParentID = %q, want drop %q", gate.ParentID, fixture.dropID)
	}
	if gate.StructuralType != domain.StructuralTypeConfluence {
		t.Fatalf("gate StructuralType = %q, want confluence", gate.StructuralType)
	}
	if gate.Owner != "STEWARD" {
		t.Fatalf("gate Owner = %q, want STEWARD", gate.Owner)
	}
	// blocked_by must include the dropID + every finding id.
	blockedSet := map[string]struct{}{}
	for _, id := range gate.Metadata.BlockedBy {
		blockedSet[id] = struct{}{}
	}
	if _, ok := blockedSet[fixture.dropID]; !ok {
		t.Fatalf("gate blocked_by missing drop id %q (got %v)", fixture.dropID, gate.Metadata.BlockedBy)
	}
	for suffix, id := range fixture.findingIDs {
		if _, ok := blockedSet[id]; !ok {
			t.Fatalf("gate blocked_by missing finding %s id %q (got %v)", suffix, id, gate.Metadata.BlockedBy)
		}
	}

	hyllaFindingsID := fixture.findingIDs["DROP_3_HYLLA_FINDINGS"]

	// Description-only update by drop-orch on STEWARD-owned finding SUCCEEDS.
	updated, err := fixture.adapter.UpdateActionItem(ctx, servercommon.UpdateActionItemRequest{
		ActionItemID: hyllaFindingsID,
		Title:        "DROP_3_HYLLA_FINDINGS",
		Description:  "Drop-orch updated body — Hylla feedback rollup for Drop 3.",
		Actor:        dropOrch,
	})
	if err != nil {
		t.Fatalf("UpdateActionItem(drop-orch description-only) error = %v, want nil", err)
	}
	if !strings.Contains(updated.Description, "Drop-orch updated body") {
		t.Fatalf("UpdateActionItem() description = %q, want updated value", updated.Description)
	}

	// move_state(complete) by drop-orch on the same STEWARD-owned finding
	// REJECTS.
	moved, moveErr := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
		ActionItemID: hyllaFindingsID,
		State:        string(domain.StateComplete),
		Actor:        dropOrch,
	})
	if !errors.Is(moveErr, servercommon.ErrAuthorizationDenied) {
		t.Fatalf("MoveActionItemState(drop-orch complete on STEWARD-owned) error = %v, want ErrAuthorizationDenied (item=%#v)", moveErr, moved)
	}
	// State unchanged.
	after, getErr := fixture.adapter.GetActionItem(ctx, hyllaFindingsID)
	if getErr != nil {
		t.Fatalf("GetActionItem() after rejection error = %v", getErr)
	}
	if after.LifecycleState != domain.StateTodo {
		t.Fatalf("rejected move mutated lifecycle_state: got %q, want %q", after.LifecycleState, domain.StateTodo)
	}
}

// TestStewardIntegrationStewardCanMoveStateThroughComplete covers Test 2
// of droplet 3.22: a steward-principal session can move STEWARD-owned
// items through state. Closes DROP_3_HYLLA_FINDINGS as proof.
func TestStewardIntegrationStewardCanMoveStateThroughComplete(t *testing.T) {
	fixture := newStewardIntegrationFixture(t)
	ctx := context.Background()
	steward := stewardIntegrationStewardActor()

	hyllaFindingsID := fixture.findingIDs["DROP_3_HYLLA_FINDINGS"]
	moved, err := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
		ActionItemID: hyllaFindingsID,
		State:        string(domain.StateComplete),
		Actor:        steward,
	})
	if err != nil {
		t.Fatalf("MoveActionItemState(STEWARD complete on STEWARD-owned) error = %v", err)
	}
	if moved.LifecycleState != domain.StateComplete {
		t.Fatalf("MoveActionItemState() lifecycle_state = %q, want complete", moved.LifecycleState)
	}
}

// TestStewardIntegrationRefinementsGateCloseSucceedsWhenAllBlockersClear
// covers Test 3 of droplet 3.22: STEWARD closes every drop_number=3 item
// + the refinements-gate, then closes DROP_3 itself. The end-to-end path
// asserts the gate's blocked-by edges resolve cleanly when no drop_number
// stragglers remain.
func TestStewardIntegrationRefinementsGateCloseSucceedsWhenAllBlockersClear(t *testing.T) {
	fixture := newStewardIntegrationFixture(t)
	ctx := context.Background()
	steward := stewardIntegrationStewardActor()

	// Close all 5 STEWARD-owned level_2 findings.
	for suffix, id := range fixture.findingIDs {
		if _, err := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
			ActionItemID: id,
			State:        string(domain.StateComplete),
			Actor:        steward,
		}); err != nil {
			t.Fatalf("MoveActionItemState(STEWARD %s complete) error = %v", suffix, err)
		}
	}

	// Close the refinements-gate. SUCCEEDS because every direct child of
	// the gate (none) and every gate.blocked_by entry has reached terminal
	// state — the always-on parent-blocks-on-incomplete-child invariant
	// (Drop 4a Wave 1.7) finds no non-archived non-Complete children to
	// block on.
	gate, err := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
		ActionItemID: fixture.gateID,
		State:        string(domain.StateComplete),
		Actor:        steward,
	})
	if err != nil {
		t.Fatalf("MoveActionItemState(STEWARD gate complete) error = %v", err)
	}
	if gate.LifecycleState != domain.StateComplete {
		t.Fatalf("gate lifecycle_state = %q, want complete", gate.LifecycleState)
	}

	// No drop_number=3 stragglers remain → no safety-net attention item
	// was raised.
	openAttentions, err := fixture.svc.ListAttentionItems(ctx, app.ListAttentionItemsInput{
		Level: domain.LevelTupleInput{
			ProjectID: fixture.projectID,
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   fixture.projectID,
		},
		UnresolvedOnly: true,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems() error = %v", err)
	}
	for _, item := range openAttentions {
		if strings.HasPrefix(item.ID, "refinements-gate-forgotten::") {
			t.Fatalf("unexpected safety-net attention raised on clean gate close: id=%q summary=%q", item.ID, item.Summary)
		}
	}

	// Close DROP_3 itself. SUCCEEDS — every direct child (the gate) is
	// complete; the always-on parent-blocks-on-incomplete-child invariant
	// (Drop 4a Wave 1.7) is satisfied by the gate's terminal state.
	if _, err := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
		ActionItemID: fixture.dropID,
		State:        string(domain.StateComplete),
		Actor:        steward,
	}); err != nil {
		t.Fatalf("MoveActionItemState(STEWARD drop complete) error = %v", err)
	}
}

// TestStewardIntegrationDropOrchReparentRejected covers Test 4 of droplet
// 3.22 (per L8 — reparenting is identically gated to MoveActionItem):
// drop-orch attempting to reparent a STEWARD-owned item REJECTS.
func TestStewardIntegrationDropOrchReparentRejected(t *testing.T) {
	fixture := newStewardIntegrationFixture(t)
	ctx := context.Background()
	dropOrch := stewardIntegrationDropOrchActor()

	// Build a sibling target parent under DROP_3 that the drop-orch could
	// (in principle) try to reparent the finding under. The sibling itself
	// is non-STEWARD-owned so its creation bypasses the gate.
	sibling, err := fixture.adapter.CreateActionItem(ctx, servercommon.CreateActionItemRequest{
		ProjectID:      fixture.projectID,
		ParentID:       fixture.dropID,
		ColumnID:       fixture.columnIDs[domain.StateTodo],
		Kind:           string(domain.KindBuild),
		StructuralType: string(domain.StructuralTypeDroplet),
		Title:          "Sibling target parent",
		Actor:          dropOrch,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(sibling) error = %v", err)
	}

	hyllaFindingsID := fixture.findingIDs["DROP_3_HYLLA_FINDINGS"]
	if _, err := fixture.adapter.ReparentActionItem(ctx, servercommon.ReparentActionItemRequest{
		ActionItemID: hyllaFindingsID,
		ParentID:     sibling.ID,
		Actor:        dropOrch,
	}); !errors.Is(err, servercommon.ErrAuthorizationDenied) {
		t.Fatalf("ReparentActionItem(drop-orch on STEWARD-owned) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestStewardIntegrationDropOrchSupersedeRejected covers Test 5 of
// droplet 3.22 (per finding 5.C.13 — supersede gate). Pre-Drop-1 the
// supersede path on action items does not exist as a real state-mutating
// operation in the codebase (handoffs have a HandoffStatusSuperseded; the
// outcome adapter recognizes "superseded" as a metadata.outcome value;
// neither is a state transition the gate guards). The droplet's own
// pre-conditions / hints block calls this out: "If `supersede` is not yet
// a real path pre-Drop-1, gate the assertion behind a TODO + skip with
// `t.Skip` + a rationale comment that points at the future supersede
// landing — DO NOT invent it."
func TestStewardIntegrationDropOrchSupersedeRejected(t *testing.T) {
	t.Skip("supersede on action_item is not yet a real state-mutating path pre-Drop-1; the outcome adapter recognizes \"superseded\" as a metadata.outcome value but no SupersedeActionItem method exists on the adapter or app service. Re-enable when Drop 1 lands the supersede transition (per finding 5.C.13 — the gate must apply identically to MoveActionItem at that point).")
}

// TestStewardIntegrationDropOrchOwnerMutationRejected covers Test 6 of
// droplet 3.22 (per L1 field-level write guard): drop-orch attempting to
// mutate Owner on a STEWARD-owned item — including clearing it to the
// empty string — REJECTS with ErrAuthorizationDenied.
func TestStewardIntegrationDropOrchOwnerMutationRejected(t *testing.T) {
	fixture := newStewardIntegrationFixture(t)
	ctx := context.Background()
	dropOrch := stewardIntegrationDropOrchActor()

	hyllaFindingsID := fixture.findingIDs["DROP_3_HYLLA_FINDINGS"]
	clearedOwner := ""
	if _, err := fixture.adapter.UpdateActionItem(ctx, servercommon.UpdateActionItemRequest{
		ActionItemID: hyllaFindingsID,
		Title:        "DROP_3_HYLLA_FINDINGS",
		Owner:        &clearedOwner,
		Actor:        dropOrch,
	}); !errors.Is(err, servercommon.ErrAuthorizationDenied) {
		t.Fatalf("UpdateActionItem(drop-orch clearing Owner on STEWARD-owned) error = %v, want ErrAuthorizationDenied", err)
	}

	// Re-fetch and confirm Owner unchanged.
	after, err := fixture.adapter.GetActionItem(ctx, hyllaFindingsID)
	if err != nil {
		t.Fatalf("GetActionItem() after rejection error = %v", err)
	}
	if after.Owner != "STEWARD" {
		t.Fatalf("rejected Owner mutation persisted change: got %q, want STEWARD", after.Owner)
	}
}

// TestStewardIntegrationRefinementsGateForgottenRegression covers Test 7
// of droplet 3.22 + finding 5.C.11 — the refinements-gate forgetfulness
// regression. The flow:
//
//  1. Drop-orch creates a mid-drop refinement plan-item with
//     drop_number=3 AFTER the gate has been seeded. Because the gate's
//     blocked_by list was assembled at gate-creation time, the rogue item
//     is NOT in the gate's blocked_by edges. The gate close therefore
//     fires anyway.
//  2. Test asserts (a) the underlying parent-blocks-on-incomplete-child
//     invariant: a STEWARD attempt to close DROP_3 (level_1) while the
//     rogue drop_number=3 child sits non-terminal REJECTS with
//     ErrTransitionBlocked. The invariant must reject independently of
//     the safety-net warning per QA falsification §1.5.
//  3. Test asserts (b) the safety-net surface: when the gate moves to
//     complete with the rogue item still non-terminal, the auto-generator
//     materializes a refinements-gate-forgotten::<gate_id> attention_item
//     warning the dev that drop_number=3 items remained non-terminal at
//     gate-close.
//
// Drop 4a Wave 1.7 made the parent-blocks-on-incomplete-child invariant
// unconditional (the CompletionPolicy.RequireChildrenComplete bit was
// removed); no per-item opt-in is needed. The safety-net surface is
// independent of the invariant and fires unconditionally when the gate
// closes with stragglers.
func TestStewardIntegrationRefinementsGateForgottenRegression(t *testing.T) {
	fixture := newStewardIntegrationFixture(t)
	ctx := context.Background()
	dropOrch := stewardIntegrationDropOrchActor()
	steward := stewardIntegrationStewardActor()

	// Step 1: drop-orch creates a rogue mid-drop refinement plan-item with
	// drop_number=3 AFTER the gate has been seeded. The drop-orch is
	// agent-principal and the rogue item is non-STEWARD-owned, so the
	// gate does not fire on its create.
	rogue, err := fixture.adapter.CreateActionItem(ctx, servercommon.CreateActionItemRequest{
		ProjectID:      fixture.projectID,
		ParentID:       fixture.dropID,
		ColumnID:       fixture.columnIDs[domain.StateTodo],
		Kind:           string(domain.KindBuild),
		StructuralType: string(domain.StructuralTypeDroplet),
		DropNumber:     3,
		Title:          "DROP_3_ROGUE_REFINEMENT",
		Description:    "Mid-drop refinement plan-item created AFTER the gate.",
		Actor:          dropOrch,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(rogue mid-drop refinement) error = %v", err)
	}
	// Move the rogue item into in_progress so it is non-terminal at
	// gate-close + drop-close.
	if _, err := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
		ActionItemID: rogue.ID,
		State:        string(domain.StateInProgress),
		Actor:        dropOrch,
	}); err != nil {
		t.Fatalf("MoveActionItemState(rogue in_progress) error = %v", err)
	}

	// Sanity: the rogue item is NOT in the gate's blocked_by list. The
	// gate's blocked_by was assembled at gate-creation time; the rogue
	// item was created AFTER that, so the bug being regression-tested is
	// "the gate forgot to track the post-creation refinement."
	gate, err := fixture.adapter.GetActionItem(ctx, fixture.gateID)
	if err != nil {
		t.Fatalf("GetActionItem(gate) error = %v", err)
	}
	for _, id := range gate.Metadata.BlockedBy {
		if id == rogue.ID {
			t.Fatalf("gate blocked_by unexpectedly includes rogue mid-drop refinement %q (test premise broken)", rogue.ID)
		}
	}

	// Close all 5 STEWARD-owned level_2 findings so the gate's pinned
	// blocked_by entries clear.
	for suffix, id := range fixture.findingIDs {
		if _, err := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
			ActionItemID: id,
			State:        string(domain.StateComplete),
			Actor:        steward,
		}); err != nil {
			t.Fatalf("MoveActionItemState(STEWARD %s complete) error = %v", suffix, err)
		}
	}

	// Step 2: gate-close fires. The rogue item is NOT in blocked_by, so
	// the gate closes despite the in_progress drop_number=3 child. This
	// is the failure mode Test 7 documents: gate close paper-overs the
	// straggler.
	if _, err := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
		ActionItemID: fixture.gateID,
		State:        string(domain.StateComplete),
		Actor:        steward,
	}); err != nil {
		t.Fatalf("MoveActionItemState(STEWARD gate complete) error = %v (gate-close path is expected to succeed despite rogue straggler — that is the failure mode being regression-tested)", err)
	}

	// Step 2(b): the safety-net surface fired. An attention_item with id
	// "refinements-gate-forgotten::<gate_id>" must exist with the rogue
	// item id surfaced in its body.
	expectedAttentionID := "refinements-gate-forgotten::" + fixture.gateID
	openAttentions, err := fixture.svc.ListAttentionItems(ctx, app.ListAttentionItemsInput{
		Level: domain.LevelTupleInput{
			ProjectID: fixture.projectID,
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   fixture.projectID,
		},
		UnresolvedOnly: true,
	})
	if err != nil {
		t.Fatalf("ListAttentionItems() error = %v", err)
	}
	var safetyNet *domain.AttentionItem
	for i := range openAttentions {
		if openAttentions[i].ID == expectedAttentionID {
			safetyNet = &openAttentions[i]
			break
		}
	}
	if safetyNet == nil {
		t.Fatalf("expected safety-net attention_item id=%q after gate close with rogue straggler; got %d open items", expectedAttentionID, len(openAttentions))
	}
	if safetyNet.Kind != domain.AttentionKindRiskNote {
		t.Fatalf("safety-net attention Kind = %q, want risk_note", safetyNet.Kind)
	}
	if !safetyNet.RequiresUserAction {
		t.Fatal("safety-net attention RequiresUserAction = false, want true")
	}
	if !strings.Contains(safetyNet.BodyMarkdown, rogue.ID) {
		t.Fatalf("safety-net body did not surface rogue id %q; got body=%q", rogue.ID, safetyNet.BodyMarkdown)
	}
	if !strings.Contains(safetyNet.BodyMarkdown, "DROP_3_ROGUE_REFINEMENT") {
		t.Fatalf("safety-net body did not surface rogue title; got body=%q", safetyNet.BodyMarkdown)
	}

	// Step 2(a): the underlying parent-blocks-on-incomplete-child
	// invariant must reject DROP_3 close while the rogue child is still
	// non-terminal. Drop 4a Wave 1.7 made the invariant unconditional
	// (the CompletionPolicy.RequireChildrenComplete bit was removed); no
	// per-item opt-in is needed. The assertion pins the invariant
	// independently of the safety-net warning (per QA falsification §1.5
	// — the warning surface alone is insufficient evidence; the invariant
	// must reject the close on its own merits).
	if _, err := fixture.adapter.MoveActionItemState(ctx, servercommon.MoveActionItemStateRequest{
		ActionItemID: fixture.dropID,
		State:        string(domain.StateComplete),
		Actor:        steward,
	}); !errors.Is(err, domain.ErrTransitionBlocked) {
		t.Fatalf("MoveActionItemState(STEWARD DROP_3 complete) error = %v, want ErrTransitionBlocked from parent-blocks-on-incomplete-child", err)
	}
}

// =============================================================================
// Drop 4a Wave 3 W3.4 — Case (d): STEWARD cross-subtree approval (integration)
// =============================================================================
//
// Case (d) is integration-style — case (a)/(b)/(c)/(e) use stub services, but
// case (d) requires the live gate's persistent-parent ancestry walk to fire.
// We stand up a real sqlite + autent + app.Service + AppServiceAdapter,
// create a persistent STEWARD-owned action_item, mint a STEWARD-typed orch
// session via the autent IssueSession path, then drive an approve over
// JSON-RPC against a real handler.
//
// The fixture below mirrors the stewardCrossSubtreeFixture pattern from
// internal/app/auth_requests_test.go but extends it with the MCP handler
// + httptest harness so the cross-subtree exception is exercised end-to-end
// through the JSON-RPC wire format.

// stewardApproveFixture wires the full stack for case (d): real adapter,
// real STEWARD session, persistent STEWARD-owned ancestor, pending cross-orch
// subagent request rooted under it, and a live MCP handler over httptest.
type stewardApproveFixture struct {
	adapter        *servercommon.AppServiceAdapter
	repo           *sqlite.Repository
	auth           *autentauth.Service
	server         *httptest.Server
	projectID      string
	persistentID   string
	stewardSession orchSessionTuple // re-uses helper struct from handler_test.go
	pendingReq     domain.AuthRequest
}

// newStewardApproveFixture builds the case-(d) integration stack. Creates
// one project, one Backlog column, one STEWARD-owned persistent action_item,
// one STEWARD-typed auth session, and one pending cross-orch subagent
// request whose path roots under the persistent ancestor. Returns the
// fixture wired to a live httptest MCP handler.
func newStewardApproveFixture(t *testing.T) stewardApproveFixture {
	t.Helper()
	ctx := context.Background()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	auth, err := autentauth.NewSharedDB(autentauth.Config{
		DB:    repo.DB(),
		Clock: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := auth.EnsureDogfoodPolicy(ctx); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}

	projectID := "p-w34-d"
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: projectID, Name: "W3.4 case (d)"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	column, err := domain.NewColumn("col-w34-d", projectID, "Backlog", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	// Persistent STEWARD-owned ancestor — the requireStewardPersistentAncestor
	// walk's success target. Owner=STEWARD + Persistent=true (case-insensitive
	// owner match per auth_requests.go:561).
	persistentID := "PERSISTENT_STEWARD_W34_D"
	ancestor, err := domain.NewActionItemForTest(domain.ActionItemInput{
		ID:         persistentID,
		ProjectID:  projectID,
		Kind:       domain.KindRefinement,
		ColumnID:   column.ID,
		Title:      "STEWARD persistent refinement (W3.4 case d)",
		Owner:      "STEWARD",
		Persistent: true,
	}, now)
	if err != nil {
		t.Fatalf("NewActionItemForTest(persistent) error = %v", err)
	}
	if err := repo.CreateActionItem(ctx, ancestor); err != nil {
		t.Fatalf("CreateActionItem(persistent) error = %v", err)
	}

	// Mint a STEWARD-typed orch session. PrincipalType "steward" persists via
	// the "auth_request_principal_type" metadata key — the gate reads it via
	// AuthSession.AuthRequestPrincipalType (set by mapSessionView).
	stewardPrincipalID := "STEWARD"
	stewardClientID := "till-mcp-stdio-steward-w34"
	stewardIssued, err := auth.IssueSession(ctx, autentauth.IssueSessionInput{
		PrincipalID:   stewardPrincipalID,
		PrincipalType: "steward",
		PrincipalName: "STEWARD",
		ClientID:      stewardClientID,
		ClientType:    "mcp-stdio",
		ClientName:    "STEWARD MCP",
		TTL:           2 * time.Hour,
		Metadata: map[string]string{
			"principal_role": "orchestrator",
			"approved_path":  "project/" + projectID,
			"project_id":     projectID,
		},
	})
	if err != nil {
		t.Fatalf("IssueSession(steward) error = %v", err)
	}

	// Wire app.Service + AppServiceAdapter for the MCP handler.
	nextID := 0
	idGen := func() string {
		nextID++
		return fmt.Sprintf("w34-d-id-%03d", nextID)
	}
	svc := app.NewService(repo, idGen, func() time.Time { return now }, app.ServiceConfig{
		AuthRequests: auth,
		AuthBackend:  auth,
	})
	adapter := servercommon.NewAppServiceAdapter(svc, auth)

	// Create the cross-orch pending subagent request whose path roots under
	// the persistent ancestor. RequestedBy is a DIFFERENT orchestrator —
	// triggers the cross-orch branch in checkOrchSelfApprovalGate, which
	// then dispatches to requireStewardPersistentAncestor.
	requestPath := "project/" + projectID + "/branch/" + persistentID
	pendingReq, err := svc.CreateAuthRequest(ctx, app.CreateAuthRequestInput{
		Path:                requestPath,
		PrincipalID:         "PLANNER_AGENT_W34",
		PrincipalType:       "agent",
		PrincipalRole:       "planner",
		ClientID:            "till-mcp-stdio-planner",
		ClientType:          "mcp-stdio",
		RequestedSessionTTL: 2 * time.Hour,
		Reason:              "STEWARD cross-subtree approval test (W3.4 case d)",
		Continuation:        map[string]any{"resume_token": "resume-w34-d"},
		RequestedBy:         "OTHER_DROP_ORCH_W34",
		RequestedType:       domain.ActorTypeAgent,
		RequesterClientID:   "till-mcp-stdio-other-drop-orch",
		Timeout:             10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("CreateAuthRequest(planner) error = %v", err)
	}

	handler, err := NewHandler(Config{}, adapter, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	// MCP initialize handshake — tools/list later requires it.
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	return stewardApproveFixture{
		adapter:      adapter,
		repo:         repo,
		auth:         auth,
		server:       server,
		projectID:    projectID,
		persistentID: persistentID,
		stewardSession: orchSessionTuple{
			sessionID:       stewardIssued.Session.ID,
			sessionSecret:   stewardIssued.Secret,
			agentInstanceID: "STEWARD_INSTANCE_W34_D",
			leaseToken:      "STEWARD_LEASE_W34_D",
			principalID:     stewardPrincipalID,
		},
		pendingReq: pendingReq,
	}
}

// TestAuthRequestApproveStewardCrossSubtreeSucceedsForPersistentParent covers
// case (d): STEWARD-typed orch session at project scope approves a pending
// planner subagent request whose path roots under a persistent STEWARD-owned
// action_item. The cross-subtree exception fires through metadata-driven
// detection (Persistent + Owner=="STEWARD" — no hardcoded IDs). End-to-end
// JSON-RPC: response carries state=approved, audit fields populated with
// STEWARD's tuple, subagent session secret issued.
func TestAuthRequestApproveStewardCrossSubtreeSucceedsForPersistentParent(t *testing.T) {
	fixture := newStewardApproveFixture(t)

	args := approveArgs(fixture.pendingReq.ID, fixture.stewardSession)
	args["resolution_note"] = "STEWARD cross-subtree exception (W3.4 case d)"

	_, approveResp := postJSONRPC(t, fixture.server.Client(), fixture.server.URL, callToolRequest(2, "till.auth_request", args))
	if isError, _ := approveResp.Result["isError"].(bool); isError {
		t.Fatalf("case (d) isError = true, want false; result = %#v", approveResp.Result)
	}

	structured := toolResultStructured(t, approveResp.Result)
	requestRaw, ok := structured["request"].(map[string]any)
	if !ok {
		t.Fatalf("case (d) structured payload missing request: %#v", structured)
	}
	if got := requestRaw["state"].(string); got != "approved" {
		t.Fatalf("case (d) state = %q, want approved", got)
	}
	if got := requestRaw["approving_principal_id"].(string); got != fixture.stewardSession.principalID {
		t.Fatalf("case (d) approving_principal_id = %q, want %q (STEWARD)", got, fixture.stewardSession.principalID)
	}
	if got := requestRaw["approving_agent_instance_id"].(string); got != fixture.stewardSession.agentInstanceID {
		t.Fatalf("case (d) approving_agent_instance_id = %q, want %q", got, fixture.stewardSession.agentInstanceID)
	}
	if got := requestRaw["approving_lease_token"].(string); got != fixture.stewardSession.leaseToken {
		t.Fatalf("case (d) approving_lease_token = %q, want %q", got, fixture.stewardSession.leaseToken)
	}
	secret, _ := structured["session_secret"].(string)
	if secret == "" {
		t.Fatal("case (d) session_secret = empty, want non-empty issued subagent secret")
	}

	// DB-level round-trip: GetAuthRequest on the freshly-approved row must
	// return the same audit-trail values, proving the audit columns
	// persisted (4a.26) AND the cross-subtree exception path (4a.24)
	// landed them via the orch-self-approval cascade plumbing (W3.1).
	persisted, err := fixture.adapter.GetAuthRequest(context.Background(), fixture.pendingReq.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest() error = %v", err)
	}
	if persisted.State != "approved" {
		t.Fatalf("case (d) persisted state = %q, want approved", persisted.State)
	}
	if persisted.ApprovingPrincipalID != fixture.stewardSession.principalID {
		t.Fatalf("case (d) persisted ApprovingPrincipalID = %q, want %q",
			persisted.ApprovingPrincipalID, fixture.stewardSession.principalID)
	}
	if persisted.ApprovingAgentInstanceID != fixture.stewardSession.agentInstanceID {
		t.Fatalf("case (d) persisted ApprovingAgentInstanceID = %q, want %q",
			persisted.ApprovingAgentInstanceID, fixture.stewardSession.agentInstanceID)
	}
	if persisted.ApprovingLeaseToken != fixture.stewardSession.leaseToken {
		t.Fatalf("case (d) persisted ApprovingLeaseToken = %q, want %q",
			persisted.ApprovingLeaseToken, fixture.stewardSession.leaseToken)
	}

	// JSON-surface check: the persisted record's `approving_*` fields are
	// emitted (omitempty does NOT elide populated values).
	encoded, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("json.Marshal(persisted) error = %v", err)
	}
	for _, key := range []string{
		`"approving_principal_id":"STEWARD"`,
		`"approving_agent_instance_id":"STEWARD_INSTANCE_W34_D"`,
		`"approving_lease_token":"STEWARD_LEASE_W34_D"`,
	} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("case (d) persisted JSON = %s, want to contain %s", encoded, key)
		}
	}
}

// TestAuthRequestApproveProjectToggleDisabledRejectedIntegration is the
// spec-faithful integration pair to the unit-style
// TestAuthRequestApproveProjectToggleDisabledRejected (handler_test.go). The
// unit test stubs approveErr to prove the wire-format-to-error mapping; this
// integration test drives the real toggle gate end-to-end through the same
// stack case (d) uses (sqlite + autent + AppServiceAdapter + httptest
// handler).
//
// W3.4 plan spec (WAVE_3_PLAN.md:139) prescribes flipping
// Metadata.OrchSelfApprovalEnabled = *false on the project. We reuse the
// case-(d) STEWARD fixture and flip the toggle via repo.UpdateProject —
// this is the same direct-repo path the service-layer test uses
// (internal/app/auth_requests_test.go setProjectOrchSelfApprovalEnabled).
// The till.project(operation=update) MCP wire format is exercised separately
// by TestProjectMCPFirstClassFieldsRoundTrip (extended_tools_test.go); this
// test owns the toggle-gate-to-handler integration, not the project-update
// wire format.
//
// The toggle is a TOTAL backstop — it rejects even the STEWARD cross-subtree
// path that case (d) proves works under default-enabled metadata. This
// mirrors the service-layer steward_cross_subtree sub-case in
// TestApproveAuthRequestRejectsWhenProjectToggleDisabled. Asserts:
//   - HTTP-level: response carries isError=true.
//   - Error-text: contains the ErrOrchSelfApprovalDisabled sentinel
//     ("orch self-approval disabled by project metadata") and the wrap
//     fragment ("opted out of orch self-approval"). Substring match (not
//     prefix) so the test stays robust regardless of any future mapToolError
//     refinement that sharpens the error code.
//   - DB-level: persisted request state remains "pending" (toggle backstop
//     prevented the approval from being committed).
func TestAuthRequestApproveProjectToggleDisabledRejectedIntegration(t *testing.T) {
	fixture := newStewardApproveFixture(t)
	ctx := context.Background()

	// Flip the project's OrchSelfApprovalEnabled toggle to *false via the
	// repository directly (mirrors setProjectOrchSelfApprovalEnabled in
	// internal/app/auth_requests_test.go). The MCP wire format for this
	// update is exercised by TestProjectMCPFirstClassFieldsRoundTrip — this
	// test owns the toggle-gate-to-handler integration, not the project-
	// update wire format.
	project, err := fixture.repo.GetProject(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("GetProject(toggle setup) error = %v", err)
	}
	disabled := false
	project.Metadata.OrchSelfApprovalEnabled = &disabled
	if err := fixture.repo.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject(toggle=false) error = %v", err)
	}

	// Sanity: the toggle round-tripped to the persisted row.
	reloaded, err := fixture.repo.GetProject(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("GetProject(toggle verify) error = %v", err)
	}
	if reloaded.Metadata.OrchSelfApprovalIsEnabled() {
		t.Fatalf("toggle setup did not persist; OrchSelfApprovalIsEnabled() = true after UpdateProject(*false)")
	}

	args := approveArgs(fixture.pendingReq.ID, fixture.stewardSession)
	args["resolution_note"] = "STEWARD attempt with toggle disabled (W3.4 case e integration)"

	_, approveResp := postJSONRPC(t, fixture.server.Client(), fixture.server.URL, callToolRequest(2, "till.auth_request", args))
	isError, _ := approveResp.Result["isError"].(bool)
	if !isError {
		t.Fatalf("toggle-disabled integration isError = false, want true; result = %#v", approveResp.Result)
	}

	text := toolResultText(t, approveResp.Result)
	if !strings.Contains(text, "orch self-approval disabled by project metadata") {
		t.Fatalf("toggle-disabled integration error text = %q, want ErrOrchSelfApprovalDisabled sentinel message", text)
	}
	if !strings.Contains(text, "opted out of orch self-approval") {
		t.Fatalf("toggle-disabled integration error text = %q, want toggle-opt-out wrap fragment", text)
	}

	// DB-level backstop: the request must still be pending. The toggle
	// gate fires BEFORE the approve mutation persists, so a failed
	// approval leaves the row untouched. Catches the regression where a
	// future refactor accidentally lands the approval state-change
	// mid-gate.
	persisted, err := fixture.adapter.GetAuthRequest(ctx, fixture.pendingReq.ID)
	if err != nil {
		t.Fatalf("GetAuthRequest() error = %v", err)
	}
	if persisted.State != "pending" {
		t.Fatalf("toggle-disabled integration persisted state = %q, want pending (toggle backstop must prevent approval persist)", persisted.State)
	}
	if persisted.ApprovingPrincipalID != "" {
		t.Fatalf("toggle-disabled integration persisted ApprovingPrincipalID = %q, want empty (no audit trail on rejected approve)", persisted.ApprovingPrincipalID)
	}
}
