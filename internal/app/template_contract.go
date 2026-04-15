package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// templateContractActor stores the resolved workflow actor context for node-contract checks.
type templateContractActor struct {
	IsHuman  bool
	IsSystem bool
	Kind     domain.TemplateActorKind
}

// currentMutationActorType returns the actor type that should drive guard enforcement for this request.
func currentMutationActorType(ctx context.Context, explicit domain.ActorType) domain.ActorType {
	if actor, ok := MutationActorFromContext(ctx); ok {
		return normalizeActorTypeInput(actor.ActorType)
	}
	if strings.TrimSpace(string(explicit)) != "" {
		return normalizeActorTypeInput(explicit)
	}
	return domain.ActorTypeUser
}

// ensureTaskEditableByNodeContract blocks non-human edits that violate a stored generated-node contract.
func (s *Service) ensureTaskEditableByNodeContract(ctx context.Context, task domain.Task) error {
	snapshot, ok, err := s.nodeContractSnapshotForTask(ctx, task.ID)
	if err != nil || !ok {
		return err
	}
	actor, err := s.resolveTemplateContractActor(ctx)
	if err != nil {
		return err
	}
	if actor.IsHuman || (actor.IsSystem && internalTemplateMutationAllowed(ctx)) {
		return nil
	}
	if slices.Contains(snapshot.EditableByActorKinds, actor.Kind) {
		return nil
	}
	return fmt.Errorf("%w: %q is editable by %s", domain.ErrNodeContractForbidden, taskDisplayLabel(task), nodeContractActorKindsSummary(snapshot.EditableByActorKinds, true, false))
}

// ensureTaskCompletableByNodeContract blocks non-human completion when the stored generated-node contract forbids it.
func (s *Service) ensureTaskCompletableByNodeContract(ctx context.Context, task domain.Task) error {
	snapshot, ok, err := s.nodeContractSnapshotForTask(ctx, task.ID)
	if err != nil || !ok {
		return err
	}
	actor, err := s.resolveTemplateContractActor(ctx)
	if err != nil {
		return err
	}
	if actor.IsHuman || (actor.IsSystem && internalTemplateMutationAllowed(ctx)) {
		return nil
	}
	if actor.Kind == domain.TemplateActorKindOrchestrator && snapshot.OrchestratorMayComplete {
		return nil
	}
	if slices.Contains(snapshot.CompletableByActorKinds, actor.Kind) {
		return nil
	}
	return fmt.Errorf("%w: %q is completable by %s", domain.ErrNodeContractForbidden, taskDisplayLabel(task), nodeContractActorKindsSummary(snapshot.CompletableByActorKinds, true, snapshot.OrchestratorMayComplete))
}

// ensureTaskCompletionBlockersClear enforces parent and containing-scope blockers from stored node contracts.
func (s *Service) ensureTaskCompletionBlockersClear(ctx context.Context, task domain.Task, projectTasks []domain.Task) error {
	children, descendants := taskChildrenAndDescendants(task.ID, projectTasks)
	activeChildren := make([]domain.Task, 0, len(children))
	blockers := make([]string, 0)
	seen := map[string]struct{}{}

	appendBlocker := func(message string) {
		message = strings.TrimSpace(message)
		if message == "" {
			return
		}
		if _, ok := seen[message]; ok {
			return
		}
		seen[message] = struct{}{}
		blockers = append(blockers, message)
	}

	for _, child := range children {
		if child.ArchivedAt != nil {
			continue
		}
		activeChildren = append(activeChildren, child)
		snapshot, ok, err := s.nodeContractSnapshotForTask(ctx, child.ID)
		if err != nil {
			return err
		}
		if ok && snapshot.RequiredForParentDone && child.LifecycleState != domain.StateDone {
			appendBlocker(formatNodeContractBlocker(child, snapshot, "parent"))
		}
	}
	for _, descendant := range descendants {
		if descendant.ArchivedAt != nil {
			continue
		}
		snapshot, ok, err := s.nodeContractSnapshotForTask(ctx, descendant.ID)
		if err != nil {
			return err
		}
		if ok && snapshot.RequiredForContainingDone && descendant.LifecycleState != domain.StateDone {
			appendBlocker(formatNodeContractBlocker(descendant, snapshot, "containing scope"))
		}
	}
	if len(blockers) > 0 {
		sort.Strings(blockers)
		return fmt.Errorf("%w: %s", domain.ErrTransitionBlocked, strings.Join(blockers, "; "))
	}
	if unmet := task.CompletionCriteriaUnmet(activeChildren); len(unmet) > 0 {
		return fmt.Errorf("%w: completion criteria unmet (%s)", domain.ErrTransitionBlocked, strings.Join(unmet, ", "))
	}
	return nil
}

// nodeContractSnapshotForTask loads one generated-node contract snapshot when present.
func (s *Service) nodeContractSnapshotForTask(ctx context.Context, taskID string) (domain.NodeContractSnapshot, bool, error) {
	snapshot, err := s.repo.GetNodeContractSnapshot(ctx, strings.TrimSpace(taskID))
	if err == nil {
		return snapshot, true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return domain.NodeContractSnapshot{}, false, nil
	}
	return domain.NodeContractSnapshot{}, false, err
}

// resolveTemplateContractActor resolves the current caller into the workflow actor kind used by template contracts.
func (s *Service) resolveTemplateContractActor(ctx context.Context) (templateContractActor, error) {
	if guard, ok := MutationGuardFromContext(ctx); ok {
		lease, err := s.repo.GetCapabilityLease(ctx, guard.AgentInstanceID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return templateContractActor{}, domain.ErrMutationLeaseInvalid
			}
			return templateContractActor{}, err
		}
		if !lease.MatchesIdentity(guard.AgentName, guard.LeaseToken) {
			return templateContractActor{}, domain.ErrMutationLeaseInvalid
		}
		now := s.clock().UTC()
		if lease.IsRevoked() {
			return templateContractActor{}, domain.ErrMutationLeaseRevoked
		}
		if lease.IsExpired(now) {
			return templateContractActor{}, domain.ErrMutationLeaseExpired
		}
		switch domain.NormalizeCapabilityRole(lease.Role) {
		case domain.CapabilityRoleOrchestrator:
			return templateContractActor{Kind: domain.TemplateActorKindOrchestrator}, nil
		case domain.CapabilityRoleBuilder:
			return templateContractActor{Kind: domain.TemplateActorKindBuilder}, nil
		case domain.CapabilityRoleQA:
			return templateContractActor{Kind: domain.TemplateActorKindQA}, nil
		case domain.CapabilityRoleResearch:
			return templateContractActor{Kind: domain.TemplateActorKindResearch}, nil
		case domain.CapabilityRoleSystem:
			return templateContractActor{IsSystem: true}, nil
		default:
			return templateContractActor{}, fmt.Errorf("%w: unsupported capability role %q", domain.ErrNodeContractForbidden, lease.Role)
		}
	}
	if actor, ok := MutationActorFromContext(ctx); ok && normalizeActorTypeInput(actor.ActorType) == domain.ActorTypeSystem {
		return templateContractActor{IsSystem: true}, nil
	}
	if actor, ok := MutationActorFromContext(ctx); ok && normalizeActorTypeInput(actor.ActorType) == domain.ActorTypeAgent {
		return templateContractActor{}, domain.ErrMutationLeaseRequired
	}
	return templateContractActor{IsHuman: true, Kind: domain.TemplateActorKindHuman}, nil
}

// taskChildrenAndDescendants returns direct children plus the full descendant tree in stable traversal order.
func taskChildrenAndDescendants(rootID string, tasks []domain.Task) ([]domain.Task, []domain.Task) {
	rootID = strings.TrimSpace(rootID)
	byParent := make(map[string][]domain.Task)
	for _, task := range tasks {
		parentID := strings.TrimSpace(task.ParentID)
		if parentID == "" {
			continue
		}
		byParent[parentID] = append(byParent[parentID], task)
	}

	children := append([]domain.Task(nil), byParent[rootID]...)
	descendants := make([]domain.Task, 0, len(children))
	queue := append([]domain.Task(nil), children...)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		descendants = append(descendants, current)
		queue = append(queue, byParent[current.ID]...)
	}
	return children, descendants
}

// formatNodeContractBlocker renders one contract-driven completion blocker with the role requirements.
func formatNodeContractBlocker(task domain.Task, snapshot domain.NodeContractSnapshot, scopeLabel string) string {
	return fmt.Sprintf("%s blocker %q is not done (responsible actor kind: %s; completable by: %s)", scopeLabel, taskDisplayLabel(task), snapshot.ResponsibleActorKind, nodeContractActorKindsSummary(snapshot.CompletableByActorKinds, true, snapshot.OrchestratorMayComplete))
}

// nodeContractActorKindsSummary returns a stable human-readable summary of one actor-kind allowlist.
func nodeContractActorKindsSummary(kinds []domain.TemplateActorKind, includeHuman bool, includeOrchestratorOverride bool) string {
	out := make([]string, 0, len(kinds)+2)
	seen := map[string]struct{}{}

	appendRole := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	for _, kind := range kinds {
		appendRole(string(domain.NormalizeTemplateActorKind(kind)))
	}
	if includeOrchestratorOverride {
		appendRole("orchestrator (override)")
	}
	if includeHuman {
		appendRole("human")
	}
	slices.Sort(out)
	return strings.Join(out, ", ")
}

// taskDisplayLabel returns the best available stable display label for one task in error messages.
func taskDisplayLabel(task domain.Task) string {
	if title := strings.TrimSpace(task.Title); title != "" {
		return title
	}
	return strings.TrimSpace(task.ID)
}
