package tui

import (
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/app"
)

// TaskFieldConfig holds configuration for task field.
type TaskFieldConfig struct {
	ShowPriority    bool
	ShowDueDate     bool
	ShowLabels      bool
	ShowDescription bool
}

// SearchConfig holds configuration for search.
type SearchConfig struct {
	CrossProject    bool
	IncludeArchived bool
	States          []string
	Levels          []string
}

// ConfirmConfig holds confirmation behavior flags.
type ConfirmConfig struct {
	Delete     bool
	Archive    bool
	HardDelete bool
	Restore    bool
}

// BoardConfig holds board rendering behavior settings.
type BoardConfig struct {
	ShowWIPWarnings bool
	GroupBy         string
}

// UIConfig holds general UI behavior settings.
type UIConfig struct {
	DueSoonWindows []time.Duration
	ShowDueSummary bool
}

// KeyConfig holds configurable keybinding settings.
type KeyConfig struct {
	CommandPalette string
	QuickActions   string
	MultiSelect    string
	ActivityLog    string
	Undo           string
	Redo           string
}

// IdentityConfig holds identity defaults used for ownership-attributed actions.
type IdentityConfig struct {
	ActorID          string
	DisplayName      string
	DefaultActorType string
}

// LabelConfig holds label suggestion and enforcement settings.
type LabelConfig struct {
	Global         []string
	Projects       map[string][]string
	EnforceAllowed bool
}

// RuntimeConfig holds TUI runtime settings that can be applied live.
type RuntimeConfig struct {
	DefaultDeleteMode app.DeleteMode
	TaskFields        TaskFieldConfig
	Search            SearchConfig
	SearchRoots       []string
	Confirm           ConfirmConfig
	Board             BoardConfig
	UI                UIConfig
	Labels            LabelConfig
	ProjectRoots      map[string]string
	Keys              KeyConfig
	Identity          IdentityConfig
}

// BootstrapConfig holds first-run bootstrap identity and global root settings.
type BootstrapConfig struct {
	ActorID          string
	DisplayName      string
	DefaultActorType string
	SearchRoots      []string
}

// ReloadConfigFunc reloads runtime config values from disk or another source.
type ReloadConfigFunc func() (RuntimeConfig, error)

// SaveProjectRootFunc persists one project-root mapping update.
type SaveProjectRootFunc func(projectSlug, rootPath string) error

// SaveBootstrapConfigFunc persists startup bootstrap identity and global root settings.
type SaveBootstrapConfigFunc func(cfg BootstrapConfig) error

// SaveLabelsConfigFunc persists label defaults for global and current-project scopes.
type SaveLabelsConfigFunc func(projectSlug string, globalLabels, projectLabels []string) error

// Option defines a functional option for model configuration.
type Option func(*Model)

// DefaultTaskFieldConfig returns default task field config.
func DefaultTaskFieldConfig() TaskFieldConfig {
	return TaskFieldConfig{
		ShowPriority:    true,
		ShowDueDate:     true,
		ShowLabels:      true,
		ShowDescription: false,
	}
}

// WithTaskFieldConfig returns an option that sets task field config.
func WithTaskFieldConfig(cfg TaskFieldConfig) Option {
	return func(m *Model) {
		m.taskFields = cfg
	}
}

// WithDefaultDeleteMode returns an option that sets default delete mode.
func WithDefaultDeleteMode(mode app.DeleteMode) Option {
	return func(m *Model) {
		switch mode {
		case app.DeleteModeArchive, app.DeleteModeHard:
			m.defaultDeleteMode = mode
		}
	}
}

// WithSearchConfig returns an option that sets search config.
func WithSearchConfig(cfg SearchConfig) Option {
	return func(m *Model) {
		m.searchDefaultCrossProject = cfg.CrossProject
		m.searchDefaultIncludeArchive = cfg.IncludeArchived
		m.searchCrossProject = cfg.CrossProject
		m.searchIncludeArchived = cfg.IncludeArchived
		if len(cfg.States) > 0 {
			m.searchDefaultStates = canonicalSearchStates(cfg.States)
			m.searchStates = append([]string(nil), m.searchDefaultStates...)
		} else {
			m.searchDefaultStates = []string{"todo", "progress", "done"}
			m.searchStates = append([]string(nil), m.searchDefaultStates...)
		}
		if len(cfg.Levels) > 0 {
			m.searchDefaultLevels = canonicalSearchLevels(cfg.Levels)
			m.searchLevels = append([]string(nil), m.searchDefaultLevels...)
		} else {
			m.searchDefaultLevels = []string{"project", "branch", "phase", "subphase", "task", "subtask"}
			m.searchLevels = append([]string(nil), m.searchDefaultLevels...)
		}
	}
}

// WithSearchRoots returns an option that sets global search-root directories.
func WithSearchRoots(roots []string) Option {
	return func(m *Model) {
		m.searchRoots = normalizeSearchRoots(roots)
	}
}

// WithLaunchProjectPicker returns an option that toggles opening project picker on initial launch.
func WithLaunchProjectPicker(enabled bool) Option {
	return func(m *Model) {
		m.launchPicker = enabled
	}
}

// WithStartupBootstrap returns an option that toggles startup bootstrap gating before project picker.
func WithStartupBootstrap(enabled bool) Option {
	return func(m *Model) {
		m.startupBootstrapRequired = enabled
	}
}

// WithConfirmConfig returns an option that sets confirmation behavior.
func WithConfirmConfig(cfg ConfirmConfig) Option {
	return func(m *Model) {
		m.confirmDelete = cfg.Delete
		m.confirmArchive = cfg.Archive
		m.confirmHardDelete = cfg.HardDelete
		m.confirmRestore = cfg.Restore
	}
}

// WithBoardConfig returns an option that sets board rendering behavior.
func WithBoardConfig(cfg BoardConfig) Option {
	return func(m *Model) {
		m.showWIPWarnings = cfg.ShowWIPWarnings
		switch normalizeBoardGroupBy(cfg.GroupBy) {
		case "priority", "state":
			m.boardGroupBy = normalizeBoardGroupBy(cfg.GroupBy)
		default:
			m.boardGroupBy = "none"
		}
	}
}

// WithUIConfig returns an option that sets UI behavior.
func WithUIConfig(cfg UIConfig) Option {
	return func(m *Model) {
		if len(cfg.DueSoonWindows) > 0 {
			m.dueSoonWindows = append([]time.Duration(nil), cfg.DueSoonWindows...)
		}
		m.showDueSummary = cfg.ShowDueSummary
	}
}

// WithAutoRefreshInterval returns an option that sets periodic board auto-refresh cadence.
func WithAutoRefreshInterval(interval time.Duration) Option {
	return func(m *Model) {
		if interval <= 0 {
			m.autoRefreshInterval = 0
			m.autoRefreshArmed = false
			m.autoRefreshInFlight = false
			return
		}
		m.autoRefreshInterval = interval
	}
}

// WithLabelConfig returns an option that sets label config behavior.
func WithLabelConfig(cfg LabelConfig) Option {
	return func(m *Model) {
		m.allowedLabelGlobal = append([]string(nil), cfg.Global...)
		m.allowedLabelProject = map[string][]string{}
		for project, labels := range cfg.Projects {
			m.allowedLabelProject[project] = append([]string(nil), labels...)
		}
		m.enforceAllowedLabels = cfg.EnforceAllowed
	}
}

// WithProjectRoots returns an option that configures per-project filesystem roots.
func WithProjectRoots(projectRoots map[string]string) Option {
	return func(m *Model) {
		m.projectRoots = map[string]string{}
		for rawSlug, rawPath := range projectRoots {
			slug := strings.TrimSpace(strings.ToLower(rawSlug))
			path := strings.TrimSpace(rawPath)
			if slug == "" || path == "" {
				continue
			}
			m.projectRoots[slug] = path
		}
	}
}

// WithKeyConfig returns an option that configures keybindings.
func WithKeyConfig(cfg KeyConfig) Option {
	return func(m *Model) {
		m.keys.applyConfig(cfg)
	}
}

// WithIdentityConfig returns an option that configures default identity attribution.
func WithIdentityConfig(cfg IdentityConfig) Option {
	return func(m *Model) {
		if actorID := strings.TrimSpace(cfg.ActorID); actorID != "" {
			m.identityActorID = actorID
		}
		m.identityDisplayName = strings.TrimSpace(cfg.DisplayName)
		m.identityDefaultActorType = strings.TrimSpace(strings.ToLower(cfg.DefaultActorType))
	}
}

// WithRuntimeConfig returns an option that applies all runtime-configurable settings.
func WithRuntimeConfig(cfg RuntimeConfig) Option {
	return func(m *Model) {
		WithDefaultDeleteMode(cfg.DefaultDeleteMode)(m)
		WithTaskFieldConfig(cfg.TaskFields)(m)
		WithSearchConfig(cfg.Search)(m)
		WithSearchRoots(cfg.SearchRoots)(m)
		WithConfirmConfig(cfg.Confirm)(m)
		WithBoardConfig(cfg.Board)(m)
		WithUIConfig(cfg.UI)(m)
		WithLabelConfig(cfg.Labels)(m)
		WithProjectRoots(cfg.ProjectRoots)(m)
		WithKeyConfig(cfg.Keys)(m)
		WithIdentityConfig(cfg.Identity)(m)
	}
}

// WithReloadConfigCallback returns an option that sets runtime config reload behavior.
func WithReloadConfigCallback(cb ReloadConfigFunc) Option {
	return func(m *Model) {
		m.reloadConfig = cb
	}
}

// WithSaveProjectRootCallback returns an option that sets root-mapping persistence behavior.
func WithSaveProjectRootCallback(cb SaveProjectRootFunc) Option {
	return func(m *Model) {
		m.saveProjectRoot = cb
	}
}

// WithSaveBootstrapConfigCallback returns an option that sets bootstrap settings persistence behavior.
func WithSaveBootstrapConfigCallback(cb SaveBootstrapConfigFunc) Option {
	return func(m *Model) {
		m.saveBootstrap = cb
	}
}

// WithSaveLabelsConfigCallback returns an option that sets labels-config persistence behavior.
func WithSaveLabelsConfigCallback(cb SaveLabelsConfigFunc) Option {
	return func(m *Model) {
		m.saveLabels = cb
	}
}
