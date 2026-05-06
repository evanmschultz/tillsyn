package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/google/uuid"
)

// validActionItemListStates is the closed set of lifecycle states accepted by
// the `till action_item list --state <value>` flag. The slice is the source
// of truth for both the flag-validation error message and the cobra `Long:`
// help text — keep them in sync if the lifecycle enum ever grows.
var validActionItemListStates = []domain.LifecycleState{
	domain.StateTodo,
	domain.StateInProgress,
	domain.StateComplete,
	domain.StateFailed,
	domain.StateArchived,
}

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

// runActionItemList is the CLI flow for the Drop 4c.5 droplet B.2
// failure-listing command. The dev runs `till action_item list --state failed
// --project tillsyn` to inventory stuck items so they can clear each via the
// Drop 4c.5 droplet B.1 supersede CLI. Default --state is "failed" — the
// canonical pre-TUI use case; other lifecycle states are accepted for
// general-purpose listing.
//
// Pre-service-call validation order:
//
//  1. The state flag is normalized (trim + lower) and validated against the
//     closed lifecycle set; an unknown state rejects naming the valid set.
//     Empty state defaults to "failed" as a defensive fallback for direct
//     callers (the CLI's flag default already supplies "failed").
//  2. Project resolution requires either --project <slug> or, when the system
//     has exactly one project, the auto-resolution fallback. Two-or-more
//     projects without --project rejects with a clear hint pointing at
//     --project.
//  3. The slug-prefix shorthand (`tillsyn:1.5.2`) is intentionally NOT
//     accepted on this command — list is project-scoped, not item-scoped,
//     and accepting `tillsyn:failed` would conflate "list filter" with
//     "dotted address" in confusing ways. Callers with a slug pass it via
//     --project.
//
// On success the result table is rendered via `writeCLITable` (laslig-styled
// in human terminals, machine-parseable when piped). The empty-state message
// names both the requested state and the project slug so the dev sees what
// was actually queried.
func runActionItemList(ctx context.Context, svc *app.Service, opts actionItemCommandOptions, stdout io.Writer) error {
	if svc == nil {
		return fmt.Errorf("app service is not configured")
	}
	rawState := strings.TrimSpace(opts.state)
	if rawState == "" {
		rawState = string(domain.StateFailed)
	}
	state := domain.LifecycleState(strings.ToLower(rawState))
	if !slices.Contains(validActionItemListStates, state) {
		return fmt.Errorf("action_item list: unknown --state %q (valid: %s)", opts.state, joinLifecycleStates(validActionItemListStates))
	}
	project, err := resolveActionItemListProject(ctx, svc, opts.projectSlug)
	if err != nil {
		return err
	}
	includeArchived := opts.includeArchived
	if state == domain.StateArchived {
		// Asking for archived items implies including them. The service-side
		// helper does the same forcing; mirroring it here keeps the CLI's
		// project-listing path coherent with the filter the user sees.
		includeArchived = true
	}
	items, err := svc.ListActionItemsByState(ctx, project.ID, state, includeArchived)
	if err != nil {
		return fmt.Errorf("action_item list: %w", err)
	}
	// Compute dotted-address for each item via a project-wide tree walk.
	// Pre-MVP scale (<1k items) makes this acceptable without an indexed
	// query; the address column is otherwise the only piece of context the
	// dev needs to navigate from the table to a follow-up `till action_item
	// supersede` invocation.
	addresses, err := computeDottedAddressesForItems(ctx, svc, project.ID, items)
	if err != nil {
		return fmt.Errorf("action_item list: compute dotted addresses: %w", err)
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			firstNonEmptyTrimmed(addresses[item.ID], "-"),
			firstNonEmptyTrimmed(item.ID, "-"),
			firstNonEmptyTrimmed(item.Title, "-"),
			firstNonEmptyTrimmed(string(item.Kind), "-"),
			firstNonEmptyTrimmed(string(item.Role), "-"),
			formatActionItemListUpdatedAt(item.UpdatedAt),
		})
	}
	emptyMsg := fmt.Sprintf("No %s action items in project %s.", state, project.Slug)
	return writeCLITable(
		stdout,
		"Action Items",
		[]string{"DOTTED", "UUID", "TITLE", "KIND", "ROLE", "UPDATED"},
		rows,
		emptyMsg,
	)
}

// resolveActionItemListProject resolves the project context for the
// `till action_item list` command. When the operator supplied --project the
// slug is looked up directly; otherwise the command falls back to the
// single-project-on-system shortcut and surfaces a clear hint when more than
// one project exists without explicit selection.
func resolveActionItemListProject(ctx context.Context, svc *app.Service, projectSlug string) (domain.Project, error) {
	projectSlug = strings.TrimSpace(projectSlug)
	if projectSlug != "" {
		project, err := svc.GetProjectBySlug(ctx, projectSlug)
		if err != nil {
			return domain.Project{}, fmt.Errorf("action_item list: look up project for slug %q: %w", projectSlug, err)
		}
		return project, nil
	}
	projects, err := svc.ListProjects(ctx, false)
	if err != nil {
		return domain.Project{}, fmt.Errorf("action_item list: list projects: %w", err)
	}
	switch len(projects) {
	case 0:
		return domain.Project{}, fmt.Errorf("action_item list: no projects on the system; create one before listing action items")
	case 1:
		return projects[0], nil
	default:
		slugs := make([]string, 0, len(projects))
		for _, p := range projects {
			slugs = append(slugs, p.Slug)
		}
		sort.Strings(slugs)
		return domain.Project{}, fmt.Errorf("action_item list: %d projects on the system; pass --project <slug> (available: %s)", len(projects), strings.Join(slugs, ", "))
	}
}

// computeDottedAddressesForItems returns a map of action_item_id → dotted
// address (e.g. "1.5.2") for every item in `items`. Pre-MVP scale means
// listing the entire project's action-item set once and walking parent
// chains in memory is cheaper than per-item repository round-trips.
//
// The dotted-address contract mirrors `app.ResolveDottedAddress`: ordering at
// every level is `(CreatedAt, ID)` ASC, segments are 0-indexed. Items whose
// ParentID is missing from the project listing (e.g. archived parent excluded
// by the caller's includeArchived=false) get an empty string in the result
// map; the caller renders empty as "-".
func computeDottedAddressesForItems(ctx context.Context, svc *app.Service, projectID string, items []domain.ActionItem) (map[string]string, error) {
	if len(items) == 0 {
		return map[string]string{}, nil
	}
	// Re-fetch the full project tree (includeArchived=true) so we can
	// compute addresses even when the listed items reference archived
	// ancestors — a `failed` item under an `archived` parent still has a
	// well-defined dotted address. This is the only repo round-trip beyond
	// the initial filtered list call.
	all, err := svc.ListActionItems(ctx, projectID, true)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.ActionItem, len(all))
	childrenByParent := make(map[string][]domain.ActionItem, len(all))
	for _, item := range all {
		byID[item.ID] = item
		childrenByParent[item.ParentID] = append(childrenByParent[item.ParentID], item)
	}
	for parent, children := range childrenByParent {
		// Sort each parent's children by (CreatedAt, ID) ASC — the same
		// total ordering ResolveDottedAddress walks.
		slices.SortFunc(children, func(a, b domain.ActionItem) int {
			if a.CreatedAt.Equal(b.CreatedAt) {
				return strings.Compare(a.ID, b.ID)
			}
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			}
			return 1
		})
		childrenByParent[parent] = children
	}
	addresses := make(map[string]string, len(items))
	for _, item := range items {
		addresses[item.ID] = computeDottedAddressFor(item, byID, childrenByParent)
	}
	_ = ctx
	return addresses, nil
}

// computeDottedAddressFor walks from item up to the root, recording each
// level's index in the parent's sorted children slice. Returns an empty
// string when an ancestor cannot be resolved (e.g. the parent set was not
// included in the lookup map).
func computeDottedAddressFor(item domain.ActionItem, byID map[string]domain.ActionItem, childrenByParent map[string][]domain.ActionItem) string {
	segments := []string{}
	current := item
	// Bound the walk to the size of the project's action-item set. A
	// well-formed tree has depth strictly less than `len(byID)`; the bound
	// guards against accidental cycles in the persisted graph.
	for i := 0; i <= len(byID); i++ {
		siblings, ok := childrenByParent[current.ParentID]
		if !ok {
			return ""
		}
		idx := -1
		for j, sibling := range siblings {
			if sibling.ID == current.ID {
				idx = j
				break
			}
		}
		if idx < 0 {
			return ""
		}
		segments = append([]string{fmt.Sprintf("%d", idx)}, segments...)
		if current.ParentID == "" {
			return strings.Join(segments, ".")
		}
		parent, ok := byID[current.ParentID]
		if !ok {
			return ""
		}
		current = parent
	}
	return ""
}

// formatActionItemListUpdatedAt renders one UpdatedAt timestamp for the list
// command. Stable RFC3339 format keeps the column width predictable in
// human terminals and trivially parseable when the output is piped. Zero
// times render as "-" for visual consistency with other empty cells.
func formatActionItemListUpdatedAt(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

// joinLifecycleStates formats a closed lifecycle-state set as a
// comma-separated list. Used in error messages so the operator sees the
// exact valid set when an unknown --state is passed.
func joinLifecycleStates(states []domain.LifecycleState) string {
	parts := make([]string, 0, len(states))
	for _, s := range states {
		parts = append(parts, string(s))
	}
	return strings.Join(parts, ", ")
}
