package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
	charmLog "github.com/charmbracelet/log"
	"github.com/google/uuid"
	fantasyembed "github.com/hylla/tillsyn/internal/adapters/embeddings/fantasy"
	serveradapter "github.com/hylla/tillsyn/internal/adapters/server"
	servercommon "github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/config"
	"github.com/hylla/tillsyn/internal/platform"
	"github.com/hylla/tillsyn/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// version stores a package-level helper value.
var version = "dev"

// program represents program data used by this package.
type program interface {
	Run() (tea.Model, error)
}

// programFactory stores a package-level helper value.
var programFactory = func(m tea.Model) program {
	return tea.NewProgram(m)
}

// serveCommandRunner starts the HTTP+MCP serve flow.
var serveCommandRunner = func(ctx context.Context, cfg serveradapter.Config, deps serveradapter.Dependencies) error {
	return serveradapter.Run(ctx, cfg, deps)
}

// mcpCommandRunner starts the stdio MCP flow.
var mcpCommandRunner = func(ctx context.Context, cfg serveradapter.Config, deps serveradapter.Dependencies) error {
	return serveradapter.RunStdio(ctx, cfg, deps)
}

// supportsStyledOutputFunc allows tests to force styled output mode.
var supportsStyledOutputFunc = supportsStyledOutput

// loggingSectionHeaderPattern matches a [logging] TOML section header.
var loggingSectionHeaderPattern = regexp.MustCompile(`(?m)^\[logging\][ \t]*$`)

// tomlSectionHeaderPattern matches any TOML section header.
var tomlSectionHeaderPattern = regexp.MustCompile(`(?m)^\[[^\]\r\n]+\][ \t]*$`)

// loggingLevelLinePattern matches a level assignment line inside [logging].
var loggingLevelLinePattern = regexp.MustCompile(`(?m)^[ \t]*level[ \t]*=[^\r\n]*$`)

// main handles main.
func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

// rootCommandOptions stores top-level CLI option values.
type rootCommandOptions struct {
	configPath  string
	dbPath      string
	appName     string
	devMode     bool
	showVersion bool
}

// serveCommandOptions stores serve subcommand option values.
type serveCommandOptions struct {
	httpBind    string
	apiEndpoint string
	mcpEndpoint string
}

// mcpCommandOptions stores stdio MCP subcommand option values.
type mcpCommandOptions struct{}

// exportCommandOptions stores export subcommand option values.
type exportCommandOptions struct {
	outPath         string
	includeArchived bool
}

// importCommandOptions stores import subcommand option values.
type importCommandOptions struct {
	inPath string
}

// run executes the CLI command tree through Fang+Cobra.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	rootOpts := rootCommandOptions{
		appName: "tillsyn",
		devMode: version == "dev",
	}
	if envDev, ok := parseBoolEnv("TILL_DEV_MODE"); ok {
		rootOpts.devMode = envDev
	}
	if envApp := strings.TrimSpace(os.Getenv("TILL_APP_NAME")); envApp != "" {
		rootOpts.appName = envApp
	}

	serveOpts := serveCommandOptions{
		httpBind:    "127.0.0.1:5437",
		apiEndpoint: "/api/v1",
		mcpEndpoint: "/mcp",
	}
	mcpOpts := mcpCommandOptions{}
	exportOpts := exportCommandOptions{
		outPath:         "-",
		includeArchived: true,
	}
	importOpts := importCommandOptions{}

	rootCmd := &cobra.Command{
		Use:           "till",
		Short:         "Terminal kanban board with stdio MCP and HTTP adapters",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "", rootOpts, serveOpts, mcpOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)

	rootCmd.PersistentFlags().StringVar(&rootOpts.configPath, "config", "", "Path to config TOML")
	rootCmd.PersistentFlags().StringVar(&rootOpts.dbPath, "db", "", "Path to sqlite database")
	rootCmd.PersistentFlags().StringVar(&rootOpts.appName, "app", rootOpts.appName, "Application name for config/data path resolution")
	rootCmd.PersistentFlags().BoolVar(&rootOpts.devMode, "dev", rootOpts.devMode, "Use dev mode paths (<app>-dev)")
	rootCmd.PersistentFlags().BoolVar(&rootOpts.showVersion, "version", false, "Show version")

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP and MCP endpoints",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "serve", rootOpts, serveOpts, mcpOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	serveCmd.Flags().StringVar(&serveOpts.httpBind, "http", serveOpts.httpBind, "HTTP listen address")
	serveCmd.Flags().StringVar(&serveOpts.apiEndpoint, "api-endpoint", serveOpts.apiEndpoint, "HTTP API base endpoint")
	serveCmd.Flags().StringVar(&serveOpts.mcpEndpoint, "mcp-endpoint", serveOpts.mcpEndpoint, "MCP streamable HTTP endpoint")

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP over stdio for local integrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "mcp", rootOpts, serveOpts, mcpOpts, exportOpts, importOpts, stdout, stderr)
		},
	}

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export a snapshot JSON payload",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "export", rootOpts, serveOpts, mcpOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	exportCmd.Flags().StringVar(&exportOpts.outPath, "out", exportOpts.outPath, "Output file path ('-' for stdout)")
	exportCmd.Flags().BoolVar(&exportOpts.includeArchived, "include-archived", exportOpts.includeArchived, "Include archived projects/columns/tasks")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a snapshot JSON payload",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "import", rootOpts, serveOpts, mcpOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	importCmd.Flags().StringVar(&importOpts.inPath, "in", "", "Input snapshot JSON file")

	pathsCmd := &cobra.Command{
		Use:   "paths",
		Short: "Print resolved config/data/db paths",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if rootOpts.showVersion {
				return writeVersion(stdout)
			}
			paths, err := platform.DefaultPathsWithOptions(platform.Options{
				AppName: rootOpts.appName,
				DevMode: rootOpts.devMode,
			})
			if err != nil {
				return err
			}
			return writePathsOutput(stdout, rootOpts, paths)
		},
	}

	initDevConfigCmd := &cobra.Command{
		Use:   "init-dev-config",
		Short: "Create the dev config file and enforce [logging] level = \"debug\"",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInitDevConfig(stdout, rootOpts)
		},
	}

	rootCmd.AddCommand(serveCmd, mcpCmd, exportCmd, importCmd, pathsCmd, initDevConfigCmd)
	return fang.Execute(
		ctx,
		rootCmd,
		fang.WithoutCompletions(),
		fang.WithoutManpage(),
		fang.WithoutVersion(),
	)
}

// writeVersion writes the current CLI version to stdout.
func writeVersion(stdout io.Writer) error {
	if _, err := fmt.Fprintf(stdout, "till %s\n", version); err != nil {
		return fmt.Errorf("write version output: %w", err)
	}
	return nil
}

// writePathsOutput renders resolved paths using Fang-aligned styling.
func writePathsOutput(stdout io.Writer, opts rootCommandOptions, paths platform.Paths) error {
	if !supportsStyledOutputFunc(stdout) {
		return writePathsPlain(stdout, opts, paths)
	}

	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	colors := fang.DefaultColorScheme(lipgloss.LightDark(isDark))
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colors.Title)
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colors.Flag)
	valueStyle := lipgloss.NewStyle().
		Foreground(colors.Description)

	rows := []struct {
		key   string
		value string
	}{
		{key: "app", value: opts.appName},
		{key: "dev_mode", value: fmt.Sprintf("%t", opts.devMode)},
		{key: "config", value: paths.ConfigPath},
		{key: "data_dir", value: paths.DataDir},
		{key: "db", value: paths.DBPath},
	}

	maxKeyWidth := 0
	for _, row := range rows {
		if len(row.key) > maxKeyWidth {
			maxKeyWidth = len(row.key)
		}
	}

	lines := make([]string, 0, len(rows)+1)
	lines = append(lines, titleStyle.Render("Resolved Paths"))
	for _, row := range rows {
		paddedKey := fmt.Sprintf("%-*s:", maxKeyWidth, row.key)
		line := lipgloss.JoinHorizontal(
			lipgloss.Left,
			keyStyle.Render(paddedKey),
			"  ",
			valueStyle.Render(row.value),
		)
		lines = append(lines, line)
	}
	if _, err := fmt.Fprintln(stdout, strings.Join(lines, "\n")); err != nil {
		return fmt.Errorf("write paths output: %w", err)
	}
	return nil
}

// writePathsPlain renders resolved paths in stable key/value text for scripts.
func writePathsPlain(stdout io.Writer, opts rootCommandOptions, paths platform.Paths) error {
	if _, err := fmt.Fprintf(stdout, "app: %s\n", opts.appName); err != nil {
		return fmt.Errorf("write paths app output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "dev_mode: %t\n", opts.devMode); err != nil {
		return fmt.Errorf("write paths dev output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "config: %s\n", paths.ConfigPath); err != nil {
		return fmt.Errorf("write paths config output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "data_dir: %s\n", paths.DataDir); err != nil {
		return fmt.Errorf("write paths data output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "db: %s\n", paths.DBPath); err != nil {
		return fmt.Errorf("write paths db output: %w", err)
	}
	return nil
}

// resolvedRuntimePaths stores CLI runtime config/db path decisions for one command.
type resolvedRuntimePaths struct {
	ConfigPath                string
	DBPath                    string
	DBOverridden              bool
	UsesLocalMCPRuntime       bool
	ConfigUsesLocalMCPRuntime bool
	DBUsesLocalMCPRuntime     bool
}

// resolveRuntimePaths resolves config and DB paths, including stdio-MCP local runtime defaults.
func resolveRuntimePaths(command string, opts rootCommandOptions, paths platform.Paths) (resolvedRuntimePaths, error) {
	configPath := strings.TrimSpace(opts.configPath)
	configOverridden := configPath != ""
	if !configOverridden {
		if envPath := strings.TrimSpace(os.Getenv("TILL_CONFIG")); envPath != "" {
			configPath = envPath
			configOverridden = true
		} else {
			configPath = paths.ConfigPath
		}
	}

	dbPath := strings.TrimSpace(opts.dbPath)
	dbOverridden := dbPath != ""
	if !dbOverridden {
		if envPath := strings.TrimSpace(os.Getenv("TILL_DB_PATH")); envPath != "" {
			dbPath = envPath
			dbOverridden = true
		} else {
			dbPath = paths.DBPath
		}
	}

	out := resolvedRuntimePaths{
		ConfigPath:   configPath,
		DBPath:       dbPath,
		DBOverridden: dbOverridden,
	}
	if command != "mcp" {
		return out, nil
	}

	localPaths, err := localMCPRuntimePaths(opts)
	if err != nil {
		return resolvedRuntimePaths{}, fmt.Errorf("resolve local mcp runtime paths: %w", err)
	}
	if !configOverridden {
		out.ConfigPath = localPaths.ConfigPath
		out.ConfigUsesLocalMCPRuntime = true
	}
	if !dbOverridden {
		out.DBPath = localPaths.DBPath
		out.DBUsesLocalMCPRuntime = true
	}
	out.UsesLocalMCPRuntime = out.ConfigUsesLocalMCPRuntime || out.DBUsesLocalMCPRuntime
	return out, nil
}

// localMCPRuntimePaths resolves repo-local config/data paths for stdio MCP sessions.
func localMCPRuntimePaths(opts rootCommandOptions) (platform.Paths, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return platform.Paths{}, fmt.Errorf("resolve working directory: %w", err)
	}
	appName := effectiveAppName(opts.appName, opts.devMode)
	baseDir := filepath.Join(workspaceRootFrom(cwd), ".tillsyn", "mcp", appName)
	return platform.Paths{
		ConfigPath: filepath.Join(baseDir, "config.toml"),
		DataDir:    baseDir,
		DBPath:     filepath.Join(baseDir, appName+".db"),
	}, nil
}

// effectiveAppName normalizes app naming the same way as platform path resolution.
func effectiveAppName(appName string, devMode bool) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "tillsyn"
	}
	if devMode {
		appName += "-dev"
	}
	return appName
}

// ensureRuntimePathParents creates repo-local stdio MCP runtime parents before startup.
func ensureRuntimePathParents(command string, paths resolvedRuntimePaths) error {
	if command != "mcp" || !paths.UsesLocalMCPRuntime {
		return nil
	}

	parents := make([]string, 0, 2)
	if paths.ConfigUsesLocalMCPRuntime {
		parents = append(parents, filepath.Dir(paths.ConfigPath))
	}
	if paths.DBUsesLocalMCPRuntime {
		parents = append(parents, filepath.Dir(paths.DBPath))
	}
	for _, parent := range parents {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return fmt.Errorf("create local mcp runtime directory %q: %w", parent, err)
		}
	}
	return nil
}

// runInitDevConfig creates the dev config file and enforces debug logging level.
func runInitDevConfig(stdout io.Writer, opts rootCommandOptions) error {
	if stdout == nil {
		stdout = io.Discard
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{
		AppName: opts.appName,
		DevMode: true,
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
		templatePath, pathErr := configExamplePath()
		if pathErr != nil {
			return pathErr
		}
		templateBytes, readErr := os.ReadFile(templatePath)
		if readErr != nil {
			return fmt.Errorf("read config example %q: %w", templatePath, readErr)
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
	if _, err := fmt.Fprintf(stdout, "%s: %s\n", msg, shellEscapePath(configPath)); err != nil {
		return fmt.Errorf("write init-dev-config output: %w", err)
	}
	return nil
}

// shellEscapePath returns a POSIX-shell-escaped path token suitable for direct paste.
func shellEscapePath(path string) string {
	var out strings.Builder
	out.Grow(len(path) + 8)
	for _, r := range path {
		switch r {
		case ' ', '\t', '\n', '\\', '(', ')', '[', ']', '\'', '"', '&', ';', '|', '<', '>', '$', '!', '*', '?', '#':
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}

// configExamplePath resolves the repository-local config example path.
func configExamplePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	return filepath.Join(workspaceRootFrom(cwd), "config.example.toml"), nil
}

// seedStartupConfigFromExampleIfMissing seeds the runtime config from config.example.toml on first-launch TUI startup.
func seedStartupConfigFromExampleIfMissing(command, configPath string) error {
	if command != "" {
		return nil
	}
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return nil
	}

	if _, err := os.Stat(configPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat startup config %q: %w", configPath, err)
	}

	templatePath, err := configExamplePath()
	if err != nil {
		return fmt.Errorf("resolve config example path: %w", err)
	}
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Keep startup behavior compatible outside repository checkouts where the template may be unavailable.
			return nil
		}
		return fmt.Errorf("read config example %q: %w", templatePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create startup config directory: %w", err)
	}
	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return fmt.Errorf("create startup config %q: %w", configPath, err)
	}
	if _, err := file.Write(templateBytes); err != nil {
		_ = file.Close()
		return fmt.Errorf("write startup config %q: %w", configPath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close startup config %q: %w", configPath, err)
	}
	return nil
}

// ensureLoggingSectionDebug rewrites TOML content so [logging].level is always "debug".
func ensureLoggingSectionDebug(content string) string {
	headerMatch := loggingSectionHeaderPattern.FindStringIndex(content)
	if headerMatch != nil {
		sectionBodyStart := headerMatch[1]
		sectionBodyEnd := len(content)
		nextSectionMatch := tomlSectionHeaderPattern.FindStringIndex(content[sectionBodyStart:])
		if nextSectionMatch != nil {
			sectionBodyEnd = sectionBodyStart + nextSectionMatch[0]
		}

		sectionBody := content[sectionBodyStart:sectionBodyEnd]
		updatedBody := sectionBody
		if loggingLevelLinePattern.MatchString(sectionBody) {
			updatedBody = loggingLevelLinePattern.ReplaceAllString(sectionBody, `level = "debug"`)
		} else {
			if updatedBody != "" && !strings.HasSuffix(updatedBody, "\n") {
				updatedBody += "\n"
			}
			updatedBody += "level = \"debug\"\n"
		}
		// Reassemble the file while preserving all non-[logging] content exactly.
		return content[:sectionBodyStart] + updatedBody + content[sectionBodyEnd:]
	}

	trimmed := strings.TrimRight(content, "\r\n")
	if trimmed == "" {
		return "[logging]\nlevel = \"debug\"\n"
	}
	return trimmed + "\n\n[logging]\nlevel = \"debug\"\n"
}

// supportsStyledOutput reports whether output should include terminal styles.
func supportsStyledOutput(w io.Writer) bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

// executeCommandFlow runs the runtime setup + command-specific execution path.
func executeCommandFlow(
	ctx context.Context,
	command string,
	rootOpts rootCommandOptions,
	serveOpts serveCommandOptions,
	_ mcpCommandOptions,
	exportOpts exportCommandOptions,
	importOpts importCommandOptions,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if rootOpts.showVersion {
		return writeVersion(stdout)
	}

	paths, err := platform.DefaultPathsWithOptions(platform.Options{
		AppName: rootOpts.appName,
		DevMode: rootOpts.devMode,
	})
	if err != nil {
		return err
	}

	resolvedPaths, err := resolveRuntimePaths(command, rootOpts, paths)
	if err != nil {
		return err
	}
	configPath := resolvedPaths.ConfigPath
	dbPath := resolvedPaths.DBPath
	dbOverridden := resolvedPaths.DBOverridden
	if err := ensureRuntimePathParents(command, resolvedPaths); err != nil {
		return err
	}
	if err := seedStartupConfigFromExampleIfMissing(command, configPath); err != nil {
		return fmt.Errorf("seed startup config %q: %w", configPath, err)
	}

	defaultCfg := config.Default(dbPath)
	cfg, err := config.Load(configPath, defaultCfg)
	if err != nil {
		return fmt.Errorf("load config %q: %w", configPath, err)
	}
	if dbOverridden || (command == "mcp" && resolvedPaths.DBUsesLocalMCPRuntime) {
		cfg.Database.Path = dbPath
	}
	if command == "" {
		if err := ensureStartupIdentityActorID(configPath, &cfg); err != nil {
			return fmt.Errorf("bootstrap identity.actor_id: %w", err)
		}
	}
	bootstrapRequired := startupBootstrapRequired(cfg)

	logger, err := newRuntimeLogger(stderr, rootOpts.appName, rootOpts.devMode, cfg.Logging, time.Now)
	if err != nil {
		return fmt.Errorf("configure runtime logger: %w", err)
	}
	if command == "" {
		// Keep TUI rendering clean: runtime logs stay in the dev-file sink while the board is active.
		logger.SetConsoleEnabled(false)
	}
	logger.InstallAsDefault(rootOpts.appName)
	defer func() {
		if closeErr := logger.Close(); closeErr != nil && logger.shouldLogToSink(logger.consoleSink) {
			// Keep TUI shutdown quiet on the terminal when console logging is intentionally muted.
			_, _ = fmt.Fprintf(stderr, "warning: close runtime log sink: %v\n", closeErr)
		}
	}()
	defer logger.RestoreDefault()

	logger.Info("startup configuration resolved", "app", rootOpts.appName, "dev_mode", rootOpts.devMode, "command", command, "bootstrap_required", bootstrapRequired)
	logger.Debug("runtime paths resolved", "config_path", configPath, "data_dir", paths.DataDir, "db_path", dbPath)
	logger.Info("configuration loaded", "config_path", configPath, "db_path", cfg.Database.Path, "log_level", cfg.Logging.Level)
	if devPath := logger.DevLogPath(); devPath != "" {
		logger.Info("dev file logging enabled", "path", devPath)
	}

	logger.Info("opening sqlite repository", "db_path", cfg.Database.Path)
	repo, err := sqlite.Open(cfg.Database.Path)
	if err != nil {
		logger.Error("sqlite open failed", "db_path", cfg.Database.Path, "err", err)
		return fmt.Errorf("open sqlite repository: %w", err)
	}
	defer func() {
		if closeErr := repo.Close(); closeErr != nil {
			logger.Warn("sqlite close failed", "db_path", cfg.Database.Path, "err", closeErr)
		}
	}()
	logger.Info("sqlite repository ready", "db_path", cfg.Database.Path, "migrations", "ensured")

	var embeddingGenerator app.EmbeddingGenerator
	if cfg.Embeddings.Enabled {
		generator, err := fantasyembed.New(ctx, fantasyembed.Config{
			Provider:   cfg.Embeddings.Provider,
			Model:      cfg.Embeddings.Model,
			APIKeyEnv:  cfg.Embeddings.APIKeyEnv,
			BaseURL:    cfg.Embeddings.BaseURL,
			Dimensions: cfg.Embeddings.Dimensions,
		})
		if err != nil {
			logger.Warn("embeddings disabled due setup error; keyword fallback remains active", "err", err)
		} else {
			embeddingGenerator = generator
			logger.Info("embeddings runtime enabled", "provider", cfg.Embeddings.Provider, "model", cfg.Embeddings.Model, "query_top_k", cfg.Embeddings.QueryTopK)
		}
	}

	svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{
		DefaultDeleteMode:        app.DeleteMode(cfg.Delete.DefaultMode),
		AutoCreateProjectColumns: true,
		EmbeddingGenerator:       embeddingGenerator,
		SearchLexicalWeight:      cfg.Embeddings.LexicalWeight,
		SearchSemanticWeight:     cfg.Embeddings.SemanticWeight,
		SearchSemanticCandidates: cfg.Embeddings.QueryTopK,
	})
	logger.Debug("application service initialized", "default_delete_mode", cfg.Delete.DefaultMode)

	switch command {
	case "":
		logger.Info("command flow start", "command", "tui")
	case "serve":
		logger.Info("command flow start", "command", "serve")
		if err := runServe(ctx, svc, rootOpts.appName, serveOpts); err != nil {
			logger.Error("command flow failed", "command", "serve", "err", err)
			return fmt.Errorf("run serve command: %w", err)
		}
		logger.Info("command flow complete", "command", "serve")
		return nil
	case "mcp":
		logger.Info("command flow start", "command", "mcp", "transport", "stdio")
		if err := runMCP(ctx, svc, rootOpts.appName, serveOpts); err != nil {
			logger.Error("command flow failed", "command", "mcp", "transport", "stdio", "err", err)
			return fmt.Errorf("run mcp command: %w", err)
		}
		logger.Info("command flow complete", "command", "mcp", "transport", "stdio")
		return nil
	case "export":
		logger.Info("command flow start", "command", "export")
		if err := runExport(ctx, svc, exportOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "export", "err", err)
			return fmt.Errorf("run export command: %w", err)
		}
		logger.Info("command flow complete", "command", "export")
		return nil
	case "import":
		logger.Info("command flow start", "command", "import")
		if err := runImport(ctx, svc, importOpts); err != nil {
			logger.Error("command flow failed", "command", "import", "err", err)
			return fmt.Errorf("run import command: %w", err)
		}
		logger.Info("command flow complete", "command", "import")
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	m := tui.NewModel(
		svc,
		tui.WithLaunchProjectPicker(true),
		tui.WithStartupBootstrap(bootstrapRequired),
		tui.WithAutoRefreshInterval(2*time.Second),
		tui.WithRuntimeConfig(toTUIRuntimeConfig(cfg)),
		tui.WithReloadConfigCallback(func() (tui.RuntimeConfig, error) {
			logger.Info("runtime config reload requested", "config_path", configPath)
			reloaded, err := loadRuntimeConfig(configPath, defaultCfg, dbPath, dbOverridden)
			if err != nil {
				logger.Error("runtime config reload failed", "config_path", configPath, "err", err)
				return tui.RuntimeConfig{}, err
			}
			logger.Info("runtime config reload complete", "config_path", configPath)
			return reloaded, nil
		}),
		tui.WithSaveProjectRootCallback(func(projectSlug, rootPath string) error {
			logger.Info("project root update requested", "project_slug", projectSlug, "root_path", rootPath, "config_path", configPath)
			if err := persistProjectRoot(configPath, projectSlug, rootPath); err != nil {
				logger.Error("project root update failed", "project_slug", projectSlug, "root_path", rootPath, "config_path", configPath, "err", err)
				return err
			}
			logger.Info("project root update complete", "project_slug", projectSlug, "root_path", rootPath, "config_path", configPath)
			return nil
		}),
		tui.WithSaveLabelsConfigCallback(func(projectSlug string, globalLabels, projectLabels []string) error {
			logger.Info("labels config update requested", "project_slug", projectSlug, "global_count", len(globalLabels), "project_count", len(projectLabels), "config_path", configPath)
			if err := persistAllowedLabels(configPath, projectSlug, globalLabels, projectLabels); err != nil {
				logger.Error("labels config update failed", "project_slug", projectSlug, "config_path", configPath, "err", err)
				return err
			}
			logger.Info("labels config update complete", "project_slug", projectSlug, "global_count", len(globalLabels), "project_count", len(projectLabels), "config_path", configPath)
			return nil
		}),
		tui.WithSaveBootstrapConfigCallback(func(bootstrap tui.BootstrapConfig) error {
			actorID := strings.TrimSpace(bootstrap.ActorID)
			if actorID == "" {
				actorID = strings.TrimSpace(cfg.Identity.ActorID)
			}
			displayName := strings.TrimSpace(bootstrap.DisplayName)
			actorType := sanitizeBootstrapActorType(bootstrap.DefaultActorType)
			searchRoots := cloneSearchRoots(bootstrap.SearchRoots)
			logger.Info("bootstrap settings update requested", "config_path", configPath, "display_name", displayName, "default_actor_type", actorType, "search_roots_count", len(searchRoots))
			if err := persistIdentity(configPath, actorID, displayName, actorType); err != nil {
				logger.Error("bootstrap identity update failed", "config_path", configPath, "display_name", displayName, "default_actor_type", actorType, "err", err)
				return err
			}
			if err := persistSearchRoots(configPath, searchRoots); err != nil {
				logger.Error("bootstrap search roots update failed", "config_path", configPath, "search_roots_count", len(searchRoots), "err", err)
				return err
			}
			logger.Info("bootstrap settings update complete", "config_path", configPath, "display_name", displayName, "default_actor_type", actorType, "search_roots_count", len(searchRoots))
			return nil
		}),
	)
	logger.Info("starting tui program loop")
	_, err = programFactory(m).Run()
	if err != nil {
		logger.Error("tui program terminated with error", "err", err)
		return fmt.Errorf("run tui program: %w", err)
	}
	logger.Info("command flow complete", "command", "tui")
	return nil
}

// runServe runs the serve subcommand flow.
func runServe(ctx context.Context, svc *app.Service, appName string, opts serveCommandOptions) error {
	appAdapter := servercommon.NewAppServiceAdapter(svc)
	return serveCommandRunner(ctx, serveradapter.Config{
		HTTPBind:      opts.httpBind,
		APIEndpoint:   opts.apiEndpoint,
		MCPEndpoint:   opts.mcpEndpoint,
		ServerName:    appName,
		ServerVersion: version,
	}, serveradapter.Dependencies{
		CaptureState: appAdapter,
		Attention:    appAdapter,
	})
}

// runMCP runs the stdio MCP subcommand flow.
func runMCP(ctx context.Context, svc *app.Service, appName string, opts serveCommandOptions) error {
	appAdapter := servercommon.NewAppServiceAdapter(svc)
	return mcpCommandRunner(ctx, serveradapter.Config{
		MCPEndpoint:   opts.mcpEndpoint,
		ServerName:    appName,
		ServerVersion: version,
	}, serveradapter.Dependencies{
		CaptureState: appAdapter,
		Attention:    appAdapter,
	})
}

// runExport runs the requested command flow.
func runExport(ctx context.Context, svc *app.Service, opts exportCommandOptions, stdout io.Writer) error {
	snap, err := svc.ExportSnapshot(ctx, opts.includeArchived)
	if err != nil {
		return fmt.Errorf("export snapshot: %w", err)
	}
	encoded, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snapshot json: %w", err)
	}
	encoded = append(encoded, '\n')

	if opts.outPath == "-" {
		if _, err := stdout.Write(encoded); err != nil {
			return fmt.Errorf("write snapshot to stdout: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(opts.outPath), 0o755); err != nil {
		return fmt.Errorf("create export output dir: %w", err)
	}
	if err := os.WriteFile(opts.outPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}
	return nil
}

// runImport runs the requested command flow.
func runImport(ctx context.Context, svc *app.Service, opts importCommandOptions) error {
	if opts.inPath == "" {
		return fmt.Errorf("--in is required")
	}

	content, err := os.ReadFile(opts.inPath)
	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}
	var snap app.Snapshot
	if err := json.Unmarshal(content, &snap); err != nil {
		return fmt.Errorf("decode snapshot json: %w", err)
	}
	if err := svc.ImportSnapshot(ctx, snap); err != nil {
		return fmt.Errorf("import snapshot: %w", err)
	}
	return nil
}

// startupBootstrapRequired reports whether startup must collect required identity/root settings in TUI.
func startupBootstrapRequired(cfg config.Config) bool {
	if strings.TrimSpace(cfg.Identity.DisplayName) == "" {
		return true
	}
	return len(cfg.Paths.SearchRoots) == 0
}

// ensureStartupIdentityActorID generates and persists identity.actor_id once for TUI startup flows.
func ensureStartupIdentityActorID(configPath string, cfg *config.Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if strings.TrimSpace(cfg.Identity.ActorID) != "" {
		return nil
	}

	actorID := uuid.NewString()
	if err := persistIdentity(configPath, actorID, cfg.Identity.DisplayName, cfg.Identity.DefaultActorType); err != nil {
		return fmt.Errorf("persist generated identity.actor_id: %w", err)
	}
	cfg.Identity.ActorID = actorID
	return nil
}

// sanitizeBootstrapActorType normalizes bootstrap actor type values to supported options.
func sanitizeBootstrapActorType(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "user", "agent", "system":
		return strings.TrimSpace(strings.ToLower(raw))
	default:
		return "user"
	}
}

// parseBoolEnv parses input into a normalized form.
func parseBoolEnv(name string) (bool, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false, false
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return v, true
}

// loadRuntimeConfig loads runtime-configurable options from disk.
func loadRuntimeConfig(configPath string, defaults config.Config, dbPath string, dbOverridden bool) (tui.RuntimeConfig, error) {
	cfg, err := config.Load(configPath, defaults)
	if err != nil {
		return tui.RuntimeConfig{}, fmt.Errorf("load config %q: %w", configPath, err)
	}
	if dbOverridden {
		cfg.Database.Path = dbPath
	}
	return toTUIRuntimeConfig(cfg), nil
}

// toTUIRuntimeConfig maps persisted config values into runtime model options.
func toTUIRuntimeConfig(cfg config.Config) tui.RuntimeConfig {
	return tui.RuntimeConfig{
		DefaultDeleteMode: app.DeleteMode(cfg.Delete.DefaultMode),
		TaskFields: tui.TaskFieldConfig{
			ShowPriority:    cfg.TaskFields.ShowPriority,
			ShowDueDate:     cfg.TaskFields.ShowDueDate,
			ShowLabels:      cfg.TaskFields.ShowLabels,
			ShowDescription: cfg.TaskFields.ShowDescription,
		},
		Search: tui.SearchConfig{
			CrossProject:    cfg.Search.CrossProject,
			IncludeArchived: cfg.Search.IncludeArchived,
			States:          append([]string(nil), cfg.Search.States...),
		},
		SearchRoots: cloneSearchRoots(cfg.Paths.SearchRoots),
		Confirm: tui.ConfirmConfig{
			Delete:     cfg.Confirm.Delete,
			Archive:    cfg.Confirm.Archive,
			HardDelete: cfg.Confirm.HardDelete,
			Restore:    cfg.Confirm.Restore,
		},
		Board: tui.BoardConfig{
			ShowWIPWarnings: cfg.Board.ShowWIPWarnings,
			GroupBy:         cfg.Board.GroupBy,
		},
		UI: tui.UIConfig{
			DueSoonWindows: cfg.DueSoonDurations(),
			ShowDueSummary: cfg.UI.ShowDueSummary,
		},
		Labels: tui.LabelConfig{
			Global:         append([]string(nil), cfg.Labels.Global...),
			Projects:       cloneLabelProjectConfig(cfg.Labels.Projects),
			EnforceAllowed: cfg.Labels.EnforceAllowed,
		},
		ProjectRoots: cloneProjectRoots(cfg.ProjectRoots),
		Keys: tui.KeyConfig{
			CommandPalette: cfg.Keys.CommandPalette,
			QuickActions:   cfg.Keys.QuickActions,
			MultiSelect:    cfg.Keys.MultiSelect,
			ActivityLog:    cfg.Keys.ActivityLog,
			Undo:           cfg.Keys.Undo,
			Redo:           cfg.Keys.Redo,
		},
		Identity: tui.IdentityConfig{
			ActorID:          cfg.Identity.ActorID,
			DisplayName:      cfg.Identity.DisplayName,
			DefaultActorType: cfg.Identity.DefaultActorType,
		},
	}
}

// persistProjectRoot updates one project-root mapping in the TOML config file.
func persistProjectRoot(configPath, projectSlug, rootPath string) error {
	if err := config.UpsertProjectRoot(configPath, projectSlug, rootPath); err != nil {
		return fmt.Errorf("persist project root: %w", err)
	}
	return nil
}

// persistIdentity updates identity defaults in the TOML config file.
func persistIdentity(configPath, actorID, displayName, defaultActorType string) error {
	if err := config.UpsertIdentity(configPath, actorID, displayName, defaultActorType); err != nil {
		return fmt.Errorf("persist identity config: %w", err)
	}
	return nil
}

// persistSearchRoots updates global search roots in the TOML config file.
func persistSearchRoots(configPath string, searchRoots []string) error {
	if err := config.UpsertSearchRoots(configPath, searchRoots); err != nil {
		return fmt.Errorf("persist search roots config: %w", err)
	}
	return nil
}

// persistAllowedLabels updates global + project label defaults in the TOML config file.
func persistAllowedLabels(configPath, projectSlug string, globalLabels, projectLabels []string) error {
	if err := config.UpsertAllowedLabels(configPath, projectSlug, globalLabels, projectLabels); err != nil {
		return fmt.Errorf("persist labels config: %w", err)
	}
	return nil
}

// cloneLabelProjectConfig deep-copies per-project label lists.
func cloneLabelProjectConfig(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for key, labels := range in {
		out[key] = append([]string(nil), labels...)
	}
	return out
}

// cloneProjectRoots deep-copies project-root path mappings.
func cloneProjectRoots(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, path := range in {
		out[key] = path
	}
	return out
}

// cloneSearchRoots deep-copies global search-root paths.
func cloneSearchRoots(in []string) []string {
	return append([]string(nil), in...)
}

// runtimeLogger fans log events to a styled console sink and an optional dev-file sink.
type runtimeLogger struct {
	sinks           []*charmLog.Logger
	consoleSink     *charmLog.Logger
	consoleWriter   io.Writer
	fileWriter      io.Writer
	consoleEnabled  bool
	closeFile       func() error
	devLog          string
	level           charmLog.Level
	defaultBridge   *runtimeLogBridgeWriter
	previousDefault *charmLog.Logger
}

// newRuntimeLogger configures runtime log sinks from CLI/config state.
func newRuntimeLogger(stderr io.Writer, appName string, devMode bool, cfg config.LoggingConfig, now func() time.Time) (*runtimeLogger, error) {
	level, err := charmLog.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse logging level %q: %w", cfg.Level, err)
	}

	if now == nil {
		now = time.Now
	}
	if stderr == nil {
		stderr = io.Discard
	}

	consoleLogger := charmLog.NewWithOptions(stderr, charmLog.Options{
		Level:           level,
		Prefix:          appName,
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Formatter:       charmLog.TextFormatter,
	})

	logger := &runtimeLogger{
		sinks:          []*charmLog.Logger{consoleLogger},
		consoleSink:    consoleLogger,
		consoleWriter:  stderr,
		consoleEnabled: true,
		level:          level,
	}
	if !devMode || !cfg.DevFile.Enabled {
		return logger, nil
	}

	devLogPath, err := devLogFilePath(cfg.DevFile.Dir, appName, now().UTC())
	if err != nil {
		return nil, fmt.Errorf("resolve dev log file path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(devLogPath), 0o755); err != nil {
		return nil, fmt.Errorf("create dev log dir: %w", err)
	}
	logFile, err := os.OpenFile(devLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open dev log file: %w", err)
	}

	// Keep file output parseable and unstyled while preserving styled console logs.
	fileLogger := charmLog.NewWithOptions(logFile, charmLog.Options{
		Level:           level,
		Prefix:          appName,
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Formatter:       charmLog.LogfmtFormatter,
	})
	logger.sinks = append(logger.sinks, fileLogger)
	logger.closeFile = logFile.Close
	logger.devLog = devLogPath
	logger.fileWriter = logFile
	return logger, nil
}

// InstallAsDefault routes package-level charm/log calls through this runtime logger's sinks.
func (l *runtimeLogger) InstallAsDefault(appName string) {
	if l == nil {
		return
	}
	bridge := l.ensureDefaultBridge()
	if bridge == nil {
		return
	}
	defaultLogger := charmLog.NewWithOptions(bridge, charmLog.Options{
		Level:           l.level,
		Prefix:          appName,
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Formatter:       charmLog.LogfmtFormatter,
	})
	l.previousDefault = charmLog.Default()
	charmLog.SetDefault(defaultLogger)
}

// RestoreDefault resets the package-level default logger captured during InstallAsDefault.
func (l *runtimeLogger) RestoreDefault() {
	if l == nil || l.previousDefault == nil {
		return
	}
	charmLog.SetDefault(l.previousDefault)
	l.previousDefault = nil
}

// ensureDefaultBridge initializes the package-log bridge writer once.
func (l *runtimeLogger) ensureDefaultBridge() *runtimeLogBridgeWriter {
	if l == nil {
		return nil
	}
	if l.defaultBridge == nil {
		l.defaultBridge = &runtimeLogBridgeWriter{runtime: l}
	}
	return l.defaultBridge
}

// runtimeLogBridgeWriter fans package-level log output into runtime logger sinks.
type runtimeLogBridgeWriter struct {
	mu      sync.Mutex
	runtime *runtimeLogger
}

// Write forwards one formatted log line to active runtime sinks.
func (w *runtimeLogBridgeWriter) Write(p []byte) (int, error) {
	if w == nil || w.runtime == nil {
		return len(p), nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	var firstErr error
	if w.runtime.consoleEnabled && w.runtime.consoleWriter != nil {
		if _, err := w.runtime.consoleWriter.Write(p); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if w.runtime.fileWriter != nil {
		if _, err := w.runtime.fileWriter.Write(p); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return len(p), firstErr
}

// DevLogPath returns the active dev log file path.
func (l *runtimeLogger) DevLogPath() string {
	if l == nil {
		return ""
	}
	return l.devLog
}

// Close closes the optional dev-file sink.
func (l *runtimeLogger) Close() error {
	if l == nil || l.closeFile == nil {
		return nil
	}
	return l.closeFile()
}

// SetConsoleEnabled toggles whether the console sink receives runtime events.
func (l *runtimeLogger) SetConsoleEnabled(enabled bool) {
	if l == nil {
		return
	}
	l.consoleEnabled = enabled
}

// shouldLogToSink reports whether one sink should receive runtime output.
func (l *runtimeLogger) shouldLogToSink(sink *charmLog.Logger) bool {
	if l == nil {
		return false
	}
	if sink == nil {
		return false
	}
	if sink == l.consoleSink && !l.consoleEnabled {
		return false
	}
	return true
}

// Debug logs a debug event to all configured sinks.
func (l *runtimeLogger) Debug(msg string, keyvals ...any) {
	if l == nil {
		return
	}
	for _, sink := range l.sinks {
		if !l.shouldLogToSink(sink) {
			continue
		}
		sink.Debug(msg, keyvals...)
	}
}

// Info logs an informational event to all configured sinks.
func (l *runtimeLogger) Info(msg string, keyvals ...any) {
	if l == nil {
		return
	}
	for _, sink := range l.sinks {
		if !l.shouldLogToSink(sink) {
			continue
		}
		sink.Info(msg, keyvals...)
	}
}

// Warn logs a warning event to all configured sinks.
func (l *runtimeLogger) Warn(msg string, keyvals ...any) {
	if l == nil {
		return
	}
	for _, sink := range l.sinks {
		if !l.shouldLogToSink(sink) {
			continue
		}
		sink.Warn(msg, keyvals...)
	}
}

// Error logs an error event to all configured sinks.
func (l *runtimeLogger) Error(msg string, keyvals ...any) {
	if l == nil {
		return
	}
	for _, sink := range l.sinks {
		if !l.shouldLogToSink(sink) {
			continue
		}
		sink.Error(msg, keyvals...)
	}
}

// devLogFilePath resolves a workspace-local dev log file path for the current run day.
func devLogFilePath(configDir, appName string, now time.Time) (string, error) {
	baseDir := strings.TrimSpace(configDir)
	if baseDir == "" {
		baseDir = ".tillsyn/log"
	}
	if !filepath.IsAbs(baseDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working dir: %w", err)
		}
		baseDir = filepath.Join(workspaceRootFrom(cwd), baseDir)
	}
	fileStem := sanitizeLogFileStem(appName)
	fileName := fmt.Sprintf("%s-%s.log", fileStem, now.Format("20060102"))
	return filepath.Join(filepath.Clean(baseDir), fileName), nil
}

// workspaceRootFrom resolves the nearest ancestor workspace marker for stable local log placement.
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

// sanitizeLogFileStem normalizes app names into safe file-name segments.
func sanitizeLogFileStem(appName string) string {
	stem := strings.TrimSpace(appName)
	if stem == "" {
		return "tillsyn"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	stem = strings.Trim(replacer.Replace(stem), "-")
	if stem == "" {
		return "tillsyn"
	}
	return stem
}
