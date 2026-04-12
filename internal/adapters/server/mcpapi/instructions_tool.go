package mcpapi

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tillsyndocs "github.com/hylla/tillsyn"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// instructionsToolMode identifies the response shape requested from till.get_instructions.
type instructionsToolMode string

// instructionsToolMode values identify the supported instruction response modes.
const (
	instructionsToolModeDocs    instructionsToolMode = "docs"
	instructionsToolModeExplain instructionsToolMode = "explain"
	instructionsToolModeHybrid  instructionsToolMode = "hybrid"
)

// instructionsToolFocus identifies the scoped explanation target for till.get_instructions.
type instructionsToolFocus string

// instructionsToolFocus values identify the supported explanation targets.
const (
	instructionsToolFocusTopic    instructionsToolFocus = "topic"
	instructionsToolFocusProject  instructionsToolFocus = "project"
	instructionsToolFocusTemplate instructionsToolFocus = "template"
	instructionsToolFocusKind     instructionsToolFocus = "kind"
	instructionsToolFocusNode     instructionsToolFocus = "node"
)

// instructionsToolRequest stores one till.get_instructions request after MCP argument normalization.
type instructionsToolRequest struct {
	Mode                   string
	Focus                  string
	Topic                  string
	ProjectID              string
	TemplateLibraryID      string
	KindID                 string
	NodeID                 string
	IncludeEvidence        bool
	DocNames               []string
	IncludeMarkdown        bool
	IncludeRecommendations bool
	MaxCharsPerDoc         int
}

// instructionsToolDoc stores one filtered or truncated embedded markdown payload.
type instructionsToolDoc struct {
	FileName      string `json:"file_name"`
	Path          string `json:"path"`
	TotalChars    int    `json:"total_chars"`
	ReturnedChars int    `json:"returned_chars"`
	Truncated     bool   `json:"truncated"`
	Markdown      string `json:"markdown,omitempty"`
}

// instructionsToolResolvedScope stores the runtime scope resolved for one scoped explanation.
type instructionsToolResolvedScope struct {
	ProjectID         string   `json:"project_id,omitempty"`
	TemplateLibraryID string   `json:"template_library_id,omitempty"`
	KindID            string   `json:"kind_id,omitempty"`
	KindDisplayName   string   `json:"kind_display_name,omitempty"`
	NodeID            string   `json:"node_id,omitempty"`
	NodeScopeType     string   `json:"node_scope_type,omitempty"`
	NodeTitle         string   `json:"node_title,omitempty"`
	Lineage           []string `json:"lineage,omitempty"`
}

// instructionsToolRelatedTool stores one follow-up tool recommendation.
type instructionsToolRelatedTool struct {
	Tool      string `json:"tool"`
	Operation string `json:"operation,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// instructionsToolEvidence stores one concrete policy or runtime source used by the explainer.
type instructionsToolEvidence struct {
	Kind     string `json:"kind"`
	ID       string `json:"id,omitempty"`
	Summary  string `json:"summary"`
	Markdown string `json:"markdown,omitempty"`
}

// instructionsToolExplanation stores one explanation-first scoped guidance payload.
type instructionsToolExplanation struct {
	Title             string                        `json:"title"`
	Overview          string                        `json:"overview"`
	WhyItApplies      []string                      `json:"why_it_applies,omitempty"`
	ScopedRules       []string                      `json:"scoped_rules,omitempty"`
	WorkflowContract  []string                      `json:"workflow_contract,omitempty"`
	AgentExpectations []string                      `json:"agent_expectations,omitempty"`
	RelatedTools      []instructionsToolRelatedTool `json:"related_tools,omitempty"`
	Evidence          []instructionsToolEvidence    `json:"evidence,omitempty"`
	Gaps              []string                      `json:"gaps,omitempty"`
}

// instructionsToolResponse stores one till.get_instructions tool payload.
type instructionsToolResponse struct {
	Summary                  string                         `json:"summary"`
	Mode                     string                         `json:"mode,omitempty"`
	Focus                    string                         `json:"focus,omitempty"`
	Topic                    string                         `json:"topic,omitempty"`
	ResolvedScope            *instructionsToolResolvedScope `json:"resolved_scope,omitempty"`
	Explanation              *instructionsToolExplanation   `json:"explanation,omitempty"`
	RecommendedAgentSettings []string                       `json:"recommended_agent_settings,omitempty"`
	MDFileGuidance           map[string][]string            `json:"md_file_guidance,omitempty"`
	AvailableDocs            []string                       `json:"available_docs"`
	Docs                     []instructionsToolDoc          `json:"docs"`
}

// registerInstructionsTool registers the embedded-doc and scoped explanation tool.
func registerInstructionsTool(srv *mcpserver.MCPServer, services instructionsExplainServices) {
	srv.AddTool(
		mcp.NewTool(
			"till.get_instructions",
			mcp.WithDescription("Return embedded markdown docs plus scoped guidance about till workflow policy, project rules, template contracts, kind usage, and concrete node expectations."),
			mcp.WithString("mode", mcp.Description("Optional response mode. Defaults to docs for plain doc lookups and explain for scoped runtime lookups."), mcp.Enum("docs", "explain", "hybrid")),
			mcp.WithString("focus", mcp.Description("Optional scoped explanation focus. Use topic|project|template|kind|node when you want instructions tied to one concrete runtime scope."), mcp.Enum("topic", "project", "template", "kind", "node")),
			mcp.WithString("topic", mcp.Description("Optional topic focus (for example: dogfooding, agents, workflows, coordination, auth, templates, recovery)")),
			mcp.WithString("project_id", mcp.Description("Optional project identifier for project-scoped explanation and policy resolution")),
			mcp.WithString("template_library_id", mcp.Description("Optional template library identifier for template-scoped explanation")),
			mcp.WithString("kind_id", mcp.Description("Optional kind identifier for kind-scoped explanation")),
			mcp.WithString("node_id", mcp.Description("Optional work-item identifier for branch|phase|task|subtask explanation")),
			mcp.WithBoolean("include_evidence", mcp.Description("Include concrete runtime policy evidence such as standards markdown, task metadata, and node-contract source details when available")),
			mcp.WithArray("doc_names", mcp.Description("Optional markdown file-name filter list (for example: README.md, AGENTS.md)"), mcp.WithStringItems()),
			mcp.WithBoolean("include_markdown", mcp.Description("Include markdown content in docs payload (default true)")),
			mcp.WithBoolean("include_recommendations", mcp.Description("Include settings and md-file guidance recommendations (default true)")),
			mcp.WithNumber("max_chars_per_doc", mcp.Description("Optional per-doc markdown truncation limit in characters (0 = no truncation)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			maxChars := req.GetInt("max_chars_per_doc", 0)
			if maxChars < 0 {
				return mcp.NewToolResultError("invalid_request: max_chars_per_doc must be >= 0"), nil
			}

			out, err := buildInstructionsToolResponse(ctx, services, instructionsToolRequest{
				Mode:                   req.GetString("mode", ""),
				Focus:                  req.GetString("focus", ""),
				Topic:                  req.GetString("topic", ""),
				ProjectID:              req.GetString("project_id", ""),
				TemplateLibraryID:      req.GetString("template_library_id", ""),
				KindID:                 req.GetString("kind_id", ""),
				NodeID:                 req.GetString("node_id", ""),
				IncludeEvidence:        req.GetBool("include_evidence", false),
				DocNames:               req.GetStringSlice("doc_names", nil),
				IncludeMarkdown:        req.GetBool("include_markdown", true),
				IncludeRecommendations: req.GetBool("include_recommendations", true),
				MaxCharsPerDoc:         maxChars,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := mcp.NewToolResultJSON(out)
			if err != nil {
				return nil, fmt.Errorf("encode get_instructions result: %w", err)
			}
			return result, nil
		},
	)
}

// buildInstructionsToolResponse assembles one deterministic instructions payload.
func buildInstructionsToolResponse(ctx context.Context, services instructionsExplainServices, req instructionsToolRequest) (instructionsToolResponse, error) {
	docs, err := tillsyndocs.EmbeddedMarkdownDocuments()
	if err != nil {
		return instructionsToolResponse{}, fmt.Errorf("load embedded markdown docs: %w", err)
	}

	mode, focus, err := normalizeInstructionsToolModeAndFocus(req)
	if err != nil {
		return instructionsToolResponse{}, err
	}

	filter := map[string]struct{}{}
	for _, raw := range req.DocNames {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		filter[name] = struct{}{}
	}

	available := make([]string, 0, len(docs))
	filtered := make([]instructionsToolDoc, 0, len(docs))
	for _, doc := range docs {
		available = append(available, doc.FileName)
		if mode == instructionsToolModeExplain {
			continue
		}
		if len(filter) > 0 {
			if _, ok := filter[strings.ToLower(strings.TrimSpace(doc.FileName))]; !ok {
				continue
			}
		}
		payload := strings.TrimSpace(doc.Markdown)
		totalChars := len(payload)
		returned := payload
		truncated := false
		if req.MaxCharsPerDoc > 0 && len(returned) > req.MaxCharsPerDoc {
			returned = returned[:req.MaxCharsPerDoc]
			truncated = true
		}
		row := instructionsToolDoc{
			FileName:      doc.FileName,
			Path:          doc.Path,
			TotalChars:    totalChars,
			ReturnedChars: len(returned),
			Truncated:     truncated,
		}
		if req.IncludeMarkdown {
			row.Markdown = returned
		}
		filtered = append(filtered, row)
	}
	sort.Strings(available)

	out := instructionsToolResponse{
		Mode:          string(mode),
		Focus:         string(focus),
		Topic:         strings.TrimSpace(req.Topic),
		AvailableDocs: available,
		Docs:          filtered,
	}

	if mode != instructionsToolModeDocs {
		explanation, err := explainInstructionsScope(ctx, services, instructionsExplainRequest{
			Focus:             focus,
			Topic:             strings.TrimSpace(req.Topic),
			ProjectID:         strings.TrimSpace(req.ProjectID),
			TemplateLibraryID: strings.TrimSpace(req.TemplateLibraryID),
			KindID:            strings.TrimSpace(req.KindID),
			NodeID:            strings.TrimSpace(req.NodeID),
			IncludeEvidence:   req.IncludeEvidence,
		})
		if err != nil {
			return instructionsToolResponse{}, err
		}
		out.ResolvedScope = &explanation.ResolvedScope
		out.Explanation = &explanation.Explanation
		out.Summary = explanation.Summary
	} else {
		out.Summary = buildInstructionsDocsSummary(strings.TrimSpace(req.Topic))
	}

	if req.IncludeRecommendations {
		out.RecommendedAgentSettings = recommendedInstructionSettings()
		out.MDFileGuidance = recommendedMDFileGuidance()
	}
	return out, nil
}

// normalizeInstructionsToolModeAndFocus validates mode/focus combinations and fills deterministic defaults.
func normalizeInstructionsToolModeAndFocus(req instructionsToolRequest) (instructionsToolMode, instructionsToolFocus, error) {
	mode := instructionsToolMode(strings.TrimSpace(strings.ToLower(req.Mode)))
	focus := instructionsToolFocus(strings.TrimSpace(strings.ToLower(req.Focus)))
	hasSelectors := strings.TrimSpace(req.ProjectID) != "" ||
		strings.TrimSpace(req.TemplateLibraryID) != "" ||
		strings.TrimSpace(req.KindID) != "" ||
		strings.TrimSpace(req.NodeID) != ""

	if mode == "" {
		if hasSelectors || (focus != "" && focus != instructionsToolFocusTopic) {
			mode = instructionsToolModeExplain
		} else {
			mode = instructionsToolModeDocs
		}
	}
	switch mode {
	case instructionsToolModeDocs, instructionsToolModeExplain, instructionsToolModeHybrid:
	default:
		return "", "", fmt.Errorf("invalid_request: unsupported mode %q", req.Mode)
	}

	if focus == "" {
		switch {
		case strings.TrimSpace(req.NodeID) != "":
			focus = instructionsToolFocusNode
		case strings.TrimSpace(req.TemplateLibraryID) != "":
			focus = instructionsToolFocusTemplate
		case strings.TrimSpace(req.KindID) != "":
			focus = instructionsToolFocusKind
		case strings.TrimSpace(req.ProjectID) != "":
			focus = instructionsToolFocusProject
		default:
			focus = instructionsToolFocusTopic
		}
	}
	switch focus {
	case instructionsToolFocusTopic, instructionsToolFocusProject, instructionsToolFocusTemplate, instructionsToolFocusKind, instructionsToolFocusNode:
	default:
		return "", "", fmt.Errorf("invalid_request: unsupported focus %q", req.Focus)
	}

	if mode == instructionsToolModeDocs && focus != instructionsToolFocusTopic && hasSelectors {
		return "", "", fmt.Errorf("invalid_request: mode=docs cannot use project_id/template_library_id/kind_id/node_id without explain or hybrid mode")
	}
	return mode, focus, nil
}

// buildInstructionsDocsSummary returns the default summary for pure embedded-doc responses.
func buildInstructionsDocsSummary(topic string) string {
	summary := "Embedded instruction docs for till MCP dogfooding, coordination, notifications, scoped rules, and agent configuration guidance."
	if topic != "" {
		summary = fmt.Sprintf("Embedded instruction docs focused on %q for till MCP dogfooding, coordination, notifications, scoped rules, and agent configuration guidance.", topic)
	}
	return summary
}

// recommendedInstructionSettings returns recommended agent behavior settings for instruction-tool usage.
func recommendedInstructionSettings() []string {
	return []string{
		"Use till.get_instructions when instructions are missing, stale, or ambiguous; skip redundant calls when AGENTS.md/README guidance is already sufficient for the current step.",
		"Use focus plus project_id/template_library_id/kind_id/node_id when you need rules for one concrete project, template, branch, phase, task, or generated node instead of a generic docs-only answer.",
		"Use doc_names to scope context (for example README.md and AGENTS.md) instead of loading every doc on each step.",
		"Use include_markdown=false for quick inventory checks; enable it when drafting or validating policy text.",
		"Set max_chars_per_doc to keep responses bounded in long docs such as README.md or AGENTS.md.",
		"Keep active tasks, actions, blockers, comments, handoffs, and worklogs in Tillsyn itself; markdown docs are durable policy/documentation surfaces, not live execution ledgers.",
		"During active coordination runs, keep waitable till.attention_item/till.comment/till.handoff list calls open with wait_timeout so they wait for the next change after current baseline state instead of polling.",
		"If the client dies or the session restarts, recover in this order: till.capture_state for scope context, till.attention_item(operation=list, all_scopes=true) for inbox state, till.handoff(operation=list) for durable coordination state, then till.comment(operation=list) for any thread you need to resume.",
		"Treat task/project details and comment summaries/bodies as markdown content in all agent-authored payloads.",
		"Treat till.comment as the default shared thread lane for discussion and status updates inside Tillsyn; comments are append-only coordination history, not private per-role mailboxes.",
		"Use role mentions intentionally in comment markdown: @human, @dev, @builder, @qa, @orchestrator, and @research are the supported routed mentions; @dev aliases to builder.",
		"Treat routed comment mentions as viewer-scoped inbox comments that belong in the Comments notifications section, not as generic warnings and not as action-required work by default.",
		"Treat till.handoff as the structured next-action lane; open handoffs should surface as Action Required rows only for the addressed viewer and otherwise remain coordination/oversight warnings until the receiving agent resolves them.",
		"When explaining attention rows, distinguish the noun family clearly: comment mentions are inbox comments, handoff mirrors are action-required coordination, and attention is the shared durable inbox substrate underneath both.",
		"Treat the scoped-auth split as expected behavior: global approved agent sessions are for template/global admin and project creation, while guarded in-project mutations should normally use project-scoped approved sessions.",
		"If a guarded mutation rejects a user session plus agent_instance_id/lease_token, either remove the guard tuple to act as a human or claim/validate a project-scoped approved agent session before retrying; issuing or renewing a lease alone does not change caller type.",
		"Never use another agent's or user's auth session, session secret, or auth_context_id; every actor should claim or validate its own scoped session.",
		"Always request the narrowest auth scope and shortest reasonable lifetime for the work, and prefer project-scoped auth over broader global auth whenever the runtime can prove that path.",
		"When documenting or using delegated auth, treat builder/qa/research child requests as an explicit acting-session flow: orchestrators request child auth with acting_session_id and acting_auth_context_id, requester ownership stays bound to the acting session, and child scope must remain within the acting approved path.",
		"Treat orchestrator cleanup as part of the workflow contract: child auth sessions, stale leases, pending requests, and stale coordination rows should be cleaned up truthfully when a run ends.",
		"Use the role model consistently: orchestrator plans/routes/delegates/cleans up, builder implements, qa verifies and closes or returns work, and research inspects code/runtime state, compiles findings, and can use local MCP tools plus Context7 to gather evidence.",
		"When template libraries are active, explain the actual scoped rule sources: project standards_markdown, template descriptions, child rules, branch/phase/task metadata, and node-contract snapshots.",
		"When creating or reconfiguring a project, have the orchestrator confirm with the dev which template library should govern the project, whether the project should stay template-only, and which generic kinds, if any, are explicitly allowed.",
		"When project setup or template refresh work compares Hylla-backed repo state with the installed DB template/binding state, the orchestrator must ask the dev before applying DB-mutating updates such as builtin ensure or template reapply.",
		"When documenting default-go or similar workflow contracts, distinguish project-only setup from the normal branch/work lifecycle and keep PLAN before BUILD explicit.",
		"Use depends_on, blocked_by, and blocked_reason to express real prerequisite order between tasks/phases today; do not rely on visual board position alone to tell agents what must finish first.",
		"When explaining template libraries, prefer concrete child_rules examples such as a build task that auto-generates one or more required QA subtasks owned by qa.",
		"When proposing policy changes, include concrete suggestions for AGENTS.md, CLAUDE.md, and any relevant SKILL.md files so builder/qa/research/orchestrator expectations stay synchronized.",
		"When workflow policy changes, update AGENTS.md, any tracked CLAUDE.md, and the relevant README/bootstrap/instruction surfaces in the same change instead of letting client guidance drift.",
		"Treat recommendations as proposal input and confirm AGENTS.md/CLAUDE.md policy updates with the user before editing.",
	}
}

// recommendedMDFileGuidance returns suggested section-level guidance for repository markdown policy files.
func recommendedMDFileGuidance() map[string][]string {
	return map[string][]string{
		"AGENTS.md": {
			"Scope boundaries: which directories and workflows each instruction block governs.",
			"Execution policy: when agents should act autonomously vs ask for approval.",
			"State explicitly that active tasks, actions, blockers, comments, handoffs, and worklogs must stay in Tillsyn instead of markdown trackers or planning files.",
			"Tooling policy: required MCP-first workflow and allowed fallback sources.",
			"Validation policy: exact Mage/test commands and required evidence before handoff.",
			"Dogfooding policy: reporting format for findings, blockers, and recovery steps.",
			"Authoring policy: task/project details and comment summaries/bodies must be written as markdown.",
			"Maintenance policy: when workflow rules change, update AGENTS.md together with any tracked CLAUDE.md plus README/bootstrap/instruction surfaces so client guidance stays aligned.",
			"Template policy: which actor kinds may draft, approve, bind, or apply template-library changes, and when human approval is mandatory.",
			"Project template policy: how project creation chooses a governing template library, whether generic kinds are allowed, and how allowed-kinds should track that decision.",
		},
		"CLAUDE.md": {
			"Interaction contract: communication style, update cadence, and escalation behavior.",
			"Decision policy: what assumptions are safe and what must be user-confirmed.",
			"Patch policy: file lock discipline, non-destructive defaults, and rollback constraints.",
			"Verification policy: required checks and how failures are reported.",
			"State explicitly that active tasks, actions, blockers, comments, handoffs, and worklogs must stay in Tillsyn instead of markdown trackers or planning files.",
			"Content policy: descriptions and comments should be authored and maintained as markdown.",
			"Coordination policy: comments and handoffs are shared communication surfaces by default; do not imply private per-role comment silos unless project policy says otherwise.",
			"Maintenance policy: keep CLAUDE.md aligned with the governing AGENTS.md plus README/bootstrap/instruction surfaces whenever workflow policy changes.",
			"Workflow policy: how actor kinds and template-generated blockers should be explained back to the user during execution.",
		},
		"README.md": {
			"Quickstart for till run/serve and MCP endpoint usage.",
			"Canonical tool index with minimal call examples for high-frequency workflows.",
			"State clearly that active task coordination, worklogs, and execution tracking belong in Tillsyn rather than in markdown files.",
			"Explain the expected scoped-auth model clearly: global agent auth for template/global admin and project creation, project-scoped auth for guarded in-project mutations.",
			"Document the delegated child-auth flow explicitly: orchestrators create bounded builder/qa/research requests through till.auth_request(operation=create) with acting session credentials, while child claim ownership stays with the approved child principal/client.",
			"Include one explicit coordination primer that explains comments vs mentions vs handoffs vs attention, so operators and agents can infer the intended workflow without reading implementation code.",
			"Document that AGENTS.md and any tracked CLAUDE.md are durable client-policy files to update when workflow guidance changes, not live task ledgers.",
			"Document that routed comment mentions belong in the viewer-scoped Comments notifications section, while open handoffs are the primary Action Required rows.",
			"Call out the supported role mentions explicitly: @human, @dev, @builder, @qa, @orchestrator, and @research, with @dev normalized to builder.",
			"Canonical template-library examples covering inspect, bind, contract lookup, and JSON transport for CLI/MCP authoring against the SQLite-backed source of truth.",
			"Document project-creation template policy explicitly: choose the governing template library up front, explain that template-bound projects can restrict allowed kinds to library-defined node kinds, and show how generic kinds are intentionally opted in.",
			"At least one readable child_rules example that shows multi-role follow-up work and truthful completion gates, such as a build task auto-generating multiple QA subtasks.",
			"Document the preferred workflow order for default-go style work: project setup when needed, then PLAN, BUILD, CLOSEOUT, and BRANCH CLEANUP.",
			"Document that depends_on, blocked_by, and blocked_reason are the current explicit sequencing tools for task/phase prerequisites until richer visual ordering rules exist.",
			"Explain where scoped rules live: project standards_markdown, template descriptions and child rules, and branch/phase/task metadata such as objective, acceptance criteria, definition of done, and validation plan.",
			"Explicit communication guidance that comments and handoffs are the default human-agent and agent-agent coordination lane inside Tillsyn.",
			"Dogfooding startup checklist and known operator guardrails.",
			"Markdown-first guidance for task/project details and comment content.",
			"Troubleshooting section for common MCP/TUI issues and recovery commands.",
		},
		"SKILL.md": {
			"State which till actor kinds and template-library workflows the skill assumes or modifies.",
			"Describe when the skill should recommend AGENTS.md or CLAUDE.md updates so human/operator policy stays aligned with runtime behavior.",
			"Call out the child_rules or blocker model directly when the skill relies on generated QA/research/builder follow-up work.",
			"Keep examples concrete and readable enough for human operators to audit quickly in TUI/CLI-oriented workflows.",
		},
		"MCP_DOGFOODING_WORKSHEET.md": {
			"Step-by-step validation scenarios covering task, thread, and attention flows.",
			"Expected vs observed result columns with timestamped evidence.",
			"Failure triage rubric and retest criteria after fixes.",
		},
	}
}
