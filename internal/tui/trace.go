package tui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/log"
)

// traceValueMaxRunes caps per-field trace value rendering to keep debug logs compact.
const traceValueMaxRunes = 96

// globalNoticeTransitionSequence provides monotonic ids for global-notification transitions.
var globalNoticeTransitionSequence uint64

// globalNoticeTransitionTrace tracks one global-notification Enter transition.
type globalNoticeTransitionTrace struct {
	ID        string
	StartedAt time.Time
	LoadCount int
}

// nextGlobalNoticeTransitionID returns the next stable id for global-notification tracing.
func nextGlobalNoticeTransitionID() string {
	seq := atomic.AddUint64(&globalNoticeTransitionSequence, 1)
	return fmt.Sprintf("global-notice-%06d", seq)
}

// traceDebug emits one structured debug event through the package-level runtime logger.
func traceDebug(msg string, keyvals ...any) {
	log.Debug(msg, keyvals...)
}

// traceActionItemScreenAction emits one structured actionItem-screen interaction trace.
func traceActionItemScreenAction(screen, action string, keyvals ...any) {
	fields := []any{
		"screen", strings.TrimSpace(screen),
		"action", strings.TrimSpace(action),
	}
	fields = append(fields, keyvals...)
	traceDebug("tui.task_screen.action", fields...)
}

// globalNoticeTransitionID returns the active global-notification transition id.
func (m Model) globalNoticeTransitionID() string {
	return strings.TrimSpace(m.globalNoticeTransition.ID)
}

// beginGlobalNoticeTransition starts a new Enter-triggered global-notification transition.
func (m *Model) beginGlobalNoticeTransition(msg tea.KeyPressMsg) {
	if m == nil {
		return
	}
	previousID := m.globalNoticeTransitionID()
	if previousID != "" {
		m.completeGlobalNoticeTransition("superseded")
	}
	transitionID := nextGlobalNoticeTransitionID()
	m.globalNoticeTransition = globalNoticeTransitionTrace{
		ID:        transitionID,
		StartedAt: time.Now().UTC(),
	}
	traceDebug(
		"tui.global_notification.enter_pressed",
		"transition_id", transitionID,
		"panel", "global",
		"key_code", msg.Code,
		"key_string", sanitizeControlRunes(msg.String()),
		"key_text", sanitizeControlRunes(msg.Text),
		"selected_index", m.globalNoticesIdx,
		"item_count", len(m.globalNoticesPanelItemsForInteraction()),
	)
}

// completeGlobalNoticeTransition closes the active global-notification transition.
func (m *Model) completeGlobalNoticeTransition(reason string, keyvals ...any) {
	if m == nil {
		return
	}
	transitionID := m.globalNoticeTransitionID()
	if transitionID == "" {
		return
	}
	elapsed := int64(0)
	if !m.globalNoticeTransition.StartedAt.IsZero() {
		elapsed = time.Since(m.globalNoticeTransition.StartedAt).Milliseconds()
	}
	fields := []any{
		"transition_id", transitionID,
		"reason", strings.TrimSpace(reason),
		"elapsed_ms", elapsed,
		"load_count", m.globalNoticeTransition.LoadCount,
	}
	fields = append(fields, keyvals...)
	traceDebug("tui.global_notification.transition_complete", fields...)
	m.globalNoticeTransition = globalNoticeTransitionTrace{}
}

// markGlobalNoticeApplyLoadedCompletion traces one applyLoadedMsg completion while a transition is active.
func (m *Model) markGlobalNoticeApplyLoadedCompletion(startedAt time.Time, loadErr error) {
	if m == nil {
		return
	}
	transitionID := m.globalNoticeTransitionID()
	if transitionID == "" {
		return
	}
	m.globalNoticeTransition.LoadCount++
	transitionElapsed := int64(0)
	if !m.globalNoticeTransition.StartedAt.IsZero() {
		transitionElapsed = time.Since(m.globalNoticeTransition.StartedAt).Milliseconds()
	}
	hasPendingProject := strings.TrimSpace(m.pendingProjectID) != ""
	hasPendingFocusActionItem := strings.TrimSpace(m.pendingFocusActionItemID) != ""
	hasPendingActionItemInfo := strings.TrimSpace(m.pendingOpenActionItemInfoID) != ""
	_, _, _, hasPendingThread := m.pendingNotificationThread()
	traceFields := []any{
		"transition_id", transitionID,
		"apply_loaded_ms", time.Since(startedAt).Milliseconds(),
		"transition_elapsed_ms", transitionElapsed,
		"load_count", m.globalNoticeTransition.LoadCount,
		"load_error", loadErr != nil,
		"pending_project", hasPendingProject,
		"pending_focus_task", hasPendingFocusActionItem,
		"pending_task_info", hasPendingActionItemInfo,
		"pending_thread", hasPendingThread,
	}
	if loadErr != nil {
		traceFields = append(traceFields, "error", loadErr.Error())
	}
	traceDebug("tui.global_notification.apply_loaded_complete", traceFields...)
	if loadErr != nil {
		m.completeGlobalNoticeTransition("load_error")
		return
	}
	if hasPendingProject || hasPendingFocusActionItem || hasPendingActionItemInfo || hasPendingThread {
		return
	}
	m.completeGlobalNoticeTransition("apply_loaded_settled")
}

// traceGlobalNoticeBranch emits one branch decision while a transition is active.
func (m Model) traceGlobalNoticeBranch(branch string, keyvals ...any) {
	transitionID := m.globalNoticeTransitionID()
	if transitionID == "" {
		return
	}
	fields := []any{
		"transition_id", transitionID,
		"branch", strings.TrimSpace(branch),
	}
	fields = append(fields, keyvals...)
	traceDebug("tui.global_notification.branch", fields...)
}

// traceGlobalNoticePending emits one pending-state mutation trace while a transition is active.
func (m Model) traceGlobalNoticePending(action, field, value string, keyvals ...any) {
	transitionID := m.globalNoticeTransitionID()
	if transitionID == "" {
		return
	}
	sanitized := sanitizeControlRunes(value)
	fields := []any{
		"transition_id", transitionID,
		"action", strings.TrimSpace(action),
		"field", strings.TrimSpace(field),
		"has_value", strings.TrimSpace(value) != "",
	}
	if strings.TrimSpace(sanitized) != "" {
		fields = append(fields, "value", clipTraceValue(sanitized, traceValueMaxRunes))
	}
	fields = append(fields, keyvals...)
	traceDebug("tui.global_notification.pending_state", fields...)
}

// traceGlobalNoticeKeyDispatch emits key dispatch details while a transition is active.
func (m Model) traceGlobalNoticeKeyDispatch(msg tea.KeyPressMsg) {
	transitionID := m.globalNoticeTransitionID()
	if transitionID == "" {
		return
	}
	traceDebug(
		"tui.global_notification.key_dispatch",
		"transition_id", transitionID,
		"mode", int(m.mode),
		"notices_focused", m.noticesFocused,
		"notices_panel", int(m.noticesPanel),
		"key_code", msg.Code,
		"key_string", sanitizeControlRunes(msg.String()),
		"key_text", sanitizeControlRunes(msg.Text),
	)
}

// traceLoadDataStage emits one loadData stage timing event with optional context.
func (m Model) traceLoadDataStage(stage string, startedAt time.Time, stageErr error, keyvals ...any) {
	duration := time.Since(startedAt)
	if !m.shouldTraceLoadDataStage(stage, duration, stageErr) {
		return
	}
	fields := []any{
		"stage", strings.TrimSpace(stage),
		"duration_ms", duration.Milliseconds(),
		"error", stageErr != nil,
	}
	if transitionID := m.globalNoticeTransitionID(); transitionID != "" {
		fields = append(fields, "transition_id", transitionID)
	}
	if stageErr != nil {
		fields = append(fields, "error_message", stageErr.Error())
	}
	fields = append(fields, keyvals...)
	traceDebug("tui.load_data.stage", fields...)
}

// shouldTraceLoadDataStage reports whether one load-data stage should be logged.
func (m Model) shouldTraceLoadDataStage(stage string, duration time.Duration, stageErr error) bool {
	if m.globalNoticeTransitionID() != "" {
		return true
	}
	if stageErr != nil {
		return true
	}
	// Keep background auto-refresh noise low; only emit non-transition load timing
	// when a full reload is unexpectedly slow.
	return strings.TrimSpace(stage) == "total" && duration >= 50*time.Millisecond
}

// traceFormControlCharacterGuard logs pre-persistence guard details for fields containing control characters.
func (m Model) traceFormControlCharacterGuard(entity, operation, field, value string) {
	if !containsControlRunes(value) {
		return
	}
	transitionID := m.globalNoticeTransitionID()
	fields := []any{
		"entity", strings.TrimSpace(entity),
		"operation", strings.TrimSpace(operation),
		"field", strings.TrimSpace(field),
		"control_count", countControlRunes(value),
		"value_runes", utf8.RuneCountInString(value),
		"value_sanitized", clipTraceValue(sanitizeControlRunes(value), traceValueMaxRunes),
	}
	if transitionID != "" {
		fields = append(fields, "transition_id", transitionID)
	}
	traceDebug("tui.form.control_character_guard", fields...)
}

// containsControlRunes reports whether value includes one or more control runes.
func containsControlRunes(value string) bool {
	return countControlRunes(value) > 0
}

// countControlRunes returns the number of control runes contained in value.
func countControlRunes(value string) int {
	count := 0
	for _, r := range value {
		if unicode.IsControl(r) {
			count++
		}
	}
	return count
}

// sanitizeControlRunes rewrites control runes to escaped unicode markers for readable traces.
func sanitizeControlRunes(value string) string {
	if value == "" {
		return ""
	}
	var out strings.Builder
	for _, r := range value {
		if !unicode.IsControl(r) {
			out.WriteRune(r)
			continue
		}
		if r <= 0xFFFF {
			out.WriteString(fmt.Sprintf("\\u%04X", r))
			continue
		}
		out.WriteString(fmt.Sprintf("\\U%08X", r))
	}
	return out.String()
}

// clipTraceValue truncates value by rune-count and appends an ellipsis marker when clipping occurs.
func clipTraceValue(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes]) + "..."
}
