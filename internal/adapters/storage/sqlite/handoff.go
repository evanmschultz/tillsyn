package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hylla/tillsyn/internal/app"
	"github.com/hylla/tillsyn/internal/domain"
)

// CreateHandoff creates one handoff row.
func (r *Repository) CreateHandoff(ctx context.Context, handoff domain.Handoff) error {
	handoff, err := normalizeHandoffForWrite(handoff)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO handoffs(
			id, project_id, branch_id, scope_type, scope_id, target_branch_id, target_scope_type, target_scope_id,
			source_role, target_role, status, summary, next_action, missing_evidence_json, related_refs_json,
			created_by_actor, created_by_type, created_at, updated_by_actor, updated_by_type, updated_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		handoff.ID,
		handoff.ProjectID,
		handoff.BranchID,
		string(handoff.ScopeType),
		handoff.ScopeID,
		handoff.TargetBranchID,
		string(handoff.TargetScopeType),
		handoff.TargetScopeID,
		handoff.SourceRole,
		handoff.TargetRole,
		string(handoff.Status),
		handoff.Summary,
		handoff.NextAction,
		string(mustJSON(handoff.MissingEvidence)),
		string(mustJSON(handoff.RelatedRefs)),
		handoff.CreatedByActor,
		string(handoff.CreatedByType),
		ts(handoff.CreatedAt),
		handoff.UpdatedByActor,
		string(handoff.UpdatedByType),
		ts(handoff.UpdatedAt),
		handoff.ResolvedByActor,
		normalizeOptionalActorType(handoff.ResolvedByType),
		nullableTS(handoff.ResolvedAt),
		handoff.ResolutionNote,
	)
	return err
}

// UpdateHandoff updates one handoff row.
func (r *Repository) UpdateHandoff(ctx context.Context, handoff domain.Handoff) error {
	handoff, err := normalizeHandoffForWrite(handoff)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE handoffs
		SET project_id = ?, branch_id = ?, scope_type = ?, scope_id = ?, target_branch_id = ?, target_scope_type = ?, target_scope_id = ?,
			source_role = ?, target_role = ?, status = ?, summary = ?, next_action = ?, missing_evidence_json = ?, related_refs_json = ?,
			created_by_actor = ?, created_by_type = ?, updated_by_actor = ?, updated_by_type = ?, updated_at = ?,
			resolved_by_actor = ?, resolved_by_type = ?, resolved_at = ?, resolution_note = ?
		WHERE id = ?
	`,
		handoff.ProjectID,
		handoff.BranchID,
		string(handoff.ScopeType),
		handoff.ScopeID,
		handoff.TargetBranchID,
		string(handoff.TargetScopeType),
		handoff.TargetScopeID,
		handoff.SourceRole,
		handoff.TargetRole,
		string(handoff.Status),
		handoff.Summary,
		handoff.NextAction,
		string(mustJSON(handoff.MissingEvidence)),
		string(mustJSON(handoff.RelatedRefs)),
		handoff.CreatedByActor,
		string(handoff.CreatedByType),
		handoff.UpdatedByActor,
		string(handoff.UpdatedByType),
		ts(handoff.UpdatedAt),
		handoff.ResolvedByActor,
		normalizeOptionalActorType(handoff.ResolvedByType),
		nullableTS(handoff.ResolvedAt),
		handoff.ResolutionNote,
		handoff.ID,
	)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetHandoff returns one handoff row by id.
func (r *Repository) GetHandoff(ctx context.Context, handoffID string) (domain.Handoff, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, project_id, branch_id, scope_type, scope_id, target_branch_id, target_scope_type, target_scope_id,
			source_role, target_role, status, summary, next_action, missing_evidence_json, related_refs_json,
			created_by_actor, created_by_type, created_at, updated_by_actor, updated_by_type, updated_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note
		FROM handoffs
		WHERE id = ?
	`, strings.TrimSpace(handoffID))
	return scanHandoff(row)
}

// ListHandoffs lists matching handoff rows in deterministic order.
func (r *Repository) ListHandoffs(ctx context.Context, filter domain.HandoffListFilter) ([]domain.Handoff, error) {
	filter, err := normalizeHandoffListFilter(filter)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT
			id, project_id, branch_id, scope_type, scope_id, target_branch_id, target_scope_type, target_scope_id,
			source_role, target_role, status, summary, next_action, missing_evidence_json, related_refs_json,
			created_by_actor, created_by_type, created_at, updated_by_actor, updated_by_type, updated_at,
			resolved_by_actor, resolved_by_type, resolved_at, resolution_note
		FROM handoffs
		WHERE project_id = ?
	`
	args := []any{filter.ProjectID}
	if filter.BranchID != "" {
		query += ` AND branch_id = ?`
		args = append(args, filter.BranchID)
	}
	if filter.ScopeType != "" {
		query += ` AND scope_type = ?`
		args = append(args, string(filter.ScopeType))
	}
	if filter.ScopeID != "" {
		query += ` AND scope_id = ?`
		args = append(args, filter.ScopeID)
	}
	if len(filter.Statuses) > 0 {
		query += ` AND status IN (` + strings.TrimSuffix(strings.Repeat("?,", len(filter.Statuses)), ",") + `)`
		for _, status := range filter.Statuses {
			args = append(args, string(status))
		}
	}
	query += ` ORDER BY updated_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Handoff, 0)
	for rows.Next() {
		handoff, scanErr := scanHandoff(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, handoff)
	}
	return out, rows.Err()
}

// normalizeHandoffForWrite validates and normalizes one handoff row.
func normalizeHandoffForWrite(handoff domain.Handoff) (domain.Handoff, error) {
	handoff.ID = strings.TrimSpace(handoff.ID)
	handoff.ProjectID = strings.TrimSpace(handoff.ProjectID)
	handoff.BranchID = strings.TrimSpace(handoff.BranchID)
	handoff.ScopeType = domain.NormalizeScopeLevel(handoff.ScopeType)
	handoff.ScopeID = strings.TrimSpace(handoff.ScopeID)
	handoff.TargetBranchID = strings.TrimSpace(handoff.TargetBranchID)
	handoff.TargetScopeType = domain.NormalizeScopeLevel(handoff.TargetScopeType)
	handoff.TargetScopeID = strings.TrimSpace(handoff.TargetScopeID)
	handoff.SourceRole = strings.TrimSpace(strings.ToLower(handoff.SourceRole))
	handoff.TargetRole = strings.TrimSpace(strings.ToLower(handoff.TargetRole))
	handoff.Status = domain.NormalizeHandoffStatus(handoff.Status)
	handoff.Summary = strings.TrimSpace(handoff.Summary)
	handoff.NextAction = strings.TrimSpace(handoff.NextAction)
	handoff.CreatedByActor = chooseActorID(handoff.CreatedByActor)
	handoff.CreatedByType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(handoff.CreatedByType))))
	handoff.UpdatedByActor = chooseActorID(handoff.UpdatedByActor, handoff.CreatedByActor)
	handoff.UpdatedByType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(handoff.UpdatedByType))))
	handoff.ResolvedByActor = strings.TrimSpace(handoff.ResolvedByActor)
	handoff.ResolvedByType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(handoff.ResolvedByType))))
	handoff.ResolutionNote = strings.TrimSpace(handoff.ResolutionNote)
	handoff.MissingEvidence = normalizeStringSlice(handoff.MissingEvidence)
	handoff.RelatedRefs = normalizeStringSlice(handoff.RelatedRefs)

	if handoff.ID == "" {
		return domain.Handoff{}, domain.ErrInvalidID
	}
	if handoff.ProjectID == "" {
		return domain.Handoff{}, domain.ErrInvalidID
	}
	level, err := domain.NewLevelTuple(domain.LevelTupleInput{
		ProjectID: handoff.ProjectID,
		BranchID:  handoff.BranchID,
		ScopeType: handoff.ScopeType,
		ScopeID:   handoff.ScopeID,
	})
	if err != nil {
		return domain.Handoff{}, err
	}
	handoff.ProjectID = level.ProjectID
	handoff.BranchID = level.BranchID
	handoff.ScopeType = level.ScopeType
	handoff.ScopeID = level.ScopeID
	if !domain.IsValidHandoffStatus(handoff.Status) {
		return domain.Handoff{}, domain.ErrInvalidHandoffStatus
	}
	if handoff.Summary == "" {
		return domain.Handoff{}, domain.ErrInvalidSummary
	}
	if handoff.CreatedAt.IsZero() || handoff.UpdatedAt.IsZero() {
		return domain.Handoff{}, domain.ErrInvalidID
	}
	if !isStoredActorType(handoff.CreatedByType) || !isStoredActorType(handoff.UpdatedByType) {
		return domain.Handoff{}, domain.ErrInvalidActorType
	}
	if strings.TrimSpace(string(handoff.ResolvedByType)) != "" && !isStoredActorType(handoff.ResolvedByType) {
		return domain.Handoff{}, domain.ErrInvalidActorType
	}

	targetLevel, hasTarget, err := normalizeHandoffTarget(handoff.ProjectID, handoff.TargetBranchID, handoff.TargetScopeType, handoff.TargetScopeID)
	if err != nil {
		return domain.Handoff{}, err
	}
	if hasTarget {
		handoff.TargetBranchID = targetLevel.BranchID
		handoff.TargetScopeType = targetLevel.ScopeType
		handoff.TargetScopeID = targetLevel.ScopeID
	} else {
		handoff.TargetBranchID = ""
		handoff.TargetScopeType = ""
		handoff.TargetScopeID = ""
	}

	if domain.IsTerminalHandoffStatus(handoff.Status) {
		if handoff.ResolvedAt == nil {
			resolvedAt := handoff.UpdatedAt.UTC()
			handoff.ResolvedAt = &resolvedAt
		}
		if handoff.ResolvedByActor == "" {
			handoff.ResolvedByActor = handoff.UpdatedByActor
		}
		if handoff.ResolvedByType == "" {
			handoff.ResolvedByType = handoff.UpdatedByType
		}
	} else {
		handoff.ResolvedByActor = ""
		handoff.ResolvedByType = ""
		handoff.ResolvedAt = nil
		handoff.ResolutionNote = ""
	}
	return handoff, nil
}

// normalizeHandoffListFilter validates and normalizes one handoff list filter.
func normalizeHandoffListFilter(filter domain.HandoffListFilter) (domain.HandoffListFilter, error) {
	filter, err := domain.NormalizeHandoffListFilter(filter)
	if err != nil {
		return domain.HandoffListFilter{}, err
	}
	if filter.ScopeType != "" || filter.ScopeID != "" || filter.BranchID != "" {
		level, err := domain.NewLevelTuple(domain.LevelTupleInput{
			ProjectID: filter.ProjectID,
			BranchID:  filter.BranchID,
			ScopeType: filter.ScopeType,
			ScopeID:   filter.ScopeID,
		})
		if err != nil {
			return domain.HandoffListFilter{}, err
		}
		filter.ProjectID = level.ProjectID
		filter.BranchID = level.BranchID
		filter.ScopeType = level.ScopeType
		filter.ScopeID = level.ScopeID
	}
	return filter, nil
}

// normalizeHandoffTarget validates one optional target tuple.
func normalizeHandoffTarget(projectID, branchID string, scopeType domain.ScopeLevel, scopeID string) (domain.LevelTuple, bool, error) {
	if strings.TrimSpace(branchID) == "" && strings.TrimSpace(string(scopeType)) == "" && strings.TrimSpace(scopeID) == "" {
		return domain.LevelTuple{}, false, nil
	}
	targetLevel, err := domain.NewLevelTuple(domain.LevelTupleInput{
		ProjectID: projectID,
		BranchID:  branchID,
		ScopeType: scopeType,
		ScopeID:   scopeID,
	})
	if err != nil {
		return domain.LevelTuple{}, false, err
	}
	return targetLevel, true, nil
}

// scanHandoff decodes one handoffs row.
func scanHandoff(s scanner) (domain.Handoff, error) {
	var (
		handoff            domain.Handoff
		scopeTypeRaw       string
		targetScopeTypeRaw string
		statusRaw          string
		missingEvidenceRaw string
		relatedRefsRaw     string
		createdByTypeRaw   string
		createdRaw         string
		updatedByTypeRaw   string
		updatedRaw         string
		resolvedByTypeRaw  string
		resolvedAtRaw      sql.NullString
	)
	if err := s.Scan(
		&handoff.ID,
		&handoff.ProjectID,
		&handoff.BranchID,
		&scopeTypeRaw,
		&handoff.ScopeID,
		&handoff.TargetBranchID,
		&targetScopeTypeRaw,
		&handoff.TargetScopeID,
		&handoff.SourceRole,
		&handoff.TargetRole,
		&statusRaw,
		&handoff.Summary,
		&handoff.NextAction,
		&missingEvidenceRaw,
		&relatedRefsRaw,
		&handoff.CreatedByActor,
		&createdByTypeRaw,
		&createdRaw,
		&handoff.UpdatedByActor,
		&updatedByTypeRaw,
		&updatedRaw,
		&handoff.ResolvedByActor,
		&resolvedByTypeRaw,
		&resolvedAtRaw,
		&handoff.ResolutionNote,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Handoff{}, app.ErrNotFound
		}
		return domain.Handoff{}, err
	}
	handoff.ScopeType = domain.NormalizeScopeLevel(domain.ScopeLevel(scopeTypeRaw))
	handoff.TargetScopeType = domain.NormalizeScopeLevel(domain.ScopeLevel(targetScopeTypeRaw))
	handoff.SourceRole = strings.TrimSpace(strings.ToLower(handoff.SourceRole))
	handoff.TargetRole = strings.TrimSpace(strings.ToLower(handoff.TargetRole))
	handoff.Status = domain.NormalizeHandoffStatus(domain.HandoffStatus(statusRaw))
	handoff.CreatedByType = normalizeActorType(domain.ActorType(createdByTypeRaw))
	handoff.UpdatedByType = normalizeActorType(domain.ActorType(updatedByTypeRaw))
	handoff.ResolvedByType = domain.ActorType(strings.TrimSpace(strings.ToLower(resolvedByTypeRaw)))
	handoff.CreatedAt = parseTS(createdRaw)
	handoff.UpdatedAt = parseTS(updatedRaw)
	handoff.ResolvedAt = parseNullTS(resolvedAtRaw)
	if strings.TrimSpace(missingEvidenceRaw) == "" {
		missingEvidenceRaw = "[]"
	}
	if err := json.Unmarshal([]byte(missingEvidenceRaw), &handoff.MissingEvidence); err != nil {
		return domain.Handoff{}, fmt.Errorf("decode handoffs.missing_evidence_json: %w", err)
	}
	if strings.TrimSpace(relatedRefsRaw) == "" {
		relatedRefsRaw = "[]"
	}
	if err := json.Unmarshal([]byte(relatedRefsRaw), &handoff.RelatedRefs); err != nil {
		return domain.Handoff{}, fmt.Errorf("decode handoffs.related_refs_json: %w", err)
	}
	return handoff, nil
}

// isStoredActorType reports whether one stored actor type is valid without applying defaults.
func isStoredActorType(actorType domain.ActorType) bool {
	switch actorType {
	case domain.ActorTypeUser, domain.ActorTypeAgent, domain.ActorTypeSystem:
		return true
	default:
		return false
	}
}

// normalizeStringSlice trims and deduplicates one string slice.
func normalizeStringSlice(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// mustJSON marshals a value that should always be serializable in this lane.
func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
