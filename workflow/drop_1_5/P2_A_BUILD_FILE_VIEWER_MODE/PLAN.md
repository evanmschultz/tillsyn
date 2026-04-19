---
task: P2-A — BUILD FILE VIEWER MODE (V KEYBINDING)
tillsyn_id: 8020fd8a-4c29-402d-9eab-da72cf7fcebf
role: builder (go-builder-agent, sonnet)
state: done
blocked_by: none (P4-T4 closed at commit e8914fc)
worktree: /Users/evanschultz/Documents/Code/hylla/tillsyn/drop/1.5/
---

# P2-A — BUILD FILE VIEWER MODE (V KEYBINDING)

## Purpose

Add a file-viewer mode opened by `v` (per DD-1: `v` goes to P2 file viewer, `ctrl+d` is P4 diff). Given the active plan-item's first `Paths` entry (or, if empty, a picker prompt — deferred to a future drop), read the file from disk, render via the shared `Model.threadMarkdown` glamour renderer when the file is markdown, or via chroma syntax highlighting (reusing patterns from P4-T2) when it's a code file. Dotfile-only filter: files whose basename starts with `.` are refused ("Dotfiles not supported in v1" banner). No `.gitignore` parsing for MVP. Unified `[tui.surfaces.file_viewer]` TOML config schema added to `internal/config/`.

## Paths (new or modified)

- `internal/tui/file_viewer_mode.go` (new — fileViewerMode struct, Update, View).
- `internal/tui/file_viewer_renderer.go` (new — chooseRenderer function: markdown → glamour, code → chroma, fallback → plain text).
- `internal/tui/file_viewer_mode_test.go` (new — unit tests).
- `internal/tui/testdata/file_viewer/` (new — sample .md, .go, dotfile fixtures + golden renders).
- `internal/tui/model.go` (modified — add `fileViewerMode *fileViewerMode` field, `modeFileViewer` enum entry, dispatch).
- `internal/tui/keymap.go` (modified — add `FileViewerToggle` key.Binding on `v`).
- `internal/tui/keymap_test.go` (modified — assert `v` non-colliding under every surface mode).
- `internal/config/config.go` (modified — add `TUISurfaces.FileViewer` struct + TOML defaults).
- `internal/config/config_test.go` (modified — test default config parses + file-viewer section).

## Packages

- `internal/tui` (modified)
- `internal/config` (modified)

## Acceptance Criteria (QA yes/no calls)

- `v` keybinding in `keymap.go` under field `FileViewerToggle`. No collision across every surface mode or with `ctrl+d` (P4-T3).
- `fileViewerMode` struct: viewport, active file path, rendered content, error (if any).
- Dotfile-only filter: if basename starts with `.`, render "Dotfiles not supported in v1" banner instead of content. Do NOT enter the viewport; keep mode enum on `modeFileViewer` but content is the banner.
- Renderer selection (`chooseRenderer`):
  - `.md`, `.markdown` → glamour via `Model.threadMarkdown` (SHARED renderer — see synthesis §5.3).
  - `.go`, `.js`, `.ts`, `.rs`, `.py`, `.sh`, `.toml`, `.yaml`, `.yml`, `.json` → chroma with lexer matching the extension; dracula style; terminal256 formatter.
  - Anything else → plain text passthrough.
- Glamour reuse: accept `Model.threadMarkdown` pointer in constructor, don't instantiate a new one.
- Service interface still 44 methods.
- Top-level `Model` field additions: ≤ 2 (counting `fileViewerMode *fileViewerMode`).
- TOML config: `[tui.surfaces.file_viewer]` section — fields `max_bytes int` (default 1048576 = 1MiB), `dotfile_banner string` (default `"Dotfiles not supported in v1"`). No other knobs for MVP.
- File-size guard: if stat-ed file exceeds `max_bytes`, show "File too large ({size}) — limit {max}" banner.
- Errors wrapped with `%w`.
- Doc comments on every exported decl.

## TDD Test List (minimum)

- `TestFileViewer_Markdown_Golden` — a sample `.md` file renders via glamour; matches golden fixture.
- `TestFileViewer_GoCode_Golden` — a sample `.go` file renders via chroma; matches golden.
- `TestFileViewer_Dotfile_Refused` — `.gitignore` input → banner "Dotfiles not supported in v1", no content read.
- `TestFileViewer_LargeFile_Refused` — file above `max_bytes` → banner; file NOT fully read into memory.
- `TestFileViewer_PlainText_Passthrough` — `.txt` input → raw content, no rendering.
- `TestFileViewer_MissingFile_Error` — non-existent path → wrapped error, no panic.
- `TestKeymap_V_NoCollision` — enumerate every existing binding across every mode, assert `v` is free.
- `TestModel_V_EntersFileViewerMode` — on eligible surface, `v` → `model.mode == modeFileViewer`.
- `TestFileViewer_SharesThreadMarkdown` — `Model.threadMarkdown` instance is the same pointer used by fileViewerMode; assert by pointer-equality.
- `TestConfig_FileViewerDefaults` — default TOML parses; max_bytes == 1048576; dotfile_banner == spec string.

## Mage Targets

- `mage test-pkg internal/tui` (-race, -count=1).
- `mage test-pkg internal/config` (-race).
- `mage test-golden`.
- `mage ci`.

## Go Idioms (reinforce)

- `fileViewerMode` unexported; constructor injects `*glamour.TermRenderer` (shared from Model.threadMarkdown) + config struct.
- `chooseRenderer` is pure: takes filename + content bytes, returns rendered string + error. No side effects.
- Config struct uses `pelletier/go-toml/v2` tags matching existing config conventions.
- Lowercase ASCII extension match (lowercase the filename before switch); never use regex for extension check.
- `os.Stat` before `os.ReadFile` for the size guard; don't slurp then check.
- Errors wrapped with `%w`; use a sentinel `ErrFileTooLarge` if callers need `errors.Is`.

## Hylla Artifact Ref

`github.com/evanmschultz/tillsyn@main`
