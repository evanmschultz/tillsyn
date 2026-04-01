package app

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hylla/tillsyn/internal/domain"
)

// GetProjectTemplateReapplyPreview returns the current drift summary plus conservative migration-review candidates.
func (s *Service) GetProjectTemplateReapplyPreview(ctx context.Context, projectID string) (domain.ProjectTemplateReapplyPreview, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.ProjectTemplateReapplyPreview{}, domain.ErrInvalidID
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return domain.ProjectTemplateReapplyPreview{}, err
	}
	binding, err := s.GetProjectTemplateBinding(ctx, projectID)
	if err != nil {
		return domain.ProjectTemplateReapplyPreview{}, err
	}
	preview := domain.ProjectTemplateReapplyPreview{
		ProjectID:              projectID,
		LibraryID:              binding.LibraryID,
		LibraryName:            binding.LibraryName,
		DriftStatus:            binding.DriftStatus,
		BoundRevision:          binding.BoundRevision,
		LatestRevision:         binding.LatestRevision,
		BoundLibraryUpdatedAt:  binding.BoundLibraryUpdatedAt,
		LatestLibraryUpdatedAt: binding.LatestLibraryUpdatedAt,
	}
	if binding.DriftStatus == domain.ProjectTemplateBindingDriftLibraryMissing || binding.LatestRevision == 0 {
		return preview, nil
	}
	if binding.BoundLibrarySnapshot == nil {
		return domain.ProjectTemplateReapplyPreview{}, fmt.Errorf("%w: binding snapshot is required for template reapply preview", domain.ErrInvalidTemplateBinding)
	}
	latest, err := s.repo.GetTemplateLibrary(ctx, binding.LibraryID)
	if err != nil {
		return domain.ProjectTemplateReapplyPreview{}, err
	}
	if latest.Status != domain.TemplateLibraryStatusApproved {
		return preview, nil
	}

	preview.ProjectDefaultChanges = projectTemplateDefaultChanges(*binding.BoundLibrarySnapshot, latest, project.Kind)
	changeByRule := projectTemplateChildRuleChangeMap(*binding.BoundLibrarySnapshot, latest)
	preview.ChildRuleChanges = make([]domain.ProjectTemplateChildRuleChange, 0, len(changeByRule))
	for _, change := range changeByRule {
		preview.ChildRuleChanges = append(preview.ChildRuleChanges, change)
	}
	sort.SliceStable(preview.ChildRuleChanges, func(i, j int) bool {
		if preview.ChildRuleChanges[i].NodeTemplateID == preview.ChildRuleChanges[j].NodeTemplateID {
			return preview.ChildRuleChanges[i].ChildRuleID < preview.ChildRuleChanges[j].ChildRuleID
		}
		return preview.ChildRuleChanges[i].NodeTemplateID < preview.ChildRuleChanges[j].NodeTemplateID
	})

	tasks, err := s.repo.ListTasks(ctx, projectID, false)
	if err != nil {
		return domain.ProjectTemplateReapplyPreview{}, err
	}
	for _, task := range tasks {
		snapshot, ok, snapshotErr := s.nodeContractSnapshotForTask(ctx, task.ID)
		if snapshotErr != nil {
			return domain.ProjectTemplateReapplyPreview{}, snapshotErr
		}
		if !ok || strings.TrimSpace(snapshot.SourceLibraryID) != strings.TrimSpace(binding.LibraryID) {
			continue
		}
		ruleKey := projectTemplateRuleKey(snapshot.SourceNodeTemplateID, snapshot.SourceChildRuleID)
		change, ok := changeByRule[ruleKey]
		if !ok {
			continue
		}
		previousRule, previousNodeTemplate, ok := findTemplateChildRule(*binding.BoundLibrarySnapshot, snapshot.SourceNodeTemplateID, snapshot.SourceChildRuleID)
		if !ok {
			continue
		}
		candidate := domain.ProjectTemplateMigrationCandidate{
			TaskID:               task.ID,
			ParentID:             strings.TrimSpace(task.ParentID),
			Title:                strings.TrimSpace(task.Title),
			Scope:                task.Scope,
			Kind:                 task.Kind,
			LifecycleState:       task.LifecycleState,
			SourceNodeTemplateID: snapshot.SourceNodeTemplateID,
			SourceChildRuleID:    snapshot.SourceChildRuleID,
			ChangeKinds:          append([]string(nil), change.ChangeKinds...),
			Status:               domain.ProjectTemplateReapplyCandidateIneligible,
		}
		if reason := templateMigrationIneligibleReason(task, snapshot, previousNodeTemplate, previousRule); reason != "" {
			candidate.Reason = reason
			preview.IneligibleMigrationCount++
		} else {
			candidate.Status = domain.ProjectTemplateReapplyCandidateEligible
			preview.EligibleMigrationCount++
		}
		preview.MigrationCandidates = append(preview.MigrationCandidates, candidate)
	}
	sort.SliceStable(preview.MigrationCandidates, func(i, j int) bool {
		if preview.MigrationCandidates[i].Status == preview.MigrationCandidates[j].Status {
			if preview.MigrationCandidates[i].Title == preview.MigrationCandidates[j].Title {
				return preview.MigrationCandidates[i].TaskID < preview.MigrationCandidates[j].TaskID
			}
			return preview.MigrationCandidates[i].Title < preview.MigrationCandidates[j].Title
		}
		return preview.MigrationCandidates[i].Status < preview.MigrationCandidates[j].Status
	})
	preview.ReviewRequired = binding.DriftStatus == domain.ProjectTemplateBindingDriftUpdateAvailable &&
		(len(preview.ProjectDefaultChanges) > 0 || len(preview.ChildRuleChanges) > 0)
	return preview, nil
}

func projectTemplateDefaultChanges(bound domain.TemplateLibrary, latest domain.TemplateLibrary, projectKind domain.KindID) []domain.ProjectTemplateDefaultChange {
	boundNode, boundOK := bound.FindNodeTemplate(domain.KindAppliesToProject, projectKind)
	latestNode, latestOK := latest.FindNodeTemplate(domain.KindAppliesToProject, projectKind)
	if !boundOK || !latestOK {
		return nil
	}
	boundDefaults, _ := domain.MergeProjectMetadata(domain.ProjectMetadata{}, boundNode.ProjectMetadataDefaults)
	latestDefaults, _ := domain.MergeProjectMetadata(domain.ProjectMetadata{}, latestNode.ProjectMetadataDefaults)
	type fieldValue struct {
		field    string
		previous string
		current  string
	}
	values := []fieldValue{
		{field: "owner", previous: strings.TrimSpace(boundDefaults.Owner), current: strings.TrimSpace(latestDefaults.Owner)},
		{field: "icon", previous: strings.TrimSpace(boundDefaults.Icon), current: strings.TrimSpace(latestDefaults.Icon)},
		{field: "color", previous: strings.TrimSpace(boundDefaults.Color), current: strings.TrimSpace(latestDefaults.Color)},
		{field: "homepage", previous: strings.TrimSpace(boundDefaults.Homepage), current: strings.TrimSpace(latestDefaults.Homepage)},
		{field: "tags", previous: strings.Join(boundDefaults.Tags, ", "), current: strings.Join(latestDefaults.Tags, ", ")},
		{field: "standards_markdown", previous: strings.TrimSpace(boundDefaults.StandardsMarkdown), current: strings.TrimSpace(latestDefaults.StandardsMarkdown)},
		{
			field:    "kind_payload",
			previous: templateJSONValue(boundDefaults.KindPayload),
			current:  templateJSONValue(latestDefaults.KindPayload),
		},
		{
			field:    "capability_policy",
			previous: templateJSONValue(boundDefaults.CapabilityPolicy),
			current:  templateJSONValue(latestDefaults.CapabilityPolicy),
		},
	}
	changes := make([]domain.ProjectTemplateDefaultChange, 0)
	for _, value := range values {
		if value.previous == value.current {
			continue
		}
		changes = append(changes, domain.ProjectTemplateDefaultChange{
			Field:    value.field,
			Previous: value.previous,
			Current:  value.current,
		})
	}
	return changes
}

func projectTemplateChildRuleChangeMap(bound domain.TemplateLibrary, latest domain.TemplateLibrary) map[string]domain.ProjectTemplateChildRuleChange {
	out := map[string]domain.ProjectTemplateChildRuleChange{}
	for _, boundNodeTemplate := range bound.NodeTemplates {
		for _, boundRule := range boundNodeTemplate.ChildRules {
			latestRule, latestNodeTemplate, ok := findTemplateChildRule(latest, boundNodeTemplate.ID, boundRule.ID)
			if !ok {
				continue
			}
			changeKinds := projectTemplateChildRuleChangeKinds(boundRule, latestRule)
			if len(changeKinds) == 0 {
				continue
			}
			key := projectTemplateRuleKey(boundNodeTemplate.ID, boundRule.ID)
			out[key] = domain.ProjectTemplateChildRuleChange{
				NodeTemplateID:                    boundNodeTemplate.ID,
				NodeTemplateName:                  firstNonEmptyTrimmed(boundNodeTemplate.DisplayName, latestNodeTemplate.DisplayName),
				ChildRuleID:                       boundRule.ID,
				ChangeKinds:                       changeKinds,
				PreviousTitleTemplate:             boundRule.TitleTemplate,
				CurrentTitleTemplate:              latestRule.TitleTemplate,
				PreviousDescriptionTemplate:       boundRule.DescriptionTemplate,
				CurrentDescriptionTemplate:        latestRule.DescriptionTemplate,
				PreviousResponsibleActorKind:      boundRule.ResponsibleActorKind,
				CurrentResponsibleActorKind:       latestRule.ResponsibleActorKind,
				PreviousEditableByActorKinds:      append([]domain.TemplateActorKind(nil), boundRule.EditableByActorKinds...),
				CurrentEditableByActorKinds:       append([]domain.TemplateActorKind(nil), latestRule.EditableByActorKinds...),
				PreviousCompletableByActorKinds:   append([]domain.TemplateActorKind(nil), boundRule.CompletableByActorKinds...),
				CurrentCompletableByActorKinds:    append([]domain.TemplateActorKind(nil), latestRule.CompletableByActorKinds...),
				PreviousOrchestratorMayComplete:   boundRule.OrchestratorMayComplete,
				CurrentOrchestratorMayComplete:    latestRule.OrchestratorMayComplete,
				PreviousRequiredForParentDone:     boundRule.RequiredForParentDone,
				CurrentRequiredForParentDone:      latestRule.RequiredForParentDone,
				PreviousRequiredForContainingDone: boundRule.RequiredForContainingDone,
				CurrentRequiredForContainingDone:  latestRule.RequiredForContainingDone,
			}
		}
	}
	return out
}

func projectTemplateChildRuleChangeKinds(previous domain.TemplateChildRule, current domain.TemplateChildRule) []string {
	changes := make([]string, 0, 8)
	if strings.TrimSpace(previous.TitleTemplate) != strings.TrimSpace(current.TitleTemplate) {
		changes = append(changes, "title")
	}
	if strings.TrimSpace(previous.DescriptionTemplate) != strings.TrimSpace(current.DescriptionTemplate) {
		changes = append(changes, "description")
	}
	if previous.ResponsibleActorKind != current.ResponsibleActorKind {
		changes = append(changes, "responsible_actor_kind")
	}
	if !slices.Equal(previous.EditableByActorKinds, current.EditableByActorKinds) {
		changes = append(changes, "editable_by")
	}
	if !slices.Equal(previous.CompletableByActorKinds, current.CompletableByActorKinds) {
		changes = append(changes, "completable_by")
	}
	if previous.OrchestratorMayComplete != current.OrchestratorMayComplete {
		changes = append(changes, "orchestrator_may_complete")
	}
	if previous.RequiredForParentDone != current.RequiredForParentDone {
		changes = append(changes, "required_for_parent_done")
	}
	if previous.RequiredForContainingDone != current.RequiredForContainingDone {
		changes = append(changes, "required_for_containing_done")
	}
	return changes
}

func findTemplateChildRule(library domain.TemplateLibrary, nodeTemplateID string, childRuleID string) (domain.TemplateChildRule, domain.NodeTemplate, bool) {
	nodeTemplateID = domain.NormalizeTemplateLibraryID(nodeTemplateID)
	childRuleID = domain.NormalizeTemplateLibraryID(childRuleID)
	for _, nodeTemplate := range library.NodeTemplates {
		if domain.NormalizeTemplateLibraryID(nodeTemplate.ID) != nodeTemplateID {
			continue
		}
		for _, childRule := range nodeTemplate.ChildRules {
			if domain.NormalizeTemplateLibraryID(childRule.ID) == childRuleID {
				return childRule, nodeTemplate, true
			}
		}
	}
	return domain.TemplateChildRule{}, domain.NodeTemplate{}, false
}

func projectTemplateRuleKey(nodeTemplateID string, childRuleID string) string {
	return domain.NormalizeTemplateLibraryID(nodeTemplateID) + "::" + domain.NormalizeTemplateLibraryID(childRuleID)
}

func templateMigrationIneligibleReason(task domain.Task, snapshot domain.NodeContractSnapshot, previousNodeTemplate domain.NodeTemplate, previousRule domain.TemplateChildRule) string {
	if strings.TrimSpace(task.CreatedByActor) != templateSystemActorID {
		return "task was not created by the template system"
	}
	if strings.TrimSpace(task.UpdatedByActor) != templateSystemActorID {
		return "task has been updated since generation"
	}
	if strings.TrimSpace(task.Title) != strings.TrimSpace(previousRule.TitleTemplate) {
		return "task title no longer matches generated template content"
	}
	if strings.TrimSpace(task.Description) != strings.TrimSpace(previousRule.DescriptionTemplate) {
		return "task description no longer matches generated template content"
	}
	if snapshot.SourceNodeTemplateID != previousNodeTemplate.ID || snapshot.SourceChildRuleID != previousRule.ID {
		return "stored node contract no longer matches the bound template revision"
	}
	if snapshot.ResponsibleActorKind != previousRule.ResponsibleActorKind {
		return "stored node contract responsible actor differs from the bound template revision"
	}
	if !slices.Equal(snapshot.EditableByActorKinds, previousRule.EditableByActorKinds) {
		return "stored node contract editable actors differ from the bound template revision"
	}
	if !slices.Equal(snapshot.CompletableByActorKinds, previousRule.CompletableByActorKinds) {
		return "stored node contract completable actors differ from the bound template revision"
	}
	if snapshot.OrchestratorMayComplete != previousRule.OrchestratorMayComplete {
		return "stored node contract orchestrator override differs from the bound template revision"
	}
	if snapshot.RequiredForParentDone != previousRule.RequiredForParentDone {
		return "stored node contract parent blocker differs from the bound template revision"
	}
	if snapshot.RequiredForContainingDone != previousRule.RequiredForContainingDone {
		return "stored node contract containing blocker differs from the bound template revision"
	}
	return ""
}

func templateJSONValue(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
