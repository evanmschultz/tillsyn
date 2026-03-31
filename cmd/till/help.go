package main

import (
	"strings"

	"github.com/spf13/cobra"
)

// commandHelpSpec stores richer long-form help and examples for one command path.
type commandHelpSpec struct {
	Long    string
	Example []string
}

// commandHelpSpecs layers richer operator guidance onto the Cobra command tree.
var commandHelpSpecs = map[string]commandHelpSpec{
	"till": {
		Long: strings.TrimSpace(`
Run the Tillsyn TUI by default, or use subcommands for MCP, auth, projects,
templates, embeddings, leases, handoffs, snapshots, and runtime inspection.

Use --db and --config to point at one specific runtime when auditing or
scripting. Use subcommand help before mutation commands so the scope, required
flags, and next-step guidance stay explicit.
`),
		Example: []string{
			"  till",
			"  till h",
			"  till project list",
			"  till project create --name Inbox",
			"  till template library list --scope global --status approved",
			"  till embeddings status --cross-project",
			"  till auth request list --state pending",
		},
	},
	"till serve": {
		Long: strings.TrimSpace(`
Start the local HTTP API and streamable HTTP MCP endpoints for one Tillsyn
runtime.

Use this when a browser client, HTTP integration, or remote MCP client needs
to talk to the local store over HTTP rather than stdio.
`),
		Example: []string{
			"  till serve",
			"  till serve --http 127.0.0.1:4848",
			"  till serve --http 127.0.0.1:4848 --api-endpoint /api --mcp-endpoint /mcp",
		},
	},
	"till mcp": {
		Long: strings.TrimSpace(`
Start the raw stdio MCP runtime for local agent integrations.

Use this for direct Codex/Claude/Desktop-style local MCP sessions where Tillsyn
is the planning and coordination backend.

Start raw MCP over stdio when you want the direct local operator path.
`),
		Example: []string{
			"  till mcp",
			"  till --db /tmp/tillsyn.db --config /tmp/tillsyn.toml mcp",
		},
	},
	"till embeddings": {
		Example: []string{
			"  till embeddings status --cross-project",
			"  till embeddings status --project-id <project-id> --status failed",
			"  till embeddings reindex --project-id <project-id>",
			"  till embeddings reindex --cross-project --wait",
		},
	},
	"till embeddings status": {
		Long: strings.TrimSpace(`
Show one summary-first embeddings lifecycle view for the chosen scope.

Use this to inspect pending, running, ready, failed, or stale embedding rows
for projects, project documents, work items, and thread-context material.
`),
		Example: []string{
			"  till embeddings status --project-id <project-id>",
			"  till embeddings status --cross-project",
			"  till embeddings status --project-id <project-id> --status failed --status stale",
			"  till embeddings status --project-id <project-id> --limit 20",
		},
	},
	"till embeddings reindex": {
		Long: strings.TrimSpace(`
Enqueue one explicit embeddings backfill or reindex pass for the chosen scope.

Use this after enabling embeddings, changing providers/models, or when the
status view shows stale or failed lifecycle rows that need a fresh pass.
`),
		Example: []string{
			"  till embeddings reindex --project-id <project-id>",
			"  till embeddings reindex --project-id <project-id> --force",
			"  till embeddings reindex --cross-project --include-archived",
			"  till embeddings reindex --project-id <project-id> --wait --wait-timeout 30s",
		},
	},
	"till kind": {
		Long: strings.TrimSpace(`
Inspect kind definitions and project allowlists.

Template-library workflow contracts live under till template, not in the kind
registry.
`),
		Example: []string{
			"  till kind list",
			"  till kind upsert --id research-task --display-name \"Research Task\" --applies-to task",
			"  till kind allowlist list --project-id <project-id>",
			"  till kind allowlist set --project-id <project-id> --kind-id task --kind-id research-task",
		},
	},
	"till kind list": {
		Long: strings.TrimSpace(`
List the kind registry used to classify projects and work nodes.

Use this to inspect which kind ids exist, what scopes they apply to, and which
definitions are available before binding templates or tightening allowlists.

Discover valid kind ids here before template work, project creation, or task creation.
`),
		Example: []string{
			"  till kind list",
			"  till kind list --include-archived",
		},
	},
	"till kind upsert": {
		Long: strings.TrimSpace(`
Create or update one kind definition in the registry.

Kinds describe structural node identity and placement rules.

Template-library workflow contracts are managed separately under till template.

The hidden legacy '--template-json' flag remains compatibility-only and should
not be used for new work.
`),
		Example: []string{
			"  till kind upsert --id go-service --display-name \"Go Service\" --applies-to project",
			"  till kind upsert --id build-task --display-name \"Build Task\" --applies-to task --allowed-parent-scopes project --allowed-parent-scopes phase",
			"  till kind upsert --id qa-check --display-name \"QA Check\" --applies-to subtask --payload-schema-json '{\"type\":\"object\"}'",
		},
	},
	"till kind allowlist": {
		Long: strings.TrimSpace(`
Inspect or replace one project's explicit kind allowlist.

Use this when a project needs tighter kind governance than the global registry
alone provides.
`),
		Example: []string{
			"  till kind allowlist list --project-id <project-id>",
			"  till kind allowlist set --project-id <project-id> --kind-id task --kind-id qa-check",
		},
	},
	"till kind allowlist list": {
		Long: strings.TrimSpace(`
Show the explicit kind ids allowed for one project.

If no explicit allowlist exists, the project is effectively using the default
registry behavior.

Check this before changing template libraries or other project-level workflow
rules.
`),
		Example: []string{
			"  till kind allowlist list --project-id <project-id>",
		},
	},
	"till kind allowlist set": {
		Long: strings.TrimSpace(`
Replace the explicit kind allowlist for one project.

This is a replace operation. Pass every allowed kind id you want to
keep in the resulting project policy.
`),
		Example: []string{
			"  till kind allowlist set --project-id <project-id> --kind-id task --kind-id subtask",
			"  till kind allowlist set --project-id <project-id> --kind-id build-task --kind-id qa-check",
		},
	},
	"till template": {
		Example: []string{
			"  till template library list --scope global --status approved",
			"  till template library show --library-id <library-id>",
			"  till template project bind --project-id <project-id> --library-id <library-id>",
			"  till template contract show --node-id <task-id>",
		},
	},
	"till template library": {
		Long: strings.TrimSpace(`
Inspect or upsert SQLite-backed template libraries.

Template libraries define generated child work, actor-kind ownership, edit and
complete permissions, and truthful completion gates.
`),
		Example: []string{
			"  till template library list --scope global --status approved",
			"  till template library show --library-id <library-id>",
			"  till template library upsert --spec-json '{\"id\":\"<library-id>\",...}'",
		},
	},
	"till template library list": {
		Long: strings.TrimSpace(`
List template libraries by optional scope, project, or lifecycle status.

Use this to find approved global libraries before binding them to projects or
to inspect project-local or draft inventory.
`),
		Example: []string{
			"  till template library list",
			"  till template library list --scope global --status approved",
			"  till template library list --scope project --project-id <project-id>",
		},
	},
	"till template library show": {
		Long: strings.TrimSpace(`
Show one template library with its node templates and child rules.

This is the quickest operator view for verifying generated follow-up work,
responsible actor kinds, blocker rules, and the child-rule contract table before
binding the library.
`),
		Example: []string{
			"  till template library show --library-id <library-id>",
		},
	},
	"till template library upsert": {
		Example: []string{
			"  till template library upsert --spec-json '{\"id\":\"<library-id>\",\"scope\":\"global\",\"status\":\"approved\",\"node_templates\":[]}'",
			"  till template library upsert --spec-json \"$(cat /tmp/template-library.json)\"",
		},
	},
	"till template project": {
		Long: strings.TrimSpace(`
Bind projects to approved template libraries and inspect the active binding.

Use this when project creation did not already bind a library or when you need
to confirm which library currently governs generated workflow contracts.
`),
		Example: []string{
			"  till template project bind --project-id <project-id> --library-id <library-id>",
			"  till template project binding --project-id <project-id>",
		},
	},
	"till template project bind": {
		Long: strings.TrimSpace(`
Bind one project to one approved template library.

The binding becomes the project-level source for future generated workflow
contracts. Existing nodes keep their stored snapshots.
`),
		Example: []string{
			"  till template project bind --project-id <project-id> --library-id <library-id>",
		},
	},
	"till template project binding": {
		Long: strings.TrimSpace(`
Show the active template-library binding for one project.

Use this to confirm which approved library currently governs generated work for
future create-time template resolution.
`),
		Example: []string{
			"  till template project binding --project-id <project-id>",
		},
	},
	"till template contract": {
		Long: strings.TrimSpace(`
Inspect stored generated-node contract snapshots.

Node contracts preserve the resolved actor-kind rules and completion gates that
were applied when generated work was created.

This is the truthful runtime record for already-generated work.
`),
		Example: []string{
			"  till template contract show --node-id <task-id>",
		},
	},
	"till template contract show": {
		Long: strings.TrimSpace(`
Show one stored node-contract snapshot for generated work.

Use this to verify the generated node-contract snapshot: responsible actor kind,
edit and complete permissions, and whether the generated node blocks parent or
containing-scope completion.
`),
		Example: []string{
			"  till template contract show --node-id <task-id>",
		},
	},
	"till lease": {
		Example: []string{
			"  till lease list --project-id <project-id>",
			"  till lease issue --project-id <project-id> --agent-name <agent-name> --role builder",
			"  till lease renew --agent-instance-id <agent-instance-id> --lease-token <lease-token> --ttl 30m",
			"  till lease revoke-all --project-id <project-id> --reason operator_reset",
		},
	},
	"till lease list": {
		Long: strings.TrimSpace(`
List capability leases for one project scope.

Use this to inspect active or revoked actor assignments before issuing a new
lease or investigating stale orchestration state.
`),
		Example: []string{
			"  till lease list --project-id <project-id>",
			"  till lease list --project-id <project-id> --scope-type task --scope-id <task-id>",
			"  till lease list --project-id <project-id> --include-revoked",
		},
	},
	"till lease issue": {
		Long: strings.TrimSpace(`
Issue one scoped capability lease for an agent instance.

Use this when an orchestrator or operator needs to assign execution authority
for a project, branch, phase, task, or subtask scope.
`),
		Example: []string{
			"  till lease issue --project-id <project-id> --agent-name <agent-name> --role builder",
			"  till lease issue --project-id <project-id> --scope-type task --scope-id <task-id> --agent-name <agent-name> --role qa --requested-ttl 30m",
			"  till lease issue --project-id <project-id> --agent-name <agent-name> --role orchestrator --allow-equal-scope-delegation",
		},
	},
	"till lease heartbeat": {
		Long: strings.TrimSpace(`
Refresh the heartbeat timestamp for one existing lease.

Agents use this to prove liveness while holding scoped capability authority.
`),
		Example: []string{
			"  till lease heartbeat --agent-instance-id <agent-instance-id> --lease-token <lease-token>",
		},
	},
	"till lease renew": {
		Long: strings.TrimSpace(`
Renew one existing capability lease for an additional TTL.

Use this when valid work is still in progress and the current lease should stay
active rather than being reissued.
`),
		Example: []string{
			"  till lease renew --agent-instance-id <agent-instance-id> --lease-token <lease-token> --ttl 30m",
		},
	},
	"till lease revoke": {
		Long: strings.TrimSpace(`
Revoke one capability lease by agent instance id.

Use this to invalidate a single agent lease during recovery, reassignment, or
operator intervention.
`),
		Example: []string{
			"  till lease revoke --agent-instance-id <agent-instance-id> --reason operator_reset",
		},
	},
	"till lease revoke-all": {
		Long: strings.TrimSpace(`
Revoke every lease inside one chosen project scope.

Use this for broad recovery when a whole branch, phase, or project needs lease
state reset before work resumes.
`),
		Example: []string{
			"  till lease revoke-all --project-id <project-id> --reason operator_reset",
			"  till lease revoke-all --project-id <project-id> --scope-type branch --scope-id <branch-id> --reason branch_recovery",
		},
	},
	"till handoff": {
		Example: []string{
			"  till handoff create --project-id <project-id> --summary \"Builder blocked on QA\"",
			"  till handoff get --handoff-id <handoff-id>",
			"  till handoff list --project-id <project-id> --status open",
			"  till handoff update --handoff-id <handoff-id> --summary \"QA resumed\"",
		},
	},
	"till handoff create": {
		Long: strings.TrimSpace(`
Create one durable, structured handoff between humans or agents.

Use handoffs when simple comments are not enough and the next owner needs an
explicit summary, target scope, next action, or missing-evidence checklist.
`),
		Example: []string{
			"  till handoff create --project-id <project-id> --summary \"Builder blocked on QA\"",
			"  till handoff create --project-id <project-id> --scope-type task --scope-id <task-id> --source-role builder --target-role qa --next-action \"re-run verification\"",
			"  till handoff create --project-id <project-id> --summary \"Need review\" --missing-evidence test-output --related-ref <task-id>",
		},
	},
	"till handoff get": {
		Long: strings.TrimSpace(`
Show one durable handoff by id.

Use this when a handoff reference appears in comments, summaries, or audit
views and you need the full structured state.
`),
		Example: []string{
			"  till handoff get --handoff-id <handoff-id>",
		},
	},
	"till handoff list": {
		Long: strings.TrimSpace(`
List durable handoffs for one chosen scope.

Use filters to narrow by project, branch, scope type, scope id, or handoff
status when reviewing collaboration state.
`),
		Example: []string{
			"  till handoff list --project-id <project-id>",
			"  till handoff list --project-id <project-id> --scope-type task --scope-id <task-id>",
			"  till handoff list --project-id <project-id> --status open --status accepted --limit 20",
		},
	},
	"till handoff update": {
		Long: strings.TrimSpace(`
Update one existing durable handoff.

Use this to move the handoff forward, adjust target scope or role, revise the
summary, or record a final resolution note.
`),
		Example: []string{
			"  till handoff update --handoff-id <handoff-id> --summary \"QA resumed\"",
			"  till handoff update --handoff-id <handoff-id> --status accepted --target-role builder",
			"  till handoff update --handoff-id <handoff-id> --summary \"Complete\" --resolution-note \"validated and closed\"",
		},
	},
	"till export": {
		Long: strings.TrimSpace(`
Export the runtime store as one snapshot JSON payload.

Use this for migration, backup, debugging, or inspection of project, task,
template, and auth-related state in one portable artifact.
`),
		Example: []string{
			"  till export --out tillsyn-export.json",
			"  till export --out -",
			"  till export --out tillsyn-export.json --include-archived",
		},
	},
	"till import": {
		Long: strings.TrimSpace(`
Import one snapshot JSON payload into the runtime store.

Use this to restore or seed a local runtime from a known export artifact.
`),
		Example: []string{
			"  till import --in tillsyn-export.json",
		},
	},
	"till paths": {
		Long: strings.TrimSpace(`
Show the resolved runtime root, config, database, and log paths for this
invocation.

Use this when debugging path resolution, dev-mode separation, or one-off
runtime overrides from --db or --config.

This is the resolved runtime paths view for the current invocation.
`),
		Example: []string{
			"  till paths",
			"  till --dev paths",
			"  till --app tillsyn paths",
			"  till --db /tmp/tillsyn.db --config /tmp/tillsyn.toml paths",
		},
	},
	"till init-dev-config": {
		Long: strings.TrimSpace(`
Create the dev config file when missing and ensure the logging level is set to
debug for local development.

Use this when bootstrapping a fresh local runtime before serving MCP or HTTP
surfaces.
`),
		Example: []string{
			"  till init-dev-config",
			"  till --app tillsyn --dev init-dev-config",
		},
	},
}

// applyCommandHelp layers richer help specs and help-command aliases onto the command tree.
func applyCommandHelp(root *cobra.Command) {
	if root == nil {
		return
	}
	applyCommandHelpSpecs(root)
	installHelpAliases(root)
}

// applyCommandHelpSpecs updates command Long/Example fields from the static help-spec map.
func applyCommandHelpSpecs(root *cobra.Command) {
	walkCommands(root, func(cmd *cobra.Command) {
		spec, ok := commandHelpSpecs[cmd.CommandPath()]
		if !ok {
			return
		}
		if strings.TrimSpace(spec.Long) != "" {
			cmd.Long = strings.TrimSpace(spec.Long)
		}
		if len(spec.Example) > 0 {
			cmd.Example = strings.Join(spec.Example, "\n")
		}
	})
}

// walkCommands visits one command and every non-help descendant exactly once.
func walkCommands(root *cobra.Command, visit func(*cobra.Command)) {
	if root == nil || visit == nil {
		return
	}
	visit(root)
	for _, child := range root.Commands() {
		if child.Name() == "help" {
			continue
		}
		walkCommands(child, visit)
	}
}
