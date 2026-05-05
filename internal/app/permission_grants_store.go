package app

import (
	"context"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// PermissionGrantsStore is the storage port the dispatcher uses to read and
// write durable tool-permission grants (Drop 4c F.7.17.7). Each grant
// records that a (project, kind, rule, cli_kind) tuple has been approved by
// some principal, so the dispatcher can pre-grant the rule on subsequent
// spawns rather than re-prompting the dev.
//
// Idempotency: InsertGrant treats a UNIQUE-constraint conflict on
// (project_id, kind, rule, cli_kind) as a noop and returns nil. The
// original row's granted_at + granted_by stay untouched. Callers that need
// to refresh those fields must DeleteGrant + InsertGrant.
//
// Filtering: ListGrantsForKind returns the grants matching one
// (project_id, kind, cli_kind) triple in deterministic granted_at-asc
// order. Cross-project and cross-CLI queries are explicit; the index
// `idx_permission_grants_lookup` covers the lookup.
//
// The interface is declared separately from the core Repository in
// `ports.go` to keep the F.7.17.7 substrate optional — adapters that do
// not need permission grants (e.g. tests using OpenInMemory without the
// dispatcher) can satisfy Repository without implementing this surface.
// The same pattern is used by HandoffRepository.
type PermissionGrantsStore interface {
	// InsertGrant inserts one grant. Returns nil on success and nil when
	// the (project_id, kind, rule, cli_kind) tuple already exists.
	InsertGrant(ctx context.Context, grant domain.PermissionGrant) error

	// ListGrantsForKind returns every grant matching the supplied
	// (projectID, kind, cliKind) triple. The returned slice is non-nil
	// (empty slice when no rows match) and ordered by granted_at ASC then
	// id ASC for determinism. cliKind is matched case-insensitively
	// because the storage adapter persists the lower-cased form.
	ListGrantsForKind(ctx context.Context, projectID string, kind domain.Kind, cliKind string) ([]domain.PermissionGrant, error)

	// DeleteGrant removes one grant by id. Deleting a non-existent id
	// returns ErrNotFound so callers can branch on errors.Is.
	DeleteGrant(ctx context.Context, id string) error
}
