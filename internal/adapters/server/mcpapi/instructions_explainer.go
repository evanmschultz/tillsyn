package mcpapi

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/evanmschultz/tillsyn/internal/adapters/server/common"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// instructionsExplainServices stores the runtime readers used for scoped explanations.
type instructionsExplainServices struct {
	bootstrap common.BootstrapGuideReader
	projects  common.ProjectService
	tasks     common.ActionItemService
	kinds     common.KindCatalogService
}

// instructionsExplainRequest stores one scoped explanation request.
type instructionsExplainRequest struct {
	Focus           instructionsToolFocus
	Topic           string
	ProjectID       string
	KindID          string
	NodeID          string
	IncludeEvidence bool
}

// instructionsExplainResult stores the synthesized explanation and resolved runtime scope.
type instructionsExplainResult struct {
	Summary       string
	ResolvedScope instructionsToolResolvedScope
	Explanation   instructionsToolExplanation
}

// explainInstructionsScope resolves one scoped instructions explanation from runtime state.
func explainInstructionsScope(ctx context.Context, services instructionsExplainServices, req instructionsExplainRequest) (instructionsExplainResult, error) {
	switch req.Focus {
	case instructionsToolFocusTopic:
		return explainTopicInstructions(ctx, services, req)
	case instructionsToolFocusProject:
		return explainProjectInstructions(ctx, services, req)
	case instructionsToolFocusKind:
		return explainKindInstructions(ctx, services, req)
	case instructionsToolFocusNode:
		return explainNodeInstructions(ctx, services, req)
	default:
		return instructionsExplainResult{}, fmt.Errorf("invalid_request: unsupported focus %q", req.Focus)
	}
}

// explainTopicInstructions returns generic or bootstrap-specific workflow guidance when no concrete runtime object was requested.
func explainTopicInstructions(ctx context.Context, services instructionsExplainServices, req instructionsExplainRequest) (instructionsExplainResult, error) {
	topic := strings.TrimSpace(req.Topic)
	if strings.EqualFold(topic, "bootstrap") && services.bootstrap != nil {
		guide, err := services.bootstrap.GetBootstrapGuide(ctx)
		if err != nil {
			return instructionsExplainResult{}, fmt.Errorf("get bootstrap guide: %w", err)
		}
		return explainBootstrapTopic(guide), nil
	}
	title := "General Till Guidance"
	overview := "Treat Tillsyn as a multi-actor coordination runtime and use embedded docs plus scoped runtime reads to understand workflow policy, coordination, and auth."
	if topic != "" {
		title = fmt.Sprintf("Topic Guidance: %s", topic)
		overview = fmt.Sprintf("Use embedded docs and runtime context together for %s guidance.", topic)
	}
	explanation := instructionsToolExplanation{
		Title:    title,
		Overview: overview,
		ScopedRules: []string{
			"Tillsyn is a multi-actor coordination runtime, not just a planning ledger; name the owning role and use the right coordination surface for the job.",
			"Keep active tasks, actions, blockers, comments, handoffs, and worklogs in Tillsyn itself rather than markdown files.",
			"Use till.get_instructions for policy context, not as a replacement for direct runtime state tools.",
			"Use till.comment for shared discussion, till.handoff for explicit next-action routing, and till.attention_item for durable inbox/notification state.",
			"Use till.capture_state first after restart, then rebuild inbox, handoff, and thread state before resuming watchers.",
		},
		AgentExpectations: []string{
			"Prefer MCP surfaces over CLI for live dogfood flows unless the operator explicitly asks for CLI validation.",
			"If workflow policy changes, update AGENTS.md, any tracked CLAUDE.md, and the relevant bootstrap/instructions docs together so clients stay aligned, but do not use those files as live worklogs.",
			"Keep role ownership explicit: orchestrator routes and cleans up, builder implements, QA verifies, research gathers evidence, and human approval stays visible when required.",
			"Use project-scoped approved sessions for guarded in-project mutation work and global sessions for template/global admin only.",
		},
		RelatedTools: []instructionsToolRelatedTool{
			{Tool: "till.get_instructions", Reason: "embedded docs and scoped explanations"},
			{Tool: "till.capture_state", Reason: "summary-first restart recovery"},
			{Tool: "till.attention_item", Operation: "list|raise|resolve", Reason: "durable inbox and notification state"},
			{Tool: "till.comment", Operation: "create|list", Reason: "shared coordination threads"},
			{Tool: "till.handoff", Operation: "create|list|update", Reason: "durable next-action routing"},
		},
	}
	summary := "General workflow guidance is available through embedded docs plus the scoped runtime tools."
	if topic != "" {
		summary = fmt.Sprintf("General %s guidance is available through embedded docs plus the scoped runtime tools.", topic)
	}
	return instructionsExplainResult{
		Summary:       summary,
		ResolvedScope: instructionsToolResolvedScope{},
		Explanation:   explanation,
	}, nil
}

// explainBootstrapTopic lifts the lightweight bootstrap guide into the richer instructions explanation shape.
func explainBootstrapTopic(guide common.BootstrapGuide) instructionsExplainResult {
	scopedRules := []string{
		"Bootstrap is for empty-instance and first-project setup; after work already exists, prefer till.capture_state plus scoped coordination reads instead of re-running bootstrap.",
		"Keep active tasks, actions, blockers, comments, handoffs, and worklogs in Tillsyn itself; do not create markdown actionItem trackers, worklogs, or temporary execution plans for the run.",
		"Do not use another agent's or user's session, session secret, or auth_context_id during bootstrap or later workflow steps.",
		"Claim or validate your own narrow approved session, then clean up child auth sessions, stale leases, and leftover coordination rows truthfully after the run.",
		"When stable and dev runtimes both exist, confirm which runtime root or DB you are talking to before interpreting missing templates, kinds, or drift state.",
	}
	workflow := append([]string(nil), guide.NextSteps...)
	related := make([]instructionsToolRelatedTool, 0, len(guide.Recommended))
	for _, tool := range guide.Recommended {
		reason := "recommended during bootstrap and first-project setup"
		switch tool {
		case "till.get_instructions":
			reason = "canonical policy and scoped explanation surface after bootstrap collapse"
		case "till.capture_state":
			reason = "restart recovery once the instance already has project state"
		case "till.auth_request":
			reason = "request, claim, validate, and clean up scoped auth"
		}
		related = append(related, instructionsToolRelatedTool{Tool: tool, Reason: reason})
	}
	gaps := make([]string, 0, 1)
	if strings.TrimSpace(guide.RoadmapNotice) != "" {
		gaps = append(gaps, guide.RoadmapNotice)
	}
	return instructionsExplainResult{
		Summary: guide.Summary,
		Explanation: instructionsToolExplanation{
			Title:            "Bootstrap Guidance",
			Overview:         strings.TrimSpace(guide.WhatTillsynIs),
			ScopedRules:      scopedRules,
			WorkflowContract: workflow,
			AgentExpectations: []string{
				"Use till.get_instructions(topic=bootstrap) as the canonical bootstrap explanation surface; till.get_bootstrap_guide is the compatibility wrapper on the frozen MCP family.",
				"If workflow policy changes, update AGENTS.md, any tracked CLAUDE.md, and the relevant bootstrap/instructions surfaces together so agent guidance stays synchronized, but keep live execution state in Tillsyn.",
				"Once a project exists, move from bootstrap into scoped project, template, kind, and node explanations instead of relying on generic startup text.",
			},
			RelatedTools: related,
			Gaps:         gaps,
		},
	}
}

// explainProjectInstructions resolves one project-scoped explanation from project and allowlist state.
func explainProjectInstructions(ctx context.Context, services instructionsExplainServices, req instructionsExplainRequest) (instructionsExplainResult, error) {
	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		return instructionsExplainResult{}, fmt.Errorf(`invalid_request: focus "project" requires project_id`)
	}
	project, err := findProjectByID(ctx, services.projects, projectID)
	if err != nil {
		return instructionsExplainResult{}, err
	}

	allowedKinds, err := listProjectAllowedKinds(ctx, services.kinds, projectID)
	if err != nil {
		return instructionsExplainResult{}, err
	}

	rules := make([]string, 0, 6)
	if standards := strings.TrimSpace(project.Metadata.StandardsMarkdown); standards != "" {
		rules = append(rules, "Project standards are defined in standards_markdown and should be treated as scoped execution policy for this project.")
	}
	if len(allowedKinds) > 0 {
		rules = append(rules, fmt.Sprintf("Allowed kinds in this project are constrained to: %s.", strings.Join(allowedKinds, ", ")))
	}

	workflow := []string{
		"Use project-scoped approved sessions for guarded in-project mutations in this project.",
		"Use till.comment for shared discussion and till.handoff for explicit next-action routing inside this project.",
	}

	expectations := []string{
		"Builders should implement work and report progress in thread comments or handoffs.",
		"QA should verify outcomes and resolve or return handoffs instead of silently editing workflow state.",
		"Research should gather evidence, summarize findings, and hand off the result back into the same project scope.",
	}

	evidence := make([]instructionsToolEvidence, 0, 2)
	if req.IncludeEvidence {
		if standards := strings.TrimSpace(project.Metadata.StandardsMarkdown); standards != "" {
			evidence = append(evidence, instructionsToolEvidence{
				Kind:     "project_standards_markdown",
				ID:       project.ID,
				Summary:  "Project-scoped standards and workflow rules",
				Markdown: standards,
			})
		}
	}

	gaps := make([]string, 0, 1)
	if strings.TrimSpace(project.Metadata.StandardsMarkdown) == "" {
		gaps = append(gaps, "This project has no standards_markdown yet, so only generic project workflow guidance is available.")
	}

	return instructionsExplainResult{
		Summary: fmt.Sprintf("Project %q is explainable from project metadata and allowed kinds.", project.Name),
		ResolvedScope: instructionsToolResolvedScope{
			ProjectID: project.ID,
		},
		Explanation: instructionsToolExplanation{
			Title:             project.Name,
			Overview:          fmt.Sprintf("Project %q.", project.Name),
			WhyItApplies:      buildProjectWhyItApplies(project),
			ScopedRules:       rules,
			WorkflowContract:  workflow,
			AgentExpectations: expectations,
			RelatedTools: []instructionsToolRelatedTool{
				{Tool: "till.project", Operation: "list_allowed_kinds", Reason: "inspect project kind allowlist"},
				{Tool: "till.comment", Operation: "create|list", Reason: "coordinate inside the project thread or project-scoped nodes"},
				{Tool: "till.handoff", Operation: "create|list|update", Reason: "route explicit next-action work inside the project"},
			},
			Evidence: evidence,
			Gaps:     gaps,
		},
	}, nil
}

// explainKindInstructions resolves one kind-scoped explanation, optionally narrowed by project context.
func explainKindInstructions(ctx context.Context, services instructionsExplainServices, req instructionsExplainRequest) (instructionsExplainResult, error) {
	kindID := domain.NormalizeKindID(domain.KindID(req.KindID))
	if kindID == "" {
		return instructionsExplainResult{}, fmt.Errorf(`invalid_request: focus "kind" requires kind_id`)
	}
	kind, err := findKindByID(ctx, services.kinds, kindID)
	if err != nil {
		return instructionsExplainResult{}, err
	}

	projectID := strings.TrimSpace(req.ProjectID)

	rules := []string{
		fmt.Sprintf("Kind %q applies to scope(s): %s.", kind.ID, joinKindScopes(kind.AppliesTo)),
	}
	if len(kind.AllowedParentScopes) > 0 {
		rules = append(rules, fmt.Sprintf("Allowed parent scopes for this kind: %s.", joinKindScopes(kind.AllowedParentScopes)))
	}
	if projectID != "" {
		allowedKinds, err := listProjectAllowedKinds(ctx, services.kinds, projectID)
		if err != nil {
			return instructionsExplainResult{}, err
		}
		if slices.Contains(allowedKinds, string(kind.ID)) {
			rules = append(rules, fmt.Sprintf("Project %q currently allows kind %q.", projectID, kind.ID))
		} else {
			rules = append(rules, fmt.Sprintf("Project %q does not currently allow kind %q.", projectID, kind.ID))
		}
	}

	gaps := make([]string, 0, 2)
	if strings.TrimSpace(kind.DescriptionMarkdown) == "" {
		gaps = append(gaps, fmt.Sprintf("Kind %q has no description_markdown yet.", kind.ID))
	}
	if projectID == "" {
		gaps = append(gaps, "No project context was supplied, so project-specific usage for this kind may be incomplete.")
	}

	evidence := make([]instructionsToolEvidence, 0, 2)
	if req.IncludeEvidence && strings.TrimSpace(kind.DescriptionMarkdown) != "" {
		evidence = append(evidence, instructionsToolEvidence{
			Kind:     "kind_description_markdown",
			ID:       string(kind.ID),
			Summary:  fmt.Sprintf("Kind %s description", kind.ID),
			Markdown: strings.TrimSpace(kind.DescriptionMarkdown),
		})
	}

	resolved := instructionsToolResolvedScope{
		ProjectID:       projectID,
		KindID:          string(kind.ID),
		KindDisplayName: strings.TrimSpace(kind.DisplayName),
	}
	return instructionsExplainResult{
		Summary:       fmt.Sprintf("Kind %q is explainable from the kind catalog%s.", kind.ID, kindContextSuffix(projectID)),
		ResolvedScope: resolved,
		Explanation: instructionsToolExplanation{
			Title:    fallbackText(strings.TrimSpace(kind.DisplayName), string(kind.ID)),
			Overview: fmt.Sprintf("Kind %q is a reusable catalog definition for %s scope(s).", kind.ID, joinKindScopes(kind.AppliesTo)),
			WhyItApplies: []string{
				"Kind definitions define reusable scope semantics, parent constraints, and baseline defaults.",
				"Project allowlists narrow how this kind should be used in one concrete project.",
			},
			ScopedRules:      rules,
			WorkflowContract: []string{"Use till.kind for catalog-level meaning and till.project for scoped usage context."},
			AgentExpectations: []string{
				"Do not infer scoped workflow rules from kind catalog data alone; combine it with project policy and node metadata.",
			},
			RelatedTools: []instructionsToolRelatedTool{
				{Tool: "till.kind", Operation: "list", Reason: "inspect catalog definitions"},
				{Tool: "till.project", Operation: "list_allowed_kinds", Reason: "check project allowlist context"},
			},
			Evidence: evidence,
			Gaps:     gaps,
		},
	}, nil
}

// explainNodeInstructions resolves one branch|phase|actionItem|subtask explanation from node lineage and stored contract state.
func explainNodeInstructions(ctx context.Context, services instructionsExplainServices, req instructionsExplainRequest) (instructionsExplainResult, error) {
	nodeID := strings.TrimSpace(req.NodeID)
	if nodeID == "" {
		return instructionsExplainResult{}, fmt.Errorf(`invalid_request: focus "node" requires node_id`)
	}
	if services.tasks == nil {
		return instructionsExplainResult{}, fmt.Errorf("not found: actionItem service is unavailable for node explanation")
	}
	actionItem, err := services.tasks.GetActionItem(ctx, nodeID)
	if err != nil {
		return instructionsExplainResult{}, fmt.Errorf("get node %q: %w", nodeID, err)
	}
	project, err := findProjectByID(ctx, services.projects, strings.TrimSpace(actionItem.ProjectID))
	if err != nil {
		return instructionsExplainResult{}, err
	}
	lineageActionItems, err := loadActionItemLineage(ctx, services.tasks, actionItem)
	if err != nil {
		return instructionsExplainResult{}, err
	}
	lineage := summarizeActionItemLineage(lineageActionItems)

	kind, kindFound, err := tryFindKindByID(ctx, services.kinds, domain.KindID(actionItem.Kind))
	if err != nil {
		return instructionsExplainResult{}, err
	}

	rules := collectNodeScopedRules(project, actionItem)
	workflow := collectNodeWorkflowContract(actionItem, kind, kindFound)
	expectations := collectNodeAgentExpectations(actionItem)
	why := collectNodeWhyItApplies(project, actionItem, kind, kindFound)
	evidence := collectNodeEvidence(project, actionItem, req.IncludeEvidence)
	gaps := collectNodeGaps(project, actionItem, kindFound)

	resolved := instructionsToolResolvedScope{
		ProjectID:     project.ID,
		KindID:        strings.TrimSpace(string(actionItem.Kind)),
		NodeID:        actionItem.ID,
		NodeScopeType: strings.TrimSpace(string(actionItem.Scope)),
		NodeTitle:     strings.TrimSpace(actionItem.Title),
		Lineage:       lineage,
	}
	if kindFound {
		resolved.KindDisplayName = strings.TrimSpace(kind.DisplayName)
	}

	return instructionsExplainResult{
		Summary:       fmt.Sprintf("%s %q is explainable from node lineage and project standards.", strings.Title(string(actionItem.Scope)), actionItem.Title),
		ResolvedScope: resolved,
		Explanation: instructionsToolExplanation{
			Title:             strings.TrimSpace(actionItem.Title),
			Overview:          fmt.Sprintf("%s %q belongs to project %q.", strings.Title(string(actionItem.Scope)), actionItem.Title, project.Name),
			WhyItApplies:      why,
			ScopedRules:       rules,
			WorkflowContract:  workflow,
			AgentExpectations: expectations,
			RelatedTools: []instructionsToolRelatedTool{
				{Tool: "till.action_item", Operation: "get|update|move_state", Reason: "inspect or advance the node lifecycle"},
				{Tool: "till.comment", Operation: "create|list", Reason: "coordinate on the node thread"},
				{Tool: "till.handoff", Operation: "create|list|update", Reason: "route explicit next-action work for the node"},
			},
			Evidence: evidence,
			Gaps:     gaps,
		},
	}, nil
}

// findProjectByID loads one project by id through the project service.
func findProjectByID(ctx context.Context, service common.ProjectService, projectID string) (domain.Project, error) {
	if service == nil {
		return domain.Project{}, fmt.Errorf("not found: project service is unavailable")
	}
	projectID = strings.TrimSpace(projectID)
	rows, err := service.ListProjects(ctx, true)
	if err != nil {
		return domain.Project{}, fmt.Errorf("list projects: %w", err)
	}
	for _, project := range rows {
		if strings.TrimSpace(project.ID) == projectID {
			return project, nil
		}
	}
	return domain.Project{}, fmt.Errorf("not found: project %q", projectID)
}

// findKindByID loads one kind definition by id through the catalog service.
func findKindByID(ctx context.Context, service common.KindCatalogService, kindID domain.KindID) (domain.KindDefinition, error) {
	kind, found, err := tryFindKindByID(ctx, service, kindID)
	if err != nil {
		return domain.KindDefinition{}, err
	}
	if !found {
		return domain.KindDefinition{}, fmt.Errorf("not found: kind %q", kindID)
	}
	return kind, nil
}

// tryFindKindByID loads one kind definition when it exists.
func tryFindKindByID(ctx context.Context, service common.KindCatalogService, kindID domain.KindID) (domain.KindDefinition, bool, error) {
	if service == nil {
		return domain.KindDefinition{}, false, nil
	}
	kindID = domain.NormalizeKindID(kindID)
	if kindID == "" {
		return domain.KindDefinition{}, false, nil
	}
	rows, err := service.ListKindDefinitions(ctx, true)
	if err != nil {
		return domain.KindDefinition{}, false, fmt.Errorf("list kind definitions: %w", err)
	}
	for _, kind := range rows {
		if domain.NormalizeKindID(kind.ID) == kindID {
			return kind, true, nil
		}
	}
	return domain.KindDefinition{}, false, nil
}

// listProjectAllowedKinds loads one project allowlist when the kind service is available.
func listProjectAllowedKinds(ctx context.Context, service common.KindCatalogService, projectID string) ([]string, error) {
	if service == nil {
		return nil, nil
	}
	rows, err := service.ListProjectAllowedKinds(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return nil, fmt.Errorf("list project allowed kinds: %w", err)
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

// loadActionItemLineage loads the root-to-leaf lineage for one node using repeated GetActionItem reads.
func loadActionItemLineage(ctx context.Context, service common.ActionItemService, leaf domain.ActionItem) ([]domain.ActionItem, error) {
	if service == nil {
		return nil, fmt.Errorf("not found: actionItem service is unavailable")
	}
	reversed := make([]domain.ActionItem, 0, 8)
	current := leaf
	seen := map[string]struct{}{}
	for {
		currentID := strings.TrimSpace(current.ID)
		if currentID == "" {
			break
		}
		if _, ok := seen[currentID]; ok {
			return nil, fmt.Errorf("invalid actionItem lineage: cycle at %q", currentID)
		}
		seen[currentID] = struct{}{}
		reversed = append(reversed, current)
		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, err := service.GetActionItem(ctx, parentID)
		if err != nil {
			return nil, fmt.Errorf("get parent actionItem %q: %w", parentID, err)
		}
		current = parent
	}
	lineage := make([]domain.ActionItem, 0, len(reversed))
	for idx := len(reversed) - 1; idx >= 0; idx-- {
		lineage = append(lineage, reversed[idx])
	}
	return lineage, nil
}

// summarizeActionItemLineage converts one actionItem lineage into readable scope markers.
func summarizeActionItemLineage(lineage []domain.ActionItem) []string {
	out := make([]string, 0, len(lineage))
	for _, actionItem := range lineage {
		label := fmt.Sprintf("%s:%s", actionItem.Scope, strings.TrimSpace(actionItem.Title))
		out = append(out, label)
	}
	return out
}

// collectNodeScopedRules lifts scoped rule sources from project and node metadata.
func collectNodeScopedRules(project domain.Project, actionItem domain.ActionItem) []string {
	rules := make([]string, 0, 12)
	if standards := strings.TrimSpace(project.Metadata.StandardsMarkdown); standards != "" {
		rules = append(rules, "Project standards_markdown applies to this node and should be treated as local execution policy.")
	}
	if desc := strings.TrimSpace(actionItem.Description); desc != "" {
		rules = append(rules, "The node description contains scoped workflow or implementation context for this exact branch, phase, actionItem, or subtask.")
	}
	if objective := strings.TrimSpace(actionItem.Metadata.Objective); objective != "" {
		rules = append(rules, "Objective: "+objective)
	}
	if notes := strings.TrimSpace(actionItem.Metadata.ImplementationNotesAgent); notes != "" {
		rules = append(rules, "Agent notes: "+notes)
	}
	if acceptance := strings.TrimSpace(actionItem.Metadata.AcceptanceCriteria); acceptance != "" {
		rules = append(rules, "Acceptance criteria: "+acceptance)
	}
	if dod := strings.TrimSpace(actionItem.Metadata.DefinitionOfDone); dod != "" {
		rules = append(rules, "Definition of done: "+dod)
	}
	if validation := strings.TrimSpace(actionItem.Metadata.ValidationPlan); validation != "" {
		rules = append(rules, "Validation plan: "+validation)
	}
	if len(actionItem.Metadata.DependsOn) > 0 {
		rules = append(rules, fmt.Sprintf("Depends on: %s. Treat these as prerequisites before starting or closing this node.", strings.Join(actionItem.Metadata.DependsOn, ", ")))
	}
	if len(actionItem.Metadata.BlockedBy) > 0 {
		rules = append(rules, fmt.Sprintf("Blocked by: %s. This node should remain blocked until those dependencies are resolved.", strings.Join(actionItem.Metadata.BlockedBy, ", ")))
	}
	if blockedReason := strings.TrimSpace(actionItem.Metadata.BlockedReason); blockedReason != "" {
		rules = append(rules, "Blocked reason: "+blockedReason)
	}
	if len(actionItem.Metadata.CommandSnippets) > 0 {
		rules = append(rules, fmt.Sprintf("Command snippets are attached to this node: %s.", strings.Join(actionItem.Metadata.CommandSnippets, ", ")))
	}
	return rules
}

// collectNodeWorkflowContract lifts kind contract facts that affect how one node should move.
func collectNodeWorkflowContract(actionItem domain.ActionItem, kind domain.KindDefinition, kindFound bool) []string {
	contract := make([]string, 0, 4)
	contract = append(contract, fmt.Sprintf("Node scope is %q and kind is %q.", actionItem.Scope, actionItem.Kind))
	if kindFound {
		contract = append(contract, fmt.Sprintf("Catalog kind %q applies to: %s.", kind.ID, joinKindScopes(kind.AppliesTo)))
	}
	if len(actionItem.Metadata.DependsOn) > 0 || len(actionItem.Metadata.BlockedBy) > 0 {
		contract = append(contract, "ActionItem-level sequencing is currently expressed through depends_on, blocked_by, and blocked_reason rather than visual board order alone.")
	}
	return contract
}

// collectNodeAgentExpectations summarizes role expectations for one node.
func collectNodeAgentExpectations(actionItem domain.ActionItem) []string {
	return []string{
		"Use till.comment for shared status and evidence on this node.",
		"Use till.handoff when the next action belongs to another actor or role.",
		fmt.Sprintf("This %s node relies on project policy and the node's own metadata for scoped rules.", actionItem.Scope),
	}
}

// collectNodeWhyItApplies explains why the returned rules apply to one node.
func collectNodeWhyItApplies(project domain.Project, _ domain.ActionItem, kind domain.KindDefinition, kindFound bool) []string {
	why := []string{
		fmt.Sprintf("This node belongs to project %q, so project-scoped standards and allowed kinds apply.", project.Name),
	}
	if kindFound {
		why = append(why, fmt.Sprintf("Kind %q defines the base semantics for this node scope.", kind.ID))
	}
	return why
}

// collectNodeEvidence returns concrete policy evidence for one node explanation.
func collectNodeEvidence(project domain.Project, actionItem domain.ActionItem, include bool) []instructionsToolEvidence {
	if !include {
		return nil
	}
	evidence := make([]instructionsToolEvidence, 0, 6)
	if standards := strings.TrimSpace(project.Metadata.StandardsMarkdown); standards != "" {
		evidence = append(evidence, instructionsToolEvidence{
			Kind:     "project_standards_markdown",
			ID:       project.ID,
			Summary:  "Project standards applied to this node",
			Markdown: standards,
		})
	}
	appendMarkdownEvidence := func(kind, id, summary, markdown string) {
		markdown = strings.TrimSpace(markdown)
		if markdown == "" {
			return
		}
		evidence = append(evidence, instructionsToolEvidence{
			Kind:     kind,
			ID:       id,
			Summary:  summary,
			Markdown: markdown,
		})
	}
	appendMarkdownEvidence("node_description", actionItem.ID, "Node description", actionItem.Description)
	appendMarkdownEvidence("node_objective", actionItem.ID, "Node objective", actionItem.Metadata.Objective)
	appendMarkdownEvidence("node_agent_notes", actionItem.ID, "Node implementation notes for agents", actionItem.Metadata.ImplementationNotesAgent)
	appendMarkdownEvidence("node_acceptance_criteria", actionItem.ID, "Node acceptance criteria", actionItem.Metadata.AcceptanceCriteria)
	appendMarkdownEvidence("node_definition_of_done", actionItem.ID, "Node definition of done", actionItem.Metadata.DefinitionOfDone)
	appendMarkdownEvidence("node_validation_plan", actionItem.ID, "Node validation plan", actionItem.Metadata.ValidationPlan)
	return evidence
}

// collectNodeGaps reports which rule sources are missing for one node explanation.
func collectNodeGaps(project domain.Project, actionItem domain.ActionItem, kindFound bool) []string {
	gaps := make([]string, 0, 4)
	if strings.TrimSpace(project.Metadata.StandardsMarkdown) == "" {
		gaps = append(gaps, "Project standards_markdown is empty, so project-local execution rules are not explicit yet.")
	}
	if strings.TrimSpace(actionItem.Metadata.Objective) == "" &&
		strings.TrimSpace(actionItem.Metadata.AcceptanceCriteria) == "" &&
		strings.TrimSpace(actionItem.Metadata.DefinitionOfDone) == "" &&
		strings.TrimSpace(actionItem.Metadata.ValidationPlan) == "" {
		gaps = append(gaps, "This node has little or no scoped actionItem metadata yet, so only generic workflow guidance is available.")
	}
	if !kindFound {
		gaps = append(gaps, "Kind-catalog detail for this node could not be resolved.")
	}
	return gaps
}

// buildProjectWhyItApplies returns one project-scoped explanation list.
func buildProjectWhyItApplies(project domain.Project) []string {
	return []string{
		fmt.Sprintf("Project metadata and standards_markdown are the canonical project-local rule source for %q.", project.Name),
	}
}

// kindContextSuffix returns one scope-context suffix for kind explanations.
func kindContextSuffix(projectID string) string {
	if strings.TrimSpace(projectID) != "" {
		return fmt.Sprintf(" in project %q", projectID)
	}
	return ""
}

// fallbackText returns the fallback string when the primary value is empty.
func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

// joinKindScopes returns one readable scope list for kind definitions.
func joinKindScopes(scopes []domain.KindAppliesTo) string {
	parts := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		value := strings.TrimSpace(string(scope))
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}
