package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// renderThreadModeView renders the full-screen project/work-item thread view.
func (m Model) renderThreadModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	hintStyle := lipgloss.NewStyle().Foreground(muted)
	statusStyle := lipgloss.NewStyle().Foreground(dim)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)

	threadTitle := strings.TrimSpace(m.threadTitle)
	if threadTitle == "" {
		threadTitle = "(untitled thread)"
	}
	header := titleStyle.Render("tillsyn thread") + "  " + threadTitle + statusStyle.Render("  ["+m.modeLabel()+"]")
	targetLine := hintStyle.Render(fmt.Sprintf("target: %s/%s  comments: %d", m.threadTarget.TargetType, m.threadTarget.TargetID, len(m.threadComments)))

	wrapWidth := max(24, m.width-8)
	bodyLines := m.threadBodyLines(wrapWidth, sectionStyle, hintStyle)

	in := m.threadInput
	in.SetWidth(max(20, m.width-18))
	composerLine := "comment: " + in.View()
	hints := hintStyle.Render("enter post • pgup/pgdown scroll • mouse wheel scroll • ctrl+r reload • ? help • esc back")

	statusLine := ""
	statusText := strings.TrimSpace(m.status)
	if statusText != "" && statusText != "ready" {
		statusLine = statusStyle.Render(statusText)
	}

	beforeBody := strings.Join([]string{header, targetLine, ""}, "\n")
	afterParts := []string{"", composerLine, hints}
	if statusLine != "" {
		afterParts = append(afterParts, statusLine)
	}
	afterBody := strings.Join(afterParts, "\n")

	bodyHeight := 12
	if m.height > 0 {
		bodyHeight = m.height - lipgloss.Height(beforeBody) - lipgloss.Height(afterBody)
		if bodyHeight < 6 {
			bodyHeight = 6
		}
	}
	maxScroll := max(0, len(bodyLines)-bodyHeight)
	scrollTop := clamp(m.threadScroll, 0, maxScroll)
	visibleEnd := min(len(bodyLines), scrollTop+bodyHeight)
	visible := append([]string(nil), bodyLines[scrollTop:visibleEnd]...)
	if len(visible) < bodyHeight {
		visible = append(visible, make([]string, bodyHeight-len(visible))...)
	}

	content := strings.Join([]string{beforeBody, strings.Join(visible, "\n"), afterBody}, "\n")
	if m.height > 0 {
		content = fitLines(content, m.height)
	}
	if m.help.ShowAll {
		overlay := m.renderHelpOverlay(accent, muted, dim, hintStyle, m.width-8)
		if overlay != "" {
			overlayHeight := lipgloss.Height(content)
			if m.height > 0 {
				overlayHeight = m.height
			}
			content = overlayOnContent(content, overlay, max(1, m.width), max(1, overlayHeight))
		}
	}

	v := tea.NewView(content)
	v.MouseMode = m.activeMouseMode()
	v.AltScreen = true
	return v
}

// threadBodyLines renders thread description and comments into scrollable body lines.
func (m Model) threadBodyLines(width int, sectionStyle, hintStyle lipgloss.Style) []string {
	lines := []string{sectionStyle.Render("Description")}
	description := strings.TrimSpace(m.threadDescriptionMarkdown)
	if description == "" {
		lines = append(lines, hintStyle.Render("(no description)"))
	} else {
		rendered := m.threadMarkdown.render(description, width)
		lines = append(lines, splitThreadMarkdownLines(rendered)...)
	}

	lines = append(lines, "", sectionStyle.Render(fmt.Sprintf("Comments (%d)", len(m.threadComments))))
	if len(m.threadComments) == 0 {
		lines = append(lines, hintStyle.Render("(no comments yet)"))
		return lines
	}

	for idx, comment := range m.threadComments {
		owner := threadCommentOwnerLabel(comment)
		actor := string(normalizeCommentActorType(string(comment.ActorType)))
		lines = append(lines, hintStyle.Render(fmt.Sprintf("[%s] %s • %s", actor, owner, formatThreadTimestamp(comment.CreatedAt))))

		body := m.threadMarkdown.render(comment.BodyMarkdown, width)
		if strings.TrimSpace(body) == "" {
			body = "(empty comment)"
		}
		for _, line := range splitThreadMarkdownLines(body) {
			lines = append(lines, "  "+line)
		}
		if idx < len(m.threadComments)-1 {
			lines = append(lines, "")
		}
	}
	return lines
}

// splitThreadMarkdownLines splits rendered markdown while preserving empty rows.
func splitThreadMarkdownLines(rendered string) []string {
	if rendered == "" {
		return []string{""}
	}
	return strings.Split(strings.TrimRight(rendered, "\n"), "\n")
}

// startProjectThread opens thread mode for the currently selected project.
func (m Model) startProjectThread(backMode inputMode) (tea.Model, tea.Cmd) {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return m, nil
	}
	target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   project.ID,
	})
	if err != nil {
		m.status = "project thread target invalid: " + err.Error()
		return m, nil
	}
	return m.startThread(backMode, target, project.Name, project.Description)
}

// startSelectedWorkItemThread opens thread mode for the currently selected board item.
func (m Model) startSelectedWorkItemThread(backMode inputMode) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	return m.startTaskThread(task, backMode)
}

// startTaskThread opens thread mode for a specific work item.
func (m Model) startTaskThread(task domain.Task, backMode inputMode) (tea.Model, tea.Cmd) {
	targetType, ok := commentTargetTypeForTask(task)
	if !ok {
		m.status = "unsupported work-item kind for comments"
		return m, nil
	}
	target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  task.ProjectID,
		TargetType: targetType,
		TargetID:   task.ID,
	})
	if err != nil {
		m.status = "work-item thread target invalid: " + err.Error()
		return m, nil
	}
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = task.ID
	}
	title = fmt.Sprintf("%s: %s", task.Kind, title)
	return m.startThread(backMode, target, title, task.Description)
}

// startThread initializes thread-mode state and kicks off comment loading.
func (m Model) startThread(backMode inputMode, target domain.CommentTarget, title, description string) (tea.Model, tea.Cmd) {
	if backMode != modeTaskInfo {
		backMode = modeNone
	}
	m.mode = modeThread
	m.threadBackMode = backMode
	m.threadTarget = target
	m.threadTitle = strings.TrimSpace(title)
	m.threadDescriptionMarkdown = strings.TrimSpace(description)
	m.threadComments = nil
	m.threadScroll = 0
	m.threadPendingCommentBody = ""
	m.threadInput.SetValue("")
	m.threadInput.CursorEnd()
	// Set focused state eagerly so both runtime behavior and unit-test command loops
	// see the composer as active even when only one command is returned.
	m.threadInput.Focus()
	m.status = "loading thread..."
	return m, m.loadThreadCommentsCmd(target)
}

// loadThreadCommentsCmd loads comments for one thread target.
func (m Model) loadThreadCommentsCmd(target domain.CommentTarget) tea.Cmd {
	input := app.ListCommentsByTargetInput{
		ProjectID:  target.ProjectID,
		TargetType: target.TargetType,
		TargetID:   target.TargetID,
	}
	return func() tea.Msg {
		comments, err := m.svc.ListCommentsByTarget(context.Background(), input)
		return threadLoadedMsg{
			target:   target,
			comments: comments,
			err:      err,
		}
	}
}

// createThreadCommentCmd persists one new thread comment with identity defaults.
func (m Model) createThreadCommentCmd(body string) tea.Cmd {
	target := m.threadTarget
	actorID := m.threadActorID()
	actorName := m.threadActorName()
	actorType := m.threadActorType()
	return func() tea.Msg {
		comment, err := m.svc.CreateComment(context.Background(), app.CreateCommentInput{
			ProjectID:    target.ProjectID,
			TargetType:   target.TargetType,
			TargetID:     target.TargetID,
			BodyMarkdown: strings.TrimSpace(body),
			ActorID:      actorID,
			ActorName:    actorName,
			ActorType:    actorType,
		})
		return threadCommentCreatedMsg{
			target: target,
			body:   body,
			value:  comment,
			err:    err,
		}
	}
}

// threadActorID resolves the immutable actor id used for new thread comments.
func (m Model) threadActorID() string {
	actorID := strings.TrimSpace(m.identityActorID)
	if actorID == "" {
		return "tillsyn-user"
	}
	return actorID
}

// threadActorName resolves the default actor name for new thread comments.
func (m Model) threadActorName() string {
	actorName := strings.TrimSpace(m.identityDisplayName)
	if actorName == "" {
		return m.threadActorID()
	}
	return actorName
}

// threadActorType resolves the default actor type for new thread comments.
func (m Model) threadActorType() domain.ActorType {
	return normalizeCommentActorType(m.identityDefaultActorType)
}

// normalizeCommentActorType canonicalizes actor text and applies a safe user fallback.
func normalizeCommentActorType(raw string) domain.ActorType {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case string(domain.ActorTypeAgent):
		return domain.ActorTypeAgent
	case string(domain.ActorTypeSystem):
		return domain.ActorTypeSystem
	case string(domain.ActorTypeUser):
		return domain.ActorTypeUser
	default:
		return domain.ActorTypeUser
	}
}

// threadCommentOwnerLabel renders comment ownership using actor_name with compact actor_id context.
func threadCommentOwnerLabel(comment domain.Comment) string {
	actorName := strings.TrimSpace(comment.ActorName)
	actorID := strings.TrimSpace(comment.ActorID)
	if actorName == "" {
		actorName = actorID
	}
	if actorName == "" {
		return "unknown"
	}
	if actorID == "" || strings.EqualFold(actorName, actorID) {
		return actorName
	}
	return fmt.Sprintf("%s (%s)", actorName, actorID)
}

// commentTargetTypeForTask maps one work item into comment target types with scope-aware overrides.
func commentTargetTypeForTask(task domain.Task) (domain.CommentTargetType, bool) {
	// Subphase items are modeled as phase kind + subphase scope, so scope takes precedence.
	if task.Scope == domain.KindAppliesToSubphase {
		return domain.CommentTargetTypeSubphase, true
	}
	if task.Scope == domain.KindAppliesToBranch {
		return domain.CommentTargetTypeBranch, true
	}
	return commentTargetTypeForWorkKind(task.Kind)
}

// commentTargetTypeForWorkKind maps work-item kinds into comment target types.
func commentTargetTypeForWorkKind(kind domain.WorkKind) (domain.CommentTargetType, bool) {
	switch kind {
	case domain.WorkKind(domain.KindAppliesToBranch):
		return domain.CommentTargetTypeBranch, true
	case domain.WorkKindTask:
		return domain.CommentTargetTypeTask, true
	case domain.WorkKindSubtask:
		return domain.CommentTargetTypeSubtask, true
	case domain.WorkKindPhase:
		return domain.CommentTargetTypePhase, true
	case domain.WorkKind(domain.KindAppliesToSubphase):
		return domain.CommentTargetTypeSubphase, true
	case domain.WorkKindDecision:
		return domain.CommentTargetTypeDecision, true
	case domain.WorkKindNote:
		return domain.CommentTargetTypeNote, true
	default:
		return "", false
	}
}

// formatThreadTimestamp formats comment timestamps for thread metadata rows.
func formatThreadTimestamp(at time.Time) string {
	if at.IsZero() {
		return "-"
	}
	return at.Local().Format("2006-01-02 15:04")
}

// sameCommentTarget reports whether two thread targets refer to the same entity.
func sameCommentTarget(a, b domain.CommentTarget) bool {
	if strings.TrimSpace(a.ProjectID) != strings.TrimSpace(b.ProjectID) {
		return false
	}
	if strings.TrimSpace(string(a.TargetType)) != strings.TrimSpace(string(b.TargetType)) {
		return false
	}
	if strings.TrimSpace(a.TargetID) != strings.TrimSpace(b.TargetID) {
		return false
	}
	return true
}

// threadViewportStep returns one paging increment for thread scroll updates.
func (m Model) threadViewportStep() int {
	if m.height <= 0 {
		return 6
	}
	return max(3, m.height/3)
}
