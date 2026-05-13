// Package config — tests for the multi-group agents.toml schema + decode
// wiring shipped in agents.go (Drop 4c.6.1 W0). Co-located with the
// production file per CLAUDE.md § "Tests" discipline.
package config

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// ── LoadMultiGroupRegistry ────────────────────────────────────────────────────

// TestLoadMultiGroupRegistry_SingleGroup loads the canonical single-group
// golden fixture and asserts every Default field decoded correctly plus the
// [go.build] override surfaces with the expected tools_allow override.
func TestLoadMultiGroupRegistry_SingleGroup(t *testing.T) {
	t.Parallel()

	registry, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "multigroup_single.toml"))
	if err != nil {
		t.Fatalf("LoadMultiGroupRegistry returned error: %v", err)
	}
	if registry == nil {
		t.Fatal("LoadMultiGroupRegistry returned nil registry without error")
	}

	gc, ok := registry["go"]
	if !ok {
		t.Fatal("registry missing [go] group")
	}

	d := gc.Default
	if d.Client != "claude" {
		t.Errorf("go Default.Client = %q, want %q", d.Client, "claude")
	}
	if d.Model != "sonnet" {
		t.Errorf("go Default.Model = %q, want %q", d.Model, "sonnet")
	}
	if d.Effort != "medium" {
		t.Errorf("go Default.Effort = %q, want %q", d.Effort, "medium")
	}
	if d.MaxTries != 3 {
		t.Errorf("go Default.MaxTries = %d, want 3", d.MaxTries)
	}
	if d.MaxBudgetUSD != 5.0 {
		t.Errorf("go Default.MaxBudgetUSD = %v, want 5.0", d.MaxBudgetUSD)
	}
	if d.MaxTurns != 40 {
		t.Errorf("go Default.MaxTurns = %d, want 40", d.MaxTurns)
	}
	if d.BlockedRetries != 2 {
		t.Errorf("go Default.BlockedRetries = %d, want 2", d.BlockedRetries)
	}
	if d.BlockedRetryCooldown != "30s" {
		t.Errorf("go Default.BlockedRetryCooldown = %q, want %q", d.BlockedRetryCooldown, "30s")
	}
	if d.AutoPush != false {
		t.Errorf("go Default.AutoPush = %v, want false", d.AutoPush)
	}
	if got := d.EnvSet["TILLSYN_DEV"]; got != "1" {
		t.Errorf("go Default.EnvSet[TILLSYN_DEV] = %q, want %q", got, "1")
	}
	if got := d.EnvFromShell["GH_TOKEN"]; got != "GH_TOKEN" {
		t.Errorf("go Default.EnvFromShell[GH_TOKEN] = %q, want %q", got, "GH_TOKEN")
	}
	if !equalStrings(d.CliArgs, []string{"--strict-mcp-config"}) {
		t.Errorf("go Default.CliArgs = %v, want [--strict-mcp-config]", d.CliArgs)
	}
	if !equalStrings(d.ToolsAllow, []string{"Read", "Edit", "Bash"}) {
		t.Errorf("go Default.ToolsAllow = %v, want [Read Edit Bash]", d.ToolsAllow)
	}
	if !equalStrings(d.ToolsDeny, []string{"WebFetch"}) {
		t.Errorf("go Default.ToolsDeny = %v, want [WebFetch]", d.ToolsDeny)
	}
	if !equalStrings(d.ClaudeMDAddons, []string{"~/.claude/output-styles/tillsyn-flow.md"}) {
		t.Errorf("go Default.ClaudeMDAddons = %v, want [tillsyn-flow.md]", d.ClaudeMDAddons)
	}

	buildOv, ok := gc.Kinds[domain.KindBuild]
	if !ok {
		t.Fatalf("go Kinds[build] missing")
	}
	if buildOv.ToolsAllow == nil {
		t.Fatal("go Kinds[build].ToolsAllow is nil; want non-nil pointer")
	}
	wantBuildAllow := []string{"Read", "Edit", "Write", "Bash"}
	if !equalStrings(*buildOv.ToolsAllow, wantBuildAllow) {
		t.Errorf("go Kinds[build].ToolsAllow = %v, want %v", *buildOv.ToolsAllow, wantBuildAllow)
	}
	if buildOv.Model == nil {
		// model was set to "sonnet" explicitly in the fixture.
		t.Error("go Kinds[build].Model = nil; want non-nil (set to sonnet in fixture)")
	}
}

// TestLoadMultiGroupRegistry_MultiGroup loads the multi-group go+fe fixture
// and asserts both groups decoded independently with correct field isolation.
func TestLoadMultiGroupRegistry_MultiGroup(t *testing.T) {
	t.Parallel()

	registry, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "multigroup_go_fe.toml"))
	if err != nil {
		t.Fatalf("LoadMultiGroupRegistry returned error: %v", err)
	}

	if _, ok := registry["go"]; !ok {
		t.Fatal("registry missing [go] group")
	}
	if _, ok := registry["fe"]; !ok {
		t.Fatal("registry missing [fe] group")
	}
	if len(registry) != 2 {
		t.Errorf("len(registry) = %d, want 2", len(registry))
	}

	goGC := registry["go"]
	if goGC.Default.Model != "sonnet" {
		t.Errorf("go Default.Model = %q, want %q", goGC.Default.Model, "sonnet")
	}
	if _, ok := goGC.Kinds[domain.KindBuild]; !ok {
		t.Error("go Kinds[build] missing")
	}
	if _, ok := goGC.Kinds[domain.KindBuildQAProof]; ok {
		t.Error("go Kinds[build-qa-proof] unexpectedly present")
	}

	feGC := registry["fe"]
	if feGC.Default.Model != "sonnet" {
		t.Errorf("fe Default.Model = %q, want %q", feGC.Default.Model, "sonnet")
	}
	if _, ok := feGC.Kinds[domain.KindBuildQAProof]; !ok {
		t.Error("fe Kinds[build-qa-proof] missing")
	}
	feQAProof, _ := feGC.Kinds[domain.KindBuildQAProof]
	if feQAProof.Model == nil || *feQAProof.Model != "opus" {
		t.Errorf("fe Kinds[build-qa-proof].Model = %v, want %q", feQAProof.Model, "opus")
	}
}

// TestLoadMultiGroupRegistry_MalformedTOML feeds a fixture with a syntax error
// and asserts the returned error wraps a *toml.DecodeError.
func TestLoadMultiGroupRegistry_MalformedTOML(t *testing.T) {
	t.Parallel()

	_, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "malformed.toml"))
	if err == nil {
		t.Fatal("LoadMultiGroupRegistry returned nil error for malformed input")
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

// TestLoadMultiGroupRegistry_UnknownKindField asserts the strict decoder rejects
// unknown kind-level fields within a group block. Catches typos early.
func TestLoadMultiGroupRegistry_UnknownKindField(t *testing.T) {
	t.Parallel()

	_, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "unknown_field.toml"))
	if err == nil {
		t.Fatal("LoadMultiGroupRegistry returned nil error for fixture with unknown field")
	}
	// The error should mention something about unknown / unexpected fields.
	if !strings.Contains(strings.ToLower(err.Error()), "unknown") &&
		!strings.Contains(strings.ToLower(err.Error()), "missing") {
		t.Errorf("error message %q does not mention unknown/missing field", err.Error())
	}
}

// TestLoadMultiGroupRegistry_FileNotFound asserts that a missing file produces
// a clear error.
func TestLoadMultiGroupRegistry_FileNotFound(t *testing.T) {
	t.Parallel()

	registry, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "does_not_exist.toml"))
	if err == nil {
		t.Fatal("LoadMultiGroupRegistry returned nil error for nonexistent path")
	}
	if registry != nil {
		t.Errorf("LoadMultiGroupRegistry returned non-nil registry alongside error: %+v", registry)
	}
}

// TestLoadMultiGroupRegistry_PositionWrapped asserts the *ConfigError envelope
// wraps a malformed-TOML error with File set to the loaded path and Line > 0.
func TestLoadMultiGroupRegistry_PositionWrapped(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "agents", "malformed.toml")
	_, err := LoadMultiGroupRegistry(path)
	if err == nil {
		t.Fatal("LoadMultiGroupRegistry returned nil error for malformed input")
	}

	var cerr *ConfigError
	if !errors.As(err, &cerr) {
		t.Fatalf("error chain does not contain *ConfigError: %v", err)
	}
	if cerr.File != path {
		t.Errorf("ConfigError.File = %q, want %q", cerr.File, path)
	}
	if cerr.Line <= 0 {
		t.Errorf("ConfigError.Line = %d, want > 0", cerr.Line)
	}

	var decodeErr *toml.DecodeError
	if !errors.As(err, &decodeErr) {
		t.Errorf("errors.As(err, &*toml.DecodeError) = false; want true (unwrap chain preservation)")
	}
}

// ── Resolve ────────────────────────────────────────────────────────────────────

// TestResolve_FullInherit asserts Resolve returns Default values verbatim
// when the group exists but has no per-kind block for the requested kind.
func TestResolve_FullInherit(t *testing.T) {
	t.Parallel()

	registry, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "multigroup_single.toml"))
	if err != nil {
		t.Fatalf("LoadMultiGroupRegistry returned error: %v", err)
	}

	// Request KindPlan — no [go.plan] in the fixture; should inherit Default.
	got, err := Resolve(registry, "go", string(domain.KindPlan))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if got.Client != "claude" {
		t.Errorf("Client = %q, want %q", got.Client, "claude")
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

// TestResolve_KindOverride asserts that a per-kind override wins over the
// group Default for the overridden fields, while non-overridden fields
// inherit from Default. Covers acceptance criterion (c).
func TestResolve_KindOverride(t *testing.T) {
	t.Parallel()

	registry, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "multigroup_kind_override.toml"))
	if err != nil {
		t.Fatalf("LoadMultiGroupRegistry returned error: %v", err)
	}

	got, err := Resolve(registry, "go", string(domain.KindPlanQAProof))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if got.Model != "opus" {
		t.Errorf("Model = %q, want %q (per-kind override wins)", got.Model, "opus")
	}
	if got.MaxBudgetUSD != 10.0 {
		t.Errorf("MaxBudgetUSD = %v, want 10.0 (per-kind override wins)", got.MaxBudgetUSD)
	}
	// ToolsAllow not overridden in [go.plan-qa-proof]; must inherit Default.
	if !equalStrings(got.ToolsAllow, []string{"Read", "Bash"}) {
		t.Errorf("ToolsAllow = %v, want [Read Bash] (inherited)", got.ToolsAllow)
	}
}

// TestResolve_MissingGroupFallback asserts that Resolve returns an empty Preset
// (no panic) when the requested group does not exist in the registry.
// Covers acceptance criterion (e).
func TestResolve_MissingGroupFallback(t *testing.T) {
	t.Parallel()

	registry, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "multigroup_single.toml"))
	if err != nil {
		t.Fatalf("LoadMultiGroupRegistry returned error: %v", err)
	}

	got, err := Resolve(registry, "rust", string(domain.KindBuild))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got.Model != "" {
		t.Errorf("Model = %q, want empty string (missing group returns empty Preset)", got.Model)
	}
	if got.MaxTries != 0 {
		t.Errorf("MaxTries = %d, want 0 (missing group returns empty Preset)", got.MaxTries)
	}
}

// TestResolve_MapMerge asserts EnvSet and EnvFromShell merge per-key — the
// per-kind block's keys add to the Default's keys; neither side's keys are
// dropped.
func TestResolve_MapMerge(t *testing.T) {
	t.Parallel()

	// Construct registry in code for precise control.
	registry := AgentsRegistry{
		"go": {
			Default: Preset{
				EnvSet:       map[string]string{"A": "1"},
				EnvFromShell: map[string]string{"SHELL_A": "SHELL_A"},
			},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {
					EnvSet:       ptrMap(map[string]string{"B": "2"}),
					EnvFromShell: ptrMap(map[string]string{"SHELL_B": "SHELL_B"}),
				},
			},
		},
	}

	got, err := Resolve(registry, "go", string(domain.KindBuild))
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
		t.Errorf("EnvFromShell[SHELL_A] = %q (present=%v), want present", v, ok)
	}
	if v, ok := got.EnvFromShell["SHELL_B"]; !ok || v != "SHELL_B" {
		t.Errorf("EnvFromShell[SHELL_B] = %q (present=%v), want present", v, ok)
	}
}

// TestResolve_MapOverrideWins asserts that when the per-kind block sets a key
// already present in the Default map, the override value wins.
func TestResolve_MapOverrideWins(t *testing.T) {
	t.Parallel()

	registry := AgentsRegistry{
		"go": {
			Default: Preset{EnvSet: map[string]string{"K": "preset"}},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {EnvSet: ptrMap(map[string]string{"K": "override"})},
			},
		},
	}

	got, err := Resolve(registry, "go", string(domain.KindBuild))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got.EnvSet["K"] != "override" {
		t.Errorf("EnvSet[K] = %q, want %q (override wins on collision)", got.EnvSet["K"], "override")
	}
}

// TestResolve_ListReplace asserts list fields full-replace when the per-kind
// block sets them — Default's list is dropped wholesale.
func TestResolve_ListReplace(t *testing.T) {
	t.Parallel()

	registry := AgentsRegistry{
		"go": {
			Default: Preset{
				ToolsAllow: []string{"Read", "Edit", "Bash"},
				CliArgs:    []string{"--strict-mcp-config"},
			},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {
					ToolsAllow: ptrSlice([]string{"Read"}),
					CliArgs:    ptrSlice([]string{"--quiet"}),
				},
			},
		},
	}

	got, err := Resolve(registry, "go", string(domain.KindBuild))
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
// empty list explicitly replaces a non-empty Default list with an empty slice.
func TestResolve_ExplicitEmptyList(t *testing.T) {
	t.Parallel()

	registry := AgentsRegistry{
		"go": {
			Default: Preset{ToolsDeny: []string{"rm", "WebFetch"}},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {ToolsDeny: ptrSlice([]string{})},
			},
		},
	}

	got, err := Resolve(registry, "go", string(domain.KindBuild))
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

// TestResolve_AbsentKindReturnsDefault asserts that calling Resolve with a kind
// for which the group has no override block returns the Default values verbatim.
func TestResolve_AbsentKindReturnsDefault(t *testing.T) {
	t.Parallel()

	registry := AgentsRegistry{
		"go": {
			Default: Preset{
				Model:        "sonnet",
				MaxBudgetUSD: 5.0,
				ToolsAllow:   []string{"Read", "Bash"},
			},
			Kinds: map[domain.Kind]Override{
				domain.KindPlan: {Model: ptrStr("opus")},
			},
		},
	}

	got, err := Resolve(registry, "go", string(domain.KindBuild))
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

// TestResolve_NilKindsMap asserts that a GroupConfig with a nil Kinds map
// returns the Default values verbatim without panicking.
func TestResolve_NilKindsMap(t *testing.T) {
	t.Parallel()

	registry := AgentsRegistry{
		"go": {
			Default: Preset{Model: "sonnet"},
			Kinds:   nil, // deliberately nil
		},
	}

	got, err := Resolve(registry, "go", string(domain.KindBuild))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got.Model != "sonnet" {
		t.Errorf("Model = %q, want %q (Default with nil Kinds)", got.Model, "sonnet")
	}
}

// ── Merge ──────────────────────────────────────────────────────────────────────

// TestMerge_LocalWinsOverProject asserts that local's per-kind model override
// wins over project's per-kind model, while project's other fields survive.
// Covers acceptance criterion (d).
func TestMerge_LocalWinsOverProject(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {
			Default: Preset{Client: "claude", Model: "sonnet"},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {
					Model:        ptrStr("sonnet"),
					MaxBudgetUSD: ptrFloat(5.0),
					ToolsAllow:   ptrSlice([]string{"Read", "Edit", "Bash"}),
				},
			},
		},
	}

	local, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "multigroup_local_override.toml"))
	if err != nil {
		t.Fatalf("LoadMultiGroupRegistry(local) returned error: %v", err)
	}

	merged, err := Merge(local, project)
	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	gc, ok := merged["go"]
	if !ok {
		t.Fatal("merged registry missing [go] group")
	}
	buildOv, ok := gc.Kinds[domain.KindBuild]
	if !ok {
		t.Fatal("merged go Kinds[build] missing")
	}
	if buildOv.Model == nil || *buildOv.Model != "opus" {
		t.Errorf("merged go Kinds[build].Model = %v, want %q (local wins)", buildOv.Model, "opus")
	}
	if buildOv.MaxBudgetUSD == nil || *buildOv.MaxBudgetUSD != 5.0 {
		t.Errorf("merged go Kinds[build].MaxBudgetUSD = %v, want 5.0 (project survives)", buildOv.MaxBudgetUSD)
	}
	if buildOv.ToolsAllow == nil || !equalStrings(*buildOv.ToolsAllow, []string{"Read", "Edit", "Bash"}) {
		t.Errorf("merged go Kinds[build].ToolsAllow = %v, want [Read Edit Bash] (project survives)", buildOv.ToolsAllow)
	}
}

// TestMerge_MultiGroupBothPreserved asserts that when local has a [go] group
// and project has both [go] and [fe], the merged registry contains all three
// groups correctly (local [go] merged; project [fe] cloned).
func TestMerge_MultiGroupBothPreserved(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {
			Default: Preset{Model: "sonnet"},
			Kinds:   map[domain.Kind]Override{},
		},
		"fe": {
			Default: Preset{Model: "sonnet"},
			Kinds: map[domain.Kind]Override{
				domain.KindBuildQAProof: {Model: ptrStr("opus")},
			},
		},
	}

	local := AgentsRegistry{
		"go": {
			Default: Preset{Model: "haiku"},
			Kinds:   map[domain.Kind]Override{},
		},
	}

	merged, err := Merge(local, project)
	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	if len(merged) != 2 {
		t.Errorf("len(merged) = %d, want 2 (go + fe)", len(merged))
	}
	if goGC, ok := merged["go"]; !ok {
		t.Error("merged missing [go]")
	} else if goGC.Default.Model != "haiku" {
		t.Errorf("merged[go].Default.Model = %q, want %q (local wins)", goGC.Default.Model, "haiku")
	}
	if feGC, ok := merged["fe"]; !ok {
		t.Error("merged missing [fe]")
	} else {
		feOv, ok := feGC.Kinds[domain.KindBuildQAProof]
		if !ok {
			t.Error("merged[fe] Kinds[build-qa-proof] missing")
		} else if feOv.Model == nil || *feOv.Model != "opus" {
			t.Errorf("merged[fe] Kinds[build-qa-proof].Model = %v, want %q (project preserved)", feOv.Model, "opus")
		}
	}
}

// TestMerge_ToolsDenyRejected asserts that tools_deny set anywhere in the local
// registry returns ErrToolsDenyNotOverridable. Covers SKETCH § 4.3.1.
func TestMerge_ToolsDenyRejected(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {
			Default: Preset{ToolsDeny: []string{"WebFetch"}},
			Kinds:   map[domain.Kind]Override{},
		},
	}

	local, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "multigroup_tools_deny_rejected.toml"))
	if err != nil {
		t.Fatalf("LoadMultiGroupRegistry(local) returned error: %v", err)
	}

	_, err = Merge(local, project)
	if err == nil {
		t.Fatal("Merge returned nil error for local tools_deny; want sentinel rejection")
	}
	if !errors.Is(err, ErrToolsDenyNotOverridable) {
		t.Errorf("error chain does not contain ErrToolsDenyNotOverridable: %v", err)
	}
}

// TestMerge_ToolsDenyDefaultBlockRejected asserts that tools_deny set in a
// group Default block (not just per-kind) is also rejected.
func TestMerge_ToolsDenyDefaultBlockRejected(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {Default: Preset{ToolsDeny: []string{"WebFetch"}}, Kinds: map[domain.Kind]Override{}},
	}
	local := AgentsRegistry{
		"go": {
			Default: Preset{ToolsDeny: []string{"AnotherTool"}},
			Kinds:   map[domain.Kind]Override{},
		},
	}

	_, err := Merge(local, project)
	if err == nil {
		t.Fatal("Merge returned nil error for local Default tools_deny; want sentinel rejection")
	}
	if !errors.Is(err, ErrToolsDenyNotOverridable) {
		t.Errorf("error chain does not contain ErrToolsDenyNotOverridable: %v", err)
	}

	var cerr *ConfigError
	if !errors.As(err, &cerr) {
		t.Fatalf("error chain does not contain *ConfigError: %v", err)
	}
	// Block should name the [go] group default.
	if cerr.Block != "[go]" {
		t.Errorf("ConfigError.Block = %q, want %q", cerr.Block, "[go]")
	}
}

// TestMerge_ToolsDenyPerKindBlockPositionWrapped asserts that a per-kind
// tools_deny rejection wraps with the correct block context "[go.build]".
func TestMerge_ToolsDenyPerKindBlockPositionWrapped(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {Default: Preset{}, Kinds: map[domain.Kind]Override{}},
	}
	local, err := LoadMultiGroupRegistry(filepath.Join("testdata", "agents", "multigroup_tools_deny_rejected.toml"))
	if err != nil {
		t.Fatalf("LoadMultiGroupRegistry(local) returned error: %v", err)
	}

	_, err = Merge(local, project)
	if err == nil {
		t.Fatal("Merge returned nil error for local per-kind tools_deny; want envelope-wrapped sentinel")
	}

	if !errors.Is(err, ErrToolsDenyNotOverridable) {
		t.Errorf("errors.Is(err, ErrToolsDenyNotOverridable) = false; want true")
	}

	var cerr *ConfigError
	if !errors.As(err, &cerr) {
		t.Fatalf("error chain does not contain *ConfigError: %v", err)
	}
	if cerr.File != "agents.local.toml" {
		t.Errorf("ConfigError.File = %q, want %q", cerr.File, "agents.local.toml")
	}
	if cerr.Block != "[go.build]" {
		t.Errorf("ConfigError.Block = %q, want %q", cerr.Block, "[go.build]")
	}
}

// TestMerge_NilLocal asserts that Merge(nil, project) returns a deep-clone
// of project — local is optional.
func TestMerge_NilLocal(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {
			Default: Preset{Client: "claude", Model: "sonnet"},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {Model: ptrStr("sonnet")},
			},
		},
	}

	merged, err := Merge(nil, project)
	if err != nil {
		t.Fatalf("Merge(nil, project) returned error: %v", err)
	}
	if merged == nil {
		t.Fatal("Merge returned nil registry for nil local")
	}
	gc, ok := merged["go"]
	if !ok {
		t.Fatal("merged missing [go] from project")
	}
	if gc.Default.Client != "claude" {
		t.Errorf("merged[go].Default.Client = %q, want %q", gc.Default.Client, "claude")
	}
	if gc.Default.Model != "sonnet" {
		t.Errorf("merged[go].Default.Model = %q, want %q", gc.Default.Model, "sonnet")
	}
	if _, ok := gc.Kinds[domain.KindBuild]; !ok {
		t.Error("merged[go] Kinds[build] missing; want preserved from project")
	}
}

// TestMerge_NilProject asserts that Merge(local, nil) returns a deep-clone
// of local — project is optional.
func TestMerge_NilProject(t *testing.T) {
	t.Parallel()

	local := AgentsRegistry{
		"go": {Default: Preset{Model: "sonnet"}, Kinds: map[domain.Kind]Override{}},
	}

	merged, err := Merge(local, nil)
	if err != nil {
		t.Fatalf("Merge(local, nil) returned error: %v", err)
	}
	if _, ok := merged["go"]; !ok {
		t.Fatal("merged missing [go] from local")
	}
}

// TestMerge_BothNil asserts that Merge(nil, nil) returns an empty initialized
// registry without error.
func TestMerge_BothNil(t *testing.T) {
	t.Parallel()

	merged, err := Merge(nil, nil)
	if err != nil {
		t.Fatalf("Merge(nil, nil) returned error: %v", err)
	}
	if merged == nil {
		t.Fatal("Merge(nil, nil) returned nil registry; want empty initialized map")
	}
	if len(merged) != 0 {
		t.Errorf("len(merged) = %d, want 0", len(merged))
	}
}

// TestMerge_NewGroupFromLocal asserts that a group present only in local lands
// as a fresh group in the merged registry.
func TestMerge_NewGroupFromLocal(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {Default: Preset{Model: "sonnet"}, Kinds: map[domain.Kind]Override{}},
	}
	local := AgentsRegistry{
		"fe": {
			Default: Preset{Model: "sonnet"},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {Model: ptrStr("haiku")},
			},
		},
	}

	merged, err := Merge(local, project)
	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}
	if len(merged) != 2 {
		t.Errorf("len(merged) = %d, want 2 (go + fe)", len(merged))
	}
	feGC, ok := merged["fe"]
	if !ok {
		t.Fatal("merged missing [fe] from local")
	}
	feBuild, ok := feGC.Kinds[domain.KindBuild]
	if !ok {
		t.Fatal("merged[fe] Kinds[build] missing")
	}
	if feBuild.Model == nil || *feBuild.Model != "haiku" {
		t.Errorf("merged[fe] Kinds[build].Model = %v, want %q", feBuild.Model, "haiku")
	}
}

// TestMerge_DefaultFieldMerge asserts that local's non-zero Default fields
// override project's Default fields, while local's zero-value fields preserve
// project's values.
func TestMerge_DefaultFieldMerge(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {
			Default: Preset{
				Client:       "claude",
				Model:        "sonnet",
				MaxBudgetUSD: 5.0,
				MaxTurns:     40,
				EnvSet:       map[string]string{"A": "1"},
				CliArgs:      []string{"--strict-mcp-config"},
			},
			Kinds: map[domain.Kind]Override{},
		},
	}
	local := AgentsRegistry{
		"go": {
			Default: Preset{
				Model:  "opus",
				EnvSet: map[string]string{"B": "2"},
				// Client, MaxBudgetUSD, MaxTurns, CliArgs absent (zero) → project survives.
			},
			Kinds: map[domain.Kind]Override{},
		},
	}

	merged, err := Merge(local, project)
	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	gc := merged["go"]
	if gc.Default.Client != "claude" {
		t.Errorf("merged Client = %q, want %q (project survives)", gc.Default.Client, "claude")
	}
	if gc.Default.Model != "opus" {
		t.Errorf("merged Model = %q, want %q (local wins)", gc.Default.Model, "opus")
	}
	if gc.Default.MaxBudgetUSD != 5.0 {
		t.Errorf("merged MaxBudgetUSD = %v, want 5.0 (project survives)", gc.Default.MaxBudgetUSD)
	}
	if gc.Default.MaxTurns != 40 {
		t.Errorf("merged MaxTurns = %d, want 40 (project survives)", gc.Default.MaxTurns)
	}
	if v, ok := gc.Default.EnvSet["A"]; !ok || v != "1" {
		t.Errorf("merged EnvSet[A] = %q present=%v, want present (project survives)", v, ok)
	}
	if v, ok := gc.Default.EnvSet["B"]; !ok || v != "2" {
		t.Errorf("merged EnvSet[B] = %q present=%v, want present (local merged)", v, ok)
	}
	if !equalStrings(gc.Default.CliArgs, []string{"--strict-mcp-config"}) {
		t.Errorf("merged CliArgs = %v, want [--strict-mcp-config] (project survives)", gc.Default.CliArgs)
	}
}

// TestMerge_PartialKindBlock asserts that a local group whose [kind] block sets
// only one field merges field-by-field over project's same [kind] block —
// project's other kind-block fields survive.
func TestMerge_PartialKindBlock(t *testing.T) {
	t.Parallel()

	project := AgentsRegistry{
		"go": {
			Default: Preset{Model: "sonnet"},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {
					Model:        ptrStr("sonnet"),
					MaxBudgetUSD: ptrFloat(5.0),
					MaxTurns:     ptrInt(40),
					ToolsAllow:   ptrSlice([]string{"Read", "Edit", "Bash"}),
				},
			},
		},
	}
	local := AgentsRegistry{
		"go": {
			Default: Preset{},
			Kinds: map[domain.Kind]Override{
				domain.KindBuild: {Model: ptrStr("haiku")},
			},
		},
	}

	merged, err := Merge(local, project)
	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	gc := merged["go"]
	buildOv, ok := gc.Kinds[domain.KindBuild]
	if !ok {
		t.Fatal("merged go Kinds[build] missing")
	}
	if buildOv.Model == nil || *buildOv.Model != "haiku" {
		t.Errorf("merged Kinds[build].Model = %v, want %q (local wins)", buildOv.Model, "haiku")
	}
	if buildOv.MaxBudgetUSD == nil || *buildOv.MaxBudgetUSD != 5.0 {
		t.Errorf("merged Kinds[build].MaxBudgetUSD = %v, want 5.0 (project survives)", buildOv.MaxBudgetUSD)
	}
	if buildOv.MaxTurns == nil || *buildOv.MaxTurns != 40 {
		t.Errorf("merged Kinds[build].MaxTurns = %v, want 40 (project survives)", buildOv.MaxTurns)
	}
	if buildOv.ToolsAllow == nil || !equalStrings(*buildOv.ToolsAllow, []string{"Read", "Edit", "Bash"}) {
		t.Errorf("merged Kinds[build].ToolsAllow = %v, want [Read Edit Bash] (project survives)", buildOv.ToolsAllow)
	}
}

// ── ConfigError ───────────────────────────────────────────────────────────────

// TestConfigError_FormatsCorrectly asserts the *ConfigError envelope's Error()
// method produces the canonical "<file> <block>:<line>: <cause>" shape.
func TestConfigError_FormatsCorrectly(t *testing.T) {
	t.Parallel()

	cause := errors.New("tools_deny is not user-overridable; remove the field")
	cerr := &ConfigError{
		File:  "agents.local.toml",
		Block: "[go.build]",
		Line:  42,
		Cause: cause,
	}

	got := cerr.Error()
	want := "agents.local.toml [go.build]:42: tools_deny is not user-overridable; remove the field"
	if got != want {
		t.Errorf("ConfigError.Error() = %q, want %q", got, want)
	}
}

// TestConfigError_FormatsWithoutLine asserts that Line == 0 omits the
// colon-line suffix instead of printing a misleading ":0".
func TestConfigError_FormatsWithoutLine(t *testing.T) {
	t.Parallel()

	cause := errors.New("tools_deny is not user-overridable; remove the field")
	cerr := &ConfigError{
		File:  "agents.local.toml",
		Block: "[go.build]",
		Line:  0,
		Cause: cause,
	}

	got := cerr.Error()
	want := "agents.local.toml [go.build]: tools_deny is not user-overridable; remove the field"
	if got != want {
		t.Errorf("ConfigError.Error() with Line=0 = %q, want %q", got, want)
	}
}

// TestConfigError_FormatsWithoutBlock asserts that Block == "" renders
// "<file>:<line>: <cause>" without an empty bracket.
func TestConfigError_FormatsWithoutBlock(t *testing.T) {
	t.Parallel()

	cause := errors.New("syntax error")
	cerr := &ConfigError{
		File:  "agents.toml",
		Block: "",
		Line:  8,
		Cause: cause,
	}

	got := cerr.Error()
	want := "agents.toml:8: syntax error"
	if got != want {
		t.Errorf("ConfigError.Error() with Block= = %q, want %q", got, want)
	}
}

// TestConfigError_UnwrapPreservesSentinel asserts errors.Is against
// ErrToolsDenyNotOverridable succeeds when wrapped inside a *ConfigError.
func TestConfigError_UnwrapPreservesSentinel(t *testing.T) {
	t.Parallel()

	cerr := &ConfigError{
		File:  "agents.local.toml",
		Block: "[go.build]",
		Cause: ErrToolsDenyNotOverridable,
	}

	if !errors.Is(cerr, ErrToolsDenyNotOverridable) {
		t.Errorf("errors.Is(*ConfigError wrapping ErrToolsDenyNotOverridable, ErrToolsDenyNotOverridable) = false; want true")
	}
	if got := errors.Unwrap(cerr); got != ErrToolsDenyNotOverridable {
		t.Errorf("errors.Unwrap(*ConfigError) = %v, want ErrToolsDenyNotOverridable", got)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

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

// ptrStr returns a pointer to s.
func ptrStr(s string) *string { return &s }

// ptrSlice returns a pointer to s.
func ptrSlice(s []string) *[]string { return &s }

// ptrMap returns a pointer to m.
func ptrMap(m map[string]string) *map[string]string { return &m }

// ptrFloat returns a pointer to f.
func ptrFloat(f float64) *float64 { return &f }

// ptrInt returns a pointer to n.
func ptrInt(n int) *int { return &n }
