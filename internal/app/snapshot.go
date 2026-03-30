package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// SnapshotVersion defines the canonical snapshot schema version.
const SnapshotVersion = "tillsyn.snapshot.v3"

// Snapshot represents snapshot data used by this package.
type Snapshot struct {
	Version             string                          `json:"version"`
	ExportedAt          time.Time                       `json:"exported_at"`
	Projects            []SnapshotProject               `json:"projects"`
	Columns             []SnapshotColumn                `json:"columns"`
	Tasks               []SnapshotTask                  `json:"tasks"`
	KindDefinitions     []SnapshotKindDefinition        `json:"kind_definitions,omitempty"`
	ProjectAllowedKinds []SnapshotProjectAllowedKinds   `json:"project_allowed_kinds,omitempty"`
	TemplateLibraries   []domain.TemplateLibrary        `json:"template_libraries,omitempty"`
	ProjectBindings     []domain.ProjectTemplateBinding `json:"project_template_bindings,omitempty"`
	NodeContracts       []domain.NodeContractSnapshot   `json:"node_contract_snapshots,omitempty"`
	Comments            []SnapshotComment               `json:"comments,omitempty"`
	CapabilityLeases    []SnapshotCapabilityLease       `json:"capability_leases,omitempty"`
	Handoffs            []SnapshotHandoff               `json:"handoffs,omitempty"`
}

// SnapshotProject represents snapshot project data used by this package.
type SnapshotProject struct {
	ID          string                 `json:"id"`
	Slug        string                 `json:"slug"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Kind        domain.KindID          `json:"kind,omitempty"`
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

// SnapshotTask represents snapshot task data used by this package.
type SnapshotTask struct {
	ID             string                `json:"id"`
	ProjectID      string                `json:"project_id"`
	ParentID       string                `json:"parent_id,omitempty"`
	Kind           domain.WorkKind       `json:"kind"`
	Scope          domain.KindAppliesTo  `json:"scope,omitempty"`
	LifecycleState domain.LifecycleState `json:"lifecycle_state"`
	ColumnID       string                `json:"column_id"`
	Position       int                   `json:"position"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Priority       domain.Priority       `json:"priority"`
	DueAt          *time.Time            `json:"due_at,omitempty"`
	Labels         []string              `json:"labels"`
	Metadata       domain.TaskMetadata   `json:"metadata"`
	CreatedByActor string                `json:"created_by_actor"`
	CreatedByName  string                `json:"created_by_name,omitempty"`
	UpdatedByActor string                `json:"updated_by_actor"`
	UpdatedByName  string                `json:"updated_by_name,omitempty"`
	UpdatedByType  domain.ActorType      `json:"updated_by_type"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	StartedAt      *time.Time            `json:"started_at,omitempty"`
	CompletedAt    *time.Time            `json:"completed_at,omitempty"`
	ArchivedAt     *time.Time            `json:"archived_at,omitempty"`
	CanceledAt     *time.Time            `json:"canceled_at,omitempty"`
}

// SnapshotKindDefinition represents one kind-catalog definition persisted in a snapshot.
type SnapshotKindDefinition struct {
	ID                  domain.KindID          `json:"id"`
	DisplayName         string                 `json:"display_name"`
	DescriptionMarkdown string                 `json:"description_markdown"`
	AppliesTo           []domain.KindAppliesTo `json:"applies_to"`
	AllowedParentScopes []domain.KindAppliesTo `json:"allowed_parent_scopes,omitempty"`
	PayloadSchemaJSON   string                 `json:"payload_schema_json,omitempty"`
	Template            domain.KindTemplate    `json:"template,omitempty"`
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
	templateLibraries, err := s.repo.ListTemplateLibraries(ctx, domain.TemplateLibraryFilter{})
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
		Tasks:               make([]SnapshotTask, 0),
		KindDefinitions:     make([]SnapshotKindDefinition, 0, len(kindDefinitions)),
		ProjectAllowedKinds: make([]SnapshotProjectAllowedKinds, 0, len(projects)),
		TemplateLibraries:   make([]domain.TemplateLibrary, 0, len(templateLibraries)),
		ProjectBindings:     make([]domain.ProjectTemplateBinding, 0, len(projects)),
		NodeContracts:       make([]domain.NodeContractSnapshot, 0),
		Comments:            make([]SnapshotComment, 0),
		CapabilityLeases:    make([]SnapshotCapabilityLease, 0),
		Handoffs:            make([]SnapshotHandoff, 0),
	}
	for _, kind := range kindDefinitions {
		snap.KindDefinitions = append(snap.KindDefinitions, snapshotKindDefinitionFromDomain(kind))
	}
	for _, library := range templateLibraries {
		snap.TemplateLibraries = append(snap.TemplateLibraries, snapshotTemplateLibraryFromDomain(library))
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
		binding, getErr := s.repo.GetProjectTemplateBinding(ctx, project.ID)
		switch {
		case getErr == nil:
			snap.ProjectBindings = append(snap.ProjectBindings, snapshotProjectTemplateBindingFromDomain(binding))
		case errors.Is(getErr, ErrNotFound):
		default:
			return Snapshot{}, getErr
		}

		columns, listErr := s.repo.ListColumns(ctx, project.ID, includeArchived)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		for _, column := range columns {
			snap.Columns = append(snap.Columns, snapshotColumnFromDomain(column))
		}

		tasks, listErr := s.repo.ListTasks(ctx, project.ID, includeArchived)
		if listErr != nil {
			return Snapshot{}, listErr
		}
		for _, task := range tasks {
			snap.Tasks = append(snap.Tasks, snapshotTaskFromDomain(task))
			nodeContract, getErr := s.repo.GetNodeContractSnapshot(ctx, task.ID)
			switch {
			case getErr == nil:
				snap.NodeContracts = append(snap.NodeContracts, snapshotNodeContractFromDomain(nodeContract))
			case errors.Is(getErr, ErrNotFound):
			default:
				return Snapshot{}, getErr
			}
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
	for _, library := range snap.TemplateLibraries {
		if err := s.upsertTemplateLibrary(ctx, library); err != nil {
			return err
		}
	}
	for _, allow := range snap.ProjectAllowedKinds {
		if err := s.repo.SetProjectAllowedKinds(ctx, strings.TrimSpace(allow.ProjectID), append([]domain.KindID(nil), allow.KindIDs...)); err != nil {
			return err
		}
	}
	for _, binding := range snap.ProjectBindings {
		if err := s.repo.UpsertProjectTemplateBinding(ctx, binding); err != nil {
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

	for _, task := range snap.Tasks {
		dt := task.toDomain()
		if _, err := s.repo.GetTask(ctx, dt.ID); err == nil {
			if err := s.repo.UpdateTask(ctx, dt); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := s.repo.CreateTask(ctx, dt); err != nil {
			return err
		}
	}
	for _, nodeContract := range snap.NodeContracts {
		if _, err := s.repo.GetNodeContractSnapshot(ctx, strings.TrimSpace(nodeContract.NodeID)); err == nil {
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := s.repo.CreateNodeContractSnapshot(ctx, nodeContract); err != nil {
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
		if domain.NormalizeKindID(p.Kind) == "" {
			p.Kind = domain.DefaultProjectKind
			s.Projects[i].Kind = p.Kind
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

	taskIDs := map[string]struct{}{}
	taskByID := map[string]SnapshotTask{}
	for i, t := range s.Tasks {
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
			t.Kind = domain.WorkKindTask
			s.Tasks[i].Kind = t.Kind
		}
		if t.Scope == "" {
			t.Scope = domain.DefaultTaskScope(t.Kind, t.ParentID)
			s.Tasks[i].Scope = t.Scope
		}
		if !domain.IsValidWorkItemAppliesTo(t.Scope) {
			return fmt.Errorf("tasks[%d].scope must be branch|phase|task|subtask", i)
		}
		if t.LifecycleState == "" {
			t.LifecycleState = domain.StateTodo
			s.Tasks[i].LifecycleState = t.LifecycleState
		}
		switch t.LifecycleState {
		case domain.StateTodo, domain.StateProgress, domain.StateDone, domain.StateArchived:
		default:
			return fmt.Errorf("tasks[%d].lifecycle_state must be todo|progress|done|archived", i)
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
		if _, exists := taskIDs[t.ID]; exists {
			return fmt.Errorf("duplicate task id: %q", t.ID)
		}
		taskIDs[t.ID] = struct{}{}
		taskByID[t.ID] = s.Tasks[i]
	}
	for i, t := range s.Tasks {
		if strings.TrimSpace(t.ParentID) == "" {
			continue
		}
		if t.ParentID == t.ID {
			return fmt.Errorf("tasks[%d].parent_id cannot reference itself", i)
		}
		if _, exists := taskIDs[t.ParentID]; !exists {
			return fmt.Errorf("tasks[%d] references unknown parent_id %q", i, t.ParentID)
		}
		parent := taskByID[t.ParentID]
		if t.Kind == domain.WorkKindPhase && parent.Scope != domain.KindAppliesToBranch && parent.Scope != domain.KindAppliesToPhase {
			return fmt.Errorf("tasks[%d].parent_id %q invalid for phase parent scope %q", i, t.ParentID, parent.Scope)
		}
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
	libraryIDs := map[string]struct{}{}
	for i, library := range s.TemplateLibraries {
		normalized, err := normalizeSnapshotTemplateLibrary(library)
		if err != nil {
			return fmt.Errorf("template_libraries[%d] invalid: %w", i, err)
		}
		if _, exists := libraryIDs[normalized.ID]; exists {
			return fmt.Errorf("duplicate template library id: %q", normalized.ID)
		}
		if normalized.Scope != domain.TemplateLibraryScopeGlobal {
			if _, ok := projectIDs[normalized.ProjectID]; !ok {
				return fmt.Errorf("template_libraries[%d] references unknown project_id %q", i, normalized.ProjectID)
			}
		}
		for _, nodeTemplate := range normalized.NodeTemplates {
			if _, ok := kindIDs[nodeTemplate.NodeKindID]; !ok {
				return fmt.Errorf("template_libraries[%d] references unknown node_kind_id %q", i, nodeTemplate.NodeKindID)
			}
			for _, childRule := range nodeTemplate.ChildRules {
				if _, ok := kindIDs[childRule.ChildKindID]; !ok {
					return fmt.Errorf("template_libraries[%d] references unknown child_kind_id %q", i, childRule.ChildKindID)
				}
			}
		}
		s.TemplateLibraries[i] = normalized
		libraryIDs[normalized.ID] = struct{}{}
	}
	bindingProjects := map[string]struct{}{}
	for i, binding := range s.ProjectBindings {
		normalized, err := normalizeSnapshotProjectTemplateBinding(binding)
		if err != nil {
			return fmt.Errorf("project_template_bindings[%d] invalid: %w", i, err)
		}
		if _, ok := projectIDs[normalized.ProjectID]; !ok {
			return fmt.Errorf("project_template_bindings[%d] references unknown project_id %q", i, normalized.ProjectID)
		}
		if _, ok := libraryIDs[domain.NormalizeTemplateLibraryID(normalized.LibraryID)]; !ok {
			return fmt.Errorf("project_template_bindings[%d] references unknown library_id %q", i, normalized.LibraryID)
		}
		if _, exists := bindingProjects[normalized.ProjectID]; exists {
			return fmt.Errorf("duplicate project_template_bindings project_id: %q", normalized.ProjectID)
		}
		s.ProjectBindings[i] = normalized
		bindingProjects[normalized.ProjectID] = struct{}{}
	}
	nodeContractIDs := map[string]struct{}{}
	for i, nodeContract := range s.NodeContracts {
		normalized, err := normalizeSnapshotNodeContract(nodeContract)
		if err != nil {
			return fmt.Errorf("node_contract_snapshots[%d] invalid: %w", i, err)
		}
		if _, ok := projectIDs[normalized.ProjectID]; !ok {
			return fmt.Errorf("node_contract_snapshots[%d] references unknown project_id %q", i, normalized.ProjectID)
		}
		if _, ok := taskIDs[normalized.NodeID]; !ok {
			return fmt.Errorf("node_contract_snapshots[%d] references unknown node_id %q", i, normalized.NodeID)
		}
		if _, ok := libraryIDs[domain.NormalizeTemplateLibraryID(normalized.SourceLibraryID)]; !ok {
			return fmt.Errorf("node_contract_snapshots[%d] references unknown source_library_id %q", i, normalized.SourceLibraryID)
		}
		if _, exists := nodeContractIDs[normalized.NodeID]; exists {
			return fmt.Errorf("duplicate node_contract_snapshots node_id: %q", normalized.NodeID)
		}
		s.NodeContracts[i] = normalized
		nodeContractIDs[normalized.NodeID] = struct{}{}
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

	availableHandoffScopes := snapshotAvailableHandoffScopes(s.Projects, s.Tasks)
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

// upsertTemplateLibrary upserts one template-library row and nested rules.
func (s *Service) upsertTemplateLibrary(ctx context.Context, library domain.TemplateLibrary) error {
	return s.repo.UpsertTemplateLibrary(ctx, library)
}

// commentsForProjectSnapshot collects project and task-scoped comments for snapshot export.
func (s *Service) commentsForProjectSnapshot(ctx context.Context, project domain.Project, tasks []domain.Task) ([]SnapshotComment, error) {
	targets := []domain.CommentTarget{{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   project.ID,
	}}
	for _, task := range tasks {
		targets = append(targets, domain.CommentTarget{
			ProjectID:  project.ID,
			TargetType: snapshotCommentTargetTypeForTask(task),
			TargetID:   task.ID,
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
func (s *Service) handoffsForProjectSnapshot(ctx context.Context, projectID string, tasks []domain.Task) ([]SnapshotHandoff, error) {
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

// capabilityLeasesForProjectSnapshot collects project/task hierarchy capability leases for snapshot export.
func (s *Service) capabilityLeasesForProjectSnapshot(ctx context.Context, projectID string, tasks []domain.Task) ([]SnapshotCapabilityLease, error) {
	type scopeQuery struct {
		scopeType domain.CapabilityScopeType
		scopeID   string
	}
	queries := []scopeQuery{{
		scopeType: domain.CapabilityScopeProject,
		scopeID:   "",
	}}
	for _, task := range tasks {
		queries = append(queries, scopeQuery{
			scopeType: snapshotCapabilityScopeTypeForTask(task),
			scopeID:   task.ID,
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

// snapshotCommentTargetTypeForTask maps one work-item row to a comment target type.
func snapshotCommentTargetTypeForTask(task domain.Task) domain.CommentTargetType {
	switch task.Kind {
	case domain.WorkKind(domain.KindAppliesToBranch):
		return domain.CommentTargetTypeBranch
	case domain.WorkKindPhase:
		return domain.CommentTargetTypePhase
	case domain.WorkKindSubtask:
		return domain.CommentTargetTypeSubtask
	case domain.WorkKindDecision:
		return domain.CommentTargetTypeDecision
	case domain.WorkKindNote:
		return domain.CommentTargetTypeNote
	default:
		if task.Scope == domain.KindAppliesToBranch {
			return domain.CommentTargetTypeBranch
		}
		if task.Scope == domain.KindAppliesToSubtask {
			return domain.CommentTargetTypeSubtask
		}
		if task.Scope == domain.KindAppliesToPhase {
			return domain.CommentTargetTypePhase
		}
		return domain.CommentTargetTypeTask
	}
}

// snapshotCapabilityScopeTypeForTask maps one work-item row to a capability scope type.
func snapshotCapabilityScopeTypeForTask(task domain.Task) domain.CapabilityScopeType {
	switch task.Scope {
	case domain.KindAppliesToBranch:
		return domain.CapabilityScopeBranch
	case domain.KindAppliesToPhase:
		return domain.CapabilityScopePhase
	case domain.KindAppliesToSubtask:
		return domain.CapabilityScopeSubtask
	default:
		return domain.CapabilityScopeTask
	}
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
	sort.Slice(s.Tasks, func(i, j int) bool {
		a := s.Tasks[i]
		b := s.Tasks[j]
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
	sort.Slice(s.TemplateLibraries, func(i, j int) bool {
		a := s.TemplateLibraries[i]
		b := s.TemplateLibraries[j]
		if a.Scope == b.Scope {
			if a.ProjectID == b.ProjectID {
				return a.ID < b.ID
			}
			return a.ProjectID < b.ProjectID
		}
		return a.Scope < b.Scope
	})
	sort.Slice(s.ProjectBindings, func(i, j int) bool {
		return s.ProjectBindings[i].ProjectID < s.ProjectBindings[j].ProjectID
	})
	sort.Slice(s.NodeContracts, func(i, j int) bool {
		return s.NodeContracts[i].NodeID < s.NodeContracts[j].NodeID
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

// snapshotTemplateLibraryFromDomain deep-copies one template library into snapshot form.
func snapshotTemplateLibraryFromDomain(library domain.TemplateLibrary) domain.TemplateLibrary {
	out := library
	out.NodeTemplates = make([]domain.NodeTemplate, 0, len(library.NodeTemplates))
	for _, nodeTemplate := range library.NodeTemplates {
		copied := nodeTemplate
		if nodeTemplate.ProjectMetadataDefaults != nil {
			projectMetadata := *nodeTemplate.ProjectMetadataDefaults
			copied.ProjectMetadataDefaults = &projectMetadata
		}
		if nodeTemplate.TaskMetadataDefaults != nil {
			taskMetadata := *nodeTemplate.TaskMetadataDefaults
			copied.TaskMetadataDefaults = &taskMetadata
		}
		copied.ChildRules = append([]domain.TemplateChildRule(nil), nodeTemplate.ChildRules...)
		out.NodeTemplates = append(out.NodeTemplates, copied)
	}
	return out
}

// snapshotProjectTemplateBindingFromDomain copies one project binding into snapshot form.
func snapshotProjectTemplateBindingFromDomain(binding domain.ProjectTemplateBinding) domain.ProjectTemplateBinding {
	return binding
}

// snapshotNodeContractFromDomain copies one node-contract snapshot into snapshot form.
func snapshotNodeContractFromDomain(nodeContract domain.NodeContractSnapshot) domain.NodeContractSnapshot {
	out := nodeContract
	out.EditableByActorKinds = append([]domain.TemplateActorKind(nil), nodeContract.EditableByActorKinds...)
	out.CompletableByActorKinds = append([]domain.TemplateActorKind(nil), nodeContract.CompletableByActorKinds...)
	return out
}

// snapshotProjectFromDomain handles snapshot project from domain.
func snapshotProjectFromDomain(p domain.Project) SnapshotProject {
	return SnapshotProject{
		ID:          p.ID,
		Slug:        p.Slug,
		Name:        p.Name,
		Description: p.Description,
		Kind:        p.Kind,
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

// snapshotTaskFromDomain handles snapshot task from domain.
func snapshotTaskFromDomain(t domain.Task) SnapshotTask {
	return SnapshotTask{
		ID:             t.ID,
		ProjectID:      t.ProjectID,
		ParentID:       t.ParentID,
		Kind:           t.Kind,
		Scope:          t.Scope,
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

// normalizeSnapshotTemplateLibrary validates and normalizes one imported template library.
func normalizeSnapshotTemplateLibrary(library domain.TemplateLibrary) (domain.TemplateLibrary, error) {
	if library.CreatedAt.IsZero() || library.UpdatedAt.IsZero() {
		return domain.TemplateLibrary{}, fmt.Errorf("timestamps are required")
	}
	input := domain.TemplateLibraryInput{
		ID:                  library.ID,
		Scope:               library.Scope,
		ProjectID:           library.ProjectID,
		Name:                library.Name,
		Description:         library.Description,
		Status:              library.Status,
		SourceLibraryID:     library.SourceLibraryID,
		CreatedByActorID:    library.CreatedByActorID,
		CreatedByActorName:  library.CreatedByActorName,
		CreatedByActorType:  library.CreatedByActorType,
		ApprovedByActorID:   library.ApprovedByActorID,
		ApprovedByActorName: library.ApprovedByActorName,
		ApprovedByActorType: library.ApprovedByActorType,
		ApprovedAt:          cloneTimePtr(library.ApprovedAt),
		NodeTemplates:       make([]domain.NodeTemplateInput, 0, len(library.NodeTemplates)),
	}
	for _, nodeTemplate := range library.NodeTemplates {
		templateInput := domain.NodeTemplateInput{
			ID:                  nodeTemplate.ID,
			ScopeLevel:          nodeTemplate.ScopeLevel,
			NodeKindID:          nodeTemplate.NodeKindID,
			DisplayName:         nodeTemplate.DisplayName,
			DescriptionMarkdown: nodeTemplate.DescriptionMarkdown,
			ChildRules:          make([]domain.TemplateChildRuleInput, 0, len(nodeTemplate.ChildRules)),
		}
		if nodeTemplate.ProjectMetadataDefaults != nil {
			projectMetadata := *nodeTemplate.ProjectMetadataDefaults
			templateInput.ProjectMetadataDefaults = &projectMetadata
		}
		if nodeTemplate.TaskMetadataDefaults != nil {
			taskMetadata := *nodeTemplate.TaskMetadataDefaults
			templateInput.TaskMetadataDefaults = &taskMetadata
		}
		for _, childRule := range nodeTemplate.ChildRules {
			templateInput.ChildRules = append(templateInput.ChildRules, domain.TemplateChildRuleInput{
				ID:                        childRule.ID,
				Position:                  childRule.Position,
				ChildScopeLevel:           childRule.ChildScopeLevel,
				ChildKindID:               childRule.ChildKindID,
				TitleTemplate:             childRule.TitleTemplate,
				DescriptionTemplate:       childRule.DescriptionTemplate,
				ResponsibleActorKind:      childRule.ResponsibleActorKind,
				EditableByActorKinds:      append([]domain.TemplateActorKind(nil), childRule.EditableByActorKinds...),
				CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), childRule.CompletableByActorKinds...),
				OrchestratorMayComplete:   childRule.OrchestratorMayComplete,
				RequiredForParentDone:     childRule.RequiredForParentDone,
				RequiredForContainingDone: childRule.RequiredForContainingDone,
			})
		}
		input.NodeTemplates = append(input.NodeTemplates, templateInput)
	}
	normalized, err := domain.NewTemplateLibrary(input, library.CreatedAt)
	if err != nil {
		return domain.TemplateLibrary{}, err
	}
	normalized.CreatedAt = library.CreatedAt.UTC()
	normalized.UpdatedAt = library.UpdatedAt.UTC()
	normalized.ApprovedAt = cloneTimePtr(library.ApprovedAt)
	return normalized, nil
}

// normalizeSnapshotProjectTemplateBinding validates and normalizes one imported project binding.
func normalizeSnapshotProjectTemplateBinding(binding domain.ProjectTemplateBinding) (domain.ProjectTemplateBinding, error) {
	if binding.BoundAt.IsZero() {
		return domain.ProjectTemplateBinding{}, fmt.Errorf("bound_at is required")
	}
	normalized, err := domain.NewProjectTemplateBinding(domain.ProjectTemplateBindingInput{
		ProjectID:        binding.ProjectID,
		LibraryID:        binding.LibraryID,
		BoundByActorID:   binding.BoundByActorID,
		BoundByActorName: binding.BoundByActorName,
		BoundByActorType: binding.BoundByActorType,
	}, binding.BoundAt)
	if err != nil {
		return domain.ProjectTemplateBinding{}, err
	}
	normalized.BoundAt = binding.BoundAt.UTC()
	return normalized, nil
}

// normalizeSnapshotNodeContract validates and normalizes one imported node-contract snapshot.
func normalizeSnapshotNodeContract(nodeContract domain.NodeContractSnapshot) (domain.NodeContractSnapshot, error) {
	if nodeContract.CreatedAt.IsZero() {
		return domain.NodeContractSnapshot{}, fmt.Errorf("created_at is required")
	}
	normalized, err := domain.NewNodeContractSnapshot(domain.NodeContractSnapshotInput{
		NodeID:                    nodeContract.NodeID,
		ProjectID:                 nodeContract.ProjectID,
		SourceLibraryID:           nodeContract.SourceLibraryID,
		SourceNodeTemplateID:      nodeContract.SourceNodeTemplateID,
		SourceChildRuleID:         nodeContract.SourceChildRuleID,
		CreatedByActorID:          nodeContract.CreatedByActorID,
		CreatedByActorType:        nodeContract.CreatedByActorType,
		ResponsibleActorKind:      nodeContract.ResponsibleActorKind,
		EditableByActorKinds:      append([]domain.TemplateActorKind(nil), nodeContract.EditableByActorKinds...),
		CompletableByActorKinds:   append([]domain.TemplateActorKind(nil), nodeContract.CompletableByActorKinds...),
		OrchestratorMayComplete:   nodeContract.OrchestratorMayComplete,
		RequiredForParentDone:     nodeContract.RequiredForParentDone,
		RequiredForContainingDone: nodeContract.RequiredForContainingDone,
	}, nodeContract.CreatedAt)
	if err != nil {
		return domain.NodeContractSnapshot{}, err
	}
	normalized.CreatedAt = nodeContract.CreatedAt.UTC()
	return normalized, nil
}

// cloneTimePtr returns a detached copy of one time pointer.
func cloneTimePtr(ts *time.Time) *time.Time {
	if ts == nil {
		return nil
	}
	cloned := ts.UTC()
	return &cloned
}

// snapshotKindDefinitionFromDomain converts one kind definition to snapshot payload form.
func snapshotKindDefinitionFromDomain(kind domain.KindDefinition) SnapshotKindDefinition {
	return SnapshotKindDefinition{
		ID:                  kind.ID,
		DisplayName:         kind.DisplayName,
		DescriptionMarkdown: kind.DescriptionMarkdown,
		AppliesTo:           append([]domain.KindAppliesTo(nil), kind.AppliesTo...),
		AllowedParentScopes: append([]domain.KindAppliesTo(nil), kind.AllowedParentScopes...),
		PayloadSchemaJSON:   kind.PayloadSchemaJSON,
		Template:            kind.Template,
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
func snapshotAvailableHandoffScopes(projects []SnapshotProject, tasks []SnapshotTask) map[string]struct{} {
	out := make(map[string]struct{}, len(projects)+len(tasks))
	for _, project := range projects {
		projectID := strings.TrimSpace(project.ID)
		if projectID == "" {
			continue
		}
		out[snapshotHandoffScopeKey(projectID, domain.ScopeLevelProject, projectID)] = struct{}{}
	}
	for _, task := range tasks {
		projectID := strings.TrimSpace(task.ProjectID)
		scopeID := strings.TrimSpace(task.ID)
		if projectID == "" || scopeID == "" {
			continue
		}
		scopeType := domain.ScopeLevelFromKindAppliesTo(task.Scope)
		if scopeType == "" {
			scopeType = domain.ScopeLevelTask
		}
		out[snapshotHandoffScopeKey(projectID, scopeType, scopeID)] = struct{}{}
	}
	return out
}

// snapshotAvailableDomainHandoffScopes builds the set of valid source/target scopes for export.
func snapshotAvailableDomainHandoffScopes(projectID string, tasks []domain.Task) map[string]struct{} {
	projectID = strings.TrimSpace(projectID)
	out := map[string]struct{}{
		snapshotHandoffScopeKey(projectID, domain.ScopeLevelProject, projectID): {},
	}
	for _, task := range tasks {
		scopeType := domain.ScopeLevelFromKindAppliesTo(task.Scope)
		if scopeType == "" {
			scopeType = domain.ScopeLevelTask
		}
		out[snapshotHandoffScopeKey(projectID, scopeType, task.ID)] = struct{}{}
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
	kind := domain.NormalizeKindID(p.Kind)
	if kind == "" {
		kind = domain.DefaultProjectKind
	}
	return domain.Project{
		ID:          strings.TrimSpace(p.ID),
		Slug:        slug,
		Name:        strings.TrimSpace(p.Name),
		Description: strings.TrimSpace(p.Description),
		Kind:        kind,
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
func (t SnapshotTask) toDomain() domain.Task {
	labels := append([]string(nil), t.Labels...)
	state := t.LifecycleState
	if state == "" {
		state = domain.StateTodo
	}
	kind := t.Kind
	if kind == "" {
		kind = domain.WorkKindTask
	}
	scope := domain.NormalizeKindAppliesTo(t.Scope)
	if scope == "" {
		scope = domain.DefaultTaskScope(kind, t.ParentID)
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
	return domain.Task{
		ID:             strings.TrimSpace(t.ID),
		ProjectID:      strings.TrimSpace(t.ProjectID),
		ParentID:       strings.TrimSpace(t.ParentID),
		Kind:           kind,
		Scope:          scope,
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
		AllowedParentScopes: append([]domain.KindAppliesTo(nil), k.AllowedParentScopes...),
		PayloadSchemaJSON:   strings.TrimSpace(k.PayloadSchemaJSON),
		Template:            k.Template,
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
