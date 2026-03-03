package tui

import (
	"context"
	"fmt"
	"image/color"
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
	sectionTitleStyle := threadSectionStyle(accent)

	if m.threadDetailsEditorActive {
		return m.renderThreadDescriptionEditorView(accent, muted, dim, titleStyle, hintStyle, statusStyle, sectionTitleStyle)
	}

	threadTitle := strings.TrimSpace(m.threadTitle)
	if threadTitle == "" {
		threadTitle = "(untitled thread)"
	}
	header := titleStyle.Render("tillsyn thread") + "  " + threadTitle + statusStyle.Render("  ["+m.modeLabel()+"]")
	targetLine := hintStyle.Render(fmt.Sprintf("target: %s/%s  comments: %d", m.threadTarget.TargetType, m.threadTarget.TargetID, len(m.threadComments)))

	footerHint := "read mode • e details focus • i compose comment • pgup/pgdown and mouse wheel scroll comments • ctrl+r reload • ? help • esc back"
	if m.threadComposerActive {
		footerHint = "composer active • ctrl+s post • enter newline • tab/esc read mode • ? help"
	}
	if m.threadDetailsActive {
		footerHint = "details focused • enter edit target • i open description editor • esc read mode • ? help"
	}

	layoutWidth := max(72, m.width-2)
	if m.width <= 0 {
		layoutWidth = 120
	}
	sidebarWidth := clamp(max(28, layoutWidth/3), 28, 44)
	if layoutWidth-sidebarWidth < 52 {
		sidebarWidth = max(24, layoutWidth-52)
	}
	leftWidth := max(48, layoutWidth-sidebarWidth-1)

	statusLine := ""
	if statusText := strings.TrimSpace(m.status); statusText != "" && statusText != "ready" {
		statusLine = statusStyle.Render(statusText)
	}
	headerBlock := strings.Join([]string{header, targetLine, ""}, "\n")
	footerBlock := strings.Join([]string{"", hintStyle.Render(footerHint)}, "\n")
	if statusLine != "" {
		footerBlock += "\n" + statusLine
	}
	workspaceHeight := 18
	if m.height > 0 {
		workspaceHeight = m.height - lipgloss.Height(headerBlock) - lipgloss.Height(footerBlock)
	}
	if workspaceHeight < 12 {
		workspaceHeight = 12
	}

	commentsHeight := max(8, workspaceHeight/4)
	descriptionHeight := max(8, workspaceHeight-commentsHeight-1)
	workspaceHeight = descriptionHeight + commentsHeight + 1

	descriptionPanel := m.renderThreadDescriptionPanel(accent, muted, dim, sectionTitleStyle, hintStyle, leftWidth, descriptionHeight)
	commentsPanel := m.renderThreadCommentsPanel(accent, muted, dim, sectionTitleStyle, hintStyle, leftWidth, commentsHeight)
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Top,
		descriptionPanel,
		lipgloss.NewStyle().MarginTop(1).Render(commentsPanel),
	)
	rightPanel := m.renderThreadContextPanel(accent, muted, dim, sectionTitleStyle, hintStyle, sidebarWidth, workspaceHeight)
	workspace := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		lipgloss.NewStyle().MarginLeft(1).Render(rightPanel),
	)

	content := strings.Join([]string{headerBlock, workspace, footerBlock}, "\n")
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

// renderThreadDescriptionPanel renders the top description/details pane for thread mode.
func (m Model) renderThreadDescriptionPanel(accent, muted, dim color.Color, sectionTitleStyle, hintStyle lipgloss.Style, width, height int) string {
	title := "Task Description"
	if m.threadTarget.TargetType == domain.CommentTargetTypeProject {
		title = "Project Description"
	}
	if m.threadDetailsActive {
		title = strings.Replace(title, "Description", "Details", 1)
	}
	contentWidth := max(20, width-4)
	contentHeight := max(4, height-2)
	lines := []string{sectionTitleStyle.Render(truncate(title, contentWidth))}
	description := strings.TrimSpace(m.threadDescriptionMarkdown)
	if description == "" {
		lines = append(lines, hintStyle.Render("(no description)"))
	} else {
		lines = append(lines, splitThreadMarkdownLines(m.threadMarkdown.render(description, contentWidth))...)
	}
	lines = append(lines, "", hintStyle.Render("e focus details • enter edit target • i markdown editor"))

	borderColor := dim
	if m.threadDetailsActive {
		borderColor = accent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Render(fitLines(strings.Join(lines, "\n"), contentHeight))
}

// renderThreadCommentsPanel renders the lower comments list pane with a compact 2-line composer.
func (m Model) renderThreadCommentsPanel(accent, muted, dim color.Color, sectionTitleStyle, hintStyle lipgloss.Style, width, height int) string {
	contentWidth := max(20, width-4)
	contentHeight := max(4, height-2)
	lines := []string{sectionTitleStyle.Render(truncate(fmt.Sprintf("Comments (%d)", len(m.threadComments)), contentWidth))}
	commentLines := m.threadCommentListLines(contentWidth, hintStyle)

	composer := m.threadInput
	composer.ShowLineNumbers = false
	composer.SetHeight(2)
	composer.SetWidth(max(20, contentWidth))
	if m.threadComposerActive {
		_ = composer.Focus()
	} else {
		composer.Blur()
	}
	composerBlock := []string{
		"",
		sectionTitleStyle.Render("New Comment"),
		composer.View(),
		hintStyle.Render("i focus composer • ctrl+s post • enter newline"),
	}

	commentListHeight := max(1, contentHeight-len(composerBlock)-1)
	scrollTop := clamp(m.threadScroll, 0, max(0, len(commentLines)-commentListHeight))
	visibleEnd := min(len(commentLines), scrollTop+commentListHeight)
	visible := append([]string(nil), commentLines[scrollTop:visibleEnd]...)
	if len(visible) < commentListHeight {
		visible = append(visible, make([]string, commentListHeight-len(visible))...)
	}

	lines = append(lines, visible...)
	lines = append(lines, composerBlock...)

	borderColor := dim
	if m.threadComposerActive {
		borderColor = accent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Render(fitLines(strings.Join(lines, "\n"), contentHeight))
}

// renderThreadContextPanel renders owner/target/history context to the right of description/comments.
func (m Model) renderThreadContextPanel(accent, muted, dim color.Color, sectionTitleStyle, hintStyle lipgloss.Style, width, height int) string {
	contentWidth := max(18, width-4)
	contentHeight := max(4, height-2)
	lines := []string{
		sectionTitleStyle.Render("Owner"),
		hintStyle.Render(truncate(m.threadActorName()+" ("+m.threadActorID()+")", contentWidth)),
		hintStyle.Render("type: " + string(m.threadActorType())),
		"",
		sectionTitleStyle.Render("Target"),
		hintStyle.Render(truncate(fmt.Sprintf("project: %s", m.threadTarget.ProjectID), contentWidth)),
		hintStyle.Render(truncate(fmt.Sprintf("type: %s", m.threadTarget.TargetType), contentWidth)),
		hintStyle.Render(truncate(fmt.Sprintf("id: %s", m.threadTarget.TargetID), contentWidth)),
		hintStyle.Render(fmt.Sprintf("comments: %d", len(m.threadComments))),
		"",
		sectionTitleStyle.Render("Brief History"),
	}
	if len(m.threadComments) == 0 {
		lines = append(lines, hintStyle.Render("(no comments yet)"))
	} else {
		start := max(0, len(m.threadComments)-5)
		for idx := start; idx < len(m.threadComments); idx++ {
			comment := m.threadComments[idx]
			actor := string(normalizeCommentActorType(string(comment.ActorType)))
			label := fmt.Sprintf("[%s] %s • %s", actor, threadCommentOwnerLabel(comment), formatThreadTimestamp(comment.CreatedAt))
			lines = append(lines, hintStyle.Render(truncate(label, contentWidth)))
			if summary := strings.TrimSpace(commentSummaryText(comment)); summary != "" {
				lines = append(lines, hintStyle.Render("  "+truncate("summary: "+summary, max(8, contentWidth-2))))
			}
		}
	}
	lines = append(lines, "", hintStyle.Render("enter: open edit when details focused"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dim).
		Padding(0, 1).
		Width(width).
		Render(fitLines(strings.Join(lines, "\n"), contentHeight))
}

// renderThreadDescriptionEditorView renders a full-screen markdown description editor with live Glamour preview.
func (m Model) renderThreadDescriptionEditorView(accent, muted, dim color.Color, titleStyle, hintStyle, statusStyle, sectionTitleStyle lipgloss.Style) tea.View {
	header := titleStyle.Render("thread description editor") + statusStyle.Render("  [markdown]")
	targetLine := hintStyle.Render(fmt.Sprintf("target: %s/%s", m.threadTarget.TargetType, m.threadTarget.TargetID))

	layoutWidth := max(72, m.width-2)
	if m.width <= 0 {
		layoutWidth = 120
	}
	availableHeight := 20
	if m.height > 0 {
		availableHeight = m.height - lipgloss.Height(header) - lipgloss.Height(targetLine) - 4
	}
	if availableHeight < 10 {
		availableHeight = 10
	}

	editorWidth := max(30, (layoutWidth-1)/2)
	previewWidth := max(30, layoutWidth-editorWidth-1)
	editor := m.threadDetailsInput
	editor.ShowLineNumbers = true
	editor.SetWidth(max(24, editorWidth-4))
	editor.SetHeight(max(8, availableHeight-2))
	_ = editor.Focus()
	editorPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(0, 1).
		Width(editorWidth).
		Render(sectionTitleStyle.Render("Editor") + "\n" + editor.View())

	previewMarkdown := strings.TrimSpace(editor.Value())
	if previewMarkdown == "" {
		previewMarkdown = "(empty description)"
	}
	previewContent := m.threadMarkdown.render(previewMarkdown, max(20, previewWidth-4))
	previewPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dim).
		Padding(0, 1).
		Width(previewWidth).
		Render(sectionTitleStyle.Render("Preview (Glamour)") + "\n" + fitLines(strings.TrimSpace(previewContent), max(6, availableHeight-2)))

	workspace := lipgloss.JoinHorizontal(lipgloss.Top, editorPanel, lipgloss.NewStyle().MarginLeft(1).Render(previewPanel))
	footer := hintStyle.Render("ctrl+s save description • esc cancel • enter newline • ? help")
	statusLine := ""
	if statusText := strings.TrimSpace(m.status); statusText != "" && statusText != "ready" {
		statusLine = statusStyle.Render(statusText)
	}
	parts := []string{header, targetLine, "", workspace, "", footer}
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	content := strings.Join(parts, "\n")
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

// threadCommentListLines renders comment metadata and markdown body lines for the comments panel.
func (m Model) threadCommentListLines(width int, hintStyle lipgloss.Style) []string {
	if len(m.threadComments) == 0 {
		return []string{hintStyle.Render("(no comments yet)")}
	}
	lines := make([]string, 0, len(m.threadComments)*4)
	for idx, comment := range m.threadComments {
		owner := threadCommentOwnerLabel(comment)
		actor := string(normalizeCommentActorType(string(comment.ActorType)))
		lines = append(lines, hintStyle.Render(fmt.Sprintf("[%s] %s • %s", actor, owner, formatThreadTimestamp(comment.CreatedAt))))
		if summary := commentSummaryText(comment); summary != "" {
			lines = append(lines, hintStyle.Render("summary: "+truncate(summary, max(24, width))))
		}
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

// threadSectionStyle returns one shared section-heading style used by thread views.
func threadSectionStyle(accent color.Color) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(accent)
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
	m.threadDescriptionMarkdown = m.threadDescriptionForTarget(target, description)
	m.threadComments = nil
	m.threadScroll = 0
	m.threadPendingCommentBody = ""
	m.threadComposerActive = false
	m.threadDetailsActive = false
	m.threadDetailsEditorActive = false
	m.threadInput.SetValue("")
	m.threadInput.CursorEnd()
	m.threadInput.Blur()
	m.threadDetailsInput.SetValue(m.threadDescriptionMarkdown)
	m.threadDetailsInput.CursorEnd()
	m.threadDetailsInput.Blur()
	m.status = "loading thread..."
	return m, m.loadThreadCommentsCmd(target)
}

// threadDescriptionForTarget resolves the thread description body, falling back to the backing project/task details when needed.
func (m Model) threadDescriptionForTarget(target domain.CommentTarget, description string) string {
	if details := strings.TrimSpace(description); details != "" {
		return details
	}
	if target.TargetType == domain.CommentTargetTypeProject {
		projectID := strings.TrimSpace(target.TargetID)
		if projectID == "" {
			projectID = strings.TrimSpace(target.ProjectID)
		}
		for _, project := range m.projects {
			if strings.TrimSpace(project.ID) == projectID {
				return strings.TrimSpace(project.Description)
			}
		}
		return ""
	}
	taskID := strings.TrimSpace(target.TargetID)
	if taskID == "" {
		return ""
	}
	task, ok := m.taskByID(taskID)
	if !ok {
		return ""
	}
	return strings.TrimSpace(task.Description)
}

// startThreadEditFlow transitions thread read mode into the matching project/task edit flow.
func (m Model) startThreadEditFlow() (tea.Model, tea.Cmd) {
	target := m.threadTarget
	switch target.TargetType {
	case domain.CommentTargetTypeProject:
		projectID := strings.TrimSpace(target.TargetID)
		if projectID == "" {
			projectID = strings.TrimSpace(target.ProjectID)
		}
		if projectID == "" {
			m.status = "thread project target unavailable"
			return m, nil
		}
		for _, project := range m.projects {
			if strings.TrimSpace(project.ID) != projectID {
				continue
			}
			return m, m.startProjectForm(&project)
		}
		m.status = "thread project not found"
		return m, nil
	default:
		taskID := strings.TrimSpace(target.TargetID)
		if taskID == "" {
			m.status = "thread work item target unavailable"
			return m, nil
		}
		task, ok := m.taskByID(taskID)
		if !ok {
			m.status = "thread work item not found"
			return m, nil
		}
		return m, m.startTaskForm(&task)
	}
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

// updateThreadDescriptionCmd updates one thread target's backing markdown details from the thread details editor.
func (m Model) updateThreadDescriptionCmd(description string) tea.Cmd {
	target := m.threadTarget
	description = strings.TrimSpace(description)
	actorID := m.threadActorID()
	actorType := m.threadActorType()
	return func() tea.Msg {
		switch target.TargetType {
		case domain.CommentTargetTypeProject:
			projectID := strings.TrimSpace(target.TargetID)
			if projectID == "" {
				projectID = strings.TrimSpace(target.ProjectID)
			}
			if projectID == "" {
				return actionMsg{err: fmt.Errorf("thread details update: project target unavailable")}
			}
			var project domain.Project
			found := false
			for _, candidate := range m.projects {
				if strings.TrimSpace(candidate.ID) == projectID {
					project = candidate
					found = true
					break
				}
			}
			if !found {
				return actionMsg{err: fmt.Errorf("thread details update: project %q not found", projectID)}
			}
			_, err := m.svc.UpdateProject(context.Background(), app.UpdateProjectInput{
				ProjectID:   project.ID,
				Name:        project.Name,
				Description: description,
				Kind:        project.Kind,
				Metadata:    project.Metadata,
				UpdatedBy:   actorID,
				UpdatedType: actorType,
			})
			if err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "thread details updated", reload: true}
		default:
			taskID := strings.TrimSpace(target.TargetID)
			if taskID == "" {
				return actionMsg{err: fmt.Errorf("thread details update: task target unavailable")}
			}
			task, ok := m.taskByID(taskID)
			if !ok {
				return actionMsg{err: fmt.Errorf("thread details update: task %q not found", taskID)}
			}
			metadata := task.Metadata
			_, err := m.svc.UpdateTask(context.Background(), app.UpdateTaskInput{
				TaskID:      task.ID,
				Title:       task.Title,
				Description: description,
				Priority:    task.Priority,
				DueAt:       task.DueAt,
				Labels:      append([]string(nil), task.Labels...),
				Metadata:    &metadata,
				UpdatedBy:   actorID,
				UpdatedType: actorType,
			})
			if err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "thread details updated", reload: true}
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

// commentSummaryText returns the normalized summary text used in thread and task-info read views.
func commentSummaryText(comment domain.Comment) string {
	return domain.NormalizeCommentSummary(comment.Summary, comment.BodyMarkdown)
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
