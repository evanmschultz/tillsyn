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

// --- W3.D2: 3-tier agent-body resolver tests --------------------------------
//
// These tests exercise render.assembleAgentFileBody's 3-tier resolution path:
// (1) project tier — <project.RepoPrimaryWorktree>/.tillsyn/agents/<basename>
// (2) user tier    — <user-home>/.tillsyn/agents/<group>/<basename>
// (3) embedded tier — templates.DefaultTemplateFS via
//                     builtin/agents/<group>/<basename> with cross-group
//                     fallback to till-gen/<basename> on fs.ErrNotExist.
//
// The tests do NOT call Render() — they call the integration surface via
// the existing Render() entrypoint when convenient (project tier + user tier
// tests do this so we exercise the renderAgentFile signature change end-to-
// end), but the cross-group fallback + ErrAgentBodyNotFound tests need direct
// access to the rendered agent file's bytes. Both styles use the standard
// Render() entry to keep tests black-box.
//
// `<group>` defaults to "till-go" when binding.SystemPromptTemplatePath is
// empty (W3-FF5 LOCKED — `path.Dir`, NOT `filepath.Dir`, on the path).

// agentTierProjectFixture writes a project-tier agent file at
// <projectDir>/.tillsyn/agents/<basename> with the supplied sentinel
// content.
func agentTierProjectFixture(t *testing.T, projectDir, basename, content string) {
	t.Helper()
	dir := filepath.Join(projectDir, ".tillsyn", "agents")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("seed project-tier dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, basename), []byte(content), 0o600); err != nil {
		t.Fatalf("seed project-tier file: %v", err)
	}
}

// agentTierUserFixture writes a user-tier agent file at
// $HOME/.tillsyn/agents/<group>/<basename> via t.Setenv("HOME", homeDir)
// so os.UserHomeDir resolves to homeDir for the duration of the test.
func agentTierUserFixture(t *testing.T, homeDir, group, basename, content string) {
	t.Helper()
	t.Setenv("HOME", homeDir)
	dir := filepath.Join(homeDir, ".tillsyn", "agents", group)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("seed user-tier dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, basename), []byte(content), 0o600); err != nil {
		t.Fatalf("seed user-tier file: %v", err)
	}
}

// readRenderedAgentFile reads the rendered <bundle>/plugin/agents/<name>.md
// file bytes after a successful Render() call.
func readRenderedAgentFile(t *testing.T, bundleRoot, agentName string) string {
	t.Helper()
	p := filepath.Join(bundleRoot, "plugin", "agents", agentName+".md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("os.ReadFile(%s): %v", p, err)
	}
	return string(b)
}

// TestAssembleAgentFileBody_EmbeddedDefault asserts that with no project-tier
// override, no user-tier override, and an empty SystemPromptTemplatePath, the
// resolver falls all the way through to the embedded tier and returns the
// content of builtin/agents/till-go/<AgentName>.md (the dogfood default group).
func TestAssembleAgentFileBody_EmbeddedDefault(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used to neutralize HOME.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir) // empty user tier; no .tillsyn/agents subdir

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir() // empty project tier

	// AgentName "go-builder-agent" maps to till-go/go-builder-agent.md
	// (verified embedded at internal/templates/embed.go:98).
	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		// SystemPromptTemplatePath empty → group defaults to till-go.
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	// The embedded placeholder carries the PLACEHOLDER marker on line 6.
	if !strings.Contains(body, "# PLACEHOLDER") {
		t.Errorf("embedded-tier body missing PLACEHOLDER marker\nbody:\n%s", body)
	}
	// Frontmatter from the embedded file must survive into the rendered
	// output (D3 may later strip / inject; D2 emits verbatim).
	if !strings.Contains(body, "name: ") {
		t.Errorf("embedded-tier body missing frontmatter `name:` line\nbody:\n%s", body)
	}
}

// TestAssembleAgentFileBody_UserOverride asserts the user tier wins over the
// embedded tier when $HOME/.tillsyn/agents/<group>/<basename> exists.
func TestAssembleAgentFileBody_UserOverride(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()

	const sentinel = "SENTINEL_USER_TIER"
	agentTierUserFixture(t, homeDir, "till-go", "go-builder-agent.md",
		"---\nname: go-builder-agent\n---\n\n"+sentinel+"\n")

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir() // empty project tier

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	if !strings.Contains(body, sentinel) {
		t.Errorf("user-tier body missing %q sentinel\nbody:\n%s", sentinel, body)
	}
}

// TestAssembleAgentFileBody_ProjectOverride asserts the project tier wins
// over both the user tier and the embedded tier when
// <project>/.tillsyn/agents/<basename> exists.
func TestAssembleAgentFileBody_ProjectOverride(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()

	// Plant the user-tier sentinel so we can confirm project tier wins.
	agentTierUserFixture(t, homeDir, "till-go", "go-builder-agent.md",
		"---\nname: go-builder-agent\n---\n\nSENTINEL_USER_TIER\n")

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir()

	const projectSentinel = "SENTINEL_PROJECT_TIER"
	agentTierProjectFixture(t, project.RepoPrimaryWorktree, "go-builder-agent.md",
		"---\nname: go-builder-agent\n---\n\n"+projectSentinel+"\n")

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	if !strings.Contains(body, projectSentinel) {
		t.Errorf("project-tier body missing %q sentinel\nbody:\n%s", projectSentinel, body)
	}
	if strings.Contains(body, "SENTINEL_USER_TIER") {
		t.Errorf("project-tier body unexpectedly contains user-tier sentinel "+
			"(project tier should win)\nbody:\n%s", body)
	}
}

// TestAssembleAgentFileBody_CrossGroupFallbackToTillGen asserts the W3-FF7
// LOCKED cross-group fallback: when the primary embedded-tier lookup at
// builtin/agents/<group>/<basename> misses with fs.ErrNotExist, the resolver
// falls back to builtin/agents/till-gen/<basename>.
//
// AgentName "orchestrator-managed" with empty SystemPromptTemplatePath
// resolves group "till-go" → builtin/agents/till-go/orchestrator-managed.md
// MISS → fallback builtin/agents/till-gen/orchestrator-managed.md HIT.
func TestAssembleAgentFileBody_CrossGroupFallbackToTillGen(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir) // empty user tier

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir() // empty project tier

	binding := dispatcher.BindingResolved{
		AgentName: "orchestrator-managed",
		CLIKind:   dispatcher.CLIKindClaude,
		// Empty SystemPromptTemplatePath → group defaults to till-go.
		// till-go/orchestrator-managed.md DOES NOT EXIST → fallback to
		// till-gen/orchestrator-managed.md.
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	// Cross-group fallback fired → till-gen/orchestrator-managed.md content.
	if !strings.Contains(body, "orchestrator-managed coordination kinds") {
		t.Errorf("cross-group fallback did not surface till-gen/orchestrator-managed.md content\nbody:\n%s", body)
	}
}

// TestAssembleAgentFileBody_CrossGroupFallbackMissesBothGroups asserts that
// when BOTH the primary group AND the till-gen fallback miss, the resolver
// returns a wrapped ErrAgentBodyNotFound sentinel — Render's rollback then
// runs and the partially-written bundle is cleaned up.
func TestAssembleAgentFileBody_CrossGroupFallbackMissesBothGroups(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir) // empty user tier

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir() // empty project tier

	binding := dispatcher.BindingResolved{
		AgentName: "nonexistent-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		// Empty SystemPromptTemplatePath → group "till-go".
		// Neither till-go/nonexistent-agent.md nor till-gen/nonexistent-agent.md
		// exists in the embedded FS → ErrAgentBodyNotFound.
	}

	_, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil)
	if err == nil {
		t.Fatalf("Render() error = nil, want ErrAgentBodyNotFound")
	}
	if !errors.Is(err, render.ErrAgentBodyNotFound) {
		t.Errorf("Render() error = %v, want errors.Is(ErrAgentBodyNotFound)", err)
	}
}

// --- W3.D3: frontmatter strip-then-inject pipeline tests --------------------
//
// D3 layers a strip-then-inject pipeline on top of D2's 3-tier resolver:
//
//   1. Split the resolved body at the leading + trailing `---\n` delimiters
//      to extract (frontmatter, postFrontmatter).
//   2. Strip template-time frontmatter keys via config.StripFrontmatterKeys.
//      Strip predicates (LOCKED):
//        - stripModel = binding.Model != nil && *binding.Model != "" (W3-FF2).
//        - stripTools = true ALWAYS (W3-FF12) — tool-gating keys are
//          template-time only; runtime per-spawn injection is the sole
//          authoritative source.
//   3. Inject runtime per-spawn fields. When binding.ToolsAllowed is
//      non-empty append `allowedTools: <comma-joined>`; same for
//      binding.ToolsDisallowed.
//   4. Re-concatenate `---\n` + stripped+injected frontmatter + `---\n` +
//      postFrontmatter.
//
// The tests below seed user-tier agent files (so test fixture content is
// controlled at runtime, not from embedded placeholders) and assert the
// expected strip + inject outcome.

// ptrString returns a pointer to the supplied string. Convenience helper for
// strip-predicate test cases that need a non-nil *string.
func ptrString(s string) *string {
	return &s
}

// d3UserTierFrontmatter constructs an agent body with the supplied
// frontmatter lines and a fixed body section. Test helper for D3 strip
// tests so individual test cases focus on frontmatter shape.
func d3UserTierFrontmatter(frontmatterLines string) string {
	return "---\n" + frontmatterLines + "---\n\nbody-bytes-preserve-marker\n"
}

// TestAssembleAgentFileBody_FrontmatterStripModelOnAgentsTOMLSet asserts the
// `model:` line is removed from the rendered frontmatter when the binding
// carries a non-empty *Model — i.e. agents.toml's model field set.
func TestAssembleAgentFileBody_FrontmatterStripModelOnAgentsTOMLSet(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()

	agentTierUserFixture(t, homeDir, "till-go", "go-builder-agent.md",
		d3UserTierFrontmatter("name: go-builder-agent\nmodel: opus\n"))

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir()

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		Model:     ptrString("sonnet"), // agents.toml SET model → strip.
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	if strings.Contains(body, "model:") {
		t.Errorf("rendered body unexpectedly contains `model:` line\nbody:\n%s", body)
	}
	// Post-frontmatter body bytes preserved.
	if !strings.Contains(body, "body-bytes-preserve-marker") {
		t.Errorf("rendered body lost post-frontmatter marker\nbody:\n%s", body)
	}
	// `name:` survives strip (not in strip universe).
	if !strings.Contains(body, "name: go-builder-agent") {
		t.Errorf("rendered body missing `name:` line\nbody:\n%s", body)
	}
}

// TestAssembleAgentFileBody_FrontmatterStripToolsOnAgentsTOMLSet asserts the
// `tools:` + `allowedTools:` + `disallowedTools:` lines are removed from the
// rendered frontmatter (strip universe per frontmatter.go:51), and then the
// runtime per-spawn `allowedTools:` line is re-injected from binding.
func TestAssembleAgentFileBody_FrontmatterStripToolsOnAgentsTOMLSet(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()

	// Stale template-time tool-gating keys land in the user-tier fixture
	// frontmatter — strip must remove them all so the runtime inject step
	// is the sole source.
	agentTierUserFixture(t, homeDir, "till-go", "go-builder-agent.md",
		d3UserTierFrontmatter(
			"name: go-builder-agent\n"+
				"tools: Read, Bash\n"+
				"allowedTools: Read\n"+
				"disallowedTools: WebFetch\n"))

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir()

	binding := dispatcher.BindingResolved{
		AgentName:    "go-builder-agent",
		CLIKind:      dispatcher.CLIKindClaude,
		ToolsAllowed: []string{"Read"}, // runtime per-spawn value.
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	// Stale template-time keys all stripped.
	if strings.Contains(body, "tools: Read, Bash") {
		t.Errorf("rendered body unexpectedly contains stale `tools: Read, Bash` line\nbody:\n%s", body)
	}
	// `disallowedTools:` from disk frontmatter was stripped; binding has
	// no ToolsDisallowed so no inject either → no disallowedTools line.
	if strings.Contains(body, "disallowedTools:") {
		t.Errorf("rendered body unexpectedly contains `disallowedTools:` line "+
			"(binding empty + strip ran)\nbody:\n%s", body)
	}
	// Runtime allowedTools injected from binding (NOT the stale disk
	// value). Match the exact comma-joined form so the inject path is
	// verified rather than the stripped-then-passed-through accident.
	if !strings.Contains(body, "allowedTools: Read") {
		t.Errorf("rendered body missing injected `allowedTools: Read`\nbody:\n%s", body)
	}
	// `name:` survives.
	if !strings.Contains(body, "name: go-builder-agent") {
		t.Errorf("rendered body missing `name:` line\nbody:\n%s", body)
	}
}

// --- W3.D2 Round 2: W3-D23-FF1 path-traversal regression -------------------
//
// build-QA-falsification on W3.D2 found a HIGH-severity path-traversal vector
// via an attacker-controllable `binding.SystemPromptTemplatePath` value:
//
//   binding.SystemPromptTemplatePath = "till-go/../../../../../../etc/passwd"
//
// On the attack input, path.Base returns "passwd" (passes the existing
// validateAgentBasename leaf check) and path.Dir returns
// "../../../../etc" (UNVALIDATED prior to round-2). The user tier then calls
// filepath.Join(home, ".tillsyn/agents", "../../../../etc", "passwd")
// which filepath.Clean cancels down to /etc/passwd, and os.ReadFile
// succeeds when /etc/passwd exists.
//
// Threat model: today bounded (SystemPromptTemplatePath comes from repo-
// owned till-*.toml templates), but becomes attacker-controllable as team-
// aware architecture matures (per memory feedback_prompt_injection_team.md
// + project_team_aware_architecture.md).
//
// Round-2 fix: introduce a full-path validator (validateAgentTemplatePath)
// that runs at the top of assembleAgentFileBody — BEFORE path.Dir / path.Base
// derivation — and rejects:
//   - absolute paths (starts with "/")
//   - any segment equal to ".."
//   - empty segments (catches "//foo" cases)
//
// Empty SystemPromptTemplatePath is STILL allowed (the "use embedded
// till-go default" sentinel per W3-FF5 LOCKED).

// TestAssembleAgentFileBody_RejectsPathTraversalInGroup pins the exact
// W3-D23-FF1 attack string and asserts the new validator rejects it via
// the new ErrInvalidAgentTemplatePath sentinel.
//
// Positive control: os.Stat("/etc/passwd") is checked first; if the file
// is absent on the host (rare on POSIX, expected on some sandboxes), the
// test t.Skip's because the original attack only succeeds when the
// traversal target actually exists — the assertion would be moot on a
// host lacking /etc/passwd.
func TestAssembleAgentFileBody_RejectsPathTraversalInGroup(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	if _, err := os.Stat("/etc/passwd"); err != nil {
		t.Skipf("/etc/passwd unavailable on this host (%v); the W3-D23-FF1 attack requires the traversal target to exist", err)
	}

	bundle := fixtureBundle(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir) // empty user tier

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir() // empty project tier

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		// W3-D23-FF1 attack string verbatim — the path used by the
		// build-QA-falsification finding. path.Base="passwd",
		// path.Dir="till-go/../../../../../../etc". Without the new
		// validator the user-tier filepath.Join collapses to
		// /etc/passwd and os.ReadFile succeeds.
		SystemPromptTemplatePath: "till-go/../../../../../../etc/passwd",
	}

	_, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil)
	if err == nil {
		t.Fatalf("Render() error = nil, want ErrInvalidAgentTemplatePath (W3-D23-FF1 traversal must be rejected)")
	}
	if !errors.Is(err, render.ErrInvalidAgentTemplatePath) {
		t.Errorf("Render() error = %v, want errors.Is(ErrInvalidAgentTemplatePath)", err)
	}

	// Defense-in-depth assertion: the rendered agent file MUST NOT exist
	// on disk after the rejection (the rollback wiped it). If a future
	// regression leaks the /etc/passwd content into the agent body, this
	// catches it independently of the sentinel-error check.
	agentPath := filepath.Join(bundle.Paths.Root, "plugin", "agents", binding.AgentName+".md")
	if _, statErr := os.Stat(agentPath); !errors.Is(statErr, os.ErrNotExist) {
		body, _ := os.ReadFile(agentPath)
		t.Errorf("rejected-traversal agent file unexpectedly present at %q\nbody:\n%s", agentPath, body)
	}
}

// TestAssembleAgentFileBody_RejectsPathTraversalSiblingCases covers the
// adjacent attack vectors the validator's reject-rules also must block:
// absolute paths, bare ".." segments, and traversal segments at non-leaf
// positions.
func TestAssembleAgentFileBody_RejectsPathTraversalSiblingCases(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		// Absolute path — would read directly from the filesystem root
		// at the user tier (filepath.Join discards earlier segments
		// when a later segment starts with "/" on POSIX).
		{"absolute_etc_passwd", "/etc/passwd"},
		// Bare ".." at end of path — path.Base returns ".." (also
		// caught by the existing basename validator, but the
		// full-path validator should fail-first so the error sentinel
		// is consistent across all traversal shapes).
		{"trailing_dotdot", "till-go/.."},
		// ".." segment positioned at leaf with a non-".." follow-on
		// would not be reachable here (the leaf IS the last segment),
		// but a ".." sandwiched mid-path is — exercise that.
		{"mid_path_dotdot", "till-go/../passwd"},
		// Empty intermediate segment via consecutive slashes — should
		// be rejected as the validator's empty-segment rule.
		{"double_slash", "till-go//passwd"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Cannot run t.Parallel — t.Setenv used by sibling
			// fixtures and we want HOME isolated per subtest.
			bundle := fixtureBundle(t)
			homeDir := t.TempDir()
			t.Setenv("HOME", homeDir)

			project := fixtureProject()
			project.RepoPrimaryWorktree = t.TempDir()

			binding := dispatcher.BindingResolved{
				AgentName:                "go-builder-agent",
				CLIKind:                  dispatcher.CLIKindClaude,
				SystemPromptTemplatePath: tc.path,
			}

			_, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil)
			if err == nil {
				t.Fatalf("Render() error = nil for %q, want ErrInvalidAgentTemplatePath", tc.path)
			}
			if !errors.Is(err, render.ErrInvalidAgentTemplatePath) {
				t.Errorf("Render() error = %v for %q, want errors.Is(ErrInvalidAgentTemplatePath)", err, tc.path)
			}
		})
	}
}

// TestAssembleAgentFileBody_AcceptsLegitimateTemplatePath is the positive
// control: a legitimate `till-<group>/<file>.md` path still resolves
// without the new validator rejecting it. Ensures the round-2 defense
// does not over-reject and break the W3-FF5 LOCKED dogfood path.
func TestAssembleAgentFileBody_AcceptsLegitimateTemplatePath(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir) // empty user tier

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir() // empty project tier

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		// Legitimate canonical form: `till-<group>/<basename>.md`.
		// Resolves via embedded-tier till-go/go-builder-agent.md.
		SystemPromptTemplatePath: "till-go/go-builder-agent.md",
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil (legitimate path must not be rejected)", err)
	}

	// Agent file rendered successfully.
	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	if body == "" {
		t.Errorf("rendered agent file is empty; legitimate path should produce non-empty body")
	}
}

// TestAssembleAgentFileBody_EmptyPathStillRoutesToTillGoDefault asserts
// the empty-SystemPromptTemplatePath sentinel (W3-FF5 LOCKED — use
// embedded till-go default) survives the round-2 validator addition.
// The validator MUST NOT reject the empty-string path; the empty branch
// short-circuits BEFORE the validator runs.
func TestAssembleAgentFileBody_EmptyPathStillRoutesToTillGoDefault(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir) // empty user tier

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir() // empty project tier

	binding := dispatcher.BindingResolved{
		AgentName: "go-builder-agent",
		CLIKind:   dispatcher.CLIKindClaude,
		// Empty path → W3-FF5 LOCKED till-go default.
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil (empty path must route to till-go default, not reject)", err)
	}

	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	// The embedded placeholder for go-builder-agent carries the
	// PLACEHOLDER marker — same assertion shape as
	// TestAssembleAgentFileBody_EmbeddedDefault.
	if !strings.Contains(body, "# PLACEHOLDER") {
		t.Errorf("empty-path render did not surface till-go default body\nbody:\n%s", body)
	}
}

// TestAssembleAgentFileBody_FrontmatterPreservedWhenAgentsTOMLAbsent asserts
// the model-not-overridden + tool-gates-empty path. With binding.Model =
// ptr("") AND binding.ToolsAllowed / ToolsDisallowed nil:
//   - stripModel is FALSE (predicate is `*Model != ""` per W3-FF2), so
//     embedded `model:` is preserved through StripFrontmatterKeys.
//   - stripTools is TRUE ALWAYS (W3-FF12), so `tools:` is stripped.
//   - No inject (binding tool-gate slices empty).
//
// Test name retains "Preserved" framing for symmetry with the strip tests;
// what's preserved is BODY bytes + the `model:` line (NOT the `tools:` line —
// strip is unconditional per W3-FF12).
func TestAssembleAgentFileBody_FrontmatterPreservedWhenAgentsTOMLAbsent(t *testing.T) {
	// Cannot run t.Parallel — t.Setenv used.
	bundle := fixtureBundle(t)
	homeDir := t.TempDir()

	agentTierUserFixture(t, homeDir, "till-go", "go-builder-agent.md",
		d3UserTierFrontmatter(
			"name: go-builder-agent\n"+
				"model: opus\n"+
				"tools: Read, Bash\n"))

	project := fixtureProject()
	project.RepoPrimaryWorktree = t.TempDir()

	binding := dispatcher.BindingResolved{
		AgentName:       "go-builder-agent",
		CLIKind:         dispatcher.CLIKindClaude,
		Model:           ptrString(""), // resolver's "absent" representation per W3-FF2.
		ToolsAllowed:    nil,
		ToolsDisallowed: nil,
	}

	if _, err := render.Render(context.Background(), bundle, fixtureItem(), project, binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	body := readRenderedAgentFile(t, bundle.Paths.Root, binding.AgentName)
	// model: line preserved (stripModel false — predicate `*Model != ""`).
	if !strings.Contains(body, "model: opus") {
		t.Errorf("rendered body missing `model: opus` line (stripModel false path)\nbody:\n%s", body)
	}
	// tools: line stripped (stripTools always true per W3-FF12).
	if strings.Contains(body, "tools: Read, Bash") {
		t.Errorf("rendered body unexpectedly contains stale `tools:` line "+
			"(stripTools is unconditional per W3-FF12)\nbody:\n%s", body)
	}
	// Empty binding tool-gates skip injection — no allowedTools/disallowedTools.
	if strings.Contains(body, "allowedTools:") {
		t.Errorf("rendered body unexpectedly contains `allowedTools:` line "+
			"(binding empty → no injection)\nbody:\n%s", body)
	}
	if strings.Contains(body, "disallowedTools:") {
		t.Errorf("rendered body unexpectedly contains `disallowedTools:` line "+
			"(binding empty → no injection)\nbody:\n%s", body)
	}
	// Body bytes preserved.
	if !strings.Contains(body, "body-bytes-preserve-marker") {
		t.Errorf("rendered body lost post-frontmatter marker\nbody:\n%s", body)
	}
}
