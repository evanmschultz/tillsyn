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
	"charm.land/lipgloss/v2"
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

	rootCmd := &cobra.Command{
		Use:           "till",
		Short:         "Local-first planning TUI with stdio MCP and HTTP adapters",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "serve", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "mcp", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.request.create", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.request.list", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.request.show", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.request.approve", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.request.deny", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.request.cancel", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.session.list", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.session.validate", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.session.revoke", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.issue-session", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "auth.revoke-session", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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
			return executeCommandFlow(cmd.Context(), "export", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
		},
	}
	exportCmd.Flags().StringVar(&exportOpts.outPath, "out", exportOpts.outPath, "Output file path ('-' for stdout)")
	exportCmd.Flags().BoolVar(&exportOpts.includeArchived, "include-archived", exportOpts.includeArchived, "Include archived projects/columns/tasks")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a snapshot JSON payload",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeCommandFlow(cmd.Context(), "import", rootOpts, serveOpts, mcpOpts, authOpts, issueSessionOpts, requestCreateOpts, requestListOpts, requestShowOpts, requestApproveOpts, requestDenyOpts, requestCancelOpts, sessionListOpts, sessionValidateOpts, revokeSessionOpts, exportOpts, importOpts, stdout, stderr)
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

	rootCmd.AddCommand(serveCmd, mcpCmd, authCmd, exportCmd, importCmd, pathsCmd, initDevConfigCmd)
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

// writeVersion writes the current CLI version to stdout.
func writeVersion(stdout io.Writer) error {
	if _, err := fmt.Fprintf(stdout, "till %s\n", version); err != nil {
		return fmt.Errorf("write version output: %w", err)
	}
	return nil
}

// writePathsOutput renders resolved paths using Fang-aligned styling.
func writePathsOutput(stdout io.Writer, opts rootCommandOptions, resolvedPaths resolvedRuntimePaths, rootDir, logDir string) error {
	if !supportsStyledOutputFunc(stdout) {
		return writePathsPlain(stdout, opts, resolvedPaths, rootDir, logDir)
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

	rows := buildPathsRows(opts, resolvedPaths, rootDir, logDir)

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

// writePathsPlain renders resolved paths in stable key/value text for scripts.
func writePathsPlain(stdout io.Writer, opts rootCommandOptions, resolvedPaths resolvedRuntimePaths, rootDir, logDir string) error {
	for _, row := range buildPathsRows(opts, resolvedPaths, rootDir, logDir) {
		if _, err := fmt.Fprintf(stdout, "%s: %s\n", row.key, row.value); err != nil {
			return fmt.Errorf("write paths %s output: %w", row.key, err)
		}
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
	logger.Debug("runtime paths resolved", "config_path", configPath, "root", rootDir, "db_path", dbPath, "logs_dir", logDir)
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
		AuthRequests:             authSvc,
		AuthBackend:              authSvc,
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
		if err := runServe(ctx, svc, authSvc, rootOpts.appName, serveOpts); err != nil {
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
		if err := runMCP(ctx, svc, authSvc, rootOpts.appName, serveOpts); err != nil {
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
	payload, err := json.MarshalIndent(struct {
		SessionID     string    `json:"session_id"`
		SessionSecret string    `json:"session_secret"`
		PrincipalID   string    `json:"principal_id"`
		PrincipalType string    `json:"principal_type"`
		PrincipalName string    `json:"principal_name"`
		ClientID      string    `json:"client_id"`
		ClientType    string    `json:"client_type"`
		ClientName    string    `json:"client_name"`
		ExpiresAt     time.Time `json:"expires_at"`
	}{
		SessionID:     issued.Session.ID,
		SessionSecret: issued.Secret,
		PrincipalID:   principalID,
		PrincipalType: strings.TrimSpace(opts.principalType),
		PrincipalName: firstNonEmpty(strings.TrimSpace(opts.principalName), principalID),
		ClientID:      strings.TrimSpace(opts.clientID),
		ClientType:    strings.TrimSpace(opts.clientType),
		ClientName:    firstNonEmpty(strings.TrimSpace(opts.clientName), strings.TrimSpace(opts.clientID)),
		ExpiresAt:     issued.Session.ExpiresAt.UTC(),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode issued auth session: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "%s\n", payload); err != nil {
		return fmt.Errorf("write issued auth session: %w", err)
	}
	return nil
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
	payload, err := json.MarshalIndent(struct {
		SessionID        string     `json:"session_id"`
		RevokedAt        *time.Time `json:"revoked_at,omitempty"`
		RevocationReason string     `json:"revocation_reason,omitempty"`
	}{
		SessionID:        session.ID,
		RevokedAt:        session.RevokedAt,
		RevocationReason: session.RevocationReason,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode revoked auth session: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "%s\n", payload); err != nil {
		return fmt.Errorf("write revoked auth session: %w", err)
	}
	return nil
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
	return writeJSON(stdout, authRequestPayload(request, ""))
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

// runAuthRequestList lists persisted auth requests in deterministic order.
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
	return writeJSON(stdout, payload)
}

// runAuthRequestShow returns one auth request by id.
func runAuthRequestShow(ctx context.Context, svc *app.Service, opts requestShowCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	request, err := svc.GetAuthRequest(ctx, strings.TrimSpace(opts.requestID))
	if err != nil {
		return fmt.Errorf("get auth request: %w", err)
	}
	return writeJSON(stdout, authRequestPayload(request, ""))
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
	return writeJSON(stdout, authRequestPayload(approved.Request, approved.SessionSecret))
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
	return writeJSON(stdout, authRequestPayload(request, ""))
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
	return writeJSON(stdout, authRequestPayload(request, ""))
}

// runAuthSessionList returns caller-safe auth-session inventory.
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
	return writeJSON(stdout, payload)
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
	return writeJSON(stdout, authSessionPayload(validated.Session))
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
	return writeJSON(stdout, authSessionPayload(session))
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
	if !devMode || !cfg.DevFile.Enabled {
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
