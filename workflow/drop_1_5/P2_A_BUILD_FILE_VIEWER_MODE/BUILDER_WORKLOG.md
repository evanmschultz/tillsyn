# P2-A Builder Worklog

Append `## Round K` per build attempt.

## Round 1

**Outcome:** `mage ci` green — 1334 tests, 0 failures, all packages ≥ 70% coverage.

### Files touched

- `internal/config/config.go` — added `TUIConfig`, `TUISurfacesConfig`, `FileViewerConfig` structs, `DefaultFileViewerMaxBytes`, `DefaultFileViewerDotfileBanner` constants; wired into `Default()` and `normalize()`.
- `internal/tui/keymap.go` — added `fileViewerToggle key.Binding` to `keyMap` struct; initialized to `v` in `newKeyMap()`; gofumpt applied.
- `internal/tui/file_viewer_renderer.go` — new file; `fileRendererKind` iota, `classifyExtension`, `chooseRenderer`, `chooseRendererWithFuncs`, `renderCodeContent`.
- `internal/tui/file_viewer_mode.go` — new file; `ErrFileTooLarge`, `fileViewerBannerDotfile`, `fileViewerMode` struct, `defaultFileViewerConfig`, `newFileViewerMode`, `openFile`, `renderInto`, `resize`, `reset`, `viewContent`, `enterFileViewerMode`, `exitFileViewerMode`, `handleFileViewerModeKey`, `renderFileViewerModeView`.
- `internal/tui/model.go` — changed `threadMarkdown` field from `markdownRenderer` (value) to `*markdownRenderer` (pointer, heap-allocated in `NewModel`) to make its address stable across Model copies; added `modeFileViewer` iota; added `fileViewer *fileViewerMode` and `fileViewerBackMode inputMode` fields; wired `NewModel`, `Update` dispatch, `View`, `activeScreenHints`, `activeBottomHelpKeyMap`, `modeString`, `modePrompt`; moved `fileViewerToggle` dispatch into `handleBoardPanelNormalKey` after `activityLog` case (prevents `v`-collision when `ActivityLog` is configured to `v`).
- `internal/tui/file_viewer_mode_test.go` — new file; 10 TDD tests as specified.
- `internal/tui/testdata/file_viewer/sample.md` — markdown fixture.
- `internal/tui/testdata/file_viewer/sample.go` — Go fixture.
- `internal/tui/testdata/file_viewer/.gitignore_fixture` — dotfile fixture.
- `internal/tui/testdata/file_viewer/sample_md.golden` — generated golden.
- `internal/tui/testdata/file_viewer/sample_go.golden` — generated golden.

### Mage output

```
mage ci — SUCCESS
1334 tests / 0 failures / 0 build errors / 20 packages
internal/tui: 70.4% coverage
internal/config: 76.8% coverage
mage testGolden — 7/7 golden tests pass
```

### Design decision: `threadMarkdown *markdownRenderer`

The `TestFileViewer_SharesThreadMarkdown` pointer-equality assertion exposed a Go value-type aliasing issue: storing `&m.threadMarkdown` (address of a value field) in `newFileViewerMode` during `NewModel` produced a dangling-ish pointer after `return m` copied the struct. Fixed by changing the field to `*markdownRenderer` (heap allocation in `NewModel`). All 10+ call sites (`m.threadMarkdown.render(...)`) work unchanged — Go auto-derefs pointer method calls.

### Design decision: `fileViewerToggle` dispatch position

Placed in `handleBoardPanelNormalKey` after the `activityLog` case rather than in the top-level `handleNormalModeKey` switch. This ensures a `KeyConfig{ActivityLog: "v"}` override takes precedence over the default `v` binding — `TestWithKeyConfigOverrides` confirms correctness.

### Hylla Feedback

- **Query**: `hylla_search` for `markdownRenderer` struct definition — needed to understand field type and whether it had a pointer receiver.
- **Missed because**: Hylla returned the call sites for `render()` but not the struct definition in `markdown_renderer.go`. The struct and its `render` method were found via `Grep`.
- **Worked via**: `Grep pattern=func.*markdownRenderer.*render path=internal/tui` — found `markdown_renderer.go:17`.
- **Suggestion**: Hylla should index struct-level definitions separately from call sites so a symbol search for `markdownRenderer` returns the type declaration as the top hit.

- **Query**: `hylla_refs_find` for `threadMarkdown` usage across `model.go` — needed all call sites before changing the field type.
- **Missed because**: Hylla returned no results (field not indexed as a standalone ref node — it's a struct field access, not a function call).
- **Worked via**: `Grep pattern=threadMarkdown path=internal/tui/model.go` — found all 9 call sites cleanly.
- **Suggestion**: Struct field access sites (selector expressions) should be indexed as refs in Hylla, not just function calls.
