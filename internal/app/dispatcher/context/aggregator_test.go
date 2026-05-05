package context

import (
	stdcontext "context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// mockReader is a deterministic in-memory ActionItemReader for tests. The
// itemsByID map is the canonical store; childrenByParent is derived index
// the test populates explicitly so different scenarios can simulate stale or
// missing edges.
type mockReader struct {
	itemsByID        map[string]domain.ActionItem
	childrenByParent map[string][]domain.ActionItem
	// getDelay simulates slow lookups for timeout tests. When non-zero the
	// mock sleeps `getDelay` before each GetActionItem return.
	getDelay time.Duration
	// getErr forces every lookup to return an error. Highest precedence.
	getErr error
}

func newMockReader() *mockReader {
	return &mockReader{
		itemsByID:        map[string]domain.ActionItem{},
		childrenByParent: map[string][]domain.ActionItem{},
	}
}

func (m *mockReader) add(item domain.ActionItem) {
	m.itemsByID[item.ID] = item
	if item.ParentID != "" {
		m.childrenByParent[item.ParentID] = append(m.childrenByParent[item.ParentID], item)
	}
}

func (m *mockReader) GetActionItem(ctx stdcontext.Context, id string) (domain.ActionItem, error) {
	if m.getDelay > 0 {
		select {
		case <-time.After(m.getDelay):
		case <-ctx.Done():
			return domain.ActionItem{}, ctx.Err()
		}
	}
	if m.getErr != nil {
		return domain.ActionItem{}, m.getErr
	}
	item, ok := m.itemsByID[id]
	if !ok {
		return domain.ActionItem{}, errors.New("not found: " + id)
	}
	return item, nil
}

func (m *mockReader) ListChildren(ctx stdcontext.Context, parentID string) ([]domain.ActionItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.childrenByParent[parentID], nil
}

func (m *mockReader) ListSiblings(ctx stdcontext.Context, parentID string) ([]domain.ActionItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.childrenByParent[parentID], nil
}

// mockDiff is a deterministic GitDiffReader. The diffOut is returned for any
// call; diffDelay simulates slow diff computation.
type mockDiff struct {
	diffOut   []byte
	diffErr   error
	diffDelay time.Duration
}

func (m *mockDiff) Diff(ctx stdcontext.Context, from, to string) ([]byte, error) {
	if m.diffDelay > 0 {
		select {
		case <-time.After(m.diffDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.diffErr != nil {
		return nil, m.diffErr
	}
	return m.diffOut, nil
}

// mustItem builds a minimal domain.ActionItem for tests. Only the fields the
// aggregator reads are populated; everything else is zero-valued. We do NOT
// go through domain.NewActionItem because tests want to exercise unusual
// shapes (cycles, missing parents) that NewActionItem would reject.
func mustItem(id, parentID string, kind domain.Kind, title string, createdAt time.Time) domain.ActionItem {
	return domain.ActionItem{
		ID:        id,
		ParentID:  parentID,
		Kind:      kind,
		Title:     title,
		CreatedAt: createdAt,
	}
}

// TestResolveEmptyBindingAgenticMode verifies the fast-path: a zero-value
// ContextRules returns an empty Bundle with no reader interactions. This is
// the agentic-mode contract from master PLAN.md L13.
func TestResolveEmptyBindingAgenticMode(t *testing.T) {
	t.Parallel()

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding: templates.AgentBinding{},
		Item:    mustItem("a", "", "build", "A", time.Now()),
		// Reader + DiffReader deliberately nil — empty binding must NOT
		// touch them.
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if bundle.RenderedInline != "" {
		t.Errorf("RenderedInline = %q, want empty", bundle.RenderedInline)
	}
	if len(bundle.Files) != 0 {
		t.Errorf("Files = %v, want empty", bundle.Files)
	}
	if len(bundle.Markers) != 0 {
		t.Errorf("Markers = %v, want empty", bundle.Markers)
	}
}

// TestResolveHappyPathAllRulesActive verifies a fully-bound binding renders
// every rule in declaration order with the expected content surfaces.
func TestResolveHappyPathAllRulesActive(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	parent := mustItem("p1", "root", "plan", "PARENT PLAN", now)
	parent.StartCommit = "abc123"
	parent.EndCommit = "def456"
	parent.Description = "Parent description text."

	root := mustItem("root", "", "plan", "ROOT", now.Add(-time.Hour))
	root.Description = "Root description."

	siblingQA := mustItem("s1", "p1", "build-qa-proof", "SIB QA", now.Add(time.Minute))
	target := mustItem("t1", "p1", "build", "TARGET", now)

	descChild := mustItem("d1", "t1", "research", "DESC CHILD", now)

	r := newMockReader()
	r.add(root)
	r.add(parent)
	r.add(target)
	r.add(siblingQA)
	r.add(descChild)

	diff := &mockDiff{diffOut: []byte("DIFF CONTENT")}

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent:            true,
			ParentGitDiff:     true,
			SiblingsByKind:    []domain.Kind{"build-qa-proof"},
			AncestorsByKind:   []domain.Kind{"plan"},
			DescendantsByKind: []domain.Kind{"research"},
			Delivery:          templates.ContextDeliveryFile,
		},
	}

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding:    binding,
		Item:       target,
		Reader:     r,
		DiffReader: diff,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// File-mode: each enabled rule should land in Files["<rule>.md"].
	for _, name := range []string{"parent", "parent_git_diff", "siblings_by_kind", "ancestors_by_kind", "descendants_by_kind"} {
		if _, ok := bundle.Files[name+".md"]; !ok {
			t.Errorf("missing Files[%s.md]; have %v", name, fileKeys(bundle.Files))
		}
	}

	// Parent block must contain the parent's title.
	if !strings.Contains(string(bundle.Files["parent.md"]), "PARENT PLAN") {
		t.Errorf("parent.md missing parent title: %s", bundle.Files["parent.md"])
	}
	// Parent git diff must contain DIFF CONTENT verbatim.
	if string(bundle.Files["parent_git_diff.md"]) != "DIFF CONTENT" {
		t.Errorf("parent_git_diff.md = %q, want DIFF CONTENT", bundle.Files["parent_git_diff.md"])
	}
	// Siblings rule renders the sibling QA item.
	if !strings.Contains(string(bundle.Files["siblings_by_kind.md"]), "SIB QA") {
		t.Errorf("siblings_by_kind.md missing sibling title: %s", bundle.Files["siblings_by_kind.md"])
	}
	// Ancestor walk: the parent IS a plan, so it matches first; ancestor walk
	// should NOT walk past the parent.
	if !strings.Contains(string(bundle.Files["ancestors_by_kind.md"]), "PARENT PLAN") {
		t.Errorf("ancestors_by_kind.md missing parent title: %s", bundle.Files["ancestors_by_kind.md"])
	}
	// Descendants walk renders the research child.
	if !strings.Contains(string(bundle.Files["descendants_by_kind.md"]), "DESC CHILD") {
		t.Errorf("descendants_by_kind.md missing descendant title: %s", bundle.Files["descendants_by_kind.md"])
	}

	// File-mode happy path emits no markers.
	if len(bundle.Markers) != 0 {
		t.Errorf("expected no markers, got %v", bundle.Markers)
	}
	if bundle.RenderedInline != "" {
		t.Errorf("file-mode RenderedInline should be empty, got %q", bundle.RenderedInline)
	}
}

// TestResolvePerRuleTruncation verifies a rule whose output exceeds the
// per-rule MaxChars cap is truncated, a marker is emitted, and the full
// content lands in Files["<rule>.full"].
func TestResolvePerRuleTruncation(t *testing.T) {
	t.Parallel()

	now := time.Now()
	parent := mustItem("p", "", "plan", "P", now)
	parent.Description = strings.Repeat("X", 1000)
	target := mustItem("t", "p", "build", "T", now)

	r := newMockReader()
	r.add(parent)
	r.add(target)

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent:   true,
			MaxChars: 100,
			Delivery: templates.ContextDeliveryFile,
		},
	}

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding: binding,
		Item:    target,
		Reader:  r,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Marker present.
	if len(bundle.Markers) == 0 || !strings.Contains(bundle.Markers[0], "truncated to 100") {
		t.Errorf("expected truncation marker, got %v", bundle.Markers)
	}

	// Truncated content in Files["parent.md"].
	if got := bundle.Files["parent.md"]; len(got) != 100 {
		t.Errorf("parent.md len = %d, want 100", len(got))
	}

	// Full content in Files["parent.full"].
	if got := bundle.Files["parent.full"]; len(got) <= 100 {
		t.Errorf("parent.full len = %d, want > 100", len(got))
	}
}

// TestResolveGreedyFitCap verifies the greedy-fit semantics from master
// PLAN.md L14: rule 1 fits, rule 2 busts the bundle cap (skipped with
// marker), rule 3 fits and lands. Greedy-fit is NOT serial-drop.
func TestResolveGreedyFitCap(t *testing.T) {
	t.Parallel()

	now := time.Now()
	// Parent has a small description (fits in cap).
	parent := mustItem("p", "", "plan", "P", now)
	parent.Description = "small parent"
	// Target action item.
	target := mustItem("t", "p", "build", "T", now)
	// Sibling that produces oversized content (will bust the cap).
	bigSib := mustItem("sib", "p", "build-qa-proof", "BIGSIB", now)
	bigSib.Description = strings.Repeat("Y", 5000)
	// Descendant that is small (will fit even after sibling skipped).
	desc := mustItem("d", "t", "research", "DESC", now)
	desc.Description = "small desc"

	r := newMockReader()
	r.add(parent)
	r.add(target)
	r.add(bigSib)
	r.add(desc)

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent:            true,
			SiblingsByKind:    []domain.Kind{"build-qa-proof"},
			DescendantsByKind: []domain.Kind{"research"},
			Delivery:          templates.ContextDeliveryFile,
			// Per-rule cap large enough to NOT truncate; bundle cap is the
			// gating factor.
			MaxChars: 10000,
		},
	}

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding:       binding,
		Item:          target,
		Reader:        r,
		BundleCharCap: 1000, // small enough that the big sibling busts it
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Rule 1 (parent) lands.
	if _, ok := bundle.Files["parent.md"]; !ok {
		t.Errorf("expected parent.md to land")
	}
	// Rule 2 (siblings) skipped — should produce a skip marker.
	hasSkipMarker := false
	for _, m := range bundle.Markers {
		if strings.Contains(m, "skipped: siblings_by_kind") {
			hasSkipMarker = true
		}
	}
	if !hasSkipMarker {
		t.Errorf("expected siblings_by_kind skip marker, got markers %v", bundle.Markers)
	}
	if _, ok := bundle.Files["siblings_by_kind.md"]; ok {
		t.Errorf("siblings_by_kind.md should be skipped, but was rendered")
	}
	// Rule 3 (descendants) STILL fits — greedy-fit, not serial-drop.
	if _, ok := bundle.Files["descendants_by_kind.md"]; !ok {
		t.Errorf("expected descendants_by_kind.md to land after sibling skip")
	}
}

// TestResolvePerRuleTimeout verifies a slow rule (parent_git_diff sleeping
// past MaxRuleDuration) emits a timeout marker, and subsequent rules still
// run with the remaining outer budget.
func TestResolvePerRuleTimeout(t *testing.T) {
	t.Parallel()

	now := time.Now()
	parent := mustItem("p", "", "plan", "P", now)
	parent.StartCommit = "a"
	parent.EndCommit = "b"
	parent.Description = "parent text"
	target := mustItem("t", "p", "build", "T", now)
	siblingQA := mustItem("s", "p", "build-qa-proof", "SIB", now)

	r := newMockReader()
	r.add(parent)
	r.add(target)
	r.add(siblingQA)

	// Diff sleeps 100ms but per-rule cap is 10ms.
	diff := &mockDiff{
		diffOut:   []byte("never"),
		diffDelay: 100 * time.Millisecond,
	}

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent:          true,
			ParentGitDiff:   true,
			SiblingsByKind:  []domain.Kind{"build-qa-proof"},
			Delivery:        templates.ContextDeliveryFile,
			MaxRuleDuration: templates.Duration(10 * time.Millisecond),
		},
	}

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding:        binding,
		Item:           target,
		Reader:         r,
		DiffReader:     diff,
		BundleDuration: 5 * time.Second, // ample outer budget
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Parent fired and landed.
	if _, ok := bundle.Files["parent.md"]; !ok {
		t.Errorf("expected parent.md to land")
	}
	// parent_git_diff timed out.
	hasTimeout := false
	for _, m := range bundle.Markers {
		if strings.Contains(m, "rule parent_git_diff timed out") {
			hasTimeout = true
		}
	}
	if !hasTimeout {
		t.Errorf("expected parent_git_diff timeout marker, got %v", bundle.Markers)
	}
	if _, ok := bundle.Files["parent_git_diff.md"]; ok {
		t.Errorf("parent_git_diff.md should not have rendered after timeout")
	}
	// siblings_by_kind STILL fired after parent_git_diff timed out.
	if _, ok := bundle.Files["siblings_by_kind.md"]; !ok {
		t.Errorf("expected siblings_by_kind.md to land after parent_git_diff timeout")
	}
}

// TestResolvePerBundleTimeout verifies that when the outer wall-clock budget
// expires before all rules finish, the engine emits an aggregator-timeout
// marker listing pending rules and returns the partial bundle.
func TestResolvePerBundleTimeout(t *testing.T) {
	t.Parallel()

	now := time.Now()
	parent := mustItem("p", "", "plan", "P", now)
	parent.StartCommit = "a"
	parent.EndCommit = "b"
	parent.Description = "parent text"
	target := mustItem("t", "p", "build", "T", now)
	siblingQA := mustItem("s", "p", "build-qa-proof", "SIB", now)

	r := newMockReader()
	r.add(parent)
	r.add(target)
	r.add(siblingQA)

	// Diff sleeps 200ms; per-rule cap is 1s but bundle cap is 50ms — outer
	// wins.
	diff := &mockDiff{
		diffOut:   []byte("nope"),
		diffDelay: 200 * time.Millisecond,
	}

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent:          true,
			ParentGitDiff:   true,
			SiblingsByKind:  []domain.Kind{"build-qa-proof"},
			Delivery:        templates.ContextDeliveryFile,
			MaxRuleDuration: templates.Duration(time.Second),
		},
	}

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding:        binding,
		Item:           target,
		Reader:         r,
		DiffReader:     diff,
		BundleDuration: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	// Aggregator-timeout marker present and lists pending rules.
	hasOuter := false
	for _, m := range bundle.Markers {
		if strings.Contains(m, "aggregator timed out") {
			hasOuter = true
		}
	}
	if !hasOuter {
		t.Errorf("expected aggregator timeout marker, got %v", bundle.Markers)
	}
	// Parent rule completed (it ran first, no delay). Subsequent rules timed
	// out via the outer.
	if _, ok := bundle.Files["parent.md"]; !ok {
		t.Errorf("expected parent.md to have landed before outer timeout")
	}
}

// TestResolveDefaultSubstitution verifies zero-valued ResolveArgs caps +
// rule durations pick up engine-time defaults from the constants.
func TestResolveDefaultSubstitution(t *testing.T) {
	t.Parallel()

	now := time.Now()
	parent := mustItem("p", "", "plan", "P", now)
	parent.Description = "ok"
	target := mustItem("t", "p", "build", "T", now)

	r := newMockReader()
	r.add(parent)
	r.add(target)

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent:   true,
			Delivery: templates.ContextDeliveryFile,
			// MaxChars + MaxRuleDuration deliberately zero — engine picks
			// 50000 / 500ms defaults.
		},
	}

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding: binding,
		Item:    target,
		Reader:  r,
		// BundleCharCap + BundleDuration deliberately zero — engine picks
		// 200000 / 2s defaults.
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, ok := bundle.Files["parent.md"]; !ok {
		t.Errorf("expected parent.md with default caps")
	}

	// Sanity: the constants are what the spec requires.
	if defaultBundleCharCap != 200_000 {
		t.Errorf("defaultBundleCharCap = %d, want 200000", defaultBundleCharCap)
	}
	if defaultRuleCharCap != 50_000 {
		t.Errorf("defaultRuleCharCap = %d, want 50000", defaultRuleCharCap)
	}
	if defaultBundleDuration != 2*time.Second {
		t.Errorf("defaultBundleDuration = %s, want 2s", defaultBundleDuration)
	}
	if defaultRuleDuration != 500*time.Millisecond {
		t.Errorf("defaultRuleDuration = %s, want 500ms", defaultRuleDuration)
	}
}

// TestResolveDeclarationOrderStable verifies rules execute in struct-
// declaration order regardless of the order in which the binding's slices
// are arranged. The slice order test isn't directly observable for a single
// rule, but we can verify the marker order in a multi-skip scenario where
// the order of skip markers reflects the rule iteration order.
func TestResolveDeclarationOrderStable(t *testing.T) {
	t.Parallel()

	now := time.Now()
	parent := mustItem("p", "", "plan", "P", now)
	parent.StartCommit = "a"
	parent.EndCommit = "b"
	parent.Description = strings.Repeat("X", 500)
	target := mustItem("t", "p", "build", "T", now)
	siblingQA := mustItem("s", "p", "build-qa-proof", "SIB", now)
	siblingQA.Description = strings.Repeat("Y", 500)
	desc := mustItem("d", "t", "research", "D", now)
	desc.Description = strings.Repeat("Z", 500)

	r := newMockReader()
	r.add(parent)
	r.add(target)
	r.add(siblingQA)
	r.add(desc)

	diff := &mockDiff{diffOut: []byte(strings.Repeat("D", 500))}

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent:            true,
			ParentGitDiff:     true,
			SiblingsByKind:    []domain.Kind{"build-qa-proof"},
			AncestorsByKind:   []domain.Kind{"plan"},
			DescendantsByKind: []domain.Kind{"research"},
			Delivery:          templates.ContextDeliveryFile,
			MaxChars:          100, // every rule will truncate
		},
	}

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding:    binding,
		Item:       target,
		Reader:     r,
		DiffReader: diff,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Markers should appear in the canonical declaration order. Build the
	// expected sequence from allRuleNames and assert the markers appear in
	// that order. The ancestor rule walks UP from item.ParentID; the parent
	// IS the plan ancestor so it matches and renders. Description >100 chars
	// → truncation marker.
	want := []string{"parent", "parent_git_diff", "siblings_by_kind", "ancestors_by_kind", "descendants_by_kind"}
	idx := 0
	for _, m := range bundle.Markers {
		if idx >= len(want) {
			break
		}
		if strings.Contains(m, want[idx]) {
			idx++
		}
	}
	if idx != len(want) {
		t.Errorf("markers did not appear in declaration order; markers=%v want order=%v", bundle.Markers, want)
	}
}

// TestResolveInlineDelivery verifies inline mode concatenates rule content
// into RenderedInline rather than landing in Files (markers also append).
func TestResolveInlineDelivery(t *testing.T) {
	t.Parallel()

	now := time.Now()
	parent := mustItem("p", "", "plan", "P", now)
	parent.Description = "parent body"
	target := mustItem("t", "p", "build", "T", now)

	r := newMockReader()
	r.add(parent)
	r.add(target)

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent:   true,
			Delivery: templates.ContextDeliveryInline,
		},
	}

	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding: binding,
		Item:    target,
		Reader:  r,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.Contains(bundle.RenderedInline, "parent body") {
		t.Errorf("RenderedInline missing parent body: %q", bundle.RenderedInline)
	}
	// File mode wasn't selected, so parent.md should NOT be in Files.
	if _, ok := bundle.Files["parent.md"]; ok {
		t.Errorf("inline-mode should not write parent.md, got %v", bundle.Files)
	}
}

// TestResolveAncestorsByKindHaltOnFirstMatch verifies the walk halts on the
// first ancestor whose Kind matches — does NOT continue up to find later
// matches.
func TestResolveAncestorsByKindHaltOnFirstMatch(t *testing.T) {
	t.Parallel()

	now := time.Now()
	greatGrand := mustItem("gg", "", "plan", "GREATGRAND", now)
	grand := mustItem("g", "gg", "plan", "GRAND", now)
	parent := mustItem("p", "g", "plan", "PARENT", now)
	target := mustItem("t", "p", "build", "T", now)

	r := newMockReader()
	r.add(greatGrand)
	r.add(grand)
	r.add(parent)
	r.add(target)

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			AncestorsByKind: []domain.Kind{"plan"},
			Delivery:        templates.ContextDeliveryFile,
		},
	}
	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding: binding,
		Item:    target,
		Reader:  r,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got := string(bundle.Files["ancestors_by_kind.md"])
	if !strings.Contains(got, "PARENT") {
		t.Errorf("ancestors_by_kind.md should contain PARENT (first plan ancestor), got %q", got)
	}
	if strings.Contains(got, "GRAND") || strings.Contains(got, "GREATGRAND") {
		t.Errorf("ancestors walk did not halt on first match; got %q", got)
	}
}

// TestResolveSiblingsByKindLatestRoundOnly verifies multiple siblings
// sharing a kind collapse to the most-recent one by CreatedAt.
func TestResolveSiblingsByKindLatestRoundOnly(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	parent := mustItem("p", "", "plan", "P", t0)
	target := mustItem("t", "p", "build", "T", t0)
	old := mustItem("old", "p", "build-qa-proof", "OLDQA", t0)
	new := mustItem("new", "p", "build-qa-proof", "NEWQA", t0.Add(time.Hour))

	r := newMockReader()
	r.add(parent)
	r.add(target)
	r.add(old)
	r.add(new)

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			SiblingsByKind: []domain.Kind{"build-qa-proof"},
			Delivery:       templates.ContextDeliveryFile,
		},
	}
	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding: binding,
		Item:    target,
		Reader:  r,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got := string(bundle.Files["siblings_by_kind.md"])
	if !strings.Contains(got, "NEWQA") {
		t.Errorf("expected NEWQA in siblings, got %q", got)
	}
	if strings.Contains(got, "OLDQA") {
		t.Errorf("did not expect OLDQA (older round), got %q", got)
	}
}

// TestResolveParentGitDiffEmptyCommitsClean verifies parent missing
// start_commit / end_commit produces empty content, not an error.
func TestResolveParentGitDiffEmptyCommitsClean(t *testing.T) {
	t.Parallel()

	now := time.Now()
	parent := mustItem("p", "", "plan", "P", now) // no commits
	target := mustItem("t", "p", "build", "T", now)

	r := newMockReader()
	r.add(parent)
	r.add(target)

	diff := &mockDiff{diffOut: []byte("never reached")}
	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			ParentGitDiff: true,
			Delivery:      templates.ContextDeliveryFile,
		},
	}
	bundle, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding:    binding,
		Item:       target,
		Reader:     r,
		DiffReader: diff,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// No file emitted because content was empty. No marker either — empty
	// content is a legitimate "nothing to render" result.
	if _, ok := bundle.Files["parent_git_diff.md"]; ok {
		t.Errorf("expected no parent_git_diff.md when commits absent, got %s",
			bundle.Files["parent_git_diff.md"])
	}
}

// TestResolveNilReaderRejected verifies a binding that requires the reader
// returns ErrNilReader rather than panicking.
func TestResolveNilReaderRejected(t *testing.T) {
	t.Parallel()

	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			Parent: true,
		},
	}
	_, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding: binding,
		Item:    mustItem("t", "p", "build", "T", time.Now()),
		Reader:  nil,
	})
	if !errors.Is(err, ErrNilReader) {
		t.Errorf("expected ErrNilReader, got %v", err)
	}
}

// TestResolveNilDiffReaderRejected verifies a binding that requires the diff
// reader returns ErrNilDiffReader.
func TestResolveNilDiffReaderRejected(t *testing.T) {
	t.Parallel()

	r := newMockReader()
	r.add(mustItem("p", "", "plan", "P", time.Now()))
	binding := templates.AgentBinding{
		Context: templates.ContextRules{
			ParentGitDiff: true,
		},
	}
	_, err := Resolve(stdcontext.Background(), ResolveArgs{
		Binding:    binding,
		Item:       mustItem("t", "p", "build", "T", time.Now()),
		Reader:     r,
		DiffReader: nil,
	})
	if !errors.Is(err, ErrNilDiffReader) {
		t.Errorf("expected ErrNilDiffReader, got %v", err)
	}
}

// fileKeys returns the sorted keys of a Files map for diagnostic messages.
func fileKeys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
