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
	project, err := domain.NewProject("p-actionitem-cli", "Tillsyn CLI Test", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
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
