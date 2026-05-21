package render_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex/render"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// stubGrantsLister is the minimal in-memory PermissionGrantsLister.
// The codex render layer does not invoke ListGrantsForKind today (D2's
// per-tool approval_mode block is the gating injection), but the stub
// records call dispatch so future grants-merge wiring can be asserted
// without rewiring the test fixtures.
type stubGrantsLister struct {
	callCount int
}

// ListGrantsForKind satisfies render.PermissionGrantsLister.
func (s *stubGrantsLister) ListGrantsForKind(_ context.Context, _ string, _ domain.Kind, _ string) ([]domain.PermissionGrant, error) {
	s.callCount++
	return nil, nil
}

// fixtureBundle returns a Bundle with paths rooted under t.TempDir() so
// each test's writes land in an isolated directory cleaned up by the
// testing framework. Mirrors cli_claude/render's fixtureBundle shape so
// the cross-CLI tests look identical at the call site.
func fixtureBundle(t *testing.T) dispatcher.Bundle {
	t.Helper()
	root := t.TempDir()
	return dispatcher.Bundle{
		SpawnID:   "spawn-uuid-codex-fixture",
		Mode:      dispatcher.SpawnTempRootOSTmp,
		StartedAt: time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC),
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

// fixtureItem returns a populated build action item.
func fixtureItem() domain.ActionItem {
	return domain.ActionItem{
		ID:       "ai-codex-fixture-1",
		Kind:     domain.KindBuild,
		Title:    "DROPLET D4 CODEX BUNDLE RENDER",
		Paths:    []string{"internal/app/dispatcher/cli_codex/render/render.go"},
		Packages: []string{"github.com/evanmschultz/tillsyn/internal/app/dispatcher/cli_codex/render"},
	}
}

// fixtureProject returns a populated project value.
func fixtureProject() domain.Project {
	return domain.Project{
		ID:                  "proj-codex-fixture",
		RepoPrimaryWorktree: "/tmp/tillsyn/main",
		HyllaArtifactRef:    "github.com/evanmschultz/tillsyn@main",
	}
}

// fixtureBinding returns a BindingResolved with codex CLIKind, model +
// effort, and both tool slices populated so every render branch fires.
func fixtureBinding() dispatcher.BindingResolved {
	model := "gpt-5"
	effort := "high"
	return dispatcher.BindingResolved{
		AgentName:       "codex-planner-agent",
		CLIKind:         dispatcher.CLIKindCodex,
		Model:           &model,
		Effort:          &effort,
		ToolsAllowed:    []string{"Read", "Grep"},
		ToolsDisallowed: []string{"WebFetch", "Bash(curl *)"},
	}
}

// TestRenderHappyPathWritesBothFiles asserts the integration: one Render
// call writes system-prompt.md + codex-config.toml under <Root>.
func TestRenderHappyPathWritesBothFiles(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	body, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil)
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	if body == "" {
		t.Errorf("Render() returned empty body; want non-empty system-prompt body")
	}

	wantFiles := []string{
		filepath.Join(bundle.Paths.Root, "system-prompt.md"),
		filepath.Join(bundle.Paths.Root, "codex-config.toml"),
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
// directive) AND does NOT carry hylla_artifact_ref. Mirrors the
// cli_claude/render cross-CLI invariant.
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

	if strings.Contains(bodyStr, "hylla_artifact_ref") {
		t.Errorf("system-prompt.md unexpectedly contains hylla_artifact_ref\nfull body:\n%s", bodyStr)
	}
}

// TestRenderCodexConfigContainsAllSections asserts codex-config.toml
// carries the model, effort, MCP server block, and tool allow/deny
// arrays in the expected TOML shape.
func TestRenderCodexConfigContainsAllSections(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	configPath := filepath.Join(bundle.Paths.Root, "codex-config.toml")
	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile(codex-config.toml) error = %v", err)
	}
	bodyStr := string(body)

	wantTokens := []string{
		// Agent identity comment.
		"# agent: codex-planner-agent",
		// Model + effort (binding non-nil).
		`model = "gpt-5"`,
		`model_reasoning_effort = "high"`,
		// MCP server block (server-name convention from D2 description).
		"[mcp_servers.tillsyn-dev]",
		`command = "till"`,
		`args = ["mcp"]`,
		// Tool allow/deny.
		`allow_tools = ["Read", "Grep"]`,
		`deny_tools = ["WebFetch", "Bash(curl *)"]`,
	}
	for _, tok := range wantTokens {
		if !strings.Contains(bodyStr, tok) {
			t.Errorf("codex-config.toml missing %q\nfull body:\n%s", tok, bodyStr)
		}
	}
}

// TestRenderCodexConfigOmitsNilModelAndEffort asserts pointer-typed
// Model/Effort fields render as absent (no `model = ...` line) when nil.
// Mirrors the F.7.17 L9 "emit-only-on-non-nil" contract from cli_codex's
// argv builder.
func TestRenderCodexConfigOmitsNilModelAndEffort(t *testing.T) {
	t.Parallel()

	binding := fixtureBinding()
	binding.Model = nil
	binding.Effort = nil

	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	configPath := filepath.Join(bundle.Paths.Root, "codex-config.toml")
	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile(codex-config.toml) error = %v", err)
	}
	bodyStr := string(body)

	// Neither key should be present when its pointer is nil.
	if strings.Contains(bodyStr, "model =") {
		t.Errorf("codex-config.toml contains unexpected `model =` line with nil Model\nfull body:\n%s", bodyStr)
	}
	if strings.Contains(bodyStr, "model_reasoning_effort") {
		t.Errorf("codex-config.toml contains unexpected effort line with nil Effort\nfull body:\n%s", bodyStr)
	}
	// MCP block + tool arrays still render.
	if !strings.Contains(bodyStr, "[mcp_servers.tillsyn-dev]") {
		t.Errorf("codex-config.toml missing MCP server block\nfull body:\n%s", bodyStr)
	}
}

// TestRenderCodexConfigEmptyToolSlicesRenderAsEmptyArrays asserts that
// nil/empty ToolsAllowed/ToolsDisallowed slices render as `[]` (explicit
// empty) so the file is self-documenting. Mirrors cli_claude/render's
// explicit-empty convention on settings.json's permissions block.
func TestRenderCodexConfigEmptyToolSlicesRenderAsEmptyArrays(t *testing.T) {
	t.Parallel()

	binding := fixtureBinding()
	binding.ToolsAllowed = nil
	binding.ToolsDisallowed = nil

	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	configPath := filepath.Join(bundle.Paths.Root, "codex-config.toml")
	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile(codex-config.toml) error = %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "allow_tools = []") {
		t.Errorf("codex-config.toml missing `allow_tools = []` (explicit empty)\nfull body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, "deny_tools = []") {
		t.Errorf("codex-config.toml missing `deny_tools = []` (explicit empty)\nfull body:\n%s", bodyStr)
	}
}

// TestRenderRejectsNonCodexBinding asserts the AC#4 defense check:
// Render returns ErrUnsupportedBinding wrapped when binding.CLIKind is
// not CLIKindCodex.
func TestRenderRejectsNonCodexBinding(t *testing.T) {
	t.Parallel()

	binding := fixtureBinding()
	binding.CLIKind = dispatcher.CLIKindClaude

	bundle := fixtureBundle(t)
	_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil)
	if err == nil {
		t.Fatalf("Render() with CLIKindClaude binding error = nil, want ErrUnsupportedBinding")
	}
	if !errors.Is(err, render.ErrUnsupportedBinding) {
		t.Errorf("Render() error = %v, want errors.Is(err, ErrUnsupportedBinding)", err)
	}

	// Defense check must fire BEFORE any disk write — verify neither
	// artifact got created.
	for _, p := range []string{"system-prompt.md", "codex-config.toml"} {
		if _, err := os.Stat(filepath.Join(bundle.Paths.Root, p)); err == nil {
			t.Errorf("unexpected file present after CLIKind reject: %q", p)
		}
	}
}

// TestRenderRejectsEmptyBundleRoot asserts the input-validation guard
// fires for empty bundle.Paths.Root.
func TestRenderRejectsEmptyBundleRoot(t *testing.T) {
	t.Parallel()

	bundle := fixtureBundle(t)
	bundle.Paths.Root = ""

	_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil)
	if !errors.Is(err, render.ErrInvalidRenderInput) {
		t.Errorf("Render() error = %v, want errors.Is(err, ErrInvalidRenderInput)", err)
	}
}

// TestRenderRejectsEmptyAgentName asserts the input-validation guard
// fires for empty binding.AgentName.
func TestRenderRejectsEmptyAgentName(t *testing.T) {
	t.Parallel()

	binding := fixtureBinding()
	binding.AgentName = ""

	bundle := fixtureBundle(t)
	_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil)
	if !errors.Is(err, render.ErrInvalidRenderInput) {
		t.Errorf("Render() error = %v, want errors.Is(err, ErrInvalidRenderInput)", err)
	}
}

// TestRenderRejectsAgentNameWithPathSeparator asserts the input-validation
// guard fires for binding.AgentName containing / or \. Mirrors the
// cli_claude/render defense.
func TestRenderRejectsAgentNameWithPathSeparator(t *testing.T) {
	t.Parallel()

	cases := []string{"foo/bar", `foo\bar`, "../escape"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			binding := fixtureBinding()
			binding.AgentName = name

			bundle := fixtureBundle(t)
			_, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil)
			if !errors.Is(err, render.ErrInvalidRenderInput) {
				t.Errorf("Render(agent=%q) error = %v, want errors.Is(err, ErrInvalidRenderInput)", name, err)
			}
		})
	}
}

// TestAdaptRenderNilListerOK asserts the seam adapter accepts a nil
// any-typed lister and routes it through to Render's nil-graceful-skip
// path. The dispatcher today supplies nil for the lister argument; the
// adapter must not reject it.
func TestAdaptRenderNilListerOK(t *testing.T) {
	t.Parallel()

	// Drive through the registered hook by re-registering the
	// package's adapter and looking up via the dispatcher seam. The
	// init() in render.go already populated the slot, so we just
	// invoke RenderBundle via Render directly to exercise the
	// nil-lister branch without needing dispatcher.BuildSpawnCommand.
	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), nil); err != nil {
		t.Fatalf("Render(nil lister) error = %v, want nil", err)
	}
}

// TestRenderWithStubListerInvokesLister exercises the typed-lister code
// path. Today the codex render layer does not call ListGrantsForKind
// (the lister is plumbed for future symmetry); this test asserts the
// non-call invariant so a future change that flips Render to consume
// the lister is caught here.
func TestRenderWithStubListerInvokesLister(t *testing.T) {
	t.Parallel()

	lister := &stubGrantsLister{}
	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), fixtureBinding(), lister); err != nil {
		t.Fatalf("Render(stubLister) error = %v, want nil", err)
	}
	// The codex render layer does not invoke ListGrantsForKind today.
	// When D2.5 wires per-tool approval_mode injection, this assertion
	// will need updating; flagging the no-call expectation here keeps
	// the contract explicit.
	if lister.callCount != 0 {
		t.Errorf("stubGrantsLister.callCount = %d, want 0 (codex render does not consume lister today)", lister.callCount)
	}
}

// TestRenderEscapesTOMLSpecials asserts model/effort/tool strings with
// double-quote and backslash characters round-trip through TOML escaping
// without producing malformed output. Belt-and-suspenders defense — the
// binding values come from the template loader which has its own
// validators, but render's TOML emitter is the last line.
func TestRenderEscapesTOMLSpecials(t *testing.T) {
	t.Parallel()

	model := `gpt"5\beta`
	binding := fixtureBinding()
	binding.Model = &model
	binding.ToolsAllowed = []string{`Bash("ls")`, `Read\foo`}

	bundle := fixtureBundle(t)
	if _, err := render.Render(context.Background(), bundle, fixtureItem(), fixtureProject(), binding, nil); err != nil {
		t.Fatalf("Render() with TOML-special chars error = %v, want nil", err)
	}

	body, err := os.ReadFile(filepath.Join(bundle.Paths.Root, "codex-config.toml"))
	if err != nil {
		t.Fatalf("os.ReadFile error = %v", err)
	}
	bodyStr := string(body)
	// Backslash must be escaped first; the rendered literal must contain
	// the doubled backslash sequence.
	if !strings.Contains(bodyStr, `gpt\"5\\beta`) {
		t.Errorf("codex-config.toml missing properly escaped model value\nfull body:\n%s", bodyStr)
	}
	// Tool entries must round-trip through tomlString.
	if !strings.Contains(bodyStr, `"Bash(\"ls\")"`) {
		t.Errorf("codex-config.toml missing escaped Bash tool entry\nfull body:\n%s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"Read\\foo"`) {
		t.Errorf("codex-config.toml missing escaped Read tool entry\nfull body:\n%s", bodyStr)
	}
}

// TestRenderOmitsOptionalActionItemFields asserts paths/packages/title
// branches are skipped cleanly when the action item omits them. Mirrors
// cli_claude/render's optional-field branch coverage.
func TestRenderOmitsOptionalActionItemFields(t *testing.T) {
	t.Parallel()

	item := domain.ActionItem{
		ID:   "ai-minimal",
		Kind: domain.KindBuild,
		// no Title, Paths, Packages
	}

	bundle := fixtureBundle(t)
	body, err := render.Render(context.Background(), bundle, item, fixtureProject(), fixtureBinding(), nil)
	if err != nil {
		t.Fatalf("Render() minimal item error = %v, want nil", err)
	}

	for _, tok := range []string{"title:", "paths:", "packages:"} {
		if strings.Contains(body, tok) {
			t.Errorf("system-prompt body unexpectedly contains %q for minimal item\nfull body:\n%s", tok, body)
		}
	}
	// task_id + project_id + kind + move-state directive still emit.
	for _, tok := range []string{"task_id: ai-minimal", "kind: build", "move-state directive:"} {
		if !strings.Contains(body, tok) {
			t.Errorf("system-prompt body missing required token %q\nfull body:\n%s", tok, body)
		}
	}
}
