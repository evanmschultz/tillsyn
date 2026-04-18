package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// descriptionEditorLayoutMetrics stores rendered dimensions for description-editor panels.
type descriptionEditorLayoutMetrics struct {
	layoutWidth int

	// Preview-mode metrics.
	previewModePanelContentHeight int
	previewModeBodyHeight         int

	// Edit-mode metrics.
	splitVertically           bool
	editorWidth               int
	previewWidth              int
	editorPanelContentHeight  int
	previewPanelContentHeight int
	editorBodyHeight          int
	previewBodyHeight         int
}

// descriptionEditorFrame stores single-line header/footer text after width clamping.
type descriptionEditorFrame struct {
	header string
	path   string
	footer string
	status string
}

// renderDescriptionEditorModeView renders the dedicated full-screen description editor surface.
func (m Model) renderDescriptionEditorModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	sectionTitleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	title := "Description Editor"
	subtitle := "mode: edit"
	if m.descriptionEditorMode == descriptionEditorViewModePreview {
		subtitle = "mode: preview"
	}
	boxWidth := actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth()))
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, title, subtitle, "")
	layout := m.descriptionEditorLayout()
	workspace := ""
	if m.descriptionEditorMode == descriptionEditorViewModePreview {
		preview := m.descriptionEditorPreviewViewport(max(20, layout.layoutWidth-4), max(4, layout.previewModeBodyHeight), false)
		workspace = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1).
			Width(layout.layoutWidth).
			Render(fitLines(sectionTitleStyle.Render("Preview")+"\n"+preview.View(), layout.previewModePanelContentHeight))
	} else {
		editor := m.descriptionEditorInput
		editor.ShowLineNumbers = true
		editor.SetWidth(max(20, layout.editorWidth-4))
		editor.SetHeight(max(3, layout.editorBodyHeight))
		preview := m.descriptionEditorPreviewViewport(max(20, layout.previewWidth-4), max(3, layout.previewBodyHeight), true)

		editorPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1).
			Width(layout.editorWidth).
			Render(fitLines(sectionTitleStyle.Render("Editor")+"\n"+editor.View(), layout.editorPanelContentHeight))
		previewPanel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dim).
			Padding(0, 1).
			Width(layout.previewWidth).
			Render(fitLines(sectionTitleStyle.Render("Preview")+"\n"+preview.View(), layout.previewPanelContentHeight))

		if layout.splitVertically {
			workspace = lipgloss.JoinVertical(
				lipgloss.Left,
				editorPanel,
				lipgloss.NewStyle().MarginTop(1).Render(previewPanel),
			)
		} else {
			workspace = lipgloss.JoinHorizontal(
				lipgloss.Top,
				editorPanel,
				lipgloss.NewStyle().MarginLeft(1).Render(previewPanel),
			)
		}
	}
	surface := renderFullPageSurfaceBody(accent, muted, metrics.boxWidth, title, subtitle, "", workspace)
	return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
}

// descriptionEditorLayout computes render dimensions for edit/preview submodes.
func (m Model) descriptionEditorLayout() descriptionEditorLayoutMetrics {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	title := "Description Editor"
	subtitle := "mode: edit"
	if m.descriptionEditorMode == descriptionEditorViewModePreview {
		subtitle = "mode: preview"
	}
	boxWidth := actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth()))
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, title, subtitle, "")
	layoutWidth := max(20, metrics.contentWidth)
	workspaceHeight := max(10, metrics.bodyHeight)

	previewModePanelContentHeight := max(6, workspaceHeight-2)
	previewModeBodyHeight := max(4, previewModePanelContentHeight-1)
	layout := descriptionEditorLayoutMetrics{
		layoutWidth:                   layoutWidth,
		previewModePanelContentHeight: previewModePanelContentHeight,
		previewModeBodyHeight:         previewModeBodyHeight,
		editorWidth:                   layoutWidth,
		previewWidth:                  layoutWidth,
		editorPanelContentHeight:      previewModePanelContentHeight,
		previewPanelContentHeight:     previewModePanelContentHeight,
		editorBodyHeight:              previewModeBodyHeight,
		previewBodyHeight:             previewModeBodyHeight,
	}
	if m.descriptionEditorMode == descriptionEditorViewModePreview {
		return layout
	}

	const minHorizontalPanelWidth = 30
	if layoutWidth >= (minHorizontalPanelWidth*2)+1 {
		editorWidth := (layoutWidth - 1) / 2
		previewWidth := layoutWidth - editorWidth - 1
		if editorWidth >= minHorizontalPanelWidth && previewWidth >= minHorizontalPanelWidth {
			layout.editorWidth = editorWidth
			layout.previewWidth = previewWidth
			layout.editorPanelContentHeight = max(6, workspaceHeight-2)
			layout.previewPanelContentHeight = layout.editorPanelContentHeight
			layout.editorBodyHeight = max(4, layout.editorPanelContentHeight-1)
			layout.previewBodyHeight = max(4, layout.previewPanelContentHeight-1)
			return layout
		}
	}

	// On narrow terminals, stack editor above preview to avoid horizontal overflow.
	layout.splitVertically = true
	availablePanelContent := max(4, workspaceHeight-5) // 2 borders + 2 borders + 1 spacer line.
	layout.editorPanelContentHeight = max(4, availablePanelContent/2)
	layout.previewPanelContentHeight = max(4, availablePanelContent-layout.editorPanelContentHeight)
	layout.editorBodyHeight = max(3, layout.editorPanelContentHeight-1)
	layout.previewBodyHeight = max(3, layout.previewPanelContentHeight-1)
	return layout
}

// descriptionEditorFrameText returns single-line clamped text for the description-editor frame.
func (m Model) descriptionEditorFrameText(lineWidth int) descriptionEditorFrame {
	lineWidth = max(20, lineWidth)
	pathPrefix := "path: "
	pathWidth := max(8, lineWidth-len(pathPrefix))
	status := strings.TrimSpace(m.status)
	if status == "ready" {
		status = ""
	}
	return descriptionEditorFrame{
		header: truncate("Description Editor", lineWidth),
		path:   pathPrefix + collapsePathForDisplay(m.descriptionEditorPathLabel(), pathWidth),
		footer: truncate(m.descriptionEditorFooterHint(), lineWidth),
		status: truncate(status, lineWidth),
	}
}

// descriptionEditorFooterHint returns bottom-hint text for description editor submodes.
func (m Model) descriptionEditorFooterHint() string {
	saveVerb := "save"
	if m.descriptionEditorBack == modeEditActionItem &&
		(m.descriptionEditorTarget == descriptionEditorTargetActionItem || m.descriptionEditorTarget == descriptionEditorTargetActionItemFormField) {
		saveVerb = "save actionItem"
	} else if m.descriptionEditorBack == modeAddActionItem {
		saveVerb = "apply field"
	}
	if m.descriptionEditorMode == descriptionEditorViewModePreview {
		return "preview mode • tab edit mode • j/k or pgup/pgdown scroll • home/end jump • ctrl+s " + saveVerb + " • esc cancel • ? help"
	}
	return "edit mode • tab preview mode • ctrl+z undo • ctrl+shift+z redo • ctrl+s " + saveVerb + " • esc cancel • enter newline • ? inserts '?'"
}

// descriptionEditorPathLabel returns the active path label for description editor header context.
func (m Model) descriptionEditorPathLabel() string {
	if path := strings.TrimSpace(m.descriptionEditorPath); path != "" {
		return path
	}
	switch m.descriptionEditorTarget {
	case descriptionEditorTargetActionItem:
		return m.descriptionEditorPathForActionItemForm()
	case descriptionEditorTargetProject:
		return m.descriptionEditorPathForProjectForm()
	case descriptionEditorTargetThread:
		return m.descriptionEditorPathForThreadTarget()
	default:
		return "(node path unavailable)"
	}
}

// descriptionEditorPreviewViewport prepares preview viewport rendering for current dimensions.
func (m Model) descriptionEditorPreviewViewport(width, height int, syncWithEditor bool) viewport.Model {
	vp := m.descriptionPreview
	prevYOffset := vp.YOffset()
	vp.SetWidth(max(1, width))
	vp.SetHeight(max(1, height))
	vp.SetContent(m.descriptionEditorRenderedPreview(max(1, width)))
	if syncWithEditor {
		vp.SetYOffset(max(0, m.descriptionEditorInput.ScrollYOffset()))
	} else {
		vp.SetYOffset(prevYOffset)
	}
	return vp
}

// descriptionEditorRenderedPreview returns rendered markdown preview content for the provided wrap width.
func (m Model) descriptionEditorRenderedPreview(width int) string {
	markdown := strings.TrimSpace(m.descriptionEditorInput.Value())
	if markdown == "" {
		markdown = "(empty description)"
	}
	return strings.TrimSpace(m.threadMarkdown.render(markdown, max(20, width)))
}

// syncDescriptionEditorViewportLayout aligns textarea/preview dimensions with the current screen layout.
func (m *Model) syncDescriptionEditorViewportLayout() {
	if m == nil {
		return
	}
	layout := m.descriptionEditorLayout()
	if m.descriptionEditorMode == descriptionEditorViewModePreview {
		previewWidth := max(20, layout.layoutWidth-4)
		prevYOffset := m.descriptionPreview.YOffset()
		m.descriptionPreview.SetWidth(max(1, previewWidth))
		m.descriptionPreview.SetHeight(max(1, layout.previewModeBodyHeight))
		m.descriptionPreview.SetContent(m.descriptionEditorRenderedPreview(previewWidth))
		m.descriptionPreview.SetYOffset(prevYOffset)
		return
	}
	m.descriptionEditorInput.ShowLineNumbers = true
	m.descriptionEditorInput.SetWidth(max(20, layout.editorWidth-4))
	m.descriptionEditorInput.SetHeight(layout.editorBodyHeight)
	previewWidth := max(20, layout.previewWidth-4)
	prevYOffset := m.descriptionPreview.YOffset()
	m.descriptionPreview.SetWidth(max(1, previewWidth))
	m.descriptionPreview.SetHeight(max(1, layout.previewBodyHeight))
	m.descriptionPreview.SetContent(m.descriptionEditorRenderedPreview(previewWidth))
	m.descriptionPreview.SetYOffset(prevYOffset)
}

// syncDescriptionPreviewOffsetToEditor keeps preview viewport scrolling aligned to editor scroll offset.
func (m *Model) syncDescriptionPreviewOffsetToEditor() {
	if m == nil {
		return
	}
	m.syncDescriptionEditorViewportLayout()
	m.descriptionPreview.SetYOffset(max(0, m.descriptionEditorInput.ScrollYOffset()))
}

// resetDescriptionPreviewToTop refreshes preview layout and ensures preview-mode opens from the top.
func (m *Model) resetDescriptionPreviewToTop() {
	if m == nil {
		return
	}
	m.syncDescriptionEditorViewportLayout()
	m.descriptionPreview.SetYOffset(0)
}

// descriptionEditorPathForActionItemForm returns a project-rooted path label for actionItem-form description editing.
func (m Model) descriptionEditorPathForActionItemForm() string {
	if actionItemID := strings.TrimSpace(m.editingActionItemID); actionItemID != "" {
		if actionItem, ok := m.actionItemByID(actionItemID); ok {
			return m.descriptionEditorActionItemPath(actionItem)
		}
	}
	kind := strings.TrimSpace(string(m.actionItemFormKind))
	if kind == "" {
		kind = string(domain.WorkKindActionItem)
	}
	if parentID := strings.TrimSpace(m.actionItemFormParentID); parentID != "" {
		if parent, ok := m.actionItemByID(parentID); ok {
			return m.descriptionEditorActionItemPath(parent) + " -> " + fmt.Sprintf("(new %s)", kind)
		}
		return m.descriptionEditorProjectLabel("") + " -> " + parentID + " -> " + fmt.Sprintf("(new %s)", kind)
	}
	return m.descriptionEditorProjectLabel("") + " -> " + fmt.Sprintf("(new %s)", kind)
}

// descriptionEditorPathForProjectForm returns a path label for project-form description editing.
func (m Model) descriptionEditorPathForProjectForm() string {
	if projectID := strings.TrimSpace(m.editingProjectID); projectID != "" {
		return m.descriptionEditorProjectLabel(projectID)
	}
	return "(new project)"
}

// descriptionEditorPathForThreadTarget returns a path label for thread-target description editing.
func (m Model) descriptionEditorPathForThreadTarget() string {
	target := m.threadTarget
	projectLabel := m.descriptionEditorProjectLabel(target.ProjectID)
	targetID := strings.TrimSpace(target.TargetID)
	if target.TargetType == domain.CommentTargetTypeProject {
		if targetID != "" && projectLabel == "(project)" {
			return targetID
		}
		return projectLabel
	}
	if targetID != "" {
		if actionItem, ok := m.actionItemByID(targetID); ok {
			return m.descriptionEditorActionItemPath(actionItem)
		}
	}
	typeLabel := strings.TrimSpace(string(target.TargetType))
	if typeLabel == "" {
		typeLabel = "node"
	}
	if targetID == "" {
		targetID = "(unknown)"
	}
	return projectLabel + " -> " + typeLabel + ":" + targetID
}

// descriptionEditorProjectLabel resolves display-safe project text for description editor path labels.
func (m Model) descriptionEditorProjectLabel(projectID string) string {
	projectID = strings.TrimSpace(projectID)
	if projectID != "" {
		for _, project := range m.projects {
			if strings.TrimSpace(project.ID) != projectID {
				continue
			}
			if label := strings.TrimSpace(projectDisplayName(project)); label != "" {
				return label
			}
			return projectID
		}
		return projectID
	}
	if project, ok := m.currentProject(); ok {
		if label := strings.TrimSpace(projectDisplayName(project)); label != "" {
			return label
		}
		if id := strings.TrimSpace(project.ID); id != "" {
			return id
		}
	}
	return "(project)"
}

// descriptionEditorActionItemPath renders a project-rooted hierarchy path for one actionItem.
func (m Model) descriptionEditorActionItemPath(actionItem domain.ActionItem) string {
	chain := []string{fallbackText(strings.TrimSpace(actionItem.Title), "(untitled)")}
	visited := map[string]struct{}{actionItem.ID: {}}
	parentID := strings.TrimSpace(actionItem.ParentID)
	for parentID != "" {
		if _, seen := visited[parentID]; seen {
			break
		}
		parent, ok := m.actionItemByID(parentID)
		if !ok {
			break
		}
		visited[parentID] = struct{}{}
		chain = append(chain, fallbackText(strings.TrimSpace(parent.Title), "(untitled)"))
		parentID = strings.TrimSpace(parent.ParentID)
	}
	for left, right := 0, len(chain)-1; left < right; left, right = left+1, right-1 {
		chain[left], chain[right] = chain[right], chain[left]
	}
	chain = append([]string{m.descriptionEditorProjectLabel(actionItem.ProjectID)}, chain...)
	return strings.Join(chain, " -> ")
}
