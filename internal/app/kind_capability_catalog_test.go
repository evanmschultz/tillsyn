package app

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
	"github.com/evanmschultz/tillsyn/internal/templates"
)

// TestKindCatalogResolutionFallsBackToRepoOnEmpty covers droplet 3.12's
// boot-compatibility acceptance criterion: a project carrying an empty
// KindCatalogJSON envelope routes resolveActionItemKindDefinition
// through the legacy repo.GetKindDefinition path. The fakeRepo pre-seeds
// the closed 12-value Kind enum so the repo lookup succeeds, asserting
// the fallback returns without calling any catalog code path.
//
// REPLACEMENT NOTE — Drop 4c.5 droplet F.1.1: pre-F.1.1 every project
// (regardless of repo paths) had an empty KindCatalogJSON because
// loadProjectTemplate was a Drop 3.14 stub returning ok=false. Post-F.1.1
// projects WITHOUT repo paths receive a non-empty catalog from the
// embedded language-default; the empty-catalog branch is now reached
// only by projects that DO declare repo paths AND whose F.1.2 walk
// (lands in a future droplet) finds no on-disk template. Until F.1.2
// lands, ANY project with non-empty RepoBareRoot or RepoPrimaryWorktree
// preserves the empty-catalog behavior — that is the construction this
// test now uses, swapping CreateProject (zero-value paths) for a
// CreateProjectWithMetadata call that supplies a non-empty primary
// worktree path. The legacy repo fallback path remains tested; the test
// just no longer relies on the embedded-template fallback being absent.
//
// Once F.1.2 lands and replaces the non-empty-path branch with the
// candidate walk, this test should pin a path that has no `.tillsyn/
// template.toml` on disk so the walk falls through to the embedded
// default — at which point this test stops asserting empty-catalog
// behavior altogether and converts to a positive walk-fallthrough test.
func TestKindCatalogResolutionFallsBackToRepoOnEmpty(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProjectWithMetadata(context.Background(), CreateProjectInput{
		Name:                "Empty Catalog",
		RepoPrimaryWorktree: "/abs/path/to/worktree",
		Language:            "go",
	})
	if err != nil {
		t.Fatalf("CreateProjectWithMetadata() error = %v", err)
	}
	if len(project.KindCatalogJSON) != 0 {
		t.Fatalf("CreateProjectWithMetadata() left non-empty KindCatalogJSON; expected empty under F.1.1's non-empty-path skip seam (F.1.2 will replace this construction): %s", string(project.KindCatalogJSON))
	}

	def, err := svc.resolveActionItemKindDefinition(
		context.Background(),
		project.ID,
		domain.KindID(domain.KindBuild),
		domain.KindAppliesToBuild,
		nil,
	)
	if err != nil {
		t.Fatalf("resolveActionItemKindDefinition() error = %v", err)
	}
	if def.ID != domain.KindID(domain.KindBuild) {
		t.Fatalf("resolveActionItemKindDefinition() id = %q, want %q", def.ID, domain.KindBuild)
	}
	// fakeRepo pre-seeds DisplayName="Build" — proves the fallback hit the
	// repo path rather than synthesizing from a catalog (which would set
	// DisplayName=string(kindID)="build").
	if def.DisplayName != "Build" {
		t.Fatalf("resolveActionItemKindDefinition() DisplayName = %q, want %q (legacy repo path)", def.DisplayName, "Build")
	}
}

// TestKindCatalogResolutionFromBakedCatalog covers droplet 3.12's
// catalog-hit acceptance criterion: when a project carries a non-empty
// KindCatalogJSON whose Kinds map contains the requested kind, the
// resolver must satisfy the request from the catalog without calling
// repo.GetKindDefinition. We assert that property by deleting the
// corresponding entry from fakeRepo.kindDefs — if the resolver falls
// through to repo, it returns ErrNotFound; if it satisfies from the
// catalog, the call succeeds and DisplayName matches the synthesized form.
func TestKindCatalogResolutionFromBakedCatalog(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(context.Background(), "Baked Catalog", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	// Bake a catalog that covers the kind under test.
	tpl := templates.Template{
		SchemaVersion: templates.SchemaVersionV1,
		Kinds: map[domain.Kind]templates.KindRule{
			domain.KindBuild: {
				AllowedParentKinds: []domain.Kind{domain.KindPlan},
				StructuralType:     domain.StructuralTypeDroplet,
			},
		},
	}
	catalog := templates.Bake(tpl)
	encoded, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("json.Marshal(catalog) error = %v", err)
	}
	project.KindCatalogJSON = encoded
	if err := repo.UpdateProject(context.Background(), project); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	// Drop the legacy repo entry so a fallback would error out.
	delete(repo.kindDefs, domain.KindID(domain.KindBuild))

	def, err := svc.resolveActionItemKindDefinition(
		context.Background(),
		project.ID,
		domain.KindID(domain.KindBuild),
		domain.KindAppliesToBuild,
		nil,
	)
	if err != nil {
		t.Fatalf("resolveActionItemKindDefinition() error = %v (catalog hit expected, repo entry absent)", err)
	}
	if def.ID != domain.KindID(domain.KindBuild) {
		t.Fatalf("resolveActionItemKindDefinition() id = %q, want %q", def.ID, domain.KindBuild)
	}
	// Synthesized from KindRule: DisplayName = string(kindID), not the
	// legacy repo's "Build" display name. This proves the catalog path was
	// taken rather than a fallback.
	if def.DisplayName != "build" {
		t.Fatalf("resolveActionItemKindDefinition() DisplayName = %q, want %q (synthesized catalog path)", def.DisplayName, "build")
	}
	// Per droplet 3.15 the synthesized definition no longer carries an
	// AllowedParentScopes mirror — that field was deleted. Parent-nesting
	// gates flow through KindCatalog.AllowsNesting on the live catalog
	// snapshot instead. The catalog hit is asserted by DisplayName above.
}

// TestKindCatalogResolutionFallsBackOnMalformedJSON covers the soft-failure
// branch in lookupKindDefinitionFromCatalog: a malformed KindCatalogJSON
// envelope must NOT brick resolution; the legacy repo path picks up. This
// is a defensive guard so a bad envelope can never bring down the create
// path before a future drop adds full schema-version routing.
func TestKindCatalogResolutionFallsBackOnMalformedJSON(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	project, err := svc.CreateProject(context.Background(), "Bad Catalog", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	project.KindCatalogJSON = json.RawMessage(`{not valid json`)
	if err := repo.UpdateProject(context.Background(), project); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	def, err := svc.resolveActionItemKindDefinition(
		context.Background(),
		project.ID,
		domain.KindID(domain.KindBuild),
		domain.KindAppliesToBuild,
		nil,
	)
	if err != nil {
		t.Fatalf("resolveActionItemKindDefinition() error = %v (expected fallback to repo)", err)
	}
	if def.DisplayName != "Build" {
		t.Fatalf("resolveActionItemKindDefinition() DisplayName = %q, want legacy %q (fallback path)", def.DisplayName, "Build")
	}
}
