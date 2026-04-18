package tui

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestFilePickerSpecialization_AppendFileRefsBuildsResourceRef verifies the
// file-picker accept helper produces one ResourceRef per regular file with
// Tags[0]=="file" and ResourceType=="local_file" (PLAN §17.1 file semantics).
func TestFilePickerSpecialization_AppendFileRefsBuildsResourceRef(t *testing.T) {
	root := t.TempDir()
	rel := "cmd/till/main.go"
	mustMkdir(t, filepath.Join(root, filepath.Dir(rel)))
	mustTouch(t, filepath.Join(root, rel))

	meta := domain.TaskMetadata{}
	selections := []filePickerEntry{
		{Name: "main.go", Path: filepath.Join(root, rel), IsDir: false},
	}
	updated := appendFileResourceRefs(meta, root, selections)

	if len(updated.ResourceRefs) != 1 {
		t.Fatalf("expected 1 ref after append, got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
	ref := updated.ResourceRefs[0]
	if ref.ResourceType != domain.ResourceTypeLocalFile {
		t.Fatalf("ref.ResourceType = %q, want local_file", ref.ResourceType)
	}
	if len(ref.Tags) == 0 || ref.Tags[0] != "file" {
		t.Fatalf("ref.Tags[0] = %v, want 'file'", ref.Tags)
	}
	if ref.PathMode != domain.PathModeRelative {
		t.Fatalf("ref.PathMode = %q, want relative (under project root)", ref.PathMode)
	}
	if ref.Location != filepath.ToSlash(rel) {
		t.Fatalf("ref.Location = %q, want %q", ref.Location, filepath.ToSlash(rel))
	}
}

// TestFilePickerSpecialization_AppendFileRefsSkipsNonexistent verifies paths
// that do not exist on disk are dropped silently (PLAN §17.1 `exist` rule).
func TestFilePickerSpecialization_AppendFileRefsSkipsNonexistent(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "does-not-exist.go")
	meta := domain.TaskMetadata{}
	selections := []filePickerEntry{
		{Name: "does-not-exist.go", Path: missing, IsDir: false},
	}
	updated := appendFileResourceRefs(meta, root, selections)
	if len(updated.ResourceRefs) != 0 {
		t.Fatalf("expected 0 refs for nonexistent path, got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
}

// TestFilePickerSpecialization_AppendFileRefsSkipsDirs verifies directories are
// dropped silently — file-picker semantics only admit regular files.
func TestFilePickerSpecialization_AppendFileRefsSkipsDirs(t *testing.T) {
	root := t.TempDir()
	dirRel := "internal/domain"
	mustMkdir(t, filepath.Join(root, dirRel))

	meta := domain.TaskMetadata{}
	selections := []filePickerEntry{
		// Caller may have flagged IsDir; helper must still gate on os.Stat.
		{Name: "domain", Path: filepath.Join(root, dirRel), IsDir: true},
	}
	updated := appendFileResourceRefs(meta, root, selections)
	if len(updated.ResourceRefs) != 0 {
		t.Fatalf("expected 0 refs for directory selection, got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
}

// TestFilePickerSpecialization_AppendFileRefsForcesFileSemantics verifies a
// caller-supplied IsDir=true flag does NOT leak local_dir ResourceType into
// the built ref when the path is actually a regular file on disk. Defends
// against future refactors that might pass through IsDir unchecked.
func TestFilePickerSpecialization_AppendFileRefsForcesFileSemantics(t *testing.T) {
	root := t.TempDir()
	rel := "some-file.go"
	mustTouch(t, filepath.Join(root, rel))

	meta := domain.TaskMetadata{}
	selections := []filePickerEntry{
		{Name: "some-file.go", Path: filepath.Join(root, rel), IsDir: true},
	}
	updated := appendFileResourceRefs(meta, root, selections)
	if len(updated.ResourceRefs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(updated.ResourceRefs))
	}
	if updated.ResourceRefs[0].ResourceType != domain.ResourceTypeLocalFile {
		t.Fatalf("ResourceType = %q, want local_file regardless of IsDir flag", updated.ResourceRefs[0].ResourceType)
	}
}

// TestFilePickerSpecialization_DefaultAllFiles simulates an accept with an
// empty selection set and verifies the default-all-files branch (selectedEntries
// returns nil → caller uses visibleEntries() → every regular file lands).
//
// The branch logic is encoded in the accept wiring in P4; here we prove the
// underlying primitive: given a visibleEntries snapshot, appendFileResourceRefs
// produces one ref per regular file and skips directories / the parent link.
func TestFilePickerSpecialization_DefaultAllFiles(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "docs"))
	mustTouch(t, filepath.Join(root, "alpha.go"))
	mustTouch(t, filepath.Join(root, "beta.go"))
	mustTouch(t, filepath.Join(root, "gamma.go"))

	entries, _, err := listFilePickerEntries(root, root)
	if err != nil {
		t.Fatalf("listFilePickerEntries() error = %v", err)
	}
	// Simulate accept-without-narrowing: no selection → callers pass the full
	// visibleEntries slice. The helper itself filters dirs and non-regulars.
	meta := domain.TaskMetadata{}
	updated := appendFileResourceRefs(meta, root, entries)

	if len(updated.ResourceRefs) != 3 {
		t.Fatalf("expected 3 file refs (alpha/beta/gamma), got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
	locations := make([]string, 0, len(updated.ResourceRefs))
	for _, ref := range updated.ResourceRefs {
		if ref.ResourceType != domain.ResourceTypeLocalFile {
			t.Fatalf("ref has ResourceType = %q, want local_file", ref.ResourceType)
		}
		if len(ref.Tags) == 0 || ref.Tags[0] != "file" {
			t.Fatalf("ref.Tags[0] = %v, want 'file'", ref.Tags)
		}
		locations = append(locations, ref.Location)
	}
	for _, needed := range []string{"alpha.go", "beta.go", "gamma.go"} {
		if !contains(locations, needed) {
			t.Fatalf("default-all-files missing %q in %v", needed, locations)
		}
	}
}

// TestFilePickerSpecialization_NarrowedSelection simulates an accept with a
// non-empty selection set — only those entries land. The helper is selection-
// agnostic; the caller passes whatever slice represents the user's intent, so
// the test proves the helper honors the caller-provided subset.
func TestFilePickerSpecialization_NarrowedSelection(t *testing.T) {
	root := t.TempDir()
	mustTouch(t, filepath.Join(root, "alpha.go"))
	mustTouch(t, filepath.Join(root, "beta.go"))
	mustTouch(t, filepath.Join(root, "gamma.go"))

	// User multi-selected only alpha and gamma; caller passes selectedEntries.
	selections := []filePickerEntry{
		{Name: "alpha.go", Path: filepath.Join(root, "alpha.go"), IsDir: false},
		{Name: "gamma.go", Path: filepath.Join(root, "gamma.go"), IsDir: false},
	}
	meta := domain.TaskMetadata{}
	updated := appendFileResourceRefs(meta, root, selections)

	if len(updated.ResourceRefs) != 2 {
		t.Fatalf("expected 2 refs for narrowed selection, got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
	locations := []string{updated.ResourceRefs[0].Location, updated.ResourceRefs[1].Location}
	if !contains(locations, "alpha.go") || !contains(locations, "gamma.go") {
		t.Fatalf("narrowed selection missing expected entries; got %v", locations)
	}
	if contains(locations, "beta.go") {
		t.Fatalf("narrowed selection leaked unselected beta.go; got %v", locations)
	}
}

// TestFilePickerSpecialization_PreservesPriorResourceRefs verifies existing
// ResourceRefs on meta (including path-tagged ones from a previous path-picker
// accept) are preserved unmutated when the file-picker appends new refs.
func TestFilePickerSpecialization_PreservesPriorResourceRefs(t *testing.T) {
	root := t.TempDir()
	rel := "new-file.go"
	mustTouch(t, filepath.Join(root, rel))

	priorPath := domain.ResourceRef{
		ResourceType: domain.ResourceTypeLocalFile,
		Location:     "existing/path.go",
		PathMode:     domain.PathModeRelative,
		BaseAlias:    "project_root",
		Tags:         []string{"path"},
	}
	priorURL := domain.ResourceRef{
		ResourceType: domain.ResourceTypeURL,
		Location:     "https://example.com/docs",
		PathMode:     domain.PathModeAbsolute,
		Tags:         []string{"doc"},
	}
	meta := domain.TaskMetadata{
		ResourceRefs: []domain.ResourceRef{priorPath, priorURL},
	}
	selections := []filePickerEntry{
		{Name: "new-file.go", Path: filepath.Join(root, rel), IsDir: false},
	}
	updated := appendFileResourceRefs(meta, root, selections)

	if len(updated.ResourceRefs) != 3 {
		t.Fatalf("expected 3 refs (2 prior + 1 new), got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
	if !reflect.DeepEqual(updated.ResourceRefs[0], priorPath) {
		t.Fatalf("prior path ref mutated: got %#v, want %#v", updated.ResourceRefs[0], priorPath)
	}
	if !reflect.DeepEqual(updated.ResourceRefs[1], priorURL) {
		t.Fatalf("prior url ref mutated: got %#v, want %#v", updated.ResourceRefs[1], priorURL)
	}
	newRef := updated.ResourceRefs[2]
	if len(newRef.Tags) == 0 || newRef.Tags[0] != "file" {
		t.Fatalf("new ref Tags[0] = %v, want 'file'", newRef.Tags)
	}
	if newRef.Location != rel {
		t.Fatalf("new ref Location = %q, want %q", newRef.Location, rel)
	}
}

// TestFilePickerSpecialization_Dedupes verifies appending a file path that is
// already present on Metadata.ResourceRefs (same ResourceType + Location +
// PathMode) does not create a duplicate — appendResourceRefIfMissing semantics
// extend to file-picker accepts.
func TestFilePickerSpecialization_Dedupes(t *testing.T) {
	root := t.TempDir()
	rel := "dup.go"
	mustTouch(t, filepath.Join(root, rel))

	meta := domain.TaskMetadata{
		ResourceRefs: []domain.ResourceRef{
			{
				ResourceType: domain.ResourceTypeLocalFile,
				Location:     rel,
				PathMode:     domain.PathModeRelative,
				BaseAlias:    "project_root",
				Tags:         []string{"file"},
			},
		},
	}
	selections := []filePickerEntry{
		{Name: "dup.go", Path: filepath.Join(root, rel), IsDir: false},
	}
	updated := appendFileResourceRefs(meta, root, selections)
	if len(updated.ResourceRefs) != 1 {
		t.Fatalf("duplicate file appended — want 1 ref, got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
}

// TestFilePickerSpecialization_TitlesAndSubtitles verifies the File variant has
// distinct render text so the chrome announces the right mode to the user.
func TestFilePickerSpecialization_TitlesAndSubtitles(t *testing.T) {
	if got := filePickerTitleFor(filePickerModeFile); got != "Add Files" {
		t.Fatalf("filePickerTitleFor(file) = %q, want 'Add Files'", got)
	}
	sub := filePickerSubtitleFor(filePickerModeFile)
	if sub == "" {
		t.Fatal("filePickerSubtitleFor(file) is empty")
	}
	// Subtitle must telegraph file-tag semantics so the user understands the
	// difference from path-picker.
	for _, needle := range []string{"file", "Tags[0]"} {
		if !containsSubstring(sub, needle) {
			t.Fatalf("filePickerSubtitleFor(file) = %q missing %q", sub, needle)
		}
	}
}

// TestFilePickerSpecialization_ModeEnumDistinct defends the enum ordering —
// filePickerModeFile must not collide with Path or None.
func TestFilePickerSpecialization_ModeEnumDistinct(t *testing.T) {
	if filePickerModeFile == filePickerModeNone {
		t.Fatal("filePickerModeFile aliases filePickerModeNone")
	}
	if filePickerModeFile == filePickerModePath {
		t.Fatal("filePickerModeFile aliases filePickerModePath")
	}
}

// containsSubstring is a small local helper — the test-only contains() in
// file_picker_core_test.go compares exact elements, not substrings.
func containsSubstring(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOfSubstring(haystack, needle) >= 0)
}

// indexOfSubstring returns the index of needle in haystack, or -1 when absent.
func indexOfSubstring(haystack, needle string) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
