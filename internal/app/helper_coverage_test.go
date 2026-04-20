package app

import (
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
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
	scopes = appendMutationScopeCandidate(scopes, mutationScopeCandidate{ScopeType: domain.CapabilityScopeActionItem, ScopeID: " actionItem-1 "})
	scopes = appendMutationScopeCandidate(scopes, mutationScopeCandidate{ScopeType: domain.CapabilityScopeActionItem, ScopeID: "actionItem-1"})
	scopes = appendMutationScopeCandidate(scopes, mutationScopeCandidate{ScopeType: domain.CapabilityScopeType(""), ScopeID: "ignored"})
	scopes = appendMutationScopeCandidate(scopes, mutationScopeCandidate{ScopeType: domain.CapabilityScopeProject, ScopeID: ""})

	if len(scopes) != 1 {
		t.Fatalf("scope count = %d, want 1", len(scopes))
	}
	if scopes[0].ScopeType != domain.CapabilityScopeActionItem || scopes[0].ScopeID != "actionItem-1" {
		t.Fatalf("normalized scope = %#v, want actionItem/actionItem-1", scopes[0])
	}
}
