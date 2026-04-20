package tui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/tui/gitdiff"
)

// Service represents service data used by this package.
type Service interface {
	ListProjects(context.Context, bool) ([]domain.Project, error)
	ListKindDefinitions(context.Context, bool) ([]domain.KindDefinition, error)
	ListColumns(context.Context, string, bool) ([]domain.Column, error)
	ListActionItems(context.Context, string, bool) ([]domain.ActionItem, error)
	CreateComment(context.Context, app.CreateCommentInput) (domain.Comment, error)
	ListCommentsByTarget(context.Context, app.ListCommentsByTargetInput) ([]domain.Comment, error)
	ListProjectChangeEvents(context.Context, string, int) ([]domain.ChangeEvent, error)
	ListAttentionItems(context.Context, app.ListAttentionItemsInput) ([]domain.AttentionItem, error)
	ResolveAttentionItem(context.Context, app.ResolveAttentionItemInput) (domain.AttentionItem, error)
	ListTemplateLibraries(context.Context, app.ListTemplateLibrariesInput) ([]domain.TemplateLibrary, error)
	ListAuthRequests(context.Context, domain.AuthRequestListFilter) ([]domain.AuthRequest, error)
	ListAuthSessions(context.Context, app.AuthSessionFilter) ([]app.AuthSession, error)
	ListCapabilityLeases(context.Context, app.ListCapabilityLeasesInput) ([]domain.CapabilityLease, error)
	ListHandoffs(context.Context, app.ListHandoffsInput) ([]domain.Handoff, error)
	GetProjectTemplateBinding(context.Context, string) (domain.ProjectTemplateBinding, error)
	GetBuiltinTemplateLibraryStatus(context.Context, string) (domain.BuiltinTemplateLibraryStatus, error)
	GetProjectTemplateReapplyPreview(context.Context, string) (domain.ProjectTemplateReapplyPreview, error)
	GetNodeContractSnapshot(context.Context, string) (domain.NodeContractSnapshot, error)
	GetAuthRequest(context.Context, string) (domain.AuthRequest, error)
	ApproveAuthRequest(context.Context, app.ApproveAuthRequestInput) (app.ApprovedAuthRequestResult, error)
	ApproveProjectTemplateMigrations(context.Context, app.ApproveProjectTemplateMigrationsInput) (domain.ProjectTemplateMigrationApprovalResult, error)
	DenyAuthRequest(context.Context, app.DenyAuthRequestInput) (domain.AuthRequest, error)
	RevokeAuthSession(context.Context, string, string) (app.AuthSession, error)
	RevokeCapabilityLease(context.Context, app.RevokeCapabilityLeaseInput) (domain.CapabilityLease, error)
	UpdateHandoff(context.Context, app.UpdateHandoffInput) (domain.Handoff, error)
	GetProjectDependencyRollup(context.Context, string) (domain.DependencyRollup, error)
	ListEmbeddingStates(context.Context, app.EmbeddingListFilter) ([]app.EmbeddingRecord, error)
	SummarizeEmbeddingStates(context.Context, app.EmbeddingListFilter) (app.EmbeddingSummary, error)
	EmbeddingsOperational() bool
	ReindexEmbeddings(context.Context, app.ReindexEmbeddingsInput) (app.ReindexEmbeddingsResult, error)
	SearchActionItems(context.Context, app.SearchActionItemsFilter) (app.SearchActionItemMatchesResult, error)
	CreateProjectWithMetadata(context.Context, app.CreateProjectInput) (domain.Project, error)
	UpdateProject(context.Context, app.UpdateProjectInput) (domain.Project, error)
	BindProjectTemplateLibrary(context.Context, app.BindProjectTemplateLibraryInput) (domain.ProjectTemplateBinding, error)
	UnbindProjectTemplateLibrary(context.Context, app.UnbindProjectTemplateLibraryInput) error
	ArchiveProject(context.Context, string) (domain.Project, error)
	RestoreProject(context.Context, string) (domain.Project, error)
	DeleteProject(context.Context, string) error
	CreateActionItem(context.Context, app.CreateActionItemInput) (domain.ActionItem, error)
	UpdateActionItem(context.Context, app.UpdateActionItemInput) (domain.ActionItem, error)
	MoveActionItem(context.Context, string, string, int) (domain.ActionItem, error)
	DeleteActionItem(context.Context, string, app.DeleteMode) error
	RestoreActionItem(context.Context, string) (domain.ActionItem, error)
	RenameActionItem(context.Context, string, string) (domain.ActionItem, error)
}

type staticHelpKeyMap struct {
	short []key.Binding
	full  [][]key.Binding
}

func (k staticHelpKeyMap) ShortHelp() []key.Binding {
	return k.short
}

func (k staticHelpKeyMap) FullHelp() [][]key.Binding {
	return k.full
}

func helpBinding(helpKey, desc string, keys ...string) key.Binding {
	if len(keys) == 0 {
		keys = []string{helpKey}
	}
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(helpKey, desc))
}

// inputMode represents a selectable mode.
type inputMode int

// modeNone and related constants define package defaults.
const (
	modeNone inputMode = iota
	modeAddActionItem
	modeSearch
	modeRenameActionItem
	modeEditActionItem
	modeDuePicker
	modeProjectPicker
	modeActionItemInfo
	modeAddProject
	modeEditProject
	modeSearchResults
	modeEmbeddingsStatus
	modeCommandPalette
	modeQuickActions
	modeConfirmAction
	modeAuthReview
	modeAuthScopePicker
	modeAuthInventory
	modeAuthSessionRevoke
	modeCoordinationDetail
	modeWarning
	modeActivityLog
	modeActivityEventInfo
	modeResourcePicker
	modeLabelPicker
	modeProjectKindPicker
	modeTemplateLibraryPicker
	modePathsRoots
	modeLabelsConfig
	modeHighlightColor
	modeBootstrapSettings
	modeDependencyInspector
	modeTemplateMigrationReview
	modeDescriptionEditor
	modeThread
	modeDiff
	modeFileViewer
)

// descriptionEditorTarget identifies which form field receives markdown-description editor output.
type descriptionEditorTarget int

const (
	descriptionEditorTargetActionItem descriptionEditorTarget = iota
	descriptionEditorTargetProject
	descriptionEditorTargetThread
	descriptionEditorTargetActionItemFormField
)

// descriptionEditorViewMode identifies active layout within the full-screen description editor.
type descriptionEditorViewMode int

const (
	descriptionEditorViewModeEdit descriptionEditorViewMode = iota
	descriptionEditorViewModePreview
)

// actionItemFormFields stores actionItem-form field keys in display/update order.
var actionItemFormFields = []string{
	"title",
	"description",
	"priority",
	"due",
	"labels",
	"depends_on",
	"blocked_by",
	"blocked_reason",
	"objective",
	"acceptance_criteria",
	"validation_plan",
	"risk_notes",
}

// terminalProbeArtifactWithPrefixPattern matches leaked OSC 10/11 rgb probe artifacts with dangling rgb-triplet prefixes.
var terminalProbeArtifactWithPrefixPattern = regexp.MustCompile(`(?i)(?:/[0-9a-f]{2,4}){2,4}\]?1[01];rgb:[0-9a-f/]{6,64}`)

// terminalProbeArtifactPattern matches leaked OSC 10/11 rgb probe artifacts that can be echoed into focused inputs.
var terminalProbeArtifactPattern = regexp.MustCompile(`(?i)\]?1[01];rgb:[0-9a-f/]{6,64}`)

// terminalProbeEscapeSequencePattern matches complete OSC escape sequences (ESC ] ... BEL / ST).
var terminalProbeEscapeSequencePattern = regexp.MustCompile(`\x1b\][^\x1b\x07]*(?:\x07|\x1b\\)`)

// actionItem-form field indexes used throughout keyboard/update logic.
const (
	actionItemFieldTitle = iota
	actionItemFieldDescription
	actionItemFieldPriority
	actionItemFieldDue
	actionItemFieldLabels
	actionItemFieldDependsOn
	actionItemFieldBlockedBy
	actionItemFieldBlockedReason
	actionItemFieldObjective
	actionItemFieldAcceptanceCriteria
	actionItemFieldValidationPlan
	actionItemFieldRiskNotes
	actionItemFieldComments
	actionItemFieldSubtasks
	actionItemFieldResources
)

// thread panel focus indexes used by the full-page thread surface.
const (
	threadPanelDetails = iota
	threadPanelComments
	threadPanelContext
	threadPanelCount
)

// project-form field indexes used for focused form actions.
const (
	projectFieldName = iota
	projectFieldDescription
	projectFieldKind
	projectFieldOwner
	projectFieldIcon
	projectFieldColor
	projectFieldHomepage
	projectFieldTags
	projectFieldTemplateLibrary
	projectFieldRootPath
	projectFieldComments
)

// activity log limits used by modal rendering and retention.
const (
	activityLogMaxItems   = 200
	activityLogViewWindow = 14
	// actionItemInfoDetailsViewportMinHeight keeps a one-line markdown preview visible for short descriptions.
	actionItemInfoDetailsViewportMinHeight = 1
	// actionItemInfoDetailsViewportMaxHeight prevents details preview from crowding other actionItem-info sections.
	actionItemInfoDetailsViewportMaxHeight = 16
	// actionItemInfoBodyViewportMinHeight keeps full actionItem-info content scrollable on short terminals.
	actionItemInfoBodyViewportMinHeight = 8
	// actionItemInfoBodyViewportMaxHeight caps full-screen node modal body viewport height.
	actionItemInfoBodyViewportMaxHeight = 120
	// textEditHistoryLimit caps per-textarea undo/redo stack growth.
	textEditHistoryLimit  = 256
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
	// defaultSearchResultsLimit keeps TUI search views explicit while matching backend defaults.
	defaultSearchResultsLimit = 50
	// defaultEmbeddingsStatusLimit keeps the TUI embeddings inventory aligned with CLI defaults.
	defaultEmbeddingsStatusLimit = 100
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
	{ID: "actionItem-info", Label: "ActionItem Info"},
	{ID: "edit-actionItem", Label: "Edit ActionItem"},
	{ID: "move-left", Label: "Move Left"},
	{ID: "move-right", Label: "Move Right"},
	{ID: "archive-actionItem", Label: "Archive ActionItem"},
	{ID: "restore-actionItem", Label: "Restore ActionItem"},
	{ID: "hard-delete", Label: "Hard Delete"},
	{ID: "toggle-selection", Label: "Toggle Selection"},
	{ID: "clear-selection", Label: "Clear Selection"},
	{ID: "bulk-move-left", Label: "Bulk Move Left"},
	{ID: "bulk-move-right", Label: "Bulk Move Right"},
	{ID: "bulk-archive", Label: "Bulk Archive"},
	{ID: "bulk-hard-delete", Label: "Bulk Hard Delete"},
	{ID: "undo", Label: "Undo"},
	{ID: "redo", Label: "Redo"},
	{ID: "auth-access", Label: "Coordination"},
	{ID: "activity-log", Label: "Activity Log"},
}

// canonicalSearchStates stores canonical searchable lifecycle states.
var canonicalSearchStatesOrdered = []string{"todo", "progress", "done", "archived"}

// canonicalSearchLevelsOrdered stores canonical searchable hierarchy levels.
var canonicalSearchLevelsOrdered = []string{"project", "branch", "phase", "actionItem", "subtask"}

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
	"project":    "Project",
	"branch":     "Branch",
	"phase":      "Phase",
	"actionItem": "ActionItem",
	"subtask":    "Subtask",
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

// templateLibraryPickerItem describes one approved template-library selection row.
type templateLibraryPickerItem struct {
	LibraryID string
	Name      string
	Clear     bool
}

// projectKindPickerItem describes one project-kind selection row.
type projectKindPickerItem struct {
	KindID      domain.KindID
	DisplayName string
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
	Match app.ActionItemMatch
	Path  string
}

// pendingProjectTemplateReview stores one staged project save that is waiting on TUI migration review.
type pendingProjectTemplateReview struct {
	ProjectID                string
	Name                     string
	Description              string
	Kind                     domain.KindID
	Metadata                 domain.ProjectMetadata
	RootPath                 string
	TemplateLibraryID        string
	CurrentTemplateLibraryID string
}

// confirmAction describes a pending confirmation action.
type confirmAction struct {
	Kind                          string
	ActionItem                    domain.ActionItem
	Project                       domain.Project
	ActionItemIDs                 []string
	Mode                          app.DeleteMode
	Label                         string
	AuthRequestID                 string
	AuthRequestAttention          string
	AuthRequestPrincipal          string
	AuthRequestPrincipalRole      string
	AuthRequestClient             string
	AuthRequestReason             string
	AuthRequestRequestedBy        string
	AuthRequestResumeClient       string
	AuthRequestTimeout            string
	AuthRequestRequestedPath      string
	AuthRequestRequestedPathLabel string
	AuthRequestRequestedTTL       string
	AuthRequestPath               string
	AuthRequestPathLabel          string
	AuthRequestTTL                string
	AuthRequestDecision           string
	AuthRequestNote               string
	AuthSessionID                 string
	AuthSessionPrincipal          string
	AuthSessionPathLabel          string
	LeaseInstanceID               string
	LeaseAgentName                string
	LeaseScopeLabel               string
	HandoffID                     string
	HandoffSummary                string
	HandoffStatus                 string
	ReturnToAuthAccess            bool
}

// coordinationDetailTone classifies the coordination-detail modal chrome treatment.
type coordinationDetailTone string

// Coordination detail tones keep inspect modals aligned with item state instead of warning-red by default.
const (
	coordinationDetailToneNeutral coordinationDetailTone = "neutral"
	coordinationDetailToneActive  coordinationDetailTone = "active"
	coordinationDetailToneMuted   coordinationDetailTone = "muted"
	coordinationDetailToneWarn    coordinationDetailTone = "warn"
	coordinationDetailToneDanger  coordinationDetailTone = "danger"
	coordinationDetailToneSuccess coordinationDetailTone = "success"
)

// coordinationDetailAction describes one actionable command exposed from a coordination detail modal.
type coordinationDetailAction struct {
	Label   string
	Confirm confirmAction
}

// authScopePickerItem describes one user-facing auth scope option in the TUI picker.
type authScopePickerItem struct {
	Path        string
	Label       string
	Description string
}

// authInventoryItem describes one selectable coordination row in the recovery screen.
type authInventoryItem struct {
	Request         *domain.AuthRequest
	ResolvedRequest *domain.AuthRequest
	Session         *app.AuthSession
	Lease           *domain.CapabilityLease
	Handoff         *domain.Handoff
	Label           string
	Detail          string
}

// authInventoryViewMode identifies which coordination slice is visible in the full-screen dashboard.
type authInventoryViewMode int

// Coordination dashboard view modes.
const (
	authInventoryViewLive authInventoryViewMode = iota
	authInventoryViewHistory
)

// authInventorySectionData stores the visible coordination rows for the active dashboard view.
type authInventorySectionData struct {
	ActionRequiredCount int
	PendingRequests     []domain.AuthRequest
	ResolvedRequests    []domain.AuthRequest
	Sessions            []app.AuthSession
	Leases              []domain.CapabilityLease
	Handoffs            []domain.Handoff
}

// activityEntry describes one recorded user action for the in-app activity log.
type activityEntry struct {
	At         time.Time
	Summary    string
	Target     string
	EventID    int64
	ActionItemID string
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
	noticesSectionCoordination noticesSectionID = iota
	noticesSectionWarnings
	noticesSectionComments
	noticesSectionAttention
	noticesSectionRecentActivity
)

// confirm dialog focus indexes used by auth-request approval editing.
const (
	confirmFocusAuthDecision = iota
	confirmFocusAuthPath
	confirmFocusAuthTTL
	confirmFocusAuthNote
	confirmFocusButtons
)

// auth review stages used by the dedicated full-screen review surface.
const (
	authReviewStageSummary = iota
	authReviewStageEditTTL
	authReviewStageEditApproveNote
	authReviewStageDeny
)

// noticesPanelItem describes one selectable row in a notices-panel section.
type noticesPanelItem struct {
	Label                 string
	AttentionID           string
	AttentionKind         domain.AttentionKind
	HandoffID             string
	ActionItemID          string
	ProjectID             string
	ScopeType             domain.ScopeLevel
	ScopeID               string
	ThreadTitle           string
	ThreadDescription     string
	Activity              activityEntry
	HasActivity           bool
	CoordinationProjectID string
	CoordinationGlobal    bool
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
	noticesSectionAttention,
	noticesSectionWarnings,
	noticesSectionComments,
	noticesSectionCoordination,
	noticesSectionRecentActivity,
}

// noticesPanelFocusTarget identifies which notifications panel currently owns focus.
type noticesPanelFocusTarget int

// Notifications panel focus targets.
const (
	noticesPanelFocusProject noticesPanelFocusTarget = iota
	noticesPanelFocusGlobal
)

// globalNoticesPanelItem describes one selectable row in the global notifications panel.
type globalNoticesPanelItem struct {
	StableKey          string
	AttentionID        string
	AttentionKind      domain.AttentionKind
	HandoffID          string
	ProjectID          string
	ProjectLabel       string
	ScopeType          domain.ScopeLevel
	ScopeID            string
	Summary            string
	ActionItemID       string
	ThreadDescription  string
	CoordinationGlobal bool
}

// noticesCoordinationSummary stores live project-scoped coordination counts for the board notices panel.
type noticesCoordinationSummary struct {
	PendingRequests []domain.AuthRequest
	ActiveSessions  []app.AuthSession
	ActiveLeases    []domain.CapabilityLease
	OpenHandoffs    []domain.Handoff
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
	ActionItemID string
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

	actionItemFields  ActionItemFieldConfig
	defaultDeleteMode app.DeleteMode

	projects                 []domain.Project
	selectedProject          int
	columns                  []domain.Column
	tasks                    []domain.ActionItem
	selectedColumn           int
	selectedActionItem       int
	launchPicker             bool
	startupBootstrapRequired bool

	mode                  inputMode
	input                 string
	searchQuery           string
	searchApplied         bool
	showArchived          bool
	showArchivedProjects  bool
	searchIncludeArchived bool

	searchInput                   textinput.Model
	commandInput                  textinput.Model
	embeddingsFilterInput         textinput.Model
	bootstrapDisplayInput         textinput.Model
	pathsRootInput                textinput.Model
	highlightColorInput           textinput.Model
	dependencyInput               textinput.Model
	confirmAuthPathInput          textinput.Model
	confirmAuthTTLInput           textinput.Model
	confirmAuthNoteInput          textinput.Model
	authReviewStage               int
	authReviewScopePickerIndex    int
	authReviewScopePickerItems    []authScopePickerItem
	authReviewReturnStage         int
	authReviewReturnMode          inputMode
	authInventoryGlobal           bool
	authInventoryView             authInventoryViewMode
	authInventoryIndex            int
	authInventoryRequests         []domain.AuthRequest
	authInventoryResolvedRequests []domain.AuthRequest
	authInventorySessions         []app.AuthSession
	authInventoryLeases           []domain.CapabilityLease
	authInventoryHandoffs         []domain.Handoff
	authInventoryBody             viewport.Model
	authInventoryNeedsReload      bool
	coordinationDetailItem        authInventoryItem
	coordinationDetailTitle       string
	coordinationDetailBody        string
	coordinationDetailTone        coordinationDetailTone
	coordinationDetailActions     []coordinationDetailAction
	coordinationDetailActionIndex int
	threadInput                   textarea.Model
	threadDetailsInput            textarea.Model
	descriptionEditorInput        textarea.Model
	searchFocus                   int
	searchStateCursor             int
	searchLevelCursor             int
	searchCrossProject            bool
	searchMode                    app.SearchMode
	searchDefaultCrossProject     bool
	searchDefaultIncludeArchive   bool
	searchDefaultMode             app.SearchMode
	searchStates                  []string
	searchDefaultStates           []string
	searchLevels                  []string
	searchDefaultLevels           []string
	searchKinds                   []string
	searchLabelsAny               []string
	searchLabelsAll               []string
	searchMatches                 []app.ActionItemMatch
	searchRequestedMode           app.SearchMode
	searchEffectiveMode           app.SearchMode
	searchFallbackReason          string
	searchEmbeddingSummary        app.EmbeddingSummary
	searchResultIndex             int
	searchLoading                 bool
	searchOpeningResult           bool
	searchRequestSeq              int
	searchActiveRequestID         int
	embeddingsGlobal              bool
	embeddingsIncludeArchived     bool
	embeddingsFilterQuery         string
	embeddingsFilterActive        bool
	embeddingsRows                []app.EmbeddingRecord
	embeddingsAllDisplayRows      []embeddingsStatusDisplayRow
	embeddingsDisplayRows         []embeddingsStatusDisplayRow
	embeddingsSummary             app.EmbeddingSummary
	embeddingsProjectIDs          []string
	embeddingsScopeLabel          string
	embeddingsIndex               int
	embeddingsSpinner             spinner.Model
	embeddingsReindexInFlight     bool
	embeddingsReindexForce        bool
	quickActionIndex              int
	quickActionBackMode           inputMode
	commandMatches                []commandPaletteItem
	commandIndex                  int
	bootstrapFocus                int
	bootstrapActorIndex           int
	bootstrapRoots                []string
	bootstrapRootIndex            int
	bootstrapMandatory            bool
	dependencyFocus               int
	dependencyStateCursor         int
	dependencyCrossProject        bool
	dependencyIncludeArchived     bool
	dependencyStates              []string
	dependencyMatches             []dependencyCandidate
	dependencyIndex               int

	formInputs                []textinput.Model
	formFocus                 int
	actionItemFormDescription string
	actionItemFormMarkdown    map[int]string
	actionItemFormTouched     map[int]bool
	priorityIdx               int
	duePicker                 int
	duePickerFocus            int
	duePickerIncludeTime      bool
	pickerBack                inputMode
	duePickerDateInput        textinput.Model
	duePickerTimeInput        textinput.Model
	// actionItemFormResourceRefs stages resource refs while creating or editing a actionItem.
	actionItemFormResourceRefs []domain.ResourceRef
	// actionItemFormSubactionItemCursor tracks the focused subtask row in edit mode (0 = create new).
	actionItemFormSubactionItemCursor int
	// actionItemFormResourceCursor tracks the focused resource row in edit mode (0 = attach new).
	actionItemFormResourceCursor int
	// actionItemFormResourceEditIndex tracks which staged resource row is being replaced from picker flow (-1 = append).
	actionItemFormResourceEditIndex int

	projectPickerIndex                   int
	projectFormInputs                    []textinput.Model
	projectFormFocus                     int
	projectFormDescription               string
	templateMigrationReviewPreview       *domain.ProjectTemplateReapplyPreview
	templateMigrationReviewDraft         *pendingProjectTemplateReview
	templateMigrationReviewLoading       bool
	templateMigrationReviewIndex         int
	templateMigrationReviewPicked        map[string]struct{}
	kindDefinitions                      []domain.KindDefinition
	templateLibraries                    []domain.TemplateLibrary
	builtinTemplateStatuses              map[string]domain.BuiltinTemplateLibraryStatus
	currentProjectTemplateBinding        *domain.ProjectTemplateBinding
	actionItemNodeContracts              map[string]domain.NodeContractSnapshot
	descriptionEditorBack                inputMode
	descriptionEditorTarget              descriptionEditorTarget
	descriptionEditorActionItemFormField int
	descriptionEditorMode                descriptionEditorViewMode
	descriptionEditorPath                string
	descriptionEditorThreadDetails       bool
	descriptionEditorUndo                []string
	descriptionEditorRedo                []string
	labelsConfigInputs                   []textinput.Model
	labelsConfigFocus                    int
	labelsConfigSlug                     string
	labelsConfigBranchActionItemID       string
	labelsConfigPhaseActionItemID        string
	editingProjectID                     string
	editingActionItemID                  string
	actionItemInfoActionItemID           string
	actionItemInfoOriginActionItemID     string
	actionItemInfoPath                   []string
	actionItemInfoSubactionItemIdx       int
	actionItemInfoFocusedSubactionItemID string
	actionItemInfoComments               []domain.Comment
	actionItemInfoCommentsError          string
	actionItemFormParentID               string
	actionItemFormKind                   domain.Kind
	actionItemFormScope                  domain.KindAppliesTo
	actionItemFormBackMode               inputMode
	actionItemFormBackActionItemID       string
	actionItemFormBackChildID            string
	pendingProjectID                     string
	pendingOpenAuthInventory             bool
	pendingOpenAuthInventoryGlobal       bool
	pendingOpenAuthInventoryHandoffID    string
	pendingFocusActionItemID             string
	pendingActivityJumpActionItem        string
	pendingOpenActionItemInfoID          string
	pendingOpenActivityLog               bool
	pendingOpenThreadTarget              domain.CommentTarget
	pendingOpenThreadTitle               string
	pendingOpenThreadBody                string

	lastArchivedActionItemID string

	confirmDelete     bool
	confirmArchive    bool
	confirmHardDelete bool
	confirmRestore    bool
	pendingConfirm    confirmAction
	confirmChoice     int
	confirmFocus      int
	warningTitle      string
	warningBody       string
	warningReturnMode inputMode

	boardGroupBy    string
	showWIPWarnings bool
	dueSoonWindows  []time.Duration
	showDueSummary  bool
	searchRoots     []string
	projectRoots    map[string]string
	defaultRootDir  string
	highlightColor  string

	projectionRootActionItemID string

	selectedActionItemIDs  map[string]struct{}
	activityLog            []activityEntry
	noticesFocused         bool
	noticesPanel           noticesPanelFocusTarget
	noticesSection         noticesSectionID
	noticesCoordinationIdx int
	noticesWarnings        int
	noticesComments        int
	noticesAttention       int
	noticesActivity        int
	attentionItems         []domain.AttentionItem
	noticesCoordination    noticesCoordinationSummary
	globalNotices          []globalNoticesPanelItem
	globalNoticesIdx       int
	// globalNoticesPartialCount reports how many projects were skipped while aggregating global notices.
	globalNoticesPartialCount int
	globalNoticeTransition    globalNoticeTransitionTrace
	activityInfoItem          activityEntry
	undoStack                 []historyActionSet
	redoStack                 []historyActionSet
	nextHistoryID             int
	dependencyRollup          domain.DependencyRollup

	resourcePickerBack         inputMode
	resourcePickerActionItemID string
	resourcePickerRoot         string
	resourcePickerDir          string
	resourcePickerIndex        int
	resourcePickerItems        []resourcePickerEntry
	resourcePickerFilter       textinput.Model

	// pickerCore holds the shared file-picker state. Path-picker, file-picker,
	// and future variants all route through this sub-struct so Model field
	// additions stay bounded (synthesis §5.2 / Drop 1.5 P3-A acceptance criteria).
	pickerCore filePickerCore

	labelPickerBack            inputMode
	labelPickerIndex           int
	labelPickerItems           []labelPickerItem
	labelPickerAllItems        []labelPickerItem
	labelPickerInput           textinput.Model
	projectKindPickerBack      inputMode
	projectKindPickerIndex     int
	projectKindPickerItems     []projectKindPickerItem
	projectKindPickerInput     textinput.Model
	templateLibraryPickerBack  inputMode
	templateLibraryPickerIndex int
	templateLibraryPickerItems []templateLibraryPickerItem
	templateLibraryPickerInput textinput.Model

	dependencyBack              inputMode
	dependencyOwnerActionItemID string
	dependencyDependsOn         []string
	dependencyBlockedBy         []string
	dependencyActiveField       int
	dependencyDirty             bool

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
	threadComposerActive      bool
	threadDetailsActive       bool
	threadPanelFocus          int
	threadDetailsEditorActive bool
	threadComposerUndo        []string
	threadComposerRedo        []string
	threadMarkdown            *markdownRenderer
	actionItemInfoBody        viewport.Model
	actionItemInfoDetails     viewport.Model
	descriptionPreview        viewport.Model

	autoRefreshInterval time.Duration
	autoRefreshArmed    bool
	autoRefreshInFlight bool

	// diff owns the ctrl+d full-page diff surface. The pointer itself is the
	// one Model-level field this feature adds; the inner state (viewport,
	// differ, highlighter, result, error, dimensions) lives on *diffMode so
	// the Model-field budget stays within Drop 1.5 P4-T3 acceptance criteria.
	diff *diffMode

	// diffBackMode captures the mode active when ctrl+d was pressed so esc
	// restores the prior surface instead of unconditionally returning to the
	// board (falsification vector 5). Required second field alongside diff.
	diffBackMode inputMode

	// fileViewer owns the v-key full-page file-viewer surface. The pointer
	// itself is the one Model-level field this feature adds; inner state
	// (viewport, markdownRenderer, content) lives on *fileViewerMode.
	fileViewer *fileViewerMode

	// fileViewerBackMode captures the mode active when v was pressed so esc
	// restores the prior surface.
	fileViewerBackMode inputMode
}

// loadedMsg carries message data through update handling.
type loadedMsg struct {
	projects                  []domain.Project
	selectedProject           int
	columns                   []domain.Column
	tasks                     []domain.ActionItem
	kindDefinitions           []domain.KindDefinition
	templateLibraries         []domain.TemplateLibrary
	builtinTemplateStatuses   map[string]domain.BuiltinTemplateLibraryStatus
	projectTemplateBinding    *domain.ProjectTemplateBinding
	actionItemNodeContracts   map[string]domain.NodeContractSnapshot
	searchRequestedMode       app.SearchMode
	searchEffectiveMode       app.SearchMode
	searchFallbackReason      string
	searchEmbeddingSummary    app.EmbeddingSummary
	activityEntries           []activityEntry
	attentionItems            []domain.AttentionItem
	globalNotices             []globalNoticesPanelItem
	globalNoticesPartialCount int
	noticesCoordination       noticesCoordinationSummary
	rollup                    domain.DependencyRollup
	err                       error
	attentionItemsCount       int
	attentionUserActionCount  int
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
	err                error
	status             string
	reload             bool
	openAuthAccess     bool
	projectID          string
	projectRootSlug    string
	projectRootPath    string
	focusActionItemID  string
	clearSelect        bool
	clearActionItemIDs []string
	historyPush        *historyActionSet
	historyUndo        *historyActionSet
	historyRedo        *historyActionSet
	activityItem       *activityEntry
}

// authInventoryLoadedMsg carries coordination inventory for the recovery screen.
type authInventoryLoadedMsg struct {
	projectScoped bool
	projectID     string
	requests      []domain.AuthRequest
	sessions      []app.AuthSession
	leases        []domain.CapabilityLease
	handoffs      []domain.Handoff
	err           error
}

// modeKey returns a stable short string for one input mode in traces.
func modeKey(mode inputMode) string {
	switch mode {
	case modeNone:
		return "board"
	case modeAddActionItem:
		return "add-actionItem"
	case modeEditActionItem:
		return "edit-actionItem"
	case modeActionItemInfo:
		return "actionItem-info"
	case modeQuickActions:
		return "quick-actions"
	default:
		return "other"
	}
}

// actionItemUpdatedMsg carries one successful actionItem update with optional reopen context.
type actionItemUpdatedMsg struct {
	actionItem             domain.ActionItem
	status                 string
	reopenEditActionItemID string
	reselectChildID        string
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
	requestID int
	result    app.SearchActionItemMatchesResult
	err       error
}

// embeddingsStatusLoadedMsg carries embeddings lifecycle inventory rows for modal rendering.
type embeddingsStatusLoadedMsg struct {
	projectIDs  []string
	scopeLabel  string
	rows        []app.EmbeddingRecord
	displayRows []embeddingsStatusDisplayRow
	summary     app.EmbeddingSummary
	err         error
}

// embeddingsReindexMsg carries explicit embeddings reindex outcomes for the active scope.
type embeddingsReindexMsg struct {
	result app.ReindexEmbeddingsResult
	err    error
}

// embeddingsStatusDisplayRow stores one operator-facing TUI row resolved from one lifecycle record.
type embeddingsStatusDisplayRow struct {
	Record        app.EmbeddingRecord
	Project       domain.Project
	HasProject    bool
	ActionItem    domain.ActionItem
	HasActionItem bool
	ProjectLabel  string
	SubjectLabel  string
	TitleLabel    string
	PathLabel     string
	DetailLabel   string
	FilterLabel   string
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

// projectTemplateReviewLoadedMsg carries the staged reapply preview used by the TUI migration-review surface.
type projectTemplateReviewLoadedMsg struct {
	draft   pendingProjectTemplateReview
	preview domain.ProjectTemplateReapplyPreview
	err     error
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
	confirmAuthPathInput := newModalInput("path: ", "project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]]", "", 256)
	confirmAuthTTLInput := newModalInput("ttl: ", "for example 2h or 30m", "", 32)
	confirmAuthNoteInput := newModalInput("note: ", "optional note for the requester and audit trail", "", 256)
	embeddingsFilterInput := newModalInput("filter: ", "type to narrow lifecycle rows", "", 120)
	threadInput := textarea.New()
	threadInput.Prompt = ""
	threadInput.Placeholder = "Write markdown comment (Ctrl+S posts)"
	threadInput.CharLimit = 12000
	threadInput.ShowLineNumbers = false
	threadInput.SetHeight(2)
	threadDetailsInput := textarea.New()
	threadDetailsInput.Prompt = ""
	threadDetailsInput.Placeholder = "Edit markdown description. Ctrl+S saves."
	threadDetailsInput.CharLimit = 20000
	threadDetailsInput.ShowLineNumbers = true
	threadDetailsInput.SetHeight(12)
	descriptionEditorInput := textarea.New()
	descriptionEditorInput.Prompt = ""
	descriptionEditorInput.Placeholder = "Edit markdown description. Ctrl+S saves."
	descriptionEditorInput.CharLimit = 20000
	descriptionEditorInput.ShowLineNumbers = true
	descriptionEditorInput.SetHeight(12)
	descriptionPreview := viewport.New()
	descriptionPreview.SoftWrap = true
	descriptionPreview.MouseWheelEnabled = false
	descriptionPreview.FillHeight = true
	actionItemInfoBody := viewport.New()
	actionItemInfoBody.SoftWrap = true
	actionItemInfoBody.MouseWheelEnabled = false
	actionItemInfoBody.FillHeight = true
	authInventoryBody := viewport.New()
	authInventoryBody.SoftWrap = true
	authInventoryBody.MouseWheelEnabled = false
	authInventoryBody.FillHeight = true
	actionItemInfoDetails := viewport.New()
	actionItemInfoDetails.SoftWrap = true
	actionItemInfoDetails.MouseWheelEnabled = false
	actionItemInfoDetails.FillHeight = true
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
	projectKindPickerInput := textinput.New()
	projectKindPickerInput.Prompt = "filter: "
	projectKindPickerInput.Placeholder = "type to fuzzy-find project kinds"
	projectKindPickerInput.CharLimit = 120
	configureTextInputClipboardBindings(&projectKindPickerInput)
	templateLibraryPickerInput := textinput.New()
	templateLibraryPickerInput.Prompt = "filter: "
	templateLibraryPickerInput.Placeholder = "type to fuzzy-find approved template libraries"
	templateLibraryPickerInput.CharLimit = 120
	configureTextInputClipboardBindings(&templateLibraryPickerInput)
	embeddingsSpinner := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)),
	)
	m := Model{
		svc:                                  svc,
		status:                               "loading...",
		help:                                 h,
		keys:                                 newKeyMap(),
		actionItemFields:                     DefaultActionItemFieldConfig(),
		defaultDeleteMode:                    app.DeleteModeArchive,
		searchInput:                          searchInput,
		commandInput:                         commandInput,
		embeddingsFilterInput:                embeddingsFilterInput,
		bootstrapDisplayInput:                bootstrapDisplayInput,
		pathsRootInput:                       pathsRootInput,
		highlightColorInput:                  highlightColorInput,
		dependencyInput:                      dependencyInput,
		confirmAuthPathInput:                 confirmAuthPathInput,
		confirmAuthTTLInput:                  confirmAuthTTLInput,
		confirmAuthNoteInput:                 confirmAuthNoteInput,
		threadInput:                          threadInput,
		threadDetailsInput:                   threadDetailsInput,
		descriptionEditorInput:               descriptionEditorInput,
		authInventoryBody:                    authInventoryBody,
		actionItemInfoBody:                   actionItemInfoBody,
		actionItemInfoDetails:                actionItemInfoDetails,
		descriptionPreview:                   descriptionPreview,
		resourcePickerFilter:                 resourcePickerFilter,
		pickerCore:                           newFilePickerCore(),
		duePickerDateInput:                   duePickerDateInput,
		duePickerTimeInput:                   duePickerTimeInput,
		labelPickerInput:                     labelPickerInput,
		projectKindPickerInput:               projectKindPickerInput,
		templateLibraryPickerInput:           templateLibraryPickerInput,
		embeddingsSpinner:                    embeddingsSpinner,
		searchMode:                           app.SearchModeHybrid,
		searchStates:                         []string{"todo", "progress", "done"},
		searchDefaultMode:                    app.SearchModeHybrid,
		searchDefaultStates:                  []string{"todo", "progress", "done"},
		searchLevels:                         []string{"project", "branch", "phase", "actionItem", "subtask"},
		searchDefaultLevels:                  []string{"project", "branch", "phase", "actionItem", "subtask"},
		dependencyStates:                     []string{"todo", "progress", "done"},
		launchPicker:                         false,
		boardGroupBy:                         "none",
		showWIPWarnings:                      true,
		dueSoonWindows:                       []time.Duration{24 * time.Hour, time.Hour},
		showDueSummary:                       true,
		highlightColor:                       defaultHighlightColor,
		selectedActionItemIDs:                map[string]struct{}{},
		activityLog:                          []activityEntry{},
		noticesPanel:                         noticesPanelFocusProject,
		noticesSection:                       noticesSectionRecentActivity,
		globalNotices:                        []globalNoticesPanelItem{},
		confirmDelete:                        true,
		confirmArchive:                       true,
		confirmHardDelete:                    true,
		confirmRestore:                       false,
		actionItemFormKind:                   domain.KindActionItem,
		actionItemFormScope:                  domain.KindAppliesToActionItem,
		allowedLabelProject:                  map[string][]string{},
		searchRoots:                          []string{},
		projectRoots:                         map[string]string{},
		templateMigrationReviewPicked:        map[string]struct{}{},
		builtinTemplateStatuses:              map[string]domain.BuiltinTemplateLibraryStatus{},
		identityDisplayName:                  "tillsyn-user",
		identityActorID:                      "tillsyn-user",
		identityDefaultActorType:             string(domain.ActorTypeUser),
		descriptionEditorActionItemFormField: -1,
		actionItemFormResourceEditIndex:      -1,
		bootstrapActorIndex:                  0,
		bootstrapRoots:                       []string{},
	}
	if cwd, err := os.Getwd(); err == nil {
		m.defaultRootDir = cwd
	} else {
		m.defaultRootDir = "."
	}
	// threadMarkdown is heap-allocated so its address is stable across Model
	// copies. fileViewerMode.md holds the same pointer; pointer equality is
	// asserted by TestFileViewer_SharesThreadMarkdown.
	m.threadMarkdown = &markdownRenderer{}
	// Default diff-mode wiring uses the exec-backed Differ and chroma
	// Highlighter. Tests override this via WithDiffMode to inject deterministic
	// fakes, so real shell invocations never happen during unit runs.
	m.diff = newDiffMode(gitdiff.NewExecDiffer(), gitdiff.NewChromaHighlighter())
	// File viewer shares the model's glamour-backed markdownRenderer so
	// markdown rendering uses the same pipeline as the thread view. The
	// pointer m.threadMarkdown is stable for the lifetime of the Model.
	m.fileViewer = newFileViewerMode(m.threadMarkdown, defaultFileViewerConfig())
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
	case modeNone, modeActionItemInfo, modeActivityLog:
		return true
	default:
		return false
	}
}

// applyLoadedMsg applies a loaded message and returns any follow-up command.
func (m *Model) applyLoadedMsg(msg loadedMsg) tea.Cmd {
	applyStartedAt := time.Now()
	defer m.markGlobalNoticeApplyLoadedCompletion(applyStartedAt, msg.err)

	if msg.err != nil {
		m.err = msg.err
		return nil
	}
	previousGlobalNoticesKey := ""
	if selectedGlobalNotice, ok := m.selectedGlobalNoticesItem(); ok {
		previousGlobalNoticesKey = strings.TrimSpace(selectedGlobalNotice.StableKey)
	}
	m.err = nil
	m.projects = msg.projects
	m.selectedProject = msg.selectedProject
	m.columns = msg.columns
	m.tasks = msg.tasks
	m.kindDefinitions = append([]domain.KindDefinition(nil), msg.kindDefinitions...)
	m.templateLibraries = append([]domain.TemplateLibrary(nil), msg.templateLibraries...)
	if msg.builtinTemplateStatuses != nil {
		m.builtinTemplateStatuses = make(map[string]domain.BuiltinTemplateLibraryStatus, len(msg.builtinTemplateStatuses))
		for libraryID, status := range msg.builtinTemplateStatuses {
			m.builtinTemplateStatuses[libraryID] = status
		}
	} else {
		m.builtinTemplateStatuses = map[string]domain.BuiltinTemplateLibraryStatus{}
	}
	if msg.projectTemplateBinding != nil {
		binding := *msg.projectTemplateBinding
		m.currentProjectTemplateBinding = &binding
	} else {
		m.currentProjectTemplateBinding = nil
	}
	if msg.actionItemNodeContracts != nil {
		m.actionItemNodeContracts = make(map[string]domain.NodeContractSnapshot, len(msg.actionItemNodeContracts))
		for actionItemID, snapshot := range msg.actionItemNodeContracts {
			m.actionItemNodeContracts[actionItemID] = snapshot
		}
	} else {
		m.actionItemNodeContracts = map[string]domain.NodeContractSnapshot{}
	}
	m.searchRequestedMode = msg.searchRequestedMode
	m.searchEffectiveMode = msg.searchEffectiveMode
	m.searchFallbackReason = msg.searchFallbackReason
	m.searchEmbeddingSummary = msg.searchEmbeddingSummary
	if msg.activityEntries != nil {
		m.activityLog = append([]activityEntry(nil), msg.activityEntries...)
	}
	if msg.attentionItems != nil {
		m.attentionItems = append([]domain.AttentionItem(nil), msg.attentionItems...)
	}
	if msg.globalNotices != nil {
		m.globalNotices = append([]globalNoticesPanelItem(nil), msg.globalNotices...)
		m.reanchorGlobalNoticesSelection(previousGlobalNoticesKey)
	}
	m.noticesCoordination = msg.noticesCoordination
	m.globalNoticesPartialCount = max(0, msg.globalNoticesPartialCount)
	m.dependencyRollup = msg.rollup
	m.warnings = buildScopeWarnings(msg.attentionItemsCount, msg.attentionUserActionCount, m.globalNoticesPartialCount)
	if len(m.projects) == 0 {
		m.selectedProject = 0
		m.selectedColumn = 0
		m.selectedActionItem = 0
		m.projectPickerIndex = 0
		m.columns = nil
		m.tasks = nil
		m.builtinTemplateStatuses = map[string]domain.BuiltinTemplateLibraryStatus{}
		m.currentProjectTemplateBinding = nil
		m.actionItemNodeContracts = map[string]domain.NodeContractSnapshot{}
		m.activityLog = []activityEntry{}
		m.attentionItems = []domain.AttentionItem{}
		m.noticesCoordination = noticesCoordinationSummary{}
		m.globalNotices = []globalNoticesPanelItem{}
		m.globalNoticesIdx = 0
		m.globalNoticesPartialCount = 0
		m.pendingOpenActivityLog = false
		m.clearPendingNotificationThread()
		m.completeGlobalNoticeTransition("no_projects")
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
		pendingProjectID := m.pendingProjectID
		for idx, project := range m.projects {
			if project.ID == m.pendingProjectID {
				m.selectedProject = idx
				break
			}
		}
		m.traceGlobalNoticePending("clear", "pending_project_id", pendingProjectID, "reason", "apply_loaded")
		m.pendingProjectID = ""
	}
	if m.pendingOpenAuthInventory {
		global := m.pendingOpenAuthInventoryGlobal
		m.pendingOpenAuthInventory = false
		m.pendingOpenAuthInventoryGlobal = false
		return m.startAuthInventory(global)
	}
	if m.projectionRootActionItemID != "" {
		if _, ok := m.actionItemByID(m.projectionRootActionItemID); !ok {
			m.projectionRootActionItemID = ""
			m.status = "focus cleared (parent not found)"
		}
	}
	m.clampSelections()
	m.retainSelectionForLoadedActionItems()
	m.normalizePanelFocus()
	if m.pendingFocusActionItemID != "" {
		pendingFocusActionItemID := m.pendingFocusActionItemID
		m.focusActionItemByID(pendingFocusActionItemID)
		m.traceGlobalNoticePending("clear", "pending_focus_action_item_id", pendingFocusActionItemID, "reason", "apply_loaded")
		m.pendingFocusActionItemID = ""
	}
	if pendingActionItemID := strings.TrimSpace(m.pendingOpenActionItemInfoID); pendingActionItemID != "" {
		if _, ok := m.actionItemByID(pendingActionItemID); ok {
			m.focusActionItemByID(pendingActionItemID)
			if m.openActionItemInfo(pendingActionItemID, "actionItem info") {
				m.noticesFocused = false
				m.clearPendingNotificationThread()
			} else {
				m.status = "notification actionItem not found"
			}
			m.traceGlobalNoticePending("clear", "pending_open_task_info_id", pendingActionItemID, "reason", "task_found")
			m.pendingOpenActionItemInfoID = ""
		} else if !m.showArchived {
			m.showArchived = true
			m.pendingFocusActionItemID = pendingActionItemID
			m.traceGlobalNoticePending("set", "pending_focus_action_item_id", pendingActionItemID, "reason", "retry_include_archived")
			m.status = "loading notification actionItem..."
			return m.loadData
		} else {
			m.traceGlobalNoticePending("clear", "pending_open_task_info_id", pendingActionItemID, "reason", "task_missing_after_reload")
			m.pendingOpenActionItemInfoID = ""
			if cmd := m.applyPendingNotificationThread(); cmd != nil {
				return cmd
			}
			m.status = "notification actionItem not found"
		}
	}
	if cmd := m.applyPendingNotificationThread(); cmd != nil {
		return cmd
	}
	if m.pendingOpenActivityLog {
		m.mode = modeActivityLog
		m.noticesFocused = false
		m.pendingOpenActivityLog = false
		m.status = "activity log"
	}
	if pendingJump := strings.TrimSpace(m.pendingActivityJumpActionItem); pendingJump != "" {
		if _, ok := m.actionItemByID(pendingJump); ok {
			m.prepareActivityJumpContext(pendingJump)
			if m.focusActionItemByID(pendingJump) {
				m.status = "jumped to activity node"
			} else {
				m.status = "activity node unavailable (possibly hard-deleted)"
			}
		} else {
			m.status = "activity node unavailable (possibly hard-deleted)"
		}
		m.pendingActivityJumpActionItem = ""
	}
	if m.mode == modeActionItemInfo {
		if currentID := strings.TrimSpace(m.actionItemInfoActionItemID); currentID != "" {
			if actionItem, ok := m.actionItemByID(currentID); ok {
				m.reanchorActionItemInfoSubactionItemSelection(currentID)
				m.syncActionItemInfoDetailsViewport(actionItem)
				m.syncActionItemInfoBodyViewport(actionItem)
			}
		}
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
	if m.searchApplied && !m.searchCrossProject {
		status := fmt.Sprintf("%d matches", len(msg.tasks))
		if len(msg.tasks) == 0 {
			status = "no matches"
		}
		if modeLabel := strings.TrimSpace(searchResultsModeLabel(m.searchRequestedMode, m.searchEffectiveMode, m.searchFallbackReason)); modeLabel != "" {
			if len(msg.tasks) == 0 {
				status = "no matches • " + modeLabel
			} else {
				status = fmt.Sprintf("%d matches • %s", len(msg.tasks), modeLabel)
			}
		}
		if summaryLabel := strings.TrimSpace(searchResultsEmbeddingSummaryLabel(m.searchEmbeddingSummary)); summaryLabel != "" {
			status += " • " + summaryLabel
		}
		m.status = status
	}
	if m.status == "" || m.status == "loading..." {
		m.status = "ready"
	}
	return nil
}

// setPendingNotificationThread stores one deferred thread-open action for applyLoadedMsg.
func (m *Model) setPendingNotificationThread(target domain.CommentTarget, title, body string) {
	m.pendingOpenThreadTarget = target
	m.pendingOpenThreadTitle = strings.TrimSpace(title)
	m.pendingOpenThreadBody = strings.TrimSpace(body)
	targetKey := fmt.Sprintf("%s/%s/%s", strings.TrimSpace(target.ProjectID), strings.TrimSpace(string(target.TargetType)), strings.TrimSpace(target.TargetID))
	m.traceGlobalNoticePending("set", "pending_notification_thread", targetKey, "reason", "set_pending_thread")
}

// clearPendingNotificationThread clears one deferred thread-open action.
func (m *Model) clearPendingNotificationThread() {
	targetKey := fmt.Sprintf(
		"%s/%s/%s",
		strings.TrimSpace(m.pendingOpenThreadTarget.ProjectID),
		strings.TrimSpace(string(m.pendingOpenThreadTarget.TargetType)),
		strings.TrimSpace(m.pendingOpenThreadTarget.TargetID),
	)
	if targetKey != "//" || strings.TrimSpace(m.pendingOpenThreadTitle) != "" || strings.TrimSpace(m.pendingOpenThreadBody) != "" {
		m.traceGlobalNoticePending("clear", "pending_notification_thread", targetKey, "reason", "clear_pending_thread")
	}
	m.pendingOpenThreadTarget = domain.CommentTarget{}
	m.pendingOpenThreadTitle = ""
	m.pendingOpenThreadBody = ""
}

// pendingNotificationThread returns one normalized deferred thread-open action when available.
func (m Model) pendingNotificationThread() (domain.CommentTarget, string, string, bool) {
	target, err := domain.NormalizeCommentTarget(m.pendingOpenThreadTarget)
	if err != nil {
		return domain.CommentTarget{}, "", "", false
	}
	title := strings.TrimSpace(m.pendingOpenThreadTitle)
	if title == "" {
		title = "notification thread"
	}
	return target, title, strings.TrimSpace(m.pendingOpenThreadBody), true
}

// applyPendingNotificationThread opens one deferred notification thread after project data reload.
func (m *Model) applyPendingNotificationThread() tea.Cmd {
	target, title, body, ok := m.pendingNotificationThread()
	if !ok {
		return nil
	}
	updated, cmd := m.startNotificationThread(target, title, body)
	next, castOK := updated.(Model)
	if castOK {
		*m = next
	}
	m.clearPendingNotificationThread()
	return cmd
}

// startNotificationThread opens thread mode from one notifications-panel action.
func (m Model) startNotificationThread(target domain.CommentTarget, title, body string) (tea.Model, tea.Cmd) {
	updated, cmd := m.startThread(modeNone, target, title, body, threadPanelDetails)
	next, ok := updated.(Model)
	if !ok {
		return updated, cmd
	}
	next.noticesFocused = false
	return next, cmd
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
		if m.mode == modeDescriptionEditor {
			if m.descriptionEditorMode == descriptionEditorViewModeEdit {
				m.syncDescriptionPreviewOffsetToEditor()
			} else {
				m.syncDescriptionEditorViewportLayout()
			}
		}
		if m.mode == modeAuthInventory {
			m.syncAuthInventoryViewport()
		}
		if m.mode == modeDiff && m.diff != nil {
			// Match the chrome math used at render time so mid-session resize
			// doesn't skew banner placement until the next frame.
			accent := lipgloss.Color("62")
			if project, ok := m.currentProject(); ok {
				accent = projectAccentColor(project)
			}
			muted := lipgloss.Color("241")
			dim := lipgloss.Color("239")
			metrics := m.fullPageSurfaceMetrics(
				accent, muted, dim,
				actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())),
				"Git Diff", "", "",
			)
			m.diff.resize(metrics.contentWidth, max(1, metrics.bodyHeight-1))
		}
		if m.mode == modeFileViewer && m.fileViewer != nil {
			accent := lipgloss.Color("62")
			if project, ok := m.currentProject(); ok {
				accent = projectAccentColor(project)
			}
			muted := lipgloss.Color("241")
			dim := lipgloss.Color("239")
			metrics := m.fullPageSurfaceMetrics(
				accent, muted, dim,
				actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())),
				"File Viewer", "", "",
			)
			m.fileViewer.resize(metrics.contentWidth, max(1, metrics.bodyHeight-1))
		}
		m.normalizePanelFocus()
		return m, nil

	case loadedMsg:
		m.autoRefreshInFlight = false
		if m.searchOpeningResult {
			m.searchLoading = false
			m.searchOpeningResult = false
		}
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

	case spinner.TickMsg:
		if !m.embeddingsReindexInFlight {
			return m, nil
		}
		cmds := make([]tea.Cmd, 0, 1)
		if m.embeddingsReindexInFlight {
			var cmd tea.Cmd
			m.embeddingsSpinner, cmd = m.embeddingsSpinner.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

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

	case actionItemUpdatedMsg:
		m.err = nil
		m.replaceActionItemInMemory(msg.actionItem)
		traceActionItemScreenAction(
			"task_edit",
			"task_updated",
			"action_item_id", msg.actionItem.ID,
			"reopen_parent_action_item_id", strings.TrimSpace(msg.reopenEditActionItemID),
			"reselect_child_id", strings.TrimSpace(msg.reselectChildID),
		)
		if msg.status != "" {
			m.status = msg.status
		}
		if parentID := strings.TrimSpace(msg.reopenEditActionItemID); parentID != "" {
			parent, ok := m.actionItemByID(parentID)
			if !ok {
				m.status = "parent actionItem not found"
				return m, m.loadData
			}
			cmd := m.startActionItemForm(&parent)
			m.selectActionItemFormSubactionItemByID(msg.reselectChildID)
			m.syncActionItemFormViewportToFocus()
			return m, tea.Batch(cmd, m.loadData)
		}
		return m, m.loadData

	case projectTemplateReviewLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.mode = modeEditProject
			m.status = "template reapply review unavailable"
			m.templateMigrationReviewLoading = false
			m.templateMigrationReviewPreview = nil
			m.templateMigrationReviewDraft = nil
			m.templateMigrationReviewPicked = map[string]struct{}{}
			m.templateMigrationReviewIndex = 0
			return m, nil
		}
		m.err = nil
		m.templateMigrationReviewLoading = false
		m.mode = modeTemplateMigrationReview
		m.templateMigrationReviewDraft = &msg.draft
		preview := msg.preview
		m.templateMigrationReviewPreview = &preview
		m.templateMigrationReviewIndex = 0
		m.templateMigrationReviewPicked = map[string]struct{}{}
		if !preview.ReviewRequired && len(preview.MigrationCandidates) == 0 {
			return m.applyProjectTemplateReviewDecision(false, false)
		}
		if preview.EligibleMigrationCount > 0 {
			m.status = "review template migrations"
		} else {
			m.status = "review template drift"
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
		if msg.focusActionItemID != "" {
			m.pendingFocusActionItemID = msg.focusActionItemID
		}
		if msg.clearSelect {
			m.clearSelection()
		}
		if len(msg.clearActionItemIDs) > 0 {
			m.unselectActionItems(msg.clearActionItemIDs)
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
		if msg.openAuthAccess {
			m.mode = modeAuthInventory
			if msg.reload {
				m.authInventoryNeedsReload = true
				return m, m.loadAuthInventoryCmd()
			}
			return m, nil
		}
		if m.mode == modeAuthInventory && msg.reload {
			m.authInventoryNeedsReload = true
			return m, m.loadAuthInventoryCmd()
		}
		if msg.reload {
			return m, m.loadData
		}
		return m, nil

	case authInventoryLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.authInventoryRequests = m.authInventoryRequests[:0]
		m.authInventoryResolvedRequests = m.authInventoryResolvedRequests[:0]
		for _, request := range msg.requests {
			if domain.NormalizeAuthRequestState(request.State) == domain.AuthRequestStatePending {
				m.authInventoryRequests = append(m.authInventoryRequests, request)
				continue
			}
			m.authInventoryResolvedRequests = append(m.authInventoryResolvedRequests, request)
		}
		m.authInventorySessions = append([]app.AuthSession(nil), msg.sessions...)
		m.authInventoryLeases = append([]domain.CapabilityLease(nil), msg.leases...)
		m.authInventoryHandoffs = append([]domain.Handoff(nil), msg.handoffs...)
		m.clampAuthInventoryIndex()
		scopeText := strings.TrimSpace(m.authInventoryScopeLabel())
		if scopeText == "" {
			scopeText = "all projects"
		}
		requestSessionScope := "global (" + scopeText + ")"
		if !m.authInventoryGlobal {
			requestSessionScope = "project scope (" + scopeText + ")"
		}
		statusPrefix := "coordination " + m.authInventoryViewLabel() + ": "
		if coordinationProjectID, coordinationProjectLabel, ok := m.authInventoryCoordinationProject(); ok && coordinationProjectID != "" {
			m.status = statusPrefix + "requests/sessions " + requestSessionScope + " • project-local " + coordinationProjectLabel
		} else {
			m.status = statusPrefix + "requests/sessions " + requestSessionScope
		}
		if handoffID := strings.TrimSpace(m.pendingOpenAuthInventoryHandoffID); handoffID != "" {
			m.pendingOpenAuthInventoryHandoffID = ""
			items := m.authInventoryItems()
			found := false
			for idx, item := range items {
				if item.Handoff == nil || strings.TrimSpace(item.Handoff.ID) != handoffID {
					continue
				}
				m.authInventoryIndex = idx
				title, body, ok := m.authInventoryItemDetail(item)
				if !ok {
					break
				}
				m.openCoordinationDetail(item, title, body)
				found = true
				break
			}
			if !found {
				m.status = "handoff not found in coordination"
			}
		}
		m.syncAuthInventoryViewport()
		return m, nil

	case searchResultsMsg:
		if msg.requestID != 0 && msg.requestID != m.searchActiveRequestID {
			return m, nil
		}
		m.searchActiveRequestID = 0
		m.searchLoading = false
		m.searchOpeningResult = false
		if msg.err != nil {
			m.err = msg.err
			m.mode = modeSearchResults
			m.status = "search failed"
			return m, nil
		}
		m.err = nil
		m.searchMatches = msg.result.Matches
		m.searchRequestedMode = msg.result.RequestedMode
		m.searchEffectiveMode = msg.result.EffectiveMode
		m.searchFallbackReason = msg.result.FallbackReason
		m.searchEmbeddingSummary = msg.result.EmbeddingSummary
		m.searchResultIndex = clamp(m.searchResultIndex, 0, len(m.searchMatches)-1)
		m.mode = modeSearchResults
		m.status = searchResultsStatusSummary(m.searchMatches, m.searchRequestedMode, m.searchEffectiveMode, m.searchFallbackReason)
		return m, nil

	case embeddingsStatusLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.embeddingsProjectIDs = append([]string(nil), msg.projectIDs...)
		m.embeddingsScopeLabel = strings.TrimSpace(msg.scopeLabel)
		m.embeddingsRows = append([]app.EmbeddingRecord(nil), msg.rows...)
		m.embeddingsAllDisplayRows = append([]embeddingsStatusDisplayRow(nil), msg.displayRows...)
		m.embeddingsSummary = msg.summary
		m.applyEmbeddingsDisplayFilter()
		m.status = "embeddings inventory"
		return m, nil

	case embeddingsReindexMsg:
		m.embeddingsReindexInFlight = false
		m.embeddingsReindexForce = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status = fmt.Sprintf(
			"embeddings reindex queued:%d ready:%d failed:%d",
			msg.result.QueuedCount,
			msg.result.ReadyCount,
			msg.result.FailedCount,
		)
		return m, m.loadEmbeddingsStatusCmd()

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
		m.status = "ready"
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

	case diffLoadedMsg:
		return m.applyDiffLoadedMsg(msg)

	case tea.KeyPressMsg:
		// Always honor terminal interrupt for deterministic emergency exit across all modes.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.traceGlobalNoticeKeyDispatch(msg)
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
	if m.mode == modeDescriptionEditor {
		return m.renderDescriptionEditorModeView()
	}
	if m.mode == modeThread {
		return m.renderThreadModeView()
	}
	if m.mode == modeDiff {
		return m.renderDiffModeView()
	}
	if m.mode == modeFileViewer {
		return m.renderFileViewerModeView()
	}
	if m.mode == modeAuthReview {
		return m.renderAuthReviewModeView()
	}
	if m.mode == modeAuthScopePicker {
		return m.renderAuthScopePickerModeView()
	}
	if m.mode == modeAuthInventory {
		return m.renderAuthInventoryModeView()
	}
	if m.mode == modeCoordinationDetail {
		return m.renderCoordinationDetailModeView()
	}
	if m.mode == modeAuthSessionRevoke {
		return m.renderAuthSessionRevokeModeView()
	}
	if isFullPageNodeMode(m.mode) {
		return m.renderFullPageNodeModeView()
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
		innerWidth := m.appInnerWidth()
		if innerWidth > 0 {
			content = lipgloss.NewStyle().Width(innerWidth).Render(content)
		}
		content = applyOuterHorizontalPadding(content)
		helpLine := applyOuterHorizontalPadding(m.renderBottomHelpLine(muted, dim, innerWidth))
		contentHeight := lipgloss.Height(content)
		if m.height > 0 {
			helpHeight := lipgloss.Height(helpLine)
			contentHeight = max(0, m.height-helpHeight)
			content = fitLines(content, contentHeight)
		}
		fullContent := content + "\n" + helpLine
		if overlay := m.renderModeOverlay(accent, muted, dim, helpStyle, m.fullPageNodeContentWidth()); overlay != "" {
			if isFullPageNodeMode(m.mode) {
				body := overlay
				if m.height > 0 {
					helpHeight := lipgloss.Height(helpLine)
					body = fitLines(body, max(0, m.height-helpHeight))
				}
				headerBlock := m.appHeaderBlock(statusStyle, innerWidth)
				body = lipgloss.PlaceHorizontal(max(1, innerWidth-(2*tuiOuterHorizontalPadding)), lipgloss.Center, body)
				fullBody := strings.Join([]string{headerBlock, "", body, ""}, "\n")
				if innerWidth > 0 {
					fullBody = lipgloss.NewStyle().
						PaddingLeft(tuiOuterHorizontalPadding).
						PaddingRight(tuiOuterHorizontalPadding).
						Render(fullBody)
				}
				fullContent = fullBody + "\n" + helpLine
			} else {
				overlayHeight := lipgloss.Height(fullContent)
				if m.height > 0 {
					overlayHeight = m.height
				}
				fullContent = overlayOnContent(fullContent, overlay, max(1, m.width), max(1, overlayHeight))
			}
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
	actionItemByID := m.tasksByID()
	attentionItems, attentionTotal, attentionBlocked, attentionTop := m.scopeAttentionSummary(actionItemByID)

	layoutWidth := m.appInnerWidth()
	headerBlock := m.appHeaderBlock(statusStyle, layoutWidth)
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
		highlight := m.selectedActionItemHighlightColor()
		selectedActionItemStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true)
		selectedMultiActionItemStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true).Underline(true)
		// Multi-select should be indicated by marker stars only; avoid extra row background fill.
		multiSelectedActionItemStyle := lipgloss.NewStyle()
		itemSubStyle := lipgloss.NewStyle().Foreground(muted)
		groupStyle := lipgloss.NewStyle().Bold(true).Foreground(muted)
		warningStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))

		for colIdx, column := range m.columns {
			colRenderWidth := colWidth + extraBoardWidthPerColumn
			if colIdx < extraBoardWidthRemainder {
				colRenderWidth++
			}
			colActionItems := m.boardActionItemsForColumn(column.ID)
			parentByID := map[string]string{}
			for _, actionItem := range colActionItems {
				parentByID[actionItem.ID] = actionItem.ParentID
			}
			activeCount := 0
			for _, actionItem := range colActionItems {
				if actionItem.ArchivedAt == nil {
					activeCount++
				}
			}

			colHeader := fmt.Sprintf("%s (%d)", column.Name, len(colActionItems))
			if column.WIPLimit > 0 {
				colHeader = fmt.Sprintf("%s (%d/%d)", column.Name, activeCount, column.WIPLimit)
			}
			headerLines := []string{colTitle.Render(colHeader)}
			if m.showWIPWarnings && column.WIPLimit > 0 && activeCount > column.WIPLimit {
				headerLines = append(headerLines, warningStyle.Render(fmt.Sprintf("WIP limit exceeded: %d/%d", activeCount, column.WIPLimit)))
			}

			actionItemLines := make([]string, 0, max(1, len(colActionItems)*3))
			selectedStart := -1
			selectedEnd := -1

			if len(colActionItems) == 0 {
				actionItemLines = append(actionItemLines, archivedStyle.Render("(empty)"))
			} else {
				prevGroup := ""
				for actionItemIdx, actionItem := range colActionItems {
					if m.boardGroupBy != "none" {
						groupLabel := m.groupLabelForActionItem(actionItem)
						if actionItemIdx == 0 || groupLabel != prevGroup {
							if actionItemIdx > 0 {
								actionItemLines = append(actionItemLines, "")
							}
							actionItemLines = append(actionItemLines, groupStyle.Render(groupLabel))
							prevGroup = groupLabel
						}
					}
					selected := colIdx == m.selectedColumn && actionItemIdx == m.selectedActionItem
					multiSelected := m.isActionItemSelected(actionItem.ID)

					prefix := "   "
					switch {
					case selected && multiSelected:
						prefix = "│* "
					case selected:
						prefix = "│  "
					case multiSelected:
						prefix = " * "
					}
					depth := actionItemDepth(actionItem.ID, parentByID, 0)
					indent := strings.Repeat("  ", min(depth, 4))
					attentionCount := m.actionItemAttentionCount(actionItem, actionItemByID)
					attentionSuffix := ""
					if attentionCount > 0 {
						attentionSuffix = fmt.Sprintf(" !%d", attentionCount)
					}
					titleWidth := max(1, colRenderWidth-(10+2*min(depth, 4))-utf8.RuneCountInString(attentionSuffix))
					title := prefix + indent + truncate(actionItem.Title, titleWidth) + attentionSuffix
					sub := m.actionItemListSecondary(actionItem)
					if sub != "" {
						sub = indent + truncate(sub, max(1, colRenderWidth-(10+2*min(depth, 4))))
					}
					if actionItem.ArchivedAt != nil {
						title = archivedStyle.Render(title)
						if sub != "" {
							sub = archivedStyle.Render(sub)
						}
					} else {
						switch {
						case selected && multiSelected:
							title = selectedMultiActionItemStyle.Render(title)
						case selected:
							title = selectedActionItemStyle.Render(title)
						case multiSelected:
							title = multiSelectedActionItemStyle.Render(title)
						}
					}

					rowStart := len(actionItemLines)
					actionItemLines = append(actionItemLines, title)
					if sub != "" {
						// Keep selection/focus markers on the title row only to avoid duplicate stars/cursor bars.
						subPrefix := "   "
						actionItemLines = append(actionItemLines, subPrefix+itemSubStyle.Render(sub))
					}
					if actionItemIdx < len(colActionItems)-1 {
						actionItemLines = append(actionItemLines, "")
					}
					if selected {
						selectedStart = rowStart
						selectedEnd = len(actionItemLines) - 1
					}
				}
			}

			innerHeight := max(1, colHeight-4)
			headerBlock := append([]string{}, headerLines...)
			headerBlock = append(headerBlock, "")
			actionItemWindowHeight := max(1, innerHeight-len(headerBlock))
			scrollTop := 0
			if colIdx == m.selectedColumn && selectedStart >= 0 {
				if selectedEnd >= scrollTop+actionItemWindowHeight {
					scrollTop = selectedEnd - actionItemWindowHeight + 1
				}
				if selectedStart < scrollTop {
					scrollTop = selectedStart
				}
			}
			maxScrollTop := max(0, len(actionItemLines)-actionItemWindowHeight)
			scrollTop = clamp(scrollTop, 0, maxScrollTop)
			if len(actionItemLines) > actionItemWindowHeight {
				actionItemLines = actionItemLines[scrollTop : scrollTop+actionItemWindowHeight]
			}
			if len(actionItemLines) < actionItemWindowHeight {
				actionItemLines = append(actionItemLines, make([]string, actionItemWindowHeight-len(actionItemLines))...)
			}

			lines := append(headerBlock, actionItemLines...)
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
				colHeight,
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

	sections := []string{headerBlock, "", mainArea}
	if infoLine != "" {
		sections = append(sections, infoLine)
	}
	if strings.TrimSpace(m.projectionRootActionItemID) != "" {
		sections = append(sections, statusStyle.Render(fmt.Sprintf("subtree focus active • %s full board", m.keys.clearFocus.Help().Key)))
	}
	if count := len(m.selectedActionItemIDs); count > 0 {
		sections = append(sections, statusStyle.Render(fmt.Sprintf("%d tasks selected • %s toggle • esc clear", count, m.keys.multiSelect.Help().Key)))
	}
	if status := m.boardStatusText(); status != "" {
		sections = append(sections, statusStyle.Render(status))
	}
	content := strings.Join(sections, "\n")
	content = applyOuterHorizontalPadding(content)

	innerWidth := layoutWidth
	helpLine := applyOuterHorizontalPadding(m.renderBottomHelpLine(muted, dim, innerWidth))

	contentHeight := lipgloss.Height(content)
	if m.height > 0 {
		helpHeight := lipgloss.Height(helpLine)
		contentHeight = max(0, m.height-helpHeight)
		content = fitLines(content, contentHeight)
	}

	fullContent := content + "\n" + helpLine
	if overlay != "" {
		if !m.help.ShowAll && isFullPageNodeMode(m.mode) {
			body := overlay
			fullSections := []string{headerBlock, ""}
			body = lipgloss.PlaceHorizontal(max(1, innerWidth-(2*tuiOuterHorizontalPadding)), lipgloss.Center, body)
			fullSections = append(fullSections, body, "")
			fullBody := strings.Join(fullSections, "\n")
			if layoutWidth > 0 {
				fullBody = lipgloss.NewStyle().
					PaddingLeft(tuiOuterHorizontalPadding).
					PaddingRight(tuiOuterHorizontalPadding).
					Render(fullBody)
			}
			if m.height > 0 {
				helpHeight := lipgloss.Height(helpLine)
				fullBody = fitLines(fullBody, max(0, m.height-helpHeight))
			}
			fullContent = fullBody + "\n" + helpLine
		} else {
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

// activeMouseMode returns the current mouse mode, including clipboard-friendly selection mode.
func (m Model) activeMouseMode() tea.MouseMode {
	if m.mouseSelectionMode {
		return tea.MouseModeNone
	}
	return tea.MouseModeCellMotion
}

// loadData loads required data for the current operation.
func (m Model) loadData() tea.Msg {
	totalStartedAt := time.Now()

	projectsStartedAt := time.Now()
	projects, err := m.svc.ListProjects(context.Background(), m.showArchivedProjects)
	m.traceLoadDataStage("projects", projectsStartedAt, err, "count", len(projects), "show_archived_projects", m.showArchivedProjects)
	if err != nil {
		m.traceLoadDataStage("total", totalStartedAt, err, "project_count", 0, "column_count", 0, "task_count", 0)
		return loadedMsg{err: err}
	}
	kindDefinitions, err := m.svc.ListKindDefinitions(context.Background(), false)
	if err != nil {
		m.traceLoadDataStage("total", totalStartedAt, err, "project_count", len(projects), "column_count", 0, "task_count", 0)
		return loadedMsg{err: err}
	}
	templateLibraries, err := m.svc.ListTemplateLibraries(context.Background(), app.ListTemplateLibrariesInput{
		Scope:  domain.TemplateLibraryScopeGlobal,
		Status: domain.TemplateLibraryStatusApproved,
	})
	if err != nil {
		m.traceLoadDataStage("total", totalStartedAt, err, "project_count", len(projects), "column_count", 0, "task_count", 0)
		return loadedMsg{err: err}
	}
	if len(projects) == 0 {
		builtinTemplateStatuses, statusErr := m.loadBuiltinTemplateStatuses(templateLibraries)
		if statusErr != nil {
			m.traceLoadDataStage("total", totalStartedAt, statusErr, "project_count", 0, "column_count", 0, "task_count", 0)
			return loadedMsg{err: statusErr}
		}
		m.traceLoadDataStage("total", totalStartedAt, nil, "project_count", 0, "column_count", 0, "task_count", 0)
		return loadedMsg{
			projects:                projects,
			kindDefinitions:         kindDefinitions,
			templateLibraries:       templateLibraries,
			builtinTemplateStatuses: builtinTemplateStatuses,
			actionItemNodeContracts: map[string]domain.NodeContractSnapshot{},
		}
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
	var projectTemplateBinding *domain.ProjectTemplateBinding
	binding, err := m.svc.GetProjectTemplateBinding(context.Background(), projectID)
	switch {
	case err == nil:
		projectTemplateBinding = &binding
	case errors.Is(err, app.ErrNotFound):
		projectTemplateBinding = nil
	default:
		m.traceLoadDataStage("total", totalStartedAt, err, "project_count", len(projects), "column_count", 0, "task_count", 0)
		return loadedMsg{err: err}
	}
	builtinTemplateStatuses, err := m.loadBuiltinTemplateStatuses(templateLibraries)
	if err != nil {
		m.traceLoadDataStage("total", totalStartedAt, err, "project_count", len(projects), "column_count", 0, "task_count", 0)
		return loadedMsg{err: err}
	}
	columnsStartedAt := time.Now()
	columns, err := m.svc.ListColumns(context.Background(), projectID, false)
	m.traceLoadDataStage("columns", columnsStartedAt, err, "project_id", projectID, "count", len(columns))
	if err != nil {
		m.traceLoadDataStage("total", totalStartedAt, err, "project_count", len(projects), "column_count", 0, "task_count", 0)
		return loadedMsg{err: err}
	}

	tasksStartedAt := time.Now()
	var tasks []domain.ActionItem
	searchRequestedMode := app.SearchMode("")
	searchEffectiveMode := app.SearchMode("")
	searchFallbackReason := ""
	searchEmbeddingSummary := app.EmbeddingSummary{}
	searchFilterActive := m.searchApplied
	searchMatchCount := 0
	actionItemSource := "list_tasks"
	if searchFilterActive {
		result, searchErr := m.svc.SearchActionItems(context.Background(), app.SearchActionItemsFilter{
			ProjectID:       projectID,
			Query:           m.searchQuery,
			CrossProject:    m.searchCrossProject,
			IncludeArchived: m.searchIncludeArchived,
			States:          append([]string(nil), m.searchStates...),
			Levels:          canonicalSearchLevels(m.searchLevels),
			Kinds:           append([]string(nil), m.searchKinds...),
			LabelsAny:       append([]string(nil), m.searchLabelsAny...),
			LabelsAll:       append([]string(nil), m.searchLabelsAll...),
			Mode:            m.searchMode,
			Sort:            app.SearchSortRankDesc,
			Limit:           defaultSearchResultsLimit,
			Offset:          0,
		})
		if searchErr != nil {
			m.traceLoadDataStage("tasks_search", tasksStartedAt, searchErr, "project_id", projectID, "source", "search_matches", "search_active", true, "tasks_count", 0, "search_match_count", 0)
			m.traceLoadDataStage("total", totalStartedAt, searchErr, "project_count", len(projects), "column_count", len(columns), "task_count", 0)
			return loadedMsg{err: searchErr}
		}
		searchRequestedMode = result.RequestedMode
		searchEffectiveMode = result.EffectiveMode
		searchFallbackReason = result.FallbackReason
		searchEmbeddingSummary = result.EmbeddingSummary
		matches := result.Matches
		searchMatchCount = len(matches)
		actionItemSource = "search_matches"
		tasks = make([]domain.ActionItem, 0, len(matches))
		for _, match := range matches {
			if match.Project.ID == projectID {
				tasks = append(tasks, match.ActionItem)
			}
		}
	} else {
		tasks, err = m.svc.ListActionItems(context.Background(), projectID, m.showArchived)
	}
	m.traceLoadDataStage("tasks_search", tasksStartedAt, err, "project_id", projectID, "source", actionItemSource, "search_active", searchFilterActive, "tasks_count", len(tasks), "search_match_count", searchMatchCount)
	if err != nil {
		m.traceLoadDataStage("total", totalStartedAt, err, "project_count", len(projects), "column_count", len(columns), "task_count", 0)
		return loadedMsg{err: err}
	}
	actionItemNodeContracts := make(map[string]domain.NodeContractSnapshot, len(tasks))
	for _, actionItem := range tasks {
		snapshot, snapshotErr := m.svc.GetNodeContractSnapshot(context.Background(), actionItem.ID)
		switch {
		case snapshotErr == nil:
			actionItemNodeContracts[actionItem.ID] = snapshot
		case errors.Is(snapshotErr, app.ErrNotFound):
			continue
		default:
			m.traceLoadDataStage("total", totalStartedAt, snapshotErr, "project_count", len(projects), "column_count", len(columns), "task_count", len(tasks))
			return loadedMsg{err: snapshotErr}
		}
	}
	rollupStartedAt := time.Now()
	rollup, err := m.svc.GetProjectDependencyRollup(context.Background(), projectID)
	m.traceLoadDataStage(
		"rollup",
		rollupStartedAt,
		err,
		"project_id",
		projectID,
		"total_items",
		rollup.TotalItems,
		"items_with_dependencies",
		rollup.ItemsWithDependencies,
		"dependency_edges",
		rollup.DependencyEdges,
		"blocked_items",
		rollup.BlockedItems,
		"blocked_by_edges",
		rollup.BlockedByEdges,
		"unresolved_dependency_edges",
		rollup.UnresolvedDependencyEdges,
	)
	if err != nil {
		m.traceLoadDataStage("total", totalStartedAt, err, "project_count", len(projects), "column_count", len(columns), "task_count", len(tasks))
		return loadedMsg{err: err}
	}
	eventsStartedAt := time.Now()
	activityEntries := []activityEntry{}
	events := []domain.ChangeEvent{}
	events, activityErr := m.svc.ListProjectChangeEvents(context.Background(), projectID, activityLogMaxItems)
	if activityErr == nil {
		activityEntries = mapChangeEventsToActivityEntries(events)
	}
	m.traceLoadDataStage("events", eventsStartedAt, activityErr, "project_id", projectID, "events_count", len(events), "activity_entries_count", len(activityEntries))

	attentionStartedAt := time.Now()
	attentionItems := []domain.AttentionItem{}
	globalCoordinationRows := make([]globalNoticesPanelItem, 0, len(projects)+1)
	globalNotices := make([]globalNoticesPanelItem, 0)
	globalNoticesPartialProjects := map[string]struct{}{}
	globalNoticesPartialCount := 0
	markGlobalNoticesPartial := func(projectID string) {
		projectID = strings.TrimSpace(projectID)
		if projectID == "" {
			return
		}
		globalNoticesPartialProjects[projectID] = struct{}{}
		globalNoticesPartialCount = len(globalNoticesPartialProjects)
	}
	for _, project := range projects {
		projectCoordination, coordinationErr := m.loadNoticesCoordinationSummary(project.ID)
		if coordinationErr != nil {
			if project.ID == projectID {
				m.traceLoadDataStage("total", totalStartedAt, coordinationErr, "project_count", len(projects), "column_count", len(columns), "task_count", len(tasks))
				return loadedMsg{err: coordinationErr}
			}
			markGlobalNoticesPartial(project.ID)
		} else if project.ID != projectID {
			if row, ok := globalNoticesPanelItemFromCoordinationSummary(project.ID, projectDisplayName(project), projectCoordination, false); ok {
				globalCoordinationRows = append(globalCoordinationRows, row)
			}
		}

		projectAttention, attentionErr := m.svc.ListAttentionItems(context.Background(), app.ListAttentionItemsInput{
			Level: domain.LevelTupleInput{
				ProjectID: project.ID,
			},
			AllScopes:      true,
			UnresolvedOnly: true,
			Limit:          256,
		})
		if attentionErr != nil {
			if project.ID == projectID {
				m.traceLoadDataStage(
					"attention_loop",
					attentionStartedAt,
					attentionErr,
					"project_count", len(projects),
					"attention_count", len(attentionItems),
					"global_notice_count", len(globalNotices),
					"partial_project_count", globalNoticesPartialCount,
				)
				m.traceLoadDataStage("total", totalStartedAt, attentionErr, "project_count", len(projects), "column_count", len(columns), "task_count", len(tasks))
				return loadedMsg{err: attentionErr}
			}
			markGlobalNoticesPartial(project.ID)
			continue
		}
		if project.ID == projectID {
			attentionItems = append(attentionItems, projectAttention...)
		}
		for _, item := range projectAttention {
			if !m.shouldShowGlobalNoticeAttention(item) {
				continue
			}
			if project.ID == projectID {
				continue
			}
			globalNotices = append(globalNotices, globalNoticesPanelItemFromAttention(project, item))
		}
	}

	_, globalCoordinationErr := m.loadNoticesCoordinationSummary(domain.AuthRequestGlobalProjectID)
	globalAttention, globalAttentionErr := m.svc.ListAttentionItems(context.Background(), app.ListAttentionItemsInput{
		Level: domain.LevelTupleInput{
			ProjectID: domain.AuthRequestGlobalProjectID,
		},
		AllScopes:      true,
		UnresolvedOnly: true,
		Limit:          256,
	})
	if globalAttentionErr == nil {
		for _, item := range globalAttention {
			if !m.shouldShowGlobalNoticeAttention(item) {
				continue
			}
			globalNotices = append(globalNotices, globalNoticesPanelItemFromAttentionLabel(domain.AuthRequestGlobalProjectID, "All Projects", item))
		}
	}
	if globalCoordinationErr != nil {
		markGlobalNoticesPartial(domain.AuthRequestGlobalProjectID)
	}
	globalNotices = append(globalCoordinationRows, globalNotices...)
	m.traceLoadDataStage(
		"attention_loop",
		attentionStartedAt,
		nil,
		"project_count", len(projects),
		"attention_count", len(attentionItems),
		"global_notice_count", len(globalNotices),
		"partial_project_count", globalNoticesPartialCount,
	)
	requiresUserAction := 0
	for _, item := range attentionItems {
		if item.RequiresUserAction {
			requiresUserAction++
		}
	}
	coordinationStartedAt := time.Now()
	noticesCoordination, coordinationErr := m.loadNoticesCoordinationSummary(projectID)
	m.traceLoadDataStage(
		"coordination",
		coordinationStartedAt,
		coordinationErr,
		"project_id", projectID,
		"pending_requests", len(noticesCoordination.PendingRequests),
		"active_sessions", len(noticesCoordination.ActiveSessions),
		"active_leases", len(noticesCoordination.ActiveLeases),
		"open_handoffs", len(noticesCoordination.OpenHandoffs),
	)
	if coordinationErr != nil {
		m.traceLoadDataStage("total", totalStartedAt, coordinationErr, "project_count", len(projects), "column_count", len(columns), "task_count", len(tasks))
		return loadedMsg{err: coordinationErr}
	}
	m.traceLoadDataStage(
		"total",
		totalStartedAt,
		nil,
		"project_count", len(projects),
		"column_count", len(columns),
		"task_count", len(tasks),
		"activity_entries_count", len(activityEntries),
		"attention_count", len(attentionItems),
		"global_notice_count", len(globalNotices),
		"partial_project_count", globalNoticesPartialCount,
	)

	return loadedMsg{
		projects:                  projects,
		selectedProject:           projectIdx,
		columns:                   columns,
		tasks:                     tasks,
		kindDefinitions:           kindDefinitions,
		templateLibraries:         templateLibraries,
		builtinTemplateStatuses:   builtinTemplateStatuses,
		projectTemplateBinding:    projectTemplateBinding,
		actionItemNodeContracts:   actionItemNodeContracts,
		searchRequestedMode:       searchRequestedMode,
		searchEffectiveMode:       searchEffectiveMode,
		searchFallbackReason:      searchFallbackReason,
		searchEmbeddingSummary:    searchEmbeddingSummary,
		activityEntries:           activityEntries,
		attentionItems:            attentionItems,
		globalNotices:             globalNotices,
		globalNoticesPartialCount: globalNoticesPartialCount,
		noticesCoordination:       noticesCoordination,
		rollup:                    rollup,
		attentionItemsCount:       len(attentionItems),
		attentionUserActionCount:  requiresUserAction,
	}
}

// loadSearchMatches loads required data for the current operation.
func (m Model) loadSearchMatchesCmd(requestID int) tea.Cmd {
	return func() tea.Msg {
		projectID, _ := m.currentProjectID()
		result, err := m.svc.SearchActionItems(context.Background(), app.SearchActionItemsFilter{
			ProjectID:       projectID,
			Query:           m.searchQuery,
			CrossProject:    m.searchCrossProject,
			IncludeArchived: m.searchIncludeArchived,
			States:          append([]string(nil), m.searchStates...),
			Levels:          canonicalSearchLevels(m.searchLevels),
			Kinds:           append([]string(nil), m.searchKinds...),
			LabelsAny:       append([]string(nil), m.searchLabelsAny...),
			LabelsAll:       append([]string(nil), m.searchLabelsAll...),
			Mode:            m.searchMode,
			Sort:            app.SearchSortRankDesc,
			Limit:           defaultSearchResultsLimit,
			Offset:          0,
		})
		if err != nil {
			return searchResultsMsg{requestID: requestID, err: err}
		}
		return searchResultsMsg{requestID: requestID, result: result}
	}
}

// startEmbeddingsStatus opens the embeddings lifecycle inventory modal for the requested scope.
func (m *Model) startEmbeddingsStatus(global bool) tea.Cmd {
	m.embeddingsGlobal = global
	m.embeddingsIndex = 0
	m.embeddingsFilterActive = false
	m.embeddingsFilterInput.Blur()
	if m.mode != modeEmbeddingsStatus {
		m.embeddingsFilterQuery = ""
		m.embeddingsFilterInput.SetValue("")
	}
	m.mode = modeEmbeddingsStatus
	m.status = "embeddings inventory"
	return m.loadEmbeddingsStatusCmd()
}

// startEmbeddingsReindex kicks off one explicit embeddings reindex and animates an in-progress indicator.
func (m *Model) startEmbeddingsReindex(force bool) tea.Cmd {
	m.mode = modeEmbeddingsStatus
	m.embeddingsReindexInFlight = true
	m.embeddingsReindexForce = force
	if force {
		m.status = "force reindexing embeddings..."
	} else {
		m.status = "reindexing embeddings..."
	}
	return tea.Batch(m.runEmbeddingsReindexCmd(force), m.embeddingsSpinner.Tick)
}

// loadEmbeddingsStatusCmd loads embeddings lifecycle inventory rows for the active TUI scope.
func (m Model) loadEmbeddingsStatusCmd() tea.Cmd {
	return func() tea.Msg {
		projectIDs, err := m.embeddingsStatusProjectIDs()
		if err != nil {
			return embeddingsStatusLoadedMsg{err: err}
		}
		rows, err := m.svc.ListEmbeddingStates(context.Background(), app.EmbeddingListFilter{
			ProjectIDs: projectIDs,
			Limit:      defaultEmbeddingsStatusLimit,
		})
		if err != nil {
			return embeddingsStatusLoadedMsg{err: err}
		}
		summary, err := m.svc.SummarizeEmbeddingStates(context.Background(), app.EmbeddingListFilter{
			ProjectIDs: projectIDs,
		})
		if err != nil {
			return embeddingsStatusLoadedMsg{err: err}
		}
		projects, err := m.svc.ListProjects(context.Background(), true)
		if err != nil {
			return embeddingsStatusLoadedMsg{err: err}
		}
		projectsByID := make(map[string]domain.Project, len(projects))
		for _, project := range projects {
			projectsByID[strings.TrimSpace(project.ID)] = project
		}
		actionItemProjectIDs := uniqueTrimmed(projectIDs)
		if len(rows) > 0 {
			actionItemProjectIDs = actionItemProjectIDs[:0]
			for _, row := range rows {
				actionItemProjectIDs = append(actionItemProjectIDs, row.ProjectID)
			}
			actionItemProjectIDs = uniqueTrimmed(actionItemProjectIDs)
		}
		tasksByID := map[string]domain.ActionItem{}
		for _, projectID := range actionItemProjectIDs {
			tasks, err := m.svc.ListActionItems(context.Background(), projectID, true)
			if err != nil {
				return embeddingsStatusLoadedMsg{err: err}
			}
			for _, actionItem := range tasks {
				tasksByID[strings.TrimSpace(actionItem.ID)] = actionItem
			}
		}
		displayRows := make([]embeddingsStatusDisplayRow, 0, len(rows))
		for _, row := range rows {
			displayRows = append(displayRows, buildEmbeddingsStatusDisplayRow(row, projectsByID, tasksByID))
		}
		return embeddingsStatusLoadedMsg{
			projectIDs:  projectIDs,
			scopeLabel:  embeddingsStatusScopeLabel(projectIDs, projectsByID),
			rows:        rows,
			displayRows: displayRows,
			summary:     summary,
		}
	}
}

// runEmbeddingsReindexCmd enqueues embeddings work for the active TUI scope and returns the operator-visible outcome.
func (m Model) runEmbeddingsReindexCmd(force bool) tea.Cmd {
	return func() tea.Msg {
		projectIDs, err := m.embeddingsStatusProjectIDs()
		if err != nil {
			return embeddingsReindexMsg{err: err}
		}
		projectID := ""
		if !m.embeddingsGlobal {
			projectID, _ = m.currentProjectID()
		}
		result, err := m.svc.ReindexEmbeddings(context.Background(), app.ReindexEmbeddingsInput{
			ProjectID:       projectID,
			CrossProject:    m.embeddingsGlobal,
			IncludeArchived: m.embeddingsIncludeArchived,
			Force:           force,
		})
		if err != nil {
			return embeddingsReindexMsg{err: err}
		}
		result.TargetProjects = append([]string(nil), projectIDs...)
		return embeddingsReindexMsg{result: result}
	}
}

// embeddingsStatusProjectIDs resolves the current TUI embeddings scope into concrete project ids.
func (m Model) embeddingsStatusProjectIDs() ([]string, error) {
	if m.embeddingsGlobal {
		projects, err := m.svc.ListProjects(context.Background(), m.embeddingsIncludeArchived)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(projects))
		for _, project := range projects {
			out = append(out, project.ID)
		}
		return out, nil
	}
	projectID, ok := m.currentProjectID()
	if !ok {
		return []string{}, nil
	}
	return []string{projectID}, nil
}

// applyEmbeddingsDisplayFilter updates visible lifecycle rows for the active inventory query.
func (m *Model) applyEmbeddingsDisplayFilter() {
	if m == nil {
		return
	}
	query := strings.ToLower(strings.TrimSpace(m.embeddingsFilterQuery))
	if query == "" {
		m.embeddingsDisplayRows = append([]embeddingsStatusDisplayRow(nil), m.embeddingsAllDisplayRows...)
		m.embeddingsIndex = clamp(m.embeddingsIndex, 0, len(m.embeddingsDisplayRows)-1)
		return
	}
	filtered := make([]embeddingsStatusDisplayRow, 0, len(m.embeddingsAllDisplayRows))
	for _, row := range m.embeddingsAllDisplayRows {
		if strings.Contains(row.FilterLabel, query) {
			filtered = append(filtered, row)
		}
	}
	m.embeddingsDisplayRows = filtered
	m.embeddingsIndex = clamp(m.embeddingsIndex, 0, len(m.embeddingsDisplayRows)-1)
}

// selectedEmbeddingStatusRow returns the currently focused operator-visible lifecycle row.
func (m Model) selectedEmbeddingStatusRow() (embeddingsStatusDisplayRow, bool) {
	if len(m.embeddingsDisplayRows) == 0 {
		return embeddingsStatusDisplayRow{}, false
	}
	idx := clamp(m.embeddingsIndex, 0, len(m.embeddingsDisplayRows)-1)
	return m.embeddingsDisplayRows[idx], true
}

// focusEmbeddingsFilter activates inline lifecycle-row filtering.
func (m *Model) focusEmbeddingsFilter() tea.Cmd {
	if m == nil {
		return nil
	}
	m.embeddingsFilterActive = true
	m.status = "filter embeddings inventory"
	return m.embeddingsFilterInput.Focus()
}

// blurEmbeddingsFilter exits inline lifecycle-row filtering without closing the inventory modal.
func (m *Model) blurEmbeddingsFilter() {
	if m == nil {
		return
	}
	m.embeddingsFilterActive = false
	m.embeddingsFilterInput.Blur()
	m.status = "embeddings inventory"
}

// selectProjectInBoardState switches the board selection to one project id when it is already loaded in memory.
func (m *Model) selectProjectInBoardState(projectID string) bool {
	if m == nil {
		return false
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return false
	}
	for idx, project := range m.projects {
		if strings.TrimSpace(project.ID) != projectID {
			continue
		}
		m.selectedProject = idx
		return true
	}
	return false
}

// openSelectedEmbeddingStatusRow opens the currently focused lifecycle row in the closest human-facing detail view.
func (m Model) openSelectedEmbeddingStatusRow() (tea.Model, tea.Cmd) {
	row, ok := m.selectedEmbeddingStatusRow()
	if !ok {
		m.status = "no lifecycle row selected"
		return m, nil
	}
	projectID := strings.TrimSpace(row.Record.ProjectID)
	if row.HasProject {
		projectID = strings.TrimSpace(row.Project.ID)
	}
	if projectID != "" {
		m.selectProjectInBoardState(projectID)
	}
	if row.HasActionItem {
		if row.ActionItem.ArchivedAt != nil {
			m.showArchived = true
		}
		m.searchApplied = false
		m.setBoardContextForActionItem(row.ActionItem)
		m.pendingFocusActionItemID = row.ActionItem.ID
		m.pendingOpenActionItemInfoID = row.ActionItem.ID
		m.mode = modeNone
		m.status = "opening embeddings row..."
		return m, m.loadData
	}
	if row.HasProject {
		target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
			ProjectID:  row.Project.ID,
			TargetType: domain.CommentTargetTypeProject,
			TargetID:   row.Project.ID,
		})
		if err != nil {
			m.status = "project detail target invalid: " + err.Error()
			return m, nil
		}
		return m.startThread(modeNone, target, row.Project.Name, row.Project.Description, threadPanelDetails)
	}
	m.status = "selected lifecycle row has no openable subject"
	return m, nil
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

// canJumpToActivityNode reports whether one activity entry references a concrete actionItem node.
func canJumpToActivityNode(entry activityEntry) bool {
	return strings.TrimSpace(entry.ActionItemID) != ""
}

// prepareActivityJumpContext adjusts focus scope so jump targets can be selected in board view.
func (m *Model) prepareActivityJumpContext(actionItemID string) bool {
	actionItem, ok := m.actionItemByID(strings.TrimSpace(actionItemID))
	if !ok {
		return false
	}
	parentID := strings.TrimSpace(actionItem.ParentID)
	if parentID == "" {
		m.projectionRootActionItemID = ""
		m.clampSelections()
		return true
	}
	if _, ok := m.actionItemByID(parentID); !ok {
		m.projectionRootActionItemID = ""
		m.clampSelections()
		return true
	}
	return m.activateSubtreeFocus(parentID)
}

// jumpToActivityNode navigates to the actionItem referenced by the current activity-detail entry when available.
func (m Model) jumpToActivityNode() (tea.Model, tea.Cmd) {
	workItemID := strings.TrimSpace(m.activityInfoItem.ActionItemID)
	if workItemID == "" {
		m.status = "activity event has no node reference"
		return m, nil
	}
	if _, ok := m.actionItemByID(workItemID); ok {
		m.mode = modeNone
		m.noticesFocused = false
		m.prepareActivityJumpContext(workItemID)
		if m.focusActionItemByID(workItemID) {
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
		m.pendingFocusActionItemID = workItemID
		m.pendingActivityJumpActionItem = workItemID
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
		target = strings.TrimSpace(event.ActionItemID)
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
		ActionItemID: strings.TrimSpace(event.ActionItemID),
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
		return "actionItem"
	}
	label := strings.TrimSpace(strings.ToLower(metadata["item_scope"]))
	if label == "" {
		label = strings.TrimSpace(strings.ToLower(metadata["scope"]))
	}
	if label == "" {
		label = strings.TrimSpace(strings.ToLower(metadata["item_kind"]))
	}
	switch label {
	case "project", "branch", "phase", "actionItem", "subtask", "decision", "note":
		return label
	case "subphase":
		return "phase"
	default:
		return "actionItem"
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

// authConfirmFieldsActive reports whether the current confirm modal is reviewing one auth request.
func (m Model) authConfirmFieldsActive() bool {
	return strings.TrimSpace(m.pendingConfirm.AuthRequestID) != ""
}

// authConfirmScopeFieldsActive reports whether the auth confirm modal currently allows path/ttl edits.
func (m Model) authConfirmScopeFieldsActive() bool {
	return strings.TrimSpace(m.pendingConfirm.Kind) == "approve-auth-request"
}

// setConfirmFocus moves auth-confirm focus between editable fields and action buttons.
func (m *Model) setConfirmFocus(focus int) tea.Cmd {
	if m == nil {
		return nil
	}
	if focus < confirmFocusAuthDecision || focus > confirmFocusButtons {
		focus = confirmFocusButtons
	}
	if !m.authConfirmScopeFieldsActive() && focus <= confirmFocusAuthTTL {
		if focus <= confirmFocusAuthDecision {
			focus = confirmFocusAuthDecision
		} else {
			focus = confirmFocusAuthNote
		}
	}
	if !m.authConfirmFieldsActive() && focus == confirmFocusAuthDecision {
		focus = confirmFocusAuthNote
	}
	m.confirmFocus = focus
	m.confirmAuthPathInput.Blur()
	m.confirmAuthTTLInput.Blur()
	m.confirmAuthNoteInput.Blur()
	switch focus {
	case confirmFocusAuthDecision:
		return nil
	case confirmFocusAuthPath:
		return m.confirmAuthPathInput.Focus()
	case confirmFocusAuthTTL:
		return m.confirmAuthTTLInput.Focus()
	case confirmFocusAuthNote:
		return m.confirmAuthNoteInput.Focus()
	default:
		return nil
	}
}

// prepareConfirmAction snapshots editable auth-request approval values before the modal closes.
func (m Model) prepareConfirmAction() (confirmAction, error) {
	action := m.pendingConfirm
	if !m.authConfirmFieldsActive() {
		return action, nil
	}
	action.AuthRequestNote = strings.TrimSpace(m.confirmAuthNoteInput.Value())
	if m.authConfirmScopeFieldsActive() {
		if path := strings.TrimSpace(m.confirmAuthPathInput.Value()); path != "" {
			if _, err := domain.ParseAuthRequestPath(path); err != nil {
				return confirmAction{}, err
			}
			action.AuthRequestPath = path
		}
		ttlRaw := strings.TrimSpace(m.confirmAuthTTLInput.Value())
		if ttlRaw != "" {
			if _, err := time.ParseDuration(ttlRaw); err != nil {
				return confirmAction{}, fmt.Errorf("invalid auth approval ttl %q: %w", ttlRaw, err)
			}
			action.AuthRequestTTL = ttlRaw
		}
	}
	return action, nil
}

// confirmActionHints returns modal help copy for the current confirmation surface.
func confirmActionHints(authMode, scopeEditable bool) string {
	if authMode && scopeEditable {
		return "enter confirm • esc return to review • h/l switch"
	}
	if authMode {
		return "enter confirm • esc return to review • h/l switch"
	}
	return "enter apply • esc cancel • h/l switch • y confirm • n cancel"
}

// setPendingAuthRequestDecision switches the active auth-request decision without leaving the review modal.
func (m *Model) setPendingAuthRequestDecision(decision string) tea.Cmd {
	if m == nil || !m.authConfirmFieldsActive() {
		return nil
	}
	decision = strings.TrimSpace(strings.ToLower(decision))
	if decision != "approve" && decision != "deny" {
		return nil
	}
	currentNote := strings.TrimSpace(m.confirmAuthNoteInput.Value())
	m.pendingConfirm.Kind = decision + "-auth-request"
	m.pendingConfirm.Label = decision + " auth request"
	m.pendingConfirm.AuthRequestDecision = decision
	m.pendingConfirm.AuthRequestNote = currentNote
	return nil
}

// authReviewResetInputFocus clears auth-review editor focus and exits any inline editor.
func (m *Model) authReviewResetInputFocus() {
	if m == nil {
		return
	}
	m.confirmAuthTTLInput.Blur()
	m.confirmAuthNoteInput.Blur()
}

// authReviewOpenTTLStage focuses the session-ttl editor for the current auth review.
func (m *Model) authReviewOpenTTLStage() tea.Cmd {
	if m == nil {
		return nil
	}
	m.authReviewStage = authReviewStageEditTTL
	m.confirmAuthNoteInput.Blur()
	return m.confirmAuthTTLInput.Focus()
}

// authReviewOpenNoteStage focuses the approval or denial note editor for the current auth review.
func (m *Model) authReviewOpenNoteStage(stage int) tea.Cmd {
	if m == nil {
		return nil
	}
	m.authReviewStage = stage
	m.confirmAuthTTLInput.Blur()
	return m.confirmAuthNoteInput.Focus()
}

// authReviewReturnToSummary restores the summary-stage auth review screen after editing a sub-step.
func (m *Model) authReviewReturnToSummary() {
	if m == nil {
		return
	}
	m.authReviewStage = authReviewStageSummary
	m.authReviewResetInputFocus()
}

// closeAuthReview exits the auth-review surface and optionally returns to the coordination view.
func (m *Model) closeAuthReview(status string, reload bool) tea.Cmd {
	if m == nil {
		return nil
	}
	m.pendingConfirm = confirmAction{}
	m.authReviewResetInputFocus()
	m.authReviewScopePickerItems = nil
	m.authReviewScopePickerIndex = 0
	returnMode := m.authReviewReturnMode
	m.authReviewReturnMode = modeNone
	m.authReviewStage = authReviewStageSummary
	if strings.TrimSpace(status) != "" {
		m.status = status
	}
	if returnMode == modeAuthInventory {
		m.mode = modeAuthInventory
		if reload {
			return m.loadAuthInventoryCmd()
		}
		return nil
	}
	m.mode = modeNone
	if reload {
		return m.loadData
	}
	return nil
}

// authReviewApplyEditedTTL validates and stores the session-ttl override typed in the auth review.
func (m *Model) authReviewApplyEditedTTL() error {
	if m == nil {
		return nil
	}
	ttlRaw := strings.TrimSpace(m.confirmAuthTTLInput.Value())
	if ttlRaw == "" {
		ttlRaw = strings.TrimSpace(m.pendingConfirm.AuthRequestTTL)
	}
	if ttlRaw == "" {
		return fmt.Errorf("missing auth approval ttl")
	}
	if _, err := time.ParseDuration(ttlRaw); err != nil {
		return fmt.Errorf("invalid auth approval ttl %q: %w", ttlRaw, err)
	}
	m.pendingConfirm.AuthRequestTTL = ttlRaw
	m.confirmAuthTTLInput.SetValue(ttlRaw)
	m.confirmAuthTTLInput.CursorEnd()
	m.authReviewReturnToSummary()
	return nil
}

// authReviewApplyEditedNote stores the current note editor value and returns to the summary stage.
func (m *Model) authReviewApplyEditedNote(decision string) {
	if m == nil {
		return
	}
	note := strings.TrimSpace(m.confirmAuthNoteInput.Value())
	m.pendingConfirm.AuthRequestNote = note
	m.pendingConfirm.AuthRequestDecision = decision
	m.pendingConfirm.Kind = decision + "-auth-request"
	m.pendingConfirm.Label = decision + " auth request"
	m.authReviewReturnToSummary()
}

// openAuthReviewConfirm opens the generic confirm modal from auth review with the current decision snapshot.
func (m *Model) openAuthReviewConfirm() error {
	if m == nil {
		return nil
	}
	action, err := m.prepareConfirmAction()
	if err != nil {
		return err
	}
	m.pendingConfirm = action
	m.authReviewReturnToSummary()
	m.mode = modeConfirmAction
	m.confirmChoice = 0
	m.status = "confirm auth decision"
	return nil
}

// startAuthInventory opens the dedicated coordination surface and loads request/session/lease/handoff state.
func (m *Model) startAuthInventory(global bool) tea.Cmd {
	if m == nil {
		return nil
	}
	m.authInventoryGlobal = global
	m.authInventoryView = authInventoryViewLive
	m.authInventoryIndex = 0
	m.authInventoryBody.GotoTop()
	m.mode = modeAuthInventory
	m.status = "loading coordination surface"
	return m.loadAuthInventoryCmd()
}

// authInventoryCoordinationProject resolves the current project label used for project-scoped coordination rows.
func (m Model) authInventoryCoordinationProject() (string, string, bool) {
	project, ok := m.currentProject()
	if !ok {
		return "", "no project selected", false
	}
	return strings.TrimSpace(project.ID), firstNonEmptyTrimmed(projectDisplayName(project), project.ID), true
}

// authInventoryProjectScope resolves the current request/session scope into project id and display label.
func (m Model) authInventoryProjectScope() (string, bool, string) {
	if m.authInventoryGlobal {
		return "", false, "all projects"
	}
	project, ok := m.currentProject()
	if !ok {
		return "", false, "all projects"
	}
	return strings.TrimSpace(project.ID), true, firstNonEmptyTrimmed(projectDisplayName(project), project.ID)
}

// loadAuthInventoryCmd loads coordination inventory for the current request/session scope.
func (m Model) loadAuthInventoryCmd() tea.Cmd {
	projectID, projectScoped, _ := m.authInventoryProjectScope()
	coordinationProjectID, _, hasCoordinationProject := m.authInventoryCoordinationProject()
	return func() tea.Msg {
		requests, err := m.svc.ListAuthRequests(context.Background(), domain.AuthRequestListFilter{
			ProjectID: projectID,
			Limit:     0,
		})
		if err != nil {
			return authInventoryLoadedMsg{err: err}
		}
		sessions, err := m.svc.ListAuthSessions(context.Background(), app.AuthSessionFilter{
			ProjectID: projectID,
			State:     "active",
			Limit:     0,
		})
		if err != nil {
			return authInventoryLoadedMsg{err: err}
		}
		leases := make([]domain.CapabilityLease, 0)
		handoffs := make([]domain.Handoff, 0)
		if hasCoordinationProject {
			leases, err = m.svc.ListCapabilityLeases(context.Background(), app.ListCapabilityLeasesInput{
				ProjectID:      coordinationProjectID,
				ScopeType:      domain.CapabilityScopeProject,
				IncludeRevoked: true,
			})
			if err != nil {
				return authInventoryLoadedMsg{err: err}
			}
			handoffs, err = m.svc.ListHandoffs(context.Background(), app.ListHandoffsInput{
				Level: domain.LevelTupleInput{
					ProjectID: coordinationProjectID,
					ScopeType: domain.ScopeLevelProject,
				},
				Limit: 0,
			})
			if err != nil {
				return authInventoryLoadedMsg{err: err}
			}
		}
		return authInventoryLoadedMsg{
			projectScoped: projectScoped,
			projectID:     projectID,
			requests:      requests,
			sessions:      sessions,
			leases:        leases,
			handoffs:      handoffs,
		}
	}
}

// loadNoticesCoordinationSummary loads compact live coordination data for the selected project's notices panel.
func (m Model) loadNoticesCoordinationSummary(projectID string) (noticesCoordinationSummary, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" || m.svc == nil {
		return noticesCoordinationSummary{}, nil
	}

	requests, err := m.svc.ListAuthRequests(context.Background(), domain.AuthRequestListFilter{
		ProjectID: projectID,
		State:     domain.AuthRequestStatePending,
		Limit:     32,
	})
	if err != nil {
		return noticesCoordinationSummary{}, err
	}
	sessions, err := m.svc.ListAuthSessions(context.Background(), app.AuthSessionFilter{
		ProjectID: projectID,
		State:     "active",
		Limit:     32,
	})
	if err != nil {
		return noticesCoordinationSummary{}, err
	}
	leases, err := m.svc.ListCapabilityLeases(context.Background(), app.ListCapabilityLeasesInput{
		ProjectID: projectID,
	})
	if err != nil {
		return noticesCoordinationSummary{}, err
	}
	handoffs, err := m.svc.ListHandoffs(context.Background(), app.ListHandoffsInput{
		Level: domain.LevelTupleInput{
			ProjectID: projectID,
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   projectID,
		},
		Statuses: []domain.HandoffStatus{
			domain.HandoffStatusReady,
			domain.HandoffStatusWaiting,
			domain.HandoffStatusBlocked,
			domain.HandoffStatusFailed,
			domain.HandoffStatusReturned,
		},
		Limit: 32,
	})
	if err != nil {
		return noticesCoordinationSummary{}, err
	}

	now := time.Now().UTC()
	activeLeases := make([]domain.CapabilityLease, 0, len(leases))
	for _, lease := range leases {
		if !authInventoryLeaseIsLive(lease, now) {
			continue
		}
		activeLeases = append(activeLeases, lease)
	}

	return noticesCoordinationSummary{
		PendingRequests: requests,
		ActiveSessions:  sessions,
		ActiveLeases:    activeLeases,
		OpenHandoffs:    handoffs,
	}, nil
}

// authInventoryLeaseIsLive reports whether one lease should remain on the live coordination surface.
func authInventoryLeaseIsLive(lease domain.CapabilityLease, now time.Time) bool {
	return !lease.IsRevoked() && !lease.IsExpired(now)
}

// authInventoryHandoffIsOpen reports whether one handoff belongs on the live coordination surface.
func authInventoryHandoffIsOpen(handoff domain.Handoff) bool {
	return !domain.IsTerminalHandoffStatus(handoff.Status)
}

// authInventoryViewLabel returns the current user-facing coordination dashboard view label.
func (m Model) authInventoryViewLabel() string {
	switch m.authInventoryView {
	case authInventoryViewHistory:
		return "history"
	default:
		return "live"
	}
}

// authInventorySectionData returns the rows visible in the active coordination dashboard view.
func (m Model) authInventorySectionData() authInventorySectionData {
	data := authInventorySectionData{}
	now := time.Now().UTC()
	switch m.authInventoryView {
	case authInventoryViewHistory:
		data.ResolvedRequests = append(data.ResolvedRequests, m.authInventoryResolvedRequests...)
		for _, lease := range m.authInventoryLeases {
			if authInventoryLeaseIsLive(lease, now) {
				continue
			}
			data.Leases = append(data.Leases, lease)
		}
		for _, handoff := range m.authInventoryHandoffs {
			if authInventoryHandoffIsOpen(handoff) {
				continue
			}
			data.Handoffs = append(data.Handoffs, handoff)
		}
	default:
		data.PendingRequests = append(data.PendingRequests, m.authInventoryRequests...)
		data.Sessions = append(data.Sessions, m.authInventorySessions...)
		for _, lease := range m.authInventoryLeases {
			if !authInventoryLeaseIsLive(lease, now) {
				continue
			}
			data.Leases = append(data.Leases, lease)
		}
		for _, handoff := range m.authInventoryHandoffs {
			if !authInventoryHandoffIsOpen(handoff) {
				continue
			}
			data.Handoffs = append(data.Handoffs, handoff)
		}
		for _, item := range m.attentionItems {
			if !item.RequiresUserAction || domain.NormalizeAttentionState(item.State) == domain.AttentionStateResolved {
				continue
			}
			data.ActionRequiredCount++
		}
	}
	return data
}

// authInventoryItems flattens the visible coordination dashboard rows into one selectable list.
func (m Model) authInventoryItems() []authInventoryItem {
	data := m.authInventorySectionData()
	items := make([]authInventoryItem, 0, len(data.PendingRequests)+len(data.ResolvedRequests)+len(data.Sessions)+len(data.Leases)+len(data.Handoffs))
	if m.authInventoryView == authInventoryViewHistory {
		for idx := range data.ResolvedRequests {
			req := data.ResolvedRequests[idx]
			requestedLabel := firstNonEmptyTrimmed(m.authRequestPathDisplay(req.Path), req.Path)
			detail := fmt.Sprintf("requested: %s • client: %s", firstNonEmptyTrimmed(requestedLabel, "-"), firstNonEmptyTrimmed(req.ClientName, req.ClientID))
			if requester := humanActorLabel(req.RequestedByActor, req.RequestedByType); requester != "" {
				detail += " • requested by: " + requester
			}
			if approvedLabel := firstNonEmptyTrimmed(m.authRequestPathDisplay(req.ApprovedPath), req.ApprovedPath); approvedLabel != "" && approvedLabel != requestedLabel {
				detail += " • approved: " + approvedLabel
			}
			if note := strings.TrimSpace(req.ResolutionNote); note != "" {
				detail += " • note: " + truncate(note, 40)
			}
			labelName := firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID)
			if role := strings.TrimSpace(req.PrincipalRole); role != "" {
				labelName += " • " + role
			}
			items = append(items, authInventoryItem{
				ResolvedRequest: &data.ResolvedRequests[idx],
				Label:           fmt.Sprintf("[%s] %s", strings.TrimSpace(string(req.State)), labelName),
				Detail:          detail,
			})
		}
	} else {
		for idx := range data.PendingRequests {
			req := data.PendingRequests[idx]
			labelName := firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID)
			if role := strings.TrimSpace(req.PrincipalRole); role != "" {
				labelName += " • " + role
			}
			detailParts := []string{
				"scope: " + firstNonEmptyTrimmed(m.authRequestPathDisplay(req.Path), req.Path),
				"client: " + firstNonEmptyTrimmed(req.ClientName, req.ClientID),
			}
			if requester := humanActorLabel(req.RequestedByActor, req.RequestedByType); requester != "" {
				detailParts = append(detailParts, "requested by: "+requester)
			}
			if reason := strings.TrimSpace(req.Reason); reason != "" {
				detailParts = append(detailParts, "reason: "+truncate(req.Reason, 40))
			}
			if resumeClient := firstNonEmptyTrimmed(app.AuthRequestClaimClientIDFromContinuation(req.Continuation, req.ClientID), req.ClientID); resumeClient != "" {
				detailParts = append(detailParts, "resume: "+resumeClient)
			}
			if timeout := formatAuthRequestTimeout(req); timeout != "" {
				detailParts = append(detailParts, "timeout: "+timeout)
			}
			items = append(items, authInventoryItem{
				Request: &data.PendingRequests[idx],
				Label:   fmt.Sprintf("[%s] %s", strings.TrimSpace(string(req.State)), labelName),
				Detail:  strings.Join(detailParts, " • "),
			})
		}
		for idx := range data.Sessions {
			session := data.Sessions[idx]
			scopePath := strings.TrimSpace(session.ApprovedPath)
			if scopePath == "" && strings.TrimSpace(session.ProjectID) != "" {
				scopePath = "project/" + strings.TrimSpace(session.ProjectID)
			}
			scopeLabel := firstNonEmptyTrimmed(m.authRequestPathDisplay(scopePath), scopePath)
			labelName := firstNonEmptyTrimmed(session.PrincipalName, session.PrincipalID)
			if role := strings.TrimSpace(session.PrincipalRole); role != "" {
				labelName += " • " + role
			}
			items = append(items, authInventoryItem{
				Session: &data.Sessions[idx],
				Label:   fmt.Sprintf("[active] %s", labelName),
				Detail:  fmt.Sprintf("scope: %s • client: %s • expires: %s", firstNonEmptyTrimmed(scopeLabel, "-"), firstNonEmptyTrimmed(session.ClientName, session.ClientID), session.ExpiresAt.In(time.Local).Format(time.RFC3339)),
			})
		}
	}
	for idx := range data.Leases {
		lease := data.Leases[idx]
		detail := fmt.Sprintf("scope: %s • role: %s • expires: %s", firstNonEmptyTrimmed(m.authInventoryLeaseScopeLabel(lease), "-"), strings.TrimSpace(string(lease.Role)), lease.ExpiresAt.In(time.Local).Format(time.RFC3339))
		if !lease.HeartbeatAt.IsZero() {
			detail += " • heartbeat: " + lease.HeartbeatAt.In(time.Local).Format(time.RFC3339)
		}
		if lease.IsRevoked() {
			detail += " • revoked: " + truncate(strings.TrimSpace(lease.RevokedReason), 40)
		}
		items = append(items, authInventoryItem{
			Lease:  &data.Leases[idx],
			Label:  fmt.Sprintf("[%s] %s", m.authInventoryLeaseStatusLabel(lease), firstNonEmptyTrimmed(lease.AgentName, lease.InstanceID)),
			Detail: detail,
		})
	}
	for idx := range data.Handoffs {
		handoff := data.Handoffs[idx]
		detail := fmt.Sprintf("scope: %s • target: %s", firstNonEmptyTrimmed(m.authInventoryHandoffScopeLabel(handoff), "-"), firstNonEmptyTrimmed(m.authInventoryHandoffTargetLabel(handoff), "-"))
		if nextAction := strings.TrimSpace(handoff.NextAction); nextAction != "" {
			detail += " • next: " + truncate(nextAction, 40)
		}
		if len(handoff.MissingEvidence) > 0 {
			detail += " • missing: " + truncate(strings.Join(handoff.MissingEvidence, ", "), 40)
		}
		if note := strings.TrimSpace(handoff.ResolutionNote); note != "" {
			detail += " • note: " + truncate(note, 40)
		}
		items = append(items, authInventoryItem{
			Handoff: &data.Handoffs[idx],
			Label:   fmt.Sprintf("[%s] %s", strings.TrimSpace(string(handoff.Status)), firstNonEmptyTrimmed(m.authInventoryHandoffLabel(handoff), handoff.ID)),
			Detail:  detail,
		})
	}
	return items
}

// clampAuthInventoryIndex keeps the selected coordination row in range.
func (m *Model) clampAuthInventoryIndex() {
	if m == nil {
		return
	}
	items := m.authInventoryItems()
	m.authInventoryIndex = clamp(m.authInventoryIndex, 0, len(items)-1)
}

// selectedAuthInventoryItem returns the currently highlighted coordination row.
func (m Model) selectedAuthInventoryItem() (authInventoryItem, bool) {
	items := m.authInventoryItems()
	if len(items) == 0 {
		return authInventoryItem{}, false
	}
	idx := clamp(m.authInventoryIndex, 0, len(items)-1)
	return items[idx], true
}

// openCoordinationFromNotice opens coordination directly or after a project reload when a notice row targets another project.
func (m *Model) openCoordinationFromNotice(projectID string, global bool, handoffID string) tea.Cmd {
	if m == nil {
		return nil
	}
	projectID = strings.TrimSpace(projectID)
	m.pendingOpenAuthInventoryHandoffID = strings.TrimSpace(handoffID)
	if global {
		return m.startAuthInventory(true)
	}
	currentProjectID, hasCurrentProject := m.currentProjectID()
	if projectID == "" || (hasCurrentProject && projectID == currentProjectID) {
		return m.startAuthInventory(false)
	}
	m.pendingProjectID = projectID
	m.pendingOpenAuthInventory = true
	m.pendingOpenAuthInventoryGlobal = false
	m.status = "loading coordination..."
	return m.loadData
}

// authInventoryItemDetail renders a full detail modal payload for one coordination row.
func (m Model) authInventoryItemDetail(item authInventoryItem) (string, string, bool) {
	switch {
	case item.Request != nil:
		req := *item.Request
		lines := []string{
			fmt.Sprintf("principal: %s", firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID)),
			fmt.Sprintf("role: %s", firstNonEmptyTrimmed(req.PrincipalRole, "-")),
			fmt.Sprintf("scope: %s", firstNonEmptyTrimmed(m.authRequestPathDisplay(req.Path), req.Path, "-")),
			fmt.Sprintf("client: %s", firstNonEmptyTrimmed(req.ClientName, req.ClientID, "-")),
		}
		if requester := humanActorLabel(req.RequestedByActor, req.RequestedByType); requester != "" {
			lines = append(lines, fmt.Sprintf("requested by: %s", requester))
		}
		if reason := strings.TrimSpace(req.Reason); reason != "" {
			lines = append(lines, fmt.Sprintf("reason: %s", reason))
		}
		if resumeClient := firstNonEmptyTrimmed(app.AuthRequestClaimClientIDFromContinuation(req.Continuation, req.ClientID), req.ClientID); resumeClient != "" {
			lines = append(lines, fmt.Sprintf("resume client: %s", resumeClient))
		}
		if timeout := formatAuthRequestTimeout(req); timeout != "" {
			lines = append(lines, fmt.Sprintf("timeout: %s", timeout))
		}
		lines = append(lines, "", "enter on pending requests opens the full auth review flow.")
		return "Pending Request", strings.Join(lines, "\n"), true
	case item.ResolvedRequest != nil:
		req := *item.ResolvedRequest
		lines := []string{
			fmt.Sprintf("state: %s", firstNonEmptyTrimmed(string(req.State), "-")),
			fmt.Sprintf("principal: %s", firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID)),
			fmt.Sprintf("requested scope: %s", firstNonEmptyTrimmed(m.authRequestPathDisplay(req.Path), req.Path, "-")),
		}
		if approved := firstNonEmptyTrimmed(m.authRequestPathDisplay(req.ApprovedPath), req.ApprovedPath); approved != "" {
			lines = append(lines, fmt.Sprintf("approved scope: %s", approved))
		}
		lines = append(lines, fmt.Sprintf("client: %s", firstNonEmptyTrimmed(req.ClientName, req.ClientID, "-")))
		if requester := humanActorLabel(req.RequestedByActor, req.RequestedByType); requester != "" {
			lines = append(lines, fmt.Sprintf("requested by: %s", requester))
		}
		if note := strings.TrimSpace(req.ResolutionNote); note != "" {
			lines = append(lines, fmt.Sprintf("note: %s", note))
		}
		return "Resolved Request", strings.Join(lines, "\n"), true
	case item.Session != nil:
		session := *item.Session
		scopePath := strings.TrimSpace(session.ApprovedPath)
		if scopePath == "" && strings.TrimSpace(session.ProjectID) != "" {
			scopePath = "project/" + strings.TrimSpace(session.ProjectID)
		}
		lines := []string{
			fmt.Sprintf("principal: %s", firstNonEmptyTrimmed(session.PrincipalName, session.PrincipalID)),
			fmt.Sprintf("role: %s", firstNonEmptyTrimmed(session.PrincipalRole, "-")),
			fmt.Sprintf("approved scope: %s", firstNonEmptyTrimmed(m.authRequestPathDisplay(scopePath), scopePath, "-")),
			fmt.Sprintf("client: %s", firstNonEmptyTrimmed(session.ClientName, session.ClientID, "-")),
			fmt.Sprintf("session id: %s", firstNonEmptyTrimmed(session.SessionID, "-")),
			fmt.Sprintf("expires: %s", session.ExpiresAt.In(time.Local).Format(time.RFC3339)),
		}
		return "Active Session", strings.Join(lines, "\n"), true
	case item.Lease != nil:
		lease := *item.Lease
		lines := []string{
			fmt.Sprintf("status: %s", m.authInventoryLeaseStatusLabel(lease)),
			fmt.Sprintf("agent: %s", firstNonEmptyTrimmed(lease.AgentName, lease.InstanceID)),
			fmt.Sprintf("instance id: %s", firstNonEmptyTrimmed(lease.InstanceID, "-")),
			fmt.Sprintf("scope: %s", firstNonEmptyTrimmed(m.authInventoryLeaseScopeLabel(lease), "-")),
			fmt.Sprintf("role: %s", firstNonEmptyTrimmed(string(lease.Role), "-")),
			fmt.Sprintf("expires: %s", lease.ExpiresAt.In(time.Local).Format(time.RFC3339)),
		}
		if !lease.HeartbeatAt.IsZero() {
			lines = append(lines, fmt.Sprintf("heartbeat: %s", lease.HeartbeatAt.In(time.Local).Format(time.RFC3339)))
		}
		if lease.IsRevoked() {
			lines = append(lines, fmt.Sprintf("revoked reason: %s", firstNonEmptyTrimmed(strings.TrimSpace(lease.RevokedReason), "-")))
		}
		return "Capability Lease", strings.Join(lines, "\n"), true
	case item.Handoff != nil:
		handoff := *item.Handoff
		lines := []string{
			fmt.Sprintf("status: %s", firstNonEmptyTrimmed(string(handoff.Status), "-")),
			fmt.Sprintf("summary: %s", firstNonEmptyTrimmed(strings.TrimSpace(handoff.Summary), "-")),
			fmt.Sprintf("scope: %s", firstNonEmptyTrimmed(m.authInventoryHandoffScopeLabel(handoff), "-")),
			fmt.Sprintf("target: %s", firstNonEmptyTrimmed(m.authInventoryHandoffTargetLabel(handoff), "-")),
			fmt.Sprintf("source role: %s", firstNonEmptyTrimmed(strings.TrimSpace(handoff.SourceRole), "-")),
			fmt.Sprintf("target role: %s", firstNonEmptyTrimmed(strings.TrimSpace(handoff.TargetRole), "-")),
		}
		if nextAction := strings.TrimSpace(handoff.NextAction); nextAction != "" {
			lines = append(lines, fmt.Sprintf("next: %s", nextAction))
		}
		if len(handoff.MissingEvidence) > 0 {
			lines = append(lines, fmt.Sprintf("missing evidence: %s", strings.Join(handoff.MissingEvidence, ", ")))
		}
		if note := strings.TrimSpace(handoff.ResolutionNote); note != "" {
			lines = append(lines, fmt.Sprintf("resolution note: %s", note))
		}
		return "Handoff", strings.Join(lines, "\n"), true
	default:
		return "", "", false
	}
}

// coordinationDetailToneForItem maps one coordination row to the modal chrome treatment that best matches its state.
func (m Model) coordinationDetailToneForItem(item authInventoryItem) coordinationDetailTone {
	switch {
	case item.ResolvedRequest != nil:
		switch domain.NormalizeAuthRequestState(item.ResolvedRequest.State) {
		case domain.AuthRequestStateApproved:
			return coordinationDetailToneSuccess
		case domain.AuthRequestStateDenied:
			return coordinationDetailToneDanger
		case domain.AuthRequestStateExpired:
			return coordinationDetailToneWarn
		default:
			return coordinationDetailToneMuted
		}
	case item.Session != nil:
		return coordinationDetailToneActive
	case item.Lease != nil:
		switch m.authInventoryLeaseStatusLabel(*item.Lease) {
		case "active":
			return coordinationDetailToneActive
		case "expired":
			return coordinationDetailToneWarn
		default:
			return coordinationDetailToneMuted
		}
	case item.Handoff != nil:
		switch domain.NormalizeHandoffStatus(item.Handoff.Status) {
		case domain.HandoffStatusReady, domain.HandoffStatusResolved:
			return coordinationDetailToneSuccess
		case domain.HandoffStatusBlocked, domain.HandoffStatusFailed:
			return coordinationDetailToneDanger
		case domain.HandoffStatusReturned, domain.HandoffStatusSuperseded:
			return coordinationDetailToneMuted
		default:
			return coordinationDetailToneNeutral
		}
	default:
		return coordinationDetailToneNeutral
	}
}

// coordinationDetailActionsForItem builds the available coordination-detail actions for one selected row.
func (m Model) coordinationDetailActionsForItem(item authInventoryItem) []coordinationDetailAction {
	actions := []coordinationDetailAction{{Label: "close"}}
	switch {
	case item.Session != nil:
		scopePath := strings.TrimSpace(item.Session.ApprovedPath)
		if scopePath == "" && strings.TrimSpace(item.Session.ProjectID) != "" {
			scopePath = "project/" + strings.TrimSpace(item.Session.ProjectID)
		}
		actions = append(actions, coordinationDetailAction{
			Label: "revoke session",
			Confirm: confirmAction{
				Kind:                 "revoke-auth-session",
				Label:                "revoke auth session",
				AuthSessionID:        strings.TrimSpace(item.Session.SessionID),
				AuthSessionPrincipal: firstNonEmptyTrimmed(item.Session.PrincipalName, item.Session.PrincipalID),
				AuthSessionPathLabel: firstNonEmptyTrimmed(m.authRequestPathDisplay(scopePath), scopePath),
				ReturnToAuthAccess:   true,
			},
		})
	case item.Lease != nil:
		if m.authInventoryLeaseStatusLabel(*item.Lease) == "active" {
			actions = append(actions, coordinationDetailAction{
				Label: "revoke lease",
				Confirm: confirmAction{
					Kind:               "revoke-capability-lease",
					Label:              "revoke capability lease",
					LeaseInstanceID:    strings.TrimSpace(item.Lease.InstanceID),
					LeaseAgentName:     firstNonEmptyTrimmed(item.Lease.AgentName, item.Lease.InstanceID),
					LeaseScopeLabel:    firstNonEmptyTrimmed(m.authInventoryLeaseScopeLabel(*item.Lease), "-"),
					ReturnToAuthAccess: true,
				},
			})
		}
	case item.Handoff != nil:
		if !item.Handoff.IsTerminal() {
			for _, status := range []domain.HandoffStatus{
				domain.HandoffStatusReady,
				domain.HandoffStatusWaiting,
				domain.HandoffStatusBlocked,
				domain.HandoffStatusReturned,
				domain.HandoffStatusFailed,
				domain.HandoffStatusSuperseded,
				domain.HandoffStatusResolved,
			} {
				if domain.NormalizeHandoffStatus(item.Handoff.Status) == status {
					continue
				}
				actions = append(actions, coordinationDetailAction{
					Label: "mark " + string(status),
					Confirm: confirmAction{
						Kind:               "update-handoff-status",
						Label:              "update handoff",
						HandoffID:          strings.TrimSpace(item.Handoff.ID),
						HandoffSummary:     strings.TrimSpace(item.Handoff.Summary),
						HandoffStatus:      string(status),
						ReturnToAuthAccess: true,
					},
				})
			}
		}
	}
	return actions
}

// openCoordinationDetail opens one typed inspect/action modal over the coordination surface.
func (m *Model) openCoordinationDetail(item authInventoryItem, title, body string) {
	if m == nil {
		return
	}
	m.coordinationDetailItem = item
	m.coordinationDetailTitle = strings.TrimSpace(title)
	m.coordinationDetailBody = strings.TrimSpace(body)
	m.coordinationDetailTone = m.coordinationDetailToneForItem(item)
	m.coordinationDetailActions = m.coordinationDetailActionsForItem(item)
	m.coordinationDetailActionIndex = 0
	m.mode = modeCoordinationDetail
	m.status = firstNonEmptyTrimmed(m.coordinationDetailTitle, "coordination detail")
}

// closeCoordinationDetail closes the typed coordination-detail modal and returns to the coordination surface.
func (m *Model) closeCoordinationDetail() {
	if m == nil {
		return
	}
	m.coordinationDetailItem = authInventoryItem{}
	m.coordinationDetailTitle = ""
	m.coordinationDetailBody = ""
	m.coordinationDetailTone = coordinationDetailToneNeutral
	m.coordinationDetailActions = nil
	m.coordinationDetailActionIndex = 0
	m.mode = modeAuthInventory
	m.status = "coordination"
}

// coordinationDetailMoveAction moves the selected coordination-detail action within bounds.
func (m *Model) coordinationDetailMoveAction(delta int) {
	if m == nil || len(m.coordinationDetailActions) == 0 {
		return
	}
	next := clamp(m.coordinationDetailActionIndex+delta, 0, len(m.coordinationDetailActions)-1)
	m.coordinationDetailActionIndex = next
}

// selectedCoordinationDetailAction returns the currently highlighted detail action.
func (m Model) selectedCoordinationDetailAction() (coordinationDetailAction, bool) {
	if len(m.coordinationDetailActions) == 0 {
		return coordinationDetailAction{}, false
	}
	idx := clamp(m.coordinationDetailActionIndex, 0, len(m.coordinationDetailActions)-1)
	return m.coordinationDetailActions[idx], true
}

// authInventoryLeaseStatusLabel renders one capability lease status for coordination visibility.
func (m Model) authInventoryLeaseStatusLabel(lease domain.CapabilityLease) string {
	now := time.Now().UTC()
	switch {
	case lease.IsRevoked():
		return "revoked"
	case lease.IsExpired(now):
		return "expired"
	default:
		return "active"
	}
}

// authInventoryLeaseScopeLabel renders one capability lease scope label.
func (m Model) authInventoryLeaseScopeLabel(lease domain.CapabilityLease) string {
	return m.authInventoryScopeEntityLabel(lease.ProjectID, domain.ScopeLevel(lease.ScopeType), lease.ScopeID)
}

// authInventoryHandoffScopeLabel renders one handoff scope label.
func (m Model) authInventoryHandoffScopeLabel(handoff domain.Handoff) string {
	return m.authInventoryScopeEntityLabel(handoff.ProjectID, handoff.ScopeType, handoff.ScopeID)
}

// authInventoryHandoffTargetLabel renders one handoff target label.
func (m Model) authInventoryHandoffTargetLabel(handoff domain.Handoff) string {
	if strings.TrimSpace(handoff.TargetBranchID) == "" && handoff.TargetScopeType == "" && strings.TrimSpace(handoff.TargetScopeID) == "" {
		targetRole := strings.TrimSpace(handoff.TargetRole)
		if targetRole == "" {
			return "-"
		}
		return "role:" + targetRole
	}
	return m.authInventoryTargetEntityLabel(handoff.ProjectID, handoff.TargetBranchID, handoff.TargetScopeType, handoff.TargetScopeID)
}

// authInventoryHandoffLabel renders one human-readable handoff row label.
func (m Model) authInventoryHandoffLabel(handoff domain.Handoff) string {
	sourceRole := strings.TrimSpace(handoff.SourceRole)
	targetRole := strings.TrimSpace(handoff.TargetRole)
	switch {
	case sourceRole != "" && targetRole != "":
		return sourceRole + " -> " + targetRole
	case sourceRole != "":
		return sourceRole
	case targetRole != "":
		return "to " + targetRole
	case strings.TrimSpace(handoff.Summary) != "":
		return truncate(strings.TrimSpace(handoff.Summary), 40)
	default:
		return ""
	}
}

// authInventorySecondaryLabel appends one level/id disambiguator when a friendly label would otherwise hide it.
func authInventorySecondaryLabel(label string, scopeType domain.ScopeLevel, scopeID string) string {
	label = strings.TrimSpace(label)
	scopeType = domain.NormalizeScopeLevel(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	secondary := ""
	switch {
	case scopeType != "" && scopeID != "":
		secondary = string(scopeType) + ":" + scopeID
	case scopeID != "":
		secondary = scopeID
	case scopeType != "":
		secondary = string(scopeType)
	}
	if secondary == "" || strings.Contains(label, secondary) {
		return label
	}
	if label == "" {
		return secondary
	}
	return label + " [" + secondary + "]"
}

// authInventoryScopeEntityLabel renders one project-scoped lease or handoff scope using human names when available.
func (m Model) authInventoryScopeEntityLabel(projectID string, scopeType domain.ScopeLevel, scopeID string) string {
	projectLabel := m.authInventoryProjectLabel(projectID)
	scopeType = domain.NormalizeScopeLevel(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	switch {
	case scopeType == domain.ScopeLevelProject:
		return firstNonEmptyTrimmed(projectLabel, projectID)
	case scopeID == "":
		if projectLabel != "" {
			return projectLabel + " • " + strings.TrimSpace(string(scopeType))
		}
		return strings.TrimSpace(string(scopeType))
	case projectLabel != "":
		if actionItem, ok := m.actionItemByID(scopeID); ok {
			return projectLabel + " -> " + authInventorySecondaryLabel(firstNonEmptyTrimmed(actionItem.Title, scopeID), scopeType, scopeID)
		}
		return projectLabel + " -> " + strings.TrimSpace(string(scopeType)) + ":" + scopeID
	default:
		if actionItem, ok := m.actionItemByID(scopeID); ok {
			return authInventorySecondaryLabel(firstNonEmptyTrimmed(actionItem.Title, scopeID), scopeType, scopeID)
		}
		return strings.TrimSpace(string(scopeType)) + ":" + scopeID
	}
}

// authInventoryTargetEntityLabel renders one handoff target label with human names when available.
func (m Model) authInventoryTargetEntityLabel(projectID, branchID string, targetType domain.ScopeLevel, targetID string) string {
	branchLabel := strings.TrimSpace(branchID)
	if actionItem, ok := m.actionItemByID(branchID); ok {
		branchLabel = authInventorySecondaryLabel(firstNonEmptyTrimmed(actionItem.Title, branchID), domain.ScopeLevelBranch, branchID)
	}
	targetType = domain.NormalizeScopeLevel(targetType)
	targetID = strings.TrimSpace(targetID)
	targetLabel := ""
	switch {
	case targetType == "" && targetID == "":
		targetLabel = ""
	case targetType == domain.ScopeLevelProject:
		targetLabel = firstNonEmptyTrimmed(m.authInventoryProjectLabel(targetID), targetID)
	case targetType == "" && targetID != "":
		if actionItem, ok := m.actionItemByID(targetID); ok {
			targetLabel = authInventorySecondaryLabel(firstNonEmptyTrimmed(actionItem.Title, targetID), targetType, targetID)
		} else {
			targetLabel = targetID
		}
	case targetID == "":
		targetLabel = string(targetType)
	default:
		if actionItem, ok := m.actionItemByID(targetID); ok {
			targetLabel = authInventorySecondaryLabel(firstNonEmptyTrimmed(actionItem.Title, targetID), targetType, targetID)
		} else {
			targetLabel = strings.TrimSpace(string(targetType)) + ":" + targetID
		}
	}
	if branchLabel == "" {
		return firstNonEmptyTrimmed(targetLabel, "-")
	}
	if targetType == domain.ScopeLevelProject {
		return firstNonEmptyTrimmed(targetLabel, branchLabel)
	}
	if targetLabel == "" || targetLabel == branchLabel {
		return branchLabel
	}
	return branchLabel + " -> " + targetLabel
}

// authInventoryProjectLabel renders one project identifier using the loaded project name when possible.
func (m Model) authInventoryProjectLabel(projectID string) string {
	projectID = strings.TrimSpace(projectID)
	if project, ok := m.projectByID(projectID); ok {
		return firstNonEmptyTrimmed(projectDisplayName(project), projectID)
	}
	return projectID
}

// authInventoryScopeLabel renders the current request/session scope label.
func (m Model) authInventoryScopeLabel() string {
	_, _, label := m.authInventoryProjectScope()
	return label
}

// authInventoryScopeLabelForSurface returns the request/session scope label for the coordination surface chrome.
func (m Model) authInventoryScopeLabelForSurface() string {
	requestSessionScopeLabel := "global (all projects)"
	if !m.authInventoryGlobal {
		if _, ok := m.currentProject(); ok {
			requestSessionScopeLabel = "project scope (" + firstNonEmptyTrimmed(m.authInventoryScopeLabel(), "no project selected") + ")"
		} else {
			requestSessionScopeLabel = "all projects (no project selected)"
		}
	}
	return requestSessionScopeLabel
}

// authInventoryMoveSelection moves the coordination cursor across selectable rows.
func (m *Model) authInventoryMoveSelection(delta int) {
	if m == nil {
		return
	}
	items := m.authInventoryItems()
	if len(items) == 0 || delta == 0 {
		return
	}
	m.authInventoryIndex = wrapIndex(m.authInventoryIndex, delta, len(items))
	m.syncAuthInventoryViewport()
}

// authInventoryBodyLines renders the coordination body and returns the selected row and section offsets for viewport alignment.
func (m Model) authInventoryBodyLines(contentWidth int, hintStyle, accentStyle lipgloss.Style) ([]string, int, int) {
	data := m.authInventorySectionData()
	selectedIndex := -1
	selectedLine := -1
	selectedSectionLine := -1
	currentSectionLine := -1
	if items := m.authInventoryItems(); len(items) > 0 {
		selectedIndex = clamp(m.authInventoryIndex, 0, len(items)-1)
	}
	lines := []string{}
	switch m.authInventoryView {
	case authInventoryViewHistory:
		lines = append(
			lines,
			fmt.Sprintf("resolved requests: %d", len(data.ResolvedRequests)),
			fmt.Sprintf("ended leases: %d", len(data.Leases)),
			fmt.Sprintf("closed handoffs: %d", len(data.Handoffs)),
			"",
		)
		if len(data.ResolvedRequests) == 0 && len(data.Leases) == 0 && len(data.Handoffs) == 0 {
			lines = append(lines, hintStyle.Render("no coordination history is visible in this scope"))
			return lines, selectedLine, selectedSectionLine
		}
	default:
		lines = append(
			lines,
			fmt.Sprintf("action required: %d", data.ActionRequiredCount),
			fmt.Sprintf("pending requests: %d", len(data.PendingRequests)),
			fmt.Sprintf("active sessions: %d", len(data.Sessions)),
			fmt.Sprintf("active leases: %d", len(data.Leases)),
			fmt.Sprintf("open handoffs: %d", len(data.Handoffs)),
			"",
		)
		if data.ActionRequiredCount == 0 && len(data.PendingRequests) == 0 && len(data.Sessions) == 0 && len(data.Leases) == 0 && len(data.Handoffs) == 0 {
			lines = append(lines, hintStyle.Render("no live coordination state is visible in this scope"))
			return lines, selectedLine, selectedSectionLine
		}
	}

	displayIndex := 0
	appendSelectable := func(label, detail string, muted bool) {
		isSelected := displayIndex == selectedIndex
		if isSelected {
			selectedLine = len(lines)
			if selectedSectionLine < 0 {
				selectedSectionLine = currentSectionLine
			}
		}
		line := "  "
		switch {
		case isSelected && muted:
			line += accentStyle.Render(label)
		case isSelected:
			line += accentStyle.Render(label)
		case muted:
			line += hintStyle.Render(label)
		default:
			line += label
		}
		lines = append(lines, line)
		if detail != "" {
			lines = append(lines, hintStyle.Render("    "+detail))
		}
		displayIndex++
	}

	if len(data.PendingRequests) > 0 {
		currentSectionLine = len(lines)
		lines = append(lines, accentStyle.Render("pending requests"))
		for idx := range data.PendingRequests {
			request := data.PendingRequests[idx]
			labelName := firstNonEmptyTrimmed(request.PrincipalName, request.PrincipalID)
			if role := strings.TrimSpace(request.PrincipalRole); role != "" {
				labelName += " • " + role
			}
			detailParts := []string{
				"scope: " + firstNonEmptyTrimmed(m.authRequestPathDisplay(request.Path), request.Path),
				"client: " + firstNonEmptyTrimmed(request.ClientName, request.ClientID),
			}
			if requestedBy := humanActorLabel(request.RequestedByActor, request.RequestedByType); requestedBy != "" {
				detailParts = append(detailParts, "requested by: "+requestedBy)
			}
			if reason := strings.TrimSpace(request.Reason); reason != "" {
				detailParts = append(detailParts, "reason: "+truncate(reason, 40))
			}
			if resumeClient := firstNonEmptyTrimmed(app.AuthRequestClaimClientIDFromContinuation(request.Continuation, request.ClientID), request.ClientID); resumeClient != "" {
				detailParts = append(detailParts, "resume: "+resumeClient)
			}
			if timeout := formatAuthRequestTimeout(request); timeout != "" {
				detailParts = append(detailParts, "timeout: "+timeout)
			}
			appendSelectable(fmt.Sprintf("[pending] %s", labelName), strings.Join(detailParts, " • "), false)
		}
	}
	if len(data.ResolvedRequests) > 0 {
		lines = append(lines, "")
		currentSectionLine = len(lines)
		lines = append(lines, accentStyle.Render("resolved requests"))
		for idx := range data.ResolvedRequests {
			request := data.ResolvedRequests[idx]
			stateLabel := strings.TrimSpace(string(request.State))
			labelName := firstNonEmptyTrimmed(request.PrincipalName, request.PrincipalID)
			if role := strings.TrimSpace(request.PrincipalRole); role != "" {
				labelName += " • " + role
			}
			label := fmt.Sprintf("[%s] %s", stateLabel, labelName)
			requestedLabel := firstNonEmptyTrimmed(m.authRequestPathDisplay(request.Path), request.Path)
			detail := fmt.Sprintf("requested: %s • client: %s", requestedLabel, firstNonEmptyTrimmed(request.ClientName, request.ClientID))
			if requester := humanActorLabel(request.RequestedByActor, request.RequestedByType); requester != "" {
				detail += " • requested by: " + requester
			}
			if approvedLabel := firstNonEmptyTrimmed(m.authRequestPathDisplay(request.ApprovedPath), request.ApprovedPath); approvedLabel != "" && approvedLabel != requestedLabel {
				detail += " • approved: " + approvedLabel
			}
			if note := strings.TrimSpace(request.ResolutionNote); note != "" {
				detail += " • note: " + truncate(note, max(24, contentWidth-18))
			}
			appendSelectable(label, detail, true)
		}
	}
	if len(data.Sessions) > 0 {
		lines = append(lines, "")
		currentSectionLine = len(lines)
		lines = append(lines, accentStyle.Render("active sessions"))
		for idx := range data.Sessions {
			session := data.Sessions[idx]
			scopePath := strings.TrimSpace(session.ApprovedPath)
			if scopePath == "" && strings.TrimSpace(session.ProjectID) != "" {
				scopePath = "project/" + strings.TrimSpace(session.ProjectID)
			}
			roleLabel := strings.TrimSpace(session.PrincipalRole)
			if roleLabel != "" {
				roleLabel = " • role: " + roleLabel
			}
			labelName := firstNonEmptyTrimmed(session.PrincipalName, session.PrincipalID)
			if role := strings.TrimSpace(session.PrincipalRole); role != "" {
				labelName += " • " + role
			}
			appendSelectable(
				fmt.Sprintf("[active] %s", labelName),
				fmt.Sprintf(
					"scope: %s • client: %s%s • expires: %s",
					firstNonEmptyTrimmed(m.authRequestPathDisplay(scopePath), scopePath),
					firstNonEmptyTrimmed(session.ClientName, session.ClientID),
					roleLabel,
					session.ExpiresAt.In(time.Local).Format(time.RFC3339),
				),
				false,
			)
		}
	}
	if len(data.Leases) > 0 {
		leaseSectionTitle := "active leases"
		if m.authInventoryView == authInventoryViewHistory {
			leaseSectionTitle = "ended leases"
		}
		lines = append(lines, "")
		currentSectionLine = len(lines)
		lines = append(lines, accentStyle.Render(leaseSectionTitle))
		for idx := range data.Leases {
			lease := data.Leases[idx]
			detail := fmt.Sprintf(
				"scope: %s • role: %s • expires: %s",
				firstNonEmptyTrimmed(m.authInventoryLeaseScopeLabel(lease), "-"),
				strings.TrimSpace(string(lease.Role)),
				lease.ExpiresAt.In(time.Local).Format(time.RFC3339),
			)
			if !lease.HeartbeatAt.IsZero() {
				detail += " • heartbeat: " + lease.HeartbeatAt.In(time.Local).Format(time.RFC3339)
			}
			if lease.IsRevoked() {
				detail += " • revoked: " + truncate(strings.TrimSpace(lease.RevokedReason), 40)
			}
			appendSelectable(
				fmt.Sprintf("[%s] %s", m.authInventoryLeaseStatusLabel(lease), firstNonEmptyTrimmed(lease.AgentName, lease.InstanceID)),
				detail,
				false,
			)
		}
	}
	if len(data.Handoffs) > 0 {
		handoffSectionTitle := "open handoffs"
		if m.authInventoryView == authInventoryViewHistory {
			handoffSectionTitle = "closed handoffs"
		}
		lines = append(lines, "")
		currentSectionLine = len(lines)
		lines = append(lines, accentStyle.Render(handoffSectionTitle))
		for idx := range data.Handoffs {
			handoff := data.Handoffs[idx]
			detail := fmt.Sprintf(
				"scope: %s • target: %s",
				firstNonEmptyTrimmed(m.authInventoryHandoffScopeLabel(handoff), "-"),
				firstNonEmptyTrimmed(m.authInventoryHandoffTargetLabel(handoff), "-"),
			)
			if nextAction := strings.TrimSpace(handoff.NextAction); nextAction != "" {
				detail += " • next: " + truncate(nextAction, 40)
			}
			if len(handoff.MissingEvidence) > 0 {
				detail += " • missing: " + truncate(strings.Join(handoff.MissingEvidence, ", "), 40)
			}
			if note := strings.TrimSpace(handoff.ResolutionNote); note != "" {
				detail += " • note: " + truncate(note, 40)
			}
			appendSelectable(
				fmt.Sprintf("[%s] %s", strings.TrimSpace(string(handoff.Status)), firstNonEmptyTrimmed(m.authInventoryHandoffLabel(handoff), handoff.ID)),
				detail,
				false,
			)
		}
	}
	if selected, ok := m.selectedAuthInventoryItem(); ok && selected.ResolvedRequest != nil {
		request := selected.ResolvedRequest
		lines = append(lines, "", accentStyle.Render("selected resolved request"))
		lines = append(lines, fmt.Sprintf("state: %s", strings.TrimSpace(string(request.State))))
		lines = append(lines, fmt.Sprintf("principal: %s", firstNonEmptyTrimmed(request.PrincipalName, request.PrincipalID)))
		if role := strings.TrimSpace(request.PrincipalRole); role != "" {
			lines = append(lines, fmt.Sprintf("role: %s", role))
		}
		if requester := humanActorLabel(request.RequestedByActor, request.RequestedByType); requester != "" {
			lines = append(lines, fmt.Sprintf("requested by: %s", requester))
		}
		lines = append(lines, fmt.Sprintf("client: %s", firstNonEmptyTrimmed(request.ClientName, request.ClientID)))
		lines = append(lines, fmt.Sprintf("requested scope: %s", firstNonEmptyTrimmed(m.authRequestPathDisplay(request.Path), request.Path)))
		if approvedLabel := firstNonEmptyTrimmed(m.authRequestPathDisplay(request.ApprovedPath), request.ApprovedPath); approvedLabel != "" {
			lines = append(lines, fmt.Sprintf("approved scope: %s", approvedLabel))
		}
		if note := strings.TrimSpace(request.ResolutionNote); note != "" {
			lines = append(lines, "note:", note)
		}
	}
	return lines, wrappedLineOffset(lines, selectedLine, contentWidth), wrappedLineOffset(lines, selectedSectionLine, contentWidth)
}

// syncAuthInventoryViewport refreshes the coordination viewport dimensions and keeps the selected row visible.
func (m *Model) syncAuthInventoryViewport() {
	if m == nil || (m.mode != modeAuthInventory && m.mode != modeCoordinationDetail) {
		return
	}
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	requestSessionScopeLabel := m.authInventoryScopeLabelForSurface()
	status := m.authInventoryViewLabel()
	if scroll := fullPageScrollStatus(m.authInventoryBody); scroll != "" {
		status += " • " + scroll
	}
	metrics := m.fullPageSurfaceMetrics(
		accent,
		muted,
		dim,
		actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())),
		"Coordination",
		requestSessionScopeLabel,
		status,
	)
	hintStyle := lipgloss.NewStyle().Foreground(muted)
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	bodyLines, focusLine, sectionLine := m.authInventoryBodyLines(metrics.contentWidth, hintStyle, accentStyle)
	prevYOffset := m.authInventoryBody.YOffset()
	m.authInventoryBody.SetWidth(metrics.contentWidth)
	m.authInventoryBody.SetHeight(max(1, metrics.bodyHeight))
	m.authInventoryBody.SetContent(strings.Join(bodyLines, "\n"))
	m.authInventoryBody.SetYOffset(prevYOffset)
	ensureViewportRangeVisible(&m.authInventoryBody, sectionLine, focusLine)
}

// beginSelectedAuthSessionRevoke opens the dedicated full-screen revoke surface
// for the currently selected active session.
func (m Model) beginSelectedAuthSessionRevoke() (tea.Model, tea.Cmd, bool) {
	item, ok := m.selectedAuthInventoryItem()
	if !ok || item.Session == nil {
		m.status = "select an active session to revoke"
		return m, nil, false
	}
	scopePath := strings.TrimSpace(item.Session.ApprovedPath)
	if scopePath == "" && strings.TrimSpace(item.Session.ProjectID) != "" {
		scopePath = "project/" + strings.TrimSpace(item.Session.ProjectID)
	}
	scopeLabel := firstNonEmptyTrimmed(m.authRequestPathDisplay(scopePath), scopePath)
	m.mode = modeAuthSessionRevoke
	m.pendingConfirm = confirmAction{
		Kind:                 "revoke-auth-session",
		Label:                "revoke auth session",
		AuthSessionID:        strings.TrimSpace(item.Session.SessionID),
		AuthSessionPrincipal: firstNonEmptyTrimmed(item.Session.PrincipalName, item.Session.PrincipalID),
		AuthSessionPathLabel: scopeLabel,
		ReturnToAuthAccess:   true,
	}
	m.status = "review session revoke"
	return m, nil, true
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
	m.searchLoading = false
	m.searchOpeningResult = false
	m.searchActiveRequestID = 0
	if m.searchDefaultMode == "" {
		m.searchDefaultMode = app.SearchModeHybrid
	}
	if m.searchMode == "" {
		m.searchMode = m.searchDefaultMode
	}
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
	backMode := m.mode
	actions := m.quickActionsForMode(backMode)
	if len(actions) == 0 {
		switch backMode {
		case modeAddActionItem, modeEditActionItem:
			m.status = "no quick actions for this field"
		case modeActionItemInfo:
			m.status = "no quick actions for this actionItem"
		default:
			m.status = "no quick actions"
		}
		return nil
	}
	m.mode = modeQuickActions
	m.quickActionBackMode = backMode
	traceActionItemScreenAction("quick_actions", "open", "back_mode", modeKey(backMode), "title", m.quickActionsTitle())
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
	m.actionItemInfoBody.SetYOffset(0)
	m.actionItemInfoBody.SetContent("")
	m.projectFormInputs = []textinput.Model{
		newModalInput("", "project name", "", 120),
		newModalInput("", "enter opens markdown description editor", "", 240),
		newModalInput("", "enter opens project-kind picker", "", 120),
		newModalInput("", "owner/team", "", 120),
		newModalInput("", "icon / emoji", "", 64),
		newModalInput("", "accent color (e.g. 62)", "", 32),
		newModalInput("", "https://...", "", 200),
		newModalInput("", "csv tags", "", 200),
		newModalInput("", "enter opens approved template-library picker", "", 160),
		newModalInput("", "project root path (optional)", "", 512),
	}
	m.editingProjectID = ""
	m.projectFormDescription = ""
	if project != nil {
		m.projectFormInputs = append(m.projectFormInputs, newModalInput("", "enter opens project comments thread", "", 160))
		m.mode = modeEditProject
		m.status = "edit project"
		m.editingProjectID = project.ID
		m.projectFormInputs[projectFieldName].SetValue(project.Name)
		m.projectFormDescription = project.Description
		m.projectFormInputs[projectFieldKind].SetValue(string(project.Kind))
		m.projectFormInputs[projectFieldOwner].SetValue(firstNonEmptyTrimmed(project.Metadata.Owner, m.identityDisplayName))
		m.projectFormInputs[projectFieldIcon].SetValue(project.Metadata.Icon)
		m.projectFormInputs[projectFieldColor].SetValue(project.Metadata.Color)
		m.projectFormInputs[projectFieldHomepage].SetValue(project.Metadata.Homepage)
		if len(project.Metadata.Tags) > 0 {
			m.projectFormInputs[projectFieldTags].SetValue(strings.Join(project.Metadata.Tags, ","))
		}
		if slug := strings.TrimSpace(strings.ToLower(project.Slug)); slug != "" {
			m.projectFormInputs[projectFieldRootPath].SetValue(strings.TrimSpace(m.projectRoots[slug]))
		}
		if binding, ok := m.activeProjectTemplateBinding(project.ID); ok {
			m.projectFormInputs[projectFieldTemplateLibrary].SetValue(binding.LibraryID)
		}
	} else {
		m.mode = modeAddProject
		m.status = "new project"
		m.projectFormInputs[projectFieldKind].SetValue(string(domain.DefaultProjectKind))
		m.projectFormInputs[projectFieldOwner].SetValue(strings.TrimSpace(m.identityDisplayName))
	}
	m.syncProjectFormDescriptionDisplay()
	return m.focusProjectFormField(0)
}

// clearProjectTemplateMigrationReview resets staged TUI migration-review state.
func (m *Model) clearProjectTemplateMigrationReview() {
	if m == nil {
		return
	}
	m.templateMigrationReviewPreview = nil
	m.templateMigrationReviewDraft = nil
	m.templateMigrationReviewLoading = false
	m.templateMigrationReviewIndex = 0
	m.templateMigrationReviewPicked = map[string]struct{}{}
}

// resetProjectFormState clears staged project-form state after save/cancel.
func (m *Model) resetProjectFormState() {
	if m == nil {
		return
	}
	m.projectFormInputs = nil
	m.projectFormFocus = 0
	m.projectFormDescription = ""
	m.editingProjectID = ""
}

// startProjectTemplateMigrationReview loads one drift preview before finalizing an edit-project reapply.
func (m *Model) startProjectTemplateMigrationReview(draft pendingProjectTemplateReview) tea.Cmd {
	if m == nil {
		return nil
	}
	m.mode = modeTemplateMigrationReview
	m.templateMigrationReviewLoading = true
	m.templateMigrationReviewDraft = &draft
	m.templateMigrationReviewPreview = nil
	m.templateMigrationReviewIndex = 0
	m.templateMigrationReviewPicked = map[string]struct{}{}
	m.status = "loading template reapply review..."
	return func() tea.Msg {
		preview, err := m.svc.GetProjectTemplateReapplyPreview(context.Background(), draft.ProjectID)
		return projectTemplateReviewLoadedMsg{
			draft:   draft,
			preview: preview,
			err:     err,
		}
	}
}

// templateMigrationReviewCandidates returns the staged preview candidates for the active review surface.
func (m Model) templateMigrationReviewCandidates() []domain.ProjectTemplateMigrationCandidate {
	if m.templateMigrationReviewPreview == nil {
		return nil
	}
	return m.templateMigrationReviewPreview.MigrationCandidates
}

// selectedTemplateMigrationReviewCandidate returns the currently highlighted migration-review candidate.
func (m Model) selectedTemplateMigrationReviewCandidate() (domain.ProjectTemplateMigrationCandidate, bool) {
	candidates := m.templateMigrationReviewCandidates()
	if len(candidates) == 0 {
		return domain.ProjectTemplateMigrationCandidate{}, false
	}
	idx := clamp(m.templateMigrationReviewIndex, 0, len(candidates)-1)
	return candidates[idx], true
}

// templateMigrationSelectionIDs returns selected eligible candidate ids in stable preview order.
func (m Model) templateMigrationSelectionIDs() []string {
	candidates := m.templateMigrationReviewCandidates()
	if len(candidates) == 0 || len(m.templateMigrationReviewPicked) == 0 {
		return nil
	}
	out := make([]string, 0, len(m.templateMigrationReviewPicked))
	for _, candidate := range candidates {
		actionItemID := strings.TrimSpace(candidate.ActionItemID)
		if actionItemID == "" {
			continue
		}
		if _, ok := m.templateMigrationReviewPicked[actionItemID]; ok {
			out = append(out, actionItemID)
		}
	}
	return out
}

// toggleTemplateMigrationSelection toggles one eligible candidate in the staged migration-review selection.
func (m *Model) toggleTemplateMigrationSelection(actionItemID string) {
	if m == nil {
		return
	}
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return
	}
	if m.templateMigrationReviewPicked == nil {
		m.templateMigrationReviewPicked = map[string]struct{}{}
	}
	if _, ok := m.templateMigrationReviewPicked[actionItemID]; ok {
		delete(m.templateMigrationReviewPicked, actionItemID)
		return
	}
	m.templateMigrationReviewPicked[actionItemID] = struct{}{}
}

// applyProjectTemplateReviewDecision finalizes one staged project edit after the operator reviews template drift.
func (m Model) applyProjectTemplateReviewDecision(approveAll, approveSelected bool) (tea.Model, tea.Cmd) {
	draft := m.templateMigrationReviewDraft
	if draft == nil {
		m.status = "template reapply review unavailable"
		return m, nil
	}
	selectedActionItemIDs := m.templateMigrationSelectionIDs()
	if approveSelected && len(selectedActionItemIDs) == 0 {
		m.status = "select one or more eligible nodes, approve all, or skip"
		return m, nil
	}
	applyMigrationApproval := approveAll || approveSelected
	approvalInput := app.ApproveProjectTemplateMigrationsInput{
		ProjectID:      draft.ProjectID,
		ApproveAll:     approveAll,
		ActionItemIDs:  selectedActionItemIDs,
		ApprovedBy:     m.threadActorID(),
		ApprovedByName: m.threadActorName(),
		ApprovedByType: m.threadActorType(),
	}
	staged := *draft
	m.mode = modeNone
	m.resetProjectFormState()
	m.clearProjectTemplateMigrationReview()
	m.status = "saving project..."
	return m, func() tea.Msg {
		project, err := m.svc.UpdateProject(context.Background(), app.UpdateProjectInput{
			ProjectID:     staged.ProjectID,
			Name:          staged.Name,
			Description:   staged.Description,
			Kind:          staged.Kind,
			Metadata:      staged.Metadata,
			UpdatedBy:     m.threadActorID(),
			UpdatedByName: m.threadActorName(),
			UpdatedType:   m.threadActorType(),
		})
		if err != nil {
			return actionMsg{err: err}
		}
		appliedCount := 0
		if applyMigrationApproval {
			result, err := m.svc.ApproveProjectTemplateMigrations(context.Background(), approvalInput)
			if err != nil {
				return actionMsg{err: err}
			}
			appliedCount = result.AppliedCount
		}
		switch {
		case staged.TemplateLibraryID == "" && staged.CurrentTemplateLibraryID != "":
			if err := m.svc.UnbindProjectTemplateLibrary(context.Background(), app.UnbindProjectTemplateLibraryInput{
				ProjectID: project.ID,
			}); err != nil {
				return actionMsg{err: err}
			}
		case staged.TemplateLibraryID != "" && staged.TemplateLibraryID != staged.CurrentTemplateLibraryID:
			if _, err := m.svc.BindProjectTemplateLibrary(context.Background(), app.BindProjectTemplateLibraryInput{
				ProjectID:        project.ID,
				LibraryID:        staged.TemplateLibraryID,
				BoundByActorID:   m.threadActorID(),
				BoundByActorName: m.threadActorName(),
				BoundByActorType: m.threadActorType(),
			}); err != nil {
				return actionMsg{err: err}
			}
		case staged.TemplateLibraryID != "":
			if _, err := m.svc.BindProjectTemplateLibrary(context.Background(), app.BindProjectTemplateLibraryInput{
				ProjectID:        project.ID,
				LibraryID:        staged.TemplateLibraryID,
				BoundByActorID:   m.threadActorID(),
				BoundByActorName: m.threadActorName(),
				BoundByActorType: m.threadActorType(),
			}); err != nil {
				return actionMsg{err: err}
			}
		}
		if m.saveProjectRoot != nil {
			if err := m.saveProjectRoot(project.Slug, staged.RootPath); err != nil {
				return actionMsg{err: err}
			}
		}
		status := "project updated"
		switch {
		case approveAll:
			status = fmt.Sprintf("project updated • approved all eligible migrations (%d applied)", appliedCount)
		case approveSelected:
			status = fmt.Sprintf("project updated • approved %d template migrations", appliedCount)
		case staged.TemplateLibraryID != "" && staged.TemplateLibraryID == staged.CurrentTemplateLibraryID:
			status = "project updated • existing-node migrations skipped"
		}
		return actionMsg{
			status:          status,
			reload:          true,
			projectID:       project.ID,
			projectRootSlug: project.Slug,
			projectRootPath: staged.RootPath,
		}
	}
}

// startActionItemForm starts actionItem form.
func (m *Model) startActionItemForm(actionItem *domain.ActionItem) tea.Cmd {
	m.formFocus = 0
	m.actionItemInfoBody.SetYOffset(0)
	m.actionItemInfoBody.SetContent("")
	m.priorityIdx = 1
	m.duePicker = 0
	m.pickerBack = modeNone
	m.input = ""
	m.actionItemFormParentID = ""
	m.actionItemFormKind = domain.KindActionItem
	m.actionItemFormScope = domain.KindAppliesToActionItem
	m.actionItemFormResourceRefs = nil
	m.actionItemFormSubactionItemCursor = 0
	m.actionItemFormResourceCursor = 0
	m.actionItemFormResourceEditIndex = -1
	m.actionItemFormBackMode = modeNone
	m.actionItemFormBackActionItemID = ""
	m.actionItemFormBackChildID = ""
	m.formInputs = []textinput.Model{
		newModalInput("", "actionItem title (required)", "", 120),
		newModalInput("", "enter opens markdown description editor", "", 240),
		newModalInput("", "low | medium | high", "", 16),
		newModalInput("", "YYYY-MM-DD[THH:MM] or -", "", 32),
		newModalInput("", "csv labels", "", 160),
		newModalInput("", "csv actionItem", "", 240),
		newModalInput("", "csv actionItem", "", 240),
		newModalInput("", "why blocked? (optional)", "", 240),
		newModalInput("", "objective (optional)", "", 400),
		newModalInput("", "acceptance criteria (optional)", "", 400),
		newModalInput("", "validation plan (optional)", "", 400),
		newModalInput("", "risk notes (optional)", "", 400),
	}
	m.formInputs[actionItemFieldPriority].SetValue(string(priorityOptions[m.priorityIdx]))
	m.actionItemFormDescription = ""
	m.initActionItemFormMarkdownDrafts()
	if actionItem != nil {
		m.actionItemFormParentID = actionItem.ParentID
		m.actionItemFormKind = actionItem.Kind
		m.actionItemFormScope = actionItem.Scope
		m.formInputs[actionItemFieldTitle].SetValue(actionItem.Title)
		m.actionItemFormDescription = actionItem.Description
		m.priorityIdx = priorityIndex(actionItem.Priority)
		m.formInputs[actionItemFieldPriority].SetValue(string(priorityOptions[m.priorityIdx]))
		if actionItem.DueAt != nil {
			m.formInputs[actionItemFieldDue].SetValue(formatDueValue(actionItem.DueAt))
		}
		if len(actionItem.Labels) > 0 {
			m.formInputs[actionItemFieldLabels].SetValue(strings.Join(actionItem.Labels, ","))
		}
		if len(actionItem.Metadata.DependsOn) > 0 {
			m.formInputs[actionItemFieldDependsOn].SetValue(strings.Join(actionItem.Metadata.DependsOn, ","))
		}
		if len(actionItem.Metadata.BlockedBy) > 0 {
			m.formInputs[actionItemFieldBlockedBy].SetValue(strings.Join(actionItem.Metadata.BlockedBy, ","))
		}
		if blockedReason := strings.TrimSpace(actionItem.Metadata.BlockedReason); blockedReason != "" {
			m.setActionItemFormMarkdownDraft(actionItemFieldBlockedReason, blockedReason, false)
		}
		if objective := strings.TrimSpace(actionItem.Metadata.Objective); objective != "" {
			m.setActionItemFormMarkdownDraft(actionItemFieldObjective, objective, false)
		}
		if acceptanceCriteria := strings.TrimSpace(actionItem.Metadata.AcceptanceCriteria); acceptanceCriteria != "" {
			m.setActionItemFormMarkdownDraft(actionItemFieldAcceptanceCriteria, acceptanceCriteria, false)
		}
		if validationPlan := strings.TrimSpace(actionItem.Metadata.ValidationPlan); validationPlan != "" {
			m.setActionItemFormMarkdownDraft(actionItemFieldValidationPlan, validationPlan, false)
		}
		if riskNotes := strings.TrimSpace(actionItem.Metadata.RiskNotes); riskNotes != "" {
			m.setActionItemFormMarkdownDraft(actionItemFieldRiskNotes, riskNotes, false)
		}
		m.actionItemFormResourceRefs = append([]domain.ResourceRef(nil), actionItem.Metadata.ResourceRefs...)
		m.mode = modeEditActionItem
		m.editingActionItemID = actionItem.ID
		m.loadActionItemInfoComments(actionItem.ID)
		m.status = "edit actionItem"
	} else {
		m.formInputs[actionItemFieldPriority].Placeholder = "medium"
		m.formInputs[actionItemFieldDue].Placeholder = "-"
		m.formInputs[actionItemFieldLabels].Placeholder = "-"
		m.mode = modeAddActionItem
		m.editingActionItemID = ""
		m.status = "new actionItem"
		m.actionItemFormParentID, m.actionItemFormKind, m.actionItemFormScope = m.newActionItemDefaultsForActiveBoardScope()
		m.actionItemInfoComments = nil
		m.actionItemInfoCommentsError = ""
	}
	m.syncActionItemFormDescriptionDisplay()
	m.refreshActionItemFormLabelSuggestions()
	return m.focusActionItemFormField(0)
}

// newActionItemDefaultsForActiveBoardScope infers parent/kind/scope defaults from active focused scope.
func (m Model) newActionItemDefaultsForActiveBoardScope() (string, domain.Kind, domain.KindAppliesTo) {
	rootID := strings.TrimSpace(m.projectionRootActionItemID)
	if rootID == "" {
		return "", domain.KindActionItem, domain.KindAppliesToActionItem
	}
	root, ok := m.actionItemByID(rootID)
	if !ok {
		return "", domain.KindActionItem, domain.KindAppliesToActionItem
	}
	levelByActionItemID := m.searchLevelByActionItemID([]domain.ActionItem{root})
	level := strings.TrimSpace(levelByActionItemID[root.ID])
	if level == "" {
		level = baseSearchLevelForActionItem(root)
	}
	switch level {
	case "actionItem", "subtask":
		return root.ID, domain.KindSubtask, domain.KindAppliesToSubtask
	default:
		return root.ID, domain.KindActionItem, domain.KindAppliesToActionItem
	}
}

// startSubactionItemForm opens the actionItem form preconfigured for a child item.
func (m *Model) startSubactionItemForm(parent domain.ActionItem) tea.Cmd {
	cmd := m.startActionItemForm(nil)
	m.actionItemFormParentID = parent.ID
	m.actionItemFormKind = domain.KindSubtask
	m.actionItemFormScope = domain.KindAppliesToSubtask
	m.refreshActionItemFormLabelSuggestions()
	m.status = "new subtask for " + parent.Title
	return cmd
}

// startBranchForm opens the actionItem form preconfigured for a branch work item.
func (m *Model) startBranchForm(parent *domain.ActionItem) tea.Cmd {
	cmd := m.startActionItemForm(nil)
	m.actionItemFormKind = domain.Kind("branch")
	m.actionItemFormScope = domain.KindAppliesToBranch
	m.actionItemFormParentID = ""
	if parent != nil && strings.TrimSpace(parent.ID) != "" {
		m.actionItemFormParentID = parent.ID
	}
	if len(m.formInputs) > actionItemFieldTitle {
		m.formInputs[actionItemFieldTitle].Placeholder = "branch title (required)"
	}
	m.refreshActionItemFormLabelSuggestions()
	m.status = "new branch"
	return cmd
}

// startPhaseForm opens the actionItem form preconfigured for a phase work item.
func (m *Model) startPhaseForm(parent *domain.ActionItem) tea.Cmd {
	cmd := m.startActionItemForm(nil)
	m.actionItemFormKind = domain.KindPhase
	m.actionItemFormScope = domain.KindAppliesToPhase
	m.actionItemFormParentID = ""
	if parent != nil && strings.TrimSpace(parent.ID) != "" {
		m.actionItemFormParentID = parent.ID
	}
	if len(m.formInputs) > actionItemFieldTitle {
		m.formInputs[actionItemFieldTitle].Placeholder = "phase title (required)"
	}
	m.status = "new phase"
	m.refreshActionItemFormLabelSuggestions()
	return cmd
}

// startSubactionItemFormFromActionItemForm opens a subtask form from create/edit actionItem modal context.
func (m *Model) startSubactionItemFormFromActionItemForm() tea.Cmd {
	if m.mode == modeEditActionItem {
		actionItemID := strings.TrimSpace(m.editingActionItemID)
		if actionItemID == "" {
			actionItem, ok := m.selectedActionItemInCurrentColumn()
			if !ok {
				m.status = "no actionItem selected"
				return nil
			}
			actionItemID = actionItem.ID
		}
		actionItem, ok := m.actionItemByID(actionItemID)
		if !ok {
			m.status = "actionItem not found"
			return nil
		}
		cmd := m.startSubactionItemForm(actionItem)
		m.actionItemFormBackMode = modeEditActionItem
		m.actionItemFormBackActionItemID = actionItem.ID
		m.actionItemFormBackChildID = ""
		return cmd
	}
	parentID := strings.TrimSpace(m.actionItemFormParentID)
	if parentID == "" {
		m.status = "save actionItem first to add subtask"
		return nil
	}
	parent, ok := m.actionItemByID(parentID)
	if !ok {
		m.status = "parent actionItem not found"
		return nil
	}
	return m.startSubactionItemForm(parent)
}

// focusActionItemFormField focuses one actionItem-form field id.
func (m *Model) focusActionItemFormField(field int) tea.Cmd {
	order := m.actionItemFormFocusOrder()
	if len(order) == 0 {
		return nil
	}
	if m.actionItemFormFocusPosition(field) < 0 {
		// Support callers that still provide a visual index by mapping into focus order.
		if field >= 0 && field < len(order) {
			field = order[field]
		} else {
			field = order[0]
		}
	}
	if m.formFocus != field {
		switch field {
		case actionItemFieldSubtasks:
			if len(m.actionItemFormContextSubtasks()) > 0 {
				m.actionItemFormSubactionItemCursor = max(1, clamp(m.actionItemFormSubactionItemCursor, 1, len(m.actionItemFormContextSubtasks())))
			}
		case actionItemFieldResources:
			if len(m.actionItemFormResourceRefs) > 0 {
				m.actionItemFormResourceCursor = max(1, clamp(m.actionItemFormResourceCursor, 1, len(m.actionItemFormResourceRefs)))
			}
		}
	}
	m.formFocus = field
	for i := range m.formInputs {
		m.formInputs[i].Blur()
	}
	var cmd tea.Cmd
	if field < len(m.formInputs) && !isActionItemFormActionField(field) {
		cmd = m.formInputs[field].Focus()
	}
	if m.mode == modeAddActionItem || m.mode == modeEditActionItem {
		m.syncActionItemFormViewportToFocus()
	}
	return cmd
}

// actionItemFormFieldCount returns the number of navigable fields for the active actionItem form mode.
func (m Model) actionItemFormFieldCount() int {
	return len(m.actionItemFormFocusOrder())
}

// actionItemFormFocusOrder returns the visual-navigation order for actionItem form fields.
func (m Model) actionItemFormFocusOrder() []int {
	if len(m.formInputs) == 0 {
		return nil
	}
	return []int{
		actionItemFieldTitle,
		actionItemFieldDescription,
		actionItemFieldSubtasks,
		actionItemFieldPriority,
		actionItemFieldDue,
		actionItemFieldLabels,
		actionItemFieldDependsOn,
		actionItemFieldBlockedBy,
		actionItemFieldBlockedReason,
		actionItemFieldComments,
		actionItemFieldObjective,
		actionItemFieldAcceptanceCriteria,
		actionItemFieldValidationPlan,
		actionItemFieldRiskNotes,
		actionItemFieldResources,
	}
}

func isActionItemFormActionField(field int) bool {
	switch field {
	case actionItemFieldPriority,
		actionItemFieldDue,
		actionItemFieldLabels,
		actionItemFieldDependsOn,
		actionItemFieldBlockedBy,
		actionItemFieldComments,
		actionItemFieldSubtasks,
		actionItemFieldResources:
		return true
	default:
		return false
	}
}

// isActionItemFormDirectTextInputField reports whether the focused actionItem-form field should consume printable text directly.
func isActionItemFormDirectTextInputField(field int) bool {
	return field == actionItemFieldTitle
}

// isProjectFormDirectTextInputField reports whether the focused project-form field should consume printable text directly.
func isProjectFormDirectTextInputField(field int) bool {
	return field != projectFieldDescription && field != projectFieldKind && field != projectFieldTemplateLibrary && field != projectFieldComments
}

// actionItemFormFocusPosition resolves one form-focus field position within the current visual order.
func (m Model) actionItemFormFocusPosition(field int) int {
	for idx, candidate := range m.actionItemFormFocusOrder() {
		if candidate == field {
			return idx
		}
	}
	return -1
}

// moveActionItemFormFocus shifts actionItem-form focus by delta and optionally wraps around field bounds.
func (m *Model) moveActionItemFormFocus(delta int, wrap bool) tea.Cmd {
	order := m.actionItemFormFocusOrder()
	total := len(order)
	if total == 0 {
		return nil
	}
	position := m.actionItemFormFocusPosition(m.formFocus)
	if position < 0 {
		position = 0
	}
	next := position + delta
	if wrap {
		next = wrapIndex(position, delta, total)
	} else {
		next = clamp(next, 0, total-1)
	}
	return m.focusActionItemFormField(order[next])
}

// isPrintableFormTextKey reports whether a keypress should insert printable text into a focused input.
func isPrintableFormTextKey(msg tea.KeyPressMsg) bool {
	if msg.Text == "" {
		return false
	}
	return (msg.Mod & ^tea.ModShift) == 0
}

// isActionItemFormMarkdownField reports whether one actionItem-form field uses the full-screen markdown editor flow.
func isActionItemFormMarkdownField(field int) bool {
	switch field {
	case actionItemFieldDescription,
		actionItemFieldBlockedReason,
		actionItemFieldObjective,
		actionItemFieldAcceptanceCriteria,
		actionItemFieldValidationPlan,
		actionItemFieldRiskNotes:
		return true
	default:
		return false
	}
}

// actionItemFormUsesDedicatedMarkdownDraft reports whether one markdown-capable field should use dedicated draft state.
func actionItemFormUsesDedicatedMarkdownDraft(field int) bool {
	switch field {
	case actionItemFieldBlockedReason,
		actionItemFieldObjective,
		actionItemFieldAcceptanceCriteria,
		actionItemFieldValidationPlan,
		actionItemFieldRiskNotes:
		return true
	default:
		return false
	}
}

// isActionItemFormDependencyField reports whether one actionItem-form field maps to dependency relations.
func isActionItemFormDependencyField(field int) bool {
	return field == actionItemFieldDependsOn || field == actionItemFieldBlockedBy
}

// actionItemFormContextSubtasks resolves direct children for the actionItem currently edited in actionItem form.
func (m Model) actionItemFormContextSubtasks() []domain.ActionItem {
	contextActionItem, ok := m.actionItemFormContextActionItem()
	if !ok {
		return nil
	}
	return m.subtasksForParent(contextActionItem.ID)
}

// moveActionItemFormSubactionItemCursor shifts focused subtask row in edit mode (0 = create new).
func (m *Model) moveActionItemFormSubactionItemCursor(delta int) {
	if m == nil || (m.mode != modeAddActionItem && m.mode != modeEditActionItem) {
		return
	}
	total := 1 + len(m.actionItemFormContextSubtasks())
	if total <= 0 {
		m.actionItemFormSubactionItemCursor = 0
		return
	}
	current := clamp(m.actionItemFormSubactionItemCursor, 0, total-1)
	m.actionItemFormSubactionItemCursor = wrapIndex(current, delta, total)
}

// selectedActionItemFormSubtask returns the focused existing subtask row in edit mode.
func (m Model) selectedActionItemFormSubtask() (domain.ActionItem, bool) {
	subtasks := m.actionItemFormContextSubtasks()
	if len(subtasks) == 0 {
		return domain.ActionItem{}, false
	}
	idx := clamp(m.actionItemFormSubactionItemCursor-1, 0, len(subtasks)-1)
	if m.actionItemFormSubactionItemCursor <= 0 {
		return domain.ActionItem{}, false
	}
	return subtasks[idx], true
}

// openFocusedActionItemFormSubtask opens the selected subtask for edit or starts create flow when create-row is selected.
func (m *Model) openFocusedActionItemFormSubtask() tea.Cmd {
	if m == nil {
		return nil
	}
	if subtask, ok := m.selectedActionItemFormSubtask(); ok {
		parentID := strings.TrimSpace(m.editingActionItemID)
		traceActionItemScreenAction("task_edit", "subtask_open", "parent_action_item_id", parentID, "child_action_item_id", subtask.ID)
		cmd := m.startActionItemForm(&subtask)
		if parentID != "" {
			m.actionItemFormBackMode = modeEditActionItem
			m.actionItemFormBackActionItemID = parentID
			m.actionItemFormBackChildID = subtask.ID
		}
		return cmd
	}
	traceActionItemScreenAction("task_edit", "subtask_create_from_row", "parent_action_item_id", strings.TrimSpace(m.editingActionItemID))
	return m.startSubactionItemFormFromActionItemForm()
}

// moveActionItemFormResourceCursor shifts focused resource row in edit mode (0 = attach new).
func (m *Model) moveActionItemFormResourceCursor(delta int) {
	if m == nil || (m.mode != modeAddActionItem && m.mode != modeEditActionItem) {
		return
	}
	total := 1 + len(m.actionItemFormResourceRefs)
	if total <= 0 {
		m.actionItemFormResourceCursor = 0
		return
	}
	current := clamp(m.actionItemFormResourceCursor, 0, total-1)
	m.actionItemFormResourceCursor = wrapIndex(current, delta, total)
}

// startActionItemFormResourcePickerFromFocus opens resource picker for add/replace based on focused resources row.
func (m *Model) startActionItemFormResourcePickerFromFocus() tea.Cmd {
	if m == nil {
		return nil
	}
	if m.mode == modeAddActionItem {
		m.status = "save actionItem first to attach resources"
		traceActionItemScreenAction("task_edit", "resource_picker_blocked", "reason", "save_task_first")
		return nil
	}
	m.actionItemFormResourceEditIndex = -1
	if m.mode == modeEditActionItem && m.actionItemFormResourceCursor > 0 {
		m.actionItemFormResourceEditIndex = clamp(m.actionItemFormResourceCursor-1, 0, len(m.actionItemFormResourceRefs)-1)
	}
	actionItemID := strings.TrimSpace(m.editingActionItemID)
	if actionItemID == "" {
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return nil
		}
		actionItemID = actionItem.ID
	}
	return m.startResourcePicker(actionItemID, m.mode)
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
	if idx == projectFieldDescription || idx == projectFieldKind || idx == projectFieldTemplateLibrary || idx == projectFieldComments {
		return nil
	}
	return m.projectFormInputs[idx].Focus()
}

// actionItemFormMarkdownFieldLabel returns one stable label for markdown-editable actionItem form fields.
func actionItemFormMarkdownFieldLabel(field int) string {
	switch field {
	case actionItemFieldDescription:
		return "description"
	case actionItemFieldBlockedReason:
		return "blocked_reason"
	case actionItemFieldObjective:
		return "objective"
	case actionItemFieldAcceptanceCriteria:
		return "acceptance_criteria"
	case actionItemFieldValidationPlan:
		return "validation_plan"
	case actionItemFieldRiskNotes:
		return "risk_notes"
	default:
		return "description"
	}
}

// initActionItemFormMarkdownDrafts resets dedicated markdown draft state for the active actionItem form.
func (m *Model) initActionItemFormMarkdownDrafts() {
	if m == nil {
		return
	}
	m.actionItemFormMarkdown = map[int]string{}
	m.actionItemFormTouched = map[int]bool{}
}

// actionItemFormMarkdownDraft returns one dedicated markdown draft value for a actionItem-form field.
func (m Model) actionItemFormMarkdownDraft(field int) string {
	if !actionItemFormUsesDedicatedMarkdownDraft(field) {
		return ""
	}
	if m.actionItemFormMarkdown == nil {
		return ""
	}
	return strings.TrimSpace(m.actionItemFormMarkdown[field])
}

// setActionItemFormMarkdownDraft stores one dedicated markdown draft and syncs the compact row display.
func (m *Model) setActionItemFormMarkdownDraft(field int, value string, touched bool) {
	if m == nil || !actionItemFormUsesDedicatedMarkdownDraft(field) {
		return
	}
	if m.actionItemFormMarkdown == nil {
		m.actionItemFormMarkdown = map[int]string{}
	}
	if m.actionItemFormTouched == nil {
		m.actionItemFormTouched = map[int]bool{}
	}
	value = strings.TrimSpace(value)
	m.actionItemFormMarkdown[field] = value
	if touched {
		m.actionItemFormTouched[field] = true
	}
	if field >= 0 && field < len(m.formInputs) {
		m.formInputs[field].SetValue(descriptionFormDisplayValue(value))
		m.formInputs[field].CursorEnd()
	}
}

// actionItemFormMarkdownFieldValue returns the current value for one markdown-editable actionItem form field.
func (m Model) actionItemFormMarkdownFieldValue(field int) string {
	switch field {
	case actionItemFieldDescription:
		return strings.TrimSpace(m.actionItemFormDescription)
	default:
		if actionItemFormUsesDedicatedMarkdownDraft(field) {
			return m.actionItemFormMarkdownDraft(field)
		}
		if field >= 0 && field < len(m.formInputs) {
			return strings.TrimSpace(m.formInputs[field].Value())
		}
		return ""
	}
}

// setActionItemFormMarkdownFieldValue persists markdown-editor output back into one actionItem form field.
func (m *Model) setActionItemFormMarkdownFieldValue(field int, value string) {
	if m == nil {
		return
	}
	value = strings.TrimSpace(value)
	switch field {
	case actionItemFieldDescription:
		m.actionItemFormDescription = value
		m.syncActionItemFormDescriptionDisplay()
	default:
		if actionItemFormUsesDedicatedMarkdownDraft(field) {
			m.setActionItemFormMarkdownDraft(field, value, true)
			return
		}
		if field >= 0 && field < len(m.formInputs) {
			m.formInputs[field].SetValue(value)
			m.formInputs[field].CursorEnd()
		}
	}
}

// startActionItemFormMarkdownEditor opens the shared full-screen markdown editor for one actionItem-form field.
func (m *Model) startActionItemFormMarkdownEditor(field int, seed tea.KeyPressMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	if !isActionItemFormMarkdownField(field) {
		return nil
	}
	m.descriptionEditorBack = m.mode
	if field == actionItemFieldDescription {
		m.descriptionEditorTarget = descriptionEditorTargetActionItem
		m.descriptionEditorActionItemFormField = -1
	} else {
		m.descriptionEditorTarget = descriptionEditorTargetActionItemFormField
		m.descriptionEditorActionItemFormField = field
	}
	m.descriptionEditorMode = descriptionEditorViewModeEdit
	m.descriptionEditorPath = m.descriptionEditorPathForActionItemForm()
	m.descriptionEditorThreadDetails = false
	m.mode = modeDescriptionEditor
	m.descriptionEditorInput.SetValue(m.actionItemFormMarkdownFieldValue(field))
	m.descriptionEditorInput.CursorEnd()
	m.descriptionEditorInput.ShowLineNumbers = true
	m.applySeedKeyToDescriptionEditor(seed)
	m.resetDescriptionEditorHistory()
	m.resetDescriptionPreviewToTop()
	m.help.ShowAll = false
	m.status = "editing " + actionItemFormMarkdownFieldLabel(field)
	return m.descriptionEditorInput.Focus()
}

// startActionItemDescriptionEditor opens the full-screen markdown description editor for actionItem forms.
func (m *Model) startActionItemDescriptionEditor(seed tea.KeyPressMsg) tea.Cmd {
	return m.startActionItemFormMarkdownEditor(actionItemFieldDescription, seed)
}

// startProjectDescriptionEditor opens the full-screen markdown description editor for project forms.
func (m *Model) startProjectDescriptionEditor(seed tea.KeyPressMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	m.descriptionEditorBack = m.mode
	m.descriptionEditorTarget = descriptionEditorTargetProject
	m.descriptionEditorActionItemFormField = -1
	m.descriptionEditorMode = descriptionEditorViewModeEdit
	m.descriptionEditorPath = m.descriptionEditorPathForProjectForm()
	m.descriptionEditorThreadDetails = false
	m.mode = modeDescriptionEditor
	m.descriptionEditorInput.SetValue(m.projectFormDescription)
	m.descriptionEditorInput.CursorEnd()
	m.descriptionEditorInput.ShowLineNumbers = true
	m.applySeedKeyToDescriptionEditor(seed)
	m.resetDescriptionEditorHistory()
	m.resetDescriptionPreviewToTop()
	m.help.ShowAll = false
	m.status = "editing description"
	return m.descriptionEditorInput.Focus()
}

// startThreadDescriptionEditor opens the full-screen markdown description editor for thread details.
func (m *Model) startThreadDescriptionEditor() tea.Cmd {
	if m == nil {
		return nil
	}
	m.descriptionEditorBack = modeThread
	m.descriptionEditorTarget = descriptionEditorTargetThread
	m.descriptionEditorActionItemFormField = -1
	m.descriptionEditorMode = descriptionEditorViewModeEdit
	m.descriptionEditorPath = m.descriptionEditorPathForThreadTarget()
	m.descriptionEditorThreadDetails = m.threadDetailsActive
	m.mode = modeDescriptionEditor
	m.threadComposerActive = false
	m.threadInput.Blur()
	m.descriptionEditorInput.SetValue(strings.TrimSpace(m.threadDescriptionMarkdown))
	m.descriptionEditorInput.CursorEnd()
	m.descriptionEditorInput.ShowLineNumbers = true
	m.resetDescriptionEditorHistory()
	m.resetDescriptionPreviewToTop()
	m.help.ShowAll = false
	m.status = "editing description"
	return m.descriptionEditorInput.Focus()
}

// startActionItemInfoDescriptionEditor opens the full-screen markdown description editor in preview mode from actionItem-info.
func (m *Model) startActionItemInfoDescriptionEditor(actionItem domain.ActionItem) tea.Cmd {
	if m == nil {
		return nil
	}
	targetType, ok := commentTargetTypeForActionItem(actionItem)
	if !ok {
		m.status = "unsupported work-item kind for details"
		return nil
	}
	target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  actionItem.ProjectID,
		TargetType: targetType,
		TargetID:   actionItem.ID,
	})
	if err != nil {
		m.status = "work-item details target invalid: " + err.Error()
		return nil
	}
	title := strings.TrimSpace(actionItem.Title)
	if title == "" {
		title = actionItem.ID
	}
	m.threadTarget = target
	m.threadTitle = fmt.Sprintf("%s: %s", actionItem.Kind, title)
	m.threadDescriptionMarkdown = m.threadDescriptionForTarget(target, actionItem.Description)
	m.descriptionEditorBack = modeActionItemInfo
	m.descriptionEditorTarget = descriptionEditorTargetThread
	m.descriptionEditorActionItemFormField = -1
	m.descriptionEditorMode = descriptionEditorViewModePreview
	m.descriptionEditorPath = m.descriptionEditorActionItemPath(actionItem)
	m.descriptionEditorThreadDetails = false
	m.mode = modeDescriptionEditor
	m.threadComposerActive = false
	m.threadDetailsActive = false
	m.threadInput.Blur()
	m.descriptionEditorInput.SetValue(strings.TrimSpace(m.threadDescriptionMarkdown))
	m.descriptionEditorInput.MoveToBegin()
	m.descriptionEditorInput.ShowLineNumbers = true
	m.resetDescriptionEditorHistory()
	m.resetDescriptionPreviewToTop()
	m.help.ShowAll = false
	m.status = "previewing description"
	return nil
}

// applySeedKeyToDescriptionEditor applies one keypress that triggered markdown-editor entry.
func (m *Model) applySeedKeyToDescriptionEditor(seed tea.KeyPressMsg) {
	if m == nil {
		return
	}
	switch {
	case seed.Text != "" && (seed.Mod&tea.ModCtrl) == 0 && (seed.Mod&tea.ModAlt) == 0:
		m.descriptionEditorInput.InsertString(seed.Text)
	case seed.Code == tea.KeyBackspace || seed.String() == "backspace":
		value := m.descriptionEditorInput.Value()
		runes := []rune(value)
		if len(runes) > 0 {
			m.descriptionEditorInput.SetValue(string(runes[:len(runes)-1]))
			m.descriptionEditorInput.CursorEnd()
		}
	}
}

// resetDescriptionEditorHistory clears per-session undo/redo stacks for description editing.
func (m *Model) resetDescriptionEditorHistory() {
	if m == nil {
		return
	}
	m.descriptionEditorUndo = nil
	m.descriptionEditorRedo = nil
}

// resetThreadComposerHistory clears per-session undo/redo stacks for comment composer editing.
func (m *Model) resetThreadComposerHistory() {
	if m == nil {
		return
	}
	m.threadComposerUndo = nil
	m.threadComposerRedo = nil
}

// pushTextEditHistory appends one undo checkpoint and clears redo checkpoints when text changes.
func pushTextEditHistory(undo, redo *[]string, before, after string) {
	if undo == nil || redo == nil || before == after {
		return
	}
	*undo = append(*undo, before)
	if len(*undo) > textEditHistoryLimit {
		*undo = append([]string(nil), (*undo)[len(*undo)-textEditHistoryLimit:]...)
	}
	*redo = nil
}

// applyTextEditUndo applies one text-level undo transition and reports whether state changed.
func applyTextEditUndo(value *string, undo, redo *[]string) bool {
	if value == nil || undo == nil || redo == nil || len(*undo) == 0 {
		return false
	}
	current := *value
	prev := (*undo)[len(*undo)-1]
	*undo = (*undo)[:len(*undo)-1]
	*redo = append(*redo, current)
	*value = prev
	return true
}

// applyTextEditRedo applies one text-level redo transition and reports whether state changed.
func applyTextEditRedo(value *string, undo, redo *[]string) bool {
	if value == nil || undo == nil || redo == nil || len(*redo) == 0 {
		return false
	}
	current := *value
	next := (*redo)[len(*redo)-1]
	*redo = (*redo)[:len(*redo)-1]
	*undo = append(*undo, current)
	*value = next
	return true
}

// saveDescriptionEditor persists markdown editor content back into the active add/edit form.
func (m *Model) saveDescriptionEditor() {
	if m == nil {
		return
	}
	text := strings.TrimSpace(m.descriptionEditorInput.Value())
	switch m.descriptionEditorTarget {
	case descriptionEditorTargetActionItem:
		m.actionItemFormDescription = text
		m.syncActionItemFormDescriptionDisplay()
	case descriptionEditorTargetActionItemFormField:
		m.setActionItemFormMarkdownFieldValue(m.descriptionEditorActionItemFormField, text)
	case descriptionEditorTargetProject:
		m.projectFormDescription = text
		m.syncProjectFormDescriptionDisplay()
	case descriptionEditorTargetThread:
		m.threadDescriptionMarkdown = text
	}
}

// closeDescriptionEditor exits markdown-description editor and returns to the previous mode context.
func (m *Model) closeDescriptionEditor(saved bool) tea.Cmd {
	if m == nil {
		return nil
	}
	back := m.descriptionEditorBack
	if back != modeAddActionItem && back != modeEditActionItem && back != modeAddProject && back != modeEditProject && back != modeThread && back != modeActionItemInfo {
		back = modeNone
	}
	m.mode = back
	m.descriptionEditorInput.Blur()
	m.descriptionEditorBack = modeNone
	target := m.descriptionEditorTarget
	field := m.descriptionEditorActionItemFormField
	m.descriptionEditorTarget = descriptionEditorTargetActionItem
	m.descriptionEditorActionItemFormField = -1
	m.descriptionEditorMode = descriptionEditorViewModeEdit
	m.descriptionEditorPath = ""
	m.descriptionPreview.SetYOffset(0)
	m.resetDescriptionEditorHistory()
	threadDetailsActive := m.descriptionEditorThreadDetails
	m.descriptionEditorThreadDetails = false
	if back == modeThread || back == modeActionItemInfo {
		m.threadDetailsActive = threadDetailsActive
		if saved {
			if back == modeActionItemInfo {
				m.status = "saving actionItem details..."
			} else {
				m.status = "saving thread details..."
			}
			return m.updateThreadDescriptionCmd(strings.TrimSpace(m.threadDescriptionMarkdown))
		}
		if back == modeActionItemInfo {
			m.status = "actionItem info"
			if actionItem, ok := m.actionItemInfoActionItem(); ok {
				m.syncActionItemInfoDetailsViewport(actionItem)
				m.syncActionItemInfoBodyViewport(actionItem)
			}
			return nil
		}
		m.status = "ready"
		return nil
	}
	if saved && back == modeEditActionItem && (target == descriptionEditorTargetActionItem || target == descriptionEditorTargetActionItemFormField) {
		cmd, err := m.persistCurrentEditActionItemCmd("actionItem updated")
		if err != nil {
			m.status = err.Error()
			if target == descriptionEditorTargetActionItemFormField && isActionItemFormMarkdownField(field) {
				return m.focusActionItemFormField(field)
			}
			return m.focusActionItemFormField(actionItemFieldDescription)
		}
		m.status = "saving actionItem..."
		return cmd
	}
	if saved {
		m.status = "description updated"
	} else {
		m.status = "description edit cancelled"
	}
	switch back {
	case modeAddActionItem, modeEditActionItem:
		if target == descriptionEditorTargetActionItemFormField && isActionItemFormMarkdownField(field) {
			return m.focusActionItemFormField(field)
		}
		return m.focusActionItemFormField(actionItemFieldDescription)
	case modeAddProject, modeEditProject:
		return m.focusProjectFormField(projectFieldDescription)
	default:
		return nil
	}
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
	m.labelsConfigBranchActionItemID = ""
	m.labelsConfigPhaseActionItemID = ""
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
	if branch, ok := m.labelsConfigContextActionItem("branch"); ok {
		m.labelsConfigBranchActionItemID = branch.ID
		if len(branch.Labels) > 0 {
			m.labelsConfigInputs[2].SetValue(strings.Join(branch.Labels, ","))
		}
	}
	if phase, ok := m.labelsConfigContextActionItem("phase"); ok {
		m.labelsConfigPhaseActionItemID = phase.ID
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

// labelsConfigContextActionItem resolves the nearest selected actionItem at one of the requested levels.
func (m Model) labelsConfigContextActionItem(levels ...string) (domain.ActionItem, bool) {
	if len(levels) == 0 {
		return domain.ActionItem{}, false
	}
	targetSet := map[string]struct{}{}
	for _, level := range levels {
		level = strings.TrimSpace(strings.ToLower(level))
		if level == "" {
			continue
		}
		targetSet[level] = struct{}{}
	}
	actionItem, ok := m.selectedActionItemForLabelInheritance()
	if !ok {
		return domain.ActionItem{}, false
	}
	visited := map[string]struct{}{}
	current := actionItem
	for strings.TrimSpace(current.ID) != "" {
		if _, seen := visited[current.ID]; seen {
			break
		}
		visited[current.ID] = struct{}{}
		if _, wanted := targetSet[baseSearchLevelForActionItem(current)]; wanted {
			return current, true
		}
		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, found := m.actionItemByID(parentID)
		if !found {
			break
		}
		current = parent
	}
	return domain.ActionItem{}, false
}

// actionItemFormValues returns actionItem form values.
func (m Model) actionItemFormValues() map[string]string {
	out := map[string]string{}
	for i, key := range actionItemFormFields {
		if i >= len(m.formInputs) {
			break
		}
		if actionItemFormUsesDedicatedMarkdownDraft(i) {
			out[key] = sanitizeFormFieldValue(m.actionItemFormMarkdownDraft(i))
			continue
		}
		out[key] = sanitizeFormFieldValue(m.formInputs[i].Value())
	}
	out["description"] = sanitizeFormFieldValue(m.actionItemFormDescription)
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
var projectFormFields = []string{"name", "description", "kind", "owner", "icon", "color", "homepage", "tags", "template_library_id", "root_path"}

// projectFormValues returns project form values.
func (m Model) projectFormValues() map[string]string {
	out := map[string]string{}
	for idx, key := range projectFormFields {
		if idx >= len(m.projectFormInputs) {
			break
		}
		out[key] = sanitizeFormFieldValue(m.projectFormInputs[idx].Value())
	}
	out["description"] = sanitizeFormFieldValue(m.projectFormDescription)
	return out
}

// projectKindDisplayLabel returns one stable id/name label for project-kind rows.
func (m Model) projectKindDisplayLabel(kindID domain.KindID, displayName string) string {
	kindID = domain.NormalizeKindID(kindID)
	displayName = strings.TrimSpace(displayName)
	if kindID == "" {
		return string(domain.DefaultProjectKind)
	}
	if displayName == "" || strings.EqualFold(displayName, string(kindID)) {
		return string(kindID)
	}
	return fmt.Sprintf("%s — %s", kindID, displayName)
}

// projectKindName returns the currently loaded kind display name for one id when available.
func (m Model) projectKindName(kindID domain.KindID) string {
	kindID = domain.NormalizeKindID(kindID)
	if kindID == "" {
		return ""
	}
	for _, kind := range m.kindDefinitions {
		if domain.NormalizeKindID(kind.ID) != kindID {
			continue
		}
		return strings.TrimSpace(kind.DisplayName)
	}
	return ""
}

// projectKindPickerOptions returns project-applicable kinds sorted for picker rendering.
func (m Model) projectKindPickerOptions() []projectKindPickerItem {
	items := make([]projectKindPickerItem, 0, len(m.kindDefinitions))
	for _, kind := range m.kindDefinitions {
		if kind.ArchivedAt != nil {
			continue
		}
		if !kind.AppliesToScope(domain.KindAppliesToProject) {
			continue
		}
		items = append(items, projectKindPickerItem{
			KindID:      domain.NormalizeKindID(kind.ID),
			DisplayName: strings.TrimSpace(kind.DisplayName),
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := m.projectKindDisplayLabel(items[i].KindID, items[i].DisplayName)
		right := m.projectKindDisplayLabel(items[j].KindID, items[j].DisplayName)
		return left < right
	})
	return items
}

// projectKindSummaryRows returns readable project-kind rows for project-form rendering.
func (m Model) projectKindSummaryRows(limit int) []string {
	options := m.projectKindPickerOptions()
	if limit <= 0 {
		limit = len(options)
	}
	rows := make([]string, 0, min(limit, len(options)))
	for _, item := range options {
		rows = append(rows, m.projectKindDisplayLabel(item.KindID, item.DisplayName))
		if len(rows) >= limit {
			break
		}
	}
	return rows
}

// hasProjectKindDefinition reports whether the given kind id is currently selectable in the project form.
func (m Model) hasProjectKindDefinition(kindID domain.KindID) bool {
	kindID = domain.NormalizeKindID(kindID)
	if kindID == "" {
		return false
	}
	for _, item := range m.projectKindPickerOptions() {
		if item.KindID == kindID {
			return true
		}
	}
	return false
}

// refreshProjectKindPickerMatches refreshes fuzzy-filtered project-kind picker rows.
func (m *Model) refreshProjectKindPickerMatches() {
	if m == nil {
		return
	}
	allItems := m.projectKindPickerOptions()
	query := strings.TrimSpace(m.projectKindPickerInput.Value())
	if query == "" {
		m.projectKindPickerItems = allItems
		m.projectKindPickerIndex = clamp(m.projectKindPickerIndex, 0, len(m.projectKindPickerItems)-1)
		return
	}
	type scoredProjectKind struct {
		item  projectKindPickerItem
		score int
	}
	scored := make([]scoredProjectKind, 0, len(allItems))
	for _, item := range allItems {
		score, ok := bestFuzzyScore(query, string(item.KindID), item.DisplayName, m.projectKindDisplayLabel(item.KindID, item.DisplayName))
		if !ok {
			continue
		}
		scored = append(scored, scoredProjectKind{
			item:  item,
			score: score,
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		left := m.projectKindDisplayLabel(scored[i].item.KindID, scored[i].item.DisplayName)
		right := m.projectKindDisplayLabel(scored[j].item.KindID, scored[j].item.DisplayName)
		return left < right
	})
	m.projectKindPickerItems = make([]projectKindPickerItem, 0, len(scored))
	for _, entry := range scored {
		m.projectKindPickerItems = append(m.projectKindPickerItems, entry.item)
	}
	m.projectKindPickerIndex = clamp(m.projectKindPickerIndex, 0, len(m.projectKindPickerItems)-1)
}

// startProjectKindPicker opens the project-kind picker with optional initial filter text.
func (m *Model) startProjectKindPicker(seed string) tea.Cmd {
	if m == nil {
		return nil
	}
	m.projectKindPickerBack = m.mode
	m.mode = modeProjectKindPicker
	m.projectKindPickerInput.SetValue(strings.TrimSpace(seed))
	m.projectKindPickerInput.CursorEnd()
	m.refreshProjectKindPickerMatches()
	current := domain.NormalizeKindID(domain.KindID(m.projectFormInputs[projectFieldKind].Value()))
	m.projectKindPickerIndex = 0
	if current != "" && strings.TrimSpace(seed) == "" {
		for idx, item := range m.projectKindPickerItems {
			if item.KindID == current {
				m.projectKindPickerIndex = idx
				break
			}
		}
	}
	if len(m.projectKindPickerItems) == 0 {
		m.status = "no project kinds available"
	} else {
		m.status = "project kind picker"
	}
	return m.projectKindPickerInput.Focus()
}

// inferProjectKindFromTemplateLibrary returns one unique project-scope kind when a selected library implies exactly one.
func (m Model) inferProjectKindFromTemplateLibrary(libraryID string) (domain.KindID, bool) {
	libraryID = domain.NormalizeTemplateLibraryID(libraryID)
	if libraryID == "" {
		return "", false
	}
	seen := map[domain.KindID]struct{}{}
	candidates := make([]domain.KindID, 0, 1)
	for _, library := range m.templateLibraries {
		if domain.NormalizeTemplateLibraryID(library.ID) != libraryID {
			continue
		}
		for _, nodeTemplate := range library.NodeTemplates {
			if nodeTemplate.ScopeLevel != domain.KindAppliesToProject {
				continue
			}
			kindID := domain.NormalizeKindID(nodeTemplate.NodeKindID)
			if kindID == "" {
				continue
			}
			if _, ok := seen[kindID]; ok {
				continue
			}
			seen[kindID] = struct{}{}
			candidates = append(candidates, kindID)
		}
		break
	}
	if len(candidates) != 1 {
		return "", false
	}
	return candidates[0], true
}

// activeProjectTemplateBinding returns the loaded active binding for one project when available.
func (m Model) activeProjectTemplateBinding(projectID string) (domain.ProjectTemplateBinding, bool) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" || m.currentProjectTemplateBinding == nil {
		return domain.ProjectTemplateBinding{}, false
	}
	if strings.TrimSpace(m.currentProjectTemplateBinding.ProjectID) != projectID {
		return domain.ProjectTemplateBinding{}, false
	}
	return *m.currentProjectTemplateBinding, true
}

// builtinTemplateStatus returns one loaded builtin lifecycle status by library id when available.
func (m Model) builtinTemplateStatus(libraryID string) (domain.BuiltinTemplateLibraryStatus, bool) {
	libraryID = domain.NormalizeTemplateLibraryID(libraryID)
	if libraryID == "" || len(m.builtinTemplateStatuses) == 0 {
		return domain.BuiltinTemplateLibraryStatus{}, false
	}
	status, ok := m.builtinTemplateStatuses[libraryID]
	if !ok {
		return domain.BuiltinTemplateLibraryStatus{}, false
	}
	return status, true
}

// hasApprovedTemplateLibrary reports whether the given global approved template library is currently visible to the TUI.
func (m Model) hasApprovedTemplateLibrary(libraryID string) bool {
	libraryID = domain.NormalizeTemplateLibraryID(libraryID)
	if libraryID == "" {
		return false
	}
	for _, library := range m.templateLibraries {
		if domain.NormalizeTemplateLibraryID(library.ID) != libraryID {
			continue
		}
		if library.Scope != domain.TemplateLibraryScopeGlobal || library.Status != domain.TemplateLibraryStatusApproved {
			continue
		}
		return true
	}
	return false
}

// templateLibrarySummaryRows returns readable approved-library rows for project-form rendering.
func (m Model) templateLibrarySummaryRows(limit int) []string {
	if limit <= 0 {
		limit = len(m.templateLibraries)
	}
	rows := make([]string, 0, min(limit, len(m.templateLibraries)))
	for _, library := range m.templateLibraries {
		if library.Scope != domain.TemplateLibraryScopeGlobal || library.Status != domain.TemplateLibraryStatusApproved {
			continue
		}
		rows = append(rows, m.templateLibraryDisplayLabel(library.ID, library.Name))
		if len(rows) >= limit {
			break
		}
	}
	return rows
}

// loadBuiltinTemplateStatuses resolves builtin lifecycle status for visible approved builtin-managed libraries.
func (m Model) loadBuiltinTemplateStatuses(templateLibraries []domain.TemplateLibrary) (map[string]domain.BuiltinTemplateLibraryStatus, error) {
	statuses := map[string]domain.BuiltinTemplateLibraryStatus{}
	for _, library := range templateLibraries {
		if library.Scope != domain.TemplateLibraryScopeGlobal || library.Status != domain.TemplateLibraryStatusApproved || !library.BuiltinManaged {
			continue
		}
		libraryID := domain.NormalizeTemplateLibraryID(library.ID)
		if libraryID == "" {
			continue
		}
		if _, ok := statuses[libraryID]; ok {
			continue
		}
		status, err := m.svc.GetBuiltinTemplateLibraryStatus(context.Background(), libraryID)
		if err != nil {
			return nil, err
		}
		statuses[libraryID] = status
	}
	return statuses, nil
}

// builtinTemplateStatusSummary renders one compact builtin lifecycle summary for TUI hints.
func (m Model) builtinTemplateStatusSummary(status domain.BuiltinTemplateLibraryStatus) string {
	if status.LibraryID == "" {
		return "-"
	}
	parts := []string{"state:" + fallbackText(strings.TrimSpace(string(status.State)), "-")}
	if version := strings.TrimSpace(status.BuiltinVersion); version != "" {
		parts = append(parts, "version:"+version)
	}
	if status.InstalledRevision > 0 {
		parts = append(parts, fmt.Sprintf("installed_rev:%d", status.InstalledRevision))
	}
	return strings.Join(parts, " • ")
}

// templateLibraryDisplayLabel returns one stable id/name label for project-form and picker rows.
func (m Model) templateLibraryDisplayLabel(libraryID, libraryName string) string {
	libraryID = domain.NormalizeTemplateLibraryID(libraryID)
	libraryName = strings.TrimSpace(libraryName)
	if libraryID == "" {
		return "(none)"
	}
	if libraryName == "" || strings.EqualFold(libraryName, libraryID) {
		return libraryID
	}
	return fmt.Sprintf("%s — %s", libraryID, libraryName)
}

// templateLibraryName returns the currently loaded approved-library name for one id when available.
func (m Model) templateLibraryName(libraryID string) string {
	libraryID = domain.NormalizeTemplateLibraryID(libraryID)
	if libraryID == "" {
		return ""
	}
	for _, library := range m.templateLibraries {
		if domain.NormalizeTemplateLibraryID(library.ID) != libraryID {
			continue
		}
		return strings.TrimSpace(library.Name)
	}
	return ""
}

func (m Model) templateBindingSummary(binding domain.ProjectTemplateBinding) string {
	label := m.templateLibraryDisplayLabel(binding.LibraryID, firstNonEmptyTrimmed(binding.LibraryName, m.templateLibraryName(binding.LibraryID)))
	if binding.LibraryID == "" {
		return "-"
	}
	parts := []string{label}
	if binding.BoundRevision > 0 {
		parts = append(parts, fmt.Sprintf("rev:%d", binding.BoundRevision))
	}
	if drift := strings.TrimSpace(binding.DriftStatus); drift != "" {
		parts = append(parts, "drift:"+drift)
	}
	if binding.LatestRevision > 0 && binding.LatestRevision != binding.BoundRevision {
		parts = append(parts, fmt.Sprintf("latest:%d", binding.LatestRevision))
	}
	return strings.Join(parts, " • ")
}

// approvedTemplateLibraryPickerItems builds the ordered approved-library rows for the project form picker.
func (m Model) approvedTemplateLibraryPickerItems() []templateLibraryPickerItem {
	items := []templateLibraryPickerItem{{
		Name:  "(none)",
		Clear: true,
	}}
	for _, library := range m.templateLibraries {
		if library.Scope != domain.TemplateLibraryScopeGlobal || library.Status != domain.TemplateLibraryStatusApproved {
			continue
		}
		items = append(items, templateLibraryPickerItem{
			LibraryID: domain.NormalizeTemplateLibraryID(library.ID),
			Name:      strings.TrimSpace(library.Name),
		})
	}
	sort.SliceStable(items[1:], func(i, j int) bool {
		left := items[i+1]
		right := items[j+1]
		leftLabel := m.templateLibraryDisplayLabel(left.LibraryID, left.Name)
		rightLabel := m.templateLibraryDisplayLabel(right.LibraryID, right.Name)
		return leftLabel < rightLabel
	})
	return items
}

// refreshTemplateLibraryPickerMatches refreshes fuzzy-filtered approved-library picker rows.
func (m *Model) refreshTemplateLibraryPickerMatches() {
	if m == nil {
		return
	}
	allItems := m.approvedTemplateLibraryPickerItems()
	query := strings.TrimSpace(m.templateLibraryPickerInput.Value())
	if query == "" {
		m.templateLibraryPickerItems = allItems
		m.templateLibraryPickerIndex = clamp(m.templateLibraryPickerIndex, 0, len(m.templateLibraryPickerItems)-1)
		return
	}

	type scoredTemplateLibrary struct {
		item  templateLibraryPickerItem
		score int
	}
	scored := make([]scoredTemplateLibrary, 0, len(allItems))
	for _, item := range allItems {
		score, ok := bestFuzzyScore(
			query,
			item.LibraryID,
			item.Name,
			m.templateLibraryDisplayLabel(item.LibraryID, item.Name),
			"none clear remove unbind template library",
		)
		if !ok {
			continue
		}
		scored = append(scored, scoredTemplateLibrary{
			item:  item,
			score: score,
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		left := m.templateLibraryDisplayLabel(scored[i].item.LibraryID, scored[i].item.Name)
		right := m.templateLibraryDisplayLabel(scored[j].item.LibraryID, scored[j].item.Name)
		return left < right
	})
	m.templateLibraryPickerItems = make([]templateLibraryPickerItem, 0, len(scored))
	for _, entry := range scored {
		m.templateLibraryPickerItems = append(m.templateLibraryPickerItems, entry.item)
	}
	m.templateLibraryPickerIndex = clamp(m.templateLibraryPickerIndex, 0, len(m.templateLibraryPickerItems)-1)
}

// startTemplateLibraryPicker opens the project template-library picker with optional initial filter text.
func (m *Model) startTemplateLibraryPicker(seed string) tea.Cmd {
	if m == nil {
		return nil
	}
	m.templateLibraryPickerBack = m.mode
	m.mode = modeTemplateLibraryPicker
	m.templateLibraryPickerInput.SetValue(strings.TrimSpace(seed))
	m.templateLibraryPickerInput.CursorEnd()
	m.refreshTemplateLibraryPickerMatches()
	current := domain.NormalizeTemplateLibraryID(m.projectFormInputs[projectFieldTemplateLibrary].Value())
	m.templateLibraryPickerIndex = 0
	if current != "" && strings.TrimSpace(seed) == "" {
		for idx, item := range m.templateLibraryPickerItems {
			if item.LibraryID == current {
				m.templateLibraryPickerIndex = idx
				break
			}
		}
	}
	if current == "" && strings.TrimSpace(seed) == "" {
		for idx, item := range m.templateLibraryPickerItems {
			if item.Clear {
				m.templateLibraryPickerIndex = idx
				break
			}
		}
	}
	if len(m.templateLibraryPickerItems) == 0 {
		m.status = "no approved template libraries"
	} else {
		m.status = "template library picker"
	}
	return m.templateLibraryPickerInput.Focus()
}

// templateActorKindsText renders actor-kind slices for readable TUI inspection output.
func templateActorKindsText(kinds []domain.TemplateActorKind) string {
	if len(kinds) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		value := strings.TrimSpace(string(kind))
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

// descriptionFormDisplayValue summarizes markdown description content for compact form rows.
func descriptionFormDisplayValue(markdown string) string {
	text := strings.TrimSpace(markdown)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	first := strings.TrimSpace(lines[0])
	if first == "" {
		first = "(markdown description)"
	}
	if len(lines) > 1 {
		return first + " …"
	}
	return first
}

// syncActionItemFormDescriptionDisplay keeps the actionItem-form description row as a compact markdown summary.
func (m *Model) syncActionItemFormDescriptionDisplay() {
	if m == nil || len(m.formInputs) <= actionItemFieldDescription {
		return
	}
	m.formInputs[actionItemFieldDescription].SetValue(descriptionFormDisplayValue(m.actionItemFormDescription))
	m.formInputs[actionItemFieldDescription].CursorEnd()
}

// syncProjectFormDescriptionDisplay keeps the project-form description row as a compact markdown summary.
func (m *Model) syncProjectFormDescriptionDisplay() {
	if m == nil || len(m.projectFormInputs) <= projectFieldDescription {
		return
	}
	m.projectFormInputs[projectFieldDescription].SetValue(descriptionFormDisplayValue(m.projectFormDescription))
	m.projectFormInputs[projectFieldDescription].CursorEnd()
}

// sanitizeFormFieldValue normalizes interactive form values and strips terminal probe artifacts.
func sanitizeFormFieldValue(value string) string {
	value = sanitizeInteractiveInputValue(value)
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.TrimSpace(value)
}

// sanitizeInteractiveInputValue strips terminal-probe artifacts and control runes while preserving user-entered spacing.
func sanitizeInteractiveInputValue(value string) string {
	value = stripTerminalProbeArtifacts(value)
	return stripDisallowedControlRunes(value)
}

// stripTerminalProbeArtifacts removes terminal OSC color-probe artifacts from freeform text.
func stripTerminalProbeArtifacts(value string) string {
	if value == "" {
		return ""
	}
	value = terminalProbeEscapeSequencePattern.ReplaceAllString(value, "")
	value = terminalProbeArtifactWithPrefixPattern.ReplaceAllString(value, "")
	value = terminalProbeArtifactPattern.ReplaceAllString(value, "")
	return value
}

// stripDisallowedControlRunes removes control runes that should never persist in actionItem/project form fields.
func stripDisallowedControlRunes(value string) string {
	if value == "" {
		return ""
	}
	var out strings.Builder
	out.Grow(len(value))
	for _, r := range value {
		if r == '\n' || r == '\t' {
			out.WriteRune(r)
			continue
		}
		if r < 0x20 || r == 0x7f {
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

// scrubTextInputTerminalArtifacts removes terminal-probe artifacts from one textinput model after key updates.
func scrubTextInputTerminalArtifacts(in *textinput.Model) bool {
	if in == nil {
		return false
	}
	before := in.Value()
	after := sanitizeInteractiveInputValue(before)
	if after == before {
		return false
	}
	cursor := clamp(in.Position(), 0, utf8.RuneCountInString(after))
	in.SetValue(after)
	in.SetCursor(cursor)
	return true
}

// scrubTextAreaTerminalArtifacts removes terminal-probe artifacts from one textarea model after key updates.
func scrubTextAreaTerminalArtifacts(in *textarea.Model) bool {
	if in == nil {
		return false
	}
	before := in.Value()
	after := sanitizeInteractiveInputValue(before)
	if after == before {
		return false
	}
	in.SetValue(after)
	return true
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

// formatSystemTimestamp renders one system metadata timestamp in local time for info views.
func formatSystemTimestamp(at *time.Time) string {
	if at == nil || at.IsZero() {
		return "-"
	}
	return at.In(time.Local).Format(time.RFC3339)
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

// parseActionItemRefIDsInput parses dependency reference ids from comma-separated actionItem-id input.
func parseActionItemRefIDsInput(raw string, current []string) []string {
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

// buildActionItemMetadataFromForm overlays dependency/resource actionItem metadata fields from form values.
func (m Model) buildActionItemMetadataFromForm(vals map[string]string, current domain.ActionItemMetadata) domain.ActionItemMetadata {
	meta := current
	meta.DependsOn = parseActionItemRefIDsInput(vals["depends_on"], current.DependsOn)
	meta.BlockedBy = parseActionItemRefIDsInput(vals["blocked_by"], current.BlockedBy)
	blockedReason := strings.TrimSpace(vals["blocked_reason"])
	switch {
	case actionItemFormUsesDedicatedMarkdownDraft(actionItemFieldBlockedReason) && m.actionItemFormTouched[actionItemFieldBlockedReason]:
		meta.BlockedReason = blockedReason
	case blockedReason == "":
		// Keep current metadata when field is untouched.
	case blockedReason == "-":
		meta.BlockedReason = ""
	default:
		meta.BlockedReason = blockedReason
	}
	objective := strings.TrimSpace(vals["objective"])
	switch {
	case actionItemFormUsesDedicatedMarkdownDraft(actionItemFieldObjective) && m.actionItemFormTouched[actionItemFieldObjective]:
		meta.Objective = objective
	case objective == "":
		// Keep current metadata when field is untouched.
	case objective == "-":
		meta.Objective = ""
	default:
		meta.Objective = objective
	}
	acceptanceCriteria := strings.TrimSpace(vals["acceptance_criteria"])
	switch {
	case actionItemFormUsesDedicatedMarkdownDraft(actionItemFieldAcceptanceCriteria) && m.actionItemFormTouched[actionItemFieldAcceptanceCriteria]:
		meta.AcceptanceCriteria = acceptanceCriteria
	case acceptanceCriteria == "":
		// Keep current metadata when field is untouched.
	case acceptanceCriteria == "-":
		meta.AcceptanceCriteria = ""
	default:
		meta.AcceptanceCriteria = acceptanceCriteria
	}
	validationPlan := strings.TrimSpace(vals["validation_plan"])
	switch {
	case actionItemFormUsesDedicatedMarkdownDraft(actionItemFieldValidationPlan) && m.actionItemFormTouched[actionItemFieldValidationPlan]:
		meta.ValidationPlan = validationPlan
	case validationPlan == "":
		// Keep current metadata when field is untouched.
	case validationPlan == "-":
		meta.ValidationPlan = ""
	default:
		meta.ValidationPlan = validationPlan
	}
	riskNotes := strings.TrimSpace(vals["risk_notes"])
	switch {
	case actionItemFormUsesDedicatedMarkdownDraft(actionItemFieldRiskNotes) && m.actionItemFormTouched[actionItemFieldRiskNotes]:
		meta.RiskNotes = riskNotes
	case riskNotes == "":
		// Keep current metadata when field is untouched.
	case riskNotes == "-":
		meta.RiskNotes = ""
	default:
		meta.RiskNotes = riskNotes
	}
	meta.ResourceRefs = append([]domain.ResourceRef(nil), m.actionItemFormResourceRefs...)
	return meta
}

// buildCurrentEditActionItemInput resolves one UpdateActionItemInput from the active edit-actionItem draft state.
func (m Model) buildCurrentEditActionItemInput() (app.UpdateActionItemInput, domain.ActionItem, error) {
	vals := m.actionItemFormValues()
	actionItemID := strings.TrimSpace(m.editingActionItemID)
	if actionItemID == "" {
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			return app.UpdateActionItemInput{}, domain.ActionItem{}, fmt.Errorf("no actionItem selected")
		}
		actionItemID = actionItem.ID
	}
	actionItem, ok := m.actionItemByID(actionItemID)
	if !ok {
		return app.UpdateActionItemInput{}, domain.ActionItem{}, fmt.Errorf("actionItem not found")
	}

	title := vals["title"]
	if title == "" {
		title = actionItem.Title
	}
	description := vals["description"]

	priority := domain.Priority(strings.ToLower(vals["priority"]))
	if priority == "" {
		priority = actionItem.Priority
	}
	switch priority {
	case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
	default:
		return app.UpdateActionItemInput{}, domain.ActionItem{}, fmt.Errorf("priority must be low|medium|high")
	}

	dueAt, err := parseDueInput(vals["due"], actionItem.DueAt)
	if err != nil {
		return app.UpdateActionItemInput{}, domain.ActionItem{}, err
	}
	labels := parseLabelsInput(vals["labels"], actionItem.Labels)
	if err := m.validateAllowedLabels(labels); err != nil {
		return app.UpdateActionItemInput{}, domain.ActionItem{}, err
	}
	metadata := m.buildActionItemMetadataFromForm(vals, actionItem.Metadata)

	return app.UpdateActionItemInput{
		ActionItemID:  actionItemID,
		Title:         title,
		Description:   description,
		Priority:      priority,
		DueAt:         dueAt,
		Labels:        labels,
		Metadata:      &metadata,
		UpdatedBy:     m.threadActorID(),
		UpdatedByName: m.threadActorName(),
		UpdatedType:   m.threadActorType(),
	}, actionItem, nil
}

// persistCurrentEditActionItemCmd writes the active edit-actionItem draft and returns an update message.
func (m *Model) persistCurrentEditActionItemCmd(status string) (tea.Cmd, error) {
	if m == nil {
		return nil, fmt.Errorf("actionItem edit unavailable")
	}
	in, _, err := m.buildCurrentEditActionItemInput()
	if err != nil {
		return nil, err
	}
	reopenEditActionItemID := strings.TrimSpace(m.actionItemFormBackActionItemID)
	reselectChildID := strings.TrimSpace(m.editingActionItemID)
	if m.actionItemFormBackMode != modeEditActionItem {
		reopenEditActionItemID = ""
		reselectChildID = ""
	}
	svc := m.svc
	traceActionItemScreenAction(
		"task_edit",
		"persist_draft",
		"action_item_id", strings.TrimSpace(in.ActionItemID),
		"reopen_parent_action_item_id", reopenEditActionItemID,
		"reselect_child_id", reselectChildID,
	)
	return func() tea.Msg {
		updated, updateErr := svc.UpdateActionItem(context.Background(), in)
		if updateErr != nil {
			return actionMsg{err: updateErr}
		}
		return actionItemUpdatedMsg{
			actionItem:             updated,
			status:                 status,
			reopenEditActionItemID: reopenEditActionItemID,
			reselectChildID:        reselectChildID,
		}
	}, nil
}

// replaceActionItemInMemory updates one loaded actionItem in place without a full reload.
func (m *Model) replaceActionItemInMemory(updated domain.ActionItem) {
	if m == nil {
		return
	}
	for idx, existing := range m.tasks {
		if existing.ID != updated.ID {
			continue
		}
		m.tasks[idx] = updated
		return
	}
	m.tasks = append(m.tasks, updated)
}

// selectActionItemFormSubactionItemByID reanchors edit-mode subtask row selection to one stable child id.
func (m *Model) selectActionItemFormSubactionItemByID(actionItemID string) {
	if m == nil {
		return
	}
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		m.actionItemFormSubactionItemCursor = 0
		return
	}
	subtasks := m.actionItemFormContextSubtasks()
	for idx, child := range subtasks {
		if child.ID == actionItemID {
			m.actionItemFormSubactionItemCursor = idx + 1
			return
		}
	}
	m.actionItemFormSubactionItemCursor = 0
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
		level := canonicalSearchLevel(raw)
		if level == "" {
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

// canonicalSearchLevel case-folds one hierarchy level label and returns the canonical mixed-case form
// defined by canonicalSearchLevelsOrdered, or "" when the input is blank or unrecognized.
func canonicalSearchLevel(raw string) string {
	lowered := strings.TrimSpace(strings.ToLower(raw))
	if lowered == "" {
		return ""
	}
	for _, candidate := range canonicalSearchLevelsOrdered {
		if strings.ToLower(candidate) == lowered {
			return candidate
		}
	}
	return ""
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
	canonical := canonicalSearchLevel(level)
	if canonical == "" {
		return
	}
	levels := canonicalSearchLevels(m.searchLevels)
	next := make([]string, 0, len(levels))
	found := false
	for _, item := range levels {
		if item == canonical {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		next = append(next, canonical)
	}
	m.searchLevels = canonicalSearchLevels(next)
}

// isSearchLevelEnabled reports whether a search hierarchy level is currently enabled.
func (m Model) isSearchLevelEnabled(level string) bool {
	canonical := canonicalSearchLevel(level)
	if canonical == "" {
		return false
	}
	for _, item := range m.searchLevels {
		if canonicalSearchLevel(item) == canonical {
			return true
		}
	}
	return false
}

var searchModesOrdered = []app.SearchMode{
	app.SearchModeHybrid,
	app.SearchModeKeyword,
	app.SearchModeSemantic,
}

// cycleSearchMode rotates the active search execution mode within the operator-visible search modal.
func (m *Model) cycleSearchMode(delta int) {
	if m == nil {
		return
	}
	if len(searchModesOrdered) == 0 {
		m.searchMode = app.SearchModeHybrid
		return
	}
	current := app.SearchMode(strings.TrimSpace(string(m.searchMode)))
	index := 0
	for idx, mode := range searchModesOrdered {
		if mode == current {
			index = idx
			break
		}
	}
	m.searchMode = searchModesOrdered[wrapIndex(index, delta, len(searchModesOrdered))]
}

// searchModeDisplayLabel resolves one readable label for the requested operator search mode.
func searchModeDisplayLabel(mode app.SearchMode) string {
	switch mode {
	case app.SearchModeKeyword:
		return "keyword"
	case app.SearchModeSemantic:
		return "semantic"
	case app.SearchModeHybrid:
		return "hybrid"
	default:
		return "hybrid"
	}
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
	if isBackwardTabKey(msg) {
		return false
	}
	return msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i"
}

// isBackwardTabKey reports whether a key press should reverse panel/form focus.
func isBackwardTabKey(msg tea.KeyPressMsg) bool {
	return msg.String() == "shift+tab" || msg.String() == "backtab" || (msg.Code == tea.KeyTab && msg.Mod&tea.ModShift != 0)
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
		if m.hasGlobalNoticesPanel() {
			count++
		}
	}
	return count
}

// panelFocusIndex resolves the focused panel index across board columns and notices.
func (m Model) panelFocusIndex() int {
	if m.noticesFocused && m.isNoticesPanelVisible() {
		if m.noticesPanel == noticesPanelFocusGlobal && m.hasGlobalNoticesPanel() {
			return len(m.columns) + 1
		}
		return len(m.columns)
	}
	if len(m.columns) == 0 {
		return 0
	}
	return clamp(m.selectedColumn, 0, len(m.columns)-1)
}

// setPanelFocusIndex applies panel focus by index and returns true when focus changed.
func (m *Model) setPanelFocusIndex(idx int, resetActionItem bool) bool {
	total := m.panelFocusCount()
	if total <= 0 {
		m.noticesFocused = false
		return false
	}
	idx = clamp(idx, 0, total-1)
	current := m.panelFocusIndex()
	if m.isNoticesPanelVisible() {
		projectPanelIdx := len(m.columns)
		globalPanelIdx := len(m.columns) + 1
		switch idx {
		case projectPanelIdx:
			changed := !m.noticesFocused || current != idx
			m.noticesFocused = true
			m.noticesPanel = noticesPanelFocusProject
			m.clampNoticesSelection()
			return changed
		case globalPanelIdx:
			if !m.hasGlobalNoticesPanel() {
				return false
			}
			changed := !m.noticesFocused || current != idx
			m.noticesFocused = true
			m.noticesPanel = noticesPanelFocusGlobal
			m.clampGlobalNoticesSelection()
			return changed
		}
	}
	if len(m.columns) == 0 {
		m.noticesFocused = false
		return false
	}
	targetColumn := clamp(idx, 0, len(m.columns)-1)
	changed := m.noticesFocused || m.selectedColumn != targetColumn
	m.noticesFocused = false
	m.selectedColumn = targetColumn
	if resetActionItem && changed {
		m.selectedActionItem = 0
	}
	return changed
}

// cyclePanelFocus moves keyboard focus across panels.
func (m *Model) cyclePanelFocus(delta int, wrap bool, resetActionItem bool) bool {
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
	return m.setPanelFocusIndex(next, resetActionItem)
}

// normalizePanelFocus keeps panel focus and selections coherent after data/layout updates.
func (m *Model) normalizePanelFocus() {
	if len(m.columns) == 0 {
		m.noticesFocused = false
		m.selectedColumn = 0
		m.selectedActionItem = 0
		return
	}
	m.selectedColumn = clamp(m.selectedColumn, 0, len(m.columns)-1)
	if !m.isNoticesPanelVisible() {
		m.noticesFocused = false
	}
	if m.noticesFocused {
		if m.noticesPanel == noticesPanelFocusGlobal && m.hasGlobalNoticesPanel() {
			m.clampGlobalNoticesSelection()
		} else {
			m.noticesPanel = noticesPanelFocusProject
			m.clampNoticesSelection()
		}
	}
}

// noticesFocusStatus returns a status label for the active panel focus target.
func (m Model) noticesFocusStatus() string {
	return ""
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
	m.searchInput.Blur()
	m.searchQuery = strings.TrimSpace(m.searchInput.Value())
	m.searchStates = canonicalSearchStates(m.searchStates)
	m.searchLevels = canonicalSearchLevels(m.searchLevels)
	if m.searchMode == "" {
		m.searchMode = app.SearchModeHybrid
	}
	m.mode = modeSearchResults
	m.searchApplied = false
	m.searchResultIndex = 0
	m.searchMatches = nil
	m.searchRequestedMode = m.searchMode
	m.searchEffectiveMode = ""
	m.searchFallbackReason = ""
	m.searchEmbeddingSummary = app.EmbeddingSummary{}
	m.searchLoading = true
	m.searchOpeningResult = false
	m.searchRequestSeq++
	m.searchActiveRequestID = m.searchRequestSeq
	m.status = "searching..."
	return m.loadSearchMatchesCmd(m.searchActiveRequestID)
}

// clearSearchQuery clears only the search query.
func (m *Model) clearSearchQuery() tea.Cmd {
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchApplied = false
	m.searchLoading = false
	m.searchOpeningResult = false
	m.searchActiveRequestID = 0
	m.status = "query cleared"
	return m.loadData
}

// resetSearchFilters resets query and filters back to defaults.
func (m *Model) resetSearchFilters() tea.Cmd {
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchCrossProject = m.searchDefaultCrossProject
	m.searchIncludeArchived = m.searchDefaultIncludeArchive
	m.searchMode = m.searchDefaultMode
	m.searchStates = canonicalSearchStates(m.searchDefaultStates)
	m.searchLevels = canonicalSearchLevels(m.searchDefaultLevels)
	m.searchKinds = nil
	m.searchLabelsAny = nil
	m.searchLabelsAll = nil
	m.searchApplied = false
	m.searchLoading = false
	m.searchOpeningResult = false
	m.searchActiveRequestID = 0
	m.searchMatches = nil
	m.searchResultIndex = 0
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
	m.refreshActionItemFormLabelSuggestions()
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
		{Command: "new-actionItem", Aliases: []string{"actionItem-new"}, Description: "create a new actionItem"},
		{Command: "new-subtask", Aliases: []string{"actionItem-subtask", "ns"}, Description: "create subtask for selected item"},
		{Command: "new-branch", Aliases: []string{"branch-new"}, Description: "create a new branch"},
		{Command: "new-phase", Aliases: []string{"phase-new"}, Description: "create a new phase"},
		{Command: "edit-branch", Aliases: []string{"branch-edit"}, Description: "edit selected branch"},
		{Command: "archive-branch", Aliases: []string{"branch-archive"}, Description: "archive selected branch"},
		{Command: "delete-branch", Aliases: []string{"branch-delete"}, Description: "hard delete selected branch"},
		{Command: "restore-branch", Aliases: []string{"branch-restore"}, Description: "restore selected archived branch"},
		{Command: "edit-actionItem", Aliases: []string{"actionItem-edit"}, Description: "edit selected actionItem"},
		{Command: "thread-item", Aliases: []string{"item-thread", "actionItem-thread"}, Description: "open selected work-item thread"},
		{Command: "new-project", Aliases: []string{"project-new"}, Description: "create a new project"},
		{Command: "edit-project", Aliases: []string{"project-edit"}, Description: "edit selected project"},
		{Command: "archive-project", Aliases: []string{"project-archive"}, Description: "archive selected project"},
		{Command: "restore-project", Aliases: []string{"project-restore"}, Description: "restore selected archived project"},
		{Command: "delete-project", Aliases: []string{"project-delete"}, Description: "hard delete selected project"},
		{Command: "thread-project", Aliases: []string{"project-thread"}, Description: "open current project thread"},
		{Command: "search", Aliases: []string{}, Description: "open search modal"},
		{Command: "search-all", Aliases: []string{}, Description: "set search scope to all projects"},
		{Command: "search-project", Aliases: []string{}, Description: "set search scope to current project"},
		{Command: "embeddings", Aliases: []string{"embeddings-status", "semantic-status"}, Description: "open embeddings lifecycle inventory"},
		{Command: "embeddings-reindex", Aliases: []string{"semantic-reindex"}, Description: "enqueue embeddings reindex for the current inventory scope"},
		{Command: "clear-query", Aliases: []string{"clear-search-query"}, Description: "clear search text only"},
		{Command: "reset-filters", Aliases: []string{"clear-search"}, Description: "reset query + states + scope + archived"},
		{Command: "toggle-archived", Aliases: []string{}, Description: "toggle archived visibility"},
		{Command: "toggle-selection-mode", Aliases: []string{"select-mode", "text-select"}, Description: "toggle mouse text-selection mode"},
		{Command: "focus-subtree", Aliases: []string{"zoom-actionItem"}, Description: "show selected actionItem subtree only"},
		{Command: "focus-clear", Aliases: []string{"zoom-reset"}, Description: "return to full board view"},
		{Command: "toggle-select", Aliases: []string{"select-actionItem"}, Description: "toggle selected actionItem in multi-select"},
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
		{Command: "auth-access", Aliases: []string{"auths", "coordination", "recovery", "access-review"}, Description: "review coordination state; list requests, sessions, leases, and handoffs"},
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
	if initialism := commandPaletteInitialism(item.Command); initialism != "" {
		if strings.HasPrefix(initialism, query) {
			score = max(score, 4200-len(item.Command))
			ok = true
		}
	}
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

// commandPaletteInitialism derives a stable abbreviation from dash/space/underscore-separated command words.
func commandPaletteInitialism(value string) string {
	value = normalizeCommandPaletteToken(value)
	if value == "" {
		return ""
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	if len(parts) == 0 {
		return ""
	}
	var out strings.Builder
	out.Grow(len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		r, _ := utf8.DecodeRuneInString(part)
		if r == utf8.RuneError {
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

// normalizeCommandPaletteToken canonicalizes typed command ids so dash, underscore, and space variants match.
func normalizeCommandPaletteToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var out strings.Builder
	out.Grow(len(value))
	lastDash := false
	for _, r := range value {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			out.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if !lastDash {
				out.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(out.String(), "-")
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
	query = normalizeCommandPaletteToken(query)
	candidate = normalizeCommandPaletteToken(candidate)
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
	return normalizeCommandPaletteToken(m.commandInput.Value())
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
	if len(m.formInputs) > actionItemFieldDue {
		current := strings.TrimSpace(m.formInputs[actionItemFieldDue].Value())
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
	m.labelPickerAllItems = m.actionItemFormLabelPickerItems()
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

// startDependencyInspectorFromActionItemInfo opens dependency inspector for one existing actionItem.
func (m *Model) startDependencyInspectorFromActionItemInfo(actionItem domain.ActionItem) tea.Cmd {
	return m.startDependencyInspector(
		modeActionItemInfo,
		actionItem.ID,
		actionItem.Metadata.DependsOn,
		actionItem.Metadata.BlockedBy,
		actionItemFieldDependsOn,
	)
}

// startDependencyInspectorFromForm opens dependency inspector for actionItem-form dependency fields.
func (m *Model) startDependencyInspectorFromForm(activeField int) tea.Cmd {
	if activeField != actionItemFieldDependsOn && activeField != actionItemFieldBlockedBy {
		activeField = actionItemFieldDependsOn
	}
	back := m.mode
	ownerActionItemID := strings.TrimSpace(m.editingActionItemID)
	dependsOn := []string{}
	blockedBy := []string{}
	if len(m.formInputs) > actionItemFieldDependsOn {
		dependsOn = parseActionItemRefIDsInput(m.formInputs[actionItemFieldDependsOn].Value(), nil)
	}
	if len(m.formInputs) > actionItemFieldBlockedBy {
		blockedBy = parseActionItemRefIDsInput(m.formInputs[actionItemFieldBlockedBy].Value(), nil)
	}
	return m.startDependencyInspector(back, ownerActionItemID, dependsOn, blockedBy, activeField)
}

// startDependencyInspector initializes the dependency inspector modal state.
func (m *Model) startDependencyInspector(back inputMode, ownerActionItemID string, dependsOn, blockedBy []string, activeField int) tea.Cmd {
	if activeField != actionItemFieldDependsOn && activeField != actionItemFieldBlockedBy {
		activeField = actionItemFieldDependsOn
	}
	m.dependencyBack = back
	m.dependencyOwnerActionItemID = strings.TrimSpace(ownerActionItemID)
	m.dependencyDependsOn = sanitizeDependencyIDs(dependsOn, m.dependencyOwnerActionItemID)
	m.dependencyBlockedBy = sanitizeDependencyIDs(blockedBy, m.dependencyOwnerActionItemID)
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
	result, err := m.svc.SearchActionItems(ctx, app.SearchActionItemsFilter{
		ProjectID:       projectID,
		Query:           strings.TrimSpace(m.dependencyInput.Value()),
		CrossProject:    m.dependencyCrossProject,
		IncludeArchived: m.dependencyIncludeArchived,
		States:          append([]string(nil), m.dependencyStates...),
		Levels:          canonicalSearchLevels(m.searchDefaultLevels),
		Mode:            app.SearchModeHybrid,
		Sort:            app.SearchSortRankDesc,
		Limit:           defaultSearchResultsLimit,
		Offset:          0,
	})
	if err != nil {
		return dependencyMatchesMsg{err: err}
	}
	matches := result.Matches

	knownByProject := map[string]map[string]domain.ActionItem{}
	loadActionItemsByProject := func(projectID string) (map[string]domain.ActionItem, error) {
		if existing, ok := knownByProject[projectID]; ok {
			return existing, nil
		}
		tasks, listErr := m.svc.ListActionItems(ctx, projectID, true)
		if listErr != nil {
			return nil, listErr
		}
		byID := make(map[string]domain.ActionItem, len(tasks))
		for _, actionItem := range tasks {
			byID[actionItem.ID] = actionItem
		}
		knownByProject[projectID] = byID
		return byID, nil
	}

	candidateByID := map[string]dependencyCandidate{}
	searchOrder := make([]string, 0, len(matches))
	ownerActionItemID := strings.TrimSpace(m.dependencyOwnerActionItemID)
	for _, match := range matches {
		actionItemID := strings.TrimSpace(match.ActionItem.ID)
		if actionItemID == "" {
			continue
		}
		if ownerActionItemID != "" && actionItemID == ownerActionItemID {
			continue
		}
		if _, ok := candidateByID[actionItemID]; ok {
			continue
		}
		tasksByID, listErr := loadActionItemsByProject(match.Project.ID)
		if listErr != nil {
			return dependencyMatchesMsg{err: listErr}
		}
		candidateByID[actionItemID] = dependencyCandidate{
			Match: match,
			Path:  buildDependencyActionItemPath(match, tasksByID),
		}
		searchOrder = append(searchOrder, actionItemID)
	}

	linkedIDs := append([]string(nil), m.dependencyDependsOn...)
	linkedIDs = append(linkedIDs, m.dependencyBlockedBy...)
	linkedIDs = sanitizeDependencyIDs(linkedIDs, ownerActionItemID)
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
				tasksByID, actionItemErr := loadActionItemsByProject(projectID)
				if actionItemErr != nil {
					return dependencyMatchesMsg{err: actionItemErr}
				}
				actionItem, ok := tasksByID[linkedID]
				if !ok {
					continue
				}
				match := app.ActionItemMatch{
					Project:    project,
					ActionItem: actionItem,
					StateID:    dependencyStateIDForActionItem(actionItem),
				}
				candidateByID[linkedID] = dependencyCandidate{
					Match: match,
					Path:  buildDependencyActionItemPath(match, tasksByID),
				}
				found = true
				break
			}
			if found {
				continue
			}
			candidateByID[linkedID] = dependencyCandidate{
				Match: app.ActionItemMatch{
					Project: domain.Project{ID: "missing", Name: "(missing)"},
					ActionItem: domain.ActionItem{
						ID:    linkedID,
						Title: "(missing actionItem reference)",
						Kind:  domain.KindActionItem,
					},
					StateID: "missing",
				},
				Path: "(missing actionItem reference)",
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
	for _, actionItemID := range searchOrder {
		if _, ok := linkedSet[actionItemID]; ok {
			continue
		}
		if candidate, ok := candidateByID[actionItemID]; ok {
			candidates = append(candidates, candidate)
		}
	}
	return dependencyMatchesMsg{candidates: candidates}
}

// buildDependencyActionItemPath formats project + hierarchy path context for one dependency candidate.
func buildDependencyActionItemPath(match app.ActionItemMatch, tasksByID map[string]domain.ActionItem) string {
	pathParts := []string{}
	current := match.ActionItem
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
		pathParts = append(pathParts, fmt.Sprintf("%s:%s", match.ActionItem.Kind, match.ActionItem.Title))
	}
	projectName := strings.TrimSpace(match.Project.Name)
	if projectName == "" {
		projectName = match.Project.ID
	}
	return projectName + " | " + strings.Join(pathParts, " | ")
}

// searchResultsStatusSummary returns the status-line text for one search result payload.
func searchResultsStatusSummary(matches []app.ActionItemMatch, requested, effective app.SearchMode, fallbackReason string) string {
	if len(matches) == 0 {
		if strings.TrimSpace(fallbackReason) != "" {
			return "no matches • fallback: " + strings.TrimSpace(fallbackReason)
		}
		return "no matches"
	}
	modeLabel := searchResultsModeLabel(requested, effective, fallbackReason)
	if strings.TrimSpace(modeLabel) == "" {
		return fmt.Sprintf("%d matches", len(matches))
	}
	return fmt.Sprintf("%d matches • %s", len(matches), modeLabel)
}

// searchResultsModeLabel returns one readable requested/effective mode label for the TUI search overlay.
func searchResultsModeLabel(requested, effective app.SearchMode, fallbackReason string) string {
	requestedText := strings.TrimSpace(string(requested))
	effectiveText := strings.TrimSpace(string(effective))
	switch {
	case requestedText == "" && effectiveText == "":
		return "mode: hybrid"
	case requestedText != "" && effectiveText != "" && requestedText != effectiveText:
		label := fmt.Sprintf("mode: %s -> %s", requestedText, effectiveText)
		if strings.TrimSpace(fallbackReason) != "" {
			label += " • fallback: " + strings.TrimSpace(fallbackReason)
		}
		return label
	case effectiveText != "":
		label := "mode: " + effectiveText
		if strings.TrimSpace(fallbackReason) != "" {
			label += " • fallback: " + strings.TrimSpace(fallbackReason)
		}
		return label
	case requestedText != "":
		return "mode: " + requestedText
	default:
		return ""
	}
}

// searchResultsEmbeddingSummaryLabel returns one compact lifecycle-summary label for the TUI search overlay.
func searchResultsEmbeddingSummaryLabel(summary app.EmbeddingSummary) string {
	return fmt.Sprintf(
		"embeddings ready:%d pending:%d running:%d failed:%d stale:%d",
		summary.ReadyCount,
		summary.PendingCount,
		summary.RunningCount,
		summary.FailedCount,
		summary.StaleCount,
	)
}

// searchMatchEmbeddingLabel returns one short per-match lifecycle/search label for the TUI search overlay.
func searchMatchEmbeddingLabel(match app.ActionItemMatch) string {
	status := strings.TrimSpace(string(match.EmbeddingStatus))
	if status == "" {
		status = "untracked"
	}
	if subjectType := strings.TrimSpace(string(match.EmbeddingSubjectType)); subjectType != "" && subjectType != string(app.EmbeddingSubjectTypeWorkItem) {
		status += "/" + subjectType
	}
	if match.UsedSemantic {
		return status + "/semantic"
	}
	return status
}

// embeddingsStatusScopeLabel returns one compact named scope label for the embeddings inventory modal.
func embeddingsStatusScopeLabel(projectIDs []string, projectsByID map[string]domain.Project) string {
	projectIDs = uniqueTrimmed(projectIDs)
	if len(projectIDs) == 0 {
		return "none"
	}
	names := make([]string, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		if project, ok := projectsByID[projectID]; ok {
			names = append(names, firstNonEmptyTrimmed(projectDisplayName(project), projectID))
			continue
		}
		names = append(names, projectID)
	}
	switch len(names) {
	case 1:
		return names[0]
	case 2, 3:
		return "all projects (" + strconv.Itoa(len(names)) + "): " + strings.Join(names, ", ")
	default:
		return fmt.Sprintf("all projects (%d): %s +%d more", len(names), strings.Join(names[:3], ", "), len(names)-3)
	}
}

// embeddingsStatusDetailLabel returns one compact operator-visible detail label for an embeddings lifecycle row.
func embeddingsStatusDetailLabel(row app.EmbeddingRecord) string {
	switch {
	case strings.TrimSpace(row.StaleReason) != "":
		return row.StaleReason
	case strings.TrimSpace(row.LastErrorSummary) != "":
		return row.LastErrorSummary
	case strings.TrimSpace(row.ModelSignature) != "":
		return row.ModelSignature
	default:
		return "-"
	}
}

// embeddingsStatusSubjectLabel resolves one human-facing subject label for an embeddings inventory row.
func embeddingsStatusSubjectLabel(row app.EmbeddingRecord, actionItem domain.ActionItem, hasActionItem bool) string {
	switch row.SubjectType {
	case app.EmbeddingSubjectTypeWorkItem:
		if hasActionItem {
			if marker := strings.TrimSpace(actionItemHierarchyMarker(actionItem)); marker != "" {
				return marker
			}
			if kind := strings.TrimSpace(string(actionItem.Kind)); kind != "" {
				return kind
			}
		}
		return "work item"
	case app.EmbeddingSubjectTypeThreadContext:
		if hasActionItem {
			if marker := strings.TrimSpace(actionItemHierarchyMarker(actionItem)); marker != "" {
				return marker + " thread"
			}
			if kind := strings.TrimSpace(string(actionItem.Kind)); kind != "" {
				return kind + " thread"
			}
			return "actionItem thread"
		}
		return "project thread"
	case app.EmbeddingSubjectTypeProjectDocument:
		return "project document"
	default:
		value := strings.TrimSpace(strings.ReplaceAll(string(row.SubjectType), "_", " "))
		if value == "" {
			return "subject"
		}
		return value
	}
}

// embeddingsStatusTitleLabel resolves one row title with stable subject-id fallback text.
func embeddingsStatusTitleLabel(row app.EmbeddingRecord, project domain.Project, hasProject bool, actionItem domain.ActionItem, hasActionItem bool) string {
	switch row.SubjectType {
	case app.EmbeddingSubjectTypeWorkItem:
		if hasActionItem {
			return firstNonEmptyTrimmed(actionItem.Title, actionItem.ID)
		}
	case app.EmbeddingSubjectTypeThreadContext:
		if hasActionItem {
			return firstNonEmptyTrimmed(actionItem.Title, actionItem.ID)
		}
		if hasProject {
			return "comments"
		}
	case app.EmbeddingSubjectTypeProjectDocument:
		if hasProject {
			return "project overview"
		}
	}
	return firstNonEmptyTrimmed(row.SubjectID, "(unknown)")
}

// embeddingsStatusActionItemPath formats one project-rooted hierarchy path for lifecycle rows.
func embeddingsStatusActionItemPath(actionItem domain.ActionItem, tasksByID map[string]domain.ActionItem, projectLabel string) string {
	chain := []string{firstNonEmptyTrimmed(actionItem.Title, actionItem.ID)}
	visited := map[string]struct{}{strings.TrimSpace(actionItem.ID): {}}
	parentID := strings.TrimSpace(actionItem.ParentID)
	for parentID != "" {
		if _, seen := visited[parentID]; seen {
			break
		}
		parent, ok := tasksByID[parentID]
		if !ok {
			break
		}
		visited[parentID] = struct{}{}
		chain = append(chain, firstNonEmptyTrimmed(parent.Title, parent.ID))
		parentID = strings.TrimSpace(parent.ParentID)
	}
	slices.Reverse(chain)
	if strings.TrimSpace(projectLabel) != "" {
		chain = append([]string{projectLabel}, chain...)
	}
	return strings.Join(chain, " -> ")
}

// buildEmbeddingsStatusDisplayRow resolves one lifecycle record into human-facing modal labels.
func buildEmbeddingsStatusDisplayRow(row app.EmbeddingRecord, projectsByID map[string]domain.Project, tasksByID map[string]domain.ActionItem) embeddingsStatusDisplayRow {
	display := embeddingsStatusDisplayRow{Record: row}
	if project, ok := projectsByID[strings.TrimSpace(row.ProjectID)]; ok {
		display.Project = project
		display.HasProject = true
		display.ProjectLabel = firstNonEmptyTrimmed(projectDisplayName(project), row.ProjectID)
	} else {
		display.ProjectLabel = firstNonEmptyTrimmed(row.ProjectID, "(project)")
	}

	switch row.SubjectType {
	case app.EmbeddingSubjectTypeWorkItem:
		if actionItem, ok := tasksByID[strings.TrimSpace(row.SubjectID)]; ok {
			display.ActionItem = actionItem
			display.HasActionItem = true
		}
	case app.EmbeddingSubjectTypeThreadContext:
		if target, err := app.ParseThreadContextSubjectID(row.SubjectID); err == nil && target.TargetType != domain.CommentTargetTypeProject {
			if actionItem, ok := tasksByID[strings.TrimSpace(target.TargetID)]; ok {
				display.ActionItem = actionItem
				display.HasActionItem = true
			}
		}
	}

	display.SubjectLabel = embeddingsStatusSubjectLabel(row, display.ActionItem, display.HasActionItem)
	display.TitleLabel = embeddingsStatusTitleLabel(row, display.Project, display.HasProject, display.ActionItem, display.HasActionItem)
	switch {
	case display.HasActionItem:
		display.PathLabel = embeddingsStatusActionItemPath(display.ActionItem, tasksByID, display.ProjectLabel)
	case display.HasProject:
		display.PathLabel = display.ProjectLabel
	default:
		display.PathLabel = firstNonEmptyTrimmed(row.ProjectID, row.SubjectID)
	}
	display.DetailLabel = embeddingsStatusDetailLabel(row)
	display.FilterLabel = strings.ToLower(strings.Join([]string{
		display.ProjectLabel,
		display.SubjectLabel,
		display.TitleLabel,
		display.PathLabel,
		display.DetailLabel,
		string(row.Status),
		string(row.SubjectType),
		row.SubjectID,
		row.ProjectID,
	}, " "))
	return display
}

// dependencyStateIDForActionItem resolves one canonical state identifier for dependency rows.
func dependencyStateIDForActionItem(actionItem domain.ActionItem) string {
	if actionItem.ArchivedAt != nil {
		return "archived"
	}
	if stateID := normalizeColumnStateID(string(actionItem.LifecycleState)); stateID != "" {
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
func hasDependencyID(ids []string, actionItemID string) bool {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return false
	}
	for _, id := range ids {
		if strings.TrimSpace(id) == actionItemID {
			return true
		}
	}
	return false
}

// toggleDependencyID adds/removes one id from a dependency-id slice.
func toggleDependencyID(ids []string, actionItemID string) ([]string, bool) {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return uniqueTrimmed(ids), false
	}
	out := make([]string, 0, len(ids))
	removed := false
	for _, id := range uniqueTrimmed(ids) {
		if id == actionItemID {
			removed = true
			continue
		}
		out = append(out, id)
	}
	if removed {
		return out, false
	}
	out = append(out, actionItemID)
	return uniqueTrimmed(out), true
}

// sanitizeDependencyIDs canonicalizes ids and removes any self-reference entry.
func sanitizeDependencyIDs(ids []string, ownerActionItemID string) []string {
	ownerActionItemID = strings.TrimSpace(ownerActionItemID)
	cleaned := make([]string, 0, len(ids))
	for _, id := range uniqueTrimmed(ids) {
		if id == "" {
			continue
		}
		if ownerActionItemID != "" && id == ownerActionItemID {
			continue
		}
		cleaned = append(cleaned, id)
	}
	return cleaned
}

// dependencyActiveFieldLabel returns the dependency field label currently targeted by space-toggle actions.
func (m Model) dependencyActiveFieldLabel() string {
	if m.dependencyActiveField == actionItemFieldBlockedBy {
		return "blocked_by"
	}
	return "depends_on"
}

// toggleDependencyCandidateInActiveField toggles highlighted actionItem id in the active dependency field.
func (m *Model) toggleDependencyCandidateInActiveField(actionItemID string) {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return
	}
	if ownerActionItemID := strings.TrimSpace(m.dependencyOwnerActionItemID); ownerActionItemID != "" && actionItemID == ownerActionItemID {
		m.status = "actionItem cannot depend on itself"
		return
	}
	if m.dependencyActiveField == actionItemFieldBlockedBy {
		var added bool
		m.dependencyBlockedBy, added = toggleDependencyID(m.dependencyBlockedBy, actionItemID)
		m.dependencyDirty = true
		if added {
			m.status = "added blocker"
		} else {
			m.status = "removed blocker"
		}
		return
	}
	var added bool
	m.dependencyDependsOn, added = toggleDependencyID(m.dependencyDependsOn, actionItemID)
	m.dependencyDirty = true
	if added {
		m.status = "added dependency"
	} else {
		m.status = "removed dependency"
	}
}

// applyDependencyInspector commits dependency selections and returns to the originating mode.
func (m Model) applyDependencyInspector() (tea.Model, tea.Cmd) {
	dependsOn := sanitizeDependencyIDs(m.dependencyDependsOn, m.dependencyOwnerActionItemID)
	blockedBy := sanitizeDependencyIDs(m.dependencyBlockedBy, m.dependencyOwnerActionItemID)
	back := m.dependencyBack
	activeField := m.dependencyActiveField
	if activeField != actionItemFieldDependsOn && activeField != actionItemFieldBlockedBy {
		activeField = actionItemFieldDependsOn
	}

	m.dependencyInput.Blur()
	m.dependencyDirty = false

	switch back {
	case modeAddActionItem, modeEditActionItem:
		if len(m.formInputs) > actionItemFieldDependsOn {
			m.formInputs[actionItemFieldDependsOn].SetValue(strings.Join(dependsOn, ","))
		}
		if len(m.formInputs) > actionItemFieldBlockedBy {
			m.formInputs[actionItemFieldBlockedBy].SetValue(strings.Join(blockedBy, ","))
		}
		m.mode = back
		m.status = "dependencies updated"
		if activeField == actionItemFieldBlockedBy {
			return m, m.focusActionItemFormField(actionItemFieldBlockedBy)
		}
		return m, m.focusActionItemFormField(actionItemFieldDependsOn)
	case modeActionItemInfo:
		actionItemID := strings.TrimSpace(m.dependencyOwnerActionItemID)
		actionItem, ok := m.actionItemByID(actionItemID)
		if !ok {
			m.mode = modeActionItemInfo
			m.actionItemInfoActionItemID = actionItemID
			m.status = "actionItem not found"
			return m, nil
		}
		meta := actionItem.Metadata
		meta.DependsOn = dependsOn
		meta.BlockedBy = blockedBy
		m.mode = modeActionItemInfo
		m.actionItemInfoActionItemID = actionItemID
		m.status = "saving dependencies..."
		return m, m.updateActionItemMetadataCmd(actionItem, meta, "dependencies updated")
	default:
		m.mode = modeNone
		m.status = "dependencies updated"
		return m, nil
	}
}

// jumpToDependencyCandidateActionItem closes dependency inspector and opens actionItem-info for the highlighted candidate.
func (m Model) jumpToDependencyCandidateActionItem() (tea.Model, tea.Cmd) {
	if m.dependencyBack != modeActionItemInfo {
		m.status = "jump to actionItem is available from actionItem-info inspector"
		return m, nil
	}
	candidate, ok := m.selectedDependencyCandidate()
	if !ok {
		m.status = "no dependency selected"
		return m, nil
	}
	actionItemID := strings.TrimSpace(candidate.Match.ActionItem.ID)
	if actionItemID == "" {
		m.status = "no dependency selected"
		return m, nil
	}
	for idx, project := range m.projects {
		if project.ID == candidate.Match.Project.ID {
			m.selectedProject = idx
			break
		}
	}
	m.pendingFocusActionItemID = actionItemID
	m.mode = modeActionItemInfo
	m.actionItemInfoActionItemID = actionItemID
	m.trackActionItemInfoPath(actionItemID)
	m.actionItemInfoDetails.SetYOffset(0)
	m.actionItemInfoBody.SetYOffset(0)
	if actionItem, ok := m.actionItemByID(actionItemID); ok {
		m.syncActionItemInfoDetailsViewport(actionItem)
		m.syncActionItemInfoBodyViewport(actionItem)
	}
	m.dependencyInput.Blur()
	m.status = "jumping to dependency"
	return m, m.loadData
}

// updateActionItemMetadataCmd persists one metadata update for the provided actionItem fields.
func (m Model) updateActionItemMetadataCmd(actionItem domain.ActionItem, metadata domain.ActionItemMetadata, status string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.svc.UpdateActionItem(context.Background(), app.UpdateActionItemInput{
			ActionItemID:  actionItem.ID,
			Title:         actionItem.Title,
			Description:   actionItem.Description,
			Priority:      actionItem.Priority,
			DueAt:         actionItem.DueAt,
			Labels:        append([]string(nil), actionItem.Labels...),
			Metadata:      &metadata,
			UpdatedBy:     m.threadActorID(),
			UpdatedByName: m.threadActorName(),
			UpdatedType:   m.threadActorType(),
		})
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{
			status:            status,
			reload:            true,
			focusActionItemID: actionItem.ID,
		}
	}
}

// refreshActionItemFormLabelSuggestions refreshes actionItem-form label suggestions from inherited sources.
func (m *Model) refreshActionItemFormLabelSuggestions() {
	if len(m.formInputs) <= actionItemFieldLabels {
		return
	}
	suggestions := mergeUniqueLabels(
		mergeLabelSources(m.actionItemFormLabelSources()),
		m.labelSuggestions(24),
		defaultLabelSuggestionsSeed,
	)
	m.formInputs[actionItemFieldLabels].SetSuggestions(suggestions)
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

// actionItemFormLabelSources resolves label inheritance sources for the active actionItem form context.
func (m Model) actionItemFormLabelSources() labelInheritanceSources {
	actionItem, ok := m.selectedActionItemForLabelInheritance()
	if !ok {
		return m.labelSourcesForActionItem(domain.ActionItem{})
	}
	return m.labelSourcesForActionItem(actionItem)
}

// labelSourcesForActionItem resolves inherited labels for one actionItem or taskless project context.
func (m Model) labelSourcesForActionItem(actionItem domain.ActionItem) labelInheritanceSources {
	sources := labelInheritanceSources{
		Global: normalizeConfigLabels(m.allowedLabelGlobal),
	}
	if project, ok := m.currentProject(); ok {
		projectSlug := strings.TrimSpace(strings.ToLower(project.Slug))
		sources.Project = normalizeConfigLabels(m.allowedLabelProject[projectSlug])
	}
	if strings.TrimSpace(actionItem.ID) != "" {
		sources.Branch = m.labelsFromBranchAncestors(actionItem)
		sources.Phase = m.labelsFromPhaseAncestors(actionItem)
	}
	return sources
}

// selectedActionItemForLabelInheritance picks the best actionItem context for inherited label sources.
func (m Model) selectedActionItemForLabelInheritance() (domain.ActionItem, bool) {
	if strings.TrimSpace(m.editingActionItemID) != "" {
		if actionItem, ok := m.actionItemByID(m.editingActionItemID); ok {
			return actionItem, true
		}
	}
	if strings.TrimSpace(m.actionItemFormParentID) != "" {
		if actionItem, ok := m.actionItemByID(m.actionItemFormParentID); ok {
			return actionItem, true
		}
	}
	return m.selectedActionItemInCurrentColumn()
}

// labelsFromPhaseAncestors collects inherited labels from phase ancestors in parent-chain order.
func (m Model) labelsFromPhaseAncestors(actionItem domain.ActionItem) []string {
	out := make([]string, 0)
	seenLabels := map[string]struct{}{}
	visited := map[string]struct{}{}
	current := actionItem
	for strings.TrimSpace(current.ID) != "" {
		if _, seen := visited[current.ID]; seen {
			break
		}
		visited[current.ID] = struct{}{}
		if current.Kind == domain.KindPhase {
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
		parent, ok := m.actionItemByID(parentID)
		if !ok {
			break
		}
		current = parent
	}
	return out
}

// labelsFromBranchAncestors collects inherited labels from branch ancestors in parent-chain order.
func (m Model) labelsFromBranchAncestors(actionItem domain.ActionItem) []string {
	out := make([]string, 0)
	seenLabels := map[string]struct{}{}
	visited := map[string]struct{}{}
	current := actionItem
	for strings.TrimSpace(current.ID) != "" {
		if _, seen := visited[current.ID]; seen {
			break
		}
		visited[current.ID] = struct{}{}
		level := baseSearchLevelForActionItem(current)
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
		parent, ok := m.actionItemByID(parentID)
		if !ok {
			break
		}
		current = parent
	}
	return out
}

// actionItemFormLabelPickerItems builds source-tagged inherited labels for modal selection.
func (m Model) actionItemFormLabelPickerItems() []labelPickerItem {
	sources := m.actionItemFormLabelSources()
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

// appendActionItemFormLabel appends one normalized label to the form without duplicating entries.
func (m *Model) appendActionItemFormLabel(label string) {
	if len(m.formInputs) <= actionItemFieldLabels {
		return
	}
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		return
	}
	current := parseLabelsInput(m.formInputs[actionItemFieldLabels].Value(), nil)
	for _, existing := range current {
		if strings.EqualFold(strings.TrimSpace(existing), label) {
			return
		}
	}
	current = append(current, label)
	m.formInputs[actionItemFieldLabels].SetValue(strings.Join(current, ","))
}

// acceptCurrentLabelSuggestion applies the active autocomplete suggestion into the labels field.
func (m *Model) acceptCurrentLabelSuggestion() bool {
	if len(m.formInputs) <= actionItemFieldLabels {
		return false
	}
	suggestion := strings.TrimSpace(strings.ToLower(m.formInputs[actionItemFieldLabels].CurrentSuggestion()))
	if suggestion == "" {
		matches := m.formInputs[actionItemFieldLabels].MatchedSuggestions()
		if len(matches) == 0 {
			return false
		}
		suggestion = strings.TrimSpace(strings.ToLower(matches[0]))
	}
	if suggestion == "" {
		return false
	}

	raw := strings.TrimSpace(m.formInputs[actionItemFieldLabels].Value())
	if raw == "" || raw == "-" {
		m.formInputs[actionItemFieldLabels].SetValue(suggestion)
		m.formInputs[actionItemFieldLabels].CursorEnd()
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
	m.formInputs[actionItemFieldLabels].SetValue(strings.Join(labels, ","))
	m.formInputs[actionItemFieldLabels].CursorEnd()
	return true
}

// startResourcePicker opens filesystem resource selection for a actionItem.
func (m *Model) startResourcePicker(actionItemID string, back inputMode) tea.Cmd {
	actionItemID = strings.TrimSpace(actionItemID)
	root := ""
	switch back {
	case modeActionItemInfo, modeAddActionItem, modeEditActionItem:
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
	m.resourcePickerActionItemID = actionItemID
	if back != modeAddActionItem && back != modeEditActionItem {
		m.actionItemFormResourceEditIndex = -1
	}
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

// attachSelectedResourceEntry attaches the currently selected resource entry to the target actionItem.
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

	// ActionItem form attachment flow stages refs for create/edit submit.
	if back == modeAddActionItem || back == modeEditActionItem {
		normalizedPath, err := normalizeAttachmentPathWithinRoot(strings.TrimSpace(m.resourcePickerRoot), entry.Path)
		if err != nil {
			m.status = err.Error()
			m.actionItemFormResourceEditIndex = -1
			return m.focusActionItemFormField(m.formFocus)
		}
		entry.Path = normalizedPath
		ref := buildResourceRef(strings.TrimSpace(m.resourcePickerRoot), entry.Path, entry.IsDir)
		editIdx := m.actionItemFormResourceEditIndex
		m.actionItemFormResourceEditIndex = -1
		if editIdx >= 0 && editIdx < len(m.actionItemFormResourceRefs) {
			for idx, existing := range m.actionItemFormResourceRefs {
				if idx == editIdx {
					continue
				}
				existingLocation := strings.TrimSpace(strings.ToLower(existing.Location))
				candidateLocation := strings.TrimSpace(strings.ToLower(ref.Location))
				if existing.ResourceType == ref.ResourceType &&
					existing.PathMode == ref.PathMode &&
					existingLocation == candidateLocation {
					m.status = "resource already staged"
					return m.focusActionItemFormField(m.formFocus)
				}
			}
			nextRefs := append([]domain.ResourceRef(nil), m.actionItemFormResourceRefs...)
			nextRefs[editIdx] = ref
			m.actionItemFormResourceRefs = nextRefs
			m.status = "resource updated"
			return m.focusActionItemFormField(m.formFocus)
		}
		refs, added := appendResourceRefIfMissing(m.actionItemFormResourceRefs, ref)
		if !added {
			m.status = "resource already staged"
			return m.focusActionItemFormField(m.formFocus)
		}
		m.actionItemFormResourceRefs = refs
		m.status = "resource staged"
		return m.focusActionItemFormField(m.formFocus)
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

	// Existing actionItem-info path persists immediately to actionItem metadata.
	if _, err := normalizeAttachmentPathWithinRoot(strings.TrimSpace(m.resourcePickerRoot), entry.Path); err != nil {
		m.status = err.Error()
		return nil
	}
	m.status = "attaching resource..."
	return m.attachResourceEntry(entry.Path, entry.IsDir)
}

// attachResourceEntry persists one filesystem reference through actionItem metadata update.
func (m Model) attachResourceEntry(path string, isDir bool) tea.Cmd {
	actionItemID := strings.TrimSpace(m.resourcePickerActionItemID)
	root := strings.TrimSpace(m.resourcePickerRoot)
	return func() tea.Msg {
		normalizedPath, err := normalizeAttachmentPathWithinRoot(root, path)
		if err != nil {
			return actionMsg{status: err.Error()}
		}
		actionItem, ok := m.actionItemByID(actionItemID)
		if !ok {
			return actionMsg{status: "resource attach failed: actionItem not found"}
		}
		ref := buildResourceRef(root, normalizedPath, isDir)
		refs, added := appendResourceRefIfMissing(actionItem.Metadata.ResourceRefs, ref)
		if !added {
			return actionMsg{status: "resource already attached"}
		}
		meta := actionItem.Metadata
		meta.ResourceRefs = refs
		_, err = m.svc.UpdateActionItem(context.Background(), app.UpdateActionItemInput{
			ActionItemID:  actionItem.ID,
			Title:         actionItem.Title,
			Description:   actionItem.Description,
			Priority:      actionItem.Priority,
			DueAt:         actionItem.DueAt,
			Labels:        append([]string(nil), actionItem.Labels...),
			Metadata:      &meta,
			UpdatedBy:     m.threadActorID(),
			UpdatedByName: m.threadActorName(),
			UpdatedType:   m.threadActorType(),
		})
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{
			status:            "resource attached",
			reload:            true,
			focusActionItemID: actionItem.ID,
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
	return m.resourcePickerBrowseRoot()
}

// resourcePickerBrowseRoot returns a best-effort browse root for non-actionItem picker flows.
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

// summarizeActionItemRefs renders dependency IDs with known actionItem titles when available.
func (m Model) summarizeActionItemRefs(ids []string, maxItems int) string {
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
		if actionItem, ok := m.actionItemByID(id); ok && strings.TrimSpace(actionItem.Title) != "" {
			label = fmt.Sprintf("%s(%s)", id, truncate(actionItem.Title, 22))
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
	for _, actionItem := range m.tasks {
		if actionItem.ProjectID != projectID {
			continue
		}
		for _, label := range actionItem.Labels {
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
	m.status = "ready"
}

// startWarningModal opens a dismiss-only warning modal with short guidance text.
func (m *Model) startWarningModal(title, body string) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Warning"
	}
	m.warningTitle = title
	m.warningBody = strings.TrimSpace(body)
	m.warningReturnMode = m.mode
	m.mode = modeWarning
}

// closeWarningModal dismisses the warning modal and clears staged text.
func (m *Model) closeWarningModal() {
	m.warningTitle = ""
	m.warningBody = ""
	m.mode = m.warningReturnMode
	m.warningReturnMode = modeNone
}

// handleNormalModeKey handles normal mode key.
func (m Model) handleNormalModeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.diffModeToggle):
		return m.enterDiffMode()
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
			m.status = ""
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
		if strings.TrimSpace(m.projectionRootActionItemID) != "" {
			m.clearSubtreeFocus()
			m.status = "full board view"
			return m, nil
		}
		return m, nil
	case key.Matches(msg, m.keys.reload):
		m.status = "reloading..."
		return m, m.loadData
	case isForwardTabKey(msg):
		_ = m.cyclePanelFocus(1, true, true)
		m.status = ""
		return m, nil
	case isBackwardTabKey(msg):
		_ = m.cyclePanelFocus(-1, true, true)
		m.status = ""
		return m, nil
	case key.Matches(msg, m.keys.moveLeft):
		_ = m.cyclePanelFocus(-1, true, true)
		m.status = ""
		return m, nil
	case key.Matches(msg, m.keys.moveRight):
		_ = m.cyclePanelFocus(1, true, true)
		m.status = ""
		return m, nil
	case key.Matches(msg, m.keys.toggleSelectMode):
		m.toggleMouseSelectionMode()
		return m, nil
	}

	if next, cmd, handled := m.handleBoardGlobalNormalKey(msg); handled {
		return next, cmd
	}
	if m.noticesFocused {
		return m.handleNoticesPanelNormalKey(msg)
	}
	return m.handleBoardPanelNormalKey(msg)
}

// handleBoardGlobalNormalKey keeps board-wide entrypoints available even when notices own focus.
func (m Model) handleBoardGlobalNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.addActionItem):
		m.help.ShowAll = false
		return m, m.startActionItemForm(nil), true
	case key.Matches(msg, m.keys.newProject):
		m.help.ShowAll = false
		return m, m.startProjectForm(nil), true
	case key.Matches(msg, m.keys.search):
		m.help.ShowAll = false
		return m, m.startSearchMode(), true
	case key.Matches(msg, m.keys.projects):
		m.help.ShowAll = false
		m.mode = modeProjectPicker
		if len(m.projects) == 0 {
			m.projectPickerIndex = 0
		} else {
			m.projectPickerIndex = clamp(m.selectedProject, 0, len(m.projects)-1)
		}
		m.status = "project picker"
		return m, nil, true
	case key.Matches(msg, m.keys.commandPalette):
		m.help.ShowAll = false
		return m, m.startCommandPalette(), true
	default:
		return m, nil, false
	}
}

// handleNoticesPanelNormalKey handles board-mode input while notices panel owns focus.
func (m Model) handleNoticesPanelNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.noticesPanel == noticesPanelFocusGlobal {
		switch {
		case key.Matches(msg, m.keys.moveDown):
			m.moveGlobalNoticesSelection(1)
			m.status = ""
			return m, nil
		case key.Matches(msg, m.keys.moveUp):
			m.moveGlobalNoticesSelection(-1)
			m.status = ""
			return m, nil
		case msg.Code == 'a' || msg.String() == "a":
			if next, cmd, ok := m.beginSelectedAuthRequestDecision("approve"); ok {
				return next, cmd
			}
			return m, nil
		case msg.Code == 'd' || msg.String() == "d":
			if next, cmd, ok := m.beginSelectedAuthRequestDecision("deny"); ok {
				return next, cmd
			}
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			m.beginGlobalNoticeTransition(msg)
			return m.activateGlobalNoticesSelection()
		case msg.String() == "x":
			item, ok := m.selectedGlobalNoticesItem()
			if !ok || domain.NormalizeAttentionKind(item.AttentionKind) != domain.AttentionKindMention {
				return m, nil
			}
			return m, m.resolveGlobalNoticeAttentionCmd(item)
		case key.Matches(msg, m.keys.activityLog):
			return m, m.openActivityLog()
		default:
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.moveDown):
		m.moveNoticesSelection(1)
		m.status = ""
		return m, nil
	case key.Matches(msg, m.keys.moveUp):
		m.moveNoticesSelection(-1)
		m.status = ""
		return m, nil
	case msg.Code == 'a' || msg.String() == "a":
		if next, cmd, ok := m.beginSelectedAuthRequestDecision("approve"); ok {
			return next, cmd
		}
		return m, nil
	case msg.Code == 'd' || msg.String() == "d":
		if next, cmd, ok := m.beginSelectedAuthRequestDecision("deny"); ok {
			return next, cmd
		}
		return m, nil
	case msg.Code == tea.KeyEnter || msg.String() == "enter":
		return m.activateNoticesSelection()
	case msg.String() == "x":
		item, ok := m.selectedNoticesPanelItem()
		if !ok || domain.NormalizeAttentionKind(item.AttentionKind) != domain.AttentionKindMention {
			return m, nil
		}
		return m, m.resolveNoticeAttentionCmd(item)
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
		tasks := m.currentColumnActionItems()
		if len(tasks) > 0 && m.selectedActionItem < len(tasks)-1 {
			m.selectedActionItem++
		}
		return m, nil
	case key.Matches(msg, m.keys.moveUp):
		if m.selectedActionItem > 0 {
			m.selectedActionItem--
		}
		return m, nil
	case key.Matches(msg, m.keys.multiSelect):
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		if m.toggleActionItemSelection(actionItem.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(actionItem.Title, 28), len(m.selectedActionItemIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "select actionItem",
				Target:  actionItem.Title,
			})
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(actionItem.Title, 28), len(m.selectedActionItemIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "unselect actionItem",
				Target:  actionItem.Title,
			})
		}
		return m, nil
	case key.Matches(msg, m.keys.activityLog):
		return m, m.openActivityLog()
	case key.Matches(msg, m.keys.fileViewerToggle):
		return m.enterFileViewerMode()
	case key.Matches(msg, m.keys.undo):
		return m.undoLastMutation()
	case key.Matches(msg, m.keys.redo):
		return m.redoLastMutation()
	case key.Matches(msg, m.keys.addActionItem):
		m.help.ShowAll = false
		return m, m.startActionItemForm(nil)
	case key.Matches(msg, m.keys.newProject):
		m.help.ShowAll = false
		return m, m.startProjectForm(nil)
	case key.Matches(msg, m.keys.actionItemInfo):
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		m.help.ShowAll = false
		m.openActionItemInfo(actionItem.ID, "actionItem info")
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
	case key.Matches(msg, m.keys.editActionItem):
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		m.help.ShowAll = false
		return m, m.startActionItemForm(&actionItem)
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
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		if !m.activateSubtreeFocus(actionItem.ID) {
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
	case key.Matches(msg, m.keys.moveActionItemLeft):
		if len(m.selectedActionItemIDs) > 0 {
			return m.moveSelectedActionItems(-1)
		}
		return m.moveSelectedActionItem(-1)
	case key.Matches(msg, m.keys.moveActionItemRight):
		if len(m.selectedActionItemIDs) > 0 {
			return m.moveSelectedActionItems(1)
		}
		return m.moveSelectedActionItem(1)
	case key.Matches(msg, m.keys.deleteActionItem):
		return m.confirmDeleteAction(m.defaultDeleteMode, m.confirmDelete, "delete actionItem")
	case key.Matches(msg, m.keys.hardDeleteActionItem):
		return m.confirmDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete actionItem")
	case key.Matches(msg, m.keys.restoreActionItem):
		return m.confirmRestoreAction()
	case key.Matches(msg, m.keys.toggleArchived):
		m.showArchived = !m.showArchived
		if m.showArchived {
			m.status = "showing archived tasks"
		} else {
			m.status = "hiding archived tasks"
		}
		m.selectedActionItem = 0
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
	helpToggleDisabled := m.mode == modeDescriptionEditor &&
		m.descriptionEditorMode == descriptionEditorViewModeEdit &&
		key.Matches(msg, m.keys.toggleHelp)
	if key.Matches(msg, m.keys.toggleHelp) && !helpToggleDisabled {
		m.toggleHelpOverlay()
		return m, nil
	}
	if m.help.ShowAll {
		if msg.Code == tea.KeyEscape || msg.String() == "esc" {
			m.toggleHelpOverlay()
		}
		return m, nil
	}

	if m.mode == modeDiff {
		return m.handleDiffModeKey(msg)
	}

	if m.mode == modeFileViewer {
		return m.handleFileViewerModeKey(msg)
	}

	if m.mode == modeActivityEventInfo {
		switch {
		case msg.String() == "esc":
			m.mode = modeNone
			if m.isNoticesPanelVisible() {
				_ = m.setPanelFocusIndex(len(m.columns), false)
				m.status = ""
			} else {
				m.noticesFocused = false
				m.status = ""
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

	if m.mode == modeDescriptionEditor {
		if m.descriptionEditorMode == descriptionEditorViewModeEdit {
			if handled, status := applyClipboardShortcutToTextArea(msg, &m.descriptionEditorInput); handled {
				m.status = status
				m.syncDescriptionPreviewOffsetToEditor()
				return m, nil
			}
			switch {
			case key.Matches(msg, m.keys.undo):
				current := m.descriptionEditorInput.Value()
				if !applyTextEditUndo(&current, &m.descriptionEditorUndo, &m.descriptionEditorRedo) {
					m.status = "nothing to undo"
					return m, nil
				}
				m.descriptionEditorInput.SetValue(current)
				m.descriptionEditorInput.CursorEnd()
				m.syncDescriptionPreviewOffsetToEditor()
				m.status = "description undo"
				return m, nil
			case key.Matches(msg, m.keys.redo):
				current := m.descriptionEditorInput.Value()
				if !applyTextEditRedo(&current, &m.descriptionEditorUndo, &m.descriptionEditorRedo) {
					m.status = "nothing to redo"
					return m, nil
				}
				m.descriptionEditorInput.SetValue(current)
				m.descriptionEditorInput.CursorEnd()
				m.syncDescriptionPreviewOffsetToEditor()
				m.status = "description redo"
				return m, nil
			case msg.Code == tea.KeyEscape || msg.String() == "esc":
				return m, m.closeDescriptionEditor(false)
			case msg.String() == "ctrl+s":
				m.saveDescriptionEditor()
				return m, m.closeDescriptionEditor(true)
			case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i":
				m.descriptionEditorMode = descriptionEditorViewModePreview
				m.descriptionEditorInput.Blur()
				m.resetDescriptionPreviewToTop()
				m.help.ShowAll = false
				m.status = "previewing description"
				return m, nil
			default:
				var cmd tea.Cmd
				before := m.descriptionEditorInput.Value()
				m.descriptionEditorInput, cmd = m.descriptionEditorInput.Update(msg)
				_ = scrubTextAreaTerminalArtifacts(&m.descriptionEditorInput)
				pushTextEditHistory(&m.descriptionEditorUndo, &m.descriptionEditorRedo, before, m.descriptionEditorInput.Value())
				m.syncDescriptionPreviewOffsetToEditor()
				return m, cmd
			}
		}
		m.syncDescriptionEditorViewportLayout()
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			return m, m.closeDescriptionEditor(false)
		case msg.String() == "ctrl+s":
			m.saveDescriptionEditor()
			return m, m.closeDescriptionEditor(true)
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i":
			m.descriptionEditorMode = descriptionEditorViewModeEdit
			m.status = "editing description"
			m.syncDescriptionPreviewOffsetToEditor()
			m.help.ShowAll = false
			return m, m.descriptionEditorInput.Focus()
		case msg.String() == "j" || msg.String() == "down":
			m.descriptionPreview.ScrollDown(1)
			return m, nil
		case msg.String() == "k" || msg.String() == "up":
			m.descriptionPreview.ScrollUp(1)
			return m, nil
		case msg.Code == tea.KeyPgDown || msg.String() == "pgdown":
			m.descriptionPreview.PageDown()
			return m, nil
		case msg.Code == tea.KeyPgUp || msg.String() == "pgup":
			m.descriptionPreview.PageUp()
			return m, nil
		case msg.String() == "home":
			m.descriptionPreview.GotoTop()
			return m, nil
		case msg.String() == "end":
			m.descriptionPreview.GotoBottom()
			return m, nil
		default:
			return m, nil
		}
	}

	if m.mode == modeThread {
		if m.threadComposerActive {
			if handled, status := applyClipboardShortcutToTextArea(msg, &m.threadInput); handled {
				m.status = status
				return m, nil
			}
		}
		updateThreadComposerInput := func() (tea.Model, tea.Cmd) {
			var cmd tea.Cmd
			before := m.threadInput.Value()
			m.threadInput, cmd = m.threadInput.Update(msg)
			_ = scrubTextAreaTerminalArtifacts(&m.threadInput)
			pushTextEditHistory(&m.threadComposerUndo, &m.threadComposerRedo, before, m.threadInput.Value())
			return m, cmd
		}
		switch {
		case msg.String() == "i":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			if m.threadPanelFocus != threadPanelComments {
				return m, nil
			}
			m.threadComposerActive = true
			m.resetThreadComposerHistory()
			m.status = "ready"
			return m, m.threadInput.Focus()
		case isForwardTabKey(msg):
			if m.threadComposerActive {
				m.threadComposerActive = false
				m.threadInput.Blur()
				m.status = "ready"
				return m, m.focusThreadPanel(threadPanelComments)
			}
			return m, m.moveThreadPanelFocus(1)
		case isBackwardTabKey(msg):
			if m.threadComposerActive {
				m.threadComposerActive = false
				m.threadInput.Blur()
				m.status = "ready"
				return m, m.focusThreadPanel(threadPanelComments)
			}
			return m, m.moveThreadPanelFocus(-1)
		case msg.String() == "left":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			return m, m.moveThreadPanelFocus(-1)
		case msg.String() == "right":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			return m, m.moveThreadPanelFocus(1)
		case msg.String() == "up":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			if m.threadPanelFocus == threadPanelComments {
				m.threadScroll = max(0, m.threadScroll-1)
			}
			return m, nil
		case msg.String() == "down":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			if m.threadPanelFocus == threadPanelComments {
				m.threadScroll++
			}
			return m, nil
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			if m.threadComposerActive {
				m.threadComposerActive = false
				m.threadInput.Blur()
				m.resetThreadComposerHistory()
				m.status = "ready"
				return m, m.focusThreadPanel(threadPanelComments)
			}
			m.threadInput.Blur()
			m.threadDetailsInput.Blur()
			m.threadPendingCommentBody = ""
			m.threadDetailsActive = false
			m.resetThreadComposerHistory()
			if m.threadBackMode == modeActionItemInfo {
				m.mode = modeActionItemInfo
				m.loadActionItemInfoComments(m.actionItemInfoActionItemID)
				m.status = "actionItem info"
				return m, nil
			}
			if m.threadBackMode == modeEditActionItem {
				m.mode = modeEditActionItem
				m.loadActionItemInfoComments(strings.TrimSpace(m.editingActionItemID))
				m.status = "edit actionItem"
				return m, nil
			}
			if m.threadBackMode == modeEditProject {
				m.mode = modeEditProject
				m.status = "edit project"
				return m, nil
			}
			m.mode = modeNone
			m.status = "ready"
			return m, nil
		case key.Matches(msg, m.keys.undo):
			if !m.threadComposerActive {
				return m, nil
			}
			current := m.threadInput.Value()
			if !applyTextEditUndo(&current, &m.threadComposerUndo, &m.threadComposerRedo) {
				m.status = "nothing to undo"
				return m, nil
			}
			m.threadInput.SetValue(current)
			m.threadInput.CursorEnd()
			m.status = "comment undo"
			return m, nil
		case key.Matches(msg, m.keys.redo):
			if !m.threadComposerActive {
				return m, nil
			}
			current := m.threadInput.Value()
			if !applyTextEditRedo(&current, &m.threadComposerUndo, &m.threadComposerRedo) {
				m.status = "nothing to redo"
				return m, nil
			}
			m.threadInput.SetValue(current)
			m.threadInput.CursorEnd()
			m.status = "comment redo"
			return m, nil
		case msg.Code == tea.KeyPgUp || msg.String() == "pgup":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			m.threadScroll = max(0, m.threadScroll-max(1, m.threadViewportStep()))
			return m, nil
		case msg.Code == tea.KeyPgDown || msg.String() == "pgdown":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			m.threadScroll += max(1, m.threadViewportStep())
			return m, nil
		case msg.String() == "home":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			m.threadScroll = 0
			return m, nil
		case msg.String() == "end":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			m.threadScroll += 1000
			return m, nil
		case msg.String() == "ctrl+s":
			if !m.threadComposerActive {
				m.status = "press i to compose a comment"
				return m, nil
			}
			body := strings.TrimSpace(m.threadInput.Value())
			if body == "" {
				m.status = "comment body required"
				return m, nil
			}
			m.threadPendingCommentBody = body
			m.threadInput.SetValue("")
			m.threadInput.CursorEnd()
			m.resetThreadComposerHistory()
			m.status = "posting comment..."
			return m, m.createThreadCommentCmd(body)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			if m.threadComposerActive {
				return updateThreadComposerInput()
			}
			switch m.threadPanelFocus {
			case threadPanelDetails:
				return m.startThreadEditFlow()
			case threadPanelComments:
				m.threadComposerActive = true
				m.resetThreadComposerHistory()
				m.status = "ready"
				return m, m.threadInput.Focus()
			default:
				return m, nil
			}
		default:
			if !m.threadComposerActive {
				return m, nil
			}
			return updateThreadComposerInput()
		}
	}

	if m.mode == modeActionItemInfo {
		actionItem, ok := m.actionItemInfoActionItem()
		if !ok {
			m.closeActionItemInfo("actionItem info unavailable")
			return m, nil
		}
		m.syncActionItemInfoDetailsViewport(actionItem)
		m.syncActionItemInfoBodyViewport(actionItem)
		subtasks := m.subtasksForParent(actionItem.ID)
		switch {
		case key.Matches(msg, m.keys.quickActions):
			return m, m.startQuickActions()
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			if m.stepBackActionItemInfoPath() {
				return m, nil
			}
			m.closeActionItemInfo("ready")
			return m, nil
		case msg.String() == "i":
			m.closeActionItemInfo("ready")
			return m, nil
		case msg.Code == tea.KeyPgDown || msg.String() == "pgdown" || msg.String() == "ctrl+d":
			step := max(1, m.actionItemInfoBody.Height()/2)
			m.actionItemInfoBody.ScrollDown(step)
			return m, nil
		case msg.Code == tea.KeyPgUp || msg.String() == "pgup" || msg.String() == "ctrl+u":
			step := max(1, m.actionItemInfoBody.Height()/2)
			m.actionItemInfoBody.ScrollUp(step)
			return m, nil
		case msg.String() == "home":
			m.actionItemInfoDetails.GotoTop()
			m.actionItemInfoBody.GotoTop()
			m.syncActionItemInfoBodyViewport(actionItem)
			return m, nil
		case msg.String() == "end":
			m.actionItemInfoBody.GotoBottom()
			return m, nil
		case msg.String() == "d":
			return m, m.startActionItemInfoDescriptionEditor(actionItem)
		case msg.String() == "j" || msg.String() == "down":
			m.actionItemInfoBody.ScrollDown(1)
			if len(subtasks) > 0 && m.actionItemInfoSubactionItemIdx < len(subtasks)-1 {
				m.actionItemInfoSubactionItemIdx++
				m.actionItemInfoFocusedSubactionItemID = subtasks[m.actionItemInfoSubactionItemIdx].ID
			}
			return m, nil
		case msg.String() == "k" || msg.String() == "up":
			m.actionItemInfoBody.ScrollUp(1)
			if m.actionItemInfoSubactionItemIdx > 0 {
				m.actionItemInfoSubactionItemIdx--
				if m.actionItemInfoSubactionItemIdx < len(subtasks) {
					m.actionItemInfoFocusedSubactionItemID = subtasks[m.actionItemInfoSubactionItemIdx].ID
				}
			}
			return m, nil
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			return m, m.openFocusedActionItemInfoSubtask(actionItem)
		case msg.Code == tea.KeyBackspace || msg.String() == "backspace" || msg.String() == "h" || msg.String() == "left":
			if !m.stepBackActionItemInfo(actionItem) {
				return m, nil
			}
			if currentID := strings.TrimSpace(m.actionItemInfoActionItemID); currentID != "" {
				m.actionItemInfoPath = []string{currentID}
			}
			return m, nil
		case msg.String() == "e":
			return m, m.startActionItemForm(&actionItem)
		case msg.String() == "s":
			return m, m.startSubactionItemForm(actionItem)
		case msg.String() == "c":
			return m.startActionItemThreadWithPanel(actionItem, modeActionItemInfo, threadPanelComments)
		case msg.String() == " " || msg.String() == "space":
			return m.toggleFocusedSubactionItemCompletion(actionItem)
		case msg.String() == "[":
			return m.moveActionItemIDs([]string{actionItem.ID}, -1, "move actionItem", actionItem.Title, false)
		case msg.String() == "]":
			return m.moveActionItemIDs([]string{actionItem.ID}, 1, "move actionItem", actionItem.Title, false)
		case msg.String() == "f":
			if !m.activateSubtreeFocus(actionItem.ID) {
				return m, nil
			}
			m.closeActionItemInfo("focused subtree")
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
			if m.mode == modeActionItemInfo {
				m.actionItemInfoActionItemID = strings.TrimSpace(m.dependencyOwnerActionItemID)
			}
			m.status = "dependency inspector cancelled"
			if m.mode == modeAddActionItem || m.mode == modeEditActionItem {
				return m, m.focusActionItemFormField(m.dependencyActiveField)
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
				m.toggleDependencyCandidateInActiveField(candidate.Match.ActionItem.ID)
				return m, nil
			default:
				return m, nil
			}
		case msg.String() == "x" && m.dependencyFocus != 0:
			if m.dependencyActiveField == actionItemFieldDependsOn {
				m.dependencyActiveField = actionItemFieldBlockedBy
			} else {
				m.dependencyActiveField = actionItemFieldDependsOn
			}
			m.status = "active field: " + m.dependencyActiveFieldLabel()
			return m, nil
		case msg.String() == "d" && m.dependencyFocus == 4:
			candidate, ok := m.selectedDependencyCandidate()
			if !ok {
				return m, nil
			}
			if ownerActionItemID := strings.TrimSpace(m.dependencyOwnerActionItemID); ownerActionItemID != "" && strings.TrimSpace(candidate.Match.ActionItem.ID) == ownerActionItemID {
				m.status = "actionItem cannot depend on itself"
				return m, nil
			}
			var added bool
			m.dependencyDependsOn, added = toggleDependencyID(m.dependencyDependsOn, candidate.Match.ActionItem.ID)
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
			if ownerActionItemID := strings.TrimSpace(m.dependencyOwnerActionItemID); ownerActionItemID != "" && strings.TrimSpace(candidate.Match.ActionItem.ID) == ownerActionItemID {
				m.status = "actionItem cannot depend on itself"
				return m, nil
			}
			var added bool
			m.dependencyBlockedBy, added = toggleDependencyID(m.dependencyBlockedBy, candidate.Match.ActionItem.ID)
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
				return m.jumpToDependencyCandidateActionItem()
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
			_ = scrubTextInputTerminalArtifacts(&m.dependencyInput)
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
			_ = scrubTextInputTerminalArtifacts(&m.bootstrapDisplayInput)
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
		case msg.String() == "j" || msg.String() == "down" || msg.String() == "right" || msg.Code == tea.KeyDown || msg.Code == tea.KeyRight:
			if m.projectPickerIndex < len(m.projects)-1 {
				m.projectPickerIndex++
			}
			return m, nil
		case msg.String() == "k" || msg.String() == "up" || msg.String() == "left" || msg.Code == tea.KeyUp || msg.Code == tea.KeyLeft:
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
			m.selectedActionItem = 0
			m.mode = modeNone
			m.status = ""
			return m, m.loadData
		default:
			return m, nil
		}
	}

	if m.mode == modeSearch {
		const searchFocusSlots = 7
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
			case 5:
				m.cycleSearchMode(-1)
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
			case 5:
				m.cycleSearchMode(1)
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
			case 5:
				m.cycleSearchMode(1)
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
			case 5:
				m.cycleSearchMode(1)
				return m, nil
			default:
				return m, m.applySearchFilter()
			}
		default:
			if m.searchFocus == 0 {
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.searchInput)
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
			if m.searchOpeningResult {
				return m, nil
			}
			m.searchLoading = false
			m.searchActiveRequestID = 0
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
			if m.searchLoading {
				return m, nil
			}
			if len(m.searchMatches) == 0 {
				m.status = searchResultsStatusSummary(nil, m.searchRequestedMode, m.searchEffectiveMode, m.searchFallbackReason)
				return m, nil
			}
			match := m.searchMatches[clamp(m.searchResultIndex, 0, len(m.searchMatches)-1)]
			for idx, project := range m.projects {
				if project.ID == match.Project.ID {
					m.selectedProject = idx
					break
				}
			}
			m.searchApplied = false
			m.setBoardContextForActionItem(match.ActionItem)
			m.pendingFocusActionItemID = match.ActionItem.ID
			m.pendingOpenActionItemInfoID = match.ActionItem.ID
			m.searchLoading = true
			m.searchOpeningResult = true
			m.status = "opening search match..."
			return m, m.loadData
		default:
			return m, nil
		}
	}

	if m.mode == modeEmbeddingsStatus {
		if m.embeddingsFilterActive {
			if handled, status := applyClipboardShortcutToInput(msg, &m.embeddingsFilterInput); handled {
				m.status = status
				m.embeddingsFilterQuery = strings.TrimSpace(m.embeddingsFilterInput.Value())
				m.applyEmbeddingsDisplayFilter()
				return m, nil
			}
			switch msg.String() {
			case "esc", "enter":
				m.blurEmbeddingsFilter()
				return m, nil
			default:
				var cmd tea.Cmd
				m.embeddingsFilterInput, cmd = m.embeddingsFilterInput.Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.embeddingsFilterInput)
				m.embeddingsFilterQuery = strings.TrimSpace(m.embeddingsFilterInput.Value())
				m.applyEmbeddingsDisplayFilter()
				return m, cmd
			}
		}
		switch msg.String() {
		case "esc":
			m.mode = modeNone
			m.status = "ready"
			return m, nil
		case "/":
			return m, m.focusEmbeddingsFilter()
		case "g":
			return m, m.startEmbeddingsStatus(!m.embeddingsGlobal)
		case "a":
			m.embeddingsIncludeArchived = !m.embeddingsIncludeArchived
			return m, m.loadEmbeddingsStatusCmd()
		case "j", "down":
			if m.embeddingsIndex < len(m.embeddingsRows)-1 {
				m.embeddingsIndex++
			}
			return m, nil
		case "k", "up":
			if m.embeddingsIndex > 0 {
				m.embeddingsIndex--
			}
			return m, nil
		case "r":
			if m.embeddingsReindexInFlight {
				return m, nil
			}
			return m, m.startEmbeddingsReindex(false)
		case "R":
			if m.embeddingsReindexInFlight {
				return m, nil
			}
			return m, m.startEmbeddingsReindex(true)
		case "enter":
			return m.openSelectedEmbeddingStatusRow()
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
			_ = scrubTextInputTerminalArtifacts(&m.commandInput)
			m.commandMatches = m.filteredCommandItems(m.commandInput.Value())
			m.commandIndex = clamp(m.commandIndex, 0, len(m.commandMatches)-1)
			return m, cmd
		}
	}

	if m.mode == modeAuthScopePicker {
		switch msg.String() {
		case "esc":
			m.mode = modeAuthReview
			m.authReviewStage = authReviewStageSummary
			m.status = "auth review"
			return m, nil
		case "j", "down", "tab":
			if len(m.authReviewScopePickerItems) > 0 {
				m.authReviewScopePickerIndex = wrapIndex(m.authReviewScopePickerIndex, 1, len(m.authReviewScopePickerItems))
			}
			return m, nil
		case "k", "up", "shift+tab":
			if len(m.authReviewScopePickerItems) > 0 {
				m.authReviewScopePickerIndex = wrapIndex(m.authReviewScopePickerIndex, -1, len(m.authReviewScopePickerItems))
			}
			return m, nil
		case "enter":
			if len(m.authReviewScopePickerItems) == 0 {
				m.mode = modeAuthReview
				m.authReviewStage = authReviewStageSummary
				m.status = "no scope choices available"
				return m, nil
			}
			m.applySelectedAuthScopePickerItem()
			m.mode = modeAuthReview
			m.authReviewStage = authReviewStageSummary
			m.status = "auth scope updated"
			return m, nil
		default:
			return m, nil
		}
	}

	if m.mode == modeAuthInventory {
		switch msg.String() {
		case "esc":
			m.mode = modeNone
			m.status = "ready"
			if m.authInventoryNeedsReload {
				m.authInventoryNeedsReload = false
				return m, m.loadData
			}
			return m, nil
		case "g":
			if !m.authInventoryGlobal {
				if _, ok := m.currentProject(); !ok {
					m.status = "no project selected"
					return m, nil
				}
			}
			return m, m.startAuthInventory(!m.authInventoryGlobal)
		case "h":
			if m.authInventoryView == authInventoryViewHistory {
				m.authInventoryView = authInventoryViewLive
			} else {
				m.authInventoryView = authInventoryViewHistory
			}
			m.authInventoryIndex = 0
			m.authInventoryBody.GotoTop()
			m.status = "coordination " + m.authInventoryViewLabel()
			m.syncAuthInventoryViewport()
			return m, nil
		case "j", "down":
			m.authInventoryMoveSelection(1)
			m.syncAuthInventoryViewport()
			return m, nil
		case "k", "up":
			m.authInventoryMoveSelection(-1)
			m.syncAuthInventoryViewport()
			return m, nil
		case "pgdown", "ctrl+d":
			m.authInventoryMoveSelection(max(1, m.authInventoryBody.Height()/4))
			m.syncAuthInventoryViewport()
			return m, nil
		case "pgup", "ctrl+u":
			m.authInventoryMoveSelection(-max(1, m.authInventoryBody.Height()/4))
			m.syncAuthInventoryViewport()
			return m, nil
		case "home":
			m.authInventoryIndex = 0
			m.syncAuthInventoryViewport()
			return m, nil
		case "end":
			if items := m.authInventoryItems(); len(items) > 0 {
				m.authInventoryIndex = len(items) - 1
			}
			m.syncAuthInventoryViewport()
			return m, nil
		case "r":
			if next, cmd, ok := m.beginSelectedAuthSessionRevoke(); ok {
				return next, cmd
			}
			m.status = "select an active session to revoke"
			return m, nil
		case "a":
			item, ok := m.selectedAuthInventoryItem()
			if !ok || item.Request == nil {
				m.status = "select a request to approve"
				return m, nil
			}
			next, cmd, _ := m.beginAuthRequestDecision(*item.Request, "approve", modeAuthInventory)
			return next, cmd
		case "d":
			item, ok := m.selectedAuthInventoryItem()
			if !ok || item.Request == nil {
				m.status = "select a request to deny"
				return m, nil
			}
			next, cmd, _ := m.beginAuthRequestDecision(*item.Request, "deny", modeAuthInventory)
			return next, cmd
		case "enter":
			item, ok := m.selectedAuthInventoryItem()
			if !ok {
				m.status = "no auth rows available"
				return m, nil
			}
			if item.Request != nil {
				next, cmd, _ := m.beginAuthRequestDecision(*item.Request, "approve", modeAuthInventory)
				return next, cmd
			}
			if title, body, ok := m.authInventoryItemDetail(item); ok {
				m.openCoordinationDetail(item, title, body)
				return m, nil
			}
			return m, nil
		default:
			return m, nil
		}
	}

	if m.mode == modeCoordinationDetail {
		switch msg.String() {
		case "esc":
			m.closeCoordinationDetail()
			return m, nil
		case "j", "down", "tab":
			m.coordinationDetailMoveAction(1)
			return m, nil
		case "k", "up", "shift+tab":
			m.coordinationDetailMoveAction(-1)
			return m, nil
		case "r":
			for idx, action := range m.coordinationDetailActions {
				switch strings.TrimSpace(action.Confirm.Kind) {
				case "revoke-auth-session", "revoke-capability-lease":
					m.coordinationDetailActionIndex = idx
					m.pendingConfirm = action.Confirm
					m.confirmChoice = 1
					m.mode = modeConfirmAction
					m.status = "review action"
					return m, nil
				}
			}
			return m, nil
		case "enter":
			action, ok := m.selectedCoordinationDetailAction()
			if !ok || strings.TrimSpace(action.Confirm.Kind) == "" {
				m.closeCoordinationDetail()
				return m, nil
			}
			m.pendingConfirm = action.Confirm
			m.confirmChoice = 1
			m.mode = modeConfirmAction
			m.status = "review action"
			return m, nil
		default:
			return m, nil
		}
	}

	if m.mode == modeAuthSessionRevoke {
		switch msg.String() {
		case "esc":
			m.mode = modeAuthInventory
			m.pendingConfirm = confirmAction{}
			m.status = "coordination"
			return m, nil
		case "enter":
			action := m.pendingConfirm
			m.mode = modeNone
			m.pendingConfirm = confirmAction{}
			m.status = "applying action..."
			return m.applyConfirmedAction(action)
		default:
			return m, nil
		}
	}

	if m.mode == modeAuthReview {
		switch m.authReviewStage {
		case authReviewStageSummary:
			switch msg.String() {
			case "esc":
				return m, m.closeAuthReview("cancelled", false)
			case "s":
				m.openAuthScopePicker()
				m.status = "pick approved scope"
				return m, nil
			case "t":
				m.status = "edit session ttl"
				return m, m.authReviewOpenTTLStage()
			case "n":
				_ = m.setPendingAuthRequestDecision("approve")
				m.status = "edit approval note"
				return m, m.authReviewOpenNoteStage(authReviewStageEditApproveNote)
			case "d":
				_ = m.setPendingAuthRequestDecision("deny")
				m.status = "review denial note"
				return m, m.authReviewOpenNoteStage(authReviewStageDeny)
			case "enter":
				_ = m.setPendingAuthRequestDecision("approve")
				if err := m.openAuthReviewConfirm(); err != nil {
					m.status = err.Error()
					return m, nil
				}
				return m, nil
			default:
				return m, nil
			}
		case authReviewStageEditTTL:
			switch {
			case msg.String() == "esc":
				m.authReviewReturnToSummary()
				m.status = "auth review"
				return m, nil
			case msg.Code == tea.KeyEnter || msg.String() == "enter":
				if err := m.authReviewApplyEditedTTL(); err != nil {
					m.status = err.Error()
					return m, nil
				}
				m.status = "auth review"
				return m, nil
			default:
				var cmd tea.Cmd
				m.confirmAuthTTLInput, cmd = m.confirmAuthTTLInput.Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.confirmAuthTTLInput)
				return m, cmd
			}
		case authReviewStageEditApproveNote, authReviewStageDeny:
			switch {
			case msg.String() == "esc":
				m.authReviewReturnToSummary()
				m.status = "auth review"
				return m, nil
			case msg.Code == tea.KeyEnter || msg.String() == "enter":
				if m.authReviewStage == authReviewStageDeny {
					_ = m.setPendingAuthRequestDecision("deny")
					m.pendingConfirm.AuthRequestNote = strings.TrimSpace(m.confirmAuthNoteInput.Value())
					if err := m.openAuthReviewConfirm(); err != nil {
						m.status = err.Error()
						return m, nil
					}
					return m, nil
				}
				m.authReviewApplyEditedNote("approve")
				m.status = "auth review"
				return m, nil
			default:
				var cmd tea.Cmd
				m.confirmAuthNoteInput, cmd = m.confirmAuthNoteInput.Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.confirmAuthNoteInput)
				m.pendingConfirm.AuthRequestNote = strings.TrimSpace(m.confirmAuthNoteInput.Value())
				return m, cmd
			}
		default:
			m.authReviewStage = authReviewStageSummary
			return m, nil
		}
	}

	if m.mode == modeConfirmAction {
		switch msg.String() {
		case "esc", "n":
			if strings.TrimSpace(m.pendingConfirm.AuthRequestID) != "" {
				m.mode = modeAuthReview
				m.authReviewReturnToSummary()
				m.status = "auth review"
				return m, nil
			}
			if m.pendingConfirm.ReturnToAuthAccess {
				m.mode = modeAuthInventory
			} else {
				m.mode = modeNone
			}
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
				if strings.TrimSpace(m.pendingConfirm.AuthRequestID) != "" {
					m.mode = modeAuthReview
					m.authReviewReturnToSummary()
					m.status = "auth review"
					return m, nil
				}
				if m.pendingConfirm.ReturnToAuthAccess {
					m.mode = modeAuthInventory
				} else {
					m.mode = modeNone
				}
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
			if m.quickActionBackMode != modeNone {
				m.mode = m.quickActionBackMode
			} else {
				m.mode = modeNone
			}
			m.quickActionBackMode = modeNone
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
			return m, m.focusActionItemFormField(actionItemFieldDue)
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
			if len(options) == 0 || len(m.formInputs) <= actionItemFieldDue {
				m.mode = m.pickerBack
				m.pickerBack = modeNone
				return m, m.focusActionItemFormField(actionItemFieldDue)
			}
			choice := options[clamp(m.duePicker, 0, len(options)-1)]
			m.formInputs[actionItemFieldDue].SetValue(choice.Value)
			m.mode = m.pickerBack
			m.pickerBack = modeNone
			m.status = "due updated"
			return m, m.focusActionItemFormField(actionItemFieldDue)
		default:
			switch m.duePickerFocus {
			case 1:
				var cmd tea.Cmd
				before := m.duePickerDateInput.Value()
				m.duePickerDateInput, cmd = m.duePickerDateInput.Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.duePickerDateInput)
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
				_ = scrubTextInputTerminalArtifacts(&m.duePickerTimeInput)
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
			_ = scrubTextInputTerminalArtifacts(&m.resourcePickerFilter)
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
			m.actionItemFormResourceEditIndex = -1
			m.status = "resource picker cancelled"
			if m.mode == modeEditActionItem {
				return m, m.focusActionItemFormField(actionItemFieldResources)
			}
			if m.mode == modeAddActionItem {
				return m, m.focusActionItemFormField(m.formFocus)
			}
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
			_ = scrubTextInputTerminalArtifacts(&m.resourcePickerFilter)
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
			_ = scrubTextInputTerminalArtifacts(&m.resourcePickerFilter)
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
			_ = scrubTextInputTerminalArtifacts(&m.labelPickerInput)
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
			if m.mode == modeAddActionItem || m.mode == modeEditActionItem {
				return m, m.focusActionItemFormField(actionItemFieldLabels)
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
			if len(m.labelPickerItems) == 0 || len(m.formInputs) <= actionItemFieldLabels {
				m.mode = m.labelPickerBack
				m.labelPickerInput.Blur()
				return m, m.focusActionItemFormField(actionItemFieldLabels)
			}
			item := m.labelPickerItems[clamp(m.labelPickerIndex, 0, len(m.labelPickerItems)-1)]
			m.appendActionItemFormLabel(item.Label)
			m.mode = m.labelPickerBack
			m.labelPickerInput.Blur()
			m.status = "label added"
			return m, m.focusActionItemFormField(actionItemFieldLabels)
		default:
			var cmd tea.Cmd
			before := m.labelPickerInput.Value()
			m.labelPickerInput, cmd = m.labelPickerInput.Update(msg)
			_ = scrubTextInputTerminalArtifacts(&m.labelPickerInput)
			if m.labelPickerInput.Value() != before {
				m.labelPickerIndex = 0
				m.refreshLabelPickerMatches()
			}
			return m, cmd
		}
	}

	if m.mode == modeProjectKindPicker {
		if handled, status := applyClipboardShortcutToInput(msg, &m.projectKindPickerInput); handled {
			m.status = status
			m.projectKindPickerIndex = 0
			m.refreshProjectKindPickerMatches()
			return m, nil
		}
		switch msg.String() {
		case "esc":
			m.mode = m.projectKindPickerBack
			m.projectKindPickerInput.Blur()
			m.status = "project kind picker cancelled"
			if m.mode == modeAddProject || m.mode == modeEditProject {
				return m, m.focusProjectFormField(projectFieldKind)
			}
			return m, nil
		case "ctrl+u":
			m.projectKindPickerInput.SetValue("")
			m.projectKindPickerInput.CursorEnd()
			m.projectKindPickerIndex = 0
			m.refreshProjectKindPickerMatches()
			return m, nil
		case "j", "down":
			if m.projectKindPickerIndex < len(m.projectKindPickerItems)-1 {
				m.projectKindPickerIndex++
			}
			return m, nil
		case "k", "up":
			if m.projectKindPickerIndex > 0 {
				m.projectKindPickerIndex--
			}
			return m, nil
		case "enter":
			if len(m.projectFormInputs) <= projectFieldKind {
				m.mode = m.projectKindPickerBack
				m.projectKindPickerInput.Blur()
				return m, m.focusProjectFormField(projectFieldKind)
			}
			if len(m.projectKindPickerItems) == 0 {
				m.mode = m.projectKindPickerBack
				m.projectKindPickerInput.Blur()
				m.status = "no project kinds available"
				return m, m.focusProjectFormField(projectFieldKind)
			}
			item := m.projectKindPickerItems[clamp(m.projectKindPickerIndex, 0, len(m.projectKindPickerItems)-1)]
			m.projectFormInputs[projectFieldKind].SetValue(string(item.KindID))
			m.mode = m.projectKindPickerBack
			m.projectKindPickerInput.Blur()
			m.status = "project kind selected"
			return m, m.focusProjectFormField(projectFieldKind + 1)
		default:
			if msg.Text != "" && (msg.Mod&tea.ModCtrl) == 0 {
				var cmd tea.Cmd
				before := m.projectKindPickerInput.Value()
				m.projectKindPickerInput, cmd = m.projectKindPickerInput.Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.projectKindPickerInput)
				if m.projectKindPickerInput.Value() != before {
					m.projectKindPickerIndex = 0
					m.refreshProjectKindPickerMatches()
				}
				return m, cmd
			}
			var cmd tea.Cmd
			before := m.projectKindPickerInput.Value()
			m.projectKindPickerInput, cmd = m.projectKindPickerInput.Update(msg)
			_ = scrubTextInputTerminalArtifacts(&m.projectKindPickerInput)
			if m.projectKindPickerInput.Value() != before {
				m.projectKindPickerIndex = 0
				m.refreshProjectKindPickerMatches()
			}
			return m, cmd
		}
	}

	if m.mode == modeTemplateLibraryPicker {
		if handled, status := applyClipboardShortcutToInput(msg, &m.templateLibraryPickerInput); handled {
			m.status = status
			m.templateLibraryPickerIndex = 0
			m.refreshTemplateLibraryPickerMatches()
			return m, nil
		}
		switch msg.String() {
		case "esc":
			m.mode = m.templateLibraryPickerBack
			m.templateLibraryPickerInput.Blur()
			m.status = "template library picker cancelled"
			if m.mode == modeAddProject || m.mode == modeEditProject {
				return m, m.focusProjectFormField(projectFieldTemplateLibrary)
			}
			return m, nil
		case "ctrl+u":
			m.templateLibraryPickerInput.SetValue("")
			m.templateLibraryPickerInput.CursorEnd()
			m.templateLibraryPickerIndex = 0
			m.refreshTemplateLibraryPickerMatches()
			return m, nil
		case "j", "down":
			if m.templateLibraryPickerIndex < len(m.templateLibraryPickerItems)-1 {
				m.templateLibraryPickerIndex++
			}
			return m, nil
		case "k", "up":
			if m.templateLibraryPickerIndex > 0 {
				m.templateLibraryPickerIndex--
			}
			return m, nil
		case "enter":
			if len(m.projectFormInputs) <= projectFieldTemplateLibrary {
				m.mode = m.templateLibraryPickerBack
				m.templateLibraryPickerInput.Blur()
				return m, m.focusProjectFormField(projectFieldTemplateLibrary)
			}
			if len(m.templateLibraryPickerItems) == 0 {
				m.mode = m.templateLibraryPickerBack
				m.templateLibraryPickerInput.Blur()
				m.status = "no approved template libraries"
				return m, m.focusProjectFormField(projectFieldTemplateLibrary)
			}
			item := m.templateLibraryPickerItems[clamp(m.templateLibraryPickerIndex, 0, len(m.templateLibraryPickerItems)-1)]
			if item.Clear {
				m.projectFormInputs[projectFieldTemplateLibrary].SetValue("")
				m.status = "template library cleared"
			} else {
				m.projectFormInputs[projectFieldTemplateLibrary].SetValue(item.LibraryID)
				currentProjectKind := domain.NormalizeKindID(domain.KindID(m.projectFormInputs[projectFieldKind].Value()))
				if (currentProjectKind == "" || currentProjectKind == domain.DefaultProjectKind) && len(m.projectFormInputs) > projectFieldKind {
					if inferredKind, ok := m.inferProjectKindFromTemplateLibrary(item.LibraryID); ok {
						m.projectFormInputs[projectFieldKind].SetValue(string(inferredKind))
					}
				}
				m.status = "template library selected"
			}
			m.mode = m.templateLibraryPickerBack
			m.templateLibraryPickerInput.Blur()
			return m, m.focusProjectFormField(projectFieldTemplateLibrary + 1)
		default:
			if msg.Text != "" && (msg.Mod&tea.ModCtrl) == 0 {
				var cmd tea.Cmd
				before := m.templateLibraryPickerInput.Value()
				m.templateLibraryPickerInput, cmd = m.templateLibraryPickerInput.Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.templateLibraryPickerInput)
				if m.templateLibraryPickerInput.Value() != before {
					m.templateLibraryPickerIndex = 0
					m.refreshTemplateLibraryPickerMatches()
				}
				return m, cmd
			}
			var cmd tea.Cmd
			before := m.templateLibraryPickerInput.Value()
			m.templateLibraryPickerInput, cmd = m.templateLibraryPickerInput.Update(msg)
			_ = scrubTextInputTerminalArtifacts(&m.templateLibraryPickerInput)
			if m.templateLibraryPickerInput.Value() != before {
				m.templateLibraryPickerIndex = 0
				m.refreshTemplateLibraryPickerMatches()
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
			_ = scrubTextInputTerminalArtifacts(&m.pathsRootInput)
			return m, cmd
		}
	}

	if m.mode == modeAddActionItem || m.mode == modeEditActionItem {
		if len(m.formInputs) > 0 && m.formFocus >= 0 && m.formFocus < len(m.formInputs) && !isActionItemFormActionField(m.formFocus) && !isActionItemFormMarkdownField(m.formFocus) {
			if handled, status := applyClipboardShortcutToInput(msg, &m.formInputs[m.formFocus]); handled {
				m.status = status
				return m, nil
			}
			if isActionItemFormDirectTextInputField(m.formFocus) && isPrintableFormTextKey(msg) {
				var cmd tea.Cmd
				m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.formInputs[m.formFocus])
				return m, cmd
			}
		}
		switch {
		case key.Matches(msg, m.keys.quickActions) && (isActionItemFormActionField(m.formFocus) || isActionItemFormMarkdownField(m.formFocus)):
			return m, m.startQuickActions()
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			if m.actionItemFormBackMode == modeEditActionItem && strings.TrimSpace(m.actionItemFormBackActionItemID) != "" {
				parentID := strings.TrimSpace(m.actionItemFormBackActionItemID)
				childID := strings.TrimSpace(m.editingActionItemID)
				if childID == "" {
					childID = strings.TrimSpace(m.actionItemFormBackChildID)
				}
				parent, ok := m.actionItemByID(parentID)
				if !ok {
					m.status = "parent actionItem not found"
					return m, nil
				}
				cmd := m.startActionItemForm(&parent)
				m.selectActionItemFormSubactionItemByID(childID)
				m.syncActionItemFormViewportToFocus()
				m.status = "edit actionItem"
				return m, cmd
			}
			m.mode = modeNone
			m.formInputs = nil
			m.formFocus = 0
			m.actionItemFormDescription = ""
			m.actionItemFormMarkdown = nil
			m.actionItemFormTouched = nil
			m.editingActionItemID = ""
			m.actionItemFormParentID = ""
			m.actionItemFormKind = domain.KindActionItem
			m.actionItemFormScope = domain.KindAppliesToActionItem
			m.actionItemFormBackMode = modeNone
			m.actionItemFormBackActionItemID = ""
			m.actionItemFormBackChildID = ""
			m.actionItemFormResourceRefs = nil
			m.actionItemFormSubactionItemCursor = 0
			m.actionItemFormResourceCursor = 0
			m.actionItemFormResourceEditIndex = -1
			m.status = "cancelled"
			return m, nil
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i":
			return m, m.moveActionItemFormFocus(1, false)
		case msg.String() == "shift+tab" || msg.String() == "backtab":
			return m, m.moveActionItemFormFocus(-1, false)
		case m.formFocus == actionItemFieldSubtasks && (msg.String() == "left" || msg.String() == "right"):
			if msg.String() == "left" {
				m.moveActionItemFormSubactionItemCursor(-1)
			} else {
				m.moveActionItemFormSubactionItemCursor(1)
			}
			m.syncActionItemFormViewportToFocus()
			return m, nil
		case m.formFocus == actionItemFieldResources && (msg.String() == "left" || msg.String() == "right"):
			if msg.String() == "left" {
				m.moveActionItemFormResourceCursor(-1)
			} else {
				m.moveActionItemFormResourceCursor(1)
			}
			m.syncActionItemFormViewportToFocus()
			return m, nil
		case msg.String() == "down":
			return m, m.moveActionItemFormFocus(1, true)
		case msg.String() == "up":
			return m, m.moveActionItemFormFocus(-1, true)
		case msg.String() == "ctrl+s":
			return m.submitInputMode()
		case msg.String() == "e":
			if next, cmd, handled := m.openFocusedActionItemFormField(tea.KeyPressMsg{}); handled {
				return next, cmd
			}
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			if next, cmd, handled := m.openFocusedActionItemFormField(msg); handled {
				return next, cmd
			}
			return m.submitInputMode()
		default:
			if m.formFocus == actionItemFieldPriority {
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
			if isActionItemFormActionField(m.formFocus) {
				return m, nil
			}
			if isActionItemFormMarkdownField(m.formFocus) {
				return m, m.startActionItemFormMarkdownEditor(m.formFocus, msg)
			}
			if len(m.formInputs) == 0 || m.formFocus < 0 || m.formFocus >= len(m.formInputs) {
				return m, nil
			}
			var cmd tea.Cmd
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
			_ = scrubTextInputTerminalArtifacts(&m.formInputs[m.formFocus])
			return m, cmd
		}
	}

	if m.mode == modeAddProject || m.mode == modeEditProject {
		if len(m.projectFormInputs) > 0 && m.projectFormFocus >= 0 && m.projectFormFocus < len(m.projectFormInputs) && m.projectFormFocus != projectFieldDescription {
			if handled, status := applyClipboardShortcutToInput(msg, &m.projectFormInputs[m.projectFormFocus]); handled {
				m.status = status
				return m, nil
			}
			if isProjectFormDirectTextInputField(m.projectFormFocus) && isPrintableFormTextKey(msg) {
				var cmd tea.Cmd
				m.projectFormInputs[m.projectFormFocus], cmd = m.projectFormInputs[m.projectFormFocus].Update(msg)
				_ = scrubTextInputTerminalArtifacts(&m.projectFormInputs[m.projectFormFocus])
				return m, cmd
			}
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeNone
			m.resetProjectFormState()
			m.clearProjectTemplateMigrationReview()
			m.status = "cancelled"
			return m, nil
		case msg.String() == "ctrl+r" && m.projectFormFocus == projectFieldRootPath:
			return m, m.startResourcePicker("", m.mode)
		case (msg.String() == "e" || msg.Code == tea.KeyEnter || msg.String() == "enter") && m.projectFormFocus == projectFieldKind:
			return m, m.startProjectKindPicker("")
		case (msg.String() == "e" || msg.Code == tea.KeyEnter || msg.String() == "enter") && m.projectFormFocus == projectFieldTemplateLibrary:
			return m, m.startTemplateLibraryPicker("")
		case (msg.String() == "e" || msg.Code == tea.KeyEnter || msg.String() == "enter") && m.projectFormFocus == projectFieldComments:
			next, cmd := m.startProjectThread(modeEditProject)
			if model, ok := next.(Model); ok {
				model.threadPanelFocus = threadPanelComments
				model.threadDetailsActive = false
				return model, cmd
			}
			return next, cmd
		case msg.String() == "i" && m.projectFormFocus == projectFieldDescription:
			return m, m.startProjectDescriptionEditor(msg)
		case msg.Code == tea.KeyTab || msg.String() == "tab" || msg.String() == "ctrl+i" || msg.String() == "down":
			return m, m.focusProjectFormField(m.projectFormFocus + 1)
		case msg.String() == "shift+tab" || msg.String() == "backtab" || msg.String() == "up":
			return m, m.focusProjectFormField(m.projectFormFocus - 1)
		case msg.Code == tea.KeyEnter || msg.String() == "enter":
			if m.projectFormFocus == projectFieldDescription {
				return m, m.startProjectDescriptionEditor(msg)
			}
			return m.submitInputMode()
		default:
			if m.projectFormFocus == projectFieldDescription {
				return m, m.startProjectDescriptionEditor(msg)
			}
			if m.projectFormFocus == projectFieldKind {
				if isPrintableFormTextKey(msg) {
					return m, m.startProjectKindPicker(msg.Text)
				}
				return m, nil
			}
			if m.projectFormFocus == projectFieldTemplateLibrary {
				if isPrintableFormTextKey(msg) {
					return m, m.startTemplateLibraryPicker(msg.Text)
				}
				return m, nil
			}
			if m.projectFormFocus == projectFieldComments {
				return m, nil
			}
			if len(m.projectFormInputs) == 0 {
				return m, nil
			}
			var cmd tea.Cmd
			m.projectFormInputs[m.projectFormFocus], cmd = m.projectFormInputs[m.projectFormFocus].Update(msg)
			_ = scrubTextInputTerminalArtifacts(&m.projectFormInputs[m.projectFormFocus])
			return m, cmd
		}
	}

	if m.mode == modeTemplateMigrationReview {
		if m.templateMigrationReviewLoading {
			switch {
			case msg.Code == tea.KeyEscape || msg.String() == "esc":
				m.mode = modeEditProject
				m.templateMigrationReviewLoading = false
				m.templateMigrationReviewPreview = nil
				m.templateMigrationReviewDraft = nil
				m.templateMigrationReviewPicked = map[string]struct{}{}
				m.templateMigrationReviewIndex = 0
				m.status = "template reapply review cancelled"
				return m, nil
			default:
				return m, nil
			}
		}
		switch {
		case msg.Code == tea.KeyEscape || msg.String() == "esc":
			m.mode = modeEditProject
			m.clearProjectTemplateMigrationReview()
			m.status = "template reapply review cancelled"
			return m, nil
		case msg.Code == tea.KeyPgDown || msg.String() == "pgdown" || msg.String() == "ctrl+d":
			m.actionItemInfoBody.ScrollDown(max(1, m.actionItemInfoBody.Height()/2))
			return m, nil
		case msg.Code == tea.KeyPgUp || msg.String() == "pgup" || msg.String() == "ctrl+u":
			m.actionItemInfoBody.ScrollUp(max(1, m.actionItemInfoBody.Height()/2))
			return m, nil
		case msg.String() == "home":
			m.actionItemInfoBody.GotoTop()
			return m, nil
		case msg.String() == "end":
			m.actionItemInfoBody.GotoBottom()
			return m, nil
		case msg.String() == "j" || msg.String() == "down":
			if len(m.templateMigrationReviewCandidates()) == 0 {
				m.actionItemInfoBody.ScrollDown(1)
				return m, nil
			}
			if m.templateMigrationReviewIndex < len(m.templateMigrationReviewCandidates())-1 {
				m.templateMigrationReviewIndex++
			}
			m.actionItemInfoBody.ScrollDown(1)
			return m, nil
		case msg.String() == "k" || msg.String() == "up":
			if len(m.templateMigrationReviewCandidates()) == 0 {
				m.actionItemInfoBody.ScrollUp(1)
				return m, nil
			}
			if m.templateMigrationReviewIndex > 0 {
				m.templateMigrationReviewIndex--
			}
			m.actionItemInfoBody.ScrollUp(1)
			return m, nil
		case msg.String() == " " || msg.String() == "space" || msg.Code == tea.KeyEnter || msg.String() == "enter":
			candidate, ok := m.selectedTemplateMigrationReviewCandidate()
			if !ok {
				m.status = "no migration candidates"
				return m, nil
			}
			if candidate.Status != domain.ProjectTemplateReapplyCandidateEligible {
				m.status = firstNonEmptyTrimmed(candidate.Reason, "candidate is not eligible")
				return m, nil
			}
			m.toggleTemplateMigrationSelection(candidate.ActionItemID)
			if _, ok := m.templateMigrationReviewPicked[strings.TrimSpace(candidate.ActionItemID)]; ok {
				m.status = "migration selected"
			} else {
				m.status = "migration unselected"
			}
			return m, nil
		case msg.String() == "a":
			return m.applyProjectTemplateReviewDecision(false, true)
		case msg.String() == "A" || msg.String() == "shift+a":
			return m.applyProjectTemplateReviewDecision(true, false)
		case msg.String() == "s":
			return m.applyProjectTemplateReviewDecision(false, false)
		default:
			return m, nil
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
			m.labelsConfigBranchActionItemID = ""
			m.labelsConfigPhaseActionItemID = ""
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
			_ = scrubTextInputTerminalArtifacts(&m.labelsConfigInputs[m.labelsConfigFocus])
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
			_ = scrubTextInputTerminalArtifacts(&m.highlightColorInput)
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
		m.editingActionItemID = ""
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
		merged = sanitizeInteractiveInputValue(merged)
		in.SetValue(merged)
		in.SetCursor(clamp(nextPos, 0, utf8.RuneCountInString(merged)))
		return true, "pasted from clipboard"
	default:
		return false, ""
	}
}

// applyClipboardShortcutToTextArea handles copy/paste shortcuts for one textarea.
func applyClipboardShortcutToTextArea(msg tea.KeyPressMsg, in *textarea.Model) (bool, string) {
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
		in.InsertString(text)
		_ = scrubTextAreaTerminalArtifacts(in)
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
	case modeAddActionItem:
		if text := strings.TrimSpace(m.input); text != "" {
			vals := m.actionItemFormValues()
			if vals["title"] == "" {
				vals["title"] = text
			}
		}
		vals := m.actionItemFormValues()
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
		metadata := m.buildActionItemMetadataFromForm(vals, domain.ActionItemMetadata{})
		parentID := m.actionItemFormParentID
		kind := m.actionItemFormKind
		scope := m.actionItemFormScope

		m.mode = modeNone
		m.formInputs = nil
		m.actionItemFormDescription = ""
		m.actionItemFormMarkdown = nil
		m.actionItemFormTouched = nil
		m.actionItemFormParentID = ""
		m.actionItemFormKind = domain.KindActionItem
		m.actionItemFormScope = domain.KindAppliesToActionItem
		m.actionItemFormBackMode = modeNone
		m.actionItemFormBackActionItemID = ""
		m.actionItemFormBackChildID = ""
		m.actionItemFormResourceRefs = nil
		m.actionItemFormSubactionItemCursor = 0
		m.actionItemFormResourceCursor = 0
		m.actionItemFormResourceEditIndex = -1
		m.traceFormControlCharacterGuard("actionItem", "create", "title", title)
		m.traceFormControlCharacterGuard("actionItem", "create", "description", vals["description"])
		return m.createActionItem(app.CreateActionItemInput{
			ParentID:       parentID,
			Kind:           kind,
			Scope:          scope,
			Title:          title,
			Description:    vals["description"],
			Priority:       priority,
			DueAt:          dueAt,
			Labels:         labels,
			Metadata:       metadata,
			CreatedByActor: m.threadActorID(),
			CreatedByName:  m.threadActorName(),
			UpdatedByActor: m.threadActorID(),
			UpdatedByName:  m.threadActorName(),
			UpdatedByType:  m.threadActorType(),
		})
	case modeSearch:
		return m, m.applySearchFilter()
	case modeRenameActionItem:
		text := strings.TrimSpace(m.input)
		m.mode = modeNone
		m.input = ""
		if text == "" {
			m.status = "title required"
			return m, nil
		}
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		actionItemID := actionItem.ID
		return m, func() tea.Msg {
			_, err := m.svc.RenameActionItem(context.Background(), actionItemID, text)
			if err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "actionItem renamed", reload: true}
		}
	case modeEditActionItem:
		if text := strings.TrimSpace(m.input); text != "" {
			actionItemID := m.editingActionItemID
			actionItem, ok := m.actionItemByID(actionItemID)
			if !ok {
				m.status = "actionItem not found"
				return m, nil
			}
			in, err := parseActionItemEditInput(text, actionItem)
			if err != nil {
				m.status = "invalid edit format: " + err.Error()
				return m, nil
			}
			m.mode = modeNone
			m.formInputs = nil
			m.actionItemFormDescription = ""
			m.actionItemFormMarkdown = nil
			m.actionItemFormTouched = nil
			m.input = ""
			m.editingActionItemID = ""
			m.actionItemFormBackMode = modeNone
			m.actionItemFormBackActionItemID = ""
			m.actionItemFormBackChildID = ""
			m.actionItemFormResourceRefs = nil
			m.actionItemFormSubactionItemCursor = 0
			m.actionItemFormResourceCursor = 0
			m.actionItemFormResourceEditIndex = -1
			in.ActionItemID = actionItemID
			m.traceFormControlCharacterGuard("actionItem", "update", "title", in.Title)
			m.traceFormControlCharacterGuard("actionItem", "update", "description", in.Description)
			return m, func() tea.Msg {
				_, updateErr := m.svc.UpdateActionItem(context.Background(), in)
				if updateErr != nil {
					return actionMsg{err: updateErr}
				}
				return actionMsg{status: "actionItem updated", reload: true}
			}
		}
		in, _, err := m.buildCurrentEditActionItemInput()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		reopenEditActionItemID := strings.TrimSpace(m.actionItemFormBackActionItemID)
		reselectChildID := strings.TrimSpace(in.ActionItemID)
		if m.actionItemFormBackMode != modeEditActionItem {
			reopenEditActionItemID = ""
			reselectChildID = ""
		}

		m.mode = modeNone
		m.formInputs = nil
		m.actionItemFormDescription = ""
		m.actionItemFormMarkdown = nil
		m.actionItemFormTouched = nil
		m.editingActionItemID = ""
		m.actionItemFormParentID = ""
		m.actionItemFormKind = domain.KindActionItem
		m.actionItemFormScope = domain.KindAppliesToActionItem
		m.actionItemFormBackMode = modeNone
		m.actionItemFormBackActionItemID = ""
		m.actionItemFormBackChildID = ""
		m.actionItemFormResourceRefs = nil
		m.actionItemFormSubactionItemCursor = 0
		m.actionItemFormResourceCursor = 0
		m.actionItemFormResourceEditIndex = -1
		m.traceFormControlCharacterGuard("actionItem", "update", "title", in.Title)
		m.traceFormControlCharacterGuard("actionItem", "update", "description", in.Description)
		return m, func() tea.Msg {
			updatedActionItem, updateErr := m.svc.UpdateActionItem(context.Background(), in)
			if updateErr != nil {
				return actionMsg{err: updateErr}
			}
			if reopenEditActionItemID != "" {
				return actionItemUpdatedMsg{
					actionItem:             updatedActionItem,
					status:                 "actionItem updated",
					reopenEditActionItemID: reopenEditActionItemID,
					reselectChildID:        reselectChildID,
				}
			}
			return actionMsg{status: "actionItem updated", reload: true}
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
		branchActionItemID := strings.TrimSpace(m.labelsConfigBranchActionItemID)
		phaseActionItemID := strings.TrimSpace(m.labelsConfigPhaseActionItemID)

		m.allowedLabelGlobal = append([]string(nil), globalLabels...)
		if len(projectLabels) == 0 {
			delete(m.allowedLabelProject, slug)
		} else {
			m.allowedLabelProject[slug] = append([]string(nil), projectLabels...)
		}
		m.refreshActionItemFormLabelSuggestions()
		m.mode = modeNone
		m.labelsConfigInputs = nil
		m.labelsConfigFocus = 0
		m.labelsConfigSlug = ""
		m.labelsConfigBranchActionItemID = ""
		m.labelsConfigPhaseActionItemID = ""
		return m, func() tea.Msg {
			if err := m.saveLabels(slug, globalLabels, projectLabels); err != nil {
				return actionMsg{err: err}
			}
			updateActionItemLabels := func(actionItemID string, labels []string) error {
				actionItemID = strings.TrimSpace(actionItemID)
				if actionItemID == "" {
					return nil
				}
				actionItem, ok := m.actionItemByID(actionItemID)
				if !ok {
					return nil
				}
				if slices.Equal(normalizeConfigLabels(actionItem.Labels), normalizeConfigLabels(labels)) {
					return nil
				}
				_, err := m.svc.UpdateActionItem(context.Background(), app.UpdateActionItemInput{
					ActionItemID:  actionItem.ID,
					Title:         actionItem.Title,
					Description:   actionItem.Description,
					Priority:      actionItem.Priority,
					DueAt:         actionItem.DueAt,
					Labels:        append([]string(nil), labels...),
					Metadata:      &actionItem.Metadata,
					UpdatedBy:     m.threadActorID(),
					UpdatedByName: m.threadActorName(),
					UpdatedType:   m.threadActorType(),
				})
				return err
			}
			if err := updateActionItemLabels(branchActionItemID, branchLabels); err != nil {
				return actionMsg{err: err}
			}
			if err := updateActionItemLabels(phaseActionItemID, phaseLabels); err != nil {
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
		kindID := domain.NormalizeKindID(domain.KindID(vals["kind"]))
		if kindID == "" {
			m.status = "project kind required"
			return m, nil
		}
		if !m.hasProjectKindDefinition(kindID) {
			m.status = "project kind not found: " + string(kindID)
			return m, nil
		}
		templateLibraryID := domain.NormalizeTemplateLibraryID(vals["template_library_id"])
		if templateLibraryID != "" && !m.hasApprovedTemplateLibrary(templateLibraryID) {
			m.status = "approved template library not found: " + templateLibraryID
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
		currentTemplateLibraryID := ""
		currentTemplateLibraryDrift := ""
		if binding, ok := m.activeProjectTemplateBinding(projectID); ok {
			currentTemplateLibraryID = binding.LibraryID
			currentTemplateLibraryDrift = strings.TrimSpace(binding.DriftStatus)
		}
		projectOp := "update"
		if isAdd || projectID == "" {
			projectOp = "create"
		}
		m.traceFormControlCharacterGuard("project", projectOp, "name", name)
		m.traceFormControlCharacterGuard("project", projectOp, "description", description)
		if isAdd || projectID == "" {
			m.mode = modeNone
			m.resetProjectFormState()
			m.clearProjectTemplateMigrationReview()
			return m, func() tea.Msg {
				project, err := m.svc.CreateProjectWithMetadata(context.Background(), app.CreateProjectInput{
					Name:              name,
					Description:       description,
					Kind:              kindID,
					TemplateLibraryID: templateLibraryID,
					Metadata:          metadata,
					UpdatedBy:         m.threadActorID(),
					UpdatedByName:     m.threadActorName(),
					UpdatedType:       m.threadActorType(),
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
		if templateLibraryID != "" &&
			templateLibraryID == currentTemplateLibraryID &&
			currentTemplateLibraryDrift == domain.ProjectTemplateBindingDriftUpdateAvailable {
			draft := pendingProjectTemplateReview{
				ProjectID:                projectID,
				Name:                     name,
				Description:              description,
				Kind:                     kindID,
				Metadata:                 metadata,
				RootPath:                 rootPath,
				TemplateLibraryID:        templateLibraryID,
				CurrentTemplateLibraryID: currentTemplateLibraryID,
			}
			return m, m.startProjectTemplateMigrationReview(draft)
		}
		m.mode = modeNone
		m.resetProjectFormState()
		m.clearProjectTemplateMigrationReview()
		return m, func() tea.Msg {
			project, err := m.svc.UpdateProject(context.Background(), app.UpdateProjectInput{
				ProjectID:     projectID,
				Name:          name,
				Description:   description,
				Kind:          kindID,
				Metadata:      metadata,
				UpdatedBy:     m.threadActorID(),
				UpdatedByName: m.threadActorName(),
				UpdatedType:   m.threadActorType(),
			})
			if err != nil {
				return actionMsg{err: err}
			}
			switch {
			case templateLibraryID == "" && currentTemplateLibraryID != "":
				if err := m.svc.UnbindProjectTemplateLibrary(context.Background(), app.UnbindProjectTemplateLibraryInput{
					ProjectID: project.ID,
				}); err != nil {
					return actionMsg{err: err}
				}
			case templateLibraryID != "" && (templateLibraryID != currentTemplateLibraryID || currentTemplateLibraryDrift == domain.ProjectTemplateBindingDriftUpdateAvailable):
				if _, err := m.svc.BindProjectTemplateLibrary(context.Background(), app.BindProjectTemplateLibraryInput{
					ProjectID:        project.ID,
					LibraryID:        templateLibraryID,
					BoundByActorID:   m.threadActorID(),
					BoundByActorName: m.threadActorName(),
					BoundByActorType: m.threadActorType(),
				}); err != nil {
					return actionMsg{err: err}
				}
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
	command = normalizeCommandPaletteToken(command)
	switch command {
	case "":
		m.status = "no command"
		return m, nil
	case "new-actionItem", "actionItem-new":
		return m, m.startActionItemForm(nil)
	case "new-subtask", "actionItem-subtask":
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		return m, m.startSubactionItemForm(actionItem)
	case "new-branch", "branch-new":
		if strings.TrimSpace(m.projectionRootActionItemID) != "" {
			m.status = "clear focus before creating a branch"
			m.startWarningModal(
				"Branch Creation Blocked",
				fmt.Sprintf("New branches are project-level items. Press %s to return to full board, then create the branch.", m.keys.clearFocus.Help().Key),
			)
			return m, nil
		}
		parent, ok := m.selectedBranchActionItem()
		if ok {
			return m, m.startBranchForm(&parent)
		}
		return m, m.startBranchForm(nil)
	case "new-phase", "phase-new":
		parent, ok := m.focusedScopeActionItemAtLevels("phase", "branch")
		if ok {
			return m, m.startPhaseForm(&parent)
		}
		if rootID := strings.TrimSpace(m.projectionRootActionItemID); rootID != "" {
			root, found := m.actionItemByID(rootID)
			if found {
				m.status = "phase creation blocked in current focus"
				m.startWarningModal(
					"Phase Creation Blocked",
					fmt.Sprintf("%s is a %s screen. Phases can only be created from project, branch, or phase screens.", root.Title, baseSearchLevelForActionItem(root)),
				)
				return m, nil
			}
		}
		return m, m.startPhaseForm(nil)
	case "edit-branch", "branch-edit":
		actionItem, ok := m.selectedBranchActionItem()
		if !ok {
			m.status = "select a branch to edit"
			return m, nil
		}
		return m, m.startActionItemForm(&actionItem)
	case "archive-branch", "branch-archive":
		if _, ok := m.selectedBranchActionItem(); !ok {
			m.status = "select a branch to archive"
			return m, nil
		}
		return m.confirmDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive branch")
	case "delete-branch", "branch-delete":
		if _, ok := m.selectedBranchActionItem(); !ok {
			m.status = "select a branch to delete"
			return m, nil
		}
		return m.confirmDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "delete branch")
	case "restore-branch", "branch-restore":
		actionItem, ok := m.selectedBranchActionItem()
		if !ok || actionItem.ArchivedAt == nil {
			m.status = "select an archived branch to restore"
			return m, nil
		}
		return m.confirmRestoreAction()
	case "edit-actionItem", "actionItem-edit":
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		return m, m.startActionItemForm(&actionItem)
	case "thread-item", "item-thread", "actionItem-thread":
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
	case "embeddings", "embeddings-status", "semantic-status":
		return m, m.startEmbeddingsStatus(false)
	case "embeddings-reindex", "semantic-reindex":
		return m, m.startEmbeddingsReindex(false)
	case "clear-query", "clear-search-query":
		return m, m.clearSearchQuery()
	case "reset-filters", "clear-search":
		return m, m.resetSearchFilters()
	case "toggle-archived":
		m.showArchived = !m.showArchived
		m.selectedActionItem = 0
		m.clearSelection()
		if m.showArchived {
			m.status = "showing archived tasks"
		} else {
			m.status = "hiding archived tasks"
		}
		return m, m.loadData
	case "toggle-selection-mode", "select-mode", "text-select":
		m.mouseSelectionMode = !m.mouseSelectionMode
		m.status = "ready"
		return m, nil
	case "focus-subtree", "zoom-actionItem":
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		if !m.activateSubtreeFocus(actionItem.ID) {
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
	case "toggle-select", "select-actionItem":
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		if m.toggleActionItemSelection(actionItem.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(actionItem.Title, 28), len(m.selectedActionItemIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "select actionItem",
				Target:  actionItem.Title,
			})
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(actionItem.Title, 28), len(m.selectedActionItemIDs))
			m.appendActivity(activityEntry{
				At:      time.Now().UTC(),
				Summary: "unselect actionItem",
				Target:  actionItem.Title,
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
		return m.moveSelectedActionItems(-1)
	case "bulk-move-right", "move-right-selected":
		return m.moveSelectedActionItems(1)
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
	case "auth-access", "auths", "access-review", "coordination", "recovery":
		return m, m.startAuthInventory(false)
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

// quickActionMode returns the screen mode that owns the active quick-actions overlay.
func (m Model) quickActionMode() inputMode {
	if m.mode == modeQuickActions && m.quickActionBackMode != modeNone {
		return m.quickActionBackMode
	}
	return m.mode
}

// quickActions returns state-aware quick actions for the active screen context.
func (m Model) quickActions() []quickActionItem {
	return m.quickActionsForMode(m.quickActionMode())
}

// quickActionsForMode resolves quick actions for one specific screen context.
func (m Model) quickActionsForMode(mode inputMode) []quickActionItem {
	switch mode {
	case modeAddActionItem, modeEditActionItem:
		return m.actionItemFormQuickActions(mode)
	case modeActionItemInfo:
		return m.actionItemInfoQuickActions()
	default:
		return m.boardQuickActions()
	}
}

// quickActionsTitle renders a context-aware quick-actions title.
func (m Model) quickActionsTitle() string {
	switch m.quickActionMode() {
	case modeActionItemInfo:
		return "Quick Actions: ActionItem Info"
	case modeAddActionItem:
		return "Quick Actions: New ActionItem"
	case modeEditActionItem:
		switch m.formFocus {
		case actionItemFieldSubtasks:
			return "Quick Actions: Subtasks"
		case actionItemFieldResources:
			return "Quick Actions: Resources"
		case actionItemFieldComments:
			return "Quick Actions: Comments"
		default:
			return "Quick Actions: Edit ActionItem"
		}
	default:
		return "Quick Actions"
	}
}

// boardQuickActions returns state-aware board quick actions with enabled entries first.
func (m Model) boardQuickActions() []quickActionItem {
	_, hasActionItem := m.selectedActionItemInCurrentColumn()
	hasSelection := len(m.selectedActionItemIDs) > 0
	enabled := make([]quickActionItem, 0, len(quickActionSpecs))
	disabled := make([]quickActionItem, 0, len(quickActionSpecs))
	for _, spec := range quickActionSpecs {
		available, reason := m.quickActionAvailability(spec.ID, hasActionItem, hasSelection)
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

// actionItemFormQuickActions resolves focused quick actions for actionItem add/edit screens.
func (m Model) actionItemFormQuickActions(_ inputMode) []quickActionItem {
	_, hasContextActionItem := m.actionItemFormContextActionItem()
	switch m.formFocus {
	case actionItemFieldSubtasks:
		items := []quickActionItem{{
			ID:             "actionItem-form-new-subtask",
			Label:          "Create Subtask",
			Enabled:        hasContextActionItem,
			DisabledReason: "save actionItem first",
		}}
		if subtask, ok := m.selectedActionItemFormSubtask(); ok {
			items = append([]quickActionItem{{
				ID:      "actionItem-form-open-subtask",
				Label:   "Open Selected Subtask",
				Enabled: true,
			}}, items...)
			_ = subtask
		}
		return items
	case actionItemFieldResources:
		enabled := hasContextActionItem
		reason := "save actionItem first"
		if enabled {
			reason = ""
		}
		label := "Attach Resource"
		if m.actionItemFormResourceCursor > 0 {
			label = "Replace Selected Resource"
		}
		return []quickActionItem{{
			ID:             "actionItem-form-resource-action",
			Label:          label,
			Enabled:        enabled,
			DisabledReason: reason,
		}}
	case actionItemFieldComments:
		return []quickActionItem{{
			ID:             "actionItem-form-open-thread",
			Label:          "Open Comments",
			Enabled:        hasContextActionItem,
			DisabledReason: "save actionItem first",
		}}
	case actionItemFieldDue, actionItemFieldLabels, actionItemFieldDependsOn, actionItemFieldBlockedBy:
		return []quickActionItem{{
			ID:      "actionItem-form-open-field",
			Label:   "Open Field Action",
			Enabled: true,
		}}
	default:
		if isActionItemFormMarkdownField(m.formFocus) {
			return []quickActionItem{{
				ID:      "actionItem-form-open-field",
				Label:   "Open Markdown Editor",
				Enabled: true,
			}}
		}
		return nil
	}
}

// actionItemInfoQuickActions resolves actionItem-info quick actions for the current actionItem and selected subtask.
func (m Model) actionItemInfoQuickActions() []quickActionItem {
	actionItem, ok := m.actionItemByID(strings.TrimSpace(m.actionItemInfoActionItemID))
	if !ok {
		return nil
	}
	items := []quickActionItem{
		{ID: "actionItem-info-edit", Label: "Edit ActionItem", Enabled: true},
		{ID: "actionItem-info-open-thread", Label: "Open Comments", Enabled: true},
		{ID: "actionItem-info-new-subtask", Label: "Create Subtask", Enabled: true},
	}
	candidate := m
	if subtask, ok := (&candidate).selectedActionItemInfoSubtask(actionItem); ok {
		state := candidate.lifecycleStateForActionItem(subtask)
		toggleLabel := "Mark Selected Subtask Complete"
		if state == domain.StateDone {
			toggleLabel = "Mark Selected Subtask Incomplete"
		}
		items = append([]quickActionItem{
			{ID: "actionItem-info-open-subtask", Label: "Open Selected Subtask", Enabled: true},
			{ID: "actionItem-info-toggle-subtask", Label: toggleLabel, Enabled: true},
		}, items...)
	}
	return items
}

// quickActionAvailability returns whether one board quick action can run in the current state.
func (m Model) quickActionAvailability(actionID string, hasActionItem bool, hasSelection bool) (bool, string) {
	switch actionID {
	case "actionItem-info", "edit-actionItem", "archive-actionItem", "hard-delete", "toggle-selection":
		if !hasActionItem {
			return false, "no actionItem selected"
		}
		return true, ""
	case "restore-actionItem":
		if actionItem, ok := m.selectedActionItemInCurrentColumn(); ok && actionItem.ArchivedAt != nil {
			return true, ""
		}
		if strings.TrimSpace(m.lastArchivedActionItemID) != "" {
			return true, ""
		}
		return false, "no archived actionItem selected"
	case "move-left":
		if !hasActionItem {
			return false, "no actionItem selected"
		}
		if m.selectedColumn <= 0 {
			return false, "already at first column"
		}
		return true, ""
	case "move-right":
		if !hasActionItem {
			return false, "no actionItem selected"
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
		if len(m.buildMoveSteps(m.sortedSelectedActionItemIDs(), -1)) == 0 {
			return false, "no movable tasks selected"
		}
		return true, ""
	case "bulk-move-right":
		if !hasSelection {
			return false, "no tasks selected"
		}
		if len(m.buildMoveSteps(m.sortedSelectedActionItemIDs(), 1)) == 0 {
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
	case "auth-access", "activity-log":
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

	contextMode := m.quickActionMode()
	traceActionItemScreenAction("quick_actions", "apply", "back_mode", modeKey(contextMode), "action_id", action.ID, "label", action.Label)
	m.mode = contextMode
	m.quickActionBackMode = modeNone
	switch action.ID {
	case "actionItem-form-open-field":
		if next, cmd, handled := m.openFocusedActionItemFormField(tea.KeyPressMsg{}); handled {
			return next, cmd
		}
		m.status = "no quick action for this field"
		return m, nil
	case "actionItem-form-open-thread":
		if actionItem, ok := m.actionItemFormContextActionItem(); ok {
			return m.startActionItemThreadWithPanel(actionItem, modeEditActionItem, threadPanelComments)
		}
		m.status = "save actionItem first to open comments"
		return m, nil
	case "actionItem-form-new-subtask":
		return m, m.startSubactionItemFormFromActionItemForm()
	case "actionItem-form-open-subtask":
		return m, m.openFocusedActionItemFormSubtask()
	case "actionItem-form-resource-action":
		return m, m.startActionItemFormResourcePickerFromFocus()
	case "actionItem-info-edit":
		actionItem, ok := m.actionItemByID(strings.TrimSpace(m.actionItemInfoActionItemID))
		if !ok {
			m.status = "actionItem not found"
			return m, nil
		}
		return m, m.startActionItemForm(&actionItem)
	case "actionItem-info-open-thread":
		actionItem, ok := m.actionItemByID(strings.TrimSpace(m.actionItemInfoActionItemID))
		if !ok {
			m.status = "actionItem not found"
			return m, nil
		}
		return m.startActionItemThreadWithPanel(actionItem, modeActionItemInfo, threadPanelComments)
	case "actionItem-info-new-subtask":
		actionItem, ok := m.actionItemByID(strings.TrimSpace(m.actionItemInfoActionItemID))
		if !ok {
			m.status = "actionItem not found"
			return m, nil
		}
		return m, m.startSubactionItemForm(actionItem)
	case "actionItem-info-open-subtask":
		actionItem, ok := m.actionItemByID(strings.TrimSpace(m.actionItemInfoActionItemID))
		if !ok {
			m.status = "actionItem not found"
			return m, nil
		}
		return m, m.openFocusedActionItemInfoSubtask(actionItem)
	case "actionItem-info-toggle-subtask":
		actionItem, ok := m.actionItemByID(strings.TrimSpace(m.actionItemInfoActionItemID))
		if !ok {
			m.status = "actionItem not found"
			return m, nil
		}
		return m.toggleFocusedSubactionItemCompletion(actionItem)
	case "actionItem-info":
		m.mode = modeNone
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		m.openActionItemInfo(actionItem.ID, "actionItem info")
		return m, nil
	case "edit-actionItem":
		m.mode = modeNone
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		return m, m.startActionItemForm(&actionItem)
	case "move-left":
		m.mode = modeNone
		return m.moveSelectedActionItem(-1)
	case "move-right":
		m.mode = modeNone
		return m.moveSelectedActionItem(1)
	case "archive-actionItem":
		m.mode = modeNone
		return m.confirmDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive actionItem")
	case "restore-actionItem":
		m.mode = modeNone
		return m.confirmRestoreAction()
	case "hard-delete":
		m.mode = modeNone
		return m.confirmDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete actionItem")
	case "toggle-selection":
		m.mode = modeNone
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if !ok {
			m.status = "no actionItem selected"
			return m, nil
		}
		if m.toggleActionItemSelection(actionItem.ID) {
			m.status = fmt.Sprintf("selected %q (%d total)", truncate(actionItem.Title, 28), len(m.selectedActionItemIDs))
		} else {
			m.status = fmt.Sprintf("unselected %q (%d total)", truncate(actionItem.Title, 28), len(m.selectedActionItemIDs))
		}
		return m, nil
	case "clear-selection":
		m.mode = modeNone
		count := m.clearSelection()
		if count == 0 {
			m.status = "selection already empty"
			return m, nil
		}
		m.status = fmt.Sprintf("cleared %d selected tasks", count)
		return m, nil
	case "bulk-move-left":
		m.mode = modeNone
		return m.moveSelectedActionItems(-1)
	case "bulk-move-right":
		m.mode = modeNone
		return m.moveSelectedActionItems(1)
	case "bulk-archive":
		m.mode = modeNone
		return m.confirmBulkDeleteAction(app.DeleteModeArchive, m.confirmArchive, "archive selected")
	case "bulk-hard-delete":
		m.mode = modeNone
		return m.confirmBulkDeleteAction(app.DeleteModeHard, m.confirmHardDelete, "hard delete selected")
	case "undo":
		m.mode = modeNone
		return m.undoLastMutation()
	case "redo":
		m.mode = modeNone
		return m.redoLastMutation()
	case "auth-access":
		return m, m.startAuthInventory(false)
	case "activity-log":
		m.mode = modeNone
		return m, m.openActivityLog()
	default:
		m.status = "unknown quick action"
		return m, nil
	}
}

// createActionItem creates actionItem.
func (m Model) createActionItem(in app.CreateActionItemInput) (tea.Model, tea.Cmd) {
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
		actionItem, err := m.svc.CreateActionItem(context.Background(), in)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{status: "actionItem created", reload: true, focusActionItemID: actionItem.ID}
	}
}

// moveSelectedActionItem moves the currently focused actionItem one column left/right.
func (m Model) moveSelectedActionItem(delta int) (tea.Model, tea.Cmd) {
	actionItem, ok := m.selectedActionItemInCurrentColumn()
	if !ok {
		m.status = "no actionItem selected"
		return m, nil
	}
	return m.moveActionItemIDs([]string{actionItem.ID}, delta, "move actionItem", actionItem.Title, false)
}

// moveSelectedActionItems moves every selected actionItem one column left/right.
func (m Model) moveSelectedActionItems(delta int) (tea.Model, tea.Cmd) {
	actionItemIDs := m.sortedSelectedActionItemIDs()
	if len(actionItemIDs) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	label := "bulk move right"
	if delta < 0 {
		label = "bulk move left"
	}
	return m.moveActionItemIDs(actionItemIDs, delta, label, fmt.Sprintf("%d tasks", len(actionItemIDs)), true)
}

// moveActionItemIDs moves the provided actionItem ids and records undo/redo history.
func (m Model) moveActionItemIDs(actionItemIDs []string, delta int, label, target string, bulk bool) (tea.Model, tea.Cmd) {
	steps := m.buildMoveSteps(actionItemIDs, delta)
	if len(steps) == 0 {
		m.status = "no movable tasks selected"
		return m, nil
	}
	direction := "right"
	if delta < 0 {
		direction = "left"
	}
	status := "actionItem moved"
	if bulk {
		status = fmt.Sprintf("moved %d tasks %s", len(steps), direction)
	}
	focusActionItemID := steps[0].ActionItemID
	if bulk {
		focusActionItemID = ""
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
			if _, err := m.svc.MoveActionItem(context.Background(), step.ActionItemID, step.ToColumnID, step.ToPosition); err != nil {
				return actionMsg{err: err}
			}
		}
		return actionMsg{
			status:            status,
			reload:            true,
			focusActionItemID: focusActionItemID,
			historyPush:       &history,
			activityItem:      &activity,
		}
	}
}

// deleteSelectedActionItem deletes or archives the currently focused actionItem.
func (m Model) deleteSelectedActionItem(mode app.DeleteMode) (tea.Model, tea.Cmd) {
	actionItem, ok := m.selectedActionItemInCurrentColumn()
	if !ok {
		m.status = "no actionItem selected"
		return m, nil
	}
	return m.deleteActionItemIDs([]string{actionItem.ID}, mode)
}

// deleteActionItemIDs archives/deletes actionItem ids and records undo metadata when possible.
func (m Model) deleteActionItemIDs(actionItemIDs []string, mode app.DeleteMode) (tea.Model, tea.Cmd) {
	ids := m.normalizeKnownActionItemIDs(actionItemIDs)
	if len(ids) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	undoable := mode != app.DeleteModeHard
	label := "archive actionItem"
	if mode == app.DeleteModeHard {
		label = "hard delete actionItem"
	}
	if len(ids) > 1 {
		if mode == app.DeleteModeHard {
			label = "bulk hard delete"
		} else {
			label = "bulk archive"
		}
	}
	status := "actionItem archived"
	if mode == app.DeleteModeHard {
		status = "actionItem deleted"
	}
	if len(ids) > 1 {
		if mode == app.DeleteModeHard {
			status = fmt.Sprintf("deleted %d tasks", len(ids))
		} else {
			status = fmt.Sprintf("archived %d tasks", len(ids))
		}
	}

	steps := make([]historyStep, 0, len(ids))
	for _, actionItemID := range ids {
		step := historyStep{ActionItemID: actionItemID}
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
		m.lastArchivedActionItemID = ids[len(ids)-1]
	}
	return m, func() tea.Msg {
		for _, actionItemID := range ids {
			if err := m.svc.DeleteActionItem(context.Background(), actionItemID, mode); err != nil {
				return actionMsg{err: err}
			}
		}
		return actionMsg{
			status:             status,
			reload:             true,
			clearActionItemIDs: ids,
			historyPush:        &history,
			activityItem:       &activity,
		}
	}
}

// confirmDeleteAction opens a confirmation modal when configured, or executes directly.
func (m Model) confirmDeleteAction(mode app.DeleteMode, needsConfirm bool, label string) (tea.Model, tea.Cmd) {
	actionItem, ok := m.selectedActionItemInCurrentColumn()
	if !ok {
		m.status = "no actionItem selected"
		return m, nil
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = "delete actionItem"
	}
	if !needsConfirm {
		return m.deleteActionItemIDs([]string{actionItem.ID}, mode)
	}
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:          "delete",
		ActionItem:    actionItem,
		ActionItemIDs: []string{actionItem.ID},
		Mode:          mode,
		Label:         label,
	}
	m.confirmChoice = 1
	m.status = "confirm action"
	return m, nil
}

// confirmBulkDeleteAction confirms and applies bulk archive/hard-delete operations.
func (m Model) confirmBulkDeleteAction(mode app.DeleteMode, needsConfirm bool, label string) (tea.Model, tea.Cmd) {
	actionItemIDs := m.sortedSelectedActionItemIDs()
	if len(actionItemIDs) == 0 {
		m.status = "no tasks selected"
		return m, nil
	}
	if !needsConfirm {
		return m.deleteActionItemIDs(actionItemIDs, mode)
	}
	actionItem, _ := m.actionItemByID(actionItemIDs[0])
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:          "delete",
		ActionItem:    actionItem,
		ActionItemIDs: actionItemIDs,
		Mode:          mode,
		Label:         label,
	}
	m.confirmChoice = 1
	m.status = "confirm action"
	return m, nil
}

// restoreActionItem restores the most-recent archived actionItem or selected archived actionItem.
func (m Model) restoreActionItem() (tea.Model, tea.Cmd) {
	actionItemID := m.lastArchivedActionItemID
	if actionItemID == "" {
		actionItem, ok := m.selectedActionItemInCurrentColumn()
		if ok && actionItem.ArchivedAt != nil {
			actionItemID = actionItem.ID
		}
	}
	if actionItemID == "" {
		m.status = "nothing to restore"
		return m, nil
	}
	return m.restoreActionItemIDs([]string{actionItemID}, "actionItem restored", "restore actionItem")
}

// restoreActionItemIDs restores tasks and records undo history.
func (m Model) restoreActionItemIDs(actionItemIDs []string, status, label string) (tea.Model, tea.Cmd) {
	ids := make([]string, 0, len(actionItemIDs))
	for _, actionItemID := range actionItemIDs {
		actionItemID = strings.TrimSpace(actionItemID)
		if actionItemID == "" {
			continue
		}
		ids = append(ids, actionItemID)
	}
	if len(ids) == 0 {
		m.status = "nothing to restore"
		return m, nil
	}
	steps := make([]historyStep, 0, len(ids))
	for _, actionItemID := range ids {
		steps = append(steps, historyStep{
			Kind:         historyStepRestore,
			ActionItemID: actionItemID,
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
		for _, actionItemID := range ids {
			if _, err := m.svc.RestoreActionItem(context.Background(), actionItemID); err != nil {
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
	actionItem, ok := m.selectedActionItemInCurrentColumn()
	if ok && actionItem.ArchivedAt == nil {
		ok = false
	}
	if !m.confirmRestore || !ok {
		return m.restoreActionItem()
	}
	m.mode = modeConfirmAction
	m.pendingConfirm = confirmAction{
		Kind:          "restore",
		ActionItem:    actionItem,
		ActionItemIDs: []string{actionItem.ID},
		Mode:          app.DeleteModeArchive,
		Label:         "restore actionItem",
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
		actionItemIDs := action.ActionItemIDs
		if len(actionItemIDs) == 0 && strings.TrimSpace(action.ActionItem.ID) != "" {
			actionItemIDs = []string{action.ActionItem.ID}
		}
		return m.deleteActionItemIDs(actionItemIDs, action.Mode)
	case "restore":
		actionItemIDs := action.ActionItemIDs
		if len(actionItemIDs) == 0 && strings.TrimSpace(action.ActionItem.ID) != "" {
			actionItemIDs = []string{action.ActionItem.ID}
		}
		return m.restoreActionItemIDs(actionItemIDs, "actionItem restored", "restore actionItem")
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
	case "approve-auth-request":
		if strings.TrimSpace(action.AuthRequestID) == "" {
			m.status = "missing auth request"
			return m, nil
		}
		sessionTTL, err := time.ParseDuration(strings.TrimSpace(action.AuthRequestTTL))
		if strings.TrimSpace(action.AuthRequestTTL) != "" && err != nil {
			m.status = err.Error()
			return m, nil
		}
		resolvedBy := m.threadActorID()
		resolvedType := m.threadActorType()
		note := strings.TrimSpace(action.AuthRequestNote)
		return m, func() tea.Msg {
			result, err := m.svc.ApproveAuthRequest(context.Background(), app.ApproveAuthRequestInput{
				RequestID:      action.AuthRequestID,
				Path:           strings.TrimSpace(action.AuthRequestPath),
				SessionTTL:     sessionTTL,
				ResolvedBy:     resolvedBy,
				ResolvedType:   resolvedType,
				ResolutionNote: note,
			})
			if err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{
				status:         "auth request approved",
				reload:         true,
				openAuthAccess: action.ReturnToAuthAccess,
				projectID:      result.Request.ProjectID,
			}
		}
	case "deny-auth-request":
		if strings.TrimSpace(action.AuthRequestID) == "" {
			m.status = "missing auth request"
			return m, nil
		}
		resolvedBy := m.threadActorID()
		resolvedType := m.threadActorType()
		note := strings.TrimSpace(action.AuthRequestNote)
		return m, func() tea.Msg {
			if _, err := m.svc.DenyAuthRequest(context.Background(), app.DenyAuthRequestInput{
				RequestID:      action.AuthRequestID,
				ResolvedBy:     resolvedBy,
				ResolvedType:   resolvedType,
				ResolutionNote: note,
			}); err != nil {
				return actionMsg{err: err}
			}
			return actionMsg{status: "auth request denied", reload: true, openAuthAccess: action.ReturnToAuthAccess}
		}
	case "revoke-auth-session":
		sessionID := strings.TrimSpace(action.AuthSessionID)
		if sessionID == "" {
			m.status = "missing auth session"
			return m, nil
		}
		return m, func() tea.Msg {
			revoked, err := m.svc.RevokeAuthSession(context.Background(), sessionID, "revoked via TUI coordination")
			if err != nil {
				return actionMsg{err: err}
			}
			status := "auth session revoked"
			if principal := firstNonEmptyTrimmed(revoked.PrincipalName, revoked.PrincipalID); principal != "" {
				status = "revoked auth session for " + principal
			}
			return actionMsg{status: status, reload: true, openAuthAccess: action.ReturnToAuthAccess}
		}
	case "revoke-capability-lease":
		instanceID := strings.TrimSpace(action.LeaseInstanceID)
		if instanceID == "" {
			m.status = "missing capability lease"
			return m, nil
		}
		return m, func() tea.Msg {
			revoked, err := m.svc.RevokeCapabilityLease(context.Background(), app.RevokeCapabilityLeaseInput{
				AgentInstanceID: instanceID,
				Reason:          "revoked via TUI coordination",
			})
			if err != nil {
				return actionMsg{err: err}
			}
			status := "capability lease revoked"
			if agent := firstNonEmptyTrimmed(revoked.AgentName, revoked.InstanceID); agent != "" {
				status = "revoked capability lease for " + agent
			}
			return actionMsg{status: status, reload: true, openAuthAccess: action.ReturnToAuthAccess}
		}
	case "update-handoff-status":
		handoffID := strings.TrimSpace(action.HandoffID)
		statusValue := domain.NormalizeHandoffStatus(domain.HandoffStatus(action.HandoffStatus))
		if handoffID == "" {
			m.status = "missing handoff"
			return m, nil
		}
		if statusValue == "" {
			m.status = "missing handoff status"
			return m, nil
		}
		updatedBy := m.threadActorID()
		updatedType := m.threadActorType()
		return m, func() tea.Msg {
			updated, err := m.svc.UpdateHandoff(context.Background(), app.UpdateHandoffInput{
				HandoffID:    handoffID,
				Status:       statusValue,
				Summary:      strings.TrimSpace(action.HandoffSummary),
				UpdatedBy:    updatedBy,
				UpdatedType:  updatedType,
				ResolvedBy:   updatedBy,
				ResolvedType: updatedType,
			})
			if err != nil {
				return actionMsg{err: err}
			}
			status := "handoff updated"
			if normalized := domain.NormalizeHandoffStatus(updated.Status); normalized != "" {
				status = "handoff marked " + string(normalized)
			}
			return actionMsg{status: status, reload: true, openAuthAccess: action.ReturnToAuthAccess}
		}
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
	if m.mode == modeDescriptionEditor {
		scrollDelta := 3
		if m.descriptionEditorMode == descriptionEditorViewModePreview {
			m.syncDescriptionEditorViewportLayout()
			switch msg.Button {
			case tea.MouseWheelUp:
				m.descriptionPreview.ScrollUp(scrollDelta)
			case tea.MouseWheelDown:
				m.descriptionPreview.ScrollDown(scrollDelta)
			default:
				return m, nil
			}
			return m, nil
		}
		switch msg.Button {
		case tea.MouseWheelUp:
			for i := 0; i < scrollDelta; i++ {
				m.descriptionEditorInput.CursorUp()
			}
		case tea.MouseWheelDown:
			for i := 0; i < scrollDelta; i++ {
				m.descriptionEditorInput.CursorDown()
			}
		default:
			return m, nil
		}
		m.syncDescriptionPreviewOffsetToEditor()
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
	if m.mode == modeActionItemInfo {
		actionItem, ok := m.actionItemInfoActionItem()
		if !ok {
			m.closeActionItemInfo("actionItem info unavailable")
			return m, nil
		}
		m.syncActionItemInfoDetailsViewport(actionItem)
		m.syncActionItemInfoBodyViewport(actionItem)
		switch msg.Button {
		case tea.MouseWheelUp:
			m.actionItemInfoBody.ScrollUp(3)
		case tea.MouseWheelDown:
			m.actionItemInfoBody.ScrollDown(3)
		}
		return m, nil
	}
	if m.mode == modeTemplateMigrationReview {
		switch msg.Button {
		case tea.MouseWheelUp:
			m.actionItemInfoBody.ScrollUp(3)
		case tea.MouseWheelDown:
			m.actionItemInfoBody.ScrollDown(3)
		}
		return m, nil
	}
	if m.mode == modeAuthInventory {
		switch msg.Button {
		case tea.MouseWheelUp:
			m.authInventoryMoveSelection(-1)
		case tea.MouseWheelDown:
			m.authInventoryMoveSelection(1)
		default:
			return m, nil
		}
		m.syncAuthInventoryViewport()
		return m, nil
	}
	if m.mode == modeAddActionItem || m.mode == modeEditActionItem {
		m.syncActionItemFormViewportToFocus()
		switch msg.Button {
		case tea.MouseWheelUp:
			m.actionItemInfoBody.ScrollUp(3)
		case tea.MouseWheelDown:
			m.actionItemInfoBody.ScrollDown(3)
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
	if m.mode == modeProjectKindPicker {
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.projectKindPickerIndex > 0 {
				m.projectKindPickerIndex--
			}
		case tea.MouseWheelDown:
			if m.projectKindPickerIndex < len(m.projectKindPickerItems)-1 {
				m.projectKindPickerIndex++
			}
		default:
			return m, nil
		}
		return m, nil
	}
	if m.mode == modeTemplateLibraryPicker {
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.templateLibraryPickerIndex > 0 {
				m.templateLibraryPickerIndex--
			}
		case tea.MouseWheelDown:
			if m.templateLibraryPickerIndex < len(m.templateLibraryPickerItems)-1 {
				m.templateLibraryPickerIndex++
			}
		default:
			return m, nil
		}
		return m, nil
	}
	if m.mode != modeNone {
		return m, nil
	}

	tasks := m.currentColumnActionItems()
	if len(tasks) == 0 {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		if m.selectedActionItem > 0 {
			m.selectedActionItem--
		}
	case tea.MouseWheelDown:
		if m.selectedActionItem < len(tasks)-1 {
			m.selectedActionItem++
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
		tasks := m.currentColumnActionItems()
		if len(tasks) > 0 {
			row := relativeY - 2
			m.selectedActionItem = clamp(m.actionItemIndexAtRow(tasks, row), 0, len(tasks)-1)
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
		m.selectedActionItem = 0
		return
	}
	m.selectedProject = clamp(m.selectedProject, 0, len(m.projects)-1)

	if len(m.columns) == 0 {
		m.selectedColumn = 0
		m.selectedActionItem = 0
		return
	}
	m.selectedColumn = clamp(m.selectedColumn, 0, len(m.columns)-1)
	colActionItems := m.currentColumnActionItems()
	if len(colActionItems) == 0 {
		m.selectedActionItem = 0
		return
	}
	m.selectedActionItem = clamp(m.selectedActionItem, 0, len(colActionItems)-1)
}

// retainSelectionForLoadedActionItems drops selected actionItem ids that are no longer loaded.
func (m *Model) retainSelectionForLoadedActionItems() {
	if len(m.selectedActionItemIDs) == 0 {
		return
	}
	known := map[string]struct{}{}
	for _, actionItem := range m.tasks {
		known[actionItem.ID] = struct{}{}
	}
	for actionItemID := range m.selectedActionItemIDs {
		if _, ok := known[actionItemID]; !ok {
			delete(m.selectedActionItemIDs, actionItemID)
		}
	}
}

// isActionItemSelected reports whether a actionItem id is currently in the multi-select set.
func (m Model) isActionItemSelected(actionItemID string) bool {
	_, ok := m.selectedActionItemIDs[strings.TrimSpace(actionItemID)]
	return ok
}

// toggleActionItemSelection adds/removes a actionItem id from the current selection.
func (m *Model) toggleActionItemSelection(actionItemID string) bool {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return false
	}
	if m.selectedActionItemIDs == nil {
		m.selectedActionItemIDs = map[string]struct{}{}
	}
	if _, ok := m.selectedActionItemIDs[actionItemID]; ok {
		delete(m.selectedActionItemIDs, actionItemID)
		return false
	}
	m.selectedActionItemIDs[actionItemID] = struct{}{}
	return true
}

// clearSelection clears all selected actionItem ids and returns the previous count.
func (m *Model) clearSelection() int {
	count := len(m.selectedActionItemIDs)
	if count == 0 {
		return 0
	}
	m.selectedActionItemIDs = map[string]struct{}{}
	return count
}

// unselectActionItems removes provided actionItem ids from multi-select state.
func (m *Model) unselectActionItems(actionItemIDs []string) int {
	if len(m.selectedActionItemIDs) == 0 {
		return 0
	}
	removed := 0
	for _, actionItemID := range actionItemIDs {
		actionItemID = strings.TrimSpace(actionItemID)
		if actionItemID == "" {
			continue
		}
		if _, ok := m.selectedActionItemIDs[actionItemID]; !ok {
			continue
		}
		delete(m.selectedActionItemIDs, actionItemID)
		removed++
	}
	return removed
}

// sortedSelectedActionItemIDs returns selected ids in board display order.
func (m Model) sortedSelectedActionItemIDs() []string {
	if len(m.selectedActionItemIDs) == 0 {
		return nil
	}
	actionItemIDs := make([]string, 0, len(m.selectedActionItemIDs))
	for actionItemID := range m.selectedActionItemIDs {
		actionItemIDs = append(actionItemIDs, actionItemID)
	}
	return m.normalizeKnownActionItemIDs(actionItemIDs)
}

// normalizeKnownActionItemIDs returns deduplicated actionItem ids in deterministic board order.
func (m Model) normalizeKnownActionItemIDs(actionItemIDs []string) []string {
	if len(actionItemIDs) == 0 {
		return nil
	}
	needed := map[string]struct{}{}
	for _, actionItemID := range actionItemIDs {
		actionItemID = strings.TrimSpace(actionItemID)
		if actionItemID == "" {
			continue
		}
		needed[actionItemID] = struct{}{}
	}
	if len(needed) == 0 {
		return nil
	}
	out := make([]string, 0, len(needed))
	seen := map[string]struct{}{}
	for _, column := range m.columns {
		for _, actionItem := range m.tasksForColumn(column.ID) {
			if _, ok := needed[actionItem.ID]; !ok {
				continue
			}
			if _, ok := seen[actionItem.ID]; ok {
				continue
			}
			seen[actionItem.ID] = struct{}{}
			out = append(out, actionItem.ID)
		}
	}
	for _, actionItemID := range actionItemIDs {
		if _, ok := seen[actionItemID]; ok {
			continue
		}
		if _, ok := m.actionItemByID(actionItemID); !ok {
			continue
		}
		seen[actionItemID] = struct{}{}
		out = append(out, actionItemID)
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
				if _, err := m.svc.MoveActionItem(context.Background(), step.ActionItemID, columnID, position); err != nil {
					return actionMsg{err: err}
				}
			case historyStepArchive:
				if undo {
					if _, err := m.svc.RestoreActionItem(context.Background(), step.ActionItemID); err != nil {
						return actionMsg{err: err}
					}
				} else {
					if err := m.svc.DeleteActionItem(context.Background(), step.ActionItemID, app.DeleteModeArchive); err != nil {
						return actionMsg{err: err}
					}
					clearIDs = append(clearIDs, step.ActionItemID)
				}
			case historyStepRestore:
				if undo {
					if err := m.svc.DeleteActionItem(context.Background(), step.ActionItemID, app.DeleteModeArchive); err != nil {
						return actionMsg{err: err}
					}
					clearIDs = append(clearIDs, step.ActionItemID)
				} else {
					if _, err := m.svc.RestoreActionItem(context.Background(), step.ActionItemID); err != nil {
						return actionMsg{err: err}
					}
				}
			case historyStepHardDelete:
				if undo {
					return actionMsg{status: "undo failed: hard delete cannot be restored"}
				}
				if err := m.svc.DeleteActionItem(context.Background(), step.ActionItemID, app.DeleteModeHard); err != nil {
					return actionMsg{err: err}
				}
				clearIDs = append(clearIDs, step.ActionItemID)
			}
		}
		status := "redo complete"
		activitySummary := "redo"
		msg := actionMsg{
			reload:             true,
			clearActionItemIDs: clearIDs,
			historyRedo:        &set,
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

// buildMoveSteps computes move history steps for actionItem ids with deterministic ordering.
func (m Model) buildMoveSteps(actionItemIDs []string, delta int) []historyStep {
	if delta == 0 {
		return nil
	}
	ids := m.normalizeKnownActionItemIDs(actionItemIDs)
	if len(ids) == 0 {
		return nil
	}
	colIndexByID := map[string]int{}
	for idx, column := range m.columns {
		colIndexByID[column.ID] = idx
	}
	steps := make([]historyStep, 0, len(ids))
	for _, actionItemID := range ids {
		actionItem, ok := m.actionItemByID(actionItemID)
		if !ok {
			continue
		}
		fromColIdx, ok := colIndexByID[actionItem.ColumnID]
		if !ok {
			continue
		}
		toColIdx := fromColIdx + delta
		if toColIdx < 0 || toColIdx >= len(m.columns) {
			continue
		}
		steps = append(steps, historyStep{
			Kind:         historyStepMove,
			ActionItemID: actionItem.ID,
			FromColumnID: actionItem.ColumnID,
			FromPosition: actionItem.Position,
			ToColumnID:   m.columns[toColIdx].ID,
		})
	}
	if len(steps) == 0 {
		return nil
	}
	sort.SliceStable(steps, func(i, j int) bool {
		iActionItem, _ := m.actionItemByID(steps[i].ActionItemID)
		jActionItem, _ := m.actionItemByID(steps[j].ActionItemID)
		if iActionItem.ColumnID == jActionItem.ColumnID {
			if iActionItem.Position == jActionItem.Position {
				return iActionItem.ID < jActionItem.ID
			}
			return iActionItem.Position < jActionItem.Position
		}
		return colIndexByID[iActionItem.ColumnID] < colIndexByID[jActionItem.ColumnID]
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

// groupLabelForActionItem returns the swimlane/group label for a actionItem under current settings.
func (m Model) groupLabelForActionItem(actionItem domain.ActionItem) string {
	switch normalizeBoardGroupBy(m.boardGroupBy) {
	case "priority":
		switch actionItem.Priority {
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
		switch strings.ToLower(strings.TrimSpace(string(actionItem.LifecycleState))) {
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
		return "ActionItems"
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

// currentColumnActionItems returns current column tasks.
func (m Model) currentColumnActionItems() []domain.ActionItem {
	columnID, ok := m.currentColumnID()
	if !ok {
		return nil
	}
	return m.boardActionItemsForColumn(columnID)
}

// boardActionItemsForColumn returns only board-visible tasks for a column.
func (m Model) boardActionItemsForColumn(columnID string) []domain.ActionItem {
	columnActionItems := m.tasksForColumn(columnID)
	if len(columnActionItems) == 0 {
		return nil
	}
	includeSubtasks := m.focusedScopeShowsSubtasks()
	out := make([]domain.ActionItem, 0, len(columnActionItems))
	for _, actionItem := range columnActionItems {
		if actionItem.Kind == domain.KindSubtask && !includeSubtasks {
			continue
		}
		out = append(out, actionItem)
	}
	return out
}

// focusedScopeShowsSubtasks reports whether the current focused scope should render subtask rows.
func (m Model) focusedScopeShowsSubtasks() bool {
	rootID := strings.TrimSpace(m.projectionRootActionItemID)
	// Once focused, show all direct children regardless of kind. This keeps
	// focus navigation resilient for older/irregular data where subtasks were
	// attached outside actionItem/subtask roots.
	return rootID != ""
}

// tasksForColumn handles tasks for column.
func (m Model) tasksForColumn(columnID string) []domain.ActionItem {
	out := make([]domain.ActionItem, 0)
	projected := m.projectedActionItemSet()
	for _, actionItem := range m.tasks {
		if actionItem.ColumnID != columnID {
			continue
		}
		if _, ok := projected[actionItem.ID]; !ok {
			continue
		}
		out = append(out, actionItem)
	}
	ordered := orderActionItemsByHierarchy(out)
	groupBy := normalizeBoardGroupBy(m.boardGroupBy)
	if groupBy != "none" {
		sort.SliceStable(ordered, func(i, j int) bool {
			iRank := actionItemGroupRank(ordered[i], groupBy)
			jRank := actionItemGroupRank(ordered[j], groupBy)
			if iRank == jRank {
				return false
			}
			return iRank < jRank
		})
	}
	return ordered
}

// baseSearchLevelForActionItem infers a canonical hierarchy level from one actionItem's scope/kind.
func baseSearchLevelForActionItem(actionItem domain.ActionItem) string {
	switch domain.NormalizeKindAppliesTo(actionItem.Scope) {
	case domain.KindAppliesToBranch:
		return "branch"
	case domain.KindAppliesToPhase:
		return "phase"
	case domain.KindAppliesToActionItem:
		return "actionItem"
	case domain.KindAppliesToSubtask:
		return "subtask"
	}
	switch strings.TrimSpace(strings.ToLower(string(actionItem.Kind))) {
	case "branch":
		return "branch"
	case "phase":
		return "phase"
	case "subtask":
		return "subtask"
	case "actionItem":
		return "actionItem"
	}
	if strings.TrimSpace(actionItem.ParentID) != "" {
		return "subtask"
	}
	return "actionItem"
}

// searchLevelByActionItemID resolves one canonical hierarchy level per actionItem ID.
func (m Model) searchLevelByActionItemID(tasks []domain.ActionItem) map[string]string {
	byID := map[string]domain.ActionItem{}
	for _, actionItem := range m.tasks {
		byID[actionItem.ID] = actionItem
	}
	for _, actionItem := range tasks {
		byID[actionItem.ID] = actionItem
	}
	if len(byID) == 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	var resolve func(string, map[string]struct{}) string
	resolve = func(actionItemID string, visiting map[string]struct{}) string {
		actionItemID = strings.TrimSpace(actionItemID)
		if actionItemID == "" {
			return "actionItem"
		}
		if level, ok := out[actionItemID]; ok {
			return level
		}
		actionItem, ok := byID[actionItemID]
		if !ok {
			return "actionItem"
		}
		if _, seen := visiting[actionItemID]; seen {
			return "actionItem"
		}
		visiting[actionItemID] = struct{}{}
		level := baseSearchLevelForActionItem(actionItem)
		delete(visiting, actionItemID)
		out[actionItemID] = level
		return level
	}
	for actionItemID := range byID {
		resolve(actionItemID, map[string]struct{}{})
	}
	return out
}

// actionItemMatchesSearchLevels reports whether one actionItem passes active search level filters.
func (m Model) actionItemMatchesSearchLevels(actionItem domain.ActionItem, levelByActionItemID map[string]string) bool {
	enabled := canonicalSearchLevels(m.searchLevels)
	enabledSet := make(map[string]struct{}, len(enabled))
	for _, level := range enabled {
		enabledSet[level] = struct{}{}
	}
	if _, ok := enabledSet["project"]; ok {
		return true
	}
	level := canonicalSearchLevel(levelByActionItemID[actionItem.ID])
	if level == "" {
		level = baseSearchLevelForActionItem(actionItem)
	}
	_, ok := enabledSet[level]
	return ok
}

// filterActionItemMatchesBySearchLevels keeps only search matches that satisfy level filters.
func (m Model) filterActionItemMatchesBySearchLevels(matches []app.ActionItemMatch) []app.ActionItemMatch {
	if len(matches) == 0 {
		return nil
	}
	tasks := make([]domain.ActionItem, 0, len(matches))
	for _, match := range matches {
		tasks = append(tasks, match.ActionItem)
	}
	levelByActionItemID := m.searchLevelByActionItemID(tasks)
	out := make([]app.ActionItemMatch, 0, len(matches))
	for _, match := range matches {
		if !m.actionItemMatchesSearchLevels(match.ActionItem, levelByActionItemID) {
			continue
		}
		out = append(out, match)
	}
	return out
}

// tasksByID builds a lookup map for loaded tasks keyed by actionItem ID.
func (m Model) tasksByID() map[string]domain.ActionItem {
	out := make(map[string]domain.ActionItem, len(m.tasks))
	for _, actionItem := range m.tasks {
		out[actionItem.ID] = actionItem
	}
	return out
}

// projectedActionItemSet returns every actionItem ID visible in the current board scope.
func (m Model) projectedActionItemSet() map[string]struct{} {
	visible := map[string]struct{}{}
	rootID := strings.TrimSpace(m.projectionRootActionItemID)
	if rootID == "" {
		known := m.tasksByID()
		for _, actionItem := range m.tasks {
			parentID := strings.TrimSpace(actionItem.ParentID)
			if parentID == "" {
				visible[actionItem.ID] = struct{}{}
				continue
			}
			// Preserve orphaned tasks in project scope so they remain recoverable in UI.
			if _, ok := known[parentID]; !ok {
				visible[actionItem.ID] = struct{}{}
			}
		}
		return visible
	}
	if _, ok := m.actionItemByID(rootID); !ok {
		return visible
	}
	for _, actionItem := range m.tasks {
		if strings.TrimSpace(actionItem.ParentID) == rootID {
			visible[actionItem.ID] = struct{}{}
		}
	}
	return visible
}

// projectionBreadcrumb returns the active subtree breadcrumb path.
func (m Model) projectionBreadcrumb() string {
	rootID := strings.TrimSpace(m.projectionRootActionItemID)
	if rootID == "" {
		return ""
	}
	root, ok := m.actionItemByID(rootID)
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
		parent, found := m.actionItemByID(parentID)
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
	rootID := strings.TrimSpace(m.projectionRootActionItemID)
	projectName = strings.TrimSpace(projectName)
	if rootID == "" {
		if projectName == "" {
			return "(project)", "(project)"
		}
		return projectName, projectName
	}
	root, ok := m.actionItemByID(rootID)
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
		parent, found := m.actionItemByID(parentID)
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

// actionItemAttentionCount returns unresolved attention signals for one board-visible actionItem.
func (m Model) actionItemAttentionCount(actionItem domain.ActionItem, byID map[string]domain.ActionItem) int {
	count := 0
	for _, depID := range uniqueTrimmed(actionItem.Metadata.DependsOn) {
		depActionItem, ok := byID[depID]
		if !ok || m.lifecycleStateForActionItem(depActionItem) != domain.StateDone {
			count++
		}
	}
	for _, blockerID := range uniqueTrimmed(actionItem.Metadata.BlockedBy) {
		blockerActionItem, ok := byID[blockerID]
		if !ok || m.lifecycleStateForActionItem(blockerActionItem) != domain.StateDone {
			count++
		}
	}
	if strings.TrimSpace(actionItem.Metadata.BlockedReason) != "" {
		count++
	}
	return count
}

// scopeAttentionSummary computes compact unresolved-attention totals for the current board scope.
func (m Model) scopeAttentionSummary(byID map[string]domain.ActionItem) (int, int, int, []string) {
	items := 0
	total := 0
	blocked := 0
	top := make([]string, 0, 3)
	for _, column := range m.columns {
		for _, actionItem := range m.boardActionItemsForColumn(column.ID) {
			count := m.actionItemAttentionCount(actionItem, byID)
			if count <= 0 {
				continue
			}
			items++
			total += count
			if strings.TrimSpace(actionItem.Metadata.BlockedReason) != "" {
				blocked++
			}
			if len(top) < 3 {
				top = append(top, fmt.Sprintf("%s !%d", truncate(actionItem.Title, 24), count))
			}
		}
	}
	return items, total, blocked, top
}

// buildScopeWarnings synthesizes warning text from attention counts.
func buildScopeWarnings(attentionItemsCount, attentionUserActionCount, globalNoticesPartialCount int) []string {
	warnings := make([]string, 0, 3)
	if attentionItemsCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d open attention items in scope", attentionItemsCount))
	}
	if attentionUserActionCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d attention items require user action", attentionUserActionCount))
	}
	if globalNoticesPartialCount > 0 {
		projectLabel := "projects"
		if globalNoticesPartialCount == 1 {
			projectLabel = "project"
		}
		warnings = append(
			warnings,
			fmt.Sprintf(
				"global notices partial: %d %s unavailable",
				globalNoticesPartialCount,
				projectLabel,
			),
		)
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

// actionItemGroupRank returns deterministic ordering rank for configured board grouping.
func actionItemGroupRank(actionItem domain.ActionItem, groupBy string) int {
	switch normalizeBoardGroupBy(groupBy) {
	case "priority":
		switch actionItem.Priority {
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
		switch strings.ToLower(strings.TrimSpace(string(actionItem.LifecycleState))) {
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

// orderActionItemsByHierarchy renders parent items before their descendants.
func orderActionItemsByHierarchy(tasks []domain.ActionItem) []domain.ActionItem {
	if len(tasks) <= 1 {
		return tasks
	}
	childrenByParent := map[string][]domain.ActionItem{}
	byID := map[string]domain.ActionItem{}
	roots := make([]domain.ActionItem, 0)
	for _, actionItem := range tasks {
		byID[actionItem.ID] = actionItem
	}
	for _, actionItem := range tasks {
		parentID := strings.TrimSpace(actionItem.ParentID)
		if parentID == "" {
			roots = append(roots, actionItem)
			continue
		}
		if _, ok := byID[parentID]; !ok {
			roots = append(roots, actionItem)
			continue
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], actionItem)
	}
	sortActionItemSlice(roots)
	for parentID := range childrenByParent {
		children := childrenByParent[parentID]
		sortActionItemSlice(children)
		childrenByParent[parentID] = children
	}
	ordered := make([]domain.ActionItem, 0, len(tasks))
	visited := map[string]struct{}{}
	var visit func(domain.ActionItem)
	visit = func(actionItem domain.ActionItem) {
		if _, ok := visited[actionItem.ID]; ok {
			return
		}
		visited[actionItem.ID] = struct{}{}
		ordered = append(ordered, actionItem)
		for _, child := range childrenByParent[actionItem.ID] {
			visit(child)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	for _, actionItem := range tasks {
		if _, ok := visited[actionItem.ID]; ok {
			continue
		}
		visit(actionItem)
	}
	return ordered
}

// sortActionItemSlice orders tasks by creation time (oldest-first) with deterministic fallbacks.
func sortActionItemSlice(tasks []domain.ActionItem) {
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

// actionItemDepth returns nesting depth for a actionItem id with cycle protection.
func actionItemDepth(actionItemID string, parentByID map[string]string, depth int) int {
	if depth > 32 {
		return depth
	}
	parentID, ok := parentByID[actionItemID]
	if !ok || strings.TrimSpace(parentID) == "" {
		return depth
	}
	if _, exists := parentByID[parentID]; !exists {
		return depth
	}
	return actionItemDepth(parentID, parentByID, depth+1)
}

// selectedActionItemInCurrentColumn returns selected actionItem in current column.
func (m Model) selectedActionItemInCurrentColumn() (domain.ActionItem, bool) {
	tasks := m.currentColumnActionItems()
	if len(tasks) == 0 {
		return domain.ActionItem{}, false
	}
	idx := clamp(m.selectedActionItem, 0, len(tasks)-1)
	return tasks[idx], true
}

// selectedBranchActionItem returns the selected actionItem when it is a branch-level work item.
func (m Model) selectedBranchActionItem() (domain.ActionItem, bool) {
	return m.selectedActionItemAtLevel("branch")
}

// selectedActionItemAtLevel returns the selected actionItem when it matches one hierarchy level.
func (m Model) selectedActionItemAtLevel(level string) (domain.ActionItem, bool) {
	return m.selectedActionItemAtLevels(level)
}

// selectedActionItemAtLevels returns the selected actionItem when it matches one of the provided hierarchy levels.
func (m Model) selectedActionItemAtLevels(levels ...string) (domain.ActionItem, bool) {
	actionItem, ok := m.selectedActionItemInCurrentColumn()
	if !ok {
		return domain.ActionItem{}, false
	}
	if !actionItemMatchesHierarchyLevel(actionItem, levels...) {
		return domain.ActionItem{}, false
	}
	return actionItem, true
}

// focusedScopeActionItemAtLevel returns the active focus root actionItem when it matches one hierarchy level.
func (m Model) focusedScopeActionItemAtLevel(level string) (domain.ActionItem, bool) {
	return m.focusedScopeActionItemAtLevels(level)
}

// focusedScopeActionItemAtLevels returns the active focus root actionItem when it matches one provided hierarchy level.
func (m Model) focusedScopeActionItemAtLevels(levels ...string) (domain.ActionItem, bool) {
	rootID := strings.TrimSpace(m.projectionRootActionItemID)
	if rootID == "" {
		return domain.ActionItem{}, false
	}
	actionItem, ok := m.actionItemByID(rootID)
	if !ok {
		return domain.ActionItem{}, false
	}
	if !actionItemMatchesHierarchyLevel(actionItem, levels...) {
		return domain.ActionItem{}, false
	}
	return actionItem, true
}

// actionItemMatchesHierarchyLevel reports whether a actionItem matches any provided hierarchy levels.
func actionItemMatchesHierarchyLevel(actionItem domain.ActionItem, levels ...string) bool {
	if len(levels) == 0 {
		return false
	}
	level := strings.TrimSpace(strings.ToLower(baseSearchLevelForActionItem(actionItem)))
	for _, candidate := range levels {
		if strings.TrimSpace(strings.ToLower(candidate)) == level {
			return true
		}
	}
	return false
}

// focusActionItemByID focuses one actionItem by id and reports whether it became selected.
func (m *Model) focusActionItemByID(actionItemID string) bool {
	if strings.TrimSpace(actionItemID) == "" {
		return false
	}
	targetColIdx := -1
	for idx, column := range m.columns {
		tasks := m.tasksForColumn(column.ID)
		for actionItemIdx, actionItem := range tasks {
			if actionItem.ID == actionItemID {
				targetColIdx = idx
				m.selectedColumn = idx
				m.selectedActionItem = actionItemIdx
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

// setBoardContextForActionItem adjusts subtree focus so closing a modal lands back on the matched node context.
func (m *Model) setBoardContextForActionItem(actionItem domain.ActionItem) {
	parentID := strings.TrimSpace(actionItem.ParentID)
	if parentID == "" {
		m.projectionRootActionItemID = ""
		return
	}
	m.projectionRootActionItemID = parentID
}

// activateSubtreeFocus enters focused scope mode and selects the first visible child when present.
func (m *Model) activateSubtreeFocus(rootActionItemID string) bool {
	rootActionItemID = strings.TrimSpace(rootActionItemID)
	if rootActionItemID == "" {
		return false
	}
	if _, ok := m.actionItemByID(rootActionItemID); !ok {
		return false
	}
	m.projectionRootActionItemID = rootActionItemID
	m.selectedActionItem = 0
	for idx, column := range m.columns {
		tasks := m.boardActionItemsForColumn(column.ID)
		if len(tasks) == 0 {
			continue
		}
		m.selectedColumn = idx
		m.selectedActionItem = 0
		m.clampSelections()
		return true
	}
	// Empty focused scopes are still valid so users can create the first child in place.
	m.clampSelections()
	return true
}

// clearSubtreeFocus exits focused scope mode and reselects the prior focus root when available.
func (m *Model) clearSubtreeFocus() bool {
	rootID := strings.TrimSpace(m.projectionRootActionItemID)
	if rootID == "" {
		return false
	}
	m.projectionRootActionItemID = ""
	m.focusActionItemByID(rootID)
	m.clampSelections()
	return true
}

// actionItemByID returns actionItem by id.
func (m Model) actionItemByID(actionItemID string) (domain.ActionItem, bool) {
	for _, actionItem := range m.tasks {
		if actionItem.ID == actionItemID {
			return actionItem, true
		}
	}
	return domain.ActionItem{}, false
}

// projectByID resolves one loaded project by stable id.
func (m Model) projectByID(projectID string) (domain.Project, bool) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.Project{}, false
	}
	for _, project := range m.projects {
		if strings.TrimSpace(project.ID) == projectID {
			return project, true
		}
	}
	return domain.Project{}, false
}

// directChildCount returns the number of direct children for one actionItem id.
func (m Model) directChildCount(actionItemID string) int {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return 0
	}
	count := 0
	for _, actionItem := range m.tasks {
		if strings.TrimSpace(actionItem.ParentID) != actionItemID {
			continue
		}
		if !m.showArchived && actionItem.ArchivedAt != nil {
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

// selectedActionItemHighlightColor returns the configured board-selection highlight color.
func (m Model) selectedActionItemHighlightColor() color.Color {
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

// hasGlobalNoticesPanel reports whether the global-notifications panel should be rendered and focusable.
func (m Model) hasGlobalNoticesPanel() bool {
	return true
}

// globalNoticesEmptyRowKey identifies the deterministic placeholder row when no global notices are available.
const globalNoticesEmptyRowKey = "global-notices:empty"

// notificationScopeLevel normalizes attention scope values and defaults empties to project scope.
func notificationScopeLevel(scopeType domain.ScopeLevel) domain.ScopeLevel {
	scopeType = domain.NormalizeScopeLevel(scopeType)
	if scopeType == "" {
		return domain.ScopeLevelProject
	}
	return scopeType
}

// notificationActionItemIDFromScope resolves a actionItem-info target id only for actionItem/subtask scoped rows.
func notificationActionItemIDFromScope(scopeType domain.ScopeLevel, scopeID string) string {
	if scopeID == "" {
		return ""
	}
	switch notificationScopeLevel(scopeType) {
	case domain.ScopeLevelActionItem, domain.ScopeLevelSubtask:
		return scopeID
	default:
		return ""
	}
}

// commentTargetTypeForScopeLevel maps scope levels into comment-target types.
func commentTargetTypeForScopeLevel(scopeType domain.ScopeLevel) (domain.CommentTargetType, bool) {
	switch notificationScopeLevel(scopeType) {
	case domain.ScopeLevelProject:
		return domain.CommentTargetTypeProject, true
	case domain.ScopeLevelBranch:
		return domain.CommentTargetTypeBranch, true
	case domain.ScopeLevelPhase:
		return domain.CommentTargetTypePhase, true
	case domain.ScopeLevelActionItem:
		return domain.CommentTargetTypeActionItem, true
	case domain.ScopeLevelSubtask:
		return domain.CommentTargetTypeSubtask, true
	default:
		return "", false
	}
}

// commentTargetForScope normalizes one comment target from attention scope metadata.
func commentTargetForScope(projectID string, scopeType domain.ScopeLevel, scopeID string) (domain.CommentTarget, bool) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.CommentTarget{}, false
	}
	scopeType = notificationScopeLevel(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if scopeType == domain.ScopeLevelProject && scopeID == "" {
		scopeID = projectID
	}
	targetType, ok := commentTargetTypeForScopeLevel(scopeType)
	if !ok {
		return domain.CommentTarget{}, false
	}
	target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  projectID,
		TargetType: targetType,
		TargetID:   scopeID,
	})
	if err != nil {
		return domain.CommentTarget{}, false
	}
	return target, true
}

// notificationAttentionLabel renders one scoped attention-row label for the project notifications panel.
func notificationAttentionLabel(scopeType domain.ScopeLevel, summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = "attention item"
	}
	return fmt.Sprintf("%s: %s", notificationScopeLevel(scopeType), summary)
}

// notificationThreadTitle renders one deterministic thread title for notification-driven opens.
func notificationThreadTitle(scopeType domain.ScopeLevel, summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = "attention item"
	}
	return fmt.Sprintf("%s attention: %s", notificationScopeLevel(scopeType), summary)
}

// globalNoticesActionItemIDFromAttention resolves one actionItem/subtask scoped id for actionItem-info opens.
func globalNoticesActionItemIDFromAttention(item domain.AttentionItem) string {
	scopeID := strings.TrimSpace(item.ScopeID)
	if scopeID == "" {
		return ""
	}
	scopeType := notificationScopeLevel(item.ScopeType)
	switch scopeType {
	case domain.ScopeLevelActionItem, domain.ScopeLevelSubtask:
		return notificationActionItemIDFromScope(scopeType, scopeID)
	}
	return ""
}

// globalNoticesStableKey returns one deterministic row identity for reload re-anchoring.
func globalNoticesStableKey(projectID, attentionID string, scopeType domain.ScopeLevel, scopeID, summary string) string {
	projectID = strings.TrimSpace(projectID)
	attentionID = strings.TrimSpace(attentionID)
	scopeType = domain.NormalizeScopeLevel(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if attentionID != "" {
		return fmt.Sprintf("project:%s|attention:%s|scope:%s|id:%s", projectID, attentionID, scopeType, scopeID)
	}
	if scopeType != "" || scopeID != "" {
		return fmt.Sprintf("project:%s|scope:%s|id:%s", projectID, scopeType, scopeID)
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = "attention item"
	}
	return fmt.Sprintf("project:%s|summary:%s", projectID, summary)
}

// globalCoordinationStableKey returns one deterministic row identity for a coordination summary row.
func globalCoordinationStableKey(projectID string, global bool) string {
	projectID = strings.TrimSpace(projectID)
	if global {
		return fmt.Sprintf("coordination:global:%s", projectID)
	}
	return fmt.Sprintf("coordination:project:%s", projectID)
}

// globalNoticesPanelItemFromCoordinationSummary maps one project/global coordination summary into a global notices row.
func globalNoticesPanelItemFromCoordinationSummary(projectID, projectLabel string, summary noticesCoordinationSummary, global bool) (globalNoticesPanelItem, bool) {
	parts := coordinationSummaryParts(summary, false)
	if len(parts) == 0 {
		return globalNoticesPanelItem{}, false
	}
	return globalNoticesPanelItem{
		StableKey:          globalCoordinationStableKey(projectID, global),
		ProjectID:          strings.TrimSpace(projectID),
		ProjectLabel:       strings.TrimSpace(projectLabel),
		Summary:            strings.Join(parts, "\n"),
		CoordinationGlobal: global,
	}, true
}

// globalNoticesPanelItemFromAttention maps one attention item into a global notifications panel row.
func globalNoticesPanelItemFromAttention(project domain.Project, item domain.AttentionItem) globalNoticesPanelItem {
	return globalNoticesPanelItemFromAttentionLabel(strings.TrimSpace(project.ID), projectDisplayName(project), item)
}

// globalNoticesPanelItemFromAttentionLabel maps one attention item into a global notifications row with an explicit project label.
func globalNoticesPanelItemFromAttentionLabel(projectID, projectLabel string, item domain.AttentionItem) globalNoticesPanelItem {
	summary := strings.TrimSpace(item.Summary)
	if summary == "" {
		summary = "attention item"
	}
	projectID = strings.TrimSpace(projectID)
	scopeType := notificationScopeLevel(item.ScopeType)
	scopeID := strings.TrimSpace(item.ScopeID)
	if scopeType == domain.ScopeLevelProject && scopeID == "" {
		scopeID = projectID
	}
	attentionID := strings.TrimSpace(item.ID)
	out := globalNoticesPanelItem{
		StableKey:         globalNoticesStableKey(projectID, attentionID, scopeType, scopeID, summary),
		AttentionID:       attentionID,
		AttentionKind:     item.Kind,
		ProjectID:         projectID,
		ProjectLabel:      strings.TrimSpace(projectLabel),
		ScopeType:         scopeType,
		ScopeID:           scopeID,
		Summary:           summary,
		ActionItemID:      globalNoticesActionItemIDFromAttention(item),
		ThreadDescription: strings.TrimSpace(item.BodyMarkdown),
	}
	if domain.NormalizeAttentionKind(item.Kind) == domain.AttentionKindHandoff {
		out.HandoffID = handoffIDFromAttentionID(attentionID)
	}
	return out
}

// handoffIDFromAttentionID strips the mirrored handoff attention suffix back to the source handoff id.
func handoffIDFromAttentionID(attentionID string) string {
	attentionID = strings.TrimSpace(attentionID)
	return strings.TrimSuffix(attentionID, "::handoff")
}

// normalizeNoticeInboxRole canonicalizes viewer-facing inbox roles and common aliases.
func normalizeNoticeInboxRole(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "dev":
		return "builder"
	case "researcher":
		return "research"
	default:
		return strings.TrimSpace(strings.ToLower(raw))
	}
}

// isNoticeInboxRole reports whether one role is routable through the comment inbox view.
func isNoticeInboxRole(role string) bool {
	switch normalizeNoticeInboxRole(role) {
	case "builder", "qa", "orchestrator", "research", "human":
		return true
	default:
		return false
	}
}

// viewerInboxRoles returns the routable inbox roles that map to the current local viewer identity.
func (m Model) viewerInboxRoles() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 2)
	add := func(raw string) {
		role := normalizeNoticeInboxRole(raw)
		if !isNoticeInboxRole(role) {
			return
		}
		if _, ok := seen[role]; ok {
			return
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	if m.threadActorType() == domain.ActorTypeUser {
		add("human")
	}
	add(m.identityActorID)
	add(m.identityDisplayName)
	return out
}

// isViewerCommentAttention reports whether one attention row belongs in the viewer-scoped comments inbox.
func (m Model) isViewerCommentAttention(item domain.AttentionItem) bool {
	if !item.IsUnresolved() || domain.NormalizeAttentionKind(item.Kind) != domain.AttentionKindMention {
		return false
	}
	targetRole := normalizeNoticeInboxRole(item.TargetRole)
	if targetRole == "" {
		return false
	}
	return slices.Contains(m.viewerInboxRoles(), targetRole)
}

// isViewerHandoffAttention reports whether one handoff row is explicitly routed to the current viewer identity.
func (m Model) isViewerHandoffAttention(item domain.AttentionItem) bool {
	if !item.IsUnresolved() || domain.NormalizeAttentionKind(item.Kind) != domain.AttentionKindHandoff {
		return false
	}
	targetRole := normalizeNoticeInboxRole(item.TargetRole)
	if targetRole == "" {
		return false
	}
	return slices.Contains(m.viewerInboxRoles(), targetRole)
}

// shouldShowWarningAttention reports whether one attention row belongs in the generic warnings section.
func (m Model) shouldShowWarningAttention(item domain.AttentionItem) bool {
	if !item.IsUnresolved() {
		return false
	}
	switch domain.NormalizeAttentionKind(item.Kind) {
	case domain.AttentionKindMention:
		return false
	case domain.AttentionKindHandoff:
		return !m.isViewerHandoffAttention(item)
	}
	return !item.RequiresUserAction
}

// shouldShowActionRequiredAttention reports whether one attention row belongs in the action-required section.
func (m Model) shouldShowActionRequiredAttention(item domain.AttentionItem) bool {
	if !item.IsUnresolved() {
		return false
	}
	if m.isViewerCommentAttention(item) {
		return false
	}
	if domain.NormalizeAttentionKind(item.Kind) == domain.AttentionKindHandoff {
		return m.isViewerHandoffAttention(item)
	}
	return item.RequiresUserAction
}

// shouldShowGlobalNoticeAttention reports whether one attention row should surface in the global notifications list.
func (m Model) shouldShowGlobalNoticeAttention(item domain.AttentionItem) bool {
	return m.shouldShowActionRequiredAttention(item) || m.isViewerCommentAttention(item) || m.shouldShowWarningAttention(item)
}

// globalNoticesPanelItemsForInteraction returns selectable global-notifications rows.
func (m Model) globalNoticesPanelItemsForInteraction() []globalNoticesPanelItem {
	if len(m.globalNotices) == 0 {
		return []globalNoticesPanelItem{{
			StableKey: globalNoticesEmptyRowKey,
			Summary:   "no coordination or notifications across other projects",
		}}
	}
	return append([]globalNoticesPanelItem(nil), m.globalNotices...)
}

// reanchorGlobalNoticesSelection keeps focus on one stable row key after global-notices reloads.
func (m *Model) reanchorGlobalNoticesSelection(previousKey string) {
	items := m.globalNoticesPanelItemsForInteraction()
	if len(items) == 0 {
		m.globalNoticesIdx = 0
		return
	}
	previousKey = strings.TrimSpace(previousKey)
	if previousKey != "" {
		for idx, item := range items {
			if strings.TrimSpace(item.StableKey) == previousKey {
				m.globalNoticesIdx = idx
				return
			}
		}
	}
	m.globalNoticesIdx = clamp(m.globalNoticesIdx, 0, len(items)-1)
}

// clampGlobalNoticesSelection keeps the global-notifications row cursor within bounds.
func (m *Model) clampGlobalNoticesSelection() {
	m.reanchorGlobalNoticesSelection("")
}

// selectedGlobalNoticesItem returns the currently selected global-notifications row.
func (m Model) selectedGlobalNoticesItem() (globalNoticesPanelItem, bool) {
	items := m.globalNoticesPanelItemsForInteraction()
	if len(items) == 0 {
		return globalNoticesPanelItem{}, false
	}
	idx := clamp(m.globalNoticesIdx, 0, len(items)-1)
	return items[idx], true
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

// coordinationSummaryParts renders one project/global summary with count-by-type parts.
func coordinationSummaryParts(summary noticesCoordinationSummary, includeZero bool) []string {
	counts := []struct {
		label string
		value int
	}{
		{label: "pending requests", value: len(summary.PendingRequests)},
		{label: "active sessions", value: len(summary.ActiveSessions)},
		{label: "active leases", value: len(summary.ActiveLeases)},
		{label: "open handoffs", value: len(summary.OpenHandoffs)},
	}
	parts := make([]string, 0, len(counts))
	for _, count := range counts {
		if !includeZero && count.value == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %d", count.label, count.value))
	}
	return parts
}

// noticesCoordinationPanelItems builds selectable coordination rows for the current project notices panel.
func (m Model) noticesCoordinationPanelItems() []noticesPanelItem {
	projectID, _ := m.currentProjectID()
	summary := m.noticesCoordination
	rows := coordinationSummaryParts(summary, true)
	items := make([]noticesPanelItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, noticesPanelItem{
			Label:                 row,
			CoordinationProjectID: projectID,
		})
	}
	return items
}

// noticesSectionTitle returns a stable header label for one notices section identifier.
func noticesSectionTitle(section noticesSectionID) string {
	switch section {
	case noticesSectionCoordination:
		return "Coordination"
	case noticesSectionWarnings:
		return "Warnings"
	case noticesSectionComments:
		return "Comments"
	case noticesSectionAttention:
		return "Action Required"
	case noticesSectionRecentActivity:
		return "Recent Activity"
	default:
		return "Notices"
	}
}

// noticesSelectionIndex returns the current selected row index for one notices section.
func (m Model) noticesSelectionIndex(section noticesSectionID) int {
	switch section {
	case noticesSectionCoordination:
		return m.noticesCoordinationIdx
	case noticesSectionWarnings:
		return m.noticesWarnings
	case noticesSectionComments:
		return m.noticesComments
	case noticesSectionAttention:
		return m.noticesAttention
	case noticesSectionRecentActivity:
		return m.noticesActivity
	default:
		return 0
	}
}

// setNoticesSelectionIndex stores one selected row index for the target notices section.
func (m *Model) setNoticesSelectionIndex(section noticesSectionID, idx int) {
	switch section {
	case noticesSectionCoordination:
		m.noticesCoordinationIdx = idx
	case noticesSectionWarnings:
		m.noticesWarnings = idx
	case noticesSectionComments:
		m.noticesComments = idx
	case noticesSectionAttention:
		m.noticesAttention = idx
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

// noticesSectionIndex resolves one section id to its concrete position in one rendered notices slice.
func noticesSectionIndex(sections []noticesPanelSection, section noticesSectionID) int {
	for idx, candidate := range sections {
		if candidate.ID == section {
			return idx
		}
	}
	return -1
}

// noticesPanelItemFromAttention maps one unresolved attention record into an actionable notices row.
func (m Model) noticesPanelItemFromAttention(item domain.AttentionItem) (noticesPanelItem, bool) {
	projectID, _ := m.currentProjectID()
	scopeType := notificationScopeLevel(item.ScopeType)
	scopeID := strings.TrimSpace(item.ScopeID)
	if scopeType == domain.ScopeLevelProject && scopeID == "" {
		scopeID = projectID
	}
	label := notificationAttentionLabel(scopeType, item.Summary)
	if label == "" {
		return noticesPanelItem{}, false
	}
	rowProjectID := strings.TrimSpace(item.ProjectID)
	if rowProjectID == "" {
		rowProjectID = projectID
	}
	return noticesPanelItem{
		Label:         label,
		AttentionID:   strings.TrimSpace(item.ID),
		AttentionKind: item.Kind,
		HandoffID: func() string {
			if domain.NormalizeAttentionKind(item.Kind) == domain.AttentionKindHandoff {
				return handoffIDFromAttentionID(item.ID)
			}
			return ""
		}(),
		ActionItemID:      notificationActionItemIDFromScope(scopeType, scopeID),
		ProjectID:         rowProjectID,
		ScopeType:         scopeType,
		ScopeID:           scopeID,
		ThreadTitle:       notificationThreadTitle(scopeType, item.Summary),
		ThreadDescription: strings.TrimSpace(item.BodyMarkdown),
		CoordinationProjectID: func() string {
			if domain.NormalizeAttentionKind(item.Kind) == domain.AttentionKindHandoff {
				return rowProjectID
			}
			return ""
		}(),
	}, true
}

// noticesWarningPanelItems builds actionable warning rows from unresolved attention records.
func (m Model) noticesWarningPanelItems() []noticesPanelItem {
	out := make([]noticesPanelItem, 0, len(m.attentionItems))
	for _, item := range m.attentionItems {
		if !m.shouldShowWarningAttention(item) {
			continue
		}
		row, ok := m.noticesPanelItemFromAttention(item)
		if !ok {
			continue
		}
		out = append(out, row)
	}
	overdue, dueSoon := m.dueCounts(time.Now().UTC())
	if overdue > 0 {
		out = append(out, noticesPanelItem{Label: fmt.Sprintf("overdue: %d", overdue)})
	}
	if dueSoon > 0 {
		out = append(out, noticesPanelItem{Label: fmt.Sprintf("due soon: %d", dueSoon)})
	}
	return out
}

// noticesCommentsPanelItems builds viewer-scoped comment inbox rows from routed mention attention.
func (m Model) noticesCommentsPanelItems() []noticesPanelItem {
	out := make([]noticesPanelItem, 0, len(m.attentionItems))
	for _, item := range m.attentionItems {
		if !m.isViewerCommentAttention(item) {
			continue
		}
		row, ok := m.noticesPanelItemFromAttention(item)
		if !ok {
			continue
		}
		out = append(out, row)
	}
	return out
}

// noticesAttentionPanelItems builds selectable action-required rows from persisted attention records.
func (m Model) noticesAttentionPanelItems() []noticesPanelItem {
	out := make([]noticesPanelItem, 0, len(m.attentionItems))
	for _, item := range m.attentionItems {
		if !m.shouldShowActionRequiredAttention(item) {
			continue
		}
		row, ok := m.noticesPanelItemFromAttention(item)
		if !ok {
			continue
		}
		out = append(out, row)
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
	_ = attentionTop

	attentionRows := m.noticesAttentionPanelItems()
	actionableAttentionCount := len(attentionRows)
	if actionableAttentionCount == 0 {
		attentionRows = append(attentionRows, noticesPanelItem{Label: "none"})
	}
	attentionSummary := []string{}
	if actionableAttentionCount > 0 {
		attentionSummary = append(attentionSummary, fmt.Sprintf("requires action: %d", actionableAttentionCount))
	}
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

	warningItems := m.noticesWarningPanelItems()
	if len(warningItems) == 0 {
		warningItems = append(warningItems, noticesPanelItem{Label: "none"})
	}
	sections = append(sections, noticesPanelSection{
		ID:      noticesSectionWarnings,
		Title:   noticesSectionTitle(noticesSectionWarnings),
		Summary: append([]string(nil), m.warnings...),
		Items:   warningItems,
	})

	commentRows := m.noticesCommentsPanelItems()
	if len(commentRows) > 0 {
		commentSummary := []string{fmt.Sprintf("mentions for you: %d", len(commentRows))}
		sections = append(sections, noticesPanelSection{
			ID:      noticesSectionComments,
			Title:   noticesSectionTitle(noticesSectionComments),
			Summary: commentSummary,
			Items:   commentRows,
		})
	}

	coordinationRows := m.noticesCoordinationPanelItems()
	sections = append(sections, noticesPanelSection{
		ID:      noticesSectionCoordination,
		Title:   noticesSectionTitle(noticesSectionCoordination),
		Summary: nil,
		Items:   coordinationRows,
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
	actionItemByID := m.tasksByID()
	attentionItems, attentionTotal, attentionBlocked, attentionTop := m.scopeAttentionSummary(actionItemByID)
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
	if noticesSectionIndex(sections, m.noticesSection) >= 0 {
		return
	}
	if idx := noticesSectionIndex(sections, noticesSectionRecentActivity); idx >= 0 {
		m.noticesSection = sections[idx].ID
		return
	}
	if len(sections) > 0 {
		m.noticesSection = sections[0].ID
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

// selectedNoticesPanelItem returns the currently selected notices-panel row.
func (m Model) selectedNoticesPanelItem() (noticesPanelItem, bool) {
	sections := m.noticesSectionsForInteraction()
	if len(sections) == 0 {
		return noticesPanelItem{}, false
	}
	m.clampNoticesSelectionForSections(sections)
	sectionPos := noticesSectionIndex(sections, m.noticesSection)
	if sectionPos < 0 {
		sectionPos = noticesSectionIndex(sections, noticesSectionRecentActivity)
		if sectionPos < 0 {
			sectionPos = 0
		}
	}
	if sectionPos >= len(sections) {
		return noticesPanelItem{}, false
	}
	section := sections[sectionPos]
	if len(section.Items) == 0 {
		return noticesPanelItem{}, false
	}
	idx := clamp(m.noticesSelectionIndex(section.ID), 0, len(section.Items)-1)
	return section.Items[idx], true
}

// selectedAuthRequestForActiveNotice resolves the current notice row to one pending auth request when present.
func (m Model) selectedAuthRequestForActiveNotice() (domain.AuthRequest, bool) {
	var attentionID string
	if m.noticesFocused && m.noticesPanel == noticesPanelFocusGlobal {
		item, ok := m.selectedGlobalNoticesItem()
		if !ok {
			return domain.AuthRequest{}, false
		}
		attentionID = strings.TrimSpace(item.AttentionID)
	} else {
		item, ok := m.selectedNoticesPanelItem()
		if !ok {
			return domain.AuthRequest{}, false
		}
		attentionID = strings.TrimSpace(item.AttentionID)
	}
	if attentionID == "" || m.svc == nil {
		return domain.AuthRequest{}, false
	}
	req, err := m.svc.GetAuthRequest(context.Background(), app.AuthRequestIDFromAttentionID(attentionID))
	if err != nil {
		return domain.AuthRequest{}, false
	}
	return req, true
}

// authRequestResolutionNote builds a deterministic audit-friendly note for one auth-request decision.
func authRequestResolutionNote(req domain.AuthRequest, decision string) string {
	return authRequestResolutionNoteWithPathLabel(req, decision, req.Path)
}

// authRequestResolutionNoteWithPathLabel builds one deterministic audit-friendly note with one user-facing scope label.
func authRequestResolutionNoteWithPathLabel(req domain.AuthRequest, decision, pathLabel string) string {
	decision = strings.TrimSpace(strings.ToLower(decision))
	action := "resolved"
	switch decision {
	case "approve":
		action = "approved"
	case "deny":
		action = "denied"
	}
	principal := firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID)
	pathLabel = firstNonEmptyTrimmed(pathLabel, req.Path)
	return fmt.Sprintf("%s in Tillsyn for %s at %s", action, principal, pathLabel)
}

// authRequestNoteLooksDefault reports whether one note still matches the built-in auth review copy.
func authRequestNoteLooksDefault(note string) bool {
	note = strings.ToLower(strings.TrimSpace(note))
	return strings.HasPrefix(note, "approved in tillsyn for ") ||
		strings.HasPrefix(note, "denied in tillsyn for ") ||
		strings.HasPrefix(note, "resolved in tillsyn for ")
}

// humanActorLabel renders one actor name with its type when the type is known.
func humanActorLabel(actorID string, actorType domain.ActorType) string {
	actorID = strings.TrimSpace(actorID)
	actorType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
	switch {
	case actorID != "" && actorType != "":
		return fmt.Sprintf("%s (%s)", actorID, actorType)
	case actorID != "":
		return actorID
	case actorType != "":
		return string(actorType)
	default:
		return ""
	}
}

// formatAuthRequestTimeout renders one auth-request timeout duration in compact human-friendly text.
func formatAuthRequestTimeout(req domain.AuthRequest) string {
	if req.CreatedAt.IsZero() || req.ExpiresAt.IsZero() || !req.ExpiresAt.After(req.CreatedAt) {
		return ""
	}
	timeout := req.ExpiresAt.Sub(req.CreatedAt)
	switch {
	case timeout%time.Hour == 0:
		return fmt.Sprintf("%dh", int(timeout/time.Hour))
	case timeout%time.Minute == 0:
		return fmt.Sprintf("%dm", int(timeout/time.Minute))
	default:
		return timeout.Round(time.Second).String()
	}
}

// authReviewRequestContextLines renders the shared auth-review context shown on summary and editor stages.
func (m Model) authReviewRequestContextLines(contentWidth int) []string {
	confirm := m.pendingConfirm
	requestedScope := firstNonEmptyTrimmed(
		confirm.AuthRequestRequestedPathLabel,
		confirm.AuthRequestRequestedPath,
		confirm.AuthRequestPathLabel,
		confirm.AuthRequestPath,
	)
	requestedRawPath := firstNonEmptyTrimmed(confirm.AuthRequestRequestedPath, confirm.AuthRequestPath)
	lines := []string{
		fmt.Sprintf("principal: %s", firstNonEmptyTrimmed(confirm.AuthRequestPrincipal, confirm.AuthRequestID)),
	}
	if role := strings.TrimSpace(confirm.AuthRequestPrincipalRole); role != "" {
		lines = append(lines, fmt.Sprintf("role: %s", role))
	}
	if requester := strings.TrimSpace(confirm.AuthRequestRequestedBy); requester != "" {
		lines = append(lines, fmt.Sprintf("requested by: %s", requester))
	}
	if client := firstNonEmptyTrimmed(confirm.AuthRequestClient, "-"); client != "" {
		lines = append(lines, fmt.Sprintf("client: %s", client))
	}
	if reason := strings.TrimSpace(confirm.AuthRequestReason); reason != "" {
		lines = append(lines, fmt.Sprintf("reason: %s", truncate(reason, max(24, contentWidth-18))))
	}
	if resumeClient := strings.TrimSpace(confirm.AuthRequestResumeClient); resumeClient != "" {
		lines = append(lines, fmt.Sprintf("resume client: %s", resumeClient))
	}
	if timeout := strings.TrimSpace(confirm.AuthRequestTimeout); timeout != "" {
		lines = append(lines, fmt.Sprintf("request timeout: %s", timeout))
	}
	lines = append(lines,
		fmt.Sprintf("requested scope: %s", firstNonEmptyTrimmed(requestedScope, "-")),
		fmt.Sprintf("requested raw path: %s", firstNonEmptyTrimmed(requestedRawPath, "-")),
	)
	if requestedTTL := strings.TrimSpace(confirm.AuthRequestRequestedTTL); requestedTTL != "" {
		lines = append(lines, fmt.Sprintf("requested session ttl: %s", requestedTTL))
	}
	return lines
}

// firstNonEmptyTrimmed returns the first non-empty trimmed string in order.
func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// authRequestPathDisplay converts one canonical auth-request path into a user-facing hierarchy label.
func (m Model) authRequestPathDisplay(rawPath string) string {
	path, err := domain.ParseAuthRequestPath(rawPath)
	if err != nil {
		return strings.TrimSpace(rawPath)
	}
	switch path.Kind {
	case domain.AuthRequestPathKindGlobal:
		return "All Projects"
	case domain.AuthRequestPathKindProjects:
		segments := make([]string, 0, len(path.ProjectIDs))
		for _, projectID := range path.ProjectIDs {
			if project, ok := m.projectByID(projectID); ok {
				segments = append(segments, firstNonEmptyTrimmed(projectDisplayName(project), projectID))
				continue
			}
			segments = append(segments, projectID)
		}
		return strings.Join(segments, ", ")
	}
	segments := make([]string, 0, 1+len(path.PhaseIDs)+1)
	if project, ok := m.projectByID(path.ProjectID); ok {
		segments = append(segments, firstNonEmptyTrimmed(projectDisplayName(project), path.ProjectID))
	} else {
		segments = append(segments, path.ProjectID)
	}
	if path.BranchID != "" {
		segments = append(segments, m.authRequestScopeSegmentDisplay("branch", path.BranchID))
	}
	for _, phaseID := range path.PhaseIDs {
		segments = append(segments, m.authRequestScopeSegmentDisplay("phase", phaseID))
	}
	return strings.Join(segments, " -> ")
}

// authRequestScopeSegmentDisplay resolves one branch/phase id to a user-facing label with stable fallback text.
func (m Model) authRequestScopeSegmentDisplay(scopeKind, scopeID string) string {
	scopeID = strings.TrimSpace(scopeID)
	if scopeID == "" {
		return ""
	}
	if actionItem, ok := m.actionItemByID(scopeID); ok {
		return firstNonEmptyTrimmed(strings.TrimSpace(actionItem.Title), scopeID)
	}
	scopeKind = strings.TrimSpace(scopeKind)
	if scopeKind == "" {
		return scopeID
	}
	return scopeKind + ":" + scopeID
}

// projectBranchActionItems returns deterministic branch candidates for one project id.
func (m Model) projectBranchActionItems(projectID string) []domain.ActionItem {
	out := make([]domain.ActionItem, 0)
	projectID = strings.TrimSpace(projectID)
	for _, actionItem := range m.tasks {
		if strings.TrimSpace(actionItem.ProjectID) != projectID {
			continue
		}
		if actionItem.Scope != domain.KindAppliesToBranch && strings.ToLower(strings.TrimSpace(string(actionItem.Kind))) != "branch" {
			continue
		}
		out = append(out, actionItem)
	}
	sortActionItemSlice(out)
	return out
}

// phaseChildrenForParent returns direct phase children for one branch/phase parent actionItem.
func (m Model) phaseChildrenForParent(parentID string) []domain.ActionItem {
	out := make([]domain.ActionItem, 0)
	parentID = strings.TrimSpace(parentID)
	for _, actionItem := range m.tasks {
		if strings.TrimSpace(actionItem.ParentID) != parentID {
			continue
		}
		if actionItem.Scope != domain.KindAppliesToPhase && strings.ToLower(strings.TrimSpace(string(actionItem.Kind))) != "phase" {
			continue
		}
		out = append(out, actionItem)
	}
	sortActionItemSlice(out)
	return out
}

// buildAuthScopePickerItems resolves all project-rooted auth scopes visible in the TUI for one request.
func (m Model) buildAuthScopePickerItems(rawPath string) []authScopePickerItem {
	path, err := domain.ParseAuthRequestPath(rawPath)
	if err != nil {
		return nil
	}
	items := make([]authScopePickerItem, 0, 8)
	seen := map[string]struct{}{}
	addItem := func(pathValue, label, description string) {
		pathValue = strings.TrimSpace(pathValue)
		if pathValue == "" {
			return
		}
		if _, ok := seen[pathValue]; ok {
			return
		}
		seen[pathValue] = struct{}{}
		items = append(items, authScopePickerItem{
			Path:        pathValue,
			Label:       strings.TrimSpace(label),
			Description: strings.TrimSpace(description),
		})
	}

	var visitPhases func(branch domain.ActionItem, phasePrefix []string, parentID string)
	visitPhases = func(branch domain.ActionItem, phasePrefix []string, parentID string) {
		for _, phase := range m.phaseChildrenForParent(parentID) {
			nextPrefix := append(append([]string(nil), phasePrefix...), phase.ID)
			scopePath := domain.AuthRequestPath{
				ProjectID: path.ProjectID,
				BranchID:  branch.ID,
				PhaseIDs:  nextPrefix,
			}.String()
			addItem(scopePath, m.authRequestPathDisplay(scopePath), "phase scope")
			visitPhases(branch, nextPrefix, phase.ID)
		}
	}

	addProjectScopes := func(projectID string) {
		projectPath := domain.AuthRequestPath{ProjectID: projectID}.String()
		addItem(projectPath, m.authRequestPathDisplay(projectPath), "project scope")
		for _, branch := range m.projectBranchActionItems(projectID) {
			scopePath := domain.AuthRequestPath{
				ProjectID: projectID,
				BranchID:  branch.ID,
			}.String()
			addItem(scopePath, m.authRequestPathDisplay(scopePath), "branch scope")
			visitPhases(branch, nil, branch.ID)
		}
	}

	switch path.Kind {
	case domain.AuthRequestPathKindGlobal:
		addItem(path.String(), m.authRequestPathDisplay(path.String()), "general orchestration scope")
		for _, project := range m.projects {
			addProjectScopes(project.ID)
		}
		return items
	case domain.AuthRequestPathKindProjects:
		addItem(path.String(), m.authRequestPathDisplay(path.String()), "multi-project orchestration scope")
		for _, projectID := range path.ProjectIDs {
			addProjectScopes(projectID)
		}
		return items
	default:
		addProjectScopes(path.ProjectID)
	}
	if currentLabel := m.authRequestPathDisplay(rawPath); strings.TrimSpace(currentLabel) != "" {
		addItem(rawPath, currentLabel, "current requested scope")
	}
	return items
}

// openAuthScopePicker opens the dedicated scope picker for the current auth review.
func (m *Model) openAuthScopePicker() {
	if m == nil {
		return
	}
	items := m.buildAuthScopePickerItems(m.pendingConfirm.AuthRequestPath)
	m.authReviewScopePickerItems = items
	m.authReviewScopePickerIndex = 0
	currentPath := strings.TrimSpace(m.pendingConfirm.AuthRequestPath)
	for idx, item := range items {
		if strings.TrimSpace(item.Path) == currentPath {
			m.authReviewScopePickerIndex = idx
			break
		}
	}
	m.authReviewReturnStage = m.authReviewStage
	m.authReviewResetInputFocus()
	m.mode = modeAuthScopePicker
}

// applySelectedAuthScopePickerItem stores the chosen scope path and refreshes the user-facing approval note when needed.
func (m *Model) applySelectedAuthScopePickerItem() {
	if m == nil || len(m.authReviewScopePickerItems) == 0 {
		return
	}
	idx := clamp(m.authReviewScopePickerIndex, 0, len(m.authReviewScopePickerItems)-1)
	item := m.authReviewScopePickerItems[idx]
	m.pendingConfirm.AuthRequestPath = strings.TrimSpace(item.Path)
	m.pendingConfirm.AuthRequestPathLabel = firstNonEmptyTrimmed(item.Label, item.Path)
	m.confirmAuthPathInput.SetValue(m.pendingConfirm.AuthRequestPath)
	m.confirmAuthPathInput.CursorEnd()
	m.pendingConfirm.AuthRequestNote = strings.TrimSpace(m.confirmAuthNoteInput.Value())
}

// beginAuthRequestDecision opens the dedicated auth-review surface for one concrete auth request.
func (m Model) beginAuthRequestDecision(req domain.AuthRequest, decision string, returnMode inputMode) (tea.Model, tea.Cmd, bool) {
	if strings.TrimSpace(req.ID) == "" {
		return m, nil, false
	}
	decision = strings.TrimSpace(strings.ToLower(decision))
	if decision != "approve" && decision != "deny" {
		return m, nil, false
	}
	principal := firstNonEmptyTrimmed(req.PrincipalName, req.PrincipalID)
	client := firstNonEmptyTrimmed(req.ClientName, req.ClientID)
	pathLabel := firstNonEmptyTrimmed(m.authRequestPathDisplay(req.Path), req.Path)
	m.mode = modeAuthReview
	requestedBy := humanActorLabel(req.RequestedByActor, req.RequestedByType)
	m.pendingConfirm = confirmAction{
		Kind:                          decision + "-auth-request",
		Label:                         decision + " auth request",
		AuthRequestID:                 req.ID,
		AuthRequestAttention:          req.ID,
		AuthRequestPrincipal:          principal,
		AuthRequestPrincipalRole:      strings.TrimSpace(req.PrincipalRole),
		AuthRequestClient:             client,
		AuthRequestReason:             strings.TrimSpace(req.Reason),
		AuthRequestRequestedBy:        requestedBy,
		AuthRequestResumeClient:       app.AuthRequestClaimClientIDFromContinuation(req.Continuation, req.ClientID),
		AuthRequestTimeout:            formatAuthRequestTimeout(req),
		AuthRequestRequestedPath:      req.Path,
		AuthRequestRequestedPathLabel: pathLabel,
		AuthRequestRequestedTTL:       req.RequestedSessionTTL.String(),
		AuthRequestPath:               req.Path,
		AuthRequestPathLabel:          pathLabel,
		AuthRequestTTL:                req.RequestedSessionTTL.String(),
		AuthRequestDecision:           decision,
		AuthRequestNote:               "",
		ReturnToAuthAccess:            returnMode == modeAuthInventory,
	}
	m.status = "auth review"
	m.authReviewStage = authReviewStageSummary
	m.authReviewScopePickerItems = nil
	m.authReviewScopePickerIndex = 0
	m.authReviewReturnStage = authReviewStageSummary
	m.authReviewReturnMode = returnMode
	m.authReviewResetInputFocus()
	m.confirmAuthTTLInput.SetValue(req.RequestedSessionTTL.String())
	m.confirmAuthTTLInput.CursorEnd()
	m.confirmAuthPathInput.SetValue(req.Path)
	m.confirmAuthPathInput.CursorEnd()
	m.confirmAuthNoteInput.SetValue(m.pendingConfirm.AuthRequestNote)
	m.confirmAuthNoteInput.CursorEnd()
	if decision == "deny" {
		_ = m.setPendingAuthRequestDecision("deny")
		return m, m.authReviewOpenNoteStage(authReviewStageDeny), true
	}
	_ = m.setPendingAuthRequestDecision("approve")
	return m, nil, true
}

// beginSelectedAuthRequestDecision opens the dedicated auth-review surface for the currently selected notice row.
func (m Model) beginSelectedAuthRequestDecision(decision string) (tea.Model, tea.Cmd, bool) {
	req, ok := m.selectedAuthRequestForActiveNotice()
	if !ok {
		return m, nil, false
	}
	return m.beginAuthRequestDecision(req, decision, modeNone)
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
	sectionPos := noticesSectionIndex(sections, m.noticesSection)
	if sectionPos < 0 {
		sectionPos = noticesSectionIndex(sections, noticesSectionRecentActivity)
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
		next := sections[0]
		m.noticesSection = next.ID
		m.setNoticesSelectionIndex(next.ID, 0)
		return true
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
	prev := sections[len(sections)-1]
	m.noticesSection = prev.ID
	m.setNoticesSelectionIndex(prev.ID, max(0, len(prev.Items)-1))
	return true
}

// moveGlobalNoticesSelection moves row focus inside the global notifications panel.
func (m *Model) moveGlobalNoticesSelection(delta int) bool {
	if delta == 0 {
		return false
	}
	items := m.globalNoticesPanelItemsForInteraction()
	if len(items) == 0 {
		return false
	}
	next := wrapIndex(m.globalNoticesIdx, delta, len(items))
	m.globalNoticesIdx = next
	return len(items) > 1
}

// activateGlobalNoticesSelection runs enter behavior for the selected global notifications row.
func (m Model) activateGlobalNoticesSelection() (tea.Model, tea.Cmd) {
	items := m.globalNoticesPanelItemsForInteraction()
	if len(items) == 0 {
		m.traceGlobalNoticeBranch("no_items")
		m.status = "no global notifications available"
		m.completeGlobalNoticeTransition("no_items")
		return m, nil
	}
	m.clampGlobalNoticesSelection()
	item := items[clamp(m.globalNoticesIdx, 0, len(items)-1)]
	if strings.TrimSpace(item.ProjectID) != "" && (strings.HasPrefix(strings.TrimSpace(item.StableKey), "coordination:") || item.CoordinationGlobal || strings.TrimSpace(item.HandoffID) != "") {
		cmd := m.openCoordinationFromNotice(item.ProjectID, item.CoordinationGlobal, item.HandoffID)
		return m, cmd
	}
	if req, ok := m.selectedAuthRequestForActiveNotice(); ok && strings.TrimSpace(req.ID) != "" {
		m.traceGlobalNoticeBranch("auth_request_open_review", "request_id", strings.TrimSpace(req.ID))
		m.completeGlobalNoticeTransition("auth_request_review")
		next, cmd, _ := m.beginSelectedAuthRequestDecision("approve")
		return next, cmd
	}
	m.traceGlobalNoticeBranch(
		"selection",
		"stable_key", strings.TrimSpace(item.StableKey),
		"project_id", strings.TrimSpace(item.ProjectID),
		"scope_type", strings.TrimSpace(string(item.ScopeType)),
		"scope_id", strings.TrimSpace(item.ScopeID),
		"action_item_id", strings.TrimSpace(item.ActionItemID),
	)
	if strings.TrimSpace(item.StableKey) == globalNoticesEmptyRowKey {
		m.traceGlobalNoticeBranch("empty_row")
		m.status = "no global notifications available"
		m.completeGlobalNoticeTransition("empty_row")
		return m, nil
	}
	projectID := strings.TrimSpace(item.ProjectID)
	currentProjectID, hasProject := m.currentProjectID()
	if projectID == "" && hasProject {
		projectID = currentProjectID
	}
	if projectID == "" {
		m.traceGlobalNoticeBranch("missing_project_context")
		m.status = "selected global notification has no project context"
		m.completeGlobalNoticeTransition("missing_project_context")
		return m, nil
	}
	threadTarget, hasThreadTarget := commentTargetForScope(projectID, item.ScopeType, item.ScopeID)
	threadTitle := notificationThreadTitle(item.ScopeType, item.Summary)
	threadBody := strings.TrimSpace(item.ThreadDescription)
	switchProject := !hasProject || projectID != currentProjectID
	actionItemID := strings.TrimSpace(item.ActionItemID)
	if actionItemID == "" {
		if !hasThreadTarget {
			m.traceGlobalNoticeBranch("no_task_no_thread_target", "switch_project", switchProject)
			m.status = "selected global notification has no comment thread target"
			m.completeGlobalNoticeTransition("no_task_no_thread_target")
			return m, nil
		}
		// For project-scoped/global notices we can open the thread directly without
		// a project-switch reload, which avoids UI stalls and preserves deterministic Enter behavior.
		m.traceGlobalNoticeBranch("no_task_open_thread_direct", "switch_project", switchProject)
		m.completeGlobalNoticeTransition("open_thread_no_task")
		return m.startNotificationThread(threadTarget, threadTitle, threadBody)
	}

	if switchProject {
		m.traceGlobalNoticeBranch("task_switch_project_reload")
		m.searchApplied = false
		m.searchQuery = ""
		m.pendingProjectID = projectID
		m.traceGlobalNoticePending("set", "pending_project_id", projectID, "reason", "switch_project_task")
		m.pendingFocusActionItemID = actionItemID
		m.traceGlobalNoticePending("set", "pending_focus_action_item_id", actionItemID, "reason", "switch_project_task")
		m.pendingOpenActionItemInfoID = actionItemID
		m.traceGlobalNoticePending("set", "pending_open_task_info_id", actionItemID, "reason", "switch_project_task")
		if hasThreadTarget {
			m.setPendingNotificationThread(threadTarget, threadTitle, threadBody)
		} else {
			m.clearPendingNotificationThread()
		}
		m.status = "loading global notification..."
		return m, m.loadData
	}

	if m.openActionItemInfo(actionItemID, "actionItem info") {
		m.traceGlobalNoticeBranch("task_open_task_info")
		m.noticesFocused = false
		m.clearPendingNotificationThread()
		m.completeGlobalNoticeTransition("open_task_info")
		return m, nil
	}
	if m.searchApplied || m.searchQuery != "" {
		m.traceGlobalNoticeBranch("task_reload_after_search_reset")
		m.searchApplied = false
		m.searchQuery = ""
		m.pendingFocusActionItemID = actionItemID
		m.traceGlobalNoticePending("set", "pending_focus_action_item_id", actionItemID, "reason", "search_reset")
		m.pendingOpenActionItemInfoID = actionItemID
		m.traceGlobalNoticePending("set", "pending_open_task_info_id", actionItemID, "reason", "search_reset")
		if hasThreadTarget {
			m.setPendingNotificationThread(threadTarget, threadTitle, threadBody)
		} else {
			m.clearPendingNotificationThread()
		}
		m.status = "loading notification actionItem..."
		return m, m.loadData
	}
	if !m.showArchived {
		m.traceGlobalNoticeBranch("task_reload_include_archived")
		m.showArchived = true
		m.pendingFocusActionItemID = actionItemID
		m.traceGlobalNoticePending("set", "pending_focus_action_item_id", actionItemID, "reason", "include_archived")
		m.pendingOpenActionItemInfoID = actionItemID
		m.traceGlobalNoticePending("set", "pending_open_task_info_id", actionItemID, "reason", "include_archived")
		if hasThreadTarget {
			m.setPendingNotificationThread(threadTarget, threadTitle, threadBody)
		} else {
			m.clearPendingNotificationThread()
		}
		m.status = "loading notification actionItem..."
		return m, m.loadData
	}
	if hasThreadTarget {
		m.traceGlobalNoticeBranch("task_open_thread_fallback")
		m.completeGlobalNoticeTransition("open_thread_fallback")
		return m.startNotificationThread(threadTarget, threadTitle, threadBody)
	}
	m.traceGlobalNoticeBranch("task_not_found")
	m.status = "actionItem not found"
	m.completeGlobalNoticeTransition("task_not_found")
	return m, nil
}

// resolveNoticeAttentionCmd resolves one selected notice attention row and reloads notifications.
func (m Model) resolveNoticeAttentionCmd(item noticesPanelItem) tea.Cmd {
	attentionID := strings.TrimSpace(item.AttentionID)
	if attentionID == "" || m.svc == nil {
		return nil
	}
	return func() tea.Msg {
		_, err := m.svc.ResolveAttentionItem(context.Background(), app.ResolveAttentionItemInput{
			AttentionID:  attentionID,
			ResolvedBy:   m.threadActorID(),
			ResolvedType: m.threadActorType(),
		})
		if err != nil {
			return actionMsg{err: err}
		}
		status := "notification cleared"
		if domain.NormalizeAttentionKind(item.AttentionKind) == domain.AttentionKindMention {
			status = "comment notification cleared"
		}
		return actionMsg{status: status, reload: true}
	}
}

// resolveGlobalNoticeAttentionCmd resolves one selected global notice attention row and reloads notifications.
func (m Model) resolveGlobalNoticeAttentionCmd(item globalNoticesPanelItem) tea.Cmd {
	attentionID := strings.TrimSpace(item.AttentionID)
	if attentionID == "" || m.svc == nil {
		return nil
	}
	return func() tea.Msg {
		_, err := m.svc.ResolveAttentionItem(context.Background(), app.ResolveAttentionItemInput{
			AttentionID:  attentionID,
			ResolvedBy:   m.threadActorID(),
			ResolvedType: m.threadActorType(),
		})
		if err != nil {
			return actionMsg{err: err}
		}
		status := "notification cleared"
		if domain.NormalizeAttentionKind(item.AttentionKind) == domain.AttentionKindMention {
			status = "comment notification cleared"
		}
		return actionMsg{status: status, reload: true}
	}
}

// activateNoticesSelection runs enter behavior for the active notices row.
func (m Model) activateNoticesSelection() (tea.Model, tea.Cmd) {
	item, ok := m.selectedNoticesPanelItem()
	if !ok {
		m.status = "no notices available"
		return m, nil
	}
	if strings.TrimSpace(item.CoordinationProjectID) != "" || item.CoordinationGlobal {
		cmd := m.openCoordinationFromNotice(item.CoordinationProjectID, item.CoordinationGlobal, item.HandoffID)
		return m, cmd
	}
	if req, ok := m.selectedAuthRequestForActiveNotice(); ok && strings.TrimSpace(req.ID) != "" {
		next, cmd, _ := m.beginSelectedAuthRequestDecision("approve")
		return next, cmd
	}
	if item.HasActivity {
		m.activityInfoItem = item.Activity
		m.mode = modeActivityEventInfo
		m.status = "activity event"
		return m, nil
	}
	actionItemID := strings.TrimSpace(item.ActionItemID)
	if actionItemID != "" {
		if m.openActionItemInfo(actionItemID, "actionItem info") {
			m.noticesFocused = false
			return m, nil
		}
		if m.searchApplied || m.searchQuery != "" {
			m.searchApplied = false
			m.searchQuery = ""
			m.pendingFocusActionItemID = actionItemID
			m.pendingOpenActionItemInfoID = actionItemID
			m.status = "loading notification actionItem..."
			return m, m.loadData
		}
		if !m.showArchived {
			m.showArchived = true
			m.pendingFocusActionItemID = actionItemID
			m.pendingOpenActionItemInfoID = actionItemID
			m.status = "loading notification actionItem..."
			return m, m.loadData
		}
	}
	projectID := strings.TrimSpace(item.ProjectID)
	if projectID == "" {
		projectID, _ = m.currentProjectID()
	}
	hasScopedThreadTarget := strings.TrimSpace(string(item.ScopeType)) != "" || strings.TrimSpace(item.ScopeID) != "" || strings.TrimSpace(item.ProjectID) != ""
	if hasScopedThreadTarget {
		title := strings.TrimSpace(item.ThreadTitle)
		if title == "" {
			title = notificationThreadTitle(item.ScopeType, item.Label)
		}
		if target, ok := commentTargetForScope(projectID, item.ScopeType, item.ScopeID); ok {
			return m.startNotificationThread(target, title, item.ThreadDescription)
		}
		// If scoped metadata is malformed, fall back to the project thread so Enter still performs a deterministic action.
		if target, ok := commentTargetForScope(projectID, domain.ScopeLevelProject, projectID); ok {
			return m.startNotificationThread(target, title, item.ThreadDescription)
		}
	}
	if actionItemID != "" {
		m.status = "actionItem not found"
		return m, nil
	}
	if hasScopedThreadTarget {
		m.status = "selected notice thread target unavailable"
		return m, nil
	}
	m.status = "selected notice has no action"
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

// opaqueActorIDPattern matches UUID-like actor identifiers that add noise in read surfaces.
var opaqueActorIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// readableActorLabel resolves one human-facing actor label and whether raw-id context should stay hidden.
func (m Model) readableActorLabel(actorType domain.ActorType, actorID, actorName string) (string, bool) {
	actorType = normalizeActivityActorType(actorType)
	actorName = strings.TrimSpace(actorName)
	actorID = strings.TrimSpace(actorID)
	hideActorIDContext := false
	if actorType == domain.ActorTypeUser {
		localActorID := strings.TrimSpace(m.identityActorID)
		localName := strings.TrimSpace(m.identityDisplayName)
		if localName != "" {
			switch {
			case strings.EqualFold(actorName, "tillsyn-user"),
				strings.EqualFold(actorID, "tillsyn-user"),
				(actorName == "" && actorID != "" && strings.EqualFold(actorID, localActorID)),
				(strings.EqualFold(actorName, actorID) && strings.EqualFold(actorID, localActorID)):
				actorName = localName
				hideActorIDContext = true
			case actorName != "" && strings.EqualFold(actorID, localActorID):
				hideActorIDContext = true
			}
		}
	}
	if actorName != "" {
		return actorName, hideActorIDContext
	}
	if actorID != "" {
		return actorID, hideActorIDContext
	}
	return "unknown", hideActorIDContext
}

// isOpaqueActorID reports whether one actor identifier is too noisy to include in read surfaces.
func isOpaqueActorID(actorID string) bool {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return false
	}
	if strings.EqualFold(actorID, "tillsyn-user") {
		return true
	}
	return opaqueActorIDPattern.MatchString(actorID)
}

// displayActivityOwner returns display-safe owner fields for activity rendering.
func (m Model) displayActivityOwner(entry activityEntry) (domain.ActorType, string) {
	actorType := normalizeActivityActorType(entry.ActorType)
	owner, _ := m.readableActorLabel(actorType, entry.ActorID, entry.ActorName)
	return actorType, owner
}

// displayActivityOwnerWithContext returns owner text with compact actor-id context when informative.
func (m Model) displayActivityOwnerWithContext(entry activityEntry) (domain.ActorType, string) {
	actorType := normalizeActivityActorType(entry.ActorType)
	owner, hideActorIDContext := m.readableActorLabel(actorType, entry.ActorID, entry.ActorName)
	actorID := strings.TrimSpace(entry.ActorID)
	if hideActorIDContext || actorID == "" || strings.EqualFold(owner, actorID) || strings.EqualFold(actorID, "unknown") || isOpaqueActorID(actorID) {
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

// actionItemSystemActorLine renders one readable system ownership line using activity identity context when available.
func (m Model) actionItemSystemActorLine(label string, actionItem domain.ActionItem, actorID string, fallbackType domain.ActorType, preferCreate bool) string {
	owner, actorType := m.actionItemSystemActorLabel(actionItem, actorID, fallbackType, preferCreate)
	if actorType == "" {
		return label + ": " + owner
	}
	return label + ": " + owner + " (" + string(actorType) + ")"
}

// actionItemSystemActorLabel resolves one readable actionItem ownership label and actor type for system sections.
func (m Model) actionItemSystemActorLabel(actionItem domain.ActionItem, actorID string, fallbackType domain.ActorType, preferCreate bool) (string, domain.ActorType) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return "-", ""
	}
	if entry, ok := m.findActionItemActivityActorEntry(actionItem.ID, actorID, preferCreate); ok {
		actorType, owner := m.displayActivityOwner(entry)
		return owner, actorType
	}
	entry := activityEntry{
		ActionItemID: actionItem.ID,
		ActorID:    actorID,
		ActorType:  fallbackType,
	}
	actorType, owner := m.displayActivityOwner(entry)
	return owner, actorType
}

// findActionItemActivityActorEntry finds a matching activity entry for one actionItem actor, preferring create or latest events.
func (m Model) findActionItemActivityActorEntry(actionItemID, actorID string, preferCreate bool) (activityEntry, bool) {
	actionItemID = strings.TrimSpace(actionItemID)
	actorID = strings.TrimSpace(actorID)
	if actionItemID == "" || actorID == "" {
		return activityEntry{}, false
	}
	if preferCreate {
		for _, entry := range m.activityLog {
			if strings.TrimSpace(entry.ActionItemID) != actionItemID {
				continue
			}
			if entry.Operation != domain.ChangeOperationCreate {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(entry.ActorID), actorID) {
				continue
			}
			return entry, true
		}
	}
	for idx := len(m.activityLog) - 1; idx >= 0; idx-- {
		entry := m.activityLog[idx]
		if strings.TrimSpace(entry.ActionItemID) != actionItemID {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(entry.ActorID), actorID) {
			continue
		}
		return entry, true
	}
	return activityEntry{}, false
}

// renderOverviewPanel renders the right-side notices panel for board scope context.
func (m Model) renderOverviewPanel(
	project domain.Project,
	accent, muted, dim color.Color,
	width int,
	height int,
	attentionItems, attentionTotal, attentionBlocked int,
	attentionTop []string,
	focused bool,
) string {
	panelWidth := max(24, width)
	panelHeight := max(10, height)
	contentWidth := max(12, panelWidth-6)
	normalStyle := lipgloss.NewStyle().Foreground(muted)
	selectedStyle := lipgloss.NewStyle().Foreground(accent).Bold(true)
	_ = project

	sections := m.noticesPanelSections(attentionItems, attentionTotal, attentionBlocked, attentionTop)
	viewModel := m
	viewModel.clampNoticesSelectionForSections(sections)

	// Keep stacked project/global notifications aligned with the board column height.
	const minStackPanelHeight = 5 // 1 content row + 4 rows of border/padding chrome
	globalPanelHeight := panelHeight / 3
	if globalPanelHeight < minStackPanelHeight {
		globalPanelHeight = minStackPanelHeight
	}
	maxGlobalHeight := panelHeight - minStackPanelHeight
	if globalPanelHeight > maxGlobalHeight {
		globalPanelHeight = maxGlobalHeight
	}
	globalNaturalHeight := max(minStackPanelHeight, min(maxGlobalHeight, 4+globalNoticesPanelContentLines(m.globalNoticesPanelItemsForInteraction(), m.globalNoticesPartialCount)))
	if globalPanelHeight > globalNaturalHeight {
		globalPanelHeight = globalNaturalHeight
	}
	projectPanelHeight := panelHeight - globalPanelHeight
	if projectPanelHeight < minStackPanelHeight {
		projectPanelHeight = minStackPanelHeight
		globalPanelHeight = panelHeight - projectPanelHeight
	}
	projectContentHeight := max(1, projectPanelHeight-4)
	globalContentHeight := max(1, globalPanelHeight-4)
	projectHeaderLines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render(truncate("Project Notifications", contentWidth)),
		"",
	}
	projectBodyLines := make([]string, 0, len(sections)*(noticesSectionViewWindow+4))
	projectFocusLine := 0
	projectFocusSet := false
	for _, section := range sections {
		if len(projectBodyLines) > 0 {
			projectBodyLines = append(projectBodyLines, "")
		}
		sectionLines, sectionFocusLine := viewModel.renderNoticesSection(
			section,
			focused && m.noticesPanel == noticesPanelFocusProject,
			accent,
			contentWidth,
			selectedStyle,
			normalStyle,
		)
		if !projectFocusSet && sectionFocusLine >= 0 {
			projectFocusLine = len(projectBodyLines) + sectionFocusLine
			projectFocusSet = true
		}
		projectBodyLines = append(projectBodyLines, sectionLines...)
	}
	projectBodyHeight := max(0, projectContentHeight-len(projectHeaderLines))
	projectLines := append(projectHeaderLines, scrollPanelBodyLines(projectBodyLines, projectFocusLine, projectBodyHeight)...)
	projectBorderColor := dim
	if focused && m.noticesPanel == noticesPanelFocusProject {
		projectBorderColor = accent
	}
	projectPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(projectBorderColor).
		Padding(1, 1).
		Width(panelWidth).
		Render(fitLines(strings.Join(projectLines, "\n"), projectContentHeight))

	globalPanel := m.renderGlobalNoticesPanel(
		accent,
		muted,
		dim,
		panelWidth,
		contentWidth,
		globalContentHeight,
		focused && m.noticesPanel == noticesPanelFocusGlobal,
		selectedStyle,
		normalStyle,
	)
	return lipgloss.JoinVertical(lipgloss.Top, projectPanel, globalPanel)
}

// globalNoticesItemLines builds display lines for one global-notifications row.
func globalNoticesItemLines(item globalNoticesPanelItem) []string {
	summary := strings.TrimSpace(item.Summary)
	if strings.TrimSpace(item.StableKey) == globalNoticesEmptyRowKey {
		if summary == "" {
			summary = "no coordination or notifications across other projects"
		}
		return []string{summary}
	}
	projectLabel := firstNonEmptyTrimmed(item.ProjectLabel, item.ProjectID, "project")
	lines := []string{projectLabel}
	if summary == "" {
		lines = append(lines, "notification")
		return lines
	}
	if strings.HasPrefix(strings.TrimSpace(item.StableKey), "coordination:") || item.CoordinationGlobal {
		for _, line := range strings.Split(summary, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			lines = append(lines, line)
		}
		return lines
	}
	switch domain.NormalizeAttentionKind(item.AttentionKind) {
	case domain.AttentionKindMention:
		lines = append(lines, "comment: "+summary)
	case domain.AttentionKindHandoff:
		lines = append(lines, "handoff: "+summary)
	default:
		lines = append(lines, "action required: "+summary)
	}
	return lines
}

// globalNoticesPanelContentLines estimates the natural content height for the lower notifications panel.
func globalNoticesPanelContentLines(items []globalNoticesPanelItem, partialCount int) int {
	lines := 2 // title + gap
	if partialCount > 0 {
		lines += 2 // partial-results line + gap
	}
	for idx, item := range items {
		lines += len(globalNoticesItemLines(item))
		if idx < len(items)-1 {
			lines++
		}
	}
	if lines < 1 {
		return 1
	}
	return lines
}

// renderGlobalNoticesPanel renders the lower global notifications panel.
func (m Model) renderGlobalNoticesPanel(
	accent, muted, dim color.Color,
	panelWidth int,
	contentWidth int,
	contentHeight int,
	focused bool,
	selectedStyle, normalStyle lipgloss.Style,
) string {
	viewModel := m
	viewModel.clampGlobalNoticesSelection()
	items := viewModel.globalNoticesPanelItemsForInteraction()
	selectedIdx := clamp(viewModel.globalNoticesIdx, 0, len(items)-1)
	start, end := windowBounds(len(items), selectedIdx, noticesSectionViewWindow)

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render(truncate("Global Notifications", contentWidth)),
		"",
	}
	if viewModel.globalNoticesPartialCount > 0 {
		projectLabel := "projects"
		if viewModel.globalNoticesPartialCount == 1 {
			projectLabel = "project"
		}
		lines = append(
			lines,
			normalStyle.Render(
				truncate(fmt.Sprintf("partial results: %d %s unavailable", viewModel.globalNoticesPartialCount, projectLabel), contentWidth),
			),
		)
		lines = append(lines, "")
	}
	if focused && start > 0 {
		lines = append(lines, normalStyle.Render(truncate("↑ more", contentWidth)))
	}
	for idx := start; idx < end; idx++ {
		item := items[idx]
		prefix := "  "
		if !focused {
			prefix = ""
		}
		itemLines := globalNoticesItemLines(item)
		for lineIdx, line := range itemLines {
			linePrefix := prefix
			style := normalStyle
			if focused && idx == selectedIdx {
				if lineIdx == 0 {
					linePrefix = "› "
				}
				style = selectedStyle
			}
			if lineIdx > 0 {
				linePrefix = strings.Repeat(" ", utf8.RuneCountInString(prefix))
				if focused && idx == selectedIdx {
					style = normalStyle
				}
			}
			lines = append(lines, style.Render(linePrefix+truncate(line, max(1, contentWidth-utf8.RuneCountInString(linePrefix)))))
		}
		if idx < end-1 {
			lines = append(lines, "")
		}
	}
	if focused && end < len(items) {
		lines = append(lines, normalStyle.Render(truncate("↓ more", contentWidth)))
	}
	borderColor := dim
	if focused {
		borderColor = accent
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 1).
		Width(panelWidth)
	return style.Render(fitLines(strings.Join(lines, "\n"), contentHeight))
}

// renderNoticesSection renders one notices-panel section with local list-window scrolling.
func (m Model) renderNoticesSection(
	section noticesPanelSection,
	focused bool,
	accent color.Color,
	contentWidth int,
	selectedStyle, normalStyle lipgloss.Style,
) ([]string, int) {
	lines := make([]string, 0, len(section.Summary)+noticesSectionViewWindow+3)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	focusLine := -1
	if focused && section.ID == m.noticesSection {
		lines = append(lines, headerStyle.Render("▸ "+section.Title))
		focusLine = 0
	} else {
		lines = append(lines, headerStyle.Render(section.Title))
	}

	renderItems := func() {
		if len(section.Items) == 0 {
			lines = append(lines, normalStyle.Render("(empty)"))
			if focusLine < 0 && focused && section.ID == m.noticesSection {
				focusLine = len(lines) - 1
			}
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
				focusLine = len(lines)
			}
			lineWidth := max(1, contentWidth-utf8.RuneCountInString(prefix))
			lines = append(lines, style.Render(prefix+truncate(item.Label, lineWidth)))
		}
		if focused && end < len(section.Items) {
			lines = append(lines, normalStyle.Render(truncate("↓ more", contentWidth)))
		}
	}

	for _, summary := range section.Summary {
		lines = append(lines, normalStyle.Render(truncate(summary, contentWidth)))
	}
	renderItems()
	return lines, focusLine
}

// scrollPanelBodyLines keeps the focused line visible inside one fixed-height side-panel body.
func scrollPanelBodyLines(lines []string, focusLine, height int) []string {
	if height <= 0 {
		return nil
	}
	if len(lines) <= height {
		return strings.Split(fitLines(strings.Join(lines, "\n"), height), "\n")
	}
	if focusLine < 0 {
		focusLine = 0
	}
	start, end := windowBounds(len(lines), focusLine, height)
	return append([]string(nil), lines[start:end]...)
}

// renderInfoLine renders output for the current model state.
func (m Model) renderInfoLine(project domain.Project, muted color.Color) string {
	_ = project
	actionItem, ok := m.selectedActionItemInCurrentColumn()
	if !ok {
		if strings.TrimSpace(m.projectionRootActionItemID) != "" {
			return lipgloss.NewStyle().Foreground(muted).Render(fmt.Sprintf("%s full board", m.keys.clearFocus.Help().Key))
		}
		return ""
	}
	parts := []string{}
	if children := m.directChildCount(actionItem.ID); children > 0 {
		parts = append(parts, fmt.Sprintf("children: %d", children))
		if strings.TrimSpace(m.projectionRootActionItemID) == "" {
			parts = append(parts, fmt.Sprintf("%s focus subtree", m.keys.focusSubtree.Help().Key))
		}
	}
	if strings.TrimSpace(m.projectionRootActionItemID) != "" {
		parts = append(parts, fmt.Sprintf("%s full board", m.keys.clearFocus.Help().Key))
	}
	if len(parts) == 0 {
		return ""
	}
	return lipgloss.NewStyle().Foreground(muted).Render(strings.Join(parts, " • "))
}

// shouldSuppressBoardStatus hides low-value transient messages from the board footer/status area.
func shouldSuppressBoardStatus(status string) bool {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "",
		"ready",
		"cancelled",
		"project switched",
		"board focus",
		"global notifications focus",
		"project notifications focus",
		"global notifications",
		"quick actions",
		"command palette",
		"actionitem info",
		"parent actionitem info",
		"edit actionitem",
		"new actionitem",
		"edit project",
		"new project",
		"thread loaded",
		"due picker",
		"resource picker",
		"label picker",
		"project picker",
		"search",
		"search results",
		"paths/roots",
		"description editor":
		return true
	}
	if strings.HasPrefix(status, "project notifications:") {
		return true
	}
	if strings.HasPrefix(status, "loading ") {
		return true
	}
	if strings.HasPrefix(status, "text selection mode ") {
		return true
	}
	if strings.HasPrefix(status, "editing ") {
		return true
	}
	if strings.HasSuffix(status, " cancelled") {
		return true
	}
	if strings.HasSuffix(status, " focus") {
		return true
	}
	return false
}

// boardStatusText returns the board-visible status string after transient filtering.
func (m Model) boardStatusText() string {
	status := strings.TrimSpace(m.status)
	if shouldSuppressBoardStatus(status) {
		return ""
	}
	return status
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
	lines = append(
		lines,
		"",
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
		panelLine := "tab/shift+tab cycle focused panel; left/right wrap panel focus; enter opens notices detail"
		if m.hasGlobalNoticesPanel() {
			panelLine = "tab/shift+tab cycle board/project/global panels; left/right wraps panel focus"
			if m.noticesFocused {
				if m.noticesPanel == noticesPanelFocusGlobal {
					panelLine = "tab/shift+tab cycle board/project/global panels; up/down move global notifications; left/right wrap panels; enter opens selected item; x clears selected comment mention"
				} else {
					panelLine = "tab/shift+tab cycle board/project/global panels; up/down move project notifications; left/right wrap panels; enter opens selected item; x clears selected comment mention"
				}
			}
		}
		return "board", []string{
			"h/l or left/right move columns; j/k or up/down move actionItem selection",
			"n new actionItem; i actionItem info; e edit actionItem",
			"space multi-select; [ / ] move actionItem; d delete; D hard delete; u restore",
			"f/enter focus subtree; F full board; t toggle archived",
			"/ search; p project picker; : command palette; . quick actions",
			panelLine,
			"ctrl+y toggles text selection mode; ctrl+c/ctrl+v copy/paste in text inputs",
			"ctrl+z undo; ctrl+shift+z redo; g activity log; q quit",
		}
	case modeAddActionItem:
		return "new actionItem", []string{
			"tab/shift+tab move fields; enter or e opens the focused field action; esc cancels",
			"description and metadata fields open the full markdown editor",
			"h/l changes priority when priority field is focused",
			"due field: enter or e opens due picker",
			"labels field: enter or e opens label picker",
			"depends_on/blocked_by fields: enter or e opens dependency picker",
			"use depends_on, blocked_by, and blocked_reason to express prerequisite order; do not rely on board position alone for execution sequencing",
			"subtasks/comments/resources are save-dependent rows here; save the actionItem first, then manage them in edit mode",
			"ctrl+s saves form",
		}
	case modeEditActionItem:
		return "edit actionItem", []string{
			"tab/shift+tab move fields; up/down wraps between first and last field; enter or e opens the focused field action; esc cancels",
			"description and metadata fields open full markdown editor (enter or e)",
			"h/l changes priority when priority field is focused",
			"due field: enter or e opens due picker",
			"labels field: enter or e opens label picker",
			"depends_on/blocked_by fields: enter or e opens dependency picker",
			"use depends_on, blocked_by, and blocked_reason to express prerequisite order; do not rely on board position alone for execution sequencing",
			"subtasks section: first existing child is focused when present; left returns to + create; enter or e opens selected row",
			"comments section: enter or e opens thread on the comments panel; . opens focused quick actions",
			"resources section: first existing item is focused when present; left returns to + attach; enter or e opens resource action",
			"press . for focused quick actions on subtasks/resources/comments and other action rows",
			"ctrl+s saves form; markdown editor ctrl+s saves the actionItem for existing items",
		}
	case modeSearch:
		return "search", []string{
			"tab cycles query, states, levels, scope, archived, mode, and apply",
			"space or enter toggles the focused state/level/scope/mode option",
			"h/l cycles state/level cursors and toggles scope/archived/mode",
			"ctrl+u clears query; ctrl+r resets filters; esc cancels",
		}
	case modeRenameActionItem:
		return "rename actionItem", []string{
			"type new title",
			"enter saves; esc cancels",
		}
	case modeDuePicker:
		return "due picker", []string{
			"tab cycles include-time, date, time, and options list focus",
			"space toggles include time when toggle is focused",
			"type date/time in the picker to update dynamic suggestions",
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
	case modeActionItemInfo:
		return "actionItem info", []string{
			"j/k and up/down scroll full info content and move subtask cursor",
			"enter opens the focused subtask when one is selected",
			"backspace moves to parent actionItem info when available",
			"pgup/pgdown, home/end, or ctrl+u/ctrl+d scroll the full info body",
			"d opens full-screen details preview; tab toggles edit mode there",
			"e edits the current actionItem; s creates a subtask; c opens thread on comments; . opens actionItem/subtask quick actions",
			"[ / ] move actionItem between columns; space toggles the focused subtask; esc back/close",
		}
	case modeAddProject:
		return "new project", []string{
			"tab/shift+tab moves fields; enter saves; esc cancels",
			"description field opens full markdown editor (enter or i)",
			"kind field opens the project-kind picker (enter/e; typing starts a filtered picker)",
			"icon field is shown in path context, notices, and picker and supports emoji",
			"template library field opens the approved-library picker (enter/e; typing starts a filtered picker) and seeds allowed kinds from the selected library",
			"confirm with the dev whether extra generic kinds should be allowed after template selection",
			"root_path field: r opens directory picker",
		}
	case modeEditProject:
		return "edit project", []string{
			"tab/shift+tab moves fields; enter saves; esc cancels",
			"description field opens full markdown editor (enter or i)",
			"kind field opens the project-kind picker; changing it updates template matching for future work",
			"icon field is shown in path context, notices, and picker and supports emoji",
			"template library field opens the approved-library picker; choose (none) to clear the active project binding",
			"rebinding should include an explicit generic-kind decision with the dev; template-only is the safe default",
			"root_path field: r opens directory picker",
			"comments row: enter or e opens the project thread on the comments panel",
		}
	case modeProjectKindPicker:
		return "project kind picker", []string{
			"type to fuzzy-filter project kinds",
			"j/k moves selection; enter chooses the highlighted kind",
			"the selected kind is sent on project create/update",
			"ctrl+u clears the picker filter; esc closes the picker",
		}
	case modeTemplateLibraryPicker:
		return "template library picker", []string{
			"type to fuzzy-filter approved global template libraries",
			"j/k moves selection; enter chooses the highlighted library",
			"the (none) row clears project-level template binding",
			"ctrl+u clears the picker filter; esc closes the picker",
		}
	case modeDescriptionEditor:
		return "description editor", []string{
			"tab toggles edit and preview layouts",
			"edit mode: full markdown editor with live preview and synced scrolling",
			"preview mode: full-screen rendered markdown with scrolling",
			"ctrl+s saves description; esc cancels editor changes",
		}
	case modeSearchResults:
		return "search results", []string{
			"j/k moves result selection",
			"enter jumps to selected result",
			"esc closes results",
		}
	case modeEmbeddingsStatus:
		return "embeddings", []string{
			"shows lifecycle counts and human-readable subject rows for the current or global scope",
			"/ opens inline filtering for the visible lifecycle rows",
			"enter opens the selected row in actionItem info or project details",
			"j/k moves the selected lifecycle row",
			"g toggles current-project vs all-project scope",
			"a toggles archived-project inclusion",
			"r enqueues reindex for the current scope; R forces reindex",
			"esc closes the modal",
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
			"actions are scoped to the current screen or focused row",
			"esc closes quick actions",
		}
	case modeConfirmAction:
		return "confirm action", []string{
			"h/l switches confirm vs cancel",
			"enter applies highlighted choice",
			"y confirms immediately; n cancels; esc cancels",
		}
	case modeCoordinationDetail:
		return "coordination detail", []string{
			"shows the selected coordination row with state-specific styling",
			"j/k, up/down, or tab move between available actions",
			"enter runs the selected action or closes when close is selected",
			"r jumps straight to revoke when the selected detail supports it",
			"esc returns to the coordination list",
		}
	case modeWarning:
		return "warning", []string{
			"warning modal blocks accidental context mistakes",
			"enter or esc closes the warning",
		}
	case modeActivityLog:
		return "activity log", []string{
			"esc closes activity log",
			"ctrl+z undo and ctrl+shift+z redo remain available",
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
			"a applies changes; enter jumps to actionItem; esc cancels",
		}
	case modeAuthInventory:
		return "coordination", []string{
			"live view defaults to pending requests, active sessions, active leases, and open handoffs",
			"h toggles between live and history slices",
			"j/k, up/down, or mouse wheel move the selected row",
			"pgup/pgdown or ctrl+u/ctrl+d move faster through long inventories",
			"g toggles request/session scope between project and global inventory",
			"enter reviews a pending request or opens typed detail/actions for the selected row",
			"r opens revoke review when an active session row is selected",
			"esc returns to the previous screen",
		}
	case modeThread:
		return "thread", []string{
			"tab/shift+tab or left/right cycle details, comments, and context panels",
			"enter opens the focused panel action",
			"i starts comment composition when the comments panel is focused",
			"ctrl+s posts while composing; esc exits composer or returns to the prior screen",
			"up/down, pgup/pgdown/home/end, or mouse wheel scroll comments",
		}
	case modeDiff:
		return "diff", []string{
			"up/down or j/k scroll one line at a time",
			"pgup/pgdown or ctrl+u/ctrl+d move half a page",
			"esc returns to the prior screen",
			"a divergence banner appears when the start commit is not an ancestor of HEAD",
		}
	case modeFileViewer:
		return "file-viewer", []string{
			"up/down or j/k scroll one line at a time",
			"pgup/pgdown move half a page",
			"esc returns to the prior screen",
		}
	default:
		return "current screen", []string{
			"enter confirms primary action",
			"esc closes current screen",
		}
	}
}

// actionItemListSecondary returns actionItem list secondary.
func (m Model) actionItemListSecondary(actionItem domain.ActionItem) string {
	if m.actionItemFields.ShowDescription {
		if desc := strings.TrimSpace(actionItem.Description); desc != "" {
			return desc
		}
	}
	if meta := m.cardMeta(actionItem); meta != "" {
		return meta
	}
	return ""
}

// actionItemIndexAtRow returns actionItem index at row.
func (m Model) actionItemIndexAtRow(tasks []domain.ActionItem, row int) int {
	if len(tasks) == 0 {
		return 0
	}
	if row <= 0 {
		return 0
	}
	current := 0
	for idx, actionItem := range tasks {
		start := current
		span := 1
		if m.actionItemListSecondary(actionItem) != "" {
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
func (m Model) cardMeta(actionItem domain.ActionItem) string {
	parts := make([]string, 0, 4)
	if marker := actionItemHierarchyMarker(actionItem); marker != "" {
		parts = append(parts, marker)
	}
	if m.actionItemFields.ShowPriority {
		parts = append(parts, string(actionItem.Priority))
	}
	if actionItem.Kind != domain.KindSubtask {
		done, total := m.subactionItemProgress(actionItem.ID)
		if total > 0 {
			parts = append(parts, fmt.Sprintf("%d/%d", done, total))
		}
	}
	if m.actionItemFields.ShowDueDate && actionItem.DueAt != nil {
		dueLabel := actionItem.DueAt.UTC().Format("01-02")
		if actionItem.DueAt.UTC().Before(time.Now().UTC()) {
			dueLabel = "!" + dueLabel
		}
		parts = append(parts, dueLabel)
	}
	if m.actionItemFields.ShowLabels && len(actionItem.Labels) > 0 {
		parts = append(parts, summarizeLabels(actionItem.Labels, 2))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, "|") + "]"
}

// actionItemHierarchyMarker returns a compact level marker for hierarchy-scoped work items.
func actionItemHierarchyMarker(actionItem domain.ActionItem) string {
	switch baseSearchLevelForActionItem(actionItem) {
	case "branch":
		return "branch"
	case "phase":
		return "phase"
	default:
		return ""
	}
}

// actionItemDueWarning reports due warning text for one actionItem in board/info contexts.
func (m Model) actionItemDueWarning(actionItem domain.ActionItem, now time.Time) string {
	if actionItem.ArchivedAt != nil || actionItem.DueAt == nil {
		return ""
	}
	now = now.UTC()
	due := actionItem.DueAt.UTC()
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

// actionItemInfoActionItem resolves the actionItem currently shown in the actionItem-info modal.
func (m Model) actionItemInfoActionItem() (domain.ActionItem, bool) {
	actionItemID := strings.TrimSpace(m.actionItemInfoActionItemID)
	if actionItemID == "" {
		return m.selectedActionItemInCurrentColumn()
	}
	return m.actionItemByID(actionItemID)
}

// clearActionItemInfoComments clears cached actionItem-info comment preview state.
func (m *Model) clearActionItemInfoComments() {
	m.actionItemInfoComments = nil
	m.actionItemInfoCommentsError = ""
}

// loadActionItemInfoComments refreshes actionItem-info comment previews for one actionItem id.
func (m *Model) loadActionItemInfoComments(actionItemID string) {
	m.clearActionItemInfoComments()
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return
	}
	actionItem, ok := m.actionItemByID(actionItemID)
	if !ok {
		return
	}
	targetType, ok := commentTargetTypeForActionItem(actionItem)
	if !ok {
		return
	}
	comments, err := m.svc.ListCommentsByTarget(context.Background(), app.ListCommentsByTargetInput{
		ProjectID:  actionItem.ProjectID,
		TargetType: targetType,
		TargetID:   actionItem.ID,
	})
	if err != nil {
		m.actionItemInfoCommentsError = err.Error()
		return
	}
	m.actionItemInfoComments = append([]domain.Comment(nil), comments...)
}

// openActionItemInfo enters actionItem-info mode and initializes traversal state for esc path retrace behavior.
func (m *Model) openActionItemInfo(actionItemID string, status string) bool {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return false
	}
	actionItem, ok := m.actionItemByID(actionItemID)
	if !ok {
		return false
	}
	m.mode = modeActionItemInfo
	m.actionItemInfoActionItemID = actionItemID
	m.actionItemInfoOriginActionItemID = actionItemID
	m.actionItemInfoPath = []string{actionItemID}
	m.actionItemInfoSubactionItemIdx = 0
	m.actionItemInfoFocusedSubactionItemID = ""
	m.actionItemInfoDetails.SetYOffset(0)
	m.actionItemInfoBody.SetYOffset(0)
	m.loadActionItemInfoComments(actionItemID)
	m.syncActionItemInfoDetailsViewport(actionItem)
	m.syncActionItemInfoBodyViewport(actionItem)
	if strings.TrimSpace(status) == "" {
		status = "actionItem info"
	}
	m.status = status
	return true
}

// closeActionItemInfo exits actionItem-info mode and clears tracked traversal/actionItem state.
func (m *Model) closeActionItemInfo(status string) {
	m.mode = modeNone
	m.actionItemInfoActionItemID = ""
	m.actionItemInfoOriginActionItemID = ""
	m.actionItemInfoPath = nil
	m.actionItemInfoSubactionItemIdx = 0
	m.actionItemInfoFocusedSubactionItemID = ""
	m.actionItemInfoDetails.SetYOffset(0)
	m.actionItemInfoBody.SetYOffset(0)
	m.clearActionItemInfoComments()
	if strings.TrimSpace(status) == "" {
		status = "ready"
	}
	m.status = status
}

// actionItemInfoOverlayBoxWidth resolves actionItem-info modal width bounds from the available terminal width.
func actionItemInfoOverlayBoxWidth(maxWidth int) int {
	if maxWidth > 0 {
		return max(36, maxWidth)
	}
	return 96
}

// markdownPreviewHeight resolves a bounded markdown-preview height from rendered content.
func (m Model) markdownPreviewHeight(rendered string) int {
	height := lipgloss.Height(rendered)
	if height <= 0 {
		height = actionItemInfoDetailsViewportMinHeight
	}
	maxHeight := actionItemInfoDetailsViewportMaxHeight
	if m.height > 0 {
		maxHeight = min(maxHeight, max(1, m.height-14))
	}
	return clamp(height, actionItemInfoDetailsViewportMinHeight, max(1, maxHeight))
}

// markdownPreviewContent renders a bounded markdown preview for node info/edit surfaces.
func (m Model) markdownPreviewContent(markdown string, width int, empty string) string {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return empty
	}
	return strings.TrimSpace(m.threadMarkdown.render(markdown, max(20, width)))
}

// actionItemInfoDescriptionMarkdown renders actionItem description markdown for the actionItem-info details viewport.
func (m Model) actionItemInfoDescriptionMarkdown(actionItem domain.ActionItem, width int) string {
	return m.markdownPreviewContent(actionItem.Description, width, "(no description)")
}

// actionItemDescriptionPreviewViewport builds the bounded top-aligned markdown preview used by info/edit screens.
func (m Model) actionItemDescriptionPreviewViewport(markdown string, boxWidth int) viewport.Model {
	return m.actionItemDescriptionPreviewViewportForContentWidth(markdown, max(24, boxWidth-4))
}

// actionItemDescriptionPreviewViewportForContentWidth builds the shared bounded markdown preview for a measured content width.
func (m Model) actionItemDescriptionPreviewViewportForContentWidth(markdown string, contentWidth int) viewport.Model {
	contentWidth = max(24, contentWidth)
	rendered := m.markdownPreviewContent(markdown, contentWidth, "(no description)")
	vp := viewport.New()
	vp.SoftWrap = true
	vp.MouseWheelEnabled = false
	vp.SetWidth(contentWidth)
	vp.SetHeight(max(1, m.markdownPreviewHeight(rendered)))
	vp.SetContent(rendered)
	vp.SetYOffset(0)
	return vp
}

// actionItemInfoDescriptionViewport builds the bounded markdown-details viewport for actionItem-info rendering.
func (m Model) actionItemInfoDescriptionViewport(actionItem domain.ActionItem, boxWidth int) viewport.Model {
	return m.actionItemDescriptionPreviewViewport(actionItem.Description, boxWidth)
}

// syncActionItemInfoDetailsViewport refreshes markdown-details viewport dimensions/content after actionItem/size changes.
func (m *Model) syncActionItemInfoDetailsViewport(actionItem domain.ActionItem) {
	if m == nil {
		return
	}
	m.actionItemInfoDetails = m.actionItemDescriptionPreviewViewport(actionItem.Description, actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())))
}

func (m Model) fullPageNodeScreenHasPath() bool {
	_, ok := m.currentProject()
	return ok
}

func (m Model) fullPageNodeContentWidth() int {
	if m.width <= 0 {
		return 96
	}
	return max(36, m.width-(2*tuiOuterHorizontalPadding))
}

// fullPageNodeBodyHeight resolves the scrollable viewport height for full-page node surfaces.
func (m Model) fullPageNodeBodyHeight(hasSubtitle bool) int {
	if m.height <= 0 {
		return 24
	}
	reserved := 12
	if hasSubtitle {
		reserved++
	}
	return clamp(m.height-reserved, actionItemInfoBodyViewportMinHeight, actionItemInfoBodyViewportMaxHeight)
}

func ensureViewportLineVisible(vp *viewport.Model, focusLine int) {
	if vp == nil || focusLine < 0 {
		return
	}
	top := vp.YOffset()
	bottom := top + vp.Height() - 1
	switch {
	case focusLine < top:
		vp.SetYOffset(focusLine)
	case focusLine > bottom:
		vp.SetYOffset(max(0, focusLine-vp.Height()+1))
	}
}

// ensureViewportRangeVisible keeps both ends of one important line range visible when the viewport height allows it.
func ensureViewportRangeVisible(vp *viewport.Model, startLine, endLine int) {
	if vp == nil {
		return
	}
	if startLine < 0 {
		ensureViewportLineVisible(vp, endLine)
		return
	}
	if endLine < 0 {
		ensureViewportLineVisible(vp, startLine)
		return
	}
	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}
	height := vp.Height()
	if height <= 0 {
		return
	}
	minTop := max(0, endLine-height+1)
	maxTop := max(0, startLine)
	if minTop <= maxTop {
		vp.SetYOffset(clamp(vp.YOffset(), minTop, maxTop))
		return
	}
	vp.SetYOffset(minTop)
}

func (m *Model) syncActionItemFormViewportToFocus() {
	if m == nil || (m.mode != modeAddActionItem && m.mode != modeEditActionItem) {
		return
	}
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	boxWidth := actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth()))
	title := "New " + m.actionItemFormNodeLabel()
	if m.mode == modeEditActionItem {
		title = "Edit " + m.actionItemFormNodeLabel()
	}
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, title, m.actionItemFormHeaderMeta(), "")
	bodyLines, focusLine := m.actionItemFormBodyLines(metrics.contentWidth, lipgloss.NewStyle(), lipgloss.Color("252"))
	prevYOffset := m.actionItemInfoBody.YOffset()
	m.actionItemInfoBody.SetWidth(metrics.contentWidth)
	m.actionItemInfoBody.SetHeight(max(1, metrics.bodyHeight))
	m.actionItemInfoBody.SetContent(strings.Join(bodyLines, "\n"))
	m.actionItemInfoBody.SetYOffset(prevYOffset)
	ensureViewportLineVisible(&m.actionItemInfoBody, focusLine)
}

// actionItemNodeLabel resolves a display-safe node type label from scope/kind context.
func actionItemNodeLabel(scope domain.KindAppliesTo, kind domain.Kind) string {
	switch domain.NormalizeKindAppliesTo(scope) {
	case domain.KindAppliesToBranch:
		return "Branch"
	case domain.KindAppliesToPhase:
		return "Phase"
	case domain.KindAppliesToSubtask:
		return "Subtask"
	case domain.KindAppliesToActionItem:
		switch strings.TrimSpace(strings.ToLower(string(kind))) {
		case "decision":
			return "Decision"
		case "note":
			return "Note"
		case "branch":
			return "Branch"
		case "phase":
			return "Phase"
		case "subtask":
			return "Subtask"
		default:
			return "ActionItem"
		}
	default:
		return "ActionItem"
	}
}

// actionItemInfoNodeLabel resolves the canonical node label displayed in actionItem-info headers.
func actionItemInfoNodeLabel(actionItem domain.ActionItem) string {
	return actionItemNodeLabel(actionItem.Scope, actionItem.Kind)
}

// actionItemFormNodeLabel resolves node label text for actionItem-form add/edit headers.
func (m Model) actionItemFormNodeLabel() string {
	return actionItemNodeLabel(m.actionItemFormScope, m.actionItemFormKind)
}

// actionItemFormNodeLabelLower renders actionItemFormNodeLabel with lower-case first letter while
// preserving internal camelCase (e.g. "ActionItem" → "actionItem") so prompts and placeholders stay
// consistent with the transport-facing scope vocabulary.
func (m Model) actionItemFormNodeLabelLower() string {
	label := m.actionItemFormNodeLabel()
	if label == "" {
		return ""
	}
	return strings.ToLower(label[:1]) + label[1:]
}

// actionItemFormContextActionItem resolves the existing actionItem being edited, when available.
func (m Model) actionItemFormContextActionItem() (domain.ActionItem, bool) {
	actionItemID := strings.TrimSpace(m.editingActionItemID)
	if actionItemID == "" {
		return domain.ActionItem{}, false
	}
	return m.actionItemByID(actionItemID)
}

// actionItemInfoHeaderMeta renders the compact actionItem metadata line for info-mode headers.
func (m Model) actionItemInfoHeaderMeta(actionItem domain.ActionItem) string {
	state := m.lifecycleStateForActionItem(actionItem)
	return fmt.Sprintf(
		"kind: %s • state: %s • complete: %s • mode: info",
		string(actionItem.Kind),
		lifecycleStateLabel(state),
		completionLabel(state == domain.StateDone),
	)
}

// actionItemFormHeaderMeta renders the compact actionItem metadata line for edit-mode headers.
func (m Model) actionItemFormHeaderMeta() string {
	stateLabel := "-"
	complete := "no"
	if contextActionItem, ok := m.actionItemFormContextActionItem(); ok {
		state := m.lifecycleStateForActionItem(contextActionItem)
		stateLabel = lifecycleStateLabel(state)
		complete = completionLabel(state == domain.StateDone)
	}
	modeLabel := "new"
	if m.mode == modeEditActionItem {
		modeLabel = "edit"
	}
	return fmt.Sprintf("kind: %s • state: %s • complete: %s • mode: %s", string(m.actionItemFormKind), stateLabel, complete, modeLabel)
}

// appendActionItemFormActionRow renders one modal-only action row and tracks focus visibility.
func appendActionItemFormActionRow(lines *[]string, hintStyle, focusStyle lipgloss.Style, field, focusedField int, label, value string, focusLine *int) {
	lineLabel := hintStyle.Render(label + ":")
	if focusedField == field {
		lineLabel = focusStyle.Render(label + ":")
	}
	if strings.TrimSpace(value) == "" {
		value = "-"
	}
	line := lineLabel + " " + value
	if focusedField == field {
		line = markViewportFocus(line)
		if *focusLine < 0 {
			*focusLine = len(*lines)
		}
	}
	*lines = append(*lines, line)
}

// actionItemFormActionFieldSummary returns the rendered summary for one modal-only action row.
func (m Model) actionItemFormActionFieldSummary(field int) string {
	switch field {
	case actionItemFieldDue:
		if field >= 0 && field < len(m.formInputs) {
			return strings.TrimSpace(m.formInputs[field].Value())
		}
	case actionItemFieldLabels:
		if field >= 0 && field < len(m.formInputs) {
			return strings.Join(parseLabelsInput(m.formInputs[field].Value(), nil), ", ")
		}
	case actionItemFieldDependsOn, actionItemFieldBlockedBy:
		if field >= 0 && field < len(m.formInputs) {
			current := parseActionItemRefIDsInput(m.formInputs[field].Value(), nil)
			return m.summarizeActionItemRefs(current, 4)
		}
	}
	return "-"
}

// openFocusedActionItemFormField routes the focused actionItem-form field through its shared action contract.
func (m *Model) openFocusedActionItemFormField(seed tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	if m == nil {
		return Model{}, nil, false
	}
	switch {
	case isActionItemFormMarkdownField(m.formFocus):
		return *m, m.startActionItemFormMarkdownEditor(m.formFocus, seed), true
	case m.formFocus == actionItemFieldDue:
		m.startDuePicker()
		m.status = "due picker"
		return *m, nil, true
	case m.formFocus == actionItemFieldLabels:
		return *m, m.startLabelPicker(), true
	case isActionItemFormDependencyField(m.formFocus):
		return *m, m.startDependencyInspectorFromForm(m.formFocus), true
	case m.formFocus == actionItemFieldComments:
		if actionItem, ok := m.actionItemFormContextActionItem(); ok {
			next, cmd := m.startActionItemThreadWithPanel(actionItem, modeEditActionItem, threadPanelComments)
			if model, ok := next.(Model); ok {
				return model, cmd, true
			}
			return *m, cmd, true
		}
		m.status = "save actionItem first to start thread/comments"
		return *m, nil, true
	case m.formFocus == actionItemFieldSubtasks:
		if _, ok := m.actionItemFormContextActionItem(); !ok {
			m.status = "save actionItem first to add subtasks"
			return *m, nil, true
		}
		return *m, m.openFocusedActionItemFormSubtask(), true
	case m.formFocus == actionItemFieldResources:
		if _, ok := m.actionItemFormContextActionItem(); !ok {
			m.status = "save actionItem first to attach resources"
			return *m, nil, true
		}
		return *m, m.startActionItemFormResourcePickerFromFocus(), true
	default:
		return *m, nil, false
	}
}

// actionItemFormBodyLines renders actionItem add/edit content using the same section structure as actionItem-info.
func (m Model) actionItemFormBodyLines(contentWidth int, hintStyle lipgloss.Style, accent color.Color) ([]string, int) {
	lines := []string{}
	focusLine := -1
	focusStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	contextActionItem, hasContextActionItem := m.actionItemFormContextActionItem()

	setFocus := func() {
		if focusLine >= 0 {
			return
		}
		focusLine = len(lines) - 1
	}

	titleInput := m.formInputs[actionItemFieldTitle]
	titleInput.SetWidth(max(18, contentWidth-8))
	titleLabel := hintStyle.Render("title:")
	if m.formFocus == actionItemFieldTitle {
		titleLabel = focusStyle.Render("title:")
	}
	titleLine := titleLabel + " " + titleInput.View()
	if m.formFocus == actionItemFieldTitle {
		titleLine = markViewportFocus(titleLine)
	}
	lines = append(lines, titleLine)
	if m.formFocus == actionItemFieldTitle {
		setFocus()
	}

	lines = append(lines, "")
	descriptionLabel := hintStyle.Render("description:")
	if m.formFocus == actionItemFieldDescription {
		descriptionLabel = focusStyle.Render("description:")
	}
	if m.formFocus == actionItemFieldDescription {
		descriptionLabel = markViewportFocus(descriptionLabel)
	}
	lines = append(lines, descriptionLabel)
	if m.formFocus == actionItemFieldDescription {
		setFocus()
	}
	descriptionPreview := m.actionItemDescriptionPreviewViewportForContentWidth(m.actionItemFormDescription, contentWidth)
	lines = append(lines, descriptionPreview.View())

	lines = append(lines, "")
	subtasksLabel := hintStyle.Render("subtasks:")
	if m.formFocus == actionItemFieldSubtasks {
		subtasksLabel = focusStyle.Render("subtasks:")
		setFocus()
	}
	lines = append(lines, subtasksLabel)
	activeRowStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	subtasks := m.actionItemFormContextSubtasks()
	done, total := 0, len(subtasks)
	if hasContextActionItem {
		done, total = m.subactionItemProgress(contextActionItem.ID)
	}
	lines = append(lines, hintStyle.Render(fmt.Sprintf("progress: %d/%d done", done, total)))
	selectedSubactionItemRow := clamp(m.actionItemFormSubactionItemCursor, 0, len(subtasks))
	newRow := "  + create new subtask"
	if !hasContextActionItem {
		newRow = "  (save this actionItem before adding subtasks)"
	}
	if m.formFocus == actionItemFieldSubtasks && selectedSubactionItemRow == 0 {
		newRow = markViewportFocus(activeRowStyle.Render("> " + strings.TrimSpace(newRow)))
		focusLine = len(lines)
	}
	lines = append(lines, newRow)
	if len(subtasks) == 0 {
		empty := "  (no subtasks yet)"
		if m.formFocus == actionItemFieldSubtasks && selectedSubactionItemRow == 0 && focusLine < 0 {
			focusLine = len(lines)
		}
		if hasContextActionItem {
			lines = append(lines, hintStyle.Render(empty))
		}
	} else {
		for idx, subtask := range subtasks {
			state := m.lifecycleStateForActionItem(subtask)
			check := "[ ]"
			if state == domain.StateDone {
				check = "[x]"
			}
			line := fmt.Sprintf("  %s %s %s", check, truncate(subtask.Title, 48), hintStyle.Render("state:"+lifecycleStateLabel(state)))
			if m.formFocus == actionItemFieldSubtasks && selectedSubactionItemRow == idx+1 {
				line = markViewportFocus(activeRowStyle.Render("> " + strings.TrimSpace(line)))
				focusLine = len(lines)
			}
			lines = append(lines, line)
		}
	}

	if warning := dueWarning(m.formInputs[actionItemFieldDue].Value(), time.Now().UTC()); warning != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")).Render(warning))
	}

	priorityLabel := hintStyle.Render("priority:")
	if m.formFocus == actionItemFieldPriority {
		priorityLabel = focusStyle.Render("priority:")
	}
	priorityLine := priorityLabel + " " + m.renderPriorityPicker(accent, lipgloss.Color("241"))
	if m.formFocus == actionItemFieldPriority {
		priorityLine = markViewportFocus(priorityLine)
	}
	lines = append(lines, priorityLine)
	if m.formFocus == actionItemFieldPriority {
		setFocus()
	}

	appendActionItemFormActionRow(&lines, hintStyle, focusStyle, actionItemFieldDue, m.formFocus, "due", m.actionItemFormActionFieldSummary(actionItemFieldDue), &focusLine)
	appendActionItemFormActionRow(&lines, hintStyle, focusStyle, actionItemFieldLabels, m.formFocus, "labels", m.actionItemFormActionFieldSummary(actionItemFieldLabels), &focusLine)

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("dependencies:"))
	appendActionItemFormActionRow(&lines, hintStyle, focusStyle, actionItemFieldDependsOn, m.formFocus, "depends_on", m.actionItemFormActionFieldSummary(actionItemFieldDependsOn), &focusLine)
	appendActionItemFormActionRow(&lines, hintStyle, focusStyle, actionItemFieldBlockedBy, m.formFocus, "blocked_by", m.actionItemFormActionFieldSummary(actionItemFieldBlockedBy), &focusLine)
	lines = append(lines, hintStyle.Render("blocked_reason:"))
	if m.formFocus == actionItemFieldBlockedReason {
		lines[len(lines)-1] = markViewportFocus(focusStyle.Render("blocked_reason:"))
		setFocus()
	}
	blockedReason := strings.TrimSpace(m.formInputs[actionItemFieldBlockedReason].Value())
	if blockedReason == "" || blockedReason == "-" {
		lines = append(lines, hintStyle.Render("(none)"))
	} else {
		lines = append(lines, splitThreadMarkdownLines(m.threadMarkdown.render(blockedReason, contentWidth))...)
	}
	lines = append(lines, "")
	commentsLabel := hintStyle.Render(fmt.Sprintf("comments (%d):", len(m.actionItemInfoComments)))
	if m.formFocus == actionItemFieldComments {
		commentsLabel = markViewportFocus(focusStyle.Render(fmt.Sprintf("comments (%d):", len(m.actionItemInfoComments))))
		setFocus()
	}
	lines = append(lines, commentsLabel)
	if !hasContextActionItem {
		lines = append(lines, hintStyle.Render("(save this actionItem before opening comments)"))
	} else if strings.TrimSpace(m.actionItemInfoCommentsError) != "" {
		lines = append(lines, hintStyle.Render("comments unavailable: "+truncate(m.actionItemInfoCommentsError, max(28, contentWidth))))
	} else if len(m.actionItemInfoComments) == 0 {
		lines = append(lines, hintStyle.Render("(no comments yet)"))
	} else {
		for idx := len(m.actionItemInfoComments) - 1; idx >= 0; idx-- {
			comment := m.actionItemInfoComments[idx]
			owner := m.commentOwnerLabel(comment)
			actor := string(normalizeCommentActorType(string(comment.ActorType)))
			lines = append(lines, hintStyle.Render(fmt.Sprintf("[%s] %s • %s", actor, owner, formatThreadTimestamp(comment.CreatedAt))))
			if summary := commentSummaryText(comment); summary != "" {
				lines = append(lines, hintStyle.Render("summary: "+truncate(summary, max(24, contentWidth))))
			}
			body := m.threadMarkdown.render(comment.BodyMarkdown, contentWidth)
			if strings.TrimSpace(body) == "" {
				body = "(empty comment)"
			}
			for _, line := range splitThreadMarkdownLines(body) {
				lines = append(lines, "  "+line)
			}
			if idx > 0 {
				lines = append(lines, "")
			}
		}
	}

	renderMetadataInput := func(label string, field int) {
		lines = append(lines, "")
		header := hintStyle.Render(label + ":")
		if m.formFocus == field {
			header = markViewportFocus(focusStyle.Render(label + ":"))
			setFocus()
		}
		lines = append(lines, header)
		value := strings.TrimSpace(m.formInputs[field].Value())
		if value != "" && value != "-" {
			lines = append(lines, splitThreadMarkdownLines(m.threadMarkdown.render(value, contentWidth))...)
		} else {
			lines = append(lines, hintStyle.Render("(none)"))
		}
	}
	renderMetadataInput("objective", actionItemFieldObjective)
	renderMetadataInput("acceptance_criteria", actionItemFieldAcceptanceCriteria)
	renderMetadataInput("validation_plan", actionItemFieldValidationPlan)
	renderMetadataInput("risk_notes", actionItemFieldRiskNotes)

	lines = append(lines, "")
	resourcesLabel := hintStyle.Render("resources:")
	if m.formFocus == actionItemFieldResources {
		resourcesLabel = focusStyle.Render("resources:")
		setFocus()
	}
	lines = append(lines, resourcesLabel)
	selectedResourceRow := clamp(m.actionItemFormResourceCursor, 0, len(m.actionItemFormResourceRefs))
	newResourceLine := "  + attach new resource"
	if !hasContextActionItem {
		newResourceLine = "  (save this actionItem before attaching resources)"
	}
	if m.formFocus == actionItemFieldResources && selectedResourceRow == 0 {
		newResourceLine = markViewportFocus(activeRowStyle.Render("> " + strings.TrimSpace(newResourceLine)))
		focusLine = len(lines)
	}
	lines = append(lines, newResourceLine)
	if len(m.actionItemFormResourceRefs) == 0 {
		if m.formFocus == actionItemFieldResources && selectedResourceRow == 0 && focusLine < 0 {
			focusLine = len(lines)
		}
		if hasContextActionItem {
			lines = append(lines, hintStyle.Render("  (no resources yet)"))
		}
	} else {
		for idx, ref := range m.actionItemFormResourceRefs {
			location := strings.TrimSpace(ref.Location)
			if ref.PathMode == domain.PathModeRelative && strings.TrimSpace(ref.BaseAlias) != "" {
				location = strings.TrimSpace(ref.BaseAlias) + ":" + location
			}
			line := "  " + fmt.Sprintf("%s %s", ref.ResourceType, truncate(location, 56))
			if m.formFocus == actionItemFieldResources && selectedResourceRow == idx+1 {
				line = markViewportFocus(activeRowStyle.Render("> " + strings.TrimSpace(line)))
				focusLine = len(lines)
			}
			lines = append(lines, line)
		}
	}
	return resolveViewportFocus(lines)
}

// projectFormBodyLines renders project add/edit content using the same full-section modal layout.
func (m Model) projectFormBodyLines(contentWidth int, hintStyle lipgloss.Style, accent color.Color) ([]string, int) {
	lines := []string{}
	focusLine := -1
	focusStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	modeLabel := "new"
	if m.mode == modeEditProject {
		modeLabel = "edit"
	}
	nameInput := m.projectFormInputs[projectFieldName]
	nameInput.SetWidth(max(18, contentWidth-8))
	nameLabel := hintStyle.Render("name:")
	if m.projectFormFocus == projectFieldName {
		nameLabel = focusStyle.Render("name:")
	}
	nameLine := nameLabel + " " + nameInput.View()
	if m.projectFormFocus == projectFieldName {
		nameLine = markViewportFocus(nameLine)
	}
	lines = append(lines, nameLine)
	if m.projectFormFocus == projectFieldName {
		focusLine = len(lines) - 1
	}
	lines = append(lines, hintStyle.Render("mode: "+modeLabel))

	lines = append(lines, "")
	descriptionLabel := hintStyle.Render("description:")
	if m.projectFormFocus == projectFieldDescription {
		descriptionLabel = markViewportFocus(focusStyle.Render("description:"))
		if focusLine < 0 {
			focusLine = len(lines)
		}
	}
	lines = append(lines, descriptionLabel)
	description := strings.TrimSpace(m.projectFormDescription)
	if description == "" {
		lines = append(lines, hintStyle.Render("(no description)"))
	} else {
		lines = append(lines, splitThreadMarkdownLines(m.threadMarkdown.render(description, contentWidth))...)
	}

	renderProjectInput := func(label string, field int) {
		in := m.projectFormInputs[field]
		in.SetWidth(max(18, contentWidth-14))
		lineLabel := hintStyle.Render(label + ":")
		if m.projectFormFocus == field {
			lineLabel = focusStyle.Render(label + ":")
		}
		line := lineLabel + " " + in.View()
		if m.projectFormFocus == field {
			line = markViewportFocus(line)
		}
		lines = append(lines, line)
		if m.projectFormFocus == field {
			focusLine = len(lines) - 1
		}
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("classification"))
	renderProjectInput("kind", projectFieldKind)
	if m.projectFormFocus == projectFieldKind {
		lines = append(lines, hintStyle.Render("enter/e opens picker; type to start a filtered picker"))
	}
	kindRows := m.projectKindSummaryRows(5)
	if len(kindRows) == 0 {
		lines = append(lines, hintStyle.Render("project_kinds: (none available)"))
	} else {
		lines = append(lines, hintStyle.Render("project_kinds:"))
		for _, row := range kindRows {
			lines = append(lines, hintStyle.Render("  - "+row))
		}
		if len(m.projectKindPickerOptions()) > len(kindRows) {
			lines = append(lines, hintStyle.Render(fmt.Sprintf("  +%d more", len(m.projectKindPickerOptions())-len(kindRows))))
		}
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("metadata"))
	renderProjectInput("owner", projectFieldOwner)
	renderProjectInput("icon", projectFieldIcon)
	renderProjectInput("color", projectFieldColor)
	renderProjectInput("homepage", projectFieldHomepage)
	renderProjectInput("tags", projectFieldTags)
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("template workflow"))
	renderProjectInput("template_library", projectFieldTemplateLibrary)
	selectedLibraryID := domain.NormalizeTemplateLibraryID(m.projectFormInputs[projectFieldTemplateLibrary].Value())
	if projectID := strings.TrimSpace(m.editingProjectID); projectID != "" {
		if binding, ok := m.activeProjectTemplateBinding(projectID); ok {
			lines = append(lines, hintStyle.Render("active_binding: "+m.templateBindingSummary(binding)))
			if selectedLibraryID == "" {
				selectedLibraryID = domain.NormalizeTemplateLibraryID(binding.LibraryID)
			}
			if domain.NormalizeTemplateLibraryID(m.projectFormInputs[projectFieldTemplateLibrary].Value()) == domain.NormalizeTemplateLibraryID(binding.LibraryID) &&
				strings.TrimSpace(binding.DriftStatus) == domain.ProjectTemplateBindingDriftUpdateAvailable {
				lines = append(lines, hintStyle.Render("save opens migration review before rebinding future generated work; existing nodes need approve/skip"))
			}
		} else {
			lines = append(lines, hintStyle.Render("active_binding: -"))
		}
	}
	if status, ok := m.builtinTemplateStatus(selectedLibraryID); ok {
		lines = append(lines, hintStyle.Render("shipped_builtin: "+m.builtinTemplateStatusSummary(status)))
		if status.State == domain.BuiltinTemplateLibraryStateUpdateAvailable {
			lines = append(lines, hintStyle.Render("run ensure builtin before rebinding projects to the newer shipped template"))
		}
	}
	if selectedLibraryID == "" {
		lines = append(lines, hintStyle.Render("template_policy: no library selected; the project starts with the broader catalog allowlist"))
	} else {
		lines = append(lines, hintStyle.Render("template_policy: selected library seeds allowed kinds from its node templates and child rules"))
		lines = append(lines, hintStyle.Render("decide with the dev whether any extra generic kinds should be explicitly allowed after setup"))
	}
	if m.projectFormFocus == projectFieldTemplateLibrary {
		lines = append(lines, hintStyle.Render("enter/e opens picker; type to start a filtered picker; choose (none) to clear"))
		lines = append(lines, hintStyle.Render("template-only is the safe default; use kind allowlist controls later to opt generic kinds in intentionally"))
	}
	libraryRows := m.templateLibrarySummaryRows(5)
	if len(libraryRows) == 0 {
		lines = append(lines, hintStyle.Render("approved_global_libraries: (none; create via CLI/MCP first)"))
	} else {
		lines = append(lines, hintStyle.Render("approved_global_libraries:"))
		for _, row := range libraryRows {
			lines = append(lines, hintStyle.Render("  - "+row))
		}
		if len(m.templateLibraries) > len(libraryRows) {
			lines = append(lines, hintStyle.Render(fmt.Sprintf("  +%d more", len(m.templateLibraries)-len(libraryRows))))
		}
	}
	renderProjectInput("root_path", projectFieldRootPath)
	if m.mode == modeEditProject && len(m.projectFormInputs) > projectFieldComments {
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("coordination"))
		renderProjectInput("comments", projectFieldComments)
		if m.projectFormFocus == projectFieldComments {
			lines = append(lines, hintStyle.Render("enter/e opens the project thread on the comments panel"))
		}
	}

	if m.mode == modeEditProject && strings.TrimSpace(m.editingProjectID) != "" {
		for _, project := range m.projects {
			if project.ID != m.editingProjectID {
				continue
			}
			lines = append(lines, "")
			lines = append(lines, hintStyle.Render("system:"))
			lines = append(lines, hintStyle.Render("id: "+project.ID))
			lines = append(lines, hintStyle.Render("slug: "+project.Slug))
			lines = append(lines, hintStyle.Render("kind: "+string(project.Kind)))
			lines = append(lines, hintStyle.Render("created_at: "+project.CreatedAt.In(time.Local).Format(time.RFC3339)))
			lines = append(lines, hintStyle.Render("updated_at: "+project.UpdatedAt.In(time.Local).Format(time.RFC3339)))
			if project.ArchivedAt != nil {
				lines = append(lines, hintStyle.Render("archived_at: "+formatSystemTimestamp(project.ArchivedAt)))
			}
			break
		}
	}

	return resolveViewportFocus(lines)
}

// templateMigrationReviewBodyLines renders the dedicated pre-bind migration-review surface for drifted project templates.
func (m Model) templateMigrationReviewBodyLines(contentWidth int, hintStyle, accentStyle lipgloss.Style) ([]string, int) {
	lines := []string{}
	if m.templateMigrationReviewLoading {
		lines = append(lines,
			accentStyle.Render("loading drift preview"),
			hintStyle.Render("checking project defaults, changed child rules, and eligible generated nodes..."),
		)
		return lines, -1
	}
	preview := m.templateMigrationReviewPreview
	if preview == nil {
		lines = append(lines,
			accentStyle.Render("template reapply review unavailable"),
			hintStyle.Render("press esc to return to project edit"),
		)
		return lines, -1
	}

	lines = append(lines, accentStyle.Render("template drift"))
	lines = append(lines, hintStyle.Render(fmt.Sprintf(
		"library: %s • bound revision: %d • latest revision: %d • drift: %s",
		firstNonEmptyTrimmed(preview.LibraryName, preview.LibraryID, "-"),
		preview.BoundRevision,
		preview.LatestRevision,
		firstNonEmptyTrimmed(preview.DriftStatus, "-"),
	)))
	lines = append(lines, hintStyle.Render("future generated work will adopt the latest approved revision after you continue"))
	lines = append(lines, "")
	lines = append(lines, accentStyle.Render("review summary"))
	lines = append(lines, hintStyle.Render(fmt.Sprintf(
		"default changes: %d • changed child rules: %d • eligible nodes: %d • ineligible nodes: %d",
		len(preview.ProjectDefaultChanges),
		len(preview.ChildRuleChanges),
		preview.EligibleMigrationCount,
		preview.IneligibleMigrationCount,
	)))
	lines = append(lines, hintStyle.Render(fmt.Sprintf("selected for approval: %d", len(m.templateMigrationSelectionIDs()))))

	if len(preview.ProjectDefaultChanges) > 0 {
		lines = append(lines, "")
		lines = append(lines, accentStyle.Render("project default changes"))
		limit := min(5, len(preview.ProjectDefaultChanges))
		for idx := 0; idx < limit; idx++ {
			change := preview.ProjectDefaultChanges[idx]
			lines = append(lines, hintStyle.Render(fmt.Sprintf(
				"- %s: %s -> %s",
				change.Field,
				truncate(firstNonEmptyTrimmed(change.Previous, "-"), max(12, contentWidth/3)),
				truncate(firstNonEmptyTrimmed(change.Current, "-"), max(12, contentWidth/3)),
			)))
		}
		if len(preview.ProjectDefaultChanges) > limit {
			lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more default changes", len(preview.ProjectDefaultChanges)-limit)))
		}
	}

	if len(preview.ChildRuleChanges) > 0 {
		lines = append(lines, "")
		lines = append(lines, accentStyle.Render("changed child rules"))
		limit := min(5, len(preview.ChildRuleChanges))
		for idx := 0; idx < limit; idx++ {
			change := preview.ChildRuleChanges[idx]
			lines = append(lines, hintStyle.Render(fmt.Sprintf(
				"- %s / %s • %s",
				firstNonEmptyTrimmed(change.NodeTemplateName, change.NodeTemplateID, "-"),
				firstNonEmptyTrimmed(change.ChildRuleID, "-"),
				firstNonEmptyTrimmed(strings.Join(change.ChangeKinds, ", "), "details changed"),
			)))
		}
		if len(preview.ChildRuleChanges) > limit {
			lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more changed child rules", len(preview.ChildRuleChanges)-limit)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, accentStyle.Render("existing-node migrations"))
	if len(preview.MigrationCandidates) == 0 {
		lines = append(lines, hintStyle.Render("(no existing generated nodes need migration review)"))
		lines = append(lines, hintStyle.Render("s continues and reapplies only for future generated work"))
		return resolveViewportFocus(lines)
	}

	start, end := windowBounds(len(preview.MigrationCandidates), m.templateMigrationReviewIndex, 7)
	for idx := start; idx < end; idx++ {
		candidate := preview.MigrationCandidates[idx]
		cursor := "  "
		if idx == m.templateMigrationReviewIndex {
			cursor = "> "
		}
		check := "[ ]"
		switch candidate.Status {
		case domain.ProjectTemplateReapplyCandidateEligible:
			if _, ok := m.templateMigrationReviewPicked[strings.TrimSpace(candidate.ActionItemID)]; ok {
				check = "[x]"
			}
		default:
			check = "[-]"
		}
		stateText := lifecycleStateLabel(candidate.LifecycleState)
		if stateText == "-" {
			stateText = firstNonEmptyTrimmed(string(candidate.LifecycleState), "-")
		}
		row := fmt.Sprintf(
			"%s%s %s • %s • %s • %s",
			cursor,
			check,
			truncate(firstNonEmptyTrimmed(candidate.Title, candidate.ActionItemID), max(20, contentWidth-36)),
			firstNonEmptyTrimmed(string(candidate.Scope), "-"),
			firstNonEmptyTrimmed(string(candidate.Kind), "-"),
			stateText,
		)
		if idx == m.templateMigrationReviewIndex {
			row = markViewportFocus(row)
			row = accentStyle.Render(row)
		}
		lines = append(lines, row)
		meta := []string{
			"rule: " + firstNonEmptyTrimmed(candidate.SourceChildRuleID, "-"),
			"changes: " + firstNonEmptyTrimmed(strings.Join(candidate.ChangeKinds, ", "), "-"),
		}
		if candidate.Status != domain.ProjectTemplateReapplyCandidateEligible {
			meta = append(meta, "blocked: "+firstNonEmptyTrimmed(candidate.Reason, "ineligible"))
		}
		lines = append(lines, hintStyle.Render("   "+truncate(strings.Join(meta, " • "), max(24, contentWidth))))
	}
	if len(preview.MigrationCandidates) > end-start {
		lines = append(lines, hintStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(preview.MigrationCandidates))))
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("space/enter toggle • a approve selected • A approve all • s skip existing nodes • esc return to edit"))
	return resolveViewportFocus(lines)
}

// actionItemInfoBodyLines renders reusable actionItem-info sections for the main actionItem-info viewport.
func (m Model) actionItemInfoBodyLines(actionItem domain.ActionItem, boxWidth, contentWidth int, hintStyle lipgloss.Style) []string {
	due := "-"
	if actionItem.DueAt != nil {
		due = formatDueValue(actionItem.DueAt)
	}
	labels := "-"
	if len(actionItem.Labels) > 0 {
		labels = strings.Join(actionItem.Labels, ", ")
	}
	lines := []string{actionItem.Title, ""}
	detailsViewport := m.actionItemInfoDescriptionViewport(actionItem, boxWidth)
	lines = append(lines, hintStyle.Render("description:"))
	lines = append(lines, detailsViewport.View())
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("priority: "+string(actionItem.Priority)))
	lines = append(lines, hintStyle.Render("due: "+due))
	lines = append(lines, hintStyle.Render("labels: "+labels))
	if warning := m.actionItemDueWarning(actionItem, time.Now().UTC()); warning != "" {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")).Render(warning))
	}

	subtasks := m.subtasksForParent(actionItem.ID)
	lines = append(lines, "")
	done, total := m.subactionItemProgress(actionItem.ID)
	lines = append(lines, hintStyle.Render(fmt.Sprintf("subtasks (%d/%d done):", done, total)))
	if len(subtasks) == 0 {
		lines = append(lines, hintStyle.Render("(no subtasks yet)"))
	} else {
		subactionItemIdx := clamp(m.actionItemInfoSubactionItemIdx, 0, len(subtasks)-1)
		for idx, subtask := range subtasks {
			subactionItemState := m.lifecycleStateForActionItem(subtask)
			subactionItemDone := subactionItemState == domain.StateDone
			prefix := "  "
			if idx == subactionItemIdx {
				prefix = "> "
			}
			check := "[ ]"
			if subactionItemDone {
				check = "[x]"
			}
			title := truncate(subtask.Title, 48)
			metaParts := []string{
				"state:" + lifecycleStateLabel(subactionItemState),
				"complete:" + completionLabel(subactionItemDone),
			}
			if subtask.DueAt != nil {
				metaParts = append(metaParts, "due:"+formatDueValue(subtask.DueAt))
			}
			line := fmt.Sprintf("%s%s %s %s", prefix, check, title, hintStyle.Render(strings.Join(metaParts, " • ")))
			lines = append(lines, line)
		}
	}

	dependsOn := uniqueTrimmed(actionItem.Metadata.DependsOn)
	blockedBy := uniqueTrimmed(actionItem.Metadata.BlockedBy)
	blockedReason := strings.TrimSpace(actionItem.Metadata.BlockedReason)
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("dependencies:"))
	lines = append(lines, hintStyle.Render("depends_on: "+m.summarizeActionItemRefs(dependsOn, 4)))
	lines = append(lines, hintStyle.Render("blocked_by: "+m.summarizeActionItemRefs(blockedBy, 4)))
	if blockedReason == "" {
		blockedReason = "-"
	}
	lines = append(lines, hintStyle.Render("blocked_reason: "+blockedReason))

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("template contract:"))
	projectLibraryID := "-"
	if binding, ok := m.activeProjectTemplateBinding(actionItem.ProjectID); ok {
		projectLibraryID = binding.LibraryID
		lines = append(lines, hintStyle.Render("project_library: "+projectLibraryID))
		if binding.BoundRevision > 0 {
			lines = append(lines, hintStyle.Render(fmt.Sprintf("project_library_revision: %d", binding.BoundRevision)))
		}
		if drift := strings.TrimSpace(binding.DriftStatus); drift != "" {
			lines = append(lines, hintStyle.Render("project_library_drift: "+drift))
		}
	} else {
		lines = append(lines, hintStyle.Render("project_library: "+projectLibraryID))
	}
	if snapshot, ok := m.actionItemNodeContracts[actionItem.ID]; ok {
		lines = append(lines, hintStyle.Render("source_library: "+fallbackText(strings.TrimSpace(snapshot.SourceLibraryID), "-")))
		lines = append(lines, hintStyle.Render("source_node_template: "+fallbackText(strings.TrimSpace(snapshot.SourceNodeTemplateID), "-")))
		lines = append(lines, hintStyle.Render("source_child_rule: "+fallbackText(strings.TrimSpace(snapshot.SourceChildRuleID), "-")))
		lines = append(lines, hintStyle.Render("responsible_actor_kind: "+fallbackText(strings.TrimSpace(string(snapshot.ResponsibleActorKind)), "-")))
		lines = append(lines, hintStyle.Render("editable_by: "+templateActorKindsText(snapshot.EditableByActorKinds)))
		lines = append(lines, hintStyle.Render("completable_by: "+templateActorKindsText(snapshot.CompletableByActorKinds)))
		lines = append(lines, hintStyle.Render(fmt.Sprintf("required_for_parent_done: %t", snapshot.RequiredForParentDone)))
		lines = append(lines, hintStyle.Render(fmt.Sprintf("required_for_containing_done: %t", snapshot.RequiredForContainingDone)))
		lines = append(lines, hintStyle.Render("generated_by: "+fallbackText(strings.TrimSpace(snapshot.CreatedByActorID), "-")))
	} else {
		lines = append(lines, hintStyle.Render("generated_contract: none"))
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render(fmt.Sprintf("comments (%d):", len(m.actionItemInfoComments))))
	if strings.TrimSpace(m.actionItemInfoCommentsError) != "" {
		lines = append(lines, hintStyle.Render("comments unavailable: "+truncate(m.actionItemInfoCommentsError, max(28, contentWidth))))
	} else if len(m.actionItemInfoComments) == 0 {
		lines = append(lines, hintStyle.Render("(no comments yet)"))
	} else {
		for idx := len(m.actionItemInfoComments) - 1; idx >= 0; idx-- {
			comment := m.actionItemInfoComments[idx]
			owner := m.commentOwnerLabel(comment)
			actor := string(normalizeCommentActorType(string(comment.ActorType)))
			lines = append(lines, hintStyle.Render(fmt.Sprintf("[%s] %s • %s", actor, owner, formatThreadTimestamp(comment.CreatedAt))))
			if id := strings.TrimSpace(comment.ID); id != "" {
				lines = append(lines, hintStyle.Render("id: "+truncate(id, max(24, contentWidth))))
			}
			if summary := commentSummaryText(comment); summary != "" {
				lines = append(lines, hintStyle.Render("summary: "+truncate(summary, max(24, contentWidth))))
			}
			body := m.threadMarkdown.render(comment.BodyMarkdown, contentWidth)
			if strings.TrimSpace(body) == "" {
				body = "(empty comment)"
			}
			for _, line := range splitThreadMarkdownLines(body) {
				lines = append(lines, "  "+line)
			}
			if idx > 0 {
				lines = append(lines, "")
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("resources:"))
	if len(actionItem.Metadata.ResourceRefs) == 0 {
		lines = append(lines, hintStyle.Render("(none)"))
	} else {
		for idx, ref := range actionItem.Metadata.ResourceRefs {
			if idx >= 4 {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("+%d more", len(actionItem.Metadata.ResourceRefs)-idx)))
				break
			}
			location := strings.TrimSpace(ref.Location)
			if ref.PathMode == domain.PathModeRelative && strings.TrimSpace(ref.BaseAlias) != "" {
				location = strings.TrimSpace(ref.BaseAlias) + ":" + location
			}
			lines = append(lines, hintStyle.Render(fmt.Sprintf("%s %s", ref.ResourceType, truncate(location, 48))))
		}
	}
	renderMetadataMarkdown := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, "", hintStyle.Render(label+":"))
		rendered := m.threadMarkdown.render(value, contentWidth)
		lines = append(lines, splitThreadMarkdownLines(rendered)...)
	}
	renderMetadataMarkdown("objective", actionItem.Metadata.Objective)
	renderMetadataMarkdown("acceptance_criteria", actionItem.Metadata.AcceptanceCriteria)
	renderMetadataMarkdown("validation_plan", actionItem.Metadata.ValidationPlan)
	renderMetadataMarkdown("risk_notes", actionItem.Metadata.RiskNotes)
	if len(actionItem.Metadata.CompletionContract.CompletionCriteria) > 0 {
		unmet := 0
		for _, item := range actionItem.Metadata.CompletionContract.CompletionCriteria {
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

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("system:"))
	lines = append(lines, hintStyle.Render("id: "+actionItem.ID))
	lines = append(lines, hintStyle.Render("project: "+fallbackText(strings.TrimSpace(actionItem.ProjectID), "-")))
	lines = append(lines, hintStyle.Render("parent: "+fallbackText(strings.TrimSpace(actionItem.ParentID), "-")))
	lines = append(lines, hintStyle.Render("kind: "+fallbackText(strings.TrimSpace(string(actionItem.Kind)), "-")))
	lines = append(lines, hintStyle.Render("scope: "+string(actionItem.Scope)))
	lines = append(lines, hintStyle.Render("state: "+fallbackText(strings.TrimSpace(string(actionItem.LifecycleState)), "-")))
	lines = append(lines, hintStyle.Render("column: "+fallbackText(strings.TrimSpace(actionItem.ColumnID), "-")))
	lines = append(lines, hintStyle.Render(fmt.Sprintf("position: %d", actionItem.Position)))
	lines = append(lines, hintStyle.Render("created_at: "+actionItem.CreatedAt.In(time.Local).Format(time.RFC3339)))
	lines = append(lines, hintStyle.Render("updated_at: "+actionItem.UpdatedAt.In(time.Local).Format(time.RFC3339)))
	lines = append(lines, hintStyle.Render(m.actionItemSystemActorLine("created_by", actionItem, actionItem.CreatedByActor, "", true)))
	lines = append(lines, hintStyle.Render(m.actionItemSystemActorLine("updated_by", actionItem, actionItem.UpdatedByActor, actionItem.UpdatedByType, false)))
	if actionItem.StartedAt != nil {
		lines = append(lines, hintStyle.Render("started_at: "+formatSystemTimestamp(actionItem.StartedAt)))
	}
	if actionItem.CompletedAt != nil {
		lines = append(lines, hintStyle.Render("completed_at: "+formatSystemTimestamp(actionItem.CompletedAt)))
	}
	if actionItem.ArchivedAt != nil {
		lines = append(lines, hintStyle.Render("archived_at: "+formatSystemTimestamp(actionItem.ArchivedAt)))
	}
	if actionItem.CanceledAt != nil {
		lines = append(lines, hintStyle.Render("canceled_at: "+formatSystemTimestamp(actionItem.CanceledAt)))
	}
	return lines
}

// syncActionItemInfoBodyViewport refreshes full actionItem-info body viewport dimensions/content after actionItem/size changes.
func (m *Model) syncActionItemInfoBodyViewport(actionItem domain.ActionItem) {
	if m == nil {
		return
	}
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	boxWidth := actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth()))
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, actionItemInfoNodeLabel(actionItem)+" Info", m.actionItemInfoHeaderMeta(actionItem), "")
	prevYOffset := m.actionItemInfoBody.YOffset()
	m.actionItemInfoBody.SetWidth(metrics.contentWidth)
	m.actionItemInfoBody.SetHeight(max(1, metrics.bodyHeight))
	m.actionItemInfoBody.SetContent(strings.Join(m.actionItemInfoBodyLines(actionItem, metrics.boxWidth, metrics.contentWidth, lipgloss.NewStyle()), "\n"))
	m.actionItemInfoBody.SetYOffset(prevYOffset)
}

// trackActionItemInfoPath appends one actionItem id to the modal traversal path, trimming loops when revisiting ancestors.
func (m *Model) trackActionItemInfoPath(actionItemID string) {
	actionItemID = strings.TrimSpace(actionItemID)
	if actionItemID == "" {
		return
	}
	if len(m.actionItemInfoPath) == 0 {
		m.actionItemInfoPath = []string{actionItemID}
		return
	}
	last := strings.TrimSpace(m.actionItemInfoPath[len(m.actionItemInfoPath)-1])
	if last == actionItemID {
		return
	}
	for idx := len(m.actionItemInfoPath) - 2; idx >= 0; idx-- {
		if strings.TrimSpace(m.actionItemInfoPath[idx]) == actionItemID {
			m.actionItemInfoPath = append([]string(nil), m.actionItemInfoPath[:idx+1]...)
			return
		}
	}
	m.actionItemInfoPath = append(m.actionItemInfoPath, actionItemID)
}

// stepBackActionItemInfoPath retraces actionItem-info modal history one step when possible.
func (m *Model) stepBackActionItemInfoPath() bool {
	if len(m.actionItemInfoPath) <= 1 {
		return false
	}
	for len(m.actionItemInfoPath) > 1 {
		m.actionItemInfoPath = append([]string(nil), m.actionItemInfoPath[:len(m.actionItemInfoPath)-1]...)
		prevID := strings.TrimSpace(m.actionItemInfoPath[len(m.actionItemInfoPath)-1])
		if prevID == "" {
			continue
		}
		if _, ok := m.actionItemByID(prevID); !ok {
			continue
		}
		m.actionItemInfoActionItemID = prevID
		m.actionItemInfoSubactionItemIdx = 0
		m.actionItemInfoFocusedSubactionItemID = ""
		m.actionItemInfoDetails.SetYOffset(0)
		m.actionItemInfoBody.SetYOffset(0)
		m.loadActionItemInfoComments(prevID)
		m.status = "actionItem info"
		if actionItem, ok := m.actionItemByID(prevID); ok {
			m.syncActionItemInfoDetailsViewport(actionItem)
			m.syncActionItemInfoBodyViewport(actionItem)
		}
		return true
	}
	return false
}

// stepBackActionItemInfo moves actionItem-info focus to the parent actionItem when available.
func (m *Model) stepBackActionItemInfo(actionItem domain.ActionItem) bool {
	parentID := strings.TrimSpace(actionItem.ParentID)
	if parentID == "" {
		return false
	}
	if _, ok := m.actionItemByID(parentID); !ok {
		return false
	}
	m.actionItemInfoActionItemID = parentID
	m.actionItemInfoSubactionItemIdx = 0
	m.actionItemInfoFocusedSubactionItemID = ""
	m.actionItemInfoDetails.SetYOffset(0)
	m.actionItemInfoBody.SetYOffset(0)
	m.loadActionItemInfoComments(parentID)
	// Keep the cursor aligned to the child we navigated from when it remains visible.
	for idx, child := range m.subtasksForParent(parentID) {
		if child.ID == actionItem.ID {
			m.actionItemInfoSubactionItemIdx = idx
			m.actionItemInfoFocusedSubactionItemID = actionItem.ID
			break
		}
	}
	if parent, ok := m.actionItemByID(parentID); ok {
		m.syncActionItemInfoDetailsViewport(parent)
		m.syncActionItemInfoBodyViewport(parent)
	}
	m.status = "parent actionItem info"
	return true
}

// actionItemIsAncestor reports whether ancestorID is in actionItemID's parent chain (or equal to actionItemID).
func (m Model) actionItemIsAncestor(ancestorID, actionItemID string) bool {
	ancestorID = strings.TrimSpace(ancestorID)
	actionItemID = strings.TrimSpace(actionItemID)
	if ancestorID == "" || actionItemID == "" {
		return false
	}
	visited := map[string]struct{}{}
	currentID := actionItemID
	for currentID != "" {
		if currentID == ancestorID {
			return true
		}
		if _, seen := visited[currentID]; seen {
			return false
		}
		visited[currentID] = struct{}{}
		actionItem, ok := m.actionItemByID(currentID)
		if !ok {
			return false
		}
		currentID = strings.TrimSpace(actionItem.ParentID)
	}
	return false
}

// subtasksForParent returns direct subtask children for a parent actionItem.
func (m Model) subtasksForParent(parentID string) []domain.ActionItem {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil
	}
	out := make([]domain.ActionItem, 0)
	for _, actionItem := range m.tasks {
		if strings.TrimSpace(actionItem.ParentID) != parentID {
			continue
		}
		if actionItem.Kind != domain.KindSubtask {
			continue
		}
		if !m.showArchived && actionItem.ArchivedAt != nil {
			continue
		}
		out = append(out, actionItem)
	}
	sortActionItemSlice(out)
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

// lifecycleStateForActionItem resolves lifecycle state using current board columns with actionItem fallback.
func (m Model) lifecycleStateForActionItem(actionItem domain.ActionItem) domain.LifecycleState {
	if actionItem.LifecycleState != "" {
		return actionItem.LifecycleState
	}
	if state, ok := m.lifecycleStateForColumnID(actionItem.ColumnID); ok && state != "" {
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

// selectedActionItemInfoSubtask returns the focused direct child in actionItem-info mode.
func (m *Model) selectedActionItemInfoSubtask(parent domain.ActionItem) (domain.ActionItem, bool) {
	if m == nil {
		return domain.ActionItem{}, false
	}
	subtasks := m.subtasksForParent(parent.ID)
	if len(subtasks) == 0 {
		m.actionItemInfoFocusedSubactionItemID = ""
		m.actionItemInfoSubactionItemIdx = 0
		return domain.ActionItem{}, false
	}
	focusedID := strings.TrimSpace(m.actionItemInfoFocusedSubactionItemID)
	if focusedID != "" {
		for idx, child := range subtasks {
			if child.ID != focusedID {
				continue
			}
			m.actionItemInfoSubactionItemIdx = idx
			return child, true
		}
	}
	idx := clamp(m.actionItemInfoSubactionItemIdx, 0, len(subtasks)-1)
	m.actionItemInfoSubactionItemIdx = idx
	m.actionItemInfoFocusedSubactionItemID = subtasks[idx].ID
	return subtasks[idx], true
}

// openFocusedActionItemInfoSubtask drills actionItem-info into the currently highlighted child actionItem.
func (m *Model) openFocusedActionItemInfoSubtask(parent domain.ActionItem) tea.Cmd {
	if m == nil {
		return nil
	}
	subtask, ok := m.selectedActionItemInfoSubtask(parent)
	if !ok {
		m.status = "no subtasks"
		return nil
	}
	traceActionItemScreenAction("task_info", "subtask_open", "parent_action_item_id", parent.ID, "child_action_item_id", subtask.ID)
	m.openActionItemInfo(subtask.ID, "actionItem info")
	return nil
}

// reanchorActionItemInfoSubactionItemSelection keeps the actionItem-info subtask highlight on a stable child id.
func (m *Model) reanchorActionItemInfoSubactionItemSelection(parentID string) {
	if m == nil {
		return
	}
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		m.actionItemInfoFocusedSubactionItemID = ""
		m.actionItemInfoSubactionItemIdx = 0
		return
	}
	subtasks := m.subtasksForParent(parentID)
	if len(subtasks) == 0 {
		m.actionItemInfoFocusedSubactionItemID = ""
		m.actionItemInfoSubactionItemIdx = 0
		return
	}
	if focusedID := strings.TrimSpace(m.actionItemInfoFocusedSubactionItemID); focusedID != "" {
		for idx, child := range subtasks {
			if child.ID != focusedID {
				continue
			}
			m.actionItemInfoSubactionItemIdx = idx
			return
		}
	}
	idx := clamp(m.actionItemInfoSubactionItemIdx, 0, len(subtasks)-1)
	m.actionItemInfoSubactionItemIdx = idx
	m.actionItemInfoFocusedSubactionItemID = subtasks[idx].ID
}

// toggleFocusedSubactionItemCompletion toggles done/non-done state for the focused subtask in actionItem-info mode.
func (m Model) toggleFocusedSubactionItemCompletion(parent domain.ActionItem) (tea.Model, tea.Cmd) {
	subtask, ok := (&m).selectedActionItemInfoSubtask(parent)
	if !ok {
		m.status = "no subtasks"
		return m, nil
	}
	traceActionItemScreenAction("task_info", "subtask_toggle", "parent_action_item_id", parent.ID, "child_action_item_id", subtask.ID)
	subactionItemIdx := m.actionItemInfoSubactionItemIdx

	fromIdx, ok := m.columnIndexByID(subtask.ColumnID)
	if !ok {
		m.status = "subtask column unavailable"
		return m, nil
	}

	status := "subtask marked complete"
	toIdx, ok := m.firstColumnIndexForState(domain.StateDone)
	if m.lifecycleStateForActionItem(subtask) == domain.StateDone {
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

	updated, cmd := m.moveActionItemIDs([]string{subtask.ID}, toIdx-fromIdx, "toggle subtask completion", subtask.Title, false)
	next, ok := updated.(Model)
	if !ok {
		return updated, cmd
	}
	next.status = status
	next.mode = modeActionItemInfo
	next.actionItemInfoActionItemID = parent.ID
	next.actionItemInfoSubactionItemIdx = subactionItemIdx
	next.actionItemInfoFocusedSubactionItemID = subtask.ID
	return next, cmd
}

// subactionItemProgress returns completed/total direct subtasks for a parent actionItem.
func (m Model) subactionItemProgress(parentID string) (int, int) {
	subtasks := m.subtasksForParent(parentID)
	if len(subtasks) == 0 {
		return 0, 0
	}
	done := 0
	for _, actionItem := range subtasks {
		if m.lifecycleStateForActionItem(actionItem) == domain.StateDone {
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
	for _, actionItem := range m.tasks {
		if actionItem.ArchivedAt != nil || actionItem.DueAt == nil {
			continue
		}
		due := actionItem.DueAt.UTC()
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

// renderActionItemDetails renders output for the current model state.
func (m Model) renderActionItemDetails(accent, muted, dim color.Color) string {
	actionItem, ok := m.selectedActionItemInCurrentColumn()
	if !ok {
		return ""
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(accent).Render("ActionItem Details"),
		actionItem.Title,
	}

	meta := make([]string, 0, 3)
	meta = append(meta, "state: "+lifecycleStateLabel(m.lifecycleStateForActionItem(actionItem)))
	if m.actionItemFields.ShowPriority {
		meta = append(meta, "priority: "+string(actionItem.Priority))
	}
	if m.actionItemFields.ShowDueDate {
		due := "-"
		if actionItem.DueAt != nil {
			due = formatDueValue(actionItem.DueAt)
		}
		meta = append(meta, "due: "+due)
	}
	if m.actionItemFields.ShowLabels {
		labels := "-"
		if len(actionItem.Labels) > 0 {
			labels = strings.Join(actionItem.Labels, ", ")
		}
		meta = append(meta, "labels: "+labels)
	}
	if len(meta) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render(strings.Join(meta, "  ")))
	}

	if m.actionItemFields.ShowDescription {
		if desc := strings.TrimSpace(actionItem.Description); desc != "" {
			renderWidth := 72
			if m.width > 0 {
				renderWidth = max(24, m.width-8)
			}
			lines = append(lines, splitThreadMarkdownLines(m.threadMarkdown.render(desc, renderWidth))...)
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

// isFullPageNodeMode reports whether the current mode should render as a full-page node surface.
func isFullPageNodeMode(mode inputMode) bool {
	switch mode {
	case modeActionItemInfo, modeAddActionItem, modeEditActionItem, modeAddProject, modeEditProject, modeTemplateMigrationReview:
		return true
	default:
		return false
	}
}

func (m Model) activeBottomHelpKeyMap() staticHelpKeyMap {
	switch m.mode {
	case modeAddActionItem:
		short := []key.Binding{
			helpBinding("enter/e", "field action"),
			helpBinding("ctrl+s", "save"),
			helpBinding("↑/↓", "wrap fields"),
			helpBinding("tab", "next field"),
			helpBinding("esc", "cancel"),
			helpBinding("?", "help"),
		}
		if m.formFocus == actionItemFieldSubtasks || m.formFocus == actionItemFieldResources {
			short = append(short[:3], append([]key.Binding{helpBinding("←/→", "list rows")}, short[3:]...)...)
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeEditActionItem:
		short := []key.Binding{
			helpBinding("enter/e", "field action"),
			helpBinding("ctrl+s", "save"),
			helpBinding("↑/↓", "wrap fields"),
			helpBinding("esc", "cancel"),
			helpBinding("?", "help"),
		}
		if m.formFocus == actionItemFieldSubtasks || m.formFocus == actionItemFieldResources {
			short = append(short[:3], append([]key.Binding{helpBinding("←/→", "list rows")}, short[3:]...)...)
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeAddProject, modeEditProject:
		short := []key.Binding{
			helpBinding("enter", "save"),
			helpBinding("tab", "next field"),
			helpBinding("i", "edit desc"),
			helpBinding("r", "pick path"),
			helpBinding("esc", "cancel"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeTemplateMigrationReview:
		short := []key.Binding{
			helpBinding("space", "toggle"),
			helpBinding("a", "approve sel"),
			helpBinding("A", "approve all"),
			helpBinding("s", "skip"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeActionItemInfo:
		short := []key.Binding{
			helpBinding("d", "details"),
			helpBinding("e", "edit"),
			helpBinding("c", "thread"),
			helpBinding("s", "new subtask"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		full := [][]key.Binding{{
			helpBinding("d", "details"),
			helpBinding("e", "edit"),
			helpBinding("space", "toggle subtask"),
			helpBinding("s", "new subtask"),
			helpBinding("c", "thread"),
			helpBinding("↑/↓", "scroll"),
			helpBinding("pgup/dn", "page"),
			helpBinding("[/]", "move"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}}
		return staticHelpKeyMap{short: short, full: full}
	case modeDescriptionEditor:
		short := []key.Binding{
			helpBinding("tab", "toggle preview"),
			helpBinding("ctrl+s", "save"),
			helpBinding("esc", "cancel"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeSearchResults:
		short := []key.Binding{
			helpBinding("enter", "open"),
			helpBinding("↑/↓", "move"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeEmbeddingsStatus:
		short := []key.Binding{
			helpBinding("enter", "open"),
			helpBinding("/", "filter"),
			helpBinding("r", "reindex"),
			helpBinding("↑/↓", "move"),
			helpBinding("g", "scope"),
			helpBinding("a", "archived"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		full := [][]key.Binding{
			short,
			{
				helpBinding("R", "force"),
			},
		}
		return staticHelpKeyMap{short: short, full: full}
	case modeAuthReview:
		short := []key.Binding{
			helpBinding("enter", "approve/save"),
			helpBinding("d", "deny"),
			helpBinding("s/t/n", "edit"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeAuthScopePicker:
		short := []key.Binding{
			helpBinding("enter", "select"),
			helpBinding("↑/↓", "move"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeAuthInventory:
		short := []key.Binding{
			helpBinding("enter", "review/details"),
			helpBinding("↑/↓", "move"),
			helpBinding("h", "history"),
			helpBinding("g", "scope"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		full := [][]key.Binding{
			short,
			{
				helpBinding("r", "revoke session"),
				helpBinding("pgup/dn", "page"),
				helpBinding("ctrl+u/d", "jump"),
			},
		}
		return staticHelpKeyMap{short: short, full: full}
	case modeAuthSessionRevoke:
		short := []key.Binding{
			helpBinding("enter", "revoke"),
			helpBinding("esc", "cancel"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeCoordinationDetail:
		short := []key.Binding{
			helpBinding("enter", "run/close"),
			helpBinding("↑/↓", "actions"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		full := [][]key.Binding{
			short,
			{
				helpBinding("j/k", "move"),
				helpBinding("tab", "next action"),
				helpBinding("r", "revoke"),
			},
		}
		return staticHelpKeyMap{short: short, full: full}
	case modeThread:
		if m.threadComposerActive {
			short := []key.Binding{
				helpBinding("ctrl+s", "post"),
				helpBinding("enter", "newline"),
				helpBinding("tab/esc", "leave composer"),
				helpBinding("?", "help"),
			}
			return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
		}
		if m.threadPanelFocus == threadPanelContext {
			short := []key.Binding{
				helpBinding("tab/shift+tab", "panels"),
				helpBinding("←/→", "wrap"),
				helpBinding("esc", "back"),
				helpBinding("?", "help"),
			}
			return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
		}
		short := []key.Binding{
			helpBinding("tab/shift+tab", "panels"),
			helpBinding("←/→", "wrap"),
		}
		if m.threadPanelFocus == threadPanelComments {
			short = append(short,
				helpBinding("enter", "comment"),
				helpBinding("i", "compose"),
				helpBinding("↑/↓", "scroll"),
			)
		} else {
			short = append(short, helpBinding("enter", "edit"))
		}
		short = append(short,
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		)
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeDiff:
		short := []key.Binding{
			helpBinding("↑/↓", "scroll"),
			helpBinding("pgup/pgdn", "page"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	case modeFileViewer:
		short := []key.Binding{
			helpBinding("↑/↓", "scroll"),
			helpBinding("pgup/pgdn", "page"),
			helpBinding("esc", "back"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	default:
		if m.mode == modeNone {
			short := []key.Binding{
				helpBinding("n", "new actionItem"),
				helpBinding("i", "actionItem info"),
				helpBinding("e", "edit"),
				helpBinding("tab", "panels"),
				helpBinding("/", "search"),
				helpBinding(":", "commands"),
				helpBinding("q", "quit"),
				helpBinding("?", "help"),
			}
			full := [][]key.Binding{
				short,
				{
					helpBinding("h/l", "columns"),
					helpBinding("j/k", "tasks"),
					helpBinding("[/]", "move"),
					helpBinding("space", "select"),
					helpBinding("f/F", "subtree"),
					helpBinding(":/.", "actions"),
				},
			}
			return staticHelpKeyMap{short: short, full: full}
		}
		short := []key.Binding{
			helpBinding("esc", "close"),
			helpBinding("?", "help"),
		}
		return staticHelpKeyMap{short: short, full: [][]key.Binding{short}}
	}
}

// nodeModalBoxStyle returns the shared full-page node-surface style for info/edit flows.
func nodeModalBoxStyle(accent color.Color, boxWidth int) lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accent)
	if boxWidth > 0 {
		style = style.Width(max(32, boxWidth-style.GetHorizontalFrameSize()))
	}
	return style
}

// buildAutoScrollViewport builds one viewport for full-screen node modals and aligns focused content into view.
func buildAutoScrollViewport(content string, width, height, focusLine int) viewport.Model {
	vp := viewport.New()
	vp.SoftWrap = true
	vp.MouseWheelEnabled = false
	vp.FillHeight = true
	vp.SetWidth(max(1, width))
	vp.SetHeight(max(1, height))
	vp.SetContent(content)
	if focusLine >= 0 {
		totalLines := len(strings.Split(content, "\n"))
		maxOffset := max(0, totalLines-vp.Height())
		target := clamp(focusLine-max(1, vp.Height()/3), 0, maxOffset)
		vp.SetYOffset(target)
	}
	return vp
}

const viewportFocusMarker = "<<tui-focus>>"

// markViewportFocus prefixes one rendered line with an internal marker used to recover the final row offset.
func markViewportFocus(line string) string {
	return viewportFocusMarker + line
}

// resolveViewportFocus strips the internal focus marker and returns the rendered row offset for the marked line.
func resolveViewportFocus(lines []string) ([]string, int) {
	content := strings.Join(lines, "\n")
	idx := strings.Index(content, viewportFocusMarker)
	if idx < 0 {
		return lines, -1
	}
	focusLine := lipgloss.Height(content[:idx])
	content = strings.ReplaceAll(content, viewportFocusMarker, "")
	return strings.Split(content, "\n"), focusLine
}

// wrappedLineOffset returns the wrapped viewport row offset for one raw line index at the requested width.
func wrappedLineOffset(lines []string, rawLine, wrapWidth int) int {
	if rawLine < 0 {
		return -1
	}
	if rawLine == 0 {
		return 0
	}
	prefix := strings.Join(lines[:rawLine], "\n")
	if prefix == "" {
		return 0
	}
	if wrapWidth <= 0 {
		return lipgloss.Height(prefix)
	}
	return lipgloss.Height(lipgloss.NewStyle().Width(wrapWidth).Render(prefix))
}

// renderNodeModalViewport renders the shared bordered body for node full-page surfaces.
func renderNodeModalViewport(accent, muted color.Color, boxWidth int, title, subtitle, status string, body viewport.Model) string {
	return renderFullPageSurfaceViewport(accent, muted, boxWidth, title, subtitle, status, body)
}

// renderFullPageNodeModeView renders actionItem/project info and form modes through one measured full-page surface contract.
func (m Model) renderFullPageNodeModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	hintStyle := lipgloss.NewStyle().Foreground(muted)
	boxWidth := actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth()))

	switch m.mode {
	case modeActionItemInfo:
		actionItem, ok := m.actionItemInfoActionItem()
		if !ok {
			return tea.NewView("")
		}
		metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, actionItemInfoNodeLabel(actionItem)+" Info", m.actionItemInfoHeaderMeta(actionItem), fullPageScrollStatus(m.actionItemInfoBody))
		bodyViewport := m.actionItemInfoBody
		bodyViewport.SetWidth(metrics.contentWidth)
		bodyViewport.SetHeight(max(1, metrics.bodyHeight))
		bodyViewport.SetContent(strings.Join(m.actionItemInfoBodyLines(actionItem, metrics.boxWidth, metrics.contentWidth, hintStyle), "\n"))
		surface := renderNodeModalViewport(accent, muted, metrics.boxWidth, actionItemInfoNodeLabel(actionItem)+" Info", m.actionItemInfoHeaderMeta(actionItem), fullPageScrollStatus(bodyViewport), bodyViewport)
		return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
	case modeAddActionItem, modeEditActionItem:
		title := "New " + m.actionItemFormNodeLabel()
		if m.mode == modeEditActionItem {
			title = "Edit " + m.actionItemFormNodeLabel()
		}
		metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, title, m.actionItemFormHeaderMeta(), fullPageScrollStatus(m.actionItemInfoBody))
		bodyLines, focusLine := m.actionItemFormBodyLines(metrics.contentWidth, hintStyle, accent)
		bodyViewport := m.actionItemInfoBody
		prevYOffset := bodyViewport.YOffset()
		bodyViewport.SetWidth(metrics.contentWidth)
		bodyViewport.SetHeight(max(1, metrics.bodyHeight))
		bodyViewport.SetContent(strings.Join(bodyLines, "\n"))
		bodyViewport.SetYOffset(prevYOffset)
		ensureViewportLineVisible(&bodyViewport, focusLine)
		surface := renderNodeModalViewport(accent, muted, metrics.boxWidth, title, m.actionItemFormHeaderMeta(), fullPageScrollStatus(bodyViewport), bodyViewport)
		return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
	case modeAddProject, modeEditProject:
		title := "New Project"
		if m.mode == modeEditProject {
			title = "Edit Project"
		}
		metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, title, "", fullPageScrollStatus(m.actionItemInfoBody))
		bodyLines, focusLine := m.projectFormBodyLines(metrics.contentWidth, hintStyle, accent)
		bodyViewport := m.actionItemInfoBody
		prevYOffset := bodyViewport.YOffset()
		bodyViewport.SetWidth(metrics.contentWidth)
		bodyViewport.SetHeight(max(1, metrics.bodyHeight))
		bodyViewport.SetContent(strings.Join(bodyLines, "\n"))
		bodyViewport.SetYOffset(prevYOffset)
		ensureViewportLineVisible(&bodyViewport, focusLine)
		surface := renderNodeModalViewport(accent, muted, metrics.boxWidth, title, "", fullPageScrollStatus(bodyViewport), bodyViewport)
		return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
	case modeTemplateMigrationReview:
		metrics := m.fullPageSurfaceMetrics(accent, muted, dim, boxWidth, "Template Migration Review", "review drift before reapply", fullPageScrollStatus(m.actionItemInfoBody))
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		accentStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		bodyLines, focusLine := m.templateMigrationReviewBodyLines(metrics.contentWidth, hintStyle, accentStyle)
		bodyViewport := m.actionItemInfoBody
		prevYOffset := bodyViewport.YOffset()
		bodyViewport.SetWidth(metrics.contentWidth)
		bodyViewport.SetHeight(max(1, metrics.bodyHeight))
		bodyViewport.SetContent(strings.Join(bodyLines, "\n"))
		bodyViewport.SetYOffset(prevYOffset)
		ensureViewportLineVisible(&bodyViewport, focusLine)
		surface := renderNodeModalViewport(accent, muted, metrics.boxWidth, "Template Migration Review", "review drift before reapply", fullPageScrollStatus(bodyViewport), bodyViewport)
		return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
	default:
		return tea.NewView("")
	}
}

// authReviewStageStatus renders compact stage text for the full-screen auth review surface.
func (m Model) authReviewStageStatus() string {
	switch m.authReviewStage {
	case authReviewStageEditTTL:
		return "edit session ttl"
	case authReviewStageEditApproveNote:
		return "edit approval note"
	case authReviewStageDeny:
		return "deny request"
	default:
		return "review pending access request"
	}
}

// authReviewSummaryBody renders the main full-screen auth review body content.
func (m Model) authReviewSummaryBody(contentWidth int, hintStyle, accentStyle lipgloss.Style) string {
	note := strings.TrimSpace(m.confirmAuthNoteInput.Value())
	if note == "" {
		note = "-"
	}
	lines := []string{
		accentStyle.Render("requested access"),
	}
	lines = append(lines, m.authReviewRequestContextLines(contentWidth)...)
	lines = append(lines,
		"",
		accentStyle.Render("approve now"),
		"default decision: approve",
		"[enter] review approval confirmation",
		"[d] deny with note",
		"[esc] cancel review",
		"",
		accentStyle.Render("optional approval changes"),
		fmt.Sprintf("approved scope: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthRequestPathLabel, m.pendingConfirm.AuthRequestPath)),
		fmt.Sprintf("path: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthRequestPath, "-")),
		fmt.Sprintf("session ttl: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthRequestTTL, "-")),
		fmt.Sprintf("note: %s", truncate(note, max(24, contentWidth-8))),
		"",
		"[s] pick approved scope",
		"[t] change session ttl",
		"[n] edit approval note",
	)
	lines = append(lines, "", hintStyle.Render("approve is the default. scope edits open a separate picker with human-readable names."))
	return strings.Join(lines, "\n")
}

// authReviewInputBody renders one auth review editor sub-step body.
func (m Model) authReviewInputBody(contentWidth int, hintStyle, accentStyle lipgloss.Style) string {
	lines := []string{
		accentStyle.Render("request"),
	}
	lines = append(lines, m.authReviewRequestContextLines(contentWidth)...)
	switch m.authReviewStage {
	case authReviewStageEditTTL:
		ttlInput := m.confirmAuthTTLInput
		ttlInput.SetWidth(max(12, contentWidth-12))
		lines = append(lines, "", accentStyle.Render("session ttl"), ttlInput.View())
		lines = append(lines, hintStyle.Render("[save: enter]  [cancel: esc]"))
	case authReviewStageEditApproveNote:
		noteInput := m.confirmAuthNoteInput
		noteInput.SetWidth(max(24, contentWidth-12))
		lines = append(lines, "", accentStyle.Render("approval note"), noteInput.View())
		lines = append(lines, hintStyle.Render("[save: enter]  [cancel: esc]"))
	case authReviewStageDeny:
		noteInput := m.confirmAuthNoteInput
		noteInput.SetWidth(max(24, contentWidth-12))
		lines = append(lines, "", accentStyle.Render("denial note"), noteInput.View())
		lines = append(lines, hintStyle.Render("[next: enter]  [cancel: esc]"))
	}
	lines = append(lines, "", hintStyle.Render("type directly in the active field. press enter to continue or esc to return to review."))
	return strings.Join(lines, "\n")
}

// renderAuthReviewModeView renders the dedicated full-screen auth review surface.
func (m Model) renderAuthReviewModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())), "Access Request Review", m.authReviewStageStatus(), "")
	hintStyle := lipgloss.NewStyle().Foreground(muted)
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	body := m.authReviewSummaryBody(metrics.contentWidth, hintStyle, accentStyle)
	if m.authReviewStage != authReviewStageSummary {
		body = m.authReviewInputBody(metrics.contentWidth, hintStyle, accentStyle)
	}
	surface := renderFullPageSurfaceBody(accent, muted, metrics.boxWidth, "Access Request Review", m.authReviewStageStatus(), "", body)
	return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
}

// renderAuthSessionRevokeModeView renders the dedicated full-screen session
// revoke review surface from coordination.
func (m Model) renderAuthSessionRevokeModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())), "Revoke Active Session", "review session revoke", "")
	hintStyle := lipgloss.NewStyle().Foreground(muted)
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	lines := []string{
		accentStyle.Render("active session"),
		fmt.Sprintf("principal: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthSessionPrincipal, m.pendingConfirm.AuthSessionID)),
		fmt.Sprintf("approved scope: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthSessionPathLabel, "-")),
		fmt.Sprintf("session id: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthSessionID, "-")),
		"",
		accentStyle.Render("revoke now"),
		"[enter] revoke and confirm",
		"[esc] cancel",
		"",
		hintStyle.Render("revoking ends this session immediately and returns to coordination."),
	}
	surface := renderFullPageSurfaceBody(accent, muted, metrics.boxWidth, "Revoke Active Session", "review session revoke", "", strings.Join(lines, "\n"))
	return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
}

// renderAuthScopePickerModeView renders the dedicated full-screen auth scope picker.
func (m Model) renderAuthScopePickerModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	metrics := m.fullPageSurfaceMetrics(accent, muted, dim, actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())), "Pick Approved Scope", "select the scope to approve", "")
	hintStyle := lipgloss.NewStyle().Foreground(muted)
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	lines := []string{}
	if len(m.authReviewScopePickerItems) == 0 {
		lines = append(lines, hintStyle.Render("no selectable scopes are visible in the current project context"))
	} else {
		for idx, item := range m.authReviewScopePickerItems {
			prefix := "  "
			label := item.Label
			if idx == clamp(m.authReviewScopePickerIndex, 0, len(m.authReviewScopePickerItems)-1) {
				prefix = "> "
				label = accentStyle.Render(label)
			}
			lines = append(lines, prefix+label)
			lines = append(lines, hintStyle.Render("    "+firstNonEmptyTrimmed(item.Description, item.Path)))
			lines = append(lines, hintStyle.Render("    path: "+item.Path))
		}
	}
	lines = append(lines, "", hintStyle.Render("enter selects the highlighted scope. esc returns to review."))
	surface := renderFullPageSurfaceBody(accent, muted, metrics.boxWidth, "Pick Approved Scope", "names are primary labels; raw paths stay visible underneath", "", strings.Join(lines, "\n"))
	return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
}

// renderAuthInventoryModeView renders the dedicated full-screen coordination surface.
func (m Model) renderAuthInventoryModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	requestSessionScopeLabel := m.authInventoryScopeLabelForSurface()
	status := m.authInventoryViewLabel()
	if scroll := fullPageScrollStatus(m.authInventoryBody); scroll != "" {
		status += " • " + scroll
	}
	metrics := m.fullPageSurfaceMetrics(
		accent,
		muted,
		dim,
		actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())),
		"Coordination",
		requestSessionScopeLabel,
		status,
	)
	bodyViewport := m.authInventoryBody
	bodyViewport.SetWidth(metrics.contentWidth)
	bodyViewport.SetHeight(max(1, metrics.bodyHeight))
	surface := renderFullPageSurfaceViewport(accent, muted, metrics.boxWidth, "Coordination", requestSessionScopeLabel, status, bodyViewport)
	return m.renderFullPageSurfaceView(accent, muted, dim, metrics, surface)
}

// renderCoordinationDetailModeView renders the coordination surface with one centered typed detail modal overlay.
func (m Model) renderCoordinationDetailModeView() tea.View {
	accent := lipgloss.Color("62")
	if project, ok := m.currentProject(); ok {
		accent = projectAccentColor(project)
	}
	muted := lipgloss.Color("241")
	dim := lipgloss.Color("239")
	requestSessionScopeLabel := m.authInventoryScopeLabelForSurface()
	status := m.authInventoryViewLabel()
	if scroll := fullPageScrollStatus(m.authInventoryBody); scroll != "" {
		status += " • " + scroll
	}
	metrics := m.fullPageSurfaceMetrics(
		accent,
		muted,
		dim,
		actionItemInfoOverlayBoxWidth(max(0, m.fullPageNodeContentWidth())),
		"Coordination",
		requestSessionScopeLabel,
		status,
	)
	bodyViewport := m.authInventoryBody
	bodyViewport.SetWidth(metrics.contentWidth)
	bodyViewport.SetHeight(max(1, metrics.bodyHeight))
	surface := renderFullPageSurfaceViewport(accent, muted, metrics.boxWidth, "Coordination", requestSessionScopeLabel, status, bodyViewport)
	overlay := m.renderModeOverlay(accent, muted, dim, lipgloss.NewStyle().Foreground(muted), m.fullPageNodeContentWidth())
	if m.help.ShowAll {
		overlay = m.renderHelpOverlay(accent, muted, dim, lipgloss.NewStyle().Foreground(muted), m.width-8)
	}
	return m.renderFullPageSurfaceWithOverlay(accent, muted, dim, metrics, surface, overlay)
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
		pathMaxWidth := max(24, m.width-14)
		if maxWidth > 0 {
			boxWidth := clamp(maxWidth, 54, 110)
			style = style.Width(boxWidth)
			pathMaxWidth = max(24, boxWidth-12)
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
		pathLabel = collapsePathForDisplay(pathLabel, pathMaxWidth)
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

	case modeActionItemInfo:
		return ""

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
			filterInput.View(),
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
			filterInput.View(),
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

	case modeProjectKindPicker:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 42, 92))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		filterInput := m.projectKindPickerInput
		filterInput.SetWidth(max(18, min(60, maxWidth-24)))
		currentKindLabel := string(domain.DefaultProjectKind)
		if current := domain.NormalizeKindID(domain.KindID(m.projectFormInputs[projectFieldKind].Value())); current != "" {
			currentKindLabel = m.projectKindDisplayLabel(current, m.projectKindName(current))
		}
		lines := []string{
			titleStyle.Render("Project Kind"),
			filterInput.View(),
			hintStyle.Render("current: " + currentKindLabel),
		}
		if len(m.projectKindPickerItems) == 0 {
			lines = append(lines, hintStyle.Render("(no matching project kinds)"))
		} else {
			start, end := windowBounds(len(m.projectKindPickerItems), m.projectKindPickerIndex, 12)
			for idx := start; idx < end; idx++ {
				item := m.projectKindPickerItems[idx]
				cursor := "  "
				if idx == m.projectKindPickerIndex {
					cursor = "> "
				}
				lines = append(lines, cursor+m.projectKindDisplayLabel(item.KindID, item.DisplayName))
			}
		}
		lines = append(lines, hintStyle.Render("type to filter • j/k navigate • enter choose • ctrl+u clear • esc close"))
		return style.Render(strings.Join(lines, "\n"))

	case modeTemplateLibraryPicker:
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 42, 92))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		filterInput := m.templateLibraryPickerInput
		filterInput.SetWidth(max(18, min(60, maxWidth-24)))
		currentLibraryLabel := "(none)"
		if current := domain.NormalizeTemplateLibraryID(m.projectFormInputs[projectFieldTemplateLibrary].Value()); current != "" {
			currentLibraryLabel = m.templateLibraryDisplayLabel(current, m.templateLibraryName(current))
		}
		lines := []string{
			titleStyle.Render("Template Library"),
			filterInput.View(),
			hintStyle.Render("current: " + currentLibraryLabel),
		}
		if len(m.templateLibraryPickerItems) == 0 {
			lines = append(lines, hintStyle.Render("(no matching approved template libraries)"))
		} else {
			start, end := windowBounds(len(m.templateLibraryPickerItems), m.templateLibraryPickerIndex, 12)
			for idx := start; idx < end; idx++ {
				item := m.templateLibraryPickerItems[idx]
				cursor := "  "
				if idx == m.templateLibraryPickerIndex {
					cursor = "> "
				}
				if item.Clear {
					lines = append(lines, cursor+"(none) remove project template binding")
					continue
				}
				label := m.templateLibraryDisplayLabel(item.LibraryID, item.Name)
				if status, ok := m.builtinTemplateStatus(item.LibraryID); ok && status.State == domain.BuiltinTemplateLibraryStateUpdateAvailable {
					label += " • shipped update available"
				}
				lines = append(lines, cursor+label)
			}
		}
		if len(m.templateLibraryPickerItems) > 0 {
			detailItem := m.templateLibraryPickerItems[clamp(m.templateLibraryPickerIndex, 0, len(m.templateLibraryPickerItems)-1)]
			if detailItem.Clear {
				for _, item := range m.templateLibraryPickerItems {
					if item.Clear {
						continue
					}
					detailItem = item
					break
				}
			}
			if !detailItem.Clear {
				if status, ok := m.builtinTemplateStatus(detailItem.LibraryID); ok {
					lines = append(lines, "")
					lines = append(lines, hintStyle.Render("shipped_builtin: "+m.builtinTemplateStatusSummary(status)))
					if status.State == domain.BuiltinTemplateLibraryStateUpdateAvailable {
						lines = append(lines, hintStyle.Render("run ensure builtin before rebinding projects to the newer shipped template"))
					}
				}
			}
		}
		lines = append(lines, hintStyle.Render("type to filter • j/k navigate • enter choose • ctrl+u clear • esc close"))
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
		if query := strings.TrimSpace(m.searchQuery); query != "" {
			lines = append(lines, hintStyle.Render("query: "+truncate(query, 72)))
		}
		lines = append(lines, hintStyle.Render(searchResultsModeLabel(m.searchRequestedMode, m.searchEffectiveMode, m.searchFallbackReason)))
		if m.searchLoading {
			action := "searching..."
			if m.searchOpeningResult {
				action = "opening search match..."
			}
			lines = append(lines, hintStyle.Render(action))
		} else {
			lines = append(lines, hintStyle.Render(searchResultsEmbeddingSummaryLabel(m.searchEmbeddingSummary)))
		}
		if m.searchLoading {
			lines = append(lines, hintStyle.Render("waiting for search results"))
		} else if len(m.searchMatches) == 0 {
			lines = append(lines, hintStyle.Render("(empty)"))
		} else {
			tasks := make([]domain.ActionItem, 0, len(m.searchMatches))
			for _, match := range m.searchMatches {
				tasks = append(tasks, match.ActionItem)
			}
			levelByActionItemID := m.searchLevelByActionItemID(tasks)
			for idx, match := range m.searchMatches {
				cursor := "  "
				if idx == m.searchResultIndex {
					cursor = "> "
				}
				level := strings.TrimSpace(strings.ToLower(levelByActionItemID[match.ActionItem.ID]))
				levelLabel := canonicalSearchLevelLabels[level]
				if levelLabel == "" {
					levelLabel = "-"
				}
				row := fmt.Sprintf("%s%s • %s • %s • %s • %s", cursor, match.Project.Name, levelLabel, match.StateID, searchMatchEmbeddingLabel(match), truncate(match.ActionItem.Title, 32))
				lines = append(lines, row)
			}
		}
		footer := "j/k navigate • enter open • esc close"
		if m.searchLoading {
			if m.searchOpeningResult {
				footer = "working..."
			} else {
				footer = "esc cancel"
			}
		} else if len(m.searchMatches) == 0 {
			footer = "esc close"
		}
		lines = append(lines, hintStyle.Render(footer))
		return resultsStyle.Render(strings.Join(lines, "\n"))

	case modeEmbeddingsStatus:
		resultsStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)
		if maxWidth > 0 {
			resultsStyle = resultsStyle.Width(clamp(maxWidth, 40, 108))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		lines := []string{titleStyle.Render("Embeddings")}
		if m.svc.EmbeddingsOperational() {
			lines = append(lines, hintStyle.Render("runtime: operational"))
		} else {
			lines = append(lines, hintStyle.Render("runtime: unavailable"))
		}
		scopeLabel := firstNonEmptyTrimmed(m.embeddingsScopeLabel, embeddingsStatusScopeLabel(m.embeddingsProjectIDs, nil))
		lines = append(lines, hintStyle.Render("scope: "+scopeLabel))
		if m.embeddingsIncludeArchived {
			lines = append(lines, hintStyle.Render("archived: shown"))
		} else {
			lines = append(lines, hintStyle.Render("archived: hidden"))
		}
		if m.embeddingsReindexInFlight {
			action := "reindexing embeddings"
			if m.embeddingsReindexForce {
				action = "force reindexing embeddings"
			}
			lines = append(lines, m.embeddingsSpinner.View()+" "+action)
		}
		lines = append(lines, hintStyle.Render(searchResultsEmbeddingSummaryLabel(m.embeddingsSummary)))
		filterLabel := "all"
		if query := strings.TrimSpace(m.embeddingsFilterQuery); query != "" {
			filterLabel = query
		}
		if m.embeddingsFilterActive {
			input := m.embeddingsFilterInput
			input.SetWidth(max(18, maxWidth-24))
			lines = append(lines, input.View())
		} else {
			lines = append(lines, hintStyle.Render("filter: "+filterLabel))
		}
		lines = append(lines, hintStyle.Render(fmt.Sprintf("showing %d of %d rows", len(m.embeddingsDisplayRows), len(m.embeddingsRows))))
		if len(m.embeddingsDisplayRows) == 0 {
			if len(m.embeddingsRows) == 0 {
				lines = append(lines, hintStyle.Render("(no lifecycle rows)"))
			} else {
				lines = append(lines, hintStyle.Render("(no lifecycle rows match the current filter)"))
			}
		} else {
			start, end := windowBounds(len(m.embeddingsDisplayRows), m.embeddingsIndex, 6)
			for idx := start; idx < end; idx++ {
				row := m.embeddingsDisplayRows[idx]
				cursor := "  "
				if idx == m.embeddingsIndex {
					cursor = "> "
				}
				label := fmt.Sprintf(
					"%s%s • %s • %s • %s",
					cursor,
					firstNonEmptyTrimmed(row.ProjectLabel, row.Record.ProjectID),
					firstNonEmptyTrimmed(row.SubjectLabel, string(row.Record.SubjectType)),
					row.Record.Status,
					truncate(firstNonEmptyTrimmed(row.TitleLabel, row.Record.SubjectID), 42),
				)
				lines = append(lines, label)
				lines = append(lines, hintStyle.Render("   path: "+truncate(firstNonEmptyTrimmed(row.PathLabel, row.Record.SubjectID), 78)))
				if detail := strings.TrimSpace(row.DetailLabel); detail != "" && detail != "-" {
					lines = append(lines, hintStyle.Render("   detail: "+truncate(detail, 78)))
				}
			}
		}
		lines = append(lines, hintStyle.Render("j/k move • / filter • enter open • g scope • a archived • r reindex • R force • esc close"))
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
		ownerLabel := "(new actionItem)"
		if actionItem, ok := m.actionItemByID(strings.TrimSpace(m.dependencyOwnerActionItemID)); ok {
			ownerLabel = actionItem.Title
		}
		activeFieldLabel := m.dependencyActiveFieldLabel()
		lines := []string{
			titleStyle.Render("Dependencies & Blockers"),
			hintStyle.Render("actionItem: " + truncate(ownerLabel, 56)),
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
				actionItemID := strings.TrimSpace(candidate.Match.ActionItem.ID)
				cursor := "  "
				if idx == m.dependencyIndex {
					cursor = "> "
				}
				depMark := " "
				if hasDependencyID(m.dependencyDependsOn, actionItemID) {
					depMark = "D"
				}
				blockMark := " "
				if hasDependencyID(m.dependencyBlockedBy, actionItemID) {
					blockMark = "B"
				}
				stateName := candidate.Match.StateID
				if label, ok := canonicalSearchStateLabels[strings.TrimSpace(strings.ToLower(candidate.Match.StateID))]; ok {
					stateName = label
				}
				row := fmt.Sprintf("%s[%s%s] %s • %s", cursor, depMark, blockMark, truncate(candidate.Match.ActionItem.Title, 32), collapsePathForDisplay(candidate.Path, 52))
				if idx == m.dependencyIndex && m.dependencyFocus == 4 {
					row = activeRowStyle.Render(row)
				}
				lines = append(lines, row)
				lines = append(lines, hintStyle.Render("    "+stateName+" • "+string(candidate.Match.ActionItem.Kind)+" • id:"+actionItemID))
			}
			if len(m.dependencyMatches) > dependencyWindowSize {
				lines = append(lines, hintStyle.Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(m.dependencyMatches))))
			}
		}

		if candidate, ok := m.selectedDependencyCandidate(); ok {
			details := candidate.Match.ActionItem
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
			lines = append(lines, hintStyle.Render("path: "+collapsePathForDisplay(candidate.Path, 86)))
			lines = append(lines, hintStyle.Render("state: "+stateText+" • kind: "+string(details.Kind)))
			lines = append(lines, hintStyle.Render("description: "+truncate(description, 86)))
		}

		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("tab focus • j/k list • d toggle depends_on • b toggle blocked_by • space toggle active field value"))
		lines = append(lines, hintStyle.Render("x switch active field • enter jump to actionItem • a apply changes • esc cancel"))
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
		targetTitle := strings.TrimSpace(m.pendingConfirm.ActionItem.Title)
		if len(m.pendingConfirm.ActionItemIDs) > 1 {
			targetTitle = fmt.Sprintf("%d selected tasks", len(m.pendingConfirm.ActionItemIDs))
		}
		if strings.TrimSpace(m.pendingConfirm.Project.ID) != "" {
			targetTitle = strings.TrimSpace(m.pendingConfirm.Project.Name)
			if targetTitle == "" {
				targetTitle = strings.TrimSpace(m.pendingConfirm.Project.ID)
			}
			targetTitle = "project " + targetTitle
		}
		if strings.TrimSpace(m.pendingConfirm.AuthSessionID) != "" {
			targetTitle = firstNonEmptyTrimmed(m.pendingConfirm.AuthSessionPrincipal, m.pendingConfirm.AuthSessionID)
			if scopeLabel := strings.TrimSpace(m.pendingConfirm.AuthSessionPathLabel); scopeLabel != "" {
				targetTitle += " @ " + scopeLabel
			}
		}
		if strings.TrimSpace(m.pendingConfirm.LeaseInstanceID) != "" {
			targetTitle = firstNonEmptyTrimmed(m.pendingConfirm.LeaseAgentName, m.pendingConfirm.LeaseInstanceID)
			if scopeLabel := strings.TrimSpace(m.pendingConfirm.LeaseScopeLabel); scopeLabel != "" {
				targetTitle += " @ " + scopeLabel
			}
		}
		if strings.TrimSpace(m.pendingConfirm.HandoffID) != "" {
			targetTitle = firstNonEmptyTrimmed(m.pendingConfirm.HandoffSummary, m.pendingConfirm.HandoffID)
			if status := strings.TrimSpace(m.pendingConfirm.HandoffStatus); status != "" {
				targetTitle += " → " + status
			}
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
		if strings.TrimSpace(m.pendingConfirm.AuthRequestID) != "" {
			subject := strings.TrimSpace(m.pendingConfirm.AuthRequestPrincipal)
			if subject == "" {
				subject = strings.TrimSpace(m.pendingConfirm.AuthRequestAttention)
			}
			if subject == "" {
				subject = strings.TrimSpace(m.pendingConfirm.AuthRequestID)
			}
			targetTitle = fmt.Sprintf("%s request", subject)
			if path := firstNonEmptyTrimmed(m.pendingConfirm.AuthRequestPathLabel, m.pendingConfirm.AuthRequestPath); path != "" {
				targetTitle += " @ " + path
			}
		}
		lines := []string{
			titleStyle.Render("Confirm Action"),
			fmt.Sprintf("%s: %s", m.pendingConfirm.Label, targetTitle),
		}
		if m.authConfirmFieldsActive() {
			decision := strings.TrimSpace(m.pendingConfirm.AuthRequestDecision)
			note := strings.TrimSpace(m.pendingConfirm.AuthRequestNote)
			if note == "" {
				note = "-"
			}
			lines = append(lines, hintStyle.Render("review this decision before it is applied"))
			lines = append(lines, fmt.Sprintf("decision: %s", firstNonEmptyTrimmed(decision, "-")))
			lines = append(lines, fmt.Sprintf("requested scope: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthRequestRequestedPathLabel, m.pendingConfirm.AuthRequestRequestedPath, "-")))
			lines = append(lines, fmt.Sprintf("approved scope: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthRequestPathLabel, m.pendingConfirm.AuthRequestPath, "-")))
			lines = append(lines, fmt.Sprintf("raw path: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthRequestPath, "-")))
			if strings.EqualFold(decision, "approve") {
				lines = append(lines, fmt.Sprintf("session ttl: %s", firstNonEmptyTrimmed(m.pendingConfirm.AuthRequestTTL, "-")))
			}
			lines = append(lines, fmt.Sprintf("note: %s", truncate(note, max(24, maxWidth-12))))
		}
		lines = append(lines,
			confirmStyle.Render("[confirm]")+"  "+cancelStyle.Render("[cancel]"),
			hintStyle.Render(confirmActionHints(m.authConfirmFieldsActive(), m.authConfirmScopeFieldsActive())),
		)
		return style.Render(strings.Join(lines, "\n"))

	case modeCoordinationDetail:
		toneColor := lipgloss.Color("81")
		switch m.coordinationDetailTone {
		case coordinationDetailToneActive:
			toneColor = lipgloss.Color("81")
		case coordinationDetailToneSuccess:
			toneColor = lipgloss.Color("42")
		case coordinationDetailToneWarn:
			toneColor = lipgloss.Color("214")
		case coordinationDetailToneDanger:
			toneColor = lipgloss.Color("203")
		case coordinationDetailToneMuted:
			toneColor = lipgloss.Color("244")
		}
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(toneColor).
			Padding(0, 1)
		if maxWidth > 0 {
			style = style.Width(clamp(maxWidth, 44, 108))
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(toneColor)
		selectedActionStyle := lipgloss.NewStyle().Bold(true).Foreground(toneColor)
		hintStyle := lipgloss.NewStyle().Foreground(muted)
		title := strings.TrimSpace(m.coordinationDetailTitle)
		if title == "" {
			title = "Coordination Detail"
		}
		body := strings.TrimSpace(m.coordinationDetailBody)
		if body == "" {
			body = "No additional detail is available for this coordination row."
		}
		lines := []string{titleStyle.Render(title), body}
		if len(m.coordinationDetailActions) > 0 {
			lines = append(lines, "", hintStyle.Render("actions"))
			for idx, action := range m.coordinationDetailActions {
				label := action.Label
				prefix := "  "
				if idx == clamp(m.coordinationDetailActionIndex, 0, len(m.coordinationDetailActions)-1) {
					prefix = "> "
					label = selectedActionStyle.Render(label)
				}
				lines = append(lines, prefix+label)
			}
		}
		hint := "enter run/close • j/k select action • esc back"
		for _, action := range m.coordinationDetailActions {
			switch strings.TrimSpace(action.Confirm.Kind) {
			case "revoke-auth-session", "revoke-capability-lease":
				hint = "enter run/close • j/k select action • r revoke • esc back"
			}
		}
		lines = append(lines, "", hintStyle.Render(hint))
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
		lines := []string{titleStyle.Render(m.quickActionsTitle())}
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

	case modeDescriptionEditor:
		return ""

	case modeAddActionItem, modeSearch, modeRenameActionItem, modeEditActionItem, modeAddProject, modeEditProject, modeLabelsConfig, modeHighlightColor:
		title := "Input"
		hint := "enter save • esc cancel • tab next field"
		switch m.mode {
		case modeAddActionItem:
			title = "New " + m.actionItemFormNodeLabel()
			hint = "enter apply field action/save • ctrl+s save • esc cancel • tab next field • enter/e opens field actions"
		case modeSearch:
			title = "Search"
			hint = "tab focus • space/enter toggle • h/l cycle • ctrl+u clear query • ctrl+r reset filters"
		case modeRenameActionItem:
			title = "Rename ActionItem"
		case modeEditActionItem:
			title = "Edit " + m.actionItemFormNodeLabel()
			hint = "enter apply field action/save • ctrl+s save • esc cancel • tab next field • up/down wraps • left/right selects subtask/resource row • enter/e opens field actions"
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

		hintStyle := lipgloss.NewStyle().Foreground(muted)
		isNodeModal := m.mode == modeAddActionItem || m.mode == modeEditActionItem || m.mode == modeAddProject || m.mode == modeEditProject
		if isNodeModal {
			boxWidth := actionItemInfoOverlayBoxWidth(maxWidth)
			contentWidth := max(24, boxWidth-4)
			subtitle := ""
			bodyViewport := m.actionItemInfoBody
			switch m.mode {
			case modeAddActionItem, modeEditActionItem:
				bodyLines, _ := m.actionItemFormBodyLines(contentWidth, hintStyle, accent)
				prevYOffset := bodyViewport.YOffset()
				bodyViewport.SetWidth(contentWidth)
				bodyViewport.SetHeight(max(1, m.fullPageNodeBodyHeight(m.mode == modeEditActionItem)))
				bodyViewport.SetContent(strings.Join(bodyLines, "\n"))
				bodyViewport.SetYOffset(prevYOffset)
				subtitle = m.actionItemFormHeaderMeta()
			case modeAddProject, modeEditProject:
				bodyLines, focusLine := m.projectFormBodyLines(contentWidth, hintStyle, accent)
				bodyViewport = buildAutoScrollViewport(
					strings.Join(bodyLines, "\n"),
					contentWidth,
					max(1, m.fullPageNodeBodyHeight(strings.TrimSpace(subtitle) != "")),
					focusLine,
				)
			}
			return renderNodeModalViewport(accent, muted, boxWidth, title, subtitle, fullPageScrollStatus(bodyViewport), bodyViewport)
		}

		boxWidth := 96
		if maxWidth > 0 {
			boxWidth = clamp(maxWidth, 24, 96)
		}
		contentWidth := max(24, boxWidth-4)
		boxStyle := nodeModalBoxStyle(accent, boxWidth)
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
		lines := []string{titleStyle.Render(title)}

		switch m.mode {
		case modeSearch:
			queryInput := m.searchInput
			queryInput.SetWidth(max(18, contentWidth-12))
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
			modeLabel := lipgloss.NewStyle().Foreground(muted)
			if m.searchFocus == 5 {
				modeLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, modeLabel.Render("mode: "+searchModeDisplayLabel(m.searchMode)))
			applyLabel := hintStyle
			if m.searchFocus == 6 {
				applyLabel = lipgloss.NewStyle().Bold(true).Foreground(accent)
			}
			lines = append(lines, applyLabel.Render("[ apply search ]"))
		case modeLabelsConfig:
			fieldWidth := max(18, contentWidth-20)
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
			in.SetWidth(max(18, contentWidth-10))
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

// formatActionItemEditInput formats values for display or serialization.
func formatActionItemEditInput(actionItem domain.ActionItem) string {
	due := "-"
	if actionItem.DueAt != nil {
		due = formatDueValue(actionItem.DueAt)
	}
	labels := "-"
	if len(actionItem.Labels) > 0 {
		labels = strings.Join(actionItem.Labels, ",")
	}
	return strings.Join([]string{
		actionItem.Title,
		actionItem.Description,
		string(actionItem.Priority),
		due,
		labels,
	}, " | ")
}

// parseActionItemEditInput parses input into a normalized form.
func parseActionItemEditInput(raw string, current domain.ActionItem) (app.UpdateActionItemInput, error) {
	parts := strings.Split(raw, "|")
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	if len(parts) > 5 {
		return app.UpdateActionItemInput{}, fmt.Errorf("expected 5 fields")
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
		return app.UpdateActionItemInput{}, fmt.Errorf("priority must be low|medium|high")
	}

	dueAt, err := parseDueInput(parts[3], current.DueAt)
	if err != nil {
		return app.UpdateActionItemInput{}, err
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

	return app.UpdateActionItemInput{
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
	case modeAddActionItem:
		return "add-actionItem"
	case modeSearch:
		return "search"
	case modeRenameActionItem:
		return "rename"
	case modeEditActionItem:
		return "edit-actionItem"
	case modeDuePicker:
		return "due-picker"
	case modeProjectPicker:
		return "project-picker"
	case modeActionItemInfo:
		return "actionItem-info"
	case modeAddProject:
		return "add-project"
	case modeEditProject:
		return "edit-project"
	case modeSearchResults:
		return "search-results"
	case modeEmbeddingsStatus:
		return "embeddings"
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
	case modeAuthReview:
		return "auth-review"
	case modeAuthScopePicker:
		return "auth-scope-picker"
	case modeAuthInventory:
		return "coordination"
	case modeAuthSessionRevoke:
		return "auth-session-revoke"
	case modeCoordinationDetail:
		return "coordination-detail"
	case modeWarning:
		return "warning"
	case modeResourcePicker:
		return "resources"
	case modeLabelPicker:
		return "labels"
	case modeProjectKindPicker:
		return "project-kinds"
	case modeTemplateLibraryPicker:
		return "template-libraries"
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
	case modeTemplateMigrationReview:
		return "template-migration-review"
	case modeDescriptionEditor:
		return "description-editor"
	case modeThread:
		return "thread"
	case modeDiff:
		return "diff"
	case modeFileViewer:
		return "file-viewer"
	default:
		return "normal"
	}
}

// modePrompt handles mode prompt.
func (m Model) modePrompt() string {
	switch m.mode {
	case modeAddActionItem:
		return "new " + m.actionItemFormNodeLabelLower() + ": enter/e opens field actions, ctrl+s saves, up/down wrap fields, left/right list rows, esc cancels"
	case modeSearch:
		return "search query: " + m.input + " (enter apply, esc cancel)"
	case modeRenameActionItem:
		return "rename actionItem: " + m.input + " (enter save, esc cancel)"
	case modeEditActionItem:
		return "edit " + m.actionItemFormNodeLabelLower() + ": enter/e opens field actions, ctrl+s saves, up/down wrap fields, left/right list rows, esc cancels"
	case modeDuePicker:
		return "due picker: tab focus controls, type date/time in picker, j/k navigate list, enter apply, esc cancel"
	case modeProjectPicker:
		return "project picker: j/k select, enter choose, N new project, A archived toggle, esc cancel"
	case modeActionItemInfo:
		return "actionItem info: enter opens selected subtask, d details preview, arrows or j/k scroll, pgup/pgdown/home/end jump, e edit, s new subtask, c thread, [ / ] move, space toggles subtask complete, backspace parent, esc back"
	case modeAddProject:
		return "new project: enter saves, i edits description, kind/template library open pickers on enter/e/type, r picks root_path, esc cancels"
	case modeEditProject:
		return "edit project: enter saves, i edits description, kind/template library open pickers on enter/e/type, r picks root_path, comments opens thread, esc cancels"
	case modeSearchResults:
		return "search results: j/k select, enter jump, esc close"
	case modeEmbeddingsStatus:
		return "embeddings: j/k move, / filter, enter open, g scope, a archived, r reindex, R force reindex, esc close"
	case modeCommandPalette:
		return "command palette: enter run, esc cancel"
	case modeQuickActions:
		return "quick actions: j/k select, enter run, esc close"
	case modeActivityLog:
		return "activity log: esc close"
	case modeActivityEventInfo:
		return "activity event: enter/g go to node, esc back"
	case modeConfirmAction:
		if m.authConfirmFieldsActive() {
			return "confirm auth decision: enter confirm, esc return to review"
		}
		return "confirm action: enter confirm, esc cancel"
	case modeAuthReview:
		return "auth review: enter opens confirm, d starts deny, s picks scope, t edits ttl, n edits note, esc cancels"
	case modeAuthScopePicker:
		return "auth scope picker: up/down selects named scopes, enter chooses one, esc returns to review"
	case modeAuthInventory:
		return "coordination: up/down select, enter opens review/detail, h toggles live/history, g toggles project/global requests, r opens revoke for selected active session, esc closes"
	case modeAuthSessionRevoke:
		return "revoke active session: enter revoke, esc cancel"
	case modeCoordinationDetail:
		return "coordination detail: enter runs selected action, j/k or tab moves actions, r revokes when available, esc closes"
	case modeWarning:
		return "warning: enter close, esc close"
	case modeResourcePicker:
		return "resource picker: type fuzzy filter, arrows navigate, enter select, ctrl+a choose/attach current, esc cancel"
	case modeLabelPicker:
		return "label picker: type fuzzy filter, j/k select, enter add label, ctrl+u clear, esc cancel"
	case modeProjectKindPicker:
		return "project kind picker: type fuzzy filter, j/k select, enter choose, ctrl+u clear filter, esc cancel"
	case modeTemplateLibraryPicker:
		return "template library picker: type fuzzy filter, j/k select, enter choose, ctrl+u clear filter, esc cancel"
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
	case modeTemplateMigrationReview:
		return "template migration review: space selects, a approves selected, A approves all, s skips existing nodes, esc returns to edit"
	case modeDescriptionEditor:
		return "description editor: tab preview/edit, ctrl+s saves current draft, esc cancel"
	case modeThread:
		return "thread: tab/shift+tab or left/right wrap panels; enter opens the focused panel action; i composes from comments; ctrl+s posts while composing; up/down or pgup/pgdown/home/end scroll comments; esc backs out"
	case modeDiff:
		return "diff: up/down or j/k scroll; pgup/pgdown or ctrl+u/ctrl+d page; esc returns to the prior screen"
	case modeFileViewer:
		return "file-viewer: up/down or j/k scroll; pgup/pgdown page; esc returns to the prior screen"
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
	if actionItem, ok := m.actionItemByID(strings.TrimSpace(entry.ActionItemID)); ok {
		node := fallbackText(strings.TrimSpace(actionItem.Title), "-")
		return node, m.activityActionItemPath(actionItem)
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

// activityActionItemPath builds a project-rooted path label for one actionItem.
func (m Model) activityActionItemPath(actionItem domain.ActionItem) string {
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
	footerLines := m.boardFooterLines()
	h := m.height - headerLines - footerLines
	if h < 10 {
		return 10
	}
	return h
}

// boardFooterLines estimates non-board rows that should remain visible below the board panels.
func (m Model) boardFooterLines() int {
	lines := 0
	if actionItem, ok := m.selectedActionItemInCurrentColumn(); ok {
		if m.directChildCount(actionItem.ID) > 0 || strings.TrimSpace(m.projectionRootActionItemID) != "" {
			lines++
		}
	} else if strings.TrimSpace(m.projectionRootActionItemID) != "" {
		lines++
	}
	if len(m.attentionItems) > 0 {
		lines += 2
	}
	if strings.TrimSpace(m.projectionRootActionItemID) != "" {
		lines++
	}
	if len(m.selectedActionItemIDs) > 0 {
		lines++
	}
	if m.boardStatusText() != "" {
		lines++
	}
	return lines
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
	// Boxed mark with inline path + divider + spacer.
	return lipgloss.Height(headerMarkStyle().Render(headerMarkText)) + 2
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
		// Hard clip overflow rows. Ellipsis markers caused user confusion in tightly-fitted panel layouts.
		lines = lines[:maxLines]
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
