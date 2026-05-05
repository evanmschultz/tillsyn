package dispatcher

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"
)

// cli_adapter.go ships the cross-CLI canonical type vocabulary for the Drop 4c
// F.7.17 adapter seam. It is PURE TYPES — no behavior, no spawn logic, no
// claude-specific argv code. Concrete adapter implementations
// (cli_adapter_claude.go in droplet 4c.F.7.17.3 and the MockAdapter test
// fixture in droplet 4c.F.7.17.4) consume this file. Spawn prompt + spec live
// at workflow/drop_4c/F7_17_CLI_ADAPTER_PLAN.md (REVISIONS POST-AUTHORING
// supersedes body text — REV-1 dropped Command/ArgsPrefix; REV-5 renamed
// ExtractTerminalCost → ExtractTerminalReport).

// CLIKind names the CLI binary family the dispatcher routes a spawn to. It is
// a closed enum on string. Drop 4c ships CLIKindClaude only; Drop 4d adds
// CLIKindCodex. Empty-string semantics (default-to-claude per F.7.17 locked
// decision L15) are handled by the dispatcher's adapter-lookup path, NOT by
// IsValidCLIKind below — IsValidCLIKind returns false on the empty string so
// callers that need the closed-set check are explicit about defaults.
//
// Mirrored as a free string on templates.AgentBinding.CLIKind because the
// templates package must accept arbitrary future kinds at TOML Load time
// without knowing this dispatcher-side enum.
type CLIKind string

// CLIKindClaude is the only CLI kind shipped in Drop 4c. The adapter
// implementation lives in cli_adapter_claude.go (droplet 4c.F.7.17.3).
const CLIKindClaude CLIKind = "claude"

// IsValidCLIKind reports whether k is a member of the closed CLIKind enum
// shipped in this build. It returns true ONLY for kinds with a registered
// adapter. Today that is CLIKindClaude alone; CLIKindCodex lands in Drop 4d.
//
// Empty string is NOT a valid CLIKind for IsValidCLIKind purposes — callers
// who want the F.7.17 L15 default-to-claude semantics must apply the default
// before calling this function.
func IsValidCLIKind(k CLIKind) bool {
	switch k {
	case CLIKindClaude:
		return true
	default:
		return false
	}
}

// CLIAdapter is the seam between the dispatcher and one specific CLI binary
// (claude, codex, …). Per F.7.17 locked decision L10 it has exactly three
// methods so adapters stay narrow and testable; richer adapter behavior
// (mid-stream tool denials, prompt rewriting, sandbox tweaks) lives inside
// each adapter under its own private types.
//
// The interface intentionally does NOT cover bundle materialization — that is
// owned by F.7.1 (the other planner) and reaches each adapter through
// BundlePaths.Root. Adapters that need CLI-specific subdirs compute them
// under paths.Root themselves; nothing in this interface forces that.
type CLIAdapter interface {
	// BuildCommand assembles the *exec.Cmd that runs the CLI for one spawn.
	// Per F.7.17 locked decision L8 the implementation MUST set cmd.Env
	// explicitly — os.Environ() is NOT inherited.
	BuildCommand(ctx context.Context, binding BindingResolved, paths BundlePaths) (*exec.Cmd, error)

	// ParseStreamEvent decodes one JSONL line emitted by the CLI's
	// stream-json output channel into the cross-CLI canonical StreamEvent
	// shape. Adapters retain the raw JSON in the returned event so
	// ExtractTerminalReport can re-decode adapter-private fields without
	// re-reading from disk.
	ParseStreamEvent(line []byte) (StreamEvent, error)

	// ExtractTerminalReport pulls the terminal report (cost, denials,
	// reason, errors) out of a parsed StreamEvent. The bool return is true
	// when ev IS the terminal event, false otherwise — non-terminal events
	// always return (TerminalReport{}, false). Per F.7.17 locked decision
	// L11 the cost field is *float64 so adapters whose CLI does not emit
	// cost telemetry signal absence cleanly.
	//
	// Renamed from ExtractTerminalCost per Drop 4c F.7.17 REV-5.
	ExtractTerminalReport(ev StreamEvent) (TerminalReport, bool)
}

// BindingResolved is the flat resolved per-spawn binding the dispatcher hands
// to CLIAdapter.BuildCommand. It is the OUTPUT of the priority-cascade
// resolver shipped in droplet 4c.F.7.17.8 (CLI > MCP > TUI > template TOML >
// absent per F.7.17 locked decision L16). Adapters do NOT re-resolve; they
// consume this struct as-is.
//
// REV-1 supersession: BindingResolved does NOT carry Command []string or
// ArgsPrefix []string. The wrapper-interop knob is gone from Tillsyn;
// adapters invoke their CLI binary directly (claude / codex). Process
// isolation is an OS-level concern (PATH-shadowed shim, container, sandbox).
//
// Pointer-typed fields distinguish "absent" (the lower-priority layer is
// authoritative or no value was specified) from "explicit zero" per F.7.17
// locked decision L9. AgentName, Tools, ToolsAllowed, ToolsDisallowed, Env,
// and CLIKind use value/slice types because their zero values (empty string,
// nil slice) ARE the identity element — no absent vs explicit distinction is
// meaningful.
type BindingResolved struct {
	// AgentName is the canonical agent identifier the dispatcher resolves
	// to a concrete subagent specification (e.g. "go-builder-agent"). The
	// resolver always populates this — it has no sensible "absent" form.
	AgentName string

	// CLIKind selects which CLI adapter the dispatcher routes the spawn to.
	// Resolver applies the F.7.17 L15 default-to-claude rule before
	// populating this field — adapters never see the empty-string sentinel.
	CLIKind CLIKind

	// Env is the per-binding allow-list of environment-variable NAMES the
	// adapter forwards from the orchestrator's process to the spawned
	// agent process. Per F.7.17 locked decision L4 each adapter resolves
	// the value via os.Getenv at BuildCommand time. Per L8 cmd.Env is set
	// explicitly to the closed POSIX baseline (L6) plus the resolved
	// values for every name in this slice — os.Environ() is NOT inherited.
	Env []string

	// Model is the LLM model identifier (e.g. "opus", "sonnet", "haiku").
	// Pointer-typed: nil means no override; the adapter falls back to its
	// CLI's default model.
	Model *string

	// Effort is the model effort tier (e.g. "low", "medium", "high").
	// Pointer-typed: nil means no override.
	Effort *string

	// Tools is the per-spawn allow-list propagated to the CLI's tool-gate
	// flag (claude: --tools). Nil means "use CLI default" — distinct from
	// an explicit empty slice which would mean "deny all tools" if the CLI
	// supports that semantic.
	Tools []string

	// ToolsAllowed and ToolsDisallowed are the CLI-flag-level allow/deny
	// pair some adapters need separately from the high-level Tools list.
	// Nil means "no override"; an empty slice means "explicit empty
	// allow/deny list" if the CLI distinguishes those.
	ToolsAllowed    []string
	ToolsDisallowed []string

	// MaxTries caps the number of dispatch attempts before the dispatcher
	// marks the action item failed. Pointer-typed because zero is not a
	// valid number-of-tries value (templates.AgentBinding.Validate rejects
	// MaxTries <= 0); a nil pointer means "use the dispatcher default."
	MaxTries *int

	// MaxBudgetUSD caps the per-spawn dollar budget propagated as the
	// claude --max-budget-usd flag value. Pointer-typed because explicit
	// zero may carry meaning ("no spend allowed") distinct from absent
	// ("dispatcher default").
	MaxBudgetUSD *float64

	// MaxTurns caps the conversation-turn count propagated as the claude
	// --max-turns flag value. Pointer-typed for absent vs explicit-zero.
	MaxTurns *int

	// AutoPush, when non-nil and true, instructs the post-build pipeline
	// to invoke `git push` after a successful build. Pointer-typed because
	// explicit-false ("never auto-push for this binding") is distinct
	// from absent ("inherit project default").
	AutoPush *bool

	// CommitAgent identifies the agent name (typically "commit-agent")
	// used to author commit messages. Pointer-typed for absent vs explicit
	// empty-string.
	CommitAgent *string

	// BlockedRetries caps how many times the dispatcher retries a spawn
	// that returned a "blocked" outcome. Pointer-typed for absent vs
	// explicit zero ("never retry blocked spawns").
	BlockedRetries *int

	// BlockedRetryCooldown is the wall-clock delay between blocked-retry
	// attempts. Pointer-typed for absent vs explicit zero ("retry
	// immediately").
	BlockedRetryCooldown *time.Duration
}

// BundlePaths is the claude-neutral handle the dispatcher hands to every
// adapter's BuildCommand. Per F.7.17 locked decision L13 it carries ONLY
// CLI-agnostic file locations; CLI-specific subdirs (claude's plugin/,
// .claude-plugin/, agents/, .mcp.json, settings.json) are NOT here. Adapters
// materialize their own subdirs under Root themselves so the seam stays
// narrow.
//
// The bundle root is owned by the dispatcher (typically
// os.TempDir()/tillsyn/<spawn-id>/ or <worktree>/.tillsyn/spawns/<spawn-id>/);
// adapters MUST NOT relocate Root or its subpaths.
type BundlePaths struct {
	// Root is the absolute path of the spawn's bundle directory. All other
	// paths in this struct are descendants of Root.
	Root string

	// SystemPromptPath is the absolute path of the system-prompt MD the
	// adapter passes to its CLI's --system-prompt-file (or equivalent)
	// flag. Conventional location: <Root>/system-prompt.md.
	SystemPromptPath string

	// SystemAppendPath is the absolute path of an optional appended-system
	// MD passed to --append-system-prompt-file. Empty when no append file
	// is configured. Conventional location: <Root>/system-append.md.
	SystemAppendPath string

	// StreamLogPath is the absolute path of the JSONL stream the CLI
	// writes to (or that the dispatcher tees from the CLI's stdout).
	// Conventional location: <Root>/stream.jsonl.
	StreamLogPath string

	// ManifestPath is the absolute path of the spawn's manifest.json (the
	// per-spawn record consumed by orphan-scan and post-mortem tooling).
	// Conventional location: <Root>/manifest.json.
	ManifestPath string

	// ContextDir is the absolute path of the per-spawn context staging
	// directory the F.7.18 context aggregator writes into. Conventional
	// location: <Root>/context/.
	ContextDir string
}

// StreamEvent is the minimal cross-CLI canonical shape produced by
// CLIAdapter.ParseStreamEvent. Per F.7.17 locked decision L14 the dispatcher
// only consumes Type / Subtype / IsTerminal / Text / ToolName /
// ToolInput / Raw; CLI-specific event-payload fields stay inside each
// adapter's private types. Adapters MAY decode Raw further when their
// own ExtractTerminalReport implementation needs adapter-private fields.
type StreamEvent struct {
	// Type is the adapter-mapped canonical event family name (e.g.
	// "system_init", "assistant", "user", "result"). Adapters translate
	// CLI-specific event-type strings into this normalized vocabulary so
	// the dispatcher's monitor stays CLI-agnostic.
	Type string

	// Subtype is an optional refinement of Type when an adapter's CLI
	// emits a sub-classification the dispatcher might want to surface
	// (e.g. claude's "init" subtype under the "system" type). Empty when
	// no refinement applies.
	Subtype string

	// IsTerminal is true when this event marks the end of the spawn's
	// event stream. ExtractTerminalReport returns (_, true) only on
	// events with IsTerminal == true.
	IsTerminal bool

	// Text is the event's primary text payload (e.g. an "assistant"
	// event's content body, or a "result" event's final-text field).
	// Empty for events that carry no text.
	Text string

	// ToolName is populated for events that represent a tool_use or
	// tool_result step (e.g. the agent invoking a tool). Empty otherwise.
	ToolName string

	// ToolInput is the raw JSON of the tool's input argument map. Opaque
	// to non-claude adapters; adapters that don't surface tool details
	// leave this nil.
	ToolInput json.RawMessage

	// Raw is the unmodified JSONL line the adapter parsed. Retained so
	// adapter-private decoding inside ExtractTerminalReport (or forensic
	// post-mortem tooling) can reach fields the canonical shape does not
	// expose without re-reading from disk.
	Raw json.RawMessage
}

// ToolDenial records one tool-call denial surfaced by the CLI's terminal
// report. The pair (ToolName, ToolInput) is enough for the dispatcher /
// monitor to log the denial and for downstream tooling to attribute it to a
// specific tool invocation.
type ToolDenial struct {
	// ToolName is the canonical name of the denied tool (e.g. "Bash",
	// "Edit"). Adapters normalize to a stable identifier so cross-CLI
	// comparison is meaningful.
	ToolName string

	// ToolInput is the raw JSON of the denied tool call's input arguments.
	// Opaque shape — varies per tool. Retained for log forensics.
	ToolInput json.RawMessage
}

// TerminalReport is the structured summary produced by
// CLIAdapter.ExtractTerminalReport from a terminal StreamEvent. Per F.7.17
// locked decision L11 Cost is *float64 so adapters whose CLI does not emit
// cost telemetry can return (TerminalReport{Cost: nil, ...}, true) without
// the caller mistaking absent-cost for zero-cost.
type TerminalReport struct {
	// Cost is the spawn's reported total spend in USD, when the CLI emits
	// it. Nil signals absence (the CLI has no cost channel, or the
	// terminal event lacked the field). Callers MUST NOT treat nil as 0.
	Cost *float64

	// Denials lists every tool-call denial the CLI surfaced for this
	// spawn. Nil and empty are equivalent — both mean "no denials." The
	// distinction is not load-bearing.
	Denials []ToolDenial

	// Reason is the CLI's terminal-state reason string (e.g. "completed",
	// "max_turns", "error"). Empty when the CLI did not report a reason.
	Reason string

	// Errors collects any spawn-level error strings the CLI reported in
	// the terminal event. Nil and empty are equivalent — both mean "no
	// errors." Specific error semantics are CLI-dependent.
	Errors []string
}
