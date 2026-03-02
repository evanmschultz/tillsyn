package tui

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// Service represents service data used by this package.
type Service interface {
	ListProjects(context.Context, bool) ([]domain.Project, error)
	ListColumns(context.Context, string, bool) ([]domain.Column, error)
	ListTasks(context.Context, string, bool) ([]domain.Task, error)
	CreateComment(context.Context, app.CreateCommentInput) (domain.Comment, error)
	ListCommentsByTarget(context.Context, app.ListCommentsByTargetInput) ([]domain.Comment, error)
	ListProjectChangeEvents(context.Context, string, int) ([]domain.ChangeEvent, error)
	ListAttentionItems(context.Context, app.ListAttentionItemsInput) ([]domain.AttentionItem, error)
	GetProjectDependencyRollup(context.Context, string) (domain.DependencyRollup, error)
	SearchTaskMatches(context.Context, app.SearchTasksFilter) ([]app.TaskMatch, error)
	CreateProjectWithMetadata(context.Context, app.CreateProjectInput) (domain.Project, error)
	UpdateProject(context.Context, app.UpdateProjectInput) (domain.Project, error)
	ArchiveProject(context.Context, string) (domain.Project, error)
	RestoreProject(context.Context, string) (domain.Project, error)
	DeleteProject(context.Context, string) error
	CreateTask(context.Context, app.CreateTaskInput) (domain.Task, error)
	UpdateTask(context.Context, app.UpdateTaskInput) (domain.Task, error)
	MoveTask(context.Context, string, string, int) (domain.Task, error)
	DeleteTask(context.Context, string, app.DeleteMode) error
	RestoreTask(context.Context, string) (domain.Task, error)
	RenameTask(context.Context, string, string) (domain.Task, error)
}

// inputMode represents a selectable mode.
type inputMode int

// modeNone and related constants define package defaults.
const (
	modeNone inputMode = iota
	modeAddTask
	modeSearch
	modeRenameTask
	modeEditTask
	modeDuePicker
	modeProjectPicker
	modeTaskInfo
	modeAddProject
	modeEditProject
	modeSearchResults
	modeCommandPalette
	modeQuickActions
	modeConfirmAction
	modeWarning
	modeActivityLog
	modeActivityEventInfo
	modeResourcePicker
	modeLabelPicker
	modePathsRoots
	modeLabelsConfig
	modeHighlightColor
	modeBootstrapSettings
	modeDependencyInspector
	modeThread
)

// taskFormFields stores task-form field keys in display/update order.
var taskFormFields = []string{"title", "description", "priority", "due", "labels", "depends_on", "blocked_by", "blocked_reason"}

// task-form field indexes used throughout keyboard/update logic.
const (
	taskFieldTitle = iota
	taskFieldDescription
	taskFieldPriority
	taskFieldDue
	taskFieldLabels
	taskFieldDependsOn
	taskFieldBlockedBy
	taskFieldBlockedReason
)

// project-form field indexes used for focused form actions.
const (
	projectFieldName = iota
	projectFieldDescription
	projectFieldOwner
	projectFieldIcon
	projectFieldColor
	projectFieldHomepage
	projectFieldTags
	projectFieldRootPath
)

// activity log limits used by modal rendering and retention.
const (
	activityLogMaxItems   = 200
	activityLogViewWindow = 14
	defaultHighlightColor = "212"
	// headerMarkText defines the boxed brand wordmark shown in the board header.
	headerMarkText = "TILLSYN"
	// tuiOuterHorizontalPadding keeps a small symmetric outer gutter around the whole TUI.
	tuiOuterHorizontalPadding = 1
	// boardColumnGapWidth is the horizontal spacing between adjacent board columns.
	boardColumnGapWidth = 1
	// noticesPanelGapWidth keeps the Done->Notices gap aligned with the outer gutter.
	noticesPanelGapWidth = tuiOuterHorizontalPadding
	// minimumColumnWidth is the minimum target style width for board columns.
	minimumColumnWidth = 24
	// minimumNoticesPanelWidth is the minimum target style width for the notices panel.
	minimumNoticesPanelWidth = 24
	// maximumNoticesPanelWidth caps notices panel growth to preserve board readability.
	maximumNoticesPanelWidth = 38
	// noticesSectionViewWindow caps visible rows per notices section before list scrolling is required.
	noticesSectionViewWindow = 4
)

// defaultLabelSuggestionsSeed provides baseline label suggestions before user/project customization exists.
var defaultLabelSuggestionsSeed = []string{"todo", "blocked", "urgent", "bug", "feature", "docs"}

// priorityOptions stores a package-level helper value.
var priorityOptions = []domain.Priority{
	domain.PriorityLow,
	domain.PriorityMedium,
	domain.PriorityHigh,
}

// duePickerOption defines a functional option for model configuration.
type duePickerOption struct {
	Label string
	Value string
}

// quickActionSpec defines one quick-action command and label.
type quickActionSpec struct {
	ID    string
	Label string
}

// quickActionItem defines one rendered quick-action entry with availability metadata.
type quickActionItem struct {
	ID             string
	Label          string
	Enabled        bool
	DisabledReason string
}

// quickActionSpecs stores the canonical quick-action ordering.
var quickActionSpecs = []quickActionSpec{
	{ID: "task-info", Label: "Task Info"},
	{ID: "edit-task", Label: "Edit Task"},
	{ID: "move-left", Label: "Move Left"},
	{ID: "move-right", Label: "Move Right"},
	{ID: "archive-task", Label: "Archive Task"},
	{ID: "restore-task", Label: "Restore Task"},
	{ID: "hard-delete", Label: "Hard Delete"},
	{ID: "toggle-selection", Label: "Toggle Selection"},
	{ID: "clear-selection", Label: "Clear Selection"},
	{ID: "bulk-move-left", Label: "Bulk Move Left"},
	{ID: "bulk-move-right", Label: "Bulk Move Right"},
	{ID: "bulk-archive", Label: "Bulk Archive"},
	{ID: "bulk-hard-delete", Label: "Bulk Hard Delete"},
	{ID: "undo", Label: "Undo"},
	{ID: "redo", Label: "Redo"},
	{ID: "activity-log", Label: "Activity Log"},
}

// canonicalSearchStates stores canonical searchable lifecycle states.
var canonicalSearchStatesOrdered = []string{"todo", "progress", "done", "archived"}

// canonicalSearchLevelsOrdered stores canonical searchable hierarchy levels.
var canonicalSearchLevelsOrdered = []string{"project", "branch", "phase", "subphase", "task", "subtask"}

// bootstrapActorTypes stores canonical actor-type options for bootstrap settings.
var bootstrapActorTypes = []string{
	string(domain.ActorTypeUser),
	string(domain.ActorTypeAgent),
	string(domain.ActorTypeSystem),
}

// canonicalSearchStateLabels stores display labels for canonical lifecycle states.
var canonicalSearchStateLabels = map[string]string{
	"todo":     "To Do",
	"progress": "In Progress",
	"done":     "Done",
	"archived": "Archived",
}

// canonicalSearchLevelLabels stores display labels for canonical hierarchy levels.
var canonicalSearchLevelLabels = map[string]string{
	"project":  "Project",
	"branch":   "Branch",
	"phase":    "Phase",
	"subphase": "Subphase",
	"task":     "Task",
	"subtask":  "Subtask",
}

// commandPaletteItem describes one command-palette command.
type commandPaletteItem struct {
	Command     string
	Aliases     []string
	Description string
}

// resourcePickerEntry describes one filesystem candidate in the resource picker.
type resourcePickerEntry struct {
	Name  string
	Path  string
	IsDir bool
}

// labelPickerItem describes one inherited label suggestion and its source.
type labelPickerItem struct {
	Label  string
	Source string
}

// labelInheritanceSources groups inherited labels by source precedence.
type labelInheritanceSources struct {
	Global  []string
	Project []string
	Branch  []string
	Phase   []string
}

// dependencyCandidate describes one dependency-picker result row.
type dependencyCandidate struct {
	Match app.TaskMatch
	Path  string
}

// confirmAction describes a pending confirmation action.
type confirmAction struct {
	Kind    string
	Task    domain.Task
	Project domain.Project
	TaskIDs []string
	Mode    app.DeleteMode
	Label   string
}

// activityEntry describes one recorded user action for the in-app activity log.
type activityEntry struct {
	At         time.Time
	Summary    string
	Target     string
	EventID    int64
	WorkItemID string
	Operation  domain.ChangeOperation
	ActorID    string
	ActorName  string
	ActorType  domain.ActorType
	Metadata   map[string]string
}

// noticesSectionID identifies one focusable list section in the notices panel.
type noticesSectionID int

// Notices panel section identifiers.
const (
	noticesSectionWarnings noticesSectionID = iota
	noticesSectionAttention
	noticesSectionSelection
	noticesSectionRecentActivity
)

// noticesPanelItem describes one selectable row in a notices-panel section.
type noticesPanelItem struct {
	Label       string
	TaskID      string
	Activity    activityEntry
	HasActivity bool
}

// noticesPanelSection describes one notices-panel list section.
type noticesPanelSection struct {
	ID      noticesSectionID
	Title   string
	Summary []string
	Items   []noticesPanelItem
}

// noticesPanelSectionOrder stores the stable section traversal order for notices navigation.
var noticesPanelSectionOrder = []noticesSectionID{
	noticesSectionWarnings,
	noticesSectionAttention,
	noticesSectionSelection,
	noticesSectionRecentActivity,
}

// historyStepKind identifies one reversible operation in a mutation set.
type historyStepKind string

// history step kinds used for undo/redo.
const (
	historyStepMove       historyStepKind = "move"
	historyStepArchive    historyStepKind = "archive"
	historyStepRestore    historyStepKind = "restore"
	historyStepHardDelete historyStepKind = "hard-delete"
)

// historyStep describes one mutation required to replay or reverse a change.
type historyStep struct {
	Kind         historyStepKind
	TaskID       string
	FromColumnID string
	FromPosition int
	ToColumnID   string
	ToPosition   int
}

// historyActionSet describes one logical user mutation for undo/redo.
type historyActionSet struct {
	ID       int
	Label    string
	Summary  string
	Target   string
	Steps    []historyStep
	Undoable bool
	At       time.Time
}

// Model represents model data used by this package.
type Model struct {
	svc Service

	ready  bool
	width  int
	height int
	err    error

	status   string
	warnings []string

	help help.Model
	keys keyMap

	taskFields        TaskFieldConfig
	defaultDeleteMode app.DeleteMode

	projects                 []domain.Project
	selectedProject          int
	columns                  []domain.Column
	tasks                    []domain.Task
	selectedColumn           int
	selectedTask             int
	launchPicker             bool
	startupBootstrapRequired bool

	mode                  inputMode
	input                 string
	searchQuery           string
	searchApplied         bool
	showArchived          bool
	showArchivedProjects  bool
	searchIncludeArchived bool

	searchInput                 textinput.Model
	commandInput                textinput.Model
	bootstrapDisplayInput       textinput.Model
	pathsRootInput              textinput.Model
	highlightColorInput         textinput.Model
	dependencyInput             textinput.Model
	threadInput                 textinput.Model
	searchFocus                 int
	searchStateCursor           int
	searchLevelCursor           int
	searchCrossProject          bool
	searchDefaultCrossProject   bool
	searchDefaultIncludeArchive bool
	searchStates                []string
	searchDefaultStates         []string
	searchLevels                []string
	searchDefaultLevels         []string
	searchMatches               []app.TaskMatch
	searchResultIndex           int
	quickActionIndex            int
	commandMatches              []commandPaletteItem
	commandIndex                int
	bootstrapFocus              int
	bootstrapActorIndex         int
	bootstrapRoots              []string
	bootstrapRootIndex          int
	bootstrapMandatory          bool
	dependencyFocus             int
	dependencyStateCursor       int
	dependencyCrossProject      bool
	dependencyIncludeArchived   bool
	dependencyStates            []string
	dependencyMatches           []dependencyCandidate
	dependencyIndex             int

	formInputs           []textinput.Model
	formFocus            int
	priorityIdx          int
	duePicker            int
	duePickerFocus       int
	duePickerIncludeTime bool
	pickerBack           inputMode
	duePickerDateInput   textinput.Model
	duePickerTimeInput   textinput.Model
	// taskFormResourceRefs stages resource refs while creating or editing a task.
	taskFormResourceRefs []domain.ResourceRef

	projectPickerIndex       int
	projectFormInputs        []textinput.Model
	projectFormFocus         int
	labelsConfigInputs       []textinput.Model
	labelsConfigFocus        int
	labelsConfigSlug         string
	labelsConfigBranchTaskID string
	labelsConfigPhaseTaskID  string
	editingProjectID         string
	editingTaskID            string
	taskInfoTaskID           string
	taskInfoOriginTaskID     string
	taskInfoSubtaskIdx       int
	taskFormParentID         string
	taskFormKind             domain.WorkKind
	taskFormScope            domain.KindAppliesTo
	pendingProjectID         string
	pendingFocusTaskID       string
	pendingActivityJumpTask  string

	lastArchivedTaskID string

	confirmDelete     bool
	confirmArchive    bool
	confirmHardDelete bool
	confirmRestore    bool
	pendingConfirm    confirmAction
	confirmChoice     int
	warningTitle      string
	warningBody       string

	boardGroupBy    string
	showWIPWarnings bool
	dueSoonWindows  []time.Duration
	showDueSummary  bool
	searchRoots     []string
	projectRoots    map[string]string
	defaultRootDir  string
	highlightColor  string

	projectionRootTaskID string

	selectedTaskIDs  map[string]struct{}
	activityLog      []activityEntry
	noticesFocused   bool
	noticesSection   noticesSectionID
	noticesWarnings  int
	noticesAttention int
	noticesSelection int
	noticesActivity  int
	activityInfoItem activityEntry
	undoStack        []historyActionSet
	redoStack        []historyActionSet
	nextHistoryID    int
	dependencyRollup domain.DependencyRollup

	resourcePickerBack   inputMode
	resourcePickerTaskID string
	resourcePickerRoot   string
	resourcePickerDir    string
	resourcePickerIndex  int
	resourcePickerItems  []resourcePickerEntry
	resourcePickerFilter textinput.Model

	labelPickerBack     inputMode
	labelPickerIndex    int
	labelPickerItems    []labelPickerItem
	labelPickerAllItems []labelPickerItem
	labelPickerInput    textinput.Model

	dependencyBack        inputMode
	dependencyOwnerTaskID string
	dependencyDependsOn   []string
	dependencyBlockedBy   []string
	dependencyActiveField int
	dependencyDirty       bool

	allowedLabelGlobal   []string
	allowedLabelProject  map[string][]string
	enforceAllowedLabels bool

	mouseSelectionMode bool

	reloadConfig    ReloadConfigFunc
	saveProjectRoot SaveProjectRootFunc
	saveBootstrap   SaveBootstrapConfigFunc
	saveLabels      SaveLabelsConfigFunc

	identityDisplayName      string
	identityActorID          string
	identityDefaultActorType string

	threadBackMode            inputMode
	threadTarget              domain.CommentTarget
	threadTitle               string
	threadDescriptionMarkdown string
	threadComments            []domain.Comment
	threadScroll              int
	threadPendingCommentBody  string
	threadMarkdown            markdownRenderer

	autoRefreshInterval time.Duration
	autoRefreshArmed    bool
	autoRefreshInFlight bool
}

// loadedMsg carries message data through update handling.
type loadedMsg struct {
	projects                 []domain.Project
	selectedProject          int
	columns                  []domain.Column
	tasks                    []domain.Task
	activityEntries          []activityEntry
	rollup                   domain.DependencyRollup
	err                      error
	attentionItemsCount      int
	attentionUserActionCount int
}

// resourcePickerLoadedMsg carries resource picker directory entries.
type resourcePickerLoadedMsg struct {
	root    string
	current string
	entries []resourcePickerEntry
	err     error
}

// actionMsg carries message data through update handling.
type actionMsg struct {
	err             error
	status          string
	reload          bool
	projectID       string
	projectRootSlug string
	projectRootPath string
	focusTaskID     string
	clearSelect     bool
	clearTaskIDs    []string
	historyPush     *historyActionSet
	historyUndo     *historyActionSet
	historyRedo     *historyActionSet
	activityItem    *activityEntry
}

// autoRefreshTickMsg triggers a periodic external-state refresh attempt.
type autoRefreshTickMsg struct{}

// autoRefreshLoadedMsg carries one background refresh load result.
type autoRefreshLoadedMsg struct {
	data loadedMsg
	err  error
}

// searchResultsMsg carries message data through update handling.
type searchResultsMsg struct {
	matches []app.TaskMatch
	err     error
}

// dependencyMatchesMsg carries dependency-candidate matches for the inspector modal.
type dependencyMatchesMsg struct {
	candidates []dependencyCandidate
	err        error
}

// activityLogLoadedMsg carries persisted activity entries for the active project.
type activityLogLoadedMsg struct {
	entries []activityEntry
	err     error
}

// configReloadedMsg carries runtime settings loaded through the reload callback.
type configReloadedMsg struct {
	config RuntimeConfig
	err    error
}

// projectRootSavedMsg carries one persisted project-root mapping update.
type projectRootSavedMsg struct {
	projectSlug string
	rootPath    string
	err         error
}

// bootstrapSettingsSavedMsg carries bootstrap-settings persistence results.
type bootstrapSettingsSavedMsg struct {
	config BootstrapConfig
	err    error
}

// threadLoadedMsg carries comments loaded for one thread target.
type threadLoadedMsg struct {
	target   domain.CommentTarget
	comments []domain.Comment
	err      error
}

// threadCommentCreatedMsg carries one persisted comment result for the active thread.
type threadCommentCreatedMsg struct {
	target domain.CommentTarget
	body   string
	value  domain.Comment
	err    error
}

// NewModel constructs a new value for this package.
func NewModel(svc Service, opts ...Option) Model {
	h := help.New()
	h.ShowAll = false
	searchInput := textinput.New()
	searchInput.Prompt = ""
	searchInput.Placeholder = "title, description, labels"
	searchInput.CharLimit = 120
	configureTextInputClipboardBindings(&searchInput)
	commandInput := textinput.New()
	commandInput.Prompt = ": "
	commandInput.Placeholder = "type to filter commands"
	commandInput.CharLimit = 120
	configureTextInputClipboardBindings(&commandInput)
	bootstrapDisplayInput := textinput.New()
	bootstrapDisplayInput.Prompt = ""
	bootstrapDisplayInput.Placeholder = "display name"
	bootstrapDisplayInput.CharLimit = 120
	configureTextInputClipboardBindings(&bootstrapDisplayInput)
	pathsRootInput := textinput.New()
	pathsRootInput.Prompt = "root: "
	pathsRootInput.Placeholder = "absolute path (empty clears mapping)"
	pathsRootInput.CharLimit = 512
	configureTextInputClipboardBindings(&pathsRootInput)
	highlightColorInput := textinput.New()
	highlightColorInput.Prompt = "color: "
	highlightColorInput.Placeholder = "ansi index (e.g. 212) or #RRGGBB"
	highlightColorInput.CharLimit = 32
	configureTextInputClipboardBindings(&highlightColorInput)
	dependencyInput := textinput.New()
	dependencyInput.Prompt = "query: "
	dependencyInput.Placeholder = "search title, description, labels"
	dependencyInput.CharLimit = 120
	configureTextInputClipboardBindings(&dependencyInput)
	threadInput := textinput.New()
	threadInput.Prompt = "comment: "
	threadInput.Placeholder = "write markdown and press enter"
	threadInput.CharLimit = 4000
	configureTextInputClipboardBindings(&threadInput)
	resourcePickerFilter := textinput.New()
	resourcePickerFilter.Prompt = "filter: "
	resourcePickerFilter.Placeholder = "type to fuzzy-filter files/dirs"
	resourcePickerFilter.CharLimit = 120
	configureTextInputClipboardBindings(&resourcePickerFilter)
	duePickerDateInput := textinput.New()
	duePickerDateInput.Prompt = "date: "
	duePickerDateInput.Placeholder = "today | tomorrow | 2026-03-01"
	duePickerDateInput.CharLimit = 64
	configureTextInputClipboardBindings(&duePickerDateInput)
	duePickerTimeInput := textinput.New()
	duePickerTimeInput.Prompt = "time: "
	duePickerTimeInput.Placeholder = "17:00"
	duePickerTimeInput.CharLimit = 16
	configureTextInputClipboardBindings(&duePickerTimeInput)
	labelPickerInput := textinput.New()
	labelPickerInput.Prompt = "filter: "
	labelPickerInput.Placeholder = "type to fuzzy-find labels"
	labelPickerInput.CharLimit = 120
	configureTextInputClipboardBindings(&labelPickerInput)
	m := Model{
		svc:                      svc,
		status:                   "loading...",
		help:                     h,
		keys:                     newKeyMap(),
		taskFields:               DefaultTaskFieldConfig(),
		defaultDeleteMode:        app.DeleteModeArchive,
		searchInput:              searchInput,
		commandInput:             commandInput,
		bootstrapDisplayInput:    bootstrapDisplayInput,
		pathsRootInput:           pathsRootInput,
		highlightColorInput:      highlightColorInput,
		dependencyInput:          dependencyInput,
		threadInput:              threadInput,
		resourcePickerFilter:     resourcePickerFilter,
		duePickerDateInput:       duePickerDateInput,
		duePickerTimeInput:       duePickerTimeInput,
		labelPickerInput:         labelPickerInput,
		searchStates:             []string{"todo", "progress", "done"},
		searchDefaultStates:      []string{"todo", "progress", "done"},
		searchLevels:             []string{"project", "branch", "phase", "subphase", "task", "subtask"},
		searchDefaultLevels:      []string{"project", "branch", "phase", "subphase", "task", "subtask"},
		dependencyStates:         []string{"todo", "progress", "done"},
		launchPicker:             false,
		boardGroupBy:             "none",
		showWIPWarnings:          true,
		dueSoonWindows:           []time.Duration{24 * time.Hour, time.Hour},
		showDueSummary:           true,
		highlightColor:           defaultHighlightColor,
		selectedTaskIDs:          map[string]struct{}{},
		activityLog:              []activityEntry{},
		noticesSection:           noticesSectionRecentActivity,
		confirmDelete:            true,
		confirmArchive:           true,
		confirmHardDelete:        true,
		confirmRestore:           false,
		taskFormKind:             domain.WorkKindTask,
		taskFormScope:            domain.KindAppliesToTask,
		allowedLabelProject:      map[string][]string{},
		searchRoots:              []string{},
		projectRoots:             map[string]string{},
		identityDisplayName:      "tillsyn-user",
		identityActorID:          "tillsyn-user",
		identityDefaultActorType: string(domain.ActorTypeUser),
		bootstrapActorIndex:      0,
		bootstrapRoots:           []string{},
	}
	if cwd, err := os.Getwd(); err == nil {
		m.defaultRootDir = cwd
	} else {
		m.defaultRootDir = "."
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&m)
		}
	}
	return m
}

// scheduleAutoRefreshTickCmd schedules one refresh tick when auto-refresh is enabled.
func (m *Model) scheduleAutoRefreshTickCmd() tea.Cmd {
	if m.autoRefreshInterval <= 0 || m.autoRefreshArmed {
		return nil
	}
	m.autoRefreshArmed = true
	return tea.Tick(m.autoRefreshInterval, func(time.Time) tea.Msg {
		return autoRefreshTickMsg{}
	})
}

// loadDataForAutoRefreshCmd fetches board data in the background and wraps the result.
func (m Model) loadDataForAutoRefreshCmd() tea.Cmd {
	return func() tea.Msg {
		msg := m.loadData()
		loaded, ok := msg.(loadedMsg)
		if !ok {
			return autoRefreshLoadedMsg{err: fmt.Errorf("auto refresh: unexpected message type %T", msg)}
		}
		return autoRefreshLoadedMsg{data: loaded}
	}
}

// shouldAutoRefresh reports whether auto-refresh can run without disrupting active input flows.
func (m Model) shouldAutoRefresh() bool {
	switch m.mode {
	case modeNone, modeTaskInfo, modeActivityLog:
		return true
	default:
		return false
	}
}

// applyLoadedMsg applies a loaded message and returns any follow-up command.
func (m *Model) applyLoadedMsg(msg loadedMsg) tea.Cmd {
	if msg.err != nil {
		m.err = msg.err
		return nil
	}
	m.err = nil
	m.projects = msg.projects
	m.selectedProject = msg.selectedProject
	m.columns = msg.columns
	m.tasks = msg.tasks
	if msg.activityEntries != nil {
		m.activityLog = append([]activityEntry(nil), msg.activityEntries...)
	}
	m.dependencyRollup = msg.rollup
	m.warnings = buildScopeWarnings(msg.attentionItemsCount, msg.attentionUserActionCount)
	if len(m.projects) == 0 {
		m.selectedProject = 0
		m.selectedColumn = 0
		m.selectedTask = 0
		m.projectPickerIndex = 0
		m.columns = nil
		m.tasks = nil
		m.activityLog = []activityEntry{}
		if m.startupBootstrapRequired {
			if m.mode != modeBootstrapSettings && m.mode != modeAddProject && m.mode != modeEditProject {
				return m.startBootstrapSettingsMode(true)
			}
			return nil
		}
		if m.mode != modeAddProject && m.mode != modeEditProject {
			m.mode = modeProjectPicker
			m.status = "project picker"
		}
		m.launchPicker = false
		return nil
	}
	if m.pendingProjectID != "" {
		for idx, project := range m.projects {
			if project.ID == m.pendingProjectID {
				m.selectedProject = idx
				break
			}
		}
		m.pendingProjectID = ""
	}
	if m.projectionRootTaskID != "" {
		if _, ok := m.taskByID(m.projectionRootTaskID); !ok {
			m.projectionRootTaskID = ""
			m.status = "focus cleared (parent not found)"
		}
	}
	m.clampSelections()
	m.retainSelectionForLoadedTasks()
	m.normalizePanelFocus()
	if m.pendingFocusTaskID != "" {
		m.focusTaskByID(m.pendingFocusTaskID)
		m.pendingFocusTaskID = ""
	}
	if pendingJump := strings.TrimSpace(m.pendingActivityJumpTask); pendingJump != "" {
		if _, ok := m.taskByID(pendingJump); ok {
			m.prepareActivityJumpContext(pendingJump)
			if m.focusTaskByID(pendingJump) {
				m.status = "jumped to activity node"
			} else {
				m.status = "activity node unavailable (possibly hard-deleted)"
			}
		} else {
			m.status = "activity node unavailable (possibly hard-deleted)"
		}
		m.pendingActivityJumpTask = ""
	}
	if m.startupBootstrapRequired {
		if m.mode != modeBootstrapSettings {
			return m.startBootstrapSettingsMode(true)
		}
		return nil
	}
	if m.launchPicker && m.mode == modeNone {
		m.mode = modeProjectPicker
		m.projectPickerIndex = m.selectedProject
		m.status = "project picker"
		m.launchPicker = false
		return nil
	}
	m.launchPicker = false
	if m.status == "" || m.status == "loading..." {
		m.status = "ready"
	}
	return nil
}

// Init handles init.
func (m Model) Init() tea.Cmd {
	return m.loadData
}

// Update updates state for the requested operation.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.width = msg.Width
		m.height = msg.Height
		m.normalizePanelFocus()
		return m, nil

	case loadedMsg:
		m.autoRefreshInFlight = false
		if cmd := m.applyLoadedMsg(msg); cmd != nil {
			return m, cmd
		}
		return m, m.scheduleAutoRefreshTickCmd()

	case autoRefreshTickMsg:
		m.autoRefreshArmed = false
		if m.autoRefreshInterval <= 0 {
			return m, nil
		}
		if m.autoRefreshInFlight || !m.shouldAutoRefresh() {
			return m, m.scheduleAutoRefreshTickCmd()
		}
		m.autoRefreshInFlight = true
		return m, m.loadDataForAutoRefreshCmd()

	case autoRefreshLoadedMsg:
		m.autoRefreshInFlight = false
		if msg.err != nil {
			m.status = "auto refresh failed: " + msg.err.Error()
			return m, m.scheduleAutoRefreshTickCmd()
		}
		if msg.data.err != nil {
			m.status = "auto refresh failed: " + msg.data.err.Error()
			return m, m.scheduleAutoRefreshTickCmd()
		}
		if cmd := m.applyLoadedMsg(msg.data); cmd != nil {
			return m, cmd
		}
		return m, m.scheduleAutoRefreshTickCmd()

	case resourcePickerLoadedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		m.resourcePickerRoot = msg.root
		m.resourcePickerDir = msg.current
		m.resourcePickerItems = msg.entries
		m.resourcePickerIndex = 0
		if len(m.resourcePickerItems) > 1 && m.resourcePickerItems[0].Name == ".." {
			m.resourcePickerIndex = 1
		}
		return m, nil

	case actionMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		if msg.status != "" {
			m.status = msg.status
		}
		if msg.projectID != "" {
			m.pendingProjectID = msg.projectID
		}
		if msg.projectRootSlug != "" {
			if m.projectRoots == nil {
				m.projectRoots = map[string]string{}
			}
			if strings.TrimSpace(msg.projectRootPath) == "" {
				delete(m.projectRoots, msg.projectRootSlug)
			} else {
				m.projectRoots[msg.projectRootSlug] = strings.TrimSpace(msg.projectRootPath)
			}
		}
		if msg.focusTaskID != "" {
			m.pendingFocusTaskID = msg.focusTaskID
		}
		if msg.clearSelect {
			m.clearSelection()
		}
		if len(msg.clearTaskIDs) > 0 {
			m.unselectTasks(msg.clearTaskIDs)
		}
		if msg.historyPush != nil {
			m.pushUndoHistory(*msg.historyPush)
		}
		if msg.historyUndo != nil {
			m.applyUndoTransition(*msg.historyUndo)
		}
		if msg.historyRedo != nil {
			m.applyRedoTransition(*msg.historyRedo)
		}
		if msg.activityItem != nil {
			m.appendActivity(*msg.activityItem)
		}
		if msg.reload {
			return m, m.loadData
		}
		return m, nil

	case searchResultsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.searchMatches = msg.matches
		m.searchResultIndex = clamp(m.searchResultIndex, 0, len(m.searchMatches)-1)
		if len(m.searchMatches) > 0 {
			m.mode = modeSearchResults
			m.status = fmt.Sprintf("%d matches", len(m.searchMatches))
		} else {
			m.mode = modeNone
			m.status = "no matches"
		}
		return m, nil

	case dependencyMatchesMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.dependencyMatches = msg.candidates
		m.dependencyIndex = clamp(m.dependencyIndex, 0, len(m.dependencyMatches)-1)
		return m, nil

	case activityLogLoadedMsg:
		if msg.err != nil {
			// Keep the app usable when persisted activity fetch fails; fall back to current in-memory log.
			if m.mode == modeActivityLog {
				m.status = "activity log unavailable: " + msg.err.Error()
			}
			return m, nil
		}
		m.activityLog = append([]activityEntry(nil), msg.entries...)
		if m.mode == modeActivityLog {
			m.status = "activity log"
		}
		return m, nil

	case configReloadedMsg:
		if msg.err != nil {
			m.status = "reload config failed: " + msg.err.Error()
			return m, nil
		}
		m.applyRuntimeConfig(msg.config)
		m.status = "config reloaded"
		return m, m.loadData

	case projectRootSavedMsg:
		if msg.err != nil {
			m.status = "save root failed: " + msg.err.Error()
			return m, nil
		}
		if m.projectRoots == nil {
			m.projectRoots = map[string]string{}
		}
		if msg.rootPath == "" {
			delete(m.projectRoots, msg.projectSlug)
			m.status = "project root cleared"
			return m, nil
		}
		m.projectRoots[msg.projectSlug] = msg.rootPath
		m.status = "project root saved"
		return m, nil

	case bootstrapSettingsSavedMsg:
		if msg.err != nil {
			m.status = "save bootstrap failed: " + msg.err.Error()
			return m, nil
		}
		m.applyBootstrapConfig(msg.config)
		m.startupBootstrapRequired = false
		m.bootstrapMandatory = false
		m.mode = modeNone
		m.bootstrapDisplayInput.Blur()
		if m.launchPicker {
			m.mode = modeProjectPicker
			if len(m.projects) == 0 {
				m.projectPickerIndex = 0
			} else {
				m.projectPickerIndex = clamp(m.selectedProject, 0, len(m.projects)-1)
			}
			m.status = "project picker"
			m.launchPicker = false
			return m, nil
		}
		m.status = "bootstrap settings saved"
		return m, nil

	case threadLoadedMsg:
		if !sameCommentTarget(m.threadTarget, msg.target) {
			return m, nil
		}
		if msg.err != nil {
			m.status = "thread load failed: " + msg.err.Error()
			return m, nil
		}
		m.threadComments = append([]domain.Comment(nil), msg.comments...)
		m.threadScroll = 0
		m.status = "thread loaded"
		return m, nil

	case threadCommentCreatedMsg:
		if !sameCommentTarget(m.threadTarget, msg.target) {
			return m, nil
		}
		if msg.err != nil {
			if strings.TrimSpace(msg.body) != "" {
				m.threadInput.SetValue(msg.body)
				m.threadInput.CursorEnd()
			}
			m.threadPendingCommentBody = ""
			m.status = "post comment failed: " + msg.err.Error()
			return m, nil
		}
		m.threadPendingCommentBody = ""
		m.threadComments = append(m.threadComments, msg.value)
		m.status = "comment posted"
		return m, nil

	case tea.KeyPressMsg:
		if m.mode != modeNone {
			return m.handleInputModeKey(msg)
		}
		return m.handleNormalModeKey(msg)

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	default:
		return m, nil
	}
}

// View handles view.
func (m Model) View() tea.View {
	if m.err != nil {
		v := tea.NewView("error: " + m.err.Error() + "\n\npress r to retry • q quit\n")
		v.MouseMode = m.activeMouseMode()
		v.AltScreen = true
		return v
	}
	if !m.ready {
		v := tea.NewView("loading...")
		v.MouseMode = m.activeMouseMode()
		v.AltScreen = true
		return v
	}
	if m.mode == modeThread {
		return m.renderThreadModeView()
	}
	if len(m.projects) == 0 {
		accent := lipgloss.Color("62")
		muted := lipgloss.Color("241")
		dim := lipgloss.Color("239")
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
		helpStyle := lipgloss.NewStyle().Foreground(muted)
		statusStyle := lipgloss.NewStyle().Foreground(dim)
		sections := []string{
			titleStyle.Render("tillsyn"),
			"",
			"No projects yet.",
			"Press N to create your first project.",
			"Press q to quit.",
		}
		if strings.TrimSpace(m.status) != "" && m.status != "ready" {
			sections = append(sections, "", statusStyle.Render(m.status))
		}
		content := strings.Join(sections, "\n")
		innerWidth := max(0, m.width-2*tuiOuterHorizontalPadding)
		if innerWidth > 0 {
			content = lipgloss.NewStyle().
				Width(innerWidth).
				PaddingLeft(tuiOuterHorizontalPadding).
				PaddingRight(tuiOuterHorizontalPadding).
				Render(content)
		}
		helpBubble := m.help
		helpBubble.ShowAll = false
		helpBubble.SetWidth(innerWidth)
		helpLine := lipgloss.NewStyle().
			Foreground(muted).
			BorderTop(true).
			BorderForeground(dim).
			Width(innerWidth).
			Render(helpBubble.View(m.keys))
		if tuiOuterHorizontalPadding > 0 {
			helpLine = lipgloss.NewStyle().
				PaddingLeft(tuiOuterHorizontalPadding).
				PaddingRight(tuiOuterHorizontalPadding).
				Render(helpLine)
		}
		contentHeight := lipgloss.Height(content)
		if m.height > 0 {
			helpHeight := lipgloss.Height(helpLine)
			contentHeight = max(0, m.height-helpHeight)
			content = fitLines(content, contentHeight)
		}
		fullContent := content + "\n" + helpLine
		if overlay := m.renderModeOverlay(accent, muted, dim, helpStyle, m.width-8); overlay != "" {
			overlayHeight := lipgloss.Height(fullContent)
			if m.height > 0 {
				overlayHeight = m.height
			}
			fullContent = overlayOnContent(fullContent, overlay, max(1, m.width), max(1, overlayHeight))
		}
		v := tea.NewView(fullContent)
		v.MouseMode = m.activeMouseMode()
		v.AltScreen = true
		return v
	}

	project := m.projects[clamp(m.selectedProject, 0, len(m.projects)-1)]
	accent := projectAccentColor(project)
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")

	helpStyle := lipgloss.NewStyle().Foreground(muted)
	statusStyle := lipgloss.NewStyle().Foreground(dim)
	taskByID := m.tasksByID()
	attentionItems, attentionTotal, attentionBlocked, attentionTop := m.scopeAttentionSummary(taskByID)

	layoutWidth := max(0, m.width-2*tuiOuterHorizontalPadding)
	headerAccent := lipgloss.Color("62")
	header := headerMarkStyle().
		BorderForeground(headerAccent).
		Foreground(lipgloss.Color("252")).
		Render(headerMarkText)
	headerRule := lipgloss.NewStyle().
		Foreground(headerAccent).
		Render(strings.Repeat("─", max(8, layoutWidth)))
	noticesWidth := m.noticesPanelWidth(layoutWidth)
	boardWidth := m.boardWidthFor(layoutWidth)

	renderMainArea := func(boardWidth, noticesWidth int) string {
		columnViews := make([]string, 0, len(m.columns))
		boardPanelFocused := !m.noticesFocused || noticesWidth <= 0
		colWidth := m.columnWidthFor(boardWidth)
		extraBoardWidthPerColumn := 0
		extraBoardWidthRemainder := 0
		if len(m.columns) > 0 {
			interColumnGaps := max(0, len(m.columns)-1) * boardColumnGapWidth
			usedBoardWidth := len(m.columns)*renderedBoardColumnWidth(colWidth) + interColumnGaps
			if extra := max(0, boardWidth-usedBoardWidth); extra > 0 {
				extraBoardWidthPerColumn = extra / len(m.columns)
				extraBoardWidthRemainder = extra % len(m.columns)
			}
		}
		colHeight := m.columnHeight()
		baseColStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dim).
			Padding(1, 2)
		selColStyle := baseColStyle.Copy().BorderForeground(accent)
		normColStyle := baseColStyle.Copy()
		colTitle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		archivedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		highlight := m.selectedTaskHighlightColor()
		selectedTaskStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true)
		selectedMultiTaskStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true).Underline(true)
		// Multi-select should be indicated by marker stars only; avoid extra row background fill.
		multiSelectedTaskStyle := lipgloss.NewStyle()
		itemSubStyle := lipgloss.NewStyle().Foreground(muted)
		groupStyle := lipgloss.NewStyle().Bold(true).Foreground(muted)
		warningStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))

		for colIdx, column := range m.columns {
			colRenderWidth := colWidth + extraBoardWidthPerColumn
			if colIdx < extraBoardWidthRemainder {
				colRenderWidth++
			}
			colTasks := m.boardTasksForColumn(column.ID)
			parentByID := map[string]string{}
			for _, task := range colTasks {
				parentByID[task.ID] = task.ParentID
			}
			activeCount := 0
			for _, task := range colTasks {
				if task.ArchivedAt == nil {
					activeCount++
				}
			}

			colHeader := fmt.Sprintf("%s (%d)", column.Name, len(colTasks))
			if column.WIPLimit > 0 {
				colHeader = fmt.Sprintf("%s (%d/%d)", column.Name, activeCount, column.WIPLimit)
			}
			headerLines := []string{colTitle.Render(colHeader)}
			if m.showWIPWarnings && column.WIPLimit > 0 && activeCount > column.WIPLimit {
				headerLines = append(headerLines, warningStyle.Render(fmt.Sprintf("WIP limit exceeded: %d/%d", activeCount, column.WIPLimit)))
			}

			taskLines := make([]string, 0, max(1, len(colTasks)*3))
			selectedStart := -1
			selectedEnd := -1

			if len(colTasks) == 0 {
				taskLines = append(taskLines, archivedStyle.Render("(empty)"))
			} else {
				prevGroup := ""
				for taskIdx, task := range colTasks {
					if m.boardGroupBy != "none" {
						groupLabel := m.groupLabelForTask(task)
						if taskIdx == 0 || groupLabel != prevGroup {
							if taskIdx > 0 {
								taskLines = append(taskLines, "")
							}
							taskLines = append(taskLines, groupStyle.Render(groupLabel))
							prevGroup = groupLabel
						}
					}
					selected := colIdx == m.selectedColumn && taskIdx == m.selectedTask
					multiSelected := m.isTaskSelected(task.ID)

					prefix := "   "
					switch {
					case selected && multiSelected:
						prefix = "│* "
					case selected:
						prefix = "│  "
					case multiSelected:
						prefix = " * "
					}
					depth := taskDepth(task.ID, parentByID, 0)
					indent := strings.Repeat("  ", min(depth, 4))
					attentionCount := m.taskAttentionCount(task, taskByID)
					attentionSuffix := ""
					if attentionCount > 0 {
						attentionSuffix = fmt.Sprintf(" !%d", attentionCount)
					}
					titleWidth := max(1, colRenderWidth-(10+2*min(depth, 4))-utf8.RuneCountInString(attentionSuffix))
					title := prefix + indent + truncate(task.Title, titleWidth) + attentionSuffix
					sub := m.taskListSecondary(task)
					if sub != "" {
						sub = indent + truncate(sub, max(1, colRenderWidth-(10+2*min(depth, 4))))
					}
					if task.ArchivedAt != nil {
						title = archivedStyle.Render(title)
						if sub != "" {
							sub = archivedStyle.Render(sub)
						}
					} else {
						switch {
						case selected && multiSelected:
							title = selectedMultiTaskStyle.Render(title)
						case selected:
							title = selectedTaskStyle.Render(title)
						case multiSelected:
							title = multiSelectedTaskStyle.Render(title)
						}
					}

					rowStart := len(taskLines)
					taskLines = append(taskLines, title)
					if sub != "" {
						// Keep selection/focus markers on the title row only to avoid duplicate stars/cursor bars.
						subPrefix := "   "
						taskLines = append(taskLines, subPrefix+itemSubStyle.Render(sub))
					}
					if taskIdx < len(colTasks)-1 {
						taskLines = append(taskLines, "")
					}
					if selected {
						selectedStart = rowStart
						selectedEnd = len(taskLines) - 1
					}
				}
			}

			innerHeight := max(1, colHeight-4)
			taskWindowHeight := max(1, innerHeight-len(headerLines))
			scrollTop := 0
			if colIdx == m.selectedColumn && selectedStart >= 0 {
				if selectedEnd >= scrollTop+taskWindowHeight {
					scrollTop = selectedEnd - taskWindowHeight + 1
				}
				if selectedStart < scrollTop {
					scrollTop = selectedStart
				}
			}
			maxScrollTop := max(0, len(taskLines)-taskWindowHeight)
			scrollTop = clamp(scrollTop, 0, maxScrollTop)
			if len(taskLines) > taskWindowHeight {
				taskLines = taskLines[scrollTop : scrollTop+taskWindowHeight]
			}
			if len(taskLines) < taskWindowHeight {
				taskLines = append(taskLines, make([]string, taskWindowHeight-len(taskLines))...)
			}

			lines := append(append([]string{}, headerLines...), taskLines...)
			content := fitLines(strings.Join(lines, "\n"), innerHeight)
			colStyle := normColStyle.Copy().Width(colRenderWidth)
			if colIdx == m.selectedColumn && boardPanelFocused {
				colStyle = selColStyle.Copy().Width(colRenderWidth)
			}
			// Keep gaps only between columns; avoid trailing right gap after the last column.
			if colIdx < len(m.columns)-1 && boardColumnGapWidth > 0 {
				colStyle = colStyle.Copy().MarginRight(boardColumnGapWidth)
			}
			columnViews = append(columnViews, colStyle.Render(content))
		}

		body := lipgloss.JoinHorizontal(lipgloss.Top, columnViews...)
		mainArea := body
		if noticesWidth > 0 {
			noticesFocused := m.noticesFocused && m.isNoticesPanelVisible()
			overviewPanel := m.renderOverviewPanel(
				project,
				accent,
				muted,
				dim,
				noticesWidth,
				attentionItems,
				attentionTotal,
				attentionBlocked,
				attentionTop,
				noticesFocused,
			)
			if noticesPanelGapWidth > 0 {
				overviewPanel = lipgloss.NewStyle().MarginLeft(noticesPanelGapWidth).Render(overviewPanel)
			}
			mainArea = lipgloss.JoinHorizontal(lipgloss.Top, body, overviewPanel)
		}
		return mainArea
	}

	mainArea := renderMainArea(boardWidth, noticesWidth)
	if layoutWidth > 0 && len(m.columns) > 0 {
		for attempt := 0; attempt < 4; attempt++ {
			delta := layoutWidth - lipgloss.Width(mainArea)
			if delta == 0 {
				break
			}
			nextBoardWidth := max(minimumColumnWidth, boardWidth+delta)
			if nextBoardWidth == boardWidth {
				break
			}
			boardWidth = nextBoardWidth
			mainArea = renderMainArea(boardWidth, noticesWidth)
		}
	}

	overlay := m.renderModeOverlay(accent, muted, dim, helpStyle, m.width-8)
	if m.help.ShowAll {
		overlay = m.renderHelpOverlay(accent, muted, dim, helpStyle, m.width-8)
	}
	infoLine := m.renderInfoLine(project, muted)

	sections := []string{header, headerRule}
	if path, _ := m.projectionPathWithProject(project.Name); path != "" {
		sections = append(sections, statusStyle.Render("path: "+truncate(path, max(24, m.width-6))), "")
	}
	sections = append(sections, mainArea)
	if infoLine != "" {
		sections = append(sections, infoLine)
	}
	if attentionTotal > 0 {
		sections = append(sections, statusStyle.Render(fmt.Sprintf("attention scope: %d items • unresolved %d • blocked %d", attentionItems, attentionTotal, attentionBlocked)))
		if len(attentionTop) > 0 {
			sections = append(sections, statusStyle.Render("attention panel: "+strings.Join(attentionTop, " • ")))
		}
	}
	sections = append(sections, statusStyle.Render(m.dependencyRollupSummary()))
	if strings.TrimSpace(m.projectionRootTaskID) != "" {
		sections = append(sections, statusStyle.Render(fmt.Sprintf("subtree focus active • %s full board", m.keys.clearFocus.Help().Key)))
	}
	if count := len(m.selectedTaskIDs); count > 0 {
		sections = append(sections, statusStyle.Render(fmt.Sprintf("%d tasks selected • %s toggle • esc clear", count, m.keys.multiSelect.Help().Key)))
	}
	if m.showDueSummary {
		overdue, dueSoon := m.dueCounts(time.Now().UTC())
		sections = append(sections, statusStyle.Render(fmt.Sprintf("%d overdue * %d due soon", overdue, dueSoon)))
	}
	if strings.TrimSpace(m.status) != "" && m.status != "ready" {
		sections = append(sections, statusStyle.Render(m.status))
	}
	content := strings.Join(sections, "\n")
	innerWidth := layoutWidth
	if innerWidth > 0 {
		content = lipgloss.NewStyle().
			PaddingLeft(tuiOuterHorizontalPadding).
			PaddingRight(tuiOuterHorizontalPadding).
			Render(content)
	}

	helpBubble := m.help
	helpBubble.ShowAll = false
	helpBubble.SetWidth(innerWidth)
	helpLine := lipgloss.NewStyle().
		Foreground(muted).
		BorderTop(true).
		BorderForeground(dim).
		Width(innerWidth).
		Render(helpBubble.View(m.keys))
	if tuiOuterHorizontalPadding > 0 {
		helpLine = lipgloss.NewStyle().
			PaddingLeft(tuiOuterHorizontalPadding).
			PaddingRight(tuiOuterHorizontalPadding).
			Render(helpLine)
	}

	contentHeight := lipgloss.Height(content)
	if m.height > 0 {
		helpHeight := lipgloss.Height(helpLine)
		contentHeight = max(0, m.height-helpHeight)
		content = fitLines(content, contentHeight)
	}

	fullContent := content + "\n" + helpLine
	if overlay != "" {
		overlayHeight := lipgloss.Height(fullContent)
		if m.height > 0 {
			overlayHeight = m.height
		}
		fullContent = overlayOnContent(fullContent, overlay, max(1, m.width), max(1, overlayHeight))
	}

	view := tea.NewView(fullContent)
	view.MouseMode = m.activeMouseMode()
	view.AltScreen = true
	return view
}

// activeMouseMode returns the current mouse mode, including clipboard-friendly selection mode.
func (m Model) activeMouseMode() tea.MouseMode {
	if m.mouseSelectionMode {
		return tea.MouseModeNone
	}
	return tea.MouseModeCellMotion
}

// loadData loads required data for the current operation.
func (m Model) loadData() tea.Msg {
	projects, err := m.svc.ListProjects(context.Background(), m.showArchivedProjects)
	if err != nil {
		return loadedMsg{err: err}
	}
	if len(projects) == 0 {
		return loadedMsg{projects: projects}
	}

	projectIdx := clamp(m.selectedProject, 0, len(projects)-1)
	if pendingProjectID := strings.TrimSpace(m.pendingProjectID); pendingProjectID != "" {
		for idx, project := range projects {
			if project.ID == pendingProjectID {
				projectIdx = idx
				break
			}
		}
	}
	projectID := projects[projectIdx].ID
	columns, err := m.svc.ListColumns(context.Background(), projectID, false)
	if err != nil {
		return loadedMsg{err: err}
	}

	var tasks []domain.Task
	searchFilterActive := m.searchApplied
	if searchFilterActive {
		matches, searchErr := m.svc.SearchTaskMatches(context.Background(), app.SearchTasksFilter{
			ProjectID:       projectID,
			Query:           m.searchQuery,
			CrossProject:    m.searchCrossProject,
			IncludeArchived: m.searchIncludeArchived,
			States:          append([]string(nil), m.searchStates...),
		})
		if searchErr != nil {
			return loadedMsg{err: searchErr}
		}
		matches = m.filterTaskMatchesBySearchLevels(matches)
		tasks = make([]domain.Task, 0, len(matches))
		for _, match := range matches {
			if match.Project.ID == projectID {
				tasks = append(tasks, match.Task)
			}
		}
	} else {
		tasks, err = m.svc.ListTasks(context.Background(), projectID, m.showArchived)
	}
	if err != nil {
		return loadedMsg{err: err}
	}
	rollup, err := m.svc.GetProjectDependencyRollup(context.Background(), projectID)
	if err != nil {
		return loadedMsg{err: err}
	}
	activityEntries := []activityEntry{}
	events, activityErr := m.svc.ListProjectChangeEvents(context.Background(), projectID, activityLogMaxItems)
	if activityErr == nil {
		activityEntries = mapChangeEventsToActivityEntries(events)
	}

	attentionItems, attentionErr := m.svc.ListAttentionItems(context.Background(), app.ListAttentionItemsInput{
		Level: domain.LevelTupleInput{
			ProjectID: projectID,
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   projectID,
		},
		UnresolvedOnly: true,
		Limit:          256,
	})
	if attentionErr != nil {
		return loadedMsg{err: attentionErr}
	}
	requiresUserAction := 0
	for _, item := range attentionItems {
		if item.RequiresUserAction {
			requiresUserAction++
		}
	}

	return loadedMsg{
		projects:                 projects,
		selectedProject:          projectIdx,
		columns:                  columns,
		tasks:                    tasks,
		activityEntries:          activityEntries,
		rollup:                   rollup,
		attentionItemsCount:      len(attentionItems),
		attentionUserActionCount: requiresUserAction,
	}
}

// loadSearchMatches loads required data for the current operation.
func (m Model) loadSearchMatches() tea.Msg {
	projectID, _ := m.currentProjectID()
	matches, err := m.svc.SearchTaskMatches(context.Background(), app.SearchTasksFilter{
		ProjectID:       projectID,
		Query:           m.searchQuery,
		CrossProject:    m.searchCrossProject,
		IncludeArchived: m.searchIncludeArchived,
		States:          append([]string(nil), m.searchStates...),
	})
	if err != nil {
		return searchResultsMsg{err: err}
	}
	matches = m.filterTaskMatchesBySearchLevels(matches)
	return searchResultsMsg{matches: matches}
}

// loadActivityLog loads persisted project activity entries for modal rendering.
func (m Model) loadActivityLog() tea.Msg {
	projectID, ok := m.currentProjectID()
	if !ok {
		return activityLogLoadedMsg{entries: nil}
	}
	events, err := m.svc.ListProjectChangeEvents(context.Background(), projectID, activityLogMaxItems)
	if err != nil {
		return activityLogLoadedMsg{err: err}
	}
	return activityLogLoadedMsg{entries: mapChangeEventsToActivityEntries(events)}
}

// openActivityLog enters activity-log mode and triggers persisted activity fetch.
func (m *Model) openActivityLog() tea.Cmd {
	m.mode = modeActivityLog
	m.status = "activity log"
	return m.loadActivityLog
}

// canJumpToActivityNode reports whether one activity entry references a concrete task node.
func canJumpToActivityNode(entry activityEntry) bool {
	return strings.TrimSpace(entry.WorkItemID) != ""
}

// prepareActivityJumpContext adjusts focus scope so jump targets can be selected in board view.
func (m *Model) prepareActivityJumpContext(taskID string) bool {
	task, ok := m.taskByID(strings.TrimSpace(taskID))
	if !ok {
		return false
	}
	parentID := strings.TrimSpace(task.ParentID)
	if parentID == "" {
		m.projectionRootTaskID = ""
		m.clampSelections()
		return true
	}
	if _, ok := m.taskByID(parentID); !ok {
		m.projectionRootTaskID = ""
		m.clampSelections()
		return true
	}
	return m.activateSubtreeFocus(parentID)
}

// jumpToActivityNode navigates to the task referenced by the current activity-detail entry when available.
func (m Model) jumpToActivityNode() (tea.Model, tea.Cmd) {
	workItemID := strings.TrimSpace(m.activityInfoItem.WorkItemID)
	if workItemID == "" {
		m.status = "activity event has no node reference"
		return m, nil
	}
	if _, ok := m.taskByID(workItemID); ok {
		m.mode = modeNone
		m.noticesFocused = false
		m.prepareActivityJumpContext(workItemID)
		if m.focusTaskByID(workItemID) {
			m.status = "jumped to activity node"
		} else {
			m.status = "activity node unavailable (possibly hard-deleted)"
		}
		return m, nil
	}
	if !m.showArchived {
		m.mode = modeNone
		m.noticesFocused = false
		m.showArchived = true
		m.pendingFocusTaskID = workItemID
		m.pendingActivityJumpTask = workItemID
		m.status = "loading activity node..."
		return m, m.loadData
	}
	m.status = "activity node unavailable (possibly hard-deleted)"
	return m, nil
}

// mapChangeEventsToActivityEntries converts newest-first persisted events into modal rows.
func mapChangeEventsToActivityEntries(events []domain.ChangeEvent) []activityEntry {
	if len(events) == 0 {
		return []activityEntry{}
	}
	entries := make([]activityEntry, 0, len(events))
	// Repository events are newest-first; modal rendering expects chronological order.
	for idx := len(events) - 1; idx >= 0; idx-- {
		entries = append(entries, mapChangeEventToActivityEntry(events[idx]))
	}
	if len(entries) > activityLogMaxItems {
		entries = append([]activityEntry(nil), entries[len(entries)-activityLogMaxItems:]...)
	}
	return entries
}

// mapChangeEventToActivityEntry derives a compact activity row from one persisted event.
func mapChangeEventToActivityEntry(event domain.ChangeEvent) activityEntry {
	operationVerb := "update"
	switch event.Operation {
	case domain.ChangeOperationCreate:
		operationVerb = "create"
	case domain.ChangeOperationUpdate:
		operationVerb = "update"
	case domain.ChangeOperationMove:
		operationVerb = "move"
	case domain.ChangeOperationArchive:
		operationVerb = "archive"
	case domain.ChangeOperationRestore:
		operationVerb = "restore"
	case domain.ChangeOperationDelete:
		operationVerb = "delete"
	}
	summary := fmt.Sprintf("%s %s", operationVerb, activityEntityLabel(event.Metadata))
	target := strings.TrimSpace(event.Metadata["title"])
	if target == "" {
		target = strings.TrimSpace(event.WorkItemID)
	}
	if target == "" {
		target = "-"
	}
	actorID := strings.TrimSpace(event.ActorID)
	if actorID == "" {
		actorID = "unknown"
	}
	actorName := strings.TrimSpace(event.ActorName)
	actorType := domain.ActorType(strings.TrimSpace(strings.ToLower(string(event.ActorType))))
	if actorType == "" {
		actorType = domain.ActorTypeUser
	}
	return activityEntry{
		At:         event.OccurredAt.UTC(),
		Summary:    summary,
		Target:     target,
		EventID:    event.ID,
		WorkItemID: strings.TrimSpace(event.WorkItemID),
		Operation:  event.Operation,
		ActorID:    actorID,
		ActorName:  actorName,
		ActorType:  actorType,
		Metadata:   copyActivityMetadata(event.Metadata),
	}
}

// activityEntityLabel derives one readable entity label from change-event metadata.
func activityEntityLabel(metadata map[string]string) string {
	if len(metadata) == 0 {
		return "task"
	}
	label := strings.TrimSpace(strings.ToLower(metadata["item_scope"]))
	if label == "" {
		label = strings.TrimSpace(strings.ToLower(metadata["scope"]))
	}
	if label == "" {
		label = strings.TrimSpace(strings.ToLower(metadata["item_kind"]))
	}
	switch label {
	case "project", "branch", "phase", "subphase", "task", "subtask", "decision", "note":
		return label
	default:
		return "task"
	}
}

// copyActivityMetadata deep-copies change-event metadata for local activity rendering.
func copyActivityMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// newModalInput constructs modal input.
func newModalInput(prompt, placeholder, value string, limit int) textinput.Model {
	in := textinput.New()
	in.Prompt = prompt
	in.Placeholder = placeholder
	in.CharLimit = limit
	configureTextInputClipboardBindings(&in)
	if value != "" {
		in.SetValue(value)
	}
	return in
}

// configureTextInputClipboardBindings adds platform-friendly clipboard paste bindings for text inputs.
func configureTextInputClipboardBindings(in *textinput.Model) {
	if in == nil {
		return
	}
	in.KeyMap.Paste.SetKeys("ctrl+v", "meta+v", "super+v")
}

// startSearchMode starts search mode.
func (m *Model) startSearchMode() tea.Cmd {
	m.mode = modeSearch
	m.input = ""
	m.searchStates = canonicalSearchStates(m.searchStates)
	m.searchLevels = canonicalSearchLevels(m.searchLevels)
	m.searchInput.SetValue(m.searchQuery)
	m.searchInput.CursorEnd()
	m.searchFocus = 0
	m.searchStateCursor = 0
	m.searchLevelCursor = 0
	m.status = "search"
	return m.searchInput.Focus()
}

// startCommandPalette starts command palette.
func (m *Model) startCommandPalette() tea.Cmd {
	m.mode = modeCommandPalette
	m.commandInput.SetValue("")
	m.commandInput.CursorEnd()
	m.commandMatches = m.filteredCommandItems("")
	m.commandIndex = 0
	m.status = "command palette"
	return m.commandInput.Focus()
}

// startBootstrapSettingsMode opens the identity + global search-roots bootstrap/settings modal.
func (m *Model) startBootstrapSettingsMode(mandatory bool) tea.Cmd {
	m.mode = modeBootstrapSettings
	m.bootstrapMandatory = mandatory
	m.bootstrapDisplayInput.SetValue(strings.TrimSpace(m.identityDisplayName))
	m.bootstrapDisplayInput.CursorEnd()
	if mandatory {
		m.bootstrapActorIndex = bootstrapActorTypeIndex(string(domain.ActorTypeUser))
	} else {
		m.bootstrapActorIndex = bootstrapActorTypeIndex(m.identityDefaultActorType)
	}
	m.bootstrapRoots = append([]string(nil), normalizeSearchRoots(m.searchRoots)...)
	m.bootstrapRootIndex = clamp(m.bootstrapRootIndex, 0, len(m.bootstrapRoots)-1)
	m.status = "bootstrap settings"
	if mandatory {
		m.status = "startup setup required"
	}
	return m.focusBootstrapField(0)
}

// focusBootstrapField sets focus to one bootstrap/settings modal section.
func (m *Model) focusBootstrapField(idx int) tea.Cmd {
	const totalFields = 3
	idx = clamp(idx, 0, totalFields-1)
	m.bootstrapFocus = idx
	m.bootstrapDisplayInput.Blur()
	if idx == 0 {
		return m.bootstrapDisplayInput.Focus()
	}
	return nil
}

// bootstrapActorType returns the currently selected bootstrap actor type.
func (m Model) bootstrapActorType() string {
	if len(bootstrapActorTypes) == 0 {
		return string(domain.ActorTypeUser)
	}
	idx := clamp(m.bootstrapActorIndex, 0, len(bootstrapActorTypes)-1)
	return bootstrapActorTypes[idx]
}

// cycleBootstrapActor cycles bootstrap actor type selection by delta.
func (m *Model) cycleBootstrapActor(delta int) {
	m.bootstrapActorIndex = wrapIndex(m.bootstrapActorIndex, delta, len(bootstrapActorTypes))
}

// addBootstrapSearchRoot sets the single normalized bootstrap default path.
func (m *Model) addBootstrapSearchRoot(root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	root = filepath.Clean(root)
	if len(m.bootstrapRoots) == 1 && strings.EqualFold(m.bootstrapRoots[0], root) {
		return false
	}
	m.bootstrapRoots = []string{root}
	m.bootstrapRootIndex = 0
	return true
}

// removeSelectedBootstrapRoot removes the currently selected bootstrap root.
func (m *Model) removeSelectedBootstrapRoot() bool {
	if len(m.bootstrapRoots) == 0 {
		return false
	}
	idx := clamp(m.bootstrapRootIndex, 0, len(m.bootstrapRoots)-1)
	m.bootstrapRoots = append(m.bootstrapRoots[:idx], m.bootstrapRoots[idx+1:]...)
	m.bootstrapRootIndex = clamp(idx, 0, len(m.bootstrapRoots)-1)
	return true
}

// submitBootstrapSettings validates and persists bootstrap/settings values.
func (m Model) submitBootstrapSettings() (tea.Model, tea.Cmd) {
	displayName := strings.TrimSpace(m.bootstrapDisplayInput.Value())
	if displayName == "" {
		m.status = "display name is required"
		return m, nil
	}
	roots := normalizeSearchRoots(m.bootstrapRoots)
	if len(roots) == 0 {
		m.status = "default path is required"
		return m, nil
	}
	if m.saveBootstrap == nil {
		m.status = "save bootstrap failed: callback unavailable"
		return m, nil
	}
	actorType := m.bootstrapActorType()
	if m.bootstrapMandatory {
		actorType = string(domain.ActorTypeUser)
	}
	actorID := strings.TrimSpace(m.identityActorID)
	if actorID == "" {
		actorID = "tillsyn-user"
	}
	cfg := BootstrapConfig{
		ActorID:          actorID,
		DisplayName:      displayName,
		DefaultActorType: actorType,
		SearchRoots:      roots,
	}
	m.status = "saving bootstrap settings..."
	return m, m.saveBootstrapSettingsCmd(cfg)
}

// saveBootstrapSettingsCmd persists bootstrap/settings values through the callback surface.
func (m Model) saveBootstrapSettingsCmd(cfg BootstrapConfig) tea.Cmd {
	return func() tea.Msg {
		if err := m.saveBootstrap(cfg); err != nil {
			return bootstrapSettingsSavedMsg{err: err}
		}
		return bootstrapSettingsSavedMsg{config: cfg}
	}
}

// applyBootstrapConfig applies saved bootstrap settings to in-memory runtime state.
func (m *Model) applyBootstrapConfig(cfg BootstrapConfig) {
	if actorID := strings.TrimSpace(cfg.ActorID); actorID != "" {
		m.identityActorID = actorID
	}
	m.identityDisplayName = strings.TrimSpace(cfg.DisplayName)
	m.identityDefaultActorType = strings.TrimSpace(strings.ToLower(cfg.DefaultActorType))
	if m.identityDefaultActorType == "" {
		m.identityDefaultActorType = string(domain.ActorTypeUser)
	}
	if strings.TrimSpace(m.identityActorID) == "" {
		m.identityActorID = "tillsyn-user"
	}
	m.searchRoots = normalizeSearchRoots(cfg.SearchRoots)
	m.bootstrapRoots = append([]string(nil), m.searchRoots...)
	m.bootstrapRootIndex = clamp(m.bootstrapRootIndex, 0, len(m.bootstrapRoots)-1)
}

// startPathsRootsMode opens the modal used to edit one current-project root mapping.
func (m *Model) startPathsRootsMode() tea.Cmd {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return nil
	}
	slug := strings.TrimSpace(strings.ToLower(project.Slug))
	if slug == "" {
		m.status = "project slug is empty"
		return nil
	}
	m.mode = modePathsRoots
	m.pathsRootInput.SetValue(strings.TrimSpace(m.projectRoots[slug]))
	m.pathsRootInput.CursorEnd()
	m.status = "paths/roots"
	return m.pathsRootInput.Focus()
}

// startHighlightColorMode opens a modal for updating focused-row highlight color.
func (m *Model) startHighlightColorMode() tea.Cmd {
	m.mode = modeHighlightColor
	m.highlightColorInput.SetValue(strings.TrimSpace(m.highlightColor))
	m.highlightColorInput.CursorEnd()
	m.status = "highlight color"
	return m.highlightColorInput.Focus()
}

// startQuickActions starts quick actions.
func (m *Model) startQuickActions() tea.Cmd {
	m.mode = modeQuickActions
	actions := m.quickActions()
	m.quickActionIndex = 0
	for idx, action := range actions {
		if action.Enabled {
			m.quickActionIndex = idx
			break
		}
	}
	m.status = "quick actions"
	return nil
}

// startProjectForm starts project form.
func (m *Model) startProjectForm(project *domain.Project) tea.Cmd {
	m.projectFormFocus = 0
	m.projectFormInputs = []textinput.Model{
		newModalInput("", "project name", "", 120),
		newModalInput("", "short description", "", 240),
		newModalInput("", "owner/team", "", 120),
		newModalInput("", "icon / emoji", "", 64),
		newModalInput("", "accent color (e.g. 62)", "", 32),
		newModalInput("", "https://...", "", 200),
		newModalInput("", "csv tags", "", 200),
		newModalInput("", "project root path (optional)", "", 512),
	}
	m.editingProjectID = ""
	if project != nil {
		m.mode = modeEditProject
		m.status = "edit project"
		m.editingProjectID = project.ID
		m.projectFormInputs[projectFieldName].SetValue(project.Name)
		m.projectFormInputs[projectFieldDescription].SetValue(project.Description)
		m.projectFormInputs[projectFieldOwner].SetValue(project.Metadata.Owner)
		m.projectFormInputs[projectFieldIcon].SetValue(project.Metadata.Icon)
		m.projectFormInputs[projectFieldColor].SetValue(project.Metadata.Color)
		m.projectFormInputs[projectFieldHomepage].SetValue(project.Metadata.Homepage)
		if len(project.Metadata.Tags) > 0 {
			m.projectFormInputs[projectFieldTags].SetValue(strings.Join(project.Metadata.Tags, ","))
		}
		if slug := strings.TrimSpace(strings.ToLower(project.Slug)); slug != "" {
			m.projectFormInputs[projectFieldRootPath].SetValue(strings.TrimSpace(m.projectRoots[slug]))
		}
	} else {
		m.mode = modeAddProject
		m.status = "new project"
	}
	return m.focusProjectFormField(0)
}

// startTaskForm starts task form.
func (m *Model) startTaskForm(task *domain.Task) tea.Cmd {
	m.formFocus = 0
	m.priorityIdx = 1
	m.duePicker = 0
	m.pickerBack = modeNone
	m.input = ""
	m.taskFormParentID = ""
	m.taskFormKind = domain.WorkKindTask
	m.taskFormScope = domain.KindAppliesToTask
	m.taskFormResourceRefs = nil
	m.formInputs = []textinput.Model{
		newModalInput("", "task title (required)", "", 120),
		newModalInput("", "short description", "", 240),
		newModalInput("", "low | medium | high", "", 16),
		newModalInput("", "YYYY-MM-DD[THH:MM] or -", "", 32),
		newModalInput("", "csv labels", "", 160),
		newModalInput("", "csv task ids", "", 240),
		newModalInput("", "csv task ids", "", 240),
		newModalInput("", "why blocked? (optional)", "", 240),
	}
	labelsIdx := taskFieldLabels
	m.formInputs[labelsIdx].ShowSuggestions = true
	m.formInputs[taskFieldPriority].SetValue(string(priorityOptions[m.priorityIdx]))
	if task != nil {
		m.taskFormParentID = task.ParentID
		m.taskFormKind = task.Kind
		m.taskFormScope = task.Scope
		m.formInputs[taskFieldTitle].SetValue(task.Title)
		m.formInputs[taskFieldDescription].SetValue(task.Description)
		m.priorityIdx = priorityIndex(task.Priority)
		m.formInputs[taskFieldPriority].SetValue(string(priorityOptions[m.priorityIdx]))
		if task.DueAt != nil {
			m.formInputs[taskFieldDue].SetValue(formatDueValue(task.DueAt))
		}
		if len(task.Labels) > 0 {
			m.formInputs[taskFieldLabels].SetValue(strings.Join(task.Labels, ","))
		}
		if len(task.Metadata.DependsOn) > 0 {
			m.formInputs[taskFieldDependsOn].SetValue(strings.Join(task.Metadata.DependsOn, ","))
		}
		if len(task.Metadata.BlockedBy) > 0 {
			m.formInputs[taskFieldBlockedBy].SetValue(strings.Join(task.Metadata.BlockedBy, ","))
		}
		if blockedReason := strings.TrimSpace(task.Metadata.BlockedReason); blockedReason != "" {
			m.formInputs[taskFieldBlockedReason].SetValue(blockedReason)
		}
		m.taskFormResourceRefs = append([]domain.ResourceRef(nil), task.Metadata.ResourceRefs...)
		m.mode = modeEditTask
		m.editingTaskID = task.ID
		m.status = "edit task"
	} else {
		m.formInputs[taskFieldPriority].Placeholder = "medium"
		m.formInputs[taskFieldDue].Placeholder = "-"
		m.formInputs[taskFieldLabels].Placeholder = "-"
		m.mode = modeAddTask
		m.editingTaskID = ""
		m.status = "new task"
		m.taskFormParentID, m.taskFormKind, m.taskFormScope = m.newTaskDefaultsForActiveBoardScope()
	}
	m.refreshTaskFormLabelSuggestions()
	return m.focusTaskFormField(0)
}

// newTaskDefaultsForActiveBoardScope infers parent/kind/scope defaults from active focused scope.
func (m Model) newTaskDefaultsForActiveBoardScope() (string, domain.WorkKind, domain.KindAppliesTo) {
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	if rootID == "" {
		return "", domain.WorkKindTask, domain.KindAppliesToTask
	}
	root, ok := m.taskByID(rootID)
	if !ok {
		return "", domain.WorkKindTask, domain.KindAppliesToTask
	}
	levelByTaskID := m.searchLevelByTaskID([]domain.Task{root})
	level := strings.TrimSpace(levelByTaskID[root.ID])
	if level == "" {
		level = baseSearchLevelForTask(root)
	}
	switch level {
	case "task", "subtask":
		return root.ID, domain.WorkKindSubtask, domain.KindAppliesToSubtask
	default:
		return root.ID, domain.WorkKindTask, domain.KindAppliesToTask
	}
}

// startSubtaskForm opens the task form preconfigured for a child item.
func (m *Model) startSubtaskForm(parent domain.Task) tea.Cmd {
	cmd := m.startTaskForm(nil)
	m.taskFormParentID = parent.ID
	m.taskFormKind = domain.WorkKindSubtask
	m.taskFormScope = domain.KindAppliesToSubtask
	m.refreshTaskFormLabelSuggestions()
	m.status = "new subtask for " + parent.Title
	return cmd
}

// startBranchForm opens the task form preconfigured for a branch work item.
func (m *Model) startBranchForm(parent *domain.Task) tea.Cmd {
	cmd := m.startTaskForm(nil)
	m.taskFormKind = domain.WorkKind("branch")
	m.taskFormScope = domain.KindAppliesToBranch
	m.taskFormParentID = ""
	if parent != nil && strings.TrimSpace(parent.ID) != "" {
		m.taskFormParentID = parent.ID
	}
	if len(m.formInputs) > taskFieldTitle {
		m.formInputs[taskFieldTitle].Placeholder = "branch title (required)"
	}
	m.refreshTaskFormLabelSuggestions()
	m.status = "new branch"
	return cmd
}

// startPhaseForm opens the task form preconfigured for a phase/subphase work item.
func (m *Model) startPhaseForm(parent domain.Task, subphase bool) tea.Cmd {
	cmd := m.startTaskForm(nil)
	m.taskFormKind = domain.WorkKindPhase
	m.taskFormParentID = parent.ID
	if subphase {
		m.taskFormScope = domain.KindAppliesToSubphase
		if len(m.formInputs) > taskFieldTitle {
			m.formInputs[taskFieldTitle].Placeholder = "subphase title (required)"
		}
		m.status = "new subphase"
	} else {
		m.taskFormScope = domain.KindAppliesToPhase
		if len(m.formInputs) > taskFieldTitle {
			m.formInputs[taskFieldTitle].Placeholder = "phase title (required)"
		}
		m.status = "new phase"
	}
	m.refreshTaskFormLabelSuggestions()
	return cmd
}

// startSubtaskFormFromTaskForm opens a subtask form from create/edit task modal context.
func (m *Model) startSubtaskFormFromTaskForm() tea.Cmd {
	if m.mode == modeEditTask {
		taskID := strings.TrimSpace(m.editingTaskID)
		if taskID == "" {
			task, ok := m.selectedTaskInCurrentColumn()
			if !ok {
				m.status = "no task selected"
				return nil
			}
			taskID = task.ID
		}
		task, ok := m.taskByID(taskID)
		if !ok {
			m.status = "task not found"
			return nil
		}
		return m.startSubtaskForm(task)
	}
	parentID := strings.TrimSpace(m.taskFormParentID)
	if parentID == "" {
		m.status = "save task first to add subtask"
		return nil
	}
	parent, ok := m.taskByID(parentID)
	if !ok {
		m.status = "parent task not found"
		return nil
	}
	return m.startSubtaskForm(parent)
}

// focusTaskFormField focuses task form field.
func (m *Model) focusTaskFormField(idx int) tea.Cmd {
	if len(m.formInputs) == 0 {
		return nil
	}
	idx = clamp(idx, 0, len(m.formInputs)-1)
	m.formFocus = idx
	for i := range m.formInputs {
		m.formInputs[i].Blur()
	}
	if idx == 2 {
		return nil
	}
	return m.formInputs[idx].Focus()
}

// focusProjectFormField focuses project form field.
func (m *Model) focusProjectFormField(idx int) tea.Cmd {
	if len(m.projectFormInputs) == 0 {
		return nil
	}
	idx = clamp(idx, 0, len(m.projectFormInputs)-1)
	m.projectFormFocus = idx
	for i := range m.projectFormInputs {
		m.projectFormInputs[i].Blur()
	}
	return m.projectFormInputs[idx].Focus()
}

// startLabelsConfigForm opens a modal for editing global/project/branch/phase label defaults.
func (m *Model) startLabelsConfigForm() tea.Cmd {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return nil
	}
	slug := strings.TrimSpace(strings.ToLower(project.Slug))
	if slug == "" {
		m.status = "project slug is empty"
		return nil
	}
	m.labelsConfigSlug = slug
	m.labelsConfigBranchTaskID = ""
	m.labelsConfigPhaseTaskID = ""
	m.labelsConfigFocus = 0
	m.labelsConfigInputs = []textinput.Model{
		newModalInput("", "global labels csv", "", 240),
		newModalInput("", "project labels csv", "", 240),
		newModalInput("", "branch labels csv (optional)", "", 240),
		newModalInput("", "phase labels csv (optional)", "", 240),
	}
	if len(m.allowedLabelGlobal) > 0 {
		m.labelsConfigInputs[0].SetValue(strings.Join(m.allowedLabelGlobal, ","))
	}
	if labels := m.allowedLabelProject[slug]; len(labels) > 0 {
		m.labelsConfigInputs[1].SetValue(strings.Join(labels, ","))
	}
	if branch, ok := m.labelsConfigContextTask("branch"); ok {
		m.labelsConfigBranchTaskID = branch.ID
		if len(branch.Labels) > 0 {
			m.labelsConfigInputs[2].SetValue(strings.Join(branch.Labels, ","))
		}
	}
	if phase, ok := m.labelsConfigContextTask("phase", "subphase"); ok {
		m.labelsConfigPhaseTaskID = phase.ID
		if len(phase.Labels) > 0 {
			m.labelsConfigInputs[3].SetValue(strings.Join(phase.Labels, ","))
		}
	}
	m.mode = modeLabelsConfig
	m.status = "edit labels config"
	return m.focusLabelsConfigField(0)
}

// focusLabelsConfigField focuses one labels-config input.
func (m *Model) focusLabelsConfigField(idx int) tea.Cmd {
	if len(m.labelsConfigInputs) == 0 {
		return nil
	}
	idx = clamp(idx, 0, len(m.labelsConfigInputs)-1)
	m.labelsConfigFocus = idx
	for i := range m.labelsConfigInputs {
		m.labelsConfigInputs[i].Blur()
	}
	return m.labelsConfigInputs[idx].Focus()
}

// labelsConfigContextTask resolves the nearest selected task at one of the requested levels.
func (m Model) labelsConfigContextTask(levels ...string) (domain.Task, bool) {
	if len(levels) == 0 {
		return domain.Task{}, false
	}
	targetSet := map[string]struct{}{}
	for _, level := range levels {
		level = strings.TrimSpace(strings.ToLower(level))
		if level == "" {
			continue
		}
		targetSet[level] = struct{}{}
	}
	task, ok := m.selectedTaskForLabelInheritance()
	if !ok {
		return domain.Task{}, false
	}
	visited := map[string]struct{}{}
	current := task
	for strings.TrimSpace(current.ID) != "" {
		if _, seen := visited[current.ID]; seen {
			break
		}
		visited[current.ID] = struct{}{}
		if _, wanted := targetSet[baseSearchLevelForTask(current)]; wanted {
			return current, true
		}
		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, found := m.taskByID(parentID)
		if !found {
			break
		}
		current = parent
	}
	return domain.Task{}, false
}

// taskFormValues returns task form values.
func (m Model) taskFormValues() map[string]string {
	out := map[string]string{}
	for i, key := range taskFormFields {
		if i >= len(m.formInputs) {
			break
		}
		out[key] = strings.TrimSpace(m.formInputs[i].Value())
	}
	return out
}

// allowedLabelsForSelectedProject returns merged global + project-scoped allowed labels.
func (m Model) allowedLabelsForSelectedProject() []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	appendUnique := func(labels []string) {
		for _, raw := range labels {
			label := strings.TrimSpace(strings.ToLower(raw))
			if label == "" {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	appendUnique(m.allowedLabelGlobal)
	if project, ok := m.currentProject(); ok {
		appendUnique(m.allowedLabelProject[strings.TrimSpace(strings.ToLower(project.Slug))])
	}
	sort.Strings(out)
	return out
}

// projectFormFields stores a package-level helper value.
var projectFormFields = []string{"name", "description", "owner", "icon", "color", "homepage", "tags", "root_path"}

// projectFormValues returns project form values.
func (m Model) projectFormValues() map[string]string {
	out := map[string]string{}
	for idx, key := range projectFormFields {
		if idx >= len(m.projectFormInputs) {
			break
		}
		out[key] = strings.TrimSpace(m.projectFormInputs[idx].Value())
	}
	return out
}

// parseDueInput parses input into a normalized form.
func parseDueInput(raw string, current *time.Time) (*time.Time, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return current, nil
	}
	if text == "-" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		ts := parsed.UTC()
		return &ts, nil
	}

	localLayouts := []string{
		"2006-01-02",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
	}
	for _, layout := range localLayouts {
		parsed, err := time.ParseInLocation(layout, text, time.Local)
		if err != nil {
			continue
		}
		ts := parsed.UTC()
		return &ts, nil
	}
	return nil, fmt.Errorf("due date must be YYYY-MM-DD, YYYY-MM-DDTHH:MM, RFC3339, or -")
}

// dueWarning returns a warning message for due input values.
func dueWarning(raw string, now time.Time) string {
	parsed, err := parseDueInput(raw, nil)
	if err != nil || parsed == nil {
		return ""
	}
	if parsed.Before(now.UTC()) {
		return "warning: due datetime is in the past"
	}
	return ""
}

// formatDueValue formats due datetime values for compact UI display and editing.
func formatDueValue(dueAt *time.Time) string {
	if dueAt == nil {
		return "-"
	}
	due := dueAt.In(time.Local)
	if due.Hour() == 0 && due.Minute() == 0 {
		return due.Format("2006-01-02")
	}
	return due.Format("2006-01-02 15:04")
}

// parseLabelsInput parses input into a normalized form.
func parseLabelsInput(raw string, current []string) []string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return current
	}
	if text == "-" {
		return nil
	}
	rawLabels := strings.Split(text, ",")
	out := make([]string, 0, len(rawLabels))
	for _, label := range rawLabels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		out = append(out, label)
	}
	return out
}

// parseTaskRefIDsInput parses dependency reference ids from comma-separated task-id input.
func parseTaskRefIDsInput(raw string, current []string) []string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return append([]string(nil), current...)
	}
	if text == "-" {
		return nil
	}
	parts := strings.Split(text, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, id)
	}
	return out
}

// buildTaskMetadataFromForm overlays dependency/resource task metadata fields from form values.
func (m Model) buildTaskMetadataFromForm(vals map[string]string, current domain.TaskMetadata) domain.TaskMetadata {
	meta := current
	meta.DependsOn = parseTaskRefIDsInput(vals["depends_on"], current.DependsOn)
	meta.BlockedBy = parseTaskRefIDsInput(vals["blocked_by"], current.BlockedBy)
	blockedReason := strings.TrimSpace(vals["blocked_reason"])
	switch blockedReason {
	case "":
		// Keep current metadata when field is untouched.
	case "-":
		meta.BlockedReason = ""
	default:
		meta.BlockedReason = blockedReason
	}
	meta.ResourceRefs = append([]domain.ResourceRef(nil), m.taskFormResourceRefs...)
	return meta
}

// validateAllowedLabels enforces label allowlists when configured.
func (m Model) validateAllowedLabels(labels []string) error {
	if !m.enforceAllowedLabels || len(labels) == 0 {
		return nil
	}
	allowed := m.allowedLabelsForSelectedProject()
	if len(allowed) == 0 {
		return fmt.Errorf("no labels configured for current project; disable labels.enforce_allowed to allow free-form labels")
	}
	allowedSet := map[string]struct{}{}
	for _, label := range allowed {
		allowedSet[strings.TrimSpace(strings.ToLower(label))] = struct{}{}
	}
	disallowed := make([]string, 0)
	for _, raw := range labels {
		label := strings.TrimSpace(strings.ToLower(raw))
		if label == "" {
			continue
		}
		if _, ok := allowedSet[label]; ok {
			continue
		}
		disallowed = append(disallowed, label)
	}
	if len(disallowed) == 0 {
		return nil
	}
	sort.Strings(disallowed)
	return fmt.Errorf("labels not allowed: %s", strings.Join(disallowed, ", "))
}

// canonicalSearchStates normalizes configured and user-selected search states.
func canonicalSearchStates(states []string) []string {
	out := make([]string, 0, len(canonicalSearchStatesOrdered))
	seen := map[string]struct{}{}
	for _, raw := range states {
		state := strings.TrimSpace(strings.ToLower(raw))
		if state == "" {
			continue
		}
		if !slices.Contains(canonicalSearchStatesOrdered, state) {
			continue
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		out = append(out, state)
	}
	if len(out) == 0 {
		return append([]string(nil), canonicalSearchStatesOrdered...)
	}
	return out
}

// canonicalSearchLevels normalizes configured and user-selected search hierarchy levels.
func canonicalSearchLevels(levels []string) []string {
	out := make([]string, 0, len(canonicalSearchLevelsOrdered))
	seen := map[string]struct{}{}
	for _, raw := range levels {
		level := strings.TrimSpace(strings.ToLower(raw))
		if level == "" {
			continue
		}
		if !slices.Contains(canonicalSearchLevelsOrdered, level) {
			continue
		}
		if _, ok := seen[level]; ok {
			continue
		}
		seen[level] = struct{}{}
		out = append(out, level)
	}
	if len(out) == 0 {
		return append([]string(nil), canonicalSearchLevelsOrdered...)
	}
	return out
}

// toggleSearchState toggles one canonical search state.
func (m *Model) toggleSearchState(state string) {
	state = strings.TrimSpace(strings.ToLower(state))
	if state == "" {
		return
	}
	states := canonicalSearchStates(m.searchStates)
	next := make([]string, 0, len(states))
	found := false
	for _, item := range states {
		if item == state {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		next = append(next, state)
	}
	m.searchStates = canonicalSearchStates(next)
}

// isSearchStateEnabled reports whether a search state is currently enabled.
func (m Model) isSearchStateEnabled(state string) bool {
	state = strings.TrimSpace(strings.ToLower(state))
	for _, item := range m.searchStates {
		if strings.TrimSpace(strings.ToLower(item)) == state {
			return true
		}
	}
	return false
}

// toggleSearchLevel toggles one canonical search hierarchy level.
func (m *Model) toggleSearchLevel(level string) {
	level = strings.TrimSpace(strings.ToLower(level))
	if level == "" {
		return
	}
	levels := canonicalSearchLevels(m.searchLevels)
	next := make([]string, 0, len(levels))
	found := false
	for _, item := range levels {
		if item == level {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		next = append(next, level)
	}
	m.searchLevels = canonicalSearchLevels(next)
}

// isSearchLevelEnabled reports whether a search hierarchy level is currently enabled.
func (m Model) isSearchLevelEnabled(level string) bool {
	level = strings.TrimSpace(strings.ToLower(level))
	for _, item := range m.searchLevels {
		if strings.TrimSpace(strings.ToLower(item)) == level {
			return true
		}
	}
	return false
}

// wrapIndex wraps an index by delta for a bounded collection.
func wrapIndex(current int, delta int, total int) int {
	if total <= 0 {
		return 0
	}
	next := current + delta
	for next < 0 {
		next += total
	}
	for next >= total {
		next -= total
	}
	return next
}

// isForwardTabKey reports whether a key press should advance panel/form focus.
func isForwardTabKey(msg tea.KeyPressMsg) bool {
	return msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i"
}

// isBackwardTabKey reports whether a key press should reverse panel/form focus.
func isBackwardTabKey(msg tea.KeyPressMsg) bool {
	return msg.String() == "shift+tab" || msg.String() == "backtab"
}

// isNoticesPanelVisible reports whether the notices panel can be rendered at current width.
func (m Model) isNoticesPanelVisible() bool {
	if len(m.columns) == 0 {
		return false
	}
	layoutWidth := max(0, m.width-2*tuiOuterHorizontalPadding)
	return m.noticesPanelWidth(layoutWidth) > 0
}

// panelFocusCount returns the number of panel targets available for keyboard focus.
func (m Model) panelFocusCount() int {
	count := len(m.columns)
	if m.isNoticesPanelVisible() {
		count++
	}
	return count
}

// panelFocusIndex resolves the focused panel index across board columns and notices.
func (m Model) panelFocusIndex() int {
	if m.noticesFocused && m.isNoticesPanelVisible() {
		return len(m.columns)
	}
	if len(m.columns) == 0 {
		return 0
	}
	return clamp(m.selectedColumn, 0, len(m.columns)-1)
}

// setPanelFocusIndex applies panel focus by index and returns true when focus changed.
func (m *Model) setPanelFocusIndex(idx int, resetTask bool) bool {
	total := m.panelFocusCount()
	if total <= 0 {
		m.noticesFocused = false
		return false
	}
	idx = clamp(idx, 0, total-1)
	current := m.panelFocusIndex()
	if m.isNoticesPanelVisible() && idx == len(m.columns) {
		changed := !m.noticesFocused || current != idx
		m.noticesFocused = true
		m.clampNoticesSelection()
		return changed
	}
	if len(m.columns) == 0 {
		m.noticesFocused = false
		return false
	}
	targetColumn := clamp(idx, 0, len(m.columns)-1)
	changed := m.noticesFocused || m.selectedColumn != targetColumn
	m.noticesFocused = false
	m.selectedColumn = targetColumn
	if resetTask && changed {
		m.selectedTask = 0
	}
	return changed
}

// cyclePanelFocus moves keyboard focus across panels.
func (m *Model) cyclePanelFocus(delta int, wrap bool, resetTask bool) bool {
	total := m.panelFocusCount()
	if total <= 0 {
		return false
	}
	current := m.panelFocusIndex()
	next := current + delta
	if wrap {
		next = wrapIndex(current, delta, total)
	} else {
		next = clamp(next, 0, total-1)
	}
	if next == current {
		return false
	}
	return m.setPanelFocusIndex(next, resetTask)
}

// normalizePanelFocus keeps panel focus and selections coherent after data/layout updates.
func (m *Model) normalizePanelFocus() {
	if len(m.columns) == 0 {
		m.noticesFocused = false
		m.selectedColumn = 0
		m.selectedTask = 0
		return
	}
	m.selectedColumn = clamp(m.selectedColumn, 0, len(m.columns)-1)
	if !m.isNoticesPanelVisible() {
		m.noticesFocused = false
	}
	if m.noticesFocused {
		m.clampNoticesSelection()
	}
}

// bootstrapActorTypeIndex resolves one actor type to its canonical bootstrap option index.
func bootstrapActorTypeIndex(actorType string) int {
	actorType = strings.TrimSpace(strings.ToLower(actorType))
	for idx, candidate := range bootstrapActorTypes {
		if actorType == candidate {
			return idx
		}
	}
	return 0
}

// windowBounds returns an inclusive-exclusive list window that keeps selected visible.
func windowBounds(total, selected, windowSize int) (int, int) {
	if total <= 0 || windowSize <= 0 {
		return 0, 0
	}
	if total <= windowSize {
		return 0, total
	}
	selected = clamp(selected, 0, total-1)
	half := windowSize / 2
	start := selected - half
	if start < 0 {
		start = 0
	}
	end := start + windowSize
	if end > total {
		end = total
		start = max(0, end-windowSize)
	}
	return start, end
}

// applySearchFilter applies current search values and returns the follow-up command.
func (m *Model) applySearchFilter() tea.Cmd {
	m.mode = modeNone
	m.searchInput.Blur()
	m.searchQuery = strings.TrimSpace(m.searchInput.Value())
	m.searchStates = canonicalSearchStates(m.searchStates)
	m.searchLevels = canonicalSearchLevels(m.searchLevels)
	m.searchApplied = true
	m.selectedTask = 0
	m.status = "search updated"
	if m.searchCrossProject {
		return m.loadSearchMatches
	}
	return m.loadData
}

// clearSearchQuery clears only the search query.
func (m *Model) clearSearchQuery() tea.Cmd {
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchApplied = true
	m.status = "query cleared"
	if m.searchCrossProject {
		return m.loadSearchMatches
	}
	return m.loadData
}

// resetSearchFilters resets query and filters back to defaults.
func (m *Model) resetSearchFilters() tea.Cmd {
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchCrossProject = m.searchDefaultCrossProject
	m.searchIncludeArchived = m.searchDefaultIncludeArchive
	m.searchStates = canonicalSearchStates(m.searchDefaultStates)
	m.searchLevels = canonicalSearchLevels(m.searchDefaultLevels)
	m.searchApplied = false
	m.status = "filters reset"
	return m.loadData
}

// applyRuntimeConfig applies runtime-updateable settings from a reload callback.
func (m *Model) applyRuntimeConfig(cfg RuntimeConfig) {
	WithRuntimeConfig(cfg)(m)
	if actorID := strings.TrimSpace(cfg.Identity.ActorID); actorID != "" {
		m.identityActorID = actorID
	}
	if strings.TrimSpace(m.identityActorID) == "" {
		m.identityActorID = "tillsyn-user"
	}
	m.refreshTaskFormLabelSuggestions()
}

// reloadRuntimeConfigCmd reloads runtime settings through the configured callback.
func (m Model) reloadRuntimeConfigCmd() tea.Cmd {
	if m.reloadConfig == nil {
		return func() tea.Msg {
			return configReloadedMsg{err: fmt.Errorf("config reload callback is unavailable")}
		}
	}
	return func() tea.Msg {
		cfg, err := m.reloadConfig()
		if err != nil {
			return configReloadedMsg{err: err}
		}
		return configReloadedMsg{config: cfg}
	}
}

// submitPathsRoots validates and persists a current-project root mapping change.
func (m Model) submitPathsRoots() (tea.Model, tea.Cmd) {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return m, nil
	}
	slug := strings.TrimSpace(strings.ToLower(project.Slug))
	if slug == "" {
		m.status = "project slug is empty"
		return m, nil
	}
	rootPath, err := normalizeProjectRootPathInput(m.pathsRootInput.Value())
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if m.saveProjectRoot == nil {
		m.status = "save root failed: callback unavailable"
		return m, nil
	}
	m.mode = modeNone
	m.pathsRootInput.Blur()
	m.status = "saving root..."
	return m, m.saveProjectRootCmd(slug, rootPath)
}

// normalizeProjectRootPathInput validates and normalizes an optional project root path value.
func normalizeProjectRootPathInput(raw string) (string, error) {
	rootPath := strings.TrimSpace(raw)
	if rootPath == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(rootPath)
	if err == nil {
		rootPath = absPath
	}
	info, err := os.Stat(rootPath)
	if err != nil {
		return "", fmt.Errorf("root path not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root path must be a directory")
	}
	return rootPath, nil
}

// normalizeSearchRootPathInput validates and normalizes a required global search root path value.
func normalizeSearchRootPathInput(raw string) (string, error) {
	rootPath := strings.TrimSpace(raw)
	if rootPath == "" {
		return "", fmt.Errorf("search root path is required")
	}
	absPath, err := filepath.Abs(rootPath)
	if err == nil {
		rootPath = absPath
	}
	info, err := os.Stat(rootPath)
	if err != nil {
		return "", fmt.Errorf("search root path not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("search root path must be a directory")
	}
	return filepath.Clean(rootPath), nil
}

// normalizeSearchRoots trims and cleans the configured default path, keeping a single entry.
func normalizeSearchRoots(roots []string) []string {
	for _, raw := range roots {
		root := strings.TrimSpace(raw)
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		return []string{root}
	}
	return nil
}

// saveProjectRootCmd persists one project-root mapping through the callback surface.
func (m Model) saveProjectRootCmd(projectSlug, rootPath string) tea.Cmd {
	return func() tea.Msg {
		if err := m.saveProjectRoot(projectSlug, rootPath); err != nil {
			return projectRootSavedMsg{err: err}
		}
		return projectRootSavedMsg{
			projectSlug: projectSlug,
			rootPath:    rootPath,
		}
	}
}

// commandPaletteItems returns all known command-palette items.
func commandPaletteItems() []commandPaletteItem {
	return []commandPaletteItem{
		{Command: "new-task", Aliases: []string{"task-new"}, Description: "create a new task"},
		{Command: "new-subtask", Aliases: []string{"task-subtask"}, Description: "create subtask for selected item"},
		{Command: "new-branch", Aliases: []string{"branch-new"}, Description: "create a new branch"},
		{Command: "new-phase", Aliases: []string{"phase-new"}, Description: "create a new phase"},
		{Command: "new-subphase", Aliases: []string{"subphase-new"}, Description: "create a new subphase"},
		{Command: "edit-branch", Aliases: []string{"branch-edit"}, Description: "edit selected branch"},
		{Command: "archive-branch", Aliases: []string{"branch-archive"}, Description: "archive selected branch"},
		{Command: "delete-branch", Aliases: []string{"branch-delete"}, Description: "hard delete selected branch"},
		{Command: "restore-branch", Aliases: []string{"branch-restore"}, Description: "restore selected archived branch"},
		{Command: "edit-task", Aliases: []string{"task-edit"}, Description: "edit selected task"},
		{Command: "thread-item", Aliases: []string{"item-thread", "task-thread"}, Description: "open selected work-item thread"},
		{Command: "new-project", Aliases: []string{"project-new"}, Description: "create a new project"},
		{Command: "edit-project", Aliases: []string{"project-edit"}, Description: "edit selected project"},
		{Command: "archive-project", Aliases: []string{"project-archive"}, Description: "archive selected project"},
		{Command: "restore-project", Aliases: []string{"project-restore"}, Description: "restore selected archived project"},
		{Command: "delete-project", Aliases: []string{"project-delete"}, Description: "hard delete selected project"},
		{Command: "thread-project", Aliases: []string{"project-thread"}, Description: "open current project thread"},
		{Command: "search", Aliases: []string{}, Description: "open search modal"},
		{Command: "search-all", Aliases: []string{}, Description: "set search scope to all projects"},
		{Command: "search-project", Aliases: []string{}, Description: "set search scope to current project"},
		{Command: "clear-query", Aliases: []string{"clear-search-query"}, Description: "clear search text only"},
		{Command: "reset-filters", Aliases: []string{"clear-search"}, Description: "reset query + states + scope + archived"},
		{Command: "toggle-archived", Aliases: []string{}, Description: "toggle archived visibility"},
		{Command: "toggle-selection-mode", Aliases: []string{"select-mode", "text-select"}, Description: "toggle mouse text-selection mode"},
		{Command: "focus-subtree", Aliases: []string{"zoom-task"}, Description: "show selected task subtree only"},
		{Command: "focus-clear", Aliases: []string{"zoom-reset"}, Description: "return to full board view"},
		{Command: "toggle-select", Aliases: []string{"select-task"}, Description: "toggle selected task in multi-select"},
		{Command: "clear-selection", Aliases: []string{"selection-clear"}, Description: "clear all selected tasks"},
		{Command: "bulk-move-left", Aliases: []string{"move-left-selected"}, Description: "move selected tasks to previous column"},
		{Command: "bulk-move-right", Aliases: []string{"move-right-selected"}, Description: "move selected tasks to next column"},
		{Command: "bulk-archive", Aliases: []string{"archive-selected"}, Description: "archive selected tasks"},
		{Command: "bulk-delete", Aliases: []string{"delete-selected"}, Description: "hard delete selected tasks"},
		{Command: "undo", Aliases: []string{}, Description: "undo last mutation"},
		{Command: "redo", Aliases: []string{}, Description: "redo last undone mutation"},
		{Command: "reload-config", Aliases: []string{"config-reload", "reload"}, Description: "reload runtime config from disk"},
		{Command: "paths-roots", Aliases: []string{"roots", "project-root"}, Description: "edit current project root mapping"},
		{Command: "bootstrap-settings", Aliases: []string{"setup", "identity-roots"}, Description: "edit identity defaults + default path"},
		{Command: "labels-config", Aliases: []string{"labels", "edit-labels"}, Description: "edit global/project/branch/phase labels"},
		{Command: "highlight-color", Aliases: []string{"set-highlight", "focus-color"}, Description: "set focused-row highlight color"},
		{Command: "activity-log", Aliases: []string{"log"}, Description: "open recent activity modal"},
		{Command: "help", Aliases: []string{}, Description: "open help modal"},
		{Command: "quit", Aliases: []string{"exit"}, Description: "quit tillsyn"},
	}
}

// filteredCommandItems returns command items filtered by query.
func (m Model) filteredCommandItems(raw string) []commandPaletteItem {
	query := strings.TrimSpace(strings.ToLower(raw))
	items := commandPaletteItems()
	if query == "" {
		return items
	}
	type scoredItem struct {
		item  commandPaletteItem
		score int
	}
	scored := make([]scoredItem, 0, len(items))
	for _, item := range items {
		score, ok := scoreCommandPaletteItem(query, item)
		if !ok {
			continue
		}
		scored = append(scored, scoredItem{item: item, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].item.Command < scored[j].item.Command
	})
	out := make([]commandPaletteItem, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.item)
	}
	return out
}

// scoreCommandPaletteItem ranks one command-palette item for a fuzzy query.
func scoreCommandPaletteItem(query string, item commandPaletteItem) (int, bool) {
	score := -1
	ok := false
	if v, match := bestFuzzyScore(query, item.Command); match {
		score = max(score, v+200)
		ok = true
	}
	if len(item.Aliases) > 0 {
		if v, match := bestFuzzyScore(query, item.Aliases...); match {
			score = max(score, v+160)
			ok = true
		}
	}
	if v, match := bestFuzzyScore(query, item.Description); match {
		score = max(score, v+80)
		ok = true
	}
	return score, ok
}

// bestFuzzyScore returns the best fuzzy score across candidate strings.
func bestFuzzyScore(query string, candidates ...string) (int, bool) {
	best := 0
	ok := false
	for _, candidate := range candidates {
		score, match := fuzzyScore(query, candidate)
		if !match {
			continue
		}
		if !ok || score > best {
			best = score
		}
		ok = true
	}
	return best, ok
}

// fuzzyScore returns a deterministic fuzzy score where higher is better.
func fuzzyScore(query, candidate string) (int, bool) {
	query = strings.TrimSpace(strings.ToLower(query))
	candidate = strings.TrimSpace(strings.ToLower(candidate))
	if query == "" {
		return 0, true
	}
	if candidate == "" {
		return 0, false
	}

	// Strongly prefer exact/prefix/contains matches before subsequence scoring.
	if query == candidate {
		return 6000, true
	}
	if strings.HasPrefix(candidate, query) {
		return 5000 - len(candidate), true
	}
	if idx := strings.Index(candidate, query); idx >= 0 {
		return 4200 - idx, true
	}

	q := []rune(query)
	c := []rune(candidate)
	qi := 0
	score := 3000
	last := -1
	for ci, r := range c {
		if qi >= len(q) {
			break
		}
		if r != q[qi] {
			continue
		}
		if last < 0 {
			score -= ci
		} else {
			gap := ci - last - 1
			score -= gap * 3
		}
		last = ci
		qi++
	}
	if qi != len(q) {
		return 0, false
	}
	score -= len(c) - len(q)
	return score, true
}

// commandToExecute returns the selected command from the palette state.
func (m Model) commandToExecute() string {
	if len(m.commandMatches) > 0 {
		idx := clamp(m.commandIndex, 0, len(m.commandMatches)-1)
		return m.commandMatches[idx].Command
	}
	return strings.TrimSpace(strings.ToLower(m.commandInput.Value()))
}

// priorityIndex handles priority index.
func priorityIndex(priority domain.Priority) int {
	for i, p := range priorityOptions {
		if p == priority {
			return i
		}
	}
	return 1
}

// cyclePriority handles cycle priority.
func (m *Model) cyclePriority(delta int) {
	if len(priorityOptions) == 0 {
		return
	}
	m.priorityIdx += delta
	if m.priorityIdx < 0 {
		m.priorityIdx = len(priorityOptions) - 1
	}
	if m.priorityIdx >= len(priorityOptions) {
		m.priorityIdx = 0
	}
	if len(m.formInputs) > 2 {
		m.formInputs[2].SetValue(string(priorityOptions[m.priorityIdx]))
	}
}

// startDuePicker starts due picker.
func (m *Model) startDuePicker() {
	m.pickerBack = m.mode
	m.mode = modeDuePicker
	m.duePicker = 0
	m.duePickerFocus = 1
	m.duePickerDateInput.SetValue("")
	m.duePickerDateInput.CursorEnd()
	m.duePickerTimeInput.SetValue("")
	m.duePickerTimeInput.CursorEnd()
	m.duePickerTimeInput.Blur()
	_ = m.duePickerDateInput.Focus()
	m.duePickerIncludeTime = false
	if len(m.formInputs) > taskFieldDue {
		current := strings.TrimSpace(m.formInputs[taskFieldDue].Value())
		if current != "" && current != "-" {
			if parsed, err := parseDueInput(current, nil); err == nil && parsed != nil {
				local := parsed.In(time.Local)
				m.duePickerDateInput.SetValue(local.Format("2006-01-02"))
				if local.Hour() != 0 || local.Minute() != 0 {
					m.duePickerIncludeTime = true
					m.duePickerTimeInput.SetValue(local.Format("15:04"))
				}
			}
		}
	}
	m.duePicker = 0
}

// duePickerOptions handles due picker options.
func (m *Model) duePickerOptions() []duePickerOption {
	now := time.Now().In(time.Local)
	baseDates := []struct {
		label string
		when  time.Time
	}{
		{label: "Today", when: now},
		{label: "Tomorrow", when: now.AddDate(0, 0, 1)},
		{label: "Next week", when: now.AddDate(0, 0, 7)},
		{label: "In two weeks", when: now.AddDate(0, 0, 14)},
	}
	options := make([]duePickerOption, 0, 16)
	options = append(options, duePickerOption{Label: "No due date", Value: "-"})
	if !m.duePickerIncludeTime {
		for _, item := range baseDates {
			value := item.when.Format("2006-01-02")
			options = append(options, duePickerOption{
				Label: fmt.Sprintf("%s (%s)", item.label, value),
				Value: value,
			})
		}
	} else {
		times := []struct {
			label string
			hour  int
			min   int
		}{
			{label: "09:00", hour: 9, min: 0},
			{label: "12:00", hour: 12, min: 0},
			{label: "17:00", hour: 17, min: 0},
		}
		for _, day := range baseDates {
			for _, tm := range times {
				dt := time.Date(day.when.Year(), day.when.Month(), day.when.Day(), tm.hour, tm.min, 0, 0, time.Local)
				value := dt.Format("2006-01-02 15:04")
				options = append(options, duePickerOption{
					Label: fmt.Sprintf("%s %s (%s)", day.label, tm.label, value),
					Value: value,
				})
			}
		}
	}

	dateInput := strings.TrimSpace(m.duePickerDateInput.Value())
	timeInput := strings.TrimSpace(m.duePickerTimeInput.Value())
	if dateInput != "" {
		if typedDate, ok := resolveDuePickerDateToken(dateInput, now); ok {
			if !m.duePickerIncludeTime {
				value := typedDate.Format("2006-01-02")
				options = append([]duePickerOption{{
					Label: fmt.Sprintf("Use typed date (%s)", value),
					Value: value,
				}}, options...)
			} else if hour, minute, ok := parseDuePickerTimeToken(timeInput); ok {
				typedDateTime := time.Date(typedDate.Year(), typedDate.Month(), typedDate.Day(), hour, minute, 0, 0, time.Local)
				value := typedDateTime.Format("2006-01-02 15:04")
				options = append([]duePickerOption{{
					Label: fmt.Sprintf("Use typed datetime (%s)", value),
					Value: value,
				}}, options...)
			}
		} else if prefixDates := resolveDuePickerDatePrefix(dateInput, now); len(prefixDates) > 0 {
			matched := make([]duePickerOption, 0, len(prefixDates))
			for _, candidate := range prefixDates {
				if !m.duePickerIncludeTime {
					value := candidate.Format("2006-01-02")
					matched = append(matched, duePickerOption{
						Label: fmt.Sprintf("Use matched date (%s)", value),
						Value: value,
					})
					continue
				}
				hour, minute, ok := parseDuePickerTimeToken(timeInput)
				if !ok {
					continue
				}
				dt := time.Date(candidate.Year(), candidate.Month(), candidate.Day(), hour, minute, 0, 0, time.Local)
				value := dt.Format("2006-01-02 15:04")
				matched = append(matched, duePickerOption{
					Label: fmt.Sprintf("Use matched datetime (%s)", value),
					Value: value,
				})
			}
			if len(matched) > 0 {
				options = append(matched, options...)
			}
		}
	}

	dateQuery := strings.TrimSpace(strings.ToLower(m.duePickerDateInput.Value()))
	timeQuery := strings.TrimSpace(strings.ToLower(m.duePickerTimeInput.Value()))
	if dateQuery == "" && (!m.duePickerIncludeTime || timeQuery == "") {
		return options
	}

	type scoredOption struct {
		option duePickerOption
		score  int
	}
	scored := make([]scoredOption, 0, len(options))
	for _, option := range options {
		score := 0
		if dateQuery != "" {
			dateScore, ok := bestFuzzyScore(dateQuery, option.Label, option.Value)
			if !ok {
				continue
			}
			score += dateScore
		}
		if m.duePickerIncludeTime && timeQuery != "" {
			timeScore, ok := bestFuzzyScore(timeQuery, option.Label, option.Value)
			if !ok {
				continue
			}
			score += timeScore
		}
		scored = append(scored, scoredOption{option: option, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].option.Value < scored[j].option.Value
	})
	out := make([]duePickerOption, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.option)
	}
	return out
}

// resolveDuePickerDateToken parses due-picker date text into a local calendar date.
func resolveDuePickerDateToken(raw string, now time.Time) (time.Time, bool) {
	token := strings.TrimSpace(strings.ToLower(raw))
	if token == "" {
		return time.Time{}, false
	}
	switch token {
	case "today":
		return now, true
	case "tomorrow":
		return now.AddDate(0, 0, 1), true
	case "next week":
		return now.AddDate(0, 0, 7), true
	case "in two weeks":
		return now.AddDate(0, 0, 14), true
	}
	parsed, err := time.ParseInLocation("2006-01-02", token, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// resolveDuePickerDatePrefix returns upcoming local dates for month/day-prefix tokens (for example 2-2).
func resolveDuePickerDatePrefix(raw string, now time.Time) []time.Time {
	token := strings.TrimSpace(strings.ToLower(raw))
	if token == "" {
		return nil
	}
	parts := strings.Split(token, "-")
	if len(parts) != 2 {
		return nil
	}
	month, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || month < 1 || month > 12 {
		return nil
	}
	dayPrefix := strings.TrimSpace(parts[1])
	if dayPrefix == "" || len(dayPrefix) > 2 {
		return nil
	}
	for _, ch := range dayPrefix {
		if ch < '0' || ch > '9' {
			return nil
		}
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	year := now.Year()
	monthTime := time.Month(month)
	daysInMonth := time.Date(year, monthTime+1, 0, 0, 0, 0, 0, time.Local).Day()
	out := make([]time.Time, 0, daysInMonth)
	for day := 1; day <= daysInMonth; day++ {
		if !strings.HasPrefix(strconv.Itoa(day), dayPrefix) {
			continue
		}
		candidate := time.Date(year, monthTime, day, 0, 0, 0, 0, time.Local)
		if candidate.Before(today) {
			candidate = candidate.AddDate(1, 0, 0)
		}
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}

// parseDuePickerTimeToken parses due-picker time text into hour and minute values.
func parseDuePickerTimeToken(raw string) (int, int, bool) {
	token := strings.TrimSpace(strings.ToLower(raw))
	if token == "" {
		return 0, 0, false
	}
	layouts := []string{
		"15:04",
		"3:04pm",
		"3pm",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, token)
		if err != nil {
			continue
		}
		return parsed.Hour(), parsed.Minute(), true
	}
	return 0, 0, false
}

// duePickerFocusSlots returns the ordered focus slots for due-picker controls.
func (m Model) duePickerFocusSlots() []int {
	slots := []int{0, 1, 3}
	if m.duePickerIncludeTime {
		slots = []int{0, 1, 2, 3}
	}
	return slots
}

// focusDuePickerSlot focuses one due-picker control slot.
func (m *Model) focusDuePickerSlot(slot int) tea.Cmd {
	m.duePickerDateInput.Blur()
	m.duePickerTimeInput.Blur()
	m.duePickerFocus = slot
	switch slot {
	case 1:
		return m.duePickerDateInput.Focus()
	case 2:
		if !m.duePickerIncludeTime {
			m.duePickerFocus = 3
			return nil
		}
		return m.duePickerTimeInput.Focus()
	default:
		return nil
	}
}

// cycleDuePickerFocus advances due-picker focus to the next/previous control.
func (m *Model) cycleDuePickerFocus(delta int) tea.Cmd {
	slots := m.duePickerFocusSlots()
	if len(slots) == 0 {
		return nil
	}
	idx := 0
	for i, slot := range slots {
		if slot == m.duePickerFocus {
			idx = i
			break
		}
	}
	idx = wrapIndex(idx, delta, len(slots))
	return m.focusDuePickerSlot(slots[idx])
}

// setDuePickerIncludeTime toggles timed due-picker options while preserving focus sanity.
func (m *Model) setDuePickerIncludeTime(enabled bool) tea.Cmd {
	m.duePickerIncludeTime = enabled
	if !enabled && m.duePickerFocus == 2 {
		return m.focusDuePickerSlot(3)
	}
	if enabled && m.duePickerFocus == 3 && strings.TrimSpace(m.duePickerTimeInput.Value()) == "" {
		return m.focusDuePickerSlot(2)
	}
	return nil
}

// startLabelPicker opens a modal picker with inherited label suggestions.
func (m *Model) startLabelPicker() tea.Cmd {
	m.labelPickerBack = m.mode
	m.mode = modeLabelPicker
	m.labelPickerInput.SetValue("")
	m.labelPickerInput.CursorEnd()
	m.labelPickerAllItems = m.taskFormLabelPickerItems()
	for _, label := range m.labelSuggestions(48) {
		m.labelPickerAllItems = append(m.labelPickerAllItems, labelPickerItem{Label: label, Source: "suggested"})
	}
	for _, label := range normalizeConfigLabels(defaultLabelSuggestionsSeed) {
		m.labelPickerAllItems = append(m.labelPickerAllItems, labelPickerItem{Label: label, Source: "default"})
	}
	m.refreshLabelPickerMatches()
	m.labelPickerIndex = 0
	if len(m.labelPickerItems) == 0 {
		m.status = "no labels available"
	} else {
		m.status = "label picker"
	}
	return m.labelPickerInput.Focus()
}

// startDependencyInspectorFromTaskInfo opens dependency inspector for one existing task.
func (m *Model) startDependencyInspectorFromTaskInfo(task domain.Task) tea.Cmd {
	return m.startDependencyInspector(
		modeTaskInfo,
		task.ID,
		task.Metadata.DependsOn,
		task.Metadata.BlockedBy,
		taskFieldDependsOn,
	)
}

// startDependencyInspectorFromForm opens dependency inspector for task-form dependency fields.
func (m *Model) startDependencyInspectorFromForm(activeField int) tea.Cmd {
	if activeField != taskFieldDependsOn && activeField != taskFieldBlockedBy {
		activeField = taskFieldDependsOn
	}
	back := m.mode
	ownerTaskID := strings.TrimSpace(m.editingTaskID)
	dependsOn := []string{}
	blockedBy := []string{}
	if len(m.formInputs) > taskFieldDependsOn {
		dependsOn = parseTaskRefIDsInput(m.formInputs[taskFieldDependsOn].Value(), nil)
	}
	if len(m.formInputs) > taskFieldBlockedBy {
		blockedBy = parseTaskRefIDsInput(m.formInputs[taskFieldBlockedBy].Value(), nil)
	}
	return m.startDependencyInspector(back, ownerTaskID, dependsOn, blockedBy, activeField)
}

// startDependencyInspector initializes the dependency inspector modal state.
func (m *Model) startDependencyInspector(back inputMode, ownerTaskID string, dependsOn, blockedBy []string, activeField int) tea.Cmd {
	if activeField != taskFieldDependsOn && activeField != taskFieldBlockedBy {
		activeField = taskFieldDependsOn
	}
	m.dependencyBack = back
	m.dependencyOwnerTaskID = strings.TrimSpace(ownerTaskID)
	m.dependencyDependsOn = sanitizeDependencyIDs(dependsOn, m.dependencyOwnerTaskID)
	m.dependencyBlockedBy = sanitizeDependencyIDs(blockedBy, m.dependencyOwnerTaskID)
	m.dependencyActiveField = activeField
	m.dependencyDirty = false
	m.dependencyFocus = 0
	m.dependencyStateCursor = 0
	m.dependencyCrossProject = m.searchCrossProject
	m.dependencyIncludeArchived = m.showArchived
	m.dependencyStates = canonicalSearchStates(m.searchStates)
	m.dependencyMatches = nil
	m.dependencyIndex = 0
	m.dependencyInput.SetValue("")
	m.dependencyInput.CursorEnd()
	m.mode = modeDependencyInspector
	m.status = "dependencies inspector"
	return tea.Batch(m.dependencyInput.Focus(), m.loadDependencyMatches)
}

// loadDependencyMatches loads filterable dependency candidates with hierarchy path context.
func (m Model) loadDependencyMatches() tea.Msg {
	ctx := context.Background()
	projectID, _ := m.currentProjectID()
	matches, err := m.svc.SearchTaskMatches(ctx, app.SearchTasksFilter{
		ProjectID:       projectID,
		Query:           strings.TrimSpace(m.dependencyInput.Value()),
		CrossProject:    m.dependencyCrossProject,
		IncludeArchived: m.dependencyIncludeArchived,
		States:          append([]string(nil), m.dependencyStates...),
	})
	if err != nil {
		return dependencyMatchesMsg{err: err}
	}

	knownByProject := map[string]map[string]domain.Task{}
	loadTasksByProject := func(projectID string) (map[string]domain.Task, error) {
		if existing, ok := knownByProject[projectID]; ok {
			return existing, nil
		}
		tasks, listErr := m.svc.ListTasks(ctx, projectID, true)
		if listErr != nil {
			return nil, listErr
		}
		byID := make(map[string]domain.Task, len(tasks))
		for _, task := range tasks {
			byID[task.ID] = task
		}
		knownByProject[projectID] = byID
		return byID, nil
	}

	candidateByID := map[string]dependencyCandidate{}
	searchOrder := make([]string, 0, len(matches))
	ownerTaskID := strings.TrimSpace(m.dependencyOwnerTaskID)
	for _, match := range matches {
		taskID := strings.TrimSpace(match.Task.ID)
		if taskID == "" {
			continue
		}
		if ownerTaskID != "" && taskID == ownerTaskID {
			continue
		}
		if _, ok := candidateByID[taskID]; ok {
			continue
		}
		tasksByID, listErr := loadTasksByProject(match.Project.ID)
		if listErr != nil {
			return dependencyMatchesMsg{err: listErr}
		}
		candidateByID[taskID] = dependencyCandidate{
			Match: match,
			Path:  buildDependencyTaskPath(match, tasksByID),
		}
		searchOrder = append(searchOrder, taskID)
	}

	linkedIDs := append([]string(nil), m.dependencyDependsOn...)
	linkedIDs = append(linkedIDs, m.dependencyBlockedBy...)
	linkedIDs = sanitizeDependencyIDs(linkedIDs, ownerTaskID)
	if len(linkedIDs) > 0 {
		projects, listErr := m.svc.ListProjects(ctx, true)
		if listErr != nil {
			return dependencyMatchesMsg{err: listErr}
		}
		projectByID := make(map[string]domain.Project, len(projects))
		for _, project := range projects {
			projectByID[project.ID] = project
		}
		for _, project := range m.projects {
			if _, ok := projectByID[project.ID]; !ok {
				projectByID[project.ID] = project
			}
		}
		for _, linkedID := range linkedIDs {
			if _, ok := candidateByID[linkedID]; ok {
				continue
			}
			found := false
			for projectID, project := range projectByID {
				tasksByID, taskErr := loadTasksByProject(projectID)
				if taskErr != nil {
					return dependencyMatchesMsg{err: taskErr}
				}
				task, ok := tasksByID[linkedID]
				if !ok {
					continue
				}
				match := app.TaskMatch{
					Project: project,
					Task:    task,
					StateID: dependencyStateIDForTask(task),
				}
				candidateByID[linkedID] = dependencyCandidate{
					Match: match,
					Path:  buildDependencyTaskPath(match, tasksByID),
				}
				found = true
				break
			}
			if found {
				continue
			}
			candidateByID[linkedID] = dependencyCandidate{
				Match: app.TaskMatch{
					Project: domain.Project{ID: "missing", Name: "(missing)"},
					Task: domain.Task{
						ID:    linkedID,
						Title: "(missing task reference)",
						Kind:  domain.WorkKindTask,
					},
					StateID: "missing",
				},
				Path: "(missing task reference)",
			}
		}
	}

	candidates := make([]dependencyCandidate, 0, len(candidateByID))
	linkedSet := map[string]struct{}{}
	for _, linkedID := range linkedIDs {
		if _, ok := linkedSet[linkedID]; ok {
			continue
		}
		linkedSet[linkedID] = struct{}{}
		if candidate, ok := candidateByID[linkedID]; ok {
			candidates = append(candidates, candidate)
		}
	}
	for _, taskID := range searchOrder {
		if _, ok := linkedSet[taskID]; ok {
			continue
		}
		if candidate, ok := candidateByID[taskID]; ok {
			candidates = append(candidates, candidate)
		}
	}
	return dependencyMatchesMsg{candidates: candidates}
}

// buildDependencyTaskPath formats project + hierarchy path context for one dependency candidate.
func buildDependencyTaskPath(match app.TaskMatch, tasksByID map[string]domain.Task) string {
	pathParts := []string{}
	current := match.Task
	visited := map[string]struct{}{}
	for {
		label := strings.TrimSpace(current.Title)
		if label == "" {
			label = current.ID
		}
		pathParts = append(pathParts, fmt.Sprintf("%s:%s", current.Kind, label))
		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		if _, ok := visited[parentID]; ok {
			break
		}
		visited[parentID] = struct{}{}
		parent, ok := tasksByID[parentID]
		if !ok {
			break
		}
		current = parent
	}
	slices.Reverse(pathParts)
	if len(pathParts) == 0 {
		pathParts = append(pathParts, fmt.Sprintf("%s:%s", match.Task.Kind, match.Task.Title))
	}
	projectName := strings.TrimSpace(match.Project.Name)
	if projectName == "" {
		projectName = match.Project.ID
	}
	return projectName + " | " + strings.Join(pathParts, " | ")
}

// dependencyStateIDForTask resolves one canonical state identifier for dependency rows.
func dependencyStateIDForTask(task domain.Task) string {
	if task.ArchivedAt != nil {
		return "archived"
	}
	if stateID := normalizeColumnStateID(string(task.LifecycleState)); stateID != "" {
		return stateID
	}
	return "todo"
}

// toggleDependencyState toggles one dependency inspector lifecycle-state filter.
func (m *Model) toggleDependencyState(state string) {
	state = strings.TrimSpace(strings.ToLower(state))
	if state == "" {
		return
	}
	states := canonicalSearchStates(m.dependencyStates)
	next := make([]string, 0, len(states))
	removed := false
	for _, item := range states {
		if item == state {
			removed = true
			continue
		}
		next = append(next, item)
	}
	if !removed {
		next = append(next, state)
	}
	m.dependencyStates = canonicalSearchStates(next)
}

// isDependencyStateEnabled reports whether one dependency inspector state filter is enabled.
func (m Model) isDependencyStateEnabled(state string) bool {
	state = strings.TrimSpace(strings.ToLower(state))
	for _, item := range m.dependencyStates {
		if strings.TrimSpace(strings.ToLower(item)) == state {
			return true
		}
	}
	return false
}

// selectedDependencyCandidate returns the currently highlighted dependency candidate row.
func (m Model) selectedDependencyCandidate() (dependencyCandidate, bool) {
	if len(m.dependencyMatches) == 0 {
		return dependencyCandidate{}, false
	}
	idx := clamp(m.dependencyIndex, 0, len(m.dependencyMatches)-1)
	return m.dependencyMatches[idx], true
}

// hasDependencyID reports whether one id exists in a dependency-id slice.
func hasDependencyID(ids []string, taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false
	}
	for _, id := range ids {
		if strings.TrimSpace(id) == taskID {
			return true
		}
	}
	return false
}

// toggleDependencyID adds/removes one id from a dependency-id slice.
func toggleDependencyID(ids []string, taskID string) ([]string, bool) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return uniqueTrimmed(ids), false
	}
	out := make([]string, 0, len(ids))
	removed := false
	for _, id := range uniqueTrimmed(ids) {
		if id == taskID {
			removed = true
			continue
		}
		out = append(out, id)
	}
	if removed {
		return out, false
	}
	out = append(out, taskID)
	return uniqueTrimmed(out), true
}

// sanitizeDependencyIDs canonicalizes ids and removes any self-reference entry.
func sanitizeDependencyIDs(ids []string, ownerTaskID string) []string {
	ownerTaskID = strings.TrimSpace(ownerTaskID)
	cleaned := make([]string, 0, len(ids))
	for _, id := range uniqueTrimmed(ids) {
		if id == "" {
			continue
		}
		if ownerTaskID != "" && id == ownerTaskID {
			continue
		}
		cleaned = append(cleaned, id)
	}
	return cleaned
}

// dependencyActiveFieldLabel returns the dependency field label currently targeted by space-toggle actions.
func (m Model) dependencyActiveFieldLabel() string {
	if m.dependencyActiveField == taskFieldBlockedBy {
		return "blocked_by"
	}
	return "depends_on"
}

// toggleDependencyCandidateInActiveField toggles highlighted task id in the active dependency field.
func (m *Model) toggleDependencyCandidateInActiveField(taskID string) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}
	if ownerTaskID := strings.TrimSpace(m.dependencyOwnerTaskID); ownerTaskID != "" && taskID == ownerTaskID {
		m.status = "task cannot depend on itself"
		return
	}
	if m.dependencyActiveField == taskFieldBlockedBy {
		var added bool
		m.dependencyBlockedBy, added = toggleDependencyID(m.dependencyBlockedBy, taskID)
		m.dependencyDirty = true
		if added {
			m.status = "added blocker"
		} else {
			m.status = "removed blocker"
		}
		return
	}
	var added bool
	m.dependencyDependsOn, added = toggleDependencyID(m.dependencyDependsOn, taskID)
	m.dependencyDirty = true
	if added {
		m.status = "added dependency"
	} else {
		m.status = "removed dependency"
	}
}

// applyDependencyInspector commits dependency selections and returns to the originating mode.
func (m Model) applyDependencyInspector() (tea.Model, tea.Cmd) {
	dependsOn := sanitizeDependencyIDs(m.dependencyDependsOn, m.dependencyOwnerTaskID)
	blockedBy := sanitizeDependencyIDs(m.dependencyBlockedBy, m.dependencyOwnerTaskID)
	back := m.dependencyBack
	activeField := m.dependencyActiveField
	if activeField != taskFieldDependsOn && activeField != taskFieldBlockedBy {
		activeField = taskFieldDependsOn
	}

	m.dependencyInput.Blur()
	m.dependencyDirty = false

	switch back {
	case modeAddTask, modeEditTask:
		if len(m.formInputs) > taskFieldDependsOn {
			m.formInputs[taskFieldDependsOn].SetValue(strings.Join(dependsOn, ","))
		}
		if len(m.formInputs) > taskFieldBlockedBy {
			m.formInputs[taskFieldBlockedBy].SetValue(strings.Join(blockedBy, ","))
		}
		m.mode = back
		m.status = "dependencies updated"
		if activeField == taskFieldBlockedBy {
			return m, m.focusTaskFormField(taskFieldBlockedBy)
		}
		return m, m.focusTaskFormField(taskFieldDependsOn)
	case modeTaskInfo:
		taskID := strings.TrimSpace(m.dependencyOwnerTaskID)
		task, ok := m.taskByID(taskID)
		if !ok {
			m.mode = modeTaskInfo
			m.taskInfoTaskID = taskID
			m.status = "task not found"
			return m, nil
		}
		meta := task.Metadata
		meta.DependsOn = dependsOn
		meta.BlockedBy = blockedBy
		m.mode = modeTaskInfo
		m.taskInfoTaskID = taskID
		m.status = "saving dependencies..."
		return m, m.updateTaskMetadataCmd(task, meta, "dependencies updated")
	default:
		m.mode = modeNone
		m.status = "dependencies updated"
		return m, nil
	}
}

// jumpToDependencyCandidateTask closes dependency inspector and opens task-info for the highlighted candidate.
func (m Model) jumpToDependencyCandidateTask() (tea.Model, tea.Cmd) {
	if m.dependencyBack != modeTaskInfo {
		m.status = "jump to task is available from task-info inspector"
		return m, nil
	}
	candidate, ok := m.selectedDependencyCandidate()
	if !ok {
		m.status = "no dependency selected"
		return m, nil
	}
	taskID := strings.TrimSpace(candidate.Match.Task.ID)
	if taskID == "" {
		m.status = "no dependency selected"
		return m, nil
	}
	for idx, project := range m.projects {
		if project.ID == candidate.Match.Project.ID {
			m.selectedProject = idx
			break
		}
	}
	m.pendingFocusTaskID = taskID
	m.mode = modeTaskInfo
	m.taskInfoTaskID = taskID
	m.dependencyInput.Blur()
	m.status = "jumping to dependency"
	return m, m.loadData
}

// updateTaskMetadataCmd persists one metadata update for the provided task fields.
func (m Model) updateTaskMetadataCmd(task domain.Task, metadata domain.TaskMetadata, status string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.svc.UpdateTask(context.Background(), app.UpdateTaskInput{
			TaskID:      task.ID,
			Title:       task.Title,
			Description: task.Description,
			Priority:    task.Priority,
			DueAt:       task.DueAt,
			Labels:      append([]string(nil), task.Labels...),
			Metadata:    &metadata,
		})
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{
			status:      status,
			reload:      true,
			focusTaskID: task.ID,
		}
	}
}

// refreshTaskFormLabelSuggestions refreshes task-form label suggestions from inherited sources.
func (m *Model) refreshTaskFormLabelSuggestions() {
	if len(m.formInputs) <= taskFieldLabels {
		return
	}
	suggestions := mergeUniqueLabels(
		mergeLabelSources(m.taskFormLabelSources()),
		m.labelSuggestions(24),
		defaultLabelSuggestionsSeed,
	)
	m.formInputs[taskFieldLabels].SetSuggestions(suggestions)
}

// mergeUniqueLabels returns normalized labels preserving first-seen order across source slices.
func mergeUniqueLabels(groups ...[]string) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, group := range groups {
		for _, raw := range group {
			label := strings.TrimSpace(strings.ToLower(raw))
			if label == "" {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	return out
}

// taskFormLabelSources resolves label inheritance sources for the active task form context.
func (m Model) taskFormLabelSources() labelInheritanceSources {
	task, ok := m.selectedTaskForLabelInheritance()
	if !ok {
		return m.labelSourcesForTask(domain.Task{})
	}
	return m.labelSourcesForTask(task)
}

// labelSourcesForTask resolves inherited labels for one task or taskless project context.
func (m Model) labelSourcesForTask(task domain.Task) labelInheritanceSources {
	sources := labelInheritanceSources{
		Global: normalizeConfigLabels(m.allowedLabelGlobal),
	}
	if project, ok := m.currentProject(); ok {
		projectSlug := strings.TrimSpace(strings.ToLower(project.Slug))
		sources.Project = normalizeConfigLabels(m.allowedLabelProject[projectSlug])
	}
	if strings.TrimSpace(task.ID) != "" {
		sources.Branch = m.labelsFromBranchAncestors(task)
		sources.Phase = m.labelsFromPhaseAncestors(task)
	}
	return sources
}

// selectedTaskForLabelInheritance picks the best task context for inherited label sources.
func (m Model) selectedTaskForLabelInheritance() (domain.Task, bool) {
	if strings.TrimSpace(m.editingTaskID) != "" {
		if task, ok := m.taskByID(m.editingTaskID); ok {
			return task, true
		}
	}
	if strings.TrimSpace(m.taskFormParentID) != "" {
		if task, ok := m.taskByID(m.taskFormParentID); ok {
			return task, true
		}
	}
	return m.selectedTaskInCurrentColumn()
}

// labelsFromPhaseAncestors collects inherited labels from phase ancestors in parent-chain order.
func (m Model) labelsFromPhaseAncestors(task domain.Task) []string {
	out := make([]string, 0)
	seenLabels := map[string]struct{}{}
	visited := map[string]struct{}{}
	current := task
	for strings.TrimSpace(current.ID) != "" {
		if _, seen := visited[current.ID]; seen {
			break
		}
		visited[current.ID] = struct{}{}
		if current.Kind == domain.WorkKindPhase {
			for _, rawLabel := range current.Labels {
				label := strings.TrimSpace(strings.ToLower(rawLabel))
				if label == "" {
					continue
				}
				if _, ok := seenLabels[label]; ok {
					continue
				}
				seenLabels[label] = struct{}{}
				out = append(out, label)
			}
		}
		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, ok := m.taskByID(parentID)
		if !ok {
			break
		}
		current = parent
	}
	return out
}

// labelsFromBranchAncestors collects inherited labels from branch ancestors in parent-chain order.
func (m Model) labelsFromBranchAncestors(task domain.Task) []string {
	out := make([]string, 0)
	seenLabels := map[string]struct{}{}
	visited := map[string]struct{}{}
	current := task
	for strings.TrimSpace(current.ID) != "" {
		if _, seen := visited[current.ID]; seen {
			break
		}
		visited[current.ID] = struct{}{}
		level := baseSearchLevelForTask(current)
		if level == "branch" {
			for _, rawLabel := range current.Labels {
				label := strings.TrimSpace(strings.ToLower(rawLabel))
				if label == "" {
					continue
				}
				if _, ok := seenLabels[label]; ok {
					continue
				}
				seenLabels[label] = struct{}{}
				out = append(out, label)
			}
		}
		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, ok := m.taskByID(parentID)
		if !ok {
			break
		}
		current = parent
	}
	return out
}

// taskFormLabelPickerItems builds source-tagged inherited labels for modal selection.
func (m Model) taskFormLabelPickerItems() []labelPickerItem {
	sources := m.taskFormLabelSources()
	out := make([]labelPickerItem, 0, len(sources.Global)+len(sources.Project)+len(sources.Branch)+len(sources.Phase))
	seen := map[string]struct{}{}
	appendItems := func(source string, labels []string) {
		for _, label := range labels {
			key := source + "\x00" + label
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, labelPickerItem{Label: label, Source: source})
		}
	}
	appendItems("global", sources.Global)
	appendItems("project", sources.Project)
	appendItems("branch", sources.Branch)
	appendItems("phase", sources.Phase)
	return out
}

// refreshLabelPickerMatches refreshes filtered label-picker rows from current query text.
func (m *Model) refreshLabelPickerMatches() {
	query := strings.TrimSpace(m.labelPickerInput.Value())
	if query == "" {
		m.labelPickerItems = append([]labelPickerItem(nil), m.labelPickerAllItems...)
		m.labelPickerIndex = clamp(m.labelPickerIndex, 0, len(m.labelPickerItems)-1)
		return
	}

	type scoredLabel struct {
		item  labelPickerItem
		score int
	}
	scored := make([]scoredLabel, 0, len(m.labelPickerAllItems))
	for _, item := range m.labelPickerAllItems {
		score, ok := bestFuzzyScore(query, item.Label, item.Source)
		if !ok {
			continue
		}
		scored = append(scored, scoredLabel{
			item:  item,
			score: score,
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].item.Label != scored[j].item.Label {
			return scored[i].item.Label < scored[j].item.Label
		}
		return scored[i].item.Source < scored[j].item.Source
	})
	out := make([]labelPickerItem, 0, len(scored))
	for _, entry := range scored {
		out = append(out, entry.item)
	}
	m.labelPickerItems = out
	m.labelPickerIndex = clamp(m.labelPickerIndex, 0, len(m.labelPickerItems)-1)
}

// appendTaskFormLabel appends one normalized label to the form without duplicating entries.
func (m *Model) appendTaskFormLabel(label string) {
	if len(m.formInputs) <= taskFieldLabels {
		return
	}
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		return
	}
	current := parseLabelsInput(m.formInputs[taskFieldLabels].Value(), nil)
	for _, existing := range current {
		if strings.EqualFold(strings.TrimSpace(existing), label) {
			return
		}
	}
	current = append(current, label)
	m.formInputs[taskFieldLabels].SetValue(strings.Join(current, ","))
}

// acceptCurrentLabelSuggestion applies the active autocomplete suggestion into the labels field.
func (m *Model) acceptCurrentLabelSuggestion() bool {
	if len(m.formInputs) <= taskFieldLabels {
		return false
	}
	suggestion := strings.TrimSpace(strings.ToLower(m.formInputs[taskFieldLabels].CurrentSuggestion()))
	if suggestion == "" {
		matches := m.formInputs[taskFieldLabels].MatchedSuggestions()
		if len(matches) == 0 {
			return false
		}
		suggestion = strings.TrimSpace(strings.ToLower(matches[0]))
	}
	if suggestion == "" {
		return false
	}

	raw := strings.TrimSpace(m.formInputs[taskFieldLabels].Value())
	if raw == "" || raw == "-" {
		m.formInputs[taskFieldLabels].SetValue(suggestion)
		m.formInputs[taskFieldLabels].CursorEnd()
		return true
	}

	parts := strings.Split(raw, ",")
	labels := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for idx, part := range parts {
		label := strings.TrimSpace(strings.ToLower(part))
		if idx == len(parts)-1 {
			label = suggestion
		}
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	if len(labels) == 0 {
		labels = append(labels, suggestion)
	}
	m.formInputs[taskFieldLabels].SetValue(strings.Join(labels, ","))
	m.formInputs[taskFieldLabels].CursorEnd()
	return true
}

// startResourcePicker opens filesystem resource selection for a task.
func (m *Model) startResourcePicker(taskID string, back inputMode) tea.Cmd {
	taskID = strings.TrimSpace(taskID)
	root := ""
	switch back {
	case modeTaskInfo, modeAddTask, modeEditTask:
		root = m.resourcePickerRootForCurrentProject()
		if strings.TrimSpace(root) == "" {
			m.status = "resource attach blocked: set project root first"
			return nil
		}
	case modeBootstrapSettings:
		if len(m.bootstrapRoots) > 0 {
			root = strings.TrimSpace(m.bootstrapRoots[clamp(m.bootstrapRootIndex, 0, len(m.bootstrapRoots)-1)])
		} else if strings.TrimSpace(m.defaultRootDir) != "" {
			root = m.defaultRootDir
		}
	default:
		root = m.resourcePickerRootForCurrentProject()
	}
	if strings.TrimSpace(root) == "" {
		root = m.resourcePickerBrowseRoot()
	}
	if strings.TrimSpace(root) == "" {
		m.status = "resource picker root unavailable"
		return nil
	}
	m.mode = modeResourcePicker
	m.resourcePickerBack = back
	m.resourcePickerTaskID = taskID
	m.resourcePickerRoot = root
	m.resourcePickerDir = root
	m.resourcePickerIndex = 0
	m.resourcePickerItems = nil
	m.resourcePickerFilter.SetValue("")
	m.resourcePickerFilter.CursorEnd()
	m.resourcePickerFilter.Focus()
	m.status = "resource picker"
	return m.openResourcePickerDir(root)
}

// openResourcePickerDir loads one directory within the picker root.
func (m Model) openResourcePickerDir(dir string) tea.Cmd {
	root := strings.TrimSpace(m.resourcePickerRoot)
	if root == "" {
		root = m.resourcePickerBrowseRoot()
	}
	return func() tea.Msg {
		if strings.TrimSpace(root) == "" {
			return resourcePickerLoadedMsg{err: fmt.Errorf("resource picker: root path unavailable")}
		}
		entries, current, err := listResourcePickerEntries(root, dir)
		if err != nil {
			return resourcePickerLoadedMsg{err: fmt.Errorf("resource picker: %w", err)}
		}
		return resourcePickerLoadedMsg{
			root:    root,
			current: current,
			entries: entries,
		}
	}
}

// openResourcePickerParent opens the current picker directory parent.
func (m Model) openResourcePickerParent() tea.Cmd {
	current := strings.TrimSpace(m.resourcePickerDir)
	if current == "" {
		current = m.resourcePickerRoot
	}
	parent := filepath.Dir(current)
	if parent == "." || parent == "" {
		parent = m.resourcePickerRoot
	}
	return m.openResourcePickerDir(parent)
}

// selectedResourcePickerEntry returns the currently highlighted resource picker entry.
func (m Model) selectedResourcePickerEntry() (resourcePickerEntry, bool) {
	items := m.visibleResourcePickerItems()
	if len(items) == 0 {
		return resourcePickerEntry{}, false
	}
	idx := clamp(m.resourcePickerIndex, 0, len(items)-1)
	return items[idx], true
}

// visibleResourcePickerItems returns resource picker entries after applying fuzzy filter text.
func (m Model) visibleResourcePickerItems() []resourcePickerEntry {
	if len(m.resourcePickerItems) == 0 {
		return nil
	}
	query := strings.TrimSpace(m.resourcePickerFilter.Value())
	if query == "" {
		return append([]resourcePickerEntry(nil), m.resourcePickerItems...)
	}

	type scoredEntry struct {
		entry resourcePickerEntry
		score int
	}
	scored := make([]scoredEntry, 0, len(m.resourcePickerItems))
	for _, entry := range m.resourcePickerItems {
		score, ok := bestFuzzyScore(query, entry.Name, filepath.ToSlash(entry.Path))
		if !ok {
			continue
		}
		if entry.IsDir {
			score += 8
		}
		if entry.Name == ".." {
			score -= 100
		}
		scored = append(scored, scoredEntry{entry: entry, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].entry.IsDir != scored[j].entry.IsDir {
			return scored[i].entry.IsDir
		}
		return strings.ToLower(scored[i].entry.Name) < strings.ToLower(scored[j].entry.Name)
	})
	out := make([]resourcePickerEntry, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.entry)
	}
	return out
}

// attachSelectedResourceEntry attaches the currently selected resource entry to the target task.
func (m *Model) attachSelectedResourceEntry() tea.Cmd {
	entry, ok := m.selectedResourcePickerEntry()
	if !ok {
		// Empty directories still allow attaching the current folder as context.
		entry = resourcePickerEntry{
			Name:  filepath.Base(m.resourcePickerDir),
			Path:  m.resourcePickerDir,
			IsDir: true,
		}
	}
	return m.attachResourcePickerEntry(entry)
}

// attachCurrentResourcePickerDir attaches the currently open picker directory.
func (m *Model) attachCurrentResourcePickerDir() tea.Cmd {
	current := strings.TrimSpace(m.resourcePickerDir)
	if current == "" {
		current = strings.TrimSpace(m.resourcePickerRoot)
	}
	if current == "" {
		m.status = "resource path is required"
		return nil
	}
	entry := resourcePickerEntry{
		Name:  filepath.Base(current),
		Path:  current,
		IsDir: true,
	}
	return m.attachResourcePickerEntry(entry)
}

// attachResourcePickerEntry applies one picker entry for the current picker back-flow.
func (m *Model) attachResourcePickerEntry(entry resourcePickerEntry) tea.Cmd {
	back := m.resourcePickerBack
	m.mode = back
	m.resourcePickerFilter.Blur()
	m.resourcePickerFilter.SetValue("")
	m.resourcePickerIndex = 0

	// Task form attachment flow stages refs for create/edit submit.
	if back == modeAddTask || back == modeEditTask {
		normalizedPath, err := normalizeAttachmentPathWithinRoot(strings.TrimSpace(m.resourcePickerRoot), entry.Path)
		if err != nil {
			m.status = err.Error()
			return m.focusTaskFormField(m.formFocus)
		}
		entry.Path = normalizedPath
		ref := buildResourceRef(strings.TrimSpace(m.resourcePickerRoot), entry.Path, entry.IsDir)
		refs, added := appendResourceRefIfMissing(m.taskFormResourceRefs, ref)
		if !added {
			m.status = "resource already staged"
			return m.focusTaskFormField(m.formFocus)
		}
		m.taskFormResourceRefs = refs
		m.status = "resource staged"
		return m.focusTaskFormField(m.formFocus)
	}

	// Project root picker flow writes selected directory back to form/input.
	if back == modeAddProject || back == modeEditProject || back == modePathsRoots {
		selectedDir := entry.Path
		if !entry.IsDir {
			selectedDir = filepath.Dir(selectedDir)
		}
		normalized, err := normalizeProjectRootPathInput(selectedDir)
		if err != nil {
			m.status = err.Error()
			return nil
		}
		if back == modePathsRoots {
			m.pathsRootInput.SetValue(normalized)
			m.pathsRootInput.CursorEnd()
			m.status = "root path selected"
			return m.pathsRootInput.Focus()
		}
		if len(m.projectFormInputs) > projectFieldRootPath {
			m.projectFormInputs[projectFieldRootPath].SetValue(normalized)
			m.projectFormInputs[projectFieldRootPath].CursorEnd()
			m.projectFormFocus = projectFieldRootPath
			m.status = "root path selected"
			return m.focusProjectFormField(projectFieldRootPath)
		}
		m.status = "root path selected"
		return nil
	}

	// Bootstrap settings flow sets one default path.
	if back == modeBootstrapSettings {
		selectedDir := entry.Path
		if !entry.IsDir {
			selectedDir = filepath.Dir(selectedDir)
		}
		root, err := normalizeSearchRootPathInput(selectedDir)
		if err != nil {
			m.status = err.Error()
			return m.focusBootstrapField(1)
		}
		if !m.addBootstrapSearchRoot(root) {
			m.status = "default path unchanged"
			return m.focusBootstrapField(1)
		}
		m.status = "default path set"
		return m.focusBootstrapField(1)
	}

	// Existing task-info path persists immediately to task metadata.
	if _, err := normalizeAttachmentPathWithinRoot(strings.TrimSpace(m.resourcePickerRoot), entry.Path); err != nil {
		m.status = err.Error()
		return nil
	}
	m.status = "attaching resource..."
	return m.attachResourceEntry(entry.Path, entry.IsDir)
}

// attachResourceEntry persists one filesystem reference through task metadata update.
func (m Model) attachResourceEntry(path string, isDir bool) tea.Cmd {
	taskID := strings.TrimSpace(m.resourcePickerTaskID)
	root := strings.TrimSpace(m.resourcePickerRoot)
	return func() tea.Msg {
		normalizedPath, err := normalizeAttachmentPathWithinRoot(root, path)
		if err != nil {
			return actionMsg{status: err.Error()}
		}
		task, ok := m.taskByID(taskID)
		if !ok {
			return actionMsg{status: "resource attach failed: task not found"}
		}
		ref := buildResourceRef(root, normalizedPath, isDir)
		refs, added := appendResourceRefIfMissing(task.Metadata.ResourceRefs, ref)
		if !added {
			return actionMsg{status: "resource already attached"}
		}
		meta := task.Metadata
		meta.ResourceRefs = refs
		_, err = m.svc.UpdateTask(context.Background(), app.UpdateTaskInput{
			TaskID:      task.ID,
			Title:       task.Title,
			Description: task.Description,
			Priority:    task.Priority,
			DueAt:       task.DueAt,
			Labels:      append([]string(nil), task.Labels...),
			Metadata:    &meta,
		})
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{
			status:      "resource attached",
			reload:      true,
			focusTaskID: task.ID,
		}
	}
}

// resourcePickerRootForCurrentProject returns the configured project root for the active project.
func (m Model) resourcePickerRootForCurrentProject() string {
	if project, ok := m.currentProject(); ok {
		slug := strings.TrimSpace(strings.ToLower(project.Slug))
		if root := strings.TrimSpace(m.projectRoots[slug]); root != "" {
			if abs, err := filepath.Abs(root); err == nil {
				return abs
			}
			return root
		}
	}
	return ""
}

// resourcePickerBrowseRoot returns a best-effort browse root for non-task picker flows.
func (m Model) resourcePickerBrowseRoot() string {
	for _, root := range m.searchRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if abs, err := filepath.Abs(root); err == nil {
			return abs
		}
		return root
	}
	if root := strings.TrimSpace(m.defaultRootDir); root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			return abs
		}
		return root
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

// summarizeTaskRefs renders dependency IDs with known task titles when available.
func (m Model) summarizeTaskRefs(ids []string, maxItems int) string {
	items := uniqueTrimmed(ids)
	if len(items) == 0 {
		return "-"
	}
	if maxItems <= 0 {
		maxItems = 4
	}
	visible := items
	extra := 0
	if len(items) > maxItems {
		visible = items[:maxItems]
		extra = len(items) - maxItems
	}
	parts := make([]string, 0, len(visible))
	for _, id := range visible {
		label := id
		if task, ok := m.taskByID(id); ok && strings.TrimSpace(task.Title) != "" {
			label = fmt.Sprintf("%s(%s)", id, truncate(task.Title, 22))
		}
		parts = append(parts, label)
	}
	joined := strings.Join(parts, ", ")
	if extra > 0 {
		joined += fmt.Sprintf(" +%d", extra)
	}
	return joined
}

// uniqueTrimmed trims and deduplicates text values while preserving order.
func uniqueTrimmed(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// formatLabelSource renders one inherited label source for modal hints.
func formatLabelSource(source string, labels []string) string {
	if len(labels) == 0 {
		return source + ": -"
	}
	return source + ": " + strings.Join(labels, ", ")
}

// mergeLabelSources merges inherited label sources using global -> project -> branch -> phase precedence.
func mergeLabelSources(sources labelInheritanceSources) []string {
	out := make([]string, 0, len(sources.Global)+len(sources.Project)+len(sources.Branch)+len(sources.Phase))
	seen := map[string]struct{}{}
	appendUnique := func(values []string) {
		for _, raw := range values {
			label := strings.TrimSpace(strings.ToLower(raw))
			if label == "" {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	appendUnique(sources.Global)
	appendUnique(sources.Project)
	appendUnique(sources.Branch)
	appendUnique(sources.Phase)
	return out
}

// normalizeConfigLabels trims and deduplicates config-provided label lists.
func normalizeConfigLabels(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		label := strings.TrimSpace(strings.ToLower(raw))
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	return out
}

// listResourcePickerEntries loads picker entries using root as the default start directory.
func listResourcePickerEntries(root, dir string) ([]resourcePickerEntry, string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	dirAbs := strings.TrimSpace(dir)
	if dirAbs == "" {
		dirAbs = root
	}
	dirAbs, err := filepath.Abs(dirAbs)
	if err != nil {
		return nil, "", err
	}
	items, err := os.ReadDir(dirAbs)
	if err != nil {
		return nil, "", err
	}
	entries := make([]resourcePickerEntry, 0, len(items)+1)
	parent := filepath.Dir(dirAbs)
	if parent != "." && parent != "" && parent != dirAbs {
		entries = append(entries, resourcePickerEntry{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}
	for _, item := range items {
		entries = append(entries, resourcePickerEntry{
			Name:  item.Name(),
			Path:  filepath.Join(dirAbs, item.Name()),
			IsDir: item.IsDir(),
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, dirAbs, nil
}

// buildResourceRef builds a normalized local-file or local-directory resource reference.
func buildResourceRef(root, path string, isDir bool) domain.ResourceRef {
	path = strings.TrimSpace(path)
	if path == "" {
		path = root
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	resourceType := domain.ResourceTypeLocalFile
	if isDir {
		resourceType = domain.ResourceTypeLocalDir
	}
	now := time.Now().UTC()
	ref := domain.ResourceRef{
		ResourceType:   resourceType,
		Location:       filepath.ToSlash(path),
		PathMode:       domain.PathModeAbsolute,
		Title:          filepath.Base(path),
		LastVerifiedAt: &now,
	}
	root = strings.TrimSpace(root)
	if root != "" {
		if absRoot, err := filepath.Abs(root); err == nil {
			if rel, relErr := filepath.Rel(absRoot, path); relErr == nil && !strings.HasPrefix(rel, "..") {
				ref.Location = filepath.ToSlash(rel)
				ref.PathMode = domain.PathModeRelative
				ref.BaseAlias = "project_root"
			}
		}
	}
	return ref
}

// normalizeAttachmentPathWithinRoot normalizes and validates one attachment path against root scope.
func normalizeAttachmentPathWithinRoot(root, path string) (string, error) {
	root = strings.TrimSpace(root)
	path = strings.TrimSpace(path)
	if root == "" {
		return "", fmt.Errorf("project root path is required for resource attachments")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("allowed root path invalid: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("allowed root path invalid: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("allowed root path invalid: not a directory")
	}
	if path == "" {
		path = absRoot
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resource path invalid: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("resource path relation failed: %w", err)
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return absPath, nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("resource path is outside allowed root")
	}
	return absPath, nil
}

// appendResourceRefIfMissing appends a resource ref unless an equivalent ref already exists.
func appendResourceRefIfMissing(in []domain.ResourceRef, candidate domain.ResourceRef) ([]domain.ResourceRef, bool) {
	candidateLocation := strings.TrimSpace(strings.ToLower(candidate.Location))
	for _, existing := range in {
		existingLocation := strings.TrimSpace(strings.ToLower(existing.Location))
		if existing.ResourceType == candidate.ResourceType &&
			existing.PathMode == candidate.PathMode &&
			existingLocation == candidateLocation {
			return in, false
		}
	}
	return append(append([]domain.ResourceRef(nil), in...), candidate), true
}

// labelSuggestions handles label suggestions.
func (m Model) labelSuggestions(maxLabels int) []string {
	if maxLabels <= 0 {
		maxLabels = 5
	}
	projectID, ok := m.currentProjectID()
	if !ok {
		return nil
	}
	counts := map[string]int{}
	for _, allowed := range m.allowedLabelsForSelectedProject() {
		counts[allowed] += 1000
	}
	for _, task := range m.tasks {
		if task.ProjectID != projectID {
			continue
		}
		for _, label := range task.Labels {
			label = strings.TrimSpace(label)
			if label == "" {
				continue
			}
			counts[label]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	type pair struct {
		label string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for label, count := range counts {
		pairs = append(pairs, pair{label: label, count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].label < pairs[j].label
		}
		return pairs[i].count > pairs[j].count
	})
	out := make([]string, 0, min(maxLabels, len(pairs)))
	for idx := range pairs {
		if idx >= maxLabels {
			break
		}
		out = append(out, pairs[idx].label)
	}
	return out
}

// toggleHelpOverlay toggles the centered help modal for the active screen.
func (m *Model) toggleHelpOverlay() {
	m.help.ShowAll = !m.help.ShowAll
	if m.help.ShowAll {
		m.status = "help"
		return
	}
	if m.mode == modeNone {
		m.status = "ready"
	}
}

// toggleMouseSelectionMode toggles terminal-friendly text selection mode.
func (m *Model) toggleMouseSelectionMode() {
	m.mouseSelectionMode = !m.mouseSelectionMode
	if m.mouseSelectionMode {
		m.status = "text selection mode enabled"
		return
	}
	m.status = "text selection mode disabled"
}

// startWarningModal opens a dismiss-only warning modal with short guidance text.
func (m *Model) startWarningModal(title, body string) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Warning"
	}
	m.warningTitle = title
	m.warningBody = strings.TrimSpace(body)
	m.mode = modeWarning
}

// closeWarningModal dismisses the warning modal and clears staged text.
func (m *Model) closeWarningModal() {
	m.warningTitle = ""
	m.warningBody = ""
	m.mode = modeNone
}

// handleNormalModeKey handles normal mode key.
func (m Model) handleNormalModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.toggleHelp):
		m.toggleHelpOverlay()
		return m, nil
	case msg.String() == "esc":
		if m.help.ShowAll {
			m.toggleHelpOverlay()
			return m, nil
		}
		if m.noticesFocused {
			m.noticesFocused = false
			m.status = "board focus"
			return m, nil
		}
		if m.searchApplied || m.searchQuery != "" {
			m.searchApplied = false
			m.searchQuery = ""
			m.status = "search cleared"
			return m, m.loadData
		}
		if count := m.clearSelection(); count > 0 {
			m.status = fmt.Sprintf("cleared %d selected tasks", count)
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "clear selection",
				Target:  fmt.Sprintf("%d tasks", count),
			})
			return m, nil
		}
		if strings.TrimSpace(m.projectionRootTaskID) != "" {
			m.clearSubtreeFocus()
			m.status = "full board view"
			return m, nil
		}
		return m, nil
	case key.Matches(msg, m.keys.reload):
		m.status = "reloading..."
		return m, m.loadData
	case isForwardTabKey(msg):
		if !m.cyclePanelFocus(1, true, true) {
			m.status = "panel focus unchanged"
			return m, nil
		}
		if m.noticesFocused {
			m.status = "notices focus"
		} else {
			m.status = "board focus"
		}
		return m, nil
	case isBackwardTabKey(msg):
		if !m.cyclePanelFocus(-1, true, true) {
			m.status = "panel focus unchanged"
			return m, nil
		}
		if m.noticesFocused {
			m.status = "notices focus"
		} else {
			m.status = "board focus"
		}
		return m, nil
	case key.Matches(msg, m.keys.moveLeft):
		if m.cyclePanelFocus(-1, false, true) {
			if m.noticesFocused {
				m.status = "notices focus"
			} else {
				m.status = "board focus"
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.moveRight):
		if m.cyclePanelFocus(1, false, true) {
			if m.noticesFocused {
				m.status = "notices focus"
			} else {
				m.status = "board focus"
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.toggleSelectMode):
		m.toggleMouseSelectionMode()
		return m, nil
	}

	if m.noticesFocused {
		return m.handleNoticesPanelNormalKey(msg)
	}
	return m.handleBoardPanelNormalKey(msg)
}

// handleNoticesPanelNormalKey handles board-mode input while notices panel owns focus.
func (m Model) handleNoticesPanelNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.moveDown):
		if m.moveNoticesSelection(1) {
			m.status = "notices: " + strings.ToLower(noticesSectionTitle(m.noticesSection))
		}
		return m, nil
	case key.Matches(msg, m.keys.moveUp):
		if m.moveNoticesSelection(-1) {
			m.status = "notices: " + strings.ToLower(noticesSectionTitle(m.noticesSection))
		}
		return m, nil
	case msg.Code == tea.KeyEnter || msg.String() == "enter":
		return m.activateNoticesSelection()
	case key.Matches(msg, m.keys.activityLog):
		return m, m.openActivityLog()
	default:
		return m, nil
	}
}

// handleBoardPanelNormalKey handles board-mode input while a board column owns focus.
func (m Model) handleBoardPanelNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.moveDown):
		tasks := m.currentColumnTasks()
		if len(tasks) > 0 && m.selectedTask < len(tasks)-1 {
			m.selectedTask++
		}
		return m, nil
	case key.Matches(msg, m.keys.moveUp):
		if m.selectedTask > 0 {
			m.selectedTask--
		}
		return m, nil
	case key.Matches(msg, m.keys.multiSelect):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		if m.toggleTaskSelection(task.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "select task",
				Target:  task.Title,
			})
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "unselect task",
				Target:  task.Title,
			})
		}
		return m, nil
	case key.Matches(msg, m.keys.activityLog):
		return m, m.openActivityLog()
	case key.Matches(msg, m.keys.undo):
		return m.undoLastMutation()
	case key.Matches(msg, m.keys.redo):
		return m.redoLastMutation()
	case key.Matches(msg, m.keys.addTask):
		m.help.ShowAll = false
		return m, m.startTaskForm(nil)
	case key.Matches(msg, m.keys.addSubtask):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.help.ShowAll = false
		return m, m.startSubtaskForm(task)
	case key.Matches(msg, m.keys.newProject):
		m.help.ShowAll = false
		return m, m.startProjectForm(nil)
	case key.Matches(msg, m.keys.taskInfo):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.help.ShowAll = false
		m.openTaskInfo(task.ID, "task info")
		return m, nil
	case key.Matches(msg, m.keys.search):
		m.help.ShowAll = false
		return m, m.startSearchMode()
	case key.Matches(msg, m.keys.commandPalette):
		m.help.ShowAll = false
		return m, m.startCommandPalette()
	case key.Matches(msg, m.keys.quickActions):
		m.help.ShowAll = false
		return m, m.startQuickActions()
	case key.Matches(msg, m.keys.editTask):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.help.ShowAll = false
		return m, m.startTaskForm(&task)
	case key.Matches(msg, m.keys.editProject):
		if len(m.projects) == 0 {
			m.status = "no project selected"
			return m, nil
		}
		m.help.ShowAll = false
		project := m.projects[clamp(m.selectedProject, 0, len(m.projects)-1)]
		return m, m.startProjectForm(&project)
	case key.Matches(msg, m.keys.projects):
		m.help.ShowAll = false
		m.mode = modeProjectPicker
		if len(m.projects) == 0 {
			m.projectPickerIndex = 0
		} else {
			m.projectPickerIndex = clamp(m.selectedProject, 0, len(m.projects)-1)
		}
		m.status = "project picker"
		return m, nil
	case key.Matches(msg, m.keys.focusSubtree):
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		if !m.activateSubtreeFocus(task.ID) {
			return m, nil
		}
		m.status = "focused subtree"
		return m, nil
	case key.Matches(msg, m.keys.clearFocus):
		if !m.clearSubtreeFocus() {
			m.status = "full board already visible"
			return m, nil
		}
		m.status = "full board view"
		return m, nil
	case key.Matches(msg, m.keys.moveTaskLeft):
		if len(m.selectedTaskIDs) > 0 {
			return m.moveSelectedTasks(-1)
		}
		return m.moveSelectedTask(-1)
	case key.Matches(msg, m.keys.moveTaskRight):
		if len(m.selectedTaskIDs) > 0 {
			return m.moveSelectedTasks(1)
		}
		return m.moveSelectedTask(1)
	case key.Matches(msg, m.keys.deleteTask):
		return m.confirmDeleteAction(m.defaultDeleteMode, m.confirmDelete, "delete task")
	case key.Matches(msg, m.keys.hardDeleteTask):
		return m.confirmDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete task")
	case key.Matches(msg, m.keys.restoreTask):
		return m.confirmRestoreAction()
	case key.Matches(msg, m.keys.toggleArchived):
		m.showArchived = !m.showArchived
		if m.showArchived {
			m.status = "showing archived tasks"
		} else {
			m.status = "hiding archived tasks"
		}
		m.selectedTask = 0
		m.clearSelection()
		return m, m.loadData
	default:
		return m, nil
	}
}

// handleInputModeKey handles input mode key.
func (m Model) handleInputModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.toggleSelectMode) {
		m.toggleMouseSelectionMode()
		return m, nil
	}
	if key.Matches(msg, m.keys.toggleHelp) {
		m.toggleHelpOverlay()
		return m, nil
	}
	if m.help.ShowAll {
		if msg.Code == tea.KeyEscape || msg.String() == "esc" {
			m.toggleHelpOverlay()
		}
		return m, nil
	}

	if m.mode == modeActivityEventInfo {
		switch {
		case msg.String() == "esc":
			m.mode = modeNone
			if m.isNoticesPanelVisible() {
				_ = m.setPanelFocusIndex(len(m.columns), false)
				m.status = "notices focus"
			} else {
				m.noticesFocused = false
				m.status = "board focus"
			}
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter" || msg.String() == "g":
			return m.jumpToActivityNode()
		default:
			return m, nil
		}
	}

	if m.mode == modeActivityLog {
		switch {
		case msg.String() == "esc" || key.Matches(msg, m.keys.activityLog):
			m.mode = modeNone
			m.status = "ready"
			return m, nil
		case key.Matches(msg, m.keys.undo):
			return m.undoLastMutation()
		case key.Matches(msg, m.keys.redo):
			return m.redoLastMutation()
		default:
			return m, nil
		}
	}

	if m.mode == modeThread {
		if handled, status := applyClipboardShortcutToInput(msg, &m.threadInput); handled {
			m.status = status
			return m, nil
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.threadInput.Blur()
			m.threadPendingCommentBody = ""
			if m.threadBackMode == modeTaskInfo {
				m.mode = modeTaskInfo
				m.status = "task info"
				return m, nil
			}
			m.mode = modeNone
			m.status = "ready"
			return m, nil
		case msg.String() == "ctrl+r":
			m.status = "reloading thread..."
			return m, m.loadThreadCommentsCmd(m.threadTarget)
		case msg.Code == tea.KeyPgUp || msg.String() == "pgup":
			m.threadScroll = max(0, m.threadScroll-max(1, m.threadViewportStep()))
			return m, nil
		case msg.Code == tea.KeyPgDown || msg.String() == "pgdown":
			m.threadScroll += max(1, m.threadViewportStep())
			return m, nil
		case msg.String() == "home":
			m.threadScroll = 0
			return m, nil
		case msg.String() == "end":
			m.threadScroll += 1000
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			body := strings.TrimSpace(m.threadInput.Value())
			if body == "" {
				m.status = "comment body required"
				return m, nil
			}
			m.threadPendingCommentBody = body
			m.threadInput.SetValue("")
			m.threadInput.CursorEnd()
			m.status = "posting comment..."
			return m, m.createThreadCommentCmd(body)
		default:
			var cmd tea.Cmd
			m.threadInput, cmd = m.threadInput.Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeTaskInfo {
		task, ok := m.taskInfoTask()
		if !ok {
			m.closeTaskInfo("task info unavailable")
			return m, nil
		}
		subtasks := m.subtasksForParent(task.ID)
		switch msg.String() {
		case "esc":
			originID := strings.TrimSpace(m.taskInfoOriginTaskID)
			if originID != "" && originID != task.ID {
				if _, ok := m.taskByID(originID); ok {
					m.taskInfoTaskID = originID
					m.taskInfoSubtaskIdx = 0
					m.status = "task info"
					return m, nil
				}
			}
			if m.stepBackTaskInfo(task) {
				return m, nil
			}
			m.closeTaskInfo("ready")
			return m, nil
		case "i":
			m.closeTaskInfo("ready")
			return m, nil
		case "j", "down":
			if len(subtasks) > 0 && m.taskInfoSubtaskIdx < len(subtasks)-1 {
				m.taskInfoSubtaskIdx++
			}
			return m, nil
		case "k", "up":
			if m.taskInfoSubtaskIdx > 0 {
				m.taskInfoSubtaskIdx--
			}
			return m, nil
		case "enter":
			if len(subtasks) == 0 {
				return m, nil
			}
			subtask := subtasks[clamp(m.taskInfoSubtaskIdx, 0, len(subtasks)-1)]
			m.taskInfoTaskID = subtask.ID
			m.taskInfoSubtaskIdx = 0
			m.status = "subtask info"
			return m, nil
		case "backspace", "h", "left":
			parentID := strings.TrimSpace(task.ParentID)
			if parentID == "" {
				return m, nil
			}
			if _, ok := m.taskByID(parentID); !ok {
				return m, nil
			}
			m.taskInfoTaskID = parentID
			m.taskInfoSubtaskIdx = 0
			m.status = "parent task info"
			return m, nil
		case "e":
			return m, m.startTaskForm(&task)
		case "s":
			return m, m.startSubtaskForm(task)
		case "b":
			return m, m.startDependencyInspectorFromTaskInfo(task)
		case "c":
			return m.startTaskThread(task, modeTaskInfo)
		case "r", "R":
			return m, m.startResourcePicker(task.ID, modeTaskInfo)
		case " ", "space":
			return m.toggleFocusedSubtaskCompletion(task)
		case "[":
			return m.moveTaskIDs([]string{task.ID}, -1, "move task", task.Title, false)
		case "]":
			return m.moveTaskIDs([]string{task.ID}, 1, "move task", task.Title, false)
		case "f":
			if !m.activateSubtreeFocus(task.ID) {
				return m, nil
			}
			m.closeTaskInfo("focused subtree")
			return m, nil
		default:
			return m, nil
		}
	}

	if m.mode == modeDependencyInspector {
		if m.dependencyFocus == 0 {
			if handled, status := applyClipboardShortcutToInput(msg, &m.dependencyInput); handled {
				m.status = status
				return m, nil
			}
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = m.dependencyBack
			m.dependencyInput.Blur()
			if m.mode == modeTaskInfo {
				m.taskInfoTaskID = strings.TrimSpace(m.dependencyOwnerTaskID)
			}
			m.status = "dependency inspector cancelled"
			if m.mode == modeAddTask || m.mode == modeEditTask {
				return m, m.focusTaskFormField(m.dependencyActiveField)
			}
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i":
			m.dependencyFocus = wrapIndex(m.dependencyFocus, 1, 5)
			if m.dependencyFocus == 0 {
				return m, m.dependencyInput.Focus()
			}
			m.dependencyInput.Blur()
			return m, nil
		case msg.String() == "shift+tab" || msg.String() == "backtab":
			m.dependencyFocus = wrapIndex(m.dependencyFocus, -1, 5)
			if m.dependencyFocus == 0 {
				return m, m.dependencyInput.Focus()
			}
			m.dependencyInput.Blur()
			return m, nil
		case msg.String() == "j" || msg.String() == "down":
			if m.dependencyFocus == 4 {
				if m.dependencyIndex < len(m.dependencyMatches)-1 {
					m.dependencyIndex++
				}
				return m, nil
			}
			m.dependencyFocus = wrapIndex(m.dependencyFocus, 1, 5)
			if m.dependencyFocus == 0 {
				return m, m.dependencyInput.Focus()
			}
			m.dependencyInput.Blur()
			return m, nil
		case msg.String() == "k" || msg.String() == "up":
			if m.dependencyFocus == 4 {
				if m.dependencyIndex > 0 {
					m.dependencyIndex--
				}
				return m, nil
			}
			m.dependencyFocus = wrapIndex(m.dependencyFocus, -1, 5)
			if m.dependencyFocus == 0 {
				return m, m.dependencyInput.Focus()
			}
			m.dependencyInput.Blur()
			return m, nil
		case msg.String() == "ctrl+u":
			m.dependencyInput.SetValue("")
			m.dependencyInput.CursorEnd()
			m.dependencyIndex = 0
			return m, m.loadDependencyMatches
		case msg.String() == "ctrl+r":
			m.dependencyCrossProject = m.searchDefaultCrossProject
			m.dependencyIncludeArchived = m.searchDefaultIncludeArchive
			m.dependencyStates = canonicalSearchStates(m.searchDefaultStates)
			m.dependencyStateCursor = 0
			m.dependencyIndex = 0
			return m, m.loadDependencyMatches
		case (msg.String() == "h" || msg.String() == "left") && m.dependencyFocus != 0:
			switch m.dependencyFocus {
			case 1:
				m.dependencyStateCursor = wrapIndex(m.dependencyStateCursor, -1, len(canonicalSearchStatesOrdered))
			case 2:
				m.dependencyCrossProject = !m.dependencyCrossProject
				return m, m.loadDependencyMatches
			case 3:
				m.dependencyIncludeArchived = !m.dependencyIncludeArchived
				return m, m.loadDependencyMatches
			case 4:
				if m.dependencyIndex > 0 {
					m.dependencyIndex--
				}
			}
			return m, nil
		case (msg.String() == "l" || msg.String() == "right") && m.dependencyFocus != 0:
			switch m.dependencyFocus {
			case 1:
				m.dependencyStateCursor = wrapIndex(m.dependencyStateCursor, 1, len(canonicalSearchStatesOrdered))
			case 2:
				m.dependencyCrossProject = !m.dependencyCrossProject
				return m, m.loadDependencyMatches
			case 3:
				m.dependencyIncludeArchived = !m.dependencyIncludeArchived
				return m, m.loadDependencyMatches
			case 4:
				if m.dependencyIndex < len(m.dependencyMatches)-1 {
					m.dependencyIndex++
				}
			}
			return m, nil
		case (msg.String() == " " || msg.String() == "space") && m.dependencyFocus != 0:
			switch m.dependencyFocus {
			case 1:
				if len(canonicalSearchStatesOrdered) > 0 {
					idx := clamp(m.dependencyStateCursor, 0, len(canonicalSearchStatesOrdered)-1)
					m.toggleDependencyState(canonicalSearchStatesOrdered[idx])
					return m, m.loadDependencyMatches
				}
				return m, nil
			case 2:
				m.dependencyCrossProject = !m.dependencyCrossProject
				return m, m.loadDependencyMatches
			case 3:
				m.dependencyIncludeArchived = !m.dependencyIncludeArchived
				return m, m.loadDependencyMatches
			case 4:
				candidate, ok := m.selectedDependencyCandidate()
				if !ok {
					return m, nil
				}
				m.toggleDependencyCandidateInActiveField(candidate.Match.Task.ID)
				return m, nil
			default:
				return m, nil
			}
		case msg.String() == "x" && m.dependencyFocus != 0:
			if m.dependencyActiveField == taskFieldDependsOn {
				m.dependencyActiveField = taskFieldBlockedBy
			} else {
				m.dependencyActiveField = taskFieldDependsOn
			}
			m.status = "active field: " + m.dependencyActiveFieldLabel()
			return m, nil
		case msg.String() == "d" && m.dependencyFocus == 4:
			candidate, ok := m.selectedDependencyCandidate()
			if !ok {
				return m, nil
			}
			if ownerTaskID := strings.TrimSpace(m.dependencyOwnerTaskID); ownerTaskID != "" && strings.TrimSpace(candidate.Match.Task.ID) == ownerTaskID {
				m.status = "task cannot depend on itself"
				return m, nil
			}
			var added bool
			m.dependencyDependsOn, added = toggleDependencyID(m.dependencyDependsOn, candidate.Match.Task.ID)
			m.dependencyDirty = true
			if added {
				m.status = "added dependency"
			} else {
				m.status = "removed dependency"
			}
			return m, nil
		case msg.String() == "b" && m.dependencyFocus == 4:
			candidate, ok := m.selectedDependencyCandidate()
			if !ok {
				return m, nil
			}
			if ownerTaskID := strings.TrimSpace(m.dependencyOwnerTaskID); ownerTaskID != "" && strings.TrimSpace(candidate.Match.Task.ID) == ownerTaskID {
				m.status = "task cannot depend on itself"
				return m, nil
			}
			var added bool
			m.dependencyBlockedBy, added = toggleDependencyID(m.dependencyBlockedBy, candidate.Match.Task.ID)
			m.dependencyDirty = true
			if added {
				m.status = "added blocker"
			} else {
				m.status = "removed blocker"
			}
			return m, nil
		case msg.String() == "a" && m.dependencyFocus != 0:
			return m.applyDependencyInspector()
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			switch m.dependencyFocus {
			case 1:
				if len(canonicalSearchStatesOrdered) > 0 {
					idx := clamp(m.dependencyStateCursor, 0, len(canonicalSearchStatesOrdered)-1)
					m.toggleDependencyState(canonicalSearchStatesOrdered[idx])
					return m, m.loadDependencyMatches
				}
				return m, nil
			case 2:
				m.dependencyCrossProject = !m.dependencyCrossProject
				return m, m.loadDependencyMatches
			case 3:
				m.dependencyIncludeArchived = !m.dependencyIncludeArchived
				return m, m.loadDependencyMatches
			case 4:
				return m.jumpToDependencyCandidateTask()
			default:
				return m, nil
			}
		default:
			if m.dependencyFocus != 0 {
				return m, nil
			}
			var cmd tea.Cmd
			before := m.dependencyInput.Value()
			m.dependencyInput, cmd = m.dependencyInput.Update(msg)
			if m.dependencyInput.Value() != before {
				m.dependencyIndex = 0
				return m, m.loadDependencyMatches
			}
			return m, cmd
		}
	}

	if m.mode == modeBootstrapSettings {
		if m.bootstrapFocus == 0 {
			if handled, status := applyClipboardShortcutToInput(msg, &m.bootstrapDisplayInput); handled {
				m.status = status
				return m, nil
			}
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			if m.bootstrapMandatory {
				m.status = "startup setup required"
				return m, nil
			}
			m.mode = modeNone
			m.bootstrapDisplayInput.Blur()
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i":
			return m, m.focusBootstrapField(wrapIndex(m.bootstrapFocus, 1, 3))
		case msg.String() == "shift+tab" || msg.String() == "backtab":
			return m, m.focusBootstrapField(wrapIndex(m.bootstrapFocus, -1, 3))
		case msg.String() == "down":
			if m.bootstrapFocus == 1 {
				if m.bootstrapRootIndex < len(m.bootstrapRoots)-1 {
					m.bootstrapRootIndex++
					return m, nil
				}
			}
			return m, m.focusBootstrapField(wrapIndex(m.bootstrapFocus, 1, 3))
		case msg.String() == "up":
			if m.bootstrapFocus == 1 {
				if m.bootstrapRootIndex > 0 {
					m.bootstrapRootIndex--
					return m, nil
				}
			}
			return m, m.focusBootstrapField(wrapIndex(m.bootstrapFocus, -1, 3))
		case (msg.String() == "j") && m.bootstrapFocus != 0:
			if m.bootstrapFocus == 1 {
				if m.bootstrapRootIndex < len(m.bootstrapRoots)-1 {
					m.bootstrapRootIndex++
					return m, nil
				}
			}
			return m, m.focusBootstrapField(wrapIndex(m.bootstrapFocus, 1, 3))
		case (msg.String() == "k") && m.bootstrapFocus != 0:
			if m.bootstrapFocus == 1 {
				if m.bootstrapRootIndex > 0 {
					m.bootstrapRootIndex--
					return m, nil
				}
			}
			return m, m.focusBootstrapField(wrapIndex(m.bootstrapFocus, -1, 3))
		case (msg.String() == "ctrl+r" || msg.String() == "r") && m.bootstrapFocus == 1:
			return m, m.startResourcePicker("", modeBootstrapSettings)
		case msg.String() == "a" && m.bootstrapFocus == 1:
			return m, m.startResourcePicker("", modeBootstrapSettings)
		case (msg.String() == "d" || msg.String() == "x" || msg.String() == "backspace") && m.bootstrapFocus == 1:
			if !m.removeSelectedBootstrapRoot() {
				m.status = "no default path selected"
				return m, nil
			}
			m.status = "default path cleared"
			return m, nil
		case msg.String() == "ctrl+u" && m.bootstrapFocus == 0:
			m.bootstrapDisplayInput.SetValue("")
			m.bootstrapDisplayInput.CursorEnd()
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			switch m.bootstrapFocus {
			case 1:
				return m, m.startResourcePicker("", modeBootstrapSettings)
			default:
				return m.submitBootstrapSettings()
			}
		default:
			if m.bootstrapFocus != 0 {
				return m, nil
			}
			var cmd tea.Cmd
			m.bootstrapDisplayInput, cmd = m.bootstrapDisplayInput.Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeProjectPicker {
		switch {
		case msg.String() == "esc":
			m.mode = modeNone
			m.status = "cancelled"
			return m, nil
		case msg.String() == "A" || msg.String() == "shift+a":
			m.showArchivedProjects = !m.showArchivedProjects
			if m.showArchivedProjects {
				m.status = "showing archived projects"
			} else {
				m.status = "hiding archived projects"
			}
			return m, m.loadData
		case key.Matches(msg, m.keys.newProject):
			return m, m.startProjectForm(nil)
		case msg.String() == "j" || msg.String() == "down":
			if m.projectPickerIndex < len(m.projects)-1 {
				m.projectPickerIndex++
			}
			return m, nil
		case msg.String() == "k" || msg.String() == "up":
			if m.projectPickerIndex > 0 {
				m.projectPickerIndex--
			}
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			if len(m.projects) == 0 {
				return m, m.startProjectForm(nil)
			}
			m.selectedProject = clamp(m.projectPickerIndex, 0, len(m.projects)-1)
			m.selectedColumn = 0
			m.selectedTask = 0
			m.mode = modeNone
			m.status = "project switched"
			return m, m.loadData
		default:
			return m, nil
		}
	}

	if m.mode == modeSearch {
		const searchFocusSlots = 6
		if m.searchFocus == 0 {
			if handled, status := applyClipboardShortcutToInput(msg, &m.searchInput); handled {
				m.status = status
				m.searchQuery = strings.TrimSpace(m.searchInput.Value())
				return m, nil
			}
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.searchInput.Blur()
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i":
			m.searchFocus = wrapIndex(m.searchFocus, 1, searchFocusSlots)
			if m.searchFocus == 0 {
				return m, m.searchInput.Focus()
			}
			m.searchInput.Blur()
			return m, nil
		case msg.String() == "shift+tab" || msg.String() == "backtab":
			m.searchFocus = wrapIndex(m.searchFocus, -1, searchFocusSlots)
			if m.searchFocus == 0 {
				return m, m.searchInput.Focus()
			}
			m.searchInput.Blur()
			return m, nil
		case msg.String() == "down":
			m.searchFocus = wrapIndex(m.searchFocus, 1, searchFocusSlots)
			if m.searchFocus == 0 {
				return m, m.searchInput.Focus()
			}
			m.searchInput.Blur()
			return m, nil
		case msg.String() == "up":
			m.searchFocus = wrapIndex(m.searchFocus, -1, searchFocusSlots)
			if m.searchFocus == 0 {
				return m, m.searchInput.Focus()
			}
			m.searchInput.Blur()
			return m, nil
		case msg.String() == "j" && m.searchFocus != 0:
			m.searchFocus = wrapIndex(m.searchFocus, 1, searchFocusSlots)
			if m.searchFocus == 0 {
				return m, m.searchInput.Focus()
			}
			m.searchInput.Blur()
			return m, nil
		case msg.String() == "k" && m.searchFocus != 0:
			m.searchFocus = wrapIndex(m.searchFocus, -1, searchFocusSlots)
			if m.searchFocus == 0 {
				return m, m.searchInput.Focus()
			}
			m.searchInput.Blur()
			return m, nil
		case msg.String() == "ctrl+p" && m.searchFocus != 0:
			m.searchCrossProject = !m.searchCrossProject
			return m, nil
		case msg.String() == "ctrl+a" && m.searchFocus != 0:
			m.searchIncludeArchived = !m.searchIncludeArchived
			return m, nil
		case msg.String() == "ctrl+u" && m.searchFocus != 0:
			return m, m.clearSearchQuery()
		case msg.String() == "ctrl+r" && m.searchFocus != 0:
			return m, m.resetSearchFilters()
		case (msg.String() == "h" || msg.String() == "left") && m.searchFocus != 0:
			switch m.searchFocus {
			case 1:
				m.searchStateCursor = wrapIndex(m.searchStateCursor, -1, len(canonicalSearchStatesOrdered))
			case 2:
				m.searchLevelCursor = wrapIndex(m.searchLevelCursor, -1, len(canonicalSearchLevelsOrdered))
			case 3:
				m.searchCrossProject = !m.searchCrossProject
			case 4:
				m.searchIncludeArchived = !m.searchIncludeArchived
			}
			return m, nil
		case (msg.String() == "l" || msg.String() == "right") && m.searchFocus != 0:
			switch m.searchFocus {
			case 1:
				m.searchStateCursor = wrapIndex(m.searchStateCursor, 1, len(canonicalSearchStatesOrdered))
			case 2:
				m.searchLevelCursor = wrapIndex(m.searchLevelCursor, 1, len(canonicalSearchLevelsOrdered))
			case 3:
				m.searchCrossProject = !m.searchCrossProject
			case 4:
				m.searchIncludeArchived = !m.searchIncludeArchived
			}
			return m, nil
		case (msg.String() == " " || msg.String() == "space") && m.searchFocus != 0:
			switch m.searchFocus {
			case 1:
				if len(canonicalSearchStatesOrdered) > 0 {
					idx := clamp(m.searchStateCursor, 0, len(canonicalSearchStatesOrdered)-1)
					m.toggleSearchState(canonicalSearchStatesOrdered[idx])
				}
			case 2:
				if len(canonicalSearchLevelsOrdered) > 0 {
					idx := clamp(m.searchLevelCursor, 0, len(canonicalSearchLevelsOrdered)-1)
					m.toggleSearchLevel(canonicalSearchLevelsOrdered[idx])
				}
			case 3:
				m.searchCrossProject = !m.searchCrossProject
			case 4:
				m.searchIncludeArchived = !m.searchIncludeArchived
			}
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			switch m.searchFocus {
			case 1:
				if len(canonicalSearchStatesOrdered) > 0 {
					idx := clamp(m.searchStateCursor, 0, len(canonicalSearchStatesOrdered)-1)
					m.toggleSearchState(canonicalSearchStatesOrdered[idx])
				}
				return m, nil
			case 2:
				if len(canonicalSearchLevelsOrdered) > 0 {
					idx := clamp(m.searchLevelCursor, 0, len(canonicalSearchLevelsOrdered)-1)
					m.toggleSearchLevel(canonicalSearchLevelsOrdered[idx])
				}
				return m, nil
			case 3:
				m.searchCrossProject = !m.searchCrossProject
				return m, nil
			case 4:
				m.searchIncludeArchived = !m.searchIncludeArchived
				return m, nil
			default:
				return m, m.applySearchFilter()
			}
		default:
			if m.searchFocus == 0 {
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.searchQuery = strings.TrimSpace(m.searchInput.Value())
				return m, cmd
			} else {
				return m, nil
			}
		}
	}

	if m.mode == modeSearchResults {
		switch msg.String() {
		case "esc":
			m.mode = modeNone
			m.status = "ready"
			return m, nil
		case "j", "down":
			if m.searchResultIndex < len(m.searchMatches)-1 {
				m.searchResultIndex++
			}
			return m, nil
		case "k", "up":
			if m.searchResultIndex > 0 {
				m.searchResultIndex--
			}
			return m, nil
		case "enter":
			if len(m.searchMatches) == 0 {
				m.mode = modeNone
				m.status = "no matches"
				return m, nil
			}
			match := m.searchMatches[clamp(m.searchResultIndex, 0, len(m.searchMatches)-1)]
			for idx, project := range m.projects {
				if project.ID == match.Project.ID {
					m.selectedProject = idx
					break
				}
			}
			m.pendingFocusTaskID = match.Task.ID
			m.mode = modeNone
			m.status = "jumped to match"
			return m, m.loadData
		default:
			return m, nil
		}
	}

	if m.mode == modeCommandPalette {
		if handled, status := applyClipboardShortcutToInput(msg, &m.commandInput); handled {
			m.status = status
			m.commandMatches = m.filteredCommandItems(m.commandInput.Value())
			m.commandIndex = clamp(m.commandIndex, 0, len(m.commandMatches)-1)
			return m, nil
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.commandInput.Blur()
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab":
			if len(m.commandMatches) == 0 {
				return m, nil
			}
			m.commandInput.SetValue(m.commandMatches[0].Command)
			m.commandInput.CursorEnd()
			m.commandMatches = m.filteredCommandItems(m.commandInput.Value())
			m.commandIndex = 0
			return m, nil
		case msg.String() == "j" || msg.String() == "down":
			if len(m.commandMatches) > 0 && m.commandIndex < len(m.commandMatches)-1 {
				m.commandIndex++
			}
			return m, nil
		case msg.String() == "k" || msg.String() == "up":
			if m.commandIndex > 0 {
				m.commandIndex--
			}
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			cmd := m.commandToExecute()
			m.mode = modeNone
			m.commandInput.Blur()
			return m.executeCommandPalette(cmd)
		default:
			var cmd tea.Cmd
			m.commandInput, cmd = m.commandInput.Update(msg)
			m.commandMatches = m.filteredCommandItems(m.commandInput.Value())
			m.commandIndex = clamp(m.commandIndex, 0, len(m.commandMatches)-1)
			return m, cmd
		}
	}

	if m.mode == modeConfirmAction {
		switch msg.String() {
		case "esc", "n":
			m.mode = modeNone
			m.pendingConfirm = confirmAction{}
			m.status = "cancelled"
			return m, nil
		case "h", "left", "l", "right":
			if m.confirmChoice == 0 {
				m.confirmChoice = 1
			} else {
				m.confirmChoice = 0
			}
			return m, nil
		case "y":
			m.confirmChoice = 0
			m.mode = modeNone
			action := m.pendingConfirm
			m.pendingConfirm = confirmAction{}
			m.status = "applying action..."
			return m.applyConfirmedAction(action)
		case "enter":
			if m.confirmChoice == 1 {
				m.mode = modeNone
				m.pendingConfirm = confirmAction{}
				m.status = "cancelled"
				return m, nil
			}
			m.mode = modeNone
			action := m.pendingConfirm
			m.pendingConfirm = confirmAction{}
			m.status = "applying action..."
			return m.applyConfirmedAction(action)
		default:
			return m, nil
		}
	}

	if m.mode == modeWarning {
		switch msg.String() {
		case "esc", "enter":
			m.closeWarningModal()
			return m, nil
		default:
			return m, nil
		}
	}

	if m.mode == modeQuickActions {
		actions := m.quickActions()
		switch msg.String() {
		case "esc":
			m.mode = modeNone
			m.status = "cancelled"
			return m, nil
		case "j", "down":
			if m.quickActionIndex < len(actions)-1 {
				m.quickActionIndex++
			}
			return m, nil
		case "k", "up":
			if m.quickActionIndex > 0 {
				m.quickActionIndex--
			}
			return m, nil
		case "enter":
			return m.applyQuickAction()
		default:
			return m, nil
		}
	}

	if m.mode == modeDuePicker {
		if m.duePickerFocus == 1 {
			if handled, status := applyClipboardShortcutToInput(msg, &m.duePickerDateInput); handled {
				m.status = status
				m.duePicker = 0
				return m, nil
			}
		}
		if m.duePickerFocus == 2 {
			if handled, status := applyClipboardShortcutToInput(msg, &m.duePickerTimeInput); handled {
				m.status = status
				m.duePicker = 0
				return m, nil
			}
		}
		options := m.duePickerOptions()
		switch msg.String() {
		case "esc":
			m.mode = m.pickerBack
			m.pickerBack = modeNone
			m.status = "due picker cancelled"
			return m, m.focusTaskFormField(taskFieldDue)
		case "tab", "ctrl+i":
			return m, m.cycleDuePickerFocus(1)
		case "shift+tab", "backtab":
			return m, m.cycleDuePickerFocus(-1)
		case "h", "left":
			if m.duePickerFocus == 0 {
				return m, m.setDuePickerIncludeTime(false)
			}
			return m, nil
		case "l", "right":
			if m.duePickerFocus == 0 {
				return m, m.setDuePickerIncludeTime(true)
			}
			return m, nil
		case " ", "space":
			if m.duePickerFocus == 0 || m.duePickerFocus == 1 || m.duePickerFocus == 2 {
				return m, m.setDuePickerIncludeTime(!m.duePickerIncludeTime)
			}
			return m, nil
		case "j", "down":
			if m.duePickerFocus == 3 {
				if m.duePicker < len(options)-1 {
					m.duePicker++
				}
				return m, nil
			}
			m.duePickerDateInput.Blur()
			m.duePickerTimeInput.Blur()
			m.duePickerFocus = 3
			if m.duePicker < len(options)-1 {
				m.duePicker++
			}
			return m, nil
		case "k", "up":
			if m.duePickerFocus == 3 {
				if m.duePicker > 0 {
					m.duePicker--
				}
				return m, nil
			}
			m.duePickerDateInput.Blur()
			m.duePickerTimeInput.Blur()
			m.duePickerFocus = 3
			if m.duePicker > 0 {
				m.duePicker--
			}
			return m, nil
		case "ctrl+u":
			switch m.duePickerFocus {
			case 1:
				m.duePickerDateInput.SetValue("")
				m.duePickerDateInput.CursorEnd()
				m.duePicker = 0
			case 2:
				m.duePickerTimeInput.SetValue("")
				m.duePickerTimeInput.CursorEnd()
				m.duePicker = 0
			}
			return m, nil
		case "enter":
			if m.duePickerFocus == 0 {
				return m, m.setDuePickerIncludeTime(!m.duePickerIncludeTime)
			}
			if len(options) == 0 || len(m.formInputs) <= taskFieldDue {
				m.mode = m.pickerBack
				m.pickerBack = modeNone
				return m, m.focusTaskFormField(taskFieldDue)
			}
			choice := options[clamp(m.duePicker, 0, len(options)-1)]
			m.formInputs[taskFieldDue].SetValue(choice.Value)
			m.mode = m.pickerBack
			m.pickerBack = modeNone
			m.status = "due updated"
			return m, m.focusTaskFormField(taskFieldDue)
		default:
			switch m.duePickerFocus {
			case 1:
				var cmd tea.Cmd
				before := m.duePickerDateInput.Value()
				m.duePickerDateInput, cmd = m.duePickerDateInput.Update(msg)
				if m.duePickerDateInput.Value() != before {
					m.duePicker = 0
				}
				return m, cmd
			case 2:
				if !m.duePickerIncludeTime {
					return m, m.focusDuePickerSlot(3)
				}
				var cmd tea.Cmd
				before := m.duePickerTimeInput.Value()
				m.duePickerTimeInput, cmd = m.duePickerTimeInput.Update(msg)
				if m.duePickerTimeInput.Value() != before {
					m.duePicker = 0
				}
				return m, cmd
			default:
				return m, nil
			}
		}
	}

	if m.mode == modeResourcePicker {
		if handled, status := applyClipboardShortcutToInput(msg, &m.resourcePickerFilter); handled {
			m.status = status
			m.resourcePickerIndex = 0
			return m, nil
		}
		if msg.Text != "" && (msg.Mod&tea.ModCtrl) == 0 {
			var cmd tea.Cmd
			before := m.resourcePickerFilter.Value()
			m.resourcePickerFilter, cmd = m.resourcePickerFilter.Update(msg)
			if m.resourcePickerFilter.Value() != before {
				m.resourcePickerIndex = 0
			}
			return m, cmd
		}
		items := m.visibleResourcePickerItems()
		switch msg.String() {
		case "esc":
			m.mode = m.resourcePickerBack
			m.resourcePickerFilter.Blur()
			m.resourcePickerFilter.SetValue("")
			m.status = "resource picker cancelled"
			return m, nil
		case "down":
			if m.resourcePickerIndex < len(items)-1 {
				m.resourcePickerIndex++
			}
			return m, nil
		case "up":
			if m.resourcePickerIndex > 0 {
				m.resourcePickerIndex--
			}
			return m, nil
		case "left":
			return m, m.openResourcePickerParent()
		case "backspace":
			var cmd tea.Cmd
			m.resourcePickerFilter, cmd = m.resourcePickerFilter.Update(msg)
			m.resourcePickerIndex = 0
			return m, cmd
		case "ctrl+u":
			m.resourcePickerFilter.SetValue("")
			m.resourcePickerFilter.CursorEnd()
			m.resourcePickerIndex = 0
			return m, nil
		case "right":
			entry, ok := m.selectedResourcePickerEntry()
			if !ok || !entry.IsDir {
				return m, nil
			}
			return m, m.openResourcePickerDir(entry.Path)
		case "ctrl+a":
			if m.resourcePickerBack == modeAddProject || m.resourcePickerBack == modeEditProject || m.resourcePickerBack == modePathsRoots || m.resourcePickerBack == modeBootstrapSettings {
				return m, m.attachCurrentResourcePickerDir()
			}
			return m, m.attachSelectedResourceEntry()
		case "enter":
			if m.resourcePickerBack == modeAddProject || m.resourcePickerBack == modeEditProject || m.resourcePickerBack == modePathsRoots || m.resourcePickerBack == modeBootstrapSettings {
				entry, ok := m.selectedResourcePickerEntry()
				if !ok {
					return m, m.attachCurrentResourcePickerDir()
				}
				return m, m.attachResourcePickerEntry(entry)
			}
			entry, ok := m.selectedResourcePickerEntry()
			if ok && entry.IsDir {
				return m, m.openResourcePickerDir(entry.Path)
			}
			return m, m.attachSelectedResourceEntry()
		default:
			var cmd tea.Cmd
			before := m.resourcePickerFilter.Value()
			m.resourcePickerFilter, cmd = m.resourcePickerFilter.Update(msg)
			if m.resourcePickerFilter.Value() != before {
				m.resourcePickerIndex = 0
			}
			return m, cmd
		}
	}

	if m.mode == modeLabelPicker {
		if handled, status := applyClipboardShortcutToInput(msg, &m.labelPickerInput); handled {
			m.status = status
			m.labelPickerIndex = 0
			m.refreshLabelPickerMatches()
			return m, nil
		}
		if msg.Text != "" && (msg.Mod&tea.ModCtrl) == 0 {
			var cmd tea.Cmd
			before := m.labelPickerInput.Value()
			m.labelPickerInput, cmd = m.labelPickerInput.Update(msg)
			if m.labelPickerInput.Value() != before {
				m.labelPickerIndex = 0
				m.refreshLabelPickerMatches()
			}
			return m, cmd
		}
		switch msg.String() {
		case "esc":
			m.mode = m.labelPickerBack
			m.labelPickerInput.Blur()
			m.status = "label picker cancelled"
			if m.mode == modeAddTask || m.mode == modeEditTask {
				return m, m.focusTaskFormField(taskFieldLabels)
			}
			return m, nil
		case "ctrl+u":
			m.labelPickerInput.SetValue("")
			m.labelPickerInput.CursorEnd()
			m.labelPickerIndex = 0
			m.refreshLabelPickerMatches()
			return m, nil
		case "j", "down":
			if m.labelPickerIndex < len(m.labelPickerItems)-1 {
				m.labelPickerIndex++
			}
			return m, nil
		case "k", "up":
			if m.labelPickerIndex > 0 {
				m.labelPickerIndex--
			}
			return m, nil
		case "enter":
			if len(m.labelPickerItems) == 0 || len(m.formInputs) <= taskFieldLabels {
				m.mode = m.labelPickerBack
				m.labelPickerInput.Blur()
				return m, m.focusTaskFormField(taskFieldLabels)
			}
			item := m.labelPickerItems[clamp(m.labelPickerIndex, 0, len(m.labelPickerItems)-1)]
			m.appendTaskFormLabel(item.Label)
			m.mode = m.labelPickerBack
			m.labelPickerInput.Blur()
			m.status = "label added"
			return m, m.focusTaskFormField(taskFieldLabels)
		default:
			var cmd tea.Cmd
			before := m.labelPickerInput.Value()
			m.labelPickerInput, cmd = m.labelPickerInput.Update(msg)
			if m.labelPickerInput.Value() != before {
				m.labelPickerIndex = 0
				m.refreshLabelPickerMatches()
			}
			return m, cmd
		}
	}

	if m.mode == modePathsRoots {
		if handled, status := applyClipboardShortcutToInput(msg, &m.pathsRootInput); handled {
			m.status = status
			return m, nil
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.pathsRootInput.Blur()
			m.status = "paths/roots cancelled"
			return m, nil
		case msg.String() == "ctrl+r" || msg.String() == "r":
			return m, m.startResourcePicker("", modePathsRoots)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitPathsRoots()
		default:
			var cmd tea.Cmd
			m.pathsRootInput, cmd = m.pathsRootInput.Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeAddTask || m.mode == modeEditTask {
		if len(m.formInputs) > 0 && m.formFocus >= 0 && m.formFocus < len(m.formInputs) && m.formFocus != taskFieldPriority {
			if handled, status := applyClipboardShortcutToInput(msg, &m.formInputs[m.formFocus]); handled {
				m.status = status
				return m, nil
			}
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.formInputs = nil
			m.formFocus = 0
			m.editingTaskID = ""
			m.taskFormParentID = ""
			m.taskFormKind = domain.WorkKindTask
			m.taskFormScope = domain.KindAppliesToTask
			m.taskFormResourceRefs = nil
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i" || msg.String() == "down":
			return m, m.focusTaskFormField(m.formFocus + 1)
		case msg.String() == "shift+tab" || msg.String() == "backtab" || msg.String() == "up":
			return m, m.focusTaskFormField(m.formFocus - 1)
		case msg.String() == "ctrl+l":
			if m.formFocus == taskFieldLabels {
				return m, m.startLabelPicker()
			}
			return m, nil
		case msg.String() == "ctrl+o" || (msg.String() == "o" && (m.formFocus == taskFieldDependsOn || m.formFocus == taskFieldBlockedBy)):
			if m.formFocus == taskFieldDependsOn || m.formFocus == taskFieldBlockedBy {
				return m, m.startDependencyInspectorFromForm(m.formFocus)
			}
			return m, nil
		case msg.String() == "ctrl+s":
			return m, m.startSubtaskFormFromTaskForm()
		case isCtrlG(msg):
			if m.formFocus == taskFieldLabels {
				if m.acceptCurrentLabelSuggestion() {
					m.status = "accepted label suggestion"
				} else {
					m.status = "no label suggestion"
				}
			}
			return m, nil
		case msg.String() == "ctrl+r":
			back := m.mode
			taskID := ""
			if back == modeEditTask {
				taskID = strings.TrimSpace(m.editingTaskID)
				if taskID == "" {
					task, ok := m.selectedTaskInCurrentColumn()
					if !ok {
						m.status = "no task selected"
						return m, nil
					}
					taskID = task.ID
				}
			}
			return m, m.startResourcePicker(taskID, back)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			if m.formFocus == taskFieldLabels {
				return m, m.startLabelPicker()
			}
			if m.formFocus == taskFieldDependsOn || m.formFocus == taskFieldBlockedBy {
				return m, m.startDependencyInspectorFromForm(m.formFocus)
			}
			return m.submitInputMode()
		default:
			if m.formFocus == taskFieldPriority {
				switch msg.String() {
				case "h", "left":
					m.cyclePriority(-1)
					return m, nil
				case "l", "right":
					m.cyclePriority(1)
					return m, nil
				}
				return m, nil
			}
			if m.formFocus == taskFieldDue && (msg.String() == "d" || msg.String() == "ctrl+d" || msg.String() == "D") {
				m.startDuePicker()
				m.status = "due picker"
				return m, nil
			}
			if len(m.formInputs) == 0 {
				return m, nil
			}
			var cmd tea.Cmd
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeAddProject || m.mode == modeEditProject {
		if len(m.projectFormInputs) > 0 && m.projectFormFocus >= 0 && m.projectFormFocus < len(m.projectFormInputs) {
			if handled, status := applyClipboardShortcutToInput(msg, &m.projectFormInputs[m.projectFormFocus]); handled {
				m.status = status
				return m, nil
			}
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.projectFormInputs = nil
			m.projectFormFocus = 0
			m.editingProjectID = ""
			m.status = "cancelled"
			return m, nil
		case (msg.String() == "ctrl+r" || msg.String() == "r") && m.projectFormFocus == projectFieldRootPath:
			return m, m.startResourcePicker("", m.mode)
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i" || msg.String() == "down":
			return m, m.focusProjectFormField(m.projectFormFocus + 1)
		case msg.String() == "shift+tab" || msg.String() == "backtab" || msg.String() == "up":
			return m, m.focusProjectFormField(m.projectFormFocus - 1)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitInputMode()
		default:
			if len(m.projectFormInputs) == 0 {
				return m, nil
			}
			var cmd tea.Cmd
			m.projectFormInputs[m.projectFormFocus], cmd = m.projectFormInputs[m.projectFormFocus].Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeLabelsConfig {
		if len(m.labelsConfigInputs) > 0 && m.labelsConfigFocus >= 0 && m.labelsConfigFocus < len(m.labelsConfigInputs) {
			if handled, status := applyClipboardShortcutToInput(msg, &m.labelsConfigInputs[m.labelsConfigFocus]); handled {
				m.status = status
				return m, nil
			}
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.labelsConfigInputs = nil
			m.labelsConfigFocus = 0
			m.labelsConfigSlug = ""
			m.labelsConfigBranchTaskID = ""
			m.labelsConfigPhaseTaskID = ""
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i" || msg.String() == "down":
			return m, m.focusLabelsConfigField(m.labelsConfigFocus + 1)
		case msg.String() == "shift+tab" || msg.String() == "backtab" || msg.String() == "up":
			return m, m.focusLabelsConfigField(m.labelsConfigFocus - 1)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitInputMode()
		default:
			if len(m.labelsConfigInputs) == 0 {
				return m, nil
			}
			var cmd tea.Cmd
			m.labelsConfigInputs[m.labelsConfigFocus], cmd = m.labelsConfigInputs[m.labelsConfigFocus].Update(msg)
			return m, cmd
		}
	}

	if m.mode == modeHighlightColor {
		if handled, status := applyClipboardShortcutToInput(msg, &m.highlightColorInput); handled {
			m.status = status
			return m, nil
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.highlightColorInput.Blur()
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m.submitInputMode()
		default:
			var cmd tea.Cmd
			m.highlightColorInput, cmd = m.highlightColorInput.Update(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case "ctrl+c", "meta+c", "super+c":
		if err := copyTextToClipboard(m.input); err != nil {
			m.status = "copy failed: " + err.Error()
		} else {
			m.status = "copied field value"
		}
		return m, nil
	case "ctrl+v", "meta+v", "super+v":
		text, err := pasteTextFromClipboard()
		if err != nil {
			m.status = "paste failed: " + err.Error()
			return m, nil
		}
		if text == "" {
			m.status = "clipboard is empty"
			return m, nil
		}
		m.input += text
		m.status = "pasted from clipboard"
		return m, nil
	case "esc":
		m.mode = modeNone
		m.input = ""
		m.editingTaskID = ""
		m.status = "cancelled"
		return m, nil
	case "backspace":
		if m.input != "" {
			_, size := utf8.DecodeLastRuneInString(m.input)
			m.input = m.input[:len(m.input)-size]
		}
		return m, nil
	case "enter":
		return m.submitInputMode()
	default:
		if msg.Text != "" {
			m.input += msg.Text
		}
		return m, nil
	}
}

// isCtrlG reports whether a keypress represents the Ctrl+G autocomplete shortcut.
func isCtrlG(msg tea.KeyPressMsg) bool {
	if msg.String() == "ctrl+g" {
		return true
	}
	if (msg.Mod & tea.ModCtrl) == 0 {
		return false
	}
	if msg.Code == 'g' || msg.Code == 'G' {
		return true
	}
	return strings.EqualFold(msg.Text, "g")
}

// isClipboardCopyKey reports whether a keypress is a platform clipboard-copy shortcut.
func isClipboardCopyKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "ctrl+c", "meta+c", "super+c":
		return true
	default:
		return false
	}
}

// isClipboardPasteKey reports whether a keypress is a platform clipboard-paste shortcut.
func isClipboardPasteKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "ctrl+v", "meta+v", "super+v":
		return true
	default:
		return false
	}
}

// copyTextToClipboard writes plain text to the system clipboard.
func copyTextToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

// pasteTextFromClipboard reads plain text from the system clipboard.
func pasteTextFromClipboard() (string, error) {
	return clipboard.ReadAll()
}

// applyClipboardShortcutToInput handles copy/paste shortcuts for one text input.
func applyClipboardShortcutToInput(msg tea.KeyPressMsg, in *textinput.Model) (bool, string) {
	if in == nil {
		return false, ""
	}
	switch {
	case isClipboardCopyKey(msg):
		if err := copyTextToClipboard(in.Value()); err != nil {
			return true, "copy failed: " + err.Error()
		}
		return true, "copied field value"
	case isClipboardPasteKey(msg):
		text, err := pasteTextFromClipboard()
		if err != nil {
			return true, "paste failed: " + err.Error()
		}
		if text == "" {
			return true, "clipboard is empty"
		}
		value := in.Value()
		pos := clamp(in.Position(), 0, utf8.RuneCountInString(value))
		merged, nextPos := spliceRunes(value, pos, text)
		in.SetValue(merged)
		in.SetCursor(nextPos)
		return true, "pasted from clipboard"
	default:
		return false, ""
	}
}

// spliceRunes inserts text at one rune index and returns merged value + next cursor position.
func spliceRunes(value string, runePos int, insert string) (string, int) {
	valueRunes := []rune(value)
	insertRunes := []rune(insert)
	runePos = clamp(runePos, 0, len(valueRunes))
	merged := string(valueRunes[:runePos]) + string(insertRunes) + string(valueRunes[runePos:])
	return merged, runePos + len(insertRunes)
}

// submitInputMode submits input mode.
func (m Model) submitInputMode() (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeAddTask:
		if text := strings.TrimSpace(m.input); text != "" {
			vals := m.taskFormValues()
			if vals["title"] == "" {
				vals["title"] = text
			}
		}
		vals := m.taskFormValues()
		title := vals["title"]
		if title == "" {
			m.mode = modeNone
			m.formInputs = nil
			m.input = ""
			m.status = "title required"
			return m, nil
		}
		priority := domain.Priority(strings.ToLower(vals["priority"]))
		if priority == "" {
			priority = domain.PriorityMedium
		}
		switch priority {
		case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
		default:
			m.status = "priority must be low|medium|high"
			return m, nil
		}
		dueAt, err := parseDueInput(vals["due"], nil)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		labels := parseLabelsInput(vals["labels"], nil)
		if err := m.validateAllowedLabels(labels); err != nil {
			m.status = err.Error()
			return m, nil
		}
		metadata := m.buildTaskMetadataFromForm(vals, domain.TaskMetadata{})
		parentID := m.taskFormParentID
		kind := m.taskFormKind
		scope := m.taskFormScope

		m.mode = modeNone
		m.formInputs = nil
		m.taskFormParentID = ""
		m.taskFormKind = domain.WorkKindTask
		m.taskFormScope = domain.KindAppliesToTask
		m.taskFormResourceRefs = nil
		return m.createTask(app.CreateTaskInput{
			ParentID:    parentID,
			Kind:        kind,
			Scope:       scope,
			Title:       title,
			Description: vals["description"],
			Priority:    priority,
			DueAt:       dueAt,
			Labels:      labels,
			Metadata:    metadata,
		})
	case modeSearch:
		return m, m.applySearchFilter()
	case modeRenameTask:
		text := strings.TrimSpace(m.input)
		m.mode = modeNone
		m.input = ""
		if text == "" {
			m.status = "title required"
			return m, nil
		}
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		taskID := task.ID
		return m, func() tea.Msg {
			_, err := m.svc.RenameTask(context.Background(), taskID, text)
			if err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "task renamed", reload: true}
		}
	case modeEditTask:
		vals := m.taskFormValues()
		taskID := m.editingTaskID
		if taskID == "" {
			task, ok := m.selectedTaskInCurrentColumn()
			if !ok {
				m.status = "no task selected"
				return m, nil
			}
			taskID = task.ID
		}
		task, ok := m.taskByID(taskID)
		if !ok {
			m.status = "task not found"
			return m, nil
		}

		if text := strings.TrimSpace(m.input); text != "" {
			in, err := parseTaskEditInput(text, task)
			if err != nil {
				m.status = "invalid edit format: " + err.Error()
				return m, nil
			}
			m.mode = modeNone
			m.formInputs = nil
			m.input = ""
			m.editingTaskID = ""
			m.taskFormResourceRefs = nil
			in.TaskID = taskID
			return m, func() tea.Msg {
				_, updateErr := m.svc.UpdateTask(context.Background(), in)
				if updateErr != nil {
					return actionMsg{err: updateErr}
				}
				return actionMsg{status: "task updated", reload: true}
			}
		}

		title := vals["title"]
		if title == "" {
			title = task.Title
		}
		description := vals["description"]
		if description == "" {
			description = task.Description
		}

		priority := domain.Priority(strings.ToLower(vals["priority"]))
		if priority == "" {
			priority = task.Priority
		}
		switch priority {
		case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
		default:
			m.status = "priority must be low|medium|high"
			return m, nil
		}

		dueAt, err := parseDueInput(vals["due"], task.DueAt)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		labels := parseLabelsInput(vals["labels"], task.Labels)
		if err := m.validateAllowedLabels(labels); err != nil {
			m.status = err.Error()
			return m, nil
		}
		metadata := m.buildTaskMetadataFromForm(vals, task.Metadata)

		m.mode = modeNone
		m.formInputs = nil
		m.editingTaskID = ""
		m.taskFormParentID = ""
		m.taskFormKind = domain.WorkKindTask
		m.taskFormScope = domain.KindAppliesToTask
		m.taskFormResourceRefs = nil
		in := app.UpdateTaskInput{
			TaskID:      taskID,
			Title:       title,
			Description: description,
			Priority:    priority,
			DueAt:       dueAt,
			Labels:      labels,
			Metadata:    &metadata,
		}
		return m, func() tea.Msg {
			_, updateErr := m.svc.UpdateTask(context.Background(), in)
			if updateErr != nil {
				return actionMsg{err: updateErr}
			}
			return actionMsg{status: "task updated", reload: true}
		}
	case modeLabelsConfig:
		if len(m.labelsConfigInputs) < 4 {
			m.status = "labels config unavailable"
			return m, nil
		}
		slug := strings.TrimSpace(strings.ToLower(m.labelsConfigSlug))
		if slug == "" {
			m.status = "project slug is empty"
			return m, nil
		}
		if m.saveLabels == nil {
			m.status = "save labels failed: callback unavailable"
			return m, nil
		}
		globalLabels := normalizeConfigLabels(parseLabelsInput(m.labelsConfigInputs[0].Value(), nil))
		projectLabels := normalizeConfigLabels(parseLabelsInput(m.labelsConfigInputs[1].Value(), nil))
		branchLabels := normalizeConfigLabels(parseLabelsInput(m.labelsConfigInputs[2].Value(), nil))
		phaseLabels := normalizeConfigLabels(parseLabelsInput(m.labelsConfigInputs[3].Value(), nil))
		branchTaskID := strings.TrimSpace(m.labelsConfigBranchTaskID)
		phaseTaskID := strings.TrimSpace(m.labelsConfigPhaseTaskID)

		m.allowedLabelGlobal = append([]string(nil), globalLabels...)
		if len(projectLabels) == 0 {
			delete(m.allowedLabelProject, slug)
		} else {
			m.allowedLabelProject[slug] = append([]string(nil), projectLabels...)
		}
		m.refreshTaskFormLabelSuggestions()
		m.mode = modeNone
		m.labelsConfigInputs = nil
		m.labelsConfigFocus = 0
		m.labelsConfigSlug = ""
		m.labelsConfigBranchTaskID = ""
		m.labelsConfigPhaseTaskID = ""
		return m, func() tea.Msg {
			if err := m.saveLabels(slug, globalLabels, projectLabels); err != nil {
				return actionMsg{err: err}
			}
			updateTaskLabels := func(taskID string, labels []string) error {
				taskID = strings.TrimSpace(taskID)
				if taskID == "" {
					return nil
				}
				task, ok := m.taskByID(taskID)
				if !ok {
					return nil
				}
				if slices.Equal(normalizeConfigLabels(task.Labels), normalizeConfigLabels(labels)) {
					return nil
				}
				_, err := m.svc.UpdateTask(context.Background(), app.UpdateTaskInput{
					TaskID:      task.ID,
					Title:       task.Title,
					Description: task.Description,
					Priority:    task.Priority,
					DueAt:       task.DueAt,
					Labels:      append([]string(nil), labels...),
					Metadata:    &task.Metadata,
				})
				return err
			}
			if err := updateTaskLabels(branchTaskID, branchLabels); err != nil {
				return actionMsg{err: err}
			}
			if err := updateTaskLabels(phaseTaskID, phaseLabels); err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "labels config saved"}
		}
	case modeHighlightColor:
		value := strings.TrimSpace(m.highlightColorInput.Value())
		if value == "" {
			value = defaultHighlightColor
		}
		m.highlightColor = value
		m.mode = modeNone
		m.highlightColorInput.Blur()
		m.status = "highlight color updated"
		return m, nil
	case modeAddProject, modeEditProject:
		isAdd := m.mode == modeAddProject
		vals := m.projectFormValues()
		name := vals["name"]
		if name == "" {
			m.status = "project name required"
			return m, nil
		}
		rootPath, err := normalizeProjectRootPathInput(vals["root_path"])
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		metadata := domain.ProjectMetadata{
			Owner:    vals["owner"],
			Icon:     vals["icon"],
			Color:    vals["color"],
			Homepage: vals["homepage"],
			Tags:     parseLabelsInput(vals["tags"], nil),
		}
		description := vals["description"]
		projectID := m.editingProjectID
		m.mode = modeNone
		m.projectFormInputs = nil
		m.projectFormFocus = 0
		m.editingProjectID = ""
		if isAdd || projectID == "" {
			return m, func() tea.Msg {
				project, err := m.svc.CreateProjectWithMetadata(context.Background(), app.CreateProjectInput{
					Name:        name,
					Description: description,
					Metadata:    metadata,
				})
				if err != nil {
					return actionMsg{err: err}
				}
				if m.saveProjectRoot != nil {
					if err := m.saveProjectRoot(project.Slug, rootPath); err != nil {
						return actionMsg{err: err}
					}
				}
				return actionMsg{
					status:          "project created",
					reload:          true,
					projectID:       project.ID,
					projectRootSlug: project.Slug,
					projectRootPath: rootPath,
				}
			}
		}
		return m, func() tea.Msg {
			project, err := m.svc.UpdateProject(context.Background(), app.UpdateProjectInput{
				ProjectID:   projectID,
				Name:        name,
				Description: description,
				Metadata:    metadata,
			})
			if err != nil {
				return actionMsg{err: err}
			}
			if m.saveProjectRoot != nil {
				if err := m.saveProjectRoot(project.Slug, rootPath); err != nil {
					return actionMsg{err: err}
				}
			}
			return actionMsg{
				status:          "project updated",
				reload:          true,
				projectID:       project.ID,
				projectRootSlug: project.Slug,
				projectRootPath: rootPath,
			}
		}
	default:
		return m, nil
	}
}

// executeCommandPalette executes command palette.
func (m Model) executeCommandPalette(command string) (tea.Model, tea.Cmd) {
	switch command {
	case "":
		m.status = "no command"
		return m, nil
	case "new-task", "task-new":
		return m, m.startTaskForm(nil)
	case "new-subtask", "task-subtask":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		return m, m.startSubtaskForm(task)
	case "new-branch", "branch-new":
		if strings.TrimSpace(m.projectionRootTaskID) != "" {
			m.status = "clear focus before creating a branch"
			m.startWarningModal(
				"Branch Creation Blocked",
				fmt.Sprintf("New branches are project-level items. Press %s to return to full board, then create the branch.", m.keys.clearFocus.Help().Key),
			)
			return m, nil
		}
		parent, ok := m.selectedBranchTask()
		if ok {
			return m, m.startBranchForm(&parent)
		}
		return m, m.startBranchForm(nil)
	case "new-phase", "phase-new":
		parent, ok := m.focusedScopeTaskAtLevel("branch")
		if !ok {
			parent, ok = m.selectedTaskAtLevel("branch")
		}
		if !ok {
			m.status = "select a branch for new phase"
			return m, nil
		}
		return m, m.startPhaseForm(parent, false)
	case "new-subphase", "subphase-new":
		parent, ok := m.focusedScopeTaskAtLevels("phase", "subphase")
		if !ok {
			parent, ok = m.selectedTaskAtLevels("phase", "subphase")
		}
		if !ok {
			m.status = "select a phase/subphase for new subphase"
			return m, nil
		}
		return m, m.startPhaseForm(parent, true)
	case "edit-branch", "branch-edit":
		task, ok := m.selectedBranchTask()
		if !ok {
			m.status = "select a branch to edit"
			return m, nil
		}
		return m, m.startTaskForm(&task)
	case "archive-branch", "branch-archive":
		if _, ok := m.selectedBranchTask(); !ok {
			m.status = "select a branch to archive"
			return m, nil
		}
		return m.confirmDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive branch")
	case "delete-branch", "branch-delete":
		if _, ok := m.selectedBranchTask(); !ok {
			m.status = "select a branch to delete"
			return m, nil
		}
		return m.confirmDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "delete branch")
	case "restore-branch", "branch-restore":
		task, ok := m.selectedBranchTask()
		if !ok || task.ArchivedAt == nil {
			m.status = "select an archived branch to restore"
			return m, nil
		}
		return m.confirmRestoreAction()
	case "edit-task", "task-edit":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		return m, m.startTaskForm(&task)
	case "thread-item", "item-thread", "task-thread":
		return m.startSelectedWorkItemThread(modeNone)
	case "new-project", "project-new":
		return m, m.startProjectForm(nil)
	case "edit-project", "project-edit":
		if len(m.projects) == 0 {
			m.status = "no project selected"
			return m, nil
		}
		project := m.projects[clamp(m.selectedProject, 0, len(m.projects)-1)]
		return m, m.startProjectForm(&project)
	case "archive-project", "project-archive":
		return m.archiveCurrentProject(m.confirmArchive)
	case "restore-project", "project-restore":
		return m.restoreCurrentProject(m.confirmRestore)
	case "delete-project", "project-delete":
		return m.deleteCurrentProject(m.confirmHardDelete)
	case "thread-project", "project-thread":
		return m.startProjectThread(modeNone)
	case "search":
		return m, m.startSearchMode()
	case "search-all":
		m.searchCrossProject = true
		return m, m.startSearchMode()
	case "search-project":
		m.searchCrossProject = false
		return m, m.startSearchMode()
	case "clear-query", "clear-search-query":
		return m, m.clearSearchQuery()
	case "reset-filters", "clear-search":
		return m, m.resetSearchFilters()
	case "toggle-archived":
		m.showArchived = !m.showArchived
		m.selectedTask = 0
		m.clearSelection()
		if m.showArchived {
			m.status = "showing archived tasks"
		} else {
			m.status = "hiding archived tasks"
		}
		return m, m.loadData
	case "toggle-selection-mode", "select-mode", "text-select":
		m.mouseSelectionMode = !m.mouseSelectionMode
		if m.mouseSelectionMode {
			m.status = "text selection mode enabled"
		} else {
			m.status = "text selection mode disabled"
		}
		return m, nil
	case "focus-subtree", "zoom-task":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		if !m.activateSubtreeFocus(task.ID) {
			return m, nil
		}
		m.status = "focused subtree"
		return m, nil
	case "focus-clear", "zoom-reset":
		if !m.clearSubtreeFocus() {
			m.status = "full board already visible"
			return m, nil
		}
		m.status = "full board view"
		return m, nil
	case "toggle-select", "select-task":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		if m.toggleTaskSelection(task.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "select task",
				Target:  task.Title,
			})
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "unselect task",
				Target:  task.Title,
			})
		}
		return m, nil
	case "clear-selection", "selection-clear":
		count := m.clearSelection()
		if count == 0 {
			m.status = "selection already empty"
			return m, nil
		}
		m.status = fmt.Sprintf("cleared %d selected tasks", count)
		m.appendActivity(activityEntry{
			At:      time.Now().UTC(),
			Summary: "clear selection",
			Target:  fmt.Sprintf("%d tasks", count),
		})
		return m, nil
	case "bulk-move-left", "move-left-selected":
		return m.moveSelectedTasks(-1)
	case "bulk-move-right", "move-right-selected":
		return m.moveSelectedTasks(1)
	case "bulk-archive", "archive-selected":
		return m.confirmBulkDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive selected")
	case "bulk-delete", "delete-selected":
		return m.confirmBulkDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete selected")
	case "undo":
		return m.undoLastMutation()
	case "redo":
		return m.redoLastMutation()
	case "reload-config", "config-reload", "reload":
		m.status = "reloading config..."
		return m, m.reloadRuntimeConfigCmd()
	case "paths-roots", "roots", "project-root":
		return m, m.startPathsRootsMode()
	case "bootstrap-settings", "setup", "identity-roots":
		return m, m.startBootstrapSettingsMode(false)
	case "labels-config", "labels", "edit-labels":
		return m, m.startLabelsConfigForm()
	case "highlight-color", "set-highlight", "focus-color":
		return m, m.startHighlightColorMode()
	case "activity-log", "log":
		return m, m.openActivityLog()
	case "help":
		m.help.ShowAll = true
		m.status = "help"
		return m, nil
	case "quit", "exit":
		return m, tea.Quit
	default:
		m.status = "unknown command: " + command
		return m, nil
	}
}

// quickActions returns state-aware quick actions with enabled entries first.
func (m Model) quickActions() []quickActionItem {
	_, hasTask := m.selectedTaskInCurrentColumn()
	hasSelection := len(m.selectedTaskIDs) > 0
	enabled := make([]quickActionItem, 0, len(quickActionSpecs))
	disabled := make([]quickActionItem, 0, len(quickActionSpecs))
	for _, spec := range quickActionSpecs {
		available, reason := m.quickActionAvailability(spec.ID, hasTask, hasSelection)
		item := quickActionItem{
			ID:             spec.ID,
			Label:          spec.Label,
			Enabled:        available,
			DisabledReason: reason,
		}
		if item.Enabled {
			enabled = append(enabled, item)
			continue
		}
		disabled = append(disabled, item)
	}
	return append(enabled, disabled...)
}

// quickActionAvailability returns whether one quick action can run in the current state.
func (m Model) quickActionAvailability(actionID string, hasTask bool, hasSelection bool) (bool, string) {
	switch actionID {
	case "task-info", "edit-task", "archive-task", "hard-delete", "toggle-selection":
		if !hasTask {
			return false, "no task selected"
		}
		return true, ""
	case "restore-task":
		if task, ok := m.selectedTaskInCurrentColumn(); ok && task.ArchivedAt != nil {
			return true, ""
		}
		if strings.TrimSpace(m.lastArchivedTaskID) != "" {
			return true, ""
		}
		return false, "no archived task selected"
	case "move-left":
		if !hasTask {
			return false, "no task selected"
		}
		if m.selectedColumn <= 0 {
			return false, "already at first column"
		}
		return true, ""
	case "move-right":
		if !hasTask {
			return false, "no task selected"
		}
		if m.selectedColumn >= len(m.columns)-1 {
			return false, "already at last column"
		}
		return true, ""
	case "clear-selection":
		if !hasSelection {
			return false, "selection already empty"
		}
		return true, ""
	case "bulk-move-left":
		if !hasSelection {
			return false, "no tasks selected"
		}
		if len(m.buildMoveSteps(m.sortedSelectedTaskIDs(), -1)) == 0 {
			return false, "no movable tasks selected"
		}
		return true, ""
	case "bulk-move-right":
		if !hasSelection {
			return false, "no tasks selected"
		}
		if len(m.buildMoveSteps(m.sortedSelectedTaskIDs(), 1)) == 0 {
			return false, "no movable tasks selected"
		}
		return true, ""
	case "bulk-archive", "bulk-hard-delete":
		if !hasSelection {
			return false, "no tasks selected"
		}
		return true, ""
	case "undo":
		if len(m.undoStack) == 0 {
			return false, "nothing to undo"
		}
		return true, ""
	case "redo":
		if len(m.redoStack) == 0 {
			return false, "nothing to redo"
		}
		return true, ""
	case "activity-log":
		return true, ""
	default:
		return false, "unknown action"
	}
}

// applyQuickAction applies the currently focused quick action when available.
func (m Model) applyQuickAction() (tea.Model, tea.Cmd) {
	actions := m.quickActions()
	if len(actions) == 0 {
		m.status = "no quick actions"
		return m, nil
	}
	idx := clamp(m.quickActionIndex, 0, len(actions)-1)
	action := actions[idx]
	if !action.Enabled {
		reason := strings.TrimSpace(action.DisabledReason)
		if reason == "" {
			reason = "unavailable"
		}
		m.status = strings.ToLower(action.Label) + " unavailable: " + reason
		return m, nil
	}

	m.mode = modeNone
	switch action.ID {
	case "task-info":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		m.openTaskInfo(task.ID, "task info")
		return m, nil
	case "edit-task":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		return m, m.startTaskForm(&task)
	case "move-left":
		return m.moveSelectedTask(-1)
	case "move-right":
		return m.moveSelectedTask(1)
	case "archive-task":
		return m.confirmDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive task")
	case "restore-task":
		return m.confirmRestoreAction()
	case "hard-delete":
		return m.confirmDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete task")
	case "toggle-selection":
		task, ok := m.selectedTaskInCurrentColumn()
		if !ok {
			m.status = "no task selected"
			return m, nil
		}
		if m.toggleTaskSelection(task.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(task.Title, 28), len(m.selectedTaskIDs))
		}
		return m, nil
	case "clear-selection":
		count := m.clearSelection()
		if count == 0 {
			m.status = "selection already empty"
			return m, nil
		}
		m.status = fmt.Sprintf("cleared %d selected tasks", count)
		return m, nil
	case "bulk-move-left":
		return m.moveSelectedTasks(-1)
	case "bulk-move-right":
		return m.moveSelectedTasks(1)
	case "bulk-archive":
		return m.confirmBulkDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive selected")
	case "bulk-hard-delete":
		return m.confirmBulkDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete selected")
	case "undo":
		return m.undoLastMutation()
	case "redo":
		return m.redoLastMutation()
	case "activity-log":
		return m, m.openActivityLog()
	default:
		m.status = "unknown quick action"
		return m, nil
	}
}

// createTask creates task.
func (m Model) createTask(in app.CreateTaskInput) (tea.Model, tea.Cmd) {
	projectID, ok := m.currentProjectID()
	if !ok {
		m.status = "no active project"
		return m, nil
	}
	columnID, ok := m.currentColumnID()
	if !ok {
		m.status = "no active column"
		return m, nil
	}
	in.ProjectID = projectID
	in.ColumnID = columnID
	return m, func() tea.Msg {
		task, err := m.svc.CreateTask(context.Background(), in)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{status: "task created", reload: true, focusTaskID: task.ID}
	}
}

// moveSelectedTask moves the currently focused task one column left/right.
func (m Model) moveSelectedTask(delta int) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	return m.moveTaskIDs([]string{task.ID}, delta, "move task", task.Title, false)
}

// moveSelectedTasks moves every selected task one column left/right.
func (m Model) moveSelectedTasks(delta int) (tea.Model, tea.Cmd) {
	taskIDs := m.sortedSelectedTaskIDs()
	if len(taskIDs) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	label := "bulk move right"
	if delta < 0 {
		label = "bulk move left"
	}
	return m.moveTaskIDs(taskIDs, delta, label, fmt.Sprintf("%d tasks", len(taskIDs)), true)
}

// moveTaskIDs moves the provided task ids and records undo/redo history.
func (m Model) moveTaskIDs(taskIDs []string, delta int, label, target string, bulk bool) (tea.Model, tea.Cmd) {
	steps := m.buildMoveSteps(taskIDs, delta)
	if len(steps) == 0 {
		m.status = "no movable tasks selected"
		return m, nil
	}
	direction := "right"
	if delta < 0 {
		direction = "left"
	}
	status := "task moved"
	if bulk {
		status = fmt.Sprintf("moved %d tasks %s", len(steps), direction)
	}
	focusTaskID := steps[0].TaskID
	if bulk {
		focusTaskID = ""
	}
	history := historyActionSet{
		Label:    label,
		Summary:  status,
		Target:   target,
		Steps:    append([]historyStep(nil), steps...),
		Undoable: true,
		At:       time.Now().UTC(),
	}
	activity := activityEntry{
		At:      history.At,
		Summary: label,
		Target:  target,
	}
	return m, func() tea.Msg {
		for _, step := range steps {
			if _, err := m.svc.MoveTask(context.Background(), step.TaskID, step.ToColumnID, step.ToPosition); err != nil {
				return actionMsg{err: err}
			}
		}
		return actionMsg{
			status:       status,
			reload:       true,
			focusTaskID:  focusTaskID,
			historyPush:  &history,
			activityItem: &activity,
		}
	}
}

// deleteSelectedTask deletes or archives the currently focused task.
func (m Model) deleteSelectedTask(mode app.DeleteMode) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	return m.deleteTaskIDs([]string{task.ID}, mode)
}

// deleteTaskIDs archives/deletes task ids and records undo metadata when possible.
func (m Model) deleteTaskIDs(taskIDs []string, mode app.DeleteMode) (tea.Model, tea.Cmd) {
	ids := m.normalizeKnownTaskIDs(taskIDs)
	if len(ids) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	undoable := mode != app.DeleteModeHard
	label := "archive task"
	if mode == app.DeleteModeHard {
		label = "hard delete task"
	}
	if len(ids) > 1 {
		if mode == app.DeleteModeHard {
			label = "bulk hard delete"
		} else {
			label = "bulk archive"
		}
	}
	status := "task archived"
	if mode == app.DeleteModeHard {
		status = "task deleted"
	}
	if len(ids) > 1 {
		if mode == app.DeleteModeHard {
			status = fmt.Sprintf("deleted %d tasks", len(ids))
		} else {
			status = fmt.Sprintf("archived %d tasks", len(ids))
		}
	}

	steps := make([]historyStep, 0, len(ids))
	for _, taskID := range ids {
		step := historyStep{TaskID: taskID}
		if mode == app.DeleteModeHard {
			step.Kind = historyStepHardDelete
		} else {
			step.Kind = historyStepArchive
		}
		steps = append(steps, step)
	}
	history := historyActionSet{
		Label:    label,
		Summary:  status,
		Target:   fmt.Sprintf("%d tasks", len(ids)),
		Steps:    steps,
		Undoable: undoable,
		At:       time.Now().UTC(),
	}
	activity := activityEntry{
		At:      history.At,
		Summary: label,
		Target:  fmt.Sprintf("%d tasks", len(ids)),
	}
	if mode == app.DeleteModeArchive {
		m.lastArchivedTaskID = ids[len(ids)-1]
	}
	return m, func() tea.Msg {
		for _, taskID := range ids {
			if err := m.svc.DeleteTask(context.Background(), taskID, mode); err != nil {
				return actionMsg{err: err}
			}
		}
		return actionMsg{
			status:       status,
			reload:       true,
			clearTaskIDs: ids,
			historyPush:  &history,
			activityItem: &activity,
		}
	}
}

// confirmDeleteAction opens a confirmation modal when configured, or executes directly.
func (m Model) confirmDeleteAction(mode app.DeleteMode, needsConfirm bool, label string) (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		m.status = "no task selected"
		return m, nil
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = "delete task"
	}
	if !needsConfirm {
		return m.deleteTaskIDs([]string{task.ID}, mode)
	}
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:    "delete",
		Task:    task,
		TaskIDs: []string{task.ID},
		Mode:    mode,
		Label:   label,
	}
	m.confirmChoice = 1
	m.status = "confirm action"
	return m, nil
}

// confirmBulkDeleteAction confirms and applies bulk archive/hard-delete operations.
func (m Model) confirmBulkDeleteAction(mode app.DeleteMode, needsConfirm bool, label string) (tea.Model, tea.Cmd) {
	taskIDs := m.sortedSelectedTaskIDs()
	if len(taskIDs) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	if !needsConfirm {
		return m.deleteTaskIDs(taskIDs, mode)
	}
	task, _ := m.taskByID(taskIDs[0])
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:    "delete",
		Task:    task,
		TaskIDs: taskIDs,
		Mode:    mode,
		Label:   label,
	}
	m.confirmChoice = 1
	m.status = "confirm action"
	return m, nil
}

// restoreTask restores the most-recent archived task or selected archived task.
func (m Model) restoreTask() (tea.Model, tea.Cmd) {
	taskID := m.lastArchivedTaskID
	if taskID == "" {
		task, ok := m.selectedTaskInCurrentColumn()
		if ok && task.ArchivedAt != nil {
			taskID = task.ID
		}
	}
	if taskID == "" {
		m.status = "nothing to restore"
		return m, nil
	}
	return m.restoreTaskIDs([]string{taskID}, "task restored", "restore task")
}

// restoreTaskIDs restores tasks and records undo history.
func (m Model) restoreTaskIDs(taskIDs []string, status, label string) (tea.Model, tea.Cmd) {
	ids := make([]string, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		ids = append(ids, taskID)
	}
	if len(ids) == 0 {
		m.status = "nothing to restore"
		return m, nil
	}
	steps := make([]historyStep, 0, len(ids))
	for _, taskID := range ids {
		steps = append(steps, historyStep{
			Kind:   historyStepRestore,
			TaskID: taskID,
		})
	}
	history := historyActionSet{
		Label:    label,
		Summary:  status,
		Target:   fmt.Sprintf("%d tasks", len(ids)),
		Steps:    steps,
		Undoable: true,
		At:       time.Now().UTC(),
	}
	activity := activityEntry{
		At:      history.At,
		Summary: label,
		Target:  fmt.Sprintf("%d tasks", len(ids)),
	}
	return m, func() tea.Msg {
		for _, taskID := range ids {
			if _, err := m.svc.RestoreTask(context.Background(), taskID); err != nil {
				return actionMsg{err: err}
			}
		}
		return actionMsg{
			status:       status,
			reload:       true,
			historyPush:  &history,
			activityItem: &activity,
		}
	}
}

// confirmRestoreAction opens restore confirmation when configured, or executes directly.
func (m Model) confirmRestoreAction() (tea.Model, tea.Cmd) {
	task, ok := m.selectedTaskInCurrentColumn()
	if ok && task.ArchivedAt == nil {
		ok = false
	}
	if !m.confirmRestore || !ok {
		return m.restoreTask()
	}
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:    "restore",
		Task:    task,
		TaskIDs: []string{task.ID},
		Mode:    app.DeleteModeArchive,
		Label:   "restore task",
	}
	m.confirmChoice = 1
	m.status = "confirm action"
	return m, nil
}

// archiveCurrentProject archives the active project with optional confirmation.
func (m Model) archiveCurrentProject(needsConfirm bool) (tea.Model, tea.Cmd) {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return m, nil
	}
	if project.ArchivedAt != nil {
		m.status = "project already archived"
		return m, nil
	}
	if needsConfirm {
		m.mode = modeConfirmAction
		m.pendingConfirm = confirmAction{
			Kind:    "archive-project",
			Project: project,
			Label:   "archive project",
		}
		m.confirmChoice = 1
		m.status = "confirm action"
		return m, nil
	}
	projectID := project.ID
	nextProjectID := ""
	if !m.showArchivedProjects {
		for _, candidate := range m.projects {
			if candidate.ID == projectID || candidate.ArchivedAt != nil {
				continue
			}
			nextProjectID = candidate.ID
			break
		}
	}
	return m, func() tea.Msg {
		if _, err := m.svc.ArchiveProject(context.Background(), projectID); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{status: "project archived", reload: true, projectID: nextProjectID}
	}
}

// restoreCurrentProject restores the active archived project with optional confirmation.
func (m Model) restoreCurrentProject(needsConfirm bool) (tea.Model, tea.Cmd) {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return m, nil
	}
	if project.ArchivedAt == nil {
		m.status = "project is not archived"
		return m, nil
	}
	if needsConfirm {
		m.mode = modeConfirmAction
		m.pendingConfirm = confirmAction{
			Kind:    "restore-project",
			Project: project,
			Label:   "restore project",
		}
		m.confirmChoice = 1
		m.status = "confirm action"
		return m, nil
	}
	projectID := project.ID
	return m, func() tea.Msg {
		updated, err := m.svc.RestoreProject(context.Background(), projectID)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{status: "project restored", reload: true, projectID: updated.ID}
	}
}

// deleteCurrentProject hard-deletes the active project with optional confirmation.
func (m Model) deleteCurrentProject(needsConfirm bool) (tea.Model, tea.Cmd) {
	project, ok := m.currentProject()
	if !ok {
		m.status = "no project selected"
		return m, nil
	}
	if needsConfirm {
		m.mode = modeConfirmAction
		m.pendingConfirm = confirmAction{
			Kind:    "delete-project",
			Project: project,
			Label:   "delete project",
		}
		m.confirmChoice = 1
		m.status = "confirm action"
		return m, nil
	}
	projectID := project.ID
	nextProjectID := ""
	for _, candidate := range m.projects {
		if candidate.ID == projectID {
			continue
		}
		if !m.showArchivedProjects && candidate.ArchivedAt != nil {
			continue
		}
		nextProjectID = candidate.ID
		break
	}
	return m, func() tea.Msg {
		if err := m.svc.DeleteProject(context.Background(), projectID); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{
			status:    "project deleted",
			reload:    true,
			projectID: nextProjectID,
		}
	}
}

// applyConfirmedAction executes a previously confirmed action.
func (m Model) applyConfirmedAction(action confirmAction) (tea.Model, tea.Cmd) {
	switch action.Kind {
	case "delete":
		taskIDs := action.TaskIDs
		if len(taskIDs) == 0 && strings.TrimSpace(action.Task.ID) != "" {
			taskIDs = []string{action.Task.ID}
		}
		return m.deleteTaskIDs(taskIDs, action.Mode)
	case "restore":
		taskIDs := action.TaskIDs
		if len(taskIDs) == 0 && strings.TrimSpace(action.Task.ID) != "" {
			taskIDs = []string{action.Task.ID}
		}
		return m.restoreTaskIDs(taskIDs, "task restored", "restore task")
	case "archive-project":
		if projectID := strings.TrimSpace(action.Project.ID); projectID != "" {
			for idx, project := range m.projects {
				if project.ID != projectID {
					continue
				}
				m.selectedProject = idx
				break
			}
		}
		return m.archiveCurrentProject(false)
	case "restore-project":
		if projectID := strings.TrimSpace(action.Project.ID); projectID != "" {
			for idx, project := range m.projects {
				if project.ID != projectID {
					continue
				}
				m.selectedProject = idx
				break
			}
		}
		return m.restoreCurrentProject(false)
	case "delete-project":
		if projectID := strings.TrimSpace(action.Project.ID); projectID != "" {
			for idx, project := range m.projects {
				if project.ID != projectID {
					continue
				}
				m.selectedProject = idx
				break
			}
		}
		return m.deleteCurrentProject(false)
	default:
		m.status = "unknown confirm action"
		return m, nil
	}
}

// handleMouseWheel handles mouse wheel.
func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if m.mouseSelectionMode {
		return m, nil
	}
	if m.help.ShowAll {
		return m, nil
	}
	if m.mode == modeThread {
		switch msg.Button {
		case tea.MouseWheelUp:
			m.threadScroll = max(0, m.threadScroll-3)
		case tea.MouseWheelDown:
			m.threadScroll += 3
		}
		return m, nil
	}
	if m.mode == modeBootstrapSettings {
		if m.bootstrapFocus != 1 || len(m.bootstrapRoots) == 0 {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.bootstrapRootIndex > 0 {
				m.bootstrapRootIndex--
			}
		case tea.MouseWheelDown:
			if m.bootstrapRootIndex < len(m.bootstrapRoots)-1 {
				m.bootstrapRootIndex++
			}
		}
		return m, nil
	}
	if m.mode == modeProjectPicker {
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.projectPickerIndex > 0 {
				m.projectPickerIndex--
			}
		case tea.MouseWheelDown:
			if m.projectPickerIndex < len(m.projects)-1 {
				m.projectPickerIndex++
			}
		}
		return m, nil
	}
	if m.mode != modeNone {
		return m, nil
	}

	tasks := m.currentColumnTasks()
	if len(tasks) == 0 {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		if m.selectedTask > 0 {
			m.selectedTask--
		}
	case tea.MouseWheelDown:
		if m.selectedTask < len(tasks)-1 {
			m.selectedTask++
		}
	}
	return m, nil
}

// handleMouseClick handles mouse click.
func (m Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.mouseSelectionMode {
		return m, nil
	}
	if m.help.ShowAll {
		return m, nil
	}
	if m.mode == modeProjectPicker {
		// Project picker overlay is centered independently from path/header variations.
		overlayTop := 3
		if len(m.projects) > 1 {
			overlayTop++
		}
		relative := msg.Y - overlayTop - 1 // inside border, first row is title
		if relative >= 1 {
			idx := relative - 1
			if idx >= 0 && idx < len(m.projects) {
				m.projectPickerIndex = idx
			}
		}
		return m, nil
	}
	if m.mode != modeNone {
		return m, nil
	}

	if len(m.columns) == 0 {
		return m, nil
	}
	colWidth := m.columnWidth() + 5 // border + padding approximation for mouse hit testing
	gap := 0
	for idx := range m.columns {
		start := idx * (colWidth + gap)
		end := start + colWidth
		if msg.X >= start && msg.X < end {
			m.selectedColumn = idx
			break
		}
	}

	relativeY := msg.Y - m.boardTop()
	if relativeY >= 2 {
		tasks := m.currentColumnTasks()
		if len(tasks) > 0 {
			row := relativeY - 2
			m.selectedTask = clamp(m.taskIndexAtRow(tasks, row), 0, len(tasks)-1)
		}
	}
	m.clampSelections()
	return m, nil
}

// clampSelections clamps selections.
func (m *Model) clampSelections() {
	if len(m.projects) == 0 {
		m.selectedProject = 0
		m.selectedColumn = 0
		m.selectedTask = 0
		return
	}
	m.selectedProject = clamp(m.selectedProject, 0, len(m.projects)-1)

	if len(m.columns) == 0 {
		m.selectedColumn = 0
		m.selectedTask = 0
		return
	}
	m.selectedColumn = clamp(m.selectedColumn, 0, len(m.columns)-1)
	colTasks := m.currentColumnTasks()
	if len(colTasks) == 0 {
		m.selectedTask = 0
		return
	}
	m.selectedTask = clamp(m.selectedTask, 0, len(colTasks)-1)
}

// retainSelectionForLoadedTasks drops selected task ids that are no longer loaded.
func (m *Model) retainSelectionForLoadedTasks() {
	if len(m.selectedTaskIDs) == 0 {
		return
	}
	known := map[string]struct{}{}
	for _, task := range m.tasks {
		known[task.ID] = struct{}{}
	}
	for taskID := range m.selectedTaskIDs {
		if _, ok := known[taskID]; !ok {
			delete(m.selectedTaskIDs, taskID)
		}
	}
}

// isTaskSelected reports whether a task id is currently in the multi-select set.
func (m Model) isTaskSelected(taskID string) bool {
	_, ok := m.selectedTaskIDs[strings.TrimSpace(taskID)]
	return ok
}

// toggleTaskSelection adds/removes a task id from the current selection.
func (m *Model) toggleTaskSelection(taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false
	}
	if m.selectedTaskIDs == nil {
		m.selectedTaskIDs = map[string]struct{}{}
	}
	if _, ok := m.selectedTaskIDs[taskID]; ok {
		delete(m.selectedTaskIDs, taskID)
		return false
	}
	m.selectedTaskIDs[taskID] = struct{}{}
	return true
}

// clearSelection clears all selected task ids and returns the previous count.
func (m *Model) clearSelection() int {
	count := len(m.selectedTaskIDs)
	if count == 0 {
		return 0
	}
	m.selectedTaskIDs = map[string]struct{}{}
	return count
}

// unselectTasks removes provided task ids from multi-select state.
func (m *Model) unselectTasks(taskIDs []string) int {
	if len(m.selectedTaskIDs) == 0 {
		return 0
	}
	removed := 0
	for _, taskID := range taskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		if _, ok := m.selectedTaskIDs[taskID]; !ok {
			continue
		}
		delete(m.selectedTaskIDs, taskID)
		removed++
	}
	return removed
}

// sortedSelectedTaskIDs returns selected ids in board display order.
func (m Model) sortedSelectedTaskIDs() []string {
	if len(m.selectedTaskIDs) == 0 {
		return nil
	}
	taskIDs := make([]string, 0, len(m.selectedTaskIDs))
	for taskID := range m.selectedTaskIDs {
		taskIDs = append(taskIDs, taskID)
	}
	return m.normalizeKnownTaskIDs(taskIDs)
}

// normalizeKnownTaskIDs returns deduplicated task ids in deterministic board order.
func (m Model) normalizeKnownTaskIDs(taskIDs []string) []string {
	if len(taskIDs) == 0 {
		return nil
	}
	needed := map[string]struct{}{}
	for _, taskID := range taskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		needed[taskID] = struct{}{}
	}
	if len(needed) == 0 {
		return nil
	}
	out := make([]string, 0, len(needed))
	seen := map[string]struct{}{}
	for _, column := range m.columns {
		for _, task := range m.tasksForColumn(column.ID) {
			if _, ok := needed[task.ID]; !ok {
				continue
			}
			if _, ok := seen[task.ID]; ok {
				continue
			}
			seen[task.ID] = struct{}{}
			out = append(out, task.ID)
		}
	}
	for _, taskID := range taskIDs {
		if _, ok := seen[taskID]; ok {
			continue
		}
		if _, ok := m.taskByID(taskID); !ok {
			continue
		}
		seen[taskID] = struct{}{}
		out = append(out, taskID)
	}
	return out
}

// appendActivity appends one item to the in-app activity log with bounded retention.
func (m *Model) appendActivity(entry activityEntry) {
	if strings.TrimSpace(entry.Summary) == "" {
		return
	}
	if entry.At.IsZero() {
		entry.At = time.Now().UTC()
	}
	if strings.TrimSpace(entry.Target) == "" {
		entry.Target = "-"
	}
	entry.ActorType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(entry.ActorType))))
	if entry.ActorType == "" {
		entry.ActorType = domain.ActorTypeUser
	}
	if strings.TrimSpace(entry.ActorID) == "" {
		entry.ActorID = strings.TrimSpace(m.identityActorID)
	}
	if strings.TrimSpace(entry.ActorID) == "" {
		entry.ActorID = "tillsyn-user"
	}
	if strings.TrimSpace(entry.ActorName) == "" {
		entry.ActorName = strings.TrimSpace(m.identityDisplayName)
	}
	if strings.TrimSpace(entry.ActorName) == "" {
		entry.ActorName = strings.TrimSpace(entry.ActorID)
	}
	entry.Metadata = copyActivityMetadata(entry.Metadata)
	m.activityLog = append(m.activityLog, entry)
	if len(m.activityLog) > activityLogMaxItems {
		m.activityLog = append([]activityEntry(nil), m.activityLog[len(m.activityLog)-activityLogMaxItems:]...)
	}
}

// pushUndoHistory records one user mutation and clears redo history.
func (m *Model) pushUndoHistory(set historyActionSet) {
	if len(set.Steps) == 0 {
		return
	}
	m.nextHistoryID++
	set.ID = m.nextHistoryID
	if set.At.IsZero() {
		set.At = time.Now().UTC()
	}
	m.undoStack = append(m.undoStack, set)
	const maxItems = 100
	if len(m.undoStack) > maxItems {
		m.undoStack = append([]historyActionSet(nil), m.undoStack[len(m.undoStack)-maxItems:]...)
	}
	m.redoStack = nil
}

// applyUndoTransition shifts one action from undo stack to redo stack after success.
func (m *Model) applyUndoTransition(set historyActionSet) {
	if len(m.undoStack) > 0 {
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
	}
	m.redoStack = append(m.redoStack, set)
}

// applyRedoTransition shifts one action from redo stack back to undo stack after success.
func (m *Model) applyRedoTransition(set historyActionSet) {
	if len(m.redoStack) > 0 {
		m.redoStack = m.redoStack[:len(m.redoStack)-1]
	}
	m.undoStack = append(m.undoStack, set)
}

// undoLastMutation reverses the most recent undoable mutation set.
func (m Model) undoLastMutation() (tea.Model, tea.Cmd) {
	if len(m.undoStack) == 0 {
		m.status = "nothing to undo"
		return m, nil
	}
	set := m.undoStack[len(m.undoStack)-1]
	if !set.Undoable {
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
		m.status = "last action cannot be undone"
		m.appendActivity(activityEntry{
			At:      time.Now().UTC(),
			Summary: "undo unavailable",
			Target:  set.Label,
		})
		return m, nil
	}
	return m, m.executeHistorySet(set, true)
}

// redoLastMutation reapplies the most recently undone mutation set.
func (m Model) redoLastMutation() (tea.Model, tea.Cmd) {
	if len(m.redoStack) == 0 {
		m.status = "nothing to redo"
		return m, nil
	}
	set := m.redoStack[len(m.redoStack)-1]
	return m, m.executeHistorySet(set, false)
}

// executeHistorySet applies one history action set in either undo or redo direction.
func (m Model) executeHistorySet(set historyActionSet, undo bool) tea.Cmd {
	steps := append([]historyStep(nil), set.Steps...)
	if undo {
		slices.Reverse(steps)
	}
	return func() tea.Msg {
		clearIDs := make([]string, 0, len(steps))
		for _, step := range steps {
			switch step.Kind {
			case historyStepMove:
				columnID := step.ToColumnID
				position := step.ToPosition
				if undo {
					columnID = step.FromColumnID
					position = step.FromPosition
				}
				if _, err := m.svc.MoveTask(context.Background(), step.TaskID, columnID, position); err != nil {
					return actionMsg{err: err}
				}
			case historyStepArchive:
				if undo {
					if _, err := m.svc.RestoreTask(context.Background(), step.TaskID); err != nil {
						return actionMsg{err: err}
					}
				} else {
					if err := m.svc.DeleteTask(context.Background(), step.TaskID, app.DeleteModeArchive); err != nil {
						return actionMsg{err: err}
					}
					clearIDs = append(clearIDs, step.TaskID)
				}
			case historyStepRestore:
				if undo {
					if err := m.svc.DeleteTask(context.Background(), step.TaskID, app.DeleteModeArchive); err != nil {
						return actionMsg{err: err}
					}
					clearIDs = append(clearIDs, step.TaskID)
				} else {
					if _, err := m.svc.RestoreTask(context.Background(), step.TaskID); err != nil {
						return actionMsg{err: err}
					}
				}
			case historyStepHardDelete:
				if undo {
					return actionMsg{status: "undo failed: hard delete cannot be restored"}
				}
				if err := m.svc.DeleteTask(context.Background(), step.TaskID, app.DeleteModeHard); err != nil {
					return actionMsg{err: err}
				}
				clearIDs = append(clearIDs, step.TaskID)
			}
		}
		status := "redo complete"
		activitySummary := "redo"
		msg := actionMsg{
			reload:       true,
			clearTaskIDs: clearIDs,
			historyRedo:  &set,
		}
		if undo {
			status = "undo complete"
			activitySummary = "undo"
			msg.historyRedo = nil
			msg.historyUndo = &set
		}
		msg.status = fmt.Sprintf("%s: %s", status, set.Label)
		msg.activityItem = &activityEntry{
			At:      time.Now().UTC(),
			Summary: activitySummary,
			Target:  set.Label,
		}
		return msg
	}
}

// buildMoveSteps computes move history steps for task ids with deterministic ordering.
func (m Model) buildMoveSteps(taskIDs []string, delta int) []historyStep {
	if delta == 0 {
		return nil
	}
	ids := m.normalizeKnownTaskIDs(taskIDs)
	if len(ids) == 0 {
		return nil
	}
	colIndexByID := map[string]int{}
	for idx, column := range m.columns {
		colIndexByID[column.ID] = idx
	}
	steps := make([]historyStep, 0, len(ids))
	for _, taskID := range ids {
		task, ok := m.taskByID(taskID)
		if !ok {
			continue
		}
		fromColIdx, ok := colIndexByID[task.ColumnID]
		if !ok {
			continue
		}
		toColIdx := fromColIdx + delta
		if toColIdx < 0 || toColIdx >= len(m.columns) {
			continue
		}
		steps = append(steps, historyStep{
			Kind:         historyStepMove,
			TaskID:       task.ID,
			FromColumnID: task.ColumnID,
			FromPosition: task.Position,
			ToColumnID:   m.columns[toColIdx].ID,
		})
	}
	if len(steps) == 0 {
		return nil
	}
	sort.SliceStable(steps, func(i, j int) bool {
		iTask, _ := m.taskByID(steps[i].TaskID)
		jTask, _ := m.taskByID(steps[j].TaskID)
		if iTask.ColumnID == jTask.ColumnID {
			if iTask.Position == jTask.Position {
				return iTask.ID < jTask.ID
			}
			return iTask.Position < jTask.Position
		}
		return colIndexByID[iTask.ColumnID] < colIndexByID[jTask.ColumnID]
	})

	targetPosByColumn := map[string]int{}
	for _, step := range steps {
		if _, ok := targetPosByColumn[step.ToColumnID]; ok {
			continue
		}
		targetPosByColumn[step.ToColumnID] = len(m.tasksForColumn(step.ToColumnID))
	}
	for idx := range steps {
		steps[idx].ToPosition = targetPosByColumn[steps[idx].ToColumnID]
		targetPosByColumn[steps[idx].ToColumnID]++
	}
	return steps
}

// groupLabelForTask returns the swimlane/group label for a task under current settings.
func (m Model) groupLabelForTask(task domain.Task) string {
	switch normalizeBoardGroupBy(m.boardGroupBy) {
	case "priority":
		switch task.Priority {
		case domain.PriorityHigh:
			return "Priority: High"
		case domain.PriorityMedium:
			return "Priority: Medium"
		case domain.PriorityLow:
			return "Priority: Low"
		default:
			return "Priority: Unknown"
		}
	case "state":
		switch strings.ToLower(strings.TrimSpace(string(task.LifecycleState))) {
		case "todo":
			return "State: To Do"
		case "progress":
			return "State: In Progress"
		case "done":
			return "State: Done"
		case "archived":
			return "State: Archived"
		default:
			return "State: Unknown"
		}
	default:
		return "Tasks"
	}
}

// currentProjectID returns current project id.
func (m Model) currentProjectID() (string, bool) {
	if len(m.projects) == 0 {
		return "", false
	}
	idx := clamp(m.selectedProject, 0, len(m.projects)-1)
	return m.projects[idx].ID, true
}

// currentColumnID returns current column id.
func (m Model) currentColumnID() (string, bool) {
	if len(m.columns) == 0 {
		return "", false
	}
	idx := clamp(m.selectedColumn, 0, len(m.columns)-1)
	return m.columns[idx].ID, true
}

// currentProject returns the currently selected project.
func (m Model) currentProject() (domain.Project, bool) {
	if len(m.projects) == 0 {
		return domain.Project{}, false
	}
	idx := clamp(m.selectedProject, 0, len(m.projects)-1)
	return m.projects[idx], true
}

// currentColumnTasks returns current column tasks.
func (m Model) currentColumnTasks() []domain.Task {
	columnID, ok := m.currentColumnID()
	if !ok {
		return nil
	}
	return m.boardTasksForColumn(columnID)
}

// boardTasksForColumn returns only board-visible tasks for a column.
func (m Model) boardTasksForColumn(columnID string) []domain.Task {
	columnTasks := m.tasksForColumn(columnID)
	if len(columnTasks) == 0 {
		return nil
	}
	includeSubtasks := m.focusedScopeShowsSubtasks()
	out := make([]domain.Task, 0, len(columnTasks))
	for _, task := range columnTasks {
		if task.Kind == domain.WorkKindSubtask && !includeSubtasks {
			continue
		}
		out = append(out, task)
	}
	return out
}

// focusedScopeShowsSubtasks reports whether the current focused scope should render subtask rows.
func (m Model) focusedScopeShowsSubtasks() bool {
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	// Once focused, show all direct children regardless of kind. This keeps
	// focus navigation resilient for older/irregular data where subtasks were
	// attached outside task/subtask roots.
	return rootID != ""
}

// tasksForColumn handles tasks for column.
func (m Model) tasksForColumn(columnID string) []domain.Task {
	out := make([]domain.Task, 0)
	projected := m.projectedTaskSet()
	for _, task := range m.tasks {
		if task.ColumnID != columnID {
			continue
		}
		if _, ok := projected[task.ID]; !ok {
			continue
		}
		out = append(out, task)
	}
	ordered := orderTasksByHierarchy(out)
	groupBy := normalizeBoardGroupBy(m.boardGroupBy)
	if groupBy != "none" {
		sort.SliceStable(ordered, func(i, j int) bool {
			iRank := taskGroupRank(ordered[i], groupBy)
			jRank := taskGroupRank(ordered[j], groupBy)
			if iRank == jRank {
				return false
			}
			return iRank < jRank
		})
	}
	return ordered
}

// baseSearchLevelForTask infers a canonical hierarchy level from one task's scope/kind.
func baseSearchLevelForTask(task domain.Task) string {
	switch domain.NormalizeKindAppliesTo(task.Scope) {
	case domain.KindAppliesToBranch:
		return "branch"
	case domain.KindAppliesToSubphase:
		return "subphase"
	case domain.KindAppliesToPhase:
		return "phase"
	case domain.KindAppliesToTask:
		return "task"
	case domain.KindAppliesToSubtask:
		return "subtask"
	}
	switch strings.TrimSpace(strings.ToLower(string(task.Kind))) {
	case "branch":
		return "branch"
	case "phase":
		return "phase"
	case "subphase":
		return "subphase"
	case "subtask":
		return "subtask"
	case "task":
		return "task"
	}
	if strings.TrimSpace(task.ParentID) != "" {
		return "subtask"
	}
	return "task"
}

// searchLevelByTaskID resolves one canonical hierarchy level per task ID.
func (m Model) searchLevelByTaskID(tasks []domain.Task) map[string]string {
	byID := map[string]domain.Task{}
	for _, task := range m.tasks {
		byID[task.ID] = task
	}
	for _, task := range tasks {
		byID[task.ID] = task
	}
	if len(byID) == 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	var resolve func(string, map[string]struct{}) string
	resolve = func(taskID string, visiting map[string]struct{}) string {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			return "task"
		}
		if level, ok := out[taskID]; ok {
			return level
		}
		task, ok := byID[taskID]
		if !ok {
			return "task"
		}
		if _, seen := visiting[taskID]; seen {
			return "task"
		}
		visiting[taskID] = struct{}{}
		level := baseSearchLevelForTask(task)
		// Nested phase nodes represent subphases in board/search level filters.
		if level == "phase" {
			parentLevel := resolve(task.ParentID, visiting)
			if parentLevel == "phase" || parentLevel == "subphase" {
				level = "subphase"
			}
		}
		delete(visiting, taskID)
		out[taskID] = level
		return level
	}
	for taskID := range byID {
		resolve(taskID, map[string]struct{}{})
	}
	return out
}

// taskMatchesSearchLevels reports whether one task passes active search level filters.
func (m Model) taskMatchesSearchLevels(task domain.Task, levelByTaskID map[string]string) bool {
	enabled := canonicalSearchLevels(m.searchLevels)
	enabledSet := make(map[string]struct{}, len(enabled))
	for _, level := range enabled {
		enabledSet[level] = struct{}{}
	}
	if _, ok := enabledSet["project"]; ok {
		return true
	}
	level := strings.TrimSpace(strings.ToLower(levelByTaskID[task.ID]))
	if level == "" {
		level = baseSearchLevelForTask(task)
	}
	_, ok := enabledSet[level]
	return ok
}

// filterTaskMatchesBySearchLevels keeps only search matches that satisfy level filters.
func (m Model) filterTaskMatchesBySearchLevels(matches []app.TaskMatch) []app.TaskMatch {
	if len(matches) == 0 {
		return nil
	}
	tasks := make([]domain.Task, 0, len(matches))
	for _, match := range matches {
		tasks = append(tasks, match.Task)
	}
	levelByTaskID := m.searchLevelByTaskID(tasks)
	out := make([]app.TaskMatch, 0, len(matches))
	for _, match := range matches {
		if !m.taskMatchesSearchLevels(match.Task, levelByTaskID) {
			continue
		}
		out = append(out, match)
	}
	return out
}

// tasksByID builds a lookup map for loaded tasks keyed by task ID.
func (m Model) tasksByID() map[string]domain.Task {
	out := make(map[string]domain.Task, len(m.tasks))
	for _, task := range m.tasks {
		out[task.ID] = task
	}
	return out
}

// projectedTaskSet returns every task ID visible in the current board scope.
func (m Model) projectedTaskSet() map[string]struct{} {
	visible := map[string]struct{}{}
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	if rootID == "" {
		known := m.tasksByID()
		for _, task := range m.tasks {
			parentID := strings.TrimSpace(task.ParentID)
			if parentID == "" {
				visible[task.ID] = struct{}{}
				continue
			}
			// Preserve orphaned tasks in project scope so they remain recoverable in UI.
			if _, ok := known[parentID]; !ok {
				visible[task.ID] = struct{}{}
			}
		}
		return visible
	}
	if _, ok := m.taskByID(rootID); !ok {
		return visible
	}
	for _, task := range m.tasks {
		if strings.TrimSpace(task.ParentID) == rootID {
			visible[task.ID] = struct{}{}
		}
	}
	return visible
}

// projectionBreadcrumb returns the active subtree breadcrumb path.
func (m Model) projectionBreadcrumb() string {
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	if rootID == "" {
		return ""
	}
	root, ok := m.taskByID(rootID)
	if !ok {
		return ""
	}
	path := []string{root.Title}
	visited := map[string]struct{}{root.ID: {}}
	parentID := strings.TrimSpace(root.ParentID)
	for parentID != "" {
		if _, seen := visited[parentID]; seen {
			break
		}
		parent, found := m.taskByID(parentID)
		if !found {
			break
		}
		visited[parentID] = struct{}{}
		path = append(path, parent.Title)
		parentID = strings.TrimSpace(parent.ParentID)
	}
	slices.Reverse(path)
	return strings.Join(path, " / ")
}

// projectionPathWithProject returns focus path and direct parent labels for the active subtree root.
func (m Model) projectionPathWithProject(projectName string) (string, string) {
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	projectName = strings.TrimSpace(projectName)
	if rootID == "" {
		if projectName == "" {
			return "(project)", "(project)"
		}
		return projectName, projectName
	}
	root, ok := m.taskByID(rootID)
	if !ok {
		if projectName == "" {
			return "(project)", "(project)"
		}
		return projectName, projectName
	}
	chain := []string{root.Title}
	visited := map[string]struct{}{root.ID: {}}
	parentLabel := projectName
	parentID := strings.TrimSpace(root.ParentID)
	if parentID == "" && parentLabel == "" {
		parentLabel = "(project)"
	}
	for parentID != "" {
		if _, seen := visited[parentID]; seen {
			break
		}
		parent, found := m.taskByID(parentID)
		if !found {
			break
		}
		if parentLabel == "" || parentLabel == strings.TrimSpace(projectName) {
			parentLabel = parent.Title
		}
		visited[parentID] = struct{}{}
		chain = append(chain, parent.Title)
		parentID = strings.TrimSpace(parent.ParentID)
	}
	slices.Reverse(chain)
	if projectName != "" {
		chain = append([]string{projectName}, chain...)
		if parentLabel == "" {
			parentLabel = projectName
		}
	}
	return strings.Join(chain, " -> "), parentLabel
}

// searchLevelsSummary returns a compact list of active non-project search levels.
func (m Model) searchLevelsSummary() string {
	levels := canonicalSearchLevels(m.searchLevels)
	if len(levels) == len(canonicalSearchLevelsOrdered) {
		return ""
	}
	items := make([]string, 0, len(levels))
	for _, level := range levels {
		if level == "project" {
			continue
		}
		label := canonicalSearchLevelLabels[level]
		if label == "" {
			label = level
		}
		items = append(items, strings.ToLower(label))
	}
	if len(items) == 0 {
		return "project"
	}
	return strings.Join(items, ",")
}

// taskAttentionCount returns unresolved attention signals for one board-visible task.
func (m Model) taskAttentionCount(task domain.Task, byID map[string]domain.Task) int {
	count := 0
	for _, depID := range uniqueTrimmed(task.Metadata.DependsOn) {
		depTask, ok := byID[depID]
		if !ok || m.lifecycleStateForTask(depTask) != domain.StateDone {
			count++
		}
	}
	for _, blockerID := range uniqueTrimmed(task.Metadata.BlockedBy) {
		blockerTask, ok := byID[blockerID]
		if !ok || m.lifecycleStateForTask(blockerTask) != domain.StateDone {
			count++
		}
	}
	if strings.TrimSpace(task.Metadata.BlockedReason) != "" {
		count++
	}
	return count
}

// scopeAttentionSummary computes compact unresolved-attention totals for the current board scope.
func (m Model) scopeAttentionSummary(byID map[string]domain.Task) (int, int, int, []string) {
	items := 0
	total := 0
	blocked := 0
	top := make([]string, 0, 3)
	for _, column := range m.columns {
		for _, task := range m.boardTasksForColumn(column.ID) {
			count := m.taskAttentionCount(task, byID)
			if count <= 0 {
				continue
			}
			items++
			total += count
			if strings.TrimSpace(task.Metadata.BlockedReason) != "" {
				blocked++
			}
			if len(top) < 3 {
				top = append(top, fmt.Sprintf("%s !%d", truncate(task.Title, 24), count))
			}
		}
	}
	return items, total, blocked, top
}

// buildScopeWarnings synthesizes warning text from attention counts.
func buildScopeWarnings(attentionItemsCount, attentionUserActionCount int) []string {
	warnings := make([]string, 0, 2)
	if attentionItemsCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d work items report open blockers", attentionItemsCount))
	}
	if attentionUserActionCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d attention items require user action", attentionUserActionCount))
	}
	return warnings
}

// dependencyRollupSummary returns compact project dependency totals for board rendering.
func (m Model) dependencyRollupSummary() string {
	rollup := m.dependencyRollup
	return fmt.Sprintf(
		"deps: total %d • blocked %d • unresolved %d • edges %d",
		rollup.TotalItems,
		rollup.BlockedItems,
		rollup.UnresolvedDependencyEdges,
		rollup.DependencyEdges,
	)
}

// taskGroupRank returns deterministic ordering rank for configured board grouping.
func taskGroupRank(task domain.Task, groupBy string) int {
	switch normalizeBoardGroupBy(groupBy) {
	case "priority":
		switch task.Priority {
		case domain.PriorityHigh:
			return 0
		case domain.PriorityMedium:
			return 1
		case domain.PriorityLow:
			return 2
		default:
			return 3
		}
	case "state":
		switch strings.ToLower(strings.TrimSpace(string(task.LifecycleState))) {
		case "todo":
			return 0
		case "progress":
			return 1
		case "done":
			return 2
		case "archived":
			return 3
		default:
			return 4
		}
	default:
		return 0
	}
}

// orderTasksByHierarchy renders parent items before their descendants.
func orderTasksByHierarchy(tasks []domain.Task) []domain.Task {
	if len(tasks) <= 1 {
		return tasks
	}
	childrenByParent := map[string][]domain.Task{}
	byID := map[string]domain.Task{}
	roots := make([]domain.Task, 0)
	for _, task := range tasks {
		byID[task.ID] = task
	}
	for _, task := range tasks {
		parentID := strings.TrimSpace(task.ParentID)
		if parentID == "" {
			roots = append(roots, task)
			continue
		}
		if _, ok := byID[parentID]; !ok {
			roots = append(roots, task)
			continue
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], task)
	}
	sortTaskSlice(roots)
	for parentID := range childrenByParent {
		children := childrenByParent[parentID]
		sortTaskSlice(children)
		childrenByParent[parentID] = children
	}
	ordered := make([]domain.Task, 0, len(tasks))
	visited := map[string]struct{}{}
	var visit func(domain.Task)
	visit = func(task domain.Task) {
		if _, ok := visited[task.ID]; ok {
			return
		}
		visited[task.ID] = struct{}{}
		ordered = append(ordered, task)
		for _, child := range childrenByParent[task.ID] {
			visit(child)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	for _, task := range tasks {
		if _, ok := visited[task.ID]; ok {
			continue
		}
		visit(task)
	}
	return ordered
}

// sortTaskSlice orders tasks by creation time (oldest-first) with deterministic fallbacks.
func sortTaskSlice(tasks []domain.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		iCreated := tasks[i].CreatedAt
		jCreated := tasks[j].CreatedAt
		if !iCreated.IsZero() && !jCreated.IsZero() && !iCreated.Equal(jCreated) {
			return iCreated.Before(jCreated)
		}
		if tasks[i].Position != tasks[j].Position {
			return tasks[i].Position < tasks[j].Position
		}
		return tasks[i].ID < tasks[j].ID
	})
}

// taskDepth returns nesting depth for a task id with cycle protection.
func taskDepth(taskID string, parentByID map[string]string, depth int) int {
	if depth > 32 {
		return depth
	}
	parentID, ok := parentByID[taskID]
	if !ok || strings.TrimSpace(parentID) == "" {
		return depth
	}
	if _, exists := parentByID[parentID]; !exists {
		return depth
	}
	return taskDepth(parentID, parentByID, depth+1)
}

// selectedTaskInCurrentColumn returns selected task in current column.
func (m Model) selectedTaskInCurrentColumn() (domain.Task, bool) {
	tasks := m.currentColumnTasks()
	if len(tasks) == 0 {
		return domain.Task{}, false
	}
	idx := clamp(m.selectedTask, 0, len(tasks)-1)
	return tasks[idx], true
}

// selectedBranchTask returns the selected task when it is a branch-level work item.
func (m Model) selectedBranchTask() (domain.Task, bool) {
	return m.selectedTaskAtLevel("branch")
}

// selectedTaskAtLevel returns the selected task when it matches one hierarchy level.
func (m Model) selectedTaskAtLevel(level string) (domain.Task, bool) {
	return m.selectedTaskAtLevels(level)
}

// selectedTaskAtLevels returns the selected task when it matches one of the provided hierarchy levels.
func (m Model) selectedTaskAtLevels(levels ...string) (domain.Task, bool) {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		return domain.Task{}, false
	}
	if !taskMatchesHierarchyLevel(task, levels...) {
		return domain.Task{}, false
	}
	return task, true
}

// focusedScopeTaskAtLevel returns the active focus root task when it matches one hierarchy level.
func (m Model) focusedScopeTaskAtLevel(level string) (domain.Task, bool) {
	return m.focusedScopeTaskAtLevels(level)
}

// focusedScopeTaskAtLevels returns the active focus root task when it matches one provided hierarchy level.
func (m Model) focusedScopeTaskAtLevels(levels ...string) (domain.Task, bool) {
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	if rootID == "" {
		return domain.Task{}, false
	}
	task, ok := m.taskByID(rootID)
	if !ok {
		return domain.Task{}, false
	}
	if !taskMatchesHierarchyLevel(task, levels...) {
		return domain.Task{}, false
	}
	return task, true
}

// taskMatchesHierarchyLevel reports whether a task matches any provided hierarchy levels.
func taskMatchesHierarchyLevel(task domain.Task, levels ...string) bool {
	if len(levels) == 0 {
		return false
	}
	level := strings.TrimSpace(strings.ToLower(baseSearchLevelForTask(task)))
	for _, candidate := range levels {
		if strings.TrimSpace(strings.ToLower(candidate)) == level {
			return true
		}
	}
	return false
}

// focusTaskByID focuses one task by id and reports whether it became selected.
func (m *Model) focusTaskByID(taskID string) bool {
	if strings.TrimSpace(taskID) == "" {
		return false
	}
	var targetColIdx = -1
	for idx, column := range m.columns {
		tasks := m.tasksForColumn(column.ID)
		for taskIdx, task := range tasks {
			if task.ID == taskID {
				targetColIdx = idx
				m.selectedColumn = idx
				m.selectedTask = taskIdx
				break
			}
		}
		if targetColIdx >= 0 {
			break
		}
	}
	if targetColIdx >= 0 {
		m.clampSelections()
		return true
	}
	return false
}

// activateSubtreeFocus enters focused scope mode and selects the first visible child when present.
func (m *Model) activateSubtreeFocus(rootTaskID string) bool {
	rootTaskID = strings.TrimSpace(rootTaskID)
	if rootTaskID == "" {
		return false
	}
	if _, ok := m.taskByID(rootTaskID); !ok {
		return false
	}
	m.projectionRootTaskID = rootTaskID
	m.selectedTask = 0
	for idx, column := range m.columns {
		tasks := m.boardTasksForColumn(column.ID)
		if len(tasks) == 0 {
			continue
		}
		m.selectedColumn = idx
		m.selectedTask = 0
		m.clampSelections()
		return true
	}
	// Empty focused scopes are still valid so users can create the first child in place.
	m.clampSelections()
	return true
}

// clearSubtreeFocus exits focused scope mode and reselects the prior focus root when available.
func (m *Model) clearSubtreeFocus() bool {
	rootID := strings.TrimSpace(m.projectionRootTaskID)
	if rootID == "" {
		return false
	}
	m.projectionRootTaskID = ""
	m.focusTaskByID(rootID)
	m.clampSelections()
	return true
}

// taskByID returns task by id.
func (m Model) taskByID(taskID string) (domain.Task, bool) {
	for _, task := range m.tasks {
		if task.ID == taskID {
			return task, true
		}
	}
	return domain.Task{}, false
}

// directChildCount returns the number of direct children for one task id.
func (m Model) directChildCount(taskID string) int {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return 0
	}
	count := 0
	for _, task := range m.tasks {
		if strings.TrimSpace(task.ParentID) != taskID {
			continue
		}
		if !m.showArchived && task.ArchivedAt != nil {
			continue
		}
		count++
	}
	return count
}

// renderProjectTabs renders output for the current model state.
func (m Model) renderProjectTabs(accent, dim color.Color) string {
	if len(m.projects) <= 1 {
		return ""
	}
	active := lipgloss.NewStyle().Bold(true).Foreground(accent)
	inactive := lipgloss.NewStyle().Foreground(dim)

	parts := make([]string, 0, len(m.projects))
	for idx, p := range m.projects {
		label := projectDisplayLabel(p)
		if idx == m.selectedProject {
			parts = append(parts, active.Render("["+label+"]"))
		} else {
			parts = append(parts, inactive.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}

// projectDisplayName returns one user-facing project name with an optional icon prefix.
func projectDisplayName(project domain.Project) string {
	name := strings.TrimSpace(project.Name)
	if name == "" {
		name = strings.TrimSpace(project.ID)
	}
	if icon := strings.TrimSpace(project.Metadata.Icon); icon != "" {
		return icon + " " + name
	}
	return name
}

// projectDisplayLabel returns one user-facing project label with archive marker text.
func projectDisplayLabel(project domain.Project) string {
	label := projectDisplayName(project)
	if project.ArchivedAt != nil {
		label += " (archived)"
	}
	return label
}

// projectAccentColor returns the project-specific accent color or the default accent.
func projectAccentColor(project domain.Project) color.Color {
	value := strings.TrimSpace(project.Metadata.Color)
	if value == "" {
		return lipgloss.Color("62")
	}
	return lipgloss.Color(value)
}

// selectedTaskHighlightColor returns the configured board-selection highlight color.
func (m Model) selectedTaskHighlightColor() color.Color {
	value := strings.TrimSpace(m.highlightColor)
	if value == "" {
		value = defaultHighlightColor
	}
	return lipgloss.Color(value)
}

// canFocusNoticesPanel reports whether the notices panel can accept keyboard focus.
func (m Model) canFocusNoticesPanel() bool {
	return m.isNoticesPanelVisible()
}

// recentActivityPanelEntries returns newest-first activity entries for notices rendering/navigation.
func (m Model) recentActivityPanelEntries() []activityEntry {
	if len(m.activityLog) == 0 {
		return nil
	}
	out := make([]activityEntry, 0, len(m.activityLog))
	for idx := len(m.activityLog) - 1; idx >= 0; idx-- {
		out = append(out, m.activityLog[idx])
	}
	return out
}

// noticesSectionTitle returns a stable header label for one notices section identifier.
func noticesSectionTitle(section noticesSectionID) string {
	switch section {
	case noticesSectionWarnings:
		return "Warnings"
	case noticesSectionAttention:
		return "Agent/User Action"
	case noticesSectionSelection:
		return "Selection"
	case noticesSectionRecentActivity:
		return "Recent Activity"
	default:
		return "Notices"
	}
}

// noticesSelectionIndex returns the current selected row index for one notices section.
func (m Model) noticesSelectionIndex(section noticesSectionID) int {
	switch section {
	case noticesSectionWarnings:
		return m.noticesWarnings
	case noticesSectionAttention:
		return m.noticesAttention
	case noticesSectionSelection:
		return m.noticesSelection
	case noticesSectionRecentActivity:
		return m.noticesActivity
	default:
		return 0
	}
}

// setNoticesSelectionIndex stores one selected row index for the target notices section.
func (m *Model) setNoticesSelectionIndex(section noticesSectionID, idx int) {
	switch section {
	case noticesSectionWarnings:
		m.noticesWarnings = idx
	case noticesSectionAttention:
		m.noticesAttention = idx
	case noticesSectionSelection:
		m.noticesSelection = idx
	case noticesSectionRecentActivity:
		m.noticesActivity = idx
	}
}

// noticesSectionPosition resolves one section id to its traversal position.
func noticesSectionPosition(section noticesSectionID) int {
	for idx, candidate := range noticesPanelSectionOrder {
		if candidate == section {
			return idx
		}
	}
	return -1
}

// noticesAttentionPanelItems builds selectable attention rows, preserving board display order.
func (m Model) noticesAttentionPanelItems(byID map[string]domain.Task, fallback []string) []noticesPanelItem {
	out := make([]noticesPanelItem, 0)
	for _, column := range m.columns {
		for _, task := range m.boardTasksForColumn(column.ID) {
			count := m.taskAttentionCount(task, byID)
			if count <= 0 {
				continue
			}
			out = append(out, noticesPanelItem{
				Label:  fmt.Sprintf("%s !%d", task.Title, count),
				TaskID: task.ID,
			})
		}
	}
	if len(out) == 0 {
		for _, item := range fallback {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			out = append(out, noticesPanelItem{Label: item})
		}
	}
	return out
}

// noticesActivityItemLabel returns the untruncated display label for one activity row.
func (m Model) noticesActivityItemLabel(entry activityEntry) string {
	actorType, owner := m.displayActivityOwner(entry)
	return fmt.Sprintf("%s|%s %s", actorType, owner, entry.Summary)
}

// noticesPanelSections builds the focusable notices-panel sections and selectable item rows.
func (m Model) noticesPanelSections(
	attentionItems, attentionTotal, attentionBlocked int,
	attentionTop []string,
) []noticesPanelSection {
	sections := make([]noticesPanelSection, 0, len(noticesPanelSectionOrder))

	attentionRows := m.noticesAttentionPanelItems(m.tasksByID(), attentionTop)
	if len(attentionRows) == 0 {
		attentionRows = append(attentionRows, noticesPanelItem{Label: "no unresolved notices"})
	}

	warningTaskID := ""
	if attentionItems == 1 {
		for _, row := range attentionRows {
			if taskID := strings.TrimSpace(row.TaskID); taskID != "" {
				warningTaskID = taskID
				break
			}
		}
	}

	warningItems := make([]noticesPanelItem, 0, max(1, len(m.warnings)))
	if len(m.warnings) == 0 {
		warningItems = append(warningItems, noticesPanelItem{Label: "none"})
	} else {
		for _, warning := range m.warnings {
			warningItems = append(warningItems, noticesPanelItem{
				Label:  warning,
				TaskID: warningTaskID,
			})
		}
	}
	sections = append(sections, noticesPanelSection{
		ID:    noticesSectionWarnings,
		Title: noticesSectionTitle(noticesSectionWarnings),
		Items: warningItems,
	})

	attentionSummary := []string{}
	if attentionTotal > 0 {
		attentionSummary = append(
			attentionSummary,
			fmt.Sprintf("scope items: %d", attentionItems),
			fmt.Sprintf("unresolved: %d", attentionTotal),
			fmt.Sprintf("blocked: %d", attentionBlocked),
		)
	}
	sections = append(sections, noticesPanelSection{
		ID:      noticesSectionAttention,
		Title:   noticesSectionTitle(noticesSectionAttention),
		Summary: attentionSummary,
		Items:   attentionRows,
	})

	task, ok := m.selectedTaskInCurrentColumn()
	selectionItems := []noticesPanelItem{}
	selectionSummary := []string{}
	if ok {
		selectionItems = append(selectionItems, noticesPanelItem{
			Label:  task.Title,
			TaskID: task.ID,
		})
		if meta := m.cardMeta(task); meta != "" {
			selectionSummary = append(selectionSummary, meta)
		}
		if m.taskFields.ShowDescription {
			desc := strings.TrimSpace(task.Description)
			if desc == "" {
				desc = "-"
			}
			selectionSummary = append(selectionSummary, "description: "+desc)
		}
	} else {
		selectionItems = append(selectionItems, noticesPanelItem{Label: "no task selected"})
		selectionSummary = append(selectionSummary, "tip: use f to drill into scope")
	}
	sections = append(sections, noticesPanelSection{
		ID:      noticesSectionSelection,
		Title:   noticesSectionTitle(noticesSectionSelection),
		Summary: selectionSummary,
		Items:   selectionItems,
	})

	activityRows := m.recentActivityPanelEntries()
	activityItems := make([]noticesPanelItem, 0, max(1, len(activityRows)))
	if len(activityRows) == 0 {
		activityItems = append(activityItems, noticesPanelItem{Label: "(no activity yet)"})
	} else {
		for _, entry := range activityRows {
			activityItems = append(activityItems, noticesPanelItem{
				Label:       m.noticesActivityItemLabel(entry),
				Activity:    entry,
				HasActivity: true,
			})
		}
	}
	sections = append(sections, noticesPanelSection{
		ID:    noticesSectionRecentActivity,
		Title: noticesSectionTitle(noticesSectionRecentActivity),
		Items: activityItems,
	})

	return sections
}

// noticesSectionsForInteraction computes section state from current board data for keyboard interaction.
func (m Model) noticesSectionsForInteraction() []noticesPanelSection {
	taskByID := m.tasksByID()
	attentionItems, attentionTotal, attentionBlocked, attentionTop := m.scopeAttentionSummary(taskByID)
	return m.noticesPanelSections(attentionItems, attentionTotal, attentionBlocked, attentionTop)
}

// clampNoticesSelection keeps notices section focus and per-section cursors in bounds.
func (m *Model) clampNoticesSelection() {
	m.clampNoticesSelectionForSections(m.noticesSectionsForInteraction())
}

// clampNoticesSelectionForSections keeps notices selection indices valid for precomputed sections.
func (m *Model) clampNoticesSelectionForSections(sections []noticesPanelSection) {
	for _, section := range sections {
		idx := clamp(m.noticesSelectionIndex(section.ID), 0, len(section.Items)-1)
		m.setNoticesSelectionIndex(section.ID, idx)
	}
	if noticesSectionPosition(m.noticesSection) < 0 {
		m.noticesSection = noticesSectionRecentActivity
	}
}

// clampNoticesActivitySelection keeps compatibility for existing activity-only call sites.
func (m *Model) clampNoticesActivitySelection() {
	m.clampNoticesSelection()
}

// selectedNoticesActivity returns the currently selected notices-panel activity entry.
func (m Model) selectedNoticesActivity() (activityEntry, bool) {
	sections := m.noticesSectionsForInteraction()
	for _, section := range sections {
		if section.ID != noticesSectionRecentActivity {
			continue
		}
		if len(section.Items) == 0 {
			return activityEntry{}, false
		}
		idx := clamp(m.noticesSelectionIndex(section.ID), 0, len(section.Items)-1)
		item := section.Items[idx]
		if !item.HasActivity {
			return activityEntry{}, false
		}
		return item.Activity, true
	}
	return activityEntry{}, false
}

// moveNoticesSelection moves section/item focus inside the notices panel.
func (m *Model) moveNoticesSelection(delta int) bool {
	if delta == 0 {
		return false
	}
	sections := m.noticesSectionsForInteraction()
	if len(sections) == 0 {
		return false
	}
	m.clampNoticesSelectionForSections(sections)
	sectionPos := noticesSectionPosition(m.noticesSection)
	if sectionPos < 0 {
		sectionPos = noticesSectionPosition(noticesSectionRecentActivity)
		if sectionPos < 0 {
			sectionPos = 0
		}
		m.noticesSection = sections[sectionPos].ID
	}
	active := sections[sectionPos]
	itemIdx := clamp(m.noticesSelectionIndex(active.ID), 0, len(active.Items)-1)

	if delta > 0 {
		if itemIdx < len(active.Items)-1 {
			m.setNoticesSelectionIndex(active.ID, itemIdx+1)
			return true
		}
		if sectionPos < len(sections)-1 {
			next := sections[sectionPos+1]
			m.noticesSection = next.ID
			idx := clamp(m.noticesSelectionIndex(next.ID), 0, len(next.Items)-1)
			m.setNoticesSelectionIndex(next.ID, idx)
			return true
		}
		return false
	}

	if itemIdx > 0 {
		m.setNoticesSelectionIndex(active.ID, itemIdx-1)
		return true
	}
	if sectionPos > 0 {
		prev := sections[sectionPos-1]
		m.noticesSection = prev.ID
		idx := clamp(m.noticesSelectionIndex(prev.ID), 0, len(prev.Items)-1)
		m.setNoticesSelectionIndex(prev.ID, idx)
		return true
	}
	return false
}

// activateNoticesSelection runs enter behavior for the active notices row.
func (m Model) activateNoticesSelection() (tea.Model, tea.Cmd) {
	sections := m.noticesSectionsForInteraction()
	if len(sections) == 0 {
		m.status = "no notices available"
		return m, nil
	}
	m.clampNoticesSelectionForSections(sections)
	sectionPos := noticesSectionPosition(m.noticesSection)
	if sectionPos < 0 {
		sectionPos = noticesSectionPosition(noticesSectionRecentActivity)
		if sectionPos < 0 {
			sectionPos = 0
		}
		m.noticesSection = sections[sectionPos].ID
	}
	section := sections[sectionPos]
	if len(section.Items) == 0 {
		m.status = "no notices available"
		return m, nil
	}
	item := section.Items[clamp(m.noticesSelectionIndex(section.ID), 0, len(section.Items)-1)]
	if item.HasActivity {
		m.activityInfoItem = item.Activity
		m.mode = modeActivityEventInfo
		m.status = "activity event"
		return m, nil
	}
	taskID := strings.TrimSpace(item.TaskID)
	if taskID == "" {
		m.status = "selected notice has no action"
		return m, nil
	}
	if !m.openTaskInfo(taskID, "task info") {
		m.status = "task not found"
		return m, nil
	}
	m.noticesFocused = false
	return m, nil
}

// normalizeActivityActorType canonicalizes actor types and defaults to user for display.
func normalizeActivityActorType(actorType domain.ActorType) domain.ActorType {
	actorType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
	switch actorType {
	case domain.ActorTypeUser, domain.ActorTypeAgent, domain.ActorTypeSystem:
		return actorType
	default:
		return domain.ActorTypeUser
	}
}

// displayActivityOwner returns display-safe owner fields for activity rendering.
func (m Model) displayActivityOwner(entry activityEntry) (domain.ActorType, string) {
	actorType := normalizeActivityActorType(entry.ActorType)
	actorName := strings.TrimSpace(entry.ActorName)
	actorID := strings.TrimSpace(entry.ActorID)
	if actorType == domain.ActorTypeUser {
		switch {
		case strings.EqualFold(actorName, "tillsyn-user"), strings.EqualFold(actorID, "tillsyn-user"):
			if name := strings.TrimSpace(m.identityDisplayName); name != "" && (actorName == "" || strings.EqualFold(actorName, "tillsyn-user")) {
				actorName = name
			}
		}
	}
	if actorName != "" {
		return actorType, actorName
	}
	if actorID != "" {
		return actorType, actorID
	}
	return actorType, "unknown"
}

// displayActivityOwnerWithContext returns owner text with compact actor-id context when informative.
func (m Model) displayActivityOwnerWithContext(entry activityEntry) (domain.ActorType, string) {
	actorType, owner := m.displayActivityOwner(entry)
	actorID := strings.TrimSpace(entry.ActorID)
	if actorID == "" || strings.EqualFold(owner, actorID) || strings.EqualFold(actorID, "unknown") {
		return actorType, owner
	}
	return actorType, owner + " (" + actorID + ")"
}

// activityOwnerLabel returns a compact owner label used in notices rows.
func (m Model) activityOwnerLabel(entry activityEntry, width int) string {
	actorType, owner := m.displayActivityOwner(entry)
	label := string(actorType) + "|" + owner
	return truncate(label, max(8, width))
}

// renderOverviewPanel renders the right-side notices panel for board scope context.
func (m Model) renderOverviewPanel(
	project domain.Project,
	accent, muted, dim color.Color,
	width int,
	attentionItems, attentionTotal, attentionBlocked int,
	attentionTop []string,
	focused bool,
) string {
	panelWidth := max(24, width)
	contentWidth := max(12, panelWidth-6)
	normalStyle := lipgloss.NewStyle().Foreground(muted)
	selectedStyle := lipgloss.NewStyle().Foreground(accent).Bold(true)
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Notices"),
		normalStyle.Render("project: " + truncate(projectDisplayName(project), contentWidth)),
	}
	if path, _ := m.projectionPathWithProject(project.Name); path != "" {
		lines = append(lines, normalStyle.Render("path: "+truncate(path, contentWidth)))
	}

	sections := m.noticesPanelSections(attentionItems, attentionTotal, attentionBlocked, attentionTop)
	viewModel := m
	viewModel.clampNoticesSelectionForSections(sections)
	for _, section := range sections {
		lines = append(lines, "")
		lines = append(
			lines,
			viewModel.renderNoticesSection(
				section,
				focused,
				accent,
				contentWidth,
				selectedStyle,
				normalStyle,
			)...,
		)
	}
	lines = append(lines, "")
	lines = append(lines, normalStyle.Render("tab/shift+tab panels • enter details • g full activity log"))

	borderColor := dim
	if focused {
		borderColor = accent
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 1).
		Width(panelWidth)
	return style.Render(strings.Join(lines, "\n"))
}

// renderNoticesSection renders one notices-panel section with local list-window scrolling.
func (m Model) renderNoticesSection(
	section noticesPanelSection,
	focused bool,
	accent color.Color,
	contentWidth int,
	selectedStyle, normalStyle lipgloss.Style,
) []string {
	lines := make([]string, 0, len(section.Summary)+noticesSectionViewWindow+3)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	if focused && section.ID == m.noticesSection {
		lines = append(lines, headerStyle.Render("▸ "+section.Title))
	} else {
		lines = append(lines, headerStyle.Render(section.Title))
	}

	renderItems := func() {
		if len(section.Items) == 0 {
			lines = append(lines, normalStyle.Render("(empty)"))
			return
		}
		selectedIdx := clamp(m.noticesSelectionIndex(section.ID), 0, len(section.Items)-1)
		start, end := windowBounds(len(section.Items), selectedIdx, noticesSectionViewWindow)
		if focused && start > 0 {
			lines = append(lines, normalStyle.Render(truncate("↑ more", contentWidth)))
		}
		for idx := start; idx < end; idx++ {
			item := section.Items[idx]
			prefix := ""
			style := normalStyle
			if focused {
				prefix = "  "
			}
			if focused && section.ID == m.noticesSection && idx == selectedIdx {
				prefix = "› "
				style = selectedStyle
			}
			lineWidth := max(1, contentWidth-utf8.RuneCountInString(prefix))
			lines = append(lines, style.Render(prefix+truncate(item.Label, lineWidth)))
		}
		if focused && end < len(section.Items) {
			lines = append(lines, normalStyle.Render(truncate("↓ more", contentWidth)))
		}
	}

	// Keep selection details in task-title-first order while still rendering the title as the selectable row.
	if section.ID == noticesSectionSelection {
		renderItems()
		for _, summary := range section.Summary {
			lines = append(lines, normalStyle.Render(truncate(summary, contentWidth)))
		}
		return lines
	}

	for _, summary := range section.Summary {
		lines = append(lines, normalStyle.Render(truncate(summary, contentWidth)))
	}
	renderItems()
	return lines
}

// renderInfoLine renders output for the current model state.
func (m Model) renderInfoLine(project domain.Project, muted color.Color) string {
	parts := []string{fmt.Sprintf("tasks: %d", len(m.tasks))}
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		parts = append(parts, "selected: none")
		if strings.TrimSpace(m.projectionRootTaskID) != "" {
			parts = append(parts, fmt.Sprintf("%s full board", m.keys.clearFocus.Help().Key))
		}
		return lipgloss.NewStyle().Foreground(muted).Render(strings.Join(parts, " • "))
	}
	parts = append(parts, fmt.Sprintf("selected: %s", truncate(task.Title, 36)))
	levelByTaskID := m.searchLevelByTaskID([]domain.Task{task})
	level := strings.TrimSpace(levelByTaskID[task.ID])
	if level == "" {
		level = baseSearchLevelForTask(task)
	}
	if label := strings.ToLower(strings.TrimSpace(canonicalSearchLevelLabels[level])); label != "" {
		parts = append(parts, "level: "+label)
	}
	if children := m.directChildCount(task.ID); children > 0 {
		parts = append(parts, fmt.Sprintf("children: %d", children))
		if strings.TrimSpace(m.projectionRootTaskID) == "" {
			parts = append(parts, fmt.Sprintf("%s focus subtree", m.keys.focusSubtree.Help().Key))
		}
	}
	if strings.TrimSpace(m.projectionRootTaskID) != "" {
		parts = append(parts, fmt.Sprintf("%s full board", m.keys.clearFocus.Help().Key))
	}
	return lipgloss.NewStyle().Foreground(muted).Render(strings.Join(parts, " • "))
}

// renderHelpOverlay renders output for the current model state.
func (m Model) renderHelpOverlay(accent, muted, dim color.Color, _ lipgloss.Style, maxWidth int) string {
	width := clamp(maxWidth, 56, 100)
	if width <= 0 {
		width = 72
	}
	screenTitle, screenHelp := m.helpOverlayScreenTitleAndLines()
	title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render("TILLSYN Help")
	subtitle := lipgloss.NewStyle().Foreground(muted).Render("screen: " + screenTitle)
	lines := []string{title, subtitle, ""}
	for _, line := range screenHelp {
		lines = append(lines, "- "+line)
	}
	selectionState := "off"
	if m.mouseSelectionMode {
		selectionState = "on"
	}
	lines = append(
		lines,
		"",
		lipgloss.NewStyle().Foreground(muted).Render(fmt.Sprintf("selection mode: %s (%s toggles)", selectionState, m.keys.toggleSelectMode.Help().Key)),
		lipgloss.NewStyle().Foreground(muted).Render("press ? or esc to close help"),
	)
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dim).
		Padding(0, 1)
	if maxWidth > 0 {
		style = style.Width(width)
	}
	return style.Render(strings.Join(lines, "\n"))
}

// helpOverlayScreenTitleAndLines returns mode-scoped help content for the expanded help overlay.
func (m Model) helpOverlayScreenTitleAndLines() (string, []string) {
	switch m.mode {
	case modeNone:
		panelLine := "tab/shift+tab cycle focused panel; left/right move adjacent panel; enter opens notices detail"
		if m.noticesFocused {
			panelLine = "tab/shift+tab cycle focused panel; j/k or arrows move notices; enter opens selected notice item"
		}
		return "board", []string{
			"h/l or left/right move columns; j/k or up/down move task selection",
			"n new task; s new subtask; i/enter task info; e edit task",
			"space multi-select; [ / ] move task; d delete; D hard delete; u restore",
			"f focus subtree; F full board; t toggle archived",
			"/ search; p project picker; : command palette; . quick actions",
			panelLine,
			"ctrl+y toggles text selection mode; ctrl+c/ctrl+v copy/paste in text inputs",
			"z undo; Z redo; g activity log; q quit",
		}
	case modeAddTask:
		return "new task", []string{
			"tab/shift+tab move fields; enter saves; esc cancels",
			"h/l changes priority when priority field is focused",
			"d opens due picker; supports date-only or local date+time",
			"labels field: enter or ctrl+l opens label picker; ctrl+g accepts suggestion",
			"depends_on/blocked_by fields: enter or o opens dependency picker",
			"ctrl+r opens resource picker",
		}
	case modeEditTask:
		return "edit task", []string{
			"tab/shift+tab move fields; enter saves; esc cancels",
			"h/l changes priority when priority field is focused",
			"d opens due picker; supports date-only or local date+time",
			"labels field: enter or ctrl+l opens label picker; ctrl+g accepts suggestion",
			"depends_on/blocked_by fields: enter or o opens dependency picker",
			"ctrl+r opens resource picker",
		}
	case modeSearch:
		return "search", []string{
			"tab cycles query, states, levels, scope, archived, and apply",
			"space or enter toggles the focused state/level/scope option",
			"h/l cycles state/level cursors and toggles scope/archived",
			"ctrl+u clears query; ctrl+r resets filters; esc cancels",
		}
	case modeRenameTask:
		return "rename task", []string{
			"type new title",
			"enter saves; esc cancels",
		}
	case modeDuePicker:
		return "due picker", []string{
			"tab cycles include-time, date, time, and options list focus",
			"space toggles include time when toggle is focused",
			"type date/time to update dynamic suggestions",
			"j/k navigates options list; enter applies; esc cancels",
		}
	case modeProjectPicker:
		return "project picker", []string{
			"j/k or mouse wheel changes selection",
			"enter chooses project",
			"N opens new-project form",
			"A toggles archived project visibility in picker",
			"esc closes picker",
		}
	case modeTaskInfo:
		return "task info", []string{
			"j/k selects subtasks; space toggles selected subtask completion",
			"enter opens selected subtask; backspace moves to parent",
			"e edit; s create subtask; c thread view",
			"b dependency inspector; r attach resource",
			"[ / ] move task between columns; f focus subtree; esc back/close",
		}
	case modeAddProject:
		return "new project", []string{
			"tab/shift+tab moves fields; enter saves; esc cancels",
			"icon field is shown in path context, notices, and picker and supports emoji",
			"root_path field: r opens directory picker",
		}
	case modeEditProject:
		return "edit project", []string{
			"tab/shift+tab moves fields; enter saves; esc cancels",
			"icon field is shown in path context, notices, and picker and supports emoji",
			"root_path field: r opens directory picker",
		}
	case modeSearchResults:
		return "search results", []string{
			"j/k moves result selection",
			"enter jumps to selected result",
			"esc closes results",
		}
	case modeCommandPalette:
		return "command palette", []string{
			"type to filter command names and aliases",
			"j/k moves selection; tab autocompletes first match",
			"enter runs command; esc closes palette",
		}
	case modeQuickActions:
		return "quick actions", []string{
			"j/k moves action selection",
			"enter runs selected action",
			"esc closes quick actions",
		}
	case modeConfirmAction:
		return "confirm action", []string{
			"h/l switches confirm vs cancel",
			"enter applies highlighted choice",
			"y confirms immediately; n cancels; esc cancels",
		}
	case modeWarning:
		return "warning", []string{
			"warning modal blocks accidental context mistakes",
			"enter or esc closes the warning",
		}
	case modeActivityLog:
		return "activity log", []string{
			"esc closes activity log",
			"z undo and Z redo remain available",
		}
	case modeActivityEventInfo:
		return "activity event", []string{
			"enter/g jumps to event node when available",
			"esc returns to notices panel focus",
		}
	case modeResourcePicker:
		if m.resourcePickerBack == modeAddProject || m.resourcePickerBack == modeEditProject || m.resourcePickerBack == modePathsRoots || m.resourcePickerBack == modeBootstrapSettings {
			return "path picker", []string{
				"type to filter entries",
				"j/k or arrows move selection",
				"right opens directory; left moves to parent directory",
				"enter chooses path (file chooses parent dir)",
				"ctrl+a chooses current directory; esc closes",
			}
		}
		return "resource picker", []string{
			"type to filter entries",
			"j/k or arrows move selection",
			"enter opens directory or attaches selected file",
			"ctrl+a attaches selected or current directory; esc closes",
		}
	case modeLabelPicker:
		return "label picker", []string{
			"type to filter labels",
			"j/k moves selection",
			"enter adds selected label",
			"ctrl+u clears filter; esc closes",
		}
	case modePathsRoots:
		return "paths / roots", []string{
			"type project root path (empty clears mapping)",
			"r opens directory picker",
			"enter saves; esc cancels",
		}
	case modeLabelsConfig:
		return "labels config", []string{
			"tab/shift+tab moves between global, project, branch, and phase fields",
			"enter saves; esc cancels",
			"global/project values save to config",
			"branch/phase values apply to current hierarchy labels",
		}
	case modeHighlightColor:
		return "highlight color", []string{
			"set ANSI index or #RRGGBB value",
			"empty value resets default color",
			"enter saves; esc cancels",
		}
	case modeBootstrapSettings:
		return "bootstrap settings", []string{
			"tab cycles name, default path, and save focus",
			"r opens directory picker for default path",
			"enter chooses path or saves settings",
			"d clears default path",
			"esc cancels when bootstrap is optional",
		}
	case modeDependencyInspector:
		return "dependency inspector", []string{
			"tab cycles query, state filters, scope, archived, and list focus",
			"type query text to filter candidates",
			"d toggles depends_on; b toggles blocked_by; x switches active relation field",
			"space toggles active relation value for selected candidate",
			"a applies changes; enter jumps to task; esc cancels",
		}
	case modeThread:
		return "thread", []string{
			"type comment text in composer",
			"enter posts comment",
			"pgup/pgdown or mouse wheel scrolls",
			"ctrl+r reloads comments",
			"esc returns to previous screen",
		}
	default:
		return "current screen", []string{
			"enter confirms primary action",
			"esc closes current screen",
		}
	}
}

// taskListSecondary returns task list secondary.
func (m Model) taskListSecondary(task domain.Task) string {
	if m.taskFields.ShowDescription {
		if desc := strings.TrimSpace(task.Description); desc != "" {
			return desc
		}
	}
	if meta := m.cardMeta(task); meta != "" {
		return meta
	}
	return ""
}

// taskIndexAtRow returns task index at row.
func (m Model) taskIndexAtRow(tasks []domain.Task, row int) int {
	if len(tasks) == 0 {
		return 0
	}
	if row <= 0 {
		return 0
	}
	current := 0
	for idx, task := range tasks {
		start := current
		span := 1
		if m.taskListSecondary(task) != "" {
			span++
		}
		if idx < len(tasks)-1 {
			span++
		}
		end := start + span - 1
		if row >= start && row <= end {
			return idx
		}
		current += span
	}
	return len(tasks) - 1
}

// cardMeta handles card meta.
func (m Model) cardMeta(task domain.Task) string {
	parts := make([]string, 0, 4)
	if marker := taskHierarchyMarker(task); marker != "" {
		parts = append(parts, marker)
	}
	if m.taskFields.ShowPriority {
		parts = append(parts, string(task.Priority))
	}
	if task.Kind != domain.WorkKindSubtask {
		done, total := m.subtaskProgress(task.ID)
		if total > 0 {
			parts = append(parts, fmt.Sprintf("%d/%d", done, total))
		}
	}
	if m.taskFields.ShowDueDate && task.DueAt != nil {
		dueLabel := task.DueAt.UTC().Format("01-02")
		if task.DueAt.UTC().Before(time.Now().UTC()) {
			dueLabel = "!" + dueLabel
		}
		parts = append(parts, dueLabel)
	}
	if m.taskFields.ShowLabels && len(task.Labels) > 0 {
		parts = append(parts, summarizeLabels(task.Labels, 2))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, "|") + "]"
}

// taskHierarchyMarker returns a compact level marker for hierarchy-scoped work items.
func taskHierarchyMarker(task domain.Task) string {
	switch baseSearchLevelForTask(task) {
	case "branch":
		return "branch"
	case "phase", "subphase":
		// Subphases are phase-level nodes for board labeling and drill-down workflows.
		return "phase"
	default:
		return ""
	}
}

// taskDueWarning reports due warning text for one task in board/info contexts.
func (m Model) taskDueWarning(task domain.Task, now time.Time) string {
	if task.ArchivedAt != nil || task.DueAt == nil {
		return ""
	}
	now = now.UTC()
	due := task.DueAt.UTC()
	if due.Before(now) {
		return "warning: overdue"
	}
	maxWindow := time.Duration(0)
	for _, window := range m.dueSoonWindows {
		if window > maxWindow {
			maxWindow = window
		}
	}
	if maxWindow > 0 && due.Sub(now) <= maxWindow {
		return "warning: due soon"
	}
	return ""
}

// taskInfoTask resolves the task currently shown in the task-info modal.
func (m Model) taskInfoTask() (domain.Task, bool) {
	taskID := strings.TrimSpace(m.taskInfoTaskID)
	if taskID == "" {
		return m.selectedTaskInCurrentColumn()
	}
	return m.taskByID(taskID)
}

// openTaskInfo enters task-info mode and stores the origin task for contextual esc back behavior.
func (m *Model) openTaskInfo(taskID string, status string) bool {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false
	}
	if _, ok := m.taskByID(taskID); !ok {
		return false
	}
	m.mode = modeTaskInfo
	m.taskInfoTaskID = taskID
	m.taskInfoOriginTaskID = taskID
	m.taskInfoSubtaskIdx = 0
	if strings.TrimSpace(status) == "" {
		status = "task info"
	}
	m.status = status
	return true
}

// closeTaskInfo exits task-info mode and clears tracked origin/task ids.
func (m *Model) closeTaskInfo(status string) {
	m.mode = modeNone
	m.taskInfoTaskID = ""
	m.taskInfoOriginTaskID = ""
	m.taskInfoSubtaskIdx = 0
	if strings.TrimSpace(status) == "" {
		status = "ready"
	}
	m.status = status
}

// stepBackTaskInfo moves task-info focus to the parent task when available.
func (m *Model) stepBackTaskInfo(task domain.Task) bool {
	parentID := strings.TrimSpace(task.ParentID)
	if parentID == "" {
		return false
	}
	if _, ok := m.taskByID(parentID); !ok {
		return false
	}
	m.taskInfoTaskID = parentID
	m.taskInfoSubtaskIdx = 0
	// Keep the cursor aligned to the child we navigated from when it remains visible.
	for idx, child := range m.subtasksForParent(parentID) {
		if child.ID == task.ID {
			m.taskInfoSubtaskIdx = idx
			break
		}
	}
	m.status = "parent task info"
	return true
}

// subtasksForParent returns direct subtask children for a parent task.
func (m Model) subtasksForParent(parentID string) []domain.Task {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil
	}
	out := make([]domain.Task, 0)
	for _, task := range m.tasks {
		if strings.TrimSpace(task.ParentID) != parentID {
			continue
		}
		if task.Kind != domain.WorkKindSubtask {
			continue
		}
		if !m.showArchived && task.ArchivedAt != nil {
			continue
		}
		out = append(out, task)
	}
	sortTaskSlice(out)
	return out
}

// normalizeColumnStateID canonicalizes column names into lifecycle-state identifiers.
func normalizeColumnStateID(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	switch strings.Trim(b.String(), "-") {
	case "to-do", "todo":
		return "todo"
	case "in-progress", "progress", "doing":
		return "progress"
	case "done", "complete", "completed":
		return "done"
	case "archived", "archive":
		return "archived"
	default:
		return strings.Trim(b.String(), "-")
	}
}

// lifecycleStateForColumnName resolves lifecycle state from one board column name.
func lifecycleStateForColumnName(name string) domain.LifecycleState {
	switch normalizeColumnStateID(name) {
	case "todo":
		return domain.StateTodo
	case "progress":
		return domain.StateProgress
	case "done":
		return domain.StateDone
	case "archived":
		return domain.StateArchived
	default:
		return domain.StateTodo
	}
}

// lifecycleStateForColumnID resolves lifecycle state for one column id in the active board.
func (m Model) lifecycleStateForColumnID(columnID string) (domain.LifecycleState, bool) {
	columnID = strings.TrimSpace(columnID)
	if columnID == "" {
		return "", false
	}
	for _, column := range m.columns {
		if column.ID == columnID {
			return lifecycleStateForColumnName(column.Name), true
		}
	}
	return "", false
}

// lifecycleStateForTask resolves lifecycle state using current board columns with task fallback.
func (m Model) lifecycleStateForTask(task domain.Task) domain.LifecycleState {
	if task.LifecycleState != "" {
		return task.LifecycleState
	}
	if state, ok := m.lifecycleStateForColumnID(task.ColumnID); ok && state != "" {
		return state
	}
	return domain.StateTodo
}

// lifecycleStateLabel renders one lifecycle state with human-readable display text.
func lifecycleStateLabel(state domain.LifecycleState) string {
	switch state {
	case domain.StateTodo:
		return canonicalSearchStateLabels["todo"]
	case domain.StateProgress:
		return canonicalSearchStateLabels["progress"]
	case domain.StateDone:
		return canonicalSearchStateLabels["done"]
	case domain.StateArchived:
		return "Archived"
	default:
		stateText := strings.TrimSpace(string(state))
		if stateText == "" {
			return "-"
		}
		return stateText
	}
}

// completionLabel renders a compact yes/no completion flag.
func completionLabel(done bool) string {
	if done {
		return "yes"
	}
	return "no"
}

// columnIndexByID finds one column index by id.
func (m Model) columnIndexByID(columnID string) (int, bool) {
	columnID = strings.TrimSpace(columnID)
	if columnID == "" {
		return 0, false
	}
	for idx, column := range m.columns {
		if column.ID == columnID {
			return idx, true
		}
	}
	return 0, false
}

// firstColumnIndexForState finds the first column index matching one lifecycle state.
func (m Model) firstColumnIndexForState(state domain.LifecycleState) (int, bool) {
	for idx, column := range m.columns {
		if lifecycleStateForColumnName(column.Name) == state {
			return idx, true
		}
	}
	return 0, false
}

// firstIncompleteColumnIndex finds the preferred destination for reopening a completed subtask.
func (m Model) firstIncompleteColumnIndex() (int, bool) {
	if idx, ok := m.firstColumnIndexForState(domain.StateProgress); ok {
		return idx, true
	}
	if idx, ok := m.firstColumnIndexForState(domain.StateTodo); ok {
		return idx, true
	}
	for idx, column := range m.columns {
		state := lifecycleStateForColumnName(column.Name)
		if state != domain.StateDone && state != domain.StateArchived {
			return idx, true
		}
	}
	return 0, false
}

// toggleFocusedSubtaskCompletion toggles done/non-done state for the focused subtask in task-info mode.
func (m Model) toggleFocusedSubtaskCompletion(parent domain.Task) (tea.Model, tea.Cmd) {
	subtasks := m.subtasksForParent(parent.ID)
	if len(subtasks) == 0 {
		m.status = "no subtasks"
		return m, nil
	}
	subtaskIdx := clamp(m.taskInfoSubtaskIdx, 0, len(subtasks)-1)
	subtask := subtasks[subtaskIdx]

	fromIdx, ok := m.columnIndexByID(subtask.ColumnID)
	if !ok {
		m.status = "subtask column unavailable"
		return m, nil
	}

	status := "subtask marked complete"
	toIdx, ok := m.firstColumnIndexForState(domain.StateDone)
	if m.lifecycleStateForTask(subtask) == domain.StateDone {
		status = "subtask marked incomplete"
		toIdx, ok = m.firstIncompleteColumnIndex()
	}
	if !ok {
		if status == "subtask marked complete" {
			m.status = "no done column configured"
		} else {
			m.status = "no active column for reopening"
		}
		return m, nil
	}
	if toIdx == fromIdx {
		m.status = status
		return m, nil
	}

	updated, cmd := m.moveTaskIDs([]string{subtask.ID}, toIdx-fromIdx, "toggle subtask completion", subtask.Title, false)
	next, ok := updated.(Model)
	if !ok {
		return updated, cmd
	}
	next.status = status
	next.mode = modeTaskInfo
	next.taskInfoTaskID = parent.ID
	next.taskInfoSubtaskIdx = subtaskIdx
	return next, cmd
}

// subtaskProgress returns completed/total direct subtasks for a parent task.
func (m Model) subtaskProgress(parentID string) (int, int) {
	subtasks := m.subtasksForParent(parentID)
	if len(subtasks) == 0 {
		return 0, 0
	}
	done := 0
	for _, task := range subtasks {
		if m.lifecycleStateForTask(task) == domain.StateDone {
			done++
		}
	}
	return done, len(subtasks)
}

// dueCounts returns overdue and due-soon counts for loaded tasks.
func (m Model) dueCounts(now time.Time) (int, int) {
	if len(m.tasks) == 0 {
		return 0, 0
	}
	overdue := 0
	dueSoon := 0
	windows := append([]time.Duration(nil), m.dueSoonWindows...)
	sort.Slice(windows, func(i, j int) bool { return windows[i] < windows[j] })
	maxWindow := time.Duration(0)
	if len(windows) > 0 {
		maxWindow = windows[len(windows)-1]
	}
	for _, task := range m.tasks {
		if task.ArchivedAt != nil || task.DueAt == nil {
			continue
		}
		due := task.DueAt.UTC()
		if due.Before(now) {
			overdue++
			continue
		}
		if maxWindow > 0 && due.Sub(now) <= maxWindow {
			dueSoon++
		}
	}
	return overdue, dueSoon
}

// renderTaskDetails renders output for the current model state.
func (m Model) renderTaskDetails(accent, muted, dim color.Color) string {
	task, ok := m.selectedTaskInCurrentColumn()
	if !ok {
		return ""
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Task Details"),
		task.Title,
	}

	meta := make([]string, 0, 3)
	meta = append(meta, "state: "+lifecycleStateLabel(m.lifecycleStateForTask(task)))
	if m.taskFields.ShowPriority {
		meta = append(meta, "priority: "+string(task.Priority))
	}
	if m.taskFields.ShowDueDate {
		due := "-"
		if task.DueAt != nil {
			due = formatDueValue(task.DueAt)
		}
		meta = append(meta, "due: "+due)
	}
	if m.taskFields.ShowLabels {
		labels := "-"
		if len(task.Labels) > 0 {
			labels = strings.Join(task.Labels, ", ")
		}
		meta = append(meta, "labels: "+labels)
	}
	if len(meta) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render(strings.Join(meta, "  ")))
	}

	if m.taskFields.ShowDescription {
		if desc := strings.TrimSpace(task.Description); desc != "" {
			lines = append(lines, desc)
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render("description: -"))
		}
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(dim).
		Padding(0, 1)
	if m.width > 0 {
		style = style.Width(max(24, m.width-2))
	}
	return style.Render(strings.Join(lines, "\n"))
}

// renderModeOverlay renders output for the current model state.
func (m Model) renderModeOverlay(accent, muted, dim color.Color, helpStyle lipgloss.Style, maxWidth int) string {
	switch m.mode {
	case modeActivityLog:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 44, 96))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Activity Log")}
		if len(m.activityLog) == 0 {
			lines = append(lines, hintStyle.Render("(no activity yet)"))
		} else {
			rendered := 0
			for idx := len(m.activityLog) - 1; idx >= 0; idx-- {
				entry := m.activityLog[idx]
				lines = append(lines, fmt.Sprintf("%s  %s • %s", formatActivityTimestamp(entry.At), entry.Summary, truncate(entry.Target, 42)))
				rendered++
				if rendered >= activityLogViewWindow {
					break
				}
			}
		}
		lines = append(lines, hintStyle.Render("esc close • undo/redo available"))
		return style.Render(strings.Join(lines, "\n"))

	case modeActivityEventInfo:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 54, 110))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		entry := m.activityInfoItem
		actorType, owner := m.displayActivityOwnerWithContext(entry)
		operation := strings.TrimSpace(string(entry.Operation))
		if operation == "" {
			operation = "update"
		}
		nodeLabel, pathLabel := m.activityEventTargetDetails(entry)
		lines := []string{
			titleStyle.Render("Activity Event"),
			hintStyle.Render("owner: " + string(actorType) + " • " + owner),
			hintStyle.Render("when: " + formatActivityTimestampLong(entry.At)),
			hintStyle.Render("operation: " + operation),
			hintStyle.Render("summary: " + entry.Summary),
			hintStyle.Render("node: " + nodeLabel),
			hintStyle.Render("path: " + pathLabel),
		}
		metaLines := m.formatActivityMetadata(entry)
		if len(metaLines) > 0 {
			lines = append(lines, "", hintStyle.Render("metadata"))
			for _, line := range metaLines {
				lines = append(lines, hintStyle.Render(line))
			}
		}
		lines = append(lines, "")
		if canJumpToActivityNode(entry) {
			lines = append(lines, hintStyle.Render("enter/g go to node • esc back"))
		} else {
			lines = append(lines, hintStyle.Render("node reference unavailable • esc back"))
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeTaskInfo:
		task, ok := m.taskInfoTask()
		if !ok {
			return ""
		}
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			boxStyle = boxStyle.Width(clamp(maxWidth, 24, 76))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		due := "-"
		if task.DueAt != nil {
			due = formatDueValue(task.DueAt)
		}
		taskState := m.lifecycleStateForTask(task)
		isComplete := taskState == domain.StateDone
		labels := "-"
		if len(task.Labels) > 0 {
			labels = strings.Join(task.Labels, ", ")
		}
		lines := []string{
			titleStyle.Render("Task Info"),
			task.Title,
			hintStyle.Render("kind: " + string(task.Kind) + " • state: " + lifecycleStateLabel(taskState) + " • complete: " + completionLabel(isComplete)),
			hintStyle.Render("priority: " + string(task.Priority) + " • due: " + due),
			hintStyle.Render("labels: " + labels),
		}
		if warning := m.taskDueWarning(task, time.Now().UTC()); warning != "" {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")).Render(warning))
		}
		lines = append(lines, hintStyle.Render("due entry supports time: YYYY-MM-DD HH:MM, YYYY-MM-DDTHH:MM, RFC3339 (UTC default)"))
		subtasks := m.subtasksForParent(task.ID)
		if len(subtasks) > 0 {
			lines = append(lines, "")
			done, total := m.subtaskProgress(task.ID)
			lines = append(lines, hintStyle.Render(fmt.Sprintf("subtasks (%d/%d done)", done, total)))
			subtaskIdx := clamp(m.taskInfoSubtaskIdx, 0, len(subtasks)-1)
			checkedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
			uncheckedStyle := lipgloss.NewStyle().Foreground(muted)
			focusStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
			for idx, subtask := range subtasks {
				subtaskState := m.lifecycleStateForTask(subtask)
				subtaskDone := subtaskState == domain.StateDone
				prefix := "  "
				if idx == subtaskIdx {
					prefix = "│ "
				}
				check := uncheckedStyle.Render("[ ]")
				if subtaskDone {
					check = checkedStyle.Render("[x]")
				}
				title := truncate(subtask.Title, 34)
				if idx == subtaskIdx {
					title = focusStyle.Render(title)
				}
				metaParts := []string{
					"state:" + lifecycleStateLabel(subtaskState),
					"complete:" + completionLabel(subtaskDone),
				}
				if subtask.DueAt != nil {
					metaParts = append(metaParts, "due:"+formatDueValue(subtask.DueAt))
				}
				line := fmt.Sprintf("%s%s %s %s", prefix, check, title, hintStyle.Render(strings.Join(metaParts, " • ")))
				lines = append(lines, line)
			}
			lines = append(lines, hintStyle.Render("j/k choose • space toggle complete • enter open subtask • backspace parent"))
		}
		inherited := m.labelSourcesForTask(task)
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("effective labels (global/project/branch/phase fallback)"))
		lines = append(lines, hintStyle.Render(formatLabelSource("global", inherited.Global)))
		lines = append(lines, hintStyle.Render(formatLabelSource("project", inherited.Project)))
		lines = append(lines, hintStyle.Render(formatLabelSource("phase", inherited.Phase)))

		dependsOn := uniqueTrimmed(task.Metadata.DependsOn)
		blockedBy := uniqueTrimmed(task.Metadata.BlockedBy)
		blockedReason := strings.TrimSpace(task.Metadata.BlockedReason)
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("dependencies"))
		lines = append(lines, hintStyle.Render("depends_on: "+m.summarizeTaskRefs(dependsOn, 4)))
		lines = append(lines, hintStyle.Render("blocked_by: "+m.summarizeTaskRefs(blockedBy, 4)))
		if blockedReason == "" {
			blockedReason = "-"
		}
		lines = append(lines, hintStyle.Render("blocked_reason: "+blockedReason))

		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("resources"))
		if len(task.Metadata.ResourceRefs) == 0 {
			lines = append(lines, hintStyle.Render("(none)"))
		} else {
			for idx, ref := range task.Metadata.ResourceRefs {
				if idx >= 4 {
					lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more", len(task.Metadata.ResourceRefs)-idx)))
					break
				}
				location := strings.TrimSpace(ref.Location)
				if ref.PathMode == domain.PathModeRelative && strings.TrimSpace(ref.BaseAlias) != "" {
					location = strings.TrimSpace(ref.BaseAlias) + ":" + location
				}
				lines = append(lines, hintStyle.Render(fmt.Sprintf("%s %s", ref.ResourceType, truncate(location, 48))))
			}
		}
		if strings.TrimSpace(task.ParentID) != "" {
			lines = append(lines, hintStyle.Render("parent: "+task.ParentID))
		}
		if objective := strings.TrimSpace(task.Metadata.Objective); objective != "" {
			lines = append(lines, "", hintStyle.Render("objective"), objective)
		}
		if len(task.Metadata.CompletionContract.CompletionCriteria) > 0 {
			unmet := 0
			for _, item := range task.Metadata.CompletionContract.CompletionCriteria {
				if strings.TrimSpace(item.Text) == "" {
					continue
				}
				if !item.Done {
					unmet++
				}
			}
			if unmet > 0 {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("completion: %d unmet checks", unmet)))
			}
		}
		if desc := strings.TrimSpace(task.Description); desc != "" {
			lines = append(lines, "", desc)
		}
		lines = append(lines, "", hintStyle.Render("e edit • s subtask • c thread • [ / ] move • b deps inspector • r attach resource • f focus subtree • esc back/close"))
		return boxStyle.Render(strings.Join(lines, "\n"))

	case modeBootstrapSettings:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 52, 108))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		focusStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		title := "Identity & Default Path"
		if m.bootstrapMandatory {
			title = "Startup Setup Required"
		}
		in := m.bootstrapDisplayInput
		in.SetWidth(max(20, maxWidth-26))
		nameLabel := hintStyle
		if m.bootstrapFocus == 0 {
			nameLabel = focusStyle
		}
		rootsLabel := hintStyle
		if m.bootstrapFocus == 1 {
			rootsLabel = focusStyle
		}
		saveLabel := hintStyle
		if m.bootstrapFocus == 2 {
			saveLabel = focusStyle
		}
		lines := []string{
			titleStyle.Render(title),
			nameLabel.Render("name:") + " " + in.View(),
			rootsLabel.Render("default path:"),
		}
		if len(m.bootstrapRoots) == 0 {
			lines = append(lines, hintStyle.Render("(none yet)"))
		} else {
			root := m.bootstrapRoots[clamp(m.bootstrapRootIndex, 0, len(m.bootstrapRoots)-1)]
			line := "> " + truncate(root, 84)
			if m.bootstrapFocus == 1 {
				line = focusStyle.Render(line)
			}
			lines = append(lines, line)
		}
		lines = append(lines, saveLabel.Render("[ save settings ]"))
		if m.bootstrapMandatory {
			lines = append(lines, hintStyle.Render("tab focus • r open picker • enter choose/save • d clear path"))
		} else {
			lines = append(lines, hintStyle.Render("tab focus • r open picker • enter choose/save • d clear path"))
		}
		if m.bootstrapMandatory {
			lines = append(lines, hintStyle.Render("esc disabled until required settings are saved"))
		} else {
			lines = append(lines, hintStyle.Render("esc cancel"))
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeResourcePicker:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 44, 108))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		currentPath := strings.TrimSpace(m.resourcePickerDir)
		if currentPath == "" {
			currentPath = m.resourcePickerRoot
		}
		if absPath, err := filepath.Abs(currentPath); err == nil {
			currentPath = absPath
		}
		title := "Attach Resource"
		if m.resourcePickerBack == modeAddProject || m.resourcePickerBack == modeEditProject || m.resourcePickerBack == modePathsRoots {
			title = "Pick Project Root"
		}
		if m.resourcePickerBack == modeBootstrapSettings {
			title = "Pick Search Root"
		}
		filterInput := m.resourcePickerFilter
		filterInput.SetWidth(max(20, min(72, maxWidth-18)))
		lines := []string{
			titleStyle.Render(title),
			hintStyle.Render("root: " + truncate(m.resourcePickerRoot, 72)),
			hintStyle.Render("current: ") + titleStyle.Render(truncate(currentPath, 72)),
			hintStyle.Render("filter: ") + filterInput.View(),
		}
		items := m.visibleResourcePickerItems()
		if len(items) == 0 {
			lines = append(lines, hintStyle.Render("(empty directory)"))
			lines = append(lines, hintStyle.Render("press enter or ctrl+a to choose current directory"))
		} else {
			for idx, entry := range items {
				cursor := "  "
				if idx == m.resourcePickerIndex {
					cursor = "> "
				}
				name := entry.Name
				if entry.IsDir {
					name += "/"
				}
				lines = append(lines, cursor+name)
				if idx >= 13 {
					lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more entries", len(items)-idx-1)))
					break
				}
			}
		}
		if m.resourcePickerBack == modeAddProject || m.resourcePickerBack == modeEditProject || m.resourcePickerBack == modePathsRoots || m.resourcePickerBack == modeBootstrapSettings {
			lines = append(lines, hintStyle.Render("enter choose path (file picks parent dir) • right open dir • left parent • ctrl+a choose current dir • ctrl+u clear filter • esc close"))
		} else {
			lines = append(lines, hintStyle.Render("enter open dir/file attach • ctrl+a attach selected/current • arrows navigate • ctrl+u clear filter • esc close"))
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeLabelPicker:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 38, 88))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		filterInput := m.labelPickerInput
		filterInput.SetWidth(max(18, min(56, maxWidth-22)))
		lines := []string{
			titleStyle.Render("Label Picker"),
			hintStyle.Render("filter: ") + filterInput.View(),
			hintStyle.Render("sources: global/project/branch/phase/suggested/default"),
		}
		if len(m.labelPickerItems) == 0 {
			lines = append(lines, hintStyle.Render("(no matching labels)"))
		} else {
			for idx, item := range m.labelPickerItems {
				cursor := "  "
				if idx == m.labelPickerIndex {
					cursor = "> "
				}
				lines = append(lines, fmt.Sprintf("%s%s (%s)", cursor, item.Label, item.Source))
				if idx >= 11 {
					lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more labels", len(m.labelPickerItems)-idx-1)))
					break
				}
			}
		}
		lines = append(lines, hintStyle.Render("type to filter • j/k navigate • enter add label • ctrl+u clear • esc close"))
		return style.Render(strings.Join(lines, "\n"))

	case modeDuePicker:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 36, 72))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		focusedStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		dateInput := m.duePickerDateInput
		dateInput.SetWidth(max(18, min(42, maxWidth-20)))
		timeInput := m.duePickerTimeInput
		timeInput.SetWidth(max(12, min(22, maxWidth-20)))
		includeTimeLine := "[ ] include time"
		if m.duePickerIncludeTime {
			includeTimeLine = "[x] include time"
		}
		if m.duePickerFocus == 0 {
			includeTimeLine = focusedStyle.Render(includeTimeLine)
		}
		lines := []string{titleStyle.Render("Due Date"), includeTimeLine}
		dateLine := "date: " + dateInput.View()
		if m.duePickerFocus == 1 {
			dateLine = focusedStyle.Render("date:") + " " + dateInput.View()
		}
		lines = append(lines, dateLine)
		if m.duePickerIncludeTime {
			timeLine := "time: " + timeInput.View()
			if m.duePickerFocus == 2 {
				timeLine = focusedStyle.Render("time:") + " " + timeInput.View()
			}
			lines = append(lines, timeLine)
		}
		lines = append(lines, "")
		options := m.duePickerOptions()
		start, end := windowBounds(len(options), m.duePicker, 10)
		for idx := start; idx < end; idx++ {
			option := options[idx]
			cursor := "  "
			if idx == m.duePicker {
				cursor = "> "
			}
			lines = append(lines, cursor+option.Label)
		}
		lines = append(lines, hintStyle.Render("tab focus • j/k navigate list • enter apply • space toggle include time • esc cancel"))
		return style.Render(strings.Join(lines, "\n"))

	case modeProjectPicker:
		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			pickerStyle = pickerStyle.Width(clamp(maxWidth, 24, 56))
		}

		title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Projects")
		lines := []string{title}
		archivedText := "archived projects: hidden"
		if m.showArchivedProjects {
			archivedText = "archived projects: shown"
		}
		lines = append(lines, helpStyle.Render(archivedText))
		if len(m.projects) == 0 {
			lines = append(lines, helpStyle.Render("(no projects yet)"))
		} else {
			for idx, p := range m.projects {
				cursor := "  "
				if idx == m.projectPickerIndex {
					cursor = "> "
				}
				label := projectDisplayLabel(p)
				lines = append(lines, cursor+label)
			}
		}
		lines = append(lines, helpStyle.Render("N new project"))
		if len(m.projects) == 0 {
			lines = append(lines, helpStyle.Render("enter/N create • A toggle archived • esc close"))
		} else {
			lines = append(lines, helpStyle.Render("j/k or wheel • enter choose • N new • A toggle archived • esc cancel"))
		}
		return pickerStyle.Render(strings.Join(lines, "\n"))

	case modeSearchResults:
		resultsStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			resultsStyle = resultsStyle.Width(clamp(maxWidth, 36, 96))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Search Results")}
		if len(m.searchMatches) == 0 {
			lines = append(lines, hintStyle.Render("(empty)"))
		} else {
			tasks := make([]domain.Task, 0, len(m.searchMatches))
			for _, match := range m.searchMatches {
				tasks = append(tasks, match.Task)
			}
			levelByTaskID := m.searchLevelByTaskID(tasks)
			for idx, match := range m.searchMatches {
				cursor := "  "
				if idx == m.searchResultIndex {
					cursor = "> "
				}
				level := strings.TrimSpace(strings.ToLower(levelByTaskID[match.Task.ID]))
				levelLabel := canonicalSearchLevelLabels[level]
				if levelLabel == "" {
					levelLabel = "-"
				}
				row := fmt.Sprintf("%s%s • %s • %s • %s", cursor, match.Project.Name, levelLabel, match.StateID, truncate(match.Task.Title, 40))
				lines = append(lines, row)
			}
		}
		lines = append(lines, hintStyle.Render("j/k navigate • enter open • esc close"))
		return resultsStyle.Render(strings.Join(lines, "\n"))

	case modeDependencyInspector:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 44, 118))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		in := m.dependencyInput
		in.SetWidth(max(20, maxWidth-22))
		ownerLabel := "(new task)"
		if task, ok := m.taskByID(strings.TrimSpace(m.dependencyOwnerTaskID)); ok {
			ownerLabel = task.Title
		}
		activeFieldLabel := m.dependencyActiveFieldLabel()
		lines := []string{
			titleStyle.Render("Dependencies & Blockers"),
			hintStyle.Render("task: " + truncate(ownerLabel, 56)),
			hintStyle.Render("active: " + activeFieldLabel),
			in.View(),
		}

		stateLabel := lipgloss.NewStyle().Foreground(muted)
		if m.dependencyFocus == 1 {
			stateLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
		}
		stateParts := make([]string, 0, len(canonicalSearchStatesOrdered))
		for idx, state := range canonicalSearchStatesOrdered {
			check := " "
			if m.isDependencyStateEnabled(state) {
				check = "x"
			}
			name := canonicalSearchStateLabels[state]
			if name == "" {
				name = state
			}
			item := fmt.Sprintf("[%s] %s", check, name)
			if idx == clamp(m.dependencyStateCursor, 0, len(canonicalSearchStatesOrdered)-1) && m.dependencyFocus == 1 {
				item = lipgloss.NewStyle().Bold(true).Foreground(accent).Render(item)
			}
			stateParts = append(stateParts, item)
		}
		lines = append(lines, stateLabel.Render("states:")+" "+strings.Join(stateParts, "   "))

		scopeLabel := lipgloss.NewStyle().Foreground(muted)
		if m.dependencyFocus == 2 {
			scopeLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
		}
		scopeText := "scope: current project"
		if m.dependencyCrossProject {
			scopeText = "scope: all projects"
		}
		lines = append(lines, scopeLabel.Render(scopeText))

		archivedLabel := lipgloss.NewStyle().Foreground(muted)
		if m.dependencyFocus == 3 {
			archivedLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
		}
		archivedText := "archived: hidden"
		if m.dependencyIncludeArchived {
			archivedText = "archived: included"
		}
		lines = append(lines, archivedLabel.Render(archivedText))

		countLine := fmt.Sprintf("depends_on: %d • blocked_by: %d", len(m.dependencyDependsOn), len(m.dependencyBlockedBy))
		if m.dependencyDirty {
			countLine += " • unsaved changes"
		}
		lines = append(lines, hintStyle.Render(countLine))
		if len(m.dependencyDependsOn) > 0 || len(m.dependencyBlockedBy) > 0 {
			lines = append(lines, hintStyle.Render("linked refs are pinned at top"))
		}
		lines = append(lines, "")

		if len(m.dependencyMatches) == 0 {
			lines = append(lines, hintStyle.Render("(no matching tasks)"))
		} else {
			const dependencyWindowSize = 8
			start, end := windowBounds(len(m.dependencyMatches), m.dependencyIndex, dependencyWindowSize)
			activeRowStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
			for idx := start; idx < end; idx++ {
				candidate := m.dependencyMatches[idx]
				taskID := strings.TrimSpace(candidate.Match.Task.ID)
				cursor := "  "
				if idx == m.dependencyIndex {
					cursor = "> "
				}
				depMark := " "
				if hasDependencyID(m.dependencyDependsOn, taskID) {
					depMark = "D"
				}
				blockMark := " "
				if hasDependencyID(m.dependencyBlockedBy, taskID) {
					blockMark = "B"
				}
				stateName := candidate.Match.StateID
				if label, ok := canonicalSearchStateLabels[strings.TrimSpace(strings.ToLower(candidate.Match.StateID))]; ok {
					stateName = label
				}
				row := fmt.Sprintf("%s[%s%s] %s • %s", cursor, depMark, blockMark, truncate(candidate.Match.Task.Title, 32), truncate(candidate.Path, 52))
				if idx == m.dependencyIndex && m.dependencyFocus == 4 {
					row = activeRowStyle.Render(row)
				}
				lines = append(lines, row)
				lines = append(lines, hintStyle.Render("    "+stateName+" • "+string(candidate.Match.Task.Kind)+" • id:"+taskID))
			}
			if len(m.dependencyMatches) > dependencyWindowSize {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(m.dependencyMatches))))
			}
		}

		if candidate, ok := m.selectedDependencyCandidate(); ok {
			details := candidate.Match.Task
			description := strings.TrimSpace(details.Description)
			if description == "" {
				description = "-"
			}
			stateText := lifecycleStateLabel(details.LifecycleState)
			if stateText == "-" {
				stateID := strings.TrimSpace(strings.ToLower(candidate.Match.StateID))
				if label, ok := canonicalSearchStateLabels[stateID]; ok {
					stateText = label
				} else if stateID != "" {
					stateText = stateID
				}
			}
			lines = append(lines, "")
			lines = append(lines, hintStyle.Render("details"))
			lines = append(lines, hintStyle.Render("id: "+details.ID))
			lines = append(lines, hintStyle.Render("path: "+truncate(candidate.Path, 86)))
			lines = append(lines, hintStyle.Render("state: "+stateText+" • kind: "+string(details.Kind)))
			lines = append(lines, hintStyle.Render("description: "+truncate(description, 86)))
		}

		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("tab focus • j/k list • d toggle depends_on • b toggle blocked_by • space toggle active field value"))
		lines = append(lines, hintStyle.Render("x switch active field • enter jump to task • a apply changes • esc cancel"))
		return style.Render(strings.Join(lines, "\n"))

	case modeCommandPalette:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 36, 96))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		in := m.commandInput
		in.SetWidth(max(18, maxWidth-20))
		lines := []string{
			titleStyle.Render("Command Palette"),
			in.View(),
		}
		if len(m.commandMatches) == 0 {
			lines = append(lines, hintStyle.Render("(no matching commands)"))
		} else {
			const commandWindowSize = 9
			start, end := windowBounds(len(m.commandMatches), m.commandIndex, commandWindowSize)
			for idx := start; idx < end; idx++ {
				item := m.commandMatches[idx]
				prefix := "  "
				if idx == m.commandIndex {
					prefix = "› "
				}
				alias := ""
				if len(item.Aliases) > 0 {
					alias = " (" + strings.Join(item.Aliases, ", ") + ")"
				}
				lines = append(lines, fmt.Sprintf("%s%s%s — %s", prefix, item.Command, alias, item.Description))
			}
			if len(m.commandMatches) > commandWindowSize {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(m.commandMatches))))
			}
		}
		lines = append(lines, hintStyle.Render("enter run • tab autocomplete • j/k move • esc cancel"))
		if m.searchApplied {
			lines = append(lines, hintStyle.Render("search hints: clear-query • reset-filters • search-all • search-project"))
		}
		return style.Render(strings.Join(lines, "\n"))

	case modePathsRoots:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 42, 100))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		projectLabel := "(none)"
		if project, ok := m.currentProject(); ok {
			projectLabel = projectDisplayName(project)
			if slug := strings.TrimSpace(strings.ToLower(project.Slug)); slug != "" {
				projectLabel += " (" + slug + ")"
			}
		}
		in := m.pathsRootInput
		in.SetWidth(max(20, maxWidth-24))
		lines := []string{
			titleStyle.Render("Paths / Roots"),
			hintStyle.Render("project: " + projectLabel),
			in.View(),
			hintStyle.Render("enter save • esc cancel • r browse dirs • empty value clears mapping"),
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeConfirmAction:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 36, 88))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		targetTitle := strings.TrimSpace(m.pendingConfirm.Task.Title)
		if len(m.pendingConfirm.TaskIDs) > 1 {
			targetTitle = fmt.Sprintf("%d selected tasks", len(m.pendingConfirm.TaskIDs))
		}
		if strings.TrimSpace(m.pendingConfirm.Project.ID) != "" {
			targetTitle = strings.TrimSpace(m.pendingConfirm.Project.Name)
			if targetTitle == "" {
				targetTitle = strings.TrimSpace(m.pendingConfirm.Project.ID)
			}
			targetTitle = "project " + targetTitle
		}
		if targetTitle == "" {
			targetTitle = "(unknown target)"
		}
		confirmStyle := lipgloss.NewStyle().Foreground(muted)
		cancelStyle := lipgloss.NewStyle().Foreground(muted)
		if m.confirmChoice == 0 {
			confirmStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
		} else {
			cancelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
		}
		lines := []string{
			titleStyle.Render("Confirm Action"),
			fmt.Sprintf("%s: %s", m.pendingConfirm.Label, targetTitle),
			confirmStyle.Render("[confirm]") + "  " + cancelStyle.Render("[cancel]"),
			hintStyle.Render("enter apply • esc cancel • h/l switch • y confirm • n cancel"),
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeWarning:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 36, 96))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		title := strings.TrimSpace(m.warningTitle)
		if title == "" {
			title = "Warning"
		}
		body := strings.TrimSpace(m.warningBody)
		if body == "" {
			body = "This action is not allowed in the current context."
		}
		lines := []string{
			titleStyle.Render(title),
			body,
			hintStyle.Render("enter close • esc close"),
		}
		return style.Render(strings.Join(lines, "\n"))

	case modeQuickActions:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 32, 78))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Quick Actions")}
		actions := m.quickActions()
		if len(actions) == 0 {
			lines = append(lines, hintStyle.Render("(no actions available)"))
		} else {
			const quickActionWindowSize = 11
			start, end := windowBounds(len(actions), m.quickActionIndex, quickActionWindowSize)
			enabledActiveStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
			disabledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
			disabledActiveStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("243"))
			for idx := start; idx < end; idx++ {
				action := actions[idx]
				cursor := "  "
				if idx == m.quickActionIndex {
					cursor = "> "
				}
				label := action.Label
				if !action.Enabled && strings.TrimSpace(action.DisabledReason) != "" {
					label += " (" + action.DisabledReason + ")"
				}
				switch {
				case action.Enabled && idx == m.quickActionIndex:
					label = enabledActiveStyle.Render(label)
				case !action.Enabled && idx == m.quickActionIndex:
					label = disabledActiveStyle.Render(label)
				case !action.Enabled:
					label = disabledStyle.Render(label)
				}
				lines = append(lines, cursor+label)
			}
			if len(actions) > quickActionWindowSize {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(actions))))
			}
		}
		lines = append(lines, hintStyle.Render("j/k navigate • enter run • esc close"))
		return style.Render(strings.Join(lines, "\n"))

	case modeAddTask, modeSearch, modeRenameTask, modeEditTask, modeAddProject, modeEditProject, modeLabelsConfig, modeHighlightColor:
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			boxStyle = boxStyle.Width(clamp(maxWidth, 24, 96))
		}

		title := "Input"
		hint := "enter save • esc cancel • tab next field"
		switch m.mode {
		case modeAddTask:
			title = "New Task"
			hint = "enter save • esc cancel • tab next field • enter/o deps picker • d due picker • ctrl+r attach resource • ctrl+g label suggestion"
		case modeSearch:
			title = "Search"
			hint = "tab focus • space/enter toggle • ctrl+u clear query • ctrl+r reset filters"
		case modeRenameTask:
			title = "Rename Task"
		case modeEditTask:
			title = "Edit Task"
			hint = "enter save • esc cancel • tab next field • enter/o deps picker • d due picker • ctrl+r attach resource • ctrl+g label suggestion"
		case modeAddProject:
			title = "New Project"
		case modeEditProject:
			title = "Edit Project"
		case modeLabelsConfig:
			title = "Labels Config"
		case modeHighlightColor:
			title = "Highlight Color"
			hint = "enter save • esc cancel • empty resets to default"
		}

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render(title)}

		switch m.mode {
		case modeSearch:
			queryInput := m.searchInput
			queryInput.SetWidth(max(18, maxWidth-20))
			scope := "current project"
			if m.searchCrossProject {
				scope = "all projects"
			}
			labelStyle := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 0 {
				labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, labelStyle.Render("query:")+" "+queryInput.View())

			stateLabel := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 1 {
				stateLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			stateParts := make([]string, 0, len(canonicalSearchStatesOrdered))
			for idx, state := range canonicalSearchStatesOrdered {
				check := " "
				if m.isSearchStateEnabled(state) {
					check = "x"
				}
				name := canonicalSearchStateLabels[state]
				if name == "" {
					name = state
				}
				item := fmt.Sprintf("[%s] %s", check, name)
				if idx == clamp(m.searchStateCursor, 0, len(canonicalSearchStatesOrdered)-1) && m.searchFocus == 1 {
					item = lipgloss.NewStyle().Bold(true).Foreground(accent).Render(item)
				}
				stateParts = append(stateParts, item)
			}
			lines = append(lines, stateLabel.Render("states:")+" "+strings.Join(stateParts, "   "))

			levelLabel := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 2 {
				levelLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			levelParts := make([]string, 0, len(canonicalSearchLevelsOrdered))
			for idx, level := range canonicalSearchLevelsOrdered {
				check := " "
				if m.isSearchLevelEnabled(level) {
					check = "x"
				}
				name := canonicalSearchLevelLabels[level]
				if name == "" {
					name = level
				}
				item := fmt.Sprintf("[%s] %s", check, name)
				if idx == clamp(m.searchLevelCursor, 0, len(canonicalSearchLevelsOrdered)-1) && m.searchFocus == 2 {
					item = lipgloss.NewStyle().Bold(true).Foreground(accent).Render(item)
				}
				levelParts = append(levelParts, item)
			}
			lines = append(lines, levelLabel.Render("levels:")+" "+strings.Join(levelParts, "   "))

			scopeLabel := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 3 {
				scopeLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, scopeLabel.Render("scope: "+scope))

			archivedLabel := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 4 {
				archivedLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			if m.searchIncludeArchived {
				lines = append(lines, archivedLabel.Render("archived: included"))
			} else {
				lines = append(lines, archivedLabel.Render("archived: hidden"))
			}
			applyLabel := hintStyle
			if m.searchFocus == 5 {
				applyLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, applyLabel.Render("[ apply search ]"))
		case modeAddTask, modeEditTask:
			fieldWidth := max(18, maxWidth-28)
			for i, in := range m.formInputs {
				label := fmt.Sprintf("%d.", i+1)
				if i < len(taskFormFields) {
					label = taskFormFields[i]
				}
				labelStyle := lipgloss.NewStyle().Foreground(muted)
				if i == m.formFocus {
					labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
				}
				if i == taskFieldPriority {
					lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+m.renderPriorityPicker(accent, muted))
					continue
				}
				in.SetWidth(fieldWidth)
				lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+in.View())
			}
			if m.formFocus == taskFieldDue {
				lines = append(lines, hintStyle.Render("d opens due-date picker (includes local-time presets)"))
				lines = append(lines, hintStyle.Render("type: YYYY-MM-DD | YYYY-MM-DD HH:MM | YYYY-MM-DDTHH:MM | RFC3339 | -"))
				lines = append(lines, hintStyle.Render("time without timezone is interpreted in local time"))
			}
			if m.formFocus == taskFieldLabels {
				lines = append(lines, hintStyle.Render("enter or ctrl+l opens label picker • ctrl+g accept autocomplete"))
			}
			if m.formFocus == taskFieldDependsOn || m.formFocus == taskFieldBlockedBy {
				lines = append(lines, hintStyle.Render("enter or o opens dependency picker • csv task IDs still supported"))
			}
			if len(m.taskFormResourceRefs) > 0 {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("staged resources: %d", len(m.taskFormResourceRefs))))
			}
			if suggestions := m.labelSuggestions(5); len(suggestions) > 0 {
				lines = append(lines, hintStyle.Render("suggested labels: "+strings.Join(suggestions, ", ")))
			}
			inherited := m.taskFormLabelSources()
			lines = append(lines, hintStyle.Render("inherited labels"))
			lines = append(lines, hintStyle.Render(formatLabelSource("global", inherited.Global)))
			lines = append(lines, hintStyle.Render(formatLabelSource("project", inherited.Project)))
			lines = append(lines, hintStyle.Render(formatLabelSource("branch", inherited.Branch)))
			lines = append(lines, hintStyle.Render(formatLabelSource("phase", inherited.Phase)))
			if warning := dueWarning(m.formInputs[taskFieldDue].Value(), time.Now().UTC()); warning != "" {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(warning))
			}
			if m.mode == modeEditTask {
				lines = append(lines, hintStyle.Render("blank values keep current task value"))
			}
		case modeAddProject, modeEditProject:
			fieldWidth := max(18, maxWidth-28)
			for i, in := range m.projectFormInputs {
				label := fmt.Sprintf("%d.", i+1)
				if i < len(projectFormFields) {
					label = projectFormFields[i]
				}
				labelStyle := lipgloss.NewStyle().Foreground(muted)
				if i == m.projectFormFocus {
					labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
				}
				in.SetWidth(fieldWidth)
				lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+in.View())
			}
			if m.projectFormFocus == projectFieldIcon {
				lines = append(lines, hintStyle.Render("icon shows in project header/tabs/picker and supports emoji"))
			}
			if m.projectFormFocus == projectFieldRootPath {
				lines = append(lines, hintStyle.Render("r browse and select a directory"))
			}
		case modeLabelsConfig:
			fieldWidth := max(18, maxWidth-28)
			labelFields := []string{"global", "project", "branch", "phase"}
			for i, in := range m.labelsConfigInputs {
				label := fmt.Sprintf("%d.", i+1)
				if i < len(labelFields) {
					label = labelFields[i]
				}
				labelStyle := lipgloss.NewStyle().Foreground(muted)
				if i == m.labelsConfigFocus {
					labelStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
				}
				in.SetWidth(fieldWidth)
				lines = append(lines, labelStyle.Render(fmt.Sprintf("%-12s", label+":"))+" "+in.View())
			}
			lines = append(lines, hintStyle.Render("global/project saved to config • branch/phase saved to current hierarchy nodes"))
		case modeHighlightColor:
			in := m.highlightColorInput
			in.SetWidth(max(18, maxWidth-24))
			lines = append(lines, hintStyle.Render("focused-row color (ansi index or #RRGGBB)"))
			lines = append(lines, "value: "+in.View())
			lines = append(lines, hintStyle.Render("example: 212 (fuchsia)"))
		default:
			lines = append(lines, m.input)
		}

		lines = append(lines, hintStyle.Render(hint))
		return boxStyle.Render(strings.Join(lines, "\n"))
	default:
		return ""
	}
}

// renderPriorityPicker renders output for the current model state.
func (m Model) renderPriorityPicker(accent, muted color.Color) string {
	parts := make([]string, 0, len(priorityOptions))
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	baseStyle := lipgloss.NewStyle().Foreground(muted)
	for i, p := range priorityOptions {
		label := string(p)
		if i == m.priorityIdx {
			label = activeStyle.Render("[" + label + "]")
		} else {
			label = baseStyle.Render(label)
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "  ")
}

// formatTaskEditInput formats values for display or serialization.
func formatTaskEditInput(task domain.Task) string {
	due := "-"
	if task.DueAt != nil {
		due = formatDueValue(task.DueAt)
	}
	labels := "-"
	if len(task.Labels) > 0 {
		labels = strings.Join(task.Labels, ",")
	}
	return strings.Join([]string{
		task.Title,
		task.Description,
		string(task.Priority),
		due,
		labels,
	}, " | ")
}

// parseTaskEditInput parses input into a normalized form.
func parseTaskEditInput(raw string, current domain.Task) (app.UpdateTaskInput, error) {
	parts := strings.Split(raw, "|")
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	if len(parts) > 5 {
		return app.UpdateTaskInput{}, fmt.Errorf("expected 5 fields")
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	title := parts[0]
	if title == "" {
		title = current.Title
	}

	description := parts[1]
	if description == "" {
		description = current.Description
	}

	priority := domain.Priority(parts[2])
	if priority == "" {
		priority = current.Priority
	}
	switch priority {
	case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
	default:
		return app.UpdateTaskInput{}, fmt.Errorf("priority must be low|medium|high")
	}

	dueAt, err := parseDueInput(parts[3], current.DueAt)
	if err != nil {
		return app.UpdateTaskInput{}, err
	}

	labels := current.Labels
	if parts[4] == "-" {
		labels = nil
	} else if parts[4] != "" {
		rawLabels := strings.Split(parts[4], ",")
		parsedLabels := make([]string, 0, len(rawLabels))
		for _, label := range rawLabels {
			label = strings.TrimSpace(label)
			if label == "" {
				continue
			}
			parsedLabels = append(parsedLabels, label)
		}
		labels = parsedLabels
	}

	return app.UpdateTaskInput{
		Title:       title,
		Description: description,
		Priority:    priority,
		DueAt:       dueAt,
		Labels:      labels,
	}, nil
}

// modeLabel handles mode label.
func (m Model) modeLabel() string {
	switch m.mode {
	case modeAddTask:
		return "add-task"
	case modeSearch:
		return "search"
	case modeRenameTask:
		return "rename"
	case modeEditTask:
		return "edit-task"
	case modeDuePicker:
		return "due-picker"
	case modeProjectPicker:
		return "project-picker"
	case modeTaskInfo:
		return "task-info"
	case modeAddProject:
		return "add-project"
	case modeEditProject:
		return "edit-project"
	case modeSearchResults:
		return "search-results"
	case modeCommandPalette:
		return "command"
	case modeQuickActions:
		return "actions"
	case modeActivityLog:
		return "activity"
	case modeActivityEventInfo:
		return "activity-event"
	case modeConfirmAction:
		return "confirm"
	case modeWarning:
		return "warning"
	case modeResourcePicker:
		return "resources"
	case modeLabelPicker:
		return "labels"
	case modePathsRoots:
		return "paths/roots"
	case modeLabelsConfig:
		return "labels-config"
	case modeHighlightColor:
		return "highlight-color"
	case modeBootstrapSettings:
		return "bootstrap"
	case modeDependencyInspector:
		return "deps"
	case modeThread:
		return "thread"
	default:
		return "normal"
	}
}

// modePrompt handles mode prompt.
func (m Model) modePrompt() string {
	switch m.mode {
	case modeAddTask:
		return "new task title: " + m.input + " (enter save, esc cancel)"
	case modeSearch:
		return "search query: " + m.input + " (enter apply, esc cancel)"
	case modeRenameTask:
		return "rename task: " + m.input + " (enter save, esc cancel)"
	case modeEditTask:
		return "edit task: " + m.input + " (title | description | priority(low|medium|high) | due(YYYY-MM-DD | YYYY-MM-DD HH:MM | YYYY-MM-DDTHH:MM | RFC3339 | -) | labels(csv))"
	case modeDuePicker:
		return "due picker: tab focus controls, type date/time to filter, j/k navigate list, enter apply, esc cancel"
	case modeProjectPicker:
		return "project picker: j/k select, enter choose, N new project, A archived toggle, esc cancel"
	case modeTaskInfo:
		return "task info: e edit, s subtask, c thread, [ / ] move, space toggle subtask complete, b deps inspector, r attach, esc back/close"
	case modeAddProject:
		return "new project: enter save, esc cancel"
	case modeEditProject:
		return "edit project: enter save, esc cancel"
	case modeSearchResults:
		return "search results: j/k select, enter jump, esc close"
	case modeCommandPalette:
		return "command palette: enter run, esc cancel"
	case modeQuickActions:
		return "quick actions: j/k select, enter run, esc close"
	case modeActivityLog:
		return "activity log: esc close"
	case modeActivityEventInfo:
		return "activity event: enter/g go to node, esc back"
	case modeConfirmAction:
		return "confirm action: enter confirm, esc cancel"
	case modeWarning:
		return "warning: enter close, esc close"
	case modeResourcePicker:
		return "resource picker: type fuzzy filter, arrows navigate, enter select, ctrl+a choose/attach current, esc cancel"
	case modeLabelPicker:
		return "label picker: type fuzzy filter, j/k select, enter add label, ctrl+u clear, esc cancel"
	case modePathsRoots:
		return "paths/roots: enter save, r browse dirs, esc cancel"
	case modeLabelsConfig:
		return "labels config: enter save, esc cancel"
	case modeHighlightColor:
		return "highlight color: enter save, esc cancel"
	case modeBootstrapSettings:
		return "bootstrap settings: tab focus, r browse/add default path, d clear path, enter save"
	case modeDependencyInspector:
		return "deps inspector: tab focus, d/b toggle, x switch active, enter jump, a apply, esc cancel"
	case modeThread:
		return "thread: enter post comment, pgup/pgdown scroll, ctrl+r reload, esc back"
	default:
		return ""
	}
}

// normalizeBoardGroupBy canonicalizes board grouping values.
func normalizeBoardGroupBy(groupBy string) string {
	switch strings.ToLower(strings.TrimSpace(groupBy)) {
	case "priority":
		return "priority"
	case "state":
		return "state"
	default:
		return "none"
	}
}

// formatActivityTimestamp formats activity timestamps for compact modal rendering.
func formatActivityTimestamp(at time.Time) string {
	if at.IsZero() {
		return "--:--:--"
	}
	local := at.Local()
	now := time.Now().In(local.Location())
	if local.Year() != now.Year() || local.YearDay() != now.YearDay() {
		return local.Format("01-02 15:04")
	}
	return local.Format("15:04:05")
}

// formatActivityTimestampLong formats activity timestamps for detailed event rendering.
func formatActivityTimestampLong(at time.Time) string {
	if at.IsZero() {
		return "-"
	}
	return at.Local().Format(time.RFC3339)
}

// activityEventTargetDetails resolves user-facing node/path labels for one activity event.
func (m Model) activityEventTargetDetails(entry activityEntry) (string, string) {
	if task, ok := m.taskByID(strings.TrimSpace(entry.WorkItemID)); ok {
		node := fallbackText(strings.TrimSpace(task.Title), "-")
		return node, m.activityTaskPath(task)
	}
	node := fallbackText(strings.TrimSpace(entry.Target), "-")
	if project, ok := m.currentProject(); ok {
		projectLabel := projectDisplayName(project)
		if projectLabel == "" {
			projectLabel = "(project)"
		}
		if node == "-" {
			return node, projectLabel
		}
		return node, projectLabel + " -> " + node
	}
	return node, node
}

// activityTaskPath builds a project-rooted path label for one task.
func (m Model) activityTaskPath(task domain.Task) string {
	chain := []string{fallbackText(strings.TrimSpace(task.Title), "(untitled)")}
	visited := map[string]struct{}{task.ID: {}}
	parentID := strings.TrimSpace(task.ParentID)
	for parentID != "" {
		if _, seen := visited[parentID]; seen {
			break
		}
		parent, ok := m.taskByID(parentID)
		if !ok {
			break
		}
		visited[parentID] = struct{}{}
		chain = append(chain, fallbackText(strings.TrimSpace(parent.Title), "(untitled)"))
		parentID = strings.TrimSpace(parent.ParentID)
	}
	slices.Reverse(chain)
	if project, ok := m.currentProject(); ok {
		projectLabel := projectDisplayName(project)
		if projectLabel != "" {
			chain = append([]string{projectLabel}, chain...)
		}
	}
	return strings.Join(chain, " -> ")
}

// activityColumnLabel resolves one column id to a display name with id fallback.
func (m Model) activityColumnLabel(columnID string) string {
	columnID = strings.TrimSpace(columnID)
	if columnID == "" {
		return "-"
	}
	for _, column := range m.columns {
		if column.ID == columnID {
			label := strings.TrimSpace(column.Name)
			if label != "" {
				return label
			}
			return columnID
		}
	}
	return columnID
}

// formatActivityMetadata renders event metadata with human-readable values and concise fallbacks.
func (m Model) formatActivityMetadata(entry activityEntry) []string {
	if len(entry.Metadata) == 0 {
		return nil
	}
	lines := make([]string, 0, len(entry.Metadata))
	consumed := make(map[string]struct{}, len(entry.Metadata))
	consume := func(keys ...string) {
		for _, key := range keys {
			consumed[key] = struct{}{}
		}
	}

	columnID := strings.TrimSpace(entry.Metadata["column_id"])
	fromColumnID := strings.TrimSpace(entry.Metadata["from_column_id"])
	toColumnID := strings.TrimSpace(entry.Metadata["to_column_id"])
	if fromColumnID != "" || toColumnID != "" {
		lines = append(lines, "column: "+m.activityColumnLabel(fromColumnID)+" -> "+m.activityColumnLabel(toColumnID))
		consume("from_column_id", "to_column_id", "from_position", "to_position")
	} else if columnID != "" {
		lines = append(lines, "column: "+m.activityColumnLabel(columnID))
		consume("column_id", "position")
	}

	if changed := strings.TrimSpace(entry.Metadata["changed_fields"]); changed != "" {
		parts := strings.Split(changed, ",")
		for idx := range parts {
			parts[idx] = strings.TrimSpace(parts[idx])
		}
		lines = append(lines, "changed fields: "+strings.Join(parts, ", "))
		consume("changed_fields")
	}

	fromState := strings.TrimSpace(entry.Metadata["from_state"])
	toState := strings.TrimSpace(entry.Metadata["to_state"])
	if fromState != "" || toState != "" {
		lines = append(lines, "state: "+lifecycleStateLabel(domain.LifecycleState(fromState))+" -> "+lifecycleStateLabel(domain.LifecycleState(toState)))
		consume("from_state", "to_state")
	}

	consume("title")

	for _, key := range sortedStringKeys(entry.Metadata) {
		if _, skip := consumed[key]; skip {
			continue
		}
		if strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "_position") {
			continue
		}
		value := strings.TrimSpace(entry.Metadata[key])
		if value == "" {
			continue
		}
		label := strings.ReplaceAll(strings.TrimSpace(key), "_", " ")
		lines = append(lines, label+": "+value)
	}
	return lines
}

// fallbackText returns fallback when value is blank.
func fallbackText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// sortedStringKeys returns deterministic ascending keys for map rendering.
func sortedStringKeys(in map[string]string) []string {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		if strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// columnWidth returns column width.
func (m Model) columnWidth() int {
	return m.columnWidthFor(m.boardWidthFor(m.width))
}

// renderedBoardColumnWidth returns the terminal render width for one board column at the given style width.
func renderedBoardColumnWidth(width int) int {
	return max(0, width)
}

// renderedNoticesPanelWidth returns the terminal render width for one notices panel at the given style width.
func renderedNoticesPanelWidth(width int) int {
	return max(0, width)
}

// columnWidthFor returns column width for.
func (m Model) columnWidthFor(boardWidth int) int {
	if len(m.columns) == 0 {
		return minimumColumnWidth
	}
	interColumnGaps := max(0, len(m.columns)-1) * boardColumnGapWidth
	usable := boardWidth - interColumnGaps
	if usable <= 0 {
		return minimumColumnWidth
	}
	w := usable / len(m.columns)
	if w < minimumColumnWidth {
		return minimumColumnWidth
	}
	return w
}

// noticesPanelWidth returns the right-panel width when the viewport can support it.
func (m Model) noticesPanelWidth(totalWidth int) int {
	if totalWidth <= 0 || len(m.columns) == 0 {
		return 0
	}
	// Preserve minimum readable column widths and the Done->Notices/right-gutter budget.
	minBoardWidth := len(m.columns)*renderedBoardColumnWidth(minimumColumnWidth) + max(0, len(m.columns)-1)*boardColumnGapWidth
	availableForPanel := totalWidth - minBoardWidth - noticesPanelGapWidth
	if availableForPanel < renderedNoticesPanelWidth(minimumNoticesPanelWidth) {
		return 0
	}
	if availableForPanel > maximumNoticesPanelWidth {
		return maximumNoticesPanelWidth
	}
	return availableForPanel
}

// boardWidthFor returns the board-body width after reserving optional side panel space.
func (m Model) boardWidthFor(totalWidth int) int {
	panelWidth := m.noticesPanelWidth(totalWidth)
	if panelWidth <= 0 {
		return totalWidth
	}
	reservedWidth := renderedNoticesPanelWidth(panelWidth) + noticesPanelGapWidth
	return max(minimumColumnWidth, totalWidth-reservedWidth)
}

// columnHeight returns column height.
func (m Model) columnHeight() int {
	// Header rows reserve boxed mark + divider + path (+ 1-based board origin offset).
	headerLines := m.boardTop()
	footerLines := 4
	h := m.height - headerLines - footerLines
	if h < 14 {
		return 14
	}
	return h
}

// headerMarkStyle returns the boxed brand style used at the top of board view.
func headerMarkStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Bold(true)
}

// boardHeaderLinesBeforeBoard returns the count of rows rendered before the board begins.
func boardHeaderLinesBeforeBoard() int {
	// Boxed mark + divider + spacer + path line.
	return lipgloss.Height(headerMarkStyle().Render(headerMarkText)) + 3
}

// boardTop handles board top.
func (m Model) boardTop() int {
	// Mouse coordinates from tea are 1-based; board starts after all header rows.
	return boardHeaderLinesBeforeBoard() + 1
}

// clamp clamps the requested operation.
func clamp(v, minV, maxV int) int {
	if maxV < minV {
		return minV
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// max returns the larger of the provided values.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of the provided values.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fitLines fits lines.
func fitLines(content string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	switch {
	case len(lines) > maxLines:
		if maxLines == 1 {
			lines = []string{"…"}
		} else {
			lines = append(lines[:maxLines-1], "…")
		}
	case len(lines) < maxLines:
		padding := make([]string, maxLines-len(lines))
		lines = append(lines, padding...)
	}
	return strings.Join(lines, "\n")
}

// overlayOnContent overlays on content.
func overlayOnContent(base, overlay string, width, height int) string {
	if width <= 0 || height <= 0 {
		if strings.TrimSpace(overlay) == "" {
			return base
		}
		return overlay + "\n\n" + base
	}

	base = fitLines(base, height)
	canvas := lipgloss.NewCanvas(width, height)
	baseLayer := lipgloss.NewLayer(base).X(0).Y(0).Z(0)
	centeredOverlay := lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
	overlayLayer := lipgloss.NewLayer(centeredOverlay).X(0).Y(0).Z(10)

	canvas.Compose(baseLayer)
	canvas.Compose(overlayLayer)
	return canvas.Render()
}

// truncate truncates the requested operation.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	if max <= 1 {
		return string(rs[:max])
	}
	return string(rs[:max-1]) + "…"
}

// summarizeLabels summarizes labels.
func summarizeLabels(labels []string, maxLabels int) string {
	if len(labels) == 0 {
		return ""
	}
	if maxLabels <= 0 {
		maxLabels = 1
	}
	visible := labels
	extra := 0
	if len(labels) > maxLabels {
		visible = labels[:maxLabels]
		extra = len(labels) - maxLabels
	}
	joined := "#" + strings.Join(visible, ",#")
	if extra > 0 {
		joined += fmt.Sprintf("+%d", extra)
	}
	return joined
}
