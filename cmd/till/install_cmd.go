package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/platform"
	"github.com/spf13/cobra"
)

// newInstallCommand returns the `till install` cobra command, which bootstraps
// the local Tillsyn dev environment by creating the dev config and enforcing
// debug logging.
//
// rootOpts is passed by pointer so the RunE closure reads the live values
// cobra wrote into &rootOpts.appName / &rootOpts.homeDir during flag parse —
// see main.go:508-513 (PersistentFlags().StringVar(&rootOpts.appName, ...)).
// Capturing by value would freeze the pre-parse defaults and ignore --app /
// --home.
func newInstallCommand(stdout io.Writer, rootOpts *rootCommandOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Bootstrap the local Tillsyn dev environment (creates the dev config, enforces [logging] level = \"debug\")",
		Long: strings.TrimSpace(`
Install the local Tillsyn dev environment: create the dev config file from
the shipped default template when missing, then force the local dev logging
level to debug.

Use this when bootstrapping a fresh local workstation or when you want the
repo default development config file restored quickly. This is a per-machine
setup command — see till init for per-project setup.
`),
		Example: strings.Join([]string{
			"  till install",
			"  till --app tillsyn install",
			"  till --home /tmp/tillsyn-dev install",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInstall(stdout, *rootOpts)
		},
	}
}

// runInstall creates the dev config file and enforces debug logging level.
// The Laslig table title is "Dev Config" byte-for-byte — test bodies assert
// this substring (install_cmd_test.go:51 + :109); do NOT rename.
func runInstall(stdout io.Writer, opts rootCommandOptions) error {
	if stdout == nil {
		stdout = io.Discard
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{
		AppName: opts.appName,
		DevMode: true,
		HomeDir: opts.homeDir,
	})
	if err != nil {
		return fmt.Errorf("resolve dev paths: %w", err)
	}

	configPath := paths.ConfigPath
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create dev config directory: %w", err)
	}

	created := false
	if _, err := os.Stat(configPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat dev config: %w", err)
		}
		templateBytes, templateErr := config.DefaultTemplate()
		if templateErr != nil {
			return templateErr
		}
		if writeErr := os.WriteFile(configPath, templateBytes, 0o644); writeErr != nil {
			return fmt.Errorf("write dev config: %w", writeErr)
		}
		created = true
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read dev config: %w", err)
	}
	updated := ensureLoggingSectionDebug(string(content))
	if updated != string(content) {
		if err := os.WriteFile(configPath, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("write updated dev config: %w", err)
		}
	}

	msg := "dev config already exists"
	if created {
		msg = "created dev config"
	}
	return writeCLIKV(stdout, "Dev Config", [][2]string{
		{"status", msg},
		{"config path", shellEscapePath(configPath)},
		{"logging level", "debug"},
	})
}
