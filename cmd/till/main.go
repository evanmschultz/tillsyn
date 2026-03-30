package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/fang"
	charmLog "github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/hylla/tillsyn/internal/adapters/auth/autentauth"
	fantasyembed "github.com/hylla/tillsyn/internal/adapters/embeddings/fantasy"
	serveradapter "github.com/hylla/tillsyn/internal/adapters/server"
	servercommon "github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/adapters/storage/sqlite"
	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/config"
	"github.com/hylla/tillsyn/internal/domain"
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

// withInterruptEchoSuppressedFunc wraps long-running terminal commands so Ctrl-C does not render as ^C before clean shutdown logs.
var withInterruptEchoSuppressedFunc = withInterruptEchoSuppressed

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
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

// authCommandOptions stores auth subcommand option values.
type authCommandOptions struct{}

// captureStateCommandOptions stores capture-state flag values.
type captureStateCommandOptions struct {
	projectID string
	scopeType string
	scopeID   string
	view      string
}

// kindListCommandOptions stores kind list flag values.
type kindListCommandOptions struct {
	includeArchived bool
}

// kindUpsertCommandOptions stores kind upsert flag values.
type kindUpsertCommandOptions struct {
	id                  string
	displayName         string
	descriptionMarkdown string
	appliesTo           []string
	allowedParentScopes []string
	payloadSchemaJSON   string
	templateJSON        string
}

// kindAllowlistCommandOptions stores project allowlist flag values.
type kindAllowlistCommandOptions struct {
	projectID string
	kindIDs   []string
}

// templateLibraryListCommandOptions stores template library list flag values.
type templateLibraryListCommandOptions struct {
	scope     string
	projectID string
	status    string
}

// templateLibraryShowCommandOptions stores template library show flag values.
type templateLibraryShowCommandOptions struct {
	libraryID string
}

// templateLibraryUpsertCommandOptions stores template library upsert flag values.
type templateLibraryUpsertCommandOptions struct {
	specJSON string
}

// templateProjectBindCommandOptions stores template project bind flag values.
type templateProjectBindCommandOptions struct {
	projectID string
	libraryID string
}

// templateProjectBindingCommandOptions stores template project binding lookup values.
type templateProjectBindingCommandOptions struct {
	projectID string
}

// templateContractShowCommandOptions stores node-contract lookup values.
type templateContractShowCommandOptions struct {
	nodeID string
}

// leaseListCommandOptions stores capability lease list flag values.
type leaseListCommandOptions struct {
	projectID      string
	scopeType      string
	scopeID        string
	includeRevoked bool
}

// leaseIssueCommandOptions stores capability lease issue flag values.
type leaseIssueCommandOptions struct {
	projectID                 string
	scopeType                 string
	scopeID                   string
	role                      string
	agentName                 string
	agentInstanceID           string
	parentInstanceID          string
	allowEqualScopeDelegation bool
	requestedTTL              time.Duration
	overrideToken             string
}

// leaseHeartbeatCommandOptions stores capability lease heartbeat flag values.
type leaseHeartbeatCommandOptions struct {
	agentInstanceID string
	leaseToken      string
}

// leaseRenewCommandOptions stores capability lease renewal flag values.
type leaseRenewCommandOptions struct {
	agentInstanceID string
	leaseToken      string
	ttl             time.Duration
}

// leaseRevokeCommandOptions stores capability lease revoke flag values.
type leaseRevokeCommandOptions struct {
	agentInstanceID string
	reason          string
}

// leaseRevokeAllCommandOptions stores capability lease revoke-all flag values.
type leaseRevokeAllCommandOptions struct {
	projectID string
	scopeType string
	scopeID   string
	reason    string
}

// handoffCreateCommandOptions stores handoff creation flag values.
type handoffCreateCommandOptions struct {
	projectID       string
	branchID        string
	scopeType       string
	scopeID         string
	sourceRole      string
	targetBranchID  string
	targetScopeType string
	targetScopeID   string
	targetRole      string
	status          string
	summary         string
	nextAction      string
	missingEvidence []string
	relatedRefs     []string
}

// handoffGetCommandOptions stores handoff lookup flag values.
type handoffGetCommandOptions struct {
	handoffID string
}

// handoffListCommandOptions stores handoff list flag values.
type handoffListCommandOptions struct {
	projectID string
	branchID  string
	scopeType string
	scopeID   string
	statuses  []string
	limit     int
}

// handoffUpdateCommandOptions stores handoff update flag values.
type handoffUpdateCommandOptions struct {
	handoffID       string
	status          string
	sourceRole      string
	targetBranchID  string
	targetScopeType string
	targetScopeID   string
	targetRole      string
	summary         string
	nextAction      string
	missingEvidence []string
	relatedRefs     []string
	resolutionNote  string
}

// projectListCommandOptions stores project list flag values.
type projectListCommandOptions struct {
	includeArchived bool
}

// projectCreateCommandOptions stores project create flag values.
type projectCreateCommandOptions struct {
	name              string
	description       string
	kind              string
	templateLibraryID string
	metadataJSON      string
	owner             string
	icon              string
	color             string
	homepage          string
	tags              []string
	standardsMarkdown string
}

// projectShowCommandOptions stores project show flag values.
type projectShowCommandOptions struct {
	projectID       string
	includeArchived bool
}

// projectReadinessCommandOptions stores project collaboration-readiness flag values.
type projectReadinessCommandOptions struct {
	projectID       string
	includeArchived bool
}

// issueSessionCommandOptions stores issue-session flag values.
type issueSessionCommandOptions struct {
	principalID   string
	principalType string
	principalName string
	clientID      string
	clientType    string
	clientName    string
	ttl           time.Duration
}

// requestCreateCommandOptions stores auth request create flag values.
type requestCreateCommandOptions struct {
	path          string
	principalID   string
	principalType string
	principalRole string
	principalName string
	clientID      string
	clientType    string
	clientName    string
	ttl           time.Duration
	timeout       time.Duration
	reason        string
	continuation  string
}

// requestListCommandOptions stores auth request list flag values.
type requestListCommandOptions struct {
	projectID string
	state     string
	limit     int
}

// requestShowCommandOptions stores auth request show flag values.
type requestShowCommandOptions struct {
	requestID string
}

// requestResolveCommandOptions stores auth request resolve flag values.
type requestResolveCommandOptions struct {
	requestID string
	path      string
	ttl       time.Duration
	note      string
}

// sessionListCommandOptions stores auth session list flag values.
type sessionListCommandOptions struct {
	sessionID   string
	projectID   string
	principalID string
	clientID    string
	state       string
	limit       int
}

// sessionValidateCommandOptions stores auth session validate flag values.
type sessionValidateCommandOptions struct {
	sessionID     string
	sessionSecret string
}

// revokeSessionCommandOptions stores revoke-session flag values.
type revokeSessionCommandOptions struct {
	sessionID string
	reason    string
}

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
		devMode: false,
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
	authOpts := authCommandOptions{}
	projectListOpts := projectListCommandOptions{}
	projectCreateOpts := projectCreateCommandOptions{}
	projectShowOpts := projectShowCommandOptions{}
	projectDiscoverOpts := projectReadinessCommandOptions{}
	issueSessionOpts := issueSessionCommandOptions{
		principalType: "user",
		clientID:      "till-mcp-stdio",
		clientType:    "mcp-stdio",
		clientName:    "Till MCP STDIO",
		ttl:           8 * time.Hour,
	}
	requestCreateOpts := requestCreateCommandOptions{
		principalType: "user",
		clientID:      "till-mcp-stdio",
		clientType:    "mcp-stdio",
		clientName:    "Till MCP STDIO",
		ttl:           8 * time.Hour,
		timeout:       15 * time.Minute,
	}
	requestListOpts := requestListCommandOptions{limit: 50}
	requestShowOpts := requestShowCommandOptions{}
	requestApproveOpts := requestResolveCommandOptions{}
	requestDenyOpts := requestResolveCommandOptions{}
	requestCancelOpts := requestResolveCommandOptions{}
	sessionListOpts := sessionListCommandOptions{state: "active", limit: 50}
	sessionValidateOpts := sessionValidateCommandOptions{}
	revokeSessionOpts := revokeSessionCommandOptions{}
	exportOpts := exportCommandOptions{
		outPath:         "-",
		includeArchived: true,
	}
	importOpts := importCommandOptions{}
	captureStateOpts := captureStateCommandOptions{view: string(app.CaptureStateViewSummary)}
	embeddingsStatusOpts := embeddingsStatusCommandOptions{limit: 100}
	embeddingsReindexOpts := embeddingsReindexCommandOptions{
		waitTimeout:      30 * time.Second,
		waitPollInterval: 2 * time.Second,
	}
	kindListOpts := kindListCommandOptions{}
	kindUpsertOpts := kindUpsertCommandOptions{}
	kindAllowlistOpts := kindAllowlistCommandOptions{}
	templateLibraryListOpts := templateLibraryListCommandOptions{}
	templateLibraryShowOpts := templateLibraryShowCommandOptions{}
	templateLibraryUpsertOpts := templateLibraryUpsertCommandOptions{}
	templateProjectBindOpts := templateProjectBindCommandOptions{}
	templateProjectBindingOpts := templateProjectBindingCommandOptions{}
	templateContractShowOpts := templateContractShowCommandOptions{}
	leaseListOpts := leaseListCommandOptions{scopeType: string(domain.CapabilityScopeProject)}
	leaseIssueOpts := leaseIssueCommandOptions{scopeType: string(domain.CapabilityScopeProject), role: string(domain.CapabilityRoleBuilder), requestedTTL: 8 * time.Hour}
	leaseHeartbeatOpts := leaseHeartbeatCommandOptions{}
	leaseRenewOpts := leaseRenewCommandOptions{}
	leaseRevokeOpts := leaseRevokeCommandOptions{}
	leaseRevokeAllOpts := leaseRevokeAllCommandOptions{scopeType: string(domain.CapabilityScopeProject)}
	handoffCreateOpts := handoffCreateCommandOptions{scopeType: string(domain.ScopeLevelProject)}
	handoffGetOpts := handoffGetCommandOptions{}
	handoffListOpts := handoffListCommandOptions{scopeType: string(domain.ScopeLevelProject), limit: 50}
	handoffUpdateOpts := handoffUpdateCommandOptions{}

	runFlow := func(ctx context.Context, command string) error {
		return executeCommandFlow(ctx, command, rootOpts, serveOpts, mcpOpts, authOpts, projectListOpts, projectCreateOpts, projectShowOpts, projectDiscoverOpts, captureStateOpts, embeddingsStatusOpts, embeddingsReindexOpts, kindListOpts, kindUpsertOpts, kindAllowlistOpts, templateLibraryListOpts, templateLibraryShowOpts, templateLibraryUpsertOpts, templateProjectBindOpts, templateProjectBindingOpts, templateContractShowOpts, leaseListOpts, leaseIssueOpts, leaseHeartbeatOpts, leaseRenewOpts, leaseRevokeOpts, leaseRevokeAllOpts, handoffCreateOpts, handoffGetOpts, handoffListOpts, handoffUpdateOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
	}

	rootCmd := &cobra.Command{
		Use:           "till",
		Short:         "Local-first planning TUI with stdio MCP and HTTP adapters",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "")
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
		Short: "Start HTTP API and streamable HTTP MCP endpoints",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "serve")
		},
	}
	serveCmd.Flags().StringVar(&serveOpts.httpBind, "http", serveOpts.httpBind, "HTTP listen address")
	serveCmd.Flags().StringVar(&serveOpts.apiEndpoint, "api-endpoint", serveOpts.apiEndpoint, "HTTP API base endpoint")
	serveCmd.Flags().StringVar(&serveOpts.mcpEndpoint, "mcp-endpoint", serveOpts.mcpEndpoint, "MCP streamable HTTP endpoint")

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start raw MCP over stdio for local integrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "mcp")
		},
	}

	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage dogfood auth requests and autent-backed sessions",
		Long: strings.TrimSpace(`
Manage dogfood auth requests and autent-backed sessions.

Use request create to raise one scoped approval request, then inspect or
resolve it with request list, show, approve, deny, or cancel. Use session list,
validate, and revoke to inspect or rotate approved caller sessions. The low-level
issue-session seam remains available for direct local testing, but request/session
subcommands are the primary operator UX.

Omit --project-id on list commands to inspect global inventory across all
projects. Add --project-id to narrow requests or sessions to one project.
Request paths may be:
- project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]]
- projects/<project-id>,<project-id>...
- global
Only orchestrators may request projects/... or global scope.
`),
		Example: strings.Join([]string{
			"  till auth request create --path project/p1 --principal-id review-agent --principal-type agent --principal-role builder --client-id till-mcp-stdio --client-type mcp-stdio --reason \"local MCP review\"",
			"  till auth request create --path projects/p1,p2 --principal-id orchestration-agent --principal-type agent --principal-role orchestrator --client-id till-mcp-stdio --client-type mcp-stdio --reason \"multi-project orchestration\"",
			"  till auth request list --project-id p1 --state pending",
			"  till auth request list --state approved",
			"  till auth request approve --request-id req-123 --note \"approved for dogfood\"",
			"  till auth session list --state active",
			"  till auth session revoke --session-id sess-123 --reason operator_revoke",
		}, "\n"),
		Args: cobra.NoArgs,
	}

	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "List, create, inspect, and bridge projects into collaboration readiness",
		Long: strings.TrimSpace(`
List projects for discovery, create new projects with reasonable metadata
defaults, inspect one project in a human-readable detail view, or bridge one
project into auth/session/lease/handoff setup with a readiness summary.

Next step: use till project list to find an id, till project discover
--project-id <project-id> to see the next collaboration step, or till project
create to add one.
`),
		Example: strings.Join([]string{
			"  till project list",
			"  till project create --name Inbox --description \"Local execution inbox\"",
			"  till project show --project-id p1",
			"  till project discover --project-id p1",
		}, "\n"),
		Args: cobra.NoArgs,
	}
	projectListCmd := &cobra.Command{
		Use:   "list",
		Short: "List projects in discovery order",
		Long: strings.TrimSpace(`
List projects in a human-readable table with names first and ids visible for
copy/paste into scoped commands.

Next step: use till project discover --project-id <project-id> to inspect one
project and see the collaboration-readiness bridge, or till project create to
add a new one.
`),
		Example: strings.Join([]string{
			"  till project list",
			"  till project list --include-archived",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "project.list")
		},
	}
	projectListCmd.Flags().BoolVar(&projectListOpts.includeArchived, "include-archived", false, "Include archived projects")
	projectCreateCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create one project",
		Long: strings.TrimSpace(`
Create one project with a required name, optional description, optional kind
override, optional approved global template library binding, and optional
metadata defaults from flags or --metadata-json.

The name may be passed either as --name or as one positional argument.

Next step: use till project list to confirm the new record, or till project
discover --project-id <project-id> to inspect the collaboration-readiness
bridge after creation.
`),
		Example: strings.Join([]string{
			"  till project create --name Inbox --description \"Local execution inbox\" --owner \"Platform\" --tag dogfood",
			"  till project create Inbox",
			"  till project create --name \"Go Migration\" --kind project --homepage https://example.invalid",
			"  till project create --name \"Go Service\" --template-library-id go-defaults",
		}, "\n"),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := resolveProjectNameInput(projectCreateOpts.name, args)
			if err != nil {
				return err
			}
			projectCreateOpts.name = name
			return runFlow(cmd.Context(), "project.create")
		},
	}
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.name, "name", "", "Project name")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.description, "description", "", "Optional project description")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.kind, "kind", "", "Optional project kind")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.templateLibraryID, "template-library-id", "", "Optional approved global template library to bind during project creation")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.metadataJSON, "metadata-json", "", "Optional project metadata JSON")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.owner, "owner", "", "Optional project owner")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.icon, "icon", "", "Optional project icon")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.color, "color", "", "Optional project color")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.homepage, "homepage", "", "Optional project homepage")
	projectCreateCmd.Flags().StringSliceVar(&projectCreateOpts.tags, "tag", nil, "Optional project tag")
	projectCreateCmd.Flags().StringVar(&projectCreateOpts.standardsMarkdown, "standards-markdown", "", "Optional project standards markdown")
	projectShowCmd := &cobra.Command{
		Use:   "show [project-id]",
		Short: "Show one project",
		Long: strings.TrimSpace(`
Show one project in a readable detail view.

If you do not know the id yet, run till project list first to discover it.
The project id may be passed either as --project-id or as one positional
argument.

Next step: after inspecting the project, use till project discover --project-id
<project-id> to see the auth/session/lease/handoff bridge, or return to till
project list to choose another record.
`),
		Example: strings.Join([]string{
			"  till project show --project-id p1",
			"  till project show p1",
		}, "\n"),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID, err := resolveProjectIDInput("project show", projectShowOpts.projectID, args)
			if err != nil {
				return err
			}
			projectShowOpts.projectID = projectID
			return runFlow(cmd.Context(), "project.show")
		},
	}
	projectShowCmd.Flags().StringVar(&projectShowOpts.projectID, "project-id", "", "Project identifier")
	projectShowCmd.Flags().BoolVar(&projectShowOpts.includeArchived, "include-archived", false, "Include archived projects")
	projectDiscoverCmd := &cobra.Command{
		Use:   "discover [project-id]",
		Short: "Show one project collaboration-readiness summary",
		Long: strings.TrimSpace(`
Show one project with a collaboration-readiness bridge that points the operator
at the next auth, session, lease, or handoff step.

The project id may be passed either as --project-id or as one positional
argument.

Next step: after reading the readiness summary, follow the recommended command
in order rather than relying on remembered setup steps.
`),
		Example: strings.Join([]string{
			"  till project discover --project-id p1",
			"  till project discover p1",
		}, "\n"),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID, err := resolveProjectIDInput("project discover", projectDiscoverOpts.projectID, args)
			if err != nil {
				return err
			}
			projectDiscoverOpts.projectID = projectID
			return runFlow(cmd.Context(), "project.discover")
		},
	}
	projectDiscoverCmd.Flags().StringVar(&projectDiscoverOpts.projectID, "project-id", "", "Project identifier")
	projectDiscoverCmd.Flags().BoolVar(&projectDiscoverOpts.includeArchived, "include-archived", false, "Include archived projects")
	projectCmd.AddCommand(projectListCmd, projectCreateCmd, projectShowCmd, projectDiscoverCmd)

	embeddingsCmd := &cobra.Command{
		Use:   "embeddings",
		Short: "Inspect and operate the background embeddings lifecycle",
		Long: strings.TrimSpace(`
Inspect persistent embeddings lifecycle state, view pending/failed/stale rows,
and trigger explicit backfill or reindex operations without blocking normal
task mutations.
`),
		Args: cobra.NoArgs,
	}
	embeddingsStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show embeddings lifecycle health and row inventory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "embeddings.status")
		},
	}
	embeddingsStatusCmd.Flags().StringVar(&embeddingsStatusOpts.projectID, "project-id", "", "Project identifier")
	embeddingsStatusCmd.Flags().BoolVar(&embeddingsStatusOpts.crossProject, "cross-project", false, "Inspect embeddings state across all projects")
	embeddingsStatusCmd.Flags().BoolVar(&embeddingsStatusOpts.includeArchived, "include-archived", false, "Include archived projects when resolving scope")
	embeddingsStatusCmd.Flags().StringSliceVar(&embeddingsStatusOpts.statuses, "status", nil, "Optional lifecycle status filter (pending|running|ready|failed|stale)")
	embeddingsStatusCmd.Flags().IntVar(&embeddingsStatusOpts.limit, "limit", embeddingsStatusOpts.limit, "Maximum lifecycle rows to print")
	embeddingsReindexCmd := &cobra.Command{
		Use:   "reindex",
		Short: "Enqueue or force one explicit embeddings backfill/reindex",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "embeddings.reindex")
		},
	}
	embeddingsReindexCmd.Flags().StringVar(&embeddingsReindexOpts.projectID, "project-id", "", "Project identifier")
	embeddingsReindexCmd.Flags().BoolVar(&embeddingsReindexOpts.crossProject, "cross-project", false, "Reindex embeddings across all projects")
	embeddingsReindexCmd.Flags().BoolVar(&embeddingsReindexOpts.includeArchived, "include-archived", false, "Include archived projects and work items in the reindex scope")
	embeddingsReindexCmd.Flags().BoolVar(&embeddingsReindexOpts.force, "force", false, "Force ready rows back into the queue even when hashes already match")
	embeddingsReindexCmd.Flags().BoolVar(&embeddingsReindexOpts.wait, "wait", false, "Wait for the requested scope to reach a steady lifecycle state")
	embeddingsReindexCmd.Flags().DurationVar(&embeddingsReindexOpts.waitTimeout, "wait-timeout", embeddingsReindexOpts.waitTimeout, "Maximum time to wait for steady state when --wait is set")
	embeddingsReindexCmd.Flags().DurationVar(&embeddingsReindexOpts.waitPollInterval, "wait-poll-interval", embeddingsReindexOpts.waitPollInterval, "Polling interval while waiting for steady state")
	embeddingsCmd.AddCommand(embeddingsStatusCmd, embeddingsReindexCmd)

	captureStateCmd := &cobra.Command{
		Use:   "capture-state",
		Short: "Capture one summary-first recovery snapshot",
		Long: strings.TrimSpace(`
Capture a deterministic, summary-first recovery snapshot for one project or
scope. The result is the same capture_state bundle exposed through MCP and HTTP.

Next step: inspect the returned JSON for the scope path, state hash, work
overview, attention overview, and follow-up pointers.
`),
		Example: strings.Join([]string{
			"  till capture-state --project-id p1",
			"  till capture-state --project-id p1 --scope-type branch --scope-id branch-1 --view full",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "capture-state")
		},
	}
	captureStateCmd.Flags().StringVar(&captureStateOpts.projectID, "project-id", "", "Project identifier")
	captureStateCmd.Flags().StringVar(&captureStateOpts.scopeType, "scope-type", captureStateOpts.scopeType, "Scope type (project|branch|phase|task|subtask)")
	captureStateCmd.Flags().StringVar(&captureStateOpts.scopeID, "scope-id", "", "Optional scope identifier")
	captureStateCmd.Flags().StringVar(&captureStateOpts.view, "view", captureStateOpts.view, "Capture state view (summary|full)")

	kindCmd := &cobra.Command{
		Use:   "kind",
		Short: "Inspect and update kind definitions and allowlists",
		Long: strings.TrimSpace(`
Inspect kind definitions and project allowlists. Template-library workflow
contracts now live under the dedicated template commands rather than the kind
registry path.
`),
		Args: cobra.NoArgs,
	}
	kindListCmd := &cobra.Command{
		Use:   "list",
		Short: "List kind definitions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "kind.list")
		},
	}
	kindListCmd.Flags().BoolVar(&kindListOpts.includeArchived, "include-archived", false, "Include archived kind definitions")
	kindUpsertCmd := &cobra.Command{
		Use:   "upsert",
		Short: "Create or update one kind definition",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "kind.upsert")
		},
	}
	kindUpsertCmd.Flags().StringVar(&kindUpsertOpts.id, "id", "", "Kind identifier")
	kindUpsertCmd.Flags().StringVar(&kindUpsertOpts.displayName, "display-name", "", "Display name")
	kindUpsertCmd.Flags().StringVar(&kindUpsertOpts.descriptionMarkdown, "description-markdown", "", "Description markdown")
	kindUpsertCmd.Flags().StringSliceVar(&kindUpsertOpts.appliesTo, "applies-to", nil, "Allowed applies-to values")
	kindUpsertCmd.Flags().StringSliceVar(&kindUpsertOpts.allowedParentScopes, "allowed-parent-scopes", nil, "Allowed parent scopes")
	kindUpsertCmd.Flags().StringVar(&kindUpsertOpts.payloadSchemaJSON, "payload-schema-json", "", "Optional payload schema JSON")
	kindUpsertCmd.Flags().StringVar(&kindUpsertOpts.templateJSON, "template-json", "", "Optional kind template JSON")
	mustMarkFlagHidden(kindUpsertCmd, "template-json")
	mustMarkFlagRequired(kindUpsertCmd, "id")
	mustMarkFlagRequired(kindUpsertCmd, "display-name")
	mustMarkFlagRequired(kindUpsertCmd, "applies-to")
	kindAllowlistCmd := &cobra.Command{
		Use:   "allowlist",
		Short: "Inspect and update a project's explicit kind allowlist",
		Args:  cobra.NoArgs,
	}
	kindAllowlistListCmd := &cobra.Command{
		Use:   "list",
		Short: "List one project's allowed kind ids",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "kind.allowlist.list")
		},
	}
	kindAllowlistListCmd.Flags().StringVar(&kindAllowlistOpts.projectID, "project-id", "", "Project identifier")
	kindAllowlistSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Replace one project's allowed kind ids",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "kind.allowlist.set")
		},
	}
	kindAllowlistSetCmd.Flags().StringVar(&kindAllowlistOpts.projectID, "project-id", "", "Project identifier")
	kindAllowlistSetCmd.Flags().StringSliceVar(&kindAllowlistOpts.kindIDs, "kind-id", nil, "Allowed kind identifier")
	kindAllowlistCmd.AddCommand(kindAllowlistListCmd, kindAllowlistSetCmd)
	kindCmd.AddCommand(kindListCmd, kindUpsertCmd, kindAllowlistCmd)

	templateCmd := &cobra.Command{
		Use:   "template",
		Short: "Inspect and bind SQLite-backed template libraries",
		Long: strings.TrimSpace(`
Inspect SQLite-backed template libraries, bind approved libraries to projects,
and inspect generated node-contract snapshots. JSON is the stable CLI/MCP
transport for template-library specs while SQLite remains the source of truth.
`),
		Args: cobra.NoArgs,
	}
	templateLibraryCmd := &cobra.Command{
		Use:   "library",
		Short: "Inspect and upsert template libraries",
		Args:  cobra.NoArgs,
	}
	templateLibraryListCmd := &cobra.Command{
		Use:   "list",
		Short: "List template libraries",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "template.library.list")
		},
	}
	templateLibraryListCmd.Flags().StringVar(&templateLibraryListOpts.scope, "scope", "", "Optional scope filter (global|project|draft)")
	templateLibraryListCmd.Flags().StringVar(&templateLibraryListOpts.projectID, "project-id", "", "Optional project identifier filter")
	templateLibraryListCmd.Flags().StringVar(&templateLibraryListOpts.status, "status", "", "Optional status filter (draft|approved|archived)")
	templateLibraryShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show one template library",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "template.library.show")
		},
	}
	templateLibraryShowCmd.Flags().StringVar(&templateLibraryShowOpts.libraryID, "library-id", "", "Template library identifier")
	templateLibraryUpsertCmd := &cobra.Command{
		Use:   "upsert",
		Short: "Create or update one template library from JSON",
		Long: strings.TrimSpace(`
Create or update one template library from a JSON object. This is a temporary
operator seam; SQLite remains the source of truth and richer TUI authoring is
planned separately.
`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "template.library.upsert")
		},
	}
	templateLibraryUpsertCmd.Flags().StringVar(&templateLibraryUpsertOpts.specJSON, "spec-json", "", "Template library JSON object")
	mustMarkFlagRequired(templateLibraryUpsertCmd, "spec-json")
	templateLibraryCmd.AddCommand(templateLibraryListCmd, templateLibraryShowCmd, templateLibraryUpsertCmd)

	templateProjectCmd := &cobra.Command{
		Use:   "project",
		Short: "Bind projects to template libraries",
		Args:  cobra.NoArgs,
	}
	templateProjectBindCmd := &cobra.Command{
		Use:   "bind",
		Short: "Bind one project to one approved template library",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "template.project.bind")
		},
	}
	templateProjectBindCmd.Flags().StringVar(&templateProjectBindOpts.projectID, "project-id", "", "Project identifier")
	templateProjectBindCmd.Flags().StringVar(&templateProjectBindOpts.libraryID, "library-id", "", "Template library identifier")
	templateProjectBindingCmd := &cobra.Command{
		Use:   "binding",
		Short: "Show one project's active template-library binding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "template.project.binding")
		},
	}
	templateProjectBindingCmd.Flags().StringVar(&templateProjectBindingOpts.projectID, "project-id", "", "Project identifier")
	templateProjectCmd.AddCommand(templateProjectBindCmd, templateProjectBindingCmd)

	templateContractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Inspect generated node-contract snapshots",
		Args:  cobra.NoArgs,
	}
	templateContractShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show one generated node-contract snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "template.contract.show")
		},
	}
	templateContractShowCmd.Flags().StringVar(&templateContractShowOpts.nodeID, "node-id", "", "Generated node identifier")
	templateContractCmd.AddCommand(templateContractShowCmd)
	templateCmd.AddCommand(templateLibraryCmd, templateProjectCmd, templateContractCmd)

	leaseCmd := &cobra.Command{
		Use:   "lease",
		Short: "Inspect and manage capability leases",
		Long: strings.TrimSpace(`
Inspect scoped capability leases, issue new ones, and rotate or revoke them.
This is the CLI surface for orchestrator and agent recovery state.
`),
		Args: cobra.NoArgs,
	}
	leaseListCmd := &cobra.Command{
		Use:   "list",
		Short: "List capability leases for one project scope",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "lease.list")
		},
	}
	leaseListCmd.Flags().StringVar(&leaseListOpts.projectID, "project-id", "", "Project identifier")
	leaseListCmd.Flags().StringVar(&leaseListOpts.scopeType, "scope-type", leaseListOpts.scopeType, "Scope type (project|branch|phase|task|subtask)")
	leaseListCmd.Flags().StringVar(&leaseListOpts.scopeID, "scope-id", "", "Optional scope identifier")
	leaseListCmd.Flags().BoolVar(&leaseListOpts.includeRevoked, "include-revoked", false, "Include revoked leases")
	leaseIssueCmd := &cobra.Command{
		Use:   "issue",
		Short: "Issue one capability lease",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "lease.issue")
		},
	}
	leaseIssueCmd.Flags().StringVar(&leaseIssueOpts.projectID, "project-id", "", "Project identifier")
	leaseIssueCmd.Flags().StringVar(&leaseIssueOpts.scopeType, "scope-type", leaseIssueOpts.scopeType, "Scope type (project|branch|phase|task|subtask)")
	leaseIssueCmd.Flags().StringVar(&leaseIssueOpts.scopeID, "scope-id", "", "Optional scope identifier")
	leaseIssueCmd.Flags().StringVar(&leaseIssueOpts.role, "role", leaseIssueOpts.role, "Lease role (orchestrator|builder|qa)")
	leaseIssueCmd.Flags().StringVar(&leaseIssueOpts.agentName, "agent-name", "", "Agent display name")
	leaseIssueCmd.Flags().StringVar(&leaseIssueOpts.agentInstanceID, "agent-instance-id", "", "Optional agent instance identifier")
	leaseIssueCmd.Flags().StringVar(&leaseIssueOpts.parentInstanceID, "parent-instance-id", "", "Optional parent lease instance identifier")
	leaseIssueCmd.Flags().BoolVar(&leaseIssueOpts.allowEqualScopeDelegation, "allow-equal-scope-delegation", false, "Allow equal-scope delegation")
	leaseIssueCmd.Flags().DurationVar(&leaseIssueOpts.requestedTTL, "requested-ttl", leaseIssueOpts.requestedTTL, "Requested lease TTL")
	leaseIssueCmd.Flags().StringVar(&leaseIssueOpts.overrideToken, "override-token", "", "Optional override token")
	leaseHeartbeatCmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Refresh one capability lease heartbeat",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "lease.heartbeat")
		},
	}
	leaseHeartbeatCmd.Flags().StringVar(&leaseHeartbeatOpts.agentInstanceID, "agent-instance-id", "", "Agent instance identifier")
	leaseHeartbeatCmd.Flags().StringVar(&leaseHeartbeatOpts.leaseToken, "lease-token", "", "Lease token")
	mustMarkFlagRequired(leaseHeartbeatCmd, "agent-instance-id")
	mustMarkFlagRequired(leaseHeartbeatCmd, "lease-token")
	leaseRenewCmd := &cobra.Command{
		Use:   "renew",
		Short: "Renew one capability lease",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "lease.renew")
		},
	}
	leaseRenewCmd.Flags().StringVar(&leaseRenewOpts.agentInstanceID, "agent-instance-id", "", "Agent instance identifier")
	leaseRenewCmd.Flags().StringVar(&leaseRenewOpts.leaseToken, "lease-token", "", "Lease token")
	leaseRenewCmd.Flags().DurationVar(&leaseRenewOpts.ttl, "ttl", 0, "Renewal TTL")
	mustMarkFlagRequired(leaseRenewCmd, "agent-instance-id")
	mustMarkFlagRequired(leaseRenewCmd, "lease-token")
	leaseRevokeCmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke one capability lease",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "lease.revoke")
		},
	}
	leaseRevokeCmd.Flags().StringVar(&leaseRevokeOpts.agentInstanceID, "agent-instance-id", "", "Agent instance identifier")
	leaseRevokeCmd.Flags().StringVar(&leaseRevokeOpts.reason, "reason", "", "Revocation reason")
	mustMarkFlagRequired(leaseRevokeCmd, "agent-instance-id")
	leaseRevokeAllCmd := &cobra.Command{
		Use:   "revoke-all",
		Short: "Revoke every lease in one project scope",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "lease.revoke-all")
		},
	}
	leaseRevokeAllCmd.Flags().StringVar(&leaseRevokeAllOpts.projectID, "project-id", "", "Project identifier")
	leaseRevokeAllCmd.Flags().StringVar(&leaseRevokeAllOpts.scopeType, "scope-type", leaseRevokeAllOpts.scopeType, "Scope type (project|branch|phase|task|subtask)")
	leaseRevokeAllCmd.Flags().StringVar(&leaseRevokeAllOpts.scopeID, "scope-id", "", "Optional scope identifier")
	leaseRevokeAllCmd.Flags().StringVar(&leaseRevokeAllOpts.reason, "reason", "", "Revocation reason")
	leaseCmd.AddCommand(leaseListCmd, leaseIssueCmd, leaseHeartbeatCmd, leaseRenewCmd, leaseRevokeCmd, leaseRevokeAllCmd)

	handoffCmd := &cobra.Command{
		Use:   "handoff",
		Short: "Inspect and manage durable agent handoffs",
		Long: strings.TrimSpace(`
Inspect and manage durable, structured handoffs that keep humans and agents
aligned across planning, execution, and recovery.
`),
		Args: cobra.NoArgs,
	}
	handoffCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create one durable handoff",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "handoff.create")
		},
	}
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.projectID, "project-id", "", "Project identifier")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.branchID, "branch-id", "", "Optional branch identifier")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.scopeType, "scope-type", handoffCreateOpts.scopeType, "Source scope type (project|branch|phase|task|subtask)")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.scopeID, "scope-id", "", "Optional source scope identifier")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.sourceRole, "source-role", "", "Optional source role")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.targetBranchID, "target-branch-id", "", "Optional target branch identifier")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.targetScopeType, "target-scope-type", "", "Optional target scope type")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.targetScopeID, "target-scope-id", "", "Optional target scope identifier")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.targetRole, "target-role", "", "Optional target role")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.status, "status", "", "Optional handoff status")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.summary, "summary", "", "Handoff summary")
	handoffCreateCmd.Flags().StringVar(&handoffCreateOpts.nextAction, "next-action", "", "Optional next action")
	handoffCreateCmd.Flags().StringSliceVar(&handoffCreateOpts.missingEvidence, "missing-evidence", nil, "Missing evidence item")
	handoffCreateCmd.Flags().StringSliceVar(&handoffCreateOpts.relatedRefs, "related-ref", nil, "Related reference")
	handoffGetCmd := &cobra.Command{
		Use:   "get",
		Short: "Show one durable handoff",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "handoff.get")
		},
	}
	handoffGetCmd.Flags().StringVar(&handoffGetOpts.handoffID, "handoff-id", "", "Handoff identifier")
	mustMarkFlagRequired(handoffGetCmd, "handoff-id")
	handoffListCmd := &cobra.Command{
		Use:   "list",
		Short: "List durable handoffs for one scope",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "handoff.list")
		},
	}
	handoffListCmd.Flags().StringVar(&handoffListOpts.projectID, "project-id", "", "Project identifier")
	handoffListCmd.Flags().StringVar(&handoffListOpts.branchID, "branch-id", "", "Optional branch identifier")
	handoffListCmd.Flags().StringVar(&handoffListOpts.scopeType, "scope-type", handoffListOpts.scopeType, "Scope type (project|branch|phase|task|subtask)")
	handoffListCmd.Flags().StringVar(&handoffListOpts.scopeID, "scope-id", "", "Optional scope identifier")
	handoffListCmd.Flags().StringSliceVar(&handoffListOpts.statuses, "status", nil, "Optional handoff status filter")
	handoffListCmd.Flags().IntVar(&handoffListOpts.limit, "limit", handoffListOpts.limit, "Maximum rows to return")
	handoffUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update one durable handoff",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "handoff.update")
		},
	}
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.handoffID, "handoff-id", "", "Handoff identifier")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.status, "status", "", "Optional handoff status")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.sourceRole, "source-role", "", "Optional source role")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.targetBranchID, "target-branch-id", "", "Optional target branch identifier")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.targetScopeType, "target-scope-type", "", "Optional target scope type")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.targetScopeID, "target-scope-id", "", "Optional target scope identifier")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.targetRole, "target-role", "", "Optional target role")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.summary, "summary", "", "Handoff summary")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.nextAction, "next-action", "", "Optional next action")
	handoffUpdateCmd.Flags().StringSliceVar(&handoffUpdateOpts.missingEvidence, "missing-evidence", nil, "Missing evidence item")
	handoffUpdateCmd.Flags().StringSliceVar(&handoffUpdateOpts.relatedRefs, "related-ref", nil, "Related reference")
	handoffUpdateCmd.Flags().StringVar(&handoffUpdateOpts.resolutionNote, "resolution-note", "", "Optional resolution note")
	mustMarkFlagRequired(handoffUpdateCmd, "handoff-id")
	mustMarkFlagRequired(handoffUpdateCmd, "summary")
	handoffCmd.AddCommand(handoffCreateCmd, handoffGetCmd, handoffListCmd, handoffUpdateCmd)

	requestCmd := &cobra.Command{
		Use:   "request",
		Short: "Create, inspect, and resolve persisted auth requests",
		Long: strings.TrimSpace(`
Create and resolve persisted auth requests tied to one explicit scope path.
Supported --path forms are:
- project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]]
- projects/<project-id>,<project-id>...
- global

Only orchestrators may request projects/... or global scope.

After create, use request show or request list to track the pending request, and
use approve, deny, or cancel to move it to a terminal state. Omit --project-id
on request list for global inventory, or add it to focus on one project.
`),
		Example: strings.Join([]string{
			"  till auth request create --path project/p1 --principal-id review-agent --principal-type agent --client-id till-mcp-stdio --client-type mcp-stdio --reason \"manual MCP review\"",
			"  till auth request list --project-id p1 --state pending",
			"  till auth request show --request-id req-123",
		}, "\n"),
		Args: cobra.NoArgs,
	}

	requestCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create one persisted auth request",
		Long: strings.TrimSpace(`
Create one persisted auth request for a specific principal, client, and
scope path. The request remains pending until it is approved, denied,
canceled, or times out. Supported --path forms are:
- project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]]
- projects/<project-id>,<project-id>...
- global

Optional --principal-role distinguishes agent-shaped requests:
- builder is the default agent role,
- orchestrator must be set explicitly when broader orchestration access is requested.

Optional --continuation-json stores client resume metadata so the requesting
surface can continue cleanly after approval. Include a requester-owned
resume_token when the requesting MCP client will later claim the result.

Next step: use till auth request show --request-id req-123 or till auth
request list --state pending to inspect the stored request, then resolve it with
approve, deny, or cancel.
`),
		Example: strings.Join([]string{
			"  till auth request create --path project/p1 --principal-id review-agent --principal-type agent --principal-role builder --client-id till-mcp-stdio --client-type mcp-stdio --reason \"manual MCP review\"",
			"  till auth request create --path project/p1 --principal-id qa-agent --principal-type agent --principal-role qa --client-id till-mcp-stdio --client-type mcp-stdio --reason \"qa review\"",
			"  till auth request create --path project/p1 --principal-id orchestration-agent --principal-type agent --principal-role orchestrator --client-id till-mcp-stdio --client-type mcp-stdio --reason \"orchestrator review\"",
			"  till auth request create --path projects/p1,p2 --principal-id orchestration-agent --principal-type agent --principal-role orchestrator --client-id till-mcp-stdio --client-type mcp-stdio --reason \"multi-project orchestration\"",
			"  till auth request create --path global --principal-id orchestration-agent --principal-type agent --principal-role orchestrator --client-id till-mcp-stdio --client-type mcp-stdio --reason \"general orchestration\"",
			"  till auth request create --path project/p1/branch/branch-1/phase/phase-a --principal-id review-user --principal-type user --client-id till-tui --client-type tui --ttl 2h --timeout 30m --reason \"branch-focused review\"",
			"  till auth request create --path project/p1 --principal-id review-agent --principal-type agent --principal-role builder --client-id till-mcp-stdio --client-type mcp-stdio --reason \"resume after approval\" --continuation-json '{\"resume_token\":\"resume-123\",\"resume_tool\":\"till.claim_auth_request\",\"resume_path\":\"project/p1\"}'",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.request.create")
		},
	}
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.path, "path", "", "Required auth scope path: project/<project-id>[/branch/<branch-id>[/phase/<phase-id>...]] | projects/<project-id>,<project-id>... | global")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.principalID, "principal-id", "", "Principal identifier")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.principalType, "principal-type", requestCreateOpts.principalType, "Principal type (user|agent|service)")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.principalRole, "principal-role", "", "Optional agent role (orchestrator|builder|qa)")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.principalName, "principal-name", "", "Optional principal display name")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.clientID, "client-id", requestCreateOpts.clientID, "Client identifier")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.clientType, "client-type", requestCreateOpts.clientType, "Client type")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.clientName, "client-name", requestCreateOpts.clientName, "Optional client display name")
	requestCreateCmd.Flags().DurationVar(&requestCreateOpts.ttl, "ttl", requestCreateOpts.ttl, "Requested approved-session lifetime")
	requestCreateCmd.Flags().DurationVar(&requestCreateOpts.timeout, "timeout", requestCreateOpts.timeout, "How long the request stays pending before timing out")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.reason, "reason", "", "Human-readable approval reason")
	requestCreateCmd.Flags().StringVar(&requestCreateOpts.continuation, "continuation-json", "", "Optional JSON object string with client continuation metadata for post-approval resume")
	mustMarkFlagRequired(requestCreateCmd, "path")
	mustMarkFlagRequired(requestCreateCmd, "principal-id")
	mustMarkFlagRequired(requestCreateCmd, "client-id")
	mustMarkFlagRequired(requestCreateCmd, "reason")

	requestListCmd := &cobra.Command{
		Use:   "list",
		Short: "List persisted auth requests",
		Long: strings.TrimSpace(`
List persisted auth requests in deterministic newest-first order.

Next step: use till auth request show --request-id req-123 to inspect one
row in detail, then resolve it with approve, deny, or cancel. Omit --project-id
for global inventory or add it to focus on one project.
`),
		Example: strings.Join([]string{
			"  till auth request list",
			"  till auth request list --project-id p1 --state pending",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.request.list")
		},
	}
	requestListCmd.Flags().StringVar(&requestListOpts.projectID, "project-id", "", "Optional project identifier filter")
	requestListCmd.Flags().StringVar(&requestListOpts.state, "state", "", "Optional state filter (pending|approved|denied|canceled|expired)")
	requestListCmd.Flags().IntVar(&requestListOpts.limit, "limit", requestListOpts.limit, "Maximum rows to return")

	requestShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show one persisted auth request",
		Long: strings.TrimSpace(`
Show one persisted auth request by id.

Approved requests continue to show issued_session_id for audit, but they do not
re-print the bearer secret. The requester should resume through the original
claim/resume flow instead of re-reading the secret from inventory.

Next step: if the request is still pending, resolve it with till auth request
approve, deny, or cancel. If it is already approved, validate or revoke the
issued session through till auth session subcommands.
`),
		Example: "  till auth request show --request-id req-123",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.request.show")
		},
	}
	requestShowCmd.Flags().StringVar(&requestShowOpts.requestID, "request-id", "", "Auth request identifier")
	mustMarkFlagRequired(requestShowCmd, "request-id")

	requestApproveCmd := &cobra.Command{
		Use:   "approve",
		Short: "Approve one pending auth request and issue a session",
		Long: strings.TrimSpace(`
Approve one pending auth request and issue a usable autent session for the
requested principal, client, path, and lifetime.

Optional --path and --ttl overrides let the operator narrow or adjust the
approved scope and session lifetime before the session is issued. --path may
narrow within the requested scope using the same forms as request create.

Next step: if the requester supplied continuation metadata with a resume_token,
the requesting MCP client should claim the result through till.claim_auth_request
using the original request_id and resume_token. Shell handoff of session_id and
session_secret remains a fallback only. Use till auth request show --request-id
req-123 to inspect the approved record, including requested vs approved path
and TTL fields, without re-printing the secret.
`),
		Example: strings.Join([]string{
			"  till auth request approve --request-id req-123 --note \"approved for dogfood\"",
			"  till auth request approve --request-id req-123 --path project/p1/branch/branch-1 --ttl 2h --note \"limited branch review\"",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.request.approve")
		},
	}
	requestApproveCmd.Flags().StringVar(&requestApproveOpts.requestID, "request-id", "", "Auth request identifier")
	requestApproveCmd.Flags().StringVar(&requestApproveOpts.path, "path", "", "Optional approved scope override using project/... | projects/... | global within the requested scope")
	requestApproveCmd.Flags().DurationVar(&requestApproveOpts.ttl, "ttl", 0, "Optional approved session lifetime override")
	requestApproveCmd.Flags().StringVar(&requestApproveOpts.note, "note", "", "Optional operator note")
	mustMarkFlagRequired(requestApproveCmd, "request-id")

	requestDenyCmd := &cobra.Command{
		Use:   "deny",
		Short: "Deny one pending auth request",
		Long: strings.TrimSpace(`
Deny one pending auth request and record an operator-visible note.

Next step: use till auth request show --request-id req-123 to verify the
stored terminal state.
`),
		Example: "  till auth request deny --request-id req-123 --note \"outside current scope\"",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.request.deny")
		},
	}
	requestDenyCmd.Flags().StringVar(&requestDenyOpts.requestID, "request-id", "", "Auth request identifier")
	requestDenyCmd.Flags().StringVar(&requestDenyOpts.note, "note", "", "Optional operator note")
	mustMarkFlagRequired(requestDenyCmd, "request-id")

	requestCancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel one pending auth request",
		Long: strings.TrimSpace(`
Cancel one pending auth request before it is approved or denied.

Next step: use till auth request show --request-id req-123 to verify the
stored terminal state.
`),
		Example: "  till auth request cancel --request-id req-123 --note \"superseded by a new request\"",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.request.cancel")
		},
	}
	requestCancelCmd.Flags().StringVar(&requestCancelOpts.requestID, "request-id", "", "Auth request identifier")
	requestCancelCmd.Flags().StringVar(&requestCancelOpts.note, "note", "", "Optional operator note")
	mustMarkFlagRequired(requestCancelCmd, "request-id")
	requestCmd.AddCommand(requestCreateCmd, requestListCmd, requestShowCmd, requestApproveCmd, requestDenyCmd, requestCancelCmd)

	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Inspect and manage autent-backed sessions",
		Long: strings.TrimSpace(`
Inspect caller-safe autent-backed sessions that were issued through auth request
approval or the low-level issue-session seam. Omit --project-id on session list
for global inventory, or add it to narrow active auth to one project.
`),
		Example: strings.Join([]string{
			"  till auth session list --state active",
			"  till auth session validate --session-id sess-123 --session-secret secret-abc",
			"  till auth session revoke --session-id sess-123 --reason operator_revoke",
		}, "\n"),
		Args: cobra.NoArgs,
	}

	sessionListCmd := &cobra.Command{
		Use:   "list",
		Short: "List caller-safe auth sessions",
		Long: strings.TrimSpace(`
List caller-safe autent session state without exposing bearer secrets.

Use --project-id to narrow inventory to sessions approved for one project. Use
--state, --principal-id, or --client-id for additional deterministic filters.

Next step: use till auth session validate with --session-id and --session-secret
to verify one specific credential pair, or use revoke to rotate it.
`),
		Example: strings.Join([]string{
			"  till auth session list",
			"  till auth session list --project-id p1 --state active",
			"  till auth session list --state active --principal-id review-agent",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.session.list")
		},
	}
	sessionListCmd.Flags().StringVar(&sessionListOpts.sessionID, "session-id", "", "Optional session identifier filter")
	sessionListCmd.Flags().StringVar(&sessionListOpts.projectID, "project-id", "", "Optional approved project identifier filter")
	sessionListCmd.Flags().StringVar(&sessionListOpts.principalID, "principal-id", "", "Optional principal identifier filter")
	sessionListCmd.Flags().StringVar(&sessionListOpts.clientID, "client-id", "", "Optional client identifier filter")
	sessionListCmd.Flags().StringVar(&sessionListOpts.state, "state", sessionListOpts.state, "Session state filter (active|revoked|expired)")
	sessionListCmd.Flags().IntVar(&sessionListOpts.limit, "limit", sessionListOpts.limit, "Maximum rows to return")

	sessionValidateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate one session id/secret pair",
		Long: strings.TrimSpace(`
Validate one session_id and session_secret pair and return caller-safe identity
details for the credential.

Next step: if the session is valid, use it with MCP mutation calls. If it is no
longer needed, revoke it with till auth session revoke --session-id sess-123.
`),
		Example: "  till auth session validate --session-id sess-123 --session-secret secret-abc",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.session.validate")
		},
	}
	sessionValidateCmd.Flags().StringVar(&sessionValidateOpts.sessionID, "session-id", "", "Session identifier")
	sessionValidateCmd.Flags().StringVar(&sessionValidateOpts.sessionSecret, "session-secret", "", "Session secret")
	mustMarkFlagRequired(sessionValidateCmd, "session-id")
	mustMarkFlagRequired(sessionValidateCmd, "session-secret")

	sessionRevokeCmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke one local auth session",
		Long: strings.TrimSpace(`
Revoke one autent-backed session.

This command requires the --session-id flag; it does not accept the session id
as a positional argument.
`),
		Example: "  till auth session revoke --session-id sess-123 --reason operator_revoke",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.session.revoke")
		},
	}
	sessionRevokeCmd.Flags().StringVar(&revokeSessionOpts.sessionID, "session-id", "", "Session identifier")
	sessionRevokeCmd.Flags().StringVar(&revokeSessionOpts.reason, "reason", "", "Revocation reason")
	mustMarkFlagRequired(sessionRevokeCmd, "session-id")
	sessionCmd.AddCommand(sessionListCmd, sessionValidateCmd, sessionRevokeCmd)

	issueSessionCmd := &cobra.Command{
		Use:   "issue-session",
		Short: "Issue one local auth session for MCP dogfooding",
		Long: strings.TrimSpace(`
Issue one local auth session directly without going through the request/approval
lifecycle. This is a low-level seam for local testing only.

This command requires --principal-id. On success it returns session_id and
session_secret.

Next step: pass the returned session_id and session_secret to the requesting MCP
client, or validate the pair with till auth session validate.
`),
		Example: "  till auth issue-session --principal-id review-agent --principal-type agent --client-id till-mcp-stdio --client-type mcp-stdio",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.issue-session")
		},
	}
	issueSessionCmd.Flags().StringVar(&issueSessionOpts.principalID, "principal-id", "", "Principal identifier")
	issueSessionCmd.Flags().StringVar(&issueSessionOpts.principalType, "principal-type", issueSessionOpts.principalType, "Principal type (user|agent|service)")
	issueSessionCmd.Flags().StringVar(&issueSessionOpts.principalName, "principal-name", "", "Optional principal display name")
	issueSessionCmd.Flags().StringVar(&issueSessionOpts.clientID, "client-id", issueSessionOpts.clientID, "Client identifier")
	issueSessionCmd.Flags().StringVar(&issueSessionOpts.clientType, "client-type", issueSessionOpts.clientType, "Client type")
	issueSessionCmd.Flags().StringVar(&issueSessionOpts.clientName, "client-name", issueSessionOpts.clientName, "Client display name")
	issueSessionCmd.Flags().DurationVar(&issueSessionOpts.ttl, "ttl", issueSessionOpts.ttl, "Session time-to-live duration")
	mustMarkFlagRequired(issueSessionCmd, "principal-id")

	revokeSessionCmd := &cobra.Command{
		Use:   "revoke-session",
		Short: "Revoke one local auth session",
		Long: strings.TrimSpace(`
Revoke one autent-backed session directly. This is a low-level seam; prefer
till auth session revoke for the primary session lifecycle UX.

This command requires the --session-id flag; it does not accept the session id
as a positional argument.
`),
		Example: "  till auth revoke-session --session-id sess-123 --reason operator_revoke",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "auth.revoke-session")
		},
	}
	revokeSessionCmd.Flags().StringVar(&revokeSessionOpts.sessionID, "session-id", "", "Session identifier")
	revokeSessionCmd.Flags().StringVar(&revokeSessionOpts.reason, "reason", "", "Revocation reason")
	mustMarkFlagRequired(revokeSessionCmd, "session-id")
	authCmd.AddCommand(requestCmd, sessionCmd, issueSessionCmd, revokeSessionCmd)

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export a snapshot JSON payload",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "export")
		},
	}
	exportCmd.Flags().StringVar(&exportOpts.outPath, "out", exportOpts.outPath, "Output file path ('-' for stdout)")
	exportCmd.Flags().BoolVar(&exportOpts.includeArchived, "include-archived", exportOpts.includeArchived, "Include archived projects/columns/tasks")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a snapshot JSON payload",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFlow(cmd.Context(), "import")
		},
	}
	importCmd.Flags().StringVar(&importOpts.inPath, "in", "", "Input snapshot JSON file")

	pathsCmd := &cobra.Command{
		Use:   "paths",
		Short: "Print resolved runtime root/config/database/log paths",
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
			resolvedPaths, err := resolveRuntimePaths("paths", rootOpts, paths)
			if err != nil {
				return err
			}
			defaultCfg := config.Default(resolvedPaths.DBPath)
			cfg, err := config.Load(resolvedPaths.ConfigPath, defaultCfg)
			if err != nil {
				return fmt.Errorf("load config %q: %w", resolvedPaths.ConfigPath, err)
			}
			if resolvedPaths.DBOverridden {
				cfg.Database.Path = resolvedPaths.DBPath
			} else {
				resolvedPaths.DBPath = cfg.Database.Path
			}
			rootDir := runtimeRootDir(paths, resolvedPaths.DBPath)
			logDir, err := resolveRuntimeLogDir(cfg.Logging.DevFile.Dir, filepath.Join(rootDir, "logs"))
			if err != nil {
				return fmt.Errorf("resolve log dir: %w", err)
			}
			return writePathsOutput(stdout, rootOpts, resolvedPaths, rootDir, logDir)
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

	rootCmd.AddCommand(serveCmd, mcpCmd, authCmd, projectCmd, embeddingsCmd, captureStateCmd, kindCmd, leaseCmd, handoffCmd, exportCmd, importCmd, pathsCmd, initDevConfigCmd)
	rootCmd.AddCommand(serveCmd, mcpCmd, authCmd, projectCmd, embeddingsCmd, captureStateCmd, kindCmd, templateCmd, leaseCmd, handoffCmd, exportCmd, importCmd, pathsCmd, initDevConfigCmd)
	return fang.Execute(
		ctx,
		rootCmd,
		fang.WithoutCompletions(),
		fang.WithoutManpage(),
		fang.WithoutVersion(),
	)
}

// mustMarkFlagRequired fails fast when Cobra cannot mark one required flag.
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Sprintf("mark %s flag required: %v", name, err))
	}
}

// mustMarkFlagHidden fails fast when Cobra cannot hide one legacy flag.
func mustMarkFlagHidden(cmd *cobra.Command, name string) {
	if err := cmd.Flags().MarkHidden(name); err != nil {
		panic(fmt.Sprintf("mark %s flag hidden: %v", name, err))
	}
}

// writeVersion writes the current CLI version to stdout.
func writeVersion(stdout io.Writer) error {
	return writeCLIKV(stdout, "Till Version", [][2]string{
		{"app", "till"},
		{"version", version},
	})
}

// writePathsOutput renders resolved paths in one structured laslig key/value view.
func writePathsOutput(stdout io.Writer, opts rootCommandOptions, resolvedPaths resolvedRuntimePaths, rootDir, logDir string) error {
	rows := buildPathsRows(opts, resolvedPaths, rootDir, logDir)
	pairs := make([][2]string, 0, len(rows))
	for _, row := range rows {
		pairs = append(pairs, [2]string{row.key, row.value})
	}
	if supportsStyledOutputFunc(stdout) {
		if err := writeCLIKVWithPrinter(newStyledCLIPrinter(stdout), "Resolved Paths", pairs); err != nil {
			return fmt.Errorf("write styled paths output: %w", err)
		}
		return nil
	}
	return writeCLIKV(stdout, "Resolved Paths", pairs)
}

// buildPathsRows returns the stable key/value rows used by both plain and styled output.
func buildPathsRows(opts rootCommandOptions, resolvedPaths resolvedRuntimePaths, rootDir, logDir string) []struct {
	key   string
	value string
} {
	return []struct {
		key   string
		value string
	}{
		{key: "app", value: opts.appName},
		{key: "root", value: rootDir},
		{key: "config", value: resolvedPaths.ConfigPath},
		{key: "database", value: resolvedPaths.DBPath},
		{key: "logs", value: logDir},
		{key: "dev_mode", value: fmt.Sprintf("%t", opts.devMode)},
	}
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

// resolveRuntimePaths resolves config and DB paths for the current command.
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
	return out, nil
}

// ensureRuntimePathParents creates any required runtime parent directories before startup.
func ensureRuntimePathParents(command string, paths resolvedRuntimePaths) error {
	_ = command
	_ = paths
	return nil
}

// runtimeRootDir resolves the effective runtime root for the active database path.
func runtimeRootDir(defaultPaths platform.Paths, dbPath string) string {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return defaultPaths.DataDir
	}
	return filepath.Dir(dbPath)
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
	return writeCLIKV(stdout, "Dev Config", [][2]string{
		{"status", msg},
		{"config path", shellEscapePath(configPath)},
		{"logging level", "debug"},
	})
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

// firstNonEmpty returns the first non-empty trimmed value in order.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
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
	_ authCommandOptions,
	projectListOpts projectListCommandOptions,
	projectCreateOpts projectCreateCommandOptions,
	projectShowOpts projectShowCommandOptions,
	projectDiscoverOpts projectReadinessCommandOptions,
	captureStateOpts captureStateCommandOptions,
	embeddingsStatusOpts embeddingsStatusCommandOptions,
	embeddingsReindexOpts embeddingsReindexCommandOptions,
	kindListOpts kindListCommandOptions,
	kindUpsertOpts kindUpsertCommandOptions,
	kindAllowlistOpts kindAllowlistCommandOptions,
	templateLibraryListOpts templateLibraryListCommandOptions,
	templateLibraryShowOpts templateLibraryShowCommandOptions,
	templateLibraryUpsertOpts templateLibraryUpsertCommandOptions,
	templateProjectBindOpts templateProjectBindCommandOptions,
	templateProjectBindingOpts templateProjectBindingCommandOptions,
	templateContractShowOpts templateContractShowCommandOptions,
	leaseListOpts leaseListCommandOptions,
	leaseIssueOpts leaseIssueCommandOptions,
	leaseHeartbeatOpts leaseHeartbeatCommandOptions,
	leaseRenewOpts leaseRenewCommandOptions,
	leaseRevokeOpts leaseRevokeCommandOptions,
	leaseRevokeAllOpts leaseRevokeAllCommandOptions,
	handoffCreateOpts handoffCreateCommandOptions,
	handoffGetOpts handoffGetCommandOptions,
	handoffListOpts handoffListCommandOptions,
	handoffUpdateOpts handoffUpdateCommandOptions,
	issueSessionOpts issueSessionCommandOptions,
	requestCreateOpts requestCreateCommandOptions,
	requestListOpts requestListCommandOptions,
	requestShowOpts requestShowCommandOptions,
	requestApproveOpts requestResolveCommandOptions,
	requestDenyOpts requestResolveCommandOptions,
	requestCancelOpts requestResolveCommandOptions,
	sessionListOpts sessionListCommandOptions,
	sessionValidateOpts sessionValidateCommandOptions,
	revokeSessionOpts revokeSessionCommandOptions,
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
	if dbOverridden {
		cfg.Database.Path = dbPath
	} else {
		dbPath = cfg.Database.Path
	}
	if command == "" {
		if err := ensureStartupIdentityActorID(configPath, &cfg); err != nil {
			return fmt.Errorf("bootstrap identity.actor_id: %w", err)
		}
	}
	bootstrapRequired := startupBootstrapRequired(cfg)
	rootDir := runtimeRootDir(paths, dbPath)

	logDir, err := resolveRuntimeLogDir(cfg.Logging.DevFile.Dir, filepath.Join(rootDir, "logs"))
	if err != nil {
		return fmt.Errorf("resolve runtime log dir: %w", err)
	}
	logger, err := newRuntimeLogger(stderr, rootOpts.appName, rootOpts.devMode, cfg.Logging, logDir, time.Now)
	if err != nil {
		return fmt.Errorf("configure runtime logger: %w", err)
	}
	if shouldMuteRuntimeConsole(command) {
		// Keep interactive and one-shot operator surfaces clean: runtime logs stay in the dev-file sink while the command is active.
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
	logger.Debug("runtime paths resolved", "config_path", configPath, "root", rootDir, "db_path", dbPath, "logs_dir", logDir)
	logger.Info("configuration loaded", "config_path", configPath, "db_path", cfg.Database.Path, "log_level", cfg.Logging.Level)
	if devPath := logger.DevLogPath(); devPath != "" {
		logger.Info("runtime file logging enabled", "path", devPath)
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

	liveWaitBroker, err := newRuntimeLiveWaitBrokerFunc(repo.DB(), rootDir)
	if err != nil {
		logger.Error("live wait broker setup failed", "db_path", cfg.Database.Path, "err", err)
		return fmt.Errorf("configure live wait broker: %w", err)
	}
	defer func() {
		if closeErr := liveWaitBroker.Close(); closeErr != nil {
			logger.Warn("live wait broker close failed", "db_path", cfg.Database.Path, "err", closeErr)
		}
	}()
	logger.Info("live wait broker ready", "db_path", cfg.Database.Path, "mode", "localipc")

	authSvc, err := autentauth.NewSharedDB(autentauth.Config{
		DB:          repo.DB(),
		TablePrefix: autentauth.DefaultTablePrefix,
		IDGenerator: uuid.NewString,
	})
	if err != nil {
		logger.Error("autent setup failed", "db_path", cfg.Database.Path, "err", err)
		return fmt.Errorf("configure autent service: %w", err)
	}
	if err := authSvc.EnsureDogfoodPolicy(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			commandName := command
			if commandName == "" {
				commandName = "tui"
			}
			logger.Info("command flow complete", "command", commandName, "shutdown", "interrupt")
			return nil
		}
		logger.Error("autent dogfood policy setup failed", "db_path", cfg.Database.Path, "err", err)
		return fmt.Errorf("ensure autent dogfood policy: %w", err)
	}
	logger.Info("autent service ready", "db_path", cfg.Database.Path, "table_prefix", autentauth.DefaultTablePrefix)

	embeddingRuntimeCfg, err := buildEmbeddingRuntimeConfig(cfg, rootOpts.appName, command)
	if err != nil {
		logger.Error("embeddings runtime config invalid", "err", err)
		return fmt.Errorf("build embeddings runtime config: %w", err)
	}

	var embeddingLifecycle app.EmbeddingLifecycleStore
	if lifecycle, ok := any(repo).(app.EmbeddingLifecycleStore); ok {
		embeddingLifecycle = lifecycle
	}
	var embeddingSearchIndex app.EmbeddingSearchIndex
	if idx, ok := any(repo).(app.EmbeddingSearchIndex); ok {
		embeddingSearchIndex = idx
	}

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
	if err := app.PrepareEmbeddingsLifecycle(ctx, embeddingLifecycle, embeddingRuntimeCfg); err != nil {
		logger.Error("embeddings lifecycle prepare failed", "err", err)
		return fmt.Errorf("prepare embeddings lifecycle: %w", err)
	}
	if cfg.Embeddings.Enabled {
		switch {
		case embeddingLifecycle == nil:
			logger.Warn("embeddings lifecycle store unavailable; semantic status tracking remains disabled")
		case embeddingGenerator == nil:
			logger.Warn("embeddings worker not started; queued states remain observable until provider setup succeeds")
		case embeddingSearchIndex == nil:
			logger.Warn("embeddings worker not started; task search index is unavailable")
		default:
			go func() {
				if err := app.NewEmbeddingWorker(repo, embeddingLifecycle, embeddingGenerator, embeddingSearchIndex, nil, embeddingRuntimeCfg).Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
					logger.Warn("embeddings worker stopped", "err", err)
				}
			}()
			logger.Info("embeddings worker ready", "worker_id", embeddingRuntimeCfg.WorkerID, "poll_interval", embeddingRuntimeCfg.PollInterval.String(), "claim_ttl", embeddingRuntimeCfg.ClaimTTL.String(), "max_attempts", embeddingRuntimeCfg.MaxAttempts)
		}
	}

	svc := app.NewService(repo, uuid.NewString, nil, app.ServiceConfig{
		DefaultDeleteMode:        app.DeleteMode(cfg.Delete.DefaultMode),
		AutoCreateProjectColumns: true,
		AuthRequests:             authSvc,
		AuthBackend:              authSvc,
		LiveWaitBroker:           liveWaitBroker,
		EmbeddingGenerator:       embeddingGenerator,
		SearchIndex:              embeddingSearchIndex,
		EmbeddingLifecycle:       embeddingLifecycle,
		EmbeddingRuntime:         embeddingRuntimeCfg,
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
		if err := withInterruptEchoSuppressedFunc(func() error {
			return runServe(ctx, svc, authSvc, rootOpts.appName, serveOpts)
		}); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("command flow complete", "command", "serve", "shutdown", "interrupt")
				return nil
			}
			logger.Error("command flow failed", "command", "serve", "err", err)
			return fmt.Errorf("run serve command: %w", err)
		}
		logger.Info("command flow complete", "command", "serve")
		return nil
	case "mcp":
		logger.Info("command flow start", "command", "mcp", "transport", "stdio")
		if err := withInterruptEchoSuppressedFunc(func() error {
			return runMCP(ctx, svc, authSvc, rootOpts.appName, serveOpts)
		}); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("command flow complete", "command", "mcp", "transport", "stdio", "shutdown", "interrupt")
				return nil
			}
			logger.Error("command flow failed", "command", "mcp", "transport", "stdio", "err", err)
			return fmt.Errorf("run mcp command: %w", err)
		}
		logger.Info("command flow complete", "command", "mcp", "transport", "stdio")
		return nil
	case "auth.issue-session":
		logger.Info("command flow start", "command", "auth.issue-session")
		if err := runAuthIssueSession(ctx, authSvc, issueSessionOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.issue-session", "err", err)
			return fmt.Errorf("run auth issue-session command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.issue-session")
		return nil
	case "auth.request.create":
		logger.Info("command flow start", "command", "auth.request.create")
		if err := runAuthRequestCreate(ctx, svc, cfg, requestCreateOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.request.create", "err", err)
			return fmt.Errorf("run auth request create command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.request.create")
		return nil
	case "auth.request.list":
		logger.Info("command flow start", "command", "auth.request.list")
		if err := runAuthRequestList(ctx, svc, requestListOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.request.list", "err", err)
			return fmt.Errorf("run auth request list command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.request.list")
		return nil
	case "auth.request.show":
		logger.Info("command flow start", "command", "auth.request.show")
		if err := runAuthRequestShow(ctx, svc, requestShowOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.request.show", "err", err)
			return fmt.Errorf("run auth request show command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.request.show")
		return nil
	case "auth.request.approve":
		logger.Info("command flow start", "command", "auth.request.approve")
		if err := runAuthRequestApprove(ctx, svc, cfg, requestApproveOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.request.approve", "err", err)
			return fmt.Errorf("run auth request approve command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.request.approve")
		return nil
	case "auth.request.deny":
		logger.Info("command flow start", "command", "auth.request.deny")
		if err := runAuthRequestDeny(ctx, svc, cfg, requestDenyOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.request.deny", "err", err)
			return fmt.Errorf("run auth request deny command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.request.deny")
		return nil
	case "auth.request.cancel":
		logger.Info("command flow start", "command", "auth.request.cancel")
		if err := runAuthRequestCancel(ctx, svc, cfg, requestCancelOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.request.cancel", "err", err)
			return fmt.Errorf("run auth request cancel command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.request.cancel")
		return nil
	case "auth.session.list":
		logger.Info("command flow start", "command", "auth.session.list")
		if err := runAuthSessionList(ctx, svc, sessionListOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.session.list", "err", err)
			return fmt.Errorf("run auth session list command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.session.list")
		return nil
	case "auth.session.validate":
		logger.Info("command flow start", "command", "auth.session.validate")
		if err := runAuthSessionValidate(ctx, svc, sessionValidateOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.session.validate", "err", err)
			return fmt.Errorf("run auth session validate command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.session.validate")
		return nil
	case "auth.session.revoke":
		logger.Info("command flow start", "command", "auth.session.revoke")
		if err := runAuthSessionRevoke(ctx, svc, revokeSessionOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.session.revoke", "err", err)
			return fmt.Errorf("run auth session revoke command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.session.revoke")
		return nil
	case "auth.revoke-session":
		logger.Info("command flow start", "command", "auth.revoke-session")
		if err := runAuthRevokeSession(ctx, authSvc, revokeSessionOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "auth.revoke-session", "err", err)
			return fmt.Errorf("run auth revoke-session command: %w", err)
		}
		logger.Info("command flow complete", "command", "auth.revoke-session")
		return nil
	case "project.list":
		logger.Info("command flow start", "command", "project.list")
		if err := runProjectList(ctx, svc, cfg, projectListOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "project.list", "err", err)
			return fmt.Errorf("run project list command: %w", err)
		}
		logger.Info("command flow complete", "command", "project.list")
		return nil
	case "project.create":
		logger.Info("command flow start", "command", "project.create")
		if err := runProjectCreate(ctx, svc, cfg, projectCreateOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "project.create", "err", err)
			return fmt.Errorf("run project create command: %w", err)
		}
		logger.Info("command flow complete", "command", "project.create")
		return nil
	case "project.show":
		logger.Info("command flow start", "command", "project.show")
		if err := runProjectShow(ctx, svc, cfg, projectShowOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "project.show", "err", err)
			return fmt.Errorf("run project show command: %w", err)
		}
		logger.Info("command flow complete", "command", "project.show")
		return nil
	case "project.discover":
		logger.Info("command flow start", "command", "project.discover")
		if err := runProjectDiscover(ctx, svc, cfg, projectDiscoverOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "project.discover", "err", err)
			return fmt.Errorf("run project discover command: %w", err)
		}
		logger.Info("command flow complete", "command", "project.discover")
		return nil
	case "embeddings.status":
		logger.Info("command flow start", "command", "embeddings.status")
		if err := runEmbeddingsStatus(ctx, svc, embeddingsStatusOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "embeddings.status", "err", err)
			return fmt.Errorf("run embeddings status command: %w", err)
		}
		logger.Info("command flow complete", "command", "embeddings.status")
		return nil
	case "embeddings.reindex":
		logger.Info("command flow start", "command", "embeddings.reindex")
		if err := runEmbeddingsReindex(ctx, svc, cfg, embeddingsReindexOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "embeddings.reindex", "err", err)
			return fmt.Errorf("run embeddings reindex command: %w", err)
		}
		logger.Info("command flow complete", "command", "embeddings.reindex")
		return nil
	case "capture-state":
		logger.Info("command flow start", "command", "capture-state")
		if err := runCaptureState(ctx, svc, authSvc, captureStateOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "capture-state", "err", err)
			return fmt.Errorf("run capture-state command: %w", err)
		}
		logger.Info("command flow complete", "command", "capture-state")
		return nil
	case "kind.list":
		logger.Info("command flow start", "command", "kind.list")
		if err := runKindList(ctx, svc, kindListOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "kind.list", "err", err)
			return fmt.Errorf("run kind list command: %w", err)
		}
		logger.Info("command flow complete", "command", "kind.list")
		return nil
	case "kind.upsert":
		logger.Info("command flow start", "command", "kind.upsert")
		if err := runKindUpsert(ctx, svc, cfg, kindUpsertOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "kind.upsert", "err", err)
			return fmt.Errorf("run kind upsert command: %w", err)
		}
		logger.Info("command flow complete", "command", "kind.upsert")
		return nil
	case "kind.allowlist.list":
		logger.Info("command flow start", "command", "kind.allowlist.list")
		if err := runKindAllowlistList(ctx, svc, kindAllowlistOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "kind.allowlist.list", "err", err)
			return fmt.Errorf("run kind allowlist list command: %w", err)
		}
		logger.Info("command flow complete", "command", "kind.allowlist.list")
		return nil
	case "kind.allowlist.set":
		logger.Info("command flow start", "command", "kind.allowlist.set")
		if err := runKindAllowlistSet(ctx, svc, cfg, kindAllowlistOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "kind.allowlist.set", "err", err)
			return fmt.Errorf("run kind allowlist set command: %w", err)
		}
		logger.Info("command flow complete", "command", "kind.allowlist.set")
		return nil
	case "template.library.list":
		logger.Info("command flow start", "command", "template.library.list")
		if err := runTemplateLibraryList(ctx, svc, templateLibraryListOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "template.library.list", "err", err)
			return fmt.Errorf("run template library list command: %w", err)
		}
		logger.Info("command flow complete", "command", "template.library.list")
		return nil
	case "template.library.show":
		logger.Info("command flow start", "command", "template.library.show")
		if err := runTemplateLibraryShow(ctx, svc, templateLibraryShowOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "template.library.show", "err", err)
			return fmt.Errorf("run template library show command: %w", err)
		}
		logger.Info("command flow complete", "command", "template.library.show")
		return nil
	case "template.library.upsert":
		logger.Info("command flow start", "command", "template.library.upsert")
		if err := runTemplateLibraryUpsert(ctx, svc, cfg, templateLibraryUpsertOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "template.library.upsert", "err", err)
			return fmt.Errorf("run template library upsert command: %w", err)
		}
		logger.Info("command flow complete", "command", "template.library.upsert")
		return nil
	case "template.project.bind":
		logger.Info("command flow start", "command", "template.project.bind")
		if err := runTemplateProjectBind(ctx, svc, cfg, templateProjectBindOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "template.project.bind", "err", err)
			return fmt.Errorf("run template project bind command: %w", err)
		}
		logger.Info("command flow complete", "command", "template.project.bind")
		return nil
	case "template.project.binding":
		logger.Info("command flow start", "command", "template.project.binding")
		if err := runTemplateProjectBinding(ctx, svc, templateProjectBindingOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "template.project.binding", "err", err)
			return fmt.Errorf("run template project binding command: %w", err)
		}
		logger.Info("command flow complete", "command", "template.project.binding")
		return nil
	case "template.contract.show":
		logger.Info("command flow start", "command", "template.contract.show")
		if err := runTemplateContractShow(ctx, svc, templateContractShowOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "template.contract.show", "err", err)
			return fmt.Errorf("run template contract show command: %w", err)
		}
		logger.Info("command flow complete", "command", "template.contract.show")
		return nil
	case "lease.list":
		logger.Info("command flow start", "command", "lease.list")
		if err := runLeaseList(ctx, svc, leaseListOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "lease.list", "err", err)
			return fmt.Errorf("run lease list command: %w", err)
		}
		logger.Info("command flow complete", "command", "lease.list")
		return nil
	case "lease.issue":
		logger.Info("command flow start", "command", "lease.issue")
		if err := runLeaseIssue(ctx, svc, cfg, leaseIssueOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "lease.issue", "err", err)
			return fmt.Errorf("run lease issue command: %w", err)
		}
		logger.Info("command flow complete", "command", "lease.issue")
		return nil
	case "lease.heartbeat":
		logger.Info("command flow start", "command", "lease.heartbeat")
		if err := runLeaseHeartbeat(ctx, svc, cfg, leaseHeartbeatOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "lease.heartbeat", "err", err)
			return fmt.Errorf("run lease heartbeat command: %w", err)
		}
		logger.Info("command flow complete", "command", "lease.heartbeat")
		return nil
	case "lease.renew":
		logger.Info("command flow start", "command", "lease.renew")
		if err := runLeaseRenew(ctx, svc, cfg, leaseRenewOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "lease.renew", "err", err)
			return fmt.Errorf("run lease renew command: %w", err)
		}
		logger.Info("command flow complete", "command", "lease.renew")
		return nil
	case "lease.revoke":
		logger.Info("command flow start", "command", "lease.revoke")
		if err := runLeaseRevoke(ctx, svc, cfg, leaseRevokeOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "lease.revoke", "err", err)
			return fmt.Errorf("run lease revoke command: %w", err)
		}
		logger.Info("command flow complete", "command", "lease.revoke")
		return nil
	case "lease.revoke-all":
		logger.Info("command flow start", "command", "lease.revoke-all")
		if err := runLeaseRevokeAll(ctx, svc, cfg, leaseRevokeAllOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "lease.revoke-all", "err", err)
			return fmt.Errorf("run lease revoke-all command: %w", err)
		}
		logger.Info("command flow complete", "command", "lease.revoke-all")
		return nil
	case "handoff.create":
		logger.Info("command flow start", "command", "handoff.create")
		if err := runHandoffCreate(ctx, svc, cfg, handoffCreateOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "handoff.create", "err", err)
			return fmt.Errorf("run handoff create command: %w", err)
		}
		logger.Info("command flow complete", "command", "handoff.create")
		return nil
	case "handoff.get":
		logger.Info("command flow start", "command", "handoff.get")
		if err := runHandoffGet(ctx, svc, handoffGetOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "handoff.get", "err", err)
			return fmt.Errorf("run handoff get command: %w", err)
		}
		logger.Info("command flow complete", "command", "handoff.get")
		return nil
	case "handoff.list":
		logger.Info("command flow start", "command", "handoff.list")
		if err := runHandoffList(ctx, svc, handoffListOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "handoff.list", "err", err)
			return fmt.Errorf("run handoff list command: %w", err)
		}
		logger.Info("command flow complete", "command", "handoff.list")
		return nil
	case "handoff.update":
		logger.Info("command flow start", "command", "handoff.update")
		if err := runHandoffUpdate(ctx, svc, cfg, handoffUpdateOpts, stdout); err != nil {
			logger.Error("command flow failed", "command", "handoff.update", "err", err)
			return fmt.Errorf("run handoff update command: %w", err)
		}
		logger.Info("command flow complete", "command", "handoff.update")
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

// shouldMuteRuntimeConsole reports whether runtime logs should stay off the console for one command.
func shouldMuteRuntimeConsole(command string) bool {
	switch command {
	case "serve", "mcp":
		return false
	default:
		return true
	}
}

// runServe runs the serve subcommand flow.
func runServe(ctx context.Context, svc *app.Service, auth *autentauth.Service, appName string, opts serveCommandOptions) error {
	appAdapter := servercommon.NewAppServiceAdapter(svc, auth)
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
func runMCP(ctx context.Context, svc *app.Service, auth *autentauth.Service, appName string, opts serveCommandOptions) error {
	appAdapter := servercommon.NewAppServiceAdapter(svc, auth)
	return mcpCommandRunner(ctx, serveradapter.Config{
		MCPEndpoint:   opts.mcpEndpoint,
		ServerName:    appName,
		ServerVersion: version,
	}, serveradapter.Dependencies{
		CaptureState: appAdapter,
		Attention:    appAdapter,
	})
}

// runAuthIssueSession issues one local auth session for dogfood MCP use.
func runAuthIssueSession(ctx context.Context, auth *autentauth.Service, opts issueSessionCommandOptions, stdout io.Writer) error {
	if auth == nil {
		return fmt.Errorf("autent service is not configured")
	}
	principalID := strings.TrimSpace(opts.principalID)
	if principalID == "" {
		return fmt.Errorf("--principal-id is required")
	}
	issued, err := auth.IssueSession(ctx, autentauth.IssueSessionInput{
		PrincipalID:   principalID,
		PrincipalType: strings.TrimSpace(opts.principalType),
		PrincipalName: strings.TrimSpace(opts.principalName),
		ClientID:      strings.TrimSpace(opts.clientID),
		ClientType:    strings.TrimSpace(opts.clientType),
		ClientName:    strings.TrimSpace(opts.clientName),
		TTL:           opts.ttl,
	})
	if err != nil {
		return fmt.Errorf("issue auth session: %w", err)
	}
	return writeAuthSessionDetailHuman(stdout, authSessionPayloadJSON{
		SessionID:     issued.Session.ID,
		State:         "active",
		PrincipalID:   principalID,
		PrincipalType: strings.TrimSpace(opts.principalType),
		PrincipalName: firstNonEmpty(strings.TrimSpace(opts.principalName), principalID),
		ClientID:      strings.TrimSpace(opts.clientID),
		ClientType:    strings.TrimSpace(opts.clientType),
		ClientName:    firstNonEmpty(strings.TrimSpace(opts.clientName), strings.TrimSpace(opts.clientID)),
		ExpiresAt:     issued.Session.ExpiresAt.UTC(),
	}, issued.Secret)
}

// runAuthRevokeSession revokes one local auth session.
func runAuthRevokeSession(ctx context.Context, auth *autentauth.Service, opts revokeSessionCommandOptions, stdout io.Writer) error {
	if auth == nil {
		return fmt.Errorf("autent service is not configured")
	}
	sessionID := strings.TrimSpace(opts.sessionID)
	if sessionID == "" {
		return fmt.Errorf("--session-id is required")
	}
	session, err := auth.RevokeSession(ctx, sessionID, strings.TrimSpace(opts.reason))
	if err != nil {
		return fmt.Errorf("revoke auth session: %w", err)
	}
	return writeAuthSessionDetailHuman(stdout, authSessionPayloadJSON{
		SessionID:        session.ID,
		State:            "revoked",
		PrincipalID:      session.PrincipalID,
		ClientID:         session.ClientID,
		ExpiresAt:        session.ExpiresAt.UTC(),
		RevokedAt:        session.RevokedAt,
		RevocationReason: session.RevocationReason,
	}, "")
}

// runCaptureState captures one summary-first recovery snapshot and writes it as stable JSON.
func runCaptureState(ctx context.Context, svc *app.Service, authSvc *autentauth.Service, opts captureStateCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if err := requireProjectID("capture-state", opts.projectID); err != nil {
		return err
	}
	adapter := servercommon.NewAppServiceAdapter(svc, authSvc)
	capture, err := adapter.CaptureState(ctx, servercommon.CaptureStateRequest{
		ProjectID: strings.TrimSpace(opts.projectID),
		ScopeType: strings.TrimSpace(opts.scopeType),
		ScopeID:   strings.TrimSpace(opts.scopeID),
		View:      strings.TrimSpace(opts.view),
	})
	if err != nil {
		return fmt.Errorf("capture state: %w", err)
	}
	return writeJSON(stdout, capture)
}

// runKindList lists kind definitions and writes them as stable JSON.
func runKindList(ctx context.Context, svc *app.Service, opts kindListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	kinds, err := svc.ListKindDefinitions(ctx, opts.includeArchived)
	if err != nil {
		return fmt.Errorf("list kind definitions: %w", err)
	}
	payload := make([]kindDefinitionPayloadJSON, 0, len(kinds))
	for _, kind := range kinds {
		payload = append(payload, kindDefinitionPayload(kind))
	}
	return writeJSON(stdout, payload)
}

// runKindUpsert creates or updates one kind definition and writes it as stable JSON.
func runKindUpsert(ctx context.Context, svc *app.Service, cfg config.Config, opts kindUpsertCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	template, err := parseOptionalKindTemplateJSON(opts.templateJSON)
	if err != nil {
		return err
	}
	ctx = cliMutationContext(ctx, cfg)
	kind, err := svc.UpsertKindDefinition(ctx, app.CreateKindDefinitionInput{
		ID:                  domain.KindID(strings.TrimSpace(opts.id)),
		DisplayName:         strings.TrimSpace(opts.displayName),
		DescriptionMarkdown: strings.TrimSpace(opts.descriptionMarkdown),
		AppliesTo:           toKindAppliesToList(opts.appliesTo),
		AllowedParentScopes: toKindAppliesToList(opts.allowedParentScopes),
		PayloadSchemaJSON:   strings.TrimSpace(opts.payloadSchemaJSON),
		Template:            template,
	})
	if err != nil {
		return fmt.Errorf("upsert kind definition: %w", err)
	}
	return writeJSON(stdout, kindDefinitionPayload(kind))
}

// runKindAllowlistList lists one project's explicit kind allowlist and writes it as stable JSON.
func runKindAllowlistList(ctx context.Context, svc *app.Service, opts kindAllowlistCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	projectID := strings.TrimSpace(opts.projectID)
	if err := requireProjectID("kind allowlist list", projectID); err != nil {
		return err
	}
	kindIDs, err := svc.ListProjectAllowedKinds(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list project allowed kinds: %w", err)
	}
	return writeJSON(stdout, kindAllowlistPayloadJSON{
		ProjectID: projectID,
		KindIDs:   toStrings(kindIDs),
	})
}

// runKindAllowlistSet replaces one project's explicit kind allowlist and writes it as stable JSON.
func runKindAllowlistSet(ctx context.Context, svc *app.Service, cfg config.Config, opts kindAllowlistCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	projectID := strings.TrimSpace(opts.projectID)
	if err := requireProjectID("kind allowlist set", projectID); err != nil {
		return err
	}
	if len(opts.kindIDs) == 0 {
		return fmt.Errorf("--kind-id is required")
	}
	if err := svc.SetProjectAllowedKinds(ctx, app.SetProjectAllowedKindsInput{
		ProjectID: projectID,
		KindIDs:   toKindIDList(opts.kindIDs),
	}); err != nil {
		return fmt.Errorf("set project allowed kinds: %w", err)
	}
	kindIDs, err := svc.ListProjectAllowedKinds(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list project allowed kinds: %w", err)
	}
	return writeJSON(stdout, kindAllowlistPayloadJSON{
		ProjectID: projectID,
		KindIDs:   toStrings(kindIDs),
	})
}

// runTemplateLibraryList lists template libraries and writes them in a human-readable operator view.
func runTemplateLibraryList(ctx context.Context, svc *app.Service, opts templateLibraryListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	libraries, err := svc.ListTemplateLibraries(ctx, app.ListTemplateLibrariesInput{
		Scope:     domain.TemplateLibraryScope(strings.TrimSpace(opts.scope)),
		ProjectID: strings.TrimSpace(opts.projectID),
		Status:    domain.TemplateLibraryStatus(strings.TrimSpace(opts.status)),
	})
	if err != nil {
		return fmt.Errorf("list template libraries: %w", err)
	}
	return writeTemplateLibraryList(stdout, libraries)
}

// runTemplateLibraryShow loads one template library and writes it in a human-readable operator view.
func runTemplateLibraryShow(ctx context.Context, svc *app.Service, opts templateLibraryShowCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	libraryID := strings.TrimSpace(opts.libraryID)
	if libraryID == "" {
		return fmt.Errorf("--library-id is required")
	}
	library, err := svc.GetTemplateLibrary(ctx, libraryID)
	if err != nil {
		return fmt.Errorf("get template library: %w", err)
	}
	return writeTemplateLibraryDetail(stdout, library)
}

// runTemplateLibraryUpsert creates or updates one template library from the JSON CLI transport and writes a human-readable result.
func runTemplateLibraryUpsert(ctx context.Context, svc *app.Service, cfg config.Config, opts templateLibraryUpsertCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	spec, err := parseTemplateLibrarySpecJSON(opts.specJSON)
	if err != nil {
		return err
	}
	ctx = cliMutationContext(ctx, cfg)
	nodeTemplates := make([]app.UpsertNodeTemplateInput, 0, len(spec.NodeTemplates))
	for _, nodeTemplate := range spec.NodeTemplates {
		childRules := make([]app.UpsertTemplateChildRuleInput, 0, len(nodeTemplate.ChildRules))
		for _, childRule := range nodeTemplate.ChildRules {
			childRules = append(childRules, app.UpsertTemplateChildRuleInput{
				ID:                        strings.TrimSpace(childRule.ID),
				Position:                  childRule.Position,
				ChildScopeLevel:           childRule.ChildScopeLevel,
				ChildKindID:               childRule.ChildKindID,
				TitleTemplate:             strings.TrimSpace(childRule.TitleTemplate),
				DescriptionTemplate:       strings.TrimSpace(childRule.DescriptionTemplate),
				ResponsibleActorKind:      childRule.ResponsibleActorKind,
				EditableByActorKinds:      append([]domain.TemplateActorKind(nil), childRule.EditableByActorKinds...),
				CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), childRule.CompletableByActorKinds...),
				OrchestratorMayComplete:   childRule.OrchestratorMayComplete,
				RequiredForParentDone:     childRule.RequiredForParentDone,
				RequiredForContainingDone: childRule.RequiredForContainingDone,
			})
		}
		nodeTemplates = append(nodeTemplates, app.UpsertNodeTemplateInput{
			ID:                      strings.TrimSpace(nodeTemplate.ID),
			ScopeLevel:              nodeTemplate.ScopeLevel,
			NodeKindID:              nodeTemplate.NodeKindID,
			DisplayName:             strings.TrimSpace(nodeTemplate.DisplayName),
			DescriptionMarkdown:     strings.TrimSpace(nodeTemplate.DescriptionMarkdown),
			ProjectMetadataDefaults: nodeTemplate.ProjectMetadataDefaults,
			TaskMetadataDefaults:    nodeTemplate.TaskMetadataDefaults,
			ChildRules:              childRules,
		})
	}
	library, err := svc.UpsertTemplateLibrary(ctx, app.UpsertTemplateLibraryInput{
		ID:              strings.TrimSpace(spec.ID),
		Scope:           spec.Scope,
		ProjectID:       strings.TrimSpace(spec.ProjectID),
		Name:            strings.TrimSpace(spec.Name),
		Description:     strings.TrimSpace(spec.Description),
		Status:          spec.Status,
		SourceLibraryID: strings.TrimSpace(spec.SourceLibraryID),
		NodeTemplates:   nodeTemplates,
	})
	if err != nil {
		return fmt.Errorf("upsert template library: %w", err)
	}
	return writeTemplateLibraryDetail(stdout, library)
}

// runTemplateProjectBind binds one project to one approved template library and writes a human-readable operator view.
func runTemplateProjectBind(ctx context.Context, svc *app.Service, cfg config.Config, opts templateProjectBindCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	projectID := strings.TrimSpace(opts.projectID)
	if err := requireProjectID("template project bind", projectID); err != nil {
		return err
	}
	libraryID := strings.TrimSpace(opts.libraryID)
	if libraryID == "" {
		return fmt.Errorf("--library-id is required")
	}
	binding, err := svc.BindProjectTemplateLibrary(ctx, app.BindProjectTemplateLibraryInput{
		ProjectID:        projectID,
		LibraryID:        libraryID,
		BoundByActorID:   cliMutationActorID(cfg),
		BoundByActorName: strings.TrimSpace(cfg.Identity.DisplayName),
		BoundByActorType: cliMutationActorType(cfg),
	})
	if err != nil {
		return fmt.Errorf("bind project template library: %w", err)
	}
	return writeProjectTemplateBindingDetail(stdout, binding)
}

// runTemplateProjectBinding loads one project's active template-library binding and writes a human-readable operator view.
func runTemplateProjectBinding(ctx context.Context, svc *app.Service, opts templateProjectBindingCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	projectID := strings.TrimSpace(opts.projectID)
	if err := requireProjectID("template project binding", projectID); err != nil {
		return err
	}
	binding, err := svc.GetProjectTemplateBinding(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get project template binding: %w", err)
	}
	return writeProjectTemplateBindingDetail(stdout, binding)
}

// runTemplateContractShow loads one generated-node contract snapshot and writes it in a human-readable operator view.
func runTemplateContractShow(ctx context.Context, svc *app.Service, opts templateContractShowCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	nodeID := strings.TrimSpace(opts.nodeID)
	if nodeID == "" {
		return fmt.Errorf("--node-id is required")
	}
	snapshot, err := svc.GetNodeContractSnapshot(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("get node contract snapshot: %w", err)
	}
	return writeNodeContractSnapshotDetail(stdout, snapshot)
}

// runLeaseList lists capability leases and writes them in a human-readable operator view.
func runLeaseList(ctx context.Context, svc *app.Service, opts leaseListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if err := requireProjectID("lease list", opts.projectID); err != nil {
		return err
	}
	leases, err := svc.ListCapabilityLeases(ctx, app.ListCapabilityLeasesInput{
		ProjectID:      strings.TrimSpace(opts.projectID),
		ScopeType:      domain.CapabilityScopeType(strings.TrimSpace(opts.scopeType)),
		ScopeID:        strings.TrimSpace(opts.scopeID),
		IncludeRevoked: opts.includeRevoked,
	})
	if err != nil {
		return fmt.Errorf("list capability leases: %w", err)
	}
	return writeCoordinationLeaseList(stdout, time.Now().UTC(), leases)
}

// runLeaseIssue issues one capability lease and writes it in a human-readable operator view.
func runLeaseIssue(ctx context.Context, svc *app.Service, cfg config.Config, opts leaseIssueCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	if err := requireProjectID("lease issue", opts.projectID); err != nil {
		return err
	}
	if strings.TrimSpace(opts.agentName) == "" {
		return fmt.Errorf("--agent-name is required")
	}
	lease, err := svc.IssueCapabilityLease(ctx, app.IssueCapabilityLeaseInput{
		ProjectID:                 strings.TrimSpace(opts.projectID),
		ScopeType:                 domain.CapabilityScopeType(strings.TrimSpace(opts.scopeType)),
		ScopeID:                   strings.TrimSpace(opts.scopeID),
		Role:                      domain.CapabilityRole(strings.TrimSpace(opts.role)),
		AgentName:                 strings.TrimSpace(opts.agentName),
		AgentInstanceID:           strings.TrimSpace(opts.agentInstanceID),
		ParentInstanceID:          strings.TrimSpace(opts.parentInstanceID),
		AllowEqualScopeDelegation: opts.allowEqualScopeDelegation,
		RequestedTTL:              opts.requestedTTL,
		OverrideToken:             strings.TrimSpace(opts.overrideToken),
	})
	if err != nil {
		return fmt.Errorf("issue capability lease: %w", err)
	}
	return writeCoordinationLeaseDetail(stdout, time.Now().UTC(), lease)
}

// runLeaseHeartbeat refreshes one capability lease heartbeat and writes it in a human-readable operator view.
func runLeaseHeartbeat(ctx context.Context, svc *app.Service, cfg config.Config, opts leaseHeartbeatCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	lease, err := svc.HeartbeatCapabilityLease(ctx, app.HeartbeatCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(opts.agentInstanceID),
		LeaseToken:      strings.TrimSpace(opts.leaseToken),
	})
	if err != nil {
		return fmt.Errorf("heartbeat capability lease: %w", err)
	}
	return writeCoordinationLeaseDetail(stdout, time.Now().UTC(), lease)
}

// runLeaseRenew renews one capability lease and writes it in a human-readable operator view.
func runLeaseRenew(ctx context.Context, svc *app.Service, cfg config.Config, opts leaseRenewCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	lease, err := svc.RenewCapabilityLease(ctx, app.RenewCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(opts.agentInstanceID),
		LeaseToken:      strings.TrimSpace(opts.leaseToken),
		TTL:             opts.ttl,
	})
	if err != nil {
		return fmt.Errorf("renew capability lease: %w", err)
	}
	return writeCoordinationLeaseDetail(stdout, time.Now().UTC(), lease)
}

// runLeaseRevoke revokes one capability lease and writes it in a human-readable operator view.
func runLeaseRevoke(ctx context.Context, svc *app.Service, cfg config.Config, opts leaseRevokeCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	lease, err := svc.RevokeCapabilityLease(ctx, app.RevokeCapabilityLeaseInput{
		AgentInstanceID: strings.TrimSpace(opts.agentInstanceID),
		Reason:          strings.TrimSpace(opts.reason),
	})
	if err != nil {
		return fmt.Errorf("revoke capability lease: %w", err)
	}
	return writeCoordinationLeaseDetail(stdout, time.Now().UTC(), lease)
}

// runLeaseRevokeAll revokes every capability lease within one scope and writes a human-readable summary.
func runLeaseRevokeAll(ctx context.Context, svc *app.Service, cfg config.Config, opts leaseRevokeAllCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	if err := requireProjectID("lease revoke-all", opts.projectID); err != nil {
		return err
	}
	if err := svc.RevokeAllCapabilityLeases(ctx, app.RevokeAllCapabilityLeasesInput{
		ProjectID: strings.TrimSpace(opts.projectID),
		ScopeType: domain.CapabilityScopeType(strings.TrimSpace(opts.scopeType)),
		ScopeID:   strings.TrimSpace(opts.scopeID),
		Reason:    strings.TrimSpace(opts.reason),
	}); err != nil {
		return fmt.Errorf("revoke all capability leases: %w", err)
	}
	return writeCoordinationLeaseRevocationSummary(
		stdout,
		strings.TrimSpace(opts.projectID),
		domain.CapabilityScopeType(strings.TrimSpace(opts.scopeType)),
		strings.TrimSpace(opts.scopeID),
		strings.TrimSpace(opts.reason),
	)
}

// runHandoffCreate creates one durable handoff and writes it in a human-readable operator view.
func runHandoffCreate(ctx context.Context, svc *app.Service, cfg config.Config, opts handoffCreateCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	if err := requireProjectID("handoff create", opts.projectID); err != nil {
		return err
	}
	if strings.TrimSpace(opts.summary) == "" {
		return fmt.Errorf("--summary is required")
	}
	handoff, err := svc.CreateHandoff(ctx, app.CreateHandoffInput{
		Level: domain.LevelTupleInput{
			ProjectID: strings.TrimSpace(opts.projectID),
			BranchID:  strings.TrimSpace(opts.branchID),
			ScopeType: domain.ScopeLevel(strings.TrimSpace(opts.scopeType)),
			ScopeID:   strings.TrimSpace(opts.scopeID),
		},
		SourceRole:      strings.TrimSpace(opts.sourceRole),
		TargetBranchID:  strings.TrimSpace(opts.targetBranchID),
		TargetScopeType: domain.ScopeLevel(strings.TrimSpace(opts.targetScopeType)),
		TargetScopeID:   strings.TrimSpace(opts.targetScopeID),
		TargetRole:      strings.TrimSpace(opts.targetRole),
		Status:          domain.HandoffStatus(strings.TrimSpace(opts.status)),
		Summary:         strings.TrimSpace(opts.summary),
		NextAction:      strings.TrimSpace(opts.nextAction),
		MissingEvidence: append([]string(nil), opts.missingEvidence...),
		RelatedRefs:     append([]string(nil), opts.relatedRefs...),
		CreatedBy:       cliMutationActorID(cfg),
		CreatedType:     cliMutationActorType(cfg),
	})
	if err != nil {
		return fmt.Errorf("create handoff: %w", err)
	}
	return writeCoordinationHandoffDetail(stdout, handoff)
}

// runHandoffGet returns one durable handoff in a human-readable operator view.
func runHandoffGet(ctx context.Context, svc *app.Service, opts handoffGetCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	handoff, err := svc.GetHandoff(ctx, strings.TrimSpace(opts.handoffID))
	if err != nil {
		return fmt.Errorf("get handoff: %w", err)
	}
	return writeCoordinationHandoffDetail(stdout, handoff)
}

// runHandoffList lists durable handoffs in a human-readable operator view.
func runHandoffList(ctx context.Context, svc *app.Service, opts handoffListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	if err := requireProjectID("handoff list", opts.projectID); err != nil {
		return err
	}
	handoffs, err := svc.ListHandoffs(ctx, app.ListHandoffsInput{
		Level: domain.LevelTupleInput{
			ProjectID: strings.TrimSpace(opts.projectID),
			BranchID:  strings.TrimSpace(opts.branchID),
			ScopeType: domain.ScopeLevel(strings.TrimSpace(opts.scopeType)),
			ScopeID:   strings.TrimSpace(opts.scopeID),
		},
		Statuses: toHandoffStatusList(opts.statuses),
		Limit:    opts.limit,
	})
	if err != nil {
		return fmt.Errorf("list handoffs: %w", err)
	}
	return writeCoordinationHandoffList(stdout, handoffs)
}

// runHandoffUpdate updates one durable handoff and writes it in a human-readable operator view.
func runHandoffUpdate(ctx context.Context, svc *app.Service, cfg config.Config, opts handoffUpdateCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	ctx = cliMutationContext(ctx, cfg)
	handoff, err := svc.UpdateHandoff(ctx, app.UpdateHandoffInput{
		HandoffID:       strings.TrimSpace(opts.handoffID),
		Status:          domain.HandoffStatus(strings.TrimSpace(opts.status)),
		SourceRole:      strings.TrimSpace(opts.sourceRole),
		TargetBranchID:  strings.TrimSpace(opts.targetBranchID),
		TargetScopeType: domain.ScopeLevel(strings.TrimSpace(opts.targetScopeType)),
		TargetScopeID:   strings.TrimSpace(opts.targetScopeID),
		TargetRole:      strings.TrimSpace(opts.targetRole),
		Summary:         strings.TrimSpace(opts.summary),
		NextAction:      strings.TrimSpace(opts.nextAction),
		MissingEvidence: append([]string(nil), opts.missingEvidence...),
		RelatedRefs:     append([]string(nil), opts.relatedRefs...),
		ResolutionNote:  strings.TrimSpace(opts.resolutionNote),
		UpdatedBy:       cliMutationActorID(cfg),
		UpdatedType:     cliMutationActorType(cfg),
		ResolvedBy:      cliMutationActorID(cfg),
		ResolvedType:    cliMutationActorType(cfg),
	})
	if err != nil {
		return fmt.Errorf("update handoff: %w", err)
	}
	return writeCoordinationHandoffDetail(stdout, handoff)
}

// runAuthRequestCreate creates one persisted auth request and mirrors it into notifications.
func runAuthRequestCreate(ctx context.Context, svc *app.Service, cfg config.Config, opts requestCreateCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	continuation, err := parseCLIContinuationJSON(opts.continuation)
	if err != nil {
		return err
	}
	actorID, actorType := cliMutationActor(cfg)
	request, err := svc.CreateAuthRequest(ctx, app.CreateAuthRequestInput{
		Path:                strings.TrimSpace(opts.path),
		PrincipalID:         strings.TrimSpace(opts.principalID),
		PrincipalType:       strings.TrimSpace(opts.principalType),
		PrincipalRole:       strings.TrimSpace(opts.principalRole),
		PrincipalName:       strings.TrimSpace(opts.principalName),
		ClientID:            strings.TrimSpace(opts.clientID),
		ClientType:          strings.TrimSpace(opts.clientType),
		ClientName:          strings.TrimSpace(opts.clientName),
		RequestedSessionTTL: opts.ttl,
		Reason:              strings.TrimSpace(opts.reason),
		Continuation:        continuation,
		RequestedBy:         actorID,
		RequestedType:       actorType,
		Timeout:             opts.timeout,
	})
	if err != nil {
		return fmt.Errorf("create auth request: %w", err)
	}
	return writeAuthRequestResultHuman(stdout, authRequestPayload(request, ""))
}

// parseCLIContinuationJSON validates one optional CLI continuation JSON object string.
func parseCLIContinuationJSON(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("parse --continuation-json: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("parse --continuation-json: continuation metadata must be a non-empty JSON object")
	}
	return out, nil
}

// parseOptionalKindTemplateJSON parses one optional kind template JSON document.
func parseOptionalKindTemplateJSON(raw string) (domain.KindTemplate, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return domain.KindTemplate{}, nil
	}
	var template domain.KindTemplate
	if err := json.Unmarshal([]byte(raw), &template); err != nil {
		return domain.KindTemplate{}, fmt.Errorf("parse --template-json: %w", err)
	}
	return template, nil
}

// parseTemplateLibrarySpecJSON parses one template-library JSON transport spec.
func parseTemplateLibrarySpecJSON(raw string) (servercommon.UpsertTemplateLibraryRequest, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return servercommon.UpsertTemplateLibraryRequest{}, fmt.Errorf("--spec-json is required")
	}
	var spec servercommon.UpsertTemplateLibraryRequest
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return servercommon.UpsertTemplateLibraryRequest{}, fmt.Errorf("parse --spec-json: %w", err)
	}
	return spec, nil
}

// cliMutationContext attaches a deterministic CLI mutation actor to context.
func cliMutationContext(ctx context.Context, cfg config.Config) context.Context {
	actorID, actorType := cliMutationActor(cfg)
	return app.WithMutationActor(ctx, app.MutationActor{
		ActorID:   actorID,
		ActorName: strings.TrimSpace(cfg.Identity.DisplayName),
		ActorType: actorType,
	})
}

// cliMutationActorID returns the resolved CLI mutation actor id.
func cliMutationActorID(cfg config.Config) string {
	actorID, _ := cliMutationActor(cfg)
	return actorID
}

// cliMutationActorType returns the resolved CLI mutation actor type.
func cliMutationActorType(cfg config.Config) domain.ActorType {
	_, actorType := cliMutationActor(cfg)
	return actorType
}

// toKindAppliesToList converts CLI string values into canonical kind applies-to values.
func toKindAppliesToList(values []string) []domain.KindAppliesTo {
	out := make([]domain.KindAppliesTo, 0, len(values))
	for _, value := range values {
		scope := domain.KindAppliesTo(strings.TrimSpace(strings.ToLower(value)))
		if scope == "" {
			continue
		}
		out = append(out, scope)
	}
	return out
}

// toKindIDList converts CLI string values into canonical kind identifiers.
func toKindIDList(values []string) []domain.KindID {
	out := make([]domain.KindID, 0, len(values))
	for _, value := range values {
		kindID := domain.KindID(strings.TrimSpace(value))
		if kindID == "" {
			continue
		}
		out = append(out, kindID)
	}
	return out
}

// toStrings converts one typed slice into its string representation.
func toStrings[T ~string](values []T) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return out
}

// toHandoffStatusList converts CLI string values into canonical handoff statuses.
func toHandoffStatusList(values []string) []domain.HandoffStatus {
	out := make([]domain.HandoffStatus, 0, len(values))
	for _, value := range values {
		status := domain.NormalizeHandoffStatus(domain.HandoffStatus(strings.TrimSpace(value)))
		if status == "" {
			continue
		}
		out = append(out, status)
	}
	return out
}

// runAuthRequestList lists persisted auth requests in a human-readable operator view.
func runAuthRequestList(ctx context.Context, svc *app.Service, opts requestListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	requests, err := svc.ListAuthRequests(ctx, domain.AuthRequestListFilter{
		ProjectID: strings.TrimSpace(opts.projectID),
		State:     domain.AuthRequestState(strings.TrimSpace(opts.state)),
		Limit:     opts.limit,
	})
	if err != nil {
		return fmt.Errorf("list auth requests: %w", err)
	}
	payload := make([]authRequestPayloadJSON, 0, len(requests))
	for _, request := range requests {
		payload = append(payload, authRequestPayload(request, ""))
	}
	return writeAuthRequestListHuman(stdout, payload)
}

// runAuthRequestShow returns one auth request by id in a human-readable operator view.
func runAuthRequestShow(ctx context.Context, svc *app.Service, opts requestShowCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	request, err := svc.GetAuthRequest(ctx, strings.TrimSpace(opts.requestID))
	if err != nil {
		return fmt.Errorf("get auth request: %w", err)
	}
	return writeAuthRequestDetailHuman(stdout, authRequestPayload(request, ""))
}

// runAuthRequestApprove approves one pending auth request and issues one usable session.
func runAuthRequestApprove(ctx context.Context, svc *app.Service, cfg config.Config, opts requestResolveCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	actorID, actorType := cliMutationActor(cfg)
	approved, err := svc.ApproveAuthRequest(ctx, app.ApproveAuthRequestInput{
		RequestID:      strings.TrimSpace(opts.requestID),
		Path:           strings.TrimSpace(opts.path),
		SessionTTL:     opts.ttl,
		ResolvedBy:     actorID,
		ResolvedType:   actorType,
		ResolutionNote: strings.TrimSpace(opts.note),
	})
	if err != nil {
		return fmt.Errorf("approve auth request: %w", err)
	}
	return writeAuthRequestResultHuman(stdout, authRequestPayload(approved.Request, approved.SessionSecret))
}

// runAuthRequestDeny denies one pending auth request.
func runAuthRequestDeny(ctx context.Context, svc *app.Service, cfg config.Config, opts requestResolveCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	actorID, actorType := cliMutationActor(cfg)
	request, err := svc.DenyAuthRequest(ctx, app.DenyAuthRequestInput{
		RequestID:      strings.TrimSpace(opts.requestID),
		ResolvedBy:     actorID,
		ResolvedType:   actorType,
		ResolutionNote: strings.TrimSpace(opts.note),
	})
	if err != nil {
		return fmt.Errorf("deny auth request: %w", err)
	}
	return writeAuthRequestResultHuman(stdout, authRequestPayload(request, ""))
}

// runAuthRequestCancel cancels one pending auth request.
func runAuthRequestCancel(ctx context.Context, svc *app.Service, cfg config.Config, opts requestResolveCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	actorID, actorType := cliMutationActor(cfg)
	request, err := svc.CancelAuthRequest(ctx, app.CancelAuthRequestInput{
		RequestID:      strings.TrimSpace(opts.requestID),
		ResolvedBy:     actorID,
		ResolvedType:   actorType,
		ResolutionNote: strings.TrimSpace(opts.note),
	})
	if err != nil {
		return fmt.Errorf("cancel auth request: %w", err)
	}
	return writeAuthRequestResultHuman(stdout, authRequestPayload(request, ""))
}

// runAuthSessionList returns caller-safe auth-session inventory in a human-readable operator view.
func runAuthSessionList(ctx context.Context, svc *app.Service, opts sessionListCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	sessions, err := svc.ListAuthSessions(ctx, app.AuthSessionFilter{
		SessionID:   strings.TrimSpace(opts.sessionID),
		ProjectID:   strings.TrimSpace(opts.projectID),
		PrincipalID: strings.TrimSpace(opts.principalID),
		ClientID:    strings.TrimSpace(opts.clientID),
		State:       strings.TrimSpace(opts.state),
		Limit:       opts.limit,
	})
	if err != nil {
		return fmt.Errorf("list auth sessions: %w", err)
	}
	payload := make([]authSessionPayloadJSON, 0, len(sessions))
	for _, session := range sessions {
		payload = append(payload, authSessionPayload(session))
	}
	return writeAuthSessionListHuman(stdout, payload)
}

// runAuthSessionValidate validates one session credential pair.
func runAuthSessionValidate(ctx context.Context, svc *app.Service, opts sessionValidateCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	validated, err := svc.ValidateAuthSession(ctx, strings.TrimSpace(opts.sessionID), strings.TrimSpace(opts.sessionSecret))
	if err != nil {
		return fmt.Errorf("validate auth session: %w", err)
	}
	return writeAuthSessionDetailHuman(stdout, authSessionPayload(validated.Session), "")
}

// runAuthSessionRevoke revokes one auth session through the app-facing backend.
func runAuthSessionRevoke(ctx context.Context, svc *app.Service, opts revokeSessionCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	session, err := svc.RevokeAuthSession(ctx, strings.TrimSpace(opts.sessionID), strings.TrimSpace(opts.reason))
	if err != nil {
		return fmt.Errorf("revoke auth session: %w", err)
	}
	return writeAuthSessionDetailHuman(stdout, authSessionPayload(session), "")
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

// kindDefinitionPayloadJSON stores JSON-friendly kind-definition output fields.
type kindDefinitionPayloadJSON struct {
	ID                  string              `json:"id"`
	DisplayName         string              `json:"display_name"`
	DescriptionMarkdown string              `json:"description_markdown,omitempty"`
	AppliesTo           []string            `json:"applies_to"`
	AllowedParentScopes []string            `json:"allowed_parent_scopes,omitempty"`
	PayloadSchemaJSON   string              `json:"payload_schema_json,omitempty"`
	Template            domain.KindTemplate `json:"template"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
	ArchivedAt          *time.Time          `json:"archived_at,omitempty"`
}

// kindAllowlistPayloadJSON stores JSON-friendly project allowlist fields.
type kindAllowlistPayloadJSON struct {
	ProjectID string   `json:"project_id"`
	KindIDs   []string `json:"kind_ids"`
}

// authRequestPayloadJSON stores JSON-friendly auth-request output fields.
type authRequestPayloadJSON struct {
	ID                     string     `json:"id"`
	State                  string     `json:"state"`
	Path                   string     `json:"path"`
	ApprovedPath           string     `json:"approved_path,omitempty"`
	ProjectID              string     `json:"project_id"`
	BranchID               string     `json:"branch_id,omitempty"`
	PhaseIDs               []string   `json:"phase_ids,omitempty"`
	ScopeType              string     `json:"scope_type"`
	ScopeID                string     `json:"scope_id"`
	PrincipalID            string     `json:"principal_id"`
	PrincipalType          string     `json:"principal_type"`
	PrincipalRole          string     `json:"principal_role,omitempty"`
	PrincipalName          string     `json:"principal_name,omitempty"`
	ClientID               string     `json:"client_id"`
	ClientType             string     `json:"client_type"`
	ClientName             string     `json:"client_name,omitempty"`
	RequestedSessionTTL    string     `json:"requested_session_ttl"`
	ApprovedSessionTTL     string     `json:"approved_session_ttl,omitempty"`
	Reason                 string     `json:"reason,omitempty"`
	HasContinuation        bool       `json:"has_continuation,omitempty"`
	RequestedByActor       string     `json:"requested_by_actor"`
	RequestedByType        string     `json:"requested_by_type"`
	CreatedAt              time.Time  `json:"created_at"`
	ExpiresAt              time.Time  `json:"expires_at"`
	ResolvedByActor        string     `json:"resolved_by_actor,omitempty"`
	ResolvedByType         string     `json:"resolved_by_type,omitempty"`
	ResolvedAt             *time.Time `json:"resolved_at,omitempty"`
	ResolutionNote         string     `json:"resolution_note,omitempty"`
	IssuedSessionID        string     `json:"issued_session_id,omitempty"`
	IssuedSessionSecret    string     `json:"issued_session_secret,omitempty"`
	IssuedSessionExpiresAt *time.Time `json:"issued_session_expires_at,omitempty"`
}

// authSessionPayloadJSON stores JSON-friendly auth-session output fields.
type authSessionPayloadJSON struct {
	SessionID        string     `json:"session_id"`
	State            string     `json:"state"`
	ProjectID        string     `json:"project_id,omitempty"`
	AuthRequestID    string     `json:"auth_request_id,omitempty"`
	ApprovedPath     string     `json:"approved_path,omitempty"`
	PrincipalID      string     `json:"principal_id"`
	PrincipalType    string     `json:"principal_type,omitempty"`
	PrincipalRole    string     `json:"principal_role,omitempty"`
	PrincipalName    string     `json:"principal_name,omitempty"`
	ClientID         string     `json:"client_id"`
	ClientType       string     `json:"client_type,omitempty"`
	ClientName       string     `json:"client_name,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason string     `json:"revocation_reason,omitempty"`
}

// writeJSON renders one stable indented JSON payload followed by a trailing newline.
func writeJSON(stdout io.Writer, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json output: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
		return fmt.Errorf("write json output: %w", err)
	}
	return nil
}

// cliMutationActor resolves deterministic CLI mutation attribution from persisted identity defaults.
func cliMutationActor(cfg config.Config) (string, domain.ActorType) {
	actorID := strings.TrimSpace(cfg.Identity.ActorID)
	if actorID == "" {
		actorID = "tillsyn-user"
	}
	actorType := domain.ActorType(strings.TrimSpace(strings.ToLower(cfg.Identity.DefaultActorType)))
	switch actorType {
	case domain.ActorTypeAgent, domain.ActorTypeUser, domain.ActorTypeSystem:
	default:
		actorType = domain.ActorTypeUser
	}
	return actorID, actorType
}

// authRequestPayload maps one domain auth-request row into stable CLI JSON output.
func authRequestPayload(request domain.AuthRequest, sessionSecret string) authRequestPayloadJSON {
	sessionSecret = strings.TrimSpace(sessionSecret)
	approvedPath := strings.TrimSpace(request.ApprovedPath)
	approvedSessionTTL := ""
	if request.ApprovedSessionTTL > 0 {
		approvedSessionTTL = request.ApprovedSessionTTL.String()
	}
	return authRequestPayloadJSON{
		ID:                     request.ID,
		State:                  string(request.State),
		Path:                   request.Path,
		ApprovedPath:           approvedPath,
		ProjectID:              request.ProjectID,
		BranchID:               request.BranchID,
		PhaseIDs:               append([]string(nil), request.PhaseIDs...),
		ScopeType:              string(request.ScopeType),
		ScopeID:                request.ScopeID,
		PrincipalID:            request.PrincipalID,
		PrincipalType:          request.PrincipalType,
		PrincipalRole:          request.PrincipalRole,
		PrincipalName:          request.PrincipalName,
		ClientID:               request.ClientID,
		ClientType:             request.ClientType,
		ClientName:             request.ClientName,
		RequestedSessionTTL:    request.RequestedSessionTTL.String(),
		ApprovedSessionTTL:     approvedSessionTTL,
		Reason:                 request.Reason,
		HasContinuation:        len(request.Continuation) > 0,
		RequestedByActor:       request.RequestedByActor,
		RequestedByType:        string(request.RequestedByType),
		CreatedAt:              request.CreatedAt.UTC(),
		ExpiresAt:              request.ExpiresAt.UTC(),
		ResolvedByActor:        request.ResolvedByActor,
		ResolvedByType:         string(request.ResolvedByType),
		ResolvedAt:             request.ResolvedAt,
		ResolutionNote:         request.ResolutionNote,
		IssuedSessionID:        request.IssuedSessionID,
		IssuedSessionSecret:    sessionSecret,
		IssuedSessionExpiresAt: request.IssuedSessionExpiresAt,
	}
}

// authSessionPayload maps one app-facing session row into stable CLI JSON output.
func authSessionPayload(session app.AuthSession) authSessionPayloadJSON {
	return authSessionPayloadJSON{
		SessionID:        session.SessionID,
		State:            authSessionState(session, time.Now().UTC()),
		ProjectID:        session.ProjectID,
		AuthRequestID:    session.AuthRequestID,
		ApprovedPath:     session.ApprovedPath,
		PrincipalID:      session.PrincipalID,
		PrincipalType:    session.PrincipalType,
		PrincipalRole:    session.PrincipalRole,
		PrincipalName:    session.PrincipalName,
		ClientID:         session.ClientID,
		ClientType:       session.ClientType,
		ClientName:       session.ClientName,
		ExpiresAt:        session.ExpiresAt.UTC(),
		RevokedAt:        session.RevokedAt,
		RevocationReason: session.RevocationReason,
	}
}

// authSessionState normalizes one auth session into the user-facing lifecycle label.
func authSessionState(session app.AuthSession, now time.Time) string {
	if session.RevokedAt != nil {
		return "revoked"
	}
	if !session.ExpiresAt.IsZero() && !now.Before(session.ExpiresAt.UTC()) {
		return "expired"
	}
	return "active"
}

// kindDefinitionPayload maps one domain kind definition into stable CLI JSON output.
func kindDefinitionPayload(kind domain.KindDefinition) kindDefinitionPayloadJSON {
	return kindDefinitionPayloadJSON{
		ID:                  string(kind.ID),
		DisplayName:         kind.DisplayName,
		DescriptionMarkdown: kind.DescriptionMarkdown,
		AppliesTo:           toStrings(kind.AppliesTo),
		AllowedParentScopes: toStrings(kind.AllowedParentScopes),
		PayloadSchemaJSON:   kind.PayloadSchemaJSON,
		Template:            kind.Template,
		CreatedAt:           kind.CreatedAt.UTC(),
		UpdatedAt:           kind.UpdatedAt.UTC(),
		ArchivedAt:          kind.ArchivedAt,
	}
}

// cloneCLIObjectMap deep-copies optional CLI JSON metadata maps.
func cloneCLIObjectMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneCLIObjectValue(value)
	}
	return out
}

// cloneCLIObjectValue deep-copies one JSON-compatible CLI metadata value.
func cloneCLIObjectValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneCLIObjectMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneCLIObjectValue(item))
		}
		return out
	default:
		return typed
	}
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
func newRuntimeLogger(stderr io.Writer, appName string, devMode bool, cfg config.LoggingConfig, defaultLogDir string, now func() time.Time) (*runtimeLogger, error) {
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
	if !cfg.DevFile.Enabled {
		return logger, nil
	}

	devLogPath, err := devLogFilePath(cfg.DevFile.Dir, defaultLogDir, appName, now().UTC())
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

// DevLogPath returns the active runtime log file path configured through logging.dev_file.
func (l *runtimeLogger) DevLogPath() string {
	if l == nil {
		return ""
	}
	return l.devLog
}

// Close closes the optional runtime log-file sink.
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

// resolveRuntimeLogDir resolves the configured log directory against the shared runtime root.
func resolveRuntimeLogDir(configDir, defaultDir string) (string, error) {
	baseDir := strings.TrimSpace(configDir)
	if baseDir == "" || filepath.Clean(baseDir) == filepath.Clean(config.DefaultDevLogDir()) {
		baseDir = strings.TrimSpace(defaultDir)
	}
	if baseDir == "" {
		return "", fmt.Errorf("empty runtime log dir")
	}
	if !filepath.IsAbs(baseDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working dir: %w", err)
		}
		baseDir = filepath.Join(workspaceRootFrom(cwd), baseDir)
	}
	return filepath.Clean(baseDir), nil
}

// devLogFilePath resolves the runtime log file path for the current run day.
func devLogFilePath(configDir, defaultDir, appName string, now time.Time) (string, error) {
	baseDir, err := resolveRuntimeLogDir(configDir, defaultDir)
	if err != nil {
		return "", err
	}
	fileStem := sanitizeLogFileStem(appName)
	fileName := fmt.Sprintf("%s-%s.log", fileStem, now.Format("20060102"))
	return filepath.Join(baseDir, fileName), nil
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
