package client

import (
	"encoding/json"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// ActionItemDetailDTO is the rich, serialization-clean DTO for a fully-detailed action item.
// All fields are exported with explicit JSON tags. Timestamps are RFC3339 strings.
// This DTO serves as the daemon wire contract (Track B — D-WIRE-CONTRACT).
type ActionItemDetailDTO struct {
	ID             string                `json:"id"`
	ProjectID      string                `json:"project_id"`
	ParentID       string                `json:"parent_id"`
	Kind           string                `json:"kind"`
	Scope          string                `json:"scope"`
	Role           string                `json:"role"`
	StructuralType string                `json:"structural_type"`
	Irreducible    bool                  `json:"irreducible"`
	Owner          string                `json:"owner"`
	DropNumber     int                   `json:"drop_number"`
	Persistent     bool                  `json:"persistent"`
	DevGated       bool                  `json:"dev_gated"`
	Paths          []string              `json:"paths"`
	Packages       []string              `json:"packages"`
	Files          []string              `json:"files"`
	StartCommit    string                `json:"start_commit"`
	EndCommit      string                `json:"end_commit"`
	LifecycleState string                `json:"lifecycle_state"`
	ColumnID       string                `json:"column_id"`
	Position       int                   `json:"position"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Priority       string                `json:"priority"`
	DueAt          *string               `json:"due_at,omitempty"`
	Labels         []string              `json:"labels"`
	Metadata       ActionItemMetadataDTO `json:"metadata"`
	CreatedByActor string                `json:"created_by_actor"`
	CreatedByName  string                `json:"created_by_name"`
	UpdatedByActor string                `json:"updated_by_actor"`
	UpdatedByName  string                `json:"updated_by_name"`
	UpdatedByType  string                `json:"updated_by_type"`
	CreatedAt      string                `json:"created_at"`
	UpdatedAt      string                `json:"updated_at"`
	StartedAt      *string               `json:"started_at,omitempty"`
	CompletedAt    *string               `json:"completed_at,omitempty"`
	ArchivedAt     *string               `json:"archived_at,omitempty"`
	CanceledAt     *string               `json:"canceled_at,omitempty"`
}

// ActionItemMetadataDTO represents the rich metadata for an action item in wire form.
type ActionItemMetadataDTO struct {
	Objective                string          `json:"objective"`
	ImplementationNotesUser  string          `json:"implementation_notes_user"`
	ImplementationNotesAgent string          `json:"implementation_notes_agent"`
	AcceptanceCriteria       string          `json:"acceptance_criteria"`
	DefinitionOfDone         string          `json:"definition_of_done"`
	ValidationPlan           string          `json:"validation_plan"`
	BlockedReason            string          `json:"blocked_reason"`
	RiskNotes                string          `json:"risk_notes"`
	CommandSnippets          []string        `json:"command_snippets"`
	ExpectedOutputs          []string        `json:"expected_outputs"`
	DecisionLog              []string        `json:"decision_log"`
	RelatedItems             []string        `json:"related_items"`
	TransitionNotes          string          `json:"transition_notes"`
	DependsOn                []string        `json:"depends_on"`
	BlockedBy                []string        `json:"blocked_by"`
	KindPayload              json.RawMessage `json:"kind_payload,omitempty"`
	Outcome                  string          `json:"outcome,omitempty"`
	SpawnBundlePath          string          `json:"spawn_bundle_path,omitempty"`
	ActualCostUSD            *float64        `json:"actual_cost_usd,omitempty"`
}

// CommentDTO is the serialization-clean DTO for a comment.
type CommentDTO struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	TargetType   string `json:"target_type"`
	TargetID     string `json:"target_id"`
	Summary      string `json:"summary"`
	BodyMarkdown string `json:"body_markdown"`
	ActorID      string `json:"actor_id"`
	ActorName    string `json:"actor_name"`
	ActorType    string `json:"actor_type"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// DependencyRollupDTO summarizes dependency and blocked-state counts for a project.
type DependencyRollupDTO struct {
	ProjectID                 string `json:"project_id"`
	TotalItems                int    `json:"total_items"`
	ItemsWithDependencies     int    `json:"items_with_dependencies"`
	DependencyEdges           int    `json:"dependency_edges"`
	BlockedItems              int    `json:"blocked_items"`
	BlockedByEdges            int    `json:"blocked_by_edges"`
	UnresolvedDependencyEdges int    `json:"unresolved_dependency_edges"`
}

// ActionItemDetailFromDomain converts domain.ActionItem to wire-friendly ActionItemDetailDTO.
func ActionItemDetailFromDomain(ai domain.ActionItem) ActionItemDetailDTO {
	return ActionItemDetailDTO{
		ID:             ai.ID,
		ProjectID:      ai.ProjectID,
		ParentID:       ai.ParentID,
		Kind:           string(ai.Kind),
		Scope:          string(ai.Scope),
		Role:           string(ai.Role),
		StructuralType: string(ai.StructuralType),
		Irreducible:    ai.Irreducible,
		Owner:          ai.Owner,
		DropNumber:     ai.DropNumber,
		Persistent:     ai.Persistent,
		DevGated:       ai.DevGated,
		Paths:          ai.Paths,
		Packages:       ai.Packages,
		Files:          ai.Files,
		StartCommit:    ai.StartCommit,
		EndCommit:      ai.EndCommit,
		LifecycleState: string(ai.LifecycleState),
		ColumnID:       ai.ColumnID,
		Position:       ai.Position,
		Title:          ai.Title,
		Description:    ai.Description,
		Priority:       string(ai.Priority),
		DueAt:          timeToPtr(ai.DueAt),
		Labels:         ai.Labels,
		Metadata:       actionItemMetadataFromDomain(ai.Metadata),
		CreatedByActor: ai.CreatedByActor,
		CreatedByName:  ai.CreatedByName,
		UpdatedByActor: ai.UpdatedByActor,
		UpdatedByName:  ai.UpdatedByName,
		UpdatedByType:  string(ai.UpdatedByType),
		CreatedAt:      ai.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      ai.UpdatedAt.Format(time.RFC3339),
		StartedAt:      timeToPtr(ai.StartedAt),
		CompletedAt:    timeToPtr(ai.CompletedAt),
		ArchivedAt:     timeToPtr(ai.ArchivedAt),
		CanceledAt:     timeToPtr(ai.CanceledAt),
	}
}

// actionItemMetadataFromDomain converts domain.ActionItemMetadata to wire form.
func actionItemMetadataFromDomain(m domain.ActionItemMetadata) ActionItemMetadataDTO {
	return ActionItemMetadataDTO{
		Objective:                m.Objective,
		ImplementationNotesUser:  m.ImplementationNotesUser,
		ImplementationNotesAgent: m.ImplementationNotesAgent,
		AcceptanceCriteria:       m.AcceptanceCriteria,
		DefinitionOfDone:         m.DefinitionOfDone,
		ValidationPlan:           m.ValidationPlan,
		BlockedReason:            m.BlockedReason,
		RiskNotes:                m.RiskNotes,
		CommandSnippets:          m.CommandSnippets,
		ExpectedOutputs:          m.ExpectedOutputs,
		DecisionLog:              m.DecisionLog,
		RelatedItems:             m.RelatedItems,
		TransitionNotes:          m.TransitionNotes,
		DependsOn:                m.DependsOn,
		BlockedBy:                m.BlockedBy,
		KindPayload:              m.KindPayload,
		Outcome:                  m.Outcome,
		SpawnBundlePath:          m.SpawnBundlePath,
		ActualCostUSD:            m.ActualCostUSD,
	}
}

// CommentFromDomain converts domain.Comment to wire-friendly CommentDTO.
func CommentFromDomain(c domain.Comment) CommentDTO {
	return CommentDTO{
		ID:           c.ID,
		ProjectID:    c.ProjectID,
		TargetType:   string(c.TargetType),
		TargetID:     c.TargetID,
		Summary:      c.Summary,
		BodyMarkdown: c.BodyMarkdown,
		ActorID:      c.ActorID,
		ActorName:    c.ActorName,
		ActorType:    string(c.ActorType),
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}
}

// DependencyRollupFromDomain converts domain.DependencyRollup to wire-friendly DependencyRollupDTO.
func DependencyRollupFromDomain(rollup domain.DependencyRollup) DependencyRollupDTO {
	return DependencyRollupDTO{
		ProjectID:                 rollup.ProjectID,
		TotalItems:                rollup.TotalItems,
		ItemsWithDependencies:     rollup.ItemsWithDependencies,
		DependencyEdges:           rollup.DependencyEdges,
		BlockedItems:              rollup.BlockedItems,
		BlockedByEdges:            rollup.BlockedByEdges,
		UnresolvedDependencyEdges: rollup.UnresolvedDependencyEdges,
	}
}

// CreateActionItemInputDTO holds input values for creating an action item.
type CreateActionItemInputDTO struct {
	ProjectID      string   `json:"project_id"`
	ParentID       string   `json:"parent_id,omitempty"`
	Kind           string   `json:"kind"`
	StructuralType string   `json:"structural_type"`
	Owner          string   `json:"owner,omitempty"`
	DropNumber     int      `json:"drop_number,omitempty"`
	Persistent     bool     `json:"persistent,omitempty"`
	DevGated       bool     `json:"dev_gated,omitempty"`
	Paths          []string `json:"paths,omitempty"`
	Packages       []string `json:"packages,omitempty"`
	Files          []string `json:"files,omitempty"`
	StartCommit    string   `json:"start_commit,omitempty"`
	EndCommit      string   `json:"end_commit,omitempty"`
	ColumnID       string   `json:"column_id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Priority       string   `json:"priority,omitempty"`
	DueAt          *string  `json:"due_at,omitempty"`
	Labels         []string `json:"labels,omitempty"`
	CreatedByActor string   `json:"created_by_actor,omitempty"`
	CreatedByName  string   `json:"created_by_name,omitempty"`
}

// ToAppInput converts the DTO to an app.CreateActionItemInput.
func (dto CreateActionItemInputDTO) ToAppInput() app.CreateActionItemInput {
	return app.CreateActionItemInput{
		ProjectID:      dto.ProjectID,
		ParentID:       dto.ParentID,
		Kind:           domain.Kind(dto.Kind),
		StructuralType: domain.StructuralType(dto.StructuralType),
		Owner:          dto.Owner,
		DropNumber:     dto.DropNumber,
		Persistent:     dto.Persistent,
		DevGated:       dto.DevGated,
		Paths:          dto.Paths,
		Packages:       dto.Packages,
		Files:          dto.Files,
		StartCommit:    dto.StartCommit,
		EndCommit:      dto.EndCommit,
		ColumnID:       dto.ColumnID,
		Title:          dto.Title,
		Description:    dto.Description,
		Priority:       domain.Priority(dto.Priority),
		DueAt:          parseDueAt(dto.DueAt),
		Labels:         dto.Labels,
		CreatedByActor: dto.CreatedByActor,
		CreatedByName:  dto.CreatedByName,
	}
}

// UpdateActionItemInputDTO holds input values for updating an action item.
type UpdateActionItemInputDTO struct {
	ActionItemID   string    `json:"action_item_id"`
	Title          *string   `json:"title,omitempty"`
	Description    *string   `json:"description,omitempty"`
	Priority       *string   `json:"priority,omitempty"`
	DueAt          **string  `json:"due_at,omitempty"`
	Labels         *[]string `json:"labels,omitempty"`
	Role           string    `json:"role,omitempty"`
	StructuralType string    `json:"structural_type,omitempty"`
	Owner          *string   `json:"owner,omitempty"`
	DropNumber     *int      `json:"drop_number,omitempty"`
	Persistent     *bool     `json:"persistent,omitempty"`
	DevGated       *bool     `json:"dev_gated,omitempty"`
	Paths          *[]string `json:"paths,omitempty"`
	Packages       *[]string `json:"packages,omitempty"`
	Files          *[]string `json:"files,omitempty"`
	StartCommit    *string   `json:"start_commit,omitempty"`
	EndCommit      *string   `json:"end_commit,omitempty"`
}

// ToAppInput converts the DTO to an app.UpdateActionItemInput.
func (dto UpdateActionItemInputDTO) ToAppInput() app.UpdateActionItemInput {
	return app.UpdateActionItemInput{
		ActionItemID:   dto.ActionItemID,
		Title:          dto.Title,
		Description:    dto.Description,
		Priority:       toPriorityPtr(dto.Priority),
		DueAt:          parseDueAtPtr(dto.DueAt),
		Labels:         dto.Labels,
		Role:           domain.Role(dto.Role),
		StructuralType: domain.StructuralType(dto.StructuralType),
		Owner:          dto.Owner,
		DropNumber:     dto.DropNumber,
		Persistent:     dto.Persistent,
		DevGated:       dto.DevGated,
		Paths:          dto.Paths,
		Packages:       dto.Packages,
		Files:          dto.Files,
		StartCommit:    dto.StartCommit,
		EndCommit:      dto.EndCommit,
	}
}

// Helper functions

// timeToPtr converts a *time.Time to a *string in RFC3339 format, or nil.
func timeToPtr(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// parseDueAt parses an RFC3339 string pointer into a *time.Time.
func parseDueAt(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		// Graceful fallback for unparseable input.
		return nil
	}
	return &t
}

// parseDueAtPtr parses an RFC3339 string **string into a **time.Time.
func parseDueAtPtr(ss **string) **time.Time {
	if ss == nil {
		return nil
	}
	if *ss == nil {
		return nil
	}
	t := parseDueAt(*ss)
	return &t
}

// toPriorityPtr converts a *string to a *domain.Priority.
func toPriorityPtr(s *string) *domain.Priority {
	if s == nil || *s == "" {
		return nil
	}
	p := domain.Priority(*s)
	return &p
}
