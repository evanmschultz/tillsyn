# P2-A Builder QA — Proof

Append `## Round K` per proof-QA pass.

## Round 1

**Verdict: PASS**

**QA Agent:** go-qa-proof-agent (Sonnet 4.6)
**Date:** 2026-04-18

### Mage Gate Outputs (re-run by QA)

```
mage test-pkg ./internal/tui/...
  390 tests / 0 failures / 2 packages / internal/tui: 70.4%

mage test-pkg ./internal/config/...
  32 tests / 0 failures / 1 package / internal/config: 76.8%

mage test-golden
  7 tests / 0 failures

mage ci
  1334 tests / 0 failures / 20 packages / all ≥ 70% coverage / build green
```

### Criterion-by-Criterion Evidence

**1. `v` keybinding under `FileViewerToggle`, no collision with ctrl+d or existing bindings**

- `internal/tui/keymap.go:44` — field `fileViewerToggle key.Binding` in `keyMap` struct.
- `internal/tui/keymap.go:81` — initialized `key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "file viewer"))`.
- `diffModeToggle` bound to `ctrl+d` at `keymap.go:80` — no collision.
- `TestKeymap_V_NoCollision` at `file_viewer_mode_test.go:250-296` exhaustively checks all 31 other bindings confirm none claim `"v"`. Passes.

**2. `fileViewerMode` struct contains: viewport, active file path, rendered content, error**

- `internal/tui/file_viewer_mode.go:42-54` — struct declaration:
  - `viewport viewport.Model` (line 46)
  - `filePath string` (line 48)
  - `content string` (line 49)
  - `err error` (line 50)
- All four required fields present.

**3. Dotfile filter: basename starting with `.` → banner, file NOT read**

- `internal/tui/file_viewer_mode.go:92-99`: `strings.HasPrefix(base, ".")` check; sets `fv.content` to banner from config; calls `fv.renderInto()` and returns `nil` without calling `os.Stat` or `os.ReadFile`.
- Banner literal at `file_viewer_mode.go:34`: `const fileViewerBannerDotfile = "Dotfiles not supported in v1"`.
- `TestFileViewer_Dotfile_Refused` at `file_viewer_mode_test.go:48-65`: opens `.gitignore_fixture`, asserts banner present, asserts `*.log` content absent. Passes.

**4. Renderer selection — extensions to kind mapping**

- `internal/tui/file_viewer_renderer.go:25-50` — `classifyExtension` pure switch:
  - `.md`, `.markdown` → `fileRendererMarkdown`
  - `.go` → `fileRendererCode, "go"` / `.js` → `"javascript"` / `.ts` → `"typescript"` / `.rs` → `"rust"` / `.py` → `"python"` / `.sh` → `"bash"` / `.toml` → `"toml"` / `.yaml`, `.yml` → `"yaml"` / `.json` → `"json"`
  - default → `fileRendererPlain`
- Chroma dracula terminal256 at `file_viewer_renderer.go:117-138` (`renderCodeContent`).
- Extension lowercased before switch at `chooseRenderer:62` (`strings.ToLower(filepath.Ext(base))`).

**5. Glamour reuse — constructor accepts shared `*markdownRenderer` pointer; no new instantiation**

- `internal/tui/file_viewer_mode.go:60` — `func newFileViewerMode(md *markdownRenderer, cfg config.FileViewerConfig) *fileViewerMode`.
- `internal/tui/model.go:1354-1362` — `NewModel` sets `m.threadMarkdown = &markdownRenderer{}` then `m.fileViewer = newFileViewerMode(m.threadMarkdown, ...)`. Same pointer passed.
- `TestFileViewer_SharesThreadMarkdown` at `file_viewer_mode_test.go:200-215` — pointer-equality assertion `m.fileViewer.md != m.threadMarkdown`. Passes.

**6. Service interface: 44 methods**

- `internal/tui/model.go:34-79` — interface declaration, lines 35-78 = 44 methods (including `EmbeddingsOperational() bool` at line 63 which takes no context). Manual count: 44. Criterion met.

**7. Top-level Model field additions ≤ 2**

- `git diff HEAD -- internal/tui/model.go` shows exactly two new fields added to `Model`:
  - `fileViewer *fileViewerMode` (model.go:975)
  - `fileViewerBackMode inputMode` (model.go:979)
- `threadMarkdown` changed from value (`markdownRenderer`) to pointer (`*markdownRenderer`) — same field, type change only. Net new field count: 2. Criterion met.

**8. TOML config `[tui.surfaces.file_viewer]`: max_bytes default 1048576, dotfile_banner exact string**

- `internal/config/config.go:151-174` — `TUIConfig`, `TUISurfacesConfig`, `FileViewerConfig` structs with TOML tags `[tui]`, `[surfaces]`, `[file_viewer]`.
- `config.go:171` — `const DefaultFileViewerMaxBytes = 1048576`
- `config.go:174` — `const DefaultFileViewerDotfileBanner = "Dotfiles not supported in v1"`
- `Default()` at `config.go:269-275` wires defaults into the struct.
- `normalize()` at `config.go:667-672` re-applies defaults on zero values.
- `TestConfig_FileViewerDefaults` at `file_viewer_mode_test.go:300-310` — asserts both values. Passes.

**9. File-size guard: `os.Stat` before `os.ReadFile`; banner on large file; large file NOT fully read**

- `internal/tui/file_viewer_mode.go:102` — `os.Stat(path)`.
- `file_viewer_mode.go:113-118` — size check `info.Size() > maxBytes`; sets error and banner; returns without calling `os.ReadFile`.
- `os.ReadFile` at line 121 — only reached if stat succeeds AND size within limit.
- `TestFileViewer_LargeFile_Refused` at `file_viewer_mode_test.go:69-98` — creates file 1 byte over limit; asserts `ErrFileTooLarge` in error chain and banner in content. Passes.

**10. Errors wrapped with `%w`; doc comments on every exported decl**

- `file_viewer_mode.go:104` — `fmt.Errorf("file viewer: stat %s: %w", path, err)` — `%w`.
- `file_viewer_mode.go:114` — `fmt.Errorf("file viewer: %w: ...", ErrFileTooLarge, ...)` — `%w`.
- `file_viewer_mode.go:123` — `fmt.Errorf("file viewer: read %s: %w", path, err)` — `%w`.
- `file_viewer_renderer.go:131` — `fmt.Errorf("file viewer: tokenise %s: %w", lexerName, err)` — `%w`.
- `file_viewer_renderer.go:135` — `fmt.Errorf("file viewer: format %s: %w", lexerName, err)` — `%w`.
- Exported decls with doc comments: `ErrFileTooLarge` (file_viewer_mode.go:26-29), `TUIConfig` (config.go:151), `TUISurfacesConfig` (config.go:156), `FileViewerConfig` (config.go:161-165), `DefaultFileViewerMaxBytes` (config.go:170), `DefaultFileViewerDotfileBanner` (config.go:173).

**11. All 10 TDD tests present and passing**

All 10 tests present in `internal/tui/file_viewer_mode_test.go`:
1. `TestFileViewer_Markdown_Golden` (line 134)
2. `TestFileViewer_GoCode_Golden` (line 167)
3. `TestFileViewer_Dotfile_Refused` (line 48)
4. `TestFileViewer_LargeFile_Refused` (line 69)
5. `TestFileViewer_PlainText_Passthrough` (line 116)
6. `TestFileViewer_MissingFile_Error` (line 102)
7. `TestKeymap_V_NoCollision` (line 250)
8. `TestModel_V_EntersFileViewerMode` (line 219)
9. `TestFileViewer_SharesThreadMarkdown` (line 200)
10. `TestConfig_FileViewerDefaults` (line 300)

`mage test-pkg ./internal/tui/...` → 390 passed, 0 failed. All 10 present and passing.

**12. Mage gates all green**

All four targets confirmed green above. See "Mage Gate Outputs."

### Cross-Cutting Verification

**`threadMarkdown` pointer change safety:**

- All 9 call sites `m.threadMarkdown.render(...)` in model.go work unchanged — Go auto-derefs pointer method calls (`(*T).method` via `x.method()`).
- `NewModel` sets `m.threadMarkdown = &markdownRenderer{}` before any code path that might dereference it (before `m.diff = newDiffMode(...)` and `m.fileViewer = newFileViewerMode(m.threadMarkdown, ...)`).
- No test reads `threadMarkdown` by value via `reflect.DeepEqual` (checked via grep — no such pattern).

**`enterFileViewerMode` usage:**

- LSP diagnostic flagged it "unused" — confirmed false positive. Method is called at `model.go:10103` inside `handleBoardPanelNormalKey` via `return m.enterFileViewerMode()`. The diagnostic applies to unexported standalone functions, not methods; methods on exported types are always reachable.

**Pre-existing LSP hints (not P2-A regressions):**

- `config.go:502,770,834,980` — `git diff HEAD -- internal/config/config.go` shows only P2-A additions (TUIConfig, FileViewerConfig, Default/normalize wiring). Lines 502/770/834/980 are pre-existing code not touched by P2-A.
- `model_test.go` hints — `git diff HEAD -- internal/tui/model_test.go` shows no P2-A changes. Pre-existing.

### Minor Organizational Note

PLAN.md listed `keymap_test.go (modified)` and `internal/config/config_test.go (modified)` as touched paths. The builder placed the keymap collision test (`TestKeymap_V_NoCollision`) and config defaults test (`TestConfig_FileViewerDefaults`) in `file_viewer_mode_test.go` instead. All test coverage is present and passing — this is a non-issue for correctness.

### Verdict

**PASS.** All 12 acceptance criteria verified by file:line citations and live mage gate execution. No gaps found.

## Hylla Feedback

P2-A's files (`file_viewer_mode.go`, `file_viewer_renderer.go`, `file_viewer_mode_test.go`, keymap.go changes, model.go changes, config.go changes) are uncommitted — Hylla has no ingest of these files. Per task instructions, Read/LSP/Grep were used for all P2-A file inspection (correct per the task brief). No Hylla queries were attempted for P2-A files.

For pre-P4-T3 baseline: no Hylla queries were needed; all evidence was in the P2-A files themselves, readable directly. N/A for this round — task touched only uncommitted files, Hylla not applicable.
