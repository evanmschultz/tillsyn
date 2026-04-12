package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestPathsForLinuxWithXDG verifies behavior for the covered scenario.
func TestPathsForLinuxWithXDG(t *testing.T) {
	p, err := PathsFor("linux", map[string]string{
		"XDG_CONFIG_HOME": "/xdg/config",
		"XDG_DATA_HOME":   "/xdg/data",
	}, "/fallback/config", "/fallback/data", "tillsyn")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	wantConfig := filepath.Join("/xdg/config", "tillsyn", "config.toml")
	wantDB := filepath.Join("/xdg/data", "tillsyn", "tillsyn.db")
	wantLogs := filepath.Join("/xdg/data", "tillsyn", "logs")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
	if p.LogsDir != wantLogs {
		t.Fatalf("unexpected logs dir %q", p.LogsDir)
	}
}

// TestPathsForWindowsUsesAppData verifies behavior for the covered scenario.
func TestPathsForWindowsUsesAppData(t *testing.T) {
	p, err := PathsFor("windows", map[string]string{
		"APPDATA":      `C:\Users\me\AppData\Roaming`,
		"LOCALAPPDATA": `C:\Users\me\AppData\Local`,
	}, `C:\fallback\config`, `C:\fallback\data`, "tillsyn")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}

	wantConfig := filepath.Join(`C:\Users\me\AppData\Roaming`, "tillsyn", "config.toml")
	wantDB := filepath.Join(`C:\Users\me\AppData\Local`, "tillsyn", "tillsyn.db")
	wantLogs := filepath.Join(`C:\Users\me\AppData\Local`, "tillsyn", "logs")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
	if p.LogsDir != wantLogs {
		t.Fatalf("unexpected logs dir %q", p.LogsDir)
	}
}

// TestPathsForEmptyDirsFails verifies behavior for the covered scenario.
func TestPathsForEmptyDirsFails(t *testing.T) {
	_, err := PathsFor("darwin", nil, "", "/tmp/data", "tillsyn")
	if err == nil {
		t.Fatal("expected error for empty dirs")
	}
}

// TestPathsForDarwinFallback verifies behavior for the covered scenario.
func TestPathsForDarwinFallback(t *testing.T) {
	p, err := PathsFor("darwin", map[string]string{
		"XDG_CONFIG_HOME": "/ignored",
		"XDG_DATA_HOME":   "/ignored",
	}, "/Users/me/Library/Application Support", "/Users/me/Library/Application Support", "tillsyn")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	wantConfig := filepath.Join("/Users/me/Library/Application Support", "tillsyn", "config.toml")
	wantDB := filepath.Join("/Users/me/Library/Application Support", "tillsyn", "tillsyn.db")
	wantLogs := filepath.Join("/Users/me/Library/Application Support", "tillsyn", "logs")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
	if p.LogsDir != wantLogs {
		t.Fatalf("unexpected logs dir %q", p.LogsDir)
	}
}

// TestPathsForUnknownFallback verifies behavior for the covered scenario.
func TestPathsForUnknownFallback(t *testing.T) {
	p, err := PathsFor("freebsd", map[string]string{}, "/cfg", "/data", "tillsyn")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	wantConfig := filepath.Join("/cfg", "tillsyn", "config.toml")
	wantData := filepath.Join("/data", "tillsyn")
	wantLogs := filepath.Join("/data", "tillsyn", "logs")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DataDir != wantData {
		t.Fatalf("unexpected data dir %q", p.DataDir)
	}
	if p.LogsDir != wantLogs {
		t.Fatalf("unexpected logs dir %q", p.LogsDir)
	}
}

// TestPathsForLinuxFallbackWithoutXDG verifies behavior for the covered scenario.
func TestPathsForLinuxFallbackWithoutXDG(t *testing.T) {
	p, err := PathsFor("linux", map[string]string{}, "/home/me/.config", "/home/me/.local/share", "tillsyn")
	if err != nil {
		t.Fatalf("PathsFor() error = %v", err)
	}
	wantConfig := filepath.Join("/home/me/.config", "tillsyn", "config.toml")
	wantDB := filepath.Join("/home/me/.local/share", "tillsyn", "tillsyn.db")
	wantLogs := filepath.Join("/home/me/.local/share", "tillsyn", "logs")
	if p.ConfigPath != wantConfig {
		t.Fatalf("unexpected config path %q", p.ConfigPath)
	}
	if p.DBPath != wantDB {
		t.Fatalf("unexpected db path %q", p.DBPath)
	}
	if p.LogsDir != wantLogs {
		t.Fatalf("unexpected logs dir %q", p.LogsDir)
	}
}

// TestDefaultPathsSmoke verifies behavior for the covered scenario.
func TestDefaultPathsSmoke(t *testing.T) {
	p, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}
	if p.ConfigPath == "" || p.DBPath == "" || p.DataDir == "" || p.LogsDir == "" {
		t.Fatalf("expected non-empty paths, got %#v", p)
	}
}

// TestDefaultPathsWithOptionsDevMode verifies behavior for the covered scenario.
func TestDefaultPathsWithOptionsDevMode(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	p, err := DefaultPathsWithOptions(Options{AppName: "tillsyn", DevMode: true, WorkingDir: workspace})
	if err != nil {
		t.Fatalf("DefaultPathsWithOptions() error = %v", err)
	}
	if got, want := filepath.Dir(p.ConfigPath), filepath.Join(workspace, ".tillsyn"); got != want {
		t.Fatalf("config home = %q, want %q", got, want)
	}
	if filepath.Base(p.DBPath) != "tillsyn.db" {
		t.Fatalf("expected dev db name, got %q", p.DBPath)
	}
}

// TestDefaultPathsWithOptionsStableHome verifies the stable runtime defaults under ~/.tillsyn-style homes.
func TestDefaultPathsWithOptionsStableHome(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}

	p, err := DefaultPathsWithOptions(Options{AppName: "tillsyn"})
	if err != nil {
		t.Fatalf("DefaultPathsWithOptions() error = %v", err)
	}
	wantRoot := filepath.Join(home, ".tillsyn")
	if got := filepath.Dir(p.ConfigPath); got != wantRoot {
		t.Fatalf("config home = %q, want %q", got, wantRoot)
	}
	if p.DBPath != filepath.Join(wantRoot, "tillsyn.db") {
		t.Fatalf("db path = %q, want %q", p.DBPath, filepath.Join(wantRoot, "tillsyn.db"))
	}
	if p.LogsDir != filepath.Join(wantRoot, "logs") {
		t.Fatalf("logs dir = %q, want %q", p.LogsDir, filepath.Join(wantRoot, "logs"))
	}
}

// TestPathsForHome verifies explicit runtime-home path resolution.
func TestPathsForHome(t *testing.T) {
	root := filepath.Join("/tmp", ".tillsyn")
	p, err := PathsForHome(root, "tillsyn")
	if err != nil {
		t.Fatalf("PathsForHome() error = %v", err)
	}
	if p.ConfigPath != filepath.Join(root, "config.toml") {
		t.Fatalf("config path = %q, want %q", p.ConfigPath, filepath.Join(root, "config.toml"))
	}
	if p.DBPath != filepath.Join(root, "tillsyn.db") {
		t.Fatalf("db path = %q, want %q", p.DBPath, filepath.Join(root, "tillsyn.db"))
	}
	if p.LogsDir != filepath.Join(root, "logs") {
		t.Fatalf("logs dir = %q, want %q", p.LogsDir, filepath.Join(root, "logs"))
	}
}

// TestDefaultPathsWithOptionsHomeOverride verifies an explicit runtime home wins over stable/dev defaults.
func TestDefaultPathsWithOptionsHomeOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), ".override")
	p, err := DefaultPathsWithOptions(Options{
		AppName: "tillsyn",
		DevMode: true,
		HomeDir: override,
	})
	if err != nil {
		t.Fatalf("DefaultPathsWithOptions() error = %v", err)
	}
	if got := filepath.Dir(p.ConfigPath); got != override {
		t.Fatalf("config home = %q, want %q", got, override)
	}
}

// TestDotRuntimeDirName verifies hidden runtime-home directory naming.
func TestDotRuntimeDirName(t *testing.T) {
	cases := []struct {
		appName string
		want    string
	}{
		{appName: "", want: ".tillsyn"},
		{appName: "tillsyn", want: ".tillsyn"},
		{appName: ".tillsyn-dev", want: ".tillsyn-dev"},
	}
	for _, tc := range cases {
		if got := dotRuntimeDirName(tc.appName); got != tc.want {
			t.Fatalf("dotRuntimeDirName(%q) = %q, want %q", tc.appName, got, tc.want)
		}
	}
}

// TestWorkspaceRootFromPrefersNearestWorkspaceMarker verifies repo-local dev homes anchor at the nearest workspace root.
func TestWorkspaceRootFromPrefersNearestWorkspaceMarker(t *testing.T) {
	workspace := t.TempDir()
	nested := filepath.Join(workspace, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	if got := workspaceRootFrom(nested); got != workspace {
		t.Fatalf("workspaceRootFrom() = %q, want %q", got, workspace)
	}
}

// TestHasWorkspaceMarkerDetectsGitDirectory verifies .git is treated as a workspace marker.
func TestHasWorkspaceMarkerDetectsGitDirectory(t *testing.T) {
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, ".git"), 0o755); err != nil {
		t.Fatalf("Mkdir(.git) error = %v", err)
	}
	if !hasWorkspaceMarker(workspace) {
		t.Fatal("hasWorkspaceMarker() = false, want true")
	}
}

// TestDefaultLegacyPathsUsesLegacyPlatformLayout verifies the compatibility helper still exposes the prior OS-specific layout.
func TestDefaultLegacyPathsUsesLegacyPlatformLayout(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
		t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
		t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	} else {
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	}

	p, err := DefaultLegacyPaths()
	if err != nil {
		t.Fatalf("DefaultLegacyPaths() error = %v", err)
	}
	wantConfig := filepath.Join(home, ".config", "tillsyn", "config.toml")
	wantDB := filepath.Join(home, ".local", "share", "tillsyn", "tillsyn.db")
	if runtime.GOOS == "darwin" {
		wantConfig = filepath.Join(home, "Library", "Application Support", "tillsyn", "config.toml")
		wantDB = filepath.Join(home, "Library", "Application Support", "tillsyn", "tillsyn.db")
	}
	if runtime.GOOS == "windows" {
		wantConfig = filepath.Join(home, "AppData", "Roaming", "tillsyn", "config.toml")
		wantDB = filepath.Join(home, "AppData", "Local", "tillsyn", "tillsyn.db")
	}
	if got := p.ConfigPath; got != wantConfig {
		t.Fatalf("config path = %q, want %q", got, wantConfig)
	}
	if got := p.DBPath; got != wantDB {
		t.Fatalf("db path = %q, want %q", got, wantDB)
	}
}
