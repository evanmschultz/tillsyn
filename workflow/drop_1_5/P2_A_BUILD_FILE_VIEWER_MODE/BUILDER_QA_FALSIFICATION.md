# P2-A Builder QA — Falsification

Append `## Round K` per falsification-QA pass.

## Round 1

**Verdict: PASS — no counterexample constructed.**

### Falsification Certificate

**Premises (what the builder's claim requires):**
1. `v` keybinding registered under `fileViewerToggle`, no collision with any existing binding.
2. `fileViewerMode` struct has viewport, filePath, content, err fields.
3. `threadMarkdown` converted to `*markdownRenderer`; stable pointer passed to `newFileViewerMode`; pointer-equality invariant holds.
4. `openFile` guards: dotfile → banner (no file read); oversize → ErrFileTooLarge banner (stat before ReadFile); missing → wrapped os.ErrNotExist.
5. `chooseRenderer` routes by lowercased extension; nil-md guard falls back to plain text.
6. All errors wrapped with `%w`.
7. Service interface still 44 methods. Model gains ≤ 2 net new fields.
8. `mage ci` green (1334 tests, 0 failures).

**Evidence (observed):**

- `file_viewer_mode.go`: struct has `md`, `cfg`, `viewport`, `filePath`, `content`, `err`, `width`, `height`. All four PLAN-required fields present.
- `openFile`: dotfile guard via `filepath.Base` + `HasPrefix(".", ...)` before any I/O. `os.Stat` called before `os.ReadFile`. All error returns use `fmt.Errorf("...: %w", ...)`.
- `file_viewer_renderer.go`: `classifyExtension` uses `strings.ToLower(filepath.Ext(base))` — correct case-insensitive routing. `chooseRenderer` guards `md == nil`.
- `model.go` diff: 2 net new fields (`fileViewer *fileViewerMode`, `fileViewerBackMode inputMode`). `threadMarkdown` type-changed from value to pointer (not a new field). `m.threadMarkdown = &markdownRenderer{}` assigned before options loop in `NewModel`.
- `keymap.go`: `fileViewerToggle key.Binding` on `"v"`. `applyConfig` does NOT configure `fileViewerToggle` — non-configurable by design.
- Service interface lines 35–78: 44 methods. Satisfies PLAN.
- LSP `findReferences` on `enterFileViewerMode` (line 190): 2 references — definition + `model.go:10103`. NOT dead code.
- `TestWithKeyConfigOverrides`: `ActivityLog: "v"` → `modeActivityLog`. `fileViewerToggle` still has `"v"` but `activityLog` case appears first in switch; Go switch exits on first match. Correct precedence.
- Zero-value `Model{}` tests (lines 5647, 11662, 11757 of model_test.go): none call `threadMarkdown.render(...)`; no nil-deref risk.
- Golden fixtures: `sample.md` 8 lines, `sample.go` 9 lines. Both routed through `chooseRendererWithFuncs` with deterministic stubs — proves routing, not live render output.

**Trace or cases (attempted counterexamples):**

- **CE-1: nil deref on `threadMarkdown` for zero-value `Model{}`** — All three `Model{}` constructions in tests only call methods that don't touch `threadMarkdown.render(...)`. `NewModel` sets `threadMarkdown` before options loop. REFUTED.
- **CE-2: `enterFileViewerMode` dead code** — LSP confirms 2 call sites (definition + model.go:10103). REFUTED.
- **CE-3: `v` fires from non-board surfaces** — Dispatch only in `handleBoardPanelNormalKey`, reached only when `m.mode == modeNone` (board). Non-board modes route through `handleInputModeKey` with mode-specific handlers. `v` silently ignored from non-board modes — intended design. REFUTED (by design).
- **CE-4: `v` collision after `ActivityLog: "v"` config** — `activityLog` case precedes `fileViewerToggle` in switch. Go switch takes first match. `TestWithKeyConfigOverrides` confirms correct behavior. `fileViewerToggle` non-configurable is intentional (no `KeyConfig.FileViewer` field in spec). REFUTED.
- **CE-5: golden fixtures too trivial to catch render bugs** — Stubs prove routing (md → glamour path, .go → chroma path) but not render quality. Accepted MVP limitation per builder's design note. NOT a counterexample to the acceptance criteria, which require the tests exist and pass. REFUTED as a blocking issue.
- **CE-6: ReadFile called before Stat** — `openFile` calls `os.Stat` first; returns `ErrFileTooLarge` before `os.ReadFile` if size exceeded. REFUTED.
- **CE-7: dotfile filter incorrect for `"foo/.bar"` paths** — `filepath.Base("foo/.bar")` = `".bar"`, `HasPrefix(".bar", ".")` = true. All common path shapes correctly handled. Test only covers one shape (coverage gap), but implementation correct. REFUTED.
- **CE-8: errors not wrapped with `%w`** — All three `fmt.Errorf` in `file_viewer_mode.go` use `%w`. All two in `file_viewer_renderer.go` use `%w`. REFUTED.
- **CE-9: uppercase extension routing broken** — `strings.ToLower(filepath.Ext(base))` normalizes before switch. REFUTED.
- **CE-10: service interface method count not 44** — Counted 44 methods (lines 35–78 inclusive). REFUTED.
- **CE-11: more than 2 new Model fields** — 2 net new fields only. REFUTED.
- **CE-12: fileViewerMode missing required struct fields** — All four PLAN-required fields present. REFUTED.
- **CE-13: nil markdownRenderer panic in chooseRenderer** — Guarded with `if md == nil { return string(content), nil }`. REFUTED.

**Conclusion: PASS** — No unmitigated counterexample found across all 13 attack vectors.

**Unknowns (routed, non-blocking):**

- Real glamour/chroma render quality is not covered by stubs — accepted MVP limitation.
- `TestModel_V_EntersFileViewerMode` exercises mode transition only; `openFile` with a real path is covered by separate unit tests. Not a gap in acceptance-criteria coverage.
- `v` is non-configurable via `KeyConfig.FileViewer` (field does not exist). Intentional omission per spec — no acceptance criterion requires configurability.
- `enterFileViewerMode` accesses only `ref.Tags[0]` (first tag) per ResourceRef. Multi-tag refs with path at index > 0 are missed — MVP scope limitation, not in acceptance criteria.

---

## Hylla Feedback

- **Query**: `hylla_search` for `markdownRenderer`, `threadMarkdown`, `fileViewerMode` — needed to understand struct definitions and call sites.
- **Missed because**: P2-A files are uncommitted; Hylla indexes committed code only. Also, the builder's worklog confirms Hylla couldn't find `markdownRenderer` struct definition (found call sites but not the definition).
- **Worked via**: `Read` on source files directly + `Grep` for symbol patterns across `internal/tui/`. LSP `findReferences` for call-site verification.
- **Suggestion**: Hylla should index struct-level definitions separately from call sites so symbol search returns the type declaration as the top hit. Also, a "live uncommitted files" mode would help QA agents working on unmerged drops.

- **Note on scope**: All P2-A files are uncommitted working-tree changes (`git status` shows HEAD at `e8914fc`, which is P4-T4). Hylla is Go-only and indexes committed code; all analysis was done via `Read`/`Grep`/`LSP`/`git diff`. This is expected and correct per project rules.
