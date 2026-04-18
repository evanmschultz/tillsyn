package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"github.com/evanmschultz/tillsyn/internal/domain"
)

// filePickerMode identifies which file-picker variant is currently active.
//
// Variants share one core (this file) but differ in accept semantics:
//   - filePickerModeNone: picker is idle / closed.
//   - filePickerModePath: selected entries materialize as Tags[0]=="path"
//     ResourceRefs on the active plan-item's Metadata.ResourceRefs.
//
// Future variants (file-picker, url-ref-picker, etc.) extend this enum; the
// core remains unchanged.
type filePickerMode int

// filePickerMode values.
const (
	filePickerModeNone filePickerMode = iota
	filePickerModePath
)

// filePickerEntry describes one filesystem row rendered by the picker.
//
// The shape mirrors the pre-existing resourcePickerEntry so scoring helpers
// (bestFuzzyScore) remain shared. Parent-link rows use Name=="..".
type filePickerEntry struct {
	Name  string
	Path  string
	IsDir bool
}

// filePickerCore holds the state shared across all file-picker variants.
//
// State is grouped into four partitions:
//   - routing: mode + taskID + back-mode so callers know where to return.
//   - navigation: root + dir + index for the highlighted row.
//   - entries: the raw directory listing + the filter input model.
//   - selection: the multi-select set keyed by absolute path.
//
// The core does not own rendering or dispatch — callers (file_picker_render.go
// and the host Model's Update loop) consume state via the accessor methods.
type filePickerCore struct {
	mode     filePickerMode
	taskID   string
	root     string
	dir      string
	entries  []filePickerEntry
	index    int
	filter   textinput.Model
	selected map[string]struct{}
}

// newFilePickerCore constructs the default core state with an initialized
// filter input. The core is idle (filePickerModeNone) until start() is called.
func newFilePickerCore() filePickerCore {
	input := textinput.New()
	input.Prompt = "filter: "
	input.Placeholder = "type to fuzzy-filter files/dirs"
	input.CharLimit = 120
	return filePickerCore{
		mode:     filePickerModeNone,
		filter:   input,
		selected: make(map[string]struct{}),
	}
}

// start resets the core to a clean open for the given mode + taskID, rooted at
// root. Callers drive entry loading via listFilePickerEntries afterwards.
func (c *filePickerCore) start(mode filePickerMode, taskID, root string) {
	c.mode = mode
	c.taskID = strings.TrimSpace(taskID)
	c.root = strings.TrimSpace(root)
	c.dir = c.root
	c.entries = nil
	c.index = 0
	c.selected = make(map[string]struct{})
	c.filter.SetValue("")
	c.filter.CursorEnd()
}

// stop clears state after accept/cancel so the picker is idle.
func (c *filePickerCore) stop() {
	c.mode = filePickerModeNone
	c.taskID = ""
	c.entries = nil
	c.index = 0
	c.selected = make(map[string]struct{})
	c.filter.SetValue("")
	c.filter.Blur()
}

// setEntries swaps the raw entries and absolute-path directory after a
// successful load. Index is clamped to the new length.
func (c *filePickerCore) setEntries(entries []filePickerEntry, dir string) {
	c.entries = entries
	c.dir = dir
	if len(entries) == 0 {
		c.index = 0
		return
	}
	c.index = clamp(c.index, 0, len(entries)-1)
}

// visibleEntries returns the fuzzy-filtered slice the renderer draws.
func (c filePickerCore) visibleEntries() []filePickerEntry {
	return filterFilePickerEntries(c.entries, c.filter.Value())
}

// selectedEntry returns the currently highlighted entry and whether the slice
// is non-empty.
func (c filePickerCore) selectedEntry() (filePickerEntry, bool) {
	items := c.visibleEntries()
	if len(items) == 0 {
		return filePickerEntry{}, false
	}
	idx := clamp(c.index, 0, len(items)-1)
	return items[idx], true
}

// toggleSelect flips the multi-select state for entry (absolute-path key).
// Parent-link rows ("..") are ignored — they are navigation-only.
func (c *filePickerCore) toggleSelect(entry filePickerEntry) {
	if entry.Name == ".." || strings.TrimSpace(entry.Path) == "" {
		return
	}
	key := filepath.Clean(entry.Path)
	if _, ok := c.selected[key]; ok {
		delete(c.selected, key)
		return
	}
	c.selected[key] = struct{}{}
}

// isSelected reports whether entry has been toggled into the selection set.
func (c filePickerCore) isSelected(entry filePickerEntry) bool {
	if strings.TrimSpace(entry.Path) == "" {
		return false
	}
	_, ok := c.selected[filepath.Clean(entry.Path)]
	return ok
}

// selectedEntries returns the multi-select set as a path-sorted slice for
// deterministic append order on accept. IsDir is rebuilt from os.Stat so
// consumers never need to retain the original entries slice across renders.
func (c filePickerCore) selectedEntries() []filePickerEntry {
	if len(c.selected) == 0 {
		return nil
	}
	paths := make([]string, 0, len(c.selected))
	for p := range c.selected {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := make([]filePickerEntry, 0, len(paths))
	for _, p := range paths {
		entry := filePickerEntry{Name: filepath.Base(p), Path: p}
		if info, err := os.Stat(p); err == nil {
			entry.IsDir = info.IsDir()
		}
		out = append(out, entry)
	}
	return out
}

// listFilePickerEntries reads one directory, filters dotfile noise, and sorts
// directories above files. Parent-link row is prepended when dir is not the
// root.
//
// Filter rules (dotfile-only MVP per synthesis §2.1 — gitignore-respect is
// deferred to REFINEMENTS):
//   - names beginning with "." are dropped (covers .git, .DS_Store, .cache, …).
//   - parent-link row for the immediate parent is retained (Name == "..").
//
// The returned dir is the absolute resolved directory the entries describe.
func listFilePickerEntries(root, dir string) ([]filePickerEntry, string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, "", fmt.Errorf("file picker: root is required")
	}
	dirAbs := strings.TrimSpace(dir)
	if dirAbs == "" {
		dirAbs = root
	}
	dirAbs, err := filepath.Abs(dirAbs)
	if err != nil {
		return nil, "", fmt.Errorf("file picker: resolve dir: %w", err)
	}
	items, err := os.ReadDir(dirAbs)
	if err != nil {
		return nil, "", fmt.Errorf("file picker: read dir: %w", err)
	}

	entries := make([]filePickerEntry, 0, len(items)+1)
	parent := filepath.Dir(dirAbs)
	if parent != "." && parent != "" && parent != dirAbs {
		entries = append(entries, filePickerEntry{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}
	for _, item := range items {
		name := item.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		entries = append(entries, filePickerEntry{
			Name:  name,
			Path:  filepath.Join(dirAbs, name),
			IsDir: item.IsDir(),
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		// Parent link is always first.
		if entries[i].Name == ".." {
			return true
		}
		if entries[j].Name == ".." {
			return false
		}
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, dirAbs, nil
}

// filterFilePickerEntries ranks entries against query using the shared
// bestFuzzyScore semantics (prefix > substring > fuzzy subsequence). Empty
// query returns entries unchanged (stable order).
//
// Reuses bestFuzzyScore from model.go so scoring stays aligned with the
// resource-picker and command-palette surfaces. Directory boost keeps
// directories ahead of files when scores tie, matching resourcePicker
// conventions (+8 for dirs, -100 for the parent link so it never bubbles to
// the top under a live query).
func filterFilePickerEntries(entries []filePickerEntry, query string) []filePickerEntry {
	if len(entries) == 0 {
		return nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		out := make([]filePickerEntry, len(entries))
		copy(out, entries)
		return out
	}

	type scoredEntry struct {
		entry filePickerEntry
		score int
	}
	scored := make([]scoredEntry, 0, len(entries))
	for _, entry := range entries {
		score, ok := bestFuzzyScore(query, entry.Name, filepath.ToSlash(entry.Path))
		if !ok {
			continue
		}
		if entry.IsDir {
			score += 8
		}
		if entry.Name == ".." {
			score -= 100
		}
		scored = append(scored, scoredEntry{entry: entry, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].entry.IsDir != scored[j].entry.IsDir {
			return scored[i].entry.IsDir
		}
		return strings.ToLower(scored[i].entry.Name) < strings.ToLower(scored[j].entry.Name)
	})
	out := make([]filePickerEntry, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.entry)
	}
	return out
}

// appendPathResourceRefs is the path-picker accept handler. For each selected
// entry it builds one ResourceRef with Tags[0]=="path" and the standard
// buildResourceRef relative/absolute encoding, then appends to meta.ResourceRefs
// via appendResourceRefIfMissing so duplicate paths no-op.
//
// Pure function — operates on the ActionItemMetadata value the caller hands in.
// Callers drive the actual domain.UpdateActionItem call separately.
func appendPathResourceRefs(meta domain.ActionItemMetadata, root string, entries []filePickerEntry) domain.ActionItemMetadata {
	out := meta
	refs := append([]domain.ResourceRef(nil), meta.ResourceRefs...)
	for _, entry := range entries {
		if strings.TrimSpace(entry.Path) == "" || entry.Name == ".." {
			continue
		}
		ref := buildResourceRef(root, entry.Path, entry.IsDir)
		ref.Tags = []string{"path"}
		next, _ := appendResourceRefIfMissing(refs, ref)
		refs = next
	}
	out.ResourceRefs = refs
	return out
}
