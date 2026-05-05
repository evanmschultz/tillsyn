package dispatcher

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// AuthBundle is the placeholder seam for the dispatcher's auth/lease plumbing.
//
// Wave 3: Wave 3 of Drop 4a (auth flow) replaces this stub with the populated
// form — session/lease IDs, MCP config materialization, capability scope. The
// Wave-2.6 spawner accepts a zero-value AuthBundle and emits an
// `--mcp-config` flag pointing at a deterministic placeholder path under
// `<project_root>/.tillsyn/`. Wave 3 will overwrite that path with a real
// per-spawn MCP config file.
//
// The struct is intentionally empty today: any populated form Wave 3 chooses
// (struct vs interface vs functional option) is a non-breaking change because
// no caller is reading fields off the bundle. Tests in this droplet pass the
// zero value verbatim.
type AuthBundle struct {
	// _ keeps positional struct-literal usage compile-broken so Wave 3 can add
	// fields without silently changing existing call sites.
	_ struct{}
}

// SpawnDescriptor captures the inputs and resolved outputs of one
// BuildSpawnCommand call. The struct is the logging + monitor handoff —
// 4a.21's process monitor consumes it to populate trace fields, and the
// continuous-mode loop in Drop 4b will surface it on dispatcher dashboards.
//
// Every field is set by BuildSpawnCommand on success. Callers MUST NOT mutate
// the returned descriptor; the dispatcher treats the value as immutable for
// the lifetime of the spawn.
type SpawnDescriptor struct {
	// AgentName is the resolved agent variant from
	// templates.KindCatalog.LookupAgentBinding (e.g. "go-builder-agent"). Per
	// the Wave 2.6 division of labor the binding is taken verbatim — this
	// droplet does NOT synthesize {lang}-{role} variants from project.Language.
	AgentName string
	// Model is the LLM model identifier propagated to the agent CLI flags
	// (e.g. "opus", "sonnet", "haiku"). Sourced from binding.Model.
	Model string
	// MaxBudgetUSD is the per-spawn dollar cap propagated as the
	// `--max-budget-usd` flag value.
	MaxBudgetUSD float64
	// MaxTurns is the conversation-turn cap propagated as the `--max-turns`
	// flag value.
	MaxTurns int
	// MCPConfigPath is the absolute filesystem path passed to `--mcp-config`.
	// Wave 2.6 uses a deterministic placeholder under the project worktree's
	// `.tillsyn/` directory; Wave 3 writes the real per-spawn config to this
	// path before the monitor in 4a.21 runs the *exec.Cmd.
	MCPConfigPath string
	// Prompt is the assembled spawn prompt passed to `-p`. The body is opaque
	// to the dispatcher; the Wave 2.6 promptAssembler only guarantees that
	// structural fields (task_id, project_dir, move-state directive) are
	// present so downstream agents can self-locate.
	Prompt string
	// WorkingDir is the project worktree the agent runs in (cmd.Dir).
	WorkingDir string
}

// ErrNoAgentBinding is returned by BuildSpawnCommand when the project's baked
// KindCatalog has no AgentBinding registered for item.Kind. Callers detect
// this via errors.Is and route the action item to a "no agent configured"
// failure rather than treating it as a transient error.
var ErrNoAgentBinding = errors.New("dispatcher: no agent binding for kind")

// ErrInvalidSpawnInput is returned by BuildSpawnCommand when a required input
// field is empty or malformed (empty action-item ID, empty
// project.RepoPrimaryWorktree, etc.). Callers detect this via errors.Is.
var ErrInvalidSpawnInput = errors.New("dispatcher: invalid spawn input")

// BuildSpawnCommand assembles one *exec.Cmd that — when later executed by the
// process monitor in 4a.21 — launches a subagent for the supplied action
// item. This droplet ONLY constructs the Cmd; it does not Start, Run, or Wait.
//
// The argv shape matches REVISION_BRIEF Wave 2 spec:
//
//	claude --agent <agentName> --bare -p "<prompt>" \
//	  --mcp-config <perRunPath> --strict-mcp-config \
//	  --permission-mode acceptEdits \
//	  --max-budget-usd <N> --max-turns <N>
//
// Agent-variant resolution is delegated to templates.KindCatalog.LookupAgentBinding:
// the binding's AgentName is taken verbatim. The Wave 2.6 droplet does NOT
// synthesize {lang}-{role} variants from project.Language — Wave 1 gives the
// planner the project field and the template gives the binding; combining
// them is the planner's responsibility, not the dispatcher's.
//
// Returns ErrNoAgentBinding wrapped with the offending kind when the catalog
// has no entry for item.Kind. Returns ErrInvalidSpawnInput wrapped with a
// reason for empty / malformed inputs. Returns templates.ErrInvalidAgentBinding
// (via Validate) when a corrupted binding survived template-load — the Validate
// re-call inside this function is defensive: AgentBinding.Validate already runs
// at template-load time, so this path is reachable only via direct catalog
// mutation by tests.
func BuildSpawnCommand(
	item domain.ActionItem,
	project domain.Project,
	catalog templates.KindCatalog,
	authBundle AuthBundle,
) (*exec.Cmd, SpawnDescriptor, error) {
	if strings.TrimSpace(item.ID) == "" {
		return nil, SpawnDescriptor{}, fmt.Errorf("%w: action item ID is empty", ErrInvalidSpawnInput)
	}
	if item.Kind == "" {
		return nil, SpawnDescriptor{}, fmt.Errorf("%w: action item kind is empty", ErrInvalidSpawnInput)
	}
	if strings.TrimSpace(project.RepoPrimaryWorktree) == "" {
		return nil, SpawnDescriptor{}, fmt.Errorf(
			"%w: project.RepoPrimaryWorktree is empty (project=%s)",
			ErrInvalidSpawnInput, project.ID,
		)
	}

	binding, ok := catalog.LookupAgentBinding(item.Kind)
	if !ok {
		return nil, SpawnDescriptor{}, fmt.Errorf("%w: kind %q", ErrNoAgentBinding, item.Kind)
	}
	// Defensive re-validate. Validate already runs at template-load (per
	// schema.go:264) so a corrupted binding only reaches here via direct
	// in-memory mutation (tests use this to assert empty-AgentName trips a
	// loud failure rather than silently emitting a `--agent ` flag).
	if err := binding.Validate(); err != nil {
		return nil, SpawnDescriptor{}, fmt.Errorf("dispatcher: invalid agent binding for kind %q: %w", item.Kind, err)
	}

	mcpConfigPath := mcpConfigPlaceholderPath(project.RepoPrimaryWorktree, item.ID)
	prompt := assemblePrompt(item, project, authBundle)

	argv := []string{
		"claude",
		"--agent", binding.AgentName,
		"--bare",
		"-p", prompt,
		"--mcp-config", mcpConfigPath,
		"--strict-mcp-config",
		"--permission-mode", "acceptEdits",
		"--max-budget-usd", formatBudget(binding.MaxBudgetUSD),
		"--max-turns", strconv.Itoa(binding.MaxTurns),
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = project.RepoPrimaryWorktree

	descriptor := SpawnDescriptor{
		AgentName:     binding.AgentName,
		Model:         binding.Model,
		MaxBudgetUSD:  binding.MaxBudgetUSD,
		MaxTurns:      binding.MaxTurns,
		MCPConfigPath: mcpConfigPath,
		Prompt:        prompt,
		WorkingDir:    project.RepoPrimaryWorktree,
	}

	return cmd, descriptor, nil
}

// mcpConfigPlaceholderPath returns the deterministic path the spawner emits
// in the `--mcp-config` slot. Wave 3 will overwrite the file at this path
// with a real per-spawn MCP config; today the path is a placeholder, the file
// is not created, and the agent will fail to load it if the *exec.Cmd is
// actually executed (4a.21 covers execution, Wave 3 covers materialization).
//
// Wave 3: replace this placeholder with a write of the real bundle to the
// returned path before BuildSpawnCommand returns. The path shape (under the
// worktree's `.tillsyn/` dir, action-item-ID-keyed) stays compatible.
func mcpConfigPlaceholderPath(worktree, actionItemID string) string {
	return filepath.Join(worktree, ".tillsyn", "dispatcher-spawn-"+actionItemID+".json")
}

// formatBudget renders a MaxBudgetUSD value for the `--max-budget-usd` CLI
// flag. Whole values render without decimals ("5", not "5.00"); fractional
// values render with the minimum digits required to round-trip via
// strconv.FormatFloat.
func formatBudget(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// assemblePrompt produces the spawn prompt body the agent receives via `-p`.
// The body is opaque to the dispatcher contract — only the structural fields
// (task_id, project_dir, move-state directive) are asserted by tests. Wave 4's
// CLAUDE.md updates document the prompt-body contract for agent authors; this
// function is the producer.
//
// Hylla awareness was deliberately removed in Drop 4c F.7.10: Hylla is a
// dev-local tool, NOT part of Tillsyn's shipped cascade. Adopters who opt
// into Hylla MCP can surface the project's HyllaArtifactRef via their own
// system-prompt template (F.7.2 system_prompt_template_path). The data
// field domain.Project.HyllaArtifactRef and project.metadata.hylla_artifact_ref
// stay because adopter-local templates legitimately consume them.
//
// Wave 3 will fold authBundle session/lease IDs into the prompt under the
// "Auth credentials" line; today the bundle is the empty-struct stub.
func assemblePrompt(item domain.ActionItem, project domain.Project, _ AuthBundle) string {
	var b strings.Builder
	b.WriteString("task_id: ")
	b.WriteString(item.ID)
	b.WriteString("\n")
	b.WriteString("project_id: ")
	b.WriteString(project.ID)
	b.WriteString("\n")
	b.WriteString("project_dir: ")
	b.WriteString(project.RepoPrimaryWorktree)
	b.WriteString("\n")
	b.WriteString("kind: ")
	b.WriteString(string(item.Kind))
	b.WriteString("\n")
	if item.Title != "" {
		b.WriteString("title: ")
		b.WriteString(item.Title)
		b.WriteString("\n")
	}
	// Move-state directive — every spawn prompt instructs the agent to take
	// ownership of its lifecycle transitions.
	b.WriteString("move-state directive: Move the action item to in_progress on start. ")
	b.WriteString("On success set metadata.outcome=\"success\" and move to complete. ")
	b.WriteString("On blocking findings record them in metadata + a closing comment and return.\n")
	return b.String()
}
