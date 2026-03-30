package domain

import (
	"errors"
	"testing"
	"time"
)

// TestNewTemplateLibraryNormalizesNestedRules verifies template-library constructors normalize and sort nested rules.
func TestNewTemplateLibraryNormalizesNestedRules(t *testing.T) {
	now := time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC)
	library, err := NewTemplateLibrary(TemplateLibraryInput{
		ID:          " Library-1 ",
		Scope:       TemplateLibraryScopeProject,
		ProjectID:   "project-1",
		Name:        " Example Library ",
		Description: " Example description ",
		Status:      TemplateLibraryStatusApproved,
		NodeTemplates: []NodeTemplateInput{
			{
				ID:         " builder-task ",
				ScopeLevel: KindAppliesToTask,
				NodeKindID: "task",
				ChildRules: []TemplateChildRuleInput{
					{
						ID:                   " qa-check ",
						Position:             -1,
						ChildScopeLevel:      KindAppliesToSubtask,
						ChildKindID:          "subtask",
						TitleTemplate:        "Run QA",
						ResponsibleActorKind: TemplateActorKindQA,
					},
				},
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("NewTemplateLibrary() error = %v", err)
	}
	if library.ID != "library-1" {
		t.Fatalf("library.ID = %q, want library-1", library.ID)
	}
	if library.Name != "Example Library" {
		t.Fatalf("library.Name = %q, want trimmed value", library.Name)
	}
	if library.CreatedByActorType != ActorTypeUser {
		t.Fatalf("library.CreatedByActorType = %q, want user default", library.CreatedByActorType)
	}
	if len(library.NodeTemplates) != 1 {
		t.Fatalf("len(library.NodeTemplates) = %d, want 1", len(library.NodeTemplates))
	}
	nodeTemplate := library.NodeTemplates[0]
	if nodeTemplate.ID != "builder-task" {
		t.Fatalf("nodeTemplate.ID = %q, want normalized value", nodeTemplate.ID)
	}
	if len(nodeTemplate.ChildRules) != 1 {
		t.Fatalf("len(nodeTemplate.ChildRules) = %d, want 1", len(nodeTemplate.ChildRules))
	}
	childRule := nodeTemplate.ChildRules[0]
	if childRule.Position != 0 {
		t.Fatalf("childRule.Position = %d, want clamped zero", childRule.Position)
	}
	if len(childRule.EditableByActorKinds) != 1 || childRule.EditableByActorKinds[0] != TemplateActorKindQA {
		t.Fatalf("childRule.EditableByActorKinds = %#v, want qa fallback", childRule.EditableByActorKinds)
	}
	if len(childRule.CompletableByActorKinds) != 1 || childRule.CompletableByActorKinds[0] != TemplateActorKindQA {
		t.Fatalf("childRule.CompletableByActorKinds = %#v, want qa fallback", childRule.CompletableByActorKinds)
	}
}

// TestNewTemplateLibraryRejectsDuplicateScopeKind verifies one library cannot define two templates for the same scope and node kind.
func TestNewTemplateLibraryRejectsDuplicateScopeKind(t *testing.T) {
	now := time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC)
	_, err := NewTemplateLibrary(TemplateLibraryInput{
		ID:        "library-1",
		Scope:     TemplateLibraryScopeProject,
		ProjectID: "project-1",
		Name:      "Example Library",
		Status:    TemplateLibraryStatusDraft,
		NodeTemplates: []NodeTemplateInput{
			{ID: "task-a", ScopeLevel: KindAppliesToTask, NodeKindID: "task"},
			{ID: "task-b", ScopeLevel: KindAppliesToTask, NodeKindID: "task"},
		},
	}, now)
	if !errors.Is(err, ErrInvalidTemplateLibrary) {
		t.Fatalf("NewTemplateLibrary() error = %v, want ErrInvalidTemplateLibrary", err)
	}
}

// TestNewNodeContractSnapshotDefaultsActorKinds verifies generated node contracts default to responsible actor ownership.
func TestNewNodeContractSnapshotDefaultsActorKinds(t *testing.T) {
	now := time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC)
	snapshot, err := NewNodeContractSnapshot(NodeContractSnapshotInput{
		NodeID:               "node-1",
		ProjectID:            "project-1",
		SourceLibraryID:      "library-1",
		SourceNodeTemplateID: "task-template",
		SourceChildRuleID:    "qa-check",
		ResponsibleActorKind: TemplateActorKindBuilder,
	}, now)
	if err != nil {
		t.Fatalf("NewNodeContractSnapshot() error = %v", err)
	}
	if snapshot.CreatedByActorType != ActorTypeSystem {
		t.Fatalf("snapshot.CreatedByActorType = %q, want system default", snapshot.CreatedByActorType)
	}
	if len(snapshot.EditableByActorKinds) != 1 || snapshot.EditableByActorKinds[0] != TemplateActorKindBuilder {
		t.Fatalf("snapshot.EditableByActorKinds = %#v, want builder fallback", snapshot.EditableByActorKinds)
	}
	if len(snapshot.CompletableByActorKinds) != 1 || snapshot.CompletableByActorKinds[0] != TemplateActorKindBuilder {
		t.Fatalf("snapshot.CompletableByActorKinds = %#v, want builder fallback", snapshot.CompletableByActorKinds)
	}
}
