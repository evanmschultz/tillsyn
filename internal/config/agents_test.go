// Package config — tests for the agents.toml schema + decode wiring shipped
// in agents.go (Drop 4c.6 W0.D1). Co-located with the production file per
// CLAUDE.md § "Tests" discipline.
package config

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestLoadRegistry_Baseline loads the canonical baseline.toml golden fixture
// and asserts every Preset field decoded with the expected value plus the
// [agents.build] override surfaces with a non-nil tools_allow pointer pointing
// at the expected slice. This is the golden-path proof that pelletier/go-toml/v2
// decodes the schema as designed in SKETCH.md § 4.1.
func TestLoadRegistry_Baseline(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry(filepath.Join("testdata", "agents", "baseline.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry returned error: %v", err)
	}
	if registry == nil {
		t.Fatal("LoadRegistry returned nil registry without error")
	}

	preset := registry.Preset
	if preset.Client != "claude" {
		t.Errorf("Preset.Client = %q, want %q", preset.Client, "claude")
	}
	if preset.Model != "sonnet" {
		t.Errorf("Preset.Model = %q, want %q", preset.Model, "sonnet")
	}
	if preset.Effort != "medium" {
		t.Errorf("Preset.Effort = %q, want %q", preset.Effort, "medium")
	}
	if preset.MaxTries != 3 {
		t.Errorf("Preset.MaxTries = %d, want 3", preset.MaxTries)
	}
	if preset.MaxBudgetUSD != 5.0 {
		t.Errorf("Preset.MaxBudgetUSD = %v, want 5.0", preset.MaxBudgetUSD)
	}
	if preset.MaxTurns != 40 {
		t.Errorf("Preset.MaxTurns = %d, want 40", preset.MaxTurns)
	}
	if preset.BlockedRetries != 2 {
		t.Errorf("Preset.BlockedRetries = %d, want 2", preset.BlockedRetries)
	}
	if preset.BlockedRetryCooldown != "30s" {
		t.Errorf("Preset.BlockedRetryCooldown = %q, want %q", preset.BlockedRetryCooldown, "30s")
	}
	if preset.AutoPush != false {
		t.Errorf("Preset.AutoPush = %v, want false", preset.AutoPush)
	}

	if got, want := preset.EnvSet["TILLSYN_DEV"], "1"; got != want {
		t.Errorf("Preset.EnvSet[TILLSYN_DEV] = %q, want %q", got, want)
	}
	if got, want := preset.EnvFromShell["GH_TOKEN"], "GH_TOKEN"; got != want {
		t.Errorf("Preset.EnvFromShell[GH_TOKEN] = %q, want %q", got, want)
	}

	wantCLIArgs := []string{"--strict-mcp-config"}
	if !equalStrings(preset.CliArgs, wantCLIArgs) {
		t.Errorf("Preset.CliArgs = %v, want %v", preset.CliArgs, wantCLIArgs)
	}
	wantToolsAllow := []string{"Read", "Edit", "Bash"}
	if !equalStrings(preset.ToolsAllow, wantToolsAllow) {
		t.Errorf("Preset.ToolsAllow = %v, want %v", preset.ToolsAllow, wantToolsAllow)
	}
	wantToolsDeny := []string{"WebFetch"}
	if !equalStrings(preset.ToolsDeny, wantToolsDeny) {
		t.Errorf("Preset.ToolsDeny = %v, want %v", preset.ToolsDeny, wantToolsDeny)
	}
	wantAddons := []string{"~/.claude/output-styles/tillsyn-flow.md"}
	if !equalStrings(preset.ClaudeMDAddons, wantAddons) {
		t.Errorf("Preset.ClaudeMDAddons = %v, want %v", preset.ClaudeMDAddons, wantAddons)
	}

	override, ok := registry.Overrides[domain.KindBuild]
	if !ok {
		t.Fatalf("Overrides[%q] missing", domain.KindBuild)
	}
	if override.ToolsAllow == nil {
		t.Fatal("Overrides[build].ToolsAllow is nil; want non-nil pointer")
	}
	wantBuildAllow := []string{"Read", "Edit", "Write", "Bash"}
	if !equalStrings(*override.ToolsAllow, wantBuildAllow) {
		t.Errorf("Overrides[build].ToolsAllow = %v, want %v", *override.ToolsAllow, wantBuildAllow)
	}
	if override.Model != nil {
		t.Errorf("Overrides[build].Model = %q, want nil (absent)", *override.Model)
	}
}

// TestLoadRegistry_MalformedTOML feeds a fixture with a syntax error and
// asserts the returned error wraps a *toml.DecodeError so callers can extract
// position information via errors.As. The error message must include the
// offending TOML line number.
func TestLoadRegistry_MalformedTOML(t *testing.T) {
	t.Parallel()

	_, err := LoadRegistry(filepath.Join("testdata", "agents", "malformed.toml"))
	if err == nil {
		t.Fatal("LoadRegistry returned nil error for malformed input")
	}

	var decodeErr *toml.DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("error chain does not contain *toml.DecodeError: %v", err)
	}
	row, _ := decodeErr.Position()
	if row <= 0 {
		t.Errorf("DecodeError row = %d, want > 0", row)
	}
}

// TestLoadRegistry_UnknownTopLevelField asserts the strict decoder rejects
// unknown top-level fields. Catches typos in user-facing TOML keys early
// rather than silently dropping the value.
func TestLoadRegistry_UnknownTopLevelField(t *testing.T) {
	t.Parallel()

	_, err := LoadRegistry(filepath.Join("testdata", "agents", "unknown_field.toml"))
	if err == nil {
		t.Fatal("LoadRegistry returned nil error for fixture with unknown field")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unknown") &&
		!strings.Contains(strings.ToLower(err.Error()), "missing") {
		t.Errorf("error message %q does not mention unknown/missing field", err.Error())
	}
}

// TestLoadRegistry_FileNotFound asserts that a missing file produces a clear
// error rather than panicking or returning a nil registry without an error.
func TestLoadRegistry_FileNotFound(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry(filepath.Join("testdata", "agents", "does_not_exist.toml"))
	if err == nil {
		t.Fatal("LoadRegistry returned nil error for nonexistent path")
	}
	if registry != nil {
		t.Errorf("LoadRegistry returned non-nil registry alongside error: %+v", registry)
	}
}

// TestLoadRegistry_AbsentBlocksNilSafe asserts that loading a TOML file with
// only a [agents] block and no per-kind overrides yields a usable registry
// where Overrides is initialized (non-nil) and lookups for absent kinds
// return the zero Override.
func TestLoadRegistry_AbsentBlocksNilSafe(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry(filepath.Join("testdata", "agents", "preset_only.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry returned error: %v", err)
	}
	if registry.Overrides == nil {
		t.Fatal("Overrides map is nil; want initialized empty map")
	}
	if _, ok := registry.Overrides[domain.KindBuild]; ok {
		t.Errorf("Overrides[build] unexpectedly present for preset-only fixture")
	}
}

// TestResolve_FullInherit loads a fixture with only an [agents] defaults block
// and asserts Resolve(reg, KindBuild) returns the Preset values verbatim — no
// per-kind block means pure inheritance.
func TestResolve_FullInherit(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry(filepath.Join("testdata", "agents", "inheritance_full_inherit.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry returned error: %v", err)
	}

	got, err := Resolve(registry, domain.KindBuild)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if got.Client != "claude" {
		t.Errorf("Client = %q, want %q", got.Client, "claude")
	}
	if got.Model != "sonnet" {
		t.Errorf("Model = %q, want %q", got.Model, "sonnet")
	}
	if got.Effort != "medium" {
		t.Errorf("Effort = %q, want %q", got.Effort, "medium")
	}
	if got.MaxTries != 3 {
		t.Errorf("MaxTries = %d, want 3", got.MaxTries)
	}
	if got.MaxBudgetUSD != 5.0 {
		t.Errorf("MaxBudgetUSD = %v, want 5.0", got.MaxBudgetUSD)
	}
	if got.MaxTurns != 40 {
		t.Errorf("MaxTurns = %d, want 40", got.MaxTurns)
	}
	if got.BlockedRetries != 2 {
		t.Errorf("BlockedRetries = %d, want 2", got.BlockedRetries)
	}
	if got.BlockedRetryCooldown != "30s" {
		t.Errorf("BlockedRetryCooldown = %q, want %q", got.BlockedRetryCooldown, "30s")
	}
	if got.AutoPush {
		t.Errorf("AutoPush = %v, want false", got.AutoPush)
	}
	if got.EnvSet["TILLSYN_DEV"] != "1" {
		t.Errorf("EnvSet[TILLSYN_DEV] = %q, want %q", got.EnvSet["TILLSYN_DEV"], "1")
	}
	if got.EnvFromShell["GH_TOKEN"] != "GH_TOKEN" {
		t.Errorf("EnvFromShell[GH_TOKEN] = %q, want %q", got.EnvFromShell["GH_TOKEN"], "GH_TOKEN")
	}
	if !equalStrings(got.CliArgs, []string{"--strict-mcp-config"}) {
		t.Errorf("CliArgs = %v, want [--strict-mcp-config]", got.CliArgs)
	}
	if !equalStrings(got.ToolsAllow, []string{"Read", "Edit", "Bash"}) {
		t.Errorf("ToolsAllow = %v, want [Read Edit Bash]", got.ToolsAllow)
	}
	if !equalStrings(got.ToolsDeny, []string{"WebFetch"}) {
		t.Errorf("ToolsDeny = %v, want [WebFetch]", got.ToolsDeny)
	}
	if !equalStrings(got.ClaudeMDAddons, []string{"~/.claude/output-styles/tillsyn-flow.md"}) {
		t.Errorf("ClaudeMDAddons = %v, want [tillsyn-flow.md]", got.ClaudeMDAddons)
	}
}

// TestResolve_PartialOverride asserts that a per-kind block overriding only
// MaxBudgetUSD reflects that one override while every other field falls
// through to the Preset.
func TestResolve_PartialOverride(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry(filepath.Join("testdata", "agents", "inheritance_partial_override.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry returned error: %v", err)
	}

	got, err := Resolve(registry, domain.KindBuild)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if got.MaxBudgetUSD != 9.5 {
		t.Errorf("MaxBudgetUSD = %v, want 9.5 (override)", got.MaxBudgetUSD)
	}
	// Every other field falls through to Preset.
	if got.Client != "claude" {
		t.Errorf("Client = %q, want %q (inherited)", got.Client, "claude")
	}
	if got.Model != "sonnet" {
		t.Errorf("Model = %q, want %q (inherited)", got.Model, "sonnet")
	}
	if got.MaxTries != 3 {
		t.Errorf("MaxTries = %d, want 3 (inherited)", got.MaxTries)
	}
	if got.MaxTurns != 40 {
		t.Errorf("MaxTurns = %d, want 40 (inherited)", got.MaxTurns)
	}
	if !equalStrings(got.ToolsAllow, []string{"Read", "Edit", "Bash"}) {
		t.Errorf("ToolsAllow = %v, want [Read Edit Bash] (inherited)", got.ToolsAllow)
	}
	if !equalStrings(got.ToolsDeny, []string{"WebFetch"}) {
		t.Errorf("ToolsDeny = %v, want [WebFetch] (inherited)", got.ToolsDeny)
	}
}

// TestResolve_MapMerge asserts EnvSet and EnvFromShell merge per-key — the
// per-kind block's keys add to the Preset's keys; neither side's keys are
// dropped. SKETCH.md § 4.2.2.
func TestResolve_MapMerge(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry(filepath.Join("testdata", "agents", "inheritance_map_merge.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry returned error: %v", err)
	}

	got, err := Resolve(registry, domain.KindBuild)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if v, ok := got.EnvSet["A"]; !ok || v != "1" {
		t.Errorf("EnvSet[A] = %q (present=%v), want %q present", v, ok, "1")
	}
	if v, ok := got.EnvSet["B"]; !ok || v != "2" {
		t.Errorf("EnvSet[B] = %q (present=%v), want %q present", v, ok, "2")
	}
	if len(got.EnvSet) != 2 {
		t.Errorf("len(EnvSet) = %d, want 2", len(got.EnvSet))
	}

	if v, ok := got.EnvFromShell["SHELL_A"]; !ok || v != "SHELL_A" {
		t.Errorf("EnvFromShell[SHELL_A] = %q (present=%v), want %q present", v, ok, "SHELL_A")
	}
	if v, ok := got.EnvFromShell["SHELL_B"]; !ok || v != "SHELL_B" {
		t.Errorf("EnvFromShell[SHELL_B] = %q (present=%v), want %q present", v, ok, "SHELL_B")
	}
	if len(got.EnvFromShell) != 2 {
		t.Errorf("len(EnvFromShell) = %d, want 2", len(got.EnvFromShell))
	}
}

// TestResolve_MapMergeOverrideWins asserts that when the per-kind block sets
// a key already present in the Preset map, the override value wins. Documents
// the precedence half of the per-key merge semantics.
func TestResolve_MapMergeOverrideWins(t *testing.T) {
	t.Parallel()

	registry := &AgentsRegistry{
		Preset: Preset{
			EnvSet: map[string]string{"K": "preset"},
		},
		Overrides: map[domain.Kind]Override{
			domain.KindBuild: {
				EnvSet: ptrMap(map[string]string{"K": "override"}),
			},
		},
	}

	got, err := Resolve(registry, domain.KindBuild)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if got.EnvSet["K"] != "override" {
		t.Errorf("EnvSet[K] = %q, want %q (override wins on collision)", got.EnvSet["K"], "override")
	}
}

// TestResolve_ListReplace asserts list fields full-replace when the per-kind
// block sets them — Preset's list is dropped wholesale, the override list
// replaces it. SKETCH.md § 4.2.3.
func TestResolve_ListReplace(t *testing.T) {
	t.Parallel()

	registry, err := LoadRegistry(filepath.Join("testdata", "agents", "inheritance_list_replace.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry returned error: %v", err)
	}

	got, err := Resolve(registry, domain.KindBuild)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if !equalStrings(got.ToolsAllow, []string{"Read"}) {
		t.Errorf("ToolsAllow = %v, want [Read] (full replace)", got.ToolsAllow)
	}
	if !equalStrings(got.CliArgs, []string{"--quiet"}) {
		t.Errorf("CliArgs = %v, want [--quiet] (full replace)", got.CliArgs)
	}
}

// TestResolve_ExplicitEmptyList asserts that an Override with a non-nil but
// empty list (`&[]string{}`) explicitly replaces a non-empty Preset list with
// an empty slice. The pointer-to-slice idiom carries the absent-vs-zero
// discrimination chosen at D1 and honored here.
func TestResolve_ExplicitEmptyList(t *testing.T) {
	t.Parallel()

	registry := &AgentsRegistry{
		Preset: Preset{
			ToolsDeny: []string{"rm", "WebFetch"},
		},
		Overrides: map[domain.Kind]Override{
			domain.KindBuild: {
				ToolsDeny: ptrSlice([]string{}),
			},
		},
	}

	got, err := Resolve(registry, domain.KindBuild)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if got.ToolsDeny == nil {
		t.Error("ToolsDeny = nil; want non-nil empty slice (explicit empty replaces non-empty)")
	}
	if len(got.ToolsDeny) != 0 {
		t.Errorf("ToolsDeny = %v, want [] (explicit empty replaces non-empty)", got.ToolsDeny)
	}
}

// TestResolve_AbsentKindReturnsPreset asserts that calling Resolve with a kind
// for which the registry has no override block returns the Preset values
// verbatim — no per-kind override means pure inheritance, same shape as the
// "no per-kind blocks anywhere" case but probed via the per-kind absent-key
// path rather than the empty-Overrides-map path.
func TestResolve_AbsentKindReturnsPreset(t *testing.T) {
	t.Parallel()

	registry := &AgentsRegistry{
		Preset: Preset{
			Model:        "sonnet",
			MaxBudgetUSD: 5.0,
			ToolsAllow:   []string{"Read", "Bash"},
		},
		Overrides: map[domain.Kind]Override{
			// KindPlan has an override, but the test queries KindBuild.
			domain.KindPlan: {
				Model: ptrStr("opus"),
			},
		},
	}

	got, err := Resolve(registry, domain.KindBuild)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if got.Model != "sonnet" {
		t.Errorf("Model = %q, want %q (inherit when kind absent)", got.Model, "sonnet")
	}
	if got.MaxBudgetUSD != 5.0 {
		t.Errorf("MaxBudgetUSD = %v, want 5.0 (inherit when kind absent)", got.MaxBudgetUSD)
	}
	if !equalStrings(got.ToolsAllow, []string{"Read", "Bash"}) {
		t.Errorf("ToolsAllow = %v, want [Read Bash] (inherit when kind absent)", got.ToolsAllow)
	}
}

// TestMergeLocal_OverrideModel loads `local_override_model.toml` over a project
// registry whose [agents.build] block sets several fields including model. The
// resulting registry must reflect local's model in the merged [agents.build]
// block while project's other build-block fields survive — block-level
// field-merge, not block-level replace.
func TestMergeLocal_OverrideModel(t *testing.T) {
	t.Parallel()

	project := &AgentsRegistry{
		Preset: Preset{
			Client: "claude",
			Model:  "sonnet",
		},
		Overrides: map[domain.Kind]Override{
			domain.KindBuild: {
				Model:        ptrStr("sonnet"),
				MaxBudgetUSD: ptrFloat(5.0),
				ToolsAllow:   ptrSlice([]string{"Read", "Edit", "Bash"}),
			},
		},
	}

	local, err := LoadRegistry(filepath.Join("testdata", "agents", "local_override_model.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry(local) returned error: %v", err)
	}

	merged, err := MergeLocal(project, local)
	if err != nil {
		t.Fatalf("MergeLocal returned error: %v", err)
	}

	build, ok := merged.Overrides[domain.KindBuild]
	if !ok {
		t.Fatal("merged Overrides[build] missing")
	}
	if build.Model == nil || *build.Model != "opus" {
		t.Errorf("merged Overrides[build].Model = %v, want %q (local override)", build.Model, "opus")
	}
	if build.MaxBudgetUSD == nil || *build.MaxBudgetUSD != 5.0 {
		t.Errorf("merged Overrides[build].MaxBudgetUSD = %v, want 5.0 (project survives)", build.MaxBudgetUSD)
	}
	if build.ToolsAllow == nil || !equalStrings(*build.ToolsAllow, []string{"Read", "Edit", "Bash"}) {
		t.Errorf("merged Overrides[build].ToolsAllow = %v, want [Read Edit Bash] (project survives)", build.ToolsAllow)
	}
}

// TestMergeLocal_ToolsDenyRejected asserts that `tools_deny` set anywhere in
// the local registry returns the bare ErrToolsDenyNotOverridable sentinel.
// D5 wraps this sentinel into a position-aware *ConfigError; D3 owns only the
// sentinel-rejection contract.
func TestMergeLocal_ToolsDenyRejected(t *testing.T) {
	t.Parallel()

	project := &AgentsRegistry{
		Preset: Preset{
			ToolsDeny: []string{"WebFetch"},
		},
		Overrides: map[domain.Kind]Override{},
	}

	local, err := LoadRegistry(filepath.Join("testdata", "agents", "local_tools_deny_rejected.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry(local) returned error: %v", err)
	}

	_, err = MergeLocal(project, local)
	if err == nil {
		t.Fatal("MergeLocal returned nil error for local tools_deny; want sentinel rejection")
	}
	if !errors.Is(err, ErrToolsDenyNotOverridable) {
		t.Errorf("error chain does not contain ErrToolsDenyNotOverridable: %v", err)
	}
}

// TestMergeLocal_ToolsDenyDefaultsBlockRejected asserts that `tools_deny` set
// in the local [agents] defaults block (not just per-kind) is also rejected.
// SKETCH § 4.3.1: "MUST NOT override," not "MUST NOT override per-kind."
func TestMergeLocal_ToolsDenyDefaultsBlockRejected(t *testing.T) {
	t.Parallel()

	project := &AgentsRegistry{
		Preset:    Preset{ToolsDeny: []string{"WebFetch"}},
		Overrides: map[domain.Kind]Override{},
	}

	// Construct local in code: tools_deny in the [agents] defaults block.
	local := &AgentsRegistry{
		Preset:    Preset{ToolsDeny: []string{"AnotherTool"}},
		Overrides: map[domain.Kind]Override{},
	}

	_, err := MergeLocal(project, local)
	if err == nil {
		t.Fatal("MergeLocal returned nil error for local defaults-block tools_deny; want sentinel rejection")
	}
	if !errors.Is(err, ErrToolsDenyNotOverridable) {
		t.Errorf("error chain does not contain ErrToolsDenyNotOverridable: %v", err)
	}
}

// TestMergeLocal_NilLocal asserts that calling MergeLocal with a nil local
// registry returns the project registry equivalent — no local file is valid
// (the .local.toml is optional per SKETCH § 4.3).
func TestMergeLocal_NilLocal(t *testing.T) {
	t.Parallel()

	project := &AgentsRegistry{
		Preset: Preset{
			Client:       "claude",
			Model:        "sonnet",
			MaxBudgetUSD: 5.0,
			ToolsAllow:   []string{"Read", "Bash"},
		},
		Overrides: map[domain.Kind]Override{
			domain.KindBuild: {Model: ptrStr("sonnet")},
		},
	}

	merged, err := MergeLocal(project, nil)
	if err != nil {
		t.Fatalf("MergeLocal(project, nil) returned error: %v", err)
	}
	if merged == nil {
		t.Fatal("MergeLocal returned nil registry for nil local")
	}
	if merged.Preset.Client != project.Preset.Client {
		t.Errorf("merged.Preset.Client = %q, want %q", merged.Preset.Client, project.Preset.Client)
	}
	if merged.Preset.Model != project.Preset.Model {
		t.Errorf("merged.Preset.Model = %q, want %q", merged.Preset.Model, project.Preset.Model)
	}
	if merged.Preset.MaxBudgetUSD != project.Preset.MaxBudgetUSD {
		t.Errorf("merged.Preset.MaxBudgetUSD = %v, want %v", merged.Preset.MaxBudgetUSD, project.Preset.MaxBudgetUSD)
	}
	if !equalStrings(merged.Preset.ToolsAllow, project.Preset.ToolsAllow) {
		t.Errorf("merged.Preset.ToolsAllow = %v, want %v", merged.Preset.ToolsAllow, project.Preset.ToolsAllow)
	}
	if _, ok := merged.Overrides[domain.KindBuild]; !ok {
		t.Errorf("merged.Overrides[build] missing; want preserved from project")
	}
}

// TestMergeLocal_NilProject asserts that calling MergeLocal with a nil project
// registry returns an error — agents.toml is required per SKETCH § 3.3 and a
// local file without a project is invalid.
func TestMergeLocal_NilProject(t *testing.T) {
	t.Parallel()

	local := &AgentsRegistry{Preset: Preset{Model: "sonnet"}, Overrides: map[domain.Kind]Override{}}

	_, err := MergeLocal(nil, local)
	if err == nil {
		t.Fatal("MergeLocal(nil, local) returned nil error; want error for missing project")
	}
}

// TestMergeLocal_PartialBlock asserts that a local registry whose [agents.build]
// block sets only one field (Model) merges field-by-field over project's
// [agents.build] — project's other field overrides survive untouched.
func TestMergeLocal_PartialBlock(t *testing.T) {
	t.Parallel()

	project := &AgentsRegistry{
		Preset: Preset{Client: "claude", Model: "sonnet"},
		Overrides: map[domain.Kind]Override{
			domain.KindBuild: {
				Model:        ptrStr("sonnet"),
				MaxBudgetUSD: ptrFloat(5.0),
				MaxTurns:     ptrInt(40),
				ToolsAllow:   ptrSlice([]string{"Read", "Edit", "Bash"}),
			},
		},
	}

	local, err := LoadRegistry(filepath.Join("testdata", "agents", "local_partial_block.toml"))
	if err != nil {
		t.Fatalf("LoadRegistry(local) returned error: %v", err)
	}

	merged, err := MergeLocal(project, local)
	if err != nil {
		t.Fatalf("MergeLocal returned error: %v", err)
	}

	build, ok := merged.Overrides[domain.KindBuild]
	if !ok {
		t.Fatal("merged Overrides[build] missing")
	}
	if build.Model == nil || *build.Model != "haiku" {
		t.Errorf("merged Overrides[build].Model = %v, want %q (local override)", build.Model, "haiku")
	}
	if build.MaxBudgetUSD == nil || *build.MaxBudgetUSD != 5.0 {
		t.Errorf("merged Overrides[build].MaxBudgetUSD = %v, want 5.0 (project survives)", build.MaxBudgetUSD)
	}
	if build.MaxTurns == nil || *build.MaxTurns != 40 {
		t.Errorf("merged Overrides[build].MaxTurns = %v, want 40 (project survives)", build.MaxTurns)
	}
	if build.ToolsAllow == nil || !equalStrings(*build.ToolsAllow, []string{"Read", "Edit", "Bash"}) {
		t.Errorf("merged Overrides[build].ToolsAllow = %v, want [Read Edit Bash] (project survives)", build.ToolsAllow)
	}
}

// TestMergeLocal_PresetFieldMerge asserts that local's [agents] defaults block
// fields override project's [agents] block field-by-field — local's set fields
// win, project's other fields survive. Top-level Preset is concrete (not
// pointer-shaped), so "absent" in local can only be inferred from the zero
// value; the merge therefore uses zero-value-as-absent semantics for Preset
// scalars and zero-length-as-absent for Preset list/map fields. (Pointer-based
// Override is layered on top via per-kind blocks for explicit-zero discrimination.)
func TestMergeLocal_PresetFieldMerge(t *testing.T) {
	t.Parallel()

	project := &AgentsRegistry{
		Preset: Preset{
			Client:       "claude",
			Model:        "sonnet",
			MaxBudgetUSD: 5.0,
			MaxTurns:     40,
			EnvSet:       map[string]string{"A": "1"},
			CliArgs:      []string{"--strict-mcp-config"},
		},
		Overrides: map[domain.Kind]Override{},
	}

	// Local sets Model and adds an EnvSet entry; everything else absent.
	local := &AgentsRegistry{
		Preset: Preset{
			Model:  "opus",
			EnvSet: map[string]string{"B": "2"},
		},
		Overrides: map[domain.Kind]Override{},
	}

	merged, err := MergeLocal(project, local)
	if err != nil {
		t.Fatalf("MergeLocal returned error: %v", err)
	}

	if merged.Preset.Client != "claude" {
		t.Errorf("merged.Preset.Client = %q, want %q (project survives)", merged.Preset.Client, "claude")
	}
	if merged.Preset.Model != "opus" {
		t.Errorf("merged.Preset.Model = %q, want %q (local wins)", merged.Preset.Model, "opus")
	}
	if merged.Preset.MaxBudgetUSD != 5.0 {
		t.Errorf("merged.Preset.MaxBudgetUSD = %v, want 5.0 (project survives)", merged.Preset.MaxBudgetUSD)
	}
	if merged.Preset.MaxTurns != 40 {
		t.Errorf("merged.Preset.MaxTurns = %d, want 40 (project survives)", merged.Preset.MaxTurns)
	}
	if v, ok := merged.Preset.EnvSet["A"]; !ok || v != "1" {
		t.Errorf("merged.Preset.EnvSet[A] = %q (present=%v), want %q present (project survives)", v, ok, "1")
	}
	if v, ok := merged.Preset.EnvSet["B"]; !ok || v != "2" {
		t.Errorf("merged.Preset.EnvSet[B] = %q (present=%v), want %q present (local merged)", v, ok, "2")
	}
	if !equalStrings(merged.Preset.CliArgs, []string{"--strict-mcp-config"}) {
		t.Errorf("merged.Preset.CliArgs = %v, want [--strict-mcp-config] (project survives, local absent)", merged.Preset.CliArgs)
	}
}

// TestMergeLocal_NewKindBlock asserts that a [agents.<kind>] block present in
// local but absent in project lands as a fresh override key in the merged
// registry.
func TestMergeLocal_NewKindBlock(t *testing.T) {
	t.Parallel()

	project := &AgentsRegistry{
		Preset:    Preset{Client: "claude", Model: "sonnet"},
		Overrides: map[domain.Kind]Override{},
	}

	local := &AgentsRegistry{
		Preset: Preset{},
		Overrides: map[domain.Kind]Override{
			domain.KindCommit: {Model: ptrStr("haiku")},
		},
	}

	merged, err := MergeLocal(project, local)
	if err != nil {
		t.Fatalf("MergeLocal returned error: %v", err)
	}
	commit, ok := merged.Overrides[domain.KindCommit]
	if !ok {
		t.Fatal("merged Overrides[commit] missing; want fresh kind from local")
	}
	if commit.Model == nil || *commit.Model != "haiku" {
		t.Errorf("merged Overrides[commit].Model = %v, want %q (local-only kind)", commit.Model, "haiku")
	}
}

// equalStrings compares two string slices element-by-element.
func equalStrings(a, b []string) bool {
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

// ptrStr returns a pointer to s. Test helper for constructing Override
// scalars in code rather than via TOML decode.
func ptrStr(s string) *string { return &s }

// ptrSlice returns a pointer to s. Test helper for constructing Override
// list fields in code rather than via TOML decode — load-bearing for the
// empty-list-vs-nil edge case where TOML cannot express "explicit empty
// list" disjoint from "absent."
func ptrSlice(s []string) *[]string { return &s }

// ptrMap returns a pointer to m. Test helper for constructing Override map
// fields in code rather than via TOML decode.
func ptrMap(m map[string]string) *map[string]string { return &m }

// ptrFloat returns a pointer to f. Test helper for constructing Override
// numeric fields in code.
func ptrFloat(f float64) *float64 { return &f }

// ptrInt returns a pointer to n. Test helper for constructing Override
// integer fields in code.
func ptrInt(n int) *int { return &n }
