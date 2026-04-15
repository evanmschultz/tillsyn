package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestServiceListCapabilityLeasesFiltersRevoked verifies default lease listing hides revoked rows.
func TestServiceListCapabilityLeasesFiltersRevoked(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	now := time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Project One", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	repo.projects[project.ID] = project

	active, err := domain.NewCapabilityLease(domain.CapabilityLeaseInput{
		InstanceID: "lease-active",
		LeaseToken: "token-active",
		AgentName:  "Builder One",
		ProjectID:  project.ID,
		ScopeType:  domain.CapabilityScopeProject,
		ScopeID:    project.ID,
		Role:       domain.CapabilityRoleBuilder,
		ExpiresAt:  now.Add(2 * time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("NewCapabilityLease(active) error = %v", err)
	}
	revoked, err := domain.NewCapabilityLease(domain.CapabilityLeaseInput{
		InstanceID: "lease-revoked",
		LeaseToken: "token-revoked",
		AgentName:  "QA One",
		ProjectID:  project.ID,
		ScopeType:  domain.CapabilityScopeProject,
		ScopeID:    project.ID,
		Role:       domain.CapabilityRoleQA,
		ExpiresAt:  now.Add(2 * time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("NewCapabilityLease(revoked) error = %v", err)
	}
	revoked.Revoke("done", now.Add(30*time.Minute))

	repo.capabilityLeases[active.InstanceID] = active
	repo.capabilityLeases[revoked.InstanceID] = revoked

	svc := NewService(repo, func() string { return "unused" }, func() time.Time { return now }, ServiceConfig{})

	listed, err := svc.ListCapabilityLeases(context.Background(), ListCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeProject,
	})
	if err != nil {
		t.Fatalf("ListCapabilityLeases(active only) error = %v", err)
	}
	if len(listed) != 1 || listed[0].InstanceID != active.InstanceID {
		t.Fatalf("ListCapabilityLeases(active only) = %#v, want only active lease", listed)
	}

	listedAll, err := svc.ListCapabilityLeases(context.Background(), ListCapabilityLeasesInput{
		ProjectID:      project.ID,
		ScopeType:      domain.CapabilityScopeProject,
		IncludeRevoked: true,
	})
	if err != nil {
		t.Fatalf("ListCapabilityLeases(include revoked) error = %v", err)
	}
	if len(listedAll) != 2 {
		t.Fatalf("ListCapabilityLeases(include revoked) len = %d, want 2", len(listedAll))
	}
}

// TestServiceListCapabilityLeasesRequiresProject verifies lease listing rejects unknown projects.
func TestServiceListCapabilityLeasesRequiresProject(t *testing.T) {
	t.Parallel()

	svc := NewService(newFakeRepo(), func() string { return "unused" }, time.Now, ServiceConfig{})
	_, err := svc.ListCapabilityLeases(context.Background(), ListCapabilityLeasesInput{
		ProjectID: "missing",
		ScopeType: domain.CapabilityScopeProject,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListCapabilityLeases() error = %v, want %v", err, ErrNotFound)
	}
}
