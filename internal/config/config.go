package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// DeleteMode represents a selectable mode.
type DeleteMode string

// DeleteModeArchive and related constants define package defaults.
const (
	DeleteModeArchive DeleteMode = "archive"
	DeleteModeHard    DeleteMode = "hard"
	defaultLogLevel              = "info"
	defaultDevLogDir             = ".tillsyn/log"
	defaultActorType             = "user"
)

// Config holds package configuration.
type Config struct {
	Database         DatabaseConfig         `toml:"database"`
	Delete           DeleteConfig           `toml:"delete"`
	Confirm          ConfirmConfig          `toml:"confirm"`
	ActionItemFields ActionItemFieldsConfig `toml:"task_fields"`
	Board            BoardConfig            `toml:"board"`
	Search           SearchConfig           `toml:"search"`
	Embeddings       EmbeddingsConfig       `toml:"embeddings"`
	Identity         IdentityConfig         `toml:"identity"`
	Paths            PathsConfig            `toml:"paths"`
	UI               UIConfig               `toml:"ui"`
	Logging          LoggingConfig          `toml:"logging"`
	ProjectRoots     map[string]string      `toml:"project_roots"`
	Labels           LabelConfig            `toml:"labels"`
	Keys             KeyConfig              `toml:"keys"`
	TUI              TUIConfig              `toml:"tui"`
}

// DatabaseConfig holds configuration for database.
type DatabaseConfig struct {
	Path string `toml:"path"`
}

// DeleteConfig holds configuration for delete.
type DeleteConfig struct {
	DefaultMode DeleteMode `toml:"default_mode"`
}

// ConfirmConfig holds configuration for confirmation behavior.
type ConfirmConfig struct {
	Delete     bool `toml:"delete"`
	Archive    bool `toml:"archive"`
	HardDelete bool `toml:"hard_delete"`
	Restore    bool `toml:"restore"`
}

// ActionItemFieldsConfig holds configuration for actionItem fields.
type ActionItemFieldsConfig struct {
	ShowPriority    bool `toml:"show_priority"`
	ShowDueDate     bool `toml:"show_due_date"`
	ShowLabels      bool `toml:"show_labels"`
	ShowDescription bool `toml:"show_description"`
}

// BoardConfig holds configuration for board.
type BoardConfig struct {
	ShowWIPWarnings bool   `toml:"show_wip_warnings"`
	GroupBy         string `toml:"group_by"` // none | priority | state
}

// SearchConfig holds configuration for search.
type SearchConfig struct {
	CrossProject    bool     `toml:"cross_project"`
	IncludeArchived bool     `toml:"include_archived"`
	States          []string `toml:"states"`
}

// EmbeddingsConfig holds runtime semantic-search settings.
type EmbeddingsConfig struct {
	Enabled             bool    `toml:"enabled"`
	Provider            string  `toml:"provider"`
	Model               string  `toml:"model"`
	APIKeyEnv           string  `toml:"api_key_env"`
	BaseURL             string  `toml:"base_url"`
	Dimensions          int64   `toml:"dimensions"`
	QueryTopK           int     `toml:"query_top_k"`
	LexicalWeight       float64 `toml:"lexical_weight"`
	SemanticWeight      float64 `toml:"semantic_weight"`
	WorkerPollInterval  string  `toml:"worker_poll_interval"`
	ClaimTTL            string  `toml:"claim_ttl"`
	MaxAttempts         int     `toml:"max_attempts"`
	InitialRetryBackoff string  `toml:"initial_retry_backoff"`
	MaxRetryBackoff     string  `toml:"max_retry_backoff"`
}

// IdentityConfig holds configuration for operator identity defaults.
type IdentityConfig struct {
	ActorID          string `toml:"actor_id"`
	DisplayName      string `toml:"display_name"`
	DefaultActorType string `toml:"default_actor_type"`
}

// PathsConfig holds filesystem root-path configuration.
type PathsConfig struct {
	SearchRoots []string `toml:"search_roots"`
}

// UIConfig holds configuration for UI behavior.
type UIConfig struct {
	DueSoonWindows []string `toml:"due_soon_windows"`
	ShowDueSummary bool     `toml:"show_due_summary"`
}

// LoggingConfig holds runtime logging configuration.
type LoggingConfig struct {
	Level   string               `toml:"level"`
	DevFile LoggingDevFileConfig `toml:"dev_file"`
}

// LoggingDevFileConfig holds development local-file logging controls.
type LoggingDevFileConfig struct {
	Enabled bool   `toml:"enabled"`
	Dir     string `toml:"dir"`
}

// LabelConfig holds label suggestion and enforcement configuration.
type LabelConfig struct {
	Global         []string            `toml:"global"`
	Projects       map[string][]string `toml:"projects"`
	EnforceAllowed bool                `toml:"enforce_allowed"`
}

// KeyConfig holds configuration for key.
type KeyConfig struct {
	CommandPalette string `toml:"command_palette"`
	QuickActions   string `toml:"quick_actions"`
	MultiSelect    string `toml:"multi_select"`
	ActivityLog    string `toml:"activity_log"`
	Undo           string `toml:"undo"`
	Redo           string `toml:"redo"`
}

// TUIConfig holds configuration for TUI-specific surface overrides.
type TUIConfig struct {
	Surfaces TUISurfacesConfig `toml:"surfaces"`
}

// TUISurfacesConfig holds per-surface TUI configuration.
type TUISurfacesConfig struct {
	FileViewer FileViewerConfig `toml:"file_viewer"`
}

// FileViewerConfig holds configuration for the file viewer surface opened by v.
// MaxBytes caps the file size that will be read into memory; files larger than
// this limit render a banner instead of content. DotfileBanner is the exact
// string shown when the active file's basename starts with a dot.
type FileViewerConfig struct {
	MaxBytes      int    `toml:"max_bytes"`
	DotfileBanner string `toml:"dotfile_banner"`
}

// DefaultFileViewerMaxBytes is the default maximum file size (1 MiB) for the file viewer.
const DefaultFileViewerMaxBytes = 1048576

// DefaultFileViewerDotfileBanner is the default banner shown for dotfiles.
const DefaultFileViewerDotfileBanner = "Dotfiles not supported in v1"

// embeddingsFieldPresence captures whether embeddings keys were explicitly present in TOML.
type embeddingsFieldPresence struct {
	Embeddings *embeddingsFieldPresenceSection `toml:"embeddings"`
}

// embeddingsFieldPresenceSection captures explicit embeddings key presence for legacy-default compatibility.
type embeddingsFieldPresenceSection struct {
	Enabled   *bool   `toml:"enabled"`
	Provider  *string `toml:"provider"`
	Model     *string `toml:"model"`
	APIKeyEnv *string `toml:"api_key_env"`
	BaseURL   *string `toml:"base_url"`
}

// Default returns default the requested value.
func Default(dbPath string) Config {
	return Config{
		Database: DatabaseConfig{
			Path: dbPath,
		},
		Delete: DeleteConfig{
			DefaultMode: DeleteModeArchive,
		},
		Confirm: ConfirmConfig{
			Delete:     true,
			Archive:    true,
			HardDelete: true,
			Restore:    false,
		},
		ActionItemFields: ActionItemFieldsConfig{
			ShowPriority:    true,
			ShowDueDate:     true,
			ShowLabels:      true,
			ShowDescription: false,
		},
		Board: BoardConfig{
			ShowWIPWarnings: true,
			GroupBy:         "none",
		},
		Search: SearchConfig{
			CrossProject:    false,
			IncludeArchived: false,
			States:          []string{"todo", "in_progress", "complete"},
		},
		Embeddings: EmbeddingsConfig{
			Enabled:             false,
			Provider:            "ollama",
			Model:               "qwen3-embedding:8b",
			APIKeyEnv:           "",
			BaseURL:             "http://127.0.0.1:11434/v1",
			Dimensions:          0,
			QueryTopK:           200,
			LexicalWeight:       0.55,
			SemanticWeight:      0.45,
			WorkerPollInterval:  "2s",
			ClaimTTL:            "2m",
			MaxAttempts:         5,
			InitialRetryBackoff: "15s",
			MaxRetryBackoff:     "15m",
		},
		Identity: IdentityConfig{
			ActorID:          "",
			DisplayName:      "",
			DefaultActorType: defaultActorType,
		},
		Paths: PathsConfig{
			SearchRoots: []string{},
		},
		UI: UIConfig{
			DueSoonWindows: []string{"24h", "1h"},
			ShowDueSummary: true,
		},
		Logging: LoggingConfig{
			Level: defaultLogLevel,
			DevFile: LoggingDevFileConfig{
				Enabled: true,
				Dir:     defaultDevLogDir,
			},
		},
		ProjectRoots: map[string]string{},
		Labels: LabelConfig{
			Global:         []string{},
			Projects:       map[string][]string{},
			EnforceAllowed: false,
		},
		Keys: KeyConfig{
			CommandPalette: ":",
			QuickActions:   ".",
			MultiSelect:    "space",
			ActivityLog:    "g",
			Undo:           "z",
			Redo:           "Z",
		},
		TUI: TUIConfig{
			Surfaces: TUISurfacesConfig{
				FileViewer: FileViewerConfig{
					MaxBytes:      DefaultFileViewerMaxBytes,
					DotfileBanner: DefaultFileViewerDotfileBanner,
				},
			},
		},
	}
}

// DefaultDevLogDir returns the default logging.dev_file.dir sentinel used by config defaults.
func DefaultDevLogDir() string {
	return defaultDevLogDir
}

// DefaultTemplate renders the default config as TOML for first-run bootstrap and install flows.
func DefaultTemplate() ([]byte, error) {
	encoded, err := toml.Marshal(Default(""))
	if err != nil {
		return nil, fmt.Errorf("encode default config template: %w", err)
	}
	return encoded, nil
}

// Load loads required data for the current operation.
func Load(path string, defaults Config) (Config, error) {
	cfg := defaults
	defaultDBPath := strings.TrimSpace(defaults.Database.Path)
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if len(content) == 0 {
		return cfg, nil
	}

	var presence embeddingsFieldPresence
	if err := toml.Unmarshal(content, &presence); err != nil {
		return Config{}, fmt.Errorf("decode toml presence: %w", err)
	}
	if err := toml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode toml: %w", err)
	}
	// A blank database.path in TOML means "use resolved default path", not
	// "erase the DB path and fail validation".
	if strings.TrimSpace(cfg.Database.Path) == "" {
		cfg.Database.Path = defaultDBPath
	}
	applyLegacyEmbeddingsDefaults(&cfg, presence)
	cfg.normalize()

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate validates the requested operation.
func (c *Config) Validate() error {
	c.Database.Path = strings.TrimSpace(c.Database.Path)
	if c.Database.Path == "" {
		return errors.New("database path is required")
	}

	switch c.Delete.DefaultMode {
	case DeleteModeArchive, DeleteModeHard:
	default:
		return fmt.Errorf("invalid delete.default_mode: %q", c.Delete.DefaultMode)
	}

	switch strings.TrimSpace(strings.ToLower(c.Board.GroupBy)) {
	case "", "none", "priority", "state":
	default:
		return fmt.Errorf("invalid board.group_by: %q", c.Board.GroupBy)
	}
	if c.Embeddings.Dimensions < 0 {
		return fmt.Errorf("embeddings.dimensions must be >= 0")
	}
	if c.Embeddings.QueryTopK < 0 {
		return fmt.Errorf("embeddings.query_top_k must be >= 0")
	}
	if c.Embeddings.LexicalWeight < 0 {
		return fmt.Errorf("embeddings.lexical_weight must be >= 0")
	}
	if c.Embeddings.SemanticWeight < 0 {
		return fmt.Errorf("embeddings.semantic_weight must be >= 0")
	}
	if c.Embeddings.MaxAttempts < 0 {
		return fmt.Errorf("embeddings.max_attempts must be >= 0")
	}
	if c.Embeddings.Enabled {
		switch c.Embeddings.Provider {
		case "openai", "ollama", "deterministic":
		default:
			return fmt.Errorf("embeddings.provider %q is not supported", c.Embeddings.Provider)
		}
		if strings.TrimSpace(c.Embeddings.Model) == "" {
			return errors.New("embeddings.model is required when embeddings are enabled")
		}
		if c.Embeddings.Provider == "openai" && strings.TrimSpace(c.Embeddings.APIKeyEnv) == "" {
			return errors.New("embeddings.api_key_env is required when embeddings are enabled")
		}
	}
	for _, durationField := range []struct {
		name  string
		value string
	}{
		{name: "embeddings.worker_poll_interval", value: c.Embeddings.WorkerPollInterval},
		{name: "embeddings.claim_ttl", value: c.Embeddings.ClaimTTL},
		{name: "embeddings.initial_retry_backoff", value: c.Embeddings.InitialRetryBackoff},
		{name: "embeddings.max_retry_backoff", value: c.Embeddings.MaxRetryBackoff},
	} {
		if strings.TrimSpace(durationField.value) == "" {
			continue
		}
		d, err := time.ParseDuration(strings.TrimSpace(durationField.value))
		if err != nil {
			return fmt.Errorf("%s invalid duration %q", durationField.name, durationField.value)
		}
		if d <= 0 {
			return fmt.Errorf("%s must be > 0", durationField.name)
		}
	}

	for i, state := range c.Search.States {
		if !isKnownLifecycleState(state) {
			return fmt.Errorf("search.states[%d] references unknown state %q", i, state)
		}
	}
	switch c.Identity.DefaultActorType {
	case "user", "agent", "system":
	default:
		return fmt.Errorf("invalid identity.default_actor_type: %q", c.Identity.DefaultActorType)
	}
	for i, searchRoot := range c.Paths.SearchRoots {
		if strings.TrimSpace(searchRoot) == "" {
			return fmt.Errorf("paths.search_roots[%d] is empty", i)
		}
	}

	for i, raw := range c.UI.DueSoonWindows {
		window := strings.TrimSpace(raw)
		if window == "" {
			continue
		}
		d, err := time.ParseDuration(window)
		if err != nil {
			return fmt.Errorf("ui.due_soon_windows[%d] invalid duration %q", i, raw)
		}
		if d <= 0 {
			return fmt.Errorf("ui.due_soon_windows[%d] must be > 0", i)
		}
	}
	c.Logging.Level = strings.TrimSpace(strings.ToLower(c.Logging.Level))
	if c.Logging.Level == "" {
		c.Logging.Level = defaultLogLevel
	}
	switch c.Logging.Level {
	case "debug", "info", "warn", "error", "fatal":
	default:
		return fmt.Errorf("invalid logging.level: %q", c.Logging.Level)
	}
	c.Logging.DevFile.Dir = strings.TrimSpace(c.Logging.DevFile.Dir)
	if c.Logging.DevFile.Dir == "" {
		c.Logging.DevFile.Dir = defaultDevLogDir
	}
	for key, rootPath := range c.ProjectRoots {
		if strings.TrimSpace(key) == "" {
			return errors.New("project_roots contains an empty key")
		}
		if strings.TrimSpace(rootPath) == "" {
			return fmt.Errorf("project_roots.%s path is empty", key)
		}
	}
	for projectSlug, labels := range c.Labels.Projects {
		if strings.TrimSpace(projectSlug) == "" {
			return errors.New("labels.projects contains an empty project key")
		}
		for i, label := range labels {
			if strings.TrimSpace(label) == "" {
				return fmt.Errorf("labels.projects.%s[%d] is empty", projectSlug, i)
			}
		}
	}

	return nil
}

// applyLegacyEmbeddingsDefaults preserves historical implicit OpenAI behavior for older configs that enabled embeddings without naming a provider.
func applyLegacyEmbeddingsDefaults(cfg *Config, presence embeddingsFieldPresence) {
	if cfg == nil || presence.Embeddings == nil || presence.Embeddings.Enabled == nil || !*presence.Embeddings.Enabled {
		return
	}
	if presence.Embeddings.Provider != nil {
		return
	}
	if presence.Embeddings.Model != nil || presence.Embeddings.APIKeyEnv != nil || presence.Embeddings.BaseURL != nil {
		return
	}
	cfg.Embeddings.Provider = "openai"
	cfg.Embeddings.Model = "text-embedding-3-small"
	cfg.Embeddings.APIKeyEnv = "OPENAI_API_KEY"
	cfg.Embeddings.BaseURL = ""
}

// DueSoonDurations handles due soon durations.
func (c Config) DueSoonDurations() []time.Duration {
	out := make([]time.Duration, 0, len(c.UI.DueSoonWindows))
	seen := map[time.Duration]struct{}{}
	for _, raw := range c.UI.DueSoonWindows {
		s := strings.TrimSpace(strings.ToLower(raw))
		if s == "" {
			continue
		}
		d, err := time.ParseDuration(s)
		if err != nil || d <= 0 {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

// AllowedLabels returns normalized allowed label suggestions for a project slug.
func (c Config) AllowedLabels(projectSlug string) []string {
	projectSlug = strings.TrimSpace(strings.ToLower(projectSlug))
	out := make([]string, 0)
	seen := map[string]struct{}{}
	appendUnique := func(values []string) {
		for _, value := range values {
			label := strings.TrimSpace(strings.ToLower(value))
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
	appendUnique(c.Labels.Global)
	if labels, ok := c.Labels.Projects[projectSlug]; ok {
		appendUnique(labels)
	}
	sort.Strings(out)
	return out
}

// normalize canonicalizes config slices/maps after defaults + TOML overlay.
func (c *Config) normalize() {
	states := make([]string, 0, len(c.Search.States))
	seenStates := map[string]struct{}{}
	for _, raw := range c.Search.States {
		state := strings.TrimSpace(strings.ToLower(raw))
		if state == "" {
			continue
		}
		if _, ok := seenStates[state]; ok {
			continue
		}
		seenStates[state] = struct{}{}
		states = append(states, state)
	}
	if len(states) == 0 {
		states = []string{"todo", "in_progress", "complete"}
	}
	c.Search.States = states
	c.Embeddings.Provider = strings.TrimSpace(strings.ToLower(c.Embeddings.Provider))
	if c.Embeddings.Provider == "" {
		c.Embeddings.Provider = "ollama"
	}
	c.Embeddings.Model = strings.TrimSpace(c.Embeddings.Model)
	if c.Embeddings.Model == "" {
		switch c.Embeddings.Provider {
		case "deterministic":
			c.Embeddings.Model = "hash-bow-v1"
		case "openai":
			c.Embeddings.Model = "text-embedding-3-small"
		default:
			c.Embeddings.Model = "qwen3-embedding:8b"
		}
	}
	c.Embeddings.APIKeyEnv = strings.TrimSpace(c.Embeddings.APIKeyEnv)
	if c.Embeddings.APIKeyEnv == "" {
		if c.Embeddings.Provider == "openai" {
			c.Embeddings.APIKeyEnv = "OPENAI_API_KEY"
		}
	}
	c.Embeddings.BaseURL = strings.TrimSpace(c.Embeddings.BaseURL)
	if c.Embeddings.Provider == "ollama" && c.Embeddings.BaseURL == "" {
		c.Embeddings.BaseURL = "http://127.0.0.1:11434/v1"
	}
	if c.Embeddings.Provider == "deterministic" && c.Embeddings.Dimensions == 0 {
		c.Embeddings.Dimensions = 256
	}
	if c.Embeddings.QueryTopK == 0 {
		c.Embeddings.QueryTopK = 200
	}
	if c.Embeddings.LexicalWeight == 0 && c.Embeddings.SemanticWeight == 0 {
		c.Embeddings.LexicalWeight = 0.55
		c.Embeddings.SemanticWeight = 0.45
	}
	c.Embeddings.WorkerPollInterval = strings.TrimSpace(strings.ToLower(c.Embeddings.WorkerPollInterval))
	if c.Embeddings.WorkerPollInterval == "" {
		c.Embeddings.WorkerPollInterval = "2s"
	}
	c.Embeddings.ClaimTTL = strings.TrimSpace(strings.ToLower(c.Embeddings.ClaimTTL))
	if c.Embeddings.ClaimTTL == "" {
		c.Embeddings.ClaimTTL = "2m"
	}
	if c.Embeddings.MaxAttempts == 0 {
		c.Embeddings.MaxAttempts = 5
	}
	c.Embeddings.InitialRetryBackoff = strings.TrimSpace(strings.ToLower(c.Embeddings.InitialRetryBackoff))
	if c.Embeddings.InitialRetryBackoff == "" {
		c.Embeddings.InitialRetryBackoff = "15s"
	}
	c.Embeddings.MaxRetryBackoff = strings.TrimSpace(strings.ToLower(c.Embeddings.MaxRetryBackoff))
	if c.Embeddings.MaxRetryBackoff == "" {
		c.Embeddings.MaxRetryBackoff = "15m"
	}
	c.Identity.ActorID = strings.TrimSpace(c.Identity.ActorID)
	c.Identity.DisplayName = strings.TrimSpace(c.Identity.DisplayName)
	c.Identity.DefaultActorType = normalizeActorType(c.Identity.DefaultActorType)
	c.Paths.SearchRoots = normalizeSearchRoots(c.Paths.SearchRoots)

	windows := make([]string, 0, len(c.UI.DueSoonWindows))
	seenWindows := map[string]struct{}{}
	for _, raw := range c.UI.DueSoonWindows {
		window := strings.TrimSpace(strings.ToLower(raw))
		if window == "" {
			continue
		}
		if _, ok := seenWindows[window]; ok {
			continue
		}
		seenWindows[window] = struct{}{}
		windows = append(windows, window)
	}
	if len(windows) == 0 {
		windows = []string{"24h", "1h"}
	}
	c.UI.DueSoonWindows = windows
	c.Logging.Level = strings.TrimSpace(strings.ToLower(c.Logging.Level))
	if c.Logging.Level == "" {
		c.Logging.Level = defaultLogLevel
	}
	c.Logging.DevFile.Dir = strings.TrimSpace(c.Logging.DevFile.Dir)
	if c.Logging.DevFile.Dir == "" {
		c.Logging.DevFile.Dir = defaultDevLogDir
	}

	roots := make(map[string]string, len(c.ProjectRoots))
	for rawKey, rawPath := range c.ProjectRoots {
		key := strings.TrimSpace(strings.ToLower(rawKey))
		path := strings.TrimSpace(rawPath)
		if key == "" || path == "" {
			continue
		}
		roots[key] = path
	}
	c.ProjectRoots = roots

	c.Labels.Global = normalizeLabelConfigList(c.Labels.Global)
	projectLabels := make(map[string][]string, len(c.Labels.Projects))
	for rawKey, labels := range c.Labels.Projects {
		key := strings.TrimSpace(strings.ToLower(rawKey))
		if key == "" {
			continue
		}
		projectLabels[key] = normalizeLabelConfigList(labels)
	}
	c.Labels.Projects = projectLabels

	c.Keys.CommandPalette = normalizeKeyBinding(c.Keys.CommandPalette, ":")
	c.Keys.QuickActions = normalizeKeyBinding(c.Keys.QuickActions, ".")
	c.Keys.MultiSelect = normalizeKeyBinding(c.Keys.MultiSelect, "space")
	c.Keys.ActivityLog = normalizeKeyBinding(c.Keys.ActivityLog, "g")
	c.Keys.Undo = normalizeKeyBinding(c.Keys.Undo, "z")
	c.Keys.Redo = normalizeKeyBinding(c.Keys.Redo, "Z")

	if c.TUI.Surfaces.FileViewer.MaxBytes <= 0 {
		c.TUI.Surfaces.FileViewer.MaxBytes = DefaultFileViewerMaxBytes
	}
	if c.TUI.Surfaces.FileViewer.DotfileBanner == "" {
		c.TUI.Surfaces.FileViewer.DotfileBanner = DefaultFileViewerDotfileBanner
	}
}

// normalizeLabelConfigList trims, lowercases, and deduplicates label config entries.
func normalizeLabelConfigList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
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
	sort.Strings(out)
	return out
}

// normalizeActorType canonicalizes configured default actor types.
func normalizeActorType(raw string) string {
	actorType := strings.TrimSpace(strings.ToLower(raw))
	if actorType == "" {
		return defaultActorType
	}
	return actorType
}

// normalizeSearchRoots trims, deduplicates, and cleans root path entries.
func normalizeSearchRoots(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		root := strings.TrimSpace(raw)
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	return out
}

// EnsureConfigDir ensures config dir.
func EnsureConfigDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// UpsertIdentity writes identity defaults to the config file.
func UpsertIdentity(path, actorID, displayName, rawActorType string) error {
	configPath := strings.TrimSpace(path)
	if configPath == "" {
		return errors.New("config path is required")
	}
	actorID = strings.TrimSpace(actorID)
	displayName = strings.TrimSpace(displayName)
	actorType := normalizeActorType(rawActorType)
	switch actorType {
	case "user", "agent", "system":
	default:
		return fmt.Errorf("invalid identity.default_actor_type: %q", actorType)
	}

	raw := map[string]any{}
	missing := false
	content, err := os.ReadFile(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read config: %w", err)
		}
		missing = true
	} else if len(content) > 0 {
		if err := toml.Unmarshal(content, &raw); err != nil {
			return fmt.Errorf("decode toml: %w", err)
		}
	}
	if missing && actorID == "" && displayName == "" && actorType == defaultActorType {
		return nil
	}

	identity := map[string]any{}
	if tableValue, ok := raw["identity"]; ok {
		table, ok := tableValue.(map[string]any)
		if !ok {
			return errors.New("identity must be a table")
		}
		for key, value := range table {
			identity[key] = value
		}
	}
	if actorID != "" {
		identity["actor_id"] = actorID
	}

	if displayName == "" {
		delete(identity, "display_name")
	} else {
		identity["display_name"] = displayName
	}
	identity["default_actor_type"] = actorType
	if len(identity) == 0 {
		delete(raw, "identity")
	} else {
		raw["identity"] = identity
	}

	encoded, err := toml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode toml: %w", err)
	}
	if err := EnsureConfigDir(configPath); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}
	if err := os.WriteFile(configPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// UpsertSearchRoots writes paths.search_roots to the config file.
func UpsertSearchRoots(path string, searchRoots []string) error {
	configPath := strings.TrimSpace(path)
	if configPath == "" {
		return errors.New("config path is required")
	}
	searchRoots = normalizeSearchRoots(searchRoots)

	raw := map[string]any{}
	missing := false
	content, err := os.ReadFile(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read config: %w", err)
		}
		missing = true
	} else if len(content) > 0 {
		if err := toml.Unmarshal(content, &raw); err != nil {
			return fmt.Errorf("decode toml: %w", err)
		}
	}
	if missing && len(searchRoots) == 0 {
		return nil
	}

	paths := map[string]any{}
	if tableValue, ok := raw["paths"]; ok {
		table, ok := tableValue.(map[string]any)
		if !ok {
			return errors.New("paths must be a table")
		}
		for key, value := range table {
			paths[key] = value
		}
		if existing, ok := paths["search_roots"]; ok {
			if _, err := decodeTrimmedStringList(existing, "paths.search_roots"); err != nil {
				return err
			}
		}
	}

	if len(searchRoots) == 0 {
		delete(paths, "search_roots")
	} else {
		paths["search_roots"] = append([]string(nil), searchRoots...)
	}
	if len(paths) == 0 {
		delete(raw, "paths")
	} else {
		raw["paths"] = paths
	}

	encoded, err := toml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode toml: %w", err)
	}
	if err := EnsureConfigDir(configPath); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}
	if err := os.WriteFile(configPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// UpsertProjectRoot writes one project_roots mapping update to the config file.
func UpsertProjectRoot(path, projectSlug, rootPath string) error {
	configPath := strings.TrimSpace(path)
	if configPath == "" {
		return errors.New("config path is required")
	}
	slug := strings.TrimSpace(strings.ToLower(projectSlug))
	if slug == "" {
		return errors.New("project slug is required")
	}
	rootPath = strings.TrimSpace(rootPath)

	raw := map[string]any{}
	missing := false
	content, err := os.ReadFile(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read config: %w", err)
		}
		missing = true
	} else if len(content) > 0 {
		if err := toml.Unmarshal(content, &raw); err != nil {
			return fmt.Errorf("decode toml: %w", err)
		}
	}
	if missing && rootPath == "" {
		return nil
	}

	roots := map[string]any{}
	if tableValue, ok := raw["project_roots"]; ok {
		table, ok := tableValue.(map[string]any)
		if !ok {
			return errors.New("project_roots must be a table")
		}
		for rawKey, rawValue := range table {
			key := strings.TrimSpace(strings.ToLower(rawKey))
			if key == "" {
				continue
			}
			pathValue, ok := rawValue.(string)
			if !ok {
				return fmt.Errorf("project_roots.%s must be a string", rawKey)
			}
			pathValue = strings.TrimSpace(pathValue)
			if pathValue == "" {
				continue
			}
			roots[key] = pathValue
		}
	}

	if rootPath == "" {
		delete(roots, slug)
	} else {
		roots[slug] = rootPath
	}
	if len(roots) == 0 {
		delete(raw, "project_roots")
	} else {
		raw["project_roots"] = roots
	}

	encoded, err := toml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode toml: %w", err)
	}
	if err := EnsureConfigDir(configPath); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}
	if err := os.WriteFile(configPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// UpsertAllowedLabels writes global + one per-project label list update to the config file.
func UpsertAllowedLabels(path, projectSlug string, globalLabels, projectLabels []string) error {
	configPath := strings.TrimSpace(path)
	if configPath == "" {
		return errors.New("config path is required")
	}
	slug := strings.TrimSpace(strings.ToLower(projectSlug))
	if slug == "" {
		return errors.New("project slug is required")
	}
	globalLabels = normalizeLabelConfigList(globalLabels)
	projectLabels = normalizeLabelConfigList(projectLabels)

	raw := map[string]any{}
	missing := false
	content, err := os.ReadFile(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read config: %w", err)
		}
		missing = true
	} else if len(content) > 0 {
		if err := toml.Unmarshal(content, &raw); err != nil {
			return fmt.Errorf("decode toml: %w", err)
		}
	}
	if missing && len(globalLabels) == 0 && len(projectLabels) == 0 {
		return nil
	}

	labels := map[string]any{}
	if labelsRaw, ok := raw["labels"]; ok {
		table, ok := labelsRaw.(map[string]any)
		if !ok {
			return errors.New("labels must be a table")
		}
		for k, v := range table {
			labels[k] = v
		}
	}

	projects := map[string]any{}
	if projectsRaw, ok := labels["projects"]; ok {
		table, ok := projectsRaw.(map[string]any)
		if !ok {
			return errors.New("labels.projects must be a table")
		}
		for rawKey, rawValue := range table {
			key := strings.TrimSpace(strings.ToLower(rawKey))
			if key == "" {
				continue
			}
			list, err := decodeStringList(rawValue, "labels.projects."+rawKey)
			if err != nil {
				return err
			}
			if len(list) == 0 {
				continue
			}
			projects[key] = list
		}
	}

	if len(globalLabels) == 0 {
		delete(labels, "global")
	} else {
		labels["global"] = append([]string(nil), globalLabels...)
	}
	if len(projectLabels) == 0 {
		delete(projects, slug)
	} else {
		projects[slug] = append([]string(nil), projectLabels...)
	}
	if len(projects) == 0 {
		delete(labels, "projects")
	} else {
		labels["projects"] = projects
	}
	if len(labels) == 0 {
		delete(raw, "labels")
	} else {
		raw["labels"] = labels
	}

	encoded, err := toml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode toml: %w", err)
	}
	if err := EnsureConfigDir(configPath); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}
	if err := os.WriteFile(configPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// decodeStringList coerces TOML list values into normalized string slices.
func decodeStringList(value any, field string) ([]string, error) {
	switch list := value.(type) {
	case []string:
		return normalizeLabelConfigList(list), nil
	case []any:
		out := make([]string, 0, len(list))
		for idx, raw := range list {
			text, ok := raw.(string)
			if !ok {
				return nil, fmt.Errorf("%s[%d] must be a string", field, idx)
			}
			out = append(out, text)
		}
		return normalizeLabelConfigList(out), nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings", field)
	}
}

// decodeTrimmedStringList coerces TOML list values into trimmed string slices.
func decodeTrimmedStringList(value any, field string) ([]string, error) {
	switch list := value.(type) {
	case []string:
		out := make([]string, 0, len(list))
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			out = append(out, item)
		}
		return out, nil
	case []any:
		out := make([]string, 0, len(list))
		for idx, raw := range list {
			text, ok := raw.(string)
			if !ok {
				return nil, fmt.Errorf("%s[%d] must be a string", field, idx)
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			out = append(out, text)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings", field)
	}
}

// isKnownLifecycleState reports whether the requested state is canonical.
// Strict-canonical: legacy aliases (done, completed, progress, in-progress, doing)
// are not accepted — pre-MVP every caller is the dev.
func isKnownLifecycleState(state string) bool {
	return slices.Contains([]string{"todo", "in_progress", "complete", "failed", "archived"}, state)
}

// normalizeKeyBinding trims keybinding text and applies fallback defaults.
func normalizeKeyBinding(raw, fallback string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	return value
}
