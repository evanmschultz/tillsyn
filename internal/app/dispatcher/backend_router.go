// Package dispatcher — backend_router.go routes agent personas to CLI kinds.
//
// BackendRouter selects the CLI kind (client identifier) for a given agent
// persona, given the resolved template + agents.toml registry. At runtime,
// exactly one of (template.Client, preset.Client) is non-empty per the
// boot-time ValidateAgentsTemplateClientConflict validator — D5's job is
// simply to pick the single non-empty value.
package dispatcher

import (
	"errors"
	"fmt"

	"github.com/evanmschultz/tillsyn/internal/config"
)

// ErrUnroutablePersona is returned when both the template and preset have
// empty Client fields — no CLI kind is configured for this persona, making
// it impossible to route to a backend.
var ErrUnroutablePersona = errors.New("dispatcher: unroutable persona — no cli_kind configured")

// ResolvedTemplate carries the resolved template bindings for a given kind.
// Client holds the CLI kind (e.g. "claude", "codex") from the template layer.
type ResolvedTemplate struct {
	Client string
}

// BackendRouter selects the CLI kind (backend) for an agent persona given
// the resolved template + agents.toml preset registry. The router is keyed
// by persona name (e.g. "ta-go-builder").
//
// Boot-time validator ValidateAgentsTemplateClientConflict guarantees at
// most one non-empty Client between template and preset, so ResolveBackend
// performs a straightforward selection: pick the single non-empty value or
// error on dual-empty. No "preset wins" precedence — that would contradict
// the validator's PEER semantics.
type BackendRouter struct {
	registry *config.AgentsRegistry
	template ResolvedTemplate
}

// NewBackendRouter constructs a router for the given template and registry.
func NewBackendRouter(registry *config.AgentsRegistry, template ResolvedTemplate) *BackendRouter {
	return &BackendRouter{
		registry: registry,
		template: template,
	}
}

// ResolveBackend returns the CLI kind to use for the given persona name.
// Exactly one of (template.Client, preset.Client) is non-empty at runtime
// thanks to the boot-time validator. ResolveBackend picks the single non-empty
// value or returns ErrUnroutablePersona if both are empty.
//
// Arguments:
//   - personaName: the agent persona name (e.g. "ta-go-builder")
//   - group: the agents.toml group name (e.g. "go")
//   - kind: the kind identifier (e.g. "build")
//
// Returns the CLI kind string or ErrUnroutablePersona if no client is configured.
func (r *BackendRouter) ResolveBackend(personaName, group, kind string) (string, error) {
	// Resolve the preset from the registry.
	preset, err := config.Resolve(*r.registry, group, kind)
	if err != nil {
		// Resolve currently never returns non-nil error, but forward-compat
		// the check per the function's reserved error signature.
		return "", fmt.Errorf("%w: resolve preset: %w", ErrUnroutablePersona, err)
	}

	templateClient := r.template.Client
	presetClient := preset.Client

	// Count non-empty clients.
	templateEmpty := templateClient == ""
	presetEmpty := presetClient == ""

	switch {
	case templateEmpty && presetEmpty:
		// Both empty — unroutable.
		return "", fmt.Errorf("%w: persona %q (group %q, kind %q)", ErrUnroutablePersona, personaName, group, kind)
	case !templateEmpty && presetEmpty:
		// Template only.
		return templateClient, nil
	case templateEmpty && !presetEmpty:
		// Preset only.
		return presetClient, nil
	default:
		// Both non-empty. The boot-time validator guarantees they are equal.
		// Defense-in-depth: reject if they differ (validator should have caught this).
		if templateClient != presetClient {
			return "", fmt.Errorf("%w: persona %q has conflicting cli_kinds: template=%q preset=%q", ErrUnroutablePersona, personaName, templateClient, presetClient)
		}
		return templateClient, nil
	}
}
