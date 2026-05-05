package render_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude/render"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stubGrantsLister is the minimal in-memory PermissionGrantsLister used
// by F.7.5c grants-merge tests. It records the (projectID, kind, cliKind)
// tuple it was called with so tests can assert dispatch correctness, and
// returns the canned slice + err set by the constructor.
type stubGrantsLister struct {
	grants []domain.PermissionGrant
	err    error

	// Recorded call arguments (last call wins). Tests assert these to
	// confirm the render layer dispatches with the right tuple.
	gotProjectID string
	gotKind      domain.Kind
	gotCLIKind   string
	callCount    int
}

// ListGrantsForKind satisfies render.PermissionGrantsLister.
func (s *stubGrantsLister) ListGrantsForKind(_ context.Context, projectID string, kind domain.Kind, cliKind string) ([]domain.PermissionGrant, error) {
	s.gotProjectID = projectID
	s.gotKind = kind
	s.gotCLIKind = cliKind
	s.callCount++
	if s.err != nil {
		return nil, s.err
	}
	if s.grants == nil {
		return []domain.PermissionGrant{}, nil
	}
	return s.grants, nil
}

// fixtureBundle returns a Bundle with paths rooted under t.TempDir() so
// each test's writes land in an isolated directory cleaned up by the
// testing framework. The shape mirrors what dispatcher.NewBundle produces
// for the os_tmp materialization mode.
func fixtureBundle(t *testing.T) dispatcher.Bundle {
	t.Helper()
	root := t.TempDir()
	return dispatcher.Bundle{
		SpawnID:   "spawn-uuid-fixture-0001",
		Mode:      dispatcher.SpawnTempRootOSTmp,
		StartedAt: time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
		Paths: dispatcher.BundlePaths{
			Root:             root,
			SystemPromptPath: filepath.Join(root, "system-prompt.md"),
			SystemAppendPath: "",
			StreamLogPath:    filepath.Join(root, "stream.jsonl"),
			ManifestPath:     filepath.Join(root, "manifest.json"),
			ContextDir:       filepath.Join(root, "context"),
		},
	}
}

// fixtureItem returns a populated build action item with paths +
// packages so the system-prompt body exercises the optional-field
// branches.
func fixtureItem() domain.ActionItem {
	return domain.ActionItem{
		ID:       "ai-build-fixture-1",
		Kind:     domain.KindBuild,
		Title:    "DROPLET F.7.3B BUNDLE RENDER",
		Paths:    []string{"internal/app/dispatcher/cli_claude/render/render.go"},
		Packages: []string{"github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_claude/render"},
	}
}

// fixtureProject returns a populated project value the prompt body reads
// from. HyllaArtifactRef is set to exercise the F.7.10 negative
// assertion (the value MUST NOT leak into the rendered prompt body).
func fixtureProject() domain.Project {
	return domain.Project{
		ID:                  "proj-fixture",
		RepoPrimaryWorktree: "/tmp/tillsyn/main",
		HyllaArtifactRef:    "github.com/evanmschultz/tillsyn@main",
		Language:            "go",
	}
}

// fixtureBinding returns a BindingResolved with both ToolsAllowed and
// ToolsDisallowed populated so the settings.json + agent file render
// branches both fire.
func fixtureBinding() dispatcher.BindingResolved {
	return dispatcher.BindingResolved{
		AgentName:       "go-builder-agent",
		CLIKind:         dispatcher.CLIKindClaude,
		ToolsAllowed:    []string{"Read", "Grep"},
		ToolsDisallowed: []string{"WebFetch", "Bash(curl *)"},
	}
}

// TestRenderHappyPathWritesAllFiveFiles is the integration assertion:
// one Render call writes every artifact the bundle subtree carries per
// memory §2 (system-prompt.md cross-CLI; plugin/{plugin.json, agents/<name>.md,
// .mcp.json, settings.json} claude-specific).
func TestRenderHappyPathWritesAllFiveFiles(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	wantFiles := []string{
		filepath.Join(bundle.Paths.Root, "system-prompt.md"),
		filepath.Join(bundle.Paths.Root, "plugin", ".claude-plugin", "plugin.json"),
		filepath.Join(bundle.Paths.Root, "plugin", "agents", "go-builder-agent.md"),
		filepath.Join(bundle.Paths.Root, "plugin", ".mcp.json"),
		filepath.Join(bundle.Paths.Root, "plugin", "settings.json"),
	}
	for _, p := range wantFiles {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("os.Stat(%q) error = %v, want file present", p, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %q has zero bytes; expected non-empty content", p)
		}
	}
}

// TestRenderSystemPromptContainsStructuralTokens asserts the rendered
// system-prompt.md carries the action-item structural fields (task_id,
// project_id, project_dir, kind, title, paths, packages, move-state
// directive) AND does NOT carry hylla_artifact_ref per F.7.10.
func TestRenderSystemPromptContainsStructuralTokens(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	item := fixtureItem()
	project := fixtureProject()
	promptBody, err := render.Render(context.Background(), bundle, item, project, fixtureBinding(), nil)
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body, err := os.ReadFile(bundle.Paths.SystemPromptPath)
	if err != nil {
		t.Fatalf("os.ReadFile(system-prompt.md) error = %v", err)
	}
	bodyStr := string(body)
	if promptBody != bodyStr {
		t.Errorf("Render() returned body != file contents\nreturned:\n%s\nfile:\n%s", promptBody, bodyStr)
	}

	wantTokens := []string{
		"task_id: " + item.ID,
		"project_id: " + project.ID,
		"project_dir: " + project.RepoPrimaryWorktree,
		"kind: " + string(item.Kind),
		"title: " + item.Title,
		"paths: " + item.Paths[0],
		"packages: " + item.Packages[0],
		"move-state directive:",
	}
	for _, tok := range wantTokens {
		if !strings.Contains(bodyStr, tok) {
			t.Errorf("system-prompt.md missing %q\nfull body:\n%s", tok, bodyStr)
		}
	}

	// F.7.10: hylla_artifact_ref MUST NOT appear in the prompt body.
	if strings.Contains(bodyStr, "hylla_artifact_ref") {
		t.Errorf("system-prompt.md unexpectedly contains hylla_artifact_ref\nfull body:\n%s", bodyStr)
	}
}

// TestRenderPluginManifestExactShape asserts plugin.json carries the
// exact shape `{"name": "spawn-<spawn-id>"}` and nothing else (parseable
// to a 1-key map).
func TestRenderPluginManifestExactShape(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	manifestPath := filepath.Join(bundle.Paths.Root, "plugin", ".claude-plugin", "plugin.json")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile(plugin.json) error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(contents, &parsed); err != nil {
		t.Fatalf("json.Unmarshal plugin.json error = %v\ncontents:\n%s", err, contents)
	}

	if got, want := parsed["name"], "spawn-"+bundle.SpawnID; got != want {
		t.Errorf("plugin.json name = %v, want %q", got, want)
	}
	if len(parsed) != 1 {
		t.Errorf("plugin.json has %d keys, want exactly 1\nparsed: %#v", len(parsed), parsed)
	}
}

// TestRenderMCPConfigExactShape asserts .mcp.json carries
// {"tillsyn": {"command": "till", "args": ["serve-mcp"]}}.
func TestRenderMCPConfigExactShape(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	mcpPath := filepath.Join(bundle.Paths.Root, "plugin", ".mcp.json")
	contents, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("os.ReadFile(.mcp.json) error = %v", err)
	}

	var parsed struct {
		Tillsyn struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"tillsyn"`
	}
	if err := json.Unmarshal(contents, &parsed); err != nil {
		t.Fatalf("json.Unmarshal .mcp.json error = %v\ncontents:\n%s", err, contents)
	}

	if parsed.Tillsyn.Command != "till" {
		t.Errorf("tillsyn.command = %q, want %q", parsed.Tillsyn.Command, "till")
	}
	if len(parsed.Tillsyn.Args) != 1 || parsed.Tillsyn.Args[0] != "serve-mcp" {
		t.Errorf("tillsyn.args = %v, want [\"serve-mcp\"]", parsed.Tillsyn.Args)
	}
}

// TestRenderSettingsPermissions asserts settings.json carries the
// permissions block with allow/deny mirroring binding.ToolsAllowed +
// binding.ToolsDisallowed and `ask` as an empty array (F.7.5b's TUI
// handshake will populate it later).
func TestRenderSettingsPermissions(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := fixtureBinding()
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	settingsPath := filepath.Join(bundle.Paths.Root, "plugin", "settings.json")
	contents, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("os.ReadFile(settings.json) error = %v", err)
	}

	var parsed struct {
		Permissions struct {
			Allow []string `json:"allow"`
			Ask   []string `json:"ask"`
			Deny  []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(contents, &parsed); err != nil {
		t.Fatalf("json.Unmarshal settings.json error = %v\ncontents:\n%s", err, contents)
	}

	if got, want := parsed.Permissions.Allow, binding.ToolsAllowed; !equalStringSlice(got, want) {
		t.Errorf("permissions.allow = %v, want %v", got, want)
	}
	if got, want := parsed.Permissions.Deny, binding.ToolsDisallowed; !equalStringSlice(got, want) {
		t.Errorf("permissions.deny = %v, want %v", got, want)
	}
	if parsed.Permissions.Ask == nil {
		t.Errorf("permissions.ask is nil, want explicit empty array (debuggability)")
	}
	if len(parsed.Permissions.Ask) != 0 {
		t.Errorf("permissions.ask = %v, want empty array", parsed.Permissions.Ask)
	}
}

// TestRenderSettingsExplicitEmptyArraysWhenBindingEmpty asserts the
// nil-slice → []string{} substitution: a binding with nil Allow / Deny
// produces explicit JSON `[]` rather than `null`. Test pins
// debuggability semantics — a dev opening settings.json sees `"allow":
// []` not `"allow": null`.
func TestRenderSettingsExplicitEmptyArraysWhenBindingEmpty(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := dispatcher.BindingResolved{
		AgentName:       "go-builder-agent",
		CLIKind:         dispatcher.CLIKindClaude,
		ToolsAllowed:    nil, // explicit nil
		ToolsDisallowed: nil, // explicit nil
	}
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	contents, err := os.ReadFile(filepath.Join(bundle.Paths.Root, "plugin", "settings.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(settings.json) error = %v", err)
	}

	// String-search the raw JSON to confirm explicit `[]` rather than `null`.
	str := string(contents)
	if !strings.Contains(str, `"allow": []`) {
		t.Errorf("settings.json missing `\"allow\": []`\nfull contents:\n%s", str)
	}
	if !strings.Contains(str, `"deny": []`) {
		t.Errorf("settings.json missing `\"deny\": []`\nfull contents:\n%s", str)
	}
	if strings.Contains(str, `null`) {
		t.Errorf("settings.json contains `null`; want explicit `[]` for permissions arrays\nfull contents:\n%s", str)
	}
}

// TestRenderAgentFileFrontmatter asserts the rendered agent file carries
// the canonical frontmatter (name + description) and the per-spawn
// tool-gating layer-A entries (allowedTools / disallowedTools mirroring
// binding.ToolsAllowed / binding.ToolsDisallowed per memory §5).
func TestRenderAgentFileFrontmatter(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := fixtureBinding()
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	agentPath := filepath.Join(bundle.Paths.Root, "plugin", "agents", binding.AgentName+".md")
	contents, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("os.ReadFile(agent file) error = %v", err)
	}
	str := string(contents)

	// Frontmatter delimiter lines + name + description load-bearing fields.
	wantTokens := []string{
		"---\n",
		"name: " + binding.AgentName,
		"description: ",
		"allowedTools: Read, Grep",
		"disallowedTools: WebFetch, Bash(curl *)",
	}
	for _, tok := range wantTokens {
		if !strings.Contains(str, tok) {
			t.Errorf("agent file missing %q\nfull contents:\n%s", tok, str)
		}
	}
}

// TestRenderAgentFileWithoutToolGating asserts a binding with empty
// ToolsAllowed + ToolsDisallowed renders an agent file with no
// allowedTools / disallowedTools frontmatter lines (those lines are
// optional; only name + description are unconditional).
func TestRenderAgentFileWithoutToolGating(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		// No ToolsAllowed / ToolsDisallowed.
	}
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	agentPath := filepath.Join(bundle.Paths.Root, "plugin", "agents", binding.AgentName+".md")
	contents, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("os.ReadFile(agent file) error = %v", err)
	}
	str := string(contents)

	if strings.Contains(str, "allowedTools:") {
		t.Errorf("agent file unexpectedly contains allowedTools line\ncontents:\n%s", str)
	}
	if strings.Contains(str, "disallowedTools:") {
		t.Errorf("agent file unexpectedly contains disallowedTools line\ncontents:\n%s", str)
	}
	// Name + description still required.
	if !strings.Contains(str, "name: "+binding.AgentName) {
		t.Errorf("agent file missing name frontmatter\ncontents:\n%s", str)
	}
}

// TestRenderRollbackOnAgentDirFailure exercises the rollback path: if a
// pre-existing read-only file blocks creation of the agents directory,
// Render fails and removes every other file it has already written.
//
// Skipped on Windows (chmod-based read-only is unreliable) and when
// running as root (root bypasses permission checks).
func TestRenderRollbackOnAgentDirFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("rollback test relies on POSIX chmod semantics")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root — chmod-based read-only is bypassed")
	}
	t.Parallel()

	bundle := fixtureBundle(t)
	// Plant a regular file at the path Render expects to create as a
	// directory: <Root>/plugin/agents. mkdir on a path that is a regular
	// file fails with ENOTDIR.
	pluginRoot := filepath.Join(bundle.Paths.Root, "plugin")
	if err := os.MkdirAll(pluginRoot, 0o700); err != nil {
		t.Fatalf("seed plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "agents"), []byte("blocker"), 0o600); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}

	_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil)
	if err == nil {
		t.Fatalf("Render() error = nil, want non-nil (agents-dir-creation should fail)")
	}

	// Rollback: every artifact Render creates must be gone. The blocker
	// file we planted is acceptable to remain (it is not Render's to
	// clean up — it is the rollback's whole-plugin-subtree wipe target,
	// so it should also be gone).
	gonePaths := []string{
		filepath.Join(bundle.Paths.Root, "system-prompt.md"),
		filepath.Join(bundle.Paths.Root, "plugin"),
	}
	for _, p := range gonePaths {
		if _, statErr := os.Stat(p); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("path %q still exists after rollback (statErr=%v); rollback should have wiped it", p, statErr)
		}
	}
}

// TestRenderRejectsEmptyBundleRoot asserts the input-validation guard
// for an empty bundle.Paths.Root.
func TestRenderRejectsEmptyBundleRoot(t *testing.T) {
	t.Parallel()

	bundle := dispatcher.Bundle{
		SpawnID: "spawn-fixture",
		// Paths.Root deliberately empty.
	}
	_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil)
	if err == nil {
		t.Fatalf("Render() error = nil, want ErrInvalidRenderInput")
	}
	if !errors.Is(err, render.ErrInvalidRenderInput) {
		t.Errorf("Render() error = %v, want errors.Is(ErrInvalidRenderInput)", err)
	}
}

// TestRenderRejectsEmptyAgentName asserts the input-validation guard
// for an empty binding.AgentName — a corrupted catalog could otherwise
// produce an agent file at <plugin>/agents/.md.
func TestRenderRejectsEmptyAgentName(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := fixtureBinding()
	binding.AgentName = "" // corrupted

	_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil)
	if err == nil {
		t.Fatalf("Render() error = nil, want ErrInvalidRenderInput")
	}
	if !errors.Is(err, render.ErrInvalidRenderInput) {
		t.Errorf("Render() error = %v, want errors.Is(ErrInvalidRenderInput)", err)
	}
}

// TestRenderRejectsAgentNameWithPathSeparator asserts the
// input-validation guard against accidental path-injection through a
// corrupted catalog: an AgentName containing `/` or `\` would otherwise
// escape the agents/ directory. Forward AND backslash both rejected.
func TestRenderRejectsAgentNameWithPathSeparator(t *testing.T) {
	t.Parallel()

	cases := []string{
		"go-builder-agent/../../etc/passwd",
		`go-builder-agent\evil`,
		"a/b",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			bundle := fixtureBundle(t)
			binding := fixtureBinding()
			binding.AgentName = name

			_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil)
			if err == nil {
				t.Fatalf("Render() error = nil, want ErrInvalidRenderInput for %q", name)
			}
			if !errors.Is(err, render.ErrInvalidRenderInput) {
				t.Errorf("Render() error = %v, want errors.Is(ErrInvalidRenderInput)", err)
			}
		})
	}
}

// TestRenderOmitsOptionalSystemPromptFields asserts the system-prompt
// body omits the paths / packages / title lines when the action item
// does not declare them. Empty slices stay omitted so the body stays
// clean.
func TestRenderOmitsOptionalSystemPromptFields(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	item := domain.ActionItem{
		ID:   "ai-min-1",
		Kind: domain.KindBuild,
		// Title, Paths, Packages all empty.
	}
	if _, err := render.Render(context.Background(), bundle, item, fixtureProject(), fixtureBinding(), nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body, err := os.ReadFile(bundle.Paths.SystemPromptPath)
	if err != nil {
		t.Fatalf("os.ReadFile(system-prompt.md) error = %v", err)
	}
	str := string(body)

	notWantTokens := []string{
		"title:",
		"paths:",
		"packages:",
	}
	for _, tok := range notWantTokens {
		if strings.Contains(str, tok) {
			t.Errorf("system-prompt.md unexpectedly contains %q\nfull body:\n%s", tok, str)
		}
	}
	// Mandatory tokens still present.
	if !strings.Contains(str, "task_id: "+item.ID) {
		t.Errorf("system-prompt.md missing task_id\nfull body:\n%s", str)
	}
	if !strings.Contains(str, "move-state directive:") {
		t.Errorf("system-prompt.md missing move-state directive\nfull body:\n%s", str)
	}
}

// readSettingsAllow reads the rendered <plugin>/settings.json under
// bundle.Paths.Root and returns permissions.allow. Test helper for the
// F.7.5c grants-merge cases.
func readSettingsAllow(t *testing.T, bundle dispatcher.Bundle) []string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(bundle.Paths.Root, "plugin", "settings.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(settings.json) error = %v", err)
	}
	var parsed struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(contents, &parsed); err != nil {
		t.Fatalf("json.Unmarshal settings.json error = %v\ncontents:\n%s", err, contents)
	}
	return parsed.Permissions.Allow
}

// grantFixture returns one PermissionGrant with the supplied rule. The
// non-rule fields use stable test values; the lookup tuple
// (projectID, kind, cliKind) carried by the grant is whatever the
// caller passes via the lister stub — render.go reads only Rule off
// each grant when merging.
func grantFixture(rule string) domain.PermissionGrant {
	return domain.PermissionGrant{
		ID:        "grant-" + rule,
		ProjectID: "proj-fixture",
		Kind:      domain.KindBuild,
		Rule:      rule,
		CLIKind:   "claude",
		GrantedBy: "dev-test",
		GrantedAt: time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
	}
}

// TestRenderSettingsNilListerSkipsGrantsLookup asserts the
// deferred-plumbing path: a nil grantsLister causes Render to render
// settings.json with binding.ToolsAllowed only — no lister call, no
// error.
func TestRenderSettingsNilListerSkipsGrantsLookup(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := fixtureBinding()
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	got := readSettingsAllow(t, bundle)
	want := []string{"Read", "Grep"}
	if !equalStringSlice(got, want) {
		t.Errorf("permissions.allow = %v, want %v (binding entries only)", got, want)
	}
}

// TestRenderSettingsListerZeroGrantsLeavesBindingOnly asserts the
// no-stored-grants path: a non-nil lister that returns an empty slice
// leaves the rendered allow list equal to binding.ToolsAllowed.
func TestRenderSettingsListerZeroGrantsLeavesBindingOnly(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := fixtureBinding()
	stub := &stubGrantsLister{grants: []domain.PermissionGrant{}}
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, stub); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	got := readSettingsAllow(t, bundle)
	want := []string{"Read", "Grep"}
	if !equalStringSlice(got, want) {
		t.Errorf("permissions.allow = %v, want %v (binding entries only)", got, want)
	}
	if stub.callCount != 1 {
		t.Errorf("lister callCount = %d, want 1", stub.callCount)
	}
	if got, want := stub.gotProjectID, fixtureProject().ID; got != want {
		t.Errorf("lister projectID = %q, want %q", got, want)
	}
	if got, want := stub.gotKind, fixtureItem().Kind; got != want {
		t.Errorf("lister kind = %q, want %q", got, want)
	}
	if got, want := stub.gotCLIKind, string(binding.CLIKind); got != want {
		t.Errorf("lister cliKind = %q, want %q", got, want)
	}
}

// TestRenderSettingsListerThreeGrantsAppendedAfterBinding asserts the
// happy path: lister returns 3 distinct grants → the rendered
// permissions.allow contains binding.ToolsAllowed first, then the 3
// grants in lister-supplied order.
func TestRenderSettingsListerThreeGrantsAppendedAfterBinding(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := fixtureBinding() // ToolsAllowed: ["Read", "Grep"]
	stub := &stubGrantsLister{
		grants: []domain.PermissionGrant{
			grantFixture("Bash(git status)"),
			grantFixture("Bash(mage check)"),
			grantFixture("WebFetch(github.com/*)"),
		},
	}
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, stub); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	got := readSettingsAllow(t, bundle)
	want := []string{
		"Read", "Grep", // binding first
		"Bash(git status)", "Bash(mage check)", "WebFetch(github.com/*)", // grants after
	}
	if !equalStringSlice(got, want) {
		t.Errorf("permissions.allow = %v, want %v", got, want)
	}
}

// TestRenderSettingsListerDuplicateRuleDeduped asserts the dedup
// invariant: a grant whose Rule already appears in binding.ToolsAllowed
// is dropped from the merged list (single entry, binding-position
// preserved).
func TestRenderSettingsListerDuplicateRuleDeduped(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := fixtureBinding() // ToolsAllowed: ["Read", "Grep"]
	stub := &stubGrantsLister{
		grants: []domain.PermissionGrant{
			grantFixture("Read"),         // dup with binding
			grantFixture("Bash(ls -la)"), // new
			grantFixture("Grep"),         // dup with binding
		},
	}
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, stub); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	got := readSettingsAllow(t, bundle)
	want := []string{"Read", "Grep", "Bash(ls -la)"}
	if !equalStringSlice(got, want) {
		t.Errorf("permissions.allow = %v, want %v (duplicates of binding entries collapsed)", got, want)
	}
}

// TestRenderSettingsListerErrorWrapsAndRollsBack asserts the
// failure-propagation path: a lister error fails Render with an error
// that wraps the underlying lister error AND triggers the bundle
// rollback (system-prompt.md + plugin/ subtree all gone).
func TestRenderSettingsListerErrorWrapsAndRollsBack(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	listerErr := errors.New("storage: connection refused")
	stub := &stubGrantsLister{err: listerErr}

	_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), stub)
	if err == nil {
		t.Fatalf("Render() error = nil, want lister error")
	}
	if !errors.Is(err, listerErr) {
		t.Errorf("Render() error = %v, want errors.Is(listerErr)", err)
	}

	// Rollback: all artifacts Render created must be gone.
	gone := []string{
		filepath.Join(bundle.Paths.Root, "system-prompt.md"),
		filepath.Join(bundle.Paths.Root, "plugin"),
	}
	for _, p := range gone {
		if _, statErr := os.Stat(p); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("path %q still exists after lister-error rollback (statErr=%v)", p, statErr)
		}
	}
}

// TestRenderSettingsEmptyCLIKindSkipsLookup asserts the storage-key
// short-circuit: a binding with empty CLIKind never hits the lister
// because the storage UNIQUE composite forbids "" cli_kind.
func TestRenderSettingsEmptyCLIKindSkipsLookup(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	binding := fixtureBinding()
	binding.CLIKind = ""
	stub := &stubGrantsLister{
		grants: []domain.PermissionGrant{grantFixture("Bash(should-not-appear)")},
	}
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, stub); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	got := readSettingsAllow(t, bundle)
	want := []string{"Read", "Grep"}
	if !equalStringSlice(got, want) {
		t.Errorf("permissions.allow = %v, want %v (binding only — empty CLIKind skips lister)", got, want)
	}
	if stub.callCount != 0 {
		t.Errorf("lister callCount = %d, want 0 (empty CLIKind must not invoke lister)", stub.callCount)
	}
}

// equalStringSlice reports element-equal for two string slices. nil and
// empty are treated as equal so callers using nonNilStringSlice's
// substitution don't fail spuriously.
func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
