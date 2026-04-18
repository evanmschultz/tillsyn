package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestFilePickerRenderBody_ListsEntries verifies renderFilePickerBody includes
// each entry name plus the filter prompt and selection footer.
func TestFilePickerRenderBody_ListsEntries(t *testing.T) {
	core := newFilePickerCore()
	core.start(filePickerModePath, "task-1", t.TempDir())
	core.setEntries([]filePickerEntry{
		{Name: "..", Path: "/tmp", IsDir: true},
		{Name: "docs", Path: "/tmp/root/docs", IsDir: true},
		{Name: "main.go", Path: "/tmp/root/main.go", IsDir: false},
	}, "/tmp/root")

	out := renderFilePickerBody(core, filePickerRenderOptions{
		accent:   lipgloss.Color("62"),
		muted:    lipgloss.Color("8"),
		dim:      lipgloss.Color("240"),
		boxWidth: 80,
		maxRows:  20,
	})

	plain := stripANSI(out)
	if !strings.Contains(plain, "filter:") {
		t.Fatalf("expected filter prompt in render, got:\n%s", plain)
	}
	for _, needed := range []string{"docs/", "main.go", "selected: 0"} {
		if !strings.Contains(plain, needed) {
			t.Fatalf("expected %q in render, got:\n%s", needed, plain)
		}
	}
	if !strings.Contains(plain, "tab/space select") {
		t.Fatalf("expected footer hint with tab/space select, got:\n%s", plain)
	}
}

// TestFilePickerRenderBody_ShowsSelectionMarker verifies selected entries
// render with a visible marker and the selection count updates.
func TestFilePickerRenderBody_ShowsSelectionMarker(t *testing.T) {
	core := newFilePickerCore()
	core.start(filePickerModePath, "task-1", t.TempDir())
	entry := filePickerEntry{Name: "main.go", Path: "/tmp/root/main.go", IsDir: false}
	core.setEntries([]filePickerEntry{entry}, "/tmp/root")
	core.toggleSelect(entry)

	out := renderFilePickerBody(core, filePickerRenderOptions{
		accent:   lipgloss.Color("62"),
		muted:    lipgloss.Color("8"),
		dim:      lipgloss.Color("240"),
		boxWidth: 80,
		maxRows:  20,
	})

	plain := stripANSI(out)
	if !strings.Contains(plain, "selected: 1") {
		t.Fatalf("expected selected: 1 after toggle, got:\n%s", plain)
	}
	// The selection-marker asterisk renders next to the entry name.
	if !strings.Contains(plain, "* main.go") {
		t.Fatalf("expected '* main.go' with selection marker, got:\n%s", plain)
	}
}

// TestFilePickerRenderBody_EmptyShowsHint verifies the renderer falls through
// to an explicit "no entries" hint when the filtered list is empty.
func TestFilePickerRenderBody_EmptyShowsHint(t *testing.T) {
	core := newFilePickerCore()
	core.start(filePickerModePath, "task-1", t.TempDir())

	out := renderFilePickerBody(core, filePickerRenderOptions{
		accent:   lipgloss.Color("62"),
		muted:    lipgloss.Color("8"),
		dim:      lipgloss.Color("240"),
		boxWidth: 80,
		maxRows:  20,
	})
	plain := stripANSI(out)
	if !strings.Contains(plain, "no entries matching filter") {
		t.Fatalf("expected empty-state hint, got:\n%s", plain)
	}
}

// TestFilePickerRenderBody_CollapsesLongLists verifies the "+N more" collapse
// line appears when there are more entries than maxRows.
func TestFilePickerRenderBody_CollapsesLongLists(t *testing.T) {
	core := newFilePickerCore()
	core.start(filePickerModePath, "task-1", t.TempDir())
	many := make([]filePickerEntry, 0, 30)
	for i := 0; i < 30; i++ {
		many = append(many, filePickerEntry{
			Name: string(rune('a'+i%26)) + ".go",
			Path: "/tmp/root/" + string(rune('a'+i%26)) + ".go",
		})
	}
	core.setEntries(many, "/tmp/root")

	out := renderFilePickerBody(core, filePickerRenderOptions{
		accent:   lipgloss.Color("62"),
		muted:    lipgloss.Color("8"),
		dim:      lipgloss.Color("240"),
		boxWidth: 80,
		maxRows:  5,
	})
	plain := stripANSI(out)
	if !strings.Contains(plain, "more entries") {
		t.Fatalf("expected 'more entries' hint with maxRows=5, got:\n%s", plain)
	}
}

// TestFilePickerTitles verifies title/subtitle helpers produce stable strings
// across the enum range.
func TestFilePickerTitles(t *testing.T) {
	if got := filePickerTitleFor(filePickerModePath); got != "Add Paths" {
		t.Fatalf("filePickerTitleFor(path) = %q, want 'Add Paths'", got)
	}
	if got := filePickerTitleFor(filePickerModeNone); got != "File Picker" {
		t.Fatalf("filePickerTitleFor(none) = %q, want 'File Picker'", got)
	}
	if !strings.Contains(filePickerSubtitleFor(filePickerModePath), "path") {
		t.Fatalf("filePickerSubtitleFor(path) missing 'path' token: %q", filePickerSubtitleFor(filePickerModePath))
	}
}
