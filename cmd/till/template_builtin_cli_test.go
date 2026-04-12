package main

import (
	"strings"
	"testing"
	"time"

	"github.com/hylla/tillsyn/internal/domain"
)

// TestWriteBuiltinTemplateLibraryStatusDetail verifies builtin lifecycle status rendering stays operator-readable.
func TestWriteBuiltinTemplateLibraryStatusDetail(t *testing.T) {
	var out strings.Builder
	updatedAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	err := writeBuiltinTemplateLibraryStatusDetail(&out, domain.BuiltinTemplateLibraryStatus{
		LibraryID:             "default-go",
		Name:                  "Default Go",
		State:                 domain.BuiltinTemplateLibraryStateUpdateAvailable,
		BuiltinSource:         "builtin://tillsyn/default-go",
		BuiltinVersion:        "2026-04-12.1",
		BuiltinRevisionDigest: "builtin-digest",
		RequiredKindIDs:       []domain.KindID{"build-task", "go-project", "plan-phase"},
		MissingKindIDs:        []domain.KindID{"qa-check"},
		Installed:             true,
		InstalledLibraryName:  "Default Go",
		InstalledStatus:       domain.TemplateLibraryStatusApproved,
		InstalledRevision:     2,
		InstalledDigest:       "installed-digest",
		InstalledBuiltin:      false,
		InstalledUpdatedAt:    &updatedAt,
	})
	if err != nil {
		t.Fatalf("writeBuiltinTemplateLibraryStatusDetail() error = %v", err)
	}
	rendered := normalizeCLIOutput(out.String())
	for _, want := range []string{
		"Builtin Template Library",
		"state update_available",
		"builtin version 2026-04-12.1",
		"missing kinds qa-check",
		"installed builtin no",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in rendered output, got %q", want, rendered)
		}
	}
}

// TestWriteBuiltinTemplateLibraryEnsureDetail verifies ensure rendering includes both mutation outcome and resulting lifecycle state.
func TestWriteBuiltinTemplateLibraryEnsureDetail(t *testing.T) {
	var out strings.Builder
	err := writeBuiltinTemplateLibraryEnsureDetail(&out, domain.BuiltinTemplateLibraryEnsureResult{
		Library: domain.TemplateLibrary{
			ID:             "default-go",
			Name:           "Default Go",
			Status:         domain.TemplateLibraryStatusApproved,
			BuiltinVersion: "2026-04-12.1",
			Revision:       3,
		},
		Status: domain.BuiltinTemplateLibraryStatus{
			LibraryID:        "default-go",
			Name:             "Default Go",
			State:            domain.BuiltinTemplateLibraryStateCurrent,
			Installed:        true,
			InstalledBuiltin: true,
		},
		Changed: true,
	})
	if err != nil {
		t.Fatalf("writeBuiltinTemplateLibraryEnsureDetail() error = %v", err)
	}
	rendered := normalizeCLIOutput(out.String())
	for _, want := range []string{
		"Builtin Template Ensure",
		"changed yes",
		"revision 3",
		"Builtin Template Library",
		"state current",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in rendered output, got %q", want, rendered)
		}
	}
}
