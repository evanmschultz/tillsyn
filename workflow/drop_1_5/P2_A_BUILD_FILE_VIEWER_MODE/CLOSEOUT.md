# P2-A Closeout

## Verdict

**DONE** — P2-A BUILD FILE VIEWER MODE (v keybinding) closed 2026-04-18.

## Rounds

- **Round 1** — build PASS (`mage ci` 1334/1334 green, `internal/tui` 70.4%, `internal/config` 76.8%). QA Proof PASS (all 12 criteria cited). QA Falsification PASS (13 attack vectors refuted).

Single round — no fixup cycle needed.

## What shipped

- New `internal/tui/file_viewer_mode.go` — `fileViewerMode` struct (viewport, filePath, content, err + impl fields md/cfg/width/height), `openFile` with `os.Stat`-before-`os.ReadFile` size guard, sentinel `ErrFileTooLarge`, dotfile filter via `strings.HasPrefix(filepath.Base(path), ".")`.
- New `internal/tui/file_viewer_renderer.go` — pure `classifyExtension` + `chooseRenderer`: markdown via shared `*markdownRenderer` (glamour), code via chroma dracula terminal256, fallback plain text.
- New `internal/tui/file_viewer_mode_test.go` — all 10 PLAN.md TDD tests (markdown golden, go-code golden, dotfile refused, large-file refused, plaintext passthrough, missing-file error, keymap `v` non-collision across 31 other bindings, model `v` enters mode, `SharesThreadMarkdown` pointer equality, config defaults).
- New `internal/tui/testdata/file_viewer/` — sample.md, sample.go, .gitignore_fixture, sample_md.golden, sample_go.golden (ANSI-free stub renders for deterministic CI).
- `internal/tui/model.go` — `threadMarkdown markdownRenderer` → `*markdownRenderer` (heap-stable pointer for sharing); new `fileViewer *fileViewerMode` + `fileViewerBackMode inputMode` fields; `modeFileViewer` enum entry; `enterFileViewerMode` dispatch in `handleBoardPanelNormalKey` (after `activityLog` to respect `KeyConfig.ActivityLog` override).
- `internal/tui/keymap.go` — `fileViewerToggle` key.Binding on `"v"`.
- `internal/config/config.go` — `TUIConfig` / `TUISurfacesConfig` / `FileViewerConfig` structs with `max_bytes` (default 1048576) and `dotfile_banner` (default "Dotfiles not supported in v1") + `Default()` and `normalize()` wiring.

## Commit

- SHA: `af2a69c`
- Branch: `drop/1.5`
- Message: `feat(tui): add file viewer mode on v keybinding`
- Stats: 11 files changed, 869 insertions(+), 1 deletion(-).

## CI

- Run ID: `24616918555`
- `ci (macos-latest)` green in 52s; `release snapshot check` green in 1m3s.

## Gates

- `mage test-pkg ./internal/tui/...` — 390 tests green (was 380 post-P4-T4; +10 new).
- `mage test-pkg ./internal/config/...` — 32 tests green, 76.8% coverage.
- `mage test-golden` — 7/7 green.
- `mage ci` — 1334 tests green, all 20 packages ≥ 70% floor, build OK, formatting OK.

## Falsification highlights

- `threadMarkdown` value→pointer conversion verified safe: zero-value `Model{}` tests (model_test.go:5647, 11662, 11757) never reach `render`; `NewModel` initializes `m.threadMarkdown = &markdownRenderer{}` before any code path dereferences it.
- `enterFileViewerMode` LSP `unusedfunc` diagnostic was a false positive — method called at `model.go:10103` inside `handleBoardPanelNormalKey`. Confirmed via LSP `findReferences`.
- `v` dispatch correctly moved from top-level switch to `handleBoardPanelNormalKey` post-`activityLog` so `KeyConfig.ActivityLog: "v"` override wins per `TestWithKeyConfigOverrides`.

## Hylla Feedback (rollup to drop-end findings drop)

- `hylla_search` did not return the `markdownRenderer` struct declaration; only call sites surfaced. Fallback: `Grep`. Suggestion: index struct-level type declarations as top-ranked hits for symbol queries.
- `hylla_refs_find` did not surface `m.threadMarkdown` field-selector read sites. Fallback: `Grep`. Suggestion: index selector expressions (`x.Field`) as ref nodes so field-rename impact analysis works via Hylla.
- QA agents hit the expected "uncommitted files invisible to Hylla" pattern — working via `Read`/`LSP`/`Grep` + `git diff`, per project rule.

## Notes for drop-end ledger

- Golden fixtures use stub render funcs (`stubMarkdownRenderFunc`, `stubCodeRenderFunc`) producing deterministic ANSI-free tags. Tests prove renderer SELECTION is correct; they do NOT prove glamour/chroma output quality. Real-render coverage is future work (not in Drop 1.5 scope).
- `v` is not user-configurable via `KeyConfig` (no `FileViewer` field). Intentional per PLAN.md acceptance — ActivityLog precedence is the only configurable variant today.
- PLAN.md listed `keymap_test.go` and `config_test.go` as separate test files; builder placed all tests in `file_viewer_mode_test.go` — organizational deviation only, full coverage present.
