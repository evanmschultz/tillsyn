package dispatcher_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evanmschultz/tillsyn/internal/app/dispatcher"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// fixtureBundleItem returns a build action item with the minimum fields
// NewBundle reads (ID for input validation; Kind + Paths surface in the
// manifest payload via WriteManifest in callers).
func fixtureBundleItem() domain.ActionItem {
	return domain.ActionItem{
		ID:    "ai-build-bundle-1",
		Kind:  domain.KindBuild,
		Title: "DROPLET 4C F.7.1 BUNDLE FIXTURE",
		Paths: []string{"internal/app/dispatcher/spawn.go"},
	}
}

// TestNewBundleOSTempMode covers the canonical happy path: empty
// spawnTempRoot resolves to "os_tmp", and the resulting bundle root is
// under os.TempDir() with the conventional "tillsyn-spawn-" prefix.
func TestNewBundleOSTempMode(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	if bundle.Mode != dispatcher.SpawnTempRootOSTmp {
		t.Errorf("Bundle.Mode = %q, want %q", bundle.Mode, dispatcher.SpawnTempRootOSTmp)
	}
	if bundle.SpawnID == "" {
		t.Errorf("Bundle.SpawnID is empty; want UUID")
	}
	if bundle.StartedAt.IsZero() {
		t.Errorf("Bundle.StartedAt is zero; want NewBundle wall-clock time")
	}
	if bundle.Paths.Root == "" {
		t.Fatalf("Bundle.Paths.Root is empty; want absolute path")
	}
	// The bundle root MUST exist on disk after NewBundle returns.
	info, err := os.Stat(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("os.Stat(%q): %v", bundle.Paths.Root, err)
	}
	if !info.IsDir() {
		t.Fatalf("Bundle.Paths.Root = %q; want directory", bundle.Paths.Root)
	}
	// In os_tmp mode the root MUST live under os.TempDir().
	tempRoot := os.TempDir()
	if !strings.HasPrefix(bundle.Paths.Root, tempRoot) {
		t.Errorf("Bundle.Paths.Root = %q; want prefix %q", bundle.Paths.Root, tempRoot)
	}
	// And the basename should reflect the conventional prefix (the
	// MkdirTemp pattern interpolates random suffix bytes after the prefix).
	base := filepath.Base(bundle.Paths.Root)
	if !strings.HasPrefix(base, "tillsyn-spawn-") {
		t.Errorf("Bundle.Paths.Root basename = %q; want prefix %q", base, "tillsyn-spawn-")
	}
}

// TestNewBundleOSTempModeExplicitConstant verifies the explicit "os_tmp"
// string produces the same path layout as the empty-string default. Pins
// that the empty-string sentinel is functionally equivalent to the
// explicit constant.
func TestNewBundleOSTempModeExplicitConstant(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), dispatcher.SpawnTempRootOSTmp, "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	if bundle.Mode != dispatcher.SpawnTempRootOSTmp {
		t.Errorf("Bundle.Mode = %q, want %q", bundle.Mode, dispatcher.SpawnTempRootOSTmp)
	}
	if !strings.HasPrefix(bundle.Paths.Root, os.TempDir()) {
		t.Errorf("Bundle.Paths.Root = %q; want prefix %q", bundle.Paths.Root, os.TempDir())
	}
}

// TestNewBundleProjectMode covers the under-worktree path. NewBundle creates
// <projectRoot>/.tillsyn/spawns/<spawn-id>/ with parent dirs idempotent.
func TestNewBundleProjectMode(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), dispatcher.SpawnTempRootProject, projectRoot)
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	if bundle.Mode != dispatcher.SpawnTempRootProject {
		t.Errorf("Bundle.Mode = %q, want %q", bundle.Mode, dispatcher.SpawnTempRootProject)
	}
	wantParent := filepath.Join(projectRoot, ".tillsyn", "spawns")
	if filepath.Dir(bundle.Paths.Root) != wantParent {
		t.Errorf("filepath.Dir(Bundle.Paths.Root) = %q; want %q",
			filepath.Dir(bundle.Paths.Root), wantParent)
	}
	if filepath.Base(bundle.Paths.Root) != bundle.SpawnID {
		t.Errorf("filepath.Base(Bundle.Paths.Root) = %q; want %q",
			filepath.Base(bundle.Paths.Root), bundle.SpawnID)
	}
	// The bundle root MUST exist on disk.
	info, err := os.Stat(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("os.Stat(%q): %v", bundle.Paths.Root, err)
	}
	if !info.IsDir() {
		t.Fatalf("Bundle.Paths.Root = %q; want directory", bundle.Paths.Root)
	}
}

// TestNewBundleProjectModeRequiresProjectRoot verifies the input-validation
// guard: project mode without a projectRoot returns ErrInvalidBundleInput
// rather than creating .tillsyn/spawns/ at filesystem root.
func TestNewBundleProjectModeRequiresProjectRoot(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), dispatcher.SpawnTempRootProject, "")
	if err == nil {
		t.Fatalf("NewBundle() error = nil, want ErrInvalidBundleInput")
	}
	if !errors.Is(err, dispatcher.ErrInvalidBundleInput) {
		t.Fatalf("NewBundle() error = %v, want errors.Is(ErrInvalidBundleInput)", err)
	}
	if bundle.Paths.Root != "" {
		t.Errorf("Bundle.Paths.Root = %q; want empty on error", bundle.Paths.Root)
	}
}

// TestNewBundleRejectsUnknownSpawnTempRoot verifies the closed-enum guard
// in resolveSpawnTempRoot: any value outside {"", "os_tmp", "project"}
// surfaces as ErrInvalidBundleInput before any disk work happens.
func TestNewBundleRejectsUnknownSpawnTempRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
	}{
		{name: "totally bogus", val: "tmpfs"},
		{name: "case mismatch upper", val: "OS_TMP"},
		{name: "case mismatch capitalized", val: "Project"},
		{name: "whitespace padded", val: " os_tmp "},
		{name: "hyphen vs underscore", val: "os-tmp"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bundle, err := dispatcher.NewBundle(fixtureBundleItem(), tc.val, t.TempDir())
			if err == nil {
				t.Fatalf("NewBundle() error = nil, want ErrInvalidBundleInput")
			}
			if !errors.Is(err, dispatcher.ErrInvalidBundleInput) {
				t.Fatalf("NewBundle() error = %v, want errors.Is(ErrInvalidBundleInput)", err)
			}
			if !strings.Contains(err.Error(), tc.val) {
				t.Errorf("NewBundle() err = %q; want offending value %q in message", err.Error(), tc.val)
			}
			if bundle.Paths.Root != "" {
				t.Errorf("Bundle.Paths.Root = %q; want empty on error", bundle.Paths.Root)
			}
		})
	}
}

// TestNewBundleRejectsEmptyActionItemID verifies the input-validation guard
// for a missing action-item ID — without it the manifest's action_item_id
// field would be empty string, defeating orphan-scan correlation.
func TestNewBundleRejectsEmptyActionItemID(t *testing.T) {
	t.Parallel()

	item := fixtureBundleItem()
	item.ID = "   "

	bundle, err := dispatcher.NewBundle(item, "", "")
	if err == nil {
		t.Fatalf("NewBundle() error = nil, want ErrInvalidBundleInput")
	}
	if !errors.Is(err, dispatcher.ErrInvalidBundleInput) {
		t.Fatalf("NewBundle() error = %v, want errors.Is(ErrInvalidBundleInput)", err)
	}
	if bundle.Paths.Root != "" {
		t.Errorf("Bundle.Paths.Root = %q; want empty on error", bundle.Paths.Root)
	}
}

// TestBundlePathsAreUnderRoot pins the invariant that every BundlePaths
// non-empty field is a descendant of Root. F.7.8's orphan scanner relies on
// this so it can reap an entire bundle by removing Root.
func TestBundlePathsAreUnderRoot(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), dispatcher.SpawnTempRootOSTmp, "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	rootSep := bundle.Paths.Root + string(filepath.Separator)
	candidates := []struct {
		name string
		path string
	}{
		{"SystemPromptPath", bundle.Paths.SystemPromptPath},
		{"StreamLogPath", bundle.Paths.StreamLogPath},
		{"ManifestPath", bundle.Paths.ManifestPath},
		{"ContextDir", bundle.Paths.ContextDir},
	}
	for _, c := range candidates {
		if c.path == "" {
			continue
		}
		if !strings.HasPrefix(c.path, rootSep) {
			t.Errorf("Bundle.Paths.%s = %q; want under root %q", c.name, c.path, rootSep)
		}
	}
}

// TestBundleCleanupIdempotent verifies Cleanup is safe to call repeatedly:
// the first call removes the directory, subsequent calls are no-ops because
// os.RemoveAll treats a non-existent path as success.
func TestBundleCleanupIdempotent(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}

	if err := bundle.Cleanup(); err != nil {
		t.Fatalf("first Cleanup() error = %v, want nil", err)
	}
	if _, err := os.Stat(bundle.Paths.Root); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("after Cleanup, os.Stat(%q) error = %v; want os.ErrNotExist", bundle.Paths.Root, err)
	}
	if err := bundle.Cleanup(); err != nil {
		t.Errorf("second Cleanup() error = %v, want nil (idempotent)", err)
	}
}

// TestBundleCleanupZeroValueIsSafe verifies the zero-value Bundle's Cleanup
// is a no-op — important for callers that defer Cleanup before the
// NewBundle call has succeeded (defensive idiom).
func TestBundleCleanupZeroValueIsSafe(t *testing.T) {
	t.Parallel()

	var bundle dispatcher.Bundle
	if err := bundle.Cleanup(); err != nil {
		t.Errorf("zero-value Bundle.Cleanup() error = %v, want nil", err)
	}
}

// TestBundleWriteManifestRoundTrip verifies the manifest payload encodes
// every required field (spawn_id, action_item_id, kind, started_at, paths)
// and round-trips through json.Unmarshal back to an equivalent value.
func TestBundleWriteManifestRoundTrip(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	payload := dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: "ai-build-bundle-1",
		Kind:         domain.KindBuild,
		StartedAt:    bundle.StartedAt,
		Paths:        []string{"internal/app/dispatcher/spawn.go"},
	}
	if err := bundle.WriteManifest(payload); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	contents, err := os.ReadFile(bundle.Paths.ManifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", bundle.Paths.ManifestPath, err)
	}

	var decoded dispatcher.ManifestMetadata
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v\nfile contents:\n%s", err, contents)
	}

	if decoded.SpawnID != payload.SpawnID {
		t.Errorf("decoded.SpawnID = %q, want %q", decoded.SpawnID, payload.SpawnID)
	}
	if decoded.ActionItemID != payload.ActionItemID {
		t.Errorf("decoded.ActionItemID = %q, want %q", decoded.ActionItemID, payload.ActionItemID)
	}
	if decoded.Kind != payload.Kind {
		t.Errorf("decoded.Kind = %q, want %q", decoded.Kind, payload.Kind)
	}
	// time.Time round-trips through RFC 3339 — comparison via Equal handles
	// nanosecond precision drift across the JSON boundary.
	if !decoded.StartedAt.Equal(payload.StartedAt) {
		t.Errorf("decoded.StartedAt = %v, want %v (Equal)", decoded.StartedAt, payload.StartedAt)
	}
	if len(decoded.Paths) != len(payload.Paths) {
		t.Fatalf("len(decoded.Paths) = %d, want %d", len(decoded.Paths), len(payload.Paths))
	}
	for i, p := range payload.Paths {
		if decoded.Paths[i] != p {
			t.Errorf("decoded.Paths[%d] = %q, want %q", i, decoded.Paths[i], p)
		}
	}
}

// TestBundleWriteManifestKeysExactShape verifies the manifest JSON uses the
// exact key names declared in the struct tags (spawn_id, action_item_id,
// kind, started_at, paths). Pins the wire format so future additions land
// as new fields, not silent renames.
func TestBundleWriteManifestKeysExactShape(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	payload := dispatcher.ManifestMetadata{
		SpawnID:      "spawn-shape-test",
		ActionItemID: "ai-shape-test",
		Kind:         domain.KindBuild,
		StartedAt:    time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
		Paths:        []string{"a.go", "b.go"},
	}
	if err := bundle.WriteManifest(payload); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	contents, err := os.ReadFile(bundle.Paths.ManifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}

	var generic map[string]any
	if err := json.Unmarshal(contents, &generic); err != nil {
		t.Fatalf("json.Unmarshal: %v\ncontents:\n%s", err, contents)
	}

	wantKeys := []string{"spawn_id", "action_item_id", "kind", "started_at", "paths"}
	for _, k := range wantKeys {
		if _, ok := generic[k]; !ok {
			t.Errorf("manifest missing JSON key %q\nfull payload:\n%s", k, contents)
		}
	}
}

// TestBundleWriteManifestRejectsZeroValueBundle verifies WriteManifest fails
// when called on a zero-value Bundle (the ManifestPath is empty). Catches
// the defensive footgun where a caller defers WriteManifest before
// NewBundle has succeeded.
func TestBundleWriteManifestRejectsZeroValueBundle(t *testing.T) {
	t.Parallel()

	var bundle dispatcher.Bundle
	err := bundle.WriteManifest(dispatcher.ManifestMetadata{SpawnID: "x"})
	if err == nil {
		t.Fatalf("WriteManifest() error = nil, want ErrInvalidBundleInput")
	}
	if !errors.Is(err, dispatcher.ErrInvalidBundleInput) {
		t.Fatalf("WriteManifest() error = %v, want errors.Is(ErrInvalidBundleInput)", err)
	}
}

// TestNewBundleSpawnIDIsUUIDLike pins the contract that SpawnID is a UUID
// string. Format check is loose — the canonical google/uuid v4 form is
// 8-4-4-4-12 hex-character groups separated by hyphens, total 36 chars.
func TestNewBundleSpawnIDIsUUIDLike(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	if got := len(bundle.SpawnID); got != 36 {
		t.Errorf("len(Bundle.SpawnID) = %d, want 36 (UUID canonical form)", got)
	}
	if got := strings.Count(bundle.SpawnID, "-"); got != 4 {
		t.Errorf("Bundle.SpawnID hyphen count = %d, want 4 (UUID canonical form)", got)
	}
}

// TestNewBundleManifestClaudePIDDefaultsToZero pins the F.7.1 contract that
// NewBundle + WriteManifest leave ClaudePID at zero — the "spawn not yet
// started, leave alone" signal F.7.8's orphan scan keys off per spawn
// architecture memory §8. The first non-zero write happens via
// UpdateManifestPID after `cmd.Start()` returns success (F.7.8 territory).
func TestNewBundleManifestClaudePIDDefaultsToZero(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	payload := dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: "ai-build-bundle-1",
		Kind:         domain.KindBuild,
		StartedAt:    bundle.StartedAt,
		Paths:        []string{"a.go"},
	}
	if err := bundle.WriteManifest(payload); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	decoded, err := dispatcher.ReadManifest(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v, want nil", err)
	}
	if decoded.ClaudePID != 0 {
		t.Errorf("decoded.ClaudePID = %d, want 0 (default zero per memory §8)", decoded.ClaudePID)
	}
	if decoded.BundlePath != bundle.Paths.Root {
		t.Errorf("decoded.BundlePath = %q, want %q (auto-populated from receiver Root)",
			decoded.BundlePath, bundle.Paths.Root)
	}
}

// TestUpdateManifestPIDRoundTrip pins the canonical F.7.8 invocation flow:
// NewBundle → WriteManifest → UpdateManifestPID(12345) → ReadManifest must
// return ClaudePID == 12345. The PID flips from zero to non-zero exactly once
// per spawn lifecycle.
func TestUpdateManifestPIDRoundTrip(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	if err := bundle.WriteManifest(dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: "ai-build-bundle-1",
		Kind:         domain.KindBuild,
		StartedAt:    bundle.StartedAt,
		Paths:        []string{"a.go"},
	}); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	const wantPID = 12345
	if err := bundle.UpdateManifestPID(wantPID); err != nil {
		t.Fatalf("UpdateManifestPID(%d) error = %v, want nil", wantPID, err)
	}

	decoded, err := dispatcher.ReadManifest(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v, want nil", err)
	}
	if decoded.ClaudePID != wantPID {
		t.Errorf("decoded.ClaudePID = %d, want %d", decoded.ClaudePID, wantPID)
	}
}

// TestReadManifestHappyPath pins the inverse symmetry of WriteManifest:
// every field round-trips identically through MarshalIndent + Unmarshal
// (excluding BundlePath, which WriteManifest auto-populates from the
// receiver Root regardless of caller input).
func TestReadManifestHappyPath(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	payload := dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: "ai-read-test",
		Kind:         domain.KindBuild,
		ClaudePID:    0,
		StartedAt:    time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
		Paths:        []string{"x.go", "y.go"},
	}
	if err := bundle.WriteManifest(payload); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	decoded, err := dispatcher.ReadManifest(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v, want nil", err)
	}
	if decoded.SpawnID != payload.SpawnID {
		t.Errorf("decoded.SpawnID = %q, want %q", decoded.SpawnID, payload.SpawnID)
	}
	if decoded.ActionItemID != payload.ActionItemID {
		t.Errorf("decoded.ActionItemID = %q, want %q", decoded.ActionItemID, payload.ActionItemID)
	}
	if decoded.Kind != payload.Kind {
		t.Errorf("decoded.Kind = %q, want %q", decoded.Kind, payload.Kind)
	}
	if decoded.ClaudePID != payload.ClaudePID {
		t.Errorf("decoded.ClaudePID = %d, want %d", decoded.ClaudePID, payload.ClaudePID)
	}
	if !decoded.StartedAt.Equal(payload.StartedAt) {
		t.Errorf("decoded.StartedAt = %v, want %v (Equal)", decoded.StartedAt, payload.StartedAt)
	}
	if len(decoded.Paths) != len(payload.Paths) {
		t.Fatalf("len(decoded.Paths) = %d, want %d", len(decoded.Paths), len(payload.Paths))
	}
	for i, p := range payload.Paths {
		if decoded.Paths[i] != p {
			t.Errorf("decoded.Paths[%d] = %q, want %q", i, decoded.Paths[i], p)
		}
	}
	if decoded.BundlePath != bundle.Paths.Root {
		t.Errorf("decoded.BundlePath = %q, want %q (auto-populated by WriteManifest)",
			decoded.BundlePath, bundle.Paths.Root)
	}
}

// TestReadManifestMissingFile pins the error contract for absent
// manifest.json: the returned error must satisfy errors.Is(err, os.ErrNotExist)
// so F.7.8's orphan scan can use the standard predicate to flag bundles
// whose dispatcher crashed before WriteManifest fired.
func TestReadManifestMissingFile(t *testing.T) {
	t.Parallel()

	// Create a bundle root that explicitly does NOT contain a manifest.json.
	bundleRoot := t.TempDir()
	_, err := dispatcher.ReadManifest(bundleRoot)
	if err == nil {
		t.Fatalf("ReadManifest() error = nil, want os.ErrNotExist")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadManifest() error = %v, want errors.Is(os.ErrNotExist)", err)
	}
}

// TestReadManifestMalformedJSON pins the malformed-payload branch: the
// function returns a non-nil structured error with a "decode manifest"
// substring so forensic tooling can log + skip without crashing.
func TestReadManifestMalformedJSON(t *testing.T) {
	t.Parallel()

	bundleRoot := t.TempDir()
	manifestPath := filepath.Join(bundleRoot, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{this is not valid json"), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	_, err := dispatcher.ReadManifest(bundleRoot)
	if err == nil {
		t.Fatalf("ReadManifest() error = nil, want decode error")
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadManifest() error = %v; should NOT be os.ErrNotExist (file exists, JSON is malformed)", err)
	}
	if !strings.Contains(err.Error(), "decode manifest") {
		t.Errorf("ReadManifest() error = %q; want substring %q", err.Error(), "decode manifest")
	}
}

// TestUpdateManifestPIDPreservesOtherFields pins the no-side-effects
// guarantee: UpdateManifestPID must mutate ONLY ClaudePID. Every other
// field (SpawnID, ActionItemID, Kind, StartedAt, Paths, BundlePath) survives
// the read-mutate-write cycle unchanged.
func TestUpdateManifestPIDPreservesOtherFields(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	original := dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: "ai-preserve-test",
		Kind:         domain.KindBuild,
		ClaudePID:    0,
		StartedAt:    time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
		Paths:        []string{"alpha.go", "beta.go", "gamma.go"},
	}
	if err := bundle.WriteManifest(original); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	preUpdate, err := dispatcher.ReadManifest(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("pre-update ReadManifest() error = %v, want nil", err)
	}

	const newPID = 98765
	if err := bundle.UpdateManifestPID(newPID); err != nil {
		t.Fatalf("UpdateManifestPID(%d) error = %v, want nil", newPID, err)
	}

	postUpdate, err := dispatcher.ReadManifest(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("post-update ReadManifest() error = %v, want nil", err)
	}

	if postUpdate.ClaudePID != newPID {
		t.Errorf("postUpdate.ClaudePID = %d, want %d", postUpdate.ClaudePID, newPID)
	}
	if postUpdate.SpawnID != preUpdate.SpawnID {
		t.Errorf("SpawnID changed: %q -> %q", preUpdate.SpawnID, postUpdate.SpawnID)
	}
	if postUpdate.ActionItemID != preUpdate.ActionItemID {
		t.Errorf("ActionItemID changed: %q -> %q", preUpdate.ActionItemID, postUpdate.ActionItemID)
	}
	if postUpdate.Kind != preUpdate.Kind {
		t.Errorf("Kind changed: %q -> %q", preUpdate.Kind, postUpdate.Kind)
	}
	if !postUpdate.StartedAt.Equal(preUpdate.StartedAt) {
		t.Errorf("StartedAt changed: %v -> %v", preUpdate.StartedAt, postUpdate.StartedAt)
	}
	if postUpdate.BundlePath != preUpdate.BundlePath {
		t.Errorf("BundlePath changed: %q -> %q", preUpdate.BundlePath, postUpdate.BundlePath)
	}
	if len(postUpdate.Paths) != len(preUpdate.Paths) {
		t.Fatalf("len(Paths) changed: %d -> %d", len(preUpdate.Paths), len(postUpdate.Paths))
	}
	for i := range preUpdate.Paths {
		if postUpdate.Paths[i] != preUpdate.Paths[i] {
			t.Errorf("Paths[%d] changed: %q -> %q", i, preUpdate.Paths[i], postUpdate.Paths[i])
		}
	}
}

// TestBundleWriteManifestPreservesCLIKind pins the F.7.17.6 round-trip
// contract: WriteManifest with CLIKind="claude" → ReadManifest returns
// CLIKind="claude". F.7.8's orphan scan keys adapter-specific liveness
// checks off this field, so any silent drop on the JSON boundary would
// break orphan-scan routing.
func TestBundleWriteManifestPreservesCLIKind(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	payload := dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: "ai-clikind-roundtrip",
		Kind:         domain.KindBuild,
		CLIKind:      "claude",
		StartedAt:    bundle.StartedAt,
		Paths:        []string{"a.go"},
	}
	if err := bundle.WriteManifest(payload); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	decoded, err := dispatcher.ReadManifest(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v, want nil", err)
	}
	if decoded.CLIKind != "claude" {
		t.Errorf("decoded.CLIKind = %q, want %q", decoded.CLIKind, "claude")
	}

	// Pin the wire-format JSON key explicitly — the spawn architecture memory
	// §2 mandates `cli_kind` (snake_case) and any silent rename to e.g.
	// `cliKind` would break the cross-CLI manifest contract.
	contents, err := os.ReadFile(bundle.Paths.ManifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(contents, &generic); err != nil {
		t.Fatalf("json.Unmarshal: %v\ncontents:\n%s", err, contents)
	}
	if got, ok := generic["cli_kind"]; !ok {
		t.Errorf("manifest missing JSON key %q\nfull payload:\n%s", "cli_kind", contents)
	} else if got != "claude" {
		t.Errorf("manifest cli_kind = %v, want %q", got, "claude")
	}
}

// TestBundleWriteManifestEmptyCLIKindIsExplicit pins the F.7.17.6 contract
// that an empty CLIKind round-trips as the empty string AND the JSON key
// stays present. The struct tag deliberately omits `omitempty` so legacy
// bundles authored before this field landed surface an explicit empty
// string to F.7.8's orphan scan rather than a missing key. A silent drift
// to `omitempty` would break the orphan scan's "missing CLI" detection.
func TestBundleWriteManifestEmptyCLIKindIsExplicit(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	payload := dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: "ai-empty-clikind",
		Kind:         domain.KindBuild,
		CLIKind:      "", // explicit empty — not "claude"
		StartedAt:    bundle.StartedAt,
		Paths:        []string{"a.go"},
	}
	if err := bundle.WriteManifest(payload); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	decoded, err := dispatcher.ReadManifest(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v, want nil", err)
	}
	if decoded.CLIKind != "" {
		t.Errorf("decoded.CLIKind = %q, want %q (empty string round-trip)", decoded.CLIKind, "")
	}

	// The JSON key MUST be present even though the value is empty — the
	// no-omitempty contract is what makes legacy-bundle detection sound.
	contents, err := os.ReadFile(bundle.Paths.ManifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(contents, &generic); err != nil {
		t.Fatalf("json.Unmarshal: %v\ncontents:\n%s", err, contents)
	}
	got, ok := generic["cli_kind"]
	if !ok {
		t.Fatalf("manifest missing JSON key %q (no-omitempty contract violated)\nfull payload:\n%s", "cli_kind", contents)
	}
	if got != "" {
		t.Errorf("manifest cli_kind = %v, want %q", got, "")
	}
}

// TestUpdateManifestPIDPreservesCLIKind pins the F.7.17.6 contract that
// UpdateManifestPID's read-mutate-write cycle preserves CLIKind verbatim.
// Without this, the orphan scan (F.7.8) would lose adapter-routing
// information the moment the dispatcher's monitor wrote a real PID into
// the manifest after `cmd.Start()`.
func TestUpdateManifestPIDPreservesCLIKind(t *testing.T) {
	t.Parallel()

	bundle, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = bundle.Cleanup() })

	if err := bundle.WriteManifest(dispatcher.ManifestMetadata{
		SpawnID:      bundle.SpawnID,
		ActionItemID: "ai-clikind-pid",
		Kind:         domain.KindBuild,
		CLIKind:      "claude",
		StartedAt:    bundle.StartedAt,
		Paths:        []string{"a.go"},
	}); err != nil {
		t.Fatalf("WriteManifest() error = %v, want nil", err)
	}

	const wantPID = 31415
	if err := bundle.UpdateManifestPID(wantPID); err != nil {
		t.Fatalf("UpdateManifestPID(%d) error = %v, want nil", wantPID, err)
	}

	decoded, err := dispatcher.ReadManifest(bundle.Paths.Root)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v, want nil", err)
	}
	if decoded.ClaudePID != wantPID {
		t.Errorf("decoded.ClaudePID = %d, want %d", decoded.ClaudePID, wantPID)
	}
	if decoded.CLIKind != "claude" {
		t.Errorf("decoded.CLIKind = %q, want %q (UpdateManifestPID must preserve CLIKind)",
			decoded.CLIKind, "claude")
	}
}

// TestNewBundleSpawnIDsUnique verifies two calls in the same process produce
// distinct SpawnIDs — defensive sanity check on the UUID generator.
func TestNewBundleSpawnIDsUnique(t *testing.T) {
	t.Parallel()

	b1, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() #1 error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = b1.Cleanup() })

	b2, err := dispatcher.NewBundle(fixtureBundleItem(), "", "")
	if err != nil {
		t.Fatalf("NewBundle() #2 error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = b2.Cleanup() })

	if b1.SpawnID == b2.SpawnID {
		t.Errorf("two NewBundle calls produced same SpawnID %q", b1.SpawnID)
	}
	if b1.Paths.Root == b2.Paths.Root {
		t.Errorf("two NewBundle calls produced same Root %q", b1.Paths.Root)
	}
}

// TestEnsureSpawnsGitignoredOSTempIsNoop covers the F.7.7 behavior matrix's
// short-circuit row: when spawn_temp_root resolves to OS-temp mode the
// helper returns nil without inspecting the worktree or creating any
// .gitignore. Pins the contract that os_tmp mode never modifies project
// state.
func TestEnsureSpawnsGitignoredOSTempIsNoop(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	// Empty string and explicit "os_tmp" should both short-circuit.
	for _, mode := range []string{"", dispatcher.SpawnTempRootOSTmp} {
		mode := mode
		t.Run("mode="+mode, func(t *testing.T) {
			t.Parallel()
			if err := dispatcher.EnsureSpawnsGitignored(projectRoot, mode); err != nil {
				t.Fatalf("EnsureSpawnsGitignored(%q, %q) error = %v, want nil", projectRoot, mode, err)
			}
			gitignorePath := filepath.Join(projectRoot, ".gitignore")
			if _, err := os.Stat(gitignorePath); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("os.Stat(%q) error = %v; want os.ErrNotExist (os_tmp must not create .gitignore)", gitignorePath, err)
			}
		})
	}
}

// TestEnsureSpawnsGitignoredCreatesWhenMissing covers the project-mode happy
// path with no pre-existing .gitignore: the helper creates one with the
// canonical entry and a trailing newline.
func TestEnsureSpawnsGitignoredCreatesWhenMissing(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	if err := dispatcher.EnsureSpawnsGitignored(projectRoot, dispatcher.SpawnTempRootProject); err != nil {
		t.Fatalf("EnsureSpawnsGitignored() error = %v, want nil", err)
	}

	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	contents, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want .gitignore created", gitignorePath, err)
	}
	got := string(contents)
	want := ".tillsyn/spawns/\n"
	if got != want {
		t.Errorf(".gitignore contents = %q, want %q", got, want)
	}
}

// TestEnsureSpawnsGitignoredIdempotentRecall verifies the second invocation
// produces no observable change — file mtime + contents stay identical.
func TestEnsureSpawnsGitignoredIdempotentRecall(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	if err := dispatcher.EnsureSpawnsGitignored(projectRoot, dispatcher.SpawnTempRootProject); err != nil {
		t.Fatalf("first EnsureSpawnsGitignored() error = %v, want nil", err)
	}
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	firstContents, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("first os.ReadFile() error = %v", err)
	}

	if err := dispatcher.EnsureSpawnsGitignored(projectRoot, dispatcher.SpawnTempRootProject); err != nil {
		t.Fatalf("second EnsureSpawnsGitignored() error = %v, want nil", err)
	}
	secondContents, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("second os.ReadFile() error = %v", err)
	}

	if string(firstContents) != string(secondContents) {
		t.Errorf(".gitignore contents drifted across re-call:\nfirst:  %q\nsecond: %q",
			firstContents, secondContents)
	}
	// Should still be exactly the canonical single-line form, NOT doubled.
	want := ".tillsyn/spawns/\n"
	if string(secondContents) != want {
		t.Errorf("after idempotent re-call, .gitignore = %q, want %q (no double-append)",
			secondContents, want)
	}
}

// TestEnsureSpawnsGitignoredAppendsToExistingEntries verifies that a project
// with a pre-populated .gitignore retains every existing entry and gets the
// new one appended at the end with proper newline framing.
func TestEnsureSpawnsGitignoredAppendsToExistingEntries(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	preExisting := "*.log\nnode_modules/\n.DS_Store\n"
	if err := os.WriteFile(gitignorePath, []byte(preExisting), 0o644); err != nil {
		t.Fatalf("seed os.WriteFile: %v", err)
	}

	if err := dispatcher.EnsureSpawnsGitignored(projectRoot, dispatcher.SpawnTempRootProject); err != nil {
		t.Fatalf("EnsureSpawnsGitignored() error = %v, want nil", err)
	}

	contents, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	got := string(contents)
	want := preExisting + ".tillsyn/spawns/\n"
	if got != want {
		t.Errorf(".gitignore contents = %q, want %q", got, want)
	}

	// Each pre-existing entry is still recognizable line-for-line.
	for _, entry := range []string{"*.log", "node_modules/", ".DS_Store", ".tillsyn/spawns/"} {
		if !strings.Contains(got, entry+"\n") {
			t.Errorf("post-append .gitignore missing entry %q\nfull contents:\n%s", entry, got)
		}
	}
}

// TestEnsureSpawnsGitignoredRecognizesAnchoredVariant verifies the
// idempotency check treats `/.tillsyn/spawns/` (the leading-slash anchored
// form devs sometimes hand-write) as already-ignored. Without this the
// helper would double-append the unanchored form alongside.
func TestEnsureSpawnsGitignoredRecognizesAnchoredVariant(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	preExisting := "*.log\n/.tillsyn/spawns/\nbuild/\n"
	if err := os.WriteFile(gitignorePath, []byte(preExisting), 0o644); err != nil {
		t.Fatalf("seed os.WriteFile: %v", err)
	}

	if err := dispatcher.EnsureSpawnsGitignored(projectRoot, dispatcher.SpawnTempRootProject); err != nil {
		t.Fatalf("EnsureSpawnsGitignored() error = %v, want nil", err)
	}

	contents, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(contents) != preExisting {
		t.Errorf(".gitignore contents = %q, want unchanged %q (anchored form already present)",
			contents, preExisting)
	}
	// Must NOT have appended an unanchored duplicate.
	if strings.Count(string(contents), ".tillsyn/spawns/") != 1 {
		t.Errorf("expected exactly 1 occurrence of .tillsyn/spawns/, got %d:\n%s",
			strings.Count(string(contents), ".tillsyn/spawns/"), contents)
	}
}

// TestEnsureSpawnsGitignoredHandlesMissingTrailingNewline verifies the
// append path correctly inserts a newline before the new entry when the
// existing file does NOT end with one — without this guard the new entry
// would be concatenated onto the previous line.
func TestEnsureSpawnsGitignoredHandlesMissingTrailingNewline(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	// Note: NO trailing newline.
	preExisting := "*.log\nnode_modules/"
	if err := os.WriteFile(gitignorePath, []byte(preExisting), 0o644); err != nil {
		t.Fatalf("seed os.WriteFile: %v", err)
	}

	if err := dispatcher.EnsureSpawnsGitignored(projectRoot, dispatcher.SpawnTempRootProject); err != nil {
		t.Fatalf("EnsureSpawnsGitignored() error = %v, want nil", err)
	}

	contents, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	got := string(contents)
	// "node_modules/" must be terminated by a newline before ".tillsyn/spawns/"
	// — the bug this guards against would produce "node_modules/.tillsyn/spawns/".
	want := "*.log\nnode_modules/\n.tillsyn/spawns/\n"
	if got != want {
		t.Errorf(".gitignore contents = %q, want %q", got, want)
	}
	if strings.Contains(got, "node_modules/.tillsyn/spawns/") {
		t.Errorf("regression: missing-newline guard failed; got concatenated entry:\n%s", got)
	}
}

// TestEnsureSpawnsGitignoredRejectsEmptyProjectRootInProjectMode verifies the
// input-validation guard for project mode: an empty projectRoot is a
// programming error (the dispatcher should always have a worktree when
// project-mode bundles are in use), so the helper returns
// ErrInvalidBundleInput rather than silently writing to filesystem root.
func TestEnsureSpawnsGitignoredRejectsEmptyProjectRootInProjectMode(t *testing.T) {
	t.Parallel()

	err := dispatcher.EnsureSpawnsGitignored("", dispatcher.SpawnTempRootProject)
	if err == nil {
		t.Fatalf("EnsureSpawnsGitignored() error = nil, want ErrInvalidBundleInput")
	}
	if !errors.Is(err, dispatcher.ErrInvalidBundleInput) {
		t.Fatalf("EnsureSpawnsGitignored() error = %v, want errors.Is(ErrInvalidBundleInput)", err)
	}
}
