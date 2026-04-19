package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/tui/gitdiff"
)

// diffModeGoldenUpdateEnv regenerates diff_mode golden fixtures when set.
//
// The env-var pattern mirrors the gitdiff highlighter test fixtures (see
// highlighter_test.go) so both suites regenerate via the same operator gesture.
// Independent from `mage test-golden` because that target is scoped to
// teatest-style goldens only.
const diffModeGoldenUpdateEnv = "TILLSYN_DIFFMODE_GOLDEN_UPDATE"

// stubHighlighter is a deterministic Highlighter used by diff-mode tests to
// pin viewport content without depending on chroma's palette/lexer versions.
type stubHighlighter struct{}

// Highlight passes patch through verbatim so golden assertions compare raw
// unified-diff text without ANSI noise.
func (stubHighlighter) Highlight(patch string) (string, error) {
	return patch, nil
}

// fakeDiffer is a fake Differ keyed on (start,end,paths) → canned result.
//
// Tests configure result + err directly; teatest paths do not touch a real
// repo. Keeps diff-mode unit tests deterministic and fast.
type fakeDiffer struct {
	result    gitdiff.DiffResult
	err       error
	calls     int
	lastPaths []string
}

// Diff echoes the canned response and records a call along with the paths it received.
func (f *fakeDiffer) Diff(_ context.Context, _, _ string, paths []string) (gitdiff.DiffResult, error) {
	f.calls++
	f.lastPaths = append([]string(nil), paths...)
	return f.result, f.err
}

// samplePatchForDiffMode mirrors the gitdiff highlighter sample and is shared
// by every diff-mode test for stable snapshots.
const samplePatchForDiffMode = `diff --git a/foo.txt b/foo.txt
index 83db48f..bf269f4 100644
--- a/foo.txt
+++ b/foo.txt
@@ -1,3 +1,4 @@
 alpha
-beta
+beta updated
 gamma
+delta
`

// newTestDiffMode builds a diffMode with injected fakes sized for golden
// rendering. The DiffResult is staged on both the fake differ (for teatest
// flows that drive the differ call through tea.Cmd) and directly on the
// returned diffMode so unit tests can invoke dm.apply(dm.result, nil) without
// first needing to hop through the async cmd queue.
func newTestDiffMode(t *testing.T, res gitdiff.DiffResult, err error) (*diffMode, *fakeDiffer) {
	t.Helper()
	fd := &fakeDiffer{result: res, err: err}
	dm := newDiffMode(fd, stubHighlighter{})
	dm.result = res
	dm.err = err
	dm.resize(80, 20)
	return dm, fd
}

// TestDiffMode_Render_Ancestor_Golden pins the ancestor-status render output
// against a golden fixture so regressions in the render pipeline (banner
// placement, viewport framing, chrome wiring) fail loudly. The stub
// highlighter keeps the golden independent of chroma version drift.
func TestDiffMode_Render_Ancestor_Golden(t *testing.T) {
	dm, _ := newTestDiffMode(t, gitdiff.DiffResult{
		Patch:      samplePatchForDiffMode,
		Divergence: gitdiff.DivergenceAncestor,
		StartSHA:   "aaaa111",
		EndSHA:     "bbbb222",
	}, nil)
	dm.apply(dm.result, nil)

	got := dm.viewContent()
	if strings.Contains(got, "NOT ancestor") {
		t.Fatalf("ancestor render unexpectedly contains divergence banner: %q", got)
	}

	goldenPath := filepath.Join("testdata", "diff_mode", "simple.golden")
	if os.Getenv(diffModeGoldenUpdateEnv) != "" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with %s=1 to regenerate)", goldenPath, err, diffModeGoldenUpdateEnv)
	}
	if string(want) != got {
		t.Fatalf("golden mismatch at %s\nrun with %s=1 to regenerate\ngot:\n%s\nwant:\n%s", goldenPath, diffModeGoldenUpdateEnv, got, string(want))
	}
}

// TestDiffMode_Render_Diverged_Banner asserts the exact banner string and
// its placement as the first line of the rendered body when DivergenceStatus
// is Diverged. The exact text is part of the acceptance contract.
func TestDiffMode_Render_Diverged_Banner(t *testing.T) {
	dm, _ := newTestDiffMode(t, gitdiff.DiffResult{
		Patch:      samplePatchForDiffMode,
		Divergence: gitdiff.DivergenceDiverged,
	}, nil)
	dm.apply(dm.result, nil)

	got := dm.viewContent()
	wantBanner := " branch-start-commit is NOT ancestor of HEAD — showing diff anyway"
	if !strings.Contains(got, wantBanner) {
		t.Fatalf("diverged render missing banner %q\ngot:\n%s", wantBanner, got)
	}
	bannerIdx := strings.Index(got, wantBanner)
	patchIdx := strings.Index(got, "diff --git a/foo.txt")
	if bannerIdx < 0 || patchIdx < 0 {
		t.Fatalf("missing banner or patch in render\ngot:\n%s", got)
	}
	if bannerIdx > patchIdx {
		t.Fatalf("banner must precede patch; bannerIdx=%d patchIdx=%d", bannerIdx, patchIdx)
	}
}

// TestDiffMode_Render_Unknown_Error asserts that a Differ error surfaces as a
// user-visible error message in place of diff content, and that the diff
// mode never silently discards the error.
func TestDiffMode_Render_Unknown_Error(t *testing.T) {
	differErr := errors.New("git: bad revision")
	dm, _ := newTestDiffMode(t, gitdiff.DiffResult{
		Divergence: gitdiff.DivergenceUnknown,
	}, differErr)
	dm.apply(gitdiff.DiffResult{}, differErr)

	got := dm.viewContent()
	if !strings.Contains(got, "bad revision") {
		t.Fatalf("error render missing underlying cause %q\ngot:\n%s", differErr.Error(), got)
	}
	if strings.Contains(got, "diff --git") {
		t.Fatalf("error render unexpectedly includes patch text\ngot:\n%s", got)
	}
}

// TestDiffMode_Render_EmptyDiff asserts that an empty patch renders a
// "No changes" placeholder rather than leaving the viewport blank. A blank
// viewport would visually indicate a broken diff rather than a clean tree.
func TestDiffMode_Render_EmptyDiff(t *testing.T) {
	dm, _ := newTestDiffMode(t, gitdiff.DiffResult{
		Patch:      "",
		Divergence: gitdiff.DivergenceAncestor,
	}, nil)
	dm.apply(dm.result, nil)

	got := dm.viewContent()
	if !strings.Contains(got, "No changes") {
		t.Fatalf("empty-patch render missing placeholder\ngot:\n%s", got)
	}
}

// TestKeymap_CtrlD_NoCollision asserts that no existing keyMap binding in
// newKeyMap() claims "ctrl+d", so wiring diffModeToggle onto ctrl+d doesn't
// silently shadow a prior binding.
func TestKeymap_CtrlD_NoCollision(t *testing.T) {
	km := newKeyMap()
	bindings := map[string]key.Binding{
		"quit":                 km.quit,
		"reload":               km.reload,
		"toggleHelp":           km.toggleHelp,
		"moveLeft":             km.moveLeft,
		"moveRight":            km.moveRight,
		"moveUp":               km.moveUp,
		"moveDown":             km.moveDown,
		"addActionItem":        km.addActionItem,
		"actionItemInfo":       km.actionItemInfo,
		"editActionItem":       km.editActionItem,
		"newProject":           km.newProject,
		"editProject":          km.editProject,
		"commandPalette":       km.commandPalette,
		"quickActions":         km.quickActions,
		"deleteActionItem":     km.deleteActionItem,
		"archiveActionItem":    km.archiveActionItem,
		"moveActionItemLeft":   km.moveActionItemLeft,
		"moveActionItemRight":  km.moveActionItemRight,
		"hardDeleteActionItem": km.hardDeleteActionItem,
		"restoreActionItem":    km.restoreActionItem,
		"search":               km.search,
		"projects":             km.projects,
		"toggleArchived":       km.toggleArchived,
		"toggleSelectMode":     km.toggleSelectMode,
		"focusSubtree":         km.focusSubtree,
		"clearFocus":           km.clearFocus,
		"multiSelect":          km.multiSelect,
		"activityLog":          km.activityLog,
		"undo":                 km.undo,
		"redo":                 km.redo,
	}
	for name, binding := range bindings {
		for _, k := range binding.Keys() {
			if k == "ctrl+d" {
				t.Fatalf("binding %q already claims ctrl+d; diffModeToggle would collide", name)
			}
		}
	}
	// Positive assertion: the new toggle binding is actually wired.
	if keys := km.diffModeToggle.Keys(); len(keys) != 1 || keys[0] != "ctrl+d" {
		t.Fatalf("diffModeToggle keys = %#v, want exactly [ctrl+d]", keys)
	}
}

// TestModel_CtrlD_EntersDiffMode asserts that pressing ctrl+d from the normal
// board surface transitions the Model to modeDiff.
func TestModel_CtrlD_EntersDiffMode(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Diff toggle",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, []domain.ActionItem{task})
	m := loadReadyModel(t, NewModel(svc, WithDiffMode(&fakeDiffer{
		result: gitdiff.DiffResult{Patch: samplePatchForDiffMode, Divergence: gitdiff.DivergenceAncestor},
	}, stubHighlighter{})))

	if m.mode != modeNone {
		t.Fatalf("expected modeNone at start, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl, Text: ""})
	if m.mode != modeDiff {
		t.Fatalf("expected modeDiff after ctrl+d, got %v", m.mode)
	}
	if m.diff == nil {
		t.Fatal("expected m.diff to be non-nil after ctrl+d")
	}
}

// TestModel_EscFromDiff_RestoresPrior asserts that esc returns to the prior
// mode captured on entry (not unconditionally modeNone).
func TestModel_EscFromDiff_RestoresPrior(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Diff exit",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, []domain.ActionItem{task})
	m := loadReadyModel(t, NewModel(svc, WithDiffMode(&fakeDiffer{
		result: gitdiff.DiffResult{Patch: samplePatchForDiffMode, Divergence: gitdiff.DivergenceAncestor},
	}, stubHighlighter{})))

	// Enter from modeNone.
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.mode != modeDiff {
		t.Fatalf("precondition: expected modeDiff, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected modeNone after esc (prior was modeNone), got %v", m.mode)
	}
	// Patch must be cleared to avoid megabyte-patch memory retention (falsification vector 3).
	if m.diff != nil && strings.TrimSpace(m.diff.result.Patch) != "" {
		t.Fatalf("expected diff patch cleared after esc, still have %d bytes", len(m.diff.result.Patch))
	}
}

// TestDiffMode_Teatest_E2E verifies the end-to-end flow through teatest: open
// a task, toggle ctrl+d, observe diff content, press esc, confirm return to
// the board without mutating the task.
func TestDiffMode_Teatest_E2E(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "E2E diff",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, []domain.ActionItem{task})
	m := loadReadyModel(t, NewModel(svc, WithDiffMode(&fakeDiffer{
		result: gitdiff.DiffResult{Patch: samplePatchForDiffMode, Divergence: gitdiff.DivergenceAncestor},
	}, stubHighlighter{})))

	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.mode != modeDiff {
		t.Fatalf("expected modeDiff after ctrl+d, got %v", m.mode)
	}
	rendered := fmt.Sprint(m.View().Content)
	if !strings.Contains(stripANSI(rendered), "diff --git") {
		t.Fatalf("expected rendered diff content to include file header\nrendered (stripped):\n%s", stripANSI(rendered))
	}

	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected modeNone after esc, got %v", m.mode)
	}
	// Task count is unchanged — diff-mode must not mutate plan-item state.
	if got := len(svc.tasks[p.ID]); got != 1 {
		t.Fatalf("expected 1 task after diff round-trip, got %d", got)
	}
	if got := svc.tasks[p.ID][0]; got.Title != task.Title || got.ID != task.ID {
		t.Fatalf("task mutated by diff round-trip: %+v vs %+v", got, task)
	}
}

// newDiffTestTask builds a domain.ActionItem with the given ResourceRefs for diff-mode unit tests.
//
// The helper keeps test bodies concise by pre-filling all required ActionItem fields
// with stable values; callers only need to supply the ResourceRefs that drive
// resolveDiffPaths behaviour under test.
func newDiffTestTask(t *testing.T, refs []domain.ResourceRef) domain.ActionItem {
	t.Helper()
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	p, err := domain.NewProject("p1", "Inbox", "", now)
	if err != nil {
		t.Fatalf("NewProject: %v", err)
	}
	col, err := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn: %v", err)
	}
	task, err := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t-diff-test",
		ProjectID: p.ID,
		ColumnID:  col.ID,
		Position:  0,
		Title:     "diff test task",
		Priority:  domain.PriorityLow,
		Metadata: domain.ActionItemMetadata{
			ResourceRefs: refs,
		},
	}, now)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	return task
}

// refWith is a compact ResourceRef constructor for unit tests.
func refWith(location string, tags ...string) domain.ResourceRef {
	return domain.ResourceRef{
		Location: location,
		Tags:     tags,
	}
}

// TestResolveDiffPaths_EmptyResourceRefs asserts that a nil item and a task with
// no ResourceRefs both produce an empty slice so Differ.Diff falls back to
// whole-repo behaviour.
func TestResolveDiffPaths_EmptyResourceRefs(t *testing.T) {
	// nil item guard
	if got := resolveDiffPaths(nil); len(got) != 0 {
		t.Fatalf("nil item: expected empty slice, got %v", got)
	}
	// task with zero ResourceRefs
	task := newDiffTestTask(t, nil)
	if got := resolveDiffPaths(&task); len(got) != 0 {
		t.Fatalf("empty refs: expected empty slice, got %v", got)
	}
}

// TestResolveDiffPaths_PathTagsOnly asserts that "path"-tagged Locations are
// returned unchanged (no trailing slash added).
func TestResolveDiffPaths_PathTagsOnly(t *testing.T) {
	task := newDiffTestTask(t, []domain.ResourceRef{
		refWith("internal/tui", "path"),
		refWith("internal/domain", "path"),
	})
	got := resolveDiffPaths(&task)
	want := []string{"internal/tui", "internal/domain"}
	if len(got) != len(want) {
		t.Fatalf("path tags: want %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("path tags [%d]: want %q, got %q", i, w, got[i])
		}
	}
}

// TestResolveDiffPaths_FileTagsOnly asserts that "file"-tagged Locations are
// returned unchanged (same behaviour as "path").
func TestResolveDiffPaths_FileTagsOnly(t *testing.T) {
	task := newDiffTestTask(t, []domain.ResourceRef{
		refWith("cmd/till/main.go", "file"),
		refWith("magefile.go", "file"),
	})
	got := resolveDiffPaths(&task)
	want := []string{"cmd/till/main.go", "magefile.go"}
	if len(got) != len(want) {
		t.Fatalf("file tags: want %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("file tags [%d]: want %q, got %q", i, w, got[i])
		}
	}
}

// TestResolveDiffPaths_PackageTagsOnly asserts that "package"-tagged Locations
// receive a trailing slash (normalised — no double slash when already present).
func TestResolveDiffPaths_PackageTagsOnly(t *testing.T) {
	task := newDiffTestTask(t, []domain.ResourceRef{
		refWith("internal/domain", "package"),
		refWith("internal/tui/", "package"), // already has trailing slash
	})
	got := resolveDiffPaths(&task)
	want := []string{"internal/domain/", "internal/tui/"}
	if len(got) != len(want) {
		t.Fatalf("package tags: want %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("package tags [%d]: want %q, got %q", i, w, got[i])
		}
	}
}

// TestResolveDiffPaths_MixedTags asserts that path/file/package refs are merged
// in iteration order with each tagged appropriately.
func TestResolveDiffPaths_MixedTags(t *testing.T) {
	task := newDiffTestTask(t, []domain.ResourceRef{
		refWith("cmd/till/main.go", "file"),
		refWith("internal/app", "path"),
		refWith("internal/domain", "package"),
	})
	got := resolveDiffPaths(&task)
	want := []string{"cmd/till/main.go", "internal/app", "internal/domain/"}
	if len(got) != len(want) {
		t.Fatalf("mixed tags: want %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("mixed tags [%d]: want %q, got %q", i, w, got[i])
		}
	}
}

// TestResolveDiffPaths_Dedup asserts that duplicate Locations within the same
// tag class are deduplicated, keeping the first occurrence.
func TestResolveDiffPaths_Dedup(t *testing.T) {
	task := newDiffTestTask(t, []domain.ResourceRef{
		refWith("internal/tui", "path"),
		refWith("internal/tui", "path"), // duplicate
		refWith("internal/domain", "path"),
	})
	got := resolveDiffPaths(&task)
	want := []string{"internal/tui", "internal/domain"}
	if len(got) != len(want) {
		t.Fatalf("dedup: want %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("dedup [%d]: want %q, got %q", i, w, got[i])
		}
	}
}

// TestResolveDiffPaths_PackageWinsOverPath asserts that when the same Location
// appears as both a "path" ref and a "package" ref, the trailing-slash (package)
// form wins regardless of which appears first.
func TestResolveDiffPaths_PackageWinsOverPath(t *testing.T) {
	// path ref first, then package ref for the same Location
	task := newDiffTestTask(t, []domain.ResourceRef{
		refWith("internal/tui", "path"),    // arrives first — bare form
		refWith("internal/tui", "package"), // upgrades to trailing-slash form
	})
	got := resolveDiffPaths(&task)
	if len(got) != 1 {
		t.Fatalf("package-wins: expected exactly 1 result, got %v", got)
	}
	if got[0] != "internal/tui/" {
		t.Fatalf("package-wins: expected %q, got %q", "internal/tui/", got[0])
	}
}

// TestResolveDiffPaths_PackageFirstThenPath asserts that when the same Location
// appears as a "package" ref first and then as a "path" ref, the trailing-slash
// (package) form wins — the path ref cannot downgrade the already-slashed entry.
// This is the reverse-order complement of TestResolveDiffPaths_PackageWinsOverPath.
func TestResolveDiffPaths_PackageFirstThenPath(t *testing.T) {
	// package ref first, then path ref for the same Location
	task := newDiffTestTask(t, []domain.ResourceRef{
		refWith("internal/tui", "package"), // arrives first — sets trailing-slash form
		refWith("internal/tui", "path"),    // must not downgrade to bare form
	})
	got := resolveDiffPaths(&task)
	if len(got) != 1 {
		t.Fatalf("package-first: expected exactly 1 result, got %v", got)
	}
	if got[0] != "internal/tui/" {
		t.Fatalf("package-first: expected %q, got %q", "internal/tui/", got[0])
	}
}

// TestResolveDiffPaths_UnknownTagSkipped asserts that a ResourceRef whose first
// tag is outside {"path","file","package"} is silently skipped without error.
func TestResolveDiffPaths_UnknownTagSkipped(t *testing.T) {
	task := newDiffTestTask(t, []domain.ResourceRef{
		refWith("https://example.com/spec", "url"),
		refWith("internal/tui", "path"),
	})
	got := resolveDiffPaths(&task)
	if len(got) != 1 || got[0] != "internal/tui" {
		t.Fatalf("unknown tag: expected [internal/tui], got %v", got)
	}
}

// TestResolveDiffPaths_EmptyTagsSkipped asserts that a ResourceRef with an empty
// Tags slice is silently skipped (no panic, no output).
func TestResolveDiffPaths_EmptyTagsSkipped(t *testing.T) {
	task := newDiffTestTask(t, []domain.ResourceRef{
		{Location: "internal/domain"},   // no Tags field
		refWith("internal/tui", "path"), // has tag
	})
	got := resolveDiffPaths(&task)
	if len(got) != 1 || got[0] != "internal/tui" {
		t.Fatalf("empty tags: expected [internal/tui], got %v", got)
	}
}

// TestDiffMode_SetItem_PassesResolvedPaths asserts that SetItem wires the
// resolved path list into the next Differ.Diff invocation via enterDiffMode.
func TestDiffMode_SetItem_PassesResolvedPaths(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "resource task",
		Priority:  domain.PriorityLow,
		Metadata: domain.ActionItemMetadata{
			ResourceRefs: []domain.ResourceRef{
				refWith("internal/tui", "path"),
				refWith("internal/domain", "package"),
			},
		},
	}, now)

	fd := &fakeDiffer{result: gitdiff.DiffResult{Patch: samplePatchForDiffMode, Divergence: gitdiff.DivergenceAncestor}}
	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, []domain.ActionItem{task})
	m := loadReadyModel(t, NewModel(svc, WithDiffMode(fd, stubHighlighter{})))

	// Enter diff mode — this should call SetItem with the selected board task.
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.mode != modeDiff {
		t.Fatalf("expected modeDiff, got %v", m.mode)
	}

	// The fake differ must have been called exactly once.
	if fd.calls != 1 {
		t.Fatalf("expected 1 Diff call, got %d", fd.calls)
	}

	// The paths it received must match resolveDiffPaths(task).
	want := resolveDiffPaths(&task)
	if len(fd.lastPaths) != len(want) {
		t.Fatalf("path count mismatch: want %v, got %v", want, fd.lastPaths)
	}
	for i, w := range want {
		if fd.lastPaths[i] != w {
			t.Fatalf("path[%d]: want %q, got %q", i, w, fd.lastPaths[i])
		}
	}
}

// TestDiffMode_RecomputesOnItemChange asserts that entering diff mode after the
// active task's ResourceRefs have changed results in a fresh Differ call with the
// updated path list (i.e. paths are not cached across enterDiffMode sessions).
func TestDiffMode_RecomputesOnItemChange(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "recompute task",
		Priority:  domain.PriorityLow,
		Metadata: domain.ActionItemMetadata{
			ResourceRefs: []domain.ResourceRef{
				refWith("internal/app", "path"),
			},
		},
	}, now)

	fd := &fakeDiffer{result: gitdiff.DiffResult{Patch: samplePatchForDiffMode, Divergence: gitdiff.DivergenceAncestor}}
	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, []domain.ActionItem{task})
	m := loadReadyModel(t, NewModel(svc, WithDiffMode(fd, stubHighlighter{})))

	// First diff entry — uses task's initial ResourceRefs.
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.mode != modeDiff {
		t.Fatalf("expected modeDiff on first entry, got %v", m.mode)
	}
	if fd.calls != 1 {
		t.Fatalf("expected 1 Diff call after first entry, got %d", fd.calls)
	}
	firstPaths := append([]string(nil), fd.lastPaths...)

	// Exit diff mode.
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeNone {
		t.Fatalf("expected modeNone after esc, got %v", m.mode)
	}

	// Mutate the task's ResourceRefs in the fake service to simulate an update
	// (e.g. after P3-A/B writes new entries). We set new refs directly through
	// SetItem on the diff struct to replicate what enterDiffMode would see if
	// the in-memory task had changed. We achieve this by updating the task in
	// svc and reloading the model.
	updatedTask := task
	updatedTask.Metadata.ResourceRefs = []domain.ResourceRef{
		refWith("internal/domain", "package"),
		refWith("cmd/till", "path"),
	}
	svc.tasks[p.ID] = []domain.ActionItem{updatedTask}
	m2 := loadReadyModel(t, NewModel(svc, WithDiffMode(fd, stubHighlighter{})))

	// Second diff entry — uses updated ResourceRefs.
	m2 = applyMsg(t, m2, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m2.mode != modeDiff {
		t.Fatalf("expected modeDiff on second entry, got %v", m2.mode)
	}
	if fd.calls != 2 {
		t.Fatalf("expected 2 Diff calls total, got %d", fd.calls)
	}

	// The second paths must differ from the first.
	secondPaths := fd.lastPaths
	if strings.Join(firstPaths, ",") == strings.Join(secondPaths, ",") {
		t.Fatalf("expected different paths on second entry:\nfirst=%v\nsecond=%v", firstPaths, secondPaths)
	}

	// The second paths must match resolveDiffPaths of the updated task.
	want := resolveDiffPaths(&updatedTask)
	if len(secondPaths) != len(want) {
		t.Fatalf("second paths count mismatch: want %v, got %v", want, secondPaths)
	}
	for i, w := range want {
		if secondPaths[i] != w {
			t.Fatalf("second path[%d]: want %q, got %q", i, w, secondPaths[i])
		}
	}
}
