package tui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// filePickerRenderOptions bundles render inputs kept out of the core so the
// renderer can style without extending filePickerCore state.
type filePickerRenderOptions struct {
	accent   color.Color
	muted    color.Color
	dim      color.Color
	boxWidth int
	maxRows  int
	status   string
}

// filePickerTitleFor returns the canonical header text for one variant.
func filePickerTitleFor(mode filePickerMode) string {
	switch mode {
	case filePickerModePath:
		return "Add Paths"
	default:
		return "File Picker"
	}
}

// filePickerSubtitleFor returns the hint line shown below the header.
func filePickerSubtitleFor(mode filePickerMode) string {
	switch mode {
	case filePickerModePath:
		return "tag path refs on the active item (Tags[0] == \"path\")"
	default:
		return "pick filesystem entries"
	}
}

// renderFilePickerBody renders the scrollable entry list inside the full-page
// surface body. Layout:
//
//	root: <root path>
//	current: <current dir>
//	filter: <query>
//	<entries>
//	<footer hint>
//
// Render respects maxRows — extra entries collapse to a "+N more" line.
func renderFilePickerBody(core filePickerCore, opts filePickerRenderOptions) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(opts.accent)
	hintStyle := lipgloss.NewStyle().Foreground(opts.muted)
	cursorStyle := lipgloss.NewStyle().Foreground(opts.accent).Bold(true)
	selStyle := lipgloss.NewStyle().Foreground(opts.accent)

	currentPath := strings.TrimSpace(core.dir)
	if currentPath == "" {
		currentPath = core.root
	}
	if absPath, err := filepath.Abs(currentPath); err == nil {
		currentPath = absPath
	}

	filterInput := core.filter
	filterInput.SetWidth(max(20, min(72, opts.boxWidth-18)))

	lines := []string{
		hintStyle.Render("root: " + truncate(core.root, 72)),
		hintStyle.Render("current: ") + titleStyle.Render(truncate(currentPath, 72)),
		filterInput.View(),
	}

	visible := core.visibleEntries()
	maxRows := opts.maxRows
	if maxRows <= 0 {
		maxRows = 14
	}

	if len(visible) == 0 {
		lines = append(lines, hintStyle.Render("(no entries matching filter)"))
	} else {
		for idx, entry := range visible {
			if idx >= maxRows {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more entries", len(visible)-idx)))
				break
			}
			cursor := "  "
			if idx == core.index {
				cursor = cursorStyle.Render("> ")
			}
			marker := " "
			if core.isSelected(entry) {
				marker = selStyle.Render("*")
			}
			name := entry.Name
			if entry.IsDir {
				name += "/"
			}
			lines = append(lines, fmt.Sprintf("%s%s %s", cursor, marker, name))
		}
	}

	selectedCount := len(core.selected)
	lines = append(lines, hintStyle.Render(fmt.Sprintf("selected: %d", selectedCount)))
	lines = append(lines, hintStyle.Render("tab/space select • enter accept • →/l open dir • ←/h parent • ctrl+u clear • esc cancel"))

	return strings.Join(lines, "\n")
}

// renderFilePickerSurface composes the full-page bordered surface used by the
// file-picker variants. It reuses renderFullPageSurfaceView /
// fullPageSurfaceBoxChrome so chrome stays identical to other TUI surfaces and
// no sibling viewport shim is introduced.
func (m Model) renderFilePickerSurface(accent, muted, dim color.Color) tea.View {
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, m.appInnerWidth(), filePickerTitleFor(m.pickerCore.mode), filePickerSubtitleFor(m.pickerCore.mode), "")
	body := renderFilePickerBody(m.pickerCore, filePickerRenderOptions{
		accent:   accent,
		muted:    muted,
		dim:      dim,
		boxWidth: metrics.boxWidth,
		maxRows:  metrics.bodyHeight - 6,
	})
	surface := renderFullPageSurfaceBody(accent, muted, metrics.boxWidth, filePickerTitleFor(m.pickerCore.mode), filePickerSubtitleFor(m.pickerCore.mode), "", body)
	return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
}
