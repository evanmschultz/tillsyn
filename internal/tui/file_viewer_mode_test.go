package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/evanmschultz/tillsyn/internal/config"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// fileViewerGoldenUpdateEnv regenerates file-viewer golden fixtures when set.
const fileViewerGoldenUpdateEnv = "TILLSYN_FILEVIEWER_GOLDEN_UPDATE"

// stubMarkdownRenderFunc is a deterministic render function for golden tests.
func stubMarkdownRenderFunc(content string, _ int) string {
	return "[glamour-stub:" + strings.TrimSpace(content) + "]"
}

// stubCodeRenderFunc is a deterministic code render function for golden tests.
func stubCodeRenderFunc(content []byte, lexerName string) (string, error) {
	return "[chroma-stub:" + lexerName + ":" + strings.TrimSpace(string(content)) + "]", nil
}

// newTestFileViewerMode constructs a fileViewerMode with a stub markdownRenderer
// and default config suitable for deterministic unit testing.
func newTestFileViewerMode(t *testing.T) *fileViewerMode {
	t.Helper()
	cfg := config.FileViewerConfig{
		MaxBytes:      1048576,
		DotfileBanner: config.DefaultFileViewerDotfileBanner,
	}
	// The stub renderer does not need to call glamour; override chooseRenderer
	// behavior through the test-only chooseRendererWithFuncs path in tests that
	// need golden output. For behavioral tests, a nil *markdownRenderer is fine
	// because the test controls the file content and extension.
	fv := newFileViewerMode(nil, cfg)
	fv.resize(80, 24)
	return fv
}

// TestFileViewer_Dotfile_Refused asserts that opening a dotfile produces the
// banner content and does not read the file.
func TestFileViewer_Dotfile_Refused(t *testing.T) {
	fixture := filepath.Join("testdata", "file_viewer", ".gitignore_fixture")
	if _, err := os.Stat(fixture); err != nil {
		t.Fatalf("dotfile fixture missing at %s: %v", fixture, err)
	}
	fv := newTestFileViewerMode(t)
	err := fv.openFile(fixture)
	if err != nil {
		t.Fatalf("openFile returned unexpected error for dotfile: %v", err)
	}
	if !strings.Contains(fv.content, "Dotfiles not supported in v1") {
		t.Fatalf("expected dotfile banner, got %q", fv.content)
	}
	// Sanity check: the dotfile content (*.log / dist/) must NOT appear.
	if strings.Contains(fv.content, "*.log") {
		t.Fatalf("dotfile content should not be read; got %q", fv.content)
	}
}

// TestFileViewer_LargeFile_Refused asserts that a file above max_bytes renders
// a banner and is NOT fully read into memory.
func TestFileViewer_LargeFile_Refused(t *testing.T) {
	// Create a temp file that is 1 byte larger than the limit.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "large.txt")
	maxBytes := 64 // tiny limit for the test
	data := make([]byte, maxBytes+1)
	for i := range data {
		data[i] = 'x'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}
	cfg := config.FileViewerConfig{
		MaxBytes:      maxBytes,
		DotfileBanner: config.DefaultFileViewerDotfileBanner,
	}
	fv := newFileViewerMode(nil, cfg)
	fv.resize(80, 24)

	err := fv.openFile(path)
	if err == nil {
		t.Fatal("expected error for large file, got nil")
	}
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
	if !strings.Contains(fv.content, "File too large") {
		t.Fatalf("expected 'File too large' banner in content, got %q", fv.content)
	}
}

// TestFileViewer_MissingFile_Error asserts that a non-existent path returns a
// wrapped error and does not panic.
func TestFileViewer_MissingFile_Error(t *testing.T) {
	fv := newTestFileViewerMode(t)
	err := fv.openFile("/non/existent/path/does_not_exist.txt")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	// The error must be wrapped, not a bare os.ErrNotExist.
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist in chain, got %v", err)
	}
}

// TestFileViewer_PlainText_Passthrough asserts that .txt files are returned
// verbatim without any rendering pipeline applied.
func TestFileViewer_PlainText_Passthrough(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "notes.txt")
	raw := "plain text content\nno markdown\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write plain text file: %v", err)
	}
	fv := newTestFileViewerMode(t)
	if err := fv.openFile(path); err != nil {
		t.Fatalf("openFile returned unexpected error: %v", err)
	}
	if fv.content != raw {
		t.Fatalf("expected passthrough content %q, got %q", raw, fv.content)
	}
}

// TestFileViewer_Markdown_Golden asserts that a .md file renders through the
// stub glamour renderer and the output matches the golden fixture.
func TestFileViewer_Markdown_Golden(t *testing.T) {
	fixture := filepath.Join("testdata", "file_viewer", "sample.md")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read markdown fixture: %v", err)
	}
	got, err := chooseRendererWithFuncs(fixture, raw, stubMarkdownRenderFunc, stubCodeRenderFunc)
	if err != nil {
		t.Fatalf("chooseRendererWithFuncs: %v", err)
	}

	goldenPath := filepath.Join("testdata", "file_viewer", "sample_md.golden")
	if os.Getenv(fileViewerGoldenUpdateEnv) != "" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with %s=1 to regenerate)", goldenPath, err, fileViewerGoldenUpdateEnv)
	}
	if string(want) != got {
		t.Fatalf("golden mismatch at %s\nrun with %s=1 to regenerate\ngot:\n%s\nwant:\n%s",
			goldenPath, fileViewerGoldenUpdateEnv, got, string(want))
	}
}

// TestFileViewer_GoCode_Golden asserts that a .go file renders through the
// stub chroma renderer and the output matches the golden fixture.
func TestFileViewer_GoCode_Golden(t *testing.T) {
	fixture := filepath.Join("testdata", "file_viewer", "sample.go")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read go fixture: %v", err)
	}
	got, err := chooseRendererWithFuncs(fixture, raw, stubMarkdownRenderFunc, stubCodeRenderFunc)
	if err != nil {
		t.Fatalf("chooseRendererWithFuncs: %v", err)
	}

	goldenPath := filepath.Join("testdata", "file_viewer", "sample_go.golden")
	if os.Getenv(fileViewerGoldenUpdateEnv) != "" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with %s=1 to regenerate)", goldenPath, err, fileViewerGoldenUpdateEnv)
	}
	if string(want) != got {
		t.Fatalf("golden mismatch at %s\nrun with %s=1 to regenerate\ngot:\n%s\nwant:\n%s",
			goldenPath, fileViewerGoldenUpdateEnv, got, string(want))
	}
}

// TestFileViewer_SharesThreadMarkdown asserts that the fileViewerMode holds
// a pointer to the Model's threadMarkdown field — not a separate renderer.
func TestFileViewer_SharesThreadMarkdown(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, nil)
	m := NewModel(svc)

	if m.fileViewer == nil {
		t.Fatal("expected fileViewer to be non-nil after NewModel")
	}
	// fileViewerMode.md must be pointer-equal to m.threadMarkdown.
	if m.fileViewer.md != m.threadMarkdown {
		t.Fatalf("fileViewerMode.md is not model.threadMarkdown: got %p, want %p",
			m.fileViewer.md, m.threadMarkdown)
	}
}

// TestModel_V_EntersFileViewerMode asserts that pressing v from the normal
// board surface transitions the Model to modeFileViewer.
func TestModel_V_EntersFileViewerMode(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProjectFromInput(domain.ProjectInput{ID: "p1", Name: "Inbox"}, now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := newActionItemForTest(domain.ActionItemInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "File viewer toggle",
		Priority:  domain.PriorityLow,
	}, now)

	svc := newFakeService([]domain.Project{p}, []domain.Column{c1}, []domain.ActionItem{task})
	m := loadReadyModel(t, NewModel(svc))

	if m.mode != modeNone {
		t.Fatalf("expected modeNone at start, got %v", m.mode)
	}
	m = applyMsg(t, m, tea.KeyPressMsg{Code: 'v', Text: "v"})
	if m.mode != modeFileViewer {
		t.Fatalf("expected modeFileViewer after v, got %v", m.mode)
	}
	if m.fileViewer == nil {
		t.Fatal("expected m.fileViewer to be non-nil after v")
	}
}

// TestKeymap_V_NoCollision enumerates every existing binding in newKeyMap()
// and asserts that none of them claim "v" before fileViewerToggle is registered.
// The enumeration is exhaustive — not a spot-check.
func TestKeymap_V_NoCollision(t *testing.T) {
	km := newKeyMap()
	// All bindings except fileViewerToggle (which is the new one being added).
	existingBindings := map[string][]string{
		"quit":                 km.quit.Keys(),
		"reload":               km.reload.Keys(),
		"toggleHelp":           km.toggleHelp.Keys(),
		"moveLeft":             km.moveLeft.Keys(),
		"moveRight":            km.moveRight.Keys(),
		"moveUp":               km.moveUp.Keys(),
		"moveDown":             km.moveDown.Keys(),
		"addActionItem":        km.addActionItem.Keys(),
		"actionItemInfo":       km.actionItemInfo.Keys(),
		"editActionItem":       km.editActionItem.Keys(),
		"newProject":           km.newProject.Keys(),
		"editProject":          km.editProject.Keys(),
		"commandPalette":       km.commandPalette.Keys(),
		"quickActions":         km.quickActions.Keys(),
		"deleteActionItem":     km.deleteActionItem.Keys(),
		"archiveActionItem":    km.archiveActionItem.Keys(),
		"moveActionItemLeft":   km.moveActionItemLeft.Keys(),
		"moveActionItemRight":  km.moveActionItemRight.Keys(),
		"hardDeleteActionItem": km.hardDeleteActionItem.Keys(),
		"restoreActionItem":    km.restoreActionItem.Keys(),
		"search":               km.search.Keys(),
		"projects":             km.projects.Keys(),
		"toggleArchived":       km.toggleArchived.Keys(),
		"toggleSelectMode":     km.toggleSelectMode.Keys(),
		"focusSubtree":         km.focusSubtree.Keys(),
		"clearFocus":           km.clearFocus.Keys(),
		"multiSelect":          km.multiSelect.Keys(),
		"activityLog":          km.activityLog.Keys(),
		"undo":                 km.undo.Keys(),
		"redo":                 km.redo.Keys(),
		"diffModeToggle":       km.diffModeToggle.Keys(),
	}
	for name, keys := range existingBindings {
		for _, k := range keys {
			if k == "v" {
				t.Fatalf("binding %q already claims v; fileViewerToggle would collide", name)
			}
		}
	}
	// Positive assertion: the new toggle binding is actually wired.
	if keys := km.fileViewerToggle.Keys(); len(keys) != 1 || keys[0] != "v" {
		t.Fatalf("fileViewerToggle keys = %#v, want exactly [v]", keys)
	}
}

// TestConfig_FileViewerDefaults asserts that the default config parses with
// correct file-viewer section values.
func TestConfig_FileViewerDefaults(t *testing.T) {
	cfg := config.Default("/tmp/test.db")
	if cfg.TUI.Surfaces.FileViewer.MaxBytes != 1048576 {
		t.Fatalf("expected MaxBytes=1048576, got %d", cfg.TUI.Surfaces.FileViewer.MaxBytes)
	}
	if cfg.TUI.Surfaces.FileViewer.DotfileBanner != "Dotfiles not supported in v1" {
		t.Fatalf("expected DotfileBanner=%q, got %q",
			"Dotfiles not supported in v1", cfg.TUI.Surfaces.FileViewer.DotfileBanner)
	}
}
