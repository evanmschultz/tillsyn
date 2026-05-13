// Drop 4c.5 droplet F.3.3 round-2 service-level rollback coverage.
//
// Round-1 falsification (workflow/drop_4c_5/BUILDER_QA_FALSIFICATION.md
// § "Droplet F.3.3 — Round 1") flagged that the existing
// TestTillTemplate_Set_RebakeFailureRollback in the mcpapi adapter
// package stubs the in-band failure shape via setProjectTemplateResultFn
// — it never reaches the production rollback branch in
// Service.SetProjectTemplate (template_service.go:494-511) where
// os.Rename(dest, failedPath) executes the orphaned-artifact dance. This
// file exercises that branch end-to-end at the service layer with a
// repo stub whose UpdateProject returns an error AFTER the atomic
// rename has already landed the file on disk.
//
// Two test cases:
//
//   - TestService_SetProjectTemplate_RollbackOnUpdateProjectFailure —
//     the spec-acceptance #6 single-failure rollback path. Asserts the
//     destination file is moved aside, the failed-sentinel file holds
//     the original bytes, and the returned error names the rollback
//     path.
//   - TestService_SetProjectTemplate_RollbackRenameAlsoFails — the
//     "ROLLBACK ALSO FAILED" branch. Pre-creates a blocker DIRECTORY at
//     the deterministic failed-path location so os.Rename(dest,
//     failedPath) returns ENOTDIR; asserts the error string carries the
//     "ROLLBACK ALSO FAILED" sentinel + the original persist failure.

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// minimalValidTemplateTOML is the smallest TOML body that survives
// templates.LoadWithOptions(...) cleanly. Mirrors the body in
// internal/templates/load_test.go § TestLoadValidTemplate so a future
// load.go validator addition that forces a richer minimum lands as a
// single fixture update here rather than per-test.
const minimalValidTemplateTOML = `
schema_version = "v1"

[kinds.build]
owner = "STEWARD"
allowed_parent_kinds = ["plan"]
allowed_child_kinds = ["build-qa-proof", "build-qa-falsification"]
structural_type = "droplet"

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build-qa-proof"
title = "BUILD-QA-PROOF"
blocked_by_parent = true

[[child_rules]]
when_parent_kind = "build"
create_child_kind = "build-qa-falsification"
title = "BUILD-QA-FALSIFICATION"
blocked_by_parent = true

[agent_bindings.build]
agent_name = "builder-agent"
model = "opus"
`

// errOnUpdateProjectRepo wraps a *fakeRepo and overrides UpdateProject
// to return a fixed error. Embedding the concrete fake means every
// other Repository method satisfies the interface unchanged — the only
// behavior delta is the controlled UpdateProject failure used to drive
// SetProjectTemplate into its rollback branch.
type errOnUpdateProjectRepo struct {
	*fakeRepo
	updateProjectErr error
}

// UpdateProject returns the configured fixed error so the caller's
// post-write persist step fails deterministically. The underlying
// fakeRepo is NOT mutated — the test asserts on the in-memory project
// state only via direct map access if needed, but the rollback branch
// itself does not depend on whether the persist partially succeeded.
func (r *errOnUpdateProjectRepo) UpdateProject(_ context.Context, _ domain.Project) error {
	return r.updateProjectErr
}

// newServiceWithRepo constructs a *Service backed by the supplied
// Repository. Mirrors the minimal NewService wiring used elsewhere in
// this package's tests (deterministic id-gen + fixed clock) so the
// failed-path suffix is predictable without pinning os.Rename
// fall-through to the clock branch.
func newServiceWithRepo(repo Repository, idGen func() string) *Service {
	if idGen == nil {
		idGen = func() string { return "test-id" }
	}
	return NewService(repo, idGen, func() time.Time { return time.Unix(0, 0).UTC() }, ServiceConfig{})
}

// seedProjectWithCheckout writes one minimal domain.Project record
// into the supplied fakeRepo with RepoBareRoot pointing at the test
// tempdir. Returns the project ID + the on-disk template path the
// service will write through.
func seedProjectWithCheckout(t *testing.T, repo *fakeRepo, projectID, bareRoot string) string {
	t.Helper()
	now := time.Unix(0, 0).UTC()
	repo.projects[projectID] = domain.Project{
		ID:           projectID,
		Slug:         "test-project",
		Name:         "Test Project",
		RepoBareRoot: bareRoot,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return filepath.Join(bareRoot, projectTemplateDir, projectTemplateFilename)
}

// TestService_SetProjectTemplate_RollbackOnUpdateProjectFailure exercises
// the rollback branch at template_service.go:494-511. The repo's
// UpdateProject returns an error AFTER the atomic rename has already
// landed the file on disk, forcing the rollback rename
// (dest → dest+".tillsyn-set-failed-<id>.toml") to fire.
//
// Asserts:
//   - returned error is non-nil and names the rollback path's
//     "tillsyn-set-failed-" sentinel substring;
//   - the original destination file no longer exists on disk;
//   - the failed-sentinel file exists at the deterministic suffix
//     location with byte-identical original template content;
//   - the failed-sentinel file is the only artifact named with the
//     suffix prefix (no orphan tmp-files leaked).
//
// Round-2 fix per workflow/drop_4c_5/BUILDER_QA_FALSIFICATION.md
// § "Droplet F.3.3 — Round 1" Attack 2: round-1's wire-only test in
// extended_tools_test.go stubs the in-band failure envelope at the
// adapter boundary; this test exercises the production
// os.Rename(dest, failedPath) branch that round-1 left uncovered.
func TestService_SetProjectTemplate_RollbackOnUpdateProjectFailure(t *testing.T) {
	t.Parallel()

	bareRoot := t.TempDir()
	persistErr := errors.New("simulated persist failure for rollback test")
	repo := &errOnUpdateProjectRepo{
		fakeRepo:         newFakeRepo(),
		updateProjectErr: persistErr,
	}

	const projectID = "test-project-rollback"
	const idSuffix = "rb-001"
	dest := seedProjectWithCheckout(t, repo.fakeRepo, projectID, bareRoot)

	svc := newServiceWithRepo(repo, func() string { return idSuffix })

	in := SetProjectTemplateInput{
		ProjectID:    projectID,
		TemplateTOML: []byte(minimalValidTemplateTOML),
		UpdatedBy:    "test-user",
		UpdatedType:  domain.ActorTypeUser,
	}
	out, err := svc.SetProjectTemplate(context.Background(), in)
	if err == nil {
		t.Fatalf("SetProjectTemplate err = nil, want non-nil rollback error; out = %+v", out)
	}

	// Assertion 1: error names the rollback path sentinel.
	if !strings.Contains(err.Error(), "tillsyn-set-failed-") {
		t.Fatalf("error %q does not name rollback path with 'tillsyn-set-failed-' substring", err.Error())
	}
	// Assertion 2: error wraps the original persist failure.
	if !errors.Is(err, persistErr) {
		t.Fatalf("error %q does not wrap original persistErr (errors.Is == false)", err.Error())
	}

	// Assertion 3: destination file MUST be absent — rollback moved it
	// aside.
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("dest %q still exists after rollback (Stat err = %v); rollback rename did not fire", dest, statErr)
	}

	// Assertion 4: failed-sentinel file exists at deterministic suffix
	// location and holds byte-identical original template content.
	wantFailedPath := dest + ".tillsyn-set-failed-" + idSuffix + ".toml"
	got, readErr := os.ReadFile(wantFailedPath)
	if readErr != nil {
		t.Fatalf("read rollback file %q: %v", wantFailedPath, readErr)
	}
	if string(got) != minimalValidTemplateTOML {
		t.Fatalf("rollback file content drift:\n  got  %q\n  want %q", string(got), minimalValidTemplateTOML)
	}

	// Assertion 5: only one rollback artifact exists for the
	// destination — no orphan .tmp file leaked. Glob across the dest's
	// parent directory for the canonical suffix pattern.
	matches, globErr := filepath.Glob(dest + ".tillsyn-set-failed-*.toml")
	if globErr != nil {
		t.Fatalf("glob rollback artifacts: %v", globErr)
	}
	if len(matches) != 1 {
		t.Fatalf("rollback-artifact glob len = %d, want 1; matches = %v", len(matches), matches)
	}
	if matches[0] != wantFailedPath {
		t.Fatalf("glob match[0] = %q, want %q", matches[0], wantFailedPath)
	}

	// Assertion 6: no leaked .tmp file from the WriteFile→Rename
	// sequence. The Rename completes successfully BEFORE UpdateProject
	// fires, so the tmp file must already be gone by the time
	// UpdateProject returns.
	tmpMatches, tmpErr := filepath.Glob(dest + ".tillsyn-set-*.tmp")
	if tmpErr != nil {
		t.Fatalf("glob tmp leftovers: %v", tmpErr)
	}
	if len(tmpMatches) != 0 {
		t.Fatalf("tmp leftovers found = %v, want none", tmpMatches)
	}
}

// TestService_SetProjectTemplate_RollbackRenameAlsoFails exercises the
// double-failure branch at template_service.go:502-507: persist fails
// AND the subsequent rollback rename fails too. Pre-creating a
// non-empty directory at the deterministic failed-path location forces
// os.Rename(dest, failedPath) to return an error (cannot overwrite a
// directory entry with a file).
//
// Asserts the returned error string carries the canonical "ROLLBACK
// ALSO FAILED" sentinel substring + the "manual cleanup required"
// adopter-facing instruction. Confirms the production path produces
// the louder error envelope when recovery itself fails.
func TestService_SetProjectTemplate_RollbackRenameAlsoFails(t *testing.T) {
	t.Parallel()

	bareRoot := t.TempDir()
	persistErr := errors.New("simulated persist failure for double-failure test")
	repo := &errOnUpdateProjectRepo{
		fakeRepo:         newFakeRepo(),
		updateProjectErr: persistErr,
	}

	const projectID = "test-project-double-fail"
	const idSuffix = "df-002"
	dest := seedProjectWithCheckout(t, repo.fakeRepo, projectID, bareRoot)

	// Pre-create the .tillsyn directory so the service does not error
	// on MkdirAll (it would create the dir itself, but we need to
	// place a directory blocker UNDER it before the service runs).
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(dest), err)
	}
	// Place a non-empty DIRECTORY at the deterministic failed-path
	// location. os.Rename(file, existing-non-empty-dir) returns
	// ENOTDIR / EISDIR / EEXIST depending on platform — always an
	// error, which is the test's lever.
	failedPath := dest + ".tillsyn-set-failed-" + idSuffix + ".toml"
	if err := os.MkdirAll(failedPath, 0o755); err != nil {
		t.Fatalf("MkdirAll blocker dir %q: %v", failedPath, err)
	}
	// Place a file inside the blocker directory so it is non-empty
	// (some platforms permit rename-over-empty-dir).
	if err := os.WriteFile(filepath.Join(failedPath, "blocker"), []byte("blocker"), 0o644); err != nil {
		t.Fatalf("seed blocker child file: %v", err)
	}

	svc := newServiceWithRepo(repo, func() string { return idSuffix })

	in := SetProjectTemplateInput{
		ProjectID:    projectID,
		TemplateTOML: []byte(minimalValidTemplateTOML),
		UpdatedBy:    "test-user",
		UpdatedType:  domain.ActorTypeUser,
	}
	out, err := svc.SetProjectTemplate(context.Background(), in)
	if err == nil {
		t.Fatalf("SetProjectTemplate err = nil, want non-nil double-failure error; out = %+v", out)
	}

	// Assertion 1: louder "ROLLBACK ALSO FAILED" sentinel.
	if !strings.Contains(err.Error(), "ROLLBACK ALSO FAILED") {
		t.Fatalf("error %q missing 'ROLLBACK ALSO FAILED' sentinel", err.Error())
	}
	// Assertion 2: dev-facing recovery instruction.
	if !strings.Contains(err.Error(), "manual cleanup required") {
		t.Fatalf("error %q missing 'manual cleanup required' guidance", err.Error())
	}
	// Assertion 3: original persist error still wrapped.
	if !errors.Is(err, persistErr) {
		t.Fatalf("error %q does not wrap original persistErr (errors.Is == false)", err.Error())
	}
}
