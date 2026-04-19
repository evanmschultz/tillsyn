// Package tui wires the gitdiff Differ + Highlighter (P4-T1 / P4-T2) into a
// full-page `diff` input mode reachable via ctrl+d.
//
// The mode owns one viewport sized to the shared full-page surface chrome,
// composes an optional branch-divergence banner for Diverged results, and
// renders a "No changes" placeholder for empty patches. Differ + Highlighter
// both enter through consumer-side interfaces so unit tests inject fakes and
// operator runs use the real chroma-backed pipeline.
package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/tui/gitdiff"
)

// diffModeDivergedBanner is the exact banner string rendered when `git diff`
// reports that the start commit is not an ancestor of HEAD. The leading space
// and em-dash are contractual — Drop 1.5 P4-T3 acceptance criteria pin the
// literal so regressions in the rendering pipeline surface as test failures,
// not as silently drifting operator-visible text.
const diffModeDivergedBanner = " branch-start-commit is NOT ancestor of HEAD — showing diff anyway"

// diffModeEmptyPlaceholder renders when the patch body is empty. The explicit
// "No changes" token (as opposed to an empty viewport) distinguishes a clean
// tree from a broken differ invocation.
const diffModeEmptyPlaceholder = "No changes"

// diffMode holds the state for the ctrl+d full-page diff surface.
//
// The struct is intentionally unexported because the Model owns the only live
// instance and every interaction routes through Model-level methods; exposing
// it would invite parallel construction paths that bypass the shared
// initializer. Viewport dimensions mirror the chrome metrics computed by the
// shared full-page surface helpers so resize math stays in one place.
type diffMode struct {
	differ      gitdiff.Differ
	highlighter gitdiff.Highlighter

	viewport viewport.Model

	result gitdiff.DiffResult
	err    error

	width  int
	height int

	// activeItem is the plan-item whose ResourceRefs were last used to resolve
	// the diff path list. SetItem writes it on each enterDiffMode call so path
	// resolution is always fresh and never cached across sessions.
	activeItem *domain.ActionItem
}

// SetItem records the active plan-item and feeds its ResourceRefs into the next
// Differ.Diff invocation. Callers (enterDiffMode) must call SetItem before
// issuing diffModeCmd so the resolved path list reflects the current board
// selection. Passing nil is safe — resolveDiffPaths returns an empty slice,
// causing Differ.Diff to fall back to whole-repo behaviour.
func (d *diffMode) SetItem(item *domain.ActionItem) {
	if d == nil {
		return
	}
	d.activeItem = item
}

// resolvePaths returns the path list derived from the most-recently set active
// plan-item's ResourceRefs. It delegates to the pure resolveDiffPaths function
// so the stored activeItem field is the single source of truth for path
// resolution, making SetItem load-bearing.
func (d *diffMode) resolvePaths() []string {
	if d == nil {
		return nil
	}
	return resolveDiffPaths(d.activeItem)
}

// resolveDiffPaths derives the path list for git diff from the active plan-item's
// ResourceRefs. It is a pure function (no receiver) so it can be called from
// tests without constructing a full diffMode.
//
// Partition rules (Tags[0] governs):
//   - "path" or "file" → append Location unchanged.
//   - "package"        → append Location with a trailing "/" (not doubled).
//   - zero Tags        → skip.
//   - any other tag    → skip silently.
//
// Deduplication preserves first-occurrence order. If the same Location appears
// as both a path/file ref and a package ref the trailing-slash (package) form
// wins — it is replaced in-place at the position the first occurrence occupied,
// so ordering is stable.
//
// An empty result (nil item / nil refs / all-skipped) signals Differ.Diff to
// run whole-repo (conventional git diff behaviour).
func resolveDiffPaths(item *domain.ActionItem) []string {
	if item == nil {
		return nil
	}
	refs := item.Metadata.ResourceRefs
	if len(refs) == 0 {
		return nil
	}

	out := make([]string, 0, len(refs))
	// seenIdx maps the bare location (without trailing slash) to its current
	// index in out. This lets package refs upgrade an earlier path/file entry
	// in-place without altering position.
	seenIdx := make(map[string]int, len(refs))

	for _, ref := range refs {
		if len(ref.Tags) == 0 {
			continue
		}
		tag := ref.Tags[0]
		loc := ref.Location

		switch tag {
		case "path", "file":
			bare := strings.TrimSuffix(loc, "/")
			if idx, exists := seenIdx[bare]; exists {
				// Already present. If the existing entry is already the
				// trailing-slash (package) form, a path/file ref cannot
				// downgrade it — skip.
				if strings.HasSuffix(out[idx], "/") {
					continue
				}
				// Existing entry is the bare form; a path/file ref is the
				// same — skip duplicate.
				continue
			}
			seenIdx[bare] = len(out)
			out = append(out, bare)

		case "package":
			bare := strings.TrimSuffix(loc, "/")
			slashed := bare + "/"
			if idx, exists := seenIdx[bare]; exists {
				// Upgrade bare path/file entry to trailing-slash form in-place.
				out[idx] = slashed
			} else {
				seenIdx[bare] = len(out)
				out = append(out, slashed)
			}

		default:
			// Unknown tag — skip silently.
		}
	}

	return out
}

// newDiffMode constructs a diffMode with Differ + Highlighter injected. Both
// arguments are interfaces so unit tests can inject deterministic fakes while
// the production wire-up passes the exec-backed Differ and chroma-backed
// Highlighter.
func newDiffMode(d gitdiff.Differ, h gitdiff.Highlighter) *diffMode {
	vp := viewport.New()
	vp.SoftWrap = false
	vp.MouseWheelEnabled = true
	vp.FillHeight = true
	return &diffMode{
		differ:      d,
		highlighter: h,
		viewport:    vp,
	}
}

// resize updates the inner viewport dimensions to match the latest chrome
// metrics. The viewport content is re-flowed through renderInto so the banner
// placement and "No changes" placeholder track terminal resizes.
func (d *diffMode) resize(width, height int) {
	if d == nil {
		return
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	d.width = width
	d.height = height
	d.viewport.SetWidth(width)
	d.viewport.SetHeight(height)
	d.renderInto()
}

// apply stashes one differ result (or error) and re-renders the viewport.
// Callers are expected to have already resolved the async differ call through
// a tea.Cmd; apply is the synchronous commit point.
func (d *diffMode) apply(result gitdiff.DiffResult, err error) {
	if d == nil {
		return
	}
	d.result = result
	d.err = err
	d.renderInto()
}

// reset clears the cached patch so the next ctrl+d opens on a clean surface.
// This matters more than it sounds: gitdiff may return multi-megabyte patches,
// and retaining the byte buffer on esc would pin that allocation for the rest
// of the session (falsification vector 3 / Drop 1.5 P4-T3).
func (d *diffMode) reset() {
	if d == nil {
		return
	}
	d.result = gitdiff.DiffResult{}
	d.err = nil
	d.viewport.SetContent("")
	d.viewport.GotoTop()
}

// viewContent returns the current rendered body, banner prefix included when
// the active result is divergent. The string is what the full-page surface
// places inside its bordered box.
func (d *diffMode) viewContent() string {
	if d == nil {
		return ""
	}
	banner := d.bannerLine()
	body := d.viewport.View()
	if banner == "" {
		return body
	}
	return banner + "\n" + body
}

// bannerLine returns the single-line banner for Diverged results, or an empty
// string otherwise. The banner text is exact and matches the acceptance
// contract pinned in TestDiffMode_Render_Diverged_Banner.
func (d *diffMode) bannerLine() string {
	if d == nil {
		return ""
	}
	if d.err != nil {
		return ""
	}
	if d.result.Divergence == gitdiff.DivergenceDiverged {
		return diffModeDivergedBanner
	}
	return ""
}

// renderInto writes the current result/error into the viewport body. Error
// renders skip the highlighter entirely so the user sees a plain message
// instead of chroma-styled noise, and so the returned body cannot be mistaken
// for successful patch output by scanners.
func (d *diffMode) renderInto() {
	if d == nil {
		return
	}
	if d.err != nil {
		d.viewport.SetContent(fmt.Sprintf("diff error: %s", d.err.Error()))
		d.viewport.GotoTop()
		return
	}
	patch := strings.TrimRight(d.result.Patch, "\n")
	if strings.TrimSpace(patch) == "" {
		d.viewport.SetContent(diffModeEmptyPlaceholder)
		d.viewport.GotoTop()
		return
	}
	styled, err := d.highlighter.Highlight(patch)
	if err != nil {
		d.viewport.SetContent(fmt.Sprintf("diff highlight error: %s\n\n%s", err.Error(), patch))
		d.viewport.GotoTop()
		return
	}
	d.viewport.SetContent(strings.TrimRight(styled, "\n"))
	d.viewport.GotoTop()
}

// diffModeStartRev resolves the start revision for ctrl+d diff invocations.
// Until a richer branch-selection UI lands the default compares HEAD against
// the upstream branch-root commit label tracked by `git`.
func diffModeStartRev() string {
	return "HEAD~1"
}

// diffModeEndRev resolves the end revision for ctrl+d diff invocations.
func diffModeEndRev() string {
	return "HEAD"
}

// diffLoadedMsg signals that a Differ call completed and its result should be
// committed into the diff-mode viewport on the Update loop.
type diffLoadedMsg struct {
	result gitdiff.DiffResult
	err    error
}

// diffModeCmd spawns a Differ call on the tea.Cmd queue so the UI stays
// responsive while `git diff` runs. The injected Differ is the same instance
// held by the Model; this keeps the async hop purely about loop-yielding
// rather than threading.
func diffModeCmd(d gitdiff.Differ, start, end string, paths []string) tea.Cmd {
	if d == nil {
		return nil
	}
	return func() tea.Msg {
		result, err := d.Diff(context.Background(), start, end, paths)
		return diffLoadedMsg{result: result, err: err}
	}
}

// renderDiffModeView renders the full-screen diff surface through the shared
// bordered-box chrome so terminal framing matches task-info / thread views.
func (m Model) renderDiffModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	title, subtitle := m.diffModeSurfaceHeader()
	status := ""
	if m.diff != nil {
		status = fullPageScrollStatus(m.diff.viewport)
	}
	boxWidth := actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth()))
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, title, subtitle, status)

	body := ""
	if m.diff != nil {
		// Re-sync the viewport to the latest chrome metrics before rendering so
		// mid-session resizes observed through WindowSizeMsg stay in step with
		// the active surface body.
		m.diff.resize(metrics.contentWidth, max(1, metrics.bodyHeight-1))
		body = m.diff.viewContent()
	}
	surface := renderFullPageSurfaceBody(accent, muted, metrics.boxWidth, title, subtitle, status, body)
	return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
}

// diffModeSurfaceHeader returns the shared bordered-surface title and subtitle
// for diff mode, sourcing the SHA/divergence fields from the active result.
func (m Model) diffModeSurfaceHeader() (string, string) {
	title := "Git Diff"
	if m.diff == nil {
		return title, ""
	}
	start := strings.TrimSpace(m.diff.result.StartSHA)
	end := strings.TrimSpace(m.diff.result.EndSHA)
	divergence := m.diff.result.Divergence.String()
	if start == "" && end == "" {
		return title, fmt.Sprintf("divergence: %s", divergence)
	}
	return title, fmt.Sprintf("start: %s • end: %s • divergence: %s", truncateSHA(start), truncateSHA(end), divergence)
}

// truncateSHA shortens one commit SHA to the conventional 7-char prefix for
// surface labels. Empty input returns the single-hyphen placeholder so the
// header never renders a literal empty string between bullets.
func truncateSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return "-"
	}
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// enterDiffMode transitions from the board (or any other read surface) into
// diff mode, capturing the prior mode so esc can restore it and kicking off
// the Differ call on the async tea.Cmd queue.
//
// The active board task (if any) is passed to SetItem before diffModeCmd fires so
// the path list reflects the current plan-item's ResourceRefs. When no task is
// selected the Differ falls back to whole-repo behaviour (empty path list).
func (m Model) enterDiffMode() (tea.Model, tea.Cmd) {
	if m.diff == nil {
		m.status = "diff mode unavailable"
		return m, nil
	}
	prior := m.mode
	m.diffBackMode = prior
	m.mode = modeDiff
	m.status = "loading diff..."
	// Resolve the active task's ResourceRefs fresh on each entry so path lists
	// are never cached across sessions (P4-T4 acceptance criterion).
	var activeTask *domain.ActionItem
	if task, ok := m.selectedActionItemInCurrentColumn(); ok {
		activeTask = &task
	}
	m.diff.SetItem(activeTask)
	paths := m.diff.resolvePaths()
	return m, diffModeCmd(m.diff.differ, diffModeStartRev(), diffModeEndRev(), paths)
}

// exitDiffMode restores the prior mode captured on entry and clears the cached
// patch to avoid pinning large byte buffers after esc. The diff struct itself
// stays alive so the next ctrl+d reuses the viewport rather than reallocating.
func (m Model) exitDiffMode() (tea.Model, tea.Cmd) {
	if m.diff != nil {
		m.diff.reset()
	}
	m.mode = m.diffBackMode
	m.diffBackMode = modeNone
	m.status = "ready"
	return m, nil
}

// handleDiffModeKey dispatches key presses while the diff surface is active.
// esc restores the prior mode; scroll keys delegate to the inner viewport so
// ctrl+d (half-page-down inside the viewport) and the mode-toggle ctrl+d on
// entry do not collide — the toggle only fires in the normal-mode dispatcher,
// never while m.mode == modeDiff.
func (m Model) handleDiffModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeDiff || m.diff == nil {
		return m, nil
	}
	if msg.Code == tea.KeyEscape || msg.String() == "esc" {
		return m.exitDiffMode()
	}
	var cmd tea.Cmd
	m.diff.viewport, cmd = m.diff.viewport.Update(msg)
	return m, cmd
}

// applyDiffLoadedMsg commits a Differ result into the active diff-mode
// viewport. When the user has already pressed esc before the Differ call
// returned the result is dropped silently.
func (m Model) applyDiffLoadedMsg(msg diffLoadedMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeDiff || m.diff == nil {
		return m, nil
	}
	m.diff.apply(msg.result, msg.err)
	if msg.err != nil {
		m.status = "diff error: " + msg.err.Error()
	} else {
		m.status = "diff loaded"
	}
	return m, nil
}
