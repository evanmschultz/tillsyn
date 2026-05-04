package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// SnapshotVersion defines the canonical snapshot schema version.
const SnapshotVersion = "tillsyn.snapshot.v5"

// Snapshot represents snapshot data used by this package.
type Snapshot struct {
	Version             string                        `json:"version"`
	ExportedAt          time.Time                     `json:"exported_at"`
	Projects            []SnapshotProject             `json:"projects"`
	Columns             []SnapshotColumn              `json:"columns"`
	ActionItems         []SnapshotActionItem          `json:"tasks"`
	KindDefinitions     []SnapshotKindDefinition      `json:"kind_definitions,omitempty"`
	ProjectAllowedKinds []SnapshotProjectAllowedKinds `json:"project_allowed_kinds,omitempty"`
	Comments            []SnapshotComment             `json:"comments,omitempty"`
	CapabilityLeases    []SnapshotCapabilityLease     `json:"capability_leases,omitempty"`
	Handoffs            []SnapshotHandoff             `json:"handoffs,omitempty"`
}

// SnapshotProject represents snapshot project data used by this package.
type SnapshotProject struct {
	ID          string                 `json:"id"`
	Slug        string                 `json:"slug"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Metadata    domain.ProjectMetadata `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ArchivedAt  *time.Time             `json:"archived_at,omitempty"`
}

// SnapshotColumn represents snapshot column data used by this package.
type SnapshotColumn struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	Name       string     `json:"name"`
	WIPLimit   int        `json:"wip_limit"`
	Position   int        `json:"position"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

// SnapshotActionItem represents snapshot actionItem data used by this package.
type SnapshotActionItem struct {
	ID             string                    `json:"id"`
	ProjectID      string                    `json:"project_id"`
	ParentID       string                    `json:"parent_id,omitempty"`
	Kind           domain.Kind               `json:"kind"`
	Scope          domain.KindAppliesTo      `json:"scope,omitempty"`
	Role           domain.Role               `json:"role,omitempty"`
	StructuralType domain.StructuralType     `json:"structural_type,omitempty"`
	Irreducible    bool                      `json:"irreducible,omitempty"`
	Owner          string                    `json:"owner,omitempty"`
	DropNumber     int                       `json:"drop_number,omitempty"`
	Persistent     bool                      `json:"persistent,omitempty"`
	DevGated       bool                      `json:"dev_gated,omitempty"`
	Paths          []string                  `json:"paths,omitempty"`
	Packages       []string                  `json:"packages,omitempty"`
	Files          []string                  `json:"files,omitempty"`
	StartCommit    string                    `json:"start_commit,omitempty"`
	LifecycleState domain.LifecycleState     `json:"lifecycle_state"`
	ColumnID       string                    `json:"column_id"`
	Position       int                       `json:"position"`
	Title          string                    `json:"title"`
	Description    string                    `json:"description"`
	Priority       domain.Priority           `json:"priority"`
	DueAt          *time.Time                `json:"due_at,omitempty"`
	Labels         []string                  `json:"labels"`
	Metadata       domain.ActionItemMetadata `json:"metadata"`
	CreatedByActor string                    `json:"created_by_actor"`
	CreatedByName  string                    `json:"created_by_name,omitempty"`
	UpdatedByActor string                    `json:"updated_by_actor"`
	UpdatedByName  string                    `json:"updated_by_name,omitempty"`
	UpdatedByType  domain.ActorType          `json:"updated_by_type"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
	StartedAt      *time.Time                `json:"started_at,omitempty"`
	CompletedAt    *time.Time                `json:"completed_at,omitempty"`
	ArchivedAt     *time.Time                `json:"archived_at,omitempty"`
	CanceledAt     *time.Time                `json:"canceled_at,omitempty"`
}

// SnapshotKindDefinition represents one kind-catalog definition persisted in a snapshot.
//
// Per Drop 3 droplet 3.15 the legacy AllowedParentScopes + Template fields
// were removed; nesting now flows through templates.Template.AllowsNesting +
// the project's baked KindCatalog. Pre-Drop-3 snapshots that carry those
// fields are silently dropped on import (no migration logic per pre-MVP
// rule — dev fresh-DBs).
type SnapshotKindDefinition struct {
	ID                  domain.KindID          `json:"id"`
	DisplayName         string                 `json:"display_name"`
	DescriptionMarkdown string                 `json:"description_markdown"`
	AppliesTo           []domain.KindAppliesTo `json:"applies_to"`
	PayloadSchemaJSON   string                 `json:"payload_schema_json,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	ArchivedAt          *time.Time             `json:"archived_at,omitempty"`
}

// SnapshotProjectAllowedKinds stores one project's explicit kind allowlist closure.
type SnapshotProjectAllowedKinds struct {
	ProjectID string          `json:"project_id"`
	KindIDs   []domain.KindID `json:"kind_ids"`
}

// SnapshotComment represents one persisted markdown comment row in a snapshot.
type SnapshotComment struct {
	ID           string                   `json:"id"`
	ProjectID    string                   `json:"project_id"`
	TargetType   domain.CommentTargetType `json:"target_type"`
	TargetID     string                   `json:"target_id"`
	Summary      string                   `json:"summary"`
	BodyMarkdown string                   `json:"body_markdown"`
	ActorID      string                   `json:"actor_id"`
	ActorName    string                   `json:"actor_name"`
	ActorType    domain.ActorType         `json:"actor_type"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

// SnapshotCapabilityLease represents one persisted capability-lease row in a snapshot.
type SnapshotCapabilityLease struct {
	InstanceID                string                     `json:"instance_id"`
	LeaseToken                string                     `json:"lease_token"`
	AgentName                 string                     `json:"agent_name"`
	ProjectID                 string                     `json:"project_id"`
	ScopeType                 domain.CapabilityScopeType `json:"scope_type"`
	ScopeID                   string                     `json:"scope_id,omitempty"`
	Role                      domain.CapabilityRole      `json:"role"`
	ParentInstanceID          string                     `json:"parent_instance_id,omitempty"`
	AllowEqualScopeDelegation bool                       `json:"allow_equal_scope_delegation"`
	IssuedAt                  time.Time                  `json:"issued_at"`
	ExpiresAt                 time.Time                  `json:"expires_at"`
	HeartbeatAt               time.Time                  `json:"heartbeat_at"`
	RevokedAt                 *time.Time                 `json:"revoked_at,omitempty"`
	RevokedReason             string                     `json:"revoked_reason,omitempty"`
}

// SnapshotHandoff represents one durable handoff row in a snapshot.
type SnapshotHandoff struct {
	ID              string               `json:"id"`
	ProjectID       string               `json:"project_id"`
	BranchID        string               `json:"branch_id,omitempty"`
	ScopeType       domain.ScopeLevel    `json:"scope_type"`
	ScopeID         string               `json:"scope_id"`
	SourceRole      string               `json:"source_role,omitempty"`
	TargetBranchID  string               `json:"target_branch_id,omitempty"`
	TargetScopeType domain.ScopeLevel    `json:"target_scope_type,omitempty"`
	TargetScopeID   string               `json:"target_scope_id,omitempty"`
	TargetRole      string               `json:"target_role,omitempty"`
	Status          domain.HandoffStatus `json:"status"`
	Summary         string               `json:"summary"`
	NextAction      string               `json:"next_action,omitempty"`
	MissingEvidence []string             `json:"missing_evidence,omitempty"`
	RelatedRefs     []string             `json:"related_refs,omitempty"`
	CreatedByActor  string               `json:"created_by_actor"`
	CreatedByType   domain.ActorType     `json:"created_by_type"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedByActor  string               `json:"updated_by_actor"`
	UpdatedByType   domain.ActorType     `json:"updated_by_type"`
	UpdatedAt       time.Time            `json:"updated_at"`
	ResolvedByActor string               `json:"resolved_by_actor,omitempty"`
	ResolvedByType  domain.ActorType     `json:"resolved_by_type,omitempty"`
	ResolvedAt      *time.Time           `json:"resolved_at,omitempty"`
	ResolutionNote  string               `json:"resolution_note,omitempty"`
}

// ExportSnapshot handles export snapshot.
func (s *Service) ExportSnapshot(ctx context.Context, includeArchived bool) (Snapshot, error) {
	kindDefinitions, err := s.repo.ListKindDefinitions(ctx, includeArchived)
	if err != nil {
		return Snapshot{}, err
	}

	projects, err := s.repo.ListProjects(ctx, includeArchived)
	if err != nil {
		return Snapshot{}, err
	}

	snap := Snapshot{
		Version:             SnapshotVersion,
		ExportedAt:          s.clock().UTC(),
		Projects:            make([]SnapshotProject, 0, len(projects)),
		Columns:             make([]SnapshotColumn, 0),
		ActionItems:         make([]SnapshotActionItem, 0),
		KindDefinitions:     make([]SnapshotKindDefinition, 0, len(kindDefinitions)),
		ProjectAllowedKinds: make([]SnapshotProjectAllowedKinds, 0, len(projects)),
		Comments:            make([]SnapshotComment, 0),
		CapabilityLeases:    make([]SnapshotCapabilityLease, 0),
		Handoffs:            make([]SnapshotHandoff, 0),
	}
	for _, kind := range kindDefinitions {
		snap.KindDefinitions = append(snap.KindDefinitions, snapshotKindDefinitionFromDomain(kind))
	}
	for _, project := range projects {
		snap.Projects = append(snap.Projects, snapshotProjectFromDomain(project))

		allowedKinds, listErr := s.repo.ListProjectAllowedKinds(ctx, project.ID)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		if len(allowedKinds) > 0 {
			snap.ProjectAllowedKinds = append(snap.ProjectAllowedKinds, SnapshotProjectAllowedKinds{
				ProjectID: project.ID,
				KindIDs:   append([]domain.KindID(nil), allowedKinds...),
			})
		}

		columns, listErr := s.repo.ListColumns(ctx, project.ID, includeArchived)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		for _, column := range columns {
			snap.Columns = append(snap.Columns, snapshotColumnFromDomain(column))
		}

		tasks, listErr := s.repo.ListActionItems(ctx, project.ID, includeArchived)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		for _, actionItem := range tasks {
			snap.ActionItems = append(snap.ActionItems, snapshotActionItemFromDomain(actionItem))
		}

		comments, listErr := s.commentsForProjectSnapshot(ctx, project, tasks)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		snap.Comments = append(snap.Comments, comments...)

		leases, listErr := s.capabilityLeasesForProjectSnapshot(ctx, project.ID, tasks)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		snap.CapabilityLeases = append(snap.CapabilityLeases, leases...)

		handoffs, listErr := s.handoffsForProjectSnapshot(ctx, project.ID, tasks)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		snap.Handoffs = append(snap.Handoffs, handoffs...)
	}

	snap.sort()
	return snap, nil
}

// ImportSnapshot handles import snapshot.
func (s *Service) ImportSnapshot(ctx context.Context, snap Snapshot) error {
	if err := snap.Validate(); err != nil {
		return err
	}
	snap.sort()

	for _, project := range snap.Projects {
		if err := s.upsertProject(ctx, project.toDomain()); err != nil {
			return err
		}
	}
	for _, kind := range snap.KindDefinitions {
		if err := s.upsertKindDefinition(ctx, kind.toDomain()); err != nil {
			return err
		}
	}
	for _, allow := range snap.ProjectAllowedKinds {
		if err := s.repo.SetProjectAllowedKinds(ctx, strings.TrimSpace(allow.ProjectID), append([]domain.KindID(nil), allow.KindIDs...)); err != nil {
			return err
		}
	}

	existingColumnsByProject := map[string]map[string]struct{}{}
	for _, project := range snap.Projects {
		columns, err := s.repo.ListColumns(ctx, project.ID, true)
		if err != nil {
			return err
		}
		byID := map[string]struct{}{}
		for _, column := range columns {
			byID[column.ID] = struct{}{}
		}
		existingColumnsByProject[project.ID] = byID
	}

	for _, column := range snap.Columns {
		dc := column.toDomain()
		if _, ok := existingColumnsByProject[dc.ProjectID][dc.ID]; ok {
			if err := s.repo.UpdateColumn(ctx, dc); err != nil {
				return err
			}
			continue
		}
		if err := s.repo.CreateColumn(ctx, dc); err != nil {
			return err
		}
		existingColumnsByProject[dc.ProjectID][dc.ID] = struct{}{}
	}

	for _, actionItem := range snap.ActionItems {
		dt := actionItem.toDomain()
		if _, err := s.repo.GetActionItem(ctx, dt.ID); err == nil {
			if err := s.repo.UpdateActionItem(ctx, dt); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := s.repo.CreateActionItem(ctx, dt); err != nil {
			return err
		}
	}
	if err := s.importSnapshotComments(ctx, snap.Comments); err != nil {
		return err
	}
	if err := s.importSnapshotCapabilityLeases(ctx, snap.CapabilityLeases); err != nil {
		return err
	}
	if err := s.importSnapshotHandoffs(ctx, snap.Handoffs); err != nil {
		return err
	}

	return nil
}

// Validate validates the requested operation.
func (s *Snapshot) Validate() error {
	if strings.TrimSpace(s.Version) != SnapshotVersion {
		return fmt.Errorf("unsupported snapshot version: %q", s.Version)
	}

	projectIDs := map[string]struct{}{}
	for i, p := range s.Projects {
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("projects[%d].id is required", i)
		}
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("projects[%d].name is required", i)
		}
		if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
			return fmt.Errorf("projects[%d] timestamps are required", i)
		}
		if _, exists := projectIDs[p.ID]; exists {
			return fmt.Errorf("duplicate project id: %q", p.ID)
		}
		projectIDs[p.ID] = struct{}{}
	}

	columnIDs := map[string]struct{}{}
	for i, c := range s.Columns {
		if strings.TrimSpace(c.ID) == "" {
			return fmt.Errorf("columns[%d].id is required", i)
		}
		if strings.TrimSpace(c.ProjectID) == "" {
			return fmt.Errorf("columns[%d].project_id is required", i)
		}
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("columns[%d].name is required", i)
		}
		if c.Position < 0 {
			return fmt.Errorf("columns[%d].position must be >= 0", i)
		}
		if c.WIPLimit < 0 {
			return fmt.Errorf("columns[%d].wip_limit must be >= 0", i)
		}
		if c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() {
			return fmt.Errorf("columns[%d] timestamps are required", i)
		}
		if _, ok := projectIDs[c.ProjectID]; !ok {
			return fmt.Errorf("columns[%d] references unknown project_id %q", i, c.ProjectID)
		}
		if _, exists := columnIDs[c.ID]; exists {
			return fmt.Errorf("duplicate column id: %q", c.ID)
		}
		columnIDs[c.ID] = struct{}{}
	}

	actionItemIDs := map[string]struct{}{}
	actionItemByID := map[string]SnapshotActionItem{}
	for i, t := range s.ActionItems {
		if strings.TrimSpace(t.ID) == "" {
			return fmt.Errorf("tasks[%d].id is required", i)
		}
		if strings.TrimSpace(t.ProjectID) == "" {
			return fmt.Errorf("tasks[%d].project_id is required", i)
		}
		if strings.TrimSpace(t.ColumnID) == "" {
			return fmt.Errorf("tasks[%d].column_id is required", i)
		}
		if strings.TrimSpace(t.Title) == "" {
			return fmt.Errorf("tasks[%d].title is required", i)
		}
		if t.Position < 0 {
			return fmt.Errorf("tasks[%d].position must be >= 0", i)
		}
		switch t.Priority {
		case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh:
		default:
			return fmt.Errorf("tasks[%d].priority must be low|medium|high", i)
		}
		if strings.TrimSpace(string(t.Kind)) == "" {
			// Snapshot imports tolerate legacy rows with empty kind by falling
			// back to KindPlan, which is the lightest-weight member of the
			// 12-value Kind enum. Domain creation rejects empty kind; this
			// fallback keeps importers forward-compatible with older exports.
			t.Kind = domain.KindPlan
			s.ActionItems[i].Kind = t.Kind
		}
		if t.Scope == "" {
			t.Scope = domain.DefaultActionItemScope(t.Kind)
			s.ActionItems[i].Scope = t.Scope
		}
		if !domain.IsValidWorkItemAppliesTo(t.Scope) {
			return fmt.Errorf("tasks[%d].scope must be a member of the 12-value Kind enum", i)
		}
		if t.LifecycleState == "" {
			t.LifecycleState = domain.StateTodo
			s.ActionItems[i].LifecycleState = t.LifecycleState
		}
		switch t.LifecycleState {
		case domain.StateTodo, domain.StateInProgress, domain.StateComplete, domain.StateFailed, domain.StateArchived:
		default:
			return fmt.Errorf("tasks[%d].lifecycle_state must be todo|in_progress|complete|failed|archived", i)
		}
		if t.CreatedAt.IsZero() || t.UpdatedAt.IsZero() {
			return fmt.Errorf("tasks[%d] timestamps are required", i)
		}
		if _, ok := projectIDs[t.ProjectID]; !ok {
			return fmt.Errorf("tasks[%d] references unknown project_id %q", i, t.ProjectID)
		}
		if _, ok := columnIDs[t.ColumnID]; !ok {
			return fmt.Errorf("tasks[%d] references unknown column_id %q", i, t.ColumnID)
		}
		if _, exists := actionItemIDs[t.ID]; exists {
			return fmt.Errorf("duplicate actionItem id: %q", t.ID)
		}
		actionItemIDs[t.ID] = struct{}{}
		actionItemByID[t.ID] = s.ActionItems[i]
	}
	for i, t := range s.ActionItems {
		if strings.TrimSpace(t.ParentID) == "" {
			continue
		}
		if t.ParentID == t.ID {
			return fmt.Errorf("tasks[%d].parent_id cannot reference itself", i)
		}
		if _, exists := actionItemIDs[t.ParentID]; !exists {
			return fmt.Errorf("tasks[%d] references unknown parent_id %q", i, t.ParentID)
		}
		// Parent-scope constraints are enforced by
		// templates.KindCatalog.AllowsNesting at action-item creation (per
		// Drop 3 droplet 3.15). Snapshot validation does not duplicate that
		// gate; an unknown-parent-id check above is the only structural
		// invariant snapshot import enforces here.
		_ = actionItemByID[t.ParentID]
	}

	kindIDs := map[domain.KindID]struct{}{}
	for i, k := range s.KindDefinitions {
		kindID := domain.NormalizeKindID(k.ID)
		if kindID == "" {
			return fmt.Errorf("kind_definitions[%d].id is required", i)
		}
		if _, exists := kindIDs[kindID]; exists {
			return fmt.Errorf("duplicate kind definition id: %q", kindID)
		}
		if k.CreatedAt.IsZero() || k.UpdatedAt.IsZero() {
			return fmt.Errorf("kind_definitions[%d] timestamps are required", i)
		}
		if strings.TrimSpace(k.PayloadSchemaJSON) != "" && !json.Valid([]byte(k.PayloadSchemaJSON)) {
			return fmt.Errorf("kind_definitions[%d].payload_schema_json invalid", i)
		}
		s.KindDefinitions[i].ID = kindID
		kindIDs[kindID] = struct{}{}
	}

	allowlistByProject := map[string]struct{}{}
	for i, allow := range s.ProjectAllowedKinds {
		projectID := strings.TrimSpace(allow.ProjectID)
		if projectID == "" {
			return fmt.Errorf("project_allowed_kinds[%d].project_id is required", i)
		}
		if _, ok := projectIDs[projectID]; !ok {
			return fmt.Errorf("project_allowed_kinds[%d] references unknown project_id %q", i, projectID)
		}
		if _, exists := allowlistByProject[projectID]; exists {
			return fmt.Errorf("duplicate project_allowed_kinds project_id: %q", projectID)
		}
		seenKinds := map[domain.KindID]struct{}{}
		normalizedKinds := make([]domain.KindID, 0, len(allow.KindIDs))
		for _, rawKindID := range allow.KindIDs {
			kindID := domain.NormalizeKindID(rawKindID)
			if kindID == "" {
				continue
			}
			if _, dup := seenKinds[kindID]; dup {
				continue
			}
			seenKinds[kindID] = struct{}{}
			normalizedKinds = append(normalizedKinds, kindID)
		}
		s.ProjectAllowedKinds[i].ProjectID = projectID
		s.ProjectAllowedKinds[i].KindIDs = normalizedKinds
		allowlistByProject[projectID] = struct{}{}
	}
	commentKeys := map[string]struct{}{}
	for i, c := range s.Comments {
		commentID := strings.TrimSpace(c.ID)
		if commentID == "" {
			return fmt.Errorf("comments[%d].id is required", i)
		}
		target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
			ProjectID:  c.ProjectID,
			TargetType: c.TargetType,
			TargetID:   c.TargetID,
		})
		if err != nil {
			return fmt.Errorf("comments[%d] target invalid: %w", i, err)
		}
		if _, ok := projectIDs[target.ProjectID]; !ok {
			return fmt.Errorf("comments[%d] references unknown project_id %q", i, target.ProjectID)
		}
		body := strings.TrimSpace(c.BodyMarkdown)
		if body == "" {
			return fmt.Errorf("comments[%d].body_markdown is required", i)
		}
		summary := domain.NormalizeCommentSummary(c.Summary, body)
		if summary == "" {
			return fmt.Errorf("comments[%d].summary is required", i)
		}
		actorType := domain.ActorType(strings.TrimSpace(strings.ToLower(string(c.ActorType))))
		if actorType == "" {
			actorType = domain.ActorTypeUser
		}
		if !isSupportedActorType(actorType) {
			return fmt.Errorf("comments[%d].actor_type invalid: %q", i, actorType)
		}
		actorID := strings.TrimSpace(c.ActorID)
		if actorID == "" {
			actorID = "tillsyn-user"
		}
		actorName := strings.TrimSpace(c.ActorName)
		if actorName == "" {
			actorName = actorID
		}
		if c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() {
			return fmt.Errorf("comments[%d] timestamps are required", i)
		}
		commentKey := strings.Join([]string{target.ProjectID, string(target.TargetType), target.TargetID, commentID}, "|")
		if _, exists := commentKeys[commentKey]; exists {
			return fmt.Errorf("duplicate comment identity: %q", commentKey)
		}
		commentKeys[commentKey] = struct{}{}
		s.Comments[i].ID = commentID
		s.Comments[i].ProjectID = target.ProjectID
		s.Comments[i].TargetType = target.TargetType
		s.Comments[i].TargetID = target.TargetID
		s.Comments[i].Summary = summary
		s.Comments[i].BodyMarkdown = body
		s.Comments[i].ActorID = actorID
		s.Comments[i].ActorName = actorName
		s.Comments[i].ActorType = actorType
	}

	leaseIDs := map[string]struct{}{}
	for i, lease := range s.CapabilityLeases {
		instanceID := strings.TrimSpace(lease.InstanceID)
		if instanceID == "" {
			return fmt.Errorf("capability_leases[%d].instance_id is required", i)
		}
		if _, exists := leaseIDs[instanceID]; exists {
			return fmt.Errorf("duplicate capability lease instance_id: %q", instanceID)
		}
		projectID := strings.TrimSpace(lease.ProjectID)
		if projectID == "" {
			return fmt.Errorf("capability_leases[%d].project_id is required", i)
		}
		if _, ok := projectIDs[projectID]; !ok {
			return fmt.Errorf("capability_leases[%d] references unknown project_id %q", i, projectID)
		}
		scopeType := domain.NormalizeCapabilityScopeType(lease.ScopeType)
		if !domain.IsValidCapabilityScopeType(scopeType) {
			return fmt.Errorf("capability_leases[%d].scope_type invalid: %q", i, lease.ScopeType)
		}
		scopeID := strings.TrimSpace(lease.ScopeID)
		if scopeType != domain.CapabilityScopeProject && scopeID == "" {
			return fmt.Errorf("capability_leases[%d].scope_id is required for scope %q", i, scopeType)
		}
		role := domain.NormalizeCapabilityRole(lease.Role)
		if !domain.IsValidCapabilityRole(role) {
			return fmt.Errorf("capability_leases[%d].role invalid: %q", i, lease.Role)
		}
		if strings.TrimSpace(lease.LeaseToken) == "" {
			return fmt.Errorf("capability_leases[%d].lease_token is required", i)
		}
		if strings.TrimSpace(lease.AgentName) == "" {
			return fmt.Errorf("capability_leases[%d].agent_name is required", i)
		}
		if lease.IssuedAt.IsZero() || lease.ExpiresAt.IsZero() || lease.HeartbeatAt.IsZero() {
			return fmt.Errorf("capability_leases[%d] timestamps are required", i)
		}
		s.CapabilityLeases[i].InstanceID = instanceID
		s.CapabilityLeases[i].ProjectID = projectID
		s.CapabilityLeases[i].ScopeType = scopeType
		s.CapabilityLeases[i].ScopeID = scopeID
		s.CapabilityLeases[i].Role = role
		s.CapabilityLeases[i].LeaseToken = strings.TrimSpace(lease.LeaseToken)
		s.CapabilityLeases[i].AgentName = strings.TrimSpace(lease.AgentName)
		s.CapabilityLeases[i].ParentInstanceID = strings.TrimSpace(lease.ParentInstanceID)
		s.CapabilityLeases[i].RevokedReason = strings.TrimSpace(lease.RevokedReason)
		leaseIDs[instanceID] = struct{}{}
	}

	availableHandoffScopes := snapshotAvailableHandoffScopes(s.Projects, s.ActionItems)
	handoffIDs := map[string]struct{}{}
	for i, handoff := range s.Handoffs {
		handoffID := strings.TrimSpace(handoff.ID)
		if handoffID == "" {
			return fmt.Errorf("handoffs[%d].id is required", i)
		}
		if _, exists := handoffIDs[handoffID]; exists {
			return fmt.Errorf("duplicate handoff id: %q", handoffID)
		}
		source, err := domain.NewLevelTuple(domain.LevelTupleInput{
			ProjectID: handoff.ProjectID,
			BranchID:  handoff.BranchID,
			ScopeType: handoff.ScopeType,
			ScopeID:   handoff.ScopeID,
		})
		if err != nil {
			return fmt.Errorf("handoffs[%d] source invalid: %w", i, err)
		}
		if _, ok := projectIDs[source.ProjectID]; !ok {
			return fmt.Errorf("handoffs[%d] references unknown project_id %q", i, source.ProjectID)
		}
		if _, ok := availableHandoffScopes[snapshotHandoffScopeKey(source.ProjectID, source.ScopeType, source.ScopeID)]; !ok {
			return fmt.Errorf("handoffs[%d] references unknown source scope %q:%q", i, source.ScopeType, source.ScopeID)
		}
		if strings.TrimSpace(handoff.Summary) == "" {
			return fmt.Errorf("handoffs[%d].summary is required", i)
		}
		status := domain.NormalizeHandoffStatus(handoff.Status)
		if !domain.IsValidHandoffStatus(status) {
			return fmt.Errorf("handoffs[%d].status invalid: %q", i, handoff.Status)
		}
		if handoff.CreatedAt.IsZero() || handoff.UpdatedAt.IsZero() {
			return fmt.Errorf("handoffs[%d] timestamps are required", i)
		}
		createdByType := domain.ActorType(strings.TrimSpace(strings.ToLower(string(handoff.CreatedByType))))
		if createdByType == "" {
			createdByType = domain.ActorTypeUser
		}
		if !isSupportedActorType(createdByType) {
			return fmt.Errorf("handoffs[%d].created_by_type invalid: %q", i, handoff.CreatedByType)
		}
		updatedByType := domain.ActorType(strings.TrimSpace(strings.ToLower(string(handoff.UpdatedByType))))
		if updatedByType == "" {
			updatedByType = createdByType
		}
		if !isSupportedActorType(updatedByType) {
			return fmt.Errorf("handoffs[%d].updated_by_type invalid: %q", i, handoff.UpdatedByType)
		}
		if domain.IsTerminalHandoffStatus(status) {
			if handoff.ResolvedAt == nil || handoff.ResolvedAt.IsZero() {
				return fmt.Errorf("handoffs[%d].resolved_at is required for terminal status", i)
			}
			if strings.TrimSpace(handoff.ResolvedByActor) == "" {
				return fmt.Errorf("handoffs[%d].resolved_by_actor is required for terminal status", i)
			}
			resolvedByType := domain.ActorType(strings.TrimSpace(strings.ToLower(string(handoff.ResolvedByType))))
			if resolvedByType == "" {
				resolvedByType = updatedByType
			}
			if !isSupportedActorType(resolvedByType) {
				return fmt.Errorf("handoffs[%d].resolved_by_type invalid: %q", i, handoff.ResolvedByType)
			}
			s.Handoffs[i].ResolvedByType = resolvedByType
		} else if handoff.ResolvedAt != nil {
			return fmt.Errorf("handoffs[%d].resolved_at must be empty for non-terminal status", i)
		}
		target, err := normalizeHandoffSnapshotTarget(source.ProjectID, handoff)
		if err != nil {
			return fmt.Errorf("handoffs[%d] target invalid: %w", i, err)
		}
		if target.ScopeType != "" {
			if _, ok := availableHandoffScopes[snapshotHandoffScopeKey(target.ProjectID, target.ScopeType, target.ScopeID)]; !ok {
				return fmt.Errorf("handoffs[%d] references unknown target scope %q:%q", i, target.ScopeType, target.ScopeID)
			}
		}
		s.Handoffs[i].ID = handoffID
		s.Handoffs[i].ProjectID = source.ProjectID
		s.Handoffs[i].BranchID = source.BranchID
		s.Handoffs[i].ScopeType = source.ScopeType
		s.Handoffs[i].ScopeID = source.ScopeID
		s.Handoffs[i].SourceRole = normalizeHandoffRole(handoff.SourceRole)
		s.Handoffs[i].TargetBranchID = strings.TrimSpace(handoff.TargetBranchID)
		s.Handoffs[i].TargetScopeType = domain.NormalizeScopeLevel(handoff.TargetScopeType)
		s.Handoffs[i].TargetScopeID = strings.TrimSpace(handoff.TargetScopeID)
		s.Handoffs[i].TargetRole = normalizeHandoffRole(handoff.TargetRole)
		s.Handoffs[i].Status = status
		s.Handoffs[i].Summary = strings.TrimSpace(handoff.Summary)
		s.Handoffs[i].NextAction = strings.TrimSpace(handoff.NextAction)
		s.Handoffs[i].CreatedByActor = chooseActorID(handoff.CreatedByActor)
		s.Handoffs[i].CreatedByType = createdByType
		s.Handoffs[i].UpdatedByActor = chooseActorID(handoff.UpdatedByActor, handoff.CreatedByActor)
		s.Handoffs[i].UpdatedByType = updatedByType
		s.Handoffs[i].MissingEvidence = normalizeHandoffList(handoff.MissingEvidence)
		s.Handoffs[i].RelatedRefs = normalizeHandoffList(handoff.RelatedRefs)
		s.Handoffs[i].ResolvedByActor = strings.TrimSpace(handoff.ResolvedByActor)
		s.Handoffs[i].ResolutionNote = strings.TrimSpace(handoff.ResolutionNote)
		handoffIDs[handoffID] = struct{}{}
	}

	return nil
}

// upsertProject handles upsert project.
func (s *Service) upsertProject(ctx context.Context, p domain.Project) error {
	if _, err := s.repo.GetProject(ctx, p.ID); err == nil {
		return s.repo.UpdateProject(ctx, p)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	return s.repo.CreateProject(ctx, p)
}

// upsertKindDefinition upserts one kind-catalog definition row.
func (s *Service) upsertKindDefinition(ctx context.Context, kind domain.KindDefinition) error {
	if _, err := s.repo.GetKindDefinition(ctx, kind.ID); err == nil {
		return s.repo.UpdateKindDefinition(ctx, kind)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	return s.repo.CreateKindDefinition(ctx, kind)
}

// commentsForProjectSnapshot collects project and actionItem-scoped comments for snapshot export.
func (s *Service) commentsForProjectSnapshot(ctx context.Context, project domain.Project, tasks []domain.ActionItem) ([]SnapshotComment, error) {
	targets := []domain.CommentTarget{{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   project.ID,
	}}
	for _, actionItem := range tasks {
		targets = append(targets, domain.CommentTarget{
			ProjectID:  project.ID,
			TargetType: snapshotCommentTargetTypeForActionItem(actionItem),
			TargetID:   actionItem.ID,
		})
	}
	out := make([]SnapshotComment, 0)
	seenTargets := map[string]struct{}{}
	for _, target := range targets {
		key := strings.Join([]string{target.ProjectID, string(target.TargetType), target.TargetID}, "|")
		if _, exists := seenTargets[key]; exists {
			continue
		}
		seenTargets[key] = struct{}{}
		comments, err := s.repo.ListCommentsByTarget(ctx, target)
		if err != nil {
			return nil, err
		}
		for _, comment := range comments {
			out = append(out, snapshotCommentFromDomain(comment))
		}
	}
	return out, nil
}

// handoffsForProjectSnapshot collects durable handoffs for snapshot export.
func (s *Service) handoffsForProjectSnapshot(ctx context.Context, projectID string, tasks []domain.ActionItem) ([]SnapshotHandoff, error) {
	if s.handoffRepo == nil {
		return nil, nil
	}
	availableScopes := snapshotAvailableDomainHandoffScopes(projectID, tasks)
	handoffs, err := s.handoffRepo.ListHandoffs(ctx, domain.HandoffListFilter{
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]SnapshotHandoff, 0, len(handoffs))
	for _, handoff := range handoffs {
		if _, ok := availableScopes[snapshotHandoffScopeKey(handoff.ProjectID, handoff.ScopeType, handoff.ScopeID)]; !ok {
			continue
		}
		if handoff.TargetScopeType != "" {
			if _, ok := availableScopes[snapshotHandoffScopeKey(handoff.ProjectID, handoff.TargetScopeType, handoff.TargetScopeID)]; !ok {
				continue
			}
		}
		out = append(out, snapshotHandoffFromDomain(handoff))
	}
	return out, nil
}

// capabilityLeasesForProjectSnapshot collects project/actionItem hierarchy capability leases for snapshot export.
func (s *Service) capabilityLeasesForProjectSnapshot(ctx context.Context, projectID string, tasks []domain.ActionItem) ([]SnapshotCapabilityLease, error) {
	type scopeQuery struct {
		scopeType domain.CapabilityScopeType
		scopeID   string
	}
	queries := []scopeQuery{{
		scopeType: domain.CapabilityScopeProject,
		scopeID:   "",
	}}
	for _, actionItem := range tasks {
		queries = append(queries, scopeQuery{
			scopeType: snapshotCapabilityScopeTypeForActionItem(actionItem),
			scopeID:   actionItem.ID,
		})
	}
	out := make([]SnapshotCapabilityLease, 0)
	seenQueries := map[string]struct{}{}
	seenLeases := map[string]struct{}{}
	for _, query := range queries {
		queryKey := strings.Join([]string{string(query.scopeType), strings.TrimSpace(query.scopeID)}, "|")
		if _, exists := seenQueries[queryKey]; exists {
			continue
		}
		seenQueries[queryKey] = struct{}{}
		leases, err := s.repo.ListCapabilityLeasesByScope(ctx, projectID, query.scopeType, query.scopeID)
		if err != nil {
			return nil, err
		}
		for _, lease := range leases {
			instanceID := strings.TrimSpace(lease.InstanceID)
			if _, exists := seenLeases[instanceID]; exists {
				continue
			}
			seenLeases[instanceID] = struct{}{}
			out = append(out, snapshotCapabilityLeaseFromDomain(lease))
		}
	}
	return out, nil
}

// importSnapshotComments upserts snapshot comments by deterministic comment identity.
func (s *Service) importSnapshotComments(ctx context.Context, comments []SnapshotComment) error {
	for _, snapshotComment := range comments {
		comment := snapshotComment.toDomain()
		target := domain.CommentTarget{
			ProjectID:  comment.ProjectID,
			TargetType: comment.TargetType,
			TargetID:   comment.TargetID,
		}
		existing, err := s.repo.ListCommentsByTarget(ctx, target)
		if err != nil {
			return err
		}
		alreadyExists := false
		for _, existingComment := range existing {
			if existingComment.ID == comment.ID {
				alreadyExists = true
				break
			}
		}
		if alreadyExists {
			continue
		}
		if err := s.repo.CreateComment(ctx, comment); err != nil {
			return err
		}
	}
	return nil
}

// importSnapshotCapabilityLeases upserts snapshot capability leases by instance id.
func (s *Service) importSnapshotCapabilityLeases(ctx context.Context, leases []SnapshotCapabilityLease) error {
	for _, snapshotLease := range leases {
		lease := snapshotLease.toDomain()
		if _, err := s.repo.GetCapabilityLease(ctx, lease.InstanceID); err == nil {
			if err := s.repo.UpdateCapabilityLease(ctx, lease); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := s.repo.CreateCapabilityLease(ctx, lease); err != nil {
			return err
		}
	}
	return nil
}

// importSnapshotHandoffs upserts snapshot handoffs by deterministic handoff identity.
func (s *Service) importSnapshotHandoffs(ctx context.Context, handoffs []SnapshotHandoff) error {
	if s.handoffRepo == nil {
		return nil
	}
	for _, snapshotHandoff := range handoffs {
		handoff := snapshotHandoff.toDomain()
		if _, err := s.handoffRepo.GetHandoff(ctx, handoff.ID); err == nil {
			if err := s.handoffRepo.UpdateHandoff(ctx, handoff); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := s.handoffRepo.CreateHandoff(ctx, handoff); err != nil {
			return err
		}
	}
	return nil
}

// snapshotCommentTargetTypeForActionItem maps one work-item row to a comment
// target type. Comments now address the action item as a whole, regardless of
// which of the 12 kinds the row carries.
func snapshotCommentTargetTypeForActionItem(actionItem domain.ActionItem) domain.CommentTargetType {
	_ = actionItem
	return domain.CommentTargetTypeActionItem
}

// snapshotCapabilityScopeTypeForActionItem maps one work-item row to a
// capability scope type. Scope mirrors kind in the 12-value enum, so every
// action-item row resolves to CapabilityScopeActionItem.
func snapshotCapabilityScopeTypeForActionItem(actionItem domain.ActionItem) domain.CapabilityScopeType {
	_ = actionItem
	return domain.CapabilityScopeActionItem
}

// isSupportedActorType validates snapshot actor types for comments.
func isSupportedActorType(actorType domain.ActorType) bool {
	switch actorType {
	case domain.ActorTypeUser, domain.ActorTypeAgent, domain.ActorTypeSystem:
		return true
	default:
		return false
	}
}

// sort handles sort.
func (s *Snapshot) sort() {
	sort.Slice(s.Projects, func(i, j int) bool {
		return s.Projects[i].ID < s.Projects[j].ID
	})
	sort.Slice(s.Columns, func(i, j int) bool {
		a := s.Columns[i]
		b := s.Columns[j]
		if a.ProjectID == b.ProjectID {
			if a.Position == b.Position {
				return a.ID < b.ID
			}
			return a.Position < b.Position
		}
		return a.ProjectID < b.ProjectID
	})
	sort.Slice(s.ActionItems, func(i, j int) bool {
		a := s.ActionItems[i]
		b := s.ActionItems[j]
		if a.ProjectID == b.ProjectID {
			if a.ColumnID == b.ColumnID {
				if a.Position == b.Position {
					return a.ID < b.ID
				}
				return a.Position < b.Position
			}
			return a.ColumnID < b.ColumnID
		}
		return a.ProjectID < b.ProjectID
	})
	for i := range s.ProjectAllowedKinds {
		sort.Slice(s.ProjectAllowedKinds[i].KindIDs, func(a, b int) bool {
			return s.ProjectAllowedKinds[i].KindIDs[a] < s.ProjectAllowedKinds[i].KindIDs[b]
		})
	}
	sort.Slice(s.KindDefinitions, func(i, j int) bool {
		return s.KindDefinitions[i].ID < s.KindDefinitions[j].ID
	})
	sort.Slice(s.ProjectAllowedKinds, func(i, j int) bool {
		return s.ProjectAllowedKinds[i].ProjectID < s.ProjectAllowedKinds[j].ProjectID
	})
	sort.Slice(s.Comments, func(i, j int) bool {
		a := s.Comments[i]
		b := s.Comments[j]
		if a.ProjectID == b.ProjectID {
			if a.TargetType == b.TargetType {
				if a.TargetID == b.TargetID {
					if a.CreatedAt.Equal(b.CreatedAt) {
						return a.ID < b.ID
					}
					return a.CreatedAt.Before(b.CreatedAt)
				}
				return a.TargetID < b.TargetID
			}
			return a.TargetType < b.TargetType
		}
		return a.ProjectID < b.ProjectID
	})
	sort.Slice(s.CapabilityLeases, func(i, j int) bool {
		a := s.CapabilityLeases[i]
		b := s.CapabilityLeases[j]
		if a.ProjectID == b.ProjectID {
			if a.ScopeType == b.ScopeType {
				if a.ScopeID == b.ScopeID {
					if a.IssuedAt.Equal(b.IssuedAt) {
						return a.InstanceID < b.InstanceID
					}
					return a.IssuedAt.Before(b.IssuedAt)
				}
				return a.ScopeID < b.ScopeID
			}
			return a.ScopeType < b.ScopeType
		}
		return a.ProjectID < b.ProjectID
	})
	sort.Slice(s.Handoffs, func(i, j int) bool {
		a := s.Handoffs[i]
		b := s.Handoffs[j]
		if a.ProjectID == b.ProjectID {
			if a.BranchID == b.BranchID {
				if a.ScopeType == b.ScopeType {
					if a.ScopeID == b.ScopeID {
						if a.CreatedAt.Equal(b.CreatedAt) {
							return a.ID < b.ID
						}
						return a.CreatedAt.Before(b.CreatedAt)
					}
					return a.ScopeID < b.ScopeID
				}
				return a.ScopeType < b.ScopeType
			}
			return a.BranchID < b.BranchID
		}
		return a.ProjectID < b.ProjectID
	})
}

// snapshotProjectFromDomain handles snapshot project from domain.
func snapshotProjectFromDomain(p domain.Project) SnapshotProject {
	return SnapshotProject{
		ID:          p.ID,
		Slug:        p.Slug,
		Name:        p.Name,
		Description: p.Description,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt.UTC(),
		UpdatedAt:   p.UpdatedAt.UTC(),
		ArchivedAt:  copyTimePtr(p.ArchivedAt),
	}
}

// snapshotColumnFromDomain handles snapshot column from domain.
func snapshotColumnFromDomain(c domain.Column) SnapshotColumn {
	return SnapshotColumn{
		ID:         c.ID,
		ProjectID:  c.ProjectID,
		Name:       c.Name,
		WIPLimit:   c.WIPLimit,
		Position:   c.Position,
		CreatedAt:  c.CreatedAt.UTC(),
		UpdatedAt:  c.UpdatedAt.UTC(),
		ArchivedAt: copyTimePtr(c.ArchivedAt),
	}
}

// snapshotActionItemFromDomain handles snapshot actionItem from domain.
func snapshotActionItemFromDomain(t domain.ActionItem) SnapshotActionItem {
	return SnapshotActionItem{
		ID:             t.ID,
		ProjectID:      t.ProjectID,
		ParentID:       t.ParentID,
		Kind:           t.Kind,
		Scope:          t.Scope,
		Role:           t.Role,
		StructuralType: t.StructuralType,
		Irreducible:    t.Irreducible,
		Owner:          t.Owner,
		DropNumber:     t.DropNumber,
		Persistent:     t.Persistent,
		DevGated:       t.DevGated,
		Paths:          append([]string(nil), t.Paths...),
		Packages:       append([]string(nil), t.Packages...),
		Files:          append([]string(nil), t.Files...),
		StartCommit:    t.StartCommit,
		LifecycleState: t.LifecycleState,
		ColumnID:       t.ColumnID,
		Position:       t.Position,
		Title:          t.Title,
		Description:    t.Description,
		Priority:       t.Priority,
		DueAt:          copyTimePtr(t.DueAt),
		Labels:         append([]string(nil), t.Labels...),
		Metadata:       t.Metadata,
		CreatedByActor: t.CreatedByActor,
		CreatedByName:  t.CreatedByName,
		UpdatedByActor: t.UpdatedByActor,
		UpdatedByName:  t.UpdatedByName,
		UpdatedByType:  t.UpdatedByType,
		CreatedAt:      t.CreatedAt.UTC(),
		UpdatedAt:      t.UpdatedAt.UTC(),
		StartedAt:      copyTimePtr(t.StartedAt),
		CompletedAt:    copyTimePtr(t.CompletedAt),
		ArchivedAt:     copyTimePtr(t.ArchivedAt),
		CanceledAt:     copyTimePtr(t.CanceledAt),
	}
}

// snapshotKindDefinitionFromDomain converts one kind definition to snapshot payload form.
func snapshotKindDefinitionFromDomain(kind domain.KindDefinition) SnapshotKindDefinition {
	return SnapshotKindDefinition{
		ID:                  kind.ID,
		DisplayName:         kind.DisplayName,
		DescriptionMarkdown: kind.DescriptionMarkdown,
		AppliesTo:           append([]domain.KindAppliesTo(nil), kind.AppliesTo...),
		PayloadSchemaJSON:   kind.PayloadSchemaJSON,
		CreatedAt:           kind.CreatedAt.UTC(),
		UpdatedAt:           kind.UpdatedAt.UTC(),
		ArchivedAt:          copyTimePtr(kind.ArchivedAt),
	}
}

// snapshotCommentFromDomain converts one comment row to snapshot payload form.
func snapshotCommentFromDomain(comment domain.Comment) SnapshotComment {
	return SnapshotComment{
		ID:           comment.ID,
		ProjectID:    comment.ProjectID,
		TargetType:   comment.TargetType,
		TargetID:     comment.TargetID,
		Summary:      comment.Summary,
		BodyMarkdown: comment.BodyMarkdown,
		ActorID:      comment.ActorID,
		ActorName:    comment.ActorName,
		ActorType:    comment.ActorType,
		CreatedAt:    comment.CreatedAt.UTC(),
		UpdatedAt:    comment.UpdatedAt.UTC(),
	}
}

// snapshotCapabilityLeaseFromDomain converts one capability lease to snapshot payload form.
func snapshotCapabilityLeaseFromDomain(lease domain.CapabilityLease) SnapshotCapabilityLease {
	return SnapshotCapabilityLease{
		InstanceID:                lease.InstanceID,
		LeaseToken:                lease.LeaseToken,
		AgentName:                 lease.AgentName,
		ProjectID:                 lease.ProjectID,
		ScopeType:                 lease.ScopeType,
		ScopeID:                   lease.ScopeID,
		Role:                      lease.Role,
		ParentInstanceID:          lease.ParentInstanceID,
		AllowEqualScopeDelegation: lease.AllowEqualScopeDelegation,
		IssuedAt:                  lease.IssuedAt.UTC(),
		ExpiresAt:                 lease.ExpiresAt.UTC(),
		HeartbeatAt:               lease.HeartbeatAt.UTC(),
		RevokedAt:                 copyTimePtr(lease.RevokedAt),
		RevokedReason:             lease.RevokedReason,
	}
}

// snapshotHandoffFromDomain converts one handoff to snapshot payload form.
func snapshotHandoffFromDomain(handoff domain.Handoff) SnapshotHandoff {
	return SnapshotHandoff{
		ID:              handoff.ID,
		ProjectID:       handoff.ProjectID,
		BranchID:        handoff.BranchID,
		ScopeType:       handoff.ScopeType,
		ScopeID:         handoff.ScopeID,
		SourceRole:      handoff.SourceRole,
		TargetBranchID:  handoff.TargetBranchID,
		TargetScopeType: handoff.TargetScopeType,
		TargetScopeID:   handoff.TargetScopeID,
		TargetRole:      handoff.TargetRole,
		Status:          handoff.Status,
		Summary:         handoff.Summary,
		NextAction:      handoff.NextAction,
		MissingEvidence: append([]string(nil), handoff.MissingEvidence...),
		RelatedRefs:     append([]string(nil), handoff.RelatedRefs...),
		CreatedByActor:  handoff.CreatedByActor,
		CreatedByType:   handoff.CreatedByType,
		CreatedAt:       handoff.CreatedAt.UTC(),
		UpdatedByActor:  handoff.UpdatedByActor,
		UpdatedByType:   handoff.UpdatedByType,
		UpdatedAt:       handoff.UpdatedAt.UTC(),
		ResolvedByActor: handoff.ResolvedByActor,
		ResolvedByType:  handoff.ResolvedByType,
		ResolvedAt:      copyTimePtr(handoff.ResolvedAt),
		ResolutionNote:  handoff.ResolutionNote,
	}
}

// normalizeHandoffSnapshotTarget validates one optional handoff target tuple.
func normalizeHandoffSnapshotTarget(projectID string, handoff SnapshotHandoff) (domain.LevelTuple, error) {
	if strings.TrimSpace(handoff.TargetBranchID) == "" && strings.TrimSpace(string(handoff.TargetScopeType)) == "" && strings.TrimSpace(handoff.TargetScopeID) == "" {
		return domain.LevelTuple{}, nil
	}
	return domain.NewLevelTuple(domain.LevelTupleInput{
		ProjectID: projectID,
		BranchID:  handoff.TargetBranchID,
		ScopeType: handoff.TargetScopeType,
		ScopeID:   handoff.TargetScopeID,
	})
}

// snapshotAvailableHandoffScopes builds the set of valid source/target scopes present in one snapshot payload.
func snapshotAvailableHandoffScopes(projects []SnapshotProject, tasks []SnapshotActionItem) map[string]struct{} {
	out := make(map[string]struct{}, len(projects)+len(tasks))
	for _, project := range projects {
		projectID := strings.TrimSpace(project.ID)
		if projectID == "" {
			continue
		}
		out[snapshotHandoffScopeKey(projectID, domain.ScopeLevelProject, projectID)] = struct{}{}
	}
	for _, actionItem := range tasks {
		projectID := strings.TrimSpace(actionItem.ProjectID)
		scopeID := strings.TrimSpace(actionItem.ID)
		if projectID == "" || scopeID == "" {
			continue
		}
		// Scope mirrors kind in the 12-value enum, so every action-item
		// handoff scope lives at ScopeLevelActionItem.
		out[snapshotHandoffScopeKey(projectID, domain.ScopeLevelActionItem, scopeID)] = struct{}{}
	}
	return out
}

// snapshotAvailableDomainHandoffScopes builds the set of valid source/target scopes for export.
func snapshotAvailableDomainHandoffScopes(projectID string, tasks []domain.ActionItem) map[string]struct{} {
	projectID = strings.TrimSpace(projectID)
	out := map[string]struct{}{
		snapshotHandoffScopeKey(projectID, domain.ScopeLevelProject, projectID): {},
	}
	for _, actionItem := range tasks {
		// Scope mirrors kind in the 12-value enum, so every action-item
		// handoff scope lives at ScopeLevelActionItem.
		out[snapshotHandoffScopeKey(projectID, domain.ScopeLevelActionItem, actionItem.ID)] = struct{}{}
	}
	return out
}

// snapshotHandoffScopeKey returns a stable key for one handoff scope reference.
func snapshotHandoffScopeKey(projectID string, scopeType domain.ScopeLevel, scopeID string) string {
	return strings.Join([]string{
		strings.TrimSpace(projectID),
		string(domain.NormalizeScopeLevel(scopeType)),
		strings.TrimSpace(scopeID),
	}, "|")
}

// toDomain converts domain.
func (p SnapshotProject) toDomain() domain.Project {
	slug := strings.TrimSpace(p.Slug)
	if slug == "" {
		slug = fallbackSlug(p.Name)
	}
	return domain.Project{
		ID:          strings.TrimSpace(p.ID),
		Slug:        slug,
		Name:        strings.TrimSpace(p.Name),
		Description: strings.TrimSpace(p.Description),
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt.UTC(),
		UpdatedAt:   p.UpdatedAt.UTC(),
		ArchivedAt:  copyTimePtr(p.ArchivedAt),
	}
}

// toDomain converts domain.
func (c SnapshotColumn) toDomain() domain.Column {
	return domain.Column{
		ID:         strings.TrimSpace(c.ID),
		ProjectID:  strings.TrimSpace(c.ProjectID),
		Name:       strings.TrimSpace(c.Name),
		WIPLimit:   c.WIPLimit,
		Position:   c.Position,
		CreatedAt:  c.CreatedAt.UTC(),
		UpdatedAt:  c.UpdatedAt.UTC(),
		ArchivedAt: copyTimePtr(c.ArchivedAt),
	}
}

// toDomain converts domain.
func (t SnapshotActionItem) toDomain() domain.ActionItem {
	labels := append([]string(nil), t.Labels...)
	state := t.LifecycleState
	if state == "" {
		state = domain.StateTodo
	}
	kind := t.Kind
	if kind == "" {
		// Legacy rows with empty kind fall back to KindPlan so imports remain
		// forward-compatible with the strict 12-value Kind enum.
		kind = domain.KindPlan
	}
	scope := domain.NormalizeKindAppliesTo(t.Scope)
	if scope == "" {
		scope = domain.DefaultActionItemScope(kind)
	}
	updatedType := t.UpdatedByType
	if updatedType == "" {
		updatedType = domain.ActorTypeUser
	}
	createdBy := strings.TrimSpace(t.CreatedByActor)
	if createdBy == "" {
		createdBy = "tillsyn-user"
	}
	createdByName := strings.TrimSpace(t.CreatedByName)
	if createdByName == "" {
		createdByName = createdBy
	}
	updatedBy := strings.TrimSpace(t.UpdatedByActor)
	if updatedBy == "" {
		updatedBy = createdBy
	}
	updatedByName := strings.TrimSpace(t.UpdatedByName)
	if updatedByName == "" {
		if updatedBy == createdBy {
			updatedByName = createdByName
		}
		if updatedByName == "" {
			updatedByName = updatedBy
		}
	}
	return domain.ActionItem{
		ID:        strings.TrimSpace(t.ID),
		ProjectID: strings.TrimSpace(t.ProjectID),
		ParentID:  strings.TrimSpace(t.ParentID),
		Kind:      kind,
		Scope:     scope,
		Role:      t.Role,
		// StructuralType intentionally has no empty-string fallback (unlike
		// Kind above): legacy snapshots without a structural_type field
		// surface as "" so the next mutation through NewActionItem catches it
		// with ErrInvalidStructuralType per droplet 3.2's required-field
		// contract. Inventing a default here would mask schema drift.
		StructuralType: t.StructuralType,
		Irreducible:    t.Irreducible,
		// Owner / DropNumber / Persistent / DevGated have no fallback — same
		// rationale as StructuralType above. Pre-droplet-3.21 snapshots
		// without these fields deserialize to zero values (empty string / 0
		// / false), which are the legitimate domain defaults; legacy-format
		// compatibility is covered by the json:",omitempty" tags. No
		// SnapshotVersion bump required.
		Owner:      t.Owner,
		DropNumber: t.DropNumber,
		Persistent: t.Persistent,
		DevGated:   t.DevGated,
		// Paths has no fallback — same rationale as the four primitives
		// above. Pre-droplet-4a.5 snapshots without this field deserialize
		// to nil, which is the legitimate domain zero value. Legacy-format
		// compatibility is covered by the json:"paths,omitempty" tag.
		Paths: append([]string(nil), t.Paths...),
		// Packages has no fallback — same rationale as Paths above.
		// Pre-droplet-4a.6 snapshots without this field deserialize to nil,
		// which is the legitimate domain zero value. Legacy-format
		// compatibility is covered by the json:"packages,omitempty" tag.
		Packages: append([]string(nil), t.Packages...),
		// Files has no fallback — same rationale as Paths/Packages above.
		// Pre-droplet-4a.7 snapshots without this field deserialize to nil,
		// which is the legitimate domain zero value. Legacy-format
		// compatibility is covered by the json:"files,omitempty" tag.
		Files: append([]string(nil), t.Files...),
		// StartCommit has no fallback — same rationale as Paths/Packages/
		// Files above. Pre-droplet-4a.8 snapshots without this field
		// deserialize to "", which is the legitimate domain zero value
		// ("not yet captured"). Legacy-format compatibility is covered by
		// the json:"start_commit,omitempty" tag.
		StartCommit:    t.StartCommit,
		LifecycleState: state,
		ColumnID:       strings.TrimSpace(t.ColumnID),
		Position:       t.Position,
		Title:          strings.TrimSpace(t.Title),
		Description:    strings.TrimSpace(t.Description),
		Priority:       t.Priority,
		DueAt:          copyTimePtr(t.DueAt),
		Labels:         labels,
		Metadata:       t.Metadata,
		CreatedByActor: createdBy,
		CreatedByName:  createdByName,
		UpdatedByActor: updatedBy,
		UpdatedByName:  updatedByName,
		UpdatedByType:  updatedType,
		CreatedAt:      t.CreatedAt.UTC(),
		UpdatedAt:      t.UpdatedAt.UTC(),
		StartedAt:      copyTimePtr(t.StartedAt),
		CompletedAt:    copyTimePtr(t.CompletedAt),
		ArchivedAt:     copyTimePtr(t.ArchivedAt),
		CanceledAt:     copyTimePtr(t.CanceledAt),
	}
}

// toDomain converts one snapshot kind definition to domain form.
func (k SnapshotKindDefinition) toDomain() domain.KindDefinition {
	return domain.KindDefinition{
		ID:                  domain.NormalizeKindID(k.ID),
		DisplayName:         strings.TrimSpace(k.DisplayName),
		DescriptionMarkdown: strings.TrimSpace(k.DescriptionMarkdown),
		AppliesTo:           append([]domain.KindAppliesTo(nil), k.AppliesTo...),
		PayloadSchemaJSON:   strings.TrimSpace(k.PayloadSchemaJSON),
		CreatedAt:           k.CreatedAt.UTC(),
		UpdatedAt:           k.UpdatedAt.UTC(),
		ArchivedAt:          copyTimePtr(k.ArchivedAt),
	}
}

// toDomain converts one snapshot comment row to domain form.
func (c SnapshotComment) toDomain() domain.Comment {
	actorType := domain.ActorType(strings.TrimSpace(strings.ToLower(string(c.ActorType))))
	if actorType == "" {
		actorType = domain.ActorTypeUser
	}
	body := strings.TrimSpace(c.BodyMarkdown)
	summary := domain.NormalizeCommentSummary(c.Summary, body)
	actorID := strings.TrimSpace(c.ActorID)
	if actorID == "" {
		actorID = "tillsyn-user"
	}
	actorName := strings.TrimSpace(c.ActorName)
	if actorName == "" {
		actorName = actorID
	}
	return domain.Comment{
		ID:           strings.TrimSpace(c.ID),
		ProjectID:    strings.TrimSpace(c.ProjectID),
		TargetType:   domain.NormalizeCommentTargetType(c.TargetType),
		TargetID:     strings.TrimSpace(c.TargetID),
		Summary:      summary,
		BodyMarkdown: body,
		ActorID:      actorID,
		ActorName:    actorName,
		ActorType:    actorType,
		CreatedAt:    c.CreatedAt.UTC(),
		UpdatedAt:    c.UpdatedAt.UTC(),
	}
}

// toDomain converts one snapshot capability lease row to domain form.
func (l SnapshotCapabilityLease) toDomain() domain.CapabilityLease {
	return domain.CapabilityLease{
		InstanceID:                strings.TrimSpace(l.InstanceID),
		LeaseToken:                strings.TrimSpace(l.LeaseToken),
		AgentName:                 strings.TrimSpace(l.AgentName),
		ProjectID:                 strings.TrimSpace(l.ProjectID),
		ScopeType:                 domain.NormalizeCapabilityScopeType(l.ScopeType),
		ScopeID:                   strings.TrimSpace(l.ScopeID),
		Role:                      domain.NormalizeCapabilityRole(l.Role),
		ParentInstanceID:          strings.TrimSpace(l.ParentInstanceID),
		AllowEqualScopeDelegation: l.AllowEqualScopeDelegation,
		IssuedAt:                  l.IssuedAt.UTC(),
		ExpiresAt:                 l.ExpiresAt.UTC(),
		HeartbeatAt:               l.HeartbeatAt.UTC(),
		RevokedAt:                 copyTimePtr(l.RevokedAt),
		RevokedReason:             strings.TrimSpace(l.RevokedReason),
	}
}

// toDomain converts one snapshot handoff row to domain form.
func (h SnapshotHandoff) toDomain() domain.Handoff {
	return domain.Handoff{
		ID:              strings.TrimSpace(h.ID),
		ProjectID:       strings.TrimSpace(h.ProjectID),
		BranchID:        strings.TrimSpace(h.BranchID),
		ScopeType:       domain.NormalizeScopeLevel(h.ScopeType),
		ScopeID:         strings.TrimSpace(h.ScopeID),
		SourceRole:      strings.TrimSpace(h.SourceRole),
		TargetBranchID:  strings.TrimSpace(h.TargetBranchID),
		TargetScopeType: domain.NormalizeScopeLevel(h.TargetScopeType),
		TargetScopeID:   strings.TrimSpace(h.TargetScopeID),
		TargetRole:      strings.TrimSpace(h.TargetRole),
		Status:          domain.NormalizeHandoffStatus(h.Status),
		Summary:         strings.TrimSpace(h.Summary),
		NextAction:      strings.TrimSpace(h.NextAction),
		MissingEvidence: append([]string(nil), h.MissingEvidence...),
		RelatedRefs:     append([]string(nil), h.RelatedRefs...),
		CreatedByActor:  strings.TrimSpace(h.CreatedByActor),
		CreatedByType:   normalizeActorTypeInput(h.CreatedByType),
		CreatedAt:       h.CreatedAt.UTC(),
		UpdatedByActor:  strings.TrimSpace(h.UpdatedByActor),
		UpdatedByType:   normalizeActorTypeInput(h.UpdatedByType),
		UpdatedAt:       h.UpdatedAt.UTC(),
		ResolvedByActor: strings.TrimSpace(h.ResolvedByActor),
		ResolvedByType:  normalizeActorTypeInput(h.ResolvedByType),
		ResolvedAt:      copyTimePtr(h.ResolvedAt),
		ResolutionNote:  strings.TrimSpace(h.ResolutionNote),
	}
}

// fallbackSlug provides fallback slug.
func fallbackSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return strings.Trim(name, "-")
}

// copyTimePtr copies time ptr.
func copyTimePtr(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	t := in.UTC().Truncate(time.Second)
	return &t
}
