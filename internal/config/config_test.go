package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDefaultConfig verifies behavior for the covered scenario.
func TestDefaultConfig(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	if cfg.Database.Path != "/tmp/tillsyn.db" {
		t.Fatalf("unexpected db path %q", cfg.Database.Path)
	}
	if cfg.Delete.DefaultMode != DeleteModeArchive {
		t.Fatalf("unexpected delete mode %q", cfg.Delete.DefaultMode)
	}
	if !cfg.Confirm.Delete || !cfg.Confirm.Archive || !cfg.Confirm.HardDelete {
		t.Fatalf("unexpected confirm defaults %#v", cfg.Confirm)
	}
	if cfg.Confirm.Restore {
		t.Fatalf("expected restore confirm disabled by default, got %#v", cfg.Confirm)
	}
	if !cfg.TaskFields.ShowPriority || !cfg.TaskFields.ShowDueDate || !cfg.TaskFields.ShowLabels {
		t.Fatal("expected priority/due_date/labels enabled by default")
	}
	if cfg.TaskFields.ShowDescription {
		t.Fatal("expected description disabled by default")
	}
	if got := cfg.UI.DueSoonWindows; len(got) != 2 || got[0] != "24h" || got[1] != "1h" {
		t.Fatalf("unexpected due windows %#v", got)
	}
	if !cfg.UI.ShowDueSummary {
		t.Fatal("expected due summary enabled by default")
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("expected default logging level info, got %q", cfg.Logging.Level)
	}
	if !cfg.Logging.DevFile.Enabled {
		t.Fatal("expected dev file logging enabled by default")
	}
	if cfg.Logging.DevFile.Dir != ".tillsyn/log" {
		t.Fatalf("expected default dev file log dir .tillsyn/log, got %q", cfg.Logging.DevFile.Dir)
	}
	if cfg.Identity.DisplayName != "" {
		t.Fatalf("expected empty default identity display name, got %q", cfg.Identity.DisplayName)
	}
	if cfg.Identity.ActorID != "" {
		t.Fatalf("expected empty default identity actor_id, got %q", cfg.Identity.ActorID)
	}
	if cfg.Identity.DefaultActorType != "user" {
		t.Fatalf("expected default identity actor type user, got %q", cfg.Identity.DefaultActorType)
	}
	if len(cfg.Paths.SearchRoots) != 0 {
		t.Fatalf("expected no default search roots, got %#v", cfg.Paths.SearchRoots)
	}
	if cfg.Embeddings.Enabled {
		t.Fatal("expected embeddings disabled by default")
	}
	if cfg.Embeddings.Provider != "ollama" {
		t.Fatalf("expected default embeddings provider ollama, got %q", cfg.Embeddings.Provider)
	}
	if cfg.Embeddings.Model != "qwen3-embedding:8b" {
		t.Fatalf("expected default embeddings model qwen3-embedding:8b, got %q", cfg.Embeddings.Model)
	}
	if cfg.Embeddings.APIKeyEnv != "" {
		t.Fatalf("expected default embeddings api_key_env empty for ollama, got %q", cfg.Embeddings.APIKeyEnv)
	}
	if cfg.Embeddings.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("expected default embeddings base_url http://127.0.0.1:11434/v1, got %q", cfg.Embeddings.BaseURL)
	}
}

// TestLoadMissingFileUsesDefaults verifies behavior for the covered scenario.
func TestLoadMissingFileUsesDefaults(t *testing.T) {
	defaults := Default("/tmp/tillsyn.db")
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"), defaults)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Path != defaults.Database.Path {
		t.Fatalf("expected default db path, got %q", cfg.Database.Path)
	}
}

// TestLoadFileOverridesDefaults verifies behavior for the covered scenario.
func TestLoadFileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/tillsyn.db"

[delete]
default_mode = "hard"

[confirm]
delete = true
archive = false
hard_delete = true
restore = true

[task_fields]
show_priority = true
show_due_date = false
show_labels = true
show_description = true

[ui]
due_soon_windows = ["12h", "45m"]
show_due_summary = false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Path != "/custom/tillsyn.db" {
		t.Fatalf("unexpected db path %q", cfg.Database.Path)
	}
	if cfg.Delete.DefaultMode != DeleteModeHard {
		t.Fatalf("unexpected delete mode %q", cfg.Delete.DefaultMode)
	}
	if cfg.TaskFields.ShowDueDate {
		t.Fatal("expected due_date hidden from config override")
	}
	if !cfg.TaskFields.ShowDescription {
		t.Fatal("expected description visible from config override")
	}
	if cfg.Confirm.Archive {
		t.Fatalf("expected archive confirm false, got %#v", cfg.Confirm)
	}
	if cfg.UI.ShowDueSummary {
		t.Fatal("expected due summary hidden from config override")
	}
}

// TestLoadEnabledEmbeddingsBlankProviderPreservesLegacyOpenAI verifies older configs that only enabled embeddings keep the historical OpenAI default.
func TestLoadEnabledEmbeddingsBlankProviderPreservesLegacyOpenAI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/tillsyn.db"

[embeddings]
enabled = true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Embeddings.Provider != "openai" {
		t.Fatalf("unexpected legacy embeddings provider %q", cfg.Embeddings.Provider)
	}
	if cfg.Embeddings.Model != "text-embedding-3-small" {
		t.Fatalf("unexpected legacy embeddings model %q", cfg.Embeddings.Model)
	}
	if cfg.Embeddings.APIKeyEnv != "OPENAI_API_KEY" {
		t.Fatalf("unexpected legacy embeddings api_key_env %q", cfg.Embeddings.APIKeyEnv)
	}
	if cfg.Embeddings.BaseURL != "" {
		t.Fatalf("unexpected legacy embeddings base_url %q", cfg.Embeddings.BaseURL)
	}
}

// TestExampleConfigEmbeddingsDefaults verifies the checked-in example file documents and parses the supported operator defaults.
func TestExampleConfigEmbeddingsDefaults(t *testing.T) {
	examplePath := filepath.Clean(filepath.Join("..", "..", "config.example.toml"))
	body, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", examplePath, err)
	}
	text := string(body)
	for _, want := range []string{
		"Ollama's OpenAI-compatible embeddings API",
		"OpenAI-compatible providers",
		"TogetherAI, OpenRouter",
		"deterministic provider is intended for tests and fixtures",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config.example.toml missing phrase %q", want)
		}
	}

	cfg, err := Load(examplePath, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load(%q) error = %v", examplePath, err)
	}
	if cfg.Embeddings.Provider != "ollama" {
		t.Fatalf("Embeddings.Provider = %q, want ollama", cfg.Embeddings.Provider)
	}
	if cfg.Embeddings.Model != "qwen3-embedding:8b" {
		t.Fatalf("Embeddings.Model = %q, want qwen3-embedding:8b", cfg.Embeddings.Model)
	}
	if cfg.Embeddings.APIKeyEnv != "" {
		t.Fatalf("Embeddings.APIKeyEnv = %q, want empty", cfg.Embeddings.APIKeyEnv)
	}
	if cfg.Embeddings.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("Embeddings.BaseURL = %q, want http://127.0.0.1:11434/v1", cfg.Embeddings.BaseURL)
	}
}

// TestLoadIdentityAndPathsOverrides verifies behavior for the covered scenario.
func TestLoadIdentityAndPathsOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/tillsyn.db"

[identity]
actor_id = "  actor-123  "
display_name = "  Evan Schultz  "
default_actor_type = "AGENT"

[paths]
search_roots = [" /tmp/a ", "/tmp/a", "/tmp/b/../b"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Identity.DisplayName != "Evan Schultz" {
		t.Fatalf("unexpected identity display name %q", cfg.Identity.DisplayName)
	}
	if cfg.Identity.ActorID != "actor-123" {
		t.Fatalf("unexpected identity actor_id %q", cfg.Identity.ActorID)
	}
	if cfg.Identity.DefaultActorType != "agent" {
		t.Fatalf("unexpected identity actor type %q", cfg.Identity.DefaultActorType)
	}
	if len(cfg.Paths.SearchRoots) != 2 {
		t.Fatalf("unexpected search roots %#v", cfg.Paths.SearchRoots)
	}
	if cfg.Paths.SearchRoots[0] != filepath.Clean("/tmp/a") {
		t.Fatalf("unexpected first search root %q", cfg.Paths.SearchRoots[0])
	}
	if cfg.Paths.SearchRoots[1] != filepath.Clean("/tmp/b") {
		t.Fatalf("unexpected second search root %q", cfg.Paths.SearchRoots[1])
	}
}

// TestLoadBlankDatabasePathFallsBackToDefault verifies behavior for the covered scenario.
func TestLoadBlankDatabasePathFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = ""
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	defaults := Default("/tmp/default.db")
	cfg, err := Load(path, defaults)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Database.Path; got != defaults.Database.Path {
		t.Fatalf("expected blank database.path to fall back to %q, got %q", defaults.Database.Path, got)
	}
}

// TestLoadRejectsInvalidDeleteMode verifies behavior for the covered scenario.
func TestLoadRejectsInvalidDeleteMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/tillsyn.db"

[delete]
default_mode = "weird"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := Load(path, Default("/tmp/default.db"))
	if err == nil {
		t.Fatal("expected error for invalid delete mode")
	}
}

// TestEnsureConfigDir verifies behavior for the covered scenario.
func TestEnsureConfigDir(t *testing.T) {
	target := filepath.Join(t.TempDir(), "a", "b", "config.toml")
	if err := EnsureConfigDir(target); err != nil {
		t.Fatalf("EnsureConfigDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(target)); err != nil {
		t.Fatalf("expected dir to exist, stat error %v", err)
	}
}

// TestLoadBoardSearchAndKeysOverrides verifies behavior for the covered scenario.
func TestLoadBoardSearchAndKeysOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/tillsyn.db"

[board]
show_wip_warnings = false
group_by = "priority"

[search]
cross_project = true
include_archived = true
states = ["todo", "progress", "archived"]

[ui]
due_soon_windows = ["2h", "48h"]
show_due_summary = true

[logging]
level = "DEBUG"

[logging.dev_file]
enabled = false
dir = "./tmp/logs"

[keys]
command_palette = ":"
quick_actions = "."
multi_select = "space"
activity_log = "g"
undo = "u"
redo = "U"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Board.GroupBy != "priority" || cfg.Board.ShowWIPWarnings {
		t.Fatalf("unexpected board settings %#v", cfg.Board)
	}
	if !cfg.Search.CrossProject || !cfg.Search.IncludeArchived {
		t.Fatalf("unexpected search settings %#v", cfg.Search)
	}
	if len(cfg.Search.States) != 3 {
		t.Fatalf("unexpected search states %#v", cfg.Search.States)
	}
	if cfg.Keys.QuickActions != "." {
		t.Fatalf("unexpected keys config %#v", cfg.Keys)
	}
	if got := cfg.DueSoonDurations(); len(got) != 2 || got[0] != 2*time.Hour || got[1] != 48*time.Hour {
		t.Fatalf("unexpected due durations %#v", got)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("unexpected logging level %q", cfg.Logging.Level)
	}
	if cfg.Logging.DevFile.Enabled {
		t.Fatalf("expected dev file logging disabled, got %#v", cfg.Logging.DevFile)
	}
	if cfg.Logging.DevFile.Dir != "./tmp/logs" {
		t.Fatalf("unexpected dev file logging dir %q", cfg.Logging.DevFile.Dir)
	}
}

// TestValidateRejectsUnknownSearchState verifies behavior for the covered scenario.
func TestValidateRejectsUnknownSearchState(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	cfg.Search.States = []string{"todo", "unknown"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unknown search state validation error")
	}
}

// TestValidateRejectsInvalidDueSoonWindow verifies behavior for the covered scenario.
func TestValidateRejectsInvalidDueSoonWindow(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	cfg.UI.DueSoonWindows = []string{"bogus"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid due-soon duration error")
	}
}

// TestValidateRejectsInvalidLoggingLevel verifies behavior for the covered scenario.
func TestValidateRejectsInvalidLoggingLevel(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	cfg.Logging.Level = "verbose"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid logging level validation error")
	}
}

// TestValidateAllowsEnabledOllamaWithoutAPIKeyEnv verifies the local default embeddings path does not require OpenAI-style credentials.
func TestValidateAllowsEnabledOllamaWithoutAPIKeyEnv(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	cfg.Embeddings.Enabled = true
	cfg.Embeddings.Provider = "ollama"
	cfg.Embeddings.Model = "qwen3-embedding:8b"
	cfg.Embeddings.APIKeyEnv = ""
	cfg.Embeddings.BaseURL = "http://127.0.0.1:11434/v1"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestValidateRejectsEnabledOpenAIWithoutAPIKeyEnv verifies the OpenAI provider still demands an env-var configuration when enabled.
func TestValidateRejectsEnabledOpenAIWithoutAPIKeyEnv(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	cfg.Embeddings.Enabled = true
	cfg.Embeddings.Provider = "openai"
	cfg.Embeddings.Model = "text-embedding-3-small"
	cfg.Embeddings.APIKeyEnv = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected OpenAI embeddings validation error without api_key_env")
	}
}

// TestValidateRejectsInvalidIdentityActorType verifies behavior for the covered scenario.
func TestValidateRejectsInvalidIdentityActorType(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	cfg.Identity.DefaultActorType = "robot"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid identity actor type validation error")
	}
}

// TestDueSoonDurationsNormalizes verifies behavior for the covered scenario.
func TestDueSoonDurationsNormalizes(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	cfg.UI.DueSoonWindows = []string{"2h", "30m", "2h", "bad", "0s"}
	got := cfg.DueSoonDurations()
	want := []time.Duration{30 * time.Minute, 2 * time.Hour}
	if len(got) != len(want) {
		t.Fatalf("unexpected due durations length %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected due duration at %d: got %s want %s", i, got[i], want[i])
		}
	}
}

// TestLoadProjectRootsAndLabels verifies behavior for the covered scenario.
func TestLoadProjectRootsAndLabels(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
path = "/custom/tillsyn.db"

[project_roots]
Inbox = "/Users/test/code/inbox"

[labels]
global = ["Planning", "Bug", "planning"]
enforce_allowed = true

[labels.projects]
inbox = ["till", "Roadmap", "till"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.ProjectRoots["inbox"]; got != "/Users/test/code/inbox" {
		t.Fatalf("unexpected project root mapping %#v", cfg.ProjectRoots)
	}
	if !cfg.Labels.EnforceAllowed {
		t.Fatalf("expected enforce_allowed true, got %#v", cfg.Labels)
	}
	allowed := cfg.AllowedLabels("inbox")
	want := []string{"bug", "planning", "roadmap", "till"}
	if len(allowed) != len(want) {
		t.Fatalf("unexpected allowed labels %#v", allowed)
	}
	for i := range want {
		if allowed[i] != want[i] {
			t.Fatalf("unexpected allowed label at %d: got %q want %q", i, allowed[i], want[i])
		}
	}
}

// TestValidateRejectsEmptyProjectRoot verifies behavior for the covered scenario.
func TestValidateRejectsEmptyProjectRoot(t *testing.T) {
	cfg := Default("/tmp/tillsyn.db")
	cfg.ProjectRoots["inbox"] = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty project root")
	}
}

// TestUpsertProjectRootWritesAndClearsMapping verifies behavior for the covered scenario.
func TestUpsertProjectRootWritesAndClearsMapping(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[database]
path = "/tmp/custom.db"

[project_roots]
legacy = "/tmp/legacy"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := UpsertProjectRoot(path, "Inbox", "/tmp/inbox"); err != nil {
		t.Fatalf("UpsertProjectRoot() error = %v", err)
	}
	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.ProjectRoots["inbox"]; got != "/tmp/inbox" {
		t.Fatalf("expected inbox root /tmp/inbox, got %#v", cfg.ProjectRoots)
	}
	if got := cfg.ProjectRoots["legacy"]; got != "/tmp/legacy" {
		t.Fatalf("expected legacy root preserved, got %#v", cfg.ProjectRoots)
	}
	if got := cfg.Database.Path; got != "/tmp/custom.db" {
		t.Fatalf("expected database path preserved, got %q", got)
	}

	if err := UpsertProjectRoot(path, "inbox", ""); err != nil {
		t.Fatalf("UpsertProjectRoot(clear) error = %v", err)
	}
	cfg, err = Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if _, ok := cfg.ProjectRoots["inbox"]; ok {
		t.Fatalf("expected inbox root cleared, got %#v", cfg.ProjectRoots)
	}
	if got := cfg.ProjectRoots["legacy"]; got != "/tmp/legacy" {
		t.Fatalf("expected legacy root preserved after clear, got %#v", cfg.ProjectRoots)
	}
}

// TestUpsertProjectRootMissingFileClearNoop verifies behavior for the covered scenario.
func TestUpsertProjectRootMissingFileClearNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	if err := UpsertProjectRoot(path, "inbox", ""); err != nil {
		t.Fatalf("UpsertProjectRoot() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no file created for clear noop, stat err=%v", err)
	}
}

// TestUpsertProjectRootRejectsInvalidInput verifies behavior for the covered scenario.
func TestUpsertProjectRootRejectsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := UpsertProjectRoot("", "inbox", "/tmp/inbox"); err == nil {
		t.Fatal("expected error for empty config path")
	}
	if err := UpsertProjectRoot(path, "", "/tmp/inbox"); err == nil {
		t.Fatal("expected error for empty project slug")
	}
}

// TestUpsertIdentityWritesAndClears verifies behavior for the covered scenario.
func TestUpsertIdentityWritesAndClears(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[database]
path = "/tmp/custom.db"

[identity]
actor_id = "lane-actor"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := UpsertIdentity(path, "", "Evan", "agent"); err != nil {
		t.Fatalf("UpsertIdentity() error = %v", err)
	}
	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Identity.DisplayName; got != "Evan" {
		t.Fatalf("expected persisted display name Evan, got %q", got)
	}
	if got := cfg.Identity.DefaultActorType; got != "agent" {
		t.Fatalf("expected persisted actor type agent, got %q", got)
	}
	if got := cfg.Identity.ActorID; got != "lane-actor" {
		t.Fatalf("expected actor_id preserved as lane-actor, got %q", got)
	}
	if got := cfg.Database.Path; got != "/tmp/custom.db" {
		t.Fatalf("expected database path preserved, got %q", got)
	}

	if err := UpsertIdentity(path, "actor-v2", "", "system"); err != nil {
		t.Fatalf("UpsertIdentity(clear display) error = %v", err)
	}
	cfg, err = Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if got := cfg.Identity.DisplayName; got != "" {
		t.Fatalf("expected display name cleared, got %q", got)
	}
	if got := cfg.Identity.DefaultActorType; got != "system" {
		t.Fatalf("expected actor type system after clear, got %q", got)
	}
	if got := cfg.Identity.ActorID; got != "actor-v2" {
		t.Fatalf("expected actor_id actor-v2 after update, got %q", got)
	}
}

// TestUpsertIdentityMissingFileClearNoop verifies behavior for the covered scenario.
func TestUpsertIdentityMissingFileClearNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	if err := UpsertIdentity(path, "", "", ""); err != nil {
		t.Fatalf("UpsertIdentity() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no file created for clear noop, stat err=%v", err)
	}
}

// TestUpsertIdentityRejectsInvalidInput verifies behavior for the covered scenario.
func TestUpsertIdentityRejectsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := UpsertIdentity("", "", "Evan", "user"); err == nil {
		t.Fatal("expected error for empty config path")
	}
	if err := UpsertIdentity(path, "", "Evan", "robot"); err == nil {
		t.Fatal("expected error for invalid actor type")
	}
}

// TestUpsertSearchRootsWritesAndClears verifies behavior for the covered scenario.
func TestUpsertSearchRootsWritesAndClears(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[database]
path = "/tmp/custom.db"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := UpsertSearchRoots(path, []string{" /tmp/a ", "/tmp/a", "/tmp/b/../b"}); err != nil {
		t.Fatalf("UpsertSearchRoots() error = %v", err)
	}
	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []string{filepath.Clean("/tmp/a"), filepath.Clean("/tmp/b")}
	if len(cfg.Paths.SearchRoots) != len(want) {
		t.Fatalf("unexpected persisted search roots %#v", cfg.Paths.SearchRoots)
	}
	for i := range want {
		if cfg.Paths.SearchRoots[i] != want[i] {
			t.Fatalf("unexpected search root at %d: got %q want %q", i, cfg.Paths.SearchRoots[i], want[i])
		}
	}

	if err := UpsertSearchRoots(path, nil); err != nil {
		t.Fatalf("UpsertSearchRoots(clear) error = %v", err)
	}
	cfg, err = Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if len(cfg.Paths.SearchRoots) != 0 {
		t.Fatalf("expected search roots cleared, got %#v", cfg.Paths.SearchRoots)
	}
}

// TestUpsertSearchRootsMissingFileClearNoop verifies behavior for the covered scenario.
func TestUpsertSearchRootsMissingFileClearNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	if err := UpsertSearchRoots(path, nil); err != nil {
		t.Fatalf("UpsertSearchRoots() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no file created for clear noop, stat err=%v", err)
	}
}

// TestUpsertSearchRootsRejectsInvalidInput verifies behavior for the covered scenario.
func TestUpsertSearchRootsRejectsInvalidInput(t *testing.T) {
	if err := UpsertSearchRoots("", []string{"/tmp/a"}); err == nil {
		t.Fatal("expected error for empty config path")
	}
}

// TestUpsertAllowedLabelsWritesAndClears verifies behavior for the covered scenario.
func TestUpsertAllowedLabelsWritesAndClears(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[database]
path = "/tmp/custom.db"

[labels]
global = ["legacy"]

[labels.projects]
legacy = ["ops"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := UpsertAllowedLabels(path, "Inbox", []string{"Bug", "chore", "bug"}, []string{"Roadmap", "till", "roadmap"}); err != nil {
		t.Fatalf("UpsertAllowedLabels() error = %v", err)
	}
	cfg, err := Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	wantGlobal := []string{"bug", "chore"}
	if len(cfg.Labels.Global) != len(wantGlobal) {
		t.Fatalf("unexpected global labels %#v", cfg.Labels.Global)
	}
	for i := range wantGlobal {
		if cfg.Labels.Global[i] != wantGlobal[i] {
			t.Fatalf("unexpected global label at %d: got %q want %q", i, cfg.Labels.Global[i], wantGlobal[i])
		}
	}
	wantInbox := []string{"roadmap", "till"}
	gotInbox := cfg.Labels.Projects["inbox"]
	if len(gotInbox) != len(wantInbox) {
		t.Fatalf("unexpected inbox labels %#v", cfg.Labels.Projects)
	}
	for i := range wantInbox {
		if gotInbox[i] != wantInbox[i] {
			t.Fatalf("unexpected inbox label at %d: got %q want %q", i, gotInbox[i], wantInbox[i])
		}
	}
	if gotLegacy := cfg.Labels.Projects["legacy"]; len(gotLegacy) != 1 || gotLegacy[0] != "ops" {
		t.Fatalf("expected legacy labels preserved, got %#v", cfg.Labels.Projects)
	}

	if err := UpsertAllowedLabels(path, "inbox", []string{"bug"}, nil); err != nil {
		t.Fatalf("UpsertAllowedLabels(clear project) error = %v", err)
	}
	cfg, err = Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after project clear error = %v", err)
	}
	if _, ok := cfg.Labels.Projects["inbox"]; ok {
		t.Fatalf("expected inbox project labels cleared, got %#v", cfg.Labels.Projects)
	}
	if len(cfg.Labels.Global) != 1 || cfg.Labels.Global[0] != "bug" {
		t.Fatalf("expected global labels to remain set, got %#v", cfg.Labels.Global)
	}

	if err := UpsertAllowedLabels(path, "inbox", nil, nil); err != nil {
		t.Fatalf("UpsertAllowedLabels(clear globals) error = %v", err)
	}
	cfg, err = Load(path, Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after globals clear error = %v", err)
	}
	if len(cfg.Labels.Global) != 0 {
		t.Fatalf("expected global labels cleared, got %#v", cfg.Labels.Global)
	}
	if gotLegacy := cfg.Labels.Projects["legacy"]; len(gotLegacy) != 1 || gotLegacy[0] != "ops" {
		t.Fatalf("expected legacy project labels preserved, got %#v", cfg.Labels.Projects)
	}
}

// TestUpsertAllowedLabelsMissingFileClearNoop verifies behavior for the covered scenario.
func TestUpsertAllowedLabelsMissingFileClearNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	if err := UpsertAllowedLabels(path, "inbox", nil, nil); err != nil {
		t.Fatalf("UpsertAllowedLabels() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no file created for clear noop, stat err=%v", err)
	}
}

// TestUpsertAllowedLabelsRejectsInvalidInput verifies behavior for the covered scenario.
func TestUpsertAllowedLabelsRejectsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := UpsertAllowedLabels("", "inbox", []string{"bug"}, nil); err == nil {
		t.Fatal("expected error for empty config path")
	}
	if err := UpsertAllowedLabels(path, "", []string{"bug"}, nil); err == nil {
		t.Fatal("expected error for empty project slug")
	}
}

// TestIsKnownLifecycleStateIncludesFailed verifies that "failed" is recognized as a known lifecycle state.
func TestIsKnownLifecycleStateIncludesFailed(t *testing.T) {
	if !isKnownLifecycleState("failed") {
		t.Fatal("isKnownLifecycleState(\"failed\") = false, want true")
	}
	for _, state := range []string{"todo", "progress", "done", "archived"} {
		if !isKnownLifecycleState(state) {
			t.Fatalf("isKnownLifecycleState(%q) = false, want true", state)
		}
	}
	if isKnownLifecycleState("invalid") {
		t.Fatal("isKnownLifecycleState(\"invalid\") = true, want false")
	}
}
