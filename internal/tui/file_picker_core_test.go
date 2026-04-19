package tui

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// TestFilePickerCore_TraversalFiltersDotfiles verifies listFilePickerEntries
// drops .git / .DS_Store / leading-dot directories while retaining regular
// files and directories.
func TestFilePickerCore_TraversalFiltersDotfiles(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".git"))
	mustMkdir(t, filepath.Join(root, ".hidden"))
	mustMkdir(t, filepath.Join(root, "docs"))
	mustTouch(t, filepath.Join(root, ".DS_Store"))
	mustTouch(t, filepath.Join(root, "readme.md"))
	mustTouch(t, filepath.Join(root, "main.go"))

	entries, _, err := listFilePickerEntries(root, root)
	if err != nil {
		t.Fatalf("listFilePickerEntries() error = %v", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	sort.Strings(names)

	for _, forbidden := range []string{".git", ".DS_Store", ".hidden"} {
		if contains(names, forbidden) {
			t.Fatalf("listFilePickerEntries() returned forbidden dotfile %q in %v", forbidden, names)
		}
	}
	for _, expected := range []string{"docs", "main.go", "readme.md"} {
		if !contains(names, expected) {
			t.Fatalf("listFilePickerEntries() missing expected entry %q in %v", expected, names)
		}
	}
}

// TestFilePickerKeymap_AcceptRoute verifies enter matches the accept binding.
func TestFilePickerKeymap_AcceptRoute(t *testing.T) {
	km := newFilePickerKeymap()
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	if !key.Matches(msg, km.accept) {
		t.Fatalf("expected enter to match accept binding")
	}
}

// TestFilePickerCore_TraversalSortsDirsFirst verifies directories sort above
// files, with case-insensitive secondary ordering.
func TestFilePickerCore_TraversalSortsDirsFirst(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "zeta"))
	mustMkdir(t, filepath.Join(root, "alpha"))
	mustTouch(t, filepath.Join(root, "beta.go"))
	mustTouch(t, filepath.Join(root, "Charlie.go"))

	entries, _, err := listFilePickerEntries(root, root)
	if err != nil {
		t.Fatalf("listFilePickerEntries() error = %v", err)
	}
	if len(entries) < 4 {
		t.Fatalf("expected at least 4 entries, got %d", len(entries))
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == ".." {
			continue
		}
		names = append(names, entry.Name)
	}
	if len(names) < 4 {
		t.Fatalf("expected 4 non-parent entries, got %v", names)
	}
	if names[0] != "alpha" || names[1] != "zeta" {
		t.Fatalf("expected dirs first (alpha,zeta), got %v", names[:2])
	}
	if names[2] != "beta.go" || names[3] != "Charlie.go" {
		t.Fatalf("expected case-insensitive file order (beta.go,Charlie.go), got %v", names[2:])
	}
}

// TestFilePickerCore_FuzzyScoreOrdering verifies filtered entries rank by
// match quality (prefix beats substring beats fuzzy subsequence).
func TestFilePickerCore_FuzzyScoreOrdering(t *testing.T) {
	entries := []filePickerEntry{
		{Name: "readme.md", Path: "/r/readme.md", IsDir: false},
		{Name: "random.md", Path: "/r/random.md", IsDir: false},
		{Name: "unrelated.go", Path: "/r/unrelated.go", IsDir: false},
		{Name: "some-readme-extra.md", Path: "/r/some-readme-extra.md", IsDir: false},
	}

	filtered := filterFilePickerEntries(entries, "readme")
	if len(filtered) < 2 {
		t.Fatalf("expected at least 2 matches for 'readme', got %v", filtered)
	}
	if filtered[0].Name != "readme.md" {
		t.Fatalf("expected 'readme.md' first (prefix match), got %q", filtered[0].Name)
	}
	if !contains(entryNames(filtered), "some-readme-extra.md") {
		t.Fatalf("expected 'some-readme-extra.md' in filter results, got %v", entryNames(filtered))
	}
	prefixIdx := indexOf(entryNames(filtered), "readme.md")
	substrIdx := indexOf(entryNames(filtered), "some-readme-extra.md")
	if prefixIdx >= substrIdx {
		t.Fatalf("prefix match must rank above substring match; got order %v", entryNames(filtered))
	}
}

// TestFilePickerCore_FilterEmptyReturnsAll verifies an empty filter query
// returns the entries unchanged (stable ordering).
func TestFilePickerCore_FilterEmptyReturnsAll(t *testing.T) {
	entries := []filePickerEntry{
		{Name: "a", Path: "/r/a", IsDir: true},
		{Name: "b.go", Path: "/r/b.go", IsDir: false},
	}
	out := filterFilePickerEntries(entries, "")
	if len(out) != len(entries) {
		t.Fatalf("empty filter changed entry count got=%d want=%d", len(out), len(entries))
	}
	if out[0].Name != "a" || out[1].Name != "b.go" {
		t.Fatalf("empty filter reordered entries: %v", entryNames(out))
	}
}

// TestFilePickerCore_AppendPathRefsBuildsResourceRef verifies the path-picker
// accept helper appends one ResourceRef per selected entry with Tags[0]=="path"
// and ResourceType reflecting file vs dir.
func TestFilePickerCore_AppendPathRefsBuildsResourceRef(t *testing.T) {
	root := t.TempDir()
	fileRel := "cmd/till/main.go"
	dirRel := "internal/domain"
	mustMkdir(t, filepath.Join(root, filepath.Dir(fileRel)))
	mustTouch(t, filepath.Join(root, fileRel))
	mustMkdir(t, filepath.Join(root, dirRel))

	meta := domain.ActionItemMetadata{
		ResourceRefs: []domain.ResourceRef{
			{
				ResourceType: domain.ResourceTypeURL,
				Location:     "https://example.com",
				PathMode:     domain.PathModeAbsolute,
				Tags:         []string{"doc"},
			},
		},
	}

	selections := []filePickerEntry{
		{Name: "main.go", Path: filepath.Join(root, fileRel), IsDir: false},
		{Name: "domain", Path: filepath.Join(root, dirRel), IsDir: true},
	}

	updated := appendPathResourceRefs(meta, root, selections)

	if len(updated.ResourceRefs) != 3 {
		t.Fatalf("expected 3 refs after append (1 pre-existing + 2 new), got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
	if updated.ResourceRefs[0].ResourceType != domain.ResourceTypeURL {
		t.Fatalf("pre-existing URL ref dropped; head = %#v", updated.ResourceRefs[0])
	}

	fileRef := updated.ResourceRefs[1]
	dirRef := updated.ResourceRefs[2]

	if fileRef.ResourceType != domain.ResourceTypeLocalFile {
		t.Fatalf("file ref ResourceType = %q, want local_file", fileRef.ResourceType)
	}
	if dirRef.ResourceType != domain.ResourceTypeLocalDir {
		t.Fatalf("dir ref ResourceType = %q, want local_dir", dirRef.ResourceType)
	}
	if len(fileRef.Tags) == 0 || fileRef.Tags[0] != "path" {
		t.Fatalf("file ref Tags[0] = %v, want 'path'", fileRef.Tags)
	}
	if len(dirRef.Tags) == 0 || dirRef.Tags[0] != "path" {
		t.Fatalf("dir ref Tags[0] = %v, want 'path'", dirRef.Tags)
	}
	if fileRef.PathMode != domain.PathModeRelative {
		t.Fatalf("file ref PathMode = %q, want relative", fileRef.PathMode)
	}
	if fileRef.Location != filepath.ToSlash(fileRel) {
		t.Fatalf("file ref Location = %q, want %q", fileRef.Location, filepath.ToSlash(fileRel))
	}
	if dirRef.Location != filepath.ToSlash(dirRel) {
		t.Fatalf("dir ref Location = %q, want %q", dirRef.Location, filepath.ToSlash(dirRel))
	}
}

// TestFilePickerCore_AppendPathRefsDedupes verifies appending a path that is
// already present on Metadata.ResourceRefs does not create a duplicate.
func TestFilePickerCore_AppendPathRefsDedupes(t *testing.T) {
	root := t.TempDir()
	rel := "readme.md"
	mustTouch(t, filepath.Join(root, rel))

	meta := domain.ActionItemMetadata{
		ResourceRefs: []domain.ResourceRef{
			{
				ResourceType: domain.ResourceTypeLocalFile,
				Location:     rel,
				PathMode:     domain.PathModeRelative,
				BaseAlias:    "project_root",
				Tags:         []string{"path"},
			},
		},
	}
	selections := []filePickerEntry{
		{Name: "readme.md", Path: filepath.Join(root, rel), IsDir: false},
	}
	updated := appendPathResourceRefs(meta, root, selections)
	if len(updated.ResourceRefs) != 1 {
		t.Fatalf("duplicate path appended — want 1 ref, got %d: %#v", len(updated.ResourceRefs), updated.ResourceRefs)
	}
}

// TestFilePickerCore_TraversalMissingDir returns an error for a non-existent
// directory without panicking.
func TestFilePickerCore_TraversalMissingDir(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "does-not-exist")
	if _, _, err := listFilePickerEntries(root, missing); err == nil {
		t.Fatal("expected error when listing missing directory, got nil")
	}
}

// TestFilePickerCore_StartResetsState verifies resetting the core seeds clean
// state for a fresh open.
func TestFilePickerCore_StartResetsState(t *testing.T) {
	root := t.TempDir()
	mustTouch(t, filepath.Join(root, "a.go"))

	core := newFilePickerCore()
	core.start(filePickerModePath, "task-1", root)

	if core.mode != filePickerModePath {
		t.Fatalf("start() mode = %v, want filePickerModePath", core.mode)
	}
	if core.taskID != "task-1" {
		t.Fatalf("start() taskID = %q, want 'task-1'", core.taskID)
	}
	if core.root == "" {
		t.Fatalf("start() left root empty")
	}
	if core.index != 0 {
		t.Fatalf("start() index = %d, want 0", core.index)
	}
	if len(core.selected) != 0 {
		t.Fatalf("start() did not clear selected set: %v", core.selected)
	}
	if core.filter.Value() != "" {
		t.Fatalf("start() did not clear filter: %q", core.filter.Value())
	}
}

// TestFilePickerCore_ToggleTracksSelection verifies toggling marks and unmarks
// entries in the selection set keyed by absolute path.
func TestFilePickerCore_ToggleTracksSelection(t *testing.T) {
	core := newFilePickerCore()
	core.start(filePickerModePath, "", t.TempDir())
	entry := filePickerEntry{Name: "a.go", Path: "/r/a.go", IsDir: false}

	core.toggleSelect(entry)
	if !core.isSelected(entry) {
		t.Fatalf("expected entry selected after toggleSelect, got %v", core.selected)
	}
	core.toggleSelect(entry)
	if core.isSelected(entry) {
		t.Fatalf("expected entry cleared after second toggleSelect, got %v", core.selected)
	}
}

// TestFilePickerCore_SelectedEntriesSortedByPath verifies the accept helper
// returns selections in stable path-sorted order, so append order is deterministic.
func TestFilePickerCore_SelectedEntriesSortedByPath(t *testing.T) {
	core := newFilePickerCore()
	core.start(filePickerModePath, "", t.TempDir())
	entries := []filePickerEntry{
		{Name: "c.go", Path: "/r/c.go"},
		{Name: "a.go", Path: "/r/a.go"},
		{Name: "b.go", Path: "/r/b.go"},
	}
	for _, entry := range entries {
		core.toggleSelect(entry)
	}
	got := core.selectedEntries()
	names := entryNames(got)
	if !equalStrings(names, []string{"a.go", "b.go", "c.go"}) {
		t.Fatalf("selectedEntries() order = %v, want [a.go b.go c.go]", names)
	}
}

// TestFilePickerKeymap_CancelRoute verifies esc matches the cancel binding.
func TestFilePickerKeymap_CancelRoute(t *testing.T) {
	km := newFilePickerKeymap()
	esc := tea.KeyPressMsg{Code: tea.KeyEsc}
	if !key.Matches(esc, km.cancel) {
		t.Fatalf("expected esc to match cancel binding")
	}
}

// TestFilePickerKeymap_ToggleRoute verifies tab and space match the toggle
// binding (multi-select route).
func TestFilePickerKeymap_ToggleRoute(t *testing.T) {
	km := newFilePickerKeymap()
	tab := tea.KeyPressMsg{Code: tea.KeyTab}
	if !key.Matches(tab, km.toggle) {
		t.Fatalf("expected tab to match toggle binding")
	}
	space := tea.KeyPressMsg{Code: ' ', Text: " "}
	if !key.Matches(space, km.toggle) {
		t.Fatalf("expected space to match toggle binding")
	}
}

// TestFilePickerKeymap_ShortHelpBindings verifies the short-help row exposes
// the canonical binding set for bottom-of-screen help.
func TestFilePickerKeymap_ShortHelpBindings(t *testing.T) {
	km := newFilePickerKeymap()
	short := km.shortHelp()
	if len(short) < 5 {
		t.Fatalf("shortHelp() = %d bindings, want at least 5", len(short))
	}
}

// mustMkdir creates a directory or fails the test.
func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", p, err)
	}
}

// mustTouch creates an empty file or fails the test.
func mustTouch(t *testing.T, p string) {
	t.Helper()
	if err := os.WriteFile(p, []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", p, err)
	}
}

// contains returns true if s appears in xs.
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// indexOf returns the index of s in xs, or -1 when missing.
func indexOf(xs []string, s string) int {
	for idx, x := range xs {
		if x == s {
			return idx
		}
	}
	return -1
}

// equalStrings reports whether two string slices are element-wise equal.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// entryNames pulls Name fields off a slice of filePickerEntry.
func entryNames(entries []filePickerEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name)
	}
	return out
}
