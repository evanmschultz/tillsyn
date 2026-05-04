package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
)

// dispatcherRunCommandOptions carries the flags consumed by `till dispatcher
// run`. The CLI exposes the cascade dispatcher's manual-trigger entry point:
// one action-item ID becomes one dispatcher.RunOnce evaluation, optionally
// previewed via --dry-run.
//
// Wave 2.10 milestone caveat: each `till dispatcher run` invocation
// bootstraps its OWN dispatcher (broker, service, lock managers, monitor).
// Two parallel CLI invocations DO race on the in-process lock managers
// because each process has its own copy. The locks DO observe the same
// SQLite-backed action-item state, so the walker's eligibility predicate
// holds across processes — but a sibling-overlap attack only triggers the
// runtime blocker when both candidates land in the same process. The dev
// runs invocations serially during the manual-trigger milestone; Drop 4b's
// daemon variant lands the long-running shared-broker dispatcher.
type dispatcherRunCommandOptions struct {
	// actionItemID is the action item to evaluate. Required.
	actionItemID string
	// projectID is the authoritative project_id override (4a.23
	// QA-Falsification §2.2 fix). When non-empty, the dispatcher MUST
	// resolve this project_id; mismatch with the action item's own
	// ProjectID returns ErrProjectMismatch and exits non-zero. When
	// empty, the dispatcher resolves the project from the action item
	// directly (the historical behaviour).
	projectID string
	// dryRun, when true, walks the eligibility check and constructs the
	// SpawnDescriptor but does not execute the spawn. Stdout receives the
	// descriptor as pretty-printed JSON. The CLI exits 0 regardless of
	// whether the spawn would have been ResultSpawned, ResultBlocked, or
	// ResultSkipped — the caller's decision branch is the JSON shape, not
	// the exit code, on dry-run.
	dryRun bool
}

// runDispatcherRun is the dispatcher CLI's RunE body, factored out so the
// test suite can drive it directly with a controlled *app.Service and broker
// (mirroring the action-item CLI test pattern).
//
// The function is responsible for:
//   - Validating opts.actionItemID is non-empty.
//   - On --dry-run: invoking dispatcher.PreviewSpawn and printing the
//     resulting SpawnDescriptor as pretty-printed JSON. Exits 0 on success.
//   - Otherwise: constructing a Dispatcher via NewDispatcher, calling
//     RunOnce, and printing a one-line human-readable summary derived
//     from the DispatchOutcome fields.
//   - Returning a non-nil error when the outcome is ResultFailed or when
//     RunOnce surfaces an infrastructure error. The cobra runtime maps
//     non-nil errors to a non-zero exit code.
func runDispatcherRun(ctx context.Context, svc *app.Service, broker app.LiveWaitBroker, opts dispatcherRunCommandOptions, stdout, stderr io.Writer) error {
	if svc == nil {
		return fmt.Errorf("dispatcher run: app service is not configured")
	}
	if broker == nil {
		return fmt.Errorf("dispatcher run: live wait broker is not configured")
	}
	actionItemID := strings.TrimSpace(opts.actionItemID)
	if actionItemID == "" {
		return fmt.Errorf("dispatcher run: --action-item is required")
	}

	disp, err := dispatcher.NewDispatcher(svc, broker, dispatcher.Options{})
	if err != nil {
		return fmt.Errorf("dispatcher run: construct dispatcher: %w", err)
	}

	projectOverride := strings.TrimSpace(opts.projectID)

	if opts.dryRun {
		preview, _, _, err := disp.PreviewSpawn(ctx, actionItemID, projectOverride)
		if err != nil {
			// Treat ErrNoAgentBinding as a soft condition the dev wants
			// to see in JSON shape rather than as a hard non-zero exit
			// — but the descriptor is empty in that case. Surface the
			// reason on stderr and emit an empty JSON object on stdout
			// so machine-driven callers do not crash on a non-JSON
			// stream when --dry-run + missing-binding combine.
			if errors.Is(err, dispatcher.ErrNoAgentBinding) {
				if stderr != nil {
					_, _ = fmt.Fprintf(stderr, "dry-run: no agent binding available: %v\n", err)
				}
				return writeDispatcherPreviewJSON(stdout, dispatcher.SpawnPreview{})
			}
			// ErrProjectMismatch is a hard non-zero exit per the
			// authoritative-override contract (4a.23 §2.2 fix). The
			// dev sees the typed error and the offending project_id
			// pair without falling through to a stale descriptor.
			return fmt.Errorf("dispatcher run --dry-run: %w", err)
		}
		return writeDispatcherPreviewJSON(stdout, preview)
	}

	outcome, err := disp.RunOnce(ctx, actionItemID, projectOverride)
	if err != nil {
		return fmt.Errorf("dispatcher run: RunOnce: %w", err)
	}

	switch outcome.Result {
	case dispatcher.ResultSpawned:
		_, _ = fmt.Fprintf(stdout, "spawned %s for %s\n", outcome.AgentName, outcome.ActionItemID)
	case dispatcher.ResultSkipped:
		reason := outcome.Reason
		if reason == "" {
			reason = "no eligible work"
		}
		_, _ = fmt.Fprintf(stdout, "skipped: %s\n", reason)
	case dispatcher.ResultBlocked:
		reason := outcome.Reason
		if reason == "" {
			reason = "runtime conflict inserted blocker"
		}
		_, _ = fmt.Fprintf(stdout, "blocked: %s\n", reason)
	case dispatcher.ResultFailed:
		reason := outcome.Reason
		if reason == "" {
			reason = "dispatcher reported failed without a reason"
		}
		if stderr != nil {
			_, _ = fmt.Fprintf(stderr, "failed: %s\n", reason)
		}
		return fmt.Errorf("dispatcher run: outcome failed: %s", reason)
	default:
		// New Result enum value not yet handled by the CLI — surface as
		// a hard error so adding a value forces a deliberate CLI edit.
		return fmt.Errorf("dispatcher run: unknown outcome result %q", outcome.Result)
	}
	return nil
}

// writeDispatcherPreviewJSON renders one SpawnPreview as indented JSON on
// stdout. The shape contains the SpawnDescriptor (snake_case wire form) plus
// the eligibility / overlap / lock-conflict fields PreviewSpawn evaluated
// (4a.23 §2.3 fix — pre-fix shape stopped at the descriptor). Empty stdout
// writers (io.Discard) are safe; the trailing newline keeps shell pipelines
// tidy.
func writeDispatcherPreviewJSON(stdout io.Writer, preview dispatcher.SpawnPreview) error {
	if stdout == nil {
		stdout = io.Discard
	}
	encoded, err := json.MarshalIndent(spawnPreviewJSON(preview), "", "  ")
	if err != nil {
		return fmt.Errorf("encode spawn preview: %w", err)
	}
	if _, err := stdout.Write(encoded); err != nil {
		return fmt.Errorf("write spawn preview: %w", err)
	}
	if _, err := stdout.Write([]byte{'\n'}); err != nil {
		return fmt.Errorf("write spawn preview terminator: %w", err)
	}
	return nil
}

// spawnDescriptorJSON renames SpawnDescriptor's fields to snake_case for the
// CLI's JSON output. The dispatcher package's struct uses Go-idiomatic Pascal
// case (AgentName, MaxBudgetUSD); the CLI surface uses snake_case so dev
// scripts (jq, python, etc.) consume the JSON without per-language casing
// hacks. The shape pins the field set for --dry-run consumers.
//
// Type-alias guarantee (4a.23 QA-Falsification §3.3 NIT addressed):
// `type spawnDescriptorJSON dispatcher.SpawnDescriptor` is structural —
// adding a field to dispatcher.SpawnDescriptor compiles into this alias
// automatically. The MarshalJSON wire struct below STILL has to enumerate
// the field set, so a new SpawnDescriptor field would silently fall off
// the JSON until a dev edits MarshalJSON. The structural-alignment test in
// dispatcher_cli_test.go locks the field count via reflect so silent drift
// fails loudly.
type spawnDescriptorJSON dispatcher.SpawnDescriptor

// MarshalJSON renders the SpawnDescriptor with snake_case keys. Implementing
// MarshalJSON on a named-type alias avoids leaking JSON tags into the
// dispatcher package's domain-shape struct (the dispatcher package treats
// SpawnDescriptor as an in-process value; the CLI is the only consumer that
// JSON-encodes it).
func (d spawnDescriptorJSON) MarshalJSON() ([]byte, error) {
	type wire struct {
		AgentName     string  `json:"agent_name"`
		Model         string  `json:"model"`
		MaxBudgetUSD  float64 `json:"max_budget_usd"`
		MaxTurns      int     `json:"max_turns"`
		MCPConfigPath string  `json:"mcp_config_path"`
		Prompt        string  `json:"prompt"`
		WorkingDir    string  `json:"working_dir"`
	}
	return json.Marshal(wire{
		AgentName:     d.AgentName,
		Model:         d.Model,
		MaxBudgetUSD:  d.MaxBudgetUSD,
		MaxTurns:      d.MaxTurns,
		MCPConfigPath: d.MCPConfigPath,
		Prompt:        d.Prompt,
		WorkingDir:    d.WorkingDir,
	})
}

// spawnPreviewJSON wraps dispatcher.SpawnPreview so MarshalJSON renders the
// CLI's snake_case shape: the descriptor's fields are emitted alongside
// eligible / reason / overlaps / file_lock_conflicts /
// package_lock_conflicts. The shape is the dev-script contract for
// --dry-run.
type spawnPreviewJSON dispatcher.SpawnPreview

// MarshalJSON renders the SpawnPreview with snake_case keys. The descriptor
// is inlined (no nested "descriptor" key) so dev scripts that already keyed
// off agent_name / model / working_dir keep working unchanged. The new
// fields are additive: scripts that ignore them are unaffected.
func (p spawnPreviewJSON) MarshalJSON() ([]byte, error) {
	type overlapWire struct {
		SiblingID            string `json:"sibling_id"`
		OverlapKind          string `json:"overlap_kind"`
		OverlapValue         string `json:"overlap_value"`
		HasExplicitBlockedBy bool   `json:"has_explicit_blocked_by"`
	}
	type wire struct {
		AgentName            string            `json:"agent_name"`
		Model                string            `json:"model"`
		MaxBudgetUSD         float64           `json:"max_budget_usd"`
		MaxTurns             int               `json:"max_turns"`
		MCPConfigPath        string            `json:"mcp_config_path"`
		Prompt               string            `json:"prompt"`
		WorkingDir           string            `json:"working_dir"`
		Eligible             bool              `json:"eligible"`
		Reason               string            `json:"reason,omitempty"`
		Overlaps             []overlapWire     `json:"overlaps,omitempty"`
		FileLockConflicts    map[string]string `json:"file_lock_conflicts,omitempty"`
		PackageLockConflicts map[string]string `json:"package_lock_conflicts,omitempty"`
	}
	overlaps := make([]overlapWire, 0, len(p.Overlaps))
	for _, ov := range p.Overlaps {
		overlaps = append(overlaps, overlapWire{
			SiblingID:            ov.SiblingID,
			OverlapKind:          string(ov.OverlapKind),
			OverlapValue:         ov.OverlapValue,
			HasExplicitBlockedBy: ov.HasExplicitBlockedBy,
		})
	}
	out := wire{
		AgentName:            p.Descriptor.AgentName,
		Model:                p.Descriptor.Model,
		MaxBudgetUSD:         p.Descriptor.MaxBudgetUSD,
		MaxTurns:             p.Descriptor.MaxTurns,
		MCPConfigPath:        p.Descriptor.MCPConfigPath,
		Prompt:               p.Descriptor.Prompt,
		WorkingDir:           p.Descriptor.WorkingDir,
		Eligible:             p.Eligible,
		Reason:               p.Reason,
		Overlaps:             overlaps,
		FileLockConflicts:    p.FileLockConflicts,
		PackageLockConflicts: p.PackageLockConflicts,
	}
	if len(out.Overlaps) == 0 {
		out.Overlaps = nil
	}
	if len(out.FileLockConflicts) == 0 {
		out.FileLockConflicts = nil
	}
	if len(out.PackageLockConflicts) == 0 {
		out.PackageLockConflicts = nil
	}
	return json.Marshal(out)
}
