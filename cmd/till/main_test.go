package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	charmLog "github.com/charmbracelet/log"
	autentdomain "github.com/evanmschultz/autent/domain"
	"github.com/google/uuid"
	"github.com/hylla/tillsyn/internal/adapters/auth/autentauth"
	serveradapter "github.com/hylla/tillsyn/internal/adapters/server"
	servercommon "github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/config"
	"github.com/hylla/tillsyn/internal/domain"
	"github.com/hylla/tillsyn/internal/platform"
)

// TestMain sets deterministic environment defaults for CLI tests.
func TestMain(m *testing.M) {
	_ = os.Setenv("TILL_DEV_MODE", "false")
	os.Exit(m.Run())
}

// ansiEscapePattern matches ANSI color/style escape sequences in forced-style output tests.
var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSITest removes ANSI escape sequences from CLI test output snapshots.
func stripANSITest(text string) string {
	return ansiEscapePattern.ReplaceAllString(text, "")
}

// extractCLIKVValue returns one laslig key/value field value from human CLI output.
func extractCLIKVValue(t *testing.T, output, label string) string {
	t.Helper()
	re := regexp.MustCompile(`(?mi)^\s*` + regexp.QuoteMeta(label) + `\s+(.+?)\s*$`)
	match := re.FindStringSubmatch(stripANSITest(output))
	if match == nil {
		t.Fatalf("expected label %q in output, got %q", label, output)
	}
	return strings.TrimSpace(match[1])
}

// fakeProgram represents fake program data used by this package.
type fakeProgram struct {
	runErr error
}

// Run runs the requested command flow.
func (f fakeProgram) Run() (tea.Model, error) {
	return nil, f.runErr
}

// scriptedProgram represents program data used to exercise model flows inside run() tests.
type scriptedProgram struct {
	model tea.Model
	runFn func(tea.Model) (tea.Model, error)
}

// Run runs scripted model interactions and returns the final state.
func (p scriptedProgram) Run() (tea.Model, error) {
	if p.runFn == nil {
		return p.model, nil
	}
	return p.runFn(p.model)
}

// applyModelMsg applies one message and any resulting command chain.
func applyModelMsg(t *testing.T, model tea.Model, msg tea.Msg) tea.Model {
	t.Helper()
	updated, cmd := model.Update(msg)
	return applyModelCmd(t, updated, cmd)
}

// applyModelCmd executes one command chain to completion (bounded for safety).
func applyModelCmd(t *testing.T, model tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	out := model
	currentCmd := cmd
	for i := 0; i < 8 && currentCmd != nil; i++ {
		msg := currentCmd()
		updated, nextCmd := out.Update(msg)
		out = updated
		currentCmd = nextCmd
	}
	return out
}

// writeBootstrapReadyConfig writes the minimum startup fields required to bypass bootstrap modal gating.
func writeBootstrapReadyConfig(t *testing.T, path, searchRoot string) {
	t.Helper()
	content := fmt.Sprintf(`
[identity]
actor_id = "lane-actor-id"
display_name = "Test User"
default_actor_type = "user"

[paths]
search_roots = [%q]
`, searchRoot)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

// writeConfigExample writes a local config.example.toml template for startup seeding tests.
func writeConfigExample(t *testing.T, workspace, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(workspace, "config.example.toml"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(config.example.toml) error = %v", err)
	}
}

// newAuthAdapterForTest constructs one shared-DB auth adapter for mutation authorization tests.
func newAuthAdapterForTest(t *testing.T) (*servercommon.AppServiceAdapter, *autentauth.Service) {
	t.Helper()

	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "tillsyn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	auth, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	return servercommon.NewAppServiceAdapter(nil, auth), auth
}

// mustIssueUserSessionForAdapterTest issues one deterministic session for adapter authorization tests.
func mustIssueUserSessionForAdapterTest(t *testing.T, auth *autentauth.Service) (string, string) {
	t.Helper()

	issued, err := auth.IssueSession(context.Background(), autentauth.IssueSessionInput{
		PrincipalID:   "user-1",
		PrincipalType: "user",
		PrincipalName: "User One",
		ClientID:      "till-mcp-stdio",
		ClientType:    "mcp-stdio",
		ClientName:    "Till MCP STDIO",
	})
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	return issued.Session.ID, issued.Secret
}

// mustNormalizeAuthRuleForTest validates one auth rule for stable adapter tests.
func mustNormalizeAuthRuleForTest(t *testing.T, rule autentdomain.Rule) autentdomain.Rule {
	t.Helper()

	normalized, err := autentdomain.ValidateAndNormalizeRule(rule)
	if err != nil {
		t.Fatalf("ValidateAndNormalizeRule() error = %v", err)
	}
	return normalized
}

// seedProjectForAuthCLITest stores one minimal project row for auth CLI lifecycle tests.
func seedProjectForAuthCLITest(t *testing.T, dbPath, projectID string) {
	t.Helper()

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	defer func() {
		_ = repo.Close()
	}()

	project, err := domain.NewProject(projectID, "Project "+projectID, "", time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
}

// seedTemplateLibraryForProjectCreateCLITest stores one approved global project template library for CLI create coverage.
func seedTemplateLibraryForProjectCreateCLITest(t *testing.T, dbPath string) {
	t.Helper()

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	defer func() {
		_ = repo.Close()
	}()

	svc := app.NewService(repo, func() string { return uuid.NewString() }, func() time.Time {
		return time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	}, app.ServiceConfig{})
	if _, err := svc.ListKindDefinitions(context.Background(), false); err != nil {
		t.Fatalf("ListKindDefinitions(seed) error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(context.Background(), app.CreateKindDefinitionInput{
		ID:          "go-service",
		DisplayName: "Go Service",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToProject},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(go-service) error = %v", err)
	}
	if _, err := svc.UpsertTemplateLibrary(context.Background(), app.UpsertTemplateLibraryInput{
		ID:                  "go-defaults",
		Scope:               domain.TemplateLibraryScopeGlobal,
		Name:                "Go Defaults",
		Description:         "Global defaults for Go projects",
		Status:              domain.TemplateLibraryStatusApproved,
		CreatedByActorID:    "user-1",
		CreatedByActorName:  "User One",
		CreatedByActorType:  domain.ActorTypeUser,
		ApprovedByActorID:   "user-1",
		ApprovedByActorName: "User One",
		ApprovedByActorType: domain.ActorTypeUser,
		NodeTemplates: []app.UpsertNodeTemplateInput{{
			ID:         "project-template",
			ScopeLevel: domain.KindAppliesToProject,
			NodeKindID: domain.KindID("go-service"),
			ProjectMetadataDefaults: &domain.ProjectMetadata{
				Owner:             "Platform",
				StandardsMarkdown: "Run Go validation",
			},
			ChildRules: []app.UpsertTemplateChildRuleInput{{
				ID:                      "main-branch",
				Position:                1,
				ChildScopeLevel:         domain.KindAppliesToBranch,
				ChildKindID:             domain.KindID("branch"),
				TitleTemplate:           "Main Branch",
				DescriptionTemplate:     "default implementation branch",
				ResponsibleActorKind:    domain.TemplateActorKindBuilder,
				EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindBuilder},
				CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindBuilder, domain.TemplateActorKindHuman},
				RequiredForParentDone:   true,
			}},
		}},
	}); err != nil {
		t.Fatalf("UpsertTemplateLibrary(go-defaults) error = %v", err)
	}
}

// archiveProjectForCLITest marks one seeded project archived for CLI discovery tests.
func archiveProjectForCLITest(t *testing.T, dbPath, projectID string) {
	t.Helper()

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	defer func() {
		_ = repo.Close()
	}()

	project, err := repo.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("GetProject(%q) error = %v", projectID, err)
	}
	project.Archive(time.Date(2026, 3, 23, 12, 30, 0, 0, time.UTC))
	if err := repo.UpdateProject(context.Background(), project); err != nil {
		t.Fatalf("UpdateProject(%q) error = %v", projectID, err)
	}
}

// TestRunVersion verifies behavior for the covered scenario.
func TestRunVersion(t *testing.T) {
	var out strings.Builder
	err := run(context.Background(), []string{"--version"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("run(version) error = %v", err)
	}
	if !strings.Contains(out.String(), "till") {
		t.Fatalf("expected version output, got %q", out.String())
	}
}

// TestRunStartsProgram verifies behavior for the covered scenario.
func TestRunStartsProgram(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })

	programFactory = func(_ tea.Model) program {
		return fakeProgram{}
	}

	dbPath := filepath.Join(t.TempDir(), "tillsyn.db")
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, t.TempDir())
	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

// TestRunStartupPreservesExistingActorID verifies startup keeps a preconfigured identity.actor_id unchanged.
func TestRunStartupPreservesExistingActorID(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program {
		return fakeProgram{}
	}

	dbPath := filepath.Join(t.TempDir(), "tillsyn.db")
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	content := fmt.Sprintf(`
[identity]
actor_id = "existing-actor-id"
display_name = "Lane User"
default_actor_type = "user"

[paths]
search_roots = [%q]
`, t.TempDir())
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default(dbPath))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Identity.ActorID; got != "existing-actor-id" {
		t.Fatalf("expected startup to preserve actor_id existing-actor-id, got %q", got)
	}
}

// TestRunSeedsMissingConfigFromExampleOnStartup verifies first-launch startup seeds a missing config from config.example.toml.
func TestRunSeedsMissingConfigFromExampleOnStartup(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	const example = "[identity]\ndisplay_name = \"\"\n\n[paths]\nsearch_roots = []\n"
	writeConfigExample(t, workspace, example)

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	cfg, err := config.Load(cfgPath, config.Default(dbPath))
	if err != nil {
		t.Fatalf("Load(config) error = %v", err)
	}
	if got := cfg.Identity.DefaultActorType; got != "user" {
		t.Fatalf("expected seeded actor type user, got %q", got)
	}
	if got := strings.TrimSpace(cfg.Identity.ActorID); got == "" {
		t.Fatal("expected startup seed flow to generate identity.actor_id")
	} else if _, parseErr := uuid.Parse(got); parseErr != nil {
		t.Fatalf("expected generated identity.actor_id to be a UUID, got %q (%v)", got, parseErr)
	}
}

// TestRunNonStartupCommandsDoNotSeedMissingConfig verifies non-startup commands keep missing config behavior side-effect free.
func TestRunNonStartupCommandsDoNotSeedMissingConfig(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	writeConfigExample(t, workspace, "[identity]\ndisplay_name = \"\"\n")

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	outPath := filepath.Join(workspace, "snapshot.json")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", outPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(export) error = %v", err)
	}

	if _, err := os.Stat(cfgPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected export path to avoid seeding config, stat err = %v", err)
	}
}

// TestRunTUIStartupDoesNotCreateDefaultProject verifies behavior for the covered scenario.
func TestRunTUIStartupDoesNotCreateDefaultProject(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, t.TempDir())
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	outPath := filepath.Join(tmp, "snapshot.json")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", outPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(export) error = %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var snap app.Snapshot
	if err := json.Unmarshal(content, &snap); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(snap.Projects) != 0 {
		t.Fatalf("expected no auto-created startup projects, got %d", len(snap.Projects))
	}
}

// TestRunBootstrapModalPersistsMissingFields verifies startup bootstrap persists through TUI callbacks.
func TestRunBootstrapModalPersistsMissingFields(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(model tea.Model) program {
		return scriptedProgram{
			model: model,
			runFn: func(current tea.Model) (tea.Model, error) {
				current = applyModelCmd(t, current, current.Init())
				current = applyModelMsg(t, current, tea.WindowSizeMsg{Width: 120, Height: 40})
				if rendered := fmt.Sprint(current.View().Content); !strings.Contains(rendered, "Startup Setup Required") {
					t.Fatalf("expected startup bootstrap modal, got\n%s", rendered)
				}

				for _, r := range "Lane User" {
					current = applyModelMsg(t, current, tea.KeyPressMsg{Code: r, Text: string(r)})
				}
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: tea.KeyTab})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: tea.KeyTab})
				current = applyModelMsg(t, current, tea.KeyPressMsg{Code: tea.KeyEnter})
				if rendered := fmt.Sprint(current.View().Content); !strings.Contains(rendered, "Projects") {
					t.Fatalf("expected project picker after bootstrap save, got\n%s", rendered)
				}
				return current, nil
			},
		}
	}

	workspace := t.TempDir()
	t.Chdir(workspace)
	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default(dbPath))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Identity.DisplayName; got != "Lane User" {
		t.Fatalf("expected persisted display name Lane User, got %q", got)
	}
	if got := cfg.Identity.DefaultActorType; got != "user" {
		t.Fatalf("expected persisted actor type user, got %q", got)
	}
	if got := strings.TrimSpace(cfg.Identity.ActorID); got == "" {
		t.Fatal("expected persisted actor_id after bootstrap flow")
	} else if _, parseErr := uuid.Parse(got); parseErr != nil {
		t.Fatalf("expected persisted actor_id to be a UUID, got %q (%v)", got, parseErr)
	}
	if len(cfg.Paths.SearchRoots) != 1 || cfg.Paths.SearchRoots[0] != filepath.Clean(workspace) {
		t.Fatalf("expected persisted search root %q, got %#v", filepath.Clean(workspace), cfg.Paths.SearchRoots)
	}
}

// TestRunInvalidFlag verifies behavior for the covered scenario.
func TestRunInvalidFlag(t *testing.T) {
	err := run(context.Background(), []string{"--unknown-flag"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected flag parse error")
	}
}

// TestRunRootHelp verifies root help output returns usage without error.
func TestRunRootHelp(t *testing.T) {
	forms := [][]string{{"--help"}, {"-h"}, {"help"}, {"h"}}
	var reference string
	for _, form := range forms {
		var out strings.Builder
		err := run(context.Background(), form, &out, io.Discard)
		if err != nil {
			t.Fatalf("run(%v) error = %v", form, err)
		}
		if reference == "" {
			reference = out.String()
		} else if out.String() != reference {
			t.Fatalf("root help mismatch for %v\n--- want ---\n%s\n--- got ---\n%s", form, reference, out.String())
		}
		output := strings.ToLower(out.String())
		if !strings.Contains(output, "usage") || !strings.Contains(output, "till [command]") {
			t.Fatalf("expected root usage output, got %q", out.String())
		}
		for _, want := range []string{"serve", "mcp", "auth", "project", "embeddings", "capture-state", "kind", "template", "lease", "handoff", "export", "import", "paths", "init-dev-config"} {
			if !strings.Contains(output, want) {
				t.Fatalf("expected %q command in root help, got %q", want, out.String())
			}
		}
		for _, want := range []string{"till project create --name inbox", "till template library list", "till embeddings status"} {
			if !strings.Contains(output, strings.ToLower(want)) {
				t.Fatalf("expected %q in root help examples, got %q", want, out.String())
			}
		}
	}
}

// TestRunSubcommandHelp verifies subcommand help output returns usage without executing command handlers.
func TestRunSubcommandHelp(t *testing.T) {
	origRunner := serveCommandRunner
	t.Cleanup(func() { serveCommandRunner = origRunner })
	serveCommandRunner = func(_ context.Context, _ serveradapter.Config, _ serveradapter.Dependencies) error {
		t.Fatal("serve command runner should not execute for --help")
		return nil
	}

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "serve",
			args: []string{"serve", "--help"},
			want: []string{"till serve", "--http", "--api-endpoint", "--mcp-endpoint"},
		},
		{
			name: "mcp",
			args: []string{"mcp", "--help"},
			want: []string{"till mcp", "start raw mcp over stdio"},
		},
		{
			name: "auth",
			args: []string{"auth", "--help"},
			want: []string{"till auth", "request", "session", "issue-session", "session revoke --session-id", "projects/PROJECT_ID_A,PROJECT_ID_B...", "global"},
		},
		{
			name: "project",
			args: []string{"project", "--help"},
			want: []string{"till project", "list", "create", "show", "discover"},
		},
		{
			name: "project list",
			args: []string{"project", "list", "--help"},
			want: []string{"till project list", "--include-archived", "names first", "ids visible"},
		},
		{
			name: "project create",
			args: []string{"project", "create", "--help"},
			want: []string{"till project create", "--name", "--kind", "--template-library-id", "--metadata-json", "one positional argument"},
		},
		{
			name: "project show",
			args: []string{"project", "show", "--help"},
			want: []string{"till project show", "--project-id", "one positional", "project list", "project discover"},
		},
		{
			name: "project discover",
			args: []string{"project", "discover", "--help"},
			want: []string{"till project discover", "--project-id", "one positional", "collaboration-readiness bridge", "auth", "session", "lease", "handoff"},
		},
		{
			name: "auth request",
			args: []string{"auth", "request", "--help"},
			want: []string{"till auth request", "create", "approve", "project/PROJECT_ID", "projects/PROJECT_ID_A,PROJECT_ID_B...", "global"},
		},
		{
			name: "auth request list",
			args: []string{"auth", "request", "list", "--help"},
			want: []string{"till auth request list", "--project-id", "--state", "--limit"},
		},
		{
			name: "auth request show",
			args: []string{"auth", "request", "show", "--help"},
			want: []string{"till auth request show", "--request-id", "issued_session_id", "auth session"},
		},
		{
			name: "auth request create",
			args: []string{"auth", "request", "create", "--help"},
			want: []string{"till auth request create", "--path", "--principal-id", "--principal-role", "--continuation-json", "resume_token", "projects/PROJECT_ID_A,PROJECT_ID_B", "global", "next step"},
		},
		{
			name: "auth request approve",
			args: []string{"auth", "request", "approve", "--help"},
			want: []string{"till auth request approve", "--path", "--ttl", "claim_auth_request", "resume_token", "projects/...", "global", "approved record"},
		},
		{
			name: "auth request deny",
			args: []string{"auth", "request", "deny", "--help"},
			want: []string{"till auth request deny", "--request-id", "--note", "terminal state"},
		},
		{
			name: "auth request cancel",
			args: []string{"auth", "request", "cancel", "--help"},
			want: []string{"till auth request cancel", "--request-id", "--note", "terminal state"},
		},
		{
			name: "auth session",
			args: []string{"auth", "session", "--help"},
			want: []string{"till auth session", "list", "validate", "revoke"},
		},
		{
			name: "auth session list",
			args: []string{"auth", "session", "list", "--help"},
			want: []string{"till auth session list", "--project-id", "--principal-id", "--client-id", "approved project identifier"},
		},
		{
			name: "auth session revoke",
			args: []string{"auth", "session", "revoke", "--help"},
			want: []string{"till auth session revoke", "--session-id", "does not accept the session id"},
		},
		{
			name: "auth session validate",
			args: []string{"auth", "session", "validate", "--help"},
			want: []string{"till auth session validate", "--session-id", "--session-secret", "use it with mcp mutation calls"},
		},
		{
			name: "auth issue-session",
			args: []string{"auth", "issue-session", "--help"},
			want: []string{"till auth issue-session", "--principal-id", "--ttl", "session_id", "session_secret", "next step"},
		},
		{
			name: "auth revoke-session",
			args: []string{"auth", "revoke-session", "--help"},
			want: []string{"till auth revoke-session", "--session-id", "--reason"},
		},
		{
			name: "capture-state",
			args: []string{"capture-state", "--help"},
			want: []string{"till capture-state", "--project-id", "--scope-type", "capture state"},
		},
		{
			name: "kind",
			args: []string{"kind", "--help"},
			want: []string{"till kind", "list", "upsert", "allowlist", "template-library workflow contracts"},
		},
		{
			name: "kind list",
			args: []string{"kind", "list", "--help"},
			want: []string{"till kind list", "--include-archived", "discover valid kind ids"},
		},
		{
			name: "kind upsert",
			args: []string{"kind", "upsert", "--help"},
			want: []string{"till kind upsert", "--id", "--display-name", "--applies-to", "--payload-schema-json", "compatibility-only"},
		},
		{
			name: "kind allowlist",
			args: []string{"kind", "allowlist", "--help"},
			want: []string{"till kind allowlist", "list", "set"},
		},
		{
			name: "kind allowlist list",
			args: []string{"kind", "allowlist", "list", "--help"},
			want: []string{"till kind allowlist list", "--project-id", "template libraries", "project"},
		},
		{
			name: "kind allowlist set",
			args: []string{"kind", "allowlist", "set", "--help"},
			want: []string{"till kind allowlist set", "--project-id", "--kind-id", "replace operation"},
		},
		{
			name: "template",
			args: []string{"template", "--help"},
			want: []string{"till template", "library", "project", "contract"},
		},
		{
			name: "template library",
			args: []string{"template", "library", "--help"},
			want: []string{"till template library", "list", "show", "upsert"},
		},
		{
			name: "template library list",
			args: []string{"template", "library", "list", "--help"},
			want: []string{"till template library list", "--scope", "--project-id", "--status"},
		},
		{
			name: "template library show",
			args: []string{"template", "library", "show", "--help"},
			want: []string{"till template library show", "--library-id", "child-rule contract table"},
		},
		{
			name: "template library upsert",
			args: []string{"template", "library", "upsert", "--help"},
			want: []string{"till template library upsert", "--spec-json", "sqlite remains the source of truth", "$(cat /tmp/template-library.json)"},
		},
		{
			name: "template project",
			args: []string{"template", "project", "--help"},
			want: []string{"till template project", "bind", "binding"},
		},
		{
			name: "template project bind",
			args: []string{"template", "project", "bind", "--help"},
			want: []string{"till template project bind", "--project-id", "--library-id", "approved template library"},
		},
		{
			name: "template project binding",
			args: []string{"template", "project", "binding", "--help"},
			want: []string{"till template project binding", "--project-id", "active template-library binding"},
		},
		{
			name: "template contract",
			args: []string{"template", "contract", "--help"},
			want: []string{"till template contract", "show", "truthful runtime record"},
		},
		{
			name: "template contract show",
			args: []string{"template", "contract", "show", "--help"},
			want: []string{"till template contract show", "--node-id", "generated node-contract snapshot"},
		},
		{
			name: "embeddings",
			args: []string{"embeddings", "--help"},
			want: []string{"till embeddings", "status", "reindex"},
		},
		{
			name: "embeddings status",
			args: []string{"embeddings", "status", "--help"},
			want: []string{"till embeddings status", "--project-id", "--cross-project", "--status", "--limit"},
		},
		{
			name: "embeddings reindex",
			args: []string{"embeddings", "reindex", "--help"},
			want: []string{"till embeddings reindex", "--project-id", "--cross-project", "--force", "--wait"},
		},
		{
			name: "lease",
			args: []string{"lease", "--help"},
			want: []string{"till lease", "list", "issue", "heartbeat", "renew", "revoke", "revoke-all"},
		},
		{
			name: "lease list",
			args: []string{"lease", "list", "--help"},
			want: []string{"till lease list", "--project-id", "--scope-type", "--include-revoked"},
		},
		{
			name: "lease issue",
			args: []string{"lease", "issue", "--help"},
			want: []string{"till lease issue", "--project-id", "--agent-name", "--role", "--requested-ttl"},
		},
		{
			name: "lease heartbeat",
			args: []string{"lease", "heartbeat", "--help"},
			want: []string{"till lease heartbeat", "--agent-instance-id", "--lease-token"},
		},
		{
			name: "lease renew",
			args: []string{"lease", "renew", "--help"},
			want: []string{"till lease renew", "--agent-instance-id", "--lease-token", "--ttl"},
		},
		{
			name: "lease revoke",
			args: []string{"lease", "revoke", "--help"},
			want: []string{"till lease revoke", "--agent-instance-id", "--reason"},
		},
		{
			name: "lease revoke-all",
			args: []string{"lease", "revoke-all", "--help"},
			want: []string{"till lease revoke-all", "--project-id", "--scope-type", "--reason"},
		},
		{
			name: "handoff",
			args: []string{"handoff", "--help"},
			want: []string{"till handoff", "create", "get", "list", "update"},
		},
		{
			name: "handoff create",
			args: []string{"handoff", "create", "--help"},
			want: []string{"till handoff create", "--project-id", "--summary", "--source-role", "--target-role"},
		},
		{
			name: "handoff get",
			args: []string{"handoff", "get", "--help"},
			want: []string{"till handoff get", "--handoff-id"},
		},
		{
			name: "handoff list",
			args: []string{"handoff", "list", "--help"},
			want: []string{"till handoff list", "--project-id", "--scope-type", "--status", "--limit"},
		},
		{
			name: "handoff update",
			args: []string{"handoff", "update", "--help"},
			want: []string{"till handoff update", "--handoff-id", "--summary", "--resolution-note"},
		},
		{
			name: "export",
			args: []string{"export", "--help"},
			want: []string{"till export", "--out", "--include-archived"},
		},
		{
			name: "import",
			args: []string{"import", "--help"},
			want: []string{"till import", "--in"},
		},
		{
			name: "paths",
			args: []string{"paths", "--help"},
			want: []string{"till paths", "--dev", "--db", "--config", "resolved runtime paths"},
		},
		{
			name: "init-dev-config",
			args: []string{"init-dev-config", "--help"},
			want: []string{"till init-dev-config", "create the dev config file"},
		},
	}
	forms := []string{"--help", "-h", "help", "h"}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			baseArgs := tc.args[:len(tc.args)-1]
			var reference string
			for _, form := range forms {
				args := append(append([]string(nil), baseArgs...), form)
				var out strings.Builder
				err := run(context.Background(), args, &out, io.Discard)
				if err != nil {
					t.Fatalf("run(%v) error = %v", args, err)
				}
				if reference == "" {
					reference = out.String()
				} else if out.String() != reference {
					t.Fatalf("help output mismatch for %v\n--- want ---\n%s\n--- got ---\n%s", args, reference, out.String())
				}
				output := strings.ToLower(out.String())
				if !strings.Contains(output, "usage") {
					t.Fatalf("expected usage section in output, got %q", out.String())
				}
				if !strings.Contains(output, "examples") {
					t.Fatalf("expected examples section in output, got %q", out.String())
				}
				for _, want := range tc.want {
					if !strings.Contains(output, strings.ToLower(want)) {
						t.Fatalf("expected %q in output, got %q", want, out.String())
					}
				}
				for _, forbidden := range []string{
					"--project-id p1",
					"project/p1",
					"projects/PROJECT_ID,PROJECT_ID...",
					"review-agent",
					"qa-agent",
					"orchestration-agent",
					"review-user",
					"builder-1",
					"qa-1",
					"orchestrator-1",
					"resume-123",
					"$(cat /tmp/go-defaults.json)",
				} {
					if strings.Contains(output, forbidden) {
						t.Fatalf("did not expect legacy opaque example %q in output %q", forbidden, out.String())
					}
				}
			}
		})
	}
}

// TestRunAuthIssueAndRevokeSession verifies the local dogfood auth command issues and revokes real shared-DB sessions.
func TestRunAuthIssueAndRevokeSession(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "config.toml")

	var issuedOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "issue-session",
		"--principal-id", "agent-1",
		"--principal-type", "agent",
		"--principal-name", "Agent One",
		"--client-id", "till-mcp-stdio",
		"--client-type", "mcp-stdio",
		"--client-name", "Till MCP STDIO",
	}, &issuedOut, io.Discard); err != nil {
		t.Fatalf("run(auth issue-session) error = %v", err)
	}
	issueOutput := issuedOut.String()
	issuedSessionID := extractCLIKVValue(t, issueOutput, "session id")
	issuedSessionSecret := extractCLIKVValue(t, issueOutput, "session secret")
	if issuedSessionID == "" || issuedSessionSecret == "" {
		t.Fatalf("issue-session returned empty credentials: %q", issueOutput)
	}
	for _, want := range []string{
		"Auth Session",
		"Agent One [agent-1]",
		"Till MCP STDIO",
		"state",
		"active",
		"principal type",
		"agent",
	} {
		if !strings.Contains(issueOutput, want) {
			t.Fatalf("expected %q in issue-session output, got %q", want, issueOutput)
		}
	}
	if got := extractCLIKVValue(t, issueOutput, "expires"); got == "-" {
		t.Fatalf("issue-session expires = %q, want timestamp", got)
	}

	var revokedOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "revoke-session",
		"--session-id", issuedSessionID,
		"--reason", "operator_revoke",
	}, &revokedOut, io.Discard); err != nil {
		t.Fatalf("run(auth revoke-session) error = %v", err)
	}
	revokeOutput := revokedOut.String()
	if got := extractCLIKVValue(t, revokeOutput, "session id"); got != issuedSessionID {
		t.Fatalf("revoke-session session id = %q, want %q", got, issuedSessionID)
	}
	if got := extractCLIKVValue(t, revokeOutput, "revoked at"); got == "-" {
		t.Fatalf("revoke-session revoked at = %q, want timestamp", got)
	}
	if got := extractCLIKVValue(t, revokeOutput, "revocation reason"); got != "operator_revoke" {
		t.Fatalf("revoke-session reason = %q, want operator_revoke", got)
	}
}

// TestRunAuthRequestApproveLifecycle verifies the primary request/session CLI issues an approved session and validates it.
func TestRunAuthRequestApproveLifecycle(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "config.toml")
	seedProjectForAuthCLITest(t, dbPath, "p1")

	var createdOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "create",
		"--path", "project/p1",
		"--principal-id", "review-agent",
		"--principal-type", "agent",
		"--principal-role", "builder",
		"--client-id", "till-mcp-stdio",
		"--client-type", "mcp-stdio",
		"--reason", "manual MCP review",
		"--continuation-json", `{"resume_tool":"till.raise_attention_item","resume_path":"project/p1","resume":{"path":"project/p1","attempt":1,"tags":["auth","dogfood"]}}`,
	}, &createdOut, io.Discard); err != nil {
		t.Fatalf("run(auth request create) error = %v", err)
	}

	createdOutput := createdOut.String()
	createdID := extractCLIKVValue(t, createdOutput, "request id")
	if strings.Contains(createdOutput, "resume_tool") || strings.Contains(createdOutput, "resume_path") {
		t.Fatalf("create output leaked continuation metadata: %s", createdOutput)
	}
	if got := extractCLIKVValue(t, createdOutput, "state"); got != "pending" {
		t.Fatalf("create state = %q, want pending", got)
	}
	if got := extractCLIKVValue(t, createdOutput, "requested path"); got != "project/p1" {
		t.Fatalf("create path = %q, want project/p1", got)
	}
	if !strings.Contains(createdOutput, "review-agent • builder") {
		t.Fatalf("expected principal label in create output, got %q", createdOutput)
	}
	if got := extractCLIKVValue(t, createdOutput, "has continuation"); got != "yes" {
		t.Fatalf("create has continuation = %q, want yes", got)
	}

	var shownOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "show",
		"--request-id", createdID,
	}, &shownOut, io.Discard); err != nil {
		t.Fatalf("run(auth request show) error = %v", err)
	}
	if strings.Contains(shownOut.String(), "resume_tool") || strings.Contains(shownOut.String(), "resume_path") {
		t.Fatalf("show output leaked continuation metadata: %s", shownOut.String())
	}
	for _, want := range []string{
		"Auth Request",
		createdID,
		"review-agent • builder",
		"requested path",
		"project/p1",
		"requested ttl",
		"8h",
		"requested by",
		"tillsyn-user (user)",
	} {
		if !strings.Contains(shownOut.String(), want) {
			t.Fatalf("expected %q in auth request show output, got %q", want, shownOut.String())
		}
	}

	var approvedOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "approve",
		"--request-id", createdID,
		"--path", "project/p1/branch/review-branch",
		"--ttl", "2h",
		"--note", "approved for dogfood",
	}, &approvedOut, io.Discard); err != nil {
		t.Fatalf("run(auth request approve) error = %v", err)
	}
	approvedOutput := approvedOut.String()
	if strings.Contains(approvedOutput, "resume_tool") || strings.Contains(approvedOutput, "resume_path") {
		t.Fatalf("approve output leaked continuation metadata: %s", approvedOutput)
	}
	if got := extractCLIKVValue(t, approvedOutput, "state"); got != "approved" {
		t.Fatalf("approve state = %q, want approved", got)
	}
	if got := extractCLIKVValue(t, approvedOutput, "requested path"); got != "project/p1" {
		t.Fatalf("approve requested path = %q, want project/p1", got)
	}
	if got := extractCLIKVValue(t, approvedOutput, "project"); got != "p1" {
		t.Fatalf("approve project_id = %q, want p1", got)
	}
	if got := extractCLIKVValue(t, approvedOutput, "approved path"); got != "project/p1/branch/review-branch" {
		t.Fatalf("approve approved_path = %q, want project/p1/branch/review-branch", got)
	}
	if got := extractCLIKVValue(t, approvedOutput, "requested ttl"); got != "8h" {
		t.Fatalf("approve requested ttl = %q, want 8h", got)
	}
	if got := extractCLIKVValue(t, approvedOutput, "approved ttl"); got != "2h" {
		t.Fatalf("approve approved ttl = %q, want 2h", got)
	}
	if got := extractCLIKVValue(t, approvedOutput, "has continuation"); got != "yes" {
		t.Fatalf("approve has continuation = %q, want yes", got)
	}
	approvedSessionID := extractCLIKVValue(t, approvedOutput, "issued session")
	approvedSessionSecret := extractCLIKVValue(t, approvedOutput, "issued session secret")
	if approvedSessionID == "" || approvedSessionSecret == "" {
		t.Fatalf("approve output missing issued credentials: %q", approvedOutput)
	}

	var approvedShowOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "show",
		"--request-id", createdID,
	}, &approvedShowOut, io.Discard); err != nil {
		t.Fatalf("run(auth request show approved) error = %v", err)
	}
	if strings.Contains(approvedShowOut.String(), "resume_tool") || strings.Contains(approvedShowOut.String(), "resume_path") {
		t.Fatalf("approved show output leaked continuation metadata: %s", approvedShowOut.String())
	}
	for _, want := range []string{
		"Auth Request",
		approvedSessionID,
		"approved path",
		"project/p1/branch/review-branch",
		"approved ttl",
		"2h",
	} {
		if !strings.Contains(approvedShowOut.String(), want) {
			t.Fatalf("expected %q in approved auth request show output, got %q", want, approvedShowOut.String())
		}
	}
	if strings.Contains(approvedShowOut.String(), approvedSessionSecret) {
		t.Fatalf("approved show output leaked issued session secret: %s", approvedShowOut.String())
	}

	var validatedOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "session", "validate",
		"--session-id", approvedSessionID,
		"--session-secret", approvedSessionSecret,
	}, &validatedOut, io.Discard); err != nil {
		t.Fatalf("run(auth session validate) error = %v", err)
	}
	validatedOutput := validatedOut.String()
	for _, want := range []string{
		"Auth Session",
		"review-agent • builder",
		"state",
		"active",
	} {
		if !strings.Contains(validatedOutput, want) {
			t.Fatalf("expected %q in auth session validate output, got %q", want, validatedOutput)
		}
	}
	if got := extractCLIKVValue(t, validatedOutput, "project"); got != "p1" {
		t.Fatalf("validate project = %q, want p1", got)
	}
	if got := extractCLIKVValue(t, validatedOutput, "auth request"); got != createdID {
		t.Fatalf("validate auth request = %q, want %q", got, createdID)
	}
	if got := extractCLIKVValue(t, validatedOutput, "approved path"); got != "project/p1/branch/review-branch" {
		t.Fatalf("validate approved_path = %q, want project/p1/branch/review-branch", got)
	}

	var sessionListOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "session", "list",
		"--project-id", "p1",
		"--principal-id", "review-agent",
	}, &sessionListOut, io.Discard); err != nil {
		t.Fatalf("run(auth session list) error = %v", err)
	}
	for _, want := range []string{
		"Auth Sessions",
		approvedSessionID,
		"review-agent • builder",
		"project/p1/branch/review-branch",
		"p1",
		"active",
	} {
		if !strings.Contains(sessionListOut.String(), want) {
			t.Fatalf("expected %q in auth session list output, got %q", want, sessionListOut.String())
		}
	}

	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "session", "list",
		"--state", "definitely-invalid",
	}, io.Discard, io.Discard); err == nil {
		t.Fatal("run(auth session list invalid state) error = nil, want validation failure")
	}
}

// TestRunAuthRequestTerminalStatesAndFilters verifies deny and cancel flows land in explicit stored states.
func TestRunAuthRequestTerminalStatesAndFilters(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "config.toml")
	seedProjectForAuthCLITest(t, dbPath, "p1")

	createRequest := func(principalID string) string {
		t.Helper()
		var out strings.Builder
		if err := run(context.Background(), []string{
			"--db", dbPath,
			"--config", cfgPath,
			"auth", "request", "create",
			"--path", "project/p1",
			"--principal-id", principalID,
			"--client-id", "till-tui",
			"--client-type", "tui",
			"--reason", "review access",
		}, &out, io.Discard); err != nil {
			t.Fatalf("run(auth request create %q) error = %v", principalID, err)
		}
		return extractCLIKVValue(t, out.String(), "request id")
	}

	deniedRequestID := createRequest("user-deny")
	var deniedOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "deny",
		"--request-id", deniedRequestID,
		"--note", "outside current scope",
	}, &deniedOut, io.Discard); err != nil {
		t.Fatalf("run(auth request deny) error = %v", err)
	}
	if got := extractCLIKVValue(t, deniedOut.String(), "state"); got != "denied" {
		t.Fatalf("deny state = %q, want denied", got)
	}

	canceledRequestID := createRequest("user-cancel")
	var canceledOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "cancel",
		"--request-id", canceledRequestID,
		"--note", "superseded by another request",
	}, &canceledOut, io.Discard); err != nil {
		t.Fatalf("run(auth request cancel) error = %v", err)
	}
	if got := extractCLIKVValue(t, canceledOut.String(), "state"); got != "canceled" {
		t.Fatalf("cancel state = %q, want canceled", got)
	}

	var deniedListOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "list",
		"--state", "denied",
	}, &deniedListOut, io.Discard); err != nil {
		t.Fatalf("run(auth request list denied) error = %v", err)
	}
	for _, want := range []string{"Auth Requests", deniedRequestID, "denied", "user-deny"} {
		if !strings.Contains(deniedListOut.String(), want) {
			t.Fatalf("expected %q in denied auth request list output, got %q", want, deniedListOut.String())
		}
	}

	var canceledListOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "list",
		"--state", "canceled",
	}, &canceledListOut, io.Discard); err != nil {
		t.Fatalf("run(auth request list canceled) error = %v", err)
	}
	for _, want := range []string{"Auth Requests", canceledRequestID, "canceled", "user-cancel"} {
		if !strings.Contains(canceledListOut.String(), want) {
			t.Fatalf("expected %q in canceled auth request list output, got %q", want, canceledListOut.String())
		}
	}
}

// TestRunAuthRequestTimeoutMaterializesExpiredState verifies request show surfaces the timeout lifecycle explicitly.
func TestRunAuthRequestTimeoutMaterializesExpiredState(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "config.toml")
	seedProjectForAuthCLITest(t, dbPath, "p1")

	var createdOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "create",
		"--path", "project/p1",
		"--principal-id", "review-user",
		"--client-id", "till-tui",
		"--client-type", "tui",
		"--timeout", "1ms",
		"--reason", "brief review",
	}, &createdOut, io.Discard); err != nil {
		t.Fatalf("run(auth request create timeout) error = %v", err)
	}

	createdID := extractCLIKVValue(t, createdOut.String(), "request id")

	time.Sleep(10 * time.Millisecond)

	var shownOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "show",
		"--request-id", createdID,
	}, &shownOut, io.Discard); err != nil {
		t.Fatalf("run(auth request show timeout) error = %v", err)
	}
	for _, want := range []string{"Auth Request", "expired", "timed_out", createdID} {
		if !strings.Contains(shownOut.String(), want) {
			t.Fatalf("expected %q in timeout auth request show output, got %q", want, shownOut.String())
		}
	}
}

// TestRunAuthIssueSessionCredentialsAuthorizeMutation verifies CLI-issued credentials are usable through the auth-backed mutation adapter seam.
func TestRunAuthIssueSessionCredentialsAuthorizeMutation(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "config.toml")

	var issuedOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "issue-session",
		"--principal-id", "agent-1",
		"--principal-type", "agent",
		"--client-id", "till-mcp-stdio",
		"--client-type", "mcp-stdio",
	}, &issuedOut, io.Discard); err != nil {
		t.Fatalf("run(auth issue-session) error = %v", err)
	}

	issuedSessionID := extractCLIKVValue(t, issuedOut.String(), "session id")
	issuedSessionSecret := extractCLIKVValue(t, issuedOut.String(), "session secret")

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	authService, err := autentauth.NewSharedDB(autentauth.Config{DB: repo.DB()})
	if err != nil {
		t.Fatalf("NewSharedDB() error = %v", err)
	}
	if err := authService.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	auth, err := servercommon.NewAppServiceAdapter(nil, authService).AuthorizeMutation(context.Background(), servercommon.MutationAuthorizationRequest{
		SessionID:     issuedSessionID,
		SessionSecret: issuedSessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if err != nil {
		t.Fatalf("AuthorizeMutation() error = %v", err)
	}
	if auth.PrincipalID != "agent-1" {
		t.Fatalf("AuthorizeMutation() principal_id = %q, want agent-1", auth.PrincipalID)
	}
	if auth.PrincipalType != domain.ActorTypeAgent {
		t.Fatalf("AuthorizeMutation() principal_type = %q, want agent", auth.PrincipalType)
	}
}

// TestAuthorizeMutationRevokedSessionReturnsInvalidAuthentication verifies revoked sessions fail closed in the real auth-backed adapter path.
func TestAuthorizeMutationRevokedSessionReturnsInvalidAuthentication(t *testing.T) {
	adapter, auth := newAuthAdapterForTest(t)
	if err := auth.EnsureDogfoodPolicy(context.Background()); err != nil {
		t.Fatalf("EnsureDogfoodPolicy() error = %v", err)
	}
	sessionID, sessionSecret := mustIssueUserSessionForAdapterTest(t, auth)
	if _, err := auth.RevokeSession(context.Background(), sessionID, "operator_revoke"); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}

	_, err := adapter.AuthorizeMutation(context.Background(), servercommon.MutationAuthorizationRequest{
		SessionID:     sessionID,
		SessionSecret: sessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if !errors.Is(err, servercommon.ErrInvalidAuthentication) {
		t.Fatalf("AuthorizeMutation() error = %v, want ErrInvalidAuthentication", err)
	}
}

// TestAuthorizeMutationDenyRuleReturnsAuthorizationDenied verifies real deny decisions map through the adapter boundary.
func TestAuthorizeMutationDenyRuleReturnsAuthorizationDenied(t *testing.T) {
	adapter, auth := newAuthAdapterForTest(t)
	if err := auth.ReplaceRules(context.Background(), []autentdomain.Rule{
		mustNormalizeAuthRuleForTest(t, autentdomain.Rule{
			ID:     "deny-create-task",
			Effect: autentdomain.EffectDeny,
			Actions: []autentdomain.StringPattern{
				{Operator: autentdomain.MatchExact, Value: "create_task"},
			},
			Resources: []autentdomain.ResourcePattern{
				{
					Namespace: autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "project:p1"},
					Type:      autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "task"},
					ID:        autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "new"},
				},
			},
			Priority: 10,
		}),
	}); err != nil {
		t.Fatalf("ReplaceRules() error = %v", err)
	}
	sessionID, sessionSecret := mustIssueUserSessionForAdapterTest(t, auth)

	_, err := adapter.AuthorizeMutation(context.Background(), servercommon.MutationAuthorizationRequest{
		SessionID:     sessionID,
		SessionSecret: sessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if !errors.Is(err, servercommon.ErrAuthorizationDenied) {
		t.Fatalf("AuthorizeMutation() error = %v, want ErrAuthorizationDenied", err)
	}
}

// TestAuthorizeMutationGrantRequiredReturnsGrantRequired verifies real grant-required decisions map through the adapter boundary.
func TestAuthorizeMutationGrantRequiredReturnsGrantRequired(t *testing.T) {
	adapter, auth := newAuthAdapterForTest(t)
	if err := auth.ReplaceRules(context.Background(), []autentdomain.Rule{
		mustNormalizeAuthRuleForTest(t, autentdomain.Rule{
			ID:     "grant-create-task",
			Effect: autentdomain.EffectAllow,
			Actions: []autentdomain.StringPattern{
				{Operator: autentdomain.MatchExact, Value: "create_task"},
			},
			Resources: []autentdomain.ResourcePattern{
				{
					Namespace: autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "project:p1"},
					Type:      autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "task"},
					ID:        autentdomain.StringPattern{Operator: autentdomain.MatchExact, Value: "new"},
				},
			},
			Escalation: &autentdomain.EscalationRequirement{Allowed: true},
			Priority:   10,
		}),
	}); err != nil {
		t.Fatalf("ReplaceRules() error = %v", err)
	}
	sessionID, sessionSecret := mustIssueUserSessionForAdapterTest(t, auth)

	_, err := adapter.AuthorizeMutation(context.Background(), servercommon.MutationAuthorizationRequest{
		SessionID:     sessionID,
		SessionSecret: sessionSecret,
		Action:        "create_task",
		Namespace:     "project:p1",
		ResourceType:  "task",
		ResourceID:    "new",
	})
	if !errors.Is(err, servercommon.ErrGrantRequired) {
		t.Fatalf("AuthorizeMutation() error = %v, want ErrGrantRequired", err)
	}
}

// TestRunCaptureStateCommand verifies the new capture-state CLI surface returns stable recovery JSON.
func TestRunCaptureStateCommand(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")

	var out strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "capture-state", "--project-id", "p1"}, &out, io.Discard); err != nil {
		t.Fatalf("run(capture-state) error = %v", err)
	}

	var got struct {
		RequestedView      string `json:"requested_view"`
		RequestedScopeType string `json:"requested_scope_type"`
		StateHash          string `json:"state_hash"`
		GoalOverview       struct {
			ProjectID string `json:"project_id"`
		} `json:"goal_overview"`
		ScopePath []struct {
			ScopeType string `json:"scope_type"`
			ScopeID   string `json:"scope_id"`
		} `json:"scope_path"`
	}
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("Unmarshal(capture-state) error = %v", err)
	}
	if got.RequestedView != "summary" {
		t.Fatalf("requested_view = %q, want summary", got.RequestedView)
	}
	if got.RequestedScopeType != "project" {
		t.Fatalf("requested_scope_type = %q, want project", got.RequestedScopeType)
	}
	if got.GoalOverview.ProjectID != "p1" {
		t.Fatalf("goal_overview.project_id = %q, want p1", got.GoalOverview.ProjectID)
	}
	if got.StateHash == "" {
		t.Fatal("expected state hash in capture-state output")
	}
	if len(got.ScopePath) == 0 || got.ScopePath[0].ScopeType != "project" || got.ScopePath[0].ScopeID != "p1" {
		t.Fatalf("scope_path = %#v, want project:p1", got.ScopePath)
	}
}

// TestRunProjectScopeGuidance verifies scoped commands point operators to project discovery when the project id is missing.
func TestRunProjectScopeGuidance(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "capture-state",
			args: []string{"--db", dbPath, "--config", cfgPath, "capture-state"},
		},
		{
			name: "project show",
			args: []string{"--db", dbPath, "--config", cfgPath, "project", "show"},
		},
		{
			name: "project discover",
			args: []string{"--db", dbPath, "--config", cfgPath, "project", "discover"},
		},
		{
			name: "kind allowlist list",
			args: []string{"--db", dbPath, "--config", cfgPath, "kind", "allowlist", "list"},
		},
		{
			name: "kind allowlist set",
			args: []string{"--db", dbPath, "--config", cfgPath, "kind", "allowlist", "set"},
		},
		{
			name: "lease list",
			args: []string{"--db", dbPath, "--config", cfgPath, "lease", "list"},
		},
		{
			name: "lease issue",
			args: []string{"--db", dbPath, "--config", cfgPath, "lease", "issue"},
		},
		{
			name: "lease revoke-all",
			args: []string{"--db", dbPath, "--config", cfgPath, "lease", "revoke-all"},
		},
		{
			name: "handoff create",
			args: []string{"--db", dbPath, "--config", cfgPath, "handoff", "create"},
		},
		{
			name: "handoff list",
			args: []string{"--db", dbPath, "--config", cfgPath, "handoff", "list"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			err := run(context.Background(), tc.args, &out, io.Discard)
			if err == nil {
				t.Fatal("expected missing project-id error")
			}
			if !strings.Contains(strings.ToLower(err.Error()), "project list") {
				t.Fatalf("expected discoverability hint in error, got %v", err)
			}
		})
	}
}

// TestRunProjectCommands verifies project discovery, create, and show output.
func TestRunProjectCommands(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")
	seedTemplateLibraryForProjectCreateCLITest(t, dbPath)

	var createOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"project", "create",
		"--name", "Inbox",
		"--description", "Local execution inbox",
		"--metadata-json", `{"owner":"Platform","tags":["dogfood"]}`,
	}, &createOut, io.Discard); err != nil {
		t.Fatalf("run(project create) error = %v", err)
	}
	if got := createOut.String(); !strings.Contains(got, "Created Project") || !strings.Contains(got, "name") || !strings.Contains(got, "Inbox") || !strings.Contains(got, "owner") || !strings.Contains(got, "Platform") {
		t.Fatalf("unexpected project create output: %q", got)
	}

	var createPositionalOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"project", "create", "Roadmap",
	}, &createPositionalOut, io.Discard); err != nil {
		t.Fatalf("run(project create positional) error = %v", err)
	}
	if got := createPositionalOut.String(); !strings.Contains(got, "Created Project") || !strings.Contains(got, "Roadmap") {
		t.Fatalf("unexpected positional project create output: %q", got)
	}

	var listOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "list"}, &listOut, io.Discard); err != nil {
		t.Fatalf("run(project list) error = %v", err)
	}
	if got := listOut.String(); !strings.Contains(got, "NAME") || !strings.Contains(got, "ID") || !strings.Contains(got, "OWNER") || !strings.Contains(got, "ARCHIVED") || !strings.Contains(got, "Project p1") || !strings.Contains(got, "Inbox") || !strings.Contains(got, "Roadmap") {
		t.Fatalf("unexpected project list output: %q", got)
	}

	var showOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "show", "--project-id", "p1"}, &showOut, io.Discard); err != nil {
		t.Fatalf("run(project show) error = %v", err)
	}
	if got := showOut.String(); !strings.Contains(got, "Project") || !strings.Contains(got, "name") || !strings.Contains(got, "Project p1") || !strings.Contains(got, "id") || !strings.Contains(got, "p1") {
		t.Fatalf("unexpected project show output: %q", got)
	}

	var showPositionalOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "show", "p1"}, &showPositionalOut, io.Discard); err != nil {
		t.Fatalf("run(project show positional) error = %v", err)
	}
	if got := showPositionalOut.String(); !strings.Contains(got, "Project") || !strings.Contains(got, "Project p1") {
		t.Fatalf("unexpected positional project show output: %q", got)
	}

	var requestOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"auth", "request", "create",
		"--path", "project/p1",
		"--principal-id", "review-agent",
		"--principal-type", "agent",
		"--principal-role", "builder",
		"--client-id", "till-mcp-stdio",
		"--client-type", "mcp-stdio",
		"--reason", "collaboration setup",
	}, &requestOut, io.Discard); err != nil {
		t.Fatalf("run(auth request create) error = %v", err)
	}

	var discoverOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "discover", "--project-id", "p1"}, &discoverOut, io.Discard); err != nil {
		t.Fatalf("run(project discover) error = %v", err)
	}
	gotDiscover := discoverOut.String()
	for _, want := range []string{"Project Collaboration Readiness", "Coordination Inventory", "pending_auth_requests", "till auth request show --request-id"} {
		if !strings.Contains(gotDiscover, want) {
			t.Fatalf("expected %q in project discover output, got %q", want, gotDiscover)
		}
	}

	var discoverPositionalOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "discover", "p1"}, &discoverPositionalOut, io.Discard); err != nil {
		t.Fatalf("run(project discover positional) error = %v", err)
	}
	if got := discoverPositionalOut.String(); !strings.Contains(got, "Project Collaboration Readiness") || !strings.Contains(got, "Project p1") {
		t.Fatalf("unexpected positional project discover output: %q", got)
	}
}

// TestRunProjectCommandsMuteRuntimeConsoleLogs verifies human-facing project commands stay quiet on stderr.
func TestRunProjectCommandsMuteRuntimeConsoleLogs(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")

	var stdout strings.Builder
	var stderr strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "list"}, &stdout, &stderr); err != nil {
		t.Fatalf("run(project list) error = %v", err)
	}
	if got := stderr.String(); strings.TrimSpace(got) != "" {
		t.Fatalf("expected quiet stderr for project list, got %q", got)
	}
	if got := stdout.String(); !strings.Contains(got, "Project p1") {
		t.Fatalf("unexpected project list output: %q", got)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "show", "p1"}, &stdout, &stderr); err != nil {
		t.Fatalf("run(project show positional) error = %v", err)
	}
	if got := stderr.String(); strings.TrimSpace(got) != "" {
		t.Fatalf("expected quiet stderr for project show, got %q", got)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "discover", "p1"}, &stdout, &stderr); err != nil {
		t.Fatalf("run(project discover positional) error = %v", err)
	}
	if got := stderr.String(); strings.TrimSpace(got) != "" {
		t.Fatalf("expected quiet stderr for project discover, got %q", got)
	}
}

// TestRunProjectCreateMissingNameGuidance keeps the current CLI path explicit until guided creation lands later.
func TestRunProjectCreateMissingNameGuidance(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)

	var out strings.Builder
	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "create"}, &out, io.Discard)
	if err == nil {
		t.Fatal("expected missing project name error")
	}
	for _, want := range []string{"project name is required", "--name", "till project create --help"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in project create guidance, got %v", want, err)
		}
	}
}

// TestRunProjectCommandConflictingInputs rejects mismatched flag and positional values.
func TestRunProjectCommandConflictingInputs(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")

	var out strings.Builder
	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "show", "--project-id", "p1", "p2"}, &out, io.Discard)
	if err == nil {
		t.Fatal("expected conflicting project show inputs error")
	}
	if !strings.Contains(err.Error(), "either --project-id or one positional project id") {
		t.Fatalf("unexpected project show conflicting-input error: %v", err)
	}

	out.Reset()
	err = run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "create", "--name", "Inbox", "Roadmap"}, &out, io.Discard)
	if err == nil {
		t.Fatal("expected conflicting project create inputs error")
	}
	if !strings.Contains(err.Error(), "either --name or one positional project name") {
		t.Fatalf("unexpected project create conflicting-input error: %v", err)
	}
}

// TestShouldMuteRuntimeConsole keeps one-shot commands quiet while daemon commands remain noisy.
func TestShouldMuteRuntimeConsole(t *testing.T) {
	cases := []struct {
		command string
		want    bool
	}{
		{command: "", want: true},
		{command: "project.list", want: true},
		{command: "capture-state", want: true},
		{command: "mcp", want: false},
		{command: "serve", want: false},
	}

	for _, tc := range cases {
		if got := shouldMuteRuntimeConsole(tc.command); got != tc.want {
			t.Fatalf("shouldMuteRuntimeConsole(%q) = %v, want %v", tc.command, got, tc.want)
		}
	}
}

// TestRunProjectListDoesNotUseInterruptEchoSuppression keeps one-shot operator commands off the daemon-only terminal wrapper path.
func TestRunProjectListDoesNotUseInterruptEchoSuppression(t *testing.T) {
	origWrapper := withInterruptEchoSuppressedFunc
	t.Cleanup(func() { withInterruptEchoSuppressedFunc = origWrapper })

	var calls int
	withInterruptEchoSuppressedFunc = func(runFn func() error) error {
		calls++
		if runFn == nil {
			return nil
		}
		return runFn()
	}

	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "list"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(project list) error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("withInterruptEchoSuppressedFunc calls = %d, want 0", calls)
	}
}

// TestRunProjectListArchivedOnlyGuidance points operators toward archived discovery before duplicate creation.
func TestRunProjectListArchivedOnlyGuidance(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")
	archiveProjectForCLITest(t, dbPath, "p1")

	var out strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "list"}, &out, io.Discard); err != nil {
		t.Fatalf("run(project list archived-only) error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "No projects found.") || !strings.Contains(got, "till project list --include-archived") {
		t.Fatalf("expected archived-only guidance, got %q", got)
	}
}

// TestRunProjectShowArchivedGuidance points operators toward include-archived when the id exists but is hidden.
func TestRunProjectShowArchivedGuidance(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")
	archiveProjectForCLITest(t, dbPath, "p1")

	var out strings.Builder
	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "show", "--project-id", "p1"}, &out, io.Discard)
	if err == nil {
		t.Fatal("expected archived project show error")
	}
	if !strings.Contains(err.Error(), "--include-archived") {
		t.Fatalf("expected archived project guidance, got %v", err)
	}
}

// TestRunProjectDiscoverArchivedGuidance points operators toward include-archived when discover targets a hidden archived project.
func TestRunProjectDiscoverArchivedGuidance(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")
	archiveProjectForCLITest(t, dbPath, "p1")

	var out strings.Builder
	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "project", "discover", "--project-id", "p1"}, &out, io.Discard)
	if err == nil {
		t.Fatal("expected archived project discover error")
	}
	if !strings.Contains(err.Error(), "--include-archived") {
		t.Fatalf("expected archived project discover guidance, got %v", err)
	}
}

// TestRunKindAndAllowlistCommands verifies kind upsert/list and project allowlist updates.
func TestRunKindAndAllowlistCommands(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")

	var upsertOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"kind", "upsert",
		"--id", "qa-check",
		"--display-name", "QA Check",
		"--applies-to", "task",
		"--template-json", "{}",
	}, &upsertOut, io.Discard); err != nil {
		t.Fatalf("run(kind upsert) error = %v", err)
	}
	var kind struct {
		ID          string   `json:"id"`
		DisplayName string   `json:"display_name"`
		AppliesTo   []string `json:"applies_to"`
	}
	if err := json.Unmarshal([]byte(upsertOut.String()), &kind); err != nil {
		t.Fatalf("Unmarshal(kind upsert) error = %v", err)
	}
	if kind.ID != "qa-check" || kind.DisplayName != "QA Check" {
		t.Fatalf("kind upsert output = %#v, want qa-check/QA Check", kind)
	}
	if strings.Contains(upsertOut.String(), "agents_file_sections") {
		t.Fatalf("kind upsert output still contains legacy agents_file_sections key: %s", upsertOut.String())
	}
	if strings.Contains(upsertOut.String(), "claude_file_sections") {
		t.Fatalf("kind upsert output still contains legacy claude_file_sections key: %s", upsertOut.String())
	}

	var listOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "kind", "list"}, &listOut, io.Discard); err != nil {
		t.Fatalf("run(kind list) error = %v", err)
	}
	var kinds []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(listOut.String()), &kinds); err != nil {
		t.Fatalf("Unmarshal(kind list) error = %v", err)
	}
	foundKind := false
	for _, item := range kinds {
		if item.ID == "qa-check" {
			foundKind = true
			break
		}
	}
	if !foundKind {
		t.Fatalf("expected kind list to include qa-check, got %#v", kinds)
	}

	var allowSetOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"kind", "allowlist", "set",
		"--project-id", "p1",
		"--kind-id", "qa-check",
	}, &allowSetOut, io.Discard); err != nil {
		t.Fatalf("run(kind allowlist set) error = %v", err)
	}
	var allowSet struct {
		ProjectID string   `json:"project_id"`
		KindIDs   []string `json:"kind_ids"`
	}
	if err := json.Unmarshal([]byte(allowSetOut.String()), &allowSet); err != nil {
		t.Fatalf("Unmarshal(kind allowlist set) error = %v", err)
	}
	if allowSet.ProjectID != "p1" || len(allowSet.KindIDs) != 1 || allowSet.KindIDs[0] != "qa-check" {
		t.Fatalf("allowlist set output = %#v, want p1/qa-check", allowSet)
	}

	var allowListOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "kind", "allowlist", "list", "--project-id", "p1"}, &allowListOut, io.Discard); err != nil {
		t.Fatalf("run(kind allowlist list) error = %v", err)
	}
	var allowList struct {
		ProjectID string   `json:"project_id"`
		KindIDs   []string `json:"kind_ids"`
	}
	if err := json.Unmarshal([]byte(allowListOut.String()), &allowList); err != nil {
		t.Fatalf("Unmarshal(kind allowlist list) error = %v", err)
	}
	if allowList.ProjectID != "p1" || len(allowList.KindIDs) != 1 || allowList.KindIDs[0] != "qa-check" {
		t.Fatalf("allowlist list output = %#v, want p1/qa-check", allowList)
	}
}

// TestRunTemplateLibraryCommands verifies template library upsert/list/show, project binding, and contract lookup.
func TestRunTemplateLibraryCommands(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")

	for _, args := range [][]string{
		{"kind", "upsert", "--id", "build-task", "--display-name", "Build Task", "--applies-to", "task"},
		{"kind", "upsert", "--id", "qa-pass", "--display-name", "QA Pass", "--applies-to", "subtask"},
	} {
		if err := run(context.Background(), append([]string{"--db", dbPath, "--config", cfgPath}, args...), io.Discard, io.Discard); err != nil {
			t.Fatalf("run(%v) error = %v", args, err)
		}
	}

	specJSON := strings.TrimSpace(`{
	  "id": "go-defaults",
	  "scope": "global",
	  "name": "Go Defaults",
	  "status": "approved",
	  "node_templates": [
	    {
	      "id": "tmpl-build-task",
	      "scope_level": "task",
	      "node_kind_id": "build-task",
	      "display_name": "Build Task",
	      "child_rules": [
	        {
	          "id": "rule-qa-pass",
	          "position": 10,
	          "child_scope_level": "subtask",
	          "child_kind_id": "qa-pass",
	          "title_template": "QA Pass",
	          "responsible_actor_kind": "qa",
	          "editable_by_actor_kinds": ["qa"],
	          "completable_by_actor_kinds": ["qa"],
	          "required_for_parent_done": true
	        }
	      ]
	    }
	  ]
	}`)

	var upsertOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"template", "library", "upsert",
		"--spec-json", specJSON,
	}, &upsertOut, io.Discard); err != nil {
		t.Fatalf("run(template library upsert) error = %v", err)
	}
	upsertOutput := upsertOut.String()
	if got := extractCLIKVValue(t, upsertOutput, "id"); got != "go-defaults" {
		t.Fatalf("template library upsert id = %q, want go-defaults", got)
	}
	if got := extractCLIKVValue(t, upsertOutput, "name"); got != "Go Defaults" {
		t.Fatalf("template library upsert name = %q, want Go Defaults", got)
	}
	for _, want := range []string{"Template Library", "Node Templates", "Template Child Rules", "Build Task", "QA Pass"} {
		if !strings.Contains(upsertOutput, want) {
			t.Fatalf("expected %q in template library upsert output, got %q", want, upsertOutput)
		}
	}

	var listOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"template", "library", "list",
		"--scope", "global",
		"--status", "approved",
	}, &listOut, io.Discard); err != nil {
		t.Fatalf("run(template library list) error = %v", err)
	}
	for _, want := range []string{"Template Libraries", "go-defaults", "Go Defaults", "global", "approved"} {
		if !strings.Contains(listOut.String(), want) {
			t.Fatalf("expected %q in template library list output, got %q", want, listOut.String())
		}
	}

	var showOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"template", "library", "show",
		"--library-id", "go-defaults",
	}, &showOut, io.Discard); err != nil {
		t.Fatalf("run(template library show) error = %v", err)
	}
	showOutput := showOut.String()
	if got := extractCLIKVValue(t, showOutput, "id"); got != "go-defaults" {
		t.Fatalf("template library show id = %q, want go-defaults", got)
	}
	for _, want := range []string{"Template Library", "Node Templates", "Build Task", "Template Child Rules", "QA Pass"} {
		if !strings.Contains(showOutput, want) {
			t.Fatalf("expected %q in template library show output, got %q", want, showOutput)
		}
	}

	var bindOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"template", "project", "bind",
		"--project-id", "p1",
		"--library-id", "go-defaults",
	}, &bindOut, io.Discard); err != nil {
		t.Fatalf("run(template project bind) error = %v", err)
	}
	bindOutput := bindOut.String()
	if got := extractCLIKVValue(t, bindOutput, "project id"); got != "p1" {
		t.Fatalf("template project bind project = %q, want p1", got)
	}
	if got := extractCLIKVValue(t, bindOutput, "library id"); got != "go-defaults" {
		t.Fatalf("template project bind library = %q, want go-defaults", got)
	}

	var bindingOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"template", "project", "binding",
		"--project-id", "p1",
	}, &bindingOut, io.Discard); err != nil {
		t.Fatalf("run(template project binding) error = %v", err)
	}
	bindingOutput := bindingOut.String()
	if got := extractCLIKVValue(t, bindingOutput, "library id"); got != "go-defaults" {
		t.Fatalf("template project binding library = %q, want go-defaults", got)
	}

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	defer func() {
		_ = repo.Close()
	}()
	column, err := domain.NewColumn("c1", "p1", "To Do", 0, 0, time.Date(2026, 3, 29, 12, 55, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(context.Background(), column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	task, err := domain.NewTask(domain.TaskInput{
		ID:        "task-qa-1",
		ProjectID: "p1",
		ColumnID:  "c1",
		Position:  0,
		Title:     "QA Pass",
		Kind:      domain.WorkKindTask,
		Scope:     domain.KindAppliesToTask,
		Priority:  domain.PriorityMedium,
	}, time.Date(2026, 3, 29, 12, 58, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if err := repo.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	snapshot, err := domain.NewNodeContractSnapshot(domain.NodeContractSnapshotInput{
		NodeID:                  "task-qa-1",
		ProjectID:               "p1",
		SourceLibraryID:         "go-defaults",
		SourceNodeTemplateID:    "tmpl-build-task",
		SourceChildRuleID:       "rule-qa-pass",
		ResponsibleActorKind:    domain.TemplateActorKindQA,
		EditableByActorKinds:    []domain.TemplateActorKind{domain.TemplateActorKindQA},
		CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
		RequiredForParentDone:   true,
	}, time.Date(2026, 3, 29, 13, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewNodeContractSnapshot() error = %v", err)
	}
	if err := repo.CreateNodeContractSnapshot(context.Background(), snapshot); err != nil {
		t.Fatalf("CreateNodeContractSnapshot() error = %v", err)
	}

	var contractOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"template", "contract", "show",
		"--node-id", "task-qa-1",
	}, &contractOut, io.Discard); err != nil {
		t.Fatalf("run(template contract show) error = %v", err)
	}
	contractOutput := contractOut.String()
	if got := extractCLIKVValue(t, contractOutput, "source library"); got != "go-defaults" {
		t.Fatalf("template contract source library = %q, want go-defaults", got)
	}
	if got := extractCLIKVValue(t, contractOutput, "responsible actor"); got != "qa" {
		t.Fatalf("template contract responsible actor = %q, want qa", got)
	}
}

// TestRunTemplateLibraryUpsertAcceptsSnakeCaseProjectMetadata verifies project metadata defaults accept snake_case JSON keys.
func TestRunTemplateLibraryUpsertAcceptsSnakeCaseProjectMetadata(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)

	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"kind", "upsert",
		"--id", "go-service",
		"--display-name", "Go Service",
		"--applies-to", "project",
	}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(kind upsert go-service) error = %v", err)
	}

	specJSON := strings.TrimSpace(`{
	  "id": "go-defaults",
	  "scope": "global",
	  "name": "Go Defaults",
	  "status": "approved",
	  "node_templates": [
	    {
	      "id": "project-template",
	      "scope_level": "project",
	      "node_kind_id": "go-service",
	      "display_name": "Go Service Project",
	      "project_metadata_defaults": {
	        "owner": "Platform",
	        "standards_markdown": "Run Go validation"
	      }
	    }
	  ]
	}`)

	var upsertOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"template", "library", "upsert",
		"--spec-json", specJSON,
	}, &upsertOut, io.Discard); err != nil {
		t.Fatalf("run(template library upsert snake_case metadata) error = %v", err)
	}
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	defer func() {
		_ = repo.Close()
	}()
	library, err := repo.GetTemplateLibrary(context.Background(), "go-defaults")
	if err != nil {
		t.Fatalf("GetTemplateLibrary(go-defaults) error = %v", err)
	}
	if len(library.NodeTemplates) != 1 || library.NodeTemplates[0].ProjectMetadataDefaults == nil {
		t.Fatalf("template library output = %#v, want one project metadata default", library)
	}
	if got := library.NodeTemplates[0].ProjectMetadataDefaults.Owner; got != "Platform" {
		t.Fatalf("project metadata owner = %q, want Platform", got)
	}
	if got := library.NodeTemplates[0].ProjectMetadataDefaults.StandardsMarkdown; got != "Run Go validation" {
		t.Fatalf("project metadata standards_markdown = %q, want Run Go validation", got)
	}
}

// TestRunCapabilityLeaseCommands verifies issue/list/revoke lease flows on the new CLI surface.
func TestRunCapabilityLeaseCommands(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")

	var issueOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"lease", "issue",
		"--project-id", "p1",
		"--agent-name", "lane-a",
		"--role", "builder",
	}, &issueOut, io.Discard); err != nil {
		t.Fatalf("run(lease issue) error = %v", err)
	}
	issuedOutput := issueOut.String()
	issuedLeaseID := extractCLIKVValue(t, issuedOutput, "id")
	if issuedLeaseID == "" {
		t.Fatalf("lease issue output missing lease id: %q", issuedOutput)
	}
	for _, want := range []string{"Capability Lease", "lane-a", "builder", "project/p1", "active"} {
		if !strings.Contains(issuedOutput, want) {
			t.Fatalf("expected %q in lease issue output, got %q", want, issuedOutput)
		}
	}

	var listOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "lease", "list", "--project-id", "p1"}, &listOut, io.Discard); err != nil {
		t.Fatalf("run(lease list) error = %v", err)
	}
	for _, want := range []string{"Capability Leases", "lane-a", issuedLeaseID, "builder", "project/p1", "active"} {
		if !strings.Contains(listOut.String(), want) {
			t.Fatalf("expected %q in lease list output, got %q", want, listOut.String())
		}
	}

	var revokeOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"lease", "revoke",
		"--agent-instance-id", issuedLeaseID,
	}, &revokeOut, io.Discard); err != nil {
		t.Fatalf("run(lease revoke) error = %v", err)
	}
	revokeOutput := revokeOut.String()
	if got := extractCLIKVValue(t, revokeOutput, "id"); got != issuedLeaseID {
		t.Fatalf("lease revoke id = %q, want %q", got, issuedLeaseID)
	}
	if got := extractCLIKVValue(t, revokeOutput, "revoked"); got == "-" {
		t.Fatalf("lease revoke revoked timestamp = %q, want timestamp", got)
	}

	var revokedListOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "lease", "list", "--project-id", "p1", "--include-revoked"}, &revokedListOut, io.Discard); err != nil {
		t.Fatalf("run(lease list include revoked) error = %v", err)
	}
	for _, want := range []string{"Capability Leases", issuedLeaseID, "revoked"} {
		if !strings.Contains(revokedListOut.String(), want) {
			t.Fatalf("expected %q in revoked lease list output, got %q", want, revokedListOut.String())
		}
	}

	var issueSecondOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"lease", "issue",
		"--project-id", "p1",
		"--agent-name", "lane-b",
		"--role", "qa",
	}, &issueSecondOut, io.Discard); err != nil {
		t.Fatalf("run(second lease issue) error = %v", err)
	}

	var revokeAllOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"lease", "revoke-all",
		"--project-id", "p1",
		"--reason", "cleanup",
	}, &revokeAllOut, io.Discard); err != nil {
		t.Fatalf("run(lease revoke-all) error = %v", err)
	}
	for _, want := range []string{"Capability Lease Revocation", "project", "p1", "scope", "project/p1", "reason", "cleanup", "status", "revoked"} {
		if !strings.Contains(revokeAllOut.String(), want) {
			t.Fatalf("expected %q in lease revoke-all output, got %q", want, revokeAllOut.String())
		}
	}
}

// TestRunHandoffCommands verifies create/list/get/update flows on the new CLI surface.
func TestRunHandoffCommands(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	seedProjectForAuthCLITest(t, dbPath, "p1")

	var createOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"handoff", "create",
		"--project-id", "p1",
		"--summary", "qa handoff",
		"--source-role", "builder",
		"--target-role", "qa",
	}, &createOut, io.Discard); err != nil {
		t.Fatalf("run(handoff create) error = %v", err)
	}
	createdOutput := createOut.String()
	createdID := extractCLIKVValue(t, createdOutput, "id")
	if createdID == "" {
		t.Fatalf("handoff create output missing id: %q", createdOutput)
	}
	for _, want := range []string{"Handoff", "builder -> qa", "waiting", "qa handoff", "role:qa"} {
		if !strings.Contains(createdOutput, want) {
			t.Fatalf("expected %q in handoff create output, got %q", want, createdOutput)
		}
	}

	var listOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "handoff", "list", "--project-id", "p1"}, &listOut, io.Discard); err != nil {
		t.Fatalf("run(handoff list) error = %v", err)
	}
	for _, want := range []string{"Handoffs", createdID, "builder", "waiting", "qa handoff"} {
		if !strings.Contains(listOut.String(), "role:qa") {
			t.Fatalf("expected role-only handoff target in handoff list output, got %q", listOut.String())
		}
		if !strings.Contains(listOut.String(), want) {
			t.Fatalf("expected %q in handoff list output, got %q", want, listOut.String())
		}
	}

	var getOut strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "handoff", "get", "--handoff-id", createdID}, &getOut, io.Discard); err != nil {
		t.Fatalf("run(handoff get) error = %v", err)
	}
	for _, want := range []string{"Handoff", createdID, "builder -> qa", "role:qa", "qa handoff", "waiting"} {
		if !strings.Contains(getOut.String(), want) {
			t.Fatalf("expected %q in handoff get output, got %q", want, getOut.String())
		}
	}

	var updateOut strings.Builder
	if err := run(context.Background(), []string{
		"--db", dbPath,
		"--config", cfgPath,
		"handoff", "update",
		"--handoff-id", createdID,
		"--summary", "qa handoff",
		"--status", "resolved",
		"--resolution-note", "complete",
	}, &updateOut, io.Discard); err != nil {
		t.Fatalf("run(handoff update) error = %v", err)
	}
	updateOutput := updateOut.String()
	if got := extractCLIKVValue(t, updateOutput, "id"); got != createdID {
		t.Fatalf("handoff update id = %q, want %q", got, createdID)
	}
	if got := extractCLIKVValue(t, updateOutput, "status"); got != "resolved" {
		t.Fatalf("handoff update status = %q, want resolved", got)
	}
	if got := extractCLIKVValue(t, updateOutput, "resolved at"); got == "-" {
		t.Fatalf("handoff update resolved at = %q, want timestamp", got)
	}
}

// TestRunHelpPathsDoNotSeedMissingConfig verifies help flows remain side-effect free even when config is missing.
func TestRunHelpPathsDoNotSeedMissingConfig(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	writeConfigExample(t, workspace, "[identity]\ndisplay_name = \"\"\n")

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "root help",
			args: []string{"--db", filepath.Join(workspace, "root-help.db"), "--config", filepath.Join(workspace, "root-help.toml"), "--help"},
		},
		{
			name: "serve help",
			args: []string{"--db", filepath.Join(workspace, "serve-help.db"), "--config", filepath.Join(workspace, "serve-help.toml"), "serve", "--help"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := run(context.Background(), tc.args, &out, io.Discard); err != nil {
				t.Fatalf("run(%s) error = %v", tc.name, err)
			}

			// Help should render usage without executing runtime bootstrap side effects.
			cfgPath := tc.args[3]
			if _, err := os.Stat(cfgPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected help path to avoid seeding config %q, stat err = %v", cfgPath, err)
			}
		})
	}
}

// TestResolveRuntimePathsMCPUsesSharedDefaultRuntime verifies stdio MCP uses the same default runtime as the base app.
func TestResolveRuntimePathsMCPUsesSharedDefaultRuntime(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	out, err := resolveRuntimePaths("mcp", rootCommandOptions{appName: "tillsyn", devMode: true}, platform.Paths{
		ConfigPath: filepath.Join(workspace, "platform-config.toml"),
		DBPath:     filepath.Join(workspace, "platform.db"),
	})
	if err != nil {
		t.Fatalf("resolveRuntimePaths(mcp) error = %v", err)
	}
	if out.ConfigPath != filepath.Join(workspace, "platform-config.toml") {
		t.Fatalf("config path = %q, want shared platform config", out.ConfigPath)
	}
	if out.DBPath != filepath.Join(workspace, "platform.db") {
		t.Fatalf("db path = %q, want shared platform db", out.DBPath)
	}
}

// TestResolveRuntimePathsMCPConfigOverrideUsesSharedDBContract verifies stdio MCP honors the same config/db contract as the base app.
func TestResolveRuntimePathsMCPConfigOverrideUsesSharedDBContract(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	configOverride := filepath.Join(workspace, "custom-config.toml")
	out, err := resolveRuntimePaths("mcp", rootCommandOptions{
		appName:    "tillsyn",
		devMode:    false,
		configPath: configOverride,
	}, platform.Paths{
		ConfigPath: filepath.Join(workspace, "platform-config.toml"),
		DBPath:     filepath.Join(workspace, "platform.db"),
	})
	if err != nil {
		t.Fatalf("resolveRuntimePaths(mcp override config) error = %v", err)
	}
	if out.ConfigPath != configOverride {
		t.Fatalf("config path = %q, want %q", out.ConfigPath, configOverride)
	}
	if out.DBPath != filepath.Join(workspace, "platform.db") {
		t.Fatalf("db path = %q, want shared platform db", out.DBPath)
	}
}

// TestResolveRuntimePathsCommandsShareDefaultNonDevRuntime verifies root, mcp, and serve resolve the same non-dev default runtime.
func TestResolveRuntimePathsCommandsShareDefaultNonDevRuntime(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	defaultPaths := platform.Paths{
		ConfigPath: filepath.Join(workspace, "platform-config.toml"),
		DBPath:     filepath.Join(workspace, "platform.db"),
	}
	commands := []string{"", "mcp", "serve"}
	for _, command := range commands {
		command := command
		t.Run(firstNonEmpty(command, "root"), func(t *testing.T) {
			got, err := resolveRuntimePaths(command, rootCommandOptions{
				appName: "tillsyn",
				devMode: false,
			}, defaultPaths)
			if err != nil {
				t.Fatalf("resolveRuntimePaths(%q) error = %v", command, err)
			}
			if got.ConfigPath != defaultPaths.ConfigPath {
				t.Fatalf("config path = %q, want %q", got.ConfigPath, defaultPaths.ConfigPath)
			}
			if got.DBPath != defaultPaths.DBPath {
				t.Fatalf("db path = %q, want %q", got.DBPath, defaultPaths.DBPath)
			}
		})
	}
}

// TestRunMCPCommandWiresStdioAndSharedRuntime verifies the stdio MCP subcommand wires the adapter and shared runtime paths.
func TestRunMCPCommandWiresStdioAndSharedRuntime(t *testing.T) {
	origRunner := mcpCommandRunner
	t.Cleanup(func() { mcpCommandRunner = origRunner })

	var gotCfg serveradapter.Config
	var gotDeps serveradapter.Dependencies
	mcpCommandRunner = func(_ context.Context, cfg serveradapter.Config, deps serveradapter.Dependencies) error {
		gotCfg = cfg
		gotDeps = deps
		return nil
	}

	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	writeConfigExample(t, workspace, "[logging]\nlevel = \"debug\"\n")

	if err := run(context.Background(), []string{"--app", "tillsyn-mcp", "--dev", "mcp"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(mcp) error = %v", err)
	}

	if gotCfg.ServerName != "tillsyn-mcp" {
		t.Fatalf("mcp server name = %q, want tillsyn-mcp", gotCfg.ServerName)
	}
	if gotCfg.MCPEndpoint != "/mcp" {
		t.Fatalf("mcp endpoint = %q, want /mcp", gotCfg.MCPEndpoint)
	}
	if gotDeps.CaptureState == nil || gotDeps.Attention == nil {
		t.Fatalf("expected stdio MCP dependencies to be wired, got %#v", gotDeps)
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{AppName: "tillsyn-mcp", DevMode: true})
	if err != nil {
		t.Fatalf("DefaultPathsWithOptions() error = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(paths.DBPath)); err != nil {
		t.Fatalf("expected shared runtime directory at %q, stat error = %v", filepath.Dir(paths.DBPath), err)
	}
	if _, err := os.Stat(paths.DBPath); err != nil {
		t.Fatalf("expected shared runtime db at %q, stat error = %v", paths.DBPath, err)
	}
	if _, err := os.Stat(paths.ConfigPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected stdio mcp to avoid seeding config automatically, stat err = %v", err)
	}
}

// TestRunMCPCommandConfigOverrideUsesConfiguredDB verifies stdio MCP now follows the same config/db contract as the base app.
func TestRunMCPCommandConfigOverrideUsesConfiguredDB(t *testing.T) {
	origRunner := mcpCommandRunner
	t.Cleanup(func() { mcpCommandRunner = origRunner })
	mcpCommandRunner = func(_ context.Context, _ serveradapter.Config, _ serveradapter.Dependencies) error {
		return nil
	}

	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	customConfig := filepath.Join(workspace, "custom.toml")
	customDB := filepath.Join(workspace, "wrong.db")
	if err := os.WriteFile(customConfig, []byte("[database]\npath = '"+customDB+"'\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(custom config) error = %v", err)
	}

	if err := run(context.Background(), []string{"--app", "tillsyn-mcp", "--dev", "--config", customConfig, "mcp"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(mcp with config override) error = %v", err)
	}

	if _, err := os.Stat(customDB); err != nil {
		t.Fatalf("expected configured database path %q to be used, stat err = %v", customDB, err)
	}
}

// TestRunMCPCommandTreatsCanceledRunnerAsCleanShutdown verifies stdio MCP interrupt shutdown does not surface as an error.
func TestRunMCPCommandTreatsCanceledRunnerAsCleanShutdown(t *testing.T) {
	origRunner := mcpCommandRunner
	t.Cleanup(func() { mcpCommandRunner = origRunner })
	started := make(chan struct{})
	mcpCommandRunner = func(ctx context.Context, _ serveradapter.Config, _ serveradapter.Dependencies) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}

	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	writeConfigExample(t, workspace, "[logging]\nlevel = \"debug\"\n")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()
	if err := run(ctx, []string{"--app", "tillsyn-mcp", "--dev", "mcp"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(mcp canceled) error = %v, want nil clean shutdown", err)
	}
}

// TestRunMCPCommandUsesInterruptEchoSuppression verifies the stdio daemon path applies the Ctrl-C echo suppression wrapper.
func TestRunMCPCommandUsesInterruptEchoSuppression(t *testing.T) {
	origRunner := mcpCommandRunner
	origWrapper := withInterruptEchoSuppressedFunc
	t.Cleanup(func() {
		mcpCommandRunner = origRunner
		withInterruptEchoSuppressedFunc = origWrapper
	})

	var calls int
	withInterruptEchoSuppressedFunc = func(runFn func() error) error {
		calls++
		if runFn == nil {
			return nil
		}
		return runFn()
	}
	mcpCommandRunner = func(_ context.Context, _ serveradapter.Config, _ serveradapter.Dependencies) error {
		return nil
	}

	workspace := t.TempDir()
	t.Chdir(workspace)
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	writeConfigExample(t, workspace, "[logging]\nlevel = \"debug\"\n")

	if err := run(context.Background(), []string{"--app", "tillsyn-mcp", "--dev", "mcp"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(mcp) error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("withInterruptEchoSuppressedFunc calls = %d, want 1", calls)
	}
}

// TestRunUnknownCommand verifies behavior for the covered scenario.
func TestRunUnknownCommand(t *testing.T) {
	err := run(context.Background(), []string{"unknown-command"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

// TestRunServeCommandWiresDefaults verifies serve command wiring with default endpoint flags.
func TestRunServeCommandWiresDefaults(t *testing.T) {
	origRunner := serveCommandRunner
	t.Cleanup(func() { serveCommandRunner = origRunner })

	var gotCfg serveradapter.Config
	var gotDeps serveradapter.Dependencies
	serveCommandRunner = func(_ context.Context, cfg serveradapter.Config, deps serveradapter.Dependencies) error {
		gotCfg = cfg
		gotDeps = deps
		return nil
	}

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "tillsyn.toml")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "serve"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(serve) error = %v", err)
	}
	if gotCfg.HTTPBind != "127.0.0.1:5437" {
		t.Fatalf("serve http bind = %q, want 127.0.0.1:5437", gotCfg.HTTPBind)
	}
	if gotCfg.APIEndpoint != "/api/v1" {
		t.Fatalf("serve api endpoint = %q, want /api/v1", gotCfg.APIEndpoint)
	}
	if gotCfg.MCPEndpoint != "/mcp" {
		t.Fatalf("serve mcp endpoint = %q, want /mcp", gotCfg.MCPEndpoint)
	}
	if gotDeps.CaptureState == nil {
		t.Fatal("expected capture_state dependency to be wired")
	}
	if gotDeps.Attention == nil {
		t.Fatal("expected attention dependency to be wired")
	}
}

// TestRunServeCommandUsesInterruptEchoSuppression verifies the HTTP daemon path applies the Ctrl-C echo suppression wrapper.
func TestRunServeCommandUsesInterruptEchoSuppression(t *testing.T) {
	origRunner := serveCommandRunner
	origWrapper := withInterruptEchoSuppressedFunc
	t.Cleanup(func() {
		serveCommandRunner = origRunner
		withInterruptEchoSuppressedFunc = origWrapper
	})

	var calls int
	withInterruptEchoSuppressedFunc = func(runFn func() error) error {
		calls++
		if runFn == nil {
			return nil
		}
		return runFn()
	}
	serveCommandRunner = func(_ context.Context, _ serveradapter.Config, _ serveradapter.Dependencies) error {
		return nil
	}

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "tillsyn.toml")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "serve"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(serve) error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("withInterruptEchoSuppressedFunc calls = %d, want 1", calls)
	}
}

// TestRunServeCommandWiresFlags verifies serve command forwards endpoint flag overrides.
func TestRunServeCommandWiresFlags(t *testing.T) {
	origRunner := serveCommandRunner
	t.Cleanup(func() { serveCommandRunner = origRunner })

	var gotCfg serveradapter.Config
	serveCommandRunner = func(_ context.Context, cfg serveradapter.Config, _ serveradapter.Dependencies) error {
		gotCfg = cfg
		return nil
	}

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "tillsyn.toml")
	args := []string{
		"--db", dbPath,
		"--config", cfgPath,
		"serve",
		"--http", "127.0.0.1:9090",
		"--api-endpoint", "/custom-api",
		"--mcp-endpoint", "/custom-mcp",
	}
	if err := run(context.Background(), args, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(serve with flags) error = %v", err)
	}
	if gotCfg.HTTPBind != "127.0.0.1:9090" {
		t.Fatalf("serve http bind = %q, want 127.0.0.1:9090", gotCfg.HTTPBind)
	}
	if gotCfg.APIEndpoint != "/custom-api" {
		t.Fatalf("serve api endpoint = %q, want /custom-api", gotCfg.APIEndpoint)
	}
	if gotCfg.MCPEndpoint != "/custom-mcp" {
		t.Fatalf("serve mcp endpoint = %q, want /custom-mcp", gotCfg.MCPEndpoint)
	}
}

// TestRunServeCommandTreatsCanceledRunnerAsCleanShutdown verifies serve interrupt shutdown stays clean through the wrapper path.
func TestRunServeCommandTreatsCanceledRunnerAsCleanShutdown(t *testing.T) {
	origRunner := serveCommandRunner
	t.Cleanup(func() { serveCommandRunner = origRunner })
	started := make(chan struct{})
	serveCommandRunner = func(ctx context.Context, _ serveradapter.Config, _ serveradapter.Dependencies) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "tillsyn.toml")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()
	if err := run(ctx, []string{"--db", dbPath, "--config", cfgPath, "serve"}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(serve canceled) error = %v, want nil clean shutdown", err)
	}
}

// TestRunServeCommandPropagatesErrors verifies serve runner failures are returned to callers.
func TestRunServeCommandPropagatesErrors(t *testing.T) {
	origRunner := serveCommandRunner
	t.Cleanup(func() { serveCommandRunner = origRunner })

	serveCommandRunner = func(_ context.Context, _ serveradapter.Config, _ serveradapter.Dependencies) error {
		return errors.New("listen failed")
	}

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "tillsyn.toml")
	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "serve"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected serve command error")
	}
	if !strings.Contains(err.Error(), "run serve command") {
		t.Fatalf("expected wrapped serve error, got %v", err)
	}
}

// TestRunExportCommandWritesSnapshot verifies behavior for the covered scenario.
func TestRunExportCommandWritesSnapshot(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "missing.toml")
	outPath := filepath.Join(tmp, "snapshot.json")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", outPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(export) error = %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var snap app.Snapshot
	if err := json.Unmarshal(content, &snap); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if snap.Version != app.SnapshotVersion {
		t.Fatalf("unexpected snapshot version %q", snap.Version)
	}
	if len(snap.Projects) != 0 {
		t.Fatalf("expected no projects in empty export snapshot, got %d", len(snap.Projects))
	}
}

// TestRunImportCommandReadsSnapshot verifies behavior for the covered scenario.
func TestRunImportCommandReadsSnapshot(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "missing.toml")

	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	snap := app.Snapshot{
		Version: app.SnapshotVersion,
		Projects: []app.SnapshotProject{
			{
				ID:        "p-import",
				Slug:      "imported",
				Name:      "Imported",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Columns: []app.SnapshotColumn{
			{
				ID:        "c-import",
				ProjectID: "p-import",
				Name:      "To Do",
				Position:  0,
				WIPLimit:  0,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Tasks: []app.SnapshotTask{
			{
				ID:        "t-import",
				ProjectID: "p-import",
				ColumnID:  "c-import",
				Position:  0,
				Title:     "Imported Task",
				Priority:  domain.PriorityMedium,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	content, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	inPath := filepath.Join(tmp, "in.json")
	if err := os.WriteFile(inPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "import", "--in", inPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(import) error = %v", err)
	}

	outPath := filepath.Join(tmp, "out.json")
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", outPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run(export) error = %v", err)
	}
	outContent, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var outSnap app.Snapshot
	if err := json.Unmarshal(outContent, &outSnap); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	foundProject := false
	foundTask := false
	for _, p := range outSnap.Projects {
		if p.ID == "p-import" {
			foundProject = true
			break
		}
	}
	for _, tk := range outSnap.Tasks {
		if tk.ID == "t-import" {
			foundTask = true
			break
		}
	}
	if !foundProject || !foundTask {
		t.Fatalf("expected imported data in exported snapshot, foundProject=%t foundTask=%t", foundProject, foundTask)
	}
}

// TestRunExportToStdoutAndImportErrors verifies behavior for the covered scenario.
func TestRunExportToStdoutAndImportErrors(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, t.TempDir())
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("initial run() error = %v", err)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "export", "--out", "-"}, &out, io.Discard); err != nil {
		t.Fatalf("run(export stdout) error = %v", err)
	}
	if !strings.Contains(out.String(), "\"version\"") {
		t.Fatalf("expected snapshot json on stdout, got %q", out.String())
	}

	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "import"}, io.Discard, io.Discard); err == nil {
		t.Fatal("expected import error for missing --in")
	}

	badIn := filepath.Join(tmp, "bad.json")
	if err := os.WriteFile(badIn, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath, "import", "--in", badIn}, io.Discard, io.Discard); err == nil {
		t.Fatal("expected import decode error")
	}
}

// TestRunConfigAndDBEnvOverrides verifies behavior for the covered scenario.
func TestRunConfigAndDBEnvOverrides(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "env.db")
	cfgPath := filepath.Join(tmp, "env.toml")
	cfgContent := "[database]\npath = \"/tmp/ignore-me.db\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("TILL_CONFIG", cfgPath)
	t.Setenv("TILL_DB_PATH", dbPath)

	err := run(context.Background(), []string{"export", "--out", filepath.Join(tmp, "out.json")}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("run(export with env paths) error = %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db created at env path, stat error %v", err)
	}
}

// TestRunInitDevConfigCreatesDebugConfig verifies init-dev-config creates the dev config and enforces debug logging.
func TestRunInitDevConfigCreatesDebugConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
	t.Chdir(tmp)
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	const example = `
[database]
path = "/tmp/ignored.db"

[logging]
level = "info"
`
	if err := os.WriteFile(filepath.Join(tmp, "config.example.toml"), []byte(example), 0o644); err != nil {
		t.Fatalf("WriteFile(config.example.toml) error = %v", err)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init-dev-config"}, &out, io.Discard); err != nil {
		t.Fatalf("run(init-dev-config) error = %v", err)
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{AppName: "tillsyn-init", DevMode: true})
	if err != nil {
		t.Fatalf("DefaultPathsWithOptions() error = %v", err)
	}
	for _, want := range []string{"Dev Config", "status", "created dev config", shellEscapePath(paths.ConfigPath), "logging level", "debug"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected %q in init-dev-config output, got %q", want, out.String())
		}
	}

	content, err := os.ReadFile(paths.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	got := string(content)
	if strings.Count(got, "[logging]") != 1 {
		t.Fatalf("expected single [logging] section, got\n%s", got)
	}
	if !strings.Contains(got, "level = \"debug\"") {
		t.Fatalf("expected debug logging level in config, got\n%s", got)
	}
}

// TestRunInitDevConfigUpdatesExistingConfig verifies init-dev-config rewrites an existing logging section to debug.
func TestRunInitDevConfigUpdatesExistingConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
	t.Chdir(tmp)
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "config.example.toml"), []byte("[database]\npath = \"/tmp/default.db\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.example.toml) error = %v", err)
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{AppName: "tillsyn-init", DevMode: true})
	if err != nil {
		t.Fatalf("DefaultPathsWithOptions() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.ConfigPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(config dir) error = %v", err)
	}
	const existing = `
[logging]
level = 'info'

[identity]
display_name = "Lane User"
`
	if err := os.WriteFile(paths.ConfigPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile(existing config) error = %v", err)
	}

	var out strings.Builder
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "init-dev-config"}, &out, io.Discard); err != nil {
		t.Fatalf("run(init-dev-config existing) error = %v", err)
	}
	for _, want := range []string{"Dev Config", "status", "dev config already exists", shellEscapePath(paths.ConfigPath), "logging level", "debug"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected %q in init-dev-config existing output, got %q", want, out.String())
		}
	}

	content, err := os.ReadFile(paths.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	got := string(content)
	if strings.Count(got, "[logging]") != 1 {
		t.Fatalf("expected single [logging] section, got\n%s", got)
	}
	if !strings.Contains(got, "level = \"debug\"") {
		t.Fatalf("expected debug logging level in config, got\n%s", got)
	}
	if !strings.Contains(got, "[identity]") {
		t.Fatalf("expected existing config sections to remain, got\n%s", got)
	}
}

// TestRunPathsCommand verifies behavior for the covered scenario.
func TestRunPathsCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var out strings.Builder
	err := run(context.Background(), []string{"--app", "tillsynx", "--dev", "paths"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("run(paths) error = %v", err)
	}
	output := out.String()
	for _, want := range []string{"Resolved Paths", "app", "tillsynx", "root", "database", "logs", "dev_mode", "true"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in paths output, got %q", want, output)
		}
	}
}

// TestRunPathsCommandUsesActiveRuntimeRootForDBOverride verifies `paths` follows the effective DB-selected runtime root.
func TestRunPathsCommandUsesActiveRuntimeRootForDBOverride(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	dbPath := filepath.Join(root, "runtime.db")

	var out strings.Builder
	err := run(context.Background(), []string{"--db", dbPath, "paths"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("run(paths with db override) error = %v", err)
	}
	output := out.String()
	for _, want := range []string{
		"Resolved Paths",
		root,
		dbPath,
		filepath.Join(root, "logs"),
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in paths output, got %q", want, output)
		}
	}
}

// TestRunPathsCommandUsesConfigDatabasePathForRootAndLogs verifies config-driven database paths reshape the reported runtime root.
func TestRunPathsCommandUsesConfigDatabasePathForRootAndLogs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	dbPath := filepath.Join(root, "runtime.db")
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	cfgContent := fmt.Sprintf("[database]\npath = %q\n", dbPath)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out strings.Builder
	err := run(context.Background(), []string{"--config", cfgPath, "paths"}, &out, io.Discard)
	if err != nil {
		t.Fatalf("run(paths with config database path) error = %v", err)
	}
	output := out.String()
	for _, want := range []string{
		"Resolved Paths",
		root,
		dbPath,
		filepath.Join(root, "logs"),
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in paths output, got %q", want, output)
		}
	}
}

// TestShellEscapePath verifies init-dev-config path output is shell-token safe.
func TestShellEscapePath(t *testing.T) {
	in := "/Users/me/Library/Application Support/tillsyn-dev/config.toml"
	want := "/Users/me/Library/Application\\ Support/tillsyn-dev/config.toml"
	if got := shellEscapePath(in); got != want {
		t.Fatalf("shellEscapePath() = %q, want %q", got, want)
	}
}

// TestWritePathsOutputPlain verifies non-terminal writers still receive structured path output.
func TestWritePathsOutputPlain(t *testing.T) {
	var out strings.Builder
	err := writePathsOutput(&out, rootCommandOptions{
		appName: "tillsynx",
		devMode: true,
	}, resolvedRuntimePaths{
		ConfigPath: "/tmp/tillsynx/config.toml",
		DBPath:     "/tmp/tillsynx/tillsynx.db",
	}, "/tmp/tillsynx", "/tmp/tillsynx/logs")
	if err != nil {
		t.Fatalf("writePathsOutput() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Resolved Paths",
		"app",
		"tillsynx",
		"/tmp/tillsynx/config.toml",
		"/tmp/tillsynx/tillsynx.db",
		"/tmp/tillsynx/logs",
		"dev_mode",
		"true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in writePathsOutput output, got %q", want, got)
		}
	}
}

// TestWritePathsOutputStyled verifies styled rendering can be forced in tests.
func TestWritePathsOutputStyled(t *testing.T) {
	orig := supportsStyledOutputFunc
	supportsStyledOutputFunc = func(io.Writer) bool { return true }
	t.Cleanup(func() { supportsStyledOutputFunc = orig })

	var out strings.Builder
	err := writePathsOutput(&out, rootCommandOptions{
		appName: "tillsynx",
		devMode: false,
	}, resolvedRuntimePaths{
		ConfigPath: "/tmp/tillsynx/config.toml",
		DBPath:     "/tmp/tillsynx/tillsynx.db",
	}, "/tmp/tillsynx", "/tmp/tillsynx/logs")
	if err != nil {
		t.Fatalf("writePathsOutput(styled) error = %v", err)
	}
	got := stripANSITest(out.String())
	if !strings.Contains(got, "Resolved Paths") {
		t.Fatalf("expected styled heading in output, got %q", got)
	}
	if !strings.Contains(got, "app") || !strings.Contains(got, "tillsynx") {
		t.Fatalf("expected app row in styled output, got %q", got)
	}
}

// TestSupportsStyledOutput verifies non-terminal and NO_COLOR behavior.
func TestSupportsStyledOutput(t *testing.T) {
	if supportsStyledOutput(&strings.Builder{}) {
		t.Fatal("expected non-file writer to disable styles")
	}

	t.Setenv("NO_COLOR", "1")
	if supportsStyledOutput(os.Stdout) {
		t.Fatal("expected NO_COLOR to disable styles")
	}
}

// TestParseBoolEnv verifies behavior for the covered scenario.
func TestParseBoolEnv(t *testing.T) {
	t.Setenv("TILL_BOOL_TEST", "true")
	got, ok := parseBoolEnv("TILL_BOOL_TEST")
	if !ok || !got {
		t.Fatalf("expected true bool env parse, got value=%t ok=%t", got, ok)
	}

	t.Setenv("TILL_BOOL_TEST", "not-bool")
	_, ok = parseBoolEnv("TILL_BOOL_TEST")
	if ok {
		t.Fatal("expected invalid bool env to return ok=false")
	}
}

// TestStartupBootstrapRequired verifies startup bootstrap requirement detection from config values.
func TestStartupBootstrapRequired(t *testing.T) {
	cfg := config.Default("/tmp/tillsyn.db")
	cfg.Identity.DisplayName = ""
	cfg.Paths.SearchRoots = []string{"/tmp/code"}
	if !startupBootstrapRequired(cfg) {
		t.Fatal("expected missing display name to require startup bootstrap")
	}

	cfg.Identity.DisplayName = "Lane User"
	cfg.Paths.SearchRoots = nil
	if !startupBootstrapRequired(cfg) {
		t.Fatal("expected missing search roots to require startup bootstrap")
	}

	cfg.Identity.DisplayName = "Lane User"
	cfg.Paths.SearchRoots = []string{"/tmp/code"}
	if startupBootstrapRequired(cfg) {
		t.Fatal("expected complete identity + search roots to bypass startup bootstrap")
	}
}

// TestSanitizeBootstrapActorType verifies actor type normalization for bootstrap persistence.
func TestSanitizeBootstrapActorType(t *testing.T) {
	cases := map[string]string{
		"user":        "user",
		"AGENT":       "agent",
		" system ":    "system",
		"unexpected":  "user",
		"":            "user",
		"\nunknown\t": "user",
	}
	for input, want := range cases {
		if got := sanitizeBootstrapActorType(input); got != want {
			t.Fatalf("sanitizeBootstrapActorType(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestRunDevModeCreatesRuntimeRootLogFile verifies dev runtime logs go under the shared runtime root logs dir.
func TestRunDevModeCreatesRuntimeRootLogFile(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	workspace := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(workspace)

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	if err := run(context.Background(), []string{"--dev", "--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	logDir := filepath.Join(workspace, "logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected dev log file in %s", logDir)
	}
	foundLog := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Fatalf("expected at least one .log file in %s, got %v", logDir, entries)
	}
}

// TestRunTUIModeWritesRuntimeLogsToFileOnly verifies default-runtime TUI logs stay out of stderr and persist to the runtime log file.
func TestRunTUIModeWritesRuntimeLogsToFileOnly(t *testing.T) {
	origFactory := programFactory
	t.Cleanup(func() { programFactory = origFactory })
	programFactory = func(_ tea.Model) program { return fakeProgram{} }

	workspace := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(workspace)

	dbPath := filepath.Join(workspace, "tillsyn.db")
	cfgPath := filepath.Join(workspace, "config.toml")
	writeBootstrapReadyConfig(t, cfgPath, workspace)
	var stderr bytes.Buffer
	if err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, &stderr); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got := strings.TrimSpace(stderr.String()); got != "" {
		t.Fatalf("expected no runtime stderr output in TUI mode, got %q", got)
	}

	logDir := filepath.Join(workspace, "logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	var logPath string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		logPath = filepath.Join(logDir, entry.Name())
		break
	}
	if logPath == "" {
		t.Fatalf("expected a .log file in %s", logDir)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	logOutput := string(content)
	if !strings.Contains(logOutput, "starting tui program loop") {
		t.Fatalf("expected runtime log file to include TUI lifecycle entries, got %q", logOutput)
	}
}

// TestWorkspaceRootFromUsesNearestMarker verifies workspace-root resolution behavior.
func TestWorkspaceRootFromUsesNearestMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	nested := filepath.Join(root, "cmd", "till")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	got := workspaceRootFrom(nested)
	if filepath.Clean(got) != filepath.Clean(root) {
		t.Fatalf("expected workspace root %q, got %q", root, got)
	}
}

// TestDevLogFilePathResolvesAgainstWorkspaceRoot verifies explicit relative overrides still anchor at workspace root.
func TestDevLogFilePathResolvesAgainstWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	nested := filepath.Join(root, "cmd", "till")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prev)
	})
	got, err := devLogFilePath("./tmp/logs", "/ignored/default/logs", "tillsyn", time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("devLogFilePath() error = %v", err)
	}
	wantPrefix := filepath.Join(root, "tmp", "logs")
	normalize := func(p string) string {
		return strings.TrimPrefix(filepath.Clean(p), "/private")
	}
	if !strings.HasPrefix(normalize(got), normalize(wantPrefix)) {
		t.Fatalf("expected log path under %q, got %q", wantPrefix, got)
	}
}

// TestResolveRuntimeLogDirUsesSharedRootForDefaultSentinel verifies the default dev log dir resolves under the runtime root.
func TestResolveRuntimeLogDirUsesSharedRootForDefaultSentinel(t *testing.T) {
	want := filepath.Join(t.TempDir(), "logs")
	got, err := resolveRuntimeLogDir(config.DefaultDevLogDir(), want)
	if err != nil {
		t.Fatalf("resolveRuntimeLogDir() error = %v", err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("resolveRuntimeLogDir() = %q, want %q", got, filepath.Clean(want))
	}
}

// TestRunRejectsInvalidLoggingLevelFromConfig verifies behavior for the covered scenario.
func TestRunRejectsInvalidLoggingLevelFromConfig(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tillsyn.db")
	cfgPath := filepath.Join(tmp, "tillsyn.toml")
	cfgContent := "[logging]\nlevel = \"verbose\"\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := run(context.Background(), []string{"--db", dbPath, "--config", cfgPath}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected invalid logging level error")
	}
	if !strings.Contains(err.Error(), "invalid logging.level") {
		t.Fatalf("expected logging level validation error, got %v", err)
	}
}

// TestEnsureLoggingSectionDebug verifies TOML logging rewrite behavior across common config shapes.
func TestEnsureLoggingSectionDebug(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "replace existing level",
			in:   "[logging]\nlevel = \"info\"\n\n[database]\npath = \"/tmp/tillsyn.db\"\n",
			want: []string{"[logging]", "level = \"debug\"", "[database]"},
		},
		{
			name: "append missing level",
			in:   "[logging]\n# comment\n\n[database]\npath = \"/tmp/tillsyn.db\"\n",
			want: []string{"[logging]", "level = \"debug\"", "[database]"},
		},
		{
			name: "append missing section",
			in:   "[database]\npath = \"/tmp/tillsyn.db\"\n",
			want: []string{"[database]", "[logging]", "level = \"debug\""},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ensureLoggingSectionDebug(tc.in)
			if strings.Count(got, "[logging]") != 1 {
				t.Fatalf("expected one [logging] section, got\n%s", got)
			}
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("expected %q in rewritten config, got\n%s", want, got)
				}
			}
		})
	}
}

// TestLoadRuntimeConfigMapsRuntimeFields verifies behavior for the covered scenario.
func TestLoadRuntimeConfigMapsRuntimeFields(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "tillsyn.toml")
	content := `
[database]
path = "/tmp/from-config.db"

[delete]
default_mode = "hard"

[confirm]
delete = false
archive = false
hard_delete = false
restore = true

[task_fields]
show_priority = false
show_due_date = false
show_labels = false
show_description = true

[board]
show_wip_warnings = false
group_by = "priority"

[search]
cross_project = true
include_archived = true
states = ["todo", "archived"]

[identity]
actor_id = "runtime-actor-id"
display_name = "Lane User"
default_actor_type = "agent"

[paths]
search_roots = ["/tmp/code", "/tmp/docs"]

[ui]
due_soon_windows = ["6h"]
show_due_summary = false

[project_roots]
inbox = "/tmp/inbox"

[labels]
global = ["bug"]
enforce_allowed = true

[labels.projects]
inbox = ["roadmap"]

[keys]
command_palette = ";"
quick_actions = ","
multi_select = "x"
activity_log = "v"
undo = "u"
redo = "U"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runtimeCfg, err := loadRuntimeConfig(cfgPath, config.Default("/tmp/default.db"), "/tmp/override.db", true)
	if err != nil {
		t.Fatalf("loadRuntimeConfig() error = %v", err)
	}
	if runtimeCfg.DefaultDeleteMode != app.DeleteModeHard {
		t.Fatalf("expected hard delete mode, got %q", runtimeCfg.DefaultDeleteMode)
	}
	if runtimeCfg.TaskFields.ShowPriority || runtimeCfg.TaskFields.ShowDueDate || runtimeCfg.TaskFields.ShowLabels || !runtimeCfg.TaskFields.ShowDescription {
		t.Fatalf("unexpected task fields runtime config %#v", runtimeCfg.TaskFields)
	}
	if !runtimeCfg.Search.CrossProject || !runtimeCfg.Search.IncludeArchived {
		t.Fatalf("unexpected search runtime config %#v", runtimeCfg.Search)
	}
	if runtimeCfg.Board.GroupBy != "priority" || runtimeCfg.Board.ShowWIPWarnings {
		t.Fatalf("unexpected board runtime config %#v", runtimeCfg.Board)
	}
	if runtimeCfg.Confirm.Delete || runtimeCfg.Confirm.Archive || runtimeCfg.Confirm.HardDelete || !runtimeCfg.Confirm.Restore {
		t.Fatalf("unexpected confirm runtime config %#v", runtimeCfg.Confirm)
	}
	if len(runtimeCfg.UI.DueSoonWindows) != 1 || runtimeCfg.UI.DueSoonWindows[0] != 6*time.Hour || runtimeCfg.UI.ShowDueSummary {
		t.Fatalf("unexpected ui runtime config %#v", runtimeCfg.UI)
	}
	wantSearchRootCode := filepath.Clean("/tmp/code")
	wantSearchRootDocs := filepath.Clean("/tmp/docs")
	if len(runtimeCfg.SearchRoots) != 2 || runtimeCfg.SearchRoots[0] != wantSearchRootCode || runtimeCfg.SearchRoots[1] != wantSearchRootDocs {
		t.Fatalf("unexpected search roots runtime config %#v", runtimeCfg.SearchRoots)
	}
	if got := runtimeCfg.Keys.CommandPalette; got != ";" {
		t.Fatalf("expected command palette key override ';', got %q", got)
	}
	if got := runtimeCfg.ProjectRoots["inbox"]; got != "/tmp/inbox" {
		t.Fatalf("unexpected project roots runtime config %#v", runtimeCfg.ProjectRoots)
	}
	if !runtimeCfg.Labels.EnforceAllowed || len(runtimeCfg.Labels.Global) != 1 || runtimeCfg.Labels.Global[0] != "bug" {
		t.Fatalf("unexpected label runtime config %#v", runtimeCfg.Labels)
	}
	if got := runtimeCfg.Labels.Projects["inbox"]; len(got) != 1 || got[0] != "roadmap" {
		t.Fatalf("unexpected project labels runtime config %#v", runtimeCfg.Labels.Projects)
	}
	if got := runtimeCfg.Identity.DisplayName; got != "Lane User" {
		t.Fatalf("expected identity display name Lane User, got %q", got)
	}
	if got := runtimeCfg.Identity.ActorID; got != "runtime-actor-id" {
		t.Fatalf("expected identity actor_id runtime-actor-id, got %q", got)
	}
	if got := runtimeCfg.Identity.DefaultActorType; got != "agent" {
		t.Fatalf("expected identity actor type agent, got %q", got)
	}
}

// TestPersistProjectRootRoundTrip verifies behavior for the covered scenario.
func TestPersistProjectRootRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "tillsyn.toml")

	if err := persistProjectRoot(cfgPath, "Inbox", "/tmp/inbox"); err != nil {
		t.Fatalf("persistProjectRoot() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.ProjectRoots["inbox"]; got != "/tmp/inbox" {
		t.Fatalf("expected persisted project root /tmp/inbox, got %#v", cfg.ProjectRoots)
	}

	if err := persistProjectRoot(cfgPath, "inbox", ""); err != nil {
		t.Fatalf("persistProjectRoot(clear) error = %v", err)
	}
	cfg, err = config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if _, ok := cfg.ProjectRoots["inbox"]; ok {
		t.Fatalf("expected project root cleared, got %#v", cfg.ProjectRoots)
	}
}

// TestPersistIdentityRoundTrip verifies behavior for the covered scenario.
func TestPersistIdentityRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "tillsyn.toml")

	if err := persistIdentity(cfgPath, "lane-actor-id", "Lane User", "agent"); err != nil {
		t.Fatalf("persistIdentity() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Identity.DisplayName; got != "Lane User" {
		t.Fatalf("expected persisted identity display name Lane User, got %q", got)
	}
	if got := cfg.Identity.DefaultActorType; got != "agent" {
		t.Fatalf("expected persisted identity actor type agent, got %q", got)
	}
	if got := cfg.Identity.ActorID; got != "lane-actor-id" {
		t.Fatalf("expected persisted identity actor_id lane-actor-id, got %q", got)
	}
}

// TestPersistSearchRootsRoundTrip verifies behavior for the covered scenario.
func TestPersistSearchRootsRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "tillsyn.toml")

	if err := persistSearchRoots(cfgPath, []string{"/tmp/code", "/tmp/docs", "/tmp/code"}); err != nil {
		t.Fatalf("persistSearchRoots() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	wantSearchRootCode := filepath.Clean("/tmp/code")
	wantSearchRootDocs := filepath.Clean("/tmp/docs")
	if len(cfg.Paths.SearchRoots) != 2 || cfg.Paths.SearchRoots[0] != wantSearchRootCode || cfg.Paths.SearchRoots[1] != wantSearchRootDocs {
		t.Fatalf("unexpected persisted search roots %#v", cfg.Paths.SearchRoots)
	}
}

// TestPersistAllowedLabelsRoundTrip verifies behavior for the covered scenario.
func TestPersistAllowedLabelsRoundTrip(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "tillsyn.toml")

	if err := persistAllowedLabels(cfgPath, "Inbox", []string{"Bug", "chore", "bug"}, []string{"Roadmap", "till", "roadmap"}); err != nil {
		t.Fatalf("persistAllowedLabels() error = %v", err)
	}
	cfg, err := config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	wantGlobal := []string{"bug", "chore"}
	if len(cfg.Labels.Global) != len(wantGlobal) {
		t.Fatalf("unexpected persisted global labels %#v", cfg.Labels.Global)
	}
	for i := range wantGlobal {
		if cfg.Labels.Global[i] != wantGlobal[i] {
			t.Fatalf("unexpected global label at %d: got %q want %q", i, cfg.Labels.Global[i], wantGlobal[i])
		}
	}
	wantProject := []string{"roadmap", "till"}
	gotProject := cfg.Labels.Projects["inbox"]
	if len(gotProject) != len(wantProject) {
		t.Fatalf("unexpected persisted project labels %#v", cfg.Labels.Projects)
	}
	for i := range wantProject {
		if gotProject[i] != wantProject[i] {
			t.Fatalf("unexpected project label at %d: got %q want %q", i, gotProject[i], wantProject[i])
		}
	}

	if err := persistAllowedLabels(cfgPath, "inbox", []string{"bug"}, nil); err != nil {
		t.Fatalf("persistAllowedLabels(clear project labels) error = %v", err)
	}
	cfg, err = config.Load(cfgPath, config.Default("/tmp/default.db"))
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if _, ok := cfg.Labels.Projects["inbox"]; ok {
		t.Fatalf("expected inbox project labels cleared, got %#v", cfg.Labels.Projects)
	}
	if len(cfg.Labels.Global) != 1 || cfg.Labels.Global[0] != "bug" {
		t.Fatalf("expected global labels to remain bug, got %#v", cfg.Labels.Global)
	}
}

// TestRuntimeLoggerCanMuteConsoleSink verifies console output can be suppressed while other sinks remain active.
func TestRuntimeLoggerCanMuteConsoleSink(t *testing.T) {
	var console bytes.Buffer
	cfg := config.Default("/tmp/tillsyn.db").Logging

	logger, err := newRuntimeLogger(&console, "till", false, cfg, "/tmp/tillsyn/logs", func() time.Time {
		return time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("newRuntimeLogger() error = %v", err)
	}

	logger.Info("before")
	logger.SetConsoleEnabled(false)
	logger.Info("during")
	logger.SetConsoleEnabled(true)
	logger.Info("after")

	out := console.String()
	if !strings.Contains(out, "before") {
		t.Fatalf("expected console log to include 'before', got %q", out)
	}
	if strings.Contains(out, "during") {
		t.Fatalf("expected muted console log to omit 'during', got %q", out)
	}
	if !strings.Contains(out, "after") {
		t.Fatalf("expected console log to include 'after', got %q", out)
	}
}

// TestRuntimeLoggerInstallAsDefaultRoutesPackageLogsToFile verifies package-level charm/log output reaches the runtime file sink.
func TestRuntimeLoggerInstallAsDefaultRoutesPackageLogsToFile(t *testing.T) {
	var console bytes.Buffer
	cfg := config.Default("/tmp/tillsyn.db").Logging
	cfg.DevFile.Enabled = true
	cfg.DevFile.Dir = t.TempDir()

	logger, err := newRuntimeLogger(&console, "till", true, cfg, "/tmp/tillsyn/logs", func() time.Time {
		return time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("newRuntimeLogger() error = %v", err)
	}
	t.Cleanup(func() {
		logger.RestoreDefault()
		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})

	logger.InstallAsDefault("till")

	charmLog.Error("mapped parity probe", "transport", "mcp")
	if got := console.String(); !strings.Contains(got, "mapped parity probe") {
		t.Fatalf("expected console to include package-level log output, got %q", got)
	}

	logPath := logger.DevLogPath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", logPath, err)
	}
	if got := string(content); !strings.Contains(got, "mapped parity probe") {
		t.Fatalf("expected dev log file to include package-level log output, got %q", got)
	}

	console.Reset()
	logger.SetConsoleEnabled(false)
	charmLog.Error("mapped parity file-only probe", "transport", "http")
	if got := console.String(); strings.Contains(got, "mapped parity file-only probe") {
		t.Fatalf("expected muted console to omit package-level log output, got %q", got)
	}

	content, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) after mute error = %v", logPath, err)
	}
	if got := string(content); !strings.Contains(got, "mapped parity file-only probe") {
		t.Fatalf("expected dev log file to include muted-console package log output, got %q", got)
	}
}
