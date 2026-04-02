package mcpapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
	"github.com/hylla/tillsyn/internal/domain"
)

// instructionsExplainServices stores the runtime readers used for scoped explanations.
type instructionsExplainServices struct {
	projects  common.ProjectService
	tasks     common.TaskService
	kinds     common.KindCatalogService
	templates common.TemplateLibraryService
}

// instructionsExplainRequest stores one scoped explanation request.
type instructionsExplainRequest struct {
	Focus             instructionsToolFocus
	Topic             string
	ProjectID         string
	TemplateLibraryID string
	KindID            string
	NodeID            string
	IncludeEvidence   bool
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
		return explainTopicInstructions(req), nil
	case instructionsToolFocusProject:
		return explainProjectInstructions(ctx, services, req)
	case instructionsToolFocusTemplate:
		return explainTemplateInstructions(ctx, services, req)
	case instructionsToolFocusKind:
		return explainKindInstructions(ctx, services, req)
	case instructionsToolFocusNode:
		return explainNodeInstructions(ctx, services, req)
	default:
		return instructionsExplainResult{}, fmt.Errorf("invalid_request: unsupported focus %q", req.Focus)
	}
}

// explainTopicInstructions returns one generic workflow explanation when no scoped runtime object was requested.
func explainTopicInstructions(req instructionsExplainRequest) instructionsExplainResult {
	topic := strings.TrimSpace(req.Topic)
	title := "General Till Guidance"
	overview := "Use embedded docs plus scoped runtime reads to understand workflow policy, coordination, auth, and template contracts."
	if topic != "" {
		title = fmt.Sprintf("Topic Guidance: %s", topic)
		overview = fmt.Sprintf("Use embedded docs and runtime context together for %s guidance.", topic)
	}
	explanation := instructionsToolExplanation{
		Title:    title,
		Overview: overview,
		ScopedRules: []string{
			"Use till.get_instructions for policy context, not as a replacement for direct runtime state tools.",
			"Use till.comment for shared discussion, till.handoff for explicit next-action routing, and till.attention_item for inbox/notification state.",
			"Use till.capture_state first after restart, then rebuild inbox, handoff, and thread state before resuming watchers.",
		},
		AgentExpectations: []string{
			"Prefer MCP surfaces over CLI for live dogfood flows unless the operator explicitly asks for CLI validation.",
			"Use project-scoped approved sessions for guarded in-project mutation work and global sessions for template/global admin only.",
		},
		RelatedTools: []instructionsToolRelatedTool{
			{Tool: "till.get_instructions", Reason: "embedded docs and scoped explanations"},
			{Tool: "till.capture_state", Reason: "summary-first restart recovery"},
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
	}
}

// explainProjectInstructions resolves one project-scoped explanation from project, allowlist, and binding state.
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
	binding, bindingFound, err := loadProjectBinding(ctx, services.templates, projectID)
	if err != nil {
		return instructionsExplainResult{}, err
	}

	rules := make([]string, 0, 8)
	if standards := strings.TrimSpace(project.Metadata.StandardsMarkdown); standards != "" {
		rules = append(rules, "Project standards are defined in standards_markdown and should be treated as scoped execution policy for this project.")
	}
	if project.Kind != "" {
		rules = append(rules, fmt.Sprintf("Project kind is %q and should be used as the base interpretation for project-level workflow expectations.", project.Kind))
	}
	if len(allowedKinds) > 0 {
		rules = append(rules, fmt.Sprintf("Allowed kinds in this project are constrained to: %s.", strings.Join(allowedKinds, ", ")))
	}
	if bindingFound {
		rules = append(rules, fmt.Sprintf("This project is bound to template library %q; generated node rules and future workflow defaults should be interpreted through that binding.", binding.LibraryID))
	}

	workflow := []string{
		"Use project-scoped approved sessions for guarded in-project mutations in this project.",
		"Use till.comment for shared discussion and till.handoff for explicit next-action routing inside this project.",
	}
	if bindingFound {
		workflow = append(workflow, fmt.Sprintf("Template drift status is %q for the active project binding.", fallbackText(strings.TrimSpace(binding.DriftStatus), "current")))
	}

	expectations := []string{
		"Builders should implement work and report progress in thread comments or handoffs.",
		"QA should verify outcomes and resolve or return handoffs instead of silently editing workflow state.",
		"Research should gather evidence, summarize findings, and hand off the result back into the same project scope.",
	}

	evidence := make([]instructionsToolEvidence, 0, 3)
	if req.IncludeEvidence {
		if standards := strings.TrimSpace(project.Metadata.StandardsMarkdown); standards != "" {
			evidence = append(evidence, instructionsToolEvidence{
				Kind:     "project_standards_markdown",
				ID:       project.ID,
				Summary:  "Project-scoped standards and workflow rules",
				Markdown: standards,
			})
		}
		if bindingFound {
			evidence = append(evidence, instructionsToolEvidence{
				Kind:    "project_template_binding",
				ID:      project.ID,
				Summary: fmt.Sprintf("Bound library %s revision %d with drift %s", binding.LibraryID, binding.BoundRevision, fallbackText(strings.TrimSpace(binding.DriftStatus), "current")),
			})
		}
	}

	gaps := make([]string, 0, 2)
	if strings.TrimSpace(project.Metadata.StandardsMarkdown) == "" {
		gaps = append(gaps, "This project has no standards_markdown yet, so only generic project workflow guidance is available.")
	}
	if !bindingFound {
		gaps = append(gaps, "This project has no active template binding, so only project-local policy and kind allowlist rules apply.")
	}

	return instructionsExplainResult{
		Summary: fmt.Sprintf("Project %q is explainable from project metadata, allowed kinds, and the active template binding.", project.Name),
		ResolvedScope: instructionsToolResolvedScope{
			ProjectID: project.ID,
		},
		Explanation: instructionsToolExplanation{
			Title:             project.Name,
			Overview:          fmt.Sprintf("Project %q is a %q project%s.", project.Name, project.Kind, bindingOverviewSuffix(binding, bindingFound)),
			WhyItApplies:      buildProjectWhyItApplies(project, binding, bindingFound),
			ScopedRules:       rules,
			WorkflowContract:  workflow,
			AgentExpectations: expectations,
			RelatedTools: []instructionsToolRelatedTool{
				{Tool: "till.project", Operation: "get_template_binding", Reason: "inspect active project binding"},
				{Tool: "till.project", Operation: "list_allowed_kinds", Reason: "inspect project kind allowlist"},
				{Tool: "till.comment", Operation: "create|list", Reason: "coordinate inside the project thread or project-scoped nodes"},
				{Tool: "till.handoff", Operation: "create|list|update", Reason: "route explicit next-action work inside the project"},
			},
			Evidence: evidence,
			Gaps:     gaps,
		},
	}, nil
}

// explainTemplateInstructions resolves one template-library or project-binding explanation.
func explainTemplateInstructions(ctx context.Context, services instructionsExplainServices, req instructionsExplainRequest) (instructionsExplainResult, error) {
	projectID := strings.TrimSpace(req.ProjectID)
	libraryID := strings.TrimSpace(req.TemplateLibraryID)

	var binding domain.ProjectTemplateBinding
	var bindingFound bool
	var err error
	if libraryID == "" && projectID != "" {
		binding, bindingFound, err = loadProjectBinding(ctx, services.templates, projectID)
		if err != nil {
			return instructionsExplainResult{}, err
		}
		if !bindingFound {
			return instructionsExplainResult{}, fmt.Errorf("not found: project %q has no active template binding", projectID)
		}
		libraryID = binding.LibraryID
	}
	if libraryID == "" {
		return instructionsExplainResult{}, fmt.Errorf(`invalid_request: focus "template" requires template_library_id or project_id with an active binding`)
	}

	library, err := loadTemplateLibraryForExplanation(ctx, services.templates, libraryID, binding, bindingFound)
	if err != nil {
		return instructionsExplainResult{}, err
	}

	actorKinds := summarizeTemplateActorKinds(library)
	blockerRules := summarizeTemplateBlockers(library)
	rules := []string{
		fmt.Sprintf("Template library %q is the source of generated workflow contracts for its matching project/node scopes.", library.ID),
		fmt.Sprintf("This library defines %d node template(s) and %d child rule(s).", len(library.NodeTemplates), countTemplateChildRules(library)),
	}
	if len(actorKinds) > 0 {
		rules = append(rules, fmt.Sprintf("Responsible actor kinds referenced by this library: %s.", strings.Join(actorKinds, ", ")))
	}
	if len(blockerRules) > 0 {
		rules = append(rules, blockerRules...)
	}

	workflow := []string{
		"Use till.project(operation=bind_template) or the TUI project edit flow to bind or rebind this library explicitly.",
		"Use till.project(operation=preview_template_reapply) before adopting newer template revisions into an existing project.",
	}
	if bindingFound {
		workflow = append(workflow, fmt.Sprintf("Project binding drift for this library is currently %q.", fallbackText(strings.TrimSpace(binding.DriftStatus), "current")))
	}

	expectations := []string{
		"Agents should treat node-template descriptions, child rules, and actor-kind gates as the source of generated workflow expectations.",
		"Project-local standards may add stricter rules on top of these template defaults.",
	}

	evidence := make([]instructionsToolEvidence, 0, 4)
	if req.IncludeEvidence {
		if desc := strings.TrimSpace(library.Description); desc != "" {
			evidence = append(evidence, instructionsToolEvidence{
				Kind:     "template_library_description",
				ID:       library.ID,
				Summary:  "Template-library description",
				Markdown: desc,
			})
		}
		for _, nodeTemplate := range library.NodeTemplates {
			if desc := strings.TrimSpace(nodeTemplate.DescriptionMarkdown); desc != "" {
				evidence = append(evidence, instructionsToolEvidence{
					Kind:     "node_template_description",
					ID:       nodeTemplate.ID,
					Summary:  fmt.Sprintf("Node template %s for kind %s", nodeTemplate.DisplayName, nodeTemplate.NodeKindID),
					Markdown: desc,
				})
			}
		}
	}

	gaps := make([]string, 0, 2)
	if strings.TrimSpace(library.Description) == "" {
		gaps = append(gaps, "This template library has no top-level description markdown yet.")
	}
	if !templateLibraryHasDescriptionMarkdown(library) {
		gaps = append(gaps, "This template library has no node-template description markdown yet, so rule explanations rely mostly on child-rule structure.")
	}

	resolved := instructionsToolResolvedScope{
		TemplateLibraryID: library.ID,
	}
	if projectID != "" {
		resolved.ProjectID = projectID
	}
	return instructionsExplainResult{
		Summary:       fmt.Sprintf("Template library %q is explainable from its node templates, child rules, and any active project binding.", library.ID),
		ResolvedScope: resolved,
		Explanation: instructionsToolExplanation{
			Title:             fallbackText(strings.TrimSpace(library.Name), library.ID),
			Overview:          fmt.Sprintf("Template library %q is a %q-scoped library at revision %d with %d node template(s).", library.ID, library.Scope, library.Revision, len(library.NodeTemplates)),
			WhyItApplies:      buildTemplateWhyItApplies(library, binding, bindingFound, projectID),
			ScopedRules:       rules,
			WorkflowContract:  workflow,
			AgentExpectations: expectations,
			RelatedTools: []instructionsToolRelatedTool{
				{Tool: "till.template", Operation: "get", Reason: "inspect raw template library data"},
				{Tool: "till.project", Operation: "get_template_binding", Reason: "inspect the active project binding for this library"},
				{Tool: "till.project", Operation: "preview_template_reapply", Reason: "review drift and migration impact before rebinding"},
			},
			Evidence: evidence,
			Gaps:     gaps,
		},
	}, nil
}

// explainKindInstructions resolves one kind-scoped explanation, optionally narrowed by project or template context.
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
	libraryID := strings.TrimSpace(req.TemplateLibraryID)
	var library domain.TemplateLibrary
	var libraryFound bool
	if libraryID == "" && projectID != "" {
		binding, bindingFound, bindingErr := loadProjectBinding(ctx, services.templates, projectID)
		if bindingErr != nil {
			return instructionsExplainResult{}, bindingErr
		}
		if bindingFound {
			libraryID = binding.LibraryID
		}
	}
	if libraryID != "" {
		library, err = loadTemplateLibraryForExplanation(ctx, services.templates, libraryID, domain.ProjectTemplateBinding{}, false)
		if err != nil {
			return instructionsExplainResult{}, err
		}
		libraryFound = true
	}

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

	if libraryFound {
		templateRefs := summarizeKindTemplateReferences(library, kind.ID)
		rules = append(rules, templateRefs...)
	}

	gaps := make([]string, 0, 2)
	if strings.TrimSpace(kind.DescriptionMarkdown) == "" {
		gaps = append(gaps, fmt.Sprintf("Kind %q has no description_markdown yet.", kind.ID))
	}
	if libraryID == "" && projectID == "" {
		gaps = append(gaps, "No project or template context was supplied, so template-specific usage for this kind may be incomplete.")
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
		ProjectID:         projectID,
		TemplateLibraryID: libraryID,
		KindID:            string(kind.ID),
		KindDisplayName:   strings.TrimSpace(kind.DisplayName),
	}
	return instructionsExplainResult{
		Summary:       fmt.Sprintf("Kind %q is explainable from the kind catalog%s.", kind.ID, kindContextSuffix(projectID, libraryID)),
		ResolvedScope: resolved,
		Explanation: instructionsToolExplanation{
			Title:    fallbackText(strings.TrimSpace(kind.DisplayName), string(kind.ID)),
			Overview: fmt.Sprintf("Kind %q is a reusable catalog definition for %s scope(s).", kind.ID, joinKindScopes(kind.AppliesTo)),
			WhyItApplies: []string{
				"Kind definitions define reusable scope semantics, parent constraints, optional payload schema, and baseline template defaults.",
				"Project allowlists and template libraries then narrow how this kind should be used in one concrete workflow.",
			},
			ScopedRules:      rules,
			WorkflowContract: []string{"Use till.kind for catalog-level meaning and till.template or till.project for scoped usage/binding context."},
			AgentExpectations: []string{
				"Do not infer scoped workflow rules from kind catalog data alone when a project binding or node contract is available.",
			},
			RelatedTools: []instructionsToolRelatedTool{
				{Tool: "till.kind", Operation: "list", Reason: "inspect catalog definitions"},
				{Tool: "till.project", Operation: "list_allowed_kinds", Reason: "check project allowlist context"},
				{Tool: "till.template", Operation: "get", Reason: "inspect template-library references for this kind"},
			},
			Evidence: evidence,
			Gaps:     gaps,
		},
	}, nil
}

// explainNodeInstructions resolves one branch|phase|task|subtask explanation from node lineage and stored contract state.
func explainNodeInstructions(ctx context.Context, services instructionsExplainServices, req instructionsExplainRequest) (instructionsExplainResult, error) {
	nodeID := strings.TrimSpace(req.NodeID)
	if nodeID == "" {
		return instructionsExplainResult{}, fmt.Errorf(`invalid_request: focus "node" requires node_id`)
	}
	if services.tasks == nil {
		return instructionsExplainResult{}, fmt.Errorf("not found: task service is unavailable for node explanation")
	}
	task, err := services.tasks.GetTask(ctx, nodeID)
	if err != nil {
		return instructionsExplainResult{}, fmt.Errorf("get node %q: %w", nodeID, err)
	}
	project, err := findProjectByID(ctx, services.projects, strings.TrimSpace(task.ProjectID))
	if err != nil {
		return instructionsExplainResult{}, err
	}
	lineageTasks, err := loadTaskLineage(ctx, services.tasks, task)
	if err != nil {
		return instructionsExplainResult{}, err
	}
	lineage := summarizeTaskLineage(lineageTasks)

	kind, kindFound, err := tryFindKindByID(ctx, services.kinds, domain.KindID(task.Kind))
	if err != nil {
		return instructionsExplainResult{}, err
	}
	binding, bindingFound, err := loadProjectBinding(ctx, services.templates, task.ProjectID)
	if err != nil {
		return instructionsExplainResult{}, err
	}
	snapshot, snapshotFound, err := loadNodeContractSnapshot(ctx, services.templates, nodeID)
	if err != nil {
		return instructionsExplainResult{}, err
	}

	var library domain.TemplateLibrary
	var libraryFound bool
	if snapshotFound && strings.TrimSpace(snapshot.SourceLibraryID) != "" {
		library, err = loadTemplateLibraryForExplanation(ctx, services.templates, snapshot.SourceLibraryID, binding, bindingFound)
		if err == nil {
			libraryFound = true
		}
	} else if bindingFound {
		library, err = loadTemplateLibraryForExplanation(ctx, services.templates, binding.LibraryID, binding, bindingFound)
		if err == nil {
			libraryFound = true
		}
	}

	rules := collectNodeScopedRules(project, task)
	workflow := collectNodeWorkflowContract(task, kind, kindFound, snapshot, snapshotFound, binding, bindingFound)
	expectations := collectNodeAgentExpectations(task, snapshot, snapshotFound)
	why := collectNodeWhyItApplies(project, task, kind, kindFound, snapshot, snapshotFound, library, libraryFound)
	evidence := collectNodeEvidence(project, task, snapshot, snapshotFound, req.IncludeEvidence)
	gaps := collectNodeGaps(project, task, snapshot, snapshotFound, kindFound, bindingFound)

	resolved := instructionsToolResolvedScope{
		ProjectID:         project.ID,
		TemplateLibraryID: firstNonEmptyString(strings.TrimSpace(snapshot.SourceLibraryID), binding.LibraryID),
		KindID:            strings.TrimSpace(string(task.Kind)),
		NodeID:            task.ID,
		NodeScopeType:     strings.TrimSpace(string(task.Scope)),
		NodeTitle:         strings.TrimSpace(task.Title),
		Lineage:           lineage,
	}
	if kindFound {
		resolved.KindDisplayName = strings.TrimSpace(kind.DisplayName)
	}

	return instructionsExplainResult{
		Summary:       fmt.Sprintf("%s %q is explainable from node lineage, project standards, and any stored template contract.", strings.Title(string(task.Scope)), task.Title),
		ResolvedScope: resolved,
		Explanation: instructionsToolExplanation{
			Title:             strings.TrimSpace(task.Title),
			Overview:          fmt.Sprintf("%s %q belongs to project %q%s.", strings.Title(string(task.Scope)), task.Title, project.Name, nodeContractOverviewSuffix(snapshot, snapshotFound)),
			WhyItApplies:      why,
			ScopedRules:       rules,
			WorkflowContract:  workflow,
			AgentExpectations: expectations,
			RelatedTools: []instructionsToolRelatedTool{
				{Tool: "till.plan_item", Operation: "get|update|move_state", Reason: "inspect or advance the node lifecycle"},
				{Tool: "till.comment", Operation: "create|list", Reason: "coordinate on the node thread"},
				{Tool: "till.handoff", Operation: "create|list|update", Reason: "route explicit next-action work for the node"},
				{Tool: "till.template", Operation: "get_node_contract", Reason: "inspect the raw stored node-contract snapshot"},
				{Tool: "till.project", Operation: "get_template_binding", Reason: "inspect the active project binding behind this node"},
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

// loadProjectBinding loads one project binding when it exists.
func loadProjectBinding(ctx context.Context, service common.TemplateLibraryService, projectID string) (domain.ProjectTemplateBinding, bool, error) {
	if service == nil || strings.TrimSpace(projectID) == "" {
		return domain.ProjectTemplateBinding{}, false, nil
	}
	binding, err := service.GetProjectTemplateBinding(ctx, strings.TrimSpace(projectID))
	if err != nil {
		if errors.Is(err, common.ErrNotFound) {
			return domain.ProjectTemplateBinding{}, false, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return domain.ProjectTemplateBinding{}, false, nil
		}
		return domain.ProjectTemplateBinding{}, false, fmt.Errorf("get project template binding: %w", err)
	}
	return binding, true, nil
}

// loadNodeContractSnapshot loads one stored node-contract snapshot when it exists.
func loadNodeContractSnapshot(ctx context.Context, service common.TemplateLibraryService, nodeID string) (domain.NodeContractSnapshot, bool, error) {
	if service == nil || strings.TrimSpace(nodeID) == "" {
		return domain.NodeContractSnapshot{}, false, nil
	}
	snapshot, err := service.GetNodeContractSnapshot(ctx, strings.TrimSpace(nodeID))
	if err != nil {
		if errors.Is(err, common.ErrNotFound) {
			return domain.NodeContractSnapshot{}, false, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return domain.NodeContractSnapshot{}, false, nil
		}
		return domain.NodeContractSnapshot{}, false, fmt.Errorf("get node contract snapshot: %w", err)
	}
	return snapshot, true, nil
}

// loadTemplateLibraryForExplanation loads one template library, preferring a bound snapshot when it matches.
func loadTemplateLibraryForExplanation(ctx context.Context, service common.TemplateLibraryService, libraryID string, binding domain.ProjectTemplateBinding, bindingFound bool) (domain.TemplateLibrary, error) {
	libraryID = strings.TrimSpace(libraryID)
	if libraryID == "" {
		return domain.TemplateLibrary{}, fmt.Errorf("not found: template library id is required")
	}
	if bindingFound && binding.BoundLibrarySnapshot != nil && strings.TrimSpace(binding.BoundLibrarySnapshot.ID) == libraryID {
		return *binding.BoundLibrarySnapshot, nil
	}
	if service == nil {
		return domain.TemplateLibrary{}, fmt.Errorf("not found: template library service is unavailable")
	}
	library, err := service.GetTemplateLibrary(ctx, libraryID)
	if err != nil {
		return domain.TemplateLibrary{}, fmt.Errorf("get template library %q: %w", libraryID, err)
	}
	return library, nil
}

// loadTaskLineage loads the root-to-leaf lineage for one node using repeated GetTask reads.
func loadTaskLineage(ctx context.Context, service common.TaskService, leaf domain.Task) ([]domain.Task, error) {
	if service == nil {
		return nil, fmt.Errorf("not found: task service is unavailable")
	}
	reversed := make([]domain.Task, 0, 8)
	current := leaf
	seen := map[string]struct{}{}
	for {
		currentID := strings.TrimSpace(current.ID)
		if currentID == "" {
			break
		}
		if _, ok := seen[currentID]; ok {
			return nil, fmt.Errorf("invalid task lineage: cycle at %q", currentID)
		}
		seen[currentID] = struct{}{}
		reversed = append(reversed, current)
		parentID := strings.TrimSpace(current.ParentID)
		if parentID == "" {
			break
		}
		parent, err := service.GetTask(ctx, parentID)
		if err != nil {
			return nil, fmt.Errorf("get parent task %q: %w", parentID, err)
		}
		current = parent
	}
	lineage := make([]domain.Task, 0, len(reversed))
	for idx := len(reversed) - 1; idx >= 0; idx-- {
		lineage = append(lineage, reversed[idx])
	}
	return lineage, nil
}

// summarizeTaskLineage converts one task lineage into readable scope markers.
func summarizeTaskLineage(lineage []domain.Task) []string {
	out := make([]string, 0, len(lineage))
	for _, task := range lineage {
		label := fmt.Sprintf("%s:%s", task.Scope, strings.TrimSpace(task.Title))
		out = append(out, label)
	}
	return out
}

// collectNodeScopedRules lifts scoped rule sources from project and node metadata.
func collectNodeScopedRules(project domain.Project, task domain.Task) []string {
	rules := make([]string, 0, 12)
	if standards := strings.TrimSpace(project.Metadata.StandardsMarkdown); standards != "" {
		rules = append(rules, "Project standards_markdown applies to this node and should be treated as local execution policy.")
	}
	if desc := strings.TrimSpace(task.Description); desc != "" {
		rules = append(rules, "The node description contains scoped workflow or implementation context for this exact branch, phase, task, or subtask.")
	}
	if objective := strings.TrimSpace(task.Metadata.Objective); objective != "" {
		rules = append(rules, "Objective: "+objective)
	}
	if notes := strings.TrimSpace(task.Metadata.ImplementationNotesAgent); notes != "" {
		rules = append(rules, "Agent notes: "+notes)
	}
	if acceptance := strings.TrimSpace(task.Metadata.AcceptanceCriteria); acceptance != "" {
		rules = append(rules, "Acceptance criteria: "+acceptance)
	}
	if dod := strings.TrimSpace(task.Metadata.DefinitionOfDone); dod != "" {
		rules = append(rules, "Definition of done: "+dod)
	}
	if validation := strings.TrimSpace(task.Metadata.ValidationPlan); validation != "" {
		rules = append(rules, "Validation plan: "+validation)
	}
	if len(task.Metadata.CommandSnippets) > 0 {
		rules = append(rules, fmt.Sprintf("Command snippets are attached to this node: %s.", strings.Join(task.Metadata.CommandSnippets, ", ")))
	}
	return rules
}

// collectNodeWorkflowContract lifts kind/template contract facts that affect how one node should move.
func collectNodeWorkflowContract(task domain.Task, kind domain.KindDefinition, kindFound bool, snapshot domain.NodeContractSnapshot, snapshotFound bool, binding domain.ProjectTemplateBinding, bindingFound bool) []string {
	contract := make([]string, 0, 10)
	contract = append(contract, fmt.Sprintf("Node scope is %q and kind is %q.", task.Scope, task.Kind))
	if kindFound {
		contract = append(contract, fmt.Sprintf("Catalog kind %q applies to: %s.", kind.ID, joinKindScopes(kind.AppliesTo)))
	}
	if bindingFound {
		contract = append(contract, fmt.Sprintf("Project binding is %q revision %d.", binding.LibraryID, binding.BoundRevision))
	}
	if snapshotFound {
		contract = append(contract,
			fmt.Sprintf("Responsible actor kind: %s.", snapshot.ResponsibleActorKind),
			fmt.Sprintf("Editable by: %s.", joinTemplateActorKinds(snapshot.EditableByActorKinds)),
			fmt.Sprintf("Completable by: %s.", joinTemplateActorKinds(snapshot.CompletableByActorKinds)),
			fmt.Sprintf("Required for parent done: %t.", snapshot.RequiredForParentDone),
			fmt.Sprintf("Required for containing done: %t.", snapshot.RequiredForContainingDone),
		)
	}
	if !snapshotFound {
		contract = append(contract, "No stored node-contract snapshot exists for this node, so only project, kind, and node-local metadata rules apply.")
	}
	return contract
}

// collectNodeAgentExpectations summarizes role expectations for one node.
func collectNodeAgentExpectations(task domain.Task, snapshot domain.NodeContractSnapshot, snapshotFound bool) []string {
	expectations := []string{
		"Use till.comment for shared status and evidence on this node.",
		"Use till.handoff when the next action belongs to another actor or role.",
	}
	if snapshotFound {
		expectations = append(expectations, fmt.Sprintf("The primary responsible actor for this node is %s.", snapshot.ResponsibleActorKind))
	} else {
		expectations = append(expectations, fmt.Sprintf("This %s node has no stored generated contract, so rely on project policy plus the node's own metadata.", task.Scope))
	}
	return expectations
}

// collectNodeWhyItApplies explains why the returned rules apply to one node.
func collectNodeWhyItApplies(project domain.Project, task domain.Task, kind domain.KindDefinition, kindFound bool, snapshot domain.NodeContractSnapshot, snapshotFound bool, library domain.TemplateLibrary, libraryFound bool) []string {
	why := []string{
		fmt.Sprintf("This node belongs to project %q, so project-scoped standards and allowed kinds apply.", project.Name),
	}
	if kindFound {
		why = append(why, fmt.Sprintf("Kind %q defines the base semantics for this node scope.", kind.ID))
	}
	if snapshotFound {
		why = append(why, fmt.Sprintf("A stored node-contract snapshot records generated workflow rules for this node from library %q.", snapshot.SourceLibraryID))
		if libraryFound {
			if nodeTemplate, childRule, ok := findNodeTemplateAndChildRule(library, snapshot.SourceNodeTemplateID, snapshot.SourceChildRuleID); ok {
				why = append(why, fmt.Sprintf("The realized contract came from node template %q and child rule %q.", fallbackText(strings.TrimSpace(nodeTemplate.DisplayName), nodeTemplate.ID), childRule.ID))
			}
		}
	} else {
		why = append(why, "No stored node-contract snapshot exists, so this explanation falls back to project policy, kind semantics, and node-local metadata.")
	}
	return why
}

// collectNodeEvidence returns concrete policy evidence for one node explanation.
func collectNodeEvidence(project domain.Project, task domain.Task, snapshot domain.NodeContractSnapshot, snapshotFound bool, include bool) []instructionsToolEvidence {
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
	appendMarkdownEvidence("node_description", task.ID, "Node description", task.Description)
	appendMarkdownEvidence("node_objective", task.ID, "Node objective", task.Metadata.Objective)
	appendMarkdownEvidence("node_agent_notes", task.ID, "Node implementation notes for agents", task.Metadata.ImplementationNotesAgent)
	appendMarkdownEvidence("node_acceptance_criteria", task.ID, "Node acceptance criteria", task.Metadata.AcceptanceCriteria)
	appendMarkdownEvidence("node_definition_of_done", task.ID, "Node definition of done", task.Metadata.DefinitionOfDone)
	appendMarkdownEvidence("node_validation_plan", task.ID, "Node validation plan", task.Metadata.ValidationPlan)
	if snapshotFound {
		evidence = append(evidence, instructionsToolEvidence{
			Kind:    "node_contract_snapshot",
			ID:      task.ID,
			Summary: fmt.Sprintf("Generated contract from library %s, template %s, child rule %s", snapshot.SourceLibraryID, snapshot.SourceNodeTemplateID, snapshot.SourceChildRuleID),
		})
	}
	return evidence
}

// collectNodeGaps reports which rule sources are missing for one node explanation.
func collectNodeGaps(project domain.Project, task domain.Task, snapshot domain.NodeContractSnapshot, snapshotFound bool, kindFound bool, bindingFound bool) []string {
	gaps := make([]string, 0, 6)
	if strings.TrimSpace(project.Metadata.StandardsMarkdown) == "" {
		gaps = append(gaps, "Project standards_markdown is empty, so project-local execution rules are not explicit yet.")
	}
	if strings.TrimSpace(task.Metadata.Objective) == "" &&
		strings.TrimSpace(task.Metadata.AcceptanceCriteria) == "" &&
		strings.TrimSpace(task.Metadata.DefinitionOfDone) == "" &&
		strings.TrimSpace(task.Metadata.ValidationPlan) == "" {
		gaps = append(gaps, "This node has little or no scoped task metadata yet, so only generic workflow guidance is available.")
	}
	if !snapshotFound {
		gaps = append(gaps, "This node has no stored generated contract snapshot.")
	}
	if !kindFound {
		gaps = append(gaps, "Kind-catalog detail for this node could not be resolved.")
	}
	if !bindingFound {
		gaps = append(gaps, "The project has no active template binding, so there is no project-level template contract to explain.")
	}
	return gaps
}

// summarizeTemplateActorKinds returns the distinct responsible actor kinds mentioned by one library.
func summarizeTemplateActorKinds(library domain.TemplateLibrary) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, nodeTemplate := range library.NodeTemplates {
		for _, childRule := range nodeTemplate.ChildRules {
			role := strings.TrimSpace(string(childRule.ResponsibleActorKind))
			if role == "" {
				continue
			}
			if _, ok := seen[role]; ok {
				continue
			}
			seen[role] = struct{}{}
			out = append(out, role)
		}
	}
	slices.Sort(out)
	return out
}

// summarizeTemplateBlockers returns readable blocker-rule summaries for one library.
func summarizeTemplateBlockers(library domain.TemplateLibrary) []string {
	out := make([]string, 0)
	for _, nodeTemplate := range library.NodeTemplates {
		for _, childRule := range nodeTemplate.ChildRules {
			if childRule.RequiredForParentDone {
				out = append(out, fmt.Sprintf("Child rule %q blocks parent done until its generated child is complete.", childRule.ID))
			}
			if childRule.RequiredForContainingDone {
				out = append(out, fmt.Sprintf("Child rule %q blocks containing-scope completion until its generated child is complete.", childRule.ID))
			}
		}
	}
	return out
}

// countTemplateChildRules counts the child rules across all node templates in one library.
func countTemplateChildRules(library domain.TemplateLibrary) int {
	total := 0
	for _, nodeTemplate := range library.NodeTemplates {
		total += len(nodeTemplate.ChildRules)
	}
	return total
}

// templateLibraryHasDescriptionMarkdown reports whether a library or any node template has description markdown.
func templateLibraryHasDescriptionMarkdown(library domain.TemplateLibrary) bool {
	if strings.TrimSpace(library.Description) != "" {
		return true
	}
	for _, nodeTemplate := range library.NodeTemplates {
		if strings.TrimSpace(nodeTemplate.DescriptionMarkdown) != "" {
			return true
		}
	}
	return false
}

// summarizeKindTemplateReferences returns readable references between one kind and one template library.
func summarizeKindTemplateReferences(library domain.TemplateLibrary, kindID domain.KindID) []string {
	out := make([]string, 0)
	kindID = domain.NormalizeKindID(kindID)
	for _, nodeTemplate := range library.NodeTemplates {
		if domain.NormalizeKindID(nodeTemplate.NodeKindID) == kindID {
			out = append(out, fmt.Sprintf("Template library %q uses this kind as node template %q at %q scope.", library.ID, fallbackText(strings.TrimSpace(nodeTemplate.DisplayName), nodeTemplate.ID), nodeTemplate.ScopeLevel))
		}
		for _, childRule := range nodeTemplate.ChildRules {
			if domain.NormalizeKindID(childRule.ChildKindID) == kindID {
				out = append(out, fmt.Sprintf("Template library %q generates this kind through child rule %q under node template %q.", library.ID, childRule.ID, fallbackText(strings.TrimSpace(nodeTemplate.DisplayName), nodeTemplate.ID)))
			}
		}
	}
	if len(out) == 0 {
		out = append(out, fmt.Sprintf("No node template or child rule in library %q currently references kind %q.", library.ID, kindID))
	}
	return out
}

// findNodeTemplateAndChildRule resolves one template node/rule pair from a stored snapshot reference.
func findNodeTemplateAndChildRule(library domain.TemplateLibrary, nodeTemplateID, childRuleID string) (domain.NodeTemplate, domain.TemplateChildRule, bool) {
	nodeTemplateID = strings.TrimSpace(nodeTemplateID)
	childRuleID = strings.TrimSpace(childRuleID)
	for _, nodeTemplate := range library.NodeTemplates {
		if strings.TrimSpace(nodeTemplate.ID) != nodeTemplateID {
			continue
		}
		if childRuleID == "" {
			return nodeTemplate, domain.TemplateChildRule{}, true
		}
		for _, childRule := range nodeTemplate.ChildRules {
			if strings.TrimSpace(childRule.ID) == childRuleID {
				return nodeTemplate, childRule, true
			}
		}
	}
	return domain.NodeTemplate{}, domain.TemplateChildRule{}, false
}

// buildProjectWhyItApplies returns one project-scoped explanation list.
func buildProjectWhyItApplies(project domain.Project, binding domain.ProjectTemplateBinding, bindingFound bool) []string {
	why := []string{
		fmt.Sprintf("Project metadata and standards_markdown are the canonical project-local rule source for %q.", project.Name),
		fmt.Sprintf("Project kind %q defines the baseline interpretation for project setup and allowed workflow shape.", project.Kind),
	}
	if bindingFound {
		why = append(why, fmt.Sprintf("The active template binding to %q adds generated workflow contracts on top of project-local policy.", binding.LibraryID))
	}
	return why
}

// buildTemplateWhyItApplies returns one template-scoped explanation list.
func buildTemplateWhyItApplies(library domain.TemplateLibrary, binding domain.ProjectTemplateBinding, bindingFound bool, projectID string) []string {
	why := []string{
		fmt.Sprintf("Template library %q defines reusable workflow contracts through node templates and child rules.", library.ID),
	}
	if projectID != "" && bindingFound {
		why = append(why, fmt.Sprintf("Project %q is currently bound to this library, so its generated-node rules apply to future project work.", projectID))
	}
	return why
}

// bindingOverviewSuffix returns one project binding suffix for project overview text.
func bindingOverviewSuffix(binding domain.ProjectTemplateBinding, found bool) string {
	if !found {
		return ""
	}
	return fmt.Sprintf(" with active template binding %q", binding.LibraryID)
}

// nodeContractOverviewSuffix returns one node-contract overview suffix.
func nodeContractOverviewSuffix(snapshot domain.NodeContractSnapshot, found bool) string {
	if !found {
		return ""
	}
	return fmt.Sprintf(" and a stored generated contract from %q", snapshot.SourceLibraryID)
}

// kindContextSuffix returns one scope-context suffix for kind explanations.
func kindContextSuffix(projectID, libraryID string) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(projectID) != "" {
		parts = append(parts, fmt.Sprintf(" in project %q", projectID))
	}
	if strings.TrimSpace(libraryID) != "" {
		parts = append(parts, fmt.Sprintf(" against template %q", libraryID))
	}
	return strings.Join(parts, "")
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

// joinTemplateActorKinds returns one readable actor-kind list.
func joinTemplateActorKinds(kinds []domain.TemplateActorKind) string {
	parts := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		value := strings.TrimSpace(string(kind))
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
