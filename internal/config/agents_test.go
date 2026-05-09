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
