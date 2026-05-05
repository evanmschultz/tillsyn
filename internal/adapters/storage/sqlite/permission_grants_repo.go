package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// InsertGrant persists one PermissionGrant. The UNIQUE composite
// (project_id, kind, rule, cli_kind) makes a duplicate insert a noop:
// `ON CONFLICT(...) DO NOTHING` keeps the original row's granted_at +
// granted_by intact so the audit trail of who first approved a rule
// survives subsequent re-grants. cli_kind is lowered before write so the
// composite remains stable across "Claude" / " claude " inputs (the
// domain constructor already lowercases, but the storage adapter
// re-applies the normalization to defend against direct callers that
// hand-build a PermissionGrant struct).
func (r *Repository) InsertGrant(ctx context.Context, grant domain.PermissionGrant) error {
	id := strings.TrimSpace(grant.ID)
	projectID := strings.TrimSpace(grant.ProjectID)
	kind := strings.TrimSpace(strings.ToLower(string(grant.Kind)))
	rule := strings.TrimSpace(grant.Rule)
	cliKind := strings.TrimSpace(strings.ToLower(grant.CLIKind))
	grantedBy := strings.TrimSpace(grant.GrantedBy)

	if id == "" || projectID == "" || kind == "" || rule == "" || cliKind == "" || grantedBy == "" {
		return domain.ErrInvalidID
	}
	if grant.GrantedAt.IsZero() {
		return domain.ErrInvalidID
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO permission_grants (
			id, project_id, kind, rule, cli_kind, granted_by, granted_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, kind, rule, cli_kind) DO NOTHING
	`,
		id,
		projectID,
		kind,
		rule,
		cliKind,
		grantedBy,
		ts(grant.GrantedAt),
	)
	return err
}

// ListGrantsForKind returns every grant matching one
// (projectID, kind, cliKind) triple in deterministic granted_at-asc,
// id-asc order. The query is index-covered by
// idx_permission_grants_lookup (project_id, kind, cli_kind).
func (r *Repository) ListGrantsForKind(ctx context.Context, projectID string, kind domain.Kind, cliKind string) ([]domain.PermissionGrant, error) {
	projectIDNorm := strings.TrimSpace(projectID)
	kindNorm := strings.TrimSpace(strings.ToLower(string(kind)))
	cliKindNorm := strings.TrimSpace(strings.ToLower(cliKind))

	if projectIDNorm == "" || kindNorm == "" || cliKindNorm == "" {
		return nil, domain.ErrInvalidID
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, kind, rule, cli_kind, granted_by, granted_at
		FROM permission_grants
		WHERE project_id = ? AND kind = ? AND cli_kind = ?
		ORDER BY granted_at ASC, id ASC
	`, projectIDNorm, kindNorm, cliKindNorm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.PermissionGrant, 0)
	for rows.Next() {
		grant, scanErr := scanPermissionGrant(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, grant)
	}
	return out, rows.Err()
}

// DeleteGrant removes one grant by id. Deleting a non-existent id returns
// app.ErrNotFound so callers can branch on errors.Is.
func (r *Repository) DeleteGrant(ctx context.Context, id string) error {
	idNorm := strings.TrimSpace(id)
	if idNorm == "" {
		return domain.ErrInvalidID
	}
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM permission_grants
		WHERE id = ?
	`, idNorm)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// scanPermissionGrant decodes one permission_grants row into a
// domain.PermissionGrant. cli_kind is intentionally not re-normalized on
// read because the write path lowered it; trusting the column lets the
// row round-trip without surprises.
func scanPermissionGrant(s scanner) (domain.PermissionGrant, error) {
	var (
		grant      domain.PermissionGrant
		kindRaw    string
		grantedRaw string
	)
	if err := s.Scan(
		&grant.ID,
		&grant.ProjectID,
		&kindRaw,
		&grant.Rule,
		&grant.CLIKind,
		&grant.GrantedBy,
		&grantedRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PermissionGrant{}, app.ErrNotFound
		}
		return domain.PermissionGrant{}, err
	}
	grant.Kind = domain.Kind(kindRaw)
	grant.GrantedAt = parseTS(grantedRaw)
	return grant, nil
}
