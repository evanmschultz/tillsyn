package mcpapi

import (
	"context"
	"strings"
	"testing"

	"github.com/hylla/tillsyn/internal/adapters/server/common"
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
		templates: service,
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

// TestBuildInstructionsToolResponseExplainTemplate verifies template explanations are synthesized from runtime state.
func TestBuildInstructionsToolResponseExplainTemplate(t *testing.T) {
	t.Parallel()

	resp, err := buildInstructionsToolResponse(context.Background(), newInstructionsTestServices(), instructionsToolRequest{
		Focus:                  "template",
		ProjectID:              "p1",
		IncludeEvidence:        true,
		IncludeMarkdown:        false,
		IncludeRecommendations: false,
	})
	if err != nil {
		t.Fatalf("buildInstructionsToolResponse() error = %v", err)
	}
	if resp.Mode != string(instructionsToolModeExplain) {
		t.Fatalf("Mode = %q, want %q", resp.Mode, instructionsToolModeExplain)
	}
	if resp.Explanation == nil {
		t.Fatal("Explanation nil, want template explanation")
	}
	if got := resp.ResolvedScope.TemplateLibraryID; got != "go-defaults" {
		t.Fatalf("ResolvedScope.TemplateLibraryID = %q, want go-defaults", got)
	}
	if !strings.Contains(strings.ToLower(resp.Explanation.Overview), "template library") {
		t.Fatalf("Overview = %q, want template-library summary", resp.Explanation.Overview)
	}
	if len(resp.Explanation.Evidence) == 0 {
		t.Fatal("Evidence empty, want template description evidence")
	}
}

// TestBuildInstructionsToolResponseExplainProjectHighlightsTemplatePolicy verifies project explanations call out template-only policy and generic-kind exceptions.
func TestBuildInstructionsToolResponseExplainProjectHighlightsTemplatePolicy(t *testing.T) {
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
	rules := strings.ToLower(strings.Join(resp.Explanation.ScopedRules, " | "))
	if !strings.Contains(rules, "additional non-template kinds") {
		t.Fatalf("ScopedRules = %q, want generic-kind exception guidance", rules)
	}
	workflow := strings.ToLower(strings.Join(resp.Explanation.WorkflowContract, " | "))
	if !strings.Contains(workflow, "which template library governs the project") {
		t.Fatalf("WorkflowContract = %q, want project-creation template discussion guidance", workflow)
	}
	if !strings.Contains(workflow, "set_allowed_kinds") {
		t.Fatalf("WorkflowContract = %q, want allowlist adjustment guidance", workflow)
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
}

// TestBuildInstructionsToolResponseExplainKind verifies kind explanations join catalog and scoped usage context.
func TestBuildInstructionsToolResponseExplainKind(t *testing.T) {
	t.Parallel()

	resp, err := buildInstructionsToolResponse(context.Background(), newInstructionsTestServices(), instructionsToolRequest{
		Focus:                  "kind",
		KindID:                 "task",
		ProjectID:              "p1",
		TemplateLibraryID:      "go-defaults",
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
	if !strings.Contains(scopedRules, "library \"go-defaults\"") {
		t.Fatalf("ScopedRules = %q, want template-library context", scopedRules)
	}
}

// TestBuildInstructionsToolResponseExplainNode verifies node explanations lift metadata and contract facts.
func TestBuildInstructionsToolResponseExplainNode(t *testing.T) {
	t.Parallel()

	resp, err := buildInstructionsToolResponse(context.Background(), newInstructionsTestServices(), instructionsToolRequest{
		Focus:                  "node",
		NodeID:                 "task-1",
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
	if got := resp.ResolvedScope.NodeID; got != "task-1" {
		t.Fatalf("ResolvedScope.NodeID = %q, want task-1", got)
	}
	scopedRules := strings.ToLower(strings.Join(resp.Explanation.ScopedRules, " | "))
	if !strings.Contains(scopedRules, "validation plan") {
		t.Fatalf("ScopedRules = %q, want validation plan guidance", scopedRules)
	}
	workflow := strings.ToLower(strings.Join(resp.Explanation.WorkflowContract, " | "))
	if !strings.Contains(workflow, "responsible actor kind") {
		t.Fatalf("WorkflowContract = %q, want responsible actor guidance", workflow)
	}
	if len(resp.Explanation.Evidence) == 0 {
		t.Fatal("Evidence empty, want node policy evidence")
	}
}

// TestNormalizeInstructionsToolModeAndFocus verifies defaulting and invalid mode/focus combinations.
func TestNormalizeInstructionsToolModeAndFocus(t *testing.T) {
	t.Parallel()

	mode, focus, err := normalizeInstructionsToolModeAndFocus(instructionsToolRequest{NodeID: "task-1"})
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
