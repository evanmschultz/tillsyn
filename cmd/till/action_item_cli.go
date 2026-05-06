package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/google/uuid"
)

// runActionItemGet resolves one action-item id (UUID or dotted address) to a
// canonical UUID via the resolver and writes the matched action item as JSON.
// Project context is supplied either by the --project flag or by the
// slug-prefix shorthand <slug>:<dotted>; UUID input bypasses both. Bare dotted
// addresses without project context error with a clear hint.
func runActionItemGet(ctx context.Context, svc *app.Service, opts actionItemCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	idOrDotted := strings.TrimSpace(opts.actionItemID)
	if idOrDotted == "" {
		return errors.New("action_item get: action_item_id is required")
	}

	// UUID-shaped input bypasses the resolver entirely.
	if _, err := uuid.Parse(idOrDotted); err == nil {
		actionItem, err := svc.GetActionItem(ctx, idOrDotted)
		if err != nil {
			return fmt.Errorf("action_item get %q: %w", idOrDotted, err)
		}
		return writeActionItemJSON(stdout, actionItem)
	}

	if !app.IsLikelyDottedAddress(idOrDotted) {
		return fmt.Errorf("action_item get: action_item_id %q is neither a UUID nor a dotted address (expected `1.5.2`, `<slug>:1.5.2`, or a UUID)", idOrDotted)
	}

	projectID, err := resolveActionItemProjectContext(ctx, svc, idOrDotted, opts.projectSlug)
	if err != nil {
		return err
	}

	resolvedID, err := svc.ResolveActionItemID(ctx, projectID, idOrDotted)
	if err != nil {
		return fmt.Errorf("action_item get %q: resolve dotted address: %w", idOrDotted, err)
	}
	actionItem, err := svc.GetActionItem(ctx, resolvedID)
	if err != nil {
		return fmt.Errorf("action_item get %q (resolved %q): %w", idOrDotted, resolvedID, err)
	}
	return writeActionItemJSON(stdout, actionItem)
}

// runActionItemSupersede is the CLI flow for the Drop 4c.5 droplet B.1
// supersede escape hatch. The flow is the dev's "I am clearing THIS failed
// item so its parent can move forward" affordance — it transitions one
// `failed` action item to `complete` with `metadata.outcome = "superseded"`
// and the supplied reason persisted on `metadata.transition_notes`.
//
// Pre-service-call validation order (each failure surfaces a distinct
// error class):
//
//  1. Empty / whitespace-only `--reason` rejects with a clear "reason
//     required" error before any service call. The supersede CLI's whole
//     point is recording dev intent; an empty reason defeats it.
//  2. Empty / dotted-form `action_item_id` rejects via
//     `app.ValidateActionItemIDForMutation` (mutations require UUID — same
//     gate as `update`/`move`/`delete`/`restore`/`reparent`).
//  3. UUID-shaped input passes the gate and reaches
//     `Service.SupersedeActionItem`, which enforces the failed-only
//     transition + writes the audit-trail metadata + flips the column.
//
// On success, the post-supersede action item is rendered as JSON on stdout
// (matching the `runActionItemGet` rendering convention) so the dev can
// confirm the new column placement + outcome stamp.
func runActionItemSupersede(ctx context.Context, svc *app.Service, opts actionItemCommandOptions, stdout io.Writer) error {
	// Validate the input shape BEFORE the service-availability check so the
	// CLI's user-facing error messages reflect what's wrong with the
	// invocation rather than the runtime wiring. Validation order:
	//
	//  1. Empty / whitespace-only --reason (required content gate).
	//  2. UUID-shape gate (mutations-require-UUID across CLI mutation paths).
	//  3. App-service availability (runtime wiring sanity check).
	reason := strings.TrimSpace(opts.reason)
	if reason == "" {
		return fmt.Errorf("action_item supersede: --reason is required (whitespace-only rejected)")
	}
	if err := app.ValidateActionItemIDForMutation(opts.actionItemID); err != nil {
		return fmt.Errorf("action_item supersede: %w", err)
	}
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	actionItem, err := svc.SupersedeActionItem(ctx, strings.TrimSpace(opts.actionItemID), reason)
	if err != nil {
		return fmt.Errorf("action_item supersede %q: %w", opts.actionItemID, err)
	}
	return writeActionItemJSON(stdout, actionItem)
}

// runActionItemMutationGate enforces the mutations-require-UUID rule for CLI
// mutation subcommands. The CLI does NOT yet implement the mutation operations
// themselves (Drop 2 wires only the validator + reject path so MCP and CLI
// share the same boundary contract); a UUID-shaped action_item_id therefore
// returns a not-implemented hint pointing the operator at the MCP surface.
// Dotted addresses are rejected with the canonical mutations-require-UUID
// error class — same wording the MCP boundary returns.
func runActionItemMutationGate(operation string, opts actionItemCommandOptions) error {
	if err := app.ValidateActionItemIDForMutation(opts.actionItemID); err != nil {
		return fmt.Errorf("action_item %s: %w", operation, err)
	}
	return fmt.Errorf("action_item %s: CLI mutation flow is not yet implemented; use the MCP surface (till.action_item operation=%s) for the matching tooling. The CLI gate accepted the UUID input — operator-facing CLI mutations land in a future drop", operation, operation)
}

// resolveActionItemProjectContext returns the projectID for a dotted address
// using either the slug-prefix shorthand (<slug>:<body>) or the explicit
// --project flag. Slug-prefix takes precedence when present, with the --project
// flag (if also supplied) treated as an extra check that the slug matches.
// Returns an error when neither source yields a project.
func resolveActionItemProjectContext(ctx context.Context, svc *app.Service, dotted, projectSlug string) (string, error) {
	dotted = strings.TrimSpace(dotted)
	projectSlug = strings.TrimSpace(projectSlug)
	prefixSlug := app.SplitDottedSlugPrefix(dotted)
	switch {
	case prefixSlug != "":
		project, err := svc.GetProjectBySlug(ctx, prefixSlug)
		if err != nil {
			return "", fmt.Errorf("action_item get: look up project for slug %q: %w", prefixSlug, err)
		}
		if projectSlug != "" && projectSlug != prefixSlug {
			return "", fmt.Errorf("action_item get: --project %q does not match dotted slug-prefix %q; pick one source of project context", projectSlug, prefixSlug)
		}
		return project.ID, nil
	case projectSlug != "":
		project, err := svc.GetProjectBySlug(ctx, projectSlug)
		if err != nil {
			return "", fmt.Errorf("action_item get: look up project for slug %q: %w", projectSlug, err)
		}
		return project.ID, nil
	default:
		return "", fmt.Errorf("action_item get: dotted address %q requires --project <slug> or the slug-prefix shorthand <slug>:<dotted>", dotted)
	}
}

// writeActionItemJSON writes one action item as indented JSON. The CLI keeps
// action-item details JSON-formatted for now to avoid adding another renderer
// surface for what is currently a single read command; future drops can
// upgrade this to a human-friendly detail view.
func writeActionItemJSON(stdout io.Writer, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode action item: %w", err)
	}
	if _, err := stdout.Write(encoded); err != nil {
		return fmt.Errorf("write action item: %w", err)
	}
	if _, err := stdout.Write([]byte{'\n'}); err != nil {
		return fmt.Errorf("write action item terminator: %w", err)
	}
	return nil
}
