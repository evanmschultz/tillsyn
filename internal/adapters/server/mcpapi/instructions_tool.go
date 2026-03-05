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

// instructionsToolDoc stores one filtered/possibly-truncated embedded markdown doc payload.
type instructionsToolDoc struct {
	FileName      string `json:"file_name"`
	Path          string `json:"path"`
	TotalChars    int    `json:"total_chars"`
	ReturnedChars int    `json:"returned_chars"`
	Truncated     bool   `json:"truncated"`
	Markdown      string `json:"markdown,omitempty"`
}

// instructionsToolResponse stores one till.get_instructions tool payload.
type instructionsToolResponse struct {
	Summary                  string                `json:"summary"`
	Topic                    string                `json:"topic,omitempty"`
	RecommendedAgentSettings []string              `json:"recommended_agent_settings,omitempty"`
	MDFileGuidance           map[string][]string   `json:"md_file_guidance,omitempty"`
	AvailableDocs            []string              `json:"available_docs"`
	Docs                     []instructionsToolDoc `json:"docs"`
}

// registerInstructionsTool registers the embedded-doc and dogfooding recommendation tool.
func registerInstructionsTool(srv *mcpserver.MCPServer) {
	srv.AddTool(
		mcp.NewTool(
			"till.get_instructions",
			mcp.WithDescription("Return embedded markdown docs and agent-facing dogfooding recommendations for using till MCP effectively."),
			mcp.WithString("topic", mcp.Description("Optional topic focus (for example: dogfooding, agents, claude, workflows)")),
			mcp.WithArray("doc_names", mcp.Description("Optional markdown file-name filter list (for example: README.md, AGENTS.md)"), mcp.WithStringItems()),
			mcp.WithBoolean("include_markdown", mcp.Description("Include markdown content in docs payload (default true)")),
			mcp.WithBoolean("include_recommendations", mcp.Description("Include settings and md-file guidance recommendations (default true)")),
			mcp.WithNumber("max_chars_per_doc", mcp.Description("Optional per-doc markdown truncation limit in characters (0 = no truncation)")),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			maxChars := req.GetInt("max_chars_per_doc", 0)
			if maxChars < 0 {
				return mcp.NewToolResultError("invalid_request: max_chars_per_doc must be >= 0"), nil
			}

			out, err := buildInstructionsToolResponse(
				req.GetString("topic", ""),
				req.GetStringSlice("doc_names", nil),
				req.GetBool("include_markdown", true),
				req.GetBool("include_recommendations", true),
				maxChars,
			)
			if err != nil {
				return nil, fmt.Errorf("build get_instructions result: %w", err)
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
func buildInstructionsToolResponse(topic string, docNames []string, includeMarkdown, includeRecommendations bool, maxChars int) (instructionsToolResponse, error) {
	docs, err := tillsyndocs.EmbeddedMarkdownDocuments()
	if err != nil {
		return instructionsToolResponse{}, fmt.Errorf("load embedded markdown docs: %w", err)
	}

	filter := map[string]struct{}{}
	for _, raw := range docNames {
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
		if len(filter) > 0 {
			if _, ok := filter[strings.ToLower(strings.TrimSpace(doc.FileName))]; !ok {
				continue
			}
		}
		payload := strings.TrimSpace(doc.Markdown)
		totalChars := len(payload)
		returned := payload
		truncated := false
		if maxChars > 0 && len(returned) > maxChars {
			returned = returned[:maxChars]
			truncated = true
		}
		row := instructionsToolDoc{
			FileName:      doc.FileName,
			Path:          doc.Path,
			TotalChars:    totalChars,
			ReturnedChars: len(returned),
			Truncated:     truncated,
		}
		if includeMarkdown {
			row.Markdown = returned
		}
		filtered = append(filtered, row)
	}
	sort.Strings(available)

	topic = strings.TrimSpace(topic)
	summary := "Embedded instruction docs for till MCP dogfooding and agent configuration guidance."
	if topic != "" {
		summary = fmt.Sprintf("Embedded instruction docs focused on %q for till MCP dogfooding and agent configuration guidance.", topic)
	}

	out := instructionsToolResponse{
		Summary:       summary,
		Topic:         topic,
		AvailableDocs: available,
		Docs:          filtered,
	}
	if includeRecommendations {
		out.RecommendedAgentSettings = recommendedInstructionSettings()
		out.MDFileGuidance = recommendedMDFileGuidance()
	}
	return out, nil
}

// recommendedInstructionSettings returns recommended agent behavior settings for instruction-tool usage.
func recommendedInstructionSettings() []string {
	return []string{
		"Use till.get_instructions when instructions are missing, stale, or ambiguous; skip redundant calls when AGENTS.md/README guidance is already sufficient for the current step.",
		"Use doc_names to scope context (for example README.md and AGENTS.md) instead of loading every doc on each step.",
		"Use include_markdown=false for quick inventory checks; enable it when drafting or validating policy text.",
		"Set max_chars_per_doc to keep responses bounded in long docs such as PLAN.md.",
		"Treat task/project details and comment summaries/bodies as markdown content in all agent-authored payloads.",
		"Treat recommendations as proposal input and confirm AGENTS.md/CLAUDE.md policy updates with the user before editing.",
	}
}

// recommendedMDFileGuidance returns suggested section-level guidance for repository markdown policy files.
func recommendedMDFileGuidance() map[string][]string {
	return map[string][]string{
		"AGENTS.md": {
			"Scope boundaries: which directories and workflows each instruction block governs.",
			"Execution policy: when agents should act autonomously vs ask for approval.",
			"Tooling policy: required MCP-first workflow and allowed fallback sources.",
			"Validation policy: exact just/test commands and required evidence before handoff.",
			"Dogfooding policy: reporting format for findings, blockers, and recovery steps.",
			"Authoring policy: task/project details and comment summaries/bodies must be written as markdown.",
		},
		"CLAUDE.md": {
			"Interaction contract: communication style, update cadence, and escalation behavior.",
			"Decision policy: what assumptions are safe and what must be user-confirmed.",
			"Patch policy: file lock discipline, non-destructive defaults, and rollback constraints.",
			"Verification policy: required checks and how failures are reported.",
			"Content policy: descriptions and comments should be authored and maintained as markdown.",
		},
		"README.md": {
			"Quickstart for till run/serve and MCP endpoint usage.",
			"Canonical tool index with minimal call examples for high-frequency workflows.",
			"Dogfooding startup checklist and known operator guardrails.",
			"Markdown-first guidance for task/project details and comment content.",
			"Troubleshooting section for common MCP/TUI issues and recovery commands.",
		},
		"MCP_DOGFOODING_WORKSHEET.md": {
			"Step-by-step validation scenarios covering task, thread, and attention flows.",
			"Expected vs observed result columns with timestamped evidence.",
			"Failure triage rubric and retest criteria after fixes.",
		},
	}
}
