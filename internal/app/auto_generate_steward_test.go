package app

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// withSeedTemplateFixture swaps the package-level loadStewardSeedTemplate
// seam with the supplied closure for the duration of the test, restoring
// the previous seam on cleanup. Tests use a hand-built Template so the
// auto-generator's behavior is exercised without depending on the embedded
// default.toml content drift.
func withSeedTemplateFixture(t *testing.T, fixture func() (templates.Template, error)) {
	t.Helper()
	prev := loadStewardSeedTemplate
	loadStewardSeedTemplate = fixture
	t.Cleanup(func() {
		loadStewardSeedTemplate = prev
	})
}

// canonicalSixSeeds returns the canonical 6 STEWARD anchor seeds the
// production default.toml ships with. Tests use this fixture to assert the
// auto-generator materializes one ActionItem per seed under the project
// root with the expected domain shape (Owner, Persistent, Kind, etc.).
func canonicalSixSeeds() []templates.StewardSeed {
	return []templates.StewardSeed{
		{Title: "DISCUSSIONS", Description: "Cross-cutting discussion topics."},
		{Title: "HYLLA_FINDINGS", Description: "Per-drop Hylla feedback."},
		{Title: "LEDGER", Description: "Per-drop ledger entries."},
		{Title: "WIKI_CHANGELOG", Description: "Per-drop wiki changelog entries."},
		{Title: "REFINEMENTS", Description: "Perpetual refinement rollup."},
		{Title: "HYLLA_REFINEMENTS", Description: "Perpetual Hylla refinement rollup."},
	}
}

// newSeederService builds a Service backed by the in-memory fakeRepo and
// the deterministic id/clock helpers, so the auto-generator's idempotency
// and field-shape assertions can be made without external dependencies.
func newSeederService(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	idCounter := 0
	svc := NewService(repo, func() string {
		idCounter++
		return "auto-gen-id-" + string(rune('a'+idCounter))
	}, func() time.Time {
		return time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	}, ServiceConfig{
		DefaultDeleteMode:        DeleteModeArchive,
		AutoCreateProjectColumns: true,
		AutoSeedStewardAnchors:   true,
	})
	return svc, repo
}

// TestAutoGenSeeds6StewardPersistentParents verifies project creation
// materializes one ActionItem per StewardSeed declared on the loaded
// Template, with the documented domain-primitive shape: Owner=STEWARD,
// Persistent=true, DevGated=false, Kind=discussion,
// StructuralType=droplet, ParentID="" (level_1 under the project root).
func TestAutoGenSeeds6StewardPersistentParents(t *testing.T) {
	withSeedTemplateFixture(t, func() (templates.Template, error) {
		return templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			StewardSeeds:  canonicalSixSeeds(),
		}, nil
	})

	svc, repo := newSeederService(t)
	project, err := svc.CreateProject(context.Background(), "Auto-Gen Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	stewardItems := make([]domain.ActionItem, 0, 6)
	for _, item := range repo.tasks {
		if item.ProjectID != project.ID {
			continue
		}
		if item.Owner != stewardOwner {
			continue
		}
		stewardItems = append(stewardItems, item)
	}
	if got := len(stewardItems); got != 6 {
		t.Fatalf("expected 6 STEWARD-owned anchor items, got %d", got)
	}

	sort.Slice(stewardItems, func(i, j int) bool {
		return stewardItems[i].Title < stewardItems[j].Title
	})
	wantTitles := []string{"DISCUSSIONS", "HYLLA_FINDINGS", "HYLLA_REFINEMENTS", "LEDGER", "REFINEMENTS", "WIKI_CHANGELOG"}
	for i, item := range stewardItems {
		if item.Title != wantTitles[i] {
			t.Fatalf("seeded title[%d] = %q, want %q", i, item.Title, wantTitles[i])
		}
		if item.Owner != stewardOwner {
			t.Fatalf("seeded[%q] Owner = %q, want %q", item.Title, item.Owner, stewardOwner)
		}
		if item.Persistent != true {
			t.Fatalf("seeded[%q] Persistent = %v, want true", item.Title, item.Persistent)
		}
		if item.DevGated != false {
			t.Fatalf("seeded[%q] DevGated = %v, want false", item.Title, item.DevGated)
		}
		if item.DropNumber != 0 {
			t.Fatalf("seeded[%q] DropNumber = %d, want 0", item.Title, item.DropNumber)
		}
		if item.Kind != domain.KindDiscussion {
			t.Fatalf("seeded[%q] Kind = %q, want %q", item.Title, item.Kind, domain.KindDiscussion)
		}
		if item.StructuralType != domain.StructuralTypeDroplet {
			t.Fatalf("seeded[%q] StructuralType = %q, want %q", item.Title, item.StructuralType, domain.StructuralTypeDroplet)
		}
		if item.ParentID != "" {
			t.Fatalf("seeded[%q] ParentID = %q, want empty (level_1 under project root)", item.Title, item.ParentID)
		}
	}
}

// TestAutoGenSeedsIdempotentOnReseed verifies re-running the seed path on
// an already-seeded project does NOT duplicate rows. The auto-generator
// keys idempotency on (project_id, owner=STEWARD, title) — a known seed
// title resolves to the existing row and the seeder skips it.
func TestAutoGenSeedsIdempotentOnReseed(t *testing.T) {
	withSeedTemplateFixture(t, func() (templates.Template, error) {
		return templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			StewardSeeds:  canonicalSixSeeds(),
		}, nil
	})

	svc, repo := newSeederService(t)
	project, err := svc.CreateProject(context.Background(), "Idempotency Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	// Re-run the seed path directly. Production code only calls this once
	// at project creation; the second invocation simulates retry-after-
	// transient-failure or a future explicit re-seed call.
	if err := svc.seedStewardAnchors(context.Background(), project); err != nil {
		t.Fatalf("seedStewardAnchors() second call error = %v", err)
	}

	stewardCount := 0
	for _, item := range repo.tasks {
		if item.ProjectID == project.ID && item.Owner == stewardOwner {
			stewardCount++
		}
	}
	if stewardCount != 6 {
		t.Fatalf("expected 6 STEWARD anchors after re-seed (idempotent), got %d", stewardCount)
	}
}

// TestAutoGenSeedsLevel2FindingsOnNumberedDropCreation verifies that
// creating a level_1 numbered drop (parent_id="" + drop_number > 0)
// materializes the canonical 5 STEWARD-owned level_2 findings under the
// matching anchor parents AND a refinements-gate confluence inside the
// drop's tree with the expected blocked_by wiring.
func TestAutoGenSeedsLevel2FindingsOnNumberedDropCreation(t *testing.T) {
	withSeedTemplateFixture(t, func() (templates.Template, error) {
		return templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			StewardSeeds:  canonicalSixSeeds(),
		}, nil
	})

	svc, repo := newSeederService(t)
	project, err := svc.CreateProject(context.Background(), "Drop Findings Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected auto-created project columns")
	}

	const dropN = 3
	drop, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       columns[0].ID,
		Kind:           domain.KindPlan,
		StructuralType: domain.StructuralTypeDrop,
		DropNumber:     dropN,
		Title:          "DROP_3",
		Description:    "Numbered drop 3.",
		Priority:       domain.PriorityMedium,
	})
	if err != nil {
		t.Fatalf("CreateActionItem(level_1 numbered drop) error = %v", err)
	}

	wantSuffixes := []string{
		"DROP_3_HYLLA_FINDINGS",
		"DROP_3_LEDGER_ENTRY",
		"DROP_3_WIKI_CHANGELOG_ENTRY",
		"DROP_3_REFINEMENTS_RAISED",
		"DROP_3_HYLLA_REFINEMENTS_RAISED",
	}
	wantParents := map[string]string{
		"DROP_3_HYLLA_FINDINGS":           "HYLLA_FINDINGS",
		"DROP_3_LEDGER_ENTRY":             "LEDGER",
		"DROP_3_WIKI_CHANGELOG_ENTRY":     "WIKI_CHANGELOG",
		"DROP_3_REFINEMENTS_RAISED":       "REFINEMENTS",
		"DROP_3_HYLLA_REFINEMENTS_RAISED": "HYLLA_REFINEMENTS",
	}
	findingsByTitle := map[string]domain.ActionItem{}
	for _, suffix := range wantSuffixes {
		var match *domain.ActionItem
		for _, item := range repo.tasks {
			if item.ProjectID == project.ID && item.Title == suffix {
				v := item
				match = &v
				break
			}
		}
		if match == nil {
			t.Fatalf("expected finding %q to exist after numbered-drop creation", suffix)
		}
		if match.Owner != stewardOwner {
			t.Fatalf("finding %q Owner = %q, want %q", suffix, match.Owner, stewardOwner)
		}
		if match.DropNumber != dropN {
			t.Fatalf("finding %q DropNumber = %d, want %d", suffix, match.DropNumber, dropN)
		}
		if match.Persistent != false {
			t.Fatalf("finding %q Persistent = %v, want false", suffix, match.Persistent)
		}
		if match.DevGated != false {
			t.Fatalf("finding %q DevGated = %v, want false", suffix, match.DevGated)
		}
		if match.Kind != domain.KindDiscussion {
			t.Fatalf("finding %q Kind = %q, want %q", suffix, match.Kind, domain.KindDiscussion)
		}
		if match.StructuralType != domain.StructuralTypeDroplet {
			t.Fatalf("finding %q StructuralType = %q, want %q", suffix, match.StructuralType, domain.StructuralTypeDroplet)
		}
		// Parent must be the matching STEWARD anchor.
		var anchor *domain.ActionItem
		for _, item := range repo.tasks {
			if item.ProjectID == project.ID && item.Owner == stewardOwner && item.Title == wantParents[suffix] {
				v := item
				anchor = &v
				break
			}
		}
		if anchor == nil {
			t.Fatalf("expected anchor %q to exist", wantParents[suffix])
		}
		if match.ParentID != anchor.ID {
			t.Fatalf("finding %q ParentID = %q, want anchor %q (id=%q)", suffix, match.ParentID, wantParents[suffix], anchor.ID)
		}
		findingsByTitle[suffix] = *match
	}

	// Refinements-gate confluence assertions.
	gateTitle := "DROP_3_REFINEMENTS_GATE_BEFORE_DROP_4"
	var gate *domain.ActionItem
	for _, item := range repo.tasks {
		if item.ProjectID == project.ID && item.Title == gateTitle {
			v := item
			gate = &v
			break
		}
	}
	if gate == nil {
		t.Fatalf("expected refinements-gate %q", gateTitle)
	}
	if gate.Kind != domain.KindPlan {
		t.Fatalf("gate Kind = %q, want %q", gate.Kind, domain.KindPlan)
	}
	if gate.StructuralType != domain.StructuralTypeConfluence {
		t.Fatalf("gate StructuralType = %q, want %q", gate.StructuralType, domain.StructuralTypeConfluence)
	}
	if gate.Owner != stewardOwner {
		t.Fatalf("gate Owner = %q, want %q", gate.Owner, stewardOwner)
	}
	if gate.DropNumber != dropN {
		t.Fatalf("gate DropNumber = %d, want %d", gate.DropNumber, dropN)
	}
	if gate.DevGated != true {
		t.Fatalf("gate DevGated = %v, want true (dev sign-off required)", gate.DevGated)
	}
	if gate.ParentID != drop.ID {
		t.Fatalf("gate ParentID = %q, want drop %q", gate.ParentID, drop.ID)
	}

	// blocked_by must enumerate the drop itself + every finding (drop_number=N)
	// and exclude the gate's own ID.
	blockedSet := map[string]struct{}{}
	for _, id := range gate.Metadata.BlockedBy {
		blockedSet[id] = struct{}{}
	}
	if _, ok := blockedSet[drop.ID]; !ok {
		t.Fatalf("gate blocked_by missing drop id %q (got %v)", drop.ID, gate.Metadata.BlockedBy)
	}
	for _, finding := range findingsByTitle {
		if _, ok := blockedSet[finding.ID]; !ok {
			t.Fatalf("gate blocked_by missing finding id %q (title=%q, got %v)", finding.ID, finding.Title, gate.Metadata.BlockedBy)
		}
	}
	if _, ok := blockedSet[gate.ID]; ok {
		t.Fatalf("gate blocked_by must not include the gate's own id %q (got %v)", gate.ID, gate.Metadata.BlockedBy)
	}
}

// TestAutoGenSeedsSkipsNonNumberedDrop verifies the level_2 finding +
// refinements-gate auto-generation does NOT fire for non-numbered drops
// (drop_number=0). Level_1 items without a drop_number are normal cascade
// nodes, not numbered drops.
func TestAutoGenSeedsSkipsNonNumberedDrop(t *testing.T) {
	withSeedTemplateFixture(t, func() (templates.Template, error) {
		return templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			StewardSeeds:  canonicalSixSeeds(),
		}, nil
	})

	svc, repo := newSeederService(t)
	project, err := svc.CreateProject(context.Background(), "Non-Numbered Demo", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected auto-created project columns")
	}

	if _, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       columns[0].ID,
		Kind:           domain.KindPlan,
		StructuralType: domain.StructuralTypeDroplet,
		DropNumber:     0,
		Title:          "NON_NUMBERED_PLAN",
		Description:    "A normal level_1 plan with no drop_number.",
		Priority:       domain.PriorityMedium,
	}); err != nil {
		t.Fatalf("CreateActionItem(non-numbered) error = %v", err)
	}

	for _, item := range repo.tasks {
		if item.ProjectID != project.ID {
			continue
		}
		// No drop-suffixed findings should exist.
		switch item.Title {
		case "DROP_0_HYLLA_FINDINGS",
			"DROP_0_LEDGER_ENTRY",
			"DROP_0_WIKI_CHANGELOG_ENTRY",
			"DROP_0_REFINEMENTS_RAISED",
			"DROP_0_HYLLA_REFINEMENTS_RAISED",
			"DROP_0_REFINEMENTS_GATE_BEFORE_DROP_1":
			t.Fatalf("unexpected drop-finding %q created for non-numbered drop", item.Title)
		}
	}
}

// TestAutoGenSeedsRejectsMissingAnchor verifies the level_2 finding
// auto-generator returns errStewardParentNotSeeded when a level_1
// numbered drop creation runs against a project whose STEWARD persistent
// anchors were never seeded — the safety net the code documents.
func TestAutoGenSeedsRejectsMissingAnchor(t *testing.T) {
	// Project creation seeds zero anchors.
	withSeedTemplateFixture(t, func() (templates.Template, error) {
		return templates.Template{SchemaVersion: templates.SchemaVersionV1}, nil
	})

	svc, _ := newSeederService(t)
	project, err := svc.CreateProject(context.Background(), "No Anchors", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	columns, err := svc.ListColumns(context.Background(), project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) == 0 {
		t.Fatal("expected auto-created project columns")
	}

	// Numbered drop creation should fail — no anchors to parent findings.
	_, err = svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       columns[0].ID,
		Kind:           domain.KindPlan,
		StructuralType: domain.StructuralTypeDrop,
		DropNumber:     5,
		Title:          "DROP_5",
		Priority:       domain.PriorityMedium,
	})
	if err == nil {
		t.Fatal("CreateActionItem(level_1 numbered drop with no anchors) error = nil, want errStewardParentNotSeeded")
	}
	if !errors.Is(err, errStewardParentNotSeeded) {
		t.Fatalf("CreateActionItem error = %v, want wrap of errStewardParentNotSeeded", err)
	}
}
