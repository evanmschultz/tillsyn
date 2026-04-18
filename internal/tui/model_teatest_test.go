package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestModelWithTeatest verifies behavior for the covered scenario.
func TestModelWithTeatest(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	actionItem, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "First actionItem",
		Priority:  domain.PriorityLow,
	}, now)

	m := NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.ActionItem{actionItem},
	))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 35))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "First actionItem")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestModelWithTeatestHelpAndProjectPicker verifies behavior for the covered scenario.
func TestModelWithTeatestHelpAndProjectPicker(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Side", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)

	m := NewModel(newFakeService(
		[]domain.Project{p1, p2},
		[]domain.Column{c1, c2},
		nil,
	))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 35))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Inbox")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: '?', Text: "?"})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "hard delete")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'p', Text: "p"})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Projects")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Side")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestModelGoldenBoardOutput verifies behavior for the covered scenario.
func TestModelGoldenBoardOutput(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	actionItem, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "Golden board actionItem",
		Description: "golden description",
		Priority:    domain.PriorityMedium,
		Labels:      []string{"alpha", "beta"},
	}, now)

	m := NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.ActionItem{actionItem},
	))
	ready := loadReadyModel(t, m)
	ready = applyMsg(t, ready, tea.WindowSizeMsg{Width: 96, Height: 28})
	rendered := strings.TrimSpace(stripANSI(fmt.Sprint(ready.View().Content)))
	teatest.RequireEqualOutput(t, []byte(rendered+"\n"))
}

// TestModelGoldenHelpExpandedOutput verifies behavior for the covered scenario.
func TestModelGoldenHelpExpandedOutput(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	actionItem, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Help Golden ActionItem",
		Priority:  domain.PriorityLow,
	}, now)

	m := NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.ActionItem{actionItem},
	))
	ready := loadReadyModel(t, m)
	ready = applyMsg(t, ready, tea.WindowSizeMsg{Width: 96, Height: 28})
	ready = applyMsg(t, ready, tea.KeyPressMsg{Code: '?', Text: "?"})
	rendered := strings.TrimSpace(stripANSI(fmt.Sprint(ready.View().Content)))
	teatest.RequireEqualOutput(t, []byte(rendered+"\n"))
}

// TestModelGoldenEmbeddingsStatusOutput verifies the embeddings inventory modal renders mixed subject families.
func TestModelGoldenEmbeddingsStatusOutput(t *testing.T) {
	now := time.Date(2026, 3, 29, 20, 45, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	actionItem, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Embeddings status actionItem",
		Kind:      domain.WorkKind("branch"),
		Scope:     domain.KindAppliesToBranch,
		Priority:  domain.PriorityLow,
	}, now)
	child, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		ParentID:  actionItem.ID,
		Position:  1,
		Title:     "Embeddings status thread actionItem",
		Priority:  domain.PriorityLow,
	}, now)
	threadSubjectID := app.BuildThreadContextSubjectID(domain.CommentTarget{
		ProjectID:  p.ID,
		TargetType: domain.CommentTargetTypeActionItem,
		TargetID:   child.ID,
	})

	svc := newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.ActionItem{actionItem, child},
	)
	svc.embeddingRows = []app.EmbeddingRecord{
		{
			SubjectType: app.EmbeddingSubjectTypeWorkItem,
			SubjectID:   actionItem.ID,
			ProjectID:   p.ID,
			Status:      app.EmbeddingLifecycleReady,
		},
		{
			SubjectType: app.EmbeddingSubjectTypeThreadContext,
			SubjectID:   threadSubjectID,
			ProjectID:   p.ID,
			Status:      app.EmbeddingLifecyclePending,
		},
		{
			SubjectType:      app.EmbeddingSubjectTypeProjectDocument,
			SubjectID:        p.ID,
			ProjectID:        p.ID,
			Status:           app.EmbeddingLifecycleFailed,
			LastErrorSummary: "provider unavailable",
		},
	}
	m := loadReadyModel(t, NewModel(svc))
	m = applyCmd(t, m, m.startEmbeddingsStatus(false))
	rendered := strings.TrimSpace(stripANSI(fmt.Sprint(m.View().Content)))
	teatest.RequireEqualOutput(t, []byte(rendered+"\n"))
}

// TestModelGoldenSearchResultsEmptyOutput verifies zero-result searches stay in the explicit results overlay.
func TestModelGoldenSearchResultsEmptyOutput(t *testing.T) {
	now := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p-search-empty", "Search Empty", "", now)
	c1, _ := domain.NewColumn("c-search-empty", p.ID, "To Do", 0, 0, now)
	actionItem, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t-search-empty",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Real search anchor",
		Priority:  domain.PriorityLow,
	}, now)

	m := loadReadyModel(t, NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.ActionItem{actionItem},
	)))
	m = applyMsg(t, m, tea.WindowSizeMsg{Width: 96, Height: 28})
	m = applyMsg(t, m, keyRune('/'))
	for _, r := range "help" {
		m = applyMsg(t, m, keyRune(r))
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	rendered := strings.TrimSpace(stripANSI(fmt.Sprint(m.View().Content)))
	teatest.RequireEqualOutput(t, []byte(rendered+"\n"))
}

// TestModelWithTeatestWIPWarning verifies behavior for the covered scenario.
func TestModelWithTeatestWIPWarning(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 1, now)
	t1, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "First",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewActionItem(domain.ActionItemInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Second",
		Priority:  domain.PriorityMedium,
	}, now)

	m := NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.ActionItem{t1, t2},
	), WithBoardConfig(BoardConfig{
		ShowWIPWarnings: true,
		GroupBy:         "none",
	}))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(140, 35))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "WIP limit exceeded")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
