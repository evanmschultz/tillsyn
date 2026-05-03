package common

import (
	"context"
	"errors"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestAssertOwnerStateGateMoveActionItemStateAgentRejected verifies the Drop 3
// droplet 3.19 STEWARD owner-state-lock blocks state transitions on
// Owner="STEWARD" items when the calling session's
// AuthRequestPrincipalType is non-steward (drop-orch as agent), and that the
// item's state column is unchanged on rejection.
func TestAssertOwnerStateGateMoveActionItemStateAgentRejected(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	stewardGated := newStewardGatedActionItem(t, fixture, "")

	rejectingActor := stewardGatedActor("agent")
	moved, err := fixture.adapter.MoveActionItemState(ctx, MoveActionItemStateRequest{
		ActionItemID: stewardGated.ID,
		State:        "in_progress",
		Actor:        rejectingActor,
	})
	if !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("MoveActionItemState(agent on STEWARD-owned) error = %v, want ErrAuthorizationDenied (item=%#v)", err, moved)
	}
	// State column unchanged — re-fetch and assert.
	after, getErr := fixture.adapter.GetActionItem(ctx, stewardGated.ID)
	if getErr != nil {
		t.Fatalf("GetActionItem() error = %v", getErr)
	}
	if after.LifecycleState != stewardGated.LifecycleState {
		t.Fatalf("MoveActionItemState rejection mutated lifecycle_state: before=%q after=%q", stewardGated.LifecycleState, after.LifecycleState)
	}
	if after.ColumnID != stewardGated.ColumnID {
		t.Fatalf("MoveActionItemState rejection mutated column_id: before=%q after=%q", stewardGated.ColumnID, after.ColumnID)
	}
}

// TestAssertOwnerStateGateMoveActionItemStateStewardSucceeds verifies a
// steward-principal session can move STEWARD-owned items through state.
func TestAssertOwnerStateGateMoveActionItemStateStewardSucceeds(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	stewardGated := newStewardGatedActionItem(t, fixture, "")

	stewardActor := stewardGatedActor("steward")
	moved, err := fixture.adapter.MoveActionItemState(ctx, MoveActionItemStateRequest{
		ActionItemID: stewardGated.ID,
		State:        "in_progress",
		Actor:        stewardActor,
	})
	if err != nil {
		t.Fatalf("MoveActionItemState(steward on STEWARD-owned) error = %v", err)
	}
	if moved.LifecycleState != domain.StateInProgress {
		t.Fatalf("MoveActionItemState(steward) lifecycle_state = %q, want %q", moved.LifecycleState, domain.StateInProgress)
	}
}

// TestAssertOwnerStateGateMoveActionItemStateNonStewardOwnerSucceeds verifies
// the gate only fires on Owner="STEWARD" items — non-STEWARD owners
// (including the empty/zero-value default) bypass the gate entirely.
func TestAssertOwnerStateGateMoveActionItemStateNonStewardOwnerSucceeds(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	// Owner left empty — the dominant case; gate must NOT fire.
	plain := newStewardGatedActionItem(t, fixture, "" /*ownerOverride*/)
	plain.Owner = "" // clear the default STEWARD value to simulate a normal item
	if err := fixture.repo.UpdateActionItem(ctx, plain); err != nil {
		t.Fatalf("repo.UpdateActionItem() clear owner error = %v", err)
	}

	rejectingActor := stewardGatedActor("agent")
	moved, err := fixture.adapter.MoveActionItemState(ctx, MoveActionItemStateRequest{
		ActionItemID: plain.ID,
		State:        "in_progress",
		Actor:        rejectingActor,
	})
	if err != nil {
		t.Fatalf("MoveActionItemState(agent on non-STEWARD-owned) error = %v", err)
	}
	if moved.LifecycleState != domain.StateInProgress {
		t.Fatalf("MoveActionItemState(agent on non-STEWARD-owned) lifecycle_state = %q, want %q", moved.LifecycleState, domain.StateInProgress)
	}
}

// TestAssertOwnerStateGateMoveActionItemAgentRejected verifies the
// column-only MoveActionItem path is gated identically to MoveActionItemState
// (per finding 5.C.1 — the column-only path now adds an explicit pre-fetch).
func TestAssertOwnerStateGateMoveActionItemAgentRejected(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	stewardGated := newStewardGatedActionItem(t, fixture, "")

	// Build a target column to move into — pick the existing in-progress
	// column the fixture seeded.
	columns, err := fixture.svc.ListColumns(ctx, stewardGated.ProjectID, true)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	var inProgressColumnID string
	for _, column := range columns {
		if actionItemLifecycleStateForColumnName(column.Name) == domain.StateInProgress {
			inProgressColumnID = column.ID
			break
		}
	}
	if inProgressColumnID == "" {
		t.Fatalf("no in_progress column seeded in fixture")
	}

	rejectingActor := stewardGatedActor("agent")
	moved, err := fixture.adapter.MoveActionItem(ctx, MoveActionItemRequest{
		ActionItemID: stewardGated.ID,
		ToColumnID:   inProgressColumnID,
		Actor:        rejectingActor,
	})
	if !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("MoveActionItem(agent on STEWARD-owned, column-only) error = %v, want ErrAuthorizationDenied (item=%#v)", err, moved)
	}
	// Column unchanged.
	after, getErr := fixture.adapter.GetActionItem(ctx, stewardGated.ID)
	if getErr != nil {
		t.Fatalf("GetActionItem() error = %v", getErr)
	}
	if after.ColumnID != stewardGated.ColumnID {
		t.Fatalf("MoveActionItem rejection mutated column_id: before=%q after=%q", stewardGated.ColumnID, after.ColumnID)
	}
}

// TestAssertOwnerStateGateUpdateActionItemDescriptionOnlyAgentSucceeds
// verifies the L1 field-level write guard's permissive side: an agent
// updating description/title/metadata only on a STEWARD-owned item is
// explicitly allowed (PLAN.md § 19.3 bullet 7).
func TestAssertOwnerStateGateUpdateActionItemDescriptionOnlyAgentSucceeds(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	stewardGated := newStewardGatedActionItem(t, fixture, "")

	rejectingActor := stewardGatedActor("agent")
	updated, err := fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: stewardGated.ID,
		Title:        "STEWARD parent — agent updated description",
		Description:  "Drop-orchs may write description/details/metadata on STEWARD-owned items.",
		Actor:        rejectingActor,
	})
	if err != nil {
		t.Fatalf("UpdateActionItem(agent description-only on STEWARD-owned) error = %v", err)
	}
	if updated.Title != "STEWARD parent — agent updated description" {
		t.Fatalf("UpdateActionItem() title = %q, want updated value", updated.Title)
	}
}

// TestAssertOwnerStateGateUpdateActionItemOwnerMutationAgentRejected verifies
// the L1 field-level write guard's restrictive side: an agent attempting to
// mutate the Owner field away from "STEWARD" is REJECTED.
func TestAssertOwnerStateGateUpdateActionItemOwnerMutationAgentRejected(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	stewardGated := newStewardGatedActionItem(t, fixture, "")

	clearedOwner := ""
	rejectingActor := stewardGatedActor("agent")
	if _, err := fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: stewardGated.ID,
		Title:        stewardGated.Title,
		Owner:        &clearedOwner,
		Actor:        rejectingActor,
	}); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("UpdateActionItem(agent clearing Owner) error = %v, want ErrAuthorizationDenied", err)
	}

	mutatedOwner := "ROGUE"
	if _, err := fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: stewardGated.ID,
		Title:        stewardGated.Title,
		Owner:        &mutatedOwner,
		Actor:        rejectingActor,
	}); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("UpdateActionItem(agent setting Owner=ROGUE) error = %v, want ErrAuthorizationDenied", err)
	}

	// Steward principal CAN change the Owner. Per droplet 3.21's MCP-
	// surface plumbing, the Owner field now flows from
	// UpdateActionItemRequest → UpdateActionItemInput → actionItem.Owner via
	// the service-side mutation block; the gate-permission check must be
	// paired with a re-fetch that proves the field actually mutated. Pre-
	// 3.21 this test asserted only the gate permission (per the 3.19
	// falsification NIT) — it cannot prove the field actually changed
	// because Owner was not yet wired through the service layer.
	stewardActor := stewardGatedActor("steward")
	if _, err := fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: stewardGated.ID,
		Title:        stewardGated.Title,
		Owner:        &mutatedOwner,
		Actor:        stewardActor,
	}); err != nil {
		t.Fatalf("UpdateActionItem(steward changing Owner) error = %v", err)
	}
	after, getErr := fixture.adapter.GetActionItem(ctx, stewardGated.ID)
	if getErr != nil {
		t.Fatalf("GetActionItem() after steward Owner mutation error = %v", getErr)
	}
	if after.Owner != mutatedOwner {
		t.Fatalf("UpdateActionItem(steward) did not persist Owner: got %q, want %q", after.Owner, mutatedOwner)
	}
}

// TestAssertOwnerStateGateUpdateActionItemDropNumberMutationAgentRejected
// verifies DropNumber is gated identically to Owner under the L1 field-level
// write guard.
func TestAssertOwnerStateGateUpdateActionItemDropNumberMutationAgentRejected(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	stewardGated := newStewardGatedActionItem(t, fixture, "")
	if stewardGated.DropNumber == 0 {
		t.Fatalf("test fixture must seed a non-zero DropNumber so the mutation comparison is meaningful")
	}

	rejectingActor := stewardGatedActor("agent")
	mutatedDropNumber := stewardGated.DropNumber + 1
	if _, err := fixture.adapter.UpdateActionItem(ctx, UpdateActionItemRequest{
		ActionItemID: stewardGated.ID,
		Title:        stewardGated.Title,
		DropNumber:   &mutatedDropNumber,
		Actor:        rejectingActor,
	}); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("UpdateActionItem(agent mutating DropNumber) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestAssertOwnerStateGateReparentActionItemAgentRejected verifies the L8
// reparent gate: changing the parent of a STEWARD-owned item by an agent is
// REJECTED.
func TestAssertOwnerStateGateReparentActionItemAgentRejected(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	stewardGated := newStewardGatedActionItem(t, fixture, "")

	// Build a sibling target parent the agent could try to reparent under.
	sibling, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:      stewardGated.ProjectID,
		ColumnID:       stewardGated.ColumnID,
		Title:          "Sibling target parent",
		StructuralType: string(domain.StructuralTypeDroplet),
		Actor:          stewardGatedActor("agent"),
	})
	if err != nil {
		t.Fatalf("CreateActionItem(sibling) error = %v", err)
	}

	rejectingActor := stewardGatedActor("agent")
	if _, err := fixture.adapter.ReparentActionItem(ctx, ReparentActionItemRequest{
		ActionItemID: stewardGated.ID,
		ParentID:     sibling.ID,
		Actor:        rejectingActor,
	}); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("ReparentActionItem(agent on STEWARD-owned) error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestAssertOwnerStateGateCreateChildUnderStewardParentAgentSucceeds
// verifies the gate does NOT fire on CreateActionItem — drop-orchs spawning
// children under STEWARD persistent parents (the auto-gen pattern) are
// explicitly permitted. The gate keys on the *target* item's Owner, and on
// create no target exists yet; a STEWARD-owned parent is irrelevant to the
// gate's surface area.
func TestAssertOwnerStateGateCreateChildUnderStewardParentAgentSucceeds(t *testing.T) {
	t.Parallel()

	fixture := newCommonLifecycleFixture(t)
	ctx := context.Background()
	stewardParent := newStewardGatedActionItem(t, fixture, "")

	rejectingActor := stewardGatedActor("agent")
	child, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:      stewardParent.ProjectID,
		ParentID:       stewardParent.ID,
		ColumnID:       stewardParent.ColumnID,
		Title:          "Drop-N finding under STEWARD parent",
		Kind:           string(domain.KindBuild),
		StructuralType: string(domain.StructuralTypeDroplet),
		Actor:          rejectingActor,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(agent under STEWARD parent) error = %v", err)
	}
	if child.ParentID != stewardParent.ID {
		t.Fatalf("CreateActionItem() parent_id = %q, want %q", child.ParentID, stewardParent.ID)
	}
}

// stewardGatedActor builds an ActorLeaseTuple with the given
// AuthRequestPrincipalType — the test entry point for the gate. The user
// actor type is reused because the underlying test fixture has no autent
// session machinery; the gate is exercised via the AuthRequestPrincipalType
// field which withMutationGuardContext threads onto AuthenticatedCaller
// regardless of actor type.
func stewardGatedActor(authRequestPrincipalType string) ActorLeaseTuple {
	return ActorLeaseTuple{
		ActorID:                  "tester-1",
		ActorName:                "Tester One",
		ActorType:                string(domain.ActorTypeUser),
		AuthRequestPrincipalType: authRequestPrincipalType,
	}
}

// newStewardGatedActionItem builds one Owner="STEWARD" action item via the
// service-API path and then promotes Owner via the repository layer. Tests
// assume domain field plumbing through CreateActionItemInput is incomplete
// (it lands in droplet 3.21); the repo direct-write keeps the fixture clean
// without requiring 3.21's transport-level wiring.
//
// Returns the persisted action item with Owner=STEWARD and DropNumber=3.
// `ownerOverride` lets a single caller request a non-STEWARD owner for the
// gate-bypass test; empty string defaults to "STEWARD".
func newStewardGatedActionItem(t *testing.T, fixture commonLifecycleFixture, ownerOverride string) domain.ActionItem {
	t.Helper()

	ctx := context.Background()
	bootstrapActor := ActorLeaseTuple{
		ActorID:                  "bootstrap-user",
		ActorName:                "Bootstrap User",
		ActorType:                string(domain.ActorTypeUser),
		AuthRequestPrincipalType: "user",
	}

	project, err := fixture.adapter.CreateProject(ctx, CreateProjectRequest{
		Name:        "STEWARD-gated lifecycle",
		Description: "fixture project for the owner-state-lock tests",
		Actor:       bootstrapActor,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if _, err := fixture.svc.CreateColumn(ctx, project.ID, "To Do", 0, 0); err != nil {
		t.Fatalf("CreateColumn(todo) error = %v", err)
	}
	inProgress, err := fixture.svc.CreateColumn(ctx, project.ID, "In Progress", 1, 0)
	if err != nil {
		t.Fatalf("CreateColumn(in_progress) error = %v", err)
	}
	if _, err := fixture.svc.CreateColumn(ctx, project.ID, "Complete", 2, 0); err != nil {
		t.Fatalf("CreateColumn(complete) error = %v", err)
	}
	_ = inProgress

	columns, err := fixture.svc.ListColumns(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	var todoColumnID string
	for _, column := range columns {
		if actionItemLifecycleStateForColumnName(column.Name) == domain.StateTodo {
			todoColumnID = column.ID
			break
		}
	}
	if todoColumnID == "" {
		t.Fatalf("no todo column seeded")
	}

	created, err := fixture.adapter.CreateActionItem(ctx, CreateActionItemRequest{
		ProjectID:      project.ID,
		ColumnID:       todoColumnID,
		Title:          "STEWARD parent — gated lifecycle",
		Description:    "owner=STEWARD seeded for owner-state-lock tests",
		StructuralType: string(domain.StructuralTypeDroplet),
		Actor:          bootstrapActor,
	})
	if err != nil {
		t.Fatalf("CreateActionItem() error = %v", err)
	}

	owner := ownerOverride
	if owner == "" {
		owner = "STEWARD"
	}
	created.Owner = owner
	created.DropNumber = 3
	created.Persistent = true
	if err := fixture.repo.UpdateActionItem(ctx, created); err != nil {
		t.Fatalf("repo.UpdateActionItem(set Owner=%q) error = %v", owner, err)
	}

	got, err := fixture.adapter.GetActionItem(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetActionItem() error = %v", err)
	}
	return got
}
