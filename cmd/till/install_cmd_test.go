package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/platform"
)

// TestRunInstall_CreatesDebugConfig verifies `till install` creates the dev
// config and enforces debug logging. Ported verbatim from the
// TestRunInitDevConfigCreatesDebugConfig body in main_test.go (D7.5 lifts
// the behavior into a new `install` cobra command; D8 later removes the
// `init-dev-config` original). The new test name introduces an underscore
// between `TestRunInstall` and the rest — TEST-NAME CONTRACT (W2-FF2 +
// W2-FF9 ROUND-2). Tests invoke `run(...)` end-to-end (NOT runInstall
// directly) — CONSUMER-TIE TEST CONTRACT (W2-FF3 ROUND-2).
func TestRunInstall_CreatesDebugConfig(t *testing.T) {
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
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard); err != nil {
		t.Fatalf("run(install) error = %v", err)
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{AppName: "tillsyn-init", DevMode: true})
	if err != nil {
		t.Fatalf("DefaultPathsWithOptions() error = %v", err)
	}
	for _, want := range []string{"Dev Config", "status", "created dev config", shellEscapePath(paths.ConfigPath), "logging level", "debug"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected %q in install output, got %q", want, out.String())
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

// TestRunInstall_UpdatesExistingConfig verifies `till install` rewrites an
// existing logging section to debug. Ported verbatim from the
// TestRunInitDevConfigUpdatesExistingConfig body in main_test.go. Name
// underscore + end-to-end invocation per contracts noted above.
func TestRunInstall_UpdatesExistingConfig(t *testing.T) {
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
	if err := run(context.Background(), []string{"--app", "tillsyn-init", "install"}, &out, io.Discard); err != nil {
		t.Fatalf("run(install existing) error = %v", err)
	}
	for _, want := range []string{"Dev Config", "status", "dev config already exists", shellEscapePath(paths.ConfigPath), "logging level", "debug"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected %q in install existing output, got %q", want, out.String())
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
