package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evanmschultz/tillsyn/internal/config"
)

// defaultFileViewerConfig returns a FileViewerConfig with the package defaults.
// It is used by NewModel to initialize the file viewer without requiring a
// loaded config file.
func defaultFileViewerConfig() config.FileViewerConfig {
	return config.FileViewerConfig{
		MaxBytes:      config.DefaultFileViewerMaxBytes,
		DotfileBanner: config.DefaultFileViewerDotfileBanner,
	}
}

// ErrFileTooLarge is the sentinel error returned when the target file exceeds
// the configured max_bytes limit. Callers can use errors.Is to detect this case
// and render a user-facing banner rather than an opaque failure message.
var ErrFileTooLarge = errors.New("file too large")

// fileViewerBannerDotfile is the exact banner string rendered when a dotfile
// is opened. The literal is pinned by TestFileViewer_Dotfile_Refused so any
// change here breaks the test intentionally.
const fileViewerBannerDotfile = "Dotfiles not supported in v1"

// fileViewerMode holds the state for the v-key full-page file-viewer surface.
//
// The struct is intentionally unexported — the Model owns the single live
// instance and all interaction routes through Model-level methods. The
// markdownRenderer pointer is shared with Model.threadMarkdown so markdown
// rendering uses the same glamour pipeline as the thread view.
type fileViewerMode struct {
	md  *markdownRenderer
	cfg config.FileViewerConfig

	viewport viewport.Model

	filePath string
	content  string
	err      error

	width  int
	height int
}

// newFileViewerMode constructs a fileViewerMode backed by the given shared
// markdownRenderer and config. The markdownRenderer pointer must be
// Model.threadMarkdown (not a new renderer) to satisfy the pointer-equality
// invariant asserted by TestFileViewer_SharesThreadMarkdown.
func newFileViewerMode(md *markdownRenderer, cfg config.FileViewerConfig) *fileViewerMode {
	vp := viewport.New()
	vp.SoftWrap = true
	vp.MouseWheelEnabled = true
	vp.FillHeight = true
	return &fileViewerMode{
		md:       md,
		cfg:      cfg,
		viewport: vp,
	}
}

// openFile loads and renders the file at path into the mode's content buffer.
// It applies the following checks in order:
//
//  1. Dotfile guard — if filepath.Base(path) starts with ".", set a banner and
//     return without reading the file.
//  2. Size guard — os.Stat to obtain file size; if size > cfg.MaxBytes return
//     ErrFileTooLarge and set a banner. The file is never read past this point.
//  3. Read — os.ReadFile populates content bytes.
//  4. Render — chooseRenderer selects the pipeline based on file extension.
//
// All errors are wrapped with %w.
func (fv *fileViewerMode) openFile(path string) error {
	if fv == nil {
		return nil
	}
	fv.filePath = path
	fv.err = nil

	// 1. Dotfile guard.
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") {
		fv.content = fv.cfg.DotfileBanner
		if fv.content == "" {
			fv.content = fileViewerBannerDotfile
		}
		fv.renderInto()
		return nil
	}

	// 2. Size guard.
	info, err := os.Stat(path)
	if err != nil {
		fv.err = fmt.Errorf("file viewer: stat %s: %w", path, err)
		fv.content = fv.err.Error()
		fv.renderInto()
		return fv.err
	}
	maxBytes := int64(fv.cfg.MaxBytes)
	if maxBytes <= 0 {
		maxBytes = 1048576
	}
	if info.Size() > maxBytes {
		fv.err = fmt.Errorf("file viewer: %w: File too large (%d) — limit %d", ErrFileTooLarge, info.Size(), maxBytes)
		fv.content = fmt.Sprintf("File too large (%d) — limit %d", info.Size(), maxBytes)
		fv.renderInto()
		return fv.err
	}

	// 3. Read.
	raw, err := os.ReadFile(path)
	if err != nil {
		fv.err = fmt.Errorf("file viewer: read %s: %w", path, err)
		fv.content = fv.err.Error()
		fv.renderInto()
		return fv.err
	}

	// 4. Render.
	rendered, err := chooseRenderer(path, raw, fv.md)
	if err != nil {
		// Fall back to raw text on render error so the viewer never shows a blank pane.
		fv.content = string(raw)
	} else {
		fv.content = rendered
	}
	fv.renderInto()
	return nil
}

// renderInto writes the current content string into the viewport body.
func (fv *fileViewerMode) renderInto() {
	if fv == nil {
		return
	}
	fv.viewport.SetContent(fv.content)
	fv.viewport.GotoTop()
}

// resize adjusts the inner viewport to the new content dimensions.
func (fv *fileViewerMode) resize(width, height int) {
	if fv == nil {
		return
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	fv.width = width
	fv.height = height
	fv.viewport.SetWidth(width)
	fv.viewport.SetHeight(height)
}

// reset clears the loaded file so the next v press opens a clean surface.
func (fv *fileViewerMode) reset() {
	if fv == nil {
		return
	}
	fv.filePath = ""
	fv.content = ""
	fv.err = nil
	fv.viewport.SetContent("")
	fv.viewport.GotoTop()
}

// viewContent returns the rendered viewport body.
func (fv *fileViewerMode) viewContent() string {
	if fv == nil {
		return ""
	}
	return fv.viewport.View()
}

// enterFileViewerMode transitions from the current surface into file viewer
// mode. The active task's first path-type ResourceRef provides the file to open;
// when no eligible path is found the viewer shows a "no file" placeholder.
func (m Model) enterFileViewerMode() (tea.Model, tea.Cmd) {
	if m.fileViewer == nil {
		m.status = "file viewer unavailable"
		return m, nil
	}
	prior := m.mode
	m.fileViewerBackMode = prior
	m.mode = modeFileViewer
	m.status = "loading file..."

	// Derive the path from the active task's ResourceRefs (first path/file ref).
	path := ""
	if task, ok := m.selectedActionItemInCurrentColumn(); ok {
		for _, ref := range task.Metadata.ResourceRefs {
			if len(ref.Tags) > 0 {
				tag := strings.ToLower(strings.TrimSpace(ref.Tags[0]))
				if tag == "path" || tag == "file" {
					path = strings.TrimSpace(ref.Location)
					break
				}
			}
		}
	}
	if path == "" {
		m.fileViewer.content = "No file path — select a task with a path resource ref"
		m.fileViewer.renderInto()
		m.status = "no file"
		return m, nil
	}

	// Load file synchronously (files are local disk reads; no network hop).
	_ = m.fileViewer.openFile(path)
	if m.fileViewer.err != nil && !errors.Is(m.fileViewer.err, ErrFileTooLarge) {
		m.status = "file error: " + m.fileViewer.err.Error()
	} else {
		m.status = "file loaded"
	}
	return m, nil
}

// exitFileViewerMode restores the prior mode.
func (m Model) exitFileViewerMode() (tea.Model, tea.Cmd) {
	if m.fileViewer != nil {
		m.fileViewer.reset()
	}
	m.mode = m.fileViewerBackMode
	m.fileViewerBackMode = modeNone
	m.status = "ready"
	return m, nil
}

// handleFileViewerModeKey dispatches key presses while the file viewer is active.
func (m Model) handleFileViewerModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeFileViewer || m.fileViewer == nil {
		return m, nil
	}
	if msg.Code == tea.KeyEscape || msg.String() == "esc" {
		return m.exitFileViewerMode()
	}
	var cmd tea.Cmd
	m.fileViewer.viewport, cmd = m.fileViewer.viewport.Update(msg)
	return m, cmd
}

// renderFileViewerModeView renders the full-screen file viewer surface through
// the shared bordered-box chrome so framing matches the diff and task-info views.
func (m Model) renderFileViewerModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	title := "File Viewer"
	subtitle := ""
	status := ""
	if m.fileViewer != nil {
		if m.fileViewer.filePath != "" {
			subtitle = m.fileViewer.filePath
		}
		status = fullPageScrollStatus(m.fileViewer.viewport)
	}

	boxWidth := actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth()))
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, title, subtitle, status)

	body := ""
	if m.fileViewer != nil {
		m.fileViewer.resize(metrics.contentWidth, max(1, metrics.bodyHeight-1))
		body = m.fileViewer.viewContent()
	}
	surface := renderFullPageSurfaceBody(accent, muted, metrics.boxWidth, title, subtitle, status, body)
	return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
}
