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
	AppName string
	DevMode bool
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
	if opts.DevMode {
		appName += "-dev"
	}

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
	return PathsFor(runtime.GOOS, env, configDir, dataDir, appName)
}

// PathsFor handles paths for.
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
