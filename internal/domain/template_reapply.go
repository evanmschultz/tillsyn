package domain

import "time"

// ProjectTemplateReapplyCandidateStatus identifies whether one generated node is safe for explicit migration review.
type ProjectTemplateReapplyCandidateStatus string

// ProjectTemplateReapplyCandidateStatus values classify migration-review candidates conservatively.
const (
	ProjectTemplateReapplyCandidateEligible   ProjectTemplateReapplyCandidateStatus = "eligible"
	ProjectTemplateReapplyCandidateIneligible ProjectTemplateReapplyCandidateStatus = "ineligible"
)

// ProjectTemplateDefaultChange stores one project-level default field change across template revisions.
type ProjectTemplateDefaultChange struct {
	Field    string `json:"field"`
	Previous string `json:"previous,omitempty"`
	Current  string `json:"current,omitempty"`
}

// ProjectTemplateChildRuleChange stores one changed child rule across the bound and latest library revisions.
type ProjectTemplateChildRuleChange struct {
	NodeTemplateID                    string              `json:"node_template_id"`
	NodeTemplateName                  string              `json:"node_template_name,omitempty"`
	ChildRuleID                       string              `json:"child_rule_id"`
	ChangeKinds                       []string            `json:"change_kinds,omitempty"`
	PreviousTitleTemplate             string              `json:"previous_title_template,omitempty"`
	CurrentTitleTemplate              string              `json:"current_title_template,omitempty"`
	PreviousDescriptionTemplate       string              `json:"previous_description_template,omitempty"`
	CurrentDescriptionTemplate        string              `json:"current_description_template,omitempty"`
	PreviousResponsibleActorKind      TemplateActorKind   `json:"previous_responsible_actor_kind,omitempty"`
	CurrentResponsibleActorKind       TemplateActorKind   `json:"current_responsible_actor_kind,omitempty"`
	PreviousEditableByActorKinds      []TemplateActorKind `json:"previous_editable_by_actor_kinds,omitempty"`
	CurrentEditableByActorKinds       []TemplateActorKind `json:"current_editable_by_actor_kinds,omitempty"`
	PreviousCompletableByActorKinds   []TemplateActorKind `json:"previous_completable_by_actor_kinds,omitempty"`
	CurrentCompletableByActorKinds    []TemplateActorKind `json:"current_completable_by_actor_kinds,omitempty"`
	PreviousOrchestratorMayComplete   bool                `json:"previous_orchestrator_may_complete,omitempty"`
	CurrentOrchestratorMayComplete    bool                `json:"current_orchestrator_may_complete,omitempty"`
	PreviousRequiredForParentDone     bool                `json:"previous_required_for_parent_done,omitempty"`
	CurrentRequiredForParentDone      bool                `json:"current_required_for_parent_done,omitempty"`
	PreviousRequiredForContainingDone bool                `json:"previous_required_for_containing_done,omitempty"`
	CurrentRequiredForContainingDone  bool                `json:"current_required_for_containing_done,omitempty"`
}

// ProjectTemplateMigrationCandidate stores one existing generated node plus its migration-review eligibility.
type ProjectTemplateMigrationCandidate struct {
	TaskID               string                                `json:"task_id"`
	ParentID             string                                `json:"parent_id,omitempty"`
	Title                string                                `json:"title"`
	Scope                KindAppliesTo                         `json:"scope"`
	Kind                 WorkKind                              `json:"kind"`
	LifecycleState       LifecycleState                        `json:"lifecycle_state"`
	SourceNodeTemplateID string                                `json:"source_node_template_id,omitempty"`
	SourceChildRuleID    string                                `json:"source_child_rule_id,omitempty"`
	Status               ProjectTemplateReapplyCandidateStatus `json:"status"`
	Reason               string                                `json:"reason,omitempty"`
	ChangeKinds          []string                              `json:"change_kinds,omitempty"`
}

// ProjectTemplateMigrationApproval stores one generated node that was explicitly approved for migration.
type ProjectTemplateMigrationApproval struct {
	TaskID      string   `json:"task_id"`
	Title       string   `json:"title"`
	ChangeKinds []string `json:"change_kinds,omitempty"`
	NewTitle    string   `json:"new_title,omitempty"`
	NewBody     string   `json:"new_body,omitempty"`
}

// ProjectTemplateReapplyPreview stores one project's current drift and explicit migration-review preview.
type ProjectTemplateReapplyPreview struct {
	ProjectID                string                              `json:"project_id"`
	LibraryID                string                              `json:"library_id"`
	LibraryName              string                              `json:"library_name,omitempty"`
	DriftStatus              string                              `json:"drift_status,omitempty"`
	BoundRevision            int                                 `json:"bound_revision"`
	LatestRevision           int                                 `json:"latest_revision,omitempty"`
	BoundLibraryUpdatedAt    time.Time                           `json:"bound_library_updated_at"`
	LatestLibraryUpdatedAt   *time.Time                          `json:"latest_library_updated_at,omitempty"`
	ProjectDefaultChanges    []ProjectTemplateDefaultChange      `json:"project_default_changes,omitempty"`
	ChildRuleChanges         []ProjectTemplateChildRuleChange    `json:"child_rule_changes,omitempty"`
	MigrationCandidates      []ProjectTemplateMigrationCandidate `json:"migration_candidates,omitempty"`
	EligibleMigrationCount   int                                 `json:"eligible_migration_count"`
	IneligibleMigrationCount int                                 `json:"ineligible_migration_count"`
	ReviewRequired           bool                                `json:"review_required"`
}

// ProjectTemplateMigrationApprovalResult stores one explicit migration-approval outcome.
type ProjectTemplateMigrationApprovalResult struct {
	ProjectID                string                             `json:"project_id"`
	LibraryID                string                             `json:"library_id"`
	LibraryName              string                             `json:"library_name,omitempty"`
	DriftStatus              string                             `json:"drift_status,omitempty"`
	ApprovedAll              bool                               `json:"approved_all,omitempty"`
	Approvals                []ProjectTemplateMigrationApproval `json:"approvals,omitempty"`
	AppliedCount             int                                `json:"applied_count"`
	RemainingEligibleCount   int                                `json:"remaining_eligible_count"`
	RemainingIneligibleCount int                                `json:"remaining_ineligible_count"`
}
