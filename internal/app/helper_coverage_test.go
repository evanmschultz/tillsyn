package app

import (
	"strings"
	"testing"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestMutationScopeHelpersNormalizeAndDeduplicate verifies scope helper normalization stays stable.
func TestMutationScopeHelpersNormalizeAndDeduplicate(t *testing.T) {
	projectScope := newProjectMutationScopeCandidate("  project-1  ")
	if projectScope.ScopeType != domain.CapabilityScopeProject {
		t.Fatalf("project scope type = %q, want %q", projectScope.ScopeType, domain.CapabilityScopeProject)
	}
	if projectScope.ScopeID != "project-1" {
		t.Fatalf("project scope id = %q, want project-1", projectScope.ScopeID)
	}

	var scopes []mutationScopeCandidate
	scopes = appendMutationScopeCandidate(scopes, mutationScopeCandidate{ScopeType: domain.CapabilityScopeTask, ScopeID: " task-1 "})
	scopes = appendMutationScopeCandidate(scopes, mutationScopeCandidate{ScopeType: domain.CapabilityScopeTask, ScopeID: "task-1"})
	scopes = appendMutationScopeCandidate(scopes, mutationScopeCandidate{ScopeType: domain.CapabilityScopeType(""), ScopeID: "ignored"})
	scopes = appendMutationScopeCandidate(scopes, mutationScopeCandidate{ScopeType: domain.CapabilityScopeProject, ScopeID: ""})

	if len(scopes) != 1 {
		t.Fatalf("scope count = %d, want 1", len(scopes))
	}
	if scopes[0].ScopeType != domain.CapabilityScopeTask || scopes[0].ScopeID != "task-1" {
		t.Fatalf("normalized scope = %#v, want task/task-1", scopes[0])
	}
}

// TestTemplateContractHelperFormatting verifies actor-kind summaries and blocker text stay operator-readable.
func TestTemplateContractHelperFormatting(t *testing.T) {
	summary := nodeContractActorKindsSummary(
		[]domain.TemplateActorKind{
			domain.TemplateActorKindQA,
			domain.TemplateActorKind(" builder "),
			domain.TemplateActorKindQA,
		},
		true,
		true,
	)
	if summary != "builder, human, orchestrator (override), qa" {
		t.Fatalf("actor kind summary = %q", summary)
	}

	if got := taskDisplayLabel(domain.Task{Title: "  ", ID: "task-123"}); got != "task-123" {
		t.Fatalf("taskDisplayLabel(empty title) = %q, want task-123", got)
	}
	if got := taskDisplayLabel(domain.Task{Title: "QA pass 1", ID: "task-123"}); got != "QA pass 1" {
		t.Fatalf("taskDisplayLabel(title) = %q, want QA pass 1", got)
	}

	blocker := formatNodeContractBlocker(
		domain.Task{Title: "QA pass 1", ID: "task-123"},
		domain.NodeContractSnapshot{
			ResponsibleActorKind:    domain.TemplateActorKindQA,
			CompletableByActorKinds: []domain.TemplateActorKind{domain.TemplateActorKindQA},
			OrchestratorMayComplete: true,
		},
		"parent",
	)
	for _, want := range []string{"parent blocker", "QA pass 1", "responsible actor kind: qa", "orchestrator (override)"} {
		if !strings.Contains(blocker, want) {
			t.Fatalf("expected blocker text to contain %q, got %q", want, blocker)
		}
	}
}

// TestFirstActorTypePrefersFirstNormalizedValue verifies actor fallback selection ignores blanks.
func TestFirstActorTypePrefersFirstNormalizedValue(t *testing.T) {
	got := firstActorType(domain.ActorTypeAgent, domain.ActorTypeUser)
	if got != domain.ActorTypeAgent {
		t.Fatalf("firstActorType(agent first) = %q, want %q", got, domain.ActorTypeAgent)
	}
	if got := firstActorType("", domain.ActorType("  ")); got != domain.ActorTypeUser {
		t.Fatalf("firstActorType(blank default) = %q, want %q", got, domain.ActorTypeUser)
	}
}
