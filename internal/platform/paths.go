package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Paths represents paths data used by this package.
type Paths struct {
	ConfigPath string
	DataDir    string
	DBPath     string
	LogsDir    string
}

// Options defines optional settings for configuration.
type Options struct {
	AppName    string
	DevMode    bool
	HomeDir    string
	WorkingDir string
}

// DefaultPaths returns default paths.
func DefaultPaths() (Paths, error) {
	return DefaultPathsWithOptions(Options{AppName: "tillsyn"})
}

// DefaultPathsWithOptions returns default paths with options.
func DefaultPathsWithOptions(opts Options) (Paths, error) {
	appName := strings.TrimSpace(opts.AppName)
	if appName == "" {
		appName = "tillsyn"
	}

	if homeDir := strings.TrimSpace(opts.HomeDir); homeDir != "" {
		return PathsForHome(homeDir, appName)
	}

	if opts.DevMode {
		workingDir := strings.TrimSpace(opts.WorkingDir)
		if workingDir == "" {
			var err error
			workingDir, err = os.Getwd()
			if err != nil {
				return Paths{}, fmt.Errorf("working directory: %w", err)
			}
		}
		return PathsForHome(filepath.Join(workspaceRootFrom(workingDir), dotRuntimeDirName(appName)), appName)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("user home dir: %w", err)
	}
	return PathsForHome(filepath.Join(homeDir, dotRuntimeDirName(appName)), appName)
}

// PathsForHome resolves config, database, and log paths under one explicit runtime home directory.
func PathsForHome(homeDir, appName string) (Paths, error) {
	homeDir = strings.TrimSpace(homeDir)
	if homeDir == "" {
		return Paths{}, fmt.Errorf("empty home dir")
	}
	appName = strings.TrimSpace(appName)
	if appName == "" {
		return Paths{}, fmt.Errorf("empty app name")
	}
	homeDir = filepath.Clean(homeDir)
	return Paths{
		ConfigPath: filepath.Join(homeDir, "config.toml"),
		DataDir:    homeDir,
		DBPath:     filepath.Join(homeDir, appName+".db"),
		LogsDir:    filepath.Join(homeDir, "logs"),
	}, nil
}

// dotRuntimeDirName returns the hidden runtime-home directory name for one app.
func dotRuntimeDirName(appName string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "tillsyn"
	}
	if strings.HasPrefix(appName, ".") {
		return appName
	}
	return "." + appName
}

// PathsFor handles paths for one OS-specific config/data root pair.
func PathsFor(goos string, env map[string]string, userConfigDir, userDataDir, appName string) (Paths, error) {
	if userConfigDir == "" || userDataDir == "" {
		return Paths{}, fmt.Errorf("empty base dirs")
	}
	appName = strings.TrimSpace(appName)
	if appName == "" {
		return Paths{}, fmt.Errorf("empty app name")
	}

	configBase := userConfigDir
	dataBase := userDataDir

	switch goos {
	case "linux":
		if v := env["XDG_CONFIG_HOME"]; v != "" {
			configBase = v
		}
		if v := env["XDG_DATA_HOME"]; v != "" {
			dataBase = v
		}
	case "windows":
		if v := env["APPDATA"]; v != "" {
			configBase = v
		}
		if v := env["LOCALAPPDATA"]; v != "" {
			dataBase = v
		}
	case "darwin":
		// Keep os.UserConfigDir/UserCacheDir defaults for macOS.
	default:
		// Fallback for other platforms.
	}

	appConfigDir := filepath.Join(configBase, appName)
	appDataDir := filepath.Join(dataBase, appName)
	dbName := appName + ".db"
	return Paths{
		ConfigPath: filepath.Join(appConfigDir, "config.toml"),
		DataDir:    appDataDir,
		DBPath:     filepath.Join(appDataDir, dbName),
		LogsDir:    filepath.Join(appDataDir, "logs"),
	}, nil
}

// workspaceRootFrom resolves the nearest ancestor workspace marker for dev runtime placement.
func workspaceRootFrom(start string) string {
	start = filepath.Clean(strings.TrimSpace(start))
	if start == "" {
		return "."
	}
	dir := start
	for {
		if hasWorkspaceMarker(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start
		}
		dir = parent
	}
}

// hasWorkspaceMarker reports whether a directory looks like a project workspace root.
func hasWorkspaceMarker(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	return false
}

// DefaultLegacyPaths returns the prior OS-specific config/data-dir layout for compatibility tests and migrations.
func DefaultLegacyPaths() (Paths, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("user config dir: %w", err)
	}
	dataDir := configDir
	if runtime.GOOS == "linux" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return Paths{}, fmt.Errorf("user home dir: %w", homeErr)
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	if runtime.GOOS == "windows" {
		if v := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); v != "" {
			dataDir = v
		}
	}

	env := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"APPDATA":         os.Getenv("APPDATA"),
		"LOCALAPPDATA":    os.Getenv("LOCALAPPDATA"),
	}
	return PathsFor(runtime.GOOS, env, configDir, dataDir, "tillsyn")
}
