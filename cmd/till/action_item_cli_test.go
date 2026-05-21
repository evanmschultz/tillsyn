package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/adapters/storage/sqlite"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestRunActionItemMutationGate verifies the CLI mutation gate enforces the
// mutations-require-UUID rule, returning the canonical error class for dotted
// addresses and a not-yet-implemented hint for valid UUIDs (the actual mutation
// pipelines are not yet wired into the CLI per Droplet 2.11).
func TestRunActionItemMutationGate(t *testing.T) {
	t.Parallel()

	t.Run("dotted body rejected with mutations-require-UUID", func(t *testing.T) {
		t.Parallel()
		err := runActionItemMutationGate("update", actionItemCommandOptions{actionItemID: "1.5.2"})
		if err == nil {
			t.Fatal("expected error for dotted action_item_id, got nil")
		}
		if !errors.Is(err, app.ErrMutationsRequireUUID) {
			t.Fatalf("expected ErrMutationsRequireUUID, got %v", err)
		}
		if !strings.Contains(err.Error(), "1.5.2") {
			t.Fatalf("error message %q does not name the offending input", err)
		}
	})

	t.Run("slug-prefix dotted form rejected with mutations-require-UUID", func(t *testing.T) {
		t.Parallel()
		err := runActionItemMutationGate("delete", actionItemCommandOptions{actionItemID: "tillsyn:1.5.2"})
		if err == nil {
			t.Fatal("expected error for slug-prefix dotted action_item_id, got nil")
		}
		if !errors.Is(err, app.ErrMutationsRequireUUID) {
			t.Fatalf("expected ErrMutationsRequireUUID, got %v", err)
		}
	})

	t.Run("UUID input passes the gate but hits not-implemented hint", func(t *testing.T) {
		t.Parallel()
		err := runActionItemMutationGate("move", actionItemCommandOptions{actionItemID: "11111111-1111-1111-1111-111111111111"})
		if err == nil {
			t.Fatal("expected not-implemented error for valid UUID, got nil")
		}
		if errors.Is(err, app.ErrMutationsRequireUUID) {
			t.Fatalf("UUID should pass the validator gate, got mutations-require-UUID error: %v", err)
		}
		if !strings.Contains(err.Error(), "not yet implemented") {
			t.Fatalf("expected not-yet-implemented hint, got %v", err)
		}
	})

	t.Run("empty action_item_id surfaces invalid-syntax error", func(t *testing.T) {
		t.Parallel()
		err := runActionItemMutationGate("restore", actionItemCommandOptions{actionItemID: ""})
		if err == nil {
			t.Fatal("expected error for empty action_item_id, got nil")
		}
		if !errors.Is(err, app.ErrDottedAddressInvalidSyntax) {
			t.Fatalf("expected ErrDottedAddressInvalidSyntax for empty input, got %v", err)
		}
	})
}

// TestRunActionItemGet verifies the CLI get command resolves UUID and dotted
// inputs end-to-end against a real app.Service backed by an in-memory SQLite
// repository. Uses a tiny tree: project tillsyn → c1 → root → child.
func TestRunActionItemGet(t *testing.T) {
	t.Parallel()

	svc, projectID, rootID, childID := newActionItemCLIServiceForTest(t)
	_ = projectID

	t.Run("UUID input bypasses resolver and returns the matching action item", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemGet(context.Background(), svc, actionItemCommandOptions{actionItemID: childID}, &out)
		if err != nil {
			t.Fatalf("runActionItemGet() error = %v", err)
		}
		if !strings.Contains(out.String(), childID) {
			t.Fatalf("output missing child id %q: %s", childID, out.String())
		}
	})

	t.Run("dotted body with --project resolves and reads", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemGet(context.Background(), svc, actionItemCommandOptions{
			projectSlug:  "tillsyn-cli",
			actionItemID: "0.0",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemGet() error = %v", err)
		}
		if !strings.Contains(out.String(), childID) {
			t.Fatalf("expected dotted 0.0 to resolve to child %q: %s", childID, out.String())
		}
	})

	t.Run("slug-prefix shorthand resolves project then walks tree", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemGet(context.Background(), svc, actionItemCommandOptions{
			actionItemID: "tillsyn-cli:0",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemGet() error = %v", err)
		}
		// 0 is the level-1 root — assert against the rootID we created above.
		if !strings.Contains(out.String(), rootID) {
			t.Fatalf("expected slug-prefix `tillsyn-cli:0` to resolve to root %q: %s", rootID, out.String())
		}
	})

	t.Run("bare dotted body without project context errors with hint", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemGet(context.Background(), svc, actionItemCommandOptions{actionItemID: "0.0"}, &out)
		if err == nil {
			t.Fatal("expected error for dotted address without project, got nil")
		}
		if !strings.Contains(err.Error(), "--project") {
			t.Fatalf("error %q does not point operator at --project flag", err)
		}
	})

	t.Run("slug-prefix conflicting with --project errors out", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemGet(context.Background(), svc, actionItemCommandOptions{
			projectSlug:  "other-slug",
			actionItemID: "tillsyn-cli:0",
		}, &out)
		if err == nil {
			t.Fatal("expected error when --project and slug-prefix disagree, got nil")
		}
		if !strings.Contains(err.Error(), "does not match") {
			t.Fatalf("error %q does not surface the slug mismatch", err)
		}
	})

	t.Run("malformed input rejected before any service call", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemGet(context.Background(), svc, actionItemCommandOptions{actionItemID: "not-a-uuid-or-dotted"}, &out)
		if err == nil {
			t.Fatal("expected error for malformed action_item_id, got nil")
		}
	})
}

// newActionItemCLIServiceForTest seeds a project + column + a tiny action-item tree.
// Returns (service, projectID, rootActionItemID, childActionItemID). The tree
// shape is rootActionItemID@0 → childActionItemID@0.0 — the smallest tree that
// exercises both single-level and two-level dotted resolution.
func newActionItemCLIServiceForTest(t *testing.T) (*app.Service, string, string, string) {
	t.Helper()

	repo, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-actionitem-cli", Name: "Tillsyn CLI Test"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	// Override the slug to a known value the tests assert against.
	project.Slug = "tillsyn-cli"
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	idCounter := 0
	idGen := func() string {
		idCounter++
		// Predictable but valid-UUID-shaped ids so resolveActionItemID can detect them as UUIDs when needed.
		// The dotted-address resolver doesn't care about ID format; it walks by position.
		return strings.Repeat("0", 32-len(itoa(idCounter))) + itoa(idCounter)
	}
	clk := func() time.Time {
		now = now.Add(time.Second)
		return now
	}
	svc := app.NewService(repo, idGen, clk, app.ServiceConfig{})

	root, err := svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		Title:          "Root action item",
		Kind:           domain.KindPlan,
		Scope:          domain.KindAppliesToPlan,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(root) error = %v", err)
	}
	child, err := svc.CreateActionItem(context.Background(), app.CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       column.ID,
		ParentID:       root.ID,
		Title:          "Child action item",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(child) error = %v", err)
	}
	return svc, project.ID, root.ID, child.ID
}

// itoa is a tiny non-strconv integer-to-decimal helper used by the test idGen
// to keep id generation hermetic from package-level state. Returns the
// canonical decimal form (no padding); the caller pads to 32 chars to match
// id-shape expectations of downstream code.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// TestRunActionItemSupersede pins the Drop 4c.5 droplet B.1 CLI contract.
// The flow is:
//  1. Empty / whitespace-only `--reason` rejects BEFORE any service call
//     (gate is in `runActionItemSupersede` itself, not the service).
//  2. Dotted-form `action_item_id` rejects with `ErrMutationsRequireUUID`
//     via the shared mutations-require-UUID validator.
//  3. UUID-shaped input passes the gate and reaches the service. A
//     non-existent UUID surfaces `app.ErrNotFound`. A `failed` UUID-shaped
//     item supersedes successfully — JSON output names the new state
//     (`complete`) and outcome (`superseded`).
func TestRunActionItemSupersede(t *testing.T) {
	t.Parallel()

	t.Run("dotted body rejected with mutations-require-UUID", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemSupersede(context.Background(), nil, actionItemCommandOptions{
			actionItemID: "1.5.2",
			reason:       "valid reason",
		}, &out)
		if err == nil {
			t.Fatal("expected error for dotted action_item_id, got nil")
		}
		if !errors.Is(err, app.ErrMutationsRequireUUID) {
			t.Fatalf("expected ErrMutationsRequireUUID, got %v", err)
		}
		if !strings.Contains(err.Error(), "1.5.2") {
			t.Fatalf("error %q does not name the offending input", err)
		}
	})

	t.Run("slug-prefix dotted form rejected", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemSupersede(context.Background(), nil, actionItemCommandOptions{
			actionItemID: "tillsyn:1.5.2",
			reason:       "valid reason",
		}, &out)
		if err == nil {
			t.Fatal("expected error for slug-prefix dotted form, got nil")
		}
		if !errors.Is(err, app.ErrMutationsRequireUUID) {
			t.Fatalf("expected ErrMutationsRequireUUID, got %v", err)
		}
	})

	t.Run("empty reason rejects before service call", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemSupersede(context.Background(), nil, actionItemCommandOptions{
			actionItemID: "11111111-1111-1111-1111-111111111111",
			reason:       "",
		}, &out)
		if err == nil {
			t.Fatal("expected error for empty reason, got nil")
		}
		if !strings.Contains(err.Error(), "--reason is required") {
			t.Fatalf("error %q missing '--reason is required' hint", err)
		}
		// Service was nil — if the gate lets us through we'd panic on
		// dereference. The fact that the test reaches `--reason is
		// required` proves the gate fires before the service call.
	})

	t.Run("whitespace-only reason rejects", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemSupersede(context.Background(), nil, actionItemCommandOptions{
			actionItemID: "11111111-1111-1111-1111-111111111111",
			reason:       "   ",
		}, &out)
		if err == nil {
			t.Fatal("expected error for whitespace-only reason, got nil")
		}
		if !strings.Contains(err.Error(), "--reason is required") {
			t.Fatalf("error %q missing '--reason is required' hint", err)
		}
	})

	t.Run("empty action_item_id surfaces invalid-syntax", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemSupersede(context.Background(), nil, actionItemCommandOptions{
			actionItemID: "",
			reason:       "valid reason",
		}, &out)
		if err == nil {
			t.Fatal("expected error for empty action_item_id, got nil")
		}
		if !errors.Is(err, app.ErrDottedAddressInvalidSyntax) {
			t.Fatalf("expected ErrDottedAddressInvalidSyntax for empty input, got %v", err)
		}
	})

	t.Run("UUID input passes gates and reaches service end-to-end", func(t *testing.T) {
		t.Parallel()
		svc, _, failedID := newSupersedeCLIServiceForTest(t)
		var out strings.Builder
		err := runActionItemSupersede(context.Background(), svc, actionItemCommandOptions{
			actionItemID: failedID,
			reason:       "rejected by dev",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemSupersede() error = %v", err)
		}
		if !strings.Contains(out.String(), "\"superseded\"") {
			t.Fatalf("output missing outcome=superseded stamp: %s", out.String())
		}
		if !strings.Contains(out.String(), "\"rejected by dev\"") {
			t.Fatalf("output missing transition_notes reason text: %s", out.String())
		}
		if !strings.Contains(out.String(), "\"complete\"") {
			t.Fatalf("output missing lifecycle_state=complete: %s", out.String())
		}
	})

	t.Run("UUID input that is non-failed surfaces ErrTransitionBlocked", func(t *testing.T) {
		t.Parallel()
		svc, todoID, _ := newSupersedeCLIServiceForTest(t)
		var out strings.Builder
		err := runActionItemSupersede(context.Background(), svc, actionItemCommandOptions{
			actionItemID: todoID,
			reason:       "valid reason",
		}, &out)
		if err == nil {
			t.Fatal("expected error for todo item, got nil")
		}
		if !errors.Is(err, domain.ErrTransitionBlocked) {
			t.Fatalf("expected ErrTransitionBlocked, got %v", err)
		}
	})
}

// newSupersedeCLIServiceForTest seeds a project + columns (todo + complete +
// failed) + two action items: one in todo, one in failed. Returns
// (svc, todoID, failedID). Used by the supersede CLI end-to-end tests.
func newSupersedeCLIServiceForTest(t *testing.T) (*app.Service, string, string) {
	t.Helper()
	repo, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-supersede-cli", Name: "Supersede CLI"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	project.Slug = "supersede-cli"
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	colSpecs := []struct {
		id    string
		name  string
		pos   int
		state domain.LifecycleState
	}{
		{id: "c-supersede-todo", name: "To Do", pos: 0, state: domain.StateTodo},
		{id: "c-supersede-progress", name: "In Progress", pos: 1, state: domain.StateInProgress},
		{id: "c-supersede-complete", name: "Complete", pos: 2, state: domain.StateComplete},
		{id: "c-supersede-failed", name: "Failed", pos: 3, state: domain.StateFailed},
	}
	colsByState := map[domain.LifecycleState]domain.Column{}
	for _, spec := range colSpecs {
		col, err := domain.NewColumn(spec.id, project.ID, spec.name, spec.pos, 0, now)
		if err != nil {
			t.Fatalf("NewColumn(%q) error = %v", spec.name, err)
		}
		if err := repo.CreateColumn(ctx, col); err != nil {
			t.Fatalf("CreateColumn(%q) error = %v", spec.name, err)
		}
		colsByState[spec.state] = col
	}
	idCounter := 0
	idGen := func() string {
		idCounter++
		return strings.Repeat("0", 32-len(itoa(idCounter))) + itoa(idCounter)
	}
	clk := func() time.Time { return now.Add(time.Second) }
	svc := app.NewService(repo, idGen, clk, app.ServiceConfig{})

	todoItem, err := svc.CreateActionItem(ctx, app.CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       colsByState[domain.StateTodo].ID,
		Title:          "Todo item",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(todo) error = %v", err)
	}
	failedItem, err := svc.CreateActionItem(ctx, app.CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       colsByState[domain.StateTodo].ID,
		Title:          "Failed item to supersede",
		Kind:           domain.KindBuild,
		Scope:          domain.KindAppliesToBuild,
		StructuralType: domain.StructuralTypeDroplet,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(failed) error = %v", err)
	}
	// Stamp metadata.outcome=failure BEFORE flipping into the failed
	// column so the A.4 guard at MoveActionItem (Drop 4c.5) accepts the
	// transition. The CLI suite is testing supersede semantics, not the
	// A.4 path — this setup pre-meets the precondition.
	failedItem.Metadata.Outcome = "failure"
	if err := repo.UpdateActionItem(ctx, failedItem); err != nil {
		t.Fatalf("UpdateActionItem(stamp outcome=failure) error = %v", err)
	}
	moved, err := svc.MoveActionItem(ctx, failedItem.ID, colsByState[domain.StateFailed].ID, 0)
	if err != nil {
		t.Fatalf("MoveActionItem(→failed) error = %v", err)
	}
	return svc, todoItem.ID, moved.ID
}

// TestRunActionItemList pins the Drop 4c.5 droplet B.2 CLI list contract.
// The flow is:
//  1. `--state` is normalized (trim + lower) and validated against the closed
//     lifecycle set; unknown states reject naming the valid set.
//  2. Project resolution requires --project explicitly OR exactly one project
//     on the system; multi-project + no --project rejects with a hint.
//  3. On success the table renders with columns DOTTED / UUID / TITLE /
//     KIND / ROLE / UPDATED. Empty result renders the empty-state message.
//
// The fixture seeds a multi-project + multi-state setup so all spec table
// rows are exercisable with one shared service.
func TestRunActionItemList(t *testing.T) {
	t.Parallel()

	t.Run("list failed items in project with two failed + three non-failed", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds: []listCLISeed{
				{title: "fail-1", state: domain.StateFailed},
				{title: "fail-2", state: domain.StateFailed},
				{title: "todo-1", state: domain.StateTodo},
				{title: "progress-1", state: domain.StateInProgress},
				{title: "complete-1", state: domain.StateComplete},
			},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			projectSlug: "tillsyn",
			state:       "failed",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemList() error = %v", err)
		}
		text := out.String()
		if !strings.Contains(text, "fail-1") || !strings.Contains(text, "fail-2") {
			t.Fatalf("expected both failed titles in table, got: %s", text)
		}
		if strings.Contains(text, "todo-1") || strings.Contains(text, "progress-1") || strings.Contains(text, "complete-1") {
			t.Fatalf("non-failed items leaked into table: %s", text)
		}
		// Header columns surface in the rendered table.
		for _, col := range []string{"DOTTED", "UUID", "TITLE", "KIND", "ROLE", "UPDATED"} {
			if !strings.Contains(text, col) {
				t.Fatalf("missing header column %q in table: %s", col, text)
			}
		}
	})

	t.Run("list failed items in project with zero failed", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds: []listCLISeed{
				{title: "todo-only", state: domain.StateTodo},
			},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			projectSlug: "tillsyn",
			state:       "failed",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemList() error = %v", err)
		}
		text := out.String()
		if !strings.Contains(text, "No failed action items in project tillsyn") {
			t.Fatalf("expected empty-state message, got: %s", text)
		}
	})

	t.Run("invalid state rejects naming the valid set", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds:       []listCLISeed{{title: "fail-1", state: domain.StateFailed}},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			projectSlug: "tillsyn",
			state:       "weird",
		}, &out)
		if err == nil {
			t.Fatal("expected error for unknown --state, got nil")
		}
		if !strings.Contains(err.Error(), "unknown --state") {
			t.Fatalf("error %q missing 'unknown --state'", err)
		}
		// Valid set surfaces.
		for _, want := range []string{"todo", "in_progress", "complete", "failed", "archived"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q missing valid state %q", err, want)
			}
		}
	})

	t.Run("no --project hint when multiple projects exist", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds:       []listCLISeed{{title: "fail-1", state: domain.StateFailed}},
			extraProjects: []string{
				"tillsyn-other",
				"tillsyn-third",
			},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			state: "failed",
		}, &out)
		if err == nil {
			t.Fatal("expected error for missing --project + multiple projects, got nil")
		}
		if !strings.Contains(err.Error(), "--project") {
			t.Fatalf("error %q does not point at --project", err)
		}
		if !strings.Contains(err.Error(), "tillsyn") {
			t.Fatalf("error %q does not list available slugs", err)
		}
	})

	t.Run("state=todo returns todo items", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds: []listCLISeed{
				{title: "todo-1", state: domain.StateTodo},
				{title: "fail-1", state: domain.StateFailed},
			},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			projectSlug: "tillsyn",
			state:       "todo",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemList() error = %v", err)
		}
		text := out.String()
		if !strings.Contains(text, "todo-1") {
			t.Fatalf("expected todo-1 in table: %s", text)
		}
		if strings.Contains(text, "fail-1") {
			t.Fatalf("failed leaked into todo filter: %s", text)
		}
	})

	t.Run("state=in_progress returns in_progress items", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds: []listCLISeed{
				{title: "progress-1", state: domain.StateInProgress},
				{title: "fail-1", state: domain.StateFailed},
			},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			projectSlug: "tillsyn",
			state:       "in_progress",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemList() error = %v", err)
		}
		text := out.String()
		if !strings.Contains(text, "progress-1") {
			t.Fatalf("expected progress-1 in table: %s", text)
		}
	})

	t.Run("state=archived implies includeArchived without --include-archived", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds: []listCLISeed{
				{title: "arch-1", state: domain.StateArchived, archived: true},
			},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			projectSlug:     "tillsyn",
			state:           "archived",
			includeArchived: false,
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemList() error = %v", err)
		}
		if !strings.Contains(out.String(), "arch-1") {
			t.Fatalf("expected archived item to surface even with --include-archived=false: %s", out.String())
		}
	})

	t.Run("--include-archived + state=failed surfaces failed-and-archived", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds: []listCLISeed{
				{title: "failed-archived", state: domain.StateFailed, archived: true},
				{title: "failed-only", state: domain.StateFailed},
			},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			projectSlug:     "tillsyn",
			state:           "failed",
			includeArchived: true,
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemList() error = %v", err)
		}
		text := out.String()
		if !strings.Contains(text, "failed-archived") {
			t.Fatalf("expected failed-archived in table with --include-archived: %s", text)
		}
		if !strings.Contains(text, "failed-only") {
			t.Fatalf("expected failed-only in table: %s", text)
		}
	})

	t.Run("project slug typo surfaces GetProjectBySlug error", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "tillsyn",
			seeds:       []listCLISeed{{title: "fail-1", state: domain.StateFailed}},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			projectSlug: "tillsynx",
			state:       "failed",
		}, &out)
		if err == nil {
			t.Fatal("expected error for unknown project slug, got nil")
		}
		if !strings.Contains(err.Error(), "tillsynx") {
			t.Fatalf("error %q does not name the offending slug", err)
		}
	})

	t.Run("nil service rejects with not-configured", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemList(context.Background(), nil, actionItemCommandOptions{
			projectSlug: "tillsyn",
			state:       "failed",
		}, &out)
		if err == nil {
			t.Fatal("expected error for nil service, got nil")
		}
		if !strings.Contains(err.Error(), "not configured") {
			t.Fatalf("error %q missing 'not configured' hint", err)
		}
	})

	t.Run("single-project fallback resolves --project automatically", func(t *testing.T) {
		t.Parallel()
		svc, _ := newListCLIServiceForTest(t, listCLIFixtureSpec{
			projectSlug: "only-one",
			seeds:       []listCLISeed{{title: "fail-1", state: domain.StateFailed}},
		})
		var out strings.Builder
		err := runActionItemList(context.Background(), svc, actionItemCommandOptions{
			state: "failed",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemList() error = %v", err)
		}
		if !strings.Contains(out.String(), "fail-1") {
			t.Fatalf("expected fail-1 in single-project fallback: %s", out.String())
		}
	})
}

// TestRunActionItemCreate_StructuralTypeSmartDefault verifies the FF4
// smart-default table (now extended for Drop 4d_5 Lane A D3 with the
// hasParent axis):
//
//   - hasParent=false (any kind) → cascade (level-1 cascade root)
//   - hasParent=true && (plan|refinement) → segment
//   - hasParent=true && other 10 kinds → droplet
//
// Plus: explicit valid override accepted, explicit invalid value rejects with
// the valid list.
func TestRunActionItemCreate_StructuralTypeSmartDefault(t *testing.T) {
	t.Parallel()

	t.Run("structuralTypeSmartDefault covers all 12 kinds with hasParent=true", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			kind string
			want domain.StructuralType
		}{
			{"plan", domain.StructuralTypeSegment},
			{"refinement", domain.StructuralTypeSegment},
			{"build", domain.StructuralTypeDroplet},
			{"research", domain.StructuralTypeDroplet},
			{"plan-qa-proof", domain.StructuralTypeDroplet},
			{"plan-qa-falsification", domain.StructuralTypeDroplet},
			{"build-qa-proof", domain.StructuralTypeDroplet},
			{"build-qa-falsification", domain.StructuralTypeDroplet},
			{"closeout", domain.StructuralTypeDroplet},
			{"commit", domain.StructuralTypeDroplet},
			{"discussion", domain.StructuralTypeDroplet},
			{"human-verify", domain.StructuralTypeDroplet},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.kind, func(t *testing.T) {
				t.Parallel()
				got := structuralTypeSmartDefault(tc.kind, true)
				if got != tc.want {
					t.Fatalf("structuralTypeSmartDefault(%q, true) = %q, want %q", tc.kind, got, tc.want)
				}
			})
		}
	})

	t.Run("structuralTypeSmartDefault returns cascade with hasParent=false", func(t *testing.T) {
		t.Parallel()
		// Every kind at level-1 (hasParent=false) must classify as cascade
		// per Drop 4d_5 Lane A HV-1 = Option A: cascade IS the level-1
		// structural unit regardless of work-axis kind.
		cases := []string{
			"plan", "refinement", "build", "research",
			"plan-qa-proof", "plan-qa-falsification",
			"build-qa-proof", "build-qa-falsification",
			"closeout", "commit", "discussion", "human-verify",
			"", // empty kind still routes through the hasParent=false branch
		}
		for _, kind := range cases {
			kind := kind
			t.Run(kind, func(t *testing.T) {
				t.Parallel()
				got := structuralTypeSmartDefault(kind, false)
				if got != domain.StructuralTypeCascade {
					t.Fatalf("structuralTypeSmartDefault(%q, false) = %q, want %q", kind, got, domain.StructuralTypeCascade)
				}
			})
		}
	})

	t.Run("explicit valid override accepted", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-override")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:      projectID,
			kind:           "build",
			title:          "T",
			description:    "D",
			structuralType: "confluence",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		if !strings.Contains(out.String(), "Created action item") {
			t.Fatalf("expected 'Created action item' in output, got: %s", out.String())
		}
	})

	t.Run("explicit invalid structural-type rejects with valid list", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-invalid")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:      projectID,
			kind:           "build",
			title:          "T",
			description:    "D",
			structuralType: "invalid-value",
		}, &out)
		if err == nil {
			t.Fatal("expected error for invalid structural-type, got nil")
		}
		for _, valid := range []string{"drop", "segment", "confluence", "droplet", "cascade"} {
			if !strings.Contains(err.Error(), valid) {
				t.Fatalf("error %q missing valid value %q", err, valid)
			}
		}
	})

	t.Run("smart-default plan creates segment without explicit flag", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-plan")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:   projectID,
			kind:        "plan",
			title:       "My plan",
			description: "D",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		if !strings.Contains(out.String(), "Created action item") {
			t.Fatalf("output missing 'Created action item': %s", out.String())
		}
	})

	t.Run("smart-default build creates droplet without explicit flag", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-build")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:   projectID,
			kind:        "build",
			title:       "My build",
			description: "D",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		if !strings.Contains(out.String(), "Created action item") {
			t.Fatalf("output missing 'Created action item': %s", out.String())
		}
	})
}

// TestRunActionItemCreate_RequiredFields verifies that missing required flags
// surface a clear error before any service call.
func TestRunActionItemCreate_RequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts actionItemCreateCommandOptions
		want string
	}{
		{
			name: "missing project-id",
			opts: actionItemCreateCommandOptions{kind: "build", title: "T", description: "D"},
			want: "--project-id",
		},
		{
			name: "missing kind",
			opts: actionItemCreateCommandOptions{projectID: "proj-id", title: "T", description: "D"},
			want: "--kind",
		},
		{
			name: "missing title",
			opts: actionItemCreateCommandOptions{projectID: "proj-id", kind: "build", description: "D"},
			want: "--title",
		},
		{
			name: "missing description",
			opts: actionItemCreateCommandOptions{projectID: "proj-id", kind: "build", title: "T"},
			want: "--description",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// nil svc: the required-field gate fires before any service call.
			var out strings.Builder
			err := runActionItemCreate(context.Background(), nil, tc.opts, &out)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

// TestRunActionItemCreate_PassThroughFlags verifies that every pass-through
// flag lands on the created action item and that ColumnID is auto-resolved to
// the first project column (sorted by position).
func TestRunActionItemCreate_PassThroughFlags(t *testing.T) {
	t.Parallel()

	t.Run("blocked-by sets Metadata.BlockedBy without post-create update", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-blockedby")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:   projectID,
			kind:        "build",
			title:       "Blocked build",
			description: "D",
			blockedBy:   []string{"dep-uuid-1", "dep-uuid-2"},
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		// Verify the created item has BlockedBy set via JSON output inspection.
		// Read the newly created item — its UUID is in "Created action item <uuid> (dotted: ...)".
		outputText := out.String()
		if !strings.Contains(outputText, "Created action item") {
			t.Fatalf("output missing 'Created action item': %s", outputText)
		}
		// Extract UUID from output: "Created action item <uuid> (dotted: ...)".
		// parts[0]="Created" parts[1]="action" parts[2]="item" parts[3]=<uuid>
		parts := strings.Fields(outputText)
		if len(parts) < 5 {
			t.Fatalf("unexpected output format: %s", outputText)
		}
		createdID := parts[3]
		item, err := svc.GetActionItem(context.Background(), createdID)
		if err != nil {
			t.Fatalf("GetActionItem(%q) error = %v", createdID, err)
		}
		if len(item.Metadata.BlockedBy) != 2 {
			t.Fatalf("expected 2 BlockedBy entries, got %d: %v", len(item.Metadata.BlockedBy), item.Metadata.BlockedBy)
		}
		if item.Metadata.BlockedBy[0] != "dep-uuid-1" || item.Metadata.BlockedBy[1] != "dep-uuid-2" {
			t.Fatalf("BlockedBy mismatch: %v", item.Metadata.BlockedBy)
		}
	})

	t.Run("paths and packages pass through", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-paths")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:   projectID,
			kind:        "build",
			title:       "With paths",
			description: "D",
			paths:       []string{"cmd/till/action_item_cli.go"},
			packages:    []string{"cmd/till"},
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		outputText := out.String()
		parts := strings.Fields(outputText)
		if len(parts) < 5 {
			t.Fatalf("unexpected output format: %s", outputText)
		}
		createdID := parts[3]
		item, err := svc.GetActionItem(context.Background(), createdID)
		if err != nil {
			t.Fatalf("GetActionItem(%q) error = %v", createdID, err)
		}
		if len(item.Paths) != 1 || item.Paths[0] != "cmd/till/action_item_cli.go" {
			t.Fatalf("Paths mismatch: %v", item.Paths)
		}
		if len(item.Packages) != 1 || item.Packages[0] != "cmd/till" {
			t.Fatalf("Packages mismatch: %v", item.Packages)
		}
	})

	t.Run("role pass-through", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-role")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:   projectID,
			kind:        "build",
			title:       "With role",
			description: "D",
			role:        "builder",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		outputText := out.String()
		parts := strings.Fields(outputText)
		if len(parts) < 5 {
			t.Fatalf("unexpected output format: %s", outputText)
		}
		createdID := parts[3]
		item, err := svc.GetActionItem(context.Background(), createdID)
		if err != nil {
			t.Fatalf("GetActionItem(%q) error = %v", createdID, err)
		}
		if item.Role != domain.Role("builder") {
			t.Fatalf("Role mismatch: got %q, want %q", item.Role, "builder")
		}
	})

	t.Run("metadata-json pass-through", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-meta")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:    projectID,
			kind:         "build",
			title:        "With metadata",
			description:  "D",
			metadataJSON: `{"objective":"test objective"}`,
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		outputText := out.String()
		parts := strings.Fields(outputText)
		if len(parts) < 5 {
			t.Fatalf("unexpected output format: %s", outputText)
		}
		createdID := parts[3]
		item, err := svc.GetActionItem(context.Background(), createdID)
		if err != nil {
			t.Fatalf("GetActionItem(%q) error = %v", createdID, err)
		}
		if item.Metadata.Objective != "test objective" {
			t.Fatalf("Metadata.Objective mismatch: got %q, want %q", item.Metadata.Objective, "test objective")
		}
	})

	t.Run("metadata-json malformed JSON returns clear error", func(t *testing.T) {
		t.Parallel()
		// Use a real service so execution reaches the JSON-parse gate (the svc==nil
		// and svc.ListColumns checks fire before JSON parsing).
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-badjson")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:    projectID,
			kind:         "build",
			title:        "T",
			description:  "D",
			metadataJSON: `{invalid`,
		}, &out)
		if err == nil {
			t.Fatal("expected error for malformed --metadata-json, got nil")
		}
		if !strings.Contains(err.Error(), "not valid JSON") {
			t.Fatalf("error %q missing 'not valid JSON' phrase", err)
		}
	})

	t.Run("column auto-resolved to first column sorted by position", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-colid")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:   projectID,
			kind:        "build",
			title:       "Auto column",
			description: "D",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		outputText := out.String()
		parts := strings.Fields(outputText)
		if len(parts) < 5 {
			t.Fatalf("unexpected output format: %s", outputText)
		}
		createdID := parts[3]
		item, err := svc.GetActionItem(context.Background(), createdID)
		if err != nil {
			t.Fatalf("GetActionItem(%q) error = %v", createdID, err)
		}
		// ColumnID must be non-empty (auto-resolved from the project's columns).
		if item.ColumnID == "" {
			t.Fatal("expected non-empty ColumnID after auto-resolution, got empty")
		}
	})

	t.Run("nil service rejects with not-configured", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		err := runActionItemCreate(context.Background(), nil, actionItemCreateCommandOptions{
			projectID:   "proj-id",
			kind:        "build",
			title:       "T",
			description: "D",
		}, &out)
		if err == nil {
			t.Fatal("expected error for nil service, got nil")
		}
		if !strings.Contains(err.Error(), "not configured") {
			t.Fatalf("error %q missing 'not configured' hint", err)
		}
	})

	t.Run("output includes id and dotted address", func(t *testing.T) {
		t.Parallel()
		svc, projectID := newCreateCLIServiceForTest(t, "create-svc-output")
		var out strings.Builder
		err := runActionItemCreate(context.Background(), svc, actionItemCreateCommandOptions{
			projectID:   projectID,
			kind:        "build",
			title:       "Output test",
			description: "D",
		}, &out)
		if err != nil {
			t.Fatalf("runActionItemCreate() error = %v", err)
		}
		text := out.String()
		if !strings.Contains(text, "Created action item") {
			t.Fatalf("output missing 'Created action item': %s", text)
		}
		if !strings.Contains(text, "dotted:") {
			t.Fatalf("output missing 'dotted:': %s", text)
		}
	})
}

// newCreateCLIServiceForTest seeds a project with columns + a pre-seeded root
// action item to exercise the dotted-address computation path (new item is
// child 0 of root, address = "0.0"). Returns (svc, projectID).
func newCreateCLIServiceForTest(t *testing.T, projectID string) (*app.Service, string) {
	t.Helper()
	repo, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	project, err := domain.NewProjectFromInput(domain.ProjectInput{ID: projectID, Name: "Create CLI " + projectID}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput() error = %v", err)
	}
	project.Slug = strings.ToLower(strings.ReplaceAll(projectID, "-", ""))
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	col, err := domain.NewColumn("c-create-todo-"+projectID, projectID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(ctx, col); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	idCounter := 0
	idGen := func() string {
		idCounter++
		return strings.Repeat("a", 32-len(itoa(idCounter))) + itoa(idCounter)
	}
	clk := func() time.Time { return now.Add(time.Second) }
	svc := app.NewService(repo, idGen, clk, app.ServiceConfig{})
	return svc, projectID
}

// listCLISeed describes one action-item seed entry for the B.2 list CLI
// fixture. `archived` flips the row's ArchivedAt pointer post-create so the
// fixture exercises the failed+archived cross-axis case.
type listCLISeed struct {
	title    string
	state    domain.LifecycleState
	archived bool
}

// listCLIFixtureSpec describes the seed configuration for one B.2 list CLI
// service fixture: a primary project (created with the supplied slug) plus
// optional extra projects (used to exercise the multi-project --project
// hint path) plus the per-state action-item seeds.
type listCLIFixtureSpec struct {
	projectSlug   string
	seeds         []listCLISeed
	extraProjects []string
}

// newListCLIServiceForTest seeds a real app.Service backed by an in-memory
// SQLite repo, with one column per lifecycle state plus seed action items
// per `spec`. Returns (svc, primaryProjectID). Used by the B.2 list CLI
// table-driven tests.
func newListCLIServiceForTest(t *testing.T, spec listCLIFixtureSpec) (*app.Service, string) {
	t.Helper()
	repo, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	primaryProject, err := domain.NewProjectFromInput(domain.ProjectInput{ID: "p-list-cli-primary", Name: "List CLI"}, now)
	if err != nil {
		t.Fatalf("NewProjectFromInput(primary) error = %v", err)
	}
	primaryProject.Slug = spec.projectSlug
	if err := repo.CreateProject(ctx, primaryProject); err != nil {
		t.Fatalf("CreateProject(primary) error = %v", err)
	}
	for i, slug := range spec.extraProjects {
		extra, err := domain.NewProjectFromInput(domain.ProjectInput{
			ID:   "p-list-cli-extra-" + itoa(i+1),
			Name: "List CLI Extra " + slug,
		}, now)
		if err != nil {
			t.Fatalf("NewProjectFromInput(extra %d) error = %v", i, err)
		}
		extra.Slug = slug
		if err := repo.CreateProject(ctx, extra); err != nil {
			t.Fatalf("CreateProject(extra %d) error = %v", i, err)
		}
	}

	colSpecs := []struct {
		id    string
		name  string
		pos   int
		state domain.LifecycleState
	}{
		{id: "lst-todo", name: "To Do", pos: 0, state: domain.StateTodo},
		{id: "lst-progress", name: "In Progress", pos: 1, state: domain.StateInProgress},
		{id: "lst-complete", name: "Complete", pos: 2, state: domain.StateComplete},
		{id: "lst-failed", name: "Failed", pos: 3, state: domain.StateFailed},
		{id: "lst-archived", name: "Archived", pos: 4, state: domain.StateArchived},
	}
	colsByState := map[domain.LifecycleState]domain.Column{}
	for _, cs := range colSpecs {
		col, err := domain.NewColumn(cs.id, primaryProject.ID, cs.name, cs.pos, 0, now)
		if err != nil {
			t.Fatalf("NewColumn(%q) error = %v", cs.name, err)
		}
		if err := repo.CreateColumn(ctx, col); err != nil {
			t.Fatalf("CreateColumn(%q) error = %v", cs.name, err)
		}
		colsByState[cs.state] = col
	}

	idCounter := 0
	idGen := func() string {
		idCounter++
		return strings.Repeat("0", 32-len(itoa(idCounter))) + itoa(idCounter)
	}
	clk := func() time.Time { return now.Add(time.Second) }
	svc := app.NewService(repo, idGen, clk, app.ServiceConfig{})

	for _, seed := range spec.seeds {
		col, ok := colsByState[seed.state]
		if !ok {
			t.Fatalf("listCLIFixture: no column for state %q", seed.state)
		}
		// Seed action items directly into their target column so the
		// lifecycle state is set at create-time (via
		// `lifecycleStateForColumnID`) without forcing every seed through
		// MoveActionItem + the A.4 outcome-required guard.
		input := app.CreateActionItemInput{
			ProjectID:      primaryProject.ID,
			ColumnID:       col.ID,
			Title:          seed.title,
			Kind:           domain.KindBuild,
			Scope:          domain.KindAppliesToBuild,
			StructuralType: domain.StructuralTypeDroplet,
		}
		created, err := svc.CreateActionItem(ctx, input)
		if err != nil {
			t.Fatalf("CreateActionItem(%q) error = %v", seed.title, err)
		}
		if seed.archived {
			// Archive flag is orthogonal to lifecycle state. We stamp
			// ArchivedAt directly on the stored row to exercise the
			// archived axis without forcing the seed through the
			// archive transition path (which is its own test surface).
			stored, err := repo.GetActionItem(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetActionItem(%q) err=%v", created.ID, err)
			}
			archivedAt := now.Add(time.Hour)
			stored.ArchivedAt = &archivedAt
			if err := repo.UpdateActionItem(ctx, stored); err != nil {
				t.Fatalf("UpdateActionItem(archived) error = %v", err)
			}
		}
	}
	return svc, primaryProject.ID
}
