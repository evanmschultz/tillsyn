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

// TestRaiseRefinementsGateForgottenAttentionIsIdempotent verifies the
// Drop 4c.5 droplet C.2 idempotency contract on
// raiseRefinementsGateForgottenAttention: re-running gate-close against an
// already-warned drop must NOT create a second attention item AND must NOT
// re-build the warning body. The helper looks up the deterministic
// attention id `refinements-gate-forgotten::<gate.ID>` BEFORE constructing
// the new attention; a non-ErrNotFound hit short-circuits to a no-op
// success.
//
// The test arranges a numbered drop whose auto-generated 5 STEWARD-owned
// findings supply the "stragglers" the safety-net warns about (all
// findings start in the todo state — non-terminal — and parent under the
// HYLLA_FINDINGS / LEDGER / WIKI_CHANGELOG / REFINEMENTS / HYLLA_REFINEMENTS
// anchors, so they are NOT excluded by the level_1-drop filter at line 391).
// The first helper call creates the warning; the test then mutates the
// stored attention's Summary in-place. A second helper call MUST leave that
// mutation intact, proving the second call took the early-return path
// instead of calling the fake's CreateAttentionItem (which overwrites the
// map entry on every call).
func TestRaiseRefinementsGateForgottenAttentionIsIdempotent(t *testing.T) {
	withSeedTemplateFixture(t, func() (templates.Template, error) {
		return templates.Template{
			SchemaVersion: templates.SchemaVersionV1,
			StewardSeeds:  canonicalSixSeeds(),
		}, nil
	})

	svc, repo := newSeederService(t)
	project, err := svc.CreateProject(context.Background(), "Idempotent Safety-Net Demo", "")
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

	const dropN = 7
	if _, err := svc.CreateActionItem(context.Background(), CreateActionItemInput{
		ProjectID:      project.ID,
		ColumnID:       columns[0].ID,
		Kind:           domain.KindPlan,
		StructuralType: domain.StructuralTypeDrop,
		DropNumber:     dropN,
		Title:          "DROP_7",
		Description:    "Numbered drop 7 for idempotency check.",
		Priority:       domain.PriorityMedium,
	}); err != nil {
		t.Fatalf("CreateActionItem(numbered drop) error = %v", err)
	}

	gateTitle := "DROP_7_REFINEMENTS_GATE_BEFORE_DROP_8"
	var gate domain.ActionItem
	for _, item := range repo.tasks {
		if item.ProjectID == project.ID && item.Title == gateTitle {
			gate = item
			break
		}
	}
	if gate.ID == "" {
		t.Fatalf("expected refinements-gate %q to exist after numbered-drop creation", gateTitle)
	}

	// First call: stragglers exist (the 5 STEWARD findings are todo), so
	// the helper builds + persists exactly one attention.
	if err := svc.raiseRefinementsGateForgottenAttention(context.Background(), gate); err != nil {
		t.Fatalf("first raiseRefinementsGateForgottenAttention() error = %v", err)
	}
	wantAttentionID := "refinements-gate-forgotten::" + gate.ID
	stored, ok := repo.attentionItems[wantAttentionID]
	if !ok {
		t.Fatalf("expected attention id %q after first call (got map keys %v)", wantAttentionID, attentionKeys(repo.attentionItems))
	}
	if stored.Summary == "" {
		t.Fatalf("expected non-empty Summary on first-call attention")
	}

	// Mutate the stored attention's Summary in-place. If the second call
	// re-enters CreateAttentionItem the fake will overwrite this with the
	// freshly-built body and our sentinel disappears — proving the helper
	// re-ran the create branch instead of taking the idempotent early
	// return.
	const sentinelMarker = "C.2 IDEMPOTENT-CHECK SENTINEL"
	mutated := stored
	mutated.Summary = sentinelMarker
	repo.attentionItems[wantAttentionID] = mutated

	// Second call on the SAME gate — must observe the existing attention
	// via GetAttentionItem and short-circuit. CreateAttentionItem must NOT
	// be called a second time.
	if err := svc.raiseRefinementsGateForgottenAttention(context.Background(), gate); err != nil {
		t.Fatalf("second raiseRefinementsGateForgottenAttention() error = %v", err)
	}

	// Total attention count for this deterministic id must remain 1.
	count := 0
	for id := range repo.attentionItems {
		if id == wantAttentionID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 attention at id %q after two calls, got %d", wantAttentionID, count)
	}

	// Sentinel must survive — proves the second call did NOT invoke
	// CreateAttentionItem (which would overwrite Summary back to the
	// freshly-built value).
	after := repo.attentionItems[wantAttentionID]
	if after.Summary != sentinelMarker {
		t.Fatalf("second call overwrote sentinel: Summary = %q, want %q (helper re-entered create path instead of taking idempotent early return)", after.Summary, sentinelMarker)
	}
}

// attentionKeys returns a sorted slice of attention-item ids in the supplied
// map, used by TestRaiseRefinementsGateForgottenAttentionIsIdempotent to
// produce stable diagnostics when the expected key is missing.
func attentionKeys(m map[string]domain.AttentionItem) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
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

// TestIsRefinementsGateAcceptsCanonicalTitle verifies the predicate returns
// true for an ActionItem carrying the full canonical refinements-gate shape
// — Owner=STEWARD, StructuralType=Confluence, DropNumber>0, and a title built
// by refinementsGateTitle. Covers the single-digit drop number (DROP_4 →
// DROP_5) plus the double-digit edge case (DROP_10 → DROP_11) called out in
// the Drop 4c.5 droplet C.3 spec.
func TestIsRefinementsGateAcceptsCanonicalTitle(t *testing.T) {
	cases := []struct {
		name       string
		dropNumber int
		wantTitle  string
	}{
		{
			name:       "single digit drop 4",
			dropNumber: 4,
			wantTitle:  "DROP_4_REFINEMENTS_GATE_BEFORE_DROP_5",
		},
		{
			name:       "double digit drop 10",
			dropNumber: 10,
			wantTitle:  "DROP_10_REFINEMENTS_GATE_BEFORE_DROP_11",
		},
		{
			name:       "triple digit drop 100",
			dropNumber: 100,
			wantTitle:  "DROP_100_REFINEMENTS_GATE_BEFORE_DROP_101",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			built := refinementsGateTitle(tc.dropNumber)
			if built != tc.wantTitle {
				t.Fatalf("refinementsGateTitle(%d) = %q, want %q", tc.dropNumber, built, tc.wantTitle)
			}
			gate := domain.ActionItem{
				Owner:          stewardOwner,
				StructuralType: domain.StructuralTypeConfluence,
				DropNumber:     tc.dropNumber,
				Title:          built,
			}
			if !isRefinementsGate(gate) {
				t.Fatalf("isRefinementsGate(canonical %q) = false, want true", built)
			}
		})
	}
}

// TestIsRefinementsGateRejectsForeignSTEWARDConfluence verifies the predicate
// returns false for a STEWARD-owned numbered confluence whose Title is NOT a
// refinements-gate. Covers the falsification surface called out in droplet
// C.3 — pre-C.3 the predicate matched on Owner / StructuralType / DropNumber
// alone, so any future STEWARD-owned numbered confluence (e.g. a hypothetical
// MERGE_WINDOW_GATE) would have tripped raiseRefinementsGateForgottenAttention's
// safety-net path. Post-C.3 the title-shape check rejects every such row.
//
// Also covers DropNumber=0 (existing rule preserved) plus several adversarial
// title shapes that satisfy one of the new title checks but not both.
func TestIsRefinementsGateRejectsForeignSTEWARDConfluence(t *testing.T) {
	cases := []struct {
		name string
		item domain.ActionItem
	}{
		{
			name: "foreign STEWARD-owned numbered confluence with arbitrary title",
			item: domain.ActionItem{
				Owner:          stewardOwner,
				StructuralType: domain.StructuralTypeConfluence,
				DropNumber:     5,
				Title:          "DROP_5_MERGE_WINDOW_GATE",
			},
		},
		{
			name: "title missing DROP_ prefix",
			item: domain.ActionItem{
				Owner:          stewardOwner,
				StructuralType: domain.StructuralTypeConfluence,
				DropNumber:     5,
				Title:          "5_REFINEMENTS_GATE_BEFORE_DROP_6",
			},
		},
		{
			name: "title has DROP_ prefix but no refinements-gate infix",
			item: domain.ActionItem{
				Owner:          stewardOwner,
				StructuralType: domain.StructuralTypeConfluence,
				DropNumber:     5,
				Title:          "DROP_5_HYLLA_FINDINGS",
			},
		},
		{
			name: "DropNumber=0 with canonical title still rejects (existing rule preserved)",
			item: domain.ActionItem{
				Owner:          stewardOwner,
				StructuralType: domain.StructuralTypeConfluence,
				DropNumber:     0,
				Title:          "DROP_0_REFINEMENTS_GATE_BEFORE_DROP_1",
			},
		},
		{
			name: "non-STEWARD owner with canonical title",
			item: domain.ActionItem{
				Owner:          "drop-orch",
				StructuralType: domain.StructuralTypeConfluence,
				DropNumber:     5,
				Title:          "DROP_5_REFINEMENTS_GATE_BEFORE_DROP_6",
			},
		},
		{
			name: "non-confluence structural type with canonical title",
			item: domain.ActionItem{
				Owner:          stewardOwner,
				StructuralType: domain.StructuralTypeDroplet,
				DropNumber:     5,
				Title:          "DROP_5_REFINEMENTS_GATE_BEFORE_DROP_6",
			},
		},
		{
			name: "empty title",
			item: domain.ActionItem{
				Owner:          stewardOwner,
				StructuralType: domain.StructuralTypeConfluence,
				DropNumber:     5,
				Title:          "",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if isRefinementsGate(tc.item) {
				t.Fatalf("isRefinementsGate(%+v) = true, want false", tc.item)
			}
		})
	}
}
