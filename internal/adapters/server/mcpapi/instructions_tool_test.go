package mcpapi

import (
	"context"
	"strings"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/adapters/server/common"
)

// newInstructionsTestServices returns one deterministic scoped-instructions test service bundle.
func newInstructionsTestServices() instructionsExplainServices {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	return instructionsExplainServices{
		bootstrap: service,
		projects:  service,
		tasks:     service,
		kinds:     service,
	}
}

// TestBuildInstructionsToolResponseDocsMode verifies the legacy docs-only behavior remains available.
func TestBuildInstructionsToolResponseDocsMode(t *testing.T) {
	t.Parallel()

	resp, err := buildInstructionsToolResponse(context.Background(), newInstructionsTestServices(), instructionsToolRequest{
		DocNames:               []string{"README.md"},
		IncludeMarkdown:        false,
		IncludeRecommendations: true,
	})
	if err != nil {
		t.Fatalf("buildInstructionsToolResponse() error = %v", err)
	}
	if resp.Mode != string(instructionsToolModeDocs) {
		t.Fatalf("Mode = %q, want %q", resp.Mode, instructionsToolModeDocs)
	}
	if resp.Focus != string(instructionsToolFocusTopic) {
		t.Fatalf("Focus = %q, want %q", resp.Focus, instructionsToolFocusTopic)
	}
	if len(resp.Docs) == 0 {
		t.Fatal("Docs empty, want filtered embedded docs")
	}
	if resp.Explanation != nil {
		t.Fatalf("Explanation = %#v, want nil for docs-only mode", resp.Explanation)
	}
}

// TestBuildInstructionsToolResponseExplainProjectHighlightsCoordinationPolicy verifies project explanations surface the project-scoped coordination contract.
func TestBuildInstructionsToolResponseExplainProjectHighlightsCoordinationPolicy(t *testing.T) {
	t.Parallel()

	resp, err := buildInstructionsToolResponse(context.Background(), newInstructionsTestServices(), instructionsToolRequest{
		Focus:                  "project",
		ProjectID:              "p1",
		IncludeEvidence:        false,
		IncludeMarkdown:        false,
		IncludeRecommendations: false,
	})
	if err != nil {
		t.Fatalf("buildInstructionsToolResponse() error = %v", err)
	}
	if resp.Explanation == nil {
		t.Fatal("Explanation nil, want project explanation")
	}
	workflow := strings.ToLower(strings.Join(resp.Explanation.WorkflowContract, " | "))
	if !strings.Contains(workflow, "project-scoped approved sessions") {
		t.Fatalf("WorkflowContract = %q, want project-scoped session guidance", workflow)
	}
	if !strings.Contains(workflow, "till.comment") || !strings.Contains(workflow, "till.handoff") {
		t.Fatalf("WorkflowContract = %q, want till.comment/till.handoff coordination guidance", workflow)
	}
}

// TestBuildInstructionsToolResponseExplainBootstrap verifies bootstrap topic uses the richer shared bootstrap guidance.
func TestBuildInstructionsToolResponseExplainBootstrap(t *testing.T) {
	t.Parallel()

	resp, err := buildInstructionsToolResponse(context.Background(), newInstructionsTestServices(), instructionsToolRequest{
		Mode:                   "explain",
		Focus:                  "topic",
		Topic:                  "bootstrap",
		IncludeMarkdown:        false,
		IncludeRecommendations: false,
	})
	if err != nil {
		t.Fatalf("buildInstructionsToolResponse() error = %v", err)
	}
	if resp.Explanation == nil {
		t.Fatal("Explanation nil, want bootstrap explanation")
	}
	if got := resp.Explanation.Title; got != "Bootstrap Guidance" {
		t.Fatalf("Title = %q, want Bootstrap Guidance", got)
	}
	workflow := strings.ToLower(strings.Join(resp.Explanation.WorkflowContract, " | "))
	if !strings.Contains(workflow, "till.auth_request(operation=create)") {
		t.Fatalf("WorkflowContract = %q, want auth-request bootstrap guidance", workflow)
	}
	expectations := strings.ToLower(strings.Join(resp.Explanation.AgentExpectations, " | "))
	if !strings.Contains(expectations, "till.get_instructions") {
		t.Fatalf("AgentExpectations = %q, want get_instructions bootstrap guidance", expectations)
	}
	if !strings.Contains(expectations, "agents.md") || !strings.Contains(expectations, "claude.md") {
		t.Fatalf("AgentExpectations = %q, want AGENTS.md/CLAUDE.md policy-sync guidance", expectations)
	}
	scopedRules := strings.ToLower(strings.Join(resp.Explanation.ScopedRules, " | "))
	if !strings.Contains(scopedRules, "worklogs in tillsyn itself") {
		t.Fatalf("ScopedRules = %q, want tillsyn-only coordination guidance", scopedRules)
	}
}

// TestBuildInstructionsToolResponseExplainKind verifies kind explanations join catalog and scoped usage context.
func TestBuildInstructionsToolResponseExplainKind(t *testing.T) {
	t.Parallel()

	resp, err := buildInstructionsToolResponse(context.Background(), newInstructionsTestServices(), instructionsToolRequest{
		Focus:                  "kind",
		KindID:                 "actionItem",
		ProjectID:              "p1",
		IncludeEvidence:        true,
		IncludeMarkdown:        false,
		IncludeRecommendations: false,
	})
	if err != nil {
		t.Fatalf("buildInstructionsToolResponse() error = %v", err)
	}
	if resp.Explanation == nil {
		t.Fatal("Explanation nil, want kind explanation")
	}
	scopedRules := strings.ToLower(strings.Join(resp.Explanation.ScopedRules, " | "))
	if !strings.Contains(scopedRules, "project \"p1\" currently allows kind") {
		t.Fatalf("ScopedRules = %q, want project allowlist guidance", scopedRules)
	}
}

// TestBuildInstructionsToolResponseExplainNode verifies node explanations lift metadata and contract facts.
func TestBuildInstructionsToolResponseExplainNode(t *testing.T) {
	t.Parallel()

	resp, err := buildInstructionsToolResponse(context.Background(), newInstructionsTestServices(), instructionsToolRequest{
		Focus:                  "node",
		NodeID:                 "actionItem-1",
		IncludeEvidence:        true,
		IncludeMarkdown:        false,
		IncludeRecommendations: false,
	})
	if err != nil {
		t.Fatalf("buildInstructionsToolResponse() error = %v", err)
	}
	if resp.Explanation == nil {
		t.Fatal("Explanation nil, want node explanation")
	}
	if got := resp.ResolvedScope.NodeID; got != "actionItem-1" {
		t.Fatalf("ResolvedScope.NodeID = %q, want actionItem-1", got)
	}
	scopedRules := strings.ToLower(strings.Join(resp.Explanation.ScopedRules, " | "))
	if !strings.Contains(scopedRules, "validation plan") {
		t.Fatalf("ScopedRules = %q, want validation plan guidance", scopedRules)
	}
	workflow := strings.ToLower(strings.Join(resp.Explanation.WorkflowContract, " | "))
	if !strings.Contains(workflow, "depends_on") || !strings.Contains(workflow, "blocked_by") {
		t.Fatalf("WorkflowContract = %q, want dependency/blocked_by sequencing guidance", workflow)
	}
	if len(resp.Explanation.Evidence) == 0 {
		t.Fatal("Evidence empty, want node policy evidence")
	}
}

// TestNormalizeInstructionsToolModeAndFocus verifies defaulting and invalid mode/focus combinations.
func TestNormalizeInstructionsToolModeAndFocus(t *testing.T) {
	t.Parallel()

	mode, focus, err := normalizeInstructionsToolModeAndFocus(instructionsToolRequest{NodeID: "actionItem-1"})
	if err != nil {
		t.Fatalf("normalizeInstructionsToolModeAndFocus(node) error = %v", err)
	}
	if mode != instructionsToolModeExplain {
		t.Fatalf("mode = %q, want %q", mode, instructionsToolModeExplain)
	}
	if focus != instructionsToolFocusNode {
		t.Fatalf("focus = %q, want %q", focus, instructionsToolFocusNode)
	}

	if _, _, err := normalizeInstructionsToolModeAndFocus(instructionsToolRequest{
		Mode:      "docs",
		ProjectID: "p1",
	}); err == nil {
		t.Fatal("normalizeInstructionsToolModeAndFocus(mode=docs, project_id) error = nil, want invalid_request")
	}
}

// TestRecommendedInstructionSettingsIncludeTillsynOnlyCoordination verifies the helper recommendations reinforce Tillsyn-only active coordination.
func TestRecommendedInstructionSettingsIncludeTillsynOnlyCoordination(t *testing.T) {
	t.Parallel()

	recommendations := strings.ToLower(strings.Join(recommendedInstructionSettings(), " | "))
	if !strings.Contains(recommendations, "worklogs in tillsyn itself") {
		t.Fatalf("recommendedInstructionSettings() = %q, want Tillsyn-only worklog guidance", recommendations)
	}
	if !strings.Contains(recommendations, "agents.md") || !strings.Contains(recommendations, "claude.md") {
		t.Fatalf("recommendedInstructionSettings() = %q, want AGENTS.md/CLAUDE.md sync guidance", recommendations)
	}
}

// TestRecommendedMDFileGuidanceHighlightsTillsynOnlyPolicy verifies repo-doc recommendations forbid markdown execution ledgers.
func TestRecommendedMDFileGuidanceHighlightsTillsynOnlyPolicy(t *testing.T) {
	t.Parallel()

	guidance := recommendedMDFileGuidance()
	agents := strings.ToLower(strings.Join(guidance["AGENTS.md"], " | "))
	if !strings.Contains(agents, "must stay in tillsyn") {
		t.Fatalf("AGENTS.md guidance = %q, want Tillsyn-only coordination guidance", agents)
	}
	if !strings.Contains(agents, "claude.md") {
		t.Fatalf("AGENTS.md guidance = %q, want CLAUDE.md alignment guidance", agents)
	}
	claude := strings.ToLower(strings.Join(guidance["CLAUDE.md"], " | "))
	if !strings.Contains(claude, "must stay in tillsyn") {
		t.Fatalf("CLAUDE.md guidance = %q, want Tillsyn-only coordination guidance", claude)
	}
}
