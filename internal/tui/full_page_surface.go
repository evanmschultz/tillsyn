package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// fullPageSurfaceMetrics stores measured layout values for one full-page bordered surface.
type fullPageSurfaceMetrics struct {
	innerWidth   int
	boxWidth     int
	contentWidth int
	bodyHeight   int
	surfaceGapY  int
	headerBlock  string
	helpLine     string
	statusLine   string
}

// appHeaderBlock renders the shared TILLSYN header with inline path context and the divider rule below it.
func (m Model) appHeaderBlock(statusStyle lipgloss.Style, innerWidth int) string {
	headerAccent := lipgloss.Color("62")
	header := headerMarkStyle().
		BorderForeground(headerAccent).
		Foreground(lipgloss.Color("252")).
		Render(headerMarkText)
	row := header
	if pathText := m.appHeaderPathText(max(16, innerWidth-lipgloss.Width(header)-2)); pathText != "" {
		pathWidth := max(16, innerWidth-lipgloss.Width(header)-2)
		pathBlock := lipgloss.NewStyle().
			MarginLeft(2).
			Width(pathWidth).
			Render(lipgloss.PlaceVertical(lipgloss.Height(header), lipgloss.Center, statusStyle.Render(pathText)))
		row = lipgloss.JoinHorizontal(lipgloss.Top, header, pathBlock)
	}
	if innerWidth > 0 {
		row = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Left, row)
	}
	rule := lipgloss.NewStyle().
		Foreground(headerAccent).
		Render(strings.Repeat("─", max(8, innerWidth)))
	return strings.Join([]string{row, rule}, "\n")
}

// appHeaderPathText renders the shared path label when a project/task path is available.
func (m Model) appHeaderPathText(maxWidth int) string {
	projectName := ""
	if project, ok := m.currentProject(); ok {
		projectName = projectDisplayName(project)
	}
	if path, _ := m.projectionPathWithProject(projectName); path != "" {
		return "path: " + collapsePathForDisplay(path, max(12, maxWidth-6))
	}
	return ""
}

// fullPageSurfaceMetrics computes the measured chrome and remaining body height for one full-page surface.
func (m Model) fullPageSurfaceMetrics(accent, muted, dim color.Color, boxWidth int, title, subtitle, status string) fullPageSurfaceMetrics {
	const (
		surfaceHorizontalInset = 1
		surfaceVerticalInset   = 1
	)
	innerWidth := max(36, m.width-2*tuiOuterHorizontalPadding)
	if m.width <= 0 {
		innerWidth = max(96, boxWidth)
	}
	maxBoxWidth := innerWidth - (2 * surfaceHorizontalInset)
	if maxBoxWidth < 36 {
		maxBoxWidth = innerWidth
	}
	boxWidth = clamp(boxWidth, 36, maxBoxWidth)
	contentWidth := max(24, boxWidth-8)

	statusStyle := lipgloss.NewStyle().Foreground(dim)
	headerBlock := m.appHeaderBlock(statusStyle, innerWidth)

	helpBubble := m.help
	helpBubble.ShowAll = false
	helpBubble.SetWidth(innerWidth)
	helpLine := lipgloss.NewStyle().
		Foreground(muted).
		BorderTop(true).
		BorderForeground(dim).
		Width(innerWidth).
		Render(helpBubble.View(m.activeBottomHelpKeyMap()))

	boxChrome := fullPageSurfaceBoxChrome(accent, muted, boxWidth, title, subtitle, status)
	statusText := strings.TrimSpace(m.status)
	if statusText == "ready" {
		statusText = ""
	}
	statusLine := ""
	if statusText != "" {
		statusLine = statusStyle.Render(truncate(statusText, innerWidth))
	}

	bodyHeight := taskInfoBodyViewportMinHeight
	if m.height > 0 {
		bodyHeight = m.height -
			lipgloss.Height(headerBlock) -
			lipgloss.Height(helpLine) -
			lipgloss.Height(statusLine) -
			lipgloss.Height(boxChrome) -
			nodeModalBoxStyle(accent, boxWidth).GetVerticalFrameSize() -
			(2 * surfaceVerticalInset)
	}
	bodyHeight = clamp(bodyHeight, taskInfoBodyViewportMinHeight, taskInfoBodyViewportMaxHeight)

	return fullPageSurfaceMetrics{
		innerWidth:   innerWidth,
		boxWidth:     boxWidth,
		contentWidth: contentWidth,
		bodyHeight:   bodyHeight,
		surfaceGapY:  surfaceVerticalInset,
		headerBlock:  headerBlock,
		helpLine:     helpLine,
		statusLine:   statusLine,
	}
}

// fullPageSurfaceBoxChrome renders the non-body lines that consume height inside one bordered surface.
func fullPageSurfaceBoxChrome(accent, muted color.Color, boxWidth int, title, subtitle, status string) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	hintStyle := lipgloss.NewStyle().Foreground(muted)
	lines := []string{titleStyle.Render(strings.TrimSpace(title))}
	if strings.TrimSpace(subtitle) != "" {
		lines = append(lines, hintStyle.Render(strings.TrimSpace(subtitle)))
	}
	lines = append(lines, titleStyle.Render(strings.Repeat("─", max(12, boxWidth-8))))
	if strings.TrimSpace(status) != "" {
		lines = append(lines, hintStyle.Render(strings.TrimSpace(status)))
	}
	return strings.Join(lines, "\n")
}

// renderFullPageSurfaceBody renders the shared bordered box for one full-page screen.
func renderFullPageSurfaceBody(accent, muted color.Color, boxWidth int, title, subtitle, status, body string) string {
	boxChrome := strings.TrimRight(fullPageSurfaceBoxChrome(accent, muted, boxWidth, title, subtitle, status), "\n")
	lines := boxChrome
	if strings.TrimSpace(body) != "" {
		lines += "\n" + body
	}
	return nodeModalBoxStyle(accent, boxWidth).Render(lines)
}

// renderFullPageSurfaceViewport renders one full-page surface backed by a viewport body.
func renderFullPageSurfaceViewport(accent, muted color.Color, boxWidth int, title, subtitle, status string, body viewport.Model) string {
	return renderFullPageSurfaceBody(accent, muted, boxWidth, title, subtitle, status, body.View())
}

// renderFullPageSurfaceView wraps one centered bordered surface with the shared TILLSYN header, path line, and bottom help.
func (m Model) renderFullPageSurfaceView(accent, muted, dim color.Color, metrics fullPageSurfaceMetrics, surface string) tea.View {
	centeredSurface := lipgloss.PlaceHorizontal(metrics.innerWidth, lipgloss.Center, surface)
	sections := []string{metrics.headerBlock}
	for i := 0; i < metrics.surfaceGapY; i++ {
		sections = append(sections, "")
	}
	sections = append(sections, centeredSurface)
	for i := 0; i < metrics.surfaceGapY; i++ {
		sections = append(sections, "")
	}
	if metrics.statusLine != "" {
		sections = append(sections, metrics.statusLine)
	}
	content := strings.Join(sections, "\n")
	if tuiOuterHorizontalPadding > 0 {
		content = lipgloss.NewStyle().
			PaddingLeft(tuiOuterHorizontalPadding).
			PaddingRight(tuiOuterHorizontalPadding).
			Render(content)
		metrics.helpLine = lipgloss.NewStyle().
			PaddingLeft(tuiOuterHorizontalPadding).
			PaddingRight(tuiOuterHorizontalPadding).
			Render(metrics.helpLine)
	}
	if m.height > 0 {
		content = fitLines(content, max(0, m.height-lipgloss.Height(metrics.helpLine)))
	}
	fullContent := content + "\n" + metrics.helpLine
	if m.help.ShowAll {
		overlay := m.renderHelpOverlay(accent, muted, dim, lipgloss.NewStyle().Foreground(muted), m.width-8)
		if overlay != "" {
			overlayHeight := lipgloss.Height(fullContent)
			if m.height > 0 {
				overlayHeight = m.height
			}
			fullContent = overlayOnContent(fullContent, overlay, max(1, m.width), max(1, overlayHeight))
		}
	}

	view := tea.NewView(fullContent)
	view.MouseMode = m.activeMouseMode()
	view.AltScreen = true
	return view
}

// fullPageScrollStatus returns a shared scroll-percent status line for viewport-backed surfaces.
func fullPageScrollStatus(body viewport.Model) string {
	return fmt.Sprintf("scroll: %d%%", int(body.ScrollPercent()*100))
}
